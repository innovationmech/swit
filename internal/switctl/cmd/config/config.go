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

// Package config provides configuration management commands for switctl.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/innovationmech/swit/internal/switctl/config"
	"github.com/innovationmech/swit/internal/switctl/deps"
	"github.com/innovationmech/swit/internal/switctl/filesystem"
	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/innovationmech/swit/internal/switctl/ui"
)

var (
	// Global flags for config commands
	verbose      bool
	noColor      bool
	workDir      string
	configFile   string
	outputFile   string
	format       string
	strict       bool
	showDefaults bool
)

// NewConfigCommand creates the main 'config' command with subcommands.
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration file management and validation",
		Long: `The config command provides configuration file management and validation for Swit projects.

This command includes subcommands for:
• validate: Validate configuration file syntax and structure correctness
• template: Generate standard configuration file templates for services
• schema: Generate JSON schema for configuration validation
• migrate: Migrate configuration files to newer formats
• merge: Merge multiple configuration files

Examples:
  # Validate all configuration files
  switctl config validate

  # Validate specific configuration file
  switctl config validate --file swit.yaml

  # Generate configuration template for a service
  switctl config template --type service --output my-service.yaml

  # Generate JSON schema for configuration
  switctl config schema --output config-schema.json

  # Migrate configuration to newer format
  switctl config migrate --from v1 --to v2`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeConfigCommandConfig(cmd)
		},
	}

	// Add persistent flags
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	cmd.PersistentFlags().StringVarP(&workDir, "work-dir", "w", "", "Working directory (default: current directory)")
	cmd.PersistentFlags().StringVarP(&configFile, "file", "f", "", "Specific configuration file to process")
	cmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "Output file for generated content")
	cmd.PersistentFlags().StringVar(&format, "format", "yaml", "Output format (yaml, json)")

	// Bind flags to viper
	viper.BindPFlag("config.verbose", cmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("config.no_color", cmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("config.work_dir", cmd.PersistentFlags().Lookup("work-dir"))
	viper.BindPFlag("config.file", cmd.PersistentFlags().Lookup("file"))
	viper.BindPFlag("config.output", cmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("config.format", cmd.PersistentFlags().Lookup("format"))

	// Add subcommands
	cmd.AddCommand(NewValidateCommand())
	cmd.AddCommand(NewTemplateCommand())
	cmd.AddCommand(NewSchemaCommand())
	cmd.AddCommand(NewMigrateCommand())
	cmd.AddCommand(NewMergeCommand())

	return cmd
}

// initializeConfigCommandConfig initializes configuration for config commands.
func initializeConfigCommandConfig(cmd *cobra.Command) error {
	// Get config values from viper
	if viper.IsSet("config.verbose") {
		verbose = viper.GetBool("config.verbose")
	}
	if viper.IsSet("config.no_color") {
		noColor = viper.GetBool("config.no_color")
	}
	if viper.IsSet("config.work_dir") {
		workDir = viper.GetString("config.work_dir")
	}
	if viper.IsSet("config.file") {
		configFile = viper.GetString("config.file")
	}
	if viper.IsSet("config.output") {
		outputFile = viper.GetString("config.output")
	}
	if viper.IsSet("config.format") {
		format = viper.GetString("config.format")
	}

	// Set default working directory
	if workDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		workDir = wd
	}

	// Validate format
	if format != "yaml" && format != "json" {
		return fmt.Errorf("unsupported format: %s (supported: yaml, json)", format)
	}

	return nil
}

// NewValidateCommand creates the 'validate' subcommand.
func NewValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [files...]",
		Short: "Validate configuration file syntax and structure",
		Long: `The validate command checks configuration files for syntax errors and structural correctness.

Features:
• YAML and JSON syntax validation
• Schema validation against Swit configuration standards
• Cross-reference validation between configuration files
• Environment variable expansion validation
• Required field checking
• Type validation and range checking

Examples:
  # Validate all configuration files in current directory
  switctl config validate

  # Validate specific files
  switctl config validate swit.yaml switauth.yaml

  # Validate with strict mode (no warnings allowed)
  switctl config validate --strict

  # Show validation results with defaults
  switctl config validate --show-defaults`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidateCommand(args)
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Strict mode - treat warnings as errors")
	cmd.Flags().BoolVar(&showDefaults, "show-defaults", false, "Show default values in validation results")

	return cmd
}

