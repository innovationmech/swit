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
	"os"
	"sync"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// ManagerConfig represents configuration for the plugin manager.
type ManagerConfig struct {
	PluginDirs            []string       `yaml:"plugin_dirs" json:"plugin_dirs"`
	SecurityPolicy        SecurityPolicy `yaml:"security" json:"security"`
	LoadTimeout           time.Duration  `yaml:"load_timeout" json:"load_timeout"`
	EnableAutoDiscovery   bool           `yaml:"auto_discovery" json:"auto_discovery"`
	EnableVersionCheck    bool           `yaml:"version_check" json:"version_check"`
	MaxPlugins            int            `yaml:"max_plugins" json:"max_plugins"`
	IsolationMode         string         `yaml:"isolation_mode" json:"isolation_mode"`
	AllowedPluginPrefixes []string       `yaml:"allowed_prefixes" json:"allowed_prefixes"`
}

// SecurityPolicy defines security policies for plugin loading.
type SecurityPolicy struct {
	AllowUnsigned   bool     `yaml:"allow_unsigned" json:"allow_unsigned"`
	TrustedSources  []string `yaml:"trusted_sources" json:"trusted_sources"`
	BlockedPatterns []string `yaml:"blocked_patterns" json:"blocked_patterns"`
	RequireChecksum bool     `yaml:"require_checksum" json:"require_checksum"`
}

// PluginMetadata represents metadata for a plugin.
type PluginMetadata struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Description     string            `json:"description"`
	Author          string            `json:"author"`
	License         string            `json:"license"`
	Homepage        string            `json:"homepage"`
	Dependencies    []string          `json:"dependencies"`
	Capabilities    []string          `json:"capabilities"`
	MinFrameworkVer string            `json:"min_framework_version"`
	MaxFrameworkVer string            `json:"max_framework_version"`
	LoadedAt        time.Time         `json:"loaded_at"`
	Path            string            `json:"path"`
	Checksum        string            `json:"checksum"`
	Config          map[string]string `json:"config"`
}

// PluginStatus represents the status of a plugin.
type PluginStatus string

const (
	PluginStatusUnloaded    PluginStatus = "unloaded"
	PluginStatusLoading     PluginStatus = "loading"
	PluginStatusLoaded      PluginStatus = "loaded"
	PluginStatusInitialized PluginStatus = "initialized"
	PluginStatusActive      PluginStatus = "active"
	PluginStatusError       PluginStatus = "error"
	PluginStatusDisabled    PluginStatus = "disabled"
)

// PluginInfo provides detailed information about a plugin.
type PluginInfo struct {
	Metadata PluginMetadata    `json:"metadata"`
	Status   PluginStatus      `json:"status"`
	Error    string            `json:"error,omitempty"`
	Plugin   interfaces.Plugin `json:"-"`
}

// PluginDiscovery represents a discovered plugin.
type PluginDiscovery struct {
	Path      string            `json:"path"`
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	Metadata  *PluginMetadata   `json:"metadata"`
	Available bool              `json:"available"`
	Error     string            `json:"error,omitempty"`
	Config    map[string]string `json:"config"`
}

// DiscoveryConfig represents configuration for plugin discovery.
type DiscoveryConfig struct {
	AutoDiscovery    bool          `yaml:"auto_discovery" json:"auto_discovery"`
	ScanInterval     time.Duration `yaml:"scan_interval" json:"scan_interval"`
	WatchDirectories bool          `yaml:"watch_directories" json:"watch_directories"`
	CacheResults     bool          `yaml:"cache_results" json:"cache_results"`
	CacheTTL         time.Duration `yaml:"cache_ttl" json:"cache_ttl"`
	MaxDepth         int           `yaml:"max_depth" json:"max_depth"`
	FollowSymlinks   bool          `yaml:"follow_symlinks" json:"follow_symlinks"`
	ExcludePatterns  []string      `yaml:"exclude_patterns" json:"exclude_patterns"`
	IncludePatterns  []string      `yaml:"include_patterns" json:"include_patterns"`
	ParallelScanning bool          `yaml:"parallel_scanning" json:"parallel_scanning"`
	MaxWorkers       int           `yaml:"max_workers" json:"max_workers"`
}

// RegistryConfig represents configuration for the plugin registry.
type RegistryConfig struct {
	CachePath         string        `yaml:"cache_path" json:"cache_path"`
	AutoSave          bool          `yaml:"auto_save" json:"auto_save"`
	SaveInterval      time.Duration `yaml:"save_interval" json:"save_interval"`
	EnableVersioning  bool          `yaml:"enable_versioning" json:"enable_versioning"`
	MaxVersionHistory int           `yaml:"max_version_history" json:"max_version_history"`
}

// VersionInfo represents version information for a plugin.
type VersionInfo struct {
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path"`
	Checksum  string    `json:"checksum"`
}

// DependencyGraph represents plugin dependencies.
type DependencyGraph struct {
	Dependencies map[string][]string `json:"dependencies"`
	Dependents   map[string][]string `json:"dependents"`
}

