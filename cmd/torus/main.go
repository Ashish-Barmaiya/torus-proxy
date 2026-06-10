package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"
	"torus-proxy/internal/health"
	"torus-proxy/internal/proxy"
	"torus-proxy/internal/routing"
	"torus-proxy/internal/service"
	"torus-proxy/internal/upstream"
)

func main() {
	// Structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Ceate backends
	b1, err := upstream.NewBackend("http://localhost:3001")
	if err != nil {
		logger.Error("failed to create backend 1", "error", err)
		os.Exit(1)
	}

	b2, err := upstream.NewBackend("http://localhost:3002")
	if err != nil {
		logger.Error("failed to create backend 2", "error", err)
		os.Exit(1)
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
			logger,
		)
	}

	// Create service
	svc := service.NewService(backends)

	// Create router
	router := routing.NewRouter()
	router.AddRoute("/api", svc)

	// Start proxy
	server := proxy.NewServer(router, logger)

	logger.Info("Torus is running", "addr", "8080")
	if err := server.Start(":8080"); err != nil {
		logger.Error("server stopped", "error", err)
	}
}
