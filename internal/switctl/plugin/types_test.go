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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type TypesTestSuite struct {
	suite.Suite
}

func (s *TypesTestSuite) TestPluginStatusConstants() {
	// Test that all plugin status constants are defined correctly
	s.Equal(PluginStatus("unloaded"), PluginStatusUnloaded)
	s.Equal(PluginStatus("loading"), PluginStatusLoading)
	s.Equal(PluginStatus("loaded"), PluginStatusLoaded)
	s.Equal(PluginStatus("initialized"), PluginStatusInitialized)
	s.Equal(PluginStatus("active"), PluginStatusActive)
	s.Equal(PluginStatus("error"), PluginStatusError)
	s.Equal(PluginStatus("disabled"), PluginStatusDisabled)
}

func (s *TypesTestSuite) TestChangeEventTypeConstants() {
	// Test that all change event type constants are defined correctly
	s.Equal(ChangeEventType("added"), EventPluginAdded)
	s.Equal(ChangeEventType("removed"), EventPluginRemoved)
	s.Equal(ChangeEventType("modified"), EventPluginModified)
	s.Equal(ChangeEventType("moved"), EventPluginMoved)
}

func (s *TypesTestSuite) TestManagerConfigDefaults() {
	config := &ManagerConfig{
		PluginDirs:            []string{"/plugins"},
		LoadTimeout:           30 * time.Second,
		EnableAutoDiscovery:   true,
		EnableVersionCheck:    true,
		MaxPlugins:            100,
		IsolationMode:         "process",
		AllowedPluginPrefixes: []string{"swit-"},
	}

	s.Equal([]string{"/plugins"}, config.PluginDirs)
	s.Equal(30*time.Second, config.LoadTimeout)
	s.True(config.EnableAutoDiscovery)
	s.True(config.EnableVersionCheck)
	s.Equal(100, config.MaxPlugins)
	s.Equal("process", config.IsolationMode)
	s.Equal([]string{"swit-"}, config.AllowedPluginPrefixes)
}

func (s *TypesTestSuite) TestSecurityPolicyDefaults() {
	policy := &SecurityPolicy{
		AllowUnsigned:   false,
		TrustedSources:  []string{"/trusted"},
		BlockedPatterns: []string{"*malware*"},
		RequireChecksum: true,
	}

	s.False(policy.AllowUnsigned)
	s.Equal([]string{"/trusted"}, policy.TrustedSources)
	s.Equal([]string{"*malware*"}, policy.BlockedPatterns)
	s.True(policy.RequireChecksum)
}

func (s *TypesTestSuite) TestPluginMetadata() {
	now := time.Now()
	metadata := &PluginMetadata{
		Name:            "test-plugin",
		Version:         "1.0.0",
		Description:     "A test plugin",
		Author:          "Test Author",
		License:         "MIT",
		Homepage:        "https://example.com",
		Dependencies:    []string{"dep1", "dep2"},
		Capabilities:    []string{"auth", "logging"},
		MinFrameworkVer: "1.0.0",
		MaxFrameworkVer: "2.0.0",
		LoadedAt:        now,
		Path:            "/plugins/test-plugin.so",
		Checksum:        "abc123",
		Config:          map[string]string{"key": "value"},
	}

	s.Equal("test-plugin", metadata.Name)
	s.Equal("1.0.0", metadata.Version)
	s.Equal("A test plugin", metadata.Description)
	s.Equal("Test Author", metadata.Author)
	s.Equal("MIT", metadata.License)
	s.Equal("https://example.com", metadata.Homepage)
	s.Equal([]string{"dep1", "dep2"}, metadata.Dependencies)
	s.Equal([]string{"auth", "logging"}, metadata.Capabilities)
	s.Equal("1.0.0", metadata.MinFrameworkVer)
	s.Equal("2.0.0", metadata.MaxFrameworkVer)
	s.Equal(now, metadata.LoadedAt)
	s.Equal("/plugins/test-plugin.so", metadata.Path)
	s.Equal("abc123", metadata.Checksum)
	s.Equal("value", metadata.Config["key"])
}

func (s *TypesTestSuite) TestPluginInfo() {
	metadata := PluginMetadata{
		Name:    "test-plugin",
		Version: "1.0.0",
	}

	info := &PluginInfo{
		Metadata: metadata,
		Status:   PluginStatusLoaded,
		Error:    "",
		Plugin:   nil,
	}

	s.Equal("test-plugin", info.Metadata.Name)
	s.Equal("1.0.0", info.Metadata.Version)
	s.Equal(PluginStatusLoaded, info.Status)
	s.Empty(info.Error)
	s.Nil(info.Plugin)
}

func (s *TypesTestSuite) TestPluginDiscovery() {
	metadata := &PluginMetadata{
		Name:    "discovered-plugin",
		Version: "2.0.0",
	}

	discovery := &PluginDiscovery{
		Path:      "/plugins/discovered-plugin.so",
		Name:      "discovered-plugin",
		Version:   "2.0.0",
		Metadata:  metadata,
		Available: true,
		Error:     "",
		Config:    map[string]string{"setting": "value"},
	}

	s.Equal("/plugins/discovered-plugin.so", discovery.Path)
	s.Equal("discovered-plugin", discovery.Name)
	s.Equal("2.0.0", discovery.Version)
	s.Equal(metadata, discovery.Metadata)
	s.True(discovery.Available)
	s.Empty(discovery.Error)
	s.Equal("value", discovery.Config["setting"])
}

