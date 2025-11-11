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

