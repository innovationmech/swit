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

// SaveSaga persists a Saga instance to PostgreSQL storage.
// This method will be implemented in a future task.
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

	// TODO: Implement PostgreSQL insert/update logic
	return ErrNotImplemented
}

// GetSaga retrieves a Saga instance by its ID from PostgreSQL storage.
// This method will be implemented in a future task.
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

	// TODO: Implement PostgreSQL query logic
	return nil, ErrNotImplemented
}

// UpdateSagaState updates only the state of a Saga instance.
// This method will be implemented in a future task.
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

	// TODO: Implement PostgreSQL update logic
	return ErrNotImplemented
}

// DeleteSaga removes a Saga instance from PostgreSQL storage.
// This method will be implemented in a future task.
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

	// TODO: Implement PostgreSQL delete logic
	return ErrNotImplemented
}

// GetActiveSagas retrieves all active Saga instances based on the filter.
// This method will be implemented in a future task.
func (p *PostgresStateStorage) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	// TODO: Implement PostgreSQL query with filter logic
	return nil, ErrNotImplemented
}

// GetTimeoutSagas retrieves Saga instances that have timed out before the specified time.
// This method will be implemented in a future task.
func (p *PostgresStateStorage) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	// TODO: Implement PostgreSQL timeout query logic
	return nil, ErrNotImplemented
}

// SaveStepState persists the state of a specific step within a Saga.
// This method will be implemented in a future task.
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

	// TODO: Implement PostgreSQL insert/update for step state
	return ErrNotImplemented
}

// GetStepStates retrieves all step states for a Saga instance.
// This method will be implemented in a future task.
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

	// TODO: Implement PostgreSQL query for step states
	return nil, ErrNotImplemented
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
