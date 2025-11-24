package persistence

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// txBeginner exposes the minimal pgx pool behaviour needed by SpaceDB.
type txBeginner interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

// SpaceDB wraps a pgx pool to execute queries within a space-specific search_path.
type SpaceDB struct {
	pool        txBeginner
	adminSchema string
}

type SpaceDBConfig struct {
	Pool        *pgxpool.Pool
	AdminSchema string
}

func NewSpaceDB(cfg SpaceDBConfig) *SpaceDB {
	if cfg.Pool == nil {
		panic("SpaceDB requires pool")
	}

	adminSchema := strings.TrimSpace(cfg.AdminSchema)
	if adminSchema == "" {
		panic("SpaceDB requires admin schema")
	}
	return &SpaceDB{pool: cfg.Pool, adminSchema: adminSchema}
}

// WithAdmin executes fn inside a transaction scoped to the admin schema only.
// No role switching is performed; caller must rely on the connection's identity.
func (db *SpaceDB) WithAdmin(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT set_config('search_path', $1, true)`, db.adminSchema); err != nil {
		return fmt.Errorf("set search_path: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// WithTenant executes fn inside a transaction with search_path set to space + admin schema.
func (db *SpaceDB) WithTenant(ctx context.Context, tenantSpace tenant.Space, fn func(tx pgx.Tx) error) error {
	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	if strings.TrimSpace(tenantSpace.RoleName) == "" {
		return fmt.Errorf("tenantSpace role is required in tenant.Space")
	}

	if _, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL ROLE %s", pgx.Identifier{tenantSpace.RoleName}.Sanitize())); err != nil {
		return fmt.Errorf("set role: %w", err)
	}

	searchPath := fmt.Sprintf("%s, %s", tenantSpace.SchemaName, db.adminSchema)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path', $1, true)`, searchPath); err != nil {
		return fmt.Errorf("set search_path: %w", err)
	}

	// TODO: Check if it is possible to set read permissions for the schema tables for this transaction.

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
