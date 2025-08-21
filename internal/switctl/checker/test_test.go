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

package checker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// TestRunnerTestSuite tests the test runner functionality
type TestRunnerTestSuite struct {
	suite.Suite
	tempDir    string
	testRunner *TestRunner
}

func TestTestRunnerTestSuite(t *testing.T) {
	suite.Run(t, new(TestRunnerTestSuite))
}

func (s *TestRunnerTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "test-runner-test-*")
	s.Require().NoError(err)

	s.testRunner = NewTestRunner(s.tempDir, TestRunnerOptions{
		EnableRaceDetection: true,
		EnableCoverage:      true,
		CoverageThreshold:   80.0,
		TestTimeout:         30 * time.Second,
		Verbose:             true,
	})
}

func (s *TestRunnerTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
	if s.testRunner != nil {
		s.testRunner.Close()
	}
}

// TestTestRunnerInterface tests that TestRunner implements IndividualChecker
func (s *TestRunnerTestSuite) TestTestRunnerInterface() {
	var _ IndividualChecker = s.testRunner
	s.NotNil(s.testRunner)
}

// TestNewTestRunner tests the constructor
func (s *TestRunnerTestSuite) TestNewTestRunner() {
	runner := NewTestRunner("/test/path", TestRunnerOptions{
		EnableCoverage: true,
		Verbose:        true,
	})

	s.NotNil(runner)
	s.Equal("/test/path", runner.projectPath)
	s.True(runner.options.EnableCoverage)
	s.True(runner.options.Verbose)
	s.Equal("test", runner.Name())
	s.Contains(runner.Description(), "test")
}

// TestRunTestsWithPassingTests tests running tests that all pass
func (s *TestRunnerTestSuite) TestRunTestsWithPassingTests() {
	// Create a simple Go package with passing tests
	err := s.createTestPackage("passing", map[string]string{
		"main.go": `package passing

func Add(a, b int) int {
	return a + b
}

func Multiply(a, b int) int {
	return a * b
}
`,
		"main_test.go": `package passing

import "testing"

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("Add(2, 3) = %d; want 5", result)
	}
}

func TestMultiply(t *testing.T) {
	result := Multiply(3, 4)
	if result != 12 {
		t.Errorf("Multiply(3, 4) = %d; want 12", result)
	}
}

func BenchmarkAdd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Add(i, i+1)
	}
}
`,
	})
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.testRunner.Check(ctx)

	s.NoError(err)
	s.Equal("test", result.Name)
	s.Equal(interfaces.CheckStatusPass, result.Status)
	s.Contains(result.Message, "passed")
	s.True(result.Duration > 0)
	s.True(result.Score >= 80)
}

// TestRunTestsWithFailingTests tests running tests that fail
func (s *TestRunnerTestSuite) TestRunTestsWithFailingTests() {
	// Create a package with failing tests
	err := s.createTestPackage("failing", map[string]string{
		"main.go": `package failing

func Divide(a, b int) int {
	return a / b // Will fail for divide by zero test
}
`,
		"main_test.go": `package failing

import "testing"

func TestDivide(t *testing.T) {
	result := Divide(10, 2)
	if result != 5 {
		t.Errorf("Divide(10, 2) = %d; want 5", result)
	}
}

func TestDivideByZero(t *testing.T) {
	// This test will fail
	result := Divide(10, 0)
	if result != 0 {
		t.Errorf("Expected divide by zero to return 0, got %d", result)
	}
}
`,
	})
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.testRunner.Check(ctx)

	s.NoError(err)
	s.Equal("test", result.Name)
	s.Equal(interfaces.CheckStatusFail, result.Status)
	s.Contains(result.Message, "failed")
	s.NotEmpty(result.Details)
	s.True(result.Score < 100)
}

