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

	for _, rConfig := range cfg.Routes {
		var backends []*upstream.Backend

		for _, upURL := range rConfig.Upstreams {
			b, err := upstream.NewBackend(upURL, cfg.Observability.EnabledValue())
			if err != nil {
				cancel()
				return nil, fmt.Errorf("create backend %q: %w", upURL, err)
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
				rt,
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

	observability.ObserveRuntimeBuildDuration(
		time.Since(start),
	)

	return rt, nil
}
