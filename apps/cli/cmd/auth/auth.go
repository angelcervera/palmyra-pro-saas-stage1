package auth

import "github.com/spf13/cobra"

// Command groups authentication-related helpers (dev tokens, future user/tenant commands).
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication utilities",
		Long:  "Authentication utilities (dev tokens, user provisioning, tenant auth setup).",
	}

	// Subcommands wired in init of individual files.
	cmd.AddCommand(devTokenCommand())

	return cmd
}
