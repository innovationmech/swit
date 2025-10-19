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
	"encoding/json"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestSagaEvent creates a sample SagaEvent for testing.
func createTestSagaEvent() *saga.SagaEvent {
	return &saga.SagaEvent{
		ID:            "event-123",
		SagaID:        "saga-456",
		StepID:        "step-1",
		Type:          saga.EventSagaStepCompleted,
		Version:       "1.0",
		Timestamp:     time.Now(),
		CorrelationID: "corr-789",
		Data: map[string]interface{}{
			"order_id": "order-123",
			"amount":   100.50,
		},
		PreviousState: map[string]interface{}{
			"status": "pending",
		},
		NewState: map[string]interface{}{
			"status": "completed",
		},
		Duration:       time.Second * 2,
		Attempt:        1,
		MaxAttempts:    3,
		Metadata: map[string]interface{}{
			"user_id": "user-001",
			"region":  "us-west-1",
		},
		Source:         "order-service",
		Service:        "order-service",
		ServiceVersion: "v1.0.0",
		TraceID:        "trace-abc",
		SpanID:         "span-def",
		ParentSpanID:   "span-ghi",
	}
}

// createTestSagaEventWithError creates a sample SagaEvent with error for testing.
func createTestSagaEventWithError() *saga.SagaEvent {
	event := createTestSagaEvent()
	event.Type = saga.EventSagaStepFailed
	event.Error = &saga.SagaError{
		Code:      "PAYMENT_FAILED",
		Message:   "Payment processing failed",
		Retryable: true,
		Details: map[string]interface{}{
			"reason": "insufficient_funds",
			"amount": 100.50,
		},
	}
	return event
}

// TestJSONSagaEventSerializer_Serialize tests JSON serialization of SagaEvent.
func TestJSONSagaEventSerializer_Serialize(t *testing.T) {
	tests := []struct {
		name      string
		event     *saga.SagaEvent
		pretty    bool
		wantErr   bool
		errString string
	}{
		{
			name:    "valid event",
			event:   createTestSagaEvent(),
			pretty:  false,
			wantErr: false,
		},
		{
			name:    "valid event with pretty printing",
			event:   createTestSagaEvent(),
			pretty:  true,
			wantErr: false,
		},
		{
			name:    "event with error",
			event:   createTestSagaEventWithError(),
			pretty:  false,
			wantErr: false,
		},
		{
			name:      "nil event",
			event:     nil,
			pretty:    false,
			wantErr:   true,
			errString: "invalid event",
		},
		{
			name: "event missing ID",
			event: &saga.SagaEvent{
				SagaID:    "saga-456",
				Type:      saga.EventSagaStepCompleted,
				Timestamp: time.Now(),
			},
			pretty:    false,
			wantErr:   true,
			errString: "event ID is required",
		},
		{
			name: "event missing SagaID",
			event: &saga.SagaEvent{
				ID:        "event-123",
				Type:      saga.EventSagaStepCompleted,
				Timestamp: time.Now(),
			},
			pretty:    false,
			wantErr:   true,
			errString: "saga ID is required",
		},
		{
			name: "event missing Type",
			event: &saga.SagaEvent{
				ID:        "event-123",
				SagaID:    "saga-456",
				Timestamp: time.Now(),
			},
			pretty:    false,
			wantErr:   true,
			errString: "event type is required",
		},
		{
			name: "event missing Timestamp",
			event: &saga.SagaEvent{
				ID:     "event-123",
				SagaID: "saga-456",
				Type:   saga.EventSagaStepCompleted,
			},
			pretty:    false,
			wantErr:   true,
			errString: "event timestamp is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := NewJSONSagaEventSerializerWithOptions(tt.pretty, "1.0")
			ctx := context.Background()

			data, err := serializer.Serialize(ctx, tt.event)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, data)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, data)
				assert.NotEmpty(t, data)

				// Verify it's valid JSON
				var wrapper jsonEventWrapper
				err = json.Unmarshal(data, &wrapper)
				require.NoError(t, err)
				assert.Equal(t, "1.0", wrapper.Version)
				assert.NotNil(t, wrapper.Event)
			}
		})
	}
}

