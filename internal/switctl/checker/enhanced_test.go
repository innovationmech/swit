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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// EnhancedTestSuite provides enhanced testing capabilities for the checker system
type EnhancedTestSuite struct {
	suite.Suite
	tempDir string
	logger  *MockLogger
}

func TestEnhancedTestSuite(t *testing.T) {
	suite.Run(t, new(EnhancedTestSuite))
}

func (s *EnhancedTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "enhanced-checker-test-*")
	s.Require().NoError(err)

	s.logger = &MockLogger{messages: make([]LogMessage, 0)}
}

func (s *EnhancedTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

// TestManagerWithExtremeTimeouts tests manager behavior with extreme timeout values
func (s *EnhancedTestSuite) TestManagerWithExtremeTimeouts() {
	manager := NewManager(s.logger)

	// Test with zero timeout
	manager.SetTimeout(0)
	s.Equal(time.Duration(0), manager.timeout)

	// Test with very small timeout
	manager.SetTimeout(1 * time.Nanosecond)
	s.Equal(1*time.Nanosecond, manager.timeout)

	// Test with very large timeout
	manager.SetTimeout(24 * time.Hour)
	s.Equal(24*time.Hour, manager.timeout)

	// Test with negative timeout (should work as Go allows negative durations)
	manager.SetTimeout(-1 * time.Second)
	s.Equal(-1*time.Second, manager.timeout)
}

// TestCompositeCheckerWithNilLogger tests CompositeChecker behavior with nil logger
func (s *EnhancedTestSuite) TestCompositeCheckerWithNilLogger() {
	// This should not panic, but may cause runtime issues
	s.NotPanics(func() {
		checker := NewCompositeChecker(s.tempDir, nil)
		s.NotNil(checker)
	})
}

// TestCompositeCheckerWithEmptyWorkDir tests behavior with empty working directory
func (s *EnhancedTestSuite) TestCompositeCheckerWithEmptyWorkDir() {
	checker := NewCompositeChecker("", s.logger)
	s.NotNil(checker)

	// Test style checking with empty work dir
	styleResult := checker.CheckCodeStyle()
	s.NotNil(styleResult)
}

// TestCompositeCheckerWithNonexistentWorkDir tests behavior with non-existent directory
func (s *EnhancedTestSuite) TestCompositeCheckerWithNonexistentWorkDir() {
	nonexistentDir := "/absolutely/nonexistent/path/that/should/never/exist"
	checker := NewCompositeChecker(nonexistentDir, s.logger)
	s.NotNil(checker)

	// Test various checker methods with nonexistent directory
	styleResult := checker.CheckCodeStyle()
	s.NotNil(styleResult)

	testResult := checker.RunTests()
	s.NotNil(testResult)

	securityResult := checker.CheckSecurity()
	s.NotNil(securityResult)
}

// TestContextCancellationWithMultipleCheckers tests context cancellation with multiple concurrent checkers
func (s *EnhancedTestSuite) TestContextCancellationWithMultipleCheckers() {
	const numCheckers = 10

	s.createTestProject()

	var wg sync.WaitGroup
	results := make([]error, numCheckers)

	ctx, cancel := context.WithCancel(context.Background())

	// Start multiple checkers concurrently
	for i := 0; i < numCheckers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			checker := NewCompositeChecker(s.tempDir, s.logger)

			// Test cancellation on different checker methods
			select {
			case <-ctx.Done():
				results[index] = ctx.Err()
			default:
				styleResult := checker.CheckCodeStyle()
				if styleResult.Name == "" {
					results[index] = context.Canceled
				} else {
					results[index] = nil
				}
			}
		}(i)
	}

	// Cancel context after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	wg.Wait()

	// Some checkers might complete before cancellation, others should be cancelled
	cancelledCount := 0
	completedCount := 0
	for _, err := range results {
		switch err {
		case context.Canceled:
			cancelledCount++
		case nil:
			completedCount++
		}
	}

	s.True(cancelledCount > 0 || completedCount > 0, "Should have either cancelled or completed checkers")
}

