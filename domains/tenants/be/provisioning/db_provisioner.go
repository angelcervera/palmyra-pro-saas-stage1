package provisioning

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// DBProvisioner creates per-tenant roles/schemas/grants and base shared tables.
type DBProvisioner struct {
	pool *pgxpool.Pool
}

func NewDBProvisioner(pool *pgxpool.Pool) *DBProvisioner {
	if pool == nil {
		panic("db provisioner requires pool")
	}
	return &DBProvisioner{pool: pool}
}

func (p *DBProvisioner) Ensure(ctx context.Context, req service.DBProvisionRequest) (service.DBProvisionResult, error) {
	ready, err := p.ensureRoleSchemaAndGrants(ctx, req)
	if err != nil {
		return service.DBProvisionResult{}, err
	}
	if err := p.ensureBaseTables(ctx, req); err != nil {
		return service.DBProvisionResult{}, err
	}
	return service.DBProvisionResult{Ready: ready}, nil
}

func (p *DBProvisioner) Check(ctx context.Context, req service.DBProvisionRequest) (service.DBProvisionResult, error) {
	if req.RoleName == "" || req.SchemaName == "" {
		return service.DBProvisionResult{Ready: false}, fmt.Errorf("role and schema required")
	}
	if req.AdminSchema == "" {
		return service.DBProvisionResult{Ready: false}, fmt.Errorf("admin schema required")
	}

	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return service.DBProvisionResult{}, fmt.Errorf("acquire conn: %w", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return service.DBProvisionResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	// Verify the tenant role exists before switching context.
	var dummy int
	if err := tx.QueryRow(ctx, "SELECT 1 FROM pg_roles WHERE rolname = $1", req.RoleName).Scan(&dummy); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.DBProvisionResult{Ready: false}, nil
		}
		return service.DBProvisionResult{}, fmt.Errorf("check role: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return service.DBProvisionResult{}, fmt.Errorf("commit role check: %w", err)
	}

	ready := true
	tenantDB := persistence.NewTenantDB(persistence.TenantDBConfig{
		Pool:        p.pool,
		AdminSchema: req.AdminSchema,
	})

	if err := tenantDB.WithTenant(ctx, tenant.Space{
		SchemaName: req.SchemaName,
		RoleName:   req.RoleName,
	}, func(txx pgx.Tx) error {
		// Check schema visibility under tenant role.
		if err := txx.QueryRow(ctx, "SELECT 1 FROM information_schema.schemata WHERE schema_name = $1", req.SchemaName).Scan(&dummy); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				ready = false
				return nil
			}
			return fmt.Errorf("check schema: %w", err)
		}
		// Confirm users table is registered.
		if err := txx.QueryRow(ctx, "SELECT to_regclass('users')").Scan(&dummy); err != nil {
			return fmt.Errorf("check users regclass: %w", err)
		}
		if dummy == 0 {
			ready = false
			return nil
		}
		// Basic read probe to ensure SELECT privilege.
		if err := txx.QueryRow(ctx, "SELECT 1 FROM users LIMIT 1").Scan(&dummy); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("read users table: %w", err)
		}
		return nil
	}); err != nil {
		return service.DBProvisionResult{}, err
	}

	return service.DBProvisionResult{Ready: ready}, nil
}

func (p *DBProvisioner) ensureRoleSchemaAndGrants(ctx context.Context, req service.DBProvisionRequest) (bool, error) {
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("acquire conn: %w", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	roleSQL := fmt.Sprintf("CREATE ROLE %s NOLOGIN", pgx.Identifier{req.RoleName}.Sanitize())
	if _, err := tx.Exec(ctx, roleSQL); err != nil {
		// ignore duplicate role
	}

	if _, err := tx.Exec(ctx, fmt.Sprintf("GRANT %s TO CURRENT_USER", pgx.Identifier{req.RoleName}.Sanitize())); err != nil {
		return false, fmt.Errorf("grant role to app user: %w", err)
	}

	createSchema := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s AUTHORIZATION %s", pgx.Identifier{req.SchemaName}.Sanitize(), pgx.Identifier{req.RoleName}.Sanitize())
	if _, err := tx.Exec(ctx, createSchema); err != nil {
		return false, fmt.Errorf("create schema: %w", err)
	}

	grantUsageTenant := fmt.Sprintf("GRANT USAGE ON SCHEMA %s TO %s", pgx.Identifier{req.SchemaName}.Sanitize(), pgx.Identifier{req.RoleName}.Sanitize())
	if _, err := tx.Exec(ctx, grantUsageTenant); err != nil {
		return false, fmt.Errorf("grant usage tenant schema: %w", err)
	}

	if req.AdminSchema != "" {
		grantUsageAdmin := fmt.Sprintf("GRANT USAGE ON SCHEMA %s TO %s", pgx.Identifier{req.AdminSchema}.Sanitize(), pgx.Identifier{req.RoleName}.Sanitize())
		if _, err := tx.Exec(ctx, grantUsageAdmin); err != nil {
			return false, fmt.Errorf("grant usage admin schema: %w", err)
		}
		grantSchemaRepo := fmt.Sprintf("GRANT SELECT ON %s.schema_repository TO %s", pgx.Identifier{req.AdminSchema}.Sanitize(), pgx.Identifier{req.RoleName}.Sanitize())
		if _, err := tx.Exec(ctx, grantSchemaRepo); err != nil {
			return false, fmt.Errorf("grant select schema_repository: %w", err)
		}
	}

	alterDefault := fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON TABLES TO %s", pgx.Identifier{req.SchemaName}.Sanitize(), pgx.Identifier{req.RoleName}.Sanitize())
	if _, err := tx.Exec(ctx, alterDefault); err != nil {
		return false, fmt.Errorf("default privs tables: %w", err)
	}
	alterDefaultSeq := fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON SEQUENCES TO %s", pgx.Identifier{req.SchemaName}.Sanitize(), pgx.Identifier{req.RoleName}.Sanitize())
	if _, err := tx.Exec(ctx, alterDefaultSeq); err != nil {
		return false, fmt.Errorf("default privs sequences: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit: %w", err)
	}
	return true, nil
}

func (p *DBProvisioner) ensureBaseTables(ctx context.Context, req service.DBProvisionRequest) error {
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire conn: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, fmt.Sprintf("SET ROLE %s", pgx.Identifier{req.RoleName}.Sanitize())); err != nil {
		return fmt.Errorf("set role: %w", err)
	}
	if req.AdminSchema != "" {
		if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path = %s, %s", pgx.Identifier{req.SchemaName}.Sanitize(), pgx.Identifier{req.AdminSchema}.Sanitize())); err != nil {
			return fmt.Errorf("set search_path: %w", err)
		}
	} else {
		if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path = %s", pgx.Identifier{req.SchemaName}.Sanitize())); err != nil {
			return fmt.Errorf("set search_path: %w", err)
		}
	}

	stmt := `
CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    full_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS users_created_at_idx ON users(created_at DESC);
`
	if _, err := conn.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("ensure base users table: %w", err)
	}

	return nil
}

var _ service.DBProvisioner = (*DBProvisioner)(nil)
