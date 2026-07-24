package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	"torus-proxy/internal/config"
	"torus-proxy/internal/proxy"
	"torus-proxy/internal/runtime"
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

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	rt, err := runtime.BuildRuntime(cfg, logger)
	if err != nil {
		logger.Error("failed to build runtime", "error", err)
		os.Exit(1)
	}

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start proxy
	server := proxy.NewServer(rt.Router, logger, rt.TLSConfig)

	go func() {
		logger.Info("Torus is running", "addr", cfg.Server.Addr)
		if err := server.Start(cfg.Server.Addr); err != nil {
			logger.Error("server stopped", "error", err)
			cancel()
			os.Exit(1)
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
