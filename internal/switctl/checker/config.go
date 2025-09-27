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

package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// ConfigChecker implements configuration file validation.
type ConfigChecker struct {
	workDir string
	logger  interfaces.Logger
	config  ConfigValidationConfig
}

// ConfigValidationConfig represents configuration for config validation.
type ConfigValidationConfig struct {
	YAMLValidation  bool     `yaml:"yaml_validation" json:"yaml_validation"`
	JSONValidation  bool     `yaml:"json_validation" json:"json_validation"`
	ProtoValidation bool     `yaml:"proto_validation" json:"proto_validation"`
	TOMLValidation  bool     `yaml:"toml_validation" json:"toml_validation"`
	ConfigPaths     []string `yaml:"config_paths" json:"config_paths"`
	ExcludePatterns []string `yaml:"exclude_patterns" json:"exclude_patterns"`
	SchemaPath      string   `yaml:"schema_path" json:"schema_path"`
	StrictMode      bool     `yaml:"strict_mode" json:"strict_mode"`
	CheckKeys       bool     `yaml:"check_keys" json:"check_keys"`
	CheckValues     bool     `yaml:"check_values" json:"check_values"`
}

// NewConfigChecker creates a new configuration checker.
func NewConfigChecker(workDir string, logger interfaces.Logger) *ConfigChecker {
	return &ConfigChecker{
		workDir: workDir,
		logger:  logger,
		config:  defaultConfigValidationConfig(),
	}
}

// SetConfig updates the config checker configuration.
func (cc *ConfigChecker) SetConfig(config ConfigValidationConfig) {
	cc.config = config
}

// Check implements IndividualChecker interface.
func (cc *ConfigChecker) Check(ctx context.Context) (interfaces.CheckResult, error) {
	startTime := time.Now()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return interfaces.CheckResult{}, ctx.Err()
	default:
	}

	// Run the config validation
	validationResult := cc.ValidateConfig()

	// Convert ValidationResult to CheckResult
	status := interfaces.CheckStatusPass
	message := "Configuration validation passed"

	if !validationResult.Valid {
		status = interfaces.CheckStatusFail
		message = fmt.Sprintf("Configuration validation failed with %d errors", len(validationResult.Errors))
	} else if len(validationResult.Warnings) > 0 {
		status = interfaces.CheckStatusWarning
		message = fmt.Sprintf("Configuration validation passed with %d warnings", len(validationResult.Warnings))
	}

	// Convert validation errors to check details
	var details []interfaces.CheckDetail
	for _, err := range validationResult.Errors {
		rule := err.Rule
		if rule == "" {
			rule = err.Code // Use Code as Rule if Rule is empty
		}
		details = append(details, interfaces.CheckDetail{
			File:     err.Field, // Use Field as File
			Message:  err.Message,
			Rule:     rule,
			Severity: "error",
			Context:  err,
		})
	}
	for _, warn := range validationResult.Warnings {
		rule := warn.Rule
		if rule == "" {
			rule = warn.Code // Use Code as Rule if Rule is empty
		}
		details = append(details, interfaces.CheckDetail{
			File:     warn.Field, // Use Field as File
			Message:  warn.Message,
			Rule:     rule,
			Severity: "warning",
			Context:  warn,
		})
	}

	// Calculate score based on validation results
	score := 100
	if !validationResult.Valid {
		score = 0 // No score if there are errors
	} else if len(validationResult.Warnings) > 0 {
		// Reduce score based on warnings
		score = 100 - (len(validationResult.Warnings) * 10)
		if score < 0 {
			score = 0
		}
	}

	return interfaces.CheckResult{
		Name:     "config",
		Status:   status,
		Message:  message,
		Details:  details,
		Duration: time.Since(startTime),
		Score:    score,
		MaxScore: 100,
		Fixable:  false, // Config issues are generally not automatically fixable
	}, nil
}

// Name returns the name of this checker.
func (cc *ConfigChecker) Name() string {
	return "config"
}

