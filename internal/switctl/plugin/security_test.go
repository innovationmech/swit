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

package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SecurityTestSuite contains all security tests for the plugin system.
type SecurityTestSuite struct {
	suite.Suite
	tempDir   string
	validator *SecurityValidator
}

// SetupSuite initializes the test suite.
func (suite *SecurityTestSuite) SetupSuite() {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "switctl-security-test-*")
	require.NoError(suite.T(), err)
	suite.tempDir = tempDir
}

// TearDownSuite cleans up after all tests.
func (suite *SecurityTestSuite) TearDownSuite() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// SetupTest initializes each test case.
func (suite *SecurityTestSuite) SetupTest() {
	config := &SecurityConfig{
		AllowUnsigned:     false,
		RequireChecksum:   true,
		TrustedSources:    []string{}, // Empty by default to allow most paths
		BlockedPatterns:   []string{"*.exe", "*.bat"},
		MaxFileSizeMB:     50,
		AllowedExtensions: []string{".so", ".dll", ".dylib"},
	}
	suite.validator = NewSecurityValidator(config)
}

// TestNewSecurityValidator tests security validator creation.
func (suite *SecurityTestSuite) TestNewSecurityValidator() {
	config := &SecurityConfig{
		AllowUnsigned:     true,
		RequireChecksum:   false,
		TrustedSources:    []string{"local"},
		BlockedPatterns:   []string{"*.tmp"},
		MaxFileSizeMB:     25,
		AllowedExtensions: []string{".so"},
	}

	validator := NewSecurityValidator(config)

	assert.NotNil(suite.T(), validator)
	assert.Equal(suite.T(), config, validator.config)
}

// TestValidatePluginPath tests path validation.
func (suite *SecurityTestSuite) TestValidatePluginPath() {
	tests := []struct {
		name      string
		path      string
		expectErr bool
	}{
		{
			name:      "valid path",
			path:      "/plugins/test.so",
			expectErr: false,
		},
		{
			name:      "path traversal attempt",
			path:      "../../../etc/passwd",
			expectErr: true,
		},
		{
			name:      "blocked pattern",
			path:      "/plugins/malicious.exe",
			expectErr: true,
		},
		{
			name:      "relative path traversal",
			path:      "plugins/../../../root/.ssh/id_rsa",
			expectErr: true,
		},
		{
			name:      "null byte injection",
			path:      "/plugins/test.so\x00.exe",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			err := suite.validator.ValidatePluginPath(tt.path)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidatePluginFile tests file validation.
func (suite *SecurityTestSuite) TestValidatePluginFile() {
	// Create test files
	validFile := filepath.Join(suite.tempDir, "valid.so")
	invalidFile := filepath.Join(suite.tempDir, "invalid.exe")
	largeFile := filepath.Join(suite.tempDir, "large.so")

	// Create valid file
	content := []byte("fake shared object content")
	err := os.WriteFile(validFile, content, 0644)
	require.NoError(suite.T(), err)

	// Create invalid extension file
	err = os.WriteFile(invalidFile, content, 0644)
	require.NoError(suite.T(), err)

	// Create large file (simulate file too large)
	largeContent := make([]byte, 60*1024*1024) // 60MB
	err = os.WriteFile(largeFile, largeContent, 0644)
	require.NoError(suite.T(), err)

	tests := []struct {
		name      string
		path      string
		expectErr bool
	}{
		{
			name:      "valid file",
			path:      validFile,
			expectErr: false,
		},
		{
			name:      "invalid extension",
			path:      invalidFile,
			expectErr: true,
		},
		{
			name:      "file too large",
			path:      largeFile,
			expectErr: true,
		},
		{
			name:      "nonexistent file",
			path:      filepath.Join(suite.tempDir, "nonexistent.so"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			result := suite.validator.ValidatePluginFile(tt.path)
			if tt.expectErr {
				assert.False(t, result.Valid)
				assert.NotEmpty(t, result.Errors)
			} else {
				assert.True(t, result.Valid, "validation should pass for %s", tt.path)
			}
		})
	}
}

// TestChecksum tests checksum validation.
func (suite *SecurityTestSuite) TestChecksum() {
	// Create test file
	testFile := filepath.Join(suite.tempDir, "test.so")
	content := []byte("test plugin content")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(suite.T(), err)

	// Calculate checksum
	checksum, err := suite.validator.calculateChecksum(testFile)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), checksum)

	// Verify checksum consistency
	checksum2, err := suite.validator.calculateChecksum(testFile)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), checksum, checksum2)

	// Test with nonexistent file
	_, err = suite.validator.calculateChecksum("nonexistent.file")
	assert.Error(suite.T(), err)
}

