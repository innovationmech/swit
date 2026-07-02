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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestNewMsgpackSerializer tests the creation of a MessagePack serializer.
func TestNewMsgpackSerializer(t *testing.T) {
	tests := []struct {
		name   string
		config *SerializerConfig
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name: "custom config",
			config: &SerializerConfig{
				Format:            SerializationFormatMsgpack,
				EnableCompression: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := NewMsgpackSerializer(tt.config)
			if serializer == nil {
				t.Fatal("expected non-nil serializer")
			}
			if got := serializer.Format(); got != SerializationFormatMsgpack {
				t.Errorf("Format() = %v, want %v", got, SerializationFormatMsgpack)
			}
		})
	}
}

// TestMsgpackSerializer_NilAndEmptyInputs tests error handling for nil/empty inputs.
func TestMsgpackSerializer_NilAndEmptyInputs(t *testing.T) {
	serializer := NewMsgpackSerializer(nil)

	if _, err := serializer.SerializeSaga(nil); err == nil {
		t.Error("SerializeSaga(nil) expected error but got nil")
	}
	if _, err := serializer.SerializeStepState(nil); err == nil {
		t.Error("SerializeStepState(nil) expected error but got nil")
	}
	if _, err := serializer.SerializeStepStates(nil); err == nil {
		t.Error("SerializeStepStates(nil) expected error but got nil")
	}
	if _, err := serializer.DeserializeSaga(nil); err == nil {
		t.Error("DeserializeSaga(nil) expected error but got nil")
	}
	if _, err := serializer.DeserializeStepState([]byte{}); err == nil {
		t.Error("DeserializeStepState(empty) expected error but got nil")
	}
	if _, err := serializer.DeserializeStepStates([]byte{}); err == nil {
		t.Error("DeserializeStepStates(empty) expected error but got nil")
	}
	if _, err := serializer.DeserializeSaga([]byte("not msgpack")); err == nil {
		t.Error("DeserializeSaga(invalid) expected error but got nil")
	}
}

