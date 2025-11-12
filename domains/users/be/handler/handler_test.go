package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/users/be/service"
	externalRef2 "github.com/zenGate-Global/palmyra-pro-saas/generated/go/common/primitives"
	users "github.com/zenGate-Global/palmyra-pro-saas/generated/go/users"
	platformauth "github.com/zenGate-Global/palmyra-pro-saas/platform/go/auth"
)

type mockService struct {
	createFn     func(ctx context.Context, input service.CreateInput) (service.User, error)
	listFn       func(ctx context.Context, opts service.ListOptions) (service.ListResult, error)
	getFn        func(ctx context.Context, id uuid.UUID) (service.User, error)
	updateFn     func(ctx context.Context, id uuid.UUID, input service.UpdateInput) (service.User, error)
	updateSelfFn func(ctx context.Context, id uuid.UUID, input service.UpdateSelfInput) (service.User, error)
	deleteFn     func(ctx context.Context, id uuid.UUID) error
}

func (m *mockService) Create(ctx context.Context, input service.CreateInput) (service.User, error) {
	if m.createFn == nil {
		panic("createFn not configured")
	}
	return m.createFn(ctx, input)
}

func (m *mockService) List(ctx context.Context, opts service.ListOptions) (service.ListResult, error) {
	if m.listFn == nil {
		panic("listFn not configured")
	}
	return m.listFn(ctx, opts)
}

func (m *mockService) Get(ctx context.Context, id uuid.UUID) (service.User, error) {
	if m.getFn == nil {
		panic("getFn not configured")
	}
	return m.getFn(ctx, id)
}

func (m *mockService) Update(ctx context.Context, id uuid.UUID, input service.UpdateInput) (service.User, error) {
	if m.updateFn == nil {
		panic("updateFn not configured")
	}
	return m.updateFn(ctx, id, input)
}

func (m *mockService) UpdateSelf(ctx context.Context, id uuid.UUID, input service.UpdateSelfInput) (service.User, error) {
	if m.updateSelfFn == nil {
		panic("updateSelfFn not configured")
	}
	return m.updateSelfFn(ctx, id, input)
}

func (m *mockService) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn == nil {
		panic("deleteFn not configured")
	}
	return m.deleteFn(ctx, id)
}

func TestUsersListSuccess(t *testing.T) {
	t.Parallel()

	svc := &mockService{}
	now := time.Now().UTC()
	userID := uuid.New()

	svc.listFn = func(ctx context.Context, opts service.ListOptions) (service.ListResult, error) {
		return service.ListResult{
			Users: []service.User{{
				ID:        userID,
				Email:     "admin@example.com",
				FullName:  "Admin",
				CreatedAt: now,
				UpdatedAt: now,
			}},
			Page:       1,
			PageSize:   20,
			TotalItems: 1,
			TotalPages: 1,
		}, nil
	}

	h := New(svc, zaptest.NewLogger(t))

	resp, err := h.UsersList(context.Background(), users.UsersListRequestObject{})
	require.NoError(t, err)

	success, ok := resp.(users.UsersList200JSONResponse)
	require.True(t, ok)
	require.Len(t, success.Items, 1)
	require.Equal(t, externalRef2.UUID(userID), success.Items[0].Id)
}

func TestUsersCreateMissingBody(t *testing.T) {
	t.Parallel()

	h := New(&mockService{}, zaptest.NewLogger(t))

	resp, err := h.UsersCreate(context.Background(), users.UsersCreateRequestObject{})
	require.NoError(t, err)

	problem, ok := resp.(users.UsersCreatedefaultApplicationProblemPlusJSONResponse)
	require.True(t, ok)
	require.Equal(t, http.StatusBadRequest, problem.StatusCode)
}

func TestUsersCreateValidationError(t *testing.T) {
	t.Parallel()

	svc := &mockService{}
	svc.createFn = func(ctx context.Context, input service.CreateInput) (service.User, error) {
		return service.User{}, &service.ValidationError{Fields: service.FieldErrors{"email": {"invalid"}}}
	}

	h := New(svc, zaptest.NewLogger(t))

	body := &users.CreateUser{
		Email:    externalRef2.Email("admin@example.com"),
		FullName: "Admin",
	}

	resp, err := h.UsersCreate(context.Background(), users.UsersCreateRequestObject{Body: body})
	require.NoError(t, err)

	problem, ok := resp.(users.UsersCreatedefaultApplicationProblemPlusJSONResponse)
	require.True(t, ok)
	require.Equal(t, http.StatusBadRequest, problem.StatusCode)
}

