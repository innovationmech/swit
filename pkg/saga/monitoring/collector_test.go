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

package monitoring

import (
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSagaMetricsCollector tests the creation of a new SagaMetricsCollector.
func TestNewSagaMetricsCollector(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "with nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "with custom config",
			config: &Config{
				Namespace:       "custom",
				Subsystem:       "test",
				Registry:        prometheus.NewRegistry(),
				DurationBuckets: []float64{0.1, 1, 10},
			},
			wantErr: false,
		},
		{
			name: "with partial config",
			config: &Config{
				Namespace: "partial",
				Registry:  prometheus.NewRegistry(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector, err := NewSagaMetricsCollector(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, collector)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, collector)
				assert.NotNil(t, collector.registry)
				assert.NotNil(t, collector.sagaStartedCounter)
				assert.NotNil(t, collector.sagaCompletedCounter)
				assert.NotNil(t, collector.sagaFailedCounter)
				assert.NotNil(t, collector.sagaDurationHistogram)
				assert.NotNil(t, collector.activeSagasGauge)
			}
		})
	}
}

// TestSagaMetricsCollector_RecordSagaStarted tests the RecordSagaStarted method.
func TestSagaMetricsCollector_RecordSagaStarted(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	// Record multiple Saga starts
	for i := 0; i < 5; i++ {
		collector.RecordSagaStarted("saga-" + string(rune(i)))
	}

	// Verify metrics
	metrics := collector.GetMetrics()
	assert.Equal(t, int64(5), metrics.SagasStarted)
	assert.Equal(t, int64(5), metrics.ActiveSagas)
	assert.Equal(t, int64(5), collector.GetActiveSagasCount())
}

// TestSagaMetricsCollector_RecordSagaCompleted tests the RecordSagaCompleted method.
func TestSagaMetricsCollector_RecordSagaCompleted(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	// Start and complete Sagas
	collector.RecordSagaStarted("saga-1")
	collector.RecordSagaStarted("saga-2")
	collector.RecordSagaStarted("saga-3")

	collector.RecordSagaCompleted("saga-1", 100*time.Millisecond)
	collector.RecordSagaCompleted("saga-2", 200*time.Millisecond)

	// Verify metrics
	metrics := collector.GetMetrics()
	assert.Equal(t, int64(3), metrics.SagasStarted)
	assert.Equal(t, int64(2), metrics.SagasCompleted)
	assert.Equal(t, int64(1), metrics.ActiveSagas)

	// Verify duration calculations
	expectedTotal := 0.1 + 0.2
	assert.InDelta(t, expectedTotal, metrics.TotalDuration, 0.001)
	assert.InDelta(t, expectedTotal/2, metrics.AvgDuration, 0.001)
}

// TestSagaMetricsCollector_RecordSagaFailed tests the RecordSagaFailed method.
func TestSagaMetricsCollector_RecordSagaFailed(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	// Start and fail Sagas with different reasons
	collector.RecordSagaStarted("saga-1")
	collector.RecordSagaStarted("saga-2")
	collector.RecordSagaStarted("saga-3")

	collector.RecordSagaFailed("saga-1", "timeout")
	collector.RecordSagaFailed("saga-2", "validation_error")
	collector.RecordSagaFailed("saga-3", "timeout")

	// Verify metrics
	metrics := collector.GetMetrics()
	assert.Equal(t, int64(3), metrics.SagasStarted)
	assert.Equal(t, int64(3), metrics.SagasFailed)
	assert.Equal(t, int64(0), metrics.ActiveSagas)

	// Verify failure reasons
	assert.Equal(t, int64(2), metrics.FailureReasons["timeout"])
	assert.Equal(t, int64(1), metrics.FailureReasons["validation_error"])
}

// TestSagaMetricsCollector_MixedOperations tests a mix of different operations.
func TestSagaMetricsCollector_MixedOperations(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	// Start 10 Sagas
	for i := 0; i < 10; i++ {
		collector.RecordSagaStarted("saga-" + string(rune(i)))
	}

	// Complete 5 Sagas
	for i := 0; i < 5; i++ {
		collector.RecordSagaCompleted("saga-"+string(rune(i)), time.Duration(i+1)*100*time.Millisecond)
	}

	// Fail 3 Sagas
	collector.RecordSagaFailed("saga-5", "timeout")
	collector.RecordSagaFailed("saga-6", "error")
	collector.RecordSagaFailed("saga-7", "timeout")

	// Verify metrics
	metrics := collector.GetMetrics()
	assert.Equal(t, int64(10), metrics.SagasStarted)
	assert.Equal(t, int64(5), metrics.SagasCompleted)
	assert.Equal(t, int64(3), metrics.SagasFailed)
	assert.Equal(t, int64(2), metrics.ActiveSagas) // 10 started - 5 completed - 3 failed = 2 active

	// Verify failure reasons
	assert.Equal(t, int64(2), metrics.FailureReasons["timeout"])
	assert.Equal(t, int64(1), metrics.FailureReasons["error"])
}

