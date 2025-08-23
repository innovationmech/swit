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

package ui

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptInputWithValidInput(t *testing.T) {
	input := strings.NewReader("valid-service-name\n")
	var output bytes.Buffer
	ui := NewTerminalUI(WithInput(input), WithOutput(&output))

	result, err := ui.PromptInput("Enter service name", ServiceNameValidator)
	require.NoError(t, err)
	assert.Equal(t, "valid-service-name", result)
}

func TestPromptInputWithInvalidThenValidInput(t *testing.T) {
	input := strings.NewReader("Invalid Name\nvalid-service-name\n")
	var output bytes.Buffer
	ui := NewTerminalUI(WithInput(input), WithOutput(&output))

	result, err := ui.PromptInput("Enter service name", ServiceNameValidator)
	require.NoError(t, err)
	assert.Equal(t, "valid-service-name", result)

	outputStr := output.String()
	assert.Contains(t, outputStr, "✗")
	assert.Contains(t, outputStr, "✓")
}

func TestPromptInputWithNoValidator(t *testing.T) {
	input := strings.NewReader("any input\n")
	var output bytes.Buffer
	ui := NewTerminalUI(WithInput(input), WithOutput(&output))

	result, err := ui.PromptInput("Enter anything", nil)
	require.NoError(t, err)
	assert.Equal(t, "any input", result)
}

func TestPromptConfirmWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue bool
		expected     bool
	}{
		{"Empty input with true default", "\n", true, true},
		{"Empty input with false default", "\n", false, false},
		{"Yes input", "y\n", false, true},
		{"Yes full input", "yes\n", false, true},
		{"No input", "n\n", true, false},
		{"No full input", "no\n", true, false},
		{"Case insensitive yes", "Y\n", false, true},
		{"Case insensitive no", "N\n", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			var output bytes.Buffer
			ui := NewTerminalUI(WithInput(input), WithOutput(&output))

			result, err := ui.PromptConfirm("Confirm action", tt.defaultValue)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPromptSelect(t *testing.T) {
	input := strings.NewReader("2\n")
	var output bytes.Buffer
	ui := NewTerminalUI(WithInput(input), WithOutput(&output))

	options := []string{"Option 1", "Option 2", "Option 3"}
	index, selection, err := ui.PromptSelect("Choose option", options, 0)

	require.NoError(t, err)
	assert.Equal(t, 1, index)
	assert.Equal(t, "Option 2", selection)
}

func TestPromptSelectWithDefault(t *testing.T) {
	input := strings.NewReader("\n")
	var output bytes.Buffer
	ui := NewTerminalUI(WithInput(input), WithOutput(&output))

	options := []string{"Option 1", "Option 2", "Option 3"}
	index, selection, err := ui.PromptSelect("Choose option", options, 1)

	require.NoError(t, err)
	assert.Equal(t, 1, index)
	assert.Equal(t, "Option 2", selection)
}

func TestPromptSelectInvalidInput(t *testing.T) {
	input := strings.NewReader("invalid\n2\n")
	var output bytes.Buffer
	ui := NewTerminalUI(WithInput(input), WithOutput(&output))

	options := []string{"Option 1", "Option 2"}
	index, selection, err := ui.PromptSelect("Choose option", options, 0)

	require.NoError(t, err)
	assert.Equal(t, 1, index)
	assert.Equal(t, "Option 2", selection)
}

func TestPromptMultiSelect(t *testing.T) {
	input := strings.NewReader("1,3\n")
	var output bytes.Buffer
	ui := NewTerminalUI(WithInput(input), WithOutput(&output))

	options := []string{"Option 1", "Option 2", "Option 3"}
	indices, selections, err := ui.PromptMultiSelect("Choose options", options)

	require.NoError(t, err)
	assert.Equal(t, []int{0, 2}, indices)
	assert.Equal(t, []string{"Option 1", "Option 3"}, selections)
}

func TestPromptMultiSelectInvalidInput(t *testing.T) {
	input := strings.NewReader("1,invalid,2\n1,2\n")
	var output bytes.Buffer
	ui := NewTerminalUI(WithInput(input), WithOutput(&output))

	options := []string{"Option 1", "Option 2", "Option 3"}
	indices, selections, err := ui.PromptMultiSelect("Choose options", options)

	require.NoError(t, err)
	assert.Equal(t, []int{0, 1}, indices)
	assert.Equal(t, []string{"Option 1", "Option 2"}, selections)
}

func TestPromptPassword(t *testing.T) {
	input := strings.NewReader("secret123\n")
	var output bytes.Buffer
	ui := NewTerminalUI(WithInput(input), WithOutput(&output))

	result, err := ui.PromptPassword("Enter password")
	require.NoError(t, err)
	assert.Equal(t, "secret123", result)
}

// Test validators

func TestServiceNameValidator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hasError bool
	}{
		{"Valid service name", "my-service", false},
		{"Single char too short", "a", true},
		{"Valid with numbers", "service123", false},
		{"Valid with hyphens", "my-awesome-service", false},
		{"Empty string", "", true},
		{"Too short removed", "ab", false}, // Two char is valid
		{"Too long", strings.Repeat("a", 51), true},
		{"Starts with number", "123service", true},
		{"Starts with hyphen", "-service", true},
		{"Ends with hyphen", "service-", true},
		{"Contains uppercase", "My-Service", true},
		{"Contains spaces", "my service", true},
		{"Contains underscores", "my_service", true},
		{"Consecutive hyphens", "my--service", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ServiceNameValidator(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPackageNameValidator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hasError bool
	}{
		{"Valid package name", "mypackage", false},
		{"Valid with numbers", "package123", false},
		{"Empty string", "", true},
		{"Starts with number", "123package", true},
		{"Contains hyphens", "my-package", true},
		{"Contains uppercase", "MyPackage", true},
		{"Contains spaces", "my package", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PackageNameValidator(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPortValidator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hasError bool
	}{
		{"Valid port", "8080", false},
		{"Valid high port", "65535", false},
		{"Valid low port", "1024", false},
		{"Empty string", "", true},
		{"Non-numeric", "abc", true},
		{"Too low", "1023", true},
		{"Too high", "65536", true},
		{"Zero", "0", true},
		{"Negative", "-1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PortValidator(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailValidator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hasError bool
	}{
		{"Valid email", "user@example.com", false},
		{"Valid with subdomain", "user@mail.example.com", false},
		{"Valid with numbers", "user123@example.com", false},
		{"Empty string", "", true},
		{"Missing @", "userexample.com", true},
		{"Missing domain", "user@", true},
		{"Missing TLD", "user@example", true},
		{"Invalid characters", "user@exam ple.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EmailValidator(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestURLValidator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hasError bool
	}{
		{"Valid HTTP URL", "http://example.com", false},
		{"Valid HTTPS URL", "https://example.com", false},
		{"Valid with path", "https://example.com/path", false},
		{"Empty string", "", true},
		{"Missing protocol", "example.com", true},
		{"Invalid protocol", "ftp://example.com", true},
		{"Missing domain", "https://", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := URLValidator(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateRangeValidator(t *testing.T) {
	validator := CreateRangeValidator(1, 10)

	assert.NoError(t, validator("5"))
	assert.NoError(t, validator("1"))
	assert.NoError(t, validator("10"))
	assert.Error(t, validator("0"))
	assert.Error(t, validator("11"))
	assert.Error(t, validator("abc"))
	assert.Error(t, validator(""))
}

func TestCreateLengthValidator(t *testing.T) {
	validator := CreateLengthValidator(3, 10)

	assert.NoError(t, validator("hello"))
	assert.NoError(t, validator("abc"))
	assert.NoError(t, validator("1234567890"))
	assert.Error(t, validator("ab"))
	assert.Error(t, validator("12345678901"))
}

func TestCreateChoiceValidator(t *testing.T) {
	choices := []string{"red", "green", "blue"}
	validator := CreateChoiceValidator(choices, false)

	assert.NoError(t, validator("red"))
	assert.NoError(t, validator("RED"))
	assert.NoError(t, validator("Green"))
	assert.Error(t, validator("yellow"))
	assert.Error(t, validator(""))
}

func TestCombineValidators(t *testing.T) {
	validator := CombineValidators(
		NonEmptyValidator,
		CreateLengthValidator(3, 10),
		PackageNameValidator,
	)

	assert.NoError(t, validator("hello"))
	assert.Error(t, validator(""))
	assert.Error(t, validator("ab"))
	assert.Error(t, validator("HELLO"))
}

// Additional comprehensive tests

func TestPromptInputIOError(t *testing.T) {
	t.Run("read error", func(t *testing.T) {
		// Use a reader that returns an error
		input := &errorReader{}
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output))

		_, err := ui.PromptInput("Test", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read input")
	})

	t.Run("EOF error", func(t *testing.T) {
		input := strings.NewReader("") // Empty reader causes EOF
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output))

		result, err := ui.PromptInput("Test", nil)
		assert.NoError(t, err) // EOF is handled gracefully
		assert.Equal(t, "", result)
	})
}

func TestPromptConfirmEdgeCases(t *testing.T) {
	t.Run("invalid input then valid", func(t *testing.T) {
		input := strings.NewReader("maybe\ninvalid\ny\n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output), WithNoColor(true))

		result, err := ui.PromptConfirm("Continue?", false)
		require.NoError(t, err)
		assert.True(t, result)

		outputStr := output.String()
		assert.Contains(t, outputStr, "please enter 'y' or 'n'")
	})

	t.Run("whitespace handling", func(t *testing.T) {
		input := strings.NewReader("  y  \n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output))

		result, err := ui.PromptConfirm("Continue?", false)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("case variations", func(t *testing.T) {
		tests := []struct {
			input    string
			expected bool
		}{
			{"YES\n", true},
			{"No\n", false},
			{"yEs\n", true},
			{"nO\n", false},
		}

		for _, tt := range tests {
			input := strings.NewReader(tt.input)
			var output bytes.Buffer
			ui := NewTerminalUI(WithInput(input), WithOutput(&output))

			result, err := ui.PromptConfirm("Test?", false)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		}
	})
}

func TestPromptSelectEdgeCases(t *testing.T) {
	t.Run("empty options", func(t *testing.T) {
		input := strings.NewReader("1\n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output))

		_, _, err := ui.PromptSelect("Choose", []string{}, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "options cannot be empty")
	})

	t.Run("out of range selection", func(t *testing.T) {
		input := strings.NewReader("0\n5\n2\n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output), WithNoColor(true))

		options := []string{"First", "Second"}
		idx, opt, err := ui.PromptSelect("Choose", options, 0)
		require.NoError(t, err)
		assert.Equal(t, 1, idx)
		assert.Equal(t, "Second", opt)

		outputStr := output.String()
		assert.Contains(t, outputStr, "between 1 and 2")
	})

	t.Run("default index adjustments", func(t *testing.T) {
		tests := []struct {
			name         string
			defaultIndex int
			expectedIcon string
		}{
			{"negative default", -1, "●"},
			{"out of range default", 10, "●"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := strings.NewReader("1\n")
				var output bytes.Buffer
				ui := NewTerminalUI(WithInput(input), WithOutput(&output), WithNoColor(true))

				options := []string{"First", "Second"}
				_, _, err := ui.PromptSelect("Choose", options, tt.defaultIndex)
				require.NoError(t, err)

				outputStr := output.String()
				assert.Contains(t, outputStr, tt.expectedIcon)
			})
		}
	})
}

func TestPromptMultiSelectEdgeCases(t *testing.T) {
	t.Run("empty options", func(t *testing.T) {
		input := strings.NewReader("1\n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output))

		_, _, err := ui.PromptMultiSelect("Choose", []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "options cannot be empty")
	})

	t.Run("empty input", func(t *testing.T) {
		input := strings.NewReader("\n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output), WithNoColor(true))

		options := []string{"First", "Second"}
		_, _, err := ui.PromptMultiSelect("Choose", options)
		assert.Error(t, err)

		outputStr := output.String()
		assert.Contains(t, outputStr, "select at least one")
	})

	t.Run("duplicate selections", func(t *testing.T) {
		input := strings.NewReader("1,1,2,2\n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output))

		options := []string{"First", "Second", "Third"}
		indices, selections, err := ui.PromptMultiSelect("Choose", options)
		require.NoError(t, err)
		assert.Equal(t, []int{0, 1}, indices) // Duplicates should be removed
		assert.Equal(t, []string{"First", "Second"}, selections)
	})

	t.Run("out of range selections", func(t *testing.T) {
		input := strings.NewReader("0,5\n1,2\n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output), WithNoColor(true))

		options := []string{"First", "Second"}
		indices, selections, err := ui.PromptMultiSelect("Choose", options)
		require.NoError(t, err)
		assert.Equal(t, []int{0, 1}, indices)
		assert.Equal(t, []string{"First", "Second"}, selections)

		outputStr := output.String()
		assert.Contains(t, outputStr, "invalid selection")
	})

	t.Run("non-numeric selections", func(t *testing.T) {
		input := strings.NewReader("1,abc,2\n1,2\n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output), WithNoColor(true))

		options := []string{"First", "Second", "Third"}
		indices, selections, err := ui.PromptMultiSelect("Choose", options)
		require.NoError(t, err)
		assert.Equal(t, []int{0, 1}, indices)
		assert.Equal(t, []string{"First", "Second"}, selections)

		outputStr := output.String()
		assert.Contains(t, outputStr, "invalid selection")
	})

	t.Run("whitespace in selections", func(t *testing.T) {
		input := strings.NewReader("  1  ,  2  ,  3  \n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output))

		options := []string{"First", "Second", "Third"}
		indices, selections, err := ui.PromptMultiSelect("Choose", options)
		require.NoError(t, err)
		assert.Equal(t, []int{0, 1, 2}, indices)
		assert.Equal(t, []string{"First", "Second", "Third"}, selections)
	})
}

func TestPromptPasswordEdgeCases(t *testing.T) {
	t.Run("empty password", func(t *testing.T) {
		input := strings.NewReader("\n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output), WithNoColor(true))

		result, err := ui.PromptPassword("Password")
		require.NoError(t, err)
		assert.Equal(t, "", result)

		outputStr := output.String()
		assert.Contains(t, outputStr, "🔒")
	})

	t.Run("password with special characters", func(t *testing.T) {
		input := strings.NewReader("p@ssw0rd!@#$%^&*()\n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output))

		result, err := ui.PromptPassword("Password")
		require.NoError(t, err)
		assert.Equal(t, "p@ssw0rd!@#$%^&*()", result)
	})

	t.Run("password with leading/trailing spaces", func(t *testing.T) {
		input := strings.NewReader("  secret  \n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output))

		result, err := ui.PromptPassword("Password")
		require.NoError(t, err)
		assert.Equal(t, "secret", result) // Spaces should be trimmed
	})

	t.Run("read error", func(t *testing.T) {
		input := &errorReader{}
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output))

		_, err := ui.PromptPassword("Password")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read password")
	})
}

// Test comprehensive validator coverage
func TestModulePathValidator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hasError bool
	}{
		{"Valid GitHub module", "github.com/user/repo", false},
		{"Valid domain module", "example.com/package", false},
		{"Valid local module", "local/package", false},
		{"Empty string", "", true},
		{"Starts with slash", "/invalid/path", true},
		{"Ends with slash", "invalid/path/", true},
		{"Contains spaces", "invalid path", true},
		{"Contains invalid chars", "invalid@path", false}, // @ is allowed in module paths
		{"Single component", "package", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ModulePathValidator(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNonEmptyValidator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hasError bool
	}{
		{"Valid input", "hello", false},
		{"Input with spaces", "  hello  ", false},
		{"Empty string", "", true},
		{"Only spaces", "   ", true},
		{"Only tabs", "\t\t", true},
		{"Only newlines", "\n\n", true},
		{"Mixed whitespace", " \t\n ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NonEmptyValidator(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOptionalValidator(t *testing.T) {
	// OptionalValidator should never return an error
	inputs := []string{"", "anything", "   ", "123", "special!@#$%"}
	for _, input := range inputs {
		t.Run("input: "+input, func(t *testing.T) {
			err := OptionalValidator(input)
			assert.NoError(t, err)
		})
	}
}

func TestValidatorFactoriesComprehensive(t *testing.T) {
	t.Run("CreateRegexValidator with complex pattern", func(t *testing.T) {
		// Email-like pattern
		validator := CreateRegexValidator(
			`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
			"must be a valid email",
		)

		tests := []struct {
			input   string
			wantErr bool
		}{
			{"user@example.com", false},
			{"test.email+tag@domain.co.uk", false},
			{"invalid-email", true},
			{"user@", true},
			{"", true},
		}

		for _, tt := range tests {
			err := validator(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "must be a valid email")
			} else {
				assert.NoError(t, err)
			}
		}
	})

	t.Run("CreateChoiceValidator edge cases", func(t *testing.T) {
		// Empty choices list
		validator := CreateChoiceValidator([]string{}, true)
		err := validator("anything")
		assert.Error(t, err)

		// Single choice
		validator = CreateChoiceValidator([]string{"only"}, true)
		assert.NoError(t, validator("only"))
		assert.Error(t, validator("other"))
	})

	t.Run("CombineValidators with all nil", func(t *testing.T) {
		validator := CombineValidators(nil, nil, nil)
		err := validator("test")
		assert.NoError(t, err)
	})

	t.Run("CombineValidators early failure", func(t *testing.T) {
		validator1 := func(input string) error {
			return errors.New("first error")
		}
		validator2 := func(input string) error {
			return errors.New("second error")
		}

		combined := CombineValidators(validator1, validator2)
		err := combined("test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "first error")
	})
}

// Test output formatting
func TestPromptOutputFormatting(t *testing.T) {
	t.Run("PromptInput output contains prompt icon", func(t *testing.T) {
		input := strings.NewReader("test\n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output), WithNoColor(true))

		ui.PromptInput("Test Prompt", nil)
		outputStr := output.String()
		assert.Contains(t, outputStr, "?")
		assert.Contains(t, outputStr, "Test Prompt")
	})

	t.Run("PromptConfirm shows correct brackets", func(t *testing.T) {
		tests := []struct {
			defaultValue bool
			expectedText string
		}{
			{true, "[Y/n]"},
			{false, "[y/N]"},
		}

		for _, tt := range tests {
			input := strings.NewReader("y\n")
			var output bytes.Buffer
			ui := NewTerminalUI(WithInput(input), WithOutput(&output), WithNoColor(true))

			ui.PromptConfirm("Test?", tt.defaultValue)
			outputStr := output.String()
			assert.Contains(t, outputStr, tt.expectedText)
		}
	})

	t.Run("PromptSelect shows option numbers and icons", func(t *testing.T) {
		input := strings.NewReader("1\n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output), WithNoColor(true))

		options := []string{"First", "Second", "Third"}
		ui.PromptSelect("Choose", options, 1)

		outputStr := output.String()
		assert.Contains(t, outputStr, "1) First")
		assert.Contains(t, outputStr, "2) Second")
		assert.Contains(t, outputStr, "3) Third")
		assert.Contains(t, outputStr, "●") // Default selection icon
		assert.Contains(t, outputStr, "○") // Non-selected icon
	})

	t.Run("PromptMultiSelect shows instructions", func(t *testing.T) {
		input := strings.NewReader("1\n")
		var output bytes.Buffer
		ui := NewTerminalUI(WithInput(input), WithOutput(&output), WithNoColor(true))

		options := []string{"First", "Second"}
		ui.PromptMultiSelect("Choose", options)

		outputStr := output.String()
		assert.Contains(t, outputStr, "comma-separated")
		assert.Contains(t, outputStr, "1,3,5")
	})
}

// Helper type for testing error conditions
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

// Benchmark tests for validators
func BenchmarkServiceNameValidator(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ServiceNameValidator("valid-service-name")
	}
}

func BenchmarkEmailValidator(b *testing.B) {
	for i := 0; i < b.N; i++ {
		EmailValidator("user@example.com")
	}
}

func BenchmarkPortValidator(b *testing.B) {
	for i := 0; i < b.N; i++ {
		PortValidator("8080")
	}
}

func BenchmarkCombinedValidators(b *testing.B) {
	validator := CombineValidators(
		NonEmptyValidator,
		CreateLengthValidator(2, 50),
		ServiceNameValidator,
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator("valid-service")
	}
}
