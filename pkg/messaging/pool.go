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
	"sync"
	"sync/atomic"
	"time"
)

// ConnectionPool manages a pool of connections with configurable size limits,
// idle timeout handling, and connection health validation.
//
// The pool maintains connections in different states:
// - Available: Ready to be used connections
// - Active: Currently in-use connections
// - Idle: Connections that have been idle for some time
//
// Features:
// - Thread-safe operations
// - Configurable min/max pool sizes
// - Automatic idle connection cleanup
// - Connection health validation
// - Waiting queue when pool is exhausted
// - Graceful shutdown with proper cleanup
type ConnectionPool struct {
	config       *ConnectionPoolConfig
	factory      ConnectionFactory
	connConfig   *ConnectionConfig
	brokerConfig *BrokerConfig

	// Connection storage
	available chan Connection
	active    map[string]Connection
	activeMux sync.RWMutex

	// Waiting queue for connection requests
	waiting    []chan Connection
	waitingMux sync.Mutex

	// Pool state
	closed           int32
	totalConnections int32
	availableMux     sync.Mutex // Protects access to available channel

	// Statistics
	stats    *ConnectionPoolStats
	statsMux sync.RWMutex
}

// ConnectionPoolConfig defines configuration for the connection pool.
type ConnectionPoolConfig struct {
	// MaxSize is the maximum number of connections in the pool
	MaxSize int `json:"max_size" yaml:"max_size"`

	// MinSize is the minimum number of connections to maintain
	MinSize int `json:"min_size" yaml:"min_size"`

	// IdleTimeout is the maximum time a connection can be idle before closure
	IdleTimeout time.Duration `json:"idle_timeout" yaml:"idle_timeout"`

	// MaxWait is the maximum time to wait for a connection when pool is full
	MaxWait time.Duration `json:"max_wait" yaml:"max_wait"`

	// ValidationInterval is how often to validate idle connections
	ValidationInterval time.Duration `json:"validation_interval" yaml:"validation_interval"`
}

// ConnectionPoolStats contains statistics about the connection pool.
type ConnectionPoolStats struct {
	PoolSize     int       `json:"pool_size"`
	ActiveCount  int       `json:"active_count"`
	IdleCount    int       `json:"idle_count"`
	WaitingCount int       `json:"waiting_count"`
	MaxSize      int       `json:"max_size"`
	MinSize      int       `json:"min_size"`
	CreatedCount int64     `json:"created_count"`
	ClosedCount  int64     `json:"closed_count"`
	HitCount     int64     `json:"hit_count"`
	MissCount    int64     `json:"miss_count"`
	LastActivity time.Time `json:"last_activity"`
}

// NewConnectionPool creates a new connection pool with the specified configuration.
func NewConnectionPool(config *ConnectionPoolConfig, factory ConnectionFactory, connConfig *ConnectionConfig, brokerConfig *BrokerConfig) (*ConnectionPool, error) {
	if config == nil {
		return nil, fmt.Errorf("pool configuration cannot be nil")
	}

	if factory == nil {
		return nil, fmt.Errorf("connection factory cannot be nil")
	}

	if connConfig == nil {
		return nil, fmt.Errorf("connection configuration cannot be nil")
	}

	if brokerConfig == nil {
		return nil, fmt.Errorf("broker configuration cannot be nil")
	}

	// Validate configuration
	if err := validatePoolConfig(config); err != nil {
		return nil, fmt.Errorf("invalid pool configuration: %w", err)
	}

	pool := &ConnectionPool{
		config:       config,
		factory:      factory,
		connConfig:   connConfig,
		brokerConfig: brokerConfig,
		available:    make(chan Connection, config.MaxSize),
		active:       make(map[string]Connection),
		waiting:      make([]chan Connection, 0),
		stats: &ConnectionPoolStats{
			MaxSize:      config.MaxSize,
			MinSize:      config.MinSize,
			LastActivity: time.Now(),
		},
	}

	return pool, nil
}

