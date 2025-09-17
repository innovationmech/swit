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
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// BenchmarkServerStartup benchmarks the server startup performance
func BenchmarkServerStartup(b *testing.B) {
	gin.SetMode(gin.TestMode)

	b.ResetTimer()
	for i := range b.N {
		config := NewServerConfig()
		config.ServiceName = fmt.Sprintf("benchmark-service-%d", i)
		config.HTTP.Port = "0" // Use random available port
		config.HTTP.Enabled = true
		config.HTTP.EnableReady = false
		config.GRPC.Port = "0" // Use random available port
		config.GRPC.Enabled = true
		config.GRPC.EnableKeepalive = false
		config.GRPC.EnableReflection = false
		config.GRPC.EnableHealthService = false
		config.ShutdownTimeout = 5 * time.Second
		config.Discovery.Enabled = false // Disable discovery for benchmarks

		registrar := setupMockRegistrar()
		server, err := NewBusinessServerCore(config, registrar, nil)
		if err != nil {
			b.Fatalf("Failed to create server: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := server.Start(ctx); err != nil {
			cancel()
			b.Fatalf("Failed to start server: %v", err)
		}

		if err := server.Stop(ctx); err != nil {
			cancel()
			b.Fatalf("Failed to stop server: %v", err)
		}
		cancel()
	}
}

// BenchmarkServerShutdown benchmarks the server shutdown performance
func BenchmarkServerShutdown(b *testing.B) {
	gin.SetMode(gin.TestMode)

	servers := make([]*BusinessServerImpl, b.N)

	// Pre-create and start servers
	for i := range b.N {
		config := &ServerConfig{
			ServiceName: fmt.Sprintf("benchmark-service-%d", i),
			HTTP: HTTPConfig{
				Port:        "0",
				Enabled:     true,
				EnableReady: false,
			},
			GRPC: GRPCConfig{
				Port:                "0",
				Enabled:             true,
				EnableKeepalive:     false,
				EnableReflection:    false,
				EnableHealthService: false,
			},
			ShutdownTimeout: 5 * time.Second,
			Discovery: DiscoveryConfig{
				Enabled: false,
			},
		}

		registrar := setupMockRegistrar()
		server, err := NewBusinessServerCore(config, registrar, nil)
		if err != nil {
			b.Fatalf("Failed to create server: %v", err)
		}

		ctx := context.Background()
		if err := server.Start(ctx); err != nil {
			b.Fatalf("Failed to start server: %v", err)
		}
		servers[i] = server
	}

	b.ResetTimer()
	for i := range b.N {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := servers[i].Stop(ctx); err != nil {
			cancel()
			b.Fatalf("Failed to stop server: %v", err)
		}
		cancel()
	}
}

// BenchmarkServiceRegistration benchmarks service registration performance
func BenchmarkServiceRegistration(b *testing.B) {
	gin.SetMode(gin.TestMode)

	config := &ServerConfig{
		ServiceName: "benchmark-service",
		HTTP: HTTPConfig{
			Port:        "0",
			Enabled:     true,
			EnableReady: false,
		},
		GRPC: GRPCConfig{
			Port:                "0",
			Enabled:             true,
			EnableKeepalive:     false,
			EnableReflection:    false,
			EnableHealthService: false,
		},
		ShutdownTimeout: 5 * time.Second,
		Discovery: DiscoveryConfig{
			Enabled: false,
		},
	}

	b.ResetTimer()
	for range b.N {
		registrar := &benchmarkServiceRegistrar{serviceCount: 10}
		server, err := NewBusinessServerCore(config, registrar, nil)
		if err != nil {
			b.Fatalf("Failed to create server: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := server.Start(ctx); err != nil {
			cancel()
			b.Fatalf("Failed to start server: %v", err)
		}

		if err := server.Stop(ctx); err != nil {
			cancel()
			b.Fatalf("Failed to stop server: %v", err)
		}
		cancel()
	}
}

// BenchmarkConcurrentServerOperations benchmarks concurrent server operations
func BenchmarkConcurrentServerOperations(b *testing.B) {
	gin.SetMode(gin.TestMode)

	config := &ServerConfig{
		ServiceName: "benchmark-service",
		HTTP: HTTPConfig{
			Port:        "0",
			Enabled:     true,
			EnableReady: false,
		},
		GRPC: GRPCConfig{
			Port:                "0",
			Enabled:             true,
			EnableKeepalive:     false,
			EnableReflection:    false,
			EnableHealthService: false,
		},
		ShutdownTimeout: 5 * time.Second,
		Discovery: DiscoveryConfig{
			Enabled: false,
		},
	}

	registrar := setupMockRegistrar()
	server, err := NewBusinessServerCore(config, registrar, nil)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}

	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		b.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(ctx)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate concurrent operations
			_ = server.GetHTTPAddress()
			_ = server.GetGRPCAddress()
			_ = server.GetTransports()
			_ = server.GetTransportStatus()
		}
	})
}

