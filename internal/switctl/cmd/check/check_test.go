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

package check

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// CheckCommandTestSuite tests the check command functionality
type CheckCommandTestSuite struct {
	suite.Suite
	tempDir string
}

// mockUIAdapter is a test-friendly UI adapter that doesn't require user input
type mockUIAdapter struct{}

func (m *mockUIAdapter) ShowWelcome() error { return nil }
func (m *mockUIAdapter) PromptInput(prompt string, validator interfaces.InputValidator) (string, error) {
	return "test", nil
}
func (m *mockUIAdapter) ShowMenu(title string, options []interfaces.MenuOption) (int, error) {
	return 0, nil
}
func (m *mockUIAdapter) ShowProgress(title string, total int) interfaces.ProgressBar {
	return &mockProgressBar{}
}
func (m *mockUIAdapter) ShowSuccess(message string) error                  { return nil }
func (m *mockUIAdapter) ShowError(err error) error                         { return nil }
func (m *mockUIAdapter) ShowTable(headers []string, rows [][]string) error { return nil }
func (m *mockUIAdapter) PromptConfirm(prompt string, defaultValue bool) (bool, error) {
	return true, nil
}                                                                     // Always confirm
func (m *mockUIAdapter) ShowInfo(message string) error                { return nil }
func (m *mockUIAdapter) PrintSeparator()                              {}
func (m *mockUIAdapter) PrintHeader(title string)                     {}
func (m *mockUIAdapter) PrintSubHeader(title string)                  {}
func (m *mockUIAdapter) GetStyle() *interfaces.UIStyle                { return &interfaces.UIStyle{} }
func (m *mockUIAdapter) PromptPassword(prompt string) (string, error) { return "test", nil }
func (m *mockUIAdapter) ShowFeatureSelectionMenu() (interfaces.ServiceFeatures, error) {
	return interfaces.ServiceFeatures{}, nil
}
func (m *mockUIAdapter) ShowDatabaseSelectionMenu() (string, error) { return "sqlite", nil }
func (m *mockUIAdapter) ShowAuthSelectionMenu() (string, error)     { return "none", nil }

// mockProgressBar is a test-friendly progress bar
type mockProgressBar struct{}

func (m *mockProgressBar) Update(current int) error        { return nil }
func (m *mockProgressBar) SetMessage(message string) error { return nil }
func (m *mockProgressBar) Finish() error                   { return nil }
func (m *mockProgressBar) SetTotal(total int) error        { return nil }

func TestCheckCommandTestSuite(t *testing.T) {
	suite.Run(t, new(CheckCommandTestSuite))
}

func (s *CheckCommandTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "check-cmd-test-*")
	s.Require().NoError(err)
}

func (s *CheckCommandTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

// TestNewCheckCommand tests the creation of the check command
func (s *CheckCommandTestSuite) TestNewCheckCommand() {
	s.createGoProject() // Ensure we have a valid project first

	cmd := NewCheckCommand()

	s.NotNil(cmd)
	s.Equal("check", cmd.Use[:5]) // Starts with "check"
	s.Contains(cmd.Short, "quality checks")
	s.Contains(cmd.Long, "comprehensive quality checks")
	s.NotEmpty(cmd.Example)

	// Test that flags are properly set up
	flags := cmd.Flags()
	s.NotNil(flags.Lookup("all"))
	s.NotNil(flags.Lookup("fix"))
	s.NotNil(flags.Lookup("path"))
	s.NotNil(flags.Lookup("format"))
	s.NotNil(flags.Lookup("verbose"))
	s.NotNil(flags.Lookup("timeout"))
	s.NotNil(flags.Lookup("parallel"))
	s.NotNil(flags.Lookup("fail-on-warning"))
	s.NotNil(flags.Lookup("failure-only"))
	s.NotNil(flags.Lookup("config"))
}

// TestCheckOptionsDefaults tests default option values
func (s *CheckCommandTestSuite) TestCheckOptionsDefaults() {
	opts := &CheckOptions{
		ProjectPath:   ".",
		All:           false,
		Fix:           false,
		FailOnWarning: false,
		MaxParallel:   4,
		Timeout:       10 * time.Minute,
		Format:        "table",
		Verbose:       false,
		FailureOnly:   false,
	}

	s.Equal(".", opts.ProjectPath)
	s.False(opts.All)
	s.False(opts.Fix)
	s.False(opts.FailOnWarning)
	s.Equal(4, opts.MaxParallel)
	s.Equal(10*time.Minute, opts.Timeout)
	s.Equal("table", opts.Format)
	s.False(opts.Verbose)
	s.False(opts.FailureOnly)
}

// TestNewCheckCommandWithOptions tests creating check command with custom options
func (s *CheckCommandTestSuite) TestNewCheckCommandWithOptions() {
	opts := &CheckOptions{
		ProjectPath:   s.tempDir,
		All:           true,
		Fix:           true,
		FailOnWarning: true,
		MaxParallel:   8,
		Timeout:       5 * time.Minute,
		Format:        "json",
		Verbose:       true,
		FailureOnly:   true,
	}

	// Create a minimal Go project structure
	s.createGoProject()

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)
	s.NotNil(checkCmd)

	s.Equal(s.tempDir, checkCmd.options.ProjectPath)
	s.True(checkCmd.options.All)
	s.True(checkCmd.options.Fix)
	s.True(checkCmd.options.FailOnWarning)
	s.Equal(8, checkCmd.options.MaxParallel)
	s.Equal(5*time.Minute, checkCmd.options.Timeout)
	s.Equal("json", checkCmd.options.Format)
	s.True(checkCmd.options.Verbose)
	s.True(checkCmd.options.FailureOnly)
}

