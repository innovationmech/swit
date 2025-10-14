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

package storage

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

const (
	// SerializationVersion is the current version of the serialization format
	SerializationVersion = "1.0.0"

	// CompressionThreshold is the minimum size in bytes before compression is applied
	CompressionThreshold = 1024 // 1KB
)

// SerializationOptions configures how saga states are serialized
type SerializationOptions struct {
	// EnableCompression enables gzip compression for large payloads
	EnableCompression bool

	// CompressionLevel sets the gzip compression level (0-9, -1 for default)
	CompressionLevel int

	// PrettyPrint formats JSON with indentation (useful for debugging)
	PrettyPrint bool
}

// DefaultSerializationOptions returns the default serialization options
func DefaultSerializationOptions() *SerializationOptions {
	return &SerializationOptions{
		EnableCompression: true,
		CompressionLevel:  gzip.DefaultCompression,
		PrettyPrint:       false,
	}
}

// SerializedData wraps the serialized saga data with metadata
type SerializedData struct {
	// Version is the serialization format version
	Version string `json:"version"`

	// Compressed indicates if the data is compressed
	Compressed bool `json:"compressed"`

	// Timestamp is when the data was serialized
	Timestamp int64 `json:"timestamp"`

	// Data contains the actual serialized saga data
	Data []byte `json:"data"`
}

// SagaSerializer handles serialization and deserialization of saga states
type SagaSerializer struct {
	options *SerializationOptions
}

// NewSagaSerializer creates a new saga serializer with the given options
func NewSagaSerializer(options *SerializationOptions) *SagaSerializer {
	if options == nil {
		options = DefaultSerializationOptions()
	}
	return &SagaSerializer{
		options: options,
	}
}

// SerializeSagaInstance serializes a SagaInstance to bytes
func (s *SagaSerializer) SerializeSagaInstance(instance saga.SagaInstance) ([]byte, error) {
	if instance == nil {
		return nil, fmt.Errorf("saga instance cannot be nil")
	}

	// Convert to SagaInstanceData for serialization
	data := convertToSerializableData(instance)

	// Serialize to JSON
	var jsonData []byte
	var err error

	if s.options.PrettyPrint {
		jsonData, err = json.MarshalIndent(data, "", "  ")
	} else {
		jsonData, err = json.Marshal(data)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal saga instance: %w", err)
	}

	// Compress if enabled and data is large enough
	compressed := false
	finalData := jsonData

	if s.options.EnableCompression && len(jsonData) >= CompressionThreshold {
		compressedData, err := s.compress(jsonData)
		if err != nil {
			// Log warning but continue with uncompressed data
			// In production, you might want to use proper logging
		} else {
			finalData = compressedData
			compressed = true
		}
	}

	// Wrap with metadata
	wrapped := SerializedData{
		Version:    SerializationVersion,
		Compressed: compressed,
		Timestamp:  time.Now().Unix(),
		Data:       finalData,
	}

	// Serialize the wrapped data (use pretty print if enabled)
	var wrappedJSON []byte
	if s.options.PrettyPrint {
		wrappedJSON, err = json.MarshalIndent(wrapped, "", "  ")
	} else {
		wrappedJSON, err = json.Marshal(wrapped)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to marshal wrapped data: %w", err)
	}

	return wrappedJSON, nil
}

// DeserializeSagaInstance deserializes bytes back to a SagaInstanceData
func (s *SagaSerializer) DeserializeSagaInstance(data []byte) (*saga.SagaInstanceData, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	// Try to unwrap the metadata first
	var wrapped SerializedData
	if err := json.Unmarshal(data, &wrapped); err != nil {
		// Fallback: try to deserialize as raw SagaInstanceData for backward compatibility
		return s.deserializeLegacyFormat(data)
	}

	// If Data field is empty, this is likely a legacy format
	// (SerializedData was parsed but it's actually SagaInstanceData)
	if len(wrapped.Data) == 0 {
		return s.deserializeLegacyFormat(data)
	}

	// Check version compatibility
	if err := s.checkVersionCompatibility(wrapped.Version); err != nil {
		return nil, err
	}

	// Decompress if needed
	finalData := wrapped.Data
	if wrapped.Compressed {
		decompressed, err := s.decompress(wrapped.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress data: %w", err)
		}
		finalData = decompressed
	}

	// Deserialize the saga instance data
	var instanceData saga.SagaInstanceData
	if err := json.Unmarshal(finalData, &instanceData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal saga instance data: %w", err)
	}

	return &instanceData, nil
}

