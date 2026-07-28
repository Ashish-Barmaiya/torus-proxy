package observability

import "github.com/prometheus/client_golang/prometheus"

var runtimeMetrics = struct {
	reloadsTotal *prometheus.CounterVec
	generation   prometheus.Gauge
	buildTime    prometheus.Histogram
}{
	reloadsTotal: prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "torus",
			Subsystem: "runtime",
			Name:      "reloads_total",
			Help:      "Total number of runtime reload attempts partitioned by result.",
		},
		[]string{
			"result",
		},
	),

	generation: prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "torus",
			Subsystem: "runtime",
			Name:      "generation",
			Help:      "Current active runtime generation.",
		},
	),

	buildTime: prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "torus",
			Subsystem: "runtime",
			Name:      "build_duration_seconds",
			Help:      "Time required to build a new runtime generation.",
			Buckets: []float64{
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
				2.500,
				5.000,
			},
		},
	),
}
