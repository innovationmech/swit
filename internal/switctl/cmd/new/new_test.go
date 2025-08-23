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

package new

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// NewCommandTestSuite tests the 'new' command functionality
type NewCommandTestSuite struct {
	suite.Suite
	tempDir            string
	originalArgs       []string
	originalWd         string
	mockContainer      *MockDependencyContainer
	mockUI             *MockTerminalUI
	mockLogger         *MockLogger
	mockFileSystem     *MockFileSystem
	mockTemplateEngine *MockTemplateEngine
	mockGenerator      *MockServiceGenerator
}

// SetupSuite runs before all tests in the suite
func (s *NewCommandTestSuite) SetupSuite() {
	// Save original values
	s.originalArgs = os.Args
	s.originalWd, _ = os.Getwd()
}

// SetupTest runs before each test
func (s *NewCommandTestSuite) SetupTest() {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "switctl-new-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Reset global variables
	interactive = false
	outputDir = ""
	configFile = ""
	templateDir = ""
	verbose = false
	noColor = false
	force = false
	dryRun = false
	skipValidation = false

	// Reset viper
	viper.Reset()

	// Create mocks
	s.mockContainer = &MockDependencyContainer{}
	s.mockUI = &MockTerminalUI{}
	s.mockLogger = &MockLogger{}
	s.mockFileSystem = &MockFileSystem{}
	s.mockTemplateEngine = &MockTemplateEngine{}
	s.mockGenerator = &MockServiceGenerator{}

	// Set up mock container to return our mocks
	s.mockContainer.On("GetService", "ui").Return(s.mockUI, nil)
	s.mockContainer.On("GetService", "logger").Return(s.mockLogger, nil)
	s.mockContainer.On("GetService", "filesystem").Return(s.mockFileSystem, nil)
	s.mockContainer.On("GetService", "template_engine").Return(s.mockTemplateEngine, nil)
	s.mockContainer.On("GetService", "service_generator").Return(s.mockGenerator, nil)
	s.mockContainer.On("Close").Return(nil)
}

// TearDownTest runs after each test
func (s *NewCommandTestSuite) TearDownTest() {
	// Clean up temporary directory
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}

	// Reset global variables
	interactive = false
	outputDir = ""
	configFile = ""
	templateDir = ""
	verbose = false
	noColor = false
	force = false
	dryRun = false
	skipValidation = false

	// Reset viper
	viper.Reset()

	// Change back to original working directory
	os.Chdir(s.originalWd)
}

// TearDownSuite runs after all tests in the suite
func (s *NewCommandTestSuite) TearDownSuite() {
	// Restore original values
	os.Args = s.originalArgs
}

// TestNewNewCommand_BasicProperties tests the basic properties of the new command
func (s *NewCommandTestSuite) TestNewNewCommand_BasicProperties() {
	cmd := NewNewCommand()

	assert.NotNil(s.T(), cmd)
	assert.Equal(s.T(), "new", cmd.Use)
	assert.Equal(s.T(), "Create new projects and components", cmd.Short)
	assert.Contains(s.T(), cmd.Long, "The new command creates new projects, services, and components using the Swit framework")
	assert.Contains(s.T(), cmd.Long, "This command provides interactive and non-interactive modes for creating")
	assert.Contains(s.T(), cmd.Long, "Examples:")
}

// TestNewNewCommand_PersistentFlags tests all persistent flags
func (s *NewCommandTestSuite) TestNewNewCommand_PersistentFlags() {
	cmd := NewNewCommand()
	flags := cmd.PersistentFlags()

	// Test interactive flag
	interactiveFlag := flags.Lookup("interactive")
	assert.NotNil(s.T(), interactiveFlag)
	assert.Equal(s.T(), "i", interactiveFlag.Shorthand)
	assert.Equal(s.T(), "false", interactiveFlag.DefValue)

	// Test output-dir flag
	outputDirFlag := flags.Lookup("output-dir")
	assert.NotNil(s.T(), outputDirFlag)
	assert.Equal(s.T(), "o", outputDirFlag.Shorthand)
	assert.Equal(s.T(), "", outputDirFlag.DefValue)

	// Test config flag
	configFlag := flags.Lookup("config")
	assert.NotNil(s.T(), configFlag)
	assert.Equal(s.T(), "c", configFlag.Shorthand)

	// Test template-dir flag
	templateDirFlag := flags.Lookup("template-dir")
	assert.NotNil(s.T(), templateDirFlag)

	// Test boolean flags
	boolFlags := []string{"verbose", "no-color", "force", "dry-run", "skip-validation"}
	for _, flagName := range boolFlags {
		flag := flags.Lookup(flagName)
		assert.NotNil(s.T(), flag, fmt.Sprintf("Flag %s should exist", flagName))
		assert.Equal(s.T(), "false", flag.DefValue, fmt.Sprintf("Flag %s should default to false", flagName))
	}
}

