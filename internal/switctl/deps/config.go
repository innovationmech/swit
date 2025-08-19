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

package deps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/spf13/viper"
)

// ConfigManager implements the ConfigManager interface using Viper.
type ConfigManager struct {
	viper *viper.Viper
	mu    sync.RWMutex
}

// NewConfigManagerWithOptions creates a new config manager with custom options.
func NewConfigManagerWithOptions(options ConfigManagerOptions) (*ConfigManager, error) {
	v := viper.New()

	// Set configuration name and paths
	v.SetConfigName(options.ConfigName)
	v.SetConfigType(options.ConfigType)

	// Add config paths
	for _, path := range options.ConfigPaths {
		v.AddConfigPath(path)
	}

	// Set environment variable prefix
	if options.EnvPrefix != "" {
		v.SetEnvPrefix(options.EnvPrefix)
	}

	// Enable automatic environment variable binding
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	cm := &ConfigManager{
		viper: v,
	}

	// Set default values
	cm.setDefaults()

	return cm, nil
}

// ConfigManagerOptions represents options for creating a config manager.
type ConfigManagerOptions struct {
	ConfigName  string   // Configuration file name (without extension)
	ConfigType  string   // Configuration file type (yaml, json, etc.)
	ConfigPaths []string // Paths to search for config files
	EnvPrefix   string   // Environment variable prefix
}

// DefaultConfigManagerOptions returns default configuration options.
func DefaultConfigManagerOptions() ConfigManagerOptions {
	homeDir, _ := os.UserHomeDir()
	currentDir, _ := os.Getwd()

	return ConfigManagerOptions{
		ConfigName: "switctl",
		ConfigType: "yaml",
		ConfigPaths: []string{
			".",                                // Current directory
			currentDir,                         // Explicit current directory
			filepath.Join(homeDir, ".switctl"), // User config directory
			"/etc/switctl",                     // System config directory
		},
		EnvPrefix: "SWITCTL",
	}
}

// Load loads configuration from files and environment variables.
func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if err := cm.viper.ReadInConfig(); err != nil {
		// Config file not found is not an error - we use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return interfaces.NewConfigError(
				interfaces.ErrCodeConfigParseError,
				"Failed to parse configuration file",
				err,
			)
		}
	}

	return nil
}

// Get retrieves a configuration value by key.
func (cm *ConfigManager) Get(key string) interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.Get(key)
}

// GetString retrieves a string configuration value.
func (cm *ConfigManager) GetString(key string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.GetString(key)
}

// GetInt retrieves an integer configuration value.
func (cm *ConfigManager) GetInt(key string) int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.GetInt(key)
}

// GetBool retrieves a boolean configuration value.
func (cm *ConfigManager) GetBool(key string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.GetBool(key)
}

// Set sets a configuration value.
func (cm *ConfigManager) Set(key string, value interface{}) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.viper.Set(key, value)
}

// Validate validates the current configuration.
func (cm *ConfigManager) Validate() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var errors []interfaces.ValidationError

	// Validate template directory
	templateDir := cm.viper.GetString("template.directory")
	if templateDir != "" {
		if _, err := os.Stat(templateDir); os.IsNotExist(err) {
			errors = append(errors, interfaces.ValidationError{
				Field:   "template.directory",
				Message: "Template directory does not exist",
				Value:   templateDir,
				Rule:    "exists",
			})
		}
	}

	// Validate output directory
	outputDir := cm.viper.GetString("generator.output_directory")
	if outputDir != "" {
		// Check if parent directory exists and is writable
		parentDir := filepath.Dir(outputDir)
		if _, err := os.Stat(parentDir); os.IsNotExist(err) {
			errors = append(errors, interfaces.ValidationError{
				Field:   "generator.output_directory",
				Message: "Parent directory does not exist",
				Value:   outputDir,
				Rule:    "parent_exists",
			})
		}
	}

	// Validate coverage threshold
	coverageThreshold := cm.viper.GetFloat64("quality.coverage_threshold")
	if coverageThreshold < 0 || coverageThreshold > 100 {
		errors = append(errors, interfaces.ValidationError{
			Field:   "quality.coverage_threshold",
			Message: "Coverage threshold must be between 0 and 100",
			Value:   fmt.Sprintf("%.2f", coverageThreshold),
			Rule:    "range",
		})
	}

	if len(errors) > 0 {
		return &interfaces.ValidationResult{
			Valid:  false,
			Errors: errors,
		}
	}

	return nil
}

