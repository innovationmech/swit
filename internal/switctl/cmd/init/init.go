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

// Package init provides commands for initializing new projects.
package init

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/innovationmech/swit/internal/switctl/deps"
	"github.com/innovationmech/swit/internal/switctl/filesystem"
	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/innovationmech/swit/internal/switctl/template"
	"github.com/innovationmech/swit/internal/switctl/ui"
)

var (
	// Command flags
	interactive    bool
	projectName    string
	projectType    string
	outputDir      string
	modulePath     string
	author         string
	description    string
	license        string
	goVersion      string
	verbose        bool
	noColor        bool
	force          bool
	dryRun         bool
	configTemplate string
)

// NewInitCommand creates the 'init' command for project initialization.
func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize a new project with interactive wizard",
		Long: `The init command creates a new project with the Swit framework.

This command provides an interactive wizard that guides you through:
• Project type selection (monolithic service, microservice cluster, API gateway)
• Basic project information configuration (name, description, author, license)
• Infrastructure component selection (database, cache, message queue, service discovery)
• Project structure preview and configuration summary
• Complete project structure creation

Examples:
  # Interactive project initialization
  switctl init

  # Interactive with specific project name
  switctl init my-project

  # Non-interactive with basic configuration
  switctl init my-project --type microservice --author "John Doe" --no-interactive

  # Preview project structure without creating files
  switctl init my-project --dry-run`,
		Args: cobra.MaximumNArgs(1),
		RunE: runInitCommand,
	}

	// Add flags
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", true, "Use interactive mode for configuration")
	cmd.Flags().StringVarP(&projectType, "type", "t", "", "Project type (monolith, microservice, library, api_gateway, cli)")
	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory for the project")
	cmd.Flags().StringVarP(&modulePath, "module-path", "m", "", "Go module path")
	cmd.Flags().StringVarP(&author, "author", "a", "", "Project author")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Project description")
	cmd.Flags().StringVarP(&license, "license", "l", "MIT", "Project license")
	cmd.Flags().StringVar(&goVersion, "go-version", "1.19", "Go version to use")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files without confirmation")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview what will be created without actually creating files")
	cmd.Flags().StringVar(&configTemplate, "config-template", "", "Configuration template to use")

	// Bind flags to viper
	viper.BindPFlag("init.interactive", cmd.Flags().Lookup("interactive"))
	viper.BindPFlag("init.project_type", cmd.Flags().Lookup("type"))
	viper.BindPFlag("init.output_dir", cmd.Flags().Lookup("output-dir"))
	viper.BindPFlag("init.module_path", cmd.Flags().Lookup("module-path"))
	viper.BindPFlag("init.author", cmd.Flags().Lookup("author"))
	viper.BindPFlag("init.description", cmd.Flags().Lookup("description"))
	viper.BindPFlag("init.license", cmd.Flags().Lookup("license"))
	viper.BindPFlag("init.go_version", cmd.Flags().Lookup("go-version"))
	viper.BindPFlag("init.verbose", cmd.Flags().Lookup("verbose"))
	viper.BindPFlag("init.no_color", cmd.Flags().Lookup("no-color"))
	viper.BindPFlag("init.force", cmd.Flags().Lookup("force"))
	viper.BindPFlag("init.dry_run", cmd.Flags().Lookup("dry-run"))
	viper.BindPFlag("init.config_template", cmd.Flags().Lookup("config-template"))

	return cmd
}

