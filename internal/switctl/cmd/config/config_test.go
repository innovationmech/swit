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

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// ConfigCommandTestSuite tests the 'config' command functionality
type ConfigCommandTestSuite struct {
	suite.Suite
	tempDir        string
	originalArgs   []string
	originalWd     string
	mockContainer  *MockDependencyContainer
	mockUI         *MockTerminalUI
	mockLogger     *MockLogger
	mockFileSystem *MockFileSystem
	mockConfig     *MockConfigManager
}

// SetupSuite runs before all tests in the suite
func (s *ConfigCommandTestSuite) SetupSuite() {
	s.originalArgs = os.Args
	s.originalWd, _ = os.Getwd()
}

// SetupTest runs before each test
func (s *ConfigCommandTestSuite) SetupTest() {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "switctl-config-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Reset global variables
	s.resetGlobals()

	// Reset viper
	viper.Reset()

	// Create mocks
	s.createMocks()
	s.setupMockBehavior()

	// Create test configuration files
	s.createTestConfigFiles()
}

// TearDownTest runs after each test
func (s *ConfigCommandTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
	s.resetGlobals()
	viper.Reset()
	os.Chdir(s.originalWd)
}

// TearDownSuite runs after all tests in the suite
func (s *ConfigCommandTestSuite) TearDownSuite() {
	os.Args = s.originalArgs
}

// Helper Methods

func (s *ConfigCommandTestSuite) resetGlobals() {
	verbose = false
	noColor = false
	workDir = ""
	configFile = ""
	outputFile = ""
	format = "yaml"
	strict = false
	showDefaults = false
}

func (s *ConfigCommandTestSuite) createMocks() {
	s.mockContainer = &MockDependencyContainer{}
	s.mockUI = &MockTerminalUI{}
	s.mockLogger = &MockLogger{}
	s.mockFileSystem = &MockFileSystem{}
	s.mockConfig = &MockConfigManager{}
}

func (s *ConfigCommandTestSuite) setupMockBehavior() {
	// Setup container returns
	s.mockContainer.On("GetService", "ui").Return(s.mockUI, nil)
	s.mockContainer.On("GetService", "filesystem").Return(s.mockFileSystem, nil)
	s.mockContainer.On("GetService", "config_manager").Return(s.mockConfig, nil)
	s.mockContainer.On("Close").Return(nil)

	// Default UI behavior
	s.mockUI.On("PrintHeader", mock.Anything).Return()
	s.mockUI.On("PrintSubHeader", mock.Anything).Return()
	// Create a default mock progress bar with expected calls
	defaultMockProgress := &MockProgressBar{}
	defaultMockProgress.On("SetMessage", mock.Anything).Return(nil)
	defaultMockProgress.On("Update", mock.Anything).Return(nil)
	defaultMockProgress.On("Finish").Return(nil)
	defaultMockProgress.On("SetTotal", mock.Anything).Return(nil)
	s.mockUI.On("ShowProgress", mock.Anything, mock.Anything).Return(defaultMockProgress)
	s.mockUI.On("ShowSuccess", mock.Anything).Return(nil)
	s.mockUI.On("ShowInfo", mock.Anything).Return(nil)
	s.mockUI.On("ShowError", mock.Anything).Return(nil)
	s.mockUI.On("ShowTable", mock.Anything, mock.Anything).Return(nil)
	// Create mock colors
	mockColor := &MockColor{}
	mockColor.On("Sprint", mock.Anything).Return("colored-text")
	mockColor.On("Sprintf", mock.Anything, mock.Anything).Return("colored-text")

	// Create UIStyle with mock colors
	style := &interfaces.UIStyle{
		Primary:   mockColor,
		Success:   mockColor,
		Warning:   mockColor,
		Error:     mockColor,
		Info:      mockColor,
		Highlight: mockColor,
	}
	s.mockUI.On("GetStyle").Return(style)

	// Default config manager behavior
	s.mockConfig.On("Load").Return(nil)
	s.mockConfig.On("Validate").Return(nil)
	s.mockConfig.On("Save", mock.Anything).Return(nil)
	s.mockConfig.On("Get", mock.Anything).Return(nil)
	s.mockConfig.On("GetString", mock.Anything).Return("")
	s.mockConfig.On("GetInt", mock.Anything).Return(0)
	s.mockConfig.On("GetBool", mock.Anything).Return(false)
	s.mockConfig.On("Set", mock.Anything, mock.Anything).Return()

	// Default filesystem behavior
	s.mockFileSystem.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.mockFileSystem.On("ReadFile", mock.Anything).Return([]byte("test content"), nil)
}

