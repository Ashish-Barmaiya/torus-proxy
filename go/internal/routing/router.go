package routing

import (
	"torus-proxy/internal/service"
)

type Router struct {
	routes map[string]*service.Service
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[string]*service.Service),
	}
}

func (r *Router) AddRoute(path string, svc *service.Service) {
	r.routes[path] = svc
}

func (r * Router) Route(path string) * service.Service {
	return r.routes[path]
}