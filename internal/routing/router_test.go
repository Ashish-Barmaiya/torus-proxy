package routing

import (
	"testing"
	"torus-proxy/internal/service"
	"torus-proxy/internal/upstream"
)

func TestRouter_LongestPrefixMatch(t *testing.T) {
	router := NewRouter()

	b1, _ := upstream.NewBackend("http://localhost:3001")
	svcApi := service.NewService([]*upstream.Backend{b1})

	b2, _ := upstream.NewBackend("http://localhost:3002")
	svcApiV1 := service.NewService([]*upstream.Backend{b2})

	router.AddRoute("/api", svcApi)
	router.AddRoute("/api/v1", svcApiV1)

	// Case 1: Exact Match
	route, svc := router.Route("/api")
	if route != "/api" {
		t.Errorf("expected matched route '/api', got %q", route)
	}
	if svc != svcApi {
		t.Error("expected /api to map to svcAPI")
	}

	// Case 2: Deeper Path Segment Match
	route, svc = router.Route("/api/v1/users")
	if route != "/api/v1" {
		t.Errorf("expected matched route '/api/v1', got %q", route)
	}
	if svc != svcApiV1 {
		t.Error("expected /api/v1/users to map to svcAPIV1 (longest prefix match)")
	}

	// Case 3: Edge Case Word-Collision Guardrail Check
	route, svc = router.Route("/api-status")
	if route != "" {
		t.Errorf("expected empty matched route, got %q", route)
	}
	if svc != nil {
		t.Errorf("expected nil due to segment boundary check, got %v", svc)
	}
}

func TestRouter_NotFound(t *testing.T) {
	router := NewRouter()
	route, svc := router.Route("/unknown")

	if route != "" {
		t.Errorf("expected empty matched route, got %q", route)
	}

	if svc != nil {
		t.Fatal("expected nil for unknown route")
	}
}
