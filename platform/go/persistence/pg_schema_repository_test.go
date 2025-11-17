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

func TestSchemaRepositoryStoreIntegration(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping persistence integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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

	store, err := NewSchemaRepositoryStore(ctx, pool)
	require.NoError(t, err)

	categoryStore, err := NewSchemaCategoryStore(ctx, pool)
	require.NoError(t, err)

	schemaID := uuid.New()
	rootCategoryID := uuid.New()
	childCategoryID := uuid.New()
	rootDescription := "System schemas"

	rootCategory, err := categoryStore.CreateSchemaCategory(ctx, CreateSchemaCategoryParams{
		CategoryID: rootCategoryID,
		Name:       "root",
		Slug:       "system-schemas",
		Description: func() *string {
			return &rootDescription
		}(),
	})
	require.NoError(t, err)
	require.Equal(t, "root", rootCategory.Name)
	require.Equal(t, "system-schemas", rootCategory.Slug)
	require.Nil(t, rootCategory.ParentCategoryID)

	childCategory, err := categoryStore.CreateSchemaCategory(ctx, CreateSchemaCategoryParams{
		CategoryID:       childCategoryID,
		ParentCategoryID: &rootCategoryID,
		Name:             "cards",
		Slug:             "cards",
	})
	require.NoError(t, err)
	require.NotNil(t, childCategory.ParentCategoryID)
	require.Equal(t, rootCategoryID, *childCategory.ParentCategoryID)
	require.Equal(t, "cards", childCategory.Slug)

	categories, err := categoryStore.ListSchemaCategories(ctx, false)
	require.NoError(t, err)
	require.Len(t, categories, 2)

	defV1 := SchemaDefinition(`{"title":"schema-v1"}`)
	versionV1 := SemanticVersion{Major: 1, Minor: 0, Patch: 0}

	recordV1, err := store.CreateOrUpdateSchema(ctx, CreateSchemaParams{
		SchemaID:   schemaID,
		Version:    versionV1,
		Definition: defV1,
		TableName:  "cards_entities",
		Slug:       "cards-schema",
		CategoryID: childCategoryID,
		Activate:   true,
	})
	require.NoError(t, err)
	require.True(t, recordV1.IsActive)
	require.Equal(t, "cards_entities", recordV1.TableName)
	require.Equal(t, "cards-schema", recordV1.Slug)
	require.Equal(t, childCategoryID, recordV1.CategoryID)

	gotV1, err := store.GetSchemaByVersion(ctx, schemaID, versionV1)
	require.NoError(t, err)
	require.JSONEq(t, string(defV1), string(gotV1.SchemaDefinition))
	require.True(t, gotV1.IsActive)

	active, err := store.GetActiveSchema(ctx, schemaID)
	require.NoError(t, err)
	require.Equal(t, versionV1, active.SchemaVersion)

	byTable, err := store.GetActiveSchemaByTableName(ctx, "cards_entities")
	require.NoError(t, err)
	require.Equal(t, active.SchemaID, byTable.SchemaID)
	require.Equal(t, "cards_entities", byTable.TableName)

	defV2 := SchemaDefinition(`{"title":"schema-v2"}`)
	versionV2 := SemanticVersion{Major: 1, Minor: 1, Patch: 0}
	recordV2, err := store.CreateOrUpdateSchema(ctx, CreateSchemaParams{
		SchemaID:   schemaID,
		Version:    versionV2,
		Definition: defV2,
		TableName:  "",
		Slug:       "cards-schema",
		CategoryID: childCategoryID,
		Activate:   true,
	})
	require.NoError(t, err)
	require.True(t, recordV2.IsActive)
	require.Equal(t, "cards_entities", recordV2.TableName)
	require.Equal(t, "cards-schema", recordV2.Slug)

	// version 1 should have been deactivated when v2 became active
	gotV1, err = store.GetSchemaByVersion(ctx, schemaID, versionV1)
	require.NoError(t, err)
	require.False(t, gotV1.IsActive)

	active, err = store.GetActiveSchema(ctx, schemaID)
	require.NoError(t, err)
	require.Equal(t, versionV2, active.SchemaVersion)

	_, err = store.GetActiveSchemaByTableName(ctx, "missing_entities")
	require.ErrorIs(t, err, ErrSchemaNotFound)

	require.NoError(t, store.ActivateSchemaVersion(ctx, schemaID, versionV1))

	active, err = store.GetActiveSchema(ctx, schemaID)
	require.NoError(t, err)
	require.Equal(t, versionV1, active.SchemaVersion)

	deleteTime := time.Now().UTC()
	require.NoError(t, store.SoftDeleteSchema(ctx, schemaID, versionV1, deleteTime))

	_, err = store.GetSchemaByVersion(ctx, schemaID, versionV1)
	require.ErrorIs(t, err, ErrSchemaNotFound)

	require.NoError(t, store.ActivateSchemaVersion(ctx, schemaID, versionV2))

	active, err = store.GetActiveSchema(ctx, schemaID)
	require.NoError(t, err)
	require.Equal(t, versionV2, active.SchemaVersion)

	records, err := store.ListSchemas(ctx, schemaID)
	require.NoError(t, err)
	require.Len(t, records, 2)

	_, err = store.CreateOrUpdateSchema(ctx, CreateSchemaParams{
		SchemaID:   schemaID,
		Version:    SemanticVersion{Major: 2, Minor: 0, Patch: 0},
		Definition: SchemaDefinition(`{"title":"schema-v3"}`),
		TableName:  "other_table",
		Slug:       "cards-schema",
		CategoryID: childCategoryID,
		Activate:   false,
	})
	require.Error(t, err)

	require.NoError(t, categoryStore.SoftDeleteSchemaCategory(ctx, rootCategoryID, time.Now().UTC()))

	_, err = categoryStore.GetSchemaCategory(ctx, rootCategoryID)
	require.ErrorIs(t, err, ErrSchemaNotFound)
}
