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

package plugin

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// SecurityConfig represents security configuration for plugin validation.
type SecurityConfig struct {
	AllowUnsigned     bool     `yaml:"allow_unsigned" json:"allow_unsigned"`
	RequireChecksum   bool     `yaml:"require_checksum" json:"require_checksum"`
	TrustedSources    []string `yaml:"trusted_sources" json:"trusted_sources"`
	BlockedPatterns   []string `yaml:"blocked_patterns" json:"blocked_patterns"`
	MaxFileSizeMB     int64    `yaml:"max_file_size_mb" json:"max_file_size_mb"`
	AllowedExtensions []string `yaml:"allowed_extensions" json:"allowed_extensions"`
}

// SecurityValidator handles plugin security validation.
type SecurityValidator struct {
	config *SecurityConfig
}

// NewSecurityValidator creates a new security validator with configuration.
func NewSecurityValidator(config *SecurityConfig) *SecurityValidator {
	if config == nil {
		config = &SecurityConfig{
			AllowUnsigned:     false,
			RequireChecksum:   true,
			TrustedSources:    []string{"./plugins", "/opt/switctl/plugins"},
			BlockedPatterns:   []string{"*.exe", "*.bat", "*.cmd"},
			MaxFileSizeMB:     50,
			AllowedExtensions: []string{".so", ".dll", ".dylib"},
		}
	}

	return &SecurityValidator{
		config: config,
	}
}

// ValidatePluginPath validates that a plugin path is safe and trusted.
func (sv *SecurityValidator) ValidatePluginPath(path string) error {
	// Check for null bytes (potential injection attack)
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null byte")
	}

	// Try to URL decode the path to catch encoded directory traversal attempts
	decoded, err := url.QueryUnescape(path)
	if err == nil {
		path = decoded
	}

	// Clean the path to resolve any .. or . elements
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") || strings.Contains(path, "..") {
		return fmt.Errorf("path contains directory traversal")
	}

	// Check blocked patterns
	for _, pattern := range sv.config.BlockedPatterns {
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			return fmt.Errorf("path matches blocked pattern: %s", pattern)
		}
	}

	// If trusted sources are configured, check if path matches any
	if len(sv.config.TrustedSources) > 0 {
		isTrusted := false
		for _, trusted := range sv.config.TrustedSources {
			if strings.HasPrefix(path, trusted) {
				isTrusted = true
				break
			}
		}

		if !isTrusted {
			return fmt.Errorf("path is not from trusted source")
		}
	}

	return nil
}

// ValidatePluginFile validates a plugin file for security.
func (sv *SecurityValidator) ValidatePluginFile(path string) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	// Validate the path first
	if err := sv.ValidatePluginPath(path); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:    "INVALID_PATH",
			Message: err.Error(),
			Level:   "error",
		})
		return result
	}

	// Check if file exists and is a regular file
	fileInfo, err := os.Stat(path)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:    "FILE_NOT_FOUND",
			Message: "Plugin file not found",
			Level:   "error",
		})
		return result
	}

	if !fileInfo.Mode().IsRegular() {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:    "NOT_REGULAR_FILE",
			Message: "Plugin path is not a regular file",
			Level:   "error",
		})
		return result
	}

	result.Size = fileInfo.Size()
	result.Permissions = fileInfo.Mode()

	// Check file size
	maxSize := sv.config.MaxFileSizeMB * 1024 * 1024
	if fileInfo.Size() > maxSize {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:    "FILE_TOO_LARGE",
			Message: fmt.Sprintf("Plugin file too large: %d bytes (max: %d MB)", fileInfo.Size(), sv.config.MaxFileSizeMB),
			Level:   "error",
		})
	}

	// Check file extension
	ext := filepath.Ext(path)
	isAllowedExt := false
	for _, allowedExt := range sv.config.AllowedExtensions {
		if ext == allowedExt {
			isAllowedExt = true
			break
		}
	}

	if !isAllowedExt {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:    "INVALID_EXTENSION",
			Message: fmt.Sprintf("Plugin file extension not allowed: %s", ext),
			Level:   "error",
		})
	}

	// Calculate checksum if file is valid so far
	if result.Valid {
		checksum, err := sv.calculateChecksum(path)
		if err == nil {
			result.Checksum = checksum
		} else {
			result.Warnings = append(result.Warnings, ValidationError{
				Code:    "CHECKSUM_FAILED",
				Message: "Failed to calculate checksum",
				Level:   "warning",
			})
		}
	}

	return result
}

