package log

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"
)

func New(ctx context.Context, level slog.Level) *Logger {
	return newLogger(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))
}

func ParseLogLevel(s string) (slog.Level, error) {
	var level slog.Level
	err := level.UnmarshalText([]byte(s))
	if err != nil {
		return level, fmt.Errorf("failed to unmarshal slog.Level: %w", err)
	}

	return level, nil
}

func ErrAttr(err error) slog.Attr {
	return slog.Any("error.message", err)
}

func URLAttr(url *url.URL) slog.Attr {
	return slog.String("http.url", url.String())
}

func StatusCodeAttr(sc int) slog.Attr {
	return slog.Int("http.status_code", sc)
}

func PathPatternAttr(path string) slog.Attr {
	return slog.String("http.path", path)
}

func RequestID(requestID string) slog.Attr {
	return slog.String("http.request_id", requestID)
}

func ExecTimeAttr(d time.Duration) slog.Attr {
	return slog.String("exec_time", d.String())
}

func AnyAttr(key string, value any) slog.Attr {
	return slog.Any(key, value)
}
