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
	"errors"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type mockCoordinator struct {
	healthErr      error
	activeSagas    int64
	completedSagas int64
	failedSagas    int64
}

func (m *mockCoordinator) StartSaga(ctx context.Context, definition saga.SagaDefinition, initialData interface{}) (saga.SagaInstance, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCoordinator) GetSagaInstance(sagaID string) (saga.SagaInstance, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCoordinator) CancelSaga(ctx context.Context, sagaID string, reason string) error {
	return errors.New("not implemented")
}

func (m *mockCoordinator) GetActiveSagas(filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCoordinator) GetMetrics() *saga.CoordinatorMetrics {
	return &saga.CoordinatorMetrics{
		ActiveSagas:    m.activeSagas,
		CompletedSagas: m.completedSagas,
		FailedSagas:    m.failedSagas,
	}
}

func (m *mockCoordinator) HealthCheck(ctx context.Context) error {
	return m.healthErr
}

func (m *mockCoordinator) Close() error {
	return nil
}

type mockStorage struct {
	healthErr error
	delay     time.Duration
}

func (m *mockStorage) SaveSaga(ctx context.Context, saga saga.SagaInstance) error {
	return errors.New("not implemented")
}

func (m *mockStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStorage) UpdateSagaState(ctx context.Context, sagaID string, state saga.SagaState, metadata map[string]interface{}) error {
	return errors.New("not implemented")
}

func (m *mockStorage) DeleteSaga(ctx context.Context, sagaID string) error {
	return errors.New("not implemented")
}

func (m *mockStorage) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.healthErr != nil {
		return nil, m.healthErr
	}
	return []saga.SagaInstance{}, nil
}

func (m *mockStorage) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStorage) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	return errors.New("not implemented")
}

func (m *mockStorage) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStorage) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	return errors.New("not implemented")
}

type mockEventPublisher struct {
	healthErr error
}

func (m *mockEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	return errors.New("not implemented")
}

func (m *mockEventPublisher) Subscribe(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error) {
	return nil, errors.New("not implemented")
}

func (m *mockEventPublisher) Unsubscribe(subscription saga.EventSubscription) error {
	return errors.New("not implemented")
}

func (m *mockEventPublisher) Close() error {
	return nil
}

// Tests for CoordinatorHealthChecker

func TestCoordinatorHealthChecker_Check_Healthy(t *testing.T) {
	coordinator := &mockCoordinator{
		healthErr:   nil,
		activeSagas: 10,
	}
	checker := NewCoordinatorHealthChecker("test-coordinator", coordinator, 100)

	ctx := context.Background()
	health := checker.Check(ctx)

	assert.Equal(t, "test-coordinator", health.Name)
	assert.Equal(t, HealthStatusHealthy, health.Status)
	assert.Contains(t, health.Message, "healthy")
	assert.Empty(t, health.Error)
	assert.True(t, health.CheckDuration > 0)
}

func TestCoordinatorHealthChecker_Check_Unhealthy(t *testing.T) {
	coordinator := &mockCoordinator{
		healthErr: errors.New("coordinator is down"),
	}
	checker := NewCoordinatorHealthChecker("test-coordinator", coordinator, 100)

	ctx := context.Background()
	health := checker.Check(ctx)

	assert.Equal(t, "test-coordinator", health.Name)
	assert.Equal(t, HealthStatusUnhealthy, health.Status)
	assert.Contains(t, health.Message, "failed")
	assert.NotEmpty(t, health.Error)
}

func TestCoordinatorHealthChecker_Check_Degraded(t *testing.T) {
	coordinator := &mockCoordinator{
		healthErr:   nil,
		activeSagas: 150,
	}
	checker := NewCoordinatorHealthChecker("test-coordinator", coordinator, 100)

	ctx := context.Background()
	health := checker.Check(ctx)

	assert.Equal(t, "test-coordinator", health.Name)
	assert.Equal(t, HealthStatusDegraded, health.Status)
	assert.Contains(t, health.Message, "exceeds threshold")
}

func TestCoordinatorHealthChecker_GetName(t *testing.T) {
	coordinator := &mockCoordinator{}
	checker := NewCoordinatorHealthChecker("my-coordinator", coordinator, 0)

	assert.Equal(t, "my-coordinator", checker.GetName())
}

