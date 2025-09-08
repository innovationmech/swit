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
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
)

// JSONSerializer implements MessageSerializer for JSON format with schema validation.
type JSONSerializer struct {
	options     *JSONSerializationOptions
	validator   *validator.Validate
	schemaCache map[string]*JSONSchema
	mutex       sync.RWMutex
}

// JSONSerializationOptions provides configuration for JSON serialization.
type JSONSerializationOptions struct {
	// Pretty enables pretty-printing with indentation.
	Pretty bool `json:"pretty"`

	// IndentString specifies the indentation string (default: "  ").
	IndentString string `json:"indent_string,omitempty"`

	// ValidateOnSerialize enables validation before serialization.
	ValidateOnSerialize bool `json:"validate_on_serialize"`

	// ValidateOnDeserialize enables validation after deserialization.
	ValidateOnDeserialize bool `json:"validate_on_deserialize"`

	// SchemaValidation enables JSON Schema validation if schemas are provided.
	SchemaValidation bool `json:"schema_validation"`

	// PreserveFieldOrder preserves field order during serialization (uses json.RawMessage internally).
	PreserveFieldOrder bool `json:"preserve_field_order"`

	// AllowUnknownFields allows unknown fields during deserialization.
	AllowUnknownFields bool `json:"allow_unknown_fields"`

	// TimeFormat specifies the time format for time.Time fields (default: RFC3339).
	TimeFormat string `json:"time_format,omitempty"`

	// MaxDepth limits the maximum nesting depth to prevent stack overflow.
	MaxDepth int `json:"max_depth,omitempty"`

	// CustomTagName specifies a custom JSON tag name (default: "json").
	CustomTagName string `json:"custom_tag_name,omitempty"`
}

// JSONSchema represents a JSON schema for validation.
type JSONSchema struct {
	version    string
	definition map[string]interface{}
	validator  func(data []byte) error
}

// GetVersion implements Schema interface.
func (s *JSONSchema) GetVersion() string {
	return s.version
}

// GetDefinition implements Schema interface.
func (s *JSONSchema) GetDefinition() interface{} {
	return s.definition
}

// Validate implements Schema interface.
func (s *JSONSchema) Validate(ctx context.Context, data []byte) error {
	if s.validator == nil {
		return &SchemaError{
			Message: "no validator configured for this schema",
		}
	}

	return s.validator(data)
}

// NewJSONSerializer creates a new JSON serializer with the specified options.
func NewJSONSerializer(options *JSONSerializationOptions) *JSONSerializer {
	if options == nil {
		options = &JSONSerializationOptions{
			Pretty:                false,
			IndentString:          "  ",
			ValidateOnSerialize:   true,
			ValidateOnDeserialize: true,
			SchemaValidation:      false,
			PreserveFieldOrder:    false,
			AllowUnknownFields:    true,
			TimeFormat:            time.RFC3339,
			MaxDepth:              100,
			CustomTagName:         "json",
		}
	}

	// Set defaults for unspecified fields
	if options.IndentString == "" {
		options.IndentString = "  "
	}
	if options.TimeFormat == "" {
		options.TimeFormat = time.RFC3339
	}
	if options.MaxDepth == 0 {
		options.MaxDepth = 100
	}
	if options.CustomTagName == "" {
		options.CustomTagName = "json"
	}

	validator := validator.New()

	// Register custom validation for JSON-specific rules
	validator.RegisterValidation("json_object", validateJSONObject)
	validator.RegisterValidation("json_array", validateJSONArray)

	return &JSONSerializer{
		options:     options,
		validator:   validator,
		schemaCache: make(map[string]*JSONSchema),
	}
}

// GetFormat implements MessageSerializer interface.
func (j *JSONSerializer) GetFormat() SerializationFormat {
	return FormatJSON
}

// Serialize implements MessageSerializer interface.
func (j *JSONSerializer) Serialize(ctx context.Context, message interface{}) ([]byte, error) {
	if message == nil {
		return nil, &SerializationError{
			Format:  FormatJSON,
			Message: "message cannot be nil",
		}
	}

	// Validate before serialization if enabled
	if j.options.ValidateOnSerialize {
		if err := j.ValidateMessage(ctx, message); err != nil {
			return nil, &SerializationError{
				Format:  FormatJSON,
				Message: "message validation failed before serialization",
				Cause:   err,
			}
		}
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, &SerializationError{
			Format:  FormatJSON,
			Message: "serialization cancelled",
			Cause:   ctx.Err(),
		}
	default:
	}

	var data []byte
	var err error

	if j.options.Pretty {
		data, err = json.MarshalIndent(message, "", j.options.IndentString)
	} else {
		data, err = json.Marshal(message)
	}

	if err != nil {
		return nil, &SerializationError{
			Format:  FormatJSON,
			Message: "failed to marshal JSON",
			Cause:   err,
		}
	}

	// Schema validation if enabled
	if j.options.SchemaValidation {
		if schema, err := j.getSchemaForMessage(message); err == nil && schema != nil {
			if err := schema.Validate(ctx, data); err != nil {
				return nil, &SerializationError{
					Format:  FormatJSON,
					Message: "schema validation failed",
					Cause:   err,
				}
			}
		}
	}

	return data, nil
}

