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

package kafka

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/innovationmech/swit/pkg/messaging"
)

const (
	schemaHeaderID      = "schema.registry.id"
	schemaHeaderSubject = "schema.registry.subject"
	schemaHeaderVersion = "schema.registry.version"
	schemaHeaderFormat  = "schema.registry.format"
)

type schemaSerDe struct {
	format   messaging.SerializationFormat
	registry *schemaRegistryClient
	config   *messaging.SerializationOptions
}

func newSchemaSerDe(cfg *messaging.SerializationOptions, manager *schemaRegistryManager) (*schemaSerDe, error) {
	if cfg == nil || cfg.SchemaRegistry == nil {
		return nil, fmt.Errorf("schema registry serialization requires configuration")
	}
	switch cfg.Format {
	case messaging.FormatAvro, messaging.FormatJSON:
	default:
		return nil, fmt.Errorf("schema registry integration supports avro or json formats, got %s", cfg.Format)
	}

	client, err := manager.getClient(cfg.SchemaRegistry)
	if err != nil {
		return nil, err
	}

	return &schemaSerDe{
		format:   cfg.Format,
		registry: client,
		config:   cfg,
	}, nil
}

func (s *schemaSerDe) Encode(ctx context.Context, message *messaging.Message) (*messaging.Message, error) {
	if s == nil {
		return message, nil
	}
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	payload := message.Payload
	if len(payload) == 0 {
		return nil, messaging.NewSerializationErrorWithFormat(s.format, "payload cannot be empty for schema registry encoding", nil)
	}

	if isConfluentWireFormat(payload) {
		return s.decorateFromEncoded(ctx, message, payload)
	}

	descriptor, err := s.registry.getSchemaForSubject(ctx, s.config.SchemaRegistry.Subject, s.config.SchemaRegistry.Version)
	if err != nil {
		return nil, err
	}

	if descriptor.format != s.format {
		return nil, fmt.Errorf("schema format mismatch: registry format %s, configured %s", descriptor.format, s.format)
	}

	var encoded []byte
	switch s.format {
	case messaging.FormatAvro:
		encoded, err = encodeAvroPayload(descriptor, payload)
	case messaging.FormatJSON:
		encoded, err = encodeJSONPayload(descriptor, payload)
	default:
		return nil, fmt.Errorf("serialization format %s not supported for schema registry integration", s.format)
	}
	if err != nil {
		return nil, err
	}

	return cloneWithSchemaHeaders(message, encoded, descriptor, s.config.SchemaRegistry.Subject), nil
}

func (s *schemaSerDe) Decode(ctx context.Context, message *messaging.Message) (*messaging.Message, error) {
	if s == nil {
		return message, nil
	}
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	payload := message.Payload
	if !isConfluentWireFormat(payload) {
		return message, nil
	}

	schemaID := int(binary.BigEndian.Uint32(payload[1:5]))
	descriptor, err := s.registry.getSchemaByID(ctx, schemaID)
	if err != nil {
		return nil, err
	}

	var decoded []byte
	switch descriptor.format {
	case messaging.FormatAvro:
		decoded, err = decodeAvroPayload(descriptor, payload[5:])
	case messaging.FormatJSON:
		decoded, err = decodeJSONPayload(descriptor, payload[5:])
	default:
		return nil, fmt.Errorf("unsupported schema type %s for decoding", descriptor.format)
	}
	if err != nil {
		return nil, err
	}

	return cloneWithSchemaHeaders(message, decoded, descriptor, s.config.SchemaRegistry.Subject), nil
}

func (s *schemaSerDe) decorateFromEncoded(ctx context.Context, message *messaging.Message, payload []byte) (*messaging.Message, error) {
	schemaID := int(binary.BigEndian.Uint32(payload[1:5]))
	descriptor, err := s.registry.getSchemaByID(ctx, schemaID)
	if err != nil {
		return nil, err
	}

	// Only validate format matches expectation when the payload was pre-encoded
	if descriptor.format != s.format {
		return nil, fmt.Errorf("encoded payload uses format %s but publisher expects %s", descriptor.format, s.format)
	}

	return cloneWithSchemaHeaders(message, payload, descriptor, s.config.SchemaRegistry.Subject), nil
}

