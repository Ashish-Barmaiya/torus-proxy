package health_test

import (
	"context"
	"net/http"
	"net/http/httptest"
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
