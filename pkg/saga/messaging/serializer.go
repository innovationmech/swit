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
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbsaga "github.com/innovationmech/swit/api/gen/go/proto/swit/saga/v1"
)

// SagaEventSerializer defines the interface for serializing Saga events.
// Implementations must be thread-safe and support multiple serialization formats.
//
// The serializer is responsible for:
//   - Converting SagaEvent structures into byte representations
//   - Supporting different serialization formats (JSON, Protobuf, etc.)
//   - Handling version compatibility
//   - Providing efficient serialization performance
//
// Example usage:
//
//	serializer := NewJSONSagaEventSerializer()
//	data, err := serializer.Serialize(ctx, event)
//	if err != nil {
//	    log.Error("Failed to serialize event", zap.Error(err))
//	    return err
//	}
type SagaEventSerializer interface {
	// Serialize converts a SagaEvent into its byte representation.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation control
	//   - event: SagaEvent to serialize
	//
	// Returns:
	//   - []byte: Serialized event data
	//   - error: Serialization error if the operation fails
	Serialize(ctx context.Context, event *saga.SagaEvent) ([]byte, error)

	// Deserialize converts byte data back into a SagaEvent.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation control
	//   - data: Serialized event data
	//
	// Returns:
	//   - *saga.SagaEvent: Deserialized event
	//   - error: Deserialization error if the operation fails
	Deserialize(ctx context.Context, data []byte) (*saga.SagaEvent, error)

	// ContentType returns the MIME type for this serialization format.
	//
	// Returns:
	//   - string: MIME content type (e.g., "application/json")
	ContentType() string

	// FormatName returns the name of the serialization format.
	//
	// Returns:
	//   - string: Format name (e.g., "json", "protobuf")
	FormatName() string

	// Validate checks if an event is valid and can be serialized.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation control
	//   - event: Event to validate
	//
	// Returns:
	//   - error: Validation error if the event is invalid
	Validate(ctx context.Context, event *saga.SagaEvent) error
}

// JSONSagaEventSerializer implements SagaEventSerializer using JSON format.
type JSONSagaEventSerializer struct {
	// pretty enables pretty-printing for human readability
	pretty bool

	// version tracks the serialization version for compatibility
	version string
}

// NewJSONSagaEventSerializer creates a new JSON serializer with default options.
//
// Returns:
//   - SagaEventSerializer: Configured JSON serializer instance
func NewJSONSagaEventSerializer() SagaEventSerializer {
	return &JSONSagaEventSerializer{
		pretty:  false,
		version: "1.0",
	}
}

// NewJSONSagaEventSerializerWithOptions creates a JSON serializer with custom options.
//
// Parameters:
//   - pretty: Enable pretty-printing for human readability
//   - version: Serialization version string
//
// Returns:
//   - SagaEventSerializer: Configured JSON serializer instance
func NewJSONSagaEventSerializerWithOptions(pretty bool, version string) SagaEventSerializer {
	if version == "" {
		version = "1.0"
	}
	return &JSONSagaEventSerializer{
		pretty:  pretty,
		version: version,
	}
}

// Serialize implements SagaEventSerializer interface for JSON format.
func (s *JSONSagaEventSerializer) Serialize(ctx context.Context, event *saga.SagaEvent) ([]byte, error) {
	if event == nil {
		return nil, ErrInvalidEvent
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("serialization cancelled: %w", ctx.Err())
	default:
	}

	// Validate event before serialization
	if err := s.Validate(ctx, event); err != nil {
		return nil, fmt.Errorf("event validation failed: %w", err)
	}

	// Create a versioned wrapper for compatibility
	wrapper := &jsonEventWrapper{
		Version: s.version,
		Event:   event,
	}

	var data []byte
	var err error

	if s.pretty {
		data, err = json.MarshalIndent(wrapper, "", "  ")
	} else {
		data, err = json.Marshal(wrapper)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal saga event: %w", err)
	}

	return data, nil
}

