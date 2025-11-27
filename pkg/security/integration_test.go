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

package security_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/innovationmech/swit/pkg/middleware"
	"github.com/innovationmech/swit/pkg/security"
	"github.com/innovationmech/swit/pkg/security/opa"
	tlspkg "github.com/innovationmech/swit/pkg/security/tls"
)

// =============================================================================
// Test Infrastructure: Mock OIDC Server
// =============================================================================

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
	Subject      string
	Roles        []string
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

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", mock.handleDiscovery)
	mux.HandleFunc("/jwks", mock.handleJWKS)
	mux.HandleFunc("/authorize", mock.handleAuthorize)
	mux.HandleFunc("/token", mock.handleToken)
	mux.HandleFunc("/userinfo", mock.handleUserInfo)
	mux.HandleFunc("/introspect", mock.handleIntrospect)
	mux.HandleFunc("/revoke", mock.handleRevoke)

	mock.server = httptest.NewServer(mux)
	mock.issuer = mock.server.URL

	return mock, nil
}

func (m *MockOIDCServer) Close() {
	m.server.Close()
}

func (m *MockOIDCServer) GetIssuerURL() string {
	return m.issuer
}

func (m *MockOIDCServer) GetClientID() string {
	return m.clientID
}

func (m *MockOIDCServer) GetClientSecret() string {
	return m.clientSecret
}

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
		"response_types_supported":              []string{"code", "token", "id_token"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(discovery)
}

