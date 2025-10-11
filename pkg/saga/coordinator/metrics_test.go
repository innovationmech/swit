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

package coordinator

import (
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrometheusMetricsCollector(t *testing.T) {
	tests := []struct {
		name    string
		config  *PrometheusMetricsConfig
		wantErr bool
	}{
		{
			name:    "with nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "with default config",
			config:  DefaultPrometheusMetricsConfig(),
			wantErr: false,
		},
		{
			name: "with custom config",
			config: &PrometheusMetricsConfig{
				Namespace:       "custom_saga",
				Subsystem:       "custom_coordinator",
				Registry:        prometheus.NewRegistry(),
				DurationBuckets: []float64{0.1, 1.0, 10.0},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector, err := NewPrometheusMetricsCollector(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, collector)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, collector)
				assert.NotNil(t, collector.GetRegistry())
			}
		})
	}
}

func TestPrometheusMetricsCollector_RecordSagaStarted(t *testing.T) {
	collector, err := NewPrometheusMetricsCollector(nil)
	require.NoError(t, err)
	require.NotNil(t, collector)

	definitionID := "test-saga-def"

	// Record saga started
	collector.RecordSagaStarted(definitionID)
	collector.RecordSagaStarted(definitionID)

	// Verify counter increased
	metricValue := getCounterValue(t, collector.sagaStartedTotal, definitionID)
	assert.Equal(t, float64(2), metricValue)
}

func TestPrometheusMetricsCollector_RecordSagaCompleted(t *testing.T) {
	collector, err := NewPrometheusMetricsCollector(nil)
	require.NoError(t, err)

	definitionID := "test-saga-def"
	duration := 5 * time.Second

	// Record saga completed
	collector.RecordSagaCompleted(definitionID, duration)

	// Verify counter increased
	metricValue := getCounterValue(t, collector.sagaCompletedTotal, definitionID)
	assert.Equal(t, float64(1), metricValue)

	// Verify histogram recorded
	histogramValue := getHistogramCount(t, collector.sagaDuration, definitionID, "completed")
	assert.Equal(t, uint64(1), histogramValue)
}

func TestPrometheusMetricsCollector_RecordSagaFailed(t *testing.T) {
	collector, err := NewPrometheusMetricsCollector(nil)
	require.NoError(t, err)

	definitionID := "test-saga-def"
	errorType := saga.ErrorTypeBusiness
	duration := 3 * time.Second

	// Record saga failed
	collector.RecordSagaFailed(definitionID, errorType, duration)

	// Verify counter increased
	metricValue := getCounterValue2(t, collector.sagaFailedTotal, definitionID, string(errorType))
	assert.Equal(t, float64(1), metricValue)

	// Verify histogram recorded
	histogramValue := getHistogramCount(t, collector.sagaDuration, definitionID, "failed")
	assert.Equal(t, uint64(1), histogramValue)
}

func TestPrometheusMetricsCollector_RecordSagaCancelled(t *testing.T) {
	collector, err := NewPrometheusMetricsCollector(nil)
	require.NoError(t, err)

	definitionID := "test-saga-def"
	duration := 2 * time.Second

	// Record saga cancelled
	collector.RecordSagaCancelled(definitionID, duration)

	// Verify counter increased
	metricValue := getCounterValue(t, collector.sagaCancelledTotal, definitionID)
	assert.Equal(t, float64(1), metricValue)

	// Verify histogram recorded
	histogramValue := getHistogramCount(t, collector.sagaDuration, definitionID, "cancelled")
	assert.Equal(t, uint64(1), histogramValue)
}

func TestPrometheusMetricsCollector_RecordSagaTimedOut(t *testing.T) {
	collector, err := NewPrometheusMetricsCollector(nil)
	require.NoError(t, err)

	definitionID := "test-saga-def"
	duration := 30 * time.Second

	// Record saga timed out
	collector.RecordSagaTimedOut(definitionID, duration)

	// Verify counter increased
	metricValue := getCounterValue(t, collector.sagaTimedOutTotal, definitionID)
	assert.Equal(t, float64(1), metricValue)

	// Verify histogram recorded
	histogramValue := getHistogramCount(t, collector.sagaDuration, definitionID, "timed_out")
	assert.Equal(t, uint64(1), histogramValue)
}

func TestPrometheusMetricsCollector_RecordStepExecuted(t *testing.T) {
	collector, err := NewPrometheusMetricsCollector(nil)
	require.NoError(t, err)

	definitionID := "test-saga-def"
	stepID := "step-1"
	duration := 1 * time.Second

	// Record successful step execution
	collector.RecordStepExecuted(definitionID, stepID, true, duration)

	// Verify counter increased
	metricValue := getCounterValue2(t, collector.stepExecutedTotal, definitionID, stepID, "true")
	assert.Equal(t, float64(1), metricValue)

	// Verify histogram recorded
	histogramValue := getHistogramCount2(t, collector.stepDuration, definitionID, stepID, "true")
	assert.Equal(t, uint64(1), histogramValue)

	// Record failed step execution
	collector.RecordStepExecuted(definitionID, stepID, false, duration)

	// Verify counter increased
	metricValue = getCounterValue2(t, collector.stepExecutedTotal, definitionID, stepID, "false")
	assert.Equal(t, float64(1), metricValue)
}

