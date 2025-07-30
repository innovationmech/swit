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
	"log/slog"
	"os"
	"time"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogFormat represents the log output format
type LogFormat string

const (
	LogFormatJSON LogFormat = "json"
	LogFormatText LogFormat = "text"
)

// LogEvent represents different server lifecycle events
type LogEvent string

const (
	EventServerStartup    LogEvent = "server.startup"
	EventServerShutdown   LogEvent = "server.shutdown"
	EventServerError      LogEvent = "server.error"
	EventTransportStart   LogEvent = "transport.start"
	EventTransportStop    LogEvent = "transport.stop"
	EventServiceRegister  LogEvent = "service.register"
	EventDiscoveryConnect LogEvent = "discovery.connect"
	EventHealthCheck      LogEvent = "health.check"
	EventConfigLoad       LogEvent = "config.load"
	EventDependencyInit   LogEvent = "dependency.init"
)

// LogConfig represents logging configuration
type LogConfig struct {
	Level      LogLevel  `yaml:"level" json:"level"`
	Format     LogFormat `yaml:"format" json:"format"`
	Output     string    `yaml:"output" json:"output"`           // file path or "stdout", "stderr"
	MaxSize    int       `yaml:"max_size" json:"max_size"`       // MB
	MaxBackups int       `yaml:"max_backups" json:"max_backups"` // number of backup files
	MaxAge     int       `yaml:"max_age" json:"max_age"`         // days
	Compress   bool      `yaml:"compress" json:"compress"`
}

// DefaultLogConfig returns default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:      LogLevelInfo,
		Format:     LogFormatJSON,
		Output:     "stdout",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}
}

// LogHook represents a function that can be called on log events
type LogHook func(ctx context.Context, event LogEvent, fields map[string]interface{})

// Logger represents the server logger interface
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	With(key string, value interface{}) Logger
	WithContext(ctx context.Context) Logger
	LogEvent(ctx context.Context, event LogEvent, fields map[string]interface{})
	AddHook(hook LogHook)
	RemoveHook(hook LogHook)
}

// StructuredLogger implements the Logger interface using slog
type StructuredLogger struct {
	logger *slog.Logger
	hooks  []LogHook
	fields map[string]interface{}
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config *LogConfig) (*StructuredLogger, error) {
	if config == nil {
		config = DefaultLogConfig()
	}

	// Convert log level
	var level slog.Level
	switch config.Level {
	case LogLevelDebug:
		level = slog.LevelDebug
	case LogLevelInfo:
		level = slog.LevelInfo
	case LogLevelWarn:
		level = slog.LevelWarn
	case LogLevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Determine output
	var handler slog.Handler
	switch config.Output {
	case "stdout":
		if config.Format == LogFormatJSON {
			handler = slog.NewJSONHandler(os.Stdout, opts)
		} else {
			handler = slog.NewTextHandler(os.Stdout, opts)
		}
	case "stderr":
		if config.Format == LogFormatJSON {
			handler = slog.NewJSONHandler(os.Stderr, opts)
		} else {
			handler = slog.NewTextHandler(os.Stderr, opts)
		}
	default:
		// File output - for now, use stdout as fallback
		// In a real implementation, you'd use a file writer with rotation
		if config.Format == LogFormatJSON {
			handler = slog.NewJSONHandler(os.Stdout, opts)
		} else {
			handler = slog.NewTextHandler(os.Stdout, opts)
		}
	}

	return &StructuredLogger{
		logger: slog.New(handler),
		hooks:  make([]LogHook, 0),
		fields: make(map[string]interface{}),
	}, nil
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(msg string, args ...interface{}) {
	l.logWithFields(slog.LevelDebug, msg, args...)
}

// Info logs an info message
func (l *StructuredLogger) Info(msg string, args ...interface{}) {
	l.logWithFields(slog.LevelInfo, msg, args...)
}

// Warn logs a warning message
func (l *StructuredLogger) Warn(msg string, args ...interface{}) {
	l.logWithFields(slog.LevelWarn, msg, args...)
}

// Error logs an error message
func (l *StructuredLogger) Error(msg string, args ...interface{}) {
	l.logWithFields(slog.LevelError, msg, args...)
}

// logWithFields logs a message with the current fields
func (l *StructuredLogger) logWithFields(level slog.Level, msg string, args ...interface{}) {
	// Convert fields to slog attributes
	attrs := make([]slog.Attr, 0, len(l.fields)+len(args)/2)

	// Add existing fields
	for k, v := range l.fields {
		attrs = append(attrs, slog.Any(k, v))
	}

	// Add args as key-value pairs
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			attrs = append(attrs, slog.Any(key, args[i+1]))
		}
	}

	l.logger.LogAttrs(context.Background(), level, msg, attrs...)
}

// With creates a new logger with additional fields
func (l *StructuredLogger) With(key string, value interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &StructuredLogger{
		logger: l.logger,
		hooks:  l.hooks,
		fields: newFields,
	}
}

