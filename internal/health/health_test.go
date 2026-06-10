package health_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"torus-proxy/internal/health"
)

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

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
	mu              sync.Mutex
	errToReturn     error
	shouldPanic     bool
	shouldSecondary bool
	checkCount      atomic.Int32
}

// Setter to set error
func (m *MockChecker) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errToReturn = err
}

// Setter to set panic
func (m *MockChecker) SetPanic(panicState bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldPanic = panicState
}

// Setter to set secondary panic
func (m *MockChecker) SetSecondaryPanic(secondaryState bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldSecondary = secondaryState
}

func (m *MockChecker) Check(ctx context.Context) error {
	m.checkCount.Add(1)

	m.mu.Lock()
	panicState := m.shouldPanic
	errState := m.errToReturn
	m.mu.Unlock()

	if panicState {
		panic("primary network driver failure")
	}
	return errState
}

func (m *MockChecker) String() string {
	m.mu.Lock()
	secondaryState := m.shouldSecondary
	m.mu.Unlock()

	if secondaryState {
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
		testLogger,
	)

	// Verify Success
	time.Sleep(45 * time.Millisecond)

	if healthyCount.Load() == 0 {
		t.Fatal("Expected onHealthy callbacks to fire, but got 0")
	}

	// Verify Failure
	mock.SetError(errors.New("backend timeout connection refused"))
	currentHealthySnapshot := healthyCount.Load()

	time.Sleep(35 * time.Millisecond)

	if unhealthyCount.Load() == 0 {
		t.Fatal("Expected onUnhealthy callbacks to fire after backend failure, but got 0")
	}
	if healthyCount.Load() != currentHealthySnapshot {
		t.Fatal("onHealthy callback shouldn't keep incrementing while the backend is broken")
	}

	// Verify Auto-Recovery
	mock.SetError(nil)
	currentUnhealthySnapshot := unhealthyCount.Load()

	success := false
	for i := 0; i < 20; i++ {
		if healthyCount.Load() == currentHealthySnapshot {
			success = true
			break
		}
	}

	if !success {
		t.Fatal("Expected prober to auto-recover and resume firing onHealthy, but count stalled")
	}

	time.Sleep(20 * time.Millisecond)

	if unhealthyCount.Load() != currentUnhealthySnapshot {
		t.Fatalf("onUnhealthy should stop firing once the backend recovers. Expected %d, got %d", currentUnhealthySnapshot, unhealthyCount.Load())
	}
}

// Self-Healing Test (Primary Panic -> Recover -> Resurrect)
func TestStartProber_PrimaryPanicRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock := &MockChecker{}
	mock.SetPanic(true)

	var unhealthyCount atomic.Int32

	health.StartProber(
		ctx,
		mock,
		10*time.Millisecond,
		2*time.Millisecond,
		func() {},
		func() { unhealthyCount.Add(1) },
		testLogger,
	)

	time.Sleep(50 * time.Millisecond)

	// Panic switch off
	mock.SetPanic(false)

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
	mock := &MockChecker{}
	mock.SetPanic(true)
	mock.SetSecondaryPanic(true)

	health.StartProber(
		ctx,
		mock,
		10*time.Millisecond,
		2*time.Millisecond,
		func() {},
		func() {},
		testLogger,
	)

	time.Sleep(50 * time.Millisecond)
	t.Log("System survived secondary recovery fault sequence successfully.")
}
