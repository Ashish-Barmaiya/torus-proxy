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
	"torus-proxy/internal/runtime"
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

	b, err := upstream.NewBackend(backend.URL, false)
	if err != nil {
		t.Fatal(err)
	}

	svc := service.NewService([]*upstream.Backend{b})
	router := routing.NewRouter()
	router.AddRoute("/api", svc)

	rt := runtime.NewRuntime(1, "", router, nil, false, nil)
	srv := NewServer(rt, testLogger)

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

// TestGracefulShutdown_ForceCancellation verifies that an in-flight request
// is cancelled when the graceful shutdown deadline is exceeded.
//
// The test exercises the full cancellation path:
//
//	shutdown timeout → forceCancel → request context cancellation →
//	upstream request exits → forced shutdown completes
func TestGracefulShutdown_ForceCancellation(t *testing.T) {
	requestStarted := make(chan struct{})
	requestCanceled := make(chan struct{})

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)

		<-r.Context().Done()

		close(requestCanceled)
	}))
	defer backend.Close()

	rt := buildRuntime(t, 1, []string{backend.URL})
	server := NewServer(rt, testLogger)

	startDone := make(chan error, 1)

	go func() {
		startDone <- server.Start("127.0.0.1:0")
	}()

	addr := server.WaitStarted()

	requestDone := make(chan error, 1)

	go func() {
		resp, err := http.Get("http://" + addr + "/api")
		if err != nil {
			requestDone <- err
			return
		}

		defer resp.Body.Close()

		_, err = io.ReadAll(resp.Body)
		requestDone <- err
	}()

	// Ensure the request has reached the upstream before shutdown begins.
	select {
	case <-requestStarted:
		// Expected.
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for upstream request to start")
	}

	// Force the graceful shutdown deadline to expire while the request is
	// still blocked in the upstream.
	if err := server.Shutdown(100 * time.Millisecond); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	// forceCancel() should propagate cancellation through the request context
	// to the upstream ReverseProxy request.
	select {
	case <-requestCanceled:
		// Expected.
	case <-time.After(2 * time.Second):
		t.Fatal("upstream request was not cancelled")
	}

	// The proxy request must also terminate.
	select {
	case <-requestDone:
		// Expected.
	case <-time.After(2 * time.Second):
		t.Fatal("proxy request did not terminate after forced cancellation")
	}

	if got := serverState(server.state.Load()); got != serverStopped {
		t.Fatalf(
			"expected serverStopped after forced shutdown, got %v",
			got,
		)
	}

	if server.ready.Load() {
		t.Fatal("server remained ready after forced shutdown")
	}

	// Start should return normally after http.Server.Shutdown().
	select {
	case err := <-startDone:
		if err != nil {
			t.Fatalf("server start returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server Start did not return after shutdown")
	}
}

// TestServerShutdown_MarksUnreadyBeforeDrain verifies that shutdown clears
// readiness as soon as the server enters the shutting-down state, while
// existing requests are still allowed to drain.
func TestServerShutdown_MarksUnreadyBeforeDrain(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)

		<-releaseRequest

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("done"))
	}))
	defer backend.Close()

	rt := buildRuntime(t, 1, []string{backend.URL})
	server := NewServer(rt, testLogger)
	markServerRunningForTest(server)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	addr := listener.Addr().String()

	mux := http.NewServeMux()
	mux.Handle("/", server.Handler())

	server.srv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		_ = server.srv.Serve(listener)
	}()

	requestDone := make(chan error, 1)

	go func() {
		resp, err := http.Get("http://" + addr + "/api")
		if err != nil {
			requestDone <- err
			return
		}

		defer resp.Body.Close()

		_, err = io.ReadAll(resp.Body)
		requestDone <- err
	}()

	// Ensure an existing request is in flight before shutdown begins
	select {
	case <-requestStarted:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for request to start")
	}

	shutdownStarted := make(chan error, 1)

	go func() {
		shutdownStarted <- server.Shutdown(5 * time.Second)
	}()

	// Wait until shutdown has entered its lifecycle transition
	deadline := time.Now().Add(2 * time.Second)

	for serverState(server.state.Load()) != serverShuttingDown {
		if time.Now().After(deadline) {
			t.Fatal("server did not enter shutting-down state")
		}

		time.Sleep(time.Millisecond)
	}

	// Readiness must be cleared immediately when shutdown begins, even though
	// the existing request is still draining
	if server.ready.Load() {
		t.Fatal("server remained ready during shutdown")
	}

	// The active request must still be allowed to complete
	close(releaseRequest)

	select {
	case err := <-requestDone:
		if err != nil {
			t.Fatalf("active request failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("active request did not complete")
	}

	select {
	case err := <-shutdownStarted:
		if err != nil {
			t.Fatalf("shutdown failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not complete")
	}
}
