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

package saga

import (
	"context"
	"testing"
	"time"
)

// Mock implementations for testing interfaces

type MockSagaCoordinator struct {
	sagas   map[string]SagaInstance
	metrics *CoordinatorMetrics
	closed  bool
	healthy bool
}

type MockSagaInstance struct {
	id             string
	definitionID   string
	state          SagaState
	currentStep    int
	startTime      time.Time
	endTime        time.Time
	result         interface{}
	err            *SagaError
	totalSteps     int
	completedSteps int
	createdAt      time.Time
	updatedAt      time.Time
	timeout        time.Duration
	metadata       map[string]interface{}
	traceID        string
}

type MockSagaDefinition struct {
	id                   string
	name                 string
	description          string
	steps                []SagaStep
	timeout              time.Duration
	retryPolicy          RetryPolicy
	compensationStrategy CompensationStrategy
	metadata             map[string]interface{}
}

type MockSagaStep struct {
	id          string
	name        string
	description string
	timeout     time.Duration
	retryPolicy RetryPolicy
	metadata    map[string]interface{}
}

type MockStateStorage struct {
	sagas      map[string]SagaInstance
	stepStates map[string][]*StepState
}

type MockEventPublisher struct {
	events []*SagaEvent
	closed bool
}

type MockRetryPolicy struct {
	maxAttempts int
	shouldRetry func(err error, attempt int) bool
	getDelay    func(attempt int) time.Duration
}

type MockCompensationStrategy struct {
	shouldCompensate     func(err *SagaError) bool
	getCompensationOrder func(completedSteps []SagaStep) []SagaStep
	getTimeout           func() time.Duration
}

// MockSagaCoordinator implementation

func NewMockSagaCoordinator() *MockSagaCoordinator {
	return &MockSagaCoordinator{
		sagas:   make(map[string]SagaInstance),
		metrics: &CoordinatorMetrics{},
		closed:  false,
		healthy: true,
	}
}

func (m *MockSagaCoordinator) StartSaga(ctx context.Context, definition SagaDefinition, initialData interface{}) (SagaInstance, error) {
	if m.closed {
		return nil, NewCoordinatorStoppedError()
	}

	instance := &MockSagaInstance{
		id:           "test-saga-id",
		definitionID: definition.GetID(),
		state:        StateRunning,
		currentStep:  0,
		startTime:    time.Now(),
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
		totalSteps:   len(definition.GetSteps()),
		timeout:      definition.GetTimeout(),
		metadata:     make(map[string]interface{}),
		traceID:      "test-trace-id",
	}

	m.sagas[instance.GetID()] = instance
	m.metrics.TotalSagas++
	m.metrics.ActiveSagas++

	return instance, nil
}

func (m *MockSagaCoordinator) GetSagaInstance(sagaID string) (SagaInstance, error) {
	if m.closed {
		return nil, NewCoordinatorStoppedError()
	}

	instance, exists := m.sagas[sagaID]
	if !exists {
		return nil, NewSagaNotFoundError(sagaID)
	}

	return instance, nil
}

func (m *MockSagaCoordinator) CancelSaga(ctx context.Context, sagaID string, reason string) error {
	if m.closed {
		return NewCoordinatorStoppedError()
	}

	instance, exists := m.sagas[sagaID]
	if !exists {
		return NewSagaNotFoundError(sagaID)
	}

	mockInstance := instance.(*MockSagaInstance)
	mockInstance.state = StateCancelled
	now := time.Now()
	mockInstance.endTime = now
	mockInstance.updatedAt = now

	m.metrics.ActiveSagas--
	m.metrics.CancelledSagas++

	return nil
}

func (m *MockSagaCoordinator) GetActiveSagas(filter *SagaFilter) ([]SagaInstance, error) {
	if m.closed {
		return nil, NewCoordinatorStoppedError()
	}

	var activeSagas []SagaInstance
	for _, instance := range m.sagas {
		if instance.IsActive() {
			activeSagas = append(activeSagas, instance)
		}
	}

	return activeSagas, nil
}

