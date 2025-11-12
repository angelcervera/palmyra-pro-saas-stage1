package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/users/be/service"
	externalRef2 "github.com/zenGate-Global/palmyra-pro-saas/generated/go/common/primitives"
	externalRef3 "github.com/zenGate-Global/palmyra-pro-saas/generated/go/common/problemdetails"
	users "github.com/zenGate-Global/palmyra-pro-saas/generated/go/users"
	platformauth "github.com/zenGate-Global/palmyra-pro-saas/platform/go/auth"
	platformlogging "github.com/zenGate-Global/palmyra-pro-saas/platform/go/logging"
)

const (
	problemTypeValidation = "https://tcg.land/problems/validation-error"
	problemTypeNotFound   = "https://tcg.land/problems/not-found"
	problemTypeConflict   = "https://tcg.land/problems/conflict"
	problemTypeInternal   = "https://tcg.land/problems/internal-error"
)

type operation string

const (
	createOperation   operation = "usersCreate"
	listOperation     operation = "usersList"
	getOperation      operation = "usersGet"
	updateOperation   operation = "usersUpdate"
	meGetOperation    operation = "usersMe"
	meUpdateOperation operation = "usersUpdateMe"
	deleteOperation   operation = "usersDelete"
)

// Handler wires the users service to the generated HTTP contract.
type Handler struct {
	svc    service.Service
	logger *zap.Logger
}

// New constructs a Handler instance.
func New(svc service.Service, logger *zap.Logger) *Handler {
	if svc == nil {
		panic("users service is required")
	}
	if logger == nil {
		panic("logger is required")
	}

	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) UsersList(ctx context.Context, request users.UsersListRequestObject) (users.UsersListResponseObject, error) {
	opts := buildListOptions(request.Params)

	result, err := h.svc.List(ctx, opts)
	if err != nil {
		status, problem := h.problemForError(ctx, err, listOperation)
		return users.UsersListdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	items := make([]users.User, 0, len(result.Users))
	for _, user := range result.Users {
		items = append(items, toAPIUser(user))
	}

	return users.UsersList200JSONResponse{
		Items:      items,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages,
	}, nil
}

func (h *Handler) UsersCreate(ctx context.Context, request users.UsersCreateRequestObject) (users.UsersCreateResponseObject, error) {
	if request.Body == nil {
		problem := h.buildProblem("Invalid request body", "request body is required", problemTypeValidation, http.StatusBadRequest, nil)
		return users.UsersCreatedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: http.StatusBadRequest}, nil
	}

	input := toServiceCreateInput(request.Body)

	created, err := h.svc.Create(ctx, input)
	if err != nil {
		status, problem := h.problemForError(ctx, err, createOperation)
		return users.UsersCreatedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	location := fmt.Sprintf("/api/v1/admin/users/%s", created.ID.String())

	return users.UsersCreate201JSONResponse{
		Headers: users.UsersCreate201ResponseHeaders{Location: location},
		Body:    toAPIUser(created),
	}, nil
}

func (h *Handler) UsersGet(ctx context.Context, request users.UsersGetRequestObject) (users.UsersGetResponseObject, error) {
	user, err := h.svc.Get(ctx, uuid.UUID(request.UserId))
	if err != nil {
		status, problem := h.problemForError(ctx, err, getOperation)
		return users.UsersGetdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	return users.UsersGet200JSONResponse(toAPIUser(user)), nil
}

func (h *Handler) UsersUpdate(ctx context.Context, request users.UsersUpdateRequestObject) (users.UsersUpdateResponseObject, error) {
	if request.Body == nil {
		problem := h.buildProblem("Invalid request body", "request body is required", problemTypeValidation, http.StatusBadRequest, nil)
		return users.UsersUpdatedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: http.StatusBadRequest}, nil
	}

	input := toServiceUpdateInput(request.Body)

	updated, err := h.svc.Update(ctx, uuid.UUID(request.UserId), input)
	if err != nil {
		status, problem := h.problemForError(ctx, err, updateOperation)
		return users.UsersUpdatedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	return users.UsersUpdate200JSONResponse(toAPIUser(updated)), nil
}

func (h *Handler) UsersMe(ctx context.Context, _ users.UsersMeRequestObject) (users.UsersMeResponseObject, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		problem := h.buildProblem("Unauthorized", err.Error(), problemTypeValidation, http.StatusUnauthorized, nil)
		return users.UsersMedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: http.StatusUnauthorized}, nil
	}

	user, svcErr := h.svc.Get(ctx, userID)
	if svcErr != nil {
		status, problem := h.problemForError(ctx, svcErr, meGetOperation)
		return users.UsersMedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	return users.UsersMe200JSONResponse(toAPIUser(user)), nil
}

func (h *Handler) UsersUpdateMe(ctx context.Context, request users.UsersUpdateMeRequestObject) (users.UsersUpdateMeResponseObject, error) {
	if request.Body == nil {
		problem := h.buildProblem("Invalid request body", "request body is required", problemTypeValidation, http.StatusBadRequest, nil)
		return users.UsersUpdateMedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: http.StatusBadRequest}, nil
	}

	userID, err := h.extractUserID(ctx)
	if err != nil {
		problem := h.buildProblem("Unauthorized", err.Error(), problemTypeValidation, http.StatusUnauthorized, nil)
		return users.UsersUpdateMedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: http.StatusUnauthorized}, nil
	}

	input := service.UpdateSelfInput{FullName: request.Body.FullName}

	updated, svcErr := h.svc.UpdateSelf(ctx, userID, input)
	if svcErr != nil {
		status, problem := h.problemForError(ctx, svcErr, meUpdateOperation)
		return users.UsersUpdateMedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	return users.UsersUpdateMe200JSONResponse(toAPIUser(updated)), nil
}

