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
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/testutil"
)

// PluginIntegrationTestSuite contains all integration tests for the plugin system.
type PluginIntegrationTestSuite struct {
	suite.Suite
	tempDir      string
	pluginDir    string
	manager      *DefaultPluginManager
	registry     *PluginRegistry
	loader       PluginLoader
	validator    *SecurityValidator
	mockLogger   *testutil.MockLogger
	pluginSystem *IntegratedPluginSystem
}

// IntegratedPluginSystem represents a fully integrated plugin system for testing.
type IntegratedPluginSystem struct {
	manager   *DefaultPluginManager
	registry  *PluginRegistry
	loader    PluginLoader
	validator *SecurityValidator
	discovery *MockPluginDiscovery
	config    *PluginSystemConfig
	mu        sync.RWMutex
}

// SetupSuite initializes the test suite.
func (suite *PluginIntegrationTestSuite) SetupSuite() {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "switctl-integration-test-*")
	require.NoError(suite.T(), err)
	suite.tempDir = tempDir
	suite.pluginDir = filepath.Join(tempDir, "plugins")

	// Create plugins directory
	err = os.MkdirAll(suite.pluginDir, 0755)
	require.NoError(suite.T(), err)
}

// TearDownSuite cleans up after all tests.
func (suite *PluginIntegrationTestSuite) TearDownSuite() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// SetupTest initializes each test case.
func (suite *PluginIntegrationTestSuite) SetupTest() {
	suite.mockLogger = testutil.NewMockLogger()

	// Setup all possible mock expectations with flexible argument matching
	suite.mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything).Return().Maybe()
	suite.mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

	// Create integrated plugin system
	suite.pluginSystem = suite.createIntegratedSystem()

	// Set individual components for direct access
	suite.manager = suite.pluginSystem.manager
	suite.registry = suite.pluginSystem.registry
	suite.loader = suite.pluginSystem.loader
	suite.validator = suite.pluginSystem.validator
}

// TearDownTest cleans up after each test.
func (suite *PluginIntegrationTestSuite) TearDownTest() {
	if suite.manager != nil {
		suite.manager.Shutdown()
	}

	// Clean plugin directory
	entries, _ := os.ReadDir(suite.pluginDir)
	for _, entry := range entries {
		os.Remove(filepath.Join(suite.pluginDir, entry.Name()))
	}
}

