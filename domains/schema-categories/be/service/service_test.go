package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/requesttrace"
)

type mockRepository struct {
	listFn       func(ctx context.Context, includeDeleted bool) ([]persistence.SchemaCategory, error)
	createFn     func(ctx context.Context, params persistence.CreateSchemaCategoryParams) (persistence.SchemaCategory, error)
	getFn        func(ctx context.Context, id uuid.UUID) (persistence.SchemaCategory, error)
	updateFn     func(ctx context.Context, id uuid.UUID, params persistence.UpdateSchemaCategoryParams) (persistence.SchemaCategory, error)
	softDeleteFn func(ctx context.Context, id uuid.UUID, deletedAt time.Time) error
}

func (m *mockRepository) List(ctx context.Context, includeDeleted bool) ([]persistence.SchemaCategory, error) {
	if m.listFn == nil {
		panic("listFn not configured")
	}
	return m.listFn(ctx, includeDeleted)
}

func (m *mockRepository) Create(ctx context.Context, params persistence.CreateSchemaCategoryParams) (persistence.SchemaCategory, error) {
	if m.createFn == nil {
		panic("createFn not configured")
	}
	return m.createFn(ctx, params)
}

func (m *mockRepository) Get(ctx context.Context, id uuid.UUID) (persistence.SchemaCategory, error) {
	if m.getFn == nil {
		panic("getFn not configured")
	}
	return m.getFn(ctx, id)
}

func (m *mockRepository) Update(ctx context.Context, id uuid.UUID, params persistence.UpdateSchemaCategoryParams) (persistence.SchemaCategory, error) {
	if m.updateFn == nil {
		panic("updateFn not configured")
	}
	return m.updateFn(ctx, id, params)
}

func (m *mockRepository) SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	if m.softDeleteFn == nil {
		panic("softDeleteFn not configured")
	}
	return m.softDeleteFn(ctx, id, deletedAt)
}

func TestServiceCreateSuccess(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	now := time.Date(2024, time.November, 24, 10, 0, 0, 0, time.UTC)

	repo.createFn = func(ctx context.Context, params persistence.CreateSchemaCategoryParams) (persistence.SchemaCategory, error) {
		require.Equal(t, "cards", params.Slug)
		require.Equal(t, "Cards", params.Name)
		require.NotEqual(t, uuid.Nil, params.CategoryID)

		return persistence.SchemaCategory{
			CategoryID:       params.CategoryID,
			ParentCategoryID: params.ParentCategoryID,
			Name:             params.Name,
			Slug:             params.Slug,
			Description:      params.Description,
			CreatedAt:        now,
			UpdatedAt:        now,
		}, nil
	}

	svc := New(repo).(*service)
	svc.now = func() time.Time { return now }

	audit := requesttrace.Anonymous("test")

	result, err := svc.Create(context.Background(), audit, CreateInput{
		Name: "  Cards ",
		Slug: "Cards",
	})

	require.NoError(t, err)
	require.Equal(t, "Cards", result.Name)
	require.Equal(t, "cards", result.Slug)
	require.Equal(t, now, result.CreatedAt)
}

func TestServiceCreateValidationError(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	svc := New(repo)

	audit := requesttrace.Anonymous("test")

	_, err := svc.Create(context.Background(), audit, CreateInput{})
	require.Error(t, err)

	validationErr, ok := err.(*ValidationError)
	require.True(t, ok)
	require.Contains(t, validationErr.Fields, "name")
	require.Contains(t, validationErr.Fields, "slug")
}

func TestServiceCreateConflict(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	repo.createFn = func(ctx context.Context, params persistence.CreateSchemaCategoryParams) (persistence.SchemaCategory, error) {
		return persistence.SchemaCategory{}, persistence.ErrSchemaCategoryConflict
	}

	svc := New(repo)
	audit := requesttrace.Anonymous("test")

	_, err := svc.Create(context.Background(), audit, CreateInput{Name: "Cards", Slug: "cards"})
	require.ErrorIs(t, err, ErrConflict)
}

func TestServiceCreateInvalidParent(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	missingParent := uuid.New()
	repo.getFn = func(ctx context.Context, id uuid.UUID) (persistence.SchemaCategory, error) {
		require.Equal(t, missingParent, id)
		return persistence.SchemaCategory{}, persistence.ErrSchemaNotFound
	}

	svc := New(repo)
	audit := requesttrace.Anonymous("test")

	_, err := svc.Create(context.Background(), audit, CreateInput{
		Name:     "Cards",
		Slug:     "cards",
		ParentID: &missingParent,
	})
	require.Error(t, err)
	validationErr, ok := err.(*ValidationError)
	require.True(t, ok)
	require.Contains(t, validationErr.Fields, "parentCategoryId")
}

