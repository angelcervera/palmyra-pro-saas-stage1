package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/schema-repository/be/service"
	externalRef2 "github.com/zenGate-Global/palmyra-pro-saas/generated/go/common/primitives"
	externalRef3 "github.com/zenGate-Global/palmyra-pro-saas/generated/go/common/problemdetails"
	schemarepository "github.com/zenGate-Global/palmyra-pro-saas/generated/go/schema-repository"
	platformlogging "github.com/zenGate-Global/palmyra-pro-saas/platform/go/logging"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
)

const (
	problemTypeValidation              = "https://palmyra.pro/problems/validation-error"
	problemTypeNotFound                = "https://palmyra.pro/problems/not-found"
	problemTypeConflict                = "https://palmyra.pro/problems/conflict"
	problemTypeInternal                = "https://palmyra.pro/problems/internal-error"
	schemaRepositoryBasePath           = "/api/v1/schema-repository/schemas"
	listOperation            operation = "listSchemaVersions"
	createOperation          operation = "createSchemaVersion"
	getOperation             operation = "getSchemaVersion"
)

type operation string

// Handler wires the schema repository service to the generated HTTP contract.
type Handler struct {
	svc    service.Service
	logger *zap.Logger
}

// New constructs a Handler instance.
func New(svc service.Service, logger *zap.Logger) *Handler {
	if svc == nil {
		panic("schema repository service is required")
	}
	if logger == nil {
		panic("logger is required")
	}

	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) CreateSchemaVersion(ctx context.Context, request schemarepository.CreateSchemaVersionRequestObject) (schemarepository.CreateSchemaVersionResponseObject, error) {
	if request.Body == nil {
		problem := h.buildProblem("Invalid request body", "request body is required", problemTypeValidation, http.StatusBadRequest, nil)
		return schemarepository.CreateSchemaVersiondefaultApplicationProblemPlusJSONResponse{
			Body:       problem,
			StatusCode: http.StatusBadRequest,
		}, nil
	}

	input, err := h.createInputFromRequest(ctx, request.Body)
	if err != nil {
		status, problem := h.problemForError(ctx, err, createOperation)
		return schemarepository.CreateSchemaVersiondefaultApplicationProblemPlusJSONResponse{
			Body:       problem,
			StatusCode: status,
		}, nil
	}

	schemaVersion, err := h.svc.Create(ctx, input)
	if err != nil {
		status, problem := h.problemForError(ctx, err, createOperation)
		return schemarepository.CreateSchemaVersiondefaultApplicationProblemPlusJSONResponse{
			Body:       problem,
			StatusCode: status,
		}, nil
	}

	location := fmt.Sprintf("%s/%s/versions/%s", schemaRepositoryBasePath, schemaVersion.SchemaID.String(), schemaVersion.Version.String())

	return schemarepository.CreateSchemaVersion201JSONResponse{
		Body:    toAPISchema(schemaVersion),
		Headers: schemarepository.CreateSchemaVersion201ResponseHeaders{Location: location},
	}, nil
}

func (h *Handler) ListAllSchemaVersions(ctx context.Context, request schemarepository.ListAllSchemaVersionsRequestObject) (schemarepository.ListAllSchemaVersionsResponseObject, error) {
	includeInactive := false
	if request.Params.IncludeInactive != nil {
		includeInactive = *request.Params.IncludeInactive
	}

	versions, err := h.svc.ListAll(ctx, includeInactive)
	if err != nil {
		status, problem := h.problemForError(ctx, err, listOperation)
		return schemarepository.ListAllSchemaVersionsdefaultApplicationProblemPlusJSONResponse{
			Body:       problem,
			StatusCode: status,
		}, nil
	}

	items := make([]schemarepository.SchemaVersion, 0, len(versions))
	for _, version := range versions {
		apiVersion, convertErr := toAPISchemaSafe(version)
		if convertErr != nil {
			status, problem := h.problemForError(ctx, convertErr, listOperation)
			return schemarepository.ListAllSchemaVersionsdefaultApplicationProblemPlusJSONResponse{
				Body:       problem,
				StatusCode: status,
			}, nil
		}
		items = append(items, apiVersion)
	}

	return schemarepository.ListAllSchemaVersions200JSONResponse{
		Items: items,
	}, nil
}

func (h *Handler) GetSchemaVersion(ctx context.Context, request schemarepository.GetSchemaVersionRequestObject) (schemarepository.GetSchemaVersionResponseObject, error) {
	schemaID := uuidFromExternal(request.SchemaId)
	version, err := persistence.ParseSemanticVersion(string(request.SchemaVersion))
	if err != nil {
		validationErr := &service.ValidationError{
			Fields: service.FieldErrors{
				"schemaVersion": {fmt.Sprintf("invalid semantic version: %v", err)},
			},
		}
		status, problem := h.problemForError(ctx, validationErr, getOperation)
		return schemarepository.GetSchemaVersiondefaultApplicationProblemPlusJSONResponse{
			Body:       problem,
			StatusCode: status,
		}, nil
	}

	schemaVersion, err := h.svc.Get(ctx, schemaID, version)
	if err != nil {
		status, problem := h.problemForError(ctx, err, getOperation)
		return schemarepository.GetSchemaVersiondefaultApplicationProblemPlusJSONResponse{
			Body:       problem,
			StatusCode: status,
		}, nil
	}

	apiSchema, convertErr := toAPISchemaSafe(schemaVersion)
	if convertErr != nil {
		status, problem := h.problemForError(ctx, convertErr, getOperation)
		return schemarepository.GetSchemaVersiondefaultApplicationProblemPlusJSONResponse{
			Body:       problem,
			StatusCode: status,
		}, nil
	}

	return schemarepository.GetSchemaVersion200JSONResponse(apiSchema), nil
}