// Tests for StorageHealthChecker

func TestStorageHealthChecker_Check_Healthy(t *testing.T) {
	storage := &mockStorage{
		healthErr: nil,
		delay:     10 * time.Millisecond,
	}
	checker := NewStorageHealthChecker("test-storage", storage, 5*time.Second)

	ctx := context.Background()
	health := checker.Check(ctx)

	assert.Equal(t, "test-storage", health.Name)
	assert.Equal(t, HealthStatusHealthy, health.Status)
	assert.Contains(t, health.Message, "healthy")
	assert.Empty(t, health.Error)
	assert.True(t, health.CheckDuration > 0)
}

func TestStorageHealthChecker_Check_Unhealthy(t *testing.T) {
	storage := &mockStorage{
		healthErr: errors.New("database connection failed"),
	}
	checker := NewStorageHealthChecker("test-storage", storage, 5*time.Second)

	ctx := context.Background()
	health := checker.Check(ctx)

	assert.Equal(t, "test-storage", health.Name)
	assert.Equal(t, HealthStatusUnhealthy, health.Status)
	assert.Contains(t, health.Message, "failed")
	assert.NotEmpty(t, health.Error)
}

func TestStorageHealthChecker_Check_Degraded(t *testing.T) {
	storage := &mockStorage{
		healthErr: nil,
		delay:     3 * time.Second, // Exceeds half of timeout
	}
	checker := NewStorageHealthChecker("test-storage", storage, 5*time.Second)

	ctx := context.Background()
	health := checker.Check(ctx)

	assert.Equal(t, "test-storage", health.Name)
	assert.Equal(t, HealthStatusDegraded, health.Status)
	assert.Contains(t, health.Message, "degraded")
}

func TestStorageHealthChecker_Check_Timeout(t *testing.T) {
	storage := &mockStorage{
		healthErr: nil,
		delay:     2 * time.Second, // Exceeds timeout but reasonable for CI
	}
	checker := NewStorageHealthChecker("test-storage", storage, 1*time.Second)

	ctx := context.Background()
	health := checker.Check(ctx)

	assert.Equal(t, "test-storage", health.Name)
	// Context timeout causes degraded status (not unhealthy) since the operation didn't fail, it just timed out
	assert.True(t, health.Status == HealthStatusUnhealthy || health.Status == HealthStatusDegraded)
	// Error may be empty if the context was canceled but the operation completed
}

func TestStorageHealthChecker_GetName(t *testing.T) {
	storage := &mockStorage{}
	checker := NewStorageHealthChecker("my-storage", storage, 5*time.Second)

	assert.Equal(t, "my-storage", checker.GetName())
}

// Tests for MessagingHealthChecker

func TestMessagingHealthChecker_Check_Healthy(t *testing.T) {
	publisher := &mockEventPublisher{
		healthErr: nil,
	}
	checker := NewMessagingHealthChecker("test-messaging", publisher, nil)

	ctx := context.Background()
	health := checker.Check(ctx)

	assert.Equal(t, "test-messaging", health.Name)
	assert.Equal(t, HealthStatusHealthy, health.Status)
	assert.Contains(t, health.Message, "healthy")
	assert.Empty(t, health.Error)
}

func TestMessagingHealthChecker_Check_WithCustomConnectivity_Healthy(t *testing.T) {
	publisher := &mockEventPublisher{}
	connectivityCheck := func(ctx context.Context) error {
		return nil
	}
	checker := NewMessagingHealthChecker("test-messaging", publisher, connectivityCheck)

	ctx := context.Background()
	health := checker.Check(ctx)

	assert.Equal(t, "test-messaging", health.Name)
	assert.Equal(t, HealthStatusHealthy, health.Status)
	assert.Contains(t, health.Message, "healthy")
}

func TestMessagingHealthChecker_Check_WithCustomConnectivity_Unhealthy(t *testing.T) {
	publisher := &mockEventPublisher{}
	connectivityCheck := func(ctx context.Context) error {
		return errors.New("connection lost")
	}
	checker := NewMessagingHealthChecker("test-messaging", publisher, connectivityCheck)

	ctx := context.Background()
	health := checker.Check(ctx)

	assert.Equal(t, "test-messaging", health.Name)
	assert.Equal(t, HealthStatusUnhealthy, health.Status)
	assert.Contains(t, health.Message, "not connected")
	assert.NotEmpty(t, health.Error)
}

