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
	"context"
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/middleware"
	"go.uber.org/zap"
)

// MiddlewareManager manages middleware registration and configuration for both HTTP and gRPC transports
type MiddlewareManager struct {
	httpMiddlewares        []HTTPMiddlewareFunc
	grpcUnaryInterceptors  []grpc.UnaryServerInterceptor
	grpcStreamInterceptors []grpc.StreamServerInterceptor
	config                 *ServerConfig
}

// HTTPMiddlewareFunc represents an HTTP middleware function
type HTTPMiddlewareFunc func(*gin.Engine) error

// NewMiddlewareManager creates a new middleware manager with the given configuration
func NewMiddlewareManager(config *ServerConfig) *MiddlewareManager {
	return &MiddlewareManager{
		httpMiddlewares:        make([]HTTPMiddlewareFunc, 0),
		grpcUnaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0),
		grpcStreamInterceptors: make([]grpc.StreamServerInterceptor, 0),
		config:                 config,
	}
}

// RegisterHTTPMiddleware registers an HTTP middleware function
func (m *MiddlewareManager) RegisterHTTPMiddleware(middleware HTTPMiddlewareFunc) {
	m.httpMiddlewares = append(m.httpMiddlewares, middleware)
}

// RegisterGRPCUnaryInterceptor registers a gRPC unary interceptor
func (m *MiddlewareManager) RegisterGRPCUnaryInterceptor(interceptor grpc.UnaryServerInterceptor) {
	m.grpcUnaryInterceptors = append(m.grpcUnaryInterceptors, interceptor)
}

// RegisterGRPCStreamInterceptor registers a gRPC stream interceptor
func (m *MiddlewareManager) RegisterGRPCStreamInterceptor(interceptor grpc.StreamServerInterceptor) {
	m.grpcStreamInterceptors = append(m.grpcStreamInterceptors, interceptor)
}

// ConfigureHTTPMiddleware configures HTTP middleware based on the server configuration
func (m *MiddlewareManager) ConfigureHTTPMiddleware(router *gin.Engine) error {
	if !m.config.IsHTTPEnabled() {
		return nil
	}

	middlewareConfig := m.config.HTTP.Middleware

	// Configure CORS middleware
	if middlewareConfig.EnableCORS {
		corsMiddleware := m.createCORSMiddleware(middlewareConfig.CORSConfig)
		if err := corsMiddleware(router); err != nil {
			return fmt.Errorf("failed to configure CORS middleware: %w", err)
		}
		logger.Logger.Debug("CORS middleware configured", zap.Any("config", middlewareConfig.CORSConfig))
	}

	// Configure logging middleware
	if middlewareConfig.EnableLogging {
		loggingMiddleware := m.createLoggingMiddleware()
		if err := loggingMiddleware(router); err != nil {
			return fmt.Errorf("failed to configure logging middleware: %w", err)
		}
		logger.Logger.Debug("Logging middleware configured")
	}

	// Configure timeout middleware
	if middlewareConfig.EnableTimeout {
		timeoutMiddleware := m.createTimeoutMiddleware(middlewareConfig.TimeoutConfig)
		if err := timeoutMiddleware(router); err != nil {
			return fmt.Errorf("failed to configure timeout middleware: %w", err)
		}
		logger.Logger.Debug("Timeout middleware configured", zap.Duration("timeout", middlewareConfig.TimeoutConfig.RequestTimeout))
	}

	// Configure rate limiting middleware
	if middlewareConfig.EnableRateLimit {
		rateLimitMiddleware := m.createRateLimitMiddleware(middlewareConfig.RateLimitConfig)
		if err := rateLimitMiddleware(router); err != nil {
			return fmt.Errorf("failed to configure rate limit middleware: %w", err)
		}
		logger.Logger.Debug("Rate limit middleware configured", zap.Any("config", middlewareConfig.RateLimitConfig))
	}

	// Configure authentication middleware
	if middlewareConfig.EnableAuth {
		authMiddleware := m.createAuthMiddleware()
		if err := authMiddleware(router); err != nil {
			return fmt.Errorf("failed to configure auth middleware: %w", err)
		}
		logger.Logger.Debug("Auth middleware configured")
	}

	// Configure custom headers middleware
	if len(middlewareConfig.CustomHeaders) > 0 {
		headersMiddleware := m.createCustomHeadersMiddleware(middlewareConfig.CustomHeaders)
		if err := headersMiddleware(router); err != nil {
			return fmt.Errorf("failed to configure custom headers middleware: %w", err)
		}
		logger.Logger.Debug("Custom headers middleware configured", zap.Any("headers", middlewareConfig.CustomHeaders))
	}

	// Configure Prometheus middleware if enabled
	if m.config.Prometheus.Enabled {
		prometheusMiddleware := m.createPrometheusHTTPMiddleware()
		if err := prometheusMiddleware(router); err != nil {
			return fmt.Errorf("failed to configure Prometheus middleware: %w", err)
		}
		logger.Logger.Debug("Prometheus HTTP middleware configured")
	}

	// Configure Sentry middleware if enabled
	if m.config.Sentry.Enabled && m.config.Sentry.IntegrateHTTP {
		sentryMiddleware := m.createSentryHTTPMiddleware()
		if err := sentryMiddleware(router); err != nil {
			return fmt.Errorf("failed to configure Sentry middleware: %w", err)
		}
		logger.Logger.Debug("Sentry HTTP middleware configured")
	}

	// Apply custom middleware functions
	for i, middlewareFunc := range m.httpMiddlewares {
		if err := middlewareFunc(router); err != nil {
			return fmt.Errorf("failed to apply custom HTTP middleware %d: %w", i, err)
		}
	}

	logger.Logger.Info("HTTP middleware configured",
		zap.Bool("cors", middlewareConfig.EnableCORS),
		zap.Bool("auth", middlewareConfig.EnableAuth),
		zap.Bool("rate_limit", middlewareConfig.EnableRateLimit),
		zap.Bool("logging", middlewareConfig.EnableLogging),
		zap.Bool("timeout", middlewareConfig.EnableTimeout),
		zap.Bool("prometheus", m.config.Prometheus.Enabled),
		zap.Bool("sentry", m.config.Sentry.Enabled && m.config.Sentry.IntegrateHTTP),
		zap.Int("custom_middleware_count", len(m.httpMiddlewares)))

	return nil
}

