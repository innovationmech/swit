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

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/innovationmech/swit/pkg/saga/security"
)

// Example demonstrates how to use the Saga authentication middleware

func main() {
	// Example 1: JWT Authentication
	jwtExample()

	// Example 2: API Key Authentication
	apiKeyExample()

	// Example 3: Combined Authentication with AuthManager
	authManagerExample()

	// Example 4: Using Authentication in Saga Operations
	sagaOperationExample()
}

func jwtExample() {
	fmt.Println("=== JWT Authentication Example ===")

	// Create JWT auth provider
	jwtProvider, err := security.NewJWTAuthProvider(&security.JWTAuthProviderConfig{
		Secret:       "my-secret-key",
		Issuer:       "swit-saga",
		Audience:     "swit-services",
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
	})
	if err != nil {
		log.Fatalf("Failed to create JWT provider: %v", err)
	}

	// Create JWT credentials
	// In a real scenario, this token would come from your authentication service
	credentials := &security.AuthCredentials{
		Type:  security.AuthTypeJWT,
		Token: "your-jwt-token-here", // This should be a valid JWT token
	}

	// Authenticate
	ctx := context.Background()
	authCtx, err := jwtProvider.Authenticate(ctx, credentials)
	if err != nil {
		log.Printf("JWT Authentication failed: %v", err)
		return
	}

	fmt.Printf("JWT Authentication successful!\n")
	fmt.Printf("User ID: %s\n", authCtx.Credentials.UserID)
	fmt.Printf("Scopes: %v\n", authCtx.Credentials.Scopes)
	fmt.Printf("Timestamp: %v\n\n", authCtx.Timestamp)
}

func apiKeyExample() {
	fmt.Println("=== API Key Authentication Example ===")

	// Create API Key auth provider with predefined keys
	apiKeyProvider, err := security.NewAPIKeyAuthProvider(&security.APIKeyAuthProviderConfig{
		APIKeys: map[string]string{
			"key-12345": "user-alice",
			"key-67890": "user-bob",
		},
		CacheEnabled: true,
		CacheTTL:     10 * time.Minute,
	})
	if err != nil {
		log.Fatalf("Failed to create API Key provider: %v", err)
	}

	// Create API Key credentials
	credentials := &security.AuthCredentials{
		Type:   security.AuthTypeAPIKey,
		APIKey: "key-12345",
	}

	// Authenticate
	ctx := context.Background()
	authCtx, err := apiKeyProvider.Authenticate(ctx, credentials)
	if err != nil {
		log.Printf("API Key Authentication failed: %v", err)
		return
	}

	fmt.Printf("API Key Authentication successful!\n")
	fmt.Printf("User ID: %s\n", authCtx.Credentials.UserID)
	fmt.Printf("Scopes: %v\n", authCtx.Credentials.Scopes)
	fmt.Printf("API Key: %s\n\n", authCtx.Credentials.APIKey)
}

func authManagerExample() {
	fmt.Println("=== Auth Manager Example (Multiple Providers) ===")

	// Create JWT provider
	jwtProvider, err := security.NewJWTAuthProvider(&security.JWTAuthProviderConfig{
		Secret:       "my-secret-key",
		Issuer:       "swit-saga",
		Audience:     "swit-services",
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
	})
	if err != nil {
		log.Fatalf("Failed to create JWT provider: %v", err)
	}

	// Create API Key provider
	apiKeyProvider, err := security.NewAPIKeyAuthProvider(&security.APIKeyAuthProviderConfig{
		APIKeys: map[string]string{
			"key-12345": "user-alice",
			"key-67890": "user-bob",
		},
		CacheEnabled: true,
		CacheTTL:     10 * time.Minute,
	})
	if err != nil {
		log.Fatalf("Failed to create API Key provider: %v", err)
	}

	// Create Auth Manager with both providers
	authManager, err := security.NewAuthManager(&security.AuthManagerConfig{
		DefaultProvider: "jwt",
		Providers: map[string]security.AuthProvider{
			"jwt":    jwtProvider,
			"apikey": apiKeyProvider,
		},
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
	})
	if err != nil {
		log.Fatalf("Failed to create auth manager: %v", err)
	}

	ctx := context.Background()

	// Authenticate with API Key
	apiKeyCredentials := &security.AuthCredentials{
		Type:   security.AuthTypeAPIKey,
		APIKey: "key-12345",
	}

	authCtx, err := authManager.Authenticate(ctx, apiKeyCredentials)
	if err != nil {
		log.Printf("Authentication failed: %v", err)
		return
	}

	fmt.Printf("Authentication successful via Auth Manager!\n")
	fmt.Printf("User ID: %s\n", authCtx.Credentials.UserID)
	fmt.Printf("Auth Type: %s\n", authCtx.Credentials.Type)

	// Check health
	if authManager.IsHealthy(ctx) {
		fmt.Println("Auth Manager is healthy!")
	}
}

