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
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
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

	// EnableInterpolation enables ${VAR} and ${VAR:-default} interpolation in file values.
	EnableInterpolation bool

	// SecretFileSuffix enables reading secrets from files using VAR+SecretFileSuffix (default: _FILE).
	// When both VAR and VAR+SecretFileSuffix are set, load will fail fast to avoid ambiguity.
	SecretFileSuffix string
}

// DefaultOptions returns sane defaults.
func DefaultOptions() Options {
	return Options{
		WorkDir:             ".",
		ConfigBaseName:      "swit",
		ConfigType:          "yaml", // set to "auto" to enable extension auto-detection (yaml/yml/json)
		EnvironmentName:     "",
		OverrideFilename:    "swit.override.yaml",
		EnvPrefix:           "SWIT",
		EnableAutomaticEnv:  true,
		EnableInterpolation: true,
		SecretFileSuffix:    "_FILE",
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
	if options.SecretFileSuffix == "" {
		options.SecretFileSuffix = "_FILE"
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

	// 1) Base layer: swit.{yaml|json}
	if err := m.mergeFirstExistingFile(m.fileCandidatesFor(BaseLayer)); err != nil {
		return fmt.Errorf("load base config: %w", err)
	}

	// 2) Environment file layer: swit.<env>.{yaml|json}
	if m.options.EnvironmentName != "" {
		if err := m.mergeFirstExistingFile(m.fileCandidatesFor(EnvironmentFileLayer)); err != nil {
			return fmt.Errorf("load env config: %w", err)
		}
	}

	// 3) Override file layer: swit.override.{yaml|json}
	if err := m.mergeFirstExistingFile(m.fileCandidatesFor(OverrideFileLayer)); err != nil {
		return fmt.Errorf("load override config: %w", err)
	}

	// 3.5) Interpolate variables within loaded file values if enabled
	if m.options.EnableInterpolation {
		if err := m.interpolateLoadedValues(); err != nil {
			return fmt.Errorf("interpolate values: %w", err)
		}
	}

	// 3.6) Process *_FILE secret environment variables into standard vars before env overlay
	if err := m.processSecretEnvFiles(); err != nil {
		return fmt.Errorf("process secret env files: %w", err)
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
func (m *Manager) filePathFor(layer Layer) string { // kept for backward-compat and single-type flows
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

// fileCandidatesFor returns candidate file paths for a layer, allowing extension auto-detection when ConfigType=="auto".
func (m *Manager) fileCandidatesFor(layer Layer) []string {
	dir := m.options.WorkDir
	base := m.options.ConfigBaseName

	// determine candidate extensions in precedence order
	var exts []string
	switch strings.ToLower(m.options.ConfigType) {
	case "auto", "":
		exts = []string{"yaml", "yml", "json"}
	default:
		exts = []string{m.normalizedConfigExt()}
	}

	// build candidate names
	switch layer {
	case BaseLayer:
		// swit.yaml | swit.yml | swit.json
		candidates := make([]string, 0, len(exts))
		for _, ext := range exts {
			candidates = append(candidates, filepath.Join(dir, fmt.Sprintf("%s.%s", base, ext)))
		}
		return candidates
	case EnvironmentFileLayer:
		// swit.<env>.yaml | swit.<env>.yml | swit.<env>.json
		env := strings.ToLower(m.options.EnvironmentName)
		candidates := make([]string, 0, len(exts))
		for _, ext := range exts {
			candidates = append(candidates, filepath.Join(dir, fmt.Sprintf("%s.%s.%s", base, env, ext)))
		}
		return candidates
	case OverrideFileLayer:
		// explicit override filename respects user-provided value
		if name := strings.TrimSpace(m.options.OverrideFilename); name != "" {
			// If name has a known config extension, treat as exact; otherwise treat as base name
			if ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), "."); ext == "yaml" || ext == "yml" || ext == "json" || ext == "toml" || ext == "hcl" {
				return []string{filepath.Join(dir, name)}
			}
			// otherwise, try with candidate extensions
			candidates := make([]string, 0, len(exts))
			for _, ext := range exts {
				candidates = append(candidates, filepath.Join(dir, fmt.Sprintf("%s.%s", name, ext)))
			}
			return candidates
		}
		// default override file name uses base + ".override"
		candidates := make([]string, 0, len(exts))
		for _, ext := range exts {
			candidates = append(candidates, filepath.Join(dir, fmt.Sprintf("%s.override.%s", base, ext)))
		}
		return candidates
	default:
		return nil
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
		// keep yaml as default for backward compatibility
		return "yaml"
	}
}

// mergeFirstExistingFile tries candidates in order and merges the first existing one.
// Missing files are ignored; parse errors are returned with actionable context.
func (m *Manager) mergeFirstExistingFile(candidates []string) error {
	for _, path := range candidates {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		return m.mergeFile(path)
	}
	return nil
}

// mergeFile merges a specific configuration file, auto-selecting parser by its extension.
func (m *Manager) mergeFile(path string) error {
	// Read file into a temporary viper to avoid changing base settings on read errors
	tmp := viper.New()
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	switch ext {
	case "yml":
		ext = "yaml"
	case "yaml", "json", "toml", "hcl":
		// ok
	case "":
		// fallback to configured default
		ext = m.normalizedConfigExt()
	default:
		// unknown extension, but still try configured default to be lenient
		ext = m.normalizedConfigExt()
	}
	tmp.SetConfigType(ext)

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

// UnmarshalStrict binds settings into target like Unmarshal, but fails on unknown fields.
// It provides actionable errors to help users fix configuration mismatches.
func (m *Manager) UnmarshalStrict(target interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if target == nil {
		return errors.New("target must not be nil")
	}
	if err := m.v.UnmarshalExact(target); err != nil {
		return fmt.Errorf("configuration validation failed (unknown or invalid fields): %w", err)
	}
	return nil
}

// RequireKeys ensures a list of keys exist after merge; returns aggregated error for missing keys.
func (m *Manager) RequireKeys(keys ...string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(keys) == 0 {
		return nil
	}
	missing := make([]string, 0)
	for _, k := range keys {
		if !m.v.IsSet(k) {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration keys: %s", strings.Join(missing, ", "))
	}
	return nil
}

// interpolateLoadedValues expands ${VAR} and ${VAR:-default} and $VAR within string values
// already loaded from configuration files. It does not affect environment override semantics.
func (m *Manager) interpolateLoadedValues() error {
	settings := m.v.AllSettings()
	expanded := expandValues(settings)
	expandedMap, ok := expanded.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected expanded settings type: %T", expanded)
	}
	return m.v.MergeConfigMap(expandedMap)
}

var (
	defaultPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\:\-([^}]*)\}`)
	// ${VAR} pattern (without default). We'll run after defaultPattern to avoid double-handling.
	simpleBracedPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
)

func expandStringEnv(s string) string {
	if s == "" {
		return s
	}
	// First handle ${VAR:-default}
	s = defaultPattern.ReplaceAllStringFunc(s, func(match string) string {
		subs := defaultPattern.FindStringSubmatch(match)
		if len(subs) != 3 {
			return match
		}
		name := subs[1]
		def := subs[2]
		val, ok := os.LookupEnv(name)
		if ok && strings.TrimSpace(val) != "" {
			return val
		}
		return def
	})
	// Then handle ${VAR}
	s = simpleBracedPattern.ReplaceAllStringFunc(s, func(match string) string {
		subs := simpleBracedPattern.FindStringSubmatch(match)
		if len(subs) != 2 {
			return match
		}
		name := subs[1]
		if val, ok := os.LookupEnv(name); ok {
			return val
		}
		return ""
	})
	// Finally handle $VAR via standard expansion
	return os.ExpandEnv(s)
}

func expandValues(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, vv := range val {
			out[k] = expandValues(vv)
		}
		return out
	case map[interface{}]interface{}: // in case YAML parser produced this shape
		out := make(map[string]interface{}, len(val))
		for k, vv := range val {
			key := fmt.Sprintf("%v", k)
			out[key] = expandValues(vv)
		}
		return out
	case []interface{}:
		for i := range val {
			val[i] = expandValues(val[i])
		}
		return val
	case string:
		return strings.TrimSuffix(expandStringEnv(val), "\n")
	default:
		return v
	}
}

// processSecretEnvFiles processes VAR+SecretFileSuffix env vars by reading the file
// content into VAR (trailing newline trimmed). If both VAR and VAR+SecretFileSuffix
// are set, returns an error to avoid ambiguity.
func (m *Manager) processSecretEnvFiles() error {
	prefix := strings.TrimSpace(m.options.EnvPrefix)
	if prefix != "" {
		prefix += "_"
	}
	suffix := m.options.SecretFileSuffix
	if suffix == "" {
		suffix = "_FILE"
	}

	// Collect presence of VAR and VAR_FILE
	has := make(map[string]bool)
	hasFile := make(map[string]bool)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		name := parts[0]
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		if strings.HasSuffix(name, suffix) {
			base := strings.TrimSuffix(name, suffix)
			hasFile[base] = true
			continue
		}
		has[name] = true
	}

	// Detect conflicts
	for base := range hasFile {
		if has[base] {
			return fmt.Errorf("both %s and %s%s are set; please set only one", base, base, suffix)
		}
	}

	// Apply file values
	for base := range hasFile {
		fileVar := base + suffix
		path := os.Getenv(fileVar)
		if strings.TrimSpace(path) == "" {
			continue
		}
		data, err := ioutil.ReadFile(path) // use ioutil for Go <1.20 compat
		if err != nil {
			return fmt.Errorf("read secret file %s: %w", path, err)
		}
		val := strings.TrimSuffix(string(data), "\n")
		if err := os.Setenv(base, val); err != nil {
			return fmt.Errorf("set env %s from file: %w", base, err)
		}
	}
	return nil
}