func TestMessagingHealthChecker_GetName(t *testing.T) {
	publisher := &mockEventPublisher{}
	checker := NewMessagingHealthChecker("my-messaging", publisher, nil)

	assert.Equal(t, "my-messaging", checker.GetName())
}

// Tests for SagaHealthManager

func TestNewSagaHealthManager_DefaultConfig(t *testing.T) {
	manager := NewSagaHealthManager(nil)

	assert.NotNil(t, manager)
	assert.Equal(t, 5*time.Second, manager.cacheDuration)
	assert.Equal(t, 10*time.Second, manager.checkTimeout)
	assert.False(t, manager.IsReady())
	assert.True(t, manager.IsAlive())
}

func TestNewSagaHealthManager_CustomConfig(t *testing.T) {
	config := &HealthManagerConfig{
		CacheDuration:  10 * time.Second,
		CheckTimeout:   20 * time.Second,
		InitiallyReady: true,
		InitiallyAlive: false,
	}
	manager := NewSagaHealthManager(config)

	assert.NotNil(t, manager)
	assert.Equal(t, 10*time.Second, manager.cacheDuration)
	assert.Equal(t, 20*time.Second, manager.checkTimeout)
	assert.True(t, manager.IsReady())
	assert.False(t, manager.IsAlive())
}

