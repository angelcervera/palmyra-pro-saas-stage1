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
		{name: "uppercase allowed", input: "Card.ABC", want: "Card.ABC"},
		{name: "colon allowed", input: "inv:item-42", want: "inv:item-42"},
		{name: "empty", input: "", wantErr: true},
		{name: "invalid char", input: "bad/char", wantErr: true},
		{name: "too long", input: strings.Repeat("a", 130), wantErr: true},
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
