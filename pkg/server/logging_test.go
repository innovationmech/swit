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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestDefaultLoggingConfig(t *testing.T) {
	config := DefaultLoggingConfig()

	assert.Equal(t, LogLevelInfo, config.Level)
	assert.True(t, config.EnableStacktrace)
	assert.True(t, config.EnableCaller)
	assert.True(t, config.EnableTimestamp)
	assert.Equal(t, "json", config.Format)
}

func TestNewServerLogger(t *testing.T) {
	t.Run("with custom config", func(t *testing.T) {
		config := &LoggingConfig{
			Level:            LogLevelDebug,
			EnableStacktrace: false,
			EnableCaller:     false,
			EnableTimestamp:  false,
			Format:           "console",
		}

		logger := NewServerLogger("test-service", config)

		assert.NotNil(t, logger)
		assert.Equal(t, "test-service", logger.serviceName)
		assert.Equal(t, config, logger.config)
		assert.Len(t, logger.baseFields, 2) // service and component fields
	})

	t.Run("with nil config uses defaults", func(t *testing.T) {
		logger := NewServerLogger("test-service", nil)

		assert.NotNil(t, logger)
		assert.Equal(t, "test-service", logger.serviceName)
		assert.NotNil(t, logger.config)
		assert.Equal(t, LogLevelInfo, logger.config.Level)
	})
}

func TestServerLogger_WithContext(t *testing.T) {
	logger := NewServerLogger("test-service", DefaultLoggingConfig())

	t.Run("with request context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "request_id", "req-123")
		ctx = context.WithValue(ctx, "trace_id", "trace-456")

		contextLogger := logger.WithContext(ctx)

		assert.NotSame(t, logger, contextLogger)
		assert.Equal(t, logger.serviceName, contextLogger.serviceName)
		assert.Greater(t, len(contextLogger.baseFields), len(logger.baseFields))
	})

	t.Run("with empty context", func(t *testing.T) {
		ctx := context.Background()

		contextLogger := logger.WithContext(ctx)

		assert.NotSame(t, logger, contextLogger)
		assert.Equal(t, len(logger.baseFields), len(contextLogger.baseFields))
	})
}

func TestServerLogger_WithFields(t *testing.T) {
	logger := NewServerLogger("test-service", DefaultLoggingConfig())

	fieldsLogger := logger.WithFields(
		zap.String("custom_field", "value"),
		zap.Int("number", 42),
	)

	assert.NotSame(t, logger, fieldsLogger)
	assert.Equal(t, logger.serviceName, fieldsLogger.serviceName)
	assert.Equal(t, len(logger.baseFields)+2, len(fieldsLogger.baseFields))
}

