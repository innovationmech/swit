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
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// consistencyMockSagaInstance implements saga.SagaInstance for consistency testing.
type consistencyMockSagaInstance struct {
	id             string
	definitionID   string
	state          saga.SagaState
	currentStep    int
	totalSteps     int
	startTime      time.Time
	endTime        time.Time
	result         interface{}
	err            *saga.SagaError
	completedSteps int
	createdAt      time.Time
	updatedAt      time.Time
	timeout        time.Duration
	metadata       map[string]interface{}
	traceID        string
}

func (m *consistencyMockSagaInstance) GetID() string                       { return m.id }
func (m *consistencyMockSagaInstance) GetDefinitionID() string             { return m.definitionID }
func (m *consistencyMockSagaInstance) GetState() saga.SagaState            { return m.state }
func (m *consistencyMockSagaInstance) GetCurrentStep() int                 { return m.currentStep }
func (m *consistencyMockSagaInstance) GetStartTime() time.Time             { return m.startTime }
func (m *consistencyMockSagaInstance) GetEndTime() time.Time               { return m.endTime }
func (m *consistencyMockSagaInstance) GetResult() interface{}              { return m.result }
func (m *consistencyMockSagaInstance) GetError() *saga.SagaError           { return m.err }
func (m *consistencyMockSagaInstance) GetTotalSteps() int                  { return m.totalSteps }
func (m *consistencyMockSagaInstance) GetCompletedSteps() int              { return m.completedSteps }
func (m *consistencyMockSagaInstance) GetCreatedAt() time.Time             { return m.createdAt }
func (m *consistencyMockSagaInstance) GetUpdatedAt() time.Time             { return m.updatedAt }
func (m *consistencyMockSagaInstance) GetTimeout() time.Duration           { return m.timeout }
func (m *consistencyMockSagaInstance) GetMetadata() map[string]interface{} { return m.metadata }
func (m *consistencyMockSagaInstance) GetTraceID() string                  { return m.traceID }
func (m *consistencyMockSagaInstance) IsTerminal() bool                    { return m.state.IsTerminal() }
func (m *consistencyMockSagaInstance) IsActive() bool                      { return m.state.IsActive() }

// newConsistencyMockSagaInstance creates a mock saga instance from SagaInstanceData for consistency tests.
func newConsistencyMockSagaInstance(data *saga.SagaInstanceData) *consistencyMockSagaInstance {
	instance := &consistencyMockSagaInstance{
		id:           data.ID,
		definitionID: data.DefinitionID,
		state:        data.State,
		currentStep:  data.CurrentStep,
		totalSteps:   data.TotalSteps,
		createdAt:    data.CreatedAt,
		updatedAt:    data.UpdatedAt,
		timeout:      data.Timeout,
		metadata:     data.Metadata,
	}

	if data.StartedAt != nil {
		instance.startTime = *data.StartedAt
	}
	if data.CompletedAt != nil {
		instance.endTime = *data.CompletedAt
	}

	return instance
}

func TestNewConsistencyChecker(t *testing.T) {
	logger := zap.NewNop()
	cc := NewConsistencyChecker(logger)

	assert.NotNil(t, cc)
	assert.NotNil(t, cc.logger)
}