// Description returns the description of this checker.
func (cc *ConfigChecker) Description() string {
	return "Validates configuration files (YAML, JSON, TOML, Proto)"
}

// Close cleans up any resources used by the checker.
func (cc *ConfigChecker) Close() error {
	// No resources to clean up for ConfigChecker
	return nil
}

// ValidateConfig performs comprehensive configuration validation.
func (cc *ConfigChecker) ValidateConfig() interfaces.ValidationResult {
	start := time.Now()
	result := interfaces.ValidationResult{
		Valid:    true,
		Errors:   make([]interfaces.ValidationError, 0),
		Warnings: make([]interfaces.ValidationError, 0),
	}

	// Find configuration files
	configFiles, err := cc.findConfigFiles()
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, interfaces.ValidationError{
			Message: fmt.Sprintf("Failed to find configuration files: %v", err),
			Code:    "CONFIG_DISCOVERY_FAILED",
		})
		result.Duration = time.Since(start)
		return result
	}

	if len(configFiles) == 0 {
		result.Duration = time.Since(start)
		return result
	}

    if cc.logger != nil {
        cc.logger.Info("Found configuration files", "count", len(configFiles))
    }

	// Validate each configuration file
	for _, file := range configFiles {
		fileErrors, fileWarnings := cc.validateConfigFile(file)
		result.Errors = append(result.Errors, fileErrors...)
		result.Warnings = append(result.Warnings, fileWarnings...)
	}

	// Validate Protocol Buffer files if enabled
	if cc.config.ProtoValidation {
		protoErrors, protoWarnings := cc.validateProtoFiles()
		result.Errors = append(result.Errors, protoErrors...)
		result.Warnings = append(result.Warnings, protoWarnings...)
	}

	// Validate Go module configuration (only if go.mod exists)
	goModPath := filepath.Join(cc.workDir, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		modErrors, modWarnings := cc.validateGoModule()
		result.Errors = append(result.Errors, modErrors...)
		result.Warnings = append(result.Warnings, modWarnings...)
	}

	// Check for common configuration issues
	commonErrors, commonWarnings := cc.checkCommonIssues()
	result.Errors = append(result.Errors, commonErrors...)
	result.Warnings = append(result.Warnings, commonWarnings...)

	// Determine overall validity
	result.Valid = len(result.Errors) == 0
	result.Duration = time.Since(start)

	return result
}

// validateConfigFile validates a single configuration file.
func (cc *ConfigChecker) validateConfigFile(filename string) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// Determine file type
	ext := strings.ToLower(filepath.Ext(filename))
	baseName := filepath.Base(filename)

	switch {
	case baseName == "go.mod":
		fileErrors, fileWarnings := cc.validateGoModFile(filename)
		errors = append(errors, fileErrors...)
		warnings = append(warnings, fileWarnings...)
	case baseName == ".env":
		fileErrors, fileWarnings := cc.validateEnvFile(filename)
		errors = append(errors, fileErrors...)
		warnings = append(warnings, fileWarnings...)
	case baseName == ".gitignore":
		fileErrors, fileWarnings := cc.validateGitignoreFile(filename)
		errors = append(errors, fileErrors...)
		warnings = append(warnings, fileWarnings...)
	case ext == ".yaml" || ext == ".yml":
		if cc.config.YAMLValidation {
			fileErrors, fileWarnings := cc.validateYAMLFile(filename)
			errors = append(errors, fileErrors...)
			warnings = append(warnings, fileWarnings...)
		}
	case ext == ".json":
		if cc.config.JSONValidation {
			fileErrors, fileWarnings := cc.validateJSONFile(filename)
			errors = append(errors, fileErrors...)
			warnings = append(warnings, fileWarnings...)
		}
	case ext == ".toml":
		if cc.config.TOMLValidation {
			fileErrors, fileWarnings := cc.validateTOMLFile(filename)
			errors = append(errors, fileErrors...)
			warnings = append(warnings, fileWarnings...)
		}
	default:
		warnings = append(warnings, interfaces.ValidationError{
			Field:   filename,
			Message: fmt.Sprintf("Unsupported configuration file type: %s", ext),
			Code:    "UNSUPPORTED_FILE_TYPE",
		})
	}

	return errors, warnings
}

