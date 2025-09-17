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

package generate

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/innovationmech/swit/internal/switctl/deps"
	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GenerateConfig holds common configuration for generation commands.
type GenerateConfig struct {
	WorkDir     string
	Service     string
	Package     string
	OutputDir   string
	DryRun      bool
	Verbose     bool
	Force       bool
	Interactive bool
}

// Copy creates a deep copy of the GenerateConfig to prevent mutations.
func (gc *GenerateConfig) Copy() *GenerateConfig {
	if gc == nil {
		return nil
	}
	return &GenerateConfig{
		WorkDir:     gc.WorkDir,
		Service:     gc.Service,
		Package:     gc.Package,
		OutputDir:   gc.OutputDir,
		DryRun:      gc.DryRun,
		Verbose:     gc.Verbose,
		Force:       gc.Force,
		Interactive: gc.Interactive,
	}
}

// DefaultGenerateConfig returns an immutable default configuration.
func DefaultGenerateConfig() *GenerateConfig {
	return &GenerateConfig{
		WorkDir:     "",
		Service:     "",
		Package:     "",
		OutputDir:   "",
		DryRun:      false,
		Verbose:     false,
		Force:       false,
		Interactive: false,
	}
}

// NewGenerateCommand creates the main generate command with subcommands.
func NewGenerateCommand() *cobra.Command {
	var config GenerateConfig

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate code and components",
		Long: `Generate API endpoints, data models, middleware, and other code components.

The generate command provides subcommands for creating various types of code:
- api: Generate API endpoints with HTTP and gRPC handlers
- model: Generate data models with CRUD operations
- middleware: Generate HTTP and gRPC middleware

All generation commands support dry-run mode to preview changes before applying them.`,
		Aliases: []string{"gen", "g"},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			validatedConfig, err := validateGenerateConfig(&config)
			if err != nil {
				return err
			}
			config = *validatedConfig
			return nil
		},
	}

	// Add persistent flags for all generate subcommands
	cmd.PersistentFlags().StringVarP(&config.Service, "service", "s", "", "Target service name")
	cmd.PersistentFlags().StringVarP(&config.Package, "package", "p", "", "Target package name")
	cmd.PersistentFlags().StringVarP(&config.OutputDir, "output", "o", "", "Output directory (defaults to current directory)")
	cmd.PersistentFlags().BoolVar(&config.DryRun, "dry-run", false, "Preview changes without applying them")
	cmd.PersistentFlags().BoolVarP(&config.Verbose, "verbose", "v", false, "Enable verbose output")
	cmd.PersistentFlags().BoolVar(&config.Force, "force", false, "Overwrite existing files")
	cmd.PersistentFlags().BoolVarP(&config.Interactive, "interactive", "i", false, "Use interactive mode")

	// Bind flags to viper
	viper.BindPFlag("generate.service", cmd.PersistentFlags().Lookup("service"))
	viper.BindPFlag("generate.package", cmd.PersistentFlags().Lookup("package"))
	viper.BindPFlag("generate.output", cmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("generate.dry-run", cmd.PersistentFlags().Lookup("dry-run"))
	viper.BindPFlag("generate.verbose", cmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("generate.force", cmd.PersistentFlags().Lookup("force"))
	viper.BindPFlag("generate.interactive", cmd.PersistentFlags().Lookup("interactive"))

	// Add subcommands
	cmd.AddCommand(NewAPICommand(&config))
	cmd.AddCommand(NewModelCommand(&config))
	cmd.AddCommand(NewMiddlewareCommand(&config))
	cmd.AddCommand(NewCapabilitiesCommand(&config))

	return cmd
}

// validateGenerateConfig validates the common generation configuration.
// Returns a validated copy instead of mutating the original.
func validateGenerateConfig(config *GenerateConfig) (*GenerateConfig, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Create a copy to avoid mutating the original
	validatedConfig := config.Copy()

	// Set working directory
	if validatedConfig.WorkDir == "" {
		wd, err := filepath.Abs(".")
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		validatedConfig.WorkDir = wd
	}

	// Validate and sanitize working directory
	if err := validateAndSanitizePath(validatedConfig.WorkDir); err != nil {
		return nil, fmt.Errorf("invalid working directory: %w", err)
	}

	// Set output directory
	if validatedConfig.OutputDir == "" {
		validatedConfig.OutputDir = validatedConfig.WorkDir
	} else if !filepath.IsAbs(validatedConfig.OutputDir) {
		validatedConfig.OutputDir = filepath.Join(validatedConfig.WorkDir, validatedConfig.OutputDir)
	}

	// Validate and sanitize output directory
	if err := validateAndSanitizePath(validatedConfig.OutputDir); err != nil {
		return nil, fmt.Errorf("invalid output directory: %w", err)
	}

	// Validate service name if provided
	if validatedConfig.Service != "" {
		if err := validateServiceName(validatedConfig.Service); err != nil {
			return nil, fmt.Errorf("invalid service name: %w", err)
		}
	}

	// Validate package name if provided
	if validatedConfig.Package != "" {
		if err := validatePackageName(validatedConfig.Package); err != nil {
			return nil, fmt.Errorf("invalid package name: %w", err)
		}
	}

	return validatedConfig, nil
}

// validateAndSanitizePath validates and sanitizes file paths to prevent path traversal attacks.
func validateAndSanitizePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Clean the path to resolve any . or .. elements
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}

	// Check for absolute paths outside allowed directories
	if filepath.IsAbs(cleanPath) {
		// For absolute paths, ensure they don't escape the project directory
		wd, err := filepath.Abs(".")
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Check if the path is within the working directory or temp directory
		if !strings.HasPrefix(cleanPath, wd) && !strings.HasPrefix(cleanPath, "/tmp") {
			return fmt.Errorf("absolute path outside allowed directories: %s", path)
		}
	}

	// Additional security checks
	if len(cleanPath) > 255 {
		return fmt.Errorf("path too long: %d characters (max 255)", len(cleanPath))
	}

	// Check for invalid characters (basic validation)
	if strings.ContainsAny(cleanPath, "<>:\"|?*") {
		return fmt.Errorf("path contains invalid characters: %s", path)
	}

	return nil
}

