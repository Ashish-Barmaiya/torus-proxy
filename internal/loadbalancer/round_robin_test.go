package loadbalancer

import (
	"testing"
	"torus-proxy/internal/upstream"
)

func TestRoundRobin_Distribution(t *testing.T) {
	b1, err := upstream.NewBackend("http://localhost:3001")
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	b2, err := upstream.NewBackend("http://localhost:3002")
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	backends := []*upstream.Backend{b1, b2}
	rr := NewRoundRobin(backends)

	counts := map[string]int{
		"http://localhost:3001": 0,
		"http://localhost:3002": 0,
	}

	for i := 0; i < 10; i++ {
		b := rr.Next()
		counts[b.URL]++
	}

	if counts["http://localhost:3001"] == 0 || counts["http://localhost:3002"] == 0 {
		t.Fatalf("expected both backends to be used, got %v", counts)
	}
}

func TestRoundRobin_Order(t *testing.T) {
	b1, _ := upstream.NewBackend("http://localhost:3001")
	b2, _ := upstream.NewBackend("http://localhost:3002")

	backends := []*upstream.Backend{b1, b2}
	rr := NewRoundRobin(backends)

	expected := []string{"http://localhost:3002", "http://localhost:3001", "http://localhost:3002", "http://localhost:3001"}

	for i, exp := range expected {
		b := rr.Next()
		if b.URL != exp {
			t.Fatalf("step %d: expected %s, got %s", i, exp, b.URL)
		}
	}
}

func TestRoundRobin_SkipsUnhealthy(t *testing.T) {
	b1, _ := upstream.NewBackend("http://localhost:3001")
	b2, _ := upstream.NewBackend("http://localhost:3002")
	b2.SetHealthy(false)

	backends := []*upstream.Backend{b1, b2}
	rr := NewRoundRobin(backends)

	for i := 0; i < 100; i++ {
		if b := rr.Next(); b.URL != b1.URL {
			t.Fatalf("expected only healthy backend, got %s", b.URL)
		}
	}
}