func TestConsistencyChecker_CheckConsistency_HealthySaga(t *testing.T) {
	cc := NewConsistencyChecker(zap.NewNop())

	now := time.Now()
	startTime := now.Add(-5 * time.Minute)
	endTime := now

	sagaData := &saga.SagaInstanceData{
		ID:           "saga-1",
		DefinitionID: "def-1",
		State:        saga.StateCompleted,
		CurrentStep:  2,
		TotalSteps:   2,
		CreatedAt:    now.Add(-10 * time.Minute),
		UpdatedAt:    endTime,
		StartedAt:    &startTime,
		CompletedAt:  &endTime,
		Version:      1,
	}

	sagaInstance := newConsistencyMockSagaInstance(sagaData)

	stepStates := []*saga.StepState{
		{
			ID:          "step-1",
			SagaID:      "saga-1",
			StepIndex:   0,
			Name:        "Step 1",
			State:       saga.StepStateCompleted,
			CreatedAt:   now.Add(-10 * time.Minute),
			StartedAt:   &startTime,
			CompletedAt: &[]time.Time{startTime.Add(1 * time.Minute)}[0],
		},
		{
			ID:          "step-2",
			SagaID:      "saga-1",
			StepIndex:   1,
			Name:        "Step 2",
			State:       saga.StepStateCompleted,
			CreatedAt:   now.Add(-9 * time.Minute),
			StartedAt:   &[]time.Time{startTime.Add(1 * time.Minute)}[0],
			CompletedAt: &endTime,
		},
	}

	issues := cc.CheckConsistency(sagaInstance, stepStates)
	assert.Empty(t, issues, "Healthy saga should have no issues")
}

func TestConsistencyChecker_CheckRequiredFields(t *testing.T) {
	cc := NewConsistencyChecker(zap.NewNop())

	tests := []struct {
		name          string
		sagaData      *saga.SagaInstanceData
		expectedCodes []string
	}{
		{
			name: "Missing Saga ID",
			sagaData: &saga.SagaInstanceData{
				ID:           "",
				DefinitionID: "def-1",
				State:        saga.StateRunning,
			},
			expectedCodes: []string{"MISSING_SAGA_ID"},
		},
		{
			name: "Missing Definition ID",
			sagaData: &saga.SagaInstanceData{
				ID:           "saga-1",
				DefinitionID: "",
				State:        saga.StateRunning,
			},
			expectedCodes: []string{"MISSING_DEFINITION_ID"},
		},
		{
			name: "All required fields present",
			sagaData: &saga.SagaInstanceData{
				ID:           "saga-1",
				DefinitionID: "def-1",
				State:        saga.StateRunning,
			},
			expectedCodes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sagaInstance := newConsistencyMockSagaInstance(tt.sagaData)
			issues := cc.checkRequiredFields(sagaInstance)

			if len(tt.expectedCodes) == 0 {
				assert.Empty(t, issues)
			} else {
				assert.Len(t, issues, len(tt.expectedCodes))
				for i, code := range tt.expectedCodes {
					assert.Equal(t, code, issues[i].Code)
				}
			}
		})
	}
}

func TestConsistencyChecker_CheckStateConsistency(t *testing.T) {
	cc := NewConsistencyChecker(zap.NewNop())

	tests := []struct {
		name          string
		sagaState     saga.SagaState
		totalSteps    int
		stepStates    []*saga.StepState
		expectedCodes []string
	}{
		{
			name:       "Completed Saga with incomplete steps",
			sagaState:  saga.StateCompleted,
			totalSteps: 3,
			stepStates: []*saga.StepState{
				{StepIndex: 0, State: saga.StepStateCompleted},
			},
			expectedCodes: []string{"INCOMPLETE_COMPLETED_SAGA"},
		},
		{
			name:       "Running Saga with no running steps",
			sagaState:  saga.StateRunning,
			totalSteps: 2,
			stepStates: []*saga.StepState{
				{StepIndex: 0, State: saga.StepStateCompleted},
			},
			expectedCodes: []string{"RUNNING_SAGA_NO_RUNNING_STEPS"},
		},
		{
			name:       "Compensating Saga with no compensating steps",
			sagaState:  saga.StateCompensating,
			totalSteps: 2,
			stepStates: []*saga.StepState{
				{StepIndex: 0, State: saga.StepStateCompleted},
			},
			expectedCodes: []string{"COMPENSATING_SAGA_NO_COMPENSATING_STEPS"},
		},
		{
			name:          "Failed Saga with no failed steps",
			sagaState:     saga.StateFailed,
			totalSteps:    2,
			stepStates:    []*saga.StepState{},
			expectedCodes: []string{"FAILED_SAGA_NO_FAILED_STEPS"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sagaData := &saga.SagaInstanceData{
				ID:           "saga-1",
				DefinitionID: "def-1",
				State:        tt.sagaState,
				TotalSteps:   tt.totalSteps,
			}
			sagaInstance := newConsistencyMockSagaInstance(sagaData)

			issues := cc.checkStateConsistency(sagaInstance, tt.stepStates)

			foundCodes := make(map[string]bool)
			for _, issue := range issues {
				foundCodes[issue.Code] = true
			}

			for _, expectedCode := range tt.expectedCodes {
				assert.True(t, foundCodes[expectedCode], "Expected to find issue code: %s", expectedCode)
			}
		})
	}
}

