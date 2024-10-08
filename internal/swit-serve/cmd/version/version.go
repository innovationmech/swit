package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewVersionCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("swit version 0.0.2")
		},
	}
	return cmds
}
