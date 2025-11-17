package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const UsersTable = "users"

// User represents a row in the users table.
type User struct {
	UserID    uuid.UUID `db:"user_id" json:"userId"`
	Email     string    `db:"email" json:"email"`
	FullName  string    `db:"full_name" json:"fullName"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

var (
	// ErrUserNotFound indicates a missing user record.
	ErrUserNotFound = errors.New("user not found")
	// ErrUserConflict indicates a uniqueness violation (e.g., duplicated email).
	ErrUserConflict = errors.New("user conflict")
)

// UserStore exposes persistence helpers for the users table.
type UserStore struct {
	pool *pgxpool.Pool
}

// NewUserStore ensures the users table exists and returns a store instance.
func NewUserStore(ctx context.Context, pool *pgxpool.Pool) (*UserStore, error) {
	if pool == nil {
		return nil, errors.New("pool is required")
	}

	return &UserStore{pool: pool}, nil
}

// ListUsersParams captures filters and pagination for ListUsers.
type ListUsersParams struct {
	Page     int
	PageSize int
	Sort     *string
	Email    *string
}

// ListUsersResult includes the rows and the total count for pagination metadata.
type ListUsersResult struct {
	Users      []User
	TotalItems int
}

// CreateUserParams captures the fields required to insert a new user record.
type CreateUserParams struct {
	UserID   uuid.UUID
	Email    string
	FullName string
}

// CreateUser inserts a new user and returns the persisted record.
func (s *UserStore) CreateUser(ctx context.Context, params CreateUserParams) (User, error) {
	if params.UserID == uuid.Nil {
		return User{}, errors.New("user id is required")
	}

	row := s.pool.QueryRow(ctx, fmt.Sprintf(`
        INSERT INTO %s (user_id, email, full_name)
        VALUES ($1, $2, $3)
        RETURNING user_id, email, full_name, created_at, updated_at
    `, UsersTable),
		params.UserID,
		strings.TrimSpace(params.Email),
		strings.TrimSpace(params.FullName),
	)

	user, err := scanUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrUserConflict
		}
		return User{}, err
	}

	return user, nil
}

// ListUsers returns users matching the filters with pagination applied.
func (s *UserStore) ListUsers(ctx context.Context, params ListUsersParams) (ListUsersResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	whereParts := []string{"1=1"}
	var args []any

	if params.Email != nil && strings.TrimSpace(*params.Email) != "" {
		email := strings.TrimSpace(*params.Email)
		args = append(args, "%"+strings.ToLower(email)+"%")
		whereParts = append(whereParts, fmt.Sprintf("LOWER(email) LIKE $%d", len(args)))
	}

	whereSQL := strings.Join(whereParts, " AND ")

	orderSQL, err := buildUserOrderBy(params.Sort)
	if err != nil {
		return ListUsersResult{}, err
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", UsersTable, whereSQL)
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return ListUsersResult{}, fmt.Errorf("count users: %w", err)
	}

	result := ListUsersResult{Users: []User{}, TotalItems: total}
	if total == 0 {
		return result, nil
	}

	limit := params.PageSize
	offset := (params.Page - 1) * params.PageSize

	dataArgs := append([]any{}, args...)
	dataArgs = append(dataArgs, limit, offset)

	query := fmt.Sprintf(`
        SELECT user_id, email, full_name, created_at, updated_at
        FROM %s
        WHERE %s
        %s
        LIMIT $%d OFFSET $%d
    `, UsersTable, whereSQL, orderSQL, len(dataArgs)-1, len(dataArgs))

	rows, err := s.pool.Query(ctx, query, dataArgs...)
	if err != nil {
		return ListUsersResult{}, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		user, scanErr := scanUser(rows)
		if scanErr != nil {
			return ListUsersResult{}, fmt.Errorf("scan user: %w", scanErr)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return ListUsersResult{}, fmt.Errorf("iterate users: %w", err)
	}

	result.Users = users
	return result, nil
}

func buildUserOrderBy(sort *string) (string, error) {
	const defaultOrder = "ORDER BY created_at DESC"
	if sort == nil || strings.TrimSpace(*sort) == "" {
		return defaultOrder, nil
	}

	fields := strings.Split(strings.TrimSpace(*sort), ",")
	orderClauses := make([]string, 0, len(fields))
	mapping := map[string]string{
		"email":     "email",
		"fullName":  "full_name",
		"createdAt": "created_at",
		"updatedAt": "updated_at",
	}

	for _, raw := range fields {
		f := strings.TrimSpace(raw)
		if f == "" {
			continue
		}

		direction := "ASC"
		if strings.HasPrefix(f, "-") {
			direction = "DESC"
			f = strings.TrimPrefix(f, "-")
		}

		column, ok := mapping[f]
		if !ok {
			return "", fmt.Errorf("unsupported sort field %q", f)
		}

		orderClauses = append(orderClauses, fmt.Sprintf("%s %s", column, direction))
	}

	if len(orderClauses) == 0 {
		return defaultOrder, nil
	}

	return "ORDER BY " + strings.Join(orderClauses, ", "), nil
}

// GetUser returns a single user by identifier.
func (s *UserStore) GetUser(ctx context.Context, id uuid.UUID) (User, error) {
	row := s.pool.QueryRow(ctx, fmt.Sprintf(`
        SELECT user_id, email, full_name, created_at, updated_at
        FROM %s WHERE user_id = $1
    `, UsersTable), id)

	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}

	return user, nil
}

// UpdateUserParams represents admin-editable fields.
type UpdateUserParams struct {
	FullName *string
}

// UpdateUser applies the provided fields and returns the updated record.
func (s *UserStore) UpdateUser(ctx context.Context, id uuid.UUID, params UpdateUserParams) (User, error) {
	setParts := []string{}
	var args []any

	if params.FullName != nil {
		args = append(args, strings.TrimSpace(*params.FullName))
		setParts = append(setParts, fmt.Sprintf("full_name = $%d", len(args)))
	}

	if len(setParts) == 0 {
		return User{}, errors.New("no fields to update")
	}

	args = append(args, id)

	query := fmt.Sprintf(`
        UPDATE %s
        SET %s, updated_at = NOW()
        WHERE user_id = $%d
        RETURNING user_id, email, full_name, created_at, updated_at
    `, UsersTable, strings.Join(setParts, ", "), len(args))

	row := s.pool.QueryRow(ctx, query, args...)

	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		if isUniqueViolation(err) {
			return User{}, ErrUserConflict
		}
		return User{}, err
	}

	return user, nil
}

// UpdateUserFullName updates only the full name for the given user id.
func (s *UserStore) UpdateUserFullName(ctx context.Context, id uuid.UUID, fullName string) (User, error) {
	row := s.pool.QueryRow(ctx, fmt.Sprintf(`
        UPDATE %s
        SET full_name = $1, updated_at = NOW()
        WHERE user_id = $2
        RETURNING user_id, email, full_name, created_at, updated_at
    `, UsersTable), strings.TrimSpace(fullName), id)

	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		if isUniqueViolation(err) {
			return User{}, ErrUserConflict
		}
		return User{}, err
	}

	return user, nil
}

// DeleteUser removes a user by identifier.
func (s *UserStore) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrUserNotFound
	}

	tag, err := s.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE user_id = $1`, UsersTable), id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

func scanUser(row pgx.Row) (User, error) {
	var user User

	if err := row.Scan(&user.UserID, &user.Email, &user.FullName, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return User{}, err
	}

	return user, nil
}
