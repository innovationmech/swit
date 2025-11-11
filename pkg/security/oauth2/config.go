// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package oauth2 provides OAuth2/OIDC authentication capabilities for the Swit framework.
// It includes client configuration, token exchange, and OIDC discovery support.
package oauth2

import (
	"fmt"
	"time"
)

// Config represents the OAuth2/OIDC client configuration.
type Config struct {
	// Enabled indicates whether OAuth2 authentication is enabled.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// Provider is the name of the OAuth2 provider (e.g., "keycloak", "auth0", "custom").
	Provider string `json:"provider" yaml:"provider" mapstructure:"provider"`

	// ClientID is the OAuth2 client identifier.
	ClientID string `json:"client_id" yaml:"client_id" mapstructure:"client_id"`

	// ClientSecret is the OAuth2 client secret.
	ClientSecret string `json:"client_secret" yaml:"client_secret" mapstructure:"client_secret"`

	// RedirectURL is the OAuth2 redirect URL for authorization code flow.
	RedirectURL string `json:"redirect_url" yaml:"redirect_url" mapstructure:"redirect_url"`

	// Scopes is the list of OAuth2 scopes to request.
	Scopes []string `json:"scopes" yaml:"scopes" mapstructure:"scopes"`

	// IssuerURL is the OIDC issuer URL (used for discovery).
	IssuerURL string `json:"issuer_url" yaml:"issuer_url" mapstructure:"issuer_url"`

	// AuthURL is the OAuth2 authorization endpoint URL.
	// If not set, it will be discovered via OIDC discovery.
	AuthURL string `json:"auth_url,omitempty" yaml:"auth_url,omitempty" mapstructure:"auth_url"`

	// TokenURL is the OAuth2 token endpoint URL.
	// If not set, it will be discovered via OIDC discovery.
	TokenURL string `json:"token_url,omitempty" yaml:"token_url,omitempty" mapstructure:"token_url"`

	// UserInfoURL is the OIDC user info endpoint URL.
	// If not set, it will be discovered via OIDC discovery.
	UserInfoURL string `json:"user_info_url,omitempty" yaml:"user_info_url,omitempty" mapstructure:"user_info_url"`

	// JWKSURL is the URL of the JSON Web Key Set for token verification.
	// If not set, it will be discovered via OIDC discovery.
	JWKSURL string `json:"jwks_url,omitempty" yaml:"jwks_url,omitempty" mapstructure:"jwks_url"`

	// UseDiscovery indicates whether to use OIDC discovery.
	// If true, endpoints will be discovered from IssuerURL.
	UseDiscovery bool `json:"use_discovery" yaml:"use_discovery" mapstructure:"use_discovery"`

	// HTTPTimeout is the HTTP client timeout for OAuth2 requests.
	HTTPTimeout time.Duration `json:"http_timeout" yaml:"http_timeout" mapstructure:"http_timeout"`

	// SkipIssuerVerification skips the issuer verification in OIDC.
	// This should only be used for testing purposes.
	SkipIssuerVerification bool `json:"skip_issuer_verification,omitempty" yaml:"skip_issuer_verification,omitempty" mapstructure:"skip_issuer_verification"`

	// SkipExpiryCheck skips the token expiry check.
	// This should only be used for testing purposes.
	SkipExpiryCheck bool `json:"skip_expiry_check,omitempty" yaml:"skip_expiry_check,omitempty" mapstructure:"skip_expiry_check"`
}

// Validate validates the OAuth2 configuration.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.ClientID == "" {
		return fmt.Errorf("oauth2: client_id is required")
	}

	if c.ClientSecret == "" {
		return fmt.Errorf("oauth2: client_secret is required")
	}

	if c.UseDiscovery {
		if c.IssuerURL == "" {
			return fmt.Errorf("oauth2: issuer_url is required when use_discovery is true")
		}
	} else {
		if c.AuthURL == "" {
			return fmt.Errorf("oauth2: auth_url is required when use_discovery is false")
		}
		if c.TokenURL == "" {
			return fmt.Errorf("oauth2: token_url is required when use_discovery is false")
		}
		if c.JWKSURL == "" {
			return fmt.Errorf("oauth2: jwks_url is required when use_discovery is false")
		}
	}

	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = 30 * time.Second
	}

	return nil
}

// SetDefaults sets default values for the configuration.
func (c *Config) SetDefaults() {
	if c.Provider == "" {
		c.Provider = "custom"
	}

	if len(c.Scopes) == 0 {
		c.Scopes = []string{"openid", "profile", "email"}
	}

	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = 30 * time.Second
	}

	if c.IssuerURL != "" && c.UseDiscovery {
		// Discovery will be used
		c.UseDiscovery = true
	}
}
