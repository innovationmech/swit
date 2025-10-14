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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

var (
	// ErrRedisUnhealthy indicates that Redis is not healthy.
	ErrRedisUnhealthy = errors.New("redis is unhealthy")

	// ErrHealthCheckTimeout indicates that the health check timed out.
	ErrHealthCheckTimeout = errors.New("health check timeout")

	// ErrSlowQuery indicates that a query exceeded the slow query threshold.
	ErrSlowQuery = errors.New("slow query detected")
)

// HealthCheckConfig contains configuration for Redis health checks.
type HealthCheckConfig struct {
	// Enabled indicates whether health checks are enabled.
	// Default: true
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Timeout is the maximum duration for a health check.
	// Default: 3 seconds
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// Interval is the interval between periodic health checks.
	// Default: 30 seconds
	Interval time.Duration `json:"interval" yaml:"interval"`

	// PingEnabled indicates whether to perform ping checks.
	// Default: true
	PingEnabled bool `json:"ping_enabled" yaml:"ping_enabled"`

	// InfoEnabled indicates whether to fetch Redis info for health checks.
	// Default: true
	InfoEnabled bool `json:"info_enabled" yaml:"info_enabled"`

	// SlowQueryThreshold is the threshold for slow query logging.
	// Queries taking longer than this duration will be logged.
	// Default: 100ms
	SlowQueryThreshold time.Duration `json:"slow_query_threshold" yaml:"slow_query_threshold"`

	// MaxMemoryWarningPercent is the memory usage percentage that triggers a warning.
	// Default: 90
	MaxMemoryWarningPercent int `json:"max_memory_warning_percent" yaml:"max_memory_warning_percent"`

	// MaxConnectionWarningPercent is the connection usage percentage that triggers a warning.
	// Default: 90
	MaxConnectionWarningPercent int `json:"max_connection_warning_percent" yaml:"max_connection_warning_percent"`
}

// DefaultHealthCheckConfig returns a HealthCheckConfig with default values.
func DefaultHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		Enabled:                     true,
		Timeout:                     3 * time.Second,
		Interval:                    30 * time.Second,
		PingEnabled:                 true,
		InfoEnabled:                 true,
		SlowQueryThreshold:          100 * time.Millisecond,
		MaxMemoryWarningPercent:     90,
		MaxConnectionWarningPercent: 90,
	}
}

// HealthStatus represents the health status of Redis storage.
type HealthStatus struct {
	// Healthy indicates whether the storage is healthy.
	Healthy bool `json:"healthy"`

	// Message provides additional information about the health status.
	Message string `json:"message,omitempty"`

	// Error contains the error message if unhealthy.
	Error string `json:"error,omitempty"`

	// Timestamp is when the health check was performed.
	Timestamp time.Time `json:"timestamp"`

	// Duration is how long the health check took.
	Duration time.Duration `json:"duration"`

	// Details contains detailed health information.
	Details *HealthDetails `json:"details,omitempty"`
}

// HealthDetails contains detailed health information.
type HealthDetails struct {
	// PingLatency is the latency of the ping command.
	PingLatency time.Duration `json:"ping_latency,omitempty"`

	// ConnectedClients is the number of connected clients.
	ConnectedClients int64 `json:"connected_clients,omitempty"`

	// UsedMemory is the memory used by Redis in bytes.
	UsedMemory int64 `json:"used_memory,omitempty"`

	// MaxMemory is the maximum memory configured for Redis in bytes.
	MaxMemory int64 `json:"max_memory,omitempty"`

	// MemoryUsagePercent is the percentage of memory used.
	MemoryUsagePercent float64 `json:"memory_usage_percent,omitempty"`

	// KeyspaceHits is the number of successful key lookups.
	KeyspaceHits int64 `json:"keyspace_hits,omitempty"`

	// KeyspaceMisses is the number of failed key lookups.
	KeyspaceMisses int64 `json:"keyspace_misses,omitempty"`

	// HitRate is the cache hit rate percentage.
	HitRate float64 `json:"hit_rate,omitempty"`

	// UptimeSeconds is Redis server uptime in seconds.
	UptimeSeconds int64 `json:"uptime_seconds,omitempty"`

	// Version is the Redis server version.
	Version string `json:"version,omitempty"`

	// Mode is the Redis mode (standalone, cluster, sentinel).
	Mode string `json:"mode,omitempty"`

	// Warnings contains any warnings detected during the health check.
	Warnings []string `json:"warnings,omitempty"`
}

// RedisHealthChecker performs health checks on Redis storage.
type RedisHealthChecker struct {
	storage *RedisStateStorage
	config  *HealthCheckConfig

	mu           sync.RWMutex
	lastStatus   *HealthStatus
	lastCheck    time.Time
	checkRunning bool

	// stopCh is used to stop the periodic health check.
	stopCh chan struct{}
	stopMu sync.Mutex
}