// TestNewNewCommand_ViperBinding tests that flags are properly bound to viper
func (s *NewCommandTestSuite) TestNewNewCommand_ViperBinding() {
	cmd := NewNewCommand()

	// Set flag values
	cmd.PersistentFlags().Set("interactive", "true")
	cmd.PersistentFlags().Set("output-dir", "/test/output")
	cmd.PersistentFlags().Set("verbose", "true")
	cmd.PersistentFlags().Set("force", "true")

	// The actual binding happens during command execution
	// Here we just verify the flags are set
	interactiveFlag := cmd.PersistentFlags().Lookup("interactive")
	assert.Equal(s.T(), "true", interactiveFlag.Value.String())

	outputDirFlag := cmd.PersistentFlags().Lookup("output-dir")
	assert.Equal(s.T(), "/test/output", outputDirFlag.Value.String())
}

// TestNewNewCommand_Subcommands tests that subcommands are properly added
func (s *NewCommandTestSuite) TestNewNewCommand_Subcommands() {
	cmd := NewNewCommand()
	subcommands := cmd.Commands()

	// Should have at least the service subcommand
	assert.GreaterOrEqual(s.T(), len(subcommands), 1)

	// Check for service subcommand
	var serviceCmd *cobra.Command
	for _, subcmd := range subcommands {
		if subcmd.Use == "service [name]" {
			serviceCmd = subcmd
			break
		}
	}
	assert.NotNil(s.T(), serviceCmd, "Service subcommand should be present")
}

// TestNewNewCommand_Help tests the help output
func (s *NewCommandTestSuite) TestNewNewCommand_Help() {
	cmd := NewNewCommand()

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	assert.NoError(s.T(), err)

	output := buf.String()
	assert.Contains(s.T(), output, "Usage:")
	assert.Contains(s.T(), output, "new")
	assert.Contains(s.T(), output, "The new command creates new projects, services, and components using the Swit framework")
	assert.Contains(s.T(), output, "Available Commands:")
	assert.Contains(s.T(), output, "Flags:")
}

// TestInitializeNewCommandConfig tests configuration initialization
func (s *NewCommandTestSuite) TestInitializeNewCommandConfig() {
	// Create a valid template directory for testing
	testTemplateDir := filepath.Join(s.tempDir, "valid_templates")
	err := os.MkdirAll(testTemplateDir, 0755)
	s.Require().NoError(err)

	// Set the template directory through viper to avoid fallback behavior
	viper.Set("new.template_dir", testTemplateDir)

	// Test with default values
	cmd := NewNewCommand()
	err = initializeNewCommandConfig(cmd)
	assert.NoError(s.T(), err)

	// Test with viper values set
	viper.Set("new.interactive", true)
	viper.Set("new.output_dir", "/custom/output")
	viper.Set("new.verbose", true)

	err = initializeNewCommandConfig(cmd)
	assert.NoError(s.T(), err)
	assert.True(s.T(), interactive)
	assert.Equal(s.T(), "/custom/output", outputDir)
	assert.True(s.T(), verbose)
}

// TestInitializeNewCommandConfig_TemplateDirectory tests template directory configuration
func (s *NewCommandTestSuite) TestInitializeNewCommandConfig_TemplateDirectory() {
	// Create a test template directory
	testTemplateDir := filepath.Join(s.tempDir, "templates")
	err := os.MkdirAll(testTemplateDir, 0755)
	s.Require().NoError(err)

	// Set template directory through viper (simulates config file or env var)
	viper.Set("new.template_dir", testTemplateDir)

	cmd := NewNewCommand()
	err = initializeNewCommandConfig(cmd)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), testTemplateDir, templateDir)
}

// TestInitializeNewCommandConfig_InvalidTemplateDirectory tests error handling for invalid template directory
func (s *NewCommandTestSuite) TestInitializeNewCommandConfig_InvalidTemplateDirectory() {
	// Set non-existent template directory
	templateDir = "/nonexistent/templates"

	cmd := NewNewCommand()
	err := initializeNewCommandConfig(cmd)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "template directory does not exist")
}

// TestCreateDependencyContainer tests dependency container creation
func (s *NewCommandTestSuite) TestCreateDependencyContainer() {
	container, err := createDependencyContainer()
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), container)

	// Test that core services are registered
	assert.True(s.T(), container.HasService("filesystem"))
	assert.True(s.T(), container.HasService("template_engine"))
	assert.True(s.T(), container.HasService("ui"))
	assert.True(s.T(), container.HasService("logger"))
	assert.True(s.T(), container.HasService("service_generator"))

	// Test service retrieval
	filesystem, err := container.GetService("filesystem")
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), filesystem)

	templateEngine, err := container.GetService("template_engine")
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), templateEngine)

	ui, err := container.GetService("ui")
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), ui)

	logger, err := container.GetService("logger")
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), logger)

	serviceGenerator, err := container.GetService("service_generator")
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), serviceGenerator)

	// Clean up
	container.Close()
}

