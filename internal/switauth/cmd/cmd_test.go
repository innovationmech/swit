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

func TestNewSwitAuthCmd(t *testing.T) {
	tests := []struct {
		name     string
		validate func(t *testing.T, cmd *cobra.Command)
	}{
		{
			name: "should create switauth command successfully",
			validate: func(t *testing.T, cmd *cobra.Command) {
				assert.NotNil(t, cmd)
				assert.Equal(t, "switauth", cmd.Use)
				assert.Equal(t, "swit authentication service", cmd.Short)
				assert.Equal(t, "0.0.2", cmd.Version)
			},
		},
		{
			name: "should have start command",
			validate: func(t *testing.T, cmd *cobra.Command) {
				startCmd, _, err := cmd.Find([]string{"start"})
				assert.NoError(t, err)
				assert.NotNil(t, startCmd)
				assert.Equal(t, "start", startCmd.Name())
			},
		},
		{
			name: "should have correct command structure",
			validate: func(t *testing.T, cmd *cobra.Command) {
				assert.NotEmpty(t, cmd.Commands())
				assert.Len(t, cmd.Commands(), 1) // Only start command
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewSwitAuthCmd()
			tt.validate(t, cmd)
		})
	}
}

func TestCommandExecution(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		validate func(t *testing.T, output string, err error)
	}{
		{
			name: "help command",
			args: []string{"--help"},
			validate: func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
				assert.Contains(t, output, "Usage:")
				assert.Contains(t, output, "switauth [command]")
			},
		},
		{
			name: "version command",
			args: []string{"--version"},
			validate: func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
				assert.Contains(t, output, "switauth version 0.0.2")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new command instance for each test to avoid state retention
			cmd := NewSwitAuthCmd()

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			_, err := cmd.ExecuteC()
			tt.validate(t, buf.String(), err)
		})
	}
}

func TestCommandWithInvalidArgs(t *testing.T) {
	cmd := NewSwitAuthCmd()
	cmd.SetArgs([]string{"invalid-command"})

	err := cmd.Execute()
	assert.Error(t, err)
}