func cloneWithSchemaHeaders(message *messaging.Message, payload []byte, descriptor *schemaDescriptor, fallbackSubject string) *messaging.Message {
	clone := *message
	headers := make(map[string]string, len(message.Headers)+4)
	for k, v := range message.Headers {
		headers[k] = v
	}

	if descriptor != nil {
		headers[schemaHeaderID] = strconv.Itoa(descriptor.id)
		subject := descriptor.subject
		if subject == "" {
			if headers[schemaHeaderSubject] != "" {
				subject = headers[schemaHeaderSubject]
			} else {
				subject = fallbackSubject
			}
		}
		headers[schemaHeaderSubject] = subject
		headers[schemaHeaderVersion] = strconv.Itoa(descriptor.version)
		headers[schemaHeaderFormat] = descriptor.format.String()
	}

	clone.Headers = headers
	clone.Payload = append([]byte(nil), payload...)
	return &clone
}

func isConfluentWireFormat(payload []byte) bool {
	return len(payload) > 5 && payload[0] == 0
}

func encodeAvroPayload(descriptor *schemaDescriptor, payload []byte) ([]byte, error) {
	codec := descriptor.schema.Codec()
	if codec == nil {
		return nil, fmt.Errorf("schema id %d does not provide an Avro codec", descriptor.id)
	}

	native, _, err := codec.NativeFromTextual(payload)
	if err != nil {
		return nil, messaging.NewSerializationErrorWithFormat(messaging.FormatAvro, "failed to convert JSON payload to Avro", err)
	}

	binaryValue, err := codec.BinaryFromNative(nil, native)
	if err != nil {
		return nil, messaging.NewSerializationErrorWithFormat(messaging.FormatAvro, "failed to encode Avro payload", err)
	}

	return wrapWithSchemaID(descriptor.id, binaryValue), nil
}

func encodeJSONPayload(descriptor *schemaDescriptor, payload []byte) ([]byte, error) {
	validator := descriptor.schema.JsonSchema()
	if validator == nil {
		return nil, fmt.Errorf("schema id %d does not have a compiled JSON Schema", descriptor.id)
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return nil, messaging.NewSerializationErrorWithFormat(messaging.FormatJSON, "invalid JSON payload for schema registry", err)
	}

	if err := validator.Validate(value); err != nil {
		return nil, messaging.NewSerializationErrorWithFormat(messaging.FormatJSON, "JSON payload failed schema validation", err)
	}

	return wrapWithSchemaID(descriptor.id, payload), nil
}

func decodeAvroPayload(descriptor *schemaDescriptor, payload []byte) ([]byte, error) {
	codec := descriptor.schema.Codec()
	if codec == nil {
		return nil, fmt.Errorf("schema id %d does not provide an Avro codec", descriptor.id)
	}

	native, _, err := codec.NativeFromBinary(payload)
	if err != nil {
		return nil, messaging.NewSerializationErrorWithFormat(messaging.FormatAvro, "failed to decode Avro payload", err)
	}

	textual, err := codec.TextualFromNative(nil, native)
	if err != nil {
		return nil, messaging.NewSerializationErrorWithFormat(messaging.FormatAvro, "failed to convert Avro payload to JSON", err)
	}

	return textual, nil
}

func decodeJSONPayload(descriptor *schemaDescriptor, payload []byte) ([]byte, error) {
	validator := descriptor.schema.JsonSchema()
	if validator == nil {
		return append([]byte(nil), payload...), nil
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return nil, messaging.NewSerializationErrorWithFormat(messaging.FormatJSON, "invalid JSON payload while decoding", err)
	}

	if err := validator.Validate(value); err != nil {
		return nil, messaging.NewSerializationErrorWithFormat(messaging.FormatJSON, "JSON payload failed schema validation during decode", err)
	}

	return append([]byte(nil), payload...), nil
}

func wrapWithSchemaID(id int, payload []byte) []byte {
	out := make([]byte, 5+len(payload))
	out[0] = 0
	binary.BigEndian.PutUint32(out[1:5], uint32(id))
	copy(out[5:], payload)
	return out
}
