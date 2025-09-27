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

package logger

import (
	"context"
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Logger is the global logger for the application.
	Logger *zap.Logger
	// SugaredLogger provides a more ergonomic API for common logging patterns
	SugaredLogger *zap.SugaredLogger
	// mu protects Logger from concurrent access
	mu sync.RWMutex
	// initialized tracks whether logger has been initialized
	initialized bool
	// logLevel holds the atomic log level to support runtime updates
	logLevel zap.AtomicLevel
)

// LoggerConfig holds configuration for the logger
type LoggerConfig struct {
	Level              string   `yaml:"level" json:"level"`
	Development        bool     `yaml:"development" json:"development"`
	Encoding           string   `yaml:"encoding" json:"encoding"`
	OutputPaths        []string `yaml:"output_paths" json:"output_paths"`
	ErrorOutputPaths   []string `yaml:"error_output_paths" json:"error_output_paths"`
	DisableCaller      bool     `yaml:"disable_caller" json:"disable_caller"`
	DisableStacktrace  bool     `yaml:"disable_stacktrace" json:"disable_stacktrace"`
	SamplingEnabled    bool     `yaml:"sampling_enabled" json:"sampling_enabled"`
	SamplingInitial    int      `yaml:"sampling_initial" json:"sampling_initial"`
	SamplingThereafter int      `yaml:"sampling_thereafter" json:"sampling_thereafter"`
}

// SetDefaults sets default values for logger configuration
func (c *LoggerConfig) SetDefaults() {
	if c.Level == "" {
		if c.Development {
			c.Level = "debug"
		} else {
			c.Level = "info"
		}
	}
	if c.Encoding == "" {
		if c.Development {
			c.Encoding = "console"
		} else {
			c.Encoding = "json"
		}
	}
	if len(c.OutputPaths) == 0 {
		c.OutputPaths = []string{"stdout"}
	}
	if len(c.ErrorOutputPaths) == 0 {
		c.ErrorOutputPaths = []string{"stderr"}
	}
	if c.SamplingEnabled && c.SamplingInitial == 0 {
		c.SamplingInitial = 100
	}
	if c.SamplingEnabled && c.SamplingThereafter == 0 {
		c.SamplingThereafter = 100
	}
}

// InitLogger initializes the global logger safely to prevent race conditions.
func InitLogger() {
	InitLoggerWithConfig(nil)
}

// InitLoggerWithConfig initializes the global logger with custom configuration
func InitLoggerWithConfig(config *LoggerConfig) {
	mu.Lock()
	defer mu.Unlock()

	// Only initialize if not already done
	if initialized && Logger != nil {
		return
	}

	if config == nil {
		config = &LoggerConfig{}
	}
	config.SetDefaults()

	var err error
	Logger, err = buildLogger(config)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	SugaredLogger = Logger.Sugar()
	initialized = true
}

// buildLogger creates a logger based on the configuration
func buildLogger(config *LoggerConfig) (*zap.Logger, error) {
	var zapConfig zap.Config

	if config.Development {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	// Set log level using an atomic level so it can be changed at runtime
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", config.Level, err)
	}
	// keep a reference to the atomic level for runtime updates
	logLevel = zap.NewAtomicLevelAt(level)
	zapConfig.Level = logLevel

	// Set encoding
	zapConfig.Encoding = config.Encoding

	// Set output paths
	zapConfig.OutputPaths = config.OutputPaths
	zapConfig.ErrorOutputPaths = config.ErrorOutputPaths

	// Configure caller and stacktrace
	zapConfig.DisableCaller = config.DisableCaller
	zapConfig.DisableStacktrace = config.DisableStacktrace

	// Configure sampling if enabled
	if config.SamplingEnabled {
		zapConfig.Sampling = &zap.SamplingConfig{
			Initial:    config.SamplingInitial,
			Thereafter: config.SamplingThereafter,
		}
	} else {
		zapConfig.Sampling = nil
	}

	// Add initial fields
	zapConfig.InitialFields = map[string]interface{}{
		"pid":      os.Getpid(),
		"hostname": getHostname(),
	}

	// Customize encoder config for better output
	if config.Encoding == "console" {
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	return zapConfig.Build()
}

// getHostname returns the hostname or "unknown" if it cannot be determined
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// ResetLogger resets the logger for testing purposes.
// This should only be used in tests.
func ResetLogger() {
	mu.Lock()
	defer mu.Unlock()

	if Logger != nil {
		Logger.Sync() // Flush any pending log entries
	}
	Logger = nil
	SugaredLogger = nil
	initialized = false
}

// GetLogger returns the global logger, initializing it if necessary
func GetLogger() *zap.Logger {
	mu.RLock()
	if initialized && Logger != nil {
		mu.RUnlock()
		return Logger
	}
	mu.RUnlock()

	InitLogger()
	return Logger
}

// GetSugaredLogger returns the global sugared logger, initializing it if necessary
func GetSugaredLogger() *zap.SugaredLogger {
	mu.RLock()
	if initialized && SugaredLogger != nil {
		mu.RUnlock()
		return SugaredLogger
	}
	mu.RUnlock()

	InitLogger()
	return SugaredLogger
}

// SetLevel updates the global logger level at runtime.
// Supported levels: debug, info, warn, error, dpanic, panic, fatal
func SetLevel(level string) error {
	mu.RLock()
	initializedLocal := initialized
	mu.RUnlock()

	if !initializedLocal {
		return fmt.Errorf("logger is not initialized")
	}

	parsed, err := zapcore.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("invalid log level %s: %w", level, err)
	}
	logLevel.SetLevel(parsed)
	return nil
}

