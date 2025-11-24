package schemacmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	schemacategoriesrepo "github.com/zenGate-Global/palmyra-pro-saas/domains/schema-categories/be/repo"
	schemacategoriesservice "github.com/zenGate-Global/palmyra-pro-saas/domains/schema-categories/be/service"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/requesttrace"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

// Command groups schema-related CLI helpers (categories today, repository later).
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Schema utilities (categories, repository)",
	}

	cmd.AddCommand(categoriesCommand())
	cmd.AddCommand(definitionsCommand())
	return cmd
}

func categoriesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "categories",
		Short: "Manage schema categories (list, upsert, delete)",
	}

	cmd.PersistentFlags().String("database-url", "", "PostgreSQL connection string")
	cmd.PersistentFlags().String("env-key", "dev", "Environment key used to derive admin schema (e.g. dev, stg, prod)")
	cmd.PersistentFlags().String("admin-tenant-slug", "admin", "Admin tenant slug used to derive admin schema")
	_ = cmd.MarkPersistentFlagRequired("database-url")

	cmd.AddCommand(listCategoriesCommand())
	cmd.AddCommand(upsertCategoryCommand())
	cmd.AddCommand(deleteCategoryCommand())
	return cmd
}

func listCategoriesCommand() *cobra.Command {
	var includeDeleted bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List schema categories",
		RunE: func(cmd *cobra.Command, _ []string) error {
			databaseURL, err := cmd.Flags().GetString("database-url")
			if err != nil {
				return err
			}
			envKey, _ := cmd.Flags().GetString("env-key")
			adminTenantSlug, _ := cmd.Flags().GetString("admin-tenant-slug")

			ctx := context.Background()
			svc, cleanup, err := newSchemaCategoryService(ctx, databaseURL, envKey, adminTenantSlug)
			if err != nil {
				return err
			}
			defer cleanup()

			audit := requesttrace.System("cli-schema-categories-list")
			ctx = requesttrace.IntoContext(ctx, audit)

			categories, err := svc.List(ctx, audit, includeDeleted)
			if err != nil {
				return fmt.Errorf("list schema categories: %w", err)
			}

			if len(categories) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No schema categories found.")
				return nil
			}

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tNAME\tSLUG\tPARENT\tDELETED_AT")
			for _, c := range categories {
				parent := "-"
				if c.ParentID != nil {
					parent = c.ParentID.String()
				}
				deleted := ""
				if c.DeletedAt != nil {
					deleted = c.DeletedAt.UTC().Format(time.RFC3339)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", c.ID, c.Name, c.Slug, parent, deleted)
			}
			return tw.Flush()
		},
	}

	cmd.Flags().BoolVar(&includeDeleted, "include-deleted", false, "Include soft-deleted categories")
	return cmd
}

