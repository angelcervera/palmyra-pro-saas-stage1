package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/provisioning"
	tenantsrepo "github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/repo"
	tenantsservice "github.com/zenGate-Global/palmyra-pro-saas/domains/tenants/be/service"
	usersrepo "github.com/zenGate-Global/palmyra-pro-saas/domains/users/be/repo"
	usersservice "github.com/zenGate-Global/palmyra-pro-saas/domains/users/be/service"
	tenantsapi "github.com/zenGate-Global/palmyra-pro-saas/generated/go/tenants"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/requesttrace"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// Command groups bootstrap helpers (platform init, future seed steps).
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap platform resources (admin tenant, admin user)",
		Long:  "Bootstrap platform resources such as the admin tenant space and initial admin user.",
	}

	cmd.AddCommand(platformCommand())
	return cmd
}

func platformCommand() *cobra.Command {
	var (
		databaseURL   string
		envKey        string
		adminSchema   string
		tenantSlug    string
		tenantName    string
		adminEmail    string
		adminFullName string
		createdBy     string
	)

	c := &cobra.Command{
		Use:   "platform",
		Short: "Bootstrap admin tenant space and admin user",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			pool, err := persistence.NewPool(ctx, persistence.PoolConfig{ConnString: databaseURL})
			if err != nil {
				return fmt.Errorf("init pool: %w", err)
			}
			defer persistence.ClosePool(pool)

			tenantStore, err := persistence.NewTenantStore(ctx, pool, adminSchema)
			if err != nil {
				return fmt.Errorf("init tenant store: %w", err)
			}

			tenantRepo := tenantsrepo.NewPostgresRepository(tenantStore)

			dbProv := provisioning.NewDBProvisioner(pool)
			authProv := &noopAuthProvisioner{}
			storageProv := &noopStorageProvisioner{}

			tenantProvisionSvc := tenantsservice.NewWithProvisioningAndAdmin(tenantRepo, envKey, adminSchema, tenantsservice.ProvisioningDeps{
				DB:      dbProv,
				Auth:    authProv,
				Storage: storageProv,
			})

			createdByID := uuid.New()
			if createdBy != "" {
				parsed, parseErr := uuid.Parse(createdBy)
				if parseErr != nil {
					return fmt.Errorf("invalid created-by uuid: %w", parseErr)
				}
				createdByID = parsed
			}

			tenantRec, err := tenantRepo.FindBySlug(ctx, tenantSlug)
			if err != nil {
				if errors.Is(err, tenantsservice.ErrNotFound) {
					tenantRec, err = tenantProvisionSvc.Create(ctx, tenantsservice.CreateInput{
						Slug:        tenantSlug,
						DisplayName: strPtrOrNil(tenantName),
						Status:      tenantsapi.Pending,
						CreatedBy:   createdByID,
					})
					if err != nil {
						return fmt.Errorf("create tenant: %w", err)
					}
				} else {
					return fmt.Errorf("get tenant by slug: %w", err)
				}
			}

			// Per-component check-or-create to avoid duplicate work.
			externalTenant := fmt.Sprintf("%s-%s", envKey, tenantRec.Slug)

			dbReady := false
			if res, err := dbProv.Check(ctx, tenantsservice.DBProvisionRequest{TenantID: tenantRec.ID, SchemaName: tenantRec.SchemaName, RoleName: tenantRec.RoleName, AdminSchema: adminSchema}); err == nil {
				dbReady = res.Ready
			}
			if !dbReady {
				if res, err := dbProv.Ensure(ctx, tenantsservice.DBProvisionRequest{TenantID: tenantRec.ID, SchemaName: tenantRec.SchemaName, RoleName: tenantRec.RoleName, AdminSchema: adminSchema}); err == nil {
					dbReady = res.Ready
				} else {
					return fmt.Errorf("provision db: %w", err)
				}
			}

			authReady := false
			if res, err := authProv.Check(ctx, externalTenant); err == nil {
				authReady = res.Ready
			}
			if !authReady {
				if res, err := authProv.Ensure(ctx, externalTenant); err == nil {
					authReady = res.Ready
				} else {
					return fmt.Errorf("provision auth: %w", err)
				}
			}

			storageReady := false
			if res, err := storageProv.Check(ctx, tenantRec.BasePrefix); err == nil {
				storageReady = res.Ready
			}
			if !storageReady {
				if res, err := storageProv.Ensure(ctx, tenantRec.BasePrefix); err == nil {
					storageReady = res.Ready
				} else {
					return fmt.Errorf("provision storage: %w", err)
				}
			}

			// Persist provisioning status (and activate if fully ready).
			prov, err := tenantProvisionSvc.ProvisionStatus(ctx, tenantRec.ID)
			if err != nil {
				return fmt.Errorf("update provisioning status: %w", err)
			}
			tenantRec.Provisioning = prov

			space, err := tenantProvisionSvc.ResolveTenantSpace(ctx, tenantRec.ID)
			if err != nil {
				return fmt.Errorf("resolve tenant space: %w", err)
			}

			// Create admin user inside tenant space.
			tenantDB := persistence.NewTenantDB(persistence.TenantDBConfig{Pool: pool, AdminSchema: adminSchema})
			userStore, err := persistence.NewUserStore(ctx, tenantDB)
			if err != nil {
				return fmt.Errorf("init user store: %w", err)
			}
			userRepo := usersrepo.NewPostgresRepository(userStore)
			userSvc := usersservice.New(userRepo)

			audit := requesttrace.AuditInfo{}
			ctxTenant := tenant.WithSpace(ctx, space)
			user, err := ensureAdminUser(ctxTenant, userSvc, audit, adminEmail, adminFullName)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Bootstrap complete. Tenant: %s (%s) | Admin user: %s (%s)\n", tenantRec.Slug, tenantRec.ID, user.Email, user.ID)
			if !tenantRec.Provisioning.DBReady || !tenantRec.Provisioning.AuthReady || !tenantRec.Provisioning.StorageReady {
				fmt.Fprintf(cmd.OutOrStdout(), "Note: provisioning status DB=%t Auth=%t Storage=%t (auth/storage are no-op in this dev tool).\n", tenantRec.Provisioning.DBReady, tenantRec.Provisioning.AuthReady, tenantRec.Provisioning.StorageReady)
			}
			return nil
		},
	}

	c.Flags().StringVar(&databaseURL, "database-url", "", "PostgreSQL connection string")
	c.Flags().StringVar(&envKey, "env-key", "dev", "Environment key prefix (e.g. dev, stg, prod)")
	c.Flags().StringVar(&adminSchema, "admin-schema", "admin", "Admin schema name for tenant registry")
	c.Flags().StringVar(&tenantSlug, "tenant-slug", "admin", "Slug for admin tenant")
	c.Flags().StringVar(&tenantName, "tenant-name", "Admin", "Display name for admin tenant")
	c.Flags().StringVar(&adminEmail, "admin-email", "", "Initial admin user email")
	c.Flags().StringVar(&adminFullName, "admin-full-name", "", "Initial admin user full name")
	c.Flags().StringVar(&createdBy, "created-by", "", "UUID for createdBy (optional; defaults to random)")

	_ = c.MarkFlagRequired("database-url")
	_ = c.MarkFlagRequired("admin-email")
	_ = c.MarkFlagRequired("admin-full-name")

	return c
}

