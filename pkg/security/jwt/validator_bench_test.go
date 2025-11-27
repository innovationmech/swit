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

// Package jwt provides JWT validation benchmarks.
// Performance Target: < 1ms P99 for local JWT validation.
package jwt

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// BenchmarkValidator_ValidateToken benchmarks token validation.
func BenchmarkValidator_ValidateToken(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"
	config := &Config{
		Secret: secret,
	}

	validator, err := NewValidator(config)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test token
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		b.Fatalf("Failed to sign token: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateToken(tokenString)
		if err != nil {
			b.Fatalf("ValidateToken() error = %v", err)
		}
	}
}

// BenchmarkValidator_ValidateWithContext benchmarks token validation with context.
func BenchmarkValidator_ValidateWithContext(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"
	config := &Config{
		Secret: secret,
	}

	validator, err := NewValidator(config)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test token
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		b.Fatalf("Failed to sign token: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateWithContext(ctx, tokenString)
		if err != nil {
			b.Fatalf("ValidateWithContext() error = %v", err)
		}
	}
}

// BenchmarkValidator_ValidateTokenWithClaims benchmarks token validation with custom claims.
func BenchmarkValidator_ValidateTokenWithClaims(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"
	config := &Config{
		Secret: secret,
	}

	validator, err := NewValidator(config)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test token with custom claims
	customClaims := &CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:   "user123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"user", "admin"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, customClaims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		b.Fatalf("Failed to sign token: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		claims := &CustomClaims{}
		err := validator.ValidateTokenWithClaims(tokenString, claims)
		if err != nil {
			b.Fatalf("ValidateTokenWithClaims() error = %v", err)
		}
	}
}

// BenchmarkValidator_ParseCustomClaims benchmarks parsing custom claims.
func BenchmarkValidator_ParseCustomClaims(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"
	config := &Config{
		Secret: secret,
	}

	validator, err := NewValidator(config)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test token with custom claims
	customClaims := &CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:   "user123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"user", "admin"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, customClaims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		b.Fatalf("Failed to sign token: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := validator.ParseCustomClaims(tokenString)
		if err != nil {
			b.Fatalf("ParseCustomClaims() error = %v", err)
		}
	}
}

