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

// ErrSchemaNotFound indicates the requested schema/version could not be located.
var ErrSchemaNotFound = errors.New("schema not found")

// SchemaRepositoryStore provides PostgreSQL-backed access to the schema_repository table.
type SchemaRepositoryStore struct {
	pool *pgxpool.Pool
}

// CreateSchemaParams defines the payload required to persist a schema version.
type CreateSchemaParams struct {
	SchemaID   uuid.UUID
	Version    SemanticVersion
	Definition SchemaDefinition
	TableName  string
	Slug       string
	CategoryID uuid.UUID
	Activate   bool
}

// NewSchemaRepositoryStore ensures the schema repository table exists and returns a store instance.
func NewSchemaRepositoryStore(ctx context.Context, pool *pgxpool.Pool) (*SchemaRepositoryStore, error) {
	if pool == nil {
		return nil, errors.New("pool is required")
	}

	return &SchemaRepositoryStore{pool: pool}, nil
}

// CreateOrUpdateSchema persists the provided schema definition and optionally activates it.
func (s *SchemaRepositoryStore) CreateOrUpdateSchema(ctx context.Context, params CreateSchemaParams) (SchemaRecord, error) {
	if params.SchemaID == uuid.Nil {
		return SchemaRecord{}, errors.New("schema id is required")
	}

	if len(params.Definition) == 0 {
		return SchemaRecord{}, errors.New("schema definition is required")
	}

	if params.CategoryID == uuid.Nil {
		return SchemaRecord{}, errors.New("category id is required")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return SchemaRecord{}, fmt.Errorf("begin schema tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	tableName, err := s.resolveSchemaTableName(ctx, tx, params.SchemaID, params.TableName)
	if err != nil {
		return SchemaRecord{}, err
	}

	slug, err := s.resolveSchemaSlug(ctx, tx, params.SchemaID, params.Slug)
	if err != nil {
		return SchemaRecord{}, err
	}

	if params.Activate {
		if _, err = tx.Exec(ctx, `
			UPDATE schema_repository
			SET is_active = FALSE
			WHERE schema_id = $1 AND is_soft_deleted = FALSE
		`, params.SchemaID); err != nil {
			return SchemaRecord{}, fmt.Errorf("deactivate previous schema versions: %w", err)
		}
	}

	if _, err = tx.Exec(ctx, `
        INSERT INTO schema_repository (
            schema_id, schema_version, schema_definition, table_name, slug, category_id, is_active, is_soft_deleted, created_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, FALSE, NOW()
        )
        ON CONFLICT (schema_id, schema_version)
        DO UPDATE
        SET schema_definition = EXCLUDED.schema_definition,
            is_soft_deleted = FALSE,
            is_active = EXCLUDED.is_active,
            table_name = EXCLUDED.table_name,
            slug = EXCLUDED.slug,
            category_id = EXCLUDED.category_id
    `, params.SchemaID, params.Version.String(), []byte(params.Definition), tableName, slug, params.CategoryID, params.Activate); err != nil {
		return SchemaRecord{}, fmt.Errorf("upsert schema: %w", err)
	}

	row := tx.QueryRow(ctx, `
        SELECT schema_id, schema_version, category_id, table_name, slug, schema_definition, created_at, is_soft_deleted, is_active
        FROM schema_repository
        WHERE schema_id = $1 AND schema_version = $2
    `, params.SchemaID, params.Version.String())

	record, err := scanSchemaRecord(row)
	if err != nil {
		return SchemaRecord{}, fmt.Errorf("fetch new schema: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return SchemaRecord{}, fmt.Errorf("commit schema tx: %w", err)
	}

	return record, nil
}

// GetSchemaByVersion retrieves a specific schema version.
func (s *SchemaRepositoryStore) GetSchemaByVersion(ctx context.Context, schemaID uuid.UUID, version SemanticVersion) (SchemaRecord, error) {
	row := s.pool.QueryRow(ctx, `
        SELECT schema_id, schema_version, category_id, table_name, slug, schema_definition, created_at, is_soft_deleted, is_active
        FROM schema_repository
        WHERE schema_id = $1 AND schema_version = $2 AND is_soft_deleted = FALSE
    `, schemaID, version.String())

	record, err := scanSchemaRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SchemaRecord{}, ErrSchemaNotFound
		}
		return SchemaRecord{}, err
	}

	return record, nil
}

// GetActiveSchema fetches the currently active schema for the provided identifier.
func (s *SchemaRepositoryStore) GetActiveSchema(ctx context.Context, schemaID uuid.UUID) (SchemaRecord, error) {
	row := s.pool.QueryRow(ctx, `
        SELECT schema_id, schema_version, category_id, table_name, slug, schema_definition, created_at, is_soft_deleted, is_active
        FROM schema_repository
        WHERE schema_id = $1 AND is_active = TRUE AND is_soft_deleted = FALSE
    `, schemaID)

	record, err := scanSchemaRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SchemaRecord{}, ErrSchemaNotFound
		}
		return SchemaRecord{}, err
	}

	return record, nil
}

