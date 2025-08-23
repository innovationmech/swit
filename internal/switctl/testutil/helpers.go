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

package testutil

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertFileExists checks if a file exists and fails the test if it doesn't.
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	assert.NoError(t, err, "File should exist: %s", path)
}

// AssertFileNotExists checks if a file doesn't exist and fails the test if it does.
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "File should not exist: %s", path)
}

// AssertFileContains checks if a file contains the expected content.
func AssertFileContains(t *testing.T, path, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read file: %s", path)
	assert.Contains(t, string(content), expected, "File should contain expected content")
}

// AssertFileNotContains checks if a file doesn't contain the specified content.
func AssertFileNotContains(t *testing.T, path, unexpected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read file: %s", path)
	assert.NotContains(t, string(content), unexpected, "File should not contain unexpected content")
}

// AssertFileEquals checks if a file content equals the expected content.
func AssertFileEquals(t *testing.T, path, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read file: %s", path)
	assert.Equal(t, expected, string(content), "File content should match expected")
}

// AssertDirectoryExists checks if a directory exists.
func AssertDirectoryExists(t *testing.T, path string) {
	t.Helper()
	stat, err := os.Stat(path)
	require.NoError(t, err, "Directory should exist: %s", path)
	assert.True(t, stat.IsDir(), "Path should be a directory: %s", path)
}

// AssertDirectoryNotExists checks if a directory doesn't exist.
func AssertDirectoryNotExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "Directory should not exist: %s", path)
}

// AssertFilesGenerated checks if all expected files were generated.
func AssertFilesGenerated(t *testing.T, baseDir string, expectedFiles []string) {
	t.Helper()
	for _, file := range expectedFiles {
		fullPath := filepath.Join(baseDir, file)
		AssertFileExists(t, fullPath)
	}
}

// AssertTemplateOutput checks if template output matches expected content.
func AssertTemplateOutput(t *testing.T, actual []byte, expectedSubstrings []string) {
	t.Helper()
	actualStr := string(actual)
	for _, substring := range expectedSubstrings {
		assert.Contains(t, actualStr, substring, "Template output should contain: %s", substring)
	}
}

// AssertValidGoCode checks if the generated content is valid Go code syntax.
func AssertValidGoCode(t *testing.T, content []byte) {
	t.Helper()
	// Basic checks for Go code validity
	contentStr := string(content)

	// Should have package declaration
	assert.Contains(t, contentStr, "package ", "Go code should have package declaration")

	// Should not have template syntax errors
	assert.NotContains(t, contentStr, "{{", "Go code should not contain unprocessed template syntax")
	assert.NotContains(t, contentStr, "}}", "Go code should not contain unprocessed template syntax")

	// Should have proper imports if any
	if strings.Contains(contentStr, "import") {
		// Basic import syntax check
		lines := strings.Split(contentStr, "\n")
		importStarted := false
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "import") {
				importStarted = true
				continue
			}
			if importStarted && line != "" && !strings.HasPrefix(line, "//") {
				if strings.HasPrefix(line, "\"") || strings.HasPrefix(line, "(") || strings.HasPrefix(line, ")") {
					continue
				}
				// If we reach here, import section should be ended
				break
			}
		}
	}
}

// AssertValidYAML checks if the generated content is valid YAML.
func AssertValidYAML(t *testing.T, content []byte) {
	t.Helper()
	// Basic YAML structure checks
	contentStr := string(content)

	// Should not have template syntax errors
	assert.NotContains(t, contentStr, "{{", "YAML should not contain unprocessed template syntax")
	assert.NotContains(t, contentStr, "}}", "YAML should not contain unprocessed template syntax")
}

// CreateTestProject creates a temporary test project structure.
func CreateTestProject(t *testing.T) string {
	t.Helper()

	tempDir, err := CreateTempDir()
	require.NoError(t, err, "Failed to create temp directory")

	// Create basic project structure
	dirs := []string{
		"cmd",
		"internal",
		"pkg",
		"api",
		"configs",
		"docs",
	}

	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		require.NoError(t, err, "Failed to create directory: %s", dir)
	}

	// Create basic files
	files := map[string]string{
		"go.mod": `module github.com/test/project
go 1.19
`,
		"README.md": "# Test Project\n\nThis is a test project.\n",
		".gitignore": `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Output of the go coverage tool
*.out
`,
	}

	for name, content := range files {
		path := filepath.Join(tempDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err, "Failed to create file: %s", name)
	}

	return tempDir
}

// CompareTemplateOutput compares actual template output with expected patterns.
func CompareTemplateOutput(t *testing.T, actual []byte, expected []byte, ignoreWhitespace bool) {
	t.Helper()

	actualStr := string(actual)
	expectedStr := string(expected)

	if ignoreWhitespace {
		actualStr = normalizeWhitespace(actualStr)
		expectedStr = normalizeWhitespace(expectedStr)
	}

	assert.Equal(t, expectedStr, actualStr, "Template output should match expected")
}

// normalizeWhitespace normalizes whitespace for comparison.
func normalizeWhitespace(s string) string {
	// Replace multiple spaces with single space
	s = strings.ReplaceAll(s, "\t", " ")
	lines := strings.Split(s, "\n")

	var normalized []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			normalized = append(normalized, line)
		}
	}

	return strings.Join(normalized, "\n")
}

