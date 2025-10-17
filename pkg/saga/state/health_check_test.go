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
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// healthCheckMockSagaInstance implements saga.SagaInstance for health check testing.
type healthCheckMockSagaInstance struct {
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

func (m *healthCheckMockSagaInstance) GetID() string                       { return m.id }
func (m *healthCheckMockSagaInstance) GetDefinitionID() string             { return m.definitionID }
func (m *healthCheckMockSagaInstance) GetState() saga.SagaState            { return m.state }
func (m *healthCheckMockSagaInstance) GetCurrentStep() int                 { return m.currentStep }
func (m *healthCheckMockSagaInstance) GetStartTime() time.Time             { return m.startTime }
func (m *healthCheckMockSagaInstance) GetEndTime() time.Time               { return m.endTime }
func (m *healthCheckMockSagaInstance) GetResult() interface{}              { return m.result }
func (m *healthCheckMockSagaInstance) GetError() *saga.SagaError           { return m.err }
func (m *healthCheckMockSagaInstance) GetTotalSteps() int                  { return m.totalSteps }
func (m *healthCheckMockSagaInstance) GetCompletedSteps() int              { return m.completedSteps }
func (m *healthCheckMockSagaInstance) GetCreatedAt() time.Time             { return m.createdAt }
func (m *healthCheckMockSagaInstance) GetUpdatedAt() time.Time             { return m.updatedAt }
func (m *healthCheckMockSagaInstance) GetTimeout() time.Duration           { return m.timeout }
func (m *healthCheckMockSagaInstance) GetMetadata() map[string]interface{} { return m.metadata }
func (m *healthCheckMockSagaInstance) GetTraceID() string                  { return m.traceID }
func (m *healthCheckMockSagaInstance) IsTerminal() bool                    { return m.state.IsTerminal() }
func (m *healthCheckMockSagaInstance) IsActive() bool                      { return m.state.IsActive() }

// newHealthCheckMockSagaInstance creates a mock saga instance from SagaInstanceData for health check tests.
func newHealthCheckMockSagaInstance(data *saga.SagaInstanceData) *healthCheckMockSagaInstance {
	instance := &healthCheckMockSagaInstance{
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

	// Calculate completed steps from step states if available
	if data.StepStates != nil {
		for _, step := range data.StepStates {
			if step.State == saga.StepStateCompleted {
				instance.completedSteps++
			}
		}
	}

	return instance
}

// healthCheckMockStateStorage is a simple mock implementation of saga.StateStorage for health check testing.
type healthCheckMockStateStorage struct {
	mu    sync.RWMutex
	sagas map[string]saga.SagaInstance
	steps map[string][]*saga.StepState
}

func newHealthCheckMockStateStorage() *healthCheckMockStateStorage {
	return &healthCheckMockStateStorage{
		sagas: make(map[string]saga.SagaInstance),
		steps: make(map[string][]*saga.StepState),
	}
}

func (m *healthCheckMockStateStorage) SaveSaga(ctx context.Context, instance saga.SagaInstance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sagas[instance.GetID()] = instance
	return nil
}

func (m *healthCheckMockStateStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	instance, exists := m.sagas[sagaID]
	if !exists {
		return nil, errors.New("saga not found")
	}
	return instance, nil
}

func (m *healthCheckMockStateStorage) UpdateSagaState(ctx context.Context, sagaID string, state saga.SagaState, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	sagaInst, exists := m.sagas[sagaID]
	if !exists {
		return errors.New("saga not found")
	}
	if mockInst, ok := sagaInst.(*healthCheckMockSagaInstance); ok {
		mockInst.state = state
		mockInst.updatedAt = time.Now()
	}
	return nil
}

func (m *healthCheckMockStateStorage) DeleteSaga(ctx context.Context, sagaID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sagas, sagaID)
	delete(m.steps, sagaID)
	return nil
}

func (m *healthCheckMockStateStorage) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []saga.SagaInstance
	for _, instance := range m.sagas {
		result = append(result, instance)
	}
	return result, nil
}

func (m *healthCheckMockStateStorage) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return []saga.SagaInstance{}, nil
}

func (m *healthCheckMockStateStorage) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.steps[sagaID] = append(m.steps[sagaID], step)
	return nil
}

