package cli

import (
	"github.com/spf13/cobra"
)

func Run(cmd *cobra.Command) error {
	if err := cmd.Execute(); err != nil {
		return err
	}
	return nil
}
