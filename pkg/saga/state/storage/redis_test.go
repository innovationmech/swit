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
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/redis/go-redis/v9"
)

// testRedisAvailable checks if Redis is available for testing
func testRedisAvailable(t *testing.T) bool {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use DB 15 for tests
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis is not available for testing:", err)
		return false
	}

	return true
}

// getTestRedisConfig returns a Redis config for testing
func getTestRedisConfig() *RedisConfig {
	return &RedisConfig{
		Mode:            RedisModeStandalone,
		Addr:            "localhost:6379",
		DB:              15, // Use DB 15 for tests
		KeyPrefix:       fmt.Sprintf("test:saga:%d:", time.Now().UnixNano()),
		TTL:             1 * time.Hour,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolSize:        10,
		MinIdleConns:    2,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		EnableMetrics:   false,
	}
}

// cleanupTestData removes all test data from Redis
func cleanupTestData(t *testing.T, storage *RedisStateStorage) {
	t.Helper()

	ctx := context.Background()
	pattern := storage.config.KeyPrefix + "*"

	iter := storage.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		storage.client.Del(ctx, iter.Val())
	}
	if err := iter.Err(); err != nil {
		t.Logf("Warning: failed to cleanup test data: %v", err)
	}
}

// createTestSagaInstance creates a test saga instance
func createTestSagaInstance(id, defID string, state saga.SagaState) *testSagaInstance {
	now := time.Now()
	startTime := now
	return &testSagaInstance{
		id:           id,
		definitionID: defID,
		state:        state,
		currentStep:  1,
		totalSteps:   3,
		createdAt:    now,
		updatedAt:    now,
		startTime:    startTime,
		endTime:      time.Time{},
		result:       nil,
		error:        nil,
		timeout:      30 * time.Minute,
		metadata:     map[string]interface{}{"test": "value"},
		traceID:      "trace-123",
	}
}

// testSagaInstance is a test implementation of saga.SagaInstance
type testSagaInstance struct {
	id           string
	definitionID string
	state        saga.SagaState
	currentStep  int
	totalSteps   int
	createdAt    time.Time
	updatedAt    time.Time
	startTime    time.Time
	endTime      time.Time
	result       interface{}
	error        *saga.SagaError
	timeout      time.Duration
	metadata     map[string]interface{}
	traceID      string
}

func (t *testSagaInstance) GetID() string                       { return t.id }
func (t *testSagaInstance) GetDefinitionID() string             { return t.definitionID }
func (t *testSagaInstance) GetState() saga.SagaState            { return t.state }
func (t *testSagaInstance) GetCurrentStep() int                 { return t.currentStep }
func (t *testSagaInstance) GetTotalSteps() int                  { return t.totalSteps }
func (t *testSagaInstance) GetStartTime() time.Time             { return t.startTime }
func (t *testSagaInstance) GetEndTime() time.Time               { return t.endTime }
func (t *testSagaInstance) GetResult() interface{}              { return t.result }
func (t *testSagaInstance) GetError() *saga.SagaError           { return t.error }
func (t *testSagaInstance) GetCompletedSteps() int              { return t.currentStep }
func (t *testSagaInstance) GetCreatedAt() time.Time             { return t.createdAt }
func (t *testSagaInstance) GetUpdatedAt() time.Time             { return t.updatedAt }
func (t *testSagaInstance) GetTimeout() time.Duration           { return t.timeout }
func (t *testSagaInstance) GetMetadata() map[string]interface{} { return t.metadata }
func (t *testSagaInstance) GetTraceID() string                  { return t.traceID }
func (t *testSagaInstance) IsTerminal() bool                    { return t.state.IsTerminal() }
func (t *testSagaInstance) IsActive() bool                      { return t.state.IsActive() }

func TestNewRedisStateStorage(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	tests := []struct {
		name    string
		config  *RedisConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  getTestRedisConfig(),
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "invalid address",
			config: &RedisConfig{
				Mode:        RedisModeStandalone,
				Addr:        "invalid:99999",
				DB:          0,
				KeyPrefix:   "test:",
				DialTimeout: 1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewRedisStateStorage(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRedisStateStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				defer storage.Close()
				if storage == nil {
					t.Error("NewRedisStateStorage() returned nil storage")
				}
			}
		})
	}
}