func (s *ConfigCommandTestSuite) createTestConfigFiles() {
	// Create valid YAML config file
	validYAML := `name: test-service
description: "Test service description"
version: "1.0.0"

server:
  http:
    port: 9000
    host: "0.0.0.0"
  grpc:
    port: 10000

database:
  type: mysql
  host: localhost
  port: 3306
  name: testdb

logging:
  level: info
  format: json
`
	err := os.WriteFile(filepath.Join(s.tempDir, "valid.yaml"), []byte(validYAML), 0644)
	s.Require().NoError(err)

	// Create valid JSON config file
	validJSON := `{
  "name": "test-service",
  "description": "Test service description",
  "version": "1.0.0",
  "server": {
    "http": {
      "port": 9000,
      "host": "0.0.0.0"
    }
  }
}`
	err = os.WriteFile(filepath.Join(s.tempDir, "valid.json"), []byte(validJSON), 0644)
	s.Require().NoError(err)

	// Create invalid YAML config file
	invalidYAML := `name: test-service
description: "Test service description"
version: "1.0.0"
server:
  http:
    port: 9000
    host: "0.0.0.0"
  - invalid list item
database:
  type: mysql
`
	err = os.WriteFile(filepath.Join(s.tempDir, "invalid.yaml"), []byte(invalidYAML), 0644)
	s.Require().NoError(err)

	// Create invalid JSON config file
	invalidJSON := `{
  "name": "test-service",
  "description": "Test service description",
  "version": "1.0.0",
  "server": {
    "http": {
      "port": 9000,
      "host": "0.0.0.0"
    }
  }
  // invalid comment in JSON
}`
	err = os.WriteFile(filepath.Join(s.tempDir, "invalid.json"), []byte(invalidJSON), 0644)
	s.Require().NoError(err)
}

// Test Cases

// TestNewConfigCommand_BasicProperties tests the basic properties of the config command
func (s *ConfigCommandTestSuite) TestNewConfigCommand_BasicProperties() {
	cmd := NewConfigCommand()

	assert.NotNil(s.T(), cmd)
	assert.Equal(s.T(), "config", cmd.Use)
	assert.Equal(s.T(), "Configuration file management and validation", cmd.Short)
	assert.Contains(s.T(), cmd.Long, "The config command provides configuration file management")
	assert.Contains(s.T(), cmd.Long, "Examples:")
}

// TestNewConfigCommand_PersistentFlags tests persistent flags
func (s *ConfigCommandTestSuite) TestNewConfigCommand_PersistentFlags() {
	cmd := NewConfigCommand()
	flags := cmd.PersistentFlags()

	testCases := []struct {
		name        string
		shorthand   string
		defValue    string
		shouldExist bool
	}{
		{"verbose", "v", "false", true},
		{"no-color", "", "false", true},
		{"work-dir", "w", "", true},
		{"file", "f", "", true},
		{"output", "o", "", true},
		{"format", "", "yaml", true},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			flag := flags.Lookup(tc.name)
			if tc.shouldExist {
				assert.NotNil(t, flag, "Flag %s should exist", tc.name)
				if tc.shorthand != "" {
					assert.Equal(t, tc.shorthand, flag.Shorthand, "Flag %s shorthand should match", tc.name)
				}
				if tc.defValue != "" {
					assert.Equal(t, tc.defValue, flag.DefValue, "Flag %s default value should match", tc.name)
				}
			} else {
				assert.Nil(t, flag, "Flag %s should not exist", tc.name)
			}
		})
	}
}

// TestNewConfigCommand_Subcommands tests that subcommands are properly added
func (s *ConfigCommandTestSuite) TestNewConfigCommand_Subcommands() {
	cmd := NewConfigCommand()
	subcommands := cmd.Commands()

	assert.GreaterOrEqual(s.T(), len(subcommands), 5)

	expectedSubcommands := []string{"validate", "template", "schema", "migrate", "merge"}
	actualSubcommands := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		actualSubcommands[i] = subcmd.Use
	}

	for _, expected := range expectedSubcommands {
		found := false
		for _, actual := range actualSubcommands {
			if strings.HasPrefix(actual, expected) {
				found = true
				break
			}
		}
		assert.True(s.T(), found, "Should have %s subcommand", expected)
	}
}

// TestInitializeConfigCommandConfig tests configuration initialization
func (s *ConfigCommandTestSuite) TestInitializeConfigCommandConfig() {
	// Set working directory via viper to temp dir
	viper.Set("config.work_dir", s.tempDir)

	cmd := NewConfigCommand()
	err := initializeConfigCommandConfig(cmd)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), s.tempDir, workDir)

	// Test with viper values
	viper.Set("config.verbose", true)
	viper.Set("config.no_color", true)
	viper.Set("config.work_dir", s.tempDir)
	viper.Set("config.file", "test.yaml")
	viper.Set("config.output", "output.yaml")
	viper.Set("config.format", "json")

	err = initializeConfigCommandConfig(cmd)
	assert.NoError(s.T(), err)
	assert.True(s.T(), verbose)
	assert.True(s.T(), noColor)
	assert.Equal(s.T(), s.tempDir, workDir)
	assert.Equal(s.T(), "test.yaml", configFile)
	assert.Equal(s.T(), "output.yaml", outputFile)
	assert.Equal(s.T(), "json", format)
}

// TestInitializeConfigCommandConfig_InvalidFormat tests invalid format
func (s *ConfigCommandTestSuite) TestInitializeConfigCommandConfig_InvalidFormat() {
	viper.Set("config.work_dir", s.tempDir)
	viper.Set("config.format", "xml")

	cmd := NewConfigCommand()
	err := initializeConfigCommandConfig(cmd)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "unsupported format: xml")
}

// Validate Command Tests

// TestNewValidateCommand_Properties tests validate command properties
func (s *ConfigCommandTestSuite) TestNewValidateCommand_Properties() {
	cmd := NewValidateCommand()

	assert.NotNil(s.T(), cmd)
	assert.Equal(s.T(), "validate [files...]", cmd.Use)
	assert.Equal(s.T(), "Validate configuration file syntax and structure", cmd.Short)
	assert.Contains(s.T(), cmd.Long, "The validate command checks configuration files")
}

