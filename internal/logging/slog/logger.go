// Package slog создает логгер на основе стандартного log/slog.
package slog

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/andrey-kalistratov/threat-info-service/internal/config"
)

// New создает настроенный логгер, записывающий в стандартный вывод.
func New(cfg *config.Log) (*slog.Logger, error) {
	level := new(slog.LevelVar)
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}
	options := &slog.HandlerOptions{Level: level}
	if cfg.Format == "json" {
		return slog.New(slog.NewJSONHandler(os.Stdout, options)), nil
	}
	return slog.New(slog.NewTextHandler(os.Stdout, options)), nil
}

// NewTemporary создает логгер для сообщений до загрузки настроек сервиса.
func NewTemporary() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
