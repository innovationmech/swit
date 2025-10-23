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

package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockVisualizationCoordinator is a mock coordinator for visualization testing.
type mockVisualizationCoordinator struct {
	mock.Mock
}

func (m *mockVisualizationCoordinator) StartSaga(ctx context.Context, definition saga.SagaDefinition, initialData interface{}) (saga.SagaInstance, error) {
	args := m.Called(ctx, definition, initialData)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(saga.SagaInstance), args.Error(1)
}

func (m *mockVisualizationCoordinator) GetSagaInstance(sagaID string) (saga.SagaInstance, error) {
	args := m.Called(sagaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(saga.SagaInstance), args.Error(1)
}

func (m *mockVisualizationCoordinator) CancelSaga(ctx context.Context, sagaID string, reason string) error {
	args := m.Called(ctx, sagaID, reason)
	return args.Error(0)
}

func (m *mockVisualizationCoordinator) GetActiveSagas(filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	args := m.Called(filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]saga.SagaInstance), args.Error(1)
}

func (m *mockVisualizationCoordinator) GetMetrics() *saga.CoordinatorMetrics {
	args := m.Called()
	return args.Get(0).(*saga.CoordinatorMetrics)
}

func (m *mockVisualizationCoordinator) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockVisualizationCoordinator) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockVisualizationCoordinator) GetDefinition(definitionID string) (saga.SagaDefinition, error) {
	args := m.Called(definitionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(saga.SagaDefinition), args.Error(1)
}

func (m *mockVisualizationCoordinator) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	args := m.Called(ctx, sagaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*saga.StepState), args.Error(1)
}

// mockVisualizationSagaInstance is a mock Saga instance for visualization testing.
type mockVisualizationSagaInstance struct {
	id             string
	definitionID   string
	state          saga.SagaState
	currentStep    int
	startTime      time.Time
	endTime        time.Time
	createdAt      time.Time
	updatedAt      time.Time
	metadata       map[string]interface{}
	totalSteps     int
	completedSteps int
}

func (m *mockVisualizationSagaInstance) GetID() string {
	return m.id
}

func (m *mockVisualizationSagaInstance) GetDefinitionID() string {
	return m.definitionID
}

func (m *mockVisualizationSagaInstance) GetState() saga.SagaState {
	return m.state
}

func (m *mockVisualizationSagaInstance) GetCurrentStep() int {
	return m.currentStep
}

func (m *mockVisualizationSagaInstance) GetStartTime() time.Time {
	return m.startTime
}

func (m *mockVisualizationSagaInstance) GetEndTime() time.Time {
	return m.endTime
}

func (m *mockVisualizationSagaInstance) GetResult() interface{} {
	return nil
}

func (m *mockVisualizationSagaInstance) GetError() *saga.SagaError {
	return nil
}

func (m *mockVisualizationSagaInstance) GetTimeout() time.Duration {
	return 30 * time.Second
}

func (m *mockVisualizationSagaInstance) GetCreatedAt() time.Time {
	return m.createdAt
}

func (m *mockVisualizationSagaInstance) GetUpdatedAt() time.Time {
	return m.updatedAt
}

func (m *mockVisualizationSagaInstance) GetTraceID() string {
	return "test-trace-id"
}

func (m *mockVisualizationSagaInstance) GetMetadata() map[string]interface{} {
	return m.metadata
}

func (m *mockVisualizationSagaInstance) IsActive() bool {
	return m.state == saga.StateRunning
}

func (m *mockVisualizationSagaInstance) GetTotalSteps() int {
	return m.totalSteps
}

func (m *mockVisualizationSagaInstance) GetCompletedSteps() int {
	return m.completedSteps
}

func (m *mockVisualizationSagaInstance) IsTerminal() bool {
	return m.state == saga.StateCompleted || m.state == saga.StateFailed ||
		m.state == saga.StateCompensated || m.state == saga.StateCancelled
}

// mockVisualizationSagaDefinition is a mock Saga definition for testing.
type mockVisualizationSagaDefinition struct {
	id          string
	name        string
	description string
	steps       []saga.SagaStep
	timeout     time.Duration
	metadata    map[string]interface{}
}