func (m *healthCheckMockStateStorage) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	steps, exists := m.steps[sagaID]
	if !exists {
		return []*saga.StepState{}, nil
	}
	return steps, nil
}

func (m *healthCheckMockStateStorage) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	return nil
}

func TestNewHealthChecker(t *testing.T) {
	memStorage := newHealthCheckMockStateStorage()
	logger := zap.NewNop()

	hc := NewHealthChecker(memStorage, logger)

	assert.NotNil(t, hc)
	assert.NotNil(t, hc.stateStorage)
	assert.NotNil(t, hc.consistencyChecker)
	assert.NotNil(t, hc.logger)
}

func TestHealthChecker_CheckSagaHealth_HealthySaga(t *testing.T) {
	ctx := context.Background()
	memStorage := newHealthCheckMockStateStorage()
	logger := zap.NewNop()
	hc := NewHealthChecker(memStorage, logger)

	// Create a healthy completed Saga
	now := time.Now()
	startTime := now.Add(-5 * time.Minute)
	endTime := now

	sagaData := &saga.SagaInstanceData{
		ID:           "saga-healthy-1",
		DefinitionID: "def-1",
		Name:         "Healthy Saga",
		State:        saga.StateCompleted,
		CurrentStep:  2,
		TotalSteps:   2,
		CreatedAt:    now.Add(-10 * time.Minute),
		UpdatedAt:    endTime,
		StartedAt:    &startTime,
		CompletedAt:  &endTime,
		Timeout:      30 * time.Minute,
		Version:      1,
	}

	sagaInstance := newHealthCheckMockSagaInstance(sagaData)
	err := memStorage.SaveSaga(ctx, sagaInstance)
	require.NoError(t, err)

	// Create step states
	step1 := &saga.StepState{
		ID:          "step-1",
		SagaID:      "saga-healthy-1",
		StepIndex:   0,
		Name:        "Step 1",
		State:       saga.StepStateCompleted,
		CreatedAt:   now.Add(-10 * time.Minute),
		StartedAt:   &startTime,
		CompletedAt: &[]time.Time{startTime.Add(1 * time.Minute)}[0],
	}

	step2 := &saga.StepState{
		ID:          "step-2",
		SagaID:      "saga-healthy-1",
		StepIndex:   1,
		Name:        "Step 2",
		State:       saga.StepStateCompleted,
		CreatedAt:   now.Add(-9 * time.Minute),
		StartedAt:   &[]time.Time{startTime.Add(1 * time.Minute)}[0],
		CompletedAt: &endTime,
	}

	err = memStorage.SaveStepState(ctx, "saga-healthy-1", step1)
	require.NoError(t, err)
	err = memStorage.SaveStepState(ctx, "saga-healthy-1", step2)
	require.NoError(t, err)

	// Perform health check
	report, err := hc.CheckSagaHealth(ctx, "saga-healthy-1")
	require.NoError(t, err)
	assert.NotNil(t, report)

	// Verify report
	assert.Equal(t, "saga-healthy-1", report.SagaID)
	assert.Equal(t, HealthStatusHealthy, report.Status)
	assert.Empty(t, report.Issues)
	assert.False(t, report.HasIssues())
	assert.False(t, report.HasCriticalIssues())
}

