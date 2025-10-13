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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestNewJSONSerializer tests the creation of a JSON serializer.
func TestNewJSONSerializer(t *testing.T) {
	tests := []struct {
		name   string
		config *SerializerConfig
		want   SerializationFormat
	}{
		{
			name:   "default config",
			config: nil,
			want:   SerializationFormatJSON,
		},
		{
			name: "custom config",
			config: &SerializerConfig{
				Format:            SerializationFormatJSON,
				EnableCompression: true,
				PrettyPrint:       true,
			},
			want: SerializationFormatJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := NewJSONSerializer(tt.config)
			if serializer == nil {
				t.Fatal("expected non-nil serializer")
			}
			if got := serializer.Format(); got != tt.want {
				t.Errorf("Format() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNewSerializer tests the factory function for creating serializers.
func TestNewSerializer(t *testing.T) {
	tests := []struct {
		name    string
		config  *SerializerConfig
		wantErr bool
	}{
		{
			name:    "default config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "JSON format",
			config: &SerializerConfig{
				Format: SerializationFormatJSON,
			},
			wantErr: false,
		},
		{
			name: "protobuf format (not implemented)",
			config: &SerializerConfig{
				Format: SerializationFormatProtobuf,
			},
			wantErr: true,
		},
		{
			name: "msgpack format (not implemented)",
			config: &SerializerConfig{
				Format: SerializationFormatMsgpack,
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			config: &SerializerConfig{
				Format: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer, err := NewSerializer(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				if serializer != nil {
					t.Error("expected nil serializer on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if serializer == nil {
					t.Error("expected non-nil serializer")
				}
			}
		})
	}
}

// TestJSONSerializer_SerializeSaga tests serialization of SagaInstanceData.
func TestJSONSerializer_SerializeSaga(t *testing.T) {
	now := time.Now()
	startedAt := now.Add(-1 * time.Hour)

	tests := []struct {
		name    string
		config  *SerializerConfig
		saga    *saga.SagaInstanceData
		wantErr bool
	}{
		{
			name:    "nil saga",
			config:  DefaultSerializerConfig(),
			saga:    nil,
			wantErr: true,
		},
		{
			name:   "basic saga",
			config: DefaultSerializerConfig(),
			saga: &saga.SagaInstanceData{
				ID:           "saga-1",
				DefinitionID: "def-1",
				Name:         "Test Saga",
				Description:  "A test saga",
				State:        saga.StateRunning,
				CurrentStep:  1,
				TotalSteps:   3,
				CreatedAt:    now,
				UpdatedAt:    now,
				StartedAt:    &startedAt,
				Timeout:      30 * time.Minute,
				TraceID:      "trace-123",
				SpanID:       "span-456",
			},
			wantErr: false,
		},
		{
			name: "saga with compression",
			config: &SerializerConfig{
				Format:            SerializationFormatJSON,
				EnableCompression: true,
				PrettyPrint:       false,
			},
			saga: &saga.SagaInstanceData{
				ID:           "saga-2",
				DefinitionID: "def-2",
				Name:         "Compressed Saga",
				State:        saga.StateCompleted,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			wantErr: false,
		},
		{
			name: "saga with pretty print",
			config: &SerializerConfig{
				Format:            SerializationFormatJSON,
				EnableCompression: false,
				PrettyPrint:       true,
			},
			saga: &saga.SagaInstanceData{
				ID:           "saga-3",
				DefinitionID: "def-3",
				Name:         "Pretty Saga",
				State:        saga.StatePending,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			wantErr: false,
		},
		{
			name:   "saga with initial data",
			config: DefaultSerializerConfig(),
			saga: &saga.SagaInstanceData{
				ID:           "saga-4",
				DefinitionID: "def-4",
				Name:         "Data Saga",
				State:        saga.StateRunning,
				CreatedAt:    now,
				UpdatedAt:    now,
				InitialData: map[string]interface{}{
					"userId": "user-123",
					"amount": 100.50,
				},
				CurrentData: map[string]interface{}{
					"step1Result": "success",
				},
			},
			wantErr: false,
		},
		{
			name:   "saga with error",
			config: DefaultSerializerConfig(),
			saga: &saga.SagaInstanceData{
				ID:           "saga-5",
				DefinitionID: "def-5",
				Name:         "Failed Saga",
				State:        saga.StateFailed,
				CreatedAt:    now,
				UpdatedAt:    now,
				Error: &saga.SagaError{
					Code:      "ERR_STEP_FAILED",
					Message:   "Step 2 failed",
					Type:      saga.ErrorTypeTimeout,
					Retryable: true,
					Timestamp: now,
				},
			},
			wantErr: false,
		},
		{
			name:   "saga with metadata",
			config: DefaultSerializerConfig(),
			saga: &saga.SagaInstanceData{
				ID:           "saga-6",
				DefinitionID: "def-6",
				Name:         "Metadata Saga",
				State:        saga.StateRunning,
				CreatedAt:    now,
				UpdatedAt:    now,
				Metadata: map[string]interface{}{
					"environment": "production",
					"priority":    "high",
					"tags":        []string{"critical", "payment"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := NewJSONSerializer(tt.config)
			data, err := serializer.SerializeSaga(tt.saga)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				if data != nil {
					t.Error("expected nil data on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if data == nil {
					t.Error("expected non-nil data")
				}
				if len(data) == 0 {
					t.Error("expected non-empty data")
				}
			}
		})
	}
}

// TestJSONSerializer_DeserializeSaga tests deserialization of SagaInstanceData.
func TestJSONSerializer_DeserializeSaga(t *testing.T) {
	now := time.Now()
	serializer := NewJSONSerializer(DefaultSerializerConfig())

	originalSaga := &saga.SagaInstanceData{
		ID:           "saga-1",
		DefinitionID: "def-1",
		Name:         "Test Saga",
		Description:  "A test saga",
		State:        saga.StateRunning,
		CurrentStep:  1,
		TotalSteps:   3,
		CreatedAt:    now,
		UpdatedAt:    now,
		Timeout:      30 * time.Minute,
		TraceID:      "trace-123",
		SpanID:       "span-456",
		InitialData: map[string]interface{}{
			"userId": "user-123",
		},
		Metadata: map[string]interface{}{
			"env": "test",
		},
	}

	data, err := serializer.SerializeSaga(originalSaga)
	if err != nil {
		t.Fatalf("failed to serialize saga: %v", err)
	}

	tests := []struct {
		name    string
		config  *SerializerConfig
		data    []byte
		wantErr bool
	}{
		{
			name:    "empty data",
			config:  DefaultSerializerConfig(),
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "nil data",
			config:  DefaultSerializerConfig(),
			data:    nil,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			config:  DefaultSerializerConfig(),
			data:    []byte("invalid json"),
			wantErr: true,
		},
		{
			name:    "valid serialized saga",
			config:  DefaultSerializerConfig(),
			data:    data,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := NewJSONSerializer(tt.config)
			saga, err := serializer.DeserializeSaga(tt.data)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				if saga != nil {
					t.Error("expected nil saga on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if saga == nil {
					t.Fatal("expected non-nil saga")
				}
				// Verify some fields
				if saga.ID != originalSaga.ID {
					t.Errorf("ID = %v, want %v", saga.ID, originalSaga.ID)
				}
				if saga.Name != originalSaga.Name {
					t.Errorf("Name = %v, want %v", saga.Name, originalSaga.Name)
				}
				if saga.State != originalSaga.State {
					t.Errorf("State = %v, want %v", saga.State, originalSaga.State)
				}
			}
		})
	}
}

// TestJSONSerializer_SerializeDeserializeSaga tests round-trip serialization.
func TestJSONSerializer_SerializeDeserializeSaga(t *testing.T) {
	now := time.Now()
	startedAt := now.Add(-1 * time.Hour)
	completedAt := now

	tests := []struct {
		name   string
		config *SerializerConfig
		saga   *saga.SagaInstanceData
	}{
		{
			name:   "basic saga round-trip",
			config: DefaultSerializerConfig(),
			saga: &saga.SagaInstanceData{
				ID:           "saga-1",
				DefinitionID: "def-1",
				Name:         "Test Saga",
				State:        saga.StateCompleted,
				CurrentStep:  3,
				TotalSteps:   3,
				CreatedAt:    now,
				UpdatedAt:    now,
				StartedAt:    &startedAt,
				CompletedAt:  &completedAt,
			},
		},
		{
			name: "compressed saga round-trip",
			config: &SerializerConfig{
				Format:            SerializationFormatJSON,
				EnableCompression: true,
			},
			saga: &saga.SagaInstanceData{
				ID:    "saga-2",
				State: saga.StateRunning,
				InitialData: map[string]interface{}{
					"key1": "value1",
					"key2": 123,
					"key3": true,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "saga with nested data",
			config: &SerializerConfig{
				Format:            SerializationFormatJSON,
				EnableCompression: false,
				PrettyPrint:       true,
			},
			saga: &saga.SagaInstanceData{
				ID:    "saga-3",
				State: saga.StateFailed,
				Error: &saga.SagaError{
					Code:      "ERR_TEST",
					Message:   "Test error",
					Type:      saga.ErrorTypeValidation,
					Retryable: false,
					Timestamp: now,
					Details: map[string]interface{}{
						"field":  "username",
						"reason": "invalid format",
					},
				},
				Metadata: map[string]interface{}{
					"nested": map[string]interface{}{
						"level2": map[string]interface{}{
							"level3": "deep value",
						},
					},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := NewJSONSerializer(tt.config)

			// Serialize
			data, err := serializer.SerializeSaga(tt.saga)
			if err != nil {
				t.Fatalf("SerializeSaga() error = %v", err)
			}

			// Deserialize
			deserialized, err := serializer.DeserializeSaga(data)
			if err != nil {
				t.Fatalf("DeserializeSaga() error = %v", err)
			}

			// Compare key fields
			if deserialized.ID != tt.saga.ID {
				t.Errorf("ID = %v, want %v", deserialized.ID, tt.saga.ID)
			}
			if deserialized.State != tt.saga.State {
				t.Errorf("State = %v, want %v", deserialized.State, tt.saga.State)
			}
			if deserialized.CurrentStep != tt.saga.CurrentStep {
				t.Errorf("CurrentStep = %v, want %v", deserialized.CurrentStep, tt.saga.CurrentStep)
			}
		})
	}
}

// TestJSONSerializer_SerializeStepState tests serialization of StepState.
func TestJSONSerializer_SerializeStepState(t *testing.T) {
	now := time.Now()
	startedAt := now.Add(-30 * time.Minute)

	tests := []struct {
		name    string
		config  *SerializerConfig
		step    *saga.StepState
		wantErr bool
	}{
		{
			name:    "nil step",
			config:  DefaultSerializerConfig(),
			step:    nil,
			wantErr: true,
		},
		{
			name:   "basic step state",
			config: DefaultSerializerConfig(),
			step: &saga.StepState{
				ID:          "step-1",
				SagaID:      "saga-1",
				StepIndex:   0,
				Name:        "Step 1",
				State:       saga.StepStateCompleted,
				Attempts:    1,
				MaxAttempts: 3,
				CreatedAt:   now,
				StartedAt:   &startedAt,
			},
			wantErr: false,
		},
		{
			name:   "step state with data",
			config: DefaultSerializerConfig(),
			step: &saga.StepState{
				ID:        "step-2",
				SagaID:    "saga-1",
				StepIndex: 1,
				Name:      "Step 2",
				State:     saga.StepStateRunning,
				InputData: map[string]interface{}{
					"param1": "value1",
				},
				OutputData: map[string]interface{}{
					"result": "success",
				},
				CreatedAt: now,
			},
			wantErr: false,
		},
		{
			name:   "step state with error",
			config: DefaultSerializerConfig(),
			step: &saga.StepState{
				ID:        "step-3",
				SagaID:    "saga-1",
				StepIndex: 2,
				Name:      "Step 3",
				State:     saga.StepStateFailed,
				Attempts:  3,
				Error: &saga.SagaError{
					Code:      "ERR_TIMEOUT",
					Message:   "Step timed out",
					Type:      saga.ErrorTypeTimeout,
					Retryable: true,
					Timestamp: now,
				},
				CreatedAt: now,
			},
			wantErr: false,
		},
		{
			name:   "step state with compensation",
			config: DefaultSerializerConfig(),
			step: &saga.StepState{
				ID:        "step-4",
				SagaID:    "saga-1",
				StepIndex: 3,
				Name:      "Step 4",
				State:     saga.StepStateCompensated,
				CompensationState: &saga.CompensationState{
					State:     saga.CompensationStateCompleted,
					Attempts:  1,
					StartedAt: &startedAt,
				},
				CreatedAt: now,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := NewJSONSerializer(tt.config)
			data, err := serializer.SerializeStepState(tt.step)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(data) == 0 {
					t.Error("expected non-empty data")
				}
			}
		})
	}
}

// TestJSONSerializer_DeserializeStepState tests deserialization of StepState.
func TestJSONSerializer_DeserializeStepState(t *testing.T) {
	now := time.Now()
	serializer := NewJSONSerializer(DefaultSerializerConfig())

	originalStep := &saga.StepState{
		ID:          "step-1",
		SagaID:      "saga-1",
		StepIndex:   0,
		Name:        "Test Step",
		State:       saga.StepStateCompleted,
		Attempts:    1,
		MaxAttempts: 3,
		CreatedAt:   now,
	}

	data, err := serializer.SerializeStepState(originalStep)
	if err != nil {
		t.Fatalf("failed to serialize step: %v", err)
	}

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			data:    []byte("invalid"),
			wantErr: true,
		},
		{
			name:    "valid data",
			data:    data,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := serializer.DeserializeStepState(tt.data)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if step == nil {
					t.Fatal("expected non-nil step")
				}
				if step.ID != originalStep.ID {
					t.Errorf("ID = %v, want %v", step.ID, originalStep.ID)
				}
				if step.State != originalStep.State {
					t.Errorf("State = %v, want %v", step.State, originalStep.State)
				}
			}
		})
	}
}

// TestJSONSerializer_SerializeStepStates tests serialization of multiple StepStates.
func TestJSONSerializer_SerializeStepStates(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		config  *SerializerConfig
		steps   []*saga.StepState
		wantErr bool
	}{
		{
			name:    "nil steps",
			config:  DefaultSerializerConfig(),
			steps:   nil,
			wantErr: true,
		},
		{
			name:    "empty steps",
			config:  DefaultSerializerConfig(),
			steps:   []*saga.StepState{},
			wantErr: false,
		},
		{
			name:   "single step",
			config: DefaultSerializerConfig(),
			steps: []*saga.StepState{
				{
					ID:        "step-1",
					SagaID:    "saga-1",
					StepIndex: 0,
					State:     saga.StepStateCompleted,
					CreatedAt: now,
				},
			},
			wantErr: false,
		},
		{
			name:   "multiple steps",
			config: DefaultSerializerConfig(),
			steps: []*saga.StepState{
				{
					ID:        "step-1",
					SagaID:    "saga-1",
					StepIndex: 0,
					State:     saga.StepStateCompleted,
					CreatedAt: now,
				},
				{
					ID:        "step-2",
					SagaID:    "saga-1",
					StepIndex: 1,
					State:     saga.StepStateRunning,
					CreatedAt: now,
				},
				{
					ID:        "step-3",
					SagaID:    "saga-1",
					StepIndex: 2,
					State:     saga.StepStatePending,
					CreatedAt: now,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := NewJSONSerializer(tt.config)
			data, err := serializer.SerializeStepStates(tt.steps)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(data) == 0 {
					t.Error("expected non-empty data")
				}
			}
		})
	}
}

// TestJSONSerializer_DeserializeStepStates tests deserialization of multiple StepStates.
func TestJSONSerializer_DeserializeStepStates(t *testing.T) {
	now := time.Now()
	serializer := NewJSONSerializer(DefaultSerializerConfig())

	originalSteps := []*saga.StepState{
		{
			ID:        "step-1",
			SagaID:    "saga-1",
			StepIndex: 0,
			State:     saga.StepStateCompleted,
			CreatedAt: now,
		},
		{
			ID:        "step-2",
			SagaID:    "saga-1",
			StepIndex: 1,
			State:     saga.StepStateRunning,
			CreatedAt: now,
		},
	}

	data, err := serializer.SerializeStepStates(originalSteps)
	if err != nil {
		t.Fatalf("failed to serialize steps: %v", err)
	}

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			data:    []byte("invalid"),
			wantErr: true,
		},
		{
			name:    "valid data",
			data:    data,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := serializer.DeserializeStepStates(tt.data)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if steps == nil {
					t.Fatal("expected non-nil steps")
				}
				if len(steps) != len(originalSteps) {
					t.Errorf("got %d steps, want %d", len(steps), len(originalSteps))
				}
			}
		})
	}
}

// TestJSONSerializer_SerializeDeserializeStepStates tests round-trip for multiple StepStates.
func TestJSONSerializer_SerializeDeserializeStepStates(t *testing.T) {
	now := time.Now()

	steps := []*saga.StepState{
		{
			ID:          "step-1",
			SagaID:      "saga-1",
			StepIndex:   0,
			Name:        "Step 1",
			State:       saga.StepStateCompleted,
			Attempts:    1,
			MaxAttempts: 3,
			CreatedAt:   now,
		},
		{
			ID:        "step-2",
			SagaID:    "saga-1",
			StepIndex: 1,
			Name:      "Step 2",
			State:     saga.StepStateRunning,
			InputData: map[string]interface{}{
				"key": "value",
			},
			CreatedAt: now,
		},
	}

	serializer := NewJSONSerializer(DefaultSerializerConfig())

	// Serialize
	data, err := serializer.SerializeStepStates(steps)
	if err != nil {
		t.Fatalf("SerializeStepStates() error = %v", err)
	}

	// Deserialize
	deserialized, err := serializer.DeserializeStepStates(data)
	if err != nil {
		t.Fatalf("DeserializeStepStates() error = %v", err)
	}

	// Compare
	if len(deserialized) != len(steps) {
		t.Errorf("got %d steps, want %d", len(deserialized), len(steps))
	}

	for i := range steps {
		if deserialized[i].ID != steps[i].ID {
			t.Errorf("step[%d].ID = %v, want %v", i, deserialized[i].ID, steps[i].ID)
		}
		if deserialized[i].State != steps[i].State {
			t.Errorf("step[%d].State = %v, want %v", i, deserialized[i].State, steps[i].State)
		}
	}
}

// TestJSONSerializer_Compression tests compression and decompression.
func TestJSONSerializer_Compression(t *testing.T) {
	now := time.Now()
	saga := &saga.SagaInstanceData{
		ID:          "saga-1",
		Name:        "Compression Test",
		Description: "Testing compression with a longer description that should compress well",
		State:       saga.StateRunning,
		CreatedAt:   now,
		UpdatedAt:   now,
		Metadata: map[string]interface{}{
			"key1": "This is a long value that should compress well",
			"key2": "Another long value with repeated words words words",
			"key3": "Yet another long value for testing compression",
		},
	}

	// Serialize without compression
	serializerNoCompression := NewJSONSerializer(&SerializerConfig{
		Format:            SerializationFormatJSON,
		EnableCompression: false,
	})
	uncompressedData, err := serializerNoCompression.SerializeSaga(saga)
	if err != nil {
		t.Fatalf("failed to serialize without compression: %v", err)
	}

	// Serialize with compression
	serializerWithCompression := NewJSONSerializer(&SerializerConfig{
		Format:            SerializationFormatJSON,
		EnableCompression: true,
	})
	compressedData, err := serializerWithCompression.SerializeSaga(saga)
	if err != nil {
		t.Fatalf("failed to serialize with compression: %v", err)
	}

	t.Logf("Uncompressed size: %d bytes", len(uncompressedData))
	t.Logf("Compressed size: %d bytes", len(compressedData))

	// Verify compression actually reduced size for this specific case
	// Note: For small data, compression might increase size due to overhead
	// So we just verify it doesn't panic and can be deserialized

	// Deserialize compressed data
	deserializedSaga, err := serializerWithCompression.DeserializeSaga(compressedData)
	if err != nil {
		t.Fatalf("failed to deserialize compressed data: %v", err)
	}

	// Verify data integrity
	if deserializedSaga.ID != saga.ID {
		t.Errorf("ID = %v, want %v", deserializedSaga.ID, saga.ID)
	}
	if deserializedSaga.Name != saga.Name {
		t.Errorf("Name = %v, want %v", deserializedSaga.Name, saga.Name)
	}
}

// TestDefaultSerializerConfig tests the default configuration.
func TestDefaultSerializerConfig(t *testing.T) {
	config := DefaultSerializerConfig()
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if config.Format != SerializationFormatJSON {
		t.Errorf("Format = %v, want %v", config.Format, SerializationFormatJSON)
	}
	if config.EnableCompression {
		t.Error("EnableCompression should be false by default")
	}
	if config.PrettyPrint {
		t.Error("PrettyPrint should be false by default")
	}
}

// BenchmarkJSONSerializer_SerializeSaga benchmarks saga serialization.
func BenchmarkJSONSerializer_SerializeSaga(b *testing.B) {
	now := time.Now()
	saga := &saga.SagaInstanceData{
		ID:           "saga-1",
		DefinitionID: "def-1",
		Name:         "Benchmark Saga",
		State:        saga.StateRunning,
		CreatedAt:    now,
		UpdatedAt:    now,
		InitialData: map[string]interface{}{
			"userId": "user-123",
			"amount": 100.50,
		},
	}

	serializer := NewJSONSerializer(DefaultSerializerConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.SerializeSaga(saga)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJSONSerializer_DeserializeSaga benchmarks saga deserialization.
func BenchmarkJSONSerializer_DeserializeSaga(b *testing.B) {
	now := time.Now()
	saga := &saga.SagaInstanceData{
		ID:           "saga-1",
		DefinitionID: "def-1",
		Name:         "Benchmark Saga",
		State:        saga.StateRunning,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	serializer := NewJSONSerializer(DefaultSerializerConfig())
	data, err := serializer.SerializeSaga(saga)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.DeserializeSaga(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJSONSerializer_SerializeWithCompression benchmarks compression.
func BenchmarkJSONSerializer_SerializeWithCompression(b *testing.B) {
	now := time.Now()
	saga := &saga.SagaInstanceData{
		ID:          "saga-1",
		Name:        "Compression Benchmark",
		Description: "Long description for compression testing " + string(make([]byte, 1000)),
		State:       saga.StateRunning,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	serializer := NewJSONSerializer(&SerializerConfig{
		Format:            SerializationFormatJSON,
		EnableCompression: true,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.SerializeSaga(saga)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestJSONSerializer_StepStateCompression tests compression for step states.
func TestJSONSerializer_StepStateCompression(t *testing.T) {
	now := time.Now()
	step := &saga.StepState{
		ID:        "step-1",
		SagaID:    "saga-1",
		StepIndex: 0,
		Name:      "Test Step with Long Name for Compression",
		State:     saga.StepStateCompleted,
		Metadata: map[string]interface{}{
			"key1": "Long value for compression testing",
			"key2": "Another long value with repeated data data data",
		},
		CreatedAt: now,
	}

	serializerWithCompression := NewJSONSerializer(&SerializerConfig{
		Format:            SerializationFormatJSON,
		EnableCompression: true,
	})

	// Serialize with compression
	compressedData, err := serializerWithCompression.SerializeStepState(step)
	if err != nil {
		t.Fatalf("failed to serialize with compression: %v", err)
	}

	// Deserialize compressed data
	deserializedStep, err := serializerWithCompression.DeserializeStepState(compressedData)
	if err != nil {
		t.Fatalf("failed to deserialize compressed data: %v", err)
	}

	// Verify data integrity
	if deserializedStep.ID != step.ID {
		t.Errorf("ID = %v, want %v", deserializedStep.ID, step.ID)
	}
	if deserializedStep.Name != step.Name {
		t.Errorf("Name = %v, want %v", deserializedStep.Name, step.Name)
	}
}

// TestJSONSerializer_StepStatesCompression tests compression for multiple step states.
func TestJSONSerializer_StepStatesCompression(t *testing.T) {
	now := time.Now()
	steps := []*saga.StepState{
		{
			ID:        "step-1",
			SagaID:    "saga-1",
			StepIndex: 0,
			Name:      "Step 1",
			State:     saga.StepStateCompleted,
			CreatedAt: now,
		},
		{
			ID:        "step-2",
			SagaID:    "saga-1",
			StepIndex: 1,
			Name:      "Step 2",
			State:     saga.StepStateRunning,
			CreatedAt: now,
		},
	}

	serializerWithCompression := NewJSONSerializer(&SerializerConfig{
		Format:            SerializationFormatJSON,
		EnableCompression: true,
	})

	// Serialize with compression
	compressedData, err := serializerWithCompression.SerializeStepStates(steps)
	if err != nil {
		t.Fatalf("failed to serialize with compression: %v", err)
	}

	// Deserialize compressed data
	deserializedSteps, err := serializerWithCompression.DeserializeStepStates(compressedData)
	if err != nil {
		t.Fatalf("failed to deserialize compressed data: %v", err)
	}

	// Verify data integrity
	if len(deserializedSteps) != len(steps) {
		t.Errorf("got %d steps, want %d", len(deserializedSteps), len(steps))
	}
}

// TestJSONSerializer_CompressionErrors tests error handling in compression.
func TestJSONSerializer_CompressionErrors(t *testing.T) {
	serializer := NewJSONSerializer(&SerializerConfig{
		Format:            SerializationFormatJSON,
		EnableCompression: true,
	})

	// Test decompressing invalid gzip data
	invalidGzipData := []byte("not a gzip data")
	_, err := serializer.DeserializeSaga(invalidGzipData)
	if err == nil {
		t.Error("expected error when decompressing invalid gzip data")
	}

	// Test decompressing truncated gzip data
	truncatedData := []byte{0x1f, 0x8b, 0x08} // Valid gzip header but truncated
	_, err = serializer.DeserializeSaga(truncatedData)
	if err == nil {
		t.Error("expected error when decompressing truncated gzip data")
	}
}

// TestJSONSerializer_StepStateDecompressionError tests error handling in step state decompression.
func TestJSONSerializer_StepStateDecompressionError(t *testing.T) {
	serializer := NewJSONSerializer(&SerializerConfig{
		Format:            SerializationFormatJSON,
		EnableCompression: true,
	})

	// Test decompressing invalid gzip data for step state
	invalidGzipData := []byte("not a gzip data")
	_, err := serializer.DeserializeStepState(invalidGzipData)
	if err == nil {
		t.Error("expected error when decompressing invalid gzip data for step state")
	}

	// Test decompressing invalid gzip data for step states
	_, err = serializer.DeserializeStepStates(invalidGzipData)
	if err == nil {
		t.Error("expected error when decompressing invalid gzip data for step states")
	}
}

// TestJSONSerializer_PrettyPrintStepState tests pretty print for step states.
func TestJSONSerializer_PrettyPrintStepState(t *testing.T) {
	now := time.Now()
	step := &saga.StepState{
		ID:        "step-1",
		SagaID:    "saga-1",
		StepIndex: 0,
		Name:      "Test Step",
		State:     saga.StepStateCompleted,
		CreatedAt: now,
	}

	serializer := NewJSONSerializer(&SerializerConfig{
		Format:      SerializationFormatJSON,
		PrettyPrint: true,
	})

	// Serialize with pretty print
	data, err := serializer.SerializeStepState(step)
	if err != nil {
		t.Fatalf("failed to serialize with pretty print: %v", err)
	}

	// Verify it contains newlines (indicating pretty print)
	if !bytes.Contains(data, []byte("\n")) {
		t.Error("expected pretty-printed JSON to contain newlines")
	}

	// Verify it can be deserialized
	_, err = serializer.DeserializeStepState(data)
	if err != nil {
		t.Errorf("failed to deserialize pretty-printed data: %v", err)
	}
}

// TestJSONSerializer_PrettyPrintStepStates tests pretty print for multiple step states.
func TestJSONSerializer_PrettyPrintStepStates(t *testing.T) {
	now := time.Now()
	steps := []*saga.StepState{
		{
			ID:        "step-1",
			SagaID:    "saga-1",
			StepIndex: 0,
			State:     saga.StepStateCompleted,
			CreatedAt: now,
		},
	}

	serializer := NewJSONSerializer(&SerializerConfig{
		Format:      SerializationFormatJSON,
		PrettyPrint: true,
	})

	// Serialize with pretty print
	data, err := serializer.SerializeStepStates(steps)
	if err != nil {
		t.Fatalf("failed to serialize with pretty print: %v", err)
	}

	// Verify it contains newlines
	if !bytes.Contains(data, []byte("\n")) {
		t.Error("expected pretty-printed JSON to contain newlines")
	}

	// Verify it can be deserialized
	_, err = serializer.DeserializeStepStates(data)
	if err != nil {
		t.Errorf("failed to deserialize pretty-printed data: %v", err)
	}
}
