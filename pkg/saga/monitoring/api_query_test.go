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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockQueryCoordinator is a simple mock for testing query API
type mockQueryCoordinator struct {
	mock.Mock
	sagas []saga.SagaInstance
}

func (m *mockQueryCoordinator) StartSaga(ctx context.Context, definition saga.SagaDefinition, initialData interface{}) (saga.SagaInstance, error) {
	args := m.Called(ctx, definition, initialData)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(saga.SagaInstance), args.Error(1)
}

func (m *mockQueryCoordinator) GetSagaInstance(sagaID string) (saga.SagaInstance, error) {
	args := m.Called(sagaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(saga.SagaInstance), args.Error(1)
}

func (m *mockQueryCoordinator) CancelSaga(ctx context.Context, sagaID string, reason string) error {
	args := m.Called(ctx, sagaID, reason)
	return args.Error(0)
}

func (m *mockQueryCoordinator) GetActiveSagas(filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	args := m.Called(filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]saga.SagaInstance), args.Error(1)
}

func (m *mockQueryCoordinator) GetMetrics() *saga.CoordinatorMetrics {
	args := m.Called()
	return args.Get(0).(*saga.CoordinatorMetrics)
}

func (m *mockQueryCoordinator) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockQueryCoordinator) Close() error {
	args := m.Called()
	return args.Error(0)
}

// mockQuerySagaInstance is a simple mock for testing query API
type mockQuerySagaInstance struct {
	id             string
	definitionID   string
	state          saga.SagaState
	currentStep    int
	startTime      time.Time
	endTime        time.Time
	result         interface{}
	err            *saga.SagaError
	totalSteps     int
	completedSteps int
	createdAt      time.Time
	updatedAt      time.Time
	timeout        time.Duration
	metadata       map[string]interface{}
	traceID        string
}

func (m *mockQuerySagaInstance) GetID() string                       { return m.id }
func (m *mockQuerySagaInstance) GetDefinitionID() string             { return m.definitionID }
func (m *mockQuerySagaInstance) GetState() saga.SagaState            { return m.state }
func (m *mockQuerySagaInstance) GetCurrentStep() int                 { return m.currentStep }
func (m *mockQuerySagaInstance) GetStartTime() time.Time             { return m.startTime }
func (m *mockQuerySagaInstance) GetEndTime() time.Time               { return m.endTime }
func (m *mockQuerySagaInstance) GetResult() interface{}              { return m.result }
func (m *mockQuerySagaInstance) GetError() *saga.SagaError           { return m.err }
func (m *mockQuerySagaInstance) GetTotalSteps() int                  { return m.totalSteps }
func (m *mockQuerySagaInstance) GetCompletedSteps() int              { return m.completedSteps }
func (m *mockQuerySagaInstance) GetCreatedAt() time.Time             { return m.createdAt }
func (m *mockQuerySagaInstance) GetUpdatedAt() time.Time             { return m.updatedAt }
func (m *mockQuerySagaInstance) GetTimeout() time.Duration           { return m.timeout }
func (m *mockQuerySagaInstance) GetMetadata() map[string]interface{} { return m.metadata }
func (m *mockQuerySagaInstance) GetTraceID() string                  { return m.traceID }
func (m *mockQuerySagaInstance) IsTerminal() bool                    { return m.state.IsTerminal() }
func (m *mockQuerySagaInstance) IsActive() bool                      { return m.state.IsActive() }

// setupTestRouter creates a test router with the query API.
func setupTestRouter(coordinator saga.SagaCoordinator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	queryAPI := NewSagaQueryAPI(coordinator)

	// Setup routes
	apiGroup := router.Group("/api")
	sagaGroup := apiGroup.Group("/sagas")
	{
		sagaGroup.GET("", queryAPI.ListSagas)
		sagaGroup.GET("/:id", queryAPI.GetSagaDetails)
	}

	return router
}

