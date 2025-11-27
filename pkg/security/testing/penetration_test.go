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

package testing

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// PenetrationTestSuite provides penetration testing capabilities.
type PenetrationTestSuite struct {
	t         *testing.T
	router    *gin.Engine
	jwtSecret string
	config    *PenTestConfig
}

// PenTestConfig holds configuration for penetration tests.
type PenTestConfig struct {
	// TargetEndpoints are the endpoints to test
	TargetEndpoints []EndpointConfig
	// JWTSecret is the secret used for JWT signing
	JWTSecret string
	// AdminRole is the role name for admin users
	AdminRole string
	// UserRole is the role name for regular users
	UserRole string
	// AuthHeader is the header name for authentication
	AuthHeader string
}

// DefaultPenTestConfig returns default penetration test configuration.
func DefaultPenTestConfig() *PenTestConfig {
	return &PenTestConfig{
		JWTSecret:  "test-secret-key",
		AdminRole:  "admin",
		UserRole:   "user",
		AuthHeader: "Authorization",
	}
}

// NewPenetrationTestSuite creates a new penetration test suite.
func NewPenetrationTestSuite(t *testing.T, router *gin.Engine, config *PenTestConfig) *PenetrationTestSuite {
	if config == nil {
		config = DefaultPenTestConfig()
	}
	return &PenetrationTestSuite{
		t:         t,
		router:    router,
		jwtSecret: config.JWTSecret,
		config:    config,
	}
}

// ============================================================================
// Authentication Bypass Tests
// ============================================================================

// TestAuthenticationBypassVectors tests various authentication bypass techniques.
func (p *PenetrationTestSuite) TestAuthenticationBypassVectors() {
	p.t.Run("AuthenticationBypass", func(t *testing.T) {
		// Test missing authentication
		t.Run("MissingAuthentication", func(t *testing.T) {
			p.testMissingAuth(t)
		})

		// Test empty token
		t.Run("EmptyToken", func(t *testing.T) {
			p.testEmptyToken(t)
		})

		// Test malformed tokens
		t.Run("MalformedTokens", func(t *testing.T) {
			p.testMalformedTokens(t)
		})

		// Test null/undefined tokens
		t.Run("NullUndefinedTokens", func(t *testing.T) {
			p.testNullUndefinedTokens(t)
		})

		// Test case sensitivity
		t.Run("CaseSensitivity", func(t *testing.T) {
			p.testCaseSensitivity(t)
		})

		// Test whitespace manipulation
		t.Run("WhitespaceManipulation", func(t *testing.T) {
			p.testWhitespaceManipulation(t)
		})

		// Test header injection
		t.Run("HeaderInjection", func(t *testing.T) {
			p.testHeaderInjection(t)
		})
	})
}

func (p *PenetrationTestSuite) testMissingAuth(t *testing.T) {
	for _, ep := range p.config.TargetEndpoints {
		if !ep.RequiresAuth {
			continue
		}

		resp := p.makeRequest(ep.Method, ep.Path, nil, nil)
		if resp.Code == http.StatusOK {
			t.Errorf("Endpoint %s %s should require authentication", ep.Method, ep.Path)
		}
	}
}

func (p *PenetrationTestSuite) testEmptyToken(t *testing.T) {
	emptyTokens := []string{
		"Bearer ",
		"Bearer",
		"Bearer  ",
		"Bearer \t",
		"Bearer \n",
	}

	for _, token := range emptyTokens {
		resp := p.makeRequest("GET", "/api/protected", map[string]string{
			p.config.AuthHeader: token,
		}, nil)

		if resp.Code == http.StatusOK {
			t.Errorf("Empty token '%s' should be rejected", token)
		}
	}
}

