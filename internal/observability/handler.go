package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns an HTTP handler that exposes Torus metrics
// in the Prometheus exposition format
func Handler() http.Handler {
	return promhttp.HandlerFor(
		registry,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	)
}
