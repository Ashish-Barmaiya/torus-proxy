package runtime

import (
	"context"
	"crypto/tls"
	"torus-proxy/internal/routing"
)

type Runtime struct {
	Router    *routing.Router
	TLSConfig *tls.Config

	cancel context.CancelFunc
}

func NewRuntime(router *routing.Router, tlsConfig *tls.Config, cancel context.CancelFunc) *Runtime {
	return &Runtime{
		Router:    router,
		TLSConfig: tlsConfig,
		cancel:    cancel,
	}
}

func (r *Runtime) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
}
