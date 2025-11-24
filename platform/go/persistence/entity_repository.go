package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// ErrEntityNotFound indicates the requested entity (or version) does not exist.
var ErrEntityNotFound = errors.New("entity not found")

// ErrEntityAlreadyExists indicates an entity is being created with an identifier that already exists.
var ErrEntityAlreadyExists = errors.New("entity already exists")

// SchemaResolver exposes the subset of schema store operations needed by the entity repository.
type SchemaResolver interface {
	GetActiveSchema(ctx context.Context, schemaID uuid.UUID) (SchemaRecord, error)
	GetSchemaByVersion(ctx context.Context, schemaID uuid.UUID, version SemanticVersion) (SchemaRecord, error)
}

// PayloadValidator validates JSON documents against schema definitions.
type PayloadValidator interface {
	Validate(ctx context.Context, schema SchemaRecord, payload []byte) error
}

// EntityRepositoryConfig provides the wiring required to manage a specific entity table.
type EntityRepositoryConfig struct {
	SchemaID uuid.UUID
}

// EntityRepository persists immutable entity documents with schema validation and versioning.
// tableName holds the raw schema-owned table (e.g. cards_entities) while tableIdent caches the quoted/sanitized identifier generated via pgx.Identifier to embed safely in SQL strings.
type EntityRepository struct {
	db         *SpaceDB
	schemas    SchemaResolver
	validator  PayloadValidator
	tableName  string
	schemaID   uuid.UUID
	tableIdent string
}

// EntityRecord mirrors the entity table shape, capturing every immutable version of a document.
type EntityRecord struct {
	EntityID      string          `json:"entityId"`
	EntityVersion SemanticVersion `json:"entityVersion"`
	SchemaID      uuid.UUID       `json:"schemaId"`
	SchemaVersion SemanticVersion `json:"schemaVersion"`
	Hash          string          `json:"hash"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"createdAt"`
	CreatedBy     *string         `json:"createdBy"`
	IsSoftDeleted bool            `json:"isSoftDeleted"`
	IsActive      bool            `json:"isActive"`
}

// CreateEntityParams defines the payload required to persist a brand-new entity.
type CreateEntityParams struct {
	EntityID      string
	SchemaVersion *SemanticVersion
	Payload       SchemaDefinition
	CreatedBy     *string
}

// UpdateEntityParams defines the payload required to add a new immutable version of an entity.
type UpdateEntityParams struct {
	EntityID      string
	SchemaVersion *SemanticVersion
	Payload       SchemaDefinition
	CreatedBy     *string
}

// CreateOrUpdateEntityParams unifies the payload for upserting immutable entity records.
// Slug is optional when updating an existing entity, but required when inserting a new one.
type CreateOrUpdateEntityParams struct {
	EntityID      string
	SchemaVersion *SemanticVersion
	Payload       SchemaDefinition
	CreatedBy     *string
}

// ListEntitiesParams defines filters when listing entities.
type ListEntitiesParams struct {
	OnlyActive     bool
	IncludeDeleted bool
	Limit          int
	Offset         int
	SortField      string
	SortOrder      string
}

// NewEntityRepository ensures the backing table exists and returns a repository instance.
func NewEntityRepository(ctx context.Context, db *SpaceDB, schemaStore SchemaResolver, validator PayloadValidator, cfg EntityRepositoryConfig) (*EntityRepository, error) {
	if db == nil {
		return nil, errors.New("space db is required")
	}
	if schemaStore == nil {
		return nil, errors.New("schema store is required")
	}
	if validator == nil {
		return nil, errors.New("payload validator is required")
	}
	if cfg.SchemaID == uuid.Nil {
		return nil, errors.New("schema id is required")
	}

	activeSchema, err := schemaStore.GetActiveSchema(ctx, cfg.SchemaID)
	if err != nil {
		return nil, fmt.Errorf("resolve active schema: %w", err)
	}
	if activeSchema.TableName == "" || !tableNamePattern.MatchString(activeSchema.TableName) {
		return nil, fmt.Errorf("schema %s has invalid table name %q", cfg.SchemaID, activeSchema.TableName)
	}

	repo := &EntityRepository{
		db:         db,
		schemas:    schemaStore,
		validator:  validator,
		tableName:  activeSchema.TableName,
		schemaID:   cfg.SchemaID,
		tableIdent: pgx.Identifier{activeSchema.TableName}.Sanitize(),
	}

	return repo, nil
}