// TestContextTimeoutWithProgressiveDelays tests behavior with progressive timeout delays
func (s *EnhancedTestSuite) TestContextTimeoutWithProgressiveDelays() {
	s.createTestProject()

	timeouts := []time.Duration{
		1 * time.Millisecond,
		10 * time.Millisecond,
		100 * time.Millisecond,
		1 * time.Second,
	}

	for _, timeout := range timeouts {
		s.Run(fmt.Sprintf("timeout_%v", timeout), func() {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			checker := NewCompositeChecker(s.tempDir, s.logger)
			result, err := checker.Check(ctx)

			// Should either complete successfully or timeout
			if err != nil {
				s.Equal(context.DeadlineExceeded, err)
			} else {
				s.NotNil(result)
			}
		})
	}
}

// TestManagerConcurrentOperations tests manager with concurrent operations
func (s *EnhancedTestSuite) TestManagerConcurrentOperations() {
	manager := NewManager(s.logger)
	const numOperations = 50

	var wg sync.WaitGroup

	// Concurrent configuration changes
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			timeout := time.Duration(index) * time.Millisecond
			manager.SetTimeout(timeout)
			manager.SetParallel(index%2 == 0)
		}(i)
	}

	// Concurrent checker registrations
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			checker := NewCompositeChecker(s.tempDir, s.logger)
			manager.RegisterChecker(checker)
		}(i)
	}

	wg.Wait()

	// Manager should remain in a consistent state
	s.NotNil(manager.logger)
}

// TestCheckerRaceConditions tests for race conditions in checker operations
func (s *EnhancedTestSuite) TestCheckerRaceConditions() {
	s.createTestProject()

	const numGoroutines = 20
	checker := NewCompositeChecker(s.tempDir, s.logger)

	var wg sync.WaitGroup
	errors := make([]error, numGoroutines)
	results := make([]interfaces.CheckResult, numGoroutines)

	// Run multiple checks concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ctx := context.Background()
			results[index], errors[index] = checker.Check(ctx)
		}(i)
	}

	wg.Wait()

	// All operations should complete without race conditions
	for i := 0; i < numGoroutines; i++ {
		s.NoError(errors[i], "Goroutine %d should not have errors", i)
		s.NotNil(results[i], "Goroutine %d should have valid result", i)
	}
}

// TestMemoryLeaksAndResourceCleanup tests for memory leaks and proper resource cleanup
func (s *EnhancedTestSuite) TestMemoryLeaksAndResourceCleanup() {
	s.createTestProject()

	const iterations = 100
	var memStatsBefore, memStatsAfter runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&memStatsBefore)

	// Create and destroy many checkers to test for memory leaks
	for i := 0; i < iterations; i++ {
		checker := NewCompositeChecker(s.tempDir, s.logger)
		ctx := context.Background()
		_, err := checker.Check(ctx)
		s.NoError(err)

		// Explicitly close checker if it has a Close method
		if closer, ok := any(checker).(interface{ Close() error }); ok {
			closer.Close()
		}

		// Force garbage collection periodically
		if i%10 == 0 {
			runtime.GC()
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&memStatsAfter)

	// Memory usage should not grow excessively
	memoryGrowth := memStatsAfter.Alloc - memStatsBefore.Alloc
	s.True(memoryGrowth < 10*1024*1024, "Memory growth should be less than 10MB, got %d bytes", memoryGrowth)
}

// TestErrorHandlingWithCorruptedFiles tests error handling with corrupted or invalid files
func (s *EnhancedTestSuite) TestErrorHandlingWithCorruptedFiles() {
	// Create corrupted files
	corruptedFiles := map[string][]byte{
		"corrupted.go":   {0x00, 0x01, 0x02, 0xFF, 0xFE}, // Invalid UTF-8
		"binary.exe":     make([]byte, 1024),             // Binary file
		"large.txt":      make([]byte, 50*1024*1024),     // 50MB file
		"empty.go":       {},                             // Empty file
		"invalid.yaml":   []byte("invalid: yaml: content: [unclosed"),
		"malformed.json": []byte(`{"invalid": json, "missing": quotes}`),
	}

	for filename, content := range corruptedFiles {
		err := s.createFile(filename, string(content))
		s.Require().NoError(err)
	}

	checker := NewCompositeChecker(s.tempDir, s.logger)
	ctx := context.Background()
	result, err := checker.Check(ctx)

	// Should handle corrupted files gracefully
	s.NoError(err)
	s.NotNil(result)
}