func (m *MockOIDCServer) handleJWKS(w http.ResponseWriter, r *http.Request) {
	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"use": "sig",
				"kid": "test-key-1",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(m.publicKey.N.Bytes()),
				"e":   "AQAB",
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jwks)
}

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

	code := generateRandomString(32)
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

	redirectURL := fmt.Sprintf("%s?code=%s&state=%s", redirectURI, code, state)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (m *MockOIDCServer) handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	grantType := r.Form.Get("grant_type")
	var clientID, clientSecret string

	if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Basic ") {
		basicAuth := strings.TrimPrefix(authHeader, "Basic ")
		decoded, err := base64.StdEncoding.DecodeString(basicAuth)
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				clientID = parts[0]
				clientSecret = parts[1]
			}
		}
	}

	if clientID == "" {
		clientID = r.Form.Get("client_id")
		clientSecret = r.Form.Get("client_secret")
	}

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

	m.mu.Lock()
	authCode.Used = true
	m.mu.Unlock()

	accessToken := generateRandomString(32)
	refreshToken := generateRandomString(32)
	idToken := m.generateIDToken("test-user", []string{"admin", "user"}, authCode.Scopes)

	m.mu.Lock()
	m.tokens[accessToken] = &mockToken{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		IssuedAt:     time.Now(),
		Subject:      "test-user",
		Roles:        []string{"admin", "user"},
	}
	m.refreshTokens[refreshToken] = &mockRefreshToken{
		Token:     refreshToken,
		ClientID:  authCode.ClientID,
		Subject:   "test-user",
		Scopes:    authCode.Scopes,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	m.mu.Unlock()

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

func (m *MockOIDCServer) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	refreshTokenStr := r.Form.Get("refresh_token")

	m.mu.Lock()
	refreshToken, exists := m.refreshTokens[refreshTokenStr]
	m.mu.Unlock()

	if !exists || time.Now().After(refreshToken.ExpiresAt) {
		http.Error(w, "invalid or expired refresh token", http.StatusBadRequest)
		return
	}

	accessToken := generateRandomString(32)
	newRefreshToken := generateRandomString(32)
	idToken := m.generateIDToken(refreshToken.Subject, []string{"admin", "user"}, refreshToken.Scopes)

	m.mu.Lock()
	m.tokens[accessToken] = &mockToken{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    3600,
		IssuedAt:     time.Now(),
		Subject:      refreshToken.Subject,
		Roles:        []string{"admin", "user"},
	}
	delete(m.refreshTokens, refreshTokenStr)
	m.refreshTokens[newRefreshToken] = &mockRefreshToken{
		Token:     newRefreshToken,
		ClientID:  refreshToken.ClientID,
		Subject:   refreshToken.Subject,
		Scopes:    refreshToken.Scopes,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	m.mu.Unlock()

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

func (m *MockOIDCServer) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "missing or invalid authorization header", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	m.mu.Lock()
	mockTok, exists := m.tokens[token]
	m.mu.Unlock()

	if !exists {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	userInfo := map[string]interface{}{
		"sub":   mockTok.Subject,
		"name":  "Test User",
		"email": "test@example.com",
		"roles": mockTok.Roles,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInfo)
}

func (m *MockOIDCServer) handleIntrospect(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	token := r.Form.Get("token")

	m.mu.Lock()
	mockTok, exists := m.tokens[token]
	m.mu.Unlock()

	if !exists {
		response := map[string]interface{}{"active": false}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	active := time.Since(mockTok.IssuedAt) < time.Duration(mockTok.ExpiresIn)*time.Second

	response := map[string]interface{}{
		"active":     active,
		"sub":        mockTok.Subject,
		"client_id":  m.clientID,
		"token_type": "Bearer",
		"exp":        mockTok.IssuedAt.Add(time.Duration(mockTok.ExpiresIn) * time.Second).Unix(),
		"iat":        mockTok.IssuedAt.Unix(),
		"scope":      "openid profile email",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *MockOIDCServer) handleRevoke(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	token := r.Form.Get("token")

	m.mu.Lock()
	delete(m.tokens, token)
	delete(m.refreshTokens, token)
	m.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

func (m *MockOIDCServer) generateIDToken(subject string, roles []string, scopes []string) string {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   m.issuer,
		"sub":   subject,
		"aud":   m.clientID,
		"exp":   now.Add(1 * time.Hour).Unix(),
		"iat":   now.Unix(),
		"nonce": "test-nonce",
		"roles": roles,
	}

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

// GenerateAccessToken generates a valid access token for testing.
func (m *MockOIDCServer) GenerateAccessToken(subject string, roles []string) string {
	accessToken := generateRandomString(32)
	idToken := m.generateIDToken(subject, roles, []string{"openid", "profile", "email"})

	m.mu.Lock()
	m.tokens[accessToken] = &mockToken{
		AccessToken: accessToken,
		IDToken:     idToken,
		ExpiresIn:   3600,
		IssuedAt:    time.Now(),
		Subject:     subject,
		Roles:       roles,
	}
	m.mu.Unlock()

	return accessToken
}

func generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.RawURLEncoding.EncodeToString(bytes)[:length]
}

// =============================================================================
// Test Infrastructure: Certificate Generation
// =============================================================================

// CertificateBundle holds certificates and keys for mTLS testing.
type CertificateBundle struct {
	CACert     *x509.Certificate
	CAKey      *rsa.PrivateKey
	CACertPEM  []byte
	CAKeyPEM   []byte
	ServerCert *x509.Certificate
	ServerKey  *rsa.PrivateKey
	ClientCert *x509.Certificate
	ClientKey  *rsa.PrivateKey
	TempDir    string
}

// GenerateCertificateBundle generates a complete certificate bundle for mTLS testing.
func GenerateCertificateBundle(t *testing.T) *CertificateBundle {
	tempDir := t.TempDir()
	bundle := &CertificateBundle{TempDir: tempDir}

	// Generate CA
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate CA key: %v", err)
	}
	bundle.CAKey = caKey

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA", Organization: []string{"Test Org"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("Failed to create CA certificate: %v", err)
	}

	bundle.CACert, _ = x509.ParseCertificate(caCertDER)
	bundle.CACertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	bundle.CAKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caKey),
	})

	// Generate Server Certificate
	serverKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	bundle.ServerKey = serverKey

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost", Organization: []string{"Test Org"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, bundle.CACert, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("Failed to create server certificate: %v", err)
	}
	bundle.ServerCert, _ = x509.ParseCertificate(serverCertDER)

	// Generate Client Certificate
	clientKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	bundle.ClientKey = clientKey

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "test-client", Organization: []string{"Test Org"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, bundle.CACert, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("Failed to create client certificate: %v", err)
	}
	bundle.ClientCert, _ = x509.ParseCertificate(clientCertDER)

	// Write certificates to files
	writeCertToFile(t, filepath.Join(tempDir, "ca.crt"), caCertDER)
	writeKeyToFile(t, filepath.Join(tempDir, "ca.key"), caKey)
	writeCertToFile(t, filepath.Join(tempDir, "server.crt"), serverCertDER)
	writeKeyToFile(t, filepath.Join(tempDir, "server.key"), serverKey)
	writeCertToFile(t, filepath.Join(tempDir, "client.crt"), clientCertDER)
	writeKeyToFile(t, filepath.Join(tempDir, "client.key"), clientKey)

	return bundle
}