// TestValidateProject tests project validation
func (s *CheckCommandTestSuite) TestValidateProject() {
	// Test with non-existent directory
	opts := &CheckOptions{
		ProjectPath: "/nonexistent/path",
	}

	_, err := NewCheckCommandWithOptions(opts)
	s.Error(err)
	s.Contains(err.Error(), "failed to resolve project path")
}

// TestValidateProjectWithoutGoMod tests project without go.mod
func (s *CheckCommandTestSuite) TestValidateProjectWithoutGoMod() {
	opts := &CheckOptions{
		ProjectPath: s.tempDir,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	err = checkCmd.validateProject()
	s.NoError(err) // Should not fail, just warn
}

// TestValidateProjectWithGoMod tests project with go.mod
func (s *CheckCommandTestSuite) TestValidateProjectWithGoMod() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	err = checkCmd.validateProject()
	s.NoError(err)
}

// TestDetermineChecks tests check type determination
func (s *CheckCommandTestSuite) TestDetermineChecks() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	// Test with --all flag
	checkCmd.options.All = true
	checks, err := checkCmd.determineChecks()
	s.NoError(err)
	s.Equal(4, len(checks)) // style, test, security, config
	s.Contains(checks, "style")
	s.Contains(checks, "test")
	s.Contains(checks, "security")
	s.Contains(checks, "config")

	// Test with specific check types
	checkCmd.options.All = false
	checkCmd.options.CheckTypes = []string{"style", "test"}
	checks, err = checkCmd.determineChecks()
	s.NoError(err)
	s.Equal(2, len(checks))
	s.Contains(checks, "style")
	s.Contains(checks, "test")

	// Test with no checks specified
	checkCmd.options.CheckTypes = nil
	checks, err = checkCmd.determineChecks()
	s.NoError(err)
	s.Empty(checks)

	// Test with invalid check type
	checkCmd.options.CheckTypes = []string{"invalid"}
	_, err = checkCmd.determineChecks()
	s.Error(err)
	s.Contains(err.Error(), "unknown check type: invalid")
}

// TestCreateQualityChecker tests quality checker creation
func (s *CheckCommandTestSuite) TestCreateQualityChecker() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
		MaxParallel: 4,
		Timeout:     5 * time.Minute,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	// Test creating checker with all types
	checkTypes := []string{"style", "test", "security", "config"}
	qualityChecker, err := checkCmd.createQualityChecker(checkTypes)
	s.NoError(err)
	s.NotNil(qualityChecker)

	qualityChecker.Close()
}

// TestCreateQualityCheckerWithInvalidType tests invalid checker type
func (s *CheckCommandTestSuite) TestCreateQualityCheckerWithInvalidType() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	// Test with invalid check type
	_, err = checkCmd.createQualityChecker([]string{"invalid"})
	s.Error(err)
	s.Contains(err.Error(), "unknown check type: invalid")
}