// noopAuthProvisioner marks auth ready without external calls (dev/bootstrap only).
type noopAuthProvisioner struct{}

func (n *noopAuthProvisioner) Ensure(ctx context.Context, externalTenant string) (tenantsservice.AuthProvisionResult, error) {
	return tenantsservice.AuthProvisionResult{Ready: true}, nil
}

func (n *noopAuthProvisioner) Check(ctx context.Context, externalTenant string) (tenantsservice.AuthProvisionResult, error) {
	return tenantsservice.AuthProvisionResult{Ready: true}, nil
}

// noopStorageProvisioner marks storage ready without external calls (dev/bootstrap only).
type noopStorageProvisioner struct{}

func (n *noopStorageProvisioner) Ensure(ctx context.Context, prefix string) (tenantsservice.StorageProvisionResult, error) {
	return tenantsservice.StorageProvisionResult{Ready: true}, nil
}

func (n *noopStorageProvisioner) Check(ctx context.Context, prefix string) (tenantsservice.StorageProvisionResult, error) {
	return tenantsservice.StorageProvisionResult{Ready: true}, nil
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ensureAdminUser performs a check-or-create for the admin user inside the tenant space.
func ensureAdminUser(ctx context.Context, userSvc usersservice.Service, audit requesttrace.AuditInfo, email, fullName string) (usersservice.User, error) {
	email = strings.TrimSpace(email)
	fullName = strings.TrimSpace(fullName)
	if email == "" || fullName == "" {
		return usersservice.User{}, fmt.Errorf("admin email and full name are required")
	}

	filterEmail := email
	res, err := userSvc.List(ctx, audit, usersservice.ListOptions{Email: &filterEmail, Page: 1, PageSize: 1})
	if err != nil {
		return usersservice.User{}, fmt.Errorf("lookup admin user: %w", err)
	}
	if len(res.Users) > 0 {
		return res.Users[0], nil
	}

	user, err := userSvc.Create(ctx, audit, usersservice.CreateInput{Email: email, FullName: fullName})
	if err != nil {
		return usersservice.User{}, fmt.Errorf("create admin user: %w", err)
	}
	return user, nil
}
