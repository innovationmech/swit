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
	"compress/gzip"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

func TestSerializeSagaInstance(t *testing.T) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())

	now := time.Now()
	instance := createSerializerTestSagaInstance(now)

	data, err := serializer.SerializeSagaInstance(instance)
	if err != nil {
		t.Fatalf("Failed to serialize saga instance: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Serialized data should not be empty")
	}

	// Verify that it can be deserialized back
	deserialized, err := serializer.DeserializeSagaInstance(data)
	if err != nil {
		t.Fatalf("Failed to deserialize saga instance: %v", err)
	}

	// Verify key fields
	if deserialized.ID != "test-saga-1" {
		t.Errorf("Expected ID 'test-saga-1', got '%s'", deserialized.ID)
	}

	if deserialized.State != saga.StateRunning {
		t.Errorf("Expected state Running, got %v", deserialized.State)
	}

	if deserialized.DefinitionID != "test-definition" {
		t.Errorf("Expected DefinitionID 'test-definition', got '%s'", deserialized.DefinitionID)
	}
}

func TestSerializeSagaInstanceNil(t *testing.T) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())

	_, err := serializer.SerializeSagaInstance(nil)
	if err == nil {
		t.Fatal("Expected error when serializing nil saga instance")
	}
}

func TestDeserializeSagaInstanceEmpty(t *testing.T) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())

	_, err := serializer.DeserializeSagaInstance([]byte{})
	if err == nil {
		t.Fatal("Expected error when deserializing empty data")
	}
}

func TestSerializeWithCompression(t *testing.T) {
	opts := &SerializationOptions{
		EnableCompression: true,
		CompressionLevel:  gzip.BestCompression,
		PrettyPrint:       false,
	}
	serializer := NewSagaSerializer(opts)

	now := time.Now()
	instance := createLargeSerializerSagaInstance(now)

	data, err := serializer.SerializeSagaInstance(instance)
	if err != nil {
		t.Fatalf("Failed to serialize saga instance with compression: %v", err)
	}

	// Verify it's wrapped and compressed
	var wrapped SerializedData
	if err := json.Unmarshal(data, &wrapped); err != nil {
		t.Fatalf("Failed to unmarshal wrapped data: %v", err)
	}

	if wrapped.Version != SerializationVersion {
		t.Errorf("Expected version %s, got %s", SerializationVersion, wrapped.Version)
	}

	if !wrapped.Compressed {
		t.Error("Expected data to be compressed")
	}

	// Verify it can be deserialized
	deserialized, err := serializer.DeserializeSagaInstance(data)
	if err != nil {
		t.Fatalf("Failed to deserialize compressed saga instance: %v", err)
	}

	if deserialized.ID != "large-saga-1" {
		t.Errorf("Expected ID 'large-saga-1', got '%s'", deserialized.ID)
	}
}

func TestSerializeWithoutCompression(t *testing.T) {
	opts := &SerializationOptions{
		EnableCompression: false,
		PrettyPrint:       false,
	}
	serializer := NewSagaSerializer(opts)

	now := time.Now()
	instance := createLargeSerializerSagaInstance(now)

	data, err := serializer.SerializeSagaInstance(instance)
	if err != nil {
		t.Fatalf("Failed to serialize saga instance without compression: %v", err)
	}

	// Verify it's not compressed
	var wrapped SerializedData
	if err := json.Unmarshal(data, &wrapped); err != nil {
		t.Fatalf("Failed to unmarshal wrapped data: %v", err)
	}

	if wrapped.Compressed {
		t.Error("Expected data to not be compressed")
	}
}

func TestSerializeWithPrettyPrint(t *testing.T) {
	opts := &SerializationOptions{
		EnableCompression: false,
		PrettyPrint:       true,
	}
	serializer := NewSagaSerializer(opts)

	now := time.Now()
	instance := createSerializerTestSagaInstance(now)

	data, err := serializer.SerializeSagaInstance(instance)
	if err != nil {
		t.Fatalf("Failed to serialize saga instance with pretty print: %v", err)
	}

	// Verify it contains newlines (indication of pretty printing)
	if !strings.Contains(string(data), "\n") {
		t.Error("Expected pretty printed JSON to contain newlines")
	}

	// Verify it can still be deserialized
	deserialized, err := serializer.DeserializeSagaInstance(data)
	if err != nil {
		t.Fatalf("Failed to deserialize pretty printed saga instance: %v", err)
	}

	if deserialized.ID != "test-saga-1" {
		t.Errorf("Expected ID 'test-saga-1', got '%s'", deserialized.ID)
	}
}

