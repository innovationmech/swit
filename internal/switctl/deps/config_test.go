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

package deps

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// ConfigTestSuite tests the configuration manager
type ConfigTestSuite struct {
	suite.Suite
	tempDir    string
	configPath string
	configMgr  *ConfigManager
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (s *ConfigTestSuite) SetupTest() {
	// Create temporary directory for test config files
	var err error
	s.tempDir, err = os.MkdirTemp("", "switctl-config-test")
	assert.NoError(s.T(), err)

	s.configPath = filepath.Join(s.tempDir, "switctl.yaml")

	// Create config manager with test options
	options := ConfigManagerOptions{
		ConfigName:  "switctl",
		ConfigType:  "yaml",
		ConfigPaths: []string{s.tempDir},
		EnvPrefix:   "SWITCTL_TEST",
	}

	s.configMgr, err = NewConfigManagerWithOptions(options)
	assert.NoError(s.T(), err)
}

func (s *ConfigTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

func (s *ConfigTestSuite) TestNewConfigManagerWithOptions() {
	options := ConfigManagerOptions{
		ConfigName:  "test-config",
		ConfigType:  "json",
		ConfigPaths: []string{"/test/path1", "/test/path2"},
		EnvPrefix:   "TEST",
	}

	cm, err := NewConfigManagerWithOptions(options)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), cm)
	assert.NotNil(s.T(), cm.viper)
}

func (s *ConfigTestSuite) TestDefaultConfigManagerOptions() {
	options := DefaultConfigManagerOptions()

	assert.Equal(s.T(), "switctl", options.ConfigName)
	assert.Equal(s.T(), "yaml", options.ConfigType)
	assert.Equal(s.T(), "SWITCTL", options.EnvPrefix)
	assert.NotEmpty(s.T(), options.ConfigPaths)
	assert.Contains(s.T(), options.ConfigPaths, ".")
}

func (s *ConfigTestSuite) TestConfigManager_Load_NoFile() {
	// Test loading when config file doesn't exist (should not error)
	err := s.configMgr.Load()
	assert.NoError(s.T(), err)
}

func (s *ConfigTestSuite) TestConfigManager_Load_ValidFile() {
	// Create a valid config file
	configContent := `
ui:
  color: true
  theme: dark
generator:
  output_directory: ./output
  overwrite: true
quality:
  coverage_threshold: 90.0
`
	err := os.WriteFile(s.configPath, []byte(configContent), 0644)
	assert.NoError(s.T(), err)

	// Load the config
	err = s.configMgr.Load()
	assert.NoError(s.T(), err)

	// Test values
	assert.True(s.T(), s.configMgr.GetBool("ui.color"))
	assert.Equal(s.T(), "dark", s.configMgr.GetString("ui.theme"))
	assert.Equal(s.T(), "./output", s.configMgr.GetString("generator.output_directory"))
	assert.True(s.T(), s.configMgr.GetBool("generator.overwrite"))
	assert.Equal(s.T(), 90.0, s.configMgr.Get("quality.coverage_threshold"))
}

func (s *ConfigTestSuite) TestConfigManager_Load_InvalidFile() {
	// Create an invalid config file
	configContent := `
invalid yaml content: [
  unclosed bracket
`
	err := os.WriteFile(s.configPath, []byte(configContent), 0644)
	assert.NoError(s.T(), err)

	// Load should return error
	err = s.configMgr.Load()
	assert.Error(s.T(), err)

	// Check error type
	switctlErr, ok := err.(*interfaces.SwitctlError)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), interfaces.ErrCodeConfigParseError, switctlErr.Code)
}

func (s *ConfigTestSuite) TestConfigManager_GettersAndSetters() {
	// Test Set and Get
	s.configMgr.Set("test.string", "test-value")
	s.configMgr.Set("test.int", 42)
	s.configMgr.Set("test.bool", true)
	s.configMgr.Set("test.float", 3.14)

	assert.Equal(s.T(), "test-value", s.configMgr.Get("test.string"))
	assert.Equal(s.T(), "test-value", s.configMgr.GetString("test.string"))
	assert.Equal(s.T(), 42, s.configMgr.GetInt("test.int"))
	assert.True(s.T(), s.configMgr.GetBool("test.bool"))

	// Test default values
	assert.Equal(s.T(), "", s.configMgr.GetString("non.existent"))
	assert.Equal(s.T(), 0, s.configMgr.GetInt("non.existent"))
	assert.False(s.T(), s.configMgr.GetBool("non.existent"))
}

