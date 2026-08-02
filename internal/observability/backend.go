package observability

import "github.com/prometheus/client_golang/prometheus"

type backendMetricsStruct struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	errorsTotal     *prometheus.CounterVec
	up              *prometheus.GaugeVec
}

func newBackendMetrics() backendMetricsStruct {
	return backendMetricsStruct{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "torus",
				Subsystem: "backend",
				Name:      "requests_total",
				Help:      "Total number of requests forwarded to backend servers.",
			},
			[]string{
				"backend",
			},
		),

		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "torus",
				Subsystem: "backend",
				Name:      "request_duration_seconds",
				Help:      "Backend request latency in seconds.",
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
				"backend",
			},
		),

		errorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "torus",
				Subsystem: "backend",
				Name:      "errors_total",
				Help:      "Total number of backend request failures.",
			},
			[]string{
				"backend",
			},
		),

		up: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "torus",
				Subsystem: "backend",
				Name:      "up",
				Help:      "Current backend health status (1 = healthy, 0 = unhealthy).",
			},
			[]string{
				"backend",
			},
		),
	}
}

var backendMetrics = newBackendMetrics()
