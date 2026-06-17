package health

import (
	"context"
	"log"
	"log/slog"
	"time"
)

type Checker interface {
	Check(ctx context.Context) error
	Target() string
}

// Prober runs a checker periodically
func StartProber(
	ctx context.Context,
	checker Checker,
	interval time.Duration,
	timeout time.Duration,
	onHealthy func(),
	onUnhealthy func(),
	logger *slog.Logger,
) {
	go func() {
		// Primary recovery
		defer func() {
			if r := recover(); r != nil {

				// Secondary recovery -> this catches a crash happening during the recovery process itself
				defer func() {
					if secondaryErr := recover(); secondaryErr != nil {
						log.Printf("[CRITICAL SYSTEM FAULT] Health check supervisor crashed completely: %v. Halted health check.", secondaryErr)
					}
				}()

				log.Printf("[HEALTH CRASH ALERT] Worker panicked: %v. Restarting worker...", r)
				time.Sleep(2 * time.Second)
				StartProber(ctx, checker, interval, timeout, onHealthy, onUnhealthy, logger) // restrat the prober
			}
		}()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				probeCtx, cancel := context.WithTimeout(ctx, timeout)
				err := checker.Check(probeCtx)
				cancel()

				if err == nil {
					onHealthy()
					logger.Debug("health check passed", "checker", checker)
				} else {
					onUnhealthy()
					logger.Warn("health check failed", "checker", checker.Target(), "error", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
