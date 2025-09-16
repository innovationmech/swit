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
	"io"
	"net/url"
	"sync"
	"time"
)

// SerializationFormat represents the type of serialization format.
type SerializationFormat string

const (
	// FormatJSON represents JSON serialization format.
	FormatJSON SerializationFormat = "json"

	// FormatProtobuf represents Protocol Buffer serialization format.
	FormatProtobuf SerializationFormat = "protobuf"

	// FormatAvro represents Apache Avro serialization format.
	FormatAvro SerializationFormat = "avro"
)

// String returns the string representation of the serialization format.
func (f SerializationFormat) String() string {
	return string(f)
}

// ContentType returns the MIME content type for the serialization format.
func (f SerializationFormat) ContentType() string {
	switch f {
	case FormatJSON:
		return "application/json"
	case FormatProtobuf:
		return "application/x-protobuf"
	case FormatAvro:
		return "application/avro"
	default:
		return "application/octet-stream"
	}
}

// MessageSerializer defines the interface for message serialization operations.
// Implementations must be thread-safe and support concurrent serialization/deserialization.
// All methods should respect context cancellation and timeouts.
type MessageSerializer interface {
	// GetFormat returns the serialization format this serializer handles.
	GetFormat() SerializationFormat

	// Serialize converts a message to its byte representation.
	// The message parameter must be compatible with the serializer format.
	// Returns SerializationError on failure.
	Serialize(ctx context.Context, message interface{}) ([]byte, error)

	// Deserialize converts byte data back to a message object.
	// The target parameter must be a pointer to the expected message type.
	// Returns SerializationError on failure.
	Deserialize(ctx context.Context, data []byte, target interface{}) error

	// SerializeToWriter writes the serialized message directly to an io.Writer.
	// This is useful for streaming scenarios or large messages to avoid buffering.
	// Returns SerializationError on failure.
	SerializeToWriter(ctx context.Context, message interface{}, writer io.Writer) error

	// DeserializeFromReader reads and deserializes a message from an io.Reader.
	// The target parameter must be a pointer to the expected message type.
	// Returns SerializationError on failure.
	DeserializeFromReader(ctx context.Context, reader io.Reader, target interface{}) error

	// ValidateMessage checks if the message is compatible with this serializer.
	// This is useful for pre-serialization validation.
	// Returns ValidationError if the message is incompatible.
	ValidateMessage(ctx context.Context, message interface{}) error

	// GetSchema returns schema information if supported by the format.
	// Returns nil for formats that don't support schemas (like JSON).
	// Returns SchemaError if schema retrieval fails.
	GetSchema(ctx context.Context) (Schema, error)

	// Close releases any resources held by the serializer.
	// After calling Close, the serializer should not be used.
	Close() error
}

// Schema represents schema information for serialization formats that support it.
type Schema interface {
	// GetVersion returns the schema version identifier.
	GetVersion() string

	// GetDefinition returns the schema definition in a format-specific representation.
	GetDefinition() interface{}

	// Validate checks if data conforms to this schema.
	Validate(ctx context.Context, data []byte) error
}

// SerializationOptions provides configuration for serialization operations.
type SerializationOptions struct {
	// Format specifies the target serialization format.
	Format SerializationFormat `json:"format" validate:"required"`

	// Pretty enables pretty-printing for human-readable formats like JSON.
	Pretty bool `json:"pretty,omitempty"`

	// Compress enables compression of serialized data.
	Compress bool `json:"compress,omitempty"`

	// SchemaRegistry specifies schema registry settings for formats that support schemas.
	SchemaRegistry *SchemaRegistryConfig `json:"schema_registry,omitempty"`

	// CustomOptions allows format-specific configuration.
	CustomOptions map[string]interface{} `json:"custom_options,omitempty"`

	// MaxSize specifies the maximum size for serialized data in bytes.
	MaxSize int64 `json:"max_size,omitempty" validate:"min=0"`

	// Timeout specifies the maximum duration for serialization operations.
	Timeout string `json:"timeout,omitempty" validate:"omitempty,duration"`
}

// Validate ensures serialization options are well-formed before use.
func (o *SerializationOptions) Validate() error {
	if o == nil {
		return nil
	}

	switch o.Format {
	case FormatJSON, FormatAvro, FormatProtobuf:
	default:
		return fmt.Errorf("unsupported serialization format: %s", o.Format)
	}

	if o.MaxSize < 0 {
		return fmt.Errorf("max_size must be non-negative")
	}

	if o.Timeout != "" {
		if _, err := time.ParseDuration(o.Timeout); err != nil {
			return fmt.Errorf("invalid timeout duration: %w", err)
		}
	}

	if o.SchemaRegistry != nil {
		if err := o.SchemaRegistry.Validate(); err != nil {
			return fmt.Errorf("schema registry config invalid: %w", err)
		}
	}

	return nil
}

