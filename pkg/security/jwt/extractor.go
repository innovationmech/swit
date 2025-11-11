// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jwt

import (
	"fmt"
	"net/http"
	"strings"
)

// TokenExtractor is a function that extracts a token from an HTTP request.
type TokenExtractor func(*http.Request) (string, error)

// FromAuthHeader extracts the token from the Authorization header.
// It expects the format: "Bearer <token>".
func FromAuthHeader() TokenExtractor {
	return func(r *http.Request) (string, error) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			return "", fmt.Errorf("jwt: authorization header not found")
		}

		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("jwt: invalid authorization header format")
		}

		if !strings.EqualFold(parts[0], "Bearer") {
			return "", fmt.Errorf("jwt: authorization scheme is not Bearer")
		}

		return parts[1], nil
	}
}

// FromHeader extracts the token from a custom header.
func FromHeader(name string) TokenExtractor {
	return func(r *http.Request) (string, error) {
		token := r.Header.Get(name)
		if token == "" {
			return "", fmt.Errorf("jwt: token not found in header %s", name)
		}
		return token, nil
	}
}

// FromQuery extracts the token from a query parameter.
func FromQuery(param string) TokenExtractor {
	return func(r *http.Request) (string, error) {
		token := r.URL.Query().Get(param)
		if token == "" {
			return "", fmt.Errorf("jwt: token not found in query parameter %s", param)
		}
		return token, nil
	}
}

// FromCookie extracts the token from a cookie.
func FromCookie(name string) TokenExtractor {
	return func(r *http.Request) (string, error) {
		cookie, err := r.Cookie(name)
		if err != nil {
			return "", fmt.Errorf("jwt: token not found in cookie %s: %w", name, err)
		}
		return cookie.Value, nil
	}
}

// FromFirst tries multiple extractors in order and returns the first successful result.
func FromFirst(extractors ...TokenExtractor) TokenExtractor {
	return func(r *http.Request) (string, error) {
		for _, extractor := range extractors {
			token, err := extractor(r)
			if err == nil {
				return token, nil
			}
		}
		return "", fmt.Errorf("jwt: token not found in any source")
	}
}

// MultiTokenExtractor combines multiple token extractors.
type MultiTokenExtractor struct {
	extractors []TokenExtractor
}

// NewMultiTokenExtractor creates a new multi-token extractor.
func NewMultiTokenExtractor(extractors ...TokenExtractor) *MultiTokenExtractor {
	return &MultiTokenExtractor{
		extractors: extractors,
	}
}

// Extract tries each extractor in order and returns the first successful token.
func (m *MultiTokenExtractor) Extract(r *http.Request) (string, error) {
	for _, extractor := range m.extractors {
		token, err := extractor(r)
		if err == nil {
			return token, nil
		}
	}
	return "", fmt.Errorf("jwt: token not found")
}

// DefaultExtractor returns a default token extractor that tries:
// 1. Authorization header (Bearer token)
// 2. X-Access-Token header
// 3. access_token query parameter
func DefaultExtractor() TokenExtractor {
	return FromFirst(
		FromAuthHeader(),
		FromHeader("X-Access-Token"),
		FromQuery("access_token"),
	)
}

// ExtractTokenFromRequest extracts a token from an HTTP request using the default extractor.
func ExtractTokenFromRequest(r *http.Request) (string, error) {
	return DefaultExtractor()(r)
}

// ExtractAndValidateToken extracts and validates a token from an HTTP request.
func (v *Validator) ExtractAndValidateToken(r *http.Request, extractor TokenExtractor) (*CustomClaims, error) {
	if extractor == nil {
		extractor = DefaultExtractor()
	}

	// Extract token
	tokenString, err := extractor(r)
	if err != nil {
		return nil, fmt.Errorf("jwt: failed to extract token: %w", err)
	}

	// Validate and parse token
	claims, err := v.ParseCustomClaims(tokenString)
	if err != nil {
		return nil, fmt.Errorf("jwt: failed to validate token: %w", err)
	}

	return claims, nil
}

// ExtractConfig configures the token extraction behavior.
type ExtractConfig struct {
	// AuthScheme is the authentication scheme (default: "Bearer").
	AuthScheme string

	// HeaderName is the name of the header containing the token.
	HeaderName string

	// QueryParam is the name of the query parameter containing the token.
	QueryParam string

	// CookieName is the name of the cookie containing the token.
	CookieName string

	// AllowQueryParam indicates whether to allow token in query parameters.
	AllowQueryParam bool

	// AllowCookie indicates whether to allow token in cookies.
	AllowCookie bool
}

// SetDefaults sets default values for the extraction configuration.
func (c *ExtractConfig) SetDefaults() {
	if c.AuthScheme == "" {
		c.AuthScheme = "Bearer"
	}
	if c.HeaderName == "" {
		c.HeaderName = "Authorization"
	}
	if c.QueryParam == "" {
		c.QueryParam = "access_token"
	}
	if c.CookieName == "" {
		c.CookieName = "access_token"
	}
}

// BuildExtractor builds a token extractor based on the configuration.
func (c *ExtractConfig) BuildExtractor() TokenExtractor {
	c.SetDefaults()

	extractors := make([]TokenExtractor, 0, 3)

	// Always include Authorization header
	extractors = append(extractors, FromAuthHeader())

	// Include query parameter if allowed
	if c.AllowQueryParam {
		extractors = append(extractors, FromQuery(c.QueryParam))
	}

	// Include cookie if allowed
	if c.AllowCookie {
		extractors = append(extractors, FromCookie(c.CookieName))
	}

	return FromFirst(extractors...)
}