// NewRedisHealthChecker creates a new Redis health checker.
func NewRedisHealthChecker(storage *RedisStateStorage, config *HealthCheckConfig) *RedisHealthChecker {
	if config == nil {
		config = DefaultHealthCheckConfig()
	}

	return &RedisHealthChecker{
		storage: storage,
		config:  config,
		stopCh:  make(chan struct{}),
	}
}

// Check performs a health check on the Redis storage.
// It implements the HealthChecker interface from pkg/server/health.
func (h *RedisHealthChecker) Check(ctx context.Context) error {
	status := h.CheckHealth(ctx)
	if !status.Healthy {
		return fmt.Errorf("%w: %s", ErrRedisUnhealthy, status.Error)
	}
	return nil
}

// GetName returns the name of the health checker.
// It implements the HealthChecker interface from pkg/server/health.
func (h *RedisHealthChecker) GetName() string {
	return "redis-saga-storage"
}

// CheckReadiness performs a readiness check on the Redis storage.
// It implements the ReadinessChecker interface from pkg/server/health.
func (h *RedisHealthChecker) CheckReadiness(ctx context.Context) error {
	return h.Check(ctx)
}

// CheckLiveness performs a liveness check on the Redis storage.
// It implements the LivenessChecker interface from pkg/server/health.
func (h *RedisHealthChecker) CheckLiveness(ctx context.Context) error {
	return h.Check(ctx)
}

// CheckHealth performs a comprehensive health check on Redis storage.
func (h *RedisHealthChecker) CheckHealth(ctx context.Context) *HealthStatus {
	if !h.config.Enabled {
		return &HealthStatus{
			Healthy:   true,
			Message:   "health checks disabled",
			Timestamp: time.Now(),
		}
	}

	// Prevent concurrent health checks
	h.mu.Lock()
	if h.checkRunning {
		h.mu.Unlock()
		// Return last status if a check is already running
		return h.GetLastStatus()
	}
	h.checkRunning = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		h.checkRunning = false
		h.mu.Unlock()
	}()

	start := time.Now()
	status := &HealthStatus{
		Healthy:   true,
		Timestamp: start,
		Details:   &HealthDetails{},
	}

	// Create a timeout context
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	// Perform ping check
	if h.config.PingEnabled {
		pingStart := time.Now()
		if err := h.storage.client.Ping(checkCtx).Err(); err != nil {
			status.Healthy = false
			status.Error = fmt.Sprintf("ping failed: %v", err)
			status.Duration = time.Since(start)
			h.updateLastStatus(status)
			return status
		}
		status.Details.PingLatency = time.Since(pingStart)
	}

	// Fetch Redis info if enabled
	if h.config.InfoEnabled {
		if err := h.fetchRedisInfo(checkCtx, status.Details); err != nil {
			logger.Logger.Warn("Failed to fetch Redis info during health check",
				zap.Error(err))
			// Don't fail the health check, just log the warning
		}
	}

	// Check for warnings
	h.checkWarnings(status.Details)

	status.Duration = time.Since(start)
	status.Message = "redis is healthy"

	// Log slow health checks
	if status.Duration > h.config.SlowQueryThreshold {
		logger.Logger.Warn("Slow health check detected",
			zap.Duration("duration", status.Duration),
			zap.Duration("threshold", h.config.SlowQueryThreshold))
	}

	h.updateLastStatus(status)
	return status
}

// fetchRedisInfo fetches Redis server information and populates the health details.
func (h *RedisHealthChecker) fetchRedisInfo(ctx context.Context, details *HealthDetails) error {
	// Get Redis INFO command output
	info, err := h.storage.client.Info(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to get redis info: %w", err)
	}

	// Parse relevant information from the INFO output
	// INFO output format: key:value\r\n
	lines := parseRedisInfo(info)

	// Extract server information
	if version, ok := lines["redis_version"]; ok {
		details.Version = version
	}
	if mode, ok := lines["redis_mode"]; ok {
		details.Mode = mode
	}
	if uptime, ok := parseRedisInfoInt(lines, "uptime_in_seconds"); ok {
		details.UptimeSeconds = uptime
	}

	// Extract memory information
	if usedMemory, ok := parseRedisInfoInt(lines, "used_memory"); ok {
		details.UsedMemory = usedMemory
	}
	if maxMemory, ok := parseRedisInfoInt(lines, "maxmemory"); ok {
		details.MaxMemory = maxMemory
		if maxMemory > 0 {
			details.MemoryUsagePercent = float64(details.UsedMemory) / float64(maxMemory) * 100
		}
	}

	// Extract client information
	if connectedClients, ok := parseRedisInfoInt(lines, "connected_clients"); ok {
		details.ConnectedClients = connectedClients
	}

	// Extract stats information
	if hits, ok := parseRedisInfoInt(lines, "keyspace_hits"); ok {
		details.KeyspaceHits = hits
	}
	if misses, ok := parseRedisInfoInt(lines, "keyspace_misses"); ok {
		details.KeyspaceMisses = misses
	}

	// Calculate hit rate
	totalOps := details.KeyspaceHits + details.KeyspaceMisses
	if totalOps > 0 {
		details.HitRate = float64(details.KeyspaceHits) / float64(totalOps) * 100
	}

	return nil
}