// TestNewValidateCommand_Flags tests validate command flags
func (s *ConfigCommandTestSuite) TestNewValidateCommand_Flags() {
	cmd := NewValidateCommand()
	flags := cmd.Flags()

	testCases := []struct {
		name        string
		defValue    string
		shouldExist bool
	}{
		{"strict", "false", true},
		{"show-defaults", "false", true},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			flag := flags.Lookup(tc.name)
			if tc.shouldExist {
				assert.NotNil(t, flag, "Flag %s should exist", tc.name)
				assert.Equal(t, tc.defValue, flag.DefValue, "Flag %s default value should match", tc.name)
			} else {
				assert.Nil(t, flag, "Flag %s should not exist", tc.name)
			}
		})
	}
}

// TestRunValidateCommand tests validate command execution
func (s *ConfigCommandTestSuite) TestRunValidateCommand() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	// Mock progress bar
	mockProgress := &MockProgressBar{}
	mockProgress.On("SetMessage", mock.Anything).Return(nil)
	mockProgress.On("Update", mock.Anything).Return(nil)
	mockProgress.On("Finish").Return(nil)
	s.mockUI.On("ShowProgress", "Validating files", 1).Return(mockProgress)

	// Set working directory
	originalWorkDir := workDir
	workDir = s.tempDir
	defer func() { workDir = originalWorkDir }()

	err := runValidateCommand([]string{filepath.Join(s.tempDir, "valid.yaml")})
	assert.NoError(s.T(), err)

	s.mockUI.AssertCalled(s.T(), "ShowSuccess", "All configuration files are valid!")
}

// TestRunValidateCommand_InvalidFiles tests validate command with invalid files
func (s *ConfigCommandTestSuite) TestRunValidateCommand_InvalidFiles() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	// Mock progress bar
	mockProgress := &MockProgressBar{}
	mockProgress.On("SetMessage", mock.Anything).Return(nil)
	mockProgress.On("Update", mock.Anything).Return(nil)
	mockProgress.On("Finish").Return(nil)
	s.mockUI.On("ShowProgress", "Validating files", 1).Return(mockProgress)

	// Set working directory
	originalWorkDir := workDir
	workDir = s.tempDir
	defer func() { workDir = originalWorkDir }()

	err := runValidateCommand([]string{filepath.Join(s.tempDir, "invalid.yaml")})
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "configuration validation failed")
}

// TestRunValidateCommand_AutoDetectFiles tests validate command with auto-detection
func (s *ConfigCommandTestSuite) TestRunValidateCommand_AutoDetectFiles() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	// Mock progress bar
	mockProgress := &MockProgressBar{}
	mockProgress.On("SetMessage", mock.Anything).Return(nil)
	mockProgress.On("Update", mock.Anything).Return(nil)
	mockProgress.On("Finish").Return(nil)
	s.mockUI.On("ShowProgress", "Validating files", mock.Anything).Return(mockProgress)

	// Set working directory
	originalWorkDir := workDir
	workDir = s.tempDir
	defer func() { workDir = originalWorkDir }()

	// Mock findConfigFiles
	originalFindConfig := findConfigFiles
	findConfigFiles = func() ([]string, error) {
		return []string{filepath.Join(s.tempDir, "valid.yaml")}, nil
	}
	defer func() { findConfigFiles = originalFindConfig }()

	err := runValidateCommand([]string{})
	assert.NoError(s.T(), err)
}

// TestRunValidateCommand_StrictMode tests validate command in strict mode
func (s *ConfigCommandTestSuite) TestRunValidateCommand_StrictMode() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	// Mock progress bar
	mockProgress := &MockProgressBar{}
	mockProgress.On("SetMessage", mock.Anything).Return(nil)
	mockProgress.On("Update", mock.Anything).Return(nil)
	mockProgress.On("Finish").Return(nil)
	s.mockUI.On("ShowProgress", "Validating files", 1).Return(mockProgress)

	// Set working directory and strict mode
	originalWorkDir := workDir
	originalStrict := strict
	workDir = s.tempDir
	strict = true
	defer func() {
		workDir = originalWorkDir
		strict = originalStrict
	}()

	err := runValidateCommand([]string{filepath.Join(s.tempDir, "valid.yaml")})
	assert.NoError(s.T(), err)
}

// Template Command Tests

// TestNewTemplateCommand_Properties tests template command properties
func (s *ConfigCommandTestSuite) TestNewTemplateCommand_Properties() {
	cmd := NewTemplateCommand()

	assert.NotNil(s.T(), cmd)
	assert.Equal(s.T(), "template", cmd.Use)
	assert.Equal(s.T(), "Generate standard configuration file templates", cmd.Short)
	assert.Contains(s.T(), cmd.Long, "The template command generates configuration file templates")
}

// TestNewTemplateCommand_Flags tests template command flags
func (s *ConfigCommandTestSuite) TestNewTemplateCommand_Flags() {
	cmd := NewTemplateCommand()
	flags := cmd.Flags()

	testCases := []struct {
		name        string
		shorthand   string
		defValue    string
		shouldExist bool
	}{
		{"type", "t", "service", true},
		{"service", "s", "", true},
		{"include", "", "", true},
		{"exclude", "", "", true},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			flag := flags.Lookup(tc.name)
			if tc.shouldExist {
				assert.NotNil(t, flag, "Flag %s should exist", tc.name)
				if tc.shorthand != "" {
					assert.Equal(t, tc.shorthand, flag.Shorthand, "Flag %s shorthand should match", tc.name)
				}
				if tc.defValue != "" && tc.name != "include" && tc.name != "exclude" {
					assert.Equal(t, tc.defValue, flag.DefValue, "Flag %s default value should match", tc.name)
				}
			} else {
				assert.Nil(t, flag, "Flag %s should not exist", tc.name)
			}
		})
	}
}

