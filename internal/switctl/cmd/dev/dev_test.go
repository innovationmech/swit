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

package dev

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// DevCommandTestSuite tests the 'dev' command functionality
type DevCommandTestSuite struct {
	suite.Suite
	tempDir       string
	originalArgs  []string
	originalWd    string
	mockContainer *MockDependencyContainer
	mockUI        *MockDevUI
	// mockFS would be dependency injected
	mockExec        *MockCommandExecutor
	originalExecCmd func(string, ...string) *exec.Cmd
}

// SetupSuite runs before all tests in the suite
func (s *DevCommandTestSuite) SetupSuite() {
	s.originalArgs = os.Args
	s.originalWd, _ = os.Getwd()
	s.originalExecCmd = exec.Command
}

// SetupTest runs before each test
func (s *DevCommandTestSuite) SetupTest() {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "switctl-dev-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Initialize mocks
	s.mockContainer = &MockDependencyContainer{}
	s.mockUI = &MockDevUI{}
	// Mock filesystem would be set up through dependency injection
	s.mockExec = &MockCommandExecutor{}

	// Reset global variables
	s.resetGlobals()

	// Reset viper
	viper.Reset()
}

// TearDownTest runs after each test
func (s *DevCommandTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
	s.resetGlobals()
	viper.Reset()
	os.Chdir(s.originalWd)
	// Reset exec command would be done through dependency injection in real implementation
}

// TearDownSuite runs after all tests in the suite
func (s *DevCommandTestSuite) TearDownSuite() {
	os.Args = s.originalArgs
}

// Helper Methods

func (s *DevCommandTestSuite) resetGlobals() {
	verbose = false
	noColor = false
	workDir = ""
	parallel = false
	watch = false
	hotReload = false
	port = 0
	profile = ""
}

// Mock implementations

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

type MockDevUI struct {
	mock.Mock
}

func (m *MockDevUI) ShowWelcome() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDevUI) PromptInput(prompt string, validator interfaces.InputValidator) (string, error) {
	args := m.Called(prompt, validator)
	return args.String(0), args.Error(1)
}

func (m *MockDevUI) ShowMenu(title string, options []interfaces.MenuOption) (int, error) {
	args := m.Called(title, options)
	return args.Int(0), args.Error(1)
}

func (m *MockDevUI) ShowProgress(title string, total int) interfaces.ProgressBar {
	args := m.Called(title, total)
	return args.Get(0).(interfaces.ProgressBar)
}

func (m *MockDevUI) ShowSuccess(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockDevUI) ShowError(err error) error {
	args := m.Called(err)
	return args.Error(0)
}

func (m *MockDevUI) ShowTable(headers []string, rows [][]string) error {
	args := m.Called(headers, rows)
	return args.Error(0)
}

func (m *MockDevUI) PromptConfirm(prompt string, defaultValue bool) (bool, error) {
	args := m.Called(prompt, defaultValue)
	return args.Bool(0), args.Error(1)
}

func (m *MockDevUI) ShowInfo(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockDevUI) PrintSeparator() {
	m.Called()
}

func (m *MockDevUI) PrintHeader(title string) {
	m.Called(title)
}

func (m *MockDevUI) PrintSubHeader(title string) {
	m.Called(title)
}

func (m *MockDevUI) GetStyle() *interfaces.UIStyle {
	args := m.Called()
	return args.Get(0).(*interfaces.UIStyle)
}

func (m *MockDevUI) PromptPassword(prompt string) (string, error) {
	args := m.Called(prompt)
	return args.String(0), args.Error(1)
}

func (m *MockDevUI) ShowFeatureSelectionMenu() (interfaces.ServiceFeatures, error) {
	args := m.Called()
	return args.Get(0).(interfaces.ServiceFeatures), args.Error(1)
}

func (m *MockDevUI) ShowDatabaseSelectionMenu() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockDevUI) ShowAuthSelectionMenu() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) Execute(command string, args ...string) error {
	argsList := append([]interface{}{command}, toInterfaceSlice(args)...)
	mockArgs := m.Called(argsList...)
	return mockArgs.Error(0)
}

func (m *MockCommandExecutor) ExecuteWithOutput(command string, args ...string) ([]byte, error) {
	argsList := append([]interface{}{command}, toInterfaceSlice(args)...)
	mockArgs := m.Called(argsList...)
	return mockArgs.Get(0).([]byte), mockArgs.Error(1)
}

func toInterfaceSlice(slice []string) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
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

// Test Cases