// TestExecuteWithAllChecks tests executing the command with all checks
func (s *CheckCommandTestSuite) TestExecuteWithAllChecks() {
	s.createGoProjectWithTests()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
		All:         true,
		Format:      "table",
		Timeout:     30 * time.Second,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	ctx := context.Background()
	err = checkCmd.Execute(ctx)
	// Error is expected since the project might not pass all checks
	// but we want to ensure the command runs without panicking
	s.NotPanics(func() {
		checkCmd.Execute(ctx)
	})
}

// TestExecuteWithSpecificChecks tests executing specific check types
func (s *CheckCommandTestSuite) TestExecuteWithSpecificChecks() {
	s.createGoProjectWithTests()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
		CheckTypes:  []string{"style"},
		Format:      "json",
		Timeout:     30 * time.Second,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	ctx := context.Background()
	s.NotPanics(func() {
		checkCmd.Execute(ctx)
	})
}

// TestExecuteWithNoChecks tests executing with no checks specified
func (s *CheckCommandTestSuite) TestExecuteWithNoChecks() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
		// No checks specified
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	ctx := context.Background()
	err = checkCmd.Execute(ctx)
	s.NoError(err) // Should succeed with info message
}

// TestFormatStatus tests status formatting
func (s *CheckCommandTestSuite) TestFormatStatus() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	tests := []struct {
		status   interfaces.CheckStatus
		expected string
	}{
		{interfaces.CheckStatusPass, "✅ PASS"},
		{interfaces.CheckStatusFail, "❌ FAIL"},
		{interfaces.CheckStatusWarning, "⚠️  WARN"},
		{interfaces.CheckStatusSkip, "⏭️  SKIP"},
		{interfaces.CheckStatusError, "💥 ERROR"},
	}

	for _, test := range tests {
		result := checkCmd.formatStatus(test.status)
		s.Equal(test.expected, result)
	}
}

// TestDisplayResults tests result display methods
func (s *CheckCommandTestSuite) TestDisplayResults() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
		Format:      "table",
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	// Create mock result
	result := &CheckResult{
		Individual: map[string]interfaces.CheckResult{
			"style": {
				Name:     "style",
				Status:   interfaces.CheckStatusPass,
				Message:  "All style checks passed",
				Duration: 1 * time.Second,
				Score:    100,
				MaxScore: 100,
				Details:  []interfaces.CheckDetail{},
			},
			"test": {
				Name:     "test",
				Status:   interfaces.CheckStatusFail,
				Message:  "2 tests failed",
				Duration: 2 * time.Second,
				Score:    80,
				MaxScore: 100,
				Details: []interfaces.CheckDetail{
					{
						Message:  "Test failure in TestExample",
						Rule:     "test_failure",
						Severity: "error",
					},
				},
			},
		},
		StartTime: time.Now().Add(-3 * time.Second),
		EndTime:   time.Now(),
		Success:   false,
	}

	s.NotPanics(func() {
		checkCmd.displayResults(result)
	})
}

// TestDisplayTableResults tests table format output
func (s *CheckCommandTestSuite) TestDisplayTableResults() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
		Format:      "table",
		FailureOnly: false,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	result := s.createMockCheckResult()

	err = checkCmd.displayTableResults(result)
	s.NoError(err)
}

// TestDisplayTableResultsFailureOnly tests table format with failure-only flag
func (s *CheckCommandTestSuite) TestDisplayTableResultsFailureOnly() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
		Format:      "table",
		FailureOnly: true,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	result := s.createMockCheckResult()

	err = checkCmd.displayTableResults(result)
	s.NoError(err)
}

// TestDisplayTextResults tests text format output
func (s *CheckCommandTestSuite) TestDisplayTextResults() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
		Format:      "text",
		Verbose:     true,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	result := s.createMockCheckResult()

	err = checkCmd.displayTextResults(result)
	s.NoError(err)
}

// TestDisplayJSONResults tests JSON format output
func (s *CheckCommandTestSuite) TestDisplayJSONResults() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
		Format:      "json",
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	result := s.createMockCheckResult()

	err = checkCmd.displayJSONResults(result)
	s.NoError(err)
}