func (m *MockSagaCoordinator) GetMetrics() *CoordinatorMetrics {
	return m.metrics
}

func (m *MockSagaCoordinator) HealthCheck(ctx context.Context) error {
	if m.closed {
		return NewCoordinatorStoppedError()
	}
	if !m.healthy {
		return NewSagaError("UNHEALTHY", "Coordinator is unhealthy", ErrorTypeSystem, false)
	}
	return nil
}

func (m *MockSagaCoordinator) Close() error {
	m.closed = true
	return nil
}

// MockSagaInstance implementation

func (m *MockSagaInstance) GetID() string {
	return m.id
}

func (m *MockSagaInstance) GetDefinitionID() string {
	return m.definitionID
}

func (m *MockSagaInstance) GetState() SagaState {
	return m.state
}

func (m *MockSagaInstance) GetCurrentStep() int {
	return m.currentStep
}

func (m *MockSagaInstance) GetStartTime() time.Time {
	return m.startTime
}

func (m *MockSagaInstance) GetEndTime() time.Time {
	return m.endTime
}

func (m *MockSagaInstance) GetResult() interface{} {
	return m.result
}

func (m *MockSagaInstance) GetError() *SagaError {
	return m.err
}

func (m *MockSagaInstance) GetTotalSteps() int {
	return m.totalSteps
}

func (m *MockSagaInstance) GetCompletedSteps() int {
	return m.completedSteps
}

func (m *MockSagaInstance) GetCreatedAt() time.Time {
	return m.createdAt
}

func (m *MockSagaInstance) GetUpdatedAt() time.Time {
	return m.updatedAt
}

func (m *MockSagaInstance) GetTimeout() time.Duration {
	return m.timeout
}

func (m *MockSagaInstance) GetMetadata() map[string]interface{} {
	return m.metadata
}

func (m *MockSagaInstance) GetTraceID() string {
	return m.traceID
}

func (m *MockSagaInstance) IsTerminal() bool {
	return m.state.IsTerminal()
}

func (m *MockSagaInstance) IsActive() bool {
	return m.state.IsActive()
}

// MockSagaDefinition implementation

func (m *MockSagaDefinition) GetID() string {
	return m.id
}

func (m *MockSagaDefinition) GetName() string {
	return m.name
}

func (m *MockSagaDefinition) GetDescription() string {
	return m.description
}

func (m *MockSagaDefinition) GetSteps() []SagaStep {
	return m.steps
}

func (m *MockSagaDefinition) GetTimeout() time.Duration {
	return m.timeout
}

func (m *MockSagaDefinition) GetRetryPolicy() RetryPolicy {
	return m.retryPolicy
}

func (m *MockSagaDefinition) GetCompensationStrategy() CompensationStrategy {
	return m.compensationStrategy
}

func (m *MockSagaDefinition) Validate() error {
	if m.id == "" {
		return NewValidationError("Definition ID cannot be empty")
	}
	if len(m.steps) == 0 {
		return NewValidationError("Definition must have at least one step")
	}
	return nil
}

func (m *MockSagaDefinition) GetMetadata() map[string]interface{} {
	return m.metadata
}

// MockSagaStep implementation

func (m *MockSagaStep) GetID() string {
	return m.id
}

func (m *MockSagaStep) GetName() string {
	return m.name
}

func (m *MockSagaStep) GetDescription() string {
	return m.description
}

func (m *MockSagaStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	return "mock-result", nil
}

func (m *MockSagaStep) Compensate(ctx context.Context, data interface{}) error {
	return nil
}

func (m *MockSagaStep) GetTimeout() time.Duration {
	return m.timeout
}

func (m *MockSagaStep) GetRetryPolicy() RetryPolicy {
	return m.retryPolicy
}

func (m *MockSagaStep) IsRetryable(err error) bool {
	return true
}

func (m *MockSagaStep) GetMetadata() map[string]interface{} {
	return m.metadata
}

// Test functions

