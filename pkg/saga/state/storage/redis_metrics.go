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

package storage

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// RedisStorageMetrics holds Prometheus metrics for Redis saga storage operations.
type RedisStorageMetrics struct {
	// Operation counters
	operationsTotal *prometheus.CounterVec
	errorTotal      *prometheus.CounterVec

	// Operation duration histograms
	operationDuration *prometheus.HistogramVec

	// Connection pool metrics
	poolActiveConns  prometheus.Gauge
	poolIdleConns    prometheus.Gauge
	poolWaitDuration prometheus.Histogram
	poolMaxOpenConns prometheus.Gauge
	poolMaxIdleConns prometheus.Gauge

	// Redis info metrics
	redisConnectedClients   prometheus.Gauge
	redisUsedMemory         prometheus.Gauge
	redisMaxMemory          prometheus.Gauge
	redisMemoryUsagePercent prometheus.Gauge
	redisKeyspaceHits       prometheus.Counter
	redisKeyspaceMisses     prometheus.Counter
	redisHitRate            prometheus.Gauge
	redisUptimeSeconds      prometheus.Gauge

	// Health check metrics
	healthCheckTotal    *prometheus.CounterVec
	healthCheckDuration prometheus.Histogram
	healthCheckStatus   prometheus.Gauge

	// Slow query metrics
	slowQueriesTotal *prometheus.CounterVec

	// Saga-specific metrics
	sagasTotal     *prometheus.CounterVec
	sagaStepsTotal *prometheus.CounterVec

	// Prometheus registry
	registry *prometheus.Registry

	// Metrics collection state
	mu                sync.RWMutex
	metricsEnabled    bool
	lastInfoUpdate    time.Time
	infoUpdateRunning bool
}

// RedisMetricsConfig contains configuration for Redis metrics collection.
type RedisMetricsConfig struct {
	// Namespace is the Prometheus namespace for all metrics.
	// Default: "saga"
	Namespace string

	// Subsystem is the Prometheus subsystem for all metrics.
	// Default: "redis_storage"
	Subsystem string

	// Registry is the Prometheus registry to use.
	// If nil, the default registry is used.
	Registry *prometheus.Registry

	// DurationBuckets defines the buckets for operation duration histograms.
	// If nil, default buckets are used.
	DurationBuckets []float64

	// EnableDetailedMetrics enables collection of detailed Redis info metrics.
	// This requires periodic INFO commands to Redis, which may have performance impact.
	// Default: true
	EnableDetailedMetrics bool

	// InfoUpdateInterval is the interval for updating Redis info metrics.
	// Default: 30 seconds
	InfoUpdateInterval time.Duration
}

// DefaultRedisMetricsConfig returns a RedisMetricsConfig with default values.
func DefaultRedisMetricsConfig() *RedisMetricsConfig {
	return &RedisMetricsConfig{
		Namespace: "saga",
		Subsystem: "redis_storage",
		DurationBuckets: []float64{
			0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
		},
		EnableDetailedMetrics: true,
		InfoUpdateInterval:    30 * time.Second,
	}
}

