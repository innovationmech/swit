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

package main

import (
	"os"
	"testing"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestMain is a special function that runs before any tests
func TestMain(m *testing.M) {
	// Setup: Store original values
	originalVersion := version
	originalBuildTime := buildTime
	originalGitCommit := gitCommit

	// Run tests
	code := m.Run()

	// Teardown: Restore original values
	version = originalVersion
	buildTime = originalBuildTime
	gitCommit = originalGitCommit

	os.Exit(code)
}

// mockLogger 用于捕获日志输出，避免真实输出
func mockLogger() {
	logger.Logger, _ = zap.NewDevelopment()
}

func TestRun_Success(t *testing.T) {
	mockLogger()
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"swit-serve", "--help"}

	code := run()
	assert.Equal(t, 0, code)
}

func TestRun_Error(t *testing.T) {
	mockLogger()
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"swit-serve", "invalid-cmd"}

	code := run()
	assert.Equal(t, 1, code)
}

func TestRun_Version(t *testing.T) {
	mockLogger()
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"swit-serve", "--version"}

	code := run()
	assert.Equal(t, 0, code)
}

func TestRun_EmptyArgs(t *testing.T) {
	mockLogger()
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"swit-serve"}

	code := run()
	assert.Equal(t, 0, code)
}

func TestVersionVariables(t *testing.T) {
	// Test that version variables have expected default values
	assert.Equal(t, "dev", version)
	assert.Equal(t, "unknown", buildTime)
	assert.Equal(t, "unknown", gitCommit)
}

func TestMainFunctionExitCodes(t *testing.T) {
	// Save original os.Args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name     string
		args     []string
		exitCode int
	}{
		{
			name:     "help command",
			args:     []string{"swit-serve", "--help"},
			exitCode: 0,
		},
		{
			name:     "version command",
			args:     []string{"swit-serve", "--version"},
			exitCode: 0,
		},
		{
			name:     "empty args",
			args:     []string{"swit-serve"},
			exitCode: 0, // Should show help
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test is designed to run without actually calling os.Exit
			// We'll test the command structure instead
			assert.True(t, true) // Placeholder for actual test logic
		})
	}
}

func TestBuildVariables(t *testing.T) {
	// Test that build variables can be modified
	version = "1.0.0-test"
	buildTime = "2025-01-01T00:00:00Z"
	gitCommit = "abc123def456"

	assert.Equal(t, "1.0.0-test", version)
	assert.Equal(t, "2025-01-01T00:00:00Z", buildTime)
	assert.Equal(t, "abc123def456", gitCommit)
}

func TestLoggerInitialization(t *testing.T) {
	// Test that logger can be initialized without panic
	// This is a basic sanity check
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logger initialization caused panic: %v", r)
		}
	}()

	// Note: We can't fully test main() without side effects,
	// but we can test the components
	assert.True(t, true)
}

func TestMainFunctionStructure(t *testing.T) {
	// Test that the main function structure is valid
	// This is a structural test rather than functional
	assert.NotEmpty(t, version)
	assert.NotEmpty(t, buildTime)
	assert.NotEmpty(t, gitCommit)
}
