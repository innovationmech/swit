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
	"fmt"
	"io"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ProtobufSerializer implements MessageSerializer for Protocol Buffer format.
type ProtobufSerializer struct {
	options     *ProtobufSerializationOptions
	schemaCache map[string]*ProtobufSchema
	mutex       sync.RWMutex
}

// ProtobufSerializationOptions provides configuration for Protobuf serialization.
type ProtobufSerializationOptions struct {
	// ValidateOnSerialize enables validation before serialization.
	ValidateOnSerialize bool `json:"validate_on_serialize"`

	// ValidateOnDeserialize enables validation after deserialization.
	ValidateOnDeserialize bool `json:"validate_on_deserialize"`

	// AllowPartial allows serialization of messages with missing required fields.
	AllowPartial bool `json:"allow_partial"`

	// Deterministic ensures deterministic serialization (same input produces same output).
	Deterministic bool `json:"deterministic"`

	// DiscardUnknown discards unknown fields during deserialization.
	DiscardUnknown bool `json:"discard_unknown"`

	// MaxRecursionDepth limits the maximum recursion depth during serialization/deserialization.
	MaxRecursionDepth int `json:"max_recursion_depth,omitempty"`

	// UseCompactEncoding uses compact encoding where possible to reduce message size.
	UseCompactEncoding bool `json:"use_compact_encoding"`

	// Registry specifies a custom type registry for message types.
	Registry *protoregistry.Types `json:"-"`
}

// ProtobufSchema represents a Protocol Buffer schema for validation.
type ProtobufSchema struct {
	version     string
	descriptor  *descriptorpb.FileDescriptorProto
	messageDesc protoreflect.MessageDescriptor
}

// GetVersion implements Schema interface.
func (s *ProtobufSchema) GetVersion() string {
	return s.version
}

// GetDefinition implements Schema interface.
func (s *ProtobufSchema) GetDefinition() interface{} {
	return s.descriptor
}

// Validate implements Schema interface.
func (s *ProtobufSchema) Validate(ctx context.Context, data []byte) error {
	if s.messageDesc == nil {
		return &SchemaError{
			Message: "no message descriptor configured for validation",
		}
	}

	// Create a dynamic message from the descriptor
	message := dynamicpb.NewMessage(s.messageDesc)

	// Try to unmarshal the data to validate its structure
	if err := proto.Unmarshal(data, message); err != nil {
		return &SchemaError{
			Message: "protobuf validation failed",
			Cause:   err,
		}
	}

	return nil
}

// GetMessageDescriptor returns the protobuf message descriptor.
func (s *ProtobufSchema) GetMessageDescriptor() protoreflect.MessageDescriptor {
	return s.messageDesc
}

// NewProtobufSerializer creates a new Protobuf serializer with the specified options.
func NewProtobufSerializer(options *ProtobufSerializationOptions) *ProtobufSerializer {
	if options == nil {
		options = &ProtobufSerializationOptions{
			ValidateOnSerialize:   true,
			ValidateOnDeserialize: true,
			AllowPartial:          false,
			Deterministic:         true,
			DiscardUnknown:        false,
			MaxRecursionDepth:     100,
			UseCompactEncoding:    false,
			Registry:              protoregistry.GlobalTypes,
		}
	}

	// Set defaults for unspecified fields
	if options.MaxRecursionDepth == 0 {
		options.MaxRecursionDepth = 100
	}
	if options.Registry == nil {
		options.Registry = protoregistry.GlobalTypes
	}

	return &ProtobufSerializer{
		options:     options,
		schemaCache: make(map[string]*ProtobufSchema),
	}
}

// GetFormat implements MessageSerializer interface.
func (p *ProtobufSerializer) GetFormat() SerializationFormat {
	return FormatProtobuf
}

// Serialize implements MessageSerializer interface.
func (p *ProtobufSerializer) Serialize(ctx context.Context, message interface{}) ([]byte, error) {
	if message == nil {
		return nil, &SerializationError{
			Format:  FormatProtobuf,
			Message: "message cannot be nil",
		}
	}

	// Check if message implements proto.Message
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return nil, &SerializationError{
			Format:  FormatProtobuf,
			Message: "message does not implement proto.Message interface",
		}
	}

	// Validate before serialization if enabled
	if p.options.ValidateOnSerialize {
		if err := p.ValidateMessage(ctx, message); err != nil {
			return nil, &SerializationError{
				Format:  FormatProtobuf,
				Message: "message validation failed before serialization",
				Cause:   err,
			}
		}
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, &SerializationError{
			Format:  FormatProtobuf,
			Message: "serialization cancelled",
			Cause:   ctx.Err(),
		}
	default:
	}

	// Set marshaling options
	marshalOptions := proto.MarshalOptions{
		AllowPartial:  p.options.AllowPartial,
		Deterministic: p.options.Deterministic,
	}

	data, err := marshalOptions.Marshal(protoMessage)
	if err != nil {
		return nil, &SerializationError{
			Format:  FormatProtobuf,
			Message: "failed to marshal protobuf message",
			Cause:   err,
		}
	}

	return data, nil
}

