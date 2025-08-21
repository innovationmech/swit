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
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/testutil"
)

// PluginStressTestSuite contains all stress tests for the plugin system.
type PluginStressTestSuite struct {
	suite.Suite
	tempDir    string
	pluginDir  string
	manager    *DefaultPluginManager
	registry   *PluginRegistry
	validator  *SecurityValidator
	mockLogger *testutil.MockLogger
}

// SetupSuite initializes the test suite.
func (suite *PluginStressTestSuite) SetupSuite() {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "switctl-stress-test-*")
	require.NoError(suite.T(), err)
	suite.tempDir = tempDir
	suite.pluginDir = filepath.Join(tempDir, "plugins")

	// Create plugins directory
	err = os.MkdirAll(suite.pluginDir, 0755)
	require.NoError(suite.T(), err)
}

// TearDownSuite cleans up after all tests.
func (suite *PluginStressTestSuite) TearDownSuite() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// SetupTest initializes each test case.
func (suite *PluginStressTestSuite) SetupTest() {
	suite.mockLogger = testutil.NewMockLogger()
	suite.setupMockExpectations()

	config := &ManagerConfig{
		PluginDirs:          []string{suite.pluginDir},
		LoadTimeout:         30 * time.Second,
		EnableAutoDiscovery: true,
		EnableVersionCheck:  true,
		MaxPlugins:          1000, // High limit for stress testing
		IsolationMode:       "sandbox",
		SecurityPolicy: SecurityPolicy{
			AllowUnsigned:   true,
			RequireChecksum: false,
		},
	}

	suite.manager = NewPluginManager(config, suite.mockLogger)
	suite.registry = NewPluginRegistry()
	suite.validator = NewSecurityValidator(&SecurityConfig{
		AllowUnsigned:     true,
		RequireChecksum:   false,
		MaxFileSizeMB:     100,
		AllowedExtensions: []string{".so", ".dll", ".dylib"},
	})
}

// TearDownTest cleans up after each test.
func (suite *PluginStressTestSuite) TearDownTest() {
	if suite.manager != nil {
		suite.manager.Shutdown()
	}
}

// setupMockExpectations sets up all mock expectations for stress tests.
func (suite *PluginStressTestSuite) setupMockExpectations() {
	// Use catch-all expectations for stress tests
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
	suite.mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

	suite.mockLogger.On("Warn", mock.Anything).Return().Maybe()
	suite.mockLogger.On("Warn", mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

	suite.mockLogger.On("Error", mock.Anything).Return().Maybe()
	suite.mockLogger.On("Error", mock.Anything, mock.Anything).Return().Maybe()
	suite.mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
}

// TestHighVolumePluginRegistration tests registration of many plugins.
func (suite *PluginStressTestSuite) TestHighVolumePluginRegistration() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	const numPlugins = 500
	plugins := make([]*TestPlugin, numPlugins)

	startTime := time.Now()

	// Register plugins sequentially
	for i := 0; i < numPlugins; i++ {
		plugins[i] = NewTestPlugin(fmt.Sprintf("stress-plugin-%d", i), "1.0.0")
		err := suite.manager.RegisterPlugin(plugins[i])
		require.NoError(suite.T(), err)
	}

	registrationTime := time.Since(startTime)
	suite.T().Logf("Registered %d plugins in %v (avg: %v per plugin)",
		numPlugins, registrationTime, registrationTime/time.Duration(numPlugins))

	// Verify all plugins are registered
	pluginList := suite.manager.ListPlugins()
	assert.Len(suite.T(), pluginList, numPlugins)

	// Average registration time should be reasonable
	avgTime := registrationTime / time.Duration(numPlugins)
	assert.Less(suite.T(), avgTime, 10*time.Millisecond, "Average registration time should be efficient")
}

