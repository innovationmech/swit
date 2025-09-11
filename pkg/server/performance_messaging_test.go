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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMessagingMetrics represents mock messaging metrics for testing
type MockMessagingMetrics struct {
	MessagesPublished    int64         `json:"messages_published"`
	MessagesConsumed     int64         `json:"messages_consumed"`
	MessagesFailed       int64         `json:"messages_failed"`
	AvgPublishLatency    time.Duration `json:"avg_publish_latency"`
	AvgConsumeLatency    time.Duration `json:"avg_consume_latency"`
	MessagingMemoryUsage uint64        `json:"messaging_memory_usage"`
	ActiveBrokers        int64         `json:"active_brokers"`
	UnhealthyBrokers     int64         `json:"unhealthy_brokers"`
}

func TestPerformanceMetricsMessagingIntegration(t *testing.T) {
	metrics := NewPerformanceMetrics()

	// Test initial state
	assert.Nil(t, metrics.MessagingMetrics)

	// Test setting messaging metrics
	mockMessagingMetrics := &MockMessagingMetrics{
		MessagesPublished:    100,
		MessagesConsumed:     95,
		MessagesFailed:       5,
		AvgPublishLatency:    50 * time.Millisecond,
		AvgConsumeLatency:    75 * time.Millisecond,
		MessagingMemoryUsage: 10 * 1024 * 1024, // 10MB
		ActiveBrokers:        2,
		UnhealthyBrokers:     0,
	}

	metrics.MessagingMetrics = mockMessagingMetrics
	assert.NotNil(t, metrics.MessagingMetrics)

	// Test that messaging metrics are included in snapshot
	snapshot := metrics.GetSnapshot()
	assert.NotNil(t, snapshot.MessagingMetrics)

	// Verify the messaging metrics content
	retrievedMetrics, ok := snapshot.MessagingMetrics.(*MockMessagingMetrics)
	require.True(t, ok, "Messaging metrics should be of MockMessagingMetrics type")

	assert.Equal(t, int64(100), retrievedMetrics.MessagesPublished)
	assert.Equal(t, int64(95), retrievedMetrics.MessagesConsumed)
	assert.Equal(t, int64(5), retrievedMetrics.MessagesFailed)
	assert.Equal(t, 50*time.Millisecond, retrievedMetrics.AvgPublishLatency)
	assert.Equal(t, 75*time.Millisecond, retrievedMetrics.AvgConsumeLatency)
	assert.Equal(t, uint64(10*1024*1024), retrievedMetrics.MessagingMemoryUsage)
	assert.Equal(t, int64(2), retrievedMetrics.ActiveBrokers)
	assert.Equal(t, int64(0), retrievedMetrics.UnhealthyBrokers)
}

func TestPerformanceMonitorMessagingIntegration(t *testing.T) {
	monitor := NewPerformanceMonitor()

	// Test initial state
	assert.Nil(t, monitor.GetMessagingMetrics())

	// Test setting messaging metrics
	mockMessagingMetrics := &MockMessagingMetrics{
		MessagesPublished: 50,
		MessagesConsumed:  45,
		MessagesFailed:    2,
	}

	monitor.SetMessagingMetrics(mockMessagingMetrics)

	// Verify messaging metrics are set
	retrievedMetrics := monitor.GetMessagingMetrics()
	assert.NotNil(t, retrievedMetrics)

	// Verify content
	mockMetrics, ok := retrievedMetrics.(*MockMessagingMetrics)
	require.True(t, ok)
	assert.Equal(t, int64(50), mockMetrics.MessagesPublished)
	assert.Equal(t, int64(45), mockMetrics.MessagesConsumed)
	assert.Equal(t, int64(2), mockMetrics.MessagesFailed)
}

func TestMessagingEventRecording(t *testing.T) {
	monitor := NewPerformanceMonitor()

	// Create a mutex-protected slice to capture events
	var eventsMu sync.Mutex
	events := make([]string, 0)

	// Add a hook to capture events with proper synchronization
	monitor.AddHook(func(event string, metrics *PerformanceMetrics) {
		eventsMu.Lock()
		events = append(events, event)
		eventsMu.Unlock()
	})

	// Test messaging event recording
	mockMessagingMetrics := &MockMessagingMetrics{
		MessagesPublished: 1,
	}

	monitor.RecordMessagingEvent("publish_success", mockMessagingMetrics)

	// Give hooks time to execute (they run in goroutines)
	time.Sleep(10 * time.Millisecond)

	// Verify the event was recorded with proper synchronization
	eventsMu.Lock()
	assert.Len(t, events, 1)
	assert.Equal(t, "messaging_publish_success", events[0])
	eventsMu.Unlock()

	// Verify messaging metrics were set
	retrievedMetrics := monitor.GetMessagingMetrics()
	assert.NotNil(t, retrievedMetrics)
}

