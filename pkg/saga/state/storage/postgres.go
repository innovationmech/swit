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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/state"
)

var (
	// ErrNotImplemented is returned when a method is not yet implemented.
	ErrNotImplemented = errors.New("not implemented")

	// ErrTransactionClosed is returned when an operation is attempted on a closed transaction.
	ErrTransactionClosed = errors.New("transaction is closed")

	// ErrTransactionAlreadyStarted is returned when trying to start a transaction that's already active.
	ErrTransactionAlreadyStarted = errors.New("transaction already started")

	// ErrConnectionPoolExhausted is returned when the connection pool has no available connections.
	ErrConnectionPoolExhausted = errors.New("connection pool exhausted")

	// ErrHealthCheckFailed is returned when a health check fails.
	ErrHealthCheckFailed = errors.New("health check failed")
)

// PostgresStateStorage provides a PostgreSQL implementation of saga.StateStorage.
// It stores Saga instances and their states in PostgreSQL database tables with
// support for transactions, complex queries, and strong consistency.
//
// The storage is thread-safe and supports concurrent access from multiple goroutines.
// All database operations are protected by connection pooling and proper transaction handling.
type PostgresStateStorage struct {
	// db is the PostgreSQL database connection pool
	db *sql.DB

	// config holds the PostgreSQL storage configuration
	config *PostgresConfig

	// mu protects concurrent access to storage state
	mu sync.RWMutex

	// closed indicates whether the storage has been closed
	closed bool

	// instancesTable is the full name of the saga_instances table
	instancesTable string

	// stepsTable is the full name of the saga_steps table
	stepsTable string

	// eventsTable is the full name of the saga_events table
	eventsTable string
}

// NewPostgresStateStorage creates a new PostgreSQL state storage with the specified configuration.
// It validates the configuration, establishes a database connection, and optionally runs
// schema migrations if AutoMigrate is enabled.
//
// Returns an error if:
//   - Configuration is invalid
//   - Database connection cannot be established
//   - Schema migration fails (if AutoMigrate is true)
func NewPostgresStateStorage(config *PostgresConfig) (*PostgresStateStorage, error) {
	if config == nil {
		config = DefaultPostgresConfig()
	}

	// Apply defaults to unset fields
	config.ApplyDefaults()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid postgres config: %w", err)
	}

	// Create PostgreSQL database connection
	db, err := sql.Open("postgres", config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Test database connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectionTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	storage := &PostgresStateStorage{
		db:             db,
		config:         config,
		closed:         false,
		instancesTable: config.GetInstancesTableName(),
		stepsTable:     config.GetStepsTableName(),
		eventsTable:    config.GetEventsTableName(),
	}

	// Run schema migrations if enabled
	if config.AutoMigrate {
		if err := storage.migrate(ctx); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to run schema migrations: %w", err)
		}
	}

	return storage, nil
}

// migrate runs database schema migrations.
// This is a placeholder for future implementation.
func (p *PostgresStateStorage) migrate(ctx context.Context) error {
	// TODO: Implement schema migration logic
	// For now, assume schema is already created via scripts/sql/saga_schema.sql
	return nil
}

// Close closes the database connection and releases resources.
// After calling Close, no further operations should be performed on this storage.
func (p *PostgresStateStorage) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	if p.db != nil {
		return p.db.Close()
	}

	return nil
}

// checkClosed returns an error if the storage has been closed.
func (p *PostgresStateStorage) checkClosed() error {
	if p.closed {
		return state.ErrStorageClosed
	}
	return nil
}

// ==========================
// Health Check and Monitoring
// ==========================

// HealthCheck performs a health check on the database connection.
// It verifies that the database is reachable and can execute a simple query.
//
// Returns an error if:
//   - The storage has been closed
//   - The database connection is not responsive
//   - A test query fails to execute
//
// The health check executes a lightweight query (SELECT 1) to verify connectivity
// and includes connection pool statistics in the result.
func (p *PostgresStateStorage) HealthCheck(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return err
	}

	// Create a timeout context for the health check
	healthCtx, cancel := context.WithTimeout(ctx, p.config.ConnectionTimeout)
	defer cancel()

	// Attempt to ping the database
	if err := p.db.PingContext(healthCtx); err != nil {
		return fmt.Errorf("%w: ping failed: %v", ErrHealthCheckFailed, err)
	}

	// Execute a simple query to verify database functionality
	var result int
	query := "SELECT 1"
	if err := p.db.QueryRowContext(healthCtx, query).Scan(&result); err != nil {
		return fmt.Errorf("%w: query failed: %v", ErrHealthCheckFailed, err)
	}

	if result != 1 {
		return fmt.Errorf("%w: unexpected query result: %d", ErrHealthCheckFailed, result)
	}

	return nil
}

// HealthCheckWithRetry performs a health check with automatic retry on failure.
// It uses the retry configuration from PostgresConfig to determine retry behavior.
//
// This is useful for startup scenarios or environments with transient connectivity issues.
func (p *PostgresStateStorage) HealthCheckWithRetry(ctx context.Context) error {
	var lastErr error

	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff duration with exponential backoff
			backoff := p.calculateBackoff(attempt)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				// Continue with retry
			}
		}

		if err := p.HealthCheck(ctx); err != nil {
			lastErr = err
			continue
		}

		// Health check succeeded
		return nil
	}

	return fmt.Errorf("health check failed after %d retries: %w", p.config.MaxRetries, lastErr)
}

// PostgresPoolStats returns connection pool statistics for monitoring.
// This provides visibility into connection pool health and usage patterns.
type PostgresPoolStats struct {
	// MaxOpenConnections is the maximum number of open connections to the database.
	MaxOpenConnections int `json:"max_open_connections"`

	// OpenConnections is the number of established connections both in use and idle.
	OpenConnections int `json:"open_connections"`

	// InUse is the number of connections currently in use.
	InUse int `json:"in_use"`

	// Idle is the number of idle connections.
	Idle int `json:"idle"`

	// WaitCount is the total number of connections waited for.
	WaitCount int64 `json:"wait_count"`

	// WaitDuration is the total time blocked waiting for a new connection.
	WaitDuration time.Duration `json:"wait_duration"`

	// MaxIdleClosed is the total number of connections closed due to SetMaxIdleConns.
	MaxIdleClosed int64 `json:"max_idle_closed"`

	// MaxIdleTimeClosed is the total number of connections closed due to SetConnMaxIdleTime.
	MaxIdleTimeClosed int64 `json:"max_idle_time_closed"`

	// MaxLifetimeClosed is the total number of connections closed due to SetConnMaxLifetime.
	MaxLifetimeClosed int64 `json:"max_lifetime_closed"`
}

// GetPoolStats returns current connection pool statistics.
// Returns an error if the storage has been closed.
func (p *PostgresStateStorage) GetPoolStats() (*PostgresPoolStats, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	stats := p.db.Stats()

	return &PostgresPoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}, nil
}

