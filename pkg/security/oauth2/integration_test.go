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
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MockOIDCServer provides a mock OIDC provider for integration testing.
type MockOIDCServer struct {
	server        *httptest.Server
	privateKey    *rsa.PrivateKey
	publicKey     *rsa.PublicKey
	issuer        string
	clientID      string
	clientSecret  string
	mu            sync.Mutex
	tokens        map[string]*mockToken
	authCodes     map[string]*mockAuthCode
	refreshTokens map[string]*mockRefreshToken
}

type mockToken struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	ExpiresIn    int64
	IssuedAt     time.Time
}

type mockAuthCode struct {
	Code        string
	ClientID    string
	RedirectURI string
	Scopes      []string
	ExpiresAt   time.Time
	Used        bool
}

type mockRefreshToken struct {
	Token     string
	ClientID  string
	Subject   string
	Scopes    []string
	ExpiresAt time.Time
}

// NewMockOIDCServer creates a new mock OIDC server for testing.
func NewMockOIDCServer() (*MockOIDCServer, error) {
	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	mock := &MockOIDCServer{
		privateKey:    privateKey,
		publicKey:     &privateKey.PublicKey,
		clientID:      "test-client",
		clientSecret:  "test-secret",
		tokens:        make(map[string]*mockToken),
		authCodes:     make(map[string]*mockAuthCode),
		refreshTokens: make(map[string]*mockRefreshToken),
	}

	// Create HTTP server
	mux := http.NewServeMux()

	// OIDC discovery endpoint
	mux.HandleFunc("/.well-known/openid-configuration", mock.handleDiscovery)

	// JWKS endpoint
	mux.HandleFunc("/jwks", mock.handleJWKS)

	// Authorization endpoint
	mux.HandleFunc("/authorize", mock.handleAuthorize)

	// Token endpoint
	mux.HandleFunc("/token", mock.handleToken)

	// UserInfo endpoint
	mux.HandleFunc("/userinfo", mock.handleUserInfo)

	// Token introspection endpoint
	mux.HandleFunc("/introspect", mock.handleIntrospect)

	// Token revocation endpoint
	mux.HandleFunc("/revoke", mock.handleRevoke)

	mock.server = httptest.NewServer(mux)
	mock.issuer = mock.server.URL

	return mock, nil
}

// Close closes the mock OIDC server.
func (m *MockOIDCServer) Close() {
	m.server.Close()
}

// GetIssuerURL returns the issuer URL of the mock server.
func (m *MockOIDCServer) GetIssuerURL() string {
	return m.issuer
}

// GetClientID returns the client ID.
func (m *MockOIDCServer) GetClientID() string {
	return m.clientID
}

// GetClientSecret returns the client secret.
func (m *MockOIDCServer) GetClientSecret() string {
	return m.clientSecret
}

// handleDiscovery handles OIDC discovery requests.
func (m *MockOIDCServer) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	discovery := map[string]interface{}{
		"issuer":                                m.issuer,
		"authorization_endpoint":                m.issuer + "/authorize",
		"token_endpoint":                        m.issuer + "/token",
		"userinfo_endpoint":                     m.issuer + "/userinfo",
		"jwks_uri":                              m.issuer + "/jwks",
		"introspection_endpoint":                m.issuer + "/introspect",
		"revocation_endpoint":                   m.issuer + "/revoke",
		"scopes_supported":                      []string{"openid", "profile", "email"},
		"response_types_supported":              []string{"code", "token", "id_token", "code id_token"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(discovery)
}

// handleJWKS handles JWKS requests.
func (m *MockOIDCServer) handleJWKS(w http.ResponseWriter, r *http.Request) {
	// Return a simple JWKS with the public key
	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"use": "sig",
				"kid": "test-key-1",
				"alg": "RS256",
				"n":   encodeBase64(m.publicKey.N.Bytes()),
				"e":   "AQAB",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jwks)
}