func writeCertToFile(t *testing.T, path string, certDER []byte) {
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err := os.WriteFile(path, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write certificate to %s: %v", path, err)
	}
}

func writeKeyToFile(t *testing.T, path string, key *rsa.PrivateKey) {
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err := os.WriteFile(path, keyPEM, 0600); err != nil {
		t.Fatalf("Failed to write key to %s: %v", path, err)
	}
}

// =============================================================================
// Integration Tests: Security Context
// =============================================================================

func TestSecurityContextIntegration(t *testing.T) {
	tests := []struct {
		name      string
		context   *security.SecurityContext
		checkRole string
		checkPerm string
		wantValid bool
		wantRole  bool
		wantPerm  bool
	}{
		{
			name: "valid admin context",
			context: &security.SecurityContext{
				UserID:      "user-123",
				Username:    "admin",
				Roles:       []string{"admin", "user"},
				Permissions: []string{"read", "write", "delete"},
				ExpiresAt:   time.Now().Add(1 * time.Hour),
			},
			checkRole: "admin",
			checkPerm: "delete",
			wantValid: true,
			wantRole:  true,
			wantPerm:  true,
		},
		{
			name: "valid user context without admin role",
			context: &security.SecurityContext{
				UserID:      "user-456",
				Username:    "regular",
				Roles:       []string{"user"},
				Permissions: []string{"read"},
				ExpiresAt:   time.Now().Add(1 * time.Hour),
			},
			checkRole: "admin",
			checkPerm: "delete",
			wantValid: true,
			wantRole:  false,
			wantPerm:  false,
		},
		{
			name: "expired context",
			context: &security.SecurityContext{
				UserID:    "user-789",
				Username:  "expired",
				Roles:     []string{"admin"},
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			checkRole: "admin",
			checkPerm: "read",
			wantValid: false,
			wantRole:  true,
			wantPerm:  false,
		},
		{
			name:      "nil context",
			context:   nil,
			checkRole: "admin",
			checkPerm: "read",
			wantValid: false,
			wantRole:  false,
			wantPerm:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.context.IsValid(); got != tt.wantValid {
				t.Errorf("IsValid() = %v, want %v", got, tt.wantValid)
			}
			if got := tt.context.HasRole(tt.checkRole); got != tt.wantRole {
				t.Errorf("HasRole(%s) = %v, want %v", tt.checkRole, got, tt.wantRole)
			}
			if got := tt.context.HasPermission(tt.checkPerm); got != tt.wantPerm {
				t.Errorf("HasPermission(%s) = %v, want %v", tt.checkPerm, got, tt.wantPerm)
			}
		})
	}
}

