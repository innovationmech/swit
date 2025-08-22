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

// Package config provides hierarchical configuration management for switctl.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// ConfigLevel represents configuration hierarchy levels.
type ConfigLevel int

const (
	// Global configuration (system-wide).
	GlobalLevel ConfigLevel = iota
	// User configuration (user-specific).
	UserLevel
	// Project configuration (project-specific).
	ProjectLevel
	// Environment configuration (environment variables).
	EnvironmentLevel
)

// String returns the string representation of ConfigLevel.
func (cl ConfigLevel) String() string {
	switch cl {
	case GlobalLevel:
		return "global"
	case UserLevel:
		return "user"
	case ProjectLevel:
		return "project"
	case EnvironmentLevel:
		return "environment"
	default:
		return "unknown"
	}
}

// HierarchicalConfigManager implements a layered configuration system
// with support for multiple configuration sources and merging strategies.
type HierarchicalConfigManager struct {
	mu             sync.RWMutex
	configs        map[ConfigLevel]*viper.Viper
	mergedConfig   *viper.Viper
	workDir        string
	configPaths    map[ConfigLevel]string
	envPrefix      string
	logger         interfaces.Logger
	validators     map[string]ConfigValidator
	defaults       map[string]interface{}
	watchers       []ConfigWatcher
	changeHandlers []ConfigChangeHandler
}

// ConfigValidator validates configuration values.
type ConfigValidator func(key string, value interface{}) error

// ConfigWatcher watches for configuration file changes.
type ConfigWatcher interface {
	Watch(path string, handler func(string)) error
	Stop() error
}

// ConfigChangeHandler handles configuration changes.
type ConfigChangeHandler func(level ConfigLevel, key string, oldValue, newValue interface{}) error

// NewHierarchicalConfigManager creates a new hierarchical configuration manager.
func NewHierarchicalConfigManager(workDir string, logger interfaces.Logger) *HierarchicalConfigManager {
	manager := &HierarchicalConfigManager{
		configs:        make(map[ConfigLevel]*viper.Viper),
		workDir:        workDir,
		configPaths:    make(map[ConfigLevel]string),
		envPrefix:      "SWITCTL",
		logger:         logger,
		validators:     make(map[string]ConfigValidator),
		defaults:       make(map[string]interface{}),
		watchers:       make([]ConfigWatcher, 0),
		changeHandlers: make([]ConfigChangeHandler, 0),
	}

	// Initialize default configuration paths
	manager.initializeConfigPaths()

	// Set up default validators
	manager.setupDefaultValidators()

	// Set up default values
	manager.setupDefaultValues()

	return manager
}

// initializeConfigPaths sets up default configuration file paths.
func (hcm *HierarchicalConfigManager) initializeConfigPaths() {
	// Global configuration in system config directory
	if globalDir := getGlobalConfigDir(); globalDir != "" {
		hcm.configPaths[GlobalLevel] = filepath.Join(globalDir, "switctl", "config.yaml")
	}

	// User configuration in user home directory
	if userDir := getUserConfigDir(); userDir != "" {
		hcm.configPaths[UserLevel] = filepath.Join(userDir, ".switctl", "config.yaml")
	}

	// Project configuration in working directory
	hcm.configPaths[ProjectLevel] = filepath.Join(hcm.workDir, ".switctl.yaml")
}

// setupDefaultValidators sets up built-in configuration validators.
func (hcm *HierarchicalConfigManager) setupDefaultValidators() {
	// Port validation
	hcm.validators["port"] = func(key string, value interface{}) error {
		if port, ok := value.(int); ok {
			if port < 1 || port > 65535 {
				return fmt.Errorf("invalid port %d for %s: must be between 1 and 65535", port, key)
			}
		}
		return nil
	}

	// URL validation
	hcm.validators["url"] = func(key string, value interface{}) error {
		if url, ok := value.(string); ok {
			if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				return fmt.Errorf("invalid URL %s for %s: must start with http:// or https://", url, key)
			}
		}
		return nil
	}

	// Required field validation
	hcm.validators["required"] = func(key string, value interface{}) error {
		if value == nil || value == "" {
			return fmt.Errorf("required field %s is missing or empty", key)
		}
		return nil
	}
}

