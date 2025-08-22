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

package version

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/innovationmech/swit/internal/switctl/config"
	"github.com/innovationmech/swit/internal/switctl/deps"
	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/innovationmech/swit/internal/switctl/ui"
	"github.com/innovationmech/swit/internal/switctl/version"
)

var (
	// Global flags for version commands
	verbose     bool
	noColor     bool
	workDir     string
	showDetails bool
	checkUpdate bool
)

// NewSwitctlVersionCmd creates the main 'version' command.
func NewSwitctlVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Long:  "",
		Run: func(cmd *cobra.Command, args []string) {
			runSimpleVersionCommand()
		},
	}

	return cmd
}

// runSimpleVersionCommand executes the simple version command.
func runSimpleVersionCommand() {
	fmt.Println("switctl version 1.0.0")
}

// initializeVersionCommandConfig initializes configuration for version commands.
func initializeVersionCommandConfig(cmd *cobra.Command) error {
	// Set default working directory
	if workDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		workDir = wd
	}

	return nil
}

// runVersionCommand executes the main version command.
func runVersionCommand() error {
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

	versionManager, err := container.GetService("version_manager")
	if err != nil {
		return fmt.Errorf("failed to get version manager: %w", err)
	}
	manager := versionManager.(interfaces.VersionManager)

	// Show version information
	currentVersion, err := manager.GetCurrentVersion()
	if err != nil {
		terminalUI.ShowError(fmt.Errorf("failed to get current version: %w", err))
		currentVersion = "unknown"
	}

	if showDetails || verbose {
		terminalUI.PrintHeader("📋 Version Information")

		// Show detailed version information
		return showDetailedVersionInfo(terminalUI, manager)
	} else {
		// Show basic version information
		fmt.Printf("switctl version %s\n", currentVersion)

		if checkUpdate {
			// Check for updates
			latestVersion, err := manager.GetLatestVersion()
			if err != nil {
				terminalUI.ShowError(fmt.Errorf("failed to check for updates: %w", err))
			} else if currentVersion != latestVersion {
				terminalUI.ShowInfo(fmt.Sprintf("Update available: %s → %s", currentVersion, latestVersion))
				terminalUI.ShowInfo("Run 'switctl version upgrade' for upgrade guidance")
			} else {
				terminalUI.ShowSuccess("You are using the latest version")
			}
		}
	}

	return nil
}

// NewVersionInfoCommand creates the 'info' subcommand.
func NewVersionInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show detailed version information",
		Long: `The info command displays comprehensive version information including:
• Current framework version
• Go version compatibility
• Available features
• Dependencies
• Release information`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersionInfoCommand()
		},
	}
}

// NewVersionCheckCommand creates the 'check' subcommand.
func NewVersionCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check version compatibility",
		Long: `The check command performs version compatibility analysis including:
• Framework version compatibility
• Breaking changes detection
• Deprecation warnings
• Migration requirements`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersionCheckCommand()
		},
	}
}

// NewVersionUpgradeCommand creates the 'upgrade' subcommand.
func NewVersionUpgradeCommand() *cobra.Command {
	var targetVersion string
	var autoMigrate bool

	cmd := &cobra.Command{
		Use:   "upgrade [target-version]",
		Short: "Guide through version upgrades",
		Long: `The upgrade command provides upgrade guidance and migration assistance:
• Migration path analysis
• Step-by-step upgrade instructions
• Automatic migration (when possible)
• Rollback capabilities`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				targetVersion = args[0]
			}
			return runVersionUpgradeCommand(targetVersion, autoMigrate)
		},
	}

	cmd.Flags().StringVarP(&targetVersion, "target", "t", "", "Target version for upgrade")
	cmd.Flags().BoolVar(&autoMigrate, "auto", false, "Perform automatic migration (when possible)")

	return cmd
}

// NewVersionListCommand creates the 'list' subcommand.
func NewVersionListCommand() *cobra.Command {
	var showAll bool
	var showPrerelease bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available versions",
		Long: `The list command shows available framework versions:
• Stable releases
• Beta versions (with --prerelease)
• All versions (with --all)
• Version stability information`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersionListCommand(showAll, showPrerelease)
		},
	}

	cmd.Flags().BoolVar(&showAll, "all", false, "Show all versions including deprecated")
	cmd.Flags().BoolVar(&showPrerelease, "prerelease", false, "Include prerelease versions")

	return cmd
}

// Command implementations

func runVersionInfoCommand() error {
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

	versionManager, err := container.GetService("version_manager")
	if err != nil {
		return fmt.Errorf("failed to get version manager: %w", err)
	}
	manager := versionManager.(interfaces.VersionManager)

	terminalUI.PrintHeader("📋 Detailed Version Information")

	return showDetailedVersionInfo(terminalUI, manager)
}