// calculateChecksum calculates SHA256 checksum of a file.
func (sv *SecurityValidator) calculateChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// CalculateFileChecksum calculates SHA256 checksum of a file.
func (sv *SecurityValidator) CalculateFileChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// ValidateChecksum validates a plugin's checksum against expected value.
func (sv *SecurityValidator) ValidateChecksum(path, expectedChecksum string) error {
	actualChecksum, err := sv.CalculateFileChecksum(path)
	if err != nil {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginLoadError,
			"Failed to calculate plugin checksum",
			err,
		)
	}

	if subtle.ConstantTimeCompare([]byte(actualChecksum), []byte(expectedChecksum)) != 1 {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginLoadError,
			"Plugin checksum validation failed",
			fmt.Errorf("expected: %s, got: %s", expectedChecksum, actualChecksum),
		)
	}

	return nil
}

// PluginManifest represents plugin metadata and security information.
type PluginManifest struct {
	Name         string             `json:"name" yaml:"name"`
	Version      string             `json:"version" yaml:"version"`
	Author       string             `json:"author" yaml:"author"`
	Description  string             `json:"description" yaml:"description"`
	Checksum     string             `json:"checksum" yaml:"checksum"`
	Signature    string             `json:"signature,omitempty" yaml:"signature,omitempty"`
	Capabilities PluginCapabilities `json:"capabilities" yaml:"capabilities"`
	Dependencies []string           `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	MinVersion   string             `json:"min_version,omitempty" yaml:"min_version,omitempty"`
	MaxVersion   string             `json:"max_version,omitempty" yaml:"max_version,omitempty"`
}

// PluginCapabilities defines what capabilities a plugin is allowed to use.
type PluginCapabilities struct {
	FileSystemAccess bool     `json:"file_system_access"`
	NetworkAccess    bool     `json:"network_access"`
	ProcessExecution bool     `json:"process_execution"`
	AllowedPaths     []string `json:"allowed_paths"`
	AllowedHosts     []string `json:"allowed_hosts"`
}

// ValidateManifest validates a plugin manifest for security and completeness.
func (sv *SecurityValidator) ValidateManifest(manifest *PluginManifest) error {
	if manifest.Name == "" {
		return interfaces.NewPluginError(
			interfaces.ErrCodeInvalidInput,
			"Plugin manifest missing name",
			nil,
		)
	}

	if manifest.Version == "" {
		return interfaces.NewPluginError(
			interfaces.ErrCodeInvalidInput,
			"Plugin manifest missing version",
			nil,
		)
	}

	if manifest.Checksum == "" {
		return interfaces.NewPluginError(
			interfaces.ErrCodeInvalidInput,
			"Plugin manifest missing checksum",
			nil,
		)
	}

	// Validate checksum format (should be 64-character hex string for SHA256)
	if len(manifest.Checksum) != 64 {
		return interfaces.NewPluginError(
			interfaces.ErrCodeInvalidInput,
			"Plugin manifest checksum invalid format",
			nil,
		)
	}

	return nil
}

// DefaultPluginCapabilities returns default (restrictive) capabilities.
func DefaultPluginCapabilities() PluginCapabilities {
	return PluginCapabilities{
		FileSystemAccess: false,
		NetworkAccess:    false,
		ProcessExecution: false,
		AllowedPaths:     []string{},
		AllowedHosts:     []string{},
	}
}

// ResourceLimits defines resource usage limits for plugins.
type ResourceLimits struct {
	MaxMemoryMB      int64         `json:"max_memory_mb"`
	MaxCPUPercent    float64       `json:"max_cpu_percent"`
	MaxExecutionTime time.Duration `json:"max_execution_time"`
	MaxFileHandles   int           `json:"max_file_handles"`
}

// DefaultResourceLimits returns default resource limits for plugins.
func DefaultResourceLimits() ResourceLimits {
	return ResourceLimits{
		MaxMemoryMB:      100,              // 100MB max memory
		MaxCPUPercent:    20.0,             // 20% max CPU
		MaxExecutionTime: 30 * time.Second, // 30 seconds max execution
		MaxFileHandles:   50,               // 50 max file handles
	}
}
