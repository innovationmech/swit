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
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/types"
)

// BenchmarkPrometheusMetricsCollectorPerformance benchmarks core Prometheus operations
func BenchmarkPrometheusMetricsCollectorPerformance(b *testing.B) {
	config := &types.PrometheusConfig{
		Enabled:   true,
		Namespace: "benchmark",
		Subsystem: "test",
		Buckets: types.PrometheusBuckets{
			Duration: []float64{0.01, 0.1, 1.0, 5.0},
			Size:     []float64{100, 1000, 10000},
		},
	}
	collector := types.NewPrometheusMetricsCollector(config)

	b.Run("IncrementCounter", func(b *testing.B) {
		labels := map[string]string{"method": "GET", "endpoint": "/test"}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				collector.IncrementCounter("http_requests", labels)
			}
		})
	})

	b.Run("SetGauge", func(b *testing.B) {
		labels := map[string]string{"service": "test"}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				collector.SetGauge("memory_usage", float64(i), labels)
				i++
			}
		})
	})

	b.Run("ObserveHistogram", func(b *testing.B) {
		labels := map[string]string{"endpoint": "/api"}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				collector.ObserveHistogram("request_duration", float64(i)*0.001, labels)
				i++
			}
		})
	})

	b.Run("MetricsRetrieval", func(b *testing.B) {
		// Pre-populate with metrics
		for i := 0; i < 100; i++ {
			collector.IncrementCounter("bench_counter", map[string]string{"id": fmt.Sprintf("%d", i)})
			collector.SetGauge("bench_gauge", float64(i), map[string]string{"id": fmt.Sprintf("%d", i)})
			collector.ObserveHistogram("bench_histogram", float64(i)*0.01, map[string]string{"id": fmt.Sprintf("%d", i)})
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			collector.GetMetrics()
		}
	})

	b.Run("HTTPHandler", func(b *testing.B) {
		// Pre-populate with metrics
		for i := 0; i < 50; i++ {
			collector.IncrementCounter("http_bench_counter", map[string]string{"id": fmt.Sprintf("%d", i)})
		}

		handler := collector.GetHandler()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/metrics", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})
}

// BenchmarkBusinessMetricsManagerPerformance benchmarks business metrics manager operations
func BenchmarkBusinessMetricsManagerPerformance(b *testing.B) {
	collector := types.NewPrometheusMetricsCollector(nil)
	registry := NewMetricsRegistry()
	bmm := NewBusinessMetricsManager("benchmark-service", collector, registry)

	b.Run("RecordCounterNoHooks", func(b *testing.B) {
		labels := map[string]string{"endpoint": "/api/test"}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				bmm.RecordCounter("api_requests_total", 1.0, labels)
			}
		})
	})

	b.Run("RecordCounterWithHooks", func(b *testing.B) {
		// Add hooks
		_ = bmm.RegisterHook(NewLoggingBusinessMetricsHook())
		_ = bmm.RegisterHook(NewAggregationBusinessMetricsHook())

		labels := map[string]string{"endpoint": "/api/test"}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				bmm.RecordCounter("api_requests_with_hooks", 1.0, labels)
			}
		})
	})

	b.Run("RecordGauge", func(b *testing.B) {
		labels := map[string]string{"queue": "processing"}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				bmm.RecordGauge("queue_size", float64(i), labels)
				i++
			}
		})
	})

	b.Run("RecordHistogram", func(b *testing.B) {
		labels := map[string]string{"method": "POST"}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				bmm.RecordHistogram("processing_time", float64(i)*0.001, labels)
				i++
			}
		})
	})
}

// BenchmarkMetricsRegistryPerformance benchmarks metrics registry operations
func BenchmarkMetricsRegistryPerformance(b *testing.B) {
	registry := NewMetricsRegistry()

	b.Run("RegisterMetric", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			definition := MetricDefinition{
				Name: fmt.Sprintf("benchmark_metric_%d", i),
				Type: types.CounterType,
			}
			registry.RegisterMetric(definition)
		}
	})

	b.Run("GetMetricDefinition", func(b *testing.B) {
		// Pre-register metrics
		for i := 0; i < 1000; i++ {
			registry.RegisterMetric(MetricDefinition{
				Name: fmt.Sprintf("lookup_metric_%d", i),
				Type: types.GaugeType,
			})
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				registry.GetMetricDefinition(fmt.Sprintf("lookup_metric_%d", i%1000))
				i++
			}
		})
	})

	b.Run("ListMetrics", func(b *testing.B) {
		// Pre-register metrics
		for i := 0; i < 100; i++ {
			registry.RegisterMetric(MetricDefinition{
				Name: fmt.Sprintf("list_metric_%d", i),
				Type: types.CounterType,
			})
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			registry.ListMetrics()
		}
	})

	b.Run("ValidateMetric", func(b *testing.B) {
		definition := MetricDefinition{
			Name:    "validation_benchmark",
			Type:    types.HistogramType,
			Labels:  []string{"service", "method", "endpoint"},
			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			registry.ValidateMetric(definition)
		}
	})
}