// GetGRPCInterceptors returns configured gRPC interceptors based on server configuration
func (m *MiddlewareManager) GetGRPCInterceptors() ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
	if !m.config.IsGRPCEnabled() {
		return nil, nil
	}

	interceptorConfig := m.config.GRPC.Interceptors
	unaryInterceptors := make([]grpc.UnaryServerInterceptor, 0)
	streamInterceptors := make([]grpc.StreamServerInterceptor, 0)

	// Add recovery interceptor (should be first)
	if interceptorConfig.EnableRecovery {
		unaryInterceptors = append(unaryInterceptors, m.createGRPCRecoveryUnaryInterceptor())
		streamInterceptors = append(streamInterceptors, m.createGRPCRecoveryStreamInterceptor())
		logger.Logger.Debug("gRPC recovery interceptors configured")
	}

	// Add logging interceptor
	if interceptorConfig.EnableLogging {
		unaryInterceptors = append(unaryInterceptors, m.createGRPCLoggingUnaryInterceptor())
		streamInterceptors = append(streamInterceptors, m.createGRPCLoggingStreamInterceptor())
		logger.Logger.Debug("gRPC logging interceptors configured")
	}

	// Add metrics interceptor
	if interceptorConfig.EnableMetrics {
		unaryInterceptors = append(unaryInterceptors, m.createGRPCMetricsUnaryInterceptor())
		streamInterceptors = append(streamInterceptors, m.createGRPCMetricsStreamInterceptor())
		logger.Logger.Debug("gRPC metrics interceptors configured")
	}

	// Add Prometheus interceptors if enabled
	if m.config.Prometheus.Enabled {
		prometheusUnaryInterceptor, prometheusStreamInterceptor := m.createPrometheusGRPCInterceptors()
		unaryInterceptors = append(unaryInterceptors, prometheusUnaryInterceptor)
		streamInterceptors = append(streamInterceptors, prometheusStreamInterceptor)
		logger.Logger.Debug("Prometheus gRPC interceptors configured")
	}

	// Add authentication interceptor
	if interceptorConfig.EnableAuth {
		unaryInterceptors = append(unaryInterceptors, m.createGRPCAuthUnaryInterceptor())
		streamInterceptors = append(streamInterceptors, m.createGRPCAuthStreamInterceptor())
		logger.Logger.Debug("gRPC auth interceptors configured")
	}

	// Add rate limiting interceptor
	if interceptorConfig.EnableRateLimit {
		unaryInterceptors = append(unaryInterceptors, m.createGRPCRateLimitUnaryInterceptor())
		streamInterceptors = append(streamInterceptors, m.createGRPCRateLimitStreamInterceptor())
		logger.Logger.Debug("gRPC rate limit interceptors configured")
	}

	// Add Sentry interceptors if enabled
	if m.config.Sentry.Enabled && m.config.Sentry.IntegrateGRPC {
		sentryUnary, sentryStream := m.createSentryGRPCInterceptors()
		unaryInterceptors = append(unaryInterceptors, sentryUnary)
		streamInterceptors = append(streamInterceptors, sentryStream)
		logger.Logger.Debug("Sentry gRPC interceptors configured")
	}

	// Add custom interceptors
	unaryInterceptors = append(unaryInterceptors, m.grpcUnaryInterceptors...)
	streamInterceptors = append(streamInterceptors, m.grpcStreamInterceptors...)

	logger.Logger.Info("gRPC interceptors configured",
		zap.Bool("recovery", interceptorConfig.EnableRecovery),
		zap.Bool("logging", interceptorConfig.EnableLogging),
		zap.Bool("metrics", interceptorConfig.EnableMetrics),
		zap.Bool("prometheus", m.config.Prometheus.Enabled),
		zap.Bool("auth", interceptorConfig.EnableAuth),
		zap.Bool("rate_limit", interceptorConfig.EnableRateLimit),
		zap.Bool("sentry", m.config.Sentry.Enabled && m.config.Sentry.IntegrateGRPC),
		zap.Int("custom_unary_count", len(m.grpcUnaryInterceptors)),
		zap.Int("custom_stream_count", len(m.grpcStreamInterceptors)))

	return unaryInterceptors, streamInterceptors
}

