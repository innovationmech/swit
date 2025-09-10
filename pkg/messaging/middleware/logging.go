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

package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// LogLevel defines the available logging levels.
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// LoggerAdapter defines the interface for different logging backends.
type LoggerAdapter interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// SlogAdapter wraps Go's standard slog logger to implement LoggerAdapter.
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a new slog-based logger adapter.
func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}
	return &SlogAdapter{logger: logger}
}

// Debug logs a debug message with structured fields.
func (s *SlogAdapter) Debug(msg string, fields map[string]interface{}) {
	attrs := make([]slog.Attr, 0, len(fields))
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	s.logger.LogAttrs(context.Background(), slog.LevelDebug, msg, attrs...)
}

// Info logs an info message with structured fields.
func (s *SlogAdapter) Info(msg string, fields map[string]interface{}) {
	attrs := make([]slog.Attr, 0, len(fields))
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	s.logger.LogAttrs(context.Background(), slog.LevelInfo, msg, attrs...)
}

// Warn logs a warning message with structured fields.
func (s *SlogAdapter) Warn(msg string, fields map[string]interface{}) {
	attrs := make([]slog.Attr, 0, len(fields))
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	s.logger.LogAttrs(context.Background(), slog.LevelWarn, msg, attrs...)
}

// Error logs an error message with structured fields.
func (s *SlogAdapter) Error(msg string, fields map[string]interface{}) {
	attrs := make([]slog.Attr, 0, len(fields))
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	s.logger.LogAttrs(context.Background(), slog.LevelError, msg, attrs...)
}

// LoggingConfig holds configuration for the logging middleware.
type LoggingConfig struct {
	Level                 LogLevel
	Logger                LoggerAdapter
	LogMessageContent     bool
	LogHeaders            bool
	LogProcessingTime     bool
	LogCorrelationID      bool
	MaxMessageSize        int
	SanitizeHeaders       []string
	IncludeStackTrace     bool
	LogSuccessfulMessages bool
}

// DefaultLoggingConfig returns a default logging configuration.
func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Level:                 LogLevelInfo,
		Logger:                NewSlogAdapter(nil),
		LogMessageContent:     false, // Don't log message content by default for security
		LogHeaders:            true,
		LogProcessingTime:     true,
		LogCorrelationID:      true,
		MaxMessageSize:        1024, // Log up to 1KB of message content
		SanitizeHeaders:       []string{"authorization", "x-api-key", "x-auth-token"},
		IncludeStackTrace:     false,
		LogSuccessfulMessages: true,
	}
}

// StructuredLoggingMiddleware provides comprehensive structured logging for message processing.
type StructuredLoggingMiddleware struct {
	config *LoggingConfig
}

// NewStructuredLoggingMiddleware creates a new structured logging middleware with the given configuration.
func NewStructuredLoggingMiddleware(config *LoggingConfig) *StructuredLoggingMiddleware {
	if config == nil {
		config = DefaultLoggingConfig()
	}
	if config.Logger == nil {
		config.Logger = NewSlogAdapter(nil)
	}
	return &StructuredLoggingMiddleware{
		config: config,
	}
}

// Name returns the middleware name.
func (slm *StructuredLoggingMiddleware) Name() string {
	return "structured-logging"
}

// Wrap wraps a handler with structured logging functionality.
func (slm *StructuredLoggingMiddleware) Wrap(next messaging.MessageHandler) messaging.MessageHandler {
	return messaging.MessageHandlerFunc(func(ctx context.Context, message *messaging.Message) error {
		start := time.Now()

		// Build base fields
		fields := slm.buildBaseFields(message)

		// Log incoming message
		if slm.config.Level <= LogLevelDebug {
			slm.config.Logger.Debug("processing message", fields)
		}

		err := next.Handle(ctx, message)

		processingTime := time.Since(start)
		fields["processing_time_ms"] = processingTime.Milliseconds()

		if err != nil {
			// Enhance fields with error information
			errorFields := slm.buildErrorFields(fields, err)

			if slm.config.Level <= LogLevelError {
				slm.config.Logger.Error("message processing failed", errorFields)
			}
		} else if slm.config.LogSuccessfulMessages {
			if slm.config.Level <= LogLevelInfo {
				slm.config.Logger.Info("message processed successfully", fields)
			}
		}

		return err
	})
}