func TestSagaCoordinator_Interface(t *testing.T) {
	coordinator := NewMockSagaCoordinator()

	// Test that the mock implements SagaCoordinator interface
	var _ SagaCoordinator = coordinator

	// Test StartSaga
	definition := &MockSagaDefinition{
		id:      "test-definition",
		name:    "Test Saga",
		timeout: 30 * time.Second,
	}

	instance, err := coordinator.StartSaga(context.Background(), definition, nil)
	if err != nil {
		t.Fatalf("StartSaga failed: %v", err)
	}

	if instance.GetID() == "" {
		t.Error("Expected non-empty saga ID")
	}

	// Test GetSagaInstance
	retrievedInstance, err := coordinator.GetSagaInstance(instance.GetID())
	if err != nil {
		t.Fatalf("GetSagaInstance failed: %v", err)
	}

	if retrievedInstance.GetID() != instance.GetID() {
		t.Error("Retrieved instance ID doesn't match")
	}

	// Test GetActiveSagas
	activeSagas, err := coordinator.GetActiveSagas(nil)
	if err != nil {
		t.Fatalf("GetActiveSagas failed: %v", err)
	}

	if len(activeSagas) != 1 {
		t.Errorf("Expected 1 active saga, got %d", len(activeSagas))
	}

	// Test GetMetrics
	metrics := coordinator.GetMetrics()
	if metrics.TotalSagas != 1 {
		t.Errorf("Expected 1 total saga, got %d", metrics.TotalSagas)
	}

	// Test HealthCheck
	err = coordinator.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	// Test CancelSaga
	err = coordinator.CancelSaga(context.Background(), instance.GetID(), "test cancel")
	if err != nil {
		t.Fatalf("CancelSaga failed: %v", err)
	}

	// Test Close
	err = coordinator.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestSagaInstance_Interface(t *testing.T) {
	instance := &MockSagaInstance{
		id:             "test-instance",
		definitionID:   "test-definition",
		state:          StateRunning,
		currentStep:    1,
		startTime:      time.Now().Add(-1 * time.Hour),
		endTime:        time.Time{},
		result:         nil,
		err:            nil,
		totalSteps:     3,
		completedSteps: 1,
		createdAt:      time.Now().Add(-2 * time.Hour),
		updatedAt:      time.Now().Add(-30 * time.Minute),
		timeout:        30 * time.Minute,
		metadata:       map[string]interface{}{"key": "value"},
		traceID:        "test-trace",
	}

	// Test that the mock implements SagaInstance interface
	var _ SagaInstance = instance

	// Test all getter methods
	if instance.GetID() != "test-instance" {
		t.Errorf("Expected 'test-instance', got '%s'", instance.GetID())
	}

	if instance.GetDefinitionID() != "test-definition" {
		t.Errorf("Expected 'test-definition', got '%s'", instance.GetDefinitionID())
	}

	if instance.GetState() != StateRunning {
		t.Errorf("Expected StateRunning, got %v", instance.GetState())
	}

	if instance.GetCurrentStep() != 1 {
		t.Errorf("Expected 1, got %d", instance.GetCurrentStep())
	}

	if instance.GetTotalSteps() != 3 {
		t.Errorf("Expected 3, got %d", instance.GetTotalSteps())
	}

	if instance.GetCompletedSteps() != 1 {
		t.Errorf("Expected 1, got %d", instance.GetCompletedSteps())
	}

	if instance.GetTimeout() != 30*time.Minute {
		t.Errorf("Expected 30m, got %v", instance.GetTimeout())
	}

	if instance.GetTraceID() != "test-trace" {
		t.Errorf("Expected 'test-trace', got '%s'", instance.GetTraceID())
	}

	if instance.IsTerminal() {
		t.Error("Expected instance to be active, not terminal")
	}

	if !instance.IsActive() {
		t.Error("Expected instance to be active")
	}

	// Test terminal state
	instance.state = StateCompleted
	if !instance.IsTerminal() {
		t.Error("Expected instance to be terminal")
	}
	if instance.IsActive() {
		t.Error("Expected instance to not be active")
	}
}

func TestSagaDefinition_Interface(t *testing.T) {
	step := &MockSagaStep{
		id:   "test-step",
		name: "Test Step",
	}

	definition := &MockSagaDefinition{
		id:          "test-definition",
		name:        "Test Definition",
		description: "Test Description",
		steps:       []SagaStep{step},
		timeout:     30 * time.Second,
		metadata:    map[string]interface{}{"key": "value"},
	}

	// Test that the mock implements SagaDefinition interface
	var _ SagaDefinition = definition

	// Test all getter methods
	if definition.GetID() != "test-definition" {
		t.Errorf("Expected 'test-definition', got '%s'", definition.GetID())
	}

	if definition.GetName() != "Test Definition" {
		t.Errorf("Expected 'Test Definition', got '%s'", definition.GetName())
	}

	if definition.GetDescription() != "Test Description" {
		t.Errorf("Expected 'Test Description', got '%s'", definition.GetDescription())
	}

	if len(definition.GetSteps()) != 1 {
		t.Errorf("Expected 1 step, got %d", len(definition.GetSteps()))
	}

	if definition.GetTimeout() != 30*time.Second {
		t.Errorf("Expected 30s, got %v", definition.GetTimeout())
	}

	// Test validation
	err := definition.Validate()
	if err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	// Test invalid definition
	invalidDef := &MockSagaDefinition{
		id:    "", // Empty ID should fail validation
		steps: []SagaStep{},
	}

	err = invalidDef.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid definition")
	}
}

