package integration

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
	"torus-proxy/internal/proxy"
	"torus-proxy/internal/reload"
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

	_, _ = fmt.Fprintln(file, "apiVersion: v1")
	_, _ = fmt.Fprintln(file)
	_, _ = fmt.Fprintln(file, "server:")
	_, _ = fmt.Fprintf(file, "  addr: %q\n", listenAddr)
	_, _ = fmt.Fprintln(file)
	_, _ = fmt.Fprintln(file, "health:")
	_, _ = fmt.Fprintln(file, "  interval_ms: 5000")
	_, _ = fmt.Fprintln(file, "  timeout_ms: 2000")
	_, _ = fmt.Fprintln(file, "  path: /health")
	_, _ = fmt.Fprintln(file)
	_, _ = fmt.Fprintln(file, "routes:")
	_, _ = fmt.Fprintln(file, "  - path: /api")
	_, _ = fmt.Fprintln(file, "    upstream:")

	for _, u := range upstreams {
		_, _ = fmt.Fprintf(file, "      - %q\n", u)
	}

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

// TestHotReload_Functional verifies that a runtime reload immediately affects
// newly arriving requests without restarting the server.
//
// Flow:
//  1. Start Backend A.
//  2. Configure the proxy to route traffic to Backend A.
//  3. Verify requests reach Backend A.
//  4. Rewrite the configuration to point to Backend B.
//  5. Trigger a runtime reload.
//  6. Verify subsequent requests are routed to Backend B.
//
// This exercises the complete runtime reload pipeline:
//
//	Config -> Runtime Builder -> Reload Manager -> Runtime Swap -> Request Routing
func TestHotReload_Functional(t *testing.T) {
	backendA := newBackend(t, "backend-a")
	defer backendA.Close()

	backendB := newBackend(t, "backend-b")
	defer backendB.Close()

	configPath := newTempConfig(t)

	// Initial configuration points to Backend A.
	writeConfig(
		t,
		configPath,
		"127.0.0.1:0",
		backendA.URL,
	)

	manager, server, listenAddr, baseURL := startProxy(t, configPath)
	defer func() {
		if err := server.Shutdown(5 * time.Second); err != nil {
			t.Errorf("shutdown server: %v", err)
		}
	}()

	// Verify initial routing.
	got := httpGet(t, baseURL+"/api")
	if got != "backend-a" {
		t.Fatalf("expected backend-a, got %q", got)
	}

	// Rewrite configuration to Backend B.
	writeConfig(
		t,
		configPath,
		listenAddr,
		backendB.URL,
	)

	// Reload runtime.
	if err := manager.Reload(); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	// Verify requests now go to Backend B.
	got = httpGet(t, baseURL+"/api")
	if got != "backend-b" {
		t.Fatalf("expected backend-b after reload, got %q", got)
	}
}

// TestHotReload_ConcurrentTraffic verifies that runtime reloads remain safe
// while the proxy is actively serving concurrent traffic.
//
// Flow:
//  1. Start the proxy using Backend A.
//  2. Launch many concurrent clients issuing requests.
//  3. Repeatedly reload the runtime, alternating between Backend A and B.
//  4. Verify every request succeeds throughout the reload process.
//
// The test ensures:
//   - No request failures.
//   - No corrupted responses.
//   - Runtime swaps remain transparent to clients.
//   - Synchronization between request handling and runtime replacement is safe.
//
// When executed with `go test -race`, this test also validates that runtime
// replacement does not introduce data races.
func TestHotReload_ConcurrentTraffic(t *testing.T) {
	backendA := newBackend(t, "backend-a")
	defer backendA.Close()

	backendB := newBackend(t, "backend-b")
	defer backendB.Close()

	configPath := newTempConfig(t)

	writeConfig(
		t,
		configPath,
		"127.0.0.1:0",
		backendA.URL,
	)

	manager, server, listenAddr, baseURL := startProxy(t, configPath)

	defer func() {
		if err := server.Shutdown(5 * time.Second); err != nil {
			t.Errorf("shutdown server: %v", err)
		}
	}()

	const (
		numWorkers        = 50
		requestsPerWorker = 100
		reloads           = 10
	)

	var (
		wg    sync.WaitGroup
		errCh = make(chan error, numWorkers*requestsPerWorker)
	)

	// Continuous traffic.
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			client := &http.Client{
				Timeout: 2 * time.Second,
			}

			for j := 0; j < requestsPerWorker; j++ {
				resp, err := client.Get(baseURL + "/api")
				if err != nil {
					errCh <- fmt.Errorf("request failed: %w", err)
					continue
				}

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()

				if err != nil {
					errCh <- err
					continue
				}

				if resp.StatusCode != http.StatusOK {
					errCh <- fmt.Errorf("unexpected status %d", resp.StatusCode)
					continue
				}

				switch string(body) {
				case "backend-a", "backend-b":
					// expected
				default:
					errCh <- fmt.Errorf("unexpected response %q", string(body))
				}
			}
		}()
	}

	// Reload repeatedly while traffic is flowing.
	for i := range reloads {

		var target string

		if i%2 == 0 {
			target = backendB.URL
		} else {
			target = backendA.URL
		}

		writeConfig(
			t,
			configPath,
			listenAddr,
			target,
		)

		if err := manager.Reload(); err != nil {
			t.Fatalf("reload %d failed: %v", i, err)
		}
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestGracefulShutdown_DrainsRequests verifies graceful shutdown semantics.
//
// Flow:
//  1. Start a backend that intentionally delays responses.
//  2. Begin a long-running request.
//  3. Initiate server shutdown while the request is in flight.
//  4. Verify the in-flight request completes successfully.
//  5. Verify shutdown completes cleanly.
//  6. Verify new connections are rejected after shutdown begins.
//
// This confirms that Torus drains active requests during shutdown instead of
// terminating them prematurely.
func TestGracefulShutdown_DrainsRequests(t *testing.T) {
	backend := newSlowBackend(t, 500*time.Millisecond)
	defer backend.Close()

	configPath := newTempConfig(t)

	writeConfig(
		t,
		configPath,
		"127.0.0.1:0",
		backend.URL,
	)

	_, server, _, baseURL := startProxy(t, configPath)

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	errCh := make(chan error, 1)

	// Start a long-running request.
	go func() {
		resp, err := client.Get(baseURL + "/api")
		if err != nil {
			errCh <- err
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			errCh <- err
			return
		}

		if resp.StatusCode != http.StatusOK {
			errCh <- fmt.Errorf("unexpected status %d", resp.StatusCode)
			return
		}

		if string(body) != "slow-backend" {
			errCh <- fmt.Errorf("unexpected body %q", body)
			return
		}

		errCh <- nil
	}()

	// Give the request time to reach the backend.
	time.Sleep(100 * time.Millisecond)

	shutdownDone := make(chan error, 1)

	go func() {
		shutdownDone <- server.Shutdown(2 * time.Second)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight request did not complete")
	}

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("shutdown failed: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown timed out")
	}

	// New connections should fail.
	_, err := client.Get(baseURL + "/api")
	if err == nil {
		t.Fatal("expected new request to fail after shutdown")
	}
}
