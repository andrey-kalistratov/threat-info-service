// Package main запускает HTTP-сервис сведений об угрозах.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrey-kalistratov/threat-info-service/internal/app"
	"github.com/andrey-kalistratov/threat-info-service/internal/config"
	"github.com/andrey-kalistratov/threat-info-service/internal/logging"
	"github.com/andrey-kalistratov/threat-info-service/internal/logging/sloglogger"
)

const serviceName = "threat-service"

func main() {
	logger := sloglogger.NewBootstrap()
	if err := run(logger); err != nil {
		logger.Error("application stopped", "service", serviceName, "error", err)
		os.Exit(1)
	}
}

func run(bootstrapLogger logging.Logger) error {
	ctx := context.Background()

	configPath := flag.String("config", "", "path to YAML configuration")
	flag.Parse()

	bootstrapLogger.InfoContext(ctx, "reading config", "path", *configPath)
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	logger, err := sloglogger.New(&cfg.Observability.Log)
	if err != nil {
		return err
	}
	logger = logger.With("service", serviceName)

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	service := app.New(cfg, logger)
	done, err := service.Run(sigCtx)
	if err != nil {
		return err
	}

	var serveErr error
	select {
	case <-sigCtx.Done():
		logger.InfoContext(ctx, "shutdown signal received")
	case serveErr = <-done:
	}

	if err := service.Shutdown(ctx); err != nil {
		return err
	}
	return serveErr
}
