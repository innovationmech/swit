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
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// PoolStats represents statistics about a Redis connection pool.
type PoolStats struct {
	// TotalConns is the total number of connections in the pool.
	TotalConns uint32 `json:"total_conns"`

	// IdleConns is the number of idle connections in the pool.
	IdleConns uint32 `json:"idle_conns"`

	// StaleConns is the number of stale connections removed from the pool.
	StaleConns uint32 `json:"stale_conns"`

	// Hits is the number of times a free connection was found in the pool.
	Hits uint64 `json:"hits"`

	// Misses is the number of times a free connection was not found.
	Misses uint64 `json:"misses"`

	// Timeouts is the number of times a wait timeout occurred.
	Timeouts uint64 `json:"timeouts"`

	// PoolSize is the maximum number of connections.
	PoolSize int `json:"pool_size"`

	// MinIdleConns is the minimum number of idle connections.
	MinIdleConns int `json:"min_idle_conns"`

	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int `json:"max_idle_conns"`

	// Timestamp is when these stats were collected.
	Timestamp time.Time `json:"timestamp"`
}

// PoolHealthStatus represents the overall health status of the connection pool.
type PoolHealthStatus struct {
	// Healthy indicates if the pool is healthy.
	Healthy bool `json:"healthy"`

	// Message provides details about the health status.
	Message string `json:"message"`

	// Stats contains the current pool statistics.
	Stats *PoolStats `json:"stats"`

	// Warnings contains any warnings about pool health.
	Warnings []string `json:"warnings,omitempty"`
}

// PoolMonitor monitors Redis connection pool statistics and health.
type PoolMonitor struct {
	client RedisClient
	config *RedisConfig

	// mu protects the stats and metrics
	mu sync.RWMutex

	// lastStats stores the last collected statistics
	lastStats *PoolStats

	// metricsEnabled indicates if metrics collection is enabled
	metricsEnabled bool

	// healthThresholds contains thresholds for health checks
	healthThresholds *PoolHealthThresholds
}

// PoolHealthThresholds defines thresholds for pool health checks.
type PoolHealthThresholds struct {
	// MaxUtilization is the maximum acceptable pool utilization (0.0-1.0).
	// If utilization exceeds this, a warning is issued.
	// Default: 0.9 (90%)
	MaxUtilization float64

	// MinIdleRatio is the minimum ratio of idle connections to total connections.
	// If the ratio falls below this, a warning is issued.
	// Default: 0.1 (10%)
	MinIdleRatio float64

	// MaxMissRate is the maximum acceptable miss rate (misses / (hits + misses)).
	// If the miss rate exceeds this, a warning is issued.
	// Default: 0.2 (20%)
	MaxMissRate float64

	// MaxTimeoutRate is the maximum acceptable timeout rate.
	// Default: 0.05 (5%)
	MaxTimeoutRate float64
}

// DefaultPoolHealthThresholds returns default health thresholds.
func DefaultPoolHealthThresholds() *PoolHealthThresholds {
	return &PoolHealthThresholds{
		MaxUtilization: 0.9,
		MinIdleRatio:   0.1,
		MaxMissRate:    0.2,
		MaxTimeoutRate: 0.05,
	}
}

// NewPoolMonitor creates a new pool monitor for the given Redis client.
func NewPoolMonitor(client RedisClient, config *RedisConfig) *PoolMonitor {
	return &PoolMonitor{
		client:           client,
		config:           config,
		metricsEnabled:   config.EnableMetrics,
		healthThresholds: DefaultPoolHealthThresholds(),
	}
}

// CollectStats collects current pool statistics from the Redis client.
func (m *PoolMonitor) CollectStats(ctx context.Context) (*PoolStats, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var redisStats *redis.PoolStats

	// Get pool stats based on client type
	switch client := m.client.(type) {
	case *redis.Client:
		redisStats = client.PoolStats()
	case *redis.ClusterClient:
		// For cluster client, we get stats from the first node
		// This is a simplified approach; in production you might want to aggregate
		redisStats = client.PoolStats()
	case *redis.Ring:
		redisStats = client.PoolStats()
	default:
		return nil, fmt.Errorf("unsupported client type for pool stats")
	}

	stats := &PoolStats{
		TotalConns:   redisStats.TotalConns,
		IdleConns:    redisStats.IdleConns,
		StaleConns:   redisStats.StaleConns,
		Hits:         uint64(redisStats.Hits),
		Misses:       uint64(redisStats.Misses),
		Timeouts:     uint64(redisStats.Timeouts),
		PoolSize:     m.config.PoolSize,
		MinIdleConns: m.config.MinIdleConns,
		MaxIdleConns: m.config.MaxIdleConns,
		Timestamp:    time.Now(),
	}

	// Store the stats
	m.mu.Lock()
	m.lastStats = stats
	m.mu.Unlock()

	return stats, nil
}