// BenchmarkMemoryUsage benchmarks memory usage during server operations
func BenchmarkMemoryUsage(b *testing.B) {
	gin.SetMode(gin.TestMode)

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	for i := range b.N {
		config := NewServerConfig()
		config.ServiceName = fmt.Sprintf("benchmark-service-%d", i)
		config.HTTP.Port = "0" // Use random available port
		config.HTTP.Enabled = true
		config.HTTP.EnableReady = false
		config.GRPC.Port = "0" // Use random available port
		config.GRPC.Enabled = true
		config.GRPC.EnableKeepalive = false
		config.GRPC.EnableReflection = false
		config.GRPC.EnableHealthService = false
		config.ShutdownTimeout = 5 * time.Second
		config.Discovery.Enabled = false

		registrar := setupMockRegistrar()
		server, err := NewBusinessServerCore(config, registrar, nil)
		if err != nil {
			b.Fatalf("Failed to create server: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := server.Start(ctx); err != nil {
			cancel()
			b.Fatalf("Failed to start server: %v", err)
		}

		if err := server.Stop(ctx); err != nil {
			cancel()
			b.Fatalf("Failed to stop server: %v", err)
		}
		cancel()
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	b.ReportMetric(float64(m2.Mallocs-m1.Mallocs)/float64(b.N), "allocs/op")
}

// benchmarkServiceRegistrar is a service registrar that registers multiple services for benchmarking
type benchmarkServiceRegistrar struct {
	serviceCount int
}

func (r *benchmarkServiceRegistrar) RegisterServices(registry BusinessServiceRegistry) error {
	// Register multiple HTTP handlers
	for i := range r.serviceCount {
		handler := &mockHTTPHandler{}
		handler.On("GetServiceName").Return(fmt.Sprintf("http-service-%d", i))
		handler.On("RegisterRoutes", mock.Anything).Return(nil)
		if err := registry.RegisterBusinessHTTPHandler(handler); err != nil {
			return err
		}
	}

	// Register multiple gRPC services
	for i := range r.serviceCount {
		service := &mockGRPCService{}
		service.On("GetServiceName").Return(fmt.Sprintf("grpc-service-%d", i))
		service.On("RegisterGRPC", mock.Anything).Return(nil)
		if err := registry.RegisterBusinessGRPCService(service); err != nil {
			return err
		}
	}

	return nil
}

// setupMockRegistrar creates a properly configured mock service registrar
func setupMockRegistrar() *mockBusinessServiceRegistrar {
	registrar := &mockBusinessServiceRegistrar{}
	registrar.On("RegisterServices", mock.Anything).Return(nil)
	return registrar
}

// Additional benchmark tests for comprehensive performance monitoring

// BenchmarkPerformanceMonitorOverhead benchmarks the overhead of performance monitoring
func BenchmarkPerformanceMonitorOverhead(b *testing.B) {
	monitor := NewPerformanceMonitor()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			monitor.RecordEvent("test_event")
			monitor.GetMetrics().IncrementRequestCount()
			monitor.GetMetrics().RecordMemoryUsage()
		}
	})
}