// Input validation using regex patterns for enhanced security
var (
	serviceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
	packageNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	identifierPattern  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// validateServiceName validates a service name with enhanced security checks.
func validateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	// Length validation
	if len(name) < 2 || len(name) > 50 {
		return fmt.Errorf("service name must be between 2 and 50 characters")
	}

	// Pattern validation
	if !serviceNamePattern.MatchString(name) {
		return fmt.Errorf("service name must start with a letter, contain only lowercase letters, numbers, and hyphens, and not end with a hyphen")
	}

	// Additional security checks
	if strings.Contains(name, "--") {
		return fmt.Errorf("service name cannot contain consecutive hyphens")
	}

	// Reserved name checks
	reservedNames := []string{"admin", "root", "system", "config", "internal", "api", "www"}
	for _, reserved := range reservedNames {
		if name == reserved {
			return fmt.Errorf("service name '%s' is reserved", name)
		}
	}

	return nil
}

// validatePackageName validates a package name with enhanced security checks.
func validatePackageName(name string) error {
	if name == "" {
		return fmt.Errorf("package name cannot be empty")
	}

	// Length validation
	if len(name) < 2 || len(name) > 30 {
		return fmt.Errorf("package name must be between 2 and 30 characters")
	}

	// Pattern validation
	if !packageNamePattern.MatchString(name) {
		return fmt.Errorf("package name must start with a letter and contain only lowercase letters and numbers")
	}

	// Reserved package names in Go
	reservedPackages := []string{
		"main", "test", "init", "go", "gofmt", "godoc", "govet",
		"builtin", "unsafe", "fmt", "os", "io", "net", "http",
	}
	for _, reserved := range reservedPackages {
		if name == reserved {
			return fmt.Errorf("package name '%s' is reserved in Go", name)
		}
	}

	return nil
}

