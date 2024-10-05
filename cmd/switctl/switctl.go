package main

import (
	"fmt"
	"os"

	"github.com/innovationmech/swit/internal/switctl/cmd"
)

func main() {
	rootCmd := cmd.NewRootSwitCtlCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