func TestNewRedisStateStorageWithRetry(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	ctx := context.Background()

	storage, err := NewRedisStateStorageWithRetry(ctx, config, 3)
	if err != nil {
		t.Fatalf("NewRedisStateStorageWithRetry() error = %v", err)
	}
	defer storage.Close()

	if storage == nil {
		t.Fatal("NewRedisStateStorageWithRetry() returned nil storage")
	}
}

func TestRedisStateStorage_SaveSaga(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	tests := []struct {
		name     string
		instance saga.SagaInstance
		wantErr  bool
	}{
		{
			name:     "save valid saga",
			instance: createTestSagaInstance("saga-1", "def-1", saga.StateRunning),
			wantErr:  false,
		},
		{
			name:     "save nil saga",
			instance: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.SaveSaga(ctx, tt.instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveSaga() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedisStateStorage_GetSaga(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Save a test saga first
	testSaga := createTestSagaInstance("saga-1", "def-1", saga.StateRunning)
	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("Failed to save test saga: %v", err)
	}

	tests := []struct {
		name    string
		sagaID  string
		wantErr bool
	}{
		{
			name:    "get existing saga",
			sagaID:  "saga-1",
			wantErr: false,
		},
		{
			name:    "get non-existent saga",
			sagaID:  "non-existent",
			wantErr: true,
		},
		{
			name:    "get with empty ID",
			sagaID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance, err := storage.GetSaga(ctx, tt.sagaID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSaga() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if instance.GetID() != tt.sagaID {
					t.Errorf("GetSaga() got ID = %v, want %v", instance.GetID(), tt.sagaID)
				}
				if instance.GetDefinitionID() != "def-1" {
					t.Errorf("GetSaga() got DefinitionID = %v, want def-1", instance.GetDefinitionID())
				}
				if instance.GetState() != saga.StateRunning {
					t.Errorf("GetSaga() got State = %v, want StateRunning", instance.GetState())
				}
			}
		})
	}
}

func TestRedisStateStorage_UpdateSagaState(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Save a test saga first
	testSaga := createTestSagaInstance("saga-1", "def-1", saga.StateRunning)
	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("Failed to save test saga: %v", err)
	}

	tests := []struct {
		name     string
		sagaID   string
		newState saga.SagaState
		metadata map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "update to completed state",
			sagaID:   "saga-1",
			newState: saga.StateCompleted,
			metadata: map[string]interface{}{"result": "success"},
			wantErr:  false,
		},
		{
			name:     "update non-existent saga",
			sagaID:   "non-existent",
			newState: saga.StateCompleted,
			metadata: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.UpdateSagaState(ctx, tt.sagaID, tt.newState, tt.metadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateSagaState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				// Verify the update
				instance, err := storage.GetSaga(ctx, tt.sagaID)
				if err != nil {
					t.Fatalf("Failed to get updated saga: %v", err)
				}
				if instance.GetState() != tt.newState {
					t.Errorf("UpdateSagaState() state = %v, want %v", instance.GetState(), tt.newState)
				}
				if tt.metadata != nil {
					meta := instance.GetMetadata()
					for k, v := range tt.metadata {
						if meta[k] != v {
							t.Errorf("UpdateSagaState() metadata[%s] = %v, want %v", k, meta[k], v)
						}
					}
				}
			}
		})
	}
}

func TestRedisStateStorage_DeleteSaga(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Save a test saga first
	testSaga := createTestSagaInstance("saga-1", "def-1", saga.StateRunning)
	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("Failed to save test saga: %v", err)
	}

	tests := []struct {
		name    string
		sagaID  string
		wantErr bool
	}{
		{
			name:    "delete existing saga",
			sagaID:  "saga-1",
			wantErr: false,
		},
		{
			name:    "delete non-existent saga",
			sagaID:  "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.DeleteSaga(ctx, tt.sagaID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteSaga() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				// Verify the deletion
				_, err := storage.GetSaga(ctx, tt.sagaID)
				if err == nil {
					t.Error("DeleteSaga() saga still exists after deletion")
				}
			}
		})
	}
}

