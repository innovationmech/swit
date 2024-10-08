package main

import (
	"fmt"
	"os"

	"github.com/innovationmech/swit/internal/switctl/cmd"
	"github.com/spf13/viper"
)

func main() {
	rootCmd := cmd.NewRootSwitCtlCommand()

	viper.BindPFlag("address", rootCmd.PersistentFlags().Lookup("address"))
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
