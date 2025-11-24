package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/caarlos0/env/v11"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
	"go.uber.org/zap"

	entitieshandler "github.com/zenGate-Global/palmyra-pro-saas/domains/entities/be/handler"
	entitiesrepo "github.com/zenGate-Global/palmyra-pro-saas/domains/entities/be/repo"
	entitiesservice "github.com/zenGate-Global/palmyra-pro-saas/domains/entities/be/service"
	schemacategorieshandler "github.com/zenGate-Global/palmyra-pro-saas/domains/schema-categories/be/handler"
	schemacategoriesrepo "github.com/zenGate-Global/palmyra-pro-saas/domains/schema-categories/be/repo"
	schemacategoriesservice "github.com/zenGate-Global/palmyra-pro-saas/domains/schema-categories/be/service"
	schemarepositoryhandler "github.com/zenGate-Global/palmyra-pro-saas/domains/schema-repository/be/handler"
	schemarepositoryrepo "github.com/zenGate-Global/palmyra-pro-saas/domains/schema-repository/be/repo"
	schemarepositoryservice "github.com/zenGate-Global/palmyra-pro-saas/domains/schema-repository/be/service"
	tenantshandler "github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/handler"
	tenantsprov "github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/provisioning"
	tenantsrepo "github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/repo"
	tenantsservice "github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
	usershandler "github.com/zenGate-Global/palmyra-pro-saas/domains/users/be/handler"
	usersrepo "github.com/zenGate-Global/palmyra-pro-saas/domains/users/be/repo"
	usersservice "github.com/zenGate-Global/palmyra-pro-saas/domains/users/be/service"
	authapi "github.com/zenGate-Global/palmyra-pro-saas/generated/go/auth"
	entitiesapi "github.com/zenGate-Global/palmyra-pro-saas/generated/go/entities"
	schemacategories "github.com/zenGate-Global/palmyra-pro-saas/generated/go/schema-categories"
	schemarepository "github.com/zenGate-Global/palmyra-pro-saas/generated/go/schema-repository"
	tenantsapi "github.com/zenGate-Global/palmyra-pro-saas/generated/go/tenants"
	users "github.com/zenGate-Global/palmyra-pro-saas/generated/go/users"
	platformauth "github.com/zenGate-Global/palmyra-pro-saas/platform/go/auth"
	platformlogging "github.com/zenGate-Global/palmyra-pro-saas/platform/go/logging"
	platformmiddleware "github.com/zenGate-Global/palmyra-pro-saas/platform/go/middleware"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
	tenantmiddleware "github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant/middleware"
)

var swaggerLoaders = map[string]func() (*openapi3.T, error){
	"contracts/entities.yaml":          entitiesapi.GetSwagger,
	"contracts/auth.yaml":              authapi.GetSwagger,
	"contracts/schema-categories.yaml": schemacategories.GetSwagger,
	"contracts/schema-repository.yaml": schemarepository.GetSwagger,
	"contracts/users.yaml":             users.GetSwagger,
	"contracts/tenants.yaml":           tenantsapi.GetSwagger,
}

type config struct {
	Port            string        `env:"PORT" envDefault:"3000"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	RequestTimeout  time.Duration `env:"REQUEST_TIMEOUT" envDefault:"15s"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
	DatabaseURL     string        `env:"DATABASE_URL,required"`
	AuthProvider    string        `env:"AUTH_PROVIDER" envDefault:"firebase"`
	EnvKey          string        `env:"ENV_KEY,required"`
	AdminTenantSlug string        `env:"ADMIN_TENANT_SLUG" envDefault:"admin"`
	StorageBackend  string        `env:"STORAGE_BACKEND" envDefault:"gcs"`               // gcs | local
	StorageBucket   string        `env:"STORAGE_BUCKET"`                                 // required when STORAGE_BACKEND=gcs
	StorageLocalDir string        `env:"STORAGE_LOCAL_DIR" envDefault:"./.data/storage"` // used when STORAGE_BACKEND=local
}