func (p *PenetrationTestSuite) testMalformedTokens(t *testing.T) {
	malformedTokens := []string{
		".",
		"..",
		"...",
		"a.b",
		"a.b.c.d",
		"Bearer.invalid.token",
		base64.StdEncoding.EncodeToString([]byte("not-a-jwt")),
		"eyJ" + strings.Repeat("A", 1000), // Truncated
		"eyJhbGciOiJIUzI1NiJ9." + strings.Repeat("A", 10000) + ".sig",
	}

	for _, token := range malformedTokens {
		resp := p.makeRequest("GET", "/api/protected", map[string]string{
			p.config.AuthHeader: "Bearer " + token,
		}, nil)

		if resp.Code == http.StatusOK {
			t.Errorf("Malformed token should be rejected: %s", truncate(token, 50))
		}
	}
}

func (p *PenetrationTestSuite) testNullUndefinedTokens(t *testing.T) {
	nullTokens := []string{
		"Bearer null",
		"Bearer undefined",
		"Bearer NaN",
		"Bearer [object Object]",
		"Bearer true",
		"Bearer false",
		"Bearer 0",
		"Bearer -1",
	}

	for _, token := range nullTokens {
		resp := p.makeRequest("GET", "/api/protected", map[string]string{
			p.config.AuthHeader: token,
		}, nil)

		if resp.Code == http.StatusOK {
			t.Errorf("Null/undefined token should be rejected: %s", token)
		}
	}
}

func (p *PenetrationTestSuite) testCaseSensitivity(t *testing.T) {
	validToken := p.createValidToken()

	// Test header name case variations
	headerVariations := []string{
		"authorization",
		"AUTHORIZATION",
		"Authorization",
		"AuThOrIzAtIoN",
	}

	for _, header := range headerVariations {
		resp := p.makeRequest("GET", "/api/protected", map[string]string{
			header: "Bearer " + validToken,
		}, nil)

		// HTTP headers should be case-insensitive per RFC 7230
		// but the auth should still work
		t.Logf("Header %s: status %d", header, resp.Code)
	}

	// Test scheme case variations
	schemeVariations := []string{
		"bearer",
		"BEARER",
		"Bearer",
		"BeArEr",
	}

	for _, scheme := range schemeVariations {
		resp := p.makeRequest("GET", "/api/protected", map[string]string{
			p.config.AuthHeader: scheme + " " + validToken,
		}, nil)

		t.Logf("Scheme %s: status %d", scheme, resp.Code)
	}
}

func (p *PenetrationTestSuite) testWhitespaceManipulation(t *testing.T) {
	validToken := p.createValidToken()

	whitespaceVariations := []string{
		"Bearer " + validToken,
		"Bearer  " + validToken,
		"Bearer\t" + validToken,
		"Bearer \t" + validToken,
		" Bearer " + validToken,
		"Bearer " + validToken + " ",
		"Bearer " + validToken + "\r\n",
	}

	for _, auth := range whitespaceVariations {
		resp := p.makeRequest("GET", "/api/protected", map[string]string{
			p.config.AuthHeader: auth,
		}, nil)

		t.Logf("Whitespace variation: status %d", resp.Code)
	}
}

func (p *PenetrationTestSuite) testHeaderInjection(t *testing.T) {
	validToken := p.createValidToken()

	// Test CRLF injection
	injectionAttempts := []string{
		"Bearer " + validToken + "\r\nX-Injected: header",
		"Bearer " + validToken + "\nX-Injected: header",
		"Bearer " + validToken + "\rX-Injected: header",
		"Bearer " + validToken + "\x00X-Injected: header",
	}

	for _, auth := range injectionAttempts {
		resp := p.makeRequest("GET", "/api/protected", map[string]string{
			p.config.AuthHeader: auth,
		}, nil)

		// Should either reject or sanitize
		if resp.Header().Get("X-Injected") != "" {
			t.Error("Header injection succeeded")
		}
	}
}

// ============================================================================
// Authorization Bypass Tests
// ============================================================================

