package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	tenantsapi "github.com/zenGate-Global/palmyra-pro-saas/generated/go/tenants"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// Errors returned by the service layer.
var (
	ErrNotFound       = errors.New("tenant not found")
	ErrConflictSlug   = errors.New("tenant slug already exists")
	ErrDisabled       = errors.New("tenant disabled")
	ErrNotImplemented = errors.New("provisioning not implemented yet")
)

// Tenant represents the domain model for a tenant registry entry.
type Tenant struct {
	ID            uuid.UUID
	Version       persistence.SemanticVersion
	Slug          string
	DisplayName   *string
	Status        tenantsapi.TenantStatus
	SchemaName    string
	BasePrefix    string
	ShortTenantID string
	CreatedAt     time.Time
	CreatedBy     uuid.UUID
	Provisioning  ProvisioningStatus
}

// ProvisioningStatus captures environment provisioning state.
type ProvisioningStatus struct {
	DBReady           bool
	AuthReady         bool
	LastProvisionedAt *time.Time
	LastError         *string
}

// TenantStatusFromString converts stored string to TenantStatus; defaults to pending on unknown.
func TenantStatusFromString(s string) tenantsapi.TenantStatus {
	switch tenantsapi.TenantStatus(s) {
	case tenantsapi.Active, tenantsapi.Disabled, tenantsapi.Pending, tenantsapi.Provisioning:
		return tenantsapi.TenantStatus(s)
	default:
		return tenantsapi.Pending
	}
}

// CreateInput represents the request to create a tenant.
type CreateInput struct {
	Slug        string
	DisplayName *string
	Status      tenantsapi.TenantStatus
	CreatedBy   uuid.UUID
}

// UpdateInput represents mutable fields for a tenant.
type UpdateInput struct {
	DisplayName *string
	Status      *tenantsapi.TenantStatus
}

// ListResult wraps paginated tenants.
type ListResult struct {
	Tenants    []Tenant
	Page       int
	PageSize   int
	TotalItems int
	TotalPages int
}

// ListOptions captures filters and pagination.
type ListOptions struct {
	Page     int
	PageSize int
	Status   *tenantsapi.TenantStatus
}

// Repository abstracts persistence.
type Repository interface {
	List(ctx context.Context, opts ListOptions) (ListResult, error)
	Create(ctx context.Context, t Tenant) (Tenant, error)
	Get(ctx context.Context, id uuid.UUID) (Tenant, error)
	AppendVersion(ctx context.Context, t Tenant) (Tenant, error)
	FindBySlug(ctx context.Context, slug string) (Tenant, error)
}

// Service provides tenant registry operations.
type Service struct {
	repo   Repository
	envKey string
}

// New constructs a Service with required dependencies.
func New(repo Repository, envKey string) *Service {
	if repo == nil {
		panic("tenants repo is required")
	}
	if envKey == "" {
		panic("envKey is required")
	}
	return &Service{repo: repo, envKey: envKey}
}

// List tenants with optional status filter.
func (s *Service) List(ctx context.Context, opts ListOptions) (ListResult, error) {
	return s.repo.List(ctx, opts)
}

// Create a new tenant with derived fields.
func (s *Service) Create(ctx context.Context, input CreateInput) (Tenant, error) {
	id := uuid.New()
	version := persistence.SemanticVersion{Major: 1, Minor: 0, Patch: 0}
	shortID := tenant.ShortID(id)
	schemaName := tenant.BuildSchemaName(tenant.ToSnake(input.Slug))
	basePrefix := tenant.BuildBasePrefix(s.envKey, input.Slug, shortID)

	now := time.Now().UTC()

	t := Tenant{
		ID:            id,
		Slug:          input.Slug,
		DisplayName:   input.DisplayName,
		Status:        input.Status,
		Version:       version,
		SchemaName:    schemaName,
		BasePrefix:    basePrefix,
		ShortTenantID: shortID,
		CreatedAt:     now,
		CreatedBy:     input.CreatedBy,
		Provisioning: ProvisioningStatus{
			DBReady:   false,
			AuthReady: false,
		},
	}

	return s.repo.Create(ctx, t)
}

// Get returns a tenant by id.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (Tenant, error) {
	return s.repo.Get(ctx, id)
}

// Update modifies mutable fields of a tenant.
func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (Tenant, error) {
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return Tenant{}, err
	}

	next := current
	if input.DisplayName != nil {
		next.DisplayName = input.DisplayName
	}
	if input.Status != nil {
		next.Status = *input.Status
	}
	next.Version = current.Version.NextPatch()
	next.CreatedAt = time.Now().UTC()

	return s.repo.AppendVersion(ctx, next)
}

// Provision performs full provisioning and updates status accordingly.
func (s *Service) Provision(ctx context.Context, id uuid.UUID) (Tenant, error) {
	return Tenant{}, ErrNotImplemented
}

// ProvisionStatus performs a live check (placeholder) and persists changes if detected.
func (s *Service) ProvisionStatus(ctx context.Context, id uuid.UUID) (ProvisioningStatus, error) {
	return ProvisioningStatus{}, ErrNotImplemented
}

// ResolveTenantSpace returns a lightweight tenant Space for middleware consumption.
func (s *Service) ResolveTenantSpace(ctx context.Context, id uuid.UUID) (tenant.Space, error) {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return tenant.Space{}, err
	}
	if t.Status == tenantsapi.Disabled {
		return tenant.Space{}, ErrDisabled
	}
	space := tenant.Space{
		TenantID:      t.ID,
		Slug:          t.Slug,
		ShortTenantID: t.ShortTenantID,
		SchemaName:    t.SchemaName,
		BasePrefix:    t.BasePrefix,
	}
	return space, nil
}
