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
	"fmt"
	"strconv"
	"strings"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// Menu represents an interactive menu with keyboard navigation.
type Menu struct {
	title       string
	options     []interfaces.MenuOption
	selected    int
	ui          *TerminalUI
	showIcons   bool
	showNumbers bool
}

// MenuConfig configures menu behavior and appearance.
type MenuConfig struct {
	ShowIcons   bool
	ShowNumbers bool
	MultiSelect bool
}

// ShowMenu displays a menu and returns the selected option index.
func (ui *TerminalUI) ShowMenu(title string, options []interfaces.MenuOption) (int, error) {
	menu := &Menu{
		title:       title,
		options:     options,
		selected:    0,
		ui:          ui,
		showIcons:   true,
		showNumbers: true,
	}

	return menu.Show()
}

// Show displays the menu and handles user interaction.
func (m *Menu) Show() (int, error) {
	// Validate options
	if len(m.options) == 0 {
		return 0, fmt.Errorf("menu options cannot be empty")
	}

	// Find first enabled option
	for i, option := range m.options {
		if option.Enabled {
			m.selected = i
			break
		}
	}

	// Simple menu implementation using numbered selection
	return m.showSimpleMenu()
}

// showSimpleMenu shows a simple numbered menu (fallback for terminals without advanced input).
func (m *Menu) showSimpleMenu() (int, error) {
	// Display title
	m.ui.PrintHeader(m.title)

	// Display options
	for i, option := range m.options {
		icon := m.getOptionIcon(option, i == m.selected)
		number := ""
		if m.showNumbers {
			number = fmt.Sprintf("%d) ", i+1)
		}

		// Format option text
		optionText := option.Label
		if option.Description != "" {
			optionText = fmt.Sprintf("%s - %s", option.Label, option.Description)
		}

		// Apply styling based on enabled state
		if !option.Enabled {
			optionText = m.ui.style.Highlight.Sprint("(disabled) " + optionText)
		}

		fmt.Fprintf(m.ui.output, "  %s %s%s\n", icon, number, optionText)
	}

	fmt.Fprintln(m.ui.output)

	// Get user selection
	for {
		input, err := m.ui.PromptInput("Please select an option", func(input string) error {
			if input == "" {
				return fmt.Errorf("please enter a selection")
			}

			choice, err := strconv.Atoi(input)
			if err != nil {
				return fmt.Errorf("please enter a valid number")
			}

			if choice < 1 || choice > len(m.options) {
				return fmt.Errorf("please enter a number between 1 and %d", len(m.options))
			}

			// Check if option is enabled
			if !m.options[choice-1].Enabled {
				return fmt.Errorf("option %d is disabled, please select another option", choice)
			}

			return nil
		})
		if err != nil {
			return 0, err
		}

		choice, _ := strconv.Atoi(input)
		return choice - 1, nil
	}
}

// getOptionIcon returns the appropriate icon for an option.
func (m *Menu) getOptionIcon(option interfaces.MenuOption, selected bool) string {
	if !m.showIcons {
		return ""
	}

	// Use custom icon if provided
	if option.Icon != "" {
		return option.Icon
	}

	// Default icons based on selection and enabled state
	if !option.Enabled {
		return m.ui.style.Highlight.Sprint("⊘")
	}

	if selected {
		return m.ui.style.Primary.Sprint("▶")
	}

	return m.ui.style.Info.Sprint("○")
}

// MultiSelectMenu represents a menu that allows multiple selections.
type MultiSelectMenu struct {
	title     string
	options   []interfaces.MenuOption
	selected  map[int]bool
	ui        *TerminalUI
	showIcons bool
}

// ShowMultiSelectMenu displays a multi-select menu and returns selected option indices.
func (ui *TerminalUI) ShowMultiSelectMenu(title string, options []interfaces.MenuOption) ([]int, error) {
	menu := &MultiSelectMenu{
		title:     title,
		options:   options,
		selected:  make(map[int]bool),
		ui:        ui,
		showIcons: true,
	}

	return menu.Show()
}

