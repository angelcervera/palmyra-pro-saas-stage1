package handler

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/TCGLandDev/tcgdb/domains/schema-categories/be/service"
	externalRef2 "github.com/TCGLandDev/tcgdb/generated/go/common/primitives"
	schemacategories "github.com/TCGLandDev/tcgdb/generated/go/schema-categories"
	"go.uber.org/zap/zaptest"
)

type mockService struct {
	listFn   func(ctx context.Context, includeDeleted bool) ([]service.Category, error)
	createFn func(ctx context.Context, input service.CreateInput) (service.Category, error)
	getFn    func(ctx context.Context, id uuid.UUID) (service.Category, error)
	updateFn func(ctx context.Context, id uuid.UUID, input service.UpdateInput) (service.Category, error)
	deleteFn func(ctx context.Context, id uuid.UUID) error
}

func (m *mockService) List(ctx context.Context, includeDeleted bool) ([]service.Category, error) {
	if m.listFn == nil {
		panic("listFn not configured")
	}
	return m.listFn(ctx, includeDeleted)
}

func (m *mockService) Create(ctx context.Context, input service.CreateInput) (service.Category, error) {
	if m.createFn == nil {
		panic("createFn not configured")
	}
	return m.createFn(ctx, input)
}

func (m *mockService) Get(ctx context.Context, id uuid.UUID) (service.Category, error) {
	if m.getFn == nil {
		panic("getFn not configured")
	}
	return m.getFn(ctx, id)
}

func (m *mockService) Update(ctx context.Context, id uuid.UUID, input service.UpdateInput) (service.Category, error) {
	if m.updateFn == nil {
		panic("updateFn not configured")
	}
	return m.updateFn(ctx, id, input)
}

func (m *mockService) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn == nil {
		panic("deleteFn not configured")
	}
	return m.deleteFn(ctx, id)
}

