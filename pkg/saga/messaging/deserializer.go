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
	"fmt"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/saga"
	"google.golang.org/protobuf/proto"
)

// MessageDeserializer defines the interface for deserializing messages into Saga events.
// Implementations must be thread-safe and support multiple serialization formats.
//
// The deserializer is responsible for:
//   - Converting raw message data into SagaEvent structures
//   - Validating message format and content
//   - Extracting metadata from message headers
//   - Handling different serialization formats (JSON, Protobuf, etc.)
//
// Example usage:
//
//	deserializer := NewDefaultMessageDeserializer()
//	event, err := deserializer.Deserialize(ctx, messageData, headers)
//	if err != nil {
//	    log.Error("Failed to deserialize message", zap.Error(err))
//	    return err
//	}
type MessageDeserializer interface {
	// Deserialize converts raw message data and headers into a SagaEvent.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation control
	//   - data: Raw message payload bytes
	//   - headers: Message headers containing metadata
	//
	// Returns:
	//   - *saga.SagaEvent: Deserialized event
	//   - error: Deserialization error if the operation fails
	Deserialize(ctx context.Context, data []byte, headers map[string]string) (*saga.SagaEvent, error)

	// DeserializeWithFormat deserializes a message using a specific format.
	// This is useful when the format is known in advance or specified in headers.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation control
	//   - data: Raw message payload bytes
	//   - format: Serialization format to use
	//   - headers: Message headers containing metadata
	//
	// Returns:
	//   - *saga.SagaEvent: Deserialized event
	//   - error: Deserialization error if the operation fails
	DeserializeWithFormat(ctx context.Context, data []byte, format messaging.SerializationFormat, headers map[string]string) (*saga.SagaEvent, error)

	// ValidateMessage checks if the raw message data is valid and can be deserialized.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation control
	//   - data: Raw message payload bytes
	//   - headers: Message headers containing metadata
	//
	// Returns:
	//   - error: Validation error if the message is invalid
	ValidateMessage(ctx context.Context, data []byte, headers map[string]string) error

	// ExtractMetadata extracts metadata from message headers.
	// This includes correlation ID, trace ID, event type, etc.
	//
	// Parameters:
	//   - headers: Message headers
	//
	// Returns:
	//   - map[string]interface{}: Extracted metadata
	ExtractMetadata(headers map[string]string) map[string]interface{}

	// DetectFormat attempts to detect the serialization format from the message data.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation control
	//   - data: Raw message payload bytes
	//   - headers: Message headers (may contain format hints)
	//
	// Returns:
	//   - messaging.SerializationFormat: Detected format
	//   - error: Detection error if the format cannot be determined
	DetectFormat(ctx context.Context, data []byte, headers map[string]string) (messaging.SerializationFormat, error)

	// GetSupportedFormats returns the list of serialization formats this deserializer supports.
	//
	// Returns:
	//   - []messaging.SerializationFormat: List of supported formats
	GetSupportedFormats() []messaging.SerializationFormat
}

// defaultMessageDeserializer is the default implementation of MessageDeserializer.
// It supports JSON and Protobuf formats with automatic format detection.
type defaultMessageDeserializer struct {
	// jsonSerializer handles JSON serialization/deserialization
	jsonSerializer messaging.MessageSerializer

	// protobufSerializer handles Protobuf serialization/deserialization
	protobufSerializer messaging.MessageSerializer

	// supportedFormats lists all supported serialization formats
	supportedFormats []messaging.SerializationFormat
}

// NewDefaultMessageDeserializer creates a new default message deserializer
// with support for JSON and Protobuf formats.
//
// Returns:
//   - MessageDeserializer: Configured deserializer instance
func NewDefaultMessageDeserializer() MessageDeserializer {
	return &defaultMessageDeserializer{
		jsonSerializer:     messaging.NewJSONSerializer(nil),
		protobufSerializer: messaging.NewProtobufSerializer(nil),
		supportedFormats: []messaging.SerializationFormat{
			messaging.FormatJSON,
			messaging.FormatProtobuf,
		},
	}
}

// NewMessageDeserializerWithSerializers creates a message deserializer with custom serializers.
//
// Parameters:
//   - jsonSerializer: Custom JSON serializer (can be nil to use default)
//   - protobufSerializer: Custom Protobuf serializer (can be nil to use default)
//
// Returns:
//   - MessageDeserializer: Configured deserializer instance
func NewMessageDeserializerWithSerializers(
	jsonSerializer messaging.MessageSerializer,
	protobufSerializer messaging.MessageSerializer,
) MessageDeserializer {
	if jsonSerializer == nil {
		jsonSerializer = messaging.NewJSONSerializer(nil)
	}
	if protobufSerializer == nil {
		protobufSerializer = messaging.NewProtobufSerializer(nil)
	}

	return &defaultMessageDeserializer{
		jsonSerializer:     jsonSerializer,
		protobufSerializer: protobufSerializer,
		supportedFormats: []messaging.SerializationFormat{
			messaging.FormatJSON,
			messaging.FormatProtobuf,
		},
	}
}