// NewRedisStorageMetrics creates a new RedisStorageMetrics instance.
func NewRedisStorageMetrics(config *RedisMetricsConfig) *RedisStorageMetrics {
	if config == nil {
		config = DefaultRedisMetricsConfig()
	}

	// Use provided registry or default
	registry := config.Registry
	if registry == nil {
		registry = prometheus.DefaultRegisterer.(*prometheus.Registry)
	}

	factory := promauto.With(registry)

	metrics := &RedisStorageMetrics{
		registry:       registry,
		metricsEnabled: true,
	}

	// Initialize operation metrics
	metrics.operationsTotal = factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "operations_total",
			Help:      "Total number of Redis storage operations by type and status",
		},
		[]string{"operation", "status"},
	)

	metrics.errorTotal = factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "errors_total",
			Help:      "Total number of errors by operation and error type",
		},
		[]string{"operation", "error_type"},
	)

	metrics.operationDuration = factory.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "operation_duration_seconds",
			Help:      "Duration of Redis storage operations in seconds",
			Buckets:   config.DurationBuckets,
		},
		[]string{"operation"},
	)

	// Initialize connection pool metrics
	metrics.poolActiveConns = factory.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "pool_active_connections",
			Help:      "Number of active connections in the pool",
		},
	)

	metrics.poolIdleConns = factory.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "pool_idle_connections",
			Help:      "Number of idle connections in the pool",
		},
	)

	metrics.poolWaitDuration = factory.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "pool_wait_duration_seconds",
			Help:      "Time spent waiting for a connection from the pool",
			Buckets:   config.DurationBuckets,
		},
	)

	metrics.poolMaxOpenConns = factory.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "pool_max_open_connections",
			Help:      "Maximum number of open connections",
		},
	)

	metrics.poolMaxIdleConns = factory.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "pool_max_idle_connections",
			Help:      "Maximum number of idle connections",
		},
	)

	// Initialize Redis info metrics
	if config.EnableDetailedMetrics {
		metrics.redisConnectedClients = factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "redis_connected_clients",
				Help:      "Number of client connections to Redis",
			},
		)

		metrics.redisUsedMemory = factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "redis_used_memory_bytes",
				Help:      "Memory used by Redis in bytes",
			},
		)

		metrics.redisMaxMemory = factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "redis_max_memory_bytes",
				Help:      "Maximum memory configured for Redis in bytes",
			},
		)

		metrics.redisMemoryUsagePercent = factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "redis_memory_usage_percent",
				Help:      "Percentage of memory used by Redis",
			},
		)

		metrics.redisKeyspaceHits = factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "redis_keyspace_hits_total",
				Help:      "Total number of successful key lookups",
			},
		)

		metrics.redisKeyspaceMisses = factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "redis_keyspace_misses_total",
				Help:      "Total number of failed key lookups",
			},
		)

		metrics.redisHitRate = factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "redis_hit_rate_percent",
				Help:      "Cache hit rate percentage",
			},
		)

		metrics.redisUptimeSeconds = factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "redis_uptime_seconds",
				Help:      "Redis server uptime in seconds",
			},
		)
	}

	// Initialize health check metrics
	metrics.healthCheckTotal = factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "health_checks_total",
			Help:      "Total number of health checks by status",
		},
		[]string{"status"},
	)

	metrics.healthCheckDuration = factory.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "health_check_duration_seconds",
			Help:      "Duration of health checks in seconds",
			Buckets:   config.DurationBuckets,
		},
	)

	metrics.healthCheckStatus = factory.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "health_check_status",
			Help:      "Current health check status (1 = healthy, 0 = unhealthy)",
		},
	)

	// Initialize slow query metrics
	metrics.slowQueriesTotal = factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "slow_queries_total",
			Help:      "Total number of slow queries by operation",
		},
		[]string{"operation"},
	)

	// Initialize saga-specific metrics
	metrics.sagasTotal = factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "sagas_total",
			Help:      "Total number of saga operations by action",
		},
		[]string{"action"}, // save, get, update, delete
	)

	metrics.sagaStepsTotal = factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "saga_steps_total",
			Help:      "Total number of saga step operations by action",
		},
		[]string{"action"}, // save, get
	)

	return metrics
}