// TestAuthorizationBypassVectors tests various authorization bypass techniques.
func (p *PenetrationTestSuite) TestAuthorizationBypassVectors() {
	p.t.Run("AuthorizationBypass", func(t *testing.T) {
		// Test privilege escalation
		t.Run("PrivilegeEscalation", func(t *testing.T) {
			p.testPrivilegeEscalation(t)
		})

		// Test path traversal
		t.Run("PathTraversal", func(t *testing.T) {
			p.testPathTraversal(t)
		})

		// Test HTTP method tampering
		t.Run("HTTPMethodTampering", func(t *testing.T) {
			p.testHTTPMethodTampering(t)
		})

		// Test parameter pollution
		t.Run("ParameterPollution", func(t *testing.T) {
			p.testParameterPollution(t)
		})

		// Test IDOR
		t.Run("IDOR", func(t *testing.T) {
			p.testIDOR(t)
		})
	})
}

func (p *PenetrationTestSuite) testPrivilegeEscalation(t *testing.T) {
	// Create user token
	userToken := p.createTokenWithRole(p.config.UserRole)

	// Try to access admin endpoints
	adminEndpoints := []string{
		"/api/admin",
		"/api/admin/users",
		"/api/admin/settings",
		"/api/users/admin",
		"/admin",
	}

	for _, endpoint := range adminEndpoints {
		resp := p.makeRequest("GET", endpoint, map[string]string{
			p.config.AuthHeader: "Bearer " + userToken,
		}, nil)

		if resp.Code == http.StatusOK {
			t.Errorf("User should not access admin endpoint: %s", endpoint)
		}
	}
}

func (p *PenetrationTestSuite) testPathTraversal(t *testing.T) {
	userToken := p.createTokenWithRole(p.config.UserRole)

	pathTraversalAttempts := []string{
		"/api/../admin",
		"/api/./admin",
		"/api/users/../admin",
		"/api/users/./admin",
		"/api%2f..%2fadmin",
		"/api/..%2fadmin",
		"/api%2f../admin",
		"/api/..;/admin",
		"/api/..%00/admin",
		"/api/..%0d%0a/admin",
		"/api/....//admin",
		"/api/.../admin",
		"/api/users/..\\admin",
	}

	for _, path := range pathTraversalAttempts {
		resp := p.makeRequest("GET", path, map[string]string{
			p.config.AuthHeader: "Bearer " + userToken,
		}, nil)

		if resp.Code == http.StatusOK {
			t.Errorf("Path traversal succeeded: %s", path)
		}
	}
}

func (p *PenetrationTestSuite) testHTTPMethodTampering(t *testing.T) {
	userToken := p.createTokenWithRole(p.config.UserRole)

	// Test method override headers
	overrideHeaders := []struct {
		header string
		method string
	}{
		{"X-HTTP-Method-Override", "DELETE"},
		{"X-HTTP-Method", "DELETE"},
		{"X-Method-Override", "DELETE"},
		{"_method", "DELETE"},
	}

	for _, oh := range overrideHeaders {
		resp := p.makeRequest("GET", "/api/admin/users/1", map[string]string{
			p.config.AuthHeader: "Bearer " + userToken,
			oh.header:           oh.method,
		}, nil)

		if resp.Code == http.StatusOK {
			t.Errorf("HTTP method override succeeded with %s", oh.header)
		}
	}

	// Test unusual HTTP methods
	unusualMethods := []string{
		"TRACE",
		"TRACK",
		"CONNECT",
		"PROPFIND",
		"PROPPATCH",
		"MKCOL",
		"COPY",
		"MOVE",
		"LOCK",
		"UNLOCK",
	}

	for _, method := range unusualMethods {
		resp := p.makeRequestWithMethod(method, "/api/admin", map[string]string{
			p.config.AuthHeader: "Bearer " + userToken,
		}, nil)

		if resp.Code == http.StatusOK {
			t.Errorf("Unusual HTTP method %s should be rejected", method)
		}
	}
}

func (p *PenetrationTestSuite) testParameterPollution(t *testing.T) {
	userToken := p.createTokenWithRole(p.config.UserRole)

	// Test parameter pollution
	pollutionAttempts := []string{
		"/api/users?role=user&role=admin",
		"/api/users?role[]=user&role[]=admin",
		"/api/users?role=user,admin",
		"/api/users?role=user%00admin",
	}

	for _, path := range pollutionAttempts {
		resp := p.makeRequest("GET", path, map[string]string{
			p.config.AuthHeader: "Bearer " + userToken,
		}, nil)

		// Should not grant admin access
		t.Logf("Parameter pollution %s: status %d", path, resp.Code)
	}
}