func TestSagaStep_Interface(t *testing.T) {
	step := &MockSagaStep{
		id:          "test-step",
		name:        "Test Step",
		description: "Test Description",
		timeout:     10 * time.Second,
		metadata:    map[string]interface{}{"key": "value"},
	}

	// Test that the mock implements SagaStep interface
	var _ SagaStep = step

	// Test all getter methods
	if step.GetID() != "test-step" {
		t.Errorf("Expected 'test-step', got '%s'", step.GetID())
	}

	if step.GetName() != "Test Step" {
		t.Errorf("Expected 'Test Step', got '%s'", step.GetName())
	}

	if step.GetDescription() != "Test Description" {
		t.Errorf("Expected 'Test Description', got '%s'", step.GetDescription())
	}

	if step.GetTimeout() != 10*time.Second {
		t.Errorf("Expected 10s, got %v", step.GetTimeout())
	}

	// Test Execute and Compensate
	result, err := step.Execute(context.Background(), "test-data")
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	err = step.Compensate(context.Background(), "test-data")
	if err != nil {
		t.Errorf("Compensate failed: %v", err)
	}

	// Test IsRetryable
	if !step.IsRetryable(nil) {
		t.Error("Expected step to be retryable")
	}
}

// MockStateStorage implementation

func NewMockStateStorage() *MockStateStorage {
	return &MockStateStorage{
		sagas:      make(map[string]SagaInstance),
		stepStates: make(map[string][]*StepState),
	}
}

func (m *MockStateStorage) SaveSaga(ctx context.Context, saga SagaInstance) error {
	if saga == nil {
		return NewValidationError("Saga instance cannot be nil")
	}
	m.sagas[saga.GetID()] = saga
	return nil
}

func (m *MockStateStorage) GetSaga(ctx context.Context, sagaID string) (SagaInstance, error) {
	if sagaID == "" {
		return nil, NewValidationError("Saga ID cannot be empty")
	}

	saga, exists := m.sagas[sagaID]
	if !exists {
		return nil, NewSagaNotFoundError(sagaID)
	}

	return saga, nil
}

func (m *MockStateStorage) UpdateSagaState(ctx context.Context, sagaID string, state SagaState, metadata map[string]interface{}) error {
	if sagaID == "" {
		return NewValidationError("Saga ID cannot be empty")
	}

	_, exists := m.sagas[sagaID]
	if !exists {
		return NewSagaNotFoundError(sagaID)
	}

	// Update the saga state - note: this is a simplified implementation
	// In a real implementation, we would need to update the underlying saga instance
	// For now, we'll just store the state change in metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["state"] = state.String()
	metadata["state_updated_at"] = time.Now()

	return nil
}