func main() {
	ctx := context.Background()

	var cfg config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("load config: %v", err)
	}

	adminSchema := tenant.BuildSchemaName(cfg.EnvKey, tenant.ToSnake(cfg.AdminTenantSlug))

	logger, err := platformlogging.NewLogger(platformlogging.Config{
		Component: "api-server",
		Level:     cfg.LogLevel,
	})
	if err != nil {
		log.Fatalf("init zap logger: %v", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	pool, err := persistence.NewPool(ctx, persistence.PoolConfig{ConnString: cfg.DatabaseURL})
	if err != nil {
		logger.Fatal("init postgres pool", zap.Error(err))
	}
	defer persistence.ClosePool(pool)

	categoryStore, err := persistence.NewSchemaCategoryStore(ctx, pool)
	if err != nil {
		logger.Fatal("init schema category store", zap.Error(err))
	}

	categoryRepo := schemacategoriesrepo.NewPostgresRepository(categoryStore)
	categoryService := schemacategoriesservice.New(categoryRepo)
	categoryHTTPHandler := schemacategorieshandler.New(categoryService, logger)

	schemaStore, err := persistence.NewSchemaRepositoryStore(ctx, pool)
	if err != nil {
		logger.Fatal("init schema repository store", zap.Error(err))
	}

	schemaRepo := schemarepositoryrepo.NewPostgresRepository(schemaStore)
	schemaService := schemarepositoryservice.New(schemaRepo)
	schemaHTTPHandler := schemarepositoryhandler.New(schemaService, logger)

	tenantStore, err := persistence.NewTenantStore(ctx, pool, adminSchema)
	if err != nil {
		logger.Fatal("init tenant store", zap.Error(err))
	}

	tenantRepo := tenantsrepo.NewPostgresRepository(tenantStore)
	dbProv := tenantsprov.NewDBProvisioner(pool, adminSchema)
	authProv := tenantsprov.NewAuthProvisioner()
	var storageProv tenantsservice.StorageProvisioner
	switch cfg.StorageBackend {
	case "gcs":
		if cfg.StorageBucket == "" {
			logger.Fatal("storage bucket required when STORAGE_BACKEND=gcs")
		}
		gcsClient, err := storage.NewClient(ctx)
		if err != nil {
			logger.Fatal("init gcs client", zap.Error(err))
		}
		defer gcsClient.Close()
		storageProv = tenantsprov.NewGCSStorageProvisioner(gcsClient, cfg.StorageBucket)
	case "local":
		if strings.TrimSpace(cfg.StorageLocalDir) == "" {
			logger.Fatal("storage local dir required when STORAGE_BACKEND=local")
		}
		storageProv = tenantsprov.NewLocalStorageProvisioner(cfg.StorageLocalDir)
	default:
		logger.Fatal("invalid STORAGE_BACKEND (use gcs or local)", zap.String("backend", cfg.StorageBackend))
	}
	tenantService := tenantsservice.New(
		tenantRepo,
		cfg.EnvKey,
		tenantsservice.ProvisioningDeps{
			DB:      dbProv,
			Auth:    authProv,
			Storage: storageProv,
		},
	)
	tenantHTTPHandler := tenantshandler.New(tenantService, logger)

	authMiddleware := buildAuthMiddleware(ctx, cfg, tenantService, logger)

	tenantDB := persistence.NewTenantDB(persistence.TenantDBConfig{
		Pool:        pool,
		AdminSchema: adminSchema,
	})

	schemaValidator := persistence.NewSchemaValidator()

	userStore, err := persistence.NewUserStore(ctx, tenantDB)
	if err != nil {
		logger.Fatal("init user store", zap.Error(err))
	}

	userRepo := usersrepo.NewPostgresRepository(userStore)
	userService := usersservice.New(userRepo)
	userHTTPHandler := usershandler.New(userService, logger)

	entitiesRepo := entitiesrepo.New(tenantDB, schemaStore, schemaValidator)
	entitiesService := entitiesservice.New(entitiesRepo)
	entitiesHTTPHandler := entitieshandler.New(entitiesService, logger)

	rootRouter := chi.NewRouter()

	rootRouter.Use(
		chimw.RequestID,
		chimw.RealIP,
		chimw.Recoverer,
		chimw.Timeout(cfg.RequestTimeout),
		platformmiddleware.DefaultCORS(),
	)

	rootRouter.Use(platformlogging.RequestLogger(logger))

	rootRouter.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	rootRouter.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// ---- Swagger UI + OpenAPI JSON (public) ----
	registerDocsRoutes(rootRouter, logger)

	apiRouter := chi.NewRouter()
	apiRouter.Use(authMiddleware)
	apiRouter.Use(platformmiddleware.RequestTrace)
	apiRouter.Use(tenantmiddleware.WithTenantSpace(tenantService, tenantmiddleware.Config{
		EnvKey:   cfg.EnvKey,
		CacheTTL: time.Minute,
	}))

	schemaCategoriesValidator := mustNewSpecValidator(logger, "contracts/schema-categories.yaml")
	apiRouter.Group(func(r chi.Router) {
		r.Use(schemaCategoriesValidator)
		_ = schemacategories.HandlerWithOptions(
			schemacategories.NewStrictHandler(categoryHTTPHandler, nil),
			schemacategories.ChiServerOptions{BaseRouter: r},
		)
	})

	schemaRepositoryValidator := mustNewSpecValidator(logger, "contracts/schema-repository.yaml")
	apiRouter.Group(func(r chi.Router) {
		r.Use(schemaRepositoryValidator)
		_ = schemarepository.HandlerWithOptions(
			schemarepository.NewStrictHandler(schemaHTTPHandler, nil),
			schemarepository.ChiServerOptions{BaseRouter: r},
		)
	})

	entitiesValidator := mustNewSpecValidator(logger, "contracts/entities.yaml")
	apiRouter.Group(func(r chi.Router) {
		r.Use(entitiesValidator)
		_ = entitiesapi.HandlerWithOptions(
			entitiesapi.NewStrictHandler(entitiesHTTPHandler, nil),
			entitiesapi.ChiServerOptions{BaseRouter: r},
		)
	})

	usersValidator := mustNewSpecValidator(logger, "contracts/users.yaml")
	apiRouter.Group(func(r chi.Router) {
		r.Use(usersValidator)
		_ = users.HandlerWithOptions(
			users.NewStrictHandler(userHTTPHandler, nil),
			users.ChiServerOptions{BaseRouter: r},
		)
	})

	tenantsValidator := mustNewSpecValidator(logger, "contracts/tenants.yaml")
	apiRouter.Group(func(r chi.Router) {
		r.Use(platformauth.RequireRole("admin"))
		r.Use(tenantsValidator)
		_ = tenantsapi.HandlerWithOptions(
			tenantsapi.NewStrictHandler(tenantHTTPHandler, nil),
			tenantsapi.ChiServerOptions{BaseRouter: r},
		)
	})

	rootRouter.Mount("/api/v1", apiRouter)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      rootRouter,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  2 * time.Minute,
	}

	go func() {
		logger.Info("starting api server", zap.String("port", cfg.Port))
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server listen failed", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}
}

// mustNewSpecValidator loads the OpenAPI document and builds oapi-codegen validator middleware.
// This can be reused by each domain group to guarantee contract compliance per docs/api-server.md
func mustNewSpecValidator(logger *zap.Logger, path string) func(http.Handler) http.Handler {
	if loaderFn, ok := swaggerLoaders[path]; ok {
		spec, err := loaderFn()
		if err != nil {
			logger.Fatal("load generated swagger", zap.String("path", path), zap.Error(err))
		}
		logSecuritySchemes(logger, path, spec)
		return oapimiddleware.OapiRequestValidatorWithOptions(spec, &oapimiddleware.Options{
			Options: openapi3filter.Options{
				AuthenticationFunc: platformmiddleware.ValidateAuthenticationViaSwagger,
			},
		})
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	absPath, err := filepath.Abs(path)
	if err != nil {
		logger.Fatal("resolve spec path", zap.String("path", path), zap.Error(err))
	}

	baseDir := filepath.Dir(absPath)
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, ref *url.URL) ([]byte, error) {
		if ref == nil {
			return nil, errors.New("nil reference URI")
		}

		if ref.IsAbs() {
			switch ref.Scheme {
			case "file":
				data, err := os.ReadFile(ref.Path)
				if err != nil {
					return nil, fmt.Errorf("read reference %q: %w", ref.Path, err)
				}
				return data, nil
			default:
				return nil, fmt.Errorf("unsupported reference scheme %q", ref.String())
			}
		}

		refPath := filepath.Clean(ref.Path)
		if refPath == "" {
			return nil, fmt.Errorf("empty reference path for %q", ref.String())
		}

		candidate := filepath.Join(baseDir, refPath)
		data, err := os.ReadFile(candidate)
		if err != nil {
			return nil, fmt.Errorf("read reference %q: %w", candidate, err)
		}
		return data, nil
	}

	spec, err := loader.LoadFromFile(absPath)
	if err != nil {
		logger.Fatal("load openapi spec", zap.String("path", path), zap.Error(err))
	}

	logSecuritySchemes(logger, path, spec)

	return oapimiddleware.OapiRequestValidatorWithOptions(spec, &oapimiddleware.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc: platformmiddleware.ValidateAuthenticationViaSwagger,
		},
	})
}