func TestPrometheusMetricsCollector_RecordStepRetried(t *testing.T) {
	collector, err := NewPrometheusMetricsCollector(nil)
	require.NoError(t, err)

	definitionID := "test-saga-def"
	stepID := "step-1"

	// Record step retries
	collector.RecordStepRetried(definitionID, stepID, 1)
	collector.RecordStepRetried(definitionID, stepID, 2)
	collector.RecordStepRetried(definitionID, stepID, 3)
	collector.RecordStepRetried(definitionID, stepID, 5) // This should be labeled as "4+"

	// Verify counters
	metricValue1 := getCounterValue2(t, collector.stepRetriedTotal, definitionID, stepID, "1")
	assert.Equal(t, float64(1), metricValue1)

	metricValue2 := getCounterValue2(t, collector.stepRetriedTotal, definitionID, stepID, "2")
	assert.Equal(t, float64(1), metricValue2)

	metricValue3 := getCounterValue2(t, collector.stepRetriedTotal, definitionID, stepID, "3")
	assert.Equal(t, float64(1), metricValue3)

	metricValue4Plus := getCounterValue2(t, collector.stepRetriedTotal, definitionID, stepID, "4+")
	assert.Equal(t, float64(1), metricValue4Plus)
}

func TestPrometheusMetricsCollector_RecordCompensationExecuted(t *testing.T) {
	collector, err := NewPrometheusMetricsCollector(nil)
	require.NoError(t, err)

	definitionID := "test-saga-def"
	stepID := "step-1"
	duration := 500 * time.Millisecond

	// Record successful compensation
	collector.RecordCompensationExecuted(definitionID, stepID, true, duration)

	// Verify counter increased
	metricValue := getCounterValue2(t, collector.compensationExecutedTotal, definitionID, stepID, "true")
	assert.Equal(t, float64(1), metricValue)

	// Verify histogram recorded
	histogramValue := getHistogramCount2(t, collector.compensationDuration, definitionID, stepID, "true")
	assert.Equal(t, uint64(1), histogramValue)

	// Record failed compensation
	collector.RecordCompensationExecuted(definitionID, stepID, false, duration)

	// Verify counter increased
	metricValue = getCounterValue2(t, collector.compensationExecutedTotal, definitionID, stepID, "false")
	assert.Equal(t, float64(1), metricValue)
}

func TestPrometheusMetricsCollector_GetRegistry(t *testing.T) {
	config := DefaultPrometheusMetricsConfig()
	collector, err := NewPrometheusMetricsCollector(config)
	require.NoError(t, err)

	registry := collector.GetRegistry()
	assert.NotNil(t, registry)
	assert.Equal(t, config.Registry, registry)
}

func TestDefaultPrometheusMetricsConfig(t *testing.T) {
	config := DefaultPrometheusMetricsConfig()

	assert.Equal(t, "saga", config.Namespace)
	assert.Equal(t, "coordinator", config.Subsystem)
	assert.NotNil(t, config.Registry)
	assert.NotNil(t, config.DurationBuckets)
	assert.Greater(t, len(config.DurationBuckets), 0)
}

// Helper function to get counter value with 1 label
func getCounterValue(t *testing.T, counter *prometheus.CounterVec, label1 string) float64 {
	metric := &dto.Metric{}
	err := counter.WithLabelValues(label1).Write(metric)
	require.NoError(t, err)
	return metric.GetCounter().GetValue()
}

// Helper function to get counter value with 2 labels
func getCounterValue2(t *testing.T, counter *prometheus.CounterVec, label1, label2 string, additionalLabels ...string) float64 {
	labels := []string{label1, label2}
	labels = append(labels, additionalLabels...)
	metric := &dto.Metric{}
	err := counter.WithLabelValues(labels...).Write(metric)
	require.NoError(t, err)
	return metric.GetCounter().GetValue()
}

// Helper function to get histogram count with 2 labels
func getHistogramCount(t *testing.T, histogram *prometheus.HistogramVec, label1, label2 string) uint64 {
	// Create a metric channel to collect metrics
	metricChan := make(chan prometheus.Metric, 1)
	histogram.WithLabelValues(label1, label2).(prometheus.Histogram).Collect(metricChan)

	metric := &dto.Metric{}
	select {
	case m := <-metricChan:
		err := m.Write(metric)
		require.NoError(t, err)
		return metric.GetHistogram().GetSampleCount()
	default:
		return 0
	}
}

// Helper function to get histogram count with 3 labels
func getHistogramCount2(t *testing.T, histogram *prometheus.HistogramVec, label1, label2, label3 string) uint64 {
	// Create a metric channel to collect metrics
	metricChan := make(chan prometheus.Metric, 1)
	histogram.WithLabelValues(label1, label2, label3).(prometheus.Histogram).Collect(metricChan)

	metric := &dto.Metric{}
	select {
	case m := <-metricChan:
		err := m.Write(metric)
		require.NoError(t, err)
		return metric.GetHistogram().GetSampleCount()
	default:
		return 0
	}
}