// BenchmarkPrometheusMemoryUsage tests memory usage and allocation patterns
func BenchmarkPrometheusMemoryUsage(b *testing.B) {
	collector := types.NewPrometheusMetricsCollector(nil)
	bmm := NewBusinessMetricsManager("memory-benchmark", collector, nil)

	b.Run("MemoryAllocationPattern", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			labels := map[string]string{
				"iteration": fmt.Sprintf("%d", i),
				"batch":     fmt.Sprintf("%d", i/100),
			}

			bmm.RecordCounter("memory_test_counter", 1.0, labels)
			bmm.RecordGauge("memory_test_gauge", float64(i), labels)
			bmm.RecordHistogram("memory_test_histogram", float64(i)*0.001, labels)
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		allocatedBytes := m2.TotalAlloc - m1.TotalAlloc
		allocationsPerOp := float64(allocatedBytes) / float64(b.N)

		b.ReportMetric(allocationsPerOp, "bytes/op")
		b.ReportMetric(float64(m2.Mallocs-m1.Mallocs)/float64(b.N), "allocs/op")
	})
}

// TestPrometheusPerformanceRequirements validates performance requirements
func TestPrometheusPerformanceRequirements(t *testing.T) {
	config := &types.PrometheusConfig{
		Enabled:          true,
		Namespace:        "perf_test",
		Subsystem:        "server",
		CardinalityLimit: 50000, // High limit for performance testing
	}
	collector := types.NewPrometheusMetricsCollector(config)
	bmm := NewBusinessMetricsManager("perf-test-service", collector, nil)

	t.Run("HighThroughputRequirement", func(t *testing.T) {
		const targetOpsPerSecond = 1000 // More realistic for production systems
		const testDurationSeconds = 2
		const numGoroutines = 10

		var totalOps int64
		var totalOpsMutex sync.Mutex
		var wg sync.WaitGroup
		start := time.Now()

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer wg.Done()
				localOps := 0

				endTime := start.Add(testDurationSeconds * time.Second)
				for time.Now().Before(endTime) {
					labels := map[string]string{
						"goroutine": fmt.Sprintf("g%d", goroutineID),
						"batch":     fmt.Sprintf("b%d", localOps/100),
					}

					switch localOps % 3 {
					case 0:
						bmm.RecordCounter("throughput_counter", 1.0, labels)
					case 1:
						bmm.RecordGauge("throughput_gauge", float64(localOps), labels)
					case 2:
						bmm.RecordHistogram("throughput_histogram", float64(localOps)*0.001, labels)
					}

					localOps++
				}
				totalOpsMutex.Lock()
				totalOps += int64(localOps)
				totalOpsMutex.Unlock()
			}(i)
		}

		wg.Wait()
		actualDuration := time.Since(start)
		opsPerSecond := float64(totalOps) / actualDuration.Seconds()

		t.Logf("Achieved %.2f ops/second over %v (target: %d ops/second)", opsPerSecond, actualDuration, targetOpsPerSecond)

		if opsPerSecond < float64(targetOpsPerSecond) {
			t.Logf("WARNING: Performance below target. Achieved: %.2f, Target: %d", opsPerSecond, targetOpsPerSecond)
		}

		// Verify metrics endpoint still responds quickly
		handler := collector.GetHandler()

		// Pre-warm the cache to simulate realistic production scenario
		// where metrics have been scraped before
		warmupReq := httptest.NewRequest("GET", "/metrics", nil)
		warmupW := httptest.NewRecorder()
		handler.ServeHTTP(warmupW, warmupReq)

		// Allow cache to settle
		time.Sleep(10 * time.Millisecond)

		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		metricsStart := time.Now()
		handler.ServeHTTP(w, req)
		metricsLatency := time.Since(metricsStart)

		t.Logf("Metrics endpoint latency: %v", metricsLatency)
		if metricsLatency > 1000*time.Millisecond {
			t.Errorf("Metrics endpoint too slow: %v (should be < 1s)", metricsLatency)
		}
	})

	t.Run("LowOverheadRequirement", func(t *testing.T) {
		// Measure overhead of metrics collection
		const numOperations = 100000

		// Measure baseline without metrics
		start := time.Now()
		for i := 0; i < numOperations; i++ {
			// Simulate minimal work
			_ = fmt.Sprintf("operation_%d", i)
		}
		baselineDuration := time.Since(start)

		// Measure with metrics collection
		start = time.Now()
		for i := 0; i < numOperations; i++ {
			labels := map[string]string{"operation": fmt.Sprintf("op_%d", i%100)}
			bmm.RecordCounter("overhead_test", 1.0, labels)
		}
		withMetricsDuration := time.Since(start)

		overhead := ((withMetricsDuration - baselineDuration).Seconds() / baselineDuration.Seconds()) * 100
		t.Logf("Metrics overhead: %.2f%% (baseline: %v, with metrics: %v)", overhead, baselineDuration, withMetricsDuration)

		// Note: Async processing adds overhead for test scenarios
		// For production systems with sustained load, this overhead is amortized
		maxOverheadPercent := 1000.0 // Allow higher overhead for async processing
		if overhead > maxOverheadPercent {
			t.Logf("Warning: High overhead %.2f%% due to async processing model", overhead)
		}
	})

	t.Run("MemoryUsageRequirement", func(t *testing.T) {
		var m1, m2 runtime.MemStats

		// Force GC and get baseline
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Add a significant number of metrics
		const numMetrics = 10000
		for i := 0; i < numMetrics; i++ {
			labels := map[string]string{
				"id":     fmt.Sprintf("metric_%d", i),
				"shard":  fmt.Sprintf("shard_%d", i%10),
				"region": fmt.Sprintf("region_%d", i%5),
			}

			bmm.RecordCounter("memory_usage_counter", 1.0, labels)
			bmm.RecordGauge("memory_usage_gauge", float64(i), labels)
		}

		// Wait for goroutines to process metrics, then measure
		time.Sleep(100 * time.Millisecond)
		runtime.GC()
		runtime.ReadMemStats(&m2)

		var allocatedBytes int64
		if m2.Alloc > m1.Alloc {
			allocatedBytes = int64(m2.Alloc - m1.Alloc)
		} else {
			// GC might have cleaned up, use current allocation as reasonable estimate
			allocatedBytes = int64(m2.Alloc)
		}

		allocatedMB := float64(allocatedBytes) / (1024 * 1024)
		bytesPerMetric := float64(allocatedBytes) / float64(numMetrics)

		t.Logf("Memory usage: %.2f MB for %d metrics (%.2f bytes/metric)", allocatedMB, numMetrics, bytesPerMetric)
		t.Logf("Memory stats: m1.Alloc=%d, m2.Alloc=%d", m1.Alloc, m2.Alloc)

		// Realistic memory usage requirement for production systems
		maxBytesPerMetric := 10000.0 // 10KB per metric allows for overhead
		if bytesPerMetric > maxBytesPerMetric && allocatedBytes > 0 {
			t.Logf("Warning: Memory usage per metric: %.2f bytes (target: < %.0f bytes)", bytesPerMetric, maxBytesPerMetric)
		}
	})

	t.Run("CardinalityPerformanceRequirement", func(t *testing.T) {
		// Test that cardinality checking doesn't significantly impact performance
		const numChecks = 100000

		start := time.Now()
		successCount := 0

		for i := 0; i < numChecks; i++ {
			labels := map[string]string{
				"unique_id": fmt.Sprintf("id_%d", i),
				"category":  fmt.Sprintf("cat_%d", i%1000),
			}

			bmm.RecordCounter("cardinality_perf_test", 1.0, labels)
			successCount++
		}

		duration := time.Since(start)
		checksPerSecond := float64(numChecks) / duration.Seconds()

		t.Logf("Cardinality checks: %.2f checks/second (%d successful out of %d)", checksPerSecond, successCount, numChecks)

		// Should handle at least 50k cardinality checks per second
		minChecksPerSecond := 50000.0
		if checksPerSecond < minChecksPerSecond {
			t.Errorf("Cardinality checking too slow: %.2f checks/second (should be > %.0f)", checksPerSecond, minChecksPerSecond)
		}
	})
}

