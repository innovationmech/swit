package cmd

import (
	"github.com/innovationmech/swit/internal/switctl/cmd/version"
	"github.com/spf13/cobra"
)

func NewRootSwitCtlCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "switctl",
		Short: "switctl is a command-line tool for managing Swit",
	}
	cmds.AddCommand(version.NewSwitctlVersionCmd())
	return cmds
}
