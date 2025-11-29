package schemacmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	schemarepositoryrepo "github.com/zenGate-Global/palmyra-pro-saas/domains/schema-repository/be/repo"
	schemarepositoryservice "github.com/zenGate-Global/palmyra-pro-saas/domains/schema-repository/be/service"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/persistence"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/requesttrace"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/tenant"
)

func definitionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "definitions",
		Short: "Manage schema definitions (list, upsert, delete)",
	}

	cmd.PersistentFlags().String("database-url", "", "PostgreSQL connection string")
	cmd.PersistentFlags().String("env-key", "dev", "Environment key used to derive admin schema (e.g. dev, stg, prod)")
	cmd.PersistentFlags().String("admin-tenant-slug", "admin", "Admin tenant slug used to derive admin schema")
	_ = cmd.MarkPersistentFlagRequired("database-url")

	cmd.AddCommand(listDefinitionsCommand())
	cmd.AddCommand(upsertDefinitionCommand())
	cmd.AddCommand(deleteDefinitionCommand())

	return cmd
}

func listDefinitionsCommand() *cobra.Command {
	var (
		schemaIDInput   string
		includeDeleted  bool
		includeInactive bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List schema definitions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			databaseURL, err := cmd.Flags().GetString("database-url")
			if err != nil {
				return err
			}
			envKey, _ := cmd.Flags().GetString("env-key")
			adminTenantSlug, _ := cmd.Flags().GetString("admin-tenant-slug")

			ctx := context.Background()
			svc, cleanup, err := newSchemaDefinitionService(ctx, databaseURL, envKey, adminTenantSlug)
			if err != nil {
				return err
			}
			defer cleanup()

			audit := requesttrace.System("cli-schema-definitions-list")
			ctx = requesttrace.IntoContext(ctx, audit)

			var schemas []schemarepositoryservice.Schema

			trimmedID := strings.TrimSpace(schemaIDInput)
			if trimmedID == "" {
				schemas, err = svc.ListAll(ctx, audit, includeInactive)
			} else {
				schemaID, parseErr := uuid.Parse(trimmedID)
				if parseErr != nil {
					return fmt.Errorf("invalid schema id: %w", parseErr)
				}
				schemas, err = svc.List(ctx, audit, schemaID, includeDeleted)
			}

			if err != nil {
				return fmt.Errorf("list schema definitions: %w", err)
			}

			if len(schemas) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No schema definitions found.")
				return nil
			}

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "SCHEMA_ID\tVERSION\tACTIVE\tDELETED\tTABLE\tSLUG\tCATEGORY_ID\tCREATED_AT")
			for _, s := range schemas {
				fmt.Fprintf(tw, "%s\t%s\t%t\t%t\t%s\t%s\t%s\t%s\n",
					s.SchemaID,
					s.Version.String(),
					s.IsActive,
					s.IsDeleted,
					s.TableName,
					s.Slug,
					s.CategoryID,
					s.CreatedAt.UTC().Format(time.RFC3339),
				)
			}
			return tw.Flush()
		},
	}

	cmd.Flags().StringVar(&schemaIDInput, "schema-id", "", "Schema ID to filter versions; when omitted, all schemas are listed")
	cmd.Flags().BoolVar(&includeDeleted, "include-deleted", false, "Include soft-deleted schema versions (only when filtering by schema-id)")
	cmd.Flags().BoolVar(&includeInactive, "include-inactive", false, "Include inactive schema versions (when listing all schemas)")

	return cmd
}

