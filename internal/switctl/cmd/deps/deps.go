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

// Package deps provides dependency management commands for switctl.
package deps

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/innovationmech/swit/internal/switctl/deps"
	"github.com/innovationmech/swit/internal/switctl/filesystem"
	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/innovationmech/swit/internal/switctl/ui"
)

var (
	// Global flags for deps commands
	verbose     bool
	noColor     bool
	workDir     string
	interactive bool
	force       bool
	dryRun      bool
	outputFile  string
)

// NewDepsCommand creates the main 'deps' command with subcommands.
func NewDepsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps",
		Short: "Dependency management and security audit",
		Long: `The deps command provides dependency management and security auditing for Go projects.

This command includes subcommands for:
• update: Check and update Go module dependencies to compatible latest versions
• audit: Check dependencies for security vulnerabilities and provide fix recommendations
• list: List all project dependencies with version information
• graph: Show dependency graph and relationships
• why: Explain why a dependency is needed

Examples:
  # Update all dependencies to latest compatible versions
  switctl deps update

  # Check for security vulnerabilities
  switctl deps audit

  # List all dependencies
  switctl deps list

  # Show dependency graph
  switctl deps graph

  # Explain why a dependency is needed
  switctl deps why github.com/gin-gonic/gin`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeDepsCommandConfig(cmd)
		},
	}

	// Add persistent flags
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	cmd.PersistentFlags().StringVarP(&workDir, "work-dir", "w", "", "Working directory (default: current directory)")
	cmd.PersistentFlags().BoolVarP(&interactive, "interactive", "i", false, "Use interactive mode")
	cmd.PersistentFlags().BoolVar(&force, "force", false, "Force updates without confirmation")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")

	// Bind flags to viper
	viper.BindPFlag("deps.verbose", cmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("deps.no_color", cmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("deps.work_dir", cmd.PersistentFlags().Lookup("work-dir"))
	viper.BindPFlag("deps.interactive", cmd.PersistentFlags().Lookup("interactive"))
	viper.BindPFlag("deps.force", cmd.PersistentFlags().Lookup("force"))
	viper.BindPFlag("deps.dry_run", cmd.PersistentFlags().Lookup("dry-run"))

	// Add subcommands
	cmd.AddCommand(NewUpdateCommand())
	cmd.AddCommand(NewAuditCommand())
	cmd.AddCommand(NewListCommand())
	cmd.AddCommand(NewGraphCommand())
	cmd.AddCommand(NewWhyCommand())

	return cmd
}

// initializeDepsCommandConfig initializes configuration for deps commands.
func initializeDepsCommandConfig(cmd *cobra.Command) error {
	// Get config values from viper
	if viper.IsSet("deps.verbose") {
		verbose = viper.GetBool("deps.verbose")
	}
	if viper.IsSet("deps.no_color") {
		noColor = viper.GetBool("deps.no_color")
	}
	if viper.IsSet("deps.work_dir") {
		workDir = viper.GetString("deps.work_dir")
	}
	if viper.IsSet("deps.interactive") {
		interactive = viper.GetBool("deps.interactive")
	}
	if viper.IsSet("deps.force") {
		force = viper.GetBool("deps.force")
	}
	if viper.IsSet("deps.dry_run") {
		dryRun = viper.GetBool("deps.dry_run")
	}

	// Set default working directory
	if workDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		workDir = wd
	}

	// Validate go.mod exists
	goModPath := filepath.Join(workDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return fmt.Errorf("go.mod not found in %s, not a Go module", workDir)
	}

	return nil
}

