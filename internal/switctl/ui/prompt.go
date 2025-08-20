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

package ui

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// PromptInput prompts the user for input with validation and real-time feedback.
func (ui *TerminalUI) PromptInput(prompt string, validator interfaces.InputValidator) (string, error) {
	for {
		// Display prompt with icon
		fmt.Fprintf(ui.output, "%s %s: ",
			ui.style.Info.Sprint("?"),
			ui.style.Primary.Sprint(prompt))

		// Read user input
		input, readErr := ui.reader.ReadString('\n')

		// Handle read errors
		if readErr != nil {
			if readErr == io.EOF {
				// If we get EOF, check if we have any input
				input = strings.TrimSpace(input)
				if input == "" {
					// If no validator is provided, return empty string gracefully
					if validator == nil {
						return "", nil
					}
					return "", fmt.Errorf("input stream exhausted with no data available")
				}
				// We have some input even with EOF, continue to validate it
				// Note: We'll validate this input and if it fails, we'll return an error
				// since we can't prompt again with EOF
			} else {
				return "", fmt.Errorf("failed to read input: %w", readErr)
			}
		}

		input = strings.TrimSpace(input)

		// Skip validation if no validator provided
		if validator == nil {
			return input, nil
		}

		// Validate input
		if validationErr := validator(input); validationErr != nil {
			fmt.Fprintf(ui.output, "%s %s\n",
				ui.style.Error.Sprint("✗"),
				ui.style.Error.Sprint(validationErr.Error()))

			// If the input was read with EOF, we can't continue prompting for more input
			if readErr == io.EOF {
				return "", fmt.Errorf("input validation failed and no more input available: %w", validationErr)
			}
			continue
		}

		// Input is valid
		fmt.Fprintf(ui.output, "%s Input accepted\n",
			ui.style.Success.Sprint("✓"))
		return input, nil
	}
}

// PromptConfirm prompts the user for a yes/no confirmation.
func (ui *TerminalUI) PromptConfirm(message string, defaultValue bool) (bool, error) {
	prompt := fmt.Sprintf("%s [y/N]", message)
	if defaultValue {
		prompt = fmt.Sprintf("%s [Y/n]", message)
	}

	input, err := ui.PromptInput(prompt, func(input string) error {
		input = strings.ToLower(strings.TrimSpace(input))
		if input == "" || input == "y" || input == "yes" || input == "n" || input == "no" {
			return nil
		}
		return errors.New("please enter 'y' or 'n'")
	})
	if err != nil {
		return false, err
	}

	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return defaultValue, nil
	}

	return input == "y" || input == "yes", nil
}

// PromptSelect prompts the user to select from a list of options.
func (ui *TerminalUI) PromptSelect(prompt string, options []string, defaultIndex int) (int, string, error) {
	if len(options) == 0 {
		return 0, "", errors.New("options cannot be empty")
	}

	if defaultIndex < 0 || defaultIndex >= len(options) {
		defaultIndex = 0
	}

	// Display options
	fmt.Fprintln(ui.output, ui.style.Primary.Sprint(prompt))
	for i, option := range options {
		icon := "○"
		if i == defaultIndex {
			icon = "●"
		}
		fmt.Fprintf(ui.output, "  %s %d) %s\n",
			ui.style.Info.Sprint(icon),
			i+1,
			option)
	}

	// Prompt for selection
	validator := func(input string) error {
		if input == "" {
			return nil // Allow default
		}
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(options) {
			return fmt.Errorf("please enter a number between 1 and %d", len(options))
		}
		return nil
	}

	choice, err := ui.PromptInput(fmt.Sprintf("Select option (default: %d)", defaultIndex+1), validator)
	if err != nil {
		return 0, "", err
	}

	if choice == "" {
		return defaultIndex, options[defaultIndex], nil
	}

	num, _ := strconv.Atoi(choice)
	selectedIndex := num - 1
	return selectedIndex, options[selectedIndex], nil
}

// PromptMultiSelect prompts the user to select multiple options from a list.
func (ui *TerminalUI) PromptMultiSelect(prompt string, options []string) ([]int, []string, error) {
	if len(options) == 0 {
		return nil, nil, errors.New("options cannot be empty")
	}

	// Display options
	fmt.Fprintln(ui.output, ui.style.Primary.Sprint(prompt))
	fmt.Fprintln(ui.output, ui.style.Info.Sprint("(Enter comma-separated numbers, e.g., 1,3,5)"))

	for i, option := range options {
		fmt.Fprintf(ui.output, "  %s %d) %s\n",
			ui.style.Info.Sprint("○"),
			i+1,
			option)
	}

	// Prompt for selection
	validator := func(input string) error {
		if input == "" {
			return errors.New("please select at least one option")
		}

		parts := strings.Split(input, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			num, err := strconv.Atoi(part)
			if err != nil || num < 1 || num > len(options) {
				return fmt.Errorf("invalid selection '%s', please enter numbers between 1 and %d", part, len(options))
			}
		}
		return nil
	}

	choice, err := ui.PromptInput("Select options", validator)
	if err != nil {
		return nil, nil, err
	}

	// Parse selections
	parts := strings.Split(choice, ",")
	selectedIndices := make([]int, 0, len(parts))
	selectedOptions := make([]string, 0, len(parts))
	seen := make(map[int]bool)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		num, _ := strconv.Atoi(part)
		index := num - 1

		// Skip duplicates
		if seen[index] {
			continue
		}
		seen[index] = true

		selectedIndices = append(selectedIndices, index)
		selectedOptions = append(selectedOptions, options[index])
	}

	return selectedIndices, selectedOptions, nil
}

