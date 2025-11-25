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
	"fmt"
	"strings"
)

// ProviderType represents the OAuth2/OIDC provider type.
type ProviderType string

const (
	// ProviderKeycloak represents Keycloak as the OAuth2 provider.
	ProviderKeycloak ProviderType = "keycloak"

	// ProviderAuth0 represents Auth0 as the OAuth2 provider.
	ProviderAuth0 ProviderType = "auth0"

	// ProviderGoogle represents Google as the OAuth2 provider.
	ProviderGoogle ProviderType = "google"

	// ProviderMicrosoft represents Microsoft/Azure AD as the OAuth2 provider.
	ProviderMicrosoft ProviderType = "microsoft"

	// ProviderOkta represents Okta as the OAuth2 provider.
	ProviderOkta ProviderType = "okta"

	// ProviderCustom represents a custom OAuth2 provider.
	ProviderCustom ProviderType = "custom"
)

// ProviderConfig holds provider-specific configuration and defaults.
type ProviderConfig struct {
	// Type is the provider type.
	Type ProviderType

	// Name is the human-readable provider name.
	Name string

	// SupportsDiscovery indicates whether the provider supports OIDC discovery.
	SupportsDiscovery bool

	// DefaultScopes are the default OAuth2 scopes for this provider.
	DefaultScopes []string

	// RequiredScopes are the required scopes that must be included.
	RequiredScopes []string

	// TokenEndpointAuthMethod is the authentication method for the token endpoint.
	// Common values: "client_secret_basic", "client_secret_post", "private_key_jwt"
	TokenEndpointAuthMethod string

	// UserInfoEndpointSupported indicates whether the provider supports the UserInfo endpoint.
	UserInfoEndpointSupported bool

	// RolesClaimName is the claim name used for roles/groups in tokens.
	RolesClaimName string

	// IssuerURLTemplate is a template for constructing the issuer URL.
	// Placeholders: {realm}, {tenant}, {domain}
	IssuerURLTemplate string

	// IntrospectionEndpoint is the URL of the token introspection endpoint (if not using discovery).
	IntrospectionEndpoint string

	// RevocationEndpoint is the URL of the token revocation endpoint (if not using discovery).
	RevocationEndpoint string
}