// TestJSONSagaEventSerializer_Deserialize tests JSON deserialization of SagaEvent.
func TestJSONSagaEventSerializer_Deserialize(t *testing.T) {
	serializer := NewJSONSagaEventSerializer()
	ctx := context.Background()

	tests := []struct {
		name      string
		setup     func() []byte
		wantErr   bool
		errString string
		validate  func(t *testing.T, event *saga.SagaEvent)
	}{
		{
			name: "valid serialized event",
			setup: func() []byte {
				event := createTestSagaEvent()
				data, err := serializer.Serialize(ctx, event)
				require.NoError(t, err)
				return data
			},
			wantErr: false,
			validate: func(t *testing.T, event *saga.SagaEvent) {
				assert.Equal(t, "event-123", event.ID)
				assert.Equal(t, "saga-456", event.SagaID)
				assert.Equal(t, "step-1", event.StepID)
				assert.Equal(t, saga.EventSagaStepCompleted, event.Type)
			},
		},
		{
			name: "valid serialized event with error",
			setup: func() []byte {
				event := createTestSagaEventWithError()
				data, err := serializer.Serialize(ctx, event)
				require.NoError(t, err)
				return data
			},
			wantErr: false,
			validate: func(t *testing.T, event *saga.SagaEvent) {
				assert.NotNil(t, event.Error)
				assert.Equal(t, "PAYMENT_FAILED", event.Error.Code)
				assert.Equal(t, "Payment processing failed", event.Error.Message)
				assert.True(t, event.Error.Retryable)
			},
		},
		{
			name: "backward compatibility - direct event JSON",
			setup: func() []byte {
				event := createTestSagaEvent()
				data, err := json.Marshal(event)
				require.NoError(t, err)
				return data
			},
			wantErr: false,
			validate: func(t *testing.T, event *saga.SagaEvent) {
				assert.Equal(t, "event-123", event.ID)
				assert.Equal(t, "saga-456", event.SagaID)
			},
		},
		{
			name: "empty data",
			setup: func() []byte {
				return []byte{}
			},
			wantErr:   true,
			errString: "cannot deserialize empty data",
		},
		{
			name: "invalid JSON",
			setup: func() []byte {
				return []byte("{invalid json")
			},
			wantErr:   true,
			errString: "failed to unmarshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.setup()
			event, err := serializer.Deserialize(ctx, data)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, event)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, event)
				if tt.validate != nil {
					tt.validate(t, event)
				}
			}
		})
	}
}

// TestJSONSagaEventSerializer_RoundTrip tests serialization and deserialization round trip.
func TestJSONSagaEventSerializer_RoundTrip(t *testing.T) {
	serializer := NewJSONSagaEventSerializer()
	ctx := context.Background()

	testCases := []struct {
		name  string
		event *saga.SagaEvent
	}{
		{
			name:  "simple event",
			event: createTestSagaEvent(),
		},
		{
			name:  "event with error",
			event: createTestSagaEventWithError(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize
			data, err := serializer.Serialize(ctx, tc.event)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Deserialize
			deserializedEvent, err := serializer.Deserialize(ctx, data)
			require.NoError(t, err)
			assert.NotNil(t, deserializedEvent)

			// Verify core fields match
			assert.Equal(t, tc.event.ID, deserializedEvent.ID)
			assert.Equal(t, tc.event.SagaID, deserializedEvent.SagaID)
			assert.Equal(t, tc.event.StepID, deserializedEvent.StepID)
			assert.Equal(t, tc.event.Type, deserializedEvent.Type)
			assert.Equal(t, tc.event.Version, deserializedEvent.Version)
			assert.Equal(t, tc.event.CorrelationID, deserializedEvent.CorrelationID)
			assert.Equal(t, tc.event.Source, deserializedEvent.Source)
			assert.Equal(t, tc.event.Service, deserializedEvent.Service)
			assert.Equal(t, tc.event.ServiceVersion, deserializedEvent.ServiceVersion)
			assert.Equal(t, tc.event.TraceID, deserializedEvent.TraceID)
			assert.Equal(t, tc.event.SpanID, deserializedEvent.SpanID)
			assert.Equal(t, tc.event.ParentSpanID, deserializedEvent.ParentSpanID)
			assert.Equal(t, tc.event.Attempt, deserializedEvent.Attempt)
			assert.Equal(t, tc.event.MaxAttempts, deserializedEvent.MaxAttempts)

			// Verify timestamp (with some tolerance for serialization precision)
			assert.WithinDuration(t, tc.event.Timestamp, deserializedEvent.Timestamp, time.Second)

			// Verify error if present
			if tc.event.Error != nil {
				require.NotNil(t, deserializedEvent.Error)
				assert.Equal(t, tc.event.Error.Code, deserializedEvent.Error.Code)
				assert.Equal(t, tc.event.Error.Message, deserializedEvent.Error.Message)
				assert.Equal(t, tc.event.Error.Retryable, deserializedEvent.Error.Retryable)
			}
		})
	}
}