// GetLastStats returns the last collected pool statistics.
func (m *PoolMonitor) GetLastStats() *PoolStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.lastStats == nil {
		return nil
	}

	// Return a copy to prevent external modification
	statsCopy := *m.lastStats
	return &statsCopy
}

// CheckHealth performs a health check on the connection pool.
func (m *PoolMonitor) CheckHealth(ctx context.Context) (*PoolHealthStatus, error) {
	// Collect current stats
	stats, err := m.CollectStats(ctx)
	if err != nil {
		return &PoolHealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("Failed to collect pool stats: %v", err),
			Stats:   nil,
		}, nil
	}

	var warnings []string
	healthy := true

	// Check pool utilization
	if stats.PoolSize > 0 {
		utilization := float64(stats.TotalConns) / float64(stats.PoolSize)
		if utilization > m.healthThresholds.MaxUtilization {
			warnings = append(warnings, fmt.Sprintf(
				"High pool utilization: %.2f%% (threshold: %.2f%%)",
				utilization*100, m.healthThresholds.MaxUtilization*100,
			))
			healthy = false
		}
	}

	// Check idle connection ratio
	if stats.TotalConns > 0 {
		idleRatio := float64(stats.IdleConns) / float64(stats.TotalConns)
		if idleRatio < m.healthThresholds.MinIdleRatio {
			warnings = append(warnings, fmt.Sprintf(
				"Low idle connection ratio: %.2f%% (threshold: %.2f%%)",
				idleRatio*100, m.healthThresholds.MinIdleRatio*100,
			))
		}
	}

	// Check miss rate
	totalRequests := stats.Hits + stats.Misses
	if totalRequests > 0 {
		missRate := float64(stats.Misses) / float64(totalRequests)
		if missRate > m.healthThresholds.MaxMissRate {
			warnings = append(warnings, fmt.Sprintf(
				"High miss rate: %.2f%% (threshold: %.2f%%)",
				missRate*100, m.healthThresholds.MaxMissRate*100,
			))
		}
	}

	// Check timeout rate
	if totalRequests > 0 {
		timeoutRate := float64(stats.Timeouts) / float64(totalRequests)
		if timeoutRate > m.healthThresholds.MaxTimeoutRate {
			warnings = append(warnings, fmt.Sprintf(
				"High timeout rate: %.2f%% (threshold: %.2f%%)",
				timeoutRate*100, m.healthThresholds.MaxTimeoutRate*100,
			))
			healthy = false
		}
	}

	message := "Pool is healthy"
	if !healthy {
		message = "Pool has health issues"
	} else if len(warnings) > 0 {
		message = "Pool is healthy with warnings"
	}

	return &PoolHealthStatus{
		Healthy:  healthy,
		Message:  message,
		Stats:    stats,
		Warnings: warnings,
	}, nil
}

// SetHealthThresholds updates the health check thresholds.
func (m *PoolMonitor) SetHealthThresholds(thresholds *PoolHealthThresholds) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthThresholds = thresholds
}

// GetHealthThresholds returns the current health check thresholds.
func (m *PoolMonitor) GetHealthThresholds() *PoolHealthThresholds {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	thresholdsCopy := *m.healthThresholds
	return &thresholdsCopy
}

// StartMonitoring starts periodic pool monitoring.
// It collects stats at the specified interval until the context is cancelled.
func (m *PoolMonitor) StartMonitoring(ctx context.Context, interval time.Duration) {
	if !m.metricsEnabled {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Collect stats (ignore errors during monitoring)
			_, _ = m.CollectStats(ctx)
		}
	}
}

// GetUtilizationMetrics returns utilization metrics for the pool.
func (m *PoolMonitor) GetUtilizationMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.lastStats == nil {
		return nil
	}

	stats := m.lastStats
	metrics := make(map[string]interface{})

	// Calculate utilization
	if stats.PoolSize > 0 {
		metrics["utilization_percent"] = (float64(stats.TotalConns) / float64(stats.PoolSize)) * 100
	}

	// Calculate idle ratio
	if stats.TotalConns > 0 {
		metrics["idle_ratio"] = float64(stats.IdleConns) / float64(stats.TotalConns)
	}

	// Calculate hit rate
	totalRequests := stats.Hits + stats.Misses
	if totalRequests > 0 {
		metrics["hit_rate"] = float64(stats.Hits) / float64(totalRequests)
		metrics["miss_rate"] = float64(stats.Misses) / float64(totalRequests)
	}

	// Calculate timeout rate
	if totalRequests > 0 {
		metrics["timeout_rate"] = float64(stats.Timeouts) / float64(totalRequests)
	}

	metrics["total_conns"] = stats.TotalConns
	metrics["idle_conns"] = stats.IdleConns
	metrics["stale_conns"] = stats.StaleConns
	metrics["hits"] = stats.Hits
	metrics["misses"] = stats.Misses
	metrics["timeouts"] = stats.Timeouts

	return metrics
}
