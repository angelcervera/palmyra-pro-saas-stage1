package root

import (
	"github.com/zenGate-Global/palmyra-pro-saas/apps/cli/cmd/auth"
)

func init() {
	Root().AddCommand(auth.Command())
}