// DetectConnectionLeaks checks for potential connection leaks by analyzing pool statistics.
// A connection leak is suspected if:
//   - All connections are in use and wait count is high
//   - Wait duration is excessive
//
// Returns true if a potential leak is detected, along with diagnostic information.
func (p *PostgresStateStorage) DetectConnectionLeaks() (bool, string, error) {
	stats, err := p.GetPoolStats()
	if err != nil {
		return false, "", err
	}

	// Check if pool is exhausted
	if stats.OpenConnections >= stats.MaxOpenConnections && stats.InUse == stats.MaxOpenConnections {
		if stats.WaitCount > 100 {
			msg := fmt.Sprintf(
				"Potential connection leak detected: pool exhausted with %d connections in use, %d waits, %.2fs total wait time",
				stats.InUse, stats.WaitCount, stats.WaitDuration.Seconds(),
			)
			return true, msg, nil
		}

		// High wait duration suggests connections are not being released
		avgWaitTime := time.Duration(0)
		if stats.WaitCount > 0 {
			avgWaitTime = stats.WaitDuration / time.Duration(stats.WaitCount)
		}

		if avgWaitTime > time.Second {
			msg := fmt.Sprintf(
				"Potential connection leak detected: high average wait time %.2fs per acquisition",
				avgWaitTime.Seconds(),
			)
			return true, msg, nil
		}
	}

	return false, "", nil
}

// ==========================
// Retry Mechanism
// ==========================

// calculateBackoff calculates the backoff duration for a given retry attempt.
// Uses exponential backoff with a maximum cap.
func (p *PostgresStateStorage) calculateBackoff(attempt int) time.Duration {
	// Calculate exponential backoff: initialBackoff * 2^(attempt-1)
	backoff := p.config.RetryBackoff * time.Duration(1<<uint(attempt-1))

	// Cap at maximum backoff
	if backoff > p.config.MaxRetryBackoff {
		backoff = p.config.MaxRetryBackoff
	}

	return backoff
}

// isRetriableError determines if an error is transient and should be retried.
func (p *PostgresStateStorage) isRetriableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for context cancellation - don't retry these
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// PostgreSQL error codes that indicate transient failures
	// These are common errors that may resolve on retry
	errMsg := err.Error()

	// Connection errors
	if errors.Is(err, sql.ErrConnDone) {
		return true
	}

	// Common transient error patterns
	retriablePatterns := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"timeout",
		"deadlock",
		"too many connections",
		"could not serialize access",
	}

	for _, pattern := range retriablePatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// executeWithRetry executes a database operation with retry logic.
func (p *PostgresStateStorage) executeWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Check if the error is retriable
			if !p.isRetriableError(lastErr) {
				return lastErr
			}

			// Calculate and wait for backoff
			backoff := p.calculateBackoff(attempt)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				// Continue with retry
			}
		}

		// Execute the operation
		if err := operation(); err != nil {
			lastErr = err
			continue
		}

		// Operation succeeded
		return nil
	}

	return fmt.Errorf("operation failed after %d retries: %w", p.config.MaxRetries, lastErr)
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// SaveSaga persists a Saga instance to PostgreSQL storage.
// It uses INSERT ... ON CONFLICT DO UPDATE to handle both creation and update scenarios.
func (p *PostgresStateStorage) SaveSaga(ctx context.Context, instance saga.SagaInstance) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return err
	}

	if instance == nil {
		return state.ErrInvalidSagaID
	}

	sagaID := instance.GetID()
	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	// Convert SagaInstance to SagaInstanceData
	data := p.convertToSagaInstanceData(instance)

	// Serialize complex data fields to JSONB
	initialDataJSON, err := p.marshalJSON(data.InitialData)
	if err != nil {
		return fmt.Errorf("failed to marshal initial_data: %w", err)
	}

	currentDataJSON, err := p.marshalJSON(data.CurrentData)
	if err != nil {
		return fmt.Errorf("failed to marshal current_data: %w", err)
	}

	resultDataJSON, err := p.marshalJSON(data.ResultData)
	if err != nil {
		return fmt.Errorf("failed to marshal result_data: %w", err)
	}

	errorJSON, err := p.marshalJSON(data.Error)
	if err != nil {
		return fmt.Errorf("failed to marshal error: %w", err)
	}

	retryPolicyJSON, err := p.marshalJSON(data.RetryPolicy)
	if err != nil {
		return fmt.Errorf("failed to marshal retry_policy: %w", err)
	}

	metadataJSON, err := p.marshalJSON(data.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Convert timeout to milliseconds
	timeoutMs := data.Timeout.Milliseconds()

	// Prepare SQL statement with upsert logic
	query := fmt.Sprintf(`
		INSERT INTO %s (
			id, definition_id, name, description,
			state, current_step, total_steps,
			created_at, updated_at, started_at, completed_at, timed_out_at,
			initial_data, current_data, result_data,
			error, timeout_ms, retry_policy,
			metadata, trace_id, span_id, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		ON CONFLICT (id) DO UPDATE SET
			definition_id = EXCLUDED.definition_id,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			state = EXCLUDED.state,
			current_step = EXCLUDED.current_step,
			total_steps = EXCLUDED.total_steps,
			updated_at = EXCLUDED.updated_at,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at,
			timed_out_at = EXCLUDED.timed_out_at,
			initial_data = EXCLUDED.initial_data,
			current_data = EXCLUDED.current_data,
			result_data = EXCLUDED.result_data,
			error = EXCLUDED.error,
			timeout_ms = EXCLUDED.timeout_ms,
			retry_policy = EXCLUDED.retry_policy,
			metadata = EXCLUDED.metadata,
			trace_id = EXCLUDED.trace_id,
			span_id = EXCLUDED.span_id,
			version = EXCLUDED.version
	`, p.instancesTable)

	// Apply query timeout
	queryCtx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	// Set version to 1 if not set
	if data.Version == 0 {
		data.Version = 1
	}

	// Execute the query
	_, err = p.db.ExecContext(
		queryCtx,
		query,
		data.ID, data.DefinitionID, data.Name, data.Description,
		int(data.State), data.CurrentStep, data.TotalSteps,
		data.CreatedAt, data.UpdatedAt, data.StartedAt, data.CompletedAt, data.TimedOutAt,
		initialDataJSON, currentDataJSON, resultDataJSON,
		errorJSON, timeoutMs, retryPolicyJSON,
		metadataJSON, data.TraceID, data.SpanID, data.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to save saga instance: %w", err)
	}

	return nil
}

