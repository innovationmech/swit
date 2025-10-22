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

// Package security provides authentication and authorization functionality for Saga operations.
package security

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// AuthType represents the type of authentication
type AuthType string

const (
	// AuthTypeNone represents no authentication
	AuthTypeNone AuthType = "none"
	// AuthTypeJWT represents JWT token authentication
	AuthTypeJWT AuthType = "jwt"
	// AuthTypeAPIKey represents API key authentication
	AuthTypeAPIKey AuthType = "apikey"
)

// Common errors
var (
	ErrInvalidAuthType      = errors.New("invalid authentication type")
	ErrMissingCredentials   = errors.New("missing authentication credentials")
	ErrInvalidToken         = errors.New("invalid authentication token")
	ErrExpiredToken         = errors.New("authentication token has expired")
	ErrInvalidAPIKey        = errors.New("invalid API key")
	ErrInsufficientScope    = errors.New("insufficient scope for operation")
	ErrAuthenticationFailed = errors.New("authentication failed")
)

// AuthCredentials represents authentication credentials for Saga operations
type AuthCredentials struct {
	// Type of authentication
	Type AuthType

	// JWT token (for JWT authentication)
	Token string

	// API key (for API key authentication)
	APIKey string

	// User ID extracted from credentials
	UserID string

	// Scopes/permissions associated with the credentials
	Scopes []string

	// Expiration time for the credentials
	ExpiresAt *time.Time

	// Metadata associated with the credentials
	Metadata map[string]string
}

// Validate validates the authentication credentials
func (c *AuthCredentials) Validate() error {
	switch c.Type {
	case AuthTypeNone:
		// No validation needed
		return nil
	case AuthTypeJWT:
		if c.Token == "" {
			return fmt.Errorf("%w: token is required for JWT authentication", ErrMissingCredentials)
		}
	case AuthTypeAPIKey:
		if c.APIKey == "" {
			return fmt.Errorf("%w: api_key is required for API key authentication", ErrMissingCredentials)
		}
	default:
		return fmt.Errorf("%w: %s", ErrInvalidAuthType, c.Type)
	}

	return nil
}

// IsExpired checks if the credentials have expired
func (c *AuthCredentials) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*c.ExpiresAt)
}

// HasScope checks if the credentials have the specified scope
func (c *AuthCredentials) HasScope(scope string) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// AuthContext represents the authentication context for a Saga operation
type AuthContext struct {
	// Credentials used for authentication
	Credentials *AuthCredentials

	// Timestamp when the context was created
	Timestamp time.Time

	// Saga ID for tracing
	SagaID string

	// Correlation ID for distributed tracing
	CorrelationID string

	// Source of the authentication (e.g., "coordinator", "step_executor")
	Source string
}

// NewAuthContext creates a new authentication context
func NewAuthContext(credentials *AuthCredentials, sagaID string) *AuthContext {
	return &AuthContext{
		Credentials: credentials,
		Timestamp:   time.Now(),
		SagaID:      sagaID,
		Source:      "saga",
	}
}

// AuthProvider defines the interface for authentication providers
type AuthProvider interface {
	// Name returns the provider name
	Name() string

	// Authenticate authenticates the given credentials
	Authenticate(ctx context.Context, credentials *AuthCredentials) (*AuthContext, error)

	// ValidateToken validates a token and returns the associated credentials
	ValidateToken(ctx context.Context, token string) (*AuthCredentials, error)

	// IsHealthy checks if the authentication provider is healthy
	IsHealthy(ctx context.Context) bool
}

// JWTAuthProvider implements JWT-based authentication for Saga operations
type JWTAuthProvider struct {
	secret       []byte
	issuer       string
	audience     string
	cache        *AuthCache
	cacheEnabled bool
	mu           sync.RWMutex
}

// JWTAuthProviderConfig configures the JWT authentication provider
type JWTAuthProviderConfig struct {
	Secret       string
	Issuer       string
	Audience     string
	CacheTTL     time.Duration
	CacheEnabled bool
}