// ListSchemas returns every non-deleted schema version for the identifier ordered by version chronology.
func (s *SchemaRepositoryStore) ListSchemas(ctx context.Context, schemaID uuid.UUID) ([]SchemaRecord, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT schema_id, schema_version, category_id, table_name, slug, schema_definition, created_at, is_soft_deleted, is_active
        FROM schema_repository
        WHERE schema_id = $1
        ORDER BY created_at DESC
    `, schemaID)
	if err != nil {
		return nil, fmt.Errorf("list schemas: %w", err)
	}
	defer rows.Close()

	var records []SchemaRecord
	for rows.Next() {
		record, scanErr := scanSchemaRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schemas: %w", err)
	}

	return records, nil
}

// ListAllSchemaVersions returns every schema version across all schema identifiers.
func (s *SchemaRepositoryStore) ListAllSchemaVersions(ctx context.Context, includeInactive bool) ([]SchemaRecord, error) {
	query := `
        SELECT schema_id, schema_version, category_id, table_name, slug, schema_definition, created_at, is_soft_deleted, is_active
        FROM schema_repository
        WHERE $1::bool = TRUE OR is_active = TRUE
        ORDER BY created_at DESC
    `

	rows, err := s.pool.Query(ctx, query, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("list all schema versions: %w", err)
	}
	defer rows.Close()

	var records []SchemaRecord
	for rows.Next() {
		record, scanErr := scanSchemaRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		if !includeInactive && !record.IsActive {
			continue
		}
		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema versions: %w", err)
	}

	return records, nil
}

// GetActiveSchemaByTableName fetches the active schema associated with the provided table name.
func (s *SchemaRepositoryStore) GetActiveSchemaByTableName(ctx context.Context, tableName string) (SchemaRecord, error) {
	normalized, err := normalizeTableName(tableName)
	if err != nil {
		return SchemaRecord{}, err
	}

	row := s.pool.QueryRow(ctx, `
		SELECT schema_id, schema_version, category_id, table_name, slug, schema_definition, created_at, is_soft_deleted, is_active
		FROM schema_repository
		WHERE table_name = $1 AND is_active = TRUE AND is_soft_deleted = FALSE
		LIMIT 1
	`, normalized)

	record, err := scanSchemaRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SchemaRecord{}, ErrSchemaNotFound
		}
		return SchemaRecord{}, err
	}

	return record, nil
}

// GetLatestSchemaBySlug returns the most recent schema record that matches the provided slug.
func (s *SchemaRepositoryStore) GetLatestSchemaBySlug(ctx context.Context, slug string) (SchemaRecord, error) {
	row := s.pool.QueryRow(ctx, `
        SELECT schema_id, schema_version, category_id, table_name, slug, schema_definition, created_at, is_soft_deleted, is_active
        FROM schema_repository
        WHERE slug = $1
        ORDER BY created_at DESC
        LIMIT 1
    `, slug)

	record, err := scanSchemaRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SchemaRecord{}, ErrSchemaNotFound
		}
		return SchemaRecord{}, err
	}

	return record, nil
}

// ActivateSchemaVersion toggles the target version as the active one (soft-deleting remains intact).
func (s *SchemaRepositoryStore) ActivateSchemaVersion(ctx context.Context, schemaID uuid.UUID, version SemanticVersion) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin activate schema tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err = tx.Exec(ctx, `
		UPDATE schema_repository
		SET is_active = FALSE
		WHERE schema_id = $1 AND is_soft_deleted = FALSE
	`, schemaID); err != nil {
		return fmt.Errorf("deactivate schemas: %w", err)
	}

	result, err := tx.Exec(ctx, `
		UPDATE schema_repository
		SET is_active = TRUE
		WHERE schema_id = $1 AND schema_version = $2 AND is_soft_deleted = FALSE
	`, schemaID, version.String())
	if err != nil {
		return fmt.Errorf("activate schema: %w", err)
	}

	affected := result.RowsAffected()
	if affected == 0 {
		return ErrSchemaNotFound
	}

	return tx.Commit(ctx)
}

// SoftDeleteSchema marks the provided schema version as deleted and deactivates it when needed.
// deletedAt is ignored because schema versions are immutable and only track creation timestamps.
func (s *SchemaRepositoryStore) SoftDeleteSchema(ctx context.Context, schemaID uuid.UUID, version SemanticVersion, _ time.Time) error {
	result, err := s.pool.Exec(ctx, `
		UPDATE schema_repository
		SET is_soft_deleted = TRUE,
		    is_active = FALSE
		WHERE schema_id = $1 AND schema_version = $2 AND is_soft_deleted = FALSE
	`, schemaID, version.String())
	if err != nil {
		return fmt.Errorf("soft delete schema: %w", err)
	}

	affected := result.RowsAffected()
	if affected == 0 {
		return ErrSchemaNotFound
	}

	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSchemaRecord(scanner rowScanner) (SchemaRecord, error) {
	var (
		schemaID      uuid.UUID
		versionText   string
		categoryID    uuid.UUID
		tableName     string
		slug          string
		rawDef        []byte
		createdAt     time.Time
		isSoftDeleted bool
		isActive      bool
	)

	if err := scanner.Scan(&schemaID, &versionText, &categoryID, &tableName, &slug, &rawDef, &createdAt, &isSoftDeleted, &isActive); err != nil {
		return SchemaRecord{}, err
	}

	version, err := ParseSemanticVersion(versionText)
	if err != nil {
		return SchemaRecord{}, fmt.Errorf("parse schema version %q: %w", versionText, err)
	}

	return SchemaRecord{
		SchemaID:         schemaID,
		SchemaVersion:    version,
		SchemaDefinition: SchemaDefinition(rawDef),
		TableName:        tableName,
		Slug:             slug,
		CategoryID:       categoryID,
		CreatedAt:        createdAt,
		IsSoftDeleted:    isSoftDeleted,
		IsActive:         isActive,
	}, nil
}

func (s *SchemaRepositoryStore) resolveSchemaTableName(ctx context.Context, tx pgx.Tx, schemaID uuid.UUID, candidate string) (string, error) {
	candidate = strings.TrimSpace(candidate)
	if candidate != "" {
		normalized, err := normalizeTableName(candidate)
		if err != nil {
			return "", err
		}
		candidate = normalized
	}

	row := tx.QueryRow(ctx, `
		SELECT table_name
		FROM schema_repository
		WHERE schema_id = $1
		LIMIT 1
	`, schemaID)

	var existing string
	err := row.Scan(&existing)
	switch {
	case err == nil:
		if existing == "" {
			return "", fmt.Errorf("schema %s has no table name recorded", schemaID)
		}
		if candidate != "" && candidate != existing {
			return "", fmt.Errorf("table name for schema %s cannot be modified", schemaID)
		}
		return existing, nil
	case errors.Is(err, pgx.ErrNoRows):
		if candidate == "" {
			return "", errors.New("table name is required when creating a new schema")
		}
		return candidate, nil
	default:
		return "", fmt.Errorf("resolve schema table name: %w", err)
	}
}

func (s *SchemaRepositoryStore) resolveSchemaSlug(ctx context.Context, tx pgx.Tx, schemaID uuid.UUID, candidate string) (string, error) {
	candidate = strings.TrimSpace(candidate)
	if candidate != "" {
		normalized, err := NormalizeSlug(candidate)
		if err != nil {
			return "", err
		}
		candidate = normalized
	}

	row := tx.QueryRow(ctx, `
		SELECT slug
		FROM schema_repository
		WHERE schema_id = $1
		LIMIT 1
	`, schemaID)

	var existing string
	err := row.Scan(&existing)
	switch {
	case err == nil:
		if existing == "" {
			return "", fmt.Errorf("schema %s has no slug recorded", schemaID)
		}
		if candidate != "" && candidate != existing {
			return "", fmt.Errorf("slug for schema %s cannot be modified", schemaID)
		}
		return existing, nil
	case errors.Is(err, pgx.ErrNoRows):
		if candidate == "" {
			return "", errors.New("slug is required when creating a new schema")
		}
		return candidate, nil
	default:
		return "", fmt.Errorf("resolve schema slug: %w", err)
	}
}