// BenchmarkMetricsCollection benchmarks metrics collection performance
func BenchmarkMetricsCollection(b *testing.B) {
	metrics := NewPerformanceMetrics()

	b.ResetTimer()
	for i := range b.N {
		metrics.RecordStartupTime(time.Duration(i) * time.Millisecond)
		metrics.RecordShutdownTime(time.Duration(i) * time.Millisecond)
		metrics.RecordMemoryUsage()
		metrics.RecordServiceMetrics(i, i+1)
		metrics.IncrementRequestCount()
		metrics.IncrementErrorCount()
		_ = metrics.GetSnapshot()
	}
}

// BenchmarkConcurrentMetricsAccess benchmarks concurrent access to metrics
func BenchmarkConcurrentMetricsAccess(b *testing.B) {
	metrics := NewPerformanceMetrics()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Mix of read and write operations
			switch b.N % 4 {
			case 0:
				metrics.IncrementRequestCount()
			case 1:
				metrics.IncrementErrorCount()
			case 2:
				metrics.RecordMemoryUsage()
			case 3:
				_ = metrics.GetSnapshot()
			}
		}
	})
}

// Performance regression tests

// TestPerformanceRegression runs performance regression tests to ensure server performance doesn't degrade
func TestPerformanceRegression(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Define performance thresholds
	maxStartupTime := 500 * time.Millisecond
	maxShutdownTime := 200 * time.Millisecond
	maxMemoryPerServer := uint64(50 * 1024 * 1024) // 50MB

	config := NewServerConfig()
	config.ServiceName = "regression-test-service"
	config.Discovery.Enabled = false

	registrar := setupMockRegistrar()
	server, err := NewBusinessServerCore(config, registrar, nil)
	require.NoError(t, err, "Failed to create server")

	// Test startup performance
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Start(ctx)
	require.NoError(t, err, "Failed to start server")

	startupTime := time.Since(startTime)
	assert.LessOrEqual(t, startupTime, maxStartupTime,
		"Server startup took too long: %v > %v", startupTime, maxStartupTime)

	// Test memory usage
	metrics := server.GetPerformanceMetrics()
	assert.LessOrEqual(t, metrics.MemoryUsage, maxMemoryPerServer,
		"Server memory usage too high: %d bytes > %d bytes", metrics.MemoryUsage, maxMemoryPerServer)

	// Test shutdown performance
	shutdownStart := time.Now()
	err = server.Stop(ctx)
	require.NoError(t, err, "Failed to stop server")

	shutdownTime := time.Since(shutdownStart)
	assert.LessOrEqual(t, shutdownTime, maxShutdownTime,
		"Server shutdown took too long: %v > %v", shutdownTime, maxShutdownTime)

	t.Logf("Performance metrics - Startup: %v, Shutdown: %v, Memory: %d bytes, Goroutines: %d",
		startupTime, shutdownTime, metrics.MemoryUsage, metrics.GoroutineCount)
}

// TestConcurrentPerformance tests performance under concurrent load
func TestConcurrentPerformance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewServerConfig()
	config.ServiceName = "concurrent-test-service"
	config.Discovery.Enabled = false

	registrar := setupMockRegistrar()
	server, err := NewBusinessServerCore(config, registrar, nil)
	require.NoError(t, err, "Failed to create server")

	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err, "Failed to start server")
	defer server.Stop(ctx)

	// Run concurrent operations
	const numGoroutines = 100
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	startTime := time.Now()

	for i := range numGoroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := range operationsPerGoroutine {
				_ = server.GetHTTPAddress()
				_ = server.GetGRPCAddress()
				_ = server.GetTransports()
				_ = server.GetPerformanceMetrics()

				// Add some variation to prevent optimization
				if (goroutineID+j)%10 == 0 {
					_ = server.GetTransportStatus()
				}
			}
		}(i)
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	totalOperations := numGoroutines * operationsPerGoroutine
	avgTimePerOp := totalTime / time.Duration(totalOperations)

	t.Logf("Concurrent performance - Total ops: %d, Total time: %v, Avg time per op: %v",
		totalOperations, totalTime, avgTimePerOp)

	// Performance threshold: should handle operations quickly
	maxAvgTimePerOp := 100 * time.Microsecond
	assert.LessOrEqual(t, avgTimePerOp, maxAvgTimePerOp,
		"Average operation time too slow: %v > %v", avgTimePerOp, maxAvgTimePerOp)
}