func TestUsersCreateSuccess(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	userID := uuid.New()

	svc := &mockService{}
	svc.createFn = func(ctx context.Context, input service.CreateInput) (service.User, error) {
		return service.User{
			ID:        userID,
			Email:     "admin@example.com",
			FullName:  "Admin",
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}

	h := New(svc, zaptest.NewLogger(t))

	body := &users.CreateUser{
		Email:    externalRef2.Email("admin@example.com"),
		FullName: "Admin",
	}

	resp, err := h.UsersCreate(context.Background(), users.UsersCreateRequestObject{Body: body})
	require.NoError(t, err)

	success, ok := resp.(users.UsersCreate201JSONResponse)
	require.True(t, ok)
	require.Equal(t, "/api/v1/admin/users/"+userID.String(), success.Headers.Location)
	require.Equal(t, externalRef2.UUID(userID), success.Body.Id)
}

func TestUsersUpdateMissingBody(t *testing.T) {
	t.Parallel()

	h := New(&mockService{}, zaptest.NewLogger(t))

	resp, err := h.UsersUpdate(context.Background(), users.UsersUpdateRequestObject{})
	require.NoError(t, err)

	problem, ok := resp.(users.UsersUpdatedefaultApplicationProblemPlusJSONResponse)
	require.True(t, ok)
	require.Equal(t, http.StatusBadRequest, problem.StatusCode)
}

func TestUsersMeUnauthorized(t *testing.T) {
	t.Parallel()

	h := New(&mockService{}, zaptest.NewLogger(t))

	resp, err := h.UsersMe(context.Background(), users.UsersMeRequestObject{})
	require.NoError(t, err)

	problem, ok := resp.(users.UsersMedefaultApplicationProblemPlusJSONResponse)
	require.True(t, ok)
	require.Equal(t, http.StatusUnauthorized, problem.StatusCode)
}

func TestUsersUpdateMeSuccess(t *testing.T) {
	t.Parallel()

	svc := &mockService{}
	userID := uuid.New()
	now := time.Now().UTC()

	svc.updateSelfFn = func(ctx context.Context, id uuid.UUID, input service.UpdateSelfInput) (service.User, error) {
		require.Equal(t, userID, id)
		require.NotNil(t, input.FullName)
		return service.User{
			ID:        id,
			Email:     "user@example.com",
			FullName:  strings.TrimSpace(*input.FullName),
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}

	h := New(svc, zaptest.NewLogger(t))

	ctx := contextWithCredentials(t, platformauth.UserCredentials{
		Id:    userID.String(),
		Email: "user@example.com",
	})

	fullName := " User "
	resp, err := h.UsersUpdateMe(ctx, users.UsersUpdateMeRequestObject{Body: &users.UsersUpdateMeJSONRequestBody{FullName: &fullName}})
	require.NoError(t, err)

	success, ok := resp.(users.UsersUpdateMe200JSONResponse)
	require.True(t, ok)
	require.Equal(t, "User", success.FullName)
}

func TestUsersDeleteSuccess(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := &mockService{}
	svc.deleteFn = func(ctx context.Context, id uuid.UUID) error {
		require.Equal(t, userID, id)
		return nil
	}

	h := New(svc, zaptest.NewLogger(t))

	resp, err := h.UsersDelete(context.Background(), users.UsersDeleteRequestObject{
		UserId: externalRef2.UUID(userID),
	})
	require.NoError(t, err)

	success, ok := resp.(users.UsersDelete204Response)
	require.True(t, ok)
	recorder := httptest.NewRecorder()
	require.NoError(t, success.VisitUsersDeleteResponse(recorder))
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestUsersDeleteNotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := &mockService{}
	svc.deleteFn = func(ctx context.Context, id uuid.UUID) error {
		return service.ErrNotFound
	}

	h := New(svc, zaptest.NewLogger(t))

	resp, err := h.UsersDelete(context.Background(), users.UsersDeleteRequestObject{
		UserId: externalRef2.UUID(userID),
	})
	require.NoError(t, err)

	problem, ok := resp.(users.UsersDeletedefaultApplicationProblemPlusJSONResponse)
	require.True(t, ok)
	require.Equal(t, http.StatusNotFound, problem.StatusCode)
}

func contextWithCredentials(t *testing.T, creds platformauth.UserCredentials) context.Context {
	t.Helper()

	verify := func(ctx context.Context, token string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"uid":            creds.Id,
			"email":          creds.Email,
			"email_verified": creds.EmailVerified,
			"name":           creds.Name,
			"isAdmin":        creds.IsAdmin,
		}, nil
	}

	extract := func(claims map[string]interface{}) (*platformauth.UserCredentials, error) {
		return &creds, nil
	}

	middleware := platformauth.JWT(verify, extract)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()

	var captured context.Context
	middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Context()
	})).ServeHTTP(recorder, req)

	require.NotNil(t, captured)
	return captured
}