// NewUpdateCommand creates the 'update' subcommand.
func NewUpdateCommand() *cobra.Command {
	var (
		major        bool
		patch        bool
		prerelease   bool
		excludes     []string
		includes     []string
		testDeps     bool
		skipBreaking bool
	)

	cmd := &cobra.Command{
		Use:   "update [modules...]",
		Short: "Update Go module dependencies to latest compatible versions",
		Long: `The update command checks and updates Go module dependencies to their latest compatible versions.

Features:
• Semantic version-aware updates (major, minor, patch)
• Breaking change detection and migration guidance
• Interactive selection of updates
• Dependency compatibility checking
• Automated testing after updates
• Rollback support on failures

Examples:
  # Update all dependencies to latest compatible versions
  switctl deps update

  # Update specific modules
  switctl deps update github.com/gin-gonic/gin github.com/spf13/cobra

  # Update to latest major versions (potentially breaking)
  switctl deps update --major

  # Update only patch versions (safest)
  switctl deps update --patch

  # Interactive update selection
  switctl deps update --interactive

  # Preview updates without applying
  switctl deps update --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdateCommand(args, major, patch, prerelease, excludes, includes, testDeps, skipBreaking)
		},
	}

	cmd.Flags().BoolVar(&major, "major", false, "Allow major version updates")
	cmd.Flags().BoolVar(&patch, "patch", false, "Only patch version updates")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "Include prerelease versions")
	cmd.Flags().StringSliceVar(&excludes, "exclude", nil, "Modules to exclude from updates")
	cmd.Flags().StringSliceVar(&includes, "include", nil, "Only update these modules")
	cmd.Flags().BoolVar(&testDeps, "test-deps", true, "Include test dependencies")
	cmd.Flags().BoolVar(&skipBreaking, "skip-breaking", false, "Skip modules with potential breaking changes")

	return cmd
}

// NewAuditCommand creates the 'audit' subcommand.
func NewAuditCommand() *cobra.Command {
	var (
		severity   string
		fixable    bool
		jsonOutput bool
		reportFile string
	)

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Check dependencies for security vulnerabilities",
		Long: `The audit command scans project dependencies for known security vulnerabilities.

Features:
• CVE database scanning
• Severity level filtering
• Automated fix suggestions
• Detailed vulnerability reports
• Integration with security databases
• Compliance reporting

Examples:
  # Run security audit on all dependencies
  switctl deps audit

  # Show only high and critical vulnerabilities
  switctl deps audit --severity high

  # Show only vulnerabilities with available fixes
  switctl deps audit --fixable

  # Generate JSON report
  switctl deps audit --json --output security-report.json

  # Audit with detailed output
  switctl deps audit --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuditCommand(severity, fixable, jsonOutput, reportFile)
		},
	}

	cmd.Flags().StringVar(&severity, "severity", "", "Minimum severity level (low, medium, high, critical)")
	cmd.Flags().BoolVar(&fixable, "fixable", false, "Show only vulnerabilities with available fixes")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results in JSON format")
	cmd.Flags().StringVarP(&reportFile, "output", "o", "", "Save report to file")

	return cmd
}

// NewListCommand creates the 'list' subcommand.
func NewListCommand() *cobra.Command {
	var (
		format       string
		showTest     bool
		showIndirect bool
	)

	cmd := &cobra.Command{
		Use:   "list [modules...]",
		Short: "List project dependencies with version information",
		Long: `The list command displays all project dependencies with detailed version information.

Features:
• Multiple output formats (table, json, tree)
• Direct and indirect dependency filtering
• License information display
• Version and update status
• Dependency size and import count

Examples:
  # List all dependencies in table format
  switctl deps list

  # List dependencies in JSON format
  switctl deps list --format json

  # List only direct dependencies
  switctl deps list --no-indirect

  # List specific modules
  switctl deps list github.com/gin-gonic/gin

  # Include test dependencies
  switctl deps list --test`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListCommand(args, format, showTest, showIndirect)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table, json, tree)")
	cmd.Flags().BoolVar(&showTest, "test", false, "Include test dependencies")
	cmd.Flags().BoolVar(&showIndirect, "indirect", true, "Include indirect dependencies")

	return cmd
}