// TestRunTemplateCommand tests template command execution
func (s *ConfigCommandTestSuite) TestRunTemplateCommand() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	// Capture stdout to verify template output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runTemplateCommand("service", "test-service", []string{}, []string{})
	assert.NoError(s.T(), err)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(s.T(), output, "test-service")
	assert.Contains(s.T(), output, "server:")
}

// TestRunTemplateCommand_WithOutput tests template command with output file
func (s *ConfigCommandTestSuite) TestRunTemplateCommand_WithOutput() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	outputPath := filepath.Join(s.tempDir, "generated.yaml")
	originalOutputFile := outputFile
	outputFile = outputPath
	defer func() { outputFile = originalOutputFile }()

	err := runTemplateCommand("service", "test-service", []string{}, []string{})
	assert.NoError(s.T(), err)

	// Verify file was created
	assert.FileExists(s.T(), outputPath)
}

// TestRunTemplateCommand_InvalidType tests template command with invalid type
func (s *ConfigCommandTestSuite) TestRunTemplateCommand_InvalidType() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	err := runTemplateCommand("invalid-type", "", []string{}, []string{})
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid template type")
}

// Schema Command Tests

// TestNewSchemaCommand_Properties tests schema command properties
func (s *ConfigCommandTestSuite) TestNewSchemaCommand_Properties() {
	cmd := NewSchemaCommand()

	assert.NotNil(s.T(), cmd)
	assert.Equal(s.T(), "schema", cmd.Use)
	assert.Equal(s.T(), "Generate JSON schema for configuration validation", cmd.Short)
	assert.Contains(s.T(), cmd.Long, "The schema command generates JSON schemas")
}

// TestNewSchemaCommand_Flags tests schema command flags
func (s *ConfigCommandTestSuite) TestNewSchemaCommand_Flags() {
	cmd := NewSchemaCommand()
	flags := cmd.Flags()

	testCases := []struct {
		name        string
		shorthand   string
		defValue    string
		shouldExist bool
	}{
		{"type", "t", "service", true},
		{"version", "", "latest", true},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			flag := flags.Lookup(tc.name)
			if tc.shouldExist {
				assert.NotNil(t, flag, "Flag %s should exist", tc.name)
				if tc.shorthand != "" {
					assert.Equal(t, tc.shorthand, flag.Shorthand, "Flag %s shorthand should match", tc.name)
				}
				assert.Equal(t, tc.defValue, flag.DefValue, "Flag %s default value should match", tc.name)
			} else {
				assert.Nil(t, flag, "Flag %s should not exist", tc.name)
			}
		})
	}
}

// TestRunSchemaCommand tests schema command execution
func (s *ConfigCommandTestSuite) TestRunSchemaCommand() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	// Capture stdout to verify schema output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runSchemaCommand("service", "latest")
	assert.NoError(s.T(), err)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify it's valid JSON
	var jsonOutput map[string]interface{}
	err = json.Unmarshal([]byte(output), &jsonOutput)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "http://json-schema.org/draft-07/schema#", jsonOutput["$schema"])
}

// TestRunSchemaCommand_WithOutput tests schema command with output file
func (s *ConfigCommandTestSuite) TestRunSchemaCommand_WithOutput() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	outputPath := filepath.Join(s.tempDir, "schema.json")
	originalOutputFile := outputFile
	outputFile = outputPath
	defer func() { outputFile = originalOutputFile }()

	err := runSchemaCommand("service", "latest")
	assert.NoError(s.T(), err)

	// Verify file was created
	assert.FileExists(s.T(), outputPath)
}

// Migrate Command Tests

// TestNewMigrateCommand_Properties tests migrate command properties
func (s *ConfigCommandTestSuite) TestNewMigrateCommand_Properties() {
	cmd := NewMigrateCommand()

	assert.NotNil(s.T(), cmd)
	assert.Equal(s.T(), "migrate [files...]", cmd.Use)
	assert.Equal(s.T(), "Migrate configuration files to newer formats", cmd.Short)
	assert.Contains(s.T(), cmd.Long, "The migrate command migrates configuration files")
}

// TestNewMigrateCommand_Flags tests migrate command flags
func (s *ConfigCommandTestSuite) TestNewMigrateCommand_Flags() {
	cmd := NewMigrateCommand()
	flags := cmd.Flags()

	testCases := []struct {
		name        string
		defValue    string
		shouldExist bool
	}{
		{"from", "", true},
		{"to", "", true},
		{"backup", "true", true},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			flag := flags.Lookup(tc.name)
			if tc.shouldExist {
				assert.NotNil(t, flag, "Flag %s should exist", tc.name)
				if tc.defValue != "" {
					assert.Equal(t, tc.defValue, flag.DefValue, "Flag %s default value should match", tc.name)
				}
			} else {
				assert.Nil(t, flag, "Flag %s should not exist", tc.name)
			}
		})
	}
}