// Deserialize implements SagaEventSerializer interface for JSON format.
func (s *JSONSagaEventSerializer) Deserialize(ctx context.Context, data []byte) (*saga.SagaEvent, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("cannot deserialize empty data")
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("deserialization cancelled: %w", ctx.Err())
	default:
	}

	// Try to deserialize as versioned wrapper first
	var wrapper jsonEventWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		// Fallback: try to deserialize directly as SagaEvent (for backward compatibility)
		var event saga.SagaEvent
		if err2 := json.Unmarshal(data, &event); err2 != nil {
			return nil, fmt.Errorf("failed to unmarshal saga event: wrapper error: %w, direct error: %v", err, err2)
		}
		// Validate the directly deserialized event
		if err := s.Validate(ctx, &event); err != nil {
			return nil, fmt.Errorf("deserialized event validation failed: %w", err)
		}
		return &event, nil
	}

	// Check if it's actually a wrapper or just a direct event
	if wrapper.Event == nil {
		// It's a direct event, not a wrapper
		var event saga.SagaEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal direct saga event: %w", err)
		}
		// Validate the directly deserialized event
		if err := s.Validate(ctx, &event); err != nil {
			return nil, fmt.Errorf("deserialized event validation failed: %w", err)
		}
		return &event, nil
	}

	// Validate version compatibility
	if err := s.validateVersion(wrapper.Version); err != nil {
		return nil, fmt.Errorf("version compatibility check failed: %w", err)
	}

	// Validate the deserialized event
	if err := s.Validate(ctx, wrapper.Event); err != nil {
		return nil, fmt.Errorf("deserialized event validation failed: %w", err)
	}

	return wrapper.Event, nil
}

// ContentType implements SagaEventSerializer interface.
func (s *JSONSagaEventSerializer) ContentType() string {
	return "application/json"
}

// FormatName implements SagaEventSerializer interface.
func (s *JSONSagaEventSerializer) FormatName() string {
	return "json"
}

// Validate implements SagaEventSerializer interface.
func (s *JSONSagaEventSerializer) Validate(ctx context.Context, event *saga.SagaEvent) error {
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

	// Validate timestamp
	if event.Timestamp.IsZero() {
		return fmt.Errorf("event timestamp is required")
	}

	// Validate version if present
	if event.Version != "" {
		if err := s.validateVersion(event.Version); err != nil {
			return fmt.Errorf("invalid event version: %w", err)
		}
	}

	return nil
}

// validateVersion checks if a version string is compatible.
func (s *JSONSagaEventSerializer) validateVersion(version string) error {
	if version == "" {
		return nil // Empty version is acceptable (default)
	}

	// For now, accept version 1.x
	// In a more sophisticated implementation, we could use semantic versioning
	if version[0] != '1' {
		return fmt.Errorf("unsupported version: %s (expected 1.x)", version)
	}

	return nil
}

// jsonEventWrapper wraps a SagaEvent with version information for forward/backward compatibility.
type jsonEventWrapper struct {
	Version string           `json:"version"`
	Event   *saga.SagaEvent  `json:"event"`
}

// ProtobufSagaEventSerializer implements SagaEventSerializer using Protocol Buffers format.
type ProtobufSagaEventSerializer struct {
	// version tracks the serialization version for compatibility
	version string
}

// NewProtobufSagaEventSerializer creates a new Protobuf serializer with default options.
//
// Returns:
//   - SagaEventSerializer: Configured Protobuf serializer instance
func NewProtobufSagaEventSerializer() SagaEventSerializer {
	return &ProtobufSagaEventSerializer{
		version: "1.0",
	}
}

// NewProtobufSagaEventSerializerWithOptions creates a Protobuf serializer with custom options.
//
// Parameters:
//   - version: Serialization version string
//
// Returns:
//   - SagaEventSerializer: Configured Protobuf serializer instance
func NewProtobufSagaEventSerializerWithOptions(version string) SagaEventSerializer {
	if version == "" {
		version = "1.0"
	}
	return &ProtobufSagaEventSerializer{
		version: version,
	}
}