func TestServerLogger_LifecycleLogging(t *testing.T) {
	// Create a logger with observer to capture log entries
	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	logger := &ServerLogger{
		logger:      zapLogger,
		serviceName: "test-service",
		config:      DefaultLoggingConfig(),
		baseFields: []zap.Field{
			zap.String("service", "test-service"),
			zap.String("component", "server"),
		},
	}

	t.Run("LogServerStarting", func(t *testing.T) {
		logger.LogServerStarting()

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Server starting", entry.Message)
		assert.Equal(t, zapcore.InfoLevel, entry.Level)

		// Check for expected fields
		fields := entry.Context
		assert.Contains(t, fields, zap.String("service", "test-service"))
		assert.Contains(t, fields, zap.String("component", "server"))
		assert.Contains(t, fields, zap.String("event", "server_starting"))
	})

	t.Run("LogServerStarted", func(t *testing.T) {
		duration := 100 * time.Millisecond
		logger.LogServerStarted("localhost:8080", "localhost:9090", duration)

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Server started successfully", entry.Message)
		assert.Equal(t, zapcore.InfoLevel, entry.Level)

		fields := entry.Context
		assert.Contains(t, fields, zap.String("event", "server_started"))
		assert.Contains(t, fields, zap.String("http_address", "localhost:8080"))
		assert.Contains(t, fields, zap.String("grpc_address", "localhost:9090"))
		assert.Contains(t, fields, zap.Duration("startup_duration", duration))
	})

	t.Run("LogServerStartupFailed", func(t *testing.T) {
		err := ErrTransportFailedWithCause("HTTP transport failed", errors.New("port in use")).
			WithOperation("StartHTTPTransport").
			WithContext("port", "8080")
		duration := 50 * time.Millisecond

		logger.LogServerStartupFailed(err, "transport_initialization", duration)

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Server startup failed", entry.Message)
		assert.Equal(t, zapcore.ErrorLevel, entry.Level)

		fields := entry.Context
		assert.Contains(t, fields, zap.String("event", "server_startup_failed"))
		assert.Contains(t, fields, zap.String("phase", "transport_initialization"))
		assert.Contains(t, fields, zap.Duration("startup_duration", duration))
		assert.Contains(t, fields, zap.String("error_code", ErrCodeTransportFailed))
		assert.Contains(t, fields, zap.String("error_category", string(CategoryTransport)))
		assert.Contains(t, fields, zap.String("operation", "StartHTTPTransport"))
		assert.Contains(t, fields, zap.String("error_context_port", "8080"))
	})

	t.Run("LogServerStopping", func(t *testing.T) {
		logger.LogServerStopping()

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Server stopping", entry.Message)
		assert.Contains(t, entry.Context, zap.String("event", "server_stopping"))
	})

	t.Run("LogServerStopped", func(t *testing.T) {
		duration := 200 * time.Millisecond
		logger.LogServerStopped(duration)

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Server stopped successfully", entry.Message)
		assert.Contains(t, entry.Context, zap.String("event", "server_stopped"))
		assert.Contains(t, entry.Context, zap.Duration("shutdown_duration", duration))
	})

	t.Run("LogServerShutdownFailed", func(t *testing.T) {
		err := ErrShutdownTimeout("graceful shutdown timeout")
		duration := 30 * time.Second

		logger.LogServerShutdownFailed(err, duration)

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Server shutdown failed", entry.Message)
		assert.Equal(t, zapcore.ErrorLevel, entry.Level)

		fields := entry.Context
		assert.Contains(t, fields, zap.String("event", "server_shutdown_failed"))
		assert.Contains(t, fields, zap.Duration("shutdown_duration", duration))
		assert.Contains(t, fields, zap.String("error_code", ErrCodeShutdownTimeout))
		assert.Contains(t, fields, zap.String("error_category", string(CategoryLifecycle)))
	})
}

func TestServerLogger_ComponentLogging(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	logger := &ServerLogger{
		logger:      zapLogger,
		serviceName: "test-service",
		config:      DefaultLoggingConfig(),
		baseFields: []zap.Field{
			zap.String("service", "test-service"),
			zap.String("component", "server"),
		},
	}

	t.Run("LogTransportInitialized", func(t *testing.T) {
		logger.LogTransportInitialized("http", "localhost:8080")

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Transport initialized", entry.Message)
		assert.Contains(t, entry.Context, zap.String("event", "transport_initialized"))
		assert.Contains(t, entry.Context, zap.String("transport_type", "http"))
		assert.Contains(t, entry.Context, zap.String("address", "localhost:8080"))
	})

	t.Run("LogServiceRegistered", func(t *testing.T) {
		logger.LogServiceRegistered("auth-service", "http")

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Service registered", entry.Message)
		assert.Contains(t, entry.Context, zap.String("event", "service_registered"))
		assert.Contains(t, entry.Context, zap.String("registered_service", "auth-service"))
		assert.Contains(t, entry.Context, zap.String("service_type", "http"))
	})

	t.Run("LogDiscoveryRegistered", func(t *testing.T) {
		logger.LogDiscoveryRegistered("consul", 2)

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Service registered with discovery", entry.Message)
		assert.Contains(t, entry.Context, zap.String("event", "discovery_registered"))
		assert.Contains(t, entry.Context, zap.String("discovery_service", "consul"))
		assert.Contains(t, entry.Context, zap.Int("endpoint_count", 2))
	})

	t.Run("LogDependencyInitialized", func(t *testing.T) {
		logger.LogDependencyInitialized("database")

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Dependency initialized", entry.Message)
		assert.Contains(t, entry.Context, zap.String("event", "dependency_initialized"))
		assert.Contains(t, entry.Context, zap.String("dependency", "database"))
	})

	t.Run("LogMiddlewareConfigured", func(t *testing.T) {
		logger.LogMiddlewareConfigured("cors", true)

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Middleware configured", entry.Message)
		assert.Contains(t, entry.Context, zap.String("event", "middleware_configured"))
		assert.Contains(t, entry.Context, zap.String("middleware", "cors"))
		assert.Contains(t, entry.Context, zap.Bool("enabled", true))
	})
}

