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

package messaging

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// AuthCredentials represents authentication credentials for messaging operations
type AuthCredentials struct {
	// Type of authentication
	Type AuthType `json:"type" yaml:"type"`

	// JWT token (for JWT authentication)
	Token string `json:"token,omitempty" yaml:"token,omitempty"`

	// API key (for API key authentication)
	APIKey string `json:"api_key,omitempty" yaml:"api_key,omitempty"`

	// Client certificate and key (for certificate authentication)
	Certificate *tls.Certificate `json:"-" yaml:"-"`

	// User ID extracted from credentials
	UserID string `json:"user_id,omitempty" yaml:"user_id,omitempty"`

	// Scopes/permissions associated with the credentials
	Scopes []string `json:"scopes,omitempty" yaml:"scopes,omitempty"`

	// Expiration time for the credentials
	ExpiresAt *time.Time `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`

	// Metadata associated with the credentials
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Validate validates the authentication credentials
func (c *AuthCredentials) Validate() error {
	switch c.Type {
	case AuthTypeNone:
		// No validation needed
	case AuthTypeJWT:
		if c.Token == "" {
			return fmt.Errorf("token is required for JWT authentication")
		}
	case AuthTypeAPIKey:
		if c.APIKey == "" {
			return fmt.Errorf("api_key is required for API key authentication")
		}
	case AuthTypeCertificate:
		if c.Certificate == nil {
			return fmt.Errorf("certificate is required for certificate authentication")
		}
	default:
		return fmt.Errorf("unsupported authentication type: %s", c.Type)
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

// AuthContext represents the authentication context for a message
type AuthContext struct {
	// Credentials used for authentication
	Credentials *AuthCredentials `json:"credentials"`

	// Timestamp when the context was created
	Timestamp time.Time `json:"timestamp"`

	// Message ID for tracing
	MessageID string `json:"message_id"`

	// Correlation ID for distributed tracing
	CorrelationID string `json:"correlation_id,omitempty"`

	// Source of the authentication (e.g., "publisher", "consumer")
	Source string `json:"source"`
}

// NewAuthContext creates a new authentication context
func NewAuthContext(credentials *AuthCredentials, messageID string) *AuthContext {
	return &AuthContext{
		Credentials: credentials,
		Timestamp:   time.Now(),
		MessageID:   messageID,
		Source:      "messaging",
	}
}

// ToHeaders converts the authentication context to message headers
func (c *AuthContext) ToHeaders() map[string]string {
	headers := make(map[string]string)

	if c.Credentials != nil {
		headers["x-auth-type"] = string(c.Credentials.Type)
		headers["x-auth-user-id"] = c.Credentials.UserID
		headers["x-auth-timestamp"] = c.Timestamp.Format(time.RFC3339)
		headers["x-auth-message-id"] = c.MessageID

		if c.CorrelationID != "" {
			headers["x-auth-correlation-id"] = c.CorrelationID
		}

		if c.Credentials.Type == AuthTypeJWT {
			headers["authorization"] = "Bearer " + c.Credentials.Token
		} else if c.Credentials.Type == AuthTypeAPIKey {
			headers["x-api-key"] = c.Credentials.APIKey
		}

		if len(c.Credentials.Scopes) > 0 {
			headers["x-auth-scopes"] = strings.Join(c.Credentials.Scopes, ",")
		}

		for k, v := range c.Credentials.Metadata {
			headers["x-auth-meta-"+k] = v
		}
	}

	return headers
}

// FromHeaders creates an authentication context from message headers
func AuthContextFromHeaders(headers map[string]string) (*AuthContext, error) {
	authType := headers["x-auth-type"]
	if authType == "" {
		return &AuthContext{
			Credentials: &AuthCredentials{Type: AuthTypeNone},
			Timestamp:   time.Now(),
			MessageID:   headers["x-auth-message-id"],
		}, nil
	}

	credentials := &AuthCredentials{
		Type:     AuthType(authType),
		UserID:   headers["x-auth-user-id"],
		Metadata: make(map[string]string),
	}

	// Extract timestamp
	if timestampStr := headers["x-auth-timestamp"]; timestampStr != "" {
		if timestamp, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			credentials.ExpiresAt = &timestamp
		}
	}

	// Extract scopes
	if scopesStr := headers["x-auth-scopes"]; scopesStr != "" {
		credentials.Scopes = strings.Split(scopesStr, ",")
	}

	// Extract authentication details based on type
	switch credentials.Type {
	case AuthTypeJWT:
		if authHeader := headers["authorization"]; authHeader != "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				credentials.Token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
	case AuthTypeAPIKey:
		credentials.APIKey = headers["x-api-key"]
	}

	// Extract metadata
	for k, v := range headers {
		if strings.HasPrefix(k, "x-auth-meta-") {
			key := strings.TrimPrefix(k, "x-auth-meta-")
			credentials.Metadata[key] = v
		}
	}

	// Validate credentials
	if err := credentials.Validate(); err != nil {
		return nil, fmt.Errorf("invalid authentication credentials: %w", err)
	}

	return &AuthContext{
		Credentials:   credentials,
		Timestamp:     time.Now(),
		MessageID:     headers["x-auth-message-id"],
		CorrelationID: headers["x-auth-correlation-id"],
		Source:        headers["x-auth-source"],
	}, nil
}

// AuthProvider defines the interface for authentication providers
type AuthProvider interface {
	// Name returns the provider name
	Name() string

	// Authenticate authenticates the given credentials
	Authenticate(ctx context.Context, credentials *AuthCredentials) (*AuthContext, error)

	// ValidateToken validates a token and returns the associated credentials
	ValidateToken(ctx context.Context, token string) (*AuthCredentials, error)

	// RefreshToken refreshes an expired token
	RefreshToken(ctx context.Context, credentials *AuthCredentials) (*AuthCredentials, error)

	// IsHealthy checks if the authentication provider is healthy
	IsHealthy(ctx context.Context) bool
}

// JWTAuthProvider implements JWT-based authentication for messaging
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
	Secret       string        `json:"secret" yaml:"secret"`
	Issuer       string        `json:"issuer,omitempty" yaml:"issuer,omitempty"`
	Audience     string        `json:"audience,omitempty" yaml:"audience,omitempty"`
	CacheTTL     time.Duration `json:"cache_ttl,omitempty" yaml:"cache_ttl,omitempty"`
	CacheEnabled bool          `json:"cache_enabled,omitempty" yaml:"cache_enabled,omitempty"`
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
		return nil, fmt.Errorf("unsupported credential type: %s", credentials.Type)
	}

	if credentials.Token == "" {
		return nil, fmt.Errorf("token is required for JWT authentication")
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
		return nil, fmt.Errorf("JWT token validation failed: %w", err)
	}

	// Extract user ID and scopes
	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user ID in JWT claims")
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
		if key != "user_id" && key != "scopes" && key != "exp" && key != "iat" && key != "nbf" && key != "jti" {
			if strVal, ok := value.(string); ok {
				authCreds.Metadata[key] = strVal
			}
		}
	}

	// Create auth context
	authCtx := NewAuthContext(authCreds, generateMessageID())

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

