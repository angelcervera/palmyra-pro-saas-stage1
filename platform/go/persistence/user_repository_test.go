package persistence

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

func TestUserStoreIsolationWithTenantDB(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping user store integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("palmyra"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(2*time.Minute)),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = pgContainer.Terminate(context.Background())
	})

	adminSchema := "tenant_admin"
	connString, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	connString = fmt.Sprintf("%s&search_path=%s", connString, adminSchema)

	pool, err := NewPool(ctx, PoolConfig{ConnString: connString})
	require.NoError(t, err)
	t.Cleanup(func() { ClosePool(pool) })

	// Ensure schemas exist.
	_, err = pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+adminSchema)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+adminSchema+`.schema_repository (schema_id UUID, schema_version TEXT, PRIMARY KEY (schema_id, schema_version))`)
	require.NoError(t, err)
	tenantSchemaA := tenant.BuildSchemaName("dev", "acme_co")
	tenantSchemaB := tenant.BuildSchemaName("dev", "beta_inc")
	_, err = pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+tenantSchemaA)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+tenantSchemaB)
	require.NoError(t, err)
	// Create tenant roles for tests and grant to current user.
	createRole := func(role string) {
		_, err = pool.Exec(ctx, `
DO $$
BEGIN
   IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '`+role+`') THEN
      CREATE ROLE `+role+` NOLOGIN;
   END IF;
END$$;`)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `GRANT `+role+` TO CURRENT_USER`)
		require.NoError(t, err)
	}
	createRole(tenantSchemaA + `_role`)
	createRole(tenantSchemaB + `_role`)
	_, err = pool.Exec(ctx, `ALTER SCHEMA `+tenantSchemaA+` OWNER TO `+tenantSchemaA+`_role`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `ALTER SCHEMA `+tenantSchemaB+` OWNER TO `+tenantSchemaB+`_role`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `GRANT USAGE ON SCHEMA `+adminSchema+` TO `+tenantSchemaA+`_role`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `GRANT USAGE ON SCHEMA `+adminSchema+` TO `+tenantSchemaB+`_role`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `GRANT SELECT ON `+adminSchema+`.schema_repository TO `+tenantSchemaA+`_role`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `GRANT SELECT ON `+adminSchema+`.schema_repository TO `+tenantSchemaB+`_role`)
	require.NoError(t, err)

	tenantDB := NewTenantDB(TenantDBConfig{
		Pool:        pool,
		AdminSchema: adminSchema,
	})

	store, err := NewUserStore(ctx, tenantDB)
	require.NoError(t, err)

	spaceA := tenant.Space{
		TenantID:      uuid.New(),
		Slug:          "acme-co",
		ShortTenantID: "acme0001",
		SchemaName:    tenantSchemaA,
		RoleName:      tenantSchemaA + "_role",
		BasePrefix:    "dev/acme-co-acme0001/",
	}
	spaceB := tenant.Space{
		TenantID:      uuid.New(),
		Slug:          "beta-inc",
		ShortTenantID: "beta0001",
		SchemaName:    tenantSchemaB,
		RoleName:      tenantSchemaB + "_role",
		BasePrefix:    "dev/beta-inc-beta0001/",
	}

	userA, err := store.CreateUser(ctx, spaceA, CreateUserParams{
		UserID:   uuid.New(),
		Email:    "a@example.com",
		FullName: "Alice A",
	})
	require.NoError(t, err)

	userB, err := store.CreateUser(ctx, spaceB, CreateUserParams{
		UserID:   uuid.New(),
		Email:    "b@example.com",
		FullName: "Bob B",
	})
	require.NoError(t, err)

	assertCount := func(schema string, expected int) {
		var count int
		err := pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s.users`, schema)).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, expected, count)
	}
	assertCount(tenantSchemaA, 1)
	assertCount(tenantSchemaB, 1)

	// Cross-tenant read should not find the other user's row.
	_, err = store.GetUser(ctx, spaceB, userA.UserID)
	require.ErrorIs(t, err, ErrUserNotFound)

	// Update in A should not affect B.
	updated, err := store.UpdateUser(ctx, spaceA, userA.UserID, UpdateUserParams{
		FullName: strPtrUser("Alice Updated"),
	})
	require.NoError(t, err)
	require.Equal(t, "Alice Updated", updated.FullName)
	assertCount(tenantSchemaA, 1)
	assertCount(tenantSchemaB, 1)

	// Delete in B keeps A untouched.
	err = store.DeleteUser(ctx, spaceB, userB.UserID)
	require.NoError(t, err)
	assertCount(tenantSchemaA, 1)
	assertCount(tenantSchemaB, 0)
}

func strPtrUser(s string) *string { return &s }