// buildBaseFields constructs the base structured log fields for a message.
func (slm *StructuredLoggingMiddleware) buildBaseFields(message *messaging.Message) map[string]interface{} {
	fields := map[string]interface{}{
		"message_id": message.ID,
		"topic":      message.Topic,
		"timestamp":  message.Timestamp.Format(time.RFC3339Nano),
	}

	// Add correlation ID if configured and present
	if slm.config.LogCorrelationID && message.CorrelationID != "" {
		fields["correlation_id"] = message.CorrelationID
	}

	// Add headers if configured
	if slm.config.LogHeaders && len(message.Headers) > 0 {
		sanitizedHeaders := slm.sanitizeHeaders(message.Headers)
		if len(sanitizedHeaders) > 0 {
			fields["headers"] = sanitizedHeaders
		}
	}

	// Add message content if configured
	if slm.config.LogMessageContent && len(message.Payload) > 0 {
		content := slm.extractMessageContent(message.Payload)
		if content != "" {
			fields["message_content"] = content
		}
	}

	// Add routing information if present
	if len(message.Key) > 0 {
		fields["partition_key"] = string(message.Key)
	}

	// Add retry information if present
	if message.DeliveryAttempt > 0 {
		fields["retry_count"] = message.DeliveryAttempt
	}

	return fields
}

// buildErrorFields enhances log fields with error-specific information.
func (slm *StructuredLoggingMiddleware) buildErrorFields(baseFields map[string]interface{}, err error) map[string]interface{} {
	errorFields := make(map[string]interface{})
	for k, v := range baseFields {
		errorFields[k] = v
	}

	errorFields["error"] = err.Error()
	errorFields["error_type"] = fmt.Sprintf("%T", err)

	// Add messaging-specific error information
	if msgErr, ok := err.(*messaging.MessagingError); ok {
		errorFields["error_code"] = string(msgErr.Code)
		errorFields["is_retryable"] = msgErr.Retryable
		if msgErr.Details != nil {
			if retryAfter, ok := msgErr.Details["retry_after"]; ok {
				errorFields["retry_after"] = retryAfter
			}
		}
	}

	return errorFields
}

// sanitizeHeaders removes sensitive header values while preserving structure.
func (slm *StructuredLoggingMiddleware) sanitizeHeaders(headers map[string]string) map[string]string {
	if len(slm.config.SanitizeHeaders) == 0 {
		return headers
	}

	sanitized := make(map[string]string)
	for k, v := range headers {
		if slm.isSensitiveHeader(k) {
			sanitized[k] = "[REDACTED]"
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}

// isSensitiveHeader checks if a header should be sanitized.
func (slm *StructuredLoggingMiddleware) isSensitiveHeader(headerName string) bool {
	lowercaseHeader := toLower(headerName)
	for _, sensitiveHeader := range slm.config.SanitizeHeaders {
		if toLower(sensitiveHeader) == lowercaseHeader {
			return true
		}
	}
	return false
}

// extractMessageContent safely extracts message content for logging.
func (slm *StructuredLoggingMiddleware) extractMessageContent(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}

	maxSize := slm.config.MaxMessageSize
	if maxSize <= 0 {
		maxSize = 1024 // Default to 1KB
	}

	content := string(payload)
	if len(content) > maxSize {
		content = content[:maxSize] + "...[TRUNCATED]"
	}

	// Try to format as JSON for better readability
	if slm.isValidJSON(payload) {
		var jsonData interface{}
		if json.Unmarshal(payload, &jsonData) == nil {
			if formatted, err := json.MarshalIndent(jsonData, "", "  "); err == nil {
				formattedStr := string(formatted)
				if len(formattedStr) <= maxSize {
					return formattedStr
				}
			}
		}
	}

	return content
}

// isValidJSON checks if the payload is valid JSON.
func (slm *StructuredLoggingMiddleware) isValidJSON(payload []byte) bool {
	var js json.RawMessage
	return json.Unmarshal(payload, &js) == nil
}

// toLower converts a string to lowercase (simple implementation).
func toLower(s string) string {
	result := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' {
			result[i] = b + ('a' - 'A')
		} else {
			result[i] = b
		}
	}
	return string(result)
}

