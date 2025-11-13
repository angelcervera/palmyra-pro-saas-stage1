package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractTenantID(t *testing.T) {
	tenant := "tenant-dev"
	firebaseTenant := "tenant-firebase"

	testCases := []struct {
		name   string
		claims map[string]interface{}
		want   *string
	}{
		{
			name:   "top level tenantId",
			claims: map[string]interface{}{"tenantId": tenant},
			want:   &tenant,
		},
		{
			name: "firebase tenant claim",
			claims: map[string]interface{}{
				"firebase": map[string]interface{}{"tenant": firebaseTenant},
			},
			want: &firebaseTenant,
		},
		{
			name:   "missing tenant",
			claims: map[string]interface{}{},
			want:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractTenantID(tc.claims)
			if tc.want == nil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, *tc.want, *got)
		})
	}
}

func TestDefaultCredentialExtractorWithTenantID(t *testing.T) {
	creds, err := DefaultCredentialExtractor(map[string]interface{}{
		"uid":            "user-123",
		"email":          "user@example.com",
		"tenantId":       "tenant-dev",
		"isAdmin":        true,
		"email_verified": true,
	})
	require.NoError(t, err)
	require.NotNil(t, creds.TenantID)
	require.Equal(t, "tenant-dev", *creds.TenantID)
}
