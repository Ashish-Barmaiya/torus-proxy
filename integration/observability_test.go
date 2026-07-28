package integration

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestMetricsEndpoint(t *testing.T) {
	resetTestState(t)

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
	resetTestState(t)

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

func TestRuntimeMetrics(t *testing.T) {
	resetTestState(t)

	backend := newBackend(t, "backend")
	defer backend.Close()

	configPath := newTempConfig(t)

	writeConfig(
		t,
		configPath,
		"127.0.0.1:0",
		backend.URL,
	)

	manager, server, _, baseURL := startProxy(t, configPath)

	defer func() {
		if err := server.Shutdown(5 * time.Second); err != nil {
			t.Errorf("shutdown server: %v", err)
		}
	}()

	// Initial runtime metrics
	metrics := httpGet(t, baseURL+"/metrics")

	expected := []string{
		"torus_runtime_generation 1",
		"torus_runtime_build_duration_seconds_count 1",
	}

	for _, metric := range expected {
		if !strings.Contains(metrics, metric) {
			t.Fatalf("expected metrics output to contain %q", metric)
		}
	}

	// Successful reload
	writeConfig(
		t,
		configPath,
		"127.0.0.1:0",
		backend.URL,
	)

	if err := manager.Reload(); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	metrics = httpGet(t, baseURL+"/metrics")

	expected = []string{
		"torus_runtime_generation 2",
		"torus_runtime_build_duration_seconds_count 2",
		`torus_runtime_reloads_total{result="success"} 1`,
	}

	for _, metric := range expected {
		if !strings.Contains(metrics, metric) {
			t.Fatalf("expected metrics output to contain %q", metric)
		}
	}

	// Failed reload
	if err := os.WriteFile(configPath, []byte("invalid yaml"), 0644); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	if err := manager.Reload(); err == nil {
		t.Fatal("expected reload to fail")
	}

	metrics = httpGet(t, baseURL+"/metrics")

	expected = []string{
		"torus_runtime_generation 2",
		"torus_runtime_build_duration_seconds_count 2",
		`torus_runtime_reloads_total{result="success"} 1`,
		`torus_runtime_reloads_total{result="failure"} 1`,
	}

	for _, metric := range expected {
		if !strings.Contains(metrics, metric) {
			t.Fatalf("expected metrics output to contain %q", metric)
		}
	}
}