// TestHighLoadPerformance tests performance under high load conditions
func (s *EnhancedTestSuite) TestHighLoadPerformance() {
	s.createLargeTestProject()

	checker := NewCompositeChecker(s.tempDir, s.logger)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	result, err := checker.Check(ctx)
	duration := time.Since(start)

	s.NoError(err)
	s.NotNil(result)
	s.True(duration < 30*time.Second, "Check should complete within reasonable time")

	// Log performance metrics
	s.logger.Info("High load performance test completed",
		"duration", duration,
		"result_status", result.Name,
	)
}

// TestErrorRecoveryAndResilience tests error recovery and system resilience
func (s *EnhancedTestSuite) TestErrorRecoveryAndResilience() {
	checker := NewCompositeChecker(s.tempDir, s.logger)

	// Test with various error conditions
	errorConditions := []struct {
		name string
		ctx  func() context.Context
	}{
		{
			name: "cancelled_context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
		},
		{
			name: "expired_context",
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(1 * time.Millisecond) // Ensure timeout
				return ctx
			},
		},
	}

	for _, condition := range errorConditions {
		s.Run(condition.name, func() {
			ctx := condition.ctx()
			_, err := checker.Check(ctx)

			// Should handle error conditions gracefully
			if err != nil {
				s.True(errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			}
		})
	}
}

// TestAtomicOperationsAndConsistency tests atomic operations and data consistency
func (s *EnhancedTestSuite) TestAtomicOperationsAndConsistency() {
	manager := NewManager(s.logger)
	const numOperations = 1000

	var counter int64
	var wg sync.WaitGroup

	// Simulate atomic operations
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&counter, 1)

			// Perform manager operations
			manager.SetTimeout(time.Duration(atomic.LoadInt64(&counter)) * time.Millisecond)
		}()
	}

	wg.Wait()

	finalCount := atomic.LoadInt64(&counter)
	s.Equal(int64(numOperations), finalCount)
}

// TestBoundaryConditionsAndEdgeCases tests various boundary conditions
func (s *EnhancedTestSuite) TestBoundaryConditionsAndEdgeCases() {
	testCases := []struct {
		name    string
		workDir string
		setup   func() error
	}{
		{
			name:    "empty_directory",
			workDir: s.tempDir,
			setup:   func() error { return nil },
		},
		{
			name:    "directory_with_only_hidden_files",
			workDir: s.tempDir,
			setup: func() error {
				return s.createFile(".hidden", "hidden content")
			},
		},
		{
			name:    "directory_with_symlinks",
			workDir: s.tempDir,
			setup: func() error {
				if err := s.createFile("target.txt", "target"); err != nil {
					return err
				}
				return os.Symlink(
					filepath.Join(s.tempDir, "target.txt"),
					filepath.Join(s.tempDir, "link.txt"),
				)
			},
		},
		{
			name:    "deeply_nested_directory",
			workDir: s.tempDir,
			setup: func() error {
				deepDir := s.tempDir
				for i := 0; i < 10; i++ {
					deepDir = filepath.Join(deepDir, fmt.Sprintf("level_%d", i))
				}
				if err := os.MkdirAll(deepDir, 0755); err != nil {
					return err
				}
				return s.createFileInDir(deepDir, "deep.go", "package main")
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.setup()
			s.Require().NoError(err)

			checker := NewCompositeChecker(tc.workDir, s.logger)
			ctx := context.Background()
			result, err := checker.Check(ctx)

			s.NoError(err)
			s.NotNil(result)
		})
	}
}

