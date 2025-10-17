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

package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRecoveryMetrics(t *testing.T) {
	tests := []struct {
		name    string
		config  *RecoveryPrometheusConfig
		wantErr bool
	}{
		{
			name:    "default config",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "custom config",
			config:  DefaultRecoveryPrometheusConfig(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := NewRecoveryMetrics(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, metrics)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, metrics)
				assert.NotNil(t, metrics.prometheus)
				assert.NotNil(t, metrics.prometheus.registry)
			}
		})
	}
}

func TestRecoveryMetrics_RecordRecoveryAttempt(t *testing.T) {
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)
	require.NotNil(t, metrics)

	// Record some attempts
	metrics.RecordRecoveryAttempt("resume", "order_saga")
	metrics.RecordRecoveryAttempt("resume", "payment_saga")
	metrics.RecordRecoveryAttempt("compensate", "order_saga")

	snapshot := metrics.GetSnapshot()
	assert.Equal(t, int64(3), snapshot.TotalAttempts)
}

func TestRecoveryMetrics_RecordRecoverySuccess(t *testing.T) {
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	// Record attempts and successes
	metrics.RecordRecoveryAttempt("resume", "order_saga")
	metrics.RecordRecoverySuccess("resume", "order_saga", 2*time.Second)

	metrics.RecordRecoveryAttempt("resume", "payment_saga")
	metrics.RecordRecoverySuccess("resume", "payment_saga", 3*time.Second)

	snapshot := metrics.GetSnapshot()
	assert.Equal(t, int64(2), snapshot.TotalAttempts)
	assert.Equal(t, int64(2), snapshot.SuccessfulRecoveries)
	assert.Equal(t, int64(0), snapshot.FailedRecoveries)
	assert.Equal(t, float64(1.0), snapshot.SuccessRate)
	assert.Greater(t, snapshot.AverageDuration, time.Duration(0))
}

func TestRecoveryMetrics_RecordRecoveryFailure(t *testing.T) {
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	// Record attempts and failures
	metrics.RecordRecoveryAttempt("resume", "order_saga")
	metrics.RecordRecoveryFailure("resume", "order_saga", "timeout", 5*time.Second)

	metrics.RecordRecoveryAttempt("compensate", "payment_saga")
	metrics.RecordRecoveryFailure("compensate", "payment_saga", "execution_error", 3*time.Second)

	snapshot := metrics.GetSnapshot()
	assert.Equal(t, int64(2), snapshot.TotalAttempts)
	assert.Equal(t, int64(0), snapshot.SuccessfulRecoveries)
	assert.Equal(t, int64(2), snapshot.FailedRecoveries)
	assert.Equal(t, float64(0.0), snapshot.SuccessRate)
}

func TestRecoveryMetrics_RecoveringInProgress(t *testing.T) {
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	// Test increment/decrement
	metrics.IncrementRecoveringInProgress("resume")
	snapshot := metrics.GetSnapshot()
	assert.Equal(t, int64(1), snapshot.CurrentlyRecovering)

	metrics.IncrementRecoveringInProgress("compensate")
	snapshot = metrics.GetSnapshot()
	assert.Equal(t, int64(2), snapshot.CurrentlyRecovering)

	metrics.DecrementRecoveringInProgress("resume")
	snapshot = metrics.GetSnapshot()
	assert.Equal(t, int64(1), snapshot.CurrentlyRecovering)

	metrics.DecrementRecoveringInProgress("compensate")
	snapshot = metrics.GetSnapshot()
	assert.Equal(t, int64(0), snapshot.CurrentlyRecovering)

	// Decrement should not go negative
	metrics.DecrementRecoveringInProgress("resume")
	snapshot = metrics.GetSnapshot()
	assert.Equal(t, int64(0), snapshot.CurrentlyRecovering)
}

func TestRecoveryMetrics_RecordDetectedFailure(t *testing.T) {
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	// Record different types of detected failures
	metrics.RecordDetectedFailure("timeout")
	metrics.RecordDetectedFailure("timeout")
	metrics.RecordDetectedFailure("stuck")
	metrics.RecordDetectedFailure("compensating")
	metrics.RecordDetectedFailure("inconsistent")

	snapshot := metrics.GetSnapshot()
	assert.Equal(t, int64(2), snapshot.DetectedTimeouts)
	assert.Equal(t, int64(1), snapshot.DetectedStuck)
	assert.Equal(t, int64(1), snapshot.DetectedCompensating)
	assert.Equal(t, int64(1), snapshot.DetectedInconsistent)
}

func TestRecoveryMetrics_RecordManualIntervention(t *testing.T) {
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	// Record manual interventions
	metrics.RecordManualIntervention()
	metrics.RecordManualIntervention()
	metrics.RecordManualIntervention()

	snapshot := metrics.GetSnapshot()
	assert.Equal(t, int64(3), snapshot.ManualInterventions)
}

