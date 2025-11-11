// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package oauth2 provides OAuth2/OIDC authentication capabilities for the Swit framework.
// It includes client configuration, token exchange, and OIDC discovery support.
package oauth2

import (
	cryptotls "crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strconv"
	"strings"
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

	// JWTConfig is the JWT token validation configuration.
	JWTConfig JWTConfig `json:"jwt" yaml:"jwt" mapstructure:"jwt"`

	// CacheConfig is the token cache configuration.
	CacheConfig TokenCacheConfig `json:"cache" yaml:"cache" mapstructure:"cache"`

	// TLSConfig is the TLS configuration for OAuth2 connections.
	TLSConfig TLSConfig `json:"tls" yaml:"tls" mapstructure:"tls"`
}

// JWTConfig holds JWT token validation configuration.
type JWTConfig struct {
	// SigningMethod is the expected JWT signing method (e.g., "RS256", "HS256").
	SigningMethod string `json:"signing_method" yaml:"signing_method" mapstructure:"signing_method"`

	// ClockSkew is the duration of time skew to tolerate when verifying time-based claims.
	ClockSkew time.Duration `json:"clock_skew" yaml:"clock_skew" mapstructure:"clock_skew"`

	// SkipClaimsValidation skips standard claims validation (iss, aud, exp, nbf, iat).
	// This should only be used for testing purposes.
	SkipClaimsValidation bool `json:"skip_claims_validation,omitempty" yaml:"skip_claims_validation,omitempty" mapstructure:"skip_claims_validation"`

	// RequiredClaims is a list of claims that must be present in the token.
	RequiredClaims []string `json:"required_claims,omitempty" yaml:"required_claims,omitempty" mapstructure:"required_claims"`

	// Audience is the expected audience claim value.
	Audience string `json:"audience,omitempty" yaml:"audience,omitempty" mapstructure:"audience"`

	// Issuer is the expected issuer claim value.
	Issuer string `json:"issuer,omitempty" yaml:"issuer,omitempty" mapstructure:"issuer"`
}