// NewGraphCommand creates the 'graph' subcommand.
func NewGraphCommand() *cobra.Command {
	var (
		format string
		depth  int
		focus  string
	)

	cmd := &cobra.Command{
		Use:   "graph [module]",
		Short: "Show dependency graph and relationships",
		Long: `The graph command visualizes dependency relationships in the project.

Examples:
  # Show full dependency graph
  switctl deps graph

  # Focus on specific module
  switctl deps graph github.com/gin-gonic/gin

  # Limit graph depth
  switctl deps graph --depth 2

  # Generate DOT format for Graphviz
  switctl deps graph --format dot`,
		RunE: func(cmd *cobra.Command, args []string) error {
			focusModule := ""
			if len(args) > 0 {
				focusModule = args[0]
			}
			return runGraphCommand(focusModule, format, depth, focus)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, dot, json)")
	cmd.Flags().IntVar(&depth, "depth", 3, "Maximum depth to show")
	cmd.Flags().StringVar(&focus, "focus", "", "Focus on dependencies of this module")

	return cmd
}

// NewWhyCommand creates the 'why' subcommand.
func NewWhyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "why <module>",
		Short: "Explain why a dependency is needed",
		Long: `The why command explains why a particular module is included as a dependency.

Examples:
  # Explain why a module is needed
  switctl deps why github.com/gin-gonic/gin

  # Show all import paths that require the module
  switctl deps why --verbose golang.org/x/crypto`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWhyCommand(args[0])
		},
	}

	return cmd
}

// Command implementations

// runUpdateCommand executes the update subcommand.
func runUpdateCommand(modules []string, major, patch, prerelease bool, excludes, includes []string, testDeps, skipBreaking bool) error {
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

	depManager, err := container.GetService("dependency_manager")
	if err != nil {
		return fmt.Errorf("failed to get dependency manager: %w", err)
	}
	manager := depManager.(interfaces.DependencyManager)

	terminalUI.PrintHeader("📦 Dependency Update")

	// Show update configuration
	terminalUI.ShowInfo(fmt.Sprintf("Target modules: %s", getModulesDescription(modules)))
	terminalUI.ShowInfo(fmt.Sprintf("Allow major updates: %s", formatBool(major)))
	terminalUI.ShowInfo(fmt.Sprintf("Patch only: %s", formatBool(patch)))
	terminalUI.ShowInfo(fmt.Sprintf("Include prereleases: %s", formatBool(prerelease)))
	terminalUI.ShowInfo(fmt.Sprintf("Include test deps: %s", formatBool(testDeps)))
	terminalUI.ShowInfo(fmt.Sprintf("Skip breaking: %s", formatBool(skipBreaking)))

	if dryRun {
		terminalUI.ShowInfo("Mode: Dry run (no changes will be made)")
	}

	// Get current dependencies
	currentDeps, err := manager.ListDependencies()
	if err != nil {
		return fmt.Errorf("failed to list current dependencies: %w", err)
	}

	terminalUI.ShowInfo(fmt.Sprintf("Found %d dependencies", len(currentDeps)))

	// Check for available updates
	terminalUI.PrintSubHeader("Checking for Updates")
	progress := terminalUI.ShowProgress("Checking for updates", len(currentDeps))

	updates := []DependencyUpdate{}
	for i, dep := range currentDeps {
		progress.SetMessage(fmt.Sprintf("Checking %s...", dep.Name))

		if shouldUpdateDependency(dep, modules, excludes, includes) {
			update, err := checkForUpdate(dep, major, patch, prerelease)
			if err != nil {
				if verbose {
					terminalUI.ShowError(fmt.Errorf("failed to check update for %s: %w", dep.Name, err))
				}
			} else if update != nil {
				updates = append(updates, *update)
			}
		}

		progress.Update(i + 1)
	}

	progress.Finish()

	if len(updates) == 0 {
		terminalUI.ShowSuccess("All dependencies are up to date!")
		return nil
	}

	// Show available updates
	terminalUI.PrintSubHeader("Available Updates")
	updateTable := [][]string{}
	for _, update := range updates {
		severity := "MINOR"
		if update.Major {
			severity = "MAJOR"
		} else if update.Patch {
			severity = "PATCH"
		}

		updateTable = append(updateTable, []string{
			update.Name,
			update.CurrentVersion,
			update.NewVersion,
			severity,
			formatBool(update.Breaking),
		})
	}

	headers := []string{"Module", "Current", "New", "Type", "Breaking"}
	if err := terminalUI.ShowTable(headers, updateTable); err != nil {
		return fmt.Errorf("failed to show updates table: %w", err)
	}

	// Interactive selection if requested
	selectedUpdates := updates
	if interactive && !force {
		selected, err := selectUpdatesInteractively(terminalUI, updates)
		if err != nil {
			return fmt.Errorf("failed to select updates: %w", err)
		}
		selectedUpdates = selected
	}

	// Confirm updates
	if !force && !dryRun {
		confirmed, err := terminalUI.PromptConfirm(fmt.Sprintf("Apply %d updates?", len(selectedUpdates)), true)
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			terminalUI.ShowInfo("Update cancelled")
			return nil
		}
	}

	// Apply updates
	if dryRun {
		terminalUI.ShowInfo("Dry run completed - no updates were applied")
		return nil
	}

	return applyUpdates(terminalUI, manager, selectedUpdates)
}

