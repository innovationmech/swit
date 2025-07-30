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

package server

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/middleware"
	"google.golang.org/grpc"
)

// MiddlewareRegistrar defines the interface for middleware registration
type MiddlewareRegistrar interface {
	// RegisterHTTPMiddleware registers HTTP middleware to the gin engine
	RegisterHTTPMiddleware(router *gin.Engine) error

	// RegisterGRPCMiddleware registers gRPC interceptors to the gRPC server
	RegisterGRPCMiddleware(server *grpc.Server) error

	// GetName returns the middleware name
	GetName() string

	// GetPriority returns the middleware priority (lower number = higher priority)
	GetPriority() int

	// IsEnabled returns whether the middleware is enabled
	IsEnabled() bool
}

// MiddlewareManager manages middleware registration for both HTTP and gRPC transports
type MiddlewareManager struct {
	mu          sync.RWMutex
	registrars  []MiddlewareRegistrar
	config      *MiddlewareConfig
	httpEnabled bool
	grpcEnabled bool
}

// NewMiddlewareManager creates a new middleware manager
func NewMiddlewareManager(config *MiddlewareConfig, httpEnabled, grpcEnabled bool) *MiddlewareManager {
	return &MiddlewareManager{
		registrars:  make([]MiddlewareRegistrar, 0),
		config:      config,
		httpEnabled: httpEnabled,
		grpcEnabled: grpcEnabled,
	}
}

// Register adds a middleware registrar to the manager
func (mm *MiddlewareManager) Register(registrar MiddlewareRegistrar) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.registrars = append(mm.registrars, registrar)

	// Sort by priority (lower number = higher priority)
	sort.Slice(mm.registrars, func(i, j int) bool {
		return mm.registrars[i].GetPriority() < mm.registrars[j].GetPriority()
	})
}

// RegisterHTTPMiddlewares registers all enabled HTTP middlewares to the gin engine
func (mm *MiddlewareManager) RegisterHTTPMiddlewares(router *gin.Engine) error {
	if !mm.httpEnabled {
		return nil
	}

	mm.mu.RLock()
	defer mm.mu.RUnlock()

	for _, registrar := range mm.registrars {
		if registrar.IsEnabled() {
			if err := registrar.RegisterHTTPMiddleware(router); err != nil {
				return fmt.Errorf("failed to register HTTP middleware %s: %w", registrar.GetName(), err)
			}
		}
	}

	return nil
}

// RegisterGRPCMiddlewares registers all enabled gRPC middlewares to the gRPC server
func (mm *MiddlewareManager) RegisterGRPCMiddlewares(server *grpc.Server) error {
	if !mm.grpcEnabled {
		return nil
	}

	mm.mu.RLock()
	defer mm.mu.RUnlock()

	for _, registrar := range mm.registrars {
		if registrar.IsEnabled() {
			if err := registrar.RegisterGRPCMiddleware(server); err != nil {
				return fmt.Errorf("failed to register gRPC middleware %s: %w", registrar.GetName(), err)
			}
		}
	}

	return nil
}

// GetRegisteredMiddlewares returns a list of all registered middleware names
func (mm *MiddlewareManager) GetRegisteredMiddlewares() []string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	names := make([]string, 0, len(mm.registrars))
	for _, registrar := range mm.registrars {
		names = append(names, registrar.GetName())
	}

	return names
}

// GetEnabledMiddlewares returns a list of enabled middleware names
func (mm *MiddlewareManager) GetEnabledMiddlewares() []string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	names := make([]string, 0)
	for _, registrar := range mm.registrars {
		if registrar.IsEnabled() {
			names = append(names, registrar.GetName())
		}
	}

	return names
}

// Built-in middleware registrars

// CORSMiddlewareRegistrar implements MiddlewareRegistrar for CORS middleware
type CORSMiddlewareRegistrar struct {
	enabled bool
}

// NewCORSMiddlewareRegistrar creates a new CORS middleware registrar
func NewCORSMiddlewareRegistrar(enabled bool) *CORSMiddlewareRegistrar {
	return &CORSMiddlewareRegistrar{enabled: enabled}
}

func (c *CORSMiddlewareRegistrar) RegisterHTTPMiddleware(router *gin.Engine) error {
	router.Use(middleware.CORSMiddleware())
	return nil
}

func (c *CORSMiddlewareRegistrar) RegisterGRPCMiddleware(server *grpc.Server) error {
	// CORS is HTTP-specific, no gRPC implementation needed
	return nil
}

func (c *CORSMiddlewareRegistrar) GetName() string {
	return "cors"
}

func (c *CORSMiddlewareRegistrar) GetPriority() int {
	return 10 // Low priority
}

func (c *CORSMiddlewareRegistrar) IsEnabled() bool {
	return c.enabled
}

// AuthMiddlewareRegistrar implements MiddlewareRegistrar for authentication middleware
type AuthMiddlewareRegistrar struct {
	enabled bool
}

// NewAuthMiddlewareRegistrar creates a new auth middleware registrar
func NewAuthMiddlewareRegistrar(enabled bool) *AuthMiddlewareRegistrar {
	return &AuthMiddlewareRegistrar{enabled: enabled}
}