// TestConcurrentPluginOperations tests high-concurrency operations.
func (suite *PluginStressTestSuite) TestConcurrentPluginOperations() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	const numPlugins = 100
	const numWorkers = 20
	const operationsPerWorker = 50

	// Pre-register plugins
	plugins := make([]*TestPlugin, numPlugins)
	for i := 0; i < numPlugins; i++ {
		plugins[i] = NewTestPlugin(fmt.Sprintf("concurrent-plugin-%d", i), "1.0.0")
		err := suite.manager.RegisterPlugin(plugins[i])
		require.NoError(suite.T(), err)
	}

	var totalOperations int64
	var successfulOperations int64
	var wg sync.WaitGroup

	startTime := time.Now()

	// Launch concurrent workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for op := 0; op < operationsPerWorker; op++ {
				atomic.AddInt64(&totalOperations, 1)

				// Random operations
				switch rand.Intn(4) {
				case 0: // Execute plugin
					pluginIdx := rand.Intn(numPlugins)
					pluginName := fmt.Sprintf("concurrent-plugin-%d", pluginIdx)
					ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
					err := suite.manager.ExecutePlugin(pluginName, ctx, []string{fmt.Sprintf("worker-%d-op-%d", workerID, op)})
					cancel()
					if err == nil {
						atomic.AddInt64(&successfulOperations, 1)
					}

				case 1: // Get plugin
					pluginIdx := rand.Intn(numPlugins)
					pluginName := fmt.Sprintf("concurrent-plugin-%d", pluginIdx)
					_, exists := suite.manager.GetPlugin(pluginName)
					if exists {
						atomic.AddInt64(&successfulOperations, 1)
					}

				case 2: // List plugins
					pluginList := suite.manager.ListPlugins()
					if len(pluginList) > 0 {
						atomic.AddInt64(&successfulOperations, 1)
					}

				case 3: // Get plugin info
					pluginIdx := rand.Intn(numPlugins)
					pluginName := fmt.Sprintf("concurrent-plugin-%d", pluginIdx)
					_, err := suite.manager.GetPluginInfo(pluginName)
					if err == nil {
						atomic.AddInt64(&successfulOperations, 1)
					}
				}
			}
		}(w)
	}

	wg.Wait()
	executionTime := time.Since(startTime)

	totalOps := atomic.LoadInt64(&totalOperations)
	successfulOps := atomic.LoadInt64(&successfulOperations)

	suite.T().Logf("Completed %d operations in %v (avg: %v per operation)",
		totalOps, executionTime, executionTime/time.Duration(totalOps))
	suite.T().Logf("Success rate: %.2f%% (%d/%d)",
		float64(successfulOps)/float64(totalOps)*100, successfulOps, totalOps)

	// Should have high success rate
	successRate := float64(successfulOps) / float64(totalOps)
	assert.Greater(suite.T(), successRate, 0.95, "Success rate should be high in stress conditions")
}

// TestMemoryStressTest tests memory usage under stress.
func (suite *PluginStressTestSuite) TestMemoryStressTest() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	// Force garbage collection and get initial memory stats
	runtime.GC()
	var initialStats, finalStats runtime.MemStats
	runtime.ReadMemStats(&initialStats)

	const numCycles = 10
	const pluginsPerCycle = 50

	for cycle := 0; cycle < numCycles; cycle++ {
		// Register plugins
		plugins := make([]*TestPlugin, pluginsPerCycle)
		for i := 0; i < pluginsPerCycle; i++ {
			plugins[i] = NewTestPlugin(fmt.Sprintf("memory-stress-%d-%d", cycle, i), "1.0.0")
			err := suite.manager.RegisterPlugin(plugins[i])
			require.NoError(suite.T(), err)
		}

		// Execute plugins
		for i := 0; i < pluginsPerCycle; i++ {
			pluginName := fmt.Sprintf("memory-stress-%d-%d", cycle, i)
			err := suite.manager.ExecutePlugin(pluginName, context.Background(), []string{"stress-test"})
			assert.NoError(suite.T(), err)
		}

		// Unload plugins
		for i := 0; i < pluginsPerCycle; i++ {
			pluginName := fmt.Sprintf("memory-stress-%d-%d", cycle, i)
			err := suite.manager.UnloadPlugin(pluginName)
			assert.NoError(suite.T(), err)
		}

		// Force garbage collection periodically
		if cycle%3 == 0 {
			runtime.GC()
		}
	}

	// Final memory measurement
	runtime.GC()
	runtime.ReadMemStats(&finalStats)

	memoryIncrease := finalStats.Alloc - initialStats.Alloc
	suite.T().Logf("Memory usage increased by %d bytes during stress test", memoryIncrease)

	// Memory increase should be reasonable (less than 50MB for this test)
	assert.Less(suite.T(), memoryIncrease, uint64(50*1024*1024), "Memory increase should be bounded")
}