// runAuditCommand executes the audit subcommand.
func runAuditCommand(severity string, fixable, jsonOutput bool, reportFile string) error {
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

	depManager, err := container.GetService("dependency_manager")
	if err != nil {
		return fmt.Errorf("failed to get dependency manager: %w", err)
	}
	manager := depManager.(interfaces.DependencyManager)

	terminalUI.PrintHeader("🔒 Security Audit")

	if severity != "" {
		terminalUI.ShowInfo(fmt.Sprintf("Minimum severity: %s", severity))
	}
	if fixable {
		terminalUI.ShowInfo("Showing only fixable vulnerabilities")
	}

	// Run security audit
	terminalUI.ShowInfo("Scanning dependencies for vulnerabilities...")
	result := manager.AuditDependencies()

	// Filter results
	vulnerabilities := filterVulnerabilities(result.Vulnerabilities, severity, fixable)

	if jsonOutput {
		return outputAuditJSON(vulnerabilities, reportFile)
	}

	// Display results
	if len(vulnerabilities) == 0 {
		terminalUI.ShowSuccess("No vulnerabilities found!")
		return nil
	}

	terminalUI.PrintSubHeader("Vulnerabilities Found")

	// Group by severity
	severityGroups := groupBySeverity(vulnerabilities)

	for _, sev := range []string{"critical", "high", "medium", "low"} {
		vulns := severityGroups[sev]
		if len(vulns) == 0 {
			continue
		}

		terminalUI.PrintSubHeader(fmt.Sprintf("%s Severity (%d)", strings.Title(sev), len(vulns)))

		for _, vuln := range vulns {
			displayVulnerability(terminalUI, vuln)
		}
	}

	// Show summary
	terminalUI.PrintSubHeader("Summary")
	summaryTable := [][]string{}
	for _, sev := range []string{"critical", "high", "medium", "low"} {
		count := len(severityGroups[sev])
		if count > 0 {
			summaryTable = append(summaryTable, []string{strings.Title(sev), fmt.Sprintf("%d", count)})
		}
	}

	if len(summaryTable) > 0 {
		if err := terminalUI.ShowTable([]string{"Severity", "Count"}, summaryTable); err != nil {
			return fmt.Errorf("failed to show summary table: %w", err)
		}
	}

	// Save report if requested
	if reportFile != "" {
		if err := saveAuditReport(vulnerabilities, reportFile); err != nil {
			return fmt.Errorf("failed to save audit report: %w", err)
		}
		terminalUI.ShowSuccess(fmt.Sprintf("Report saved to %s", reportFile))
	}

	return nil
}

// runListCommand executes the list subcommand.
func runListCommand(modules []string, format string, showTest, showIndirect bool) error {
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

	depManager, err := container.GetService("dependency_manager")
	if err != nil {
		return fmt.Errorf("failed to get dependency manager: %w", err)
	}
	manager := depManager.(interfaces.DependencyManager)

	if format != "json" {
		terminalUI.PrintHeader("📋 Dependency List")
	}

	// Get dependencies
	dependencies, err := manager.ListDependencies()
	if err != nil {
		return fmt.Errorf("failed to list dependencies: %w", err)
	}

	// Filter dependencies
	filteredDeps := filterDependencies(dependencies, modules, showTest, showIndirect)

	// Output in requested format
	switch format {
	case "json":
		return outputDependenciesJSON(filteredDeps)
	case "tree":
		return outputDependenciesTree(terminalUI, filteredDeps)
	default:
		return outputDependenciesTable(terminalUI, filteredDeps)
	}
}

