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
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrConnectionFailed indicates that the Redis connection failed.
	ErrConnectionFailed = errors.New("redis connection failed")

	// ErrConnectionClosed indicates that the Redis connection is closed.
	ErrConnectionClosed = errors.New("redis connection is closed")

	// ErrPingFailed indicates that the Redis ping command failed.
	ErrPingFailed = errors.New("redis ping failed")
)

// RedisClient is an interface that wraps the redis.Client, redis.ClusterClient,
// and redis.SentinelClient to provide a unified interface for different Redis modes.
type RedisClient interface {
	redis.Cmdable
	Close() error
	Ping(ctx context.Context) *redis.StatusCmd
}

// RedisConnection manages the Redis client connection with retry logic
// and health checking capabilities.
type RedisConnection struct {
	config *RedisConfig
	client RedisClient
	closed bool
}

// NewRedisConnection creates and initializes a new Redis connection based on the configuration.
// It validates the configuration, creates the appropriate client type (standalone, cluster, or sentinel),
// and verifies connectivity with an initial ping.
func NewRedisConnection(config *RedisConfig) (*RedisConnection, error) {
	if config == nil {
		return nil, ErrInvalidRedisConfig
	}

	// Apply default values
	config.ApplyDefaults()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid redis config: %w", err)
	}

	conn := &RedisConnection{
		config: config,
		closed: false,
	}

	// Create the appropriate client based on mode
	client, err := conn.createClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create redis client: %w", err)
	}

	conn.client = client

	return conn, nil
}

// NewRedisConnectionWithRetry creates a new Redis connection with retry logic.
// It will attempt to connect multiple times with exponential backoff before giving up.
func NewRedisConnectionWithRetry(ctx context.Context, config *RedisConfig, maxAttempts int) (*RedisConnection, error) {
	if maxAttempts <= 0 {
		maxAttempts = config.MaxRetries
		if maxAttempts <= 0 {
			maxAttempts = 3
		}
	}

	var lastErr error
	backoff := config.MinRetryBackoff
	if backoff == 0 {
		backoff = 8 * time.Millisecond
	}

	maxBackoff := config.MaxRetryBackoff
	if maxBackoff == 0 {
		maxBackoff = 512 * time.Millisecond
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		conn, err := NewRedisConnection(config)
		if err == nil {
			// Test the connection
			if err := conn.Ping(ctx); err == nil {
				return conn, nil
			}
			lastErr = err
			conn.Close()
		} else {
			lastErr = err
		}

		// Don't sleep after the last attempt
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("connection retry cancelled: %w", ctx.Err())
			case <-time.After(backoff):
				// Exponential backoff with max limit
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
		}
	}

	return nil, fmt.Errorf("%w after %d attempts: %v", ErrConnectionFailed, maxAttempts, lastErr)
}

// createClient creates the appropriate Redis client based on the configuration mode.
func (c *RedisConnection) createClient() (RedisClient, error) {
	// Build TLS config if needed
	tlsConfig, err := c.config.BuildTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build TLS config: %w", err)
	}

	switch c.config.Mode {
	case RedisModeStandalone, "":
		return c.createStandaloneClient(tlsConfig)

	case RedisModeCluster:
		return c.createClusterClient(tlsConfig)

	case RedisModeSentinel:
		return c.createSentinelClient(tlsConfig)

	default:
		return nil, fmt.Errorf("unsupported redis mode: %s", c.config.Mode)
	}
}

// createStandaloneClient creates a standalone Redis client.
func (c *RedisConnection) createStandaloneClient(tlsConfig interface{}) (RedisClient, error) {
	addr := c.config.Addr
	if addr == "" && len(c.config.Addrs) > 0 {
		addr = c.config.Addrs[0]
	}

	options := &redis.Options{
		Addr:            addr,
		Username:        c.config.Username,
		Password:        c.config.Password,
		DB:              c.config.DB,
		DialTimeout:     c.config.DialTimeout,
		ReadTimeout:     c.config.ReadTimeout,
		WriteTimeout:    c.config.WriteTimeout,
		PoolSize:        c.config.PoolSize,
		MinIdleConns:    c.config.MinIdleConns,
		MaxIdleConns:    c.config.MaxIdleConns,
		ConnMaxIdleTime: c.config.ConnMaxIdleTime,
		ConnMaxLifetime: c.config.ConnMaxLifetime,
		MaxRetries:      c.config.MaxRetries,
		MinRetryBackoff: c.config.MinRetryBackoff,
		MaxRetryBackoff: c.config.MaxRetryBackoff,
	}

	if tlsConfig != nil {
		options.TLSConfig = tlsConfig.(*tls.Config)
	}

	return redis.NewClient(options), nil
}

