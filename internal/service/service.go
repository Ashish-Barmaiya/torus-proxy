package service

import (
	"torus-proxy/internal/loadbalancer"
	"torus-proxy/internal/upstream"
)

type Service struct {
	lb loadbalancer.LoadBalancer
}

func NewService(backends []*upstream.Backend) *Service {
	return &Service{
		lb : loadbalancer.NewRoundRobin(backends),
	}
}

func (s *Service) NextBackend() *upstream.Backend {
	return s.lb.Next()
}
