package proxy

import (
	"context"
	"crypto/tls"
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

type Server struct {
	runtime     atomic.Pointer[runtime.Runtime]
	runtimeMu   sync.RWMutex
	logger      *slog.Logger
	srv         *http.Server
	listener    net.Listener
	started     chan string
	ready       atomic.Bool
	baseCtx     context.Context
	forceCancel context.CancelFunc
}

func NewServer(rt *runtime.Runtime, logger *slog.Logger) *Server {
	s := &Server{
		logger:  logger,
		started: make(chan string, 1),
	}

	s.runtime.Store(rt)

	return s
}

// The HTTP Handler function
func (s *Server) httpHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	done := observability.TrackInflight()
	defer done()

	rec := middleware.NewStatusRecorder(w)
	w = rec

	// Acquire a runtime context for this request
	// RWMutex prevents race window between load() and acquire() operation
	s.runtimeMu.RLock()
	rt := s.runtime.Load()

	rt.AcquireRequest()
	s.runtimeMu.RUnlock()
	defer rt.ReleaseRequest() // Release the runtime context when the request is done

	// find the correct service using routing logic
	route, svc := rt.Router.Route(r.URL.Path)

	defer func() {
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

	// find the next available backend
	backend := svc.NextBackend()

	if backend == nil {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable) // returns 503
		return
	}

	// forward the request to the upstream connection pipeline
	transport.Forward(w, r, backend.Proxy)
}

func (s *Server) Handler() http.Handler {
	var h http.Handler = http.HandlerFunc(s.httpHandler)
	if s.logger != nil {
		h = middleware.LoggingMiddleware(s.logger)(h)
	}
	return h
}

func (s *Server) WaitStarted() string {
	return <-s.started
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/", s.Handler())
	if observability.Enabled() {
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
		return err
	}

	s.listener = ln

	s.started <- ln.Addr().String()

	rt := s.runtime.Load()

	if rt.TLSConfig != nil {
		ln = tls.NewListener(ln, rt.TLSConfig)
	}

	s.ready.Store(true)

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
		return err
	}

	return nil
}

// Shutdown gracefully with a timeout context
func (s *Server) Shutdown(timeout time.Duration) error {
	s.ready.Store(false) // mark server as not ready to receive traffic and prevents k8s from sending new traffic

	s.logger.Info("Shutting down server", "timeout", timeout)
	if s.srv == nil {
		return nil
	}

	defer func() {
		if rt := s.runtime.Load(); rt != nil {
			rt.Stop()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := s.srv.Shutdown(ctx)
	if err == nil {
		s.logger.Info("graceful shutdown complete")
		return nil
	}

	s.logger.Warn("graceful shutdown deadline exceeded, forcing cancellation of pending requests")
	s.forceCancel() // force cancel all pending requests

	forcedCtx, forcedCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer forcedCancel()

	if err2 := s.srv.Shutdown(forcedCtx); err2 != nil {
		s.logger.Error("forced shutdown failed", "error", err2)
		return err2
	}

	s.logger.Info("forced shutdown complete")
	return nil
}

// Reload replaces the current runtime with a new one and stops the old runtime.
func (s *Server) Reload(newRuntime *runtime.Runtime) {
	// Swap the runtime pointer
	// Lock prevents any new request to load and acquire old runtime while runtimes are being swapped
	s.runtimeMu.Lock()
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
}
