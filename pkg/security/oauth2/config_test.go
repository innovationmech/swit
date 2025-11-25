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

package oauth2

import (
	"os"
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

func TestJWTConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *JWTConfig
		wantErr bool
	}{
		{
			name: "valid RS256",
			config: &JWTConfig{
				SigningMethod: "RS256",
				ClockSkew:     30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid HS256",
			config: &JWTConfig{
				SigningMethod: "HS256",
			},
			wantErr: false,
		},
		{
			name: "valid ES256",
			config: &JWTConfig{
				SigningMethod: "ES256",
			},
			wantErr: false,
		},
		{
			name: "valid PS256",
			config: &JWTConfig{
				SigningMethod: "PS256",
			},
			wantErr: false,
		},
		{
			name: "invalid signing method",
			config: &JWTConfig{
				SigningMethod: "INVALID",
			},
			wantErr: true,
		},
		{
			name:    "empty signing method is valid (will use default)",
			config:  &JWTConfig{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("JWTConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJWTConfig_SetDefaults(t *testing.T) {
	t.Run("sets default signing method", func(t *testing.T) {
		config := &JWTConfig{}
		config.SetDefaults()

		if config.SigningMethod != "RS256" {
			t.Errorf("Expected default signing method to be 'RS256', got %q", config.SigningMethod)
		}
	})

	t.Run("sets default clock skew", func(t *testing.T) {
		config := &JWTConfig{}
		config.SetDefaults()

		expectedClockSkew := 30 * time.Second
		if config.ClockSkew != expectedClockSkew {
			t.Errorf("Expected default clock skew to be %v, got %v", expectedClockSkew, config.ClockSkew)
		}
	})

	t.Run("does not override existing values", func(t *testing.T) {
		config := &JWTConfig{
			SigningMethod: "HS256",
			ClockSkew:     60 * time.Second,
		}
		config.SetDefaults()

		if config.SigningMethod != "HS256" {
			t.Errorf("Expected signing method to remain 'HS256', got %q", config.SigningMethod)
		}

		if config.ClockSkew != 60*time.Second {
			t.Errorf("Expected clock skew to remain 60s, got %v", config.ClockSkew)
		}
	})
}

func TestTLSConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *TLSConfig
		wantErr bool
	}{
		{
			name: "disabled config is valid",
			config: &TLSConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "cert without key",
			config: &TLSConfig{
				Enabled:  true,
				CertFile: "/path/to/cert.pem",
			},
			wantErr: true,
		},
		{
			name: "key without cert",
			config: &TLSConfig{
				Enabled: true,
				KeyFile: "/path/to/key.pem",
			},
			wantErr: true,
		},
		{
			name: "invalid min version",
			config: &TLSConfig{
				Enabled:    true,
				MinVersion: "TLS0.9",
			},
			wantErr: true,
		},
		{
			name: "invalid max version",
			config: &TLSConfig{
				Enabled:    true,
				MaxVersion: "TLS2.0",
			},
			wantErr: true,
		},
		{
			name: "valid TLS versions",
			config: &TLSConfig{
				Enabled:    true,
				MinVersion: "TLS1.2",
				MaxVersion: "TLS1.3",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TLSConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTLSConfig_SetDefaults(t *testing.T) {
	t.Run("sets default min version when enabled", func(t *testing.T) {
		config := &TLSConfig{
			Enabled: true,
		}
		config.SetDefaults()

		if config.MinVersion != "TLS1.2" {
			t.Errorf("Expected default min version to be 'TLS1.2', got %q", config.MinVersion)
		}
	})

	t.Run("does not set defaults when disabled", func(t *testing.T) {
		config := &TLSConfig{
			Enabled: false,
		}
		config.SetDefaults()

		if config.MinVersion != "" {
			t.Errorf("Expected no default min version when disabled, got %q", config.MinVersion)
		}
	})

	t.Run("does not override existing values", func(t *testing.T) {
		config := &TLSConfig{
			Enabled:    true,
			MinVersion: "TLS1.3",
		}
		config.SetDefaults()

		if config.MinVersion != "TLS1.3" {
			t.Errorf("Expected min version to remain 'TLS1.3', got %q", config.MinVersion)
		}
	})
}

func TestTLSConfig_ToGoTLSConfig(t *testing.T) {
	t.Run("returns nil when disabled", func(t *testing.T) {
		config := &TLSConfig{
			Enabled: false,
		}
		tlsConfig, err := config.ToGoTLSConfig()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if tlsConfig != nil {
			t.Error("Expected nil TLS config when disabled")
		}
	})

	t.Run("creates config with insecure skip verify", func(t *testing.T) {
		config := &TLSConfig{
			Enabled:            true,
			InsecureSkipVerify: true,
		}
		tlsConfig, err := config.ToGoTLSConfig()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if tlsConfig == nil {
			t.Fatal("Expected non-nil TLS config")
		}
		if !tlsConfig.InsecureSkipVerify {
			t.Error("Expected InsecureSkipVerify to be true")
		}
	})

	t.Run("sets TLS versions correctly", func(t *testing.T) {
		config := &TLSConfig{
			Enabled:    true,
			MinVersion: "TLS1.2",
			MaxVersion: "TLS1.3",
		}
		tlsConfig, err := config.ToGoTLSConfig()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if tlsConfig == nil {
			t.Fatal("Expected non-nil TLS config")
		}
		// Check that versions are set (exact values depend on crypto/tls constants)
		if tlsConfig.MinVersion == 0 {
			t.Error("Expected MinVersion to be set")
		}
		if tlsConfig.MaxVersion == 0 {
			t.Error("Expected MaxVersion to be set")
		}
	})
}

func TestConfig_SubConfigValidation(t *testing.T) {
	t.Run("validates JWT config", func(t *testing.T) {
		config := &Config{
			Enabled:      true,
			ClientID:     "client123",
			ClientSecret: "secret123",
			IssuerURL:    "https://example.com",
			UseDiscovery: true,
			JWTConfig: JWTConfig{
				SigningMethod: "INVALID",
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("Expected validation to fail for invalid JWT config")
		}
	})

	t.Run("validates cache config", func(t *testing.T) {
		config := &Config{
			Enabled:      true,
			ClientID:     "client123",
			ClientSecret: "secret123",
			IssuerURL:    "https://example.com",
			UseDiscovery: true,
			CacheConfig: TokenCacheConfig{
				Enabled: true,
				TTL:     -1 * time.Second,
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("Expected validation to fail for invalid cache config")
		}
	})

	t.Run("validates TLS config", func(t *testing.T) {
		config := &Config{
			Enabled:      true,
			ClientID:     "client123",
			ClientSecret: "secret123",
			IssuerURL:    "https://example.com",
			UseDiscovery: true,
			TLSConfig: TLSConfig{
				Enabled:    true,
				MinVersion: "INVALID",
			},
		}

		err := config.Validate()
		if err == nil {
			t.Error("Expected validation to fail for invalid TLS config")
		}
	})
}

func TestConfig_SetDefaultsSubConfigs(t *testing.T) {
	t.Run("sets defaults for all sub-configs", func(t *testing.T) {
		config := &Config{
			TLSConfig: TLSConfig{
				Enabled: true,
			},
		}
		config.SetDefaults()

		// Check JWT defaults
		if config.JWTConfig.SigningMethod != "RS256" {
			t.Errorf("Expected JWT signing method to be 'RS256', got %q", config.JWTConfig.SigningMethod)
		}

		if config.JWTConfig.ClockSkew != 30*time.Second {
			t.Errorf("Expected JWT clock skew to be 30s, got %v", config.JWTConfig.ClockSkew)
		}

		// Check TLS defaults
		if config.TLSConfig.MinVersion != "TLS1.2" {
			t.Errorf("Expected TLS min version to be 'TLS1.2', got %q", config.TLSConfig.MinVersion)
		}
	})
}

func TestParseTLSVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected uint16
	}{
		{"TLS 1.0", "TLS1.0", 0x0301},
		{"TLS 1.1", "TLS1.1", 0x0302},
		{"TLS 1.2", "TLS1.2", 0x0303},
		{"TLS 1.3", "TLS1.3", 0x0304},
		{"Unknown defaults to TLS 1.2", "UNKNOWN", 0x0303},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTLSVersion(tt.version)
			if result != tt.expected {
				t.Errorf("parseTLSVersion(%q) = 0x%04x, expected 0x%04x", tt.version, result, tt.expected)
			}
		})
	}
}

func TestIsValidTLSVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"TLS 1.0", "TLS1.0", true},
		{"TLS 1.1", "TLS1.1", true},
		{"TLS 1.2", "TLS1.2", true},
		{"TLS 1.3", "TLS1.3", true},
		{"Invalid", "TLS2.0", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidTLSVersion(tt.version); got != tt.want {
				t.Errorf("isValidTLSVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestConfig_LoadFromEnv(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"OAUTH2_TEST_ENABLED",
		"OAUTH2_TEST_PROVIDER",
		"OAUTH2_TEST_CLIENT_ID",
		"OAUTH2_TEST_CLIENT_SECRET",
		"OAUTH2_TEST_REDIRECT_URL",
		"OAUTH2_TEST_SCOPES",
		"OAUTH2_TEST_ISSUER_URL",
		"OAUTH2_TEST_USE_DISCOVERY",
		"OAUTH2_TEST_HTTP_TIMEOUT",
		"OAUTH2_TEST_JWT_SIGNING_METHOD",
		"OAUTH2_TEST_JWT_CLOCK_SKEW",
		"OAUTH2_TEST_CACHE_ENABLED",
		"OAUTH2_TEST_CACHE_TYPE",
		"OAUTH2_TEST_CACHE_TTL",
		"OAUTH2_TEST_TLS_ENABLED",
		"OAUTH2_TEST_TLS_MIN_VERSION",
	}
	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
	}
	defer func() {
		// Restore original environment
		for env, val := range originalEnv {
			if val == "" {
				os.Unsetenv(env)
			} else {
				os.Setenv(env, val)
			}
		}
	}()

	t.Run("loads basic config from environment", func(t *testing.T) {
		os.Setenv("OAUTH2_TEST_ENABLED", "true")
		os.Setenv("OAUTH2_TEST_PROVIDER", "keycloak")
		os.Setenv("OAUTH2_TEST_CLIENT_ID", "test-client")
		os.Setenv("OAUTH2_TEST_CLIENT_SECRET", "test-secret")
		os.Setenv("OAUTH2_TEST_REDIRECT_URL", "https://example.com/callback")
		os.Setenv("OAUTH2_TEST_SCOPES", "openid, profile, email")
		os.Setenv("OAUTH2_TEST_ISSUER_URL", "https://auth.example.com")
		os.Setenv("OAUTH2_TEST_USE_DISCOVERY", "yes")
		os.Setenv("OAUTH2_TEST_HTTP_TIMEOUT", "45s")

		config := &Config{}
		config.LoadFromEnv("OAUTH2_TEST_")

		if !config.Enabled {
			t.Error("Expected Enabled to be true")
		}
		if config.Provider != "keycloak" {
			t.Errorf("Expected Provider to be 'keycloak', got %q", config.Provider)
		}
		if config.ClientID != "test-client" {
			t.Errorf("Expected ClientID to be 'test-client', got %q", config.ClientID)
		}
		if config.ClientSecret != "test-secret" {
			t.Errorf("Expected ClientSecret to be 'test-secret', got %q", config.ClientSecret)
		}
		if config.RedirectURL != "https://example.com/callback" {
			t.Errorf("Expected RedirectURL to be 'https://example.com/callback', got %q", config.RedirectURL)
		}
		if len(config.Scopes) != 3 {
			t.Errorf("Expected 3 scopes, got %d", len(config.Scopes))
		}
		if config.IssuerURL != "https://auth.example.com" {
			t.Errorf("Expected IssuerURL to be 'https://auth.example.com', got %q", config.IssuerURL)
		}
		if !config.UseDiscovery {
			t.Error("Expected UseDiscovery to be true")
		}
		if config.HTTPTimeout != 45*time.Second {
			t.Errorf("Expected HTTPTimeout to be 45s, got %v", config.HTTPTimeout)
		}
	})

	t.Run("loads JWT config from environment", func(t *testing.T) {
		os.Setenv("OAUTH2_TEST_JWT_SIGNING_METHOD", "ES256")
		os.Setenv("OAUTH2_TEST_JWT_CLOCK_SKEW", "60s")

		config := &Config{}
		config.LoadFromEnv("OAUTH2_TEST_")

		if config.JWTConfig.SigningMethod != "ES256" {
			t.Errorf("Expected JWT signing method to be 'ES256', got %q", config.JWTConfig.SigningMethod)
		}
		if config.JWTConfig.ClockSkew != 60*time.Second {
			t.Errorf("Expected JWT clock skew to be 60s, got %v", config.JWTConfig.ClockSkew)
		}
	})

	t.Run("loads cache config from environment", func(t *testing.T) {
		os.Setenv("OAUTH2_TEST_CACHE_ENABLED", "1")
		os.Setenv("OAUTH2_TEST_CACHE_TYPE", "redis")
		os.Setenv("OAUTH2_TEST_CACHE_TTL", "10m")

		config := &Config{}
		config.LoadFromEnv("OAUTH2_TEST_")

		if !config.CacheConfig.Enabled {
			t.Error("Expected Cache to be enabled")
		}
		if config.CacheConfig.Type != "redis" {
			t.Errorf("Expected Cache type to be 'redis', got %q", config.CacheConfig.Type)
		}
		if config.CacheConfig.TTL != 10*time.Minute {
			t.Errorf("Expected Cache TTL to be 10m, got %v", config.CacheConfig.TTL)
		}
	})

	t.Run("loads TLS config from environment", func(t *testing.T) {
		os.Setenv("OAUTH2_TEST_TLS_ENABLED", "on")
		os.Setenv("OAUTH2_TEST_TLS_MIN_VERSION", "TLS1.3")

		config := &Config{}
		config.LoadFromEnv("OAUTH2_TEST_")

		if !config.TLSConfig.Enabled {
			t.Error("Expected TLS to be enabled")
		}
		if config.TLSConfig.MinVersion != "TLS1.3" {
			t.Errorf("Expected TLS min version to be 'TLS1.3', got %q", config.TLSConfig.MinVersion)
		}
	})

	t.Run("uses default prefix when empty", func(t *testing.T) {
		os.Setenv("OAUTH2_CLIENT_ID", "default-prefix-test")

		config := &Config{}
		config.LoadFromEnv("")

		if config.ClientID != "default-prefix-test" {
			t.Errorf("Expected ClientID to be 'default-prefix-test', got %q", config.ClientID)
		}

		os.Unsetenv("OAUTH2_CLIENT_ID")
	})

	t.Run("does not override existing values with empty env vars", func(t *testing.T) {
		config := &Config{
			ClientID:     "existing-client",
			ClientSecret: "existing-secret",
		}

		// Don't set any environment variables
		config.LoadFromEnv("OAUTH2_NONEXISTENT_")

		if config.ClientID != "existing-client" {
			t.Errorf("Expected ClientID to remain 'existing-client', got %q", config.ClientID)
		}
		if config.ClientSecret != "existing-secret" {
			t.Errorf("Expected ClientSecret to remain 'existing-secret', got %q", config.ClientSecret)
		}
	})
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"true", "true", true},
		{"TRUE", "TRUE", true},
		{"1", "1", true},
		{"yes", "yes", true},
		{"YES", "YES", true},
		{"on", "on", true},
		{"ON", "ON", true},
		{"false", "false", false},
		{"0", "0", false},
		{"no", "no", false},
		{"off", "off", false},
		{"empty", "", false},
		{"invalid", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseBool(tt.input); got != tt.want {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
