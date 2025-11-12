package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
)

// ValidateAuthenticationViaSwagger OpenAPI request validation against the embedded spec (with permissive auth func for public endpoints)
// Provide AuthenticationFunc to satisfy operations that declare security in OpenAPI.
func ValidateAuthenticationViaSwagger(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	// Enforce presence of Bearer token for endpoints that require bearerAuth.
	// For operations that allow anonymous (security: [{}] or no security), the validator will not require bearerAuth.
	if input != nil && input.SecuritySchemeName == "bearerAuth" {
		// TODO: extract role from `r.Context().Value(ctxUserCredentials)` injected in JWT middleware.
		r := input.RequestValidationInput.Request
		if r == nil {
			return fmt.Errorf("no request in validation input")
		}
		authz := r.Header.Get("Authorization")
		if authz == "" || !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			return fmt.Errorf("missing or invalid Authorization header")
		}

		// TODO: Validate if the user has one of the required roles.
	}
	return nil
}
