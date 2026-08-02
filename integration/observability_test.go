package integration

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestMetricsEndpoint verifies that the proxy exposes the expected metrics
// endpoint help text.
//
// Flow:
//  1. Start a test backend and proxy.
//  2. Request the /metrics endpoint.
//  3. Verify the output contains the expected help text for core metrics.
//
// This ensures the observability surface is registered and exported correctly.
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

// TestRequestMetrics verifies that request and backend metrics are emitted for
// proxied traffic.
//
// Flow:
//  1. Start a test backend and proxy.
//  2. Issue several requests through the proxy.
//  3. Read /metrics and verify the request-count and duration metrics are
//     present.
//
// This ensures request-level observability is updated for each proxied call.
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

// TestRuntimeMetrics verifies that runtime generation and reload metrics are
// updated after successful and failed reloads.
//
// Flow:
//  1. Start a proxy with a valid configuration.
//  2. Verify the initial runtime metrics are emitted.
//  3. Trigger a successful reload and verify the generation and reload-count
//     metrics advance.
//  4. Trigger a failed reload and verify the failure counter increments without
//     changing the generation.
//
// This validates the runtime observability lifecycle during config changes.
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

// TestBackendHealthMetrics verifies that backend health transitions are
// reflected in the exported health metrics.
//
// Flow:
//  1. Start a proxy with a health-checking backend.
//  2. Wait until the backend is marked healthy.
//  3. Simulate backend failure.
//  4. Verify the health metric transitions to unhealthy.
//
// This ensures health probes are surfaced through the observability metrics.
func TestBackendHealthMetrics(t *testing.T) {
	resetTestState(t)

	backend := newBackend(t, "healthy")
	defer backend.Close()

	configPath := newTempConfig(t)

	writeHealthConfig(
		t,
		configPath,
		"127.0.0.1:0",
		100,
		100,
		backend.URL,
	)

	_, server, _, baseURL := startProxy(t, configPath)
	defer func() {
		if err := server.Shutdown(5 * time.Second); err != nil {
			t.Fatalf("shutdown proxy: %v", err)
		}
	}()

	// Wait until the first successful health probe updates the metric
	waitForBackendHealth(t, baseURL, backend.URL, true)

	// Simulate backend failure.
	backend.Close()

	// Wait until the health checker marks the backend unhealthy
	waitForBackendHealth(t, baseURL, backend.URL, false)
}