// handleAuthorize handles authorization requests.
func (m *MockOIDCServer) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	clientID := query.Get("client_id")
	redirectURI := query.Get("redirect_uri")
	state := query.Get("state")
	scopes := strings.Split(query.Get("scope"), " ")

	if clientID != m.clientID {
		http.Error(w, "invalid client_id", http.StatusBadRequest)
		return
	}

	// Generate authorization code
	code, _ := generateRandomString(32)
	m.mu.Lock()
	m.authCodes[code] = &mockAuthCode{
		Code:        code,
		ClientID:    clientID,
		RedirectURI: redirectURI,
		Scopes:      scopes,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		Used:        false,
	}
	m.mu.Unlock()

	// Redirect back with code and state
	redirectURL, _ := url.Parse(redirectURI)
	q := redirectURL.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	redirectURL.RawQuery = q.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// handleToken handles token exchange requests.
func (m *MockOIDCServer) handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	grantType := r.Form.Get("grant_type")

	// Extract client credentials from Authorization header (HTTP Basic Auth) or form parameters
	// OAuth2 spec (RFC 6749 Section 2.3.1) recommends HTTP Basic Auth for confidential clients
	var clientID, clientSecret string

	// Try HTTP Basic Auth first (standard method used by golang.org/x/oauth2)
	if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Basic ") {
		// Decode Basic auth credentials
		basicAuth := strings.TrimPrefix(authHeader, "Basic ")
		decoded, err := base64.StdEncoding.DecodeString(basicAuth)
		if err == nil {
			// Split username:password
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				clientID = parts[0]
				clientSecret = parts[1]
			}
		}
	}

	// Fall back to form parameters if Basic Auth not present
	if clientID == "" {
		clientID = r.Form.Get("client_id")
		clientSecret = r.Form.Get("client_secret")
	}

	// Validate client credentials
	if clientID != m.clientID || clientSecret != m.clientSecret {
		http.Error(w, "invalid client credentials", http.StatusUnauthorized)
		return
	}

	switch grantType {
	case "authorization_code":
		m.handleAuthorizationCodeGrant(w, r)
	case "refresh_token":
		m.handleRefreshTokenGrant(w, r)
	default:
		http.Error(w, "unsupported grant type", http.StatusBadRequest)
	}
}

