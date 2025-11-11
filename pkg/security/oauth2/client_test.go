// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oauth2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// TestNewClient tests the NewClient function with various configurations.
func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "config cannot be nil",
		},
		{
			name: "missing client ID",
			config: &Config{
				Enabled:      true,
				ClientSecret: "secret",
				IssuerURL:    "https://example.com",
				UseDiscovery: false,
				AuthURL:      "https://example.com/auth",
				TokenURL:     "https://example.com/token",
				JWKSURL:      "https://example.com/jwks",
			},
			wantErr: true,
			errMsg:  "client_id is required",
		},
		{
			name: "missing client secret",
			config: &Config{
				Enabled:      true,
				ClientID:     "test-client",
				IssuerURL:    "https://example.com",
				UseDiscovery: false,
				AuthURL:      "https://example.com/auth",
				TokenURL:     "https://example.com/token",
				JWKSURL:      "https://example.com/jwks",
			},
			wantErr: true,
			errMsg:  "client_secret is required",
		},
		{
			name: "manual configuration without auth URL",
			config: &Config{
				Enabled:      true,
				ClientID:     "test-client",
				ClientSecret: "secret",
				UseDiscovery: false,
				TokenURL:     "https://example.com/token",
				JWKSURL:      "https://example.com/jwks",
			},
			wantErr: true,
			errMsg:  "auth_url is required",
		},
		{
			name: "manual configuration without token URL",
			config: &Config{
				Enabled:      true,
				ClientID:     "test-client",
				ClientSecret: "secret",
				UseDiscovery: false,
				AuthURL:      "https://example.com/auth",
				JWKSURL:      "https://example.com/jwks",
			},
			wantErr: true,
			errMsg:  "token_url is required",
		},
		{
			name: "valid manual configuration",
			config: &Config{
				Enabled:      true,
				Provider:     "custom",
				ClientID:     "test-client",
				ClientSecret: "secret",
				UseDiscovery: false,
				AuthURL:      "https://example.com/auth",
				TokenURL:     "https://example.com/token",
				JWKSURL:      "https://example.com/jwks",
				Scopes:       []string{"openid", "profile"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := NewClient(ctx, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewClient() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("NewClient() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("NewClient() unexpected error = %v", err)
				return
			}

			if client == nil {
				t.Error("NewClient() returned nil client")
				return
			}

			// Verify client fields
			if client.config == nil {
				t.Error("client.config is nil")
			}
			if client.oauth2Config == nil {
				t.Error("client.oauth2Config is nil")
			}
			if client.httpClient == nil {
				t.Error("client.httpClient is nil")
			}
			if client.providerConfig == nil {
				t.Error("client.providerConfig is nil")
			}
			if client.metadata == nil {
				t.Error("client.metadata is nil")
			}
		})
	}
}