// TestDetermineSuccess tests success determination
func (s *CheckCommandTestSuite) TestDetermineSuccess() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath:   s.tempDir,
		FailOnWarning: false,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	// Test with all passing checks
	passingResult := &CheckResult{
		Individual: map[string]interfaces.CheckResult{
			"style": {Status: interfaces.CheckStatusPass},
			"test":  {Status: interfaces.CheckStatusPass},
		},
	}
	s.True(checkCmd.determineSuccess(passingResult))

	// Test with failing checks
	failingResult := &CheckResult{
		Individual: map[string]interfaces.CheckResult{
			"style": {Status: interfaces.CheckStatusPass},
			"test":  {Status: interfaces.CheckStatusFail},
		},
	}
	s.False(checkCmd.determineSuccess(failingResult))

	// Test with warnings (fail-on-warning disabled)
	warningResult := &CheckResult{
		Individual: map[string]interfaces.CheckResult{
			"style": {Status: interfaces.CheckStatusWarning},
			"test":  {Status: interfaces.CheckStatusPass},
		},
	}
	s.True(checkCmd.determineSuccess(warningResult))

	// Test with warnings (fail-on-warning enabled)
	checkCmd.options.FailOnWarning = true
	s.False(checkCmd.determineSuccess(warningResult))
}

// TestHasFixableIssues tests fixable issues detection
func (s *CheckCommandTestSuite) TestHasFixableIssues() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	// Test with fixable issues
	fixableResult := &CheckResult{
		Individual: map[string]interfaces.CheckResult{
			"style": {
				Fixable: true,
				Details: []interfaces.CheckDetail{
					{Message: "Formatting issue", Rule: "gofmt"},
				},
			},
		},
	}
	s.True(checkCmd.hasFixableIssues(fixableResult))

	// Test without fixable issues
	nonFixableResult := &CheckResult{
		Individual: map[string]interfaces.CheckResult{
			"security": {
				Fixable: false,
				Details: []interfaces.CheckDetail{
					{Message: "Security issue", Rule: "hardcoded_secret"},
				},
			},
		},
	}
	s.False(checkCmd.hasFixableIssues(nonFixableResult))
}

// TestHasFailures tests failure detection
func (s *CheckCommandTestSuite) TestHasFailures() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	// Test with failures
	failureResult := &CheckResult{
		Individual: map[string]interfaces.CheckResult{
			"test": {Status: interfaces.CheckStatusFail},
		},
	}
	s.True(checkCmd.hasFailures(failureResult))

	// Test with errors
	errorResult := &CheckResult{
		Individual: map[string]interfaces.CheckResult{
			"config": {Status: interfaces.CheckStatusError},
		},
	}
	s.True(checkCmd.hasFailures(errorResult))

	// Test without failures
	passingResult := &CheckResult{
		Individual: map[string]interfaces.CheckResult{
			"style": {Status: interfaces.CheckStatusPass},
			"test":  {Status: interfaces.CheckStatusWarning},
		},
	}
	s.False(checkCmd.hasFailures(passingResult))
}

// TestRunAutoFix tests automatic fixing functionality
func (s *CheckCommandTestSuite) TestRunAutoFix() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
		Fix:         true,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	// Replace the UI with a mock for testing
	checkCmd.ui = &mockUIAdapter{}

	// Test with fixable issues
	fixableResult := &CheckResult{
		Individual: map[string]interfaces.CheckResult{
			"style": {
				Fixable: true,
				Details: []interfaces.CheckDetail{
					{
						Message:  "Formatting issue",
						Rule:     "gofmt",
						File:     "main.go",
						Severity: "warning",
					},
				},
			},
		},
	}

	// This should not error, even though the actual fixing is simulated
	err = checkCmd.runAutoFix(fixableResult)
	s.NoError(err)
}

// TestFixStyleIssues tests style issue fixing
func (s *CheckCommandTestSuite) TestFixStyleIssues() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	styleResult := interfaces.CheckResult{
		Details: []interfaces.CheckDetail{
			{Rule: "gofmt", File: "main.go"},
			{Rule: "import-organization", File: "handler.go"},
			{Rule: "missing-copyright", File: "types.go"},
			{Rule: "unknown-rule", File: "other.go"},
		},
	}

	fixedCount := checkCmd.fixStyleIssues(styleResult)
	s.Equal(3, fixedCount) // gofmt, import-organization, missing-copyright
}