// GetSaga retrieves a Saga instance by its ID from PostgreSQL storage.
// Returns state.ErrSagaNotFound if the Saga does not exist.
func (p *PostgresStateStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	if sagaID == "" {
		return nil, state.ErrInvalidSagaID
	}

	// Prepare query
	query := fmt.Sprintf(`
		SELECT 
			id, definition_id, name, description,
			state, current_step, total_steps,
			created_at, updated_at, started_at, completed_at, timed_out_at,
			initial_data, current_data, result_data,
			error, timeout_ms, retry_policy,
			metadata, trace_id, span_id, version
		FROM %s
		WHERE id = $1
	`, p.instancesTable)

	// Apply query timeout
	queryCtx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	// Execute query
	var data saga.SagaInstanceData
	var stateInt int
	var timeoutMs int64
	var initialDataJSON, currentDataJSON, resultDataJSON []byte
	var errorJSON, retryPolicyJSON, metadataJSON []byte

	err := p.db.QueryRowContext(queryCtx, query, sagaID).Scan(
		&data.ID, &data.DefinitionID, &data.Name, &data.Description,
		&stateInt, &data.CurrentStep, &data.TotalSteps,
		&data.CreatedAt, &data.UpdatedAt, &data.StartedAt, &data.CompletedAt, &data.TimedOutAt,
		&initialDataJSON, &currentDataJSON, &resultDataJSON,
		&errorJSON, &timeoutMs, &retryPolicyJSON,
		&metadataJSON, &data.TraceID, &data.SpanID, &data.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, state.ErrSagaNotFound
		}
		return nil, fmt.Errorf("failed to query saga instance: %w", err)
	}

	// Convert state integer to SagaState
	data.State = saga.SagaState(stateInt)

	// Convert timeout from milliseconds
	data.Timeout = time.Duration(timeoutMs) * time.Millisecond

	// Unmarshal JSONB fields
	if err := p.unmarshalJSON(initialDataJSON, &data.InitialData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal initial_data: %w", err)
	}

	if err := p.unmarshalJSON(currentDataJSON, &data.CurrentData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal current_data: %w", err)
	}

	if err := p.unmarshalJSON(resultDataJSON, &data.ResultData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result_data: %w", err)
	}

	if err := p.unmarshalJSON(errorJSON, &data.Error); err != nil {
		return nil, fmt.Errorf("failed to unmarshal error: %w", err)
	}

	if err := p.unmarshalJSON(retryPolicyJSON, &data.RetryPolicy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal retry_policy: %w", err)
	}

	if err := p.unmarshalJSON(metadataJSON, &data.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Convert to SagaInstance
	return p.convertToSagaInstance(&data), nil
}

// UpdateSagaState updates only the state of a Saga instance.
// This is a lightweight operation that doesn't require loading the entire Saga.
func (p *PostgresStateStorage) UpdateSagaState(ctx context.Context, sagaID string, sagaState saga.SagaState, metadata map[string]interface{}) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return err
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	// Apply query timeout
	queryCtx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	// If metadata is provided, merge it with existing metadata
	if len(metadata) > 0 {
		// Serialize new metadata
		metadataJSON, err := p.marshalJSON(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		// Use JSONB concatenation to merge metadata
		query := fmt.Sprintf(`
			UPDATE %s
			SET state = $1, 
			    updated_at = $2,
			    metadata = COALESCE(metadata, '{}'::jsonb) || $3::jsonb
			WHERE id = $4
		`, p.instancesTable)

		result, err := p.db.ExecContext(queryCtx, query, int(sagaState), time.Now(), metadataJSON, sagaID)
		if err != nil {
			return fmt.Errorf("failed to update saga state: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return state.ErrSagaNotFound
		}
	} else {
		// No metadata to update, just update state
		query := fmt.Sprintf(`
			UPDATE %s
			SET state = $1, updated_at = $2
			WHERE id = $3
		`, p.instancesTable)

		result, err := p.db.ExecContext(queryCtx, query, int(sagaState), time.Now(), sagaID)
		if err != nil {
			return fmt.Errorf("failed to update saga state: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return state.ErrSagaNotFound
		}
	}

	return nil
}

// DeleteSaga removes a Saga instance from PostgreSQL storage.
// Returns state.ErrSagaNotFound if the Saga does not exist.
// Due to CASCADE constraints, this will also delete associated steps and events.
func (p *PostgresStateStorage) DeleteSaga(ctx context.Context, sagaID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return err
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	// Prepare delete query
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, p.instancesTable)

	// Apply query timeout
	queryCtx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	// Execute delete
	result, err := p.db.ExecContext(queryCtx, query, sagaID)
	if err != nil {
		return fmt.Errorf("failed to delete saga instance: %w", err)
	}

	// Check if saga was found and deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return state.ErrSagaNotFound
	}

	return nil
}

// GetActiveSagas retrieves all active Saga instances based on the filter.
// It supports complex filtering by state, definition ID, time range, and metadata,
// as well as pagination and sorting capabilities.
func (p *PostgresStateStorage) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	// Build the query with WHERE clause based on filter
	query, args := p.buildListQuery(filter)

	// Apply query timeout
	queryCtx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	// Execute query
	rows, err := p.db.QueryContext(queryCtx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query active sagas: %w", err)
	}
	defer rows.Close()

	// Parse results
	instances, err := p.scanSagaInstances(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan saga instances: %w", err)
	}

	return instances, nil
}

// GetTimeoutSagas retrieves Saga instances that have timed out before the specified time.
// It queries for Sagas in running states that have exceeded their timeout or have an explicit timeout timestamp.
func (p *PostgresStateStorage) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	// Query for Sagas that have timed out
	// Check both explicitly set timed_out_at field and calculated timeouts
	query := fmt.Sprintf(`
		SELECT 
			id, definition_id, name, description,
			state, current_step, total_steps,
			created_at, updated_at, started_at, completed_at, timed_out_at,
			initial_data, current_data, result_data,
			error, timeout_ms, retry_policy,
			metadata, trace_id, span_id, version
		FROM %s
		WHERE (
			timed_out_at IS NOT NULL AND timed_out_at < $1
		) OR (
			state IN (%d, %d, %d) 
			AND started_at IS NOT NULL 
			AND timeout_ms > 0
			AND (started_at + (timeout_ms || ' milliseconds')::INTERVAL) < $1
		)
		ORDER BY created_at ASC
	`, p.instancesTable, saga.StateRunning, saga.StateStepCompleted, saga.StateCompensating)

	// Apply query timeout
	queryCtx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	// Execute query
	rows, err := p.db.QueryContext(queryCtx, query, before)
	if err != nil {
		return nil, fmt.Errorf("failed to query timeout sagas: %w", err)
	}
	defer rows.Close()

	// Parse results
	instances, err := p.scanSagaInstances(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan saga instances: %w", err)
	}

	return instances, nil
}

// SaveStepState persists the state of a specific step within a Saga.
// It uses INSERT ... ON CONFLICT DO UPDATE to handle both creation and update scenarios.
func (p *PostgresStateStorage) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return err
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	if step == nil {
		return state.ErrInvalidState
	}

	// Serialize complex data fields to JSONB
	inputDataJSON, err := p.marshalJSON(step.InputData)
	if err != nil {
		return fmt.Errorf("failed to marshal input_data: %w", err)
	}

	outputDataJSON, err := p.marshalJSON(step.OutputData)
	if err != nil {
		return fmt.Errorf("failed to marshal output_data: %w", err)
	}

	errorJSON, err := p.marshalJSON(step.Error)
	if err != nil {
		return fmt.Errorf("failed to marshal error: %w", err)
	}

	compensationStateJSON, err := p.marshalJSON(step.CompensationState)
	if err != nil {
		return fmt.Errorf("failed to marshal compensation_state: %w", err)
	}

	metadataJSON, err := p.marshalJSON(step.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Prepare SQL statement with upsert logic
	query := fmt.Sprintf(`
		INSERT INTO %s (
			id, saga_id, step_index, name,
			state, attempts, max_attempts,
			created_at, started_at, completed_at, last_attempt_at,
			input_data, output_data, error,
			compensation_state, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE SET
			saga_id = EXCLUDED.saga_id,
			step_index = EXCLUDED.step_index,
			name = EXCLUDED.name,
			state = EXCLUDED.state,
			attempts = EXCLUDED.attempts,
			max_attempts = EXCLUDED.max_attempts,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at,
			last_attempt_at = EXCLUDED.last_attempt_at,
			input_data = EXCLUDED.input_data,
			output_data = EXCLUDED.output_data,
			error = EXCLUDED.error,
			compensation_state = EXCLUDED.compensation_state,
			metadata = EXCLUDED.metadata
	`, p.stepsTable)

	// Apply query timeout
	queryCtx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	// Execute the query
	_, err = p.db.ExecContext(
		queryCtx,
		query,
		step.ID, sagaID, step.StepIndex, step.Name,
		int(step.State), step.Attempts, step.MaxAttempts,
		step.CreatedAt, step.StartedAt, step.CompletedAt, step.LastAttemptAt,
		inputDataJSON, outputDataJSON, errorJSON,
		compensationStateJSON, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to save step state: %w", err)
	}

	return nil
}

