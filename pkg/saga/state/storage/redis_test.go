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
			saga := createTestSagaInstance(fmt.Sprintf("saga-%d", id), "def-1", saga.StateRunning)
			errCh <- storage.SaveSaga(ctx, saga)
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