// BenchmarkValidator_ValidateWithIssuerAndAudience benchmarks validation with issuer and audience checks.
func BenchmarkValidator_ValidateWithIssuerAndAudience(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"
	config := &Config{
		Secret:   secret,
		Issuer:   "https://auth.example.com",
		Audience: "https://api.example.com",
	}

	validator, err := NewValidator(config)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test token
	claims := jwt.MapClaims{
		"sub": "user123",
		"iss": "https://auth.example.com",
		"aud": "https://api.example.com",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		b.Fatalf("Failed to sign token: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateToken(tokenString)
		if err != nil {
			b.Fatalf("ValidateToken() error = %v", err)
		}
	}
}

// BenchmarkValidator_ValidateWithBlacklist benchmarks validation with blacklist check.
func BenchmarkValidator_ValidateWithBlacklist(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"
	config := &Config{
		Secret: secret,
	}

	blacklist := NewInMemoryBlacklist()
	defer blacklist.Stop()

	validator, err := NewValidator(config, WithBlacklist(blacklist))
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test token
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		b.Fatalf("Failed to sign token: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateWithContext(ctx, tokenString)
		if err != nil {
			b.Fatalf("ValidateWithContext() error = %v", err)
		}
	}
}

// BenchmarkExtractClaims benchmarks extracting claims from a token.
func BenchmarkExtractClaims(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"

	// Create a test token
	claims := jwt.MapClaims{
		"sub":  "user123",
		"name": "John Doe",
		"exp":  time.Now().Add(1 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		b.Fatalf("Failed to sign token: %v", err)
	}

	config := &Config{
		Secret: secret,
	}

	validator, err := NewValidator(config)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	validatedToken, err := validator.ValidateToken(tokenString)
	if err != nil {
		b.Fatalf("Failed to validate token: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ExtractClaims(validatedToken)
		if err != nil {
			b.Fatalf("ExtractClaims() error = %v", err)
		}
	}
}

// BenchmarkGetClaimString benchmarks extracting a string claim.
func BenchmarkGetClaimString(b *testing.B) {
	claims := jwt.MapClaims{
		"sub":  "user123",
		"name": "John Doe",
		"role": "admin",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = GetClaimString(claims, "name")
	}
}

// BenchmarkGetClaimInt64 benchmarks extracting an int64 claim.
func BenchmarkGetClaimInt64(b *testing.B) {
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = GetClaimInt64(claims, "exp")
	}
}

// BenchmarkInMemoryBlacklist_IsBlacklisted benchmarks blacklist check.
func BenchmarkInMemoryBlacklist_IsBlacklisted(b *testing.B) {
	bl := NewInMemoryBlacklist()
	defer bl.Stop()

	ctx := context.Background()
	token := "test-token"
	expiry := time.Now().Add(1 * time.Hour)

	// Blacklist a token
	err := bl.Blacklist(ctx, token, expiry)
	if err != nil {
		b.Fatalf("Blacklist() error = %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := bl.IsBlacklisted(ctx, token)
		if err != nil {
			b.Fatalf("IsBlacklisted() error = %v", err)
		}
	}
}

// BenchmarkInMemoryBlacklist_Blacklist benchmarks adding tokens to blacklist.
func BenchmarkInMemoryBlacklist_Blacklist(b *testing.B) {
	bl := NewInMemoryBlacklist()
	defer bl.Stop()

	ctx := context.Background()
	expiry := time.Now().Add(1 * time.Hour)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := bl.Blacklist(ctx, "token-"+string(rune(i)), expiry)
		if err != nil {
			b.Fatalf("Blacklist() error = %v", err)
		}
	}
}

// ============================================================================
// RSA Key Validation Benchmarks
// ============================================================================

// BenchmarkValidator_ValidateTokenRSA benchmarks RSA token validation.
func BenchmarkValidator_ValidateTokenRSA(b *testing.B) {
	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatalf("Failed to generate RSA key: %v", err)
	}

	config := &Config{
		PublicKey: &privateKey.PublicKey,
	}

	validator, err := NewValidator(config)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test token with RSA
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		b.Fatalf("Failed to sign token: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateToken(tokenString)
		if err != nil {
			b.Fatalf("ValidateToken() error = %v", err)
		}
	}
}

// ============================================================================
// Concurrent Validation Benchmarks
// ============================================================================

// BenchmarkValidator_ConcurrentValidation benchmarks concurrent token validation.
func BenchmarkValidator_ConcurrentValidation(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"
	config := &Config{
		Secret: secret,
	}

	validator, err := NewValidator(config)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test token
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		b.Fatalf("Failed to sign token: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := validator.ValidateToken(tokenString)
			if err != nil {
				b.Fatalf("ValidateToken() error = %v", err)
			}
		}
	})
}

// BenchmarkValidator_ConcurrentWithBlacklist benchmarks concurrent validation with blacklist.
func BenchmarkValidator_ConcurrentWithBlacklist(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"
	config := &Config{
		Secret: secret,
	}

	blacklist := NewInMemoryBlacklist()
	defer blacklist.Stop()

	validator, err := NewValidator(config, WithBlacklist(blacklist))
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test token
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		b.Fatalf("Failed to sign token: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := validator.ValidateWithContext(ctx, tokenString)
			if err != nil {
				b.Fatalf("ValidateWithContext() error = %v", err)
			}
		}
	})
}

// ============================================================================
// Large Payload Benchmarks
// ============================================================================

