package log

import (
	"context"
	"log/slog"
	"os"

	"github.com/vit0rr/chat/api/constants"
)

type Logger struct {
	logger *slog.Logger
}

var loggerSingleton *Logger

func newLogger(logger *slog.Logger) *Logger {
	newLogger := &Logger{
		logger: logger,
	}

	loggerSingleton = newLogger

	return newLogger
}

func getLogger() *Logger {
	if loggerSingleton != nil {
		return loggerSingleton
	}

	return newLogger(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
}

// Need to return any to satisfy the .With() method
func getAttrsFromContext(ctx context.Context) []any {
	attrs := make([]any, 0)

	for _, entry := range constants.ContextKeys {
		if value := ctx.Value(entry.Key); value != nil {
			attrs = append(attrs, slog.Any(entry.Label, value))
		}
	}
	return attrs
}

func Debug(ctx context.Context, msg string, args ...any) {
	getLogger().logger.With(getAttrsFromContext(ctx)...).DebugContext(ctx, msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	getLogger().logger.With(getAttrsFromContext(ctx)...).ErrorContext(ctx, msg, args...)
}

func Info(ctx context.Context, msg string, args ...any) {
	getLogger().logger.With(getAttrsFromContext(ctx)...).InfoContext(ctx, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	getLogger().logger.With(getAttrsFromContext(ctx)...).WarnContext(ctx, msg, args...)
}