// NewJWTAuthProvider creates a new JWT authentication provider
func NewJWTAuthProvider(config *JWTAuthProviderConfig) (*JWTAuthProvider, error) {
	if config.Secret == "" {
		return nil, fmt.Errorf("secret is required for JWT authentication provider")
	}

	provider := &JWTAuthProvider{
		secret:       []byte(config.Secret),
		issuer:       config.Issuer,
		audience:     config.Audience,
		cacheEnabled: config.CacheEnabled,
	}

	if config.CacheEnabled && config.CacheTTL > 0 {
		provider.cache = NewAuthCache(config.CacheTTL)
	}

	return provider, nil
}

// Name implements AuthProvider interface
func (p *JWTAuthProvider) Name() string {
	return "jwt"
}

// Authenticate implements AuthProvider interface
func (p *JWTAuthProvider) Authenticate(ctx context.Context, credentials *AuthCredentials) (*AuthContext, error) {
	if credentials.Type != AuthTypeJWT {
		return nil, fmt.Errorf("%w: expected JWT, got %s", ErrInvalidAuthType, credentials.Type)
	}

	if credentials.Token == "" {
		return nil, fmt.Errorf("%w: token is required", ErrMissingCredentials)
	}

	// Check cache first
	if p.cache != nil {
		if cached, found := p.cache.Get(credentials.Token); found {
			return cached, nil
		}
	}

	// Validate JWT token
	claims, err := p.validateJWTToken(credentials.Token)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthenticationFailed, err)
	}

	// Extract user ID and scopes
	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: missing user_id in claims", ErrInvalidToken)
	}

	var scopes []string
	if scopesClaim, ok := claims["scopes"].([]interface{}); ok {
		for _, scope := range scopesClaim {
			if scopeStr, ok := scope.(string); ok {
				scopes = append(scopes, scopeStr)
			}
		}
	}

	// Extract expiration
	var expiresAt *time.Time
	if exp, ok := claims["exp"].(float64); ok {
		expTime := time.Unix(int64(exp), 0)
		expiresAt = &expTime
	}

	// Create credentials with extracted information
	authCreds := &AuthCredentials{
		Type:      AuthTypeJWT,
		Token:     credentials.Token,
		UserID:    userID,
		Scopes:    scopes,
		ExpiresAt: expiresAt,
		Metadata:  make(map[string]string),
	}

	// Extract additional claims as metadata
	for key, value := range claims {
		if key != "user_id" && key != "scopes" && key != "exp" && key != "iat" && key != "nbf" && key != "jti" && key != "iss" && key != "aud" {
			if strVal, ok := value.(string); ok {
				authCreds.Metadata[key] = strVal
			}
		}
	}

	// Create auth context
	authCtx := NewAuthContext(authCreds, generateSagaAuthID())

	// Cache the result
	if p.cache != nil {
		p.cache.Set(credentials.Token, authCtx)
	}

	return authCtx, nil
}

// ValidateToken implements AuthProvider interface
func (p *JWTAuthProvider) ValidateToken(ctx context.Context, token string) (*AuthCredentials, error) {
	authCtx, err := p.Authenticate(ctx, &AuthCredentials{
		Type:  AuthTypeJWT,
		Token: token,
	})
	if err != nil {
		return nil, err
	}
	return authCtx.Credentials, nil
}

// IsHealthy implements AuthProvider interface
func (p *JWTAuthProvider) IsHealthy(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.secret) > 0
}

// validateJWTToken validates a JWT token and returns its claims
func (p *JWTAuthProvider) validateJWTToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Validate issuer if specified
		if p.issuer != "" {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if issuer, ok := claims["iss"].(string); ok && issuer != p.issuer {
					return nil, fmt.Errorf("invalid issuer: %s", issuer)
				}
			}
		}

		// Validate audience if specified
		if p.audience != "" {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if audience, ok := claims["aud"].(string); ok && audience != p.audience {
					return nil, fmt.Errorf("invalid audience: %s", audience)
				}
			}
		}

		return p.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// APIKeyAuthProvider implements API key-based authentication for Saga operations
