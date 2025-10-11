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

package state

import (
	"errors"
)

// Storage backend types
const (
	// StorageTypeMemory represents in-memory storage backend.
	StorageTypeMemory = "memory"

	// StorageTypeRedis represents Redis storage backend.
	StorageTypeRedis = "redis"

	// StorageTypeDatabase represents database storage backend.
	StorageTypeDatabase = "database"
)

// Common errors for state storage operations.
var (
	// ErrSagaNotFound is returned when a Saga instance cannot be found in storage.
	ErrSagaNotFound = errors.New("saga not found")

	// ErrInvalidSagaID is returned when the provided Saga ID is invalid.
	ErrInvalidSagaID = errors.New("invalid saga ID")

	// ErrStorageFailure is returned when a storage operation fails.
	ErrStorageFailure = errors.New("storage operation failed")

	// ErrInvalidState is returned when attempting to update to an invalid state.
	ErrInvalidState = errors.New("invalid saga state")

	// ErrStepNotFound is returned when a step state cannot be found.
	ErrStepNotFound = errors.New("step state not found")

	// ErrStorageClosed is returned when attempting to use a closed storage.
	ErrStorageClosed = errors.New("storage is closed")
)

// SerializationFormat represents the format used for serializing Saga state.
type SerializationFormat string

const (
	// SerializationFormatJSON uses JSON serialization.
	SerializationFormatJSON SerializationFormat = "json"

	// SerializationFormatProtobuf uses Protocol Buffers serialization.
	SerializationFormatProtobuf SerializationFormat = "protobuf"

	// SerializationFormatMsgpack uses MessagePack serialization.
	SerializationFormatMsgpack SerializationFormat = "msgpack"
)

// StorageConfig provides configuration for state storage backends.
type StorageConfig struct {
	// Type specifies the storage backend type (memory, redis, database).
	Type string `json:"type" yaml:"type"`

	// SerializationFormat specifies the format for serializing Saga state.
	SerializationFormat SerializationFormat `json:"serialization_format" yaml:"serialization_format"`

	// EnableCompression enables compression of serialized data.
	EnableCompression bool `json:"enable_compression" yaml:"enable_compression"`

	// MaxRetentionDays specifies how many days to retain completed Sagas.
	// Zero or negative value means no automatic cleanup.
	MaxRetentionDays int `json:"max_retention_days" yaml:"max_retention_days"`

	// Memory storage specific options
	Memory *MemoryStorageConfig `json:"memory,omitempty" yaml:"memory,omitempty"`

	// Redis storage specific options (for future use)
	Redis *RedisStorageConfig `json:"redis,omitempty" yaml:"redis,omitempty"`

	// Database storage specific options (for future use)
	Database *DatabaseStorageConfig `json:"database,omitempty" yaml:"database,omitempty"`
}

// MemoryStorageConfig provides configuration for memory storage backend.
type MemoryStorageConfig struct {
	// InitialCapacity specifies the initial capacity of the storage map.
	InitialCapacity int `json:"initial_capacity" yaml:"initial_capacity"`

	// MaxCapacity specifies the maximum number of Sagas to store.
	// Zero means no limit.
	MaxCapacity int `json:"max_capacity" yaml:"max_capacity"`

	// EnableMetrics enables collection of storage metrics.
	EnableMetrics bool `json:"enable_metrics" yaml:"enable_metrics"`
}

// RedisStorageConfig provides configuration for Redis storage backend (future).
type RedisStorageConfig struct {
	// Addresses is the list of Redis server addresses.
	Addresses []string `json:"addresses" yaml:"addresses"`

	// Password is the Redis authentication password.
	Password string `json:"password" yaml:"password"`

	// Database is the Redis database number.
	Database int `json:"database" yaml:"database"`

	// KeyPrefix is the prefix for all Redis keys.
	KeyPrefix string `json:"key_prefix" yaml:"key_prefix"`

	// MaxRetries is the maximum number of retries for failed operations.
	MaxRetries int `json:"max_retries" yaml:"max_retries"`
}

// DatabaseStorageConfig provides configuration for database storage backend (future).
type DatabaseStorageConfig struct {
	// Driver is the database driver (postgres, mysql, etc.).
	Driver string `json:"driver" yaml:"driver"`

	// DSN is the database connection string.
	DSN string `json:"dsn" yaml:"dsn"`

	// TableName is the name of the table for storing Saga instances.
	TableName string `json:"table_name" yaml:"table_name"`

	// MaxOpenConns is the maximum number of open connections.
	MaxOpenConns int `json:"max_open_conns" yaml:"max_open_conns"`

	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int `json:"max_idle_conns" yaml:"max_idle_conns"`
}

// DefaultStorageConfig returns a default storage configuration.
func DefaultStorageConfig() *StorageConfig {
	return &StorageConfig{
		Type:                StorageTypeMemory,
		SerializationFormat: SerializationFormatJSON,
		EnableCompression:   false,
		MaxRetentionDays:    30,
		Memory: &MemoryStorageConfig{
			InitialCapacity: 100,
			MaxCapacity:     0, // No limit
			EnableMetrics:   true,
		},
	}
}
