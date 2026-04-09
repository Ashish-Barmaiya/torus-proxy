package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"torus-proxy/internal/routing"
	"torus-proxy/internal/service"
	"torus-proxy/internal/upstream"
)

func setupProxy(backends []*upstream.Backend) *Server {
	svc := service.NewService(backends)
	router := routing.NewRouter()
	router.AddRoute("/api", svc)
	return NewServer(router)
}

func TestProxyFlow_Basic(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("backend_response"))
	}))
	defer backend.Close()

	proxy := setupProxy([]*upstream.Backend{
		{URL: backend.URL},
	})

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

func TestProxyFlow_NotFound(t *testing.T) {
	proxy := setupProxy(nil)

	req := httptest.NewRequest("GET", "/unknown", nil)
	w := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Result().StatusCode)
	}
}

func TestProxyFlow_LoadBalancing(t *testing.T) {
	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("S1"))
	}))
	defer backend1.Close()

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("S2"))
	}))
	defer backend2.Close()

	proxy := setupProxy([]*upstream.Backend{
		{URL: backend1.URL},
		{URL: backend2.URL},
	})

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
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	proxy := setupProxy([]*upstream.Backend{
		{URL: backend.URL},
	})

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
		w.Write([]byte("ok"))
	}))

	proxy := setupProxy([]*upstream.Backend{
		{URL: backend.URL},
	})

	// simulate failure
	backend.Close()

	req := httptest.NewRequest("GET", "/api", nil)
	w := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Result().StatusCode)
	}
}