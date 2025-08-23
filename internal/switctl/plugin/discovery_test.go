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
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PluginDiscoveryTestSuite contains all tests for plugin discovery functionality.
type PluginDiscoveryTestSuite struct {
	suite.Suite
	tempDir         string
	pluginDir1      string
	pluginDir2      string
	discoveryConfig *DiscoveryConfig
}

// SetupSuite initializes the test suite.
func (suite *PluginDiscoveryTestSuite) SetupSuite() {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "switctl-discovery-test-*")
	require.NoError(suite.T(), err)
	suite.tempDir = tempDir

	suite.pluginDir1 = filepath.Join(tempDir, "plugins1")
	suite.pluginDir2 = filepath.Join(tempDir, "plugins2")

	// Create plugin directories
	err = os.MkdirAll(suite.pluginDir1, 0755)
	require.NoError(suite.T(), err)
	err = os.MkdirAll(suite.pluginDir2, 0755)
	require.NoError(suite.T(), err)
}

// TearDownSuite cleans up after all tests.
func (suite *PluginDiscoveryTestSuite) TearDownSuite() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// SetupTest initializes each test case.
func (suite *PluginDiscoveryTestSuite) SetupTest() {
	suite.discoveryConfig = &DiscoveryConfig{
		AutoDiscovery:    true,
		ScanInterval:     1 * time.Minute,
		WatchDirectories: false,
		CacheResults:     true,
		CacheTTL:         5 * time.Minute,
		MaxDepth:         3,
		FollowSymlinks:   false,
		ExcludePatterns:  []string{"*.tmp", "*.bak"},
		IncludePatterns:  []string{"*.so", "*.dll", "*.dylib"},
		ParallelScanning: true,
		MaxWorkers:       4,
	}
}

// TearDownTest cleans up after each test.
func (suite *PluginDiscoveryTestSuite) TearDownTest() {
	// Clean plugin directories
	suite.cleanPluginDirs()
}

// cleanPluginDirs removes all files from plugin directories.
func (suite *PluginDiscoveryTestSuite) cleanPluginDirs() {
	dirs := []string{suite.pluginDir1, suite.pluginDir2}
	for _, dir := range dirs {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, entry := range entries {
				os.RemoveAll(filepath.Join(dir, entry.Name()))
			}
		}
	}
}

// createTestPluginFile creates a test plugin file.
func (suite *PluginDiscoveryTestSuite) createTestPluginFile(dir, name string) string {
	pluginPath := filepath.Join(dir, name)
	content := fmt.Sprintf("// Plugin: %s\n// This is a test plugin file", name)
	err := os.WriteFile(pluginPath, []byte(content), 0644)
	require.NoError(suite.T(), err)
	return pluginPath
}

// TestPluginDiscoveryBasic tests basic plugin discovery functionality.
func (suite *PluginDiscoveryTestSuite) TestPluginDiscoveryBasic() {
	// Create test plugin files
	plugin1 := suite.createTestPluginFile(suite.pluginDir1, "plugin1.so")
	plugin2 := suite.createTestPluginFile(suite.pluginDir1, "plugin2.dll")
	plugin3 := suite.createTestPluginFile(suite.pluginDir2, "plugin3.dylib")

	// Create non-plugin files (should be excluded)
	suite.createTestPluginFile(suite.pluginDir1, "readme.txt")
	suite.createTestPluginFile(suite.pluginDir1, "temp.tmp")

	// Mock discovery implementation
	discovery := &MockPluginDiscovery{
		dirs:   []string{suite.pluginDir1, suite.pluginDir2},
		config: suite.discoveryConfig,
	}

	result, err := discovery.DiscoverPlugins()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)

	// Should find 3 plugin files
	assert.Len(suite.T(), result.Plugins, 3)
	assert.Equal(suite.T(), 2, result.DirectoriesScanned)
	assert.True(suite.T(), result.ScanTime > 0)

	// Verify plugin paths
	pluginPaths := make(map[string]bool)
	for _, plugin := range result.Plugins {
		pluginPaths[plugin.Path] = true
	}

	assert.True(suite.T(), pluginPaths[plugin1])
	assert.True(suite.T(), pluginPaths[plugin2])
	assert.True(suite.T(), pluginPaths[plugin3])
}