func upsertDefinitionCommand() *cobra.Command {
	var (
		schemaIDInput      string
		schemaVersionInput string
		tableNameInput     string
		slugInput          string
		categoryIDInput    string
		definitionPath     string
	)

	cmd := &cobra.Command{
		Use:   "upsert",
		Short: "Create a new schema definition or a new version of an existing schema",
		RunE: func(cmd *cobra.Command, _ []string) error {
			databaseURL, err := cmd.Flags().GetString("database-url")
			if err != nil {
				return err
			}
			envKey, _ := cmd.Flags().GetString("env-key")
			adminTenantSlug, _ := cmd.Flags().GetString("admin-tenant-slug")

			ctx := context.Background()
			svc, cleanup, err := newSchemaDefinitionService(ctx, databaseURL, envKey, adminTenantSlug)
			if err != nil {
				return err
			}
			defer cleanup()

			audit := requesttrace.System("cli-schema-definitions-upsert")
			ctx = requesttrace.IntoContext(ctx, audit)

			categoryID, err := uuid.Parse(strings.TrimSpace(categoryIDInput))
			if err != nil {
				return fmt.Errorf("invalid category id: %w", err)
			}

			var schemaID *uuid.UUID
			if id, parseErr := parseOptionalUUID(schemaIDInput, "schema-id"); parseErr != nil {
				return parseErr
			} else {
				schemaID = id
			}

			version, err := parseSemanticVersion(schemaVersionInput, "schema-version")
			if err != nil {
				return err
			}

			definition, err := readDefinition(definitionPath)
			if err != nil {
				return err
			}

			input := schemarepositoryservice.CreateInput{
				SchemaID:   schemaID,
				Version:    version,
				Definition: definition,
				TableName:  tableNameInput,
				Slug:       slugInput,
				CategoryID: categoryID,
			}

			schema, createErr := svc.Create(ctx, audit, input)
			if createErr != nil {
				return wrapDefinitionError("upsert", createErr)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Upserted schema definition %s version %s\n", schema.SchemaID, schema.Version.String())
			printDefinitionSummary(cmd.OutOrStdout(), schema)
			return nil
		},
	}

	cmd.Flags().StringVar(&schemaIDInput, "schema-id", "", "Schema ID; when omitted a new schema will be created or resolved by slug")
	cmd.Flags().StringVar(&schemaVersionInput, "schema-version", "", "Optional semantic version (e.g. 1.2.3). When omitted the next patch is used")
	cmd.Flags().StringVar(&tableNameInput, "table-name", "", "Table name backing the schema definitions")
	cmd.Flags().StringVar(&slugInput, "slug", "", "Schema slug; required when creating a new schema")
	cmd.Flags().StringVar(&categoryIDInput, "category-id", "", "Schema category ID (required)")
	cmd.Flags().StringVar(&definitionPath, "definition-file", "", "Path to the JSON Schema definition file (required)")

	_ = cmd.MarkFlagRequired("table-name")
	_ = cmd.MarkFlagRequired("slug")
	_ = cmd.MarkFlagRequired("category-id")
	_ = cmd.MarkFlagRequired("definition-file")

	return cmd
}

func deleteDefinitionCommand() *cobra.Command {
	var (
		schemaIDInput      string
		schemaVersionInput string
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Soft delete a schema definition version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			databaseURL, err := cmd.Flags().GetString("database-url")
			if err != nil {
				return err
			}
			envKey, _ := cmd.Flags().GetString("env-key")
			adminTenantSlug, _ := cmd.Flags().GetString("admin-tenant-slug")

			schemaID, err := uuid.Parse(strings.TrimSpace(schemaIDInput))
			if err != nil {
				return fmt.Errorf("invalid schema id: %w", err)
			}

			version, err := parseSemanticVersion(schemaVersionInput, "schema-version")
			if err != nil {
				return err
			}
			if version == nil {
				return errors.New("schema-version is required")
			}

			ctx := context.Background()
			svc, cleanup, err := newSchemaDefinitionService(ctx, databaseURL, envKey, adminTenantSlug)
			if err != nil {
				return err
			}
			defer cleanup()

			audit := requesttrace.System("cli-schema-definitions-delete")
			ctx = requesttrace.IntoContext(ctx, audit)

			if err := svc.Delete(ctx, audit, schemaID, *version); err != nil {
				return wrapDefinitionError("delete", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Soft deleted schema definition %s version %s\n", schemaID, version.String())
			return nil
		},
	}

	cmd.Flags().StringVar(&schemaIDInput, "schema-id", "", "Schema ID to delete")
	cmd.Flags().StringVar(&schemaVersionInput, "schema-version", "", "Semantic version to delete (e.g. 1.0.0)")
	_ = cmd.MarkFlagRequired("schema-id")
	_ = cmd.MarkFlagRequired("schema-version")

	return cmd
}

func newSchemaDefinitionService(ctx context.Context, databaseURL, envKey, adminTenantSlug string) (schemarepositoryservice.Service, func(), error) {
	pool, err := persistence.NewPool(ctx, persistence.PoolConfig{ConnString: databaseURL})
	if err != nil {
		return nil, nil, fmt.Errorf("init pool: %w", err)
	}

	adminSchema := tenant.BuildSchemaName(envKey, tenant.ToSnake(adminTenantSlug))

	spaceDB := persistence.NewSpaceDB(persistence.SpaceDBConfig{
		Pool:        pool,
		AdminSchema: adminSchema,
	})

	store, err := persistence.NewSchemaRepositoryStore(ctx, pool)
	if err != nil {
		persistence.ClosePool(pool)
		return nil, nil, fmt.Errorf("init schema repository store: %w", err)
	}

	repo := schemarepositoryrepo.NewPostgresRepository(spaceDB, store)
	svc := schemarepositoryservice.New(repo)

	cleanup := func() {
		persistence.ClosePool(pool)
	}

	return svc, cleanup, nil
}

func wrapDefinitionError(action string, err error) error {
	var validationErr *schemarepositoryservice.ValidationError
	switch {
	case errors.As(err, &validationErr):
		return fmt.Errorf("%s validation failed:\n%s", action, formatFieldErrors(map[string][]string(validationErr.Fields)))
	case errors.Is(err, schemarepositoryservice.ErrConflict):
		return fmt.Errorf("%s conflict: schema version already exists", action)
	case errors.Is(err, schemarepositoryservice.ErrNotFound):
		return fmt.Errorf("%s failed: schema or version not found", action)
	default:
		return fmt.Errorf("%s failed: %w", action, err)
	}
}

func parseSemanticVersion(value, field string) (*persistence.SemanticVersion, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	version, err := persistence.ParseSemanticVersion(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid semantic version: %w", field, err)
	}

	return &version, nil
}

func readDefinition(path string) (json.RawMessage, error) {
	data, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return nil, fmt.Errorf("read definition file: %w", err)
	}

	if !json.Valid(data) {
		return nil, errors.New("definition file does not contain valid JSON")
	}

	return json.RawMessage(data), nil
}

func printDefinitionSummary(out io.Writer, schema schemarepositoryservice.Schema) {
	fields := []string{
		fmt.Sprintf("SchemaID: %s", schema.SchemaID),
		fmt.Sprintf("Version: %s", schema.Version.String()),
		fmt.Sprintf("Slug: %s", schema.Slug),
		fmt.Sprintf("Table: %s", schema.TableName),
		fmt.Sprintf("CategoryID: %s", schema.CategoryID),
		fmt.Sprintf("Active: %t", schema.IsActive),
		fmt.Sprintf("Deleted: %t", schema.IsDeleted),
		fmt.Sprintf("CreatedAt: %s", schema.CreatedAt.UTC().Format(time.RFC3339)),
	}

	sort.Strings(fields)
	for _, f := range fields {
		fmt.Fprintln(out, f)
	}
}