// TestProtobufSagaEventSerializer_Serialize tests Protobuf serialization of SagaEvent.
func TestProtobufSagaEventSerializer_Serialize(t *testing.T) {
	tests := []struct {
		name      string
		event     *saga.SagaEvent
		wantErr   bool
		errString string
	}{
		{
			name:    "valid event",
			event:   createTestSagaEvent(),
			wantErr: false,
		},
		{
			name:    "event with error",
			event:   createTestSagaEventWithError(),
			wantErr: false,
		},
		{
			name:      "nil event",
			event:     nil,
			wantErr:   true,
			errString: "invalid event",
		},
		{
			name: "event missing ID",
			event: &saga.SagaEvent{
				SagaID:    "saga-456",
				Type:      saga.EventSagaStepCompleted,
				Timestamp: time.Now(),
			},
			wantErr:   true,
			errString: "event ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := NewProtobufSagaEventSerializer()
			ctx := context.Background()

			data, err := serializer.Serialize(ctx, tt.event)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, data)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, data)
				assert.NotEmpty(t, data)
			}
		})
	}
}

// TestProtobufSagaEventSerializer_Deserialize tests Protobuf deserialization of SagaEvent.
func TestProtobufSagaEventSerializer_Deserialize(t *testing.T) {
	serializer := NewProtobufSagaEventSerializer()
	ctx := context.Background()

	tests := []struct {
		name      string
		setup     func() []byte
		wantErr   bool
		errString string
		validate  func(t *testing.T, event *saga.SagaEvent)
	}{
		{
			name: "valid serialized event",
			setup: func() []byte {
				event := createTestSagaEvent()
				data, err := serializer.Serialize(ctx, event)
				require.NoError(t, err)
				return data
			},
			wantErr: false,
			validate: func(t *testing.T, event *saga.SagaEvent) {
				assert.Equal(t, "event-123", event.ID)
				assert.Equal(t, "saga-456", event.SagaID)
				assert.Equal(t, "step-1", event.StepID)
				assert.Equal(t, saga.EventSagaStepCompleted, event.Type)
			},
		},
		{
			name: "valid serialized event with error",
			setup: func() []byte {
				event := createTestSagaEventWithError()
				data, err := serializer.Serialize(ctx, event)
				require.NoError(t, err)
				return data
			},
			wantErr: false,
			validate: func(t *testing.T, event *saga.SagaEvent) {
				assert.NotNil(t, event.Error)
				assert.Equal(t, "PAYMENT_FAILED", event.Error.Code)
				assert.Equal(t, "Payment processing failed", event.Error.Message)
				assert.True(t, event.Error.Retryable)
			},
		},
		{
			name: "empty data",
			setup: func() []byte {
				return []byte{}
			},
			wantErr:   true,
			errString: "cannot deserialize empty data",
		},
		{
			name: "invalid protobuf data",
			setup: func() []byte {
				return []byte{0xFF, 0xFF, 0xFF}
			},
			wantErr:   true,
			errString: "failed to unmarshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.setup()
			event, err := serializer.Deserialize(ctx, data)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, event)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, event)
				if tt.validate != nil {
					tt.validate(t, event)
				}
			}
		})
	}
}