// createClusterClient creates a Redis cluster client.
func (c *RedisConnection) createClusterClient(tlsConfig interface{}) (RedisClient, error) {
	addrs := c.config.Addrs
	if len(addrs) == 0 && c.config.Addr != "" {
		// Split comma-separated addresses
		addrs = strings.Split(c.config.Addr, ",")
		for i := range addrs {
			addrs[i] = strings.TrimSpace(addrs[i])
		}
	}

	options := &redis.ClusterOptions{
		Addrs:           addrs,
		Username:        c.config.Username,
		Password:        c.config.Password,
		DialTimeout:     c.config.DialTimeout,
		ReadTimeout:     c.config.ReadTimeout,
		WriteTimeout:    c.config.WriteTimeout,
		PoolSize:        c.config.PoolSize,
		MinIdleConns:    c.config.MinIdleConns,
		MaxIdleConns:    c.config.MaxIdleConns,
		ConnMaxIdleTime: c.config.ConnMaxIdleTime,
		ConnMaxLifetime: c.config.ConnMaxLifetime,
		MaxRetries:      c.config.MaxRetries,
		MinRetryBackoff: c.config.MinRetryBackoff,
		MaxRetryBackoff: c.config.MaxRetryBackoff,
	}

	if tlsConfig != nil {
		options.TLSConfig = tlsConfig.(*tls.Config)
	}

	return redis.NewClusterClient(options), nil
}

// createSentinelClient creates a Redis sentinel client for high availability.
func (c *RedisConnection) createSentinelClient(tlsConfig interface{}) (RedisClient, error) {
	options := &redis.FailoverOptions{
		MasterName:       c.config.MasterName,
		SentinelAddrs:    c.config.SentinelAddrs,
		Username:         c.config.Username,
		Password:         c.config.Password,
		DB:               c.config.DB,
		DialTimeout:      c.config.DialTimeout,
		ReadTimeout:      c.config.ReadTimeout,
		WriteTimeout:     c.config.WriteTimeout,
		PoolSize:         c.config.PoolSize,
		MinIdleConns:     c.config.MinIdleConns,
		MaxIdleConns:     c.config.MaxIdleConns,
		ConnMaxIdleTime:  c.config.ConnMaxIdleTime,
		ConnMaxLifetime:  c.config.ConnMaxLifetime,
		MaxRetries:       c.config.MaxRetries,
		MinRetryBackoff:  c.config.MinRetryBackoff,
		MaxRetryBackoff:  c.config.MaxRetryBackoff,
		SentinelUsername: c.config.Username, // Use same credentials for sentinels
		SentinelPassword: c.config.Password,
	}

	if tlsConfig != nil {
		options.TLSConfig = tlsConfig.(*tls.Config)
	}

	return redis.NewFailoverClient(options), nil
}

// GetClient returns the underlying Redis client.
// Returns ErrConnectionClosed if the connection is closed.
func (c *RedisConnection) GetClient() (RedisClient, error) {
	if c.closed {
		return nil, ErrConnectionClosed
	}
	return c.client, nil
}

// Ping checks the connection to Redis by sending a PING command.
func (c *RedisConnection) Ping(ctx context.Context) error {
	if c.closed {
		return ErrConnectionClosed
	}

	result := c.client.Ping(ctx)
	if err := result.Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrPingFailed, err)
	}

	return nil
}

// HealthCheck performs a comprehensive health check on the Redis connection.
// It tests basic connectivity and optionally performs additional checks.
func (c *RedisConnection) HealthCheck(ctx context.Context) error {
	if c.closed {
		return ErrConnectionClosed
	}

	// Basic ping check
	if err := c.Ping(ctx); err != nil {
		return err
	}

	// Additional check: try to get a non-existent key to verify read operations
	key := fmt.Sprintf("%s_health_check_test", c.config.KeyPrefix)
	_, err := c.client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Close closes the Redis connection and releases all resources.
func (c *RedisConnection) Close() error {
	if c.closed {
		return nil
	}

	c.closed = true

	if c.client != nil {
		return c.client.Close()
	}

	return nil
}

// IsClosed returns true if the connection is closed.
func (c *RedisConnection) IsClosed() bool {
	return c.closed
}

// GetConfig returns a copy of the Redis configuration (without sensitive data).
func (c *RedisConnection) GetConfig() RedisConfig {
	config := *c.config

	// Redact sensitive information
	if config.Password != "" {
		config.Password = "***REDACTED***"
	}
	if config.TLS != nil {
		tlsCopy := *config.TLS
		config.TLS = &tlsCopy
	}

	return config
}
