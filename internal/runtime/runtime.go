package runtime

import (
	"context"
	"crypto/tls"
	"sync"
	"sync/atomic"
	"torus-proxy/internal/routing"
)

var nextGeneration atomic.Uint64

type Runtime struct {
	Generation uint64
	Router     *routing.Router
	TLSConfig  *tls.Config

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewRuntime(generation uint64, router *routing.Router, tlsConfig *tls.Config, cancel context.CancelFunc) *Runtime {
	return &Runtime{
		Generation: generation,
		Router:     router,
		TLSConfig:  tlsConfig,
		cancel:     cancel,
	}
}

func (r *Runtime) Acquire() {
	r.wg.Add(1)
}

func (r *Runtime) Release() {
	r.wg.Done()
}

func (r *Runtime) Stop() {
	if r.cancel != nil {
		r.cancel()
	}

	r.wg.Wait()
}