// validateYAMLFile validates a YAML configuration file.
func (cc *ConfigChecker) validateYAMLFile(filename string) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// Read file content
	fullPath := filepath.Join(cc.workDir, filename)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		errors = append(errors, interfaces.ValidationError{
			Field:   filename,
			Message: fmt.Sprintf("Failed to read YAML file: %v", err),
			Code:    "FILE_READ_ERROR",
		})
		return errors, warnings
	}

	// Check for empty file
	if len(content) == 0 {
		warnings = append(warnings, interfaces.ValidationError{
			Field:   filename,
			Message: "Configuration file is empty",
			Code:    "EMPTY_CONFIG",
		})
		return errors, warnings
	}

	// Parse YAML
	var yamlData any
	err = yaml.Unmarshal(content, &yamlData)
	if err != nil {
		errors = append(errors, interfaces.ValidationError{
			Field:   filename,
			Message: fmt.Sprintf("Invalid YAML syntax: %v", err),
			Code:    "YAML_SYNTAX_ERROR",
		})
		return errors, warnings
	}

	// Check for common YAML issues
	yamlErrors, yamlWarnings := cc.checkYAMLStructure(filename, yamlData)
	errors = append(errors, yamlErrors...)
	warnings = append(warnings, yamlWarnings...)

	// Validate specific configuration schemas
	schemaErrors, schemaWarnings := cc.validateConfigSchema(filename, yamlData)
	errors = append(errors, schemaErrors...)
	warnings = append(warnings, schemaWarnings...)

	return errors, warnings
}

// validateJSONFile validates a JSON configuration file.
func (cc *ConfigChecker) validateJSONFile(filename string) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// Read file content
	fullPath := filepath.Join(cc.workDir, filename)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		errors = append(errors, interfaces.ValidationError{
			Field:   filename,
			Message: fmt.Sprintf("Failed to read JSON file: %v", err),
			Code:    "FILE_READ_ERROR",
		})
		return errors, warnings
	}

	// Parse JSON
	var jsonData any
	err = json.Unmarshal(content, &jsonData)
	if err != nil {
		errors = append(errors, interfaces.ValidationError{
			Field:   filename,
			Message: fmt.Sprintf("Invalid JSON syntax: %v", err),
			Code:    "JSON_SYNTAX_ERROR",
		})
		return errors, warnings
	}

	// Check for common JSON issues
	jsonErrors, jsonWarnings := cc.checkJSONStructure(filename, jsonData)
	errors = append(errors, jsonErrors...)
	warnings = append(warnings, jsonWarnings...)

	return errors, warnings
}

// validateTOMLFile validates a TOML configuration file.
func (cc *ConfigChecker) validateTOMLFile(filename string) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// For now, we'll just check if the file is readable
	// Full TOML parsing would require importing a TOML library
	fullPath := filepath.Join(cc.workDir, filename)
	_, err := os.ReadFile(fullPath)
	if err != nil {
		errors = append(errors, interfaces.ValidationError{
			Field:   filename,
			Message: fmt.Sprintf("Failed to read TOML file: %v", err),
			Code:    "FILE_READ_ERROR",
		})
	} else {
		warnings = append(warnings, interfaces.ValidationError{
			Field:   filename,
			Message: "TOML validation not fully implemented - syntax check only",
			Code:    "PARTIAL_VALIDATION",
		})
	}

	return errors, warnings
}