// TestRunTestsWithCoverage tests coverage analysis
func (s *TestRunnerTestSuite) TestRunTestsWithCoverage() {
	// Create a package with partial test coverage
	err := s.createTestPackage("coverage", map[string]string{
		"math.go": `package coverage

func Add(a, b int) int {
	return a + b
}

func Subtract(a, b int) int {
	return a - b
}

func Multiply(a, b int) int {
	return a * b
}

// This function is not tested
func Divide(a, b int) int {
	if b == 0 {
		return 0
	}
	return a / b
}
`,
		"math_test.go": `package coverage

import "testing"

func TestAdd(t *testing.T) {
	if Add(2, 3) != 5 {
		t.Error("Add failed")
	}
}

func TestSubtract(t *testing.T) {
	if Subtract(5, 3) != 2 {
		t.Error("Subtract failed")
	}
}

func TestMultiply(t *testing.T) {
	if Multiply(3, 4) != 12 {
		t.Error("Multiply failed")
	}
}
`,
	})
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.testRunner.Check(ctx)

	s.NoError(err)
	s.Equal("test", result.Name)

	// Should pass tests but may have coverage warnings
	s.True(result.Status == interfaces.CheckStatusPass || result.Status == interfaces.CheckStatusWarning)
}

// TestRunTestsWithBenchmarks tests benchmark execution
func (s *TestRunnerTestSuite) TestRunTestsWithBenchmarks() {
	err := s.createTestPackage("benchmarks", map[string]string{
		"compute.go": `package benchmarks

import "time"

func FastOperation() int {
	return 42
}

func SlowOperation() int {
	time.Sleep(1 * time.Millisecond)
	return 42
}
`,
		"compute_test.go": `package benchmarks

import "testing"

func TestFastOperation(t *testing.T) {
	if FastOperation() != 42 {
		t.Error("FastOperation failed")
	}
}

func BenchmarkFastOperation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FastOperation()
	}
}

func BenchmarkSlowOperation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SlowOperation()
	}
}
`,
	})
	s.Require().NoError(err)

	// Test with benchmarks enabled
	runner := NewTestRunner(s.tempDir, TestRunnerOptions{
		EnableBenchmarks: true,
		BenchmarkTime:    1 * time.Second,
	})
	defer runner.Close()

	ctx := context.Background()
	benchResults, err := runner.RunBenchmarks(ctx, []string{"./benchmarks"})

	s.NoError(err)
	s.NotEmpty(benchResults.Benchmarks)
	s.True(benchResults.TotalRuns > 0)
}

// TestRunTestsWithRaceDetection tests race detection
func (s *TestRunnerTestSuite) TestRunTestsWithRaceDetection() {
	// Create package with potential race condition
	err := s.createTestPackage("race", map[string]string{
		"counter.go": `package race

var GlobalCounter int

func IncrementCounter() {
	GlobalCounter++
}

func GetCounter() int {
	return GlobalCounter
}
`,
		"counter_test.go": `package race

import (
	"sync"
	"testing"
)

func TestConcurrentIncrement(t *testing.T) {
	GlobalCounter = 0
	
	var wg sync.WaitGroup
	numGoroutines := 10
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			IncrementCounter()
		}()
	}
	
	wg.Wait()
	
	// This might detect race condition with -race flag
	if GetCounter() != numGoroutines {
		t.Logf("Counter value: %d, expected: %d", GetCounter(), numGoroutines)
	}
}
`,
	})
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.testRunner.Check(ctx)

	s.NoError(err)
	s.Equal("test", result.Name)
	// Race detection might flag this as a warning or pass depending on the race detector
}

// TestRunTestsWithTimeout tests handling of test timeouts
func (s *TestRunnerTestSuite) TestRunTestsWithTimeout() {
	// Create a test that takes too long
	err := s.createTestPackage("timeout", map[string]string{
		"slow.go": `package timeout

import "time"

func SlowFunction() {
	time.Sleep(5 * time.Second)
}
`,
		"slow_test.go": `package timeout

import "testing"

func TestSlowFunction(t *testing.T) {
	SlowFunction()
}
`,
	})
	s.Require().NoError(err)

	// Create runner with very short timeout
	shortTimeoutRunner := NewTestRunner(s.tempDir, TestRunnerOptions{
		TestTimeout: 100 * time.Millisecond,
	})
	defer shortTimeoutRunner.Close()

	ctx := context.Background()
	result, err := shortTimeoutRunner.Check(ctx)

	// Should either error or detect timeout
	if err != nil {
		s.Contains(err.Error(), "timeout")
	} else if result.Status == interfaces.CheckStatusError {
		s.Contains(result.Message, "timeout")
	} else {
		s.Equal(interfaces.CheckStatusFail, result.Status)
	}
}

