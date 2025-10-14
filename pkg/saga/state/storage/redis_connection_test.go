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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedisConnection_InvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *RedisConfig
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "invalid standalone config",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				// Missing Addr
			},
			expectError: true,
		},
		{
			name: "invalid DB number",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				DB:   -1,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := NewRedisConnection(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, conn)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, conn)
				if conn != nil {
					conn.Close()
				}
			}
		})
	}
}

func TestNewRedisConnection_ValidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *RedisConfig
	}{
		{
			name: "standalone mode",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
			},
		},
		{
			name: "standalone mode with addrs",
			config: &RedisConfig{
				Mode:  RedisModeStandalone,
				Addrs: []string{"localhost:6379"},
			},
		},
		{
			name: "cluster mode",
			config: &RedisConfig{
				Mode:  RedisModeCluster,
				Addrs: []string{"localhost:7000", "localhost:7001"},
			},
		},
		{
			name: "cluster mode with addr string",
			config: &RedisConfig{
				Mode: RedisModeCluster,
				Addr: "localhost:7000,localhost:7001",
			},
		},
		{
			name: "sentinel mode",
			config: &RedisConfig{
				Mode:          RedisModeSentinel,
				MasterName:    "mymaster",
				SentinelAddrs: []string{"localhost:26379"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := NewRedisConnection(tt.config)

			// Connection creation should succeed even if Redis is not available
			// The actual connection is lazy
			assert.NoError(t, err)
			require.NotNil(t, conn)

			// Verify config is applied
			assert.NotNil(t, conn.config)
			assert.Equal(t, tt.config.Mode, conn.config.Mode)

			// Verify client is created
			client, err := conn.GetClient()
			assert.NoError(t, err)
			assert.NotNil(t, client)

			// Close the connection
			err = conn.Close()
			assert.NoError(t, err)
		})
	}
}

func TestNewRedisConnection_DefaultsApplied(t *testing.T) {
	config := &RedisConfig{
		Mode: RedisModeStandalone,
		Addr: "localhost:6379",
	}

	conn, err := NewRedisConnection(config)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close()

	// Verify defaults are applied
	assert.Equal(t, "saga:", conn.config.KeyPrefix)
	assert.Equal(t, 24*time.Hour, conn.config.TTL)
	assert.Equal(t, 5*time.Second, conn.config.DialTimeout)
	assert.Equal(t, 3*time.Second, conn.config.ReadTimeout)
	assert.Equal(t, 3*time.Second, conn.config.WriteTimeout)
	assert.Equal(t, 10, conn.config.PoolSize)
	assert.Equal(t, 5, conn.config.MinIdleConns)
}

func TestNewRedisConnectionWithRetry_Success(t *testing.T) {
	t.Skip("Requires running Redis instance")

	config := &RedisConfig{
		Mode: RedisModeStandalone,
		Addr: "localhost:6379",
	}

	ctx := context.Background()
	conn, err := NewRedisConnectionWithRetry(ctx, config, 3)

	assert.NoError(t, err)
	require.NotNil(t, conn)

	// Verify connection works
	err = conn.Ping(ctx)
	assert.NoError(t, err)

	conn.Close()
}

func TestNewRedisConnectionWithRetry_Failure(t *testing.T) {
	config := &RedisConfig{
		Mode:        RedisModeStandalone,
		Addr:        "localhost:9999", // Non-existent port
		DialTimeout: 100 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := NewRedisConnectionWithRetry(ctx, config, 2)

	assert.Error(t, err)
	assert.Nil(t, conn)
	assert.ErrorIs(t, err, ErrConnectionFailed)
}

func TestNewRedisConnectionWithRetry_ContextCancelled(t *testing.T) {
	config := &RedisConfig{
		Mode:        RedisModeStandalone,
		Addr:        "localhost:9999", // Non-existent port
		DialTimeout: 100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	conn, err := NewRedisConnectionWithRetry(ctx, config, 3)

	assert.Error(t, err)
	assert.Nil(t, conn)
}

func TestRedisConnection_GetClient(t *testing.T) {
	config := &RedisConfig{
		Mode: RedisModeStandalone,
		Addr: "localhost:6379",
	}

	conn, err := NewRedisConnection(config)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close()

	// Get client before closing
	client, err := conn.GetClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Close connection
	err = conn.Close()
	assert.NoError(t, err)

	// Try to get client after closing
	client, err = conn.GetClient()
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrConnectionClosed)
	assert.Nil(t, client)
}

func TestRedisConnection_Close(t *testing.T) {
	config := &RedisConfig{
		Mode: RedisModeStandalone,
		Addr: "localhost:6379",
	}

	conn, err := NewRedisConnection(config)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// First close should succeed
	err = conn.Close()
	assert.NoError(t, err)
	assert.True(t, conn.IsClosed())

	// Second close should also succeed (idempotent)
	err = conn.Close()
	assert.NoError(t, err)
}

func TestRedisConnection_IsClosed(t *testing.T) {
	config := &RedisConfig{
		Mode: RedisModeStandalone,
		Addr: "localhost:6379",
	}

	conn, err := NewRedisConnection(config)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Initially not closed
	assert.False(t, conn.IsClosed())

	// After closing
	conn.Close()
	assert.True(t, conn.IsClosed())
}

func TestRedisConnection_GetConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *RedisConfig
		verify func(t *testing.T, original *RedisConfig, copy RedisConfig)
	}{
		{
			name: "basic config without password",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
			},
			verify: func(t *testing.T, original *RedisConfig, copy RedisConfig) {
				assert.Equal(t, original.Mode, copy.Mode)
				assert.Equal(t, original.Addr, copy.Addr)
			},
		},
		{
			name: "config with password",
			config: &RedisConfig{
				Mode:     RedisModeStandalone,
				Addr:     "localhost:6379",
				Password: "secret",
			},
			verify: func(t *testing.T, original *RedisConfig, copy RedisConfig) {
				assert.NotEqual(t, original.Password, copy.Password)
				assert.Equal(t, "***REDACTED***", copy.Password)
			},
		},
		{
			name: "config with empty password",
			config: &RedisConfig{
				Mode:     RedisModeStandalone,
				Addr:     "localhost:6379",
				Password: "",
			},
			verify: func(t *testing.T, original *RedisConfig, copy RedisConfig) {
				assert.Equal(t, "", copy.Password)
			},
		},
		{
			name: "config with TLS",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				TLS: &RedisTLSConfig{
					Enabled:    true,
					ServerName: "redis.example.com",
				},
			},
			verify: func(t *testing.T, original *RedisConfig, copy RedisConfig) {
				assert.NotNil(t, copy.TLS)
				assert.Equal(t, original.TLS.Enabled, copy.TLS.Enabled)
				assert.Equal(t, original.TLS.ServerName, copy.TLS.ServerName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := NewRedisConnection(tt.config)
			require.NoError(t, err)
			require.NotNil(t, conn)
			defer conn.Close()

			// Get config copy
			configCopy := conn.GetConfig()

			// Verify with custom verification function
			tt.verify(t, tt.config, configCopy)
		})
	}
}

