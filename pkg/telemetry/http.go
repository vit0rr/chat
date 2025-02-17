package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/vit0rr/chat/api/handler"
	"github.com/vit0rr/chat/pkg/log"
)

// RoundTripperLogger implements http.RoundTripper to add outbound request start
// & end logs.
type RoundTripperLogger struct {
	Transport http.RoundTripper
	Logger    *slog.Logger
}

// RoundTrip handles adds outbound request start/end logs to any RoundTrip
// executions.
func (rtl RoundTripperLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	logger := rtl.Logger.With(log.URLAttr(req.URL))
	logger.InfoContext(ctx, "outbound request start")

	start := time.Now()
	level := slog.LevelInfo
	res, err := rtl.Transport.RoundTrip(req)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			level = slog.LevelError
		}
	} else {
		logger = logger.With(log.StatusCodeAttr(res.StatusCode))
	}

	logger.LogAttrs(
		ctx,
		level,
		"outbound request end",
		log.ErrAttr(err),
		log.ExecTimeAttr(time.Since(start)),
	)
	return res, err
}

// HandleFuncLogger wraps http.HandlerFunc with incoming request start & end
// logs.
func HandleFuncLogger(h handler.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		start := time.Now()
		log.Info(ctx, "Incoming request start")
		h.ServeHTTP(w, r)
		log.Info(ctx, "Incoming request end", log.ExecTimeAttr(time.Since(start)))
	})
}