func (s *ConfigTestSuite) TestConfigManager_Defaults() {
	// Test that defaults are set correctly
	assert.True(s.T(), s.configMgr.GetBool("ui.color"))
	assert.True(s.T(), s.configMgr.GetBool("ui.progress"))
	assert.True(s.T(), s.configMgr.GetBool("ui.interactive"))
	assert.Equal(s.T(), "default", s.configMgr.GetString("ui.theme"))

	assert.Equal(s.T(), "./generated", s.configMgr.GetString("generator.output_directory"))
	assert.False(s.T(), s.configMgr.GetBool("generator.overwrite"))
	assert.True(s.T(), s.configMgr.GetBool("generator.backup"))
	assert.True(s.T(), s.configMgr.GetBool("generator.format"))

	assert.Equal(s.T(), 80.0, s.configMgr.Get("quality.coverage_threshold"))
	assert.True(s.T(), s.configMgr.GetBool("quality.lint_enabled"))

	assert.Equal(s.T(), "info", s.configMgr.GetString("logging.level"))
	assert.Equal(s.T(), "text", s.configMgr.GetString("logging.format"))
}

func (s *ConfigTestSuite) TestConfigManager_Validate_Valid() {
	// Set valid values and clear template directory to avoid validation error
	s.configMgr.Set("quality.coverage_threshold", 85.0)
	s.configMgr.Set("template.directory", "") // Clear template directory to avoid validation error

	err := s.configMgr.Validate()
	assert.NoError(s.T(), err)
}

func (s *ConfigTestSuite) TestConfigManager_Validate_InvalidCoverage() {
	// Set invalid coverage threshold and clear template directory to avoid multiple errors
	s.configMgr.Set("template.directory", "")
	s.configMgr.Set("quality.coverage_threshold", 150.0)

	err := s.configMgr.Validate()
	assert.Error(s.T(), err)

	validationResult, ok := err.(*interfaces.ValidationResult)
	assert.True(s.T(), ok)
	assert.False(s.T(), validationResult.Valid)
	assert.Len(s.T(), validationResult.Errors, 1)
	assert.Equal(s.T(), "quality.coverage_threshold", validationResult.Errors[0].Field)
	assert.Contains(s.T(), validationResult.Errors[0].Message, "between 0 and 100")
}

func (s *ConfigTestSuite) TestConfigManager_Validate_NonExistentTemplate() {
	// Set non-existent template directory
	nonExistentDir := filepath.Join(s.tempDir, "non-existent")
	s.configMgr.Set("template.directory", nonExistentDir)

	err := s.configMgr.Validate()
	assert.Error(s.T(), err)

	validationResult, ok := err.(*interfaces.ValidationResult)
	assert.True(s.T(), ok)
	assert.False(s.T(), validationResult.Valid)
	assert.Len(s.T(), validationResult.Errors, 1)
	assert.Equal(s.T(), "template.directory", validationResult.Errors[0].Field)
	assert.Contains(s.T(), validationResult.Errors[0].Message, "does not exist")
}

func (s *ConfigTestSuite) TestConfigManager_Validate_InvalidOutputDirectory() {
	// Set output directory with non-existent parent and clear template directory
	s.configMgr.Set("template.directory", "")
	invalidDir := "/non/existent/parent/output"
	s.configMgr.Set("generator.output_directory", invalidDir)

	err := s.configMgr.Validate()
	assert.Error(s.T(), err)

	validationResult, ok := err.(*interfaces.ValidationResult)
	assert.True(s.T(), ok)
	assert.False(s.T(), validationResult.Valid)
	assert.Len(s.T(), validationResult.Errors, 1)
	assert.Equal(s.T(), "generator.output_directory", validationResult.Errors[0].Field)
	assert.Contains(s.T(), validationResult.Errors[0].Message, "Parent directory does not exist")
}

func (s *ConfigTestSuite) TestConfigManager_Save() {
	// Set some values
	s.configMgr.Set("test.key", "test-value")
	s.configMgr.Set("test.number", 123)

	// Save to file
	savePath := filepath.Join(s.tempDir, "saved-config.yaml")
	err := s.configMgr.Save(savePath)
	assert.NoError(s.T(), err)

	// Verify file exists and contains data
	assert.FileExists(s.T(), savePath)

	content, err := os.ReadFile(savePath)
	assert.NoError(s.T(), err)
	assert.Contains(s.T(), string(content), "test-value")
	assert.Contains(s.T(), string(content), "123")
}

