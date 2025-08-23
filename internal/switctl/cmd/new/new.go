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

// NewCommandConfig holds configuration for new commands
type NewCommandConfig struct {
	Interactive    bool
	OutputDir      string
	ConfigFile     string
	TemplateDir    string
	Verbose        bool
	NoColor        bool
	Force          bool
	DryRun         bool
	SkipValidation bool
}

// NewNewCommand creates the main 'new' command with subcommands.
func NewNewCommand() *cobra.Command {
	config := &NewCommandConfig{}

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
			return initializeNewCommandConfig(cmd, config)
		},
	}

	// Add persistent flags
	cmd.PersistentFlags().BoolVarP(&config.Interactive, "interactive", "i", false, "Use interactive mode for configuration")
	cmd.PersistentFlags().StringVarP(&config.OutputDir, "output-dir", "o", "", "Output directory for generated files")
	cmd.PersistentFlags().StringVarP(&config.ConfigFile, "config", "c", "", "Configuration file to use")
	cmd.PersistentFlags().StringVar(&config.TemplateDir, "template-dir", "", "Custom template directory")
	cmd.PersistentFlags().BoolVarP(&config.Verbose, "verbose", "v", false, "Enable verbose output")
	cmd.PersistentFlags().BoolVar(&config.NoColor, "no-color", false, "Disable colored output")
	cmd.PersistentFlags().BoolVar(&config.Force, "force", false, "Overwrite existing files without confirmation")
	cmd.PersistentFlags().BoolVar(&config.DryRun, "dry-run", false, "Preview what will be created without actually creating files")
	cmd.PersistentFlags().BoolVar(&config.SkipValidation, "skip-validation", false, "Skip configuration validation")

	// Create a new viper instance for this command to avoid global state conflicts
	v := viper.New()

	// Bind flags to viper
	v.BindPFlag("new.interactive", cmd.PersistentFlags().Lookup("interactive"))
	v.BindPFlag("new.output_dir", cmd.PersistentFlags().Lookup("output-dir"))
	v.BindPFlag("new.config_file", cmd.PersistentFlags().Lookup("config"))
	v.BindPFlag("new.template_dir", cmd.PersistentFlags().Lookup("template-dir"))
	v.BindPFlag("new.verbose", cmd.PersistentFlags().Lookup("verbose"))
	v.BindPFlag("new.no_color", cmd.PersistentFlags().Lookup("no-color"))
	v.BindPFlag("new.force", cmd.PersistentFlags().Lookup("force"))
	v.BindPFlag("new.dry_run", cmd.PersistentFlags().Lookup("dry-run"))
	v.BindPFlag("new.skip_validation", cmd.PersistentFlags().Lookup("skip-validation"))

	// Store the viper instance and config in command context
	ctx := context.WithValue(context.Background(), "viper", v)
	ctx = context.WithValue(ctx, "config", config)
	cmd.SetContext(ctx)

	// Add subcommands
	cmd.AddCommand(NewServiceCommand(config))
	// TODO: Add more subcommands as needed
	// cmd.AddCommand(NewProjectCommand())
	// cmd.AddCommand(NewAPICommand())
	// cmd.AddCommand(NewModelCommand())

	return cmd
}

