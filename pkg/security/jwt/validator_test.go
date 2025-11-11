// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jwt

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewValidator(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config with secret",
			config: &Config{
				Secret: "test-secret",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "empty config",
			config:  &Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewValidator(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewValidator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_SetDefaults(t *testing.T) {
	config := &Config{
		Secret: "test-secret",
	}
	config.SetDefaults()

	if len(config.AllowedAlgorithms) == 0 {
		t.Error("Expected default allowed algorithms to be set")
	}

	if config.LeewayDuration != 5*time.Second {
		t.Errorf("Expected default leeway duration to be 5s, got %v", config.LeewayDuration)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid with secret",
			config: &Config{
				Secret: "test-secret",
			},
			wantErr: false,
		},
		{
			name: "valid with public key",
			config: &Config{
				PublicKey: "dummy-key",
			},
			wantErr: false,
		},
		{
			name: "valid with JWKS config",
			config: &Config{
				JWKSConfig: &JWKSCacheConfig{
					URL: "https://example.com/.well-known/jwks.json",
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid - no key source configured",
			config:  &Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateToken(t *testing.T) {
	secret := "test-secret-key-for-testing"
	config := &Config{
		Secret: secret,
	}

	validator, err := NewValidator(config)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	t.Run("valid token", func(t *testing.T) {
		// Create a valid token
		claims := jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
			"iat": time.Now().Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		// Validate the token
		validatedToken, err := validator.ValidateToken(tokenString)
		if err != nil {
			t.Errorf("ValidateToken() error = %v, want nil", err)
		}

		if validatedToken == nil || !validatedToken.Valid {
			t.Error("Expected token to be valid")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		// Create an expired token
		claims := jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(-1 * time.Hour).Unix(),
			"iat": time.Now().Add(-2 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		// Validate the token
		_, err = validator.ValidateToken(tokenString)
		if err != ErrTokenExpired {
			t.Errorf("ValidateToken() error = %v, want ErrTokenExpired", err)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		// Create a token with wrong secret
		claims := jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte("wrong-secret"))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		// Validate the token
		_, err = validator.ValidateToken(tokenString)
		if err == nil {
			t.Error("Expected validation to fail for token with invalid signature")
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := validator.ValidateToken("not.a.valid.token")
		if err == nil {
			t.Error("Expected validation to fail for malformed token")
		}
	})

	t.Run("valid token with correct issuer", func(t *testing.T) {
		configWithIssuer := &Config{
			Secret: secret,
			Issuer: "https://auth.example.com",
		}
		validatorWithIssuer, err := NewValidator(configWithIssuer)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		claims := jwt.MapClaims{
			"sub": "user123",
			"iss": "https://auth.example.com",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		_, err = validatorWithIssuer.ValidateToken(tokenString)
		if err != nil {
			t.Errorf("ValidateToken() error = %v, want nil for correct issuer", err)
		}
	})

	t.Run("invalid issuer", func(t *testing.T) {
		configWithIssuer := &Config{
			Secret: secret,
			Issuer: "https://auth.example.com",
		}
		validatorWithIssuer, err := NewValidator(configWithIssuer)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		claims := jwt.MapClaims{
			"sub": "user123",
			"iss": "https://evil.com",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		_, err = validatorWithIssuer.ValidateToken(tokenString)
		if !errors.Is(err, ErrInvalidIssuer) {
			t.Errorf("ValidateToken() error = %v, want ErrInvalidIssuer", err)
		}
	})

	t.Run("missing issuer claim when required", func(t *testing.T) {
		configWithIssuer := &Config{
			Secret: secret,
			Issuer: "https://auth.example.com",
		}
		validatorWithIssuer, err := NewValidator(configWithIssuer)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		claims := jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		_, err = validatorWithIssuer.ValidateToken(tokenString)
		if !errors.Is(err, ErrInvalidIssuer) {
			t.Errorf("ValidateToken() error = %v, want ErrInvalidIssuer for missing issuer", err)
		}
	})

	t.Run("skip issuer validation", func(t *testing.T) {
		configSkipIssuer := &Config{
			Secret:     secret,
			Issuer:     "https://auth.example.com",
			SkipIssuer: true,
		}
		validatorSkipIssuer, err := NewValidator(configSkipIssuer)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		claims := jwt.MapClaims{
			"sub": "user123",
			"iss": "https://different.com",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		_, err = validatorSkipIssuer.ValidateToken(tokenString)
		if err != nil {
			t.Errorf("ValidateToken() error = %v, want nil when skipping issuer", err)
		}
	})

	t.Run("valid token with correct audience", func(t *testing.T) {
		configWithAudience := &Config{
			Secret:   secret,
			Audience: "https://api.example.com",
		}
		validatorWithAudience, err := NewValidator(configWithAudience)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		claims := jwt.MapClaims{
			"sub": "user123",
			"aud": "https://api.example.com",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		_, err = validatorWithAudience.ValidateToken(tokenString)
		if err != nil {
			t.Errorf("ValidateToken() error = %v, want nil for correct audience", err)
		}
	})

	t.Run("valid token with multiple audiences", func(t *testing.T) {
		configWithAudience := &Config{
			Secret:   secret,
			Audience: "https://api.example.com",
		}
		validatorWithAudience, err := NewValidator(configWithAudience)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		claims := jwt.MapClaims{
			"sub": "user123",
			"aud": []string{"https://api.example.com", "https://other.example.com"},
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		_, err = validatorWithAudience.ValidateToken(tokenString)
		if err != nil {
			t.Errorf("ValidateToken() error = %v, want nil for correct audience in list", err)
		}
	})

	t.Run("invalid audience", func(t *testing.T) {
		configWithAudience := &Config{
			Secret:   secret,
			Audience: "https://api.example.com",
		}
		validatorWithAudience, err := NewValidator(configWithAudience)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		claims := jwt.MapClaims{
			"sub": "user123",
			"aud": "https://evil.com",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		_, err = validatorWithAudience.ValidateToken(tokenString)
		if !errors.Is(err, ErrInvalidAudience) {
			t.Errorf("ValidateToken() error = %v, want ErrInvalidAudience", err)
		}
	})

	t.Run("missing audience claim when required", func(t *testing.T) {
		configWithAudience := &Config{
			Secret:   secret,
			Audience: "https://api.example.com",
		}
		validatorWithAudience, err := NewValidator(configWithAudience)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		claims := jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		_, err = validatorWithAudience.ValidateToken(tokenString)
		if !errors.Is(err, ErrInvalidAudience) {
			t.Errorf("ValidateToken() error = %v, want ErrInvalidAudience for missing audience", err)
		}
	})
}

func TestGetClaimString(t *testing.T) {
	claims := jwt.MapClaims{
		"string_claim": "test-value",
		"int_claim":    123,
		"bool_claim":   true,
	}

	tests := []struct {
		name      string
		key       string
		wantValue string
		wantOk    bool
	}{
		{
			name:      "existing string claim",
			key:       "string_claim",
			wantValue: "test-value",
			wantOk:    true,
		},
		{
			name:      "non-existent claim",
			key:       "missing_claim",
			wantValue: "",
			wantOk:    false,
		},
		{
			name:      "non-string claim",
			key:       "int_claim",
			wantValue: "",
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotOk := GetClaimString(claims, tt.key)
			if gotValue != tt.wantValue || gotOk != tt.wantOk {
				t.Errorf("GetClaimString() = (%q, %v), want (%q, %v)",
					gotValue, gotOk, tt.wantValue, tt.wantOk)
			}
		})
	}
}

func TestGetClaimInt64(t *testing.T) {
	claims := jwt.MapClaims{
		"int_claim":    123,
		"int64_claim":  int64(456),
		"float_claim":  float64(789),
		"string_claim": "not-an-int",
	}

	tests := []struct {
		name      string
		key       string
		wantValue int64
		wantOk    bool
	}{
		{
			name:      "int claim",
			key:       "int_claim",
			wantValue: 123,
			wantOk:    true,
		},
		{
			name:      "int64 claim",
			key:       "int64_claim",
			wantValue: 456,
			wantOk:    true,
		},
		{
			name:      "float64 claim",
			key:       "float_claim",
			wantValue: 789,
			wantOk:    true,
		},
		{
			name:      "non-existent claim",
			key:       "missing_claim",
			wantValue: 0,
			wantOk:    false,
		},
		{
			name:      "non-numeric claim",
			key:       "string_claim",
			wantValue: 0,
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotOk := GetClaimInt64(claims, tt.key)
			if gotValue != tt.wantValue || gotOk != tt.wantOk {
				t.Errorf("GetClaimInt64() = (%d, %v), want (%d, %v)",
					gotValue, gotOk, tt.wantValue, tt.wantOk)
			}
		})
	}
}

func TestGetClaimStringSlice(t *testing.T) {
	claims := jwt.MapClaims{
		"string_slice":    []string{"a", "b", "c"},
		"interface_slice": []interface{}{"x", "y", "z"},
		"mixed_slice":     []interface{}{"valid", 123, "another"},
		"string_claim":    "not-a-slice",
	}

	tests := []struct {
		name      string
		key       string
		wantValue []string
		wantOk    bool
	}{
		{
			name:      "string slice",
			key:       "string_slice",
			wantValue: []string{"a", "b", "c"},
			wantOk:    true,
		},
		{
			name:      "interface slice with strings",
			key:       "interface_slice",
			wantValue: []string{"x", "y", "z"},
			wantOk:    true,
		},
		{
			name:      "mixed interface slice",
			key:       "mixed_slice",
			wantValue: []string{"valid", "another"},
			wantOk:    true,
		},
		{
			name:      "non-existent claim",
			key:       "missing_claim",
			wantValue: nil,
			wantOk:    false,
		},
		{
			name:      "non-slice claim",
			key:       "string_claim",
			wantValue: nil,
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotOk := GetClaimStringSlice(claims, tt.key)
			if gotOk != tt.wantOk {
				t.Errorf("GetClaimStringSlice() ok = %v, want %v", gotOk, tt.wantOk)
			}
			if gotOk && !stringSlicesEqual(gotValue, tt.wantValue) {
				t.Errorf("GetClaimStringSlice() value = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestValidator_ValidateWithContext(t *testing.T) {
	secret := "test-secret-key-for-testing"
	config := &Config{
		Secret: secret,
	}

	validator, err := NewValidator(config)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	t.Run("valid token with context", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
			"iat": time.Now().Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		ctx := context.Background()
		validatedToken, err := validator.ValidateWithContext(ctx, tokenString)
		if err != nil {
			t.Errorf("ValidateWithContext() error = %v, want nil", err)
		}
		if validatedToken == nil || !validatedToken.Valid {
			t.Error("Expected token to be valid")
		}
	})

	t.Run("expired token with context", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(-1 * time.Hour).Unix(),
			"iat": time.Now().Add(-2 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		ctx := context.Background()
		_, err = validator.ValidateWithContext(ctx, tokenString)
		if err != ErrTokenExpired {
			t.Errorf("ValidateWithContext() error = %v, want ErrTokenExpired", err)
		}
	})
}

func TestValidator_ParseToken(t *testing.T) {
	secret := "test-secret-key-for-testing"
	config := &Config{
		Secret: secret,
	}

	validator, err := NewValidator(config)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	t.Run("parse valid token", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		parsedToken, err := validator.ParseToken(tokenString)
		if err != nil {
			t.Errorf("ParseToken() error = %v, want nil", err)
		}
		if parsedToken == nil {
			t.Error("Expected parsed token to be non-nil")
		}
	})

	t.Run("parse invalid token", func(t *testing.T) {
		_, err := validator.ParseToken("not.a.valid.token")
		if err == nil {
			t.Error("Expected error for invalid token")
		}
	})
}

func TestGetClaimTime(t *testing.T) {
	now := time.Now()
	claims := jwt.MapClaims{
		"exp": now.Unix(),
		"iat": float64(now.Unix()),
	}

	tests := []struct {
		name    string
		key     string
		wantOk  bool
		wantVal time.Time
	}{
		{
			name:    "existing time claim",
			key:     "exp",
			wantOk:  true,
			wantVal: time.Unix(now.Unix(), 0),
		},
		{
			name:   "non-existent claim",
			key:    "missing",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVal, gotOk := GetClaimTime(claims, tt.key)
			if gotOk != tt.wantOk {
				t.Errorf("GetClaimTime() ok = %v, want %v", gotOk, tt.wantOk)
			}
			if gotOk && gotVal.Unix() != tt.wantVal.Unix() {
				t.Errorf("GetClaimTime() value = %v, want %v", gotVal, tt.wantVal)
			}
		})
	}
}

func TestValidatorOptions(t *testing.T) {
	secret := "test-secret-key-for-testing"

	t.Run("WithSkipExpiry", func(t *testing.T) {
		config := &Config{
			Secret: secret,
		}

		validator, err := NewValidator(config, WithSkipExpiry())
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		if !validator.config.SkipExpiry {
			t.Error("Expected SkipExpiry to be true")
		}
	})

	t.Run("WithSkipIssuer", func(t *testing.T) {
		config := &Config{
			Secret: secret,
			Issuer: "https://auth.example.com",
		}

		validator, err := NewValidator(config, WithSkipIssuer())
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		if !validator.config.SkipIssuer {
			t.Error("Expected SkipIssuer to be true")
		}
	})
}

func TestValidator_isAllowedSigningMethod(t *testing.T) {
	tests := []struct {
		name              string
		allowedAlgorithms []string
		alg               string
		want              bool
	}{
		{
			name:              "allowed algorithm",
			allowedAlgorithms: []string{"HS256", "RS256"},
			alg:               "HS256",
			want:              true,
		},
		{
			name:              "disallowed algorithm",
			allowedAlgorithms: []string{"HS256", "RS256"},
			alg:               "ES256",
			want:              false,
		},
		{
			name:              "empty allowed algorithms - allow all",
			allowedAlgorithms: []string{},
			alg:               "HS256",
			want:              true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Secret:            "test-secret",
				AllowedAlgorithms: tt.allowedAlgorithms,
			}
			validator, err := NewValidator(config)
			if err != nil {
				t.Fatalf("Failed to create validator: %v", err)
			}

			got := validator.isAllowedSigningMethod(tt.alg)
			if got != tt.want {
				t.Errorf("isAllowedSigningMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractClaims(t *testing.T) {
	secret := "test-secret-key-for-testing"

	t.Run("extract from MapClaims", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub":  "user123",
			"name": "John Doe",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		config := &Config{
			Secret: secret,
		}
		validator, err := NewValidator(config)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		validatedToken, err := validator.ValidateToken(tokenString)
		if err != nil {
			t.Fatalf("Failed to validate token: %v", err)
		}

		extractedClaims, err := ExtractClaims(validatedToken)
		if err != nil {
			t.Errorf("ExtractClaims() error = %v", err)
		}
		if extractedClaims == nil {
			t.Error("Expected extracted claims to be non-nil")
		}
		if sub, ok := extractedClaims["sub"].(string); !ok || sub != "user123" {
			t.Error("Expected sub claim to be user123")
		}
	})
}

func TestNewValidator_WithOptions(t *testing.T) {
	secret := "test-secret-key-for-testing"
	blacklist := NewInMemoryBlacklist()
	defer blacklist.Stop()

	config := &Config{
		Secret: secret,
	}

	validator, err := NewValidator(config, WithBlacklist(blacklist))
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	if validator.blacklist == nil {
		t.Error("Expected validator to have blacklist configured")
	}
}

func TestValidator_JWKSOnlyConfig(t *testing.T) {
	// Create a test server that returns JWKS with RSA key
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := JWKSet{
			Keys: []JWK{
				{
					Kid: "test-rsa-key",
					Kty: "RSA",
					Alg: "RS256",
					Use: "sig",
					N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
					E:   "AQAB",
				},
			},
		}
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	// Create validator with JWKS-only configuration
	config := &Config{
		JWKSConfig: &JWKSCacheConfig{
			URL:         server.URL,
			RefreshTTL:  1 * time.Hour,
			AutoRefresh: false,
		},
	}

	validator, err := NewValidator(config)
	if err != nil {
		t.Fatalf("Failed to create validator with JWKS-only config: %v", err)
	}

	if validator.jwksCache == nil {
		t.Error("Expected validator to have JWKS cache configured")
	}

	// Verify JWKS cache is initialized and has the key
	if validator.jwksCache.KeyCount() != 1 {
		t.Errorf("Expected JWKS cache to have 1 key, got %d", validator.jwksCache.KeyCount())
	}

	// Verify we can get the key from cache
	key, err := validator.jwksCache.GetKey("test-rsa-key")
	if err != nil {
		t.Errorf("Failed to get key from JWKS cache: %v", err)
	}
	if key == nil {
		t.Error("Expected key to be non-nil")
	}
}
