package persistence

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const maxEntityIdentifierLength = 128

// InvalidEntityIdentifierError indicates the identifier is missing or does not match the required pattern.
type InvalidEntityIdentifierError struct {
	reason string
}

func (e *InvalidEntityIdentifierError) Error() string {
	return e.reason
}

// NormalizeEntityIdentifier trims input and ensures it is non-empty and within the maximum length.
func NormalizeEntityIdentifier(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", &InvalidEntityIdentifierError{reason: "entityId is required"}
	}
	if utf8.RuneCountInString(trimmed) > maxEntityIdentifierLength {
		return "", &InvalidEntityIdentifierError{reason: fmt.Sprintf("entityId must be at most %d characters", maxEntityIdentifierLength)}
	}
	return trimmed, nil
}
