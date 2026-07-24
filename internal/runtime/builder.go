package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"torus-proxy/internal/config"
	"torus-proxy/internal/health"
	"torus-proxy/internal/routing"
	"torus-proxy/internal/service"
	"torus-proxy/internal/upstream"
)

func BuildRuntime(cfg *config.Config, logger *slog.Logger) (*Runtime, error) {
	router := routing.NewRouter()

	ctx, cancel := context.WithCancel(context.Background())

	healthClient := &http.Client{}

	for _, rConfig := range cfg.Routes {
		var backends []*upstream.Backend

		for _, upURL := range rConfig.Upstreams {
			b, err := upstream.NewBackend(upURL)
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

	tlsCfg, err := cfg.Tls.LoadTlsConfig()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("load TLS config: %w", err)
	}

	logger.Info("TLS config loaded", "enabled", tlsCfg != nil)

	rt := NewRuntime(router, tlsCfg)

	_ = cancel

	return rt, nil
}