// BenchmarkValidator_LargePayload benchmarks validation with large payload.
func BenchmarkValidator_LargePayload(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"
	config := &Config{
		Secret: secret,
	}

	validator, err := NewValidator(config)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create a token with large payload
	largeData := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeData[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d_with_some_longer_content", i)
	}

	claims := jwt.MapClaims{
		"sub":  "user123",
		"exp":  time.Now().Add(1 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
		"data": largeData,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		b.Fatalf("Failed to sign token: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateToken(tokenString)
		if err != nil {
			b.Fatalf("ValidateToken() error = %v", err)
		}
	}
}

// ============================================================================
// Claim Extraction Benchmarks
// ============================================================================

// BenchmarkGetClaimStringSlice benchmarks extracting a string slice claim.
func BenchmarkGetClaimStringSlice(b *testing.B) {
	claims := jwt.MapClaims{
		"sub":   "user123",
		"roles": []interface{}{"admin", "user", "moderator", "editor", "viewer"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = GetClaimStringSlice(claims, "roles")
	}
}

// BenchmarkGetClaimTime benchmarks extracting a time claim.
func BenchmarkGetClaimTime(b *testing.B) {
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = GetClaimTime(claims, "exp")
	}
}

// ============================================================================
// Token Parsing Benchmarks
// ============================================================================

// BenchmarkValidator_ParseToken benchmarks unverified token parsing.
func BenchmarkValidator_ParseToken(b *testing.B) {
	secret := "test-secret-key-for-benchmarking"
	config := &Config{
		Secret: secret,
	}

	validator, err := NewValidator(config)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test token
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		b.Fatalf("Failed to sign token: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := validator.ParseToken(tokenString)
		if err != nil {
			b.Fatalf("ParseToken() error = %v", err)
		}
	}
}

// ============================================================================
// Blacklist Concurrent Benchmarks
// ============================================================================

// BenchmarkInMemoryBlacklist_ConcurrentAccess benchmarks concurrent blacklist access.
func BenchmarkInMemoryBlacklist_ConcurrentAccess(b *testing.B) {
	bl := NewInMemoryBlacklist()
	defer bl.Stop()

	ctx := context.Background()
	expiry := time.Now().Add(1 * time.Hour)

	// Pre-populate blacklist
	for i := 0; i < 1000; i++ {
		err := bl.Blacklist(ctx, fmt.Sprintf("token-%d", i), expiry)
		if err != nil {
			b.Fatalf("Blacklist() error = %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			token := fmt.Sprintf("token-%d", i%1000)
			_, err := bl.IsBlacklisted(ctx, token)
			if err != nil {
				b.Fatalf("IsBlacklisted() error = %v", err)
			}
			i++
		}
	})
}

// BenchmarkInMemoryBlacklist_MixedOperations benchmarks mixed read/write operations.
func BenchmarkInMemoryBlacklist_MixedOperations(b *testing.B) {
	bl := NewInMemoryBlacklist()
	defer bl.Stop()

	ctx := context.Background()
	expiry := time.Now().Add(1 * time.Hour)

	// Pre-populate blacklist
	for i := 0; i < 500; i++ {
		err := bl.Blacklist(ctx, fmt.Sprintf("token-%d", i), expiry)
		if err != nil {
			b.Fatalf("Blacklist() error = %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			token := fmt.Sprintf("token-%d", i%1000)
			if i%5 == 0 {
				// 20% writes
				_ = bl.Blacklist(ctx, token, expiry)
			} else {
				// 80% reads
				_, _ = bl.IsBlacklisted(ctx, token)
			}
			i++
		}
	})
}

// ============================================================================
// MapClaimsToCustomClaims Benchmark
// ============================================================================

// BenchmarkMapClaimsToCustomClaims benchmarks converting map claims to custom claims.
func BenchmarkMapClaimsToCustomClaims(b *testing.B) {
	claims := jwt.MapClaims{
		"sub":         "user123",
		"exp":         time.Now().Add(1 * time.Hour).Unix(),
		"iat":         time.Now().Unix(),
		"user_id":     "user123",
		"username":    "testuser",
		"email":       "test@example.com",
		"roles":       []interface{}{"admin", "user"},
		"permissions": []interface{}{"read", "write"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := MapClaimsToCustomClaims(claims)
		if err != nil {
			b.Fatalf("MapClaimsToCustomClaims() error = %v", err)
		}
	}
}
