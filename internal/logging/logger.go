// Package logging описывает общий интерфейс журналирования приложения.
package logging

import "context"

// Logger записывает события приложения и диагностические ошибки.
type Logger interface {
	InfoContext(context.Context, string, ...any)
	ErrorContext(context.Context, string, ...any)
}