// createCORSMiddleware creates CORS middleware based on configuration
func (m *MiddlewareManager) createCORSMiddleware(config CORSConfig) HTTPMiddlewareFunc {
	return func(router *gin.Engine) error {
		// Create CORS middleware with server configuration instead of hardcoded values
		corsMiddleware := m.createConfigurableCORSMiddleware(config)
		router.Use(corsMiddleware)
		return nil
	}
}

// createConfigurableCORSMiddleware creates a CORS middleware with the provided configuration
func (m *MiddlewareManager) createConfigurableCORSMiddleware(config CORSConfig) gin.HandlerFunc {
	// Import the gin-contrib/cors package
	corsConfig := cors.DefaultConfig()

	// Apply server configuration
	corsConfig.AllowOrigins = config.AllowOrigins
	corsConfig.AllowMethods = config.AllowMethods
	corsConfig.AllowHeaders = config.AllowHeaders
	corsConfig.ExposeHeaders = config.ExposeHeaders
	corsConfig.AllowCredentials = config.AllowCredentials
	corsConfig.MaxAge = time.Duration(config.MaxAge) * time.Second

	// Log the CORS configuration for security auditing
	logger.Logger.Info("CORS middleware configured",
		zap.Strings("allow_origins", config.AllowOrigins),
		zap.Strings("allow_methods", config.AllowMethods),
		zap.Strings("allow_headers", config.AllowHeaders),
		zap.Strings("expose_headers", config.ExposeHeaders),
		zap.Bool("allow_credentials", config.AllowCredentials),
		zap.Int("max_age_seconds", config.MaxAge))

	return cors.New(corsConfig)
}

// createLoggingMiddleware creates logging middleware
func (m *MiddlewareManager) createLoggingMiddleware() HTTPMiddlewareFunc {
	return func(router *gin.Engine) error {
		loggingMiddleware := middleware.RequestLogger()
		router.Use(loggingMiddleware)
		return nil
	}
}

// createTimeoutMiddleware creates timeout middleware based on configuration
func (m *MiddlewareManager) createTimeoutMiddleware(config TimeoutConfig) HTTPMiddlewareFunc {
	return func(router *gin.Engine) error {
		timeoutMiddleware := middleware.TimeoutMiddleware(config.RequestTimeout)
		router.Use(timeoutMiddleware)
		return nil
	}
}

// createRateLimitMiddleware creates rate limiting middleware based on configuration
func (m *MiddlewareManager) createRateLimitMiddleware(config RateLimitConfig) HTTPMiddlewareFunc {
	return func(router *gin.Engine) error {
		// Create rate limiter with configuration
		rateLimiter := middleware.NewRateLimiter(config.RequestsPerSecond, config.WindowSize)
		rateLimitMiddleware := rateLimiter.Middleware()
		router.Use(rateLimitMiddleware)
		return nil
	}
}

// createAuthMiddleware creates authentication middleware
func (m *MiddlewareManager) createAuthMiddleware() HTTPMiddlewareFunc {
	return func(router *gin.Engine) error {
		authMiddleware := middleware.AuthMiddleware()
		router.Use(authMiddleware)
		return nil
	}
}

