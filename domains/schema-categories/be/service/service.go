package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	domainrepo "github.com/zenGate-Global/palmyra-pro-saas/domains/schema-categories/be/repo"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/requesttrace"
)

// FieldErrors maps request fields to validation issues.
type FieldErrors map[string][]string

// ValidationError captures input validation problems surfaced by the service.
type ValidationError struct {
	Fields FieldErrors
}

func (v *ValidationError) Error() string {
	return "validation error"
}

// Domain-level error sentinel values.
var (
	ErrNotFound = errors.New("schema category not found")
	ErrConflict = errors.New("schema category conflict")
)

// Category represents a schema category managed by the domain service.
type Category struct {
	ID          uuid.UUID
	ParentID    *uuid.UUID
	Name        string
	Slug        string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// CreateInput defines the payload required to create a schema category.
type CreateInput struct {
	Name        string
	Slug        string
	ParentID    *uuid.UUID
	Description *string
}

// UpdateInput defines the fields that can be modified for an existing schema category.
type UpdateInput struct {
	Name        *string
	ParentID    *uuid.UUID
	Description *string
	Slug        *string
}

// Service exposes the schema categories domain operations.
type Service interface {
	List(ctx context.Context, audit requesttrace.AuditInfo, includeDeleted bool) ([]Category, error)
	Create(ctx context.Context, audit requesttrace.AuditInfo, input CreateInput) (Category, error)
	Get(ctx context.Context, audit requesttrace.AuditInfo, id uuid.UUID) (Category, error)
	Update(ctx context.Context, audit requesttrace.AuditInfo, id uuid.UUID, input UpdateInput) (Category, error)
	Delete(ctx context.Context, audit requesttrace.AuditInfo, id uuid.UUID) error
}

type service struct {
	repo domainrepo.Repository
	now  func() time.Time
}

// New builds a schema categories Service backed by the provided repository.
func New(repo domainrepo.Repository) Service {
	return &service{
		repo: repo,
		now:  time.Now,
	}
}

func (s *service) List(ctx context.Context, audit requesttrace.AuditInfo, includeDeleted bool) ([]Category, error) { //nolint:revive
	records, err := s.repo.List(ctx, includeDeleted)
	if err != nil {
		return nil, err
	}

	categories := make([]Category, 0, len(records))
	for _, record := range records {
		categories = append(categories, mapCategory(record))
	}

	return categories, nil
}

func (s *service) Create(ctx context.Context, audit requesttrace.AuditInfo, input CreateInput) (Category, error) { //nolint:revive
	if err := s.ensureParentExists(ctx, input.ParentID, uuid.Nil); err != nil {
		return Category{}, err
	}

	normalized, validationErr := s.validateCreateInput(input)
	if validationErr != nil {
		return Category{}, validationErr
	}

	params := persistence.CreateSchemaCategoryParams{
		CategoryID:       uuid.New(),
		ParentCategoryID: input.ParentID,
		Name:             normalized.name,
		Slug:             normalized.slug,
		Description:      input.Description,
	}

	record, err := s.repo.Create(ctx, params)
	if err != nil {
		if errors.Is(err, persistence.ErrSchemaCategoryConflict) {
			return Category{}, ErrConflict
		}
		return Category{}, err
	}

	return mapCategory(record), nil
}

func (s *service) Get(ctx context.Context, audit requesttrace.AuditInfo, id uuid.UUID) (Category, error) { //nolint:revive
	record, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, persistence.ErrSchemaNotFound) {
			return Category{}, ErrNotFound
		}
		return Category{}, err
	}

	return mapCategory(record), nil
}

