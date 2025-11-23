package devtoken

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildUnsignedFirebaseToken(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()

	token, err := BuildUnsignedFirebaseToken(Params{
		ProjectID:              "local-palmyra",
		Tenant:                 "tenant-dev",
		UserID:                 "admin-123",
		Email:                  "admin@example.com",
		Name:                   "Dev Admin",
		EmailVerified:          true,
		IsAdmin:                true,
		PalmyraRoles:           []string{"admin"},
		TenantRoles:            []string{"admin"},
		FirebaseSignInProvider: "password",
		ExpiresIn:              time.Hour,
	}, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	header, payload := splitToken(t, token)
	if got, want := header["alg"], "none"; got != want {
		t.Fatalf("header alg = %v, want %v", got, want)
	}

	if got, want := payload["iss"], "https://securetoken.google.com/local-palmyra"; got != want {
		t.Errorf("iss = %v, want %v", got, want)
	}
	if got, want := payload["aud"], "local-palmyra"; got != want {
		t.Errorf("aud = %v, want %v", got, want)
	}
	if got, want := payload["user_id"], "admin-123"; got != want {
		t.Errorf("user_id = %v, want %v", got, want)
	}
	if got, want := payload["sub"], "admin-123"; got != want {
		t.Errorf("sub = %v, want %v", got, want)
	}
	if got, want := payload["email"], "admin@example.com"; got != want {
		t.Errorf("email = %v, want %v", got, want)
	}
	if got, want := payload["email_verified"], true; got != want {
		t.Errorf("email_verified = %v, want %v", got, want)
	}
	if got, want := payload["isAdmin"], true; got != want {
		t.Errorf("isAdmin = %v, want %v", got, want)
	}

	firebaseClaim, ok := payload["firebase"].(map[string]interface{})
	if !ok {
		t.Fatalf("firebase claim missing or invalid type: %T", payload["firebase"])
	}
	if got, want := firebaseClaim["tenant"], "tenant-dev"; got != want {
		t.Errorf("firebase.tenant = %v, want %v", got, want)
	}
	if got, want := firebaseClaim["sign_in_provider"], "password"; got != want {
		t.Errorf("firebase.sign_in_provider = %v, want %v", got, want)
	}

	roles, ok := payload["palmyraRoles"].([]interface{})
	if !ok || len(roles) != 1 || roles[0] != "admin" {
		t.Errorf("palmyraRoles = %v, want [\"admin\"]", payload["palmyraRoles"])
	}
}

func splitToken(t *testing.T, token string) (map[string]interface{}, map[string]interface{}) {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		t.Fatalf("invalid token format: %q", token)
	}

	header := decodeSegment(t, parts[0])
	payload := decodeSegment(t, parts[1])
	return header, payload
}

func decodeSegment(t *testing.T, segment string) map[string]interface{} {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		t.Fatalf("decode segment: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal segment: %v", err)
	}
	return out
}