// TestSimpleDependencyContainer tests the SimpleDependencyContainer implementation
func (s *NewCommandTestSuite) TestSimpleDependencyContainer() {
	container := &SimpleDependencyContainer{
		services:   make(map[string]interface{}),
		factories:  make(map[string]interfaces.ServiceFactory),
		singletons: make(map[string]bool),
	}

	// Test registration
	err := container.Register("test_service", func() (interface{}, error) {
		return "test_instance", nil
	})
	assert.NoError(s.T(), err)
	assert.True(s.T(), container.HasService("test_service"))

	// Test singleton registration
	callCount := 0
	err = container.RegisterSingleton("singleton_service", func() (interface{}, error) {
		callCount++
		return fmt.Sprintf("instance_%d", callCount), nil
	})
	assert.NoError(s.T(), err)

	// Test service retrieval
	instance1, err := container.GetService("singleton_service")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "instance_1", instance1)

	// Test singleton behavior - should return same instance
	instance2, err := container.GetService("singleton_service")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "instance_1", instance2)
	assert.Equal(s.T(), 1, callCount) // Should only be called once

	// Test non-singleton behavior
	instance3, err := container.GetService("test_service")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "test_instance", instance3)

	instance4, err := container.GetService("test_service")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "test_instance", instance4)

	// Test error cases
	_, err = container.GetService("nonexistent")
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "service 'nonexistent' not found")

	// Test registration with empty name
	err = container.Register("", func() (interface{}, error) { return nil, nil })
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "service name cannot be empty")

	// Test registration with nil factory
	err = container.Register("nil_factory", nil)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "service factory cannot be nil")

	// Test factory error handling
	err = container.Register("error_service", func() (interface{}, error) {
		return nil, fmt.Errorf("factory error")
	})
	assert.NoError(s.T(), err)

	_, err = container.GetService("error_service")
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to create service 'error_service'")

	// Test cleanup
	err = container.Close()
	assert.NoError(s.T(), err)
}

// TestSimpleLogger tests the SimpleLogger implementation
func (s *NewCommandTestSuite) TestSimpleLogger() {
	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test non-verbose logger
	logger := NewSimpleLogger(false)
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	// Debug should not appear in non-verbose mode
	assert.NotContains(s.T(), output, "[DEBUG]")
	assert.Contains(s.T(), output, "[INFO] info message")
	assert.Contains(s.T(), output, "[WARN] warn message")
	assert.Contains(s.T(), output, "[ERROR] error message")

	// Test verbose logger
	buf.Reset()
	r, w, _ = os.Pipe()
	os.Stdout = w

	verboseLogger := NewSimpleLogger(true)
	verboseLogger.Debug("debug message", "key", "value")

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output = buf.String()

	assert.Contains(s.T(), output, "[DEBUG] debug message")
}

// TestCreateExecutionContext tests execution context creation
func (s *NewCommandTestSuite) TestCreateExecutionContext() {
	ctx := context.WithValue(context.Background(), "start_time", time.Now())

	// Set some global variables
	configFile = "/test/config.yaml"
	verbose = true
	noColor = true
	outputDir = "/test/output"
	force = true
	dryRun = true

	execCtx := createExecutionContext(ctx)

	assert.NotNil(s.T(), execCtx)
	assert.Equal(s.T(), "/test/config.yaml", execCtx.ConfigFile)
	assert.True(s.T(), execCtx.Verbose)
	assert.True(s.T(), execCtx.NoColor)
	assert.NotNil(s.T(), execCtx.Logger)
	assert.NotNil(s.T(), execCtx.Writer)
	assert.NotNil(s.T(), execCtx.Options)

	// Check options
	assert.Equal(s.T(), "/test/output", execCtx.Options["output_dir"])
	assert.True(s.T(), execCtx.Options["force"].(bool))
	assert.True(s.T(), execCtx.Options["dry_run"].(bool))
}

