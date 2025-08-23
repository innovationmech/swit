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
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// MockFileSystem provides a mock implementation of FileSystem for testing
type MockFileSystem struct {
	mock.Mock
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{}
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

// MockIndividualChecker provides a mock implementation of IndividualChecker
type MockIndividualChecker struct {
	mock.Mock
}

func NewMockIndividualChecker() *MockIndividualChecker {
	return &MockIndividualChecker{}
}

func (m *MockIndividualChecker) Check(ctx context.Context) (interfaces.CheckResult, error) {
	args := m.Called(ctx)
	return args.Get(0).(interfaces.CheckResult), args.Error(1)
}

func (m *MockIndividualChecker) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockIndividualChecker) Description() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockIndividualChecker) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockStyleChecker provides a mock implementation of StyleChecker
type MockStyleChecker struct {
	mock.Mock
}

func NewMockStyleChecker() *MockStyleChecker {
	return &MockStyleChecker{}
}

func (m *MockStyleChecker) CheckCodeStyle() interfaces.CheckResult {
	args := m.Called()
	return args.Get(0).(interfaces.CheckResult)
}

func (m *MockStyleChecker) RunTests() interfaces.TestResult {
	args := m.Called()
	return args.Get(0).(interfaces.TestResult)
}

func (m *MockStyleChecker) CheckSecurity() interfaces.SecurityResult {
	args := m.Called()
	return args.Get(0).(interfaces.SecurityResult)
}

func (m *MockStyleChecker) CheckCoverage() interfaces.CoverageResult {
	args := m.Called()
	return args.Get(0).(interfaces.CoverageResult)
}

func (m *MockStyleChecker) ValidateConfig() interfaces.ValidationResult {
	args := m.Called()
	return args.Get(0).(interfaces.ValidationResult)
}

func (m *MockStyleChecker) CheckInterfaces() interfaces.CheckResult {
	args := m.Called()
	return args.Get(0).(interfaces.CheckResult)
}

func (m *MockStyleChecker) CheckImports() interfaces.CheckResult {
	args := m.Called()
	return args.Get(0).(interfaces.CheckResult)
}

func (m *MockStyleChecker) CheckCopyright() interfaces.CheckResult {
	args := m.Called()
	return args.Get(0).(interfaces.CheckResult)
}

// MockTestRunner provides a mock implementation of TestRunner
type MockTestRunner struct {
	mock.Mock
}

func NewMockTestRunner() *MockTestRunner {
	return &MockTestRunner{}
}

func (m *MockTestRunner) RunTests(ctx context.Context, packages []string) (interfaces.TestResult, error) {
	args := m.Called(ctx, packages)
	return args.Get(0).(interfaces.TestResult), args.Error(1)
}

func (m *MockTestRunner) RunTestsWithCoverage(ctx context.Context, packages []string) (interfaces.TestResult, error) {
	args := m.Called(ctx, packages)
	return args.Get(0).(interfaces.TestResult), args.Error(1)
}

func (m *MockTestRunner) RunBenchmarks(ctx context.Context, packages []string) (BenchmarkResult, error) {
	args := m.Called(ctx, packages)
	return args.Get(0).(BenchmarkResult), args.Error(1)
}

func (m *MockTestRunner) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockSecurityScanner provides a mock implementation of SecurityScanner
type MockSecurityScanner struct {
	mock.Mock
}

func NewMockSecurityScanner() *MockSecurityScanner {
	return &MockSecurityScanner{}
}

func (m *MockSecurityScanner) ScanForVulnerabilities(ctx context.Context) (interfaces.SecurityResult, error) {
	args := m.Called(ctx)
	return args.Get(0).(interfaces.SecurityResult), args.Error(1)
}

func (m *MockSecurityScanner) ScanDependencies(ctx context.Context) (DependencySecurityResult, error) {
	args := m.Called(ctx)
	return args.Get(0).(DependencySecurityResult), args.Error(1)
}

func (m *MockSecurityScanner) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockConfigValidator provides a mock implementation of ConfigValidator
type MockConfigValidator struct {
	mock.Mock
}

func NewMockConfigValidator() *MockConfigValidator {
	return &MockConfigValidator{}
}

func (m *MockConfigValidator) ValidateYAML(ctx context.Context, filePath string) (interfaces.ValidationResult, error) {
	args := m.Called(ctx, filePath)
	return args.Get(0).(interfaces.ValidationResult), args.Error(1)
}

func (m *MockConfigValidator) ValidateJSON(ctx context.Context, filePath string) (interfaces.ValidationResult, error) {
	args := m.Called(ctx, filePath)
	return args.Get(0).(interfaces.ValidationResult), args.Error(1)
}

func (m *MockConfigValidator) ValidateProtobuf(ctx context.Context, filePath string) (interfaces.ValidationResult, error) {
	args := m.Called(ctx, filePath)
	return args.Get(0).(interfaces.ValidationResult), args.Error(1)
}

func (m *MockConfigValidator) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Test helper functions