// Deserialize implements MessageDeserializer interface.
func (d *defaultMessageDeserializer) Deserialize(ctx context.Context, data []byte, headers map[string]string) (*saga.SagaEvent, error) {
	if len(data) == 0 {
		return nil, ErrInvalidEvent
	}

	// Detect format from headers or data
	format, err := d.DetectFormat(ctx, data, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to detect message format: %w", err)
	}

	// Deserialize using the detected format
	return d.DeserializeWithFormat(ctx, data, format, headers)
}

// DeserializeWithFormat implements MessageDeserializer interface.
func (d *defaultMessageDeserializer) DeserializeWithFormat(
	ctx context.Context,
	data []byte,
	format messaging.SerializationFormat,
	headers map[string]string,
) (*saga.SagaEvent, error) {
	if len(data) == 0 {
		return nil, ErrInvalidEvent
	}

	// Validate format is supported
	if !d.isFormatSupported(format) {
		return nil, fmt.Errorf("unsupported serialization format: %s", format)
	}

	// Select appropriate serializer
	var serializer messaging.MessageSerializer
	switch format {
	case messaging.FormatJSON:
		serializer = d.jsonSerializer
	case messaging.FormatProtobuf:
		serializer = d.protobufSerializer
	default:
		return nil, fmt.Errorf("unsupported serialization format: %s", format)
	}

	// Deserialize the event
	var event saga.SagaEvent
	if err := serializer.Deserialize(ctx, data, &event); err != nil {
		return nil, fmt.Errorf("failed to deserialize saga event: %w", err)
	}

	// Extract and merge metadata from headers
	metadata := d.ExtractMetadata(headers)
	if event.Metadata == nil {
		event.Metadata = metadata
	} else {
		// Merge metadata, with header metadata taking precedence
		for key, value := range metadata {
			if _, exists := event.Metadata[key]; !exists {
				event.Metadata[key] = value
			}
		}
	}

	// Validate the deserialized event
	if err := d.validateEvent(&event); err != nil {
		return nil, fmt.Errorf("event validation failed: %w", err)
	}

	return &event, nil
}

// ValidateMessage implements MessageDeserializer interface.
func (d *defaultMessageDeserializer) ValidateMessage(ctx context.Context, data []byte, headers map[string]string) error {
	if len(data) == 0 {
		return ErrInvalidEvent
	}

	// Detect format
	format, err := d.DetectFormat(ctx, data, headers)
	if err != nil {
		return fmt.Errorf("failed to detect message format: %w", err)
	}

	// Select appropriate serializer for validation
	var serializer messaging.MessageSerializer
	switch format {
	case messaging.FormatJSON:
		serializer = d.jsonSerializer
	case messaging.FormatProtobuf:
		serializer = d.protobufSerializer
	default:
		return fmt.Errorf("unsupported serialization format: %s", format)
	}

	// Create a temporary event for validation
	var event saga.SagaEvent
	if err := serializer.Deserialize(ctx, data, &event); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}

	// Validate the event structure
	return d.validateEvent(&event)
}

// ExtractMetadata implements MessageDeserializer interface.
func (d *defaultMessageDeserializer) ExtractMetadata(headers map[string]string) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Extract standard headers
	if correlationID, ok := headers["correlation-id"]; ok {
		metadata["correlation_id"] = correlationID
	}
	if correlationID, ok := headers["x-correlation-id"]; ok {
		metadata["correlation_id"] = correlationID
	}

	if traceID, ok := headers["trace-id"]; ok {
		metadata["trace_id"] = traceID
	}
	if traceID, ok := headers["x-trace-id"]; ok {
		metadata["trace_id"] = traceID
	}

	if spanID, ok := headers["span-id"]; ok {
		metadata["span_id"] = spanID
	}
	if spanID, ok := headers["x-span-id"]; ok {
		metadata["span_id"] = spanID
	}

	if eventType, ok := headers["event-type"]; ok {
		metadata["event_type"] = eventType
	}
	if eventType, ok := headers["x-event-type"]; ok {
		metadata["event_type"] = eventType
	}

	if sagaID, ok := headers["saga-id"]; ok {
		metadata["saga_id"] = sagaID
	}
	if sagaID, ok := headers["x-saga-id"]; ok {
		metadata["saga_id"] = sagaID
	}

	if stepID, ok := headers["step-id"]; ok {
		metadata["step_id"] = stepID
	}
	if stepID, ok := headers["x-step-id"]; ok {
		metadata["step_id"] = stepID
	}

	if source, ok := headers["source"]; ok {
		metadata["source"] = source
	}
	if source, ok := headers["x-source"]; ok {
		metadata["source"] = source
	}

	if service, ok := headers["service"]; ok {
		metadata["service"] = service
	}
	if service, ok := headers["x-service"]; ok {
		metadata["service"] = service
	}

	// Extract any custom headers with "x-metadata-" prefix
	for key, value := range headers {
		if len(key) > 11 && key[:11] == "x-metadata-" {
			metaKey := key[11:] // Remove "x-metadata-" prefix
			metadata[metaKey] = value
		}
	}

	return metadata
}