func TestConsistencyChecker_CheckTimestampLogic(t *testing.T) {
	cc := NewConsistencyChecker(zap.NewNop())

	now := time.Now()

	tests := []struct {
		name          string
		sagaData      *saga.SagaInstanceData
		stepStates    []*saga.StepState
		expectedCodes []string
	}{
		{
			name: "UpdatedAt before CreatedAt",
			sagaData: &saga.SagaInstanceData{
				ID:        "saga-1",
				State:     saga.StateRunning,
				CreatedAt: now,
				UpdatedAt: now.Add(-1 * time.Minute),
			},
			stepStates:    []*saga.StepState{},
			expectedCodes: []string{"INVALID_TIMESTAMP_ORDER_UPDATED_BEFORE_CREATED"},
		},
		{
			name: "StartTime before CreatedAt",
			sagaData: &saga.SagaInstanceData{
				ID:        "saga-1",
				State:     saga.StateRunning,
				CreatedAt: now,
				UpdatedAt: now,
				StartedAt: &[]time.Time{now.Add(-1 * time.Minute)}[0],
			},
			stepStates:    []*saga.StepState{},
			expectedCodes: []string{"INVALID_TIMESTAMP_ORDER_START_BEFORE_CREATED"},
		},
		{
			name: "EndTime before StartTime",
			sagaData: &saga.SagaInstanceData{
				ID:          "saga-1",
				State:       saga.StateCompleted,
				CreatedAt:   now.Add(-10 * time.Minute),
				UpdatedAt:   now,
				StartedAt:   &[]time.Time{now.Add(-5 * time.Minute)}[0],
				CompletedAt: &[]time.Time{now.Add(-6 * time.Minute)}[0],
			},
			stepStates:    []*saga.StepState{},
			expectedCodes: []string{"INVALID_TIMESTAMP_ORDER_END_BEFORE_START"},
		},
		{
			name: "Step started before Saga created",
			sagaData: &saga.SagaInstanceData{
				ID:        "saga-1",
				State:     saga.StateRunning,
				CreatedAt: now,
				UpdatedAt: now,
			},
			stepStates: []*saga.StepState{
				{
					ID:        "step-1",
					Name:      "Step 1",
					StepIndex: 0,
					StartedAt: &[]time.Time{now.Add(-1 * time.Minute)}[0],
				},
			},
			expectedCodes: []string{"STEP_STARTED_BEFORE_SAGA_CREATED"},
		},
		{
			name: "Step completed before started",
			sagaData: &saga.SagaInstanceData{
				ID:        "saga-1",
				State:     saga.StateRunning,
				CreatedAt: now.Add(-10 * time.Minute),
				UpdatedAt: now,
			},
			stepStates: []*saga.StepState{
				{
					ID:          "step-1",
					Name:        "Step 1",
					StepIndex:   0,
					StartedAt:   &[]time.Time{now.Add(-5 * time.Minute)}[0],
					CompletedAt: &[]time.Time{now.Add(-6 * time.Minute)}[0],
				},
			},
			expectedCodes: []string{"STEP_COMPLETED_BEFORE_STARTED"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sagaInstance := newConsistencyMockSagaInstance(tt.sagaData)
			issues := cc.checkTimestampLogic(sagaInstance, tt.stepStates)

			foundCodes := make(map[string]bool)
			for _, issue := range issues {
				foundCodes[issue.Code] = true
			}

			for _, expectedCode := range tt.expectedCodes {
				assert.True(t, foundCodes[expectedCode], "Expected to find issue code: %s", expectedCode)
			}
		})
	}
}

