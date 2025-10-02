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
