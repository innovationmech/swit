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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRedisConfig(t *testing.T) {
	config := DefaultRedisConfig()

	assert.NotNil(t, config)
	assert.Equal(t, RedisModeStandalone, config.Mode)
	assert.Equal(t, "localhost:6379", config.Addr)
	assert.Equal(t, 0, config.DB)
	assert.Equal(t, "saga:", config.KeyPrefix)
	assert.Equal(t, 24*time.Hour, config.TTL)
	assert.Equal(t, 5*time.Second, config.DialTimeout)
	assert.Equal(t, 3*time.Second, config.ReadTimeout)
	assert.Equal(t, 3*time.Second, config.WriteTimeout)
	assert.Equal(t, 10, config.PoolSize)
	assert.Equal(t, 5, config.MinIdleConns)
	assert.Equal(t, 30*time.Minute, config.ConnMaxIdleTime)
	assert.Equal(t, 1*time.Hour, config.ConnMaxLifetime)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 8*time.Millisecond, config.MinRetryBackoff)
	assert.Equal(t, 512*time.Millisecond, config.MaxRetryBackoff)
	assert.True(t, config.EnableMetrics)
}

func TestRedisConfig_ApplyDefaults(t *testing.T) {
	tests := []struct {
		name   string
		config *RedisConfig
		verify func(t *testing.T, config *RedisConfig)
	}{
		{
			name:   "empty config gets all defaults",
			config: &RedisConfig{},
			verify: func(t *testing.T, config *RedisConfig) {
				assert.Equal(t, RedisModeStandalone, config.Mode)
				assert.Equal(t, "saga:", config.KeyPrefix)
				assert.Equal(t, 24*time.Hour, config.TTL)
				assert.Equal(t, 5*time.Second, config.DialTimeout)
				assert.Equal(t, 3*time.Second, config.ReadTimeout)
				assert.Equal(t, 3*time.Second, config.WriteTimeout)
				assert.Equal(t, 10, config.PoolSize)
				assert.Equal(t, 5, config.MinIdleConns)
			},
		},
		{
			name: "partial config preserves values",
			config: &RedisConfig{
				Addr:      "redis:6379",
				PoolSize:  20,
				KeyPrefix: "test:",
			},
			verify: func(t *testing.T, config *RedisConfig) {
				assert.Equal(t, "redis:6379", config.Addr)
				assert.Equal(t, 20, config.PoolSize)
				assert.Equal(t, "test:", config.KeyPrefix)
				assert.Equal(t, 5*time.Second, config.DialTimeout) // default applied
			},
		},
		{
			name: "max idle conns defaults to pool size",
			config: &RedisConfig{
				PoolSize: 15,
			},
			verify: func(t *testing.T, config *RedisConfig) {
				assert.Equal(t, 15, config.PoolSize)
				assert.Equal(t, 15, config.MaxIdleConns)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.ApplyDefaults()
			tt.verify(t, tt.config)
		})
	}
}