func runVersionCheckCommand() error {
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

	versionManager, err := container.GetService("version_manager")
	if err != nil {
		return fmt.Errorf("failed to get version manager: %w", err)
	}
	manager := versionManager.(interfaces.VersionManager)

	terminalUI.PrintHeader("🔍 Version Compatibility Check")

	// Perform compatibility check
	result := manager.CheckCompatibility()

	// Display results
	if result.Compatible {
		terminalUI.ShowSuccess("Framework version is compatible")
	} else {
		terminalUI.ShowError(fmt.Errorf("compatibility issues detected"))
	}

	// Show version information
	terminalUI.ShowInfo(fmt.Sprintf("Current version: %s", result.CurrentVersion))
	terminalUI.ShowInfo(fmt.Sprintf("Latest version: %s", result.LatestVersion))

	// Show issues if any
	if len(result.Issues) > 0 {
		terminalUI.PrintSubHeader("Issues Found")

		for _, issue := range result.Issues {
			icon := "⚠️"
			if issue.Breaking {
				icon = "💥"
			}

			fmt.Printf("%s %s: %s\n", icon, strings.Title(issue.Type), issue.Message)
			if issue.Component != "" {
				fmt.Printf("   Component: %s\n", issue.Component)
			}
			if issue.Fix != "" {
				fmt.Printf("   Fix: %s\n", issue.Fix)
			}
			fmt.Println()
		}
	}

	// Show migration recommendation
	if result.MigrationNeeded {
		terminalUI.ShowInfo("Migration recommended. Run 'switctl version upgrade' for guidance.")
	}

	return nil
}

func runVersionUpgradeCommand(targetVersion string, autoMigrate bool) error {
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

	versionManager, err := container.GetService("version_manager")
	if err != nil {
		return fmt.Errorf("failed to get version manager: %w", err)
	}
	manager := versionManager.(interfaces.VersionManager)

	terminalUI.PrintHeader("⬆️ Version Upgrade")

	// Get current version
	currentVersion, err := manager.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Determine target version
	if targetVersion == "" {
		latestVersion, err := manager.GetLatestVersion()
		if err != nil {
			return fmt.Errorf("failed to get latest version: %w", err)
		}
		targetVersion = latestVersion
	}

	terminalUI.ShowInfo(fmt.Sprintf("Current version: %s", currentVersion))
	terminalUI.ShowInfo(fmt.Sprintf("Target version: %s", targetVersion))

	if currentVersion == targetVersion {
		terminalUI.ShowSuccess("Already at target version")
		return nil
	}

	// Check if migration is available
	compatibilityManager := manager.(*version.CompatibilityManager)
	migrationPath, err := compatibilityManager.GetMigrationPath(currentVersion, targetVersion)
	if err != nil {
		terminalUI.ShowError(fmt.Errorf("no migration path available: %w", err))
		return err
	}

	// Show migration steps
	terminalUI.PrintSubHeader("Migration Steps")
	for i, step := range migrationPath.Steps {
		icon := "🔧"
		if step.Manual {
			icon = "👤"
		}
		fmt.Printf("%s Step %d: %s\n", icon, i+1, step.Name)
		if step.Description != "" {
			fmt.Printf("   %s\n", step.Description)
		}
	}

	// Ask for confirmation or perform automatic migration
	if autoMigrate {
		terminalUI.ShowInfo("Performing automatic migration...")
		err = manager.MigrateProject(targetVersion)
		if err != nil {
			terminalUI.ShowError(fmt.Errorf("migration failed: %w", err))
			return err
		}
		terminalUI.ShowSuccess("Migration completed successfully")
	} else {
		confirmed, err := terminalUI.PromptConfirm("Proceed with migration?", false)
		if err != nil {
			return err
		}

		if confirmed {
			err = manager.MigrateProject(targetVersion)
			if err != nil {
				terminalUI.ShowError(fmt.Errorf("migration failed: %w", err))
				return err
			}
			terminalUI.ShowSuccess("Migration completed successfully")
		} else {
			terminalUI.ShowInfo("Migration cancelled")
		}
	}

	return nil
}