func upsertCategoryCommand() *cobra.Command {
	var (
		categoryIDInput string
		nameInput       string
		slugInput       string
		parentInput     string
		description     string
	)

	cmd := &cobra.Command{
		Use:   "upsert",
		Short: "Create a new category or update an existing one",
		RunE: func(cmd *cobra.Command, _ []string) error {
			databaseURL, err := cmd.Flags().GetString("database-url")
			if err != nil {
				return err
			}
			envKey, _ := cmd.Flags().GetString("env-key")
			adminTenantSlug, _ := cmd.Flags().GetString("admin-tenant-slug")

			ctx := context.Background()
			svc, cleanup, err := newSchemaCategoryService(ctx, databaseURL, envKey, adminTenantSlug)
			if err != nil {
				return err
			}
			defer cleanup()

			audit := requesttrace.System("cli-schema-categories-upsert")
			ctx = requesttrace.IntoContext(ctx, audit)

			parentID, err := parseOptionalUUID(parentInput, "parent-id")
			if err != nil {
				return err
			}

			if strings.TrimSpace(categoryIDInput) == "" {
				if strings.TrimSpace(nameInput) == "" {
					return errors.New("name is required when creating a category")
				}
				if strings.TrimSpace(slugInput) == "" {
					return errors.New("slug is required when creating a category")
				}

				category, createErr := svc.Create(ctx, audit, schemacategoriesservice.CreateInput{
					Name:        nameInput,
					Slug:        slugInput,
					ParentID:    parentID,
					Description: stringPtrOrNil(description),
				})
				if createErr != nil {
					return wrapCategoryError("create", createErr)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "Created schema category %s (%s)\n", category.Name, category.ID)
				printCategorySummary(cmd.OutOrStdout(), category)
				return nil
			}

			categoryID, err := uuid.Parse(strings.TrimSpace(categoryIDInput))
			if err != nil {
				return fmt.Errorf("invalid category id: %w", err)
			}

			input := schemacategoriesservice.UpdateInput{}
			if cmd.Flags().Changed("name") {
				input.Name = stringPtr(nameInput)
			}
			if cmd.Flags().Changed("slug") {
				input.Slug = stringPtr(slugInput)
			}
			if cmd.Flags().Changed("description") {
				input.Description = stringPtr(description)
			}
			if cmd.Flags().Changed("parent-id") {
				input.ParentID = parentID
			}

			category, updateErr := svc.Update(ctx, audit, categoryID, input)
			if updateErr != nil {
				return wrapCategoryError("update", updateErr)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updated schema category %s (%s)\n", category.Name, category.ID)
			printCategorySummary(cmd.OutOrStdout(), category)
			return nil
		},
	}

	cmd.Flags().StringVar(&categoryIDInput, "id", "", "Category ID; when omitted, a new category is created")
	cmd.Flags().StringVar(&nameInput, "name", "", "Category name")
	cmd.Flags().StringVar(&slugInput, "slug", "", "Category slug")
	cmd.Flags().StringVar(&parentInput, "parent-id", "", "Optional parent category ID")
	cmd.Flags().StringVar(&description, "description", "", "Optional description")

	return cmd
}

func deleteCategoryCommand() *cobra.Command {
	var categoryIDInput string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Soft delete a schema category by ID",
		RunE: func(cmd *cobra.Command, _ []string) error {
			databaseURL, err := cmd.Flags().GetString("database-url")
			if err != nil {
				return err
			}
			envKey, _ := cmd.Flags().GetString("env-key")
			adminTenantSlug, _ := cmd.Flags().GetString("admin-tenant-slug")

			categoryID, err := uuid.Parse(strings.TrimSpace(categoryIDInput))
			if err != nil {
				return fmt.Errorf("invalid category id: %w", err)
			}

			ctx := context.Background()
			svc, cleanup, err := newSchemaCategoryService(ctx, databaseURL, envKey, adminTenantSlug)
			if err != nil {
				return err
			}
			defer cleanup()

			audit := requesttrace.System("cli-schema-categories-delete")
			ctx = requesttrace.IntoContext(ctx, audit)

			if err := svc.Delete(ctx, audit, categoryID); err != nil {
				return wrapCategoryError("delete", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Soft deleted schema category %s\n", categoryID)
			return nil
		},
	}

	cmd.Flags().StringVar(&categoryIDInput, "id", "", "Category ID to delete")
	_ = cmd.MarkFlagRequired("id")

	return cmd
}

func newSchemaCategoryService(ctx context.Context, databaseURL, envKey, adminTenantSlug string) (schemacategoriesservice.Service, func(), error) {
	pool, err := persistence.NewPool(ctx, persistence.PoolConfig{ConnString: databaseURL})
	if err != nil {
		return nil, nil, fmt.Errorf("init pool: %w", err)
	}

	adminSchema := tenant.BuildSchemaName(envKey, tenant.ToSnake(adminTenantSlug))

	spaceDB := persistence.NewSpaceDB(persistence.SpaceDBConfig{
		Pool:        pool,
		AdminSchema: adminSchema,
	})

	store, err := persistence.NewSchemaCategoryStore(ctx, pool)
	if err != nil {
		persistence.ClosePool(pool)
		return nil, nil, fmt.Errorf("init schema category store: %w", err)
	}

	repo := schemacategoriesrepo.NewPostgresRepository(spaceDB, store)
	svc := schemacategoriesservice.New(repo)

	cleanup := func() {
		persistence.ClosePool(pool)
	}

	return svc, cleanup, nil
}

func parseOptionalUUID(value, field string) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid UUID: %w", field, err)
	}
	return &parsed, nil
}

func wrapCategoryError(action string, err error) error {
	var validationErr *schemacategoriesservice.ValidationError
	switch {
	case errors.As(err, &validationErr):
		return fmt.Errorf("%s validation failed:\n%s", action, formatFieldErrors(map[string][]string(validationErr.Fields)))
	case errors.Is(err, schemacategoriesservice.ErrConflict):
		return fmt.Errorf("%s conflict: name or slug already exists", action)
	case errors.Is(err, schemacategoriesservice.ErrNotFound):
		return fmt.Errorf("%s failed: category not found", action)
	default:
		return fmt.Errorf("%s failed: %w", action, err)
	}
}

func formatFieldErrors(fields map[string][]string) string {
	keys := make([]string, 0, len(fields))
	for field := range fields {
		keys = append(keys, field)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, field := range keys {
		for _, msg := range fields[field] {
			fmt.Fprintf(&b, "- %s: %s\n", field, msg)
		}
	}
	return strings.TrimSpace(b.String())
}

func printCategorySummary(out io.Writer, category schemacategoriesservice.Category) {
	parent := "-"
	if category.ParentID != nil {
		parent = category.ParentID.String()
	}
	deleted := ""
	if category.DeletedAt != nil {
		deleted = category.DeletedAt.UTC().Format(time.RFC3339)
	}
	description := ""
	if category.Description != nil {
		description = *category.Description
	}

	fmt.Fprintf(out, "Name: %s\nSlug: %s\nParent: %s\nDescription: %s\nDeletedAt: %s\n", category.Name, category.Slug, parent, description, deleted)
}

func stringPtr(s string) *string {
	return &s
}

func stringPtrOrNil(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return stringPtr(s)
}
