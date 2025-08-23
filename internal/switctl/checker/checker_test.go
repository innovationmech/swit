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
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// MockLogger implements a simple logger for testing.
type MockLogger struct {
	messages []LogMessage
}

// LogMessage represents a logged message.
type LogMessage struct {
	Level  string
	Msg    string
	Fields map[string]any
}

func (m *MockLogger) Debug(msg string, fields ...any) {
	m.messages = append(m.messages, LogMessage{Level: "DEBUG", Msg: msg, Fields: m.fieldsToMap(fields)})
}

func (m *MockLogger) Info(msg string, fields ...any) {
	m.messages = append(m.messages, LogMessage{Level: "INFO", Msg: msg, Fields: m.fieldsToMap(fields)})
}

func (m *MockLogger) Warn(msg string, fields ...any) {
	m.messages = append(m.messages, LogMessage{Level: "WARN", Msg: msg, Fields: m.fieldsToMap(fields)})
}

func (m *MockLogger) Error(msg string, fields ...any) {
	m.messages = append(m.messages, LogMessage{Level: "ERROR", Msg: msg, Fields: m.fieldsToMap(fields)})
}

func (m *MockLogger) Fatal(msg string, fields ...any) {
	m.messages = append(m.messages, LogMessage{Level: "FATAL", Msg: msg, Fields: m.fieldsToMap(fields)})
}

func (m *MockLogger) fieldsToMap(fields []any) map[string]any {
	result := make(map[string]any)
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok && i+1 < len(fields) {
			result[key] = fields[i+1]
		}
	}
	return result
}

func (m *MockLogger) GetMessages(level string) []LogMessage {
	var filtered []LogMessage
	for _, msg := range m.messages {
		if msg.Level == level {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// CheckerTestSuite is the test suite for the quality checking system.
type CheckerTestSuite struct {
	suite.Suite
	tempDir   string
	logger    *MockLogger
	manager   *Manager
	composite *CompositeChecker
}

// SetupSuite runs before the test suite starts.
func (suite *CheckerTestSuite) SetupSuite() {
	// Create temporary directory for tests
	tempDir, err := os.MkdirTemp("", "switctl-checker-test-*")
	require.NoError(suite.T(), err)

	suite.tempDir = tempDir
	suite.logger = &MockLogger{messages: make([]LogMessage, 0)}
}

// TearDownSuite runs after the test suite finishes.
func (suite *CheckerTestSuite) TearDownSuite() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// SetupTest runs before each test method.
func (suite *CheckerTestSuite) SetupTest() {
	suite.logger.messages = nil // Reset logger messages
	suite.manager = NewManager(suite.logger)
	suite.composite = NewCompositeChecker(suite.tempDir, suite.logger)
}

// TestManagerCreation tests creating a quality checker manager.
func (suite *CheckerTestSuite) TestManagerCreation() {
	manager := NewManager(suite.logger)
	assert.NotNil(suite.T(), manager)
	assert.Equal(suite.T(), 5*time.Minute, manager.timeout)
	assert.True(suite.T(), manager.parallel)
}

// TestManagerConfiguration tests manager configuration methods.
func (suite *CheckerTestSuite) TestManagerConfiguration() {
	manager := NewManager(suite.logger)

	// Test timeout configuration
	manager.SetTimeout(2 * time.Minute)
	assert.Equal(suite.T(), 2*time.Minute, manager.timeout)

	// Test parallel configuration
	manager.SetParallel(false)
	assert.False(suite.T(), manager.parallel)
}

// TestCheckerRegistration tests registering checkers with the manager.
func (suite *CheckerTestSuite) TestCheckerRegistration() {
	manager := NewManager(suite.logger)

	// Register composite checker
	manager.RegisterChecker(suite.composite)
	assert.Len(suite.T(), manager.checkers, 1)
}

// TestRunAllChecksWithNoCheckers tests running checks with no registered checkers.
func (suite *CheckerTestSuite) TestRunAllChecksWithNoCheckers() {
	manager := NewManager(suite.logger)

	ctx := context.Background()
	report, err := manager.RunAllChecks(ctx)

	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), report)
	assert.Equal(suite.T(), interfaces.CheckStatusSkip, report.Overall.Status)
	assert.Contains(suite.T(), report.Overall.Message, "No checkers registered")
}

// TestRunAllChecksWithTimeout tests running checks with timeout.
func (suite *CheckerTestSuite) TestRunAllChecksWithTimeout() {
	manager := NewManager(suite.logger)
	manager.SetTimeout(1 * time.Nanosecond) // Very short timeout
	manager.RegisterChecker(suite.composite)

	ctx := context.Background()
	report, err := manager.RunAllChecks(ctx)

	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), report)
	// Should have some timeout-related results
}