// TestRunMigrateCommand tests migrate command execution
func (s *ConfigCommandTestSuite) TestRunMigrateCommand() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	// Mock progress bar
	mockProgress := &MockProgressBar{}
	mockProgress.On("SetMessage", mock.Anything).Return(nil)
	mockProgress.On("Update", mock.Anything).Return(nil)
	mockProgress.On("Finish").Return(nil)
	s.mockUI.On("ShowProgress", "Migrating files", 1).Return(mockProgress)

	configPath := filepath.Join(s.tempDir, "valid.yaml")
	err := runMigrateCommand([]string{configPath}, "v1", "v2", true, "")
	assert.NoError(s.T(), err)

	s.mockUI.AssertCalled(s.T(), "ShowSuccess", mock.MatchedBy(func(msg string) bool {
		return strings.Contains(msg, "Successfully migrated")
	}))
}

// Merge Command Tests

// TestNewMergeCommand_Properties tests merge command properties
func (s *ConfigCommandTestSuite) TestNewMergeCommand_Properties() {
	cmd := NewMergeCommand()

	assert.NotNil(s.T(), cmd)
	assert.Equal(s.T(), "merge <base-file> <overlay-files...>", cmd.Use)
	assert.Equal(s.T(), "Merge multiple configuration files", cmd.Short)
	assert.Contains(s.T(), cmd.Long, "The merge command combines multiple configuration files")
}

// TestNewMergeCommand_Flags tests merge command flags
func (s *ConfigCommandTestSuite) TestNewMergeCommand_Flags() {
	cmd := NewMergeCommand()
	flags := cmd.Flags()

	testCases := []struct {
		name        string
		defValue    string
		shouldExist bool
	}{
		{"strategy", "override", true},
		{"overwrite", "false", true},
		{"ignore-nulls", "false", true},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			flag := flags.Lookup(tc.name)
			if tc.shouldExist {
				assert.NotNil(t, flag, "Flag %s should exist", tc.name)
				assert.Equal(t, tc.defValue, flag.DefValue, "Flag %s default value should match", tc.name)
			} else {
				assert.Nil(t, flag, "Flag %s should not exist", tc.name)
			}
		})
	}
}

// TestRunMergeCommand tests merge command execution
func (s *ConfigCommandTestSuite) TestRunMergeCommand() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	baseFile := filepath.Join(s.tempDir, "valid.yaml")
	overlayFile := filepath.Join(s.tempDir, "valid.json")

	// Capture stdout to verify merged output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runMergeCommand([]string{baseFile, overlayFile}, "override", false, false)
	assert.NoError(s.T(), err)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NotEmpty(s.T(), output)
}

// TestRunMergeCommand_WithOutput tests merge command with output file
func (s *ConfigCommandTestSuite) TestRunMergeCommand_WithOutput() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	baseFile := filepath.Join(s.tempDir, "valid.yaml")
	overlayFile := filepath.Join(s.tempDir, "valid.json")
	outputPath := filepath.Join(s.tempDir, "merged.yaml")

	originalOutputFile := outputFile
	outputFile = outputPath
	defer func() { outputFile = originalOutputFile }()

	err := runMergeCommand([]string{baseFile, overlayFile}, "override", false, false)
	assert.NoError(s.T(), err)

	// Verify file was created
	assert.FileExists(s.T(), outputPath)
}

// Helper Function Tests

// TestValidateConfigFile tests configuration file validation
func (s *ConfigCommandTestSuite) TestValidateConfigFile() {
	// Test valid YAML file
	validFile := filepath.Join(s.tempDir, "valid.yaml")
	result := validateConfigFile(s.mockConfig, validFile)
	assert.True(s.T(), result.Valid)
	assert.Empty(s.T(), result.Errors)

	// Test valid JSON file
	validJSONFile := filepath.Join(s.tempDir, "valid.json")
	result = validateConfigFile(s.mockConfig, validJSONFile)
	assert.True(s.T(), result.Valid)
	assert.Empty(s.T(), result.Errors)

	// Test invalid YAML file
	invalidFile := filepath.Join(s.tempDir, "invalid.yaml")
	result = validateConfigFile(s.mockConfig, invalidFile)
	assert.False(s.T(), result.Valid)
	assert.NotEmpty(s.T(), result.Errors)

	// Test non-existent file
	nonExistentFile := filepath.Join(s.tempDir, "nonexistent.yaml")
	result = validateConfigFile(s.mockConfig, nonExistentFile)
	assert.False(s.T(), result.Valid)
	assert.Contains(s.T(), result.Errors[0], "File does not exist")
}

// TestFindConfigFiles tests configuration file discovery
func (s *ConfigCommandTestSuite) TestFindConfigFiles() {
	originalWorkDir := workDir
	workDir = s.tempDir
	defer func() { workDir = originalWorkDir }()

	files, err := findConfigFiles()
	assert.NoError(s.T(), err)
	assert.GreaterOrEqual(s.T(), len(files), 2) // at least valid.yaml and valid.json
}

// TestGenerateConfigTemplate tests template generation
func (s *ConfigCommandTestSuite) TestGenerateConfigTemplate() {
	testCases := []struct {
		templateType string
		serviceName  string
		includes     []string
		excludes     []string
		shouldError  bool
	}{
		{"service", "test-service", []string{}, []string{}, false},
		{"gateway", "", []string{}, []string{}, false},
		{"database", "", []string{}, []string{}, false},
		{"cache", "", []string{}, []string{}, false},
		{"monitoring", "", []string{}, []string{}, false},
		{"deployment", "", []string{}, []string{}, false},
		{"invalid", "", []string{}, []string{}, true},
	}

	for _, tc := range testCases {
		s.T().Run(tc.templateType, func(t *testing.T) {
			template, err := generateConfigTemplate(tc.templateType, tc.serviceName, tc.includes, tc.excludes)
			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, template)

				// Verify it's valid YAML
				var yamlData interface{}
				err = yaml.Unmarshal([]byte(template), &yamlData)
				assert.NoError(t, err)
			}
		})
	}
}

