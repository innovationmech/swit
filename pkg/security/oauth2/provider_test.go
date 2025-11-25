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
	"testing"
)

// TestGetProviderConfig tests provider configuration retrieval.
func TestGetProviderConfig(t *testing.T) {
	tests := []struct {
		name          string
		providerType  string
		wantErr       bool
		wantType      ProviderType
		wantName      string
		wantDiscovery bool
	}{
		{
			name:          "keycloak provider",
			providerType:  "keycloak",
			wantErr:       false,
			wantType:      ProviderKeycloak,
			wantName:      "Keycloak",
			wantDiscovery: true,
		},
		{
			name:          "auth0 provider",
			providerType:  "auth0",
			wantErr:       false,
			wantType:      ProviderAuth0,
			wantName:      "Auth0",
			wantDiscovery: true,
		},
		{
			name:          "google provider",
			providerType:  "google",
			wantErr:       false,
			wantType:      ProviderGoogle,
			wantName:      "Google",
			wantDiscovery: true,
		},
		{
			name:          "microsoft provider",
			providerType:  "microsoft",
			wantErr:       false,
			wantType:      ProviderMicrosoft,
			wantName:      "Microsoft/Azure AD",
			wantDiscovery: true,
		},
		{
			name:          "okta provider",
			providerType:  "okta",
			wantErr:       false,
			wantType:      ProviderOkta,
			wantName:      "Okta",
			wantDiscovery: true,
		},
		{
			name:          "custom provider",
			providerType:  "custom",
			wantErr:       false,
			wantType:      ProviderCustom,
			wantName:      "Custom",
			wantDiscovery: false,
		},
		{
			name:          "empty provider defaults to custom",
			providerType:  "",
			wantErr:       false,
			wantType:      ProviderCustom,
			wantName:      "Custom",
			wantDiscovery: false,
		},
		{
			name:         "unsupported provider",
			providerType: "unknown",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := GetProviderConfig(tt.providerType)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetProviderConfig() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetProviderConfig() unexpected error = %v", err)
				return
			}

			if config == nil {
				t.Error("GetProviderConfig() returned nil config")
				return
			}

			if config.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", config.Type, tt.wantType)
			}
			if config.Name != tt.wantName {
				t.Errorf("Name = %v, want %v", config.Name, tt.wantName)
			}
			if config.SupportsDiscovery != tt.wantDiscovery {
				t.Errorf("SupportsDiscovery = %v, want %v", config.SupportsDiscovery, tt.wantDiscovery)
			}

			// Verify default scopes
			if len(config.DefaultScopes) == 0 {
				t.Error("DefaultScopes is empty")
			}

			// Verify roles claim name
			if config.RolesClaimName == "" {
				t.Error("RolesClaimName is empty")
			}
		})
	}
}