// SchemaRegistryConfig provides configuration for schema registry integration.
type SchemaRegistryConfig struct {
	// URL is the schema registry endpoint URL.
	URL string `json:"url" validate:"required,url"`

	// Authentication provides credentials for accessing the schema registry.
	Authentication *SchemaRegistryAuth `json:"authentication,omitempty"`

	// Subject is the schema subject name.
	Subject string `json:"subject" validate:"required"`

	// Version specifies the schema version to use (latest if empty).
	Version string `json:"version,omitempty"`

	// CacheSize specifies the maximum number of schemas to cache.
	CacheSize int `json:"cache_size,omitempty" validate:"min=0"`
}

// Validate checks schema registry configuration for completeness and consistency.
func (c *SchemaRegistryConfig) Validate() error {
	if c == nil {
		return nil
	}

	if c.URL == "" {
		return fmt.Errorf("schema registry url is required")
	}
	if _, err := url.Parse(c.URL); err != nil {
		return fmt.Errorf("invalid schema registry url: %w", err)
	}

	if c.Subject == "" {
		return fmt.Errorf("schema registry subject is required")
	}

	if c.CacheSize < 0 {
		return fmt.Errorf("schema registry cache size must be non-negative")
	}

	if c.Authentication != nil {
		if err := c.Authentication.Validate(); err != nil {
			return fmt.Errorf("authentication config invalid: %w", err)
		}
	}

	return nil
}

// SchemaRegistryAuth provides authentication credentials for schema registry.
type SchemaRegistryAuth struct {
	// Type specifies the authentication type (basic, bearer, api-key).
	Type string `json:"type" validate:"required,oneof=basic bearer api-key"`

	// Username for basic authentication.
	Username string `json:"username,omitempty"`

	// Password for basic authentication.
	Password string `json:"password,omitempty"`

	// Token for bearer authentication.
	Token string `json:"token,omitempty"`

	// APIKey for API key authentication.
	APIKey string `json:"api_key,omitempty"`

	// Headers provides custom authentication headers.
	Headers map[string]string `json:"headers,omitempty"`
}

// Validate ensures the authentication configuration is complete for the selected type.
func (a *SchemaRegistryAuth) Validate() error {
	if a == nil {
		return nil
	}

	switch a.Type {
	case "basic":
		if a.Username == "" || a.Password == "" {
			return fmt.Errorf("basic auth requires username and password")
		}
	case "bearer":
		if a.Token == "" {
			return fmt.Errorf("bearer auth requires a token")
		}
	case "api-key":
		if a.APIKey == "" {
			return fmt.Errorf("api-key auth requires an api_key value")
		}
	default:
		return fmt.Errorf("unsupported authentication type: %s", a.Type)
	}

	return nil
}

// SerializationRegistry provides a pluggable registry for message serializers.
// It allows registration of different serialization formats and provides
// a unified interface for serialization operations.
type SerializationRegistry interface {
	// RegisterSerializer registers a serializer for a specific format.
	// Returns RegistrationError if the format is already registered or invalid.
	RegisterSerializer(format SerializationFormat, serializer MessageSerializer) error

	// UnregisterSerializer removes a serializer for a specific format.
	// Returns RegistrationError if the format is not registered.
	UnregisterSerializer(format SerializationFormat) error

	// GetSerializer retrieves a serializer for the specified format.
	// Returns SerializationError if no serializer is registered for the format.
	GetSerializer(format SerializationFormat) (MessageSerializer, error)

	// GetSupportedFormats returns a list of all registered serialization formats.
	GetSupportedFormats() []SerializationFormat

	// DetectFormat attempts to detect the serialization format from data.
	// Returns FormatDetectionError if format cannot be determined.
	DetectFormat(ctx context.Context, data []byte) (SerializationFormat, error)

	// CreateSerializer creates a new serializer instance with the given options.
	// This is useful for creating configured serializers on demand.
	CreateSerializer(format SerializationFormat, options *SerializationOptions) (MessageSerializer, error)

	// ValidateFormat checks if a format is supported and properly registered.
	ValidateFormat(format SerializationFormat) error

	// GetMetrics returns serialization metrics for monitoring and debugging.
	GetMetrics() *SerializationMetrics

	// Close closes all registered serializers and releases resources.
	Close() error
}

