// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/golang-jwt/jwt/v5"
	secjwt "github.com/innovationmech/swit/pkg/security/jwt"
	"github.com/innovationmech/swit/pkg/security/oauth2"
)

// Context keys for gRPC authentication
type grpcContextKey string

const (
	// GRPCContextKeyUserInfo is the key for storing user info in context
	GRPCContextKeyUserInfo grpcContextKey = "grpc:auth:user_info"
	// GRPCContextKeyClaims is the key for storing JWT claims in context
	GRPCContextKeyClaims grpcContextKey = "grpc:auth:claims"
	// GRPCContextKeyTokenString is the key for storing raw token string in context
	GRPCContextKeyTokenString grpcContextKey = "grpc:auth:token_string"
)

// Default metadata keys for extracting tokens
const (
	// DefaultAuthorizationKey is the default metadata key for authorization
	DefaultAuthorizationKey = "authorization"
	// AlternativeAuthorizationKey is an alternative metadata key (lowercase)
	AlternativeAuthorizationKey = "Authorization"
)

// GRPCAuthConfig holds configuration for gRPC authentication interceptors
type GRPCAuthConfig struct {
	// OAuth2Client is the OAuth2 client used for token validation
	OAuth2Client *oauth2.Client

	// JWTValidator is the JWT validator used for local token validation
	JWTValidator *secjwt.Validator

	// UseIntrospection enables token introspection instead of local validation
	UseIntrospection bool

	// SkipMethods is a list of method names that should skip authentication
	// Format: "/package.service/method" or use "*" for wildcard
	SkipMethods []string

	// ErrorHandler is a custom error handler function
	ErrorHandler func(context.Context, error) error

	// TokenExtractor is a custom function to extract token from metadata
	TokenExtractor func(metadata.MD) (string, error)

	// Optional makes authentication optional (doesn't abort on missing/invalid token)
	Optional bool

	// MetadataKeys specifies custom metadata keys to search for the token
	// Default is ["authorization"]
	MetadataKeys []string
}

// GRPCUserInfo represents authenticated user information stored in gRPC context
type GRPCUserInfo struct {
	Subject  string                 `json:"sub"`
	Username string                 `json:"username,omitempty"`
	Email    string                 `json:"email,omitempty"`
	Roles    []string               `json:"roles,omitempty"`
	Scopes   []string               `json:"scopes,omitempty"`
	Claims   map[string]interface{} `json:"claims,omitempty"`
}

// UnaryAuthInterceptor creates a unary server interceptor for OAuth2/JWT authentication
func UnaryAuthInterceptor(client *oauth2.Client, validator *secjwt.Validator) grpc.UnaryServerInterceptor {
	config := &GRPCAuthConfig{
		OAuth2Client: client,
		JWTValidator: validator,
	}
	return UnaryAuthInterceptorWithConfig(config)
}

// UnaryAuthInterceptorWithConfig creates a unary server interceptor with custom configuration
func UnaryAuthInterceptorWithConfig(config *GRPCAuthConfig) grpc.UnaryServerInterceptor {
	if config == nil {
		panic("grpc-auth: interceptor config cannot be nil")
	}

	// Set defaults
	setGRPCAuthConfigDefaults(config)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Check if method should skip authentication
		if shouldSkipMethod(info.FullMethod, config.SkipMethods) {
			return handler(ctx, req)
		}

		// Extract metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			if config.Optional {
				return handler(ctx, req)
			}
			return nil, status.Error(codes.Unauthenticated, "grpc-auth: missing metadata")
		}

		// Extract token from metadata
		tokenString, err := config.TokenExtractor(md)
		if err != nil {
			if config.Optional {
				return handler(ctx, req)
			}
			return nil, mapAuthError(err, config)
		}

		// Validate token and build authenticated context
		authCtx, err := authenticateToken(ctx, config, tokenString)
		if err != nil {
			if config.Optional {
				return handler(ctx, req)
			}
			return nil, mapAuthError(err, config)
		}

		// Call handler with authenticated context
		return handler(authCtx, req)
	}
}

// StreamAuthInterceptor creates a stream server interceptor for OAuth2/JWT authentication
func StreamAuthInterceptor(client *oauth2.Client, validator *secjwt.Validator) grpc.StreamServerInterceptor {
	config := &GRPCAuthConfig{
		OAuth2Client: client,
		JWTValidator: validator,
	}
	return StreamAuthInterceptorWithConfig(config)
}