type APIKeyAuthProvider struct {
	validAPIKeys map[string]*AuthCredentials
	cache        *AuthCache
	cacheEnabled bool
	mu           sync.RWMutex
}

// APIKeyAuthProviderConfig configures the API key authentication provider
type APIKeyAuthProviderConfig struct {
	APIKeys      map[string]string // key -> user_id mapping
	CacheTTL     time.Duration
	CacheEnabled bool
}

// NewAPIKeyAuthProvider creates a new API key authentication provider
func NewAPIKeyAuthProvider(config *APIKeyAuthProviderConfig) (*APIKeyAuthProvider, error) {
	if len(config.APIKeys) == 0 {
		return nil, fmt.Errorf("at least one API key is required")
	}

	provider := &APIKeyAuthProvider{
		validAPIKeys: make(map[string]*AuthCredentials),
		cacheEnabled: config.CacheEnabled,
	}

	// Initialize API keys
	for apiKey, userID := range config.APIKeys {
		provider.validAPIKeys[apiKey] = &AuthCredentials{
			Type:   AuthTypeAPIKey,
			APIKey: apiKey,
			UserID: userID,
			Scopes: []string{"saga:execute", "saga:compensate", "saga:read"},
		}
	}

	if config.CacheEnabled && config.CacheTTL > 0 {
		provider.cache = NewAuthCache(config.CacheTTL)
	}

	return provider, nil
}

// Name implements AuthProvider interface
func (p *APIKeyAuthProvider) Name() string {
	return "apikey"
}

// Authenticate implements AuthProvider interface
func (p *APIKeyAuthProvider) Authenticate(ctx context.Context, credentials *AuthCredentials) (*AuthContext, error) {
	if credentials.Type != AuthTypeAPIKey {
		return nil, fmt.Errorf("%w: expected API Key, got %s", ErrInvalidAuthType, credentials.Type)
	}

	if credentials.APIKey == "" {
		return nil, fmt.Errorf("%w: API key is required", ErrMissingCredentials)
	}

	// Check cache first
	if p.cache != nil {
		if cached, found := p.cache.Get(credentials.APIKey); found {
			return cached, nil
		}
	}

	p.mu.RLock()
	validCreds, exists := p.validAPIKeys[credentials.APIKey]
	p.mu.RUnlock()

	if !exists {
		return nil, ErrInvalidAPIKey
	}

	// Create auth context with the valid credentials
	authCtx := NewAuthContext(validCreds, generateSagaAuthID())

	// Cache the result
	if p.cache != nil {
		p.cache.Set(credentials.APIKey, authCtx)
	}

	return authCtx, nil
}

// ValidateToken implements AuthProvider interface
func (p *APIKeyAuthProvider) ValidateToken(ctx context.Context, token string) (*AuthCredentials, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	creds, exists := p.validAPIKeys[token]
	if !exists {
		return nil, ErrInvalidAPIKey
	}

	return creds, nil
}

// IsHealthy implements AuthProvider interface
func (p *APIKeyAuthProvider) IsHealthy(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.validAPIKeys) > 0
}

// AuthCache provides caching for authentication results
type AuthCache struct {
	items map[string]*AuthContext
	ttl   time.Duration
	mu    sync.RWMutex
}

// NewAuthCache creates a new authentication cache
func NewAuthCache(ttl time.Duration) *AuthCache {
	cache := &AuthCache{
		items: make(map[string]*AuthContext),
		ttl:   ttl,
	}

	// Start cleanup routine
	go cache.cleanup()

	return cache
}

