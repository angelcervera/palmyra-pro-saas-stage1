package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
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
	db *SpaceDB
}

// NewUserStore ensures the users table exists and returns a store instance.
func NewUserStore(ctx context.Context, db *SpaceDB) (*UserStore, error) {
	if db == nil {
		return nil, errors.New("space db is required")
	}

	return &UserStore{db: db}, nil
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
func (s *UserStore) CreateUser(ctx context.Context, space tenant.Space, params CreateUserParams) (User, error) {
	if params.UserID == uuid.Nil {
		return User{}, errors.New("user id is required")
	}

	var user User
	err := s.db.WithTenant(ctx, space, func(tx pgx.Tx) error {
		if err := ensureUserTable(ctx, tx); err != nil {
			return err
		}

		row := tx.QueryRow(ctx, fmt.Sprintf(`
        INSERT INTO %s (user_id, email, full_name)
        VALUES ($1, $2, $3)
        RETURNING user_id, email, full_name, created_at, updated_at
    `, UsersTable),
			params.UserID,
			strings.TrimSpace(params.Email),
			strings.TrimSpace(params.FullName),
		)

		scanned, scanErr := scanUser(row)
		if scanErr != nil {
			if isUniqueViolation(scanErr) {
				return ErrUserConflict
			}
			return scanErr
		}
		user = scanned
		return nil
	})
	if err != nil {
		return User{}, err
	}

	return user, nil
}

// ListUsers returns users matching the filters with pagination applied.
func (s *UserStore) ListUsers(ctx context.Context, space tenant.Space, params ListUsersParams) (ListUsersResult, error) {
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

	var result ListUsersResult
	err = s.db.WithTenant(ctx, space, func(tx pgx.Tx) error {
		if err := ensureUserTable(ctx, tx); err != nil {
			return err
		}

		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", UsersTable, whereSQL)
		var total int
		if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
			return fmt.Errorf("count users: %w", err)
		}

		result.TotalItems = total
		result.Users = []User{}
		if total == 0 {
			return nil
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

		rows, err := tx.Query(ctx, query, dataArgs...)
		if err != nil {
			return fmt.Errorf("list users: %w", err)
		}
		defer rows.Close()

		users := make([]User, 0)
		for rows.Next() {
			user, scanErr := scanUser(rows)
			if scanErr != nil {
				return fmt.Errorf("scan user: %w", scanErr)
			}
			users = append(users, user)
		}

		if err = rows.Err(); err != nil {
			return fmt.Errorf("iterate users: %w", err)
		}

		result.Users = users
		return nil
	})
	if err != nil {
		return ListUsersResult{}, err
	}

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
func (s *UserStore) GetUser(ctx context.Context, space tenant.Space, id uuid.UUID) (User, error) {
	var user User
	err := s.db.WithTenant(ctx, space, func(tx pgx.Tx) error {
		if err := ensureUserTable(ctx, tx); err != nil {
			return err
		}

		row := tx.QueryRow(ctx, fmt.Sprintf(`
        SELECT user_id, email, full_name, created_at, updated_at
        FROM %s WHERE user_id = $1
    `, UsersTable), id)

		scanned, scanErr := scanUser(row)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return ErrUserNotFound
			}
			return scanErr
		}
		user = scanned
		return nil
	})
	if err != nil {
		return User{}, err
	}

	return user, nil
}

// UpdateUserParams represents admin-editable fields.
type UpdateUserParams struct {
	FullName *string
}

// UpdateUser applies the provided fields and returns the updated record.
func (s *UserStore) UpdateUser(ctx context.Context, space tenant.Space, id uuid.UUID, params UpdateUserParams) (User, error) {
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

	var user User
	err := s.db.WithTenant(ctx, space, func(tx pgx.Tx) error {
		if err := ensureUserTable(ctx, tx); err != nil {
			return err
		}

		query := fmt.Sprintf(`
        UPDATE %s
        SET %s, updated_at = NOW()
        WHERE user_id = $%d
        RETURNING user_id, email, full_name, created_at, updated_at
    `, UsersTable, strings.Join(setParts, ", "), len(args))

		row := tx.QueryRow(ctx, query, args...)

		scanned, scanErr := scanUser(row)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return ErrUserNotFound
			}
			if isUniqueViolation(scanErr) {
				return ErrUserConflict
			}
			return scanErr
		}
		user = scanned
		return nil
	})
	if err != nil {
		return User{}, err
	}

	return user, nil
}

// UpdateUserFullName updates only the full name for the given user id.
func (s *UserStore) UpdateUserFullName(ctx context.Context, space tenant.Space, id uuid.UUID, fullName string) (User, error) {
	var user User
	err := s.db.WithTenant(ctx, space, func(tx pgx.Tx) error {
		if err := ensureUserTable(ctx, tx); err != nil {
			return err
		}

		row := tx.QueryRow(ctx, fmt.Sprintf(`
        UPDATE %s
        SET full_name = $1, updated_at = NOW()
        WHERE user_id = $2
        RETURNING user_id, email, full_name, created_at, updated_at
    `, UsersTable), strings.TrimSpace(fullName), id)

		scanned, scanErr := scanUser(row)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return ErrUserNotFound
			}
			if isUniqueViolation(scanErr) {
				return ErrUserConflict
			}
			return scanErr
		}

		user = scanned
		return nil
	})
	if err != nil {
		return User{}, err
	}

	return user, nil
}

// DeleteUser removes a user by identifier.
func (s *UserStore) DeleteUser(ctx context.Context, space tenant.Space, id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrUserNotFound
	}

	return s.db.WithTenant(ctx, space, func(tx pgx.Tx) error {
		if err := ensureUserTable(ctx, tx); err != nil {
			return err
		}

		tag, err := tx.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE user_id = $1`, UsersTable), id)
		if err != nil {
			return fmt.Errorf("delete user: %w", err)
		}

		if tag.RowsAffected() == 0 {
			return ErrUserNotFound
		}

		return nil
	})
}

func scanUser(row pgx.Row) (User, error) {
	var user User

	if err := row.Scan(&user.UserID, &user.Email, &user.FullName, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return User{}, err
	}

	return user, nil
}

func ensureUserTable(ctx context.Context, tx pgx.Tx) error {
	stmt := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    user_id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    full_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`, UsersTable)

	indexStmt := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s_created_at_idx ON %s(created_at DESC);`, UsersTable, UsersTable)

	if _, err := tx.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("ensure users table: %w", err)
	}
	if _, err := tx.Exec(ctx, indexStmt); err != nil {
		return fmt.Errorf("ensure users index: %w", err)
	}
	return nil
}
