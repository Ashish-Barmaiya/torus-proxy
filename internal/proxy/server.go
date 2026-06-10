package proxy

import (
	"log/slog"
	"net/http"
	"time"
	"torus-proxy/internal/middleware"
	"torus-proxy/internal/routing"
	"torus-proxy/internal/transport"
)

type Server struct {
	router *routing.Router
	logger *slog.Logger
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

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return srv.ListenAndServe()
}
