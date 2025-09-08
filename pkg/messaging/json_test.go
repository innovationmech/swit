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
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewJSONSerializer(t *testing.T) {
	tests := []struct {
		name     string
		options  *JSONSerializationOptions
		validate func(*JSONSerializer) error
	}{
		{
			name:    "Default options",
			options: nil,
			validate: func(s *JSONSerializer) error {
				if s.options.Pretty {
					t.Error("Expected Pretty to be false by default")
				}
				if s.options.IndentString != "  " {
					t.Errorf("Expected IndentString to be '  ', got '%s'", s.options.IndentString)
				}
				if !s.options.ValidateOnSerialize {
					t.Error("Expected ValidateOnSerialize to be true by default")
				}
				return nil
			},
		},
		{
			name: "Custom options",
			options: &JSONSerializationOptions{
				Pretty:                true,
				IndentString:          "\t",
				ValidateOnSerialize:   false,
				ValidateOnDeserialize: false,
				SchemaValidation:      true,
				PreserveFieldOrder:    true,
				AllowUnknownFields:    false,
				TimeFormat:            time.RFC822,
				MaxDepth:              50,
				CustomTagName:         "custom",
			},
			validate: func(s *JSONSerializer) error {
				if !s.options.Pretty {
					t.Error("Expected Pretty to be true")
				}
				if s.options.IndentString != "\t" {
					t.Errorf("Expected IndentString to be '\t', got '%s'", s.options.IndentString)
				}
				if s.options.ValidateOnSerialize {
					t.Error("Expected ValidateOnSerialize to be false")
				}
				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			serializer := NewJSONSerializer(test.options)

			if serializer == nil {
				t.Fatal("NewJSONSerializer() returned nil")
			}

			if serializer.GetFormat() != FormatJSON {
				t.Errorf("Expected format JSON, got %v", serializer.GetFormat())
			}

			if test.validate != nil {
				if err := test.validate(serializer); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			}
		})
	}
}

func TestJSONSerializer_Serialize(t *testing.T) {
	ctx := context.Background()
	serializer := NewJSONSerializer(&JSONSerializationOptions{
		ValidateOnSerialize: false, // Disable validation for basic tests
	})

	testMessage := TestMessage{
		ID:        "test-123",
		Name:      "Test Message",
		Value:     42,
		Tags:      []string{"tag1", "tag2"},
		Metadata:  map[string]string{"key1": "value1"},
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	data, err := serializer.Serialize(ctx, testMessage)
	if err != nil {
		t.Fatalf("Serialize() returned error: %v", err)
	}

	// Verify the data is valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Errorf("Serialized data is not valid JSON: %v", err)
	}

	// Check some fields
	if result["id"] != "test-123" {
		t.Errorf("Expected id 'test-123', got %v", result["id"])
	}
	if result["name"] != "Test Message" {
		t.Errorf("Expected name 'Test Message', got %v", result["name"])
	}
	if result["value"] != float64(42) { // JSON numbers are float64
		t.Errorf("Expected value 42, got %v", result["value"])
	}
}

func TestJSONSerializer_SerializePretty(t *testing.T) {
	ctx := context.Background()
	serializer := NewJSONSerializer(&JSONSerializationOptions{
		Pretty:              true,
		IndentString:        "\t",
		ValidateOnSerialize: false,
	})

	testMessage := TestMessage{
		ID:   "test-123",
		Name: "Test Message",
	}

	data, err := serializer.Serialize(ctx, testMessage)
	if err != nil {
		t.Fatalf("Serialize() returned error: %v", err)
	}

	// Check that the output is pretty-printed
	dataString := string(data)
	if !strings.Contains(dataString, "\n") || !strings.Contains(dataString, "\t") {
		t.Error("Expected pretty-printed JSON with newlines and tabs")
	}
}

func TestJSONSerializer_Deserialize(t *testing.T) {
	ctx := context.Background()
	serializer := NewJSONSerializer(&JSONSerializationOptions{
		ValidateOnDeserialize: false,
	})

	jsonData := `{
		"id": "test-123",
		"name": "Test Message",
		"value": 42,
		"tags": ["tag1", "tag2"],
		"metadata": {"key1": "value1"},
		"created_at": "2023-01-01T12:00:00Z"
	}`

	var result TestMessage
	err := serializer.Deserialize(ctx, []byte(jsonData), &result)
	if err != nil {
		t.Fatalf("Deserialize() returned error: %v", err)
	}

	// Verify the deserialized data
	if result.ID != "test-123" {
		t.Errorf("Expected id 'test-123', got %v", result.ID)
	}
	if result.Name != "Test Message" {
		t.Errorf("Expected name 'Test Message', got %v", result.Name)
	}
	if result.Value != 42 {
		t.Errorf("Expected value 42, got %v", result.Value)
	}
	if len(result.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(result.Tags))
	}
	if result.Metadata["key1"] != "value1" {
		t.Errorf("Expected metadata key1='value1', got %v", result.Metadata["key1"])
	}
}