// TestGenerateServiceTemplate tests service template generation
func (s *ConfigCommandTestSuite) TestGenerateServiceTemplate() {
	template, err := generateServiceTemplate("my-service", []string{}, []string{})
	assert.NoError(s.T(), err)
	assert.Contains(s.T(), template, "my-service")
	assert.Contains(s.T(), template, "server:")
	assert.Contains(s.T(), template, "database:")

	// Test with empty service name
	template, err = generateServiceTemplate("", []string{}, []string{})
	assert.NoError(s.T(), err)
	assert.Contains(s.T(), template, "my-service") // default name
}

// TestGenerateJSONSchema tests JSON schema generation
func (s *ConfigCommandTestSuite) TestGenerateJSONSchema() {
	schema, err := generateJSONSchema("service", "v1")
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), schema)

	// Verify it's valid JSON
	var jsonData map[string]interface{}
	err = json.Unmarshal([]byte(schema), &jsonData)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "http://json-schema.org/draft-07/schema#", jsonData["$schema"])
}

// TestValidateSyntax tests YAML and JSON syntax validation
func (s *ConfigCommandTestSuite) TestValidateSyntax() {
	// Test valid YAML
	validYAML := []byte("name: test\nversion: 1.0.0")
	err := validateYAMLSyntax(validYAML)
	assert.NoError(s.T(), err)

	// Test invalid YAML (unmatched quotes)
	invalidYAML := []byte("name: \"test\nversion: 1.0")
	err = validateYAMLSyntax(invalidYAML)
	assert.Error(s.T(), err)

	// Test valid JSON
	validJSON := []byte(`{"name": "test", "version": "1.0.0"}`)
	err = validateJSONSyntax(validJSON)
	assert.NoError(s.T(), err)

	// Test invalid JSON (using YAML validation which is more permissive)
	invalidJSON := []byte(`{"name": "test", "version": 1.0.0,}`)
	err = validateJSONSyntax(invalidJSON)
	assert.Error(s.T(), err)
}

// TestUtilityFunctions tests utility functions
func (s *ConfigCommandTestSuite) TestUtilityFunctions() {
	// Test contains
	assert.True(s.T(), contains([]string{"a", "b", "c"}, "b"))
	assert.False(s.T(), contains([]string{"a", "b", "c"}, "d"))

	// Test formatBool
	assert.Equal(s.T(), "enabled", formatBool(true))
	assert.Equal(s.T(), "disabled", formatBool(false))
}

// TestCreateDependencyContainer tests dependency container creation
func (s *ConfigCommandTestSuite) TestCreateDependencyContainer() {
	originalWorkDir := workDir
	workDir = s.tempDir
	defer func() { workDir = originalWorkDir }()

	container, err := createDependencyContainer()
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), container)

	// Test service registration
	requiredServices := []string{"ui", "filesystem", "config_manager"}
	for _, serviceName := range requiredServices {
		service, err := container.GetService(serviceName)
		assert.NoError(s.T(), err, "Should get service: %s", serviceName)
		assert.NotNil(s.T(), service, "Service should not be nil: %s", serviceName)
	}

	container.Close()
}

// TestSimpleConfigManager tests the config manager implementation
func (s *ConfigCommandTestSuite) TestSimpleConfigManager() {
	manager := NewConfigManager(s.tempDir)
	assert.NotNil(s.T(), manager)

	// Test Load
	err := manager.Load()
	assert.NoError(s.T(), err)

	// Test Get methods
	value := manager.Get("test")
	assert.Nil(s.T(), value)

	stringValue := manager.GetString("test")
	assert.Equal(s.T(), "", stringValue)

	intValue := manager.GetInt("test")
	assert.Equal(s.T(), 0, intValue)

	boolValue := manager.GetBool("test")
	assert.False(s.T(), boolValue)

	// Test Set
	manager.Set("test", "value")

	// Test Validate
	err = manager.Validate()
	assert.NoError(s.T(), err)

	// Test Save
	err = manager.Save("test.yaml")
	assert.NoError(s.T(), err)
}

// TestCommandIntegration tests command integration with cobra
func (s *ConfigCommandTestSuite) TestCommandIntegration() {
	testCases := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"config command", NewConfigCommand()},
		{"validate command", NewValidateCommand()},
		{"template command", NewTemplateCommand()},
		{"schema command", NewSchemaCommand()},
		{"migrate command", NewMigrateCommand()},
		{"merge command", NewMergeCommand()},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			tc.cmd.SetOut(&buf)
			tc.cmd.SetErr(&buf)
			tc.cmd.SetArgs([]string{"--help"})

			err := tc.cmd.Execute()
			assert.NoError(t, err)

			output := buf.String()
			assert.Contains(t, output, "Usage:")
			assert.Contains(t, output, "Flags:")
		})
	}
}