func (h *Handler) createInputFromRequest(ctx context.Context, body *schemarepository.CreateSchemaVersionRequest) (service.CreateInput, error) {
	definitionBytes, err := json.Marshal(body.SchemaDefinition)
	if err != nil {
		return service.CreateInput{}, fmt.Errorf("encode schemaDefinition: %w", err)
	}

	input := service.CreateInput{
		Definition: definitionBytes,
		TableName:  string(body.TableName),
		Slug:       string(body.Slug),
		CategoryID: uuidFromExternal(body.CategoryId),
	}

	return input, nil
}

func toAPISchema(schema service.Schema) schemarepository.SchemaVersion {
	apiSchema, err := toAPISchemaSafe(schema)
	if err != nil {
		// toAPISchema should only be used when conversion is guaranteed to succeed.
		panic(err)
	}
	return apiSchema
}

func toAPISchemaSafe(schema service.Schema) (schemarepository.SchemaVersion, error) {
	definitionMap, err := rawMessageToMap(schema.Definition)
	if err != nil {
		return schemarepository.SchemaVersion{}, err
	}

	apiSchema := schemarepository.SchemaVersion{
		SchemaId:         externalRef2.UUID(schema.SchemaID),
		SchemaVersion:    externalRef2.SemanticVersion(schema.Version.String()),
		SchemaDefinition: definitionMap,
		TableName:        externalRef2.TableName(schema.TableName),
		Slug:             externalRef2.Slug(schema.Slug),
		CategoryId:       externalRef2.UUID(schema.CategoryID),
		CreatedAt:        externalRef2.Timestamp(schema.CreatedAt),
		IsActive:         schema.IsActive,
		IsSoftDeleted:    schema.IsSoftDeleted,
	}

	return apiSchema, nil
}

func rawMessageToMap(raw json.RawMessage) (map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode schema definition: %w", err)
	}
	return payload, nil
}

func uuidFromExternal(id externalRef2.UUID) uuid.UUID {
	return uuid.UUID(id)
}

func (h *Handler) problemForError(ctx context.Context, err error, op operation) (int, externalRef3.ProblemDetails) {
	status, title, detail, problemType, fieldErrors := h.classifyError(err)

	logger := h.loggerFrom(ctx)
	fields := []zap.Field{
		zap.String("operation", string(op)),
		zap.Int("status", status),
	}

	switch {
	case status >= http.StatusInternalServerError:
		logger.Error("schema repository operation failed", append(fields, zap.Error(err))...)
	case status == http.StatusNotFound:
		logger.Info("schema repository resource not found", append(fields, zap.Error(err))...)
	default:
		logger.Warn("schema repository request rejected", append(fields, zap.Error(err))...)
	}

	return status, h.buildProblem(title, detail, problemType, status, fieldErrors)
}

func (h *Handler) classifyError(err error) (status int, title, detail, problemType string, fieldErrors service.FieldErrors) {
	var validationErr *service.ValidationError
	switch {
	case errors.As(err, &validationErr):
		return http.StatusBadRequest,
			"Validation failed",
			"one or more fields are invalid",
			problemTypeValidation,
			validationErr.Fields
	case errors.Is(err, service.ErrNotFound):
		return http.StatusNotFound,
			"Resource not found",
			"schema version not found",
			problemTypeNotFound,
			nil
	case errors.Is(err, service.ErrConflict):
		return http.StatusConflict,
			"Conflict",
			"schema version already exists",
			problemTypeConflict,
			nil
	default:
		return http.StatusInternalServerError,
			"Internal server error",
			"an unexpected error occurred",
			problemTypeInternal,
			nil
	}
}

func (h *Handler) buildProblem(title, detail, problemType string, status int, fieldErrors service.FieldErrors) externalRef3.ProblemDetails {
	problem := externalRef3.ProblemDetails{
		Title:  title,
		Status: status,
	}

	if detail != "" {
		problem.Detail = &detail
	}
	if problemType != "" {
		problem.Type = &problemType
	}

	if len(fieldErrors) > 0 {
		copied := make(map[string][]string, len(fieldErrors))
		for field, messages := range fieldErrors {
			copied[field] = append([]string(nil), messages...)
		}
		problem.Errors = &copied
	}

	return problem
}

func (h *Handler) loggerFrom(ctx context.Context) *zap.Logger {
	if logger, ok := platformlogging.FromContext(ctx); ok {
		return logger
	}
	return h.logger
}