func TestJSONSerializer_SerializeToWriter(t *testing.T) {
	ctx := context.Background()
	serializer := NewJSONSerializer(&JSONSerializationOptions{
		ValidateOnSerialize: false,
	})

	testMessage := TestMessage{
		ID:   "test-123",
		Name: "Test Message",
	}

	var buffer bytes.Buffer
	err := serializer.SerializeToWriter(ctx, testMessage, &buffer)
	if err != nil {
		t.Fatalf("SerializeToWriter() returned error: %v", err)
	}

	// Verify the written data is valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
		t.Errorf("Written data is not valid JSON: %v", err)
	}
}

func TestJSONSerializer_DeserializeFromReader(t *testing.T) {
	ctx := context.Background()
	serializer := NewJSONSerializer(&JSONSerializationOptions{
		ValidateOnDeserialize: false,
	})

	jsonData := `{
		"id": "test-123",
		"name": "Test Message",
		"value": 42
	}`

	reader := strings.NewReader(jsonData)
	var result TestMessage
	err := serializer.DeserializeFromReader(ctx, reader, &result)
	if err != nil {
		t.Fatalf("DeserializeFromReader() returned error: %v", err)
	}

	if result.ID != "test-123" {
		t.Errorf("Expected id 'test-123', got %v", result.ID)
	}
}

func TestJSONSerializer_ValidateMessage(t *testing.T) {
	ctx := context.Background()
	serializer := NewJSONSerializer(nil)

	tests := []struct {
		name      string
		message   interface{}
		expectErr bool
	}{
		{
			name:      "Nil message",
			message:   nil,
			expectErr: true,
		},
		{
			name: "Valid message",
			message: TestMessage{
				ID:        "test-123",
				Name:      "Test Message",
				Value:     42,
				CreatedAt: time.Now(),
			},
			expectErr: false,
		},
		{
			name: "Invalid message - missing required field",
			message: TestMessage{
				Name:      "Test Message",
				Value:     42,
				CreatedAt: time.Now(),
			},
			expectErr: true,
		},
		{
			name: "Invalid message - negative value",
			message: TestMessage{
				ID:        "test-123",
				Name:      "Test Message",
				Value:     -1,
				CreatedAt: time.Now(),
			},
			expectErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := serializer.ValidateMessage(ctx, test.message)

			if test.expectErr && err == nil {
				t.Error("Expected validation error but got nil")
			} else if !test.expectErr && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestJSONSerializer_GetSchema(t *testing.T) {
	ctx := context.Background()
	serializer := NewJSONSerializer(nil)

	schema, err := serializer.GetSchema(ctx)
	if err == nil {
		t.Error("Expected error for GetSchema() on JSON serializer")
	}
	if schema != nil {
		t.Error("Expected nil schema for JSON serializer")
	}
}

func TestJSONSerializer_SetSchema(t *testing.T) {
	serializer := NewJSONSerializer(nil)

	schema := NewJSONSchemaFromDefinition("1.0", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type": "string",
			},
		},
	})

	err := serializer.SetSchema("TestMessage", schema)
	if err != nil {
		t.Errorf("SetSchema() returned error: %v", err)
	}

	// Test invalid inputs
	err = serializer.SetSchema("", schema)
	if err == nil {
		t.Error("Expected error for empty type name")
	}

	err = serializer.SetSchema("TestMessage", nil)
	if err == nil {
		t.Error("Expected error for nil schema")
	}
}