// SerializationMetrics provides metrics for monitoring serialization operations.
type SerializationMetrics struct {
	// SerializationCount tracks the number of successful serializations by format.
	SerializationCount map[SerializationFormat]int64 `json:"serialization_count"`

	// DeserializationCount tracks the number of successful deserializations by format.
	DeserializationCount map[SerializationFormat]int64 `json:"deserialization_count"`

	// ErrorCount tracks serialization errors by format.
	ErrorCount map[SerializationFormat]int64 `json:"error_count"`

	// AverageSerializationTime tracks average serialization time by format (in milliseconds).
	AverageSerializationTime map[SerializationFormat]float64 `json:"average_serialization_time"`

	// AverageDeserializationTime tracks average deserialization time by format (in milliseconds).
	AverageDeserializationTime map[SerializationFormat]float64 `json:"average_deserialization_time"`

	// TotalBytesProcessed tracks total bytes processed by format.
	TotalBytesProcessed map[SerializationFormat]int64 `json:"total_bytes_processed"`

	// RegisteredFormats lists all currently registered formats.
	RegisteredFormats []SerializationFormat `json:"registered_formats"`
}

// DefaultSerializationRegistry provides a concrete implementation of SerializationRegistry.
type DefaultSerializationRegistry struct {
	serializers map[SerializationFormat]MessageSerializer
	metrics     *SerializationMetrics
	mutex       sync.RWMutex
}

// NewSerializationRegistry creates a new serialization registry with default serializers.
func NewSerializationRegistry() SerializationRegistry {
	registry := &DefaultSerializationRegistry{
		serializers: make(map[SerializationFormat]MessageSerializer),
		metrics: &SerializationMetrics{
			SerializationCount:         make(map[SerializationFormat]int64),
			DeserializationCount:       make(map[SerializationFormat]int64),
			ErrorCount:                 make(map[SerializationFormat]int64),
			AverageSerializationTime:   make(map[SerializationFormat]float64),
			AverageDeserializationTime: make(map[SerializationFormat]float64),
			TotalBytesProcessed:        make(map[SerializationFormat]int64),
			RegisteredFormats:          make([]SerializationFormat, 0),
		},
	}

	return registry
}

// RegisterSerializer implements SerializationRegistry interface.
func (r *DefaultSerializationRegistry) RegisterSerializer(format SerializationFormat, serializer MessageSerializer) error {
	if format == "" {
		return &RegistrationError{
			Format:  format,
			Message: "format cannot be empty",
		}
	}

	if serializer == nil {
		return &RegistrationError{
			Format:  format,
			Message: "serializer cannot be nil",
		}
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.serializers[format]; exists {
		return &RegistrationError{
			Format:  format,
			Message: fmt.Sprintf("serializer for format '%s' already registered", format),
		}
	}

	r.serializers[format] = serializer
	r.metrics.RegisteredFormats = append(r.metrics.RegisteredFormats, format)

	// Initialize metrics for the new format
	r.metrics.SerializationCount[format] = 0
	r.metrics.DeserializationCount[format] = 0
	r.metrics.ErrorCount[format] = 0
	r.metrics.AverageSerializationTime[format] = 0
	r.metrics.AverageDeserializationTime[format] = 0
	r.metrics.TotalBytesProcessed[format] = 0

	return nil
}

// UnregisterSerializer implements SerializationRegistry interface.
func (r *DefaultSerializationRegistry) UnregisterSerializer(format SerializationFormat) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	serializer, exists := r.serializers[format]
	if !exists {
		return &RegistrationError{
			Format:  format,
			Message: fmt.Sprintf("no serializer registered for format '%s'", format),
		}
	}

	// Close the serializer if it implements io.Closer
	if err := serializer.Close(); err != nil {
		return &RegistrationError{
			Format:  format,
			Message: fmt.Sprintf("failed to close serializer for format '%s': %v", format, err),
			Cause:   err,
		}
	}

	delete(r.serializers, format)

	// Remove from registered formats list
	for i, f := range r.metrics.RegisteredFormats {
		if f == format {
			r.metrics.RegisteredFormats = append(
				r.metrics.RegisteredFormats[:i],
				r.metrics.RegisteredFormats[i+1:]...,
			)
			break
		}
	}

	// Clean up metrics
	delete(r.metrics.SerializationCount, format)
	delete(r.metrics.DeserializationCount, format)
	delete(r.metrics.ErrorCount, format)
	delete(r.metrics.AverageSerializationTime, format)
	delete(r.metrics.AverageDeserializationTime, format)
	delete(r.metrics.TotalBytesProcessed, format)

	return nil
}