// TestIsProviderSupported tests provider support checking.
func TestIsProviderSupported(t *testing.T) {
	tests := []struct {
		name         string
		providerType string
		want         bool
	}{
		{"keycloak", "keycloak", true},
		{"auth0", "auth0", true},
		{"google", "google", true},
		{"microsoft", "microsoft", true},
		{"okta", "okta", true},
		{"custom", "custom", true},
		{"empty", "", true},
		{"unsupported", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsProviderSupported(tt.providerType)
			if got != tt.want {
				t.Errorf("IsProviderSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetSupportedProviders tests getting list of supported providers.
func TestGetSupportedProviders(t *testing.T) {
	providers := GetSupportedProviders()

	if len(providers) == 0 {
		t.Error("GetSupportedProviders() returned empty list")
	}

	// Check that all expected providers are in the list
	expectedProviders := []ProviderType{
		ProviderKeycloak,
		ProviderAuth0,
		ProviderGoogle,
		ProviderMicrosoft,
		ProviderOkta,
		ProviderCustom,
	}

	for _, expected := range expectedProviders {
		found := false
		for _, provider := range providers {
			if provider == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected provider %v not found in supported providers", expected)
		}
	}
}

// TestApplyProviderDefaults tests applying provider-specific defaults.
func TestApplyProviderDefaults(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		wantErr       bool
		wantScopes    []string
		wantDiscovery bool
	}{
		{
			name: "keycloak with empty scopes",
			config: &Config{
				Provider:  "keycloak",
				IssuerURL: "https://keycloak.example.com/realms/test",
			},
			wantErr:       false,
			wantScopes:    []string{"openid", "profile", "email"},
			wantDiscovery: true,
		},
		{
			name: "auth0 with custom scopes",
			config: &Config{
				Provider:  "auth0",
				IssuerURL: "https://example.auth0.com/",
				Scopes:    []string{"read:users"},
			},
			wantErr:       false,
			wantScopes:    []string{"read:users", "openid"}, // openid should be added
			wantDiscovery: true,
		},
		{
			name: "custom provider without issuer URL",
			config: &Config{
				Provider: "custom",
			},
			wantErr:       false,
			wantScopes:    []string{"openid", "profile", "email"},
			wantDiscovery: false,
		},
		{
			name: "google with all required scopes",
			config: &Config{
				Provider:  "google",
				IssuerURL: "https://accounts.google.com",
				Scopes:    []string{"openid", "profile", "email"},
			},
			wantErr:       false,
			wantScopes:    []string{"openid", "profile", "email"},
			wantDiscovery: true,
		},
		{
			name: "unsupported provider",
			config: &Config{
				Provider: "unknown",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ApplyProviderDefaults()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ApplyProviderDefaults() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ApplyProviderDefaults() unexpected error = %v", err)
				return
			}

			// Check scopes
			if len(tt.config.Scopes) != len(tt.wantScopes) {
				t.Errorf("Scopes length = %v, want %v", len(tt.config.Scopes), len(tt.wantScopes))
			}

			for _, wantScope := range tt.wantScopes {
				found := false
				for _, scope := range tt.config.Scopes {
					if scope == wantScope {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected scope %v not found in config.Scopes", wantScope)
				}
			}

			// Check discovery
			if tt.config.UseDiscovery != tt.wantDiscovery {
				t.Errorf("UseDiscovery = %v, want %v", tt.config.UseDiscovery, tt.wantDiscovery)
			}
		})
	}
}

// TestEnsureRequiredScopes tests ensuring required scopes are included.
func TestEnsureRequiredScopes(t *testing.T) {
	tests := []struct {
		name           string
		scopes         []string
		requiredScopes []string
		want           []string
	}{
		{
			name:           "no required scopes",
			scopes:         []string{"profile", "email"},
			requiredScopes: []string{},
			want:           []string{"profile", "email"},
		},
		{
			name:           "all required scopes present",
			scopes:         []string{"openid", "profile", "email"},
			requiredScopes: []string{"openid"},
			want:           []string{"openid", "profile", "email"},
		},
		{
			name:           "missing required scope",
			scopes:         []string{"profile", "email"},
			requiredScopes: []string{"openid"},
			want:           []string{"profile", "email", "openid"},
		},
		{
			name:           "multiple missing required scopes",
			scopes:         []string{"email"},
			requiredScopes: []string{"openid", "profile"},
			want:           []string{"email", "openid", "profile"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureRequiredScopes(tt.scopes, tt.requiredScopes)

			if len(got) != len(tt.want) {
				t.Errorf("ensureRequiredScopes() length = %v, want %v", len(got), len(tt.want))
			}

			for _, wantScope := range tt.want {
				found := false
				for _, scope := range got {
					if scope == wantScope {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected scope %v not found in result", wantScope)
				}
			}
		})
	}
}

// TestGetRolesClaimName tests getting roles claim name for different providers.
func TestGetRolesClaimName(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     string
	}{
		{
			name:     "keycloak",
			provider: "keycloak",
			want:     "realm_access.roles",
		},
		{
			name:     "auth0",
			provider: "auth0",
			want:     "roles",
		},
		{
			name:     "google",
			provider: "google",
			want:     "groups",
		},
		{
			name:     "microsoft",
			provider: "microsoft",
			want:     "roles",
		},
		{
			name:     "okta",
			provider: "okta",
			want:     "groups",
		},
		{
			name:     "custom",
			provider: "custom",
			want:     "roles",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Provider: tt.provider,
			}

			got := config.GetRolesClaimName()
			if got != tt.want {
				t.Errorf("GetRolesClaimName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestProviderSpecificDefaults tests provider-specific default configurations.
func TestProviderSpecificDefaults(t *testing.T) {
	providers := []string{"keycloak", "auth0", "google", "microsoft", "okta"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			config, err := GetProviderConfig(provider)
			if err != nil {
				t.Fatalf("GetProviderConfig() error = %v", err)
			}

			// All non-custom providers should support discovery
			if !config.SupportsDiscovery {
				t.Error("Provider should support discovery")
			}

			// All providers should have default scopes including openid
			hasOpenID := false
			for _, scope := range config.DefaultScopes {
				if scope == "openid" {
					hasOpenID = true
					break
				}
			}
			if !hasOpenID {
				t.Error("Default scopes should include 'openid'")
			}

			// All providers should have required scopes
			if len(config.RequiredScopes) == 0 {
				t.Error("Provider should have required scopes")
			}

			// All providers should have a token endpoint auth method
			if config.TokenEndpointAuthMethod == "" {
				t.Error("Provider should have token endpoint auth method")
			}

			// All providers should have a roles claim name
			if config.RolesClaimName == "" {
				t.Error("Provider should have roles claim name")
			}
		})
	}
}
