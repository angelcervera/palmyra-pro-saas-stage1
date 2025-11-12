package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/TCGLandDev/tcgdb/platform/go/persistence"
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
	return r.store.ListUsers(ctx, params)
}

func (r *postgresRepository) Create(ctx context.Context, params persistence.CreateUserParams) (persistence.User, error) {
	return r.store.CreateUser(ctx, params)
}

func (r *postgresRepository) Get(ctx context.Context, id uuid.UUID) (persistence.User, error) {
	return r.store.GetUser(ctx, id)
}

func (r *postgresRepository) Update(ctx context.Context, id uuid.UUID, params persistence.UpdateUserParams) (persistence.User, error) {
	return r.store.UpdateUser(ctx, id, params)
}

func (r *postgresRepository) UpdateFullName(ctx context.Context, id uuid.UUID, fullName string) (persistence.User, error) {
	return r.store.UpdateUserFullName(ctx, id, fullName)
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.store.DeleteUser(ctx, id)
}
