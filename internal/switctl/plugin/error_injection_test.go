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

	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/innovationmech/swit/internal/switctl/testutil"
)

// PluginErrorInjectionTestSuite contains all error injection and recovery tests.
type PluginErrorInjectionTestSuite struct {
	suite.Suite
	tempDir        string
	pluginDir      string
	manager        *DefaultPluginManager
	registry       *PluginRegistry
	mockLogger     *testutil.MockLogger
	errorSimulator *ErrorSimulator
}

// ErrorSimulator simulates various types of errors for testing recovery mechanisms.
type ErrorSimulator struct {
	enabledErrors map[string]bool
	errorCounters map[string]int
	mu            sync.RWMutex
}

// NewErrorSimulator creates a new error simulator.
func NewErrorSimulator() *ErrorSimulator {
	return &ErrorSimulator{
		enabledErrors: make(map[string]bool),
		errorCounters: make(map[string]int),
	}
}

// EnableError enables a specific type of error.
func (es *ErrorSimulator) EnableError(errorType string) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.enabledErrors[errorType] = true
}

// DisableError disables a specific type of error.
func (es *ErrorSimulator) DisableError(errorType string) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.enabledErrors[errorType] = false
}

// ShouldTriggerError checks if an error should be triggered.
func (es *ErrorSimulator) ShouldTriggerError(errorType string) bool {
	es.mu.Lock()
	defer es.mu.Unlock()

	if enabled, exists := es.enabledErrors[errorType]; exists && enabled {
		es.errorCounters[errorType]++
		return true
	}
	return false
}

// GetErrorCount returns the number of times an error was triggered.
func (es *ErrorSimulator) GetErrorCount(errorType string) int {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return es.errorCounters[errorType]
}

// Reset resets all error states.
func (es *ErrorSimulator) Reset() {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.enabledErrors = make(map[string]bool)
	es.errorCounters = make(map[string]int)
}

// FaultTolerantPlugin is a test plugin that can simulate various failures.
type FaultTolerantPlugin struct {
	*TestPlugin
	simulator *ErrorSimulator
}

// NewFaultTolerantPlugin creates a new fault-tolerant test plugin.
func NewFaultTolerantPlugin(name, version string, simulator *ErrorSimulator) *FaultTolerantPlugin {
	return &FaultTolerantPlugin{
		TestPlugin: NewTestPlugin(name, version),
		simulator:  simulator,
	}
}

// Initialize simulates initialization failures.
func (p *FaultTolerantPlugin) Initialize(config interfaces.PluginConfig) error {
	if p.simulator.ShouldTriggerError("init_failure") {
		return fmt.Errorf("simulated initialization failure")
	}
	return p.TestPlugin.Initialize(config)
}

// Execute simulates execution failures.
func (p *FaultTolerantPlugin) Execute(ctx context.Context, args []string) error {
	if p.simulator.ShouldTriggerError("exec_failure") {
		return fmt.Errorf("simulated execution failure")
	}

	if p.simulator.ShouldTriggerError("exec_timeout") {
		time.Sleep(100 * time.Millisecond) // Simulate long-running operation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	if p.simulator.ShouldTriggerError("exec_panic") {
		// Simulate panic recovery
		defer func() {
			if r := recover(); r != nil {
				// Convert panic to error
			}
		}()
		panic("simulated plugin panic")
	}

	return p.TestPlugin.Execute(ctx, args)
}

// Cleanup simulates cleanup failures.
func (p *FaultTolerantPlugin) Cleanup() error {
	if p.simulator.ShouldTriggerError("cleanup_failure") {
		return fmt.Errorf("simulated cleanup failure")
	}
	return p.TestPlugin.Cleanup()
}

// SetupSuite initializes the test suite.
func (suite *PluginErrorInjectionTestSuite) SetupSuite() {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "switctl-error-test-*")
	require.NoError(suite.T(), err)
	suite.tempDir = tempDir
	suite.pluginDir = filepath.Join(tempDir, "plugins")

	// Create plugins directory
	err = os.MkdirAll(suite.pluginDir, 0755)
	require.NoError(suite.T(), err)
}

