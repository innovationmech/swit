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

package monitoring

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// MiddlewareConfig contains configuration for monitoring middleware.
type MiddlewareConfig struct {
	// EnableLogging enables request logging middleware.
	EnableLogging bool

	// EnableCORS enables CORS middleware.
	EnableCORS bool

	// EnableErrorHandle enables error handling middleware.
	EnableErrorHandle bool

	// EnableRecovery enables panic recovery middleware.
	EnableRecovery bool

	// CORSConfig contains CORS configuration.
	CORSConfig *CORSConfig
}

// CORSConfig contains CORS middleware configuration.
type CORSConfig struct {
	// AllowOrigins is a list of origins that may access the resource.
	// Default is ["*"].
	AllowOrigins []string

	// AllowMethods is a list of methods the client is allowed to use.
	// Default is ["GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"].
	AllowMethods []string

	// AllowHeaders is list of non simple headers the client is allowed to use.
	// Default is ["Origin", "Content-Type", "Accept", "Authorization"].
	AllowHeaders []string

	// ExposeHeaders indicates which headers are safe to expose.
	ExposeHeaders []string

	// AllowCredentials indicates whether the request can include credentials.
	AllowCredentials bool

	// MaxAge indicates how long the results of a preflight request can be cached.
	MaxAge time.Duration
}

// DefaultMiddlewareConfig returns a default middleware configuration.
func DefaultMiddlewareConfig() *MiddlewareConfig {
	return &MiddlewareConfig{
		EnableLogging:     true,
		EnableCORS:        true,
		EnableErrorHandle: true,
		EnableRecovery:    true,
		CORSConfig:        DefaultCORSConfig(),
	}
}

// DefaultCORSConfig returns a default CORS configuration.
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}

// Middleware manages middleware for the monitoring server.
type Middleware struct {
	config *MiddlewareConfig
}

// NewMonitoringMiddleware creates a new monitoring middleware manager.
func NewMonitoringMiddleware(config *MiddlewareConfig) *Middleware {
	if config == nil {
		config = DefaultMiddlewareConfig()
	}

	// Apply defaults for CORS config
	if config.EnableCORS && config.CORSConfig == nil {
		config.CORSConfig = DefaultCORSConfig()
	}

	return &Middleware{
		config: config,
	}
}

// sanitizeForLog sanitizes user input for logging to prevent log injection attacks.
func sanitizeForLog(input string) string {
	// Remove newlines and tabs to prevent log injection
	sanitized := strings.ReplaceAll(input, "\n", "")
	sanitized = strings.ReplaceAll(sanitized, "\r", "")
	sanitized = strings.ReplaceAll(sanitized, "\t", "")

	// Limit length to prevent log flooding
	if len(sanitized) > 500 {
		sanitized = sanitized[:500] + "... [truncated]"
	}

	return sanitized
}

// ApplyToRouter applies all configured middleware to the Gin router.
func (m *Middleware) ApplyToRouter(router *gin.Engine) {
	// Recovery middleware should be first to catch panics in other middleware
	if m.config.EnableRecovery {
		router.Use(m.recoveryMiddleware())
		if logger.Logger != nil {
			logger.Logger.Debug("Recovery middleware enabled")
		}
	}

	// Logging middleware
	if m.config.EnableLogging {
		router.Use(m.loggingMiddleware())
		if logger.Logger != nil {
			logger.Logger.Debug("Logging middleware enabled")
		}
	}

	// CORS middleware
	if m.config.EnableCORS {
		router.Use(m.corsMiddleware())
		if logger.Logger != nil {
			logger.Logger.Debug("CORS middleware enabled")
		}
	}

	// Error handling middleware
	if m.config.EnableErrorHandle {
		router.Use(m.errorHandlingMiddleware())
		if logger.Logger != nil {
			logger.Logger.Debug("Error handling middleware enabled")
		}
	}
}

// loggingMiddleware creates a request logging middleware.
func (m *Middleware) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Log request details
		if logger.Logger != nil {
			logger.Logger.Info("HTTP request",
				zap.String("method", c.Request.Method),
				zap.String("path", sanitizeForLog(path)),
				zap.String("query", sanitizeForLog(query)),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("latency", latency),
				zap.String("client_ip", sanitizeForLog(c.ClientIP())),
				zap.String("user_agent", sanitizeForLog(c.Request.UserAgent())),
				zap.Int("body_size", c.Writer.Size()),
			)
		}
	}
}

// corsMiddleware creates a CORS middleware.
func (m *Middleware) corsMiddleware() gin.HandlerFunc {
	config := cors.Config{
		AllowOrigins:     m.config.CORSConfig.AllowOrigins,
		AllowMethods:     m.config.CORSConfig.AllowMethods,
		AllowHeaders:     m.config.CORSConfig.AllowHeaders,
		ExposeHeaders:    m.config.CORSConfig.ExposeHeaders,
		AllowCredentials: m.config.CORSConfig.AllowCredentials,
		MaxAge:           m.config.CORSConfig.MaxAge,
	}

	return cors.New(config)
}

// errorHandlingMiddleware creates an error handling middleware.
func (m *Middleware) errorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check for errors
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last()

			// Log the error
			if logger.Logger != nil {
				logger.Logger.Error("Request error",
					zap.String("path", sanitizeForLog(c.Request.URL.Path)),
					zap.String("method", c.Request.Method),
					zap.Error(err.Err),
					zap.Uint32("error_type", uint32(err.Type)),
				)
			}

			// Send error response if not already sent
			if !c.Writer.Written() {
				statusCode := http.StatusInternalServerError

				// Try to determine appropriate status code
				if err.Type == gin.ErrorTypeBind {
					statusCode = http.StatusBadRequest
				} else if err.Type == gin.ErrorTypeRender {
					statusCode = http.StatusInternalServerError
				}

				c.JSON(statusCode, gin.H{
					"error":     err.Error(),
					"timestamp": time.Now(),
				})
			}
		}
	}
}

// recoveryMiddleware creates a panic recovery middleware.
func (m *Middleware) recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				if logger.Logger != nil {
					logger.Logger.Error("Panic recovered",
						zap.Any("error", err),
						zap.String("path", sanitizeForLog(c.Request.URL.Path)),
						zap.String("method", c.Request.Method),
						zap.String("client_ip", sanitizeForLog(c.ClientIP())),
					)
				}

				// Send error response
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":     fmt.Sprintf("Internal server error: %v", err),
					"timestamp": time.Now(),
				})
			}
		}()

		c.Next()
	}
}

// GetConfig returns the middleware configuration.
func (m *Middleware) GetConfig() *MiddlewareConfig {
	return m.config
}
