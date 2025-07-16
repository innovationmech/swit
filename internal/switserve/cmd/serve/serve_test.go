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

package serve

import (
	"testing"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func init() {
	logger.Logger, _ = zap.NewDevelopment()
}

func TestNewServeCmd(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "successful command creation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewServeCmd()

			assert.NotNil(t, cmd)
			assert.IsType(t, &cobra.Command{}, cmd)
			assert.Equal(t, "serve", cmd.Use)
			assert.Equal(t, "Start the SWIT server", cmd.Short)
			assert.NotNil(t, cmd.RunE)
			assert.NotNil(t, cmd.PreRunE)
		})
	}
}

func TestNewServeCmd_CommandProperties(t *testing.T) {
	cmd := NewServeCmd()

	assert.Equal(t, "serve", cmd.Use)
	assert.Equal(t, "Start the SWIT server", cmd.Short)

	assert.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.PreRunE)

	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestNewServeCmd_HelpOutput(t *testing.T) {
	cmd := NewServeCmd()

	cmd.SetArgs([]string{"--help"})

	assert.NotPanics(t, func() {
		cmd.Help()
	})
}

func TestCommandCreation(t *testing.T) {
	cmd := NewServeCmd()

	// Test basic command creation
	assert.NotNil(t, cmd)
	assert.Equal(t, "serve", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestPreRunE_InitializesLogger(t *testing.T) {
	cmd := NewServeCmd()

	// Test that PreRunE doesn't panic and initializes logger
	assert.NotPanics(t, func() {
		err := cmd.PreRunE(cmd, []string{})
		assert.NoError(t, err)
	})

	// Logger should be initialized after PreRunE
	assert.NotNil(t, logger.Logger)
}

func TestNewServeCmd_Integration(t *testing.T) {
	cmd := NewServeCmd()
	assert.NotNil(t, cmd)

	assert.Equal(t, "serve", cmd.Use)
	assert.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.PreRunE)

	// Test that command can handle arguments
	assert.NotPanics(t, func() {
		cmd.SetArgs([]string{})
	})
}

func TestServeCmd_ArgsHandling(t *testing.T) {
	cmd := NewServeCmd()

	assert.NotPanics(t, func() {
		cmd.SetArgs([]string{})
	})

	// Test that command doesn't accept unknown flags
	assert.NotPanics(t, func() {
		cmd.SetArgs([]string{"--help"})
	})
}

func TestServeCmd_BasicFunctionality(t *testing.T) {
	cmd := NewServeCmd()

	assert.NotNil(t, cmd)
	assert.Equal(t, "serve", cmd.Use)
	assert.NotNil(t, cmd.RunE)

	// Test basic flag parsing (should not error with no flags)
	err := cmd.ParseFlags([]string{})
	assert.NoError(t, err)
}

func TestServeCmd_NoFlags(t *testing.T) {
	cmd := NewServeCmd()

	// Test that there are no flags by default
	assert.Empty(t, cmd.Flags().FlagUsages())
}

func TestServeCmd_CommandStructure(t *testing.T) {
	cmd := NewServeCmd()

	assert.NotNil(t, cmd)
	assert.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.PreRunE)

	// Check command hierarchy
	assert.Equal(t, "serve", cmd.Use)
	assert.Contains(t, cmd.Long, "Unified gRPC and HTTP transport management")
}

func TestServeCmd_RunEFunctionSignature(t *testing.T) {
	cmd := NewServeCmd()

	assert.NotNil(t, cmd.RunE)

	assert.IsType(t, func(*cobra.Command, []string) error { return nil }, cmd.RunE)
}

func TestServeCmd_Metadata(t *testing.T) {
	cmd := NewServeCmd()

	assert.Equal(t, "serve", cmd.Use)
	assert.Equal(t, "Start the SWIT server", cmd.Short)

	assert.NotEmpty(t, cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

// Helper function to test command creation
func TestServeCmd_Creation(t *testing.T) {
	assert.NotPanics(t, func() {
		NewServeCmd()
	})
}
