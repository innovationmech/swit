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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockConnection implements the Connection interface for testing.
type MockConnection struct {
	id         string
	healthy    int32
	lastUsed   time.Time
	closed     int32
	validateFn func(ctx context.Context) error

	// Metrics
	timesUsed        int64
	totalActiveTime  time.Duration
	healthChecks     int64
	healthCheckFails int64
	lastHealthCheck  time.Time

	mu sync.RWMutex
}

// NewMockConnection creates a new mock connection for testing.
func NewMockConnection(id string) *MockConnection {
	return &MockConnection{
		id:       id,
		healthy:  1,
		lastUsed: time.Now(),
		validateFn: func(ctx context.Context) error {
			return nil
		},
	}
}

func (mc *MockConnection) ID() string {
	return mc.id
}

func (mc *MockConnection) IsHealthy() bool {
	return atomic.LoadInt32(&mc.healthy) == 1
}

func (mc *MockConnection) Validate(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	atomic.AddInt64(&mc.healthChecks, 1)
	mc.lastHealthCheck = time.Now()

	if mc.validateFn != nil {
		err := mc.validateFn(ctx)
		if err != nil {
			atomic.AddInt64(&mc.healthCheckFails, 1)
		}
		return err
	}

	return nil
}

func (mc *MockConnection) LastUsed() time.Time {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.lastUsed
}

func (mc *MockConnection) MarkUsed() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.lastUsed = time.Now()
	atomic.AddInt64(&mc.timesUsed, 1)
}

func (mc *MockConnection) Close() error {
	atomic.StoreInt32(&mc.closed, 1)
	atomic.StoreInt32(&mc.healthy, 0)
	return nil
}

func (mc *MockConnection) GetMetrics() *ConnectionMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return &ConnectionMetrics{
		ConnectionID:     mc.id,
		Endpoint:         "mock://localhost:9092",
		CreatedAt:        time.Now().Add(-time.Hour),
		TimesUsed:        atomic.LoadInt64(&mc.timesUsed),
		TotalActiveTime:  mc.totalActiveTime,
		LastUsed:         mc.lastUsed,
		HealthChecks:     atomic.LoadInt64(&mc.healthChecks),
		HealthCheckFails: atomic.LoadInt64(&mc.healthCheckFails),
		LastHealthCheck:  mc.lastHealthCheck,
		LastHealthStatus: mc.IsHealthy(),
		IsHealthy:        mc.IsHealthy(),
		IsIdle:           time.Since(mc.lastUsed) > time.Minute,
		IdleDuration:     time.Since(mc.lastUsed),
	}
}

// SetHealthy sets the connection health status for testing.
func (mc *MockConnection) SetHealthy(healthy bool) {
	if healthy {
		atomic.StoreInt32(&mc.healthy, 1)
	} else {
		atomic.StoreInt32(&mc.healthy, 0)
	}
}

// SetValidateFunc sets a custom validation function for testing.
func (mc *MockConnection) SetValidateFunc(fn func(ctx context.Context) error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.validateFn = fn
}

// IsClosed returns true if the connection has been closed.
func (mc *MockConnection) IsClosed() bool {
	return atomic.LoadInt32(&mc.closed) == 1
}

// MockConnectionFactory implements the ConnectionFactory interface for testing.
type MockConnectionFactory struct {
	connections []*MockConnection
	createFunc  func(ctx context.Context, config *ConnectionConfig, endpoint string) (Connection, error)
	createCount int64
	failAfter   int64

	mu sync.Mutex
}

// NewMockConnectionFactory creates a new mock connection factory for testing.
func NewMockConnectionFactory() *MockConnectionFactory {
	return &MockConnectionFactory{
		connections: make([]*MockConnection, 0),
		failAfter:   -1, // Never fail by default
	}
}