// TestPluginDiscoveryWithFilters tests discovery with include/exclude patterns.
func (suite *PluginDiscoveryTestSuite) TestPluginDiscoveryWithFilters() {
	// Create various test files
	suite.createTestPluginFile(suite.pluginDir1, "plugin1.so")
	suite.createTestPluginFile(suite.pluginDir1, "plugin2.dll")
	suite.createTestPluginFile(suite.pluginDir1, "plugin3.exe") // Should be excluded
	suite.createTestPluginFile(suite.pluginDir1, "backup.bak")  // Should be excluded
	suite.createTestPluginFile(suite.pluginDir1, "temp.tmp")    // Should be excluded

	config := &DiscoveryConfig{
		AutoDiscovery:    true,
		ExcludePatterns:  []string{"*.exe", "*.bak", "*.tmp"},
		IncludePatterns:  []string{"*.so", "*.dll"},
		ParallelScanning: false,
		MaxWorkers:       1,
	}

	discovery := &MockPluginDiscovery{
		dirs:   []string{suite.pluginDir1},
		config: config,
	}

	result, err := discovery.DiscoverPlugins()
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result.Plugins, 2) // Only .so and .dll files
}

// TestPluginDiscoveryParallel tests parallel scanning.
func (suite *PluginDiscoveryTestSuite) TestPluginDiscoveryParallel() {
	// Create multiple plugin files across directories
	for i := 0; i < 10; i++ {
		suite.createTestPluginFile(suite.pluginDir1, fmt.Sprintf("plugin%d.so", i))
		suite.createTestPluginFile(suite.pluginDir2, fmt.Sprintf("plugin%d.dll", i))
	}

	config := &DiscoveryConfig{
		AutoDiscovery:    true,
		ParallelScanning: true,
		MaxWorkers:       4,
		IncludePatterns:  []string{"*.so", "*.dll"},
	}

	discovery := &MockPluginDiscovery{
		dirs:   []string{suite.pluginDir1, suite.pluginDir2},
		config: config,
	}

	result, err := discovery.DiscoverPlugins()
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result.Plugins, 20) // 10 + 10 plugins
}

// TestPluginDiscoveryErrors tests discovery error handling.
func (suite *PluginDiscoveryTestSuite) TestPluginDiscoveryErrors() {
	// Test with non-existent directory
	discovery := &MockPluginDiscovery{
		dirs:   []string{"/nonexistent/directory"},
		config: suite.discoveryConfig,
	}

	result, err := discovery.DiscoverPlugins()
	// Should handle gracefully without fatal error
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Empty(suite.T(), result.Plugins)
	assert.NotEmpty(suite.T(), result.Errors)
}

// TestPluginDiscoveryCaching tests result caching functionality.
func (suite *PluginDiscoveryTestSuite) TestPluginDiscoveryCaching() {
	suite.createTestPluginFile(suite.pluginDir1, "plugin1.so")

	config := &DiscoveryConfig{
		AutoDiscovery:   true,
		CacheResults:    true,
		CacheTTL:        1 * time.Second,
		IncludePatterns: []string{"*.so"},
	}

	discovery := &MockPluginDiscovery{
		dirs:   []string{suite.pluginDir1},
		config: config,
		cache:  make(map[string]*DiscoveryResult),
	}

	// First scan - should not be cached
	result1, err := discovery.DiscoverPlugins()
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), result1.CacheHit)

	// Second scan - should be cached
	result2, err := discovery.DiscoverPlugins()
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), result2.CacheHit)

	// Wait for cache to expire
	time.Sleep(1100 * time.Millisecond)

	// Third scan - cache expired, should rescan
	result3, err := discovery.DiscoverPlugins()
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), result3.CacheHit)
}

// TestPluginDiscoveryConcurrent tests concurrent discovery operations.
func (suite *PluginDiscoveryTestSuite) TestPluginDiscoveryConcurrent() {
	// Create test files
	for i := 0; i < 5; i++ {
		suite.createTestPluginFile(suite.pluginDir1, fmt.Sprintf("plugin%d.so", i))
	}

	discovery := &MockPluginDiscovery{
		dirs:   []string{suite.pluginDir1},
		config: suite.discoveryConfig,
		cache:  make(map[string]*DiscoveryResult),
	}

	const numGoroutines = 10
	results := make(chan *DiscoveryResult, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Launch concurrent discovery operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			result, err := discovery.DiscoverPlugins()
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			assert.Len(suite.T(), result.Plugins, 5)
		case err := <-errors:
			assert.NoError(suite.T(), err, "No errors expected in concurrent discovery")
		case <-time.After(5 * time.Second):
			suite.T().Fatal("Timeout waiting for discovery results")
		}
	}
}

