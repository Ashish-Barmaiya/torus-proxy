package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var registry = newRegistry()

func newRegistry() *prometheus.Registry {
	registry := prometheus.NewRegistry()

	// Go runtime and process collectors
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(
			collectors.ProcessCollectorOpts{},
		),
	)

	// HTTP metrics
	registry.MustRegister(
		httpMetrics.requestsTotal,
		httpMetrics.requestDuration,
		httpMetrics.inflight,
	)

	// Backend metrics
	registry.MustRegister(
		backendMetrics.requestsTotal,
		backendMetrics.requestDuration,
		backendMetrics.errorsTotal,
		backendMetrics.up,
	)

	// Runtime metrics
	registry.MustRegister(
		runtimeMetrics.reloadsTotal,
		runtimeMetrics.generation,
		runtimeMetrics.buildTime,
	)

	return registry
}