// TestCompositeCheckerCreation tests creating a composite checker.
func (suite *CheckerTestSuite) TestCompositeCheckerCreation() {
	composite := NewCompositeChecker(suite.tempDir, suite.logger)

	assert.NotNil(suite.T(), composite)
	assert.NotNil(suite.T(), composite.GetStyleChecker())
	assert.NotNil(suite.T(), composite.GetTestRunner())
	assert.NotNil(suite.T(), composite.GetSecurityChecker())
	assert.NotNil(suite.T(), composite.GetConfigChecker())
}

// TestStyleCheckerBasic tests basic style checker functionality.
func (suite *CheckerTestSuite) TestStyleCheckerBasic() {
	// Create a test Go file
	suite.createTestGoFile("test.go", `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
`)

	options := StyleCheckerOptions{
		EnableGofmt:          true,
		EnableGolangciLint:   true,
		EnableImportCheck:    true,
		EnableCopyrightCheck: true,
	}
	styleChecker := NewStyleChecker(suite.tempDir, options)
	result, err := styleChecker.Check(context.Background())
	require.NoError(suite.T(), err)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "code_style", result.Name)
	assert.NotZero(suite.T(), result.Duration)
}

// TestStyleCheckerImportCheck tests import organization checking.
func (suite *CheckerTestSuite) TestStyleCheckerImportCheck() {
	// Create a test Go file with poor import organization
	suite.createTestGoFile("bad_imports.go", `package main

import (
    "github.com/some/package"
    "fmt"
    . "os"
)

func main() {
    fmt.Println("Hello")
}
`)

	options := StyleCheckerOptions{
		EnableGofmt:          true,
		EnableGolangciLint:   true,
		EnableImportCheck:    true,
		EnableCopyrightCheck: true,
	}
	styleChecker := NewStyleChecker(suite.tempDir, options)
	result, err := styleChecker.Check(context.Background())
	require.NoError(suite.T(), err)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "code_style", result.Name)
	// Should detect dot import issue
	assert.True(suite.T(), len(result.Details) > 0)
}

// TestStyleCheckerCopyrightCheck tests copyright header checking.
func (suite *CheckerTestSuite) TestStyleCheckerCopyrightCheck() {
	// Create a test Go file without copyright header
	suite.createTestGoFile("no_copyright.go", `package main

func main() {
}
`)

	options := StyleCheckerOptions{
		EnableGofmt:          true,
		EnableGolangciLint:   true,
		EnableImportCheck:    true,
		EnableCopyrightCheck: true,
	}
	styleChecker := NewStyleChecker(suite.tempDir, options)
	result, err := styleChecker.Check(context.Background())
	require.NoError(suite.T(), err)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "code_style", result.Name)
	// Should detect missing copyright
	if result.Status != interfaces.CheckStatusSkip {
		assert.True(suite.T(), len(result.Details) > 0)
	}
}

