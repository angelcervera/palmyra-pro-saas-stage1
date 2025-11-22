package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	tenantsapi "github.com/zenGate-Global/palmyra-pro-saas/generated/go/tenants"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// Errors returned by the service layer.
var (
	ErrNotFound     = errors.New("tenant not found")
	ErrConflictSlug = errors.New("tenant slug already exists")
)

// Tenant represents the domain model for a tenant registry entry.
type Tenant struct {
	ID            uuid.UUID
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
	Update(ctx context.Context, id uuid.UUID, update UpdateInput) (Tenant, error)
	UpdateProvisioning(ctx context.Context, id uuid.UUID, status ProvisioningStatus) (Tenant, error)
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
	shortID := tenant.ShortID(id)
	schemaName := tenant.BuildSchemaName(tenant.ToSnake(input.Slug))
	basePrefix := tenant.BuildBasePrefix(s.envKey, input.Slug, shortID)

	now := time.Now().UTC()

	t := Tenant{
		ID:            id,
		Slug:          input.Slug,
		DisplayName:   input.DisplayName,
		Status:        input.Status,
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
	return s.repo.Update(ctx, id, input)
}

// Provision performs full provisioning and updates status accordingly.
func (s *Service) Provision(ctx context.Context, id uuid.UUID) (Tenant, error) {
	// For now, simulate provisioning success synchronously.
	now := time.Now().UTC()
	status := ProvisioningStatus{DBReady: true, AuthReady: true, LastProvisionedAt: &now}
	tenant, err := s.repo.UpdateProvisioning(ctx, id, status)
	if err != nil {
		return Tenant{}, err
	}

	// If provisioning succeeds, mark tenant active when pending/provisioning.
	if tenant.Status == tenantsapi.Pending || tenant.Status == tenantsapi.Provisioning {
		active := tenantsapi.Active
		tenant, err = s.repo.Update(ctx, id, UpdateInput{Status: &active})
		if err != nil {
			return Tenant{}, err
		}
	}

	return tenant, nil
}

// ProvisionStatus performs a live check (placeholder) and persists changes if detected.
func (s *Service) ProvisionStatus(ctx context.Context, id uuid.UUID) (ProvisioningStatus, error) {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return ProvisioningStatus{}, err
	}

	// Placeholder: in real impl, re-check external systems. Here just return current and ensure stored.
	// No-op persistence since we have no change detection in the stub implementation.
	return t.Provisioning, nil
}
