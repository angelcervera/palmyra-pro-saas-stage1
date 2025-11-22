package service

import (
	"context"

	"github.com/google/uuid"
)

// DBProvisioner encapsulates creation/check of tenant-specific DB artifacts (role, schema, grants, base tables).
// Ensure is mutating/idempotent, Check is read-only/health verification.
type DBProvisioner interface {
	Ensure(ctx context.Context, req DBProvisionRequest) (DBProvisionResult, error)
	Check(ctx context.Context, req DBProvisionRequest) (DBProvisionResult, error)
}

type DBProvisionRequest struct {
	TenantID    uuid.UUID
	SchemaName  string
	RoleName    string
	AdminSchema string
}

type DBProvisionResult struct {
	Ready bool
}

// AuthProvisioner manages external auth tenant creation/check.
// Ensure is mutating/idempotent, Check is read-only/health verification.
type AuthProvisioner interface {
	Ensure(ctx context.Context, externalTenant string) (AuthProvisionResult, error)
	Check(ctx context.Context, externalTenant string) (AuthProvisionResult, error)
}

type AuthProvisionResult struct {
	Ready bool
}

// StorageProvisioner validates storage reachability.
// Ensure is mutating/idempotent, Check is read-only/health verification.
type StorageProvisioner interface {
	Ensure(ctx context.Context, prefix string) (StorageProvisionResult, error)
	Check(ctx context.Context, prefix string) (StorageProvisionResult, error)
}

type StorageProvisionResult struct {
	Ready bool
}

type ProvisioningDeps struct {
	DB      DBProvisioner
	Auth    AuthProvisioner
	Storage StorageProvisioner
}