// handleAuthorizationCodeGrant handles authorization code grant.
func (m *MockOIDCServer) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
	code := r.Form.Get("code")
	redirectURI := r.Form.Get("redirect_uri")

	m.mu.Lock()
	authCode, exists := m.authCodes[code]
	m.mu.Unlock()

	if !exists || authCode.Used || time.Now().After(authCode.ExpiresAt) {
		http.Error(w, "invalid or expired authorization code", http.StatusBadRequest)
		return
	}

	if authCode.RedirectURI != redirectURI {
		http.Error(w, "redirect_uri mismatch", http.StatusBadRequest)
		return
	}

	// Mark code as used
	m.mu.Lock()
	authCode.Used = true
	m.mu.Unlock()

	// Generate tokens with timestamp for uniqueness
	baseAccessToken, _ := generateRandomString(24)
	baseRefreshToken, _ := generateRandomString(24)
	timestamp := fmt.Sprintf("-%d", time.Now().UnixNano())
	accessToken := baseAccessToken + timestamp
	refreshToken := baseRefreshToken + timestamp
	idToken := m.generateIDToken("test-user", authCode.Scopes)

	// Store tokens
	m.mu.Lock()
	m.tokens[accessToken] = &mockToken{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		IssuedAt:     time.Now(),
	}
	m.refreshTokens[refreshToken] = &mockRefreshToken{
		Token:     refreshToken,
		ClientID:  authCode.ClientID,
		Subject:   "test-user",
		Scopes:    authCode.Scopes,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	m.mu.Unlock()

	// Return token response
	response := map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": refreshToken,
		"id_token":      idToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRefreshTokenGrant handles refresh token grant.
func (m *MockOIDCServer) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	refreshTokenStr := r.Form.Get("refresh_token")

	m.mu.Lock()
	refreshToken, exists := m.refreshTokens[refreshTokenStr]
	m.mu.Unlock()

	if !exists || time.Now().After(refreshToken.ExpiresAt) {
		http.Error(w, "invalid or expired refresh token", http.StatusBadRequest)
		return
	}

	// Generate new tokens (different from old ones by adding timestamp)
	baseAccessToken, _ := generateRandomString(24)
	baseRefreshToken, _ := generateRandomString(24)

	// Add timestamp suffix to ensure uniqueness
	timestamp := fmt.Sprintf("-%d", time.Now().UnixNano())
	accessToken := baseAccessToken + timestamp
	newRefreshToken := baseRefreshToken + timestamp

	idToken := m.generateIDToken(refreshToken.Subject, refreshToken.Scopes)

	// Store new tokens
	m.mu.Lock()
	m.tokens[accessToken] = &mockToken{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    3600,
		IssuedAt:     time.Now(),
	}
	// Invalidate old refresh token and create new one
	delete(m.refreshTokens, refreshTokenStr)
	m.refreshTokens[newRefreshToken] = &mockRefreshToken{
		Token:     newRefreshToken,
		ClientID:  refreshToken.ClientID,
		Subject:   refreshToken.Subject,
		Scopes:    refreshToken.Scopes,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	m.mu.Unlock()

	// Return token response
	response := map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": newRefreshToken,
		"id_token":      idToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleUserInfo handles userinfo requests.
func (m *MockOIDCServer) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	// Extract bearer token
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "missing or invalid authorization header", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	m.mu.Lock()
	_, exists := m.tokens[token]
	m.mu.Unlock()

	if !exists {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// Return user info
	userInfo := map[string]interface{}{
		"sub":   "test-user",
		"name":  "Test User",
		"email": "test@example.com",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInfo)
}

// handleIntrospect handles token introspection requests.
func (m *MockOIDCServer) handleIntrospect(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	token := r.Form.Get("token")

	m.mu.Lock()
	mockToken, exists := m.tokens[token]
	m.mu.Unlock()

	if !exists {
		// Return inactive token
		response := map[string]interface{}{
			"active": false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if token is expired
	active := time.Since(mockToken.IssuedAt) < time.Duration(mockToken.ExpiresIn)*time.Second

	response := map[string]interface{}{
		"active":     active,
		"sub":        "test-user",
		"client_id":  m.clientID,
		"token_type": "Bearer",
		"exp":        mockToken.IssuedAt.Add(time.Duration(mockToken.ExpiresIn) * time.Second).Unix(),
		"iat":        mockToken.IssuedAt.Unix(),
		"scope":      "openid profile email",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRevoke handles token revocation requests.
func (m *MockOIDCServer) handleRevoke(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	token := r.Form.Get("token")

	// Remove token from storage
	m.mu.Lock()
	delete(m.tokens, token)
	delete(m.refreshTokens, token)
	m.mu.Unlock()

	// RFC 7009: always return 200 OK
	w.WriteHeader(http.StatusOK)
}

// generateIDToken generates a mock ID token.
func (m *MockOIDCServer) generateIDToken(subject string, scopes []string) string {
	now := time.Now()

	claims := jwt.MapClaims{
		"iss":   m.issuer,
		"sub":   subject,
		"aud":   m.clientID,
		"exp":   now.Add(1 * time.Hour).Unix(),
		"iat":   now.Unix(),
		"nonce": "test-nonce",
	}

	// Add email claim if email scope is present
	for _, scope := range scopes {
		if scope == "email" {
			claims["email"] = "test@example.com"
			claims["email_verified"] = true
		}
		if scope == "profile" {
			claims["name"] = "Test User"
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key-1"

	signedToken, _ := token.SignedString(m.privateKey)
	return signedToken
}

// TestIntegrationAuthorizationCodeFlow tests the full authorization code flow.
func TestIntegrationAuthorizationCodeFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock OIDC server
	mockServer, err := NewMockOIDCServer()
	if err != nil {
		t.Fatalf("Failed to create mock OIDC server: %v", err)
	}
	defer mockServer.Close()

	// Create OAuth2 client with OIDC discovery
	config := &Config{
		Enabled:      true,
		Provider:     "keycloak",
		ClientID:     mockServer.GetClientID(),
		ClientSecret: mockServer.GetClientSecret(),
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"openid", "profile", "email"},
		IssuerURL:    mockServer.GetIssuerURL(),
		UseDiscovery: true,
		HTTPTimeout:  10 * time.Second,
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create OAuth2 client: %v", err)
	}
	defer client.Close()

	// Test 1: Generate authorization URL
	state := "test-state"
	authURL := client.AuthCodeURL(state)

	if !strings.Contains(authURL, mockServer.GetIssuerURL()) {
		t.Errorf("Authorization URL does not contain issuer URL")
	}
	if !strings.Contains(authURL, "state="+state) {
		t.Errorf("Authorization URL does not contain state parameter")
	}

	// Test 2: Simulate authorization and get code
	// In real flow, user would be redirected to authorization endpoint
	// Here we directly call the mock server to get a code
	code := simulateAuthorizationFlow(t, mockServer, config.RedirectURL, config.Scopes)

	// Test 3: Exchange code for token
	token, err := client.Exchange(ctx, code)
	if err != nil {
		t.Fatalf("Failed to exchange code for token: %v", err)
	}

	if token.AccessToken == "" {
		t.Error("Access token is empty")
	}
	if token.RefreshToken == "" {
		t.Error("Refresh token is empty")
	}

	// Test 4: Verify ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		t.Fatal("ID token not found in token response")
	}

	idToken, err := client.VerifyIDToken(ctx, rawIDToken)
	if err != nil {
		t.Fatalf("Failed to verify ID token: %v", err)
	}

	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		t.Fatalf("Failed to extract ID token claims: %v", err)
	}

	if claims["sub"] != "test-user" {
		t.Errorf("Expected subject 'test-user', got %v", claims["sub"])
	}
}

// TestIntegrationTokenRefresh tests token refresh flow.
func TestIntegrationTokenRefresh(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock OIDC server
	mockServer, err := NewMockOIDCServer()
	if err != nil {
		t.Fatalf("Failed to create mock OIDC server: %v", err)
	}
	defer mockServer.Close()

	// Create OAuth2 client
	config := &Config{
		Enabled:      true,
		Provider:     "keycloak",
		ClientID:     mockServer.GetClientID(),
		ClientSecret: mockServer.GetClientSecret(),
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"openid", "profile", "email"},
		IssuerURL:    mockServer.GetIssuerURL(),
		UseDiscovery: true,
		HTTPTimeout:  10 * time.Second,
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create OAuth2 client: %v", err)
	}
	defer client.Close()

	// Get initial token
	code := simulateAuthorizationFlow(t, mockServer, config.RedirectURL, config.Scopes)
	token, err := client.Exchange(ctx, code)
	if err != nil {
		t.Fatalf("Failed to exchange code for token: %v", err)
	}

	originalAccessToken := token.AccessToken
	originalRefreshToken := token.RefreshToken

	t.Logf("Original access token: %s", originalAccessToken[:20]+"...")
	t.Logf("Original refresh token: %s", originalRefreshToken[:20]+"...")

	// Force token expiry to trigger refresh
	token.Expiry = time.Now().Add(-1 * time.Hour)

	// Refresh token
	newToken, err := client.RefreshToken(ctx, token)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	t.Logf("New access token: %s", newToken.AccessToken[:20]+"...")
	t.Logf("New refresh token: %s", newToken.RefreshToken[:20]+"...")

	if newToken.AccessToken == originalAccessToken {
		t.Errorf("Access token was not refreshed - same token returned")
	}
	if newToken.RefreshToken == originalRefreshToken {
		t.Errorf("Refresh token was not rotated - same refresh token returned")
	}
	if newToken.AccessToken == "" {
		t.Error("New access token is empty")
	}
}

// TestIntegrationTokenIntrospection tests token introspection.
func TestIntegrationTokenIntrospection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock OIDC server
	mockServer, err := NewMockOIDCServer()
	if err != nil {
		t.Fatalf("Failed to create mock OIDC server: %v", err)
	}
	defer mockServer.Close()

	// Create OAuth2 client with explicit token URL for introspection
	config := &Config{
		Enabled:      true,
		Provider:     "keycloak",
		ClientID:     mockServer.GetClientID(),
		ClientSecret: mockServer.GetClientSecret(),
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"openid", "profile", "email"},
		IssuerURL:    mockServer.GetIssuerURL(),
		UseDiscovery: true,
		TokenURL:     mockServer.GetIssuerURL() + "/token", // Explicitly set TokenURL
		HTTPTimeout:  10 * time.Second,
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create OAuth2 client: %v", err)
	}
	defer client.Close()

	// Get token
	code := simulateAuthorizationFlow(t, mockServer, config.RedirectURL, config.Scopes)
	token, err := client.Exchange(ctx, code)
	if err != nil {
		t.Fatalf("Failed to exchange code for token: %v", err)
	}

	// Introspect token
	introspection, err := client.IntrospectToken(ctx, token.AccessToken)
	if err != nil {
		t.Fatalf("Failed to introspect token: %v", err)
	}

	if !introspection.Active {
		t.Error("Token should be active")
	}
	if introspection.Sub != "test-user" {
		t.Errorf("Expected subject 'test-user', got %s", introspection.Sub)
	}
}

// TestIntegrationTokenRevocation tests token revocation.
func TestIntegrationTokenRevocation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock OIDC server
	mockServer, err := NewMockOIDCServer()
	if err != nil {
		t.Fatalf("Failed to create mock OIDC server: %v", err)
	}
	defer mockServer.Close()

	// Create OAuth2 client with explicit token URL for revocation
	config := &Config{
		Enabled:      true,
		Provider:     "keycloak",
		ClientID:     mockServer.GetClientID(),
		ClientSecret: mockServer.GetClientSecret(),
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"openid", "profile", "email"},
		IssuerURL:    mockServer.GetIssuerURL(),
		UseDiscovery: true,
		TokenURL:     mockServer.GetIssuerURL() + "/token", // Explicitly set TokenURL
		HTTPTimeout:  10 * time.Second,
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create OAuth2 client: %v", err)
	}
	defer client.Close()

	// Get token
	code := simulateAuthorizationFlow(t, mockServer, config.RedirectURL, config.Scopes)
	token, err := client.Exchange(ctx, code)
	if err != nil {
		t.Fatalf("Failed to exchange code for token: %v", err)
	}

	// Revoke token
	err = client.RevokeToken(ctx, token.AccessToken)
	if err != nil {
		t.Fatalf("Failed to revoke token: %v", err)
	}

	// Try to introspect revoked token
	introspection, err := client.IntrospectToken(ctx, token.AccessToken)
	if err != nil {
		t.Fatalf("Failed to introspect token: %v", err)
	}

	if introspection.Active {
		t.Error("Token should be inactive after revocation")
	}
}