func (s *service) Update(ctx context.Context, audit requesttrace.AuditInfo, id uuid.UUID, input UpdateInput) (Category, error) { //nolint:revive
	if id == uuid.Nil {
		return Category{}, ErrNotFound
	}

	if err := s.ensureParentExists(ctx, input.ParentID, id); err != nil {
		return Category{}, err
	}

	normalized, validationErr := s.validateUpdateInput(input)
	if validationErr != nil {
		return Category{}, validationErr
	}

	params := persistence.UpdateSchemaCategoryParams{
		ParentCategoryID: normalized.parentID,
		Name:             normalized.name,
		Description:      input.Description,
		Slug:             normalized.slug,
	}

	record, err := s.repo.Update(ctx, id, params)
	if err != nil {
		switch {
		case errors.Is(err, persistence.ErrSchemaNotFound):
			return Category{}, ErrNotFound
		case errors.Is(err, persistence.ErrSchemaCategoryConflict):
			return Category{}, ErrConflict
		default:
			return Category{}, err
		}
	}

	return mapCategory(record), nil
}

func (s *service) Delete(ctx context.Context, audit requesttrace.AuditInfo, id uuid.UUID) error { //nolint:revive
	if id == uuid.Nil {
		return ErrNotFound
	}

	if err := s.repo.Delete(ctx, id, s.now().UTC()); err != nil {
		if errors.Is(err, persistence.ErrSchemaNotFound) {
			return ErrNotFound
		}
		return err
	}

	return nil
}

type normalizedCreateInput struct {
	name string
	slug string
}

type normalizedUpdateInput struct {
	name     *string
	parentID *uuid.UUID
	slug     *string
}

func (s *service) validateCreateInput(input CreateInput) (normalizedCreateInput, error) {
	errs := FieldErrors{}

	trimmedName := strings.TrimSpace(input.Name)
	if trimmedName == "" {
		errs.add("name", "name is required")
	}

	slug, err := persistence.NormalizeSlug(input.Slug)
	if err != nil {
		errs.add("slug", err.Error())
	}

	if len(errs) > 0 {
		return normalizedCreateInput{}, &ValidationError{Fields: errs}
	}

	return normalizedCreateInput{name: trimmedName, slug: slug}, nil
}

func (s *service) validateUpdateInput(input UpdateInput) (normalizedUpdateInput, error) {
	errs := FieldErrors{}
	var normalized normalizedUpdateInput

	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			errs.add("name", "name is required")
		} else {
			normalized.name = &trimmed
		}
	}

	if input.ParentID != nil {
		normalized.parentID = input.ParentID
	}

	if input.Slug != nil {
		normalizedSlug, err := persistence.NormalizeSlug(*input.Slug)
		if err != nil {
			errs.add("slug", err.Error())
		} else {
			normalized.slug = &normalizedSlug
		}
	}

	if input.Name == nil && input.ParentID == nil && input.Description == nil && input.Slug == nil {
		errs.add("body", "at least one field must be provided")
	}

	if len(errs) > 0 {
		return normalizedUpdateInput{}, &ValidationError{Fields: errs}
	}

	return normalized, nil
}

func (s *service) ensureParentExists(ctx context.Context, parentID *uuid.UUID, currentID uuid.UUID) error {
	if parentID == nil {
		return nil
	}

	if *parentID == uuid.Nil {
		return &ValidationError{Fields: FieldErrors{"parentCategoryId": []string{"parentCategoryId must be a valid UUID"}}}
	}

	if *parentID == currentID {
		return &ValidationError{Fields: FieldErrors{"parentCategoryId": []string{"parent category cannot reference itself"}}}
	}

	if _, err := s.repo.Get(ctx, *parentID); err != nil {
		if errors.Is(err, persistence.ErrSchemaNotFound) {
			return &ValidationError{Fields: FieldErrors{"parentCategoryId": []string{"parent category not found"}}}
		}
		return err
	}

	return nil
}

func mapCategory(record persistence.SchemaCategory) Category {
	return Category{
		ID:          record.CategoryID,
		ParentID:    record.ParentCategoryID,
		Name:        record.Name,
		Slug:        record.Slug,
		Description: record.Description,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
		DeletedAt:   record.DeletedAt,
	}
}

func (f FieldErrors) add(field, message string) {
	if _, ok := f[field]; !ok {
		f[field] = []string{message}
		return
	}
	f[field] = append(f[field], message)
}