// mustLoadSpec loads and returns the OpenAPI document for docs serving.
func mustLoadSpec(logger *zap.Logger, path string) *openapi3.T {
	if loaderFn, ok := swaggerLoaders[path]; ok {
		spec, err := loaderFn()
		if err != nil {
			logger.Fatal("load generated swagger", zap.String("path", path), zap.Error(err))
		}
		logSecuritySchemes(logger, path, spec)
		return spec
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	absPath, err := filepath.Abs(path)
	if err != nil {
		logger.Fatal("resolve spec path", zap.String("path", path), zap.Error(err))
	}

	baseDir := filepath.Dir(absPath)
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, ref *url.URL) ([]byte, error) {
		if ref == nil {
			return nil, errors.New("nil reference URI")
		}
		if ref.IsAbs() {
			switch ref.Scheme {
			case "file":
				data, err := os.ReadFile(ref.Path)
				if err != nil {
					return nil, fmt.Errorf("read reference %q: %w", ref.Path, err)
				}
				return data, nil
			default:
				return nil, fmt.Errorf("unsupported reference scheme %q", ref.String())
			}
		}
		refPath := filepath.Clean(ref.Path)
		if refPath == "" {
			return nil, fmt.Errorf("empty reference path for %q", ref.String())
		}
		candidate := filepath.Join(baseDir, refPath)
		data, err := os.ReadFile(candidate)
		if err != nil {
			return nil, fmt.Errorf("read reference %q: %w", candidate, err)
		}
		return data, nil
	}

	spec, err := loader.LoadFromFile(absPath)
	if err != nil {
		logger.Fatal("load openapi spec", zap.String("path", path), zap.Error(err))
	}
	logSecuritySchemes(logger, path, spec)
	return spec
}

func logSecuritySchemes(logger *zap.Logger, path string, spec *openapi3.T) {
	if spec.Components.SecuritySchemes == nil {
		spec.Components.SecuritySchemes = openapi3.SecuritySchemes{}
	}

	if _, ok := spec.Components.SecuritySchemes["bearerAuth"]; !ok {
		spec.Components.SecuritySchemes["bearerAuth"] = &openapi3.SecuritySchemeRef{
			Value: &openapi3.SecurityScheme{
				Type:   "http",
				Scheme: "bearer",
			},
		}
		logger.Warn("injecting default bearerAuth security scheme", zap.String("path", path))
	}

	names := make([]string, 0, len(spec.Components.SecuritySchemes))
	for name := range spec.Components.SecuritySchemes {
		names = append(names, name)
	}
	logger.Info("loaded security schemes", zap.String("path", path), zap.Strings("names", names))
}