func (p *PenetrationTestSuite) testIDOR(t *testing.T) {
	// Create two users
	user1Token := p.createTokenWithSubject("user1", p.config.UserRole)
	// user2Token := p.createTokenWithSubject("user2", p.config.UserRole)

	// User1 tries to access User2's resources
	idorAttempts := []struct {
		path   string
		method string
	}{
		{"/api/users/user2", "GET"},
		{"/api/users/user2/profile", "GET"},
		{"/api/users/user2/settings", "GET"},
		{"/api/users/user2", "PUT"},
		{"/api/users/user2", "DELETE"},
	}

	for _, attempt := range idorAttempts {
		resp := p.makeRequest(attempt.method, attempt.path, map[string]string{
			p.config.AuthHeader: "Bearer " + user1Token,
		}, nil)

		if resp.Code == http.StatusOK {
			t.Errorf("IDOR vulnerability: user1 accessed %s %s", attempt.method, attempt.path)
		}
	}
}

// ============================================================================
// Token Forgery Tests
// ============================================================================

// TestTokenForgeryVectors tests various token forgery techniques.
func (p *PenetrationTestSuite) TestTokenForgeryVectors() {
	p.t.Run("TokenForgery", func(t *testing.T) {
		// Test algorithm confusion
		t.Run("AlgorithmConfusion", func(t *testing.T) {
			p.testAlgorithmConfusion(t)
		})

		// Test signature stripping
		t.Run("SignatureStripping", func(t *testing.T) {
			p.testSignatureStripping(t)
		})

		// Test claim manipulation
		t.Run("ClaimManipulation", func(t *testing.T) {
			p.testClaimManipulation(t)
		})

		// Test key confusion
		t.Run("KeyConfusion", func(t *testing.T) {
			p.testKeyConfusion(t)
		})

		// Test token reuse
		t.Run("TokenReuse", func(t *testing.T) {
			p.testTokenReuse(t)
		})
	})
}

func (p *PenetrationTestSuite) testAlgorithmConfusion(t *testing.T) {
	// None algorithm attack
	noneTokens := []string{
		p.createTokenWithAlg("none"),
		p.createTokenWithAlg("None"),
		p.createTokenWithAlg("NONE"),
		p.createTokenWithAlg("nOnE"),
	}

	for _, token := range noneTokens {
		resp := p.makeRequest("GET", "/api/protected", map[string]string{
			p.config.AuthHeader: "Bearer " + token,
		}, nil)

		if resp.Code == http.StatusOK {
			t.Error("None algorithm attack succeeded")
		}
	}

	// Algorithm downgrade attacks
	weakAlgorithms := []string{
		"HS1",
		"MD5",
		"SHA1",
	}

	for _, alg := range weakAlgorithms {
		token := p.createTokenWithAlg(alg)
		resp := p.makeRequest("GET", "/api/protected", map[string]string{
			p.config.AuthHeader: "Bearer " + token,
		}, nil)

		if resp.Code == http.StatusOK {
			t.Errorf("Weak algorithm %s should be rejected", alg)
		}
	}
}

func (p *PenetrationTestSuite) testSignatureStripping(t *testing.T) {
	validToken := p.createValidToken()
	parts := strings.Split(validToken, ".")

	// Strip signature
	strippedTokens := []string{
		parts[0] + "." + parts[1] + ".",
		parts[0] + "." + parts[1],
		parts[0] + "." + parts[1] + ".invalidSignature",
		parts[0] + "." + parts[1] + "." + base64.RawURLEncoding.EncodeToString([]byte("fake")),
	}

	for _, token := range strippedTokens {
		resp := p.makeRequest("GET", "/api/protected", map[string]string{
			p.config.AuthHeader: "Bearer " + token,
		}, nil)

		if resp.Code == http.StatusOK {
			t.Errorf("Token with stripped/invalid signature should be rejected")
		}
	}
}

