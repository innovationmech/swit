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

package messaging

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultShutdownConfig(t *testing.T) {
	config := DefaultShutdownConfig()

	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 60*time.Second, config.ForceTimeout)
	assert.Equal(t, 20*time.Second, config.DrainTimeout)
	assert.Equal(t, 5*time.Second, config.ReportInterval)
	assert.Equal(t, 1000, config.MaxInflightMessages)
}

func TestShutdownPhase_String(t *testing.T) {
	tests := []struct {
		phase    ShutdownPhase
		expected string
	}{
		{ShutdownPhaseIdle, "idle"},
		{ShutdownPhaseInitiated, "initiated"},
		{ShutdownPhaseStoppingNewMessages, "stopping_new_messages"},
		{ShutdownPhaseDrainingInflight, "draining_inflight"},
		{ShutdownPhaseClosingConnections, "closing_connections"},
		{ShutdownPhaseCleaningHandlers, "cleaning_handlers"},
		{ShutdownPhaseReleasingResources, "releasing_resources"},
		{ShutdownPhaseCompleted, "completed"},
		{ShutdownPhaseForcedTermination, "forced_termination"},
		{ShutdownPhase(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.phase.String())
		})
	}
}

func TestNewGracefulShutdownManager(t *testing.T) {
	coordinator := NewMessagingCoordinator()

	// Test with nil config
	manager := NewGracefulShutdownManager(coordinator, nil)
	assert.NotNil(t, manager)
	assert.Equal(t, ShutdownPhaseIdle, manager.status.Phase)
	assert.NotNil(t, manager.config)

	// Test with custom config
	customConfig := &ShutdownConfig{
		Timeout:             45 * time.Second,
		ForceTimeout:        90 * time.Second,
		DrainTimeout:        30 * time.Second,
		ReportInterval:      10 * time.Second,
		MaxInflightMessages: 2000,
	}

	manager = NewGracefulShutdownManager(coordinator, customConfig)
	assert.NotNil(t, manager)
	assert.Equal(t, customConfig, manager.config)
}

func TestGracefulShutdownManager_GetShutdownStatus(t *testing.T) {
	coordinator := NewMessagingCoordinator()
	manager := NewGracefulShutdownManager(coordinator, nil)

	status := manager.GetShutdownStatus()
	assert.NotNil(t, status)
	assert.Equal(t, ShutdownPhaseIdle, status.Phase)
	assert.NotNil(t, status.BrokerStatuses)
	assert.NotNil(t, status.HandlerStatuses)
	assert.NotNil(t, status.CompletedSteps)
	assert.NotNil(t, status.PendingSteps)
}

func TestGracefulShutdownManager_InitiateGracefulShutdown(t *testing.T) {
	coordinator := NewMessagingCoordinator()
	config := &ShutdownConfig{
		Timeout:             10 * time.Second,
		ForceTimeout:        20 * time.Second,
		DrainTimeout:        5 * time.Second,
		ReportInterval:      time.Second,
		MaxInflightMessages: 100,
	}

	manager := NewGracefulShutdownManager(coordinator, config)
	ctx := context.Background()

	// Test normal initiation
	err := manager.InitiateGracefulShutdown(ctx)
	assert.NoError(t, err)

	status := manager.GetShutdownStatus()
	assert.Equal(t, ShutdownPhaseInitiated, status.Phase)
	assert.NotZero(t, status.StartTime)
	assert.Greater(t, len(status.PendingSteps), 0)

	// Test double initiation (should fail)
	err = manager.InitiateGracefulShutdown(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "shutdown already in progress")
}

func TestGracefulShutdownManager_WaitForCompletion(t *testing.T) {
	coordinator := NewMessagingCoordinator()
	config := &ShutdownConfig{
		Timeout:             2 * time.Second,
		ForceTimeout:        3 * time.Second,
		DrainTimeout:        time.Second,
		ReportInterval:      500 * time.Millisecond,
		MaxInflightMessages: 10,
	}

	manager := NewGracefulShutdownManager(coordinator, config)
	ctx := context.Background()

	// Initiate shutdown
	err := manager.InitiateGracefulShutdown(ctx)
	require.NoError(t, err)

	// Wait for completion (should complete quickly with no brokers/handlers)
	err = manager.WaitForCompletion()
	assert.NoError(t, err)

	status := manager.GetShutdownStatus()
	assert.Equal(t, ShutdownPhaseCompleted, status.Phase)
}

func TestGracefulShutdownManager_ForceShutdown(t *testing.T) {
	coordinator := NewMessagingCoordinator()
	config := &ShutdownConfig{
		Timeout:             100 * time.Millisecond,
		ForceTimeout:        50 * time.Millisecond, // Force timeout shorter than drain timeout to trigger force shutdown
		DrainTimeout:        100 * time.Millisecond,
		ReportInterval:      10 * time.Millisecond,
		MaxInflightMessages: 10,
	}

	manager := NewGracefulShutdownManager(coordinator, config)

	ctx := context.Background()

	err := manager.InitiateGracefulShutdown(ctx)
	require.NoError(t, err)

	// Wait for completion - should be forced due to timeout
	err = manager.WaitForCompletion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forcefully terminated")
}

