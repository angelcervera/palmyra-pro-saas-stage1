package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	ErrEnvMismatch    = errors.New("tenant environment mismatch")
)

// Tenant represents the domain model for a tenant registry entry.
type Tenant struct {
	ID            uuid.UUID
	Version       persistence.SemanticVersion
	Slug          string
	DisplayName   *string
	Status        tenantsapi.TenantStatus
	SchemaName    string
	RoleName      string
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

// TenantStatusFromString converts stored string to TenantStatus; returns error on unknown.
func TenantStatusFromString(s string) (tenantsapi.TenantStatus, error) {
	switch tenantsapi.TenantStatus(s) {
	case tenantsapi.Active, tenantsapi.Disabled, tenantsapi.Pending, tenantsapi.Provisioning:
		return tenantsapi.TenantStatus(s), nil
	default:
		return tenantsapi.Pending, fmt.Errorf("unknown tenant status: %s", s)
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
	repo         Repository
	envKey       string
	adminSchema  string
	provisioning ProvisioningDeps
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

// NewWithProvisioning allows injecting provisioning dependencies; defaults stay zero-value safe for CRUD-only code paths.
func NewWithProvisioning(repo Repository, envKey string, deps ProvisioningDeps) *Service {
	if repo == nil {
		panic("tenants repo is required")
	}
	if envKey == "" {
		panic("envKey is required")
	}
	if deps.DB == nil || deps.Auth == nil || deps.Storage == nil {
		panic("provisioning deps must be non-nil")
	}
	return &Service{repo: repo, envKey: envKey, provisioning: deps}
}

// NewWithProvisioningAndAdmin allows specifying admin schema for DB grants.
func NewWithProvisioningAndAdmin(repo Repository, envKey, adminSchema string, deps ProvisioningDeps) *Service {
	if repo == nil {
		panic("tenants repo is required")
	}
	if envKey == "" {
		panic("envKey is required")
	}
	if deps.DB == nil || deps.Auth == nil || deps.Storage == nil {
		panic("provisioning deps must be non-nil")
	}
	return &Service{repo: repo, envKey: envKey, adminSchema: adminSchema, provisioning: deps}
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
	roleName := tenant.BuildRoleName(schemaName)
	basePrefix := tenant.BuildBasePrefix(s.envKey, input.Slug, shortID)

	now := time.Now().UTC()

	t := Tenant{
		ID:            id,
		Slug:          input.Slug,
		DisplayName:   input.DisplayName,
		Status:        input.Status,
		Version:       version,
		SchemaName:    schemaName,
		RoleName:      roleName,
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
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return Tenant{}, err
	}
	if current.Status == tenantsapi.Disabled {
		return Tenant{}, ErrDisabled
	}
	if strings.TrimSpace(current.SchemaName) == "" {
		return Tenant{}, fmt.Errorf("tenant missing schema name")
	}
	if strings.TrimSpace(current.BasePrefix) == "" {
		return Tenant{}, fmt.Errorf("tenant missing base prefix")
	}
	if strings.TrimSpace(current.RoleName) == "" {
		return Tenant{}, fmt.Errorf("tenant missing role name")
	}

	now := time.Now().UTC()
	roleName := current.RoleName

	dbRes, dbErr := s.provisioning.DB.Ensure(ctx, DBProvisionRequest{
		TenantID:    current.ID,
		SchemaName:  current.SchemaName,
		RoleName:    roleName,
		AdminSchema: s.adminSchema,
	})
	authRes, authErr := s.provisioning.Auth.Ensure(ctx, fmt.Sprintf("%s-%s", s.envKey, current.Slug))
	_, storageErr := s.provisioning.Storage.Check(ctx, current.BasePrefix)

	dbReady := current.Provisioning.DBReady || dbRes.Ready
	authReady := current.Provisioning.AuthReady || authRes.Ready

	status := current.Status
	if dbReady && authReady {
		status = tenantsapi.Active
	} else {
		status = tenantsapi.Provisioning
	}

	var lastErr *string
	if dbErr != nil {
		s := dbErr.Error()
		lastErr = &s
	}
	if authErr != nil && lastErr == nil {
		s := authErr.Error()
		lastErr = &s
	}
	if storageErr != nil && lastErr == nil {
		s := storageErr.Error()
		lastErr = &s
	}

	prov := ProvisioningStatus{
		DBReady:           dbReady,
		AuthReady:         authReady,
		LastProvisionedAt: current.Provisioning.LastProvisionedAt,
		LastError:         lastErr,
	}
	if dbReady && authReady {
		prov.LastProvisionedAt = &now
	}

	next := current
	next.RoleName = roleName
	next.Status = status
	next.Provisioning = prov
	next.Version = current.Version.NextPatch()
	next.CreatedAt = now

	updated, err := s.repo.AppendVersion(ctx, next)
	if err != nil {
		return Tenant{}, err
	}
	return updated, nil
}

// ProvisionStatus performs a live check (placeholder) and persists changes if detected.
func (s *Service) ProvisionStatus(ctx context.Context, id uuid.UUID) (ProvisioningStatus, error) {
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return ProvisioningStatus{}, err
	}
	if strings.TrimSpace(current.SchemaName) == "" {
		return ProvisioningStatus{}, fmt.Errorf("tenant missing schema name")
	}
	if strings.TrimSpace(current.BasePrefix) == "" {
		return ProvisioningStatus{}, fmt.Errorf("tenant missing base prefix")
	}
	if strings.TrimSpace(current.RoleName) == "" {
		return ProvisioningStatus{}, fmt.Errorf("tenant missing role name")
	}

	roleName := current.RoleName

	dbRes, dbErr := s.provisioning.DB.Check(ctx, DBProvisionRequest{TenantID: current.ID, SchemaName: current.SchemaName, RoleName: roleName, AdminSchema: s.adminSchema})
	if dbErr != nil {
		return ProvisioningStatus{}, dbErr
	}
	authRes, authErr := s.provisioning.Auth.Check(ctx, fmt.Sprintf("%s-%s", s.envKey, current.Slug))
	if authErr != nil {
		return ProvisioningStatus{}, authErr
	}
	if _, storageErr := s.provisioning.Storage.Check(ctx, current.BasePrefix); storageErr != nil {
		return ProvisioningStatus{}, storageErr
	}

	dbReady := dbRes.Ready
	authReady := authRes.Ready

	var lastErr *string

	status := current.Status
	if dbReady && authReady {
		status = tenantsapi.Active
	} else if status == tenantsapi.Active {
		status = tenantsapi.Provisioning
	}

	prov := ProvisioningStatus{
		DBReady:           dbReady,
		AuthReady:         authReady,
		LastProvisionedAt: current.Provisioning.LastProvisionedAt,
		LastError:         lastErr,
	}

	if dbReady && authReady && prov.LastProvisionedAt == nil {
		now := time.Now().UTC()
		prov.LastProvisionedAt = &now
	}

	// Only append a new version if anything changed.
	if status == current.Status && provisioningEqual(prov, current.Provisioning) {
		return current.Provisioning, nil
	}

	next := current
	next.RoleName = roleName
	next.Status = status
	next.Provisioning = prov
	next.Version = current.Version.NextPatch()
	next.CreatedAt = time.Now().UTC()

	updated, err := s.repo.AppendVersion(ctx, next)
	if err != nil {
		return ProvisioningStatus{}, err
	}
	return updated.Provisioning, nil
}

func provisioningEqual(a, b ProvisioningStatus) bool {
	if a.DBReady != b.DBReady || a.AuthReady != b.AuthReady {
		return false
	}
	if (a.LastError == nil) != (b.LastError == nil) {
		return false
	}
	if a.LastError != nil && b.LastError != nil && *a.LastError != *b.LastError {
		return false
	}
	if (a.LastProvisionedAt == nil) != (b.LastProvisionedAt == nil) {
		return false
	}
	if a.LastProvisionedAt != nil && b.LastProvisionedAt != nil && !a.LastProvisionedAt.Equal(*b.LastProvisionedAt) {
		return false
	}
	return true
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
		RoleName:      t.RoleName,
	}
	return space, nil
}

// ResolveTenantSpaceByExternal maps an external tenant key (envKey-prefixed slug) to a tenant.Space.
func (s *Service) ResolveTenantSpaceByExternal(ctx context.Context, external string) (tenant.Space, error) {
	external = strings.TrimSpace(external)
	if external == "" {
		return tenant.Space{}, ErrNotFound
	}

	prefix := s.envKey + "-"
	if !strings.HasPrefix(external, prefix) {
		return tenant.Space{}, ErrEnvMismatch
	}
	slug := strings.TrimPrefix(external, prefix)

	t, err := s.repo.FindBySlug(ctx, slug)
	if err != nil {
		return tenant.Space{}, fmt.Errorf("lookup tenant by slug: %w", err)
	}
	if t.Status == tenantsapi.Disabled {
		return tenant.Space{}, ErrDisabled
	}

	return tenant.Space{
		TenantID:      t.ID,
		Slug:          t.Slug,
		ShortTenantID: t.ShortTenantID,
		SchemaName:    t.SchemaName,
		BasePrefix:    t.BasePrefix,
		RoleName:      t.RoleName,
	}, nil
}
