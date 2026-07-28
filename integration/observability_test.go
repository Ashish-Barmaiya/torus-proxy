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

func TestRequestMetrics(t *testing.T) {
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

	// Generate traffic through the proxy
	const requests = 3

	for range requests {
		body := httpGet(t, baseURL+"/api")

		if body != "backend" {
			t.Fatalf("expected backend response, got %q", body)
		}
	}

	metrics := httpGet(t, baseURL+"/metrics")

	expected := []string{
		`torus_http_requests_total{method="GET",route="/api",status="200"} 3`,

		`torus_http_request_duration_seconds_count{method="GET",route="/api"} 3`,

		`torus_backend_requests_total{backend="` + backend.URL + `"} 3`,

		`torus_backend_request_duration_seconds_count{backend="` + backend.URL + `"} 3`,
	}

	for _, metric := range expected {
		if !strings.Contains(metrics, metric) {
			t.Fatalf("expected metrics output to contain %q", metric)
		}
	}
}
