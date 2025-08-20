// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package cmd

import (
	"github.com/innovationmech/swit/internal/switctl/cmd/new"
	"github.com/innovationmech/swit/internal/switctl/cmd/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// verbose enables verbose output
	verbose bool
	// noColor disables colored output
	noColor bool
	// configFile is the path to the config file
	configFile string
)

// NewRootSwitCtlCommand creates a new root command for SWITCTL.
func NewRootSwitCtlCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "switctl",
		Short: "Swit Framework Scaffolding Tool",
		Long: `switctl is the official scaffolding tool for the Swit microservice framework.
It provides project generation, code quality checks, and development assistance features.

Use switctl to:
- Generate new services and components
- Run code quality checks and tests  
- Manage project dependencies and configuration
- Create API endpoints and data models`,
		Version: "0.1.0",
	}

	// Add global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path")

	// Add subcommands
	rootCmd.AddCommand(version.NewSwitctlVersionCmd())
	rootCmd.AddCommand(new.NewNewCommand())

	// TODO: Add remaining commands (generate, check, init, dev, deps, config)

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	return rootCmd
}