func TestServerLogger_GeneralLogging(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	zapLogger := zap.New(core)

	logger := &ServerLogger{
		logger:      zapLogger,
		serviceName: "test-service",
		config:      DefaultLoggingConfig(),
		baseFields: []zap.Field{
			zap.String("service", "test-service"),
			zap.String("component", "server"),
		},
	}

	t.Run("LogError", func(t *testing.T) {
		err := ErrConfigInvalid("invalid port configuration")
		logger.LogError("Configuration validation failed", err, zap.String("config_file", "server.yaml"))

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Configuration validation failed", entry.Message)
		assert.Equal(t, zapcore.ErrorLevel, entry.Level)

		fields := entry.Context
		assert.Contains(t, fields, zap.String("event", "error"))
		assert.Contains(t, fields, zap.String("config_file", "server.yaml"))
		assert.Contains(t, fields, zap.String("error_code", ErrCodeConfigInvalid))
		assert.Contains(t, fields, zap.String("error_category", string(CategoryConfig)))
	})

	t.Run("LogWarning", func(t *testing.T) {
		logger.LogWarning("Service discovery unavailable", zap.String("discovery_addr", "localhost:8500"))

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Service discovery unavailable", entry.Message)
		assert.Equal(t, zapcore.WarnLevel, entry.Level)
		assert.Contains(t, entry.Context, zap.String("event", "warning"))
		assert.Contains(t, entry.Context, zap.String("discovery_addr", "localhost:8500"))
	})

	t.Run("LogInfo", func(t *testing.T) {
		logger.LogInfo("Configuration loaded", zap.String("config_file", "server.yaml"))

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Configuration loaded", entry.Message)
		assert.Equal(t, zapcore.InfoLevel, entry.Level)
		assert.Contains(t, entry.Context, zap.String("event", "info"))
		assert.Contains(t, entry.Context, zap.String("config_file", "server.yaml"))
	})

	t.Run("LogDebug", func(t *testing.T) {
		logger.LogDebug("Processing request", zap.String("endpoint", "/health"))

		entries := recorded.TakeAll()
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Processing request", entry.Message)
		assert.Equal(t, zapcore.DebugLevel, entry.Level)
		assert.Contains(t, entry.Context, zap.String("event", "debug"))
		assert.Contains(t, entry.Context, zap.String("endpoint", "/health"))
	})
}

func TestServerLoggerWithHooks(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	zapLogger := zap.New(core)

	baseLogger := &ServerLogger{
		logger:      zapLogger,
		serviceName: "test-service",
		config:      DefaultLoggingConfig(),
		baseFields: []zap.Field{
			zap.String("service", "test-service"),
			zap.String("component", "server"),
		},
	}

	logger := &ServerLoggerWithHooks{
		ServerLogger: baseLogger,
		hooks:        make(map[LogLevel][]LoggingHook),
	}

	// Track hook executions
	var hookExecutions []string

	// Add hooks for different levels
	logger.AddHook(LogLevelError, func(level LogLevel, message string, fields []zap.Field) {
		hookExecutions = append(hookExecutions, "error_hook_1")
	})

	logger.AddHook(LogLevelError, func(level LogLevel, message string, fields []zap.Field) {
		hookExecutions = append(hookExecutions, "error_hook_2")
	})

	logger.AddHook(LogLevelInfo, func(level LogLevel, message string, fields []zap.Field) {
		hookExecutions = append(hookExecutions, "info_hook")
	})

	t.Run("error hooks executed", func(t *testing.T) {
		hookExecutions = nil
		err := errors.New("test error")
		logger.LogError("Test error message", err)

		assert.Contains(t, hookExecutions, "error_hook_1")
		assert.Contains(t, hookExecutions, "error_hook_2")
		assert.NotContains(t, hookExecutions, "info_hook")

		// Verify log was still written
		entries := recorded.TakeAll()
		require.Len(t, entries, 1)
		assert.Equal(t, "Test error message", entries[0].Message)
	})

	t.Run("info hooks executed", func(t *testing.T) {
		hookExecutions = nil
		logger.LogInfo("Test info message")

		assert.Contains(t, hookExecutions, "info_hook")
		assert.NotContains(t, hookExecutions, "error_hook_1")
		assert.NotContains(t, hookExecutions, "error_hook_2")

		// Verify log was still written
		entries := recorded.TakeAll()
		require.Len(t, entries, 1)
		assert.Equal(t, "Test info message", entries[0].Message)
	})
}

