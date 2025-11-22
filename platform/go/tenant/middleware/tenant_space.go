package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	platformauth "github.com/zenGate-Global/palmyra-pro-saas/platform/go/auth"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// Resolver defines the minimal lookup capability required to populate a Tenant Space.
// Implemented by the tenant registry repository/service.
type Resolver interface {
	ResolveTenantSpace(tenantID uuid.UUID) (tenant.Space, error)
}

// Config controls middleware behavior.
type Config struct {
	EnvKey string
	// Optional small in-memory TTL cache to avoid DB hits; zero disables caching.
	CacheTTL time.Duration
}

// WithTenantSpace resolves tenant from JWT claims and attaches tenant.Space to context.
// It enforces that the tenant claim is present and that the resolved space matches the current envKey.
func WithTenantSpace(resolver Resolver, cfg Config) func(http.Handler) http.Handler {
	if resolver == nil {
		panic("tenant middleware: resolver is required")
	}
	if cfg.EnvKey == "" {
		panic("tenant middleware: envKey is required")
	}

	var cache *tenantCache
	if cfg.CacheTTL > 0 {
		cache = newTenantCache(cfg.CacheTTL)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			creds, ok := platformauth.UserFromContext(r.Context())
			if !ok || creds == nil || creds.TenantID == nil || *creds.TenantID == "" {
				http.Error(w, "tenant required", http.StatusUnauthorized)
				return
			}

			// Parse tenant claim (expected to be tenant UUID string).
			tid, err := uuid.Parse(*creds.TenantID)
			if err != nil {
				http.Error(w, "invalid tenant id", http.StatusUnauthorized)
				return
			}

			if cached := cacheGet(cache, tid); cached != nil {
				ctx := tenant.WithSpace(r.Context(), *cached)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			space, err := resolver.ResolveTenantSpace(tid)
			if err != nil {
				http.Error(w, "tenant not found", http.StatusUnauthorized)
				return
			}

			// EnvKey alignment check: basePrefix must start with envKey + "/".
			prefix := cfg.EnvKey + "/"
			if len(space.BasePrefix) < len(prefix) || space.BasePrefix[:len(prefix)] != prefix {
				http.Error(w, "tenant env mismatch", http.StatusForbidden)
				return
			}

			cachePut(cache, space)

			ctx := tenant.WithSpace(r.Context(), space)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type tenantCache struct {
	ttl   time.Duration
	items map[uuid.UUID]cacheItem
}

type cacheItem struct {
	space     tenant.Space
	expiresAt time.Time
}

func newTenantCache(ttl time.Duration) *tenantCache {
	return &tenantCache{ttl: ttl, items: make(map[uuid.UUID]cacheItem)}
}

func cacheGet(c *tenantCache, id uuid.UUID) *tenant.Space {
	if c == nil {
		return nil
	}
	item, ok := c.items[id]
	if !ok || time.Now().After(item.expiresAt) {
		return nil
	}
	return &item.space
}

func cachePut(c *tenantCache, space tenant.Space) {
	if c == nil {
		return
	}
	c.items[space.TenantID] = cacheItem{space: space, expiresAt: time.Now().Add(c.ttl)}
}