// createIntegratedSystem creates a fully integrated plugin system.
func (suite *PluginIntegrationTestSuite) createIntegratedSystem() *IntegratedPluginSystem {
	// Security configuration
	securityConfig := &SecurityConfig{
		AllowUnsigned:     true, // For testing
		RequireChecksum:   false,
		TrustedSources:    []string{suite.pluginDir},
		BlockedPatterns:   []string{"*.exe", "*.tmp"},
		MaxFileSizeMB:     50,
		AllowedExtensions: []string{".so", ".dll", ".dylib"},
	}

	// Manager configuration
	managerConfig := &ManagerConfig{
		PluginDirs:          []string{suite.pluginDir},
		LoadTimeout:         10 * time.Second,
		EnableAutoDiscovery: true,
		EnableVersionCheck:  true,
		MaxPlugins:          100, // Higher limit for integration tests
		IsolationMode:       "sandbox",
		SecurityPolicy: SecurityPolicy{
			AllowUnsigned:   true,
			RequireChecksum: false,
			TrustedSources:  []string{suite.pluginDir},
			BlockedPatterns: []string{"*.exe", "*.tmp"},
		},
	}

	// Discovery configuration
	discoveryConfig := &DiscoveryConfig{
		AutoDiscovery:    true,
		ScanInterval:     30 * time.Second,
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

	// System configuration
	systemConfig := &PluginSystemConfig{
		Manager:   managerConfig,
		Discovery: discoveryConfig,
		Security: &SecurityPolicy{
			AllowUnsigned:   securityConfig.AllowUnsigned,
			RequireChecksum: securityConfig.RequireChecksum,
			TrustedSources:  securityConfig.TrustedSources,
			BlockedPatterns: securityConfig.BlockedPatterns,
		},
		Global: &GlobalConfig{
			PluginVersion:      "1.0.0",
			FrameworkVersion:   "2.0.0",
			ConfigVersion:      "1.0",
			DefaultTimeout:     30 * time.Second,
			LogLevel:           "info",
			MetricsEnabled:     true,
			HealthCheckEnabled: true,
			Environment:        "test",
			Tags:               []string{"test", "integration"},
			Metadata:           map[string]string{"test": "true"},
		},
	}

	// Create components
	registry := NewPluginRegistry()
	loader := NewGoPluginLoader(suite.mockLogger)
	validator := NewSecurityValidator(securityConfig)
	manager := NewPluginManager(managerConfig, suite.mockLogger)
	discovery := &MockPluginDiscovery{
		dirs:   []string{suite.pluginDir},
		config: discoveryConfig,
		cache:  make(map[string]*DiscoveryResult),
	}

	return &IntegratedPluginSystem{
		manager:   manager,
		registry:  registry,
		loader:    loader,
		validator: validator,
		discovery: discovery,
		config:    systemConfig,
	}
}

// TestCompletePluginLifecycle tests the complete plugin lifecycle.
func (suite *PluginIntegrationTestSuite) TestCompletePluginLifecycle() {
	// Set up comprehensive mock expectations
	suite.mockLogger.On("Info", mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

	// 1. Initialize the plugin system
	err := suite.pluginSystem.manager.Initialize()
	require.NoError(suite.T(), err)

	// 2. Create and register a test plugin
	plugin := NewTestPlugin("lifecycle-plugin", "1.0.0")
	plugin.SetDescription("Complete lifecycle test plugin")

	err = suite.pluginSystem.manager.RegisterPlugin(plugin)
	require.NoError(suite.T(), err)

	// 3. Verify plugin registration
	retrievedPlugin, exists := suite.pluginSystem.manager.GetPlugin("lifecycle-plugin")
	assert.True(suite.T(), exists)
	assert.Equal(suite.T(), plugin, retrievedPlugin)

	// 4. Get plugin info from registry
	info, err := suite.pluginSystem.manager.GetPluginInfo("lifecycle-plugin")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "lifecycle-plugin", info.Metadata.Name)
	assert.Equal(suite.T(), "1.0.0", info.Metadata.Version)
	assert.Equal(suite.T(), PluginStatusInitialized, info.Status)

	// 5. Execute the plugin
	ctx := context.Background()
	args := []string{"test", "args"}
	err = suite.pluginSystem.manager.ExecutePlugin("lifecycle-plugin", ctx, args)
	assert.NoError(suite.T(), err)

	// 6. Verify execution
	assert.Equal(suite.T(), 1, plugin.GetExecuteCount())
	assert.Equal(suite.T(), args, plugin.GetLastArgs())

	// 7. Disable and re-enable plugin
	err = suite.pluginSystem.manager.DisablePlugin("lifecycle-plugin")
	assert.NoError(suite.T(), err)

	info, _ = suite.pluginSystem.manager.GetPluginInfo("lifecycle-plugin")
	assert.Equal(suite.T(), PluginStatusDisabled, info.Status)

	// Execution should fail when disabled
	err = suite.pluginSystem.manager.ExecutePlugin("lifecycle-plugin", ctx, args)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "disabled")

	// Re-enable
	err = suite.pluginSystem.manager.EnablePlugin("lifecycle-plugin")
	assert.NoError(suite.T(), err)

	// 8. Unload plugin
	err = suite.pluginSystem.manager.UnloadPlugin("lifecycle-plugin")
	assert.NoError(suite.T(), err)

	// 9. Verify plugin is unloaded
	_, exists = suite.pluginSystem.manager.GetPlugin("lifecycle-plugin")
	assert.False(suite.T(), exists)

	// 10. Shutdown system
	err = suite.pluginSystem.manager.Shutdown()
	assert.NoError(suite.T(), err)
}

