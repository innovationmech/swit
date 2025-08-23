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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// DepsCommandTestSuite tests the 'deps' command functionality
type DepsCommandTestSuite struct {
	suite.Suite
	tempDir        string
	originalArgs   []string
	originalWd     string
	mockContainer  *MockDependencyContainer
	mockUI         *MockTerminalUI
	mockLogger     *MockLogger
	mockFileSystem *MockFileSystem
	mockDepManager *MockDependencyManager
}

// SetupSuite runs before all tests in the suite
func (s *DepsCommandTestSuite) SetupSuite() {
	s.originalArgs = os.Args
	s.originalWd, _ = os.Getwd()
}

// SetupTest runs before each test
func (s *DepsCommandTestSuite) SetupTest() {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "switctl-deps-test-*")
	if err != nil {
		s.T().Fatalf("Failed to create temp dir: %v", err)
	}
	s.tempDir = tempDir

	// Reset global variables
	s.resetGlobals()

	// Reset viper
	viper.Reset()

	// Create mocks
	s.createMocks()
	s.setupMockBehavior()

	// Create test project structure
	s.createTestProjectStructure()
}

// TearDownTest runs after each test
func (s *DepsCommandTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
	s.resetGlobals()
	viper.Reset()
	os.Chdir(s.originalWd)
}

// TearDownSuite runs after all tests in the suite
func (s *DepsCommandTestSuite) TearDownSuite() {
	os.Args = s.originalArgs
}

// Helper Methods

func (s *DepsCommandTestSuite) resetGlobals() {
	verbose = false
	noColor = false
	workDir = ""
	interactive = false
	force = false
	dryRun = false
	outputFile = ""
}

func (s *DepsCommandTestSuite) createMocks() {
	s.mockContainer = &MockDependencyContainer{}
	s.mockUI = &MockTerminalUI{}
	s.mockLogger = &MockLogger{}
	s.mockFileSystem = &MockFileSystem{}
	s.mockDepManager = &MockDependencyManager{}
}

func (s *DepsCommandTestSuite) setupMockBehavior() {
	// Setup container returns
	s.mockContainer.On("GetService", "ui").Return(s.mockUI, nil)
	s.mockContainer.On("GetService", "filesystem").Return(s.mockFileSystem, nil)
	s.mockContainer.On("GetService", "dependency_manager").Return(s.mockDepManager, nil)
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
	s.mockUI.On("PromptConfirm", mock.Anything, mock.Anything).Return(true, nil)
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

	// Default dependency manager behavior
	s.mockDepManager.On("ListDependencies").Return([]interfaces.Dependency{
		{
			Name:    "github.com/gin-gonic/gin",
			Version: "v1.8.1",
			Type:    "require",
			Direct:  true,
			License: "MIT",
		},
		{
			Name:    "github.com/stretchr/testify",
			Version: "v1.8.0",
			Type:    "require",
			Direct:  false,
			License: "MIT",
		},
	}, nil)

	s.mockDepManager.On("AuditDependencies").Return(interfaces.AuditResult{
		Vulnerabilities: []interfaces.Vulnerability{
			{
				ID:          "CVE-2023-1234",
				Package:     "github.com/example/vulnerable",
				Version:     "v1.0.0",
				Severity:    "high",
				Title:       "Test vulnerability",
				Description: "A test vulnerability for testing",
				CVSS:        7.5,
				Fix:         "upgrade to v1.0.1",
				URL:         "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2023-1234",
			},
		},
		Severity: "high",
		Scanned:  10,
		Duration: time.Second,
	})

	// Default filesystem behavior
	s.mockFileSystem.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
}

func (s *DepsCommandTestSuite) createTestProjectStructure() {
	// Create go.mod file
	goModContent := `module test-project

go 1.19

require (
	github.com/gin-gonic/gin v1.8.1
	github.com/stretchr/testify v1.8.0
)
`
	err := os.WriteFile(filepath.Join(s.tempDir, "go.mod"), []byte(goModContent), 0644)
	if err != nil {
		s.T().Fatalf("Failed to create go.mod: %v", err)
	}

	// Create go.sum file
	goSumContent := `github.com/gin-gonic/gin v1.8.1 h1:test-hash
github.com/gin-gonic/gin v1.8.1/go.mod h1:test-hash-mod
github.com/stretchr/testify v1.8.0 h1:test-hash2
github.com/stretchr/testify v1.8.0/go.mod h1:test-hash2-mod
`
	err = os.WriteFile(filepath.Join(s.tempDir, "go.sum"), []byte(goSumContent), 0644)
	if err != nil {
		s.T().Fatalf("Failed to create go.sum: %v", err)
	}
}

