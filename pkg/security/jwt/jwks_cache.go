// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jwt

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// JWK represents a JSON Web Key.
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
	Crv string `json:"crv,omitempty"`
}

// JWKSet represents a set of JSON Web Keys.
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// JWKSCache provides caching for JWKS with automatic refresh.
type JWKSCache struct {
	url            string
	client         *http.Client
	keys           map[string]interface{} // kid -> public key
	mu             sync.RWMutex
	refreshTTL     time.Duration
	lastRefresh    time.Time
	autoRefresh    bool
	stopCh         chan struct{}
	refreshErrCh   chan error
	minRefreshWait time.Duration
}

// JWKSCacheConfig holds the configuration for JWKS cache.
type JWKSCacheConfig struct {
	// URL is the JWKS endpoint URL.
	URL string

	// RefreshTTL is the duration between automatic refreshes.
	RefreshTTL time.Duration

	// AutoRefresh enables automatic background refresh.
	AutoRefresh bool

	// HTTPClient is the HTTP client to use for fetching JWKS.
	HTTPClient *http.Client

	// MinRefreshWait is the minimum time to wait between refreshes.
	MinRefreshWait time.Duration
}

// SetDefaults sets default values for JWKS cache configuration.
func (c *JWKSCacheConfig) SetDefaults() {
	if c.RefreshTTL <= 0 {
		c.RefreshTTL = 1 * time.Hour
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	if c.MinRefreshWait <= 0 {
		c.MinRefreshWait = 1 * time.Minute
	}
}

// Validate validates the JWKS cache configuration.
func (c *JWKSCacheConfig) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("jwks: URL cannot be empty")
	}
	return nil
}

// NewJWKSCache creates a new JWKS cache.
func NewJWKSCache(config *JWKSCacheConfig) (*JWKSCache, error) {
	if config == nil {
		return nil, fmt.Errorf("jwks: config cannot be nil")
	}

	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("jwks: invalid config: %w", err)
	}

	cache := &JWKSCache{
		url:            config.URL,
		client:         config.HTTPClient,
		keys:           make(map[string]interface{}),
		refreshTTL:     config.RefreshTTL,
		autoRefresh:    config.AutoRefresh,
		stopCh:         make(chan struct{}),
		refreshErrCh:   make(chan error, 1),
		minRefreshWait: config.MinRefreshWait,
	}

	// Initial fetch
	if err := cache.Refresh(context.Background()); err != nil {
		return nil, fmt.Errorf("jwks: failed to fetch initial keys: %w", err)
	}

	// Start auto-refresh if enabled
	if cache.autoRefresh {
		go cache.startAutoRefresh()
	}

	return cache, nil
}

// GetKey retrieves a public key by key ID.
func (c *JWKSCache) GetKey(kid string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key, exists := c.keys[kid]
	if !exists {
		return nil, fmt.Errorf("jwks: key with id %q not found", kid)
	}

	return key, nil
}

// Refresh fetches and updates the keys from the JWKS endpoint.
func (c *JWKSCache) Refresh(ctx context.Context) error {
	c.mu.Lock()
	// Check if we need to wait before refreshing
	if time.Since(c.lastRefresh) < c.minRefreshWait {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return fmt.Errorf("jwks: failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("jwks: failed to fetch keys: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks: unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("jwks: failed to read response: %w", err)
	}

	var jwks JWKSet
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("jwks: failed to parse response: %w", err)
	}

	// Parse and update keys
	newKeys := make(map[string]interface{})
	for _, jwk := range jwks.Keys {
		key, err := parseJWK(&jwk)
		if err != nil {
			// Log warning but continue with other keys
			continue
		}
		newKeys[jwk.Kid] = key
	}

	c.mu.Lock()
	c.keys = newKeys
	c.lastRefresh = time.Now()
	c.mu.Unlock()

	return nil
}

// startAutoRefresh starts the automatic refresh goroutine.
func (c *JWKSCache) startAutoRefresh() {
	ticker := time.NewTicker(c.refreshTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := c.Refresh(ctx); err != nil {
				// Send error to error channel (non-blocking)
				select {
				case c.refreshErrCh <- err:
				default:
				}
			}
			cancel()
		case <-c.stopCh:
			return
		}
	}
}

// Stop stops the automatic refresh goroutine.
func (c *JWKSCache) Stop() {
	if c.autoRefresh {
		close(c.stopCh)
	}
}

// RefreshErrors returns a channel that receives refresh errors.
func (c *JWKSCache) RefreshErrors() <-chan error {
	return c.refreshErrCh
}

// parseJWK parses a JWK and returns the corresponding public key.
func parseJWK(jwk *JWK) (interface{}, error) {
	switch jwk.Kty {
	case "RSA":
		return parseRSAKey(jwk)
	case "EC":
		return parseECKey(jwk)
	default:
		return nil, fmt.Errorf("jwks: unsupported key type: %s", jwk.Kty)
	}
}

// parseRSAKey parses an RSA public key from a JWK.
func parseRSAKey(jwk *JWK) (*rsa.PublicKey, error) {
	// Decode modulus
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("jwks: failed to decode modulus: %w", err)
	}

	// Decode exponent
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("jwks: failed to decode exponent: %w", err)
	}

	// Construct public key
	pubKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}

	return pubKey, nil
}

// parseECKey parses an EC public key from a JWK.
func parseECKey(jwk *JWK) (interface{}, error) {
	// Decode X coordinate
	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("jwks: failed to decode X coordinate: %w", err)
	}

	// Decode Y coordinate
	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("jwks: failed to decode Y coordinate: %w", err)
	}

	// Determine curve
	var curve string
	switch jwk.Crv {
	case "P-256":
		curve = "P-256"
	case "P-384":
		curve = "P-384"
	case "P-521":
		curve = "P-521"
	default:
		return nil, fmt.Errorf("jwks: unsupported curve: %s", jwk.Crv)
	}

	// Create public key using x509 parsing
	pubKeyBytes := make([]byte, 1+len(xBytes)+len(yBytes))
	pubKeyBytes[0] = 4 // uncompressed point
	copy(pubKeyBytes[1:], xBytes)
	copy(pubKeyBytes[1+len(xBytes):], yBytes)

	// Parse based on curve
	switch curve {
	case "P-256":
		return x509.ParsePKIXPublicKey(pubKeyBytes)
	case "P-384":
		return x509.ParsePKIXPublicKey(pubKeyBytes)
	case "P-521":
		return x509.ParsePKIXPublicKey(pubKeyBytes)
	default:
		return nil, fmt.Errorf("jwks: unsupported curve: %s", curve)
	}
}

// KeyCount returns the number of cached keys.
func (c *JWKSCache) KeyCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.keys)
}

// LastRefreshTime returns the time of the last successful refresh.
func (c *JWKSCache) LastRefreshTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastRefresh
}

