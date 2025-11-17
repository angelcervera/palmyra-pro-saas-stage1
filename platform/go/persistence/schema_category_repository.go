package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const SchemaCategoryTable = "schema_categories"

type SchemaCategory struct {
	CategoryID       uuid.UUID  `db:"category_id" json:"categoryId"`
	ParentCategoryID *uuid.UUID `db:"parent_category_id" json:"parentCategoryId,omitempty"`
	Name             string     `db:"name" json:"name"`
	Slug             string     `db:"slug" json:"slug"`
	Description      *string    `db:"description" json:"description,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updatedAt"`
	DeletedAt        *time.Time `db:"deleted_at,omitempty" json:"deletedAt,omitempty"`
}

type SchemaCategoryStore struct {
	pool *pgxpool.Pool
}

func NewSchemaCategoryStore(ctx context.Context, pool *pgxpool.Pool) (*SchemaCategoryStore, error) {
	if pool == nil {
		return nil, errors.New("pool is required")
	}

	return &SchemaCategoryStore{pool: pool}, nil
}

type CreateSchemaCategoryParams struct {
	CategoryID       uuid.UUID
	ParentCategoryID *uuid.UUID
	Name             string
	Slug             string
	Description      *string
}

var (
	// ErrSchemaCategoryConflict indicates a uniqueness violation (name or slug already exists).
	ErrSchemaCategoryConflict = errors.New("schema category conflict")
)