// TearDownSuite cleans up after all tests.
func (suite *PluginErrorInjectionTestSuite) TearDownSuite() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// SetupTest initializes each test case.
func (suite *PluginErrorInjectionTestSuite) SetupTest() {
	suite.mockLogger = testutil.NewMockLogger()
	suite.errorSimulator = NewErrorSimulator()
	suite.setupMockExpectations()

	config := &ManagerConfig{
		PluginDirs:          []string{suite.pluginDir},
		LoadTimeout:         5 * time.Second,
		EnableAutoDiscovery: true,
		EnableVersionCheck:  true,
		MaxPlugins:          50,
		IsolationMode:       "sandbox",
		SecurityPolicy: SecurityPolicy{
			AllowUnsigned:   true,
			RequireChecksum: false,
		},
	}

	suite.manager = NewPluginManager(config, suite.mockLogger)
	suite.registry = NewPluginRegistry()
}

// TearDownTest cleans up after each test.
func (suite *PluginErrorInjectionTestSuite) TearDownTest() {
	if suite.manager != nil {
		suite.manager.Shutdown()
	}
	suite.errorSimulator.Reset()
}

// setupMockExpectations sets up mock expectations.
func (suite *PluginErrorInjectionTestSuite) setupMockExpectations() {
	// Use catch-all expectations for error injection tests
	suite.mockLogger.On("Info", mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

	suite.mockLogger.On("Debug", mock.Anything).Return().Maybe()
	suite.mockLogger.On("Debug", mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

	suite.mockLogger.On("Warn", mock.Anything).Return().Maybe()
	suite.mockLogger.On("Warn", mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

	suite.mockLogger.On("Error", mock.Anything).Return().Maybe()
	suite.mockLogger.On("Error", mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
}

// TestInitializationFailureRecovery tests recovery from initialization failures.
func (suite *PluginErrorInjectionTestSuite) TestInitializationFailureRecovery() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	// Enable initialization failure
	suite.errorSimulator.EnableError("init_failure")

	// Try to register a plugin that will fail initialization
	faultyPlugin := NewFaultTolerantPlugin("faulty-plugin", "1.0.0", suite.errorSimulator)
	err = suite.manager.RegisterPlugin(faultyPlugin)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "Plugin initialization failed")

	// Plugin should not be registered
	_, exists := suite.manager.GetPlugin("faulty-plugin")
	assert.False(suite.T(), exists)

	// Disable error and try again
	suite.errorSimulator.DisableError("init_failure")
	err = suite.manager.RegisterPlugin(faultyPlugin)
	assert.NoError(suite.T(), err)

	// Now it should be registered
	_, exists = suite.manager.GetPlugin("faulty-plugin")
	assert.True(suite.T(), exists)

	// Verify error was triggered
	assert.Equal(suite.T(), 1, suite.errorSimulator.GetErrorCount("init_failure"))
}

// TestExecutionFailureRecovery tests recovery from execution failures.
func (suite *PluginErrorInjectionTestSuite) TestExecutionFailureRecovery() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	// Register a fault-tolerant plugin
	plugin := NewFaultTolerantPlugin("execution-test", "1.0.0", suite.errorSimulator)
	err = suite.manager.RegisterPlugin(plugin)
	require.NoError(suite.T(), err)

	// First execution should succeed
	err = suite.manager.ExecutePlugin("execution-test", context.Background(), []string{"test"})
	assert.NoError(suite.T(), err)

	// Enable execution failure
	suite.errorSimulator.EnableError("exec_failure")

	// Execution should fail
	err = suite.manager.ExecutePlugin("execution-test", context.Background(), []string{"test"})
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "Plugin execution failed")

	// Plugin should still be registered and functional after failure
	_, exists := suite.manager.GetPlugin("execution-test")
	assert.True(suite.T(), exists)

	// Disable error and try again
	suite.errorSimulator.DisableError("exec_failure")
	err = suite.manager.ExecutePlugin("execution-test", context.Background(), []string{"test"})
	assert.NoError(suite.T(), err)

	// Verify error count
	assert.Equal(suite.T(), 1, suite.errorSimulator.GetErrorCount("exec_failure"))
}

