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
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// RecoveryMetrics tracks recovery operation metrics for monitoring and alerting.
// It provides comprehensive insights into the recovery system's performance,
// including attempt counts, success/failure rates, timing information, and
// failure classification.
type RecoveryMetrics struct {
	// Total recovery attempts
	totalAttempts int64

	// Successful recoveries
	successfulRecoveries int64

	// Failed recoveries
	failedRecoveries int64

	// Currently recovering Sagas
	currentlyRecovering int64

	// Manual intervention requests
	manualInterventions int64

	// Detected failures by type
	detectedTimeouts     int64
	detectedStuck        int64
	detectedCompensating int64
	detectedInconsistent int64

	// Recovery timing statistics
	recoveryDurations    []time.Duration
	maxRecordedDurations int

	// Prometheus metrics
	prometheus *RecoveryPrometheusMetrics

	// Mutex for thread-safety
	mu sync.RWMutex
}

// RecoveryPrometheusMetrics holds Prometheus metrics for recovery operations.
type RecoveryPrometheusMetrics struct {
	// Recovery attempt counters
	recoveryAttemptsTotal *prometheus.CounterVec
	recoverySuccessTotal  *prometheus.CounterVec
	recoveryFailureTotal  *prometheus.CounterVec

	// Recovery duration histogram
	recoveryDuration *prometheus.HistogramVec

	// Currently recovering gauge
	recoveringInProgress *prometheus.GaugeVec

	// Failure detection counters
	detectedFailuresTotal *prometheus.CounterVec

	// Manual intervention counter
	manualInterventionsTotal prometheus.Counter

	// Recovery success rate summary
	recoverySuccessRate prometheus.Summary

	// Registry
	registry *prometheus.Registry
}

// RecoveryPrometheusConfig contains configuration for Prometheus metrics.
type RecoveryPrometheusConfig struct {
	// Namespace is the Prometheus namespace for all metrics (default: "saga")
	Namespace string

	// Subsystem is the Prometheus subsystem for all metrics (default: "recovery")
	Subsystem string

	// Registry is the Prometheus registry to use. If nil, a new registry is created.
	Registry *prometheus.Registry

	// DurationBuckets defines the buckets for duration histograms.
	// If nil, default buckets are used for recovery operations
	DurationBuckets []float64
}

// DefaultRecoveryPrometheusConfig returns a default configuration for Prometheus metrics.
func DefaultRecoveryPrometheusConfig() *RecoveryPrometheusConfig {
	return &RecoveryPrometheusConfig{
		Namespace: "saga",
		Subsystem: "recovery",
		Registry:  prometheus.NewRegistry(),
		// Recovery operations typically take 1-30 seconds
		DurationBuckets: []float64{
			0.5, 1.0, 2.5, 5.0, 10.0, 15.0, 20.0, 30.0, 60.0, 120.0,
		},
	}
}

// NewRecoveryMetrics creates a new RecoveryMetrics instance with Prometheus integration.
//
// Parameters:
//   - config: Configuration for Prometheus metrics. If nil, default config is used.
//
// Returns:
//   - *RecoveryMetrics: A configured metrics instance.
//   - error: An error if Prometheus registration fails.
func NewRecoveryMetrics(config *RecoveryPrometheusConfig) (*RecoveryMetrics, error) {
	if config == nil {
		config = DefaultRecoveryPrometheusConfig()
	}

	if config.Registry == nil {
		config.Registry = prometheus.NewRegistry()
	}

	if config.Namespace == "" {
		config.Namespace = "saga"
	}

	if config.Subsystem == "" {
		config.Subsystem = "recovery"
	}

	if config.DurationBuckets == nil {
		config.DurationBuckets = []float64{
			0.5, 1.0, 2.5, 5.0, 10.0, 15.0, 20.0, 30.0, 60.0, 120.0,
		}
	}

	pm := &RecoveryPrometheusMetrics{
		registry: config.Registry,
	}

	// Initialize recovery attempt counters
	pm.recoveryAttemptsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "attempts_total",
			Help:      "Total number of recovery attempts",
		},
		[]string{"strategy", "saga_type"},
	)

	pm.recoverySuccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "success_total",
			Help:      "Total number of successful recoveries",
		},
		[]string{"strategy", "saga_type"},
	)

	pm.recoveryFailureTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "failure_total",
			Help:      "Total number of failed recoveries",
		},
		[]string{"strategy", "saga_type", "error_type"},
	)

	// Initialize recovery duration histogram
	pm.recoveryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "duration_seconds",
			Help:      "Recovery operation duration in seconds",
			Buckets:   config.DurationBuckets,
		},
		[]string{"strategy", "saga_type"},
	)

	// Initialize currently recovering gauge
	pm.recoveringInProgress = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "in_progress",
			Help:      "Number of Sagas currently being recovered",
		},
		[]string{"strategy"},
	)

	// Initialize failure detection counters
	pm.detectedFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "detected_failures_total",
			Help:      "Total number of detected failures by type",
		},
		[]string{"failure_type"},
	)

	// Initialize manual intervention counter
	pm.manualInterventionsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "manual_interventions_total",
			Help:      "Total number of manual intervention requests",
		},
	)

	// Initialize recovery success rate summary
	pm.recoverySuccessRate = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "success_rate",
			Help:      "Recovery success rate (0-1)",
			Objectives: map[float64]float64{
				0.5:  0.05,
				0.9:  0.01,
				0.99: 0.001,
			},
		},
	)

	// Register all metrics
	metrics := []prometheus.Collector{
		pm.recoveryAttemptsTotal,
		pm.recoverySuccessTotal,
		pm.recoveryFailureTotal,
		pm.recoveryDuration,
		pm.recoveringInProgress,
		pm.detectedFailuresTotal,
		pm.manualInterventionsTotal,
		pm.recoverySuccessRate,
	}

	for _, metric := range metrics {
		if err := config.Registry.Register(metric); err != nil {
			return nil, err
		}
	}

	return &RecoveryMetrics{
		prometheus:           pm,
		recoveryDurations:    make([]time.Duration, 0, 1000),
		maxRecordedDurations: 1000, // Keep last 1000 durations for statistics
	}, nil
}