func TestServiceUpdateSuccess(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	categoryID := uuid.New()
	now := time.Date(2024, time.November, 24, 10, 0, 0, 0, time.UTC)

	repo.getFn = func(ctx context.Context, id uuid.UUID) (persistence.SchemaCategory, error) {
		if id == categoryID {
			return persistence.SchemaCategory{}, persistence.ErrSchemaNotFound
		}
		return persistence.SchemaCategory{CategoryID: id}, nil
	}

	repo.updateFn = func(ctx context.Context, id uuid.UUID, params persistence.UpdateSchemaCategoryParams) (persistence.SchemaCategory, error) {
		require.Equal(t, "Renamed", *params.Name)
		return persistence.SchemaCategory{
			CategoryID: id,
			Name:       *params.Name,
			Slug:       "cards",
			CreatedAt:  now,
			UpdatedAt:  now,
		}, nil
	}

	svc := New(repo)
	audit := requesttrace.Anonymous("test")

	updated, err := svc.Update(context.Background(), audit, categoryID, UpdateInput{Name: stringPtr(" Renamed ")})
	require.NoError(t, err)
	require.Equal(t, "Renamed", updated.Name)
}

func TestServiceUpdateSlug(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	categoryID := uuid.New()
	now := time.Date(2024, time.November, 24, 10, 0, 0, 0, time.UTC)

	repo.getFn = func(ctx context.Context, id uuid.UUID) (persistence.SchemaCategory, error) {
		return persistence.SchemaCategory{
			CategoryID: id,
			Name:       "Original",
			Slug:       "original-slug",
			CreatedAt:  now,
			UpdatedAt:  now,
		}, nil
	}

	repo.updateFn = func(ctx context.Context, id uuid.UUID, params persistence.UpdateSchemaCategoryParams) (persistence.SchemaCategory, error) {
		require.NotNil(t, params.Slug)
		require.Equal(t, "updated-slug", *params.Slug)
		return persistence.SchemaCategory{
			CategoryID: id,
			Name:       "Original",
			Slug:       *params.Slug,
			CreatedAt:  now,
			UpdatedAt:  now,
		}, nil
	}

	svc := New(repo)
	audit := requesttrace.Anonymous("test")

	updated, err := svc.Update(context.Background(), audit, categoryID, UpdateInput{Slug: stringPtr("updated-slug")})
	require.NoError(t, err)
	require.Equal(t, "updated-slug", updated.Slug)
}

func TestServiceUpdateParentSelfReference(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	id := uuid.New()
	svc := New(repo)
	audit := requesttrace.Anonymous("test")

	_, err := svc.Update(context.Background(), audit, id, UpdateInput{
		ParentID: &id,
	})
	require.Error(t, err)
	validationErr, ok := err.(*ValidationError)
	require.True(t, ok)
	require.Contains(t, validationErr.Fields, "parentCategoryId")
}

func TestServiceUpdateValidation(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	svc := New(repo)
	audit := requesttrace.Anonymous("test")

	_, err := svc.Update(context.Background(), audit, uuid.New(), UpdateInput{})
	require.Error(t, err)
	_, ok := err.(*ValidationError)
	require.True(t, ok)
}

func TestServiceDeleteNotFound(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	repo.softDeleteFn = func(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
		return persistence.ErrSchemaNotFound
	}

	svc := New(repo)
	audit := requesttrace.Anonymous("test")

	err := svc.Delete(context.Background(), audit, uuid.New())
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceList(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	now := time.Now().UTC()
	repo.listFn = func(ctx context.Context, includeDeleted bool) ([]persistence.SchemaCategory, error) {
		require.True(t, includeDeleted)
		return []persistence.SchemaCategory{
			{
				CategoryID: uuid.New(),
				Name:       "Cards",
				Slug:       "cards",
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		}, nil
	}

	svc := New(repo)
	audit := requesttrace.Anonymous("test")

	list, err := svc.List(context.Background(), audit, true)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "Cards", list[0].Name)
}

func stringPtr(value string) *string {
	return &value
}