func TestHandlerListSchemaCategories(t *testing.T) {
	t.Parallel()

	svc := &mockService{}
	logger := zaptest.NewLogger(t)
	handler := New(svc, logger)

	svc.listFn = func(ctx context.Context, includeDeleted bool) ([]service.Category, error) {
		require.True(t, includeDeleted)
		now := time.Now().UTC()
		return []service.Category{
			{
				ID:        uuid.New(),
				Name:      "Cards",
				Slug:      "cards",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}, nil
	}

	includeDeleted := true
	response, err := handler.ListSchemaCategories(context.Background(), schemacategories.ListSchemaCategoriesRequestObject{
		Params: schemacategories.ListSchemaCategoriesParams{IncludeDeleted: &includeDeleted},
	})
	require.NoError(t, err)

	success, ok := response.(schemacategories.ListSchemaCategories200JSONResponse)
	require.True(t, ok)
	require.Len(t, success.Items, 1)
	require.Equal(t, "Cards", success.Items[0].Name)
}

func TestHandlerCreateSchemaCategory(t *testing.T) {
	t.Parallel()

	svc := &mockService{}
	logger := zaptest.NewLogger(t)
	handler := New(svc, logger)

	svc.createFn = func(ctx context.Context, input service.CreateInput) (service.Category, error) {
		require.Equal(t, "Cards", input.Name)
		require.Equal(t, "cards", input.Slug)
		now := time.Now().UTC()
		return service.Category{
			ID:        uuid.New(),
			Name:      input.Name,
			Slug:      input.Slug,
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}

	body := schemacategories.CreateSchemaCategoryRequest{
		Name: "Cards",
		Slug: externalRef2.Slug("cards"),
	}

	response, err := handler.CreateSchemaCategory(context.Background(), schemacategories.CreateSchemaCategoryRequestObject{Body: &body})
	require.NoError(t, err)

	success, ok := response.(schemacategories.CreateSchemaCategory201JSONResponse)
	require.True(t, ok)
	require.NotEmpty(t, success.Headers.Location)
	require.Equal(t, "Cards", success.Body.Name)
}

func TestHandlerCreateSchemaCategoryValidationError(t *testing.T) {
	t.Parallel()

	svc := &mockService{}
	logger := zaptest.NewLogger(t)
	handler := New(svc, logger)

	svc.createFn = func(ctx context.Context, input service.CreateInput) (service.Category, error) {
		return service.Category{}, &service.ValidationError{Fields: service.FieldErrors{"name": {"required"}}}
	}

	body := schemacategories.CreateSchemaCategoryRequest{
		Name: "",
		Slug: externalRef2.Slug("cards"),
	}

	response, err := handler.CreateSchemaCategory(context.Background(), schemacategories.CreateSchemaCategoryRequestObject{Body: &body})
	require.NoError(t, err)

	problem, ok := response.(schemacategories.CreateSchemaCategorydefaultApplicationProblemPlusJSONResponse)
	require.True(t, ok)
	require.Equal(t, http.StatusBadRequest, problem.StatusCode)
	require.NotNil(t, problem.Body.Errors)
	require.Contains(t, (*problem.Body.Errors)["name"], "required")
}

func TestHandlerGetSchemaCategoryNotFound(t *testing.T) {
	t.Parallel()

	svc := &mockService{}
	logger := zaptest.NewLogger(t)
	handler := New(svc, logger)

	svc.getFn = func(ctx context.Context, id uuid.UUID) (service.Category, error) {
		return service.Category{}, service.ErrNotFound
	}

	response, err := handler.GetSchemaCategory(context.Background(), schemacategories.GetSchemaCategoryRequestObject{
		CategoryId: externalRef2.UUID(uuid.New()),
	})
	require.NoError(t, err)

	problem, ok := response.(schemacategories.GetSchemaCategorydefaultApplicationProblemPlusJSONResponse)
	require.True(t, ok)
	require.Equal(t, http.StatusNotFound, problem.StatusCode)
}

func TestHandlerUpdateSchemaCategoryMissingBody(t *testing.T) {
	t.Parallel()

	svc := &mockService{}
	logger := zaptest.NewLogger(t)
	handler := New(svc, logger)

	response, err := handler.UpdateSchemaCategory(context.Background(), schemacategories.UpdateSchemaCategoryRequestObject{})
	require.NoError(t, err)

	problem, ok := response.(schemacategories.UpdateSchemaCategorydefaultApplicationProblemPlusJSONResponse)
	require.True(t, ok)
	require.Equal(t, http.StatusBadRequest, problem.StatusCode)
}

func TestHandlerUpdateSchemaCategorySuccess(t *testing.T) {
	t.Parallel()

	svc := &mockService{}
	logger := zaptest.NewLogger(t)
	handler := New(svc, logger)

	categoryID := uuid.New()
	now := time.Now().UTC()

	svc.updateFn = func(ctx context.Context, id uuid.UUID, input service.UpdateInput) (service.Category, error) {
		require.Equal(t, categoryID, id)
		require.NotNil(t, input.Slug)
		require.Equal(t, "updated-slug", *input.Slug)
		return service.Category{
			ID:        id,
			Name:      "Updated",
			Slug:      *input.Slug,
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}

	body := schemacategories.UpdateSchemaCategoryJSONRequestBody{
		Slug: slugPtr("updated-slug"),
		Name: ptrString("Updated"),
	}

	response, err := handler.UpdateSchemaCategory(context.Background(), schemacategories.UpdateSchemaCategoryRequestObject{
		CategoryId: externalRef2.UUID(categoryID),
		Body:       &body,
	})
	require.NoError(t, err)

	success, ok := response.(schemacategories.UpdateSchemaCategory200JSONResponse)
	require.True(t, ok)
	require.Equal(t, externalRef2.Slug("updated-slug"), success.Slug)
}

func ptrString(value string) *string {
	return &value
}

func slugPtr(value string) *externalRef2.Slug {
	slug := externalRef2.Slug(value)
	return &slug
}