func TestSecurityContextMultipleRolesAndPermissions(t *testing.T) {
	ctx := &security.SecurityContext{
		UserID:      "user-multi",
		Username:    "multi-role-user",
		Roles:       []string{"admin", "editor", "viewer"},
		Permissions: []string{"read", "write", "delete", "admin"},
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	// Test HasAnyRole
	if !ctx.HasAnyRole("admin", "superuser") {
		t.Error("HasAnyRole should return true when user has one of the roles")
	}
	if ctx.HasAnyRole("superuser", "root") {
		t.Error("HasAnyRole should return false when user has none of the roles")
	}

	// Test HasAllRoles
	if !ctx.HasAllRoles("admin", "editor") {
		t.Error("HasAllRoles should return true when user has all roles")
	}
	if ctx.HasAllRoles("admin", "superuser") {
		t.Error("HasAllRoles should return false when user is missing a role")
	}

	// Test HasAnyPermission
	if !ctx.HasAnyPermission("delete", "superadmin") {
		t.Error("HasAnyPermission should return true when user has one of the permissions")
	}
	if ctx.HasAnyPermission("superadmin", "root") {
		t.Error("HasAnyPermission should return false when user has none of the permissions")
	}

	// Test HasAllPermissions
	if !ctx.HasAllPermissions("read", "write") {
		t.Error("HasAllPermissions should return true when user has all permissions")
	}
	if ctx.HasAllPermissions("read", "superadmin") {
		t.Error("HasAllPermissions should return false when user is missing a permission")
	}
}

// =============================================================================
// Integration Tests: mTLS Handshake
// =============================================================================

func TestMTLSHandshakeIntegration(t *testing.T) {
	bundle := GenerateCertificateBundle(t)

	// Create server TLS config
	serverTLSConfig := &tlspkg.TLSConfig{
		Enabled:    true,
		CertFile:   filepath.Join(bundle.TempDir, "server.crt"),
		KeyFile:    filepath.Join(bundle.TempDir, "server.key"),
		CAFiles:    []string{filepath.Join(bundle.TempDir, "ca.crt")},
		ClientAuth: "require_and_verify",
		MinVersion: "TLS1.2",
	}

	serverTLS, err := tlspkg.NewTLSConfig(serverTLSConfig)
	if err != nil {
		t.Fatalf("Failed to create server TLS config: %v", err)
	}

	// Create test server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			cert := r.TLS.PeerCertificates[0]
			w.Header().Set("X-Client-CN", cert.Subject.CommonName)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := httptest.NewUnstartedServer(handler)
	server.TLS = serverTLS
	server.StartTLS()
	defer server.Close()

	tests := []struct {
		name       string
		useCert    bool
		wantStatus int
		wantCN     string
	}{
		{
			name:       "valid client certificate",
			useCert:    true,
			wantStatus: http.StatusOK,
			wantCN:     "test-client",
		},
		{
			name:       "no client certificate",
			useCert:    false,
			wantStatus: 0, // Connection should fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var clientTLS *tls.Config

			if tt.useCert {
				clientCert, err := tls.LoadX509KeyPair(
					filepath.Join(bundle.TempDir, "client.crt"),
					filepath.Join(bundle.TempDir, "client.key"),
				)
				if err != nil {
					t.Fatalf("Failed to load client certificate: %v", err)
				}

				caPool := x509.NewCertPool()
				caPool.AddCert(bundle.CACert)

				clientTLS = &tls.Config{
					Certificates: []tls.Certificate{clientCert},
					RootCAs:      caPool,
				}
			} else {
				caPool := x509.NewCertPool()
				caPool.AddCert(bundle.CACert)

				clientTLS = &tls.Config{
					RootCAs: caPool,
				}
			}

			client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: clientTLS,
				},
			}

			resp, err := client.Get(server.URL)
			if tt.useCert {
				if err != nil {
					t.Fatalf("Request failed: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != tt.wantStatus {
					t.Errorf("Status = %d, want %d", resp.StatusCode, tt.wantStatus)
				}
				if cn := resp.Header.Get("X-Client-CN"); cn != tt.wantCN {
					t.Errorf("Client CN = %s, want %s", cn, tt.wantCN)
				}
			} else {
				if err == nil {
					t.Error("Expected connection to fail without client certificate")
				}
			}
		})
	}
}

func TestMTLSCertificateExtraction(t *testing.T) {
	bundle := GenerateCertificateBundle(t)

	info := tlspkg.ExtractCertificateInfo(bundle.ClientCert)
	if info == nil {
		t.Fatal("ExtractCertificateInfo returned nil")
	}

	if info.CommonName != "test-client" {
		t.Errorf("CommonName = %s, want test-client", info.CommonName)
	}

	if len(info.Organization) == 0 || info.Organization[0] != "Test Org" {
		t.Errorf("Organization = %v, want [Test Org]", info.Organization)
	}
}

// =============================================================================
// Integration Tests: OPA Policy with Middleware
// =============================================================================