// Show displays the multi-select menu and handles user interaction.
func (m *MultiSelectMenu) Show() ([]int, error) {
	if len(m.options) == 0 {
		return nil, fmt.Errorf("menu options cannot be empty")
	}

	// Display title
	m.ui.PrintHeader(m.title)

	// Display instructions
	fmt.Fprintln(m.ui.output, m.ui.style.Info.Sprint("Instructions:"))
	fmt.Fprintln(m.ui.output, "  • Enter comma-separated numbers to select multiple options")
	fmt.Fprintln(m.ui.output, "  • Example: 1,3,5 to select options 1, 3, and 5")
	fmt.Fprintln(m.ui.output, "  • Press Enter when done")
	fmt.Fprintln(m.ui.output)

	// Display options
	for i, option := range m.options {
		icon := m.getOptionIcon(option, false)

		optionText := fmt.Sprintf("%d) %s", i+1, option.Label)
		if option.Description != "" {
			optionText += fmt.Sprintf(" - %s", option.Description)
		}

		if !option.Enabled {
			optionText = m.ui.style.Highlight.Sprint("(disabled) " + optionText)
		}

		fmt.Fprintf(m.ui.output, "  %s %s\n", icon, optionText)
	}

	fmt.Fprintln(m.ui.output)

	// Get user selections
	input, err := m.ui.PromptInput("Select options", func(input string) error {
		if input == "" {
			return fmt.Errorf("please select at least one option")
		}

		parts := strings.Split(input, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			choice, err := strconv.Atoi(part)
			if err != nil {
				return fmt.Errorf("'%s' is not a valid number", part)
			}

			if choice < 1 || choice > len(m.options) {
				return fmt.Errorf("option %d is out of range (1-%d)", choice, len(m.options))
			}

			if !m.options[choice-1].Enabled {
				return fmt.Errorf("option %d is disabled", choice)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Parse selections
	parts := strings.Split(input, ",")
	selectedIndices := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		choice, _ := strconv.Atoi(part)
		index := choice - 1
		if !m.selected[index] {
			m.selected[index] = true
			selectedIndices = append(selectedIndices, index)
		}
	}

	return selectedIndices, nil
}

// getOptionIcon returns the appropriate icon for an option in multi-select mode.
func (m *MultiSelectMenu) getOptionIcon(option interfaces.MenuOption, selected bool) string {
	if !m.showIcons {
		return ""
	}

	if option.Icon != "" {
		return option.Icon
	}

	if !option.Enabled {
		return m.ui.style.Highlight.Sprint("⊘")
	}

	if selected {
		return m.ui.style.Success.Sprint("☑")
	}

	return m.ui.style.Info.Sprint("☐")
}

// ConfirmationMenu represents a simple yes/no confirmation menu.
type ConfirmationMenu struct {
	message      string
	defaultValue bool
	ui           *TerminalUI
}

// ShowConfirmationMenu displays a confirmation dialog.
func (ui *TerminalUI) ShowConfirmationMenu(message string, defaultValue bool) (bool, error) {
	menu := &ConfirmationMenu{
		message:      message,
		defaultValue: defaultValue,
		ui:           ui,
	}

	return menu.Show()
}

// Show displays the confirmation menu.
func (c *ConfirmationMenu) Show() (bool, error) {
	return c.ui.PromptConfirm(c.message, c.defaultValue)
}

// ProgressMenu represents a menu that shows progress for long-running operations.
type ProgressMenu struct {
	title    string
	steps    []string
	current  int
	ui       *TerminalUI
	progress interfaces.ProgressBar
}

// ShowProgressMenu displays a progress menu for multi-step operations.
func (ui *TerminalUI) ShowProgressMenu(title string, steps []string) *ProgressMenu {
	menu := &ProgressMenu{
		title:   title,
		steps:   steps,
		current: 0,
		ui:      ui,
	}

	menu.progress = ui.ShowProgress(title, len(steps))
	return menu
}

// NextStep advances to the next step in the progress menu.
func (p *ProgressMenu) NextStep() error {
	if p.current < len(p.steps) {
		p.current++
		if p.current <= len(p.steps) {
			message := ""
			if p.current < len(p.steps) {
				message = p.steps[p.current]
			}
			p.progress.SetMessage(message)
			return p.progress.Update(p.current)
		}
	}
	return nil
}

// Finish completes the progress menu.
func (p *ProgressMenu) Finish() error {
	return p.progress.Finish()
}

// SetStepMessage sets a custom message for the current step.
func (p *ProgressMenu) SetStepMessage(message string) error {
	return p.progress.SetMessage(message)
}

// BuildMenuOption creates a MenuOption with the given parameters.
func BuildMenuOption(label, description string, value interface{}, icon string, enabled bool) interfaces.MenuOption {
	return interfaces.MenuOption{
		Label:       label,
		Description: description,
		Value:       value,
		Icon:        icon,
		Enabled:     enabled,
	}
}

// BuildSimpleMenuOption creates a simple MenuOption with just label and value.
func BuildSimpleMenuOption(label string, value interface{}) interfaces.MenuOption {
	return interfaces.MenuOption{
		Label:   label,
		Value:   value,
		Enabled: true,
	}
}

// BuildMenuOptions creates a list of MenuOptions from labels and values.
func BuildMenuOptions(items map[string]interface{}) []interfaces.MenuOption {
	options := make([]interfaces.MenuOption, 0, len(items))
	for label, value := range items {
		options = append(options, BuildSimpleMenuOption(label, value))
	}
	return options
}

// BuildStringMenuOptions creates MenuOptions from a slice of strings.
func BuildStringMenuOptions(items []string) []interfaces.MenuOption {
	options := make([]interfaces.MenuOption, 0, len(items))
	for _, item := range items {
		options = append(options, BuildSimpleMenuOption(item, item))
	}
	return options
}

// Feature selection menu helpers for service generation

// ShowFeatureSelectionMenu displays a menu for selecting service features.
func (ui *TerminalUI) ShowFeatureSelectionMenu() (interfaces.ServiceFeatures, error) {
	features := interfaces.ServiceFeatures{}

	ui.PrintHeader("Service Features Selection")
	fmt.Fprintln(ui.output, ui.style.Info.Sprint("Select the features you want to include in your service:"))
	fmt.Fprintln(ui.output)

	// Database
	database, err := ui.PromptConfirm("Include database support", true)
	if err != nil {
		return features, err
	}
	features.Database = database

	// Authentication
	auth, err := ui.PromptConfirm("Include authentication support", false)
	if err != nil {
		return features, err
	}
	features.Authentication = auth

	// Cache
	cache, err := ui.PromptConfirm("Include cache support", false)
	if err != nil {
		return features, err
	}
	features.Cache = cache

	// Message Queue
	messageQueue, err := ui.PromptConfirm("Include message queue support", false)
	if err != nil {
		return features, err
	}
	features.MessageQueue = messageQueue

	// Monitoring
	monitoring, err := ui.PromptConfirm("Include monitoring support", true)
	if err != nil {
		return features, err
	}
	features.Monitoring = monitoring

	// Tracing
	tracing, err := ui.PromptConfirm("Include tracing support", true)
	if err != nil {
		return features, err
	}
	features.Tracing = tracing

	// Logging
	logging, err := ui.PromptConfirm("Include structured logging", true)
	if err != nil {
		return features, err
	}
	features.Logging = logging

	// Health Check
	healthCheck, err := ui.PromptConfirm("Include health check endpoints", true)
	if err != nil {
		return features, err
	}
	features.HealthCheck = healthCheck

	// Metrics
	metrics, err := ui.PromptConfirm("Include metrics collection", true)
	if err != nil {
		return features, err
	}
	features.Metrics = metrics

	// Docker
	docker, err := ui.PromptConfirm("Include Docker configuration", true)
	if err != nil {
		return features, err
	}
	features.Docker = docker

	// Kubernetes
	kubernetes, err := ui.PromptConfirm("Include Kubernetes configuration", false)
	if err != nil {
		return features, err
	}
	features.Kubernetes = kubernetes

	return features, nil
}

// ShowDatabaseSelectionMenu displays a menu for selecting database type.
func (ui *TerminalUI) ShowDatabaseSelectionMenu() (string, error) {
	options := []interfaces.MenuOption{
		BuildMenuOption("MySQL", "Popular relational database", "mysql", "🐬", true),
		BuildMenuOption("PostgreSQL", "Advanced relational database", "postgresql", "🐘", true),
		BuildMenuOption("MongoDB", "Document-oriented NoSQL database", "mongodb", "🍃", true),
		BuildMenuOption("Redis", "In-memory data structure store", "redis", "📦", true),
		BuildMenuOption("SQLite", "Lightweight embedded database", "sqlite", "📁", true),
	}

	selectedIndex, err := ui.ShowMenu("Select Database Type", options)
	if err != nil {
		return "", err
	}

	return options[selectedIndex].Value.(string), nil
}

// ShowAuthSelectionMenu displays a menu for selecting authentication type.
func (ui *TerminalUI) ShowAuthSelectionMenu() (string, error) {
	options := []interfaces.MenuOption{
		BuildMenuOption("JWT", "JSON Web Tokens", "jwt", "🔑", true),
		BuildMenuOption("OAuth2", "OAuth 2.0 protocol", "oauth2", "🛡️", true),
		BuildMenuOption("Basic Auth", "HTTP Basic Authentication", "basic", "🔐", true),
		BuildMenuOption("Session", "Session-based authentication", "session", "🍪", true),
	}

	selectedIndex, err := ui.ShowMenu("Select Authentication Type", options)
	if err != nil {
		return "", err
	}

	return options[selectedIndex].Value.(string), nil
}
