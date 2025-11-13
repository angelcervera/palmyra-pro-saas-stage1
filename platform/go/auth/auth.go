package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
)

type ctxKey string

const (
	ctxUserCredentials ctxKey = "PALMYRA_USER_CREDENTIALS"
)

type UserCredentials struct {
	Id            string
	Email         string
	EmailVerified bool
	Name          *string
	PictureURL    *string
	IsAdmin       bool
	TenantID      *string
}

func UserFromContext(ctx context.Context) (*UserCredentials, bool) {
	v := ctx.Value(ctxUserCredentials)
	if v == nil {
		return nil, false
	}
	u, ok := v.(*UserCredentials)
	return u, ok
}

// VerifyFunc validates the incoming JWT and returns its claims map.
type VerifyFunc func(ctx context.Context, token string) (map[string]interface{}, error)

// ExtractFunc converts a claims map into UserCredentials.
type ExtractFunc func(claims map[string]interface{}) (*UserCredentials, error)

// JWT parses the request and sets the context credentials using the provided verify/extract functions.
func JWT(verify VerifyFunc, extract ExtractFunc) func(http.Handler) http.Handler {
	if verify == nil {
		panic("auth.JWT: verify func must not be nil")
	}
	if extract == nil {
		extract = DefaultCredentialExtractor
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			token, found := ExtractJWTToken(r)
			if token == "" || !found {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := verify(r.Context(), token)
			if err != nil {
				w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="api", error="invalid_token", error_description="%s"`, err.Error()))
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			creds, err := extract(claims)
			if err != nil {
				w.Header().Set("WWW-Authenticate", `Bearer realm="api", error="invalid_token", error_description="invalid claims"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ctxUserCredentials, creds)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// DefaultCredentialExtractor converts standard claims into UserCredentials.
func DefaultCredentialExtractor(claims map[string]interface{}) (*UserCredentials, error) {
	if claims == nil {
		return nil, errors.New("missing claims")
	}

	creds := &UserCredentials{
		Id:            fallbackStringClaim(claims, []string{"uid", "user_id", "sub"}, "unknown-user"),
		Email:         extractStringClaim(claims, "email"),
		EmailVerified: extractBoolClaim(claims, "email_verified"),
		Name:          extractOptionalStringClaim(claims, "name"),
		PictureURL:    extractOptionalStringClaim(claims, "picture"),
		IsAdmin:       extractBoolClaim(claims, "isAdmin"),
		TenantID:      extractTenantID(claims),
	}

	return creds, nil
}

func extractBoolClaim(claims map[string]interface{}, key string) bool {
	if v, ok := claims[key]; ok {
		if boolVal, valid := v.(bool); valid {
			return boolVal
		}
	}
	return false
}

func extractStringClaim(claims map[string]interface{}, key string) string {
	if v, ok := claims[key]; ok {
		if strVal, valid := v.(string); valid {
			return strVal
		}
	}
	return ""
}

func extractOptionalStringClaim(claims map[string]interface{}, key string) *string {
	if v, ok := claims[key]; ok {
		if strVal, valid := v.(string); valid && strVal != "" {
			return &strVal
		}
	}
	return nil
}

func extractTenantID(claims map[string]interface{}) *string {
	firebaseClaim, ok := claims["firebase"].(map[string]interface{})
	if !ok {
		return nil
	}

	if tenant, ok := firebaseClaim["tenant"].(string); ok && tenant != "" {
		return &tenant
	}

	return nil
}

func parseUnsignedJWTClaims(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, errors.New("invalid token format")
	}

	payload := parts[1]
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	claims := make(map[string]interface{})
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}

	return claims, nil
}

func fallbackStringClaim(claims map[string]interface{}, keys []string, def string) string {
	for _, key := range keys {
		if v := extractStringClaim(claims, key); v != "" {
			return v
		}
	}
	return def
}

// FirebaseTokenVerifier returns a VerifyFunc that validates tokens via Firebase Auth.
func FirebaseTokenVerifier(fbAuth *auth.Client) VerifyFunc {
	return func(ctx context.Context, token string) (map[string]interface{}, error) {
		t, err := fbAuth.VerifyIDToken(ctx, token)
		if err != nil {
			return nil, err
		}

		claims := make(map[string]interface{}, len(t.Claims)+2)
		for k, v := range t.Claims {
			claims[k] = v
		}
		claims["uid"] = t.UID
		claims["sub"] = t.Subject
		if tenant := t.Firebase.Tenant; tenant != "" {
			if firebaseClaim, ok := claims["firebase"].(map[string]interface{}); ok {
				firebaseClaim["tenant"] = tenant
				claims["firebase"] = firebaseClaim
			} else {
				claims["firebase"] = map[string]interface{}{"tenant": tenant}
			}
		}

		return claims, nil
	}
}

// UnsignedTokenVerifier returns a VerifyFunc that decodes unsigned JWT payloads without validation.
func UnsignedTokenVerifier() VerifyFunc {
	return func(ctx context.Context, token string) (map[string]interface{}, error) {
		return parseUnsignedJWTClaims(token)
	}
}

// RequireRole is a helper to gate endpoints inside handlers if necessary:
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			creds, ok := UserFromContext(r.Context())
			if !ok || creds == nil {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			switch role {
			case "admin":
				if !creds.IsAdmin {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			default:
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