func (a *AuthMiddlewareRegistrar) RegisterHTTPMiddleware(router *gin.Engine) error {
	router.Use(middleware.AuthMiddleware())
	return nil
}

func (a *AuthMiddlewareRegistrar) RegisterGRPCMiddleware(server *grpc.Server) error {
	// TODO: Implement gRPC auth interceptor
	return nil
}

func (a *AuthMiddlewareRegistrar) GetName() string {
	return "auth"
}

func (a *AuthMiddlewareRegistrar) GetPriority() int {
	return 5 // High priority
}

func (a *AuthMiddlewareRegistrar) IsEnabled() bool {
	return a.enabled
}

// RateLimitMiddlewareRegistrar implements MiddlewareRegistrar for rate limiting middleware
type RateLimitMiddlewareRegistrar struct {
	enabled bool
	rps     int
}

// NewRateLimitMiddlewareRegistrar creates a new rate limit middleware registrar
func NewRateLimitMiddlewareRegistrar(enabled bool, rps int) *RateLimitMiddlewareRegistrar {
	return &RateLimitMiddlewareRegistrar{enabled: enabled, rps: rps}
}

func (r *RateLimitMiddlewareRegistrar) RegisterHTTPMiddleware(router *gin.Engine) error {
	rateLimiter := middleware.NewRateLimiter(r.rps, time.Minute)
	router.Use(rateLimiter.Middleware())
	return nil
}

func (r *RateLimitMiddlewareRegistrar) RegisterGRPCMiddleware(server *grpc.Server) error {
	// TODO: Implement gRPC rate limit interceptor
	return nil
}

func (r *RateLimitMiddlewareRegistrar) GetName() string {
	return "rate_limit"
}

func (r *RateLimitMiddlewareRegistrar) GetPriority() int {
	return 3 // Very high priority
}

func (r *RateLimitMiddlewareRegistrar) IsEnabled() bool {
	return r.enabled
}

// RequestLoggerMiddlewareRegistrar implements MiddlewareRegistrar for request logging middleware
type RequestLoggerMiddlewareRegistrar struct {
	enabled bool
}

// NewRequestLoggerMiddlewareRegistrar creates a new request logger middleware registrar
func NewRequestLoggerMiddlewareRegistrar(enabled bool) *RequestLoggerMiddlewareRegistrar {
	return &RequestLoggerMiddlewareRegistrar{enabled: enabled}
}

func (r *RequestLoggerMiddlewareRegistrar) RegisterHTTPMiddleware(router *gin.Engine) error {
	router.Use(middleware.RequestLogger())
	return nil
}

func (r *RequestLoggerMiddlewareRegistrar) RegisterGRPCMiddleware(server *grpc.Server) error {
	// TODO: Implement gRPC logging interceptor
	return nil
}

func (r *RequestLoggerMiddlewareRegistrar) GetName() string {
	return "request_logger"
}

func (r *RequestLoggerMiddlewareRegistrar) GetPriority() int {
	return 8 // Medium priority
}

func (r *RequestLoggerMiddlewareRegistrar) IsEnabled() bool {
	return r.enabled
}

// TimeoutMiddlewareRegistrar implements MiddlewareRegistrar for timeout middleware
type TimeoutMiddlewareRegistrar struct {
	enabled bool
	timeout time.Duration
}

// NewTimeoutMiddlewareRegistrar creates a new timeout middleware registrar
func NewTimeoutMiddlewareRegistrar(enabled bool, timeout time.Duration) *TimeoutMiddlewareRegistrar {
	return &TimeoutMiddlewareRegistrar{enabled: enabled, timeout: timeout}
}

func (t *TimeoutMiddlewareRegistrar) RegisterHTTPMiddleware(router *gin.Engine) error {
	router.Use(middleware.TimeoutMiddleware(t.timeout))
	return nil
}

func (t *TimeoutMiddlewareRegistrar) RegisterGRPCMiddleware(server *grpc.Server) error {
	// TODO: Implement gRPC timeout interceptor
	return nil
}

func (t *TimeoutMiddlewareRegistrar) GetName() string {
	return "timeout"
}

func (t *TimeoutMiddlewareRegistrar) GetPriority() int {
	return 2 // Very high priority
}

func (t *TimeoutMiddlewareRegistrar) IsEnabled() bool {
	return t.enabled
}

// CreateDefaultMiddlewareManager creates a middleware manager with default middlewares based on config
func CreateDefaultMiddlewareManager(config *MiddlewareConfig, httpEnabled, grpcEnabled bool) *MiddlewareManager {
	mm := NewMiddlewareManager(config, httpEnabled, grpcEnabled)

	// Register built-in middlewares based on configuration
	if config.EnableTimeout {
		mm.Register(NewTimeoutMiddlewareRegistrar(true, config.TimeoutDuration))
	}

	if config.EnableRateLimit {
		mm.Register(NewRateLimitMiddlewareRegistrar(true, config.RateLimitRPS))
	}

	if config.EnableAuth {
		mm.Register(NewAuthMiddlewareRegistrar(true))
	}

	if config.EnableRequestLogger {
		mm.Register(NewRequestLoggerMiddlewareRegistrar(true))
	}

	if config.EnableCORS {
		mm.Register(NewCORSMiddlewareRegistrar(true))
	}

	return mm
}