// Get retrieves an authentication context from the cache
func (c *AuthCache) Get(key string) (*AuthContext, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Check if item has expired
	if time.Since(item.Timestamp) > c.ttl {
		return nil, false
	}

	return item, true
}

// Set stores an authentication context in the cache
func (c *AuthCache) Set(key string, ctx *AuthContext) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = ctx
}

// Delete removes an item from the cache
func (c *AuthCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// cleanup periodically removes expired items from the cache
func (c *AuthCache) cleanup() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.Sub(item.Timestamp) > c.ttl {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// AuthManager manages authentication for Saga operations
type AuthManager struct {
	providers       map[string]AuthProvider
	defaultProvider string
	cache           *AuthCache
	cacheEnabled    bool
	mu              sync.RWMutex
	logger          *zap.Logger
}

// AuthManagerConfig configures the authentication manager
type AuthManagerConfig struct {
	DefaultProvider string
	Providers       map[string]AuthProvider
	CacheTTL        time.Duration
	CacheEnabled    bool
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config *AuthManagerConfig) (*AuthManager, error) {
	if config.DefaultProvider == "" {
		return nil, fmt.Errorf("default provider is required")
	}

	if _, exists := config.Providers[config.DefaultProvider]; !exists {
		return nil, fmt.Errorf("default provider '%s' not found", config.DefaultProvider)
	}

	manager := &AuthManager{
		providers:       config.Providers,
		defaultProvider: config.DefaultProvider,
		cacheEnabled:    config.CacheEnabled,
		logger:          logger.Logger,
	}

	// Initialize logger if not set
	if manager.logger == nil {
		manager.logger = zap.NewNop()
	}

	if config.CacheEnabled && config.CacheTTL > 0 {
		manager.cache = NewAuthCache(config.CacheTTL)
	}

	return manager, nil
}

// Authenticate authenticates credentials using the appropriate provider
func (m *AuthManager) Authenticate(ctx context.Context, credentials *AuthCredentials) (*AuthContext, error) {
	if credentials == nil {
		return &AuthContext{
			Credentials: &AuthCredentials{Type: AuthTypeNone},
			Timestamp:   time.Now(),
			SagaID:      generateSagaAuthID(),
		}, nil
	}

	if err := credentials.Validate(); err != nil {
		return nil, err
	}

	// Determine which provider to use
	providerName := m.defaultProvider
	if credentials.Type != AuthTypeNone {
		providerName = string(credentials.Type)
	}

	m.mu.RLock()
	provider, exists := m.providers[providerName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("authentication provider '%s' not found", providerName)
	}

	return provider.Authenticate(ctx, credentials)
}

// ValidateToken validates a token using the appropriate provider
func (m *AuthManager) ValidateToken(ctx context.Context, token string, authType AuthType) (*AuthCredentials, error) {
	providerName := string(authType)
	if providerName == "" {
		providerName = m.defaultProvider
	}

	m.mu.RLock()
	provider, exists := m.providers[providerName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("authentication provider '%s' not found", providerName)
	}

	return provider.ValidateToken(ctx, token)
}

// IsHealthy checks if all authentication providers are healthy
func (m *AuthManager) IsHealthy(ctx context.Context) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, provider := range m.providers {
		if !provider.IsHealthy(ctx) {
			return false
		}
	}

	return true
}

// AddProvider adds an authentication provider
func (m *AuthManager) AddProvider(name string, provider AuthProvider) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.providers[name]; exists {
		return fmt.Errorf("provider '%s' already exists", name)
	}

	m.providers[name] = provider
	return nil
}

// RemoveProvider removes an authentication provider
func (m *AuthManager) RemoveProvider(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == m.defaultProvider {
		return fmt.Errorf("cannot remove default provider")
	}

	if _, exists := m.providers[name]; !exists {
		return fmt.Errorf("provider '%s' not found", name)
	}

	delete(m.providers, name)
	return nil
}