// TestPluginSystemIntegration tests integration between all components.
func (suite *PluginIntegrationTestSuite) TestPluginSystemIntegration() {
	// Set up comprehensive mock expectations
	suite.mockLogger.On("Info", mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

	// 1. Initialize system
	err := suite.pluginSystem.Initialize()
	require.NoError(suite.T(), err)

	// 2. Create and register multiple plugins
	plugin1 := NewTestPlugin("integration-plugin1", "1.0.0")
	plugin2 := NewTestPlugin("integration-plugin2", "2.0.0")

	err = suite.pluginSystem.RegisterPlugin(plugin1)
	require.NoError(suite.T(), err)

	err = suite.pluginSystem.RegisterPlugin(plugin2)
	require.NoError(suite.T(), err)

	// 3. Test registry functionality
	allPlugins := suite.pluginSystem.manager.registry.GetAllPluginInfo()
	assert.Len(suite.T(), allPlugins, 2)

	pluginNames := suite.pluginSystem.manager.ListPlugins()
	assert.Len(suite.T(), pluginNames, 2)
	assert.Contains(suite.T(), pluginNames, "integration-plugin1")
	assert.Contains(suite.T(), pluginNames, "integration-plugin2")

	// 4. Test security validation
	testFile := filepath.Join(suite.pluginDir, "test-validation.so")
	content := []byte("test plugin content for validation")
	err = os.WriteFile(testFile, content, 0644)
	require.NoError(suite.T(), err)

	result := suite.pluginSystem.validator.ValidatePluginFile(testFile)
	assert.True(suite.T(), result.Valid)
	assert.NotEmpty(suite.T(), result.Checksum)

	// 5. Test discovery
	// Create fake plugin files
	suite.createTestPluginFile("discovery-plugin1.so")
	suite.createTestPluginFile("discovery-plugin2.dll")

	discoveryResult, err := suite.pluginSystem.discovery.DiscoverPlugins()
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), discoveryResult.Plugins, 3) // Including test-validation.so

	// 6. Test concurrent operations
	const numOperations = 10
	var wg sync.WaitGroup
	errors := make(chan error, numOperations)

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Test concurrent plugin execution
			pluginName := fmt.Sprintf("integration-plugin%d", (id%2)+1)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := suite.pluginSystem.ExecutePlugin(pluginName, ctx, []string{fmt.Sprintf("arg%d", id)})
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		assert.NoError(suite.T(), err, "Concurrent operations should not fail")
	}
}