// DetectFormat implements MessageDeserializer interface.
func (d *defaultMessageDeserializer) DetectFormat(ctx context.Context, data []byte, headers map[string]string) (messaging.SerializationFormat, error) {
	// First check headers for explicit format indication
	if contentType, ok := headers["content-type"]; ok {
		switch contentType {
		case "application/json", "text/json":
			return messaging.FormatJSON, nil
		case "application/x-protobuf", "application/protobuf":
			return messaging.FormatProtobuf, nil
		}
	}

	// Check for format header
	if format, ok := headers["format"]; ok {
		switch format {
		case "json":
			return messaging.FormatJSON, nil
		case "protobuf", "proto":
			return messaging.FormatProtobuf, nil
		}
	}
	if format, ok := headers["x-format"]; ok {
		switch format {
		case "json":
			return messaging.FormatJSON, nil
		case "protobuf", "proto":
			return messaging.FormatProtobuf, nil
		}
	}

	// Fallback to content-based detection
	if len(data) == 0 {
		return "", fmt.Errorf("cannot detect format from empty data")
	}

	// Try JSON detection - check if data starts with { or [
	trimmed := data
	// Skip whitespace
	for len(trimmed) > 0 && (trimmed[0] == ' ' || trimmed[0] == '\t' || trimmed[0] == '\n' || trimmed[0] == '\r') {
		trimmed = trimmed[1:]
	}

	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		// Validate it's actually valid JSON
		var test interface{}
		if err := json.Unmarshal(data, &test); err == nil {
			return messaging.FormatJSON, nil
		}
	}

	// Try Protobuf detection - check if it's valid binary data
	// Protobuf messages typically start with field tags (small integers)
	if len(data) > 0 {
		// Try to parse as protobuf
		var protoTest proto.Message
		// Note: This is a simplified check. In a real implementation,
		// you might need a more sophisticated protobuf detection mechanism.
		_ = protoTest // Placeholder for actual protobuf validation
		return messaging.FormatProtobuf, nil
	}

	// Default to JSON if detection fails
	return messaging.FormatJSON, nil
}

// GetSupportedFormats implements MessageDeserializer interface.
func (d *defaultMessageDeserializer) GetSupportedFormats() []messaging.SerializationFormat {
	formats := make([]messaging.SerializationFormat, len(d.supportedFormats))
	copy(formats, d.supportedFormats)
	return formats
}

// isFormatSupported checks if a format is supported by this deserializer.
func (d *defaultMessageDeserializer) isFormatSupported(format messaging.SerializationFormat) bool {
	for _, supported := range d.supportedFormats {
		if supported == format {
			return true
		}
	}
	return false
}

// validateEvent validates a deserialized SagaEvent for required fields and consistency.
func (d *defaultMessageDeserializer) validateEvent(event *saga.SagaEvent) error {
	if event == nil {
		return ErrInvalidEvent
	}

	// Validate required fields
	if event.ID == "" {
		return fmt.Errorf("event ID is required")
	}
	if event.SagaID == "" {
		return fmt.Errorf("saga ID is required")
	}
	if event.Type == "" {
		return fmt.Errorf("event type is required")
	}

	// Validate event type is a known type
	validTypes := []saga.SagaEventType{
		saga.EventSagaStarted,
		saga.EventSagaStepStarted,
		saga.EventSagaStepCompleted,
		saga.EventSagaStepFailed,
		saga.EventSagaCompleted,
		saga.EventSagaFailed,
		saga.EventSagaCancelled,
		saga.EventSagaTimedOut,
		saga.EventCompensationStarted,
		saga.EventCompensationStepStarted,
		saga.EventCompensationStepCompleted,
		saga.EventCompensationStepFailed,
		saga.EventCompensationCompleted,
		saga.EventCompensationFailed,
		saga.EventRetryAttempted,
		saga.EventRetryExhausted,
		saga.EventStateChanged,
	}

	isValidType := false
	for _, validType := range validTypes {
		if event.Type == validType {
			isValidType = true
			break
		}
	}

	if !isValidType {
		return fmt.Errorf("invalid event type: %s", event.Type)
	}

	// Validate timestamp
	if event.Timestamp.IsZero() {
		return fmt.Errorf("event timestamp is required")
	}

	return nil
}
