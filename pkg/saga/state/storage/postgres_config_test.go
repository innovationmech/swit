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
	"testing"
	"time"
)

func TestDefaultPostgresConfig(t *testing.T) {
	config := DefaultPostgresConfig()

	if config == nil {
		t.Fatal("DefaultPostgresConfig returned nil")
	}

	// Verify default values
	if config.MaxOpenConns != 25 {
		t.Errorf("Expected MaxOpenConns=25, got %d", config.MaxOpenConns)
	}

	if config.MaxIdleConns != 5 {
		t.Errorf("Expected MaxIdleConns=5, got %d", config.MaxIdleConns)
	}

	if config.ConnMaxLifetime != 1*time.Hour {
		t.Errorf("Expected ConnMaxLifetime=1h, got %v", config.ConnMaxLifetime)
	}

	if config.ConnMaxIdleTime != 30*time.Minute {
		t.Errorf("Expected ConnMaxIdleTime=30m, got %v", config.ConnMaxIdleTime)
	}

	if config.ConnectionTimeout != 10*time.Second {
		t.Errorf("Expected ConnectionTimeout=10s, got %v", config.ConnectionTimeout)
	}

	if config.QueryTimeout != 30*time.Second {
		t.Errorf("Expected QueryTimeout=30s, got %v", config.QueryTimeout)
	}

	if !config.EnablePreparedStatements {
		t.Error("Expected EnablePreparedStatements=true")
	}

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries=3, got %d", config.MaxRetries)
	}

	if config.RetryBackoff != 100*time.Millisecond {
		t.Errorf("Expected RetryBackoff=100ms, got %v", config.RetryBackoff)
	}

	if config.MaxRetryBackoff != 5*time.Second {
		t.Errorf("Expected MaxRetryBackoff=5s, got %v", config.MaxRetryBackoff)
	}

	if !config.EnableMetrics {
		t.Error("Expected EnableMetrics=true")
	}

	if !config.EnableTracing {
		t.Error("Expected EnableTracing=true")
	}

	if config.SchemaName != "public" {
		t.Errorf("Expected SchemaName=public, got %s", config.SchemaName)
	}

	if config.AutoMigrate {
		t.Error("Expected AutoMigrate=false")
	}
}

func TestPostgresConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *PostgresConfig
		wantError bool
		errorType error
	}{
		{
			name:      "nil config",
			config:    nil,
			wantError: true,
			errorType: ErrInvalidPostgresConfig,
		},
		{
			name: "empty DSN",
			config: &PostgresConfig{
				DSN:               "",
				MaxOpenConns:      25,
				MaxIdleConns:      5,
				ConnectionTimeout: 10 * time.Second,
				QueryTimeout:      30 * time.Second,
			},
			wantError: true,
			errorType: ErrEmptyDSN,
		},
		{
			name: "invalid max open conns",
			config: &PostgresConfig{
				DSN:               "postgres://localhost/test",
				MaxOpenConns:      0,
				MaxIdleConns:      5,
				ConnectionTimeout: 10 * time.Second,
				QueryTimeout:      30 * time.Second,
			},
			wantError: true,
			errorType: ErrInvalidMaxConnections,
		},
		{
			name: "negative max idle conns",
			config: &PostgresConfig{
				DSN:               "postgres://localhost/test",
				MaxOpenConns:      25,
				MaxIdleConns:      -1,
				ConnectionTimeout: 10 * time.Second,
				QueryTimeout:      30 * time.Second,
			},
			wantError: true,
			errorType: ErrInvalidMinConnections,
		},
		{
			name: "max idle exceeds max open",
			config: &PostgresConfig{
				DSN:               "postgres://localhost/test",
				MaxOpenConns:      10,
				MaxIdleConns:      20,
				ConnectionTimeout: 10 * time.Second,
				QueryTimeout:      30 * time.Second,
			},
			wantError: true,
			errorType: ErrInvalidPostgresConfig,
		},
		{
			name: "invalid connection timeout",
			config: &PostgresConfig{
				DSN:               "postgres://localhost/test",
				MaxOpenConns:      25,
				MaxIdleConns:      5,
				ConnectionTimeout: 0,
				QueryTimeout:      30 * time.Second,
			},
			wantError: true,
			errorType: ErrInvalidConnectionTimeout,
		},
		{
			name: "invalid query timeout",
			config: &PostgresConfig{
				DSN:               "postgres://localhost/test",
				MaxOpenConns:      25,
				MaxIdleConns:      5,
				ConnectionTimeout: 10 * time.Second,
				QueryTimeout:      0,
			},
			wantError: true,
			errorType: ErrInvalidQueryTimeout,
		},
		{
			name: "negative max retries",
			config: &PostgresConfig{
				DSN:               "postgres://localhost/test",
				MaxOpenConns:      25,
				MaxIdleConns:      5,
				ConnectionTimeout: 10 * time.Second,
				QueryTimeout:      30 * time.Second,
				MaxRetries:        -1,
			},
			wantError: true,
			errorType: ErrInvalidPostgresConfig,
		},
		{
			name: "retry backoff exceeds max",
			config: &PostgresConfig{
				DSN:               "postgres://localhost/test",
				MaxOpenConns:      25,
				MaxIdleConns:      5,
				ConnectionTimeout: 10 * time.Second,
				QueryTimeout:      30 * time.Second,
				MaxRetries:        3,
				RetryBackoff:      10 * time.Second,
				MaxRetryBackoff:   5 * time.Second,
			},
			wantError: true,
			errorType: ErrInvalidPostgresConfig,
		},
		{
			name: "valid config",
			config: &PostgresConfig{
				DSN:               "postgres://localhost/test",
				MaxOpenConns:      25,
				MaxIdleConns:      5,
				ConnectionTimeout: 10 * time.Second,
				QueryTimeout:      30 * time.Second,
				MaxRetries:        3,
				RetryBackoff:      100 * time.Millisecond,
				MaxRetryBackoff:   5 * time.Second,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got nil")
					return
				}
				// Check if error is of expected type using errors.Is
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("Expected error containing %v, got %v", tt.errorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestPostgresConfigApplyDefaults(t *testing.T) {
	config := &PostgresConfig{
		DSN: "postgres://localhost/test",
	}

	config.ApplyDefaults()

	if config.MaxOpenConns != 25 {
		t.Errorf("Expected MaxOpenConns=25, got %d", config.MaxOpenConns)
	}

	if config.MaxIdleConns != 5 {
		t.Errorf("Expected MaxIdleConns=5, got %d", config.MaxIdleConns)
	}

	if config.ConnectionTimeout != 10*time.Second {
		t.Errorf("Expected ConnectionTimeout=10s, got %v", config.ConnectionTimeout)
	}

	if config.QueryTimeout != 30*time.Second {
		t.Errorf("Expected QueryTimeout=30s, got %v", config.QueryTimeout)
	}

	if config.SchemaName != "public" {
		t.Errorf("Expected SchemaName=public, got %s", config.SchemaName)
	}

	// Test that existing values are not overwritten
	config2 := &PostgresConfig{
		DSN:          "postgres://localhost/test",
		MaxOpenConns: 50,
		SchemaName:   "custom",
	}

	config2.ApplyDefaults()

	if config2.MaxOpenConns != 50 {
		t.Errorf("Expected MaxOpenConns=50 (not overwritten), got %d", config2.MaxOpenConns)
	}

	if config2.SchemaName != "custom" {
		t.Errorf("Expected SchemaName=custom (not overwritten), got %s", config2.SchemaName)
	}
}

func TestPostgresConfigTableNames(t *testing.T) {
	tests := []struct {
		name                    string
		prefix                  string
		expectedInstancesTable  string
		expectedStepsTable      string
		expectedEventsTable     string
		expectedFullTableResult string
	}{
		{
			name:                    "no prefix",
			prefix:                  "",
			expectedInstancesTable:  "saga_instances",
			expectedStepsTable:      "saga_steps",
			expectedEventsTable:     "saga_events",
			expectedFullTableResult: "test_table",
		},
		{
			name:                    "with prefix",
			prefix:                  "app_",
			expectedInstancesTable:  "app_saga_instances",
			expectedStepsTable:      "app_saga_steps",
			expectedEventsTable:     "app_saga_events",
			expectedFullTableResult: "app_test_table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &PostgresConfig{
				TablePrefix: tt.prefix,
			}

			if got := config.GetInstancesTableName(); got != tt.expectedInstancesTable {
				t.Errorf("GetInstancesTableName() = %v, want %v", got, tt.expectedInstancesTable)
			}

			if got := config.GetStepsTableName(); got != tt.expectedStepsTable {
				t.Errorf("GetStepsTableName() = %v, want %v", got, tt.expectedStepsTable)
			}

			if got := config.GetEventsTableName(); got != tt.expectedEventsTable {
				t.Errorf("GetEventsTableName() = %v, want %v", got, tt.expectedEventsTable)
			}

			if got := config.GetFullTableName("test_table"); got != tt.expectedFullTableResult {
				t.Errorf("GetFullTableName(test_table) = %v, want %v", got, tt.expectedFullTableResult)
			}
		})
	}
}
