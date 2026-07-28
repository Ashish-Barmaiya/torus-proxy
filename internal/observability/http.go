package observability

import "github.com/prometheus/client_golang/prometheus"

var httpMetrics = struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	inflight        prometheus.Gauge
}{
	requestsTotal: prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "torus",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests proxied.",
		},
		[]string{
			"method",
			"route",
			"status",
		},
	),

	requestDuration: prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "torus",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "End-to-end HTTP request latency in seconds.",
			Buckets: []float64{
				0.0001,
				0.00025,
				0.0005,
				0.001,
				0.0025,
				0.005,
				0.010,
				0.025,
				0.050,
				0.100,
				0.250,
				0.500,
				1.000,
			},
		},
		[]string{
			"method",
			"route",
		},
	),

	inflight: prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "torus",
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "Current number of HTTP requests being processed.",
		},
	),
}
