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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"time"
)

var (
	// ErrInvalidRedisConfig indicates that the Redis configuration is invalid.
	ErrInvalidRedisConfig = errors.New("invalid redis configuration")

	// ErrEmptyAddress indicates that the Redis address is empty.
	ErrEmptyAddress = errors.New("redis address cannot be empty")

	// ErrInvalidDB indicates that the Redis DB number is invalid.
	ErrInvalidDB = errors.New("redis DB number must be >= 0")

	// ErrInvalidTimeout indicates that a timeout value is invalid.
	ErrInvalidTimeout = errors.New("timeout must be positive")

	// ErrInvalidPoolSize indicates that the pool size is invalid.
	ErrInvalidPoolSize = errors.New("pool size must be positive")

	// ErrInvalidMaxRetries indicates that max retries is invalid.
	ErrInvalidMaxRetries = errors.New("max retries must be >= 0")

	// ErrInvalidTTL indicates that the TTL is invalid.
	ErrInvalidTTL = errors.New("TTL must be positive")

	// ErrTLSCertNotFound indicates that the TLS certificate file was not found.
	ErrTLSCertNotFound = errors.New("TLS certificate file not found")

	// ErrTLSKeyNotFound indicates that the TLS key file was not found.
	ErrTLSKeyNotFound = errors.New("TLS key file not found")

	// ErrTLSCANotFound indicates that the TLS CA file was not found.
	ErrTLSCANotFound = errors.New("TLS CA file not found")
)

// RedisMode defines the operation mode of Redis.
type RedisMode string

const (
	// RedisModeStandalone represents standalone Redis mode.
	RedisModeStandalone RedisMode = "standalone"

	// RedisModeCluster represents Redis cluster mode.
	RedisModeCluster RedisMode = "cluster"

	// RedisModeSentinel represents Redis sentinel mode for high availability.
	RedisModeSentinel RedisMode = "sentinel"
)

