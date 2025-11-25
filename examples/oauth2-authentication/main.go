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
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/middleware"
	"github.com/innovationmech/swit/pkg/security/jwt"
	switoauth2 "github.com/innovationmech/swit/pkg/security/oauth2"
	"github.com/innovationmech/swit/pkg/server"
)

// OAuth2ExampleService implements the ServiceRegistrar interface.
type OAuth2ExampleService struct {
	name         string
	oauth2Client *switoauth2.Client
	jwtValidator *jwt.Validator
	flowManager  *switoauth2.FlowManager
}

// NewOAuth2ExampleService creates a new OAuth2 example service.
func NewOAuth2ExampleService(name string, oauth2Client *switoauth2.Client, jwtValidator *jwt.Validator) *OAuth2ExampleService {
	return &OAuth2ExampleService{
		name:         name,
		oauth2Client: oauth2Client,
		jwtValidator: jwtValidator,
		flowManager:  switoauth2.NewFlowManager(),
	}
}

// RegisterServices registers the HTTP handlers with the server.
func (s *OAuth2ExampleService) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register HTTP handler
	httpHandler := &OAuth2HTTPHandler{
		serviceName:  s.name,
		oauth2Client: s.oauth2Client,
		jwtValidator: s.jwtValidator,
		flowManager:  s.flowManager,
	}
	if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}

	// Register health check
	healthCheck := &OAuth2HealthCheck{serviceName: s.name}
	if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health check: %w", err)
	}

	return nil
}

// OAuth2HTTPHandler implements the HTTPHandler interface.
type OAuth2HTTPHandler struct {
	serviceName  string
	oauth2Client *switoauth2.Client
	jwtValidator *jwt.Validator
	flowManager  *switoauth2.FlowManager
}

// RegisterRoutes registers HTTP routes with the router.
func (h *OAuth2HTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(gin.IRouter)
	if !ok {
		return fmt.Errorf("expected gin.IRouter, got %T", router)
	}

	// Public endpoints (no authentication required)
	public := ginRouter.Group("/api/v1/public")
	{
		public.GET("/info", h.handlePublicInfo)
		public.GET("/login", h.handleLogin)
		public.GET("/callback", h.handleCallback)
		public.POST("/refresh", h.handleRefresh)
		public.POST("/logout", h.handleLogout)
	}

	// Protected endpoints (OAuth2 authentication required)
	protected := ginRouter.Group("/api/v1/protected")
	protected.Use(middleware.OAuth2Middleware(h.oauth2Client, h.jwtValidator))
	{
		protected.GET("/profile", h.handleProfile)
		protected.GET("/data", h.handleData)
	}

	// Optional auth endpoints (authentication is optional)
	optional := ginRouter.Group("/api/v1/optional")
	optional.Use(middleware.OptionalAuth(h.oauth2Client, h.jwtValidator))
	{
		optional.GET("/content", h.handleOptionalContent)
	}

	// Role-based access control examples
	admin := ginRouter.Group("/api/v1/admin")
	admin.Use(middleware.OAuth2Middleware(h.oauth2Client, h.jwtValidator))
	admin.Use(middleware.RequireRoles("admin"))
	{
		admin.GET("/dashboard", h.handleAdminDashboard)
	}

	return nil
}

// GetServiceName returns the service name.
func (h *OAuth2HTTPHandler) GetServiceName() string {
	return h.serviceName
}

// handlePublicInfo handles the public info endpoint.
func (h *OAuth2HTTPHandler) handlePublicInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":   "This is a public endpoint - no authentication required",
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
		"endpoints": gin.H{
			"login":   "/api/v1/public/login",
			"profile": "/api/v1/protected/profile",
		},
	})
}

// handleLogin initiates the OAuth2 authorization code flow.
func (h *OAuth2HTTPHandler) handleLogin(c *gin.Context) {
	// Start OAuth2 flow with state and PKCE
	flowState, err := h.flowManager.StartFlow(
		"http://localhost:8080/api/v1/public/callback",
		[]string{"openid", "profile", "email"},
		true, // Use PKCE
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to start OAuth2 flow",
			"details": err.Error(),
		})
		return
	}

	// Generate authorization URL with flow state
	authURL := h.oauth2Client.AuthCodeURLWithFlow(flowState)

	c.JSON(http.StatusOK, gin.H{
		"message":           "Redirect to this URL to authenticate",
		"authorization_url": authURL,
		"state":             flowState.State,
		"hint":              "Open the authorization_url in your browser",
	})
}

