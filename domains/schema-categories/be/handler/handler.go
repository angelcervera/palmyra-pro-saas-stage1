package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/schema-categories/be/service"
	externalRef2 "github.com/zenGate-Global/palmyra-pro-saas/generated/go/common/primitives"
	externalRef3 "github.com/zenGate-Global/palmyra-pro-saas/generated/go/common/problemdetails"
	schemacategories "github.com/zenGate-Global/palmyra-pro-saas/generated/go/schema-categories"
	platformlogging "github.com/zenGate-Global/palmyra-pro-saas/platform/go/logging"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/requesttrace"
)

const (
	problemTypeValidation    = "https://palmyra.pro/problems/validation-error"
	problemTypeNotFound      = "https://palmyra.pro/problems/not-found"
	problemTypeConflict      = "https://palmyra.pro/problems/conflict"
	problemTypeInternal      = "https://palmyra.pro/problems/internal-error"
	schemaCategoriesBasePath = "/api/v1/schema-categories"
)

type operation string

const (
	listOperation   operation = "listSchemaCategories"
	createOperation operation = "createSchemaCategory"
	getOperation    operation = "getSchemaCategory"
	updateOperation operation = "updateSchemaCategory"
	deleteOperation operation = "deleteSchemaCategory"
)

// Handler wires the schema categories service to the generated HTTP contract.
type Handler struct {
	svc    service.Service
	logger *zap.Logger
}

func (h *Handler) audit(ctx context.Context) requesttrace.AuditInfo {
	return requesttrace.FromContextOrAnonymous(ctx)
}

// New constructs a Handler instance.
func New(svc service.Service, logger *zap.Logger) *Handler {
	if svc == nil {
		panic("schema categories service is required")
	}
	if logger == nil {
		panic("logger is required")
	}

	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) ListSchemaCategories(ctx context.Context, request schemacategories.ListSchemaCategoriesRequestObject) (schemacategories.ListSchemaCategoriesResponseObject, error) {
	audit := h.audit(ctx)
	includeDeleted := false
	if request.Params.IncludeDeleted != nil {
		includeDeleted = *request.Params.IncludeDeleted
	}

	categories, err := h.svc.List(ctx, audit, includeDeleted)
	if err != nil {
		status, problem := h.problemForError(ctx, err, listOperation)
		return schemacategories.ListSchemaCategoriesdefaultApplicationProblemPlusJSONResponse{
			Body:       problem,
			StatusCode: status,
		}, nil
	}

	items := make([]schemacategories.SchemaCategory, 0, len(categories))
	for _, category := range categories {
		items = append(items, toAPICategory(category))
	}

	return schemacategories.ListSchemaCategories200JSONResponse(schemacategories.SchemaCategoryList{Items: items}), nil
}

func (h *Handler) CreateSchemaCategory(ctx context.Context, request schemacategories.CreateSchemaCategoryRequestObject) (schemacategories.CreateSchemaCategoryResponseObject, error) {
	audit := h.audit(ctx)
	if request.Body == nil {
		problem := h.buildProblem("Invalid request body", "request body is required", problemTypeValidation, http.StatusBadRequest, nil)
		return schemacategories.CreateSchemaCategorydefaultApplicationProblemPlusJSONResponse{
			Body:       problem,
			StatusCode: http.StatusBadRequest,
		}, nil
	}

	input := service.CreateInput{
		Name:        request.Body.Name,
		Slug:        string(request.Body.Slug),
		Description: request.Body.Description,
	}

	if request.Body.ParentCategoryId != nil {
		parent := uuidFromExternal(*request.Body.ParentCategoryId)
		input.ParentID = &parent
	}

	category, err := h.svc.Create(ctx, audit, input)
	if err != nil {
		status, problem := h.problemForError(ctx, err, createOperation)
		return schemacategories.CreateSchemaCategorydefaultApplicationProblemPlusJSONResponse{
			Body:       problem,
			StatusCode: status,
		}, nil
	}

	location := fmt.Sprintf("%s/%s", schemaCategoriesBasePath, category.ID)
	return schemacategories.CreateSchemaCategory201JSONResponse{
		Body:    toAPICategory(category),
		Headers: schemacategories.CreateSchemaCategory201ResponseHeaders{Location: location},
	}, nil
}