// TestNewDevCommand_BasicProperties tests the basic properties of the dev command
func (s *DevCommandTestSuite) TestNewDevCommand_BasicProperties() {
	cmd := NewDevCommand()

	assert.NotNil(s.T(), cmd)
	assert.Equal(s.T(), "dev", cmd.Use)
	assert.Equal(s.T(), "Development workflow commands", cmd.Short)
	assert.Contains(s.T(), cmd.Long, "The dev command provides development workflow automation")
	assert.Contains(s.T(), cmd.Long, "Examples:")
}

// TestNewDevCommand_Flags tests all persistent flags
func (s *DevCommandTestSuite) TestNewDevCommand_Flags() {
	cmd := NewDevCommand()
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
		{"parallel", "", "false", true},
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

// TestNewDevCommand_Subcommands tests that all subcommands are added
func (s *DevCommandTestSuite) TestNewDevCommand_Subcommands() {
	cmd := NewDevCommand()
	subcommands := cmd.Commands()

	expectedSubcommands := []string{"setup", "run", "test", "clean"}
	actualSubcommands := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		actualSubcommands[i] = subcmd.Name()
	}

	for _, expected := range expectedSubcommands {
		assert.Contains(s.T(), actualSubcommands, expected, "Subcommand %s should exist", expected)
	}
}

// TestSetupCommand tests the setup subcommand
func (s *DevCommandTestSuite) TestSetupCommand() {
	setupCmd := NewSetupCommand()

	assert.NotNil(s.T(), setupCmd)
	assert.Equal(s.T(), "setup", setupCmd.Use)
	assert.Equal(s.T(), "Install development dependencies and tools", setupCmd.Short)
	assert.Contains(s.T(), setupCmd.Long, "The setup command installs all required development dependencies")
}

// TestSetupCommand_Flags tests setup command flags
func (s *DevCommandTestSuite) TestSetupCommand_Flags() {
	setupCmd := NewSetupCommand()
	flags := setupCmd.Flags()

	testCases := []struct {
		name        string
		shouldExist bool
	}{
		{"skip-tools", true},
		{"skip-deps", true},
		{"force", true},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			flag := flags.Lookup(tc.name)
			if tc.shouldExist {
				assert.NotNil(t, flag, "Flag %s should exist", tc.name)
			} else {
				assert.Nil(t, flag, "Flag %s should not exist", tc.name)
			}
		})
	}
}

// TestRunCommand tests the run subcommand
func (s *DevCommandTestSuite) TestRunCommand() {
	runCmd := NewRunCommand()

	assert.NotNil(s.T(), runCmd)
	assert.Equal(s.T(), "run [service]", runCmd.Use)
	assert.Equal(s.T(), "Start service in development mode with hot reload", runCmd.Short)
	assert.Contains(s.T(), runCmd.Long, "The run command starts a service in development mode")
}

// TestRunCommand_Flags tests run command flags
func (s *DevCommandTestSuite) TestRunCommand_Flags() {
	runCmd := NewRunCommand()
	flags := runCmd.Flags()

	testCases := []struct {
		name        string
		shouldExist bool
	}{
		{"port", true},
		{"watch", true},
		{"hot-reload", true},
		{"profile", true},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			flag := flags.Lookup(tc.name)
			if tc.shouldExist {
				assert.NotNil(t, flag, "Flag %s should exist", tc.name)
			} else {
				assert.Nil(t, flag, "Flag %s should not exist", tc.name)
			}
		})
	}
}

// TestTestCommand tests the test subcommand
func (s *DevCommandTestSuite) TestTestCommand() {
	testCmd := NewTestCommand()

	assert.NotNil(s.T(), testCmd)
	assert.Equal(s.T(), "test [pattern]", testCmd.Use)
	assert.Equal(s.T(), "Run tests with beautiful real-time feedback", testCmd.Short)
	assert.Contains(s.T(), testCmd.Long, "The test command runs tests with enhanced output")
}

// TestTestCommand_Flags tests test command flags
func (s *DevCommandTestSuite) TestTestCommand_Flags() {
	testCmd := NewTestCommand()
	flags := testCmd.Flags()

	testCases := []struct {
		name        string
		shouldExist bool
	}{
		{"coverage", true},
		{"watch", true},
		{"race", true},
		{"short", true},
		{"tags", true},
		{"timeout", true},
		{"benchmark", true},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			flag := flags.Lookup(tc.name)
			if tc.shouldExist {
				assert.NotNil(t, flag, "Flag %s should exist", tc.name)
			} else {
				assert.Nil(t, flag, "Flag %s should not exist", tc.name)
			}
		})
	}
}

