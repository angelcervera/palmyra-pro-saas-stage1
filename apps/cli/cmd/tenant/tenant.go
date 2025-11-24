package tenantcmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/provisioning"
	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/repo"
	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
	tenantsapi "github.com/zenGate-Global/palmyra-pro-saas/generated/go/tenants"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// Command groups tenant-related helpers.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Tenant utilities (create/provision)",
	}

	cmd.AddCommand(createCommand())
	return cmd
}

func createCommand() *cobra.Command {
	var (
		databaseURL string
		envKey      string
		tenantSlug  string
		tenantName  string
		adminEmail  string
		adminName   string
	)

	c := &cobra.Command{
		Use:   "create",
		Short: "Create and bootstrap a tenant space (role, schema, grants, base tables, admin user)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			pool, err := persistence.NewPool(ctx, persistence.PoolConfig{ConnString: databaseURL})
			if err != nil {
				return fmt.Errorf("init pool: %w", err)
			}
			defer persistence.ClosePool(pool)

			adminSchema := tenant.BuildSchemaName(envKey, tenant.ToSnake("admin"))

			tenantStore, err := persistence.NewTenantStore(ctx, pool, adminSchema)
			if err != nil {
				return fmt.Errorf("init tenant store: %w", err)
			}
			tenantRepo := repo.NewPostgresRepository(tenantStore)

			if err := ensureTenantSlugIndex(ctx, pool, adminSchema); err != nil {
				return fmt.Errorf("ensure slug index: %w", err)
			}

			dbProv := provisioning.NewDBProvisioner(pool, adminSchema)
			authProv := readyAuthProvisioner{}
			storageProv := readyStorageProvisioner{}

			svc := service.New(
				tenantRepo,
				envKey,
				adminSchema,
				service.ProvisioningDeps{
					DB:      dbProv,
					Auth:    authProv,
					Storage: storageProv,
				},
			)

			createdBy := uuid.New()
			input := service.CreateInput{
				Slug:        tenantSlug,
				DisplayName: strPtrOrNil(tenantName),
				Status:      tenantsapi.Provisioning,
				CreatedBy:   createdBy,
			}

			t, err := svc.Create(ctx, input)
			if err != nil {
				if errors.Is(err, service.ErrConflictSlug) {
					existing, getErr := tenantRepo.FindBySlug(ctx, tenantSlug)
					if getErr != nil {
						return fmt.Errorf("tenant exists but could not fetch: %w", getErr)
					}
					t = existing
				} else {
					return fmt.Errorf("create tenant: %w", err)
				}
			}

			// Create DB artifacts first (idempotent), then refresh status.
			if _, err := dbProv.Ensure(ctx, service.DBProvisionRequest{
				TenantID:    t.ID,
				SchemaName:  t.SchemaName,
				RoleName:    t.RoleName,
				AdminSchema: adminSchema,
			}); err != nil {
				return fmt.Errorf("db ensure: %w", err)
			}
			if _, err := svc.ProvisionStatus(ctx, t.ID); err != nil {
				return fmt.Errorf("refresh provision status: %w", err)
			}

			tenantDB := persistence.NewTenantDB(persistence.TenantDBConfig{
				Pool:        pool,
				AdminSchema: adminSchema,
			})
			space := tenant.Space{
				TenantID:      t.ID,
				Slug:          t.Slug,
				ShortTenantID: t.ShortTenantID,
				SchemaName:    t.SchemaName,
				RoleName:      t.RoleName,
			}
			if err := seedTenantAdminUser(ctx, tenantDB, space, adminEmail, adminName); err != nil {
				return fmt.Errorf("seed tenant admin user: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Tenant bootstrap complete. Tenant: %s (%s)\n", t.Slug, t.ID)
			return nil
		},
	}

	c.Flags().StringVar(&databaseURL, "database-url", "", "PostgreSQL connection string")
	c.Flags().StringVar(&envKey, "env-key", "dev", "Environment key prefix (e.g. dev, stg, prod)")
	c.Flags().StringVar(&tenantSlug, "tenant-slug", "", "Slug for tenant to create/bootstrap")
	c.Flags().StringVar(&tenantName, "tenant-name", "", "Display name for tenant")
	c.Flags().StringVar(&adminEmail, "admin-email", "", "Tenant admin user email")
	c.Flags().StringVar(&adminName, "admin-full-name", "", "Tenant admin user full name")

	_ = c.MarkFlagRequired("database-url")
	_ = c.MarkFlagRequired("env-key")
	_ = c.MarkFlagRequired("tenant-slug")
	_ = c.MarkFlagRequired("admin-email")
	_ = c.MarkFlagRequired("admin-full-name")

	return c
}

// seedTenantAdminUser inserts or updates an admin user inside the tenant schema.
func seedTenantAdminUser(ctx context.Context, tenantDB *persistence.TenantDB, space tenant.Space, email, fullName string) error {
	email = strings.TrimSpace(email)
	fullName = strings.TrimSpace(fullName)
	if email == "" || fullName == "" {
		return fmt.Errorf("tenant admin email and full name are required to seed user")
	}

	return tenantDB.WithTenant(ctx, space, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
            INSERT INTO users (user_id, email, full_name)
            VALUES ($1, $2, $3)
            ON CONFLICT (email) DO UPDATE SET full_name = EXCLUDED.full_name
        `, uuid.New(), email, fullName)
		return err
	})
}

func strPtrOrNil(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// ensureTenantSlugIndex normalizes the unique index predicate to is_active = TRUE.
func ensureTenantSlugIndex(ctx context.Context, pool *pgxpool.Pool, adminSchema string) error {
	stmt := `
		DROP INDEX IF EXISTS tenants_slug_unique_active;
		DROP INDEX IF EXISTS ` + adminSchema + `.tenants_slug_unique_active;
		CREATE UNIQUE INDEX tenants_slug_unique_active
			ON ` + adminSchema + `.tenants (slug) WHERE is_active = TRUE;
	`
	_, err := pool.Exec(ctx, stmt)
	return err
}

// readyAuthProvisioner is a no-op auth provisioner that reports readiness.
type readyAuthProvisioner struct{}

func (readyAuthProvisioner) Ensure(context.Context, string) (service.AuthProvisionResult, error) {
	return service.AuthProvisionResult{Ready: true}, nil
}

func (readyAuthProvisioner) Check(context.Context, string) (service.AuthProvisionResult, error) {
	return service.AuthProvisionResult{Ready: true}, nil
}

// readyStorageProvisioner is a no-op storage provisioner that reports readiness.
type readyStorageProvisioner struct{}

func (readyStorageProvisioner) Ensure(context.Context, string) (service.StorageProvisionResult, error) {
	return service.StorageProvisionResult{Ready: true}, nil
}

func (readyStorageProvisioner) Check(context.Context, string) (service.StorageProvisionResult, error) {
	return service.StorageProvisionResult{Ready: true}, nil
}