// TestRunSpecificPackages tests running tests for specific packages
func (s *TestRunnerTestSuite) TestRunSpecificPackages() {
	// Create multiple packages
	err := s.createTestPackage("pkg1", map[string]string{
		"main.go":      `package pkg1; func Func1() int { return 1 }`,
		"main_test.go": `package pkg1; import "testing"; func TestFunc1(t *testing.T) { if Func1() != 1 { t.Error("failed") } }`,
	})
	s.Require().NoError(err)

	err = s.createTestPackage("pkg2", map[string]string{
		"main.go":      `package pkg2; func Func2() int { return 2 }`,
		"main_test.go": `package pkg2; import "testing"; func TestFunc2(t *testing.T) { if Func2() != 2 { t.Error("failed") } }`,
	})
	s.Require().NoError(err)

	ctx := context.Background()

	// Test specific package
	packages := []string{"./pkg1"}
	result, err := s.testRunner.RunTests(ctx, packages)

	s.NoError(err)
	s.Equal(1, result.TotalTests) // Only tests from pkg1
	s.Equal(1, result.PassedTests)
}

// TestRunTestsWithCoverageThreshold tests coverage threshold checking
func (s *TestRunnerTestSuite) TestRunTestsWithCoverageThreshold() {
	// Create package with low test coverage
	err := s.createTestPackage("lowcoverage", map[string]string{
		"functions.go": `package lowcoverage

func TestedFunction() int {
	return 42
}

func UntestedFunction1() int {
	return 1
}

func UntestedFunction2() int {
	return 2
}

func UntestedFunction3() int {
	return 3
}

func UntestedFunction4() int {
	return 4
}
`,
		"functions_test.go": `package lowcoverage

import "testing"

func TestTestedFunction(t *testing.T) {
	if TestedFunction() != 42 {
		t.Error("failed")
	}
}
`,
	})
	s.Require().NoError(err)

	// Set high coverage threshold
	highThresholdRunner := NewTestRunner(s.tempDir, TestRunnerOptions{
		EnableCoverage:    true,
		CoverageThreshold: 90.0,
	})
	defer highThresholdRunner.Close()

	ctx := context.Background()
	result, err := highThresholdRunner.Check(ctx)

	s.NoError(err)
	// Should pass tests but fail coverage threshold
	s.True(result.Status == interfaces.CheckStatusWarning || result.Status == interfaces.CheckStatusFail)
	s.Contains(result.Message, "coverage")
}

// TestEmptyProject tests handling of project with no tests
func (s *TestRunnerTestSuite) TestEmptyProject() {
	// Create package with no tests
	err := s.createTestPackage("notests", map[string]string{
		"main.go": `package notests

func NoTestsHere() int {
	return 42
}
`,
	})
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.testRunner.Check(ctx)

	s.NoError(err)
	s.Equal("test", result.Name)
	s.Equal(interfaces.CheckStatusSkip, result.Status)
	s.Contains(result.Message, "no tests")
}

// TestNonExistentProject tests handling of non-existent project
func (s *TestRunnerTestSuite) TestNonExistentProject() {
	nonExistentRunner := NewTestRunner("/nonexistent/path", TestRunnerOptions{})
	defer nonExistentRunner.Close()

	ctx := context.Background()
	result, err := nonExistentRunner.Check(ctx)

	s.NoError(err)
	s.Equal(interfaces.CheckStatusSkip, result.Status)
	s.Contains(result.Message, "not exist")
}

// TestContextCancellation tests handling of context cancellation
func (s *TestRunnerTestSuite) TestContextCancellation() {
	// Create a simple test package
	err := s.createTestPackage("cancel", map[string]string{
		"main.go":      `package cancel; func Test() int { return 1 }`,
		"main_test.go": `package cancel; import "testing"; func TestTest(t *testing.T) { if Test() != 1 { t.Error("failed") } }`,
	})
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = s.testRunner.Check(ctx)
	s.Error(err)
	s.Equal(context.Canceled, err)
}

