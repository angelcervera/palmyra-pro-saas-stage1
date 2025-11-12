package persistence

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// NormalizeSlug trims whitespace, lowercases the value, and ensures it matches
// the canonical URL-safe slug pattern required for public identifiers.
func NormalizeSlug(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", errors.New("slug is required")
	}

	normalized := strings.ToLower(trimmed)
	if !slugPattern.MatchString(normalized) {
		return "", fmt.Errorf("invalid slug %q: must match ^[a-z0-9]+(?:-[a-z0-9]+)*$", input)
	}

	return normalized, nil
}
