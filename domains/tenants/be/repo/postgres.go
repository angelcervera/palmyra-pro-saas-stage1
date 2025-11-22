package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
)

// PostgresRepository implements the tenant repository using the shared persistence layer with immutable versions.
type PostgresRepository struct {
	store *persistence.TenantStore
}

// NewPostgresRepository constructs a repository backed by TenantStore.
func NewPostgresRepository(store *persistence.TenantStore) *PostgresRepository {
	if store == nil {
		panic("tenant store is required")
	}
	return &PostgresRepository{store: store}
}

func (r *PostgresRepository) List(ctx context.Context, opts service.ListOptions) (service.ListResult, error) {
	page := opts.Page
	if page < 1 {
		page = 1
	}
	size := opts.PageSize
	if size <= 0 {
		size = 20
	}
	offset := (page - 1) * size

	var statusStr *string
	if opts.Status != nil {
		s := string(*opts.Status)
		statusStr = &s
	}

	rows, total, err := r.store.ListActive(ctx, statusStr, size, offset)
	if err != nil {
		return service.ListResult{}, err
	}

	tenants := make([]service.Tenant, 0, len(rows))
	for _, rec := range rows {
		t, err := toServiceTenant(rec)
		if err != nil {
			return service.ListResult{}, err
		}
		tenants = append(tenants, t)
	}

	totalPages := (total + size - 1) / size
	return service.ListResult{Tenants: tenants, Page: page, PageSize: size, TotalItems: total, TotalPages: totalPages}, nil
}

func (r *PostgresRepository) Create(ctx context.Context, t service.Tenant) (service.Tenant, error) {
	rec := toRecord(t)
	out, err := r.store.Create(ctx, rec)
	if err != nil {
		return service.Tenant{}, mapConflict(err)
	}
	return toServiceTenant(out)
}

func (r *PostgresRepository) Get(ctx context.Context, id uuid.UUID) (service.Tenant, error) {
	rec, err := r.store.GetActive(ctx, id)
	if err != nil {
		return service.Tenant{}, mapNotFound(err)
	}
	return toServiceTenant(rec)
}

func (r *PostgresRepository) AppendVersion(ctx context.Context, t service.Tenant) (service.Tenant, error) {
	rec := toRecord(t)
	out, err := r.store.AppendVersion(ctx, rec)
	if err != nil {
		return service.Tenant{}, err
	}
	return toServiceTenant(out)
}

func (r *PostgresRepository) FindBySlug(ctx context.Context, slug string) (service.Tenant, error) {
	rec, err := r.store.GetBySlug(ctx, slug)
	if err != nil {
		return service.Tenant{}, mapNotFound(err)
	}
	return toServiceTenant(rec)
}

func toRecord(t service.Tenant) persistence.TenantRecord {
	return persistence.TenantRecord{
		TenantID:          t.ID,
		TenantVersion:     t.Version,
		Slug:              t.Slug,
		DisplayName:       t.DisplayName,
		Status:            string(t.Status),
		SchemaName:        t.SchemaName,
		RoleName:          t.RoleName,
		BasePrefix:        t.BasePrefix,
		ShortTenantID:     t.ShortTenantID,
		IsActive:          true,
		IsSoftDeleted:     false,
		CreatedAt:         t.CreatedAt,
		CreatedBy:         t.CreatedBy,
		DBReady:           t.Provisioning.DBReady,
		AuthReady:         t.Provisioning.AuthReady,
		LastProvisionedAt: t.Provisioning.LastProvisionedAt,
		LastError:         t.Provisioning.LastError,
	}
}

func toServiceTenant(rec persistence.TenantRecord) (service.Tenant, error) {
	status, err := service.TenantStatusFromString(rec.Status)
	if err != nil {
		return service.Tenant{}, err
	}
	if strings.TrimSpace(rec.RoleName) == "" {
		return service.Tenant{}, fmt.Errorf("tenant %s missing role name", rec.TenantID)
	}
	return service.Tenant{
		ID:            rec.TenantID,
		Version:       rec.TenantVersion,
		Slug:          rec.Slug,
		DisplayName:   rec.DisplayName,
		Status:        status,
		SchemaName:    rec.SchemaName,
		RoleName:      rec.RoleName,
		BasePrefix:    rec.BasePrefix,
		ShortTenantID: rec.ShortTenantID,
		CreatedAt:     rec.CreatedAt,
		CreatedBy:     rec.CreatedBy,
		Provisioning: service.ProvisioningStatus{
			DBReady:           rec.DBReady,
			AuthReady:         rec.AuthReady,
			LastProvisionedAt: rec.LastProvisionedAt,
			LastError:         rec.LastError,
		},
	}, nil
}

func mapNotFound(err error) error {
	if errors.Is(err, persistence.ErrNotFound) {
		return service.ErrNotFound
	}
	return err
}

func mapConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" && strings.EqualFold(pgErr.ConstraintName, "tenants_slug_unique_active") {
			return service.ErrConflictSlug
		}
	}
	return err
}

// Ensure interface compliance.
var _ service.Repository = (*PostgresRepository)(nil)

// helper to avoid unused import warning
var _ = fmt.Sprintf
