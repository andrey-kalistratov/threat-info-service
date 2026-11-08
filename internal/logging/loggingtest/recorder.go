// Package loggingtest содержит приёмник записей для проверки журналирования в тестах.
package loggingtest

import (
	"context"
	"log/slog"
)

// Recorder сохраняет записи журнала для последующих проверок.
type Recorder struct {
	Info   []Entry
	Errors []Entry
}

// Entry — сообщение и именованные атрибуты одной записи журнала.
type Entry struct {
	Message string
	Attrs   map[string]any
}

// InfoContext сохраняет информационную запись.
func (r *Recorder) InfoContext(_ context.Context, message string, attrs ...any) {
	r.Info = append(r.Info, newEntry(message, attrs...))
}

// ErrorContext сохраняет запись об ошибке.
func (r *Recorder) ErrorContext(_ context.Context, message string, attrs ...any) {
	r.Errors = append(r.Errors, newEntry(message, attrs...))
}

func newEntry(message string, attrs ...any) Entry {
	record := slog.Record{
		Message: message,
	}
	record.Add(attrs...)
	fields := make(map[string]any)
	record.Attrs(func(attr slog.Attr) bool {
		fields[attr.Key] = attr.Value.Any()
		return true
	})

	return Entry{
		Message: message,
		Attrs:   fields,
	}
}