// runInitCommand executes the init command.
func runInitCommand(cmd *cobra.Command, args []string) error {
	// Get project name from args if provided
	if len(args) > 0 {
		projectName = args[0]
	}

	// Override flags with viper values
	if viper.IsSet("init.interactive") {
		interactive = viper.GetBool("init.interactive")
	}
	if viper.IsSet("init.project_type") {
		projectType = viper.GetString("init.project_type")
	}
	if viper.IsSet("init.output_dir") {
		outputDir = viper.GetString("init.output_dir")
	}
	if viper.IsSet("init.module_path") {
		modulePath = viper.GetString("init.module_path")
	}
	if viper.IsSet("init.author") {
		author = viper.GetString("init.author")
	}
	if viper.IsSet("init.description") {
		description = viper.GetString("init.description")
	}
	if viper.IsSet("init.license") {
		license = viper.GetString("init.license")
	}
	if viper.IsSet("init.go_version") {
		goVersion = viper.GetString("init.go_version")
	}
	if viper.IsSet("init.verbose") {
		verbose = viper.GetBool("init.verbose")
	}
	if viper.IsSet("init.no_color") {
		noColor = viper.GetBool("init.no_color")
	}
	if viper.IsSet("init.force") {
		force = viper.GetBool("init.force")
	}
	if viper.IsSet("init.dry_run") {
		dryRun = viper.GetBool("init.dry_run")
	}
	if viper.IsSet("init.config_template") {
		configTemplate = viper.GetString("init.config_template")
	}

	// Create dependency container and services
	container, err := createDependencyContainer()
	if err != nil {
		return fmt.Errorf("failed to create dependency container: %w", err)
	}
	defer container.Close()

	// Get services from container
	uiService, err := container.GetService("ui")
	if err != nil {
		return fmt.Errorf("failed to get UI service: %w", err)
	}
	terminalUI := uiService.(*ui.TerminalUI)

	projectManager, err := container.GetService("project_manager")
	if err != nil {
		return fmt.Errorf("failed to get project manager: %w", err)
	}
	manager := projectManager.(interfaces.ProjectManager)

	// Show welcome screen
	if err := terminalUI.ShowWelcome(); err != nil {
		return fmt.Errorf("failed to show welcome screen: %w", err)
	}

	// Create execution context
	ctx := context.WithValue(context.Background(), "start_time", time.Now())
	execCtx := createExecutionContext(ctx)

	// Initialize project
	if interactive {
		return runInteractiveInit(terminalUI, manager, execCtx)
	}
	return runNonInteractiveInit(terminalUI, manager, execCtx)
}

// runInteractiveInit runs the interactive initialization wizard.
func runInteractiveInit(ui *ui.TerminalUI, manager interfaces.ProjectManager, ctx *interfaces.ExecutionContext) error {
	ui.PrintHeader("🚀 Project Initialization Wizard")

	// Collect project configuration through interactive prompts
	config, err := collectProjectConfig(ui)
	if err != nil {
		return fmt.Errorf("failed to collect project configuration: %w", err)
	}

	// Show configuration preview
	if err := showConfigurationPreview(ui, config); err != nil {
		return fmt.Errorf("failed to show configuration preview: %w", err)
	}

	// Confirm creation
	confirmed, err := ui.PromptConfirm("Create project with this configuration?", true)
	if err != nil {
		return fmt.Errorf("failed to get confirmation: %w", err)
	}

	if !confirmed {
		ui.ShowInfo("Project initialization cancelled")
		return nil
	}

	// Create the project
	return createProject(ui, manager, config)
}

