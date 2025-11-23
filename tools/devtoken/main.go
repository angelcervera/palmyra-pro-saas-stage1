package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/auth/devtoken"
)

func main() {
	projectID := flag.String("project-id", "", "Firebase project ID (used for iss/aud)")
	tenant := flag.String("tenant", "", "firebase.tenant claim")
	userID := flag.String("user-id", "", "user_id/sub/uid claim")
	email := flag.String("email", "", "email claim")
	name := flag.String("name", "", "display name")
	emailVerified := flag.Bool("email-verified", true, "email_verified claim")
	isAdmin := flag.Bool("admin", false, "set isAdmin=true for admin role")
	palmyraRoles := flag.String("palmyra-roles", "", "comma-separated custom palmyraRoles array")
	tenantRoles := flag.String("tenant-roles", "", "comma-separated custom tenantRoles array")
	signInProvider := flag.String("sign-in-provider", "password", "firebase.sign_in_provider claim")
	expiresIn := flag.Duration("expires-in", time.Hour, "token lifetime (duration, e.g. 30m, 2h)")
	audience := flag.String("audience", "", "override aud (defaults to project-id)")
	issuer := flag.String("issuer", "", "override iss (defaults to https://securetoken.google.com/<project-id>)")

	flag.Parse()

	params := devtoken.Params{
		ProjectID:              strings.TrimSpace(*projectID),
		Tenant:                 strings.TrimSpace(*tenant),
		UserID:                 strings.TrimSpace(*userID),
		Email:                  strings.TrimSpace(*email),
		Name:                   strings.TrimSpace(*name),
		EmailVerified:          *emailVerified,
		IsAdmin:                *isAdmin,
		PalmyraRoles:           splitCSV(*palmyraRoles),
		TenantRoles:            splitCSV(*tenantRoles),
		FirebaseSignInProvider: strings.TrimSpace(*signInProvider),
		ExpiresIn:              *expiresIn,
		Audience:               strings.TrimSpace(*audience),
		Issuer:                 strings.TrimSpace(*issuer),
	}

	token, err := devtoken.BuildUnsignedFirebaseToken(params, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(token)
}

func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