// simulateAuthorizationFlow simulates the authorization flow and returns an authorization code.
func simulateAuthorizationFlow(t testing.TB, mockServer *MockOIDCServer, redirectURI string, scopes []string) string {
	// Generate authorization code directly
	code, _ := generateRandomString(32)
	mockServer.mu.Lock()
	mockServer.authCodes[code] = &mockAuthCode{
		Code:        code,
		ClientID:    mockServer.GetClientID(),
		RedirectURI: redirectURI,
		Scopes:      scopes,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		Used:        false,
	}
	mockServer.mu.Unlock()
	return code
}

// encodeBase64 encodes bytes to base64 URL encoding without padding.
func encodeBase64(b []byte) string {
	return strings.TrimRight(base64.RawURLEncoding.EncodeToString(b), "=")
}

// BenchmarkOAuth2ClientCreation benchmarks OAuth2 client creation.
func BenchmarkOAuth2ClientCreation(b *testing.B) {
	mockServer, err := NewMockOIDCServer()
	if err != nil {
		b.Fatalf("Failed to create mock OIDC server: %v", err)
	}
	defer mockServer.Close()

	config := &Config{
		Enabled:      true,
		Provider:     "keycloak",
		ClientID:     mockServer.GetClientID(),
		ClientSecret: mockServer.GetClientSecret(),
		IssuerURL:    mockServer.GetIssuerURL(),
		UseDiscovery: true,
		HTTPTimeout:  10 * time.Second,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, err := NewClient(ctx, config)
		if err != nil {
			b.Fatalf("Failed to create client: %v", err)
		}
		client.Close()
	}
}