// TestNewClientWithDiscovery tests client creation with OIDC discovery.
func TestNewClientWithDiscovery(t *testing.T) {
	// Create a mock OIDC discovery server
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"issuer":                                server.URL,
				"authorization_endpoint":                server.URL + "/auth",
				"token_endpoint":                        server.URL + "/token",
				"jwks_uri":                              server.URL + "/jwks",
				"userinfo_endpoint":                     server.URL + "/userinfo",
				"response_types_supported":              []string{"code", "token", "id_token"},
				"subject_types_supported":               []string{"public"},
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	config := &Config{
		Enabled:      true,
		Provider:     "custom",
		ClientID:     "test-client",
		ClientSecret: "secret",
		IssuerURL:    server.URL,
		UseDiscovery: true,
		Scopes:       []string{"openid", "profile"},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Verify discovery worked
	if client.provider == nil {
		t.Error("client.provider is nil")
	}
	if client.verifier == nil {
		t.Error("client.verifier is nil")
	}
	if client.oauth2Config == nil {
		t.Error("client.oauth2Config is nil")
	}

	// Verify endpoints were discovered
	if client.oauth2Config.Endpoint.AuthURL != server.URL+"/auth" {
		t.Errorf("AuthURL = %v, want %v", client.oauth2Config.Endpoint.AuthURL, server.URL+"/auth")
	}
	if client.oauth2Config.Endpoint.TokenURL != server.URL+"/token" {
		t.Errorf("TokenURL = %v, want %v", client.oauth2Config.Endpoint.TokenURL, server.URL+"/token")
	}

	// Verify metadata
	if client.metadata.JWKSEndpoint != server.URL+"/jwks" {
		t.Errorf("JWKSEndpoint = %v, want %v", client.metadata.JWKSEndpoint, server.URL+"/jwks")
	}
	if client.metadata.UserInfoEndpoint != server.URL+"/userinfo" {
		t.Errorf("UserInfoEndpoint = %v, want %v", client.metadata.UserInfoEndpoint, server.URL+"/userinfo")
	}
}

// TestCreateHTTPClient tests HTTP client creation with various TLS configurations.
func TestCreateHTTPClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "default configuration",
			config: &Config{
				HTTPTimeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "with TLS disabled",
			config: &Config{
				HTTPTimeout: 30 * time.Second,
				TLSConfig: TLSConfig{
					Enabled: false,
				},
			},
			wantErr: false,
		},
		{
			name: "with TLS enabled but insecure skip verify",
			config: &Config{
				HTTPTimeout: 30 * time.Second,
				TLSConfig: TLSConfig{
					Enabled:            true,
					InsecureSkipVerify: true,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := createHTTPClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("createHTTPClient() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("createHTTPClient() unexpected error = %v", err)
				return
			}

			if client == nil {
				t.Error("createHTTPClient() returned nil client")
				return
			}

			if client.Timeout != tt.config.HTTPTimeout {
				t.Errorf("client.Timeout = %v, want %v", client.Timeout, tt.config.HTTPTimeout)
			}

			transport, ok := client.Transport.(*http.Transport)
			if !ok {
				t.Error("client.Transport is not *http.Transport")
				return
			}

			// Verify connection pooling settings
			if transport.MaxIdleConns != 100 {
				t.Errorf("MaxIdleConns = %v, want 100", transport.MaxIdleConns)
			}
			if transport.MaxIdleConnsPerHost != 10 {
				t.Errorf("MaxIdleConnsPerHost = %v, want 10", transport.MaxIdleConnsPerHost)
			}
			if transport.MaxConnsPerHost != 100 {
				t.Errorf("MaxConnsPerHost = %v, want 100", transport.MaxConnsPerHost)
			}

			// Verify TLS config
			if transport.TLSClientConfig == nil {
				t.Error("TLSClientConfig is nil")
			}

			// Verify proxy is configured to respect environment
			if transport.Proxy == nil {
				t.Error("Proxy function is nil, should be http.ProxyFromEnvironment")
			}
		})
	}
}

// TestClientInitialize tests client initialization.
func TestClientInitialize(t *testing.T) {
	// Manual configuration
	config := &Config{
		Enabled:      true,
		Provider:     "custom",
		ClientID:     "test-client",
		ClientSecret: "secret",
		UseDiscovery: false,
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		JWKSURL:      "https://example.com/jwks",
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Reinitialize
	err = client.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize() error = %v", err)
	}
}

