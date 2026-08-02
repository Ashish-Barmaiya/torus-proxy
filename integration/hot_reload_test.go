package integration

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

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
	resetTestState(t)

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
	resetTestState(t)

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
	resetTestState(t)

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
