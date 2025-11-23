package persistence

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// applyCoreSchemaDDL bootstraps the admin schema for tests using the shared helper.
// It remains unexported for backward compatibility in persistence tests.
func applyCoreSchemaDDL(ctx context.Context, pool *pgxpool.Pool) error {
	return BootstrapAdminSchema(ctx, pool, "tenant_admin")
}

// applyDDLToSchema applies the provided SQL statements into the given schema using search_path.
func applyDDLToSchema(ctx context.Context, pool *pgxpool.Pool, schema string, sqlContent string) error {
	if _, err := pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+schema); err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx, `SELECT set_config('search_path', $1, false)`, schema); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, sqlContent); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
