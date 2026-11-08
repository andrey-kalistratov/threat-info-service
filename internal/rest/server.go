// Package rest собирает общий HTTP-сервер приложения.
package rest

import (
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/handlers"

	"github.com/andrey-kalistratov/threat-info-service/internal/config"
	"github.com/andrey-kalistratov/threat-info-service/internal/logging"
	"github.com/andrey-kalistratov/threat-info-service/internal/rest/generated"
	threatrest "github.com/andrey-kalistratov/threat-info-service/internal/threat/rest"
)

// NewServer собирает HTTP-сервер. Запуском и завершением управляет вызывающий код.
func NewServer(cfg *config.HTTP, logger logging.Logger, threatHandler *threatrest.Handler) (*http.Server, error) {
	errorHandler := &transportErrorHandler{
		logger: logger,
	}

	server, err := generated.NewServer(
		threatHandler,
		generated.WithErrorHandler(errorHandler.Handle),
	)
	if err != nil {
		return nil, err
	}

	logRequest := func(_ io.Writer, p handlers.LogFormatterParams) {
		route, _ := server.FindPath(p.Request.Method, p.Request.URL)
		logger.InfoContext(p.Request.Context(), "HTTP request",
			"method", p.Request.Method,
			"route", route.PathPattern(),
			"status", p.StatusCode,
			"duration", time.Since(p.TimeStamp),
		)
	}

	limitedServer := http.MaxBytesHandler(server, cfg.MaxBodyBytes)
	loggedServer := handlers.CustomLoggingHandler(io.Discard, limitedServer, logRequest)

	return &http.Server{
		Addr:           net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Handler:        loggedServer,
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		IdleTimeout:    cfg.IdleTimeout,
		MaxHeaderBytes: cfg.MaxHeaderBytes,
	}, nil
}