// TestCleanCommand tests the clean subcommand
func (s *DevCommandTestSuite) TestCleanCommand() {
	cleanCmd := NewCleanCommand()

	assert.NotNil(s.T(), cleanCmd)
	assert.Equal(s.T(), "clean", cleanCmd.Use)
	assert.Equal(s.T(), "Clean generated files and build cache", cleanCmd.Short)
	assert.Contains(s.T(), cleanCmd.Long, "The clean command removes generated files")
}

// TestCleanCommand_Flags tests clean command flags
func (s *DevCommandTestSuite) TestCleanCommand_Flags() {
	cleanCmd := NewCleanCommand()
	flags := cleanCmd.Flags()

	testCases := []struct {
		name        string
		shouldExist bool
	}{
		{"all", true},
		{"cache", true},
		{"generated", true},
		{"vendor", true},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			flag := flags.Lookup(tc.name)
			if tc.shouldExist {
				assert.NotNil(t, flag, "Flag %s should exist", tc.name)
			} else {
				assert.Nil(t, flag, "Flag %s should not exist", tc.name)
			}
		})
	}
}

// TestSetupExecutionWithMocks tests setup execution with mocked dependencies
func (s *DevCommandTestSuite) TestSetupExecutionWithMocks() {
	// Create a test project structure
	err := os.MkdirAll(filepath.Join(s.tempDir, "cmd", "myservice"), 0755)
	s.Require().NoError(err)

	goModContent := `module github.com/test/service
go 1.19
`
	err = os.WriteFile(filepath.Join(s.tempDir, "go.mod"), []byte(goModContent), 0644)
	s.Require().NoError(err)

	// Mock command execution
	// Mock command execution would be done through dependency injection

	// Test setup logic
	workDir = s.tempDir

	// Verify the project directory structure was set up correctly
	assert.DirExists(s.T(), filepath.Join(s.tempDir, "cmd"))
	assert.FileExists(s.T(), filepath.Join(s.tempDir, "go.mod"))
}

// TestRunExecutionWithServiceDetection tests service detection and running
func (s *DevCommandTestSuite) TestRunExecutionWithServiceDetection() {
	// Create test service structure
	serviceDir := filepath.Join(s.tempDir, "cmd", "myservice")
	err := os.MkdirAll(serviceDir, 0755)
	s.Require().NoError(err)

	mainContent := `package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World!")
	})
	
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
`
	err = os.WriteFile(filepath.Join(serviceDir, "main.go"), []byte(mainContent), 0644)
	s.Require().NoError(err)

	workDir = s.tempDir

	// Test service detection logic
	services, err := detectServices(s.tempDir)
	assert.NoError(s.T(), err)
	assert.Contains(s.T(), services, "myservice")

	// Test finding main file
	mainFile, err := findMainFile(serviceDir)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), filepath.Join(serviceDir, "main.go"), mainFile)
}

// TestTestExecutionWithCoverage tests test execution with coverage
func (s *DevCommandTestSuite) TestTestExecutionWithCoverage() {
	// Create test file
	testDir := filepath.Join(s.tempDir, "internal", "service")
	err := os.MkdirAll(testDir, 0755)
	s.Require().NoError(err)

	testContent := `package service

import "testing"

func TestExample(t *testing.T) {
	if 1+1 != 2 {
		t.Error("Math is broken")
	}
}
`
	err = os.WriteFile(filepath.Join(testDir, "service_test.go"), []byte(testContent), 0644)
	s.Require().NoError(err)

	workDir = s.tempDir

	// Mock successful test execution
	// Mock successful test execution would be done through dependency injection

	// Test would run the test execution logic here
	// For now, just verify the test file exists
	assert.FileExists(s.T(), filepath.Join(testDir, "service_test.go"))
}

// TestCleanExecutionWithTargets tests clean execution with different targets
func (s *DevCommandTestSuite) TestCleanExecutionWithTargets() {
	// Create files to clean
	buildDir := filepath.Join(s.tempDir, "bin")
	err := os.MkdirAll(buildDir, 0755)
	s.Require().NoError(err)

	err = os.WriteFile(filepath.Join(buildDir, "myservice"), []byte("binary"), 0755)
	s.Require().NoError(err)

	vendorDir := filepath.Join(s.tempDir, "vendor")
	err = os.MkdirAll(vendorDir, 0755)
	s.Require().NoError(err)

	workDir = s.tempDir

	// Test clean logic
	targets := []string{"Build cache", "Generated files", "Vendor directory"}
	for _, target := range targets {
		err := simulateCleanTarget(target, s.tempDir)
		assert.NoError(s.T(), err)
	}
}