// TLSConfig holds TLS configuration for OAuth2 connections.
type TLSConfig struct {
	// Enabled indicates whether TLS is enabled.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// InsecureSkipVerify controls whether the client verifies the server's certificate chain.
	// This should only be used for testing purposes.
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty" yaml:"insecure_skip_verify,omitempty" mapstructure:"insecure_skip_verify"`

	// CertFile is the path to the client certificate file.
	CertFile string `json:"cert_file,omitempty" yaml:"cert_file,omitempty" mapstructure:"cert_file"`

	// KeyFile is the path to the client private key file.
	KeyFile string `json:"key_file,omitempty" yaml:"key_file,omitempty" mapstructure:"key_file"`

	// CAFile is the path to the CA certificate file.
	CAFile string `json:"ca_file,omitempty" yaml:"ca_file,omitempty" mapstructure:"ca_file"`

	// MinVersion is the minimum TLS version to use (e.g., "TLS1.2", "TLS1.3").
	MinVersion string `json:"min_version,omitempty" yaml:"min_version,omitempty" mapstructure:"min_version"`

	// MaxVersion is the maximum TLS version to use.
	MaxVersion string `json:"max_version,omitempty" yaml:"max_version,omitempty" mapstructure:"max_version"`
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

	// Validate sub-configurations
	if err := c.JWTConfig.Validate(); err != nil {
		return fmt.Errorf("oauth2: jwt config: %w", err)
	}

	if err := c.CacheConfig.Validate(); err != nil {
		return fmt.Errorf("oauth2: cache config: %w", err)
	}

	if err := c.TLSConfig.Validate(); err != nil {
		return fmt.Errorf("oauth2: tls config: %w", err)
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

	// Set defaults for sub-configurations
	c.JWTConfig.SetDefaults()
	c.CacheConfig.SetDefaults()
	c.TLSConfig.SetDefaults()
}

// Validate validates the JWT configuration.
func (j *JWTConfig) Validate() error {
	if j.SigningMethod == "" {
		return nil // Will use default
	}

	// Validate signing method is one of the supported methods
	validMethods := []string{"HS256", "HS384", "HS512", "RS256", "RS384", "RS512", "ES256", "ES384", "ES512", "PS256", "PS384", "PS512"}
	isValid := false
	for _, method := range validMethods {
		if j.SigningMethod == method {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("oauth2: invalid jwt signing_method %q", j.SigningMethod)
	}

	return nil
}

// SetDefaults sets default values for the JWT configuration.
func (j *JWTConfig) SetDefaults() {
	if j.SigningMethod == "" {
		j.SigningMethod = "RS256"
	}

	if j.ClockSkew <= 0 {
		j.ClockSkew = 30 * time.Second
	}
}

// Validate validates the TLS configuration.
func (tls *TLSConfig) Validate() error {
	if !tls.Enabled {
		return nil
	}

	// If cert file is provided, key file must also be provided
	if tls.CertFile != "" && tls.KeyFile == "" {
		return fmt.Errorf("oauth2: tls key_file is required when cert_file is provided")
	}

	if tls.KeyFile != "" && tls.CertFile == "" {
		return fmt.Errorf("oauth2: tls cert_file is required when key_file is provided")
	}

	// Validate certificate files exist
	if tls.CertFile != "" {
		if _, err := os.Stat(tls.CertFile); err != nil {
			return fmt.Errorf("oauth2: tls cert_file not found: %w", err)
		}
	}

	if tls.KeyFile != "" {
		if _, err := os.Stat(tls.KeyFile); err != nil {
			return fmt.Errorf("oauth2: tls key_file not found: %w", err)
		}
	}

	if tls.CAFile != "" {
		if _, err := os.Stat(tls.CAFile); err != nil {
			return fmt.Errorf("oauth2: tls ca_file not found: %w", err)
		}
	}

	// Validate TLS versions
	if tls.MinVersion != "" {
		if !isValidTLSVersion(tls.MinVersion) {
			return fmt.Errorf("oauth2: invalid tls min_version %q", tls.MinVersion)
		}
	}

	if tls.MaxVersion != "" {
		if !isValidTLSVersion(tls.MaxVersion) {
			return fmt.Errorf("oauth2: invalid tls max_version %q", tls.MaxVersion)
		}
	}

	return nil
}

// SetDefaults sets default values for the TLS configuration.
func (tls *TLSConfig) SetDefaults() {
	if tls.Enabled {
		if tls.MinVersion == "" {
			tls.MinVersion = "TLS1.2"
		}
	}
}

// ToGoTLSConfig converts the TLSConfig to a *cryptotls.Config.
func (tc *TLSConfig) ToGoTLSConfig() (*cryptotls.Config, error) {
	if !tc.Enabled {
		return nil, nil
	}

	config := &cryptotls.Config{
		InsecureSkipVerify: tc.InsecureSkipVerify,
	}

	// Load client certificate if provided
	if tc.CertFile != "" && tc.KeyFile != "" {
		cert, err := cryptotls.LoadX509KeyPair(tc.CertFile, tc.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("oauth2: failed to load tls certificate: %w", err)
		}
		config.Certificates = []cryptotls.Certificate{cert}
	}

	// Load CA certificate if provided
	if tc.CAFile != "" {
		caCert, err := os.ReadFile(tc.CAFile)
		if err != nil {
			return nil, fmt.Errorf("oauth2: failed to read ca_file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("oauth2: failed to parse ca_file")
		}
		config.RootCAs = caCertPool
	}

	// Set TLS version
	if tc.MinVersion != "" {
		config.MinVersion = parseTLSVersion(tc.MinVersion)
	}

	if tc.MaxVersion != "" {
		config.MaxVersion = parseTLSVersion(tc.MaxVersion)
	}

	return config, nil
}

// isValidTLSVersion checks if the TLS version string is valid.
func isValidTLSVersion(version string) bool {
	validVersions := []string{"TLS1.0", "TLS1.1", "TLS1.2", "TLS1.3"}
	for _, v := range validVersions {
		if v == version {
			return true
		}
	}
	return false
}

// parseTLSVersion converts a TLS version string to a uint16.
func parseTLSVersion(version string) uint16 {
	switch version {
	case "TLS1.0":
		return cryptotls.VersionTLS10
	case "TLS1.1":
		return cryptotls.VersionTLS11
	case "TLS1.2":
		return cryptotls.VersionTLS12
	case "TLS1.3":
		return cryptotls.VersionTLS13
	default:
		return cryptotls.VersionTLS12
	}
}

// LoadFromEnv loads configuration from environment variables and overrides existing values.
// Environment variable names are constructed by prefixing the field name with the given prefix.
// For example, with prefix "OAUTH2_", the ClientID field maps to OAUTH2_CLIENT_ID.
func (c *Config) LoadFromEnv(prefix string) {
	if prefix == "" {
		prefix = "OAUTH2_"
	}

	// Main OAuth2 config
	if val := os.Getenv(prefix + "ENABLED"); val != "" {
		c.Enabled = parseBool(val)
	}
	if val := os.Getenv(prefix + "PROVIDER"); val != "" {
		c.Provider = val
	}
	if val := os.Getenv(prefix + "CLIENT_ID"); val != "" {
		c.ClientID = val
	}
	if val := os.Getenv(prefix + "CLIENT_SECRET"); val != "" {
		c.ClientSecret = val
	}
	if val := os.Getenv(prefix + "REDIRECT_URL"); val != "" {
		c.RedirectURL = val
	}
	if val := os.Getenv(prefix + "SCOPES"); val != "" {
		c.Scopes = strings.Split(val, ",")
		for i := range c.Scopes {
			c.Scopes[i] = strings.TrimSpace(c.Scopes[i])
		}
	}
	if val := os.Getenv(prefix + "ISSUER_URL"); val != "" {
		c.IssuerURL = val
	}
	if val := os.Getenv(prefix + "AUTH_URL"); val != "" {
		c.AuthURL = val
	}
	if val := os.Getenv(prefix + "TOKEN_URL"); val != "" {
		c.TokenURL = val
	}
	if val := os.Getenv(prefix + "USER_INFO_URL"); val != "" {
		c.UserInfoURL = val
	}
	if val := os.Getenv(prefix + "JWKS_URL"); val != "" {
		c.JWKSURL = val
	}
	if val := os.Getenv(prefix + "USE_DISCOVERY"); val != "" {
		c.UseDiscovery = parseBool(val)
	}
	if val := os.Getenv(prefix + "HTTP_TIMEOUT"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			c.HTTPTimeout = duration
		}
	}
	if val := os.Getenv(prefix + "SKIP_ISSUER_VERIFICATION"); val != "" {
		c.SkipIssuerVerification = parseBool(val)
	}
	if val := os.Getenv(prefix + "SKIP_EXPIRY_CHECK"); val != "" {
		c.SkipExpiryCheck = parseBool(val)
	}

	// JWT config
	jwtPrefix := prefix + "JWT_"
	if val := os.Getenv(jwtPrefix + "SIGNING_METHOD"); val != "" {
		c.JWTConfig.SigningMethod = val
	}
	if val := os.Getenv(jwtPrefix + "CLOCK_SKEW"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			c.JWTConfig.ClockSkew = duration
		}
	}
	if val := os.Getenv(jwtPrefix + "SKIP_CLAIMS_VALIDATION"); val != "" {
		c.JWTConfig.SkipClaimsValidation = parseBool(val)
	}
	if val := os.Getenv(jwtPrefix + "REQUIRED_CLAIMS"); val != "" {
		c.JWTConfig.RequiredClaims = strings.Split(val, ",")
		for i := range c.JWTConfig.RequiredClaims {
			c.JWTConfig.RequiredClaims[i] = strings.TrimSpace(c.JWTConfig.RequiredClaims[i])
		}
	}
	if val := os.Getenv(jwtPrefix + "AUDIENCE"); val != "" {
		c.JWTConfig.Audience = val
	}
	if val := os.Getenv(jwtPrefix + "ISSUER"); val != "" {
		c.JWTConfig.Issuer = val
	}

	// Cache config
	cachePrefix := prefix + "CACHE_"
	if val := os.Getenv(cachePrefix + "ENABLED"); val != "" {
		c.CacheConfig.Enabled = parseBool(val)
	}
	if val := os.Getenv(cachePrefix + "TYPE"); val != "" {
		c.CacheConfig.Type = val
	}
	if val := os.Getenv(cachePrefix + "TTL"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			c.CacheConfig.TTL = duration
		}
	}

	// TLS config
	tlsPrefix := prefix + "TLS_"
	if val := os.Getenv(tlsPrefix + "ENABLED"); val != "" {
		c.TLSConfig.Enabled = parseBool(val)
	}
	if val := os.Getenv(tlsPrefix + "INSECURE_SKIP_VERIFY"); val != "" {
		c.TLSConfig.InsecureSkipVerify = parseBool(val)
	}
	if val := os.Getenv(tlsPrefix + "CERT_FILE"); val != "" {
		c.TLSConfig.CertFile = val
	}
	if val := os.Getenv(tlsPrefix + "KEY_FILE"); val != "" {
		c.TLSConfig.KeyFile = val
	}
	if val := os.Getenv(tlsPrefix + "CA_FILE"); val != "" {
		c.TLSConfig.CAFile = val
	}
	if val := os.Getenv(tlsPrefix + "MIN_VERSION"); val != "" {
		c.TLSConfig.MinVersion = val
	}
	if val := os.Getenv(tlsPrefix + "MAX_VERSION"); val != "" {
		c.TLSConfig.MaxVersion = val
	}
}

// parseBool parses a string to a boolean value.
// Accepts: "true", "1", "yes", "on" (case-insensitive) as true.
// Everything else is false.
func parseBool(val string) bool {
	val = strings.ToLower(strings.TrimSpace(val))
	switch val {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

// parseIntWithDefault parses a string to an int with a default value.
func parseIntWithDefault(val string, defaultVal int) int {
	if i, err := strconv.Atoi(val); err == nil {
		return i
	}
	return defaultVal
}