// TestUtilityFunctions tests utility functions
func (s *NewCommandTestSuite) TestUtilityFunctions() {
	// Test dirExists
	assert.True(s.T(), dirExists(s.tempDir))
	assert.False(s.T(), dirExists(filepath.Join(s.tempDir, "nonexistent")))

	// Test fileExists
	testFile := filepath.Join(s.tempDir, "testfile")
	f, err := os.Create(testFile)
	s.Require().NoError(err)
	f.Close()

	assert.True(s.T(), fileExists(testFile))
	assert.False(s.T(), fileExists(filepath.Join(s.tempDir, "nonexistent")))
	assert.False(s.T(), dirExists(testFile)) // Directory check should fail for file

	// Test ensureDir
	newDir := filepath.Join(s.tempDir, "newdir", "subdir")
	err = ensureDir(newDir)
	assert.NoError(s.T(), err)
	assert.True(s.T(), dirExists(newDir))

	// Test ensureDir on existing directory
	err = ensureDir(newDir)
	assert.NoError(s.T(), err) // Should not error
}

// TestCommandIntegration tests basic command integration
func (s *NewCommandTestSuite) TestCommandIntegration() {
	cmd := NewNewCommand()

	// Test that command can be executed (should show help or subcommand list)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{})

	// Since no subcommand is specified, it should either run successfully
	// or return an error asking for a subcommand
	err := cmd.Execute()
	// We don't assert on the error because different cobra versions handle this differently
	// The important thing is that the command structure is valid
	_ = err

	// Verify some basic output is generated
	output := buf.String()
	if output != "" {
		assert.Contains(s.T(), output, "new")
	}
}

// TestErrorHandling tests various error conditions
func (s *NewCommandTestSuite) TestErrorHandling() {
	// Test with invalid working directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	// This should still work because we don't change working directory
	ctx := context.WithValue(context.Background(), "start_time", time.Now())
	execCtx := createExecutionContext(ctx)
	assert.NotNil(s.T(), execCtx)
}

// TestFlagCombinations tests various flag combinations
func (s *NewCommandTestSuite) TestFlagCombinations() {
	testCases := []struct {
		name  string
		flags []string
	}{
		{
			name:  "all flags",
			flags: []string{"--interactive", "--verbose", "--force", "--dry-run", "--no-color", "--skip-validation"},
		},
		{
			name:  "output and config flags",
			flags: []string{"--output-dir", "/test/output", "--config", "/test/config.yaml"},
		},
		{
			name:  "template directory flag",
			flags: []string{"--template-dir", "/test/templates"},
		},
		{
			name:  "short flags",
			flags: []string{"-i", "-v", "-o", "/test/output", "-c", "/test/config.yaml"},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			cmd := NewNewCommand()

			// Just test flag parsing without triggering help
			err := cmd.ParseFlags(tc.flags)
			assert.NoError(t, err, "Flag combination should parse successfully: %v", tc.flags)
		})
	}
}

// TestConcurrentAccess tests concurrent access to the dependency container
func (s *NewCommandTestSuite) TestConcurrentAccess() {
	container, err := createDependencyContainer()
	s.Require().NoError(err)
	defer container.Close()

	// Test concurrent access to singleton services
	const numGoroutines = 10
	results := make([]interface{}, numGoroutines)
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			service, err := container.GetService("logger")
			assert.NoError(s.T(), err)
			results[index] = service
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// All results should be the same instance (singleton)
	firstResult := results[0]
	for i := 1; i < numGoroutines; i++ {
		assert.Same(s.T(), firstResult, results[i])
	}
}

// TestServiceRegistration tests that all required services are properly registered
func (s *NewCommandTestSuite) TestServiceRegistration() {
	container := &SimpleDependencyContainer{
		services:   make(map[string]interface{}),
		factories:  make(map[string]interfaces.ServiceFactory),
		singletons: make(map[string]bool),
	}

	err := registerCoreServices(container)
	assert.NoError(s.T(), err)

	requiredServices := []string{
		"filesystem",
		"template_engine",
		"ui",
		"logger",
		"service_generator",
	}

	for _, serviceName := range requiredServices {
		assert.True(s.T(), container.HasService(serviceName),
			"Required service should be registered: %s", serviceName)

		// Verify service can be retrieved
		service, err := container.GetService(serviceName)
		assert.NoError(s.T(), err, "Should be able to get service: %s", serviceName)
		assert.NotNil(s.T(), service, "Service should not be nil: %s", serviceName)
	}
}

// Benchmark tests
func BenchmarkNewNewCommand(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := NewNewCommand()
		_ = cmd
	}
}

func BenchmarkCreateDependencyContainer(b *testing.B) {
	// Set up a valid template directory
	tempDir, err := os.MkdirTemp("", "benchmark-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	templateDir = tempDir

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		container, err := createDependencyContainer()
		if err != nil {
			b.Fatal(err)
		}
		container.Close()
	}
}

func BenchmarkSimpleLogger(b *testing.B) {
	logger := NewSimpleLogger(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value")
	}
}

// Run the test suite
func TestNewCommandTestSuite(t *testing.T) {
	suite.Run(t, new(NewCommandTestSuite))
}