// handleCallback handles the OAuth2 callback.
func (h *OAuth2HTTPHandler) handleCallback(c *gin.Context) {
	// Extract code and state from query parameters
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	if errorParam != "" {
		errorDesc := c.Query("error_description")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             errorParam,
			"error_description": errorDesc,
		})
		return
	}

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing authorization code",
		})
		return
	}

	// Validate state parameter
	flowState, err := h.flowManager.ValidateState(state)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid state parameter",
			"details": err.Error(),
		})
		return
	}

	// Exchange authorization code for tokens
	ctx := context.Background()
	token, err := h.oauth2Client.ExchangeWithFlow(ctx, code, flowState)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to exchange authorization code",
			"details": err.Error(),
		})
		return
	}

	// Extract and verify ID token
	var idTokenClaims map[string]interface{}
	if rawIDToken, ok := token.Extra("id_token").(string); ok {
		idToken, err := h.oauth2Client.VerifyIDToken(ctx, rawIDToken)
		if err != nil {
			logger.GetLogger().Warn("Failed to verify ID token", zap.Error(err))
		} else {
			if err := idToken.Claims(&idTokenClaims); err == nil {
				// Verify nonce if present
				if nonce, ok := idTokenClaims["nonce"].(string); ok && flowState.Nonce != "" {
					if err := switoauth2.ValidateNonce(flowState.Nonce, nonce); err != nil {
						c.JSON(http.StatusBadRequest, gin.H{
							"error":   "Nonce validation failed",
							"details": err.Error(),
						})
						return
					}
				}
			}
		}
	}

	// In a real application, you would:
	// 1. Store the token securely (e.g., in a session or secure cookie)
	// 2. Create a user session
	// 3. Redirect to the application

	c.JSON(http.StatusOK, gin.H{
		"message":         "Authentication successful",
		"access_token":    token.AccessToken,
		"token_type":      token.TokenType,
		"expires_in":      int(time.Until(token.Expiry).Seconds()),
		"refresh_token":   token.RefreshToken,
		"id_token_claims": idTokenClaims,
		"hint":            "Use the access_token in the Authorization header for protected endpoints",
	})
}

// handleRefresh handles token refresh requests.
func (h *OAuth2HTTPHandler) handleRefresh(c *gin.Context) {
	var request struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Create a token with the refresh token
	token := &oauth2.Token{
		RefreshToken: request.RefreshToken,
	}

	// Refresh the token
	ctx := context.Background()
	newToken, err := h.oauth2Client.RefreshToken(ctx, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to refresh token",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Token refreshed successfully",
		"access_token":  newToken.AccessToken,
		"token_type":    newToken.TokenType,
		"expires_in":    int(time.Until(newToken.Expiry).Seconds()),
		"refresh_token": newToken.RefreshToken,
	})
}

