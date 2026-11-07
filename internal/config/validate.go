package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
)

var (
	// ErrEmptyValue означает отсутствие обязательного значения или явно пустой env.
	ErrEmptyValue = errors.New("value must not be empty")

	// ErrInvalidPort означает выход порта за допустимый диапазон.
	ErrInvalidPort = errors.New("port must be between 1 and 65535")

	// ErrNonPositiveValue означает неположительный таймаут или лимит.
	ErrNonPositiveValue = errors.New("value must be positive")

	// ErrLogLevel означает неподдерживаемый slog уровень логирования.
	ErrLogLevel = errors.New("observability.log.level is not supported by slog")

	// ErrLogFormat означает неподдерживаемый формат логирования.
	ErrLogFormat = errors.New("observability.log.format must be json or text")

	// ErrInspectEnv означает невозможность получить метаданные env-полей конфигурации.
	ErrInspectEnv = errors.New("cannot inspect environment fields")
)

func (c *Config) validate() error {
	if err := checkEmptyEnv(c); err != nil {
		return err
	}

	if err := c.validateRequired(); err != nil {
		return err
	}

	if err := c.validatePorts(); err != nil {
		return err
	}

	if err := c.validateTimeouts(); err != nil {
		return err
	}

	if err := c.validateLimits(); err != nil {
		return err
	}

	return c.validateLog()
}

func (c *Config) validateRequired() error {
	required := []struct {
		name  string
		value string
	}{
		{
			name:  "http.host",
			value: c.HTTP.Host,
		},
		{
			name:  "db.postgres.host",
			value: c.DB.Postgres.Host,
		},
		{
			name:  "db.postgres.database",
			value: c.DB.Postgres.Database,
		},
		{
			name:  "db.postgres.user",
			value: c.DB.Postgres.User,
		},
		{
			name:  "db.postgres.password",
			value: c.DB.Postgres.Password,
		},
		{
			name:  "db.postgres.sslmode",
			value: c.DB.Postgres.SSLMode,
		},
		{
			name:  "observability.log.level",
			value: c.Observability.Log.Level,
		},
		{
			name:  "observability.log.format",
			value: c.Observability.Log.Format,
		},
	}

	for _, field := range required {
		if field.value == "" {
			return fmt.Errorf("%s: %w", field.name, ErrEmptyValue)
		}
	}

	return nil
}

func (c *Config) validatePorts() error {
	ports := []struct {
		name  string
		value int
	}{
		{
			name:  "http.port",
			value: c.HTTP.Port,
		},
		{
			name:  "db.postgres.port",
			value: c.DB.Postgres.Port,
		},
	}

	for _, field := range ports {
		if field.value < 1 || field.value > 65535 {
			return fmt.Errorf("%s: %w", field.name, ErrInvalidPort)
		}
	}

	return nil
}

func (c *Config) validateTimeouts() error {
	timeouts := []struct {
		name  string
		value time.Duration
	}{
		{
			name:  "app.shutdown_timeout",
			value: c.App.ShutdownTimeout,
		},
		{
			name:  "http.read_timeout",
			value: c.HTTP.ReadTimeout,
		},
		{
			name:  "http.write_timeout",
			value: c.HTTP.WriteTimeout,
		},
		{
			name:  "http.idle_timeout",
			value: c.HTTP.IdleTimeout,
		},
	}

	for _, field := range timeouts {
		if field.value <= 0 {
			return fmt.Errorf("%s: %w", field.name, ErrNonPositiveValue)
		}
	}

	return nil
}

func (c *Config) validateLimits() error {
	if c.HTTP.MaxHeaderBytes <= 0 {
		return fmt.Errorf("http.max_header_bytes: %w", ErrNonPositiveValue)
	}

	if c.HTTP.MaxBodyBytes <= 0 {
		return fmt.Errorf("http.max_body_bytes: %w", ErrNonPositiveValue)
	}

	return nil
}

func (c *Config) validateLog() error {
	var level slog.Level
	if err := level.UnmarshalText([]byte(c.Observability.Log.Level)); err != nil {
		return ErrLogLevel
	}

	if c.Observability.Log.Format != "json" && c.Observability.Log.Format != "text" {
		return ErrLogFormat
	}

	return nil
}

func checkEmptyEnv(cfg *Config) error {
	fields, err := env.GetFieldParams(cfg)
	if err != nil {
		// Причину можно показать: GetFieldParams проверяет теги, а не значения env.
		return fmt.Errorf("%w: %s", ErrInspectEnv, err.Error())
	}

	for _, field := range fields {
		if value, ok := os.LookupEnv(field.Key); ok && value == "" {
			return fmt.Errorf("%s: %w", field.Key, ErrEmptyValue)
		}
	}

	return nil
}
