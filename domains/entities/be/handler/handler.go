package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/entities/be/service"
	externalPrimitives "github.com/zenGate-Global/palmyra-pro-saas/generated/go/common/primitives"
	externalProblems "github.com/zenGate-Global/palmyra-pro-saas/generated/go/common/problemdetails"
	entitiesapi "github.com/zenGate-Global/palmyra-pro-saas/generated/go/entities"
)

const (
	problemTypeValidation = "https://palmyra.pro/problems/validation-error"
	problemTypeNotFound   = "https://palmyra.pro/problems/not-found"
	problemTypeConflict   = "https://palmyra.pro/problems/conflict"
	problemTypeInternal   = "https://palmyra.pro/problems/internal-error"
)

// Handler wires the entities service to the generated HTTP contract.
type Handler struct {
	svc    service.Service
	logger *zap.Logger
}

// New constructs a Handler instance.
func New(svc service.Service, logger *zap.Logger) *Handler {
	if svc == nil {
		panic("entities service is required")
	}
	if logger == nil {
		panic("logger is required")
	}

	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) ListDocuments(ctx context.Context, request entitiesapi.ListDocumentsRequestObject) (entitiesapi.ListDocumentsResponseObject, error) {
	page := 1
	if request.Params.Page != nil {
		page = int(*request.Params.Page)
	}
	pageSize := 20
	if request.Params.PageSize != nil {
		pageSize = int(*request.Params.PageSize)
	}
	sort := ""
	if request.Params.Sort != nil {
		sort = string(*request.Params.Sort)
	}

	result, err := h.svc.List(ctx, string(request.TableName), service.ListOptions{
		Page:     page,
		PageSize: pageSize,
		Sort:     sort,
	})
	if err != nil {
		status, problem := h.problemForError(err)
		return entitiesapi.ListDocumentsdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	items := make([]entitiesapi.EntityDocument, 0, len(result.Items))
	for _, doc := range result.Items {
		apiDoc, convErr := toAPIDocument(doc)
		if convErr != nil {
			status, problem := h.problemForInternal(convErr)
			return entitiesapi.ListDocumentsdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
		}
		items = append(items, apiDoc)
	}

	return entitiesapi.ListDocuments200JSONResponse{
		Items:      items,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalItems: int(result.TotalItems),
		TotalPages: result.TotalPages,
	}, nil
}

func (h *Handler) CreateDocument(ctx context.Context, request entitiesapi.CreateDocumentRequestObject) (entitiesapi.CreateDocumentResponseObject, error) {
	if request.Body == nil || request.Body.Payload == nil {
		status, problem := h.validationProblem("payload is required")
		return entitiesapi.CreateDocumentdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	doc, err := h.svc.Create(ctx, string(request.TableName), request.Body.Payload)
	if err != nil {
		status, problem := h.problemForError(err)
		return entitiesapi.CreateDocumentdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	apiDoc, convErr := toAPIDocument(doc)
	if convErr != nil {
		status, problem := h.problemForInternal(convErr)
		return entitiesapi.CreateDocumentdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	location := fmt.Sprintf("/api/v1/entities/%s/documents/%s", request.TableName, doc.EntityID)

	return entitiesapi.CreateDocument201JSONResponse{
		Body: apiDoc,
		Headers: entitiesapi.CreateDocument201ResponseHeaders{
			Location: location,
		},
	}, nil
}

func (h *Handler) GetDocument(ctx context.Context, request entitiesapi.GetDocumentRequestObject) (entitiesapi.GetDocumentResponseObject, error) {
	doc, err := h.svc.Get(ctx, string(request.TableName), uuid.UUID(request.EntityId))
	if err != nil {
		status, problem := h.problemForError(err)
		return entitiesapi.GetDocumentdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	apiDoc, convErr := toAPIDocument(doc)
	if convErr != nil {
		status, problem := h.problemForInternal(convErr)
		return entitiesapi.GetDocumentdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	return entitiesapi.GetDocument200JSONResponse(apiDoc), nil
}

func (h *Handler) UpdateDocument(ctx context.Context, request entitiesapi.UpdateDocumentRequestObject) (entitiesapi.UpdateDocumentResponseObject, error) {
	if request.Body == nil || request.Body.Payload == nil {
		status, problem := h.validationProblem("payload is required")
		return entitiesapi.UpdateDocumentdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	doc, err := h.svc.Update(ctx, string(request.TableName), uuid.UUID(request.EntityId), *request.Body.Payload)
	if err != nil {
		status, problem := h.problemForError(err)
		return entitiesapi.UpdateDocumentdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	apiDoc, convErr := toAPIDocument(doc)
	if convErr != nil {
		status, problem := h.problemForInternal(convErr)
		return entitiesapi.UpdateDocumentdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	return entitiesapi.UpdateDocument200JSONResponse(apiDoc), nil
}

func (h *Handler) DeleteDocument(ctx context.Context, request entitiesapi.DeleteDocumentRequestObject) (entitiesapi.DeleteDocumentResponseObject, error) {
	if err := h.svc.Delete(ctx, string(request.TableName), uuid.UUID(request.EntityId)); err != nil {
		status, problem := h.problemForError(err)
		return entitiesapi.DeleteDocumentdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	return entitiesapi.DeleteDocument204Response{}, nil
}

func toAPIDocument(doc service.Document) (entitiesapi.EntityDocument, error) {
	payload := map[string]interface{}{}
	if doc.Payload != nil {
		for k, v := range doc.Payload {
			payload[k] = v
		}
	}

	apiDoc := entitiesapi.EntityDocument{
		EntityId:      externalPrimitives.UUID(doc.EntityID),
		EntityVersion: externalPrimitives.SemanticVersion(doc.EntityVersion.String()),
		SchemaId:      externalPrimitives.UUID(doc.SchemaID),
		SchemaVersion: externalPrimitives.SemanticVersion(doc.SchemaVersion.String()),
		Payload:       payload,
		CreatedAt:     externalPrimitives.Timestamp(doc.CreatedAt),
		IsActive:      doc.IsActive,
		IsSoftDeleted: doc.IsSoftDeleted,
	}

	return apiDoc, nil
}

func (h *Handler) validationProblem(detail string) (int, externalProblems.ProblemDetails) {
	problem := externalProblems.ProblemDetails{
		Type:   strPtr(problemTypeValidation),
		Title:  "Validation error",
		Detail: strPtr(detail),
		Status: http.StatusBadRequest,
	}
	return http.StatusBadRequest, problem
}

func (h *Handler) problemForError(err error) (int, externalProblems.ProblemDetails) {
	var validationErr *service.ValidationError
	if errors.As(err, &validationErr) {
		return h.validationProblem(validationErr.Error())
	}

	if errors.Is(err, service.ErrTableNotFound) || errors.Is(err, service.ErrDocumentNotFound) {
		problem := externalProblems.ProblemDetails{
			Type:   strPtr(problemTypeNotFound),
			Title:  "Not found",
			Detail: strPtr("resource not found"),
			Status: http.StatusNotFound,
		}
		return http.StatusNotFound, problem
	}

	if errors.Is(err, service.ErrConflict) {
		problem := externalProblems.ProblemDetails{
			Type:   strPtr(problemTypeConflict),
			Title:  "Conflict",
			Detail: strPtr("entity already exists"),
			Status: http.StatusConflict,
		}
		return http.StatusConflict, problem
	}

	return h.problemForInternal(err)
}

func (h *Handler) problemForInternal(err error) (int, externalProblems.ProblemDetails) {
	if h.logger != nil {
		h.logger.Error("entities handler", zap.Error(err))
	}
	problem := externalProblems.ProblemDetails{
		Type:   strPtr(problemTypeInternal),
		Title:  "Internal error",
		Detail: strPtr("unexpected error"),
		Status: http.StatusInternalServerError,
	}
	return http.StatusInternalServerError, problem
}

func strPtr(value string) *string {
	return &value
}

// compile-time assertions to ensure interface compliance
var _ entitiesapi.StrictServerInterface = (*Handler)(nil)
