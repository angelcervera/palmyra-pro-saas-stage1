package service

import (
	"context"

	"github.com/google/uuid"
)

// DBProvisioner encapsulates creation/check of tenant-specific DB artifacts
// (role, schema, grants, base shared tables). Implementations must be idempotent.
type DBProvisioner interface {
	Ensure(ctx context.Context, req DBProvisionRequest) (DBProvisionResult, error)
	Check(ctx context.Context, req DBProvisionRequest) (DBProvisionResult, error)
}

// DBProvisionRequest carries derived identifiers needed for DB provisioning.
type DBProvisionRequest struct {
	TenantID    uuid.UUID
	SchemaName  string
	RoleName    string
	AdminSchema string
}

// DBProvisionResult summarizes readiness; can be extended with diagnostics later.
type DBProvisionResult struct {
	Ready bool
}

// AuthProvisioner manages external auth tenant creation/check (e.g., Firebase/Identity).
type AuthProvisioner interface {
	Ensure(ctx context.Context, externalTenant string) (AuthProvisionResult, error)
	Check(ctx context.Context, externalTenant string) (AuthProvisionResult, error)
}

type AuthProvisionResult struct {
	Ready bool
}

// StorageProvisioner validates storage reachability (e.g., GCS prefix).
type StorageProvisioner interface {
	Check(ctx context.Context, prefix string) (StorageProvisionResult, error)
}

type StorageProvisionResult struct {
	Ready bool
}

// Locker provides per-tenant mutual exclusion during provisioning.
type Locker interface {
	WithLock(ctx context.Context, tenantID uuid.UUID, fn func(context.Context) error) error
}

// ProvisioningDeps groups the injectable collaborators used by Provision/ProvisionStatus.
type ProvisioningDeps struct {
	DB      DBProvisioner
	Auth    AuthProvisioner
	Storage StorageProvisioner
}

// WithDefaults returns deps with nil fields replaced by no-op stubs; to be implemented in wiring.
func (d ProvisioningDeps) WithDefaults() ProvisioningDeps { return d }

// FillNil sets default no-op implementations where nil to avoid panics in CRUD-only usage.
func (d ProvisioningDeps) FillNil() ProvisioningDeps {
	if d.DB == nil {
		d.DB = noopDBProvisioner{}
	}
	if d.Auth == nil {
		d.Auth = noopAuthProvisioner{}
	}
	if d.Storage == nil {
		d.Storage = noopStorageProvisioner{}
	}
	return d
}

type noopDBProvisioner struct{}

func (noopDBProvisioner) Ensure(context.Context, DBProvisionRequest) (DBProvisionResult, error) {
	return DBProvisionResult{Ready: true}, nil
}
func (noopDBProvisioner) Check(context.Context, DBProvisionRequest) (DBProvisionResult, error) {
	return DBProvisionResult{Ready: true}, nil
}

type noopAuthProvisioner struct{}

func (noopAuthProvisioner) Ensure(context.Context, string) (AuthProvisionResult, error) {
	return AuthProvisionResult{Ready: true}, nil
}
func (noopAuthProvisioner) Check(context.Context, string) (AuthProvisionResult, error) {
	return AuthProvisionResult{Ready: true}, nil
}

type noopStorageProvisioner struct{}

func (noopStorageProvisioner) Check(context.Context, string) (StorageProvisionResult, error) {
	return StorageProvisionResult{Ready: true}, nil
}