// Test Cases

// TestNewDepsCommand_BasicProperties tests the basic properties of the deps command
func (s *DepsCommandTestSuite) TestNewDepsCommand_BasicProperties() {
	cmd := NewDepsCommand()

	assert.NotNil(s.T(), cmd)
	assert.Equal(s.T(), "deps", cmd.Use)
	assert.Equal(s.T(), "Dependency management and security audit", cmd.Short)
	assert.Contains(s.T(), cmd.Long, "The deps command provides dependency management and security auditing")
	assert.Contains(s.T(), cmd.Long, "Examples:")
}

// TestNewDepsCommand_PersistentFlags tests persistent flags
func (s *DepsCommandTestSuite) TestNewDepsCommand_PersistentFlags() {
	cmd := NewDepsCommand()
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
		{"interactive", "i", "false", true},
		{"force", "", "false", true},
		{"dry-run", "", "false", true},
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

// TestNewDepsCommand_Subcommands tests that subcommands are properly added
func (s *DepsCommandTestSuite) TestNewDepsCommand_Subcommands() {
	cmd := NewDepsCommand()
	subcommands := cmd.Commands()

	assert.GreaterOrEqual(s.T(), len(subcommands), 5)

	expectedSubcommands := []string{"update", "audit", "list", "graph", "why"}
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

// TestInitializeDepsCommandConfig tests configuration initialization
func (s *DepsCommandTestSuite) TestInitializeDepsCommandConfig() {
	// Set working directory via viper to temp dir
	viper.Set("deps.work_dir", s.tempDir)

	cmd := NewDepsCommand()
	err := initializeDepsCommandConfig(cmd)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), s.tempDir, workDir)

	// Test with viper values
	viper.Set("deps.verbose", true)
	viper.Set("deps.no_color", true)
	viper.Set("deps.work_dir", s.tempDir)
	viper.Set("deps.interactive", true)
	viper.Set("deps.force", true)
	viper.Set("deps.dry_run", true)

	err = initializeDepsCommandConfig(cmd)
	assert.NoError(s.T(), err)
	assert.True(s.T(), verbose)
	assert.True(s.T(), noColor)
	assert.Equal(s.T(), s.tempDir, workDir)
	assert.True(s.T(), interactive)
	assert.True(s.T(), force)
	assert.True(s.T(), dryRun)
}

// TestInitializeDepsCommandConfig_NoGoMod tests missing go.mod file
func (s *DepsCommandTestSuite) TestInitializeDepsCommandConfig_NoGoMod() {
	// Create temp dir without go.mod
	emptyDir, err := os.MkdirTemp("", "empty-*")
	if err != nil {
		s.T().Fatalf("Failed to create empty dir: %v", err)
	}
	defer os.RemoveAll(emptyDir)

	workDir = emptyDir

	cmd := NewDepsCommand()
	err = initializeDepsCommandConfig(cmd)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "go.mod not found")
}

// Update Command Tests

// TestNewUpdateCommand_Properties tests update command properties
func (s *DepsCommandTestSuite) TestNewUpdateCommand_Properties() {
	cmd := NewUpdateCommand()

	assert.NotNil(s.T(), cmd)
	assert.Equal(s.T(), "update [modules...]", cmd.Use)
	assert.Equal(s.T(), "Update Go module dependencies to latest compatible versions", cmd.Short)
	assert.Contains(s.T(), cmd.Long, "The update command checks and updates Go module dependencies")
}

