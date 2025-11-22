package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantsTable defines the fully-qualified table for tenant registry.
const TenantsTable = "admin.tenants"

// TenantRecord represents a versioned tenant row.
type TenantRecord struct {
	TenantID          uuid.UUID       `db:"tenant_id"`
	TenantVersion     SemanticVersion `db:"tenant_version"`
	Slug              string          `db:"slug"`
	DisplayName       *string         `db:"display_name"`
	Status            string          `db:"status"`
	SchemaName        string          `db:"schema_name"`
	BasePrefix        string          `db:"base_prefix"`
	ShortTenantID     string          `db:"short_tenant_id"`
	IsActive          bool            `db:"is_active"`
	IsSoftDeleted     bool            `db:"is_soft_deleted"`
	CreatedAt         time.Time       `db:"created_at"`
	CreatedBy         uuid.UUID       `db:"created_by"`
	DBReady           bool            `db:"db_ready"`
	AuthReady         bool            `db:"auth_ready"`
	LastProvisionedAt *time.Time      `db:"last_provisioned_at"`
	LastError         *string         `db:"last_error"`
}

// TenantStore provides access to the tenants table.
type TenantStore struct {
	pool *pgxpool.Pool
}

// NewTenantStore creates a store; assumes migrations already created the table.
func NewTenantStore(ctx context.Context, pool *pgxpool.Pool) (*TenantStore, error) {
	if pool == nil {
		return nil, errors.New("pool is required")
	}
	return &TenantStore{pool: pool}, nil
}

// Create inserts the initial tenant version.
func (s *TenantStore) Create(ctx context.Context, rec TenantRecord) (TenantRecord, error) {
	if rec.TenantID == uuid.Nil {
		return TenantRecord{}, errors.New("tenant id is required")
	}
	if rec.TenantVersion == (SemanticVersion{}) {
		return TenantRecord{}, errors.New("tenant version is required")
	}

	query := fmt.Sprintf(`
        INSERT INTO %s (
            tenant_id, tenant_version, slug, display_name, status, schema_name,
            base_prefix, short_tenant_id, is_active, is_soft_deleted, created_at,
            created_by, db_ready, auth_ready, last_provisioned_at, last_error
        ) VALUES (
            $1,$2,$3,$4,$5,$6,$7,$8,TRUE,FALSE,$9,$10,$11,$12,$13,$14
        )
        RETURNING tenant_id, tenant_version, slug, display_name, status, schema_name,
            base_prefix, short_tenant_id, is_active, is_soft_deleted, created_at,
            created_by, db_ready, auth_ready, last_provisioned_at, last_error
    `, TenantsTable)

	row := s.pool.QueryRow(ctx, query,
		rec.TenantID, rec.TenantVersion.String(), rec.Slug, rec.DisplayName, rec.Status,
		rec.SchemaName, rec.BasePrefix, rec.ShortTenantID, rec.CreatedAt, rec.CreatedBy,
		rec.DBReady, rec.AuthReady, rec.LastProvisionedAt, rec.LastError,
	)

	return scanTenantRecord(row)
}