func TestRedisStateStorage_GetActiveSagas(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Save test sagas with different states
	sagas := []saga.SagaInstance{
		createTestSagaInstance("saga-1", "def-1", saga.StateRunning),
		createTestSagaInstance("saga-2", "def-1", saga.StateCompleted),
		createTestSagaInstance("saga-3", "def-2", saga.StateRunning),
		createTestSagaInstance("saga-4", "def-1", saga.StateCompensating),
	}

	for _, s := range sagas {
		if err := storage.SaveSaga(ctx, s); err != nil {
			t.Fatalf("Failed to save test saga: %v", err)
		}
	}

	tests := []struct {
		name      string
		filter    *saga.SagaFilter
		wantCount int
	}{
		{
			name:      "get all active sagas",
			filter:    nil,
			wantCount: 3, // StateRunning, StateRunning, StateCompensating
		},
		{
			name: "filter by state",
			filter: &saga.SagaFilter{
				States: []saga.SagaState{saga.StateRunning},
			},
			wantCount: 2,
		},
		{
			name: "filter by definition ID",
			filter: &saga.SagaFilter{
				DefinitionIDs: []string{"def-1"},
			},
			wantCount: 2, // Only active sagas with def-1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instances, err := storage.GetActiveSagas(ctx, tt.filter)
			if err != nil {
				t.Errorf("GetActiveSagas() error = %v", err)
				return
			}

			if len(instances) != tt.wantCount {
				t.Errorf("GetActiveSagas() count = %v, want %v", len(instances), tt.wantCount)
			}
		})
	}
}

func TestRedisStateStorage_GetTimeoutSagas(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Create sagas with different start times
	now := time.Now()
	oldSaga := createTestSagaInstance("saga-old", "def-1", saga.StateRunning)
	oldSaga.startTime = now.Add(-2 * time.Hour)
	oldSaga.timeout = 1 * time.Hour

	recentSaga := createTestSagaInstance("saga-recent", "def-1", saga.StateRunning)
	recentSaga.startTime = now.Add(-30 * time.Minute)
	recentSaga.timeout = 1 * time.Hour

	if err := storage.SaveSaga(ctx, oldSaga); err != nil {
		t.Fatalf("Failed to save old saga: %v", err)
	}
	if err := storage.SaveSaga(ctx, recentSaga); err != nil {
		t.Fatalf("Failed to save recent saga: %v", err)
	}

	// Query for timeouts before now (should get oldSaga)
	instances, err := storage.GetTimeoutSagas(ctx, now)
	if err != nil {
		t.Fatalf("GetTimeoutSagas() error = %v", err)
	}

	if len(instances) != 1 {
		t.Errorf("GetTimeoutSagas() count = %v, want 1", len(instances))
	}

	if len(instances) > 0 && instances[0].GetID() != "saga-old" {
		t.Errorf("GetTimeoutSagas() got ID = %v, want saga-old", instances[0].GetID())
	}
}

func TestRedisStateStorage_SaveStepState(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Save a test saga first
	testSaga := createTestSagaInstance("saga-1", "def-1", saga.StateRunning)
	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("Failed to save test saga: %v", err)
	}

	now := time.Now()
	step := &saga.StepState{
		ID:        "step-1",
		Name:      "TestStep",
		State:     saga.StepStateCompleted,
		Attempts:  1,
		CreatedAt: now,
		StartedAt: &now,
	}

	tests := []struct {
		name    string
		sagaID  string
		step    *saga.StepState
		wantErr bool
	}{
		{
			name:    "save valid step",
			sagaID:  "saga-1",
			step:    step,
			wantErr: false,
		},
		{
			name:    "save step for non-existent saga",
			sagaID:  "non-existent",
			step:    step,
			wantErr: true,
		},
		{
			name:    "save nil step",
			sagaID:  "saga-1",
			step:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.SaveStepState(ctx, tt.sagaID, tt.step)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveStepState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedisStateStorage_GetStepStates(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Save a test saga first
	testSaga := createTestSagaInstance("saga-1", "def-1", saga.StateRunning)
	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("Failed to save test saga: %v", err)
	}

	// Save step states
	now := time.Now()
	steps := []*saga.StepState{
		{
			ID:        "step-1",
			Name:      "Step1",
			State:     saga.StepStateCompleted,
			Attempts:  1,
			CreatedAt: now,
			StartedAt: &now,
		},
		{
			ID:        "step-2",
			Name:      "Step2",
			State:     saga.StepStateRunning,
			Attempts:  1,
			CreatedAt: now,
			StartedAt: &now,
		},
	}

	for _, step := range steps {
		if err := storage.SaveStepState(ctx, "saga-1", step); err != nil {
			t.Fatalf("Failed to save step state: %v", err)
		}
	}

	tests := []struct {
		name      string
		sagaID    string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "get steps for existing saga",
			sagaID:    "saga-1",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "get steps for non-existent saga",
			sagaID:    "non-existent",
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := storage.GetStepStates(ctx, tt.sagaID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStepStates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && len(steps) != tt.wantCount {
				t.Errorf("GetStepStates() count = %v, want %v", len(steps), tt.wantCount)
			}
		})
	}
}