// SerializeStepState serializes a StepState to bytes
func (s *SagaSerializer) SerializeStepState(step *saga.StepState) ([]byte, error) {
	if step == nil {
		return nil, fmt.Errorf("step state cannot be nil")
	}

	// Serialize to JSON
	var jsonData []byte
	var err error

	if s.options.PrettyPrint {
		jsonData, err = json.MarshalIndent(step, "", "  ")
	} else {
		jsonData, err = json.Marshal(step)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal step state: %w", err)
	}

	// Compress if enabled and data is large enough
	compressed := false
	finalData := jsonData

	if s.options.EnableCompression && len(jsonData) >= CompressionThreshold {
		compressedData, err := s.compress(jsonData)
		if err == nil {
			finalData = compressedData
			compressed = true
		}
	}

	// Wrap with metadata
	wrapped := SerializedData{
		Version:    SerializationVersion,
		Compressed: compressed,
		Timestamp:  time.Now().Unix(),
		Data:       finalData,
	}

	// Serialize the wrapped data
	wrappedJSON, err := json.Marshal(wrapped)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal wrapped step data: %w", err)
	}

	return wrappedJSON, nil
}

// DeserializeStepState deserializes bytes back to a StepState
func (s *SagaSerializer) DeserializeStepState(data []byte) (*saga.StepState, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	// Try to unwrap the metadata first
	var wrapped SerializedData
	if err := json.Unmarshal(data, &wrapped); err != nil {
		// Fallback: try to deserialize as raw StepState for backward compatibility
		var step saga.StepState
		if err := json.Unmarshal(data, &step); err != nil {
			return nil, fmt.Errorf("failed to unmarshal step state: %w", err)
		}
		return &step, nil
	}

	// Check version compatibility
	if err := s.checkVersionCompatibility(wrapped.Version); err != nil {
		return nil, err
	}

	// Decompress if needed
	finalData := wrapped.Data
	if wrapped.Compressed {
		decompressed, err := s.decompress(wrapped.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress step data: %w", err)
		}
		finalData = decompressed
	}

	// Deserialize the step state
	var step saga.StepState
	if err := json.Unmarshal(finalData, &step); err != nil {
		return nil, fmt.Errorf("failed to unmarshal step state: %w", err)
	}

	return &step, nil
}

// compress compresses data using gzip
func (s *SagaSerializer) compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, s.options.CompressionLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		w.Close()
		return nil, fmt.Errorf("failed to write compressed data: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// decompress decompresses gzip data
func (s *SagaSerializer) decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read decompressed data: %w", err)
	}

	return decompressed, nil
}

// checkVersionCompatibility checks if the serialization version is compatible
func (s *SagaSerializer) checkVersionCompatibility(version string) error {
	// Empty version means legacy format (no version wrapper), which is compatible
	if version == "" {
		return nil
	}

	// For now, we only support version 1.0.0
	// In the future, we can add version migration logic here
	if version != SerializationVersion {
		// Try to parse version and check compatibility
		// For now, we'll be lenient and accept any 1.x.x version
		if len(version) > 0 && version[0] == '1' {
			return nil
		}
		return fmt.Errorf("incompatible serialization version: %s (expected: %s)", version, SerializationVersion)
	}
	return nil
}

// deserializeLegacyFormat attempts to deserialize data in the old format (without metadata wrapper)
func (s *SagaSerializer) deserializeLegacyFormat(data []byte) (*saga.SagaInstanceData, error) {
	var instanceData saga.SagaInstanceData
	if err := json.Unmarshal(data, &instanceData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal legacy format: %w", err)
	}
	return &instanceData, nil
}

// convertToSerializableData converts a SagaInstance to SagaInstanceData for serialization
// This handles all the special fields like timestamps properly
func convertToSerializableData(instance saga.SagaInstance) *saga.SagaInstanceData {
	now := time.Now()
	startTime := instance.GetStartTime()
	endTime := instance.GetEndTime()

	data := &saga.SagaInstanceData{
		ID:           instance.GetID(),
		DefinitionID: instance.GetDefinitionID(),
		State:        instance.GetState(),
		CurrentStep:  instance.GetCurrentStep(),
		TotalSteps:   instance.GetTotalSteps(),
		CreatedAt:    instance.GetCreatedAt(),
		UpdatedAt:    instance.GetUpdatedAt(),
		Timeout:      instance.GetTimeout(),
		Metadata:     instance.GetMetadata(),
		TraceID:      instance.GetTraceID(),
		Error:        instance.GetError(),
		ResultData:   instance.GetResult(),
	}

	// Handle optional timestamp fields
	// Set started time if available
	if !startTime.IsZero() {
		data.StartedAt = &startTime
	}

	// Set completed time or timed out time based on state
	if !endTime.IsZero() {
		if instance.GetState() == saga.StateTimedOut {
			data.TimedOutAt = &endTime
		} else {
			data.CompletedAt = &endTime
		}
	}

	// Ensure UpdatedAt is set
	if data.UpdatedAt.IsZero() {
		data.UpdatedAt = now
	}

	// Ensure CreatedAt is set
	if data.CreatedAt.IsZero() {
		data.CreatedAt = now
	}

	return data
}