// AppendVersion inserts a new version and deactivates previous active version.
func (s *TenantStore) AppendVersion(ctx context.Context, rec TenantRecord) (TenantRecord, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return TenantRecord{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	deactivate := fmt.Sprintf("UPDATE %s SET is_active = FALSE WHERE tenant_id = $1", TenantsTable)
	if _, err = tx.Exec(ctx, deactivate, rec.TenantID); err != nil {
		return TenantRecord{}, err
	}

	insert := fmt.Sprintf(`
        INSERT INTO %s (
            tenant_id, tenant_version, slug, display_name, status, schema_name,
            base_prefix, short_tenant_id, is_active, is_soft_deleted, created_at,
            created_by, db_ready, auth_ready, last_provisioned_at, last_error
        ) VALUES (
            $1,$2,$3,$4,$5,$6,$7,$8,TRUE,FALSE,$9,$10,$11,$12,$13,$14
        )
        RETURNING tenant_id, tenant_version, slug, display_name, status, schema_name,
            base_prefix, short_tenant_id, is_active, is_soft_deleted, created_at,
            created_by, db_ready, auth_ready, last_provisioned_at, last_error
    `, TenantsTable)

	row := tx.QueryRow(ctx, insert,
		rec.TenantID, rec.TenantVersion.String(), rec.Slug, rec.DisplayName, rec.Status,
		rec.SchemaName, rec.BasePrefix, rec.ShortTenantID, rec.CreatedAt, rec.CreatedBy,
		rec.DBReady, rec.AuthReady, rec.LastProvisionedAt, rec.LastError,
	)

	out, err := scanTenantRecord(row)
	if err != nil {
		return TenantRecord{}, err
	}

	if err = tx.Commit(ctx); err != nil {
		return TenantRecord{}, err
	}
	return out, nil
}

// GetActive fetches the active tenant version.
func (s *TenantStore) GetActive(ctx context.Context, id uuid.UUID) (TenantRecord, error) {
	query := fmt.Sprintf(`SELECT tenant_id, tenant_version, slug, display_name, status, schema_name,
        base_prefix, short_tenant_id, is_active, is_soft_deleted, created_at, created_by,
        db_ready, auth_ready, last_provisioned_at, last_error
        FROM %s WHERE tenant_id = $1 AND is_active = TRUE AND is_soft_deleted = FALSE`, TenantsTable)
	return scanTenantRecord(s.pool.QueryRow(ctx, query, id))
}

// GetBySlug returns the active tenant by slug.
func (s *TenantStore) GetBySlug(ctx context.Context, slug string) (TenantRecord, error) {
	query := fmt.Sprintf(`SELECT tenant_id, tenant_version, slug, display_name, status, schema_name,
        base_prefix, short_tenant_id, is_active, is_soft_deleted, created_at, created_by,
        db_ready, auth_ready, last_provisioned_at, last_error
        FROM %s WHERE slug = $1 AND is_active = TRUE AND is_soft_deleted = FALSE`, TenantsTable)
	return scanTenantRecord(s.pool.QueryRow(ctx, query, slug))
}

// ListActive returns paginated active tenants with optional status filter.
func (s *TenantStore) ListActive(ctx context.Context, status *string, limit, offset int) ([]TenantRecord, int, error) {
	where := "WHERE is_active = TRUE AND is_soft_deleted = FALSE"
	args := []any{}
	if status != nil {
		where += " AND status = $1"
		args = append(args, *status)
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", TenantsTable, where)
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`SELECT tenant_id, tenant_version, slug, display_name, status, schema_name,
        base_prefix, short_tenant_id, is_active, is_soft_deleted, created_at, created_by,
        db_ready, auth_ready, last_provisioned_at, last_error
        FROM %s %s
        ORDER BY created_at DESC
        LIMIT %d OFFSET %d`, TenantsTable, where, limit, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var records []TenantRecord
	for rows.Next() {
		rec, err := scanTenantRecord(rows)
		if err != nil {
			return nil, 0, err
		}
		records = append(records, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

func scanTenantRecord(row pgx.Row) (TenantRecord, error) {
	var rec TenantRecord
	var versionStr string
	if err := row.Scan(&rec.TenantID, &versionStr, &rec.Slug, &rec.DisplayName, &rec.Status, &rec.SchemaName, &rec.BasePrefix, &rec.ShortTenantID, &rec.IsActive, &rec.IsSoftDeleted, &rec.CreatedAt, &rec.CreatedBy, &rec.DBReady, &rec.AuthReady, &rec.LastProvisionedAt, &rec.LastError); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TenantRecord{}, ErrNotFound
		}
		return TenantRecord{}, err
	}
	ver, err := ParseSemanticVersion(versionStr)
	if err != nil {
		return TenantRecord{}, fmt.Errorf("parse tenant version: %w", err)
	}
	rec.TenantVersion = ver
	return rec, nil
}

// ErrNotFound is returned when a tenant record is not found.
var ErrNotFound = errors.New("tenant not found")

// helper to avoid unused import warning when strings isn't used (but currently used in fmt building)
var _ = strings.Compare
