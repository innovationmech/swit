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

package messaging

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// ConnectionManager defines the interface for managing broker connections.
// It provides connection pooling, health monitoring, and automatic reconnection
// with exponential backoff.
//
// The connection manager ensures efficient resource utilization while providing
// robust connection handling for messaging brokers. It supports:
// - Connection pooling with configurable min/max limits
// - Automatic reconnection with exponential backoff
// - Connection health validation and monitoring
// - Resource optimization through connection reuse
// - Connection statistics and monitoring metrics
// - Graceful shutdown with proper cleanup
type ConnectionManager interface {
	// Initialize prepares the connection manager with the provided configuration.
	// This method sets up the connection pool and starts background processes
	// for health monitoring and maintenance.
	//
	// Parameters:
	//   - ctx: Context for initialization timeout and cancellation
	//   - config: Connection configuration including pool settings and retry policies
	//   - brokerConfig: Broker configuration including endpoints and authentication
	//
	// Returns:
	//   - error: Configuration or initialization error, nil on success
	Initialize(ctx context.Context, config *ConnectionConfig, brokerConfig *BrokerConfig) error

	// GetConnection retrieves a healthy connection from the pool.
	// If no healthy connection is available, it may create a new one up to the pool limit.
	// This method blocks until a connection is available or the context is cancelled.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation control
	//
	// Returns:
	//   - Connection: A healthy connection ready for use
	//   - error: Connection retrieval error, nil on success
	GetConnection(ctx context.Context) (Connection, error)

	// ReturnConnection returns a connection to the pool after use.
	// The connection manager will validate the connection health before
	// returning it to the pool. Unhealthy connections are discarded.
	//
	// Parameters:
	//   - conn: Connection to return to the pool
	//
	// Returns:
	//   - error: Return operation error, nil on success
	ReturnConnection(conn Connection) error

	// GetStats returns current connection manager statistics.
	// This includes pool status, connection counts, and performance metrics.
	//
	// Returns:
	//   - *ConnectionStats: Current statistics snapshot
	GetStats() *ConnectionStats

	// HealthCheck performs a health check on all pooled connections.
	// Unhealthy connections are removed from the pool and closed.
	//
	// Parameters:
	//   - ctx: Context for health check timeout
	//
	// Returns:
	//   - *HealthStatus: Overall health status of the connection manager
	//   - error: Health check operation error, nil on success
	HealthCheck(ctx context.Context) (*HealthStatus, error)

	// Close gracefully shuts down the connection manager.
	// All connections in the pool are closed and resources are cleaned up.
	//
	// Returns:
	//   - error: Shutdown error, nil on success
	Close() error
}

// Connection represents a single broker connection with health monitoring capabilities.
// Connections are managed by the ConnectionManager and provide the actual
// communication channel to the message broker.
type Connection interface {
	// ID returns the unique identifier for this connection.
	//
	// Returns:
	//   - string: Unique connection identifier
	ID() string

	// IsHealthy checks if the connection is healthy and ready for use.
	// This method should be fast and non-blocking.
	//
	// Returns:
	//   - bool: true if connection is healthy, false otherwise
	IsHealthy() bool

	// Validate performs a comprehensive validation of the connection.
	// This may include sending a ping or performing a lightweight operation
	// to verify the connection is working properly.
	//
	// Parameters:
	//   - ctx: Context for validation timeout
	//
	// Returns:
	//   - error: Validation error, nil if connection is valid
	Validate(ctx context.Context) error

	// LastUsed returns the timestamp when this connection was last used.
	//
	// Returns:
	//   - time.Time: Last usage timestamp
	LastUsed() time.Time

	// MarkUsed updates the last usage timestamp to the current time.
	MarkUsed()

	// Close closes the connection and releases associated resources.
	//
	// Returns:
	//   - error: Close operation error, nil on success
	Close() error

	// GetMetrics returns connection-specific metrics.
	//
	// Returns:
	//   - *ConnectionMetrics: Current connection metrics
	GetMetrics() *ConnectionMetrics
}