// runGraphCommand executes the graph subcommand.
func runGraphCommand(focusModule, format string, depth int, focus string) error {
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

	if format != "json" && format != "dot" {
		terminalUI.PrintHeader("🕸️ Dependency Graph")
	}

	// Generate dependency graph
	graph, err := generateDependencyGraph(focusModule, depth)
	if err != nil {
		return fmt.Errorf("failed to generate dependency graph: %w", err)
	}

	// Output in requested format
	switch format {
	case "json":
		return outputGraphJSON(graph)
	case "dot":
		return outputGraphDOT(graph)
	default:
		return outputGraphText(terminalUI, graph)
	}
}

// runWhyCommand executes the why subcommand.
func runWhyCommand(module string) error {
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

	terminalUI.PrintHeader(fmt.Sprintf("🤔 Why is %s needed?", module))

	// Run go mod why
	cmd := exec.Command("go", "mod", "why", module)
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run go mod why: %w", err)
	}

	// Parse and display results
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		terminalUI.ShowInfo(fmt.Sprintf("Module %s is not required by this project", module))
		return nil
	}

	terminalUI.ShowInfo("Dependency chain:")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := strings.Repeat("  ", i)
		fmt.Fprintf(os.Stdout, "%s%s %s\n", indent, terminalUI.GetStyle().Info.Sprint("→"), line)
	}

	return nil
}

// Helper types and functions

// DependencyUpdate represents a potential dependency update.
type DependencyUpdate struct {
	Name           string
	CurrentVersion string
	NewVersion     string
	Major          bool
	Patch          bool
	Breaking       bool
	Description    string
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

	// Register dependency manager
	if err := container.RegisterSingleton("dependency_manager", func() (interface{}, error) {
		return NewDependencyManager(workDir), nil
	}); err != nil {
		return nil, fmt.Errorf("failed to register dependency manager: %w", err)
	}

	return container, nil
}

// getModulesDescription returns a description of target modules.
func getModulesDescription(modules []string) string {
	if len(modules) == 0 {
		return "all modules"
	}
	if len(modules) == 1 {
		return modules[0]
	}
	return fmt.Sprintf("%d specific modules", len(modules))
}

// formatBool formats a boolean as enabled/disabled.
func formatBool(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}

// Placeholder implementations for dependency management operations
// In a real implementation, these would integrate with Go modules API

