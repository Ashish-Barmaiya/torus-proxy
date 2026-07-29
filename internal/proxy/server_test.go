package proxy

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"torus-proxy/internal/routing"
	"torus-proxy/internal/runtime"
	"torus-proxy/internal/service"
	"torus-proxy/internal/upstream"
)

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func buildRuntime(t *testing.T, generation uint64, targetURLs []string) *runtime.Runtime {
	t.Helper()

	var backends []*upstream.Backend

	for _, url := range targetURLs {
		b, err := upstream.NewBackend(url)
		if err != nil {
			t.Fatalf("failed to build test backend node: %v", err)
		}
		backends = append(backends, b)
	}

	svc := service.NewService(backends)

	router := routing.NewRouter()
	router.AddRoute("/api", svc)

	return runtime.NewRuntime(
		generation,
		"",
		router,
		nil,
		nil,
	)
}

func setupProxy(t *testing.T, targetURLs []string) *Server {
	t.Helper()
	return NewServer(buildRuntime(t, 1, targetURLs), testLogger)
}

func TestProxyFlow_Basic(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("backend_response"))
	}))
	defer backend.Close()

	proxy := setupProxy(t, []string{backend.URL})

	req := httptest.NewRequest("GET", "/api", nil)
	w := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if !strings.Contains(string(body), "backend_response") {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestProxyFlow_HeadersAndTracing(t *testing.T) {
	var capturedTrackingID string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTrackingID = r.Header.Get("X-Request-ID")
		_, _ = w.Write([]byte("ok"))
	}))

	defer backend.Close()

	proxy := setupProxy(t, []string{backend.URL})

	req := httptest.NewRequest("GET", "/api", nil)
	w := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(w, req)

	if capturedTrackingID == "" {
		t.Error("expected proxy to inject a tracing X-Request-ID, got empty string")
	}
	if len(capturedTrackingID) != 36 {
		t.Errorf("expected standard crypto UUID format (36 chars), got length %d", len(capturedTrackingID))
	}
}

func TestProxyFlow_NotFound(t *testing.T) {
	proxy := setupProxy(t, nil)

	req := httptest.NewRequest("GET", "/unknown", nil)
	w := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Result().StatusCode)
	}
}

func TestProxyFlow_LoadBalancing(t *testing.T) {
	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("S1"))
	}))
	defer backend1.Close()

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("S2"))
	}))
	defer backend2.Close()

	proxy := setupProxy(t, []string{backend1.URL, backend2.URL})

	results := make(map[string]int)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/api", nil)
		w := httptest.NewRecorder()

		proxy.Handler().ServeHTTP(w, req)

		body, _ := io.ReadAll(w.Result().Body)
		results[string(body)]++
	}

	if len(results) < 2 {
		t.Fatalf("expected load balancing across backends, got: %v", results)
	}
}

func TestProxyFlow_Concurrency(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer backend.Close()

	proxy := setupProxy(t, []string{backend.URL})

	defer func() {
		if err := proxy.Shutdown(5 * time.Second); err != nil {
			t.Fatal(err)
		}
	}()

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest("GET", "/api", nil)
			w := httptest.NewRecorder()

			proxy.Handler().ServeHTTP(w, req)

			if w.Result().StatusCode != http.StatusOK {
				t.Errorf("unexpected status: %d", w.Result().StatusCode)
			}
		}()
	}

	wg.Wait()
}

func TestProxyFlow_BackendFailure(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	proxy := setupProxy(t, []string{backend.URL})

	// simulate failure by forcing connection closure error
	backend.Close()

	req := httptest.NewRequest("GET", "/api", nil)
	w := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Result().StatusCode)
	}
}

func TestServerReload_RuntimeSwap(t *testing.T) {
	backendA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("A"))
	}))
	defer backendA.Close()

	backendB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("B"))
	}))
	defer backendB.Close()

	rt1 := buildRuntime(t, 1, []string{backendA.URL})
	rt2 := buildRuntime(t, 2, []string{backendB.URL})

	server := NewServer(rt1, testLogger)

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	if string(body) != "A" {
		t.Fatalf("expected backend A before reload, got %q", body)
	}

	server.Reload(rt2)

	req = httptest.NewRequest(http.MethodGet, "/api", nil)
	w = httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	body, _ = io.ReadAll(w.Result().Body)
	if string(body) != "B" {
		t.Fatalf("expected backend B after reload, got %q", body)
	}
}

func TestServerReload_WaitsForActiveRequests(t *testing.T) {
	block := make(chan struct{})

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
		_, _ = w.Write([]byte("done"))
	}))
	defer backend.Close()

	rt1 := buildRuntime(t, 1, []string{backend.URL})
	rt2 := buildRuntime(t, 2, []string{backend.URL})

	server := NewServer(rt1, testLogger)

	done := make(chan struct{})

	go func() {
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	reloadDone := make(chan struct{})

	go func() {
		server.Reload(rt2)
		close(reloadDone)
	}()

	select {
	case <-reloadDone:
		t.Fatal("reload returned before active request completed")
	case <-time.After(100 * time.Millisecond):
		// expected
	}

	close(block)

	<-done
	<-reloadDone
}
