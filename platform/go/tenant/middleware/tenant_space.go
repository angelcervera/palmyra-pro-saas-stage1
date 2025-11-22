package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
	problems "github.com/zenGate-Global/palmyra-pro-saas/generated/go/common/problemdetails"
	platformauth "github.com/zenGate-Global/palmyra-pro-saas/platform/go/auth"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// Resolver defines the minimal lookup capability required to populate a Tenant Space.
// Implemented by the tenant registry repository/service.
type Resolver interface {
	ResolveTenantSpace(ctx context.Context, tenantID uuid.UUID) (tenant.Space, error)
	ResolveTenantSpaceByExternal(ctx context.Context, external string) (tenant.Space, error)
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
				writeProblem(w, http.StatusUnauthorized, "Unauthorized", "tenant required", problemTypeAuth)
				return
			}

			var (
				space tenant.Space
				err   error
			)

			if tid, parseErr := uuid.Parse(*creds.TenantID); parseErr == nil {
				if cached := cacheGet(cache, tid); cached != nil {
					ctx := tenant.WithSpace(r.Context(), *cached)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				space, err = resolver.ResolveTenantSpace(r.Context(), tid)
			} else {
				space, err = resolver.ResolveTenantSpaceByExternal(r.Context(), *creds.TenantID)
			}

			if err != nil {
				switch {
				case errors.Is(err, service.ErrEnvMismatch):
					writeProblem(w, http.StatusForbidden, "Forbidden", "tenant env mismatch", problemTypeAuth)
				case errors.Is(err, service.ErrDisabled):
					writeProblem(w, http.StatusForbidden, "Forbidden", "tenant disabled", problemTypeAuth)
				case errors.Is(err, service.ErrNotFound):
					writeProblem(w, http.StatusForbidden, "Forbidden", "tenant unknown", problemTypeAuth)
				default:
					writeProblem(w, http.StatusUnauthorized, "Unauthorized", "invalid tenant", problemTypeAuth)
				}
				return
			}
			if cached := cacheGet(cache, space.TenantID); cached != nil {
				ctx := tenant.WithSpace(r.Context(), *cached)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			prefix := cfg.EnvKey + "/"
			if len(space.BasePrefix) < len(prefix) || !strings.HasPrefix(space.BasePrefix, prefix) {
				writeProblem(w, http.StatusForbidden, "Forbidden", "tenant env mismatch", problemTypeAuth)
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
	mu    sync.RWMutex
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
	c.mu.RLock()
	item, ok := c.items[id]
	c.mu.RUnlock()
	if !ok || time.Now().After(item.expiresAt) {
		if ok {
			c.mu.Lock()
			delete(c.items, id)
			c.mu.Unlock()
		}
		return nil
	}
	return &item.space
}

func cachePut(c *tenantCache, space tenant.Space) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.items[space.TenantID] = cacheItem{space: space, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

const problemTypeAuth = "https://palmyra.pro/problems/auth"

func writeProblem(w http.ResponseWriter, status int, title, detail, problemType string) {
	p := problems.ProblemDetails{
		Title:  title,
		Status: status,
		Type:   &problemType,
		Detail: &detail,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}
