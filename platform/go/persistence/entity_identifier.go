package persistence

import (
	"fmt"
	"regexp"
	"strings"
)

var entityIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// InvalidEntityIdentifierError indicates the identifier is missing or does not match the required pattern.
type InvalidEntityIdentifierError struct {
	reason string
}

func (e *InvalidEntityIdentifierError) Error() string {
	return e.reason
}

// NormalizeEntityIdentifier trims input and ensures it matches the allowed pattern.
func NormalizeEntityIdentifier(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", &InvalidEntityIdentifierError{reason: "entityId is required"}
	}
	if !entityIdentifierPattern.MatchString(trimmed) {
		return "", &InvalidEntityIdentifierError{reason: fmt.Sprintf("entityId %q does not match required pattern", input)}
	}
	return trimmed, nil
}
