package devtoken

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Params captures the Firebase-compatible claims required to mint an unsigned JWT
// for local and CI environments. All fields should be provided by the caller; no
// environment variables are read so the builder stays deterministic for tooling.
type Params struct {
	ProjectID              string        // Firebase project id; used for aud and iss
	Tenant                 string        // firebase.tenant claim (required)
	UserID                 string        // user_id/sub/uid (required)
	Email                  string        // email claim (required)
	Name                   string        // display name (optional but recommended)
	EmailVerified          bool          // email_verified claim
	IsAdmin                bool          // isAdmin custom claim for backend role checks
	PalmyraRoles           []string      // optional custom roles array
	TenantRoles            []string      // optional tenant-scoped roles array
	FirebaseSignInProvider string        // firebase.sign_in_provider; default "password"
	ExpiresIn              time.Duration // relative expiry; default 1h if zero
	Audience               string        // optional override; defaults to ProjectID
	Issuer                 string        // optional override; defaults to https://securetoken.google.com/<projectId>
}

// BuildUnsignedFirebaseToken returns a JWT string with alg "none" and no signature.
// The payload mirrors Firebase ID token shape so it can flow through the existing
// auth middleware when AUTH_PROVIDER=dev.
func BuildUnsignedFirebaseToken(p Params, now time.Time) (string, error) {
	if strings.TrimSpace(p.ProjectID) == "" {
		return "", errors.New("projectID is required")
	}
	if strings.TrimSpace(p.Tenant) == "" {
		return "", errors.New("tenant is required")
	}
	if strings.TrimSpace(p.UserID) == "" {
		return "", errors.New("userID is required")
	}
	if strings.TrimSpace(p.Email) == "" {
		return "", errors.New("email is required")
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}

	expiresIn := p.ExpiresIn
	if expiresIn == 0 {
		expiresIn = time.Hour
	}

	issuer := p.Issuer
	if strings.TrimSpace(issuer) == "" {
		issuer = fmt.Sprintf("https://securetoken.google.com/%s", p.ProjectID)
	}

	audience := p.Audience
	if strings.TrimSpace(audience) == "" {
		audience = p.ProjectID
	}

	signInProvider := p.FirebaseSignInProvider
	if strings.TrimSpace(signInProvider) == "" {
		signInProvider = "password"
	}

	firebaseIdentities := map[string]interface{}{
		"email": []string{p.Email},
	}

	payload := map[string]interface{}{
		"iss":            issuer,
		"aud":            audience,
		"auth_time":      now.Unix(),
		"user_id":        p.UserID,
		"sub":            p.UserID,
		"iat":            now.Unix(),
		"exp":            now.Add(expiresIn).Unix(),
		"email":          p.Email,
		"email_verified": p.EmailVerified,
		"name":           p.Name,
		"isAdmin":        p.IsAdmin,
		"firebase": map[string]interface{}{
			"identities":       firebaseIdentities,
			"sign_in_provider": signInProvider,
			"tenant":           p.Tenant,
		},
	}

	if len(p.PalmyraRoles) > 0 {
		payload["palmyraRoles"] = p.PalmyraRoles
	}
	if len(p.TenantRoles) > 0 {
		payload["tenantRoles"] = p.TenantRoles
	}

	header := map[string]interface{}{
		"alg": "none",
		"typ": "JWT",
	}

	headerSegment, err := encodeSegment(header)
	if err != nil {
		return "", err
	}

	payloadSegment, err := encodeSegment(payload)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s", headerSegment, payloadSegment), nil
}

func encodeSegment(v interface{}) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