// initializeNewCommandConfig initializes configuration for new commands.
func initializeNewCommandConfig(cmd *cobra.Command, config *NewCommandConfig) error {
	// Get viper instance from context
	v, ok := cmd.Context().Value("viper").(*viper.Viper)
	if !ok {
		v = viper.GetViper() // fallback to global viper
	}

	// Get config values from viper (environment variables, config files, etc.)
	if v.IsSet("new.interactive") {
		config.Interactive = v.GetBool("new.interactive")
	}
	if v.IsSet("new.output_dir") {
		config.OutputDir = v.GetString("new.output_dir")
	}
	if v.IsSet("new.config_file") {
		config.ConfigFile = v.GetString("new.config_file")
	}
	if v.IsSet("new.template_dir") {
		config.TemplateDir = v.GetString("new.template_dir")
	}
	if v.IsSet("new.verbose") {
		config.Verbose = v.GetBool("new.verbose")
	}
	if v.IsSet("new.no_color") {
		config.NoColor = v.GetBool("new.no_color")
	}
	if v.IsSet("new.force") {
		config.Force = v.GetBool("new.force")
	}
	if v.IsSet("new.dry_run") {
		config.DryRun = v.GetBool("new.dry_run")
	}
	if v.IsSet("new.skip_validation") {
		config.SkipValidation = v.GetBool("new.skip_validation")
	}

	// Set default template directory if not provided
	if config.TemplateDir == "" {
		// Try to find templates in the switctl binary directory
		execPath, err := os.Executable()
		if err == nil {
			config.TemplateDir = filepath.Join(filepath.Dir(execPath), "templates")
		}

		// Fallback to embedded templates or current directory
		if config.TemplateDir == "" || !dirExists(config.TemplateDir) {
			// Use embedded templates or fallback
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
			config.TemplateDir = filepath.Join(wd, "internal", "switctl", "template", "templates")
		}
	}

	// Validate template directory exists
	if !dirExists(config.TemplateDir) {
		return fmt.Errorf("template directory does not exist: %s", config.TemplateDir)
	}

	if config.Verbose {
		fmt.Printf("Using template directory: %s\n", config.TemplateDir)
	}

	return nil
}

// createDependencyContainer creates and configures the dependency injection container for new commands.
var createDependencyContainer = func(config *NewCommandConfig) (interfaces.DependencyContainer, error) {
	// Create a simple dependency container implementation
	container := &SimpleDependencyContainer{
		services:   make(map[string]interface{}),
		factories:  make(map[string]interfaces.ServiceFactory),
		singletons: make(map[string]bool),
	}

	// Register core services
	if err := registerCoreServices(container, config); err != nil {
		return nil, fmt.Errorf("failed to register core services: %w", err)
	}

	return container, nil
}

// registerCoreServices registers core services with the dependency container.
func registerCoreServices(container *SimpleDependencyContainer, config *NewCommandConfig) error {
	// Register filesystem service
	if err := container.RegisterSingleton("filesystem", func() (interface{}, error) {
		return filesystem.NewOSFileSystem(""), nil
	}); err != nil {
		return fmt.Errorf("failed to register filesystem service: %w", err)
	}

	// Register template engine
	if err := container.RegisterSingleton("template_engine", func() (interface{}, error) {
		engine := template.NewEngine(config.TemplateDir)
		return engine, nil
	}); err != nil {
		return fmt.Errorf("failed to register template engine: %w", err)
	}

	// Register UI service
	if err := container.RegisterSingleton("ui", func() (interface{}, error) {
		return ui.NewTerminalUI(
			ui.WithVerbose(config.Verbose),
			ui.WithNoColor(config.NoColor),
		), nil
	}); err != nil {
		return fmt.Errorf("failed to register UI service: %w", err)
	}

	// Register logger service
	if err := container.RegisterSingleton("logger", func() (interface{}, error) {
		return NewSimpleLogger(config.Verbose), nil
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
func createExecutionContext(ctx context.Context, config *NewCommandConfig) *interfaces.ExecutionContext {
	wd, _ := os.Getwd()

	return &interfaces.ExecutionContext{
		WorkDir:    wd,
		ConfigFile: config.ConfigFile,
		Verbose:    config.Verbose,
		NoColor:    config.NoColor,
		Options: map[string]interface{}{
			"output_dir":      config.OutputDir,
			"template_dir":    config.TemplateDir,
			"force":           config.Force,
			"dry_run":         config.DryRun,
			"skip_validation": config.SkipValidation,
			"interactive":     config.Interactive,
		},
		Logger:    NewSimpleLogger(config.Verbose),
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
