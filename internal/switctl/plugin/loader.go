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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plugin"
	"runtime"
	"strings"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// PluginLoader defines the interface for loading plugins.
type PluginLoader interface {
	// LoadPlugin loads a plugin from the specified path.
	LoadPlugin(ctx context.Context, path string) (interfaces.Plugin, error)
	// ValidatePlugin validates a plugin before loading.
	ValidatePlugin(path string) error
	// GetSupportedFormats returns supported plugin formats.
	GetSupportedFormats() []string
	// CalculateChecksum calculates checksum for a plugin file.
	CalculateChecksum(path string) (string, error)
}

// GoPluginLoader implements PluginLoader for Go plugins.
type GoPluginLoader struct {
	logger         interfaces.Logger
	securityPolicy *SecurityPolicy
	validator      *PluginValidator
	loadedPlugins  map[string]*plugin.Plugin
}

// NewGoPluginLoader creates a new Go plugin loader.
func NewGoPluginLoader(logger interfaces.Logger) *GoPluginLoader {
	return &GoPluginLoader{
		logger: logger,
		securityPolicy: &SecurityPolicy{
			AllowUnsigned:   false,
			RequireChecksum: true,
		},
		validator: &PluginValidator{
			maxFileSize:       50 * 1024 * 1024, // 50MB
			allowedExtensions: []string{".so", ".dll", ".dylib"},
			allowedSymbols:    []string{"Plugin", "GetPlugin", "NewPlugin"},
		},
		loadedPlugins: make(map[string]*plugin.Plugin),
	}
}

// LoadPlugin loads a plugin from the specified path.
func (l *GoPluginLoader) LoadPlugin(ctx context.Context, path string) (interfaces.Plugin, error) {
	startTime := time.Now()

	l.logger.Debug("Loading plugin from path", "path", path)

	// Validate plugin first
	if err := l.ValidatePlugin(path); err != nil {
		return nil, interfaces.NewPluginError(
			interfaces.ErrCodePluginLoadError,
			"Plugin validation failed",
			err,
		)
	}

	// Check if plugin is already loaded
	if existingPlugin, exists := l.loadedPlugins[path]; exists {
		l.logger.Debug("Plugin already loaded, returning existing instance", "path", path)
		return l.extractPluginInterface(existingPlugin)
	}

	// Load the plugin with timeout
	pluginChan := make(chan *plugin.Plugin, 1)
	errChan := make(chan error, 1)

	go func() {
		p, err := plugin.Open(path)
		if err != nil {
			errChan <- err
			return
		}
		pluginChan <- p
	}()

	select {
	case <-ctx.Done():
		return nil, interfaces.NewPluginError(
			interfaces.ErrCodePluginLoadError,
			"Plugin loading timed out",
			ctx.Err(),
		)
	case err := <-errChan:
		return nil, interfaces.NewPluginError(
			interfaces.ErrCodePluginLoadError,
			fmt.Sprintf("Failed to open plugin: %s", path),
			err,
		)
	case p := <-pluginChan:
		l.loadedPlugins[path] = p

		pluginInterface, err := l.extractPluginInterface(p)
		if err != nil {
			delete(l.loadedPlugins, path)
			return nil, err
		}

		loadTime := time.Since(startTime)
		l.logger.Info("Plugin loaded successfully",
			"path", path,
			"load_time", loadTime,
			"name", pluginInterface.Name(),
			"version", pluginInterface.Version())

		return pluginInterface, nil
	}
}

// ValidatePlugin validates a plugin before loading.
func (l *GoPluginLoader) ValidatePlugin(path string) error {
	result, err := l.validatePluginDetailed(path)
	if err != nil {
		return err
	}

	if !result.Valid {
		if len(result.Errors) > 0 {
			return fmt.Errorf("plugin validation failed: %s", result.Errors[0].Message)
		}
		return fmt.Errorf("plugin validation failed")
	}

	// Log warnings if any
	for _, warning := range result.Warnings {
		l.logger.Warn("Plugin validation warning",
			"path", path,
			"code", warning.Code,
			"message", warning.Message)
	}

	return nil
}