// TestCleanupFailureRecovery tests recovery from cleanup failures.
func (suite *PluginErrorInjectionTestSuite) TestCleanupFailureRecovery() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	// Register a fault-tolerant plugin
	plugin := NewFaultTolerantPlugin("cleanup-test", "1.0.0", suite.errorSimulator)
	err = suite.manager.RegisterPlugin(plugin)
	require.NoError(suite.T(), err)

	// Enable cleanup failure
	suite.errorSimulator.EnableError("cleanup_failure")

	// Unload should handle cleanup failure gracefully
	err = suite.manager.UnloadPlugin("cleanup-test")
	assert.NoError(suite.T(), err) // Manager should handle cleanup failures gracefully

	// Plugin should still be unloaded despite cleanup failure
	_, exists := suite.manager.GetPlugin("cleanup-test")
	assert.False(suite.T(), exists)

	// Verify error was triggered
	assert.Equal(suite.T(), 1, suite.errorSimulator.GetErrorCount("cleanup_failure"))
}

// TestPartialSystemFailure tests recovery from partial system failures.
func (suite *PluginErrorInjectionTestSuite) TestPartialSystemFailure() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	// Register multiple plugins
	goodPlugin1 := NewTestPlugin("good-plugin-1", "1.0.0")
	goodPlugin2 := NewTestPlugin("good-plugin-2", "1.0.0")
	faultyPlugin := NewFaultTolerantPlugin("faulty-plugin", "1.0.0", suite.errorSimulator)

	// Register all plugins
	err = suite.manager.RegisterPlugin(goodPlugin1)
	require.NoError(suite.T(), err)

	err = suite.manager.RegisterPlugin(faultyPlugin)
	require.NoError(suite.T(), err)

	err = suite.manager.RegisterPlugin(goodPlugin2)
	require.NoError(suite.T(), err)

	// All plugins should be registered
	assert.Len(suite.T(), suite.manager.ListPlugins(), 3)

	// Enable execution failure for the faulty plugin
	suite.errorSimulator.EnableError("exec_failure")

	// Good plugins should still work
	err = suite.manager.ExecutePlugin("good-plugin-1", context.Background(), []string{"test"})
	assert.NoError(suite.T(), err)

	err = suite.manager.ExecutePlugin("good-plugin-2", context.Background(), []string{"test"})
	assert.NoError(suite.T(), err)

	// Faulty plugin should fail
	err = suite.manager.ExecutePlugin("faulty-plugin", context.Background(), []string{"test"})
	assert.Error(suite.T(), err)

	// System should remain stable
	assert.Len(suite.T(), suite.manager.ListPlugins(), 3)

	// Other plugins should still work after one fails
	err = suite.manager.ExecutePlugin("good-plugin-1", context.Background(), []string{"test-after-failure"})
	assert.NoError(suite.T(), err)
}

// TestTimeoutRecovery tests recovery from timeout scenarios.
func (suite *PluginErrorInjectionTestSuite) TestTimeoutRecovery() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	// Register a plugin that can simulate timeouts
	plugin := NewFaultTolerantPlugin("timeout-test", "1.0.0", suite.errorSimulator)
	err = suite.manager.RegisterPlugin(plugin)
	require.NoError(suite.T(), err)

	// Enable timeout simulation
	suite.errorSimulator.EnableError("exec_timeout")

	// Execute with short timeout - should fail
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err = suite.manager.ExecutePlugin("timeout-test", ctx, []string{"timeout-test"})
	elapsed := time.Since(start)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "Plugin execution failed")
	assert.Less(suite.T(), elapsed, 200*time.Millisecond, "Should timeout quickly")

	// Plugin should still be functional after timeout
	_, exists := suite.manager.GetPlugin("timeout-test")
	assert.True(suite.T(), exists)

	// Disable timeout and try again with longer timeout
	suite.errorSimulator.DisableError("exec_timeout")
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel2()

	err = suite.manager.ExecutePlugin("timeout-test", ctx2, []string{"normal-test"})
	assert.NoError(suite.T(), err)
}

