package repo

import (
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
)

// MemoryRepository is a simple in-memory implementation suitable for tests and early development.
type MemoryRepository struct {
	mu     sync.RWMutex
	byID   map[uuid.UUID]service.Tenant
	bySlug map[string]uuid.UUID
}

// NewMemoryRepository constructs a MemoryRepository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{byID: make(map[uuid.UUID]service.Tenant), bySlug: make(map[string]uuid.UUID)}
}

func (r *MemoryRepository) List(ctx context.Context, opts service.ListOptions) (service.ListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]service.Tenant, 0, len(r.byID))
	for _, t := range r.byID {
		if opts.Status != nil && t.Status != *opts.Status {
			continue
		}
		items = append(items, t)
	}

	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })

	page := opts.Page
	if page < 1 {
		page = 1
	}
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(items) {
		start = len(items)
	}
	if end > len(items) {
		end = len(items)
	}

	paged := items[start:end]
	totalPages := (len(items) + pageSize - 1) / pageSize

	return service.ListResult{
		Tenants:    paged,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: len(items),
		TotalPages: totalPages,
	}, nil
}

func (r *MemoryRepository) Create(ctx context.Context, t service.Tenant) (service.Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.bySlug[t.Slug]; exists {
		return service.Tenant{}, service.ErrConflictSlug
	}

	r.byID[t.ID] = t
	r.bySlug[t.Slug] = t.ID
	return t, nil
}

func (r *MemoryRepository) Get(ctx context.Context, id uuid.UUID) (service.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.byID[id]
	if !ok {
		return service.Tenant{}, service.ErrNotFound
	}
	return t, nil
}

func (r *MemoryRepository) Update(ctx context.Context, id uuid.UUID, input service.UpdateInput) (service.Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.byID[id]
	if !ok {
		return service.Tenant{}, service.ErrNotFound
	}

	if input.DisplayName != nil {
		t.DisplayName = input.DisplayName
	}
	if input.Status != nil {
		t.Status = *input.Status
	}

	r.byID[id] = t
	return t, nil
}

func (r *MemoryRepository) UpdateProvisioning(ctx context.Context, id uuid.UUID, status service.ProvisioningStatus) (service.Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.byID[id]
	if !ok {
		return service.Tenant{}, service.ErrNotFound
	}

	t.Provisioning = status
	r.byID[id] = t
	return t, nil
}

func (r *MemoryRepository) FindBySlug(ctx context.Context, slug string) (service.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.bySlug[slug]
	if !ok {
		return service.Tenant{}, service.ErrNotFound
	}
	return r.byID[id], nil
}

// Ensure interface compliance.
var _ service.Repository = (*MemoryRepository)(nil)
