package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/santhosh-tekuri/jsonschema/v5"

	domainrepo "github.com/zenGate-Global/palmyra-pro-saas/domains/entities/be/repo"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
)

// ValidationError captures payload validation issues surfaced by the JSON schema validator.
type ValidationError struct {
	Reason string
}

func (e *ValidationError) Error() string {
	return "validation error"
}

// Domain-level errors surfaced by the service.
var (
	ErrTableNotFound    = errors.New("table not found")
	ErrDocumentNotFound = errors.New("document not found")
	ErrConflict         = errors.New("entity conflict")
)

// Document represents an entity record enriched for API rendering.
type Document struct {
	EntityID      uuid.UUID
	EntityVersion persistence.SemanticVersion
	SchemaID      uuid.UUID
	SchemaVersion persistence.SemanticVersion
	Payload       map[string]interface{}
	CreatedAt     time.Time
	IsActive      bool
	IsSoftDeleted bool
}

// ListResult contains paginated documents and metadata.
type ListResult struct {
	Items      []Document
	Page       int
	PageSize   int
	TotalItems int64
	TotalPages int
}

// ListOptions defines pagination inputs.
type ListOptions struct {
	Page     int
	PageSize int
	Sort     string
}

// Service exposes entity operations backed by the persistence layer.
type Service interface {
	List(ctx context.Context, tableName string, opts ListOptions) (ListResult, error)
	Create(ctx context.Context, tableName string, payload map[string]interface{}) (Document, error)
	Get(ctx context.Context, tableName string, entityID uuid.UUID) (Document, error)
	Update(ctx context.Context, tableName string, entityID uuid.UUID, payload map[string]interface{}) (Document, error)
	Delete(ctx context.Context, tableName string, entityID uuid.UUID) error
}

type service struct {
	repo domainrepo.Repository
}

// New constructs a Service instance.
func New(repo domainrepo.Repository) Service {
	if repo == nil {
		panic("entities repository is required")
	}

	return &service{repo: repo}
}

func (s *service) List(ctx context.Context, tableName string, opts ListOptions) (ListResult, error) {
	if strings.TrimSpace(tableName) == "" {
		return ListResult{}, &ValidationError{Reason: "tableName is required"}
	}

	page := opts.Page
	if page < 1 {
		page = 1
	}
	pageSize := opts.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	sortColumn, sortOrder := normalizeSort(opts.Sort)

	result, err := s.repo.List(ctx, tableName, domainrepo.ListParams{
		Page:       page,
		PageSize:   pageSize,
		SortColumn: sortColumn,
		SortOrder:  sortOrder,
	})
	if err != nil {
		return ListResult{}, translateError(err)
	}

	items := make([]Document, 0, len(result.Records))
	for _, record := range result.Records {
		doc, mapErr := mapRecord(record)
		if mapErr != nil {
			return ListResult{}, mapErr
		}
		items = append(items, doc)
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = int(math.Ceil(float64(result.Total) / float64(pageSize)))
	}

	return ListResult{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: result.Total,
		TotalPages: totalPages,
	}, nil
}

func (s *service) Create(ctx context.Context, tableName string, payload map[string]interface{}) (Document, error) {
	if strings.TrimSpace(tableName) == "" {
		return Document{}, &ValidationError{Reason: "tableName is required"}
	}
	if payload == nil {
		return Document{}, &ValidationError{Reason: "payload is required"}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Document{}, fmt.Errorf("encode payload: %w", err)
	}

	record, err := s.repo.Create(ctx, tableName, body)
	if err != nil {
		return Document{}, translateError(err)
	}

	return mapRecord(record)
}

func (s *service) Get(ctx context.Context, tableName string, entityID uuid.UUID) (Document, error) {
	if strings.TrimSpace(tableName) == "" {
		return Document{}, &ValidationError{Reason: "tableName is required"}
	}
	if entityID == uuid.Nil {
		return Document{}, &ValidationError{Reason: "entityId is required"}
	}

	record, err := s.repo.Get(ctx, tableName, entityID)
	if err != nil {
		return Document{}, translateError(err)
	}

	return mapRecord(record)
}

func (s *service) Update(ctx context.Context, tableName string, entityID uuid.UUID, payload map[string]interface{}) (Document, error) {
	if strings.TrimSpace(tableName) == "" {
		return Document{}, &ValidationError{Reason: "tableName is required"}
	}
	if entityID == uuid.Nil {
		return Document{}, &ValidationError{Reason: "entityId is required"}
	}
	if payload == nil {
		return Document{}, &ValidationError{Reason: "payload is required"}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Document{}, fmt.Errorf("encode payload: %w", err)
	}

	record, err := s.repo.Update(ctx, tableName, entityID, body)
	if err != nil {
		return Document{}, translateError(err)
	}

	return mapRecord(record)
}

func (s *service) Delete(ctx context.Context, tableName string, entityID uuid.UUID) error {
	if strings.TrimSpace(tableName) == "" {
		return &ValidationError{Reason: "tableName is required"}
	}
	if entityID == uuid.Nil {
		return &ValidationError{Reason: "entityId is required"}
	}

	if err := s.repo.Delete(ctx, tableName, entityID); err != nil {
		return translateError(err)
	}

	return nil
}

func mapRecord(record persistence.EntityRecord) (Document, error) {
	var payload map[string]interface{}
	if len(record.Payload) > 0 {
		if err := json.Unmarshal(record.Payload, &payload); err != nil {
			return Document{}, fmt.Errorf("decode entity payload: %w", err)
		}
	} else {
		payload = map[string]interface{}{}
	}

	return Document{
		EntityID:      record.EntityID,
		EntityVersion: record.EntityVersion,
		SchemaID:      record.SchemaID,
		SchemaVersion: record.SchemaVersion,
		Payload:       payload,
		CreatedAt:     record.CreatedAt,
		IsActive:      record.IsActive,
		IsSoftDeleted: record.IsSoftDeleted,
	}, nil
}

func normalizeSort(sort string) (string, string) {
	if sort == "" {
		return "created_at", "desc"
	}

	parts := strings.Split(sort, ",")
	first := strings.TrimSpace(parts[0])
	order := "asc"
	field := first
	if strings.HasPrefix(first, "-") {
		order = "desc"
		field = strings.TrimPrefix(first, "-")
	}

	switch field {
	case "slug":
		return "slug", order
	case "createdAt":
		return "created_at", order
	default:
		return "created_at", "desc"
	}
}

func translateError(err error) error {
	switch {
	case errors.Is(err, persistence.ErrSchemaNotFound):
		return ErrTableNotFound
	case errors.Is(err, persistence.ErrEntityNotFound):
		return ErrDocumentNotFound
	case errors.Is(err, persistence.ErrEntityAlreadyExists):
		return ErrConflict
	default:
		var validationErr *jsonschema.ValidationError
		if errors.As(err, &validationErr) {
			return &ValidationError{Reason: validationErr.Error()}
		}
		return err
	}
}