func TestRedisStateStorage_CleanupExpiredSagas(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Create sagas with different update times
	now := time.Now()
	oldSaga := createTestSagaInstance("saga-old", "def-1", saga.StateCompleted)
	oldSaga.updatedAt = now.Add(-2 * time.Hour)

	recentSaga := createTestSagaInstance("saga-recent", "def-1", saga.StateCompleted)
	recentSaga.updatedAt = now.Add(-30 * time.Minute)

	if err := storage.SaveSaga(ctx, oldSaga); err != nil {
		t.Fatalf("Failed to save old saga: %v", err)
	}
	if err := storage.SaveSaga(ctx, recentSaga); err != nil {
		t.Fatalf("Failed to save recent saga: %v", err)
	}

	// Cleanup sagas older than 1 hour
	olderThan := now.Add(-1 * time.Hour)
	if err := storage.CleanupExpiredSagas(ctx, olderThan); err != nil {
		t.Fatalf("CleanupExpiredSagas() error = %v", err)
	}

	// Verify old saga is deleted
	_, err = storage.GetSaga(ctx, "saga-old")
	if err == nil {
		t.Error("CleanupExpiredSagas() old saga should be deleted")
	}

	// Verify recent saga still exists
	_, err = storage.GetSaga(ctx, "saga-recent")
	if err != nil {
		t.Error("CleanupExpiredSagas() recent saga should still exist")
	}
}

func TestRedisStateStorage_Close(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Close the storage
	if err := storage.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify operations fail after close
	ctx := context.Background()
	testSaga := createTestSagaInstance("saga-1", "def-1", saga.StateRunning)
	err = storage.SaveSaga(ctx, testSaga)
	if err == nil {
		t.Error("SaveSaga() should fail after Close()")
	}

	// Verify Close() is idempotent
	if err := storage.Close(); err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

func TestRedisStateStorage_HealthCheck(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	if err := storage.HealthCheck(ctx); err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}

	// Close and verify health check fails
	storage.Close()
	if err := storage.HealthCheck(ctx); err == nil {
		t.Error("HealthCheck() should fail after Close()")
	}
}