// setupDefaultValues sets up default configuration values.
func (hcm *HierarchicalConfigManager) setupDefaultValues() {
	hcm.defaults = map[string]interface{}{
		"logging.level":       "info",
		"logging.format":      "text",
		"server.http.port":    9000,
		"server.grpc.port":    10000,
		"server.health.port":  8080,
		"server.metrics.port": 9090,
		"database.type":       "mysql",
		"database.host":       "localhost",
		"database.port":       3306,
		"cache.type":          "redis",
		"cache.host":          "localhost",
		"cache.port":          6379,
		"timeout.read":        "30s",
		"timeout.write":       "30s",
		"timeout.idle":        "60s",
		"auth.type":           "jwt",
		"auth.expiration":     "15m",
		"auth.algorithm":      "HS256",
	}
}

// Load loads configuration from all hierarchy levels and merges them.
func (hcm *HierarchicalConfigManager) Load() error {
	hcm.mu.Lock()
	defer hcm.mu.Unlock()

	// Load configurations from each level
	levels := []ConfigLevel{GlobalLevel, UserLevel, ProjectLevel, EnvironmentLevel}

	for _, level := range levels {
		if err := hcm.loadConfigLevel(level); err != nil {
			if hcm.logger != nil {
				hcm.logger.Warn(fmt.Sprintf("Failed to load %s configuration: %v", level.String(), err))
			}
			// Continue loading other levels even if one fails
		}
	}

	// Merge configurations with proper precedence
	if err := hcm.mergeConfigurations(); err != nil {
		return fmt.Errorf("failed to merge configurations: %w", err)
	}

	// Validate merged configuration
	if err := hcm.validateConfiguration(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	if hcm.logger != nil {
		hcm.logger.Info("Configuration loaded successfully from all hierarchy levels")
	}

	return nil
}

// loadConfigLevel loads configuration for a specific level.
func (hcm *HierarchicalConfigManager) loadConfigLevel(level ConfigLevel) error {
	v := viper.New()

	switch level {
	case GlobalLevel, UserLevel, ProjectLevel:
		configPath, exists := hcm.configPaths[level]
		if !exists {
			return fmt.Errorf("no config path defined for %s level", level.String())
		}

		// Check if config file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Config file doesn't exist, which is acceptable for optional levels
			return nil
		}

		// Set config file path and name
		v.SetConfigFile(configPath)

		// Read configuration
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}

	case EnvironmentLevel:
		// Load environment variables
		v.SetEnvPrefix(hcm.envPrefix)
		v.AutomaticEnv()
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

		// Bind all known configuration keys to environment variables
		allKeys := make(map[string]bool)

		// Add defaults
		for key := range hcm.defaults {
			allKeys[key] = true
		}

		// Add keys from all existing config levels
		for _, config := range hcm.configs {
			if config != nil {
				for _, key := range config.AllKeys() {
					allKeys[key] = true
				}
			}
		}

		// Check environment variables for all known keys
		for key := range allKeys {
			envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
			if envValue := os.Getenv(fmt.Sprintf("%s_%s", hcm.envPrefix, envKey)); envValue != "" {
				v.Set(key, envValue)
			}
		}
	}

	hcm.configs[level] = v
	return nil
}

// mergeConfigurations merges configurations from all levels with proper precedence.
func (hcm *HierarchicalConfigManager) mergeConfigurations() error {
	merged := viper.New()

	// Start with defaults
	for key, value := range hcm.defaults {
		merged.Set(key, value)
	}

	// Merge in order of precedence (lower precedence first)
	levels := []ConfigLevel{GlobalLevel, UserLevel, ProjectLevel, EnvironmentLevel}

	for _, level := range levels {
		config, exists := hcm.configs[level]
		if !exists || config == nil {
			continue
		}

		// Merge all settings from this level
		for _, key := range config.AllKeys() {
			value := config.Get(key)
			merged.Set(key, value)
		}
	}

	hcm.mergedConfig = merged
	return nil
}