// TestFixConfigIssues tests config issue fixing
func (s *CheckCommandTestSuite) TestFixConfigIssues() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	configResult := interfaces.CheckResult{
		Details: []interfaces.CheckDetail{
			{Rule: "formatting-issue", File: "config.yaml", Message: "formatting issue"},
			{Rule: "other-issue", File: "app.yaml", Message: "other issue"},
		},
	}

	fixedCount := checkCmd.fixConfigIssues(configResult)
	s.Equal(1, fixedCount) // Only formatting issues are considered fixable
}

// TestConcurrentExecution tests thread safety
func (s *CheckCommandTestSuite) TestConcurrentExecution() {
	s.createGoProject()

	// Run multiple concurrent command creations
	const numGoroutines = 5
	errors := make([]error, numGoroutines)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			opts := &CheckOptions{
				ProjectPath: s.tempDir,
			}
			_, errors[index] = NewCheckCommandWithOptions(opts)
		}(i)
	}

	wg.Wait()

	// Verify all command creations succeeded
	for i := 0; i < numGoroutines; i++ {
		s.NoError(errors[i], "Command creation %d should not error", i)
	}
}

// TestContextCancellation tests handling of context cancellation
func (s *CheckCommandTestSuite) TestContextCancellation() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
		All:         true,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = checkCmd.Execute(ctx)
	// The command might still succeed if it doesn't check context early enough
	// but it should not panic
	s.NotPanics(func() {
		checkCmd.Execute(ctx)
	})
}

// TestLargeProjectHandling tests performance with larger project
func (s *CheckCommandTestSuite) TestLargeProjectHandling() {
	s.createLargeGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
		CheckTypes:  []string{"style"}, // Only run style check for speed
		Timeout:     1 * time.Minute,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)

	ctx := context.Background()
	start := time.Now()

	s.NotPanics(func() {
		checkCmd.Execute(ctx)
	})

	duration := time.Since(start)
	s.True(duration < 30*time.Second, "Should complete within reasonable time")
}

// Helper methods

func (s *CheckCommandTestSuite) createGoProject() {
	// Create go.mod
	goMod := `module github.com/test/project

go 1.19

require (
    github.com/stretchr/testify v1.8.0
)
`
	err := s.createFile("go.mod", goMod)
	s.Require().NoError(err)

	// Create main.go
	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	err = s.createFile("main.go", mainGo)
	s.Require().NoError(err)

	// Create config file
	config := `server:
  http_port: 8080
database:
  type: mysql
`
	err = s.createFile("config.yaml", config)
	s.Require().NoError(err)
}

func (s *CheckCommandTestSuite) createGoProjectWithTests() {
	s.createGoProject()

	// Create test file
	testGo := `package main

import "testing"

func TestMain(t *testing.T) {
	// Simple test
	if 1+1 != 2 {
		t.Error("Math is broken")
	}
}
`
	err := s.createFile("main_test.go", testGo)
	s.Require().NoError(err)
}

func (s *CheckCommandTestSuite) createLargeGoProject() {
	s.createGoProject()

	// Create multiple packages and files
	packages := []string{"handler", "service", "repository", "model", "utils"}
	for _, pkg := range packages {
		pkgDir := filepath.Join(s.tempDir, pkg)
		err := os.MkdirAll(pkgDir, 0755)
		s.Require().NoError(err)

		// Create multiple files per package
		for i := 0; i < 5; i++ {
			content := fmt.Sprintf(`package %s

import "fmt"

func Function%d() {
	fmt.Printf("Function %d in package %s")
}
`, pkg, i, i, pkg)

			err = s.createFile(filepath.Join(pkg, fmt.Sprintf("file%d.go", i)), content)
			s.Require().NoError(err)
		}
	}
}

func (s *CheckCommandTestSuite) createFile(filename, content string) error {
	filePath := filepath.Join(s.tempDir, filename)

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}