func sagaOperationExample() {
	fmt.Println("=== Using Authentication in Saga Operations ===")

	// Create authentication providers
	jwtProvider, err := security.NewJWTAuthProvider(&security.JWTAuthProviderConfig{
		Secret:       "my-secret-key",
		Issuer:       "swit-saga",
		Audience:     "swit-services",
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
	})
	if err != nil {
		log.Fatalf("Failed to create JWT provider: %v", err)
	}

	// Create Auth Manager
	authManager, err := security.NewAuthManager(&security.AuthManagerConfig{
		DefaultProvider: "jwt",
		Providers: map[string]security.AuthProvider{
			"jwt": jwtProvider,
		},
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
	})
	if err != nil {
		log.Fatalf("Failed to create auth manager: %v", err)
	}

	// Create Saga Auth Middleware
	sagaAuthMiddleware := security.NewSagaAuthMiddleware(&security.SagaAuthMiddlewareConfig{
		AuthManager: authManager,
		Required:    true, // Authentication is required for saga operations
	})

	// Create API Key credentials for demonstration
	apiKeyProvider, _ := security.NewAPIKeyAuthProvider(&security.APIKeyAuthProviderConfig{
		APIKeys: map[string]string{
			"saga-key-12345": "user-saga-operator",
		},
	})
	_ = authManager.AddProvider("apikey", apiKeyProvider)

	credentials := &security.AuthCredentials{
		Type:   security.AuthTypeAPIKey,
		APIKey: "saga-key-12345",
	}

	ctx := context.Background()

	// Step 1: Authenticate the saga operation
	authCtx, err := sagaAuthMiddleware.Authenticate(ctx, credentials)
	if err != nil {
		log.Printf("Saga authentication failed: %v", err)
		return
	}

	fmt.Printf("Saga operation authenticated successfully!\n")
	fmt.Printf("User ID: %s\n", authCtx.Credentials.UserID)

	// Step 2: Add auth context to the operation context
	ctx = security.ContextWithAuth(ctx, authCtx)

	// Step 3: Check required scope before executing saga steps
	requiredScope := "saga:execute"
	if err := sagaAuthMiddleware.RequireScope(authCtx, requiredScope); err != nil {
		log.Printf("Insufficient scope: %v", err)
		return
	}

	fmt.Printf("User has required scope: %s\n", requiredScope)

	// Step 4: Execute saga operation (pseudo-code)
	executeSagaOperation(ctx)

	// Step 5: Retrieve auth context from context
	if retrievedAuthCtx, ok := security.AuthFromContext(ctx); ok {
		fmt.Printf("Retrieved auth context from context: User ID = %s\n", retrievedAuthCtx.Credentials.UserID)
	}

	fmt.Println("Saga operation completed successfully with authentication!")
}

// executeSagaOperation simulates executing a saga operation with authentication context
func executeSagaOperation(ctx context.Context) {
	// Retrieve auth context from the context
	authCtx, ok := security.AuthFromContext(ctx)
	if !ok {
		log.Println("Warning: No authentication context found")
		return
	}

	fmt.Printf("\nExecuting saga operation with authentication...\n")
	fmt.Printf("Authenticated user: %s\n", authCtx.Credentials.UserID)
	fmt.Printf("Available scopes: %v\n", authCtx.Credentials.Scopes)

	// Your saga execution logic here
	// For example:
	// - Execute saga steps
	// - Handle compensation
	// - Track progress
	// etc.

	fmt.Println("Saga operation executed successfully!")
}

// Additional Example: Scope-based Authorization

func scopeBasedAuthorizationExample() {
	fmt.Println("=== Scope-based Authorization Example ===")

	// Create credentials with specific scopes
	credentials := &security.AuthCredentials{
		Type:   security.AuthTypeAPIKey,
		APIKey: "admin-key",
		UserID: "admin-user",
		Scopes: []string{"saga:execute", "saga:compensate", "saga:admin"},
	}

	// Check if user has specific scopes
	if credentials.HasScope("saga:execute") {
		fmt.Println("User can execute saga operations")
	}

	if credentials.HasScope("saga:admin") {
		fmt.Println("User has admin privileges")
	}

	if !credentials.HasScope("saga:delete") {
		fmt.Println("User cannot delete saga operations")
	}
}

// Additional Example: Token Expiration Handling

func tokenExpirationExample() {
	fmt.Println("\n=== Token Expiration Example ===")

	// Create credentials with expiration
	expiresAt := time.Now().Add(1 * time.Hour)
	credentials := &security.AuthCredentials{
		Type:      security.AuthTypeJWT,
		Token:     "sample-token",
		UserID:    "user-123",
		ExpiresAt: &expiresAt,
	}

	// Check if credentials are expired
	if credentials.IsExpired() {
		fmt.Println("Credentials have expired, need to refresh")
	} else {
		fmt.Printf("Credentials are valid until: %v\n", credentials.ExpiresAt)
	}

	// Simulate expired credentials
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredCredentials := &security.AuthCredentials{
		Type:      security.AuthTypeJWT,
		Token:     "expired-token",
		UserID:    "user-123",
		ExpiresAt: &expiredTime,
	}

	if expiredCredentials.IsExpired() {
		fmt.Println("Expired credentials detected")
	}
}