func (mcf *MockConnectionFactory) CreateConnection(ctx context.Context, config *ConnectionConfig, endpoint string) (Connection, error) {
	mcf.mu.Lock()
	defer mcf.mu.Unlock()

	count := atomic.AddInt64(&mcf.createCount, 1)

	// Check if we should fail
	if mcf.failAfter >= 0 && count > mcf.failAfter {
		return nil, NewConnectionError("mock connection creation failed", nil)
	}

	// Use custom create function if provided
	if mcf.createFunc != nil {
		return mcf.createFunc(ctx, config, endpoint)
	}

	// Create default mock connection
	id := fmt.Sprintf("mock-conn-%d", count)
	conn := NewMockConnection(id)
	mcf.connections = append(mcf.connections, conn)

	return conn, nil
}

// SetCreateFunc sets a custom create function for testing.
func (mcf *MockConnectionFactory) SetCreateFunc(fn func(ctx context.Context, config *ConnectionConfig, endpoint string) (Connection, error)) {
	mcf.mu.Lock()
	defer mcf.mu.Unlock()
	mcf.createFunc = fn
}

// SetFailAfter sets the factory to fail after creating N connections.
func (mcf *MockConnectionFactory) SetFailAfter(count int64) {
	mcf.mu.Lock()
	defer mcf.mu.Unlock()
	mcf.failAfter = count
}

// GetCreateCount returns the number of connections created.
func (mcf *MockConnectionFactory) GetCreateCount() int64 {
	return atomic.LoadInt64(&mcf.createCount)
}

// GetConnections returns all created connections.
func (mcf *MockConnectionFactory) GetConnections() []*MockConnection {
	mcf.mu.Lock()
	defer mcf.mu.Unlock()
	return append([]*MockConnection{}, mcf.connections...)
}

// Test DefaultConnectionManager

func TestDefaultConnectionManager_Initialize(t *testing.T) {
	factory := NewMockConnectionFactory()
	cm := NewDefaultConnectionManager(factory)

	config := &ConnectionConfig{
		Timeout:     10 * time.Second,
		KeepAlive:   30 * time.Second,
		MaxAttempts: 3,
		PoolSize:    5,
		IdleTimeout: 5 * time.Minute,
	}

	brokerConfig := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092"},
	}

	ctx := context.Background()
	err := cm.Initialize(ctx, config, brokerConfig)
	require.NoError(t, err)

	// Test double initialization fails
	err = cm.Initialize(ctx, config, brokerConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already initialized")

	err = cm.Close()
	require.NoError(t, err)
}

func TestDefaultConnectionManager_InitializeInvalidConfig(t *testing.T) {
	// Test nil config
	factory := NewMockConnectionFactory()
	cm := NewDefaultConnectionManager(factory)

	brokerConfig := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092"},
	}
	err := cm.Initialize(context.Background(), nil, brokerConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")

	// Test nil broker config
	factory2 := NewMockConnectionFactory()
	cm2 := NewDefaultConnectionManager(factory2)
	config := &ConnectionConfig{
		Timeout:  10 * time.Second,
		PoolSize: 5,
	}
	err = cm2.Initialize(context.Background(), config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")

	// Test invalid config
	factory3 := NewMockConnectionFactory()
	cm3 := NewDefaultConnectionManager(factory3)
	invalidConfig := &ConnectionConfig{
		Timeout:  -1, // Invalid
		PoolSize: 5,
	}
	err = cm3.Initialize(context.Background(), invalidConfig, brokerConfig)
	assert.Error(t, err)
}

func TestDefaultConnectionManager_GetConnection(t *testing.T) {
	factory := NewMockConnectionFactory()
	cm := NewDefaultConnectionManager(factory)

	config := &ConnectionConfig{
		Timeout:     10 * time.Second,
		KeepAlive:   30 * time.Second,
		MaxAttempts: 3,
		PoolSize:    2,
		IdleTimeout: 5 * time.Minute,
	}

	brokerConfig := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092"},
	}

	ctx := context.Background()
	err := cm.Initialize(ctx, config, brokerConfig)
	require.NoError(t, err)
	defer cm.Close()

	// Get first connection
	conn1, err := cm.GetConnection(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn1)
	assert.True(t, conn1.IsHealthy())

	// Get second connection
	conn2, err := cm.GetConnection(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn2)
	assert.NotEqual(t, conn1.ID(), conn2.ID())

	// Return connections
	err = cm.ReturnConnection(conn1)
	require.NoError(t, err)
	err = cm.ReturnConnection(conn2)
	require.NoError(t, err)

	// Get connection again, should reuse
	conn3, err := cm.GetConnection(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn3)
}

