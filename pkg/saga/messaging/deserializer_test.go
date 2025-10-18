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

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/saga"
)

func TestNewDefaultMessageDeserializer(t *testing.T) {
	deserializer := NewDefaultMessageDeserializer()
	if deserializer == nil {
		t.Fatal("Expected non-nil deserializer")
	}

	formats := deserializer.GetSupportedFormats()
	if len(formats) != 2 {
		t.Errorf("Expected 2 supported formats, got %d", len(formats))
	}
}

func TestMessageDeserializer_Deserialize_JSON(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		headers     map[string]string
		expectError bool
		validate    func(*testing.T, *saga.SagaEvent)
	}{
		{
			name: "valid JSON event",
			data: []byte(`{
				"id": "event-123",
				"saga_id": "saga-456",
				"type": "saga.started",
				"timestamp": "2025-01-01T00:00:00Z",
				"data": {"key": "value"}
			}`),
			headers:     map[string]string{"content-type": "application/json"},
			expectError: false,
			validate: func(t *testing.T, event *saga.SagaEvent) {
				if event.ID != "event-123" {
					t.Errorf("Expected event ID 'event-123', got '%s'", event.ID)
				}
				if event.SagaID != "saga-456" {
					t.Errorf("Expected saga ID 'saga-456', got '%s'", event.SagaID)
				}
				if event.Type != saga.EventSagaStarted {
					t.Errorf("Expected event type 'saga.started', got '%s'", event.Type)
				}
			},
		},
		{
			name: "valid JSON with metadata headers",
			data: []byte(`{
				"id": "event-789",
				"saga_id": "saga-012",
				"type": "saga.step.completed",
				"timestamp": "2025-01-01T00:00:00Z"
			}`),
			headers: map[string]string{
				"content-type":      "application/json",
				"x-correlation-id":  "corr-123",
				"x-trace-id":        "trace-456",
				"x-metadata-custom": "custom-value",
			},
			expectError: false,
			validate: func(t *testing.T, event *saga.SagaEvent) {
				if event.Metadata == nil {
					t.Fatal("Expected metadata to be set")
				}
				if event.Metadata["correlation_id"] != "corr-123" {
					t.Errorf("Expected correlation_id 'corr-123', got '%v'", event.Metadata["correlation_id"])
				}
				if event.Metadata["trace_id"] != "trace-456" {
					t.Errorf("Expected trace_id 'trace-456', got '%v'", event.Metadata["trace_id"])
				}
				if event.Metadata["custom"] != "custom-value" {
					t.Errorf("Expected custom metadata 'custom-value', got '%v'", event.Metadata["custom"])
				}
			},
		},
		{
			name:        "empty data",
			data:        []byte{},
			headers:     map[string]string{},
			expectError: true,
		},
		{
			name: "invalid JSON",
			data: []byte(`{invalid json`),
			headers: map[string]string{
				"content-type": "application/json",
			},
			expectError: true,
		},
		{
			name: "missing required field - event ID",
			data: []byte(`{
				"saga_id": "saga-123",
				"type": "saga.started",
				"timestamp": "2025-01-01T00:00:00Z"
			}`),
			headers: map[string]string{
				"content-type": "application/json",
			},
			expectError: true,
		},
		{
			name: "missing required field - saga ID",
			data: []byte(`{
				"id": "event-123",
				"type": "saga.started",
				"timestamp": "2025-01-01T00:00:00Z"
			}`),
			headers: map[string]string{
				"content-type": "application/json",
			},
			expectError: true,
		},
		{
			name: "invalid event type",
			data: []byte(`{
				"id": "event-123",
				"saga_id": "saga-456",
				"type": "invalid.type",
				"timestamp": "2025-01-01T00:00:00Z"
			}`),
			headers: map[string]string{
				"content-type": "application/json",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deserializer := NewDefaultMessageDeserializer()
			ctx := context.Background()

			event, err := deserializer.Deserialize(ctx, tt.data, tt.headers)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if event == nil {
				t.Fatal("Expected non-nil event")
			}

			if tt.validate != nil {
				tt.validate(t, event)
			}
		})
	}
}