func TestSerializeStepState(t *testing.T) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())

	now := time.Now()
	step := &saga.StepState{
		ID:        "step-1",
		SagaID:    "saga-1",
		StepIndex: 0,
		Name:      "Test Step",
		State:     saga.StepStateCompleted,
		Attempts:  1,
		CreatedAt: now,
		StartedAt: &now,
		Metadata:  map[string]interface{}{"key": "value"},
	}

	data, err := serializer.SerializeStepState(step)
	if err != nil {
		t.Fatalf("Failed to serialize step state: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Serialized step data should not be empty")
	}

	// Deserialize and verify
	deserialized, err := serializer.DeserializeStepState(data)
	if err != nil {
		t.Fatalf("Failed to deserialize step state: %v", err)
	}

	if deserialized.ID != step.ID {
		t.Errorf("Expected ID '%s', got '%s'", step.ID, deserialized.ID)
	}

	if deserialized.State != step.State {
		t.Errorf("Expected state %v, got %v", step.State, deserialized.State)
	}
}

func TestSerializeStepStateNil(t *testing.T) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())

	_, err := serializer.SerializeStepState(nil)
	if err == nil {
		t.Fatal("Expected error when serializing nil step state")
	}
}

func TestDeserializeStepStateEmpty(t *testing.T) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())

	_, err := serializer.DeserializeStepState([]byte{})
	if err == nil {
		t.Fatal("Expected error when deserializing empty step data")
	}
}

func TestVersionCompatibility(t *testing.T) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())

	tests := []struct {
		name        string
		version     string
		shouldError bool
	}{
		{"exact match", "1.0.0", false},
		{"minor version", "1.1.0", false},
		{"patch version", "1.0.1", false},
		{"major version", "2.0.0", true},
		{"invalid version", "0.9.0", true},
		{"empty version", "", false}, // Empty version is compatible (legacy format)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := serializer.checkVersionCompatibility(tt.version)
			if tt.shouldError && err == nil {
				t.Errorf("Expected error for version %s", tt.version)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error for version %s: %v", tt.version, err)
			}
		})
	}
}

func TestBackwardCompatibility(t *testing.T) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())

	// Create a legacy format (raw JSON without wrapper)
	now := time.Now()
	legacyData := &saga.SagaInstanceData{
		ID:           "legacy-saga",
		DefinitionID: "legacy-def",
		State:        saga.StateCompleted,
		CurrentStep:  1,
		TotalSteps:   1,
		CreatedAt:    now,
		UpdatedAt:    now,
		Timeout:      30 * time.Second,
	}

	legacyJSON, err := json.Marshal(legacyData)
	if err != nil {
		t.Fatalf("Failed to create legacy JSON: %v", err)
	}

	// Try to deserialize legacy format
	deserialized, err := serializer.DeserializeSagaInstance(legacyJSON)
	if err != nil {
		t.Fatalf("Failed to deserialize legacy format: %v", err)
	}

	if deserialized.ID != "legacy-saga" {
		t.Errorf("Expected ID 'legacy-saga', got '%s'", deserialized.ID)
	}

	if deserialized.State != saga.StateCompleted {
		t.Errorf("Expected state Completed, got %v", deserialized.State)
	}
}

func TestTimestampHandling(t *testing.T) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())

	now := time.Now()
	startedAt := now.Add(-10 * time.Minute)
	completedAt := now

	instance := &serializerTestSagaInstance{
		id:           "timestamp-test",
		definitionID: "test-def",
		state:        saga.StateCompleted,
		currentStep:  2,
		totalSteps:   2,
		createdAt:    now.Add(-15 * time.Minute),
		updatedAt:    now,
		startTime:    startedAt,
		endTime:      completedAt,
		timeout:      30 * time.Minute,
	}

	data, err := serializer.SerializeSagaInstance(instance)
	if err != nil {
		t.Fatalf("Failed to serialize saga with timestamps: %v", err)
	}

	deserialized, err := serializer.DeserializeSagaInstance(data)
	if err != nil {
		t.Fatalf("Failed to deserialize saga with timestamps: %v", err)
	}

	if deserialized.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	} else if !deserialized.StartedAt.Equal(startedAt) {
		t.Errorf("StartedAt mismatch: expected %v, got %v", startedAt, *deserialized.StartedAt)
	}

	if deserialized.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	} else if !deserialized.CompletedAt.Equal(completedAt) {
		t.Errorf("CompletedAt mismatch: expected %v, got %v", completedAt, *deserialized.CompletedAt)
	}
}

