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

// Package middleware provides middleware benchmarks.
// Performance Target: < 15ms P99 total overhead for security middleware stack.
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	secjwt "github.com/innovationmech/swit/pkg/security/jwt"
	"github.com/innovationmech/swit/pkg/security/opa"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

// ============================================================================
// JWT Authentication Middleware Benchmarks
// ============================================================================

// createBenchToken creates a valid JWT token for benchmarking.
func createBenchToken(secret string) string {
	claims := jwt.MapClaims{
		"sub":      "user123",
		"username": "testuser",
		"email":    "test@example.com",
		"roles":    []interface{}{"admin", "user"},
		"exp":      time.Now().Add(time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

// BenchmarkOAuth2Middleware_ValidToken benchmarks OAuth2 middleware with valid token.
func BenchmarkOAuth2Middleware_ValidToken(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"

	jwtConfig := &secjwt.Config{
		Secret: secret,
	}
	jwtValidator, err := secjwt.NewValidator(jwtConfig)
	if err != nil {
		b.Fatalf("Failed to create JWT validator: %v", err)
	}

	middleware := OAuth2Middleware(nil, jwtValidator)

	router := gin.New()
	router.Use(middleware)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	token := createBenchToken(secret)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkOAuth2Middleware_SkipPath benchmarks middleware path skipping.
func BenchmarkOAuth2Middleware_SkipPath(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"

	jwtConfig := &secjwt.Config{
		Secret: secret,
	}
	jwtValidator, err := secjwt.NewValidator(jwtConfig)
	if err != nil {
		b.Fatalf("Failed to create JWT validator: %v", err)
	}

	config := &OAuth2MiddlewareConfig{
		JWTValidator: jwtValidator,
		SkipPaths:    []string{"/health", "/api/public/*"},
	}
	middleware := OAuth2MiddlewareWithConfig(config)

	router := gin.New()
	router.Use(middleware)
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkOAuth2Middleware_InvalidToken benchmarks middleware with invalid token.
func BenchmarkOAuth2Middleware_InvalidToken(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"

	jwtConfig := &secjwt.Config{
		Secret: secret,
	}
	jwtValidator, err := secjwt.NewValidator(jwtConfig)
	if err != nil {
		b.Fatalf("Failed to create JWT validator: %v", err)
	}

	middleware := OAuth2Middleware(nil, jwtValidator)

	router := gin.New()
	router.Use(middleware)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			b.Fatalf("Expected status 401, got %d", w.Code)
		}
	}
}

// BenchmarkOAuth2Middleware_MissingToken benchmarks middleware with missing token.
func BenchmarkOAuth2Middleware_MissingToken(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"

	jwtConfig := &secjwt.Config{
		Secret: secret,
	}
	jwtValidator, err := secjwt.NewValidator(jwtConfig)
	if err != nil {
		b.Fatalf("Failed to create JWT validator: %v", err)
	}

	middleware := OAuth2Middleware(nil, jwtValidator)

	router := gin.New()
	router.Use(middleware)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			b.Fatalf("Expected status 401, got %d", w.Code)
		}
	}
}

// ============================================================================
// OPA Policy Middleware Benchmarks
// ============================================================================

// simplePolicy is a basic RBAC policy for benchmarking.
const simplePolicy = `
package authz

default allow = false

allow {
    input.user.roles[_] == "admin"
}

allow {
    input.user.id == input.resource.owner
}
`

// BenchmarkOPAMiddleware_Allow benchmarks OPA middleware with allowed request.
func BenchmarkOPAMiddleware_Allow(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	opaConfig := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "authz.allow",
	}

	opaClient, err := opa.NewClient(ctx, opaConfig)
	if err != nil {
		b.Fatalf("Failed to create OPA client: %v", err)
	}
	defer opaClient.Close(ctx)

	// Load the simple policy
	if err := opaClient.LoadPolicy(ctx, "authz.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	// Custom input builder that sets admin role
	inputBuilder := func(c *gin.Context) (*opa.PolicyInput, error) {
		builder := opa.NewPolicyInputBuilder().
			FromHTTPRequest(c).
			WithUserID("user123").
			WithUserRoles([]string{"admin"})
		return builder.Build(), nil
	}

	middleware := OPAMiddleware(opaClient,
		WithDecisionPath("authz.allow"),
		WithInputBuilder(inputBuilder),
	)

	router := gin.New()
	router.Use(middleware)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkOPAMiddleware_Deny benchmarks OPA middleware with denied request.
func BenchmarkOPAMiddleware_Deny(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	opaConfig := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "authz.allow",
	}

	opaClient, err := opa.NewClient(ctx, opaConfig)
	if err != nil {
		b.Fatalf("Failed to create OPA client: %v", err)
	}
	defer opaClient.Close(ctx)

	// Load the simple policy
	if err := opaClient.LoadPolicy(ctx, "authz.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	// Custom input builder that sets viewer role (not admin)
	inputBuilder := func(c *gin.Context) (*opa.PolicyInput, error) {
		builder := opa.NewPolicyInputBuilder().
			FromHTTPRequest(c).
			WithUserID("user123").
			WithUserRoles([]string{"viewer"})
		return builder.Build(), nil
	}

	middleware := OPAMiddleware(opaClient,
		WithDecisionPath("authz.allow"),
		WithInputBuilder(inputBuilder),
	)

	router := gin.New()
	router.Use(middleware)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			b.Fatalf("Expected status 403, got %d", w.Code)
		}
	}
}