// DirectoryWatcher watches a directory for plugin changes.
type DirectoryWatcher struct {
	path        string
	lastScan    time.Time
	pluginFiles map[string]time.Time
	config      *DiscoveryConfig
	onChange    func([]PluginChangeEvent)
}

// PluginChangeEvent represents a plugin file change event.
type PluginChangeEvent struct {
	Type     ChangeEventType `json:"type"`
	Path     string          `json:"path"`
	OldPath  string          `json:"old_path,omitempty"`
	Time     time.Time       `json:"time"`
	Metadata *PluginMetadata `json:"metadata,omitempty"`
}

// ChangeEventType represents the type of change event.
type ChangeEventType string

const (
	EventPluginAdded    ChangeEventType = "added"
	EventPluginRemoved  ChangeEventType = "removed"
	EventPluginModified ChangeEventType = "modified"
	EventPluginMoved    ChangeEventType = "moved"
)

// DiscoveryResult represents the result of plugin discovery.
type DiscoveryResult struct {
	Plugins            []PluginDiscovery `json:"plugins"`
	ScanTime           time.Duration     `json:"scan_time"`
	DirectoriesScanned int               `json:"directories_scanned"`
	FilesScanned       int               `json:"files_scanned"`
	Errors             []DiscoveryError  `json:"errors,omitempty"`
	CacheHit           bool              `json:"cache_hit"`
	Timestamp          time.Time         `json:"timestamp"`
}

// DiscoveryError represents an error that occurred during discovery.
type DiscoveryError struct {
	Path    string `json:"path"`
	Error   string `json:"error"`
	Skipped bool   `json:"skipped"`
}

// VersionManager manages plugin versions and compatibility.
type VersionManager struct {
	versions       map[string][]VersionInfo
	compatibility  map[string]CompatibilityInfo
	currentVersion string
	mu             sync.RWMutex
}

// CompatibilityInfo represents compatibility information for a plugin.
type CompatibilityInfo struct {
	MinFrameworkVersion string   `json:"min_framework_version"`
	MaxFrameworkVersion string   `json:"max_framework_version"`
	SupportedVersions   []string `json:"supported_versions"`
	DeprecatedVersions  []string `json:"deprecated_versions"`
	BreakingChanges     []string `json:"breaking_changes"`
}

// RegistryStatistics represents statistics about the plugin registry.
type RegistryStatistics struct {
	TotalPlugins int                  `json:"total_plugins"`
	StatusCounts map[PluginStatus]int `json:"status_counts"`
	LoadedAt     time.Time            `json:"loaded_at"`
}

// PluginSystemConfig represents the complete plugin system configuration.
type PluginSystemConfig struct {
	Manager   *ManagerConfig   `yaml:"manager" json:"manager"`
	Discovery *DiscoveryConfig `yaml:"discovery" json:"discovery"`
	Registry  *RegistryConfig  `yaml:"registry" json:"registry"`
	Security  *SecurityPolicy  `yaml:"security" json:"security"`
	Global    *GlobalConfig    `yaml:"global" json:"global"`
}

// GlobalConfig represents global plugin system configuration.
type GlobalConfig struct {
	PluginVersion      string            `yaml:"plugin_version" json:"plugin_version"`
	FrameworkVersion   string            `yaml:"framework_version" json:"framework_version"`
	ConfigVersion      string            `yaml:"config_version" json:"config_version"`
	DefaultTimeout     time.Duration     `yaml:"default_timeout" json:"default_timeout"`
	LogLevel           string            `yaml:"log_level" json:"log_level"`
	MetricsEnabled     bool              `yaml:"metrics_enabled" json:"metrics_enabled"`
	HealthCheckEnabled bool              `yaml:"health_check_enabled" json:"health_check_enabled"`
	Environment        string            `yaml:"environment" json:"environment"`
	Tags               []string          `yaml:"tags" json:"tags"`
	Metadata           map[string]string `yaml:"metadata" json:"metadata"`
}

// ConfigWatcher is called when configuration changes.
type ConfigWatcher func(oldConfig, newConfig *PluginSystemConfig) error

// ValidationResult represents the result of plugin validation.
type ValidationResult struct {
	Valid       bool              `json:"valid"`
	Errors      []ValidationError `json:"errors,omitempty"`
	Warnings    []ValidationError `json:"warnings,omitempty"`
	Checksum    string            `json:"checksum"`
	Size        int64             `json:"size"`
	Permissions os.FileMode       `json:"permissions"`
	Symbols     []string          `json:"symbols,omitempty"`
}

// ValidationError represents a validation error or warning.
type ValidationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Level   string `json:"level"`
}

// PluginValidator provides plugin validation capabilities.
type PluginValidator struct {
	trustedSources    []string
	blockedPatterns   []string
	requireSignature  bool
	requireChecksum   bool
	allowedSymbols    []string
	blockedSymbols    []string
	maxFileSize       int64
	allowedExtensions []string
}