func runVersionListCommand(showAll, showPrerelease bool) error {
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

	versionManager, err := container.GetService("version_manager")
	if err != nil {
		return fmt.Errorf("failed to get version manager: %w", err)
	}
	manager := versionManager.(interfaces.VersionManager)

	terminalUI.PrintHeader("📋 Available Versions")

	// Get version manager as compatibility manager to access version info
	compatibilityManager := manager.(*version.CompatibilityManager)
	versions := compatibilityManager.ListAvailableVersions()

	// Filter versions based on flags
	filteredVersions := make([]string, 0)
	for _, ver := range versions {
		versionInfo, err := compatibilityManager.GetVersionInfo(ver)
		if err != nil {
			continue
		}

		// Filter based on stability
		if !showAll && versionInfo.Stability == version.StabilityDeprecated {
			continue
		}
		if !showAll && versionInfo.Stability == version.StabilityEOL {
			continue
		}
		if !showPrerelease && (versionInfo.Stability == version.StabilityAlpha || versionInfo.Stability == version.StabilityBeta) {
			continue
		}

		filteredVersions = append(filteredVersions, ver)
	}

	// Display versions in a table
	headers := []string{"Version", "Stability", "Release Date", "Go Version"}
	rows := make([][]string, 0)

	for _, ver := range filteredVersions {
		versionInfo, err := compatibilityManager.GetVersionInfo(ver)
		if err != nil {
			continue
		}

		stability := string(versionInfo.Stability)
		releaseDate := versionInfo.ReleaseDate.Format("2006-01-02")
		goVersion := versionInfo.MinGoVersion

		rows = append(rows, []string{ver, stability, releaseDate, goVersion})
	}

	if len(rows) == 0 {
		terminalUI.ShowInfo("No versions found matching the criteria")
		return nil
	}

	err = terminalUI.ShowTable(headers, rows)
	if err != nil {
		return fmt.Errorf("failed to show version table: %w", err)
	}

	return nil
}

func showDetailedVersionInfo(ui interfaces.InteractiveUI, manager interfaces.VersionManager) error {
	// Get current version
	currentVersion, err := manager.GetCurrentVersion()
	if err != nil {
		ui.ShowError(fmt.Errorf("failed to get current version: %w", err))
		currentVersion = "unknown"
	}

	// Get latest version
	latestVersion, err := manager.GetLatestVersion()
	if err != nil {
		ui.ShowError(fmt.Errorf("failed to get latest version: %w", err))
		latestVersion = "unknown"
	}

	// Basic information
	ui.PrintSubHeader("Basic Information")
	basicInfo := [][]string{
		{"Current Version", currentVersion},
		{"Latest Version", latestVersion},
		{"CLI Tool", "switctl"},
		{"Go Version", "1.23.12"}, // This could be detected dynamically
	}

	if err := ui.ShowTable([]string{"Property", "Value"}, basicInfo); err != nil {
		return fmt.Errorf("failed to show basic info table: %w", err)
	}

	// Version compatibility
	ui.PrintSubHeader("Compatibility Status")
	result := manager.CheckCompatibility()

	status := "✅ Compatible"
	if !result.Compatible {
		status = "❌ Issues detected"
	}

	compatInfo := [][]string{
		{"Status", status},
		{"Migration Needed", formatBool(result.MigrationNeeded)},
		{"Issues Found", fmt.Sprintf("%d", len(result.Issues))},
	}

	if err := ui.ShowTable([]string{"Check", "Result"}, compatInfo); err != nil {
		return fmt.Errorf("failed to show compatibility table: %w", err)
	}

	// Show issues if any
	if len(result.Issues) > 0 {
		ui.PrintSubHeader("Detected Issues")
		for i, issue := range result.Issues {
			fmt.Printf("%d. %s (%s)\n", i+1, issue.Message, issue.Type)
			if issue.Component != "" {
				fmt.Printf("   Component: %s\n", issue.Component)
			}
			if issue.Fix != "" {
				fmt.Printf("   Resolution: %s\n", issue.Fix)
			}
		}
	}

	return nil
}

// Helper functions

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

	// Register config manager
	if err := container.RegisterSingleton("config_manager", func() (interface{}, error) {
		var logger interfaces.Logger
		configManager := config.NewHierarchicalConfigManager(workDir, logger)

		if err := configManager.Load(); err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to load configuration: %v\n", err)
		}

		return configManager, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to register config manager: %w", err)
	}

	// Register version manager
	if err := container.RegisterSingleton("version_manager", func() (interface{}, error) {
		// Get dependencies
		configManagerService, err := container.GetService("config_manager")
		if err != nil {
			return nil, fmt.Errorf("failed to get config manager: %w", err)
		}
		configManager := configManagerService.(interfaces.ConfigManager)

		var logger interfaces.Logger

		return version.NewCompatibilityManager(workDir, logger, configManager), nil
	}); err != nil {
		return nil, fmt.Errorf("failed to register version manager: %w", err)
	}

	return container, nil
}

func formatBool(value bool) string {
	if value {
		return "Yes"
	}
	return "No"
}