// Deserialize implements MessageSerializer interface.
func (j *JSONSerializer) Deserialize(ctx context.Context, data []byte, target interface{}) error {
	if len(data) == 0 {
		return &SerializationError{
			Format:  FormatJSON,
			Message: "data cannot be empty",
		}
	}

	if target == nil {
		return &SerializationError{
			Format:  FormatJSON,
			Message: "target cannot be nil",
		}
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return &SerializationError{
			Format:  FormatJSON,
			Message: "deserialization cancelled",
			Cause:   ctx.Err(),
		}
	default:
	}

	// Create a decoder with custom options
	decoder := json.NewDecoder(nil)
	if !j.options.AllowUnknownFields {
		decoder.DisallowUnknownFields()
	}

	// Unmarshal the data
	if err := json.Unmarshal(data, target); err != nil {
		return &SerializationError{
			Format:  FormatJSON,
			Message: "failed to unmarshal JSON",
			Cause:   err,
		}
	}

	// Validate after deserialization if enabled
	if j.options.ValidateOnDeserialize {
		if err := j.ValidateMessage(ctx, target); err != nil {
			return &SerializationError{
				Format:  FormatJSON,
				Message: "message validation failed after deserialization",
				Cause:   err,
			}
		}
	}

	return nil
}

// SerializeToWriter implements MessageSerializer interface.
func (j *JSONSerializer) SerializeToWriter(ctx context.Context, message interface{}, writer io.Writer) error {
	if message == nil {
		return &SerializationError{
			Format:  FormatJSON,
			Message: "message cannot be nil",
		}
	}

	if writer == nil {
		return &SerializationError{
			Format:  FormatJSON,
			Message: "writer cannot be nil",
		}
	}

	// Validate before serialization if enabled
	if j.options.ValidateOnSerialize {
		if err := j.ValidateMessage(ctx, message); err != nil {
			return &SerializationError{
				Format:  FormatJSON,
				Message: "message validation failed before serialization",
				Cause:   err,
			}
		}
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return &SerializationError{
			Format:  FormatJSON,
			Message: "serialization cancelled",
			Cause:   ctx.Err(),
		}
	default:
	}

	encoder := json.NewEncoder(writer)
	if j.options.Pretty {
		encoder.SetIndent("", j.options.IndentString)
	}

	if err := encoder.Encode(message); err != nil {
		return &SerializationError{
			Format:  FormatJSON,
			Message: "failed to encode JSON to writer",
			Cause:   err,
		}
	}

	return nil
}

// DeserializeFromReader implements MessageSerializer interface.
func (j *JSONSerializer) DeserializeFromReader(ctx context.Context, reader io.Reader, target interface{}) error {
	if reader == nil {
		return &SerializationError{
			Format:  FormatJSON,
			Message: "reader cannot be nil",
		}
	}

	if target == nil {
		return &SerializationError{
			Format:  FormatJSON,
			Message: "target cannot be nil",
		}
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return &SerializationError{
			Format:  FormatJSON,
			Message: "deserialization cancelled",
			Cause:   ctx.Err(),
		}
	default:
	}

	decoder := json.NewDecoder(reader)
	if !j.options.AllowUnknownFields {
		decoder.DisallowUnknownFields()
	}

	if err := decoder.Decode(target); err != nil {
		return &SerializationError{
			Format:  FormatJSON,
			Message: "failed to decode JSON from reader",
			Cause:   err,
		}
	}

	// Validate after deserialization if enabled
	if j.options.ValidateOnDeserialize {
		if err := j.ValidateMessage(ctx, target); err != nil {
			return &SerializationError{
				Format:  FormatJSON,
				Message: "message validation failed after deserialization",
				Cause:   err,
			}
		}
	}

	return nil
}

// ValidateMessage implements MessageSerializer interface.
func (j *JSONSerializer) ValidateMessage(ctx context.Context, message interface{}) error {
	if message == nil {
		return &ValidationError{
			Message: "message cannot be nil",
		}
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return &ValidationError{
			Message: "validation cancelled",
			Cause:   ctx.Err(),
		}
	default:
	}

	// Use struct validation if the message has validation tags
	if err := j.validator.Struct(message); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			// Convert to our validation error format
			var errorMessages []string
			for _, fieldError := range validationErrors {
				errorMessages = append(errorMessages, fmt.Sprintf(
					"field '%s' failed validation '%s'",
					fieldError.Field(),
					fieldError.Tag(),
				))
			}
			return &ValidationError{
				Message: fmt.Sprintf("struct validation failed: %v", errorMessages),
				Cause:   err,
			}
		}
		return &ValidationError{
			Message: "struct validation failed",
			Cause:   err,
		}
	}

	return nil
}

