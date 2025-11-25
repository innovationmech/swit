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

package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Token extends oauth2.Token with additional OIDC information.
type Token struct {
	*oauth2.Token
	IDToken string                 `json:"id_token,omitempty"`
	Claims  map[string]interface{} `json:"claims,omitempty"`
}

// TokenResponse represents the response from a token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// ParseToken parses a token response and extracts claims if OIDC is used.
func (c *Client) ParseToken(ctx context.Context, token *oauth2.Token) (*Token, error) {
	result := &Token{
		Token:  token,
		Claims: make(map[string]interface{}),
	}

	// Extract ID token if present
	idToken, ok := token.Extra("id_token").(string)
	if !ok || idToken == "" {
		return result, nil
	}
	result.IDToken = idToken

	// Verify and parse ID token if verifier is available
	if c.verifier != nil {
		verified, err := c.VerifyIDToken(ctx, idToken)
		if err != nil {
			return nil, fmt.Errorf("failed to verify ID token: %w", err)
		}

		// Extract standard claims
		if err := verified.Claims(&result.Claims); err != nil {
			return nil, fmt.Errorf("failed to parse ID token claims: %w", err)
		}
	}

	return result, nil
}

// GetClaim retrieves a claim value from the token.
func (t *Token) GetClaim(key string) (interface{}, bool) {
	if t.Claims == nil {
		return nil, false
	}
	val, ok := t.Claims[key]
	return val, ok
}

