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

package messaging

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TestStructMessageValidator tests the message validation functionality.
func TestStructMessageValidator(t *testing.T) {
	validator := NewStructMessageValidator()

	tests := []struct {
		name       string
		message    *ValidatedMessage
		wantError  bool
		errorField string
	}{
		{
			name: "valid message",
			message: &ValidatedMessage{
				ID:        uuid.New().String(),
				Topic:     "test.topic",
				Payload:   []byte("test payload"),
				Timestamp: time.Now(),
			},
			wantError: false,
		},
		{
			name: "missing ID",
			message: &ValidatedMessage{
				Topic:     "test.topic",
				Payload:   []byte("test payload"),
				Timestamp: time.Now(),
			},
			wantError:  true,
			errorField: "ID",
		},
		{
			name: "missing topic",
			message: &ValidatedMessage{
				ID:        uuid.New().String(),
				Payload:   []byte("test payload"),
				Timestamp: time.Now(),
			},
			wantError:  true,
			errorField: "Topic",
		},
		{
			name: "missing payload",
			message: &ValidatedMessage{
				ID:        uuid.New().String(),
				Topic:     "test.topic",
				Timestamp: time.Now(),
			},
			wantError:  true,
			errorField: "Payload",
		},
		{
			name: "invalid topic name",
			message: &ValidatedMessage{
				ID:        uuid.New().String(),
				Topic:     ".invalid-topic",
				Payload:   []byte("test payload"),
				Timestamp: time.Now(),
			},
			wantError:  true,
			errorField: "Topic",
		},
		{
			name: "invalid priority",
			message: &ValidatedMessage{
				ID:        uuid.New().String(),
				Topic:     "test.topic",
				Payload:   []byte("test payload"),
				Timestamp: time.Now(),
				Priority:  15, // Invalid: should be 0-9
			},
			wantError:  true,
			errorField: "Priority",
		},
		{
			name: "future timestamp",
			message: &ValidatedMessage{
				ID:        uuid.New().String(),
				Topic:     "test.topic",
				Payload:   []byte("test payload"),
				Timestamp: time.Now().Add(10 * time.Minute), // Too far in future
			},
			wantError:  true,
			errorField: "timestamp",
		},
		{
			name: "expired TTL",
			message: &ValidatedMessage{
				ID:        uuid.New().String(),
				Topic:     "test.topic",
				Payload:   []byte("test payload"),
				Timestamp: time.Now().Add(-2 * time.Hour), // Old timestamp
				TTL:       3600,                           // 1 hour TTL, already expired
			},
			wantError:  true,
			errorField: "ttl",
		},
		{
			name: "correlation ID mismatch",
			message: &ValidatedMessage{
				ID:            uuid.New().String(),
				Topic:         "test.topic",
				Payload:       []byte("test payload"),
				Timestamp:     time.Now(),
				CorrelationID: "corr-123",
				Headers: MessageHeaders{
					CorrelationID: "corr-456", // Different from message level
				},
			},
			wantError:  true,
			errorField: "correlation_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.message)

			if tt.wantError {
				if err == nil {
					t.Error("expected validation error, got nil")
					return
				}

				if validErr, ok := err.(*MessageValidationError); ok {
					if tt.errorField != "" && validErr.Field != tt.errorField {
						t.Errorf("expected error field %s, got %s", tt.errorField, validErr.Field)
					}
				} else {
					t.Errorf("expected MessageValidationError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected validation error: %v", err)
				}
			}
		})
	}
}