// GetStepStates retrieves all step states for a Saga instance.
// Returns an empty slice if no steps are found.
func (p *PostgresStateStorage) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	if sagaID == "" {
		return nil, state.ErrInvalidSagaID
	}

	// Prepare query
	query := fmt.Sprintf(`
		SELECT 
			id, saga_id, step_index, name,
			state, attempts, max_attempts,
			created_at, started_at, completed_at, last_attempt_at,
			input_data, output_data, error,
			compensation_state, metadata
		FROM %s
		WHERE saga_id = $1
		ORDER BY step_index ASC
	`, p.stepsTable)

	// Apply query timeout
	queryCtx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	// Execute query
	rows, err := p.db.QueryContext(queryCtx, query, sagaID)
	if err != nil {
		return nil, fmt.Errorf("failed to query step states: %w", err)
	}
	defer rows.Close()

	var steps []*saga.StepState

	for rows.Next() {
		var step saga.StepState
		var stateInt int
		var inputDataJSON, outputDataJSON, errorJSON []byte
		var compensationStateJSON, metadataJSON []byte

		err := rows.Scan(
			&step.ID, &step.SagaID, &step.StepIndex, &step.Name,
			&stateInt, &step.Attempts, &step.MaxAttempts,
			&step.CreatedAt, &step.StartedAt, &step.CompletedAt, &step.LastAttemptAt,
			&inputDataJSON, &outputDataJSON, &errorJSON,
			&compensationStateJSON, &metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan step state: %w", err)
		}

		// Convert state integer to StepStateEnum
		step.State = saga.StepStateEnum(stateInt)

		// Unmarshal JSONB fields
		if err := p.unmarshalJSON(inputDataJSON, &step.InputData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input_data: %w", err)
		}

		if err := p.unmarshalJSON(outputDataJSON, &step.OutputData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal output_data: %w", err)
		}

		if err := p.unmarshalJSON(errorJSON, &step.Error); err != nil {
			return nil, fmt.Errorf("failed to unmarshal error: %w", err)
		}

		if err := p.unmarshalJSON(compensationStateJSON, &step.CompensationState); err != nil {
			return nil, fmt.Errorf("failed to unmarshal compensation_state: %w", err)
		}

		if err := p.unmarshalJSON(metadataJSON, &step.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		steps = append(steps, &step)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating step states: %w", err)
	}

	return steps, nil
}

// CleanupExpiredSagas removes Saga instances that are older than the specified time.
// This method will be implemented in a future task.
func (p *PostgresStateStorage) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return err
	}

	// TODO: Implement PostgreSQL cleanup logic for expired sagas
	return ErrNotImplemented
}

// marshalJSON marshals a value to JSON bytes.
// Returns nil bytes for nil values.
func (p *PostgresStateStorage) marshalJSON(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

// unmarshalJSON unmarshals JSON bytes to a value.
// Skips unmarshaling for nil or empty bytes.
func (p *PostgresStateStorage) unmarshalJSON(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

// convertToSagaInstanceData converts a SagaInstance to SagaInstanceData for storage.
func (p *PostgresStateStorage) convertToSagaInstanceData(instance saga.SagaInstance) *saga.SagaInstanceData {
	now := time.Now()
	startTime := instance.GetStartTime()
	endTime := instance.GetEndTime()

	data := &saga.SagaInstanceData{
		ID:           instance.GetID(),
		DefinitionID: instance.GetDefinitionID(),
		State:        instance.GetState(),
		CurrentStep:  instance.GetCurrentStep(),
		TotalSteps:   instance.GetTotalSteps(),
		CreatedAt:    instance.GetCreatedAt(),
		UpdatedAt:    instance.GetUpdatedAt(),
		Timeout:      instance.GetTimeout(),
		Metadata:     instance.GetMetadata(),
		TraceID:      instance.GetTraceID(),
		Error:        instance.GetError(),
		ResultData:   instance.GetResult(),
	}

	// Set started time if available
	if !startTime.IsZero() {
		data.StartedAt = &startTime
	}

	// Set completed time if available
	if !endTime.IsZero() {
		if instance.GetState() == saga.StateTimedOut {
			data.TimedOutAt = &endTime
		} else {
			data.CompletedAt = &endTime
		}
	}

	// Update UpdatedAt if it's zero
	if data.UpdatedAt.IsZero() {
		data.UpdatedAt = now
	}

	return data
}

// convertToSagaInstance converts SagaInstanceData to a SagaInstance implementation.
func (p *PostgresStateStorage) convertToSagaInstance(data *saga.SagaInstanceData) saga.SagaInstance {
	return &postgresSagaInstance{
		data: data,
	}
}

// postgresSagaInstance is a PostgreSQL-backed implementation of saga.SagaInstance.
type postgresSagaInstance struct {
	data *saga.SagaInstanceData
}

// GetID returns the unique identifier of this Saga instance.
func (p *postgresSagaInstance) GetID() string {
	return p.data.ID
}

// GetDefinitionID returns the identifier of the Saga definition this instance follows.
func (p *postgresSagaInstance) GetDefinitionID() string {
	return p.data.DefinitionID
}

// GetState returns the current state of the Saga instance.
func (p *postgresSagaInstance) GetState() saga.SagaState {
	return p.data.State
}

// GetCurrentStep returns the index of the currently executing step.
func (p *postgresSagaInstance) GetCurrentStep() int {
	return p.data.CurrentStep
}

// GetStartTime returns the time when the Saga instance was created.
func (p *postgresSagaInstance) GetStartTime() time.Time {
	if p.data.StartedAt != nil {
		return *p.data.StartedAt
	}
	return time.Time{}
}

// GetEndTime returns the time when the Saga instance reached a terminal state.
func (p *postgresSagaInstance) GetEndTime() time.Time {
	if p.data.CompletedAt != nil {
		return *p.data.CompletedAt
	}
	if p.data.TimedOutAt != nil {
		return *p.data.TimedOutAt
	}
	return time.Time{}
}

// GetResult returns the final result data of the Saga execution.
func (p *postgresSagaInstance) GetResult() interface{} {
	return p.data.ResultData
}

// GetError returns the error that caused the Saga to fail.
func (p *postgresSagaInstance) GetError() *saga.SagaError {
	return p.data.Error
}

// GetTotalSteps returns the total number of steps in the Saga definition.
func (p *postgresSagaInstance) GetTotalSteps() int {
	return p.data.TotalSteps
}

// GetCompletedSteps returns the number of steps that have completed successfully.
func (p *postgresSagaInstance) GetCompletedSteps() int {
	return p.data.CurrentStep
}

// GetCreatedAt returns the creation time of the Saga instance.
func (p *postgresSagaInstance) GetCreatedAt() time.Time {
	return p.data.CreatedAt
}

// GetUpdatedAt returns the last update time of the Saga instance.
func (p *postgresSagaInstance) GetUpdatedAt() time.Time {
	return p.data.UpdatedAt
}

// GetTimeout returns the timeout duration for this Saga instance.
func (p *postgresSagaInstance) GetTimeout() time.Duration {
	return p.data.Timeout
}

// GetMetadata returns the metadata associated with this Saga instance.
func (p *postgresSagaInstance) GetMetadata() map[string]interface{} {
	return p.data.Metadata
}

// GetTraceID returns the distributed tracing identifier for this Saga.
func (p *postgresSagaInstance) GetTraceID() string {
	return p.data.TraceID
}

// IsTerminal returns true if the Saga is in a terminal state.
func (p *postgresSagaInstance) IsTerminal() bool {
	return p.data.State.IsTerminal()
}

// IsActive returns true if the Saga is currently active.
func (p *postgresSagaInstance) IsActive() bool {
	return p.data.State.IsActive()
}

// ==========================
// Query Builder Methods
// ==========================

// buildListQuery constructs a SQL query with WHERE clause based on the provided filter.
// It returns the complete SQL query string and the corresponding arguments for parameterized execution.
func (p *PostgresStateStorage) buildListQuery(filter *saga.SagaFilter) (string, []interface{}) {
	// Base SELECT statement
	baseQuery := fmt.Sprintf(`
		SELECT 
			id, definition_id, name, description,
			state, current_step, total_steps,
			created_at, updated_at, started_at, completed_at, timed_out_at,
			initial_data, current_data, result_data,
			error, timeout_ms, retry_policy,
			metadata, trace_id, span_id, version
		FROM %s`, p.instancesTable)

	// Build WHERE clause and collect arguments
	whereClauses, args := p.buildWhereClause(filter)

	// Combine base query with WHERE clause
	if len(whereClauses) > 0 {
		baseQuery += "\nWHERE " + whereClauses
	}

	// Add ORDER BY clause
	baseQuery += p.buildOrderByClause(filter)

	// Add LIMIT and OFFSET for pagination
	baseQuery, args = p.buildPaginationClause(baseQuery, args, filter)

	return baseQuery, args
}

// buildWhereClause constructs the WHERE clause and arguments based on the filter.
// Returns the WHERE clause string (without "WHERE" keyword) and the arguments slice.
func (p *PostgresStateStorage) buildWhereClause(filter *saga.SagaFilter) (string, []interface{}) {
	if filter == nil {
		return "", nil
	}

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Filter by state
	if len(filter.States) > 0 {
		statePlaceholders := make([]string, len(filter.States))
		for i, state := range filter.States {
			statePlaceholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, int(state))
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("state IN (%s)", joinStrings(statePlaceholders, ", ")))
	}

	// Filter by definition ID
	if len(filter.DefinitionIDs) > 0 {
		defPlaceholders := make([]string, len(filter.DefinitionIDs))
		for i, defID := range filter.DefinitionIDs {
			defPlaceholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, defID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("definition_id IN (%s)", joinStrings(defPlaceholders, ", ")))
	}

	// Filter by creation time range
	if filter.CreatedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.CreatedAfter)
		argIndex++
	}

	if filter.CreatedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.CreatedBefore)
		argIndex++
	}

	// Filter by metadata (JSONB containment)
	if len(filter.Metadata) > 0 {
		metadataJSON, err := json.Marshal(filter.Metadata)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("metadata @> $%d::jsonb", argIndex))
			args = append(args, metadataJSON)
			argIndex++
		}
	}

	// Join all conditions with AND
	if len(conditions) == 0 {
		return "", args
	}

	return joinStrings(conditions, " AND "), args
}