// checkWarnings checks for warning conditions in the health details.
func (h *RedisHealthChecker) checkWarnings(details *HealthDetails) {
	if details.MaxMemory > 0 && details.MemoryUsagePercent >= float64(h.config.MaxMemoryWarningPercent) {
		warning := fmt.Sprintf("memory usage is high: %.2f%%", details.MemoryUsagePercent)
		details.Warnings = append(details.Warnings, warning)
	}

	// Additional warning checks can be added here
	if details.HitRate < 50 && (details.KeyspaceHits+details.KeyspaceMisses) > 1000 {
		warning := fmt.Sprintf("low cache hit rate: %.2f%%", details.HitRate)
		details.Warnings = append(details.Warnings, warning)
	}
}

// GetLastStatus returns the last health check status.
func (h *RedisHealthChecker) GetLastStatus() *HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.lastStatus == nil {
		return &HealthStatus{
			Healthy:   true,
			Message:   "no health check performed yet",
			Timestamp: time.Now(),
		}
	}

	// Return a copy to prevent external modifications
	statusCopy := *h.lastStatus
	if h.lastStatus.Details != nil {
		detailsCopy := *h.lastStatus.Details
		statusCopy.Details = &detailsCopy
	}

	return &statusCopy
}

// updateLastStatus updates the last health check status.
func (h *RedisHealthChecker) updateLastStatus(status *HealthStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.lastStatus = status
	h.lastCheck = time.Now()
}

// StartPeriodicCheck starts periodic health checks in the background.
// It returns a channel that will be closed when the periodic check stops.
func (h *RedisHealthChecker) StartPeriodicCheck(ctx context.Context) <-chan struct{} {
	if !h.config.Enabled || h.config.Interval <= 0 {
		doneCh := make(chan struct{})
		close(doneCh)
		return doneCh
	}

	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)

		ticker := time.NewTicker(h.config.Interval)
		defer ticker.Stop()

		logger.Logger.Info("Started periodic health checks for Redis saga storage",
			zap.Duration("interval", h.config.Interval))

		for {
			select {
			case <-ctx.Done():
				logger.Logger.Info("Stopped periodic health checks for Redis saga storage: context cancelled")
				return
			case <-h.stopCh:
				logger.Logger.Info("Stopped periodic health checks for Redis saga storage")
				return
			case <-ticker.C:
				status := h.CheckHealth(ctx)
				if !status.Healthy {
					logger.Logger.Error("Redis health check failed",
						zap.String("error", status.Error),
						zap.Duration("duration", status.Duration))
				} else if len(status.Details.Warnings) > 0 {
					logger.Logger.Warn("Redis health check passed with warnings",
						zap.Strings("warnings", status.Details.Warnings),
						zap.Duration("duration", status.Duration))
				} else {
					logger.Logger.Debug("Redis health check passed",
						zap.Duration("duration", status.Duration),
						zap.Duration("ping_latency", status.Details.PingLatency))
				}
			}
		}
	}()

	return doneCh
}

// StopPeriodicCheck stops the periodic health check.
func (h *RedisHealthChecker) StopPeriodicCheck() {
	h.stopMu.Lock()
	defer h.stopMu.Unlock()

	select {
	case <-h.stopCh:
		// Already stopped
	default:
		close(h.stopCh)
	}
}

// parseRedisInfo parses Redis INFO command output into a key-value map.
func parseRedisInfo(info string) map[string]string {
	result := make(map[string]string)
	lines := []byte(info)

	var key, value []byte
	inKey := true

	for i := 0; i < len(lines); i++ {
		switch lines[i] {
		case ':':
			if inKey {
				inKey = false
			}
		case '\r':
			if i+1 < len(lines) && lines[i+1] == '\n' {
				if len(key) > 0 && len(value) > 0 {
					result[string(key)] = string(value)
				}
				key = nil
				value = nil
				inKey = true
				i++ // Skip \n
			}
		case '\n':
			if len(key) > 0 && len(value) > 0 {
				result[string(key)] = string(value)
			}
			key = nil
			value = nil
			inKey = true
		case '#':
			// Skip comment lines
			for i < len(lines) && lines[i] != '\r' && lines[i] != '\n' {
				i++
			}
			if i < len(lines) && lines[i] == '\r' {
				i++ // Skip \r
			}
		default:
			if inKey {
				key = append(key, lines[i])
			} else {
				value = append(value, lines[i])
			}
		}
	}

	return result
}

// parseRedisInfoInt parses an integer value from Redis INFO output.
func parseRedisInfoInt(info map[string]string, key string) (int64, bool) {
	value, ok := info[key]
	if !ok {
		return 0, false
	}

	var result int64
	for _, ch := range value {
		if ch >= '0' && ch <= '9' {
			result = result*10 + int64(ch-'0')
		} else {
			break
		}
	}

	return result, true
}
