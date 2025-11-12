package auth

import (
	"reflect"
	"testing"

	"firebase.google.com/go/v4/auth"
)

func TestExtractClaims(t *testing.T) {
	vendorId := "VendorXXX"
	tests := []struct {
		name  string
		token auth.Token
		want  TcgLandClaims
	}{
		{
			"Is admin but is not Vendor",
			auth.Token{Claims: map[string]interface{}{"isAdmin": true}},
			TcgLandClaims{
				IsAdmin:  true,
				VendorId: nil,
			},
		},
		{
			"Is admin and vendor",
			auth.Token{Claims: map[string]interface{}{"isAdmin": true, "vendorId": vendorId}},
			TcgLandClaims{
				IsAdmin:  true,
				VendorId: &vendorId,
			},
		},
		{
			"No admin and vendor",
			auth.Token{Claims: map[string]interface{}{"isAdmin": false, "vendorId": vendorId}},
			TcgLandClaims{
				IsAdmin:  false,
				VendorId: &vendorId,
			},
		},
		{
			"No data in claims",
			auth.Token{Claims: map[string]interface{}{}},
			TcgLandClaims{
				IsAdmin:  false,
				VendorId: nil,
			},
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