// Save saves the configuration to a file.
func (cm *ConfigManager) Save(path string) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if err := cm.viper.WriteConfigAs(path); err != nil {
		return interfaces.NewConfigError(
			interfaces.ErrCodeConfigSaveError,
			"Failed to save configuration file",
			err,
		)
	}

	return nil
}

// GetConfigFilePath returns the path of the loaded config file.
func (cm *ConfigManager) GetConfigFilePath() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.ConfigFileUsed()
}

// setDefaults sets default configuration values.
func (cm *ConfigManager) setDefaults() {
	// UI defaults
	cm.viper.SetDefault("ui.color", true)
	cm.viper.SetDefault("ui.progress", true)
	cm.viper.SetDefault("ui.interactive", true)
	cm.viper.SetDefault("ui.theme", "default")

	// Generator defaults
	cm.viper.SetDefault("generator.output_directory", "./generated")
	cm.viper.SetDefault("generator.overwrite", false)
	cm.viper.SetDefault("generator.backup", true)
	cm.viper.SetDefault("generator.format", true)

	// Template defaults
	cm.viper.SetDefault("template.directory", "./templates")
	cm.viper.SetDefault("template.cache", true)
	cm.viper.SetDefault("template.reload", false)

	// Quality check defaults
	cm.viper.SetDefault("quality.coverage_threshold", 80.0)
	cm.viper.SetDefault("quality.lint_enabled", true)
	cm.viper.SetDefault("quality.security_scan", true)
	cm.viper.SetDefault("quality.format_check", true)
	cm.viper.SetDefault("quality.import_check", true)
	cm.viper.SetDefault("quality.copyright_check", true)

	// Plugin defaults
	cm.viper.SetDefault("plugins.enabled", true)
	cm.viper.SetDefault("plugins.directory", "./plugins")
	cm.viper.SetDefault("plugins.auto_load", true)

	// Logging defaults
	cm.viper.SetDefault("logging.level", "info")
	cm.viper.SetDefault("logging.format", "text")
	cm.viper.SetDefault("logging.output", "stdout")

	// Development defaults
	cm.viper.SetDefault("dev.hot_reload", true)
	cm.viper.SetDefault("dev.auto_format", true)
	cm.viper.SetDefault("dev.auto_test", false)

	// Project defaults
	cm.viper.SetDefault("project.author", "")
	cm.viper.SetDefault("project.license", "MIT")
	cm.viper.SetDefault("project.go_version", "1.19")
	cm.viper.SetDefault("project.module_prefix", "")
}

// GetUIConfig returns UI-specific configuration.
func (cm *ConfigManager) GetUIConfig() UIConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return UIConfig{
		Color:       cm.viper.GetBool("ui.color"),
		Progress:    cm.viper.GetBool("ui.progress"),
		Interactive: cm.viper.GetBool("ui.interactive"),
		Theme:       cm.viper.GetString("ui.theme"),
	}
}

// GetGeneratorConfig returns generator-specific configuration.
func (cm *ConfigManager) GetGeneratorConfig() GeneratorConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return GeneratorConfig{
		OutputDirectory: cm.viper.GetString("generator.output_directory"),
		Overwrite:       cm.viper.GetBool("generator.overwrite"),
		Backup:          cm.viper.GetBool("generator.backup"),
		Format:          cm.viper.GetBool("generator.format"),
	}
}

// GetTemplateConfig returns template-specific configuration.
func (cm *ConfigManager) GetTemplateConfig() TemplateConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return TemplateConfig{
		Directory: cm.viper.GetString("template.directory"),
		Cache:     cm.viper.GetBool("template.cache"),
		Reload:    cm.viper.GetBool("template.reload"),
	}
}

// GetQualityConfig returns quality check configuration.
func (cm *ConfigManager) GetQualityConfig() QualityConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return QualityConfig{
		CoverageThreshold: cm.viper.GetFloat64("quality.coverage_threshold"),
		LintEnabled:       cm.viper.GetBool("quality.lint_enabled"),
		SecurityScan:      cm.viper.GetBool("quality.security_scan"),
		FormatCheck:       cm.viper.GetBool("quality.format_check"),
		ImportCheck:       cm.viper.GetBool("quality.import_check"),
		CopyrightCheck:    cm.viper.GetBool("quality.copyright_check"),
	}
}

// UIConfig represents UI configuration.
type UIConfig struct {
	Color       bool   `yaml:"color" json:"color"`
	Progress    bool   `yaml:"progress" json:"progress"`
	Interactive bool   `yaml:"interactive" json:"interactive"`
	Theme       string `yaml:"theme" json:"theme"`
}

