package observability

func ResetForTesting() {
	httpMetrics = newHTTPMetrics()
	backendMetrics = newBackendMetrics()
	runtimeMetrics = newRuntimeMetrics()

	registry = newRegistry()
}