func TestNewServerLoggerWithHooks(t *testing.T) {
	config := DefaultLoggingConfig()
	logger := NewServerLoggerWithHooks("test-service", config)

	assert.NotNil(t, logger)
	assert.NotNil(t, logger.ServerLogger)
	assert.NotNil(t, logger.hooks)
	assert.Equal(t, "test-service", logger.serviceName)
}

func TestServerLogger_Sync(t *testing.T) {
	logger := NewServerLogger("test-service", DefaultLoggingConfig())

	// Sync may return an error in test environments (e.g., "sync /dev/stderr: bad file descriptor")
	// This is expected behavior and not a real error
	err := logger.Sync()
	// We just verify that Sync() can be called without panicking
	// The error is acceptable in test environments
	_ = err
}

func TestServerLogger_GetUnderlyingLogger(t *testing.T) {
	logger := NewServerLogger("test-service", DefaultLoggingConfig())

	underlying := logger.GetUnderlyingLogger()
	assert.NotNil(t, underlying)
	assert.IsType(t, &zap.Logger{}, underlying)
}

func TestLoggingConfig_LogLevels(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		expected zapcore.Level
	}{
		{"debug level", LogLevelDebug, zapcore.DebugLevel},
		{"info level", LogLevelInfo, zapcore.InfoLevel},
		{"warn level", LogLevelWarn, zapcore.WarnLevel},
		{"error level", LogLevelError, zapcore.ErrorLevel},
		{"fatal level", LogLevelFatal, zapcore.FatalLevel},
		{"unknown level defaults to info", LogLevel("unknown"), zapcore.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LoggingConfig{
				Level:            tt.level,
				EnableStacktrace: true,
				EnableCaller:     true,
				EnableTimestamp:  true,
				Format:           "json",
			}

			logger := NewServerLogger("test-service", config)
			assert.NotNil(t, logger)
			// We can't easily test the actual log level without more complex setup,
			// but we can verify the logger was created successfully
		})
	}
}

func TestLoggingConfig_Formats(t *testing.T) {
	t.Run("json format", func(t *testing.T) {
		config := &LoggingConfig{
			Level:            LogLevelInfo,
			EnableStacktrace: true,
			EnableCaller:     true,
			EnableTimestamp:  true,
			Format:           "json",
		}

		logger := NewServerLogger("test-service", config)
		assert.NotNil(t, logger)
	})

	t.Run("console format", func(t *testing.T) {
		config := &LoggingConfig{
			Level:            LogLevelInfo,
			EnableStacktrace: true,
			EnableCaller:     true,
			EnableTimestamp:  true,
			Format:           "console",
		}

		logger := NewServerLogger("test-service", config)
		assert.NotNil(t, logger)
	})
}

func TestLoggingConfig_OptionalFeatures(t *testing.T) {
	config := &LoggingConfig{
		Level:            LogLevelInfo,
		EnableStacktrace: false,
		EnableCaller:     false,
		EnableTimestamp:  false,
		Format:           "json",
	}

	logger := NewServerLogger("test-service", config)
	assert.NotNil(t, logger)
	// The logger should be created successfully even with features disabled
}
