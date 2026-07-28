package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	"torus-proxy/internal/configwatcher"
	"torus-proxy/internal/observability"
	"torus-proxy/internal/proxy"
	"torus-proxy/internal/reload"
)

func main() {
	// Structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	configPath := flag.String(
		"config",
		"torus.yaml",
		"Path to configuration file",
	)
	flag.Parse()

	// Build runtime manager
	manager := reload.NewManager(
		*configPath,
		logger,
	)

	// Build initial runtime
	rt, err := manager.BuildInitialRuntime()
	if err != nil {
		logger.Error("failed to build initial runtime", "error", err)
		os.Exit(1)
	}

	observability.SetRuntimeGeneration(rt.Generation)

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start proxy
	server := proxy.NewServer(rt, logger)
	manager.SetServer(server)

	go func() {
		logger.Info("Torus is running", "addr", rt.Addr)
		if err := server.Start(rt.Addr); err != nil {
			logger.Error("server stopped", "error", err)
			cancel()
			os.Exit(1)
		}
	}()

	// Create and start config watcher
	watcher := configwatcher.New(
		*configPath,
		logger,
		manager,
	)

	go func() {
		if err := watcher.Start(rootCtx); err != nil {
			logger.Error(
				"configuration watcher stopped",
				"error", err,
			)
		}
	}()

	// This ctx is cancelled on SIGINT or SIGTERM
	signalCtx, stop := signal.NotifyContext(rootCtx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-signalCtx.Done() // Wait for shutdown signal
	logger.Info("received shutdown signal", "signal", signalCtx.Err())

	// Cancel root ctx
	cancel()

	// Shutdown server with timeout
	shutdownTimeout := 25 * time.Second
	if err := server.Shutdown(shutdownTimeout); err != nil {
		logger.Error("failed to shutdown server gracefully", "error", err)
		os.Exit(1)
	}

	logger.Info("Server shutdown complete")
	os.Exit(0)
}
