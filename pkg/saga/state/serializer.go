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

package state

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"

	"github.com/innovationmech/swit/pkg/saga"
)

// Serializer defines the interface for serializing and deserializing Saga state.
// It provides methods to convert Saga instances and step states to and from byte arrays,
// enabling persistent storage and network transmission.
type Serializer interface {
	// SerializeSaga serializes a SagaInstanceData to bytes.
	SerializeSaga(saga *saga.SagaInstanceData) ([]byte, error)

	// DeserializeSaga deserializes bytes to a SagaInstanceData.
	DeserializeSaga(data []byte) (*saga.SagaInstanceData, error)

	// SerializeStepState serializes a StepState to bytes.
	SerializeStepState(step *saga.StepState) ([]byte, error)

	// DeserializeStepState deserializes bytes to a StepState.
	DeserializeStepState(data []byte) (*saga.StepState, error)

	// SerializeStepStates serializes a slice of StepState to bytes.
	SerializeStepStates(steps []*saga.StepState) ([]byte, error)

	// DeserializeStepStates deserializes bytes to a slice of StepState.
	DeserializeStepStates(data []byte) ([]*saga.StepState, error)

	// Format returns the serialization format used by this serializer.
	Format() SerializationFormat
}

// SerializerConfig provides configuration for serializers.
type SerializerConfig struct {
	// Format specifies the serialization format to use.
	Format SerializationFormat `json:"format" yaml:"format"`

	// EnableCompression enables gzip compression of serialized data.
	EnableCompression bool `json:"enable_compression" yaml:"enable_compression"`

	// PrettyPrint enables pretty-printing for human-readable formats (JSON).
	// This is useful for debugging but increases the serialized data size.
	PrettyPrint bool `json:"pretty_print" yaml:"pretty_print"`
}

// DefaultSerializerConfig returns a default serializer configuration.
func DefaultSerializerConfig() *SerializerConfig {
	return &SerializerConfig{
		Format:            SerializationFormatJSON,
		EnableCompression: false,
		PrettyPrint:       false,
	}
}

// JSONSerializer implements the Serializer interface using JSON encoding.
// It provides a human-readable and widely compatible serialization format.
type JSONSerializer struct {
	config *SerializerConfig
}

// NewJSONSerializer creates a new JSON serializer with the given configuration.
// If config is nil, a default configuration is used.
func NewJSONSerializer(config *SerializerConfig) *JSONSerializer {
	if config == nil {
		config = DefaultSerializerConfig()
	}
	return &JSONSerializer{
		config: config,
	}
}

// SerializeSaga serializes a SagaInstanceData to JSON bytes.
func (s *JSONSerializer) SerializeSaga(saga *saga.SagaInstanceData) ([]byte, error) {
	if saga == nil {
		return nil, fmt.Errorf("saga instance is nil")
	}

	var data []byte
	var err error

	if s.config.PrettyPrint {
		data, err = json.MarshalIndent(saga, "", "  ")
	} else {
		data, err = json.Marshal(saga)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal saga: %w", err)
	}

	if s.config.EnableCompression {
		return s.compress(data)
	}

	return data, nil
}

// DeserializeSaga deserializes JSON bytes to a SagaInstanceData.
func (s *JSONSerializer) DeserializeSaga(data []byte) (*saga.SagaInstanceData, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data is empty")
	}

	var err error
	if s.config.EnableCompression {
		data, err = s.decompress(data)
		if err != nil {
			return nil, err
		}
	}

	var sagaData saga.SagaInstanceData
	if err := json.Unmarshal(data, &sagaData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal saga: %w", err)
	}

	return &sagaData, nil
}

// SerializeStepState serializes a StepState to JSON bytes.
func (s *JSONSerializer) SerializeStepState(step *saga.StepState) ([]byte, error) {
	if step == nil {
		return nil, fmt.Errorf("step state is nil")
	}

	var data []byte
	var err error

	if s.config.PrettyPrint {
		data, err = json.MarshalIndent(step, "", "  ")
	} else {
		data, err = json.Marshal(step)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal step state: %w", err)
	}

	if s.config.EnableCompression {
		return s.compress(data)
	}

	return data, nil
}

// DeserializeStepState deserializes JSON bytes to a StepState.
func (s *JSONSerializer) DeserializeStepState(data []byte) (*saga.StepState, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data is empty")
	}

	var err error
	if s.config.EnableCompression {
		data, err = s.decompress(data)
		if err != nil {
			return nil, err
		}
	}

	var stepState saga.StepState
	if err := json.Unmarshal(data, &stepState); err != nil {
		return nil, fmt.Errorf("failed to unmarshal step state: %w", err)
	}

	return &stepState, nil
}

// SerializeStepStates serializes a slice of StepState to JSON bytes.
func (s *JSONSerializer) SerializeStepStates(steps []*saga.StepState) ([]byte, error) {
	if steps == nil {
		return nil, fmt.Errorf("step states are nil")
	}

	var data []byte
	var err error

	if s.config.PrettyPrint {
		data, err = json.MarshalIndent(steps, "", "  ")
	} else {
		data, err = json.Marshal(steps)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal step states: %w", err)
	}

	if s.config.EnableCompression {
		return s.compress(data)
	}

	return data, nil
}

// DeserializeStepStates deserializes JSON bytes to a slice of StepState.
func (s *JSONSerializer) DeserializeStepStates(data []byte) ([]*saga.StepState, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data is empty")
	}

	var err error
	if s.config.EnableCompression {
		data, err = s.decompress(data)
		if err != nil {
			return nil, err
		}
	}

	var stepStates []*saga.StepState
	if err := json.Unmarshal(data, &stepStates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal step states: %w", err)
	}

	return stepStates, nil
}

// Format returns the serialization format used by this serializer.
func (s *JSONSerializer) Format() SerializationFormat {
	return SerializationFormatJSON
}

// compress compresses data using gzip.
func (s *JSONSerializer) compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// decompress decompresses gzip-compressed data.
func (s *JSONSerializer) decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	return decompressed, nil
}

// NewSerializer creates a serializer based on the provided configuration.
// Currently, only JSON serialization is supported.
func NewSerializer(config *SerializerConfig) (Serializer, error) {
	if config == nil {
		config = DefaultSerializerConfig()
	}

	switch config.Format {
	case SerializationFormatJSON:
		return NewJSONSerializer(config), nil
	case SerializationFormatProtobuf:
		return nil, fmt.Errorf("protobuf serialization is not yet implemented")
	case SerializationFormatMsgpack:
		return nil, fmt.Errorf("msgpack serialization is not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported serialization format: %s", config.Format)
	}
}