// checkYAMLStructure checks YAML structure for common issues.
func (cc *ConfigChecker) checkYAMLStructure(filename string, data any) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// Check if root is a map (most config files should be)
	if _, isMap := data.(map[string]any); !isMap {
		warnings = append(warnings, interfaces.ValidationError{
			Field:   filename,
			Message: "Root element is not a map/object - this may be intentional",
			Code:    "NON_MAP_ROOT",
		})
		return errors, warnings
	}

	rootMap := data.(map[string]any)

	// Check for empty configuration
	if len(rootMap) == 0 {
		warnings = append(warnings, interfaces.ValidationError{
			Field:   filename,
			Message: "Configuration file is empty",
			Code:    "EMPTY_CONFIG",
		})
	}

	// Check for common key naming issues
	for key := range rootMap {
		if cc.config.CheckKeys {
			keyErrors, keyWarnings := cc.validateConfigKey(filename, key)
			errors = append(errors, keyErrors...)
			warnings = append(warnings, keyWarnings...)
		}
	}

	return errors, warnings
}

// checkJSONStructure checks JSON structure for common issues.
func (cc *ConfigChecker) checkJSONStructure(filename string, data any) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// Similar checks as YAML
	if _, isMap := data.(map[string]any); !isMap {
		warnings = append(warnings, interfaces.ValidationError{
			Field:   filename,
			Message: "Root element is not an object - this may be intentional",
			Code:    "NON_OBJECT_ROOT",
		})
		return errors, warnings
	}

	rootMap := data.(map[string]any)
	if len(rootMap) == 0 {
		warnings = append(warnings, interfaces.ValidationError{
			Field:   filename,
			Message: "Configuration file is empty",
			Code:    "EMPTY_CONFIG",
		})
	}

	return errors, warnings
}

// validateConfigKey validates configuration key naming.
func (cc *ConfigChecker) validateConfigKey(filename, key string) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// Check for camelCase vs snake_case consistency
	hasCamelCase := regexp.MustCompile(`[a-z][A-Z]`).MatchString(key)
	hasSnakeCase := strings.Contains(key, "_")

	if hasCamelCase && hasSnakeCase {
		warnings = append(warnings, interfaces.ValidationError{
			Field:   fmt.Sprintf("%s:%s", filename, key),
			Message: "Mixed camelCase and snake_case in key name",
			Code:    "INCONSISTENT_KEY_NAMING",
		})
	}

	// Check for reserved keywords or problematic names
	problematicKeys := []string{"type", "class", "function", "import", "export"}
	for _, problematic := range problematicKeys {
		if key == problematic {
			warnings = append(warnings, interfaces.ValidationError{
				Field:   fmt.Sprintf("%s:%s", filename, key),
				Message: fmt.Sprintf("Key '%s' might conflict with reserved words", key),
				Code:    "RESERVED_KEY_NAME",
			})
		}
	}

	return errors, warnings
}

// validateConfigSchema validates configuration against known schemas.
func (cc *ConfigChecker) validateConfigSchema(filename string, data any) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// Determine config type from filename
	baseName := filepath.Base(filename)

	switch {
	case strings.Contains(baseName, "swit") || strings.Contains(baseName, "server"):
		schemaErrors, schemaWarnings := cc.validateSwitConfig(filename, data)
		errors = append(errors, schemaErrors...)
		warnings = append(warnings, schemaWarnings...)
	case strings.Contains(baseName, "docker"):
		// Docker compose validation could be added here
	case strings.Contains(baseName, "k8s") || strings.Contains(baseName, "kubernetes"):
		// Kubernetes manifest validation could be added here
	}

	return errors, warnings
}

// validateSwitConfig validates Swit framework configuration files.
func (cc *ConfigChecker) validateSwitConfig(filename string, data any) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	rootMap, ok := data.(map[string]any)
	if !ok {
		return errors, warnings
	}

	// Check for required Swit configuration fields
	requiredFields := []string{"server", "database"}
	for _, field := range requiredFields {
		if _, exists := rootMap[field]; !exists {
			errors = append(errors, interfaces.ValidationError{
				Field:   fmt.Sprintf("%s:%s", filename, field),
				Message: fmt.Sprintf("Required field '%s' is missing", field),
				Code:    "MISSING_REQUIRED_FIELD",
			})
		}
	}

	// Validate server configuration
	if serverConfig, exists := rootMap["server"]; exists {
		serverErrors, serverWarnings := cc.validateServerConfig(filename, serverConfig)
		errors = append(errors, serverErrors...)
		warnings = append(warnings, serverWarnings...)
	}

	// Validate database configuration
	if dbConfig, exists := rootMap["database"]; exists {
		dbErrors, dbWarnings := cc.validateDatabaseConfig(filename, dbConfig)
		errors = append(errors, dbErrors...)
		warnings = append(warnings, dbWarnings...)
	}

	return errors, warnings
}