// ConnectionFactory defines the interface for creating new connections.
// Different broker implementations provide their own connection factories
// to create broker-specific connections.
type ConnectionFactory interface {
	// CreateConnection creates a new connection using the provided configuration.
	// The connection is not automatically added to any pool - that's handled
	// by the ConnectionManager.
	//
	// Parameters:
	//   - ctx: Context for connection creation timeout
	//   - config: Connection configuration
	//   - endpoint: Specific endpoint to connect to
	//
	// Returns:
	//   - Connection: New connection instance
	//   - error: Connection creation error, nil on success
	CreateConnection(ctx context.Context, config *ConnectionConfig, endpoint string) (Connection, error)
}

// ConnectionStats contains connection manager statistics and metrics.
type ConnectionStats struct {
	// Pool statistics
	PoolSize     int `json:"pool_size"`
	ActiveCount  int `json:"active_count"`
	IdleCount    int `json:"idle_count"`
	WaitingCount int `json:"waiting_count"`
	MaxPoolSize  int `json:"max_pool_size"`
	MinPoolSize  int `json:"min_pool_size"`

	// Connection lifecycle statistics
	ConnectionsCreated int64 `json:"connections_created"`
	ConnectionsClosed  int64 `json:"connections_closed"`
	ConnectionsReused  int64 `json:"connections_reused"`
	ConnectionsFailed  int64 `json:"connections_failed"`

	// Health check statistics
	HealthChecksPerformed int64     `json:"health_checks_performed"`
	HealthChecksFailed    int64     `json:"health_checks_failed"`
	LastHealthCheck       time.Time `json:"last_health_check"`

	// Reconnection statistics
	ReconnectionAttempts    int64     `json:"reconnection_attempts"`
	SuccessfulReconnections int64     `json:"successful_reconnections"`
	FailedReconnections     int64     `json:"failed_reconnections"`
	LastReconnectionTime    time.Time `json:"last_reconnection_time"`

	// Performance statistics
	AverageGetTime    time.Duration `json:"average_get_time"`
	AverageReturnTime time.Duration `json:"average_return_time"`
	AverageIdleTime   time.Duration `json:"average_idle_time"`
	TotalWaitTime     time.Duration `json:"total_wait_time"`

	// Current state
	LastUpdated time.Time `json:"last_updated"`
}

// ConnectionMetrics contains metrics for individual connections.
type ConnectionMetrics struct {
	// Connection identification
	ConnectionID string    `json:"connection_id"`
	Endpoint     string    `json:"endpoint"`
	CreatedAt    time.Time `json:"created_at"`

	// Usage statistics
	TimesUsed       int64         `json:"times_used"`
	TotalActiveTime time.Duration `json:"total_active_time"`
	LastUsed        time.Time     `json:"last_used"`

	// Health statistics
	HealthChecks     int64     `json:"health_checks"`
	HealthCheckFails int64     `json:"health_check_fails"`
	LastHealthCheck  time.Time `json:"last_health_check"`
	LastHealthStatus bool      `json:"last_health_status"`

	// Connection state
	IsHealthy    bool          `json:"is_healthy"`
	IsIdle       bool          `json:"is_idle"`
	IdleDuration time.Duration `json:"idle_duration"`
}

// DefaultConnectionManager provides a robust connection management implementation
// with pooling, health monitoring, and automatic reconnection capabilities.
type DefaultConnectionManager struct {
	factory ConnectionFactory
	config  *ConnectionConfig

	// Connection pool
	pool *ConnectionPool

	// Statistics
	stats    *ConnectionStats
	statsMux sync.RWMutex

	// State management
	initialized int32
	closed      int32
	closeCh     chan struct{}

	// Background processes
	healthCheckTicker *time.Ticker
	maintenanceTicker *time.Ticker
	tickerMux         sync.RWMutex

	// Retry mechanism
	retryPolicy *ConnectionRetryPolicy
}