func (h *Handler) DeleteSchemaCategory(ctx context.Context, request schemacategories.DeleteSchemaCategoryRequestObject) (schemacategories.DeleteSchemaCategoryResponseObject, error) {
	id := uuidFromExternal(request.CategoryId)
	audit := h.audit(ctx)
	if err := h.svc.Delete(ctx, audit, id); err != nil {
		status, problem := h.problemForError(ctx, err, deleteOperation)
		return schemacategories.DeleteSchemaCategorydefaultApplicationProblemPlusJSONResponse{
			Body:       problem,
			StatusCode: status,
		}, nil
	}

	return schemacategories.DeleteSchemaCategory204Response{}, nil
}

func (h *Handler) GetSchemaCategory(ctx context.Context, request schemacategories.GetSchemaCategoryRequestObject) (schemacategories.GetSchemaCategoryResponseObject, error) {
	audit := h.audit(ctx)
	category, err := h.svc.Get(ctx, audit, uuidFromExternal(request.CategoryId))
	if err != nil {
		status, problem := h.problemForError(ctx, err, getOperation)
		return schemacategories.GetSchemaCategorydefaultApplicationProblemPlusJSONResponse{
			Body:       problem,
			StatusCode: status,
		}, nil
	}

	return schemacategories.GetSchemaCategory200JSONResponse(toAPICategory(category)), nil
}

func (h *Handler) UpdateSchemaCategory(ctx context.Context, request schemacategories.UpdateSchemaCategoryRequestObject) (schemacategories.UpdateSchemaCategoryResponseObject, error) {
	audit := requesttrace.FromContextOrAnonymous(ctx)
	if request.Body == nil {
		problem := h.buildProblem("Invalid request body", "request body is required", problemTypeValidation, http.StatusBadRequest, nil)
		return schemacategories.UpdateSchemaCategorydefaultApplicationProblemPlusJSONResponse{
			Body:       problem,
			StatusCode: http.StatusBadRequest,
		}, nil
	}

	input := service.UpdateInput{
		Description: request.Body.Description,
	}

	if request.Body.Name != nil {
		name := *request.Body.Name
		input.Name = &name
	}

	if request.Body.ParentCategoryId != nil {
		parent := uuidFromExternal(*request.Body.ParentCategoryId)
		input.ParentID = &parent
	}

	if request.Body.Slug != nil {
		slug := string(*request.Body.Slug)
		input.Slug = &slug
	}

	category, err := h.svc.Update(ctx, audit, uuidFromExternal(request.CategoryId), input)
	if err != nil {
		status, problem := h.problemForError(ctx, err, updateOperation)
		return schemacategories.UpdateSchemaCategorydefaultApplicationProblemPlusJSONResponse{
			Body:       problem,
			StatusCode: status,
		}, nil
	}

	return schemacategories.UpdateSchemaCategory200JSONResponse(toAPICategory(category)), nil
}

func toAPICategory(category service.Category) schemacategories.SchemaCategory {
	apiCategory := schemacategories.SchemaCategory{
		CategoryId:  externalRef2.UUID(category.ID),
		Name:        category.Name,
		Slug:        externalRef2.Slug(category.Slug),
		CreatedAt:   externalRef2.Timestamp(category.CreatedAt),
		UpdatedAt:   externalRef2.Timestamp(category.UpdatedAt),
		Description: category.Description,
	}

	if category.ParentID != nil {
		parent := externalRef2.UUID(*category.ParentID)
		apiCategory.ParentCategoryId = &parent
	}

	if category.DeletedAt != nil {
		deleted := externalRef2.Timestamp(*category.DeletedAt)
		apiCategory.DeletedAt = &deleted
	}

	return apiCategory
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
		logger.Error("schema categories operation failed", append(fields, zap.Error(err))...)
	case status == http.StatusNotFound:
		logger.Info("schema categories resource not found", append(fields, zap.Error(err))...)
	default:
		logger.Warn("schema categories request rejected", append(fields, zap.Error(err))...)
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
			"schema category not found",
			problemTypeNotFound,
			nil
	case errors.Is(err, service.ErrConflict):
		return http.StatusConflict,
			"Conflict",
			"schema category already exists",
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
