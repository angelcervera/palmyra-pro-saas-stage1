package persistence

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// TenantRecord represents a versioned tenant row.
type TenantRecord struct {
	TenantID          uuid.UUID       `db:"tenant_id"`
	TenantVersion     SemanticVersion `db:"tenant_version"`
	Slug              string          `db:"slug"`
	DisplayName       *string         `db:"display_name"`
	Status            string          `db:"status"`
	SchemaName        string          `db:"schema_name"`
	RoleName          string          `db:"role_name"`
	BasePrefix        string          `db:"base_prefix"`
	ShortTenantID     string          `db:"short_tenant_id"`
	IsActive          bool            `db:"is_active"`
	IsDeleted         bool            `db:"is_deleted"`
	CreatedAt         time.Time       `db:"created_at"`
	CreatedBy         uuid.UUID       `db:"created_by"`
	DBReady           bool            `db:"db_ready"`
	AuthReady         bool            `db:"auth_ready"`
	LastProvisionedAt *time.Time      `db:"last_provisioned_at"`
	LastError         *string         `db:"last_error"`
}

// ErrNotFound is returned when a tenant record is not found.
var ErrNotFound = errors.New("tenant not found")