// validateServerConfig validates server configuration section.
func (cc *ConfigChecker) validateServerConfig(filename string, serverConfig any) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	serverMap, ok := serverConfig.(map[string]any)
	if !ok {
		errors = append(errors, interfaces.ValidationError{
			Field:   fmt.Sprintf("%s:server", filename),
			Message: "Server configuration must be an object",
			Code:    "INVALID_SERVER_CONFIG_TYPE",
		})
		return errors, warnings
	}

	// Check for port configurations
	if httpPort, exists := serverMap["http_port"]; exists {
		if port, ok := httpPort.(int); ok {
			if port < 1024 || port > 65535 {
				errors = append(errors, interfaces.ValidationError{
					Field:   fmt.Sprintf("%s:server.http_port", filename),
					Message: "HTTP port must be between 1024 and 65535",
					Value:   port,
					Code:    "INVALID_PORT_RANGE",
				})
			}
		} else {
			errors = append(errors, interfaces.ValidationError{
				Field:   fmt.Sprintf("%s:server.http_port", filename),
				Message: "HTTP port must be an integer",
				Value:   httpPort,
				Code:    "INVALID_PORT_TYPE",
			})
		}
	}

	return errors, warnings
}

// validateDatabaseConfig validates database configuration section.
func (cc *ConfigChecker) validateDatabaseConfig(filename string, dbConfig any) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	dbMap, ok := dbConfig.(map[string]any)
	if !ok {
		errors = append(errors, interfaces.ValidationError{
			Field:   fmt.Sprintf("%s:database", filename),
			Message: "Database configuration must be an object",
			Code:    "INVALID_DATABASE_CONFIG_TYPE",
		})
		return errors, warnings
	}

	// Check database type
	if dbType, exists := dbMap["type"]; exists {
		if typeStr, ok := dbType.(string); ok {
			supportedTypes := []string{"mysql", "postgresql", "mongodb", "sqlite"}
			if !slices.Contains(supportedTypes, typeStr) {
				warnings = append(warnings, interfaces.ValidationError{
					Field:   fmt.Sprintf("%s:database.type", filename),
					Message: fmt.Sprintf("Database type '%s' may not be fully supported", typeStr),
					Value:   typeStr,
					Code:    "UNSUPPORTED_DATABASE_TYPE",
				})
			}
		}
	}

	return errors, warnings
}

// validateProtoFiles validates Protocol Buffer files.
func (cc *ConfigChecker) validateProtoFiles() ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// Find proto files
	protoFiles, err := cc.findProtoFiles()
	if err != nil {
		errors = append(errors, interfaces.ValidationError{
			Message: fmt.Sprintf("Failed to find proto files: %v", err),
			Code:    "PROTO_DISCOVERY_FAILED",
		})
		return errors, warnings
	}

	if len(protoFiles) == 0 {
		return errors, warnings
	}

	// Use buf to validate proto files if available
	if cc.isCommandAvailable("buf") {
		protoErrors, protoWarnings := cc.validateProtoWithBuf()
		errors = append(errors, protoErrors...)
		warnings = append(warnings, protoWarnings...)
	} else {
		// Basic proto file validation
		for _, file := range protoFiles {
			fileErrors, fileWarnings := cc.validateProtoFileSyntax(file)
			errors = append(errors, fileErrors...)
			warnings = append(warnings, fileWarnings...)
		}
	}

	return errors, warnings
}

