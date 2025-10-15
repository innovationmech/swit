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
	"errors"
	"fmt"
	"time"
)

var (
	// ErrInvalidPostgresConfig indicates that the PostgreSQL configuration is invalid.
	ErrInvalidPostgresConfig = errors.New("invalid postgres configuration")

	// ErrEmptyDSN indicates that the DSN is empty.
	ErrEmptyDSN = errors.New("postgres DSN cannot be empty")

	// ErrInvalidMaxConnections indicates that max connections is invalid.
	ErrInvalidMaxConnections = errors.New("max connections must be positive")

	// ErrInvalidMinConnections indicates that min connections is invalid.
	ErrInvalidMinConnections = errors.New("min connections must be >= 0")

	// ErrInvalidConnectionTimeout indicates that connection timeout is invalid.
	ErrInvalidConnectionTimeout = errors.New("connection timeout must be positive")

	// ErrInvalidQueryTimeout indicates that query timeout is invalid.
	ErrInvalidQueryTimeout = errors.New("query timeout must be positive")
)

// PostgresConfig holds the configuration for PostgreSQL connection and behavior.
type PostgresConfig struct {
	// DSN is the database connection string (Data Source Name).
	// Format: "postgres://username:password@host:port/database?options"
	// Required.
	DSN string `json:"dsn" yaml:"dsn"`

	// MaxOpenConns sets the maximum number of open connections to the database.
	// Default: 25
	MaxOpenConns int `json:"max_open_conns" yaml:"max_open_conns"`

	// MaxIdleConns sets the maximum number of connections in the idle connection pool.
	// Should be less than or equal to MaxOpenConns.
	// Default: 5
	MaxIdleConns int `json:"max_idle_conns" yaml:"max_idle_conns"`

	// ConnMaxLifetime sets the maximum amount of time a connection may be reused.
	// Expired connections may be closed lazily before reuse.
	// Default: 1 hour
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`

	// ConnMaxIdleTime sets the maximum amount of time a connection may be idle.
	// Expired connections may be closed lazily before reuse.
	// Default: 30 minutes
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"`

	// ConnectionTimeout is the timeout for establishing a database connection.
	// Default: 10 seconds
	ConnectionTimeout time.Duration `json:"connection_timeout" yaml:"connection_timeout"`

	// QueryTimeout is the default timeout for database queries.
	// Individual operations may override this.
	// Default: 30 seconds
	QueryTimeout time.Duration `json:"query_timeout" yaml:"query_timeout"`

	// EnablePreparedStatements enables prepared statement caching for performance.
	// Default: true
	EnablePreparedStatements bool `json:"enable_prepared_statements" yaml:"enable_prepared_statements"`

	// MaxRetries is the maximum number of retries for transient database errors.
	// Default: 3
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// RetryBackoff is the initial backoff duration between retries.
	// Backoff will double on each retry up to a maximum.
	// Default: 100 milliseconds
	RetryBackoff time.Duration `json:"retry_backoff" yaml:"retry_backoff"`

	// MaxRetryBackoff is the maximum backoff duration between retries.
	// Default: 5 seconds
	MaxRetryBackoff time.Duration `json:"max_retry_backoff" yaml:"max_retry_backoff"`

	// EnableMetrics enables collection of database operation metrics.
	// Default: true
	EnableMetrics bool `json:"enable_metrics" yaml:"enable_metrics"`

	// EnableTracing enables distributed tracing for database operations.
	// Default: true
	EnableTracing bool `json:"enable_tracing" yaml:"enable_tracing"`

	// TablePrefix is the prefix for all database tables.
	// This allows multiple applications to share the same database.
	// Default: "" (no prefix)
	TablePrefix string `json:"table_prefix" yaml:"table_prefix"`

	// SchemaName is the database schema name to use.
	// Default: "public"
	SchemaName string `json:"schema_name" yaml:"schema_name"`

	// AutoMigrate indicates whether to automatically run schema migrations on startup.
	// Default: false (for safety)
	AutoMigrate bool `json:"auto_migrate" yaml:"auto_migrate"`
}