// TestMsgpackSerializer_SerializeDeserializeSaga tests round-trip serialization
// of SagaInstanceData with MessagePack.
func TestMsgpackSerializer_SerializeDeserializeSaga(t *testing.T) {
	now := time.Now()
	startedAt := now.Add(-1 * time.Hour)
	completedAt := now

	tests := []struct {
		name   string
		config *SerializerConfig
		saga   *saga.SagaInstanceData
	}{
		{
			name: "basic saga round-trip",
			config: &SerializerConfig{
				Format: SerializationFormatMsgpack,
			},
			saga: &saga.SagaInstanceData{
				ID:           "saga-1",
				DefinitionID: "def-1",
				Name:         "Test Saga",
				Description:  "A test saga",
				State:        saga.StateCompleted,
				CurrentStep:  3,
				TotalSteps:   3,
				CreatedAt:    now,
				UpdatedAt:    now,
				StartedAt:    &startedAt,
				CompletedAt:  &completedAt,
				Timeout:      30 * time.Minute,
				TraceID:      "trace-123",
				SpanID:       "span-456",
				Version:      2,
			},
		},
		{
			name: "compressed saga round-trip",
			config: &SerializerConfig{
				Format:            SerializationFormatMsgpack,
				EnableCompression: true,
			},
			saga: &saga.SagaInstanceData{
				ID:    "saga-2",
				State: saga.StateRunning,
				InitialData: map[string]interface{}{
					"key1": "value1",
					"key2": int64(123),
					"key3": true,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "saga with error and nested metadata",
			config: &SerializerConfig{
				Format: SerializationFormatMsgpack,
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
						"level2": "deep value",
					},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := NewMsgpackSerializer(tt.config)

			data, err := serializer.SerializeSaga(tt.saga)
			if err != nil {
				t.Fatalf("SerializeSaga() error = %v", err)
			}
			if len(data) == 0 {
				t.Fatal("expected non-empty data")
			}

			deserialized, err := serializer.DeserializeSaga(data)
			if err != nil {
				t.Fatalf("DeserializeSaga() error = %v", err)
			}

			if deserialized.ID != tt.saga.ID {
				t.Errorf("ID = %v, want %v", deserialized.ID, tt.saga.ID)
			}
			if deserialized.State != tt.saga.State {
				t.Errorf("State = %v, want %v", deserialized.State, tt.saga.State)
			}
			if deserialized.CurrentStep != tt.saga.CurrentStep {
				t.Errorf("CurrentStep = %v, want %v", deserialized.CurrentStep, tt.saga.CurrentStep)
			}
			if deserialized.Timeout != tt.saga.Timeout {
				t.Errorf("Timeout = %v, want %v", deserialized.Timeout, tt.saga.Timeout)
			}
			if deserialized.Version != tt.saga.Version {
				t.Errorf("Version = %v, want %v", deserialized.Version, tt.saga.Version)
			}
			if !deserialized.CreatedAt.Equal(tt.saga.CreatedAt) {
				t.Errorf("CreatedAt = %v, want %v", deserialized.CreatedAt, tt.saga.CreatedAt)
			}
			if tt.saga.StartedAt != nil {
				if deserialized.StartedAt == nil || !deserialized.StartedAt.Equal(*tt.saga.StartedAt) {
					t.Errorf("StartedAt = %v, want %v", deserialized.StartedAt, tt.saga.StartedAt)
				}
			}
			if tt.saga.Error != nil {
				if deserialized.Error == nil {
					t.Fatal("expected non-nil error after round-trip")
				}
				if deserialized.Error.Code != tt.saga.Error.Code {
					t.Errorf("Error.Code = %v, want %v", deserialized.Error.Code, tt.saga.Error.Code)
				}
				if deserialized.Error.Message != tt.saga.Error.Message {
					t.Errorf("Error.Message = %v, want %v", deserialized.Error.Message, tt.saga.Error.Message)
				}
			}
		})
	}
}

// TestMsgpackSerializer_SerializeDeserializeStepState tests round-trip
// serialization of a single StepState with MessagePack.
func TestMsgpackSerializer_SerializeDeserializeStepState(t *testing.T) {
	now := time.Now()
	startedAt := now.Add(-30 * time.Minute)

	step := &saga.StepState{
		ID:          "step-1",
		SagaID:      "saga-1",
		StepIndex:   0,
		Name:        "Test Step",
		State:       saga.StepStateCompensated,
		Attempts:    2,
		MaxAttempts: 3,
		CreatedAt:   now,
		StartedAt:   &startedAt,
		InputData: map[string]interface{}{
			"param1": "value1",
		},
		CompensationState: &saga.CompensationState{
			State:     saga.CompensationStateCompleted,
			Attempts:  1,
			StartedAt: &startedAt,
		},
	}

	for _, compression := range []bool{false, true} {
		name := "no compression"
		if compression {
			name = "with compression"
		}
		t.Run(name, func(t *testing.T) {
			serializer := NewMsgpackSerializer(&SerializerConfig{
				Format:            SerializationFormatMsgpack,
				EnableCompression: compression,
			})

			data, err := serializer.SerializeStepState(step)
			if err != nil {
				t.Fatalf("SerializeStepState() error = %v", err)
			}

			deserialized, err := serializer.DeserializeStepState(data)
			if err != nil {
				t.Fatalf("DeserializeStepState() error = %v", err)
			}

			if deserialized.ID != step.ID {
				t.Errorf("ID = %v, want %v", deserialized.ID, step.ID)
			}
			if deserialized.State != step.State {
				t.Errorf("State = %v, want %v", deserialized.State, step.State)
			}
			if deserialized.Attempts != step.Attempts {
				t.Errorf("Attempts = %v, want %v", deserialized.Attempts, step.Attempts)
			}
			if deserialized.CompensationState == nil {
				t.Fatal("expected non-nil compensation state after round-trip")
			}
			if deserialized.CompensationState.State != step.CompensationState.State {
				t.Errorf("CompensationState.State = %v, want %v",
					deserialized.CompensationState.State, step.CompensationState.State)
			}
		})
	}
}

// TestMsgpackSerializer_SerializeDeserializeStepStates tests round-trip
// serialization of multiple StepStates with MessagePack.
func TestMsgpackSerializer_SerializeDeserializeStepStates(t *testing.T) {
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

	serializer := NewMsgpackSerializer(&SerializerConfig{
		Format: SerializationFormatMsgpack,
	})

	data, err := serializer.SerializeStepStates(steps)
	if err != nil {
		t.Fatalf("SerializeStepStates() error = %v", err)
	}

	deserialized, err := serializer.DeserializeStepStates(data)
	if err != nil {
		t.Fatalf("DeserializeStepStates() error = %v", err)
	}

	if len(deserialized) != len(steps) {
		t.Fatalf("got %d steps, want %d", len(deserialized), len(steps))
	}

	for i := range steps {
		if deserialized[i].ID != steps[i].ID {
			t.Errorf("step[%d].ID = %v, want %v", i, deserialized[i].ID, steps[i].ID)
		}
		if deserialized[i].State != steps[i].State {
			t.Errorf("step[%d].State = %v, want %v", i, deserialized[i].State, steps[i].State)
		}
	}

	// Empty (non-nil) slice should round-trip as well.
	emptyData, err := serializer.SerializeStepStates([]*saga.StepState{})
	if err != nil {
		t.Fatalf("SerializeStepStates(empty) error = %v", err)
	}
	emptySteps, err := serializer.DeserializeStepStates(emptyData)
	if err != nil {
		t.Fatalf("DeserializeStepStates(empty) error = %v", err)
	}
	if len(emptySteps) != 0 {
		t.Errorf("got %d steps, want 0", len(emptySteps))
	}
}

// TestNewSerializer_MsgpackRoundTrip verifies the factory-produced msgpack
// serializer round-trips saga data end to end.
func TestNewSerializer_MsgpackRoundTrip(t *testing.T) {
	serializer, err := NewSerializer(&SerializerConfig{
		Format: SerializationFormatMsgpack,
	})
	if err != nil {
		t.Fatalf("NewSerializer() error = %v", err)
	}
	if serializer.Format() != SerializationFormatMsgpack {
		t.Fatalf("Format() = %v, want %v", serializer.Format(), SerializationFormatMsgpack)
	}

	now := time.Now()
	original := &saga.SagaInstanceData{
		ID:           "saga-factory",
		DefinitionID: "def-1",
		Name:         "Factory Saga",
		State:        saga.StateRunning,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	data, err := serializer.SerializeSaga(original)
	if err != nil {
		t.Fatalf("SerializeSaga() error = %v", err)
	}

	deserialized, err := serializer.DeserializeSaga(data)
	if err != nil {
		t.Fatalf("DeserializeSaga() error = %v", err)
	}
	if deserialized.ID != original.ID || deserialized.State != original.State {
		t.Errorf("round-trip mismatch: got %+v, want %+v", deserialized, original)
	}
}