// validatePluginDetailed performs detailed plugin validation.
func (l *GoPluginLoader) validatePluginDetailed(path string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid: true,
	}

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Code:    "FILE_NOT_FOUND",
				Message: fmt.Sprintf("Plugin file not found: %s", path),
				Level:   "error",
			})
			return result, nil
		}
		return nil, fmt.Errorf("failed to stat plugin file: %w", err)
	}

	result.Size = info.Size()
	result.Permissions = info.Mode()

	// Validate file extension
	if !l.validator.isValidExtension(path) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:    "INVALID_EXTENSION",
			Message: fmt.Sprintf("Invalid plugin file extension: %s", filepath.Ext(path)),
			Level:   "error",
		})
	}

	// Check file size
	if l.validator.maxFileSize > 0 && info.Size() > l.validator.maxFileSize {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:    "FILE_TOO_LARGE",
			Message: fmt.Sprintf("Plugin file too large: %d bytes (max: %d)", info.Size(), l.validator.maxFileSize),
			Level:   "error",
		})
	}

	// Calculate checksum
	checksum, err := l.CalculateChecksum(path)
	if err != nil {
		result.Warnings = append(result.Warnings, ValidationError{
			Code:    "CHECKSUM_FAILED",
			Message: fmt.Sprintf("Failed to calculate checksum: %v", err),
			Level:   "warning",
		})
	} else {
		result.Checksum = checksum
	}

	// Basic shared library format validation
	if err := l.validateSharedLibraryFormat(path); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:    "INVALID_FORMAT",
			Message: fmt.Sprintf("Invalid shared library format: %v", err),
			Level:   "error",
		})
	}

	// Validate against blocked patterns
	if l.validator.isBlocked(path) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Code:    "BLOCKED_PATTERN",
			Message: "Plugin path matches blocked pattern",
			Level:   "error",
		})
	}

	return result, nil
}

// validateSharedLibraryFormat performs basic validation of shared library format
func (l *GoPluginLoader) validateSharedLibraryFormat(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	// Read first few bytes to check magic numbers
	header := make([]byte, 16)
	n, err := file.Read(header)
	if err != nil || n < 4 {
		return fmt.Errorf("cannot read file header")
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".so", ".dylib":
		// Check for ELF magic number (0x7f 'E' 'L' 'F')
		if len(header) >= 4 && header[0] == 0x7f && header[1] == 'E' && header[2] == 'L' && header[3] == 'F' {
			return nil // Valid ELF file
		}
		// Check for Mach-O magic numbers for .dylib
		if len(header) >= 4 {
			// Mach-O 32-bit: 0xfeedface, 0xcefaedfe
			// Mach-O 64-bit: 0xfeedfacf, 0xcffaedfe
			magic := uint32(header[0]) | uint32(header[1])<<8 | uint32(header[2])<<16 | uint32(header[3])<<24
			if magic == 0xfeedface || magic == 0xcefaedfe || magic == 0xfeedfacf || magic == 0xcffaedfe {
				return nil // Valid Mach-O file
			}
		}
		return fmt.Errorf("not a valid shared library file")
	case ".dll":
		// Check for PE magic number ("MZ")
		if len(header) >= 2 && header[0] == 'M' && header[1] == 'Z' {
			return nil // Valid PE file
		}
		return fmt.Errorf("not a valid DLL file")
	default:
		return fmt.Errorf("unsupported file extension: %s", ext)
	}
}

// GetSupportedFormats returns supported plugin formats.
func (l *GoPluginLoader) GetSupportedFormats() []string {
	switch runtime.GOOS {
	case "linux":
		return []string{".so"}
	case "darwin":
		return []string{".dylib", ".so"}
	case "windows":
		return []string{".dll"}
	default:
		return []string{".so"}
	}
}

// CalculateChecksum calculates SHA256 checksum for a plugin file.
func (l *GoPluginLoader) CalculateChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// extractPluginInterface extracts the Plugin interface from a loaded Go plugin.
func (l *GoPluginLoader) extractPluginInterface(p *plugin.Plugin) (interfaces.Plugin, error) {
	// Try different symbol names
	symbolNames := []string{"Plugin", "GetPlugin", "NewPlugin"}

	for _, symbolName := range symbolNames {
		symbol, err := p.Lookup(symbolName)
		if err != nil {
			l.logger.Debug("Symbol not found", "symbol", symbolName, "error", err)
			continue
		}

		// Try different symbol types
		if pluginFunc, ok := symbol.(func() interfaces.Plugin); ok {
			return pluginFunc(), nil
		}

		if pluginInstance, ok := symbol.(interfaces.Plugin); ok {
			return pluginInstance, nil
		}

		if pluginFactory, ok := symbol.(*func() interfaces.Plugin); ok {
			return (*pluginFactory)(), nil
		}

		l.logger.Debug("Symbol found but wrong type", "symbol", symbolName, "type", fmt.Sprintf("%T", symbol))
	}

	return nil, interfaces.NewPluginError(
		interfaces.ErrCodePluginLoadError,
		"Plugin does not export a valid Plugin interface",
		fmt.Errorf("tried symbols: %v", symbolNames),
	)
}

// isValidExtension checks if the file extension is valid for plugins.
func (pv *PluginValidator) isValidExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, allowedExt := range pv.allowedExtensions {
		if ext == allowedExt {
			return true
		}
	}
	return false
}

// isBlocked checks if the plugin path matches any blocked patterns.
func (pv *PluginValidator) isBlocked(path string) bool {
	filename := filepath.Base(path)
	for _, pattern := range pv.blockedPatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
	}
	return false
}
