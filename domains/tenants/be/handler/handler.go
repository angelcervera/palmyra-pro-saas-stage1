package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
	externalPrimitives "github.com/zenGate-Global/palmyra-pro-saas/generated/go/common/primitives"
	externalProblems "github.com/zenGate-Global/palmyra-pro-saas/generated/go/common/problemdetails"
	tenantsapi "github.com/zenGate-Global/palmyra-pro-saas/generated/go/tenants"
	platformauth "github.com/zenGate-Global/palmyra-pro-saas/platform/go/auth"
)

const (
	problemTypeValidation = "https://palmyra.pro/problems/validation-error"
	problemTypeNotFound   = "https://palmyra.pro/problems/not-found"
	problemTypeConflict   = "https://palmyra.pro/problems/conflict"
	problemTypeInternal   = "https://palmyra.pro/problems/internal-error"
)

// Handler wires tenants service to generated HTTP contract.
type Handler struct {
	svc    *service.Service
	logger *zap.Logger
}

// New constructs a Handler instance.
func New(svc *service.Service, logger *zap.Logger) *Handler {
	if svc == nil {
		panic("tenants service is required")
	}
	if logger == nil {
		panic("logger is required")
	}
	return &Handler{svc: svc, logger: logger}
}

// TenantsList implements GET /admin/tenants
func (h *Handler) TenantsList(ctx context.Context, request tenantsapi.TenantsListRequestObject) (tenantsapi.TenantsListResponseObject, error) {
	opts := buildListOptions(request.Params)
	result, err := h.svc.List(ctx, opts)
	if err != nil {
		status, problem := h.problemForError(ctx, err, http.StatusInternalServerError)
		return tenantsapi.TenantsListdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: status}, nil
	}

	items := make([]tenantsapi.Tenant, 0, len(result.Tenants))
	for _, t := range result.Tenants {
		items = append(items, toAPITenant(t))
	}

	return tenantsapi.TenantsList200JSONResponse{
		Items:      items,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalItems: result.TotalItems,
		TotalPages: result.TotalPages,
	}, nil
}

// TenantsCreate implements POST /admin/tenants
func (h *Handler) TenantsCreate(ctx context.Context, request tenantsapi.TenantsCreateRequestObject) (tenantsapi.TenantsCreateResponseObject, error) {
	if request.Body == nil {
		problem := h.buildProblem("Invalid request body", "request body is required", problemTypeValidation, http.StatusBadRequest, nil)
		return tenantsapi.TenantsCreatedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: http.StatusBadRequest}, nil
	}

	createdBy, err := h.extractAdminID(ctx)
	if err != nil {
		problem := h.buildProblem("Forbidden", err.Error(), problemTypeValidation, http.StatusForbidden, nil)
		return tenantsapi.TenantsCreatedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: http.StatusForbidden}, nil
	}

	status := tenantsapi.Active
	if request.Body.Status != nil {
		status = *request.Body.Status
	}

	input := service.CreateInput{
		Slug:        string(request.Body.Slug),
		DisplayName: request.Body.DisplayName,
		Status:      status,
		CreatedBy:   createdBy,
	}

	t, err := h.svc.Create(ctx, input)
	if err != nil {
		statusCode, problem := h.problemForError(ctx, err, http.StatusInternalServerError)
		return tenantsapi.TenantsCreatedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: statusCode}, nil
	}

	location := fmt.Sprintf("/api/v1/admin/tenants/%s", t.ID)
	return tenantsapi.TenantsCreate201JSONResponse{
		Headers: tenantsapi.TenantsCreate201ResponseHeaders{Location: location},
		Body:    toAPITenant(t),
	}, nil
}

// TenantsGet implements GET /admin/tenants/{tenantId}
func (h *Handler) TenantsGet(ctx context.Context, request tenantsapi.TenantsGetRequestObject) (tenantsapi.TenantsGetResponseObject, error) {
	t, err := h.svc.Get(ctx, uuid.UUID(request.TenantId))
	if err != nil {
		statusCode, problem := h.problemForError(ctx, err, http.StatusNotFound)
		return tenantsapi.TenantsGetdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: statusCode}, nil
	}
	return tenantsapi.TenantsGet200JSONResponse(toAPITenant(t)), nil
}

// TenantsUpdate implements PATCH /admin/tenants/{tenantId}
func (h *Handler) TenantsUpdate(ctx context.Context, request tenantsapi.TenantsUpdateRequestObject) (tenantsapi.TenantsUpdateResponseObject, error) {
	if request.Body == nil {
		problem := h.buildProblem("Invalid request body", "request body is required", problemTypeValidation, http.StatusBadRequest, nil)
		return tenantsapi.TenantsUpdatedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: http.StatusBadRequest}, nil
	}

	input := service.UpdateInput{
		DisplayName: request.Body.DisplayName,
		Status:      request.Body.Status,
	}

	updated, err := h.svc.Update(ctx, uuid.UUID(request.TenantId), input)
	if err != nil {
		statusCode, problem := h.problemForError(ctx, err, http.StatusInternalServerError)
		return tenantsapi.TenantsUpdatedefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: statusCode}, nil
	}

	return tenantsapi.TenantsUpdate200JSONResponse(toAPITenant(updated)), nil
}

