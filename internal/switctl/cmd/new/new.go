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

// Package new provides commands for creating new projects and components.
package new

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/innovationmech/swit/internal/switctl/filesystem"
	"github.com/innovationmech/swit/internal/switctl/generator"
	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/innovationmech/swit/internal/switctl/template"
	"github.com/innovationmech/swit/internal/switctl/ui"
)

var (
	// Global flags for new commands
	interactive    bool
	outputDir      string
	configFile     string
	templateDir    string
	verbose        bool
	noColor        bool
	force          bool
	dryRun         bool
	skipValidation bool
)

// NewNewCommand creates the main 'new' command with subcommands.
func NewNewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create new projects and components",
		Long: `The new command creates new projects, services, and components using the Swit framework.

This command provides interactive and non-interactive modes for creating:
• Microservices with full framework integration
• API endpoints and handlers
• Data models and repositories
• Middleware components
• Complete project structures

Examples:
  # Interactive service creation
  switctl new service --interactive

  # Quick service creation with defaults
  switctl new service my-service

  # Create service with custom configuration
  switctl new service my-service --output-dir ./services --config custom.yaml

  # Dry run to preview what will be created
  switctl new service my-service --dry-run`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeNewCommandConfig(cmd)
		},
	}

	// Add persistent flags
	cmd.PersistentFlags().BoolVarP(&interactive, "interactive", "i", false, "Use interactive mode for configuration")
	cmd.PersistentFlags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory for generated files")
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Configuration file to use")
	cmd.PersistentFlags().StringVar(&templateDir, "template-dir", "", "Custom template directory")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	cmd.PersistentFlags().BoolVar(&force, "force", false, "Overwrite existing files without confirmation")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Preview what will be created without actually creating files")
	cmd.PersistentFlags().BoolVar(&skipValidation, "skip-validation", false, "Skip configuration validation")

	// Bind flags to viper
	viper.BindPFlag("new.interactive", cmd.PersistentFlags().Lookup("interactive"))
	viper.BindPFlag("new.output_dir", cmd.PersistentFlags().Lookup("output-dir"))
	viper.BindPFlag("new.config_file", cmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("new.template_dir", cmd.PersistentFlags().Lookup("template-dir"))
	viper.BindPFlag("new.verbose", cmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("new.no_color", cmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("new.force", cmd.PersistentFlags().Lookup("force"))
	viper.BindPFlag("new.dry_run", cmd.PersistentFlags().Lookup("dry-run"))
	viper.BindPFlag("new.skip_validation", cmd.PersistentFlags().Lookup("skip-validation"))

	// Add subcommands
	cmd.AddCommand(NewServiceCommand())
	// TODO: Add more subcommands as needed
	// cmd.AddCommand(NewProjectCommand())
	// cmd.AddCommand(NewAPICommand())
	// cmd.AddCommand(NewModelCommand())

	return cmd
}

// initializeNewCommandConfig initializes configuration for new commands.
func initializeNewCommandConfig(cmd *cobra.Command) error {
	// Get config values from viper (environment variables, config files, etc.)
	if viper.IsSet("new.interactive") {
		interactive = viper.GetBool("new.interactive")
	}
	if viper.IsSet("new.output_dir") {
		outputDir = viper.GetString("new.output_dir")
	}
	if viper.IsSet("new.config_file") {
		configFile = viper.GetString("new.config_file")
	}
	if viper.IsSet("new.template_dir") {
		templateDir = viper.GetString("new.template_dir")
	}
	if viper.IsSet("new.verbose") {
		verbose = viper.GetBool("new.verbose")
	}
	if viper.IsSet("new.no_color") {
		noColor = viper.GetBool("new.no_color")
	}
	if viper.IsSet("new.force") {
		force = viper.GetBool("new.force")
	}
	if viper.IsSet("new.dry_run") {
		dryRun = viper.GetBool("new.dry_run")
	}
	if viper.IsSet("new.skip_validation") {
		skipValidation = viper.GetBool("new.skip_validation")
	}

	// Set default template directory if not provided
	if templateDir == "" {
		// Try to find templates in the switctl binary directory
		execPath, err := os.Executable()
		if err == nil {
			templateDir = filepath.Join(filepath.Dir(execPath), "templates")
		}

		// Fallback to embedded templates or current directory
		if templateDir == "" || !dirExists(templateDir) {
			// Use embedded templates or fallback
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
			templateDir = filepath.Join(wd, "internal", "switctl", "template", "templates")
		}
	}

	// Validate template directory exists
	if !dirExists(templateDir) {
		return fmt.Errorf("template directory does not exist: %s", templateDir)
	}

	if verbose {
		fmt.Printf("Using template directory: %s\n", templateDir)
	}

	return nil
}

// createDependencyContainer creates and configures the dependency injection container for new commands.
var createDependencyContainer = func() (interfaces.DependencyContainer, error) {
	// Create a simple dependency container implementation
	container := &SimpleDependencyContainer{
		services:   make(map[string]interface{}),
		factories:  make(map[string]interfaces.ServiceFactory),
		singletons: make(map[string]bool),
	}

	// Register core services
	if err := registerCoreServices(container); err != nil {
		return nil, fmt.Errorf("failed to register core services: %w", err)
	}

	return container, nil
}

// registerCoreServices registers core services with the dependency container.
func registerCoreServices(container *SimpleDependencyContainer) error {
	// Register filesystem service
	if err := container.RegisterSingleton("filesystem", func() (interface{}, error) {
		return filesystem.NewOSFileSystem(""), nil
	}); err != nil {
		return fmt.Errorf("failed to register filesystem service: %w", err)
	}

	// Register template engine
	if err := container.RegisterSingleton("template_engine", func() (interface{}, error) {
		engine := template.NewEngine(templateDir)
		return engine, nil
	}); err != nil {
		return fmt.Errorf("failed to register template engine: %w", err)
	}

	// Register UI service
	if err := container.RegisterSingleton("ui", func() (interface{}, error) {
		return ui.NewTerminalUI(
			ui.WithVerbose(verbose),
			ui.WithNoColor(noColor),
		), nil
	}); err != nil {
		return fmt.Errorf("failed to register UI service: %w", err)
	}

	// Register logger service
	if err := container.RegisterSingleton("logger", func() (interface{}, error) {
		return NewSimpleLogger(verbose), nil
	}); err != nil {
		return fmt.Errorf("failed to register logger service: %w", err)
	}

	// Register service generator
	if err := container.RegisterSingleton("service_generator", func() (interface{}, error) {
		templateEngine, err := container.GetService("template_engine")
		if err != nil {
			return nil, err
		}

		fileSystem, err := container.GetService("filesystem")
		if err != nil {
			return nil, err
		}

		logger, err := container.GetService("logger")
		if err != nil {
			return nil, err
		}

		return generator.NewServiceGenerator(
			templateEngine.(interfaces.TemplateEngine),
			fileSystem.(interfaces.FileSystem),
			logger.(interfaces.Logger),
		), nil
	}); err != nil {
		return fmt.Errorf("failed to register service generator: %w", err)
	}

	return nil
}

// SimpleDependencyContainer provides a basic dependency injection container.
type SimpleDependencyContainer struct {
	mu         sync.RWMutex
	services   map[string]interface{}
	factories  map[string]interfaces.ServiceFactory
	singletons map[string]bool
}

// Register registers a service factory with the container.
func (c *SimpleDependencyContainer) Register(name string, factory interfaces.ServiceFactory) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if factory == nil {
		return fmt.Errorf("service factory cannot be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.factories[name] = factory
	c.singletons[name] = false
	return nil
}

// RegisterSingleton registers a singleton service factory with the container.
func (c *SimpleDependencyContainer) RegisterSingleton(name string, factory interfaces.ServiceFactory) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if factory == nil {
		return fmt.Errorf("service factory cannot be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.factories[name] = factory
	c.singletons[name] = true
	return nil
}

// GetService retrieves a service instance by name.
func (c *SimpleDependencyContainer) GetService(name string) (interface{}, error) {
	// First, check if we have a cached singleton instance (read lock)
	c.mu.RLock()
	if instance, exists := c.services[name]; exists && c.singletons[name] {
		c.mu.RUnlock()
		return instance, nil
	}

	// Get the factory while still holding read lock
	factory, exists := c.factories[name]
	isSingleton := c.singletons[name]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("service '%s' not found", name)
	}

	// Create new instance (no lock needed for factory call)
	instance, err := factory()
	if err != nil {
		return nil, fmt.Errorf("failed to create service '%s': %w", name, err)
	}

	// Cache singleton instances (write lock)
	if isSingleton {
		c.mu.Lock()
		// Double-check pattern: another goroutine might have created the instance
		if existingInstance, exists := c.services[name]; exists {
			c.mu.Unlock()
			return existingInstance, nil
		}
		c.services[name] = instance
		c.mu.Unlock()
	}

	return instance, nil
}

// HasService checks if a service is registered.
func (c *SimpleDependencyContainer) HasService(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.factories[name]
	return exists
}

// Close closes the container and cleans up resources.
func (c *SimpleDependencyContainer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clean up singleton instances if they implement io.Closer
	for name, instance := range c.services {
		if closer, ok := instance.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				// Log error but continue cleanup
				fmt.Printf("Warning: failed to close service '%s': %v\n", name, err)
			}
		}
	}

	// Clear all maps
	c.services = make(map[string]interface{})
	c.factories = make(map[string]interfaces.ServiceFactory)
	c.singletons = make(map[string]bool)

	return nil
}

// SimpleLogger provides a basic logger implementation.
type SimpleLogger struct {
	verbose bool
}

// NewSimpleLogger creates a new simple logger.
func NewSimpleLogger(verbose bool) *SimpleLogger {
	return &SimpleLogger{verbose: verbose}
}

// Debug logs a debug message.
func (l *SimpleLogger) Debug(msg string, fields ...interface{}) {
	if l.verbose {
		fmt.Printf("[DEBUG] %s %v\n", msg, fields)
	}
}

// Info logs an info message.
func (l *SimpleLogger) Info(msg string, fields ...interface{}) {
	fmt.Printf("[INFO] %s %v\n", msg, fields)
}

// Warn logs a warning message.
func (l *SimpleLogger) Warn(msg string, fields ...interface{}) {
	fmt.Printf("[WARN] %s %v\n", msg, fields)
}

// Error logs an error message.
func (l *SimpleLogger) Error(msg string, fields ...interface{}) {
	fmt.Printf("[ERROR] %s %v\n", msg, fields)
}

// Fatal logs a fatal message and exits.
func (l *SimpleLogger) Fatal(msg string, fields ...interface{}) {
	fmt.Printf("[FATAL] %s %v\n", msg, fields)
	os.Exit(1)
}

// ExecutionContext creates an execution context for command execution.
func createExecutionContext(ctx context.Context) *interfaces.ExecutionContext {
	wd, _ := os.Getwd()

	return &interfaces.ExecutionContext{
		WorkDir:    wd,
		ConfigFile: configFile,
		Verbose:    verbose,
		NoColor:    noColor,
		Options: map[string]interface{}{
			"output_dir":      outputDir,
			"template_dir":    templateDir,
			"force":           force,
			"dry_run":         dryRun,
			"skip_validation": skipValidation,
			"interactive":     interactive,
		},
		Logger:    NewSimpleLogger(verbose),
		StartTime: ctx.Value("start_time").(time.Time),
		Writer:    os.Stdout,
	}
}

// Utility functions

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// ensureDir creates a directory if it doesn't exist.
func ensureDir(path string) error {
	if !dirExists(path) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}