func (s *ConfigTestSuite) TestConfigManager_Save_Error() {
	// Try to save to invalid path
	invalidPath := "/invalid/path/config.yaml"
	err := s.configMgr.Save(invalidPath)
	assert.Error(s.T(), err)

	switctlErr, ok := err.(*interfaces.SwitctlError)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), interfaces.ErrCodeConfigSaveError, switctlErr.Code)
}

func (s *ConfigTestSuite) TestConfigManager_LoadFromFile() {
	// Create a test config file
	configContent := `
custom:
  value: from-file
  number: 999
`
	testConfigPath := filepath.Join(s.tempDir, "custom-config.yaml")
	err := os.WriteFile(testConfigPath, []byte(configContent), 0644)
	assert.NoError(s.T(), err)

	// Load from specific file
	err = s.configMgr.LoadFromFile(testConfigPath)
	assert.NoError(s.T(), err)

	// Verify values loaded
	assert.Equal(s.T(), "from-file", s.configMgr.GetString("custom.value"))
	assert.Equal(s.T(), 999, s.configMgr.GetInt("custom.number"))
}

func (s *ConfigTestSuite) TestConfigManager_LoadFromFile_Error() {
	// Try to load non-existent file
	err := s.configMgr.LoadFromFile("/non/existent/file.yaml")
	assert.Error(s.T(), err)

	switctlErr, ok := err.(*interfaces.SwitctlError)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), interfaces.ErrCodeConfigParseError, switctlErr.Code)
}

func (s *ConfigTestSuite) TestConfigManager_Merge() {
	// Create another config manager
	tempDir2, err := os.MkdirTemp("", "switctl-config-test-2")
	assert.NoError(s.T(), err)
	defer os.RemoveAll(tempDir2)

	options2 := ConfigManagerOptions{
		ConfigName:  "config2",
		ConfigType:  "yaml",
		ConfigPaths: []string{tempDir2},
		EnvPrefix:   "TEST2",
	}
	configMgr2, err := NewConfigManagerWithOptions(options2)
	assert.NoError(s.T(), err)

	// Set different values in each manager
	s.configMgr.Set("key1", "value1")
	s.configMgr.Set("common", "original")

	configMgr2.Set("key2", "value2")
	configMgr2.Set("common", "override")

	// Merge configMgr2 into s.configMgr
	err = s.configMgr.Merge(configMgr2)
	assert.NoError(s.T(), err)

	// Verify merged values
	assert.Equal(s.T(), "value1", s.configMgr.GetString("key1"))
	assert.Equal(s.T(), "value2", s.configMgr.GetString("key2"))
	assert.Equal(s.T(), "override", s.configMgr.GetString("common"))
}

func (s *ConfigTestSuite) TestConfigManager_AllSettings() {
	s.configMgr.Set("test.key1", "value1")
	s.configMgr.Set("test.key2", 42)

	settings := s.configMgr.AllSettings()
	assert.NotEmpty(s.T(), settings)

	// Check that our test values are included
	testSettings, ok := settings["test"].(map[string]interface{})
	assert.True(s.T(), ok)
	assert.Equal(s.T(), "value1", testSettings["key1"])
	assert.Equal(s.T(), 42, testSettings["key2"])
}

func (s *ConfigTestSuite) TestConfigManager_GetConfigFilePath() {
	// Initially no file loaded
	path := s.configMgr.GetConfigFilePath()
	assert.Empty(s.T(), path)

	// Create and load a config file
	configContent := "test: value"
	err := os.WriteFile(s.configPath, []byte(configContent), 0644)
	assert.NoError(s.T(), err)

	err = s.configMgr.Load()
	assert.NoError(s.T(), err)

	// Now should return the loaded file path
	path = s.configMgr.GetConfigFilePath()
	assert.Equal(s.T(), s.configPath, path)
}

