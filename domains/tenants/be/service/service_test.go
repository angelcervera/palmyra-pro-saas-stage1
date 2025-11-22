package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	tenantsapi "github.com/zenGate-Global/palmyra-pro-saas/generated/go/tenants"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// inMemoryRepo is a minimal in-memory impl of Repository for tests.
type inMemoryRepo struct {
	mu   sync.Mutex
	data map[uuid.UUID]Tenant
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{data: make(map[uuid.UUID]Tenant)}
}

func (r *inMemoryRepo) List(ctx context.Context, opts ListOptions) (ListResult, error) {
	return ListResult{}, errors.New("not implemented")
}

func (r *inMemoryRepo) Create(ctx context.Context, t Tenant) (Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[t.ID] = t
	return t, nil
}

func (r *inMemoryRepo) Get(ctx context.Context, id uuid.UUID) (Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.data[id]
	if !ok {
		return Tenant{}, ErrNotFound
	}
	return t, nil
}

func (r *inMemoryRepo) AppendVersion(ctx context.Context, t Tenant) (Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[t.ID] = t
	return t, nil
}

func (r *inMemoryRepo) FindBySlug(ctx context.Context, slug string) (Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.data {
		if t.Slug == slug {
			return t, nil
		}
	}
	return Tenant{}, ErrNotFound
}

// stub provisioners

type stubDB struct {
	ensureRes DBProvisionResult
	ensureErr error
	checkRes  DBProvisionResult
	checkErr  error
}

func (s stubDB) Ensure(context.Context, DBProvisionRequest) (DBProvisionResult, error) {
	return s.ensureRes, s.ensureErr
}
func (s stubDB) Check(context.Context, DBProvisionRequest) (DBProvisionResult, error) {
	return s.checkRes, s.checkErr
}

type stubAuth struct {
	ensureRes AuthProvisionResult
	ensureErr error
	checkRes  AuthProvisionResult
	checkErr  error
}

func (s stubAuth) Ensure(context.Context, string) (AuthProvisionResult, error) {
	return s.ensureRes, s.ensureErr
}
func (s stubAuth) Check(context.Context, string) (AuthProvisionResult, error) {
	return s.checkRes, s.checkErr
}

type stubStorage struct {
	res StorageProvisionResult
	err error
}

func (s stubStorage) Ensure(context.Context, string) (StorageProvisionResult, error) {
	return s.res, s.err
}
func (s stubStorage) Check(context.Context, string) (StorageProvisionResult, error) {
	return s.res, s.err
}

func newTenantRecord(slug string) Tenant {
	id := uuid.New()
	schema := tenant.BuildSchemaName(tenant.ToSnake(slug))
	return Tenant{
		ID:            id,
		Version:       persistence.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		Slug:          slug,
		Status:        tenantsapi.Pending,
		SchemaName:    schema,
		RoleName:      tenant.BuildRoleName(schema),
		BasePrefix:    tenant.BuildBasePrefix("dev", slug, tenant.ShortID(id)),
		ShortTenantID: tenant.ShortID(id),
		CreatedAt:     time.Now().UTC(),
		CreatedBy:     uuid.New(),
		Provisioning:  ProvisioningStatus{},
	}
}

func TestProvisionHappyPath(t *testing.T) {
	repo := newInMemoryRepo()
	tenantRecord := newTenantRecord("acme-co")
	_, _ = repo.Create(context.Background(), tenantRecord)

	deps := ProvisioningDeps{
		DB:      stubDB{ensureRes: DBProvisionResult{Ready: true}},
		Auth:    stubAuth{ensureRes: AuthProvisionResult{Ready: true}},
		Storage: stubStorage{res: StorageProvisionResult{Ready: true}},
	}

	svc := NewWithProvisioning(repo, "dev", deps)

	updated, err := svc.Provision(context.Background(), tenantRecord.ID)
	require.NoError(t, err)
	require.Equal(t, tenantsapi.Active, updated.Status)
	require.True(t, updated.Provisioning.DBReady)
	require.True(t, updated.Provisioning.AuthReady)
	require.NotNil(t, updated.Provisioning.LastProvisionedAt)
	require.Nil(t, updated.Provisioning.LastError)
}

func TestProvisionPartialFailureKeepsFlags(t *testing.T) {
	repo := newInMemoryRepo()
	tenantRecord := newTenantRecord("beta-co")
	_, _ = repo.Create(context.Background(), tenantRecord)

	deps := ProvisioningDeps{
		DB:      stubDB{ensureRes: DBProvisionResult{Ready: false}, ensureErr: errors.New("db failed")},
		Auth:    stubAuth{ensureRes: AuthProvisionResult{Ready: true}},
		Storage: stubStorage{res: StorageProvisionResult{Ready: true}},
	}

	svc := NewWithProvisioning(repo, "dev", deps)

	updated, err := svc.Provision(context.Background(), tenantRecord.ID)
	require.NoError(t, err)
	require.Equal(t, tenantsapi.Provisioning, updated.Status)
	require.False(t, updated.Provisioning.DBReady)
	require.True(t, updated.Provisioning.AuthReady)
	require.Nil(t, updated.Provisioning.LastProvisionedAt)
	require.NotNil(t, updated.Provisioning.LastError)
}

func TestProvisionStatusPromotesWhenReady(t *testing.T) {
	repo := newInMemoryRepo()
	tenantRecord := newTenantRecord("gamma-co")
	tenantRecord.Status = tenantsapi.Provisioning
	_, _ = repo.Create(context.Background(), tenantRecord)

	deps := ProvisioningDeps{
		DB:      stubDB{checkRes: DBProvisionResult{Ready: true}},
		Auth:    stubAuth{checkRes: AuthProvisionResult{Ready: true}},
		Storage: stubStorage{res: StorageProvisionResult{Ready: true}},
	}

	svc := NewWithProvisioning(repo, "dev", deps)

	status, err := svc.ProvisionStatus(context.Background(), tenantRecord.ID)
	require.NoError(t, err)
	require.True(t, status.DBReady)
	require.True(t, status.AuthReady)
	require.NotNil(t, status.LastProvisionedAt)

	updated, _ := repo.Get(context.Background(), tenantRecord.ID)
	require.Equal(t, tenantsapi.Active, updated.Status)
}
