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
	// ErrInvalidIssuer indicates that the token issuer does not match the expected issuer.
	ErrInvalidIssuer = errors.New("jwt: invalid issuer")
	// ErrInvalidAudience indicates that the token audience does not match the expected audience.
	ErrInvalidAudience = errors.New("jwt: invalid audience")
)

// Validator provides JWT token validation functionality.
type Validator struct {
	config    *Config
	jwksCache *JWKSCache
	blacklist TokenBlacklist
}

// NewValidator creates a new JWT validator with the given configuration.
func NewValidator(config *Config, opts ...ValidatorOption) (*Validator, error) {
	if config == nil {
		return nil, fmt.Errorf("jwt: config cannot be nil")
	}

	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("jwt: invalid config: %w", err)
	}

	validator := &Validator{
		config: config,
	}

	// Apply options
	for _, opt := range opts {
		opt(validator)
	}

	// Initialize JWKS cache if configured
	if config.JWKSConfig != nil {
		cache, err := NewJWKSCache(config.JWKSConfig)
		if err != nil {
			return nil, fmt.Errorf("jwt: failed to initialize JWKS cache: %w", err)
		}
		validator.jwksCache = cache
	}

	return validator, nil
}

// ValidateToken validates a JWT token string and returns the parsed token.
func (v *Validator) ValidateToken(tokenString string) (*jwt.Token, error) {
	return v.ValidateWithContext(context.Background(), tokenString)
}

// ValidateWithContext validates a JWT token string with context and returns the parsed token.
func (v *Validator) ValidateWithContext(ctx context.Context, tokenString string) (*jwt.Token, error) {
	// Check blacklist if configured
	if v.blacklist != nil {
		blacklisted, err := v.blacklist.IsBlacklisted(ctx, tokenString)
		if err != nil {
			return nil, fmt.Errorf("jwt: failed to check blacklist: %w", err)
		}
		if blacklisted {
			return nil, fmt.Errorf("%w: token is blacklisted", ErrInvalidToken)
		}
	}

	// Parse and validate token with leeway
	parser := jwt.NewParser(jwt.WithLeeway(v.config.LeewayDuration))
	token, err := parser.Parse(tokenString, v.keyFunc)
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

	// Validate issuer and audience claims against configuration
	if err := v.validateClaims(token.Claims); err != nil {
		return nil, err
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

	// Validate issuer and audience claims against configuration
	if err := v.validateClaims(token.Claims); err != nil {
		return err
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
	case *jwt.SigningMethodRSA, *jwt.SigningMethodECDSA:
		// Try JWKS cache first if available
		if v.jwksCache != nil {
			kid, ok := token.Header["kid"].(string)
			if ok {
				key, err := v.jwksCache.GetKey(kid)
				if err == nil {
					return key, nil
				}
				// If key not found in cache, try to refresh and get again
				if err := v.jwksCache.Refresh(context.Background()); err == nil {
					if key, err := v.jwksCache.GetKey(kid); err == nil {
						return key, nil
					}
				}
			}
		}

		// Fallback to configured public key
		if v.config.PublicKey != nil {
			return v.config.PublicKey, nil
		}
		return nil, fmt.Errorf("jwt: public key not configured and not found in JWKS")
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

// validateClaims validates the issuer and audience claims against the configuration.
func (v *Validator) validateClaims(claims jwt.Claims) error {
	// Try to extract claims as MapClaims first
	var issuer string
	var audiences []string

	switch c := claims.(type) {
	case jwt.MapClaims:
		// Extract issuer
		if iss, ok := c["iss"].(string); ok {
			issuer = iss
		}

		// Extract audience (can be string or array)
		switch aud := c["aud"].(type) {
		case string:
			audiences = []string{aud}
		case []string:
			audiences = aud
		case []interface{}:
			for _, a := range aud {
				if audStr, ok := a.(string); ok {
					audiences = append(audiences, audStr)
				}
			}
		}

	case *jwt.RegisteredClaims:
		issuer = c.Issuer
		audiences = c.Audience

	default:
		// For other claim types, we can't validate
		// This is acceptable as not all tokens will have these claims
		return nil
	}

	// Validate issuer if configured and not skipped
	if v.config.Issuer != "" && !v.config.SkipIssuer {
		if issuer == "" {
			return fmt.Errorf("%w: missing issuer claim", ErrInvalidIssuer)
		}
		if issuer != v.config.Issuer {
			return fmt.Errorf("%w: expected %q, got %q", ErrInvalidIssuer, v.config.Issuer, issuer)
		}
	}

	// Validate audience if configured
	if v.config.Audience != "" {
		if len(audiences) == 0 {
			return fmt.Errorf("%w: missing audience claim", ErrInvalidAudience)
		}

		// Check if configured audience is in the token's audience list
		found := false
		for _, aud := range audiences {
			if aud == v.config.Audience {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: expected %q in audience, got %v", ErrInvalidAudience, v.config.Audience, audiences)
		}
	}

	return nil
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

	// JWKSConfig is the JWKS cache configuration (optional).
	JWKSConfig *JWKSCacheConfig `json:"jwks_config,omitempty" yaml:"jwks_config,omitempty" mapstructure:"jwks_config"`
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
	// At least one key source must be configured:
	// 1. Secret (for HMAC algorithms)
	// 2. PublicKey (for RSA/ECDSA with static key)
	// 3. JWKSConfig (for RSA/ECDSA with dynamic key discovery)
	if c.Secret == "" && c.PublicKey == nil && c.JWKSConfig == nil {
		return fmt.Errorf("jwt: at least one of secret, public_key, or jwks_config must be configured")
	}

	return nil
}
