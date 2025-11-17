package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestEntityRepositoryIntegration(t *testing.T) {
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

	connString, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := NewPool(ctx, PoolConfig{ConnString: connString})
	require.NoError(t, err)
	t.Cleanup(func() {
		ClosePool(pool)
	})

	require.NoError(t, applyCoreSchemaDDL(ctx, pool))

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
	entityRepo, err := NewEntityRepository(ctx, pool, schemaStore, validator, EntityRepositoryConfig{
		SchemaID: schemaID,
	})
	require.NoError(t, err)

	createPayload := SchemaDefinition([]byte(`{"name":"Black Lotus"}`))
	created, err := entityRepo.CreateEntity(ctx, CreateEntityParams{
		Slug:    "black-lotus",
		Payload: createPayload,
	})
	require.NoError(t, err)
	require.Equal(t, SemanticVersion{Major: 1, Minor: 0, Patch: 0}, created.EntityVersion)
	require.True(t, created.IsActive)
	require.Equal(t, "black-lotus", created.Slug)
	require.False(t, created.IsSoftDeleted)

	fetched, err := entityRepo.GetEntityByID(ctx, created.EntityID)
	require.NoError(t, err)
	require.Equal(t, created.EntityVersion, fetched.EntityVersion)
	require.Equal(t, "black-lotus", fetched.Slug)

	updatePayload := SchemaDefinition([]byte(`{"name":"Black Lotus","rarity":"mythic"}`))
	updated, err := entityRepo.UpdateEntity(ctx, UpdateEntityParams{
		EntityID: created.EntityID,
		Payload:  updatePayload,
	})
	require.NoError(t, err)
	require.Equal(t, created.EntityVersion.NextPatch(), updated.EntityVersion)
	require.True(t, updated.IsActive)
	require.Equal(t, "black-lotus", updated.Slug)
	require.False(t, updated.IsSoftDeleted)

	oldVersion, err := entityRepo.GetEntityVersion(ctx, created.EntityID, created.EntityVersion)
	require.NoError(t, err)
	require.False(t, oldVersion.IsActive)

	list, err := entityRepo.ListEntities(ctx, ListEntitiesParams{
		OnlyActive:     true,
		IncludeDeleted: false,
		Limit:          10,
		Offset:         0,
		SortField:      "created_at",
		SortOrder:      "desc",
	})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, updated.EntityID, list[0].EntityID)
	require.Equal(t, updated.EntityVersion, list[0].EntityVersion)
	require.Equal(t, "black-lotus", list[0].Slug)

	total, err := entityRepo.CountEntities(ctx, ListEntitiesParams{
		OnlyActive:     true,
		IncludeDeleted: false,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)

	err = entityRepo.SoftDeleteEntity(ctx, created.EntityID, time.Now().UTC())
	require.NoError(t, err)

	_, err = entityRepo.GetEntityByID(ctx, created.EntityID)
	require.ErrorIs(t, err, ErrEntityNotFound)

	records, err := entityRepo.ListEntities(ctx, ListEntitiesParams{
		OnlyActive:     true,
		IncludeDeleted: false,
		Limit:          10,
		SortField:      "created_at",
		SortOrder:      "desc",
	})
	require.NoError(t, err)
	require.Len(t, records, 0)

	totalAfterDelete, err := entityRepo.CountEntities(ctx, ListEntitiesParams{
		OnlyActive:     true,
		IncludeDeleted: false,
	})
	require.NoError(t, err)
	require.EqualValues(t, 0, totalAfterDelete)

	deletedRecords, err := entityRepo.ListEntities(ctx, ListEntitiesParams{
		OnlyActive:     false,
		IncludeDeleted: true,
		Limit:          10,
		SortField:      "created_at",
		SortOrder:      "desc",
	})
	require.NoError(t, err)
	require.NotEmpty(t, deletedRecords)
	require.True(t, deletedRecords[0].IsSoftDeleted)

	_, err = entityRepo.CreateEntity(ctx, CreateEntityParams{
		Slug:    "no-name",
		Payload: SchemaDefinition([]byte(`{"rarity":"rare"}`)),
	})
	require.Error(t, err)
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
		{name: "desc", field: "slug", order: "desc", wantField: "slug", wantOrder: "DESC"},
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