// RecordRecoveryAttempt records a recovery attempt.
func (rm *RecoveryMetrics) RecordRecoveryAttempt(strategy, sagaType string) {
	rm.mu.Lock()
	rm.totalAttempts++
	rm.mu.Unlock()

	if rm.prometheus != nil {
		rm.prometheus.recoveryAttemptsTotal.WithLabelValues(strategy, sagaType).Inc()
	}
}

// RecordRecoverySuccess records a successful recovery.
func (rm *RecoveryMetrics) RecordRecoverySuccess(strategy, sagaType string, duration time.Duration) {
	rm.mu.Lock()
	rm.successfulRecoveries++
	rm.recordDuration(duration)

	// Calculate success rate while holding the lock
	var successRate float64
	total := float64(rm.totalAttempts)
	if total > 0 {
		successRate = float64(rm.successfulRecoveries) / total
	}
	rm.mu.Unlock()

	if rm.prometheus != nil {
		rm.prometheus.recoverySuccessTotal.WithLabelValues(strategy, sagaType).Inc()
		rm.prometheus.recoveryDuration.WithLabelValues(strategy, sagaType).Observe(duration.Seconds())

		// Record success rate
		if total > 0 {
			rm.prometheus.recoverySuccessRate.Observe(successRate)
		}
	}
}

// RecordRecoveryFailure records a failed recovery.
func (rm *RecoveryMetrics) RecordRecoveryFailure(strategy, sagaType, errorType string, duration time.Duration) {
	rm.mu.Lock()
	rm.failedRecoveries++
	rm.recordDuration(duration)

	// Calculate success rate while holding the lock
	var successRate float64
	total := float64(rm.totalAttempts)
	if total > 0 {
		successRate = float64(rm.successfulRecoveries) / total
	}
	rm.mu.Unlock()

	if rm.prometheus != nil {
		rm.prometheus.recoveryFailureTotal.WithLabelValues(strategy, sagaType, errorType).Inc()
		rm.prometheus.recoveryDuration.WithLabelValues(strategy, sagaType).Observe(duration.Seconds())

		// Record success rate
		if total > 0 {
			rm.prometheus.recoverySuccessRate.Observe(successRate)
		}
	}
}

// IncrementRecoveringInProgress increments the count of currently recovering Sagas.
func (rm *RecoveryMetrics) IncrementRecoveringInProgress(strategy string) {
	rm.mu.Lock()
	rm.currentlyRecovering++
	rm.mu.Unlock()

	if rm.prometheus != nil {
		rm.prometheus.recoveringInProgress.WithLabelValues(strategy).Inc()
	}
}

// DecrementRecoveringInProgress decrements the count of currently recovering Sagas.
func (rm *RecoveryMetrics) DecrementRecoveringInProgress(strategy string) {
	rm.mu.Lock()
	if rm.currentlyRecovering > 0 {
		rm.currentlyRecovering--
	}
	rm.mu.Unlock()

	if rm.prometheus != nil {
		rm.prometheus.recoveringInProgress.WithLabelValues(strategy).Dec()
	}
}

// RecordDetectedFailure records a detected failure by type.
func (rm *RecoveryMetrics) RecordDetectedFailure(failureType string) {
	rm.mu.Lock()
	switch failureType {
	case "timeout":
		rm.detectedTimeouts++
	case "stuck":
		rm.detectedStuck++
	case "compensating":
		rm.detectedCompensating++
	case "inconsistent":
		rm.detectedInconsistent++
	}
	rm.mu.Unlock()

	if rm.prometheus != nil {
		rm.prometheus.detectedFailuresTotal.WithLabelValues(failureType).Inc()
	}
}