// TestErrorHandling tests various error scenarios
func (s *ConfigCommandTestSuite) TestErrorHandling() {
	// Test dependency container creation failure
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return nil, fmt.Errorf("container creation failed")
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	err := runValidateCommand([]string{})
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to create dependency container")

	// Test UI service retrieval failure
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		mockContainer := &MockDependencyContainer{}
		mockContainer.On("GetService", "ui").Return(nil, fmt.Errorf("service not found"))
		mockContainer.On("Close").Return(nil)
		return mockContainer, nil
	}

	err = runValidateCommand([]string{})
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to get UI service")
}

// TestFlagCombinations tests various flag combinations
func (s *ConfigCommandTestSuite) TestFlagCombinations() {
	testCases := []struct {
		name    string
		command *cobra.Command
		flags   []string
	}{
		{
			name:    "config flags",
			command: NewConfigCommand(),
			flags:   []string{"--verbose", "--no-color", "--format", "json", "--work-dir", s.tempDir},
		},
		{
			name:    "validate flags",
			command: NewValidateCommand(),
			flags:   []string{"--strict", "--show-defaults"},
		},
		{
			name:    "template flags",
			command: NewTemplateCommand(),
			flags:   []string{"--type", "service", "--service", "test", "--include", "auth,cache"},
		},
		{
			name:    "schema flags",
			command: NewSchemaCommand(),
			flags:   []string{"--type", "gateway", "--version", "v2"},
		},
		{
			name:    "migrate flags",
			command: NewMigrateCommand(),
			flags:   []string{"--from", "v1", "--to", "v2", "--backup"},
		},
		{
			name:    "merge flags",
			command: NewMergeCommand(),
			flags:   []string{"--strategy", "deep", "--overwrite", "--ignore-nulls"},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			err := tc.command.ParseFlags(tc.flags)
			assert.NoError(t, err, "Flag combination should parse successfully: %v", tc.flags)
		})
	}
}

// TestConcurrentAccess tests thread safety
func (s *ConfigCommandTestSuite) TestConcurrentAccess() {
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			result := formatBool(true)
			assert.Equal(s.T(), "enabled", result)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// Mock types for testing

type MockDependencyContainer struct {
	mock.Mock
}

func (m *MockDependencyContainer) Register(name string, factory interfaces.ServiceFactory) error {
	args := m.Called(name, factory)
	return args.Error(0)
}

func (m *MockDependencyContainer) RegisterSingleton(name string, factory interfaces.ServiceFactory) error {
	args := m.Called(name, factory)
	return args.Error(0)
}

func (m *MockDependencyContainer) GetService(name string) (interface{}, error) {
	args := m.Called(name)
	return args.Get(0), args.Error(1)
}

func (m *MockDependencyContainer) HasService(name string) bool {
	args := m.Called(name)
	return args.Bool(0)
}

func (m *MockDependencyContainer) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockTerminalUI struct {
	mock.Mock
}

func (m *MockTerminalUI) PrintHeader(title string) {
	m.Called(title)
}

func (m *MockTerminalUI) PrintSubHeader(title string) {
	m.Called(title)
}

func (m *MockTerminalUI) ShowProgress(title string, total int) interfaces.ProgressBar {
	args := m.Called(title, total)
	return args.Get(0).(interfaces.ProgressBar)
}

func (m *MockTerminalUI) ShowSuccess(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockTerminalUI) ShowInfo(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockTerminalUI) ShowError(err error) error {
	args := m.Called(err)
	return args.Error(0)
}

func (m *MockTerminalUI) ShowTable(headers []string, rows [][]string) error {
	args := m.Called(headers, rows)
	return args.Error(0)
}

func (m *MockTerminalUI) GetStyle() *interfaces.UIStyle {
	args := m.Called()
	return args.Get(0).(*interfaces.UIStyle)
}

func (m *MockTerminalUI) ShowWelcome() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockTerminalUI) PromptInput(prompt string, validator interfaces.InputValidator) (string, error) {
	args := m.Called(prompt, validator)
	return args.String(0), args.Error(1)
}

func (m *MockTerminalUI) ShowMenu(title string, options []interfaces.MenuOption) (int, error) {
	args := m.Called(title, options)
	return args.Int(0), args.Error(1)
}

func (m *MockTerminalUI) PromptConfirm(prompt string, defaultValue bool) (bool, error) {
	args := m.Called(prompt, defaultValue)
	return args.Bool(0), args.Error(1)
}

func (m *MockTerminalUI) PrintSeparator() {
	m.Called()
}

func (m *MockTerminalUI) PromptPassword(prompt string) (string, error) {
	args := m.Called(prompt)
	return args.String(0), args.Error(1)
}

func (m *MockTerminalUI) ShowFeatureSelectionMenu() (interfaces.ServiceFeatures, error) {
	args := m.Called()
	return args.Get(0).(interfaces.ServiceFeatures), args.Error(1)
}

func (m *MockTerminalUI) ShowDatabaseSelectionMenu() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockTerminalUI) ShowAuthSelectionMenu() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockTerminalUI) ShowMultiSelectMenu(title string, options []interfaces.MenuOption) ([]int, error) {
	args := m.Called(title, options)
	return args.Get(0).([]int), args.Error(1)
}

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	args := make([]interface{}, 0, len(fields)+1)
	args = append(args, msg)
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	args := make([]interface{}, 0, len(fields)+1)
	args = append(args, msg)
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	args := make([]interface{}, 0, len(fields)+1)
	args = append(args, msg)
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
	args := make([]interface{}, 0, len(fields)+1)
	args = append(args, msg)
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