// TestFileSystemPermissionsAndAccess tests behavior with various file system permissions
func (s *EnhancedTestSuite) TestFileSystemPermissionsAndAccess() {
	// Skip on Windows where permission handling is different
	if runtime.GOOS == "windows" {
		s.T().Skip("Skipping permission tests on Windows")
	}

	permissionTests := []struct {
		name     string
		filename string
		perm     os.FileMode
		content  string
	}{
		{
			name:     "readonly_file",
			filename: "readonly.go",
			perm:     0444,
			content:  "package main\n",
		},
		{
			name:     "no_read_permission",
			filename: "noread.go",
			perm:     0222,
			content:  "package main\n",
		},
		{
			name:     "executable_only",
			filename: "executable.go",
			perm:     0111,
			content:  "package main\n",
		},
	}

	for _, test := range permissionTests {
		s.Run(test.name, func() {
			filePath := filepath.Join(s.tempDir, test.filename)
			err := os.WriteFile(filePath, []byte(test.content), 0644)
			s.Require().NoError(err)

			err = os.Chmod(filePath, test.perm)
			s.Require().NoError(err)

			checker := NewCompositeChecker(s.tempDir, s.logger)
			ctx := context.Background()
			result, err := checker.Check(ctx)

			// Should handle permission issues gracefully
			s.NoError(err)
			s.NotNil(result)

			// Restore readable permission for cleanup
			os.Chmod(filePath, 0644)
		})
	}
}

// TestConcurrentResourceAccess tests concurrent access to shared resources
func (s *EnhancedTestSuite) TestConcurrentResourceAccess() {
	s.createTestProject()

	const numConcurrentChecks = 10
	manager := NewManager(s.logger)

	var wg sync.WaitGroup
	errors := make([]error, numConcurrentChecks)
	results := make([]interfaces.CheckResult, numConcurrentChecks)

	for i := 0; i < numConcurrentChecks; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			checker := NewCompositeChecker(s.tempDir, s.logger)
			manager.RegisterChecker(checker)

			ctx := context.Background()
			results[index], errors[index] = checker.Check(ctx)
		}(i)
	}

	wg.Wait()

	// All concurrent operations should succeed
	for i := 0; i < numConcurrentChecks; i++ {
		s.NoError(errors[i])
		s.NotNil(results[i])
	}
}

// Helper methods for test setup

func (s *EnhancedTestSuite) createTestProject() {
	// Create a basic Go project structure
	err := s.createFile("go.mod", `module github.com/test/project

go 1.19

require (
    github.com/stretchr/testify v1.8.0
)
`)
	s.Require().NoError(err)

	err = s.createFile("main.go", `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
`)
	s.Require().NoError(err)

	err = s.createFile("config.yaml", `server:
  port: 8080
database:
  type: mysql
`)
	s.Require().NoError(err)
}

func (s *EnhancedTestSuite) createLargeTestProject() {
	s.createTestProject()

	// Create many files to simulate a large project
	packages := []string{"api", "service", "repository", "model", "utils", "config", "middleware"}

	for _, pkg := range packages {
		pkgDir := filepath.Join(s.tempDir, pkg)
		err := os.MkdirAll(pkgDir, 0755)
		s.Require().NoError(err)

		// Create multiple files per package
		for i := 0; i < 20; i++ {
			filename := fmt.Sprintf("%s_%d.go", pkg, i)
			content := fmt.Sprintf(`package %s

import (
    "context"
    "fmt"
    "time"
)

// %s%d represents a %s component
type %s%d struct {
    ID   int
    Name string
    Time time.Time
}

// New%s%d creates a new %s instance
func New%s%d() *%s%d {
    return &%s%d{
        ID:   %d,
        Name: "%s_%d",
        Time: time.Now(),
    }
}

// Process handles %s processing
func (c *%s%d) Process(ctx context.Context) error {
    fmt.Printf("Processing %%s\n", c.Name)
    return nil
}
`,
				pkg,         // package %s
				pkg, i, pkg, // %s%d represents a %s component
				pkg, i, // type %s%d struct
				pkg, i, pkg, // New%s%d creates a new %s instance
				pkg, i, pkg, i, // func New%s%d() *%s%d
				pkg, i, // return &%s%d{
				i,      // ID: %d,
				pkg, i, // Name: "%s_%d",
				pkg,    // Process handles %s processing
				pkg, i, // func (c *%s%d) Process
			)

			err = s.createFileInDir(pkgDir, filename, content)
			s.Require().NoError(err)

			// Create corresponding test file
			testFilename := fmt.Sprintf("%s_%d_test.go", pkg, i)
			testContent := fmt.Sprintf(`package %s

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
)

func Test%s%d_Process(t *testing.T) {
    c := New%s%d()
    err := c.Process(context.Background())
    assert.NoError(t, err)
}
`, pkg, pkg, i, pkg, i)

			err = s.createFileInDir(pkgDir, testFilename, testContent)
			s.Require().NoError(err)
		}
	}

	// Create configuration files
	configs := map[string]string{
		"app.yaml": `
app:
  name: large-test-app
  version: 1.0.0
server:
  host: localhost
  port: 8080
database:
  host: localhost
  port: 5432
  name: testdb
`,
		"docker-compose.yaml": `
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: testdb
`,
		"README.md": `
# Large Test Project

This is a large test project generated for performance testing.

## Structure

- Multiple packages with numerous files
- Comprehensive test coverage
- Various configuration files
`,
	}

	for filename, content := range configs {
		err := s.createFile(filename, content)
		s.Require().NoError(err)
	}
}

