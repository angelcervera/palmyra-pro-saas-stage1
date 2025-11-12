package persistence

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var tableNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// normalizeTableName trims the input and enforces a lowercase snake_case identifier that is safe to embed in SQL.
func normalizeTableName(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", errors.New("table name is required")
	}

	if !tableNamePattern.MatchString(trimmed) {
		return "", fmt.Errorf("invalid table name %q: must match ^[a-z][a-z0-9_]*$", trimmed)
	}

	return trimmed, nil
}