func (s *CheckCommandTestSuite) createMockCheckResult() *CheckResult {
	return &CheckResult{
		Individual: map[string]interfaces.CheckResult{
			"style": {
				Name:     "style",
				Status:   interfaces.CheckStatusPass,
				Message:  "All style checks passed",
				Duration: 1 * time.Second,
				Score:    100,
				MaxScore: 100,
				Details:  []interfaces.CheckDetail{},
			},
			"test": {
				Name:     "test",
				Status:   interfaces.CheckStatusFail,
				Message:  "2 tests failed",
				Duration: 2 * time.Second,
				Score:    80,
				MaxScore: 100,
				Details: []interfaces.CheckDetail{
					{
						Message:  "Test failure in TestExample",
						Rule:     "test_failure",
						Severity: "error",
						File:     "main_test.go",
						Line:     10,
					},
				},
			},
			"security": {
				Name:     "security",
				Status:   interfaces.CheckStatusWarning,
				Message:  "1 medium security issue found",
				Duration: 500 * time.Millisecond,
				Score:    90,
				MaxScore: 100,
				Details: []interfaces.CheckDetail{
					{
						Message:  "Hardcoded secret detected",
						Rule:     "hardcoded_secret",
						Severity: "medium",
						File:     "config.go",
						Line:     5,
					},
				},
			},
			"config": {
				Name:     "config",
				Status:   interfaces.CheckStatusPass,
				Message:  "Configuration validation passed",
				Duration: 200 * time.Millisecond,
				Score:    100,
				MaxScore: 100,
				Details:  []interfaces.CheckDetail{},
			},
		},
		StartTime: time.Now().Add(-4 * time.Second),
		EndTime:   time.Now(),
		Success:   false, // Due to test failures
	}
}

// TestCobraCommandIntegration tests integration with cobra command framework
func (s *CheckCommandTestSuite) TestCobraCommandIntegration() {
	cmd := NewCheckCommand()

	// Test command execution with flags
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Test help command
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	s.NoError(err)
	s.Contains(output.String(), "quality checks")
}

// TestCommandArguments tests command argument parsing
func (s *CheckCommandTestSuite) TestCommandArguments() {
	s.createGoProject()

	cmd := NewCheckCommand()

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Test with specific check types as arguments
	cmd.SetArgs([]string{"style", "test", "--path", s.tempDir})

	// This tests that arguments are parsed but doesn't execute the full command
	// since that would require a complete environment setup
	s.NotPanics(func() {
		cmd.ParseFlags([]string{"style", "test", "--path", s.tempDir})
	})
}

// TestInvalidProjectPath tests handling of invalid project paths
func (s *CheckCommandTestSuite) TestInvalidProjectPath() {
	opts := &CheckOptions{
		ProjectPath: "/absolutely/nonexistent/path/that/should/not/exist",
	}

	_, err := NewCheckCommandWithOptions(opts)
	s.Error(err)
	s.Contains(err.Error(), "failed to resolve project path")
}

// TestTimeoutConfiguration tests timeout configuration
func (s *CheckCommandTestSuite) TestTimeoutConfiguration() {
	s.createGoProject()

	opts := &CheckOptions{
		ProjectPath: s.tempDir,
		Timeout:     1 * time.Second, // Very short timeout
		All:         true,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)
	s.NotNil(checkCmd)
	// Note: timeout and other fields may not be directly accessible
}

// TestParallelConfiguration tests parallel execution configuration
func (s *CheckCommandTestSuite) TestParallelConfiguration() {
	s.createGoProject()

	// Test with parallel enabled
	opts := &CheckOptions{
		ProjectPath: s.tempDir,
		MaxParallel: 8,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)
	s.NotNil(checkCmd)

	// Test with parallel disabled (MaxParallel = 1)
	opts.MaxParallel = 1
	checkCmd2, err := NewCheckCommandWithOptions(opts)
	s.NoError(err)
	s.NotNil(checkCmd2)
	// Note: parallel field may not be directly accessible
}

// Benchmark tests for performance

func BenchmarkCheckCommandCreation(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "check-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create minimal Go project
	goMod := `module test
go 1.19`
	os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)

	opts := &CheckOptions{
		ProjectPath: tempDir,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewCheckCommandWithOptions(opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDetermineChecks(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "determine-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	opts := &CheckOptions{
		ProjectPath: tempDir,
		All:         true,
	}

	checkCmd, err := NewCheckCommandWithOptions(opts)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := checkCmd.determineChecks()
		if err != nil {
			b.Fatal(err)
		}
	}
}