func (p *PenetrationTestSuite) testClaimManipulation(t *testing.T) {
	validToken := p.createValidToken()
	parts := strings.Split(validToken, ".")

	// Decode payload
	payloadBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]interface{}
	json.Unmarshal(payloadBytes, &claims)

	// Manipulate claims
	manipulations := []map[string]interface{}{
		{"sub": "admin", "roles": []string{"admin"}},
		{"sub": "test", "roles": []string{"admin", "superuser"}},
		{"sub": "test", "exp": time.Now().Add(100 * 365 * 24 * time.Hour).Unix()},
		{"sub": "test", "admin": true},
		{"sub": "test", "is_admin": true},
	}

	for _, manip := range manipulations {
		for k, v := range manip {
			claims[k] = v
		}
		newPayload, _ := json.Marshal(claims)
		newPayloadEncoded := base64.RawURLEncoding.EncodeToString(newPayload)

		// Keep original signature (should fail validation)
		manipulatedToken := parts[0] + "." + newPayloadEncoded + "." + parts[2]

		resp := p.makeRequest("GET", "/api/protected", map[string]string{
			p.config.AuthHeader: "Bearer " + manipulatedToken,
		}, nil)

		if resp.Code == http.StatusOK {
			t.Errorf("Manipulated token should be rejected")
		}
	}
}

func (p *PenetrationTestSuite) testKeyConfusion(t *testing.T) {
	// Try different secrets
	wrongSecrets := []string{
		"",
		"wrong-secret",
		p.jwtSecret + "extra",
		strings.ToUpper(p.jwtSecret),
		strings.ToLower(p.jwtSecret),
	}

	for _, secret := range wrongSecrets {
		token := p.createTokenWithSecret(secret)
		resp := p.makeRequest("GET", "/api/protected", map[string]string{
			p.config.AuthHeader: "Bearer " + token,
		}, nil)

		if resp.Code == http.StatusOK && secret != p.jwtSecret {
			t.Errorf("Token with wrong secret should be rejected")
		}
	}
}

func (p *PenetrationTestSuite) testTokenReuse(t *testing.T) {
	// Test expired token reuse
	expiredToken := p.createExpiredToken()
	resp := p.makeRequest("GET", "/api/protected", map[string]string{
		p.config.AuthHeader: "Bearer " + expiredToken,
	}, nil)

	if resp.Code == http.StatusOK {
		t.Error("Expired token should be rejected")
	}

	// Test not-yet-valid token
	futureToken := p.createFutureToken()
	resp = p.makeRequest("GET", "/api/protected", map[string]string{
		p.config.AuthHeader: "Bearer " + futureToken,
	}, nil)

	if resp.Code == http.StatusOK {
		t.Error("Future token should be rejected")
	}
}

// ============================================================================
// Helper Methods
// ============================================================================

func (p *PenetrationTestSuite) makeRequest(method, path string, headers map[string]string, body []byte) *httptest.ResponseRecorder {
	return p.makeRequestWithMethod(method, path, headers, body)
}

func (p *PenetrationTestSuite) makeRequestWithMethod(method, path string, headers map[string]string, body []byte) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	p.router.ServeHTTP(w, req)
	return w
}

func (p *PenetrationTestSuite) createValidToken() string {
	return p.createTokenWithRole(p.config.UserRole)
}

func (p *PenetrationTestSuite) createTokenWithRole(role string) string {
	claims := jwt.MapClaims{
		"sub":   "test-user",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"roles": []string{role},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(p.jwtSecret))
	return tokenString
}

func (p *PenetrationTestSuite) createTokenWithSubject(subject, role string) string {
	claims := jwt.MapClaims{
		"sub":   subject,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"roles": []string{role},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(p.jwtSecret))
	return tokenString
}