func (m *mockVisualizationSagaDefinition) GetID() string {
	return m.id
}

func (m *mockVisualizationSagaDefinition) GetName() string {
	return m.name
}

func (m *mockVisualizationSagaDefinition) GetDescription() string {
	return m.description
}

func (m *mockVisualizationSagaDefinition) GetSteps() []saga.SagaStep {
	return m.steps
}

func (m *mockVisualizationSagaDefinition) GetTimeout() time.Duration {
	return m.timeout
}

func (m *mockVisualizationSagaDefinition) GetRetryPolicy() saga.RetryPolicy {
	return nil
}

func (m *mockVisualizationSagaDefinition) GetCompensationStrategy() saga.CompensationStrategy {
	return nil
}

func (m *mockVisualizationSagaDefinition) Validate() error {
	return nil
}

func (m *mockVisualizationSagaDefinition) GetMetadata() map[string]interface{} {
	return m.metadata
}

// mockVisualizationSagaStep is a mock Saga step for testing.
type mockVisualizationSagaStep struct {
	id          string
	name        string
	description string
	timeout     time.Duration
	metadata    map[string]interface{}
}

func (m *mockVisualizationSagaStep) GetID() string {
	return m.id
}

func (m *mockVisualizationSagaStep) GetName() string {
	return m.name
}

func (m *mockVisualizationSagaStep) GetDescription() string {
	return m.description
}

func (m *mockVisualizationSagaStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	return nil, nil
}

func (m *mockVisualizationSagaStep) Compensate(ctx context.Context, data interface{}) error {
	return nil
}

func (m *mockVisualizationSagaStep) GetTimeout() time.Duration {
	return m.timeout
}

func (m *mockVisualizationSagaStep) GetRetryPolicy() saga.RetryPolicy {
	return nil
}

func (m *mockVisualizationSagaStep) IsRetryable(err error) bool {
	return false
}

func (m *mockVisualizationSagaStep) GetMetadata() map[string]interface{} {
	return m.metadata
}

// TestNewSagaVisualizer tests the creation of a new visualizer.
func TestNewSagaVisualizer(t *testing.T) {
	coordinator := &mockVisualizationCoordinator{}
	visualizer := NewSagaVisualizer(coordinator)

	assert.NotNil(t, visualizer)
	assert.Equal(t, coordinator, visualizer.coordinator)
}

// TestGenerateVisualization_NotFound tests visualization generation when Saga is not found.
func TestGenerateVisualization_NotFound(t *testing.T) {
	coordinator := &mockVisualizationCoordinator{}
	notFoundErr := saga.NewSagaNotFoundError("non-existent")
	coordinator.On("GetSagaInstance", "non-existent").Return(nil, notFoundErr)

	visualizer := NewSagaVisualizer(coordinator)
	vizData, err := visualizer.GenerateVisualization(context.Background(), "non-existent")

	assert.Error(t, err)
	assert.Nil(t, vizData)
	coordinator.AssertExpectations(t)
}