// GetProvider returns an authentication provider by name
func (m *AuthManager) GetProvider(name string) (AuthProvider, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[name]
	return provider, exists
}

// SagaAuthMiddleware provides authentication middleware for Saga operations
type SagaAuthMiddleware struct {
	authManager *AuthManager
	required    bool
	logger      *zap.Logger
}

// SagaAuthMiddlewareConfig configures the Saga authentication middleware
type SagaAuthMiddlewareConfig struct {
	AuthManager *AuthManager
	Required    bool
}

// NewSagaAuthMiddleware creates a new Saga authentication middleware
func NewSagaAuthMiddleware(config *SagaAuthMiddlewareConfig) *SagaAuthMiddleware {
	middleware := &SagaAuthMiddleware{
		authManager: config.AuthManager,
		required:    config.Required,
		logger:      logger.Logger,
	}

	// Initialize logger if not set
	if middleware.logger == nil {
		middleware.logger = zap.NewNop()
	}

	return middleware
}

// Authenticate performs authentication for a Saga operation
func (m *SagaAuthMiddleware) Authenticate(ctx context.Context, credentials *AuthCredentials) (*AuthContext, error) {
	// If credentials are nil and authentication is not required, allow access
	if credentials == nil && !m.required {
		return NewAuthContext(&AuthCredentials{Type: AuthTypeNone}, generateSagaAuthID()), nil
	}

	// If credentials are nil but authentication is required, return error
	if credentials == nil && m.required {
		m.logger.Error("Authentication required but no credentials provided")
		return nil, ErrMissingCredentials
	}

	// Authenticate using the auth manager
	authCtx, err := m.authManager.Authenticate(ctx, credentials)
	if err != nil {
		m.logger.Error("Authentication failed",
			zap.String("auth_type", string(credentials.Type)),
			zap.String("user_id", credentials.UserID),
			zap.Error(err))
		return nil, err
	}

	// Check if credentials have expired
	if authCtx.Credentials.IsExpired() {
		m.logger.Error("Authentication credentials expired",
			zap.String("user_id", authCtx.Credentials.UserID))
		return nil, ErrExpiredToken
	}

	m.logger.Debug("Authentication successful",
		zap.String("auth_type", string(authCtx.Credentials.Type)),
		zap.String("user_id", authCtx.Credentials.UserID),
		zap.Strings("scopes", authCtx.Credentials.Scopes))

	return authCtx, nil
}

// RequireScope checks if the authenticated user has the required scope
func (m *SagaAuthMiddleware) RequireScope(authCtx *AuthContext, requiredScope string) error {
	if authCtx == nil || authCtx.Credentials == nil {
		return ErrMissingCredentials
	}

	if !authCtx.Credentials.HasScope(requiredScope) {
		m.logger.Error("Insufficient scope",
			zap.String("user_id", authCtx.Credentials.UserID),
			zap.String("required_scope", requiredScope),
			zap.Strings("available_scopes", authCtx.Credentials.Scopes))
		return fmt.Errorf("%w: required '%s'", ErrInsufficientScope, requiredScope)
	}

	return nil
}

// IsHealthy checks if the authentication middleware is healthy
func (m *SagaAuthMiddleware) IsHealthy(ctx context.Context) bool {
	return m.authManager.IsHealthy(ctx)
}

// generateSagaAuthID generates a unique ID for saga authentication
func generateSagaAuthID() string {
	return fmt.Sprintf("saga-auth-%d", time.Now().UnixNano())
}

// ContextWithAuth stores the authentication context in the context
func ContextWithAuth(ctx context.Context, authCtx *AuthContext) context.Context {
	return context.WithValue(ctx, "saga_auth_context", authCtx)
}

// AuthFromContext retrieves the authentication context from the context
func AuthFromContext(ctx context.Context) (*AuthContext, bool) {
	authCtx, ok := ctx.Value("saga_auth_context").(*AuthContext)
	return authCtx, ok
}
