package storage

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

func TestResolveObjectLocation(t *testing.T) {
	space := tenant.Space{
		TenantID:      uuid.New(),
		Slug:          "acme-co",
		ShortTenantID: "12345678",
		SchemaName:    "tenant_acme_co",
		BasePrefix:    "dev/acme-co-12345678/",
	}

	entityID := uuid.New()
	loc, err := ResolveObjectLocation(space, "palmyra-dev-assets", "entities/cards_entities/"+entityID.String()+"/logo/1.0.0/file.pdf")
	require.NoError(t, err)
	require.Equal(t, "palmyra-dev-assets", loc.Bucket)
	require.Equal(t, "dev/acme-co-12345678/entities/cards_entities/"+entityID.String()+"/logo/1.0.0/file.pdf", loc.FullPath)
}

func TestResolveObjectLocation_trimsSlashAndValidates(t *testing.T) {
	space := tenant.Space{
		TenantID:      uuid.New(),
		Slug:          "acme-co",
		ShortTenantID: "12345678",
		SchemaName:    "tenant_acme_co",
		BasePrefix:    "dev/acme-co-12345678", // no trailing slash
	}

	loc, err := ResolveObjectLocation(space, "bucket", "/avatars/user.png")
	require.NoError(t, err)
	require.Equal(t, "dev/acme-co-12345678/avatars/user.png", loc.FullPath)

	_, err = ResolveObjectLocation(space, "", "file")
	require.Error(t, err)

	_, err = ResolveObjectLocation(space, "bucket", " ")
	require.Error(t, err)

	space.BasePrefix = ""
	_, err = ResolveObjectLocation(space, "bucket", "file")
	require.Error(t, err)
}
