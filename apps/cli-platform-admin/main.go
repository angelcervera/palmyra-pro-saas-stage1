package main

import (
	"fmt"
	"os"

	"github.com/zenGate-Global/palmyra-pro-saas/apps/cli-platform-admin/root"
)

func main() {
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