func TestMessageDeserializer_DeserializeWithFormat(t *testing.T) {
	deserializer := NewDefaultMessageDeserializer()
	ctx := context.Background()

	event := &saga.SagaEvent{
		ID:        "event-123",
		SagaID:    "saga-456",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}

	// Serialize to JSON
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	headers := map[string]string{}

	// Deserialize with explicit format
	deserializedEvent, err := deserializer.DeserializeWithFormat(ctx, data, messaging.FormatJSON, headers)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	if deserializedEvent.ID != event.ID {
		t.Errorf("Expected ID '%s', got '%s'", event.ID, deserializedEvent.ID)
	}
	if deserializedEvent.SagaID != event.SagaID {
		t.Errorf("Expected SagaID '%s', got '%s'", event.SagaID, deserializedEvent.SagaID)
	}
}

func TestMessageDeserializer_ValidateMessage(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		headers     map[string]string
		expectError bool
	}{
		{
			name: "valid message",
			data: []byte(`{
				"id": "event-123",
				"saga_id": "saga-456",
				"type": "saga.started",
				"timestamp": "2025-01-01T00:00:00Z"
			}`),
			headers:     map[string]string{"content-type": "application/json"},
			expectError: false,
		},
		{
			name:        "empty data",
			data:        []byte{},
			headers:     map[string]string{},
			expectError: true,
		},
		{
			name: "invalid JSON",
			data: []byte(`{invalid`),
			headers: map[string]string{
				"content-type": "application/json",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deserializer := NewDefaultMessageDeserializer()
			ctx := context.Background()

			err := deserializer.ValidateMessage(ctx, tt.data, tt.headers)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMessageDeserializer_ExtractMetadata(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected map[string]interface{}
	}{
		{
			name: "all standard headers",
			headers: map[string]string{
				"x-correlation-id": "corr-123",
				"x-trace-id":       "trace-456",
				"x-span-id":        "span-789",
				"x-event-type":     "saga.started",
				"x-saga-id":        "saga-123",
				"x-step-id":        "step-456",
				"x-source":         "service-a",
				"x-service":        "saga-coordinator",
			},
			expected: map[string]interface{}{
				"correlation_id": "corr-123",
				"trace_id":       "trace-456",
				"span_id":        "span-789",
				"event_type":     "saga.started",
				"saga_id":        "saga-123",
				"step_id":        "step-456",
				"source":         "service-a",
				"service":        "saga-coordinator",
			},
		},
		{
			name: "custom metadata headers",
			headers: map[string]string{
				"x-metadata-key1": "value1",
				"x-metadata-key2": "value2",
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "headers without x- prefix",
			headers: map[string]string{
				"correlation-id": "corr-123",
				"trace-id":       "trace-456",
			},
			expected: map[string]interface{}{
				"correlation_id": "corr-123",
				"trace_id":       "trace-456",
			},
		},
		{
			name:     "empty headers",
			headers:  map[string]string{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deserializer := NewDefaultMessageDeserializer()
			metadata := deserializer.ExtractMetadata(tt.headers)

			if len(metadata) != len(tt.expected) {
				t.Errorf("Expected %d metadata entries, got %d", len(tt.expected), len(metadata))
			}

			for key, expectedValue := range tt.expected {
				actualValue, exists := metadata[key]
				if !exists {
					t.Errorf("Expected metadata key '%s' not found", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("Expected metadata['%s'] = '%v', got '%v'", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestMessageDeserializer_DetectFormat(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		headers        map[string]string
		expectedFormat messaging.SerializationFormat
		expectError    bool
	}{
		{
			name: "JSON from content-type header",
			data: []byte(`{"test": "data"}`),
			headers: map[string]string{
				"content-type": "application/json",
			},
			expectedFormat: messaging.FormatJSON,
			expectError:    false,
		},
		{
			name: "Protobuf from content-type header",
			data: []byte{0x08, 0x01},
			headers: map[string]string{
				"content-type": "application/x-protobuf",
			},
			expectedFormat: messaging.FormatProtobuf,
			expectError:    false,
		},
		{
			name: "JSON from format header",
			data: []byte(`{"test": "data"}`),
			headers: map[string]string{
				"x-format": "json",
			},
			expectedFormat: messaging.FormatJSON,
			expectError:    false,
		},
		{
			name:           "JSON from content detection",
			data:           []byte(`{"test": "data"}`),
			headers:        map[string]string{},
			expectedFormat: messaging.FormatJSON,
			expectError:    false,
		},
		{
			name:           "JSON array from content detection",
			data:           []byte(`[{"test": "data"}]`),
			headers:        map[string]string{},
			expectedFormat: messaging.FormatJSON,
			expectError:    false,
		},
		{
			name:           "JSON with whitespace",
			data:           []byte(`   {"test": "data"}`),
			headers:        map[string]string{},
			expectedFormat: messaging.FormatJSON,
			expectError:    false,
		},
		{
			name:        "empty data",
			data:        []byte{},
			headers:     map[string]string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deserializer := NewDefaultMessageDeserializer()
			ctx := context.Background()

			format, err := deserializer.DetectFormat(ctx, tt.data, tt.headers)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if format != tt.expectedFormat {
				t.Errorf("Expected format '%s', got '%s'", tt.expectedFormat, format)
			}
		})
	}
}

func TestMessageDeserializer_GetSupportedFormats(t *testing.T) {
	deserializer := NewDefaultMessageDeserializer()
	formats := deserializer.GetSupportedFormats()

	if len(formats) != 2 {
		t.Errorf("Expected 2 supported formats, got %d", len(formats))
	}

	hasJSON := false
	hasProtobuf := false
	for _, format := range formats {
		if format == messaging.FormatJSON {
			hasJSON = true
		}
		if format == messaging.FormatProtobuf {
			hasProtobuf = true
		}
	}

	if !hasJSON {
		t.Error("Expected JSON format to be supported")
	}
	if !hasProtobuf {
		t.Error("Expected Protobuf format to be supported")
	}
}

func TestNewMessageDeserializerWithSerializers(t *testing.T) {
	// Create custom serializers
	jsonSerializer := messaging.NewJSONSerializer(nil)
	protobufSerializer := messaging.NewProtobufSerializer(nil)

	// Test with custom serializers
	deserializer := NewMessageDeserializerWithSerializers(jsonSerializer, protobufSerializer)
	if deserializer == nil {
		t.Fatal("Expected non-nil deserializer")
	}

	formats := deserializer.GetSupportedFormats()
	if len(formats) != 2 {
		t.Errorf("Expected 2 supported formats, got %d", len(formats))
	}

	// Test with nil serializers (should use defaults)
	deserializer = NewMessageDeserializerWithSerializers(nil, nil)
	if deserializer == nil {
		t.Fatal("Expected non-nil deserializer")
	}

	formats = deserializer.GetSupportedFormats()
	if len(formats) != 2 {
		t.Errorf("Expected 2 supported formats, got %d", len(formats))
	}
}

func BenchmarkMessageDeserializer_Deserialize_JSON(b *testing.B) {
	deserializer := NewDefaultMessageDeserializer()
	ctx := context.Background()

	data := []byte(`{
		"id": "event-123",
		"saga_id": "saga-456",
		"type": "saga.started",
		"timestamp": "2025-01-01T00:00:00Z",
		"data": {"key": "value"}
	}`)
	headers := map[string]string{"content-type": "application/json"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := deserializer.Deserialize(ctx, data, headers)
		if err != nil {
			b.Fatalf("Deserialization error: %v", err)
		}
	}
}

func BenchmarkMessageDeserializer_ExtractMetadata(b *testing.B) {
	deserializer := NewDefaultMessageDeserializer()

	headers := map[string]string{
		"x-correlation-id": "corr-123",
		"x-trace-id":       "trace-456",
		"x-span-id":        "span-789",
		"x-saga-id":        "saga-123",
		"x-metadata-key1":  "value1",
		"x-metadata-key2":  "value2",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = deserializer.ExtractMetadata(headers)
	}
}