// TestRaceConditionDetection tests for potential race conditions.
func (suite *PluginStressTestSuite) TestRaceConditionDetection() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	const numPlugins = 20
	const numWorkers = 50
	const operationsPerWorker = 100

	// Pre-register some plugins
	for i := 0; i < numPlugins/2; i++ {
		plugin := NewTestPlugin(fmt.Sprintf("race-plugin-%d", i), "1.0.0")
		err := suite.manager.RegisterPlugin(plugin)
		require.NoError(suite.T(), err)
	}

	var wg sync.WaitGroup
	var operations int64

	startTime := time.Now()

	// Launch workers that perform mixed operations
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for op := 0; op < operationsPerWorker; op++ {
				atomic.AddInt64(&operations, 1)

				switch rand.Intn(6) {
				case 0: // Register new plugin
					pluginName := fmt.Sprintf("race-dynamic-%d-%d", workerID, op)
					plugin := NewTestPlugin(pluginName, "1.0.0")
					suite.manager.RegisterPlugin(plugin)

				case 1: // Unload plugin (might fail if not exists)
					pluginName := fmt.Sprintf("race-dynamic-%d-%d", rand.Intn(numWorkers), rand.Intn(operationsPerWorker))
					suite.manager.UnloadPlugin(pluginName)

				case 2: // Execute existing plugin
					pluginIdx := rand.Intn(numPlugins / 2)
					pluginName := fmt.Sprintf("race-plugin-%d", pluginIdx)
					ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
					suite.manager.ExecutePlugin(pluginName, ctx, []string{"race-test"})
					cancel()

				case 3: // List plugins
					suite.manager.ListPlugins()

				case 4: // Get plugin info
					pluginIdx := rand.Intn(numPlugins / 2)
					pluginName := fmt.Sprintf("race-plugin-%d", pluginIdx)
					suite.manager.GetPluginInfo(pluginName)

				case 5: // Registry operations
					suite.registry.ListPlugins()
					suite.registry.GetStatistics()
				}
			}
		}(w)
	}

	wg.Wait()
	totalTime := time.Since(startTime)
	totalOps := atomic.LoadInt64(&operations)

	suite.T().Logf("Race condition test completed: %d operations in %v", totalOps, totalTime)

	// If we get here without deadlock or panic, the race condition test passed
	assert.True(suite.T(), true, "Race condition test completed successfully")
}

// TestDeadlockDetection tests for potential deadlocks.
func (suite *PluginStressTestSuite) TestDeadlockDetection() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	const numPlugins = 10
	const numWorkers = 30

	// Register plugins
	for i := 0; i < numPlugins; i++ {
		plugin := NewTestPlugin(fmt.Sprintf("deadlock-plugin-%d", i), "1.0.0")
		err := suite.manager.RegisterPlugin(plugin)
		require.NoError(suite.T(), err)
	}

	// Test timeout for deadlock detection
	testTimeout := 30 * time.Second
	done := make(chan bool, 1)

	go func() {
		var wg sync.WaitGroup

		// Launch workers that acquire locks in different orders
		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				// Perform operations that might cause lock contention
				for op := 0; op < 50; op++ {
					switch op % 4 {
					case 0:
						// Manager operations (acquires manager lock)
						pluginIdx := rand.Intn(numPlugins)
						pluginName := fmt.Sprintf("deadlock-plugin-%d", pluginIdx)
						suite.manager.ExecutePlugin(pluginName, context.Background(), []string{"deadlock-test"})

					case 1:
						// Registry operations (acquires registry lock)
						suite.registry.ListPlugins()
						suite.registry.GetStatistics()

					case 2:
						// Mixed operations
						pluginList := suite.manager.ListPlugins()
						if len(pluginList) > 0 {
							suite.manager.GetPluginInfo(pluginList[rand.Intn(len(pluginList))])
						}

					case 3:
						// Plugin-level operations
						pluginIdx := rand.Intn(numPlugins)
						pluginName := fmt.Sprintf("deadlock-plugin-%d", pluginIdx)
						if plugin, exists := suite.manager.GetPlugin(pluginName); exists {
							plugin.(*TestPlugin).GetExecuteCount()
							plugin.(*TestPlugin).IsInitialized()
						}
					}
				}
			}(w)
		}

		wg.Wait()
		done <- true
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		suite.T().Log("Deadlock test completed successfully")
		assert.True(suite.T(), true, "No deadlock detected")
	case <-time.After(testTimeout):
		suite.T().Fatal("Potential deadlock detected - test timed out")
	}
}

