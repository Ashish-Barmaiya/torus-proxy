package proxy

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
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

// markServerRunningForTest marks a server as running for tests that bypass Start
func markServerRunningForTest(server *Server) {
	server.state.Store(uint32(serverRunning))
	server.ready.Store(true)
}

// buildRuntime creates a test runtime with the given upstreams routed under /api
func buildRuntime(t *testing.T, generation uint64, targetURLs []string) *runtime.Runtime {
	t.Helper()

	var backends []*upstream.Backend

	for _, url := range targetURLs {
		b, err := upstream.NewBackend(url, false)
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
		false,
		nil,
	)
}

// setupProxy creates a server backed by a test runtime
func setupProxy(t *testing.T, targetURLs []string) *Server {
	t.Helper()
	return NewServer(buildRuntime(t, 1, targetURLs), testLogger)
}

// TestProxyFlow_Basic verifies that requests are routed to the configured backend
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

// TestProxyFlow_HeadersAndTracing verifies that forwarded requests contain an X-Request-ID
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

// TestProxyFlow_NotFound verifies that unmatched routes return 404
func TestProxyFlow_NotFound(t *testing.T) {
	proxy := setupProxy(t, nil)

	req := httptest.NewRequest("GET", "/unknown", nil)
	w := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Result().StatusCode)
	}
}

// TestProxyFlow_LoadBalancing verifies round-robin distribution across backends
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

// TestProxyFlow_Concurrency verifies concurrent requests can be served safely
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

// TestProxyFlow_BackendFailure verifies that an upstream failure returns 502
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

// TestServerReload_RuntimeSwap verifies that a successful reload replaces the
// active runtime used by subsequent requests.
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
	markServerRunningForTest(server)

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	if string(body) != "A" {
		t.Fatalf("expected backend A before reload, got %q", body)
	}

	if err := server.Reload(rt2); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api", nil)
	w = httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	body, _ = io.ReadAll(w.Result().Body)
	if string(body) != "B" {
		t.Fatalf("expected backend B after reload, got %q", body)
	}
}

// TestServerReload_WaitsForActiveRequests verifies that the old runtime is
// not retired until its in-flight requests complete.
func TestServerReload_WaitsForActiveRequests(t *testing.T) {
	block := make(chan struct{})
	requestStarted := make(chan struct{})

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)

		<-block

		_, _ = w.Write([]byte("done"))
	}))
	defer backend.Close()

	rt1 := buildRuntime(t, 1, []string{backend.URL})
	rt2 := buildRuntime(t, 2, []string{backend.URL})

	server := NewServer(rt1, testLogger)
	markServerRunningForTest(server)

	requestDone := make(chan struct{})

	go func() {
		defer close(requestDone)

		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		w := httptest.NewRecorder()

		server.Handler().ServeHTTP(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Result().StatusCode)
		}
	}()

	// Ensure the request has reached the backend and is therefore
	// associated with Runtime 1 before reload starts
	select {
	case <-requestStarted:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for request to reach backend")
	}

	reloadDone := make(chan error, 1)

	go func() {
		reloadDone <- server.Reload(rt2)
	}()

	// Reload should block while Runtime 1 still has an active request.
	select {
	case err := <-reloadDone:
		t.Fatalf("reload returned before active request completed: %v", err)
	case <-time.After(100 * time.Millisecond):
		// Expected
	}

	// Allow the active request to finish
	close(block)

	select {
	case <-requestDone:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("active request did not complete")
	}

	select {
	case err := <-reloadDone:
		if err != nil {
			t.Fatalf("reload failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("reload did not complete after active request finished")
	}
}

// TestServerReload_AfterShutdownBegins verifies that reloads are rejected once
// the server enters the shutting-down state.
func TestServerReload_AfterShutdownBegins(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("backend"))
	}))
	defer backend.Close()

	rt1 := buildRuntime(t, 1, []string{backend.URL})
	rt2 := buildRuntime(t, 2, []string{backend.URL})

	server := NewServer(rt1, testLogger)
	markServerRunningForTest(server)

	// Create a real HTTP server for Shutdown() to operate on
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	addr := listener.Addr().String()

	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})

	httpServer := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(requestStarted)

			<-releaseRequest

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("done"))
		}),
	}

	server.srv = httpServer

	serveDone := make(chan error, 1)

	go func() {
		err := httpServer.Serve(listener)

		if err != nil && err != http.ErrServerClosed {
			serveDone <- err
			return
		}

		serveDone <- nil
	}()

	// Create an active request so http.Server.Shutdown() blocks
	// while waiting for that request to finish
	requestDone := make(chan error, 1)

	go func() {
		resp, err := http.Get("http://" + addr)
		if err != nil {
			requestDone <- err
			return
		}

		defer resp.Body.Close()

		_, err = io.ReadAll(resp.Body)
		requestDone <- err
	}()

	// Ensure Shutdown() will have an active request to wait for
	select {
	case <-requestStarted:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for request to start")
	}

	// Begin shutdown
	shutdownDone := make(chan error, 1)

	go func() {
		shutdownDone <- server.Shutdown(5 * time.Second)
	}()

	// Wait until Shutdown() has entered its lifecycle transition
	deadline := time.Now().Add(2 * time.Second)

	for serverState(server.state.Load()) != serverShuttingDown {
		if time.Now().After(deadline) {
			t.Fatal("server did not enter shutting-down state")
		}

		time.Sleep(time.Millisecond)
	}

	// Shutdown has begun. Reload must now be rejected
	if err := server.Reload(rt2); !errors.Is(err, ErrServerNotRunning) {
		t.Fatalf(
			"expected ErrServerNotRunning after shutdown began, got %v",
			err,
		)
	}

	// The old runtime must still be installed
	if got := server.runtime.Load().Generation; got != rt1.Generation {
		t.Fatalf(
			"runtime changed during shutdown: expected generation %d, got %d",
			rt1.Generation,
			got,
		)
	}

	// Allow the active request to finish so Shutdown() can complete
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
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("shutdown failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not complete")
	}

	if got := serverState(server.state.Load()); got != serverStopped {
		t.Fatalf(
			"expected serverStopped after shutdown, got %v",
			got,
		)
	}

	// Ensure the Serve goroutine has exited
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("HTTP server failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP server did not stop")
	}
}

