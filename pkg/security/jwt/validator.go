// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package jwt provides JWT token validation and management for the Swit framework.
// It supports standard JWT validation, custom claims extraction, and multiple signing methods.
package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken indicates that the token is invalid.
	ErrInvalidToken = errors.New("jwt: invalid token")
	// ErrTokenExpired indicates that the token has expired.
	ErrTokenExpired = errors.New("jwt: token has expired")
	// ErrTokenNotYetValid indicates that the token is not yet valid.
	ErrTokenNotYetValid = errors.New("jwt: token not yet valid")
	// ErrInvalidSigningMethod indicates that the signing method is not supported.
	ErrInvalidSigningMethod = errors.New("jwt: invalid signing method")
)

// Validator provides JWT token validation functionality.
type Validator struct {
	config *Config
}

// NewValidator creates a new JWT validator with the given configuration.
func NewValidator(config *Config) (*Validator, error) {
	if config == nil {
		return nil, fmt.Errorf("jwt: config cannot be nil")
	}

	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("jwt: invalid config: %w", err)
	}

	return &Validator{
		config: config,
	}, nil
}

// ValidateToken validates a JWT token string and returns the parsed token.
func (v *Validator) ValidateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, v.keyFunc)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, ErrTokenNotYetValid
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return token, nil
}

// ValidateTokenWithClaims validates a JWT token and parses it into custom claims.
func (v *Validator) ValidateTokenWithClaims(tokenString string, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, claims, v.keyFunc)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return ErrTokenNotYetValid
		}
		return fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return ErrInvalidToken
	}

	return nil
}

// ParseToken parses a JWT token without validation.
// This should only be used for extracting claims from trusted tokens.
func (v *Validator) ParseToken(tokenString string) (*jwt.Token, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("jwt: failed to parse token: %w", err)
	}
	return token, nil
}

// keyFunc returns the key for validating the JWT token.
func (v *Validator) keyFunc(token *jwt.Token) (interface{}, error) {
	// Validate signing method
	if !v.isAllowedSigningMethod(token.Method.Alg()) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidSigningMethod, token.Method.Alg())
	}

	// Return the appropriate key based on signing method
	switch token.Method.(type) {
	case *jwt.SigningMethodHMAC:
		return []byte(v.config.Secret), nil
	case *jwt.SigningMethodRSA:
		if v.config.PublicKey != nil {
			return v.config.PublicKey, nil
		}
		return nil, fmt.Errorf("jwt: RSA public key not configured")
	case *jwt.SigningMethodECDSA:
		if v.config.PublicKey != nil {
			return v.config.PublicKey, nil
		}
		return nil, fmt.Errorf("jwt: ECDSA public key not configured")
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidSigningMethod, token.Method.Alg())
	}
}

// isAllowedSigningMethod checks if the signing method is allowed.
func (v *Validator) isAllowedSigningMethod(alg string) bool {
	if len(v.config.AllowedAlgorithms) == 0 {
		return true
	}

	for _, allowed := range v.config.AllowedAlgorithms {
		if alg == allowed {
			return true
		}
	}

	return false
}

// ExtractClaims extracts claims from a validated token.
func ExtractClaims(token *jwt.Token) (jwt.MapClaims, error) {
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return nil, fmt.Errorf("jwt: unable to extract claims")
}

// GetClaimString extracts a string claim from the token.
func GetClaimString(claims jwt.MapClaims, key string) (string, bool) {
	val, ok := claims[key]
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetClaimInt64 extracts an int64 claim from the token.
func GetClaimInt64(claims jwt.MapClaims, key string) (int64, bool) {
	val, ok := claims[key]
	if !ok {
		return 0, false
	}

	switch v := val.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

// GetClaimTime extracts a time claim from the token.
func GetClaimTime(claims jwt.MapClaims, key string) (time.Time, bool) {
	val, ok := GetClaimInt64(claims, key)
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(val, 0), true
}

// GetClaimStringSlice extracts a string slice claim from the token.
func GetClaimStringSlice(claims jwt.MapClaims, key string) ([]string, bool) {
	val, ok := claims[key]
	if !ok {
		return nil, false
	}

	switch v := val.(type) {
	case []string:
		return v, true
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result, true
	default:
		return nil, false
	}
}

// ValidatorOption is a functional option for configuring the validator.
type ValidatorOption func(*Validator)

// WithSkipExpiry skips expiry validation.
// This should only be used for testing purposes.
func WithSkipExpiry() ValidatorOption {
	return func(v *Validator) {
		v.config.SkipExpiry = true
	}
}

// WithSkipIssuer skips issuer validation.
// This should only be used for testing purposes.
func WithSkipIssuer() ValidatorOption {
	return func(v *Validator) {
		v.config.SkipIssuer = true
	}
}

// Config holds the JWT validator configuration.
type Config struct {
	// Secret is the HMAC secret key.
	Secret string `json:"-" yaml:"secret" mapstructure:"secret"`

	// PublicKey is the RSA/ECDSA public key for verification.
	PublicKey interface{} `json:"-" yaml:"-" mapstructure:"-"`

	// AllowedAlgorithms is the list of allowed signing algorithms.
	AllowedAlgorithms []string `json:"allowed_algorithms" yaml:"allowed_algorithms" mapstructure:"allowed_algorithms"`

	// Issuer is the expected token issuer.
	Issuer string `json:"issuer,omitempty" yaml:"issuer,omitempty" mapstructure:"issuer"`

	// Audience is the expected token audience.
	Audience string `json:"audience,omitempty" yaml:"audience,omitempty" mapstructure:"audience"`

	// SkipExpiry skips expiry validation (for testing).
	SkipExpiry bool `json:"skip_expiry,omitempty" yaml:"skip_expiry,omitempty" mapstructure:"skip_expiry"`

	// SkipIssuer skips issuer validation (for testing).
	SkipIssuer bool `json:"skip_issuer,omitempty" yaml:"skip_issuer,omitempty" mapstructure:"skip_issuer"`

	// LeewayDuration is the time leeway for validating time-based claims.
	LeewayDuration time.Duration `json:"leeway_duration" yaml:"leeway_duration" mapstructure:"leeway_duration"`
}

// SetDefaults sets default values for the configuration.
func (c *Config) SetDefaults() {
	if len(c.AllowedAlgorithms) == 0 {
		c.AllowedAlgorithms = []string{"HS256", "HS384", "HS512", "RS256", "RS384", "RS512", "ES256", "ES384", "ES512"}
	}

	if c.LeewayDuration <= 0 {
		c.LeewayDuration = 5 * time.Second
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Secret == "" && c.PublicKey == nil {
		return fmt.Errorf("jwt: either secret or public_key must be configured")
	}

	return nil
}
