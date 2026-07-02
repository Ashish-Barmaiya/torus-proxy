package proxy

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"torus-proxy/internal/routing"
	"torus-proxy/internal/service"
	"torus-proxy/internal/upstream"
)

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func setupProxy(t *testing.T, targetURLs []string) *Server {
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
	return NewServer(router, testLogger, nil)
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