func (m *MockStateStorage) DeleteSaga(ctx context.Context, sagaID string) error {
	if sagaID == "" {
		return NewValidationError("Saga ID cannot be empty")
	}

	if _, exists := m.sagas[sagaID]; !exists {
		return NewSagaNotFoundError(sagaID)
	}

	delete(m.sagas, sagaID)
	delete(m.stepStates, sagaID)

	return nil
}

func (m *MockStateStorage) GetActiveSagas(ctx context.Context, filter *SagaFilter) ([]SagaInstance, error) {
	var activeSagas []SagaInstance

	for _, saga := range m.sagas {
		if saga.IsActive() {
			// Apply filter if provided
			if filter != nil {
				// Simple filtering logic - in real implementation would be more comprehensive
				if len(filter.States) > 0 {
					found := false
					for _, state := range filter.States {
						if saga.GetState() == state {
							found = true
							break
						}
					}
					if !found {
						continue
					}
				}
			}
			activeSagas = append(activeSagas, saga)
		}
	}

	return activeSagas, nil
}

func (m *MockStateStorage) GetTimeoutSagas(ctx context.Context, before time.Time) ([]SagaInstance, error) {
	var timeoutSagas []SagaInstance

	for _, saga := range m.sagas {
		if saga.IsActive() && saga.GetStartTime().Add(saga.GetTimeout()).Before(before) {
			timeoutSagas = append(timeoutSagas, saga)
		}
	}

	return timeoutSagas, nil
}

func (m *MockStateStorage) SaveStepState(ctx context.Context, sagaID string, step *StepState) error {
	if sagaID == "" {
		return NewValidationError("Saga ID cannot be empty")
	}
	if step == nil {
		return NewValidationError("Step state cannot be nil")
	}

	if _, exists := m.sagas[sagaID]; !exists {
		return NewSagaNotFoundError(sagaID)
	}

	if m.stepStates[sagaID] == nil {
		m.stepStates[sagaID] = []*StepState{}
	}

	// Find existing step state and update, or append new one
	for i, existingStep := range m.stepStates[sagaID] {
		if existingStep.ID == step.ID {
			m.stepStates[sagaID][i] = step
			return nil
		}
	}

	m.stepStates[sagaID] = append(m.stepStates[sagaID], step)
	return nil
}

func (m *MockStateStorage) GetStepStates(ctx context.Context, sagaID string) ([]*StepState, error) {
	if sagaID == "" {
		return nil, NewValidationError("Saga ID cannot be empty")
	}

	if _, exists := m.sagas[sagaID]; !exists {
		return nil, NewSagaNotFoundError(sagaID)
	}

	states := m.stepStates[sagaID]
	if states == nil {
		return []*StepState{}, nil
	}

	return states, nil
}

func (m *MockStateStorage) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	var sagasToDelete []string

	for sagaID, saga := range m.sagas {
		if saga.GetEndTime().Before(olderThan) && saga.IsTerminal() {
			sagasToDelete = append(sagasToDelete, sagaID)
		}
	}

	for _, sagaID := range sagasToDelete {
		delete(m.sagas, sagaID)
		delete(m.stepStates, sagaID)
	}

	return nil
}

// Test functions for StateStorage interface