func (p *PenetrationTestSuite) createTokenWithSecret(secret string) string {
	claims := jwt.MapClaims{
		"sub":   "test-user",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"roles": []string{p.config.AdminRole},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func (p *PenetrationTestSuite) createTokenWithAlg(alg string) string {
	header := map[string]string{
		"alg": alg,
		"typ": "JWT",
	}
	headerBytes, _ := json.Marshal(header)
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerBytes)

	payload := map[string]interface{}{
		"sub":   "admin",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"roles": []string{p.config.AdminRole},
	}
	payloadBytes, _ := json.Marshal(payload)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadBytes)

	var signature string
	if alg == "none" || alg == "None" || alg == "NONE" || alg == "nOnE" {
		signature = ""
	} else {
		h := hmac.New(sha256.New, []byte(p.jwtSecret))
		h.Write([]byte(headerEncoded + "." + payloadEncoded))
		signature = base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	}

	return headerEncoded + "." + payloadEncoded + "." + signature
}

func (p *PenetrationTestSuite) createExpiredToken() string {
	claims := jwt.MapClaims{
		"sub":   "test-user",
		"exp":   time.Now().Add(-time.Hour).Unix(),
		"iat":   time.Now().Add(-2 * time.Hour).Unix(),
		"roles": []string{p.config.UserRole},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(p.jwtSecret))
	return tokenString
}

func (p *PenetrationTestSuite) createFutureToken() string {
	claims := jwt.MapClaims{
		"sub":   "test-user",
		"exp":   time.Now().Add(2 * time.Hour).Unix(),
		"iat":   time.Now().Add(time.Hour).Unix(),
		"nbf":   time.Now().Add(time.Hour).Unix(),
		"roles": []string{p.config.UserRole},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(p.jwtSecret))
	return tokenString
}

// ============================================================================
// Additional Attack Vectors
// ============================================================================

// TestJWTKIDInjection tests for JWT KID header injection vulnerabilities.
func (p *PenetrationTestSuite) TestJWTKIDInjection() {
	p.t.Run("JWTKIDInjection", func(t *testing.T) {
		kidPayloads := []string{
			"../../../etc/passwd",
			"/dev/null",
			"| cat /etc/passwd",
			"; cat /etc/passwd",
			"key1' OR '1'='1",
			"key1\" OR \"1\"=\"1",
			"../../../../../../etc/passwd\x00",
			"key1\r\nX-Injected: header",
		}

		for _, kid := range kidPayloads {
			token := p.createTokenWithKID(kid)
			resp := p.makeRequest("GET", "/api/protected", map[string]string{
				p.config.AuthHeader: "Bearer " + token,
			}, nil)

			if resp.Code == http.StatusOK {
				t.Errorf("KID injection payload should be rejected: %s", kid)
			}
		}
	})
}

