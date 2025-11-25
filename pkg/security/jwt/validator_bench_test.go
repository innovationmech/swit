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

package jwt

import (
	"context"
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
