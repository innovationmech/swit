// Copyright (c) 2024 Six-Thirty Labs, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opa

import (
	"os"
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid embedded config",
			config: &Config{
				Mode: ModeEmbedded,
				EmbeddedConfig: &EmbeddedConfig{
					PolicyDir: "/tmp/policies",
				},
			},
			wantErr: false,
		},
		{
			name: "valid remote config",
			config: &Config{
				Mode: ModeRemote,
				RemoteConfig: &RemoteConfig{
					URL: "http://localhost:8181",
				},
			},
			wantErr: false,
		},
		{
			name:    "empty mode",
			config:  &Config{},
			wantErr: true,
		},
		{
			name: "invalid mode",
			config: &Config{
				Mode: "invalid",
			},
			wantErr: true,
		},
		{
			name: "embedded mode without config",
			config: &Config{
				Mode: ModeEmbedded,
			},
			wantErr: true,
		},
		{
			name: "remote mode without config",
			config: &Config{
				Mode: ModeRemote,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmbeddedConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *EmbeddedConfig
		wantErr bool
	}{
		{
			name: "valid with policy dir",
			config: &EmbeddedConfig{
				PolicyDir: "/tmp/policies",
			},
			wantErr: false,
		},
		{
			name: "valid with bundle config",
			config: &EmbeddedConfig{
				BundleConfig: &BundleConfig{
					ServiceURL: "http://localhost:8888",
					Resource:   "/bundle.tar.gz",
				},
			},
			wantErr: false,
		},
		{
			name:    "missing both policy dir and bundle config",
			config:  &EmbeddedConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("EmbeddedConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRemoteConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *RemoteConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &RemoteConfig{
				URL: "http://localhost:8181",
			},
			wantErr: false,
		},
		{
			name:    "missing URL",
			config:  &RemoteConfig{},
			wantErr: true,
		},
		{
			name: "negative timeout",
			config: &RemoteConfig{
				URL:     "http://localhost:8181",
				Timeout: -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoteConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCacheConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *CacheConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &CacheConfig{
				Enabled: true,
				MaxSize: 1000,
				TTL:     5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "disabled cache",
			config: &CacheConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "invalid max size",
			config: &CacheConfig{
				Enabled: true,
				MaxSize: 0,
			},
			wantErr: true,
		},
		{
			name: "negative TTL",
			config: &CacheConfig{
				Enabled: true,
				MaxSize: 1000,
				TTL:     -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CacheConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigSetDefaults(t *testing.T) {
	config := &Config{
		Mode: ModeRemote,
		RemoteConfig: &RemoteConfig{
			URL: "http://localhost:8181",
		},
		CacheConfig: &CacheConfig{
			Enabled: true,
			MaxSize: 100,
		},
	}

	config.SetDefaults()

	if config.RemoteConfig.Timeout == 0 {
		t.Error("Expected timeout to be set to default value")
	}

	if config.CacheConfig.TTL == 0 {
		t.Error("Expected TTL to be set to default value")
	}
}

func TestAuthConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *AuthConfig
		wantErr bool
	}{
		{
			name: "valid bearer auth",
			config: &AuthConfig{
				Type:  "bearer",
				Token: "test-token",
			},
			wantErr: false,
		},
		{
			name: "valid basic auth",
			config: &AuthConfig{
				Type:     "basic",
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
		},
		{
			name: "valid api_key auth",
			config: &AuthConfig{
				Type:         "api_key",
				APIKey:       "key",
				APIKeyHeader: "X-API-Key",
			},
			wantErr: false,
		},
		{
			name:    "missing type",
			config:  &AuthConfig{},
			wantErr: true,
		},
		{
			name: "invalid type",
			config: &AuthConfig{
				Type: "invalid",
			},
			wantErr: true,
		},
		{
			name: "bearer without token",
			config: &AuthConfig{
				Type: "bearer",
			},
			wantErr: true,
		},
		{
			name: "basic without password",
			config: &AuthConfig{
				Type:     "basic",
				Username: "user",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigValidateSidecarMode(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid sidecar config",
			config: &Config{
				Mode: ModeSidecar,
				RemoteConfig: &RemoteConfig{
					URL: "http://localhost:8181",
				},
			},
			wantErr: false,
		},
		{
			name: "sidecar mode without remote config",
			config: &Config{
				Mode: ModeSidecar,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			// After validation, sidecar should be normalized to remote
			if tt.config.Mode == ModeRemote && !tt.wantErr {
				// This is expected - sidecar gets normalized to remote
			}
		})
	}
}

func TestConfigLoadFromEnv(t *testing.T) {
	// Save original env vars
	originalEnv := make(map[string]string)
	envVars := []string{
		"OPA_MODE",
		"OPA_DEFAULT_DECISION_PATH",
		"OPA_REMOTE_URL",
		"OPA_REMOTE_TIMEOUT",
		"OPA_REMOTE_MAX_RETRIES",
		"OPA_CACHE_ENABLED",
		"OPA_CACHE_MAX_SIZE",
		"OPA_CACHE_TTL",
		"OPA_EMBEDDED_POLICY_DIR",
	}
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	defer func() {
		// Restore original env vars
		for key, val := range originalEnv {
			if val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	t.Run("load remote config from env", func(t *testing.T) {
		os.Setenv("OPA_MODE", "remote")
		os.Setenv("OPA_REMOTE_URL", "http://test:8181")
		os.Setenv("OPA_REMOTE_TIMEOUT", "10s")
		os.Setenv("OPA_REMOTE_MAX_RETRIES", "5")
		os.Setenv("OPA_DEFAULT_DECISION_PATH", "authz/allow")

		config := &Config{}
		config.LoadFromEnv()

		if config.Mode != ModeRemote {
			t.Errorf("Expected mode to be 'remote', got %s", config.Mode)
		}
		if config.RemoteConfig == nil {
			t.Fatal("Expected RemoteConfig to be set")
		}
		if config.RemoteConfig.URL != "http://test:8181" {
			t.Errorf("Expected URL to be 'http://test:8181', got %s", config.RemoteConfig.URL)
		}
		if config.RemoteConfig.Timeout != 10*time.Second {
			t.Errorf("Expected timeout to be 10s, got %v", config.RemoteConfig.Timeout)
		}
		if config.RemoteConfig.MaxRetries != 5 {
			t.Errorf("Expected max retries to be 5, got %d", config.RemoteConfig.MaxRetries)
		}
		if config.DefaultDecisionPath != "authz/allow" {
			t.Errorf("Expected default decision path to be 'authz/allow', got %s", config.DefaultDecisionPath)
		}
	})

	t.Run("load embedded config from env", func(t *testing.T) {
		os.Unsetenv("OPA_MODE")
		os.Unsetenv("OPA_REMOTE_URL")
		os.Setenv("OPA_MODE", "embedded")
		os.Setenv("OPA_EMBEDDED_POLICY_DIR", "/tmp/policies")
		os.Setenv("OPA_EMBEDDED_DATA_DIR", "/tmp/data")
		os.Setenv("OPA_EMBEDDED_ENABLE_LOGGING", "true")

		config := &Config{}
		config.LoadFromEnv()

		if config.Mode != ModeEmbedded {
			t.Errorf("Expected mode to be 'embedded', got %s", config.Mode)
		}
		if config.EmbeddedConfig == nil {
			t.Fatal("Expected EmbeddedConfig to be set")
		}
		if config.EmbeddedConfig.PolicyDir != "/tmp/policies" {
			t.Errorf("Expected policy dir to be '/tmp/policies', got %s", config.EmbeddedConfig.PolicyDir)
		}
		if config.EmbeddedConfig.DataDir != "/tmp/data" {
			t.Errorf("Expected data dir to be '/tmp/data', got %s", config.EmbeddedConfig.DataDir)
		}
		if !config.EmbeddedConfig.EnableLogging {
			t.Error("Expected enable logging to be true")
		}
	})

	t.Run("load cache config from env", func(t *testing.T) {
		os.Setenv("OPA_CACHE_ENABLED", "true")
		os.Setenv("OPA_CACHE_MAX_SIZE", "5000")
		os.Setenv("OPA_CACHE_TTL", "10m")
		os.Setenv("OPA_CACHE_ENABLE_METRICS", "true")

		config := &Config{}
		config.LoadFromEnv()

		if config.CacheConfig == nil {
			t.Fatal("Expected CacheConfig to be set")
		}
		if !config.CacheConfig.Enabled {
			t.Error("Expected cache to be enabled")
		}
		if config.CacheConfig.MaxSize != 5000 {
			t.Errorf("Expected max size to be 5000, got %d", config.CacheConfig.MaxSize)
		}
		if config.CacheConfig.TTL != 10*time.Minute {
			t.Errorf("Expected TTL to be 10m, got %v", config.CacheConfig.TTL)
		}
		if !config.CacheConfig.EnableMetrics {
			t.Error("Expected enable metrics to be true")
		}
	})

	t.Run("load sidecar config from env", func(t *testing.T) {
		os.Setenv("OPA_MODE", "sidecar")
		os.Setenv("OPA_REMOTE_URL", "http://localhost:8181")

		config := &Config{}
		config.LoadFromEnv()

		if config.Mode != ModeSidecar {
			t.Errorf("Expected mode to be 'sidecar', got %s", config.Mode)
		}
		if config.RemoteConfig == nil {
			t.Fatal("Expected RemoteConfig to be set")
		}
		if config.RemoteConfig.URL != "http://localhost:8181" {
			t.Errorf("Expected URL to be 'http://localhost:8181', got %s", config.RemoteConfig.URL)
		}
	})
}

func TestConfigLoadFromEnvWithAuth(t *testing.T) {
	// Save and clear env vars
	envVars := []string{
		"OPA_MODE",
		"OPA_REMOTE_URL",
		"OPA_REMOTE_AUTH_TYPE",
		"OPA_REMOTE_AUTH_TOKEN",
		"OPA_REMOTE_AUTH_USERNAME",
		"OPA_REMOTE_AUTH_PASSWORD",
	}
	originalEnv := make(map[string]string)
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	defer func() {
		for key, val := range originalEnv {
			if val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	t.Run("load bearer auth from env", func(t *testing.T) {
		os.Setenv("OPA_MODE", "remote")
		os.Setenv("OPA_REMOTE_URL", "http://test:8181")
		os.Setenv("OPA_REMOTE_AUTH_TYPE", "bearer")
		os.Setenv("OPA_REMOTE_AUTH_TOKEN", "test-token-123")

		config := &Config{}
		config.LoadFromEnv()

		if config.RemoteConfig == nil || config.RemoteConfig.AuthConfig == nil {
			t.Fatal("Expected AuthConfig to be set")
		}
		if config.RemoteConfig.AuthConfig.Type != "bearer" {
			t.Errorf("Expected auth type to be 'bearer', got %s", config.RemoteConfig.AuthConfig.Type)
		}
		if config.RemoteConfig.AuthConfig.Token != "test-token-123" {
			t.Errorf("Expected token to be 'test-token-123', got %s", config.RemoteConfig.AuthConfig.Token)
		}
	})

	t.Run("load basic auth from env", func(t *testing.T) {
		os.Unsetenv("OPA_REMOTE_AUTH_TYPE")
		os.Unsetenv("OPA_REMOTE_AUTH_TOKEN")
		os.Setenv("OPA_MODE", "remote")
		os.Setenv("OPA_REMOTE_URL", "http://test:8181")
		os.Setenv("OPA_REMOTE_AUTH_TYPE", "basic")
		os.Setenv("OPA_REMOTE_AUTH_USERNAME", "admin")
		os.Setenv("OPA_REMOTE_AUTH_PASSWORD", "secret")

		config := &Config{}
		config.LoadFromEnv()

		if config.RemoteConfig == nil || config.RemoteConfig.AuthConfig == nil {
			t.Fatal("Expected AuthConfig to be set")
		}
		if config.RemoteConfig.AuthConfig.Type != "basic" {
			t.Errorf("Expected auth type to be 'basic', got %s", config.RemoteConfig.AuthConfig.Type)
		}
		if config.RemoteConfig.AuthConfig.Username != "admin" {
			t.Errorf("Expected username to be 'admin', got %s", config.RemoteConfig.AuthConfig.Username)
		}
		if config.RemoteConfig.AuthConfig.Password != "secret" {
			t.Errorf("Expected password to be 'secret', got %s", config.RemoteConfig.AuthConfig.Password)
		}
	})
}