func (s *ConfigTestSuite) TestConfigManager_GetUIConfig() {
	// Set some UI config values
	s.configMgr.Set("ui.color", false)
	s.configMgr.Set("ui.theme", "dark")
	s.configMgr.Set("ui.progress", false)
	s.configMgr.Set("ui.interactive", false)

	config := s.configMgr.GetUIConfig()

	assert.False(s.T(), config.Color)
	assert.Equal(s.T(), "dark", config.Theme)
	assert.False(s.T(), config.Progress)
	assert.False(s.T(), config.Interactive)
}

func (s *ConfigTestSuite) TestConfigManager_GetGeneratorConfig() {
	s.configMgr.Set("generator.output_directory", "/custom/output")
	s.configMgr.Set("generator.overwrite", true)
	s.configMgr.Set("generator.backup", false)
	s.configMgr.Set("generator.format", false)

	config := s.configMgr.GetGeneratorConfig()

	assert.Equal(s.T(), "/custom/output", config.OutputDirectory)
	assert.True(s.T(), config.Overwrite)
	assert.False(s.T(), config.Backup)
	assert.False(s.T(), config.Format)
}

func (s *ConfigTestSuite) TestConfigManager_GetTemplateConfig() {
	s.configMgr.Set("template.directory", "/custom/templates")
	s.configMgr.Set("template.cache", false)
	s.configMgr.Set("template.reload", true)

	config := s.configMgr.GetTemplateConfig()

	assert.Equal(s.T(), "/custom/templates", config.Directory)
	assert.False(s.T(), config.Cache)
	assert.True(s.T(), config.Reload)
}

func (s *ConfigTestSuite) TestConfigManager_GetQualityConfig() {
	s.configMgr.Set("quality.coverage_threshold", 95.0)
	s.configMgr.Set("quality.lint_enabled", false)
	s.configMgr.Set("quality.security_scan", false)
	s.configMgr.Set("quality.format_check", false)
	s.configMgr.Set("quality.import_check", false)
	s.configMgr.Set("quality.copyright_check", false)

	config := s.configMgr.GetQualityConfig()

	assert.Equal(s.T(), 95.0, config.CoverageThreshold)
	assert.False(s.T(), config.LintEnabled)
	assert.False(s.T(), config.SecurityScan)
	assert.False(s.T(), config.FormatCheck)
	assert.False(s.T(), config.ImportCheck)
	assert.False(s.T(), config.CopyrightCheck)
}

func (s *ConfigTestSuite) TestConfigManager_GenerateConfigTemplate() {
	template, err := s.configMgr.GenerateConfigTemplate()
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), template)

	// Verify template contains expected sections
	assert.Contains(s.T(), template, "ui:")
	assert.Contains(s.T(), template, "generator:")
	assert.Contains(s.T(), template, "template:")
	assert.Contains(s.T(), template, "quality:")
	assert.Contains(s.T(), template, "plugins:")
	assert.Contains(s.T(), template, "logging:")
	assert.Contains(s.T(), template, "dev:")
	assert.Contains(s.T(), template, "project:")

	// Verify some specific values
	assert.Contains(s.T(), template, "color: true")
	assert.Contains(s.T(), template, "coverage_threshold: 80.0")
	assert.Contains(s.T(), template, "level: info")
}

// Environment variable tests
func (s *ConfigTestSuite) TestConfigManager_EnvironmentVariables() {
	// Set environment variable
	envKey := "SWITCTL_TEST_UI_COLOR"
	envValue := "false"
	os.Setenv(envKey, envValue)
	defer os.Unsetenv(envKey)

	// Create new config manager to pick up env var
	options := ConfigManagerOptions{
		ConfigName:  "switctl",
		ConfigType:  "yaml",
		ConfigPaths: []string{s.tempDir},
		EnvPrefix:   "SWITCTL_TEST",
	}
	envConfigMgr, err := NewConfigManagerWithOptions(options)
	assert.NoError(s.T(), err)

	// Load configuration
	err = envConfigMgr.Load()
	assert.NoError(s.T(), err)

	// Environment variable should override default
	assert.False(s.T(), envConfigMgr.GetBool("ui.color"))
}