// TestResourceExhaustion tests behavior under resource exhaustion.
func (suite *PluginStressTestSuite) TestResourceExhaustion() {
	// Create manager with very low limits
	config := &ManagerConfig{
		PluginDirs:          []string{suite.pluginDir},
		LoadTimeout:         1 * time.Second, // Very short timeout
		EnableAutoDiscovery: true,
		MaxPlugins:          5, // Very low limit
		SecurityPolicy: SecurityPolicy{
			AllowUnsigned:   true,
			RequireChecksum: false,
		},
	}

	manager := NewPluginManager(config, suite.mockLogger)
	err := manager.Initialize()
	require.NoError(suite.T(), err)

	defer func() {
		if r := recover(); r != nil {
			suite.T().Fatalf("Plugin system panicked under resource exhaustion: %v", r)
		}
		manager.Shutdown()
	}()

	// Try to register more plugins than the limit allows
	const attemptedPlugins = 20
	var successfulRegistrations int
	var registrationErrors []error

	for i := 0; i < attemptedPlugins; i++ {
		plugin := NewTestPlugin(fmt.Sprintf("resource-plugin-%d", i), "1.0.0")
		err := manager.RegisterPlugin(plugin)
		if err != nil {
			registrationErrors = append(registrationErrors, err)
		} else {
			successfulRegistrations++
		}
	}

	suite.T().Logf("Successfully registered %d out of %d attempted plugins",
		successfulRegistrations, attemptedPlugins)
	suite.T().Logf("Registration errors: %d", len(registrationErrors))

	// Should have registered up to the limit
	assert.LessOrEqual(suite.T(), successfulRegistrations, config.MaxPlugins)

	// System should still be functional
	pluginList := manager.ListPlugins()
	assert.Len(suite.T(), pluginList, successfulRegistrations)

	// Should be able to execute registered plugins
	if len(pluginList) > 0 {
		err := manager.ExecutePlugin(pluginList[0], context.Background(), []string{"resource-test"})
		assert.NoError(suite.T(), err, "System should remain functional under resource limits")
	}
}

// TestLongRunningOperations tests system behavior with long-running operations.
func (suite *PluginStressTestSuite) TestLongRunningOperations() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	// Create a plugin that simulates long-running work
	slowPlugin := &SlowTestPlugin{
		TestPlugin:    *NewTestPlugin("slow-plugin", "1.0.0"),
		executionTime: 500 * time.Millisecond,
	}

	err = suite.manager.RegisterPlugin(slowPlugin)
	require.NoError(suite.T(), err)

	const numConcurrentExecutions = 10
	var wg sync.WaitGroup
	results := make(chan error, numConcurrentExecutions)

	startTime := time.Now()

	// Launch concurrent long-running operations
	for i := 0; i < numConcurrentExecutions; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := suite.manager.ExecutePlugin("slow-plugin", ctx, []string{fmt.Sprintf("long-op-%d", id)})
			results <- err
		}(i)
	}

	wg.Wait()
	close(results)
	totalTime := time.Since(startTime)

	// Collect results
	var successCount, errorCount int
	for err := range results {
		if err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	suite.T().Logf("Long-running operations completed in %v", totalTime)
	suite.T().Logf("Successful executions: %d, Failed executions: %d", successCount, errorCount)

	// All executions should succeed (no timeout)
	assert.Equal(suite.T(), numConcurrentExecutions, successCount)
	assert.Equal(suite.T(), 0, errorCount)

	// Total time should be reasonable for concurrent execution
	// (significantly less than sequential execution time)
	expectedSequentialTime := time.Duration(numConcurrentExecutions) * slowPlugin.executionTime
	assert.Less(suite.T(), totalTime, expectedSequentialTime/2, "Concurrent execution should be more efficient")
}

