package health

import (
	"context"
	"log"
	"time"
)

type Checker interface {
	Check(ctx context.Context) error
}

// Prober runs a checker periodically
func StartProber(
	ctx context.Context,
	checker Checker,
	interval time.Duration,
	timeout time.Duration,
	onHealthy func(),
	onUnhealthy func(),
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
				StartProber(ctx, checker, interval, timeout, onHealthy, onUnhealthy) // restrat the prober
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
				} else {
					onUnhealthy()
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
