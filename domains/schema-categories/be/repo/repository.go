package repo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
)

// Repository exposes persistence operations required by the schema categories service.
type Repository interface {
	List(ctx context.Context, includeDeleted bool) ([]persistence.SchemaCategory, error)
	Create(ctx context.Context, params persistence.CreateSchemaCategoryParams) (persistence.SchemaCategory, error)
	Get(ctx context.Context, id uuid.UUID) (persistence.SchemaCategory, error)
	Update(ctx context.Context, id uuid.UUID, params persistence.UpdateSchemaCategoryParams) (persistence.SchemaCategory, error)
	Delete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error
}

type postgresRepository struct {
	adminDB *persistence.SpaceDB
	store   *persistence.SchemaCategoryStore
}

// NewPostgresRepository builds a Repository backed by the shared persistence layer.
func NewPostgresRepository(adminDB *persistence.SpaceDB, store *persistence.SchemaCategoryStore) Repository {
	if adminDB == nil {
		panic("space db is required")
	}
	if store == nil {
		panic("schema category store is required")
	}
	return &postgresRepository{adminDB: adminDB, store: store}
}

func (r *postgresRepository) List(ctx context.Context, includeDeleted bool) ([]persistence.SchemaCategory, error) {
	return r.store.ListSchemaCategories(ctx, r.adminDB, includeDeleted)
}

func (r *postgresRepository) Create(ctx context.Context, params persistence.CreateSchemaCategoryParams) (persistence.SchemaCategory, error) {
	return r.store.CreateSchemaCategory(ctx, r.adminDB, params)
}

func (r *postgresRepository) Get(ctx context.Context, id uuid.UUID) (persistence.SchemaCategory, error) {
	return r.store.GetSchemaCategory(ctx, r.adminDB, id)
}

func (r *postgresRepository) Update(ctx context.Context, id uuid.UUID, params persistence.UpdateSchemaCategoryParams) (persistence.SchemaCategory, error) {
	return r.store.UpdateSchemaCategory(ctx, r.adminDB, id, params)
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	return r.store.DeleteSchemaCategory(ctx, r.adminDB, id, deletedAt)
}
