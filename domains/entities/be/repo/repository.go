package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
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
	spaceDB     *persistence.SpaceDB
	schemaStore *persistence.SchemaRepositoryStore
	validator   *persistence.SchemaValidator
}

// New constructs a Repository backed by the shared persistence layer.
func New(spaceDB *persistence.SpaceDB, schemaStore *persistence.SchemaRepositoryStore, validator *persistence.SchemaValidator) Repository {
	if spaceDB == nil {
		panic("space db is required")
	}
	if schemaStore == nil {
		panic("schema repository store is required")
	}
	if validator == nil {
		panic("schema validator is required")
	}

	return &repository{spaceDB: spaceDB, schemaStore: schemaStore, validator: validator}
}

func (r *repository) List(ctx context.Context, tableName string, params ListParams) (ListResult, error) {
	space, err := r.requireTenantSpace(ctx)
	if err != nil {
		return ListResult{}, err
	}

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

	records, err := repo.ListEntities(ctx, space, listParams)
	if err != nil {
		return ListResult{}, err
	}

	total, err := repo.CountEntities(ctx, space, listParams)
	if err != nil {
		return ListResult{}, err
	}

	return ListResult{Records: records, Total: total}, nil
}

func (r *repository) Create(ctx context.Context, tableName string, entityID string, payload json.RawMessage, createdBy *string) (persistence.EntityRecord, error) {
	space, err := r.requireTenantSpace(ctx)
	if err != nil {
		return persistence.EntityRecord{}, err
	}

	repo, err := r.resolveEntityRepo(ctx, tableName)
	if err != nil {
		return persistence.EntityRecord{}, err
	}

	return repo.CreateEntity(ctx, space, persistence.CreateEntityParams{
		EntityID:  entityID,
		Payload:   payload,
		CreatedBy: createdBy,
	})
}

func (r *repository) Get(ctx context.Context, tableName string, entityID string) (persistence.EntityRecord, error) {
	space, err := r.requireTenantSpace(ctx)
	if err != nil {
		return persistence.EntityRecord{}, err
	}

	repo, err := r.resolveEntityRepo(ctx, tableName)
	if err != nil {
		return persistence.EntityRecord{}, err
	}

	return repo.GetEntityByID(ctx, space, entityID)
}

func (r *repository) Update(ctx context.Context, tableName string, entityID string, payload json.RawMessage, createdBy *string) (persistence.EntityRecord, error) {
	space, err := r.requireTenantSpace(ctx)
	if err != nil {
		return persistence.EntityRecord{}, err
	}

	repo, err := r.resolveEntityRepo(ctx, tableName)
	if err != nil {
		return persistence.EntityRecord{}, err
	}

	return repo.UpdateEntity(ctx, space, persistence.UpdateEntityParams{
		EntityID:  entityID,
		Payload:   payload,
		CreatedBy: createdBy,
	})
}

func (r *repository) Delete(ctx context.Context, tableName string, entityID string) error {
	space, err := r.requireTenantSpace(ctx)
	if err != nil {
		return err
	}

	repo, err := r.resolveEntityRepo(ctx, tableName)
	if err != nil {
		return err
	}

	return repo.DeleteEntity(ctx, space, entityID, time.Now().UTC())
}

func (r *repository) resolveEntityRepo(ctx context.Context, tableName string) (*persistence.EntityRepository, error) {
	if tableName == "" {
		return nil, errors.New("table name is required")
	}

	schemaRecord, err := r.schemaStore.GetActiveSchemaByTableName(ctx, r.spaceDB, tableName)
	if err != nil {
		return nil, err
	}

	return persistence.NewEntityRepository(ctx, r.spaceDB, r.schemaStore, r.validator, persistence.EntityRepositoryConfig{
		SchemaID: schemaRecord.SchemaID,
	})
}

func (r *repository) requireTenantSpace(ctx context.Context) (tenant.Space, error) {
	space, ok := tenant.FromContext(ctx)
	if !ok {
		return tenant.Space{}, errors.New("tenant space missing from context")
	}
	return space, nil
}