func TestStateStorage_Interface(t *testing.T) {
	storage := NewMockStateStorage()

	// Test that the mock implements StateStorage interface
	var _ StateStorage = storage

	// Create test saga instance
	instance := &MockSagaInstance{
		id:           "test-saga-id",
		definitionID: "test-definition",
		state:        StateRunning,
		currentStep:  0,
		startTime:    time.Now(),
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
		totalSteps:   3,
		timeout:      30 * time.Minute,
		metadata:     map[string]interface{}{"key": "value"},
		traceID:      "test-trace",
	}

	ctx := context.Background()

	// Test SaveSaga
	err := storage.SaveSaga(ctx, instance)
	if err != nil {
		t.Fatalf("SaveSaga failed: %v", err)
	}

	// Test GetSaga
	retrievedInstance, err := storage.GetSaga(ctx, "test-saga-id")
	if err != nil {
		t.Fatalf("GetSaga failed: %v", err)
	}

	if retrievedInstance.GetID() != "test-saga-id" {
		t.Errorf("Expected 'test-saga-id', got '%s'", retrievedInstance.GetID())
	}

	// Test GetSaga with non-existent ID
	_, err = storage.GetSaga(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent saga")
	}
	if !IsSagaNotFound(err) {
		t.Errorf("Expected SagaNotFoundError, got %v", err)
	}

	// Test UpdateSagaState
	err = storage.UpdateSagaState(ctx, "test-saga-id", StateCompleted, map[string]interface{}{"updated": true})
	if err != nil {
		t.Fatalf("UpdateSagaState failed: %v", err)
	}

	// Test GetActiveSagas
	activeSagas, err := storage.GetActiveSagas(ctx, nil)
	if err != nil {
		t.Fatalf("GetActiveSagas failed: %v", err)
	}

	if len(activeSagas) != 1 {
		t.Errorf("Expected 1 active saga, got %d", len(activeSagas))
	}

	// Test GetActiveSagas with filter
	filter := &SagaFilter{
		States: []SagaState{StateRunning},
	}
	filteredSagas, err := storage.GetActiveSagas(ctx, filter)
	if err != nil {
		t.Fatalf("GetActiveSagas with filter failed: %v", err)
	}

	if len(filteredSagas) != 1 {
		t.Errorf("Expected 1 filtered saga, got %d", len(filteredSagas))
	}

	// Test SaveStepState
	stepState := &StepState{
		ID:        "test-step-id",
		SagaID:    "test-saga-id",
		StepIndex: 0,
		Name:      "Test Step",
		State:     StepStateCompleted,
		CreatedAt: time.Now(),
	}

	err = storage.SaveStepState(ctx, "test-saga-id", stepState)
	if err != nil {
		t.Fatalf("SaveStepState failed: %v", err)
	}

	// Test GetStepStates
	stepStates, err := storage.GetStepStates(ctx, "test-saga-id")
	if err != nil {
		t.Fatalf("GetStepStates failed: %v", err)
	}

	if len(stepStates) != 1 {
		t.Errorf("Expected 1 step state, got %d", len(stepStates))
	}

	if stepStates[0].ID != "test-step-id" {
		t.Errorf("Expected 'test-step-id', got '%s'", stepStates[0].ID)
	}

	// Test GetTimeoutSagas (should return none since saga hasn't timed out)
	timeoutSagas, err := storage.GetTimeoutSagas(ctx, time.Now())
	if err != nil {
		t.Fatalf("GetTimeoutSagas failed: %v", err)
	}

	if len(timeoutSagas) != 0 {
		t.Errorf("Expected 0 timeout sagas, got %d", len(timeoutSagas))
	}

	// Test CleanupExpiredSagas
	err = storage.CleanupExpiredSagas(ctx, time.Now())
	if err != nil {
		t.Fatalf("CleanupExpiredSagas failed: %v", err)
	}

	// Test DeleteSaga
	err = storage.DeleteSaga(ctx, "test-saga-id")
	if err != nil {
		t.Fatalf("DeleteSaga failed: %v", err)
	}

	// Verify saga is deleted
	_, err = storage.GetSaga(ctx, "test-saga-id")
	if err == nil {
		t.Error("Expected error for deleted saga")
	}
}