// TestPluginChangeEvent tests plugin change event functionality.
func (suite *PluginDiscoveryTestSuite) TestPluginChangeEvent() {
	events := []PluginChangeEvent{
		{
			Type: EventPluginAdded,
			Path: "/plugins/new-plugin.so",
			Time: time.Now(),
		},
		{
			Type: EventPluginRemoved,
			Path: "/plugins/old-plugin.so",
			Time: time.Now(),
		},
		{
			Type: EventPluginModified,
			Path: "/plugins/changed-plugin.so",
			Time: time.Now(),
		},
		{
			Type:    EventPluginMoved,
			Path:    "/plugins/moved-plugin.so",
			OldPath: "/plugins/old-location.so",
			Time:    time.Now(),
		},
	}

	for i, event := range events {
		suite.T().Run(fmt.Sprintf("event_%d_%s", i, event.Type), func(t *testing.T) {
			assert.NotEmpty(t, event.Type)
			assert.NotEmpty(t, event.Path)
			assert.False(t, event.Time.IsZero())

			if event.Type == EventPluginMoved {
				assert.NotEmpty(t, event.OldPath)
			}
		})
	}
}

// TestVersionManager tests version management functionality.
func (suite *PluginDiscoveryTestSuite) TestVersionManager() {
	vm := &VersionManager{
		versions:       make(map[string][]VersionInfo),
		compatibility:  make(map[string]CompatibilityInfo),
		currentVersion: "1.0.0",
		mu:             sync.RWMutex{},
	}

	// Add version info
	pluginName := "test-plugin"
	version1 := VersionInfo{
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Path:      "/plugins/test-plugin-v1.0.0.so",
		Checksum:  "abc123",
	}

	version2 := VersionInfo{
		Version:   "1.1.0",
		Timestamp: time.Now().Add(1 * time.Hour),
		Path:      "/plugins/test-plugin-v1.1.0.so",
		Checksum:  "def456",
	}

	vm.mu.Lock()
	vm.versions[pluginName] = []VersionInfo{version1, version2}
	vm.compatibility[pluginName] = CompatibilityInfo{
		MinFrameworkVersion: "1.0.0",
		MaxFrameworkVersion: "2.0.0",
		SupportedVersions:   []string{"1.0.0", "1.1.0"},
	}
	vm.mu.Unlock()

	// Test version retrieval
	vm.mu.RLock()
	versions := vm.versions[pluginName]
	compatibility := vm.compatibility[pluginName]
	vm.mu.RUnlock()

	assert.Len(suite.T(), versions, 2)
	assert.Equal(suite.T(), "1.0.0", versions[0].Version)
	assert.Equal(suite.T(), "1.1.0", versions[1].Version)
	assert.Equal(suite.T(), "1.0.0", compatibility.MinFrameworkVersion)
	assert.Equal(suite.T(), "2.0.0", compatibility.MaxFrameworkVersion)
}

// TestDirectoryWatcher tests directory watching functionality.
func (suite *PluginDiscoveryTestSuite) TestDirectoryWatcher() {
	watcher := &DirectoryWatcher{
		path:        suite.pluginDir1,
		lastScan:    time.Now(),
		pluginFiles: make(map[string]time.Time),
		config:      suite.discoveryConfig,
	}

	// Initial scan
	suite.createTestPluginFile(suite.pluginDir1, "initial.so")

	// Simulate file change detection
	entries, err := os.ReadDir(watcher.path)
	assert.NoError(suite.T(), err)

	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err == nil {
				watcher.pluginFiles[entry.Name()] = info.ModTime()
			}
		}
	}

	assert.Len(suite.T(), watcher.pluginFiles, 1)
	assert.Contains(suite.T(), watcher.pluginFiles, "initial.so")

	// Add new file
	suite.createTestPluginFile(suite.pluginDir1, "new.so")

	// Simulate change detection
	newEntries, err := os.ReadDir(watcher.path)
	assert.NoError(suite.T(), err)

	var changes []PluginChangeEvent
	currentFiles := make(map[string]time.Time)

	for _, entry := range newEntries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err == nil {
				currentFiles[entry.Name()] = info.ModTime()

				// Check if this is a new file
				if _, exists := watcher.pluginFiles[entry.Name()]; !exists {
					changes = append(changes, PluginChangeEvent{
						Type: EventPluginAdded,
						Path: filepath.Join(watcher.path, entry.Name()),
						Time: time.Now(),
					})
				}
			}
		}
	}

	assert.Len(suite.T(), changes, 1)
	assert.Equal(suite.T(), EventPluginAdded, changes[0].Type)
	assert.Contains(suite.T(), changes[0].Path, "new.so")
}

