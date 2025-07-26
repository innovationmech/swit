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

package v1

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switauth/interfaces"
	"github.com/innovationmech/swit/internal/switauth/types"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/middleware"
	"github.com/innovationmech/swit/pkg/transport"
	"google.golang.org/grpc"
)

// AuthHandler handles HTTP requests for authentication operations
// including login, logout, token refresh, and token validation
// and implements HandlerRegister interface
type AuthHandler struct {
	authService interfaces.AuthService
	startTime   time.Time
}

// NewAuthHandler creates a new auth AuthHandler instance
// with the provided authentication service
func NewAuthHandler(authService interfaces.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		startTime:   time.Now(),
	}
}

// HandlerRegister interface implementation

// RegisterHTTP registers HTTP routes for authentication
func (c *AuthHandler) RegisterHTTP(router *gin.Engine) error {
	logger.Logger.Info("Registering auth HTTP routes")

	// Create auth group
	auth := router.Group("/auth")

	// Apply rate limiting middleware
	auth.Use(c.rateLimitMiddleware())

	// Register routes using the existing auth controller
	auth.POST("/login", c.Login)
	auth.POST("/logout", c.Logout)
	auth.POST("/refresh", c.RefreshToken)
	auth.GET("/validate", c.ValidateToken)

	return nil
}

// RegisterGRPC registers gRPC services for authentication
func (c *AuthHandler) RegisterGRPC(server *grpc.Server) error {
	logger.Logger.Info("Registering auth gRPC services")
	// TODO: Implement gRPC service registration when needed
	return nil
}

// GetMetadata returns service metadata
func (c *AuthHandler) GetMetadata() *transport.HandlerMetadata {
	return &transport.HandlerMetadata{
		Name:           "auth-service",
		Version:        "v1.0.0",
		Description:    "Authentication service providing login, logout, refresh, and token validation",
		HealthEndpoint: "/auth/health",
		Tags:           []string{"auth", "authentication"},
		Dependencies:   []string{"database", "user-service"},
	}
}

// GetHealthEndpoint returns the health check endpoint
func (c *AuthHandler) GetHealthEndpoint() string {
	return "/auth/health"
}

// IsHealthy performs a health check
func (c *AuthHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	// Check if auth service is healthy
	if c.authService == nil {
		return types.NewHealthStatus(
			types.HealthStatusUnhealthy,
			"v1",
			time.Since(c.startTime),
		), nil
	}

	return types.NewHealthStatus(
		types.HealthStatusHealthy,
		"v1",
		time.Since(c.startTime),
	), nil
}

// Initialize initializes the service
func (c *AuthHandler) Initialize(ctx context.Context) error {
	logger.Logger.Info("Initializing auth service handler")
	// Perform any initialization logic here
	return nil
}

// Shutdown gracefully shuts down the service
func (c *AuthHandler) Shutdown(ctx context.Context) error {
	logger.Logger.Info("Shutting down auth service handler")
	// Perform any cleanup logic here
	return nil
}

// rateLimitMiddleware applies rate limiting to auth endpoints
func (c *AuthHandler) rateLimitMiddleware() gin.HandlerFunc {
	// Use the auth-specific rate limiting middleware
	return middleware.NewAuthRateLimiter().Middleware()
}

// HTTP Handler Methods

// Login authenticates a user and returns access and refresh tokens
//
//	@Summary		User login
//	@Description	Authenticate a user with username and password, returns access and refresh tokens
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Param			login	body		types.LoginRequest	true	"User login credentials"
//	@Success		200		{object}	types.AuthResponse	"Login successful"
//	@Failure		400		{object}	types.AuthError		"Bad request"
//	@Failure		401		{object}	types.AuthError		"Invalid credentials"
//	@Router			/auth/login [post]
func (c *AuthHandler) Login(ctx *gin.Context) {
	var loginData types.LoginRequest

	if err := ctx.ShouldBindJSON(&loginData); err != nil {
		ctx.JSON(http.StatusBadRequest, types.NewAuthError(types.ErrorCodeInvalidRequest, "Invalid request format", err))
		return
	}

	response, err := c.authService.Login(ctx.Request.Context(), loginData.Username, loginData.Password)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, types.NewAuthError(types.ErrorCodeInvalidCredentials, "Authentication failed", err))
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Logout invalidates the user's access token and logs them out
//
//	@Summary		User logout
//	@Description	Invalidate the user's access token and log them out
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	types.ResponseStatus	"Logout successful"
//	@Failure		400	{object}	types.AuthError			"Authorization header is missing"
//	@Failure		500	{object}	types.AuthError			"Internal server error"
//	@Router			/auth/logout [post]
func (c *AuthHandler) Logout(ctx *gin.Context) {
	tokenString := ctx.GetHeader("Authorization")
	if tokenString == "" {
		ctx.JSON(http.StatusBadRequest, types.NewAuthError(types.ErrorCodeInvalidRequest, "Authorization header is missing", nil))
		return
	}

	if err := c.authService.Logout(ctx.Request.Context(), tokenString); err != nil {
		ctx.JSON(http.StatusInternalServerError, types.NewAuthError(types.ErrorCodeInternalError, "Logout failed", err))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessStatus("Logged out successfully"))
}

// RefreshToken generates new access and refresh tokens using a valid refresh token
//
//	@Summary		Refresh access token
//	@Description	Generate new access and refresh tokens using a valid refresh token
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Param			refresh	body		types.RefreshTokenRequest	true	"Refresh token"
//	@Success		200		{object}	types.AuthResponse			"Token refresh successful"
//	@Failure		400		{object}	types.AuthError				"Bad request"
//	@Failure		401		{object}	types.AuthError				"Invalid or expired refresh token"
//	@Router			/auth/refresh [post]
func (c *AuthHandler) RefreshToken(ctx *gin.Context) {
	var data types.RefreshTokenRequest

	if err := ctx.ShouldBindJSON(&data); err != nil {
		ctx.JSON(http.StatusBadRequest, types.NewAuthError(types.ErrorCodeInvalidRequest, "Invalid request format", err))
		return
	}

	response, err := c.authService.RefreshToken(ctx.Request.Context(), data.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, types.NewAuthError(types.ErrorCodeTokenInvalid, "Token refresh failed", err))
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// ValidateToken validates an access token and returns token information
//
//	@Summary		Validate access token
//	@Description	Validate an access token and return token information including user ID
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	types.ValidateTokenResponse	"Token is valid"
//	@Failure		401	{object}	types.AuthError				"Invalid or expired token"
//	@Router			/auth/validate [get]
func (c *AuthHandler) ValidateToken(ctx *gin.Context) {
	// Get Authorization header
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		ctx.JSON(http.StatusUnauthorized, types.NewAuthError(types.ErrorCodeTokenInvalid, "Missing Authorization header", nil))
		return
	}

	// Split the Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		ctx.JSON(http.StatusUnauthorized, types.NewAuthError(types.ErrorCodeTokenInvalid, "Invalid Authorization header format", nil))
		return
	}

	// Extract the actual token
	tokenString := parts[1]

	// Validate token
	token, err := c.authService.ValidateToken(ctx.Request.Context(), tokenString)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, types.NewAuthError(types.ErrorCodeTokenInvalid, "Token validation failed", err))
		return
	}

	ctx.JSON(http.StatusOK, &types.ValidateTokenResponse{
		Message: "Token is valid",
		UserID:  token.UserID.String(),
	})
}