// PromptPassword prompts the user for a password (hidden input).
func (ui *TerminalUI) PromptPassword(prompt string) (string, error) {
	fmt.Fprintf(ui.output, "%s %s: ",
		ui.style.Info.Sprint("🔒"),
		ui.style.Primary.Sprint(prompt))

	// For now, we'll just read normally as implementing true password hiding
	// requires platform-specific terminal manipulation
	input, err := ui.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	return strings.TrimSpace(input), nil
}

// Common input validators that can be used with PromptInput

// ServiceNameValidator validates service names (lowercase, hyphens allowed).
var ServiceNameValidator interfaces.InputValidator = func(input string) error {
	if input == "" {
		return errors.New("service name cannot be empty")
	}

	if len(input) < 2 {
		return errors.New("service name must be at least 2 characters long")
	}

	if len(input) > 50 {
		return errors.New("service name cannot be longer than 50 characters")
	}

	matched, _ := regexp.MatchString(`^[a-z][a-z0-9-]*[a-z0-9]$`, input)
	if !matched {
		return errors.New("service name must start with a letter, contain only lowercase letters, numbers, and hyphens, and end with a letter or number")
	}

	// Check for consecutive hyphens
	if strings.Contains(input, "--") {
		return errors.New("service name cannot contain consecutive hyphens")
	}

	return nil
}

// PackageNameValidator validates Go package names.
var PackageNameValidator interfaces.InputValidator = func(input string) error {
	if input == "" {
		return errors.New("package name cannot be empty")
	}

	matched, _ := regexp.MatchString(`^[a-z][a-z0-9]*$`, input)
	if !matched {
		return errors.New("package name must contain only lowercase letters and numbers, starting with a letter")
	}

	return nil
}

// PortValidator validates port numbers (1024-65535).
var PortValidator interfaces.InputValidator = func(input string) error {
	if input == "" {
		return errors.New("port number cannot be empty")
	}

	port, err := strconv.Atoi(input)
	if err != nil {
		return errors.New("port must be a valid number")
	}

	if port < 1024 || port > 65535 {
		return errors.New("port number must be between 1024 and 65535")
	}

	return nil
}

// EmailValidator validates email addresses.
var EmailValidator interfaces.InputValidator = func(input string) error {
	if input == "" {
		return errors.New("email address cannot be empty")
	}

	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, input)
	if !matched {
		return errors.New("please enter a valid email address")
	}

	return nil
}

// URLValidator validates URLs.
var URLValidator interfaces.InputValidator = func(input string) error {
	if input == "" {
		return errors.New("URL cannot be empty")
	}

	matched, _ := regexp.MatchString(`^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}.*$`, input)
	if !matched {
		return errors.New("please enter a valid URL (http:// or https://)")
	}

	return nil
}

// ModulePathValidator validates Go module paths.
var ModulePathValidator interfaces.InputValidator = func(input string) error {
	if input == "" {
		return errors.New("module path cannot be empty")
	}

	// Basic validation for Go module paths
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._/@-]+$`, input)
	if !matched {
		return errors.New("module path contains invalid characters")
	}

	if strings.HasPrefix(input, "/") || strings.HasSuffix(input, "/") {
		return errors.New("module path cannot start or end with a slash")
	}

	return nil
}

// NonEmptyValidator validates that input is not empty.
var NonEmptyValidator interfaces.InputValidator = func(input string) error {
	if strings.TrimSpace(input) == "" {
		return errors.New("input cannot be empty")
	}
	return nil
}

// OptionalValidator always passes validation (for optional fields).
var OptionalValidator interfaces.InputValidator = func(input string) error {
	return nil
}

// CreateRangeValidator creates a validator for numeric ranges.
func CreateRangeValidator(min, max int) interfaces.InputValidator {
	return func(input string) error {
		if input == "" {
			return errors.New("value cannot be empty")
		}

		num, err := strconv.Atoi(input)
		if err != nil {
			return errors.New("value must be a valid number")
		}

		if num < min || num > max {
			return fmt.Errorf("value must be between %d and %d", min, max)
		}

		return nil
	}
}

// CreateLengthValidator creates a validator for string length.
func CreateLengthValidator(min, max int) interfaces.InputValidator {
	return func(input string) error {
		length := len(strings.TrimSpace(input))

		if length < min {
			return fmt.Errorf("input must be at least %d characters long", min)
		}

		if length > max {
			return fmt.Errorf("input cannot be longer than %d characters", max)
		}

		return nil
	}
}

// CreateRegexValidator creates a validator using a regular expression.
func CreateRegexValidator(pattern, message string) interfaces.InputValidator {
	regex := regexp.MustCompile(pattern)
	return func(input string) error {
		if !regex.MatchString(input) {
			return errors.New(message)
		}
		return nil
	}
}

// CreateChoiceValidator creates a validator for predefined choices.
func CreateChoiceValidator(choices []string, caseSensitive bool) interfaces.InputValidator {
	choiceMap := make(map[string]bool)
	for _, choice := range choices {
		key := choice
		if !caseSensitive {
			key = strings.ToLower(choice)
		}
		choiceMap[key] = true
	}

	return func(input string) error {
		key := input
		if !caseSensitive {
			key = strings.ToLower(input)
		}

		if !choiceMap[key] {
			return fmt.Errorf("value must be one of: %s", strings.Join(choices, ", "))
		}

		return nil
	}
}

// CombineValidators combines multiple validators into one.
func CombineValidators(validators ...interfaces.InputValidator) interfaces.InputValidator {
	return func(input string) error {
		for _, validator := range validators {
			if validator != nil {
				if err := validator(input); err != nil {
					return err
				}
			}
		}
		return nil
	}
}