func TestTimedOutTimestamp(t *testing.T) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())

	now := time.Now()
	timedOutAt := now

	instance := &serializerTestSagaInstance{
		id:           "timeout-test",
		definitionID: "test-def",
		state:        saga.StateTimedOut,
		currentStep:  1,
		totalSteps:   2,
		createdAt:    now.Add(-15 * time.Minute),
		updatedAt:    now,
		startTime:    now.Add(-10 * time.Minute),
		endTime:      timedOutAt,
		timeout:      5 * time.Minute,
	}

	data, err := serializer.SerializeSagaInstance(instance)
	if err != nil {
		t.Fatalf("Failed to serialize timed out saga: %v", err)
	}

	deserialized, err := serializer.DeserializeSagaInstance(data)
	if err != nil {
		t.Fatalf("Failed to deserialize timed out saga: %v", err)
	}

	if deserialized.TimedOutAt == nil {
		t.Error("Expected TimedOutAt to be set")
	} else if !deserialized.TimedOutAt.Equal(timedOutAt) {
		t.Errorf("TimedOutAt mismatch: expected %v, got %v", timedOutAt, *deserialized.TimedOutAt)
	}

	if deserialized.CompletedAt != nil {
		t.Error("Expected CompletedAt to be nil for timed out saga")
	}
}

func TestMetadataHandling(t *testing.T) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())

	now := time.Now()
	metadata := map[string]interface{}{
		"user_id":     "user-123",
		"order_id":    "order-456",
		"retry_count": 3,
		"tags":        []string{"important", "high-priority"},
	}

	instance := &serializerTestSagaInstance{
		id:           "metadata-test",
		definitionID: "test-def",
		state:        saga.StateRunning,
		currentStep:  1,
		totalSteps:   3,
		createdAt:    now,
		updatedAt:    now,
		timeout:      30 * time.Minute,
		metadata:     metadata,
	}

	data, err := serializer.SerializeSagaInstance(instance)
	if err != nil {
		t.Fatalf("Failed to serialize saga with metadata: %v", err)
	}

	deserialized, err := serializer.DeserializeSagaInstance(data)
	if err != nil {
		t.Fatalf("Failed to deserialize saga with metadata: %v", err)
	}

	if deserialized.Metadata == nil {
		t.Fatal("Expected metadata to be preserved")
	}

	if deserialized.Metadata["user_id"] != "user-123" {
		t.Errorf("Expected user_id 'user-123', got '%v'", deserialized.Metadata["user_id"])
	}

	if deserialized.Metadata["order_id"] != "order-456" {
		t.Errorf("Expected order_id 'order-456', got '%v'", deserialized.Metadata["order_id"])
	}
}

func TestErrorHandling(t *testing.T) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())

	now := time.Now()
	sagaError := &saga.SagaError{
		Code:      "STEP_FAILED",
		Message:   "Step execution failed",
		Type:      saga.ErrorTypeService,
		Retryable: true,
		Timestamp: now,
	}

	instance := &serializerTestSagaInstance{
		id:           "error-test",
		definitionID: "test-def",
		state:        saga.StateFailed,
		currentStep:  2,
		totalSteps:   3,
		createdAt:    now,
		updatedAt:    now,
		timeout:      30 * time.Minute,
		error:        sagaError,
	}

	data, err := serializer.SerializeSagaInstance(instance)
	if err != nil {
		t.Fatalf("Failed to serialize saga with error: %v", err)
	}

	deserialized, err := serializer.DeserializeSagaInstance(data)
	if err != nil {
		t.Fatalf("Failed to deserialize saga with error: %v", err)
	}

	if deserialized.Error == nil {
		t.Fatal("Expected error to be preserved")
	}

	if deserialized.Error.Code != "STEP_FAILED" {
		t.Errorf("Expected error code 'STEP_FAILED', got '%s'", deserialized.Error.Code)
	}

	if deserialized.Error.Message != "Step execution failed" {
		t.Errorf("Expected error message 'Step execution failed', got '%s'", deserialized.Error.Message)
	}

	if !deserialized.Error.Retryable {
		t.Error("Expected error to be retryable")
	}
}

// Benchmark tests

func BenchmarkSerializeSagaInstance(b *testing.B) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())
	now := time.Now()
	instance := createSerializerTestSagaInstance(now)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.SerializeSagaInstance(instance)
		if err != nil {
			b.Fatalf("Failed to serialize: %v", err)
		}
	}
}

