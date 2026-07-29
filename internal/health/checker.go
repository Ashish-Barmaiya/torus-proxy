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

type WorkerGroup interface {
	AddWorker()
	DoneWorker()
}

// Prober runs a checker periodically
func StartProber(
	ctx context.Context,
	workers WorkerGroup,
	checker Checker,
	interval time.Duration,
	timeout time.Duration,
	onHealthy func(),
	onUnhealthy func(),
	logger *slog.Logger,
) {
	// Register this goroutine
	workers.AddWorker()

	go func() {
		// Unregister this goroutine
		defer workers.DoneWorker()

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
				select {
				case <-time.After(2 * time.Second):
					// Don't restart if this runtime has already been shut down.
					if ctx.Err() == nil {
						StartProber(ctx, workers, checker, interval, timeout, onHealthy, onUnhealthy, logger) // restrat the prober
					}
				case <-ctx.Done():
					// Runtime is shutting down; do not restart.
					return
				}

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
