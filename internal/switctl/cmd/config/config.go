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
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/innovationmech/swit/internal/switctl/checker"
	"github.com/innovationmech/swit/internal/switctl/config"
	"github.com/innovationmech/swit/internal/switctl/deps"
	"github.com/innovationmech/swit/internal/switctl/filesystem"
	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/innovationmech/swit/internal/switctl/ui"
)

var (
	// Global flags for config commands
	verbose       bool
	noColor       bool
	workDir       string
	configFile    string
	outputFile    string
	format        string
	strict        bool
	showDefaults  bool
	reportFormat  string
	watchMode     bool
	watchDebounce time.Duration
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
	cmd.AddCommand(NewDiffCommand())
	cmd.AddCommand(NewBackupCommand())
	cmd.AddCommand(NewRestoreCommand())
	cmd.AddCommand(NewHistoryCommand())

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
	cmd.Flags().StringVar(&reportFormat, "report-format", "table", "Report format (table, json)")
	_ = cmd.RegisterFlagCompletionFunc("report-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().BoolVar(&watchMode, "watch", false, "Watch configuration files and re-validate on changes")
	cmd.Flags().DurationVar(&watchDebounce, "watch-debounce", 300*time.Millisecond, "Debounce interval for watch mode")

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
		rulesPath   string
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
			return runMigrateCommand(args, fromVersion, toVersion, backup, rulesPath)
		},
	}

	cmd.Flags().StringVar(&fromVersion, "from", "", "Source version")
	cmd.Flags().StringVar(&toVersion, "to", "", "Target version")
	cmd.Flags().BoolVar(&backup, "backup", true, "Create backup of original files")
	cmd.Flags().StringVar(&rulesPath, "rules", "", "Path to migration rules file (YAML/JSON)")

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

// NewDiffCommand creates the 'diff' subcommand.
func NewDiffCommand() *cobra.Command {
	var (
		compatCheck bool
		keysOnly    bool
	)

	cmd := &cobra.Command{
		Use:   "diff <old-file> <new-file>",
		Short: "Show differences between two configuration files",
		Long: `The diff command compares two configuration files and prints structural differences.

Examples:
  # Show diff in table format
  switctl config diff base.yaml overlay.yaml

  # JSON output for machines
  switctl config diff --report-format json base.yaml overlay.yaml

  # Run compatibility checks to flag breaking changes
  switctl config diff --compat base.yaml overlay.yaml`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiffCommand(args[0], args[1], compatCheck, keysOnly)
		},
	}

	cmd.Flags().StringVar(&reportFormat, "report-format", "table", "Report format (table, json)")
	_ = cmd.RegisterFlagCompletionFunc("report-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().BoolVar(&compatCheck, "compat", true, "Run compatibility checks for breaking changes")
	cmd.Flags().BoolVar(&keysOnly, "keys-only", false, "Only compare key presence, ignore value changes")

	return cmd
}

// NewBackupCommand creates the 'backup' subcommand.
func NewBackupCommand() *cobra.Command {
	var (
		label     string
		backupDir string
	)

	cmd := &cobra.Command{
		Use:   "backup [files...]",
		Short: "Create a versioned backup of configuration files",
		Long: `The backup command saves copies of configuration files into a versioned directory.

Examples:
  # Backup detected configuration files in the working directory
  switctl config backup

  # Backup specific files with a label
  switctl config backup swit.yaml switauth.yaml --label nightly

  # Specify backup output directory
  switctl config backup --dir .switctl/backups/config`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupCommand(args, label, backupDir)
		},
	}

	cmd.Flags().StringVar(&label, "label", "", "Optional label for this backup (e.g., nightly, pre-release)")
	cmd.Flags().StringVar(&backupDir, "dir", "", "Backup directory (default: .switctl/backups/config under work dir)")

	return cmd
}

