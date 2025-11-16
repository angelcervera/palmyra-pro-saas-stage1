package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	domainrepo "github.com/zenGate-Global/palmyra-pro-saas/domains/schema-repository/be/repo"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
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
	ErrNotFound = errors.New("schema version not found")
	ErrConflict = errors.New("schema version conflict")
)

// Schema represents a schema repository record managed by the domain service.
type Schema struct {
	SchemaID      uuid.UUID
	Version       persistence.SemanticVersion
	Definition    json.RawMessage
	TableName     string
	Slug          string
	CategoryID    uuid.UUID
	CreatedAt     time.Time
	IsActive      bool
	IsSoftDeleted bool
}

// CreateInput defines the payload required to register a schema version.
type CreateInput struct {
	SchemaID   *uuid.UUID
	Version    *persistence.SemanticVersion
	Definition json.RawMessage
	TableName  string
	Slug       string
	CategoryID uuid.UUID
}

// Service exposes schema repository operations.
type Service interface {
	Create(ctx context.Context, input CreateInput) (Schema, error)
	ListAll(ctx context.Context, includeInactive bool) ([]Schema, error)
	List(ctx context.Context, schemaID uuid.UUID, includeDeleted bool) ([]Schema, error)
	Get(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion) (Schema, error)
	GetActive(ctx context.Context, schemaID uuid.UUID) (Schema, error)
	Activate(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion) (Schema, error)
	Delete(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion) error
}

type service struct {
	repo domainrepo.Repository
	now  func() time.Time
}

var tableNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// New builds a schema repository Service backed by the provided repository.
func New(repo domainrepo.Repository) Service {
	if repo == nil {
		panic("schema repository repo is required")
	}

	return &service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *service) Create(ctx context.Context, input CreateInput) (Schema, error) {
	normalized, validationErr := s.validateCreateInput(input)
	if validationErr != nil {
		return Schema{}, validationErr
	}

	schemaID, existingRecords, err := s.resolveSchemaID(ctx, input, normalized)
	if err != nil {
		return Schema{}, err
	}

	version, err := s.resolveVersion(existingRecords, input.Version)
	if err != nil {
		return Schema{}, err
	}

	if _, err := s.repo.GetByVersion(ctx, schemaID, version); err == nil {
		return Schema{}, ErrConflict
	} else if err != nil && !errors.Is(err, persistence.ErrSchemaNotFound) {
		return Schema{}, err
	}

	if err := s.ensureSchemaConsistency(existingRecords, normalized); err != nil {
		return Schema{}, err
	}

	params := persistence.CreateSchemaParams{
		SchemaID:   schemaID,
		Version:    version,
		Definition: cloneRawMessage(input.Definition),
		TableName:  normalized.tableName,
		Slug:       normalized.slug,
		CategoryID: input.CategoryID,
		Activate:   true,
	}

	record, err := s.repo.Upsert(ctx, params)
	if err != nil {
		return Schema{}, s.translateUpsertError(err)
	}

	return mapRecord(record), nil
}