// StreamAuthInterceptorWithConfig creates a stream server interceptor with custom configuration
func StreamAuthInterceptorWithConfig(config *GRPCAuthConfig) grpc.StreamServerInterceptor {
	if config == nil {
		panic("grpc-auth: interceptor config cannot be nil")
	}

	// Set defaults
	setGRPCAuthConfigDefaults(config)

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Check if method should skip authentication
		if shouldSkipMethod(info.FullMethod, config.SkipMethods) {
			return handler(srv, ss)
		}

		// Extract metadata from stream context
		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			if config.Optional {
				return handler(srv, ss)
			}
			return status.Error(codes.Unauthenticated, "grpc-auth: missing metadata")
		}

		// Extract token from metadata
		tokenString, err := config.TokenExtractor(md)
		if err != nil {
			if config.Optional {
				return handler(srv, ss)
			}
			return mapAuthError(err, config)
		}

		// Validate token and build authenticated context
		authCtx, err := authenticateToken(ctx, config, tokenString)
		if err != nil {
			if config.Optional {
				return handler(srv, ss)
			}
			return mapAuthError(err, config)
		}

		// Create wrapped stream with authenticated context
		wrappedStream := &authenticatedServerStream{
			ServerStream: ss,
			ctx:          authCtx,
		}

		// Call handler with authenticated stream
		return handler(srv, wrappedStream)
	}
}

// authenticatedServerStream wraps grpc.ServerStream with an authenticated context
type authenticatedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context with authentication information
func (s *authenticatedServerStream) Context() context.Context {
	return s.ctx
}

// UnaryRequireRoles creates a unary interceptor that requires specific roles
func UnaryRequireRoles(roles ...string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		userInfo, err := GetGRPCUserInfo(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "grpc-auth: user not authenticated")
		}

		if !hasAnyRole(userInfo.Roles, roles) {
			return nil, status.Errorf(codes.PermissionDenied,
				"grpc-auth: user lacks required roles: %v", roles)
		}

		return handler(ctx, req)
	}
}

// StreamRequireRoles creates a stream interceptor that requires specific roles
func StreamRequireRoles(roles ...string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		userInfo, err := GetGRPCUserInfo(ctx)
		if err != nil {
			return status.Error(codes.Unauthenticated, "grpc-auth: user not authenticated")
		}

		if !hasAnyRole(userInfo.Roles, roles) {
			return status.Errorf(codes.PermissionDenied,
				"grpc-auth: user lacks required roles: %v", roles)
		}

		return handler(srv, ss)
	}
}

// UnaryRequireScopes creates a unary interceptor that requires specific scopes
func UnaryRequireScopes(scopes ...string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		userInfo, err := GetGRPCUserInfo(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "grpc-auth: user not authenticated")
		}

		if !hasAllScopes(userInfo.Scopes, scopes) {
			return nil, status.Errorf(codes.PermissionDenied,
				"grpc-auth: user lacks required scopes: %v", scopes)
		}

		return handler(ctx, req)
	}
}

// StreamRequireScopes creates a stream interceptor that requires specific scopes
func StreamRequireScopes(scopes ...string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		userInfo, err := GetGRPCUserInfo(ctx)
		if err != nil {
			return status.Error(codes.Unauthenticated, "grpc-auth: user not authenticated")
		}

		if !hasAllScopes(userInfo.Scopes, scopes) {
			return status.Errorf(codes.PermissionDenied,
				"grpc-auth: user lacks required scopes: %v", scopes)
		}

		return handler(srv, ss)
	}
}

// authenticateToken validates the token and returns an authenticated context
func authenticateToken(ctx context.Context, config *GRPCAuthConfig, tokenString string) (context.Context, error) {
	if config.UseIntrospection {
		return authenticateWithIntrospection(ctx, config, tokenString)
	}
	return authenticateLocally(ctx, config, tokenString)
}

