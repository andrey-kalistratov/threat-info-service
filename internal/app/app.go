// Package app собирает и запускает приложение.
package app

import (
	"context"
	"errors"
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
	cfg             *config.Config
	server          *http.Server
	pool            *pgxpool.Pool
	logger          logging.Logger
	shutdownTimeout time.Duration
	serveResult     chan error
}

// New создает приложение с загруженной конфигурацией и логгером.
func New(cfg *config.Config, logger logging.Logger) *App {
	return &App{
		cfg:             cfg,
		logger:          logger,
		shutdownTimeout: cfg.App.ShutdownTimeout,
		serveResult:     make(chan error, 1),
	}
}

func newHTTPServer(cfg *config.HTTP, logger logging.Logger, handler *threatrest.Handler) (*http.Server, error) {
	return rest.NewServer(cfg, logger, handler)
}

// Run открывает HTTP-слушатель и запускает сервер в фоне.
func (a *App) Run(ctx context.Context) error {
	poolConfig, err := pgxpool.ParseConfig(postgresURL(a.cfg.DB.Postgres))
	if err != nil {
		return fmt.Errorf("parse database configuration: %w", err)
	}

	connectCtx, cancelConnect := context.WithTimeout(ctx, databaseConnectTimeout)
	defer cancelConnect()
	a.pool, err = pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	if pingErr := a.pool.Ping(connectCtx); pingErr != nil {
		a.pool.Close()
		return fmt.Errorf("connect to database: %w", pingErr)
	}

	repository := threatpostgres.NewRepository(a.pool)
	service := threat.NewService(repository)
	handler := threatrest.NewHandler(a.logger, service)
	//nolint:contextcheck // Контекст запроса используется обработчиком после сборки сервера.
	a.server, err = newHTTPServer(&a.cfg.HTTP, a.logger, handler)
	if err != nil {
		a.pool.Close()
		return fmt.Errorf("create HTTP server: %w", err)
	}

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", a.server.Addr)
	if err != nil {
		a.pool.Close()
		return fmt.Errorf("listen HTTP: %w", err)
	}

	a.logger.InfoContext(ctx, "starting HTTP server", "address", a.server.Addr)
	go func() {
		a.serveResult <- a.server.Serve(listener)
	}()
	return nil
}

// Done сообщает об остановке HTTP-сервера или его неожиданной ошибке.
func (a *App) Done() <-chan error {
	return a.serveResult
}

// Shutdown останавливает HTTP-сервер, затем закрывает пул PostgreSQL.
func (a *App) Shutdown(ctx context.Context) error {
	shutdownCtx, cancelShutdown := context.WithTimeout(ctx, a.shutdownTimeout)
	defer cancelShutdown()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		closeErr := a.server.Close()
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
			return errors.Join(fmt.Errorf("shutdown timed out: %w", err), closeErr)
		}
		return errors.Join(fmt.Errorf("shutdown HTTP server: %w", err), closeErr)
	}

	a.pool.Close()
	a.logger.InfoContext(ctx, "HTTP server stopped")
	return nil
}

func postgresURL(cfg config.Postgres) string {
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