// TestSagaMetricsCollector_GetActiveSagasCount tests the GetActiveSagasCount method.
func TestSagaMetricsCollector_GetActiveSagasCount(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	assert.Equal(t, int64(0), collector.GetActiveSagasCount())

	collector.RecordSagaStarted("saga-1")
	assert.Equal(t, int64(1), collector.GetActiveSagasCount())

	collector.RecordSagaStarted("saga-2")
	assert.Equal(t, int64(2), collector.GetActiveSagasCount())

	collector.RecordSagaCompleted("saga-1", 100*time.Millisecond)
	assert.Equal(t, int64(1), collector.GetActiveSagasCount())

	collector.RecordSagaFailed("saga-2", "error")
	assert.Equal(t, int64(0), collector.GetActiveSagasCount())
}

// TestSagaMetricsCollector_Reset tests the Reset method.
func TestSagaMetricsCollector_Reset(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	// Record some metrics
	collector.RecordSagaStarted("saga-1")
	collector.RecordSagaCompleted("saga-1", 100*time.Millisecond)
	collector.RecordSagaStarted("saga-2")
	collector.RecordSagaFailed("saga-2", "error")

	// Verify metrics are recorded
	metrics := collector.GetMetrics()
	assert.Equal(t, int64(2), metrics.SagasStarted)
	assert.Equal(t, int64(1), metrics.SagasCompleted)
	assert.Equal(t, int64(1), metrics.SagasFailed)

	// Reset
	collector.Reset()

	// Verify metrics are cleared
	metrics = collector.GetMetrics()
	assert.Equal(t, int64(0), metrics.SagasStarted)
	assert.Equal(t, int64(0), metrics.SagasCompleted)
	assert.Equal(t, int64(0), metrics.SagasFailed)
	assert.Equal(t, int64(0), metrics.ActiveSagas)
	assert.Equal(t, float64(0), metrics.TotalDuration)
	assert.Equal(t, float64(0), metrics.AvgDuration)
	assert.Empty(t, metrics.FailureReasons)
}

// TestSagaMetricsCollector_Concurrent tests concurrent access to the collector.
func TestSagaMetricsCollector_Concurrent(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	var wg sync.WaitGroup
	numGoroutines := 100
	operationsPerGoroutine := 10

	// Concurrent saga starts
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				collector.RecordSagaStarted("saga-" + string(rune(id*operationsPerGoroutine+j)))
			}
		}(i)
	}
	wg.Wait()

	// Concurrent saga completions and failures
	wg.Add(numGoroutines * 2)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine/2; j++ {
				collector.RecordSagaCompleted("saga-"+string(rune(id*operationsPerGoroutine+j)), 100*time.Millisecond)
			}
		}(i)

		go func(id int) {
			defer wg.Done()
			for j := operationsPerGoroutine / 2; j < operationsPerGoroutine; j++ {
				collector.RecordSagaFailed("saga-"+string(rune(id*operationsPerGoroutine+j)), "concurrent_error")
			}
		}(i)
	}
	wg.Wait()

	// Verify metrics
	metrics := collector.GetMetrics()
	expectedTotal := numGoroutines * operationsPerGoroutine
	assert.Equal(t, int64(expectedTotal), metrics.SagasStarted)
	assert.Equal(t, int64(expectedTotal/2), metrics.SagasCompleted)
	assert.Equal(t, int64(expectedTotal/2), metrics.SagasFailed)
	assert.Equal(t, int64(0), metrics.ActiveSagas)
}

