package root

import (
	"github.com/zenGate-Global/palmyra-pro-saas/apps/cli-platform-admin/cmd/auth"
	"github.com/zenGate-Global/palmyra-pro-saas/apps/cli-platform-admin/cmd/bootstrap"
	schemacmd "github.com/zenGate-Global/palmyra-pro-saas/apps/cli-platform-admin/cmd/schema"
	tenantcmd "github.com/zenGate-Global/palmyra-pro-saas/apps/cli-platform-admin/cmd/tenant"
)

func init() {
	Root().AddCommand(auth.Command())
	Root().AddCommand(bootstrap.Command())
	Root().AddCommand(schemacmd.Command())
	Root().AddCommand(tenantcmd.Command())
}
