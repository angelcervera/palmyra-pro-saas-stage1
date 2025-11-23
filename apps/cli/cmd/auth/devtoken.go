package auth

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zenGate-Global/palmyra-pro-saas/platform/go/auth/devtoken"
)

func devTokenCommand() *cobra.Command {
	var params devtoken.Params
	var palmyraRoles []string
	var tenantRoles []string
	var expiresIn time.Duration

	cmd := &cobra.Command{
		Use:   "devtoken",
		Short: "Generate an unsigned Firebase-compatible JWT for dev/local use",
		RunE: func(cmd *cobra.Command, args []string) error {
			params.PalmyraRoles = palmyraRoles
			params.TenantRoles = tenantRoles
			params.ExpiresIn = expiresIn

			token, err := devtoken.BuildUnsignedFirebaseToken(params, time.Now().UTC())
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), token)
			return nil
		},
	}

	// Required claims
	cmd.Flags().StringVar(&params.ProjectID, "project-id", "", "Firebase project ID (iss/aud)")
	cmd.Flags().StringVar(&params.Tenant, "tenant", "", "firebase.tenant claim")
	cmd.Flags().StringVar(&params.UserID, "user-id", "", "user_id/sub/uid claim")
	cmd.Flags().StringVar(&params.Email, "email", "", "email claim")

	// Optional claims
	cmd.Flags().StringVar(&params.Name, "name", "", "display name")
	cmd.Flags().BoolVar(&params.EmailVerified, "email-verified", true, "email_verified claim")
	cmd.Flags().BoolVar(&params.IsAdmin, "admin", false, "set isAdmin=true")
	cmd.Flags().StringSliceVar(&palmyraRoles, "palmyra-roles", nil, "custom palmyraRoles array (comma-separated)")
	cmd.Flags().StringSliceVar(&tenantRoles, "tenant-roles", nil, "custom tenantRoles array (comma-separated)")
	cmd.Flags().StringVar(&params.FirebaseSignInProvider, "sign-in-provider", "password", "firebase.sign_in_provider claim")
	cmd.Flags().DurationVar(&expiresIn, "expires-in", time.Hour, "token lifetime (e.g. 30m, 2h)")
	cmd.Flags().StringVar(&params.Audience, "audience", "", "override aud; defaults to project-id")
	cmd.Flags().StringVar(&params.Issuer, "issuer", "", "override iss; defaults to securetoken URL")

	_ = cmd.MarkFlagRequired("project-id")
	_ = cmd.MarkFlagRequired("tenant")
	_ = cmd.MarkFlagRequired("user-id")
	_ = cmd.MarkFlagRequired("email")

	return cmd
}