func (s *service) List(ctx context.Context, schemaID uuid.UUID, includeDeleted bool) ([]Schema, error) {
	if schemaID == uuid.Nil {
		return nil, ErrNotFound
	}

	records, err := s.repo.List(ctx, schemaID)
	if err != nil {
		if errors.Is(err, persistence.ErrSchemaNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	results := make([]Schema, 0, len(records))
	for _, record := range records {
		if !includeDeleted && record.IsSoftDeleted {
			continue
		}
		results = append(results, mapRecord(record))
	}

	return results, nil
}

func (s *service) ListAll(ctx context.Context, includeInactive bool) ([]Schema, error) {
	records, err := s.repo.ListAll(ctx, includeInactive)
	if err != nil {
		return nil, err
	}

	results := make([]Schema, 0, len(records))
	for _, record := range records {
		if !includeInactive && !record.IsActive {
			continue
		}
		results = append(results, mapRecord(record))
	}

	return results, nil
}

func (s *service) Get(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion) (Schema, error) {
	if schemaID == uuid.Nil {
		return Schema{}, ErrNotFound
	}

	record, err := s.repo.GetByVersion(ctx, schemaID, version)
	if err != nil {
		if errors.Is(err, persistence.ErrSchemaNotFound) {
			return Schema{}, ErrNotFound
		}
		return Schema{}, err
	}

	return mapRecord(record), nil
}

func (s *service) GetActive(ctx context.Context, schemaID uuid.UUID) (Schema, error) {
	if schemaID == uuid.Nil {
		return Schema{}, ErrNotFound
	}

	record, err := s.repo.GetActive(ctx, schemaID)
	if err != nil {
		if errors.Is(err, persistence.ErrSchemaNotFound) {
			return Schema{}, ErrNotFound
		}
		return Schema{}, err
	}

	return mapRecord(record), nil
}

func (s *service) Activate(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion) (Schema, error) {
	if schemaID == uuid.Nil {
		return Schema{}, ErrNotFound
	}

	if err := s.repo.Activate(ctx, schemaID, version); err != nil {
		if errors.Is(err, persistence.ErrSchemaNotFound) {
			return Schema{}, ErrNotFound
		}
		return Schema{}, err
	}

	record, err := s.repo.GetByVersion(ctx, schemaID, version)
	if err != nil {
		if errors.Is(err, persistence.ErrSchemaNotFound) {
			return Schema{}, ErrNotFound
		}
		return Schema{}, err
	}

	return mapRecord(record), nil
}

func (s *service) Delete(ctx context.Context, schemaID uuid.UUID, version persistence.SemanticVersion) error {
	if schemaID == uuid.Nil {
		return ErrNotFound
	}

	if err := s.repo.SoftDelete(ctx, schemaID, version, s.now()); err != nil {
		if errors.Is(err, persistence.ErrSchemaNotFound) {
			return ErrNotFound
		}
		return err
	}

	return nil
}

type normalizedCreateInput struct {
	slug      string
	tableName string
}

func (s *service) validateCreateInput(input CreateInput) (normalizedCreateInput, error) {
	fieldErrors := FieldErrors{}

	if input.SchemaID != nil && *input.SchemaID == uuid.Nil {
		addFieldError(fieldErrors, "schemaId", "schemaId is required")
	}

	if input.CategoryID == uuid.Nil {
		addFieldError(fieldErrors, "categoryId", "categoryId is required")
	}

	var normalized normalizedCreateInput

	if slug, err := persistence.NormalizeSlug(input.Slug); err != nil {
		addFieldError(fieldErrors, "slug", err.Error())
	} else {
		normalized.slug = slug
	}

	tableName := strings.TrimSpace(input.TableName)
	switch {
	case tableName == "":
		addFieldError(fieldErrors, "tableName", "tableName is required")
	case !tableNamePattern.MatchString(tableName):
		addFieldError(fieldErrors, "tableName", fmt.Sprintf("tableName must match %s", tableNamePattern.String()))
	default:
		normalized.tableName = tableName
	}

	if len(input.Definition) == 0 {
		addFieldError(fieldErrors, "schemaDefinition", "schemaDefinition is required")
	} else if !isJSONObject(input.Definition) {
		addFieldError(fieldErrors, "schemaDefinition", "schemaDefinition must be a JSON object")
	}

	if len(fieldErrors) > 0 {
		return normalizedCreateInput{}, &ValidationError{Fields: fieldErrors}
	}

	return normalized, nil
}

func (s *service) resolveSchemaID(ctx context.Context, input CreateInput, normalized normalizedCreateInput) (uuid.UUID, []persistence.SchemaRecord, error) {
	if input.SchemaID != nil {
		records, err := s.repo.List(ctx, *input.SchemaID)
		if err != nil {
			if errors.Is(err, persistence.ErrSchemaNotFound) {
				return uuid.Nil, nil, ErrNotFound
			}
			return uuid.Nil, nil, err
		}
		if len(records) == 0 {
			return uuid.Nil, nil, ErrNotFound
		}
		return *input.SchemaID, records, nil
	}

	record, err := s.repo.GetLatestBySlug(ctx, normalized.slug)
	switch {
	case err == nil:
		records, listErr := s.repo.List(ctx, record.SchemaID)
		if listErr != nil {
			if errors.Is(listErr, persistence.ErrSchemaNotFound) {
				records = []persistence.SchemaRecord{record}
				return record.SchemaID, records, nil
			}
			return uuid.Nil, nil, listErr
		}
		if len(records) == 0 {
			records = append(records, record)
		}
		return record.SchemaID, records, nil
	case errors.Is(err, persistence.ErrSchemaNotFound):
		return uuid.New(), nil, nil
	default:
		return uuid.Nil, nil, err
	}
}

func (s *service) resolveVersion(existing []persistence.SchemaRecord, requested *persistence.SemanticVersion) (persistence.SemanticVersion, error) {
	if requested != nil {
		return *requested, nil
	}

	if len(existing) == 0 {
		return persistence.SemanticVersion{Major: 1, Minor: 0, Patch: 0}, nil
	}

	maxVersion := existing[0].SchemaVersion
	for _, record := range existing[1:] {
		if record.SchemaVersion.Compare(maxVersion) > 0 {
			maxVersion = record.SchemaVersion
		}
	}

	return maxVersion.NextPatch(), nil
}

func (s *service) ensureSchemaConsistency(existing []persistence.SchemaRecord, normalized normalizedCreateInput) error {
	if len(existing) == 0 {
		return nil
	}

	reference := existing[0]
	if reference.TableName != "" && reference.TableName != normalized.tableName {
		return &ValidationError{
			Fields: FieldErrors{
				"tableName": {"table name must match the existing schema table"},
			},
		}
	}

	if reference.Slug != "" && reference.Slug != normalized.slug {
		return &ValidationError{
			Fields: FieldErrors{
				"slug": {"slug must match the existing schema slug"},
			},
		}
	}

	return nil
}

func (s *service) translateUpsertError(err error) error {
	if errors.Is(err, persistence.ErrSchemaNotFound) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrConflict
		case "23503":
			return &ValidationError{
				Fields: FieldErrors{
					"categoryId": {"categoryId does not reference an existing schema category"},
				},
			}
		}
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "table name"):
		return &ValidationError{Fields: FieldErrors{"tableName": {message}}}
	case strings.Contains(message, "slug"):
		return &ValidationError{Fields: FieldErrors{"slug": {message}}}
	case strings.Contains(message, "schema definition"):
		return &ValidationError{Fields: FieldErrors{"schemaDefinition": {message}}}
	case strings.Contains(message, "schema id"):
		return &ValidationError{Fields: FieldErrors{"schemaId": {message}}}
	}

	return err
}

func addFieldError(m FieldErrors, field, message string) {
	m[field] = append(m[field], message)
}

func isJSONObject(raw json.RawMessage) bool {
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}
	_, ok := payload.(map[string]any)
	return ok
}

func mapRecord(record persistence.SchemaRecord) Schema {
	return Schema{
		SchemaID:      record.SchemaID,
		Version:       record.SchemaVersion,
		Definition:    cloneRawMessage(record.SchemaDefinition),
		TableName:     record.TableName,
		Slug:          record.Slug,
		CategoryID:    record.CategoryID,
		CreatedAt:     record.CreatedAt,
		IsActive:      record.IsActive,
		IsSoftDeleted: record.IsSoftDeleted,
	}
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	buf := make([]byte, len(raw))
	copy(buf, raw)
	return buf
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}
