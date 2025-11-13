package auth

import (
	"reflect"
	"testing"

	"firebase.google.com/go/v4/auth"
)

func TestExtractClaims(t *testing.T) {
	tests := []struct {
		name  string
		token auth.Token
		want  PalmyraClaims
	}{
		{
			"Is admin",
			auth.Token{Claims: map[string]interface{}{"isAdmin": true}},
			PalmyraClaims{IsAdmin: true},
		},
		{
			"Not admin",
			auth.Token{Claims: map[string]interface{}{"isAdmin": false}},
			PalmyraClaims{IsAdmin: false},
		},
		{
			"Missing flag",
			auth.Token{Claims: map[string]interface{}{}},
			PalmyraClaims{IsAdmin: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractClaims(tt.token); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractClaims() = %v, want %v", got, tt.want)
			}
		})
	}
}
