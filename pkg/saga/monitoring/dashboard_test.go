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
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// dashboardMockCoordinator is a mock implementation of saga.SagaCoordinator for testing.
type dashboardMockCoordinator struct {
	mu                sync.RWMutex
	started           int
	cancelled         int
	healthCheckErr    error
	getInstanceErr    error
	getActiveSagasErr error
	startSagaErr      error
	cancelSagaErr     error
	metrics           *saga.CoordinatorMetrics
}

func newMockCoordinator() *dashboardMockCoordinator {
	return &dashboardMockCoordinator{
		metrics: &saga.CoordinatorMetrics{
			TotalSagas:     10,
			ActiveSagas:    5,
			CompletedSagas: 3,
			FailedSagas:    2,
		},
	}
}

func (m *dashboardMockCoordinator) StartSaga(ctx context.Context, definition saga.SagaDefinition, initialData interface{}) (saga.SagaInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started++
	if m.startSagaErr != nil {
		return nil, m.startSagaErr
	}
	return &mockSagaInstance{id: "test-saga-1"}, nil
}

func (m *dashboardMockCoordinator) GetSagaInstance(sagaID string) (saga.SagaInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getInstanceErr != nil {
		return nil, m.getInstanceErr
	}
	return &mockSagaInstance{id: sagaID}, nil
}

func (m *dashboardMockCoordinator) CancelSaga(ctx context.Context, sagaID string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelled++
	return m.cancelSagaErr
}

func (m *dashboardMockCoordinator) GetActiveSagas(filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getActiveSagasErr != nil {
		return nil, m.getActiveSagasErr
	}
	return []saga.SagaInstance{
		&mockSagaInstance{id: "saga-1"},
		&mockSagaInstance{id: "saga-2"},
	}, nil
}

func (m *dashboardMockCoordinator) GetMetrics() *saga.CoordinatorMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics
}

func (m *dashboardMockCoordinator) HealthCheck(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.healthCheckErr
}

func (m *dashboardMockCoordinator) Close() error {
	return nil
}

// mockSagaInstance is a mock implementation of saga.SagaInstance.
type mockSagaInstance struct {
	id string
}

func (m *mockSagaInstance) GetID() string {
	return m.id
}

func (m *mockSagaInstance) GetDefinitionID() string {
	return "test-definition"
}

func (m *mockSagaInstance) GetState() saga.SagaState {
	return saga.StateRunning
}

func (m *mockSagaInstance) GetCurrentStep() int {
	return 0
}

func (m *mockSagaInstance) GetStartTime() time.Time {
	return time.Now()
}

func (m *mockSagaInstance) GetEndTime() time.Time {
	return time.Time{}
}

func (m *mockSagaInstance) GetResult() interface{} {
	return nil
}

func (m *mockSagaInstance) GetError() *saga.SagaError {
	return nil
}

func (m *mockSagaInstance) GetTotalSteps() int {
	return 3
}

func (m *mockSagaInstance) GetCompletedSteps() int {
	return 0
}

func (m *mockSagaInstance) GetCreatedAt() time.Time {
	return time.Now()
}

func (m *mockSagaInstance) GetUpdatedAt() time.Time {
	return time.Now()
}

func (m *mockSagaInstance) GetTimeout() time.Duration {
	return 30 * time.Second
}

func (m *mockSagaInstance) GetMetadata() map[string]interface{} {
	return nil
}

func (m *mockSagaInstance) GetTraceID() string {
	return "test-trace-id"
}

func (m *mockSagaInstance) IsTerminal() bool {
	return false
}

func (m *mockSagaInstance) IsActive() bool {
	return true
}

// TestDashboardConfigValidate tests the validation of dashboard configuration.
func TestDashboardConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *DashboardConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &DashboardConfig{
				Coordinator:  newMockCoordinator(),
				ServerConfig: DefaultServerConfig(),
			},
			wantErr: false,
		},
		{
			name: "missing coordinator",
			config: &DashboardConfig{
				ServerConfig: DefaultServerConfig(),
			},
			wantErr: true,
			errMsg:  "coordinator is required",
		},
		{
			name: "invalid server config",
			config: &DashboardConfig{
				Coordinator: newMockCoordinator(),
				ServerConfig: &ServerConfig{
					Address: "",
					Port:    "",
				},
			},
			wantErr: true,
			errMsg:  "invalid server configuration",
		},
		{
			name: "nil server config is allowed",
			config: &DashboardConfig{
				Coordinator: newMockCoordinator(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DashboardConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg && !contains(err.Error(), tt.errMsg) {
					t.Errorf("DashboardConfig.Validate() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestDefaultDashboardConfig tests the default dashboard configuration.
func TestDefaultDashboardConfig(t *testing.T) {
	config := DefaultDashboardConfig()

	if config == nil {
		t.Fatal("DefaultDashboardConfig() returned nil")
	}

	if config.ServerConfig == nil {
		t.Error("DefaultDashboardConfig() ServerConfig is nil")
	}

	if config.EnableAutoStart {
		t.Error("DefaultDashboardConfig() EnableAutoStart should be false by default")
	}

	if !config.EnableContextPropagation {
		t.Error("DefaultDashboardConfig() EnableContextPropagation should be true by default")
	}
}

// TestNewSagaDashboard tests the creation of a new dashboard without starting the server.
// Server functionality is tested separately in server_test.go (issue #674).
func TestNewSagaDashboard(t *testing.T) {
	t.Run("missing coordinator returns error", func(t *testing.T) {
		config := &DashboardConfig{
			ServerConfig: DefaultServerConfig(),
		}

		_, err := NewSagaDashboard(config)
		if err == nil {
			t.Error("NewSagaDashboard() should return error when coordinator is missing")
		}
		if err != nil && !contains(err.Error(), "invalid dashboard configuration") {
			t.Errorf("NewSagaDashboard() error should mention invalid dashboard configuration, got: %v", err)
		}
	})
}

// Note: Server start/stop/lifecycle tests are skipped here as they conflict with
// existing server_test.go tests. The web server framework is thoroughly tested in
// server_test.go (issue #674). This test file focuses on dashboard configuration
// and structure validation.

// mockMetricsCollector is a mock implementation of MetricsCollector.
type mockMetricsCollector struct {
	mu        sync.Mutex
	started   int64
	completed int64
	failed    int64
}

func (m *mockMetricsCollector) RecordSagaStarted(sagaID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started++
}

func (m *mockMetricsCollector) RecordSagaCompleted(sagaID string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completed++
}

func (m *mockMetricsCollector) RecordSagaFailed(sagaID string, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failed++
}

func (m *mockMetricsCollector) GetActiveSagasCount() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started - m.completed - m.failed
}

func (m *mockMetricsCollector) GetMetrics() *Metrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &Metrics{
		SagasStarted:   m.started,
		SagasCompleted: m.completed,
		SagasFailed:    m.failed,
		ActiveSagas:    m.started - m.completed - m.failed,
		Timestamp:      time.Now(),
	}
}

func (m *mockMetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = 0
	m.completed = 0
	m.failed = 0
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