// validateProtoWithBuf uses buf tool to validate proto files.
func (cc *ConfigChecker) validateProtoWithBuf() ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// Create context for timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "buf", "lint")
	cmd.Dir = cc.workDir

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// buf lint returns non-zero exit code when issues are found
			lines := strings.Split(string(exitErr.Stderr), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					errors = append(errors, interfaces.ValidationError{
						Message: line,
						Code:    "BUF_LINT_ERROR",
					})
				}
			}
		} else {
			errors = append(errors, interfaces.ValidationError{
				Message: fmt.Sprintf("Failed to run buf lint: %v", err),
				Code:    "BUF_EXECUTION_FAILED",
			})
		}
	}

	// Parse successful output for warnings
	if len(output) > 0 {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				warnings = append(warnings, interfaces.ValidationError{
					Message: line,
					Code:    "BUF_LINT_WARNING",
				})
			}
		}
	}

	return errors, warnings
}

// validateProtoFileSyntax performs basic proto file syntax validation.
func (cc *ConfigChecker) validateProtoFileSyntax(filename string) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	content, err := os.ReadFile(filename)
	if err != nil {
		errors = append(errors, interfaces.ValidationError{
			Field:   filename,
			Message: fmt.Sprintf("Failed to read proto file: %v", err),
			Code:    "PROTO_READ_ERROR",
		})
		return errors, warnings
	}

	lines := strings.Split(string(content), "\n")

	// Basic syntax checks
	hasPackage := false

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		lineNum++ // Line numbers start at 1

		if strings.HasPrefix(line, "package ") {
			hasPackage = true
		}

		// Check for common issues
		if strings.Contains(line, "import") && !strings.Contains(line, `"google/`) {
			// Custom import - might want to validate path
		}
	}

	if !hasPackage {
		warnings = append(warnings, interfaces.ValidationError{
			Field:   filename,
			Message: "Proto file missing package declaration",
			Code:    "MISSING_PACKAGE",
		})
	}

	return errors, warnings
}

// validateGoModule validates Go module configuration.
func (cc *ConfigChecker) validateGoModule() ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// Read and validate go.mod (assumes file exists since this method is only called when it does)
	goModPath := filepath.Join(cc.workDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		errors = append(errors, interfaces.ValidationError{
			Field:   "go.mod",
			Message: fmt.Sprintf("Failed to read go.mod: %v", err),
			Code:    "GO_MOD_READ_ERROR",
		})
		return errors, warnings
	}

	// Basic go.mod validation
	goModContent := string(content)
	lines := strings.Split(goModContent, "\n")

	hasModule := false
	hasGoVersion := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "module ") {
			hasModule = true
		}

		if strings.HasPrefix(line, "go ") {
			hasGoVersion = true
		}
	}

	if !hasModule {
		errors = append(errors, interfaces.ValidationError{
			Field:   "go.mod",
			Message: "go.mod missing module declaration",
			Code:    "MISSING_MODULE_DECLARATION",
		})
	}

	if !hasGoVersion {
		warnings = append(warnings, interfaces.ValidationError{
			Field:   "go.mod",
			Message: "go.mod missing Go version declaration",
			Code:    "MISSING_GO_VERSION",
		})
	}

	return errors, warnings
}

// checkCommonIssues checks for common configuration problems.
func (cc *ConfigChecker) checkCommonIssues() ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// Check for .env files with potentially sensitive data
	envFiles := []string{".env", ".env.local", ".env.production"}
	for _, envFile := range envFiles {
		envPath := filepath.Join(cc.workDir, envFile)
		if _, err := os.Stat(envPath); err == nil {
			warnings = append(warnings, interfaces.ValidationError{
				Field:   envFile,
				Message: "Environment file found - ensure it's not committed to version control",
				Code:    "ENV_FILE_FOUND",
			})
		}
	}

	// Check for configuration files in version control
	gitignorePath := filepath.Join(cc.workDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		gitignoreContent, err := os.ReadFile(gitignorePath)
		if err == nil {
			gitignoreStr := string(gitignoreContent)
			if !strings.Contains(gitignoreStr, "*.env") {
				warnings = append(warnings, interfaces.ValidationError{
					Field:   ".gitignore",
					Message: "Consider adding *.env to .gitignore",
					Code:    "ENV_NOT_IGNORED",
				})
			}
		}
	}

	return errors, warnings
}