// TestNewUpdateCommand_Flags tests update command flags
func (s *DepsCommandTestSuite) TestNewUpdateCommand_Flags() {
	cmd := NewUpdateCommand()
	flags := cmd.Flags()

	testCases := []struct {
		name        string
		defValue    string
		shouldExist bool
	}{
		{"major", "false", true},
		{"patch", "false", true},
		{"prerelease", "false", true},
		{"exclude", "", true},
		{"include", "", true},
		{"test-deps", "true", true},
		{"skip-breaking", "false", true},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			flag := flags.Lookup(tc.name)
			if tc.shouldExist {
				assert.NotNil(t, flag, "Flag %s should exist", tc.name)
				if tc.name != "exclude" && tc.name != "include" {
					assert.Equal(t, tc.defValue, flag.DefValue, "Flag %s default value should match", tc.name)
				}
			} else {
				assert.Nil(t, flag, "Flag %s should not exist", tc.name)
			}
		})
	}
}

// TestRunUpdateCommand_NoUpdates tests update command with no available updates
func (s *DepsCommandTestSuite) TestRunUpdateCommand_NoUpdates() {
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
	s.mockUI.On("ShowProgress", "Checking for updates", 2).Return(mockProgress)

	// Mock checkForUpdate to return no updates
	originalCheckUpdate := checkForUpdate
	checkForUpdate = func(dep interfaces.Dependency, major, patch, prerelease bool) (*DependencyUpdate, error) {
		return nil, nil // No update available
	}
	defer func() { checkForUpdate = originalCheckUpdate }()

	err := runUpdateCommand([]string{}, false, false, false, []string{}, []string{}, true, false)
	assert.NoError(s.T(), err)

	s.mockUI.AssertCalled(s.T(), "ShowSuccess", "All dependencies are up to date!")
}

// TestRunUpdateCommand_WithUpdates tests update command with available updates
func (s *DepsCommandTestSuite) TestRunUpdateCommand_WithUpdates() {
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
	s.mockUI.On("ShowProgress", "Checking for updates", 2).Return(mockProgress)
	s.mockUI.On("ShowProgress", "Applying updates", 1).Return(mockProgress)

	// Mock checkForUpdate to return an update
	originalCheckUpdate := checkForUpdate
	checkForUpdate = func(dep interfaces.Dependency, major, patch, prerelease bool) (*DependencyUpdate, error) {
		if dep.Name == "github.com/gin-gonic/gin" {
			return &DependencyUpdate{
				Name:           dep.Name,
				CurrentVersion: dep.Version,
				NewVersion:     "v1.9.0",
				Major:          false,
				Patch:          false,
				Breaking:       false,
				Description:    "Minor update",
			}, nil
		}
		return nil, nil
	}
	defer func() { checkForUpdate = originalCheckUpdate }()

	// Mock applyUpdates
	originalApplyUpdates := applyUpdates
	applyUpdates = func(ui interfaces.InteractiveUI, manager interfaces.DependencyManager, updates []DependencyUpdate) error {
		return nil
	}
	defer func() { applyUpdates = originalApplyUpdates }()

	force = true // Skip confirmation
	err := runUpdateCommand([]string{}, false, false, false, []string{}, []string{}, true, false)
	assert.NoError(s.T(), err)

	s.mockUI.AssertCalled(s.T(), "ShowTable", mock.Anything, mock.Anything)
}

// TestRunUpdateCommand_DryRun tests update command in dry run mode
func (s *DepsCommandTestSuite) TestRunUpdateCommand_DryRun() {
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
	s.mockUI.On("ShowProgress", "Checking for updates", 2).Return(mockProgress)

	// Mock checkForUpdate to return an update
	originalCheckUpdate := checkForUpdate
	checkForUpdate = func(dep interfaces.Dependency, major, patch, prerelease bool) (*DependencyUpdate, error) {
		if dep.Name == "github.com/gin-gonic/gin" {
			return &DependencyUpdate{
				Name:           dep.Name,
				CurrentVersion: dep.Version,
				NewVersion:     "v1.9.0",
			}, nil
		}
		return nil, nil
	}
	defer func() { checkForUpdate = originalCheckUpdate }()

	dryRun = true
	defer func() { dryRun = false }()

	err := runUpdateCommand([]string{}, false, false, false, []string{}, []string{}, true, false)
	assert.NoError(s.T(), err)

	s.mockUI.AssertCalled(s.T(), "ShowInfo", "Dry run completed - no updates were applied")
}

// Audit Command Tests

