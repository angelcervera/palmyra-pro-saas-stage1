package tenant

import (
	"strings"

	"github.com/google/uuid"
)

// ToSnake converts a kebab-case slug into snake_case for schema names.
func ToSnake(slug string) string {
	return strings.ReplaceAll(strings.ToLower(slug), "-", "_")
}

// ShortID returns the first 8 hexadecimal characters of a UUID (without dashes).
func ShortID(id uuid.UUID) string {
	hex := strings.ReplaceAll(id.String(), "-", "")
	if len(hex) < 8 {
		return hex
	}
	return hex[:8]
}

// BuildBasePrefix returns `<envKey>/<tenantSlug>-<shortTenantId>/`.
func BuildBasePrefix(envKey, slug string, shortID string) string {
	envKey = strings.TrimSuffix(envKey, "/")
	return envKey + "/" + slug + "-" + shortID + "/"
}

// BuildRoleName returns the tenant runtime role name derived from the schema name.
func BuildRoleName(schemaName string) string {
	return schemaName + "_role"
}

// DerivedIdentifiers groups the identifiers derived from slug/env/tenantID.
type DerivedIdentifiers struct {
	SchemaName    string
	RoleName      string
	BasePrefix    string
	ShortTenantID string
}

// DeriveIdentifiers returns schema name, role name, base prefix, and short ID for a tenant.
func DeriveIdentifiers(envKey, slug string, tenantID uuid.UUID) DerivedIdentifiers {
	slugSnake := ToSnake(slug)
	schema := BuildSchemaName(slugSnake)
	shortID := ShortID(tenantID)
	return DerivedIdentifiers{
		SchemaName:    schema,
		RoleName:      BuildRoleName(schema),
		BasePrefix:    BuildBasePrefix(envKey, slug, shortID),
		ShortTenantID: shortID,
	}
}