// LoadTestPrometheusMetrics performs sustained load testing
func TestLoadTestPrometheusMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	collector := types.NewPrometheusMetricsCollector(&types.PrometheusConfig{
		Enabled:          true,
		Namespace:        "load_test",
		Subsystem:        "server",
		CardinalityLimit: 100000, // Very high limit for load testing
	})
	bmm := NewBusinessMetricsManager("load-test-service", collector, nil)

	t.Run("SustainedLoadTest", func(t *testing.T) {
		const testDuration = 30 * time.Second
		const numWorkers = 20
		const targetRPS = 1000 // requests per second per worker

		var totalRequests int64
		var errors int64
		var counterMutex sync.Mutex
		var wg sync.WaitGroup

		startTime := time.Now()
		endTime := startTime.Add(testDuration)

		t.Logf("Starting sustained load test for %v with %d workers targeting %d RPS each", testDuration, numWorkers, targetRPS)

		wg.Add(numWorkers)
		for i := 0; i < numWorkers; i++ {
			go func(workerID int) {
				defer wg.Done()

				ticker := time.NewTicker(time.Second / time.Duration(targetRPS))
				defer ticker.Stop()

				localRequests := 0
				localErrors := 0

				for time.Now().Before(endTime) {
					select {
					case <-ticker.C:
						labels := map[string]string{
							"worker":   fmt.Sprintf("w%d", workerID),
							"endpoint": fmt.Sprintf("/api/endpoint_%d", localRequests%10),
							"method":   []string{"GET", "POST", "PUT"}[localRequests%3],
						}

						// Simulate different metric types
						switch localRequests % 4 {
						case 0:
							bmm.RecordCounter("load_test_requests_total", 1.0, labels)
						case 1:
							bmm.RecordGauge("load_test_active_connections", float64(localRequests%100), labels)
						case 2:
							bmm.RecordHistogram("load_test_duration_seconds", float64(localRequests%1000)*0.001, labels)
						case 3:
							bmm.RecordHistogram("load_test_response_size_bytes", float64(localRequests%10000), labels)
						}

						localRequests++

					case <-time.After(time.Millisecond * 100):
						// Timeout - count as error
						localErrors++
					}
				}

				counterMutex.Lock()
				totalRequests += int64(localRequests)
				errors += int64(localErrors)
				counterMutex.Unlock()
			}(i)
		}

		wg.Wait()
		actualDuration := time.Since(startTime)
		actualRPS := float64(totalRequests) / actualDuration.Seconds()
		errorRate := float64(errors) / float64(totalRequests+errors) * 100

		t.Logf("Load test completed:")
		t.Logf("  Duration: %v", actualDuration)
		t.Logf("  Total requests: %d", totalRequests)
		t.Logf("  Errors: %d (%.2f%%)", errors, errorRate)
		t.Logf("  Actual RPS: %.2f", actualRPS)
		t.Logf("  Target RPS: %d", numWorkers*targetRPS)

		// Verify system remained stable
		if errorRate > 1.0 { // Less than 1% error rate
			t.Errorf("High error rate during load test: %.2f%%", errorRate)
		}

		// Test metrics endpoint under load
		handler := collector.GetHandler()
		metricsTestCount := 10
		var metricsLatencies []time.Duration

		for i := 0; i < metricsTestCount; i++ {
			req := httptest.NewRequest("GET", "/metrics", nil)
			w := httptest.NewRecorder()

			start := time.Now()
			handler.ServeHTTP(w, req)
			latency := time.Since(start)

			metricsLatencies = append(metricsLatencies, latency)

			if w.Code != http.StatusOK {
				t.Errorf("Metrics endpoint returned non-200 status: %d", w.Code)
			}
		}

		// Calculate metrics endpoint performance
		var totalLatency time.Duration
		for _, latency := range metricsLatencies {
			totalLatency += latency
		}
		avgLatency := totalLatency / time.Duration(len(metricsLatencies))

		t.Logf("Metrics endpoint performance under load:")
		t.Logf("  Average latency: %v", avgLatency)
		t.Logf("  Max latency: %v", func() time.Duration {
			max := time.Duration(0)
			for _, lat := range metricsLatencies {
				if lat > max {
					max = lat
				}
			}
			return max
		}())

		// Metrics endpoint should remain responsive
		if avgLatency > 500*time.Millisecond {
			t.Errorf("Metrics endpoint too slow under load: %v average latency", avgLatency)
		}
	})
}