// TestValidateTopicName tests the topic name validation function.
func TestValidateTopicName(t *testing.T) {
	validator := NewStructMessageValidator()

	validTopics := []string{
		"topic",
		"test.topic",
		"user-events",
		"system_events",
		"order.payment.completed",
		"service-1.events",
		"a",
		"123",
	}

	for _, topic := range validTopics {
		t.Run("valid_"+topic, func(t *testing.T) {
			msg := &ValidatedMessage{
				ID:        uuid.New().String(),
				Topic:     topic,
				Payload:   []byte("test"),
				Timestamp: time.Now(),
			}

			if err := validator.Validate(msg); err != nil {
				t.Errorf("topic %s should be valid, got error: %v", topic, err)
			}
		})
	}

	invalidTopics := []string{
		"",
		".topic",
		"topic.",
		"topic..name",
		"topic-",
		"-topic",
		"topic_",
		"_topic",
	}

	for _, topic := range invalidTopics {
		t.Run("invalid_"+topic, func(t *testing.T) {
			msg := &ValidatedMessage{
				ID:        uuid.New().String(),
				Topic:     topic,
				Payload:   []byte("test"),
				Timestamp: time.Now(),
			}

			if err := validator.Validate(msg); err == nil {
				t.Errorf("topic %s should be invalid", topic)
			}
		})
	}
}

// TestMessageBuilder tests the message builder functionality.
func TestMessageBuilder(t *testing.T) {
	t.Run("build valid message", func(t *testing.T) {
		msg, err := NewMessageBuilder().
			WithID(uuid.New().String()).
			WithTopic("test.topic").
			WithPayload([]byte("test payload")).
			WithCorrelationID("corr-123").
			WithPriority(5).
			WithTTL(3600).
			WithHeader("custom-header", "custom-value").
			Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if msg.Topic != "test.topic" {
			t.Errorf("expected topic 'test.topic', got %s", msg.Topic)
		}

		if msg.CorrelationID != "corr-123" {
			t.Errorf("expected correlation ID 'corr-123', got %s", msg.CorrelationID)
		}

		if msg.Headers.CorrelationID != "corr-123" {
			t.Errorf("expected header correlation ID 'corr-123', got %s", msg.Headers.CorrelationID)
		}

		if msg.Priority != 5 {
			t.Errorf("expected priority 5, got %d", msg.Priority)
		}

		if msg.TTL != 3600 {
			t.Errorf("expected TTL 3600, got %d", msg.TTL)
		}

		if msg.Headers.Custom["custom-header"] != "custom-value" {
			t.Errorf("expected custom header 'custom-value', got %s", msg.Headers.Custom["custom-header"])
		}

		if msg.Headers.ExpiresAt == nil {
			t.Error("expected ExpiresAt to be set when TTL is provided")
		}
	})

	t.Run("build with trace context", func(t *testing.T) {
		// Create a mock trace context
		traceID := trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
		spanID := trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8}

		spanContext := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		})

		ctx := trace.ContextWithSpanContext(context.Background(), spanContext)

		msg, err := NewMessageBuilder().
			WithID(uuid.New().String()).
			WithTopic("test.topic").
			WithPayload([]byte("test payload")).
			WithTraceContext(ctx).
			Build()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if msg.TraceContext.TraceID != traceID {
			t.Errorf("expected trace ID %v, got %v", traceID, msg.TraceContext.TraceID)
		}

		if msg.TraceContext.SpanID != spanID {
			t.Errorf("expected span ID %v, got %v", spanID, msg.TraceContext.SpanID)
		}

		if msg.Headers.TraceID != traceID.String() {
			t.Errorf("expected header trace ID %s, got %s", traceID.String(), msg.Headers.TraceID)
		}
	})

	t.Run("build invalid message", func(t *testing.T) {
		// Missing required fields
		_, err := NewMessageBuilder().
			WithTopic("test.topic").
			Build()

		if err == nil {
			t.Error("expected validation error for missing ID")
		}

		if validErr, ok := err.(*MessageValidationError); ok {
			if validErr.Field != "ID" {
				t.Errorf("expected error field 'ID', got %s", validErr.Field)
			}
		} else {
			t.Errorf("expected MessageValidationError, got %T", err)
		}
	})
}

