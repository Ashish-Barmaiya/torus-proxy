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

func (s *Server) httpHandler(w http.ResponseWriter, r *http.Request) {
	svc := s.router.Route(r.URL.Path)
	if svc == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	backend := svc.NextBackend()
	transport.Forward(w, r, backend.URL)
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.httpHandler)
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.httpHandler)

	srv := &http.Server{
		Addr: addr,
		Handler: mux,

		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout: 120 * time.Second,
	}

	return srv.ListenAndServe()
}