// TestConcurrentFailureRecovery tests recovery from concurrent failures.
func (suite *PluginErrorInjectionTestSuite) TestConcurrentFailureRecovery() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	const numPlugins = 10
	const numWorkers = 20

	// Register multiple plugins
	plugins := make([]*FaultTolerantPlugin, numPlugins)
	for i := 0; i < numPlugins; i++ {
		plugins[i] = NewFaultTolerantPlugin(fmt.Sprintf("concurrent-plugin-%d", i), "1.0.0", suite.errorSimulator)
		err := suite.manager.RegisterPlugin(plugins[i])
		require.NoError(suite.T(), err)
	}

	// Enable random execution failures
	suite.errorSimulator.EnableError("exec_failure")

	var wg sync.WaitGroup
	var successCount, failureCount int64
	resultChan := make(chan bool, numWorkers)

	// Launch concurrent executions
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			pluginIndex := workerID % numPlugins
			pluginName := fmt.Sprintf("concurrent-plugin-%d", pluginIndex)

			err := suite.manager.ExecutePlugin(pluginName, context.Background(), []string{fmt.Sprintf("worker-%d", workerID)})

			if err != nil {
				resultChan <- false
			} else {
				resultChan <- true
			}
		}(w)
	}

	wg.Wait()
	close(resultChan)

	// Count results
	for result := range resultChan {
		if result {
			successCount++
		} else {
			failureCount++
		}
	}

	suite.T().Logf("Concurrent executions: %d successful, %d failed", successCount, failureCount)

	// Some should succeed and some should fail due to error injection
	assert.Greater(suite.T(), failureCount, int64(0), "Some executions should fail due to error injection")

	// System should remain stable
	assert.Len(suite.T(), suite.manager.ListPlugins(), numPlugins)

	// All plugins should still be registered and accessible
	for i := 0; i < numPlugins; i++ {
		pluginName := fmt.Sprintf("concurrent-plugin-%d", i)
		_, exists := suite.manager.GetPlugin(pluginName)
		assert.True(suite.T(), exists, "Plugin %s should still be registered", pluginName)
	}
}

// TestSystemRecoveryAfterShutdown tests system recovery after shutdown errors.
func (suite *PluginErrorInjectionTestSuite) TestSystemRecoveryAfterShutdown() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	// Register plugins with cleanup failures
	plugin1 := NewFaultTolerantPlugin("shutdown-plugin-1", "1.0.0", suite.errorSimulator)
	plugin2 := NewTestPlugin("shutdown-plugin-2", "1.0.0")

	err = suite.manager.RegisterPlugin(plugin1)
	require.NoError(suite.T(), err)

	err = suite.manager.RegisterPlugin(plugin2)
	require.NoError(suite.T(), err)

	// Enable cleanup failure
	suite.errorSimulator.EnableError("cleanup_failure")

	// Shutdown should handle cleanup failures gracefully
	err = suite.manager.Shutdown()
	assert.Error(suite.T(), err) // Should report cleanup errors
	assert.Contains(suite.T(), err.Error(), "Some plugins failed to cleanup")

	// Manager should be in shutdown state
	assert.Len(suite.T(), suite.manager.ListPlugins(), 0)

	// Verify error was triggered
	assert.Greater(suite.T(), suite.errorSimulator.GetErrorCount("cleanup_failure"), 0)
}

// TestErrorPropagation tests proper error propagation through the system.
func (suite *PluginErrorInjectionTestSuite) TestErrorPropagation() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	// Register a plugin that will fail
	plugin := NewFaultTolerantPlugin("error-prop-test", "1.0.0", suite.errorSimulator)
	err = suite.manager.RegisterPlugin(plugin)
	require.NoError(suite.T(), err)

	// Enable execution failure
	suite.errorSimulator.EnableError("exec_failure")

	// Execute and verify error propagation
	err = suite.manager.ExecutePlugin("error-prop-test", context.Background(), []string{"test"})
	assert.Error(suite.T(), err)

	// Error should be wrapped with plugin context
	assert.Contains(suite.T(), err.Error(), "Plugin execution failed")
	assert.Contains(suite.T(), err.Error(), "error-prop-test")

	// Verify it's a plugin error
	pluginErr, ok := err.(*interfaces.SwitctlError)
	assert.True(suite.T(), ok, "Error should be of type PluginError")
	if ok {
		assert.Equal(suite.T(), interfaces.ErrCodePluginExecError, pluginErr.Code)
	}
}

