package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	domainrepo "github.com/zenGate-Global/palmyra-pro-saas/domains/entities/be/repo"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
)

func TestService_ListSuccess(t *testing.T) {
	ctx := context.Background()
	entityID := uuid.New()
	schemaID := uuid.New()
	createdAt := time.Now().UTC()
	repo := &stubRepository{
		listFn: func(_ context.Context, table string, params domainrepo.ListParams) (domainrepo.ListResult, error) {
			require.Equal(t, "cards_entities", table)
			require.Equal(t, 1, params.Page)
			require.Equal(t, 20, params.PageSize)
			return domainrepo.ListResult{
				Records: []persistence.EntityRecord{{
					EntityID:      entityID,
					SchemaID:      schemaID,
					SchemaVersion: persistence.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
					Payload:       []byte(`{"name":"Lotus"}`),
					CreatedAt:     createdAt,
					IsActive:      true,
					IsSoftDeleted: false,
				}},
				Total: 1,
			}, nil
		},
	}

	svc := New(repo)
	res, err := svc.List(ctx, "cards_entities", ListOptions{Page: 1, PageSize: 20, Sort: "-createdAt"})
	require.NoError(t, err)
	require.Equal(t, 1, res.TotalPages)
	require.Len(t, res.Items, 1)
	require.Equal(t, entityID, res.Items[0].EntityID)
	require.Equal(t, "Lotus", res.Items[0].Payload["name"])
}

func TestService_CreateValidation(t *testing.T) {
	svc := New(&stubRepository{})
	_, err := svc.Create(context.Background(), "", map[string]interface{}{"name": "test"})
	require.Error(t, err)
	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
}

func TestService_CreateNotFound(t *testing.T) {
	repo := &stubRepository{
		createFn: func(context.Context, string, json.RawMessage) (persistence.EntityRecord, error) {
			return persistence.EntityRecord{}, persistence.ErrSchemaNotFound
		},
	}
	svc := New(repo)
	_, err := svc.Create(context.Background(), "cards_entities", map[string]interface{}{"name": "test"})
	require.ErrorIs(t, err, ErrTableNotFound)
}

func TestService_UpdateRequiresPayload(t *testing.T) {
	svc := New(&stubRepository{})
	_, err := svc.Update(context.Background(), "cards_entities", uuid.New(), nil)
	require.Error(t, err)
	var valErr *ValidationError
	require.ErrorAs(t, err, &valErr)
}

func TestService_DeleteNotFound(t *testing.T) {
	repo := &stubRepository{
		deleteFn: func(context.Context, string, uuid.UUID) error {
			return persistence.ErrEntityNotFound
		},
	}
	svc := New(repo)
	err := svc.Delete(context.Background(), "cards_entities", uuid.New())
	require.ErrorIs(t, err, ErrDocumentNotFound)
}

type stubRepository struct {
	listFn   func(context.Context, string, domainrepo.ListParams) (domainrepo.ListResult, error)
	createFn func(context.Context, string, json.RawMessage) (persistence.EntityRecord, error)
	getFn    func(context.Context, string, uuid.UUID) (persistence.EntityRecord, error)
	updateFn func(context.Context, string, uuid.UUID, json.RawMessage) (persistence.EntityRecord, error)
	deleteFn func(context.Context, string, uuid.UUID) error
}

func (s *stubRepository) List(ctx context.Context, table string, params domainrepo.ListParams) (domainrepo.ListResult, error) {
	if s.listFn == nil {
		return domainrepo.ListResult{}, nil
	}
	return s.listFn(ctx, table, params)
}

func (s *stubRepository) Create(ctx context.Context, table string, payload json.RawMessage) (persistence.EntityRecord, error) {
	if s.createFn == nil {
		return persistence.EntityRecord{}, nil
	}
	return s.createFn(ctx, table, payload)
}

func (s *stubRepository) Get(ctx context.Context, table string, entityID uuid.UUID) (persistence.EntityRecord, error) {
	if s.getFn == nil {
		return persistence.EntityRecord{}, nil
	}
	return s.getFn(ctx, table, entityID)
}

func (s *stubRepository) Update(ctx context.Context, table string, entityID uuid.UUID, payload json.RawMessage) (persistence.EntityRecord, error) {
	if s.updateFn == nil {
		return persistence.EntityRecord{}, nil
	}
	return s.updateFn(ctx, table, entityID, payload)
}

func (s *stubRepository) Delete(ctx context.Context, table string, entityID uuid.UUID) error {
	if s.deleteFn == nil {
		return nil
	}
	return s.deleteFn(ctx, table, entityID)
}