// Concurrency tests
func (s *ConfigTestSuite) TestConfigManager_ConcurrentAccess() {
	const numGoroutines = 100

	// Set initial values
	s.configMgr.Set("concurrent.test", "initial")

	// Start multiple goroutines reading and writing
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			// Read operations
			_ = s.configMgr.GetString("concurrent.test")
			_ = s.configMgr.GetBool("ui.color")
			_ = s.configMgr.GetInt("quality.coverage_threshold")

			// Write operations
			s.configMgr.Set(fmt.Sprintf("concurrent.goroutine.%d", id), fmt.Sprintf("value-%d", id))

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify some values were set
	assert.Equal(s.T(), "value-0", s.configMgr.GetString("concurrent.goroutine.0"))
	assert.Equal(s.T(), fmt.Sprintf("value-%d", numGoroutines-1), s.configMgr.GetString(fmt.Sprintf("concurrent.goroutine.%d", numGoroutines-1)))
}

// Performance benchmarks
func BenchmarkConfigManager_GetString(b *testing.B) {
	options := DefaultConfigManagerOptions()
	configMgr, _ := NewConfigManagerWithOptions(options)
	configMgr.Set("benchmark.key", "benchmark-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = configMgr.GetString("benchmark.key")
	}
}

func BenchmarkConfigManager_Set(b *testing.B) {
	options := DefaultConfigManagerOptions()
	configMgr, _ := NewConfigManagerWithOptions(options)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		configMgr.Set(fmt.Sprintf("benchmark.key.%d", i), fmt.Sprintf("value-%d", i))
	}
}

func BenchmarkConfigManager_Validate(b *testing.B) {
	options := DefaultConfigManagerOptions()
	configMgr, _ := NewConfigManagerWithOptions(options)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = configMgr.Validate()
	}
}

// Edge cases and error conditions
func (s *ConfigTestSuite) TestConfigManager_EdgeCases() {
	// Test empty string keys
	s.configMgr.Set("", "empty-key-value")
	assert.Equal(s.T(), "empty-key-value", s.configMgr.GetString(""))

	// Test nested empty keys
	s.configMgr.Set("parent.", "nested-empty")
	assert.Equal(s.T(), "nested-empty", s.configMgr.GetString("parent."))

	// Test special characters in keys
	specialKey := "key.with-special_chars@123"
	s.configMgr.Set(specialKey, "special-value")
	assert.Equal(s.T(), "special-value", s.configMgr.GetString(specialKey))

	// Test nil values
	s.configMgr.Set("nil.test", nil)
	assert.Nil(s.T(), s.configMgr.Get("nil.test"))
	assert.Equal(s.T(), "", s.configMgr.GetString("nil.test"))
	assert.Equal(s.T(), 0, s.configMgr.GetInt("nil.test"))
	assert.False(s.T(), s.configMgr.GetBool("nil.test"))
}

// Mock test to verify error message formatting
func (s *ConfigTestSuite) TestConfigTypes() {
	// Test ConfigTemplate structure
	template := ConfigTemplate{
		Name:        "test-template",
		Description: "Test configuration template",
		Config: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
		Tags: []string{"test", "config"},
	}

	assert.Equal(s.T(), "test-template", template.Name)
	assert.Equal(s.T(), "Test configuration template", template.Description)
	assert.Len(s.T(), template.Config, 2)
	assert.Len(s.T(), template.Tags, 2)
	assert.Contains(s.T(), template.Tags, "test")
	assert.Contains(s.T(), template.Tags, "config")

	// Test config structures
	uiConfig := UIConfig{
		Color:       true,
		Progress:    true,
		Interactive: false,
		Theme:       "dark",
	}
	assert.True(s.T(), uiConfig.Color)
	assert.Equal(s.T(), "dark", uiConfig.Theme)

	generatorConfig := GeneratorConfig{
		OutputDirectory: "/test/output",
		Overwrite:       true,
		Backup:          false,
		Format:          true,
	}
	assert.Equal(s.T(), "/test/output", generatorConfig.OutputDirectory)
	assert.True(s.T(), generatorConfig.Overwrite)

	templateConfig := TemplateConfig{
		Directory: "/test/templates",
		Cache:     true,
		Reload:    false,
	}
	assert.Equal(s.T(), "/test/templates", templateConfig.Directory)
	assert.True(s.T(), templateConfig.Cache)

	qualityConfig := QualityConfig{
		CoverageThreshold: 85.0,
		LintEnabled:       true,
		SecurityScan:      true,
		FormatCheck:       false,
		ImportCheck:       true,
		CopyrightCheck:    false,
	}
	assert.Equal(s.T(), 85.0, qualityConfig.CoverageThreshold)
	assert.True(s.T(), qualityConfig.LintEnabled)
	assert.False(s.T(), qualityConfig.FormatCheck)
}