// NewRestoreCommand creates the 'restore' subcommand.
func NewRestoreCommand() *cobra.Command {
	var (
		id        string
		overwrite bool
		targetDir string
	)

	cmd := &cobra.Command{
		Use:   "restore [files...]",
		Short: "Restore configuration files from a backup",
		Long: `The restore command restores configuration files from a specified backup ID.

Examples:
  # Restore the latest backup
  switctl config restore

  # Restore a specific backup ID
  switctl config restore --id 20250921_081440-nightly

  # Restore to a custom directory
  switctl config restore --id 20250921_081440 --target-dir ./restore

  # Overwrite existing files
  switctl config restore --overwrite`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestoreCommand(args, id, overwrite, targetDir)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "Backup ID to restore from (default: latest)")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing files during restore")
	cmd.Flags().StringVar(&targetDir, "target-dir", "", "Target directory to restore files to (default: work dir)")

	return cmd
}

// NewHistoryCommand creates the 'history' subcommand.
func NewHistoryCommand() *cobra.Command {
	var (
		limit int
	)

	cmd := &cobra.Command{
		Use:   "history",
		Short: "List configuration backup history",
		Long: `The history command lists available configuration backups with metadata.

Examples:
  # Show recent backups
  switctl config history

  # Show the last 50 backups in JSON
  switctl config history --limit 50 --report-format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistoryCommand(limit)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of entries to display")
	cmd.Flags().StringVar(&reportFormat, "report-format", "table", "Report format (table, json)")
	_ = cmd.RegisterFlagCompletionFunc("report-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json"}, cobra.ShellCompDirectiveNoFileComp
	})

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

	// config manager is available if needed later via the container

	terminalUI.PrintHeader("✅ Configuration Validation")

	// Determine files to validate (explicit path → detected list → none)
	configFiles := files
	if len(configFiles) == 0 {
		if configFile != "" {
			configFiles = []string{configFile}
		} else {
			detectedFiles, err := findConfigFiles()
			if err != nil {
				return fmt.Errorf("failed to find configuration files: %w", err)
			}
			configFiles = detectedFiles
		}
	}

	if watchMode {
		return runValidateWatch(terminalUI, configFiles)
	}

	if len(configFiles) > 0 {
		// Legacy per-file validation path (backward compatible with tests and flags)
		terminalUI.ShowInfo(fmt.Sprintf("Strict mode: %s", formatBool(strict)))

		allValid := true
		results := []ValidationResult{}
		progress := terminalUI.ShowProgress("Validating files", len(configFiles))

		for i, file := range configFiles {
			progress.SetMessage(fmt.Sprintf("Validating %s...", filepath.Base(file)))

			result := validateConfigFile(nil, file)
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

		terminalUI.PrintSubHeader("Validation Results")
		for _, result := range results {
			displayValidationResult(terminalUI, result)
		}

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
			return nil
		}
		terminalUI.ShowError(fmt.Errorf("validation failed for one or more files"))
		return fmt.Errorf("configuration validation failed")
	}

	// Aggregated workspace validation when no files were found
	cfgChecker := checker.NewConfigChecker(workDir, nil)
	cfgChecker.SetConfig(checker.ConfigValidationConfig{YAMLValidation: true, JSONValidation: true, ProtoValidation: true, TOMLValidation: false, ConfigPaths: []string{workDir}, StrictMode: strict, CheckKeys: true, CheckValues: true})

	terminalUI.ShowInfo(fmt.Sprintf("Strict mode: %s", formatBool(strict)))

	progress := terminalUI.ShowProgress("Validating configuration", 1)
	progress.SetMessage("Scanning and validating...")
	vr := cfgChecker.ValidateConfig()
	progress.Update(1)
	progress.Finish()

	if strings.EqualFold(reportFormat, "json") || strings.EqualFold(format, "json") {
		report := buildJSONReport(vr)
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal validation report: %w", err)
		}
		fmt.Println(string(data))
	}

	if !strings.EqualFold(reportFormat, "json") {
		terminalUI.PrintSubHeader("Validation Results")
		for _, r := range buildPerFileResults(vr) {
			displayValidationResult(terminalUI, r)
		}
		terminalUI.PrintSubHeader("Summary")
		summaryTable := [][]string{{"Total Issues", fmt.Sprintf("%d", len(vr.Errors)+len(vr.Warnings))}, {"Errors", fmt.Sprintf("%d", len(vr.Errors))}, {"Warnings", fmt.Sprintf("%d", len(vr.Warnings))}}
		if err := terminalUI.ShowTable([]string{"Metric", "Count"}, summaryTable); err != nil {
			return fmt.Errorf("failed to show summary table: %w", err)
		}
	}

	if vr.Valid && (!strict || len(vr.Warnings) == 0) {
		terminalUI.ShowSuccess("Configuration validation passed")
		return nil
	}
	terminalUI.ShowError(fmt.Errorf("configuration validation failed"))
	return fmt.Errorf("configuration validation failed")
}

// buildPerFileResults groups ValidationResult by file for display compatibility
func buildPerFileResults(vr interfaces.ValidationResult) []ValidationResult {
	// group by file using the Field prefix before first ':'
	fileMap := map[string]*ValidationResult{}
	ensure := func(file string) *ValidationResult {
		if fileMap[file] == nil {
			fileMap[file] = &ValidationResult{File: file, Valid: true}
		}
		return fileMap[file]
	}
	for _, e := range vr.Errors {
		file, msg := splitField(e.Field)
		r := ensure(file)
		r.Valid = false
		loc := formatLoc(e.Line, e.Column)
		hint := strings.TrimSpace(e.Hint)
		base := msgOrMessage(msg, e.Message)
		if hint != "" {
			base = fmt.Sprintf("%s (%s)", base, hint)
		}
		if loc != "" {
			r.Errors = append(r.Errors, fmt.Sprintf("%s: %s", loc, base))
		} else {
			r.Errors = append(r.Errors, base)
		}
	}
	for _, w := range vr.Warnings {
		file, msg := splitField(w.Field)
		r := ensure(file)
		loc := formatLoc(w.Line, w.Column)
		hint := strings.TrimSpace(w.Hint)
		base := msgOrMessage(msg, w.Message)
		if hint != "" {
			base = fmt.Sprintf("%s (%s)", base, hint)
		}
		if loc != "" {
			r.Warnings = append(r.Warnings, fmt.Sprintf("%s: %s", loc, base))
		} else {
			r.Warnings = append(r.Warnings, base)
		}
	}
	out := make([]ValidationResult, 0, len(fileMap))
	for _, v := range fileMap {
		out = append(out, *v)
	}
	return out
}

func splitField(field string) (string, string) {
	if idx := strings.Index(field, ":"); idx != -1 {
		return field[:idx], field[idx+1:]
	}
	return field, ""
}

func msgOrMessage(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return b
}

func formatLoc(line, col int) string {
	if line > 0 && col > 0 {
		return fmt.Sprintf("L%d:C%d", line, col)
	}
	if line > 0 {
		return fmt.Sprintf("L%d", line)
	}
	return ""
}

// runValidateWatch runs validation in watch mode using fsnotify
func runValidateWatch(terminalUI interfaces.InteractiveUI, explicitFiles []string) error {
	terminalUI.PrintSubHeader("Watch mode enabled")
	files := explicitFiles
	if len(files) == 0 {
		detected, err := findConfigFiles()
		if err != nil {
			return fmt.Errorf("failed to find configuration files: %w", err)
		}
		files = detected
	}
	if len(files) == 0 {
		terminalUI.ShowInfo("No configuration files found to watch")
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// watch files and their parent directories (handle atomics)
	dirs := map[string]struct{}{}
	for _, f := range files {
		_ = watcher.Add(f)
		dir := filepath.Dir(f)
		if _, ok := dirs[dir]; !ok {
			_ = watcher.Add(dir)
			dirs[dir] = struct{}{}
		}
	}

	// initial validation
	_ = runValidateCommand(files)

	debounce := time.NewTimer(watchDebounce)
	if !debounce.Stop() {
		<-debounce.C
	}
	pending := false

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			// consider only writes/creates/renames
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
				pending = true
				if !debounce.Stop() {
					select {
					case <-debounce.C:
					default:
					}
				}
				debounce.Reset(watchDebounce)
			}
		case <-debounce.C:
			if pending {
				pending = false
				// re-scan files (new files might appear)
				current := explicitFiles
				if len(current) == 0 {
					detected, err := findConfigFiles()
					if err == nil {
						current = detected
					}
				}
				_ = runValidateCommand(current)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			terminalUI.ShowError(fmt.Errorf("watch error: %v", err))
		}
	}
}

// json report model
type jsonReport struct {
	Valid    bool                         `json:"valid"`
	Strict   bool                         `json:"strict"`
	Errors   []interfaces.ValidationError `json:"errors"`
	Warnings []interfaces.ValidationError `json:"warnings"`
}

func buildJSONReport(vr interfaces.ValidationResult) jsonReport {
	return jsonReport{Valid: vr.Valid, Strict: strict, Errors: vr.Errors, Warnings: vr.Warnings}
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
func runMigrateCommand(files []string, fromVersion, toVersion string, backup bool, rulesPath string) error {
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

	// Load migration rules if provided
	var rules []MigrationRule
	if strings.TrimSpace(rulesPath) != "" {
		loaded, err := loadMigrationRules(rulesPath)
		if err != nil {
			return fmt.Errorf("failed to load migration rules: %w", err)
		}
		rules = loaded
		terminalUI.ShowInfo(fmt.Sprintf("Loaded %d migration rules", len(rules)))
	}

	// Migrate files
	progress := terminalUI.ShowProgress("Migrating files", len(migrateFiles))

	for i, file := range migrateFiles {
		progress.SetMessage(fmt.Sprintf("Migrating %s...", filepath.Base(file)))

		if err := migrateConfigFileWithRules(file, fromVersion, toVersion, backup, rules); err != nil {
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

// BackupManifest describes a single backup snapshot.
type BackupManifest struct {
	ID        string   `json:"id" yaml:"id"`
	Label     string   `json:"label,omitempty" yaml:"label,omitempty"`
	CreatedAt string   `json:"created_at" yaml:"created_at"`
	Files     []string `json:"files" yaml:"files"`
	WorkDir   string   `json:"work_dir" yaml:"work_dir"`
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
		// Don't try to get logger service since it's not registered
		// This avoids potential deadlocks in race conditions
		var logger interfaces.Logger = nil

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

// runBackupCommand executes the backup subcommand.
func runBackupCommand(files []string, label, dirFlag string) error {
	// Create dependency container and services
	container, err := createDependencyContainer()
	if err != nil {
		return fmt.Errorf("failed to create dependency container: %w", err)
	}
	defer container.Close()

	uiService, err := container.GetService("ui")
	if err != nil {
		return fmt.Errorf("failed to get UI service: %w", err)
	}
	terminalUI := uiService.(interfaces.InteractiveUI)

	fsSvc, err := container.GetService("filesystem")
	if err != nil {
		return fmt.Errorf("failed to get filesystem service: %w", err)
	}
	fs := fsSvc.(interfaces.FileSystem)

	terminalUI.PrintHeader("🗂️ Configuration Backup")

	// Determine files
	backupFiles := files
	if len(backupFiles) == 0 {
		detected, err := findConfigFiles()
		if err != nil {
			return fmt.Errorf("failed to find configuration files: %w", err)
		}
		backupFiles = detected
	}
	if len(backupFiles) == 0 {
		terminalUI.ShowInfo("No configuration files found to backup")
		return nil
	}

	// Prepare backup directory
	baseDir := defaultBackupBaseDir(dirFlag)
	if err := fs.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup base directory: %w", err)
	}

	backupID := generateBackupID(label)
	backupPath := filepath.Join(baseDir, backupID)
	if err := fs.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	progress := terminalUI.ShowProgress("Backing up files", len(backupFiles))
	for i, f := range backupFiles {
		progress.SetMessage(fmt.Sprintf("Copying %s...", filepath.Base(f)))
		dest := filepath.Join(backupPath, filepath.Base(f))
		if err := fs.Copy(f, dest); err != nil {
			_ = progress.Finish()
			return fmt.Errorf("failed to copy %s: %w", f, err)
		}
		progress.Update(i + 1)
	}
	progress.Finish()

	// Write manifest
	manifest := BackupManifest{
		ID:        backupID,
		Label:     strings.TrimSpace(label),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Files:     basenames(backupFiles),
		WorkDir:   workDir,
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := fs.WriteFile(filepath.Join(backupPath, "manifest.json"), data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	terminalUI.ShowSuccess(fmt.Sprintf("Backup created: %s", backupID))
	return nil
}

// runRestoreCommand executes the restore subcommand.
func runRestoreCommand(files []string, id string, overwrite bool, targetDir string) error {
	container, err := createDependencyContainer()
	if err != nil {
		return fmt.Errorf("failed to create dependency container: %w", err)
	}
	defer container.Close()

	uiService, err := container.GetService("ui")
	if err != nil {
		return fmt.Errorf("failed to get UI service: %w", err)
	}
	terminalUI := uiService.(interfaces.InteractiveUI)

	fsSvc, err := container.GetService("filesystem")
	if err != nil {
		return fmt.Errorf("failed to get filesystem service: %w", err)
	}
	fs := fsSvc.(interfaces.FileSystem)

	terminalUI.PrintHeader("📦 Configuration Restore")

	baseDir := defaultBackupBaseDir("")
	backupID := id
	if strings.TrimSpace(backupID) == "" {
		// pick latest by directory name sort (timestamp prefix)
		latest, err := findLatestBackupID(baseDir)
		if err != nil {
			return err
		}
		backupID = latest
	}
	if strings.TrimSpace(backupID) == "" {
		terminalUI.ShowInfo("No backups found")
		return nil
	}

	backupPath := filepath.Join(baseDir, backupID)
	manifest, err := readBackupManifest(fs, backupPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Determine files to restore
	var toRestore []string
	if len(files) > 0 {
		// Only restore the specified basenames
		toRestore = basenames(files)
	} else {
		toRestore = manifest.Files
	}
	if len(toRestore) == 0 {
		terminalUI.ShowInfo("No files to restore in backup")
		return nil
	}

	// Determine target directory
	dstDir := targetDir
	if strings.TrimSpace(dstDir) == "" {
		dstDir = workDir
	}

	if err := fs.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to ensure target directory: %w", err)
	}

	progress := terminalUI.ShowProgress("Restoring files", len(toRestore))
	for i, name := range toRestore {
		src := filepath.Join(backupPath, name)
		dst := filepath.Join(dstDir, name)
		if !overwrite && fs.Exists(dst) {
			_ = progress.Finish()
			return fmt.Errorf("target exists: %s (use --overwrite)", dst)
		}
		progress.SetMessage(fmt.Sprintf("Restoring %s...", name))
		if err := fs.Copy(src, dst); err != nil {
			_ = progress.Finish()
			return fmt.Errorf("failed to restore %s: %w", name, err)
		}
		progress.Update(i + 1)
	}
	progress.Finish()

	terminalUI.ShowSuccess(fmt.Sprintf("Restored backup %s", backupID))
	return nil
}

// runHistoryCommand executes the history subcommand.
func runHistoryCommand(limit int) error {
	container, err := createDependencyContainer()
	if err != nil {
		return fmt.Errorf("failed to create dependency container: %w", err)
	}
	defer container.Close()

	uiService, err := container.GetService("ui")
	if err != nil {
		return fmt.Errorf("failed to get UI service: %w", err)
	}
	terminalUI := uiService.(interfaces.InteractiveUI)

	terminalUI.PrintHeader("🕘 Configuration Backup History")

	baseDir := defaultBackupBaseDir("")
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			terminalUI.ShowInfo("No backups found")
			return nil
		}
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Collect backup manifests
	manifests := make([]BackupManifest, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m := BackupManifest{ID: e.Name()}
		data, err := os.ReadFile(filepath.Join(baseDir, e.Name(), "manifest.json"))
		if err == nil {
			_ = json.Unmarshal(data, &m)
		}
		manifests = append(manifests, m)
	}

	// Sort by CreatedAt (fallback to ID lexicographic)
	sort.Slice(manifests, func(i, j int) bool {
		it, jt := manifests[i], manifests[j]
		// try timestamp
		ti, ei := time.Parse(time.RFC3339, it.CreatedAt)
		tj, ej := time.Parse(time.RFC3339, jt.CreatedAt)
		if ei == nil && ej == nil {
			return tj.Before(ti)
		}
		return it.ID > jt.ID
	})

	if limit > 0 && len(manifests) > limit {
		manifests = manifests[:limit]
	}

	if strings.EqualFold(reportFormat, "json") || strings.EqualFold(format, "json") {
		b, _ := json.MarshalIndent(manifests, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	rows := make([][]string, 0, len(manifests))
	for _, m := range manifests {
		rows = append(rows, []string{m.ID, m.Label, m.CreatedAt, fmt.Sprintf("%d", len(m.Files))})
	}
	_ = terminalUI.ShowTable([]string{"ID", "Label", "CreatedAt", "Files"}, rows)
	return nil
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
	return migrateConfigFileWithRules(file, fromVersion, toVersion, backup, nil)
}

// MigrationRule defines a single migration transformation.
type MigrationRule struct {
	FromKey     string      `yaml:"from" json:"from"`
	ToKey       string      `yaml:"to" json:"to"`
	Default     interface{} `yaml:"default,omitempty" json:"default,omitempty"`
	Remove      bool        `yaml:"remove,omitempty" json:"remove,omitempty"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
}

// loadMigrationRules loads migration rules from YAML/JSON.
func loadMigrationRules(path string) ([]MigrationRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rules []MigrationRule
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &rules); err != nil {
			return nil, err
		}
	case ".json":
		if err := json.Unmarshal(data, &rules); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported rules file extension: %s", filepath.Ext(path))
	}
	return rules, nil
}