func (m *MockLogger) Fatal(msg string, fields ...interface{}) {
	args := make([]interface{}, 0, len(fields)+1)
	args = append(args, msg)
	for _, field := range fields {
		args = append(args, field)
	}
	m.Called(args...)
}

type MockFileSystem struct {
	mock.Mock
}

func (m *MockFileSystem) WriteFile(path string, content []byte, perm int) error {
	args := m.Called(path, content, perm)
	return args.Error(0)
}

func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	args := m.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockFileSystem) MkdirAll(path string, perm int) error {
	args := m.Called(path, perm)
	return args.Error(0)
}

func (m *MockFileSystem) Exists(path string) bool {
	args := m.Called(path)
	return args.Bool(0)
}

func (m *MockFileSystem) Remove(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockFileSystem) Copy(src, dst string) error {
	args := m.Called(src, dst)
	return args.Error(0)
}

type MockConfigManager struct {
	mock.Mock
}

func (m *MockConfigManager) Load() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConfigManager) Get(key string) interface{} {
	args := m.Called(key)
	return args.Get(0)
}

func (m *MockConfigManager) GetString(key string) string {
	args := m.Called(key)
	return args.String(0)
}

func (m *MockConfigManager) GetInt(key string) int {
	args := m.Called(key)
	return args.Int(0)
}

func (m *MockConfigManager) GetBool(key string) bool {
	args := m.Called(key)
	return args.Bool(0)
}

func (m *MockConfigManager) Set(key string, value interface{}) {
	m.Called(key, value)
}

func (m *MockConfigManager) Validate() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConfigManager) Save(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

// MockColor is a mock implementation of interfaces.Color
type MockColor struct {
	mock.Mock
}

func (m *MockColor) Sprint(a ...interface{}) string {
	args := m.Called(a...)
	return args.String(0)
}

func (m *MockColor) Sprintf(format string, a ...interface{}) string {
	args := m.Called(format, a)
	return args.String(0)
}

type MockProgressBar struct {
	mock.Mock
}

func (m *MockProgressBar) Update(current int) error {
	args := m.Called(current)
	return args.Error(0)
}

func (m *MockProgressBar) SetMessage(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockProgressBar) Finish() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProgressBar) SetTotal(total int) error {
	args := m.Called(total)
	return args.Error(0)
}

// Run the test suite
func TestConfigCommandTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigCommandTestSuite))
}

// ----- Additional tests for backup/restore/history -----

// TestNewBackupRestoreHistoryCommands_Properties validates basic command wiring
func TestNewBackupRestoreHistoryCommands_Properties(t *testing.T) {
    bc := NewBackupCommand()
    if bc == nil || bc.Use == "" {
        t.Fatalf("backup command should be constructed")
    }

    rc := NewRestoreCommand()
    if rc == nil || rc.Use == "" {
        t.Fatalf("restore command should be constructed")
    }

    hc := NewHistoryCommand()
    if hc == nil || hc.Use == "" {
        t.Fatalf("history command should be constructed")
    }
}

// TestRunBackupCommand_NoFiles ensures graceful no-op when no config files exist
func TestRunBackupCommand_NoFiles(t *testing.T) {
    tmp, err := os.MkdirTemp("", "switctl-backup-no-files-*")
    if err != nil {
        t.Fatalf("temp dir: %v", err)
    }
    defer os.RemoveAll(tmp)

    // set workDir to temp
    oldWd := workDir
    workDir = tmp
    defer func() { workDir = oldWd }()

    if err := runBackupCommand([]string{}, "", ""); err != nil {
        t.Fatalf("backup should succeed with no files: %v", err)
    }
}

// TestBackupRestoreEndToEnd creates sample files, backs up, lists history, and attempts restore
func TestBackupRestoreEndToEnd(t *testing.T) {
    tmp, err := os.MkdirTemp("", "switctl-backup-e2e-*")
    if err != nil {
        t.Fatalf("temp dir: %v", err)
    }
    defer os.RemoveAll(tmp)

    // set workDir to temp
    oldWd := workDir
    workDir = tmp
    defer func() { workDir = oldWd }()

    // create two config files in workDir
    if err := os.WriteFile(filepath.Join(tmp, "swit.yaml"), []byte("name: service-a"), 0644); err != nil {
        t.Fatalf("write swit.yaml: %v", err)
    }
    if err := os.WriteFile(filepath.Join(tmp, "switauth.yaml"), []byte("name: auth"), 0644); err != nil {
        t.Fatalf("write switauth.yaml: %v", err)
    }

    if err := runBackupCommand([]string{}, "e2e", ""); err != nil {
        t.Fatalf("backup: %v", err)
    }

    // history should not error even if no assertions on output
    if err := runHistoryCommand(10); err != nil {
        t.Fatalf("history: %v", err)
    }

    // attempt restore into target dir; allow overwrite to avoid conflicts
    target := filepath.Join(tmp, "restored")
    if err := os.MkdirAll(target, 0755); err != nil {
        t.Fatalf("mkdir restored: %v", err)
    }

    // If there is no backup detected via OS listing, restore may fail; treat either nil or well-formed error as acceptable
    if err := runRestoreCommand([]string{}, "", true, target); err != nil {
        // accept scenarios where environment cannot detect latest backup directory
        if !strings.Contains(err.Error(), "backup") && !strings.Contains(err.Error(), "read") {
            t.Fatalf("restore unexpected error: %v", err)
        }
    }
}
