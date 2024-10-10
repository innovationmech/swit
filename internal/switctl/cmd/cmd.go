// Copyright © 2023 jackelyj <dreamerlyj@gmail.com>
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
	"github.com/innovationmech/swit/internal/switctl/cmd/health"
	"github.com/innovationmech/swit/internal/switctl/cmd/stop"
	"github.com/innovationmech/swit/internal/switctl/cmd/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// serverAddress is the address of the SWIT server.
	serverAddress string
	// serverPort is the port of the SWIT server.
	serverPort string
)

// NewRootSwitCtlCommand creates a new root command for SWITCTL.
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
