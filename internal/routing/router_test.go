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
	if res := router.Route("/api"); res != svcApi {
		t.Error("expected /api to map to svcApi")
	}

	// Case 2: Deeper Path Segment Match
	if res := router.Route("/api/v1/users"); res != svcApiV1 {
		t.Error("expected /api/v1/users to map to svcApiV1 (longest prefix match)")
	}

	// Case 3: Edge Case Word-Collision Guardrail Check
	if res := router.Route("/api-status"); res != nil {
		t.Errorf("expected /api-status to return nil due to segment boundary check, got %v", res)
	}
}

func TestRouter_NotFound(t *testing.T) {
	router := NewRouter()
	if result := router.Route("/unknown"); result != nil {
		t.Fatal("expected nil for unknown route")
	}
}