// Deserialize implements MessageSerializer interface.
func (p *ProtobufSerializer) Deserialize(ctx context.Context, data []byte, target interface{}) error {
	if len(data) == 0 {
		return &SerializationError{
			Format:  FormatProtobuf,
			Message: "data cannot be empty",
		}
	}

	if target == nil {
		return &SerializationError{
			Format:  FormatProtobuf,
			Message: "target cannot be nil",
		}
	}

	// Check if target implements proto.Message
	protoMessage, ok := target.(proto.Message)
	if !ok {
		return &SerializationError{
			Format:  FormatProtobuf,
			Message: "target does not implement proto.Message interface",
		}
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return &SerializationError{
			Format:  FormatProtobuf,
			Message: "deserialization cancelled",
			Cause:   ctx.Err(),
		}
	default:
	}

	// Set unmarshaling options
	unmarshalOptions := proto.UnmarshalOptions{
		AllowPartial:   p.options.AllowPartial,
		DiscardUnknown: p.options.DiscardUnknown,
		Resolver:       p.options.Registry,
	}

	if err := unmarshalOptions.Unmarshal(data, protoMessage); err != nil {
		return &SerializationError{
			Format:  FormatProtobuf,
			Message: "failed to unmarshal protobuf message",
			Cause:   err,
		}
	}

	// Validate after deserialization if enabled
	if p.options.ValidateOnDeserialize {
		if err := p.ValidateMessage(ctx, target); err != nil {
			return &SerializationError{
				Format:  FormatProtobuf,
				Message: "message validation failed after deserialization",
				Cause:   err,
			}
		}
	}

	return nil
}

// SerializeToWriter implements MessageSerializer interface.
func (p *ProtobufSerializer) SerializeToWriter(ctx context.Context, message interface{}, writer io.Writer) error {
	if message == nil {
		return &SerializationError{
			Format:  FormatProtobuf,
			Message: "message cannot be nil",
		}
	}

	if writer == nil {
		return &SerializationError{
			Format:  FormatProtobuf,
			Message: "writer cannot be nil",
		}
	}

	// Serialize to bytes first, then write to writer
	data, err := p.Serialize(ctx, message)
	if err != nil {
		return err
	}

	if _, err := writer.Write(data); err != nil {
		return &SerializationError{
			Format:  FormatProtobuf,
			Message: "failed to write protobuf data to writer",
			Cause:   err,
		}
	}

	return nil
}

// DeserializeFromReader implements MessageSerializer interface.
func (p *ProtobufSerializer) DeserializeFromReader(ctx context.Context, reader io.Reader, target interface{}) error {
	if reader == nil {
		return &SerializationError{
			Format:  FormatProtobuf,
			Message: "reader cannot be nil",
		}
	}

	if target == nil {
		return &SerializationError{
			Format:  FormatProtobuf,
			Message: "target cannot be nil",
		}
	}

	// Read all data from reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return &SerializationError{
			Format:  FormatProtobuf,
			Message: "failed to read data from reader",
			Cause:   err,
		}
	}

	return p.Deserialize(ctx, data, target)
}

// ValidateMessage implements MessageSerializer interface.
func (p *ProtobufSerializer) ValidateMessage(ctx context.Context, message interface{}) error {
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

	// Check if message implements proto.Message
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return &ValidationError{
			Message: "message does not implement proto.Message interface",
		}
	}

	// Basic protobuf validation - check if message is valid
	if !protoMessage.ProtoReflect().IsValid() {
		return &ValidationError{
			Message: "protobuf message is not valid",
		}
	}

	// Additional validation can be added here based on schema
	if schema, err := p.getSchemaForMessage(message); err == nil && schema != nil {
		// Serialize the message and validate against schema
		data, err := p.Serialize(context.Background(), message)
		if err != nil {
			return &ValidationError{
				Message: "failed to serialize message for schema validation",
				Cause:   err,
			}
		}

		if err := schema.Validate(ctx, data); err != nil {
			return &ValidationError{
				Message: "schema validation failed",
				Cause:   err,
			}
		}
	}

	return nil
}

// GetSchema implements MessageSerializer interface.
func (p *ProtobufSerializer) GetSchema(ctx context.Context) (Schema, error) {
	// For protobuf, schema information comes from message descriptors
	// This is a placeholder implementation
	return nil, &SchemaError{
		Message: "protobuf serializer requires message-specific schema lookup",
	}
}

// SetSchema allows setting a protobuf schema for validation.
func (p *ProtobufSerializer) SetSchema(messageName string, schema *ProtobufSchema) error {
	if messageName == "" {
		return &SchemaError{
			Message: "message name cannot be empty",
		}
	}

	if schema == nil {
		return &SchemaError{
			Message: "schema cannot be nil",
		}
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.schemaCache[messageName] = schema
	return nil
}

// Close implements MessageSerializer interface.
func (p *ProtobufSerializer) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Clear schema cache
	p.schemaCache = make(map[string]*ProtobufSchema)

	return nil
}

