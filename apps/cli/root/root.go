package root

import (
	"github.com/spf13/cobra"
)

// rootCmd is the base command for the Palmyra admin CLI. Subcommands (auth, bootstrap, etc.) are attached here.
var rootCmd = &cobra.Command{
	Use:           "palmyra",
	Short:         "Palmyra admin CLI",
	Long:          "Administrative utilities for Palmyra (dev tokens, bootstrap helpers, tenant/user management).",
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute runs the CLI.
func Execute() error {
	return rootCmd.Execute()
}

// Root returns the mutable root command for wiring from subpackages.
func Root() *cobra.Command {
	return rootCmd
}
