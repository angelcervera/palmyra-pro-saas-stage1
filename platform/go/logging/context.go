package logging

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type ctxKey struct{}

// WithLogger stores the provided logger on the context.
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// FromContext retrieves the logger from context, if present.
func FromContext(ctx context.Context) (*zap.Logger, bool) {
	logger, ok := ctx.Value(ctxKey{}).(*zap.Logger)
	return logger, ok
}

// FromRequest pulls the request-scoped logger from the HTTP request when available, falling back to the provided default.
func FromRequest(r *http.Request, fallback *zap.Logger) *zap.Logger {
	if logger, ok := FromContext(r.Context()); ok {
		return logger
	}
	return fallback
}

// RequestLogger returns an HTTP middleware that enriches the base logger with request scoped fields,
// stores it on the context, and emits a completion log once the handler finishes.
func RequestLogger(base *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := middleware.GetReqID(r.Context())

			logger := base
			if requestID != "" {
				logger = logger.With(zap.String("request_id", requestID))
			}

			logger = logger.With(
				zap.String("http_method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
			)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			ctx := WithLogger(r.Context(), logger)

			next.ServeHTTP(ww, r.WithContext(ctx))

			logger.Info("request completed",
				zap.Int("status", ww.Status()),
				zap.Int("bytes", ww.BytesWritten()),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}