func TestRecoveryMetrics_Percentiles(t *testing.T) {
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	// Record multiple successes with different durations
	durations := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		3 * time.Second,
		4 * time.Second,
		5 * time.Second,
		6 * time.Second,
		7 * time.Second,
		8 * time.Second,
		9 * time.Second,
		10 * time.Second,
	}

	for _, d := range durations {
		metrics.RecordRecoveryAttempt("resume", "test_saga")
		metrics.RecordRecoverySuccess("resume", "test_saga", d)
	}

	snapshot := metrics.GetSnapshot()
	assert.Equal(t, 5*time.Second+500*time.Millisecond, snapshot.AverageDuration)
	assert.Greater(t, snapshot.P50Duration, time.Duration(0))
	assert.GreaterOrEqual(t, snapshot.P95Duration, snapshot.P50Duration)
	assert.GreaterOrEqual(t, snapshot.P99Duration, snapshot.P95Duration)
}

func TestRecoveryMetrics_SuccessRate(t *testing.T) {
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	// Record mix of successes and failures
	for i := 0; i < 7; i++ {
		metrics.RecordRecoveryAttempt("resume", "test_saga")
		metrics.RecordRecoverySuccess("resume", "test_saga", time.Second)
	}

	for i := 0; i < 3; i++ {
		metrics.RecordRecoveryAttempt("resume", "test_saga")
		metrics.RecordRecoveryFailure("resume", "test_saga", "timeout", time.Second)
	}

	snapshot := metrics.GetSnapshot()
	assert.Equal(t, int64(10), snapshot.TotalAttempts)
	assert.Equal(t, int64(7), snapshot.SuccessfulRecoveries)
	assert.Equal(t, int64(3), snapshot.FailedRecoveries)
	assert.InDelta(t, 0.7, snapshot.SuccessRate, 0.01)
}

func TestRecoveryMetrics_Reset(t *testing.T) {
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	// Record some data
	metrics.RecordRecoveryAttempt("resume", "test_saga")
	metrics.RecordRecoverySuccess("resume", "test_saga", time.Second)
	metrics.RecordDetectedFailure("timeout")
	metrics.RecordManualIntervention()
	metrics.IncrementRecoveringInProgress("resume")

	// Reset
	metrics.Reset()

	// Verify all counters are reset
	snapshot := metrics.GetSnapshot()
	assert.Equal(t, int64(0), snapshot.TotalAttempts)
	assert.Equal(t, int64(0), snapshot.SuccessfulRecoveries)
	assert.Equal(t, int64(0), snapshot.FailedRecoveries)
	assert.Equal(t, int64(0), snapshot.CurrentlyRecovering)
	assert.Equal(t, int64(0), snapshot.DetectedTimeouts)
	assert.Equal(t, int64(0), snapshot.ManualInterventions)
	assert.Equal(t, float64(0), snapshot.SuccessRate)
}

func TestRecoveryMetrics_DurationLimiting(t *testing.T) {
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	// Record more durations than the max
	for i := 0; i < 1500; i++ {
		metrics.RecordRecoveryAttempt("resume", "test_saga")
		metrics.RecordRecoverySuccess("resume", "test_saga", time.Duration(i)*time.Millisecond)
	}

	// Should only keep the last 1000 durations
	snapshot := metrics.GetSnapshot()
	assert.Equal(t, int64(1500), snapshot.TotalAttempts)

	// The durations array should be limited
	metrics.mu.RLock()
	assert.LessOrEqual(t, len(metrics.recoveryDurations), metrics.maxRecordedDurations)
	metrics.mu.RUnlock()
}

func TestRecoveryMetrics_GetRegistry(t *testing.T) {
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	registry := metrics.GetRegistry()
	assert.NotNil(t, registry)
	assert.Equal(t, metrics.prometheus.registry, registry)
}

func TestRecoveryMetrics_ConcurrentAccess(t *testing.T) {
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	// Test concurrent access
	done := make(chan bool)
	operations := 100

	// Concurrent increments
	go func() {
		for i := 0; i < operations; i++ {
			metrics.RecordRecoveryAttempt("resume", "test_saga")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < operations; i++ {
			metrics.RecordRecoverySuccess("resume", "test_saga", time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < operations; i++ {
			metrics.RecordDetectedFailure("timeout")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < operations; i++ {
			metrics.IncrementRecoveringInProgress("resume")
			metrics.DecrementRecoveringInProgress("resume")
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}

	// Verify no race conditions occurred
	snapshot := metrics.GetSnapshot()
	assert.Equal(t, int64(operations), snapshot.TotalAttempts)
	assert.Equal(t, int64(operations), snapshot.SuccessfulRecoveries)
	assert.Equal(t, int64(operations), snapshot.DetectedTimeouts)
}