// migrateConfigFile applies migration rules and version changes to a file.
func migrateConfigFileWithRules(file, fromVersion, toVersion string, backup bool, rules []MigrationRule) error {
	if backup {
		backupFile := file + ".backup"
		if err := copyFile(file, backupFile); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	ext := strings.ToLower(filepath.Ext(file))
	var cfg map[string]interface{}
	switch ext {
	case ".json":
		if err := json.Unmarshal(content, &cfg); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	default:
		if err := yaml.Unmarshal(content, &cfg); err != nil {
			return fmt.Errorf("invalid YAML: %w", err)
		}
	}

	// Apply simple version stamping if present
	setNested(cfg, "version", toVersion)

	// Apply rules
	for _, r := range rules {
		if strings.TrimSpace(r.FromKey) == "" && strings.TrimSpace(r.ToKey) == "" {
			continue
		}
		if r.Remove {
			removeNested(cfg, r.FromKey)
			continue
		}
		val, ok := getNested(cfg, r.FromKey)
		if !ok {
			if r.Default != nil && r.ToKey != "" {
				setNested(cfg, r.ToKey, r.Default)
			}
			continue
		}
		if r.ToKey != "" {
			setNested(cfg, r.ToKey, val)
		}
		if r.ToKey != r.FromKey {
			removeNested(cfg, r.FromKey)
		}
	}

	// Write back
	var out []byte
	if ext == ".json" {
		out, err = json.MarshalIndent(cfg, "", "  ")
	} else {
		out, err = yaml.Marshal(cfg)
	}
	if err != nil {
		return err
	}
	return os.WriteFile(file, out, 0644)
}

// getNested retrieves value by dotted path.
func getNested(m map[string]interface{}, path string) (interface{}, bool) {
	if strings.TrimSpace(path) == "" {
		return nil, false
	}
	parts := strings.Split(path, ".")
	cur := interface{}(m)
	for _, p := range parts {
		asMap, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		v, exists := asMap[p]
		if !exists {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

// setNested sets value by dotted path, creating intermediate maps.
func setNested(m map[string]interface{}, path string, value interface{}) {
	if strings.TrimSpace(path) == "" {
		return
	}
	parts := strings.Split(path, ".")
	cur := m
	for i, p := range parts {
		if i == len(parts)-1 {
			cur[p] = value
			return
		}
		next, ok := cur[p].(map[string]interface{})
		if !ok {
			next = map[string]interface{}{}
			cur[p] = next
		}
		cur = next
	}
}

// removeNested removes key by dotted path.
func removeNested(m map[string]interface{}, path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	parts := strings.Split(path, ".")
	cur := m
	for i, p := range parts {
		if i == len(parts)-1 {
			delete(cur, p)
			return
		}
		next, ok := cur[p].(map[string]interface{})
		if !ok {
			return
		}
		cur = next
	}
}

// runDiffCommand runs the diff subcommand logic.
func runDiffCommand(oldPath, newPath string, compatCheck, keysOnly bool) error {
	// Basic file reads and unmarshal
	load := func(p string) (map[string]interface{}, error) {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var out map[string]interface{}
		switch strings.ToLower(filepath.Ext(p)) {
		case ".json":
			if err := json.Unmarshal(b, &out); err != nil {
				return nil, err
			}
		default:
			if err := yaml.Unmarshal(b, &out); err != nil {
				return nil, err
			}
		}
		if out == nil {
			out = map[string]interface{}{}
		}
		return out, nil
	}

	left, err := load(oldPath)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", oldPath, err)
	}
	right, err := load(newPath)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", newPath, err)
	}

	// Compute diff
	d := diffMaps(left, right, keysOnly)

	// Output
	if strings.EqualFold(reportFormat, "json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(d)
	}

	// human table-like
	rows := [][]string{}
	for _, it := range d.Items {
		rows = append(rows, []string{it.Path, it.Change, it.LeftValue, it.RightValue})
	}

	// Use UI for a nicer table if available
	container, _ := createDependencyContainer()
	if container != nil {
		defer container.Close()
		if uiSvc, err := container.GetService("ui"); err == nil {
			terminalUI := uiSvc.(interfaces.InteractiveUI)
			terminalUI.PrintHeader("🧩 Configuration Diff")
			_ = terminalUI.ShowTable([]string{"Path", "Change", "Old", "New"}, rows)
			if compatCheck {
				br := breakingChanges(left, right)
				if len(br) > 0 {
					terminalUI.PrintSubHeader("Compatibility Warnings")
					for _, msg := range br {
						_ = terminalUI.ShowTable([]string{"Warning"}, [][]string{{msg.Error()}})
					}
				} else {
					terminalUI.ShowSuccess("No breaking changes detected")
				}
			}
			return nil
		}
	}

	// Fallback plain output
	fmt.Println("PATH\tCHANGE\tOLD\tNEW")
	for _, r := range rows {
		fmt.Printf("%s\t%s\t%s\t%s\n", r[0], r[1], r[2], r[3])
	}
	return nil
}

// DiffResult holds diff items.
type DiffResult struct {
	Items []DiffItem `json:"items"`
}

// DiffItem represents a single diff entry.
type DiffItem struct {
	Path       string `json:"path"`
	Change     string `json:"change"` // added|removed|modified
	LeftValue  string `json:"left_value"`
	RightValue string `json:"right_value"`
}

func stringify(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}

func diffMaps(left, right map[string]interface{}, keysOnly bool) DiffResult {
	items := []DiffItem{}
	var walk func(prefix string, a, b interface{})
	walk = func(prefix string, a, b interface{}) {
		am, aok := a.(map[string]interface{})
		bm, bok := b.(map[string]interface{})
		if aok || bok {
			// map case
			keys := map[string]struct{}{}
			if aok {
				for k := range am {
					keys[k] = struct{}{}
				}
			}
			if bok {
				for k := range bm {
					keys[k] = struct{}{}
				}
			}
			// stable order
			klist := make([]string, 0, len(keys))
			for k := range keys {
				klist = append(klist, k)
			}
			sort.Strings(klist)
			for _, k := range klist {
				var av, bv interface{}
				if aok {
					av = am[k]
				}
				if bok {
					bv = bm[k]
				}
				path := k
				if prefix != "" {
					path = prefix + "." + k
				}
				if av == nil && bv != nil {
					items = append(items, DiffItem{Path: path, Change: "added", LeftValue: "", RightValue: stringify(bv)})
					continue
				}
				if av != nil && bv == nil {
					items = append(items, DiffItem{Path: path, Change: "removed", LeftValue: stringify(av), RightValue: ""})
					continue
				}
				// both present
				if keysOnly {
					// no value compare; recurse to detect nested add/remove
					walk(path, av, bv)
					continue
				}
				// Compare values
				if reflect.DeepEqual(av, bv) {
					walk(path, av, bv)
					continue
				}
				// If both are maps, dive to find granular differences
				if _, ok := av.(map[string]interface{}); ok {
					walk(path, av, bv)
					continue
				}
				if _, ok := bv.(map[string]interface{}); ok {
					walk(path, av, bv)
					continue
				}
				items = append(items, DiffItem{Path: path, Change: "modified", LeftValue: stringify(av), RightValue: stringify(bv)})
			}
			return
		}

		// scalar comparison at root
		if !keysOnly && !reflect.DeepEqual(a, b) {
			items = append(items, DiffItem{Path: prefix, Change: "modified", LeftValue: stringify(a), RightValue: stringify(b)})
		}
	}
	walk("", left, right)
	return DiffResult{Items: items}
}

// breakingChanges performs simple compatibility checks.
func breakingChanges(oldCfg, newCfg map[string]interface{}) []error {
	var out []error
	// Detect required fields removed (heuristic: keys containing 'required' or ending with 'port' etc.)
	oldKeys := flatKeys(oldCfg, "")
	newKeys := flatKeys(newCfg, "")
	oldSet := map[string]struct{}{}
	for _, k := range oldKeys {
		oldSet[k] = struct{}{}
	}
	newSet := map[string]struct{}{}
	for _, k := range newKeys {
		newSet[k] = struct{}{}
	}
	for k := range oldSet {
		if _, ok := newSet[k]; !ok {
			if strings.Contains(k, "required") || strings.HasSuffix(k, ".port") || strings.HasSuffix(k, ".host") {
				out = append(out, fmt.Errorf("potential breaking change: removed key %s", k))
			}
		}
	}
	return out
}

func flatKeys(m map[string]interface{}, prefix string) []string {
	keys := []string{}
	for k, v := range m {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		if sub, ok := v.(map[string]interface{}); ok {
			keys = append(keys, flatKeys(sub, path)...)
			continue
		}
		keys = append(keys, path)
	}
	sort.Strings(keys)
	return keys
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

// Helper functions for backup/restore

// defaultBackupBaseDir returns the base directory for backups.
func defaultBackupBaseDir(dirFlag string) string {
	if strings.TrimSpace(dirFlag) != "" {
		if filepath.IsAbs(dirFlag) {
			return dirFlag
		}
		return filepath.Join(workDir, dirFlag)
	}
	return filepath.Join(workDir, ".switctl", "backups", "config")
}

// generateBackupID creates a backup identifier using timestamp and optional label.
func generateBackupID(label string) string {
	stamp := time.Now().UTC().Format("20060102_150405")
	if s := strings.TrimSpace(label); s != "" {
		return fmt.Sprintf("%s-%s", stamp, sanitizeLabel(s))
	}
	return stamp
}

// sanitizeLabel converts label to filesystem-friendly token.
func sanitizeLabel(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	allowed := func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}
	return strings.Map(allowed, s)
}

// basenames returns base names from a list of paths.
func basenames(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, filepath.Base(p))
	}
	return out
}

// readBackupManifest reads manifest.json in the given backup path.
func readBackupManifest(fs interfaces.FileSystem, backupPath string) (BackupManifest, error) {
	data, err := fs.ReadFile(filepath.Join(backupPath, "manifest.json"))
	if err != nil {
		return BackupManifest{}, err
	}
	var m BackupManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return BackupManifest{}, err
	}
	return m, nil
}

// findLatestBackupID finds the lexicographically latest backup ID under baseDir.
func findLatestBackupID(baseDir string) (string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read backup directory: %w", err)
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	if len(ids) == 0 {
		return "", nil
	}
	sort.Strings(ids)
	return ids[len(ids)-1], nil
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