// validateConfiguration validates the merged configuration.
func (hcm *HierarchicalConfigManager) validateConfiguration() error {
	if hcm.mergedConfig == nil {
		return fmt.Errorf("no merged configuration available")
	}

	var validationErrors []string

	// Validate each configuration key using registered validators
	for _, key := range hcm.mergedConfig.AllKeys() {
		value := hcm.mergedConfig.Get(key)

		// Check for specific validators based on key patterns
		for pattern, validator := range hcm.validators {
			if strings.Contains(key, pattern) {
				if err := validator(key, value); err != nil {
					validationErrors = append(validationErrors, err.Error())
				}
			}
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("configuration validation errors: %s", strings.Join(validationErrors, "; "))
	}

	return nil
}

// Get retrieves a configuration value by key.
func (hcm *HierarchicalConfigManager) Get(key string) interface{} {
	hcm.mu.RLock()
	defer hcm.mu.RUnlock()

	if hcm.mergedConfig == nil {
		return nil
	}

	return hcm.mergedConfig.Get(key)
}

// GetString retrieves a string configuration value.
func (hcm *HierarchicalConfigManager) GetString(key string) string {
	hcm.mu.RLock()
	defer hcm.mu.RUnlock()

	if hcm.mergedConfig == nil {
		return ""
	}

	return hcm.mergedConfig.GetString(key)
}

// GetInt retrieves an integer configuration value.
func (hcm *HierarchicalConfigManager) GetInt(key string) int {
	hcm.mu.RLock()
	defer hcm.mu.RUnlock()

	if hcm.mergedConfig == nil {
		return 0
	}

	return hcm.mergedConfig.GetInt(key)
}

// GetBool retrieves a boolean configuration value.
func (hcm *HierarchicalConfigManager) GetBool(key string) bool {
	hcm.mu.RLock()
	defer hcm.mu.RUnlock()

	if hcm.mergedConfig == nil {
		return false
	}

	return hcm.mergedConfig.GetBool(key)
}

// GetDuration retrieves a duration configuration value.
func (hcm *HierarchicalConfigManager) GetDuration(key string) time.Duration {
	hcm.mu.RLock()
	defer hcm.mu.RUnlock()

	if hcm.mergedConfig == nil {
		return 0
	}

	return hcm.mergedConfig.GetDuration(key)
}

// GetStringSlice retrieves a string slice configuration value.
func (hcm *HierarchicalConfigManager) GetStringSlice(key string) []string {
	hcm.mu.RLock()
	defer hcm.mu.RUnlock()

	if hcm.mergedConfig == nil {
		return nil
	}

	return hcm.mergedConfig.GetStringSlice(key)
}

// Set sets a configuration value at the project level.
func (hcm *HierarchicalConfigManager) Set(key string, value interface{}) {
	hcm.mu.Lock()
	defer hcm.mu.Unlock()

	// Set at project level by default
	if hcm.configs[ProjectLevel] == nil {
		hcm.configs[ProjectLevel] = viper.New()
	}

	// Get old value directly without calling Get() to avoid deadlock
	var oldValue interface{}
	if hcm.mergedConfig != nil {
		oldValue = hcm.mergedConfig.Get(key)
	}
	hcm.configs[ProjectLevel].Set(key, value)

	// Re-merge configurations
	if err := hcm.mergeConfigurations(); err != nil && hcm.logger != nil {
		hcm.logger.Error(fmt.Sprintf("Failed to re-merge configurations after setting %s: %v", key, err))
	}

	// Notify change handlers
	for _, handler := range hcm.changeHandlers {
		if err := handler(ProjectLevel, key, oldValue, value); err != nil && hcm.logger != nil {
			hcm.logger.Error(fmt.Sprintf("Configuration change handler error: %v", err))
		}
	}
}

// SetLevel sets a configuration value at a specific level.
func (hcm *HierarchicalConfigManager) SetLevel(level ConfigLevel, key string, value interface{}) error {
	hcm.mu.Lock()
	defer hcm.mu.Unlock()

	if level == EnvironmentLevel {
		return fmt.Errorf("cannot set environment level configuration programmatically")
	}

	if hcm.configs[level] == nil {
		hcm.configs[level] = viper.New()
	}

	// Get old value directly without calling Get() to avoid deadlock
	var oldValue interface{}
	if hcm.mergedConfig != nil {
		oldValue = hcm.mergedConfig.Get(key)
	}
	hcm.configs[level].Set(key, value)

	// Re-merge configurations
	if err := hcm.mergeConfigurations(); err != nil {
		return fmt.Errorf("failed to re-merge configurations: %w", err)
	}

	// Notify change handlers
	for _, handler := range hcm.changeHandlers {
		if err := handler(level, key, oldValue, value); err != nil {
			return fmt.Errorf("configuration change handler error: %w", err)
		}
	}

	return nil
}

// Validate validates the current configuration.
func (hcm *HierarchicalConfigManager) Validate() error {
	hcm.mu.RLock()
	defer hcm.mu.RUnlock()

	return hcm.validateConfiguration()
}

// Save saves the configuration to file at the specified level.
func (hcm *HierarchicalConfigManager) Save(path string) error {
	return hcm.SaveLevel(ProjectLevel, path)
}

// SaveLevel saves configuration for a specific level to a file.
func (hcm *HierarchicalConfigManager) SaveLevel(level ConfigLevel, path string) error {
	hcm.mu.RLock()
	config, exists := hcm.configs[level]
	hcm.mu.RUnlock()

	if !exists || config == nil {
		return fmt.Errorf("no configuration available for %s level", level.String())
	}

	// Determine file format based on extension
	ext := strings.ToLower(filepath.Ext(path))

	// Get all settings as a map
	settings := config.AllSettings()

	var data []byte
	var err error

	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(settings)
	case ".json":
		data, err = json.MarshalIndent(settings, "", "  ")
	default:
		// Default to YAML
		data, err = yaml.Marshal(settings)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if hcm.logger != nil {
		hcm.logger.Info(fmt.Sprintf("Configuration saved to %s", path))
	}

	return nil
}

// GetConfigPaths returns the configuration file paths for all levels.
func (hcm *HierarchicalConfigManager) GetConfigPaths() map[ConfigLevel]string {
	hcm.mu.RLock()
	defer hcm.mu.RUnlock()

	paths := make(map[ConfigLevel]string)
	for level, path := range hcm.configPaths {
		paths[level] = path
	}
	return paths
}

// SetConfigPath sets the configuration file path for a specific level.
func (hcm *HierarchicalConfigManager) SetConfigPath(level ConfigLevel, path string) {
	hcm.mu.Lock()
	defer hcm.mu.Unlock()

	hcm.configPaths[level] = path
}

// GetConfigLevel returns the configuration for a specific level.
func (hcm *HierarchicalConfigManager) GetConfigLevel(level ConfigLevel) map[string]interface{} {
	hcm.mu.RLock()
	defer hcm.mu.RUnlock()

	config, exists := hcm.configs[level]
	if !exists || config == nil {
		return make(map[string]interface{})
	}

	return config.AllSettings()
}

// GetMergedConfig returns all merged configuration as a map.
func (hcm *HierarchicalConfigManager) GetMergedConfig() map[string]interface{} {
	hcm.mu.RLock()
	defer hcm.mu.RUnlock()

	if hcm.mergedConfig == nil {
		return make(map[string]interface{})
	}

	return hcm.mergedConfig.AllSettings()
}

// AddValidator adds a configuration validator for a specific key pattern.
func (hcm *HierarchicalConfigManager) AddValidator(pattern string, validator ConfigValidator) {
	hcm.mu.Lock()
	defer hcm.mu.Unlock()

	hcm.validators[pattern] = validator
}

// AddChangeHandler adds a configuration change handler.
func (hcm *HierarchicalConfigManager) AddChangeHandler(handler ConfigChangeHandler) {
	hcm.mu.Lock()
	defer hcm.mu.Unlock()

	hcm.changeHandlers = append(hcm.changeHandlers, handler)
}

// Clone creates a deep copy of the configuration manager.
func (hcm *HierarchicalConfigManager) Clone() *HierarchicalConfigManager {
	hcm.mu.RLock()
	defer hcm.mu.RUnlock()

	clone := &HierarchicalConfigManager{
		configs:        make(map[ConfigLevel]*viper.Viper),
		workDir:        hcm.workDir,
		configPaths:    make(map[ConfigLevel]string),
		envPrefix:      hcm.envPrefix,
		logger:         hcm.logger,
		validators:     make(map[string]ConfigValidator),
		defaults:       make(map[string]interface{}),
		watchers:       make([]ConfigWatcher, 0),
		changeHandlers: make([]ConfigChangeHandler, 0),
	}

	// Copy config paths
	for level, path := range hcm.configPaths {
		clone.configPaths[level] = path
	}

	// Copy defaults
	for key, value := range hcm.defaults {
		clone.defaults[key] = value
	}

	// Copy validators
	for pattern, validator := range hcm.validators {
		clone.validators[pattern] = validator
	}

	return clone
}

// Reset resets the configuration manager to its initial state.
func (hcm *HierarchicalConfigManager) Reset() error {
	hcm.mu.Lock()
	defer hcm.mu.Unlock()

	// Clear all configurations
	hcm.configs = make(map[ConfigLevel]*viper.Viper)
	hcm.mergedConfig = nil

	// Reinitialize
	hcm.setupDefaultValidators()
	hcm.setupDefaultValues()

	if hcm.logger != nil {
		hcm.logger.Info("Configuration manager reset successfully")
	}

	return nil
}

// Merge merges external configuration into the manager at the specified level.
func (hcm *HierarchicalConfigManager) Merge(level ConfigLevel, config map[string]interface{}) error {
	hcm.mu.Lock()
	defer hcm.mu.Unlock()

	if hcm.configs[level] == nil {
		hcm.configs[level] = viper.New()
	}

	// Merge the configuration
	for key, value := range config {
		hcm.configs[level].Set(key, value)
	}

	// Re-merge all configurations
	if err := hcm.mergeConfigurations(); err != nil {
		return fmt.Errorf("failed to merge configurations: %w", err)
	}

	// Validate merged configuration
	if err := hcm.validateConfiguration(); err != nil {
		return fmt.Errorf("validation failed after merge: %w", err)
	}

	return nil
}

// Export exports configuration at the specified level to a map.
func (hcm *HierarchicalConfigManager) Export(level ConfigLevel) (map[string]interface{}, error) {
	hcm.mu.RLock()
	defer hcm.mu.RUnlock()

	config, exists := hcm.configs[level]
	if !exists || config == nil {
		return make(map[string]interface{}), nil
	}

	return deepCopy(config.AllSettings()), nil
}

// Helper functions

// getGlobalConfigDir returns the global configuration directory.
func getGlobalConfigDir() string {
	// On Unix systems, use /etc
	if os.Getuid() == 0 {
		return "/etc"
	}
	// For non-root users, fall back to user config
	return getUserConfigDir()
}

// getUserConfigDir returns the user configuration directory.
func getUserConfigDir() string {
	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		return configDir
	}

	if homeDir := os.Getenv("HOME"); homeDir != "" {
		return homeDir
	}

	return ""
}

// deepCopy creates a deep copy of a map[string]interface{}.
func deepCopy(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{})

	for key, value := range src {
		dst[key] = deepCopyValue(value)
	}

	return dst
}

// deepCopyValue creates a deep copy of an interface{} value.
func deepCopyValue(src interface{}) interface{} {
	if src == nil {
		return nil
	}

	// Use reflection to handle different types
	srcValue := reflect.ValueOf(src)

	switch srcValue.Kind() {
	case reflect.Map:
		dst := make(map[string]interface{})
		for _, key := range srcValue.MapKeys() {
			dst[key.String()] = deepCopyValue(srcValue.MapIndex(key).Interface())
		}
		return dst

	case reflect.Slice:
		length := srcValue.Len()
		dst := make([]interface{}, length)
		for i := 0; i < length; i++ {
			dst[i] = deepCopyValue(srcValue.Index(i).Interface())
		}
		return dst

	default:
		// For basic types, direct assignment is sufficient
		return src
	}
}
