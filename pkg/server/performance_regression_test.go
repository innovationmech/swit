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

package server

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// PerformanceThresholds defines performance thresholds for regression testing
type PerformanceThresholds struct {
	MaxStartupTime      time.Duration
	MaxShutdownTime     time.Duration
	MaxMemoryUsage      uint64
	MaxGoroutineCount   int
	MaxConcurrentOpTime time.Duration
	MaxMemoryLeakSize   uint64
}

// DefaultPerformanceThresholds returns default performance thresholds
func DefaultPerformanceThresholds() PerformanceThresholds {
	return PerformanceThresholds{
		MaxStartupTime:      500 * time.Millisecond,
		MaxShutdownTime:     200 * time.Millisecond,
		MaxMemoryUsage:      50 * 1024 * 1024, // 50MB
		MaxGoroutineCount:   100,
		MaxConcurrentOpTime: 100 * time.Microsecond,
		MaxMemoryLeakSize:   50 * 1024 * 1024, // 50MB - increased threshold for test environment
	}
}

// PerformanceRegressionSuite runs comprehensive performance regression tests
type PerformanceRegressionSuite struct {
	thresholds PerformanceThresholds
	results    []PerformanceTestResult
	mu         sync.Mutex
}

// PerformanceTestResult holds the result of a performance test
type PerformanceTestResult struct {
	TestName    string
	Duration    time.Duration
	MemoryUsage uint64
	Success     bool
	Error       error
	Metrics     map[string]interface{}
}

// NewPerformanceRegressionSuite creates a new performance regression test suite
func NewPerformanceRegressionSuite() *PerformanceRegressionSuite {
	return &PerformanceRegressionSuite{
		thresholds: DefaultPerformanceThresholds(),
		results:    make([]PerformanceTestResult, 0),
	}
}

// SetThresholds sets custom performance thresholds
func (s *PerformanceRegressionSuite) SetThresholds(thresholds PerformanceThresholds) {
	s.thresholds = thresholds
}

// AddResult adds a test result to the suite
func (s *PerformanceRegressionSuite) AddResult(result PerformanceTestResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results = append(s.results, result)
}

// GetResults returns all test results
func (s *PerformanceRegressionSuite) GetResults() []PerformanceTestResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]PerformanceTestResult(nil), s.results...)
}

// GenerateReport generates a performance test report
func (s *PerformanceRegressionSuite) GenerateReport() string {
	results := s.GetResults()
	report := "Performance Regression Test Report\n"
	report += "==================================\n\n"

	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}

		status := "PASS"
		if !result.Success {
			status = "FAIL"
		}

		report += fmt.Sprintf("Test: %s - %s\n", result.TestName, status)
		report += fmt.Sprintf("  Duration: %v\n", result.Duration)
		report += fmt.Sprintf("  Memory: %d bytes\n", result.MemoryUsage)

		if result.Error != nil {
			report += fmt.Sprintf("  Error: %v\n", result.Error)
		}

		for key, value := range result.Metrics {
			report += fmt.Sprintf("  %s: %v\n", key, value)
		}
		report += "\n"
	}

	report += fmt.Sprintf("Summary: %d/%d tests passed\n", successCount, len(results))
	return report
}

// TestPerformanceRegressionSuite runs the complete performance regression test suite
func TestPerformanceRegressionSuite(t *testing.T) {
	gin.SetMode(gin.TestMode)

	suite := NewPerformanceRegressionSuite()

	// Run all performance regression tests
	t.Run("StartupPerformance", func(t *testing.T) {
		suite.testStartupPerformance(t)
	})

	t.Run("ShutdownPerformance", func(t *testing.T) {
		suite.testShutdownPerformance(t)
	})

	t.Run("MemoryUsage", func(t *testing.T) {
		suite.testMemoryUsage(t)
	})

	t.Run("ConcurrentOperations", func(t *testing.T) {
		suite.testConcurrentOperations(t)
	})

	t.Run("MemoryLeakDetection", func(t *testing.T) {
		suite.testMemoryLeakDetection(t)
	})

	t.Run("PerformanceMonitoringOverhead", func(t *testing.T) {
		suite.testPerformanceMonitoringOverhead(t)
	})

	// Generate and log report
	report := suite.GenerateReport()
	t.Log(report)

	// Verify all tests passed
	results := suite.GetResults()
	for _, result := range results {
		if !result.Success {
			t.Errorf("Performance regression test failed: %s - %v", result.TestName, result.Error)
		}
	}
}

// testStartupPerformance tests server startup performance
func (s *PerformanceRegressionSuite) testStartupPerformance(t *testing.T) {
	config := NewServerConfig()
	config.ServiceName = "startup-perf-test"
	config.Discovery.Enabled = false

	registrar := setupMockRegistrar()
	server, err := NewBusinessServerCore(config, registrar, nil)
	require.NoError(t, err)

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Start(ctx)
	duration := time.Since(start)

	var memUsage uint64
	if err == nil {
		metrics := server.GetPerformanceMetrics()
		memUsage = metrics.MemoryUsage
		server.Stop(ctx)
	}

	success := err == nil && duration <= s.thresholds.MaxStartupTime

	result := PerformanceTestResult{
		TestName:    "StartupPerformance",
		Duration:    duration,
		MemoryUsage: memUsage,
		Success:     success,
		Error:       err,
		Metrics: map[string]interface{}{
			"threshold": s.thresholds.MaxStartupTime,
		},
	}

	s.AddResult(result)

	if !success && err == nil {
		t.Errorf("Startup performance regression: %v > %v", duration, s.thresholds.MaxStartupTime)
	}
}