// GetProviderConfig returns the provider configuration for the given provider type.
func GetProviderConfig(providerType string) (*ProviderConfig, error) {
	provider := ProviderType(strings.ToLower(providerType))

	switch provider {
	case ProviderKeycloak:
		return &ProviderConfig{
			Type:                      ProviderKeycloak,
			Name:                      "Keycloak",
			SupportsDiscovery:         true,
			DefaultScopes:             []string{"openid", "profile", "email"},
			RequiredScopes:            []string{"openid"},
			TokenEndpointAuthMethod:   "client_secret_basic",
			UserInfoEndpointSupported: true,
			RolesClaimName:            "realm_access.roles",
			IssuerURLTemplate:         "{base_url}/realms/{realm}",
		}, nil

	case ProviderAuth0:
		return &ProviderConfig{
			Type:                      ProviderAuth0,
			Name:                      "Auth0",
			SupportsDiscovery:         true,
			DefaultScopes:             []string{"openid", "profile", "email"},
			RequiredScopes:            []string{"openid"},
			TokenEndpointAuthMethod:   "client_secret_post",
			UserInfoEndpointSupported: true,
			RolesClaimName:            "roles",
			IssuerURLTemplate:         "https://{domain}/",
		}, nil

	case ProviderGoogle:
		return &ProviderConfig{
			Type:                      ProviderGoogle,
			Name:                      "Google",
			SupportsDiscovery:         true,
			DefaultScopes:             []string{"openid", "profile", "email"},
			RequiredScopes:            []string{"openid"},
			TokenEndpointAuthMethod:   "client_secret_post",
			UserInfoEndpointSupported: true,
			RolesClaimName:            "groups",
			IssuerURLTemplate:         "https://accounts.google.com",
		}, nil

	case ProviderMicrosoft:
		return &ProviderConfig{
			Type:                      ProviderMicrosoft,
			Name:                      "Microsoft/Azure AD",
			SupportsDiscovery:         true,
			DefaultScopes:             []string{"openid", "profile", "email"},
			RequiredScopes:            []string{"openid"},
			TokenEndpointAuthMethod:   "client_secret_post",
			UserInfoEndpointSupported: true,
			RolesClaimName:            "roles",
			IssuerURLTemplate:         "https://login.microsoftonline.com/{tenant}/v2.0",
		}, nil

	case ProviderOkta:
		return &ProviderConfig{
			Type:                      ProviderOkta,
			Name:                      "Okta",
			SupportsDiscovery:         true,
			DefaultScopes:             []string{"openid", "profile", "email"},
			RequiredScopes:            []string{"openid"},
			TokenEndpointAuthMethod:   "client_secret_basic",
			UserInfoEndpointSupported: true,
			RolesClaimName:            "groups",
			IssuerURLTemplate:         "https://{domain}/oauth2/default",
		}, nil

	case ProviderCustom, "":
		return &ProviderConfig{
			Type:                      ProviderCustom,
			Name:                      "Custom",
			SupportsDiscovery:         false,
			DefaultScopes:             []string{"openid", "profile", "email"},
			RequiredScopes:            []string{},
			TokenEndpointAuthMethod:   "client_secret_basic",
			UserInfoEndpointSupported: false,
			RolesClaimName:            "roles",
			IssuerURLTemplate:         "",
		}, nil

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// IsProviderSupported checks if the given provider type is supported.
func IsProviderSupported(providerType string) bool {
	_, err := GetProviderConfig(providerType)
	return err == nil
}

// GetSupportedProviders returns a list of all supported provider types.
func GetSupportedProviders() []ProviderType {
	return []ProviderType{
		ProviderKeycloak,
		ProviderAuth0,
		ProviderGoogle,
		ProviderMicrosoft,
		ProviderOkta,
		ProviderCustom,
	}
}

// ApplyProviderDefaults applies provider-specific defaults to the configuration.
func (c *Config) ApplyProviderDefaults() error {
	if c.Provider == "" {
		c.Provider = string(ProviderCustom)
	}

	providerConfig, err := GetProviderConfig(c.Provider)
	if err != nil {
		return err
	}

	// Apply default scopes if not set
	if len(c.Scopes) == 0 {
		c.Scopes = providerConfig.DefaultScopes
	} else {
		// Ensure required scopes are included
		c.Scopes = ensureRequiredScopes(c.Scopes, providerConfig.RequiredScopes)
	}

	// Enable discovery if provider supports it and IssuerURL is set
	if providerConfig.SupportsDiscovery && c.IssuerURL != "" {
		c.UseDiscovery = true
	}

	return nil
}

// ensureRequiredScopes ensures that all required scopes are included in the scopes list.
func ensureRequiredScopes(scopes, requiredScopes []string) []string {
	scopeMap := make(map[string]bool)
	for _, scope := range scopes {
		scopeMap[scope] = true
	}

	for _, requiredScope := range requiredScopes {
		if !scopeMap[requiredScope] {
			scopes = append(scopes, requiredScope)
		}
	}

	return scopes
}

// GetRolesClaimName returns the claim name used for roles/groups for the configured provider.
func (c *Config) GetRolesClaimName() string {
	providerConfig, err := GetProviderConfig(c.Provider)
	if err != nil {
		return "roles" // Default fallback
	}
	return providerConfig.RolesClaimName
}

// ProviderMetadata holds additional provider metadata.
type ProviderMetadata struct {
	// Issuer is the provider's issuer URL.
	Issuer string

	// AuthorizationEndpoint is the URL of the authorization endpoint.
	AuthorizationEndpoint string

	// TokenEndpoint is the URL of the token endpoint.
	TokenEndpoint string

	// UserInfoEndpoint is the URL of the UserInfo endpoint.
	UserInfoEndpoint string

	// JWKSEndpoint is the URL of the JWKS endpoint.
	JWKSEndpoint string

	// IntrospectionEndpoint is the URL of the token introspection endpoint.
	IntrospectionEndpoint string

	// RevocationEndpoint is the URL of the token revocation endpoint.
	RevocationEndpoint string

	// EndSessionEndpoint is the URL of the end session endpoint (logout).
	EndSessionEndpoint string

	// ScopesSupported lists the OAuth2 scopes supported by the provider.
	ScopesSupported []string

	// ResponseTypesSupported lists the OAuth2 response types supported.
	ResponseTypesSupported []string

	// GrantTypesSupported lists the OAuth2 grant types supported.
	GrantTypesSupported []string

	// TokenEndpointAuthMethodsSupported lists the token endpoint auth methods supported.
	TokenEndpointAuthMethodsSupported []string

	// CodeChallengeMethodsSupported lists the PKCE code challenge methods supported.
	CodeChallengeMethodsSupported []string
}
