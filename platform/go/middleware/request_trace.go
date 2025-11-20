package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	platformauth "github.com/zenGate-Global/palmyra-pro-saas/platform/go/auth"
	platformlogging "github.com/zenGate-Global/palmyra-pro-saas/platform/go/logging"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/requesttrace"
)

// RequestTrace populates the context with request-scoped AuditInfo so services and repositories can stamp audit fields.
// It should run after authentication middleware so user credentials are available when present.
func RequestTrace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := platformlogging.FromRequest(r, nil)
		requestID, _ := r.Context().Value(middleware.RequestIDKey).(string)

		var audit requesttrace.AuditInfo
		if creds, ok := platformauth.UserFromContext(r.Context()); ok && creds != nil {
			var err error
			audit, err = requesttrace.FromCredentials(creds, requestID)
			if err != nil {
				if logger != nil {
					logger.Error("build audit info from credentials", zap.Error(err))
				}
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		} else {
			audit = requesttrace.Anonymous(requestID)
		}

		ctx := requesttrace.IntoContext(r.Context(), audit)
		if logger != nil {
			fields := []zap.Field{zap.String("actor_kind", string(audit.ActorKind))}
			if audit.UserID != nil && *audit.UserID != "" {
				fields = append(fields, zap.String("user_id", *audit.UserID))
			}
			logger = logger.With(fields...)
			ctx = platformlogging.WithLogger(ctx, logger)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