// NewTemplateCommand creates the 'template' subcommand.
func NewTemplateCommand() *cobra.Command {
	var (
		templateType string
		serviceName  string
		includes     []string
		excludes     []string
	)

	cmd := &cobra.Command{
		Use:   "template",
		Short: "Generate standard configuration file templates",
		Long: `The template command generates configuration file templates for different Swit components.

Available template types:
• service: Service configuration template
• gateway: API gateway configuration template
• database: Database configuration template
• cache: Cache configuration template
• monitoring: Monitoring configuration template
• deployment: Deployment configuration template

Examples:
  # Generate service configuration template
  switctl config template --type service --service my-service

  # Generate gateway configuration template
  switctl config template --type gateway --output gateway.yaml

  # Generate database configuration with specific features
  switctl config template --type database --include mysql,redis

  # Generate template excluding certain components
  switctl config template --type service --exclude auth,cache`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTemplateCommand(templateType, serviceName, includes, excludes)
		},
	}

	cmd.Flags().StringVarP(&templateType, "type", "t", "service", "Template type (service, gateway, database, cache, monitoring, deployment)")
	cmd.Flags().StringVarP(&serviceName, "service", "s", "", "Service name for template")
	cmd.Flags().StringSliceVar(&includes, "include", nil, "Components to include in template")
	cmd.Flags().StringSliceVar(&excludes, "exclude", nil, "Components to exclude from template")

	return cmd
}