func TestDefaultConnectionManager_GetConnectionClosed(t *testing.T) {
	factory := NewMockConnectionFactory()
	cm := NewDefaultConnectionManager(factory)

	config := &ConnectionConfig{
		Timeout:     10 * time.Second,
		KeepAlive:   30 * time.Second,
		MaxAttempts: 3,
		PoolSize:    2,
		IdleTimeout: 5 * time.Minute,
	}

	brokerConfig := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092"},
	}

	ctx := context.Background()
	err := cm.Initialize(ctx, config, brokerConfig)
	require.NoError(t, err)

	// Close the manager
	err = cm.Close()
	require.NoError(t, err)

	// Try to get connection after close
	conn, err := cm.GetConnection(ctx)
	assert.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "closed")
}

func TestDefaultConnectionManager_GetConnectionUninitialized(t *testing.T) {
	factory := NewMockConnectionFactory()
	cm := NewDefaultConnectionManager(factory)

	ctx := context.Background()
	conn, err := cm.GetConnection(ctx)
	assert.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestDefaultConnectionManager_ReturnConnection(t *testing.T) {
	factory := NewMockConnectionFactory()
	cm := NewDefaultConnectionManager(factory)

	config := &ConnectionConfig{
		Timeout:     10 * time.Second,
		KeepAlive:   30 * time.Second,
		MaxAttempts: 3,
		PoolSize:    2,
		IdleTimeout: 5 * time.Minute,
	}

	brokerConfig := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092"},
	}

	ctx := context.Background()
	err := cm.Initialize(ctx, config, brokerConfig)
	require.NoError(t, err)
	defer cm.Close()

	// Get connection
	conn, err := cm.GetConnection(ctx)
	require.NoError(t, err)

	// Return connection
	err = cm.ReturnConnection(conn)
	require.NoError(t, err)

	// Return unhealthy connection
	mockConn := conn.(*MockConnection)
	mockConn.SetHealthy(false)

	err = cm.ReturnConnection(conn)
	require.NoError(t, err)
	assert.True(t, mockConn.IsClosed())
}

func TestDefaultConnectionManager_GetStats(t *testing.T) {
	factory := NewMockConnectionFactory()
	cm := NewDefaultConnectionManager(factory)

	config := &ConnectionConfig{
		Timeout:     10 * time.Second,
		KeepAlive:   30 * time.Second,
		MaxAttempts: 3,
		PoolSize:    2,
		IdleTimeout: 5 * time.Minute,
	}

	brokerConfig := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092"},
	}

	ctx := context.Background()
	err := cm.Initialize(ctx, config, brokerConfig)
	require.NoError(t, err)
	defer cm.Close()

	// Get initial stats
	stats := cm.GetStats()
	require.NotNil(t, stats)
	assert.Equal(t, 2, stats.MaxPoolSize)
	assert.Equal(t, 0, stats.ActiveCount)

	// Get connection and check stats
	conn, err := cm.GetConnection(ctx)
	require.NoError(t, err)

	stats = cm.GetStats()
	assert.Equal(t, 1, stats.ActiveCount)
	assert.Equal(t, int64(1), stats.ConnectionsReused)

	// Return connection and check stats
	err = cm.ReturnConnection(conn)
	require.NoError(t, err)

	stats = cm.GetStats()
	assert.Equal(t, 0, stats.ActiveCount)
	assert.Equal(t, 1, stats.IdleCount)
}

func TestDefaultConnectionManager_HealthCheck(t *testing.T) {
	factory := NewMockConnectionFactory()
	cm := NewDefaultConnectionManager(factory)

	config := &ConnectionConfig{
		Timeout:     10 * time.Second,
		KeepAlive:   30 * time.Second,
		MaxAttempts: 3,
		PoolSize:    2,
		IdleTimeout: 5 * time.Minute,
	}

	brokerConfig := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092"},
	}

	ctx := context.Background()
	err := cm.Initialize(ctx, config, brokerConfig)
	require.NoError(t, err)
	defer cm.Close()

	// Health check on empty pool
	health, err := cm.HealthCheck(ctx)
	require.NoError(t, err)
	assert.Equal(t, HealthStatusDegraded, health.Status)
	assert.Contains(t, health.Message, "No connections")

	// Get connection and check health
	conn, err := cm.GetConnection(ctx)
	require.NoError(t, err)
	err = cm.ReturnConnection(conn)
	require.NoError(t, err)

	health, err = cm.HealthCheck(ctx)
	require.NoError(t, err)
	assert.Equal(t, HealthStatusHealthy, health.Status)
}