// TestGenerateVisualization_OrchestrationPattern tests visualization for orchestration pattern.
func TestGenerateVisualization_OrchestrationPattern(t *testing.T) {
	// Create test data
	now := time.Now()
	instance := &mockVisualizationSagaInstance{
		id:           "test-saga-1",
		definitionID: "order-saga",
		state:        saga.StateRunning,
		currentStep:  1,
		startTime:    now.Add(-5 * time.Minute),
		createdAt:    now.Add(-5 * time.Minute),
		updatedAt:    now,
		metadata:     map[string]interface{}{"orderID": "12345"},
	}

	steps := []saga.SagaStep{
		&mockVisualizationSagaStep{
			id:          "step-1",
			name:        "Reserve Inventory",
			description: "Reserve items in inventory",
			timeout:     5 * time.Second,
		},
		&mockVisualizationSagaStep{
			id:          "step-2",
			name:        "Process Payment",
			description: "Process customer payment",
			timeout:     10 * time.Second,
		},
		&mockVisualizationSagaStep{
			id:          "step-3",
			name:        "Create Shipment",
			description: "Create shipment order",
			timeout:     5 * time.Second,
		},
	}

	definition := &mockVisualizationSagaDefinition{
		id:          "order-saga",
		name:        "Order Processing Saga",
		description: "Orchestrates order processing",
		steps:       steps,
		timeout:     30 * time.Second,
		metadata:    map[string]interface{}{"flowType": "orchestration"},
	}

	startTime := now.Add(-4 * time.Minute)
	endTime := now.Add(-3 * time.Minute)
	stepStates := []*saga.StepState{
		{
			ID:          "test-saga-1-step-0",
			SagaID:      "test-saga-1",
			StepIndex:   0,
			Name:        "Reserve Inventory",
			State:       saga.StepStateCompleted,
			Attempts:    1,
			MaxAttempts: 3,
			CreatedAt:   startTime,
			StartedAt:   &startTime,
			CompletedAt: &endTime,
		},
		{
			ID:          "test-saga-1-step-1",
			SagaID:      "test-saga-1",
			StepIndex:   1,
			Name:        "Process Payment",
			State:       saga.StepStateRunning,
			Attempts:    1,
			MaxAttempts: 3,
			CreatedAt:   endTime,
			StartedAt:   &endTime,
		},
		{
			ID:          "test-saga-1-step-2",
			SagaID:      "test-saga-1",
			StepIndex:   2,
			Name:        "Create Shipment",
			State:       saga.StepStatePending,
			Attempts:    0,
			MaxAttempts: 3,
			CreatedAt:   now,
		},
	}

	coordinator := &mockVisualizationCoordinator{}
	coordinator.On("GetSagaInstance", "test-saga-1").Return(instance, nil)
	coordinator.On("GetDefinition", "order-saga").Return(definition, nil)
	coordinator.On("GetStepStates", mock.Anything, "test-saga-1").Return(stepStates, nil)

	visualizer := NewSagaVisualizer(coordinator)
	vizData, err := visualizer.GenerateVisualization(context.Background(), "test-saga-1")

	assert.NoError(t, err)
	assert.NotNil(t, vizData)

	// Verify basic properties
	assert.Equal(t, "test-saga-1", vizData.SagaID)
	assert.Equal(t, "order-saga", vizData.DefinitionID)
	assert.Equal(t, FlowTypeOrchestration, vizData.FlowType)
	assert.Equal(t, "running", vizData.State)
	assert.Equal(t, 1, vizData.CurrentStep)
	assert.Equal(t, 3, vizData.TotalSteps)
	assert.Equal(t, 1, vizData.CompletedSteps)
	assert.False(t, vizData.CompensationActive)

	// Verify nodes
	assert.NotEmpty(t, vizData.Nodes)
	// Should have: start + 3 steps + end = 5 nodes
	assert.GreaterOrEqual(t, len(vizData.Nodes), 5)

	// Verify start node
	startNode := vizData.Nodes[0]
	assert.Equal(t, "start", startNode.ID)
	assert.Equal(t, NodeTypeStart, startNode.Type)
	assert.Equal(t, NodeStateActive, startNode.State)

	// Verify step nodes
	hasStep0 := false
	hasStep1 := false
	hasStep2 := false
	for _, node := range vizData.Nodes {
		if node.ID == "step-0" {
			hasStep0 = true
			assert.Equal(t, NodeTypeStep, node.Type)
			assert.Equal(t, "Reserve Inventory", node.Label)
			assert.Equal(t, NodeStateCompleted, node.State)
			assert.Equal(t, 1, node.Attempts)
		}
		if node.ID == "step-1" {
			hasStep1 = true
			assert.Equal(t, NodeTypeStep, node.Type)
			assert.Equal(t, "Process Payment", node.Label)
			assert.Equal(t, NodeStateRunning, node.State)
		}
		if node.ID == "step-2" {
			hasStep2 = true
			assert.Equal(t, NodeTypeStep, node.Type)
			assert.Equal(t, "Create Shipment", node.Label)
			assert.Equal(t, NodeStatePending, node.State)
		}
	}
	assert.True(t, hasStep0, "Should have step-0 node")
	assert.True(t, hasStep1, "Should have step-1 node")
	assert.True(t, hasStep2, "Should have step-2 node")

	// Verify edges
	assert.NotEmpty(t, vizData.Edges)
	// Should have at least: start->step0, step0->step1, step1->step2, step2->end = 4 edges
	assert.GreaterOrEqual(t, len(vizData.Edges), 4)

	// Verify edge connectivity
	hasStartToStep0 := false
	hasStep0ToStep1 := false
	for _, edge := range vizData.Edges {
		if edge.Source == "start" && edge.Target == "step-0" {
			hasStartToStep0 = true
			assert.Equal(t, EdgeTypeSequential, edge.Type)
			assert.True(t, edge.Active)
		}
		if edge.Source == "step-0" && edge.Target == "step-1" {
			hasStep0ToStep1 = true
			assert.Equal(t, EdgeTypeSequential, edge.Type)
			assert.True(t, edge.Active)
		}
	}
	assert.True(t, hasStartToStep0, "Should have start->step-0 edge")
	assert.True(t, hasStep0ToStep1, "Should have step-0->step-1 edge")

	coordinator.AssertExpectations(t)
}

