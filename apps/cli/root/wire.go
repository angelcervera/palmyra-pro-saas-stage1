package root

import (
	"github.com/zenGate-Global/palmyra-pro-saas/apps/cli/cmd/auth"
	"github.com/zenGate-Global/palmyra-pro-saas/apps/cli/cmd/bootstrap"
)

func init() {
	Root().AddCommand(auth.Command())
	Root().AddCommand(bootstrap.Command())
}