// runNonInteractiveInit runs non-interactive initialization.
func runNonInteractiveInit(ui *ui.TerminalUI, manager interfaces.ProjectManager, ctx *interfaces.ExecutionContext) error {
	// Build configuration from flags
	config := buildConfigFromFlags()

	// Validate required fields
	if err := validateRequiredConfig(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Show what will be created (if verbose)
	if verbose {
		if err := showConfigurationPreview(ui, config); err != nil {
			return fmt.Errorf("failed to show configuration preview: %w", err)
		}
	}

	// Create the project
	return createProject(ui, manager, config)
}

// collectProjectConfig collects project configuration through interactive prompts.
func collectProjectConfig(terminalUI *ui.TerminalUI) (interfaces.ProjectConfig, error) {
	config := interfaces.ProjectConfig{}

	// Project name
	if projectName == "" {
		name, err := terminalUI.PromptInput("Project name", ui.ServiceNameValidator)
		if err != nil {
			return config, err
		}
		config.Name = name
	} else {
		config.Name = projectName
	}

	// Project type selection
	terminalUI.PrintSubHeader("Project Type Selection")
	typeOptions := []interfaces.MenuOption{
		{Label: "Monolithic Service", Description: "Single service with all features", Value: "monolith", Icon: "🏢", Enabled: true},
		{Label: "Microservice", Description: "Individual microservice", Value: "microservice", Icon: "🧩", Enabled: true},
		{Label: "Library", Description: "Reusable Go library", Value: "library", Icon: "📚", Enabled: true},
		{Label: "API Gateway", Description: "Gateway service for routing", Value: "api_gateway", Icon: "🚪", Enabled: true},
		{Label: "CLI Tool", Description: "Command-line application", Value: "cli", Icon: "💻", Enabled: true},
	}

	selectedTypeIndex, err := terminalUI.ShowMenu("Select project type", typeOptions)
	if err != nil {
		return config, err
	}
	config.Type = interfaces.ProjectType(typeOptions[selectedTypeIndex].Value.(string))

	// Basic project information
	terminalUI.PrintSubHeader("Basic Information")

	// Description
	desc, err := terminalUI.PromptInput("Project description", ui.OptionalValidator)
	if err != nil {
		return config, err
	}
	config.Description = desc

	// Author
	authorName := author
	if authorName == "" {
		authorName, err = terminalUI.PromptInput("Author name", ui.NonEmptyValidator)
		if err != nil {
			return config, err
		}
	}
	config.Author = authorName

	// License
	licenseOptions := []interfaces.MenuOption{
		{Label: "MIT", Description: "Permissive license", Value: "MIT", Icon: "📄", Enabled: true},
		{Label: "Apache-2.0", Description: "Permissive with patent grant", Value: "Apache-2.0", Icon: "📄", Enabled: true},
		{Label: "GPL-3.0", Description: "Copyleft license", Value: "GPL-3.0", Icon: "📄", Enabled: true},
		{Label: "BSD-3-Clause", Description: "Permissive BSD license", Value: "BSD-3-Clause", Icon: "📄", Enabled: true},
		{Label: "Proprietary", Description: "Closed source", Value: "Proprietary", Icon: "🔒", Enabled: true},
	}

	selectedLicenseIndex, err := terminalUI.ShowMenu("Select license", licenseOptions)
	if err != nil {
		return config, err
	}
	config.License = licenseOptions[selectedLicenseIndex].Value.(string)

	// Module path
	if modulePath == "" {
		modulePath, err = terminalUI.PromptInput("Go module path (e.g., github.com/user/project)", ui.ModulePathValidator)
		if err != nil {
			return config, err
		}
	}
	config.ModulePath = modulePath

	// Output directory
	if outputDir == "" {
		wd, _ := os.Getwd()
		defaultOutput := filepath.Join(wd, config.Name)
		outputDir, err = terminalUI.PromptInput(fmt.Sprintf("Output directory (default: %s)", defaultOutput), ui.OptionalValidator)
		if err != nil {
			return config, err
		}
		if outputDir == "" {
			outputDir = defaultOutput
		}
	}
	config.OutputDir = outputDir

	// Infrastructure components selection
	if config.Type == interfaces.ProjectTypeMicroservice || config.Type == interfaces.ProjectTypeMonolith {
		terminalUI.PrintSubHeader("Infrastructure Components")

		infrastructure, err := collectInfrastructureConfig(terminalUI)
		if err != nil {
			return config, err
		}
		config.Infrastructure = infrastructure
	}

	// Set defaults
	config.GoVersion = goVersion
	if config.GoVersion == "" {
		config.GoVersion = "1.19"
	}

	return config, nil
}

// collectInfrastructureConfig collects infrastructure configuration.
func collectInfrastructureConfig(terminalUI *ui.TerminalUI) (interfaces.Infrastructure, error) {
	infrastructure := interfaces.Infrastructure{}

	// Database selection
	includeDB, err := terminalUI.PromptConfirm("Include database support?", true)
	if err != nil {
		return infrastructure, err
	}

	if includeDB {
		dbType, err := terminalUI.ShowDatabaseSelectionMenu()
		if err != nil {
			return infrastructure, err
		}
		infrastructure.Database = interfaces.DatabaseConfig{
			Type: dbType,
			Host: "localhost",
			Port: getDefaultDatabasePort(dbType),
		}
	}

	// Cache selection
	includeCache, err := terminalUI.PromptConfirm("Include cache support (Redis)?", false)
	if err != nil {
		return infrastructure, err
	}

	if includeCache {
		infrastructure.Cache = interfaces.CacheConfig{
			Type: "redis",
			Host: "localhost",
			Port: 6379,
		}
	}

	// Message queue selection
	includeMQ, err := terminalUI.PromptConfirm("Include message queue support?", false)
	if err != nil {
		return infrastructure, err
	}

	if includeMQ {
		mqOptions := []interfaces.MenuOption{
			{Label: "RabbitMQ", Description: "Feature-rich message broker", Value: "rabbitmq", Icon: "🐰", Enabled: true},
			{Label: "Apache Kafka", Description: "Distributed streaming platform", Value: "kafka", Icon: "🚀", Enabled: true},
			{Label: "NATS", Description: "Lightweight messaging system", Value: "nats", Icon: "💨", Enabled: true},
		}

		selectedMQIndex, err := terminalUI.ShowMenu("Select message queue type", mqOptions)
		if err != nil {
			return infrastructure, err
		}

		mqType := mqOptions[selectedMQIndex].Value.(string)
		infrastructure.MessageQueue = interfaces.MessageQueueConfig{
			Type: mqType,
			Host: "localhost",
			Port: getDefaultMessageQueuePort(mqType),
		}
	}

	// Service discovery
	includeSD, err := terminalUI.PromptConfirm("Include service discovery (Consul)?", false)
	if err != nil {
		return infrastructure, err
	}

	if includeSD {
		infrastructure.ServiceDiscovery = interfaces.ServiceDiscoveryConfig{
			Type: "consul",
			Host: "localhost",
			Port: 8500,
		}
	}

	// Monitoring
	includeMonitoring, err := terminalUI.PromptConfirm("Include monitoring (Prometheus + Jaeger)?", true)
	if err != nil {
		return infrastructure, err
	}

	if includeMonitoring {
		infrastructure.Monitoring = interfaces.MonitoringConfig{
			Prometheus: interfaces.PrometheusConfig{
				Enabled: true,
				Host:    "localhost",
				Port:    9090,
				Path:    "/metrics",
			},
			Jaeger: interfaces.JaegerConfig{
				Enabled: true,
				Host:    "localhost",
				Port:    14268,
			},
		}
	}

	// Logging
	infrastructure.Logging = interfaces.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	return infrastructure, nil
}

// buildConfigFromFlags builds configuration from command-line flags.
func buildConfigFromFlags() interfaces.ProjectConfig {
	config := interfaces.ProjectConfig{
		Name:        projectName,
		Description: description,
		Author:      author,
		License:     license,
		ModulePath:  modulePath,
		GoVersion:   goVersion,
		OutputDir:   outputDir,
	}

	if projectType != "" {
		config.Type = interfaces.ProjectType(projectType)
	} else {
		config.Type = interfaces.ProjectTypeMicroservice
	}

	// Set output directory if not specified
	if config.OutputDir == "" {
		wd, _ := os.Getwd()
		config.OutputDir = filepath.Join(wd, config.Name)
	}

	return config
}

// validateRequiredConfig validates that required configuration fields are present.
func validateRequiredConfig(config interfaces.ProjectConfig) error {
	if config.Name == "" {
		return fmt.Errorf("project name is required")
	}

	if config.ModulePath == "" {
		return fmt.Errorf("module path is required")
	}

	if config.Author == "" {
		return fmt.Errorf("author is required")
	}

	return nil
}

// showConfigurationPreview displays the configuration preview.
func showConfigurationPreview(ui *ui.TerminalUI, config interfaces.ProjectConfig) error {
	ui.PrintHeader("📋 Configuration Preview")

	// Basic information table
	basicInfo := [][]string{
		{"Project Name", config.Name},
		{"Type", string(config.Type)},
		{"Description", config.Description},
		{"Author", config.Author},
		{"License", config.License},
		{"Module Path", config.ModulePath},
		{"Go Version", config.GoVersion},
		{"Output Directory", config.OutputDir},
	}

	if err := ui.ShowTable([]string{"Setting", "Value"}, basicInfo); err != nil {
		return fmt.Errorf("failed to show basic info table: %w", err)
	}

	// Infrastructure table (if applicable)
	if config.Type == interfaces.ProjectTypeMicroservice || config.Type == interfaces.ProjectTypeMonolith {
		ui.PrintSubHeader("Infrastructure Components")

		infraInfo := [][]string{}

		if config.Infrastructure.Database.Type != "" {
			infraInfo = append(infraInfo, []string{"Database", fmt.Sprintf("%s (%s:%d)",
				config.Infrastructure.Database.Type,
				config.Infrastructure.Database.Host,
				config.Infrastructure.Database.Port)})
		}

		if config.Infrastructure.Cache.Type != "" {
			infraInfo = append(infraInfo, []string{"Cache", fmt.Sprintf("%s (%s:%d)",
				config.Infrastructure.Cache.Type,
				config.Infrastructure.Cache.Host,
				config.Infrastructure.Cache.Port)})
		}

		if config.Infrastructure.MessageQueue.Type != "" {
			infraInfo = append(infraInfo, []string{"Message Queue", fmt.Sprintf("%s (%s:%d)",
				config.Infrastructure.MessageQueue.Type,
				config.Infrastructure.MessageQueue.Host,
				config.Infrastructure.MessageQueue.Port)})
		}

		if config.Infrastructure.ServiceDiscovery.Type != "" {
			infraInfo = append(infraInfo, []string{"Service Discovery", fmt.Sprintf("%s (%s:%d)",
				config.Infrastructure.ServiceDiscovery.Type,
				config.Infrastructure.ServiceDiscovery.Host,
				config.Infrastructure.ServiceDiscovery.Port)})
		}

		if config.Infrastructure.Monitoring.Prometheus.Enabled {
			infraInfo = append(infraInfo, []string{"Monitoring", "Prometheus + Jaeger"})
		}

		if len(infraInfo) > 0 {
			if err := ui.ShowTable([]string{"Component", "Configuration"}, infraInfo); err != nil {
				return fmt.Errorf("failed to show infrastructure table: %w", err)
			}
		}
	}

	return nil
}

// createProject creates the project with the given configuration.
func createProject(ui *ui.TerminalUI, manager interfaces.ProjectManager, config interfaces.ProjectConfig) error {
	if dryRun {
		ui.ShowInfo("Dry run mode: Project structure preview completed")
		return nil
	}

	// Show creation progress
	progress := ui.ShowProgress("Creating project", 5)

	// Step 1: Initialize project structure
	progress.SetMessage("Initializing project structure...")
	if err := manager.InitProject(config); err != nil {
		return fmt.Errorf("failed to initialize project: %w", err)
	}
	progress.Update(1)

	// Step 2: Create directory structure
	progress.SetMessage("Creating directory structure...")
	time.Sleep(500 * time.Millisecond) // Simulate work
	progress.Update(2)

	// Step 3: Generate configuration files
	progress.SetMessage("Generating configuration files...")
	time.Sleep(500 * time.Millisecond) // Simulate work
	progress.Update(3)

	// Step 4: Create template files
	progress.SetMessage("Creating template files...")
	time.Sleep(500 * time.Millisecond) // Simulate work
	progress.Update(4)

	// Step 5: Initialize git repository
	progress.SetMessage("Initializing git repository...")
	time.Sleep(500 * time.Millisecond) // Simulate work
	progress.Update(5)

	progress.Finish()

	// Show success message with next steps
	ui.ShowSuccess(fmt.Sprintf("Project '%s' created successfully!", config.Name))

	ui.PrintSubHeader("Next Steps")
	fmt.Fprintf(os.Stdout, "1. %s\n", ui.GetStyle().Info.Sprint("cd "+config.OutputDir))
	fmt.Fprintf(os.Stdout, "2. %s\n", ui.GetStyle().Info.Sprint("go mod tidy"))
	fmt.Fprintf(os.Stdout, "3. %s\n", ui.GetStyle().Info.Sprint("make build"))
	fmt.Fprintf(os.Stdout, "4. %s\n", ui.GetStyle().Info.Sprint("switctl dev run"))

	return nil
}

// createDependencyContainer creates the dependency injection container.
func createDependencyContainer() (interfaces.DependencyContainer, error) {
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
		return filesystem.NewOSFileSystem(""), nil
	}); err != nil {
		return nil, fmt.Errorf("failed to register filesystem service: %w", err)
	}

	// Register template engine
	if err := container.RegisterSingleton("template_engine", func() (interface{}, error) {
		return template.NewEngine(""), nil
	}); err != nil {
		return nil, fmt.Errorf("failed to register template engine: %w", err)
	}

	// Register project manager
	if err := container.RegisterSingleton("project_manager", func() (interface{}, error) {
		fsService, err := container.GetService("filesystem")
		if err != nil {
			return nil, err
		}

		templateService, err := container.GetService("template_engine")
		if err != nil {
			return nil, err
		}

		return NewProjectManager(
			fsService.(interfaces.FileSystem),
			templateService.(interfaces.TemplateEngine),
		), nil
	}); err != nil {
		return nil, fmt.Errorf("failed to register project manager: %w", err)
	}

	return container, nil
}