func (h *Handler) UsersDelete(ctx context.Context, request users.UsersDeleteRequestObject) (users.UsersDeleteResponseObject, error) {
	if err := h.svc.Delete(ctx, uuid.UUID(request.UserId)); err != nil {
		status, problem := h.problemForError(ctx, err, deleteOperation)
		return users.UsersDeletedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	return users.UsersDelete204Response{}, nil
}

func buildListOptions(params users.UsersListParams) service.ListOptions {
	opts := service.ListOptions{}

	if params.Page != nil {
		opts.Page = int(*params.Page)
	}
	if params.PageSize != nil {
		opts.PageSize = int(*params.PageSize)
	}
	if params.Email != nil {
		email := strings.TrimSpace(*params.Email)
		opts.Email = &email
	}
	if params.Sort != nil {
		s := string(*params.Sort)
		opts.Sort = &s
	}

	return opts
}

func toAPIUser(user service.User) users.User {
	return users.User{
		Id:        externalRef2.UUID(user.ID),
		Email:     externalRef2.Email(user.Email),
		FullName:  user.FullName,
		CreatedAt: externalRef2.Timestamp(user.CreatedAt),
		UpdatedAt: externalRef2.Timestamp(user.UpdatedAt),
	}
}

func toServiceCreateInput(body *users.CreateUser) service.CreateInput {
	input := service.CreateInput{
		Email:    string(body.Email),
		FullName: body.FullName,
	}

	return input
}

func toServiceUpdateInput(body *users.UsersUpdateJSONRequestBody) service.UpdateInput {
	input := service.UpdateInput{}

	if body.FullName != nil {
		input.FullName = body.FullName
	}

	return input
}

func (h *Handler) extractUserID(ctx context.Context) (uuid.UUID, error) {
	credentials, ok := platformauth.UserFromContext(ctx)
	if !ok || credentials == nil {
		return uuid.Nil, errors.New("missing credentials")
	}

	id, err := uuid.Parse(credentials.Id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user id")
	}

	return id, nil
}

func (h *Handler) problemForError(ctx context.Context, err error, op operation) (int, externalRef3.ProblemDetails) {
	status, title, detail, problemType, fields := h.classifyError(err)

	logger := h.loggerFrom(ctx)
	fieldsForLog := []zap.Field{
		zap.String("operation", string(op)),
		zap.Int("status", status),
	}

	switch {
	case status >= http.StatusInternalServerError:
		logger.Error("users operation failed", append(fieldsForLog, zap.Error(err))...)
	case status == http.StatusNotFound:
		logger.Info("users resource not found", append(fieldsForLog, zap.Error(err))...)
	default:
		logger.Warn("users request rejected", append(fieldsForLog, zap.Error(err))...)
	}

	return status, h.buildProblem(title, detail, problemType, status, fields)
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
			"user not found",
			problemTypeNotFound,
			nil
	case errors.Is(err, service.ErrConflict):
		return http.StatusConflict,
			"Conflict",
			"user conflict",
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