// RecordManualIntervention records a manual intervention request.
func (rm *RecoveryMetrics) RecordManualIntervention() {
	rm.mu.Lock()
	rm.manualInterventions++
	rm.mu.Unlock()

	if rm.prometheus != nil {
		rm.prometheus.manualInterventionsTotal.Inc()
	}
}

// GetSnapshot returns a snapshot of current metrics.
func (rm *RecoveryMetrics) GetSnapshot() RecoveryMetricsSnapshot {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	snapshot := RecoveryMetricsSnapshot{
		TotalAttempts:        rm.totalAttempts,
		SuccessfulRecoveries: rm.successfulRecoveries,
		FailedRecoveries:     rm.failedRecoveries,
		CurrentlyRecovering:  rm.currentlyRecovering,
		ManualInterventions:  rm.manualInterventions,
		DetectedTimeouts:     rm.detectedTimeouts,
		DetectedStuck:        rm.detectedStuck,
		DetectedCompensating: rm.detectedCompensating,
		DetectedInconsistent: rm.detectedInconsistent,
	}

	// Calculate statistics from recorded durations
	if len(rm.recoveryDurations) > 0 {
		snapshot.AverageDuration = rm.calculateAverage()
		snapshot.P50Duration = rm.calculatePercentile(0.5)
		snapshot.P95Duration = rm.calculatePercentile(0.95)
		snapshot.P99Duration = rm.calculatePercentile(0.99)
	}

	// Calculate success rate
	if rm.totalAttempts > 0 {
		snapshot.SuccessRate = float64(rm.successfulRecoveries) / float64(rm.totalAttempts)
	}

	return snapshot
}

// RecoveryMetricsSnapshot represents a point-in-time snapshot of recovery metrics.
type RecoveryMetricsSnapshot struct {
	TotalAttempts        int64         `json:"total_attempts"`
	SuccessfulRecoveries int64         `json:"successful_recoveries"`
	FailedRecoveries     int64         `json:"failed_recoveries"`
	CurrentlyRecovering  int64         `json:"currently_recovering"`
	ManualInterventions  int64         `json:"manual_interventions"`
	DetectedTimeouts     int64         `json:"detected_timeouts"`
	DetectedStuck        int64         `json:"detected_stuck"`
	DetectedCompensating int64         `json:"detected_compensating"`
	DetectedInconsistent int64         `json:"detected_inconsistent"`
	AverageDuration      time.Duration `json:"average_duration"`
	P50Duration          time.Duration `json:"p50_duration"`
	P95Duration          time.Duration `json:"p95_duration"`
	P99Duration          time.Duration `json:"p99_duration"`
	SuccessRate          float64       `json:"success_rate"`
}

// GetRegistry returns the Prometheus registry for external usage.
func (rm *RecoveryMetrics) GetRegistry() *prometheus.Registry {
	if rm.prometheus != nil {
		return rm.prometheus.registry
	}
	return nil
}

// recordDuration records a recovery duration (internal method, assumes lock is held).
func (rm *RecoveryMetrics) recordDuration(duration time.Duration) {
	rm.recoveryDurations = append(rm.recoveryDurations, duration)

	// Trim if exceeds max
	if len(rm.recoveryDurations) > rm.maxRecordedDurations {
		// Keep most recent durations
		rm.recoveryDurations = rm.recoveryDurations[len(rm.recoveryDurations)-rm.maxRecordedDurations:]
	}
}

// calculateAverage calculates the average duration (assumes lock is held).
func (rm *RecoveryMetrics) calculateAverage() time.Duration {
	if len(rm.recoveryDurations) == 0 {
		return 0
	}

	var sum time.Duration
	for _, d := range rm.recoveryDurations {
		sum += d
	}

	return sum / time.Duration(len(rm.recoveryDurations))
}

// calculatePercentile calculates a percentile duration (assumes lock is held).
func (rm *RecoveryMetrics) calculatePercentile(percentile float64) time.Duration {
	if len(rm.recoveryDurations) == 0 {
		return 0
	}

	// Create a sorted copy
	sorted := make([]time.Duration, len(rm.recoveryDurations))
	copy(sorted, rm.recoveryDurations)

	// Simple bubble sort for small arrays
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate index
	idx := int(float64(len(sorted)-1) * percentile)
	return sorted[idx]
}

// Reset clears all metrics.
func (rm *RecoveryMetrics) Reset() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.totalAttempts = 0
	rm.successfulRecoveries = 0
	rm.failedRecoveries = 0
	rm.currentlyRecovering = 0
	rm.manualInterventions = 0
	rm.detectedTimeouts = 0
	rm.detectedStuck = 0
	rm.detectedCompensating = 0
	rm.detectedInconsistent = 0
	rm.recoveryDurations = make([]time.Duration, 0, rm.maxRecordedDurations)

	// Note: Prometheus metrics cannot be reset, they are cumulative
}
