package cmd

import (
	"github.com/innovationmech/swit/internal/swit-serve/cmd/serve"
	"github.com/innovationmech/swit/internal/swit-serve/cmd/version"
	"github.com/spf13/cobra"
)

func NewRootServeCmdCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "swit-serve",
		Short: "swit server application",
	}
	cmds.AddCommand(serve.NewServeCmd())
	cmds.AddCommand(version.NewVersionCommand())
	return cmds
}