// BenchmarkOPAMiddleware_WhiteList benchmarks OPA middleware whitelist.
func BenchmarkOPAMiddleware_WhiteList(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	opaConfig := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "authz.allow",
	}

	opaClient, err := opa.NewClient(ctx, opaConfig)
	if err != nil {
		b.Fatalf("Failed to create OPA client: %v", err)
	}
	defer opaClient.Close(ctx)

	// Load the simple policy
	if err := opaClient.LoadPolicy(ctx, "authz.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	middleware := OPAMiddleware(opaClient,
		WithDecisionPath("authz.allow"),
		WithWhiteList([]string{"/health", "/metrics"}),
	)

	router := gin.New()
	router.Use(middleware)
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// ============================================================================
// Combined Middleware Stack Benchmarks
// ============================================================================

// BenchmarkMiddlewareStack_Full benchmarks full security middleware stack.
func BenchmarkMiddlewareStack_Full(b *testing.B) {
	ctx := context.Background()
	secret := "test-secret-key-for-benchmarking"

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	// Setup JWT validator
	jwtConfig := &secjwt.Config{
		Secret: secret,
	}
	jwtValidator, err := secjwt.NewValidator(jwtConfig)
	if err != nil {
		b.Fatalf("Failed to create JWT validator: %v", err)
	}

	// Setup OPA client
	opaConfig := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "authz.allow",
	}

	opaClient, err := opa.NewClient(ctx, opaConfig)
	if err != nil {
		b.Fatalf("Failed to create OPA client: %v", err)
	}
	defer opaClient.Close(ctx)

	// Load the simple policy
	if err := opaClient.LoadPolicy(ctx, "authz.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	// Custom input builder that extracts user from context
	inputBuilder := func(c *gin.Context) (*opa.PolicyInput, error) {
		userInfo, _ := GetUserInfo(c)
		builder := opa.NewPolicyInputBuilder().FromHTTPRequest(c)
		if userInfo != nil {
			builder.WithUserID(userInfo.Subject).
				WithUserRoles(userInfo.Roles)
		} else {
			builder.WithUserRoles([]string{"admin"}) // Default for benchmark
		}
		return builder.Build(), nil
	}

	// Create middleware stack
	authMiddleware := OAuth2Middleware(nil, jwtValidator)
	policyMiddleware := OPAMiddleware(opaClient,
		WithDecisionPath("authz.allow"),
		WithInputBuilder(inputBuilder),
	)

	router := gin.New()
	router.Use(authMiddleware)
	router.Use(policyMiddleware)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	token := createBenchToken(secret)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// ============================================================================
// Concurrent Middleware Benchmarks
// ============================================================================

// BenchmarkOAuth2Middleware_Concurrent benchmarks concurrent OAuth2 middleware.
func BenchmarkOAuth2Middleware_Concurrent(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"

	jwtConfig := &secjwt.Config{
		Secret: secret,
	}
	jwtValidator, err := secjwt.NewValidator(jwtConfig)
	if err != nil {
		b.Fatalf("Failed to create JWT validator: %v", err)
	}

	middleware := OAuth2Middleware(nil, jwtValidator)

	router := gin.New()
	router.Use(middleware)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	token := createBenchToken(secret)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Fatalf("Expected status 200, got %d", w.Code)
			}
		}
	})
}