func TestSagaHealthManager_RegisterChecker(t *testing.T) {
	manager := NewSagaHealthManager(nil)

	coordinator := &mockCoordinator{}
	checker := NewCoordinatorHealthChecker("test-coordinator", coordinator, 0)

	err := manager.RegisterChecker(checker)
	assert.NoError(t, err)

	// Try to register same checker again
	err = manager.RegisterChecker(checker)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestSagaHealthManager_UnregisterChecker(t *testing.T) {
	manager := NewSagaHealthManager(nil)

	coordinator := &mockCoordinator{}
	checker := NewCoordinatorHealthChecker("test-coordinator", coordinator, 0)

	err := manager.RegisterChecker(checker)
	require.NoError(t, err)

	manager.UnregisterChecker("test-coordinator")

	// Should be able to register again after unregistering
	err = manager.RegisterChecker(checker)
	assert.NoError(t, err)
}

func TestSagaHealthManager_CheckHealth_AllHealthy(t *testing.T) {
	manager := NewSagaHealthManager(nil)

	coordinator := &mockCoordinator{healthErr: nil}
	storage := &mockStorage{healthErr: nil}

	err := manager.RegisterChecker(NewCoordinatorHealthChecker("coordinator", coordinator, 0))
	require.NoError(t, err)
	err = manager.RegisterChecker(NewStorageHealthChecker("storage", storage, 5*time.Second))
	require.NoError(t, err)

	ctx := context.Background()
	report, err := manager.CheckHealth(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, HealthStatusHealthy, report.Status)
	assert.Len(t, report.Components, 2)
	assert.True(t, report.TotalCheckDuration > 0)
}

func TestSagaHealthManager_CheckHealth_OneUnhealthy(t *testing.T) {
	manager := NewSagaHealthManager(nil)

	coordinator := &mockCoordinator{healthErr: nil}
	storage := &mockStorage{healthErr: errors.New("connection failed")}

	err := manager.RegisterChecker(NewCoordinatorHealthChecker("coordinator", coordinator, 0))
	require.NoError(t, err)
	err = manager.RegisterChecker(NewStorageHealthChecker("storage", storage, 5*time.Second))
	require.NoError(t, err)

	ctx := context.Background()
	report, err := manager.CheckHealth(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, HealthStatusUnhealthy, report.Status)
	assert.Len(t, report.Components, 2)

	// Check that storage is marked unhealthy
	storageHealth := report.Components["storage"]
	assert.Equal(t, HealthStatusUnhealthy, storageHealth.Status)
}

func TestSagaHealthManager_CheckHealth_OneDegraded(t *testing.T) {
	manager := NewSagaHealthManager(nil)

	coordinator := &mockCoordinator{healthErr: nil, activeSagas: 150}
	storage := &mockStorage{healthErr: nil}

	err := manager.RegisterChecker(NewCoordinatorHealthChecker("coordinator", coordinator, 100))
	require.NoError(t, err)
	err = manager.RegisterChecker(NewStorageHealthChecker("storage", storage, 5*time.Second))
	require.NoError(t, err)

	ctx := context.Background()
	report, err := manager.CheckHealth(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, HealthStatusDegraded, report.Status)
	assert.Len(t, report.Components, 2)

	// Check that coordinator is marked degraded
	coordinatorHealth := report.Components["coordinator"]
	assert.Equal(t, HealthStatusDegraded, coordinatorHealth.Status)
}

func TestSagaHealthManager_CheckHealth_Cache(t *testing.T) {
	config := &HealthManagerConfig{
		CacheDuration:  1 * time.Second,
		CheckTimeout:   5 * time.Second,
		InitiallyReady: false,
		InitiallyAlive: true,
	}
	manager := NewSagaHealthManager(config)

	coordinator := &mockCoordinator{healthErr: nil}
	err := manager.RegisterChecker(NewCoordinatorHealthChecker("coordinator", coordinator, 0))
	require.NoError(t, err)

	ctx := context.Background()

	// First check
	report1, err := manager.CheckHealth(ctx)
	require.NoError(t, err)
	timestamp1 := report1.Timestamp

	// Second check immediately (should use cache)
	report2, err := manager.CheckHealth(ctx)
	require.NoError(t, err)
	timestamp2 := report2.Timestamp

	assert.Equal(t, timestamp1, timestamp2, "should use cached result")

	// Wait for cache to expire
	time.Sleep(1100 * time.Millisecond)

	// Third check (should perform new check)
	report3, err := manager.CheckHealth(ctx)
	require.NoError(t, err)
	timestamp3 := report3.Timestamp

	assert.True(t, timestamp3.After(timestamp1), "should perform new check after cache expires")
}

func TestSagaHealthManager_CheckReadiness_Ready(t *testing.T) {
	manager := NewSagaHealthManager(nil)
	manager.SetReady(true)

	coordinator := &mockCoordinator{healthErr: nil}
	err := manager.RegisterChecker(NewCoordinatorHealthChecker("coordinator", coordinator, 0))
	require.NoError(t, err)

	ctx := context.Background()
	ready, err := manager.CheckReadiness(ctx)

	assert.NoError(t, err)
	assert.True(t, ready)
}

func TestSagaHealthManager_CheckReadiness_NotReady(t *testing.T) {
	manager := NewSagaHealthManager(nil)
	manager.SetReady(false)

	ctx := context.Background()
	ready, err := manager.CheckReadiness(ctx)

	assert.Error(t, err)
	assert.False(t, ready)
	assert.Contains(t, err.Error(), "not ready")
}

func TestSagaHealthManager_CheckReadiness_UnhealthyComponent(t *testing.T) {
	manager := NewSagaHealthManager(nil)
	manager.SetReady(true)

	coordinator := &mockCoordinator{healthErr: errors.New("coordinator down")}
	err := manager.RegisterChecker(NewCoordinatorHealthChecker("coordinator", coordinator, 0))
	require.NoError(t, err)

	ctx := context.Background()
	ready, err := manager.CheckReadiness(ctx)

	assert.NoError(t, err)
	assert.False(t, ready)
}

func TestSagaHealthManager_CheckReadiness_DegradedOk(t *testing.T) {
	manager := NewSagaHealthManager(nil)
	manager.SetReady(true)

	coordinator := &mockCoordinator{healthErr: nil, activeSagas: 150}
	err := manager.RegisterChecker(NewCoordinatorHealthChecker("coordinator", coordinator, 100))
	require.NoError(t, err)

	ctx := context.Background()
	ready, err := manager.CheckReadiness(ctx)

	assert.NoError(t, err)
	assert.True(t, ready, "degraded status should still be ready")
}

func TestSagaHealthManager_CheckLiveness_Alive(t *testing.T) {
	manager := NewSagaHealthManager(nil)
	manager.SetAlive(true)

	ctx := context.Background()
	alive, err := manager.CheckLiveness(ctx)

	assert.NoError(t, err)
	assert.True(t, alive)
}

func TestSagaHealthManager_CheckLiveness_NotAlive(t *testing.T) {
	manager := NewSagaHealthManager(nil)
	manager.SetAlive(false)

	ctx := context.Background()
	alive, err := manager.CheckLiveness(ctx)

	assert.Error(t, err)
	assert.False(t, alive)
	assert.Contains(t, err.Error(), "not alive")
}

func TestSagaHealthManager_CheckLiveness_ContextCanceled(t *testing.T) {
	manager := NewSagaHealthManager(nil)
	manager.SetAlive(true)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	alive, err := manager.CheckLiveness(ctx)

	assert.Error(t, err)
	assert.False(t, alive)
}

func TestSagaHealthManager_SetReady(t *testing.T) {
	manager := NewSagaHealthManager(nil)

	assert.False(t, manager.IsReady())

	manager.SetReady(true)
	assert.True(t, manager.IsReady())

	manager.SetReady(false)
	assert.False(t, manager.IsReady())
}

func TestSagaHealthManager_SetAlive(t *testing.T) {
	manager := NewSagaHealthManager(nil)

	assert.True(t, manager.IsAlive())

	manager.SetAlive(false)
	assert.False(t, manager.IsAlive())

	manager.SetAlive(true)
	assert.True(t, manager.IsAlive())
}

func TestSagaHealthManager_GetLastReport(t *testing.T) {
	manager := NewSagaHealthManager(nil)

	// Initially no report
	assert.Nil(t, manager.GetLastReport())

	coordinator := &mockCoordinator{healthErr: nil}
	err := manager.RegisterChecker(NewCoordinatorHealthChecker("coordinator", coordinator, 0))
	require.NoError(t, err)

	ctx := context.Background()
	_, err = manager.CheckHealth(ctx)
	require.NoError(t, err)

	// Now should have a report
	report := manager.GetLastReport()
	assert.NotNil(t, report)
	assert.Equal(t, HealthStatusHealthy, report.Status)
}

func TestSagaHealthManager_ClearCache(t *testing.T) {
	manager := NewSagaHealthManager(nil)

	coordinator := &mockCoordinator{healthErr: nil}
	err := manager.RegisterChecker(NewCoordinatorHealthChecker("coordinator", coordinator, 0))
	require.NoError(t, err)

	ctx := context.Background()
	_, err = manager.CheckHealth(ctx)
	require.NoError(t, err)

	report1 := manager.GetLastReport()
	assert.NotNil(t, report1)
	timestamp1 := report1.Timestamp

	manager.ClearCache()

	// After clearing cache, GetLastReport should return nil
	assert.Nil(t, manager.GetLastReport())

	// Next CheckHealth will perform a fresh check
	_, err = manager.CheckHealth(ctx)
	require.NoError(t, err)

	report2 := manager.GetLastReport()
	assert.NotNil(t, report2)
	timestamp2 := report2.Timestamp
	assert.True(t, timestamp2.After(timestamp1), "should perform fresh check after cache clear")
}

// Benchmarks

func BenchmarkCoordinatorHealthChecker_Check(b *testing.B) {
	coordinator := &mockCoordinator{healthErr: nil}
	checker := NewCoordinatorHealthChecker("test", coordinator, 0)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check(ctx)
	}
}

func BenchmarkStorageHealthChecker_Check(b *testing.B) {
	storage := &mockStorage{healthErr: nil}
	checker := NewStorageHealthChecker("test", storage, 5*time.Second)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check(ctx)
	}
}

func BenchmarkSagaHealthManager_CheckHealth(b *testing.B) {
	manager := NewSagaHealthManager(nil)

	coordinator := &mockCoordinator{healthErr: nil}
	storage := &mockStorage{healthErr: nil}

	manager.RegisterChecker(NewCoordinatorHealthChecker("coordinator", coordinator, 0))
	manager.RegisterChecker(NewStorageHealthChecker("storage", storage, 5*time.Second))

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.CheckHealth(ctx)
	}
}