// NewDefaultConnectionManager creates a new connection manager with the specified factory.
func NewDefaultConnectionManager(factory ConnectionFactory) *DefaultConnectionManager {
	return &DefaultConnectionManager{
		factory: factory,
		stats:   &ConnectionStats{},
		closeCh: make(chan struct{}),
	}
}

// Initialize implements the ConnectionManager interface.
func (cm *DefaultConnectionManager) Initialize(ctx context.Context, config *ConnectionConfig, brokerConfig *BrokerConfig) error {
	if !atomic.CompareAndSwapInt32(&cm.initialized, 0, 1) {
		return NewConfigError("connection manager already initialized", nil)
	}

	if config == nil {
		return NewConfigError("connection configuration cannot be nil", nil)
	}

	if brokerConfig == nil {
		return NewConfigError("broker configuration cannot be nil", nil)
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid connection configuration: %w", err)
	}

	cm.config = config

	// Initialize connection pool
	poolConfig := &ConnectionPoolConfig{
		MaxSize:     config.PoolSize,
		MinSize:     maxInt(1, config.PoolSize/4), // Min size is 25% of max size
		IdleTimeout: config.IdleTimeout,
		MaxWait:     config.Timeout,
	}

	var err error
	cm.pool, err = NewConnectionPool(poolConfig, cm.factory, config, brokerConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Initialize retry policy
	retryConfig := &ConnectionRetryConfig{
		MaxAttempts:  config.MaxAttempts,
		InitialDelay: time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	cm.retryPolicy = NewConnectionRetryPolicy(retryConfig)

	// Initialize statistics
	cm.statsMux.Lock()
	cm.stats.MaxPoolSize = config.PoolSize
	cm.stats.MinPoolSize = poolConfig.MinSize
	cm.stats.LastUpdated = time.Now()
	cm.statsMux.Unlock()

	// Start background processes
	go cm.startHealthCheckLoop()
	go cm.startMaintenanceLoop()

	return nil
}

// GetConnection implements the ConnectionManager interface.
func (cm *DefaultConnectionManager) GetConnection(ctx context.Context) (Connection, error) {
	if atomic.LoadInt32(&cm.closed) == 1 {
		return nil, NewConnectionError("connection manager is closed", nil)
	}

	if atomic.LoadInt32(&cm.initialized) == 0 {
		return nil, NewConnectionError("connection manager not initialized", nil)
	}

	startTime := time.Now()
	defer func() {
		cm.updateGetTime(time.Since(startTime))
	}()

	conn, err := cm.pool.Get(ctx)
	if err != nil {
		cm.incrementConnectionsFailedStat()
		return nil, err
	}

	cm.incrementConnectionsReusedStat()
	return conn, nil
}

// ReturnConnection implements the ConnectionManager interface.
func (cm *DefaultConnectionManager) ReturnConnection(conn Connection) error {
	if atomic.LoadInt32(&cm.closed) == 1 {
		// If manager is closed, just close the connection
		return conn.Close()
	}

	startTime := time.Now()
	defer func() {
		cm.updateReturnTime(time.Since(startTime))
	}()

	return cm.pool.Return(conn)
}

// GetStats implements the ConnectionManager interface.
func (cm *DefaultConnectionManager) GetStats() *ConnectionStats {
	cm.statsMux.RLock()
	defer cm.statsMux.RUnlock()

	// Create a copy to avoid race conditions
	stats := *cm.stats

	// Update with current pool statistics
	if cm.pool != nil {
		poolStats := cm.pool.GetStats()
		stats.PoolSize = poolStats.PoolSize
		stats.ActiveCount = poolStats.ActiveCount
		stats.IdleCount = poolStats.IdleCount
		stats.WaitingCount = poolStats.WaitingCount
		stats.MaxPoolSize = poolStats.MaxSize
		stats.MinPoolSize = poolStats.MinSize
		stats.ConnectionsCreated = poolStats.CreatedCount
		stats.ConnectionsClosed = poolStats.ClosedCount
		// Note: ConnectionsReused is tracked separately by the connection manager
		// and should not be overwritten with poolStats.HitCount
	}

	stats.LastUpdated = time.Now()
	return &stats
}

// HealthCheck implements the ConnectionManager interface.
func (cm *DefaultConnectionManager) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if atomic.LoadInt32(&cm.closed) == 1 {
		return &HealthStatus{
			Status:      HealthStatusUnhealthy,
			Message:     "Connection manager is closed",
			LastChecked: time.Now(),
		}, nil
	}

	if atomic.LoadInt32(&cm.initialized) == 0 {
		return &HealthStatus{
			Status:      HealthStatusUnhealthy,
			Message:     "Connection manager not initialized",
			LastChecked: time.Now(),
		}, nil
	}

	startTime := time.Now()

	// Perform health check on the pool
	poolHealth, err := cm.pool.HealthCheck(ctx)
	if err != nil {
		cm.incrementHealthCheckFailedStat()
		return &HealthStatus{
			Status:       HealthStatusUnhealthy,
			Message:      fmt.Sprintf("Pool health check failed: %v", err),
			LastChecked:  time.Now(),
			ResponseTime: time.Since(startTime),
		}, err
	}

	cm.incrementHealthCheckPerformedStat()

	// Determine overall health status
	status := HealthStatusHealthy
	message := "Connection manager is healthy"

	stats := cm.GetStats()
	if stats.ActiveCount == 0 && stats.IdleCount == 0 {
		status = HealthStatusDegraded
		message = "No connections available in pool"
	} else if poolHealth.Status != HealthStatusHealthy {
		status = poolHealth.Status
		message = poolHealth.Message
	}

	return &HealthStatus{
		Status:       status,
		Message:      message,
		LastChecked:  time.Now(),
		ResponseTime: time.Since(startTime),
		Details: map[string]any{
			"pool_health": poolHealth,
			"stats":       stats,
		},
	}, nil
}

// Close implements the ConnectionManager interface.
func (cm *DefaultConnectionManager) Close() error {
	if !atomic.CompareAndSwapInt32(&cm.closed, 0, 1) {
		return nil // Already closed
	}

	// Signal background processes to stop
	close(cm.closeCh)

	// Stop background timers
	cm.tickerMux.Lock()
	if cm.healthCheckTicker != nil {
		cm.healthCheckTicker.Stop()
		cm.healthCheckTicker = nil
	}
	if cm.maintenanceTicker != nil {
		cm.maintenanceTicker.Stop()
		cm.maintenanceTicker = nil
	}
	cm.tickerMux.Unlock()

	// Close connection pool
	if cm.pool != nil {
		return cm.pool.Close()
	}

	return nil
}

// Background process methods

func (cm *DefaultConnectionManager) startHealthCheckLoop() {
	if cm.config.KeepAlive <= 0 {
		return // Health checks disabled
	}

	ticker := time.NewTicker(cm.config.KeepAlive)
	
	cm.tickerMux.Lock()
	cm.healthCheckTicker = ticker
	cm.tickerMux.Unlock()
	
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			cm.performScheduledHealthCheck(ctx)
			cancel()
		case <-cm.closeCh:
			return
		}
	}
}

