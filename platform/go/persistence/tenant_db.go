package persistence

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// TenantDB wraps a pgx pool to execute queries within a tenant-specific search_path.
type txBeginner interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

// TenantDB wraps a pgx pool to execute queries within a tenant-specific search_path.
type TenantDB struct {
	pool        txBeginner
	adminSchema string
}

type TenantDBConfig struct {
	Pool        *pgxpool.Pool
	AdminSchema string
}

func NewTenantDB(cfg TenantDBConfig) *TenantDB {
	if cfg.Pool == nil {
		panic("TenantDB requires pool")
	}

	adminSchema := strings.TrimSpace(cfg.AdminSchema)
	if adminSchema == "" {
		panic("TenantDB requires admin schema")
	}
	return &TenantDB{pool: cfg.Pool, adminSchema: adminSchema}
}

// WithAdmin executes fn inside a transaction scoped to the admin schema only.
// No role switching is performed; caller must rely on the connection's identity.
func (db *TenantDB) WithAdmin(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT set_config('search_path', $1, false)`, db.adminSchema); err != nil {
		return fmt.Errorf("set search_path: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// WithTenant executes fn inside a transaction with search_path set to tenant + admin schema.
func (db *TenantDB) WithTenant(ctx context.Context, space tenant.Space, fn func(tx pgx.Tx) error) error {
	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	if strings.TrimSpace(space.RoleName) == "" {
		return fmt.Errorf("tenant role is required in tenant.Space")
	}

	if _, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL ROLE %s", pgx.Identifier{space.RoleName}.Sanitize())); err != nil {
		return fmt.Errorf("set role: %w", err)
	}

	searchPath := fmt.Sprintf("%s, %s", space.SchemaName, db.adminSchema)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path', $1, false)`, searchPath); err != nil {
		return fmt.Errorf("set search_path: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
