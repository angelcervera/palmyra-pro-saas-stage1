package repo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
)

// Repository exposes persistence operations for schema repository records.
type Repository interface {
	Upsert(ctx context.Context, params persistence.CreateSchemaParams) (persistence.SchemaRecord, error)
	GetByVersion(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion) (persistence.SchemaRecord, error)
	GetActive(ctx context.Context, schemaID uuid.UUID) (persistence.SchemaRecord, error)
	List(ctx context.Context, schemaID uuid.UUID) ([]persistence.SchemaRecord, error)
	ListAll(ctx context.Context, includeInactive bool) ([]persistence.SchemaRecord, error)
	GetLatestBySlug(ctx context.Context, slug string) (persistence.SchemaRecord, error)
	Activate(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion) error
	Delete(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion, deletedAt time.Time) error
}

type postgresRepository struct {
	spaceDB *persistence.SpaceDB
	store   *persistence.SchemaRepositoryStore
}

// NewPostgresRepository constructs a Repository backed by the shared persistence layer.
func NewPostgresRepository(spaceDB *persistence.SpaceDB, store *persistence.SchemaRepositoryStore) Repository {
	if spaceDB == nil {
		panic("admin db is required")
	}
	if store == nil {
		panic("schema repository store is required")
	}
	return &postgresRepository{spaceDB: spaceDB, store: store}
}

func (r *postgresRepository) Upsert(ctx context.Context, params persistence.CreateSchemaParams) (persistence.SchemaRecord, error) {
	return r.store.CreateOrUpdateSchema(ctx, r.spaceDB, params)
}

func (r *postgresRepository) GetByVersion(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion) (persistence.SchemaRecord, error) {
	return r.store.GetSchemaByVersion(ctx, r.spaceDB, schemaID, version)
}

func (r *postgresRepository) GetActive(ctx context.Context, schemaID uuid.UUID) (persistence.SchemaRecord, error) {
	return r.store.GetActiveSchema(ctx, r.spaceDB, schemaID)
}

func (r *postgresRepository) List(ctx context.Context, schemaID uuid.UUID) ([]persistence.SchemaRecord, error) {
	return r.store.ListSchemas(ctx, r.spaceDB, schemaID)
}

func (r *postgresRepository) ListAll(ctx context.Context, includeInactive bool) ([]persistence.SchemaRecord, error) {
	return r.store.ListAllSchemaVersions(ctx, r.spaceDB, includeInactive)
}

func (r *postgresRepository) GetLatestBySlug(ctx context.Context, slug string) (persistence.SchemaRecord, error) {
	return r.store.GetLatestSchemaBySlug(ctx, r.spaceDB, slug)
}

func (r *postgresRepository) Activate(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion) error {
	return r.store.ActivateSchemaVersion(ctx, r.spaceDB, schemaID, version)
}

func (r *postgresRepository) Delete(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion, deletedAt time.Time) error {
	return r.store.DeleteSchema(ctx, r.spaceDB, schemaID, version, deletedAt)
}