func (cm *DefaultConnectionManager) startMaintenanceLoop() {
	maintenanceInterval := cm.config.IdleTimeout / 2
	if maintenanceInterval < time.Minute {
		maintenanceInterval = time.Minute
	}
	
	ticker := time.NewTicker(maintenanceInterval)
	
	cm.tickerMux.Lock()
	cm.maintenanceTicker = ticker
	cm.tickerMux.Unlock()
	
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.performMaintenance()
		case <-cm.closeCh:
			return
		}
	}
}

func (cm *DefaultConnectionManager) performScheduledHealthCheck(ctx context.Context) {
	_, err := cm.HealthCheck(ctx)
	if err != nil {
		// Health check errors are already recorded in the HealthCheck method
		return
	}

	// Perform pool maintenance
	if cm.pool != nil {
		cm.pool.ValidateConnections(ctx)
	}
}

func (cm *DefaultConnectionManager) performMaintenance() {
	if cm.pool != nil {
		cm.pool.CleanupIdleConnections()
		cm.pool.EnsureMinimumConnections()
	}
}

// Statistics update methods

func (cm *DefaultConnectionManager) updateGetTime(duration time.Duration) {
	cm.statsMux.Lock()
	defer cm.statsMux.Unlock()

	// Simple moving average calculation
	if cm.stats.AverageGetTime == 0 {
		cm.stats.AverageGetTime = duration
	} else {
		cm.stats.AverageGetTime = (cm.stats.AverageGetTime + duration) / 2
	}
}