func TestDefaultConnectionManager_Close(t *testing.T) {
	factory := NewMockConnectionFactory()
	cm := NewDefaultConnectionManager(factory)

	config := &ConnectionConfig{
		Timeout:     10 * time.Second,
		KeepAlive:   30 * time.Second,
		MaxAttempts: 3,
		PoolSize:    2,
		IdleTimeout: 5 * time.Minute,
	}

	brokerConfig := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092"},
	}

	ctx := context.Background()
	err := cm.Initialize(ctx, config, brokerConfig)
	require.NoError(t, err)

	// Get and use some connections
	conn1, err := cm.GetConnection(ctx)
	require.NoError(t, err)
	conn2, err := cm.GetConnection(ctx)
	require.NoError(t, err)
	err = cm.ReturnConnection(conn1)
	require.NoError(t, err)
	// Keep conn2 active

	// Close manager
	err = cm.Close()
	require.NoError(t, err)

	// Verify connections are closed
	mockConn1 := conn1.(*MockConnection)
	mockConn2 := conn2.(*MockConnection)
	assert.True(t, mockConn1.IsClosed())
	assert.True(t, mockConn2.IsClosed())

	// Double close should be safe
	err = cm.Close()
	require.NoError(t, err)
}

// Test ConnectionRetryPolicy

func TestConnectionRetryPolicy_ShouldRetry(t *testing.T) {
	config := &ConnectionRetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	policy := NewConnectionRetryPolicy(config)

	// Test retryable error
	retryableErr := NewConnectionError("connection failed", nil)
	assert.True(t, policy.ShouldRetry(1, retryableErr))
	assert.True(t, policy.ShouldRetry(2, retryableErr))
	assert.False(t, policy.ShouldRetry(3, retryableErr)) // Max attempts reached

	// Test non-retryable error
	nonRetryableErr := NewConfigError("invalid config", nil)
	assert.False(t, policy.ShouldRetry(1, nonRetryableErr))
}