func TestConsistencyChecker_CheckTerminalStateCompleteness(t *testing.T) {
	cc := NewConsistencyChecker(zap.NewNop())

	now := time.Now()

	tests := []struct {
		name          string
		sagaData      *saga.SagaInstanceData
		stepStates    []*saga.StepState
		expectedCodes []string
	}{
		{
			name: "Terminal Saga missing EndTime",
			sagaData: &saga.SagaInstanceData{
				ID:         "saga-1",
				State:      saga.StateCompleted,
				TotalSteps: 1,
				CreatedAt:  now.Add(-10 * time.Minute),
				UpdatedAt:  now,
			},
			stepStates: []*saga.StepState{
				{StepIndex: 0, State: saga.StepStateCompleted},
			},
			expectedCodes: []string{"TERMINAL_SAGA_MISSING_END_TIME"},
		},
		{
			name: "Completed Saga with incomplete steps",
			sagaData: &saga.SagaInstanceData{
				ID:          "saga-1",
				State:       saga.StateCompleted,
				TotalSteps:  3,
				CreatedAt:   now.Add(-10 * time.Minute),
				UpdatedAt:   now,
				CompletedAt: &now,
			},
			stepStates: []*saga.StepState{
				{StepIndex: 0, State: saga.StepStateCompleted},
			},
			expectedCodes: []string{"COMPLETED_SAGA_INCOMPLETE_STEPS"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sagaInstance := newConsistencyMockSagaInstance(tt.sagaData)
			issues := cc.checkTerminalStateCompleteness(sagaInstance, tt.stepStates)

			foundCodes := make(map[string]bool)
			for _, issue := range issues {
				foundCodes[issue.Code] = true
			}

			for _, expectedCode := range tt.expectedCodes {
				assert.True(t, foundCodes[expectedCode], "Expected to find issue code: %s", expectedCode)
			}
		})
	}
}

func TestConsistencyChecker_CheckDataIntegrity(t *testing.T) {
	cc := NewConsistencyChecker(zap.NewNop())

	tests := []struct {
		name          string
		sagaID        string
		stepStates    []*saga.StepState
		expectedCodes []string
	}{
		{
			name:   "Step Saga ID mismatch",
			sagaID: "saga-1",
			stepStates: []*saga.StepState{
				{
					ID:        "step-1",
					SagaID:    "wrong-saga-id",
					Name:      "Step 1",
					StepIndex: 0,
				},
			},
			expectedCodes: []string{"STEP_SAGA_ID_MISMATCH"},
		},
		{
			name:   "Missing Step ID",
			sagaID: "saga-1",
			stepStates: []*saga.StepState{
				{
					ID:        "",
					SagaID:    "saga-1",
					Name:      "Step 1",
					StepIndex: 0,
				},
			},
			expectedCodes: []string{"MISSING_STEP_ID"},
		},
		{
			name:   "Duplicate step index",
			sagaID: "saga-1",
			stepStates: []*saga.StepState{
				{
					ID:        "step-1",
					SagaID:    "saga-1",
					StepIndex: 0,
				},
				{
					ID:        "step-2",
					SagaID:    "saga-1",
					StepIndex: 0,
				},
			},
			expectedCodes: []string{"DUPLICATE_STEP_INDEX"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sagaData := &saga.SagaInstanceData{
				ID:    tt.sagaID,
				State: saga.StateRunning,
			}
			sagaInstance := newConsistencyMockSagaInstance(sagaData)

			issues := cc.checkDataIntegrity(sagaInstance, tt.stepStates)

			foundCodes := make(map[string]bool)
			for _, issue := range issues {
				foundCodes[issue.Code] = true
			}

			for _, expectedCode := range tt.expectedCodes {
				assert.True(t, foundCodes[expectedCode], "Expected to find issue code: %s", expectedCode)
			}
		})
	}
}