// buildOrderByClause constructs the ORDER BY clause based on the filter.
func (p *PostgresStateStorage) buildOrderByClause(filter *saga.SagaFilter) string {
	if filter == nil || filter.SortBy == "" {
		// Default sorting by created_at descending
		return "\nORDER BY created_at DESC"
	}

	// Validate and sanitize sort field to prevent SQL injection
	sortField := p.sanitizeSortField(filter.SortBy)
	sortOrder := "DESC"
	if filter.SortOrder == "asc" || filter.SortOrder == "ASC" {
		sortOrder = "ASC"
	}

	return fmt.Sprintf("\nORDER BY %s %s", sortField, sortOrder)
}

// sanitizeSortField validates and returns a safe sort field name.
// Only whitelisted fields are allowed to prevent SQL injection.
func (p *PostgresStateStorage) sanitizeSortField(field string) string {
	// Whitelist of allowed sort fields
	allowedFields := map[string]string{
		"id":            "id",
		"created_at":    "created_at",
		"updated_at":    "updated_at",
		"started_at":    "started_at",
		"completed_at":  "completed_at",
		"state":         "state",
		"definition_id": "definition_id",
		"current_step":  "current_step",
	}

	if safeField, ok := allowedFields[field]; ok {
		return safeField
	}

	// Default to created_at if invalid field specified
	return "created_at"
}