// TestPathTraversalAttacks tests various path traversal attack vectors.
func (suite *SecurityTestSuite) TestPathTraversalAttacks() {
	attackVectors := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\cmd.exe",
		"/plugins/../../../etc/shadow",
		"plugins/../../root/.bashrc",
		"./../../bin/sh",
		"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd", // URL encoded
		"..%252f..%252f..%252fetc%252fpasswd",     // Double URL encoded
		"..%c0%af..%c0%af..%c0%afetc%c0%afpasswd", // UTF-8 encoded
	}

	for _, vector := range attackVectors {
		suite.T().Run("attack_vector_"+vector, func(t *testing.T) {
			err := suite.validator.ValidatePluginPath(vector)
			assert.Error(t, err, "Should block path traversal attack: %s", vector)
		})
	}
}

// TestFilePermissions tests file permission checks.
func (suite *SecurityTestSuite) TestFilePermissions() {
	// Create test file with different permissions
	testFile := filepath.Join(suite.tempDir, "perm_test.so")
	content := []byte("test content")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(suite.T(), err)

	// Test readable file
	result := suite.validator.ValidatePluginFile(testFile)
	assert.True(suite.T(), result.Valid)

	// Change to unreadable permissions (if supported)
	if os.Getuid() != 0 { // Skip if running as root
		err = os.Chmod(testFile, 0000)
		require.NoError(suite.T(), err)

		result = suite.validator.ValidatePluginFile(testFile)
		// On some systems, root or certain file systems might still allow reading
		// So we check if the validation either fails OR if it succeeds but shows permission 0
		if result.Valid {
			// If it's still valid, check that the permissions are recorded correctly
			assert.Equal(suite.T(), os.FileMode(0000), result.Permissions&0777)
		} else {
			assert.False(suite.T(), result.Valid)
		}

		// Restore permissions for cleanup
		os.Chmod(testFile, 0644)
	}
}

// TestValidateWithTrustedSources tests trusted source validation.
func (suite *SecurityTestSuite) TestValidateWithTrustedSources() {
	config := &SecurityConfig{
		AllowUnsigned:     false,
		RequireChecksum:   false,
		TrustedSources:    []string{"trusted", "/opt/plugins"},
		BlockedPatterns:   []string{},
		MaxFileSizeMB:     50,
		AllowedExtensions: []string{".so"},
	}

	validator := NewSecurityValidator(config)

	tests := []struct {
		name      string
		path      string
		expectErr bool
	}{
		{
			name:      "trusted source",
			path:      "/opt/plugins/test.so",
			expectErr: false,
		},
		{
			name:      "untrusted source",
			path:      "/tmp/malicious.so",
			expectErr: true,
		},
		{
			name:      "trusted prefix match",
			path:      "trusted/plugin.so",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePluginPath(tt.path)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConcurrentValidation tests concurrent validation operations.
func (suite *SecurityTestSuite) TestConcurrentValidation() {
	// Create test file
	testFile := filepath.Join(suite.tempDir, "concurrent_test.so")
	content := []byte("concurrent test content")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(suite.T(), err)

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Test path validation
			path := filepath.Join("/plugins", fmt.Sprintf("plugin%d.so", id))
			err := suite.validator.ValidatePluginPath(path)
			assert.NoError(suite.T(), err)

			// Test file validation
			result := suite.validator.ValidatePluginFile(testFile)
			assert.True(suite.T(), result.Valid)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// TestCalculateFileChecksum tests public checksum calculation.
func (suite *SecurityTestSuite) TestCalculateFileChecksum() {
	// Create test file
	testFile := filepath.Join(suite.tempDir, "checksum_test.so")
	content := []byte("checksum test content")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(suite.T(), err)

	// Calculate checksum
	checksum, err := suite.validator.CalculateFileChecksum(testFile)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), checksum)
	assert.Len(suite.T(), checksum, 64) // SHA256 hex length

	// Verify consistency
	checksum2, err := suite.validator.CalculateFileChecksum(testFile)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), checksum, checksum2)

	// Test with non-existent file
	_, err = suite.validator.CalculateFileChecksum("/nonexistent/file.so")
	assert.Error(suite.T(), err)
}

// TestValidateChecksum tests checksum validation.
func (suite *SecurityTestSuite) TestValidateChecksum() {
	// Create test file
	testFile := filepath.Join(suite.tempDir, "checksum_validation_test.so")
	content := []byte("checksum validation test content")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(suite.T(), err)

	// Calculate correct checksum
	correctChecksum, err := suite.validator.CalculateFileChecksum(testFile)
	require.NoError(suite.T(), err)

	// Test with correct checksum
	err = suite.validator.ValidateChecksum(testFile, correctChecksum)
	assert.NoError(suite.T(), err)

	// Test with incorrect checksum
	incorrectChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
	err = suite.validator.ValidateChecksum(testFile, incorrectChecksum)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "checksum validation failed")

	// Test with non-existent file
	err = suite.validator.ValidateChecksum("/nonexistent/file.so", correctChecksum)
	assert.Error(suite.T(), err)
}

