// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oauth2

import (
	"context"
	cryptotls "crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Client represents an OAuth2/OIDC client.
type Client struct {
	config         *Config
	oauth2Config   *oauth2.Config
	provider       *oidc.Provider
	verifier       *oidc.IDTokenVerifier
	httpClient     *http.Client
	providerConfig *ProviderConfig
	metadata       *ProviderMetadata
}

// NewClient creates a new OAuth2/OIDC client.
// It performs OIDC discovery if UseDiscovery is enabled in the configuration.
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("oauth2: config cannot be nil")
	}

	// Set defaults and validate
	cfg.SetDefaults()

	// Apply provider-specific defaults
	if err := cfg.ApplyProviderDefaults(); err != nil {
		return nil, fmt.Errorf("oauth2: failed to apply provider defaults: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("oauth2: invalid config: %w", err)
	}

	// Get provider configuration
	providerConfig, err := GetProviderConfig(cfg.Provider)
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to get provider config: %w", err)
	}

	// Create HTTP client with TLS and connection pooling
	httpClient, err := createHTTPClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to create HTTP client: %w", err)
	}

	client := &Client{
		config:         cfg,
		httpClient:     httpClient,
		providerConfig: providerConfig,
		metadata:       &ProviderMetadata{},
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

// createHTTPClient creates an HTTP client with TLS configuration and connection pooling.
func createHTTPClient(cfg *Config) (*http.Client, error) {
	// Create custom transport with connection pooling
	transport := &http.Transport{
		// Proxy settings - respect environment variables (HTTP_PROXY, HTTPS_PROXY, NO_PROXY)
		Proxy: http.ProxyFromEnvironment,

		// Connection pooling settings
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,

		// Timeout settings
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		// Dial settings
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,

		// Force HTTP/2
		ForceAttemptHTTP2: true,
	}

	// Configure TLS if enabled
	if cfg.TLSConfig.Enabled {
		tlsConfig, err := cfg.TLSConfig.ToGoTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
		transport.TLSClientConfig = tlsConfig
	} else {
		// Use default secure TLS settings
		transport.TLSClientConfig = &cryptotls.Config{
			MinVersion: cryptotls.VersionTLS12,
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   cfg.HTTPTimeout,
	}, nil
}

// Initialize initializes the client.
// This method can be called to reinitialize the client with a new context.
func (c *Client) Initialize(ctx context.Context) error {
	if c.config.UseDiscovery {
		return c.initWithDiscovery(ctx)
	}
	return c.initManual()
}

// initWithDiscovery initializes the client using OIDC discovery.
func (c *Client) initWithDiscovery(ctx context.Context) error {
	// Use custom HTTP client for OIDC discovery
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.httpClient)

	// Create OIDC provider
	provider, err := oidc.NewProvider(ctx, c.config.IssuerURL)
	if err != nil {
		return fmt.Errorf("failed to create OIDC provider: %w", err)
	}
	c.provider = provider

	// Extract provider metadata
	endpoint := provider.Endpoint()
	c.metadata.Issuer = c.config.IssuerURL
	c.metadata.AuthorizationEndpoint = endpoint.AuthURL
	c.metadata.TokenEndpoint = endpoint.TokenURL

	// Get JWKS URL from provider claims
	var claims struct {
		Issuer      string `json:"issuer"`
		JWKSURL     string `json:"jwks_uri"`
		UserInfoURL string `json:"userinfo_endpoint"`
	}
	if err := provider.Claims(&claims); err == nil {
		c.metadata.JWKSEndpoint = claims.JWKSURL
		c.metadata.UserInfoEndpoint = claims.UserInfoURL
	}

	// Create OAuth2 config
	c.oauth2Config = &oauth2.Config{
		ClientID:     c.config.ClientID,
		ClientSecret: c.config.ClientSecret,
		RedirectURL:  c.config.RedirectURL,
		Endpoint:     endpoint,
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

	// Populate metadata from manual configuration
	c.metadata.AuthorizationEndpoint = c.config.AuthURL
	c.metadata.TokenEndpoint = c.config.TokenURL
	c.metadata.UserInfoEndpoint = c.config.UserInfoURL
	c.metadata.JWKSEndpoint = c.config.JWKSURL
	c.metadata.Issuer = c.config.IssuerURL

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

// GetProviderConfig returns the provider configuration.
func (c *Client) GetProviderConfig() *ProviderConfig {
	return c.providerConfig
}

// GetMetadata returns the provider metadata.
func (c *Client) GetMetadata() *ProviderMetadata {
	return c.metadata
}

// GetHTTPClient returns the HTTP client used by the OAuth2 client.
func (c *Client) GetHTTPClient() *http.Client {
	return c.httpClient
}

// GetIssuerURL returns the issuer URL.
func (c *Client) GetIssuerURL() string {
	if c.metadata != nil {
		return c.metadata.Issuer
	}
	return c.config.IssuerURL
}

// GetJWKSURL returns the JWKS URL.
func (c *Client) GetJWKSURL() string {
	if c.metadata != nil && c.metadata.JWKSEndpoint != "" {
		return c.metadata.JWKSEndpoint
	}
	return c.config.JWKSURL
}

// IntrospectToken introspects an OAuth2 token to check its validity and retrieve metadata.
// This requires the OAuth2 server to support the token introspection endpoint (RFC 7662).
func (c *Client) IntrospectToken(ctx context.Context, token string) (*TokenIntrospection, error) {
	if token == "" {
		return nil, fmt.Errorf("oauth2: token cannot be empty")
	}

	// Construct introspection endpoint URL
	// Most providers use a standard /introspect endpoint
	introspectionURL := c.config.TokenURL
	if introspectionURL == "" {
		return nil, fmt.Errorf("oauth2: token URL not configured")
	}

	// Replace /token with /introspect for standard OAuth2 providers
	// This is a common pattern, but can be overridden by provider configuration
	if c.providerConfig != nil && c.providerConfig.IntrospectionEndpoint != "" {
		introspectionURL = c.providerConfig.IntrospectionEndpoint
	} else {
		// Try to construct introspection URL from token URL
		introspectionURL = c.metadata.TokenEndpoint
		if introspectionURL == "" {
			introspectionURL = c.config.TokenURL
		}
		// This is a heuristic - most providers have /introspect alongside /token
		introspectionURL = strings.Replace(introspectionURL, "/token", "/introspect", 1)
	}

	// Prepare the request
	data := url.Values{}
	data.Set("token", token)
	data.Set("client_id", c.config.ClientID)
	data.Set("client_secret", c.config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", introspectionURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to create introspection request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth2: introspection request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("oauth2: introspection failed with status %d: %s", resp.StatusCode, string(body))
	}

	var introspection TokenIntrospection
	if err := json.NewDecoder(resp.Body).Decode(&introspection); err != nil {
		return nil, fmt.Errorf("oauth2: failed to decode introspection response: %w", err)
	}

	return &introspection, nil
}

// RevokeToken revokes an OAuth2 token (access token or refresh token).
// This requires the OAuth2 server to support the token revocation endpoint (RFC 7009).
func (c *Client) RevokeToken(ctx context.Context, token string) error {
	if token == "" {
		return fmt.Errorf("oauth2: token cannot be empty")
	}

	// Construct revocation endpoint URL
	revocationURL := c.config.TokenURL
	if revocationURL == "" {
		return fmt.Errorf("oauth2: token URL not configured")
	}

	// Replace /token with /revoke for standard OAuth2 providers
	// This is a common pattern, but can be overridden by provider configuration
	if c.providerConfig != nil && c.providerConfig.RevocationEndpoint != "" {
		revocationURL = c.providerConfig.RevocationEndpoint
	} else {
		// Try to construct revocation URL from token URL
		revocationURL = c.metadata.TokenEndpoint
		if revocationURL == "" {
			revocationURL = c.config.TokenURL
		}
		// This is a heuristic - most providers have /revoke alongside /token
		revocationURL = strings.Replace(revocationURL, "/token", "/revoke", 1)
	}

	// Prepare the request
	data := url.Values{}
	data.Set("token", token)
	data.Set("client_id", c.config.ClientID)
	data.Set("client_secret", c.config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", revocationURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("oauth2: failed to create revocation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("oauth2: revocation request failed: %w", err)
	}
	defer resp.Body.Close()

	// RFC 7009 states that the authorization server responds with HTTP status code 200
	// if the token has been revoked successfully or if the client submitted an invalid token.
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("oauth2: revocation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Close closes the HTTP client and releases resources.
func (c *Client) Close() error {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
	return nil
}