func (s *TypesTestSuite) TestDiscoveryConfig() {
	config := &DiscoveryConfig{
		AutoDiscovery:    true,
		ScanInterval:     5 * time.Minute,
		WatchDirectories: true,
		CacheResults:     true,
		CacheTTL:         1 * time.Hour,
		MaxDepth:         3,
		FollowSymlinks:   false,
		ExcludePatterns:  []string{"*.tmp"},
		IncludePatterns:  []string{"*.so", "*.dylib"},
		ParallelScanning: true,
		MaxWorkers:       4,
	}

	s.True(config.AutoDiscovery)
	s.Equal(5*time.Minute, config.ScanInterval)
	s.True(config.WatchDirectories)
	s.True(config.CacheResults)
	s.Equal(1*time.Hour, config.CacheTTL)
	s.Equal(3, config.MaxDepth)
	s.False(config.FollowSymlinks)
	s.Equal([]string{"*.tmp"}, config.ExcludePatterns)
	s.Equal([]string{"*.so", "*.dylib"}, config.IncludePatterns)
	s.True(config.ParallelScanning)
	s.Equal(4, config.MaxWorkers)
}

func (s *TypesTestSuite) TestRegistryConfig() {
	config := &RegistryConfig{
		CachePath:         "/cache/plugins.db",
		AutoSave:          true,
		SaveInterval:      10 * time.Minute,
		EnableVersioning:  true,
		MaxVersionHistory: 5,
	}

	s.Equal("/cache/plugins.db", config.CachePath)
	s.True(config.AutoSave)
	s.Equal(10*time.Minute, config.SaveInterval)
	s.True(config.EnableVersioning)
	s.Equal(5, config.MaxVersionHistory)
}

func (s *TypesTestSuite) TestVersionInfo() {
	now := time.Now()
	version := &VersionInfo{
		Version:   "1.2.3",
		Timestamp: now,
		Path:      "/plugins/v1.2.3/plugin.so",
		Checksum:  "def456",
	}

	s.Equal("1.2.3", version.Version)
	s.Equal(now, version.Timestamp)
	s.Equal("/plugins/v1.2.3/plugin.so", version.Path)
	s.Equal("def456", version.Checksum)
}

func (s *TypesTestSuite) TestDependencyGraph() {
	graph := &DependencyGraph{
		Dependencies: map[string][]string{
			"plugin1": {"dep1", "dep2"},
			"plugin2": {"dep1"},
		},
		Dependents: map[string][]string{
			"dep1": {"plugin1", "plugin2"},
			"dep2": {"plugin1"},
		},
	}

	s.Equal([]string{"dep1", "dep2"}, graph.Dependencies["plugin1"])
	s.Equal([]string{"dep1"}, graph.Dependencies["plugin2"])
	s.Equal([]string{"plugin1", "plugin2"}, graph.Dependents["dep1"])
	s.Equal([]string{"plugin1"}, graph.Dependents["dep2"])
}

func (s *TypesTestSuite) TestPluginChangeEvent() {
	now := time.Now()
	metadata := &PluginMetadata{Name: "changed-plugin"}

	event := &PluginChangeEvent{
		Type:     EventPluginModified,
		Path:     "/plugins/changed-plugin.so",
		OldPath:  "/plugins/old-plugin.so",
		Time:     now,
		Metadata: metadata,
	}

	s.Equal(EventPluginModified, event.Type)
	s.Equal("/plugins/changed-plugin.so", event.Path)
	s.Equal("/plugins/old-plugin.so", event.OldPath)
	s.Equal(now, event.Time)
	s.Equal(metadata, event.Metadata)
}

func (s *TypesTestSuite) TestDiscoveryResult() {
	now := time.Now()
	plugins := []PluginDiscovery{
		{Name: "plugin1", Available: true},
		{Name: "plugin2", Available: false},
	}
	errors := []DiscoveryError{
		{Path: "/error/path", Error: "test error", Skipped: true},
	}

	result := &DiscoveryResult{
		Plugins:            plugins,
		ScanTime:           2 * time.Second,
		DirectoriesScanned: 3,
		FilesScanned:       10,
		Errors:             errors,
		CacheHit:           false,
		Timestamp:          now,
	}

	s.Len(result.Plugins, 2)
	s.Equal("plugin1", result.Plugins[0].Name)
	s.True(result.Plugins[0].Available)
	s.Equal("plugin2", result.Plugins[1].Name)
	s.False(result.Plugins[1].Available)
	s.Equal(2*time.Second, result.ScanTime)
	s.Equal(3, result.DirectoriesScanned)
	s.Equal(10, result.FilesScanned)
	s.Len(result.Errors, 1)
	s.Equal("/error/path", result.Errors[0].Path)
	s.Equal("test error", result.Errors[0].Error)
	s.True(result.Errors[0].Skipped)
	s.False(result.CacheHit)
	s.Equal(now, result.Timestamp)
}

