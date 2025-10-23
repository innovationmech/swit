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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSagaCoordinator is a mock implementation of saga.SagaCoordinator for testing.
type MockSagaCoordinator struct {
	mock.Mock
}

func (m *MockSagaCoordinator) StartSaga(ctx context.Context, definition saga.SagaDefinition, initialData interface{}) (saga.SagaInstance, error) {
	args := m.Called(ctx, definition, initialData)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(saga.SagaInstance), args.Error(1)
}

func (m *MockSagaCoordinator) GetSagaInstance(sagaID string) (saga.SagaInstance, error) {
	args := m.Called(sagaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(saga.SagaInstance), args.Error(1)
}

func (m *MockSagaCoordinator) CancelSaga(ctx context.Context, sagaID string, reason string) error {
	args := m.Called(ctx, sagaID, reason)
	return args.Error(0)
}

func (m *MockSagaCoordinator) GetActiveSagas(filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	args := m.Called(filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]saga.SagaInstance), args.Error(1)
}

func (m *MockSagaCoordinator) GetMetrics() *saga.CoordinatorMetrics {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*saga.CoordinatorMetrics)
}

func (m *MockSagaCoordinator) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSagaCoordinator) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockSagaInstance is a mock implementation of saga.SagaInstance for testing.
type MockSagaInstance struct {
	mock.Mock
}

func (m *MockSagaInstance) GetID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSagaInstance) GetDefinitionID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSagaInstance) GetState() saga.SagaState {
	args := m.Called()
	return args.Get(0).(saga.SagaState)
}

func (m *MockSagaInstance) GetCurrentStep() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockSagaInstance) GetTotalSteps() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockSagaInstance) GetCompletedSteps() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockSagaInstance) GetTraceID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSagaInstance) GetCreatedAt() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func (m *MockSagaInstance) GetUpdatedAt() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func (m *MockSagaInstance) GetStartTime() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func (m *MockSagaInstance) GetEndTime() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func (m *MockSagaInstance) GetTimeout() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func (m *MockSagaInstance) IsActive() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSagaInstance) IsTerminal() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSagaInstance) GetResult() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *MockSagaInstance) GetError() *saga.SagaError {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*saga.SagaError)
}

func (m *MockSagaInstance) GetMetadata() map[string]interface{} {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[string]interface{})
}

// setupControlTestRouter creates a test router with the control API.
func setupControlTestRouter(controlAPI *SagaControlAPI) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.POST("/api/sagas/:id/cancel", controlAPI.CancelSaga)
	router.POST("/api/sagas/:id/retry", controlAPI.RetrySaga)

	return router
}

