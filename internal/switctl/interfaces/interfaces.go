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

// Package interfaces defines the core interfaces and contracts for switctl.
package interfaces

import (
	"context"
	"io"
	"time"
)

// Command represents a switctl command that can be executed.
type Command interface {
	// Execute runs the command with the given context and arguments.
	Execute(ctx context.Context, args []string) error
	// GetHelp returns the help text for this command.
	GetHelp() string
	// GetUsage returns the usage text for this command.
	GetUsage() string
}

// InteractiveUI provides interactive user interface capabilities.
type InteractiveUI interface {
	// ShowWelcome displays the welcome screen.
	ShowWelcome() error
	// PromptInput prompts the user for input with validation.
	PromptInput(prompt string, validator InputValidator) (string, error)
	// ShowMenu displays a menu and returns the selected option index.
	ShowMenu(title string, options []MenuOption) (int, error)
	// ShowProgress creates and returns a progress bar.
	ShowProgress(title string, total int) ProgressBar
	// ShowSuccess displays a success message.
	ShowSuccess(message string) error
	// ShowError displays an error message.
	ShowError(err error) error
	// ShowTable displays tabular data.
	ShowTable(headers []string, rows [][]string) error
	// PromptConfirm prompts the user for yes/no confirmation.
	PromptConfirm(prompt string, defaultValue bool) (bool, error)
	// ShowInfo displays an informational message.
	ShowInfo(message string) error
	// PrintSeparator prints a visual separator line.
	PrintSeparator()
	// PrintHeader prints a formatted header.
	PrintHeader(title string)
	// PrintSubHeader prints a formatted subheader.
	PrintSubHeader(title string)
	// GetStyle returns the UI styling configuration.
	GetStyle() *UIStyle
	// PromptPassword prompts for password input (hidden).
	PromptPassword(prompt string) (string, error)
	// ShowFeatureSelectionMenu shows a feature selection menu.
	ShowFeatureSelectionMenu() (ServiceFeatures, error)
	// ShowDatabaseSelectionMenu shows a database selection menu.
	ShowDatabaseSelectionMenu() (string, error)
	// ShowAuthSelectionMenu shows an authentication selection menu.
	ShowAuthSelectionMenu() (string, error)
	// ShowMultiSelectMenu displays a multi-select menu and returns selected option indices.
	ShowMultiSelectMenu(title string, options []MenuOption) ([]int, error)
}

// Generator provides code generation capabilities.
type Generator interface {
	// GenerateService generates a new service based on the configuration.
	GenerateService(config ServiceConfig) error
	// GenerateAPI generates API endpoints and related code.
	GenerateAPI(config APIConfig) error
	// GenerateModel generates data models and CRUD operations.
	GenerateModel(config ModelConfig) error
	// GenerateMiddleware generates middleware templates.
	GenerateMiddleware(config MiddlewareConfig) error
}

// TemplateEngine provides template processing capabilities.
type TemplateEngine interface {
	// LoadTemplate loads a template by name.
	LoadTemplate(name string) (Template, error)
	// RenderTemplate renders a template with the provided data.
	RenderTemplate(template Template, data interface{}) ([]byte, error)
	// RegisterFunction registers a custom template function.
	RegisterFunction(name string, fn interface{}) error
	// SetTemplateDir sets the template directory path.
	SetTemplateDir(dir string) error
}

// Template represents a loaded template.
type Template interface {
	// Name returns the template name.
	Name() string
	// Render renders the template with the provided data.
	Render(data interface{}) ([]byte, error)
}

// QualityChecker provides code quality checking capabilities.
type QualityChecker interface {
	// CheckCodeStyle runs code style checks.
	CheckCodeStyle() CheckResult
	// RunTests executes tests and returns results.
	RunTests() TestResult
	// CheckSecurity runs security vulnerability scans.
	CheckSecurity() SecurityResult
	// CheckCoverage analyzes test coverage.
	CheckCoverage() CoverageResult
	// ValidateConfig validates configuration files.
	ValidateConfig() ValidationResult
	// CheckInterfaces validates service interface implementations.
	CheckInterfaces() CheckResult
	// CheckImports validates import organization.
	CheckImports() CheckResult
	// CheckCopyright validates copyright headers.
	CheckCopyright() CheckResult
}

// Plugin represents a loadable plugin.
type Plugin interface {
	// Name returns the plugin name.
	Name() string
	// Version returns the plugin version.
	Version() string
	// Initialize initializes the plugin with configuration.
	Initialize(config PluginConfig) error
	// Execute runs the plugin with the given context and arguments.
	Execute(ctx context.Context, args []string) error
	// Cleanup performs plugin cleanup.
	Cleanup() error
}

// PluginManager manages plugin loading and execution.
type PluginManager interface {
	// LoadPlugins loads all available plugins.
	LoadPlugins() error
	// GetPlugin retrieves a plugin by name.
	GetPlugin(name string) (Plugin, bool)
	// ListPlugins returns all loaded plugin names.
	ListPlugins() []string
	// RegisterPlugin registers a plugin.
	RegisterPlugin(plugin Plugin) error
	// UnloadPlugin unloads a plugin by name.
	UnloadPlugin(name string) error
}

