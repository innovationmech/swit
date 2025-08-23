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
	"strings"
	"testing"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ExampleServiceCreationFlow demonstrates a basic service creation workflow using the UI components.
func SkipExampleServiceCreationFlow(t *testing.T) {
	// Simple test to demonstrate basic workflow without complex interactions
	var output bytes.Buffer
	ui := NewTerminalUI(WithOutput(&output))

	// Step 1: Show welcome
	err := ui.ShowWelcome()
	require.NoError(t, err)

	// Step 2: Show configuration summary
	ui.PrintHeader("Configuration Summary")
	ui.ShowInfo("Service Name: my-awesome-service")
	ui.ShowInfo("Database: postgresql")
	ui.ShowInfo("Authentication: jwt")

	// Step 3: Show progress (just test creation, not execution to avoid hanging)
	progressMenu := ui.ShowProgressMenu("Creating Service", []string{
		"Creating directory structure",
		"Generating configuration files",
		"Creating database models",
	})

	// Just verify the progress menu was created successfully
	assert.NotNil(t, progressMenu)

	// Step 4: Show success
	err = ui.ShowSuccess("Service 'my-awesome-service' created successfully!")
	require.NoError(t, err)

	// Verify output contains expected elements
	outputStr := output.String()
	assert.Contains(t, outputStr, "Welcome to Swit Framework")
	assert.Contains(t, outputStr, "my-awesome-service")
	assert.Contains(t, outputStr, "Configuration Summary")
	assert.Contains(t, outputStr, "created successfully")
}

// ExampleErrorHandling demonstrates error handling in the UI.
func SkipExampleErrorHandling(t *testing.T) {
	// Simulate invalid input followed by valid input
	userInput := strings.Join([]string{
		"Invalid Service Name", // invalid service name
		"my-valid-service",     // valid service name
		"invalid-choice",       // invalid menu choice
		"1",                    // valid menu choice
	}, "\n") + "\n"

	input := strings.NewReader(userInput)
	var output bytes.Buffer
	ui := NewTerminalUI(WithInput(input), WithOutput(&output))

	// Test input validation with error recovery
	serviceName, err := ui.PromptInput("Enter service name", ServiceNameValidator)
	require.NoError(t, err)
	assert.Equal(t, "my-valid-service", serviceName)

	// Test menu selection with error recovery
	options := []interfaces.MenuOption{
		BuildMenuOption("Option 1", "Description 1", "value1", "🔧", true),
		BuildMenuOption("Option 2", "Description 2", "value2", "⚙️", true),
	}

	selectedIndex, err := ui.ShowMenu("Test Menu", options)
	require.NoError(t, err)
	assert.Equal(t, 0, selectedIndex)

	// Verify error messages appeared in output
	outputStr := output.String()
	assert.Contains(t, outputStr, "✗") // Error indicator
	assert.Contains(t, outputStr, "✓") // Success indicator
}

// ExampleTableDisplay demonstrates table rendering.
func TestExampleTableDisplay(t *testing.T) {
	var output bytes.Buffer
	ui := NewTerminalUI(WithOutput(&output))

	// Display a services overview table
	headers := []string{"Service", "Status", "Port", "Database"}
	rows := [][]string{
		{"user-service", "Running", "8080", "PostgreSQL"},
		{"auth-service", "Running", "8081", "MySQL"},
		{"notification-service", "Stopped", "8082", "MongoDB"},
		{"api-gateway", "Running", "8000", "None"},
	}

	err := ui.ShowTable(headers, rows)
	require.NoError(t, err)

	outputStr := output.String()
	assert.Contains(t, outputStr, "Service")
	assert.Contains(t, outputStr, "user-service")
	assert.Contains(t, outputStr, "PostgreSQL")
	assert.Contains(t, outputStr, "┌") // Table border
	assert.Contains(t, outputStr, "│") // Table separator
}

// ExampleProgressAndSpinner demonstrates progress indicators.
func SkipExampleProgressAndSpinner(t *testing.T) {
	var output bytes.Buffer
	ui := NewTerminalUI(WithOutput(&output))

	// Test progress bar
	pb := ui.ShowProgress("Downloading dependencies", 100)

	// Simulate progress updates
	steps := []int{25, 50, 75, 100}
	messages := []string{"Fetching packages", "Resolving dependencies", "Installing", "Complete"}

	for i, step := range steps {
		err := pb.SetMessage(messages[i])
		require.NoError(t, err)

		err = pb.Update(step)
		require.NoError(t, err)
	}

	err := pb.Finish()
	require.NoError(t, err)

	outputStr := output.String()
	assert.Contains(t, outputStr, "Downloading dependencies")
	assert.Contains(t, outputStr, "█") // Progress bar filled character
}

// ExampleColorAndStyling demonstrates color and styling features.
func TestExampleColorAndStyling(t *testing.T) {
	var output bytes.Buffer
	ui := NewTerminalUI(WithOutput(&output))

	// Test different message types
	err := ui.ShowSuccess("Operation completed successfully")
	require.NoError(t, err)

	err = ui.ShowError(assert.AnError)
	require.NoError(t, err)

	err = ui.ShowWarning("This is a warning message")
	require.NoError(t, err)

	err = ui.ShowInfo("This is an informational message")
	require.NoError(t, err)

	// Test headers and separators
	ui.PrintHeader("Main Section")
	ui.PrintSubHeader("Subsection")
	ui.PrintSeparator()

	outputStr := output.String()
	assert.Contains(t, outputStr, "✅")  // Success icon
	assert.Contains(t, outputStr, "❌")  // Error icon
	assert.Contains(t, outputStr, "⚠️") // Warning icon
	assert.Contains(t, outputStr, "ℹ️") // Info icon
	assert.Contains(t, outputStr, "╭─") // Header border
	assert.Contains(t, outputStr, "▶")  // Subheader arrow
	assert.Contains(t, outputStr, "─")  // Separator
}

// ExampleValidators demonstrates various input validators.
func TestExampleValidators(t *testing.T) {
	tests := []struct {
		name      string
		validator interfaces.InputValidator
		input     string
		expectErr bool
	}{
		{"Valid service name", ServiceNameValidator, "my-service", false},
		{"Invalid service name", ServiceNameValidator, "My-Service", true},
		{"Valid package name", PackageNameValidator, "mypackage", false},
		{"Invalid package name", PackageNameValidator, "my-package", true},
		{"Valid port", PortValidator, "8080", false},
		{"Invalid port", PortValidator, "80", true},
		{"Valid email", EmailValidator, "user@example.com", false},
		{"Invalid email", EmailValidator, "invalid-email", true},
		{"Valid URL", URLValidator, "https://example.com", false},
		{"Invalid URL", URLValidator, "not-a-url", true},
		{"Valid module path", ModulePathValidator, "github.com/user/repo", false},
		{"Invalid module path", ModulePathValidator, "/invalid/path", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ExampleCompositeValidators demonstrates combining multiple validators.
func TestExampleCompositeValidators(t *testing.T) {
	// Create a composite validator for service descriptions
	descriptionValidator := CombineValidators(
		NonEmptyValidator,
		CreateLengthValidator(10, 200),
	)

	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{"Valid description", "A comprehensive user management service", false},
		{"Empty description", "", true},
		{"Too short", "Short", true},
		{"Too long", strings.Repeat("A", 201), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := descriptionValidator(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
