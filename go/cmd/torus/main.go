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
	backends := []*upstream.Backend {
		{URL: "http://localhost:3001"},
		{URL: "http://localhost:3002"},
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