// RefreshToken implements AuthProvider interface
func (p *JWTAuthProvider) RefreshToken(ctx context.Context, credentials *AuthCredentials) (*AuthCredentials, error) {
	// For JWT, we typically use refresh tokens rather than refreshing the access token
	// This would be implemented based on your specific JWT refresh strategy
	return nil, fmt.Errorf("JWT token refresh not implemented")
}

// IsHealthy implements AuthProvider interface
func (p *JWTAuthProvider) IsHealthy(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Basic health check - secret is available
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
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// APIKeyAuthProvider implements API key-based authentication for messaging
type APIKeyAuthProvider struct {
	validAPIKeys map[string]*AuthCredentials
	cache        *AuthCache
	cacheEnabled bool
	mu           sync.RWMutex
}

// APIKeyAuthProviderConfig configures the API key authentication provider
type APIKeyAuthProviderConfig struct {
	APIKeys      map[string]string `json:"api_keys" yaml:"api_keys"` // key -> user_id mapping
	CacheTTL     time.Duration     `json:"cache_ttl,omitempty" yaml:"cache_ttl,omitempty"`
	CacheEnabled bool              `json:"cache_enabled,omitempty" yaml:"cache_enabled,omitempty"`
}

// NewAPIKeyAuthProvider creates a new API key authentication provider
func NewAPIKeyAuthProvider(config *APIKeyAuthProviderConfig) (*APIKeyAuthProvider, error) {
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
			Scopes: []string{"messaging:publish", "messaging:consume"},
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
		return nil, fmt.Errorf("unsupported credential type: %s", credentials.Type)
	}

	if credentials.APIKey == "" {
		return nil, fmt.Errorf("API key is required for API key authentication")
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
		return nil, fmt.Errorf("invalid API key")
	}

	// Create auth context
	authCtx := NewAuthContext(validCreds, generateMessageID())

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
		return nil, fmt.Errorf("invalid API key")
	}

	return creds, nil
}

// RefreshToken implements AuthProvider interface
func (p *APIKeyAuthProvider) RefreshToken(ctx context.Context, credentials *AuthCredentials) (*AuthCredentials, error) {
	// API keys don't expire, so no refresh needed
	return credentials, nil
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

// AuthManager manages authentication for messaging operations
type AuthManager struct {
	providers       map[string]AuthProvider
	defaultProvider string
	cache           *AuthCache
	cacheEnabled    bool
	mu              sync.RWMutex
}

// AuthManagerConfig configures the authentication manager
type AuthManagerConfig struct {
	DefaultProvider string                  `json:"default_provider" yaml:"default_provider"`
	Providers       map[string]AuthProvider `json:"providers" yaml:"providers"`
	CacheTTL        time.Duration           `json:"cache_ttl,omitempty" yaml:"cache_ttl,omitempty"`
	CacheEnabled    bool                    `json:"cache_enabled,omitempty" yaml:"cache_enabled,omitempty"`
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
			MessageID:   generateMessageID(),
		}, nil
	}

	if err := credentials.Validate(); err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
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

