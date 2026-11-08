// Package main запускает HTTP-сервис сведений об угрозах.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrey-kalistratov/threat-info-service/internal/app"
	"github.com/andrey-kalistratov/threat-info-service/internal/config"
	loggingSlog "github.com/andrey-kalistratov/threat-info-service/internal/logging/slog"
)

const serviceName = "threat-service"

func main() {
	logger := loggingSlog.NewTemporary()
	if err := run(logger); err != nil {
		logger.Error("application stopped", "service", serviceName, "error", err)
		os.Exit(1)
	}
}

func run(bootstrapLogger *slog.Logger) error {
	ctx := context.Background()
	configPath := flag.String("config", "", "path to YAML configuration")
	flag.Parse()

	bootstrapLogger.InfoContext(ctx, "reading config", "path", *configPath)
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	logger, err := loggingSlog.New(&cfg.Observability.Log)
	if err != nil {
		return err
	}
	logger = logger.With("service", serviceName)
	slog.SetDefault(logger)

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	service := app.New(cfg, logger)

	if err := service.Run(sigCtx); err != nil {
		return err
	}

	var serveErr error
	select {
	case <-sigCtx.Done():
		logger.InfoContext(ctx, "shutdown signal received")
	case serveErr = <-service.Done():
	}

	return errors.Join(serveErr, service.Shutdown(ctx))
}
