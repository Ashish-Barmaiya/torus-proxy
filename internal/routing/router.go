package routing

import (
	"strings"
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

// Longest Prefix Match
func (r *Router) Route(path string) *service.Service {
	var bestmatch string
	var bestSvc *service.Service

	for routeKey, svc := range r.routes {
		if !strings.HasPrefix(path, routeKey) {
			continue
		}
		if strings.HasPrefix(path, routeKey) {
			if len(path) == len(routeKey) || path[len(routeKey)] == '/' {
				if len(routeKey) > len(bestmatch) {
					bestmatch = routeKey
					bestSvc = svc
				}
			}
		}
	}

	return bestSvc
}