// WithContext creates a new logger with context
func (l *StructuredLogger) WithContext(ctx context.Context) Logger {
	// Extract common context values
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}

	// Add request ID if available
	if requestID := ctx.Value("request_id"); requestID != nil {
		newFields["request_id"] = requestID
	}

	// Add trace ID if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		newFields["trace_id"] = traceID
	}

	return &StructuredLogger{
		logger: l.logger,
		hooks:  l.hooks,
		fields: newFields,
	}
}

// LogEvent logs a structured event and triggers hooks
func (l *StructuredLogger) LogEvent(ctx context.Context, event LogEvent, fields map[string]interface{}) {
	// Merge fields
	allFields := make(map[string]interface{})
	for k, v := range l.fields {
		allFields[k] = v
	}
	for k, v := range fields {
		allFields[k] = v
	}

	// Add event metadata
	allFields["event"] = string(event)
	allFields["timestamp"] = time.Now().UTC()

	// Convert to slog attributes
	attrs := make([]slog.Attr, 0, len(allFields))
	for k, v := range allFields {
		attrs = append(attrs, slog.Any(k, v))
	}

	// Log the event
	l.logger.LogAttrs(ctx, slog.LevelInfo, fmt.Sprintf("Event: %s", event), attrs...)

	// Trigger hooks
	for _, hook := range l.hooks {
		go func(h LogHook) {
			defer func() {
				if r := recover(); r != nil {
					l.Error("Log hook panicked", "error", r, "event", event)
				}
			}()
			h(ctx, event, allFields)
		}(hook)
	}
}

// AddHook adds a log hook
func (l *StructuredLogger) AddHook(hook LogHook) {
	l.hooks = append(l.hooks, hook)
}

// RemoveHook removes a log hook
func (l *StructuredLogger) RemoveHook(hook LogHook) {
	// Note: This is a simplified implementation
	// In practice, you might want to use a more sophisticated approach
	// to identify and remove specific hooks
	for i, h := range l.hooks {
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", hook) {
			l.hooks = append(l.hooks[:i], l.hooks[i+1:]...)
			break
		}
	}
}

// Observability represents observability configuration
type Observability struct {
	Metrics MetricsConfig `yaml:"metrics" json:"metrics"`
	Tracing TracingConfig `yaml:"tracing" json:"tracing"`
	Health  HealthConfig  `yaml:"health" json:"health"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Endpoint  string `yaml:"endpoint" json:"endpoint"`
	Namespace string `yaml:"namespace" json:"namespace"`
	Interval  string `yaml:"interval" json:"interval"`
}

// TracingConfig represents tracing configuration
type TracingConfig struct {
	Enabled     bool    `yaml:"enabled" json:"enabled"`
	Endpoint    string  `yaml:"endpoint" json:"endpoint"`
	ServiceName string  `yaml:"service_name" json:"service_name"`
	SampleRate  float64 `yaml:"sample_rate" json:"sample_rate"`
}

// HealthConfig represents health check configuration
type HealthConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	Interval string `yaml:"interval" json:"interval"`
	Timeout  string `yaml:"timeout" json:"timeout"`
}

// DefaultObservability returns default observability configuration
func DefaultObservability() *Observability {
	return &Observability{
		Metrics: MetricsConfig{
			Enabled:   true,
			Endpoint:  "/metrics",
			Namespace: "swit",
			Interval:  "30s",
		},
		Tracing: TracingConfig{
			Enabled:     false,
			Endpoint:    "",
			ServiceName: "swit-service",
			SampleRate:  0.1,
		},
		Health: HealthConfig{
			Enabled:  true,
			Endpoint: "/health",
			Interval: "10s",
			Timeout:  "5s",
		},
	}
}

// Common log hooks for server events

// MetricsHook creates a hook that emits metrics for log events
func MetricsHook() LogHook {
	return func(ctx context.Context, event LogEvent, fields map[string]interface{}) {
		// In a real implementation, this would emit metrics to your metrics system
		// For now, we'll just log that a metric would be emitted
		fmt.Printf("METRICS: Event %s occurred with fields: %+v\n", event, fields)
	}
}

// TracingHook creates a hook that adds tracing information
func TracingHook() LogHook {
	return func(ctx context.Context, event LogEvent, fields map[string]interface{}) {
		// In a real implementation, this would add spans to your tracing system
		// For now, we'll just log that a trace would be created
		fmt.Printf("TRACING: Event %s traced with fields: %+v\n", event, fields)
	}
}

// AlertingHook creates a hook that sends alerts for error events
func AlertingHook() LogHook {
	return func(ctx context.Context, event LogEvent, fields map[string]interface{}) {
		// Only alert on error events
		if event == EventServerError {
			// In a real implementation, this would send alerts
			fmt.Printf("ALERT: Critical event %s occurred: %+v\n", event, fields)
		}
	}
}

// AuditHook creates a hook that logs audit events
func AuditHook() LogHook {
	return func(ctx context.Context, event LogEvent, fields map[string]interface{}) {
		// Log all events to audit log
		// In a real implementation, this would write to a separate audit log
		fmt.Printf("AUDIT: %s - %+v\n", event, fields)
	}
}