// TestResourceLeakPrevention tests prevention of resource leaks during failures.
func (suite *PluginErrorInjectionTestSuite) TestResourceLeakPrevention() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	const numIterations = 20

	// Repeatedly register and fail plugins to test for leaks
	for i := 0; i < numIterations; i++ {
		// Enable initialization failure
		suite.errorSimulator.EnableError("init_failure")

		plugin := NewFaultTolerantPlugin(fmt.Sprintf("leak-test-%d", i), "1.0.0", suite.errorSimulator)
		err := suite.manager.RegisterPlugin(plugin)
		assert.Error(suite.T(), err)

		// Plugin should not be registered due to failure
		_, exists := suite.manager.GetPlugin(fmt.Sprintf("leak-test-%d", i))
		assert.False(suite.T(), exists)

		// Disable error and register successfully
		suite.errorSimulator.DisableError("init_failure")
		err = suite.manager.RegisterPlugin(plugin)
		assert.NoError(suite.T(), err)

		// Unload plugin to clean up
		err = suite.manager.UnloadPlugin(fmt.Sprintf("leak-test-%d", i))
		assert.NoError(suite.T(), err)
	}

	// No plugins should remain registered
	assert.Len(suite.T(), suite.manager.ListPlugins(), 0)

	// Verify error counts
	assert.Equal(suite.T(), numIterations, suite.errorSimulator.GetErrorCount("init_failure"))
}

// TestCircuitBreakerPattern tests circuit breaker pattern for failing plugins.
func (suite *PluginErrorInjectionTestSuite) TestCircuitBreakerPattern() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	// Register a plugin that will consistently fail
	plugin := NewFaultTolerantPlugin("circuit-test", "1.0.0", suite.errorSimulator)
	err = suite.manager.RegisterPlugin(plugin)
	require.NoError(suite.T(), err)

	// Enable execution failure
	suite.errorSimulator.EnableError("exec_failure")

	// Execute multiple times - all should fail
	const numFailures = 5
	for i := 0; i < numFailures; i++ {
		err = suite.manager.ExecutePlugin("circuit-test", context.Background(), []string{fmt.Sprintf("attempt-%d", i)})
		assert.Error(suite.T(), err)
	}

	// Verify all failures were recorded
	assert.Equal(suite.T(), numFailures, suite.errorSimulator.GetErrorCount("exec_failure"))

	// Plugin should still be accessible (no circuit breaker implemented yet)
	_, exists := suite.manager.GetPlugin("circuit-test")
	assert.True(suite.T(), exists)

	// Disable error - plugin should work again
	suite.errorSimulator.DisableError("exec_failure")
	err = suite.manager.ExecutePlugin("circuit-test", context.Background(), []string{"recovery-test"})
	assert.NoError(suite.T(), err)
}

// TestGracefulDegradation tests graceful system degradation under errors.
func (suite *PluginErrorInjectionTestSuite) TestGracefulDegradation() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	// Register critical and non-critical plugins
	criticalPlugin := NewTestPlugin("critical-plugin", "1.0.0")
	nonCriticalPlugin := NewFaultTolerantPlugin("non-critical-plugin", "1.0.0", suite.errorSimulator)

	err = suite.manager.RegisterPlugin(criticalPlugin)
	require.NoError(suite.T(), err)

	err = suite.manager.RegisterPlugin(nonCriticalPlugin)
	require.NoError(suite.T(), err)

	// Enable failure for non-critical plugin
	suite.errorSimulator.EnableError("exec_failure")

	// Critical plugin should continue working
	err = suite.manager.ExecutePlugin("critical-plugin", context.Background(), []string{"test"})
	assert.NoError(suite.T(), err)

	// Non-critical plugin should fail gracefully
	err = suite.manager.ExecutePlugin("non-critical-plugin", context.Background(), []string{"test"})
	assert.Error(suite.T(), err)

	// System should remain operational
	plugins := suite.manager.ListPlugins()
	assert.Len(suite.T(), plugins, 2)

	// Critical functionality should still work
	err = suite.manager.ExecutePlugin("critical-plugin", context.Background(), []string{"post-failure-test"})
	assert.NoError(suite.T(), err)
}

// Run all tests in the suite.
func TestPluginErrorInjectionTestSuite(t *testing.T) {
	suite.Run(t, new(PluginErrorInjectionTestSuite))
}