// testShutdownPerformance tests server shutdown performance
func (s *PerformanceRegressionSuite) testShutdownPerformance(t *testing.T) {
	config := NewServerConfig()
	config.ServiceName = "shutdown-perf-test"
	config.Discovery.Enabled = false

	registrar := setupMockRegistrar()
	server, err := NewBusinessServerCore(config, registrar, nil)
	require.NoError(t, err)

	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)

	start := time.Now()
	err = server.Stop(ctx)
	duration := time.Since(start)

	var memUsage uint64
	if err == nil {
		metrics := server.GetPerformanceMetrics()
		memUsage = metrics.MemoryUsage
	}

	success := err == nil && duration <= s.thresholds.MaxShutdownTime

	result := PerformanceTestResult{
		TestName:    "ShutdownPerformance",
		Duration:    duration,
		MemoryUsage: memUsage,
		Success:     success,
		Error:       err,
		Metrics: map[string]interface{}{
			"threshold": s.thresholds.MaxShutdownTime,
		},
	}

	s.AddResult(result)

	if !success && err == nil {
		t.Errorf("Shutdown performance regression: %v > %v", duration, s.thresholds.MaxShutdownTime)
	}
}

// testMemoryUsage tests server memory usage
func (s *PerformanceRegressionSuite) testMemoryUsage(t *testing.T) {
	config := NewServerConfig()
	config.ServiceName = "memory-usage-test"
	config.Discovery.Enabled = false

	registrar := setupMockRegistrar()
	server, err := NewBusinessServerCore(config, registrar, nil)
	require.NoError(t, err)

	ctx := context.Background()
	start := time.Now()

	err = server.Start(ctx)
	require.NoError(t, err)

	metrics := server.GetPerformanceMetrics()
	memUsage := metrics.MemoryUsage
	goroutineCount := metrics.GoroutineCount

	server.Stop(ctx)
	duration := time.Since(start)

	success := memUsage <= s.thresholds.MaxMemoryUsage && goroutineCount <= s.thresholds.MaxGoroutineCount

	var testError error
	if memUsage > s.thresholds.MaxMemoryUsage {
		testError = fmt.Errorf("memory usage %d exceeds threshold %d", memUsage, s.thresholds.MaxMemoryUsage)
	} else if goroutineCount > s.thresholds.MaxGoroutineCount {
		testError = fmt.Errorf("goroutine count %d exceeds threshold %d", goroutineCount, s.thresholds.MaxGoroutineCount)
	}

	result := PerformanceTestResult{
		TestName:    "MemoryUsage",
		Duration:    duration,
		MemoryUsage: memUsage,
		Success:     success,
		Error:       testError,
		Metrics: map[string]interface{}{
			"memory_threshold":    s.thresholds.MaxMemoryUsage,
			"goroutine_threshold": s.thresholds.MaxGoroutineCount,
			"goroutine_count":     goroutineCount,
		},
	}

	s.AddResult(result)

	if !success {
		t.Error(testError)
	}
}

// testConcurrentOperations tests concurrent operation performance
func (s *PerformanceRegressionSuite) testConcurrentOperations(t *testing.T) {
	config := NewServerConfig()
	config.ServiceName = "concurrent-ops-test"
	config.Discovery.Enabled = false

	registrar := setupMockRegistrar()
	server, err := NewBusinessServerCore(config, registrar, nil)
	require.NoError(t, err)

	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)
	defer server.Stop(ctx)

	const numGoroutines = 50
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	start := time.Now()

	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range operationsPerGoroutine {
				_ = server.GetHTTPAddress()
				_ = server.GetGRPCAddress()
				_ = server.GetTransports()
				_ = server.GetPerformanceMetrics()

				// Add variation to prevent optimization
				if (id+j)%10 == 0 {
					_ = server.GetTransportStatus()
				}
			}
		}(i)
	}

	wg.Wait()
	totalTime := time.Since(start)

	totalOperations := numGoroutines * operationsPerGoroutine
	avgTimePerOp := totalTime / time.Duration(totalOperations)

	metrics := server.GetPerformanceMetrics()
	success := avgTimePerOp <= s.thresholds.MaxConcurrentOpTime

	var testError error
	if !success {
		testError = fmt.Errorf("average operation time %v exceeds threshold %v", avgTimePerOp, s.thresholds.MaxConcurrentOpTime)
	}

	result := PerformanceTestResult{
		TestName:    "ConcurrentOperations",
		Duration:    totalTime,
		MemoryUsage: metrics.MemoryUsage,
		Success:     success,
		Error:       testError,
		Metrics: map[string]interface{}{
			"total_operations": totalOperations,
			"avg_time_per_op":  avgTimePerOp,
			"threshold":        s.thresholds.MaxConcurrentOpTime,
		},
	}

	s.AddResult(result)

	if !success {
		t.Error(testError)
	}
}