// TestPluginSystemRecovery tests system recovery from failures.
func (suite *PluginIntegrationTestSuite) TestPluginSystemRecovery() {
	// Set up mock expectations
	suite.mockLogger.On("Info", "Initializing plugin manager", "dirs", []string{suite.pluginDir}).Return()
	suite.mockLogger.On("Info", "Plugin manager initialized successfully").Return()
	suite.mockLogger.On("Info", "Plugin registered successfully", "name", "recovery-plugin", "version", "1.0.0").Return()
	suite.mockLogger.On("Warn", "Plugin cleanup failed during shutdown", "name", "failing-plugin", "error", "cleanup failed").Return()
	suite.mockLogger.On("Info", "Plugin manager shutdown completed").Return()

	// Initialize system
	err := suite.pluginSystem.Initialize()
	require.NoError(suite.T(), err)

	// 1. Test recovery from plugin execution failure
	goodPlugin := NewTestPlugin("recovery-plugin", "1.0.0")
	execError := fmt.Errorf("execution failed")
	badPlugin := NewTestPluginWithError("failing-plugin", "1.0.0", execError, nil)

	err = suite.pluginSystem.RegisterPlugin(goodPlugin)
	require.NoError(suite.T(), err)

	err = suite.pluginSystem.RegisterPlugin(badPlugin)
	require.NoError(suite.T(), err)

	// Execute good plugin - should work
	err = suite.pluginSystem.ExecutePlugin("recovery-plugin", context.Background(), []string{})
	assert.NoError(suite.T(), err)

	// Execute bad plugin - should fail gracefully
	err = suite.pluginSystem.ExecutePlugin("failing-plugin", context.Background(), []string{})
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "execution failed")

	// System should still be operational
	err = suite.pluginSystem.ExecutePlugin("recovery-plugin", context.Background(), []string{})
	assert.NoError(suite.T(), err)

	// 2. Test recovery from cleanup failure during shutdown
	cleanupError := fmt.Errorf("cleanup failed")
	badPlugin.cleanupError = cleanupError

	err = suite.pluginSystem.Shutdown()
	assert.Error(suite.T(), err) // Should report cleanup errors but not crash
	assert.Contains(suite.T(), err.Error(), "Some plugins failed to cleanup")
}

// TestPluginSystemPerformance tests basic performance characteristics.
func (suite *PluginIntegrationTestSuite) TestPluginSystemPerformance() {
	// Set up mock expectations for performance testing
	suite.mockLogger.On("Info", "Initializing plugin manager", "dirs", []string{suite.pluginDir}).Return()
	suite.mockLogger.On("Info", "Plugin manager initialized successfully").Return()

	// Initialize system
	startTime := time.Now()
	err := suite.pluginSystem.Initialize()
	require.NoError(suite.T(), err)
	initTime := time.Since(startTime)

	// Should initialize quickly
	assert.Less(suite.T(), initTime, 1*time.Second, "System initialization should be fast")

	// Register multiple plugins and measure performance
	const numPlugins = 50
	plugins := make([]*TestPlugin, numPlugins)

	startTime = time.Now()
	for i := 0; i < numPlugins; i++ {
		plugins[i] = NewTestPlugin(fmt.Sprintf("perf-plugin-%d", i), "1.0.0")
		err = suite.pluginSystem.RegisterPlugin(plugins[i])
		require.NoError(suite.T(), err)
	}
	registerTime := time.Since(startTime)

	// Registration should be reasonably fast
	avgRegisterTime := registerTime / numPlugins
	assert.Less(suite.T(), avgRegisterTime, 10*time.Millisecond, "Plugin registration should be efficient")

	// Test execution performance
	startTime = time.Now()
	for i := 0; i < numPlugins; i++ {
		err = suite.pluginSystem.ExecutePlugin(fmt.Sprintf("perf-plugin-%d", i), context.Background(), []string{})
		require.NoError(suite.T(), err)
	}
	execTime := time.Since(startTime)

	avgExecTime := execTime / numPlugins
	assert.Less(suite.T(), avgExecTime, 50*time.Millisecond, "Plugin execution should be efficient")

	// Test concurrent execution performance
	startTime = time.Now()
	var wg sync.WaitGroup
	for i := 0; i < numPlugins; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			suite.pluginSystem.ExecutePlugin(fmt.Sprintf("perf-plugin-%d", id), context.Background(), []string{})
		}(i)
	}
	wg.Wait()
	concurrentExecTime := time.Since(startTime)

	// Concurrent execution should be faster than sequential
	assert.Less(suite.T(), concurrentExecTime, execTime, "Concurrent execution should be more efficient")
}

