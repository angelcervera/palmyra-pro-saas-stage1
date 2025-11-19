package persistence

import (
	"strings"
	"testing"
)

func TestNormalizeEntityIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "trim and accept", input: "  order-123 ", want: "order-123"},
		{name: "any punctuation allowed", input: "  inv/123 🔥  ", want: "inv/123 🔥"},
		{name: "emoji allowed", input: "🚀-alpha", want: "🚀-alpha"},
		{name: "empty", input: "", wantErr: true},
		{name: "too long ascii", input: strings.Repeat("a", 130), wantErr: true},
		{name: "too long unicode", input: strings.Repeat("🚀", 129), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeEntityIdentifier(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("want %q got %q", tt.want, got)
			}
		})
	}
}
