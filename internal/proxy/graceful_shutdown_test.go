package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"torus-proxy/internal/routing"
	"torus-proxy/internal/service"
	"torus-proxy/internal/upstream"
)

func TestGracefulShutdown(t *testing.T) {
	// Slow mock backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer backend.Close()

	b, err := upstream.NewBackend(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	b.Proxy.Transport.(*http.Transport).ResponseHeaderTimeout = 10 * time.Second

	svc := service.NewService([]*upstream.Backend{b})
	router := routing.NewRouter()
	router.AddRoute("/api", svc)

	srv := NewServer(router, testLogger, nil)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()

	// Server setup
	mux := http.NewServeMux()
	mux.Handle("/", srv.Handler())
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if srv.ready.Load() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ready"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))
		}
	})

	baseCtx, forceCancel := context.WithCancel(context.Background())
	srv.baseCtx = baseCtx
	srv.forceCancel = forceCancel

	srv.srv = &http.Server{
		Addr:        addr,
		Handler:     mux,
		BaseContext: func(l net.Listener) context.Context { return baseCtx },
		// long timeouts
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	srv.ready.Store(true)

	go func() {
		_ = srv.srv.Serve(listener)
	}()

	// Wait until the server is ready
	readyURL := fmt.Sprintf("http://%s/readyz", addr)
	waitForReady(t, readyURL, 5*time.Second)

	type result struct {
		status int
		body   string
		err    error
	}

	resCh := make(chan result, 1)
	go func() {
		req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/api", addr), nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			resCh <- result{err: err}
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		resCh <- result{status: resp.StatusCode, body: string(body)}
	}()

	time.Sleep(500 * time.Millisecond)

	// Trigger graceful shutdown
	shutdownErr := srv.Shutdown(5 * time.Second) // wait up to 5s
	if shutdownErr != nil {
		t.Fatalf("graceful shutdown failed: %v", shutdownErr)
	}

	// The in flight request must complete successfully
	res := <-resCh
	if res.err != nil {
		t.Fatalf("request failed: %v", res.err)
	}
	if res.status != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", res.status)
	}
	if res.body != "OK" {
		t.Errorf("expected body 'OK', got '%s'", res.body)
	}

	// After shutdown, new requests should be refused because the listener is closed.
	_, err = http.Get(fmt.Sprintf("http://%s/", addr))
	if err == nil {
		t.Error("expected connection refused after shutdown, but request succeeded")
	}

	// Readiness should now return 503
	resp, err := http.Get(readyURL)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("expected readiness 503 after shutdown, got %d", resp.StatusCode)
		}
	} // if err, that is also a sign that the server is down
}

// helper to poll readiness until it returns 200
func waitForReady(t *testing.T, url string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server did not become ready within %v", timeout)
}