func TestRedisConnection_Ping(t *testing.T) {
	t.Skip("Requires running Redis instance")

	config := &RedisConfig{
		Mode: RedisModeStandalone,
		Addr: "localhost:6379",
	}

	conn, err := NewRedisConnection(config)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close()

	ctx := context.Background()

	// Ping should succeed
	err = conn.Ping(ctx)
	assert.NoError(t, err)

	// Close and try to ping
	conn.Close()
	err = conn.Ping(ctx)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrConnectionClosed)
}

func TestRedisConnection_Ping_Timeout(t *testing.T) {
	config := &RedisConfig{
		Mode:        RedisModeStandalone,
		Addr:        "localhost:9999", // Non-existent port
		DialTimeout: 100 * time.Millisecond,
	}

	conn, err := NewRedisConnection(config)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close()

	ctx := context.Background()

	// Ping should fail
	err = conn.Ping(ctx)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrPingFailed)
}

func TestRedisConnection_HealthCheck(t *testing.T) {
	t.Skip("Requires running Redis instance")

	config := &RedisConfig{
		Mode: RedisModeStandalone,
		Addr: "localhost:6379",
	}

	conn, err := NewRedisConnection(config)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close()

	ctx := context.Background()

	// Health check should succeed
	err = conn.HealthCheck(ctx)
	assert.NoError(t, err)

	// Close and try health check
	conn.Close()
	err = conn.HealthCheck(ctx)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrConnectionClosed)
}

func TestRedisConnection_CreateClient_StandaloneMode(t *testing.T) {
	tests := []struct {
		name   string
		config *RedisConfig
	}{
		{
			name: "with addr",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
			},
		},
		{
			name: "with addrs",
			config: &RedisConfig{
				Mode:  RedisModeStandalone,
				Addrs: []string{"localhost:6379"},
			},
		},
		{
			name: "with authentication",
			config: &RedisConfig{
				Mode:     RedisModeStandalone,
				Addr:     "localhost:6379",
				Username: "user",
				Password: "pass",
			},
		},
		{
			name: "with custom pool settings",
			config: &RedisConfig{
				Mode:         RedisModeStandalone,
				Addr:         "localhost:6379",
				PoolSize:     20,
				MinIdleConns: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := NewRedisConnection(tt.config)
			assert.NoError(t, err)
			require.NotNil(t, conn)
			defer conn.Close()

			client, err := conn.GetClient()
			assert.NoError(t, err)
			assert.NotNil(t, client)
		})
	}
}

func TestRedisConnection_CreateClient_ClusterMode(t *testing.T) {
	tests := []struct {
		name   string
		config *RedisConfig
	}{
		{
			name: "with addrs list",
			config: &RedisConfig{
				Mode:  RedisModeCluster,
				Addrs: []string{"node1:6379", "node2:6379"},
			},
		},
		{
			name: "with comma-separated addr",
			config: &RedisConfig{
				Mode: RedisModeCluster,
				Addr: "node1:6379, node2:6379, node3:6379",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := NewRedisConnection(tt.config)
			assert.NoError(t, err)
			require.NotNil(t, conn)
			defer conn.Close()

			client, err := conn.GetClient()
			assert.NoError(t, err)
			assert.NotNil(t, client)
		})
	}
}

func TestRedisConnection_CreateClient_SentinelMode(t *testing.T) {
	config := &RedisConfig{
		Mode:          RedisModeSentinel,
		MasterName:    "mymaster",
		SentinelAddrs: []string{"sentinel1:26379", "sentinel2:26379"},
		Username:      "user",
		Password:      "pass",
	}

	conn, err := NewRedisConnection(config)
	assert.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close()

	client, err := conn.GetClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestRedisConnection_CreateClient_UnsupportedMode(t *testing.T) {
	config := &RedisConfig{
		Mode: RedisMode("unsupported"),
		Addr: "localhost:6379",
	}

	conn, err := NewRedisConnection(config)
	assert.Error(t, err)
	assert.Nil(t, conn)
}