// TestGenerateVisualization_CompensationFlow tests visualization with compensation.
func TestGenerateVisualization_CompensationFlow(t *testing.T) {
	now := time.Now()
	instance := &mockVisualizationSagaInstance{
		id:           "test-saga-2",
		definitionID: "order-saga",
		state:        saga.StateCompensating,
		currentStep:  2,
		startTime:    now.Add(-10 * time.Minute),
		createdAt:    now.Add(-10 * time.Minute),
		updatedAt:    now,
		metadata:     map[string]interface{}{},
	}

	steps := []saga.SagaStep{
		&mockVisualizationSagaStep{
			id:   "step-1",
			name: "Reserve Inventory",
		},
		&mockVisualizationSagaStep{
			id:   "step-2",
			name: "Process Payment",
		},
	}

	definition := &mockVisualizationSagaDefinition{
		id:    "order-saga",
		steps: steps,
		metadata: map[string]interface{}{
			"flowType": "orchestration",
		},
	}

	compStartTime := now.Add(-1 * time.Minute)
	stepStates := []*saga.StepState{
		{
			ID:        "test-saga-2-step-0",
			SagaID:    "test-saga-2",
			StepIndex: 0,
			Name:      "Reserve Inventory",
			State:     saga.StepStateCompensated,
			CompensationState: &saga.CompensationState{
				State:     saga.CompensationStateCompleted,
				Attempts:  1,
				StartedAt: &compStartTime,
			},
		},
		{
			ID:        "test-saga-2-step-1",
			SagaID:    "test-saga-2",
			StepIndex: 1,
			Name:      "Process Payment",
			State:     saga.StepStateCompleted,
			CompensationState: &saga.CompensationState{
				State:    saga.CompensationStateRunning,
				Attempts: 1,
			},
		},
	}

	coordinator := &mockVisualizationCoordinator{}
	coordinator.On("GetSagaInstance", "test-saga-2").Return(instance, nil)
	coordinator.On("GetDefinition", "order-saga").Return(definition, nil)
	coordinator.On("GetStepStates", mock.Anything, "test-saga-2").Return(stepStates, nil)

	visualizer := NewSagaVisualizer(coordinator)
	vizData, err := visualizer.GenerateVisualization(context.Background(), "test-saga-2")

	assert.NoError(t, err)
	assert.NotNil(t, vizData)
	assert.Equal(t, "compensating", vizData.State)
	assert.True(t, vizData.CompensationActive)

	// Should have compensation nodes
	hasCompensation0 := false
	hasCompensation1 := false
	for _, node := range vizData.Nodes {
		if node.ID == "compensation-0" {
			hasCompensation0 = true
			assert.Equal(t, NodeTypeCompensation, node.Type)
			assert.Equal(t, NodeStateCompensated, node.State)
		}
		if node.ID == "compensation-1" {
			hasCompensation1 = true
			assert.Equal(t, NodeTypeCompensation, node.Type)
			assert.Equal(t, NodeStateRunning, node.State)
		}
	}
	assert.True(t, hasCompensation0, "Should have compensation-0 node")
	assert.True(t, hasCompensation1, "Should have compensation-1 node")

	// Should have compensation edges
	hasCompensationEdge := false
	for _, edge := range vizData.Edges {
		if edge.Type == EdgeTypeCompensation {
			hasCompensationEdge = true
		}
	}
	assert.True(t, hasCompensationEdge, "Should have compensation edges")

	coordinator.AssertExpectations(t)
}