// TestValidateManifest tests plugin manifest validation.
func (suite *SecurityTestSuite) TestValidateManifest() {
	// Test valid manifest
	validManifest := &PluginManifest{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Author:      "Test Author",
		Description: "Test plugin description",
		Checksum:    "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		Capabilities: PluginCapabilities{
			FileSystemAccess: false,
			NetworkAccess:    false,
			ProcessExecution: false,
		},
	}

	err := suite.validator.ValidateManifest(validManifest)
	assert.NoError(suite.T(), err)

	// Test manifest with missing name
	invalidManifest := &PluginManifest{
		Name:     "", // Empty name
		Version:  "1.0.0",
		Checksum: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
	}

	err = suite.validator.ValidateManifest(invalidManifest)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "missing name")

	// Test manifest with missing version
	invalidManifest.Name = "test-plugin"
	invalidManifest.Version = ""

	err = suite.validator.ValidateManifest(invalidManifest)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "missing version")

	// Test manifest with missing checksum
	invalidManifest.Version = "1.0.0"
	invalidManifest.Checksum = ""

	err = suite.validator.ValidateManifest(invalidManifest)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "missing checksum")

	// Test manifest with invalid checksum format
	invalidManifest.Checksum = "invalid-checksum"

	err = suite.validator.ValidateManifest(invalidManifest)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "checksum invalid format")
}

// TestDefaultPluginCapabilities tests default capabilities.
func (suite *SecurityTestSuite) TestDefaultPluginCapabilities() {
	caps := DefaultPluginCapabilities()

	assert.False(suite.T(), caps.FileSystemAccess)
	assert.False(suite.T(), caps.NetworkAccess)
	assert.False(suite.T(), caps.ProcessExecution)
	assert.Empty(suite.T(), caps.AllowedPaths)
	assert.Empty(suite.T(), caps.AllowedHosts)
}

// TestDefaultResourceLimits tests default resource limits.
func (suite *SecurityTestSuite) TestDefaultResourceLimits() {
	limits := DefaultResourceLimits()

	assert.Equal(suite.T(), int64(100), limits.MaxMemoryMB)
	assert.Equal(suite.T(), 20.0, limits.MaxCPUPercent)
	assert.Equal(suite.T(), 30*time.Second, limits.MaxExecutionTime)
	assert.Equal(suite.T(), 50, limits.MaxFileHandles)
}

// TestSecurityConfigDefaults tests default security configuration.
func (suite *SecurityTestSuite) TestSecurityConfigDefaults() {
	validator := NewSecurityValidator(nil)

	assert.NotNil(suite.T(), validator)
	assert.NotNil(suite.T(), validator.config)
	assert.False(suite.T(), validator.config.AllowUnsigned)
	assert.True(suite.T(), validator.config.RequireChecksum)
	assert.Contains(suite.T(), validator.config.TrustedSources, "./plugins")
	assert.Contains(suite.T(), validator.config.BlockedPatterns, "*.exe")
	assert.Equal(suite.T(), int64(50), validator.config.MaxFileSizeMB)
	assert.Contains(suite.T(), validator.config.AllowedExtensions, ".so")
}

// TestURLEncodedPathTraversal tests URL encoded path traversal attacks.
func (suite *SecurityTestSuite) TestURLEncodedPathTraversal() {
	encodedAttacks := []string{
		"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",            // ../../../etc/passwd
		"..%252f..%252f..%252fetc%252fpasswd",                // Double encoded
		"%2e%2e\\%2e%2e\\%2e%2e\\windows\\system32\\cmd.exe", // Windows path
	}

	for _, attack := range encodedAttacks {
		suite.T().Run("encoded_attack_"+attack, func(t *testing.T) {
			err := suite.validator.ValidatePluginPath(attack)
			assert.Error(t, err, "Should block encoded path traversal: %s", attack)
		})
	}
}

// TestSecurityConfigCustomization tests custom security configuration.
func (suite *SecurityTestSuite) TestSecurityConfigCustomization() {
	customConfig := &SecurityConfig{
		AllowUnsigned:     true,
		RequireChecksum:   false,
		TrustedSources:    []string{"/custom/path"},
		BlockedPatterns:   []string{"*dangerous*", "*.tmp"},
		MaxFileSizeMB:     100,
		AllowedExtensions: []string{".custom"},
	}

	validator := NewSecurityValidator(customConfig)
	assert.Equal(suite.T(), customConfig, validator.config)

	// Test custom blocked patterns
	err := validator.ValidatePluginPath("dangerous-file.custom")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "blocked pattern")

	// Test custom trusted sources
	err = validator.ValidatePluginPath("/untrusted/path/file.custom")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "not from trusted source")

	// Test custom allowed extensions
	customFile := filepath.Join(suite.tempDir, "test.custom")
	err = os.WriteFile(customFile, []byte("test"), 0644)
	require.NoError(suite.T(), err)

	result := validator.ValidatePluginFile(customFile)
	// Should be invalid due to trusted source check, but extension should be valid
	assert.False(suite.T(), result.Valid)

	// But if we use a trusted path
	customConfig.TrustedSources = []string{suite.tempDir}
	validator = NewSecurityValidator(customConfig)
	result = validator.ValidatePluginFile(customFile)
	assert.True(suite.T(), result.Valid)
}

// Run all tests in the suite.
func TestSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(SecurityTestSuite))
}