// DependencyContainer provides dependency injection capabilities.
type DependencyContainer interface {
	// Register registers a service with the container.
	Register(name string, factory ServiceFactory) error
	// RegisterSingleton registers a singleton service.
	RegisterSingleton(name string, factory ServiceFactory) error
	// GetService retrieves a service by name.
	GetService(name string) (interface{}, error)
	// HasService checks if a service is registered.
	HasService(name string) bool
	// Close closes the container and cleans up resources.
	Close() error
}

// ConfigManager handles configuration loading and management.
type ConfigManager interface {
	// Load loads configuration from multiple sources.
	Load() error
	// Get retrieves a configuration value by key.
	Get(key string) interface{}
	// GetString retrieves a string configuration value.
	GetString(key string) string
	// GetInt retrieves an integer configuration value.
	GetInt(key string) int
	// GetBool retrieves a boolean configuration value.
	GetBool(key string) bool
	// Set sets a configuration value.
	Set(key string, value interface{})
	// Validate validates the current configuration.
	Validate() error
	// Save saves the configuration to file.
	Save(path string) error
}

// FileSystem provides file system operations abstraction.
type FileSystem interface {
	// WriteFile writes content to a file.
	WriteFile(path string, content []byte, perm int) error
	// ReadFile reads content from a file.
	ReadFile(path string) ([]byte, error)
	// MkdirAll creates directories recursively.
	MkdirAll(path string, perm int) error
	// Exists checks if a path exists.
	Exists(path string) bool
	// Remove removes a file or directory.
	Remove(path string) error
	// Copy copies a file or directory.
	Copy(src, dst string) error
}

// ProgressBar represents a progress indicator.
type ProgressBar interface {
	// Update updates the progress bar with the current value.
	Update(current int) error
	// SetMessage sets the progress message.
	SetMessage(message string) error
	// Finish completes the progress bar.
	Finish() error
	// SetTotal sets the total value.
	SetTotal(total int) error
}

// MenuOption represents a menu option.
type MenuOption struct {
	Label       string      // Display label
	Description string      // Option description
	Value       interface{} // Associated value
	Icon        string      // Optional icon
	Enabled     bool        // Whether the option is enabled
}

// InputValidator is a function that validates user input.
type InputValidator func(input string) error

// ServiceFactory is a function that creates service instances.
type ServiceFactory func() (interface{}, error)

// UIStyle represents user interface styling options.
type UIStyle struct {
	Primary   Color // Primary color
	Success   Color // Success color
	Warning   Color // Warning color
	Error     Color // Error color
	Info      Color // Info color
	Highlight Color // Highlight color
}

// Color represents a color value.
type Color interface {
	// Sprint colorizes text.
	Sprint(a ...interface{}) string
	// Sprintf colorizes formatted text.
	Sprintf(format string, a ...interface{}) string
}

// ProjectManager handles project-level operations.
type ProjectManager interface {
	// InitProject initializes a new project.
	InitProject(config ProjectConfig) error
	// ValidateProject validates project structure.
	ValidateProject() error
	// GetProjectInfo returns project information.
	GetProjectInfo() (ProjectInfo, error)
	// UpdateProject updates project configuration.
	UpdateProject(updates map[string]interface{}) error
}

// DependencyManager manages project dependencies.
type DependencyManager interface {
	// UpdateDependencies updates project dependencies.
	UpdateDependencies() error
	// AuditDependencies checks for security vulnerabilities.
	AuditDependencies() AuditResult
	// ListDependencies lists all project dependencies.
	ListDependencies() ([]Dependency, error)
	// AddDependency adds a new dependency.
	AddDependency(dep Dependency) error
	// RemoveDependency removes a dependency.
	RemoveDependency(name string) error
}

// VersionManager handles version compatibility and migration.
type VersionManager interface {
	// CheckCompatibility checks framework version compatibility.
	CheckCompatibility() CompatibilityResult
	// GetCurrentVersion returns the current framework version.
	GetCurrentVersion() (string, error)
	// GetLatestVersion returns the latest available version.
	GetLatestVersion() (string, error)
	// MigrateProject migrates project to newer version.
	MigrateProject(targetVersion string) error
}

// Logger provides logging capabilities.
type Logger interface {
	// Debug logs a debug message.
	Debug(msg string, fields ...interface{})
	// Info logs an info message.
	Info(msg string, fields ...interface{})
	// Warn logs a warning message.
	Warn(msg string, fields ...interface{})
	// Error logs an error message.
	Error(msg string, fields ...interface{})
	// Fatal logs a fatal message and exits.
	Fatal(msg string, fields ...interface{})
}

// ExecutionContext provides context for command execution.
type ExecutionContext struct {
	WorkDir    string                 // Working directory
	ConfigFile string                 // Configuration file path
	Verbose    bool                   // Verbose output enabled
	NoColor    bool                   // Color output disabled
	Options    map[string]interface{} // Additional options
	Logger     Logger                 // Logger instance
	StartTime  time.Time              // Execution start time
	Writer     io.Writer              // Output writer
}