func TestCancelSaga_Success(t *testing.T) {
	// Setup mock coordinator
	mockCoordinator := new(MockSagaCoordinator)
	mockInstance := new(MockSagaInstance)

	sagaID := "test-saga-123"

	// Mock GetSagaInstance to return a running saga
	mockInstance.On("GetState").Return(saga.StateRunning)
	mockInstance.On("IsTerminal").Return(false)
	mockCoordinator.On("GetSagaInstance", sagaID).Return(mockInstance, nil).Times(2)

	// Mock CancelSaga to succeed
	mockCoordinator.On("CancelSaga", mock.Anything, sagaID, "User requested cancellation").Return(nil)

	// Create control API
	controlAPI := NewSagaControlAPI(mockCoordinator, nil)
	router := setupControlTestRouter(controlAPI)

	// Create request
	reqBody := CancelSagaRequest{
		Reason: "User requested cancellation",
		Force:  false,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/sagas/"+sagaID+"/cancel", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response OperationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "cancelled successfully")
	assert.NotEmpty(t, response.OperationID)

	// Verify mocks
	mockCoordinator.AssertExpectations(t)
	mockInstance.AssertExpectations(t)
}

func TestCancelSaga_SagaNotFound(t *testing.T) {
	// Setup mock coordinator
	mockCoordinator := new(MockSagaCoordinator)

	sagaID := "non-existent-saga"

	// Mock GetSagaInstance to return not found error
	mockCoordinator.On("GetSagaInstance", sagaID).Return(nil, saga.NewSagaNotFoundError(sagaID))

	// Create control API
	controlAPI := NewSagaControlAPI(mockCoordinator, nil)
	router := setupControlTestRouter(controlAPI)

	// Create request
	reqBody := CancelSagaRequest{
		Reason: "User requested cancellation",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/sagas/"+sagaID+"/cancel", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "saga_not_found", response.Error)

	// Verify mocks
	mockCoordinator.AssertExpectations(t)
}

func TestCancelSaga_AlreadyTerminal(t *testing.T) {
	// Setup mock coordinator
	mockCoordinator := new(MockSagaCoordinator)
	mockInstance := new(MockSagaInstance)

	sagaID := "completed-saga"

	// Mock GetSagaInstance to return a completed saga
	mockInstance.On("IsTerminal").Return(true)
	mockInstance.On("GetState").Return(saga.StateCompleted)
	mockCoordinator.On("GetSagaInstance", sagaID).Return(mockInstance, nil)

	// Create control API
	controlAPI := NewSagaControlAPI(mockCoordinator, nil)
	router := setupControlTestRouter(controlAPI)

	// Create request
	reqBody := CancelSagaRequest{
		Reason: "User requested cancellation",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/sagas/"+sagaID+"/cancel", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "saga_already_terminal", response.Error)

	// Verify mocks
	mockCoordinator.AssertExpectations(t)
	mockInstance.AssertExpectations(t)
}

func TestCancelSaga_InvalidRequest(t *testing.T) {
	// Setup mock coordinator
	mockCoordinator := new(MockSagaCoordinator)

	// Create control API
	controlAPI := NewSagaControlAPI(mockCoordinator, nil)
	router := setupControlTestRouter(controlAPI)

	// Create invalid request (missing reason)
	reqBody := map[string]interface{}{
		"force": false,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/sagas/test-saga/cancel", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_request", response.Error)
}

func TestCancelSaga_ConcurrentOperation(t *testing.T) {
	// Setup mock coordinator
	mockCoordinator := new(MockSagaCoordinator)
	mockInstance := new(MockSagaInstance)

	sagaID := "test-saga-concurrent"

	// Mock GetSagaInstance to return a running saga
	mockInstance.On("GetState").Return(saga.StateRunning)
	mockInstance.On("IsTerminal").Return(false)
	mockCoordinator.On("GetSagaInstance", sagaID).Return(mockInstance, nil).Maybe()

	// Mock CancelSaga to succeed (but with delay to simulate concurrent access)
	mockCoordinator.On("CancelSaga", mock.Anything, sagaID, mock.Anything).Return(nil).Maybe()

	// Create control API
	controlAPI := NewSagaControlAPI(mockCoordinator, nil)
	router := setupControlTestRouter(controlAPI)

	// Manually acquire lock to simulate concurrent operation
	_ = controlAPI.acquireOperationLock(sagaID)

	// Second request should fail with concurrent operation error
	reqBody2 := CancelSagaRequest{Reason: "Second request"}
	bodyBytes2, _ := json.Marshal(reqBody2)
	req2 := httptest.NewRequest(http.MethodPost, "/api/sagas/"+sagaID+"/cancel", bytes.NewBuffer(bodyBytes2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	// Execute second request (should fail)
	router.ServeHTTP(w2, req2)

	// Assert second request got conflict error
	assert.Equal(t, http.StatusConflict, w2.Code)

	var response ErrorResponse
	err := json.Unmarshal(w2.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "concurrent_operation", response.Error)

	// Clean up lock
	controlAPI.releaseOperationLock(sagaID)
}

func TestRetrySaga_Success(t *testing.T) {
	// Setup mock coordinator
	mockCoordinator := new(MockSagaCoordinator)
	mockInstance := new(MockSagaInstance)

	sagaID := "failed-saga-123"

	// Mock GetSagaInstance to return a failed saga
	mockInstance.On("GetState").Return(saga.StateFailed).Maybe()
	mockInstance.On("IsTerminal").Return(false).Maybe()
	mockInstance.On("GetTotalSteps").Return(5).Maybe()
	mockCoordinator.On("GetSagaInstance", sagaID).Return(mockInstance, nil).Maybe()

	// Create control API (without recovery manager, will return error)
	controlAPI := NewSagaControlAPI(mockCoordinator, nil)
	router := setupControlTestRouter(controlAPI)

	// Create request
	reqBody := RetrySagaRequest{
		FromStep: -1,
		Reason:   "Transient error resolved",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/sagas/"+sagaID+"/retry", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response (should fail because no recovery manager)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "retry_failed", response.Error)

	// Verify mocks
	mockCoordinator.AssertExpectations(t)
	mockInstance.AssertExpectations(t)
}

func TestRetrySaga_InvalidState(t *testing.T) {
	// Setup mock coordinator
	mockCoordinator := new(MockSagaCoordinator)
	mockInstance := new(MockSagaInstance)

	sagaID := "completed-saga"

	// Mock GetSagaInstance to return a completed saga
	mockInstance.On("GetState").Return(saga.StateCompleted).Maybe()
	mockInstance.On("IsTerminal").Return(false).Maybe() // For validation
	mockCoordinator.On("GetSagaInstance", sagaID).Return(mockInstance, nil).Maybe()

	// Create control API
	controlAPI := NewSagaControlAPI(mockCoordinator, nil)
	router := setupControlTestRouter(controlAPI)

	// Create request
	reqBody := RetrySagaRequest{
		FromStep: -1,
		Reason:   "Retry completed saga",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/sagas/"+sagaID+"/retry", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_state", response.Error)

	// Verify mocks
	mockCoordinator.AssertExpectations(t)
	mockInstance.AssertExpectations(t)
}

func TestRetrySaga_InvalidStepIndex(t *testing.T) {
	// Setup mock coordinator
	mockCoordinator := new(MockSagaCoordinator)
	mockInstance := new(MockSagaInstance)

	sagaID := "failed-saga"

	// Mock GetSagaInstance to return a failed saga with 5 steps
	mockInstance.On("GetState").Return(saga.StateFailed).Maybe()
	mockInstance.On("IsTerminal").Return(false).Maybe()
	mockInstance.On("GetTotalSteps").Return(5).Maybe()
	mockCoordinator.On("GetSagaInstance", sagaID).Return(mockInstance, nil).Maybe()

	// Create control API
	controlAPI := NewSagaControlAPI(mockCoordinator, nil)
	router := setupControlTestRouter(controlAPI)

	// Create request with invalid step index
	reqBody := RetrySagaRequest{
		FromStep: 10, // Out of range (total steps = 5)
		Reason:   "Retry from step 10",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/sagas/"+sagaID+"/retry", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "validation_failed", response.Error)
	assert.Contains(t, response.Message, "out of range")

	// Verify mocks
	mockCoordinator.AssertExpectations(t)
	mockInstance.AssertExpectations(t)
}

func TestOperationValidator_ValidateCancelOperation(t *testing.T) {
	tests := []struct {
		name          string
		sagaState     saga.SagaState
		isTerminal    bool
		expectError   bool
		expectedError error
	}{
		{
			name:        "Running saga can be cancelled",
			sagaState:   saga.StateRunning,
			isTerminal:  false,
			expectError: false,
		},
		{
			name:        "Pending saga can be cancelled",
			sagaState:   saga.StatePending,
			isTerminal:  false,
			expectError: false,
		},
		{
			name:        "StepCompleted saga can be cancelled",
			sagaState:   saga.StateStepCompleted,
			isTerminal:  false,
			expectError: false,
		},
		{
			name:          "Completed saga cannot be cancelled",
			sagaState:     saga.StateCompleted,
			isTerminal:    true,
			expectError:   true,
			expectedError: ErrSagaAlreadyTerminal,
		},
		{
			name:          "Failed saga cannot be cancelled",
			sagaState:     saga.StateFailed,
			isTerminal:    true,
			expectError:   true,
			expectedError: ErrSagaAlreadyTerminal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCoordinator := new(MockSagaCoordinator)
			mockInstance := new(MockSagaInstance)

			sagaID := "test-saga"

			mockInstance.On("GetState").Return(tt.sagaState)
			mockInstance.On("IsTerminal").Return(tt.isTerminal)
			mockCoordinator.On("GetSagaInstance", sagaID).Return(mockInstance, nil)

			validator := NewOperationValidator(mockCoordinator)
			_, err := validator.ValidateCancelOperation(context.Background(), sagaID, "test reason")

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}

			mockCoordinator.AssertExpectations(t)
			mockInstance.AssertExpectations(t)
		})
	}
}

func TestOperationValidator_ValidateRetryOperation(t *testing.T) {
	tests := []struct {
		name          string
		sagaState     saga.SagaState
		fromStep      int
		totalSteps    int
		expectError   bool
		expectedError error
	}{
		{
			name:        "Failed saga can be retried",
			sagaState:   saga.StateFailed,
			fromStep:    -1,
			totalSteps:  5,
			expectError: false,
		},
		{
			name:        "Compensated saga can be retried",
			sagaState:   saga.StateCompensated,
			fromStep:    -1,
			totalSteps:  5,
			expectError: false,
		},
		{
			name:        "Running saga can be retried from specific step",
			sagaState:   saga.StateRunning,
			fromStep:    2,
			totalSteps:  5,
			expectError: false,
		},
		{
			name:          "Completed saga cannot be retried",
			sagaState:     saga.StateCompleted,
			fromStep:      -1,
			totalSteps:    5,
			expectError:   true,
			expectedError: ErrInvalidSagaState,
		},
		{
			name:        "Invalid step index",
			sagaState:   saga.StateFailed,
			fromStep:    10,
			totalSteps:  5,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCoordinator := new(MockSagaCoordinator)
			mockInstance := new(MockSagaInstance)

			sagaID := "test-saga"

			mockInstance.On("GetState").Return(tt.sagaState).Maybe()
			mockInstance.On("IsTerminal").Return(false).Maybe()
			if tt.fromStep >= 0 {
				mockInstance.On("GetTotalSteps").Return(tt.totalSteps).Maybe()
			}
			mockCoordinator.On("GetSagaInstance", sagaID).Return(mockInstance, nil).Maybe()

			validator := NewOperationValidator(mockCoordinator)
			_, err := validator.ValidateRetryOperation(context.Background(), sagaID, tt.fromStep)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}

			mockCoordinator.AssertExpectations(t)
			mockInstance.AssertExpectations(t)
		})
	}
}

func TestGenerateOperationID(t *testing.T) {
	operation := "cancel"
	sagaID := "test-saga-123"

	id1 := generateOperationID(operation, sagaID)
	time.Sleep(1100 * time.Millisecond) // Just over 1 second to ensure different timestamps
	id2 := generateOperationID(operation, sagaID)

	// IDs should be unique
	assert.NotEqual(t, id1, id2, "Operation IDs should be unique")

	// IDs should contain operation and saga ID
	assert.Contains(t, id1, operation)
	assert.Contains(t, id1, sagaID)
}

func TestCancelSaga_CoordinatorError(t *testing.T) {
	// Setup mock coordinator
	mockCoordinator := new(MockSagaCoordinator)
	mockInstance := new(MockSagaInstance)

	sagaID := "test-saga-error"

	// Mock GetSagaInstance to return a running saga
	mockInstance.On("GetState").Return(saga.StateRunning)
	mockInstance.On("IsTerminal").Return(false)
	mockCoordinator.On("GetSagaInstance", sagaID).Return(mockInstance, nil).Once()

	// Mock CancelSaga to return an error
	mockCoordinator.On("CancelSaga", mock.Anything, sagaID, "Test cancellation").
		Return(errors.New("coordinator internal error"))

	// Create control API
	controlAPI := NewSagaControlAPI(mockCoordinator, nil)
	router := setupControlTestRouter(controlAPI)

	// Create request
	reqBody := CancelSagaRequest{
		Reason: "Test cancellation",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/sagas/"+sagaID+"/cancel", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "cancel_failed", response.Error)
	assert.Contains(t, response.Message, "coordinator internal error")

	// Verify mocks
	mockCoordinator.AssertExpectations(t)
	mockInstance.AssertExpectations(t)
}