func TestConsistencyChecker_CheckOrphanedSteps(t *testing.T) {
	cc := NewConsistencyChecker(zap.NewNop())

	sagaData := &saga.SagaInstanceData{
		ID:         "saga-1",
		State:      saga.StateRunning,
		TotalSteps: 2,
	}
	sagaInstance := newConsistencyMockSagaInstance(sagaData)

	stepStates := []*saga.StepState{
		{
			ID:        "step-1",
			SagaID:    "saga-1",
			Name:      "Step 1",
			StepIndex: 0,
		},
		{
			ID:        "step-3",
			SagaID:    "saga-1",
			Name:      "Step 3",
			StepIndex: 5, // Beyond total steps!
		},
	}

	issues := cc.checkOrphanedSteps(sagaInstance, stepStates)

	assert.Len(t, issues, 1)
	assert.Equal(t, "ORPHANED_STEP", issues[0].Code)
	assert.Equal(t, CategoryDataIntegrity, issues[0].Category)
}

func TestConsistencyChecker_CheckMissingSteps(t *testing.T) {
	cc := NewConsistencyChecker(zap.NewNop())

	sagaData := &saga.SagaInstanceData{
		ID:         "saga-1",
		State:      saga.StateRunning,
		TotalSteps: 5,
	}
	sagaInstance := newConsistencyMockSagaInstance(sagaData)

	// Only provide steps 0 and 2, missing 1, 3, 4
	stepStates := []*saga.StepState{
		{
			ID:        "step-1",
			SagaID:    "saga-1",
			StepIndex: 0,
		},
		{
			ID:        "step-3",
			SagaID:    "saga-1",
			StepIndex: 2,
		},
	}

	issues := cc.checkMissingSteps(sagaInstance, stepStates)

	assert.Len(t, issues, 1)
	assert.Equal(t, "MISSING_STEP_STATES", issues[0].Code)
	assert.Equal(t, CategoryCompleteness, issues[0].Category)

	// Check details
	details := issues[0].Details
	assert.Equal(t, 5, details["total_steps"])
	assert.Equal(t, 2, details["existing_steps"])
	missingIndices := details["missing_indices"].([]int)
	assert.Contains(t, missingIndices, 1)
	assert.Contains(t, missingIndices, 3)
	assert.Contains(t, missingIndices, 4)
}

func TestConsistencyChecker_CheckStepExecutionOrder(t *testing.T) {
	cc := NewConsistencyChecker(zap.NewNop())

	stepStates := []*saga.StepState{
		{
			ID:        "step-1",
			Name:      "Step 1",
			StepIndex: 0,
			State:     saga.StepStateCompleted,
		},
		{
			ID:        "step-2",
			Name:      "Step 2",
			StepIndex: 1,
			State:     saga.StepStatePending,
		},
		{
			ID:        "step-3",
			Name:      "Step 3",
			StepIndex: 2,
			State:     saga.StepStateCompleted, // Completed but previous step is not!
		},
	}

	issues := cc.checkStepExecutionOrder(stepStates)

	assert.NotEmpty(t, issues)
	foundNonSequential := false
	for _, issue := range issues {
		if issue.Code == "NON_SEQUENTIAL_STEP_COMPLETION" {
			foundNonSequential = true
			assert.Equal(t, CategoryStateConsistency, issue.Category)
		}
	}
	assert.True(t, foundNonSequential, "Should detect non-sequential completion")
}
