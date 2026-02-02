package goinertia_test

import (
	"log/slog"
	"testing"

	"github.com/assurrussa/goinertia"
)

func TestLoggerAdapter(t *testing.T) {
	args := []any{
		slog.String("test", "value"),
		slog.String("test2", "value2"),
	}
	adapter := goinertia.NewLoggerAdapter(slog.Default())
	adapter.ErrorContext(t.Context(), "test message", args...)
	adapter.WarnContext(t.Context(), "test message", args...)
	adapter.InfoContext(t.Context(), "test message", args...)
	adapter.DebugContext(t.Context(), "test message", args...)

	adapterNil := goinertia.NewLoggerAdapter(nil)
	adapterNil.ErrorContext(t.Context(), "test message", args...)
	adapterNil.WarnContext(t.Context(), "test message", args...)
	adapterNil.InfoContext(t.Context(), "test message", args...)
	adapterNil.DebugContext(t.Context(), "test message", args...)
}