func TestSagaQueryAPI_ListSagas(t *testing.T) {
	t.Run("successful list with defaults", func(t *testing.T) {
		// Create mock coordinator
		mockCoord := new(mockQueryCoordinator)

		// Setup mock instances
		now := time.Now()
		instances := []saga.SagaInstance{
			&mockQuerySagaInstance{
				id:             "saga-1",
				definitionID:   "test-def",
				state:          saga.StateRunning,
				totalSteps:     5,
				completedSteps: 2,
				createdAt:      now,
				updatedAt:      now,
				startTime:      now,
			},
			&mockQuerySagaInstance{
				id:             "saga-2",
				definitionID:   "test-def",
				state:          saga.StateRunning,
				totalSteps:     5,
				completedSteps: 3,
				createdAt:      now,
				updatedAt:      now,
				startTime:      now,
			},
		}

		mockCoord.On("GetActiveSagas", mock.Anything).Return(instances, nil)

		// Create router
		router := setupTestRouter(mockCoord)

		// Create request
		req, _ := http.NewRequest("GET", "/api/sagas", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var resp SagaListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Data, 2)
		assert.Equal(t, 1, resp.Pagination.Page)
		assert.Equal(t, 20, resp.Pagination.PageSize)
	})

	t.Run("list with pagination", func(t *testing.T) {
		mockCoord := new(mockQueryCoordinator)

		// Create 5 mock instances
		now := time.Now()
		instances := make([]saga.SagaInstance, 5)
		for i := 0; i < 5; i++ {
			instances[i] = &mockQuerySagaInstance{
				id:             "saga-" + string(rune(i)),
				definitionID:   "test-def",
				state:          saga.StateRunning,
				totalSteps:     5,
				completedSteps: i,
				createdAt:      now,
				updatedAt:      now,
				startTime:      now,
			}
		}

		mockCoord.On("GetActiveSagas", mock.Anything).Return(instances, nil)

		router := setupTestRouter(mockCoord)
		req, _ := http.NewRequest("GET", "/api/sagas?page=2&pageSize=2", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp SagaListResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Len(t, resp.Data, 2)
		assert.Equal(t, 2, resp.Pagination.Page)
		assert.Equal(t, 3, resp.Pagination.TotalPages)
	})

	t.Run("invalid page parameter", func(t *testing.T) {
		mockCoord := new(mockQueryCoordinator)
		router := setupTestRouter(mockCoord)

		req, _ := http.NewRequest("GET", "/api/sagas?page=-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty list", func(t *testing.T) {
		mockCoord := new(mockQueryCoordinator)
		mockCoord.On("GetActiveSagas", mock.Anything).Return([]saga.SagaInstance{}, nil)

		router := setupTestRouter(mockCoord)
		req, _ := http.NewRequest("GET", "/api/sagas", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp SagaListResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Empty(t, resp.Data)
		assert.Equal(t, 0, resp.Pagination.TotalItems)
	})
}

func TestSagaQueryAPI_GetSagaDetails(t *testing.T) {
	t.Run("successful get saga details", func(t *testing.T) {
		mockCoord := new(mockQueryCoordinator)
		now := time.Now()

		instance := &mockQuerySagaInstance{
			id:             "saga-123",
			definitionID:   "test-def",
			state:          saga.StateRunning,
			totalSteps:     5,
			completedSteps: 2,
			createdAt:      now,
			updatedAt:      now,
			startTime:      now,
			timeout:        30 * time.Second,
		}

		mockCoord.On("GetSagaInstance", "saga-123").Return(instance, nil)

		router := setupTestRouter(mockCoord)
		req, _ := http.NewRequest("GET", "/api/sagas/saga-123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp SagaDetailDTO
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "saga-123", resp.ID)
		assert.Equal(t, "test-def", resp.DefinitionID)
		assert.NotNil(t, resp.ExecutionDetails)
	})

	t.Run("saga not found", func(t *testing.T) {
		mockCoord := new(mockQueryCoordinator)
		mockCoord.On("GetSagaInstance", "non-existent").Return(nil, saga.NewSagaNotFoundError("non-existent"))

		router := setupTestRouter(mockCoord)
		req, _ := http.NewRequest("GET", "/api/sagas/non-existent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestSagaListRequest_ApplyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		req      SagaListRequest
		expected SagaListRequest
	}{
		{
			name: "apply all defaults",
			req:  SagaListRequest{},
			expected: SagaListRequest{
				Page:      1,
				PageSize:  20,
				SortBy:    "created_at",
				SortOrder: "desc",
			},
		},
		{
			name: "preserve existing values",
			req: SagaListRequest{
				Page:      2,
				PageSize:  50,
				SortBy:    "updated_at",
				SortOrder: "asc",
			},
			expected: SagaListRequest{
				Page:      2,
				PageSize:  50,
				SortBy:    "updated_at",
				SortOrder: "asc",
			},
		},
		{
			name: "fix invalid page",
			req: SagaListRequest{
				Page:     0,
				PageSize: 10,
			},
			expected: SagaListRequest{
				Page:      1,
				PageSize:  10,
				SortBy:    "created_at",
				SortOrder: "desc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.req.ApplyDefaults()

			if tt.req.Page != tt.expected.Page {
				t.Errorf("Expected page %d, got %d", tt.expected.Page, tt.req.Page)
			}
			if tt.req.PageSize != tt.expected.PageSize {
				t.Errorf("Expected pageSize %d, got %d", tt.expected.PageSize, tt.req.PageSize)
			}
			if tt.req.SortBy != tt.expected.SortBy {
				t.Errorf("Expected sortBy '%s', got '%s'", tt.expected.SortBy, tt.req.SortBy)
			}
			if tt.req.SortOrder != tt.expected.SortOrder {
				t.Errorf("Expected sortOrder '%s', got '%s'", tt.expected.SortOrder, tt.req.SortOrder)
			}
		})
	}
}

func TestFromSagaInstance(t *testing.T) {
	now := time.Now()
	instance := &mockQuerySagaInstance{
		id:             "test-saga",
		definitionID:   "test-def",
		state:          saga.StateRunning,
		totalSteps:     5,
		completedSteps: 2,
		createdAt:      now,
		updatedAt:      now,
		startTime:      now,
	}

	dto := FromSagaInstance(instance)

	assert.NotNil(t, dto)
	assert.Equal(t, "test-saga", dto.ID)
	assert.Equal(t, "test-def", dto.DefinitionID)
	assert.Equal(t, "running", dto.State)
	assert.Equal(t, 5, dto.TotalSteps)
}

func TestFromSagaInstanceDetailed(t *testing.T) {
	now := time.Now()
	pastTime := now.Add(-5 * time.Second)
	instance := &mockQuerySagaInstance{
		id:             "test-saga",
		definitionID:   "test-def",
		state:          saga.StateRunning,
		totalSteps:     5,
		completedSteps: 2,
		createdAt:      pastTime,
		updatedAt:      now,
		startTime:      pastTime, // Start time in the past so duration > 0
	}

	dto := FromSagaInstanceDetailed(instance)

	assert.NotNil(t, dto)
	assert.NotNil(t, dto.SagaDTO)
	assert.NotNil(t, dto.ExecutionDetails)
	assert.GreaterOrEqual(t, dto.ExecutionDetails.Duration, int64(0))
}

func TestApplyFilters(t *testing.T) {
	now := time.Now()
	instances := []saga.SagaInstance{
		&mockQuerySagaInstance{
			id:           "saga-1",
			definitionID: "def-1",
			state:        saga.StateRunning,
			createdAt:    now,
		},
		&mockQuerySagaInstance{
			id:           "saga-2",
			definitionID: "def-2",
			state:        saga.StateRunning,
			createdAt:    now,
		},
		&mockQuerySagaInstance{
			id:           "saga-3",
			definitionID: "def-1",
			state:        saga.StateCompleted,
			createdAt:    now,
		},
	}

	mockCoord := new(mockQueryCoordinator)
	api := NewSagaQueryAPI(mockCoord)

	t.Run("no filters", func(t *testing.T) {
		filtered := api.applyFilters(instances, &SagaListRequest{})
		assert.Len(t, filtered, 3)
	})

	t.Run("filter by definition ID", func(t *testing.T) {
		filtered := api.applyFilters(instances, &SagaListRequest{
			DefinitionIDs: []string{"def-1"},
		})
		assert.Len(t, filtered, 2)
	})

	t.Run("filter by state", func(t *testing.T) {
		filtered := api.applyFilters(instances, &SagaListRequest{
			States: []string{"running"}, // Use lowercase as returned by String() method
		})
		assert.Len(t, filtered, 2)
	})
}