// TestServerShutdown_AfterReloadStopsNewRuntime verifies that shutdown retires
// the runtime that is active after a successful reload.
func TestServerShutdown_AfterReloadStopsNewRuntime(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("backend"))
	}))
	defer backend.Close()

	rt1 := buildRuntime(t, 1, []string{backend.URL})
	rt2 := buildRuntime(t, 2, []string{backend.URL})

	server := NewServer(rt1, testLogger)
	markServerRunningForTest(server)

	// Reload must succeed while the server is running
	if err := server.Reload(rt2); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if got := server.runtime.Load().Generation; got != rt2.Generation {
		t.Fatalf(
			"expected runtime generation %d after reload, got %d",
			rt2.Generation,
			got,
		)
	}

	// Shutdown after the reload must retire the newly active runtime
	if err := server.Shutdown(5 * time.Second); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	if got := serverState(server.state.Load()); got != serverStopped {
		t.Fatalf(
			"expected serverStopped after shutdown, got %v",
			got,
		)
	}
}

// TestServerReload_AfterStopped verifies that stopped servers reject reloads
func TestServerReload_AfterStopped(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("backend"))
	}))
	defer backend.Close()

	rt1 := buildRuntime(t, 1, []string{backend.URL})
	rt2 := buildRuntime(t, 2, []string{backend.URL})

	server := NewServer(rt1, testLogger)
	markServerRunningForTest(server)

	// Shut the server down completely
	if err := server.Shutdown(0); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	if got := serverState(server.state.Load()); got != serverStopped {
		t.Fatalf(
			"expected serverStopped, got %v",
			got,
		)
	}

	// Reload must be rejected once the server has stopped
	err := server.Reload(rt2)

	if !errors.Is(err, ErrServerNotRunning) {
		t.Fatalf(
			"expected ErrServerNotRunning after server stopped, got %v",
			err,
		)
	}

	// The stopped server must continue pointing at the original runtime
	if got := server.runtime.Load().Generation; got != rt1.Generation {
		t.Fatalf(
			"runtime changed after server stopped: expected generation %d, got %d",
			rt1.Generation,
			got,
		)
	}
}