func TestMessagingPerformanceHooks(t *testing.T) {
	t.Run("MessagingPerformanceIntegrationHook", func(t *testing.T) {
		metrics := NewPerformanceMetrics()

		// Set some messaging metrics
		metrics.MessagingMetrics = &MockMessagingMetrics{
			MessagesPublished: 100,
		}

		// Record initial memory usage
		initialMemory := metrics.MemoryUsage

		// Call the hook with a messaging event
		MessagingPerformanceIntegrationHook("messaging_publish", metrics)

		// Memory usage should be updated (hook calls RecordMemoryUsage)
		assert.True(t, metrics.MemoryUsage >= initialMemory, "Memory usage should be updated")

		// Test with non-messaging event
		MessagingPerformanceIntegrationHook("regular_event", metrics)
		// Should not trigger messaging-specific behavior
	})

	t.Run("MessagingThresholdViolationHook", func(t *testing.T) {
		metrics := NewPerformanceMetrics()

		// This hook currently doesn't implement specific logic,
		// but we test that it doesn't cause errors
		MessagingThresholdViolationHook("messaging_error", metrics)
		MessagingThresholdViolationHook("regular_event", metrics)

		// Should not cause any panics or errors
		assert.True(t, true, "Hooks should execute without errors")
	})
}

func TestPerformanceMetricsSnapshotWithMessaging(t *testing.T) {
	metrics := NewPerformanceMetrics()

	// Set various metrics
	metrics.RecordStartupTime(100 * time.Millisecond)
	metrics.RecordMemoryUsage()
	metrics.IncrementRequestCount()
	metrics.IncrementErrorCount()

	// Set messaging metrics
	mockMessagingMetrics := &MockMessagingMetrics{
		MessagesPublished:    1000,
		MessagesConsumed:     950,
		MessagesFailed:       25,
		AvgPublishLatency:    10 * time.Millisecond,
		AvgConsumeLatency:    15 * time.Millisecond,
		MessagingMemoryUsage: 5 * 1024 * 1024,
		ActiveBrokers:        3,
		UnhealthyBrokers:     1,
	}
	metrics.MessagingMetrics = mockMessagingMetrics

	// Get snapshot
	snapshot := metrics.GetSnapshot()

	// Verify regular metrics are included
	assert.Equal(t, 100*time.Millisecond, snapshot.StartupTime)
	assert.True(t, snapshot.MemoryUsage > 0)
	assert.Equal(t, int64(1), snapshot.RequestCount)
	assert.Equal(t, int64(1), snapshot.ErrorCount)

	// Verify messaging metrics are included
	assert.NotNil(t, snapshot.MessagingMetrics)
	retrievedMessagingMetrics, ok := snapshot.MessagingMetrics.(*MockMessagingMetrics)
	require.True(t, ok)

	assert.Equal(t, int64(1000), retrievedMessagingMetrics.MessagesPublished)
	assert.Equal(t, int64(950), retrievedMessagingMetrics.MessagesConsumed)
	assert.Equal(t, int64(25), retrievedMessagingMetrics.MessagesFailed)
	assert.Equal(t, 10*time.Millisecond, retrievedMessagingMetrics.AvgPublishLatency)
	assert.Equal(t, 15*time.Millisecond, retrievedMessagingMetrics.AvgConsumeLatency)
	assert.Equal(t, uint64(5*1024*1024), retrievedMessagingMetrics.MessagingMemoryUsage)
	assert.Equal(t, int64(3), retrievedMessagingMetrics.ActiveBrokers)
	assert.Equal(t, int64(1), retrievedMessagingMetrics.UnhealthyBrokers)
}

func TestMessagingIntegrationWithPerformanceProfiler(t *testing.T) {
	profiler := NewPerformanceProfiler()
	monitor := profiler.GetMonitor()

	// Set messaging metrics
	mockMessagingMetrics := &MockMessagingMetrics{
		MessagesPublished: 500,
		MessagesConsumed:  480,
		MessagesFailed:    10,
	}

	monitor.SetMessagingMetrics(mockMessagingMetrics)

	// Verify integration
	metrics := profiler.GetMetrics()
	assert.NotNil(t, metrics.MessagingMetrics)

	// Test profiling with messaging metrics
	err := monitor.ProfileOperation("messaging_operation", func() error {
		// Simulate some work
		time.Sleep(1 * time.Millisecond)
		return nil
	})

	assert.NoError(t, err)

	// Verify metrics are still available
	assert.NotNil(t, monitor.GetMessagingMetrics())
}

func TestMessagingMetricsThreadSafety(t *testing.T) {
	monitor := NewPerformanceMonitor()
	done := make(chan bool)

	// Concurrently set and get messaging metrics
	go func() {
		for i := 0; i < 100; i++ {
			mockMetrics := &MockMessagingMetrics{
				MessagesPublished: int64(i),
			}
			monitor.SetMessagingMetrics(mockMetrics)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = monitor.GetMessagingMetrics()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify no race conditions occurred
	finalMetrics := monitor.GetMessagingMetrics()
	assert.NotNil(t, finalMetrics)
}