// findConfigFiles finds all configuration files in the project.
func (cc *ConfigChecker) findConfigFiles() ([]string, error) {
	var files []string

	// Define paths to search
	searchPaths := cc.config.ConfigPaths
	if len(searchPaths) == 0 {
		searchPaths = []string{cc.workDir}
	}

	configExtensions := []string{".yaml", ".yml", ".json", ".toml"}
	specialConfigFiles := []string{"go.mod", "go.sum", ".env", ".gitignore"}

	for _, searchPath := range searchPaths {
		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				// Skip certain directories
				if info.Name() == "vendor" || info.Name() == ".git" ||
					info.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}

			fileName := info.Name()

			// Check for special config files (go.mod, .env, etc.)
			if slices.Contains(specialConfigFiles, fileName) {
				if !cc.isExcludedFile(path) {
					relPath, err := filepath.Rel(cc.workDir, path)
					if err == nil {
						files = append(files, relPath)
					}
				}
				return nil
			}

			// Check if file is a configuration file by extension
			ext := strings.ToLower(filepath.Ext(path))
			if slices.Contains(configExtensions, ext) {
				if !cc.isExcludedFile(path) {
					relPath, err := filepath.Rel(cc.workDir, path)
					if err == nil {
						files = append(files, relPath)
					}
				}
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

// findProtoFiles finds all Protocol Buffer files.
func (cc *ConfigChecker) findProtoFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(cc.workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip certain directories
			if info.Name() == "vendor" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(path, ".proto") {
			relPath, err := filepath.Rel(cc.workDir, path)
			if err == nil {
				files = append(files, relPath)
			}
		}

		return nil
	})

	return files, err
}

// isExcludedFile checks if a file should be excluded from validation.
func (cc *ConfigChecker) isExcludedFile(path string) bool {
	for _, pattern := range cc.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		if matched, _ := regexp.MatchString(pattern, path); matched {
			return true
		}
	}
	return false
}

// isCommandAvailable checks if a command is available in PATH.
func (cc *ConfigChecker) isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// Placeholder implementations for QualityChecker interface compatibility
func (cc *ConfigChecker) CheckCodeStyle() interfaces.CheckResult {
	return interfaces.CheckResult{Name: "code_style", Status: interfaces.CheckStatusSkip}
}

func (cc *ConfigChecker) RunTests() interfaces.TestResult {
	return interfaces.TestResult{}
}

func (cc *ConfigChecker) CheckSecurity() interfaces.SecurityResult {
	return interfaces.SecurityResult{}
}

func (cc *ConfigChecker) CheckCoverage() interfaces.CoverageResult {
	return interfaces.CoverageResult{}
}

func (cc *ConfigChecker) CheckInterfaces() interfaces.CheckResult {
	return interfaces.CheckResult{Name: "interfaces", Status: interfaces.CheckStatusSkip}
}

func (cc *ConfigChecker) CheckImports() interfaces.CheckResult {
	return interfaces.CheckResult{Name: "imports", Status: interfaces.CheckStatusSkip}
}

func (cc *ConfigChecker) CheckCopyright() interfaces.CheckResult {
	return interfaces.CheckResult{Name: "copyright", Status: interfaces.CheckStatusSkip}
}

// defaultConfigValidationConfig returns default configuration validation settings.
func defaultConfigValidationConfig() ConfigValidationConfig {
	return ConfigValidationConfig{
		YAMLValidation:  true,
		JSONValidation:  true,
		ProtoValidation: true,
		TOMLValidation:  false, // Requires additional dependencies
		StrictMode:      false,
		CheckKeys:       true,
		CheckValues:     true,
		ExcludePatterns: []string{
			"vendor/*",
			"node_modules/*",
			".git/*",
			"*.lock",
		},
	}
}