// GeneratorConfig represents generator configuration.
type GeneratorConfig struct {
	OutputDirectory string `yaml:"output_directory" json:"output_directory"`
	Overwrite       bool   `yaml:"overwrite" json:"overwrite"`
	Backup          bool   `yaml:"backup" json:"backup"`
	Format          bool   `yaml:"format" json:"format"`
}

// TemplateConfig represents template configuration.
type TemplateConfig struct {
	Directory string `yaml:"directory" json:"directory"`
	Cache     bool   `yaml:"cache" json:"cache"`
	Reload    bool   `yaml:"reload" json:"reload"`
}

// QualityConfig represents quality check configuration.
type QualityConfig struct {
	CoverageThreshold float64 `yaml:"coverage_threshold" json:"coverage_threshold"`
	LintEnabled       bool    `yaml:"lint_enabled" json:"lint_enabled"`
	SecurityScan      bool    `yaml:"security_scan" json:"security_scan"`
	FormatCheck       bool    `yaml:"format_check" json:"format_check"`
	ImportCheck       bool    `yaml:"import_check" json:"import_check"`
	CopyrightCheck    bool    `yaml:"copyright_check" json:"copyright_check"`
}

// ConfigTemplate represents a configuration template.
type ConfigTemplate struct {
	Name        string                 `yaml:"name" json:"name"`
	Description string                 `yaml:"description" json:"description"`
	Config      map[string]interface{} `yaml:"config" json:"config"`
	Tags        []string               `yaml:"tags" json:"tags"`
}

// GenerateConfigTemplate generates a configuration template file.
func (cm *ConfigManager) GenerateConfigTemplate() (string, error) {
	template := `# switctl configuration file
# This file contains configuration for the Swit framework scaffolding tool

# UI configuration
ui:
  color: true              # Enable colored output
  progress: true           # Show progress bars
  interactive: true        # Enable interactive prompts
  theme: default          # UI theme (default, dark, light)

# Generator configuration
generator:
  output_directory: "./generated"  # Default output directory
  overwrite: false                 # Overwrite existing files
  backup: true                     # Create backups before overwriting
  format: true                     # Format generated code

# Template configuration
template:
  directory: "./templates"  # Template directory
  cache: true              # Cache templates
  reload: false            # Reload templates on change

# Quality check configuration
quality:
  coverage_threshold: 80.0  # Minimum test coverage percentage
  lint_enabled: true       # Enable linting
  security_scan: true      # Enable security scanning
  format_check: true       # Check code formatting
  import_check: true       # Check import organization
  copyright_check: true    # Check copyright headers

# Plugin configuration
plugins:
  enabled: true           # Enable plugin system
  directory: "./plugins"  # Plugin directory
  auto_load: true        # Auto-load plugins

# Logging configuration
logging:
  level: info            # Log level (debug, info, warn, error)
  format: text           # Log format (text, json)
  output: stdout         # Log output (stdout, stderr, file)

# Development configuration
dev:
  hot_reload: true       # Enable hot reload
  auto_format: true      # Auto-format code
  auto_test: false       # Auto-run tests

# Project defaults
project:
  author: ""             # Default author name
  license: MIT           # Default license
  go_version: "1.19"     # Default Go version
  module_prefix: ""      # Default module prefix
`

	return template, nil
}

// LoadFromFile loads configuration from a specific file.
func (cm *ConfigManager) LoadFromFile(path string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.viper.SetConfigFile(path)
	if err := cm.viper.ReadInConfig(); err != nil {
		return interfaces.NewConfigError(
			interfaces.ErrCodeConfigParseError,
			"Failed to load configuration file",
			err,
		)
	}

	return nil
}

// Merge merges configuration from another config manager.
func (cm *ConfigManager) Merge(other *ConfigManager) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	other.mu.RLock()
	defer other.mu.RUnlock()

	// Get all settings from the other config manager
	otherSettings := other.viper.AllSettings()

	// Recursively set all values from other to override current values
	cm.mergeMap("", otherSettings)

	return nil
}

// mergeMap recursively merges a map of settings into the current config
func (cm *ConfigManager) mergeMap(prefix string, data map[string]interface{}) {
	for key, value := range data {
		var fullKey string
		if prefix == "" {
			fullKey = key
		} else {
			fullKey = prefix + "." + key
		}

		if valueMap, ok := value.(map[string]interface{}); ok {
			// Recursively merge nested maps
			cm.mergeMap(fullKey, valueMap)
		} else {
			// Set the value directly, which will override any existing value
			cm.viper.Set(fullKey, value)
		}
	}
}

// AllSettings returns all configuration settings.
func (cm *ConfigManager) AllSettings() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.viper.AllSettings()
}