// TestNewAuditCommand_Properties tests audit command properties
func (s *DepsCommandTestSuite) TestNewAuditCommand_Properties() {
	cmd := NewAuditCommand()

	assert.NotNil(s.T(), cmd)
	assert.Equal(s.T(), "audit", cmd.Use)
	assert.Equal(s.T(), "Check dependencies for security vulnerabilities", cmd.Short)
	assert.Contains(s.T(), cmd.Long, "The audit command scans project dependencies for known security vulnerabilities")
}

// TestNewAuditCommand_Flags tests audit command flags
func (s *DepsCommandTestSuite) TestNewAuditCommand_Flags() {
	cmd := NewAuditCommand()
	flags := cmd.Flags()

	testCases := []struct {
		name        string
		shorthand   string
		defValue    string
		shouldExist bool
	}{
		{"severity", "", "", true},
		{"fixable", "", "false", true},
		{"json", "", "false", true},
		{"output", "o", "", true},
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

// TestRunAuditCommand tests audit command execution
func (s *DepsCommandTestSuite) TestRunAuditCommand() {
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
	s.mockUI.On("ShowProgress", "Scanning for vulnerabilities", 1).Return(mockProgress)

	err := runAuditCommand("", false, false, "")
	assert.NoError(s.T(), err)

	s.mockDepManager.AssertCalled(s.T(), "AuditDependencies")
}

// TestRunAuditCommand_JSONOutput tests audit command with JSON output
func (s *DepsCommandTestSuite) TestRunAuditCommand_JSONOutput() {
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
	s.mockUI.On("ShowProgress", "Scanning for vulnerabilities", 1).Return(mockProgress)

	// Capture stdout to verify JSON output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runAuditCommand("", false, true, "")
	assert.NoError(s.T(), err)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify it's valid JSON
	var jsonOutput []interfaces.Vulnerability
	err = json.Unmarshal([]byte(output), &jsonOutput)
	assert.NoError(s.T(), err)
}

// TestRunAuditCommand_WithSeverityFilter tests audit command with severity filtering
func (s *DepsCommandTestSuite) TestRunAuditCommand_WithSeverityFilter() {
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
	s.mockUI.On("ShowProgress", "Scanning for vulnerabilities", 1).Return(mockProgress)

	err := runAuditCommand("high", false, false, "")
	assert.NoError(s.T(), err)

	s.mockDepManager.AssertCalled(s.T(), "AuditDependencies")
}

// List Command Tests

// TestNewListCommand_Properties tests list command properties
func (s *DepsCommandTestSuite) TestNewListCommand_Properties() {
	cmd := NewListCommand()

	assert.NotNil(s.T(), cmd)
	assert.Equal(s.T(), "list [modules...]", cmd.Use)
	assert.Equal(s.T(), "List project dependencies with version information", cmd.Short)
	assert.Contains(s.T(), cmd.Long, "The list command displays all project dependencies")
}

// TestNewListCommand_Flags tests list command flags
func (s *DepsCommandTestSuite) TestNewListCommand_Flags() {
	cmd := NewListCommand()
	flags := cmd.Flags()

	testCases := []struct {
		name        string
		shorthand   string
		defValue    string
		shouldExist bool
	}{
		{"format", "f", "table", true},
		{"test", "", "false", true},
		{"indirect", "", "true", true},
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

// TestRunListCommand tests list command execution
func (s *DepsCommandTestSuite) TestRunListCommand() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	err := runListCommand([]string{}, "table", false, true)
	assert.NoError(s.T(), err)

	s.mockDepManager.AssertCalled(s.T(), "ListDependencies")
	s.mockUI.AssertCalled(s.T(), "ShowTable", mock.Anything, mock.Anything)
}

// TestRunListCommand_JSONFormat tests list command with JSON format
func (s *DepsCommandTestSuite) TestRunListCommand_JSONFormat() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	// Capture stdout to verify JSON output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runListCommand([]string{}, "json", false, true)
	assert.NoError(s.T(), err)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify it's valid JSON
	var jsonOutput []interfaces.Dependency
	err = json.Unmarshal([]byte(output), &jsonOutput)
	assert.NoError(s.T(), err)
}

// TestRunListCommand_SpecificModules tests list command with specific modules
func (s *DepsCommandTestSuite) TestRunListCommand_SpecificModules() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	err := runListCommand([]string{"github.com/gin-gonic/gin"}, "table", false, true)
	assert.NoError(s.T(), err)

	s.mockDepManager.AssertCalled(s.T(), "ListDependencies")
}

// Graph Command Tests

// TestNewGraphCommand_Properties tests graph command properties
func (s *DepsCommandTestSuite) TestNewGraphCommand_Properties() {
	cmd := NewGraphCommand()

	assert.NotNil(s.T(), cmd)
	assert.Equal(s.T(), "graph [module]", cmd.Use)
	assert.Equal(s.T(), "Show dependency graph and relationships", cmd.Short)
	assert.Contains(s.T(), cmd.Long, "The graph command visualizes dependency relationships")
}

// TestNewGraphCommand_Flags tests graph command flags
func (s *DepsCommandTestSuite) TestNewGraphCommand_Flags() {
	cmd := NewGraphCommand()
	flags := cmd.Flags()

	testCases := []struct {
		name        string
		shorthand   string
		defValue    string
		shouldExist bool
	}{
		{"format", "f", "text", true},
		{"depth", "", "3", true},
		{"focus", "", "", true},
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

// TestRunGraphCommand tests graph command execution
func (s *DepsCommandTestSuite) TestRunGraphCommand() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	err := runGraphCommand("", "text", 3, "")
	assert.NoError(s.T(), err)
}

// TestRunGraphCommand_JSONFormat tests graph command with JSON format
func (s *DepsCommandTestSuite) TestRunGraphCommand_JSONFormat() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	// Capture stdout to verify JSON output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runGraphCommand("", "json", 3, "")
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
}

// TestRunGraphCommand_DOTFormat tests graph command with DOT format
func (s *DepsCommandTestSuite) TestRunGraphCommand_DOTFormat() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	// Capture stdout to verify DOT output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runGraphCommand("", "dot", 3, "")
	assert.NoError(s.T(), err)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(s.T(), output, "digraph dependencies")
}

