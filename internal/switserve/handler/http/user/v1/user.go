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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switserve/interfaces"
	"github.com/innovationmech/swit/internal/switserve/model"
	"github.com/innovationmech/swit/internal/switserve/transport"
	"github.com/innovationmech/swit/internal/switserve/types"
	"github.com/innovationmech/swit/pkg/utils"
	"google.golang.org/grpc"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userSrv   interfaces.UserService
	startTime time.Time
}

// NewUserHandler creates a new user controller with dependency injection.
func NewUserHandler(userSrv interfaces.UserService) *UserHandler {
	return &UserHandler{
		userSrv:   userSrv,
		startTime: time.Now(),
	}
}

// CreateUser creates a new user.
//
//	@Summary		Create a new user
//	@Description	Create a new user with username, email and password
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		model.CreateUserRequest	true	"User information"
//	@Success		201		{object}	map[string]interface{}	"User created successfully"
//	@Failure		400		{object}	map[string]interface{}	"Bad request"
//	@Failure		500		{object}	map[string]interface{}	"Internal server error"
//	@Router			/users/create [post]
func (uc *UserHandler) CreateUser(c *gin.Context) {
	var req model.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password, // 原始密码，服务层会进行哈希处理
	}

	err := uc.userSrv.CreateUser(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "User created successfully",
	})
}

// GetUserByUsername gets a user by username.
//
//	@Summary		Get user by username
//	@Description	Get user details by username
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			username	path		string					true	"Username"
//	@Success		200			{object}	map[string]interface{}	"User details"
//	@Failure		404			{object}	map[string]interface{}	"User not found"
//	@Failure		500			{object}	map[string]interface{}	"Internal server error"
//	@Security		BearerAuth
//	@Router			/users/username/{username} [get]
func (uc *UserHandler) GetUserByUsername(c *gin.Context) {
	username := c.Param("username")
	user, err := uc.userSrv.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetUserByEmail gets a user by email.
//
//	@Summary		Get user by email
//	@Description	Get user details by email address
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			email	path		string					true	"Email address"
//	@Success		200		{object}	map[string]interface{}	"User details"
//	@Failure		404		{object}	map[string]interface{}	"User not found"
//	@Failure		500		{object}	map[string]interface{}	"Internal server error"
//	@Security		BearerAuth
//	@Router			/users/email/{email} [get]
func (uc *UserHandler) GetUserByEmail(c *gin.Context) {
	email := c.Param("email")
	user, err := uc.userSrv.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser deletes a user.
//
//	@Summary		Delete a user
//	@Description	Delete a user by ID
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"User ID"
//	@Success		200	{object}	map[string]interface{}	"User deleted successfully"
//	@Failure		400	{object}	map[string]interface{}	"Invalid user ID"
//	@Failure		500	{object}	map[string]interface{}	"Internal server error"
//	@Security		BearerAuth
//	@Router			/users/{id} [delete]
func (uc *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	err := uc.userSrv.DeleteUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "User deleted successfully",
	})
}

// ValidateUserCredentials is an internal API for validating user credentials
//
//	@Summary		Validate user credentials (Internal API)
//	@Description	Internal API for validating user credentials, used by authentication service
//	@Tags			internal
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		map[string]string		true	"User credentials"	SchemaExample({"username": "john_doe", "password": "secret123"})
//	@Success		200			{object}	map[string]interface{}	"Validation successful"
//	@Failure		400			{object}	map[string]interface{}	"Bad request"
//	@Failure		401			{object}	map[string]interface{}	"Invalid credentials"
//	@Failure		500			{object}	map[string]interface{}	"Internal server error"
//	@Router			/internal/validate-user [post]
func (uc *UserHandler) ValidateUserCredentials(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid request format",
			},
		})
		return
	}

	// Get user by username
	user, err := uc.userSrv.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Invalid credentials",
			},
		})
		return
	}

	// Verify password
	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Invalid credentials",
			},
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"valid": true,
			"user": map[string]interface{}{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
				"role":     user.Role,
			},
		},
	})
}

// RegisterHTTP registers HTTP routes with the given router
func (h *UserHandler) RegisterHTTP(router *gin.Engine) error {
	v1 := router.Group("/api/v1")
	{
		// User management routes
		v1.POST("/users/create", h.CreateUser)
		v1.GET("/users/username/:username", h.GetUserByUsername)
		v1.GET("/users/email/:email", h.GetUserByEmail)
		v1.DELETE("/users/:id", h.DeleteUser)
	}

	// Internal routes
	internal := router.Group("/internal")
	{
		internal.POST("/validate-user", h.ValidateUserCredentials)
	}

	return nil
}

// RegisterGRPC registers gRPC services with the given server
func (h *UserHandler) RegisterGRPC(server *grpc.Server) error {
	// User service doesn't have gRPC implementation yet
	return nil
}

// GetMetadata returns service metadata information
func (h *UserHandler) GetMetadata() *transport.ServiceMetadata {
	return &transport.ServiceMetadata{
		Name:           "user-service",
		Version:        "v1",
		Description:    "User management service",
		HealthEndpoint: "/api/v1/users/health",
		Tags:           []string{"user", "management"},
	}
}

// GetHealthEndpoint returns the health check endpoint path
func (h *UserHandler) GetHealthEndpoint() string {
	return "/api/v1/users/health"
}

// IsHealthy performs a health check and returns the current status
func (h *UserHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	// Simple health check - verify user service is available
	if h.userSrv == nil {
		return types.NewHealthStatus(types.HealthStatusUnhealthy, "v1", time.Since(h.startTime)), nil
	}

	return types.NewHealthStatus(types.HealthStatusHealthy, "v1", time.Since(h.startTime)), nil
}

// Initialize performs any necessary initialization before service registration
func (h *UserHandler) Initialize(ctx context.Context) error {
	// No special initialization needed for user handler
	return nil
}

// Shutdown performs graceful shutdown of the service
func (h *UserHandler) Shutdown(ctx context.Context) error {
	// No special cleanup needed for user handler
	return nil
}
