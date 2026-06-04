package proxy

import (
	"net/http"
	"time"
	"torus-proxy/internal/routing"
	"torus-proxy/internal/transport"
)

type Server struct {
	router *routing.Router
}

func NewServer(router *routing.Router) *Server {
	return &Server{router: router}
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

	// forward the request to the upstream connection pipeline
	transport.Forward(w, r, backend.Proxy)
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.httpHandler)
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.httpHandler)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,

		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return srv.ListenAndServe()
}