func (s *TypesTestSuite) TestCompatibilityInfo() {
	info := &CompatibilityInfo{
		MinFrameworkVersion: "1.0.0",
		MaxFrameworkVersion: "2.0.0",
		SupportedVersions:   []string{"1.1.0", "1.2.0", "1.3.0"},
		DeprecatedVersions:  []string{"1.0.0"},
		BreakingChanges:     []string{"API change in v2.0.0"},
	}

	s.Equal("1.0.0", info.MinFrameworkVersion)
	s.Equal("2.0.0", info.MaxFrameworkVersion)
	s.Equal([]string{"1.1.0", "1.2.0", "1.3.0"}, info.SupportedVersions)
	s.Equal([]string{"1.0.0"}, info.DeprecatedVersions)
	s.Equal([]string{"API change in v2.0.0"}, info.BreakingChanges)
}

func (s *TypesTestSuite) TestRegistryStatistics() {
	now := time.Now()
	statusCounts := map[PluginStatus]int{
		PluginStatusActive:   3,
		PluginStatusDisabled: 1,
		PluginStatusError:    1,
	}

	stats := &RegistryStatistics{
		TotalPlugins: 5,
		StatusCounts: statusCounts,
		LoadedAt:     now,
	}

	s.Equal(5, stats.TotalPlugins)
	s.Equal(3, stats.StatusCounts[PluginStatusActive])
	s.Equal(1, stats.StatusCounts[PluginStatusDisabled])
	s.Equal(1, stats.StatusCounts[PluginStatusError])
	s.Equal(now, stats.LoadedAt)
}

func (s *TypesTestSuite) TestPluginSystemConfig() {
	managerConfig := &ManagerConfig{MaxPlugins: 50}
	discoveryConfig := &DiscoveryConfig{AutoDiscovery: true}
	registryConfig := &RegistryConfig{AutoSave: true}
	securityPolicy := &SecurityPolicy{AllowUnsigned: false}
	globalConfig := &GlobalConfig{PluginVersion: "1.0.0"}

	systemConfig := &PluginSystemConfig{
		Manager:   managerConfig,
		Discovery: discoveryConfig,
		Registry:  registryConfig,
		Security:  securityPolicy,
		Global:    globalConfig,
	}

	s.Equal(managerConfig, systemConfig.Manager)
	s.Equal(discoveryConfig, systemConfig.Discovery)
	s.Equal(registryConfig, systemConfig.Registry)
	s.Equal(securityPolicy, systemConfig.Security)
	s.Equal(globalConfig, systemConfig.Global)
}

func (s *TypesTestSuite) TestGlobalConfig() {
	config := &GlobalConfig{
		PluginVersion:      "2.1.0",
		FrameworkVersion:   "3.0.0",
		ConfigVersion:      "1.0",
		DefaultTimeout:     30 * time.Second,
		LogLevel:           "info",
		MetricsEnabled:     true,
		HealthCheckEnabled: true,
		Environment:        "production",
		Tags:               []string{"stable", "production"},
		Metadata:           map[string]string{"region": "us-west-2"},
	}

	s.Equal("2.1.0", config.PluginVersion)
	s.Equal("3.0.0", config.FrameworkVersion)
	s.Equal("1.0", config.ConfigVersion)
	s.Equal(30*time.Second, config.DefaultTimeout)
	s.Equal("info", config.LogLevel)
	s.True(config.MetricsEnabled)
	s.True(config.HealthCheckEnabled)
	s.Equal("production", config.Environment)
	s.Equal([]string{"stable", "production"}, config.Tags)
	s.Equal("us-west-2", config.Metadata["region"])
}

func (s *TypesTestSuite) TestValidationResult() {
	errors := []ValidationError{
		{Code: "E001", Message: "Invalid signature", Level: "error"},
	}
	warnings := []ValidationError{
		{Code: "W001", Message: "Deprecated API", Level: "warning"},
	}

	result := &ValidationResult{
		Valid:       false,
		Errors:      errors,
		Warnings:    warnings,
		Checksum:    "abc123def456",
		Size:        1024,
		Permissions: 0755,
		Symbols:     []string{"Initialize", "Execute", "Cleanup"},
	}

	s.False(result.Valid)
	s.Len(result.Errors, 1)
	s.Equal("E001", result.Errors[0].Code)
	s.Equal("Invalid signature", result.Errors[0].Message)
	s.Equal("error", result.Errors[0].Level)
	s.Len(result.Warnings, 1)
	s.Equal("W001", result.Warnings[0].Code)
	s.Equal("Deprecated API", result.Warnings[0].Message)
	s.Equal("warning", result.Warnings[0].Level)
	s.Equal("abc123def456", result.Checksum)
	s.Equal(int64(1024), result.Size)
	s.Equal(os.FileMode(0755), result.Permissions)
	s.Equal([]string{"Initialize", "Execute", "Cleanup"}, result.Symbols)
}

func TestTypesTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}