// TestMemoryLeakDetection tests for memory leaks during server lifecycle
func TestMemoryLeakDetection(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping memory leak detection in -short mode")
    }
	gin.SetMode(gin.TestMode)

	var initialMem, finalMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&initialMem)

	// Create and destroy servers multiple times
	for i := range 10 {
		config := NewServerConfig()
		config.ServiceName = fmt.Sprintf("leak-test-service-%d", i)
		config.Discovery.Enabled = false

		registrar := setupMockRegistrar()
		server, err := NewBusinessServerCore(config, registrar, nil)
		require.NoError(t, err, "Failed to create server")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = server.Start(ctx)
		require.NoError(t, err, "Failed to start server")

		err = server.Stop(ctx)
		require.NoError(t, err, "Failed to stop server")
		cancel()
	}

	runtime.GC()
	runtime.ReadMemStats(&finalMem)

	memoryIncrease := finalMem.Alloc - initialMem.Alloc
	maxMemoryIncrease := uint64(10 * 1024 * 1024) // 10MB threshold

	assert.LessOrEqual(t, memoryIncrease, maxMemoryIncrease,
		"Potential memory leak detected: memory increased by %d bytes > %d bytes",
		memoryIncrease, maxMemoryIncrease)

	t.Logf("Memory leak test - Initial: %d bytes, Final: %d bytes, Increase: %d bytes",
		initialMem.Alloc, finalMem.Alloc, memoryIncrease)
}

// TestPerformanceMonitoringHooks tests the performance monitoring capabilities
func TestPerformanceMonitoringHooks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewServerConfig()
	config.ServiceName = "monitoring-test-service"
	config.Discovery.Enabled = false

	registrar := setupMockRegistrar()
	server, err := NewBusinessServerCore(config, registrar, nil)
	require.NoError(t, err, "Failed to create server")

	// Test performance profiler
	profiler := NewPerformanceProfiler()

	startupTime, err := profiler.ProfileServerStartup(server)
	require.NoError(t, err, "Failed to profile server startup")
	assert.Greater(t, startupTime, time.Duration(0), "Startup time should be positive")

	shutdownTime, err := profiler.ProfileServerShutdown(server)
	require.NoError(t, err, "Failed to profile server shutdown")
	assert.Greater(t, shutdownTime, time.Duration(0), "Shutdown time should be positive")

	metrics := profiler.GetMetrics()
	assert.Equal(t, startupTime, metrics.StartupTime,
		"Metrics startup time mismatch: got %v, want %v", metrics.StartupTime, startupTime)
	assert.Equal(t, shutdownTime, metrics.ShutdownTime,
		"Metrics shutdown time mismatch: got %v, want %v", metrics.ShutdownTime, shutdownTime)

	t.Logf("Performance monitoring - %s", metrics.String())
}

