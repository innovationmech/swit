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

package new

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/innovationmech/swit/internal/switctl/ui"
)

var (
	// Service command specific flags
	serviceName    string
	serviceDesc    string
	serviceAuthor  string
	serviceVersion string
	modulePathFlag string
	httpPort       int
	grpcPort       int
	fromConfig     string
	quickMode      bool
)

// NewServiceCommand creates the 'new service' command.
func NewServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service [name]",
		Short: "Create a new microservice",
		Long: `Create a new microservice with the Swit framework structure and conventions.

This command generates a complete microservice project with:
• Standard Swit framework project layout
• HTTP and gRPC transport layer setup  
• Configuration management with validation
• Database integration (optional)
• Authentication middleware (optional)
• Docker and Kubernetes deployment files (optional)
• Comprehensive test suite structure
• Protocol Buffer definitions

The command supports both interactive and non-interactive modes:

Interactive Mode:
  switctl new service --interactive
  
  Guides you through all configuration options with beautiful menus and validation.

Non-Interactive Mode:
  switctl new service my-service
  
  Creates a service with sensible defaults. You can customize with flags:
  
  switctl new service my-service \
    --description "My awesome service" \
    --author "John Doe <john@example.com>" \
    --http-port 8080 \
    --grpc-port 9090 \
    --module-path github.com/myorg/my-service

Quick Mode:
  switctl new service my-service --quick
  
  Creates a minimal service with basic features only.

Configuration File:
  switctl new service --from-config service-config.yaml
  
  Loads configuration from a YAML file.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set service name from argument if provided
			if len(args) > 0 {
				serviceName = args[0]
			}

			// Use interactive mode if explicitly requested or no service name provided
			if interactive || (len(args) == 0 && fromConfig == "") {
				return runInteractiveServiceCreation(cmd.Context())
			}

			// Use configuration file if specified
			if fromConfig != "" {
				return runServiceCreationFromConfig(cmd.Context(), fromConfig)
			}

			// Use non-interactive mode with provided arguments
			return runNonInteractiveServiceCreation(cmd.Context())
		},
	}

	// Add service-specific flags
	cmd.Flags().StringVarP(&serviceName, "name", "n", "", "Service name (lowercase, hyphen-separated)")
	cmd.Flags().StringVarP(&serviceDesc, "description", "d", "", "Service description")
	cmd.Flags().StringVarP(&serviceAuthor, "author", "a", "", "Service author (Name <email@example.com>)")
	cmd.Flags().StringVar(&serviceVersion, "version", "0.1.0", "Initial service version")
	cmd.Flags().StringVarP(&modulePathFlag, "module-path", "m", "", "Go module path (e.g., github.com/org/service)")
	cmd.Flags().IntVar(&httpPort, "http-port", 9000, "HTTP server port")
	cmd.Flags().IntVar(&grpcPort, "grpc-port", 10000, "gRPC server port")
	cmd.Flags().StringVar(&fromConfig, "from-config", "", "Load configuration from YAML file")
	cmd.Flags().BoolVar(&quickMode, "quick", false, "Create service with minimal features")

	return cmd
}

// runInteractiveServiceCreation runs the interactive service creation flow.
func runInteractiveServiceCreation(ctx context.Context) error {
	// Create dependency container
	container, err := createDependencyContainer()
	if err != nil {
		return fmt.Errorf("failed to create dependency container: %w", err)
	}
	defer container.Close()

	// Get UI service
	uiService, err := container.GetService("ui")
	if err != nil {
		return fmt.Errorf("failed to get UI service: %w", err)
	}
	terminalUI := uiService.(interfaces.InteractiveUI)

	// Get logger
	loggerService, err := container.GetService("logger")
	if err != nil {
		return fmt.Errorf("failed to get logger service: %w", err)
	}
	logger := loggerService.(interfaces.Logger)

	logger.Info("Starting interactive service creation")

	// Show welcome message
	if err := terminalUI.ShowWelcome(); err != nil {
		return fmt.Errorf("failed to show welcome: %w", err)
	}

	// Create service configuration through interactive prompts
	config, err := collectServiceConfigurationInteractively(terminalUI)
	if err != nil {
		return fmt.Errorf("failed to collect service configuration: %w", err)
	}

	// Show configuration preview
	if err := showConfigurationPreview(terminalUI, config); err != nil {
		return fmt.Errorf("failed to show configuration preview: %w", err)
	}

	// Confirm before generation
	confirmed, err := terminalUI.PromptConfirm("Do you want to proceed with service generation?", true)
	if err != nil {
		return fmt.Errorf("failed to get confirmation: %w", err)
	}

	if !confirmed {
		terminalUI.ShowInfo("Service generation cancelled.")
		return nil
	}

	// Generate the service
	return generateService(ctx, container, config)
}

// runNonInteractiveServiceCreation runs non-interactive service creation with flags/defaults.
func runNonInteractiveServiceCreation(ctx context.Context) error {
	// Create dependency container
	container, err := createDependencyContainer()
	if err != nil {
		return fmt.Errorf("failed to create dependency container: %w", err)
	}
	defer container.Close()

	// Get logger
	loggerService, err := container.GetService("logger")
	if err != nil {
		return fmt.Errorf("failed to get logger service: %w", err)
	}
	logger := loggerService.(interfaces.Logger)

	// Validate required parameters
	if serviceName == "" {
		return fmt.Errorf("service name is required. Use --name flag or run in interactive mode")
	}

	// Build configuration from flags and defaults
	config, err := buildServiceConfigFromFlags()
	if err != nil {
		return fmt.Errorf("failed to build service configuration: %w", err)
	}

	logger.Info("Creating service with non-interactive mode", "service", config.Name)

	// Generate the service
	return generateService(ctx, container, config)
}

// runServiceCreationFromConfig loads configuration from a file and creates the service.
func runServiceCreationFromConfig(ctx context.Context, configPath string) error {
	// Create dependency container
	container, err := createDependencyContainer()
	if err != nil {
		return fmt.Errorf("failed to create dependency container: %w", err)
	}
	defer container.Close()

	// Get logger
	loggerService, err := container.GetService("logger")
	if err != nil {
		return fmt.Errorf("failed to get logger service: %w", err)
	}
	logger := loggerService.(interfaces.Logger)

	// Load configuration from file
	config, err := loadServiceConfigFromFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration from file: %w", err)
	}

	logger.Info("Creating service from configuration file", "config", configPath, "service", config.Name)

	// Generate the service
	return generateService(ctx, container, config)
}

// collectServiceConfigurationInteractively collects service configuration through interactive prompts.
func collectServiceConfigurationInteractively(terminalUI interfaces.InteractiveUI) (*interfaces.ServiceConfig, error) {
	config := &interfaces.ServiceConfig{
		Version:  "0.1.0",
		Metadata: make(map[string]string),
	}

	terminalUI.PrintHeader("Service Configuration")

	// Basic Information
	terminalUI.PrintSubHeader("Basic Information")

	name, err := terminalUI.PromptInput("Service name", ui.ServiceNameValidator)
	if err != nil {
		return nil, err
	}
	config.Name = name

	description, err := terminalUI.PromptInput("Service description (optional)", ui.OptionalValidator)
	if err != nil {
		return nil, err
	}
	config.Description = description

	author, err := terminalUI.PromptInput("Author (optional)", ui.OptionalValidator)
	if err != nil {
		return nil, err
	}
	config.Author = author

	version, err := terminalUI.PromptInput("Version", func(input string) error {
		if input == "" {
			return nil // Use default
		}
		// Basic version validation (could be enhanced)
		if len(input) == 0 {
			return fmt.Errorf("version cannot be empty")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if version != "" {
		config.Version = version
	}

	// Module Path
	modulePath, err := terminalUI.PromptInput("Go module path", ui.ModulePathValidator)
	if err != nil {
		return nil, err
	}
	config.ModulePath = modulePath

	// Output Directory
	outputDirInput, err := terminalUI.PromptInput("Output directory (default: current directory)", ui.OptionalValidator)
	if err != nil {
		return nil, err
	}
	if outputDirInput != "" {
		config.OutputDir = outputDirInput
	} else {
		config.OutputDir = config.Name
	}

	// Port Configuration
	terminalUI.PrintSubHeader("Port Configuration")

	httpPortStr, err := terminalUI.PromptInput("HTTP port", ui.CreateRangeValidator(1024, 65535))
	if err != nil {
		return nil, err
	}
	if httpPortStr != "" {
		config.Ports.HTTP, _ = strconv.Atoi(httpPortStr)
	} else {
		config.Ports.HTTP = 9000
	}

	grpcPortStr, err := terminalUI.PromptInput("gRPC port", ui.CreateRangeValidator(1024, 65535))
	if err != nil {
		return nil, err
	}
	if grpcPortStr != "" {
		config.Ports.GRPC, _ = strconv.Atoi(grpcPortStr)
	} else {
		config.Ports.GRPC = 10000
	}

	// Feature Selection
	features, err := terminalUI.ShowFeatureSelectionMenu()
	if err != nil {
		return nil, err
	}
	config.Features = features

	// Database Configuration (if database feature is enabled)
	if config.Features.Database {
		terminalUI.PrintSubHeader("Database Configuration")

		dbType, err := terminalUI.ShowDatabaseSelectionMenu()
		if err != nil {
			return nil, err
		}
		config.Database.Type = dbType

		dbHost, err := terminalUI.PromptInput("Database host", ui.NonEmptyValidator)
		if err != nil {
			return nil, err
		}
		config.Database.Host = dbHost

		// Set default port based on database type
		defaultPort := getDefaultDatabasePort(dbType)
		dbPortStr, err := terminalUI.PromptInput(
			fmt.Sprintf("Database port (default: %d)", defaultPort),
			ui.CreateRangeValidator(1, 65535),
		)
		if err != nil {
			return nil, err
		}
		if dbPortStr != "" {
			config.Database.Port, _ = strconv.Atoi(dbPortStr)
		} else {
			config.Database.Port = defaultPort
		}

		dbName, err := terminalUI.PromptInput("Database name", ui.NonEmptyValidator)
		if err != nil {
			return nil, err
		}
		config.Database.Database = dbName

		dbUser, err := terminalUI.PromptInput("Database username", ui.NonEmptyValidator)
		if err != nil {
			return nil, err
		}
		config.Database.Username = dbUser

		dbPassword, err := terminalUI.PromptPassword("Database password")
		if err != nil {
			return nil, err
		}
		config.Database.Password = dbPassword
	}

	// Authentication Configuration (if auth feature is enabled)
	if config.Features.Authentication {
		terminalUI.PrintSubHeader("Authentication Configuration")

		authType, err := terminalUI.ShowAuthSelectionMenu()
		if err != nil {
			return nil, err
		}
		config.Auth.Type = authType

		if authType == "jwt" {
			secretKey, err := terminalUI.PromptInput("JWT secret key", ui.NonEmptyValidator)
			if err != nil {
				return nil, err
			}
			config.Auth.SecretKey = secretKey

			issuer, err := terminalUI.PromptInput("JWT issuer (optional)", ui.OptionalValidator)
			if err != nil {
				return nil, err
			}
			config.Auth.Issuer = issuer

			// Set default expiration
			config.Auth.Expiration = 15 * time.Minute
		}
	}

	return config, nil
}

// showConfigurationPreview displays a preview of the service configuration.
func showConfigurationPreview(terminalUI interfaces.InteractiveUI, config *interfaces.ServiceConfig) error {
	terminalUI.PrintHeader("Configuration Preview")

	// Basic Information
	fmt.Printf("Service Name: %s\n", terminalUI.GetStyle().Primary.Sprint(config.Name))
	if config.Description != "" {
		fmt.Printf("Description: %s\n", config.Description)
	}
	if config.Author != "" {
		fmt.Printf("Author: %s\n", config.Author)
	}
	fmt.Printf("Version: %s\n", config.Version)
	fmt.Printf("Module Path: %s\n", config.ModulePath)
	fmt.Printf("Output Directory: %s\n", config.OutputDir)
	fmt.Println()

	// Ports
	fmt.Printf("HTTP Port: %s\n", terminalUI.GetStyle().Info.Sprint(strconv.Itoa(config.Ports.HTTP)))
	fmt.Printf("gRPC Port: %s\n", terminalUI.GetStyle().Info.Sprint(strconv.Itoa(config.Ports.GRPC)))
	fmt.Println()

	// Features
	fmt.Printf("Features:\n")
	showFeatureStatus(terminalUI, "Database", config.Features.Database)
	showFeatureStatus(terminalUI, "Authentication", config.Features.Authentication)
	showFeatureStatus(terminalUI, "Cache", config.Features.Cache)
	showFeatureStatus(terminalUI, "Message Queue", config.Features.MessageQueue)
	showFeatureStatus(terminalUI, "Monitoring", config.Features.Monitoring)
	showFeatureStatus(terminalUI, "Tracing", config.Features.Tracing)
	showFeatureStatus(terminalUI, "Logging", config.Features.Logging)
	showFeatureStatus(terminalUI, "Health Check", config.Features.HealthCheck)
	showFeatureStatus(terminalUI, "Metrics", config.Features.Metrics)
	showFeatureStatus(terminalUI, "Docker", config.Features.Docker)
	showFeatureStatus(terminalUI, "Kubernetes", config.Features.Kubernetes)
	fmt.Println()

	// Database Configuration
	if config.Features.Database {
		fmt.Printf("Database Configuration:\n")
		fmt.Printf("  Type: %s\n", config.Database.Type)
		fmt.Printf("  Host: %s\n", config.Database.Host)
		fmt.Printf("  Port: %d\n", config.Database.Port)
		fmt.Printf("  Database: %s\n", config.Database.Database)
		fmt.Printf("  Username: %s\n", config.Database.Username)
		fmt.Println()
	}

	// Authentication Configuration
	if config.Features.Authentication {
		fmt.Printf("Authentication Configuration:\n")
		fmt.Printf("  Type: %s\n", config.Auth.Type)
		if config.Auth.Issuer != "" {
			fmt.Printf("  Issuer: %s\n", config.Auth.Issuer)
		}
		fmt.Println()
	}

	return nil
}

// showFeatureStatus displays a feature status with colored icon.
func showFeatureStatus(terminalUI interfaces.InteractiveUI, name string, enabled bool) {
	icon := "✗"
	color := terminalUI.GetStyle().Error
	if enabled {
		icon = "✓"
		color = terminalUI.GetStyle().Success
	}
	fmt.Printf("  %s %s\n", color.Sprint(icon), name)
}

// buildServiceConfigFromFlags builds service configuration from command line flags.
func buildServiceConfigFromFlags() (*interfaces.ServiceConfig, error) {
	config := &interfaces.ServiceConfig{
		Name:        serviceName,
		Description: serviceDesc,
		Author:      serviceAuthor,
		Version:     serviceVersion,
		ModulePath:  modulePathFlag,
		OutputDir:   outputDir,
		Metadata:    make(map[string]string),
		Ports: interfaces.PortConfig{
			HTTP: httpPort,
			GRPC: grpcPort,
		},
	}

	// Apply quick mode defaults
	if quickMode {
		config.Features = interfaces.ServiceFeatures{
			Database:       false,
			Authentication: false,
			Cache:          false,
			MessageQueue:   false,
			Monitoring:     true,
			Tracing:        false,
			Logging:        true,
			HealthCheck:    true,
			Metrics:        false,
			Docker:         true,
			Kubernetes:     false,
		}
	} else {
		// Default features
		config.Features = interfaces.ServiceFeatures{
			Database:       true,
			Authentication: false,
			Cache:          false,
			MessageQueue:   false,
			Monitoring:     true,
			Tracing:        true,
			Logging:        true,
			HealthCheck:    true,
			Metrics:        true,
			Docker:         true,
			Kubernetes:     false,
		}
	}

	// Set output directory default
	if config.OutputDir == "" {
		config.OutputDir = config.Name
	}

	// Set module path default
	if config.ModulePath == "" {
		config.ModulePath = fmt.Sprintf("github.com/example/%s", config.Name)
	}

	// Set Go version if not provided
	if config.GoVersion == "" {
		config.GoVersion = "1.19"
	}

	// Set database defaults if enabled
	if config.Features.Database {
		config.Database = interfaces.DatabaseConfig{
			Type:     "mysql",
			Host:     "localhost",
			Port:     3306,
			Database: strings.ReplaceAll(config.Name, "-", "_"),
			Username: "user",
			Password: "password",
		}
	}

	return config, nil
}

// loadServiceConfigFromFile loads service configuration from a YAML file.
func loadServiceConfigFromFile(configPath string) (*interfaces.ServiceConfig, error) {
	if !fileExists(configPath) {
		return nil, fmt.Errorf("configuration file does not exist: %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	var config interfaces.ServiceConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file: %w", err)
	}

	// Apply defaults for missing fields
	if config.Version == "" {
		config.Version = "0.1.0"
	}
	if config.Metadata == nil {
		config.Metadata = make(map[string]string)
	}
	if config.Ports.HTTP == 0 {
		config.Ports.HTTP = 9000
	}
	if config.Ports.GRPC == 0 {
		config.Ports.GRPC = 10000
	}

	return &config, nil
}

// generateService generates the service using the provided configuration.
func generateService(ctx context.Context, container interfaces.DependencyContainer, config *interfaces.ServiceConfig) error {
	// Get services from container
	generatorService, err := container.GetService("service_generator")
	if err != nil {
		return fmt.Errorf("failed to get service generator: %w", err)
	}
	serviceGenerator := generatorService.(interfaces.Generator)

	uiService, err := container.GetService("ui")
	if err != nil {
		return fmt.Errorf("failed to get UI service: %w", err)
	}
	terminalUI := uiService.(interfaces.InteractiveUI)

	loggerService, err := container.GetService("logger")
	if err != nil {
		return fmt.Errorf("failed to get logger service: %w", err)
	}
	logger := loggerService.(interfaces.Logger)

	// Check if output directory exists and handle accordingly
	if !force && dirExists(config.OutputDir) {
		confirmed, err := terminalUI.PromptConfirm(
			fmt.Sprintf("Directory '%s' already exists. Overwrite?", config.OutputDir),
			false,
		)
		if err != nil {
			return fmt.Errorf("failed to get overwrite confirmation: %w", err)
		}
		if !confirmed {
			terminalUI.ShowInfo("Service generation cancelled.")
			return nil
		}
	}

	// Show progress during generation
	progress := terminalUI.ShowProgress("Generating service", 100)

	// Dry run mode
	if dryRun {
		terminalUI.ShowInfo("DRY RUN MODE - No files will be created")
		terminalUI.PrintSeparator()

		// Show what would be generated
		return showDryRunPreview(terminalUI, config)
	}

	// Generate the service
	progress.SetMessage("Creating directory structure...")
	progress.Update(10)

	if err := serviceGenerator.GenerateService(*config); err != nil {
		progress.Finish()
		return fmt.Errorf("failed to generate service: %w", err)
	}

	progress.Update(100)
	progress.Finish()

	// Show success message and next steps
	return showGenerationSuccess(terminalUI, logger, config)
}

// showDryRunPreview shows what would be generated in dry run mode.
func showDryRunPreview(terminalUI interfaces.InteractiveUI, config *interfaces.ServiceConfig) error {
	terminalUI.PrintHeader("Dry Run Preview")

	fmt.Printf("Would create service: %s\n", terminalUI.GetStyle().Primary.Sprint(config.Name))
	fmt.Printf("Output directory: %s\n", config.OutputDir)
	fmt.Printf("Module path: %s\n", config.ModulePath)
	fmt.Println()

	fmt.Println("Files that would be created:")
	files := []string{
		"go.mod",
		"go.sum",
		"main.go",
		"Makefile",
		"README.md",
		".gitignore",
		"Dockerfile",
		".dockerignore",
		filepath.Join("internal", toPackageName(config.Name), "server.go"),
		filepath.Join("internal", toPackageName(config.Name), "adapter.go"),
		filepath.Join("internal", toPackageName(config.Name), "config", "config.go"),
		filepath.Join("internal", toPackageName(config.Name), "cmd", "cmd.go"),
		filepath.Join("internal", toPackageName(config.Name), "interfaces", "interfaces.go"),
		filepath.Join("internal", toPackageName(config.Name), "types", "types.go"),
		filepath.Join("internal", toPackageName(config.Name), "types", "errors.go"),
		config.Name + ".yaml",
	}

	// Add feature-specific files
	if config.Features.Database {
		files = append(files,
			filepath.Join("internal", toPackageName(config.Name), "db", "db.go"),
			filepath.Join("internal", toPackageName(config.Name), "model", ".gitkeep"),
			filepath.Join("internal", toPackageName(config.Name), "repository", ".gitkeep"),
			filepath.Join("migrations", ".gitkeep"),
		)
	}

	if config.Features.Authentication {
		files = append(files,
			filepath.Join("internal", toPackageName(config.Name), "auth", "auth.go"),
			filepath.Join("internal", toPackageName(config.Name), "middleware", "auth.go"),
		)
	}

	if config.Features.Kubernetes {
		files = append(files,
			filepath.Join("deployments", "k8s", "deployment.yaml"),
			filepath.Join("deployments", "k8s", "service.yaml"),
			filepath.Join("deployments", "k8s", "configmap.yaml"),
		)
	}

	for _, file := range files {
		fmt.Printf("  %s %s\n", terminalUI.GetStyle().Info.Sprint("•"), file)
	}

	fmt.Println()
	terminalUI.ShowInfo("Run without --dry-run to create these files.")

	return nil
}

// showGenerationSuccess shows success message and next steps.
func showGenerationSuccess(terminalUI interfaces.InteractiveUI, logger interfaces.Logger, config *interfaces.ServiceConfig) error {
	terminalUI.ShowSuccess(fmt.Sprintf("Service '%s' generated successfully!", config.Name))

	terminalUI.PrintHeader("Next Steps")

	fmt.Printf("1. Navigate to your service directory:\n")
	fmt.Printf("   %s\n\n", terminalUI.GetStyle().Primary.Sprint(fmt.Sprintf("cd %s", config.OutputDir)))

	fmt.Printf("2. Install dependencies:\n")
	fmt.Printf("   %s\n\n", terminalUI.GetStyle().Primary.Sprint("go mod tidy"))

	fmt.Printf("3. Build your service:\n")
	fmt.Printf("   %s\n\n", terminalUI.GetStyle().Primary.Sprint("make build"))

	fmt.Printf("4. Run your service:\n")
	fmt.Printf("   %s\n\n", terminalUI.GetStyle().Primary.Sprint("make run"))

	if config.Features.Database {
		fmt.Printf("5. Configure your database connection in %s\n\n", terminalUI.GetStyle().Highlight.Sprint(config.Name+".yaml"))
	}

	fmt.Printf("For more information, see the generated README.md file.\n")

	logger.Info("Service generation completed successfully",
		"service", config.Name,
		"output", config.OutputDir,
		"features", fmt.Sprintf("%+v", config.Features))

	return nil
}

// getDefaultDatabasePort returns the default port for a database type.
func getDefaultDatabasePort(dbType string) int {
	switch dbType {
	case "mysql":
		return 3306
	case "postgresql":
		return 5432
	case "mongodb":
		return 27017
	case "redis":
		return 6379
	case "sqlite":
		return 0 // SQLite doesn't use a port
	default:
		return 3306
	}
}

// toPackageName converts service name to valid Go package name.
func toPackageName(serviceName string) string {
	return strings.ReplaceAll(serviceName, "-", "")
}