// GetSerializer implements SerializationRegistry interface.
func (r *DefaultSerializationRegistry) GetSerializer(format SerializationFormat) (MessageSerializer, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	serializer, exists := r.serializers[format]
	if !exists {
		return nil, &SerializationError{
			Format:  format,
			Message: fmt.Sprintf("no serializer registered for format '%s'", format),
		}
	}

	return serializer, nil
}

// GetSupportedFormats implements SerializationRegistry interface.
func (r *DefaultSerializationRegistry) GetSupportedFormats() []SerializationFormat {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	formats := make([]SerializationFormat, len(r.metrics.RegisteredFormats))
	copy(formats, r.metrics.RegisteredFormats)
	return formats
}

// DetectFormat implements SerializationRegistry interface.
func (r *DefaultSerializationRegistry) DetectFormat(ctx context.Context, data []byte) (SerializationFormat, error) {
	if len(data) == 0 {
		return "", &FormatDetectionError{
			Message: "cannot detect format from empty data",
		}
	}

	// Simple heuristic-based format detection
	// This is a basic implementation - more sophisticated detection could be added

	// Check for JSON format
	trimmed := string(data)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		return FormatJSON, nil
	}

	// Check for protobuf format (binary data with certain patterns)
	// This is a simplified check - real implementation would be more robust
	if len(data) > 0 && data[0] < 32 && data[0] != '\n' && data[0] != '\r' && data[0] != '\t' {
		return FormatProtobuf, nil
	}

	return "", &FormatDetectionError{
		Message: "unable to detect serialization format from data",
	}
}

// CreateSerializer implements SerializationRegistry interface.
func (r *DefaultSerializationRegistry) CreateSerializer(format SerializationFormat, options *SerializationOptions) (MessageSerializer, error) {
	r.mutex.RLock()
	baseSerializer, exists := r.serializers[format]
	r.mutex.RUnlock()

	if !exists {
		return nil, &SerializationError{
			Format:  format,
			Message: fmt.Sprintf("no serializer registered for format '%s'", format),
		}
	}

	// For now, return the base serializer
	// In a more advanced implementation, we would create a configured wrapper
	// that applies the provided options
	return baseSerializer, nil
}

// ValidateFormat implements SerializationRegistry interface.
func (r *DefaultSerializationRegistry) ValidateFormat(format SerializationFormat) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if _, exists := r.serializers[format]; !exists {
		return &ValidationError{
			Field:   "format",
			Message: fmt.Sprintf("format '%s' is not supported", format),
		}
	}

	return nil
}

// GetMetrics implements SerializationRegistry interface.
func (r *DefaultSerializationRegistry) GetMetrics() *SerializationMetrics {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Create a deep copy of metrics to prevent external modification
	metrics := &SerializationMetrics{
		SerializationCount:         make(map[SerializationFormat]int64),
		DeserializationCount:       make(map[SerializationFormat]int64),
		ErrorCount:                 make(map[SerializationFormat]int64),
		AverageSerializationTime:   make(map[SerializationFormat]float64),
		AverageDeserializationTime: make(map[SerializationFormat]float64),
		TotalBytesProcessed:        make(map[SerializationFormat]int64),
		RegisteredFormats:          make([]SerializationFormat, len(r.metrics.RegisteredFormats)),
	}

	for format, count := range r.metrics.SerializationCount {
		metrics.SerializationCount[format] = count
	}
	for format, count := range r.metrics.DeserializationCount {
		metrics.DeserializationCount[format] = count
	}
	for format, count := range r.metrics.ErrorCount {
		metrics.ErrorCount[format] = count
	}
	for format, time := range r.metrics.AverageSerializationTime {
		metrics.AverageSerializationTime[format] = time
	}
	for format, time := range r.metrics.AverageDeserializationTime {
		metrics.AverageDeserializationTime[format] = time
	}
	for format, bytes := range r.metrics.TotalBytesProcessed {
		metrics.TotalBytesProcessed[format] = bytes
	}
	copy(metrics.RegisteredFormats, r.metrics.RegisteredFormats)

	return metrics
}