func TestGracefulShutdownManager_InflightMessageTracking(t *testing.T) {
	coordinator := NewMessagingCoordinator()
	manager := NewGracefulShutdownManager(coordinator, nil)

	// Test initial count
	assert.Equal(t, int64(0), manager.getInflightMessageCount())

	// Test increment
	manager.IncrementInflightMessages()
	assert.Equal(t, int64(1), manager.getInflightMessageCount())

	manager.IncrementInflightMessages()
	assert.Equal(t, int64(2), manager.getInflightMessageCount())

	// Test decrement
	manager.DecrementInflightMessages()
	assert.Equal(t, int64(1), manager.getInflightMessageCount())

	manager.DecrementInflightMessages()
	assert.Equal(t, int64(0), manager.getInflightMessageCount())

	// Test decrement below zero (should not go negative)
	manager.DecrementInflightMessages()
	assert.Equal(t, int64(0), manager.getInflightMessageCount())
}

func TestGracefulShutdownManager_MarkStepCompleted(t *testing.T) {
	coordinator := NewMessagingCoordinator()
	manager := NewGracefulShutdownManager(coordinator, nil)

	// Initialize pending steps
	manager.status.PendingSteps = []string{"step1", "step2", "step3"}

	// Mark step as completed
	manager.markStepCompleted("step2")

	status := manager.GetShutdownStatus()
	assert.Contains(t, status.CompletedSteps, "step2")
	assert.NotContains(t, status.PendingSteps, "step2")
	assert.Contains(t, status.PendingSteps, "step1")
	assert.Contains(t, status.PendingSteps, "step3")
}

func TestGracefulShutdownManager_WithCoordinatorAndBrokers(t *testing.T) {
	// Create a coordinator with mock brokers
	coordinator := NewMessagingCoordinator()

	// Create mock broker
	mockBroker := &mockBroker{connected: true}
	err := coordinator.RegisterBroker("test-broker", mockBroker)
	require.NoError(t, err)

	// Start coordinator
	ctx := context.Background()
	err = coordinator.Start(ctx)
	require.NoError(t, err)

	// Create graceful shutdown manager
	config := &ShutdownConfig{
		Timeout:             5 * time.Second,
		ForceTimeout:        10 * time.Second,
		DrainTimeout:        2 * time.Second,
		ReportInterval:      time.Second,
		MaxInflightMessages: 100,
	}

	manager := NewGracefulShutdownManager(coordinator, config)

	// Initiate graceful shutdown
	err = manager.InitiateGracefulShutdown(ctx)
	require.NoError(t, err)

	// Wait for completion
	err = manager.WaitForCompletion()
	assert.NoError(t, err)

	// Verify final status
	status := manager.GetShutdownStatus()
	assert.Equal(t, ShutdownPhaseCompleted, status.Phase)
	assert.Len(t, status.BrokerStatuses, 1)
	assert.Contains(t, status.BrokerStatuses, "test-broker")
	assert.True(t, status.BrokerStatuses["test-broker"].ConnectionsClosed)
}

// mockBroker for testing
type mockBroker struct {
	connected bool
}

func (m *mockBroker) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *mockBroker) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *mockBroker) Close() error {
	m.connected = false
	return nil
}

func (m *mockBroker) IsConnected() bool {
	return m.connected
}

func (m *mockBroker) CreatePublisher(config PublisherConfig) (EventPublisher, error) {
	return &mockPublisher{}, nil
}

func (m *mockBroker) CreateSubscriber(config SubscriberConfig) (EventSubscriber, error) {
	return &mockSubscriber{}, nil
}

func (m *mockBroker) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		Status:  HealthStatusHealthy,
		Message: "Mock broker is healthy",
	}
	return status, nil
}

func (m *mockBroker) GetMetrics() *BrokerMetrics {
	return &BrokerMetrics{}
}

func (m *mockBroker) GetCapabilities() *BrokerCapabilities {
	return &BrokerCapabilities{}
}

// mockPublisher for testing
type mockPublisher struct{}

func (m *mockPublisher) Publish(ctx context.Context, message *Message) error         { return nil }
func (m *mockPublisher) PublishBatch(ctx context.Context, messages []*Message) error { return nil }
func (m *mockPublisher) PublishWithConfirm(ctx context.Context, message *Message) (*PublishConfirmation, error) {
	return &PublishConfirmation{}, nil
}
func (m *mockPublisher) PublishAsync(ctx context.Context, message *Message, callback PublishCallback) error {
	return nil
}
func (m *mockPublisher) BeginTransaction(ctx context.Context) (Transaction, error) {
	// Use the existing mockTransaction from interfaces_test.go
	return &mockTransaction{id: "test-tx"}, nil
}
func (m *mockPublisher) Flush(ctx context.Context) error { return nil }
func (m *mockPublisher) Close() error                    { return nil }
func (m *mockPublisher) GetMetrics() *PublisherMetrics   { return &PublisherMetrics{} }

// mockSubscriber for testing
type mockSubscriber struct{}

func (m *mockSubscriber) Subscribe(ctx context.Context, handler MessageHandler) error { return nil }
func (m *mockSubscriber) SubscribeWithMiddleware(ctx context.Context, handler MessageHandler, middleware ...Middleware) error {
	return nil
}
func (m *mockSubscriber) Unsubscribe(ctx context.Context) error                 { return nil }
func (m *mockSubscriber) Pause(ctx context.Context) error                       { return nil }
func (m *mockSubscriber) Resume(ctx context.Context) error                      { return nil }
func (m *mockSubscriber) Seek(ctx context.Context, position SeekPosition) error { return nil }
func (m *mockSubscriber) GetLag(ctx context.Context) (int64, error)             { return 0, nil }
func (m *mockSubscriber) Close() error                                          { return nil }
func (m *mockSubscriber) GetMetrics() *SubscriberMetrics                        { return &SubscriberMetrics{} }