// validateIdentifier validates a Go identifier (variable, function, type name).
func validateIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("identifier cannot be empty")
	}

	// Length validation
	if len(name) > 50 {
		return fmt.Errorf("identifier too long: %d characters (max 50)", len(name))
	}

	// Pattern validation
	if !identifierPattern.MatchString(name) {
		return fmt.Errorf("identifier must start with a letter or underscore and contain only letters, numbers, and underscores")
	}

	// Go reserved keywords
	reservedKeywords := []string{
		"break", "case", "chan", "const", "continue", "default", "defer",
		"else", "fallthrough", "for", "func", "go", "goto", "if",
		"import", "interface", "map", "package", "range", "return",
		"select", "struct", "switch", "type", "var",
	}
	for _, keyword := range reservedKeywords {
		if name == keyword {
			return fmt.Errorf("identifier '%s' is a reserved Go keyword", name)
		}
	}

	return nil
}

// GeneratorCommand represents a base generator command.
type GeneratorCommand struct {
	config     *GenerateConfig
	deps       interfaces.DependencyContainer
	ui         interfaces.InteractiveUI
	generator  interfaces.Generator
	fileSystem interfaces.FileSystem
	logger     interfaces.Logger
}

// NewGeneratorCommand creates a new base generator command.
func NewGeneratorCommand(config *GenerateConfig) (*GeneratorCommand, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Create a copy to prevent configuration mutation
	configCopy := config.Copy()

	// Initialize dependency container
	container := deps.NewContainer()

	// Get services from container with safe type assertions
	ui, err := getTypedService[interfaces.InteractiveUI](container, "ui")
	if err != nil {
		return nil, fmt.Errorf("failed to get UI service: %w", err)
	}

	generator, err := getTypedService[interfaces.Generator](container, "generator")
	if err != nil {
		return nil, fmt.Errorf("failed to get generator service: %w", err)
	}

	fileSystem, err := getTypedService[interfaces.FileSystem](container, "filesystem")
	if err != nil {
		return nil, fmt.Errorf("failed to get filesystem service: %w", err)
	}

	logger, err := getTypedService[interfaces.Logger](container, "logger")
	if err != nil {
		return nil, fmt.Errorf("failed to get logger service: %w", err)
	}

	return &GeneratorCommand{
		config:     configCopy,
		deps:       container,
		ui:         ui,
		generator:  generator,
		fileSystem: fileSystem,
		logger:     logger,
	}, nil
}

// getTypedService safely retrieves and type-checks a service from the container.
func getTypedService[T any](container interfaces.DependencyContainer, name string) (T, error) {
	var zero T
	service, err := container.GetService(name)
	if err != nil {
		return zero, err
	}

	typedService, ok := service.(T)
	if !ok {
		return zero, fmt.Errorf("service %s does not implement expected interface", name)
	}

	return typedService, nil
}

// Execute executes the generator command with common setup and validation.
func (gc *GeneratorCommand) Execute(ctx context.Context, generateFunc func(ctx context.Context) error) error {
	if gc.config.Verbose {
		gc.logger.Info("Starting code generation",
			"service", gc.config.Service,
			"package", gc.config.Package,
			"output", gc.config.OutputDir,
			"dry_run", gc.config.DryRun,
		)
	}

	// Show dry run warning if enabled
	if gc.config.DryRun {
		gc.ui.ShowInfo("Running in dry-run mode - no files will be modified")
		gc.ui.PrintSeparator()
	}

	// Execute the generation function
	if err := generateFunc(ctx); err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	// Show completion message
	if gc.config.DryRun {
		gc.ui.ShowSuccess("Dry run completed successfully")
	} else {
		gc.ui.ShowSuccess("Code generation completed successfully")
	}

	return nil
}