// TestTraceContextInjectionExtraction tests trace context propagation.
func TestTraceContextInjectionExtraction(t *testing.T) {
	msg := &ValidatedMessage{
		Headers: MessageHeaders{
			Custom: make(map[string]string),
		},
	}

	// Create a mock trace context
	traceID := trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	spanID := trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8}

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	originalCtx := trace.ContextWithSpanContext(context.Background(), spanContext)

	// Use TraceContext propagator for testing
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{})

	// Inject trace context into message headers
	msg.InjectTraceContext(originalCtx, propagator)

	// Verify that headers were set
	if len(msg.Headers.Custom) == 0 {
		t.Error("expected trace headers to be set")
	}

	// Extract trace context from message headers
	extractedCtx := msg.ExtractTraceContext(context.Background(), propagator)

	// Verify the extracted context contains the same trace information
	extractedSpanContext := trace.SpanFromContext(extractedCtx).SpanContext()
	if extractedSpanContext.TraceID() != traceID {
		t.Errorf("expected extracted trace ID %v, got %v", traceID, extractedSpanContext.TraceID())
	}
}

// TestHeaderCarrier tests the HeaderCarrier implementation.
func TestHeaderCarrier(t *testing.T) {
	carrier := &HeaderCarrier{headers: make(map[string]string)}

	// Test Set and Get
	carrier.Set("test-key", "test-value")
	value := carrier.Get("test-key")
	if value != "test-value" {
		t.Errorf("expected 'test-value', got '%s'", value)
	}

	// Test Keys
	carrier.Set("key1", "value1")
	carrier.Set("key2", "value2")
	keys := carrier.Keys()

	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}

	// Test Get non-existent key
	value = carrier.Get("non-existent")
	if value != "" {
		t.Errorf("expected empty string for non-existent key, got '%s'", value)
	}
}

// TestMessageValidationError tests the MessageValidationError type.
func TestMessageValidationError(t *testing.T) {
	t.Run("error with field", func(t *testing.T) {
		err := &MessageValidationError{
			Field:   "test_field",
			Message: "test message",
		}

		expected := "validation failed for field 'test_field': test message"
		if err.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("error without field", func(t *testing.T) {
		err := &MessageValidationError{
			Message: "test message",
		}

		expected := "validation failed: test message"
		if err.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("unwrap error", func(t *testing.T) {
		cause := &MessageValidationError{Message: "cause"}
		err := &MessageValidationError{
			Message: "wrapper",
			Cause:   cause,
		}

		unwrapped := err.Unwrap()
		if unwrapped != cause {
			t.Errorf("expected unwrapped error to be cause, got %v", unwrapped)
		}
	})
}

// TestMessageHeaders tests the MessageHeaders structure.
func TestMessageHeaders(t *testing.T) {
	headers := MessageHeaders{
		TraceID:       "trace-123",
		SpanID:        "span-456",
		CorrelationID: "corr-789",
		MessageType:   "UserCreated",
		ContentType:   "application/json",
		CreatedAt:     time.Now(),
		RetryCount:    0,
		MaxRetries:    3,
		Custom:        make(map[string]string),
	}

	headers.Custom["custom-key"] = "custom-value"

	// Test that all fields are accessible
	if headers.TraceID != "trace-123" {
		t.Errorf("expected trace ID 'trace-123', got %s", headers.TraceID)
	}

	if headers.Custom["custom-key"] != "custom-value" {
		t.Errorf("expected custom value 'custom-value', got %s", headers.Custom["custom-key"])
	}
}

// BenchmarkMessageValidation benchmarks message validation performance.
func BenchmarkMessageValidation(b *testing.B) {
	validator := NewStructMessageValidator()
	msg := &ValidatedMessage{
		ID:        uuid.New().String(),
		Topic:     "test.topic",
		Payload:   []byte("test payload"),
		Timestamp: time.Now(),
		Headers:   MessageHeaders{Custom: make(map[string]string)},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(msg)
	}
}

// BenchmarkMessageBuilder benchmarks message building performance.
func BenchmarkMessageBuilder(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewMessageBuilder().
			WithID(uuid.New().String()).
			WithTopic("test.topic").
			WithPayload([]byte("test payload")).
			Build()
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