func TestRedisStateStorage_KeyGeneration(t *testing.T) {
	config := &RedisConfig{
		KeyPrefix: "test:",
	}

	storage := &RedisStateStorage{
		config: config,
	}

	tests := []struct {
		name     string
		fn       func() string
		expected string
	}{
		{
			name:     "saga key",
			fn:       func() string { return storage.getSagaKey("saga-1") },
			expected: "test:saga:saga-1",
		},
		{
			name:     "step key",
			fn:       func() string { return storage.getStepKey("saga-1", "step-1") },
			expected: "test:step:saga-1:step-1",
		},
		{
			name:     "steps set key",
			fn:       func() string { return storage.getStepsSetKey("saga-1") },
			expected: "test:steps:saga-1",
		},
		{
			name:     "state index key",
			fn:       func() string { return storage.getStateIndexKey("running") },
			expected: "test:index:state:running",
		},
		{
			name:     "timeout key",
			fn:       func() string { return storage.getTimeoutKey() },
			expected: "test:timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got != tt.expected {
				t.Errorf("key = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRedisStateStorage_ConcurrentAccess(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Test concurrent writes
	const numGoroutines = 10
	errCh := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			sagaInst := createTestSagaInstance(fmt.Sprintf("saga-%d", id), "def-1", saga.StateRunning)
			errCh <- storage.SaveSaga(ctx, sagaInst)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("Concurrent SaveSaga() error = %v", err)
		}
	}

	// Verify all sagas were saved
	for i := 0; i < numGoroutines; i++ {
		sagaID := fmt.Sprintf("saga-%d", i)
		_, err := storage.GetSaga(ctx, sagaID)
		if err != nil {
			t.Errorf("GetSaga(%s) error = %v", sagaID, err)
		}
	}
}

// TestRedisStateStorage_FailureScenarios tests various failure scenarios
func TestRedisStateStorage_FailureScenarios(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	tests := []struct {
		name    string
		fn      func() error
		wantErr bool
	}{
		{
			name: "save saga with empty ID",
			fn: func() error {
				sagaInst := createTestSagaInstance("", "def-1", saga.StateRunning)
				return storage.SaveSaga(ctx, sagaInst)
			},
			wantErr: true,
		},
		{
			name: "get saga with empty ID",
			fn: func() error {
				_, err := storage.GetSaga(ctx, "")
				return err
			},
			wantErr: true,
		},
		{
			name: "update saga with empty ID",
			fn: func() error {
				return storage.UpdateSagaState(ctx, "", saga.StateCompleted, nil)
			},
			wantErr: true,
		},
		{
			name: "delete saga with empty ID",
			fn: func() error {
				return storage.DeleteSaga(ctx, "")
			},
			wantErr: true,
		},
		{
			name: "save step state with empty saga ID",
			fn: func() error {
				now := time.Now()
				step := &saga.StepState{
					ID:        "step-1",
					Name:      "Step1",
					State:     saga.StepStateCompleted,
					Attempts:  1,
					CreatedAt: now,
					StartedAt: &now,
				}
				return storage.SaveStepState(ctx, "", step)
			},
			wantErr: true,
		},
		{
			name: "get step states with empty saga ID",
			fn: func() error {
				_, err := storage.GetStepStates(ctx, "")
				return err
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRedisStateStorage_ContextCancellation tests context cancellation
func TestRedisStateStorage_ContextCancellation(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sagaInst := createTestSagaInstance("saga-cancelled", "def-1", saga.StateRunning)
	err = storage.SaveSaga(ctx, sagaInst)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestRedisStateStorage_ClosedStorageOperations tests operations on closed storage
func TestRedisStateStorage_ClosedStorageOperations(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Close the storage
	storage.Close()

	ctx := context.Background()
	sagaInst := createTestSagaInstance("saga-1", "def-1", saga.StateRunning)

	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "SaveSaga on closed storage",
			fn:   func() error { return storage.SaveSaga(ctx, sagaInst) },
		},
		{
			name: "GetSaga on closed storage",
			fn:   func() error { _, err := storage.GetSaga(ctx, "saga-1"); return err },
		},
		{
			name: "UpdateSagaState on closed storage",
			fn:   func() error { return storage.UpdateSagaState(ctx, "saga-1", saga.StateCompleted, nil) },
		},
		{
			name: "DeleteSaga on closed storage",
			fn:   func() error { return storage.DeleteSaga(ctx, "saga-1") },
		},
		{
			name: "GetActiveSagas on closed storage",
			fn:   func() error { _, err := storage.GetActiveSagas(ctx, nil); return err },
		},
		{
			name: "GetTimeoutSagas on closed storage",
			fn:   func() error { _, err := storage.GetTimeoutSagas(ctx, time.Now()); return err },
		},
		{
			name: "CleanupExpiredSagas on closed storage",
			fn:   func() error { return storage.CleanupExpiredSagas(ctx, time.Now()) },
		},
		{
			name: "HealthCheck on closed storage",
			fn:   func() error { return storage.HealthCheck(ctx) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Error("Expected error on closed storage, got nil")
			}
		})
	}
}

// TestRedisStateStorage_ConcurrentUpdates tests concurrent updates to the same saga
func TestRedisStateStorage_ConcurrentUpdates(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Create initial saga
	sagaInst := createTestSagaInstance("saga-concurrent-update", "def-1", saga.StateRunning)
	if err := storage.SaveSaga(ctx, sagaInst); err != nil {
		t.Fatalf("Failed to save initial saga: %v", err)
	}

	// Launch multiple goroutines to update the same saga
	const numUpdates = 20
	errCh := make(chan error, numUpdates)

	for i := 0; i < numUpdates; i++ {
		go func(iteration int) {
			metadata := map[string]interface{}{
				"update": iteration,
			}
			errCh <- storage.UpdateSagaState(ctx, "saga-concurrent-update", saga.StateRunning, metadata)
		}(i)
	}

	// Check for errors
	for i := 0; i < numUpdates; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("Concurrent update error: %v", err)
		}
	}

	// Verify saga still exists and is in a valid state
	retrieved, err := storage.GetSaga(ctx, "saga-concurrent-update")
	if err != nil {
		t.Fatalf("Failed to get saga after concurrent updates: %v", err)
	}
	if retrieved.GetID() != "saga-concurrent-update" {
		t.Errorf("Saga ID mismatch after concurrent updates")
	}
}

// TestRedisStateStorage_StressTest tests storage under high load
func TestRedisStateStorage_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	const (
		numGoroutines = 50
		opsPerRoutine = 100
	)

	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines*opsPerRoutine)

	// Launch concurrent workers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < opsPerRoutine; j++ {
				sagaID := fmt.Sprintf("saga-stress-%d-%d", workerID, j)
				sagaInst := createTestSagaInstance(sagaID, "def-stress", saga.StateRunning)

				// Save
				if err := storage.SaveSaga(ctx, sagaInst); err != nil {
					errCh <- fmt.Errorf("save error: %w", err)
					continue
				}

				// Get
				if _, err := storage.GetSaga(ctx, sagaID); err != nil {
					errCh <- fmt.Errorf("get error: %w", err)
					continue
				}

				// Update
				if err := storage.UpdateSagaState(ctx, sagaID, saga.StateCompleted, nil); err != nil {
					errCh <- fmt.Errorf("update error: %w", err)
					continue
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	// Check for errors
	errorCount := 0
	for err := range errCh {
		t.Errorf("Stress test error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Fatalf("Stress test failed with %d errors", errorCount)
	}
}

// TestRedisStateStorage_InvalidStateTransitions tests handling of invalid state transitions
func TestRedisStateStorage_InvalidStateTransitions(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Create a completed saga
	sagaInst := createTestSagaInstance("saga-state-trans", "def-1", saga.StateCompleted)
	if err := storage.SaveSaga(ctx, sagaInst); err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Try to transition back to running (should succeed as storage doesn't enforce state machine)
	err = storage.UpdateSagaState(ctx, "saga-state-trans", saga.StateRunning, nil)
	if err != nil {
		t.Errorf("UpdateSagaState should not enforce state transitions, got error: %v", err)
	}

	// Verify the state was updated
	retrieved, err := storage.GetSaga(ctx, "saga-state-trans")
	if err != nil {
		t.Fatalf("Failed to get saga: %v", err)
	}
	if retrieved.GetState() != saga.StateRunning {
		t.Errorf("State = %v, want StateRunning", retrieved.GetState())
	}
}

// TestRedisStateStorage_StepStatePersistence tests step state persistence across saga updates
func TestRedisStateStorage_StepStatePersistence(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	// Create saga
	sagaInst := createTestSagaInstance("saga-step-persist", "def-1", saga.StateRunning)
	if err := storage.SaveSaga(ctx, sagaInst); err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Save step states
	now := time.Now()
	step1 := &saga.StepState{
		ID:        "step-1",
		Name:      "Step1",
		State:     saga.StepStateCompleted,
		Attempts:  1,
		CreatedAt: now,
		StartedAt: &now,
	}
	if err := storage.SaveStepState(ctx, "saga-step-persist", step1); err != nil {
		t.Fatalf("Failed to save step state: %v", err)
	}

	// Update saga state
	if err := storage.UpdateSagaState(ctx, "saga-step-persist", saga.StateStepCompleted, nil); err != nil {
		t.Fatalf("Failed to update saga state: %v", err)
	}

	// Verify step state persists
	steps, err := storage.GetStepStates(ctx, "saga-step-persist")
	if err != nil {
		t.Fatalf("Failed to get step states: %v", err)
	}
	if len(steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(steps))
	}
	if len(steps) > 0 && steps[0].ID != "step-1" {
		t.Errorf("Step ID = %v, want step-1", steps[0].ID)
	}
}

// TestRedisStateStorage_MetadataHandling tests various metadata handling scenarios
func TestRedisStateStorage_MetadataHandling(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()

	tests := []struct {
		name     string
		metadata map[string]interface{}
	}{
		{
			name:     "nil metadata",
			metadata: nil,
		},
		{
			name:     "empty metadata",
			metadata: map[string]interface{}{},
		},
		{
			name: "complex metadata",
			metadata: map[string]interface{}{
				"string": "value",
				"number": 42,
				"float":  3.14,
				"bool":   true,
				"nested": map[string]interface{}{
					"key": "value",
				},
				"array": []interface{}{"a", "b", "c"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sagaID := fmt.Sprintf("saga-meta-%s", tt.name)
			sagaInst := createTestSagaInstance(sagaID, "def-meta", saga.StateRunning)
			sagaInst.metadata = tt.metadata

			// Save
			if err := storage.SaveSaga(ctx, sagaInst); err != nil {
				t.Fatalf("Failed to save saga: %v", err)
			}

			// Retrieve
			retrieved, err := storage.GetSaga(ctx, sagaID)
			if err != nil {
				t.Fatalf("Failed to get saga: %v", err)
			}

			// Verify metadata
			if tt.metadata == nil {
				if retrieved.GetMetadata() == nil {
					// Both nil is ok
					return
				}
			}
		})
	}
}

// TestRedisStateStorage_TimeoutIndexManagement tests timeout index management
func TestRedisStateStorage_TimeoutIndexManagement(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()
	now := time.Now()

	// Create saga that will timeout
	sagaInst := createTestSagaInstance("saga-timeout-idx", "def-1", saga.StateRunning)
	sagaInst.startTime = now.Add(-2 * time.Hour)
	sagaInst.timeout = 1 * time.Hour
	if err := storage.SaveSaga(ctx, sagaInst); err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Verify it's in timeout index
	timeouts, err := storage.GetTimeoutSagas(ctx, now)
	if err != nil {
		t.Fatalf("Failed to get timeout sagas: %v", err)
	}
	if len(timeouts) != 1 {
		t.Errorf("Expected 1 timeout saga, got %d", len(timeouts))
	}

	// Update to terminal state
	if err := storage.UpdateSagaState(ctx, "saga-timeout-idx", saga.StateCompleted, nil); err != nil {
		t.Fatalf("Failed to update saga state: %v", err)
	}

	// Verify it's removed from timeout index
	timeouts, err = storage.GetTimeoutSagas(ctx, now)
	if err != nil {
		t.Fatalf("Failed to get timeout sagas: %v", err)
	}
	if len(timeouts) != 0 {
		t.Errorf("Expected 0 timeout sagas after completion, got %d", len(timeouts))
	}
}

// TestRedisStateStorage_DeletionCleansUpIndexes tests that deletion cleans up all indexes
func TestRedisStateStorage_DeletionCleansUpIndexes(t *testing.T) {
	if !testRedisAvailable(t) {
		return
	}

	config := getTestRedisConfig()
	storage, err := NewRedisStateStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	defer cleanupTestData(t, storage)

	ctx := context.Background()
	now := time.Now()

	// Create saga with timeout
	sagaInst := createTestSagaInstance("saga-delete-idx", "def-1", saga.StateRunning)
	sagaInst.startTime = now
	sagaInst.timeout = 1 * time.Hour
	if err := storage.SaveSaga(ctx, sagaInst); err != nil {
		t.Fatalf("Failed to save saga: %v", err)
	}

	// Add step states
	step := &saga.StepState{
		ID:        "step-1",
		Name:      "Step1",
		State:     saga.StepStateCompleted,
		Attempts:  1,
		CreatedAt: now,
		StartedAt: &now,
	}
	if err := storage.SaveStepState(ctx, "saga-delete-idx", step); err != nil {
		t.Fatalf("Failed to save step state: %v", err)
	}

	// Delete saga
	if err := storage.DeleteSaga(ctx, "saga-delete-idx"); err != nil {
		t.Fatalf("Failed to delete saga: %v", err)
	}

	// Verify state index is cleaned up
	stateIndexKey := storage.getStateIndexKey(saga.StateRunning.String())
	isMember := storage.client.SIsMember(ctx, stateIndexKey, "saga-delete-idx").Val()
	if isMember {
		t.Error("Saga should not be in state index after deletion")
	}

	// Verify timeout index is cleaned up
	timeoutKey := storage.getTimeoutKey()
	score := storage.client.ZScore(ctx, timeoutKey, "saga-delete-idx").Val()
	if score != 0 {
		// ZScore returns 0 if member doesn't exist, but we need to check Err() too
		_, err := storage.client.ZScore(ctx, timeoutKey, "saga-delete-idx").Result()
		if err == nil {
			t.Error("Saga should not be in timeout index after deletion")
		}
	}

	// Verify step states are cleaned up
	_, err = storage.GetStepStates(ctx, "saga-delete-idx")
	if err == nil {
		t.Error("Expected error when getting steps for deleted saga")
	}
}