// handleLogout handles logout requests (token revocation).
func (h *OAuth2HTTPHandler) handleLogout(c *gin.Context) {
	var request struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Revoke the token
	ctx := context.Background()
	if err := h.oauth2Client.RevokeToken(ctx, request.Token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to revoke token",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// handleProfile handles the protected profile endpoint.
func (h *OAuth2HTTPHandler) handleProfile(c *gin.Context) {
	// Get user info from context (set by OAuth2 middleware)
	userInfo, ok := middleware.GetUserInfo(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User info not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "This is a protected endpoint",
		"user_info": userInfo,
		"timestamp": time.Now().UTC(),
	})
}

// handleData handles the protected data endpoint.
func (h *OAuth2HTTPHandler) handleData(c *gin.Context) {
	userInfo, _ := middleware.GetUserInfo(c)

	c.JSON(http.StatusOK, gin.H{
		"message": "This is protected data",
		"data": gin.H{
			"user_subject": userInfo.Subject,
			"access_level": "user",
		},
		"timestamp": time.Now().UTC(),
	})
}

// handleOptionalContent handles the optional auth endpoint.
func (h *OAuth2HTTPHandler) handleOptionalContent(c *gin.Context) {
	userInfo, authenticated := middleware.GetUserInfo(c)

	if authenticated {
		c.JSON(http.StatusOK, gin.H{
			"message":       "Welcome back, authenticated user!",
			"user_subject":  userInfo.Subject,
			"authenticated": true,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message":       "Welcome, anonymous user!",
			"authenticated": false,
			"hint":          "Login to access personalized content",
		})
	}
}

// handleAdminDashboard handles the admin-only endpoint.
func (h *OAuth2HTTPHandler) handleAdminDashboard(c *gin.Context) {
	userInfo, _ := middleware.GetUserInfo(c)

	c.JSON(http.StatusOK, gin.H{
		"message":   "Welcome to the admin dashboard",
		"user_info": userInfo,
		"admin_data": gin.H{
			"total_users":  100,
			"active_users": 75,
		},
	})
}

// OAuth2HealthCheck implements the HealthCheck interface.
type OAuth2HealthCheck struct {
	serviceName string
}

// Check performs a health check.
func (h *OAuth2HealthCheck) Check(ctx context.Context) error {
	// In a real service, check OAuth2 provider availability, database connections, etc.
	return nil
}

// GetServiceName returns the service name.
func (h *OAuth2HealthCheck) GetServiceName() string {
	return h.serviceName
}

// OAuth2DependencyContainer implements the DependencyContainer interface.
type OAuth2DependencyContainer struct {
	services map[string]interface{}
	closed   bool
}

// NewOAuth2DependencyContainer creates a new dependency container.
func NewOAuth2DependencyContainer(oauth2Client *switoauth2.Client, jwtValidator *jwt.Validator) *OAuth2DependencyContainer {
	container := &OAuth2DependencyContainer{
		services: make(map[string]interface{}),
		closed:   false,
	}

	// Add OAuth2 client and JWT validator as services
	container.services["oauth2_client"] = oauth2Client
	container.services["jwt_validator"] = jwtValidator

	return container
}

// GetService retrieves a service by name.
func (d *OAuth2DependencyContainer) GetService(name string) (interface{}, error) {
	if d.closed {
		return nil, fmt.Errorf("dependency container is closed")
	}

	service, exists := d.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return service, nil
}

// Close closes the dependency container and cleans up resources.
func (d *OAuth2DependencyContainer) Close() error {
	if d.closed {
		return nil
	}

	// Close OAuth2 client
	if oauth2Client, ok := d.services["oauth2_client"].(*switoauth2.Client); ok {
		if err := oauth2Client.Close(); err != nil {
			logger.GetLogger().Warn("Failed to close OAuth2 client", zap.Error(err))
		}
	}

	d.closed = true
	return nil
}

// loadConfigFromFile loads configuration from YAML file.
func loadConfigFromFile(configPath string) (*server.ServerConfig, error) {
	config := &server.ServerConfig{}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// File doesn't exist, return default config
		return createDefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	applyEnvironmentOverrides(config)

	return config, nil
}

// createDefaultConfig creates default configuration.
func createDefaultConfig() *server.ServerConfig {
	return &server.ServerConfig{
		ServiceName: "oauth2-authentication-example",
		HTTP: server.HTTPConfig{
			Port:         getEnv("HTTP_PORT", "8080"),
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: server.GRPCConfig{
			Enabled: false, // Disable gRPC for this HTTP-only example
		},
		ShutdownTimeout: 30 * time.Second,
		Middleware: server.MiddlewareConfig{
			EnableCORS:    true,
			EnableLogging: true,
		},
	}
}

// applyEnvironmentOverrides applies environment variable overrides.
func applyEnvironmentOverrides(config *server.ServerConfig) {
	if httpPort := os.Getenv("HTTP_PORT"); httpPort != "" {
		config.HTTP.Port = httpPort
	}
}

func main() {
	// Initialize logger
	logger.InitLogger()

	// Load server configuration
	configPath := getEnv("CONFIG_PATH", "swit.yaml")
	if !filepath.IsAbs(configPath) {
		if execDir, err := os.Executable(); err == nil {
			configPath = filepath.Join(filepath.Dir(execDir), configPath)
		}
	}

	serverConfig, err := loadConfigFromFile(configPath)
	if err != nil {
		logger.GetLogger().Fatal("Failed to load server configuration", zap.Error(err))
	}

	if serverConfig == nil {
		logger.GetLogger().Warn("Using default server configuration")
		serverConfig = createDefaultConfig()
	}

	// Validate server configuration
	if err := serverConfig.Validate(); err != nil {
		logger.GetLogger().Fatal("Invalid server configuration", zap.Error(err))
	}

	// Load OAuth2 configuration from file or environment
	oauth2Config := &switoauth2.Config{
		Enabled:      true,
		Provider:     getEnv("OAUTH2_PROVIDER", "keycloak"),
		ClientID:     getEnv("OAUTH2_CLIENT_ID", "swit-example"),
		ClientSecret: getEnv("OAUTH2_CLIENT_SECRET", "swit-example-secret"),
		RedirectURL:  getEnv("OAUTH2_REDIRECT_URL", "http://localhost:8080/api/v1/public/callback"),
		Scopes:       []string{"openid", "profile", "email"},
		IssuerURL:    getEnv("OAUTH2_ISSUER_URL", "http://localhost:8081/realms/swit"),
		UseDiscovery: getBoolEnv("OAUTH2_USE_DISCOVERY", true),
		HTTPTimeout:  30 * time.Second,
	}

	// Load from environment variables
	oauth2Config.LoadFromEnv("OAUTH2_")

	// Create OAuth2 client with OIDC discovery
	ctx := context.Background()
	oauth2Client, err := switoauth2.NewClient(ctx, oauth2Config)
	if err != nil {
		logger.GetLogger().Fatal("Failed to create OAuth2 client", zap.Error(err))
	}

	// Create JWT validator for local token validation
	jwtConfig := &jwt.Config{
		Issuer:         oauth2Client.GetIssuerURL(),
		Audience:       oauth2Config.ClientID,
		LeewayDuration: oauth2Config.JWTConfig.ClockSkew,
		JWKSConfig: &jwt.JWKSCacheConfig{
			URL:            oauth2Client.GetJWKSURL(),
			RefreshTTL:     15 * time.Minute,
			AutoRefresh:    true,
			MinRefreshWait: 1 * time.Minute,
		},
	}

	jwtValidator, err := jwt.NewValidator(jwtConfig)
	if err != nil {
		logger.GetLogger().Fatal("Failed to create JWT validator", zap.Error(err))
	}

	// Create service and dependencies
	service := NewOAuth2ExampleService("oauth2-authentication-example", oauth2Client, jwtValidator)
	deps := NewOAuth2DependencyContainer(oauth2Client, jwtValidator)

	// Create base server
	baseServer, err := server.NewBusinessServerCore(serverConfig, service, deps)
	if err != nil {
		logger.GetLogger().Fatal("Failed to create server", zap.Error(err))
	}

	// Start server
	startCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := baseServer.Start(startCtx); err != nil {
		logger.GetLogger().Fatal("Failed to start server", zap.Error(err))
	}

	logger.GetLogger().Info("OAuth2 authentication example service started",
		zap.String("http_address", baseServer.GetHTTPAddress()),
		zap.String("service_name", "oauth2-authentication-example"),
		zap.String("oauth2_provider", oauth2Config.Provider),
		zap.String("issuer_url", oauth2Config.IssuerURL))
	logger.GetLogger().Info("Example endpoints",
		zap.String("public_info", fmt.Sprintf("http://%s/api/v1/public/info", baseServer.GetHTTPAddress())),
		zap.String("login", fmt.Sprintf("http://%s/api/v1/public/login", baseServer.GetHTTPAddress())))

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.GetLogger().Info("Shutdown signal received, stopping server")

	// Graceful shutdown
	if err := baseServer.Shutdown(); err != nil {
		logger.GetLogger().Error("Error during shutdown", zap.Error(err))
	} else {
		logger.GetLogger().Info("Server stopped gracefully")
	}
}

// getEnv gets an environment variable with a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getBoolEnv gets a boolean environment variable with a default value.
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}