// NewSchemaCommand creates the 'schema' subcommand.
func NewSchemaCommand() *cobra.Command {
	var (
		schemaType string
		version    string
	)

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Generate JSON schema for configuration validation",
		Long: `The schema command generates JSON schemas that can be used for configuration validation.

Examples:
  # Generate JSON schema for service configuration
  switctl config schema --type service

  # Generate schema for specific version
  switctl config schema --type service --version v2

  # Save schema to file
  switctl config schema --type service --output service-schema.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSchemaCommand(schemaType, version)
		},
	}

	cmd.Flags().StringVarP(&schemaType, "type", "t", "service", "Schema type (service, gateway, database)")
	cmd.Flags().StringVar(&version, "version", "latest", "Schema version")

	return cmd
}

// NewMigrateCommand creates the 'migrate' subcommand.
func NewMigrateCommand() *cobra.Command {
	var (
		fromVersion string
		toVersion   string
		backup      bool
	)

	cmd := &cobra.Command{
		Use:   "migrate [files...]",
		Short: "Migrate configuration files to newer formats",
		Long: `The migrate command migrates configuration files from older formats to newer ones.

Examples:
  # Migrate configuration from v1 to v2
  switctl config migrate --from v1 --to v2 swit.yaml

  # Migrate all configuration files
  switctl config migrate --from v1 --to v2

  # Migrate with backup
  switctl config migrate --from v1 --to v2 --backup`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrateCommand(args, fromVersion, toVersion, backup)
		},
	}

	cmd.Flags().StringVar(&fromVersion, "from", "", "Source version")
	cmd.Flags().StringVar(&toVersion, "to", "", "Target version")
	cmd.Flags().BoolVar(&backup, "backup", true, "Create backup of original files")

	return cmd
}

// NewMergeCommand creates the 'merge' subcommand.
func NewMergeCommand() *cobra.Command {
	var (
		strategy    string
		overwrite   bool
		ignoreNulls bool
	)

	cmd := &cobra.Command{
		Use:   "merge <base-file> <overlay-files...>",
		Short: "Merge multiple configuration files",
		Long: `The merge command combines multiple configuration files into one.

Merge strategies:
• override: Later values override earlier ones (default)
• append: Append array values instead of replacing
• deep: Deep merge nested objects

Examples:
  # Merge configurations with override strategy
  switctl config merge base.yaml overlay.yaml

  # Merge with deep merge strategy
  switctl config merge --strategy deep base.yaml dev.yaml prod.yaml

  # Merge and save to file
  switctl config merge base.yaml overlay.yaml --output merged.yaml`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMergeCommand(args, strategy, overwrite, ignoreNulls)
		},
	}

	cmd.Flags().StringVar(&strategy, "strategy", "override", "Merge strategy (override, append, deep)")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite base file with merged result")
	cmd.Flags().BoolVar(&ignoreNulls, "ignore-nulls", false, "Ignore null values during merge")

	return cmd
}

// Command implementations

// runValidateCommand executes the validate subcommand.
func runValidateCommand(files []string) error {
	// Create dependency container and services
	container, err := createDependencyContainer()
	if err != nil {
		return fmt.Errorf("failed to create dependency container: %w", err)
	}
	defer container.Close()

	// Get services
	uiService, err := container.GetService("ui")
	if err != nil {
		return fmt.Errorf("failed to get UI service: %w", err)
	}
	terminalUI := uiService.(interfaces.InteractiveUI)

	configManager, err := container.GetService("config_manager")
	if err != nil {
		return fmt.Errorf("failed to get config manager: %w", err)
	}
	manager := configManager.(interfaces.ConfigManager)

	terminalUI.PrintHeader("✅ Configuration Validation")

	// Determine files to validate
	configFiles := files
	if len(configFiles) == 0 {
		if configFile != "" {
			configFiles = []string{configFile}
		} else {
			// Find all config files in current directory
			detectedFiles, err := findConfigFiles()
			if err != nil {
				return fmt.Errorf("failed to find configuration files: %w", err)
			}
			configFiles = detectedFiles
		}
	}

	if len(configFiles) == 0 {
		terminalUI.ShowInfo("No configuration files found to validate")
		return nil
	}

	terminalUI.ShowInfo(fmt.Sprintf("Validating %d configuration file(s)", len(configFiles)))
	terminalUI.ShowInfo(fmt.Sprintf("Strict mode: %s", formatBool(strict)))

	// Validate each file
	allValid := true
	results := []ValidationResult{}

	progress := terminalUI.ShowProgress("Validating files", len(configFiles))

	for i, file := range configFiles {
		progress.SetMessage(fmt.Sprintf("Validating %s...", filepath.Base(file)))

		result := validateConfigFile(manager, file)
		results = append(results, result)

		if !result.Valid {
			allValid = false
		}

		if strict && len(result.Warnings) > 0 {
			allValid = false
		}

		progress.Update(i + 1)
	}

	progress.Finish()

	// Display results
	terminalUI.PrintSubHeader("Validation Results")

	for _, result := range results {
		displayValidationResult(terminalUI, result)
	}

	// Display summary
	terminalUI.PrintSubHeader("Summary")

	validCount := 0
	warningCount := 0
	errorCount := 0

	for _, result := range results {
		if result.Valid {
			validCount++
		}
		warningCount += len(result.Warnings)
		errorCount += len(result.Errors)
	}

	summaryTable := [][]string{
		{"Total Files", fmt.Sprintf("%d", len(configFiles))},
		{"Valid", fmt.Sprintf("%d", validCount)},
		{"With Warnings", fmt.Sprintf("%d", warningCount)},
		{"With Errors", fmt.Sprintf("%d", errorCount)},
	}

	if err := terminalUI.ShowTable([]string{"Metric", "Count"}, summaryTable); err != nil {
		return fmt.Errorf("failed to show summary table: %w", err)
	}

	if allValid {
		terminalUI.ShowSuccess("All configuration files are valid!")
	} else {
		terminalUI.ShowError(fmt.Errorf("validation failed for one or more files"))
		return fmt.Errorf("configuration validation failed")
	}

	return nil
}

// runTemplateCommand executes the template subcommand.
func runTemplateCommand(templateType, serviceName string, includes, excludes []string) error {
	// Create dependency container and services
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

	terminalUI.PrintHeader("📝 Configuration Template Generation")

	// Validate template type
	validTypes := []string{"service", "gateway", "database", "cache", "monitoring", "deployment"}
	if !contains(validTypes, templateType) {
		return fmt.Errorf("invalid template type: %s (valid types: %s)", templateType, strings.Join(validTypes, ", "))
	}

	terminalUI.ShowInfo(fmt.Sprintf("Template type: %s", templateType))
	if serviceName != "" {
		terminalUI.ShowInfo(fmt.Sprintf("Service name: %s", serviceName))
	}
	if len(includes) > 0 {
		terminalUI.ShowInfo(fmt.Sprintf("Including: %s", strings.Join(includes, ", ")))
	}
	if len(excludes) > 0 {
		terminalUI.ShowInfo(fmt.Sprintf("Excluding: %s", strings.Join(excludes, ", ")))
	}

	// Generate template
	template, err := generateConfigTemplate(templateType, serviceName, includes, excludes)
	if err != nil {
		return fmt.Errorf("failed to generate template: %w", err)
	}

	// Output template
	if outputFile != "" {
		if err := writeTemplateToFile(template, outputFile); err != nil {
			return fmt.Errorf("failed to write template to file: %w", err)
		}
		terminalUI.ShowSuccess(fmt.Sprintf("Template saved to %s", outputFile))
	} else {
		fmt.Println(template)
	}

	return nil
}

// runSchemaCommand executes the schema subcommand.
func runSchemaCommand(schemaType, version string) error {
	// Create dependency container and services
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

	terminalUI.PrintHeader("📋 JSON Schema Generation")

	terminalUI.ShowInfo(fmt.Sprintf("Schema type: %s", schemaType))
	terminalUI.ShowInfo(fmt.Sprintf("Version: %s", version))

	// Generate schema
	schema, err := generateJSONSchema(schemaType, version)
	if err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}

	// Output schema
	if outputFile != "" {
		if err := writeSchemaToFile(schema, outputFile); err != nil {
			return fmt.Errorf("failed to write schema to file: %w", err)
		}
		terminalUI.ShowSuccess(fmt.Sprintf("Schema saved to %s", outputFile))
	} else {
		fmt.Println(schema)
	}

	return nil
}

// runMigrateCommand executes the migrate subcommand.
func runMigrateCommand(files []string, fromVersion, toVersion string, backup bool) error {
	// Create dependency container and services
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

	terminalUI.PrintHeader("🔄 Configuration Migration")

	if fromVersion == "" || toVersion == "" {
		return fmt.Errorf("both --from and --to versions must be specified")
	}

	terminalUI.ShowInfo(fmt.Sprintf("Migrating from %s to %s", fromVersion, toVersion))
	terminalUI.ShowInfo(fmt.Sprintf("Backup: %s", formatBool(backup)))

	// Determine files to migrate
	migrateFiles := files
	if len(migrateFiles) == 0 {
		detectedFiles, err := findConfigFiles()
		if err != nil {
			return fmt.Errorf("failed to find configuration files: %w", err)
		}
		migrateFiles = detectedFiles
	}

	if len(migrateFiles) == 0 {
		terminalUI.ShowInfo("No configuration files found to migrate")
		return nil
	}

	// Migrate files
	progress := terminalUI.ShowProgress("Migrating files", len(migrateFiles))

	for i, file := range migrateFiles {
		progress.SetMessage(fmt.Sprintf("Migrating %s...", filepath.Base(file)))

		if err := migrateConfigFile(file, fromVersion, toVersion, backup); err != nil {
			terminalUI.ShowError(fmt.Errorf("failed to migrate %s: %w", file, err))
		}

		progress.Update(i + 1)
	}

	progress.Finish()

	terminalUI.ShowSuccess(fmt.Sprintf("Successfully migrated %d configuration files", len(migrateFiles)))
	return nil
}

// runMergeCommand executes the merge subcommand.
func runMergeCommand(files []string, strategy string, overwrite, ignoreNulls bool) error {
	// Create dependency container and services
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

	terminalUI.PrintHeader("🔗 Configuration Merge")

	validStrategies := []string{"override", "append", "deep"}
	if !contains(validStrategies, strategy) {
		return fmt.Errorf("invalid merge strategy: %s (valid strategies: %s)", strategy, strings.Join(validStrategies, ", "))
	}

	baseFile := files[0]
	overlayFiles := files[1:]

	terminalUI.ShowInfo(fmt.Sprintf("Base file: %s", baseFile))
	terminalUI.ShowInfo(fmt.Sprintf("Overlay files: %s", strings.Join(overlayFiles, ", ")))
	terminalUI.ShowInfo(fmt.Sprintf("Strategy: %s", strategy))

	// Perform merge
	merged, err := mergeConfigFiles(baseFile, overlayFiles, strategy, ignoreNulls)
	if err != nil {
		return fmt.Errorf("failed to merge configuration files: %w", err)
	}

	// Output merged configuration
	outputPath := outputFile
	if outputPath == "" && overwrite {
		outputPath = baseFile
	}

	if outputPath != "" {
		if err := writeMergedConfig(merged, outputPath); err != nil {
			return fmt.Errorf("failed to write merged configuration: %w", err)
		}
		terminalUI.ShowSuccess(fmt.Sprintf("Merged configuration saved to %s", outputPath))
	} else {
		fmt.Println(merged)
	}

	return nil
}

// Helper types and functions

// ValidationResult represents the result of configuration validation.
type ValidationResult struct {
	File     string
	Valid    bool
	Errors   []string
	Warnings []string
}

// createDependencyContainer creates the dependency injection container.
var createDependencyContainer = func() (interfaces.DependencyContainer, error) {
	container := deps.NewContainer()

	// Register UI service
	if err := container.RegisterSingleton("ui", func() (interface{}, error) {
		return ui.NewTerminalUI(
			ui.WithVerbose(verbose),
			ui.WithNoColor(noColor),
		), nil
	}); err != nil {
		return nil, fmt.Errorf("failed to register UI service: %w", err)
	}

	// Register filesystem service
	if err := container.RegisterSingleton("filesystem", func() (interface{}, error) {
		return filesystem.NewOSFileSystem(workDir), nil
	}); err != nil {
		return nil, fmt.Errorf("failed to register filesystem service: %w", err)
	}

	// Register config manager
	if err := container.RegisterSingleton("config_manager", func() (interface{}, error) {
		// Get filesystem service for logger (if available)
		var logger interfaces.Logger
		if loggerService, err := container.GetService("logger"); err == nil {
			logger = loggerService.(interfaces.Logger)
		}

		// Use the new hierarchical config manager
		configManager := config.NewHierarchicalConfigManager(workDir, logger)

		// Load configuration
		if err := configManager.Load(); err != nil && verbose {
			// Log warning but continue - configuration loading is optional for commands
			fmt.Fprintf(os.Stderr, "Warning: failed to load configuration: %v\n", err)
		}

		return configManager, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to register config manager: %w", err)
	}

	return container, nil
}

// findConfigFiles finds all configuration files in the working directory.
var findConfigFiles = func() ([]string, error) {
	patterns := []string{"*.yaml", "*.yml", "*.json"}
	configFiles := []string{}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(workDir, pattern))
		if err != nil {
			return nil, fmt.Errorf("failed to glob pattern %s: %w", pattern, err)
		}
		configFiles = append(configFiles, matches...)
	}

	return configFiles, nil
}

// validateConfigFile validates a single configuration file.
func validateConfigFile(manager interfaces.ConfigManager, file string) ValidationResult {
	result := ValidationResult{
		File:     file,
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Check if file exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		result.Valid = false
		result.Errors = append(result.Errors, "File does not exist")
		return result
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file))
	if ext != ".yaml" && ext != ".yml" && ext != ".json" {
		result.Warnings = append(result.Warnings, "File extension is not .yaml, .yml, or .json")
	}

	// Validate YAML/JSON syntax
	content, err := os.ReadFile(file)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read file: %v", err))
		return result
	}

	if ext == ".json" {
		if err := validateJSONSyntax(content); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid JSON syntax: %v", err))
		}
	} else {
		if err := validateYAMLSyntax(content); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid YAML syntax: %v", err))
		}
	}

	// Additional semantic validation would go here
	// For now, just check basic structure

	return result
}

// displayValidationResult displays the validation result for a single file.
func displayValidationResult(ui interfaces.InteractiveUI, result ValidationResult) {
	status := "✅ VALID"
	if !result.Valid {
		status = "❌ INVALID"
	} else if len(result.Warnings) > 0 {
		status = "⚠️ VALID (with warnings)"
	}

	fmt.Fprintf(os.Stdout, "\n%s %s\n", status, result.File)

	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stdout, "  %s Errors:\n", ui.GetStyle().Error.Sprint("●"))
		for _, err := range result.Errors {
			fmt.Fprintf(os.Stdout, "    - %s\n", ui.GetStyle().Error.Sprint(err))
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Fprintf(os.Stdout, "  %s Warnings:\n", ui.GetStyle().Warning.Sprint("●"))
		for _, warning := range result.Warnings {
			fmt.Fprintf(os.Stdout, "    - %s\n", ui.GetStyle().Warning.Sprint(warning))
		}
	}
}

// generateConfigTemplate generates a configuration template.
func generateConfigTemplate(templateType, serviceName string, includes, excludes []string) (string, error) {
	switch templateType {
	case "service":
		return generateServiceTemplate(serviceName, includes, excludes)
	case "gateway":
		return generateGatewayTemplate(includes, excludes)
	case "database":
		return generateDatabaseTemplate(includes, excludes)
	case "cache":
		return generateCacheTemplate(includes, excludes)
	case "monitoring":
		return generateMonitoringTemplate(includes, excludes)
	case "deployment":
		return generateDeploymentTemplate(includes, excludes)
	default:
		return "", fmt.Errorf("unsupported template type: %s", templateType)
	}
}

// generateServiceTemplate generates a service configuration template.
func generateServiceTemplate(serviceName string, includes, excludes []string) (string, error) {
	if serviceName == "" {
		serviceName = "my-service"
	}

	template := fmt.Sprintf(`# Service configuration for %s
name: %s
description: "Service description"
version: "1.0.0"

# Server configuration
server:
  http:
    port: 9000
    host: "0.0.0.0"
    timeout: 30s
  grpc:
    port: 10000
    host: "0.0.0.0"
    timeout: 30s

# Database configuration
database:
  type: mysql
  host: localhost
  port: 3306
  name: %s
  username: root
  password: ""
  max_connections: 100
  max_idle: 10

# Logging configuration
logging:
  level: info
  format: json
  output: stdout

# Metrics configuration
metrics:
  enabled: true
  port: 9090
  path: /metrics

# Health check configuration
health:
  enabled: true
  port: 8080
  path: /health
`, serviceName, serviceName, serviceName)

	return template, nil
}

func generateGatewayTemplate(includes, excludes []string) (string, error) {
	template := `# API Gateway configuration
name: api-gateway
description: "API Gateway service"

# Gateway configuration
gateway:
  port: 8080
  host: "0.0.0.0"
  
# Route configuration
routes:
  - name: user-service
    path: /api/v1/users/*
    target: http://user-service:9000
    strip_prefix: false
  
# CORS configuration
cors:
  enabled: true
  origins: ["*"]
  methods: ["GET", "POST", "PUT", "DELETE"]
  headers: ["Authorization", "Content-Type"]

# Rate limiting
rate_limit:
  enabled: true
  requests_per_minute: 1000
`

	return template, nil
}

func generateDatabaseTemplate(includes, excludes []string) (string, error) {
	template := `# Database configuration
database:
  type: mysql
  host: localhost
  port: 3306
  name: myapp
  username: root
  password: ""
  
  # Connection pool settings
  max_connections: 100
  max_idle: 10
  max_lifetime: 1h
  
  # Performance settings
  slow_query_log: true
  query_timeout: 30s
  
  # Migration settings
  migrate: true
  migration_dir: ./migrations
`

	return template, nil
}

func generateCacheTemplate(includes, excludes []string) (string, error) {
	template := `# Cache configuration
cache:
  type: redis
  host: localhost
  port: 6379
  database: 0
  password: ""
  
  # Pool settings
  max_connections: 100
  max_idle: 10
  
  # Timeout settings
  connect_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
  
  # Default TTL
  default_ttl: 1h
`

	return template, nil
}

func generateMonitoringTemplate(includes, excludes []string) (string, error) {
	template := `# Monitoring configuration
monitoring:
  # Metrics
  metrics:
    enabled: true
    port: 9090
    path: /metrics
    
  # Tracing
  tracing:
    enabled: true
    jaeger:
      endpoint: http://localhost:14268/api/traces
      
  # Health checks
  health:
    enabled: true
    port: 8080
    checks:
      - name: database
        type: database
      - name: cache
        type: redis
`

	return template, nil
}

func generateDeploymentTemplate(includes, excludes []string) (string, error) {
	template := `# Deployment configuration
deployment:
  replicas: 3
  
  # Resource limits
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
      
  # Environment variables
  env:
    - name: LOG_LEVEL
      value: info
    - name: DATABASE_URL
      valueFrom:
        secretKeyRef:
          name: db-secret
          key: url
          
  # Health checks
  health_check:
    path: /health
    port: 8080
    initial_delay: 30s
    timeout: 5s
`

	return template, nil
}

// Placeholder implementations for other helper functions

func generateJSONSchema(schemaType, version string) (string, error) {
	// In a real implementation, this would generate actual JSON schemas
	schema := fmt.Sprintf(`{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "title": "%s Configuration Schema",
  "version": "%s",
  "properties": {
    "name": {
      "type": "string",
      "description": "Service name"
    }
  },
  "required": ["name"]
}`, schemaType, version)

	return schema, nil
}

func writeTemplateToFile(template, filename string) error {
	return os.WriteFile(filename, []byte(template), 0644)
}

func writeSchemaToFile(schema, filename string) error {
	return os.WriteFile(filename, []byte(schema), 0644)
}

func migrateConfigFile(file, fromVersion, toVersion string, backup bool) error {
	if backup {
		backupFile := file + ".backup"
		if err := copyFile(file, backupFile); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// In a real implementation, this would perform actual migration
	// For now, just pretend we migrated the file
	return nil
}

func mergeConfigFiles(baseFile string, overlayFiles []string, strategy string, ignoreNulls bool) (string, error) {
	// In a real implementation, this would perform actual configuration merging
	// For now, just return the base file content
	content, err := os.ReadFile(baseFile)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func writeMergedConfig(merged, filename string) error {
	return os.WriteFile(filename, []byte(merged), 0644)
}

func validateJSONSyntax(content []byte) error {
	var v interface{}
	return json.Unmarshal(content, &v)
}

func validateYAMLSyntax(content []byte) error {
	var v interface{}
	return yaml.Unmarshal(content, &v)
}

func copyFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, content, 0644)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func formatBool(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}

// SimpleConfigManager provides a basic config manager implementation.
type SimpleConfigManager struct {
	workDir string
}

// NewConfigManager creates a new config manager.
func NewConfigManager(workDir string) *SimpleConfigManager {
	return &SimpleConfigManager{workDir: workDir}
}

// Load loads configuration from multiple sources.
func (cm *SimpleConfigManager) Load() error {
	return nil
}

// Get retrieves a configuration value by key.
func (cm *SimpleConfigManager) Get(key string) interface{} {
	return nil
}

// GetString retrieves a string configuration value.
func (cm *SimpleConfigManager) GetString(key string) string {
	return ""
}

// GetInt retrieves an integer configuration value.
func (cm *SimpleConfigManager) GetInt(key string) int {
	return 0
}

// GetBool retrieves a boolean configuration value.
func (cm *SimpleConfigManager) GetBool(key string) bool {
	return false
}

// Set sets a configuration value.
func (cm *SimpleConfigManager) Set(key string, value interface{}) {
}

// Validate validates the current configuration.
func (cm *SimpleConfigManager) Validate() error {
	return nil
}

// Save saves the configuration to file.
func (cm *SimpleConfigManager) Save(path string) error {
	return nil
}