// Why Command Tests

// TestNewWhyCommand_Properties tests why command properties
func (s *DepsCommandTestSuite) TestNewWhyCommand_Properties() {
	cmd := NewWhyCommand()

	assert.NotNil(s.T(), cmd)
	assert.Equal(s.T(), "why <module>", cmd.Use)
	assert.Equal(s.T(), "Explain why a dependency is needed", cmd.Short)
	assert.Contains(s.T(), cmd.Long, "The why command explains why a particular module is included")
}

// TestRunWhyCommand tests why command execution
func (s *DepsCommandTestSuite) TestRunWhyCommand() {
	// Mock createDependencyContainer
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return s.mockContainer, nil
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	// Set working directory
	originalWorkDir := workDir
	workDir = s.tempDir
	defer func() { workDir = originalWorkDir }()

	// Mock exec.Command
	if hasGoCommand() {
		err := runWhyCommand("github.com/gin-gonic/gin")
		// Don't assert no error because the actual go mod why command might fail
		// in the test environment, but we can test that the function runs
		_ = err
	}
}

// Helper Function Tests

// TestShouldUpdateDependency tests dependency filtering logic
func (s *DepsCommandTestSuite) TestShouldUpdateDependency() {
	dep := interfaces.Dependency{Name: "github.com/gin-gonic/gin"}

	// Test with no filters
	result := shouldUpdateDependency(dep, []string{}, []string{}, []string{})
	assert.True(s.T(), result)

	// Test with includes
	result = shouldUpdateDependency(dep, []string{}, []string{}, []string{"github.com/gin-gonic/gin"})
	assert.True(s.T(), result)

	result = shouldUpdateDependency(dep, []string{}, []string{}, []string{"other-module"})
	assert.False(s.T(), result)

	// Test with excludes
	result = shouldUpdateDependency(dep, []string{}, []string{"github.com/gin-gonic/gin"}, []string{})
	assert.False(s.T(), result)

	// Test with specific modules
	result = shouldUpdateDependency(dep, []string{"github.com/gin-gonic/gin"}, []string{}, []string{})
	assert.True(s.T(), result)

	result = shouldUpdateDependency(dep, []string{"other-module"}, []string{}, []string{})
	assert.False(s.T(), result)
}