// TestTestCheckerBasic tests basic test checker functionality.
func (suite *CheckerTestSuite) TestTestCheckerBasic() {
	// Create a test file
	suite.createTestGoFile("math_test.go", `package main

import "testing"

func TestAdd(t *testing.T) {
    result := 2 + 2
    if result != 4 {
        t.Errorf("Expected 4, got %d", result)
    }
}
`)

	testOptions := TestRunnerOptions{
		EnableCoverage:      true,
		EnableRaceDetection: true,
		EnableBenchmarks:    true,
		CoverageThreshold:   80.0,
		Verbose:             true,
	}
	testChecker := NewTestRunner(suite.tempDir, testOptions)
	result, err := testChecker.Check(context.Background())
	require.NoError(suite.T(), err)

	assert.NotNil(suite.T(), result)
	assert.NotZero(suite.T(), result.Duration)
}

// TestTestCheckerCoverage tests coverage analysis.
func (suite *CheckerTestSuite) TestTestCheckerCoverage() {
	// Create main file and test file
	suite.createTestGoFile("math.go", `package main

func Add(a, b int) int {
    return a + b
}

func main() {}
`)

	suite.createTestGoFile("math_test.go", `package main

import "testing"

func TestAdd(t *testing.T) {
    result := Add(2, 2)
    if result != 4 {
        t.Errorf("Expected 4, got %d", result)
    }
}
`)

	testOptions := TestRunnerOptions{
		EnableCoverage:      true,
		EnableRaceDetection: true,
		EnableBenchmarks:    true,
		CoverageThreshold:   80.0,
		Verbose:             true,
	}
	testChecker := NewTestRunner(suite.tempDir, testOptions)
	result, err := testChecker.Check(context.Background())
	require.NoError(suite.T(), err)

	assert.NotNil(suite.T(), result)
	assert.NotZero(suite.T(), result.Duration)
}

// TestSecurityCheckerBasic tests basic security checker functionality.
func (suite *CheckerTestSuite) TestSecurityCheckerBasic() {
	// Create a test Go file with potential security issues
	suite.createTestGoFile("insecure.go", `package main

import "fmt"

func main() {
    password := "hardcoded123"
    fmt.Println(password)
}
`)

	securityChecker := NewSecurityChecker(suite.tempDir, suite.logger)
	result, err := securityChecker.Check(context.Background())
	require.NoError(suite.T(), err)

	assert.NotNil(suite.T(), result)
	assert.NotZero(suite.T(), result.Duration)
	// Should detect hardcoded secret
	assert.True(suite.T(), len(result.Details) > 0)
}

// TestConfigCheckerBasic tests basic config checker functionality.
func (suite *CheckerTestSuite) TestConfigCheckerBasic() {
	// Create a test YAML config file
	suite.createTestFile("config.yaml", `
server:
  http_port: 8080
  grpc_port: 9090

database:
  type: mysql
  host: localhost
  port: 3306
`)

	configChecker := NewConfigChecker(suite.tempDir, suite.logger)
	result, err := configChecker.Check(context.Background())
	require.NoError(suite.T(), err)

	assert.NotNil(suite.T(), result)
	assert.NotZero(suite.T(), result.Duration)
	assert.Equal(suite.T(), interfaces.CheckStatusPass, result.Status)
}

// TestConfigCheckerInvalidYAML tests config checker with invalid YAML.
func (suite *CheckerTestSuite) TestConfigCheckerInvalidYAML() {
	// Create an invalid YAML file
	suite.createTestFile("invalid.yaml", `
server:
  http_port: 8080
  invalid_yaml: [unclosed_array
`)

	configChecker := NewConfigChecker(suite.tempDir, suite.logger)
	result, err := configChecker.Check(context.Background())
	require.NoError(suite.T(), err)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), interfaces.CheckStatusFail, result.Status)
	assert.True(suite.T(), len(result.Details) > 0)
}