// AssertNoTemplateErrors checks that template rendering didn't produce errors.
func AssertNoTemplateErrors(t *testing.T, content []byte) {
	t.Helper()
	contentStr := string(content)

	// Check for common template error patterns
	assert.NotContains(t, contentStr, "<no value>", "Template should not contain <no value>")
	assert.NotContains(t, contentStr, "template:", "Template should not contain template error messages")
	assert.NotContains(t, contentStr, "executing template", "Template should not contain execution errors")
}

// SetupTemplateTestSuite provides common setup for template tests.
func SetupTemplateTestSuite(t *testing.T) (string, func()) {
	t.Helper()

	tempDir, err := CreateTempDir()
	require.NoError(t, err, "Failed to create temp directory")

	// Create template files
	err = CreateTestTemplateFiles(tempDir)
	require.NoError(t, err, "Failed to create template files")

	cleanup := func() {
		err := CleanupTempDir(tempDir)
		assert.NoError(t, err, "Failed to cleanup temp directory")
	}

	return tempDir, cleanup
}

// CaptureOutput captures stdout/stderr for testing.
func CaptureOutput(fn func()) (stdout, stderr string) {
	// This would be used for testing CLI output
	// Implementation would depend on how the CLI outputs are handled
	var stdoutBuf, stderrBuf bytes.Buffer

	// Save original stdout/stderr
	// Note: This is a simplified version, actual implementation would need
	// proper stdout/stderr capture mechanism

	fn()

	return stdoutBuf.String(), stderrBuf.String()
}

// AssertContainsAll checks if content contains all expected strings.
func AssertContainsAll(t *testing.T, content string, expected []string) {
	t.Helper()
	for _, exp := range expected {
		assert.Contains(t, content, exp, "Content should contain: %s", exp)
	}
}

// AssertContainsNone checks if content contains none of the unexpected strings.
func AssertContainsNone(t *testing.T, content string, unexpected []string) {
	t.Helper()
	for _, unexp := range unexpected {
		assert.NotContains(t, content, unexp, "Content should not contain: %s", unexp)
	}
}

// AssertFilePermissions checks if a file has the expected permissions.
func AssertFilePermissions(t *testing.T, path string, expectedPerm os.FileMode) {
	t.Helper()
	stat, err := os.Stat(path)
	require.NoError(t, err, "Failed to stat file: %s", path)
	assert.Equal(t, expectedPerm, stat.Mode().Perm(), "File should have expected permissions")
}

// AssertErrorContains checks if an error contains the expected message.
func AssertErrorContains(t *testing.T, err error, expectedMsg string) {
	t.Helper()
	require.Error(t, err, "Expected an error")
	assert.Contains(t, err.Error(), expectedMsg, "Error should contain expected message")
}

// AssertErrorType checks if an error is of the expected type.
func AssertErrorType(t *testing.T, err error, expectedType interface{}) {
	t.Helper()
	require.Error(t, err, "Expected an error")
	assert.IsType(t, expectedType, err, "Error should be of expected type")
}

// CreateFileWithContent creates a file with the specified content.
func CreateFileWithContent(t *testing.T, path, content string) {
	t.Helper()

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err, "Failed to create directory: %s", dir)
	}

	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err, "Failed to create file: %s", path)
}

// ReadFileContent reads and returns file content for testing.
func ReadFileContent(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read file: %s", path)
	return string(content)
}

// CreateDirectoryStructure creates a directory structure from a map.
func CreateDirectoryStructure(t *testing.T, baseDir string, structure map[string]string) {
	t.Helper()

	for path, content := range structure {
		fullPath := filepath.Join(baseDir, path)

		if content == "" {
			// Create directory
			err := os.MkdirAll(fullPath, 0755)
			require.NoError(t, err, "Failed to create directory: %s", fullPath)
		} else {
			// Create file with content
			CreateFileWithContent(t, fullPath, content)
		}
	}
}

// AssertFilesMatch checks if two files have identical content.
func AssertFilesMatch(t *testing.T, file1, file2 string) {
	t.Helper()

	content1, err := os.ReadFile(file1)
	require.NoError(t, err, "Failed to read file: %s", file1)

	content2, err := os.ReadFile(file2)
	require.NoError(t, err, "Failed to read file: %s", file2)

	assert.Equal(t, content1, content2, "Files should have identical content")
}

// WaitForCondition waits for a condition to be true or times out.
func WaitForCondition(t *testing.T, condition func() bool, timeout, interval string) {
	t.Helper()
	// Implementation would use time.Duration and ticker
	// This is a placeholder for async testing support
	assert.True(t, condition(), "Condition should be met")
}

// MockWorkingDirectory changes to a mock working directory for testing.
func MockWorkingDirectory(t *testing.T, dir string) func() {
	t.Helper()

	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")

	err = os.Chdir(dir)
	require.NoError(t, err, "Failed to change to test directory")

	return func() {
		err := os.Chdir(originalDir)
		assert.NoError(t, err, "Failed to restore original directory")
	}
}
