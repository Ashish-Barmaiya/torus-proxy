package runtime

import (
	"context"
	"crypto/tls"
	"sync"
	"sync/atomic"
	"torus-proxy/internal/routing"
)

var nextGeneration atomic.Uint64

func ResetGenerationForTesting() {
	nextGeneration.Store(0)
}

type Runtime struct {
	Generation           uint64
	Addr                 string
	Router               *routing.Router
	TLSConfig            *tls.Config
	ObservabilityEnabled bool

	cancel    context.CancelFunc
	requestWG sync.WaitGroup
	workerWG  sync.WaitGroup
}

// Runtime owns:
//   - background workers
//   - health probe lifecycle
//   - in-flight request accounting
//
// Stop() cancels workers and waits until all workers and requests
// associated with this runtime have completed before returning.
func NewRuntime(generation uint64, addr string, router *routing.Router, tlsConfig *tls.Config, observabilityEnabled bool, cancel context.CancelFunc) *Runtime {
	return &Runtime{
		Generation:           generation,
		Addr:                 addr,
		Router:               router,
		TLSConfig:            tlsConfig,
		ObservabilityEnabled: observabilityEnabled,
		cancel:               cancel,
	}
}

func (r *Runtime) AcquireRequest() {
	r.requestWG.Add(1)
}

func (r *Runtime) ReleaseRequest() {
	r.requestWG.Done()
}

func (r *Runtime) AddWorker() {
	r.workerWG.Add(1)
}

func (r *Runtime) DoneWorker() {
	r.workerWG.Done()
}

func (r *Runtime) Stop() {
	if r.cancel != nil {
		r.cancel()
	}

	r.requestWG.Wait()
	r.workerWG.Wait()
}