// TestPerformanceMonitorHooks tests performance monitoring hooks functionality
func TestPerformanceMonitorHooks(t *testing.T) {
	monitor := NewPerformanceMonitor()

	// Test hook registration and execution
	var hookCalled int32
	var eventReceived string
	var mu sync.Mutex

	hook := func(event string, metrics *PerformanceMetrics) {
		atomic.StoreInt32(&hookCalled, 1)
		mu.Lock()
		eventReceived = event
		mu.Unlock()
	}

	monitor.AddHook(hook)
	monitor.RecordEvent("test_event")

	// Give hook time to execute (it runs in goroutine)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&hookCalled), "Hook should have been called")
	mu.Lock()
	receivedEvent := eventReceived
	mu.Unlock()
	assert.Equal(t, "test_event", receivedEvent, "Event should match")

	// Test hook removal
	monitor.RemoveAllHooks()
	atomic.StoreInt32(&hookCalled, 0)
	monitor.RecordEvent("another_event")
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, int32(0), atomic.LoadInt32(&hookCalled), "Hook should not be called after removal")
}

// TestPerformanceThresholds tests performance threshold checking
func TestPerformanceThresholds(t *testing.T) {
	metrics := NewPerformanceMetrics()

	// Set some values that exceed thresholds
	metrics.RecordStartupTime(1 * time.Second)         // Exceeds 500ms threshold
	metrics.RecordShutdownTime(300 * time.Millisecond) // Exceeds 200ms threshold

	violations := metrics.CheckThresholds()
	assert.Len(t, violations, 2, "Should have 2 threshold violations")

	// Test with values within thresholds
	metrics.RecordStartupTime(100 * time.Millisecond)
	metrics.RecordShutdownTime(50 * time.Millisecond)

	violations = metrics.CheckThresholds()
	assert.Len(t, violations, 0, "Should have no threshold violations")
}

// TestPerformanceMetricsSnapshot tests thread-safe metrics snapshots
func TestPerformanceMetricsSnapshot(t *testing.T) {
	metrics := NewPerformanceMetrics()

	// Record some metrics
	metrics.RecordStartupTime(100 * time.Millisecond)
	metrics.RecordShutdownTime(50 * time.Millisecond)
	metrics.IncrementRequestCount()
	metrics.IncrementErrorCount()

	snapshot := metrics.GetSnapshot()

	assert.Equal(t, 100*time.Millisecond, snapshot.StartupTime)
	assert.Equal(t, 50*time.Millisecond, snapshot.ShutdownTime)
	assert.Equal(t, int64(1), snapshot.RequestCount)
	assert.Equal(t, int64(1), snapshot.ErrorCount)
}

// TestPerformanceProfileOperation tests operation profiling
func TestPerformanceProfileOperation(t *testing.T) {
	profiler := NewPerformanceProfiler()
	monitor := profiler.GetMonitor()

	// Test successful operation
	err := monitor.ProfileOperation("test_operation", func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	assert.NoError(t, err, "Operation should succeed")

	// Test failed operation
	err = monitor.ProfileOperation("failing_operation", func() error {
		return fmt.Errorf("test error")
	})

	assert.Error(t, err, "Operation should fail")

	metrics := monitor.GetMetrics()
	assert.Equal(t, int64(1), metrics.GetSnapshot().ErrorCount, "Error count should be incremented")
}

// TestPerformancePeriodicCollection tests periodic metrics collection
func TestPerformancePeriodicCollection(t *testing.T) {
	monitor := NewPerformanceMonitor()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start periodic collection with short interval
	monitor.StartPeriodicCollection(ctx, 20*time.Millisecond)

	// Wait for collection to run
	<-ctx.Done()

	// Verify metrics were collected using thread-safe snapshot
	metrics := monitor.GetMetrics()
	snapshot := metrics.GetSnapshot()
	assert.Greater(t, snapshot.MemoryUsage, uint64(0), "Memory usage should be recorded")
}

// TestPerformanceMonitorEnableDisable tests enable/disable functionality
func TestPerformanceMonitorEnableDisable(t *testing.T) {
	monitor := NewPerformanceMonitor()

	assert.True(t, monitor.IsEnabled(), "Monitor should be enabled by default")

	monitor.Disable()
	assert.False(t, monitor.IsEnabled(), "Monitor should be disabled")

	monitor.Enable()
	assert.True(t, monitor.IsEnabled(), "Monitor should be enabled again")
}
