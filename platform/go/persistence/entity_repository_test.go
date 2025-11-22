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

func TestEntityRepositoryIsolationWithTenantDB(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping entity repository integration test in short mode")
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
	t.Cleanup(func() {
		ClosePool(pool)
	})

	require.NoError(t, applyDDLToSchema(ctx, pool, adminSchema, "001_core_schema.sql"))
	require.NoError(t, applyDDLToSchema(ctx, pool, adminSchema, "002_tenants_schema.sql"))

	tenantSchemaA := tenant.BuildSchemaName("acme_co")
	tenantSchemaB := tenant.BuildSchemaName("beta_inc")
	_, err = pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+tenantSchemaA)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+tenantSchemaB)
	require.NoError(t, err)

	schemaStore, err := NewSchemaRepositoryStore(ctx, pool)
	require.NoError(t, err)

	categoryStore, err := NewSchemaCategoryStore(ctx, pool)
	require.NoError(t, err)

	schemaID := uuid.New()
	categoryID := uuid.New()

	_, err = categoryStore.CreateSchemaCategory(ctx, CreateSchemaCategoryParams{
		CategoryID: categoryID,
		Name:       "cards",
		Slug:       "cards",
	})
	require.NoError(t, err)

	baseSchema := SchemaDefinition([]byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": { "type": "string" },
			"rarity": { "type": "string" }
		},
		"required": ["name"],
		"additionalProperties": false
	}`))

	version := SemanticVersion{Major: 1, Minor: 0, Patch: 0}
	_, err = schemaStore.CreateOrUpdateSchema(ctx, CreateSchemaParams{
		SchemaID:   schemaID,
		Version:    version,
		Definition: baseSchema,
		TableName:  "cards_entities",
		Slug:       "cards-schema",
		CategoryID: categoryID,
		Activate:   true,
	})
	require.NoError(t, err)

	validator := NewSchemaValidator()
	tenantDB := NewTenantDB(TenantDBConfig{
		Pool:        pool,
		AdminSchema: adminSchema,
	})

	entityRepo, err := NewEntityRepository(ctx, tenantDB, schemaStore, validator, EntityRepositoryConfig{
		SchemaID: schemaID,
	})
	require.NoError(t, err)

	spaceA := tenant.Space{
		TenantID:      uuid.New(),
		Slug:          "acme-co",
		ShortTenantID: "acme0001",
		SchemaName:    tenantSchemaA,
		BasePrefix:    "dev/acme-co-acme0001/",
	}
	spaceB := tenant.Space{
		TenantID:      uuid.New(),
		Slug:          "beta-inc",
		ShortTenantID: "beta0001",
		SchemaName:    tenantSchemaB,
		BasePrefix:    "dev/beta-inc-beta0001/",
	}

	// Tenant A create
	createPayload := SchemaDefinition([]byte(`{"name":"Black Lotus"}`))
	createdA, err := entityRepo.CreateEntity(ctx, spaceA, CreateEntityParams{
		Payload: createPayload,
	})
	require.NoError(t, err)

	// Tenant B create
	createPayloadB := SchemaDefinition([]byte(`{"name":"Time Walk"}`))
	createdB, err := entityRepo.CreateEntity(ctx, spaceB, CreateEntityParams{
		Payload: createPayloadB,
	})
	require.NoError(t, err)

	// Verify isolation via raw queries
	assertCount := func(schema string, expected int) {
		var count int
		err := pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s.cards_entities`, schema)).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, expected, count)
	}
	assertCount(tenantSchemaA, 1)
	assertCount(tenantSchemaB, 1)

	// List per tenant should only see its own records.
	listA, err := entityRepo.ListEntities(ctx, spaceA, ListEntitiesParams{
		OnlyActive:     true,
		IncludeDeleted: false,
		Limit:          10,
	})
	require.NoError(t, err)
	require.Len(t, listA, 1)
	require.Equal(t, createdA.EntityID, listA[0].EntityID)

	listB, err := entityRepo.ListEntities(ctx, spaceB, ListEntitiesParams{
		OnlyActive:     true,
		IncludeDeleted: false,
		Limit:          10,
	})
	require.NoError(t, err)
	require.Len(t, listB, 1)
	require.Equal(t, createdB.EntityID, listB[0].EntityID)

	_, err = entityRepo.GetEntityByID(ctx, spaceB, createdA.EntityID)
	require.ErrorIs(t, err, ErrEntityNotFound)

	// Update in tenant A should not affect tenant B.
	updatePayload := SchemaDefinition([]byte(`{"name":"Black Lotus","rarity":"mythic"}`))
	updatedA, err := entityRepo.UpdateEntity(ctx, spaceA, UpdateEntityParams{
		EntityID: createdA.EntityID,
		Payload:  updatePayload,
	})
	require.NoError(t, err)
	require.Equal(t, createdA.EntityVersion.NextPatch(), updatedA.EntityVersion)

	assertCount(tenantSchemaA, 2) // new version inserted; old still present
	assertCount(tenantSchemaB, 1)
}

func TestSanitizeEntitySort(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		order     string
		wantField string
		wantOrder string
		wantErr   bool
	}{
		{name: "defaults", field: "", order: "", wantField: "created_at", wantOrder: "DESC"},
		{name: "asc", field: "created_at", order: "asc", wantField: "created_at", wantOrder: "ASC"},
		{name: "invalid-field", field: "DROP", order: "asc", wantErr: true},
		{name: "invalid-order", field: "created_at", order: "sideways", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, order, err := sanitizeEntitySort(tt.field, tt.order)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantField, field)
			require.Equal(t, tt.wantOrder, order)
		})
	}
}