// TestConfigCheckerGoModule tests Go module validation.
func (suite *CheckerTestSuite) TestConfigCheckerGoModule() {
	// Create a go.mod file
	suite.createTestFile("go.mod", `module github.com/test/project

go 1.19

require (
    github.com/stretchr/testify v1.8.0
)
`)

	configChecker := NewConfigChecker(suite.tempDir, suite.logger)
	result, err := configChecker.Check(context.Background())
	require.NoError(suite.T(), err)

	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), interfaces.CheckStatusPass, result.Status)
}

// TestQualityReport tests quality report functionality.
func (suite *CheckerTestSuite) TestQualityReport() {
	report := &QualityReport{
		Overall: QualityOverall{
			Status:  interfaces.CheckStatusPass,
			Message: "Tests passed",
		},
		StyleCheck: interfaces.CheckResult{
			Name:   "style",
			Status: interfaces.CheckStatusPass,
		},
		TestResult: interfaces.TestResult{
			TotalTests:  10,
			PassedTests: 10,
		},
		SecurityResult: interfaces.SecurityResult{
			Issues:   []interfaces.SecurityIssue{},
			Severity: "low",
		},
		ConfigResult: interfaces.ValidationResult{
			Valid: true,
		},
	}

	assert.NotNil(suite.T(), report)
	assert.Equal(suite.T(), interfaces.CheckStatusPass, report.Overall.Status)
	assert.Equal(suite.T(), "Tests passed", report.Overall.Message)
	assert.Equal(suite.T(), "style", report.StyleCheck.Name)
}

// TestCheckerOptions tests checker options.
func (suite *CheckerTestSuite) TestCheckerOptions() {
	options := CheckerOptions{
		Timeout:       30 * time.Second,
		MaxGoroutines: 4,
		EnableCache:   true,
	}

	assert.Equal(suite.T(), 30*time.Second, options.Timeout)
	assert.Equal(suite.T(), 4, options.MaxGoroutines)
	assert.True(suite.T(), options.EnableCache)
}

// Helper methods

// createTestGoFile creates a test Go file with the given content.
func (suite *CheckerTestSuite) createTestGoFile(filename, content string) {
	suite.createTestFile(filename, content)
}

// createTestFile creates a test file with the given content.
func (suite *CheckerTestSuite) createTestFile(filename, content string) {
	filePath := filepath.Join(suite.tempDir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(suite.T(), err)
}

// createTestDirectory creates a test directory.
func (suite *CheckerTestSuite) createTestDirectory(dirname string) {
	dirPath := filepath.Join(suite.tempDir, dirname)
	err := os.MkdirAll(dirPath, 0755)
	require.NoError(suite.T(), err)
}

// TestSuite runs the checker test suite.
func TestCheckerSuite(t *testing.T) {
	suite.Run(t, new(CheckerTestSuite))
}

// Benchmark tests

// BenchmarkStyleChecker benchmarks style checking performance.
func BenchmarkStyleChecker(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "switctl-bench-*")
	require.NoError(b, err)
	defer os.RemoveAll(tempDir)

	// Create test Go files
	for i := 0; i < 10; i++ {
		content := `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
`
		filePath := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(b, err)
	}

	options := StyleCheckerOptions{
		EnableGofmt:          true,
		EnableGolangciLint:   true,
		EnableImportCheck:    true,
		EnableCopyrightCheck: true,
	}
	styleChecker := NewStyleChecker(tempDir, options)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := styleChecker.Check(context.Background())
		require.NoError(b, err)
		_ = result
	}
}

// BenchmarkSecurityChecker benchmarks security checking performance.
func BenchmarkSecurityChecker(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "switctl-bench-*")
	require.NoError(b, err)
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{}

	// Create test Go files with potential security issues
	for i := 0; i < 10; i++ {
		content := `package main

import "fmt"

func main() {
    password := "secret123"
    fmt.Println(password)
}
`
		filePath := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(b, err)
	}

	securityChecker := NewSecurityChecker(tempDir, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := securityChecker.Check(context.Background())
		require.NoError(b, err)
		_ = result
	}
}