// Serialize implements SagaEventSerializer interface for Protobuf format.
func (s *ProtobufSagaEventSerializer) Serialize(ctx context.Context, event *saga.SagaEvent) ([]byte, error) {
	if event == nil {
		return nil, ErrInvalidEvent
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("serialization cancelled: %w", ctx.Err())
	default:
	}

	// Validate event before serialization
	if err := s.Validate(ctx, event); err != nil {
		return nil, fmt.Errorf("event validation failed: %w", err)
	}

	// Convert to protobuf message
	pbEvent := s.toProtobufEvent(event)

	// Serialize to bytes
	data, err := proto.Marshal(pbEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf saga event: %w", err)
	}

	return data, nil
}

// Deserialize implements SagaEventSerializer interface for Protobuf format.
func (s *ProtobufSagaEventSerializer) Deserialize(ctx context.Context, data []byte) (*saga.SagaEvent, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("cannot deserialize empty data")
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("deserialization cancelled: %w", ctx.Err())
	default:
	}

	// Unmarshal protobuf message
	pbEvent := &pbsaga.SagaEvent{}
	if err := proto.Unmarshal(data, pbEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal protobuf saga event: %w", err)
	}

	// Convert from protobuf message
	event := s.fromProtobufEvent(pbEvent)

	// Validate the deserialized event
	if err := s.Validate(ctx, event); err != nil {
		return nil, fmt.Errorf("deserialized event validation failed: %w", err)
	}

	return event, nil
}

// ContentType implements SagaEventSerializer interface.
func (s *ProtobufSagaEventSerializer) ContentType() string {
	return "application/x-protobuf"
}

// FormatName implements SagaEventSerializer interface.
func (s *ProtobufSagaEventSerializer) FormatName() string {
	return "protobuf"
}

// Validate implements SagaEventSerializer interface.
func (s *ProtobufSagaEventSerializer) Validate(ctx context.Context, event *saga.SagaEvent) error {
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

	// Validate timestamp
	if event.Timestamp.IsZero() {
		return fmt.Errorf("event timestamp is required")
	}

	return nil
}

// toProtobufEvent converts a saga.SagaEvent to a protobuf SagaEvent message.
func (s *ProtobufSagaEventSerializer) toProtobufEvent(event *saga.SagaEvent) *pbsaga.SagaEvent {
	pbEvent := &pbsaga.SagaEvent{
		Id:            event.ID,
		SagaId:        event.SagaID,
		StepId:        event.StepID,
		Type:          string(event.Type),
		Version:       event.Version,
		Timestamp:     timestamppb.New(event.Timestamp),
		CorrelationId: event.CorrelationID,
		Source:        event.Source,
		Service:       event.Service,
		ServiceVersion: event.ServiceVersion,
		TraceId:       event.TraceID,
		SpanId:        event.SpanID,
		ParentSpanId:  event.ParentSpanID,
		Attempt:       int32(event.Attempt),
		MaxAttempts:   int32(event.MaxAttempts),
	}

	// Convert duration
	if event.Duration > 0 {
		pbEvent.Duration = int64(event.Duration)
	}

	// Convert error if present
	if event.Error != nil {
		pbEvent.Error = &pbsaga.SagaError{
			Code:      event.Error.Code,
			Message:   event.Error.Message,
			Retryable: event.Error.Retryable,
		}
		if event.Error.Details != nil {
			// Serialize error details as JSON
			if detailsJSON, err := json.Marshal(event.Error.Details); err == nil {
				pbEvent.Error.Details = string(detailsJSON)
			}
		}
	}

	// Convert metadata
	if event.Metadata != nil {
		pbEvent.Metadata = make(map[string]string)
		for k, v := range event.Metadata {
			if strVal, ok := v.(string); ok {
				pbEvent.Metadata[k] = strVal
			} else {
				// Convert non-string values to JSON
				if jsonVal, err := json.Marshal(v); err == nil {
					pbEvent.Metadata[k] = string(jsonVal)
				}
			}
		}
	}

	// Convert data, previous state, and new state
	if event.Data != nil {
		if dataJSON, err := json.Marshal(event.Data); err == nil {
			pbEvent.Data = dataJSON
		}
	}
	if event.PreviousState != nil {
		if stateJSON, err := json.Marshal(event.PreviousState); err == nil {
			pbEvent.PreviousState = stateJSON
		}
	}
	if event.NewState != nil {
		if stateJSON, err := json.Marshal(event.NewState); err == nil {
			pbEvent.NewState = stateJSON
		}
	}

	return pbEvent
}