// TestConcurrentExecution tests thread safety
func (s *TestRunnerTestSuite) TestConcurrentExecution() {
	// Create test package
	err := s.createTestPackage("concurrent", map[string]string{
		"main.go":      `package concurrent; func Func() int { return 42 }`,
		"main_test.go": `package concurrent; import "testing"; func TestFunc(t *testing.T) { if Func() != 42 { t.Error("failed") } }`,
	})
	s.Require().NoError(err)

	// Run multiple concurrent checks
	const numGoroutines = 5
	results := make([]interfaces.CheckResult, numGoroutines)
	errors := make([]error, numGoroutines)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ctx := context.Background()
			results[index], errors[index] = s.testRunner.Check(ctx)
		}(i)
	}

	wg.Wait()

	// Verify all checks completed successfully
	for i := 0; i < numGoroutines; i++ {
		s.NoError(errors[i], "Check %d should not error", i)
		s.Equal("test", results[i].Name)
	}
}

// TestResultDetailStructure tests the structure of test failure details
func (s *TestRunnerTestSuite) TestResultDetailStructure() {
	// Create package with detailed test failures
	err := s.createTestPackage("details", map[string]string{
		"calculator.go": `package details

func Add(a, b int) int {
	return a + b
}

func Subtract(a, b int) int {
	return a - b
}
`,
		"calculator_test.go": `package details

import "testing"

func TestAdd(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positive", 2, 3, 5},
		{"negative", -1, -2, -3},
		{"zero", 0, 5, 5},
		{"failing", 2, 3, 6}, // This will fail
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Add(test.a, test.b)
			if result != test.expected {
				t.Errorf("Add(%d, %d) = %d; want %d", test.a, test.b, result, test.expected)
			}
		})
	}
}

func TestSubtract(t *testing.T) {
	result := Subtract(10, 3)
	if result != 8 { // This will fail (should be 7)
		t.Errorf("Subtract(10, 3) = %d; want 8", result)
	}
}
`,
	})
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.testRunner.Check(ctx)

	s.NoError(err)
	s.Equal(interfaces.CheckStatusFail, result.Status)
	s.NotEmpty(result.Details)

	// Verify detail structure
	for _, detail := range result.Details {
		s.NotEmpty(detail.Message)
		s.NotEmpty(detail.Rule)
		if detail.File != "" {
			s.True(strings.HasSuffix(detail.File, "_test.go"))
		}
	}
}

// TestClose tests proper cleanup
func (s *TestRunnerTestSuite) TestClose() {
	err := s.testRunner.Close()
	s.NoError(err)
}

// Helper methods

func (s *TestRunnerTestSuite) createTestPackage(packageName string, files map[string]string) error {
	packageDir := filepath.Join(s.tempDir, packageName)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return err
	}

	for filename, content := range files {
		filePath := filepath.Join(packageDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

// Benchmark tests

func BenchmarkTestRunner(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "test-runner-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create simple test package
	packageDir := filepath.Join(tempDir, "bench")
	os.MkdirAll(packageDir, 0755)

	mainContent := `package bench
func Add(a, b int) int { return a + b }`
	testContent := `package bench
import "testing"
func TestAdd(t *testing.T) {
	if Add(2, 3) != 5 { t.Error("failed") }
}`

	os.WriteFile(filepath.Join(packageDir, "main.go"), []byte(mainContent), 0644)
	os.WriteFile(filepath.Join(packageDir, "main_test.go"), []byte(testContent), 0644)

	runner := NewTestRunner(tempDir, TestRunnerOptions{
		EnableCoverage: false, // Disable coverage for faster benchmarks
	})
	defer runner.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runner.Check(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTestRunnerLargeCoverage(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "test-runner-coverage-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test package with multiple files
	packageDir := filepath.Join(tempDir, "coverage")
	os.MkdirAll(packageDir, 0755)

	for i := 0; i < 5; i++ {
		mainContent := fmt.Sprintf(`package coverage
func Func%d(x int) int { return x * %d }`, i, i+1)
		testContent := fmt.Sprintf(`package coverage
import "testing"
func TestFunc%d(t *testing.T) {
	if Func%d(2) != %d { t.Error("failed") }
}`, i, i, 2*(i+1))

		os.WriteFile(filepath.Join(packageDir, fmt.Sprintf("file%d.go", i)), []byte(mainContent), 0644)
		os.WriteFile(filepath.Join(packageDir, fmt.Sprintf("file%d_test.go", i)), []byte(testContent), 0644)
	}

	runner := NewTestRunner(tempDir, TestRunnerOptions{
		EnableCoverage: true,
	})
	defer runner.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runner.Check(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
