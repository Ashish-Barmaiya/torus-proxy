package main

import (
	"context"
	"log"
	"net/http"
	"time"
	"torus-proxy/internal/health"
	"torus-proxy/internal/proxy"
	"torus-proxy/internal/routing"
	"torus-proxy/internal/service"
	"torus-proxy/internal/upstream"
)

func main() {
	// Ceate backends
	b1, err := upstream.NewBackend("http://localhost:3001")
	if err != nil {
		log.Fatalf("failed to create backend 1: %v", err)
	}

	b2, err := upstream.NewBackend("http://localhost:3002")
	if err != nil {
		log.Fatalf("failed to create backend 2: %v", err)
	}

	backends := []*upstream.Backend{b1, b2}

	// Health check
	healthClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background()) // root context for entire proxy
	defer cancel()

	for _, b := range backends {
		checker := &health.HTTPChecker{
			URL:    b.URL,
			Client: healthClient,
			Path:   "/health",
		}

		backend := b
		health.StartProber(
			ctx,
			checker,
			5*time.Second, // interval between every check
			2*time.Second, // per-check timeout
			func() { backend.SetHealthy(true) },
			func() { backend.SetHealthy(false) },
		)
	}

	// Create service
	svc := service.NewService(backends)

	// Create router
	router := routing.NewRouter()
	router.AddRoute("/api", svc)

	// Start proxy
	server := proxy.NewServer(router)

	log.Println("Proxy is running on :8080")
	log.Fatal(server.Start(":8080"))
}
