package provisioning

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	sqlassets "github.com/zenGate-Global/palmyra-pro-saas/database"
	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// DBProvisioner creates per-tenant roles/schemas/grants and base shared tables.
type DBProvisioner struct {
	pool        *pgxpool.Pool
	spaceDB     *persistence.SpaceDB
	adminSchema string
}

func NewDBProvisioner(pool *pgxpool.Pool, adminSchema string) *DBProvisioner {
	if pool == nil {
		panic("db provisioner requires pool")
	}

	adminSchema = strings.TrimSpace(adminSchema)
	if adminSchema == "" {
		panic("db provisioner requires admin schema")
	}

	return &DBProvisioner{
		pool:        pool,
		adminSchema: adminSchema,
		spaceDB: persistence.NewSpaceDB(persistence.SpaceDBConfig{
			Pool:        pool,
			AdminSchema: adminSchema,
		}),
	}
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

	// Ensure current user can assume the tenant role; otherwise SET ROLE will fail later.
	if err := tx.QueryRow(ctx, "SELECT pg_has_role(current_user, $1, 'MEMBER')::int", req.RoleName).Scan(&dummy); err != nil {
		return service.DBProvisionResult{}, fmt.Errorf("check role membership: %w", err)
	}
	if dummy == 0 {
		return service.DBProvisionResult{Ready: false}, nil
	}

	if err := tx.Commit(ctx); err != nil {
		return service.DBProvisionResult{}, fmt.Errorf("commit role check: %w", err)
	}

	ready := true

	if err := p.spaceDB.WithTenant(ctx, tenant.Space{
		SchemaName: req.SchemaName,
		RoleName:   req.RoleName,
	}, func(txx pgx.Tx) error {
		// Check schema visibility under tenant role.
		var dummy int
		if err := txx.QueryRow(ctx, "SELECT 1 FROM information_schema.schemata WHERE schema_name = $1", req.SchemaName).Scan(&dummy); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				ready = false
				return nil
			}
			return fmt.Errorf("check schema: %w", err)
		}
		// Confirm users table is registered.
		var usersExists bool
		if err := txx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_class c
				JOIN pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname = $1 AND c.relname = 'users'
			)`, req.SchemaName).Scan(&usersExists); err != nil {
			return fmt.Errorf("check users regclass: %w", err)
		}
		if !usersExists {
			ready = false
			return nil
		}
		// Basic read probe to ensure SELECT privilege.
		if err := txx.QueryRow(ctx, "SELECT 1 FROM "+pgx.Identifier{req.SchemaName, "users"}.Sanitize()+" LIMIT 1").Scan(&dummy); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("read users table: %w", err)
		}
		// Ensure read access to shared catalog tables in admin schema via search_path.
		var hasSchemaRepo bool
		if err := txx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_class c
				JOIN pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname = $1 AND c.relname = 'schema_repository'
			)`, p.adminSchema).Scan(&hasSchemaRepo); err != nil {
			return fmt.Errorf("check schema_repository regclass: %w", err)
		}
		if !hasSchemaRepo {
			ready = false
			return nil
		}
		if err := txx.QueryRow(ctx, "SELECT 1 FROM "+pgx.Identifier{p.adminSchema, "schema_repository"}.Sanitize()+" LIMIT 1").Scan(&dummy); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("read schema_repository: %w", err)
		}
		var hasSchemaCategories bool
		if err := txx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_class c
				JOIN pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname = $1 AND c.relname = 'schema_categories'
			)`, p.adminSchema).Scan(&hasSchemaCategories); err != nil {
			return fmt.Errorf("check schema_categories regclass: %w", err)
		}
		if !hasSchemaCategories {
			ready = false
			return nil
		}
		if err := txx.QueryRow(ctx, "SELECT 1 FROM "+pgx.Identifier{p.adminSchema, "schema_categories"}.Sanitize()+" LIMIT 1").Scan(&dummy); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("read schema_categories: %w", err)
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

	// Create tenant role only if missing to avoid aborting the transaction.
	var roleExists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", req.RoleName).Scan(&roleExists); err != nil {
		return false, fmt.Errorf("check role existence: %w", err)
	}
	if !roleExists {
		roleSQL := fmt.Sprintf("CREATE ROLE %s NOLOGIN", pgx.Identifier{req.RoleName}.Sanitize())
		if _, err := tx.Exec(ctx, roleSQL); err != nil {
			return false, fmt.Errorf("create role: %w", err)
		}
	}

	createSchema := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s AUTHORIZATION %s", pgx.Identifier{req.SchemaName}.Sanitize(), pgx.Identifier{req.RoleName}.Sanitize())
	if _, err := tx.Exec(ctx, createSchema); err != nil {
		return false, fmt.Errorf("create schema: %w", err)
	}

	// Ensure the application role can assume the tenant role to execute SET ROLE in SpaceDB.
	if _, err := tx.Exec(ctx, fmt.Sprintf("GRANT %s TO CURRENT_USER", pgx.Identifier{req.RoleName}.Sanitize())); err != nil {
		return false, fmt.Errorf("grant tenant role to app user: %w", err)
	}

	grantUsageTenant := fmt.Sprintf("GRANT USAGE ON SCHEMA %s TO %s", pgx.Identifier{req.SchemaName}.Sanitize(), pgx.Identifier{req.RoleName}.Sanitize())
	if _, err := tx.Exec(ctx, grantUsageTenant); err != nil {
		return false, fmt.Errorf("grant usage tenant schema: %w", err)
	}

	grantUsageAdmin := fmt.Sprintf("GRANT USAGE ON SCHEMA %s TO %s", pgx.Identifier{p.adminSchema}.Sanitize(), pgx.Identifier{req.RoleName}.Sanitize())
	if _, err := tx.Exec(ctx, grantUsageAdmin); err != nil {
		return false, fmt.Errorf("grant usage admin schema: %w", err)
	}
	for _, table := range []string{"schema_repository", "schema_categories"} { // future catalog tables must be added here
		selectGrant := fmt.Sprintf("GRANT SELECT ON %s.%s TO %s", pgx.Identifier{p.adminSchema}.Sanitize(), pgx.Identifier{table}.Sanitize(), pgx.Identifier{req.RoleName}.Sanitize())
		if _, err := tx.Exec(ctx, selectGrant); err != nil {
			return false, fmt.Errorf("grant select %s: %w", table, err)
		}
		// Needed to create FKs pointing at schema_repository from tenant tables.
		referencesGrant := fmt.Sprintf("GRANT REFERENCES ON %s.%s TO %s", pgx.Identifier{p.adminSchema}.Sanitize(), pgx.Identifier{table}.Sanitize(), pgx.Identifier{req.RoleName}.Sanitize())
		if _, err := tx.Exec(ctx, referencesGrant); err != nil {
			return false, fmt.Errorf("grant references %s: %w", table, err)
		}
	}

	// Apply default privileges while scoped to the tenant role but contained within the same admin-owned transaction.
	setRole := fmt.Sprintf("SET LOCAL ROLE %s", pgx.Identifier{req.RoleName}.Sanitize())
	if _, err := tx.Exec(ctx, setRole); err != nil {
		return false, fmt.Errorf("set local role: %w", err)
	}
	searchPath := fmt.Sprintf("%s, %s", pgx.Identifier{req.SchemaName}.Sanitize(), pgx.Identifier{p.adminSchema}.Sanitize())
	if _, err := tx.Exec(ctx, `SELECT set_config('search_path', $1, true)`, searchPath); err != nil {
		return false, fmt.Errorf("set search_path: %w", err)
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
	return p.spaceDB.WithTenant(ctx, tenant.Space{
		SchemaName: req.SchemaName,
		RoleName:   req.RoleName,
	}, func(tx pgx.Tx) error {
		// If the users table already exists (e.g., created by init SQL), skip creation.
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_class c
				JOIN pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname = $1 AND c.relname = 'users'
			)`, req.SchemaName).Scan(&exists); err != nil {
			return fmt.Errorf("check users table: %w", err)
		}
		if exists {
			return nil
		}
		if _, err := tx.Exec(ctx, sqlassets.UsersSQL); err != nil {
			return fmt.Errorf("ensure base users table: %w", err)
		}
		return nil
	})
}

var _ service.DBProvisioner = (*DBProvisioner)(nil)