func TestConnectionRetryPolicy_GetDelay(t *testing.T) {
	config := &ConnectionRetryConfig{
		MaxAttempts:  5,
		InitialDelay: time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	policy := NewConnectionRetryPolicy(config)

	// Test delay calculation
	delay0 := policy.GetDelay(0)
	assert.Equal(t, time.Duration(0), delay0)

	delay1 := policy.GetDelay(1)
	assert.True(t, delay1 >= 900*time.Millisecond)  // 1s - 10% jitter
	assert.True(t, delay1 <= 1100*time.Millisecond) // 1s + 10% jitter

	delay2 := policy.GetDelay(2)
	assert.True(t, delay2 >= 1800*time.Millisecond) // 2s - 10% jitter
	assert.True(t, delay2 <= 2200*time.Millisecond) // 2s + 10% jitter

	// Test max delay limit
	delay5 := policy.GetDelay(5)
	assert.True(t, delay5 <= 11*time.Second) // Should not exceed max delay + jitter
}

func TestConnectionRetryPolicy_ExecuteWithRetry(t *testing.T) {
	config := &ConnectionRetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	policy := NewConnectionRetryPolicy(config)
	ctx := context.Background()

	// Test successful operation
	attempts := 0
	err := policy.ExecuteWithRetry(ctx, func() error {
		attempts++
		if attempts < 2 {
			return NewConnectionError("temporary failure", nil)
		}
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)

	// Test operation that always fails
	attempts = 0
	err = policy.ExecuteWithRetry(ctx, func() error {
		attempts++
		return NewConnectionError("persistent failure", nil)
	})
	assert.Error(t, err)
	assert.Equal(t, 3, attempts) // Should try max attempts

	// Test non-retryable error
	attempts = 0
	err = policy.ExecuteWithRetry(ctx, func() error {
		attempts++
		return NewConfigError("config error", nil)
	})
	assert.Error(t, err)
	assert.Equal(t, 1, attempts) // Should not retry
}

func TestConnectionRetryPolicy_ExecuteWithRetryContextCancellation(t *testing.T) {
	config := &ConnectionRetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	policy := NewConnectionRetryPolicy(config)

	// Test context cancellation during retry
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	start := time.Now()
	err := policy.ExecuteWithRetry(ctx, func() error {
		attempts++
		return NewConnectionError("failure", nil)
	})

	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.True(t, time.Since(start) < 200*time.Millisecond) // Should exit quickly
	assert.Equal(t, 1, attempts)                             // Should not complete second attempt
}

// Benchmark tests

func BenchmarkDefaultConnectionManager_GetReturnConnection(b *testing.B) {
	factory := NewMockConnectionFactory()
	cm := NewDefaultConnectionManager(factory)

	config := &ConnectionConfig{
		Timeout:     10 * time.Second,
		KeepAlive:   0, // Disable health checks for benchmark
		MaxAttempts: 3,
		PoolSize:    10,
		IdleTimeout: 5 * time.Minute,
	}

	brokerConfig := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092"},
	}

	ctx := context.Background()
	err := cm.Initialize(ctx, config, brokerConfig)
	require.NoError(b, err)
	defer cm.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, err := cm.GetConnection(ctx)
			if err != nil {
				b.Fatalf("Failed to get connection: %v", err)
			}

			err = cm.ReturnConnection(conn)
			if err != nil {
				b.Fatalf("Failed to return connection: %v", err)
			}
		}
	})
}

func BenchmarkConnectionRetryPolicy_ExecuteWithRetry(b *testing.B) {
	config := &ConnectionRetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Microsecond,
		MaxDelay:     time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	policy := NewConnectionRetryPolicy(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		policy.ExecuteWithRetry(ctx, func() error {
			return nil // Always succeed for benchmark
		})
	}
}

// Integration tests

func TestConnectionManagerIntegration(t *testing.T) {
	factory := NewMockConnectionFactory()
	cm := NewDefaultConnectionManager(factory)

	config := &ConnectionConfig{
		Timeout:     5 * time.Second,
		KeepAlive:   100 * time.Millisecond, // Fast health checks for testing
		MaxAttempts: 3,
		PoolSize:    3,
		IdleTimeout: 200 * time.Millisecond,
	}

	brokerConfig := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092", "localhost:9093"},
	}

	ctx := context.Background()
	err := cm.Initialize(ctx, config, brokerConfig)
	require.NoError(t, err)
	defer cm.Close()

	// Test concurrent access
	const numWorkers = 10
	const numOperations = 50

	var wg sync.WaitGroup
	errors := make(chan error, numWorkers)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Get connection
				conn, err := cm.GetConnection(ctx)
				if err != nil {
					errors <- fmt.Errorf("worker %d: failed to get connection: %w", workerID, err)
					return
				}

				// Simulate work
				time.Sleep(time.Millisecond)

				// Return connection
				err = cm.ReturnConnection(conn)
				if err != nil {
					errors <- fmt.Errorf("worker %d: failed to return connection: %w", workerID, err)
					return
				}
			}
		}(i)
	}

	// Wait for all workers to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case err := <-errors:
		t.Fatalf("Concurrent test failed: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("Test timed out")
	}

	// Verify final stats
	stats := cm.GetStats()
	assert.Equal(t, numWorkers*numOperations, int(stats.ConnectionsReused))
	assert.True(t, stats.ConnectionsCreated > 0)
	assert.Equal(t, 0, stats.ActiveCount) // All connections should be returned

	// Wait a bit for idle cleanup to potentially run
	time.Sleep(300 * time.Millisecond)

	// Verify health check ran
	health, err := cm.HealthCheck(ctx)
	require.NoError(t, err)
	assert.Equal(t, HealthStatusHealthy, health.Status)
}