// BenchmarkOPAMiddleware_Concurrent benchmarks concurrent OPA middleware.
func BenchmarkOPAMiddleware_Concurrent(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	opaConfig := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "authz.allow",
	}

	opaClient, err := opa.NewClient(ctx, opaConfig)
	if err != nil {
		b.Fatalf("Failed to create OPA client: %v", err)
	}
	defer opaClient.Close(ctx)

	// Load the simple policy
	if err := opaClient.LoadPolicy(ctx, "authz.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	// Custom input builder that sets admin role
	inputBuilder := func(c *gin.Context) (*opa.PolicyInput, error) {
		builder := opa.NewPolicyInputBuilder().
			FromHTTPRequest(c).
			WithUserID("user123").
			WithUserRoles([]string{"admin"})
		return builder.Build(), nil
	}

	middleware := OPAMiddleware(opaClient,
		WithDecisionPath("authz.allow"),
		WithInputBuilder(inputBuilder),
	)

	router := gin.New()
	router.Use(middleware)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Fatalf("Expected status 200, got %d", w.Code)
			}
		}
	})
}

// BenchmarkMiddlewareStack_Concurrent benchmarks concurrent full stack.
func BenchmarkMiddlewareStack_Concurrent(b *testing.B) {
	ctx := context.Background()
	secret := "test-secret-key-for-benchmarking"

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	// Setup JWT validator
	jwtConfig := &secjwt.Config{
		Secret: secret,
	}
	jwtValidator, err := secjwt.NewValidator(jwtConfig)
	if err != nil {
		b.Fatalf("Failed to create JWT validator: %v", err)
	}

	// Setup OPA client
	opaConfig := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "authz.allow",
	}

	opaClient, err := opa.NewClient(ctx, opaConfig)
	if err != nil {
		b.Fatalf("Failed to create OPA client: %v", err)
	}
	defer opaClient.Close(ctx)

	// Load the simple policy
	if err := opaClient.LoadPolicy(ctx, "authz.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	// Custom input builder
	inputBuilder := func(c *gin.Context) (*opa.PolicyInput, error) {
		builder := opa.NewPolicyInputBuilder().
			FromHTTPRequest(c).
			WithUserRoles([]string{"admin"})
		return builder.Build(), nil
	}

	// Create middleware stack
	authMiddleware := OAuth2Middleware(nil, jwtValidator)
	policyMiddleware := OPAMiddleware(opaClient,
		WithDecisionPath("authz.allow"),
		WithInputBuilder(inputBuilder),
	)

	router := gin.New()
	router.Use(authMiddleware)
	router.Use(policyMiddleware)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	token := createBenchToken(secret)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Fatalf("Expected status 200, got %d", w.Code)
			}
		}
	})
}

// ============================================================================
// Helper Function Benchmarks
// ============================================================================

// BenchmarkExtractBearerToken benchmarks token extraction.
func BenchmarkExtractBearerToken(b *testing.B) {
	authHeader := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMTIzIn0.signature"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := extractBearerToken(authHeader)
		if err != nil {
			b.Fatalf("extractBearerToken() error = %v", err)
		}
	}
}

// BenchmarkShouldSkipPath benchmarks path skipping check.
func BenchmarkShouldSkipPath(b *testing.B) {
	skipPaths := []string{"/health", "/metrics", "/api/public/*", "/swagger/*"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = shouldSkipPath("/api/public/docs", skipPaths)
	}
}

// BenchmarkSplitScopes benchmarks scope splitting.
func BenchmarkSplitScopes(b *testing.B) {
	scopeString := "openid profile email read:users write:users admin"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = splitScopes(scopeString)
	}
}

// BenchmarkExpandScopes benchmarks scope expansion.
func BenchmarkExpandScopes(b *testing.B) {
	scopes := []string{"openid", "profile email", "read:users write:users", "admin"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = expandScopes(scopes)
	}
}

// ============================================================================
// Context Helper Benchmarks
// ============================================================================

// BenchmarkGetUserInfo benchmarks user info retrieval from context.
func BenchmarkGetUserInfo(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	userInfo := &UserInfo{
		Subject:  "user123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"admin", "user"},
		Scopes:   []string{"read", "write"},
	}
	c.Set(ContextKeyUserInfo, userInfo)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = GetUserInfo(c)
	}
}

// BenchmarkGetClaims benchmarks claims retrieval from context.
func BenchmarkGetClaims(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	claims := jwt.MapClaims{
		"sub":      "user123",
		"username": "testuser",
		"email":    "test@example.com",
		"roles":    []interface{}{"admin", "user"},
	}
	c.Set(ContextKeyClaims, claims)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = GetClaims(c)
	}
}
