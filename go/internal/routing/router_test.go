package routing

import (
	"testing"
	"torus-proxy/internal/service"
	"torus-proxy/internal/upstream"
)

func TestRouter_BasicRouting(t *testing.T) {
	router := NewRouter()

	backends := []*upstream.Backend{
		{URL: "http://localhost:3001"},
	}

	svc := service.NewService(backends)

	router.AddRoute("/api", svc)

	result := router.Route("/api")

	if result == nil {
		t.Fatal("expected service, got nil")
	}
}

func TestRouter_NotFound(t *testing.T) {
	router := NewRouter()

	result := router.Route("/unknown")

	if result != nil {
		t.Fatal("expected nil for unknown route")
	}
}