// createCustomHeadersMiddleware creates custom headers middleware
func (m *MiddlewareManager) createCustomHeadersMiddleware(headers map[string]string) HTTPMiddlewareFunc {
	return func(router *gin.Engine) error {
		headersMiddleware := func(c *gin.Context) {
			for key, value := range headers {
				c.Header(key, value)
			}
			c.Next()
		}
		router.Use(headersMiddleware)
		return nil
	}
}

// createPrometheusHTTPMiddleware creates Prometheus HTTP middleware
func (m *MiddlewareManager) createPrometheusHTTPMiddleware() HTTPMiddlewareFunc {
	return func(router *gin.Engine) error {
		// Create Prometheus metrics collector
		prometheusCollector := NewPrometheusMetricsCollector(&m.config.Prometheus)

		// Create HTTP middleware configuration
		httpConfig := middleware.DefaultPrometheusHTTPConfig()
		httpConfig.ServiceName = m.config.ServiceName
		httpConfig.MaxPathCardinality = m.config.Prometheus.CardinalityLimit

		// Create HTTP middleware
		prometheusMiddleware := middleware.NewPrometheusHTTPMiddleware(prometheusCollector, httpConfig)
		router.Use(prometheusMiddleware.Middleware())

		// Expose metrics endpoint
		router.GET(m.config.Prometheus.Endpoint, gin.WrapH(prometheusCollector.GetHandler()))

		logger.Logger.Info("Prometheus HTTP middleware configured",
			zap.String("endpoint", m.config.Prometheus.Endpoint),
			zap.String("namespace", m.config.Prometheus.Namespace),
			zap.String("service", m.config.ServiceName))
		return nil
	}
}

// createSentryHTTPMiddleware creates Sentry HTTP middleware
func (m *MiddlewareManager) createSentryHTTPMiddleware() HTTPMiddlewareFunc {
	return func(router *gin.Engine) error {
		sentryMiddleware := middleware.NewSentryHTTPMiddleware(middleware.SentryHTTPMiddlewareOptions{
			IgnorePaths:       m.config.Sentry.HTTPIgnorePaths,
			IgnoreStatusCodes: m.config.Sentry.HTTPIgnoreStatusCode,
		})
		router.Use(sentryMiddleware.GinMiddleware())
		return nil
	}
}

// getRateLimitKeyFunc returns the appropriate key function for rate limiting
func (m *MiddlewareManager) getRateLimitKeyFunc(keyFunc string) func(*gin.Context) string {
	switch keyFunc {
	case "ip":
		return func(c *gin.Context) string {
			return c.ClientIP()
		}
	case "user":
		return func(c *gin.Context) string {
			// Extract user ID from context or JWT token
			if userID, exists := c.Get("user_id"); exists {
				if uid, ok := userID.(string); ok {
					return uid
				}
			}
			// Fallback to IP if user ID not available
			return c.ClientIP()
		}
	case "custom":
		return func(c *gin.Context) string {
			// Custom key function - can be overridden by services
			return c.ClientIP()
		}
	default:
		return func(c *gin.Context) string {
			return c.ClientIP()
		}
	}
}

// gRPC interceptor creation methods

// createGRPCRecoveryUnaryInterceptor creates a gRPC recovery unary interceptor
func (m *MiddlewareManager) createGRPCRecoveryUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Logger.Error("gRPC unary handler panic recovered",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r))
			}
		}()
		return handler(ctx, req)
	}
}

// createGRPCRecoveryStreamInterceptor creates a gRPC recovery stream interceptor
func (m *MiddlewareManager) createGRPCRecoveryStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		defer func() {
			if r := recover(); r != nil {
				logger.Logger.Error("gRPC stream handler panic recovered",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r))
			}
		}()
		return handler(srv, stream)
	}
}

// createGRPCLoggingUnaryInterceptor creates a gRPC logging unary interceptor
func (m *MiddlewareManager) createGRPCLoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		logger.Logger.Info("gRPC unary call",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.Error(err))

		return resp, err
	}
}

// createGRPCLoggingStreamInterceptor creates a gRPC logging stream interceptor
func (m *MiddlewareManager) createGRPCLoggingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, stream)
		duration := time.Since(start)

		logger.Logger.Info("gRPC stream call",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.Error(err))

		return err
	}
}

// createGRPCMetricsUnaryInterceptor creates a gRPC metrics unary interceptor
func (m *MiddlewareManager) createGRPCMetricsUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// TODO: Implement metrics collection
		// This would typically integrate with a metrics system like Prometheus
		return handler(ctx, req)
	}
}

