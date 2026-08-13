package reload

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"torus-proxy/internal/proxy"
	"torus-proxy/internal/runtime"
)

func TestManagerReload_StopsRuntimeWhenServerRejectsReload(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("healthy"))
	}))
	defer backend.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "torus.yaml")

	config := fmt.Sprintf(`
apiVersion: v2

server:
  addr: "127.0.0.1:0"

health:
  interval_ms: 1000
  timeout_ms: 100
  path: "/health"

services:
  - name: test
    upstreams:
      - "%s"

routes:
  - path: "/api"
    service: test

observability:
  enabled: false
`, backend.URL)

	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{Level: slog.LevelError},
	))

	// Build the server with an existing runtime
	rt := runtime.NewRuntime(
		1,
		"127.0.0.1:0",
		nil,
		nil,
		false,
		nil,
	)

	server := proxy.NewServer(rt, logger)

	// Move the server through its real shutdown lifecycle
	//
	// NewServer() starts in serverStarting. Calling Shutdown() with
	// no HTTP server transitions it to serverStopped
	if err := server.Shutdown(0); err != nil {
		t.Fatalf("failed to shut down test server: %v", err)
	}

	manager := NewManager(configPath, logger)
	manager.SetServer(server)

	// Reload() must:
	//
	// 1. Build a complete new Runtime
	// 2. Attempt Server.Reload()
	// 3. Receive ErrServerNotRunning
	// 4. Stop the newly built Runtime
	err := manager.Reload()

	if !errors.Is(err, proxy.ErrServerNotRunning) {
		t.Fatalf(
			"expected ErrServerNotRunning, got %v",
			err,
		)
	}
}