// DefaultPostgresConfig returns a PostgresConfig with default values.
func DefaultPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		DSN:                      "",
		MaxOpenConns:             25,
		MaxIdleConns:             5,
		ConnMaxLifetime:          1 * time.Hour,
		ConnMaxIdleTime:          30 * time.Minute,
		ConnectionTimeout:        10 * time.Second,
		QueryTimeout:             30 * time.Second,
		EnablePreparedStatements: true,
		MaxRetries:               3,
		RetryBackoff:             100 * time.Millisecond,
		MaxRetryBackoff:          5 * time.Second,
		EnableMetrics:            true,
		EnableTracing:            true,
		TablePrefix:              "",
		SchemaName:               "public",
		AutoMigrate:              false,
	}
}

// Validate validates the PostgreSQL configuration and returns an error if invalid.
func (c *PostgresConfig) Validate() error {
	if c == nil {
		return ErrInvalidPostgresConfig
	}

	// DSN is required
	if c.DSN == "" {
		return ErrEmptyDSN
	}

	// Validate connection pool settings
	if c.MaxOpenConns <= 0 {
		return fmt.Errorf("%w: max_open_conns must be > 0", ErrInvalidMaxConnections)
	}

	if c.MaxIdleConns < 0 {
		return fmt.Errorf("%w: max_idle_conns must be >= 0", ErrInvalidMinConnections)
	}

	if c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("%w: max_idle_conns (%d) cannot exceed max_open_conns (%d)",
			ErrInvalidPostgresConfig, c.MaxIdleConns, c.MaxOpenConns)
	}

	// Validate timeouts
	if c.ConnectionTimeout <= 0 {
		return ErrInvalidConnectionTimeout
	}

	if c.QueryTimeout <= 0 {
		return ErrInvalidQueryTimeout
	}

	if c.ConnMaxLifetime < 0 {
		return fmt.Errorf("%w: conn_max_lifetime must be >= 0", ErrInvalidPostgresConfig)
	}

	if c.ConnMaxIdleTime < 0 {
		return fmt.Errorf("%w: conn_max_idle_time must be >= 0", ErrInvalidPostgresConfig)
	}

	// Validate retry settings
	if c.MaxRetries < 0 {
		return fmt.Errorf("%w: max_retries must be >= 0", ErrInvalidPostgresConfig)
	}

	if c.RetryBackoff < 0 {
		return fmt.Errorf("%w: retry_backoff must be >= 0", ErrInvalidPostgresConfig)
	}

	if c.MaxRetryBackoff < 0 {
		return fmt.Errorf("%w: max_retry_backoff must be >= 0", ErrInvalidPostgresConfig)
	}

	if c.RetryBackoff > c.MaxRetryBackoff {
		return fmt.Errorf("%w: retry_backoff (%v) cannot exceed max_retry_backoff (%v)",
			ErrInvalidPostgresConfig, c.RetryBackoff, c.MaxRetryBackoff)
	}

	return nil
}

// ApplyDefaults applies default values to unset fields.
func (c *PostgresConfig) ApplyDefaults() {
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = 25
	}

	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 5
	}

	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = 1 * time.Hour
	}

	if c.ConnMaxIdleTime == 0 {
		c.ConnMaxIdleTime = 30 * time.Minute
	}

	if c.ConnectionTimeout == 0 {
		c.ConnectionTimeout = 10 * time.Second
	}

	if c.QueryTimeout == 0 {
		c.QueryTimeout = 30 * time.Second
	}

	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}

	if c.RetryBackoff == 0 {
		c.RetryBackoff = 100 * time.Millisecond
	}

	if c.MaxRetryBackoff == 0 {
		c.MaxRetryBackoff = 5 * time.Second
	}

	if c.SchemaName == "" {
		c.SchemaName = "public"
	}
}

// GetFullTableName returns the full table name with prefix.
func (c *PostgresConfig) GetFullTableName(tableName string) string {
	if c.TablePrefix == "" {
		return tableName
	}
	return c.TablePrefix + tableName
}

// GetInstancesTableName returns the full name of the saga_instances table.
func (c *PostgresConfig) GetInstancesTableName() string {
	return c.GetFullTableName("saga_instances")
}

// GetStepsTableName returns the full name of the saga_steps table.
func (c *PostgresConfig) GetStepsTableName() string {
	return c.GetFullTableName("saga_steps")
}

// GetEventsTableName returns the full name of the saga_events table.
func (c *PostgresConfig) GetEventsTableName() string {
	return c.GetFullTableName("saga_events")
}
