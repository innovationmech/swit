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
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogLevel represents different logging levels
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// LoggingConfig represents the logging configuration for the server
type LoggingConfig struct {
	Level            LogLevel `yaml:"level" json:"level"`
	EnableStacktrace bool     `yaml:"enable_stacktrace" json:"enable_stacktrace"`
	EnableCaller     bool     `yaml:"enable_caller" json:"enable_caller"`
	EnableTimestamp  bool     `yaml:"enable_timestamp" json:"enable_timestamp"`
	Format           string   `yaml:"format" json:"format"` // "json" or "console"
}

// DefaultLoggingConfig returns the default logging configuration
func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Level:            LogLevelInfo,
		EnableStacktrace: true,
		EnableCaller:     true,
		EnableTimestamp:  true,
		Format:           "json",
	}
}

// ServerLogger provides structured logging for server operations with consistent context
type ServerLogger struct {
	logger      *zap.Logger
	serviceName string
	config      *LoggingConfig
	baseFields  []zap.Field
}

// NewServerLogger creates a new server logger with the given configuration
func NewServerLogger(serviceName string, config *LoggingConfig) *ServerLogger {
	if config == nil {
		config = DefaultLoggingConfig()
	}

	// Create zap config based on our configuration
	zapConfig := zap.NewProductionConfig()

	// Set log level
	switch config.Level {
	case LogLevelDebug:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case LogLevelInfo:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case LogLevelWarn:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case LogLevelError:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case LogLevelFatal:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	// Configure output format
	if config.Format == "console" {
		zapConfig.Encoding = "console"
		zapConfig.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	} else {
		zapConfig.Encoding = "json"
		zapConfig.EncoderConfig = zap.NewProductionEncoderConfig()
	}

	// Configure optional features
	if !config.EnableTimestamp {
		zapConfig.EncoderConfig.TimeKey = ""
	}
	if !config.EnableCaller {
		zapConfig.DisableCaller = true
	}
	if !config.EnableStacktrace {
		zapConfig.DisableStacktrace = true
	}

	// Build the logger
	logger, err := zapConfig.Build()
	if err != nil {
		// Fallback to a basic logger if configuration fails
		logger = zap.NewNop()
	}

	// Create base fields that will be included in all log entries
	baseFields := []zap.Field{
		zap.String("service", serviceName),
		zap.String("component", "server"),
	}

	return &ServerLogger{
		logger:      logger,
		serviceName: serviceName,
		config:      config,
		baseFields:  baseFields,
	}
}

// WithContext creates a new logger with additional context fields
func (sl *ServerLogger) WithContext(ctx context.Context) *ServerLogger {
	fields := make([]zap.Field, len(sl.baseFields))
	copy(fields, sl.baseFields)

	// Add context-specific fields if available
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			fields = append(fields, zap.String("request_id", id))
		}
	}

	if traceID := ctx.Value("trace_id"); traceID != nil {
		if id, ok := traceID.(string); ok {
			fields = append(fields, zap.String("trace_id", id))
		}
	}

	return &ServerLogger{
		logger:      sl.logger,
		serviceName: sl.serviceName,
		config:      sl.config,
		baseFields:  fields,
	}
}

// WithFields creates a new logger with additional fields
func (sl *ServerLogger) WithFields(fields ...zap.Field) *ServerLogger {
	allFields := make([]zap.Field, len(sl.baseFields)+len(fields))
	copy(allFields, sl.baseFields)
	copy(allFields[len(sl.baseFields):], fields)

	return &ServerLogger{
		logger:      sl.logger,
		serviceName: sl.serviceName,
		config:      sl.config,
		baseFields:  allFields,
	}
}

// Server lifecycle logging methods

// LogServerStarting logs server startup initiation
func (sl *ServerLogger) LogServerStarting() {
	sl.logger.Info("Server starting",
		append(sl.baseFields,
			zap.String("event", "server_starting"),
			zap.Time("timestamp", time.Now()),
		)...)
}

// LogServerStarted logs successful server startup
func (sl *ServerLogger) LogServerStarted(httpAddr, grpcAddr string, startupDuration time.Duration) {
	fields := append(sl.baseFields,
		zap.String("event", "server_started"),
		zap.Time("timestamp", time.Now()),
		zap.Duration("startup_duration", startupDuration),
	)

	if httpAddr != "" {
		fields = append(fields, zap.String("http_address", httpAddr))
	}
	if grpcAddr != "" {
		fields = append(fields, zap.String("grpc_address", grpcAddr))
	}

	sl.logger.Info("Server started successfully", fields...)
}