// CreateEntity persists a new entity (version 1.0.0) after schema validation.
func (r *EntityRepository) CreateEntity(ctx context.Context, space tenant.Space, params CreateEntityParams) (EntityRecord, error) {
	entityID := strings.TrimSpace(params.EntityID)
	var err error
	if entityID == "" {
		entityID = uuid.NewString()
	} else {
		entityID, err = NormalizeEntityIdentifier(entityID)
		if err != nil {
			return EntityRecord{}, err
		}
	}

	if len(params.Payload) == 0 {
		return EntityRecord{}, errors.New("payload is required")
	}

	schemaRecord, err := r.resolveSchema(ctx, params.SchemaVersion)
	if err != nil {
		return EntityRecord{}, err
	}

	if err := r.validator.Validate(ctx, schemaRecord, params.Payload); err != nil {
		return EntityRecord{}, err
	}

	hash, err := computeJSONHash(params.Payload)
	if err != nil {
		return EntityRecord{}, fmt.Errorf("compute entity hash: %w", err)
	}

	var record EntityRecord
	err = r.db.WithTenant(ctx, space, func(tx pgx.Tx) error {
		if err := r.ensureEntityTable(ctx, tx); err != nil {
			return err
		}

		existsQuery := fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s WHERE entity_id = $1)`, r.tableIdent)
		var exists bool
		if err := tx.QueryRow(ctx, existsQuery, entityID).Scan(&exists); err != nil {
			return fmt.Errorf("check entity existence: %w", err)
		}
		if exists {
			return ErrEntityAlreadyExists
		}

		version := SemanticVersion{Major: 1, Minor: 0, Patch: 0}
		insertStmt := fmt.Sprintf(`
        INSERT INTO %s (
			entity_id, entity_version, schema_id, schema_version, payload, hash, is_active, is_soft_deleted, created_at, created_by
        ) VALUES (
			$1, $2, $3, $4, $5, $6, TRUE, FALSE, NOW(), $7
        )`, r.tableIdent)

		if _, err := tx.Exec(ctx, insertStmt, entityID, version.String(), schemaRecord.SchemaID, schemaRecord.VersionString(), []byte(params.Payload), hash, params.CreatedBy); err != nil {
			return fmt.Errorf("insert entity: %w", err)
		}

		selectStmt := fmt.Sprintf(`
	SELECT entity_id, entity_version, schema_id, schema_version, payload, hash, created_at, created_by, is_soft_deleted, is_active
FROM %s
WHERE entity_id = $1 AND entity_version = $2
`, r.tableIdent)

		row := tx.QueryRow(ctx, selectStmt, entityID, version.String())
		record, err = scanEntityRecord(row)
		if err != nil {
			return fmt.Errorf("fetch entity: %w", err)
		}
		return nil
	})
	if err != nil {
		return EntityRecord{}, err
	}

	return record, nil
}

// UpdateEntity creates a new immutable version of an existing entity, bumping the patch segment.
func (r *EntityRepository) UpdateEntity(ctx context.Context, space tenant.Space, params UpdateEntityParams) (EntityRecord, error) {
	entityID, err := NormalizeEntityIdentifier(params.EntityID)
	if err != nil {
		return EntityRecord{}, err
	}
	if len(params.Payload) == 0 {
		return EntityRecord{}, errors.New("payload is required")
	}

	schemaRecord, err := r.resolveSchema(ctx, params.SchemaVersion)
	if err != nil {
		return EntityRecord{}, err
	}

	if err := r.validator.Validate(ctx, schemaRecord, params.Payload); err != nil {
		return EntityRecord{}, err
	}

	hash, err := computeJSONHash(params.Payload)
	if err != nil {
		return EntityRecord{}, fmt.Errorf("compute entity hash: %w", err)
	}

	var record EntityRecord
	err = r.db.WithTenant(ctx, space, func(tx pgx.Tx) error {
		if err := r.ensureEntityTable(ctx, tx); err != nil {
			return err
		}

		activeSelect := fmt.Sprintf(`
		SELECT entity_id, entity_version, schema_id, schema_version, payload, hash, created_at, created_by, is_soft_deleted, is_active
		FROM %s
		WHERE entity_id = $1 AND is_active = TRUE AND is_soft_deleted = FALSE
		FOR UPDATE
	`, r.tableIdent)
		currentRow := tx.QueryRow(ctx, activeSelect, entityID)
		currentRecord, err := scanEntityRecord(currentRow)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrEntityNotFound
			}
			return fmt.Errorf("fetch active entity: %w", err)
		}

		nextVersion := currentRecord.EntityVersion.NextPatch()
		deactivateStmt := fmt.Sprintf(`
	UPDATE %s
	SET is_active = FALSE
		WHERE entity_id = $1 AND entity_version = $2
	`, r.tableIdent)
		if _, err := tx.Exec(ctx, deactivateStmt, entityID, currentRecord.EntityVersion.String()); err != nil {
			return fmt.Errorf("deactivate entity version: %w", err)
		}

		insertStmt := fmt.Sprintf(`
        INSERT INTO %s (
			entity_id, entity_version, schema_id, schema_version, payload, hash, is_active, is_soft_deleted, created_at, created_by
        ) VALUES (
			$1, $2, $3, $4, $5, $6, TRUE, FALSE, NOW(), $7
        )
    `, r.tableIdent)
		if _, err := tx.Exec(ctx, insertStmt, entityID, nextVersion.String(), schemaRecord.SchemaID, schemaRecord.VersionString(), []byte(params.Payload), hash, params.CreatedBy); err != nil {
			return fmt.Errorf("insert entity version: %w", err)
		}

		selectStmt := fmt.Sprintf(`
        SELECT entity_id, entity_version, schema_id, schema_version, payload, hash, created_at, created_by, is_soft_deleted, is_active
        FROM %s
        WHERE entity_id = $1 AND entity_version = $2
    `, r.tableIdent)
		row := tx.QueryRow(ctx, selectStmt, entityID, nextVersion.String())
		record, err = scanEntityRecord(row)
		if err != nil {
			return fmt.Errorf("fetch new entity version: %w", err)
		}
		return nil
	})
	if err != nil {
		return EntityRecord{}, err
	}

	return record, nil
}

// CreateOrUpdateEntity attempts to update an existing entity version; if it does not exist it falls back to creation.
func (r *EntityRepository) CreateOrUpdateEntity(ctx context.Context, space tenant.Space, params CreateOrUpdateEntityParams) (EntityRecord, error) {
	if len(params.Payload) == 0 {
		return EntityRecord{}, errors.New("payload is required")
	}

	if strings.TrimSpace(params.EntityID) == "" {
		return r.CreateEntity(ctx, space, CreateEntityParams{
			SchemaVersion: params.SchemaVersion,
			Payload:       params.Payload,
			CreatedBy:     params.CreatedBy,
		})
	}

	updateParams := UpdateEntityParams{
		EntityID:      params.EntityID,
		SchemaVersion: params.SchemaVersion,
		Payload:       params.Payload,
		CreatedBy:     params.CreatedBy,
	}
	record, err := r.UpdateEntity(ctx, space, updateParams)
	if err == nil {
		return record, nil
	}
	if !errors.Is(err, ErrEntityNotFound) {
		return EntityRecord{}, err
	}

	return r.CreateEntity(ctx, space, CreateEntityParams{
		EntityID:      params.EntityID,
		SchemaVersion: params.SchemaVersion,
		Payload:       params.Payload,
		CreatedBy:     params.CreatedBy,
	})
}

// GetEntityByID fetches the latest active entity version.

func (r *EntityRepository) GetEntityByID(ctx context.Context, space tenant.Space, entityID string) (EntityRecord, error) {
	normalized, err := NormalizeEntityIdentifier(entityID)
	if err != nil {
		return EntityRecord{}, err
	}

	var record EntityRecord
	err = r.db.WithTenant(ctx, space, func(tx pgx.Tx) error {
		if err := r.ensureEntityTable(ctx, tx); err != nil {
			return err
		}

		query := fmt.Sprintf(`
		SELECT entity_id, entity_version, schema_id, schema_version, payload, hash, created_at, created_by, is_soft_deleted, is_active
		FROM %s
		WHERE entity_id = $1 AND is_active = TRUE AND is_soft_deleted = FALSE
	`, r.tableIdent)

		row := tx.QueryRow(ctx, query, normalized)
		var scanErr error
		record, scanErr = scanEntityRecord(row)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return ErrEntityNotFound
			}
			return scanErr
		}
		return nil
	})
	if err != nil {
		return EntityRecord{}, err
	}

	return record, nil
}

// GetEntityVersion fetches a specific entity version.
func (r *EntityRepository) GetEntityVersion(ctx context.Context, space tenant.Space, entityID string, version SemanticVersion) (EntityRecord, error) {
	normalized, err := NormalizeEntityIdentifier(entityID)
	if err != nil {
		return EntityRecord{}, err
	}

	var record EntityRecord
	err = r.db.WithTenant(ctx, space, func(tx pgx.Tx) error {
		if err := r.ensureEntityTable(ctx, tx); err != nil {
			return err
		}

		query := fmt.Sprintf(`
		SELECT entity_id, entity_version, schema_id, schema_version, payload, hash, created_at, created_by, is_soft_deleted, is_active
		FROM %s
		WHERE entity_id = $1 AND entity_version = $2
	`, r.tableIdent)

		row := tx.QueryRow(ctx, query, normalized, version.String())
		var scanErr error
		record, scanErr = scanEntityRecord(row)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return ErrEntityNotFound
			}
			return scanErr
		}
		return nil
	})
	if err != nil {
		return EntityRecord{}, err
	}

	return record, nil
}

// ListEntities returns entities ordered by creation time.
func (r *EntityRepository) ListEntities(ctx context.Context, space tenant.Space, params ListEntitiesParams) ([]EntityRecord, error) {
	limit := params.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	sortField, sortOrder, err := sanitizeEntitySort(params.SortField, params.SortOrder)
	if err != nil {
		return nil, err
	}

	var records []EntityRecord
	err = r.db.WithTenant(ctx, space, func(tx pgx.Tx) error {
		if err := r.ensureEntityTable(ctx, tx); err != nil {
			return err
		}

		query := fmt.Sprintf(`
		SELECT entity_id, entity_version, schema_id, schema_version, payload, hash, created_at, created_by, is_soft_deleted, is_active
		FROM %s
		WHERE ($1::bool = FALSE OR is_active = TRUE)
		  AND ($2::bool = TRUE OR is_soft_deleted = FALSE)
		ORDER BY %s %s
		LIMIT $3 OFFSET $4
	`, r.tableIdent, sortField, sortOrder)

		rows, err := tx.Query(ctx, query, params.OnlyActive, params.IncludeDeleted, limit, offset)
		if err != nil {
			return fmt.Errorf("list entities: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			record, err := scanEntityRecord(rows)
			if err != nil {
				return err
			}
			records = append(records, record)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return records, nil
}

// CountEntities returns the total number of entities matching the provided filters.
func (r *EntityRepository) CountEntities(ctx context.Context, space tenant.Space, params ListEntitiesParams) (int64, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s
		WHERE ($1::bool = FALSE OR is_active = TRUE)
		  AND ($2::bool = TRUE OR is_soft_deleted = FALSE)
	`, r.tableIdent)

	var total int64
	err := r.db.WithTenant(ctx, space, func(tx pgx.Tx) error {
		if err := r.ensureEntityTable(ctx, tx); err != nil {
			return err
		}

		if err := tx.QueryRow(ctx, query, params.OnlyActive, params.IncludeDeleted).Scan(&total); err != nil {
			return fmt.Errorf("count entities: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return total, nil
}

func sanitizeEntitySort(field, order string) (string, string, error) {
	column := "created_at"
	if field != "" {
		switch field {
		case "created_at":
			column = field
		default:
			return "", "", fmt.Errorf("unsupported sort field %q", field)
		}
	}

	sortOrder := "DESC"
	if strings.EqualFold(order, "asc") {
		sortOrder = "ASC"
	} else if strings.EqualFold(order, "desc") || order == "" {
		sortOrder = "DESC"
	} else {
		return "", "", fmt.Errorf("unsupported sort order %q", order)
	}

	return column, sortOrder, nil
}

// SoftDeleteEntity marks all versions of the entity as deleted and non-active.
// deletedAt is ignored because entity versions are immutable and only track creation time.
func (r *EntityRepository) SoftDeleteEntity(ctx context.Context, space tenant.Space, entityID string, _ time.Time) error {
	normalized, err := NormalizeEntityIdentifier(entityID)
	if err != nil {
		return err
	}

	stmt := fmt.Sprintf(`
		UPDATE %s
		SET is_soft_deleted = TRUE,
		    is_active = FALSE
		WHERE entity_id = $1 AND is_soft_deleted = FALSE
	`, r.tableIdent)

	err = r.db.WithTenant(ctx, space, func(tx pgx.Tx) error {
		if err := r.ensureEntityTable(ctx, tx); err != nil {
			return err
		}

		tag, execErr := tx.Exec(ctx, stmt, normalized)
		if execErr != nil {
			return fmt.Errorf("soft delete entity: %w", execErr)
		}

		if tag.RowsAffected() == 0 {
			return ErrEntityNotFound
		}
		return nil
	})

	return err
}

func (r *EntityRepository) resolveSchema(ctx context.Context, version *SemanticVersion) (SchemaRecord, error) {
	if version == nil {
		schema, err := r.schemas.GetActiveSchema(ctx, r.schemaID)
		if err != nil {
			return SchemaRecord{}, err
		}
		if schema.TableName != r.tableName {
			return SchemaRecord{}, fmt.Errorf("schema %s table name mismatch", r.schemaID)
		}
		return schema, nil
	}
	schema, err := r.schemas.GetSchemaByVersion(ctx, r.schemaID, *version)
	if err != nil {
		return SchemaRecord{}, err
	}
	if schema.TableName != r.tableName {
		return SchemaRecord{}, fmt.Errorf("schema %s table name mismatch", r.schemaID)
	}
	return schema, nil
}

func (r *EntityRepository) ensureEntityTable(ctx context.Context, tx pgx.Tx) error {
	tableDDL := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	entity_id TEXT NOT NULL CHECK (char_length(entity_id) >= 1 AND char_length(entity_id) <= 128),
	entity_version TEXT NOT NULL CHECK (entity_version ~ '^\d+\.\d+\.\d+$'),
	schema_id UUID NOT NULL,
	schema_version TEXT NOT NULL CHECK (schema_version ~ '^\d+\.\d+\.\d+$'),
	payload JSONB NOT NULL,
	hash TEXT NOT NULL CHECK (hash ~ '^[a-f0-9]{64}$'),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	created_by TEXT NULL,
	is_active BOOLEAN NOT NULL DEFAULT TRUE,
	is_soft_deleted BOOLEAN NOT NULL DEFAULT FALSE,
	PRIMARY KEY (entity_id, entity_version),
	FOREIGN KEY (schema_id, schema_version) REFERENCES schema_repository(schema_id, schema_version)
);`, r.tableIdent)

	activeIndex := fmt.Sprintf(`
CREATE UNIQUE INDEX IF NOT EXISTS %s_active_idx ON %s (entity_id)
WHERE is_active AND NOT is_soft_deleted;
`, r.tableName, r.tableIdent)
	schemaIndex := fmt.Sprintf(`
CREATE INDEX IF NOT EXISTS %s_schema_idx ON %s (schema_id, schema_version);
`, r.tableName, r.tableIdent)

	statements := []string{tableDDL, activeIndex, schemaIndex}
	for _, stmt := range statements {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("ensure entity table %s: %w", r.tableName, err)
		}
	}

	return nil
}

func scanEntityRecord(scanner rowScanner) (EntityRecord, error) {
	var (
		entityID      string
		entityVersion string
		schemaID      uuid.UUID
		schemaVersion string
		payload       []byte
		hash          string
		createdAt     time.Time
		createdBy     *string
		isSoftDeleted bool
		isActive      bool
	)

	if err := scanner.Scan(&entityID, &entityVersion, &schemaID, &schemaVersion, &payload, &hash, &createdAt, &createdBy, &isSoftDeleted, &isActive); err != nil {
		return EntityRecord{}, err
	}

	ev, err := ParseSemanticVersion(entityVersion)
	if err != nil {
		return EntityRecord{}, fmt.Errorf("parse entity version %q: %w", entityVersion, err)
	}

	sv, err := ParseSemanticVersion(schemaVersion)
	if err != nil {
		return EntityRecord{}, fmt.Errorf("parse schema version %q: %w", schemaVersion, err)
	}

	return EntityRecord{
		EntityID:      entityID,
		EntityVersion: ev,
		SchemaID:      schemaID,
		SchemaVersion: sv,
		Payload:       json.RawMessage(payload),
		Hash:          hash,
		CreatedAt:     createdAt,
		CreatedBy:     createdBy,
		IsSoftDeleted: isSoftDeleted,
		IsActive:      isActive,
	}, nil
}
