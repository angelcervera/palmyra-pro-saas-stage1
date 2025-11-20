package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/requesttrace"
)

type mockRepository struct {
	createFn     func(ctx context.Context, params persistence.CreateUserParams) (persistence.User, error)
	listFn       func(ctx context.Context, params persistence.ListUsersParams) (persistence.ListUsersResult, error)
	getFn        func(ctx context.Context, id uuid.UUID) (persistence.User, error)
	updateFn     func(ctx context.Context, id uuid.UUID, params persistence.UpdateUserParams) (persistence.User, error)
	updateNameFn func(ctx context.Context, id uuid.UUID, fullName string) (persistence.User, error)
	deleteFn     func(ctx context.Context, id uuid.UUID) error
}

func (m *mockRepository) Create(ctx context.Context, params persistence.CreateUserParams) (persistence.User, error) {
	if m.createFn == nil {
		panic("createFn not configured")
	}
	return m.createFn(ctx, params)
}

func (m *mockRepository) List(ctx context.Context, params persistence.ListUsersParams) (persistence.ListUsersResult, error) {
	if m.listFn == nil {
		panic("listFn not configured")
	}
	return m.listFn(ctx, params)
}

func (m *mockRepository) Get(ctx context.Context, id uuid.UUID) (persistence.User, error) {
	if m.getFn == nil {
		panic("getFn not configured")
	}
	return m.getFn(ctx, id)
}

func (m *mockRepository) Update(ctx context.Context, id uuid.UUID, params persistence.UpdateUserParams) (persistence.User, error) {
	if m.updateFn == nil {
		panic("updateFn not configured")
	}
	return m.updateFn(ctx, id, params)
}

func (m *mockRepository) UpdateFullName(ctx context.Context, id uuid.UUID, fullName string) (persistence.User, error) {
	if m.updateNameFn == nil {
		panic("updateNameFn not configured")
	}
	return m.updateNameFn(ctx, id, fullName)
}

func (m *mockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn == nil {
		panic("deleteFn not configured")
	}
	return m.deleteFn(ctx, id)
}

func TestServiceCreateValidation(t *testing.T) {
	t.Parallel()

	svc := New(&mockRepository{})
	audit := requesttrace.Anonymous("test")

	_, err := svc.Create(context.Background(), audit, CreateInput{})
	require.Error(t, err)

	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr))
	require.Contains(t, validationErr.Fields, "email")
	require.Contains(t, validationErr.Fields, "fullName")
}

func TestServiceCreateSuccess(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	repository := &mockRepository{}

	repository.createFn = func(ctx context.Context, params persistence.CreateUserParams) (persistence.User, error) {
		require.NotEqual(t, uuid.Nil, params.UserID)
		require.Equal(t, "admin@example.com", params.Email)
		require.Equal(t, "Admin", params.FullName)

		return persistence.User{
			UserID:    params.UserID,
			Email:     params.Email,
			FullName:  params.FullName,
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}

	svc := New(repository)
	audit := requesttrace.Anonymous("test")

	user, err := svc.Create(context.Background(), audit, CreateInput{
		Email:    "  Admin@example.com ",
		FullName: " Admin ",
	})
	require.NoError(t, err)
	require.Equal(t, "admin@example.com", user.Email)
	require.Equal(t, "Admin", user.FullName)
}

func TestServiceListSuccess(t *testing.T) {
	t.Parallel()

	repository := &mockRepository{}
	now := time.Now().UTC()
	userID := uuid.New()

	repository.listFn = func(ctx context.Context, params persistence.ListUsersParams) (persistence.ListUsersResult, error) {
		require.Equal(t, 2, params.Page)
		require.Equal(t, 10, params.PageSize)
		require.NotNil(t, params.Sort)
		require.Equal(t, "createdAt", *params.Sort)
		require.NotNil(t, params.Email)
		require.Equal(t, "admin@example.com", *params.Email)

		return persistence.ListUsersResult{
			Users: []persistence.User{{
				UserID:    userID,
				Email:     "admin@example.com",
				FullName:  "Admin",
				CreatedAt: now,
				UpdatedAt: now,
			}},
			TotalItems: 15,
		}, nil
	}

	svc := New(repository)
	audit := requesttrace.Anonymous("test")

	sort := "createdAt"
	result, err := svc.List(context.Background(), audit, ListOptions{
		Page:     2,
		PageSize: 10,
		Sort:     &sort,
		Email:    ptrString(" admin@example.com "),
	})

	require.NoError(t, err)
	require.Equal(t, 2, result.Page)
	require.Equal(t, 10, result.PageSize)
	require.Equal(t, 15, result.TotalItems)
	require.Equal(t, 2, result.TotalPages)
	require.Len(t, result.Users, 1)
	require.Equal(t, userID, result.Users[0].ID)
	require.Equal(t, "Admin", result.Users[0].FullName)
}

func TestServiceListInvalidSort(t *testing.T) {
	t.Parallel()

	svc := New(&mockRepository{listFn: func(ctx context.Context, params persistence.ListUsersParams) (persistence.ListUsersResult, error) {
		return persistence.ListUsersResult{}, nil
	}})
	audit := requesttrace.Anonymous("test")

	sort := "-invalid"
	_, err := svc.List(context.Background(), audit, ListOptions{Sort: &sort})
	require.Error(t, err)

	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr))
	require.Contains(t, validationErr.Fields, "sort")
}