func BenchmarkDeserializeSagaInstance(b *testing.B) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())
	now := time.Now()
	instance := createSerializerTestSagaInstance(now)

	data, err := serializer.SerializeSagaInstance(instance)
	if err != nil {
		b.Fatalf("Failed to serialize: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.DeserializeSagaInstance(data)
		if err != nil {
			b.Fatalf("Failed to deserialize: %v", err)
		}
	}
}

func BenchmarkSerializeWithCompression(b *testing.B) {
	opts := &SerializationOptions{
		EnableCompression: true,
		CompressionLevel:  gzip.DefaultCompression,
		PrettyPrint:       false,
	}
	serializer := NewSagaSerializer(opts)
	now := time.Now()
	instance := createLargeSerializerSagaInstance(now)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.SerializeSagaInstance(instance)
		if err != nil {
			b.Fatalf("Failed to serialize: %v", err)
		}
	}
}

func BenchmarkSerializeStepState(b *testing.B) {
	serializer := NewSagaSerializer(DefaultSerializationOptions())
	now := time.Now()
	step := &saga.StepState{
		ID:        "step-1",
		SagaID:    "saga-1",
		StepIndex: 0,
		Name:      "Benchmark Step",
		State:     saga.StepStateCompleted,
		Attempts:  1,
		CreatedAt: now,
		Metadata:  map[string]interface{}{"key": "value"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.SerializeStepState(step)
		if err != nil {
			b.Fatalf("Failed to serialize step: %v", err)
		}
	}
}

// Helper functions and types

type serializerTestSagaInstance struct {
	id           string
	definitionID string
	state        saga.SagaState
	currentStep  int
	totalSteps   int
	createdAt    time.Time
	updatedAt    time.Time
	startTime    time.Time
	endTime      time.Time
	timeout      time.Duration
	metadata     map[string]interface{}
	traceID      string
	error        *saga.SagaError
	result       interface{}
}

func (t *serializerTestSagaInstance) GetID() string                       { return t.id }
func (t *serializerTestSagaInstance) GetDefinitionID() string             { return t.definitionID }
func (t *serializerTestSagaInstance) GetState() saga.SagaState            { return t.state }
func (t *serializerTestSagaInstance) GetCurrentStep() int                 { return t.currentStep }
func (t *serializerTestSagaInstance) GetTotalSteps() int                  { return t.totalSteps }
func (t *serializerTestSagaInstance) GetStartTime() time.Time             { return t.startTime }
func (t *serializerTestSagaInstance) GetEndTime() time.Time               { return t.endTime }
func (t *serializerTestSagaInstance) GetResult() interface{}              { return t.result }
func (t *serializerTestSagaInstance) GetError() *saga.SagaError           { return t.error }
func (t *serializerTestSagaInstance) GetCompletedSteps() int              { return t.currentStep }
func (t *serializerTestSagaInstance) GetCreatedAt() time.Time             { return t.createdAt }
func (t *serializerTestSagaInstance) GetUpdatedAt() time.Time             { return t.updatedAt }
func (t *serializerTestSagaInstance) GetTimeout() time.Duration           { return t.timeout }
func (t *serializerTestSagaInstance) GetMetadata() map[string]interface{} { return t.metadata }
func (t *serializerTestSagaInstance) GetTraceID() string                  { return t.traceID }
func (t *serializerTestSagaInstance) IsTerminal() bool                    { return t.state.IsTerminal() }
func (t *serializerTestSagaInstance) IsActive() bool                      { return t.state.IsActive() }

func createSerializerTestSagaInstance(now time.Time) *serializerTestSagaInstance {
	return &serializerTestSagaInstance{
		id:           "test-saga-1",
		definitionID: "test-definition",
		state:        saga.StateRunning,
		currentStep:  1,
		totalSteps:   3,
		createdAt:    now,
		updatedAt:    now,
		startTime:    now,
		timeout:      30 * time.Minute,
		metadata: map[string]interface{}{
			"test": "value",
		},
		traceID: "trace-123",
	}
}

func createLargeSerializerSagaInstance(now time.Time) *serializerTestSagaInstance {
	// Create large metadata to trigger compression
	metadata := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("%c%d", rune('a'+i%26), i)
		metadata[key] = strings.Repeat("data", 100)
	}

	return &serializerTestSagaInstance{
		id:           "large-saga-1",
		definitionID: "large-definition",
		state:        saga.StateRunning,
		currentStep:  1,
		totalSteps:   10,
		createdAt:    now,
		updatedAt:    now,
		startTime:    now,
		timeout:      1 * time.Hour,
		metadata:     metadata,
		traceID:      "trace-large-123",
	}
}