// TestDiscoveryConfigValidation tests discovery configuration validation.
func (suite *PluginDiscoveryTestSuite) TestDiscoveryConfigValidation() {
	// Test valid configuration
	validConfig := &DiscoveryConfig{
		AutoDiscovery:    true,
		ScanInterval:     1 * time.Minute,
		WatchDirectories: true,
		CacheResults:     true,
		CacheTTL:         5 * time.Minute,
		MaxDepth:         5,
		FollowSymlinks:   false,
		ExcludePatterns:  []string{"*.tmp"},
		IncludePatterns:  []string{"*.so"},
		ParallelScanning: true,
		MaxWorkers:       8,
	}

	assert.True(suite.T(), validConfig.AutoDiscovery)
	assert.Equal(suite.T(), 1*time.Minute, validConfig.ScanInterval)
	assert.True(suite.T(), validConfig.WatchDirectories)
	assert.True(suite.T(), validConfig.CacheResults)
	assert.Equal(suite.T(), 5*time.Minute, validConfig.CacheTTL)
	assert.Equal(suite.T(), 5, validConfig.MaxDepth)
	assert.False(suite.T(), validConfig.FollowSymlinks)
	assert.Equal(suite.T(), []string{"*.tmp"}, validConfig.ExcludePatterns)
	assert.Equal(suite.T(), []string{"*.so"}, validConfig.IncludePatterns)
	assert.True(suite.T(), validConfig.ParallelScanning)
	assert.Equal(suite.T(), 8, validConfig.MaxWorkers)
}

// MockPluginDiscovery is a mock implementation for testing.
type MockPluginDiscovery struct {
	dirs   []string
	config *DiscoveryConfig
	cache  map[string]*DiscoveryResult
	mu     sync.RWMutex
}

// DiscoverPlugins implements plugin discovery logic for testing.
func (m *MockPluginDiscovery) DiscoverPlugins() (*DiscoveryResult, error) {
	startTime := time.Now()

	// Check cache if enabled
	if m.config.CacheResults && m.cache != nil {
		cacheKey := fmt.Sprintf("%v", m.dirs)

		m.mu.RLock()
		if cached, exists := m.cache[cacheKey]; exists {
			if time.Since(cached.Timestamp) < m.config.CacheTTL {
				cached.CacheHit = true
				m.mu.RUnlock()
				return cached, nil
			}
		}
		m.mu.RUnlock()
	}

	result := &DiscoveryResult{
		Plugins:            []PluginDiscovery{},
		ScanTime:           0,
		DirectoriesScanned: len(m.dirs),
		FilesScanned:       0,
		Errors:             []DiscoveryError{},
		CacheHit:           false,
		Timestamp:          time.Now(),
	}

	// Scan directories
	for _, dir := range m.dirs {
		if err := m.scanDirectory(dir, result); err != nil {
			result.Errors = append(result.Errors, DiscoveryError{
				Path:  dir,
				Error: err.Error(),
			})
		}
	}

	result.ScanTime = time.Since(startTime)

	// Cache result if enabled
	if m.config.CacheResults && m.cache != nil {
		cacheKey := fmt.Sprintf("%v", m.dirs)
		m.mu.Lock()
		m.cache[cacheKey] = result
		m.mu.Unlock()
	}

	return result, nil
}

// scanDirectory scans a directory for plugins.
func (m *MockPluginDiscovery) scanDirectory(dir string, result *DiscoveryResult) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		fullPath := filepath.Join(dir, filename)
		result.FilesScanned++

		// Apply filters
		if m.shouldIncludeFile(filename) {
			plugin := PluginDiscovery{
				Path:      fullPath,
				Name:      filename,
				Available: true,
			}
			result.Plugins = append(result.Plugins, plugin)
		}
	}

	return nil
}

// shouldIncludeFile checks if a file should be included based on patterns.
func (m *MockPluginDiscovery) shouldIncludeFile(filename string) bool {
	// Check exclude patterns
	for _, pattern := range m.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return false
		}
	}

	// Check include patterns (if any)
	if len(m.config.IncludePatterns) > 0 {
		for _, pattern := range m.config.IncludePatterns {
			if matched, _ := filepath.Match(pattern, filename); matched {
				return true
			}
		}
		return false
	}

	return true
}

// Run all tests in the suite.
func TestPluginDiscoveryTestSuite(t *testing.T) {
	suite.Run(t, new(PluginDiscoveryTestSuite))
}
