package telemetry

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/segmentio/ksuid"
	"github.com/vit0rr/chat/api/constants"
)

func getRoutePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return r.URL.Path
	}
	if pattern := rctx.RoutePattern(); pattern != "" {
		// Pattern is already available
		return pattern
	}

	routePath := r.URL.Path
	if r.URL.RawPath != "" {
		routePath = r.URL.RawPath
	}

	tctx := chi.NewRouteContext()
	if !rctx.Routes.Match(tctx, r.Method, routePath) {
		// No matching pattern, so just return the request path.
		// Depending on your use case, it might make sense to
		// return an empty string or error here instead
		return routePath
	}

	// tctx has the updated pattern, since Match mutates it
	return tctx.RoutePattern()
}

// TelemetryMiddleware is a middleware that adds general telemetry info to the request
func TelemetryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Set the request ID
		var requestID string
		requestID = r.Header.Get(constants.HeaderRequestID)
		if requestID == "" {
			requestID = ksuid.New().String()
		}

		ctx = context.WithValue(ctx, constants.CtxKeyRequestID, requestID)

		// Set the Real IP
		if r.RemoteAddr != "" {
			ctx = context.WithValue(ctx, constants.CtxKeyRealIP, r.RemoteAddr)
		}

		// Set the path pattern
		ctx = context.WithValue(ctx, constants.CtxKeyPathPattern, getRoutePattern(r))

		// Set the method
		ctx = context.WithValue(ctx, constants.CtxKeyMethod, r.Method)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
