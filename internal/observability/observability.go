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
	httpMetrics.requestsTotal.
		WithLabelValues(method, normalizeRoute(route), strconv.Itoa(status)).
		Inc()
}

// ObserveHTTPRequestDuration records the end-to-end request latency
func ObserveHTTPRequestDuration(method, route string, duration time.Duration) {
	httpMetrics.requestDuration.
		WithLabelValues(method, normalizeRoute(route)).
		Observe(duration.Seconds())
}

// TrackInflight increments the in-flight request gauge and returns
// a cleanup function that must be deferred by the caller
func TrackInflight() func() {
	httpMetrics.inflight.Inc()

	return func() {
		httpMetrics.inflight.Dec()
	}
}

// RecordBackendRequest increments the backend request counter
func RecordBackendRequest(service, backend string) {
	backendMetrics.requestsTotal.
		WithLabelValues(service, backend).
		Inc()
}

// ObserveBackendRequestDuration records backend latency
func ObserveBackendRequestDuration(
	service,
	backend string,
	duration time.Duration,
) {
	backendMetrics.requestDuration.
		WithLabelValues(service, backend).
		Observe(duration.Seconds())
}

// RecordBackendError increments the backend error counter
func RecordBackendError(service, backend string) {
	backendMetrics.errorsTotal.
		WithLabelValues(service, backend).
		Inc()
}

// SetBackendHealth updates the backend health gauge
func SetBackendHealth(service, backend string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}

	backendMetrics.up.
		WithLabelValues(service, backend).
		Set(value)
}

// RecordRuntimeReload records the result of a runtime reload
func RecordRuntimeReload(success bool) {
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
	runtimeMetrics.generation.Set(float64(generation))
}

// ObserveRuntimeBuildDuration records the runtime build duration
func ObserveRuntimeBuildDuration(duration time.Duration) {
	runtimeMetrics.buildTime.Observe(duration.Seconds())
}