// TestSagaMetricsCollector_GetMetricsSnapshot tests that GetMetrics returns a consistent snapshot.
func TestSagaMetricsCollector_GetMetricsSnapshot(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	collector.RecordSagaStarted("saga-1")
	collector.RecordSagaCompleted("saga-1", 100*time.Millisecond)

	metrics1 := collector.GetMetrics()
	metrics2 := collector.GetMetrics()

	// Verify snapshots are consistent
	assert.Equal(t, metrics1.SagasStarted, metrics2.SagasStarted)
	assert.Equal(t, metrics1.SagasCompleted, metrics2.SagasCompleted)
	assert.Equal(t, metrics1.SagasFailed, metrics2.SagasFailed)
	assert.Equal(t, metrics1.ActiveSagas, metrics2.ActiveSagas)

	// Verify failure reasons map is copied (not the same reference)
	metrics1.FailureReasons["test"] = 999
	assert.NotEqual(t, metrics1.FailureReasons["test"], metrics2.FailureReasons["test"])
}

// TestDefaultConfig tests the DefaultConfig function.
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "saga", config.Namespace)
	assert.Equal(t, "monitoring", config.Subsystem)
	assert.NotNil(t, config.Registry)
	assert.NotEmpty(t, config.DurationBuckets)
	assert.Equal(t, []float64{0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0}, config.DurationBuckets)
}

// TestMetricsTimestamp tests that the Metrics struct has a valid timestamp.
func TestMetricsTimestamp(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	before := time.Now()
	metrics := collector.GetMetrics()
	after := time.Now()

	assert.True(t, metrics.Timestamp.After(before) || metrics.Timestamp.Equal(before))
	assert.True(t, metrics.Timestamp.Before(after) || metrics.Timestamp.Equal(after))
}

// TestSagaMetricsCollector_DurationAccuracy tests duration recording accuracy.
func TestSagaMetricsCollector_DurationAccuracy(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	durations := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
	}

	for i, d := range durations {
		collector.RecordSagaStarted("saga-" + string(rune(i)))
		collector.RecordSagaCompleted("saga-"+string(rune(i)), d)
	}

	metrics := collector.GetMetrics()

	// Calculate expected total and average
	var expectedTotal float64
	for _, d := range durations {
		expectedTotal += d.Seconds()
	}
	expectedAvg := expectedTotal / float64(len(durations))

	assert.InDelta(t, expectedTotal, metrics.TotalDuration, 0.001)
	assert.InDelta(t, expectedAvg, metrics.AvgDuration, 0.001)
}

// TestSagaMetricsCollector_EmptyFailureReasons tests failure reasons when no failures occur.
func TestSagaMetricsCollector_EmptyFailureReasons(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	collector.RecordSagaStarted("saga-1")
	collector.RecordSagaCompleted("saga-1", 100*time.Millisecond)

	metrics := collector.GetMetrics()
	assert.Empty(t, metrics.FailureReasons)
}

// TestSagaMetricsCollector_DuplicateRegistration tests that duplicate registration fails.
func TestSagaMetricsCollector_DuplicateRegistration(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	// First registration should succeed
	collector1, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)
	require.NotNil(t, collector1)

	// Second registration with the same registry should fail
	collector2, err := NewSagaMetricsCollector(config)
	assert.Error(t, err)
	assert.Nil(t, collector2)
}

// BenchmarkSagaMetricsCollector_RecordSagaStarted benchmarks RecordSagaStarted.
func BenchmarkSagaMetricsCollector_RecordSagaStarted(b *testing.B) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordSagaStarted("saga-" + string(rune(i)))
	}
}

// BenchmarkSagaMetricsCollector_RecordSagaCompleted benchmarks RecordSagaCompleted.
func BenchmarkSagaMetricsCollector_RecordSagaCompleted(b *testing.B) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordSagaCompleted("saga-"+string(rune(i)), 100*time.Millisecond)
	}
}

// BenchmarkSagaMetricsCollector_GetMetrics benchmarks GetMetrics.
func BenchmarkSagaMetricsCollector_GetMetrics(b *testing.B) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(b, err)

	// Populate with some data
	for i := 0; i < 100; i++ {
		collector.RecordSagaStarted("saga-" + string(rune(i)))
		collector.RecordSagaCompleted("saga-"+string(rune(i)), 100*time.Millisecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = collector.GetMetrics()
	}
}

// BenchmarkSagaMetricsCollector_Concurrent benchmarks concurrent operations.
func BenchmarkSagaMetricsCollector_Concurrent(b *testing.B) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			collector.RecordSagaStarted("saga-" + string(rune(i)))
			collector.RecordSagaCompleted("saga-"+string(rune(i)), 100*time.Millisecond)
			i++
		}
	})
}