// buildPaginationClause adds LIMIT and OFFSET to the query for pagination.
func (p *PostgresStateStorage) buildPaginationClause(query string, args []interface{}, filter *saga.SagaFilter) (string, []interface{}) {
	if filter == nil {
		return query, args
	}

	argIndex := len(args) + 1

	// Add LIMIT
	if filter.Limit > 0 {
		query += fmt.Sprintf("\nLIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	// Add OFFSET
	if filter.Offset > 0 {
		query += fmt.Sprintf("\nOFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	return query, args
}

// scanSagaInstances scans multiple Saga instances from SQL query results.
func (p *PostgresStateStorage) scanSagaInstances(rows *sql.Rows) ([]saga.SagaInstance, error) {
	var instances []saga.SagaInstance

	for rows.Next() {
		var data saga.SagaInstanceData
		var stateInt int
		var timeoutMs int64
		var initialDataJSON, currentDataJSON, resultDataJSON []byte
		var errorJSON, retryPolicyJSON, metadataJSON []byte

		err := rows.Scan(
			&data.ID, &data.DefinitionID, &data.Name, &data.Description,
			&stateInt, &data.CurrentStep, &data.TotalSteps,
			&data.CreatedAt, &data.UpdatedAt, &data.StartedAt, &data.CompletedAt, &data.TimedOutAt,
			&initialDataJSON, &currentDataJSON, &resultDataJSON,
			&errorJSON, &timeoutMs, &retryPolicyJSON,
			&metadataJSON, &data.TraceID, &data.SpanID, &data.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan saga instance: %w", err)
		}

		// Convert state integer to SagaState
		data.State = saga.SagaState(stateInt)

		// Convert timeout from milliseconds
		data.Timeout = time.Duration(timeoutMs) * time.Millisecond

		// Unmarshal JSONB fields
		if err := p.unmarshalJSON(initialDataJSON, &data.InitialData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal initial_data: %w", err)
		}

		if err := p.unmarshalJSON(currentDataJSON, &data.CurrentData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal current_data: %w", err)
		}

		if err := p.unmarshalJSON(resultDataJSON, &data.ResultData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result_data: %w", err)
		}

		if err := p.unmarshalJSON(errorJSON, &data.Error); err != nil {
			return nil, fmt.Errorf("failed to unmarshal error: %w", err)
		}

		if err := p.unmarshalJSON(retryPolicyJSON, &data.RetryPolicy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal retry_policy: %w", err)
		}

		if err := p.unmarshalJSON(metadataJSON, &data.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		// Convert to SagaInstance and add to list
		instances = append(instances, p.convertToSagaInstance(&data))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating saga instances: %w", err)
	}

	return instances, nil
}

// joinStrings joins a slice of strings with a separator.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// ==========================
// Helper Query Methods
// ==========================

// ListSagas retrieves a list of Saga instances with filtering, sorting, and pagination.
// This is a convenience wrapper around GetActiveSagas with a more intuitive name.
func (p *PostgresStateStorage) ListSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	return p.GetActiveSagas(ctx, filter)
}

// FindByStatus retrieves Saga instances filtered by their state.
// It's a convenience method that creates a filter with only state criteria.
func (p *PostgresStateStorage) FindByStatus(ctx context.Context, states []saga.SagaState, limit, offset int) ([]saga.SagaInstance, error) {
	filter := &saga.SagaFilter{
		States: states,
		Limit:  limit,
		Offset: offset,
	}
	return p.GetActiveSagas(ctx, filter)
}

// FindByTimeRange retrieves Saga instances created within a specified time range.
// It's a convenience method that creates a filter with time range criteria.
func (p *PostgresStateStorage) FindByTimeRange(ctx context.Context, after, before *time.Time, limit, offset int) ([]saga.SagaInstance, error) {
	filter := &saga.SagaFilter{
		CreatedAfter:  after,
		CreatedBefore: before,
		Limit:         limit,
		Offset:        offset,
	}
	return p.GetActiveSagas(ctx, filter)
}

// CountSagas returns the total count of Saga instances matching the filter.
// This is useful for implementing pagination with total count.
func (p *PostgresStateStorage) CountSagas(ctx context.Context, filter *saga.SagaFilter) (int64, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return 0, err
	}

	// Build WHERE clause
	whereClauses, args := p.buildWhereClause(filter)

	// Build count query
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", p.instancesTable)
	if len(whereClauses) > 0 {
		query += "\nWHERE " + whereClauses
	}

	// Apply query timeout
	queryCtx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	// Execute count query
	var count int64
	err := p.db.QueryRowContext(queryCtx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count sagas: %w", err)
	}

	return count, nil
}

// ==========================
// Transaction Support
// ==========================

// SagaTransaction represents a database transaction for atomic Saga operations.
// It allows multiple Saga and step state operations to be grouped together
// and committed or rolled back atomically, ensuring data consistency.
//
// The transaction is not thread-safe and should be used from a single goroutine.
type SagaTransaction struct {
	// tx is the underlying SQL transaction
	tx *sql.Tx

	// storage is the parent storage instance
	storage *PostgresStateStorage

	// closed indicates whether the transaction has been committed or rolled back
	closed bool

	// mu protects transaction state
	mu sync.Mutex

	// timeout is the transaction timeout duration
	timeout time.Duration

	// startTime records when the transaction was started
	startTime time.Time
}

// BeginTransaction starts a new database transaction with the specified timeout.
// Returns a SagaTransaction that can be used to perform multiple operations atomically.
//
// The transaction must be explicitly committed using Commit() or rolled back using Rollback().
// Failure to do so will leak database connections.
//
// Example:
//
//	tx, err := storage.BeginTransaction(ctx, 30*time.Second)
//	if err != nil {
//	    return err
//	}
//	defer tx.Rollback(ctx) // Ensure rollback on error
//
//	if err := tx.SaveSaga(ctx, saga1); err != nil {
//	    return err
//	}
//	if err := tx.SaveSaga(ctx, saga2); err != nil {
//	    return err
//	}
//
//	return tx.Commit(ctx)
func (p *PostgresStateStorage) BeginTransaction(ctx context.Context, timeout time.Duration) (*SagaTransaction, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	// Use default timeout if not specified
	if timeout <= 0 {
		timeout = p.config.QueryTimeout
	}

	// Create transaction context with timeout
	txCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Begin SQL transaction
	tx, err := p.db.BeginTx(txCtx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &SagaTransaction{
		tx:        tx,
		storage:   p,
		closed:    false,
		timeout:   timeout,
		startTime: time.Now(),
	}, nil
}

// checkClosed returns an error if the transaction has been closed.
func (st *SagaTransaction) checkClosed() error {
	if st.closed {
		return ErrTransactionClosed
	}
	return nil
}

// checkTimeout returns an error if the transaction has exceeded its timeout.
func (st *SagaTransaction) checkTimeout() error {
	if time.Since(st.startTime) > st.timeout {
		return fmt.Errorf("transaction timeout exceeded: %v", st.timeout)
	}
	return nil
}

// Commit commits the transaction, making all changes permanent.
// After commit, the transaction is closed and cannot be used anymore.
//
// Returns an error if:
//   - The transaction is already closed
//   - The transaction has timed out
//   - The commit operation fails
func (st *SagaTransaction) Commit(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	if err := st.checkClosed(); err != nil {
		return err
	}

	if err := st.checkTimeout(); err != nil {
		// Rollback on timeout
		_ = st.tx.Rollback()
		st.closed = true
		return err
	}

	// Commit the transaction
	if err := st.tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	st.closed = true
	return nil
}

// Rollback rolls back the transaction, discarding all changes.
// After rollback, the transaction is closed and cannot be used anymore.
//
// Returns an error if:
//   - The transaction is already closed
//   - The rollback operation fails
//
// Note: Rollback is idempotent and safe to call multiple times.
func (st *SagaTransaction) Rollback(ctx context.Context) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.closed {
		// Already closed, no-op
		return nil
	}

	// Rollback the transaction
	if err := st.tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	st.closed = true
	return nil
}

// SaveSaga persists a Saga instance within the transaction.
// It uses INSERT ... ON CONFLICT DO UPDATE to handle both creation and update scenarios.
func (st *SagaTransaction) SaveSaga(ctx context.Context, instance saga.SagaInstance) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	if err := st.checkClosed(); err != nil {
		return err
	}

	if err := st.checkTimeout(); err != nil {
		return err
	}

	if instance == nil {
		return state.ErrInvalidSagaID
	}

	sagaID := instance.GetID()
	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	// Convert SagaInstance to SagaInstanceData
	data := st.storage.convertToSagaInstanceData(instance)

	// Serialize complex data fields to JSONB
	initialDataJSON, err := st.storage.marshalJSON(data.InitialData)
	if err != nil {
		return fmt.Errorf("failed to marshal initial_data: %w", err)
	}

	currentDataJSON, err := st.storage.marshalJSON(data.CurrentData)
	if err != nil {
		return fmt.Errorf("failed to marshal current_data: %w", err)
	}

	resultDataJSON, err := st.storage.marshalJSON(data.ResultData)
	if err != nil {
		return fmt.Errorf("failed to marshal result_data: %w", err)
	}

	errorJSON, err := st.storage.marshalJSON(data.Error)
	if err != nil {
		return fmt.Errorf("failed to marshal error: %w", err)
	}

	retryPolicyJSON, err := st.storage.marshalJSON(data.RetryPolicy)
	if err != nil {
		return fmt.Errorf("failed to marshal retry_policy: %w", err)
	}

	metadataJSON, err := st.storage.marshalJSON(data.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Convert timeout to milliseconds
	timeoutMs := data.Timeout.Milliseconds()

	// Set version to 1 if not set
	if data.Version == 0 {
		data.Version = 1
	}

	// Prepare SQL statement with upsert logic
	query := fmt.Sprintf(`
		INSERT INTO %s (
			id, definition_id, name, description,
			state, current_step, total_steps,
			created_at, updated_at, started_at, completed_at, timed_out_at,
			initial_data, current_data, result_data,
			error, timeout_ms, retry_policy,
			metadata, trace_id, span_id, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		ON CONFLICT (id) DO UPDATE SET
			definition_id = EXCLUDED.definition_id,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			state = EXCLUDED.state,
			current_step = EXCLUDED.current_step,
			total_steps = EXCLUDED.total_steps,
			updated_at = EXCLUDED.updated_at,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at,
			timed_out_at = EXCLUDED.timed_out_at,
			initial_data = EXCLUDED.initial_data,
			current_data = EXCLUDED.current_data,
			result_data = EXCLUDED.result_data,
			error = EXCLUDED.error,
			timeout_ms = EXCLUDED.timeout_ms,
			retry_policy = EXCLUDED.retry_policy,
			metadata = EXCLUDED.metadata,
			trace_id = EXCLUDED.trace_id,
			span_id = EXCLUDED.span_id,
			version = EXCLUDED.version
	`, st.storage.instancesTable)

	// Execute the query within transaction
	_, err = st.tx.ExecContext(
		ctx,
		query,
		data.ID, data.DefinitionID, data.Name, data.Description,
		int(data.State), data.CurrentStep, data.TotalSteps,
		data.CreatedAt, data.UpdatedAt, data.StartedAt, data.CompletedAt, data.TimedOutAt,
		initialDataJSON, currentDataJSON, resultDataJSON,
		errorJSON, timeoutMs, retryPolicyJSON,
		metadataJSON, data.TraceID, data.SpanID, data.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to save saga instance in transaction: %w", err)
	}

	return nil
}

// SaveStepState persists the state of a specific step within a Saga in the transaction.
// It uses INSERT ... ON CONFLICT DO UPDATE to handle both creation and update scenarios.
func (st *SagaTransaction) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	if err := st.checkClosed(); err != nil {
		return err
	}

	if err := st.checkTimeout(); err != nil {
		return err
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	if step == nil {
		return state.ErrInvalidState
	}

	// Serialize complex data fields to JSONB
	inputDataJSON, err := st.storage.marshalJSON(step.InputData)
	if err != nil {
		return fmt.Errorf("failed to marshal input_data: %w", err)
	}

	outputDataJSON, err := st.storage.marshalJSON(step.OutputData)
	if err != nil {
		return fmt.Errorf("failed to marshal output_data: %w", err)
	}

	errorJSON, err := st.storage.marshalJSON(step.Error)
	if err != nil {
		return fmt.Errorf("failed to marshal error: %w", err)
	}

	compensationStateJSON, err := st.storage.marshalJSON(step.CompensationState)
	if err != nil {
		return fmt.Errorf("failed to marshal compensation_state: %w", err)
	}

	metadataJSON, err := st.storage.marshalJSON(step.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Prepare SQL statement with upsert logic
	query := fmt.Sprintf(`
		INSERT INTO %s (
			id, saga_id, step_index, name,
			state, attempts, max_attempts,
			created_at, started_at, completed_at, last_attempt_at,
			input_data, output_data, error,
			compensation_state, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE SET
			saga_id = EXCLUDED.saga_id,
			step_index = EXCLUDED.step_index,
			name = EXCLUDED.name,
			state = EXCLUDED.state,
			attempts = EXCLUDED.attempts,
			max_attempts = EXCLUDED.max_attempts,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at,
			last_attempt_at = EXCLUDED.last_attempt_at,
			input_data = EXCLUDED.input_data,
			output_data = EXCLUDED.output_data,
			error = EXCLUDED.error,
			compensation_state = EXCLUDED.compensation_state,
			metadata = EXCLUDED.metadata
	`, st.storage.stepsTable)

	// Execute the query within transaction
	_, err = st.tx.ExecContext(
		ctx,
		query,
		step.ID, sagaID, step.StepIndex, step.Name,
		int(step.State), step.Attempts, step.MaxAttempts,
		step.CreatedAt, step.StartedAt, step.CompletedAt, step.LastAttemptAt,
		inputDataJSON, outputDataJSON, errorJSON,
		compensationStateJSON, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to save step state in transaction: %w", err)
	}

	return nil
}

// UpdateSagaState updates only the state of a Saga instance within the transaction.
// This is a lightweight operation that doesn't require loading the entire Saga.
func (st *SagaTransaction) UpdateSagaState(ctx context.Context, sagaID string, sagaState saga.SagaState, metadata map[string]interface{}) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	if err := st.checkClosed(); err != nil {
		return err
	}

	if err := st.checkTimeout(); err != nil {
		return err
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	// If metadata is provided, merge it with existing metadata
	if len(metadata) > 0 {
		// Serialize new metadata
		metadataJSON, err := st.storage.marshalJSON(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		// Use JSONB concatenation to merge metadata
		query := fmt.Sprintf(`
			UPDATE %s
			SET state = $1, 
			    updated_at = $2,
			    metadata = COALESCE(metadata, '{}'::jsonb) || $3::jsonb
			WHERE id = $4
		`, st.storage.instancesTable)

		result, err := st.tx.ExecContext(ctx, query, int(sagaState), time.Now(), metadataJSON, sagaID)
		if err != nil {
			return fmt.Errorf("failed to update saga state in transaction: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return state.ErrSagaNotFound
		}
	} else {
		// No metadata to update, just update state
		query := fmt.Sprintf(`
			UPDATE %s
			SET state = $1, updated_at = $2
			WHERE id = $3
		`, st.storage.instancesTable)

		result, err := st.tx.ExecContext(ctx, query, int(sagaState), time.Now(), sagaID)
		if err != nil {
			return fmt.Errorf("failed to update saga state in transaction: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return state.ErrSagaNotFound
		}
	}

	return nil
}

// DeleteSaga removes a Saga instance from storage within the transaction.
// Due to CASCADE constraints, this will also delete associated steps and events.
func (st *SagaTransaction) DeleteSaga(ctx context.Context, sagaID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	if err := st.checkClosed(); err != nil {
		return err
	}

	if err := st.checkTimeout(); err != nil {
		return err
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	// Prepare delete query
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, st.storage.instancesTable)

	// Execute delete within transaction
	result, err := st.tx.ExecContext(ctx, query, sagaID)
	if err != nil {
		return fmt.Errorf("failed to delete saga instance in transaction: %w", err)
	}

	// Check if saga was found and deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return state.ErrSagaNotFound
	}

	return nil
}

// Exec executes a custom SQL query within the transaction.
// This allows performing additional database operations atomically with Saga operations.
//
// Returns the number of rows affected and any error encountered.
func (st *SagaTransaction) Exec(ctx context.Context, query string, args ...interface{}) (int64, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	if err := st.checkClosed(); err != nil {
		return 0, err
	}

	if err := st.checkTimeout(); err != nil {
		return 0, err
	}

	result, err := st.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query in transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// ==========================
// Batch Operations with Transactions
// ==========================

// BatchSaveSagas saves multiple Saga instances atomically within a single transaction.
// All Sagas are saved or none are saved.
//
// This is a convenience method that wraps BeginTransaction, multiple SaveSaga calls,
// and Commit/Rollback.
func (p *PostgresStateStorage) BatchSaveSagas(ctx context.Context, instances []saga.SagaInstance, timeout time.Duration) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if len(instances) == 0 {
		return nil
	}

	// Begin transaction
	tx, err := p.BeginTransaction(ctx, timeout)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Save all instances
	for _, instance := range instances {
		if err := tx.SaveSaga(ctx, instance); err != nil {
			return fmt.Errorf("failed to save saga %s in batch: %w", instance.GetID(), err)
		}
	}

	// Commit transaction
	return tx.Commit(ctx)
}

// BatchSaveStepStates saves multiple step states for a Saga atomically within a single transaction.
// All steps are saved or none are saved.
func (p *PostgresStateStorage) BatchSaveStepStates(ctx context.Context, sagaID string, steps []*saga.StepState, timeout time.Duration) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	if len(steps) == 0 {
		return nil
	}

	// Begin transaction
	tx, err := p.BeginTransaction(ctx, timeout)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Save all steps
	for _, step := range steps {
		if err := tx.SaveStepState(ctx, sagaID, step); err != nil {
			return fmt.Errorf("failed to save step %s in batch: %w", step.ID, err)
		}
	}

	// Commit transaction
	return tx.Commit(ctx)
}

// ==========================
// Optimistic Locking Support
// ==========================

// UpdateSagaWithOptimisticLock updates a Saga instance with optimistic locking.
// It checks that the current version in the database matches the expected version,
// and only updates if they match. This prevents lost updates in concurrent scenarios.
//
// Returns ErrOptimisticLockFailed if the version doesn't match (someone else modified the Saga).
// Returns state.ErrSagaNotFound if the Saga doesn't exist.
func (p *PostgresStateStorage) UpdateSagaWithOptimisticLock(ctx context.Context, instance saga.SagaInstance, expectedVersion int) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return err
	}

	if instance == nil {
		return state.ErrInvalidSagaID
	}

	sagaID := instance.GetID()
	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	// Convert SagaInstance to SagaInstanceData
	data := p.convertToSagaInstanceData(instance)

	// Serialize complex data fields to JSONB
	initialDataJSON, err := p.marshalJSON(data.InitialData)
	if err != nil {
		return fmt.Errorf("failed to marshal initial_data: %w", err)
	}

	currentDataJSON, err := p.marshalJSON(data.CurrentData)
	if err != nil {
		return fmt.Errorf("failed to marshal current_data: %w", err)
	}

	resultDataJSON, err := p.marshalJSON(data.ResultData)
	if err != nil {
		return fmt.Errorf("failed to marshal result_data: %w", err)
	}

	errorJSON, err := p.marshalJSON(data.Error)
	if err != nil {
		return fmt.Errorf("failed to marshal error: %w", err)
	}

	retryPolicyJSON, err := p.marshalJSON(data.RetryPolicy)
	if err != nil {
		return fmt.Errorf("failed to marshal retry_policy: %w", err)
	}

	metadataJSON, err := p.marshalJSON(data.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Convert timeout to milliseconds
	timeoutMs := data.Timeout.Milliseconds()

	// Increment version for update
	newVersion := expectedVersion + 1

	// Prepare UPDATE statement with optimistic lock check
	query := fmt.Sprintf(`
		UPDATE %s SET
			definition_id = $1,
			name = $2,
			description = $3,
			state = $4,
			current_step = $5,
			total_steps = $6,
			updated_at = $7,
			started_at = $8,
			completed_at = $9,
			timed_out_at = $10,
			initial_data = $11,
			current_data = $12,
			result_data = $13,
			error = $14,
			timeout_ms = $15,
			retry_policy = $16,
			metadata = $17,
			trace_id = $18,
			span_id = $19,
			version = $20
		WHERE id = $21 AND version = $22
	`, p.instancesTable)

	// Apply query timeout
	queryCtx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	// Execute the update with version check
	result, err := p.db.ExecContext(
		queryCtx,
		query,
		data.DefinitionID, data.Name, data.Description,
		int(data.State), data.CurrentStep, data.TotalSteps,
		time.Now(), data.StartedAt, data.CompletedAt, data.TimedOutAt,
		initialDataJSON, currentDataJSON, resultDataJSON,
		errorJSON, timeoutMs, retryPolicyJSON,
		metadataJSON, data.TraceID, data.SpanID,
		newVersion, sagaID, expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("failed to update saga instance with optimistic lock: %w", err)
	}

	// Check if the update affected any rows
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Check if saga exists to provide better error message
		var exists bool
		checkQuery := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1)", p.instancesTable)
		if err := p.db.QueryRowContext(queryCtx, checkQuery, sagaID).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check saga existence: %w", err)
		}

		if !exists {
			return state.ErrSagaNotFound
		}

		// Saga exists but version didn't match
		return ErrOptimisticLockFailed
	}

	return nil
}

// UpdateSagaStateWithOptimisticLock updates only the state of a Saga instance with optimistic locking.
// This is a lightweight operation that doesn't require loading the entire Saga.
func (p *PostgresStateStorage) UpdateSagaStateWithOptimisticLock(ctx context.Context, sagaID string, sagaState saga.SagaState, expectedVersion int, metadata map[string]interface{}) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return err
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	// Increment version for update
	newVersion := expectedVersion + 1

	// Apply query timeout
	queryCtx, cancel := context.WithTimeout(ctx, p.config.QueryTimeout)
	defer cancel()

	var result sql.Result
	var err error

	// If metadata is provided, merge it with existing metadata
	if len(metadata) > 0 {
		// Serialize new metadata
		metadataJSON, err := p.marshalJSON(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		// Use JSONB concatenation to merge metadata
		query := fmt.Sprintf(`
			UPDATE %s
			SET state = $1, 
			    updated_at = $2,
			    metadata = COALESCE(metadata, '{}'::jsonb) || $3::jsonb,
			    version = $4
			WHERE id = $5 AND version = $6
		`, p.instancesTable)

		result, err = p.db.ExecContext(queryCtx, query, int(sagaState), time.Now(), metadataJSON, newVersion, sagaID, expectedVersion)
	} else {
		// No metadata to update, just update state
		query := fmt.Sprintf(`
			UPDATE %s
			SET state = $1, updated_at = $2, version = $3
			WHERE id = $4 AND version = $5
		`, p.instancesTable)

		result, err = p.db.ExecContext(queryCtx, query, int(sagaState), time.Now(), newVersion, sagaID, expectedVersion)
	}

	if err != nil {
		return fmt.Errorf("failed to update saga state with optimistic lock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Check if saga exists to provide better error message
		var exists bool
		checkQuery := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1)", p.instancesTable)
		if err := p.db.QueryRowContext(queryCtx, checkQuery, sagaID).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check saga existence: %w", err)
		}

		if !exists {
			return state.ErrSagaNotFound
		}

		// Saga exists but version didn't match
		return ErrOptimisticLockFailed
	}

	return nil
}