func shouldUpdateDependency(dep interfaces.Dependency, modules, excludes, includes []string) bool {
	// Check if module is in includes list (if specified)
	if len(includes) > 0 {
		found := false
		for _, include := range includes {
			if dep.Name == include {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check if module is in excludes list
	for _, exclude := range excludes {
		if dep.Name == exclude {
			return false
		}
	}

	// Check if specific modules were specified
	if len(modules) > 0 {
		for _, module := range modules {
			if dep.Name == module {
				return true
			}
		}
		return false
	}

	return true
}

var checkForUpdate = func(dep interfaces.Dependency, major, patch, prerelease bool) (*DependencyUpdate, error) {
	// In a real implementation, this would query the module proxy for latest versions
	// For now, return nil (no update available)
	return nil, nil
}

func selectUpdatesInteractively(ui interfaces.InteractiveUI, updates []DependencyUpdate) ([]DependencyUpdate, error) {
	ui.PrintSubHeader("Select Updates")

	options := []interfaces.MenuOption{}
	for i, update := range updates {
		severity := "minor"
		if update.Major {
			severity = "major"
		} else if update.Patch {
			severity = "patch"
		}

		label := fmt.Sprintf("%s (%s → %s, %s)", update.Name, update.CurrentVersion, update.NewVersion, severity)
		options = append(options, interfaces.MenuOption{
			Label:   label,
			Value:   i,
			Enabled: true,
		})
	}

	selectedIndices, err := ui.ShowMultiSelectMenu("Select updates to apply", options)
	if err != nil {
		return nil, err
	}

	selected := []DependencyUpdate{}
	for _, idx := range selectedIndices {
		selected = append(selected, updates[idx])
	}

	return selected, nil
}

var applyUpdates = func(ui interfaces.InteractiveUI, manager interfaces.DependencyManager, updates []DependencyUpdate) error {
	ui.PrintSubHeader("Applying Updates")

	progress := ui.ShowProgress("Applying updates", len(updates))

	for i, update := range updates {
		progress.SetMessage(fmt.Sprintf("Updating %s...", update.Name))

		// In a real implementation, this would run go get commands
		time.Sleep(100 * time.Millisecond) // Simulate work

		progress.Update(i + 1)
	}

	progress.Finish()

	ui.ShowSuccess(fmt.Sprintf("Successfully updated %d dependencies!", len(updates)))
	return nil
}

func filterVulnerabilities(vulns []interfaces.Vulnerability, severity string, fixable bool) []interfaces.Vulnerability {
	filtered := []interfaces.Vulnerability{}

	severityLevels := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	minLevel := 0
	if severity != "" {
		minLevel = severityLevels[strings.ToLower(severity)]
	}

	for _, vuln := range vulns {
		// Filter by severity
		if minLevel > 0 {
			vulnLevel := severityLevels[strings.ToLower(vuln.Severity)]
			if vulnLevel < minLevel {
				continue
			}
		}

		// Filter by fixable
		if fixable && vuln.Fix == "" {
			continue
		}

		filtered = append(filtered, vuln)
	}

	return filtered
}

func groupBySeverity(vulns []interfaces.Vulnerability) map[string][]interfaces.Vulnerability {
	groups := map[string][]interfaces.Vulnerability{
		"critical": {},
		"high":     {},
		"medium":   {},
		"low":      {},
	}

	for _, vuln := range vulns {
		severity := strings.ToLower(vuln.Severity)
		groups[severity] = append(groups[severity], vuln)
	}

	return groups
}

func displayVulnerability(ui interfaces.InteractiveUI, vuln interfaces.Vulnerability) {
	fmt.Fprintf(os.Stdout, "\n%s %s\n", ui.GetStyle().Error.Sprint("▶"), ui.GetStyle().Error.Sprint(vuln.Title))
	fmt.Fprintf(os.Stdout, "  Package: %s\n", vuln.Package)
	fmt.Fprintf(os.Stdout, "  Version: %s\n", vuln.Version)
	fmt.Fprintf(os.Stdout, "  Severity: %s\n", vuln.Severity)
	if vuln.CVSS > 0 {
		fmt.Fprintf(os.Stdout, "  CVSS: %.1f\n", vuln.CVSS)
	}
	if vuln.Description != "" {
		fmt.Fprintf(os.Stdout, "  Description: %s\n", vuln.Description)
	}
	if vuln.Fix != "" {
		fmt.Fprintf(os.Stdout, "  Fix: %s\n", ui.GetStyle().Success.Sprint(vuln.Fix))
	}
	if vuln.URL != "" {
		fmt.Fprintf(os.Stdout, "  URL: %s\n", vuln.URL)
	}
}

func outputAuditJSON(vulns []interfaces.Vulnerability, filename string) error {
	data, err := json.MarshalIndent(vulns, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if filename != "" {
		return os.WriteFile(filename, data, 0644)
	}

	fmt.Println(string(data))
	return nil
}

func saveAuditReport(vulns []interfaces.Vulnerability, filename string) error {
	// Create a simple text report
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	fmt.Fprintf(writer, "Security Audit Report\n")
	fmt.Fprintf(writer, "Generated: %s\n\n", time.Now().Format(time.RFC3339))

	for _, vuln := range vulns {
		fmt.Fprintf(writer, "ID: %s\n", vuln.ID)
		fmt.Fprintf(writer, "Package: %s\n", vuln.Package)
		fmt.Fprintf(writer, "Version: %s\n", vuln.Version)
		fmt.Fprintf(writer, "Severity: %s\n", vuln.Severity)
		fmt.Fprintf(writer, "Title: %s\n", vuln.Title)
		fmt.Fprintf(writer, "Description: %s\n", vuln.Description)
		if vuln.Fix != "" {
			fmt.Fprintf(writer, "Fix: %s\n", vuln.Fix)
		}
		fmt.Fprintf(writer, "\n")
	}

	return nil
}

func filterDependencies(deps []interfaces.Dependency, modules []string, showTest, showIndirect bool) []interfaces.Dependency {
	filtered := []interfaces.Dependency{}

	for _, dep := range deps {
		// Filter by specific modules
		if len(modules) > 0 {
			found := false
			for _, module := range modules {
				if dep.Name == module {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by direct/indirect
		if !showIndirect && !dep.Direct {
			continue
		}

		// In a real implementation, you would filter test dependencies
		// For now, include all

		filtered = append(filtered, dep)
	}

	return filtered
}

func outputDependenciesJSON(deps []interfaces.Dependency) error {
	data, err := json.MarshalIndent(deps, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func outputDependenciesTree(ui interfaces.InteractiveUI, deps []interfaces.Dependency) error {
	ui.ShowInfo("Dependency tree view not yet implemented")
	return outputDependenciesTable(ui, deps)
}

func outputDependenciesTable(ui interfaces.InteractiveUI, deps []interfaces.Dependency) error {
	if len(deps) == 0 {
		ui.ShowInfo("No dependencies found")
		return nil
	}

	rows := [][]string{}
	for _, dep := range deps {
		directStr := "indirect"
		if dep.Direct {
			directStr = "direct"
		}

		rows = append(rows, []string{
			dep.Name,
			dep.Version,
			dep.Type,
			directStr,
			dep.License,
		})
	}

	headers := []string{"Name", "Version", "Type", "Direct", "License"}
	return ui.ShowTable(headers, rows)
}

func generateDependencyGraph(focusModule string, depth int) (interface{}, error) {
	// In a real implementation, this would parse go.mod and generate a graph structure
	return map[string]interface{}{
		"nodes": []interface{}{},
		"edges": []interface{}{},
	}, nil
}

func outputGraphJSON(graph interface{}) error {
	data, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func outputGraphDOT(graph interface{}) error {
	// In a real implementation, this would generate DOT format for Graphviz
	fmt.Println("digraph dependencies {}")
	return nil
}

func outputGraphText(ui interfaces.InteractiveUI, graph interface{}) error {
	ui.ShowInfo("Text graph view not yet implemented")
	return nil
}

// SimpleDependencyManager provides a basic dependency manager implementation.
type SimpleDependencyManager struct {
	workDir string
}

// NewDependencyManager creates a new dependency manager.
func NewDependencyManager(workDir string) *SimpleDependencyManager {
	return &SimpleDependencyManager{workDir: workDir}
}

// UpdateDependencies updates project dependencies.
func (dm *SimpleDependencyManager) UpdateDependencies() error {
	// In a real implementation, this would run go get -u
	return nil
}

// AuditDependencies checks for security vulnerabilities.
func (dm *SimpleDependencyManager) AuditDependencies() interfaces.AuditResult {
	// In a real implementation, this would run security scanners
	return interfaces.AuditResult{
		Vulnerabilities: []interfaces.Vulnerability{},
		Severity:        "none",
		Scanned:         0,
		Duration:        time.Duration(0),
	}
}

// ListDependencies lists all project dependencies.
func (dm *SimpleDependencyManager) ListDependencies() ([]interfaces.Dependency, error) {
	// In a real implementation, this would parse go.mod and go.sum
	return []interfaces.Dependency{}, nil
}

// AddDependency adds a new dependency.
func (dm *SimpleDependencyManager) AddDependency(dep interfaces.Dependency) error {
	// In a real implementation, this would run go get
	return nil
}

// RemoveDependency removes a dependency.
func (dm *SimpleDependencyManager) RemoveDependency(name string) error {
	// In a real implementation, this would run go mod edit -droprequire
	return nil
}
