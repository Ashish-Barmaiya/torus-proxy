package proxy

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"
	"torus-proxy/internal/middleware"
	"torus-proxy/internal/routing"
	"torus-proxy/internal/transport"
)

type Server struct {
	router      *routing.Router
	logger      *slog.Logger
	srv         *http.Server
	ready       atomic.Bool
	baseCtx     context.Context
	forceCancel context.CancelFunc
}

func NewServer(router *routing.Router, logger *slog.Logger) *Server {
	return &Server{router: router, logger: logger}
}

// The HTTP Handler function
func (s *Server) httpHandler(w http.ResponseWriter, r *http.Request) {
	// find the correct service using routing logic
	svc := s.router.Route(r.URL.Path)
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

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/", s.Handler())

	// Readiness endpoint - used by Kubernetes to check if the server is ready to receive traffic
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if s.ready.Load() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("not ready"))
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

	s.ready.Store(true)

	s.logger.Info("Torus listening", "addr", addr)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
