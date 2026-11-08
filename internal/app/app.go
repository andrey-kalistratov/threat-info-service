// Package app собирает и запускает приложение.
package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/andrey-kalistratov/threat-info-service/internal/config"
	"github.com/andrey-kalistratov/threat-info-service/internal/logging"
	"github.com/andrey-kalistratov/threat-info-service/internal/rest"
	"github.com/andrey-kalistratov/threat-info-service/internal/threat"
	threatpostgres "github.com/andrey-kalistratov/threat-info-service/internal/threat/postgres"
	threatrest "github.com/andrey-kalistratov/threat-info-service/internal/threat/rest"
)

const databaseConnectTimeout = 5 * time.Second

// App управляет HTTP-сервером и пулом PostgreSQL.
type App struct {
	cfg    *config.Config
	server *http.Server
	pool   *pgxpool.Pool
	logger logging.Logger
}

// New создает приложение с загруженной конфигурацией и логгером.
func New(cfg *config.Config, logger logging.Logger) *App {
	return &App{
		cfg:    cfg,
		logger: logger,
	}
}

// Run подключается к PostgreSQL и запускает HTTP-сервер в фоне.
func (a *App) Run(ctx context.Context) (done <-chan error, err error) {
	poolConfig, err := pgxpool.ParseConfig(postgresURL(&a.cfg.DB.Postgres))
	if err != nil {
		return nil, fmt.Errorf("parse database configuration: %w", err)
	}

	connectCtx, cancelConnect := context.WithTimeout(ctx, databaseConnectTimeout)
	defer cancelConnect()
	a.pool, err = pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	// При неудачном запуске закрываем пул здесь; после запуска это делает Shutdown.
	defer func() {
		if err != nil {
			a.pool.Close()
		}
	}()

	if pingErr := a.pool.Ping(connectCtx); pingErr != nil {
		return nil, fmt.Errorf("connect to database: %w", pingErr)
	}

	repository := threatpostgres.NewRepository(a.pool)
	service := threat.NewService(repository)
	handler := threatrest.NewHandler(a.logger, service)
	//nolint:contextcheck // Контекст запроса используется обработчиком после сборки сервера.
	a.server, err = rest.NewServer(&a.cfg.HTTP, a.logger, handler)
	if err != nil {
		return nil, fmt.Errorf("create HTTP server: %w", err)
	}

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", a.server.Addr)
	if err != nil {
		return nil, fmt.Errorf("listen HTTP: %w", err)
	}

	a.logger.InfoContext(ctx, "starting HTTP server", "address", a.server.Addr)
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- a.server.Serve(listener)
	}()
	return serveResult, nil
}

// Shutdown останавливает HTTP-сервер и закрывает пул PostgreSQL.
func (a *App) Shutdown(ctx context.Context) error {
	defer a.pool.Close()

	shutdownCtx, cancelShutdown := context.WithTimeout(ctx, a.cfg.App.ShutdownTimeout)
	defer cancelShutdown()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		if closeErr := a.server.Close(); closeErr != nil {
			a.logger.ErrorContext(ctx, "force close HTTP server", "error", closeErr)
		}
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	a.logger.InfoContext(ctx, "HTTP server stopped")
	return nil
}

func postgresURL(cfg *config.Postgres) string {
	connectionURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port)),
		Path:   cfg.Database,
	}
	query := connectionURL.Query()
	query.Set("sslmode", cfg.SSLMode)
	connectionURL.RawQuery = query.Encode()
	return connectionURL.String()
}
