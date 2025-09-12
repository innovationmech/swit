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

package messaging

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestMessagingAuditConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *MessagingAuditConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid config",
			config: &MessagingAuditConfig{
				Enabled:         true,
				LogLevel:        AuditLevelStandard,
				SamplingRate:    1.0,
				BufferSize:      1000,
				FlushInterval:   5 * time.Second,
				RetentionPeriod: 168 * time.Hour,
			},
			expectError: false,
		},
		{
			name: "Invalid sampling rate - too low",
			config: &MessagingAuditConfig{
				SamplingRate: -0.1,
			},
			expectError: true,
			errorMsg:    "sampling_rate must be between 0.0 and 1.0",
		},
		{
			name: "Invalid sampling rate - too high",
			config: &MessagingAuditConfig{
				SamplingRate: 1.1,
			},
			expectError: true,
			errorMsg:    "sampling_rate must be between 0.0 and 1.0",
		},
		{
			name: "Invalid buffer size",
			config: &MessagingAuditConfig{
				BufferSize: -1,
			},
			expectError: true,
			errorMsg:    "buffer_size must be positive",
		},
		{
			name: "Invalid flush interval",
			config: &MessagingAuditConfig{
				BufferSize:    1000, // Set valid default
				FlushInterval: 0,
			},
			expectError: true,
			errorMsg:    "flush_interval must be positive",
		},
		{
			name: "Invalid log level",
			config: &MessagingAuditConfig{
				LogLevel:      "invalid",
				BufferSize:    1000, // Set valid defaults
				FlushInterval: 5 * time.Second,
			},
			expectError: true,
			errorMsg:    "invalid log_level: invalid",
		},
		{
			name: "Valid minimal log level",
			config: &MessagingAuditConfig{
				LogLevel:      AuditLevelMinimal,
				BufferSize:    1000, // Set valid defaults
				FlushInterval: 5 * time.Second,
			},
			expectError: false,
		},
		{
			name: "Valid standard log level",
			config: &MessagingAuditConfig{
				LogLevel:      AuditLevelStandard,
				BufferSize:    1000, // Set valid defaults
				FlushInterval: 5 * time.Second,
			},
			expectError: false,
		},
		{
			name: "Valid comprehensive log level",
			config: &MessagingAuditConfig{
				LogLevel:      AuditLevelComprehensive,
				BufferSize:    1000, // Set valid defaults
				FlushInterval: 5 * time.Second,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMessagingAuditConfig_SetDefaults(t *testing.T) {
	config := &MessagingAuditConfig{}
	config.SetDefaults()

	assert.True(t, config.Enabled)
	assert.Equal(t, AuditLevelStandard, config.LogLevel)
	assert.Empty(t, config.Events) // Should be empty (all events)
	assert.False(t, config.IncludePayload)
	assert.True(t, config.IncludeHeaders)
	assert.True(t, config.IncludeAuthContext)
	assert.Equal(t, 168*time.Hour, config.RetentionPeriod)
	assert.Equal(t, 1.0, config.SamplingRate)
	assert.Equal(t, 1000, config.BufferSize)
	assert.Equal(t, 5*time.Second, config.FlushInterval)
}

func TestMessagingAuditor_NewAuditor(t *testing.T) {
	t.Run("With enabled config", func(t *testing.T) {
		config := &MessagingAuditConfig{Enabled: true}
		config.SetDefaults()
		auditor := NewMessagingAuditor(config)

		require.NotNil(t, auditor)
		assert.True(t, auditor.config.Enabled)
		assert.NotNil(t, auditor.metrics)
		assert.NotNil(t, auditor.eventChan)
		assert.NotNil(t, auditor.shutdownChan)

		auditor.Close()
	})

	t.Run("With disabled config", func(t *testing.T) {
		config := &MessagingAuditConfig{Enabled: false}
		auditor := NewMessagingAuditor(config)

		require.NotNil(t, auditor)
		assert.False(t, auditor.config.Enabled)

		auditor.Close()
	})

	t.Run("With nil config", func(t *testing.T) {
		auditor := NewMessagingAuditor(nil)

		require.NotNil(t, auditor)
		assert.False(t, auditor.config.Enabled)

		auditor.Close()
	})
}

func TestMessagingAuditor_LogAuditEvent(t *testing.T) {
	// Create a logger that captures logs
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	t.Run("Event logging when enabled", func(t *testing.T) {
		config := &MessagingAuditConfig{
			Enabled:       true,
			LogLevel:      AuditLevelComprehensive, // Bypass sampling
			SamplingRate:  1.0,
			BufferSize:    1, // Small buffer to force immediate flush
			FlushInterval: 10 * time.Millisecond, // Very short flush interval
		}
		config.SetDefaults()

		auditor := NewMessagingAuditor(config)
		auditor.SetLogger(logger)

		event := &AuditEvent{
			Type:      AuditEventMessageCreated,
			MessageID: "test-message-id",
			Topic:     "test-topic",
			Operation: "create_message",
			Success:   true,
		}

		// Test synchronous logging (buffer full fallback)
		auditor.LogAuditEvent(context.Background(), event)
		// Log another event to fill the buffer and trigger synchronous logging
		auditor.LogAuditEvent(context.Background(), event)

		// Force flush by closing the auditor
		auditor.Close()

		// Check that events were logged
		assert.Greater(t, logs.Len(), 0)
		logEntry := logs.All()[0]
		assert.Equal(t, "Messaging audit event", logEntry.Message)
		assert.Equal(t, zapcore.InfoLevel, logEntry.Level)
	})

	t.Run("Event filtering by type", func(t *testing.T) {
		config := &MessagingAuditConfig{
			Enabled:  true,
			LogLevel: AuditLevelStandard,
			Events:   []AuditEventType{AuditEventMessagePublished},
		}
		config.SetDefaults()

		auditor := NewMessagingAuditor(config)
		auditor.SetLogger(logger)

		// Log an event that's not in the filter
		event := &AuditEvent{
			Type:      AuditEventMessageCreated,
			MessageID: "test-message-id",
			Operation: "create_message",
			Success:   true,
		}

		auditor.LogAuditEvent(context.Background(), event)

		// Wait for async processing
		time.Sleep(100 * time.Millisecond)

		// Should not be logged
		logCount := logs.Len()

		// Log an event that's in the filter
		event.Type = AuditEventMessagePublished
		auditor.LogAuditEvent(context.Background(), event)

		// Wait for async processing
		time.Sleep(100 * time.Millisecond)

		// Should be logged
		assert.Equal(t, logCount+1, logs.Len())

		auditor.Close()
	})

	t.Run("Event logging when disabled", func(t *testing.T) {
		config := &MessagingAuditConfig{Enabled: false}
		config.SetDefaults()
		auditor := NewMessagingAuditor(config)
		auditor.SetLogger(logger)

		event := &AuditEvent{
			Type:      AuditEventMessageCreated,
			MessageID: "test-message-id",
			Operation: "create_message",
			Success:   true,
		}

		auditor.LogAuditEvent(context.Background(), event)

		// Should not be logged
		assert.Equal(t, 0, logs.Len())

		auditor.Close()
	})
}

func TestMessagingAuditor_Sampling(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	t.Run("Minimal level sampling", func(t *testing.T) {
		config := &MessagingAuditConfig{
			Enabled:       true,
			LogLevel:      AuditLevelMinimal,
			SamplingRate:  0.1,
			BufferSize:    10,
			FlushInterval: 100 * time.Millisecond,
		}

		auditor := NewMessagingAuditor(config)
		auditor.SetLogger(logger)

		// Non-critical event should be sampled out
		event := &AuditEvent{
			Type:      AuditEventMessageCreated,
			MessageID: "test-message-id",
			Operation: "create_message",
			Success:   true,
		}

		auditor.LogAuditEvent(context.Background(), event)
		time.Sleep(100 * time.Millisecond)

		// Should not be logged
		logCount := logs.Len()

		// Critical event should not be sampled out
		event.Type = AuditEventMessageFailed
		auditor.LogAuditEvent(context.Background(), event)
		time.Sleep(100 * time.Millisecond)

		// Should be logged
		assert.Equal(t, logCount+1, logs.Len())

		auditor.Close()
	})

	t.Run("Standard level with sampling", func(t *testing.T) {
		config := &MessagingAuditConfig{
			Enabled:       true,
			LogLevel:      AuditLevelStandard,
			SamplingRate:  0.0, // No sampling
			BufferSize:    10,
			FlushInterval: 100 * time.Millisecond,
		}

		auditor := NewMessagingAuditor(config)
		auditor.SetLogger(logger)

		event := &AuditEvent{
			Type:      AuditEventMessageCreated,
			MessageID: "test-message-id",
			Operation: "create_message",
			Success:   true,
		}

		auditor.LogAuditEvent(context.Background(), event)
		time.Sleep(100 * time.Millisecond)

		// Should be logged
		assert.Greater(t, logs.Len(), 0)

		auditor.Close()
	})
}

func TestMessagingAuditor_ContextExtraction(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	config := &MessagingAuditConfig{
		Enabled:            true,
		LogLevel:           AuditLevelStandard,
		IncludeAuthContext: true,
		SamplingRate:       1.0,
		BufferSize:         10,
		FlushInterval:      100 * time.Millisecond,
	}

	auditor := NewMessagingAuditor(config)
	auditor.SetLogger(logger)

	// Create context with correlation ID and user ID
	ctx := context.WithValue(context.Background(), "correlation_id", "test-correlation-id")
	ctx = context.WithValue(ctx, "user_id", "test-user-id")

	// Create context with auth context
	authCtx := &AuthContext{
		Credentials: &AuthCredentials{
			UserID: "auth-user-id",
			Scopes: []string{"read", "write"},
		},
		Timestamp: time.Now(),
		Source:    "test",
	}
	ctx = context.WithValue(ctx, "auth_context", authCtx)

	event := &AuditEvent{
		Type:      AuditEventMessageCreated,
		MessageID: "test-message-id",
		Operation: "create_message",
		Success:   true,
	}

	auditor.LogAuditEvent(ctx, event)
	time.Sleep(200 * time.Millisecond)

	// Check that context information was extracted
	assert.Greater(t, logs.Len(), 0)
	logEntry := logs.All()[0]

	// Check for correlation ID and user ID in log fields
	var foundCorrelationID, foundUserID bool
	for _, field := range logEntry.Context {
		if field.Key == "correlation_id" && field.String == "test-correlation-id" {
			foundCorrelationID = true
		}
		if field.Key == "user_id" && field.String == "test-user-id" {
			foundUserID = true
		}
		if field.Key == "auth_user_id" && field.String == "auth-user-id" {
			foundUserID = true
		}
	}

	assert.True(t, foundCorrelationID, "Correlation ID should be extracted from context")
	assert.True(t, foundUserID, "User ID should be extracted from context")

	auditor.Close()
}

func TestMessagingAuditor_Metrics(t *testing.T) {
	config := &MessagingAuditConfig{
		Enabled:       true,
		LogLevel:      AuditLevelStandard,
		SamplingRate:  1.0,
		BufferSize:    10,
		FlushInterval: 100 * time.Millisecond,
	}

	auditor := NewMessagingAuditor(config)

	// Log some events
	auditor.LogMessageCreated(context.Background(), "msg1", "topic1", 1024)
	auditor.LogMessagePublished(context.Background(), "msg2", "topic2", BrokerTypeKafka, 10.5, true, nil)
	auditor.LogMessageFailed(context.Background(), "msg3", "topic3", 15.2, assert.AnError)

	time.Sleep(200 * time.Millisecond)

	metrics := auditor.GetMetrics()

	assert.Equal(t, int64(3), metrics.TotalEvents)
	assert.Equal(t, int64(2), metrics.SuccessfulEvents)
	assert.Equal(t, int64(1), metrics.FailedEvents)
	assert.Equal(t, int64(1), metrics.EventsByType[AuditEventMessageCreated])
	assert.Equal(t, int64(1), metrics.EventsByType[AuditEventMessagePublished])
	assert.Equal(t, int64(1), metrics.EventsByType[AuditEventMessageFailed])
	assert.Greater(t, metrics.AverageProcessingTime, 0.0)

	auditor.Close()
}

func TestMessagingAuditor_ConvenienceMethods(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	config := &MessagingAuditConfig{
		Enabled:       true,
		LogLevel:      AuditLevelStandard,
		SamplingRate:  1.0,
		BufferSize:    10,
		FlushInterval: 100 * time.Millisecond,
	}

	auditor := NewMessagingAuditor(config)
	auditor.SetLogger(logger)

	// Test convenience methods
	auditor.LogMessageCreated(context.Background(), "msg1", "topic1", 1024)
	auditor.LogMessagePublished(context.Background(), "msg2", "topic2", BrokerTypeKafka, 10.5, true, nil)
	auditor.LogMessageReceived(context.Background(), "msg3", "topic3", BrokerTypeNATS, 2048)
	auditor.LogMessageProcessed(context.Background(), "msg4", "topic4", 5.2, true, nil)
	auditor.LogMessageFailed(context.Background(), "msg5", "topic5", 2.1, assert.AnError)
	auditor.LogBrokerConnected(context.Background(), BrokerTypeRabbitMQ, true, nil)
	auditor.LogBrokerDisconnected(context.Background(), BrokerTypeRabbitMQ)
	auditor.LogSubscriptionStarted(context.Background(), "topic6", "group1")
	auditor.LogSubscriptionStopped(context.Background(), "topic7", "group2")
	auditor.LogAuthSuccess(context.Background(), "user1", "jwt")
	auditor.LogAuthFailure(context.Background(), "user2", "apikey", assert.AnError)
	auditor.LogPolicyViolation(context.Background(), "policy1", "msg6", "invalid-format")

	time.Sleep(200 * time.Millisecond)

	// Check that all events were logged
	assert.GreaterOrEqual(t, logs.Len(), 11)

	auditor.Close()
}

func TestMessagingAuditor_GracefulShutdown(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	config := &MessagingAuditConfig{
		Enabled:       true,
		LogLevel:      AuditLevelStandard,
		SamplingRate:  1.0,
		BufferSize:    100,
		FlushInterval: 1 * time.Second,
	}

	auditor := NewMessagingAuditor(config)
	auditor.SetLogger(logger)

	// Log some events
	for i := 0; i < 10; i++ {
		auditor.LogMessageCreated(context.Background(),
			fmt.Sprintf("msg%d", i), "topic", int64(1024+i))
	}

	// Immediately close - should flush remaining events
	err := auditor.Close()
	require.NoError(t, err)

	// All events should be logged despite the long flush interval
	assert.GreaterOrEqual(t, logs.Len(), 10)
}

func TestBrokerConfig_WithAudit(t *testing.T) {
	t.Run("Valid broker config with audit", func(t *testing.T) {
		config := &BrokerConfig{
			Type:      BrokerTypeKafka,
			Endpoints: []string{"localhost:9092"},
			Audit: &MessagingAuditConfig{
				Enabled:       true,
				LogLevel:      AuditLevelStandard,
				SamplingRate:  1.0,
				BufferSize:    1000,
				FlushInterval: 5 * time.Second,
			},
		}

		config.SetDefaults()
		err := config.Validate()
		require.NoError(t, err)
		assert.NotNil(t, config.Audit)
		assert.True(t, config.Audit.Enabled)
	})

	t.Run("Invalid broker config with invalid audit", func(t *testing.T) {
		config := &BrokerConfig{
			Type:      BrokerTypeKafka,
			Endpoints: []string{"localhost:9092"},
			Audit:     &MessagingAuditConfig{}, // Initialize audit config
		}
		config.SetDefaults() // Set defaults for connection and other fields
		// Set invalid audit config after defaults
		config.Audit.SamplingRate = 1.1 // Invalid

		err := config.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "audit config validation failed")
	})
}
