package integration

import (
	"strings"
	"testing"
	"time"
)

func TestMetricsEndpoint(t *testing.T) {
	backend := newBackend(t, "backend")
	defer backend.Close()

	configPath := newTempConfig(t)

	writeConfig(
		t,
		configPath,
		"127.0.0.1:0",
		backend.URL,
	)

	_, server, _, baseURL := startProxy(t, configPath)

	defer func() {
		if err := server.Shutdown(5 * time.Second); err != nil {
			t.Errorf("shutdown server: %v", err)
		}
	}()

	metrics := httpGet(t, baseURL+"/metrics")

	expected := []string{
		"# HELP torus_http_requests_in_flight",
		"# HELP torus_backend_up",
		"# HELP torus_runtime_build_duration_seconds",
		"# HELP torus_runtime_generation",
	}

	for _, metric := range expected {
		if !strings.Contains(metrics, metric) {
			t.Fatalf("expected metrics output to contain %q", metric)
		}
	}
}
