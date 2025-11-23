package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	sqlassets "github.com/zenGate-Global/palmyra-pro-saas/database"
)

// BootstrapAdminSchema creates the admin schema (if missing) and applies the
// platform bootstrap DDL in a single transaction. The statements are executed
// with search_path set to the admin schema, in this order:
//  1. tenant_space/users.sql
//  2. platform/entity_schemas.sql
//  3. platform/tenants.sql
//
// SQL is embedded at build time so binaries stay self-contained. The helper is
// idempotent and intended for CLI bootstrap and tests.
func BootstrapAdminSchema(ctx context.Context, pool *pgxpool.Pool, adminSchema string) error {
	if pool == nil {
		return fmt.Errorf("bootstrap admin schema: pool is required")
	}
	if adminSchema == "" {
		return fmt.Errorf("bootstrap admin schema: admin schema is required")
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	if _, err := tx.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS "+pgx.Identifier{adminSchema}.Sanitize()); err != nil {
		return fmt.Errorf("create admin schema: %w", err)
	}

	if _, err := tx.Exec(ctx, `SELECT set_config('search_path', $1, false)`, adminSchema); err != nil {
		return fmt.Errorf("set search_path: %w", err)
	}

	for _, ddl := range []string{sqlassets.UsersSQL, sqlassets.EntitySchemasSQL, sqlassets.TenantsSQL} {
		if _, err := tx.Exec(ctx, ddl); err != nil {
			return fmt.Errorf("apply ddl: %w", err)
		}
	}

	return tx.Commit(ctx)
}