// getSchemaForMessage attempts to get a schema for the given message type.
func (p *ProtobufSerializer) getSchemaForMessage(message interface{}) (*ProtobufSchema, error) {
	if message == nil {
		return nil, &SchemaError{
			Message: "message cannot be nil",
		}
	}

	protoMessage, ok := message.(proto.Message)
	if !ok {
		return nil, &SchemaError{
			Message: "message does not implement proto.Message interface",
		}
	}

	// Get the message name
	messageName := string(protoMessage.ProtoReflect().Descriptor().Name())

	p.mutex.RLock()
	schema, exists := p.schemaCache[messageName]
	p.mutex.RUnlock()

	if !exists {
		return nil, &SchemaError{
			Message: fmt.Sprintf("no schema found for message type '%s'", messageName),
		}
	}

	return schema, nil
}

// GetMessageType returns the protobuf message type for a given message name.
func (p *ProtobufSerializer) GetMessageType(messageName string) (protoreflect.MessageType, error) {
	if messageName == "" {
		return nil, &SerializationError{
			Format:  FormatProtobuf,
			Message: "message name cannot be empty",
		}
	}

	messageType, err := p.options.Registry.FindMessageByName(protoreflect.FullName(messageName))
	if err != nil {
		return nil, &SerializationError{
			Format:  FormatProtobuf,
			Message: fmt.Sprintf("message type '%s' not found in registry", messageName),
			Cause:   err,
		}
	}

	return messageType, nil
}

// CreateMessage creates a new instance of a protobuf message by name.
func (p *ProtobufSerializer) CreateMessage(messageName string) (proto.Message, error) {
	messageType, err := p.GetMessageType(messageName)
	if err != nil {
		return nil, err
	}

	message := messageType.New()
	return message.Interface(), nil
}

// NewProtobufSchemaFromDescriptor creates a new protobuf schema from a message descriptor.
func NewProtobufSchemaFromDescriptor(version string, descriptor protoreflect.MessageDescriptor) *ProtobufSchema {
	return &ProtobufSchema{
		version:     version,
		messageDesc: descriptor,
	}
}

// NewProtobufSchemaFromFileDescriptor creates a new protobuf schema from a file descriptor.
func NewProtobufSchemaFromFileDescriptor(version string, fileDescriptor *descriptorpb.FileDescriptorProto, messageName string) (*ProtobufSchema, error) {
	if fileDescriptor == nil {
		return nil, &SchemaError{
			Message: "file descriptor cannot be nil",
		}
	}

	if messageName == "" {
		return nil, &SchemaError{
			Message: "message name cannot be empty",
		}
	}

	// Find the specific message descriptor within the file descriptor
	var messageDesc *descriptorpb.DescriptorProto
	for _, msg := range fileDescriptor.MessageType {
		if msg.GetName() == messageName {
			messageDesc = msg
			break
		}
	}

	if messageDesc == nil {
		return nil, &SchemaError{
			Message: fmt.Sprintf("message '%s' not found in file descriptor", messageName),
		}
	}

	return &ProtobufSchema{
		version:    version,
		descriptor: fileDescriptor,
		// messageDesc would need to be converted from DescriptorProto to MessageDescriptor
		// This is a simplified implementation
	}, nil
}

// ProtobufSchemaBuilder provides a fluent interface for building protobuf schemas.
type ProtobufSchemaBuilder struct {
	schema *ProtobufSchema
}

// NewProtobufSchemaBuilder creates a new protobuf schema builder.
func NewProtobufSchemaBuilder() *ProtobufSchemaBuilder {
	return &ProtobufSchemaBuilder{
		schema: &ProtobufSchema{
			version: "proto3",
		},
	}
}

// WithVersion sets the schema version.
func (b *ProtobufSchemaBuilder) WithVersion(version string) *ProtobufSchemaBuilder {
	b.schema.version = version
	return b
}

// WithMessageDescriptor sets the message descriptor.
func (b *ProtobufSchemaBuilder) WithMessageDescriptor(descriptor protoreflect.MessageDescriptor) *ProtobufSchemaBuilder {
	b.schema.messageDesc = descriptor
	return b
}

// WithFileDescriptor sets the file descriptor.
func (b *ProtobufSchemaBuilder) WithFileDescriptor(descriptor *descriptorpb.FileDescriptorProto) *ProtobufSchemaBuilder {
	b.schema.descriptor = descriptor
	return b
}

// Build creates the final protobuf schema.
func (b *ProtobufSchemaBuilder) Build() *ProtobufSchema {
	return b.schema
}

// RegisterProtobufType registers a protobuf message type with the global registry.
func RegisterProtobufType(messageType proto.Message) error {
	// This would register the message type with the global protobuf registry
	// In a real implementation, this would use protoregistry.GlobalTypes.RegisterMessage
	return nil
}

// GetRegisteredProtobufTypes returns all registered protobuf message types.
func GetRegisteredProtobufTypes() []string {
	// This would return all registered message types from the registry
	// This is a placeholder implementation
	return []string{}
}