// GetLevel returns the current global logger level as string.
func GetLevel() string {
	mu.RLock()
	defer mu.RUnlock()
	return logLevel.Level().String()
}

// WithContext returns a logger with context fields
func WithContext(ctx context.Context) *zap.Logger {
	logger := GetLogger()

	// Extract trace ID if present
	if traceID := extractTraceID(ctx); traceID != "" {
		logger = logger.With(zap.String("trace_id", traceID))
	}

	// Extract request ID if present
	if requestID := extractRequestID(ctx); requestID != "" {
		logger = logger.With(zap.String("request_id", requestID))
	}

	// Extract user ID if present
	if userID := extractUserID(ctx); userID != "" {
		logger = logger.With(zap.String("user_id", userID))
	}

	return logger
}

// WithFields returns a logger with additional fields
func WithFields(fields ...zap.Field) *zap.Logger {
	return GetLogger().With(fields...)
}

// WithServiceInfo returns a logger with service metadata
func WithServiceInfo(serviceName, version, environment string) *zap.Logger {
	return GetLogger().With(
		zap.String("service", serviceName),
		zap.String("version", version),
		zap.String("environment", environment),
	)
}

// ContextKey is a type for context keys
type ContextKey string

const (
	// ContextKeyTraceID is the context key for trace ID
	ContextKeyTraceID ContextKey = "trace_id"
	// ContextKeyRequestID is the context key for request ID
	ContextKeyRequestID ContextKey = "request_id"
	// ContextKeyUserID is the context key for user ID
	ContextKeyUserID ContextKey = "user_id"
)

// extractTraceID extracts trace ID from context
func extractTraceID(ctx context.Context) string {
	if v := ctx.Value(ContextKeyTraceID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// extractRequestID extracts request ID from context
func extractRequestID(ctx context.Context) string {
	if v := ctx.Value(ContextKeyRequestID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// extractUserID extracts user ID from context
func extractUserID(ctx context.Context) string {
	if v := ctx.Value(ContextKeyUserID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// LogPerformance logs performance metrics with structured fields
func LogPerformance(operation string, duration float64, success bool, metadata map[string]interface{}) {
	fields := []zap.Field{
		zap.String("operation", operation),
		zap.Float64("duration_ms", duration),
		zap.Bool("success", success),
	}

	for k, v := range metadata {
		fields = append(fields, zap.Any(k, v))
	}

	if success {
		GetLogger().Info("operation completed", fields...)
	} else {
		GetLogger().Warn("operation failed", fields...)
	}
}

// LogRequest logs HTTP request details
func LogRequest(method, path string, statusCode int, duration float64, clientIP string) {
	fields := []zap.Field{
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status", statusCode),
		zap.Float64("duration_ms", duration),
		zap.String("client_ip", clientIP),
	}

	logger := GetLogger()
	if statusCode >= 500 {
		logger.Error("request failed", fields...)
	} else if statusCode >= 400 {
		logger.Warn("request client error", fields...)
	} else {
		logger.Info("request completed", fields...)
	}
}

// LogGRPCCall logs gRPC call details
func LogGRPCCall(method string, duration float64, err error, metadata map[string]interface{}) {
	fields := []zap.Field{
		zap.String("grpc_method", method),
		zap.Float64("duration_ms", duration),
	}

	for k, v := range metadata {
		fields = append(fields, zap.Any(k, v))
	}

	logger := GetLogger()
	if err != nil {
		fields = append(fields, zap.Error(err))
		logger.Error("gRPC call failed", fields...)
	} else {
		logger.Info("gRPC call completed", fields...)
	}
}

// CreateServiceLogger creates a logger instance for a specific service
func CreateServiceLogger(serviceName string, config *LoggerConfig) (*zap.Logger, error) {
	if config == nil {
		config = &LoggerConfig{}
	}
	config.SetDefaults()

	logger, err := buildLogger(config)
	if err != nil {
		return nil, err
	}

	return logger.With(zap.String("service", serviceName)), nil
}
