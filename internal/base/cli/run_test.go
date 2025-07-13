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

package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRun_Success(t *testing.T) {
	tests := []struct {
		name        string
		cmd         *cobra.Command
		expectError bool
	}{
		{
			name: "successful command execution",
			cmd: &cobra.Command{
				Use: "test",
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			},
			expectError: false,
		},
		{
			name: "successful command with Run function",
			cmd: &cobra.Command{
				Use: "test-run",
				Run: func(cmd *cobra.Command, args []string) {
					// Success, no error
				},
			},
			expectError: false,
		},
		{
			name: "successful command with no run function",
			cmd: &cobra.Command{
				Use:   "test-no-run",
				Short: "A test command with no run function",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run(tt.cmd)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRun_Error(t *testing.T) {
	tests := []struct {
		name          string
		cmd           *cobra.Command
		expectedError string
		expectError   bool
	}{
		{
			name: "command execution returns error",
			cmd: &cobra.Command{
				Use: "test-error",
				RunE: func(cmd *cobra.Command, args []string) error {
					return errors.New("command failed")
				},
			},
			expectedError: "command failed",
			expectError:   true,
		},
		{
			name: "command execution returns formatted error",
			cmd: &cobra.Command{
				Use: "test-formatted-error",
				RunE: func(cmd *cobra.Command, args []string) error {
					return fmt.Errorf("operation failed with code %d", 42)
				},
			},
			expectedError: "operation failed with code 42",
			expectError:   true,
		},
		{
			name: "command with invalid arguments",
			cmd: &cobra.Command{
				Use:  "test-invalid-args",
				Args: cobra.ExactArgs(1), // Require exactly 1 argument
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			},
			expectError: true, // Will fail because no args provided
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run(tt.cmd)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRun_NilCommand(t *testing.T) {
	// Test what happens when a nil command is passed
	// This should panic or return an error, depending on cobra's behavior
	assert.Panics(t, func() {
		Run(nil)
	}, "Run should panic when given a nil command")
}

func TestRun_CommandWithSubcommands(t *testing.T) {
	// Create a root command with subcommands
	rootCmd := &cobra.Command{
		Use:   "root",
		Short: "Root command",
	}

	subCmd := &cobra.Command{
		Use:   "sub",
		Short: "Sub command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	rootCmd.AddCommand(subCmd)

	// Test executing the root command (should succeed even without a run function)
	err := Run(rootCmd)
	assert.NoError(t, err)
}

func TestRun_CommandWithFlags(t *testing.T) {
	executed := false

	cmd := &cobra.Command{
		Use:   "test-flags",
		Short: "Test command with flags",
		RunE: func(cmd *cobra.Command, args []string) error {
			executed = true
			// Verify flag was set
			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				return nil
			}
			return errors.New("verbose flag not set")
		},
	}

	cmd.Flags().Bool("verbose", false, "Enable verbose output")

	// Set the flag and arguments before execution
	cmd.SetArgs([]string{"--verbose"})

	err := Run(cmd)
	assert.NoError(t, err)
	assert.True(t, executed, "Command should have been executed")
}

func TestRun_CommandExecutionOrder(t *testing.T) {
	var executionOrder []string

	cmd := &cobra.Command{
		Use:   "test-order",
		Short: "Test execution order",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			executionOrder = append(executionOrder, "PersistentPreRunE")
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			executionOrder = append(executionOrder, "PreRunE")
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			executionOrder = append(executionOrder, "RunE")
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			executionOrder = append(executionOrder, "PostRunE")
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			executionOrder = append(executionOrder, "PersistentPostRunE")
			return nil
		},
	}

	err := Run(cmd)
	assert.NoError(t, err)

	expectedOrder := []string{
		"PersistentPreRunE",
		"PreRunE",
		"RunE",
		"PostRunE",
		"PersistentPostRunE",
	}
	assert.Equal(t, expectedOrder, executionOrder, "Command hooks should execute in correct order")
}

func TestRun_PreRunError(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test-prerun-error",
		Short: "Test command with PreRun error",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("pre-run failed")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			t.Error("RunE should not be called when PreRunE fails")
			return nil
		},
	}

	err := Run(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pre-run failed")
}

func TestRun_PostRunError(t *testing.T) {
	var executed bool

	cmd := &cobra.Command{
		Use:   "test-postrun-error",
		Short: "Test command with PostRun error",
		RunE: func(cmd *cobra.Command, args []string) error {
			executed = true
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("post-run failed")
		},
	}

	err := Run(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "post-run failed")
	assert.True(t, executed, "RunE should have been executed even though PostRunE failed")
}

func TestRun_ArgumentValidation(t *testing.T) {
	tests := []struct {
		name        string
		cmd         *cobra.Command
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name: "exact args validation success",
			cmd: &cobra.Command{
				Use:  "test-exact",
				Args: cobra.ExactArgs(2),
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			},
			args:        []string{"arg1", "arg2"},
			expectError: false,
		},
		{
			name: "exact args validation failure",
			cmd: &cobra.Command{
				Use:  "test-exact-fail",
				Args: cobra.ExactArgs(2),
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			},
			args:        []string{"arg1"},
			expectError: true,
			errorMsg:    "accepts 2 arg(s), received 1",
		},
		{
			name: "minimum args validation success",
			cmd: &cobra.Command{
				Use:  "test-min",
				Args: cobra.MinimumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			},
			args:        []string{"arg1", "arg2", "arg3"},
			expectError: false,
		},
		{
			name: "minimum args validation failure",
			cmd: &cobra.Command{
				Use:  "test-min-fail",
				Args: cobra.MinimumNArgs(2),
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			},
			args:        []string{"arg1"},
			expectError: true,
			errorMsg:    "requires at least 2 arg(s), only received 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cmd.SetArgs(tt.args)

			err := Run(tt.cmd)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRun_HelpCommand(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test-help",
		Short: "Test command for help",
		Long:  "This is a longer description of the test command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	// Test help flag
	cmd.SetArgs([]string{"--help"})

	err := Run(cmd)
	// Help command should execute successfully
	assert.NoError(t, err)
}

func TestRun_VersionCommand(t *testing.T) {
	cmd := &cobra.Command{
		Use:     "test-version",
		Short:   "Test command for version",
		Version: "1.0.0",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	// Test version flag
	cmd.SetArgs([]string{"--version"})

	err := Run(cmd)
	// Version command should execute successfully
	assert.NoError(t, err)
}

func TestRun_ComplexCommand(t *testing.T) {
	var (
		verbose bool
		output  string
		count   int
	)

	cmd := &cobra.Command{
		Use:   "complex",
		Short: "A complex command with multiple flags and args",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("no arguments provided")
			}
			if count < 0 {
				return errors.New("count must be non-negative")
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file")
	cmd.Flags().IntVarP(&count, "count", "c", 0, "Number of items")

	// Test successful execution
	cmd.SetArgs([]string{"--verbose", "--output", "test.txt", "--count", "5", "input.txt"})

	err := Run(cmd)
	assert.NoError(t, err)
	assert.True(t, verbose)
	assert.Equal(t, "test.txt", output)
	assert.Equal(t, 5, count)
}

func TestRun_ErrorTypes(t *testing.T) {
	tests := []struct {
		name      string
		cmd       *cobra.Command
		expectErr bool
		errType   string
	}{
		{
			name: "custom error type",
			cmd: &cobra.Command{
				Use: "custom-error",
				RunE: func(cmd *cobra.Command, args []string) error {
					return &CustomError{Message: "custom failure"}
				},
			},
			expectErr: true,
			errType:   "*cli.CustomError",
		},
		{
			name: "standard error",
			cmd: &cobra.Command{
				Use: "standard-error",
				RunE: func(cmd *cobra.Command, args []string) error {
					return errors.New("standard failure")
				},
			},
			expectErr: true,
			errType:   "*errors.errorString",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run(tt.cmd)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errType, fmt.Sprintf("%T", err))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// CustomError is a custom error type for testing
type CustomError struct {
	Message string
}

func (e *CustomError) Error() string {
	return e.Message
}

func TestRun_ReturnValueConsistency(t *testing.T) {
	// Test that Run consistently returns the same error as cmd.Execute()
	testError := errors.New("test error")

	cmd := &cobra.Command{
		Use: "consistency-test",
		RunE: func(cmd *cobra.Command, args []string) error {
			return testError
		},
	}

	// Call cmd.Execute() directly
	directErr := cmd.Execute()

	// Reset command state for second execution
	cmd.ResetCommands()
	cmd.ResetFlags()

	// Call through Run function
	runErr := Run(cmd)

	// Both should return the same error
	assert.Equal(t, directErr, runErr)
	assert.Same(t, testError, runErr)
}

func TestRun_FunctionSignature(t *testing.T) {
	// Test that Run function has the expected signature
	cmd := &cobra.Command{
		Use: "signature-test",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	// This test verifies the function can be called and returns an error
	result := Run(cmd)
	assert.Nil(t, result) // Should be nil for successful execution

	// Test the function signature is correct
	var runFunc func(*cobra.Command) error = Run
	assert.NotNil(t, runFunc)
}
