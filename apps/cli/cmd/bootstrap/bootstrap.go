package bootstrap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// Notes/constraints:
// - This command performs platform bootstrap only (admin schema + base tables + admin tenant/user seed).
// - It does NOT perform tenant-space provisioning (roles/schemas/grants); that remains in domains/tenants provisioning.
// - Tenant creation (TenantStore) always writes to the admin schema via admin-scoped connections.

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

			// Derive admin schema from slug.
			slugSnake := tenant.ToSnake(tenantSlug)
			adminSchema := tenant.BuildSchemaName(slugSnake)

			// Phase 1: platform bootstrap (admin schema + base tables)
			if err := persistence.BootstrapAdminSchema(ctx, pool, adminSchema); err != nil {
				return fmt.Errorf("bootstrap admin schema: %w", err)
			}

			tenantStore, err := persistence.NewTenantStore(ctx, pool, adminSchema)
			if err != nil {
				return fmt.Errorf("init tenant store: %w", err)
			}

			createdByID := uuid.New()
			if createdBy != "" {
				parsed, parseErr := uuid.Parse(createdBy)
				if parseErr != nil {
					return fmt.Errorf("invalid created-by uuid: %w", parseErr)
				}
				createdByID = parsed
			}

			// Seed admin tenant if missing.
			tenantRec, err := tenantStore.GetBySlug(ctx, tenantSlug)
			if err != nil {
				// Create initial admin tenant version 1.0.0
				adminID := uuid.New()
				der := tenant.DeriveIdentifiers(envKey, tenantSlug, adminID)
				rec := persistence.TenantRecord{
					TenantID:      adminID,
					TenantVersion: persistence.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
					Slug:          tenantSlug,
					DisplayName:   strPtrOrNil(defaultName(tenantSlug, tenantName)),
					Status:        "active",
					SchemaName:    adminSchema,
					RoleName:      der.RoleName, // role may be provisioned later
					BasePrefix:    der.BasePrefix,
					ShortTenantID: der.ShortTenantID,
					IsActive:      true,
					CreatedAt:     time.Now().UTC(),
					CreatedBy:     createdByID,
					DBReady:       true,
					AuthReady:     true,
				}
				tenantRec, err = tenantStore.Create(ctx, rec)
				if err != nil {
					return fmt.Errorf("create admin tenant: %w", err)
				}
			}

			// Optionally seed admin user inside admin schema without role switching.
			if strings.TrimSpace(adminEmail) != "" && strings.TrimSpace(adminFullName) != "" {
				tenantDB := persistence.NewTenantDB(persistence.TenantDBConfig{Pool: pool, AdminSchema: adminSchema})
				if err := seedAdminUser(ctx, tenantDB, adminEmail, adminFullName); err != nil {
					return fmt.Errorf("seed admin user: %w", err)
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Platform bootstrap complete. Admin tenant: %s (%s)\n", tenantRec.Slug, tenantRec.TenantID)
			return nil
		},
	}

	c.Flags().StringVar(&databaseURL, "database-url", "", "PostgreSQL connection string")
	c.Flags().StringVar(&envKey, "env-key", "dev", "Environment key prefix (e.g. dev, stg, prod)")
	c.Flags().StringVar(&tenantSlug, "tenant-slug", "admin", "Slug for admin tenant")
	c.Flags().StringVar(&tenantName, "tenant-name", "", "Display name for admin tenant (defaults to slug)")
	c.Flags().StringVar(&adminEmail, "admin-email", "", "Initial admin user email")
	c.Flags().StringVar(&adminFullName, "admin-full-name", "", "Initial admin user full name")
	c.Flags().StringVar(&createdBy, "created-by", "", "UUID for createdBy (optional; defaults to random)")

	_ = c.MarkFlagRequired("database-url")
	_ = c.MarkFlagRequired("env-key")
	_ = c.MarkFlagRequired("tenant-slug")
	// Admin email/full-name optional; when set we will seed admin user in admin schema.

	return c
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// defaultName returns the provided name or falls back to the slug.
func defaultName(slug, name string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return slug
}

// seedAdminUser inserts an admin user row inside the admin schema using WithAdmin.
// It is safe to run multiple times (unique email constraint).
func seedAdminUser(ctx context.Context, tenantDB *persistence.TenantDB, email, fullName string) error {
	email = strings.TrimSpace(email)
	fullName = strings.TrimSpace(fullName)
	if email == "" || fullName == "" {
		return fmt.Errorf("admin email and full name are required to seed user")
	}

	return tenantDB.WithAdmin(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
            INSERT INTO users (user_id, email, full_name)
            VALUES ($1, $2, $3)
            ON CONFLICT (email) DO NOTHING
        `, uuid.New(), email, fullName); err != nil {
			return err
		}
		return nil
	})
}
