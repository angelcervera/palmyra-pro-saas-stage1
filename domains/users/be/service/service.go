package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/users/be/repo"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
)

// FieldErrors maps request fields to validation issues.
type FieldErrors map[string][]string

// ValidationError is returned when the input payload is invalid.
type ValidationError struct {
	Fields FieldErrors
}

func (v *ValidationError) Error() string {
	return "validation error"
}

// Domain sentinel errors.
var (
	ErrNotFound = errors.New("user not found")
	ErrConflict = errors.New("user conflict")
)

// User represents the domain view of a user record.
type User struct {
	ID        uuid.UUID
	Email     string
	FullName  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListOptions controls filtering and pagination.
type ListOptions struct {
	Email    *string
	Page     int
	PageSize int
	Sort     *string
}

// ListResult wraps a page of users with pagination metadata.
type ListResult struct {
	Users      []User
	Page       int
	PageSize   int
	TotalItems int
	TotalPages int
}

// CreateInput represents the payload required to create a new user.
type CreateInput struct {
	Email    string
	FullName string
}

// UpdateInput encapsulates fields that can be modified by administrators.
type UpdateInput struct {
	FullName *string
}

// UpdateSelfInput encapsulates fields that the authenticated user can modify.
type UpdateSelfInput struct {
	FullName *string
}

// Service defines the business operations for the users domain.
type Service interface {
	Create(ctx context.Context, input CreateInput) (User, error)
	List(ctx context.Context, opts ListOptions) (ListResult, error)
	Get(ctx context.Context, id uuid.UUID) (User, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateInput) (User, error)
	UpdateSelf(ctx context.Context, id uuid.UUID, input UpdateSelfInput) (User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo repo.Repository
}

// New constructs a users Service instance backed by the provided repository.
func New(r repo.Repository) Service {
	if r == nil {
		panic("users repository is required")
	}
	return &service{repo: r}
}

func (s *service) List(ctx context.Context, opts ListOptions) (ListResult, error) {
	page := opts.Page
	if page < 1 {
		page = 1
	}
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	sortValue, sortErr := sanitizeSort(opts.Sort)
	if sortErr != nil {
		return ListResult{}, sortErr
	}

	repoParams := persistence.ListUsersParams{
		Page:     page,
		PageSize: pageSize,
		Sort:     sortValue,
	}

	if opts.Email != nil && strings.TrimSpace(*opts.Email) != "" {
		email := strings.TrimSpace(*opts.Email)
		repoParams.Email = &email
	}

	result, err := s.repo.List(ctx, repoParams)
	if err != nil {
		return ListResult{}, err
	}

	users := make([]User, 0, len(result.Users))
	for _, record := range result.Users {
		users = append(users, mapUser(record))
	}

	totalPages := 0
	if result.TotalItems > 0 {
		totalPages = (result.TotalItems + pageSize - 1) / pageSize
	}

	return ListResult{
		Users:      users,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: result.TotalItems,
		TotalPages: totalPages,
	}, nil
}

func (s *service) Create(ctx context.Context, input CreateInput) (User, error) {
	fieldErrors := FieldErrors{}

	email := strings.TrimSpace(input.Email)
	if email == "" {
		fieldErrors.add("email", "email is required")
	} else if !strings.Contains(email, "@") {
		fieldErrors.add("email", "email must contain '@'")
	}

	fullName := strings.TrimSpace(input.FullName)
	if fullName == "" {
		fieldErrors.add("fullName", "fullName is required")
	}

	if len(fieldErrors) > 0 {
		return User{}, &ValidationError{Fields: fieldErrors}
	}

	record, err := s.repo.Create(ctx, persistence.CreateUserParams{
		UserID:   uuid.New(),
		Email:    strings.ToLower(email),
		FullName: fullName,
	})
	if err != nil {
		return User{}, mapPersistenceError(err)
	}

	return mapUser(record), nil
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (User, error) {
	if id == uuid.Nil {
		return User{}, ErrNotFound
	}

	record, err := s.repo.Get(ctx, id)
	if err != nil {
		return User{}, mapPersistenceError(err)
	}

	return mapUser(record), nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (User, error) {
	if id == uuid.Nil {
		return User{}, ErrNotFound
	}

	params, err := s.buildUpdateParams(input)
	if err != nil {
		return User{}, err
	}

	record, repoErr := s.repo.Update(ctx, id, params)
	if repoErr != nil {
		return User{}, mapPersistenceError(repoErr)
	}

	return mapUser(record), nil
}

func (s *service) UpdateSelf(ctx context.Context, id uuid.UUID, input UpdateSelfInput) (User, error) {
	if id == uuid.Nil {
		return User{}, ErrNotFound
	}

	if input.FullName == nil {
		return User{}, newValidationError(map[string]string{"fullName": "fullName is required"})
	}

	fullName := strings.TrimSpace(*input.FullName)
	if fullName == "" {
		return User{}, newValidationError(map[string]string{"fullName": "fullName cannot be empty"})
	}

	record, err := s.repo.UpdateFullName(ctx, id, fullName)
	if err != nil {
		return User{}, mapPersistenceError(err)
	}

	return mapUser(record), nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNotFound
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return mapPersistenceError(err)
	}

	return nil
}

func (s *service) buildUpdateParams(input UpdateInput) (persistence.UpdateUserParams, error) {
	fieldErrors := FieldErrors{}
	params := persistence.UpdateUserParams{}
	fieldsSet := 0

	if input.FullName != nil {
		name := strings.TrimSpace(*input.FullName)
		if name == "" {
			fieldErrors.add("fullName", "fullName cannot be empty")
		} else {
			params.FullName = &name
			fieldsSet++
		}
	}

	if fieldsSet == 0 {
		fieldErrors.add("payload", "at least one field must be provided")
	}

	if len(fieldErrors) > 0 {
		return persistence.UpdateUserParams{}, &ValidationError{Fields: fieldErrors}
	}

	return params, nil
}

func sanitizeSort(sort *string) (*string, error) {
	if sort == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*sort)
	if trimmed == "" {
		return nil, nil
	}

	allowed := map[string]struct{}{
		"email":     {},
		"fullName":  {},
		"createdAt": {},
		"updatedAt": {},
	}

	for _, raw := range strings.Split(trimmed, ",") {
		field := strings.TrimSpace(raw)
		if field == "" {
			continue
		}
		if strings.HasPrefix(field, "-") {
			field = strings.TrimPrefix(field, "-")
		}
		if _, ok := allowed[field]; !ok {
			return nil, newValidationError(map[string]string{"sort": fmt.Sprintf("unsupported sort field %q", field)})
		}
	}

	return &trimmed, nil
}

func mapUser(record persistence.User) User {
	return User{
		ID:        record.UserID,
		Email:     record.Email,
		FullName:  record.FullName,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func mapPersistenceError(err error) error {
	switch {
	case errors.Is(err, persistence.ErrUserNotFound):
		return ErrNotFound
	case errors.Is(err, persistence.ErrUserConflict):
		return ErrConflict
	default:
		return err
	}
}

func newValidationError(fields map[string]string) error {
	fe := FieldErrors{}
	for key, message := range fields {
		fe.add(key, message)
	}
	return &ValidationError{Fields: fe}
}

func (f FieldErrors) add(field, message string) {
	if f == nil {
		return
	}
	f[field] = append(f[field], message)
}