// StandardLoggerAdapter wraps Go's standard log package to implement LoggerAdapter.
type StandardLoggerAdapter struct {
	logger *log.Logger
}

// NewStandardLoggerAdapter creates a new standard logger adapter.
func NewStandardLoggerAdapter(logger *log.Logger) *StandardLoggerAdapter {
	if logger == nil {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	}
	return &StandardLoggerAdapter{logger: logger}
}

// Debug logs a debug message with structured fields using standard logger.
func (s *StandardLoggerAdapter) Debug(msg string, fields map[string]interface{}) {
	s.logWithFields("DEBUG", msg, fields)
}

// Info logs an info message with structured fields using standard logger.
func (s *StandardLoggerAdapter) Info(msg string, fields map[string]interface{}) {
	s.logWithFields("INFO", msg, fields)
}

// Warn logs a warning message with structured fields using standard logger.
func (s *StandardLoggerAdapter) Warn(msg string, fields map[string]interface{}) {
	s.logWithFields("WARN", msg, fields)
}

// Error logs an error message with structured fields using standard logger.
func (s *StandardLoggerAdapter) Error(msg string, fields map[string]interface{}) {
	s.logWithFields("ERROR", msg, fields)
}

// logWithFields formats and logs a message with structured fields.
func (s *StandardLoggerAdapter) logWithFields(level, msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		s.logger.Printf("[%s] %s", level, msg)
		return
	}

	fieldsJSON, _ := json.Marshal(fields)
	s.logger.Printf("[%s] %s | %s", level, msg, string(fieldsJSON))
}

// CreateLoggingMiddleware is a factory function to create logging middleware from configuration.
func CreateLoggingMiddleware(config map[string]interface{}) (messaging.Middleware, error) {
	loggingConfig := DefaultLoggingConfig()

	// Configure log level
	if level, ok := config["level"]; ok {
		if levelStr, ok := level.(string); ok {
			switch levelStr {
			case "debug":
				loggingConfig.Level = LogLevelDebug
			case "info":
				loggingConfig.Level = LogLevelInfo
			case "warn":
				loggingConfig.Level = LogLevelWarn
			case "error":
				loggingConfig.Level = LogLevelError
			}
		}
	}

	// Configure logger type
	if loggerType, ok := config["logger_type"]; ok {
		if loggerTypeStr, ok := loggerType.(string); ok {
			switch loggerTypeStr {
			case "slog":
				loggingConfig.Logger = NewSlogAdapter(nil)
			case "standard":
				loggingConfig.Logger = NewStandardLoggerAdapter(nil)
			default:
				loggingConfig.Logger = NewSlogAdapter(nil)
			}
		}
	}

	// Configure other options
	if logContent, ok := config["log_message_content"]; ok {
		if logContentBool, ok := logContent.(bool); ok {
			loggingConfig.LogMessageContent = logContentBool
		}
	}

	if logHeaders, ok := config["log_headers"]; ok {
		if logHeadersBool, ok := logHeaders.(bool); ok {
			loggingConfig.LogHeaders = logHeadersBool
		}
	}

	if maxSize, ok := config["max_message_size"]; ok {
		if maxSizeInt, ok := maxSize.(int); ok {
			loggingConfig.MaxMessageSize = maxSizeInt
		}
	}

	return NewStructuredLoggingMiddleware(loggingConfig), nil
}