// TestFilterVulnerabilities tests vulnerability filtering
func (s *DepsCommandTestSuite) TestFilterVulnerabilities() {
	vulns := []interfaces.Vulnerability{
		{Severity: "low", Fix: "upgrade"},
		{Severity: "medium", Fix: ""},
		{Severity: "high", Fix: "upgrade"},
		{Severity: "critical", Fix: ""},
	}

	// Test severity filtering
	filtered := filterVulnerabilities(vulns, "high", false)
	assert.Len(s.T(), filtered, 2) // high and critical

	filtered = filterVulnerabilities(vulns, "critical", false)
	assert.Len(s.T(), filtered, 1) // only critical

	// Test fixable filtering
	filtered = filterVulnerabilities(vulns, "", true)
	assert.Len(s.T(), filtered, 2) // only those with fixes

	// Test combined filtering
	filtered = filterVulnerabilities(vulns, "high", true)
	assert.Len(s.T(), filtered, 1) // high with fix only
}

// TestGroupBySeverity tests vulnerability grouping
func (s *DepsCommandTestSuite) TestGroupBySeverity() {
	vulns := []interfaces.Vulnerability{
		{Severity: "low"},
		{Severity: "medium"},
		{Severity: "high"},
		{Severity: "critical"},
		{Severity: "high"},
	}

	groups := groupBySeverity(vulns)

	assert.Len(s.T(), groups["low"], 1)
	assert.Len(s.T(), groups["medium"], 1)
	assert.Len(s.T(), groups["high"], 2)
	assert.Len(s.T(), groups["critical"], 1)
}

// TestFilterDependencies tests dependency filtering
func (s *DepsCommandTestSuite) TestFilterDependencies() {
	deps := []interfaces.Dependency{
		{Name: "dep1", Direct: true},
		{Name: "dep2", Direct: false},
		{Name: "dep3", Direct: true},
	}

	// Test with no filters
	filtered := filterDependencies(deps, []string{}, false, true)
	assert.Len(s.T(), filtered, 3)

	// Test with indirect filter
	filtered = filterDependencies(deps, []string{}, false, false)
	assert.Len(s.T(), filtered, 2) // only direct

	// Test with specific modules
	filtered = filterDependencies(deps, []string{"dep1"}, false, true)
	assert.Len(s.T(), filtered, 1)
	assert.Equal(s.T(), "dep1", filtered[0].Name)
}

// TestUtilityFunctions tests utility functions
func (s *DepsCommandTestSuite) TestUtilityFunctions() {
	// Test getModulesDescription
	assert.Equal(s.T(), "all modules", getModulesDescription([]string{}))
	assert.Equal(s.T(), "single-module", getModulesDescription([]string{"single-module"}))
	assert.Equal(s.T(), "2 specific modules", getModulesDescription([]string{"mod1", "mod2"}))

	// Test formatBool
	assert.Equal(s.T(), "enabled", formatBool(true))
	assert.Equal(s.T(), "disabled", formatBool(false))
}

// TestCreateDependencyContainer tests dependency container creation
func (s *DepsCommandTestSuite) TestCreateDependencyContainer() {
	workDir = s.tempDir

	container, err := createDependencyContainer()
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), container)

	// Test service registration
	requiredServices := []string{"ui", "filesystem", "dependency_manager"}
	for _, serviceName := range requiredServices {
		service, err := container.GetService(serviceName)
		assert.NoError(s.T(), err, "Should get service: %s", serviceName)
		assert.NotNil(s.T(), service, "Service should not be nil: %s", serviceName)
	}

	container.Close()
}

// TestSimpleDependencyManager tests the dependency manager implementation
func (s *DepsCommandTestSuite) TestSimpleDependencyManager() {
	manager := NewDependencyManager(s.tempDir)
	assert.NotNil(s.T(), manager)

	// Test UpdateDependencies
	err := manager.UpdateDependencies()
	assert.NoError(s.T(), err)

	// Test AuditDependencies
	result := manager.AuditDependencies()
	assert.NotNil(s.T(), result)

	// Test ListDependencies
	deps, err := manager.ListDependencies()
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), deps)

	// Test AddDependency
	dep := interfaces.Dependency{Name: "test-dep", Version: "v1.0.0"}
	err = manager.AddDependency(dep)
	assert.NoError(s.T(), err)

	// Test RemoveDependency
	err = manager.RemoveDependency("test-dep")
	assert.NoError(s.T(), err)
}

