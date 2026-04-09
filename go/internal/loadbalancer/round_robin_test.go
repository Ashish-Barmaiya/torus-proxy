package loadbalancer

import (
	"testing"
	"torus-proxy/internal/upstream"
)

func TestRoundRobin_Distribution(t *testing.T) {
	backends := []*upstream.Backend{
		{URL: "S1"},
		{URL: "S2"},
	}

	rr := NewRoundRobin(backends)

	counts := map[string]int{
		"S1": 0,
		"S2": 0,
	}

	for i := 0; i < 10; i++ {
		b := rr.Next()
		counts[b.URL]++
	}

	if counts["S1"] == 0 || counts["S2"] == 0 {
		t.Fatalf("expected both backends to be used, got %v", counts)
	}
}

func TestRoundRobin_Order(t *testing.T) {
	backends := []*upstream.Backend{
		{URL: "S1"},
		{URL: "S2"},
	}

	rr := NewRoundRobin(backends)

	expected := []string{"S2", "S1", "S2", "S1"}

	for i, exp := range expected {
		b := rr.Next()
		if b.URL != exp {
			t.Fatalf("step %d: expected %s, got %s", i, exp, b.URL)
		}
	}
}