func TestHealthChecker_CheckSagaHealth_InconsistentState(t *testing.T) {
	ctx := context.Background()
	memStorage := newHealthCheckMockStateStorage()
	logger := zap.NewNop()
	hc := NewHealthChecker(memStorage, logger)

	// Create a Saga with inconsistent state (Completed but not all steps completed)
	now := time.Now()
	startTime := now.Add(-5 * time.Minute)
	endTime := now

	sagaData := &saga.SagaInstanceData{
		ID:           "saga-inconsistent-1",
		DefinitionID: "def-1",
		Name:         "Inconsistent Saga",
		State:        saga.StateCompleted,
		CurrentStep:  1,
		TotalSteps:   3,
		CreatedAt:    now.Add(-10 * time.Minute),
		UpdatedAt:    endTime,
		StartedAt:    &startTime,
		CompletedAt:  &endTime,
		Timeout:      30 * time.Minute,
		Version:      1,
	}

	sagaInstance := newHealthCheckMockSagaInstance(sagaData)
	err := memStorage.SaveSaga(ctx, sagaInstance)
	require.NoError(t, err)

	// Create only one completed step (but Saga claims to be completed)
	step1 := &saga.StepState{
		ID:          "step-1",
		SagaID:      "saga-inconsistent-1",
		StepIndex:   0,
		Name:        "Step 1",
		State:       saga.StepStateCompleted,
		CreatedAt:   now.Add(-10 * time.Minute),
		StartedAt:   &startTime,
		CompletedAt: &endTime,
	}

	err = memStorage.SaveStepState(ctx, "saga-inconsistent-1", step1)
	require.NoError(t, err)

	// Perform health check
	report, err := hc.CheckSagaHealth(ctx, "saga-inconsistent-1")
	require.NoError(t, err)
	assert.NotNil(t, report)

	// Verify report
	assert.Equal(t, "saga-inconsistent-1", report.SagaID)
	assert.Equal(t, HealthStatusUnhealthy, report.Status)
	assert.True(t, report.HasIssues())

	// Check for specific issues
	var hasIncompleteCompletedSaga bool
	var hasMissingSteps bool

	for _, issue := range report.Issues {
		if issue.Code == "INCOMPLETE_COMPLETED_SAGA" {
			hasIncompleteCompletedSaga = true
			assert.Equal(t, CategoryStateConsistency, issue.Category)
			assert.Equal(t, SeverityError, issue.Severity)
		}
		if issue.Code == "MISSING_STEP_STATES" {
			hasMissingSteps = true
			assert.Equal(t, CategoryCompleteness, issue.Category)
		}
	}

	assert.True(t, hasIncompleteCompletedSaga, "Should detect incomplete completed saga")
	assert.True(t, hasMissingSteps, "Should detect missing steps")
}

func TestHealthChecker_CheckSagaHealth_TimestampIssues(t *testing.T) {
	ctx := context.Background()
	memStorage := newHealthCheckMockStateStorage()
	logger := zap.NewNop()
	hc := NewHealthChecker(memStorage, logger)

	// Create a Saga with timestamp issues
	now := time.Now()
	startTime := now.Add(-5 * time.Minute)
	endTime := startTime.Add(-1 * time.Minute) // EndTime before StartTime!

	sagaData := &saga.SagaInstanceData{
		ID:           "saga-timestamp-1",
		DefinitionID: "def-1",
		Name:         "Timestamp Issue Saga",
		State:        saga.StateCompleted,
		CurrentStep:  1,
		TotalSteps:   1,
		CreatedAt:    now.Add(-10 * time.Minute),
		UpdatedAt:    now,
		StartedAt:    &startTime,
		CompletedAt:  &endTime,
		Timeout:      30 * time.Minute,
		Version:      1,
	}

	sagaInstance := newHealthCheckMockSagaInstance(sagaData)
	err := memStorage.SaveSaga(ctx, sagaInstance)
	require.NoError(t, err)

	// Create step with timestamp issue
	step1CompletedAt := startTime.Add(-2 * time.Minute) // Completed before started
	step1 := &saga.StepState{
		ID:          "step-1",
		SagaID:      "saga-timestamp-1",
		StepIndex:   0,
		Name:        "Step 1",
		State:       saga.StepStateCompleted,
		CreatedAt:   now.Add(-10 * time.Minute),
		StartedAt:   &startTime,
		CompletedAt: &step1CompletedAt,
	}

	err = memStorage.SaveStepState(ctx, "saga-timestamp-1", step1)
	require.NoError(t, err)

	// Perform health check
	report, err := hc.CheckSagaHealth(ctx, "saga-timestamp-1")
	require.NoError(t, err)
	assert.NotNil(t, report)

	// Verify report
	assert.Equal(t, "saga-timestamp-1", report.SagaID)
	assert.Equal(t, HealthStatusUnhealthy, report.Status)
	assert.True(t, report.HasIssues())

	// Check for timestamp issues
	var hasEndBeforeStart bool
	var hasStepCompletedBeforeStarted bool

	for _, issue := range report.Issues {
		if issue.Code == "INVALID_TIMESTAMP_ORDER_END_BEFORE_START" {
			hasEndBeforeStart = true
			assert.Equal(t, CategoryTimestamp, issue.Category)
			assert.Equal(t, SeverityError, issue.Severity)
		}
		if issue.Code == "STEP_COMPLETED_BEFORE_STARTED" {
			hasStepCompletedBeforeStarted = true
			assert.Equal(t, CategoryTimestamp, issue.Category)
			assert.Equal(t, SeverityError, issue.Severity)
		}
	}

	assert.True(t, hasEndBeforeStart, "Should detect end before start")
	assert.True(t, hasStepCompletedBeforeStarted, "Should detect step completed before started")
}