// Get retrieves a connection from the pool.
// If no connection is available, it may create a new one or wait for one to become available.
func (p *ConnectionPool) Get(ctx context.Context) (Connection, error) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return nil, fmt.Errorf("connection pool is closed")
	}

	p.updateLastActivity()

	// Try to get an available connection first
	select {
	case conn := <-p.available:
		if conn.IsHealthy() {
			p.moveToActive(conn)
			p.incrementHitCount()
			return conn, nil
		} else {
			// Connection is unhealthy, close it and try to create a new one
			conn.Close()
			p.decrementTotalConnections()
		}
	default:
		// No available connections
	}

	// Try to create a new connection if under limit
	if p.canCreateConnection() {
		conn, err := p.createConnection(ctx)
		if err != nil {
			p.incrementMissCount()
			return nil, err
		}
		p.moveToActive(conn)
		return conn, nil
	}

	// Pool is full, wait for a connection to become available
	return p.waitForConnection(ctx)
}

// Return returns a connection to the pool.
func (p *ConnectionPool) Return(conn Connection) error {
	if atomic.LoadInt32(&p.closed) == 1 {
		// Pool is closed, just close the connection
		return conn.Close()
	}

	if conn == nil {
		return fmt.Errorf("cannot return nil connection")
	}

	p.updateLastActivity()

	// Remove from active connections
	p.removeFromActive(conn)

	// Check if connection is still healthy
	if !conn.IsHealthy() {
		conn.Close()
		p.decrementTotalConnections()
		return nil
	}

	// Try to satisfy a waiting request first
	if p.satisfyWaiter(conn) {
		return nil
	}

	// Return to available pool
	select {
	case p.available <- conn:
		return nil
	default:
		// Pool is full, close the connection
		conn.Close()
		p.decrementTotalConnections()
		return nil
	}
}

// GetStats returns current pool statistics.
func (p *ConnectionPool) GetStats() *ConnectionPoolStats {
	p.statsMux.RLock()
	defer p.statsMux.RUnlock()

	stats := *p.stats
	stats.PoolSize = int(atomic.LoadInt32(&p.totalConnections))
	stats.ActiveCount = p.getActiveCount()
	stats.IdleCount = len(p.available)
	stats.WaitingCount = p.getWaitingCount()

	return &stats
}

// HealthCheck performs a health check on the pool and its connections.
func (p *ConnectionPool) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return &HealthStatus{
			Status:      HealthStatusUnhealthy,
			Message:     "Connection pool is closed",
			LastChecked: time.Now(),
		}, nil
	}

	startTime := time.Now()
	stats := p.GetStats()

	status := HealthStatusHealthy
	message := "Connection pool is healthy"

	// Check pool utilization
	utilization := float64(stats.ActiveCount) / float64(stats.MaxSize)
	if utilization > 0.9 {
		status = HealthStatusDegraded
		message = fmt.Sprintf("High pool utilization: %.1f%%", utilization*100)
	} else if stats.PoolSize == 0 {
		status = HealthStatusDegraded
		message = "No connections in pool"
	}

	// Check for excessive waiting
	if stats.WaitingCount > stats.MaxSize/2 {
		status = HealthStatusDegraded
		message = fmt.Sprintf("High wait queue: %d requests waiting", stats.WaitingCount)
	}

	return &HealthStatus{
		Status:       status,
		Message:      message,
		LastChecked:  time.Now(),
		ResponseTime: time.Since(startTime),
		Details: map[string]any{
			"pool_stats":  stats,
			"utilization": utilization,
		},
	}, nil
}

// ValidateConnections validates all idle connections and removes unhealthy ones.
func (p *ConnectionPool) ValidateConnections(ctx context.Context) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return
	}

	// Create a slice to hold connections temporarily
	var connections []Connection

	// Drain available connections
	for {
		select {
		case conn := <-p.available:
			connections = append(connections, conn)
		default:
			goto validate
		}
	}

validate:
	// Validate each connection
	for _, conn := range connections {
		if conn.IsHealthy() {
			// Perform deeper validation
			validationCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Validate(validationCtx)
			cancel()

			if err == nil {
				// Connection is valid, try to return to pool
				// Check if pool is closed before attempting to return connection
				if atomic.LoadInt32(&p.closed) == 0 {
					p.availableMux.Lock()
					select {
					case p.available <- conn:
						// Successfully returned to pool
					case <-time.After(time.Millisecond):
						// Timeout - pool might be closing or full, close connection
						conn.Close()
						p.decrementTotalConnections()
					case <-ctx.Done():
						// Context was cancelled, close connection
						conn.Close()
						p.decrementTotalConnections()
					}
					p.availableMux.Unlock()
				} else {
					// Pool is closing, close connection
					conn.Close()
					p.decrementTotalConnections()
				}
			} else {
				// Validation failed, close connection
				conn.Close()
				p.decrementTotalConnections()
			}
		} else {
			// Connection is unhealthy, close it
			conn.Close()
			p.decrementTotalConnections()
		}
	}
}

