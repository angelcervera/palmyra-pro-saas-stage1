package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
)

// ListParams defines pagination and sorting inputs for listing entities.
type ListParams struct {
	Page       int
	PageSize   int
	SortColumn string
	SortOrder  string
}

// ListResult wraps persistence records with total count metadata.
type ListResult struct {
	Records []persistence.EntityRecord
	Total   int64
}

// Repository exposes entity persistence operations scoped by table name.
type Repository interface {
	List(ctx context.Context, tableName string, params ListParams) (ListResult, error)
	Create(ctx context.Context, tableName string, entityID string, payload json.RawMessage, createdBy *string) (persistence.EntityRecord, error)
	Get(ctx context.Context, tableName string, entityID string) (persistence.EntityRecord, error)
	Update(ctx context.Context, tableName string, entityID string, payload json.RawMessage, createdBy *string) (persistence.EntityRecord, error)
	Delete(ctx context.Context, tableName string, entityID string) error
}

type repository struct {
	pool        *pgxpool.Pool
	schemaStore *persistence.SchemaRepositoryStore
	validator   *persistence.SchemaValidator
}

// New constructs a Repository backed by the shared persistence layer.
func New(pool *pgxpool.Pool, schemaStore *persistence.SchemaRepositoryStore, validator *persistence.SchemaValidator) Repository {
	if pool == nil {
		panic("postgres pool is required")
	}
	if schemaStore == nil {
		panic("schema repository store is required")
	}
	if validator == nil {
		panic("schema validator is required")
	}

	return &repository{pool: pool, schemaStore: schemaStore, validator: validator}
}

func (r *repository) List(ctx context.Context, tableName string, params ListParams) (ListResult, error) {
	repo, err := r.resolveEntityRepo(ctx, tableName)
	if err != nil {
		return ListResult{}, err
	}

	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	listParams := persistence.ListEntitiesParams{
		OnlyActive:     true,
		IncludeDeleted: false,
		Limit:          pageSize,
		Offset:         (page - 1) * pageSize,
		SortField:      params.SortColumn,
		SortOrder:      params.SortOrder,
	}

	records, err := repo.ListEntities(ctx, listParams)
	if err != nil {
		return ListResult{}, err
	}

	total, err := repo.CountEntities(ctx, listParams)
	if err != nil {
		return ListResult{}, err
	}

	return ListResult{Records: records, Total: total}, nil
}

func (r *repository) Create(ctx context.Context, tableName string, entityID string, payload json.RawMessage, createdBy *string) (persistence.EntityRecord, error) {
	repo, err := r.resolveEntityRepo(ctx, tableName)
	if err != nil {
		return persistence.EntityRecord{}, err
	}

	return repo.CreateEntity(ctx, persistence.CreateEntityParams{
		EntityID:  entityID,
		Payload:   payload,
		CreatedBy: createdBy,
	})
}

func (r *repository) Get(ctx context.Context, tableName string, entityID string) (persistence.EntityRecord, error) {
	repo, err := r.resolveEntityRepo(ctx, tableName)
	if err != nil {
		return persistence.EntityRecord{}, err
	}

	return repo.GetEntityByID(ctx, entityID)
}

func (r *repository) Update(ctx context.Context, tableName string, entityID string, payload json.RawMessage, createdBy *string) (persistence.EntityRecord, error) {
	repo, err := r.resolveEntityRepo(ctx, tableName)
	if err != nil {
		return persistence.EntityRecord{}, err
	}

	return repo.UpdateEntity(ctx, persistence.UpdateEntityParams{
		EntityID:  entityID,
		Payload:   payload,
		CreatedBy: createdBy,
	})
}

func (r *repository) Delete(ctx context.Context, tableName string, entityID string) error {
	repo, err := r.resolveEntityRepo(ctx, tableName)
	if err != nil {
		return err
	}

	return repo.SoftDeleteEntity(ctx, entityID, time.Now().UTC())
}

func (r *repository) resolveEntityRepo(ctx context.Context, tableName string) (*persistence.EntityRepository, error) {
	if tableName == "" {
		return nil, errors.New("table name is required")
	}

	schemaRecord, err := r.schemaStore.GetActiveSchemaByTableName(ctx, tableName)
	if err != nil {
		return nil, err
	}

	return persistence.NewEntityRepository(ctx, r.pool, r.schemaStore, r.validator, persistence.EntityRepositoryConfig{
		SchemaID: schemaRecord.SchemaID,
	})
}