func TestStateStorage_ErrorCases(t *testing.T) {
	storage := NewMockStateStorage()
	ctx := context.Background()

	// Test SaveSaga with nil saga
	err := storage.SaveSaga(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil saga")
	}

	// Test GetSaga with empty ID
	_, err = storage.GetSaga(ctx, "")
	if err == nil {
		t.Error("Expected error for empty saga ID")
	}

	// Test UpdateSagaState with empty ID
	err = storage.UpdateSagaState(ctx, "", StateCompleted, nil)
	if err == nil {
		t.Error("Expected error for empty saga ID")
	}

	// Test DeleteSaga with empty ID
	err = storage.DeleteSaga(ctx, "")
	if err == nil {
		t.Error("Expected error for empty saga ID")
	}

	// Test SaveStepState with empty saga ID
	stepState := &StepState{ID: "test-step", SagaID: "test-saga"}
	err = storage.SaveStepState(ctx, "", stepState)
	if err == nil {
		t.Error("Expected error for empty saga ID")
	}

	// Test SaveStepState with nil step
	err = storage.SaveStepState(ctx, "test-saga-id", nil)
	if err == nil {
		t.Error("Expected error for nil step")
	}

	// Test GetStepStates with empty ID
	_, err = storage.GetStepStates(ctx, "")
	if err == nil {
		t.Error("Expected error for empty saga ID")
	}

	// Test operations on non-existent saga
	_, err = storage.GetStepStates(ctx, "non-existent-saga")
	if err == nil {
		t.Error("Expected error for non-existent saga")
	}
}

func TestStateStorage_MultipleSagas(t *testing.T) {
	storage := NewMockStateStorage()
	ctx := context.Background()

	// Create multiple test sagas
	saga1 := &MockSagaInstance{
		id:           "saga-1",
		definitionID: "test-definition",
		state:        StateRunning,
		startTime:    time.Now().Add(-2 * time.Hour),
		createdAt:    time.Now().Add(-2 * time.Hour),
		updatedAt:    time.Now(),
		totalSteps:   3,
		timeout:      1 * time.Hour, // Will be timed out
	}

	saga2 := &MockSagaInstance{
		id:           "saga-2",
		definitionID: "test-definition",
		state:        StateCompleted,
		startTime:    time.Now().Add(-3 * time.Hour),
		endTime:      time.Now().Add(-2 * time.Hour),
		createdAt:    time.Now().Add(-3 * time.Hour),
		updatedAt:    time.Now(),
		totalSteps:   3,
		timeout:      1 * time.Hour,
	}

	saga3 := &MockSagaInstance{
		id:           "saga-3",
		definitionID: "test-definition",
		state:        StateRunning,
		startTime:    time.Now().Add(-30 * time.Minute),
		createdAt:    time.Now().Add(-30 * time.Minute),
		updatedAt:    time.Now(),
		totalSteps:   3,
		timeout:      2 * time.Hour, // Not timed out
	}

	// Save all sagas
	err := storage.SaveSaga(ctx, saga1)
	if err != nil {
		t.Fatalf("SaveSaga saga1 failed: %v", err)
	}

	err = storage.SaveSaga(ctx, saga2)
	if err != nil {
		t.Fatalf("SaveSaga saga2 failed: %v", err)
	}

	err = storage.SaveSaga(ctx, saga3)
	if err != nil {
		t.Fatalf("SaveSaga saga3 failed: %v", err)
	}

	// Test GetActiveSagas
	activeSagas, err := storage.GetActiveSagas(ctx, nil)
	if err != nil {
		t.Fatalf("GetActiveSagas failed: %v", err)
	}

	if len(activeSagas) != 2 {
		t.Errorf("Expected 2 active sagas, got %d", len(activeSagas))
	}

	// Test GetTimeoutSagas - saga1 should be timed out
	timeoutSagas, err := storage.GetTimeoutSagas(ctx, time.Now())
	if err != nil {
		t.Fatalf("GetTimeoutSagas failed: %v", err)
	}

	if len(timeoutSagas) != 1 {
		t.Errorf("Expected 1 timeout saga, got %d", len(timeoutSagas))
	}

	if timeoutSagas[0].GetID() != "saga-1" {
		t.Errorf("Expected 'saga-1', got '%s'", timeoutSagas[0].GetID())
	}

	// Test CleanupExpiredSagas - saga2 is completed and old enough
	err = storage.CleanupExpiredSagas(ctx, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("CleanupExpiredSagas failed: %v", err)
	}

	// Verify saga2 was cleaned up
	_, err = storage.GetSaga(ctx, "saga-2")
	if err == nil {
		t.Error("Expected saga2 to be cleaned up")
	}
}