// TestGenerateVisualization_ChoreographyPattern tests choreography pattern detection.
func TestGenerateVisualization_ChoreographyPattern(t *testing.T) {
	now := time.Now()
	instance := &mockVisualizationSagaInstance{
		id:           "test-saga-3",
		definitionID: "event-driven-saga",
		state:        saga.StateRunning,
		currentStep:  0,
		startTime:    now,
		createdAt:    now,
		updatedAt:    now,
		metadata:     map[string]interface{}{},
	}

	steps := []saga.SagaStep{
		&mockVisualizationSagaStep{
			id:   "step-1",
			name: "Publish Order Event",
		},
	}

	definition := &mockVisualizationSagaDefinition{
		id:    "event-driven-saga",
		steps: steps,
		metadata: map[string]interface{}{
			"flowType": "choreography",
		},
	}

	coordinator := &mockVisualizationCoordinator{}
	coordinator.On("GetSagaInstance", "test-saga-3").Return(instance, nil)
	coordinator.On("GetDefinition", "event-driven-saga").Return(definition, nil)
	coordinator.On("GetStepStates", mock.Anything, "test-saga-3").Return([]*saga.StepState{}, nil)

	visualizer := NewSagaVisualizer(coordinator)
	vizData, err := visualizer.GenerateVisualization(context.Background(), "test-saga-3")

	assert.NoError(t, err)
	assert.NotNil(t, vizData)
	assert.Equal(t, FlowTypeChoreography, vizData.FlowType)

	coordinator.AssertExpectations(t)
}