// createGRPCMetricsStreamInterceptor creates a gRPC metrics stream interceptor
func (m *MiddlewareManager) createGRPCMetricsStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// TODO: Implement metrics collection
		// This would typically integrate with a metrics system like Prometheus
		return handler(srv, stream)
	}
}

// createGRPCAuthUnaryInterceptor creates a gRPC authentication unary interceptor
func (m *MiddlewareManager) createGRPCAuthUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// TODO: Implement gRPC authentication
		// This would typically validate JWT tokens or other auth mechanisms
		return handler(ctx, req)
	}
}

// createGRPCAuthStreamInterceptor creates a gRPC authentication stream interceptor
func (m *MiddlewareManager) createGRPCAuthStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// TODO: Implement gRPC authentication
		// This would typically validate JWT tokens or other auth mechanisms
		return handler(srv, stream)
	}
}

// createGRPCRateLimitUnaryInterceptor creates a gRPC rate limiting unary interceptor
func (m *MiddlewareManager) createGRPCRateLimitUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// TODO: Implement gRPC rate limiting
		// This would typically use a rate limiter similar to HTTP middleware
		return handler(ctx, req)
	}
}

// createGRPCRateLimitStreamInterceptor creates a gRPC rate limiting stream interceptor
func (m *MiddlewareManager) createGRPCRateLimitStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// TODO: Implement gRPC rate limiting
		// This would typically use a rate limiter similar to HTTP middleware
		return handler(srv, stream)
	}
}

// createPrometheusGRPCInterceptors creates Prometheus gRPC interceptors
func (m *MiddlewareManager) createPrometheusGRPCInterceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	// Create Prometheus metrics collector
	prometheusCollector := NewPrometheusMetricsCollector(&m.config.Prometheus)

	// Create gRPC interceptor configuration
	grpcConfig := middleware.DefaultPrometheusGRPCConfig()
	grpcConfig.ServiceName = m.config.ServiceName

	// Create gRPC interceptor
	prometheusInterceptor := middleware.NewPrometheusGRPCInterceptor(prometheusCollector, grpcConfig)

	return prometheusInterceptor.UnaryServerInterceptor(), prometheusInterceptor.StreamServerInterceptor()
}

// createSentryGRPCInterceptors creates Sentry gRPC interceptors
func (m *MiddlewareManager) createSentryGRPCInterceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	sentryInterceptor := middleware.NewSentryGRPCInterceptor(middleware.SentryGRPCInterceptorOptions{
		// Default ignore codes - can be configured
		IgnoreStatusCodes: []codes.Code{codes.NotFound, codes.InvalidArgument},
	})
	return sentryInterceptor.UnaryServerInterceptor(), sentryInterceptor.StreamServerInterceptor()
}

// ServiceSpecificMiddlewareConfig allows services to customize middleware configuration
type ServiceSpecificMiddlewareConfig struct {
	ServiceName            string
	HTTPMiddleware         []HTTPMiddlewareFunc
	GRPCUnaryInterceptors  []grpc.UnaryServerInterceptor
	GRPCStreamInterceptors []grpc.StreamServerInterceptor
	OverrideHTTPConfig     *HTTPMiddleware
	OverrideGRPCConfig     *GRPCInterceptorConfig
}

// ApplyServiceSpecificMiddleware applies service-specific middleware configuration
func (m *MiddlewareManager) ApplyServiceSpecificMiddleware(config ServiceSpecificMiddlewareConfig) {
	// Add service-specific HTTP middleware
	for _, middleware := range config.HTTPMiddleware {
		m.RegisterHTTPMiddleware(middleware)
	}

	// Add service-specific gRPC interceptors
	for _, interceptor := range config.GRPCUnaryInterceptors {
		m.RegisterGRPCUnaryInterceptor(interceptor)
	}

	for _, interceptor := range config.GRPCStreamInterceptors {
		m.RegisterGRPCStreamInterceptor(interceptor)
	}

	// Override HTTP configuration if provided
	if config.OverrideHTTPConfig != nil {
		m.config.HTTP.Middleware = *config.OverrideHTTPConfig
	}

	// Override gRPC configuration if provided
	if config.OverrideGRPCConfig != nil {
		m.config.GRPC.Interceptors = *config.OverrideGRPCConfig
	}

	logger.Logger.Info("Service-specific middleware applied",
		zap.String("service", config.ServiceName),
		zap.Int("http_middleware_count", len(config.HTTPMiddleware)),
		zap.Int("grpc_unary_interceptor_count", len(config.GRPCUnaryInterceptors)),
		zap.Int("grpc_stream_interceptor_count", len(config.GRPCStreamInterceptors)))
}