func (cm *DefaultConnectionManager) updateReturnTime(duration time.Duration) {
	cm.statsMux.Lock()
	defer cm.statsMux.Unlock()

	if cm.stats.AverageReturnTime == 0 {
		cm.stats.AverageReturnTime = duration
	} else {
		cm.stats.AverageReturnTime = (cm.stats.AverageReturnTime + duration) / 2
	}
}

func (cm *DefaultConnectionManager) incrementConnectionsReusedStat() {
	cm.statsMux.Lock()
	defer cm.statsMux.Unlock()
	cm.stats.ConnectionsReused++
}

func (cm *DefaultConnectionManager) incrementConnectionsFailedStat() {
	cm.statsMux.Lock()
	defer cm.statsMux.Unlock()
	cm.stats.ConnectionsFailed++
}

func (cm *DefaultConnectionManager) incrementHealthCheckPerformedStat() {
	cm.statsMux.Lock()
	defer cm.statsMux.Unlock()
	cm.stats.HealthChecksPerformed++
	cm.stats.LastHealthCheck = time.Now()
}

func (cm *DefaultConnectionManager) incrementHealthCheckFailedStat() {
	cm.statsMux.Lock()
	defer cm.statsMux.Unlock()
	cm.stats.HealthChecksFailed++
}

// ConnectionRetryConfig defines configuration for connection retry operations.
type ConnectionRetryConfig struct {
	MaxAttempts  int           `json:"max_attempts"`
	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
	Multiplier   float64       `json:"multiplier"`
	Jitter       float64       `json:"jitter"`
}

// ConnectionRetryPolicy implements exponential backoff with jitter for connection retry operations.
type ConnectionRetryPolicy struct {
	config *ConnectionRetryConfig
}

// NewConnectionRetryPolicy creates a new connection retry policy with the given configuration.
func NewConnectionRetryPolicy(config *ConnectionRetryConfig) *ConnectionRetryPolicy {
	return &ConnectionRetryPolicy{
		config: config,
	}
}

// ShouldRetry determines if an operation should be retried based on the attempt count and error.
func (rp *ConnectionRetryPolicy) ShouldRetry(attempt int, err error) bool {
	if attempt >= rp.config.MaxAttempts {
		return false
	}

	// Check if error is retryable
	return IsRetryableError(err)
}

// GetDelay calculates the delay before the next retry attempt.
// Uses exponential backoff with jitter to prevent thundering herd problems.
func (rp *ConnectionRetryPolicy) GetDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate exponential backoff delay
	delay := float64(rp.config.InitialDelay) * math.Pow(rp.config.Multiplier, float64(attempt-1))

	// Apply maximum delay limit
	if time.Duration(delay) > rp.config.MaxDelay {
		delay = float64(rp.config.MaxDelay)
	}

	// Add jitter to prevent thundering herd
	jitter := delay * rp.config.Jitter * (rand.Float64()*2 - 1) // Random value between -jitter and +jitter
	delay += jitter

	// Ensure delay is not negative
	if delay < 0 {
		delay = float64(rp.config.InitialDelay)
	}

	return time.Duration(delay)
}

// ExecuteWithRetry executes an operation with retry logic.
func (rp *ConnectionRetryPolicy) ExecuteWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 1; attempt <= rp.config.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the operation
		err := operation()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry
		if !rp.ShouldRetry(attempt, err) {
			break
		}

		// Calculate delay and wait
		delay := rp.GetDelay(attempt)
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
				// Continue to next attempt
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			}
		}
	}

	return lastErr
}

// Utility functions

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