// CreateTestCheckResult creates a test check result
func CreateTestCheckResult(name string, status interfaces.CheckStatus, message string) interfaces.CheckResult {
	return interfaces.CheckResult{
		Name:     name,
		Status:   status,
		Message:  message,
		Duration: 1 * time.Second,
		Score:    80,
		MaxScore: 100,
		Fixable:  true,
		Details: []interfaces.CheckDetail{
			{
				File:     "test.go",
				Line:     10,
				Column:   5,
				Message:  "Test detail message",
				Rule:     "test-rule",
				Severity: "warning",
			},
		},
	}
}

// CreateTestTestResult creates a test result for testing
func CreateTestTestResult(totalTests, passedTests, failedTests int) interfaces.TestResult {
	return interfaces.TestResult{
		TotalTests:   totalTests,
		PassedTests:  passedTests,
		FailedTests:  failedTests,
		SkippedTests: 0,
		Coverage:     85.5,
		Duration:     5 * time.Second,
		Failures: []interfaces.TestFailure{
			{
				Test:    "TestExample",
				Package: "github.com/example/package",
				Message: "assertion failed",
				Output:  "Expected 'true' but got 'false'",
				File:    "example_test.go",
			},
		},
	}
}

// CreateTestSecurityResult creates a security result for testing
func CreateTestSecurityResult(vulnerabilities int) interfaces.SecurityResult {
	var issues []interfaces.SecurityIssue
	if vulnerabilities > 0 {
		issues = make([]interfaces.SecurityIssue, vulnerabilities)
		for i := 0; i < vulnerabilities; i++ {
			issues[i] = interfaces.SecurityIssue{
				ID:          fmt.Sprintf("CVE-2023-%04d", 1234+i),
				Severity:    "medium",
				Title:       "Example vulnerability",
				Description: "This is an example security vulnerability",
				File:        "example.go",
				Line:        10 + i,
			}
		}
	}

	return interfaces.SecurityResult{
		Issues:   issues,
		Severity: "medium",
		Score:    80,
		MaxScore: 100,
		Scanned:  5,
		Duration: 3 * time.Second,
	}
}

// CreateTestValidationResult creates a validation result for testing
func CreateTestValidationResult(isValid bool, errorCount int) interfaces.ValidationResult {
	result := interfaces.ValidationResult{
		Valid:    isValid,
		Duration: 2 * time.Second,
	}

	if !isValid {
		result.Errors = make([]interfaces.ValidationError, errorCount)
		for i := 0; i < errorCount; i++ {
			result.Errors[i] = interfaces.ValidationError{
				Field:   fmt.Sprintf("field_%d", i),
				Message: fmt.Sprintf("validation error %d", i),
				Value:   fmt.Sprintf("invalid_value_%d", i),
				Rule:    "required",
				Code:    "REQUIRED_FIELD",
			}
		}
	}

	return result
}

// SlowChecker is a test checker that simulates slow operations
type SlowChecker struct {
	delay  time.Duration
	result interfaces.CheckResult
	err    error
}

func NewSlowChecker(delay time.Duration, result interfaces.CheckResult, err error) *SlowChecker {
	return &SlowChecker{
		delay:  delay,
		result: result,
		err:    err,
	}
}

func (s *SlowChecker) Check(ctx context.Context) (interfaces.CheckResult, error) {
	select {
	case <-time.After(s.delay):
		return s.result, s.err
	case <-ctx.Done():
		return interfaces.CheckResult{}, ctx.Err()
	}
}

func (s *SlowChecker) Name() string {
	return "slow-checker"
}

func (s *SlowChecker) Description() string {
	return "A slow checker for testing timeout scenarios"
}

func (s *SlowChecker) Close() error {
	return nil
}

// ErrorChecker is a test checker that always returns errors
type ErrorChecker struct {
	err error
}

func NewErrorChecker(err error) *ErrorChecker {
	return &ErrorChecker{err: err}
}

func (e *ErrorChecker) Check(ctx context.Context) (interfaces.CheckResult, error) {
	return interfaces.CheckResult{
		Name:    "error-checker",
		Status:  interfaces.CheckStatusError,
		Message: e.err.Error(),
	}, e.err
}

func (e *ErrorChecker) Name() string {
	return "error-checker"
}

func (e *ErrorChecker) Description() string {
	return "A checker that always returns errors"
}

func (e *ErrorChecker) Close() error {
	return nil
}

// Additional result types for specialized checkers

// Note: BenchmarkResult and BenchmarkInfo are now defined in test.go

// SecurityVulnerability represents a security vulnerability
type SecurityVulnerability struct {
	ID           string    `json:"id"`
	Severity     string    `json:"severity"`
	Package      string    `json:"package"`
	Version      string    `json:"version"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	FixedVersion string    `json:"fixed_version,omitempty"`
	URLs         []string  `json:"urls,omitempty"`
	CVSS         float64   `json:"cvss,omitempty"`
	PublishedAt  time.Time `json:"published_at,omitempty"`
}

// DependencySecurityResult represents security scan results for dependencies
type DependencySecurityResult struct {
	TotalDependencies      int                     `json:"total_dependencies"`
	VulnerableDependencies int                     `json:"vulnerable_dependencies"`
	Vulnerabilities        []SecurityVulnerability `json:"vulnerabilities"`
	Duration               time.Duration           `json:"duration"`
	ScanTime               time.Time               `json:"scan_time"`
}
