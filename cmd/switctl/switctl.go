package main

import (
	"fmt"
	"os"

	"github.com/innovationmech/swit/internal/component-base/cli"
	"github.com/innovationmech/swit/internal/switctl/cmd"
)

func main() {
	rootCmd := cmd.NewRootSwitCtlCommand()
	if err := cli.Run(rootCmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