func (s *SchemaCategoryStore) CreateSchemaCategory(ctx context.Context, params CreateSchemaCategoryParams) (SchemaCategory, error) {
	if params.CategoryID == uuid.Nil {
		return SchemaCategory{}, errors.New("category id is required")
	}
	if strings.TrimSpace(params.Name) == "" {
		return SchemaCategory{}, errors.New("category name is required")
	}
	if params.ParentCategoryID != nil && *params.ParentCategoryID == params.CategoryID {
		return SchemaCategory{}, errors.New("category cannot reference itself as parent")
	}

	slug, err := NormalizeSlug(params.Slug)
	if err != nil {
		return SchemaCategory{}, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return SchemaCategory{}, fmt.Errorf("begin category tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var existingSlug string
	switch scanErr := tx.QueryRow(ctx, `
		SELECT slug
		FROM schema_categories
		WHERE category_id = $1
	`, params.CategoryID).Scan(&existingSlug); {
	case scanErr == nil:
		if existingSlug != slug {
			return SchemaCategory{}, fmt.Errorf("slug for category %s cannot be modified", params.CategoryID)
		}
	case errors.Is(scanErr, pgx.ErrNoRows):
		// new record, continue
	default:
		return SchemaCategory{}, fmt.Errorf("check existing category slug: %w", scanErr)
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO schema_categories (
			category_id, parent_category_id, name, slug, description, created_at, updated_at, deleted_at
		) VALUES (
			$1, $2, $3, $4, $5, NOW(), NOW(), NULL
		)
	`, params.CategoryID, params.ParentCategoryID, params.Name, slug, params.Description); err != nil {
		if isUniqueViolation(err) {
			return SchemaCategory{}, ErrSchemaCategoryConflict
		}
		return SchemaCategory{}, fmt.Errorf("insert schema category: %w", err)
	}

	row := tx.QueryRow(ctx, `
		SELECT category_id, parent_category_id, name, slug, description, created_at, updated_at, deleted_at
		FROM schema_categories
		WHERE category_id = $1
	`, params.CategoryID)

	category, err := scanSchemaCategory(row)
	if err != nil {
		return SchemaCategory{}, fmt.Errorf("fetch schema category: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return SchemaCategory{}, fmt.Errorf("commit schema category tx: %w", err)
	}

	return category, nil
}

func (s *SchemaCategoryStore) GetSchemaCategory(ctx context.Context, categoryID uuid.UUID) (SchemaCategory, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT category_id, parent_category_id, name, slug, description, created_at, updated_at, deleted_at
		FROM schema_categories
		WHERE category_id = $1 AND deleted_at IS NULL
	`, categoryID)

	category, err := scanSchemaCategory(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SchemaCategory{}, ErrSchemaNotFound
		}
		return SchemaCategory{}, err
	}

	return category, nil
}

func (s *SchemaCategoryStore) ListSchemaCategories(ctx context.Context, includeDeleted bool) ([]SchemaCategory, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT category_id, parent_category_id, name, slug, description, created_at, updated_at, deleted_at
		FROM schema_categories
		WHERE ($1::bool = TRUE OR deleted_at IS NULL)
		ORDER BY created_at ASC
	`, includeDeleted)
	if err != nil {
		return nil, fmt.Errorf("list schema categories: %w", err)
	}
	defer rows.Close()

	var categories []SchemaCategory
	for rows.Next() {
		category, scanErr := scanSchemaCategory(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema categories: %w", err)
	}

	return categories, nil
}

func (s *SchemaCategoryStore) SoftDeleteSchemaCategory(ctx context.Context, categoryID uuid.UUID, deletedAt time.Time) error {
	if deletedAt.IsZero() {
		deletedAt = time.Now().UTC()
	}

	result, err := s.pool.Exec(ctx, `
		UPDATE schema_categories
		SET deleted_at = $2,
		    updated_at = NOW()
		WHERE category_id = $1 AND deleted_at IS NULL
	`, categoryID, deletedAt)
	if err != nil {
		return fmt.Errorf("soft delete schema category: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSchemaNotFound
	}

	return nil
}

type UpdateSchemaCategoryParams struct {
	ParentCategoryID *uuid.UUID
	Name             *string
	Description      *string
	Slug             *string
}

func (s *SchemaCategoryStore) UpdateSchemaCategory(ctx context.Context, categoryID uuid.UUID, params UpdateSchemaCategoryParams) (SchemaCategory, error) {
	if categoryID == uuid.Nil {
		return SchemaCategory{}, errors.New("category id is required")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return SchemaCategory{}, fmt.Errorf("begin update schema category tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	row := tx.QueryRow(ctx, `
		SELECT category_id, parent_category_id, name, slug, description, created_at, updated_at, deleted_at
		FROM schema_categories
		WHERE category_id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, categoryID)

	current, err := scanSchemaCategory(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SchemaCategory{}, ErrSchemaNotFound
		}
		return SchemaCategory{}, fmt.Errorf("load schema category: %w", err)
	}

	parentID := current.ParentCategoryID
	if params.ParentCategoryID != nil {
		if *params.ParentCategoryID == categoryID {
			return SchemaCategory{}, errors.New("category cannot reference itself as parent")
		}
		parentID = params.ParentCategoryID
	}

	name := current.Name
	if params.Name != nil {
		trimmed := strings.TrimSpace(*params.Name)
		if trimmed == "" {
			return SchemaCategory{}, errors.New("category name is required")
		}
		name = trimmed
	}

	description := current.Description
	if params.Description != nil {
		description = params.Description
	}

	slug := current.Slug
	if params.Slug != nil {
		normalized, err := NormalizeSlug(*params.Slug)
		if err != nil {
			return SchemaCategory{}, err
		}
		slug = normalized
	}

	if _, err = tx.Exec(ctx, `
		UPDATE schema_categories
		SET parent_category_id = $2,
		    name = $3,
		    description = $4,
		    slug = $5,
		    updated_at = NOW()
		WHERE category_id = $1
	`, categoryID, parentID, name, description, slug); err != nil {
		if isUniqueViolation(err) {
			return SchemaCategory{}, ErrSchemaCategoryConflict
		}
		return SchemaCategory{}, fmt.Errorf("update schema category: %w", err)
	}

	row = tx.QueryRow(ctx, `
		SELECT category_id, parent_category_id, name, slug, description, created_at, updated_at, deleted_at
		FROM schema_categories
		WHERE category_id = $1
	`, categoryID)

	category, err := scanSchemaCategory(row)
	if err != nil {
		return SchemaCategory{}, fmt.Errorf("fetch updated schema category: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return SchemaCategory{}, fmt.Errorf("commit update schema category tx: %w", err)
	}

	return category, nil
}

func scanSchemaCategory(scanner rowScanner) (SchemaCategory, error) {
	var (
		categoryID       uuid.UUID
		parentCategoryID pgtype.UUID
		name             string
		slug             string
		description      pgtype.Text
		createdAt        time.Time
		updatedAt        time.Time
		deletedAt        pgtype.Timestamptz
	)

	if err := scanner.Scan(&categoryID, &parentCategoryID, &name, &slug, &description, &createdAt, &updatedAt, &deletedAt); err != nil {
		return SchemaCategory{}, err
	}

	var parentPtr *uuid.UUID
	if parentCategoryID.Valid {
		id, err := uuid.FromBytes(parentCategoryID.Bytes[:])
		if err != nil {
			return SchemaCategory{}, fmt.Errorf("parse parent category id: %w", err)
		}
		parentPtr = &id
	}

	var deletedPtr *time.Time
	if deletedAt.Valid {
		ts := deletedAt.Time
		deletedPtr = &ts
	}

	var descriptionPtr *string
	if description.Valid {
		desc := description.String
		descriptionPtr = &desc
	}

	return SchemaCategory{
		CategoryID:       categoryID,
		ParentCategoryID: parentPtr,
		Name:             name,
		Slug:             slug,
		Description:      descriptionPtr,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		DeletedAt:        deletedPtr,
	}, nil
}