func TestServiceUpdateValidation(t *testing.T) {
	t.Parallel()

	svc := New(&mockRepository{})
	audit := requesttrace.Anonymous("test")
	_, err := svc.Update(context.Background(), audit, uuid.New(), UpdateInput{})
	require.Error(t, err)

	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr))
	require.Contains(t, validationErr.Fields, "payload")
}

func TestServiceUpdateSuccess(t *testing.T) {
	t.Parallel()

	repository := &mockRepository{}
	now := time.Now().UTC()
	userID := uuid.New()

	repository.updateFn = func(ctx context.Context, id uuid.UUID, params persistence.UpdateUserParams) (persistence.User, error) {
		require.Equal(t, userID, id)
		require.NotNil(t, params.FullName)
		require.Equal(t, "Admin", *params.FullName)

		return persistence.User{
			UserID:    id,
			Email:     "admin@example.com",
			FullName:  *params.FullName,
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}

	svc := New(repository)
	audit := requesttrace.Anonymous("test")

	updated, err := svc.Update(context.Background(), audit, userID, UpdateInput{
		FullName: ptrString("Admin"),
	})

	require.NoError(t, err)
	require.Equal(t, "Admin", updated.FullName)
}

func TestServiceUpdateSelfValidation(t *testing.T) {
	t.Parallel()

	svc := New(&mockRepository{})
	audit := requesttrace.Anonymous("test")
	_, err := svc.UpdateSelf(context.Background(), audit, uuid.New(), UpdateSelfInput{})
	require.Error(t, err)

	var validationErr *ValidationError
	require.True(t, errors.As(err, &validationErr))
	require.Contains(t, validationErr.Fields, "fullName")
}

func TestServiceUpdateSelfSuccess(t *testing.T) {
	t.Parallel()

	repository := &mockRepository{}
	now := time.Now().UTC()
	userID := uuid.New()

	repository.updateNameFn = func(ctx context.Context, id uuid.UUID, fullName string) (persistence.User, error) {
		require.Equal(t, userID, id)
		require.Equal(t, "Admin", fullName)
		return persistence.User{UserID: id, FullName: fullName, CreatedAt: now, UpdatedAt: now}, nil
	}

	svc := New(repository)
	audit := requesttrace.Anonymous("test")

	n, err := svc.UpdateSelf(context.Background(), audit, userID, UpdateSelfInput{FullName: ptrString(" Admin ")})
	require.NoError(t, err)
	require.Equal(t, "Admin", n.FullName)
}

func TestServiceDeleteInvalidID(t *testing.T) {
	t.Parallel()

	svc := New(&mockRepository{})
	audit := requesttrace.Anonymous("test")

	err := svc.Delete(context.Background(), audit, uuid.Nil)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceDeleteNotFound(t *testing.T) {
	t.Parallel()

	repository := &mockRepository{}
	repository.deleteFn = func(ctx context.Context, id uuid.UUID) error {
		return persistence.ErrUserNotFound
	}

	svc := New(repository)
	audit := requesttrace.Anonymous("test")

	err := svc.Delete(context.Background(), audit, uuid.New())
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceDeleteSuccess(t *testing.T) {
	t.Parallel()

	repository := &mockRepository{}
	userID := uuid.New()
	called := false

	repository.deleteFn = func(ctx context.Context, id uuid.UUID) error {
		require.Equal(t, userID, id)
		called = true
		return nil
	}

	svc := New(repository)
	audit := requesttrace.Anonymous("test")

	err := svc.Delete(context.Background(), audit, userID)
	require.NoError(t, err)
	require.True(t, called)
}

func ptrString(v string) *string {
	s := v
	return &s
}