// CleanupIdleConnections removes connections that have been idle for too long.
func (p *ConnectionPool) CleanupIdleConnections() {
	if atomic.LoadInt32(&p.closed) == 1 || p.config.IdleTimeout <= 0 {
		return
	}

	var connections []Connection

	// Drain available connections
	for {
		select {
		case conn := <-p.available:
			connections = append(connections, conn)
		default:
			goto cleanup
		}
	}

cleanup:
	cutoffTime := time.Now().Add(-p.config.IdleTimeout)

	for _, conn := range connections {
		if conn.LastUsed().Before(cutoffTime) {
			// Connection has been idle too long, close it
			conn.Close()
			p.decrementTotalConnections()
		} else {
			// Connection is not too old, return to pool
			// Check if pool is closed before attempting to return connection
			if atomic.LoadInt32(&p.closed) == 0 {
				p.availableMux.Lock()
				select {
				case p.available <- conn:
				default:
					// Pool is full, close connection
					conn.Close()
					p.decrementTotalConnections()
				}
				p.availableMux.Unlock()
			} else {
				// Pool is closing, close connection
				conn.Close()
				p.decrementTotalConnections()
			}
		}
	}
}

// EnsureMinimumConnections ensures the pool has at least the minimum number of connections.
func (p *ConnectionPool) EnsureMinimumConnections() {
	if atomic.LoadInt32(&p.closed) == 1 {
		return
	}

	current := int(atomic.LoadInt32(&p.totalConnections))
	needed := p.config.MinSize - current

	if needed > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		for i := 0; i < needed; i++ {
			if !p.canCreateConnection() {
				break
			}

			conn, err := p.createConnection(ctx)
			if err != nil {
				break
			}

			// Add to available pool
			select {
			case p.available <- conn:
			default:
				// Pool is full, close connection
				conn.Close()
				p.decrementTotalConnections()
				break
			}
		}
	}
}

// Close closes the connection pool and all its connections.
func (p *ConnectionPool) Close() error {
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return nil // Already closed
	}

	// Cancel all waiting requests
	p.cancelAllWaiters()

	// Close all available connections
	p.availableMux.Lock()
	close(p.available)
	for conn := range p.available {
		conn.Close()
	}
	p.availableMux.Unlock()

	// Close all active connections
	p.activeMux.RLock()
	activeConns := make([]Connection, 0, len(p.active))
	for _, conn := range p.active {
		activeConns = append(activeConns, conn)
	}
	p.activeMux.RUnlock()

	for _, conn := range activeConns {
		conn.Close()
	}

	// Update statistics
	p.statsMux.Lock()
	p.stats.ClosedCount += int64(len(activeConns))
	p.statsMux.Unlock()

	return nil
}

// Private methods

func (p *ConnectionPool) createConnection(ctx context.Context) (Connection, error) {
	if len(p.brokerConfig.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints configured")
	}

	// Select an endpoint (simple round-robin for now)
	endpoint := p.brokerConfig.Endpoints[int(atomic.LoadInt32(&p.totalConnections))%len(p.brokerConfig.Endpoints)]

	conn, err := p.factory.CreateConnection(ctx, p.connConfig, endpoint)
	if err != nil {
		return nil, err
	}

	p.incrementTotalConnections()
	p.incrementCreatedCount()

	return conn, nil
}

func (p *ConnectionPool) canCreateConnection() bool {
	return atomic.LoadInt32(&p.totalConnections) < int32(p.config.MaxSize)
}

func (p *ConnectionPool) moveToActive(conn Connection) {
	p.activeMux.Lock()
	defer p.activeMux.Unlock()

	conn.MarkUsed()
	p.active[conn.ID()] = conn
}

