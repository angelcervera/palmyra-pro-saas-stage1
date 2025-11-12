package persistence

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectSlug  string
		expectError bool
	}{
		{
			name:       "already normalized",
			input:      "cards-schema",
			expectSlug: "cards-schema",
		},
		{
			name:       "trims whitespace and lowercases",
			input:      "  Deck-Builders ",
			expectSlug: "deck-builders",
		},
		{
			name:        "empty string",
			input:       "   ",
			expectError: true,
		},
		{
			name:        "invalid characters",
			input:       "cards_schema",
			expectError: true,
		},
		{
			name:        "leading hyphen",
			input:       "-bad-slug",
			expectError: true,
		},
		{
			name:        "trailing hyphen",
			input:       "bad-slug-",
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			slug, err := NormalizeSlug(tt.input)
			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectSlug, slug)
		})
	}
}