// LogServerStartupFailed logs server startup failure
func (sl *ServerLogger) LogServerStartupFailed(err error, phase string, startupDuration time.Duration) {
	fields := append(sl.baseFields,
		zap.String("event", "server_startup_failed"),
		zap.Time("timestamp", time.Now()),
		zap.Error(err),
		zap.String("phase", phase),
		zap.Duration("startup_duration", startupDuration),
	)

	// Add server error context if available
	if serverErr := GetServerError(err); serverErr != nil {
		fields = append(fields,
			zap.String("error_code", serverErr.Code),
			zap.String("error_category", string(serverErr.Category)),
		)
		if serverErr.Operation != "" {
			fields = append(fields, zap.String("operation", serverErr.Operation))
		}
		if len(serverErr.Context) > 0 {
			for key, value := range serverErr.Context {
				fields = append(fields, zap.String("error_context_"+key, value))
			}
		}
	}

	sl.logger.Error("Server startup failed", fields...)
}

// LogServerStopping logs server shutdown initiation
func (sl *ServerLogger) LogServerStopping() {
	sl.logger.Info("Server stopping",
		append(sl.baseFields,
			zap.String("event", "server_stopping"),
			zap.Time("timestamp", time.Now()),
		)...)
}

// LogServerStopped logs successful server shutdown
func (sl *ServerLogger) LogServerStopped(shutdownDuration time.Duration) {
	sl.logger.Info("Server stopped successfully",
		append(sl.baseFields,
			zap.String("event", "server_stopped"),
			zap.Time("timestamp", time.Now()),
			zap.Duration("shutdown_duration", shutdownDuration),
		)...)
}

// LogServerShutdownFailed logs server shutdown failure
func (sl *ServerLogger) LogServerShutdownFailed(err error, shutdownDuration time.Duration) {
	fields := append(sl.baseFields,
		zap.String("event", "server_shutdown_failed"),
		zap.Time("timestamp", time.Now()),
		zap.Error(err),
		zap.Duration("shutdown_duration", shutdownDuration),
	)

	// Add server error context if available
	if serverErr := GetServerError(err); serverErr != nil {
		fields = append(fields,
			zap.String("error_code", serverErr.Code),
			zap.String("error_category", string(serverErr.Category)),
		)
	}

	sl.logger.Error("Server shutdown failed", fields...)
}

// Component lifecycle logging methods

// LogTransportInitialized logs transport initialization
func (sl *ServerLogger) LogTransportInitialized(transportType, address string) {
	sl.logger.Info("Transport initialized",
		append(sl.baseFields,
			zap.String("event", "transport_initialized"),
			zap.String("transport_type", transportType),
			zap.String("address", address),
			zap.Time("timestamp", time.Now()),
		)...)
}

// LogTransportStarted logs transport startup
func (sl *ServerLogger) LogTransportStarted(transportType, address string) {
	sl.logger.Info("Transport started",
		append(sl.baseFields,
			zap.String("event", "transport_started"),
			zap.String("transport_type", transportType),
			zap.String("address", address),
			zap.Time("timestamp", time.Now()),
		)...)
}

// LogTransportStopped logs transport shutdown
func (sl *ServerLogger) LogTransportStopped(transportType string, shutdownDuration time.Duration) {
	sl.logger.Info("Transport stopped",
		append(sl.baseFields,
			zap.String("event", "transport_stopped"),
			zap.String("transport_type", transportType),
			zap.Duration("shutdown_duration", shutdownDuration),
			zap.Time("timestamp", time.Now()),
		)...)
}

// LogServiceRegistered logs service registration
func (sl *ServerLogger) LogServiceRegistered(serviceName, serviceType string) {
	sl.logger.Info("Service registered",
		append(sl.baseFields,
			zap.String("event", "service_registered"),
			zap.String("registered_service", serviceName),
			zap.String("service_type", serviceType),
			zap.Time("timestamp", time.Now()),
		)...)
}

// LogDiscoveryRegistered logs service discovery registration
func (sl *ServerLogger) LogDiscoveryRegistered(discoveryService string, endpointCount int) {
	sl.logger.Info("Service registered with discovery",
		append(sl.baseFields,
			zap.String("event", "discovery_registered"),
			zap.String("discovery_service", discoveryService),
			zap.Int("endpoint_count", endpointCount),
			zap.Time("timestamp", time.Now()),
		)...)
}

// LogDiscoveryDeregistered logs service discovery deregistration
func (sl *ServerLogger) LogDiscoveryDeregistered(discoveryService string, endpointCount int) {
	sl.logger.Info("Service deregistered from discovery",
		append(sl.baseFields,
			zap.String("event", "discovery_deregistered"),
			zap.String("discovery_service", discoveryService),
			zap.Int("endpoint_count", endpointCount),
			zap.Time("timestamp", time.Now()),
		)...)
}

// LogDependencyInitialized logs dependency initialization
func (sl *ServerLogger) LogDependencyInitialized(dependencyName string) {
	sl.logger.Info("Dependency initialized",
		append(sl.baseFields,
			zap.String("event", "dependency_initialized"),
			zap.String("dependency", dependencyName),
			zap.Time("timestamp", time.Now()),
		)...)
}

