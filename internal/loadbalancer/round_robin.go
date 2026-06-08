package loadbalancer

import (
	"sync/atomic"
	"torus-proxy/internal/upstream"
)

type RoundRobin struct {
	backends []*upstream.Backend
	index    atomic.Uint64
}

func NewRoundRobin(backends []*upstream.Backend) *RoundRobin {
	return &RoundRobin{backends: backends}
}

func (r *RoundRobin) Next() *upstream.Backend {
	if len(r.backends) == 0 {
		return nil
	}

	for i := 0; i < len(r.backends); i++ {
		idx := r.index.Add(1) % uint64(len(r.backends))
		backend := r.backends[idx]
		if backend.IsHealthy() {
			return backend
		}
	}
	return nil // if all unhealthy
}
