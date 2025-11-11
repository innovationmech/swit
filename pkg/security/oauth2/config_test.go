// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oauth2

import (
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config with discovery",
			config: &Config{
				Enabled:      true,
				ClientID:     "client123",
				ClientSecret: "secret123",
				IssuerURL:    "https://example.com",
				UseDiscovery: true,
			},
			wantErr: false,
		},
		{
			name: "valid config without discovery",
			config: &Config{
				Enabled:      true,
				ClientID:     "client123",
				ClientSecret: "secret123",
				AuthURL:      "https://example.com/auth",
				TokenURL:     "https://example.com/token",
				JWKSURL:      "https://example.com/jwks",
				UseDiscovery: false,
			},
			wantErr: false,
		},
		{
			name: "disabled config",
			config: &Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &Config{
				Enabled:      true,
				ClientSecret: "secret123",
				IssuerURL:    "https://example.com",
				UseDiscovery: true,
			},
			wantErr: true,
		},
		{
			name: "missing client secret",
			config: &Config{
				Enabled:      true,
				ClientID:     "client123",
				IssuerURL:    "https://example.com",
				UseDiscovery: true,
			},
			wantErr: true,
		},
		{
			name: "discovery enabled but no issuer URL",
			config: &Config{
				Enabled:      true,
				ClientID:     "client123",
				ClientSecret: "secret123",
				UseDiscovery: true,
			},
			wantErr: true,
		},
		{
			name: "discovery disabled but missing auth URL",
			config: &Config{
				Enabled:      true,
				ClientID:     "client123",
				ClientSecret: "secret123",
				TokenURL:     "https://example.com/token",
				JWKSURL:      "https://example.com/jwks",
				UseDiscovery: false,
			},
			wantErr: true,
		},
		{
			name: "discovery disabled but missing token URL",
			config: &Config{
				Enabled:      true,
				ClientID:     "client123",
				ClientSecret: "secret123",
				AuthURL:      "https://example.com/auth",
				JWKSURL:      "https://example.com/jwks",
				UseDiscovery: false,
			},
			wantErr: true,
		},
		{
			name: "discovery disabled but missing JWKS URL",
			config: &Config{
				Enabled:      true,
				ClientID:     "client123",
				ClientSecret: "secret123",
				AuthURL:      "https://example.com/auth",
				TokenURL:     "https://example.com/token",
				UseDiscovery: false,
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

func TestConfig_SetDefaults(t *testing.T) {
	t.Run("sets default provider", func(t *testing.T) {
		config := &Config{}
		config.SetDefaults()

		if config.Provider != "custom" {
			t.Errorf("Expected default provider to be 'custom', got %q", config.Provider)
		}
	})

	t.Run("sets default scopes", func(t *testing.T) {
		config := &Config{}
		config.SetDefaults()

		expectedScopes := []string{"openid", "profile", "email"}
		if len(config.Scopes) != len(expectedScopes) {
			t.Errorf("Expected %d default scopes, got %d", len(expectedScopes), len(config.Scopes))
		}

		for i, scope := range expectedScopes {
			if config.Scopes[i] != scope {
				t.Errorf("Expected scope[%d] to be %q, got %q", i, scope, config.Scopes[i])
			}
		}
	})

	t.Run("sets default HTTP timeout", func(t *testing.T) {
		config := &Config{}
		config.SetDefaults()

		expectedTimeout := 30 * time.Second
		if config.HTTPTimeout != expectedTimeout {
			t.Errorf("Expected default HTTP timeout to be %v, got %v", expectedTimeout, config.HTTPTimeout)
		}
	})

	t.Run("does not override existing values", func(t *testing.T) {
		config := &Config{
			Provider:    "keycloak",
			Scopes:      []string{"custom:scope"},
			HTTPTimeout: 60 * time.Second,
		}
		config.SetDefaults()

		if config.Provider != "keycloak" {
			t.Errorf("Expected provider to remain 'keycloak', got %q", config.Provider)
		}

		if len(config.Scopes) != 1 || config.Scopes[0] != "custom:scope" {
			t.Errorf("Expected scopes to remain ['custom:scope'], got %v", config.Scopes)
		}

		if config.HTTPTimeout != 60*time.Second {
			t.Errorf("Expected HTTP timeout to remain 60s, got %v", config.HTTPTimeout)
		}
	})

	t.Run("enables discovery when issuer URL is set", func(t *testing.T) {
		config := &Config{
			IssuerURL:    "https://example.com",
			UseDiscovery: true,
		}
		config.SetDefaults()

		if !config.UseDiscovery {
			t.Error("Expected UseDiscovery to be true when IssuerURL is set")
		}
	})
}

func TestConfig_ValidateWithDefaults(t *testing.T) {
	t.Run("valid after setting defaults", func(t *testing.T) {
		config := &Config{
			Enabled:      true,
			ClientID:     "client123",
			ClientSecret: "secret123",
			IssuerURL:    "https://example.com",
			UseDiscovery: true,
		}
		config.SetDefaults()

		if err := config.Validate(); err != nil {
			t.Errorf("Expected validation to pass after setting defaults, got error: %v", err)
		}

		if config.HTTPTimeout != 30*time.Second {
			t.Errorf("Expected default HTTP timeout to be set during validation, got %v", config.HTTPTimeout)
		}
	})
}
