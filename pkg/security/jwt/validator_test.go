// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jwt

import (
	"errors"
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
			name:    "invalid - no secret or public key",
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
