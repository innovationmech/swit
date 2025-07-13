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
// OUT of OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/innovationmech/swit/internal/switserve/cmd/version"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func init() {
	logger.Logger, _ = zap.NewDevelopment()
}

func TestNewRootServeCmdCommand(t *testing.T) {
	// Helper function to execute a command and capture its output
	execute := func(cmd *cobra.Command, args ...string) (string, string, error) {
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		cmd.SetOut(stdout)
		cmd.SetErr(stderr)
		cmd.SetArgs(args)
		err := cmd.Execute()
		return stdout.String(), stderr.String(), err
	}

	t.Run("root command properties", func(t *testing.T) {
		cmd := NewRootServeCmdCommand()
		assert.NotNil(t, cmd)
		assert.Equal(t, "swit", cmd.Use)
		assert.Equal(t, "swit server application", cmd.Short)
		assert.Equal(t, "0.0.2", cmd.Version)
		assert.False(t, cmd.HasParent())
	})

	t.Run("subcommands", func(t *testing.T) {
		cmd := NewRootServeCmdCommand()
		subcommands := cmd.Commands()
		assert.Len(t, subcommands, 2)

		var serveCmd, versionCmd *cobra.Command
		for _, subcmd := range subcommands {
			switch subcmd.Use {
			case "serve":
				serveCmd = subcmd
			case "version":
				versionCmd = subcmd
			}
		}

		assert.NotNil(t, serveCmd)
		assert.Equal(t, "serve", serveCmd.Use)
		assert.Equal(t, "Start the SWIT server", serveCmd.Short)
		assert.True(t, serveCmd.HasParent())
		assert.Equal(t, cmd, serveCmd.Parent())

		assert.NotNil(t, versionCmd)
		assert.Equal(t, "version", versionCmd.Use)
		assert.Equal(t, "Print version", versionCmd.Short)
		assert.True(t, versionCmd.HasParent())
		assert.Equal(t, cmd, versionCmd.Parent())
	})

	t.Run("help output", func(t *testing.T) {
		cmd := NewRootServeCmdCommand()
		// Redirect output to a buffer to check if help is printed
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"--help"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "Usage:")
		assert.Contains(t, buf.String(), "swit [command]")
	})

	t.Run("version output", func(t *testing.T) {
		cmd := NewRootServeCmdCommand()
		output, _, err := execute(cmd, "--version")
		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(output, "swit version"))
	})

	t.Run("version subcommand execution", func(t *testing.T) {
		// Create a root command without serve to avoid config dependency
		cmd := &cobra.Command{
			Use:     "swit",
			Short:   "swit server application",
			Version: "0.0.2",
		}
		cmd.AddCommand(version.NewVersionCommand())

		output, _, err := execute(cmd, "version")
		assert.NoError(t, err)
		assert.Contains(t, output, "swit version")
	})

	t.Run("invalid subcommand", func(t *testing.T) {
		// Create a root command without serve to avoid config dependency
		cmd := &cobra.Command{
			Use:     "swit",
			Short:   "swit server application",
			Version: "0.0.2",
		}
		cmd.AddCommand(version.NewVersionCommand())

		_, _, err := execute(cmd, "invalid")
		assert.Error(t, err)
	})
}