// GetProviders returns all registered authentication providers
func (m *AuthManager) GetProviders() map[string]AuthProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make(map[string]AuthProvider)
	for name, provider := range m.providers {
		providers[name] = provider
	}

	return providers
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}

// AuthMiddleware creates middleware for message authentication
type AuthMiddleware struct {
	authManager *AuthManager
	required    bool
	logger      *zap.Logger
}

// AuthMiddlewareConfig configures the authentication middleware
type AuthMiddlewareConfig struct {
	AuthManager *AuthManager `json:"auth_manager" yaml:"auth_manager"`
	Required    bool         `json:"required" yaml:"required"`
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(config *AuthMiddlewareConfig) *AuthMiddleware {
	return &AuthMiddleware{
		authManager: config.AuthManager,
		required:    config.Required,
		logger:      logger.Logger,
	}
}

// Name implements Middleware interface
func (m *AuthMiddleware) Name() string {
	return "authentication"
}

// Wrap implements Middleware interface
func (m *AuthMiddleware) Wrap(next MessageHandler) MessageHandler {
	return &authMessageHandler{
		next:        next,
		authManager: m.authManager,
		required:    m.required,
		logger:      m.logger,
	}
}

// authMessageHandler implements MessageHandler with authentication
type authMessageHandler struct {
	next        MessageHandler
	authManager *AuthManager
	required    bool
	logger      *zap.Logger
}

// Handle implements MessageHandler interface
func (h *authMessageHandler) Handle(ctx context.Context, message *Message) error {
	// Extract authentication context from message headers
	authCtx, err := AuthContextFromHeaders(message.Headers)
	if err != nil {
		if h.required {
			h.logger.Error("Authentication failed",
				zap.String("message_id", message.ID),
				zap.Error(err))
			return fmt.Errorf("authentication failed: %w", err)
		}

		// If authentication is not required, proceed with anonymous access
		authCtx = NewAuthContext(&AuthCredentials{Type: AuthTypeNone}, message.ID)
	}

	// Validate authentication context if required
	if h.required && authCtx.Credentials.Type != AuthTypeNone {
		if authCtx.Credentials.IsExpired() {
			h.logger.Error("Authentication credentials expired",
				zap.String("message_id", message.ID),
				zap.String("user_id", authCtx.Credentials.UserID))
			return fmt.Errorf("authentication credentials expired")
		}

		// Validate credentials with auth manager
		if _, err := h.authManager.Authenticate(ctx, authCtx.Credentials); err != nil {
			h.logger.Error("Authentication validation failed",
				zap.String("message_id", message.ID),
				zap.String("user_id", authCtx.Credentials.UserID),
				zap.Error(err))
			return fmt.Errorf("authentication validation failed: %w", err)
		}
	}

	// Add authentication context to the message context
	ctx = context.WithValue(ctx, "auth_context", authCtx)

	// Call the next handler
	return h.next.Handle(ctx, message)
}

// OnError implements MessageHandler interface
func (h *authMessageHandler) OnError(ctx context.Context, message *Message, err error) ErrorAction {
	// Get authentication context from context
	if authCtx, ok := ctx.Value("auth_context").(*AuthContext); ok {
		h.logger.Error("Message handling error",
			zap.String("message_id", message.ID),
			zap.String("user_id", authCtx.Credentials.UserID),
			zap.Error(err))
	} else {
		h.logger.Error("Message handling error",
			zap.String("message_id", message.ID),
			zap.Error(err))
	}

	// Determine error action based on authentication status
	if errors.Is(err, NewAuthError(ErrAuthenticationFailed, "")) || errors.Is(err, ErrCredentialsExpired) {
		return ErrorActionDeadLetter
	}

	return ErrorActionRetry
}

// Authentication errors
var (
	ErrCredentialsExpired = errors.New("credentials expired")
	ErrProviderNotFound   = errors.New("authentication provider not found")
)

// AuthError wraps ErrorCode to implement error interface
type AuthError struct {
	Code ErrorCode
	Msg  string
}

func (e *AuthError) Error() string {
	return e.Msg
}

func (e *AuthError) Is(target error) bool {
	if other, ok := target.(*AuthError); ok {
		return e.Code == other.Code
	}
	return false
}

func NewAuthError(code ErrorCode, msg string) *AuthError {
	return &AuthError{Code: code, Msg: msg}
}