// LogMiddlewareConfigured logs middleware configuration
func (sl *ServerLogger) LogMiddlewareConfigured(middlewareName string, enabled bool) {
	sl.logger.Info("Middleware configured",
		append(sl.baseFields,
			zap.String("event", "middleware_configured"),
			zap.String("middleware", middlewareName),
			zap.Bool("enabled", enabled),
			zap.Time("timestamp", time.Now()),
		)...)
}

// Error logging methods

// LogError logs a general error with server context
func (sl *ServerLogger) LogError(message string, err error, fields ...zap.Field) {
	allFields := append(sl.baseFields,
		zap.String("event", "error"),
		zap.Error(err),
		zap.Time("timestamp", time.Now()),
	)
	allFields = append(allFields, fields...)

	// Add server error context if available
	if serverErr := GetServerError(err); serverErr != nil {
		allFields = append(allFields,
			zap.String("error_code", serverErr.Code),
			zap.String("error_category", string(serverErr.Category)),
		)
		if serverErr.Operation != "" {
			allFields = append(allFields, zap.String("operation", serverErr.Operation))
		}
	}

	sl.logger.Error(message, allFields...)
}

// LogWarning logs a warning with server context
func (sl *ServerLogger) LogWarning(message string, fields ...zap.Field) {
	allFields := append(sl.baseFields,
		zap.String("event", "warning"),
		zap.Time("timestamp", time.Now()),
	)
	allFields = append(allFields, fields...)

	sl.logger.Warn(message, allFields...)
}

// LogInfo logs an informational message with server context
func (sl *ServerLogger) LogInfo(message string, fields ...zap.Field) {
	allFields := append(sl.baseFields,
		zap.String("event", "info"),
		zap.Time("timestamp", time.Now()),
	)
	allFields = append(allFields, fields...)

	sl.logger.Info(message, allFields...)
}

// LogDebug logs a debug message with server context
func (sl *ServerLogger) LogDebug(message string, fields ...zap.Field) {
	allFields := append(sl.baseFields,
		zap.String("event", "debug"),
		zap.Time("timestamp", time.Now()),
	)
	allFields = append(allFields, fields...)

	sl.logger.Debug(message, allFields...)
}

// GetUnderlyingLogger returns the underlying zap logger for advanced usage
func (sl *ServerLogger) GetUnderlyingLogger() *zap.Logger {
	return sl.logger
}

// Sync flushes any buffered log entries
func (sl *ServerLogger) Sync() error {
	return sl.logger.Sync()
}

// LoggingHook represents a function that can be called on specific logging events
type LoggingHook func(level LogLevel, message string, fields []zap.Field)

// ServerLoggerWithHooks extends ServerLogger with hook support
type ServerLoggerWithHooks struct {
	*ServerLogger
	hooks map[LogLevel][]LoggingHook
}

// NewServerLoggerWithHooks creates a new server logger with hook support
func NewServerLoggerWithHooks(serviceName string, config *LoggingConfig) *ServerLoggerWithHooks {
	return &ServerLoggerWithHooks{
		ServerLogger: NewServerLogger(serviceName, config),
		hooks:        make(map[LogLevel][]LoggingHook),
	}
}

// AddHook adds a logging hook for the specified level
func (slh *ServerLoggerWithHooks) AddHook(level LogLevel, hook LoggingHook) {
	if slh.hooks[level] == nil {
		slh.hooks[level] = make([]LoggingHook, 0)
	}
	slh.hooks[level] = append(slh.hooks[level], hook)
}

// executeHooks executes all hooks for the given level
func (slh *ServerLoggerWithHooks) executeHooks(level LogLevel, message string, fields []zap.Field) {
	if hooks, exists := slh.hooks[level]; exists {
		for _, hook := range hooks {
			hook(level, message, fields)
		}
	}
}

// Override logging methods to execute hooks

// LogError with hooks
func (slh *ServerLoggerWithHooks) LogError(message string, err error, fields ...zap.Field) {
	allFields := append(slh.baseFields, fields...)
	allFields = append(allFields, zap.Error(err))
	slh.executeHooks(LogLevelError, message, allFields)
	slh.ServerLogger.LogError(message, err, fields...)
}

// LogWarning with hooks
func (slh *ServerLoggerWithHooks) LogWarning(message string, fields ...zap.Field) {
	allFields := append(slh.baseFields, fields...)
	slh.executeHooks(LogLevelWarn, message, allFields)
	slh.ServerLogger.LogWarning(message, fields...)
}

// LogInfo with hooks
func (slh *ServerLoggerWithHooks) LogInfo(message string, fields ...zap.Field) {
	allFields := append(slh.baseFields, fields...)
	slh.executeHooks(LogLevelInfo, message, allFields)
	slh.ServerLogger.LogInfo(message, fields...)
}

// LogDebug with hooks
func (slh *ServerLoggerWithHooks) LogDebug(message string, fields ...zap.Field) {
	allFields := append(slh.baseFields, fields...)
	slh.executeHooks(LogLevelDebug, message, allFields)
	slh.ServerLogger.LogDebug(message, fields...)
}
