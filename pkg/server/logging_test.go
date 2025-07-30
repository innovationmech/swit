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
	"testing"
	"time"
)

func TestLogConfig(t *testing.T) {
	t.Run("DefaultLogConfig", func(t *testing.T) {
		config := DefaultLogConfig()
		if config.Level != LogLevelInfo {
			t.Errorf("Expected default level to be %s, got %s", LogLevelInfo, config.Level)
		}
		if config.Format != LogFormatJSON {
			t.Errorf("Expected default format to be %s, got %s", LogFormatJSON, config.Format)
		}
		if config.Output != "stdout" {
			t.Errorf("Expected default output to be stdout, got %s", config.Output)
		}
	})
}

func TestStructuredLogger(t *testing.T) {
	t.Run("NewStructuredLogger", func(t *testing.T) {
		logger, err := NewStructuredLogger(nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if logger == nil {
			t.Errorf("Expected logger to be created")
		}
	})

	t.Run("NewStructuredLogger with config", func(t *testing.T) {
		config := &LogConfig{
			Level:  LogLevelDebug,
			Format: LogFormatText,
			Output: "stderr",
		}
		logger, err := NewStructuredLogger(config)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if logger == nil {
			t.Errorf("Expected logger to be created")
		}
	})

	t.Run("Logger methods", func(t *testing.T) {
		logger, _ := NewStructuredLogger(nil)

		// Test that these don't panic
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")

		// Test with key-value pairs
		logger.Info("message with fields", "key1", "value1", "key2", 42)
	})

	t.Run("With method", func(t *testing.T) {
		logger, _ := NewStructuredLogger(nil)
		newLogger := logger.With("service", "test-service")

		if newLogger == logger {
			t.Errorf("Expected With to return a new logger instance")
		}

		// Test that the new logger has the field
		structLogger, ok := newLogger.(*StructuredLogger)
		if !ok {
			t.Errorf("Expected newLogger to be *StructuredLogger")
			return
		}
		if structLogger.fields["service"] != "test-service" {
			t.Errorf("Expected field to be set in new logger")
		}

		// Test that original logger is unchanged
		if _, exists := logger.fields["service"]; exists {
			t.Errorf("Expected original logger to be unchanged")
		}
	})

	t.Run("WithContext method", func(t *testing.T) {
		logger, _ := NewStructuredLogger(nil)
		ctx := context.WithValue(context.Background(), "request_id", "req-123")
		ctx = context.WithValue(ctx, "trace_id", "trace-456")

		newLogger := logger.WithContext(ctx)
		structLogger := newLogger.(*StructuredLogger)

		if structLogger.fields["request_id"] != "req-123" {
			t.Errorf("Expected request_id to be extracted from context")
		}
		if structLogger.fields["trace_id"] != "trace-456" {
			t.Errorf("Expected trace_id to be extracted from context")
		}
	})
}

func TestLogHooks(t *testing.T) {
	t.Run("AddHook and LogEvent", func(t *testing.T) {
		logger, _ := NewStructuredLogger(nil)

		hookCalled := false
		var capturedEvent LogEvent
		var capturedFields map[string]interface{}

		hook := func(ctx context.Context, event LogEvent, fields map[string]interface{}) {
			hookCalled = true
			capturedEvent = event
			capturedFields = fields
		}

		logger.AddHook(hook)

		fields := map[string]interface{}{
			"component": "test",
			"status":    "success",
		}

		logger.LogEvent(context.Background(), EventServerStartup, fields)

		// Give the goroutine time to execute
		time.Sleep(100 * time.Millisecond)

		if !hookCalled {
			t.Errorf("Expected hook to be called")
		}
		if capturedEvent != EventServerStartup {
			t.Errorf("Expected event %s, got %s", EventServerStartup, capturedEvent)
		}
		if capturedFields["component"] != "test" {
			t.Errorf("Expected component field to be passed to hook")
		}
		if capturedFields["event"] != string(EventServerStartup) {
			t.Errorf("Expected event field to be added")
		}
		if _, exists := capturedFields["timestamp"]; !exists {
			t.Errorf("Expected timestamp field to be added")
		}
	})

	t.Run("RemoveHook", func(t *testing.T) {
		logger, _ := NewStructuredLogger(nil)

		hookCalled := false
		hook := func(ctx context.Context, event LogEvent, fields map[string]interface{}) {
			hookCalled = true
		}

		logger.AddHook(hook)
		logger.RemoveHook(hook)

		logger.LogEvent(context.Background(), EventServerStartup, map[string]interface{}{})

		// Give time for potential hook execution
		time.Sleep(100 * time.Millisecond)

		if hookCalled {
			t.Errorf("Expected hook to be removed and not called")
		}
	})

	t.Run("Multiple hooks", func(t *testing.T) {
		logger, _ := NewStructuredLogger(nil)

		hook1Called := false
		hook2Called := false

		hook1 := func(ctx context.Context, event LogEvent, fields map[string]interface{}) {
			hook1Called = true
		}
		hook2 := func(ctx context.Context, event LogEvent, fields map[string]interface{}) {
			hook2Called = true
		}

		logger.AddHook(hook1)
		logger.AddHook(hook2)

		logger.LogEvent(context.Background(), EventServerStartup, map[string]interface{}{})

		// Give time for hook execution
		time.Sleep(100 * time.Millisecond)

		if !hook1Called {
			t.Errorf("Expected hook1 to be called")
		}
		if !hook2Called {
			t.Errorf("Expected hook2 to be called")
		}
	})
}

func TestObservability(t *testing.T) {
	t.Run("DefaultObservability", func(t *testing.T) {
		obs := DefaultObservability()

		if !obs.Metrics.Enabled {
			t.Errorf("Expected metrics to be enabled by default")
		}
		if obs.Metrics.Endpoint != "/metrics" {
			t.Errorf("Expected default metrics endpoint to be /metrics")
		}
		if obs.Metrics.Namespace != "swit" {
			t.Errorf("Expected default namespace to be swit")
		}

		if obs.Tracing.Enabled {
			t.Errorf("Expected tracing to be disabled by default")
		}
		if obs.Tracing.ServiceName != "swit-service" {
			t.Errorf("Expected default service name to be swit-service")
		}

		if !obs.Health.Enabled {
			t.Errorf("Expected health checks to be enabled by default")
		}
		if obs.Health.Endpoint != "/health" {
			t.Errorf("Expected default health endpoint to be /health")
		}
	})
}

func TestPredefinedHooks(t *testing.T) {
	t.Run("MetricsHook", func(t *testing.T) {
		hook := MetricsHook()
		if hook == nil {
			t.Errorf("Expected MetricsHook to return a function")
		}

		// Test that it doesn't panic
		hook(context.Background(), EventServerStartup, map[string]interface{}{"test": "value"})
	})

	t.Run("TracingHook", func(t *testing.T) {
		hook := TracingHook()
		if hook == nil {
			t.Errorf("Expected TracingHook to return a function")
		}

		// Test that it doesn't panic
		hook(context.Background(), EventServerStartup, map[string]interface{}{"test": "value"})
	})

	t.Run("AlertingHook", func(t *testing.T) {
		hook := AlertingHook()
		if hook == nil {
			t.Errorf("Expected AlertingHook to return a function")
		}

		// Test that it doesn't panic
		hook(context.Background(), EventServerError, map[string]interface{}{"error": "test error"})
		hook(context.Background(), EventServerStartup, map[string]interface{}{"test": "value"})
	})

	t.Run("AuditHook", func(t *testing.T) {
		hook := AuditHook()
		if hook == nil {
			t.Errorf("Expected AuditHook to return a function")
		}

		// Test that it doesn't panic
		hook(context.Background(), EventServerStartup, map[string]interface{}{"test": "value"})
	})
}

func TestLogEvents(t *testing.T) {
	t.Run("All log events are defined", func(t *testing.T) {
		events := []LogEvent{
			EventServerStartup,
			EventServerShutdown,
			EventServerError,
			EventTransportStart,
			EventTransportStop,
			EventServiceRegister,
			EventDiscoveryConnect,
			EventHealthCheck,
			EventConfigLoad,
			EventDependencyInit,
		}

		for _, event := range events {
			if string(event) == "" {
				t.Errorf("Event should not be empty")
			}
		}
	})
}

func TestLogLevelsAndFormats(t *testing.T) {
	t.Run("Log levels", func(t *testing.T) {
		levels := []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}
		for _, level := range levels {
			if string(level) == "" {
				t.Errorf("Log level should not be empty")
			}
		}
	})

	t.Run("Log formats", func(t *testing.T) {
		formats := []LogFormat{LogFormatJSON, LogFormatText}
		for _, format := range formats {
			if string(format) == "" {
				t.Errorf("Log format should not be empty")
			}
		}
	})
}
