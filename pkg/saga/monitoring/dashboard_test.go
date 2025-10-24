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

// TestNewSagaDashboard_WithDefaultConfig tests dashboard creation with default config.
func TestNewSagaDashboard_WithDefaultConfig(t *testing.T) {
	coordinator := newMockCoordinator()
	config := &DashboardConfig{
		Coordinator: coordinator,
	}

	dashboard, err := NewSagaDashboard(config)
	if err != nil {
		t.Fatalf("NewSagaDashboard() error = %v", err)
	}

	if dashboard == nil {
		t.Fatal("NewSagaDashboard() returned nil dashboard")
	}

	// Verify components are initialized
	if dashboard.coordinator == nil {
		t.Error("Dashboard coordinator is nil")
	}
	if dashboard.metricsCollector == nil {
		t.Error("Dashboard metricsCollector is nil")
	}
	if dashboard.healthManager == nil {
		t.Error("Dashboard healthManager is nil")
	}
	if dashboard.server == nil {
		t.Error("Dashboard server is nil")
	}
}

// TestNewSagaDashboard_WithCustomComponents tests dashboard creation with custom components.
func TestNewSagaDashboard_WithCustomComponents(t *testing.T) {
	coordinator := newMockCoordinator()
	mockCollector := &mockMetricsCollector{}
	healthManager := NewSagaHealthManager(nil)

	config := &DashboardConfig{
		Coordinator:      coordinator,
		MetricsCollector: mockCollector,
		HealthManager:    healthManager,
	}

	dashboard, err := NewSagaDashboard(config)
	if err != nil {
		t.Fatalf("NewSagaDashboard() error = %v", err)
	}

	// Verify custom components are used
	if dashboard.metricsCollector != mockCollector {
		t.Error("Dashboard should use provided metrics collector")
	}
	if dashboard.healthManager != healthManager {
		t.Error("Dashboard should use provided health manager")
	}
}

// TestDashboardLifecycle tests the full lifecycle of dashboard.
func TestDashboardLifecycle(t *testing.T) {
	coordinator := newMockCoordinator()
	config := &DashboardConfig{
		Coordinator: coordinator,
		ServerConfig: &ServerConfig{
			Address: "127.0.0.1",
			Port:    "0", // Use random port
		},
	}

	dashboard, err := NewSagaDashboard(config)
	if err != nil {
		t.Fatalf("NewSagaDashboard() error = %v", err)
	}

	ctx := context.Background()

	// Test Start
	if err := dashboard.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Verify dashboard is running
	if !dashboard.IsRunning() {
		t.Error("Dashboard should be running after Start()")
	}

	// Test getting components
	if dashboard.GetCoordinator() == nil {
		t.Error("GetCoordinator() should not return nil")
	}
	if dashboard.GetMetricsCollector() == nil {
		t.Error("GetMetricsCollector() should not return nil")
	}
	if dashboard.GetHealthManager() == nil {
		t.Error("GetHealthManager() should not return nil")
	}
	if dashboard.GetServer() == nil {
		t.Error("GetServer() should not return nil")
	}
	if dashboard.GetConfig() == nil {
		t.Error("GetConfig() should not return nil")
	}
	if dashboard.GetAddress() == "" {
		t.Error("GetAddress() should return a valid address after Start()")
	}

	// Test GetContext
	dashboardCtx := dashboard.GetContext()
	if dashboardCtx == nil {
		t.Error("GetContext() should not return nil")
	}

	// Test Stop
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dashboard.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Verify dashboard is not running
	if dashboard.IsRunning() {
		t.Error("Dashboard should not be running after Stop()")
	}

	// Test Stop on already stopped dashboard (should be no-op)
	if err := dashboard.Stop(ctx); err != nil {
		t.Errorf("Stop() on stopped dashboard should not error, got: %v", err)
	}
}

// TestDashboardStartWhenAlreadyRunning tests starting dashboard when already running.
func TestDashboardStartWhenAlreadyRunning(t *testing.T) {
	coordinator := newMockCoordinator()
	config := &DashboardConfig{
		Coordinator: coordinator,
		ServerConfig: &ServerConfig{
			Address: "127.0.0.1",
			Port:    "0",
		},
	}

	dashboard, err := NewSagaDashboard(config)
	if err != nil {
		t.Fatalf("NewSagaDashboard() error = %v", err)
	}

	ctx := context.Background()
	if err := dashboard.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = dashboard.Stop(stopCtx)
	}()

	// Try to start again
	err = dashboard.Start(ctx)
	if err == nil {
		t.Error("Start() should return error when dashboard is already running")
	}
	if err != nil && !contains(err.Error(), "already running") {
		t.Errorf("Start() error should mention 'already running', got: %v", err)
	}
}

// TestDashboardShutdown tests complete shutdown of dashboard.
func TestDashboardShutdown(t *testing.T) {
	coordinator := newMockCoordinator()
	config := &DashboardConfig{
		Coordinator: coordinator,
		ServerConfig: &ServerConfig{
			Address: "127.0.0.1",
			Port:    "0",
		},
	}

	dashboard, err := NewSagaDashboard(config)
	if err != nil {
		t.Fatalf("NewSagaDashboard() error = %v", err)
	}

	ctx := context.Background()
	if err := dashboard.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Test Shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dashboard.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	// Verify dashboard is not running
	if dashboard.IsRunning() {
		t.Error("Dashboard should not be running after Shutdown()")
	}

	// Test Shutdown on already stopped dashboard (should be no-op)
	if err := dashboard.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown() on stopped dashboard should not error, got: %v", err)
	}
}

// TestDashboardContextPropagation tests context propagation in dashboard.
func TestDashboardContextPropagation(t *testing.T) {
	coordinator := newMockCoordinator()
	config := &DashboardConfig{
		Coordinator:              coordinator,
		EnableContextPropagation: true,
		ServerConfig: &ServerConfig{
			Address: "127.0.0.1",
			Port:    "0",
		},
	}

	dashboard, err := NewSagaDashboard(config)
	if err != nil {
		t.Fatalf("NewSagaDashboard() error = %v", err)
	}

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	if err := dashboard.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Cancel the provided context
	cancel()

	// Give it time to propagate
	time.Sleep(100 * time.Millisecond)

	// Verify dashboard context is also cancelled
	select {
	case <-dashboard.GetContext().Done():
		// Expected
	case <-time.After(1 * time.Second):
		t.Error("Dashboard context should be cancelled when parent context is cancelled")
	}

	// Clean up
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	_ = dashboard.Stop(stopCtx)
}

// TestDashboardAutoStart tests auto-start functionality.
func TestDashboardAutoStart(t *testing.T) {
	coordinator := newMockCoordinator()
	config := &DashboardConfig{
		Coordinator:     coordinator,
		EnableAutoStart: true,
		ServerConfig: &ServerConfig{
			Address: "127.0.0.1",
			Port:    "0",
		},
	}

	dashboard, err := NewSagaDashboard(config)
	if err != nil {
		t.Fatalf("NewSagaDashboard() error = %v", err)
	}

	// Dashboard should be running after creation
	if !dashboard.IsRunning() {
		t.Error("Dashboard should be running when EnableAutoStart is true")
	}

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = dashboard.Stop(ctx)
}

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