// authenticateWithIntrospection validates token using OAuth2 introspection
func authenticateWithIntrospection(ctx context.Context, config *GRPCAuthConfig, tokenString string) (context.Context, error) {
	if config.OAuth2Client == nil {
		return nil, errors.New("grpc-auth: OAuth2 client not configured for introspection")
	}

	introspection, err := config.OAuth2Client.IntrospectToken(ctx, tokenString)
	if err != nil {
		return nil, fmt.Errorf("grpc-auth: token introspection failed: %w", err)
	}

	if !introspection.Active {
		return nil, errors.New("grpc-auth: token is not active")
	}

	if introspection.IsExpired() {
		return nil, errors.New("grpc-auth: token has expired")
	}

	// Build user info from introspection response
	userInfo := &GRPCUserInfo{
		Subject:  introspection.Sub,
		Username: introspection.Username,
		Claims:   make(map[string]interface{}),
	}

	// Parse scopes (OAuth2 scopes are space-delimited)
	if introspection.Scope != "" {
		userInfo.Scopes = splitScopes(introspection.Scope)
	}

	// Build claims map
	claims := jwt.MapClaims{
		"sub":        introspection.Sub,
		"username":   introspection.Username,
		"client_id":  introspection.ClientID,
		"token_type": introspection.TokenType,
		"iss":        introspection.Iss,
		"aud":        introspection.Aud,
		"exp":        introspection.Exp,
		"iat":        introspection.Iat,
		"nbf":        introspection.Nbf,
		"jti":        introspection.Jti,
		"scope":      introspection.Scope,
	}

	// Add extensions to claims and user info
	for key, value := range introspection.Extensions {
		claims[key] = value
		userInfo.Claims[key] = value
	}

	// Store in context
	ctx = context.WithValue(ctx, GRPCContextKeyUserInfo, userInfo)
	ctx = context.WithValue(ctx, GRPCContextKeyClaims, claims)
	ctx = context.WithValue(ctx, GRPCContextKeyTokenString, tokenString)

	return ctx, nil
}

// authenticateLocally validates token using local JWT validation
func authenticateLocally(ctx context.Context, config *GRPCAuthConfig, tokenString string) (context.Context, error) {
	if config.JWTValidator == nil {
		return nil, errors.New("grpc-auth: JWT validator not configured")
	}

	token, err := config.JWTValidator.ValidateWithContext(ctx, tokenString)
	if err != nil {
		return nil, fmt.Errorf("grpc-auth: token validation failed: %w", err)
	}

	// Extract claims
	claims, err := secjwt.ExtractClaims(token)
	if err != nil {
		return nil, fmt.Errorf("grpc-auth: failed to extract claims: %w", err)
	}

	// Convert to custom claims for easier access
	customClaims, err := secjwt.MapClaimsToCustomClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("grpc-auth: failed to convert claims: %w", err)
	}

	// Build user info from claims
	userInfo := &GRPCUserInfo{
		Subject:  customClaims.Subject,
		Username: customClaims.Username,
		Email:    customClaims.Email,
		Roles:    customClaims.Roles,
		Scopes:   expandScopes(customClaims.Permissions),
		Claims:   customClaims.Metadata,
	}

	// Store in context
	ctx = context.WithValue(ctx, GRPCContextKeyUserInfo, userInfo)
	ctx = context.WithValue(ctx, GRPCContextKeyClaims, claims)
	ctx = context.WithValue(ctx, GRPCContextKeyTokenString, tokenString)

	return ctx, nil
}

// defaultGRPCTokenExtractor extracts bearer token from gRPC metadata
func defaultGRPCTokenExtractor(metadataKeys []string) func(metadata.MD) (string, error) {
	return func(md metadata.MD) (string, error) {
		// Try each configured metadata key
		for _, key := range metadataKeys {
			// Try lowercase version first (gRPC metadata keys are case-insensitive but stored lowercase)
			values := md.Get(strings.ToLower(key))
			if len(values) == 0 {
				// Try original case
				values = md.Get(key)
			}

			if len(values) > 0 {
				// Extract bearer token from the first value
				token, err := extractBearerToken(values[0])
				if err == nil {
					return token, nil
				}
			}
		}

		return "", errors.New("grpc-auth: missing authentication token in metadata")
	}
}

// setGRPCAuthConfigDefaults sets default values for gRPC auth config
func setGRPCAuthConfigDefaults(config *GRPCAuthConfig) {
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultGRPCErrorHandler
	}

	if config.TokenExtractor == nil {
		if len(config.MetadataKeys) == 0 {
			config.MetadataKeys = []string{DefaultAuthorizationKey}
		}
		config.TokenExtractor = defaultGRPCTokenExtractor(config.MetadataKeys)
	}
}

