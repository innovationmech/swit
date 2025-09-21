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

package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// Layer represents a configuration layer in the hierarchy.
//
// Precedence (low → high): Defaults < Base < EnvironmentFile < OverrideFile < EnvironmentVariables
type Layer int

const (
	// DefaultsLayer holds hard-coded default values set via SetDefault.
	DefaultsLayer Layer = iota
	// BaseLayer is the base configuration file, typically committed (e.g., swit.yaml).
	BaseLayer
	// EnvironmentFileLayer is the environment-specific file (e.g., swit.dev.yaml, swit.prod.yaml).
	EnvironmentFileLayer
	// OverrideFileLayer is a developer/operator local override file (e.g., swit.override.yaml).
	OverrideFileLayer
	// EnvironmentVariablesLayer represents environment variables (highest precedence).
	EnvironmentVariablesLayer
)

// Options configures the Manager.
type Options struct {
	// WorkDir is the working directory to resolve relative config file paths.
	WorkDir string

	// ConfigBaseName is the base name of the configuration file without extension (default: "swit").
	ConfigBaseName string

	// ConfigType is the configuration file type (yaml|yml|json). Default: "yaml".
	ConfigType string

	// EnvironmentName selects the environment file suffix, e.g., "dev" → swit.dev.yaml.
	EnvironmentName string

	// OverrideFilename is the optional override file name. Default: "swit.override.yaml".
	OverrideFilename string

	// EnvPrefix is the prefix for environment variables (e.g., "SWIT").
	EnvPrefix string

	// EnableAutomaticEnv enables automatic env var binding with dot→underscore mapping.
	EnableAutomaticEnv bool
}

// DefaultOptions returns sane defaults.
func DefaultOptions() Options {
	return Options{
		WorkDir:            ".",
		ConfigBaseName:     "swit",
		ConfigType:         "yaml",
		EnvironmentName:    "",
		OverrideFilename:   "swit.override.yaml",
		EnvPrefix:          "SWIT",
		EnableAutomaticEnv: true,
	}
}

// Manager provides hierarchical configuration loading, merging and access.
// It uses an internal viper instance with controlled merge order.
type Manager struct {
	mu      sync.RWMutex
	v       *viper.Viper
	options Options
}

// NewManager creates a new Manager with the given options.
func NewManager(options Options) *Manager {
	v := viper.New()
	if options.ConfigType == "" {
		options.ConfigType = "yaml"
	}
	if options.ConfigBaseName == "" {
		options.ConfigBaseName = "swit"
	}
	if options.WorkDir == "" {
		options.WorkDir = "."
	}

	// Configure environment variables
	if options.EnableAutomaticEnv {
		if options.EnvPrefix != "" {
			v.SetEnvPrefix(options.EnvPrefix)
		}
		v.AutomaticEnv()
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	}

	return &Manager{v: v, options: options}
}

// SetDefault sets a default value for the given key.
func (m *Manager) SetDefault(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.v.SetDefault(key, value)
}

// Load loads and merges all configured layers in precedence order.
// Defaults are already in viper via SetDefault; the method merges files in order and finally applies env vars.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1) Base layer: swit.yaml
	if err := m.mergeFileIfExists(m.filePathFor(BaseLayer)); err != nil {
		return fmt.Errorf("load base config: %w", err)
	}

	// 2) Environment file layer: swit.<env>.yaml
	if m.options.EnvironmentName != "" {
		if err := m.mergeFileIfExists(m.filePathFor(EnvironmentFileLayer)); err != nil {
			return fmt.Errorf("load env config: %w", err)
		}
	}

	// 3) Override file layer: swit.override.yaml
	if err := m.mergeFileIfExists(m.filePathFor(OverrideFileLayer)); err != nil {
		return fmt.Errorf("load override config: %w", err)
	}

	// 4) Environment variables layer is automatic via v.AutomaticEnv
	return nil
}

// Unmarshal binds all merged settings into the given struct pointer.
func (m *Manager) Unmarshal(target interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if target == nil {
		return errors.New("target must not be nil")
	}
	return m.v.Unmarshal(target)
}

// Get returns a value by key from merged configuration.
func (m *Manager) Get(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.v.Get(key)
}

// AllSettings returns a copy of all merged settings as a map.
func (m *Manager) AllSettings() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.v.AllSettings()
}

// MergeConfigMap allows callers to merge an arbitrary settings map with low precedence.
// Later file merges and environment variables can still override these values.
func (m *Manager) MergeConfigMap(settings map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.v.MergeConfigMap(settings)
}

// filePathFor returns the absolute file path for a given layer.
func (m *Manager) filePathFor(layer Layer) string {
	dir := m.options.WorkDir
	base := m.options.ConfigBaseName
	switch layer {
	case BaseLayer:
		return filepath.Join(dir, fmt.Sprintf("%s.%s", base, m.normalizedConfigExt()))
	case EnvironmentFileLayer:
		env := m.options.EnvironmentName
		return filepath.Join(dir, fmt.Sprintf("%s.%s.%s", base, strings.ToLower(env), m.normalizedConfigExt()))
	case OverrideFileLayer:
		name := m.options.OverrideFilename
		if name == "" {
			name = fmt.Sprintf("%s.override.%s", base, m.normalizedConfigExt())
		}
		return filepath.Join(dir, name)
	default:
		return ""
	}
}

func (m *Manager) normalizedConfigExt() string {
	t := strings.ToLower(m.options.ConfigType)
	switch t {
	case "yml":
		return "yaml"
	case "yaml", "json", "toml", "hcl":
		return t
	default:
		return "yaml"
	}
}

// mergeFileIfExists merges a configuration file if it exists. Missing files are ignored.
func (m *Manager) mergeFileIfExists(path string) error {
	if path == "" {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Read file into a temporary viper to avoid changing base settings on read errors
	tmp := viper.New()
	tmp.SetConfigType(m.normalizedConfigExt())

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := tmp.ReadConfig(bytes.NewReader(content)); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	// Merge using parsed map to avoid relying on base viper's ConfigType
	return m.v.MergeConfigMap(tmp.AllSettings())
}