func (p *PenetrationTestSuite) createTokenWithKID(kid string) string {
	claims := jwt.MapClaims{
		"sub":   "test-user",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"roles": []string{p.config.UserRole},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = kid
	tokenString, _ := token.SignedString([]byte(p.jwtSecret))
	return tokenString
}

// TestJWKUAttack tests for JKU (JWK Set URL) header injection.
func (p *PenetrationTestSuite) TestJWKUAttack() {
	p.t.Run("JWKUAttack", func(t *testing.T) {
		jkuPayloads := []string{
			"https://attacker.com/.well-known/jwks.json",
			"http://localhost:8080/jwks.json",
			"file:///etc/passwd",
			"http://169.254.169.254/latest/meta-data/",
		}

		for _, jku := range jkuPayloads {
			token := p.createTokenWithJKU(jku)
			resp := p.makeRequest("GET", "/api/protected", map[string]string{
				p.config.AuthHeader: "Bearer " + token,
			}, nil)

			if resp.Code == http.StatusOK {
				t.Errorf("JKU injection payload should be rejected: %s", jku)
			}
		}
	})
}

func (p *PenetrationTestSuite) createTokenWithJKU(jku string) string {
	claims := jwt.MapClaims{
		"sub":   "admin",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"roles": []string{p.config.AdminRole},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["jku"] = jku
	tokenString, _ := token.SignedString([]byte(p.jwtSecret))
	return tokenString
}

// TestX5UAttack tests for x5u (X.509 URL) header injection.
func (p *PenetrationTestSuite) TestX5UAttack() {
	p.t.Run("X5UAttack", func(t *testing.T) {
		x5uPayloads := []string{
			"https://attacker.com/cert.pem",
			"http://localhost:8080/cert.pem",
			"file:///etc/ssl/certs/ca-certificates.crt",
		}

		for _, x5u := range x5uPayloads {
			token := p.createTokenWithX5U(x5u)
			resp := p.makeRequest("GET", "/api/protected", map[string]string{
				p.config.AuthHeader: "Bearer " + token,
			}, nil)

			if resp.Code == http.StatusOK {
				t.Errorf("x5u injection payload should be rejected: %s", x5u)
			}
		}
	})
}

func (p *PenetrationTestSuite) createTokenWithX5U(x5u string) string {
	claims := jwt.MapClaims{
		"sub":   "admin",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"roles": []string{p.config.AdminRole},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["x5u"] = x5u
	tokenString, _ := token.SignedString([]byte(p.jwtSecret))
	return tokenString
}

// ============================================================================
// Run All Penetration Tests
// ============================================================================

// RunAllPenetrationTests runs all penetration tests.
func (p *PenetrationTestSuite) RunAllPenetrationTests() {
	p.TestAuthenticationBypassVectors()
	p.TestAuthorizationBypassVectors()
	p.TestTokenForgeryVectors()
	p.TestJWTKIDInjection()
	p.TestJWKUAttack()
	p.TestX5UAttack()
}

// ============================================================================
// Penetration Test Report
// ============================================================================

// PenTestResult represents the result of a penetration test.
type PenTestResult struct {
	TestName    string
	Category    string
	Severity    string
	Passed      bool
	Description string
	Remediation string
}

// PenTestReport represents a penetration test report.
type PenTestReport struct {
	Timestamp   time.Time
	TotalTests  int
	PassedTests int
	FailedTests int
	Results     []PenTestResult
}

// GenerateReport generates a penetration test report.
func (p *PenetrationTestSuite) GenerateReport() *PenTestReport {
	// This would be populated by running tests and collecting results
	return &PenTestReport{
		Timestamp:   time.Now(),
		TotalTests:  0,
		PassedTests: 0,
		FailedTests: 0,
		Results:     []PenTestResult{},
	}
}

// ToJSON converts the report to JSON.
func (r *PenTestReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// ToHTML converts the report to HTML.
func (r *PenTestReport) ToHTML() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Penetration Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .passed { color: green; }
        .failed { color: red; }
        table { border-collapse: collapse; width: 100%%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #4CAF50; color: white; }
        .severity-critical { background-color: #ff0000; color: white; }
        .severity-high { background-color: #ff6600; color: white; }
        .severity-medium { background-color: #ffcc00; }
        .severity-low { background-color: #99ff99; }
    </style>
</head>
<body>
    <h1>Penetration Test Report</h1>
    <p>Generated: %s</p>
    <p>Total Tests: %d | Passed: %d | Failed: %d</p>
    <table>
        <tr>
            <th>Test Name</th>
            <th>Category</th>
            <th>Severity</th>
            <th>Status</th>
            <th>Description</th>
        </tr>
`, r.Timestamp.Format(time.RFC3339), r.TotalTests, r.PassedTests, r.FailedTests))

	for _, result := range r.Results {
		status := "PASSED"
		statusClass := "passed"
		if !result.Passed {
			status = "FAILED"
			statusClass = "failed"
		}

		sb.WriteString(fmt.Sprintf(`        <tr>
            <td>%s</td>
            <td>%s</td>
            <td class="severity-%s">%s</td>
            <td class="%s">%s</td>
            <td>%s</td>
        </tr>
`, result.TestName, result.Category, strings.ToLower(result.Severity), result.Severity, statusClass, status, result.Description))
	}

	sb.WriteString(`    </table>
</body>
</html>`)

	return sb.String()
}
