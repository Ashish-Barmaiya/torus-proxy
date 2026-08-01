package observability

import (
	"strconv"
	"time"
)

func normalizeRoute(route string) string {
	if route == "" {
		return "unmatched"
	}
	return route
}

// RecordHTTPRequest increments the total HTTP request counter
func RecordHTTPRequest(method, route string, status int) {
	if !enabled {
		return
	}

	httpMetrics.requestsTotal.
		WithLabelValues(method, normalizeRoute(route), strconv.Itoa(status)).
		Inc()
}

// ObserveHTTPRequestDuration records the end-to-end request latency
func ObserveHTTPRequestDuration(method, route string, duration time.Duration) {
	if !enabled {
		return
	}

	httpMetrics.requestDuration.
		WithLabelValues(method, normalizeRoute(route)).
		Observe(duration.Seconds())
}

// TrackInflight increments the in-flight request gauge and returns
// a cleanup function that must be deferred by the caller
func TrackInflight() func() {
	if !enabled {
		return func() {}
	}

	httpMetrics.inflight.Inc()

	return func() {
		httpMetrics.inflight.Dec()
	}
}

// RecordBackendRequest increments the backend request counter
func RecordBackendRequest(backend string) {
	if !enabled {
		return
	}
	backendMetrics.requestsTotal.
		WithLabelValues(backend).
		Inc()
}

// ObserveBackendRequestDuration records backend latency
func ObserveBackendRequestDuration(
	backend string,
	duration time.Duration,
) {
	if !enabled {
		return
	}
	backendMetrics.requestDuration.
		WithLabelValues(backend).
		Observe(duration.Seconds())
}

// RecordBackendError increments the backend error counter
func RecordBackendError(backend string) {
	if !enabled {
		return
	}
	backendMetrics.errorsTotal.
		WithLabelValues(backend).
		Inc()
}

// SetBackendHealth updates the backend health gauge
func SetBackendHealth(backend string, healthy bool) {
	if !enabled {
		return
	}

	value := 0.0
	if healthy {
		value = 1.0
	}

	backendMetrics.up.
		WithLabelValues(backend).
		Set(value)
}

// RecordRuntimeReload records the result of a runtime reload
func RecordRuntimeReload(success bool) {
	if !enabled {
		return
	}

	result := "failure"
	if success {
		result = "success"
	}

	runtimeMetrics.reloadsTotal.
		WithLabelValues(result).
		Inc()
}

// SetRuntimeGeneration updates the current runtime generation
func SetRuntimeGeneration(generation uint64) {
	if !enabled {
		return
	}
	runtimeMetrics.generation.Set(float64(generation))
}

// ObserveRuntimeBuildDuration records the runtime build duration
func ObserveRuntimeBuildDuration(duration time.Duration) {
	if !enabled {
		return
	}
	runtimeMetrics.buildTime.Observe(duration.Seconds())
}