// TestVisualizationAPI_GetVisualization tests the HTTP API endpoint.
func TestVisualizationAPI_GetVisualization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		sagaID         string
		setupMock      func(*mockVisualizationCoordinator)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "Success",
			sagaID: "test-saga-1",
			setupMock: func(coord *mockVisualizationCoordinator) {
				instance := &mockVisualizationSagaInstance{
					id:           "test-saga-1",
					definitionID: "order-saga",
					state:        saga.StateRunning,
					currentStep:  0,
					createdAt:    time.Now(),
					updatedAt:    time.Now(),
				}
				definition := &mockVisualizationSagaDefinition{
					id: "order-saga",
					steps: []saga.SagaStep{
						&mockVisualizationSagaStep{
							id:   "step-1",
							name: "Step 1",
						},
					},
					metadata: map[string]interface{}{},
				}
				coord.On("GetSagaInstance", "test-saga-1").Return(instance, nil)
				coord.On("GetDefinition", "order-saga").Return(definition, nil)
				coord.On("GetStepStates", mock.Anything, "test-saga-1").Return([]*saga.StepState{}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var vizData VisualizationData
				err := json.Unmarshal(rec.Body.Bytes(), &vizData)
				assert.NoError(t, err)
				assert.Equal(t, "test-saga-1", vizData.SagaID)
				assert.Equal(t, "order-saga", vizData.DefinitionID)
				assert.NotEmpty(t, vizData.Nodes)
				assert.NotEmpty(t, vizData.Edges)
			},
		},
		{
			name:   "Not Found",
			sagaID: "non-existent",
			setupMock: func(coord *mockVisualizationCoordinator) {
				notFoundErr := saga.NewSagaNotFoundError("non-existent")
				coord.On("GetSagaInstance", "non-existent").Return(nil, notFoundErr)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var errResp ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &errResp)
				assert.NoError(t, err)
				assert.Equal(t, "not_found", errResp.Error)
			},
		},
		{
			name:   "Internal Error",
			sagaID: "error-saga",
			setupMock: func(coord *mockVisualizationCoordinator) {
				coord.On("GetSagaInstance", "error-saga").Return(nil, fmt.Errorf("internal error"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var errResp ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &errResp)
				assert.NoError(t, err)
				assert.Equal(t, "visualization_failed", errResp.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coordinator := &mockVisualizationCoordinator{}
			tt.setupMock(coordinator)

			api := NewSagaVisualizationAPI(coordinator)

			router := gin.New()
			router.GET("/api/sagas/:id/visualization", api.GetVisualization)

			req := httptest.NewRequest(http.MethodGet, "/api/sagas/"+tt.sagaID+"/visualization", nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.checkResponse(t, rec)

			coordinator.AssertExpectations(t)
		})
	}
}

// TestVisualizationAPI_GetVisualization_MissingSagaID tests handling of missing Saga ID.
func TestVisualizationAPI_GetVisualization_MissingSagaID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	coordinator := &mockVisualizationCoordinator{}
	api := NewSagaVisualizationAPI(coordinator)

	router := gin.New()
	// Test with empty ID parameter
	router.GET("/api/sagas/:id/visualization", api.GetVisualization)

	// Use empty string as sagaID in URL path - this actually gets caught by the API handler
	// because gin will pass empty string as the :id param
	req := httptest.NewRequest(http.MethodGet, "/api/sagas/"+""+"/visualization", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// When the sagaID param is empty, API returns 400
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestCalculateProgress tests the progress calculation logic.
func TestCalculateProgress(t *testing.T) {
	tests := []struct {
		name            string
		currentStep     int
		totalSteps      int
		state           saga.SagaState
		expectedMin     float64
		expectedMax     float64
		setupDefinition bool
	}{
		{
			name:            "Half Complete",
			currentStep:     2,
			totalSteps:      4,
			state:           saga.StateRunning,
			expectedMin:     40.0,
			expectedMax:     60.0,
			setupDefinition: true,
		},
		{
			name:            "Fully Complete",
			currentStep:     4,
			totalSteps:      4,
			state:           saga.StateCompleted,
			expectedMin:     90.0,
			expectedMax:     100.0,
			setupDefinition: true,
		},
		{
			name:            "Just Started",
			currentStep:     0,
			totalSteps:      5,
			state:           saga.StateRunning,
			expectedMin:     0.0,
			expectedMax:     10.0,
			setupDefinition: true,
		},
		{
			name:            "No Definition",
			currentStep:     2,
			totalSteps:      0,
			state:           saga.StateRunning,
			expectedMin:     0.0,
			expectedMax:     10.0,
			setupDefinition: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &mockVisualizationSagaInstance{
				id:           "test-saga",
				definitionID: "test-def",
				state:        tt.state,
				currentStep:  tt.currentStep,
				createdAt:    time.Now(),
				updatedAt:    time.Now(),
			}

			coordinator := &mockVisualizationCoordinator{}

			var definition saga.SagaDefinition
			if tt.setupDefinition {
				steps := make([]saga.SagaStep, tt.totalSteps)
				for i := 0; i < tt.totalSteps; i++ {
					steps[i] = &mockVisualizationSagaStep{
						id:   fmt.Sprintf("step-%d", i),
						name: fmt.Sprintf("Step %d", i),
					}
				}
				definition = &mockVisualizationSagaDefinition{
					id:    "test-def",
					steps: steps,
				}
				coordinator.On("GetDefinition", "test-def").Return(definition, nil)
			} else {
				coordinator.On("GetDefinition", "test-def").Return(nil, fmt.Errorf("not found"))
			}

			visualizer := NewSagaVisualizer(coordinator)
			progress := visualizer.calculateProgress(instance)

			assert.GreaterOrEqual(t, progress, tt.expectedMin)
			assert.LessOrEqual(t, progress, tt.expectedMax)

			coordinator.AssertExpectations(t)
		})
	}
}
