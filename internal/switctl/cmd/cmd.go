package cmd

import (
	"github.com/innovationmech/swit/internal/switctl/cmd/health"
	"github.com/innovationmech/swit/internal/switctl/cmd/stop"
	"github.com/innovationmech/swit/internal/switctl/cmd/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	serverAddress string
	serverPort    string
)

func NewRootSwitCtlCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "switctl",
		Short:   "SWIT Control Application",
		Long:    `SWITCTL is a command-line tool for controlling the SWIT server.`,
		Version: "0.0.2",
	}

	rootCmd.PersistentFlags().StringVarP(&serverAddress, "address", "a", "localhost", "SWIT server address")
	rootCmd.PersistentFlags().StringVarP(&serverPort, "port", "p", "8080", "SWIT server port")

	rootCmd.AddCommand(stop.NewStopCmd())
	rootCmd.AddCommand(version.NewSwitctlVersionCmd())
	rootCmd.AddCommand(health.NewHealthCmd())

	viper.BindPFlag("address", rootCmd.PersistentFlags().Lookup("address"))
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))

	return rootCmd
}
