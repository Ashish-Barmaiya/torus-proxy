package runtime

import (
	"context"
	"crypto/tls"
	"sync"
	"torus-proxy/internal/routing"
)

type Runtime struct {
	Router    *routing.Router
	TLSConfig *tls.Config

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewRuntime(router *routing.Router, tlsConfig *tls.Config, cancel context.CancelFunc) *Runtime {
	return &Runtime{
		Router:    router,
		TLSConfig: tlsConfig,
		cancel:    cancel,
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