func TestHealthChecker_CheckSagaHealth_DataIntegrityIssues(t *testing.T) {
	ctx := context.Background()
	memStorage := newHealthCheckMockStateStorage()
	logger := zap.NewNop()
	hc := NewHealthChecker(memStorage, logger)

	// Create a Saga
	now := time.Now()
	sagaData := &saga.SagaInstanceData{
		ID:           "saga-integrity-1",
		DefinitionID: "def-1",
		Name:         "Data Integrity Issue Saga",
		State:        saga.StateRunning,
		CurrentStep:  0,
		TotalSteps:   2,
		CreatedAt:    now.Add(-10 * time.Minute),
		UpdatedAt:    now,
		Timeout:      30 * time.Minute,
		Version:      1,
	}

	sagaInstance := newHealthCheckMockSagaInstance(sagaData)
	err := memStorage.SaveSaga(ctx, sagaInstance)
	require.NoError(t, err)

	// Create step with wrong Saga ID (foreign key violation)
	step1 := &saga.StepState{
		ID:        "step-1",
		SagaID:    "wrong-saga-id", // Wrong Saga ID!
		StepIndex: 0,
		Name:      "Step 1",
		State:     saga.StepStateRunning,
		CreatedAt: now.Add(-5 * time.Minute),
	}

	err = memStorage.SaveStepState(ctx, "saga-integrity-1", step1)
	require.NoError(t, err)

	// Perform health check
	report, err := hc.CheckSagaHealth(ctx, "saga-integrity-1")
	require.NoError(t, err)
	assert.NotNil(t, report)

	// Verify report
	assert.True(t, report.HasIssues())
	assert.True(t, report.HasCriticalIssues())

	// Check for data integrity issue
	var hasSagaIDMismatch bool
	for _, issue := range report.Issues {
		if issue.Code == "STEP_SAGA_ID_MISMATCH" {
			hasSagaIDMismatch = true
			assert.Equal(t, CategoryDataIntegrity, issue.Category)
			assert.Equal(t, SeverityCritical, issue.Severity)
		}
	}

	assert.True(t, hasSagaIDMismatch, "Should detect saga ID mismatch")
}

func TestHealthReport_ToJSON(t *testing.T) {
	now := time.Now()
	report := &HealthReport{
		SagaID:    "saga-1",
		Status:    HealthStatusWarning,
		CheckTime: now,
		Duration:  100 * time.Millisecond,
		Issues: []HealthIssue{
			{
				Category:  CategoryStateConsistency,
				Severity:  SeverityWarning,
				Code:      "TEST_ISSUE",
				Message:   "Test issue",
				Timestamp: now,
			},
		},
		Summary: map[string]interface{}{
			"total_steps":     3,
			"completed_steps": 2,
		},
	}

	jsonStr, err := report.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonStr)
	assert.Contains(t, jsonStr, "saga-1")
	assert.Contains(t, jsonStr, "warning")
	assert.Contains(t, jsonStr, "TEST_ISSUE")
}

func TestHealthReport_GetIssuesBySeverity(t *testing.T) {
	now := time.Now()
	report := &HealthReport{
		SagaID:    "saga-1",
		Status:    HealthStatusUnhealthy,
		CheckTime: now,
		Issues: []HealthIssue{
			{Severity: SeverityCritical, Code: "ISSUE_1", Timestamp: now},
			{Severity: SeverityError, Code: "ISSUE_2", Timestamp: now},
			{Severity: SeverityWarning, Code: "ISSUE_3", Timestamp: now},
			{Severity: SeverityCritical, Code: "ISSUE_4", Timestamp: now},
		},
	}

	criticalIssues := report.GetIssuesBySeverity(SeverityCritical)
	assert.Len(t, criticalIssues, 2)

	errorIssues := report.GetIssuesBySeverity(SeverityError)
	assert.Len(t, errorIssues, 1)

	warningIssues := report.GetIssuesBySeverity(SeverityWarning)
	assert.Len(t, warningIssues, 1)
}