// GetStringClaim retrieves a string claim value from the token.
func (t *Token) GetStringClaim(key string) (string, bool) {
	val, ok := t.GetClaim(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetSubject returns the subject (user ID) from the token claims.
func (t *Token) GetSubject() string {
	sub, _ := t.GetStringClaim("sub")
	return sub
}

// GetEmail returns the email from the token claims.
func (t *Token) GetEmail() string {
	email, _ := t.GetStringClaim("email")
	return email
}

// GetName returns the name from the token claims.
func (t *Token) GetName() string {
	name, _ := t.GetStringClaim("name")
	return name
}

// GetPreferredUsername returns the preferred username from the token claims.
func (t *Token) GetPreferredUsername() string {
	username, _ := t.GetStringClaim("preferred_username")
	return username
}

// GetRoles returns the roles from the token claims.
// This method checks common claim names used by different providers.
func (t *Token) GetRoles() []string {
	// Try common role claim names
	claimNames := []string{"roles", "groups", "realm_access.roles"}

	for _, claimName := range claimNames {
		val, ok := t.GetClaim(claimName)
		if !ok {
			continue
		}

		switch v := val.(type) {
		case []string:
			return v
		case []interface{}:
			roles := make([]string, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					roles = append(roles, str)
				}
			}
			return roles
		case map[string]interface{}:
			// For Keycloak realm_access.roles structure
			if rolesVal, ok := v["roles"]; ok {
				if rolesList, ok := rolesVal.([]interface{}); ok {
					roles := make([]string, 0, len(rolesList))
					for _, item := range rolesList {
						if str, ok := item.(string); ok {
							roles = append(roles, str)
						}
					}
					return roles
				}
			}
		}
	}

	return nil
}

// TokenIntrospectionResponse represents the response from a token introspection endpoint.
type TokenIntrospectionResponse struct {
	Active     bool                   `json:"active"`
	Scope      string                 `json:"scope,omitempty"`
	ClientID   string                 `json:"client_id,omitempty"`
	Username   string                 `json:"username,omitempty"`
	TokenType  string                 `json:"token_type,omitempty"`
	Exp        int64                  `json:"exp,omitempty"`
	Iat        int64                  `json:"iat,omitempty"`
	Nbf        int64                  `json:"nbf,omitempty"`
	Sub        string                 `json:"sub,omitempty"`
	Aud        string                 `json:"aud,omitempty"`
	Iss        string                 `json:"iss,omitempty"`
	Jti        string                 `json:"jti,omitempty"`
	Extensions map[string]interface{} `json:"-"`
}

// IsExpired checks if the token is expired.
func (t *TokenIntrospectionResponse) IsExpired() bool {
	if t.Exp == 0 {
		return false
	}
	return time.Now().Unix() > t.Exp
}

// ExpiresAt returns the expiration time of the token.
func (t *TokenIntrospectionResponse) ExpiresAt() time.Time {
	if t.Exp == 0 {
		return time.Time{}
	}
	return time.Unix(t.Exp, 0)
}

// IssuedAt returns the issued time of the token.
func (t *TokenIntrospectionResponse) IssuedAt() time.Time {
	if t.Iat == 0 {
		return time.Time{}
	}
	return time.Unix(t.Iat, 0)
}

// NotBefore returns the not-before time of the token.
func (t *TokenIntrospectionResponse) NotBefore() time.Time {
	if t.Nbf == 0 {
		return time.Time{}
	}
	return time.Unix(t.Nbf, 0)
}

// StandardClaims represents standard JWT claims.
type StandardClaims struct {
	Issuer    string `json:"iss,omitempty"`
	Subject   string `json:"sub,omitempty"`
	Audience  string `json:"aud,omitempty"`
	ExpiresAt int64  `json:"exp,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
	NotBefore int64  `json:"nbf,omitempty"`
	JWTID     string `json:"jti,omitempty"`
}

// ParseIDTokenClaims parses an ID token and returns the standard claims.
func ParseIDTokenClaims(idToken *oidc.IDToken) (*StandardClaims, error) {
	var claims StandardClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse ID token claims: %w", err)
	}
	return &claims, nil
}

// ParseIDTokenCustomClaims parses an ID token into custom claims structure.
func ParseIDTokenCustomClaims(idToken *oidc.IDToken, claims interface{}) error {
	if err := idToken.Claims(claims); err != nil {
		return fmt.Errorf("failed to parse ID token claims: %w", err)
	}
	return nil
}

// ToJSON converts the token to JSON format.
func (t *Token) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}

// FromJSON parses a token from JSON format.
func FromJSON(data []byte) (*Token, error) {
	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token from JSON: %w", err)
	}
	return &token, nil
}

// TokenIntrospection represents the result of a token introspection request.
type TokenIntrospection struct {
	Active     bool                   `json:"active"`
	Scope      string                 `json:"scope,omitempty"`
	ClientID   string                 `json:"client_id,omitempty"`
	Username   string                 `json:"username,omitempty"`
	TokenType  string                 `json:"token_type,omitempty"`
	Exp        int64                  `json:"exp,omitempty"`
	Iat        int64                  `json:"iat,omitempty"`
	Nbf        int64                  `json:"nbf,omitempty"`
	Sub        string                 `json:"sub,omitempty"`
	Aud        string                 `json:"aud,omitempty"`
	Iss        string                 `json:"iss,omitempty"`
	Jti        string                 `json:"jti,omitempty"`
	Extensions map[string]interface{} `json:"-"`
}

// IsExpired checks if the introspected token is expired.
func (ti *TokenIntrospection) IsExpired() bool {
	if ti.Exp == 0 {
		return false
	}
	return time.Now().Unix() > ti.Exp
}

// ExpiresAt returns the expiration time of the introspected token.
func (ti *TokenIntrospection) ExpiresAt() time.Time {
	if ti.Exp == 0 {
		return time.Time{}
	}
	return time.Unix(ti.Exp, 0)
}

// IssuedAt returns the issued time of the introspected token.
func (ti *TokenIntrospection) IssuedAt() time.Time {
	if ti.Iat == 0 {
		return time.Time{}
	}
	return time.Unix(ti.Iat, 0)
}

// NotBefore returns the not-before time of the introspected token.
func (ti *TokenIntrospection) NotBefore() time.Time {
	if ti.Nbf == 0 {
		return time.Time{}
	}
	return time.Unix(ti.Nbf, 0)
}