// shouldSkipMethod checks if the method should skip authentication
func shouldSkipMethod(fullMethod string, skipMethods []string) bool {
	for _, skipMethod := range skipMethods {
		if skipMethod == fullMethod {
			return true
		}
		// Support wildcard matching
		if strings.HasSuffix(skipMethod, "*") {
			prefix := strings.TrimSuffix(skipMethod, "*")
			if strings.HasPrefix(fullMethod, prefix) {
				return true
			}
		}
	}
	return false
}

// defaultGRPCErrorHandler maps authentication errors to gRPC status codes
func defaultGRPCErrorHandler(ctx context.Context, err error) error {
	return mapAuthError(err, nil)
}

// mapAuthError maps authentication errors to appropriate gRPC status codes
func mapAuthError(err error, config *GRPCAuthConfig) error {
	if err == nil {
		return nil
	}

	// Check for custom error handler
	if config != nil && config.ErrorHandler != nil {
		return config.ErrorHandler(context.Background(), err)
	}

	// Map common JWT errors to gRPC status codes
	errMsg := err.Error()

	// Token expired
	if errors.Is(err, secjwt.ErrTokenExpired) || strings.Contains(errMsg, "expired") {
		return status.Error(codes.Unauthenticated, "grpc-auth: token has expired")
	}

	// Token not yet valid
	if errors.Is(err, secjwt.ErrTokenNotYetValid) || strings.Contains(errMsg, "not yet valid") {
		return status.Error(codes.Unauthenticated, "grpc-auth: token is not yet valid")
	}

	// Invalid token
	if errors.Is(err, secjwt.ErrInvalidToken) || strings.Contains(errMsg, "invalid") {
		return status.Error(codes.Unauthenticated, "grpc-auth: invalid token")
	}

	// Invalid issuer
	if errors.Is(err, secjwt.ErrInvalidIssuer) || strings.Contains(errMsg, "issuer") {
		return status.Error(codes.Unauthenticated, "grpc-auth: invalid token issuer")
	}

	// Invalid audience
	if errors.Is(err, secjwt.ErrInvalidAudience) || strings.Contains(errMsg, "audience") {
		return status.Error(codes.Unauthenticated, "grpc-auth: invalid token audience")
	}

	// Missing token
	if strings.Contains(errMsg, "missing") {
		return status.Error(codes.Unauthenticated, "grpc-auth: missing authentication token")
	}

	// Token not active
	if strings.Contains(errMsg, "not active") {
		return status.Error(codes.Unauthenticated, "grpc-auth: token is not active")
	}

	// Default to Unauthenticated for other errors
	return status.Errorf(codes.Unauthenticated, "grpc-auth: authentication failed: %v", err)
}

// GetGRPCUserInfo retrieves user information from gRPC context
func GetGRPCUserInfo(ctx context.Context) (*GRPCUserInfo, error) {
	userInfo := ctx.Value(GRPCContextKeyUserInfo)
	if userInfo == nil {
		return nil, errors.New("grpc-auth: user info not found in context")
	}

	info, ok := userInfo.(*GRPCUserInfo)
	if !ok {
		return nil, errors.New("grpc-auth: invalid user info type in context")
	}

	return info, nil
}

// GetGRPCClaims retrieves JWT claims from gRPC context
func GetGRPCClaims(ctx context.Context) (jwt.MapClaims, error) {
	claims := ctx.Value(GRPCContextKeyClaims)
	if claims == nil {
		return nil, errors.New("grpc-auth: claims not found in context")
	}

	mapClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("grpc-auth: invalid claims type in context")
	}

	return mapClaims, nil
}

// GetGRPCTokenString retrieves the raw token string from gRPC context
func GetGRPCTokenString(ctx context.Context) (string, error) {
	token := ctx.Value(GRPCContextKeyTokenString)
	if token == nil {
		return "", errors.New("grpc-auth: token string not found in context")
	}

	tokenStr, ok := token.(string)
	if !ok {
		return "", errors.New("grpc-auth: invalid token string type in context")
	}

	return tokenStr, nil
}

// MustGetGRPCUserInfo retrieves user information from context or panics
func MustGetGRPCUserInfo(ctx context.Context) *GRPCUserInfo {
	userInfo, err := GetGRPCUserInfo(ctx)
	if err != nil {
		panic(err)
	}
	return userInfo
}

// MustGetGRPCClaims retrieves claims from context or panics
func MustGetGRPCClaims(ctx context.Context) jwt.MapClaims {
	claims, err := GetGRPCClaims(ctx)
	if err != nil {
		panic(err)
	}
	return claims
}