func (p *ConnectionPool) removeFromActive(conn Connection) {
	p.activeMux.Lock()
	defer p.activeMux.Unlock()

	delete(p.active, conn.ID())
}

func (p *ConnectionPool) getActiveCount() int {
	p.activeMux.RLock()
	defer p.activeMux.RUnlock()
	return len(p.active)
}

func (p *ConnectionPool) waitForConnection(ctx context.Context) (Connection, error) {
	p.incrementMissCount()

	// Create a channel for this waiter
	waiterCh := make(chan Connection, 1)

	// Add to waiting queue
	p.waitingMux.Lock()
	p.waiting = append(p.waiting, waiterCh)
	p.waitingMux.Unlock()

	defer func() {
		// Remove from waiting queue
		p.removeWaiter(waiterCh)
	}()

	// Wait for connection or timeout
	maxWait := p.config.MaxWait
	if maxWait <= 0 {
		maxWait = 30 * time.Second // Default timeout
	}

	timeout := time.NewTimer(maxWait)
	defer timeout.Stop()

	select {
	case conn := <-waiterCh:
		if conn != nil {
			p.moveToActive(conn)
			return conn, nil
		}
		return nil, fmt.Errorf("received nil connection from waiter")
	case <-timeout.C:
		return nil, fmt.Errorf("timeout waiting for connection after %v", maxWait)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *ConnectionPool) satisfyWaiter(conn Connection) bool {
	p.waitingMux.Lock()
	defer p.waitingMux.Unlock()

	if len(p.waiting) == 0 {
		return false
	}

	// Get the first waiter
	waiterCh := p.waiting[0]
	p.waiting = p.waiting[1:]

	// Send connection to waiter
	select {
	case waiterCh <- conn:
		return true
	default:
		// Waiter is gone, return false
		return false
	}
}

func (p *ConnectionPool) removeWaiter(target chan Connection) {
	p.waitingMux.Lock()
	defer p.waitingMux.Unlock()

	for i, waiterCh := range p.waiting {
		if waiterCh == target {
			// Remove this waiter
			p.waiting = append(p.waiting[:i], p.waiting[i+1:]...)
			break
		}
	}
}

func (p *ConnectionPool) cancelAllWaiters() {
	p.waitingMux.Lock()
	defer p.waitingMux.Unlock()

	for _, waiterCh := range p.waiting {
		close(waiterCh)
	}
	p.waiting = nil
}

func (p *ConnectionPool) getWaitingCount() int {
	p.waitingMux.Lock()
	defer p.waitingMux.Unlock()
	return len(p.waiting)
}

// Statistics methods

func (p *ConnectionPool) incrementTotalConnections() {
	atomic.AddInt32(&p.totalConnections, 1)
}

func (p *ConnectionPool) decrementTotalConnections() {
	atomic.AddInt32(&p.totalConnections, -1)
}

func (p *ConnectionPool) incrementCreatedCount() {
	p.statsMux.Lock()
	defer p.statsMux.Unlock()
	p.stats.CreatedCount++
}

func (p *ConnectionPool) incrementClosedCount() {
	p.statsMux.Lock()
	defer p.statsMux.Unlock()
	p.stats.ClosedCount++
}

func (p *ConnectionPool) incrementHitCount() {
	p.statsMux.Lock()
	defer p.statsMux.Unlock()
	p.stats.HitCount++
}

func (p *ConnectionPool) incrementMissCount() {
	p.statsMux.Lock()
	defer p.statsMux.Unlock()
	p.stats.MissCount++
}

func (p *ConnectionPool) updateLastActivity() {
	p.statsMux.Lock()
	defer p.statsMux.Unlock()
	p.stats.LastActivity = time.Now()
}

// Validation functions

func validatePoolConfig(config *ConnectionPoolConfig) error {
	if config.MaxSize <= 0 {
		return fmt.Errorf("max size must be positive")
	}

	if config.MinSize < 0 {
		return fmt.Errorf("min size cannot be negative")
	}

	if config.MinSize > config.MaxSize {
		return fmt.Errorf("min size cannot be greater than max size")
	}

	if config.IdleTimeout < 0 {
		return fmt.Errorf("idle timeout cannot be negative")
	}

	if config.MaxWait < 0 {
		return fmt.Errorf("max wait cannot be negative")
	}

	return nil
}
