package proxy

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
	"torus-proxy/internal/middleware"
	"torus-proxy/internal/observability"
	"torus-proxy/internal/runtime"
	"torus-proxy/internal/transport"
)

type serverState uint32

var ErrServerNotRunning = errors.New("server is not running")

const (
	serverStarting serverState = iota
	serverRunning
	serverShuttingDown
	serverStopped
)

type Server struct {
	runtime      atomic.Pointer[runtime.Runtime]
	runtimeMu    sync.RWMutex
	logger       *slog.Logger
	srv          *http.Server
	listener     net.Listener
	started      chan string
	shutdownDone chan struct{}
	shutdownErr  error
	ready        atomic.Bool
	state        atomic.Uint32
	baseCtx      context.Context
	forceCancel  context.CancelFunc
}

// NewServer creates a proxy server for the given runtime.
func NewServer(rt *runtime.Runtime, logger *slog.Logger) *Server {
	s := &Server{
		logger:       logger,
		started:      make(chan string, 1),
		shutdownDone: make(chan struct{}),
	}

	s.runtime.Store(rt)
	s.state.Store(uint32(serverStarting))

	return s
}

// httpHandler routes a request through the current runtime and forwards it to
// the selected upstream backend.
func (s *Server) httpHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	rec := middleware.NewStatusRecorder(w)
	w = rec

	// Acquire the runtime and register the request before releasing runtimeMu
	// This prevents a concurrent reload from retiring the runtime between load
	// and acquire
	s.runtimeMu.RLock()
	rt := s.runtime.Load()

	rt.AcquireRequest()
	s.runtimeMu.RUnlock()
	defer rt.ReleaseRequest() // Release the runtime request reference when the request completes

	var done func()
	if rt.ObservabilityEnabled {
		done = observability.TrackInflight()
	} else {
		done = func() {}
	}
	defer done()

	route, svc := rt.Router.Route(r.URL.Path)

	defer func() {
		if !rt.ObservabilityEnabled {
			return
		}

		observability.RecordHTTPRequest(
			r.Method,
			route,
			rec.Status(),
		)

		observability.ObserveHTTPRequestDuration(
			r.Method,
			route,
			time.Since(start),
		)
	}()

	if svc == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	backend := svc.NextBackend()

	if backend == nil {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable) // returns 503
		return
	}

	transport.Forward(w, r, backend.Proxy)
}

// Handler returns the proxy's HTTP handler with request logging applied
func (s *Server) Handler() http.Handler {
	var h http.Handler = http.HandlerFunc(s.httpHandler)
	if s.logger != nil {
		h = middleware.LoggingMiddleware(s.logger)(h)
	}
	return h
}

// WaitStarted blocks until the server has successfully bound its listener
func (s *Server) WaitStarted() string {
	return <-s.started
}

// Start initializes and serves the HTTP server for the current runtime.
//
// A successful start transitions the server to serverRunning. Listener or
// unexpected Serve failures transition it to serverStopped and clean up the
// associated runtime.
func (s *Server) Start(addr string) error {
	rt := s.runtime.Load()

	mux := http.NewServeMux()
	mux.Handle("/", s.Handler())

	if rt.ObservabilityEnabled {
		mux.Handle("/metrics", observability.Handler())
	}

	// Readiness endpoint - used by Kubernetes to check if the server is ready to receive traffic
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if s.ready.Load() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ready"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))
		}
	})

	// Base ctx for every request
	baseCtx, forceCancel := context.WithCancel(context.Background())
	s.baseCtx = baseCtx
	s.forceCancel = forceCancel

	s.srv = &http.Server{
		Addr:    addr,
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			return baseCtx
		},
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Create listener
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		s.ready.Store(false)
		s.state.Store(uint32(serverStopped))

		if s.forceCancel != nil {
			s.forceCancel()
		}

		if rt != nil {
			rt.Stop()
		}
		return err
	}

	s.listener = ln

	s.started <- ln.Addr().String()

	if rt.TLSConfig != nil {
		ln = tls.NewListener(ln, rt.TLSConfig)
	}

	s.ready.Store(true)
	s.state.Store(uint32(serverRunning))

	s.logger.Info(
		"runtime started",
		"generation", rt.Generation,
	)
	s.logger.Info(
		"Torus listening",
		"addr", addr,
		"tls", rt.TLSConfig != nil,
	)

	// Start serving
	if err := s.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		s.logger.Error("server stopped", "error", err)

		s.runtimeMu.Lock()

		// If shutdown is already in progress, Shutdown() owns the
		// lifecycle transition and runtime cleanup.
		currentState := serverState(s.state.Load())
		if currentState == serverRunning {
			s.ready.Store(false)
			s.state.Store(uint32(serverStopped))

			rt := s.runtime.Load()

			s.runtimeMu.Unlock()

			if s.forceCancel != nil {
				s.forceCancel()
			}

			if rt != nil {
				rt.Stop()
			}
		} else {
			s.runtimeMu.Unlock()
		}

		return err
	}

	return nil
}