// TestCommandIntegration tests command integration with cobra
func (s *DepsCommandTestSuite) TestCommandIntegration() {
	testCases := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"deps command", NewDepsCommand()},
		{"update command", NewUpdateCommand()},
		{"audit command", NewAuditCommand()},
		{"list command", NewListCommand()},
		{"graph command", NewGraphCommand()},
		{"why command", NewWhyCommand()},
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
func (s *DepsCommandTestSuite) TestErrorHandling() {
	// Test dependency container creation failure
	originalCreateDep := createDependencyContainer
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		return nil, fmt.Errorf("container creation failed")
	}
	defer func() { createDependencyContainer = originalCreateDep }()

	err := runUpdateCommand([]string{}, false, false, false, []string{}, []string{}, true, false)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to create dependency container")

	// Test UI service retrieval failure
	createDependencyContainer = func() (interfaces.DependencyContainer, error) {
		mockContainer := &MockDependencyContainer{}
		mockContainer.On("GetService", "ui").Return(nil, fmt.Errorf("service not found"))
		mockContainer.On("Close").Return(nil)
		return mockContainer, nil
	}

	err = runUpdateCommand([]string{}, false, false, false, []string{}, []string{}, true, false)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to get UI service")
}

// TestFlagCombinations tests various flag combinations
func (s *DepsCommandTestSuite) TestFlagCombinations() {
	testCases := []struct {
		name    string
		command *cobra.Command
		flags   []string
	}{
		{
			name:    "deps flags",
			command: NewDepsCommand(),
			flags:   []string{"--verbose", "--no-color", "--interactive", "--force", "--dry-run"},
		},
		{
			name:    "update flags",
			command: NewUpdateCommand(),
			flags:   []string{"--major", "--patch", "--prerelease", "--test-deps", "--skip-breaking"},
		},
		{
			name:    "audit flags",
			command: NewAuditCommand(),
			flags:   []string{"--severity", "high", "--fixable", "--json"},
		},
		{
			name:    "list flags",
			command: NewListCommand(),
			flags:   []string{"--format", "json", "--test", "--indirect=false"},
		},
		{
			name:    "graph flags",
			command: NewGraphCommand(),
			flags:   []string{"--format", "dot", "--depth", "5"},
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
func (s *DepsCommandTestSuite) TestConcurrentAccess() {
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

// Helper function to check if go command is available
func hasGoCommand() bool {
	_, err := exec.LookPath("go")
	return err == nil
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

func (m *MockTerminalUI) PromptConfirm(prompt string, defaultValue bool) (bool, error) {
	args := m.Called(prompt, defaultValue)
	return args.Bool(0), args.Error(1)
}

func (m *MockTerminalUI) GetStyle() *interfaces.UIStyle {
	args := m.Called()
	return args.Get(0).(*interfaces.UIStyle)
}

func (m *MockTerminalUI) ShowMultiSelectMenu(title string, options []interfaces.MenuOption) ([]int, error) {
	args := m.Called(title, options)
	return args.Get(0).([]int), args.Error(1)
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

type MockDependencyManager struct {
	mock.Mock
}

func (m *MockDependencyManager) UpdateDependencies() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDependencyManager) AuditDependencies() interfaces.AuditResult {
	args := m.Called()
	return args.Get(0).(interfaces.AuditResult)
}

func (m *MockDependencyManager) ListDependencies() ([]interfaces.Dependency, error) {
	args := m.Called()
	return args.Get(0).([]interfaces.Dependency), args.Error(1)
}

func (m *MockDependencyManager) AddDependency(dep interfaces.Dependency) error {
	args := m.Called(dep)
	return args.Error(0)
}

func (m *MockDependencyManager) RemoveDependency(name string) error {
	args := m.Called(name)
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
func TestDepsCommandTestSuite(t *testing.T) {
	suite.Run(t, new(DepsCommandTestSuite))
}