// SlowTestPlugin simulates a plugin with slow execution.
type SlowTestPlugin struct {
	TestPlugin
	executionTime time.Duration
}

// Execute simulates slow execution.
func (p *SlowTestPlugin) Execute(ctx context.Context, args []string) error {
	// Call parent Execute first
	if err := p.TestPlugin.Execute(ctx, args); err != nil {
		return err
	}

	// Add simulated work
	select {
	case <-time.After(p.executionTime):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TestSystemStability tests overall system stability under stress.
func (suite *PluginStressTestSuite) TestSystemStability() {
	err := suite.manager.Initialize()
	require.NoError(suite.T(), err)

	const testDuration = 10 * time.Second
	const numWorkers = 20

	var totalOperations int64
	var totalErrors int64
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	// Launch workers performing various operations
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			localOps := 0
			localErrors := 0

			for {
				select {
				case <-ctx.Done():
					atomic.AddInt64(&totalOperations, int64(localOps))
					atomic.AddInt64(&totalErrors, int64(localErrors))
					return
				default:
					localOps++

					// Random operation
					switch rand.Intn(7) {
					case 0: // Register plugin
						plugin := NewTestPlugin(fmt.Sprintf("stability-%d-%d", workerID, localOps), "1.0.0")
						if err := suite.manager.RegisterPlugin(plugin); err != nil {
							localErrors++
						}

					case 1: // Unload plugin
						plugins := suite.manager.ListPlugins()
						if len(plugins) > 0 {
							pluginToUnload := plugins[rand.Intn(len(plugins))]
							if err := suite.manager.UnloadPlugin(pluginToUnload); err != nil {
								localErrors++
							}
						}

					case 2: // Execute plugin
						plugins := suite.manager.ListPlugins()
						if len(plugins) > 0 {
							pluginToExecute := plugins[rand.Intn(len(plugins))]
							if err := suite.manager.ExecutePlugin(pluginToExecute, context.Background(), []string{"stability-test"}); err != nil {
								localErrors++
							}
						}

					case 3: // List plugins
						suite.manager.ListPlugins()

					case 4: // Get plugin info
						plugins := suite.manager.ListPlugins()
						if len(plugins) > 0 {
							pluginName := plugins[rand.Intn(len(plugins))]
							if _, err := suite.manager.GetPluginInfo(pluginName); err != nil {
								localErrors++
							}
						}

					case 5: // Enable/Disable plugin
						plugins := suite.manager.ListPlugins()
						if len(plugins) > 0 {
							pluginName := plugins[rand.Intn(len(plugins))]
							if rand.Intn(2) == 0 {
								suite.manager.DisablePlugin(pluginName)
							} else {
								suite.manager.EnablePlugin(pluginName)
							}
						}

					case 6: // Registry operations
						suite.registry.GetStatistics()
						suite.registry.ListPlugins()
					}
				}
			}
		}(w)
	}

	wg.Wait()

	finalOps := atomic.LoadInt64(&totalOperations)
	finalErrors := atomic.LoadInt64(&totalErrors)

	suite.T().Logf("Stability test completed: %d operations, %d errors in %v",
		finalOps, finalErrors, testDuration)

	errorRate := float64(finalErrors) / float64(finalOps)
	suite.T().Logf("Error rate: %.2f%%", errorRate*100)

	// System should remain stable with low error rate
	assert.Less(suite.T(), errorRate, 0.1, "Error rate should be low under stress")
	assert.Greater(suite.T(), finalOps, int64(1000), "Should complete reasonable number of operations")

	// System should still be functional
	plugins := suite.manager.ListPlugins()
	suite.T().Logf("Final plugin count: %d", len(plugins))

	// Should be able to perform basic operations
	if len(plugins) > 0 {
		_, err := suite.manager.GetPluginInfo(plugins[0])
		assert.NoError(suite.T(), err, "System should remain functional after stress test")
	}
}

// Run all tests in the suite.
func TestPluginStressTestSuite(t *testing.T) {
	suite.Run(t, new(PluginStressTestSuite))
}