// validateGoModFile validates a Go module file.
func (cc *ConfigChecker) validateGoModFile(filename string) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// Read file content
	fullPath := filepath.Join(cc.workDir, filename)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		errors = append(errors, interfaces.ValidationError{
			Field:   filename,
			Message: fmt.Sprintf("Failed to read go.mod file: %v", err),
			Code:    "FILE_READ_ERROR",
		})
		return errors, warnings
	}

	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// Check for module declaration
	hasModule := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if strings.HasPrefix(line, "module ") {
			hasModule = true
			// Validate module name
			moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module"))
			if moduleName == "" {
				errors = append(errors, interfaces.ValidationError{
					Field:   filename,
					Message: fmt.Sprintf("Module name cannot be empty (line %d)", i+1),
					Code:    "EMPTY_MODULE_NAME",
				})
			}
			break
		}
	}

	if !hasModule {
		errors = append(errors, interfaces.ValidationError{
			Field:   filename,
			Message: "missing module declaration in go.mod",
			Code:    "MISSING_MODULE_DECLARATION",
		})
	}

	// Check Go version if present
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			version := strings.TrimSpace(strings.TrimPrefix(line, "go"))
			if !regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`).MatchString(version) {
				warnings = append(warnings, interfaces.ValidationError{
					Field:   filename,
					Message: fmt.Sprintf("Invalid Go version format: %s (line %d)", version, i+1),
					Code:    "INVALID_GO_VERSION",
				})
			}
			break
		}
	}

	return errors, warnings
}

// validateEnvFile validates an environment file.
func (cc *ConfigChecker) validateEnvFile(filename string) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// Read file content
	fullPath := filepath.Join(cc.workDir, filename)
	_, err := os.ReadFile(fullPath)
	if err != nil {
		errors = append(errors, interfaces.ValidationError{
			Field:   filename,
			Message: fmt.Sprintf("Failed to read .env file: %v", err),
			Code:    "FILE_READ_ERROR",
		})
		return errors, warnings
	}

	// Warn about environment files
	warnings = append(warnings, interfaces.ValidationError{
		Field:   filename,
		Message: "Environment file found - ensure it's not committed to version control",
		Code:    "ENV_FILE_FOUND",
	})

	// Check if .env is in .gitignore
	gitignorePath := filepath.Join(cc.workDir, ".gitignore")
	if gitignoreContent, err := os.ReadFile(gitignorePath); err == nil {
		gitignoreStr := string(gitignoreContent)
		if !strings.Contains(gitignoreStr, ".env") {
			warnings = append(warnings, interfaces.ValidationError{
				Field:   filename,
				Message: "Add *.env to .gitignore to prevent committing sensitive data",
				Code:    "ENV_NOT_IGNORED",
			})
		}
	}

	return errors, warnings
}

// validateGitignoreFile validates a .gitignore file.
func (cc *ConfigChecker) validateGitignoreFile(filename string) ([]interfaces.ValidationError, []interfaces.ValidationError) {
	var errors []interfaces.ValidationError
	var warnings []interfaces.ValidationError

	// Read file content
	fullPath := filepath.Join(cc.workDir, filename)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		errors = append(errors, interfaces.ValidationError{
			Field:   filename,
			Message: fmt.Sprintf("Failed to read .gitignore file: %v", err),
			Code:    "FILE_READ_ERROR",
		})
		return errors, warnings
	}

	contentStr := string(content)

	// Check for common patterns that should be ignored
	commonPatterns := []string{"*.log", "tmp/", "build/", "dist/", ".env"}
	missing := []string{}

	for _, pattern := range commonPatterns {
		if !strings.Contains(contentStr, pattern) {
			missing = append(missing, pattern)
		}
	}

	if len(missing) > 0 {
		warnings = append(warnings, interfaces.ValidationError{
			Field:   filename,
			Message: fmt.Sprintf("Consider adding common patterns to .gitignore: %s", strings.Join(missing, ", ")),
			Code:    "MISSING_COMMON_PATTERNS",
		})
	}

	return errors, warnings
}
