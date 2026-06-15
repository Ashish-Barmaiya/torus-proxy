package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"
	"torus-proxy/internal/config"
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

	// Load configuration
	cfg, err := config.LoadConfig("torus.yaml")
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Health check
	healthClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background()) // root context for entire proxy
	defer cancel()

	router := routing.NewRouter()

	for _, rConfig := range cfg.Routes {
		var backends []*upstream.Backend

		for _, upURL := range rConfig.Upstreams {
			b, err := upstream.NewBackend(upURL)
			if err != nil {
				logger.Error("failed to create backend", "url", upURL, "error", err)
				os.Exit(1)
			}
			backends = append(backends, b)

			checker := &health.HTTPChecker{
				URL:    b.URL,
				Client: healthClient,
				Path:   cfg.HealthCheck.Path,
			}

			backend := b
			health.StartProber(
				ctx,
				checker,
				cfg.HealthCheck.Interval(),
				cfg.HealthCheck.Timeout(),
				func() { backend.SetHealthy(true) },
				func() { backend.SetHealthy(false) },
				logger,
			)
		}

		svc := service.NewService(backends)
		router.AddRoute(rConfig.Path, svc)
	}

	// Start proxy
	server := proxy.NewServer(router, logger)

	logger.Info("Torus is running", "addr", "8080")
	if err := server.Start(":8080"); err != nil {
		logger.Error("server stopped", "error", err)
	}
}
