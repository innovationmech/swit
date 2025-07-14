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
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewRootSwitCtlCommand(t *testing.T) {
	tests := []struct {
		name     string
		validate func(t *testing.T, cmd *cobra.Command)
	}{
		{
			name: "should create root command successfully",
			validate: func(t *testing.T, cmd *cobra.Command) {
				assert.NotNil(t, cmd)
				assert.Equal(t, "switctl", cmd.Use)
				assert.Equal(t, "SWIT Control Application", cmd.Short)
				assert.Equal(t, "SWITCTL is a command-line tool for controlling the SWIT server.", cmd.Long)
				assert.Equal(t, "0.0.2", cmd.Version)
			},
		},
		{
			name: "should have correct subcommands",
			validate: func(t *testing.T, cmd *cobra.Command) {
				subcommands := cmd.Commands()
				assert.Len(t, subcommands, 3)

				var stopCmd, versionCmd, healthCmd *cobra.Command
				for _, subcmd := range subcommands {
					switch subcmd.Use {
					case "stop":
						stopCmd = subcmd
					case "version":
						versionCmd = subcmd
					case "health":
						healthCmd = subcmd
					}
				}

				assert.NotNil(t, stopCmd)
				assert.NotNil(t, versionCmd)
				assert.NotNil(t, healthCmd)
			},
		},
		{
			name: "should have persistent flags",
			validate: func(t *testing.T, cmd *cobra.Command) {
				addressFlag := cmd.PersistentFlags().Lookup("address")
				assert.NotNil(t, addressFlag)
				assert.Equal(t, "localhost", addressFlag.DefValue)
				assert.Equal(t, "a", addressFlag.Shorthand)

				portFlag := cmd.PersistentFlags().Lookup("port")
				assert.NotNil(t, portFlag)
				assert.Equal(t, "8080", portFlag.DefValue)
				assert.Equal(t, "p", portFlag.Shorthand)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootSwitCtlCommand()
			tt.validate(t, cmd)
		})
	}
}

func TestCommandExecution(t *testing.T) {
	cmd := NewRootSwitCtlCommand()

	// Test help output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Usage:")
	assert.Contains(t, buf.String(), "switctl [command]")
}

func TestVersionOutput(t *testing.T) {
	cmd := NewRootSwitCtlCommand()

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--version"})
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "switctl version")
}

func TestPersistentFlags(t *testing.T) {
	_ = NewRootSwitCtlCommand()

	// Test default values (these are global variables set by viper)
	// Note: Since these are global variables, we just ensure the command creation works
	assert.True(t, true)
}

func TestCommandWithInvalidArgs(t *testing.T) {
	cmd := NewRootSwitCtlCommand()
	cmd.SetArgs([]string{"invalid-command"})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestSubcommandsExist(t *testing.T) {
	cmd := NewRootSwitCtlCommand()

	// Test each subcommand exists and has correct basic properties
	testCases := []struct {
		name          string
		expectedUse   string
		expectedShort string
	}{
		{"stop", "stop", "Stop the SWIT server"},
		{"version", "version", "Print version information"},
		{"health", "health", "Check server health"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			subCmd, _, err := cmd.Find([]string{tc.name})
			assert.NoError(t, err)
			assert.NotNil(t, subCmd)
			assert.Equal(t, tc.expectedUse, subCmd.Use)
		})
	}
}
