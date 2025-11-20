package requesttrace

import (
	"context"
	"errors"

	platformauth "github.com/zenGate-Global/palmyra-pro-saas/platform/go/auth"
)

type contextKey string

const (
	ctxAuditInfo contextKey = "PALMYRA_REQUEST_TRACE"
)

// ActorKind represents who initiated a request.
type ActorKind string

const (
	ActorKindUser      ActorKind = "user"
	ActorKindAnonymous ActorKind = "anonymous"
	ActorKindSystem    ActorKind = "system"
)

// AuditInfo captures request-scoped metadata needed for traceability and auditing.
// UserID is optional; set only when ActorKind is user.
// TenantID may be nil when the auth provider does not supply a tenant; RequestID is optional but encouraged.
type AuditInfo struct {
	ActorKind ActorKind
	UserID    *string
	TenantID  *string
	RequestID string
}

// IntoContext stores the AuditInfo in the provided context.
func IntoContext(ctx context.Context, audit AuditInfo) context.Context {
	return context.WithValue(ctx, ctxAuditInfo, audit)
}

// FromContext extracts the AuditInfo from context, returning false when not present.
func FromContext(ctx context.Context) (AuditInfo, bool) {
	if ctx == nil {
		return AuditInfo{}, false
	}
	v := ctx.Value(ctxAuditInfo)
	if v == nil {
		return AuditInfo{}, false
	}

	audit, ok := v.(AuditInfo)
	return audit, ok
}

// FromContextOrAnonymous returns the AuditInfo stored on the context, or an anonymous record when absent.
func FromContextOrAnonymous(ctx context.Context) AuditInfo {
	if audit, ok := FromContext(ctx); ok {
		return audit
	}
	return Anonymous("")
}

// FromCredentials builds an AuditInfo from authenticated user credentials and a request ID.
// Returns an error when creds are nil or missing a UserID.
func FromCredentials(creds *platformauth.UserCredentials, requestID string) (AuditInfo, error) {
	if creds == nil {
		return AuditInfo{}, errors.New("credentials are required to build audit info")
	}
	if creds.Id == "" {
		return AuditInfo{}, errors.New("user id is required to build audit info")
	}

	return AuditInfo{
		ActorKind: ActorKindUser,
		UserID:    &creds.Id,
		TenantID:  creds.TenantID,
		RequestID: requestID,
	}, nil
}

// Anonymous builds an AuditInfo for unauthenticated requests (e.g., signup) where no user ID exists yet.
func Anonymous(requestID string) AuditInfo {
	return AuditInfo{ActorKind: ActorKindAnonymous, RequestID: requestID}
}

// System builds an AuditInfo for background/system operations.
func System(requestID string) AuditInfo {
	return AuditInfo{ActorKind: ActorKindSystem, RequestID: requestID}
}