// RecordOperation records a Redis storage operation.
func (m *RedisStorageMetrics) RecordOperation(operation string, duration time.Duration, err error) {
	if !m.metricsEnabled {
		return
	}

	// Record operation count
	status := "success"
	if err != nil {
		status = "error"
		m.recordError(operation, err)
	}
	m.operationsTotal.WithLabelValues(operation, status).Inc()

	// Record operation duration
	m.operationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordSlowQuery records a slow query.
func (m *RedisStorageMetrics) RecordSlowQuery(operation string, duration time.Duration, threshold time.Duration) {
	if !m.metricsEnabled {
		return
	}

	if duration > threshold {
		m.slowQueriesTotal.WithLabelValues(operation).Inc()
	}
}

// RecordSagaOperation records a saga-level operation.
func (m *RedisStorageMetrics) RecordSagaOperation(action string) {
	if !m.metricsEnabled {
		return
	}
	m.sagasTotal.WithLabelValues(action).Inc()
}

// RecordStepOperation records a step-level operation.
func (m *RedisStorageMetrics) RecordStepOperation(action string) {
	if !m.metricsEnabled {
		return
	}
	m.sagaStepsTotal.WithLabelValues(action).Inc()
}

// RecordHealthCheck records a health check result.
func (m *RedisStorageMetrics) RecordHealthCheck(healthy bool, duration time.Duration) {
	if !m.metricsEnabled {
		return
	}

	status := "healthy"
	statusValue := 1.0
	if !healthy {
		status = "unhealthy"
		statusValue = 0.0
	}

	m.healthCheckTotal.WithLabelValues(status).Inc()
	m.healthCheckDuration.Observe(duration.Seconds())
	m.healthCheckStatus.Set(statusValue)
}

// UpdatePoolMetrics updates connection pool metrics.
func (m *RedisStorageMetrics) UpdatePoolMetrics(stats *PoolStats) {
	if !m.metricsEnabled || stats == nil {
		return
	}

	m.poolActiveConns.Set(float64(stats.TotalConns - stats.IdleConns))
	m.poolIdleConns.Set(float64(stats.IdleConns))
	m.poolMaxOpenConns.Set(float64(stats.TotalConns))
	m.poolMaxIdleConns.Set(float64(stats.IdleConns))
}

// UpdateRedisInfo updates Redis server information metrics.
func (m *RedisStorageMetrics) UpdateRedisInfo(details *HealthDetails) {
	if !m.metricsEnabled || details == nil {
		return
	}

	// Prevent concurrent updates
	m.mu.Lock()
	if m.infoUpdateRunning {
		m.mu.Unlock()
		return
	}
	m.infoUpdateRunning = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.infoUpdateRunning = false
		m.lastInfoUpdate = time.Now()
		m.mu.Unlock()
	}()

	if m.redisConnectedClients != nil {
		m.redisConnectedClients.Set(float64(details.ConnectedClients))
	}

	if m.redisUsedMemory != nil {
		m.redisUsedMemory.Set(float64(details.UsedMemory))
	}

	if m.redisMaxMemory != nil {
		m.redisMaxMemory.Set(float64(details.MaxMemory))
	}

	if m.redisMemoryUsagePercent != nil {
		m.redisMemoryUsagePercent.Set(details.MemoryUsagePercent)
	}

	// Note: KeyspaceHits and KeyspaceMisses are counters, so we need to track deltas
	// For simplicity, we set the hit rate gauge directly
	if m.redisHitRate != nil {
		m.redisHitRate.Set(details.HitRate)
	}

	if m.redisUptimeSeconds != nil {
		m.redisUptimeSeconds.Set(float64(details.UptimeSeconds))
	}
}

// recordError records an error by type.
func (m *RedisStorageMetrics) recordError(operation string, err error) {
	if err == nil {
		return
	}

	errorType := "unknown"
	switch {
	case err == context.Canceled:
		errorType = "canceled"
	case err == context.DeadlineExceeded:
		errorType = "timeout"
	case err.Error() == "redis: nil":
		errorType = "not_found"
	default:
		errorType = "internal"
	}

	m.errorTotal.WithLabelValues(operation, errorType).Inc()
}

// Disable disables metrics collection.
func (m *RedisStorageMetrics) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metricsEnabled = false
}

// Enable enables metrics collection.
func (m *RedisStorageMetrics) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metricsEnabled = true
}

// IsEnabled returns whether metrics collection is enabled.
func (m *RedisStorageMetrics) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metricsEnabled
}

// GetRegistry returns the Prometheus registry.
func (m *RedisStorageMetrics) GetRegistry() *prometheus.Registry {
	return m.registry
}
