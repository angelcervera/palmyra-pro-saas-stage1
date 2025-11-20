package requesttrace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	platformauth "github.com/zenGate-Global/palmyra-pro-saas/platform/go/auth"
)

func TestIntoContextAndFromContext(t *testing.T) {
	audit := AuditInfo{ActorKind: ActorKindUser, UserID: ptr("user-123"), RequestID: "req-abc"}

	ctx := IntoContext(context.Background(), audit)

	got, ok := FromContext(ctx)
	require.True(t, ok)
	require.Equal(t, audit, got)
}

func TestFromContextMissing(t *testing.T) {
	_, ok := FromContext(context.Background())
	require.False(t, ok)
}

func TestFromCredentials(t *testing.T) {
	creds := &platformauth.UserCredentials{Id: "user-456", TenantID: ptr("tenant-1")}

	audit, err := FromCredentials(creds, "req-xyz")
	require.NoError(t, err)
	require.Equal(t, ActorKindUser, audit.ActorKind)
	require.NotNil(t, audit.UserID)
	require.Equal(t, "user-456", *audit.UserID)
	require.Equal(t, "tenant-1", *audit.TenantID)
	require.Equal(t, "req-xyz", audit.RequestID)
}

func TestFromCredentialsMissingUser(t *testing.T) {
	_, err := FromCredentials(&platformauth.UserCredentials{}, "req-1")
	require.Error(t, err)
}

func TestAnonymous(t *testing.T) {
	audit := Anonymous("req-anon")
	require.Equal(t, ActorKindAnonymous, audit.ActorKind)
	require.Nil(t, audit.UserID)
	require.Equal(t, "req-anon", audit.RequestID)
}

func TestSystem(t *testing.T) {
	audit := System("req-sys")
	require.Equal(t, ActorKindSystem, audit.ActorKind)
	require.Nil(t, audit.UserID)
}

func ptr[T any](v T) *T { return &v }