// PromptForMissingConfig prompts the user for missing configuration values.
func (gc *GeneratorCommand) PromptForMissingConfig(requiredFields map[string]string) error {
	if !gc.config.Interactive {
		// Check if all required fields are provided
		missing := []string{}
		for field, description := range requiredFields {
			switch field {
			case "service":
				if gc.config.Service == "" {
					missing = append(missing, description)
				}
			case "package":
				if gc.config.Package == "" {
					missing = append(missing, description)
				}
			}
		}

		if len(missing) > 0 {
			return fmt.Errorf("missing required configuration: %s. Use --interactive or provide via flags", strings.Join(missing, ", "))
		}
		return nil
	}

	// Prompt for missing service name
	if gc.config.Service == "" {
		if _, required := requiredFields["service"]; required {
			service, err := gc.ui.PromptInput("Service name", validateServiceName)
			if err != nil {
				return fmt.Errorf("failed to get service name: %w", err)
			}
			gc.config.Service = service
		}
	}

	// Prompt for missing package name
	if gc.config.Package == "" {
		if _, required := requiredFields["package"]; required {
			pkg, err := gc.ui.PromptInput("Package name", validatePackageName)
			if err != nil {
				return fmt.Errorf("failed to get package name: %w", err)
			}
			gc.config.Package = pkg
		}
	}

	return nil
}

// ShowPreview shows a preview of what will be generated.
func (gc *GeneratorCommand) ShowPreview(files []string, description string) error {
	gc.ui.PrintHeader("Generation Preview")
	gc.ui.ShowInfo(description)
	gc.ui.PrintSeparator()

	if len(files) == 0 {
		gc.ui.ShowInfo("No files will be generated")
		return nil
	}

	gc.ui.ShowInfo(fmt.Sprintf("Files to be generated (%d):", len(files)))
	for _, file := range files {
		// Make path relative to output directory for display
		relPath, err := filepath.Rel(gc.config.OutputDir, file)
		if err != nil {
			relPath = file
		}
		fmt.Printf("  • %s\n", relPath)
	}

	gc.ui.PrintSeparator()

	// Confirm if not in dry-run mode
	if !gc.config.DryRun && gc.config.Interactive {
		confirm, err := gc.ui.PromptConfirm("Proceed with generation?", true)
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirm {
			return fmt.Errorf("generation cancelled by user")
		}
	}

	return nil
}

// CheckFileConflicts checks for file conflicts and handles them according to configuration.
func (gc *GeneratorCommand) CheckFileConflicts(files []string) error {
	conflicts := []string{}
	for _, file := range files {
		if gc.fileSystem.Exists(file) {
			conflicts = append(conflicts, file)
		}
	}

	if len(conflicts) == 0 {
		return nil
	}

	if !gc.config.Force {
		gc.ui.ShowError(fmt.Errorf("file conflicts detected"))
		for _, file := range conflicts {
			relPath, err := filepath.Rel(gc.config.OutputDir, file)
			if err != nil {
				relPath = file
			}
			fmt.Printf("  • %s (already exists)\n", relPath)
		}

		if gc.config.Interactive {
			confirm, err := gc.ui.PromptConfirm("Overwrite existing files?", false)
			if err != nil {
				return fmt.Errorf("failed to get confirmation: %w", err)
			}
			if !confirm {
				return fmt.Errorf("generation cancelled due to file conflicts")
			}
		} else {
			return fmt.Errorf("file conflicts detected. Use --force to overwrite or --interactive to choose")
		}
	}

	return nil
}

// logGeneration logs the generation operation.
func (gc *GeneratorCommand) logGeneration(operation string, details map[string]interface{}) {
	if gc.config.Verbose {
		logData := []interface{}{
			"operation", operation,
			"service", gc.config.Service,
			"package", gc.config.Package,
			"output", gc.config.OutputDir,
		}

		for key, value := range details {
			logData = append(logData, key, value)
		}

		gc.logger.Info("Code generation step", logData...)
	}
}