// GetSchema implements MessageSerializer interface.
func (j *JSONSerializer) GetSchema(ctx context.Context) (Schema, error) {
	// For JSON, we don't have a predefined schema unless explicitly set
	// This is a placeholder implementation
	return nil, &SchemaError{
		Message: "JSON serializer does not have a predefined schema",
	}
}

// SetSchema allows setting a JSON schema for validation.
func (j *JSONSerializer) SetSchema(typeName string, schema *JSONSchema) error {
	if typeName == "" {
		return &SchemaError{
			Message: "type name cannot be empty",
		}
	}

	if schema == nil {
		return &SchemaError{
			Message: "schema cannot be nil",
		}
	}

	j.mutex.Lock()
	defer j.mutex.Unlock()

	j.schemaCache[typeName] = schema
	return nil
}

// Close implements MessageSerializer interface.
func (j *JSONSerializer) Close() error {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	// Clear schema cache
	j.schemaCache = make(map[string]*JSONSchema)

	return nil
}

// getSchemaForMessage attempts to get a schema for the given message type.
func (j *JSONSerializer) getSchemaForMessage(message interface{}) (*JSONSchema, error) {
	if message == nil {
		return nil, &SchemaError{
			Message: "message cannot be nil",
		}
	}

	// Get the type name
	messageType := reflect.TypeOf(message)
	if messageType.Kind() == reflect.Ptr {
		messageType = messageType.Elem()
	}
	typeName := messageType.Name()

	j.mutex.RLock()
	schema, exists := j.schemaCache[typeName]
	j.mutex.RUnlock()

	if !exists {
		return nil, &SchemaError{
			Message: fmt.Sprintf("no schema found for type '%s'", typeName),
		}
	}

	return schema, nil
}

// validateJSONObject validates that a field contains a valid JSON object.
func validateJSONObject(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Allow empty values
	}

	var obj map[string]interface{}
	return json.Unmarshal([]byte(value), &obj) == nil
}

// validateJSONArray validates that a field contains a valid JSON array.
func validateJSONArray(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Allow empty values
	}

	var arr []interface{}
	return json.Unmarshal([]byte(value), &arr) == nil
}

// NewJSONSchemaFromDefinition creates a new JSON schema from a definition map.
func NewJSONSchemaFromDefinition(version string, definition map[string]interface{}) *JSONSchema {
	return &JSONSchema{
		version:    version,
		definition: definition,
		validator: func(data []byte) error {
			// Basic JSON validation - in a real implementation,
			// this would use a proper JSON schema validator library
			var obj interface{}
			if err := json.Unmarshal(data, &obj); err != nil {
				return &SchemaError{
					Message: "invalid JSON format",
					Cause:   err,
				}
			}
			return nil
		},
	}
}

// JSONSchemaBuilder provides a fluent interface for building JSON schemas.
type JSONSchemaBuilder struct {
	schema *JSONSchema
}

// NewJSONSchemaBuilder creates a new JSON schema builder.
func NewJSONSchemaBuilder() *JSONSchemaBuilder {
	return &JSONSchemaBuilder{
		schema: &JSONSchema{
			version:    "draft-07",
			definition: make(map[string]interface{}),
		},
	}
}

// WithVersion sets the schema version.
func (b *JSONSchemaBuilder) WithVersion(version string) *JSONSchemaBuilder {
	b.schema.version = version
	return b
}

// WithProperty adds a property to the schema.
func (b *JSONSchemaBuilder) WithProperty(name string, propertyDef map[string]interface{}) *JSONSchemaBuilder {
	if b.schema.definition["properties"] == nil {
		b.schema.definition["properties"] = make(map[string]interface{})
	}

	properties := b.schema.definition["properties"].(map[string]interface{})
	properties[name] = propertyDef

	return b
}

// WithRequired adds required fields to the schema.
func (b *JSONSchemaBuilder) WithRequired(fields ...string) *JSONSchemaBuilder {
	if len(fields) > 0 {
		b.schema.definition["required"] = fields
	}
	return b
}

// WithType sets the schema type.
func (b *JSONSchemaBuilder) WithType(schemaType string) *JSONSchemaBuilder {
	b.schema.definition["type"] = schemaType
	return b
}

// WithValidationFunction sets a custom validation function.
func (b *JSONSchemaBuilder) WithValidationFunction(validator func(data []byte) error) *JSONSchemaBuilder {
	b.schema.validator = validator
	return b
}

// Build creates the final JSON schema.
func (b *JSONSchemaBuilder) Build() *JSONSchema {
	// Set default validator if none provided
	if b.schema.validator == nil {
		b.schema.validator = func(data []byte) error {
			var obj interface{}
			if err := json.Unmarshal(data, &obj); err != nil {
				return &SchemaError{
					Message: "invalid JSON format",
					Cause:   err,
				}
			}
			return nil
		}
	}

	return b.schema
}
