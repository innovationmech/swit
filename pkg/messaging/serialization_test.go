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
	"strings"
	"testing"
	"time"
)

// TestMessage represents a test message for serialization testing.
type TestMessage struct {
	ID        string            `json:"id" validate:"required"`
	Name      string            `json:"name" validate:"required,min=1"`
	Value     int               `json:"value" validate:"min=0"`
	Tags      []string          `json:"tags,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

func TestSerializationFormat_String(t *testing.T) {
	tests := []struct {
		format   SerializationFormat
		expected string
	}{
		{FormatJSON, "json"},
		{FormatProtobuf, "protobuf"},
		{FormatAvro, "avro"},
		{SerializationFormat("unknown"), "unknown"},
	}

	for _, test := range tests {
		t.Run(string(test.format), func(t *testing.T) {
			if got := test.format.String(); got != test.expected {
				t.Errorf("SerializationFormat.String() = %v, want %v", got, test.expected)
			}
		})
	}
}

func TestSerializationFormat_ContentType(t *testing.T) {
	tests := []struct {
		format   SerializationFormat
		expected string
	}{
		{FormatJSON, "application/json"},
		{FormatProtobuf, "application/x-protobuf"},
		{FormatAvro, "application/avro"},
		{SerializationFormat("unknown"), "application/octet-stream"},
	}

	for _, test := range tests {
		t.Run(string(test.format), func(t *testing.T) {
			if got := test.format.ContentType(); got != test.expected {
				t.Errorf("SerializationFormat.ContentType() = %v, want %v", got, test.expected)
			}
		})
	}
}

func TestDefaultSerializationRegistry_RegisterAndGetSerializer(t *testing.T) {
	registry := NewSerializationRegistry()

	// Test registering a JSON serializer
	jsonSerializer := NewJSONSerializer(nil)
	err := registry.RegisterSerializer(FormatJSON, jsonSerializer)
	if err != nil {
		t.Fatalf("Failed to register JSON serializer: %v", err)
	}

	// Test retrieving the serializer
	retrievedSerializer, err := registry.GetSerializer(FormatJSON)
	if err != nil {
		t.Fatalf("Failed to get JSON serializer: %v", err)
	}

	if retrievedSerializer.GetFormat() != FormatJSON {
		t.Errorf("Retrieved serializer format = %v, want %v", retrievedSerializer.GetFormat(), FormatJSON)
	}
}

func TestDefaultSerializationRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewSerializationRegistry()

	// Register first serializer
	jsonSerializer1 := NewJSONSerializer(nil)
	err := registry.RegisterSerializer(FormatJSON, jsonSerializer1)
	if err != nil {
		t.Fatalf("Failed to register first JSON serializer: %v", err)
	}

	// Try to register duplicate
	jsonSerializer2 := NewJSONSerializer(nil)
	err = registry.RegisterSerializer(FormatJSON, jsonSerializer2)

	if err == nil {
		t.Error("Expected error when registering duplicate serializer, got nil")
	}

	var regErr *RegistrationError
	if !errorAs(err, &regErr) {
		t.Errorf("Expected RegistrationError, got %T", err)
	}
}

func TestDefaultSerializationRegistry_UnregisterSerializer(t *testing.T) {
	registry := NewSerializationRegistry()

	// Register a serializer
	jsonSerializer := NewJSONSerializer(nil)
	err := registry.RegisterSerializer(FormatJSON, jsonSerializer)
	if err != nil {
		t.Fatalf("Failed to register JSON serializer: %v", err)
	}

	// Verify it exists
	_, err = registry.GetSerializer(FormatJSON)
	if err != nil {
		t.Fatalf("Failed to get registered serializer: %v", err)
	}

	// Unregister it
	err = registry.UnregisterSerializer(FormatJSON)
	if err != nil {
		t.Fatalf("Failed to unregister serializer: %v", err)
	}

	// Verify it's gone
	_, err = registry.GetSerializer(FormatJSON)
	if err == nil {
		t.Error("Expected error when getting unregistered serializer, got nil")
	}
}

func TestDefaultSerializationRegistry_GetSupportedFormats(t *testing.T) {
	registry := NewSerializationRegistry()

	// Initially should be empty
	formats := registry.GetSupportedFormats()
	if len(formats) != 0 {
		t.Errorf("Expected 0 formats initially, got %d", len(formats))
	}

	// Register serializers
	jsonSerializer := NewJSONSerializer(nil)
	protobufSerializer := NewProtobufSerializer(nil)

	_ = registry.RegisterSerializer(FormatJSON, jsonSerializer)
	_ = registry.RegisterSerializer(FormatProtobuf, protobufSerializer)

	formats = registry.GetSupportedFormats()
	if len(formats) != 2 {
		t.Errorf("Expected 2 formats after registration, got %d", len(formats))
	}

	// Check if both formats are present
	hasJSON := false
	hasProtobuf := false
	for _, format := range formats {
		if format == FormatJSON {
			hasJSON = true
		} else if format == FormatProtobuf {
			hasProtobuf = true
		}
	}

	if !hasJSON {
		t.Error("JSON format not found in supported formats")
	}
	if !hasProtobuf {
		t.Error("Protobuf format not found in supported formats")
	}
}

func TestDefaultSerializationRegistry_DetectFormat(t *testing.T) {
	registry := NewSerializationRegistry()
	ctx := context.Background()

	tests := []struct {
		name     string
		data     []byte
		expected SerializationFormat
		hasError bool
	}{
		{
			name:     "JSON object",
			data:     []byte(`{"id": "test", "name": "example"}`),
			expected: FormatJSON,
			hasError: false,
		},
		{
			name:     "JSON array",
			data:     []byte(`[{"id": "test"}]`),
			expected: FormatJSON,
			hasError: false,
		},
		{
			name:     "Binary data (protobuf-like)",
			data:     []byte{0x08, 0x96, 0x01, 0x12, 0x04, 0x74, 0x65, 0x73, 0x74},
			expected: FormatProtobuf,
			hasError: false,
		},
		{
			name:     "Empty data",
			data:     []byte{},
			expected: "",
			hasError: true,
		},
		{
			name:     "Unrecognizable format",
			data:     []byte("plain text content"),
			expected: "",
			hasError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			format, err := registry.DetectFormat(ctx, test.data)

			if test.hasError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if format != test.expected {
					t.Errorf("DetectFormat() = %v, want %v", format, test.expected)
				}
			}
		})
	}
}

func TestDefaultSerializationRegistry_ValidateFormat(t *testing.T) {
	registry := NewSerializationRegistry()

	// Register JSON serializer
	jsonSerializer := NewJSONSerializer(nil)
	_ = registry.RegisterSerializer(FormatJSON, jsonSerializer)

	// Test valid format
	err := registry.ValidateFormat(FormatJSON)
	if err != nil {
		t.Errorf("ValidateFormat() for registered format returned error: %v", err)
	}

	// Test invalid format
	err = registry.ValidateFormat(FormatProtobuf)
	if err == nil {
		t.Error("ValidateFormat() for unregistered format should return error")
	}
}

func TestDefaultSerializationRegistry_GetMetrics(t *testing.T) {
	registry := NewSerializationRegistry()

	// Register a serializer
	jsonSerializer := NewJSONSerializer(nil)
	_ = registry.RegisterSerializer(FormatJSON, jsonSerializer)

	metrics := registry.GetMetrics()

	if metrics == nil {
		t.Fatal("GetMetrics() returned nil")
	}

	if len(metrics.RegisteredFormats) != 1 {
		t.Errorf("Expected 1 registered format, got %d", len(metrics.RegisteredFormats))
	}

	if metrics.RegisteredFormats[0] != FormatJSON {
		t.Errorf("Expected JSON format, got %v", metrics.RegisteredFormats[0])
	}

	// Check that all metric maps are initialized
	if metrics.SerializationCount == nil {
		t.Error("SerializationCount map is nil")
	}
	if metrics.DeserializationCount == nil {
		t.Error("DeserializationCount map is nil")
	}
	if metrics.ErrorCount == nil {
		t.Error("ErrorCount map is nil")
	}
}

func TestDefaultSerializationRegistry_Close(t *testing.T) {
	registry := NewSerializationRegistry()

	// Register serializers
	jsonSerializer := NewJSONSerializer(nil)
	protobufSerializer := NewProtobufSerializer(nil)

	_ = registry.RegisterSerializer(FormatJSON, jsonSerializer)
	_ = registry.RegisterSerializer(FormatProtobuf, protobufSerializer)

	// Verify they're registered
	formats := registry.GetSupportedFormats()
	if len(formats) != 2 {
		t.Errorf("Expected 2 formats before close, got %d", len(formats))
	}

	// Close the registry
	err := registry.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Verify they're all removed
	formats = registry.GetSupportedFormats()
	if len(formats) != 0 {
		t.Errorf("Expected 0 formats after close, got %d", len(formats))
	}
}

func TestNewDefaultSerializationRegistry(t *testing.T) {
	registry, err := NewDefaultSerializationRegistry()
	if err != nil {
		t.Fatalf("NewDefaultSerializationRegistry() returned error: %v", err)
	}

	// Should have JSON and Protobuf serializers registered
	formats := registry.GetSupportedFormats()
	if len(formats) != 2 {
		t.Errorf("Expected 2 default serializers, got %d", len(formats))
	}

	// Test JSON serializer
	jsonSerializer, err := registry.GetSerializer(FormatJSON)
	if err != nil {
		t.Errorf("Failed to get JSON serializer: %v", err)
	}
	if jsonSerializer.GetFormat() != FormatJSON {
		t.Errorf("JSON serializer format = %v, want %v", jsonSerializer.GetFormat(), FormatJSON)
	}

	// Test Protobuf serializer
	protobufSerializer, err := registry.GetSerializer(FormatProtobuf)
	if err != nil {
		t.Errorf("Failed to get Protobuf serializer: %v", err)
	}
	if protobufSerializer.GetFormat() != FormatProtobuf {
		t.Errorf("Protobuf serializer format = %v, want %v", protobufSerializer.GetFormat(), FormatProtobuf)
	}
}

func TestSerializationError(t *testing.T) {
	tests := []struct {
		name     string
		error    SerializationError
		expected string
	}{
		{
			name: "With format",
			error: SerializationError{
				Format:  FormatJSON,
				Message: "test error",
			},
			expected: "serialization error for format 'json': test error",
		},
		{
			name: "Without format",
			error: SerializationError{
				Message: "test error",
			},
			expected: "serialization error: test error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.error.Error(); got != test.expected {
				t.Errorf("SerializationError.Error() = %v, want %v", got, test.expected)
			}
		})
	}
}

func TestRegistrationError(t *testing.T) {
	tests := []struct {
		name     string
		error    RegistrationError
		expected string
	}{
		{
			name: "With format",
			error: RegistrationError{
				Format:  FormatJSON,
				Message: "test error",
			},
			expected: "registration error for format 'json': test error",
		},
		{
			name: "Without format",
			error: RegistrationError{
				Message: "test error",
			},
			expected: "registration error: test error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.error.Error(); got != test.expected {
				t.Errorf("RegistrationError.Error() = %v, want %v", got, test.expected)
			}
		})
	}
}

func TestFormatDetectionError(t *testing.T) {
	err := FormatDetectionError{
		Message: "test error",
	}

	expected := "format detection error: test error"
	if got := err.Error(); got != expected {
		t.Errorf("FormatDetectionError.Error() = %v, want %v", got, expected)
	}
}

func TestSchemaError(t *testing.T) {
	err := SchemaError{
		Message: "test error",
	}

	expected := "schema error: test error"
	if got := err.Error(); got != expected {
		t.Errorf("SchemaError.Error() = %v, want %v", got, expected)
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name     string
		error    ValidationError
		expected string
	}{
		{
			name: "With field",
			error: ValidationError{
				Field:   "name",
				Message: "test error",
			},
			expected: "validation error for field 'name': test error",
		},
		{
			name: "Without field",
			error: ValidationError{
				Message: "test error",
			},
			expected: "validation error: test error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.error.Error(); got != test.expected {
				t.Errorf("ValidationError.Error() = %v, want %v", got, test.expected)
			}
		})
	}
}

// Benchmarks
func BenchmarkSerializationRegistry_GetSerializer(b *testing.B) {
	registry := NewSerializationRegistry()
	jsonSerializer := NewJSONSerializer(nil)
	_ = registry.RegisterSerializer(FormatJSON, jsonSerializer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.GetSerializer(FormatJSON)
	}
}

func BenchmarkDefaultSerializationRegistry_DetectFormat(b *testing.B) {
	registry := NewSerializationRegistry()
	ctx := context.Background()
	testData := []byte(`{"id": "test", "name": "benchmark"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.DetectFormat(ctx, testData)
	}
}

// Helper function for error assertion (simplified version of errors.As)
func errorAs(err error, target interface{}) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "registration error")
}
