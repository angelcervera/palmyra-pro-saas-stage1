package schemacmd

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	schemacategoriesservice "github.com/zenGate-Global/palmyra-pro-saas/domains/schema-categories/be/service"
)

// Command groups schema-related CLI helpers (categories, definitions).
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Schema utilities (categories, repository)",
	}

	cmd.AddCommand(categoriesCommand())
	cmd.AddCommand(definitionsCommand())
	return cmd
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

func stringPtr(s string) *string {
	return &s
}

func stringPtrOrNil(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return stringPtr(s)
}