// TestProtobufSagaEventSerializer_RoundTrip tests Protobuf serialization and deserialization round trip.
func TestProtobufSagaEventSerializer_RoundTrip(t *testing.T) {
	serializer := NewProtobufSagaEventSerializer()
	ctx := context.Background()

	testCases := []struct {
		name  string
		event *saga.SagaEvent
	}{
		{
			name:  "simple event",
			event: createTestSagaEvent(),
		},
		{
			name:  "event with error",
			event: createTestSagaEventWithError(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize
			data, err := serializer.Serialize(ctx, tc.event)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Deserialize
			deserializedEvent, err := serializer.Deserialize(ctx, data)
			require.NoError(t, err)
			assert.NotNil(t, deserializedEvent)

			// Verify core fields match
			assert.Equal(t, tc.event.ID, deserializedEvent.ID)
			assert.Equal(t, tc.event.SagaID, deserializedEvent.SagaID)
			assert.Equal(t, tc.event.StepID, deserializedEvent.StepID)
			assert.Equal(t, tc.event.Type, deserializedEvent.Type)
			assert.Equal(t, tc.event.CorrelationID, deserializedEvent.CorrelationID)
			assert.Equal(t, tc.event.Source, deserializedEvent.Source)
			assert.Equal(t, tc.event.Service, deserializedEvent.Service)
			assert.Equal(t, tc.event.ServiceVersion, deserializedEvent.ServiceVersion)
			assert.Equal(t, tc.event.TraceID, deserializedEvent.TraceID)
			assert.Equal(t, tc.event.SpanID, deserializedEvent.SpanID)
			assert.Equal(t, tc.event.ParentSpanID, deserializedEvent.ParentSpanID)
			assert.Equal(t, tc.event.Attempt, deserializedEvent.Attempt)
			assert.Equal(t, tc.event.MaxAttempts, deserializedEvent.MaxAttempts)

			// Verify error if present
			if tc.event.Error != nil {
				require.NotNil(t, deserializedEvent.Error)
				assert.Equal(t, tc.event.Error.Code, deserializedEvent.Error.Code)
				assert.Equal(t, tc.event.Error.Message, deserializedEvent.Error.Message)
				assert.Equal(t, tc.event.Error.Retryable, deserializedEvent.Error.Retryable)
			}
		})
	}
}

// TestSagaEventSerializer_ContentType tests ContentType method.
func TestSagaEventSerializer_ContentType(t *testing.T) {
	tests := []struct {
		name       string
		serializer SagaEventSerializer
		expected   string
	}{
		{
			name:       "JSON serializer",
			serializer: NewJSONSagaEventSerializer(),
			expected:   "application/json",
		},
		{
			name:       "Protobuf serializer",
			serializer: NewProtobufSagaEventSerializer(),
			expected:   "application/x-protobuf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contentType := tt.serializer.ContentType()
			assert.Equal(t, tt.expected, contentType)
		})
	}
}

// TestSagaEventSerializer_FormatName tests FormatName method.
func TestSagaEventSerializer_FormatName(t *testing.T) {
	tests := []struct {
		name       string
		serializer SagaEventSerializer
		expected   string
	}{
		{
			name:       "JSON serializer",
			serializer: NewJSONSagaEventSerializer(),
			expected:   "json",
		},
		{
			name:       "Protobuf serializer",
			serializer: NewProtobufSagaEventSerializer(),
			expected:   "protobuf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatName := tt.serializer.FormatName()
			assert.Equal(t, tt.expected, formatName)
		})
	}
}

// TestNewSagaEventSerializer tests the factory function.
func TestNewSagaEventSerializer(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		options     *SerializerOptions
		wantErr     bool
		errString   string
		validateFn  func(t *testing.T, serializer SagaEventSerializer)
	}{
		{
			name:    "JSON format",
			format:  "json",
			options: nil,
			wantErr: false,
			validateFn: func(t *testing.T, serializer SagaEventSerializer) {
				assert.Equal(t, "json", serializer.FormatName())
			},
		},
		{
			name:    "Protobuf format",
			format:  "protobuf",
			options: nil,
			wantErr: false,
			validateFn: func(t *testing.T, serializer SagaEventSerializer) {
				assert.Equal(t, "protobuf", serializer.FormatName())
			},
		},
		{
			name:    "Proto alias format",
			format:  "proto",
			options: nil,
			wantErr: false,
			validateFn: func(t *testing.T, serializer SagaEventSerializer) {
				assert.Equal(t, "protobuf", serializer.FormatName())
			},
		},
		{
			name:   "JSON with options",
			format: "json",
			options: &SerializerOptions{
				Pretty:  true,
				Version: "2.0",
			},
			wantErr: false,
			validateFn: func(t *testing.T, serializer SagaEventSerializer) {
				assert.Equal(t, "json", serializer.FormatName())
			},
		},
		{
			name:      "unsupported format",
			format:    "xml",
			options:   nil,
			wantErr:   true,
			errString: "unsupported serialization format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer, err := NewSagaEventSerializer(tt.format, tt.options)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, serializer)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, serializer)
				if tt.validateFn != nil {
					tt.validateFn(t, serializer)
				}
			}
		})
	}
}

