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
	i := r.index.Add(1)
	math := int(i) % len(r.backends)
	return r.backends[math]
}