// createExecutionContext creates an execution context.
func createExecutionContext(ctx context.Context) *interfaces.ExecutionContext {
	wd, _ := os.Getwd()

	return &interfaces.ExecutionContext{
		WorkDir:    wd,
		ConfigFile: configTemplate,
		Verbose:    verbose,
		NoColor:    noColor,
		Options: map[string]interface{}{
			"output_dir":      outputDir,
			"force":           force,
			"dry_run":         dryRun,
			"interactive":     interactive,
			"config_template": configTemplate,
		},
		Logger:    NewSimpleLogger(verbose),
		StartTime: ctx.Value("start_time").(time.Time),
		Writer:    os.Stdout,
	}
}

// Helper functions

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
		return 0 // No port for SQLite
	default:
		return 3306
	}
}

// getDefaultMessageQueuePort returns the default port for a message queue type.
func getDefaultMessageQueuePort(mqType string) int {
	switch mqType {
	case "rabbitmq":
		return 5672
	case "kafka":
		return 9092
	case "nats":
		return 4222
	default:
		return 5672
	}
}

// SimpleProjectManager provides a basic project manager implementation.
type SimpleProjectManager struct {
	fileSystem     interfaces.FileSystem
	templateEngine interfaces.TemplateEngine
}

// NewProjectManager creates a new project manager.
func NewProjectManager(fs interfaces.FileSystem, te interfaces.TemplateEngine) *SimpleProjectManager {
	return &SimpleProjectManager{
		fileSystem:     fs,
		templateEngine: te,
	}
}

