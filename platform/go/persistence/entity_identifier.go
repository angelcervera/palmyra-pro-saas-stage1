package persistence

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var entityIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// NormalizeEntityIdentifier trims input and ensures it matches the allowed pattern.
func NormalizeEntityIdentifier(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", errors.New("entity id is required")
	}
	if !entityIdentifierPattern.MatchString(trimmed) {
		return "", fmt.Errorf("invalid entity id %q", input)
	}
	return trimmed, nil
}