// TestSagaEventSerializer_ContextCancellation tests context cancellation handling.
func TestSagaEventSerializer_ContextCancellation(t *testing.T) {
	serializers := []struct {
		name       string
		serializer SagaEventSerializer
	}{
		{
			name:       "JSON serializer",
			serializer: NewJSONSagaEventSerializer(),
		},
		{
			name:       "Protobuf serializer",
			serializer: NewProtobufSagaEventSerializer(),
		},
	}

	for _, ts := range serializers {
		t.Run(ts.name+" serialize with cancelled context", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			event := createTestSagaEvent()
			data, err := ts.serializer.Serialize(ctx, event)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "cancelled")
			assert.Nil(t, data)
		})

		t.Run(ts.name+" deserialize with cancelled context", func(t *testing.T) {
			// First serialize with valid context
			event := createTestSagaEvent()
			data, err := ts.serializer.Serialize(context.Background(), event)
			require.NoError(t, err)

			// Then try to deserialize with cancelled context
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			deserializedEvent, err := ts.serializer.Deserialize(ctx, data)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "cancelled")
			assert.Nil(t, deserializedEvent)
		})
	}
}

// TestSagaEventSerializer_Validate tests the Validate method.
func TestSagaEventSerializer_Validate(t *testing.T) {
	serializers := []struct {
		name       string
		serializer SagaEventSerializer
	}{
		{
			name:       "JSON serializer",
			serializer: NewJSONSagaEventSerializer(),
		},
		{
			name:       "Protobuf serializer",
			serializer: NewProtobufSagaEventSerializer(),
		},
	}

	tests := []struct {
		name      string
		event     *saga.SagaEvent
		wantErr   bool
		errString string
	}{
		{
			name:    "valid event",
			event:   createTestSagaEvent(),
			wantErr: false,
		},
		{
			name:      "nil event",
			event:     nil,
			wantErr:   true,
			errString: "invalid event",
		},
		{
			name: "missing ID",
			event: &saga.SagaEvent{
				SagaID:    "saga-456",
				Type:      saga.EventSagaStepCompleted,
				Timestamp: time.Now(),
			},
			wantErr:   true,
			errString: "event ID is required",
		},
		{
			name: "missing SagaID",
			event: &saga.SagaEvent{
				ID:        "event-123",
				Type:      saga.EventSagaStepCompleted,
				Timestamp: time.Now(),
			},
			wantErr:   true,
			errString: "saga ID is required",
		},
		{
			name: "missing Type",
			event: &saga.SagaEvent{
				ID:        "event-123",
				SagaID:    "saga-456",
				Timestamp: time.Now(),
			},
			wantErr:   true,
			errString: "event type is required",
		},
		{
			name: "missing Timestamp",
			event: &saga.SagaEvent{
				ID:     "event-123",
				SagaID: "saga-456",
				Type:   saga.EventSagaStepCompleted,
			},
			wantErr:   true,
			errString: "event timestamp is required",
		},
	}

	for _, ts := range serializers {
		for _, tt := range tests {
			t.Run(ts.name+" - "+tt.name, func(t *testing.T) {
				ctx := context.Background()
				err := ts.serializer.Validate(ctx, tt.event)

				if tt.wantErr {
					require.Error(t, err)
					if tt.errString != "" {
						assert.Contains(t, err.Error(), tt.errString)
					}
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}

