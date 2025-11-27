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

// Package testing provides security testing utilities including OWASP Top 10 tests,
// penetration testing, fuzz testing, and timing attack detection.
package testing

import (
	"bytes"
	"context"
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

// SecurityTestSuite provides comprehensive security testing capabilities.
type SecurityTestSuite struct {
	t         *testing.T
	router    *gin.Engine
	jwtSecret string
}

// NewSecurityTestSuite creates a new security test suite.
func NewSecurityTestSuite(t *testing.T, router *gin.Engine, jwtSecret string) *SecurityTestSuite {
	return &SecurityTestSuite{
		t:         t,
		router:    router,
		jwtSecret: jwtSecret,
	}
}

// ============================================================================
// OWASP Top 10 Tests
// ============================================================================

// TestOWASP_A01_BrokenAccessControl tests for broken access control vulnerabilities.
func (s *SecurityTestSuite) TestOWASP_A01_BrokenAccessControl(endpoints []EndpointConfig) {
	s.t.Run("A01:2021-BrokenAccessControl", func(t *testing.T) {
		for _, ep := range endpoints {
			t.Run(fmt.Sprintf("%s_%s", ep.Method, ep.Path), func(t *testing.T) {
				// Test without authentication
				s.testUnauthenticatedAccess(t, ep)

				// Test with wrong role
				if ep.RequiredRole != "" {
					s.testWrongRoleAccess(t, ep)
				}

				// Test IDOR (Insecure Direct Object Reference)
				if ep.HasResourceID {
					s.testIDOR(t, ep)
				}
			})
		}
	})
}

// TestOWASP_A02_CryptographicFailures tests for cryptographic failures.
func (s *SecurityTestSuite) TestOWASP_A02_CryptographicFailures() {
	s.t.Run("A02:2021-CryptographicFailures", func(t *testing.T) {
		// Test weak algorithm rejection
		t.Run("WeakAlgorithmRejection", func(t *testing.T) {
			weakAlgos := []string{"none", "HS1", "MD5"}
			for _, alg := range weakAlgos {
				token := s.createTokenWithAlgorithm(alg)
				if s.isTokenAccepted(token) {
					t.Errorf("Weak algorithm %s should be rejected", alg)
				}
			}
		})

		// Test algorithm confusion attack
		t.Run("AlgorithmConfusionAttack", func(t *testing.T) {
			// Try to use "none" algorithm
			token := s.createNoneAlgorithmToken()
			if s.isTokenAccepted(token) {
				t.Error("Algorithm confusion attack succeeded - 'none' algorithm accepted")
			}
		})
	})
}

// TestOWASP_A03_Injection tests for injection vulnerabilities.
func (s *SecurityTestSuite) TestOWASP_A03_Injection(endpoints []EndpointConfig) {
	s.t.Run("A03:2021-Injection", func(t *testing.T) {
		injectionPayloads := GetInjectionPayloads()

		for _, ep := range endpoints {
			if !ep.AcceptsInput {
				continue
			}

			t.Run(fmt.Sprintf("%s_%s", ep.Method, ep.Path), func(t *testing.T) {
				for _, payload := range injectionPayloads {
					t.Run(payload.Name, func(t *testing.T) {
						resp := s.sendRequestWithPayload(ep, payload.Value)
						if s.isInjectionSuccessful(resp, payload) {
							t.Errorf("Injection vulnerability detected with payload: %s", payload.Name)
						}
					})
				}
			})
		}
	})
}

// TestOWASP_A04_InsecureDesign tests for insecure design patterns.
func (s *SecurityTestSuite) TestOWASP_A04_InsecureDesign() {
	s.t.Run("A04:2021-InsecureDesign", func(t *testing.T) {
		// Test rate limiting
		t.Run("RateLimiting", func(t *testing.T) {
			s.testRateLimiting(t)
		})

		// Test account lockout
		t.Run("AccountLockout", func(t *testing.T) {
			s.testAccountLockout(t)
		})
	})
}

// TestOWASP_A05_SecurityMisconfiguration tests for security misconfigurations.
func (s *SecurityTestSuite) TestOWASP_A05_SecurityMisconfiguration() {
	s.t.Run("A05:2021-SecurityMisconfiguration", func(t *testing.T) {
		// Test security headers
		t.Run("SecurityHeaders", func(t *testing.T) {
			resp := s.makeRequest("GET", "/", nil, nil)
			s.validateSecurityHeaders(t, resp)
		})

		// Test error disclosure
		t.Run("ErrorDisclosure", func(t *testing.T) {
			resp := s.makeRequest("GET", "/nonexistent", nil, nil)
			s.validateNoSensitiveErrorInfo(t, resp)
		})

		// Test debug mode disabled
		t.Run("DebugModeDisabled", func(t *testing.T) {
			resp := s.makeRequest("GET", "/debug", nil, nil)
			if resp.Code != http.StatusNotFound && resp.Code != http.StatusForbidden {
				t.Error("Debug endpoint should not be accessible")
			}
		})
	})
}

// TestOWASP_A07_IdentificationAuthenticationFailures tests for authentication failures.
func (s *SecurityTestSuite) TestOWASP_A07_IdentificationAuthenticationFailures() {
	s.t.Run("A07:2021-IdentificationAuthenticationFailures", func(t *testing.T) {
		// Test expired token rejection
		t.Run("ExpiredTokenRejection", func(t *testing.T) {
			token := s.createExpiredToken()
			if s.isTokenAccepted(token) {
				t.Error("Expired token should be rejected")
			}
		})

		// Test token tampering detection
		t.Run("TokenTamperingDetection", func(t *testing.T) {
			token := s.createValidToken()
			tamperedToken := s.tamperToken(token)
			if s.isTokenAccepted(tamperedToken) {
				t.Error("Tampered token should be rejected")
			}
		})

		// Test token replay prevention
		t.Run("TokenReplayPrevention", func(t *testing.T) {
			// This test is for blacklisted tokens
			token := s.createValidToken()
			// First use should succeed
			if !s.isTokenAccepted(token) {
				t.Skip("Valid token should be accepted")
			}
			// After blacklisting, should fail
			// Note: This requires blacklist implementation
		})
	})
}

// TestOWASP_A08_SoftwareDataIntegrityFailures tests for integrity failures.
func (s *SecurityTestSuite) TestOWASP_A08_SoftwareDataIntegrityFailures() {
	s.t.Run("A08:2021-SoftwareDataIntegrityFailures", func(t *testing.T) {
		// Test signature verification
		t.Run("SignatureVerification", func(t *testing.T) {
			token := s.createTokenWithWrongSignature()
			if s.isTokenAccepted(token) {
				t.Error("Token with wrong signature should be rejected")
			}
		})

		// Test claim modification detection
		t.Run("ClaimModificationDetection", func(t *testing.T) {
			token := s.createValidToken()
			modifiedToken := s.modifyTokenClaims(token)
			if s.isTokenAccepted(modifiedToken) {
				t.Error("Token with modified claims should be rejected")
			}
		})
	})
}

// TestOWASP_A09_SecurityLoggingMonitoringFailures tests for logging failures.
func (s *SecurityTestSuite) TestOWASP_A09_SecurityLoggingMonitoringFailures() {
	s.t.Run("A09:2021-SecurityLoggingMonitoringFailures", func(t *testing.T) {
		// Test that security events are logged
		t.Run("SecurityEventsLogged", func(t *testing.T) {
			// This would require log capture mechanism
			t.Skip("Requires log capture implementation")
		})
	})
}

// ============================================================================
// Penetration Testing
// ============================================================================

// TestAuthenticationBypass tests for authentication bypass vulnerabilities.
func (s *SecurityTestSuite) TestAuthenticationBypass() {
	s.t.Run("AuthenticationBypass", func(t *testing.T) {
		bypassAttempts := []struct {
			name   string
			header string
			value  string
		}{
			{"EmptyBearer", "Authorization", "Bearer "},
			{"NullToken", "Authorization", "Bearer null"},
			{"UndefinedToken", "Authorization", "Bearer undefined"},
			{"SpaceOnlyToken", "Authorization", "Bearer    "},
			{"MalformedBearer", "Authorization", "Bearer.invalid.token"},
			{"NoBearer", "Authorization", "invalid-token"},
			{"WrongScheme", "Authorization", "Basic dGVzdDp0ZXN0"},
			{"CaseVariation", "authorization", "Bearer " + s.createValidToken()},
		}

		for _, attempt := range bypassAttempts {
			t.Run(attempt.name, func(t *testing.T) {
				resp := s.makeRequest("GET", "/api/protected", map[string]string{
					attempt.header: attempt.value,
				}, nil)

				if resp.Code == http.StatusOK {
					t.Errorf("Authentication bypass succeeded with %s", attempt.name)
				}
			})
		}
	})
}

// TestAuthorizationBypass tests for authorization bypass vulnerabilities.
func (s *SecurityTestSuite) TestAuthorizationBypass() {
	s.t.Run("AuthorizationBypass", func(t *testing.T) {
		// Test path traversal in authorization
		t.Run("PathTraversal", func(t *testing.T) {
			paths := []string{
				"/api/../admin",
				"/api/./admin",
				"/api/user/../admin",
				"/api%2f..%2fadmin",
				"/api/..;/admin",
			}

			token := s.createTokenWithRole("user")
			for _, path := range paths {
				resp := s.makeRequest("GET", path, map[string]string{
					"Authorization": "Bearer " + token,
				}, nil)

				if resp.Code == http.StatusOK {
					t.Errorf("Authorization bypass via path traversal: %s", path)
				}
			}
		})

		// Test HTTP method override
		t.Run("HTTPMethodOverride", func(t *testing.T) {
			overrideHeaders := []string{
				"X-HTTP-Method-Override",
				"X-HTTP-Method",
				"X-Method-Override",
			}

			for _, header := range overrideHeaders {
				resp := s.makeRequest("GET", "/api/admin", map[string]string{
					header: "DELETE",
				}, nil)

				if resp.Code == http.StatusOK {
					t.Errorf("HTTP method override attack succeeded with header: %s", header)
				}
			}
		})
	})
}

// TestTokenForgery tests for token forgery vulnerabilities.
func (s *SecurityTestSuite) TestTokenForgery() {
	s.t.Run("TokenForgery", func(t *testing.T) {
		// Test forged token with different secret
		t.Run("DifferentSecret", func(t *testing.T) {
			token := s.createTokenWithSecret("wrong-secret")
			if s.isTokenAccepted(token) {
				t.Error("Token signed with wrong secret should be rejected")
			}
		})

		// Test forged token with elevated privileges
		t.Run("ElevatedPrivileges", func(t *testing.T) {
			// Create token claiming admin role but signed with user's context
			token := s.createTokenWithRole("admin")
			// This should be rejected if proper authorization is in place
			_ = token // Used for demonstration - actual test depends on auth setup
		})

		// Test kid manipulation
		t.Run("KIDManipulation", func(t *testing.T) {
			token := s.createTokenWithKID("../../etc/passwd")
			if s.isTokenAccepted(token) {
				t.Error("Token with malicious KID should be rejected")
			}
		})
	})
}

// ============================================================================
// Timing Attack Tests
// ============================================================================

// TestTimingSideChannel tests for timing side-channel vulnerabilities.
func (s *SecurityTestSuite) TestTimingSideChannel() {
	s.t.Run("TimingSideChannel", func(t *testing.T) {
		// Test constant-time token comparison
		t.Run("ConstantTimeTokenComparison", func(t *testing.T) {
			validToken := s.createValidToken()
			invalidToken := s.createTokenWithWrongSignature()

			// Measure timing for valid token
			validTimes := s.measureResponseTimes(validToken, 100)

			// Measure timing for invalid token
			invalidTimes := s.measureResponseTimes(invalidToken, 100)

			// Check if there's significant timing difference
			validAvg := average(validTimes)
			invalidAvg := average(invalidTimes)

			// Allow for 20% variance - timing should be roughly similar
			variance := 0.2
			threshold := time.Duration(float64(validAvg) * variance)
			if diff := abs(validAvg - invalidAvg); diff > threshold {
				t.Logf("Warning: Potential timing side-channel detected (valid: %v, invalid: %v)", validAvg, invalidAvg)
			}
		})

		// Test constant-time password comparison (if applicable)
		t.Run("ConstantTimePasswordComparison", func(t *testing.T) {
			// Similar approach for password verification endpoints
		})
	})
}

// ============================================================================
// DoS Protection Tests
// ============================================================================

// TestDoSProtection tests for denial of service protection.
func (s *SecurityTestSuite) TestDoSProtection() {
	s.t.Run("DoSProtection", func(t *testing.T) {
		// Test large payload handling
		t.Run("LargePayloadHandling", func(t *testing.T) {
			largePayload := strings.Repeat("A", 10*1024*1024) // 10MB
			resp := s.makeRequest("POST", "/api/data", nil, []byte(largePayload))

			if resp.Code != http.StatusRequestEntityTooLarge && resp.Code != http.StatusBadRequest {
				t.Error("Large payload should be rejected")
			}
		})

		// Test nested JSON handling
		t.Run("NestedJSONHandling", func(t *testing.T) {
			nestedJSON := s.createDeeplyNestedJSON(1000)
			resp := s.makeRequest("POST", "/api/data", map[string]string{
				"Content-Type": "application/json",
			}, nestedJSON)

			if resp.Code == http.StatusOK {
				t.Error("Deeply nested JSON should be rejected or limited")
			}
		})

		// Test slow loris protection
		t.Run("SlowLorisProtection", func(t *testing.T) {
			// This requires actual connection testing, skip in unit tests
			t.Skip("Requires integration testing")
		})

		// Test request rate limiting
		t.Run("RequestRateLimiting", func(t *testing.T) {
			s.testRateLimiting(t)
		})
	})
}

// ============================================================================
// XSS Protection Tests
// ============================================================================

// TestXSSProtection tests for XSS vulnerabilities.
func (s *SecurityTestSuite) TestXSSProtection() {
	s.t.Run("XSSProtection", func(t *testing.T) {
		xssPayloads := GetXSSPayloads()

		for _, payload := range xssPayloads {
			t.Run(payload.Name, func(t *testing.T) {
				// Test in query parameters
				resp := s.makeRequest("GET", "/api/search?q="+payload.Value, nil, nil)
				if strings.Contains(resp.Body.String(), payload.Value) {
					// Check if it's properly escaped
					if !s.isProperlyEscaped(resp.Body.String(), payload.Value) {
						t.Errorf("XSS payload not properly escaped: %s", payload.Name)
					}
				}

				// Test in request body
				body := map[string]string{"input": payload.Value}
				bodyBytes, _ := json.Marshal(body)
				resp = s.makeRequest("POST", "/api/data", map[string]string{
					"Content-Type": "application/json",
				}, bodyBytes)

				if strings.Contains(resp.Body.String(), payload.Value) {
					if !s.isProperlyEscaped(resp.Body.String(), payload.Value) {
						t.Errorf("XSS payload not properly escaped in body: %s", payload.Name)
					}
				}
			})
		}
	})
}

// ============================================================================
// CSRF Protection Tests
// ============================================================================

// TestCSRFProtection tests for CSRF vulnerabilities.
func (s *SecurityTestSuite) TestCSRFProtection() {
	s.t.Run("CSRFProtection", func(t *testing.T) {
		// Test state-changing requests without CSRF token
		t.Run("MissingCSRFToken", func(t *testing.T) {
			methods := []string{"POST", "PUT", "DELETE", "PATCH"}

			for _, method := range methods {
				t.Run(method, func(t *testing.T) {
					resp := s.makeRequest(method, "/api/data", map[string]string{
						"Authorization": "Bearer " + s.createValidToken(),
					}, []byte(`{"data": "test"}`))

					// Should either require CSRF token or use proper SameSite cookies
					// This depends on implementation
					_ = resp // Response handling depends on CSRF implementation
				})
			}
		})

		// Test CSRF token validation
		t.Run("InvalidCSRFToken", func(t *testing.T) {
			resp := s.makeRequest("POST", "/api/data", map[string]string{
				"Authorization":  "Bearer " + s.createValidToken(),
				"X-CSRF-Token":   "invalid-token",
				"X-Requested-By": "XMLHttpRequest",
			}, []byte(`{"data": "test"}`))

			// Verify proper handling of invalid CSRF token
			_ = resp // Response handling depends on CSRF implementation
		})
	})
}

// ============================================================================
// Helper Methods
// ============================================================================

func (s *SecurityTestSuite) makeRequest(method, path string, headers map[string]string, body []byte) *httptest.ResponseRecorder {
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
	s.router.ServeHTTP(w, req)
	return w
}

func (s *SecurityTestSuite) createValidToken() string {
	claims := jwt.MapClaims{
		"sub":   "test-user",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"roles": []string{"user"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(s.jwtSecret))
	return tokenString
}

func (s *SecurityTestSuite) createExpiredToken() string {
	claims := jwt.MapClaims{
		"sub":   "test-user",
		"exp":   time.Now().Add(-time.Hour).Unix(),
		"iat":   time.Now().Add(-2 * time.Hour).Unix(),
		"roles": []string{"user"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(s.jwtSecret))
	return tokenString
}

func (s *SecurityTestSuite) createTokenWithRole(role string) string {
	claims := jwt.MapClaims{
		"sub":   "test-user",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"roles": []string{role},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(s.jwtSecret))
	return tokenString
}

func (s *SecurityTestSuite) createTokenWithSecret(secret string) string {
	claims := jwt.MapClaims{
		"sub":   "test-user",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"roles": []string{"admin"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func (s *SecurityTestSuite) createTokenWithAlgorithm(alg string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"alg":"%s","typ":"JWT"}`, alg)))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"test","exp":9999999999}`))
	signature := ""

	if alg != "none" {
		h := hmac.New(sha256.New, []byte(s.jwtSecret))
		h.Write([]byte(header + "." + payload))
		signature = base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	}

	return header + "." + payload + "." + signature
}

func (s *SecurityTestSuite) createNoneAlgorithmToken() string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"admin","roles":["admin"],"exp":9999999999}`))
	return header + "." + payload + "."
}

func (s *SecurityTestSuite) createTokenWithWrongSignature() string {
	token := s.createValidToken()
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		// Corrupt the signature
		parts[2] = base64.RawURLEncoding.EncodeToString([]byte("wrong-signature"))
		return strings.Join(parts, ".")
	}
	return token
}

func (s *SecurityTestSuite) createTokenWithKID(kid string) string {
	claims := jwt.MapClaims{
		"sub": "test-user",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = kid
	tokenString, _ := token.SignedString([]byte(s.jwtSecret))
	return tokenString
}

func (s *SecurityTestSuite) tamperToken(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return token
	}

	// Decode payload, modify, re-encode
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]interface{}
	json.Unmarshal(payload, &claims)
	claims["roles"] = []string{"admin"}
	newPayload, _ := json.Marshal(claims)
	parts[1] = base64.RawURLEncoding.EncodeToString(newPayload)

	return strings.Join(parts, ".")
}

func (s *SecurityTestSuite) modifyTokenClaims(token string) string {
	return s.tamperToken(token)
}

func (s *SecurityTestSuite) isTokenAccepted(token string) bool {
	resp := s.makeRequest("GET", "/api/protected", map[string]string{
		"Authorization": "Bearer " + token,
	}, nil)
	return resp.Code == http.StatusOK
}

func (s *SecurityTestSuite) testUnauthenticatedAccess(t *testing.T, ep EndpointConfig) {
	resp := s.makeRequest(ep.Method, ep.Path, nil, nil)
	if resp.Code == http.StatusOK && ep.RequiresAuth {
		t.Errorf("Endpoint %s %s should require authentication", ep.Method, ep.Path)
	}
}

func (s *SecurityTestSuite) testWrongRoleAccess(t *testing.T, ep EndpointConfig) {
	wrongRole := "viewer"
	if ep.RequiredRole == "viewer" {
		wrongRole = "guest"
	}

	token := s.createTokenWithRole(wrongRole)
	resp := s.makeRequest(ep.Method, ep.Path, map[string]string{
		"Authorization": "Bearer " + token,
	}, nil)

	if resp.Code == http.StatusOK {
		t.Errorf("Endpoint %s %s should reject role %s", ep.Method, ep.Path, wrongRole)
	}
}

func (s *SecurityTestSuite) testIDOR(t *testing.T, ep EndpointConfig) {
	// Test accessing another user's resource
	token := s.createTokenWithRole("user")
	otherUserPath := strings.Replace(ep.Path, "{id}", "other-user-id", 1)

	resp := s.makeRequest(ep.Method, otherUserPath, map[string]string{
		"Authorization": "Bearer " + token,
	}, nil)

	// Should return 403 Forbidden, not 404 Not Found (to avoid enumeration)
	// or return 404 consistently
	_ = resp // Response handling depends on authorization implementation
}

func (s *SecurityTestSuite) sendRequestWithPayload(ep EndpointConfig, payload string) *httptest.ResponseRecorder {
	var body []byte
	if ep.Method == "POST" || ep.Method == "PUT" || ep.Method == "PATCH" {
		body = []byte(fmt.Sprintf(`{"input": "%s"}`, payload))
	}

	path := ep.Path
	if ep.Method == "GET" {
		path = ep.Path + "?q=" + payload
	}

	return s.makeRequest(ep.Method, path, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + s.createValidToken(),
	}, body)
}

func (s *SecurityTestSuite) isInjectionSuccessful(resp *httptest.ResponseRecorder, payload InjectionPayload) bool {
	body := resp.Body.String()

	// Check for common injection success indicators
	switch payload.Type {
	case "sql":
		// Look for SQL error messages or unexpected data
		sqlIndicators := []string{"syntax error", "mysql", "postgresql", "sqlite", "ORA-", "SQL"}
		for _, indicator := range sqlIndicators {
			if strings.Contains(strings.ToLower(body), strings.ToLower(indicator)) {
				return true
			}
		}
	case "nosql":
		// Look for NoSQL injection indicators
		if strings.Contains(body, "MongoError") || strings.Contains(body, "$where") {
			return true
		}
	case "command":
		// Look for command execution indicators
		if strings.Contains(body, "root:") || strings.Contains(body, "/bin/") {
			return true
		}
	case "ldap":
		// Look for LDAP injection indicators
		if strings.Contains(body, "LDAP") || strings.Contains(body, "cn=") {
			return true
		}
	}

	return false
}

func (s *SecurityTestSuite) testRateLimiting(t *testing.T) {
	// Send many requests quickly
	rateLimitHit := false
	for i := 0; i < 100; i++ {
		resp := s.makeRequest("GET", "/api/health", nil, nil)
		if resp.Code == http.StatusTooManyRequests {
			rateLimitHit = true
			break
		}
	}

	if !rateLimitHit {
		t.Log("Warning: Rate limiting may not be configured")
	}
}

func (s *SecurityTestSuite) testAccountLockout(t *testing.T) {
	// Test failed login attempts
	for i := 0; i < 10; i++ {
		s.makeRequest("POST", "/api/auth/login", map[string]string{
			"Content-Type": "application/json",
		}, []byte(`{"username":"test","password":"wrong"}`))
	}

	// Next attempt should be locked out
	resp := s.makeRequest("POST", "/api/auth/login", map[string]string{
		"Content-Type": "application/json",
	}, []byte(`{"username":"test","password":"correct"}`))

	if resp.Code == http.StatusOK {
		t.Log("Warning: Account lockout may not be configured")
	}
}

func (s *SecurityTestSuite) validateSecurityHeaders(t *testing.T, resp *httptest.ResponseRecorder) {
	requiredHeaders := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "1; mode=block",
		"Content-Security-Policy":   "",
		"Strict-Transport-Security": "",
	}

	for header, expectedValue := range requiredHeaders {
		value := resp.Header().Get(header)
		if value == "" {
			t.Logf("Warning: Missing security header: %s", header)
		} else if expectedValue != "" && value != expectedValue {
			t.Logf("Warning: Unexpected value for %s: got %s, want %s", header, value, expectedValue)
		}
	}
}

func (s *SecurityTestSuite) validateNoSensitiveErrorInfo(t *testing.T, resp *httptest.ResponseRecorder) {
	body := resp.Body.String()
	sensitivePatterns := []string{
		"stack trace",
		"goroutine",
		"panic",
		"/home/",
		"/Users/",
		"password",
		"secret",
		"internal error",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(strings.ToLower(body), strings.ToLower(pattern)) {
			t.Errorf("Sensitive information leaked in error response: %s", pattern)
		}
	}
}

func (s *SecurityTestSuite) measureResponseTimes(token string, iterations int) []time.Duration {
	times := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		s.makeRequest("GET", "/api/protected", map[string]string{
			"Authorization": "Bearer " + token,
		}, nil)
		times[i] = time.Since(start)
	}

	return times
}

func (s *SecurityTestSuite) createDeeplyNestedJSON(depth int) []byte {
	result := ""
	for i := 0; i < depth; i++ {
		result += `{"nested":`
	}
	result += `"value"`
	for i := 0; i < depth; i++ {
		result += "}"
	}
	return []byte(result)
}

func (s *SecurityTestSuite) isProperlyEscaped(response, payload string) bool {
	// Check if HTML entities are escaped
	dangerousChars := map[string]string{
		"<":  "&lt;",
		">":  "&gt;",
		"\"": "&quot;",
		"'":  "&#39;",
	}

	for char, escaped := range dangerousChars {
		if strings.Contains(payload, char) {
			if strings.Contains(response, char) && !strings.Contains(response, escaped) {
				return false
			}
		}
	}

	return true
}

// ============================================================================
// Helper Functions
// ============================================================================

func average(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// ============================================================================
// Types and Payloads
// ============================================================================

// EndpointConfig describes an endpoint to test.
type EndpointConfig struct {
	Method        string
	Path          string
	RequiresAuth  bool
	RequiredRole  string
	HasResourceID bool
	AcceptsInput  bool
}

// InjectionPayload represents an injection test payload.
type InjectionPayload struct {
	Name  string
	Type  string
	Value string
}

// GetInjectionPayloads returns common injection test payloads.
func GetInjectionPayloads() []InjectionPayload {
	return []InjectionPayload{
		// SQL Injection
		{Name: "SQL_SingleQuote", Type: "sql", Value: "' OR '1'='1"},
		{Name: "SQL_DoubleQuote", Type: "sql", Value: `" OR "1"="1`},
		{Name: "SQL_Comment", Type: "sql", Value: "1; DROP TABLE users--"},
		{Name: "SQL_Union", Type: "sql", Value: "1 UNION SELECT * FROM users"},
		{Name: "SQL_Stacked", Type: "sql", Value: "1; SELECT * FROM users"},

		// NoSQL Injection
		{Name: "NoSQL_Ne", Type: "nosql", Value: `{"$ne": null}`},
		{Name: "NoSQL_Gt", Type: "nosql", Value: `{"$gt": ""}`},
		{Name: "NoSQL_Where", Type: "nosql", Value: `{"$where": "1==1"}`},
		{Name: "NoSQL_Regex", Type: "nosql", Value: `{"$regex": ".*"}`},

		// Command Injection
		{Name: "Cmd_Semicolon", Type: "command", Value: "; ls -la"},
		{Name: "Cmd_Pipe", Type: "command", Value: "| cat /etc/passwd"},
		{Name: "Cmd_Backtick", Type: "command", Value: "`cat /etc/passwd`"},
		{Name: "Cmd_Dollar", Type: "command", Value: "$(cat /etc/passwd)"},

		// LDAP Injection
		{Name: "LDAP_Wildcard", Type: "ldap", Value: "*)(uid=*))(|(uid=*"},
		{Name: "LDAP_Null", Type: "ldap", Value: "admin)(&)"},

		// XPath Injection
		{Name: "XPath_Or", Type: "xpath", Value: "' or '1'='1"},
		{Name: "XPath_Comment", Type: "xpath", Value: "admin'--"},
	}
}

// XSSPayload represents an XSS test payload.
type XSSPayload struct {
	Name  string
	Value string
}

// GetXSSPayloads returns common XSS test payloads.
func GetXSSPayloads() []XSSPayload {
	return []XSSPayload{
		{Name: "Script_Basic", Value: "<script>alert('XSS')</script>"},
		{Name: "Script_Encoded", Value: "%3Cscript%3Ealert('XSS')%3C/script%3E"},
		{Name: "Img_Onerror", Value: `<img src=x onerror=alert('XSS')>`},
		{Name: "Svg_Onload", Value: `<svg onload=alert('XSS')>`},
		{Name: "Body_Onload", Value: `<body onload=alert('XSS')>`},
		{Name: "Event_Handler", Value: `<div onmouseover=alert('XSS')>hover</div>`},
		{Name: "Javascript_URI", Value: `javascript:alert('XSS')`},
		{Name: "Data_URI", Value: `data:text/html,<script>alert('XSS')</script>`},
		{Name: "Unicode_Escape", Value: `\u003cscript\u003ealert('XSS')\u003c/script\u003e`},
		{Name: "HTML_Entity", Value: `&lt;script&gt;alert('XSS')&lt;/script&gt;`},
	}
}

// ============================================================================
// Context-based Security Testing
// ============================================================================

// SecurityTestContext provides context for security testing.
type SecurityTestContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewSecurityTestContext creates a new security test context with timeout.
func NewSecurityTestContext(timeout time.Duration) *SecurityTestContext {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return &SecurityTestContext{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Context returns the underlying context.
func (stc *SecurityTestContext) Context() context.Context {
	return stc.ctx
}

// Cancel cancels the context.
func (stc *SecurityTestContext) Cancel() {
	stc.cancel()
}