// fromProtobufEvent converts a protobuf SagaEvent message to a saga.SagaEvent.
func (s *ProtobufSagaEventSerializer) fromProtobufEvent(pbEvent *pbsaga.SagaEvent) *saga.SagaEvent {
	event := &saga.SagaEvent{
		ID:             pbEvent.Id,
		SagaID:         pbEvent.SagaId,
		StepID:         pbEvent.StepId,
		Type:           saga.SagaEventType(pbEvent.Type),
		Version:        pbEvent.Version,
		CorrelationID:  pbEvent.CorrelationId,
		Source:         pbEvent.Source,
		Service:        pbEvent.Service,
		ServiceVersion: pbEvent.ServiceVersion,
		TraceID:        pbEvent.TraceId,
		SpanID:         pbEvent.SpanId,
		ParentSpanID:   pbEvent.ParentSpanId,
		Attempt:        int(pbEvent.Attempt),
		MaxAttempts:    int(pbEvent.MaxAttempts),
	}

	// Convert timestamp
	if pbEvent.Timestamp != nil {
		event.Timestamp = pbEvent.Timestamp.AsTime()
	}

	// Convert duration
	if pbEvent.Duration > 0 {
		event.Duration = time.Duration(pbEvent.Duration)
	}

	// Convert error if present
	if pbEvent.Error != nil {
		event.Error = &saga.SagaError{
			Code:      pbEvent.Error.Code,
			Message:   pbEvent.Error.Message,
			Retryable: pbEvent.Error.Retryable,
		}
		// Deserialize error details from JSON
		if pbEvent.Error.Details != "" {
			var details map[string]interface{}
			if err := json.Unmarshal([]byte(pbEvent.Error.Details), &details); err == nil {
				event.Error.Details = details
			}
		}
	}

	// Convert metadata
	if pbEvent.Metadata != nil {
		event.Metadata = make(map[string]interface{})
		for k, v := range pbEvent.Metadata {
			// Try to parse as JSON first
			var jsonVal interface{}
			if err := json.Unmarshal([]byte(v), &jsonVal); err == nil {
				event.Metadata[k] = jsonVal
			} else {
				event.Metadata[k] = v
			}
		}
	}

	// Convert data, previous state, and new state
	if len(pbEvent.Data) > 0 {
		var data interface{}
		if err := json.Unmarshal(pbEvent.Data, &data); err == nil {
			event.Data = data
		}
	}
	if len(pbEvent.PreviousState) > 0 {
		var state interface{}
		if err := json.Unmarshal(pbEvent.PreviousState, &state); err == nil {
			event.PreviousState = state
		}
	}
	if len(pbEvent.NewState) > 0 {
		var state interface{}
		if err := json.Unmarshal(pbEvent.NewState, &state); err == nil {
			event.NewState = state
		}
	}

	return event
}

// SerializerOptions provides configuration options for serializer creation.
type SerializerOptions struct {
	// Format specifies the serialization format
	Format string `json:"format"`

	// Pretty enables pretty-printing for formats that support it
	Pretty bool `json:"pretty,omitempty"`

	// Version specifies the serialization version
	Version string `json:"version,omitempty"`
}

// NewSagaEventSerializer creates a new SagaEventSerializer based on the specified format.
//
// Parameters:
//   - format: Serialization format ("json" or "protobuf")
//   - options: Optional serializer options
//
// Returns:
//   - SagaEventSerializer: Configured serializer instance
//   - error: Error if the format is unsupported
func NewSagaEventSerializer(format string, options *SerializerOptions) (SagaEventSerializer, error) {
	if options == nil {
		options = &SerializerOptions{
			Format:  format,
			Pretty:  false,
			Version: "1.0",
		}
	}

	switch format {
	case "json":
		return NewJSONSagaEventSerializerWithOptions(options.Pretty, options.Version), nil
	case "protobuf", "proto":
		return NewProtobufSagaEventSerializerWithOptions(options.Version), nil
	default:
		return nil, fmt.Errorf("unsupported serialization format: %s", format)
	}
}