// Close implements SerializationRegistry interface.
func (r *DefaultSerializationRegistry) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var errors []error

	for format, serializer := range r.serializers {
		if err := serializer.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close serializer for format '%s': %w", format, err))
		}
	}

	// Clear all serializers and metrics
	r.serializers = make(map[SerializationFormat]MessageSerializer)
	r.metrics = &SerializationMetrics{
		SerializationCount:         make(map[SerializationFormat]int64),
		DeserializationCount:       make(map[SerializationFormat]int64),
		ErrorCount:                 make(map[SerializationFormat]int64),
		AverageSerializationTime:   make(map[SerializationFormat]float64),
		AverageDeserializationTime: make(map[SerializationFormat]float64),
		TotalBytesProcessed:        make(map[SerializationFormat]int64),
		RegisteredFormats:          make([]SerializationFormat, 0),
	}

	if len(errors) > 0 {
		return &SerializationError{
			Message: fmt.Sprintf("multiple errors occurred during close: %v", errors),
		}
	}

	return nil
}

// SerializationError represents errors that occur during serialization operations.
type SerializationError struct {
	Format  SerializationFormat `json:"format,omitempty"`
	Message string              `json:"message"`
	Cause   error               `json:"-"`
}

// NewSerializationError creates a serialization error without format context.
func NewSerializationError(message string, cause error) *SerializationError {
	return &SerializationError{Message: message, Cause: cause}
}

// NewSerializationErrorWithFormat creates a serialization error for a specific format.
func NewSerializationErrorWithFormat(format SerializationFormat, message string, cause error) *SerializationError {
	return &SerializationError{Format: format, Message: message, Cause: cause}
}

// Error implements the error interface.
func (e *SerializationError) Error() string {
	if e.Format != "" {
		return fmt.Sprintf("serialization error for format '%s': %s", e.Format, e.Message)
	}
	return fmt.Sprintf("serialization error: %s", e.Message)
}

// Unwrap returns the underlying cause of the serialization error.
func (e *SerializationError) Unwrap() error {
	return e.Cause
}

// RegistrationError represents errors that occur during serializer registration.
type RegistrationError struct {
	Format  SerializationFormat `json:"format,omitempty"`
	Message string              `json:"message"`
	Cause   error               `json:"-"`
}

// Error implements the error interface.
func (e *RegistrationError) Error() string {
	if e.Format != "" {
		return fmt.Sprintf("registration error for format '%s': %s", e.Format, e.Message)
	}
	return fmt.Sprintf("registration error: %s", e.Message)
}

// Unwrap returns the underlying cause of the registration error.
func (e *RegistrationError) Unwrap() error {
	return e.Cause
}

// FormatDetectionError represents errors that occur during format detection.
type FormatDetectionError struct {
	Message string `json:"message"`
	Cause   error  `json:"-"`
}

// Error implements the error interface.
func (e *FormatDetectionError) Error() string {
	return fmt.Sprintf("format detection error: %s", e.Message)
}

// Unwrap returns the underlying cause of the format detection error.
func (e *FormatDetectionError) Unwrap() error {
	return e.Cause
}

// SchemaError represents errors that occur during schema operations.
type SchemaError struct {
	Message string `json:"message"`
	Cause   error  `json:"-"`
}

// Error implements the error interface.
func (e *SchemaError) Error() string {
	return fmt.Sprintf("schema error: %s", e.Message)
}

// Unwrap returns the underlying cause of the schema error.
func (e *SchemaError) Unwrap() error {
	return e.Cause
}

// ValidationError represents errors that occur during validation.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Cause   error  `json:"-"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// Unwrap returns the underlying cause of the validation error.
func (e *ValidationError) Unwrap() error {
	return e.Cause
}

// NewDefaultSerializationRegistry creates a new serialization registry with default serializers.
// This includes JSON and Protobuf serializers with standard configurations.
func NewDefaultSerializationRegistry() (SerializationRegistry, error) {
	registry := NewSerializationRegistry()

	// Register JSON serializer with default options
	jsonSerializer := NewJSONSerializer(nil)
	if err := registry.RegisterSerializer(FormatJSON, jsonSerializer); err != nil {
		return nil, &RegistrationError{
			Format:  FormatJSON,
			Message: "failed to register JSON serializer",
			Cause:   err,
		}
	}

	// Register Protobuf serializer with default options
	protobufSerializer := NewProtobufSerializer(nil)
	if err := registry.RegisterSerializer(FormatProtobuf, protobufSerializer); err != nil {
		return nil, &RegistrationError{
			Format:  FormatProtobuf,
			Message: "failed to register Protobuf serializer",
			Cause:   err,
		}
	}

	return registry, nil
}