func TestJSONSerializer_Close(t *testing.T) {
	serializer := NewJSONSerializer(nil)

	// Add a schema first
	schema := NewJSONSchemaFromDefinition("1.0", map[string]interface{}{})
	_ = serializer.SetSchema("TestMessage", schema)

	err := serializer.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Verify schema cache is cleared (this is internal, so we can't directly test it)
	// But we can test that Close() doesn't panic or return errors
}

func TestJSONSerializer_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	serializer := NewJSONSerializer(&JSONSerializationOptions{
		ValidateOnSerialize:   false,
		ValidateOnDeserialize: false,
	})

	// Test serialize with nil message
	_, err := serializer.Serialize(ctx, nil)
	if err == nil {
		t.Error("Expected error when serializing nil message")
	}

	// Test deserialize with empty data
	var result TestMessage
	err = serializer.Deserialize(ctx, []byte{}, &result)
	if err == nil {
		t.Error("Expected error when deserializing empty data")
	}

	// Test deserialize with nil target
	err = serializer.Deserialize(ctx, []byte("{}"), nil)
	if err == nil {
		t.Error("Expected error when deserializing to nil target")
	}

	// Test SerializeToWriter with nil writer
	testMessage := TestMessage{ID: "test"}
	err = serializer.SerializeToWriter(ctx, testMessage, nil)
	if err == nil {
		t.Error("Expected error when writing to nil writer")
	}

	// Test DeserializeFromReader with nil reader
	err = serializer.DeserializeFromReader(ctx, nil, &result)
	if err == nil {
		t.Error("Expected error when reading from nil reader")
	}
}

func TestJSONSerializer_ContextCancellation(t *testing.T) {
	serializer := NewJSONSerializer(&JSONSerializationOptions{
		ValidateOnSerialize:   false,
		ValidateOnDeserialize: false,
	})

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	testMessage := TestMessage{ID: "test", Name: "Test"}

	// Test serialize with cancelled context
	_, err := serializer.Serialize(ctx, testMessage)
	if err == nil {
		t.Error("Expected error when serializing with cancelled context")
	}

	// Test deserialize with cancelled context
	var result TestMessage
	err = serializer.Deserialize(ctx, []byte("{}"), &result)
	if err == nil {
		t.Error("Expected error when deserializing with cancelled context")
	}
}

func TestJSONSchema(t *testing.T) {
	definition := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{"type": "string"},
		},
	}

	schema := NewJSONSchemaFromDefinition("1.0", definition)

	if schema.GetVersion() != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", schema.GetVersion())
	}

	def := schema.GetDefinition()
	if def == nil {
		t.Error("GetDefinition() returned nil")
	}

	// Test validation with valid JSON
	ctx := context.Background()
	validJSON := []byte(`{"id": "test"}`)
	err := schema.Validate(ctx, validJSON)
	if err != nil {
		t.Errorf("Validate() returned error for valid JSON: %v", err)
	}

	// Test validation with invalid JSON
	invalidJSON := []byte(`{invalid json`)
	err = schema.Validate(ctx, invalidJSON)
	if err == nil {
		t.Error("Expected validation error for invalid JSON")
	}
}

func TestJSONSchemaBuilder(t *testing.T) {
	schema := NewJSONSchemaBuilder().
		WithVersion("2.0").
		WithType("object").
		WithProperty("id", map[string]interface{}{"type": "string"}).
		WithProperty("name", map[string]interface{}{"type": "string"}).
		WithRequired("id", "name").
		Build()

	if schema.GetVersion() != "2.0" {
		t.Errorf("Expected version '2.0', got '%s'", schema.GetVersion())
	}

	definition := schema.GetDefinition().(map[string]interface{})
	if definition["type"] != "object" {
		t.Errorf("Expected type 'object', got %v", definition["type"])
	}

	properties := definition["properties"].(map[string]interface{})
	if len(properties) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(properties))
	}

	required := definition["required"].([]string)
	if len(required) != 2 {
		t.Errorf("Expected 2 required fields, got %d", len(required))
	}
}

// Benchmarks
func BenchmarkJSONSerializer_Serialize(b *testing.B) {
	ctx := context.Background()
	serializer := NewJSONSerializer(&JSONSerializationOptions{ValidateOnSerialize: false})

	testMessage := TestMessage{
		ID:        "test-123",
		Name:      "Benchmark Test",
		Value:     42,
		Tags:      []string{"tag1", "tag2"},
		CreatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.Serialize(ctx, testMessage)
		if err != nil {
			b.Fatalf("Serialize error: %v", err)
		}
	}
}

func BenchmarkJSONSerializer_Deserialize(b *testing.B) {
	ctx := context.Background()
	serializer := NewJSONSerializer(&JSONSerializationOptions{ValidateOnDeserialize: false})

	jsonData := []byte(`{
		"id": "test-123",
		"name": "Benchmark Test",
		"value": 42,
		"tags": ["tag1", "tag2"],
		"created_at": "2023-01-01T12:00:00Z"
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result TestMessage
		err := serializer.Deserialize(ctx, jsonData, &result)
		if err != nil {
			b.Fatalf("Deserialize error: %v", err)
		}
	}
}
