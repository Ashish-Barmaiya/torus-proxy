package proxy

import (
	"net/http"
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

func (s *Server) Start(addr string) error {
	http.HandleFunc("/", s.httpHandler)
	return http.ListenAndServe(addr, nil)
}