// TestPluginSystemMemoryManagement tests memory management.
func (suite *PluginIntegrationTestSuite) TestPluginSystemMemoryManagement() {
	// Initialize system
	err := suite.pluginSystem.Initialize()
	require.NoError(suite.T(), err)

	// Create and register plugins
	const numPlugins = 10
	for i := 0; i < numPlugins; i++ {
		plugin := NewTestPlugin(fmt.Sprintf("memory-plugin-%d", i), "1.0.0")
		err = suite.pluginSystem.RegisterPlugin(plugin)
		require.NoError(suite.T(), err)
	}

	// Verify all plugins are registered
	assert.Len(suite.T(), suite.pluginSystem.manager.ListPlugins(), numPlugins)

	// Unload half the plugins
	for i := 0; i < numPlugins/2; i++ {
		err = suite.pluginSystem.UnloadPlugin(fmt.Sprintf("memory-plugin-%d", i))
		assert.NoError(suite.T(), err)
	}

	// Verify only remaining plugins are present
	assert.Len(suite.T(), suite.pluginSystem.manager.ListPlugins(), numPlugins/2)

	// Shutdown and verify cleanup
	err = suite.pluginSystem.Shutdown()
	assert.NoError(suite.T(), err)

	// After shutdown, no plugins should be present
	assert.Len(suite.T(), suite.pluginSystem.manager.ListPlugins(), 0)
}

// TestPluginSystemValidation tests system-wide validation.
func (suite *PluginIntegrationTestSuite) TestPluginSystemValidation() {
	// Test configuration validation
	assert.NotNil(suite.T(), suite.pluginSystem.config)
	assert.NotNil(suite.T(), suite.pluginSystem.config.Manager)
	assert.NotNil(suite.T(), suite.pluginSystem.config.Discovery)
	assert.NotNil(suite.T(), suite.pluginSystem.config.Security)
	assert.NotNil(suite.T(), suite.pluginSystem.config.Global)

	// Test component integration validation
	assert.NotNil(suite.T(), suite.pluginSystem.manager)
	assert.NotNil(suite.T(), suite.pluginSystem.registry)
	assert.NotNil(suite.T(), suite.pluginSystem.loader)
	assert.NotNil(suite.T(), suite.pluginSystem.validator)
	assert.NotNil(suite.T(), suite.pluginSystem.discovery)

	// Test security policy enforcement
	securityPolicy := suite.pluginSystem.config.Security
	assert.False(suite.T(), securityPolicy.RequireChecksum) // Set to false for testing
	assert.Contains(suite.T(), securityPolicy.TrustedSources, suite.pluginDir)
	assert.Contains(suite.T(), securityPolicy.BlockedPatterns, "*.exe")
}

// Helper methods

// createTestPluginFile creates a test plugin file.
func (suite *PluginIntegrationTestSuite) createTestPluginFile(filename string) string {
	pluginPath := filepath.Join(suite.pluginDir, filename)
	content := fmt.Sprintf("// Plugin: %s\n// This is a test plugin file", filename)
	err := os.WriteFile(pluginPath, []byte(content), 0644)
	require.NoError(suite.T(), err)
	return pluginPath
}

// IntegratedPluginSystem methods

// Initialize initializes the integrated plugin system.
func (ips *IntegratedPluginSystem) Initialize() error {
	return ips.manager.Initialize()
}

// RegisterPlugin registers a plugin with the system.
func (ips *IntegratedPluginSystem) RegisterPlugin(plugin *TestPlugin) error {
	return ips.manager.RegisterPlugin(plugin)
}

// ExecutePlugin executes a plugin by name.
func (ips *IntegratedPluginSystem) ExecutePlugin(name string, ctx context.Context, args []string) error {
	return ips.manager.ExecutePlugin(name, ctx, args)
}

// UnloadPlugin unloads a plugin by name.
func (ips *IntegratedPluginSystem) UnloadPlugin(name string) error {
	return ips.manager.UnloadPlugin(name)
}

// Shutdown shuts down the integrated plugin system.
func (ips *IntegratedPluginSystem) Shutdown() error {
	return ips.manager.Shutdown()
}

// Run all tests in the suite.
func TestPluginIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PluginIntegrationTestSuite))
}
