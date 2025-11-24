package provisioning

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

func singleConnTestPool(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()
	dbURL := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	if url, ok := os.LookupEnv("TEST_DATABASE_URL"); ok && strings.TrimSpace(url) != "" {
		dbURL = url
	}

	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Fatalf("parse pool config: %v", err)
	}
	cfg.MaxConns = 1
	cfg.MinConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}

	if err := persistence.BootstrapAdminSchema(ctx, pool, "tenant_admin"); err != nil {
		pool.Close()
		t.Fatalf("bootstrap admin schema: %v", err)
	}

	cleanup := func() { pool.Close() }
	return pool, cleanup
}

func TestDBProvisionerEnsure_NoLeakAndAccess(t *testing.T) {
	if _, ok := os.LookupEnv("TEST_DATABASE_URL"); !ok {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	pool, cleanup := singleConnTestPool(t)
	defer cleanup()

	schemaName := "tenant_dev_" + strings.ToLower(uuid.New().String()[:8])
	roleName := tenant.BuildRoleName(schemaName)

	prov := NewDBProvisioner(pool, "tenant_admin")

	req := service.DBProvisionRequest{
		TenantID:   uuid.New(),
		SchemaName: schemaName,
		RoleName:   roleName,
	}

	_, err := prov.Ensure(ctx, req)
	require.NoError(t, err)

	// (1) verify connection state is clean after Ensure (search_path doesn't stick to tenant schema)
	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()

	var searchPath string
	require.NoError(t, conn.QueryRow(ctx, "SHOW search_path").Scan(&searchPath))
	require.NotContains(t, searchPath, schemaName)

	var currentRole string
	require.NoError(t, conn.QueryRow(ctx, "SELECT current_role").Scan(&currentRole))
	require.NotEqual(t, roleName, currentRole)

	// (2) tenant role can read admin schema repository tables
	tenantDB := persistence.NewTenantDB(persistence.TenantDBConfig{Pool: pool, AdminSchema: "tenant_admin"})
	err = tenantDB.WithTenant(ctx, tenant.Space{SchemaName: schemaName, RoleName: roleName}, func(tx pgx.Tx) error {
		var c int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM schema_repository").Scan(&c); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM schema_categories").Scan(&c); err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)

	// (3) users table lives in tenant schema and is owned by the tenant role
	var tableSchema, tableOwner string
	err = pool.QueryRow(ctx, `
        SELECT n.nspname, pg_get_userbyid(c.relowner)
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relname = 'users' AND n.nspname = $1
        LIMIT 1`, schemaName).Scan(&tableSchema, &tableOwner)
	require.NoError(t, err)
	require.Equal(t, schemaName, tableSchema)
	require.Equal(t, roleName, tableOwner)
}