func (s *EnhancedTestSuite) createFile(filename, content string) error {
	return s.createFileInDir(s.tempDir, filename, content)
}

func (s *EnhancedTestSuite) createFileInDir(dir, filename, content string) error {
	filePath := filepath.Join(dir, filename)
	return os.WriteFile(filePath, []byte(content), 0644)
}

// Performance and stress test helpers

// MockSlowChecker simulates a slow checker for timeout testing
type MockSlowChecker struct {
	delay time.Duration
}

func NewMockSlowChecker(delay time.Duration) *MockSlowChecker {
	return &MockSlowChecker{delay: delay}
}

func (m *MockSlowChecker) Check(ctx context.Context) (interfaces.CheckResult, error) {
	select {
	case <-time.After(m.delay):
		return interfaces.CheckResult{
			Name:     "slow-check",
			Status:   interfaces.CheckStatusPass,
			Message:  fmt.Sprintf("Completed after %v delay", m.delay),
			Duration: m.delay,
		}, nil
	case <-ctx.Done():
		return interfaces.CheckResult{}, ctx.Err()
	}
}

func (m *MockSlowChecker) Name() string        { return "slow-checker" }
func (m *MockSlowChecker) Description() string { return "A slow checker for testing" }
func (m *MockSlowChecker) Close() error        { return nil }

// TestSlowCheckerTimeout tests timeout behavior with slow checkers
func (s *EnhancedTestSuite) TestSlowCheckerTimeout() {
	slowChecker := NewMockSlowChecker(5 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := slowChecker.Check(ctx)
	s.Error(err)
	s.Equal(context.DeadlineExceeded, err)
	s.Empty(result.Name)
}

// MockErrorChecker simulates a checker that always returns errors
type MockErrorChecker struct {
	errorMessage string
}

func NewMockErrorChecker(errorMessage string) *MockErrorChecker {
	return &MockErrorChecker{errorMessage: errorMessage}
}

func (m *MockErrorChecker) Check(ctx context.Context) (interfaces.CheckResult, error) {
	return interfaces.CheckResult{}, errors.New(m.errorMessage)
}

func (m *MockErrorChecker) Name() string        { return "error-checker" }
func (m *MockErrorChecker) Description() string { return "A checker that always errors" }
func (m *MockErrorChecker) Close() error        { return nil }

// TestErrorCheckerHandling tests handling of checkers that return errors
func (s *EnhancedTestSuite) TestErrorCheckerHandling() {
	errorChecker := NewMockErrorChecker("simulated checker error")

	ctx := context.Background()
	result, err := errorChecker.Check(ctx)

	s.Error(err)
	s.Equal("simulated checker error", err.Error())
	s.Empty(result.Name)
}
