package tenant

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// Space captures the resolved tenant routing metadata for a request.
// It is intended to be attached to the context by middleware once the tenant
// has been resolved from credentials/claims.
type Space struct {
	TenantID      uuid.UUID
	Slug          string
	ShortTenantID string
	SchemaName    string
	BasePrefix    string
	RoleName      string
}

type ctxKey string

const spaceKey ctxKey = "PALMYRA_TENANT_SPACE"

// WithSpace returns a derived context carrying the tenant Space.
func WithSpace(ctx context.Context, space Space) context.Context {
	return context.WithValue(ctx, spaceKey, space)
}

// FromContext extracts the tenant Space and a boolean indicating presence.
func FromContext(ctx context.Context) (Space, bool) {
	v := ctx.Value(spaceKey)
	if v == nil {
		return Space{}, false
	}

	space, ok := v.(Space)
	return space, ok
}

// BuildSchemaName returns the canonical PostgreSQL schema name for a tenant
// given envKey and the tenant slug transformed to snake_case.
// Format: <envKey>__tenant_<slugSnake> — double underscore keeps the env prefix visually separated
// from the fixed segment and reduces accidental collisions across shared DBs.
func BuildSchemaName(envKey, slugSnake string) string {
	envKey = strings.TrimSpace(envKey)
	return envKey + "__tenant_" + slugSnake
}