func TestRedisConfig_Validate_Standalone(t *testing.T) {
	tests := []struct {
		name        string
		config      *RedisConfig
		expectError bool
		errorType   error
	}{
		{
			name: "valid standalone config",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				DB:   0,
			},
			expectError: false,
		},
		{
			name: "valid standalone config with addrs",
			config: &RedisConfig{
				Mode:  RedisModeStandalone,
				Addrs: []string{"localhost:6379"},
				DB:    0,
			},
			expectError: false,
		},
		{
			name: "missing address",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				DB:   0,
			},
			expectError: true,
			errorType:   ErrInvalidRedisConfig,
		},
		{
			name: "invalid DB number",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				DB:   -1,
			},
			expectError: true,
			errorType:   ErrInvalidDB,
		},
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorType:   ErrInvalidRedisConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisConfig_Validate_Cluster(t *testing.T) {
	tests := []struct {
		name        string
		config      *RedisConfig
		expectError bool
	}{
		{
			name: "valid cluster config with addrs",
			config: &RedisConfig{
				Mode:  RedisModeCluster,
				Addrs: []string{"node1:6379", "node2:6379", "node3:6379"},
			},
			expectError: false,
		},
		{
			name: "valid cluster config with addr",
			config: &RedisConfig{
				Mode: RedisModeCluster,
				Addr: "node1:6379,node2:6379",
			},
			expectError: false,
		},
		{
			name: "missing addresses",
			config: &RedisConfig{
				Mode: RedisModeCluster,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisConfig_Validate_Sentinel(t *testing.T) {
	tests := []struct {
		name        string
		config      *RedisConfig
		expectError bool
	}{
		{
			name: "valid sentinel config",
			config: &RedisConfig{
				Mode:          RedisModeSentinel,
				MasterName:    "mymaster",
				SentinelAddrs: []string{"sentinel1:26379", "sentinel2:26379"},
			},
			expectError: false,
		},
		{
			name: "missing master name",
			config: &RedisConfig{
				Mode:          RedisModeSentinel,
				SentinelAddrs: []string{"sentinel1:26379"},
			},
			expectError: true,
		},
		{
			name: "missing sentinel addresses",
			config: &RedisConfig{
				Mode:       RedisModeSentinel,
				MasterName: "mymaster",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisConfig_Validate_Timeouts(t *testing.T) {
	tests := []struct {
		name        string
		config      *RedisConfig
		expectError bool
		errorType   error
	}{
		{
			name: "valid timeouts",
			config: &RedisConfig{
				Mode:         RedisModeStandalone,
				Addr:         "localhost:6379",
				DialTimeout:  5 * time.Second,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 3 * time.Second,
			},
			expectError: false,
		},
		{
			name: "negative dial timeout",
			config: &RedisConfig{
				Mode:        RedisModeStandalone,
				Addr:        "localhost:6379",
				DialTimeout: -1 * time.Second,
			},
			expectError: true,
			errorType:   ErrInvalidTimeout,
		},
		{
			name: "negative read timeout",
			config: &RedisConfig{
				Mode:        RedisModeStandalone,
				Addr:        "localhost:6379",
				ReadTimeout: -1 * time.Second,
			},
			expectError: true,
			errorType:   ErrInvalidTimeout,
		},
		{
			name: "negative write timeout",
			config: &RedisConfig{
				Mode:         RedisModeStandalone,
				Addr:         "localhost:6379",
				WriteTimeout: -1 * time.Second,
			},
			expectError: true,
			errorType:   ErrInvalidTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisConfig_Validate_PoolConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *RedisConfig
		expectError bool
		errorType   error
	}{
		{
			name: "valid pool config",
			config: &RedisConfig{
				Mode:         RedisModeStandalone,
				Addr:         "localhost:6379",
				PoolSize:     10,
				MinIdleConns: 5,
				MaxIdleConns: 10,
			},
			expectError: false,
		},
		{
			name: "negative pool size",
			config: &RedisConfig{
				Mode:     RedisModeStandalone,
				Addr:     "localhost:6379",
				PoolSize: -1,
			},
			expectError: true,
			errorType:   ErrInvalidPoolSize,
		},
		{
			name: "negative min idle conns",
			config: &RedisConfig{
				Mode:         RedisModeStandalone,
				Addr:         "localhost:6379",
				MinIdleConns: -1,
			},
			expectError: true,
			errorType:   ErrInvalidPoolSize,
		},
		{
			name: "negative max idle conns",
			config: &RedisConfig{
				Mode:         RedisModeStandalone,
				Addr:         "localhost:6379",
				MaxIdleConns: -1,
			},
			expectError: true,
			errorType:   ErrInvalidPoolSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisConfig_Validate_RetryConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *RedisConfig
		expectError bool
		errorType   error
	}{
		{
			name: "valid retry config",
			config: &RedisConfig{
				Mode:            RedisModeStandalone,
				Addr:            "localhost:6379",
				MaxRetries:      3,
				MinRetryBackoff: 8 * time.Millisecond,
				MaxRetryBackoff: 512 * time.Millisecond,
			},
			expectError: false,
		},
		{
			name: "negative max retries",
			config: &RedisConfig{
				Mode:       RedisModeStandalone,
				Addr:       "localhost:6379",
				MaxRetries: -1,
			},
			expectError: true,
			errorType:   ErrInvalidMaxRetries,
		},
		{
			name: "negative min retry backoff",
			config: &RedisConfig{
				Mode:            RedisModeStandalone,
				Addr:            "localhost:6379",
				MinRetryBackoff: -1 * time.Millisecond,
			},
			expectError: true,
			errorType:   ErrInvalidTimeout,
		},
		{
			name: "min retry backoff greater than max",
			config: &RedisConfig{
				Mode:            RedisModeStandalone,
				Addr:            "localhost:6379",
				MinRetryBackoff: 512 * time.Millisecond,
				MaxRetryBackoff: 8 * time.Millisecond,
			},
			expectError: true,
			errorType:   ErrInvalidTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisConfig_Validate_TTL(t *testing.T) {
	tests := []struct {
		name        string
		config      *RedisConfig
		expectError bool
	}{
		{
			name: "valid TTL",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				TTL:  24 * time.Hour,
			},
			expectError: false,
		},
		{
			name: "zero TTL is valid",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				TTL:  0,
			},
			expectError: false,
		},
		{
			name: "negative TTL",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				TTL:  -1 * time.Hour,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisConfig_ValidateTLSConfig(t *testing.T) {
	// Create temporary test files
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	caFile := filepath.Join(tmpDir, "ca.pem")

	// Create dummy files
	require.NoError(t, os.WriteFile(certFile, []byte("cert"), 0600))
	require.NoError(t, os.WriteFile(keyFile, []byte("key"), 0600))
	require.NoError(t, os.WriteFile(caFile, []byte("ca"), 0600))

	tests := []struct {
		name        string
		config      *RedisConfig
		expectError bool
		errorType   error
	}{
		{
			name: "valid TLS config with all files",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				TLS: &RedisTLSConfig{
					Enabled:  true,
					CertFile: certFile,
					KeyFile:  keyFile,
					CAFile:   caFile,
				},
			},
			expectError: false,
		},
		{
			name: "valid TLS config with only CA",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				TLS: &RedisTLSConfig{
					Enabled: true,
					CAFile:  caFile,
				},
			},
			expectError: false,
		},
		{
			name: "TLS disabled",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				TLS: &RedisTLSConfig{
					Enabled: false,
				},
			},
			expectError: false,
		},
		{
			name: "cert without key",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				TLS: &RedisTLSConfig{
					Enabled:  true,
					CertFile: certFile,
				},
			},
			expectError: true,
			errorType:   ErrInvalidRedisConfig,
		},
		{
			name: "key without cert",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				TLS: &RedisTLSConfig{
					Enabled: true,
					KeyFile: keyFile,
				},
			},
			expectError: true,
			errorType:   ErrInvalidRedisConfig,
		},
		{
			name: "non-existent cert file",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				TLS: &RedisTLSConfig{
					Enabled:  true,
					CertFile: "/nonexistent/cert.pem",
					KeyFile:  keyFile,
				},
			},
			expectError: true,
			errorType:   ErrTLSCertNotFound,
		},
		{
			name: "non-existent key file",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				TLS: &RedisTLSConfig{
					Enabled:  true,
					CertFile: certFile,
					KeyFile:  "/nonexistent/key.pem",
				},
			},
			expectError: true,
			errorType:   ErrTLSKeyNotFound,
		},
		{
			name: "non-existent CA file",
			config: &RedisConfig{
				Mode: RedisModeStandalone,
				Addr: "localhost:6379",
				TLS: &RedisTLSConfig{
					Enabled: true,
					CAFile:  "/nonexistent/ca.pem",
				},
			},
			expectError: true,
			errorType:   ErrTLSCANotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisConfig_BuildTLSConfig(t *testing.T) {
	// Create temporary test files
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	caFile := filepath.Join(tmpDir, "ca.pem")

	// Create valid certificate files for testing
	// Note: These are dummy certificates just for testing file loading
	certPEM := `-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`

	keyPEM := `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`

	caPEM := `-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`

	require.NoError(t, os.WriteFile(certFile, []byte(certPEM), 0600))
	require.NoError(t, os.WriteFile(keyFile, []byte(keyPEM), 0600))
	require.NoError(t, os.WriteFile(caFile, []byte(caPEM), 0600))

	tests := []struct {
		name        string
		config      *RedisConfig
		expectNil   bool
		expectError bool
		verify      func(t *testing.T, tlsConfig *tls.Config)
	}{
		{
			name: "no TLS config",
			config: &RedisConfig{
				TLS: nil,
			},
			expectNil:   true,
			expectError: false,
		},
		{
			name: "TLS disabled",
			config: &RedisConfig{
				TLS: &RedisTLSConfig{
					Enabled: false,
				},
			},
			expectNil:   true,
			expectError: false,
		},
		{
			name: "TLS enabled with insecure skip verify",
			config: &RedisConfig{
				TLS: &RedisTLSConfig{
					Enabled:            true,
					InsecureSkipVerify: true,
				},
			},
			expectNil:   false,
			expectError: false,
			verify: func(t *testing.T, tlsConfig *tls.Config) {
				assert.True(t, tlsConfig.InsecureSkipVerify)
			},
		},
		{
			name: "TLS enabled with server name",
			config: &RedisConfig{
				TLS: &RedisTLSConfig{
					Enabled:    true,
					ServerName: "redis.example.com",
				},
			},
			expectNil:   false,
			expectError: false,
			verify: func(t *testing.T, tlsConfig *tls.Config) {
				assert.Equal(t, "redis.example.com", tlsConfig.ServerName)
			},
		},
		{
			name: "TLS with client certificate",
			config: &RedisConfig{
				TLS: &RedisTLSConfig{
					Enabled:  true,
					CertFile: certFile,
					KeyFile:  keyFile,
				},
			},
			expectNil:   false,
			expectError: false,
			verify: func(t *testing.T, tlsConfig *tls.Config) {
				assert.Len(t, tlsConfig.Certificates, 1)
			},
		},
		{
			name: "TLS with CA certificate",
			config: &RedisConfig{
				TLS: &RedisTLSConfig{
					Enabled: true,
					CAFile:  caFile,
				},
			},
			expectNil:   false,
			expectError: false,
			verify: func(t *testing.T, tlsConfig *tls.Config) {
				assert.NotNil(t, tlsConfig.RootCAs)
			},
		},
		{
			name: "TLS with all certificates",
			config: &RedisConfig{
				TLS: &RedisTLSConfig{
					Enabled:            true,
					CertFile:           certFile,
					KeyFile:            keyFile,
					CAFile:             caFile,
					ServerName:         "redis.example.com",
					InsecureSkipVerify: false,
				},
			},
			expectNil:   false,
			expectError: false,
			verify: func(t *testing.T, tlsConfig *tls.Config) {
				assert.Len(t, tlsConfig.Certificates, 1)
				assert.NotNil(t, tlsConfig.RootCAs)
				assert.Equal(t, "redis.example.com", tlsConfig.ServerName)
				assert.False(t, tlsConfig.InsecureSkipVerify)
			},
		},
		{
			name: "TLS with invalid cert file",
			config: &RedisConfig{
				TLS: &RedisTLSConfig{
					Enabled:  true,
					CertFile: "/nonexistent/cert.pem",
					KeyFile:  keyFile,
				},
			},
			expectNil:   false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig, err := tt.config.BuildTLSConfig()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectNil {
				assert.Nil(t, tlsConfig)
			} else if !tt.expectError {
				assert.NotNil(t, tlsConfig)
				if tt.verify != nil {
					tt.verify(t, tlsConfig)
				}
			}
		})
	}
}
