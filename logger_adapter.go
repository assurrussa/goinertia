package goinertia

import (
	"context"
)

type LoggerAdapter struct {
	logger Logger
}

func NewLoggerAdapter(logger Logger) *LoggerAdapter {
	return &LoggerAdapter{logger: logger}
}

func (l *LoggerAdapter) ErrorContext(ctx context.Context, msg string, args ...any) {
	if l.logger == nil {
		return
	}

	l.logger.ErrorContext(ctx, msg, args...)
}

func (l *LoggerAdapter) WarnContext(ctx context.Context, msg string, args ...any) {
	if l.logger == nil {
		return
	}

	l.logger.WarnContext(ctx, msg, args...)
}

func (l *LoggerAdapter) InfoContext(ctx context.Context, msg string, args ...any) {
	if l.logger == nil {
		return
	}

	l.logger.InfoContext(ctx, msg, args...)
}

func (l *LoggerAdapter) DebugContext(ctx context.Context, msg string, args ...any) {
	if l.logger == nil {
		return
	}

	l.logger.DebugContext(ctx, msg, args...)
}