func TestOPAMiddlewareIntegration(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create RBAC policy
	rbacPolicy := `
package authz

import rego.v1

default allow := false

allow if {
    input.user.roles[_] == "admin"
}

allow if {
    input.request.method == "GET"
    input.user.roles[_] == "viewer"
}

allow if {
    input.request.method in ["GET", "POST", "PUT"]
    input.user.roles[_] == "editor"
}
`

	if err := os.WriteFile(filepath.Join(tempDir, "authz.rego"), []byte(rbacPolicy), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	// Create OPA client
	config := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir,
		},
		DefaultDecisionPath: "authz/allow",
	}

	client, err := opa.NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create OPA client: %v", err)
	}
	defer client.Close(ctx)

	// Create Gin router with OPA middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Custom input builder for testing
	inputBuilder := func(c *gin.Context) (*opa.PolicyInput, error) {
		roles := c.GetHeader("X-User-Roles")
		userID := c.GetHeader("X-User-ID")
		return &opa.PolicyInput{
			User: opa.UserInfo{
				ID:    userID,
				Roles: strings.Split(roles, ","),
			},
			Request: opa.RequestInfo{
				Method: c.Request.Method,
				Path:   c.Request.URL.Path,
			},
		}, nil
	}

	router.Use(middleware.OPAMiddleware(client,
		middleware.WithInputBuilder(inputBuilder),
		middleware.WithWhiteList([]string{"/health"}),
	))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/api/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "test"})
	})

	router.POST("/api/data", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"created": true})
	})

	router.DELETE("/api/data/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"deleted": c.Param("id")})
	})

	tests := []struct {
		name       string
		method     string
		path       string
		userID     string
		roles      string
		wantStatus int
	}{
		{
			name:       "health endpoint bypasses auth",
			method:     "GET",
			path:       "/health",
			userID:     "",
			roles:      "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "admin can access GET",
			method:     "GET",
			path:       "/api/data",
			userID:     "admin-user",
			roles:      "admin",
			wantStatus: http.StatusOK,
		},
		{
			name:       "admin can DELETE",
			method:     "DELETE",
			path:       "/api/data/123",
			userID:     "admin-user",
			roles:      "admin",
			wantStatus: http.StatusOK,
		},
		{
			name:       "viewer can GET",
			method:     "GET",
			path:       "/api/data",
			userID:     "viewer-user",
			roles:      "viewer",
			wantStatus: http.StatusOK,
		},
		{
			name:       "viewer cannot POST",
			method:     "POST",
			path:       "/api/data",
			userID:     "viewer-user",
			roles:      "viewer",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "editor can POST",
			method:     "POST",
			path:       "/api/data",
			userID:     "editor-user",
			roles:      "editor",
			wantStatus: http.StatusCreated,
		},
		{
			name:       "editor cannot DELETE",
			method:     "DELETE",
			path:       "/api/data/123",
			userID:     "editor-user",
			roles:      "editor",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no role denied",
			method:     "GET",
			path:       "/api/data",
			userID:     "no-role-user",
			roles:      "",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			if tt.roles != "" {
				req.Header.Set("X-User-Roles", tt.roles)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d, body = %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

// =============================================================================
// Integration Tests: Cross-Component Security Stack
// =============================================================================

func TestFullSecurityStackIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full security stack test in short mode")
	}

	ctx := context.Background()

	// Setup Mock OIDC Server
	oidcServer, err := NewMockOIDCServer()
	if err != nil {
		t.Fatalf("Failed to create mock OIDC server: %v", err)
	}
	defer oidcServer.Close()

	// Setup OPA Policy
	tempDir := t.TempDir()
	rbacPolicy := `
package authz

import rego.v1

default allow := false

allow if {
    input.user.roles[_] == "admin"
}

allow if {
    input.request.method == "GET"
    input.user.attributes.authenticated == true
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "authz.rego"), []byte(rbacPolicy), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	opaClient, err := opa.NewClient(ctx, &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir,
		},
		DefaultDecisionPath: "authz/allow",
	})
	if err != nil {
		t.Fatalf("Failed to create OPA client: %v", err)
	}
	defer opaClient.Close(ctx)

	// Setup Gin router with full security stack
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Token validation middleware (simulated)
	tokenValidator := func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Set("authenticated", false)
			c.Set("roles", []string{})
			c.Next()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate token against mock server
		req, _ := http.NewRequest("GET", oidcServer.GetIssuerURL()+"/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			c.Set("authenticated", false)
			c.Set("roles", []string{})
			c.Next()
			return
		}
		defer resp.Body.Close()

		var userInfo map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&userInfo)

		c.Set("authenticated", true)
		c.Set("user_id", userInfo["sub"])
		if roles, ok := userInfo["roles"].([]interface{}); ok {
			roleStrings := make([]string, len(roles))
			for i, r := range roles {
				roleStrings[i] = r.(string)
			}
			c.Set("roles", roleStrings)
		}
		c.Next()
	}

	// OPA middleware with custom input builder
	opaInputBuilder := func(c *gin.Context) (*opa.PolicyInput, error) {
		authenticated, _ := c.Get("authenticated")
		roles, _ := c.Get("roles")
		userID, _ := c.Get("user_id")

		roleStrings := []string{}
		if r, ok := roles.([]string); ok {
			roleStrings = r
		}

		input := &opa.PolicyInput{
			User: opa.UserInfo{
				ID:    fmt.Sprintf("%v", userID),
				Roles: roleStrings,
			},
			Request: opa.RequestInfo{
				Method: c.Request.Method,
				Path:   c.Request.URL.Path,
			},
		}
		// Store authenticated status in user attributes
		if input.User.Attributes == nil {
			input.User.Attributes = make(map[string]interface{})
		}
		input.User.Attributes["authenticated"] = authenticated == true
		return input, nil
	}

	router.Use(tokenValidator)
	router.Use(middleware.OPAMiddleware(opaClient,
		middleware.WithInputBuilder(opaInputBuilder),
		middleware.WithWhiteList([]string{"/health", "/public"}),
	))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	router.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "public endpoint"})
	})

	router.GET("/api/protected", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID, "message": "protected data"})
	})

	router.DELETE("/api/admin-only", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin action completed"})
	})

	// Generate tokens
	adminToken := oidcServer.GenerateAccessToken("admin-user", []string{"admin", "user"})
	userToken := oidcServer.GenerateAccessToken("regular-user", []string{"user"})

	tests := []struct {
		name       string
		method     string
		path       string
		token      string
		wantStatus int
	}{
		{
			name:       "health endpoint public",
			method:     "GET",
			path:       "/health",
			token:      "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "public endpoint accessible",
			method:     "GET",
			path:       "/public",
			token:      "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "protected GET with valid user token",
			method:     "GET",
			path:       "/api/protected",
			token:      userToken,
			wantStatus: http.StatusOK,
		},
		{
			name:       "protected GET without token",
			method:     "GET",
			path:       "/api/protected",
			token:      "",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin DELETE with admin token",
			method:     "DELETE",
			path:       "/api/admin-only",
			token:      adminToken,
			wantStatus: http.StatusOK,
		},
		{
			name:       "admin DELETE with user token",
			method:     "DELETE",
			path:       "/api/admin-only",
			token:      userToken,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "protected with invalid token",
			method:     "GET",
			path:       "/api/protected",
			token:      "invalid-token",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d, body = %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkSecurityContextValidation(b *testing.B) {
	ctx := &security.SecurityContext{
		UserID:      "user-123",
		Username:    "testuser",
		Roles:       []string{"admin", "user", "editor"},
		Permissions: []string{"read", "write", "delete"},
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.IsValid()
		ctx.HasRole("admin")
		ctx.HasPermission("write")
	}
}

func BenchmarkOPAPolicyEvaluation(b *testing.B) {
	ctx := context.Background()
	tempDir := b.TempDir()

	policy := `
package authz
import rego.v1
default allow := false
allow if { input.user.roles[_] == "admin" }
`
	os.WriteFile(filepath.Join(tempDir, "authz.rego"), []byte(policy), 0644)

	client, err := opa.NewClient(ctx, &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir,
		},
		DefaultDecisionPath: "authz/allow",
	})
	if err != nil {
		b.Fatalf("Failed to create OPA client: %v", err)
	}
	defer client.Close(ctx)

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"roles": []string{"admin"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Evaluate(ctx, "", input)
		if err != nil {
			b.Fatalf("Evaluation failed: %v", err)
		}
	}
}

func BenchmarkMTLSHandshake(b *testing.B) {
	// Generate certificates
	caKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caCertDER, _ := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	caCert, _ := x509.ParseCertificate(caCertDER)

	serverKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	serverCertDER, _ := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)

	clientKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "client"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientCertDER, _ := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)

	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{serverCertDER},
			PrivateKey:  serverKey,
		}},
		ClientCAs:  caPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.TLS = serverTLS
	server.StartTLS()
	defer server.Close()

	clientTLS := &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{clientCertDER},
			PrivateKey:  clientKey,
		}},
		RootCAs: caPool,
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: clientTLS,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()
	}
}
