// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	secjwt "github.com/innovationmech/swit/pkg/security/jwt"
)

// Test keys for JWT signing
const (
	testPrivateKey = `-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDLNobYaSFdMSTe
9ymDHd4Qf8V9qnpjzs/Ax+WKnp2chge+P600SxqlW3UKinBdDdxUZVXwx1sAiRPV
U+xpmPAN/0bjQyJzWnke3LDvMdFHBfRB4Nt010OcSh96vVuCuS+KB+fxhUXdotEx
wfzqTQdcoVeD3vcETQVD9aVcqNgXDU1T1OJr2Dg7d4qGLYG23PFyp/QEt4RJ0VDm
LEKqbujwBE1BQQUbDUOsXTciu4E9qNH7XoIwIHMyBX7Z122Nv7mETiYOT7DXZbqg
gZVttvW0RqJSqhl6jzhFmpOLLjYS9ghLCb54vauymKrgq68gUaj9FQroyoZmRTKG
2T1FQkE7AgMBAAECggEAHqNLGQAKDIu3GY9d02ENP3iRdSe0SdhUS1D4GHpfR0o6
ivenwgLGyC0t9sJg5tlgTDB2ZhXFxc1k+kdWaqSpILsfBsRIQpMJQDAZfwldmkiE
B0CtuwPPiOtGAWfg5FMjnzsflbLjEeOVMwbNKSLRnCjOfIPWbfASKKkLCZmCnEAw
zkeKPfAlNvW9sciYTVEhZ2EdV2s5AW8zzb/yvGpH6aVLlJ/+BcBXvLw0nCmntcyp
RNP6/hSOXVP8Gkap9c+tiA04ji4MLXzZW2+XjDv/Z70fmnndPAs/7a9Fg88UrkzO
XGc8TzEjCDz3TMqSoOUzAosk5kK9Mc7WvUGAVmF3JQKBgQDv1X2J74eCknJOSkQT
yZnNKna/bouEXQL3hRum1x7IMqx4X9H+Gk7ZQSo2ns/21FlXcQSRY14egQdzMeiB
7We4guapb8tUWeWF5v7GbFf0eoJ3gcQDj7oWsa890EmzBKzCKJtokW2kqtBJ8sJq
Qh0oLxTwlCZPWtlA6J0GMUZeRQKBgQDY6R15PTjFMbHOHNKN2CcYF3hn0xlUgR6d
7SJM6Bh/ZRIoNr2dB6J8gnmzzYzY9Y/pddr5ozELbcNYxEDWdr996guYJlqaufib
oHZ9Sy38c8rgnFxp/lrt0FeIhIrP0nsE1srPsrk7ldCEbjx36LcuQM1o2GEo92BF
TQftyz3ZfwKBgFrmGIGaBksnxCkGHs09IIzRJlahyEEvm3tCuNtAN0t7YUDyWD2t
rOrMtvoisQGFNCNfE3MjLT30e2Veqhfsad5VxqS7WV4sAEEC7tc3oxJnCGHRDgCn
jckiKSANfJFcGToxd81nKR47G1ybpLHvQuvDBHW2QNrcvPDL+Q+qx1fdAoGAS+VN
NcWxHnZj4118Yrs1+p0DuThIzaOcJd/6N3SiVbj0oHN+5vnr5ar1kG8kkClj4Gkn
ZF+wYnJWfrG0ihXkrNb+lY8d9rOJhFKiAvcSMRoG645qW3/vKvTSG+dcdpkMCEZr
kj7Tx0CFREEaEU6xAZMVDFFhtabQ3Y61boPFsVsCgYAruAFhGo1EvqUG3LstuHjF
Nd2j7Mdog2iKKWdRDfEvqb9bJVslatQojh7kAkGPqGff4+JL7Tt31blUwjUAFcTN
dWo4adIErLj5jwK/boHlcvRBOVXst7PAfKprZxHt0x0szJcF8ulGQBSqf+m6LPTD
zzGSdM6FG2jdl7FNAt6PjQ==
-----END PRIVATE KEY-----`

	testPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAyzaG2GkhXTEk3vcpgx3e
EH/Ffap6Y87PwMflip6dnIYHvj+tNEsapVt1CopwXQ3cVGVV8MdbAIkT1VPsaZjw
Df9G40Mic1p5Htyw7zHRRwX0QeDbdNdDnEofer1bgrkvigfn8YVF3aLRMcH86k0H
XKFXg973BE0FQ/WlXKjYFw1NU9Tia9g4O3eKhi2Bttzxcqf0BLeESdFQ5ixCqm7o
8ARNQUEFGw1DrF03IruBPajR+16CMCBzMgV+2ddtjb+5hE4mDk+w12W6oIGVbbb1
tEaiUqoZeo84RZqTiy42EvYISwm+eL2rspiq4KuvIFGo/RUK6MqGZkUyhtk9RUJB
OwIDAQAB
-----END PUBLIC KEY-----`
)

func init() {
	gin.SetMode(gin.TestMode)
}

// createTestJWTValidator creates a JWT validator for testing
func createTestJWTValidator(t *testing.T) *secjwt.Validator {
	block, _ := pem.Decode([]byte(testPublicKey))
	if block == nil {
		t.Fatal("failed to parse PEM block containing the public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse public key: %v", err)
	}

	config := &secjwt.Config{
		PublicKey:         pub.(*rsa.PublicKey),
		AllowedAlgorithms: []string{"RS256"},
		Issuer:            "test-issuer",
		Audience:          "test-audience",
		LeewayDuration:    5 * time.Second,
	}

	validator, err := secjwt.NewValidator(config)
	if err != nil {
		t.Fatalf("failed to create JWT validator: %v", err)
	}

	return validator
}

// createTestToken creates a test JWT token
func createTestToken(t *testing.T, claims jwt.MapClaims) string {
	block, _ := pem.Decode([]byte(testPrivateKey))
	if block == nil {
		t.Fatal("failed to parse PEM block containing the private key")
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse private key: %v", err)
	}

	key, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		t.Fatal("not an RSA private key")
	}

	// Set default claims
	now := time.Now()
	if claims["iss"] == nil {
		claims["iss"] = "test-issuer"
	}
	if claims["aud"] == nil {
		claims["aud"] = "test-audience"
	}
	if claims["exp"] == nil {
		claims["exp"] = now.Add(time.Hour).Unix()
	}
	if claims["iat"] == nil {
		claims["iat"] = now.Unix()
	}
	if claims["sub"] == nil {
		claims["sub"] = "test-user"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	return tokenString
}

// TestOAuth2Middleware tests the basic OAuth2 middleware functionality
func TestOAuth2Middleware(t *testing.T) {
	validator := createTestJWTValidator(t)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		checkUserInfo  bool
	}{
		{
			name: "valid token",
			token: createTestToken(t, jwt.MapClaims{
				"username": "testuser",
				"email":    "test@example.com",
				"roles":    []string{"user", "admin"},
			}),
			expectedStatus: http.StatusOK,
			checkUserInfo:  true,
		},
		{
			name:           "missing token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			checkUserInfo:  false,
		},
		{
			name:           "invalid token",
			token:          "invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
			checkUserInfo:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(OAuth2Middleware(nil, validator))
			router.GET("/protected", func(c *gin.Context) {
				if tt.checkUserInfo {
					userInfo, exists := GetUserInfo(c)
					if !exists {
						t.Error("expected user info in context")
						return
					}
					if userInfo.Subject != "test-user" {
						t.Errorf("expected subject 'test-user', got %s", userInfo.Subject)
					}
				}
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestOAuth2MiddlewareWithConfig tests OAuth2 middleware with custom configuration
func TestOAuth2MiddlewareWithConfig(t *testing.T) {
	validator := createTestJWTValidator(t)

	tests := []struct {
		name           string
		config         *OAuth2MiddlewareConfig
		path           string
		token          string
		expectedStatus int
	}{
		{
			name: "skip path",
			config: &OAuth2MiddlewareConfig{
				JWTValidator: validator,
				SkipPaths:    []string{"/health", "/public"},
			},
			path:           "/health",
			token:          "",
			expectedStatus: http.StatusOK,
		},
		{
			name: "optional auth - no token",
			config: &OAuth2MiddlewareConfig{
				JWTValidator: validator,
				Optional:     true,
			},
			path:           "/protected",
			token:          "",
			expectedStatus: http.StatusOK,
		},
		{
			name: "optional auth - valid token",
			config: &OAuth2MiddlewareConfig{
				JWTValidator: validator,
				Optional:     true,
			},
			path: "/protected",
			token: createTestToken(t, jwt.MapClaims{
				"username": "testuser",
			}),
			expectedStatus: http.StatusOK,
		},
		{
			name: "wildcard skip path",
			config: &OAuth2MiddlewareConfig{
				JWTValidator: validator,
				SkipPaths:    []string{"/api/public/*"},
			},
			path:           "/api/public/info",
			token:          "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(OAuth2MiddlewareWithConfig(tt.config))
			router.GET(tt.path, func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestOptionalAuth tests optional authentication middleware
func TestOptionalAuth(t *testing.T) {
	validator := createTestJWTValidator(t)

	router := gin.New()
	router.Use(OptionalAuth(nil, validator))
	router.GET("/resource", func(c *gin.Context) {
		userInfo, exists := GetUserInfo(c)
		if exists {
			c.JSON(http.StatusOK, gin.H{"authenticated": true, "user": userInfo.Subject})
		} else {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		}
	})

	tests := []struct {
		name            string
		token           string
		expectAuth      bool
		expectedSubject string
	}{
		{
			name:            "with valid token",
			token:           createTestToken(t, jwt.MapClaims{}),
			expectAuth:      true,
			expectedSubject: "test-user",
		},
		{
			name:       "without token",
			token:      "",
			expectAuth: false,
		},
		{
			name:       "with invalid token",
			token:      "invalid.token",
			expectAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/resource", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}
		})
	}
}

// TestExtractBearerToken tests bearer token extraction
func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		expectToken string
		expectError bool
	}{
		{
			name:        "valid bearer token",
			authHeader:  "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expectToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expectError: false,
		},
		{
			name:        "bearer with extra spaces",
			authHeader:  "Bearer  eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9  ",
			expectToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expectError: false,
		},
		{
			name:        "invalid format - no bearer",
			authHeader:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expectToken: "",
			expectError: true,
		},
		{
			name:        "invalid scheme",
			authHeader:  "Basic dXNlcjpwYXNz",
			expectToken: "",
			expectError: true,
		},
		{
			name:        "empty token",
			authHeader:  "Bearer ",
			expectToken: "",
			expectError: true,
		},
		{
			name:        "empty header",
			authHeader:  "",
			expectToken: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := extractBearerToken(tt.authHeader)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if token != tt.expectToken {
					t.Errorf("expected token %q, got %q", tt.expectToken, token)
				}
			}
		})
	}
}

// TestGetUserInfo tests retrieving user info from context
func TestGetUserInfo(t *testing.T) {
	validator := createTestJWTValidator(t)

	router := gin.New()
	router.Use(OAuth2Middleware(nil, validator))
	router.GET("/user", func(c *gin.Context) {
		userInfo, exists := GetUserInfo(c)
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user info not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"subject":  userInfo.Subject,
			"username": userInfo.Username,
			"email":    userInfo.Email,
			"roles":    userInfo.Roles,
		})
	})

	token := createTestToken(t, jwt.MapClaims{
		"username": "john",
		"email":    "john@example.com",
		"roles":    []string{"user", "admin"},
	})

	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestGetClaims tests retrieving claims from context
func TestGetClaims(t *testing.T) {
	validator := createTestJWTValidator(t)

	router := gin.New()
	router.Use(OAuth2Middleware(nil, validator))
	router.GET("/claims", func(c *gin.Context) {
		claims, exists := GetClaims(c)
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "claims not found"})
			return
		}

		c.JSON(http.StatusOK, claims)
	})

	token := createTestToken(t, jwt.MapClaims{
		"custom_claim": "custom_value",
	})

	req := httptest.NewRequest(http.MethodGet, "/claims", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestGetTokenString tests retrieving token string from context
func TestGetTokenString(t *testing.T) {
	validator := createTestJWTValidator(t)

	var capturedToken string

	router := gin.New()
	router.Use(OAuth2Middleware(nil, validator))
	router.GET("/token", func(c *gin.Context) {
		tokenStr, exists := GetTokenString(c)
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token not found"})
			return
		}

		capturedToken = tokenStr
		c.JSON(http.StatusOK, gin.H{"has_token": true})
	})

	token := createTestToken(t, jwt.MapClaims{})

	req := httptest.NewRequest(http.MethodGet, "/token", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if capturedToken != token {
		t.Error("captured token does not match original token")
	}
}

// TestShouldSkipPath tests path skipping logic
func TestShouldSkipPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		skipPaths  []string
		shouldSkip bool
	}{
		{
			name:       "exact match",
			path:       "/health",
			skipPaths:  []string{"/health", "/metrics"},
			shouldSkip: true,
		},
		{
			name:       "no match",
			path:       "/api/users",
			skipPaths:  []string{"/health", "/metrics"},
			shouldSkip: false,
		},
		{
			name:       "wildcard match",
			path:       "/public/assets/style.css",
			skipPaths:  []string{"/public/*"},
			shouldSkip: true,
		},
		{
			name:       "wildcard no match",
			path:       "/api/users",
			skipPaths:  []string{"/public/*"},
			shouldSkip: false,
		},
		{
			name:       "empty skip paths",
			path:       "/api/users",
			skipPaths:  []string{},
			shouldSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipPath(tt.path, tt.skipPaths)
			if result != tt.shouldSkip {
				t.Errorf("expected shouldSkip=%v, got %v", tt.shouldSkip, result)
			}
		})
	}
}

// TestCustomErrorHandler tests custom error handler
func TestCustomErrorHandler(t *testing.T) {
	validator := createTestJWTValidator(t)

	customCalled := false
	customErrorHandler := func(c *gin.Context, err error) {
		customCalled = true
		c.JSON(http.StatusUnauthorized, gin.H{
			"custom_error": true,
			"message":      err.Error(),
		})
		c.Abort()
	}

	router := gin.New()
	config := &OAuth2MiddlewareConfig{
		JWTValidator: validator,
		ErrorHandler: customErrorHandler,
	}
	router.Use(OAuth2MiddlewareWithConfig(config))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	// No authorization header

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !customCalled {
		t.Error("expected custom error handler to be called")
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestCookieAuth tests cookie-based authentication
func TestCookieAuth(t *testing.T) {
	validator := createTestJWTValidator(t)

	config := &OAuth2MiddlewareConfig{
		JWTValidator: validator,
		CookieName:   "auth_token",
	}

	router := gin.New()
	router.Use(OAuth2MiddlewareWithConfig(config))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	token := createTestToken(t, jwt.MapClaims{})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: token,
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestMustGetUserInfo tests MustGetUserInfo panic behavior
func TestMustGetUserInfo(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic but did not occur")
		}
	}()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	MustGetUserInfo(c) // Should panic
}

// TestMustGetClaims tests MustGetClaims panic behavior
func TestMustGetClaims(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic but did not occur")
		}
	}()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	MustGetClaims(c) // Should panic
}

// TestRequireRoles tests role-based authorization
func TestRequireRoles(t *testing.T) {
	tests := []struct {
		name           string
		userRoles      []string
		requiredRoles  []string
		expectedStatus int
	}{
		{
			name:           "user has required role",
			userRoles:      []string{"user", "admin"},
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user has one of multiple required roles",
			userRoles:      []string{"user"},
			requiredRoles:  []string{"admin", "user"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user missing required role",
			userRoles:      []string{"user"},
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "user has no roles",
			userRoles:      []string{},
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				// Mock user info in context
				userInfo := &UserInfo{
					Subject: "test-user",
					Roles:   tt.userRoles,
				}
				c.Set(ContextKeyUserInfo, userInfo)
				c.Next()
			})
			router.Use(RequireRoles(tt.requiredRoles...))
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestRequireAllRoles tests requiring all specified roles
func TestRequireAllRoles(t *testing.T) {
	tests := []struct {
		name           string
		userRoles      []string
		requiredRoles  []string
		expectedStatus int
	}{
		{
			name:           "user has all required roles",
			userRoles:      []string{"user", "admin", "moderator"},
			requiredRoles:  []string{"admin", "moderator"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user missing one required role",
			userRoles:      []string{"user", "admin"},
			requiredRoles:  []string{"admin", "moderator"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "user has no roles",
			userRoles:      []string{},
			requiredRoles:  []string{"admin", "moderator"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				userInfo := &UserInfo{
					Subject: "test-user",
					Roles:   tt.userRoles,
				}
				c.Set(ContextKeyUserInfo, userInfo)
				c.Next()
			})
			router.Use(RequireAllRoles(tt.requiredRoles...))
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestRequireScopes tests scope-based authorization
func TestRequireScopes(t *testing.T) {
	tests := []struct {
		name           string
		userScopes     []string
		requiredScopes []string
		expectedStatus int
	}{
		{
			name:           "user has required scope",
			userScopes:     []string{"read:users", "write:users"},
			requiredScopes: []string{"read:users"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user has one of multiple required scopes",
			userScopes:     []string{"read:users"},
			requiredScopes: []string{"read:users", "write:users"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user missing required scope",
			userScopes:     []string{"read:users"},
			requiredScopes: []string{"write:users"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "user has no scopes",
			userScopes:     []string{},
			requiredScopes: []string{"read:users"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				userInfo := &UserInfo{
					Subject: "test-user",
					Scopes:  tt.userScopes,
				}
				c.Set(ContextKeyUserInfo, userInfo)
				c.Next()
			})
			router.Use(RequireScopes(tt.requiredScopes...))
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestRequireAllScopes tests requiring all specified scopes
func TestRequireAllScopes(t *testing.T) {
	tests := []struct {
		name           string
		userScopes     []string
		requiredScopes []string
		expectedStatus int
	}{
		{
			name:           "user has all required scopes",
			userScopes:     []string{"read:users", "write:users", "delete:users"},
			requiredScopes: []string{"read:users", "write:users"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user missing one required scope",
			userScopes:     []string{"read:users"},
			requiredScopes: []string{"read:users", "write:users"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				userInfo := &UserInfo{
					Subject: "test-user",
					Scopes:  tt.userScopes,
				}
				c.Set(ContextKeyUserInfo, userInfo)
				c.Next()
			})
			router.Use(RequireAllScopes(tt.requiredScopes...))
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestRequirePermissions tests permission-based authorization
func TestRequirePermissions(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		userInfo := &UserInfo{
			Subject: "test-user",
			Scopes:  []string{"read:data", "write:data"},
		}
		c.Set(ContextKeyUserInfo, userInfo)
		c.Next()
	})
	router.Use(RequirePermissions("read:data"))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestRequireRoleOrScope tests combined role or scope authorization
func TestRequireRoleOrScope(t *testing.T) {
	tests := []struct {
		name           string
		userRoles      []string
		userScopes     []string
		required       []string
		expectedStatus int
	}{
		{
			name:           "user has required role",
			userRoles:      []string{"admin"},
			userScopes:     []string{},
			required:       []string{"admin", "write:all"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user has required scope",
			userRoles:      []string{},
			userScopes:     []string{"write:all"},
			required:       []string{"admin", "write:all"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user has neither role nor scope",
			userRoles:      []string{"user"},
			userScopes:     []string{"read:all"},
			required:       []string{"admin", "write:all"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				userInfo := &UserInfo{
					Subject: "test-user",
					Roles:   tt.userRoles,
					Scopes:  tt.userScopes,
				}
				c.Set(ContextKeyUserInfo, userInfo)
				c.Next()
			})
			router.Use(RequireRoleOrScope(tt.required...))
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestRequireAuthenticatedUser tests authenticated user requirement
func TestRequireAuthenticatedUser(t *testing.T) {
	tests := []struct {
		name           string
		hasUserInfo    bool
		expectedStatus int
	}{
		{
			name:           "authenticated user",
			hasUserInfo:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthenticated user",
			hasUserInfo:    false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			if tt.hasUserInfo {
				router.Use(func(c *gin.Context) {
					userInfo := &UserInfo{Subject: "test-user"}
					c.Set(ContextKeyUserInfo, userInfo)
					c.Next()
				})
			}
			router.Use(RequireAuthenticatedUser())
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestRequireCustomClaim tests custom claim validation
func TestRequireCustomClaim(t *testing.T) {
	tests := []struct {
		name           string
		claims         jwt.MapClaims
		claimName      string
		requiredValue  interface{}
		expectedStatus int
	}{
		{
			name: "matching claim",
			claims: jwt.MapClaims{
				"department": "engineering",
			},
			claimName:      "department",
			requiredValue:  "engineering",
			expectedStatus: http.StatusOK,
		},
		{
			name: "non-matching claim",
			claims: jwt.MapClaims{
				"department": "sales",
			},
			claimName:      "department",
			requiredValue:  "engineering",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "missing claim",
			claims:         jwt.MapClaims{},
			claimName:      "department",
			requiredValue:  "engineering",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(ContextKeyClaims, tt.claims)
				c.Next()
			})
			router.Use(RequireCustomClaim(tt.claimName, tt.requiredValue))
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestRequireClaimValidator tests custom claim validator function
func TestRequireClaimValidator(t *testing.T) {
	validator := func(claims map[string]interface{}) error {
		level, ok := claims["level"].(float64)
		if !ok {
			return fmt.Errorf("missing or invalid level claim")
		}
		if level < 5 {
			return fmt.Errorf("insufficient level: requires level >= 5")
		}
		return nil
	}

	tests := []struct {
		name           string
		claims         jwt.MapClaims
		expectedStatus int
	}{
		{
			name: "valid claims",
			claims: jwt.MapClaims{
				"level": float64(10),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "insufficient level",
			claims: jwt.MapClaims{
				"level": float64(3),
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "missing level",
			claims:         jwt.MapClaims{},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(ContextKeyClaims, tt.claims)
				c.Next()
			})
			router.Use(RequireClaimValidator(validator))
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestHasAnyRole tests the hasAnyRole helper function
func TestHasAnyRole(t *testing.T) {
	tests := []struct {
		name          string
		userRoles     []string
		requiredRoles []string
		expected      bool
	}{
		{
			name:          "has one role",
			userRoles:     []string{"user", "admin"},
			requiredRoles: []string{"admin"},
			expected:      true,
		},
		{
			name:          "has none",
			userRoles:     []string{"user"},
			requiredRoles: []string{"admin", "moderator"},
			expected:      false,
		},
		{
			name:          "empty required roles",
			userRoles:     []string{"user"},
			requiredRoles: []string{},
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasAnyRole(tt.userRoles, tt.requiredRoles)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestHasAllRoles tests the hasAllRoles helper function
func TestHasAllRoles(t *testing.T) {
	tests := []struct {
		name          string
		userRoles     []string
		requiredRoles []string
		expected      bool
	}{
		{
			name:          "has all roles",
			userRoles:     []string{"user", "admin", "moderator"},
			requiredRoles: []string{"user", "admin"},
			expected:      true,
		},
		{
			name:          "missing one role",
			userRoles:     []string{"user", "admin"},
			requiredRoles: []string{"user", "admin", "moderator"},
			expected:      false,
		},
		{
			name:          "empty required roles",
			userRoles:     []string{"user"},
			requiredRoles: []string{},
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasAllRoles(tt.userRoles, tt.requiredRoles)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