// BenchmarkTokenExchange benchmarks token exchange performance.
func BenchmarkTokenExchange(b *testing.B) {
	mockServer, err := NewMockOIDCServer()
	if err != nil {
		b.Fatalf("Failed to create mock OIDC server: %v", err)
	}
	defer mockServer.Close()

	config := &Config{
		Enabled:      true,
		Provider:     "keycloak",
		ClientID:     mockServer.GetClientID(),
		ClientSecret: mockServer.GetClientSecret(),
		RedirectURL:  "http://localhost:8080/callback",
		IssuerURL:    mockServer.GetIssuerURL(),
		UseDiscovery: true,
		HTTPTimeout:  10 * time.Second,
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Pre-generate codes
	codes := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		codes[i] = simulateAuthorizationFlow(b, mockServer, config.RedirectURL, config.Scopes)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Exchange(ctx, codes[i])
		if err != nil {
			b.Fatalf("Failed to exchange token: %v", err)
		}
	}
}

// BenchmarkTokenIntrospection benchmarks token introspection performance.
func BenchmarkTokenIntrospection(b *testing.B) {
	mockServer, err := NewMockOIDCServer()
	if err != nil {
		b.Fatalf("Failed to create mock OIDC server: %v", err)
	}
	defer mockServer.Close()

	config := &Config{
		Enabled:      true,
		Provider:     "keycloak",
		ClientID:     mockServer.GetClientID(),
		ClientSecret: mockServer.GetClientSecret(),
		RedirectURL:  "http://localhost:8080/callback",
		IssuerURL:    mockServer.GetIssuerURL(),
		UseDiscovery: true,
		HTTPTimeout:  10 * time.Second,
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Get a token
	code := simulateAuthorizationFlow(b, mockServer, config.RedirectURL, config.Scopes)
	token, err := client.Exchange(ctx, code)
	if err != nil {
		b.Fatalf("Failed to exchange token: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.IntrospectToken(ctx, token.AccessToken)
		if err != nil {
			b.Fatalf("Failed to introspect token: %v", err)
		}
	}
}

// BenchmarkIDTokenVerification benchmarks ID token verification performance.
func BenchmarkIDTokenVerification(b *testing.B) {
	mockServer, err := NewMockOIDCServer()
	if err != nil {
		b.Fatalf("Failed to create mock OIDC server: %v", err)
	}
	defer mockServer.Close()

	config := &Config{
		Enabled:      true,
		Provider:     "keycloak",
		ClientID:     mockServer.GetClientID(),
		ClientSecret: mockServer.GetClientSecret(),
		RedirectURL:  "http://localhost:8080/callback",
		IssuerURL:    mockServer.GetIssuerURL(),
		UseDiscovery: true,
		HTTPTimeout:  10 * time.Second,
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Get a token with ID token
	code := simulateAuthorizationFlow(b, mockServer, config.RedirectURL, []string{"openid", "profile", "email"})
	token, err := client.Exchange(ctx, code)
	if err != nil {
		b.Fatalf("Failed to exchange token: %v", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		b.Fatal("ID token not found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.VerifyIDToken(ctx, rawIDToken)
		if err != nil {
			b.Fatalf("Failed to verify ID token: %v", err)
		}
	}
}

// generateAuthCode generates an authorization code for testing.
func (m *MockOIDCServer) generateAuthCode(clientID, redirectURI string, scopes []string) string {
	code, _ := generateRandomString(32)
	m.mu.Lock()
	m.authCodes[code] = &mockAuthCode{
		Code:        code,
		ClientID:    clientID,
		RedirectURI: redirectURI,
		Scopes:      scopes,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		Used:        false,
	}
	m.mu.Unlock()
	return code
}
