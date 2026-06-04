package main

import (
	"log"
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