// Shutdown drains the HTTP server and retires the active runtime.
//
// The first caller owns the shutdown; concurrent callers wait for the same
// shutdown result. runtimeMu protects the lifecycle transition and runtime
// snapshot, but is not held while blocking on request or worker completion.
func (s *Server) Shutdown(timeout time.Duration) error {
	s.runtimeMu.Lock()

	switch serverState(s.state.Load()) {
	case serverStopped:
		err := s.shutdownErr
		s.runtimeMu.Unlock()
		return err

	case serverShuttingDown:
		done := s.shutdownDone
		s.runtimeMu.Unlock()

		// Another goroutine owns the shutdown. Wait for it to finish
		// and return the same result
		<-done

		s.runtimeMu.Lock()
		err := s.shutdownErr
		s.runtimeMu.Unlock()

		return err

	case serverStarting, serverRunning:
		// This goroutine becomes the owner of the shutdown
		s.state.Store(uint32(serverShuttingDown))
		s.ready.Store(false)

		// Capture the runtime that was active when shutdown began
		rt := s.runtime.Load()

		s.runtimeMu.Unlock()

		s.logger.Info("Shutting down server", "timeout", timeout)

		complete := func(err error) error {
			s.runtimeMu.Lock()

			s.shutdownErr = err

			// If the HTTP server itself failed to shut down, keep the
			// lifecycle state as SHUTTING_DOWN rather than falsely
			// reporting STOPPED
			if err == nil {
				s.state.Store(uint32(serverStopped))
			}

			close(s.shutdownDone)

			s.runtimeMu.Unlock()

			return err
		}

		if s.srv == nil {
			if rt != nil {
				rt.Stop()
			}

			return complete(nil)
		}

		ctx, cancel := context.WithTimeout(
			context.Background(),
			timeout,
		)
		defer cancel()

		err := s.srv.Shutdown(ctx)
		if err == nil {
			if rt != nil {
				rt.Stop()
			}

			s.logger.Info("graceful shutdown complete")
			return complete(nil)
		}

		s.logger.Warn(
			"graceful shutdown deadline exceeded, forcing cancellation of pending requests",
		)

		if s.forceCancel != nil {
			s.forceCancel()
		}

		forcedCtx, forcedCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer forcedCancel()

		if err2 := s.srv.Shutdown(forcedCtx); err2 != nil {
			s.logger.Error("forced shutdown failed", "error", err2)

			// The shutdown operation has finished and all concurrent
			// callers must receive the same failure
			return complete(err2)
		}

		if rt != nil {
			rt.Stop()
		}

		s.logger.Info("forced shutdown complete")
		return complete(nil)

	default:
		s.runtimeMu.Unlock()
		return nil
	}
}

// Reload installs a new runtime and retires the previous runtime.
//
// Reload is only allowed while the server is running. The runtime swap is
// synchronized with request acquisition and shutdown through runtimeMu.
func (s *Server) Reload(newRuntime *runtime.Runtime) error {
	// Swap the runtime pointer
	// Lock prevents any new request to load and acquire old runtime while runtimes are being swapped
	s.runtimeMu.Lock()

	if serverState(s.state.Load()) != serverRunning {
		s.runtimeMu.Unlock()
		return ErrServerNotRunning
	}

	oldRuntime := s.runtime.Swap(newRuntime)

	s.runtimeMu.Unlock()

	observability.SetRuntimeGeneration(newRuntime.Generation)
	observability.RecordRuntimeReload(true)

	s.logger.Info(
		"runtime reloaded",
		"old_generation", oldRuntime.Generation,
		"new_generation", newRuntime.Generation,
	)

	oldRuntime.Stop()

	s.logger.Info(
		"runtime retired",
		"generation", oldRuntime.Generation,
	)

	return nil
}
