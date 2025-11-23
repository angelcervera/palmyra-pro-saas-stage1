package persistence

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sqlassets "github.com/zenGate-Global/palmyra-pro-saas/database"
)

func TestTenantRepositoryLifecycle(t *testing.T) {
	if _, ok := os.LookupEnv("TEST_DATABASE_URL"); !ok {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	pool, cleanup := mustTestPool(t)
	defer cleanup()

	store, err := NewTenantStore(ctx, pool, "tenant_admin")
	require.NoError(t, err)
	repo := store

	tenantID := uuid.New()
	createdBy := uuid.New()
	rec := TenantRecord{
		TenantID:      tenantID,
		TenantVersion: SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		Slug:          "acme-co",
		DisplayName:   strPtr("Acme Co"),
		Status:        "pending",
		SchemaName:    "tenant_acme_co",
		RoleName:      "tenant_acme_co_role",
		BasePrefix:    "dev/acme-co-12345678/",
		ShortTenantID: "12345678",
		IsActive:      true,
		CreatedAt:     time.Now().UTC(),
		CreatedBy:     createdBy,
	}

	inserted, err := repo.Create(ctx, rec)
	require.NoError(t, err)
	require.Equal(t, rec.Slug, inserted.Slug)
	require.True(t, inserted.IsActive)

	fetched, err := repo.GetActive(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, inserted.TenantVersion, fetched.TenantVersion)

	// Append new version with status active and provisioning set.
	now := time.Now().UTC()
	rec2 := inserted
	rec2.TenantVersion = SemanticVersion{Major: 1, Minor: 0, Patch: 1}
	rec2.Status = "active"
	rec2.DBReady = true
	rec2.AuthReady = true
	rec2.LastProvisionedAt = &now

	appended, err := repo.AppendVersion(ctx, rec2)
	require.NoError(t, err)
	require.Equal(t, rec2.TenantVersion, appended.TenantVersion)
	require.True(t, appended.DBReady)
	require.True(t, appended.AuthReady)

	// Active pointer should now be the new version.
	active, err := repo.GetActive(ctx, tenantID)
	require.NoError(t, err)
	require.Equal(t, rec2.TenantVersion, active.TenantVersion)

	// Listing should return only active versions (1 item) for default includeInactive=false.
	records, total, err := repo.ListActive(ctx, nil, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, records, 1)
	require.Equal(t, tenantID, records[0].TenantID)
}

func strPtr(s string) *string { return &s }

func TestTenantRepositoryUsesConfiguredSchema(t *testing.T) {
	if _, ok := os.LookupEnv("TEST_DATABASE_URL"); !ok {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	pool, cleanup := mustTestPool(t)
	defer cleanup()

	schema := "tenant_test_schema"
	require.NoError(t, applyDDLToSchema(ctx, pool, schema, sqlassets.TenantsSQL))

	store, err := NewTenantStore(ctx, pool, schema)
	require.NoError(t, err)

	repo := store

	tenantID := uuid.New()
	rec := TenantRecord{
		TenantID:      tenantID,
		TenantVersion: SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		Slug:          "acme-schema-check",
		Status:        "active",
		SchemaName:    "tenant_acme_schema_check",
		RoleName:      "tenant_acme_schema_check_role",
		BasePrefix:    "dev/acme-schema-check-12345678/",
		ShortTenantID: "12345678",
		IsActive:      true,
		CreatedAt:     time.Now().UTC(),
		CreatedBy:     uuid.New(),
	}

	_, err = repo.Create(ctx, rec)
	require.NoError(t, err)

	// Verify row exists in configured schema.
	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM `+schema+`.tenants WHERE tenant_id = $1`, tenantID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Verify table in default schema (if present) does not contain the row.
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM tenants WHERE tenant_id = $1`, tenantID).Scan(&count)
	require.Equal(t, 0, count)
}