// testMemoryLeakDetection tests for memory leaks
func (s *PerformanceRegressionSuite) testMemoryLeakDetection(t *testing.T) {
	var initialMem, finalMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&initialMem)

	start := time.Now()

	// Create and destroy servers multiple times
	for i := range 5 {
		config := NewServerConfig()
		config.ServiceName = fmt.Sprintf("leak-test-%d", i)
		config.Discovery.Enabled = false

		registrar := setupMockRegistrar()
		server, err := NewBusinessServerCore(config, registrar, nil)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = server.Start(ctx)
		require.NoError(t, err)

		err = server.Stop(ctx)
		require.NoError(t, err)
		cancel()
	}

	runtime.GC()
	runtime.ReadMemStats(&finalMem)

	duration := time.Since(start)
	// Use signed arithmetic to handle cases where final memory is less than initial
	memoryIncrease := int64(finalMem.Alloc) - int64(initialMem.Alloc)
	// Only consider positive memory increases as potential leaks
	if memoryIncrease < 0 {
		memoryIncrease = 0
	}
	success := uint64(memoryIncrease) <= s.thresholds.MaxMemoryLeakSize

	var testError error
	if !success {
		testError = fmt.Errorf("potential memory leak: %d bytes > %d bytes", memoryIncrease, s.thresholds.MaxMemoryLeakSize)
	}

	result := PerformanceTestResult{
		TestName:    "MemoryLeakDetection",
		Duration:    duration,
		MemoryUsage: uint64(memoryIncrease),
		Success:     success,
		Error:       testError,
		Metrics: map[string]interface{}{
			"initial_memory":  initialMem.Alloc,
			"final_memory":    finalMem.Alloc,
			"memory_increase": memoryIncrease,
			"threshold":       s.thresholds.MaxMemoryLeakSize,
		},
	}

	s.AddResult(result)

	if !success {
		t.Error(testError)
	}
}

// testPerformanceMonitoringOverhead tests the overhead of performance monitoring
func (s *PerformanceRegressionSuite) testPerformanceMonitoringOverhead(t *testing.T) {
	monitor := NewPerformanceMonitor()

	const numOperations = 10000

	// Test with monitoring enabled
	start := time.Now()
	for i := range numOperations {
		monitor.RecordEvent(fmt.Sprintf("test_event_%d", i))
		monitor.GetMetrics().IncrementRequestCount()
	}
	enabledTime := time.Since(start)

	// Test with monitoring disabled
	monitor.Disable()
	start = time.Now()
	for i := range numOperations {
		monitor.RecordEvent(fmt.Sprintf("test_event_%d", i))
		monitor.GetMetrics().IncrementRequestCount()
	}
	disabledTime := time.Since(start)

	overhead := enabledTime - disabledTime
	overheadPerOp := overhead / numOperations

	// Overhead should be minimal (less than 1 microsecond per operation)
	maxOverheadPerOp := 1 * time.Microsecond
	success := overheadPerOp <= maxOverheadPerOp

	var testError error
	if !success {
		testError = fmt.Errorf("monitoring overhead too high: %v per operation > %v", overheadPerOp, maxOverheadPerOp)
	}

	result := PerformanceTestResult{
		TestName:    "PerformanceMonitoringOverhead",
		Duration:    enabledTime,
		MemoryUsage: 0, // Not applicable for this test
		Success:     success,
		Error:       testError,
		Metrics: map[string]interface{}{
			"enabled_time":        enabledTime,
			"disabled_time":       disabledTime,
			"overhead":            overhead,
			"overhead_per_op":     overheadPerOp,
			"max_overhead_per_op": maxOverheadPerOp,
		},
	}

	s.AddResult(result)

	if !success {
		t.Error(testError)
	}
}

// TestPerformanceRegressionWithCustomThresholds tests with custom thresholds
func TestPerformanceRegressionWithCustomThresholds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	suite := NewPerformanceRegressionSuite()

	// Set more strict thresholds for this test
	customThresholds := PerformanceThresholds{
		MaxStartupTime:      300 * time.Millisecond,
		MaxShutdownTime:     100 * time.Millisecond,
		MaxMemoryUsage:      30 * 1024 * 1024, // 30MB
		MaxGoroutineCount:   50,
		MaxConcurrentOpTime: 50 * time.Microsecond,
		MaxMemoryLeakSize:   5 * 1024 * 1024, // 5MB
	}

	suite.SetThresholds(customThresholds)

	// Run a subset of tests with strict thresholds
	t.Run("StrictStartupPerformance", func(t *testing.T) {
		suite.testStartupPerformance(t)
	})

	// Generate report
	report := suite.GenerateReport()
	t.Log("Custom Thresholds Report:")
	t.Log(report)
}