// TenantsProvision implements POST /admin/tenants/{tenantId}:provision
func (h *Handler) TenantsProvision(ctx context.Context, request tenantsapi.TenantsProvisionRequestObject) (tenantsapi.TenantsProvisionResponseObject, error) {
	t, err := h.svc.Provision(ctx, uuid.UUID(request.TenantId))
	if err != nil {
		statusCode, problem := h.problemForError(ctx, err, http.StatusInternalServerError)
		return tenantsapi.TenantsProvisiondefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: statusCode}, nil
	}
	return tenantsapi.TenantsProvision202JSONResponse(toAPITenant(t)), nil
}

// TenantsProvisionStatus implements GET /admin/tenants/{tenantId}:provision-status
func (h *Handler) TenantsProvisionStatus(ctx context.Context, request tenantsapi.TenantsProvisionStatusRequestObject) (tenantsapi.TenantsProvisionStatusResponseObject, error) {
	status, err := h.svc.ProvisionStatus(ctx, uuid.UUID(request.TenantId))
	if err != nil {
		code, problem := h.problemForError(ctx, err, http.StatusInternalServerError)
		return tenantsapi.TenantsProvisionStatusdefaultApplicationProblemPlusJSONResponse{Body: problem, StatusCode: code}, nil
	}
	return tenantsapi.TenantsProvisionStatus200JSONResponse(toAPIProvisioningStatus(status)), nil
}

func (h *Handler) extractAdminID(ctx context.Context) (uuid.UUID, error) {
	creds, ok := platformauth.UserFromContext(ctx)
	if !ok || creds == nil {
		return uuid.Nil, errors.New("missing credentials")
	}
	if !creds.IsAdmin {
		return uuid.Nil, errors.New("admin role required")
	}
	id, err := uuid.Parse(creds.Id)
	if err != nil {
		return uuid.Nil, errors.New("invalid admin id")
	}
	return id, nil
}

func (h *Handler) problemForError(ctx context.Context, err error, defaultStatus int) (int, externalProblems.ProblemDetails) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		return http.StatusNotFound, h.buildProblem("Not found", err.Error(), problemTypeNotFound, http.StatusNotFound, nil)
	case errors.Is(err, service.ErrConflictSlug):
		return http.StatusConflict, h.buildProblem("Conflict", err.Error(), problemTypeConflict, http.StatusConflict, nil)
	default:
		h.logger.Error("tenant operation failed", zap.Error(err))
		return defaultStatus, h.buildProblem("Internal error", "internal error", problemTypeInternal, http.StatusInternalServerError, nil)
	}
}

func (h *Handler) buildProblem(title, detail, problemType string, status int, errs map[string][]string) externalProblems.ProblemDetails {
	return externalProblems.ProblemDetails{
		Title:  title,
		Detail: strPtr(detail),
		Status: status,
		Type:   strPtr(problemType),
		Errors: mapPtr(errs),
	}
}

func buildListOptions(params tenantsapi.TenantsListParams) service.ListOptions {
	opts := service.ListOptions{Page: 1, PageSize: 20}
	if params.Page != nil {
		opts.Page = int(*params.Page)
	}
	if params.PageSize != nil {
		opts.PageSize = int(*params.PageSize)
	}
	if params.Status != nil {
		opts.Status = params.Status
	}
	return opts
}

func toAPITenant(t service.Tenant) tenantsapi.Tenant {
	return tenantsapi.Tenant{
		TenantId:      externalPrimitives.UUID(t.ID),
		Slug:          externalPrimitives.Slug(t.Slug),
		DisplayName:   t.DisplayName,
		Status:        t.Status,
		SchemaName:    &t.SchemaName,
		BasePrefix:    &t.BasePrefix,
		ShortTenantId: &t.ShortTenantID,
		CreatedAt:     externalPrimitives.Timestamp(t.CreatedAt),
		CreatedBy:     externalPrimitives.UUID(t.CreatedBy),
		Provisioning:  toAPIProvisioningStatus(t.Provisioning),
	}
}

func toAPIProvisioningStatus(p service.ProvisioningStatus) tenantsapi.TenantProvisioningStatus {
	return tenantsapi.TenantProvisioningStatus{
		DbReady:           &p.DBReady,
		AuthReady:         &p.AuthReady,
		StorageReady:      &p.StorageReady,
		LastProvisionedAt: (*externalPrimitives.Timestamp)(p.LastProvisionedAt),
		LastError:         p.LastError,
	}
}

func strPtr(v string) *string {
	return &v
}

func mapPtr(m map[string][]string) *map[string][]string {
	if m == nil {
		return nil
	}
	return &m
}