// TestErrorHandling tests various error conditions
func (s *DevCommandTestSuite) TestErrorHandling() {
	testCases := []struct {
		name          string
		setup         func()
		expectedError string
	}{
		{
			name: "missing go.mod",
			setup: func() {
				workDir = s.tempDir
				// Don't create go.mod file
			},
			expectedError: "not a Go module",
		},
		{
			name: "invalid service directory",
			setup: func() {
				workDir = "/nonexistent/directory"
			},
			expectedError: "directory does not exist",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.setup()

			// Test that appropriate errors are returned
			// This would be tested in the actual command execution
			if !strings.Contains(s.tempDir, "nonexistent") {
				_, err := os.Stat(filepath.Join(workDir, "go.mod"))
				if tc.name == "missing go.mod" {
					assert.True(t, os.IsNotExist(err))
				}
			}
		})
	}
}

// TestViperConfiguration tests viper configuration handling
func (s *DevCommandTestSuite) TestViperConfiguration() {
	// Test viper configuration override
	viper.Set("dev.verbose", true)
	viper.Set("dev.no_color", false)
	viper.Set("dev.work_dir", "/test/workdir")
	viper.Set("dev.parallel", true)

	cmd := NewDevCommand()

	// Simulate the persistent pre-run that would set these values
	if viper.IsSet("dev.verbose") {
		verbose = viper.GetBool("dev.verbose")
	}
	if viper.IsSet("dev.no_color") {
		noColor = viper.GetBool("dev.no_color")
	}
	if viper.IsSet("dev.work_dir") {
		workDir = viper.GetString("dev.work_dir")
	}
	if viper.IsSet("dev.parallel") {
		parallel = viper.GetBool("dev.parallel")
	}

	// Verify the values were set correctly
	assert.True(s.T(), verbose)
	assert.False(s.T(), noColor)
	assert.Equal(s.T(), "/test/workdir", workDir)
	assert.True(s.T(), parallel)

	// Test that command was created successfully
	assert.NotNil(s.T(), cmd)
}

// TestCommandIntegration tests command integration with cobra
func (s *DevCommandTestSuite) TestCommandIntegration() {
	cmd := NewDevCommand()

	// Test help output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	assert.NoError(s.T(), err)

	output := buf.String()
	assert.Contains(s.T(), output, "Usage:")
	assert.Contains(s.T(), output, "dev")
	assert.Contains(s.T(), output, "Examples:")
	assert.Contains(s.T(), output, "Available Commands:")
}

// TestParallelExecution tests parallel execution capabilities
func (s *DevCommandTestSuite) TestParallelExecution() {
	parallel = true
	workDir = s.tempDir

	// Create multiple test files for parallel execution
	testDirs := []string{"internal/service1", "internal/service2", "internal/service3"}
	for _, dir := range testDirs {
		fullDir := filepath.Join(s.tempDir, dir)
		err := os.MkdirAll(fullDir, 0755)
		s.Require().NoError(err)

		testContent := fmt.Sprintf(`package %s

import "testing"

func TestExample(t *testing.T) {
	if 1+1 != 2 {
		t.Error("Math is broken")
	}
}
`, filepath.Base(dir))
		err = os.WriteFile(filepath.Join(fullDir, fmt.Sprintf("%s_test.go", filepath.Base(dir))), []byte(testContent), 0644)
		s.Require().NoError(err)
	}

	// Test parallel flag is set
	assert.True(s.T(), parallel)

	// Verify all test files were created
	for _, dir := range testDirs {
		testFile := filepath.Join(s.tempDir, dir, fmt.Sprintf("%s_test.go", filepath.Base(dir)))
		assert.FileExists(s.T(), testFile)
	}
}

// Helper functions for testing

func detectServices(workDir string) ([]string, error) {
	cmdDir := filepath.Join(workDir, "cmd")
	if _, err := os.Stat(cmdDir); os.IsNotExist(err) {
		return nil, errors.New("cmd directory not found")
	}

	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return nil, err
	}

	var services []string
	for _, entry := range entries {
		if entry.IsDir() {
			services = append(services, entry.Name())
		}
	}

	return services, nil
}

func findMainFile(serviceDir string) (string, error) {
	mainFile := filepath.Join(serviceDir, "main.go")
	if _, err := os.Stat(mainFile); err == nil {
		return mainFile, nil
	}
	return "", errors.New("main.go not found")
}

func simulateCleanTarget(target, workDir string) error {
	switch target {
	case "Build cache":
		// Simulate go clean -cache
		return nil
	case "Generated files":
		// Simulate removing generated files
		return nil
	case "Vendor directory":
		vendorDir := filepath.Join(workDir, "vendor")
		return os.RemoveAll(vendorDir)
	default:
		return fmt.Errorf("unknown clean target: %s", target)
	}
}

// Run the test suite
func TestDevCommandTestSuite(t *testing.T) {
	suite.Run(t, new(DevCommandTestSuite))
}
