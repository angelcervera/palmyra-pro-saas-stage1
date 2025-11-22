package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// Repository defines the persistence operations required by the users service.
type Repository interface {
	Create(ctx context.Context, params persistence.CreateUserParams) (persistence.User, error)
	List(ctx context.Context, params persistence.ListUsersParams) (persistence.ListUsersResult, error)
	Get(ctx context.Context, id uuid.UUID) (persistence.User, error)
	Update(ctx context.Context, id uuid.UUID, params persistence.UpdateUserParams) (persistence.User, error)
	UpdateFullName(ctx context.Context, id uuid.UUID, fullName string) (persistence.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type postgresRepository struct {
	store *persistence.UserStore
}

// NewPostgresRepository constructs a repository backed by the shared persistence layer.
func NewPostgresRepository(store *persistence.UserStore) Repository {
	if store == nil {
		panic("user store is required")
	}
	return &postgresRepository{store: store}
}

func (r *postgresRepository) List(ctx context.Context, params persistence.ListUsersParams) (persistence.ListUsersResult, error) {
	space, err := requireTenantSpace(ctx)
	if err != nil {
		return persistence.ListUsersResult{}, err
	}
	return r.store.ListUsers(ctx, space, params)
}

func (r *postgresRepository) Create(ctx context.Context, params persistence.CreateUserParams) (persistence.User, error) {
	space, err := requireTenantSpace(ctx)
	if err != nil {
		return persistence.User{}, err
	}
	return r.store.CreateUser(ctx, space, params)
}

func (r *postgresRepository) Get(ctx context.Context, id uuid.UUID) (persistence.User, error) {
	space, err := requireTenantSpace(ctx)
	if err != nil {
		return persistence.User{}, err
	}
	return r.store.GetUser(ctx, space, id)
}

func (r *postgresRepository) Update(ctx context.Context, id uuid.UUID, params persistence.UpdateUserParams) (persistence.User, error) {
	space, err := requireTenantSpace(ctx)
	if err != nil {
		return persistence.User{}, err
	}
	return r.store.UpdateUser(ctx, space, id, params)
}

func (r *postgresRepository) UpdateFullName(ctx context.Context, id uuid.UUID, fullName string) (persistence.User, error) {
	space, err := requireTenantSpace(ctx)
	if err != nil {
		return persistence.User{}, err
	}
	return r.store.UpdateUserFullName(ctx, space, id, fullName)
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	space, err := requireTenantSpace(ctx)
	if err != nil {
		return err
	}
	return r.store.DeleteUser(ctx, space, id)
}

func requireTenantSpace(ctx context.Context) (tenant.Space, error) {
	space, ok := tenant.FromContext(ctx)
	if !ok {
		return tenant.Space{}, errors.New("tenant space missing from context")
	}
	return space, nil
}
