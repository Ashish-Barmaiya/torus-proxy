package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
	"torus-proxy/internal/config"
	"torus-proxy/internal/health"
	"torus-proxy/internal/observability"
	"torus-proxy/internal/routing"
	"torus-proxy/internal/service"
	"torus-proxy/internal/upstream"
)

func BuildRuntime(cfg *config.Config, logger *slog.Logger) (*Runtime, error) {
	start := time.Now()

	router := routing.NewRouter()

	ctx, cancel := context.WithCancel(context.Background())

	healthClient := &http.Client{}

	tlsCfg, err := cfg.Tls.LoadTlsConfig()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("load TLS config: %w", err)
	}

	logger.Info("TLS config loaded", "enabled", tlsCfg != nil)

	generation := nextGeneration.Add(1)

	rt := NewRuntime(generation, cfg.Server.Addr, router, tlsCfg, cfg.Observability.EnabledValue(), cancel)

	services := make(map[string]*service.Service, len(cfg.Services))

	// Build services and their backend pools.
	for _, serviceConfig := range cfg.Services {
		backends := make([]*upstream.Backend, 0, len(serviceConfig.Upstreams))

		for _, upstreamURL := range serviceConfig.Upstreams {
			backend, err := upstream.NewBackend(
				upstreamURL,
				cfg.Observability.EnabledValue(),
			)
			if err != nil {
				cancel()
				return nil, fmt.Errorf(
					"create backend %q for service %q: %w",
					upstreamURL,
					serviceConfig.Name,
					err,
				)
			}

			backends = append(backends, backend)

			checker := &health.HTTPChecker{
				URL:    backend.URL,
				Client: healthClient,
				Path:   cfg.HealthCheck.Path,
			}

			backendRef := backend

			health.StartProber(
				ctx,
				rt,
				checker,
				cfg.HealthCheck.Interval(),
				cfg.HealthCheck.Timeout(),
				func() {
					backendRef.SetHealthy(true)
				},
				func() {
					backendRef.SetHealthy(false)
				},
				logger,
			)
		}

		services[serviceConfig.Name] = service.NewService(backends)
	}

	for _, routeConfig := range cfg.Routes {
		svc := services[routeConfig.Service]

		router.AddRoute(routeConfig.Path, svc)
	}

	observability.ObserveRuntimeBuildDuration(
		time.Since(start),
	)

	return rt, nil
}