// TestServerShutdown_ConcurrentCallsWaitForSameShutdown verifies that concurrent
// shutdown callers wait for the same shutdown operation.
func TestServerShutdown_ConcurrentCallsWaitForSameShutdown(t *testing.T) {
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

	baseCtx, forceCancel := context.WithCancel(context.Background())
	defer forceCancel()

	mux := http.NewServeMux()
	mux.Handle("/", server.Handler())

	server.baseCtx = baseCtx
	server.forceCancel = forceCancel

	server.srv = &http.Server{
		Addr:    addr,
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			return baseCtx
		},
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	server.ready.Store(true)

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

	select {
	case <-requestStarted:
		// Expected.
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for request to start")
	}

	// First shutdown starts and blocks on the active request
	firstShutdownDone := make(chan error, 1)

	go func() {
		firstShutdownDone <- server.Shutdown(5 * time.Second)
	}()

	// Wait until the first shutdown has entered SHUTTING_DOWN
	deadline := time.Now().Add(2 * time.Second)

	for serverState(server.state.Load()) != serverShuttingDown {
		if time.Now().After(deadline) {
			t.Fatal("server did not enter shutting-down state")
		}

		time.Sleep(time.Millisecond)
	}

	// Second shutdown starts while the first one is still in progress
	secondShutdownDone := make(chan error, 1)

	go func() {
		secondShutdownDone <- server.Shutdown(5 * time.Second)
	}()

	// The second shutdown must not report completion while the first
	// shutdown is still waiting for the active request
	select {
	case err := <-secondShutdownDone:
		t.Fatalf(
			"second shutdown returned before first shutdown completed: %v",
			err,
		)
	case <-time.After(100 * time.Millisecond):
		// Expected
	}

	// Let the active request finish
	close(releaseRequest)

	select {
	case err := <-requestDone:
		if err != nil {
			t.Fatalf("active request failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("active request did not complete")
	}

	// First shutdown should now complete
	select {
	case err := <-firstShutdownDone:
		if err != nil {
			t.Fatalf("first shutdown failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first shutdown did not complete")
	}

	// Second shutdown should also complete after the first one
	select {
	case err := <-secondShutdownDone:
		if err != nil {
			t.Fatalf("second shutdown failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second shutdown did not complete")
	}

	if got := serverState(server.state.Load()); got != serverStopped {
		t.Fatalf(
			"expected serverStopped, got %v",
			got,
		)
	}
}

// TestServerReload_ConcurrentReloads verifies that concurrent runtime swaps are
// serialized safely and leave a valid runtime active.
func TestServerReload_ConcurrentReloads(t *testing.T) {
	rt1 := runtime.NewRuntime(1, "", nil, nil, false, nil)
	rt2 := runtime.NewRuntime(2, "", nil, nil, false, nil)
	rt3 := runtime.NewRuntime(3, "", nil, nil, false, nil)

	server := NewServer(rt1, testLogger)
	markServerRunningForTest(server)

	start := make(chan struct{})
	reloadDone := make(chan error, 2)

	go func() {
		<-start
		reloadDone <- server.Reload(rt2)
	}()

	go func() {
		<-start
		reloadDone <- server.Reload(rt3)
	}()

	// Release both reloads simultaneously
	close(start)

	var errs []error

	for range 2 {
		select {
		case err := <-reloadDone:
			errs = append(errs, err)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for concurrent reload")
		}
	}

	for _, err := range errs {
		if err != nil {
			t.Fatalf("concurrent reload failed: %v", err)
		}
	}

	finalRuntime := server.runtime.Load()
	if finalRuntime == nil {
		t.Fatal("expected an active runtime after concurrent reload")
	}

	// Runtime 1 must no longer be active
	if finalRuntime.Generation == rt1.Generation {
		t.Fatalf("runtime 1 remained active after concurrent reloads")
	}

	// The final runtime must be one of the two successfully reloaded runtimes
	if finalRuntime.Generation != rt2.Generation &&
		finalRuntime.Generation != rt3.Generation {
		t.Fatalf(
			"unexpected final runtime generation: %d",
			finalRuntime.Generation,
		)
	}
}

// TestServerStart_ListenerFailureStopsServer verifies that a listener failure
// terminates startup and leaves the server stopped.
func TestServerStart_ListenerFailureStopsServer(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("backend"))
	}))
	defer backend.Close()

	rt := buildRuntime(t, 1, []string{backend.URL})
	server := NewServer(rt, testLogger)

	// Occupy the address so the server's listener creation fails
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create blocking listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	err = server.Start(addr)
	if err == nil {
		t.Fatal("expected Start to fail because the address is already in use")
	}

	if got := serverState(server.state.Load()); got != serverStopped {
		t.Fatalf(
			"expected serverStopped after startup failure, got %v",
			got,
		)
	}

	if server.ready.Load() {
		t.Fatal("server must not remain ready after startup failure")
	}
}

// TestServerStart_UnexpectedServeFailureStopsServer verifies that an unexpected
// Serve failure transitions the running server to stopped.
func TestServerStart_UnexpectedServeFailureStopsServer(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("backend"))
	}))
	defer backend.Close()

	rt := buildRuntime(t, 1, []string{backend.URL})
	server := NewServer(rt, testLogger)

	startDone := make(chan error, 1)

	go func() {
		startDone <- server.Start("127.0.0.1:0")
	}()

	// Wait until Start() has successfully created the listener
	select {
	case <-server.started:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server to start")
	}

	if got := serverState(server.state.Load()); got != serverRunning {
		t.Fatalf(
			"expected serverRunning after startup, got %v",
			got,
		)
	}

	// Simulate an unexpected external listener failure
	if server.listener == nil {
		t.Fatal("server listener is nil after startup")
	}

	if err := server.listener.Close(); err != nil {
		t.Fatalf("failed to close listener: %v", err)
	}

	// Start() should return the unexpected Serve() error
	select {
	case err := <-startDone:
		if err == nil {
			t.Fatal("expected Start to return an unexpected Serve error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after listener failure")
	}

	// A server that can no longer serve traffic must not remain RUNNING
	if got := serverState(server.state.Load()); got != serverStopped {
		t.Fatalf(
			"expected serverStopped after unexpected Serve failure, got %v",
			got,
		)
	}

	if server.ready.Load() {
		t.Fatal("server must not remain ready after unexpected Serve failure")
	}
}
