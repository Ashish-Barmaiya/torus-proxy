package integration

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"torus-proxy/internal/observability"
	"torus-proxy/internal/proxy"
	"torus-proxy/internal/reload"

	runtimepkg "torus-proxy/internal/runtime"
)

// newBackend creates a lightweight HTTP backend used by integration tests.
// It exposes:
//
//   - GET /health -> 200 OK
//   - GET /api    -> returns the supplied response body
//
// Each backend represents an independent upstream service, allowing tests to
// verify routing, runtime reloads, and backend switching.
func newBackend(t *testing.T, response string) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, response)
	})

	return httptest.NewServer(mux)
}

// newSlowBackend creates an HTTP backend that intentionally delays responses.
//
// It is used to verify graceful shutdown behavior by keeping requests
// in-flight while the proxy begins shutting down.
func newSlowBackend(t *testing.T, delay time.Duration) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		_, _ = io.WriteString(w, "slow-backend")
	})

	return httptest.NewServer(mux)
}

// newTempConfig creates an isolated temporary configuration file for a test.
//
// Each test receives its own configuration file to avoid interference between
// parallel or repeated test runs.
func newTempConfig(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	return filepath.Join(dir, "torus.yaml")
}

// writeConfig generates a Torus configuration file pointing to the supplied
// upstream backends.
//
// Tests rewrite this configuration to simulate runtime changes before invoking
// the reload manager.
func writeConfig(t *testing.T, path, listenAddr string, upstreams ...string) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create config: %v", err)
	}

	defer file.Close()

	_, _ = fmt.Fprintln(file, "apiVersion: v2")
	_, _ = fmt.Fprintln(file)

	_, _ = fmt.Fprintln(file, "server:")
	_, _ = fmt.Fprintf(file, "  addr: %q\n", listenAddr)
	_, _ = fmt.Fprintln(file)

	_, _ = fmt.Fprintln(file, "health:")
	_, _ = fmt.Fprintln(file, "  interval_ms: 5000")
	_, _ = fmt.Fprintln(file, "  timeout_ms: 2000")
	_, _ = fmt.Fprintln(file, "  path: /health")
	_, _ = fmt.Fprintln(file)

	_, _ = fmt.Fprintln(file, "services:")
	_, _ = fmt.Fprintln(file, "  - name: api")
	_, _ = fmt.Fprintln(file, "    upstreams:")

	for _, u := range upstreams {
		_, _ = fmt.Fprintf(file, "      - %q\n", u)
	}

	_, _ = fmt.Fprintln(file)

	_, _ = fmt.Fprintln(file, "routes:")
	_, _ = fmt.Fprintln(file, "  - path: /api")
	_, _ = fmt.Fprintln(file, "    service: api")

	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
}

// waitUntilReady blocks until the proxy reports itself ready via /readyz.
//
// This ensures the server has completed startup before requests are issued,
// avoiding races between server initialization and the test.
func waitUntilReady(t *testing.T, baseURL string) {
	t.Helper()

	client := &http.Client{
		Timeout: time.Second,
	}

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {

		resp, err := client.Get(baseURL + "/readyz")
		if err == nil {
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				return
			}
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("server never became ready")
}

// httpGet performs an HTTP GET request and returns the response body.
func httpGet(t *testing.T, url string) string {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	return string(body)
}

// startProxy boots a production Torus server using a real configuration file.
//
// The helper builds the runtime, starts the HTTP server, waits until it is
// accepting traffic, and returns the reload manager, server instance,
// listening address, and base URL for use by the test.
func startProxy(
	t *testing.T,
	configPath string,
) (*reload.Manager, *proxy.Server, string, string) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	manager := reload.NewManager(configPath, logger)

	rt, err := manager.BuildInitialRuntime()
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	server := proxy.NewServer(rt, logger)
	manager.SetServer(server)

	go func() {
		if err := server.Start(rt.Addr); err != nil &&
			err != http.ErrServerClosed {
			t.Errorf("server exited: %v", err)
		}
	}()

	addr := server.WaitStarted()

	baseURL := "http://" + addr

	waitUntilReady(t, baseURL)

	return manager, server, addr, baseURL
}

// resetTestState resets observability and runtime state between tests.
//
// It clears the test-specific observability registry and runtime generation
// counters so each integration test starts from a clean slate.
func resetTestState(t *testing.T) {
	t.Helper()

	observability.ResetForTesting()
	runtimepkg.ResetGenerationForTesting()
}

// writeHealthConfig generates a Torus configuration file with a custom health
// check interval and timeout.
//
// It is intended for integration tests that need faster health state
// transitions than the default configuration.
func writeHealthConfig(
	t *testing.T,
	path, listenAddr string,
	intervalMS, timeoutMS int,
	upstreams ...string,
) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create config: %v", err)
	}

	defer file.Close()

	_, _ = fmt.Fprintln(file, "apiVersion: v2")
	_, _ = fmt.Fprintln(file)

	_, _ = fmt.Fprintln(file, "server:")
	_, _ = fmt.Fprintf(file, "  addr: %q\n", listenAddr)
	_, _ = fmt.Fprintln(file)

	_, _ = fmt.Fprintln(file, "health:")
	_, _ = fmt.Fprintf(file, "  interval_ms: %d\n", intervalMS)
	_, _ = fmt.Fprintf(file, "  timeout_ms: %d\n", timeoutMS)
	_, _ = fmt.Fprintln(file, "  path: /health")
	_, _ = fmt.Fprintln(file)

	_, _ = fmt.Fprintln(file, "services:")
	_, _ = fmt.Fprintln(file, "  - name: api")
	_, _ = fmt.Fprintln(file, "    upstreams:")

	for _, u := range upstreams {
		_, _ = fmt.Fprintf(file, "      - %q\n", u)
	}

	_, _ = fmt.Fprintln(file)

	_, _ = fmt.Fprintln(file, "routes:")
	_, _ = fmt.Fprintln(file, "  - path: /api")
	_, _ = fmt.Fprintln(file, "    service: api")

	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
}

// waitForBackendHealth waits until the backend health metric reaches the
// expected value.
//
// It polls the proxy metrics endpoint until the backend's health status matches
// the requested state or the timeout expires.
func waitForBackendHealth(
	t *testing.T,
	baseURL string,
	backendURL string,
	expected bool,
) {
	t.Helper()

	expectedValue := "0"
	if expected {
		expectedValue = "1"
	}

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		metrics := httpGet(t, baseURL+"/metrics")

		if strings.Contains(
			metrics,
			fmt.Sprintf(
				`torus_backend_up{backend="%s"} %s`,
				backendURL,
				expectedValue,
			),
		) {
			return
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf(
		"backend health never became %s for %q",
		expectedValue,
		backendURL,
	)
}