// TestClientAuthCodeURL tests authorization code URL generation.
func TestClientAuthCodeURL(t *testing.T) {
	config := &Config{
		Enabled:      true,
		Provider:     "custom",
		ClientID:     "test-client",
		ClientSecret: "secret",
		RedirectURL:  "https://example.com/callback",
		UseDiscovery: false,
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		JWKSURL:      "https://example.com/jwks",
		Scopes:       []string{"openid", "profile"},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	state := "random-state"
	authURL := client.AuthCodeURL(state)

	if authURL == "" {
		t.Error("AuthCodeURL returned empty string")
	}
	if !contains(authURL, "https://example.com/auth") {
		t.Errorf("AuthCodeURL = %v, want URL containing %v", authURL, "https://example.com/auth")
	}
	if !contains(authURL, "client_id=test-client") {
		t.Error("AuthCodeURL missing client_id parameter")
	}
	if !contains(authURL, "state="+state) {
		t.Error("AuthCodeURL missing state parameter")
	}
}

// TestClientExchange tests authorization code exchange.
func TestClientExchange(t *testing.T) {
	// Create a mock token server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":  "test-access-token",
				"token_type":    "Bearer",
				"expires_in":    3600,
				"refresh_token": "test-refresh-token",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	config := &Config{
		Enabled:      true,
		Provider:     "custom",
		ClientID:     "test-client",
		ClientSecret: "secret",
		UseDiscovery: false,
		AuthURL:      server.URL + "/auth",
		TokenURL:     server.URL + "/token",
		JWKSURL:      server.URL + "/jwks",
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	token, err := client.Exchange(ctx, "test-code")
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}

	if token.AccessToken != "test-access-token" {
		t.Errorf("AccessToken = %v, want %v", token.AccessToken, "test-access-token")
	}
	if token.RefreshToken != "test-refresh-token" {
		t.Errorf("RefreshToken = %v, want %v", token.RefreshToken, "test-refresh-token")
	}
	if token.TokenType != "Bearer" {
		t.Errorf("TokenType = %v, want %v", token.TokenType, "Bearer")
	}
}

// TestIsTokenExpired tests token expiration checking.
func TestIsTokenExpired(t *testing.T) {
	tests := []struct {
		name  string
		token *oauth2.Token
		want  bool
	}{
		{
			name:  "nil token",
			token: nil,
			want:  true,
		},
		{
			name: "token with zero expiry",
			token: &oauth2.Token{
				AccessToken: "test",
			},
			want: false,
		},
		{
			name: "expired token",
			token: &oauth2.Token{
				AccessToken: "test",
				Expiry:      time.Now().Add(-1 * time.Hour),
			},
			want: true,
		},
		{
			name: "valid token",
			token: &oauth2.Token{
				AccessToken: "test",
				Expiry:      time.Now().Add(1 * time.Hour),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTokenExpired(tt.token)
			if got != tt.want {
				t.Errorf("IsTokenExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsTokenExpiring tests token expiring soon checking.
func TestIsTokenExpiring(t *testing.T) {
	tests := []struct {
		name   string
		token  *oauth2.Token
		within time.Duration
		want   bool
	}{
		{
			name:   "nil token",
			token:  nil,
			within: 5 * time.Minute,
			want:   true,
		},
		{
			name: "token with zero expiry",
			token: &oauth2.Token{
				AccessToken: "test",
			},
			within: 5 * time.Minute,
			want:   false,
		},
		{
			name: "token expiring soon",
			token: &oauth2.Token{
				AccessToken: "test",
				Expiry:      time.Now().Add(3 * time.Minute),
			},
			within: 5 * time.Minute,
			want:   true,
		},
		{
			name: "token not expiring soon",
			token: &oauth2.Token{
				AccessToken: "test",
				Expiry:      time.Now().Add(10 * time.Minute),
			},
			within: 5 * time.Minute,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTokenExpiring(tt.token, tt.within)
			if got != tt.want {
				t.Errorf("IsTokenExpiring() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestClientGetters tests various getter methods.
func TestClientGetters(t *testing.T) {
	config := &Config{
		Enabled:      true,
		Provider:     "custom",
		ClientID:     "test-client",
		ClientSecret: "secret",
		IssuerURL:    "https://example.com",
		UseDiscovery: false,
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		JWKSURL:      "https://example.com/jwks",
		UserInfoURL:  "https://example.com/userinfo",
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Test GetConfig
	oauth2Config := client.GetConfig()
	if oauth2Config == nil {
		t.Error("GetConfig() returned nil")
	}

	// Test GetProvider (should be nil for manual config)
	provider := client.GetProvider()
	if provider != nil {
		t.Error("GetProvider() should return nil for manual configuration")
	}

	// Test GetVerifier (should be nil for manual config)
	verifier := client.GetVerifier()
	if verifier != nil {
		t.Error("GetVerifier() should return nil for manual configuration")
	}

	// Test GetProviderConfig
	providerConfig := client.GetProviderConfig()
	if providerConfig == nil {
		t.Error("GetProviderConfig() returned nil")
	} else if providerConfig.Type != ProviderCustom {
		t.Errorf("GetProviderConfig().Type = %v, want %v", providerConfig.Type, ProviderCustom)
	}

	// Test GetMetadata
	metadata := client.GetMetadata()
	if metadata == nil {
		t.Error("GetMetadata() returned nil")
	}

	// Test GetHTTPClient
	httpClient := client.GetHTTPClient()
	if httpClient == nil {
		t.Error("GetHTTPClient() returned nil")
	}

	// Test GetIssuerURL
	issuerURL := client.GetIssuerURL()
	if issuerURL != config.IssuerURL {
		t.Errorf("GetIssuerURL() = %v, want %v", issuerURL, config.IssuerURL)
	}

	// Test GetJWKSURL
	jwksURL := client.GetJWKSURL()
	if jwksURL != config.JWKSURL {
		t.Errorf("GetJWKSURL() = %v, want %v", jwksURL, config.JWKSURL)
	}
}

// TestClientClose tests client closing.
func TestClientClose(t *testing.T) {
	config := &Config{
		Enabled:      true,
		Provider:     "custom",
		ClientID:     "test-client",
		ClientSecret: "secret",
		UseDiscovery: false,
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		JWKSURL:      "https://example.com/jwks",
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// contains is a helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
