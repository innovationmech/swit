package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewSwitctlVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("switctl version 0.0.1")
		},
	}
}