func TestHealthReport_GetIssuesByCategory(t *testing.T) {
	now := time.Now()
	report := &HealthReport{
		SagaID:    "saga-1",
		Status:    HealthStatusUnhealthy,
		CheckTime: now,
		Issues: []HealthIssue{
			{Category: CategoryStateConsistency, Code: "ISSUE_1", Timestamp: now},
			{Category: CategoryDataIntegrity, Code: "ISSUE_2", Timestamp: now},
			{Category: CategoryStateConsistency, Code: "ISSUE_3", Timestamp: now},
			{Category: CategoryTimestamp, Code: "ISSUE_4", Timestamp: now},
		},
	}

	stateIssues := report.GetIssuesByCategory(CategoryStateConsistency)
	assert.Len(t, stateIssues, 2)

	dataIssues := report.GetIssuesByCategory(CategoryDataIntegrity)
	assert.Len(t, dataIssues, 1)

	timestampIssues := report.GetIssuesByCategory(CategoryTimestamp)
	assert.Len(t, timestampIssues, 1)
}

func TestHealthChecker_CheckMultipleSagasHealth(t *testing.T) {
	ctx := context.Background()
	memStorage := newHealthCheckMockStateStorage()
	logger := zap.NewNop()
	hc := NewHealthChecker(memStorage, logger)

	now := time.Now()

	// Create multiple Sagas
	for i := 1; i <= 3; i++ {
		sagaData := &saga.SagaInstanceData{
			ID:           string(rune('A'+i-1)) + "-saga-" + string(rune('0'+i)),
			DefinitionID: "def-1",
			Name:         "Saga " + string(rune('0'+i)),
			State:        saga.StateRunning,
			CurrentStep:  0,
			TotalSteps:   2,
			CreatedAt:    now.Add(-10 * time.Minute),
			UpdatedAt:    now,
			Timeout:      30 * time.Minute,
			Version:      1,
		}

		sagaInstance := newHealthCheckMockSagaInstance(sagaData)
		err := memStorage.SaveSaga(ctx, sagaInstance)
		require.NoError(t, err)
	}

	sagaIDs := []string{"A-saga-1", "B-saga-2", "C-saga-3"}

	// Check multiple Sagas
	reports, err := hc.CheckMultipleSagasHealth(ctx, sagaIDs)
	require.NoError(t, err)
	assert.Len(t, reports, 3)

	for _, sagaID := range sagaIDs {
		assert.Contains(t, reports, sagaID)
		assert.NotNil(t, reports[sagaID])
	}
}

func TestSystemHealthReport_GetUnhealthySagas(t *testing.T) {
	now := time.Now()
	report := &SystemHealthReport{
		CheckTime: now,
		Reports: map[string]*HealthReport{
			"saga-1": {SagaID: "saga-1", Status: HealthStatusHealthy},
			"saga-2": {SagaID: "saga-2", Status: HealthStatusUnhealthy},
			"saga-3": {SagaID: "saga-3", Status: HealthStatusWarning},
			"saga-4": {SagaID: "saga-4", Status: HealthStatusUnhealthy},
		},
	}

	unhealthy := report.GetUnhealthySagas()
	assert.Len(t, unhealthy, 2)
	assert.Contains(t, unhealthy, "saga-2")
	assert.Contains(t, unhealthy, "saga-4")
}

func TestSystemHealthReport_GetSagasWithCriticalIssues(t *testing.T) {
	now := time.Now()
	report := &SystemHealthReport{
		CheckTime: now,
		Reports: map[string]*HealthReport{
			"saga-1": {
				SagaID: "saga-1",
				Status: HealthStatusHealthy,
				Issues: []HealthIssue{},
			},
			"saga-2": {
				SagaID: "saga-2",
				Status: HealthStatusUnhealthy,
				Issues: []HealthIssue{
					{Severity: SeverityCritical, Code: "CRITICAL_1", Timestamp: now},
				},
			},
			"saga-3": {
				SagaID: "saga-3",
				Status: HealthStatusWarning,
				Issues: []HealthIssue{
					{Severity: SeverityWarning, Code: "WARNING_1", Timestamp: now},
				},
			},
		},
	}

	critical := report.GetSagasWithCriticalIssues()
	assert.Len(t, critical, 1)
	assert.Contains(t, critical, "saga-2")
}