// InitProject initializes a new project.
func (pm *SimpleProjectManager) InitProject(config interfaces.ProjectConfig) error {
	// Create output directory
	if err := pm.fileSystem.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create basic project structure
	dirs := []string{
		"cmd",
		"internal",
		"pkg",
		"api/proto",
		"api/gen",
		"deployments",
		"scripts",
		"docs",
		"configs",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(config.OutputDir, dir)
		if err := pm.fileSystem.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create basic files
	if err := pm.createBasicFiles(config); err != nil {
		return fmt.Errorf("failed to create basic files: %w", err)
	}

	return nil
}

// ValidateProject validates project structure.
func (pm *SimpleProjectManager) ValidateProject() error {
	// TODO: Implement project validation
	return nil
}

// GetProjectInfo returns project information.
func (pm *SimpleProjectManager) GetProjectInfo() (interfaces.ProjectInfo, error) {
	// TODO: Implement project info retrieval
	return interfaces.ProjectInfo{}, nil
}

// UpdateProject updates project configuration.
func (pm *SimpleProjectManager) UpdateProject(updates map[string]interface{}) error {
	// TODO: Implement project updates
	return nil
}

// createBasicFiles creates basic project files.
func (pm *SimpleProjectManager) createBasicFiles(config interfaces.ProjectConfig) error {
	// Create go.mod
	goModContent := fmt.Sprintf("module %s\n\ngo %s\n", config.ModulePath, config.GoVersion)
	if err := pm.fileSystem.WriteFile(filepath.Join(config.OutputDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}

	// Create README.md
	readmeContent := fmt.Sprintf("# %s\n\n%s\n\n## Author\n\n%s\n\n## License\n\n%s\n",
		config.Name, config.Description, config.Author, config.License)
	if err := pm.fileSystem.WriteFile(filepath.Join(config.OutputDir, "README.md"), []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	// Create .gitignore
	gitignoreContent := `# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with ` + "`go test -c`" + `
*.test

# Output of the go coverage tool, specifically when used with LiteIDE
*.out

# Go workspace file
go.work

# Build directories
_output/
bin/
dist/

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS files
.DS_Store
Thumbs.db

# Config files with secrets
*.local.yaml
*.secret.yaml
`
	if err := pm.fileSystem.WriteFile(filepath.Join(config.OutputDir, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

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
