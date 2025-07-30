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

package adapter

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/innovationmech/swit/internal/switauth/deps"
	auth "github.com/innovationmech/swit/internal/switauth/handler/http/auth/v1"
	"github.com/innovationmech/swit/internal/switauth/handler/http/health"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/types"
)

// SwitauthServiceRegistrar implements the ServiceRegistrar interface for switauth service
type SwitauthServiceRegistrar struct {
	deps *deps.Dependencies
}

// NewSwitauthServiceRegistrar creates a new switauth service registrar
func NewSwitauthServiceRegistrar(dependencies *deps.Dependencies) *SwitauthServiceRegistrar {
	return &SwitauthServiceRegistrar{
		deps: dependencies,
	}
}

// RegisterServices registers all switauth services with the provided registry
func (s *SwitauthServiceRegistrar) RegisterServices(registry server.ServiceRegistry) error {
	// Register auth HTTP handler
	authHandler := auth.NewAuthHandler(s.deps.AuthSrv)
	authHTTPHandler := &AuthHTTPHandler{handler: authHandler}
	if err := registry.RegisterHTTPHandler(authHTTPHandler); err != nil {
		return err
	}

	// Register health HTTP handler
	healthHandler := health.NewHandler()
	healthHTTPHandler := &HealthHTTPHandler{handler: healthHandler}
	if err := registry.RegisterHTTPHandler(healthHTTPHandler); err != nil {
		return err
	}

	// Register health check
	healthCheck := &SwitauthHealthCheck{
		deps:      s.deps,
		startTime: time.Now(),
	}
	if err := registry.RegisterHealthCheck(healthCheck); err != nil {
		return err
	}

	return nil
}

// AuthHTTPHandler wraps the auth handler to implement the HTTPHandler interface
type AuthHTTPHandler struct {
	handler *auth.AuthHandler
}

// RegisterRoutes registers HTTP routes with the given router
func (a *AuthHTTPHandler) RegisterRoutes(router gin.IRouter) error {
	// Create auth group
	authGroup := router.Group("/auth")

	// Register routes
	authGroup.POST("/login", a.handler.Login)
	authGroup.POST("/logout", a.handler.Logout)
	authGroup.POST("/refresh", a.handler.RefreshToken)
	authGroup.GET("/validate", a.handler.ValidateToken)

	return nil
}

// GetServiceName returns the service name
func (a *AuthHTTPHandler) GetServiceName() string {
	return "switauth-auth"
}

// GetVersion returns the service version
func (a *AuthHTTPHandler) GetVersion() string {
	return "v1.0.0"
}

// HealthHTTPHandler wraps the health handler to implement the HTTPHandler interface
type HealthHTTPHandler struct {
	handler *health.Handler
}

// RegisterRoutes registers HTTP routes with the given router
func (h *HealthHTTPHandler) RegisterRoutes(router gin.IRouter) error {
	// Register health check routes
	router.GET("/health", h.handler.HealthCheck)
	router.GET("/health/ready", h.handler.ReadinessCheck)
	router.GET("/health/live", h.handler.LivenessCheck)

	return nil
}

// GetServiceName returns the service name
func (h *HealthHTTPHandler) GetServiceName() string {
	return "switauth-health"
}

// GetVersion returns the service version
func (h *HealthHTTPHandler) GetVersion() string {
	return "v1.0.0"
}

// SwitauthHealthCheck implements the HealthCheck interface for switauth
type SwitauthHealthCheck struct {
	deps      *deps.Dependencies
	startTime time.Time
}

// Check performs the health check
func (s *SwitauthHealthCheck) Check(ctx context.Context) *types.HealthStatus {
	// Check database connectivity
	if s.deps.DB != nil {
		if sqlDB, err := s.deps.DB.DB(); err == nil {
			if err := sqlDB.Ping(); err != nil {
				return &types.HealthStatus{
					Status:    types.HealthStatusUnhealthy,
					Timestamp: time.Now(),
					Version:   "v1.0.0",
					Uptime:    time.Since(s.startTime),
				}
			}
		}
	}

	// Check service discovery connectivity
	if s.deps.SD != nil {
		// Note: ServiceDiscovery doesn't have IsHealthy method
		// We'll assume it's healthy if it exists
	}

	return &types.HealthStatus{
		Status:    types.HealthStatusHealthy,
		Timestamp: time.Now(),
		Version:   "v1.0.0",
		Uptime:    time.Since(s.startTime),
	}
}

// GetName returns the health check name
func (s *SwitauthHealthCheck) GetName() string {
	return "switauth-service"
}
