package health_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"torus-proxy/internal/health"
)

func TestHTTPChecker_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	checker := &health.HTTPChecker{
		URL:    srv.URL,
		Client: srv.Client(),
		Path:   "/health",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := checker.Check(ctx)
	if err != nil {
		t.Fatalf("expected healthy, got error: %v", err)
	}
}

func TestHTTPChecker_Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	checker := &health.HTTPChecker{
		URL:    srv.URL,
		Client: srv.Client(),
		Path:   "/health",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := checker.Check(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

type MockChecker struct {
	errToReturn     error
	shouldPanic     bool
	shouldSecondary bool
	checkCount      atomic.Int32
}

func (m *MockChecker) Check(ctx context.Context) error {
	m.checkCount.Add(1)

	if m.shouldPanic {
		panic("primary network driver failure")
	}
	return m.errToReturn
}

func (m *MockChecker) String() string {
	if m.shouldSecondary {
		panic("secondary nested fault during string formatting")
	}
	return "MockChecker"
}

// The Lifecycle Test (Success -> Failure -> Auto-Recovery)
func TestStartProber_Lifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock := &MockChecker{}

	var healthyCount atomic.Int32
	var unhealthyCount atomic.Int32

	health.StartProber(
		ctx,
		mock,
		10*time.Millisecond,
		2*time.Millisecond,
		func() { healthyCount.Add(1) },
		func() { unhealthyCount.Add(1) },
	)

	// Verify Success
	time.Sleep(45 * time.Millisecond)

	if healthyCount.Load() == 0 {
		t.Fatal("Expected onHealthy callbacks to fire, but got 0")
	}

	// Verify Failure
	mock.errToReturn = errors.New("backend timeout connection refused")
	currentHealthySnapshot := healthyCount.Load()

	time.Sleep(35 * time.Millisecond)

	if unhealthyCount.Load() == 0 {
		t.Fatal("Expected onUnhealthy callbacks to fire after backend failure, but got 0")
	}
	if healthyCount.Load() != currentHealthySnapshot {
		t.Fatal("onHealthy callback shouldn't keep incrementing while the backend is broken")
	}

	// Verify Auto-Recovery
	mock.errToReturn = nil
	currentUnhealthySnapshot := unhealthyCount.Load()

	time.Sleep(35 * time.Millisecond)

	if healthyCount.Load() == currentHealthySnapshot {
		t.Fatal("Expected prober to auto-recover and resume firing onHealthy, but count stalled")
	}
	if unhealthyCount.Load() != currentUnhealthySnapshot {
		t.Fatal("onUnhealthy should stop firing once the backend recovers")
	}
}

// Self-Healing Test (Primary Panic -> Recover -> Resurrect)
func TestStartProber_PrimaryPanicRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock := &MockChecker{shouldPanic: true}

	var unhealthyCount atomic.Int32

	health.StartProber(
		ctx,
		mock,
		10*time.Millisecond,
		2*time.Millisecond,
		func() {},
		func() { unhealthyCount.Add(1) },
	)

	time.Sleep(50 * time.Millisecond)

	// Panic switch off
	mock.shouldPanic = false

	time.Sleep(2200 * time.Millisecond)

	// Verify resurrection
	if mock.checkCount.Load() <= 1 {
		t.Fatalf("Self-healing failed. Expected checkCount to grow past 1 after resurrection, got %d", mock.checkCount.Load())
	}
}

// Secondary Fault Isolation Test (Primary Panic -> Secondary Panic on Recovery -> Goroutine Halted)
func TestStartProber_SecondaryFaultIsolation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// This triggers a primary panic on Check() and secondary panic on String()
	mock := &MockChecker{
		shouldPanic:     true,
		shouldSecondary: true,
	}

	health.StartProber(
		ctx,
		mock,
		10*time.Millisecond,
		2*time.Millisecond,
		func() {},
		func() {},
	)

	time.Sleep(50 * time.Millisecond)
	t.Log("System survived secondary recovery fault sequence successfully.")
}
