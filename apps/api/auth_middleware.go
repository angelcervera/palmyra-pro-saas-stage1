package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	platformauth "github.com/zenGate-Global/palmyra-pro-saas/platform/go/auth"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/gcp"

	tenantsservice "github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
)

// buildAuthMiddleware constructs the JWT middleware with tenant claim enforcement and external->internal tenant mapping.
func buildAuthMiddleware(ctx context.Context, cfg config, tenantService *tenantsservice.Service, logger *zap.Logger) func(http.Handler) http.Handler {
	var verify platformauth.VerifyFunc
	switch cfg.AuthProvider {
	case "firebase":
		_, fbAuth, err := gcp.InitFirebaseAuth(ctx)
		if err != nil {
			logger.Fatal("init firebase auth", zap.Error(err))
		}
		verify = platformauth.FirebaseTokenVerifier(fbAuth)
	case "dev":
		logger.Warn("using dev auth middleware; do not use in production")
		verify = platformauth.UnsignedTokenVerifier()
	default:
		logger.Fatal("unsupported auth provider", zap.String("provider", cfg.AuthProvider))
	}

	authExtractor := func(claims map[string]interface{}) (*platformauth.UserCredentials, error) {
		creds, err := platformauth.DefaultCredentialExtractor(claims)
		if err != nil {
			return nil, err
		}
		if creds.TenantID == nil || *creds.TenantID == "" {
			return nil, errors.New("tenant claim required")
		}

		// Already an internal UUID? keep it.
		if tid, parseErr := uuid.Parse(*creds.TenantID); parseErr == nil {
			idStr := tid.String()
			creds.TenantID = &idStr
			return creds, nil
		}

		// Otherwise treat as external <envKey>-<slug>.
		space, resolveErr := tenantService.ResolveTenantSpaceByExternal(context.Background(), *creds.TenantID)
		if resolveErr != nil {
			return nil, resolveErr
		}
		idStr := space.TenantID.String()
		creds.TenantID = &idStr
		return creds, nil
	}

	return platformauth.JWT(verify, authExtractor)
}
