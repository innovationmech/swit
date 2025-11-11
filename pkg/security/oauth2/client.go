// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Client represents an OAuth2/OIDC client.
type Client struct {
	config       *Config
	oauth2Config *oauth2.Config
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	httpClient   *http.Client
}

// NewClient creates a new OAuth2/OIDC client.
// It performs OIDC discovery if UseDiscovery is enabled in the configuration.
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("oauth2: config cannot be nil")
	}

	// Set defaults and validate
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("oauth2: invalid config: %w", err)
	}

	client := &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
	}

	// Perform OIDC discovery if enabled
	if cfg.UseDiscovery {
		if err := client.initWithDiscovery(ctx); err != nil {
			return nil, fmt.Errorf("oauth2: discovery failed: %w", err)
		}
	} else {
		if err := client.initManual(); err != nil {
			return nil, fmt.Errorf("oauth2: manual initialization failed: %w", err)
		}
	}

	return client, nil
}

// initWithDiscovery initializes the client using OIDC discovery.
func (c *Client) initWithDiscovery(ctx context.Context) error {
	// Create OIDC provider
	provider, err := oidc.NewProvider(ctx, c.config.IssuerURL)
	if err != nil {
		return fmt.Errorf("failed to create OIDC provider: %w", err)
	}
	c.provider = provider

	// Create OAuth2 config
	c.oauth2Config = &oauth2.Config{
		ClientID:     c.config.ClientID,
		ClientSecret: c.config.ClientSecret,
		RedirectURL:  c.config.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       c.config.Scopes,
	}

	// Create ID token verifier
	verifierConfig := &oidc.Config{
		ClientID:                   c.config.ClientID,
		SkipIssuerCheck:            c.config.SkipIssuerVerification,
		SkipExpiryCheck:            c.config.SkipExpiryCheck,
		InsecureSkipSignatureCheck: false,
	}
	c.verifier = provider.Verifier(verifierConfig)

	return nil
}

// initManual initializes the client with manual endpoint configuration.
func (c *Client) initManual() error {
	// Create OAuth2 config with manual endpoints
	c.oauth2Config = &oauth2.Config{
		ClientID:     c.config.ClientID,
		ClientSecret: c.config.ClientSecret,
		RedirectURL:  c.config.RedirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  c.config.AuthURL,
			TokenURL: c.config.TokenURL,
		},
		Scopes: c.config.Scopes,
	}

	// Note: Without OIDC discovery, we cannot create a verifier.
	// Token verification will need to be done manually using JWT libraries.

	return nil
}

// GetConfig returns the OAuth2 configuration.
func (c *Client) GetConfig() *oauth2.Config {
	return c.oauth2Config
}

// GetProvider returns the OIDC provider.
// Returns nil if OIDC discovery was not used.
func (c *Client) GetProvider() *oidc.Provider {
	return c.provider
}

// GetVerifier returns the OIDC ID token verifier.
// Returns nil if OIDC discovery was not used.
func (c *Client) GetVerifier() *oidc.IDTokenVerifier {
	return c.verifier
}

// AuthCodeURL returns the URL for the authorization code flow.
func (c *Client) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	return c.oauth2Config.AuthCodeURL(state, opts...)
}

// Exchange exchanges an authorization code for a token.
func (c *Client) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	return c.oauth2Config.Exchange(ctx, code)
}

// TokenSource creates a token source that automatically refreshes tokens.
func (c *Client) TokenSource(ctx context.Context, token *oauth2.Token) oauth2.TokenSource {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	return c.oauth2Config.TokenSource(ctx, token)
}

// Client returns an HTTP client that automatically includes the access token in requests.
func (c *Client) HTTPClient(ctx context.Context, token *oauth2.Token) *http.Client {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	return c.oauth2Config.Client(ctx, token)
}

// VerifyIDToken verifies an ID token and returns the parsed claims.
// This method is only available if OIDC discovery was used.
func (c *Client) VerifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	if c.verifier == nil {
		return nil, fmt.Errorf("oauth2: ID token verification not available without OIDC discovery")
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to verify ID token: %w", err)
	}

	return idToken, nil
}

// RefreshToken refreshes an OAuth2 token.
func (c *Client) RefreshToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	if token.RefreshToken == "" {
		return nil, fmt.Errorf("oauth2: no refresh token available")
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)
	ts := c.oauth2Config.TokenSource(ctx, token)

	newToken, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to refresh token: %w", err)
	}

	return newToken, nil
}

// IsTokenExpired checks if a token is expired.
func IsTokenExpired(token *oauth2.Token) bool {
	if token == nil {
		return true
	}
	if token.Expiry.IsZero() {
		return false
	}
	return time.Now().After(token.Expiry)
}

// IsTokenExpiring checks if a token will expire within the given duration.
func IsTokenExpiring(token *oauth2.Token, within time.Duration) bool {
	if token == nil {
		return true
	}
	if token.Expiry.IsZero() {
		return false
	}
	return time.Now().Add(within).After(token.Expiry)
}