// RedisConfig holds the configuration for Redis connection.
type RedisConfig struct {
	// Mode specifies the Redis operation mode (standalone, cluster, sentinel).
	// Default: standalone
	Mode RedisMode `json:"mode" yaml:"mode"`

	// Addr is the Redis server address in the format "host:port".
	// For cluster mode, this can be a comma-separated list of addresses.
	// Required for standalone mode.
	Addr string `json:"addr" yaml:"addr"`

	// Addrs is a list of Redis server addresses for cluster mode.
	// Either Addr or Addrs should be specified, not both.
	Addrs []string `json:"addrs" yaml:"addrs"`

	// MasterName is the name of the Redis sentinel master.
	// Required for sentinel mode.
	MasterName string `json:"master_name" yaml:"master_name"`

	// SentinelAddrs is a list of Redis sentinel addresses.
	// Required for sentinel mode.
	SentinelAddrs []string `json:"sentinel_addrs" yaml:"sentinel_addrs"`

	// Username for Redis authentication (Redis 6.0+).
	// Optional.
	Username string `json:"username" yaml:"username"`

	// Password for Redis authentication.
	// Optional.
	Password string `json:"password" yaml:"password"`

	// DB is the Redis database number to use.
	// Only applicable in standalone mode. Must be >= 0.
	// Default: 0
	DB int `json:"db" yaml:"db"`

	// KeyPrefix is the prefix for all Redis keys used by the storage.
	// This allows multiple applications to share the same Redis instance.
	// Default: "saga:"
	KeyPrefix string `json:"key_prefix" yaml:"key_prefix"`

	// TTL is the time-to-live for Saga state data in Redis.
	// After this duration, completed Sagas may be automatically removed.
	// Set to 0 for no expiration.
	// Default: 24 hours
	TTL time.Duration `json:"ttl" yaml:"ttl"`

	// DialTimeout is the timeout for establishing new connections.
	// Default: 5 seconds
	DialTimeout time.Duration `json:"dial_timeout" yaml:"dial_timeout"`

	// ReadTimeout is the timeout for socket reads.
	// If reached, commands will fail with a timeout instead of blocking.
	// Default: 3 seconds
	ReadTimeout time.Duration `json:"read_timeout" yaml:"read_timeout"`

	// WriteTimeout is the timeout for socket writes.
	// If reached, commands will fail with a timeout instead of blocking.
	// Default: 3 seconds
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`

	// PoolSize is the maximum number of socket connections.
	// Default: 10 * runtime.GOMAXPROCS
	PoolSize int `json:"pool_size" yaml:"pool_size"`

	// MinIdleConns is the minimum number of idle connections to maintain.
	// Default: 5
	MinIdleConns int `json:"min_idle_conns" yaml:"min_idle_conns"`

	// MaxIdleConns is the maximum number of idle connections.
	// Default: PoolSize
	MaxIdleConns int `json:"max_idle_conns" yaml:"max_idle_conns"`

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle.
	// Default: 30 minutes
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"`

	// ConnMaxLifetime is the maximum lifetime of a connection.
	// Default: 1 hour
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`

	// MaxRetries is the maximum number of retries before giving up.
	// Default: 3
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// MinRetryBackoff is the minimum backoff between each retry.
	// Default: 8 milliseconds
	MinRetryBackoff time.Duration `json:"min_retry_backoff" yaml:"min_retry_backoff"`

	// MaxRetryBackoff is the maximum backoff between each retry.
	// Default: 512 milliseconds
	MaxRetryBackoff time.Duration `json:"max_retry_backoff" yaml:"max_retry_backoff"`

	// TLS configuration for secure connections.
	// Optional.
	TLS *RedisTLSConfig `json:"tls" yaml:"tls"`

	// EnableMetrics enables Redis connection metrics collection.
	// Default: true
	EnableMetrics bool `json:"enable_metrics" yaml:"enable_metrics"`
}

// RedisTLSConfig holds TLS/SSL configuration for Redis connections.
type RedisTLSConfig struct {
	// Enabled indicates whether TLS should be used.
	Enabled bool `json:"enabled" yaml:"enabled"`

	// InsecureSkipVerify controls whether a client verifies the server's
	// certificate chain and host name. Should only be true for testing.
	// Default: false
	InsecureSkipVerify bool `json:"insecure_skip_verify" yaml:"insecure_skip_verify"`

	// CertFile is the path to the client certificate file.
	// Optional for mutual TLS authentication.
	CertFile string `json:"cert_file" yaml:"cert_file"`

	// KeyFile is the path to the client private key file.
	// Optional for mutual TLS authentication.
	KeyFile string `json:"key_file" yaml:"key_file"`

	// CAFile is the path to the CA certificate file.
	// Optional. If not specified, system CA pool is used.
	CAFile string `json:"ca_file" yaml:"ca_file"`

	// ServerName is used to verify the hostname on the returned certificates.
	// Optional. If not specified, the hostname from Addr is used.
	ServerName string `json:"server_name" yaml:"server_name"`
}

// DefaultRedisConfig returns a RedisConfig with default values.
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Mode:            RedisModeStandalone,
		Addr:            "localhost:6379",
		DB:              0,
		KeyPrefix:       "saga:",
		TTL:             24 * time.Hour,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolSize:        10,
		MinIdleConns:    5,
		MaxIdleConns:    0, // Will be set to PoolSize
		ConnMaxIdleTime: 30 * time.Minute,
		ConnMaxLifetime: 1 * time.Hour,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		EnableMetrics:   true,
	}
}

// Validate validates the Redis configuration and returns an error if invalid.
func (c *RedisConfig) Validate() error {
	if c == nil {
		return ErrInvalidRedisConfig
	}

	// Validate mode-specific configurations
	switch c.Mode {
	case RedisModeStandalone, "":
		if c.Addr == "" && len(c.Addrs) == 0 {
			return fmt.Errorf("%w: address is required for standalone mode", ErrInvalidRedisConfig)
		}
		if c.DB < 0 {
			return ErrInvalidDB
		}

	case RedisModeCluster:
		if len(c.Addrs) == 0 && c.Addr == "" {
			return fmt.Errorf("%w: at least one address is required for cluster mode", ErrInvalidRedisConfig)
		}

	case RedisModeSentinel:
		if c.MasterName == "" {
			return fmt.Errorf("%w: master name is required for sentinel mode", ErrInvalidRedisConfig)
		}
		if len(c.SentinelAddrs) == 0 {
			return fmt.Errorf("%w: sentinel addresses are required for sentinel mode", ErrInvalidRedisConfig)
		}

	default:
		return fmt.Errorf("%w: unsupported mode %s", ErrInvalidRedisConfig, c.Mode)
	}

	// Validate timeouts
	if c.DialTimeout < 0 {
		return fmt.Errorf("%w: dial timeout", ErrInvalidTimeout)
	}
	if c.ReadTimeout < 0 {
		return fmt.Errorf("%w: read timeout", ErrInvalidTimeout)
	}
	if c.WriteTimeout < 0 {
		return fmt.Errorf("%w: write timeout", ErrInvalidTimeout)
	}

	// Validate pool configuration
	if c.PoolSize < 0 {
		return ErrInvalidPoolSize
	}
	if c.MinIdleConns < 0 {
		return fmt.Errorf("%w: min idle conns must be >= 0", ErrInvalidPoolSize)
	}
	if c.MaxIdleConns < 0 {
		return fmt.Errorf("%w: max idle conns must be >= 0", ErrInvalidPoolSize)
	}

	// Validate retry configuration
	if c.MaxRetries < 0 {
		return ErrInvalidMaxRetries
	}
	if c.MinRetryBackoff < 0 {
		return fmt.Errorf("%w: min retry backoff", ErrInvalidTimeout)
	}
	if c.MaxRetryBackoff < 0 {
		return fmt.Errorf("%w: max retry backoff", ErrInvalidTimeout)
	}
	if c.MinRetryBackoff > c.MaxRetryBackoff {
		return fmt.Errorf("%w: min retry backoff must be <= max retry backoff", ErrInvalidTimeout)
	}

	// Validate TTL
	if c.TTL < 0 {
		return ErrInvalidTTL
	}

	// Validate TLS configuration
	if c.TLS != nil && c.TLS.Enabled {
		if err := c.validateTLSConfig(); err != nil {
			return err
		}
	}

	return nil
}

// validateTLSConfig validates the TLS configuration.
func (c *RedisConfig) validateTLSConfig() error {
	tls := c.TLS

	// Check if cert and key are both provided or both empty
	if (tls.CertFile == "") != (tls.KeyFile == "") {
		return fmt.Errorf("%w: both cert and key must be provided for mutual TLS", ErrInvalidRedisConfig)
	}

	// Validate cert file if provided
	if tls.CertFile != "" {
		if _, err := os.Stat(tls.CertFile); os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrTLSCertNotFound, tls.CertFile)
		}
	}

	// Validate key file if provided
	if tls.KeyFile != "" {
		if _, err := os.Stat(tls.KeyFile); os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrTLSKeyNotFound, tls.KeyFile)
		}
	}

	// Validate CA file if provided
	if tls.CAFile != "" {
		if _, err := os.Stat(tls.CAFile); os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrTLSCANotFound, tls.CAFile)
		}
	}

	return nil
}

// ApplyDefaults applies default values to unset fields.
func (c *RedisConfig) ApplyDefaults() {
	if c.Mode == "" {
		c.Mode = RedisModeStandalone
	}

	if c.KeyPrefix == "" {
		c.KeyPrefix = "saga:"
	}

	if c.TTL == 0 {
		c.TTL = 24 * time.Hour
	}

	if c.DialTimeout == 0 {
		c.DialTimeout = 5 * time.Second
	}

	if c.ReadTimeout == 0 {
		c.ReadTimeout = 3 * time.Second
	}

	if c.WriteTimeout == 0 {
		c.WriteTimeout = 3 * time.Second
	}

	if c.PoolSize == 0 {
		c.PoolSize = 10
	}

	if c.MinIdleConns == 0 {
		c.MinIdleConns = 5
	}

	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = c.PoolSize
	}

	if c.ConnMaxIdleTime == 0 {
		c.ConnMaxIdleTime = 30 * time.Minute
	}

	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = 1 * time.Hour
	}

	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}

	if c.MinRetryBackoff == 0 {
		c.MinRetryBackoff = 8 * time.Millisecond
	}

	if c.MaxRetryBackoff == 0 {
		c.MaxRetryBackoff = 512 * time.Millisecond
	}
}

// BuildTLSConfig creates a tls.Config from the Redis TLS configuration.
func (c *RedisConfig) BuildTLSConfig() (*tls.Config, error) {
	if c.TLS == nil || !c.TLS.Enabled {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.TLS.InsecureSkipVerify,
		ServerName:         c.TLS.ServerName,
	}

	// Load client certificate if provided
	if c.TLS.CertFile != "" && c.TLS.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.TLS.CertFile, c.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate if provided
	if c.TLS.CAFile != "" {
		caCert, err := os.ReadFile(c.TLS.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}
