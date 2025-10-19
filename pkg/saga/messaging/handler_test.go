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
	"errors"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestSagaEventType tests the SagaEventType helper functions
func TestSagaEventType(t *testing.T) {
	tests := []struct {
		name                   string
		eventType              SagaEventType
		expectedIsStepEvent    bool
		expectedIsCompensation bool
		expectedIsLifecycle    bool
		expectedIsRetry        bool
		expectedStringValue    string
	}{
		{
			name:                   "step started event",
			eventType:              EventTypeSagaStepStarted,
			expectedIsStepEvent:    true,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRetry:        false,
			expectedStringValue:    "saga.step.started",
		},
		{
			name:                   "step completed event",
			eventType:              EventTypeSagaStepCompleted,
			expectedIsStepEvent:    true,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRetry:        false,
			expectedStringValue:    "saga.step.completed",
		},
		{
			name:                   "step failed event",
			eventType:              EventTypeSagaStepFailed,
			expectedIsStepEvent:    true,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRetry:        false,
			expectedStringValue:    "saga.step.failed",
		},
		{
			name:                   "saga started event",
			eventType:              EventTypeSagaStarted,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    true,
			expectedIsRetry:        false,
			expectedStringValue:    "saga.started",
		},
		{
			name:                   "saga completed event",
			eventType:              EventTypeSagaCompleted,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    true,
			expectedIsRetry:        false,
			expectedStringValue:    "saga.completed",
		},
		{
			name:                   "saga failed event",
			eventType:              EventTypeSagaFailed,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    true,
			expectedIsRetry:        false,
			expectedStringValue:    "saga.failed",
		},
		{
			name:                   "saga cancelled event",
			eventType:              EventTypeSagaCancelled,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    true,
			expectedIsRetry:        false,
			expectedStringValue:    "saga.cancelled",
		},
		{
			name:                   "saga timed out event",
			eventType:              EventTypeSagaTimedOut,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    true,
			expectedIsRetry:        false,
			expectedStringValue:    "saga.timed_out",
		},
		{
			name:                   "compensation started event",
			eventType:              EventTypeCompensationStarted,
			expectedIsStepEvent:    false,
			expectedIsCompensation: true,
			expectedIsLifecycle:    false,
			expectedIsRetry:        false,
			expectedStringValue:    "compensation.started",
		},
		{
			name:                   "compensation step started event",
			eventType:              EventTypeCompensationStepStarted,
			expectedIsStepEvent:    false,
			expectedIsCompensation: true,
			expectedIsLifecycle:    false,
			expectedIsRetry:        false,
			expectedStringValue:    "compensation.step.started",
		},
		{
			name:                   "compensation step completed event",
			eventType:              EventTypeCompensationStepCompleted,
			expectedIsStepEvent:    false,
			expectedIsCompensation: true,
			expectedIsLifecycle:    false,
			expectedIsRetry:        false,
			expectedStringValue:    "compensation.step.completed",
		},
		{
			name:                   "compensation step failed event",
			eventType:              EventTypeCompensationStepFailed,
			expectedIsStepEvent:    false,
			expectedIsCompensation: true,
			expectedIsLifecycle:    false,
			expectedIsRetry:        false,
			expectedStringValue:    "compensation.step.failed",
		},
		{
			name:                   "compensation completed event",
			eventType:              EventTypeCompensationCompleted,
			expectedIsStepEvent:    false,
			expectedIsCompensation: true,
			expectedIsLifecycle:    false,
			expectedIsRetry:        false,
			expectedStringValue:    "compensation.completed",
		},
		{
			name:                   "compensation failed event",
			eventType:              EventTypeCompensationFailed,
			expectedIsStepEvent:    false,
			expectedIsCompensation: true,
			expectedIsLifecycle:    false,
			expectedIsRetry:        false,
			expectedStringValue:    "compensation.failed",
		},
		{
			name:                   "retry attempted event",
			eventType:              EventTypeRetryAttempted,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRetry:        true,
			expectedStringValue:    "retry.attempted",
		},
		{
			name:                   "retry exhausted event",
			eventType:              EventTypeRetryExhausted,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRetry:        true,
			expectedStringValue:    "retry.exhausted",
		},
		{
			name:                   "state changed event",
			eventType:              EventTypeStateChanged,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRetry:        false,
			expectedStringValue:    "state.changed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test String() - using the underlying type's string value
			if got := string(tt.eventType); got != tt.expectedStringValue {
				t.Errorf("String() = %v, want %v", got, tt.expectedStringValue)
			}

			// Test IsStepEvent()
			if got := IsStepEvent(tt.eventType); got != tt.expectedIsStepEvent {
				t.Errorf("IsStepEvent() = %v, want %v", got, tt.expectedIsStepEvent)
			}

			// Test IsCompensationEvent()
			if got := IsCompensationEvent(tt.eventType); got != tt.expectedIsCompensation {
				t.Errorf("IsCompensationEvent() = %v, want %v", got, tt.expectedIsCompensation)
			}

			// Test IsSagaLifecycleEvent()
			if got := IsSagaLifecycleEvent(tt.eventType); got != tt.expectedIsLifecycle {
				t.Errorf("IsSagaLifecycleEvent() = %v, want %v", got, tt.expectedIsLifecycle)
			}

			// Test IsRetryEvent()
			if got := IsRetryEvent(tt.eventType); got != tt.expectedIsRetry {
				t.Errorf("IsRetryEvent() = %v, want %v", got, tt.expectedIsRetry)
			}
		})
	}
}

// TestEventFailureAction tests the EventFailureAction methods
func TestEventFailureAction(t *testing.T) {
	tests := []struct {
		name           string
		action         EventFailureAction
		expectedString string
	}{
		{
			name:           "retry action",
			action:         FailureActionRetry,
			expectedString: "retry",
		},
		{
			name:           "dead letter action",
			action:         FailureActionDeadLetter,
			expectedString: "dead_letter",
		},
		{
			name:           "discard action",
			action:         FailureActionDiscard,
			expectedString: "discard",
		},
		{
			name:           "requeue action",
			action:         FailureActionRequeue,
			expectedString: "requeue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.action.String(); got != tt.expectedString {
				t.Errorf("String() = %v, want %v", got, tt.expectedString)
			}
		})
	}
}

// TestRoutingStrategy tests the RoutingStrategy methods
func TestRoutingStrategy(t *testing.T) {
	tests := []struct {
		name           string
		strategy       RoutingStrategy
		expectedString string
	}{
		{
			name:           "all strategy",
			strategy:       RoutingStrategyAll,
			expectedString: "all",
		},
		{
			name:           "priority strategy",
			strategy:       RoutingStrategyPriority,
			expectedString: "priority",
		},
		{
			name:           "round robin strategy",
			strategy:       RoutingStrategyRoundRobin,
			expectedString: "round_robin",
		},
		{
			name:           "random strategy",
			strategy:       RoutingStrategyRandom,
			expectedString: "random",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.strategy.String(); got != tt.expectedString {
				t.Errorf("String() = %v, want %v", got, tt.expectedString)
			}
		})
	}
}

// TestHandlerConfigValidate tests the HandlerConfig validation
func TestHandlerConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *HandlerConfig
		expectError bool
		expectedErr error
	}{
		{
			name: "valid config",
			config: &HandlerConfig{
				HandlerID:         "test-handler",
				HandlerName:       "Test Handler",
				Topics:            []string{"test-topic"},
				Concurrency:       5,
				BatchSize:         10,
				ProcessingTimeout: 30 * time.Second,
			},
			expectError: false,
		},
		{
			name: "empty handler ID",
			config: &HandlerConfig{
				HandlerName:       "Test Handler",
				Topics:            []string{"test-topic"},
				ProcessingTimeout: 30 * time.Second,
			},
			expectError: true,
			expectedErr: ErrInvalidHandlerID,
		},
		{
			name: "empty handler name",
			config: &HandlerConfig{
				HandlerID:         "test-handler",
				Topics:            []string{"test-topic"},
				ProcessingTimeout: 30 * time.Second,
			},
			expectError: true,
			expectedErr: ErrInvalidHandlerName,
		},
		{
			name: "no topics configured",
			config: &HandlerConfig{
				HandlerID:         "test-handler",
				HandlerName:       "Test Handler",
				ProcessingTimeout: 30 * time.Second,
			},
			expectError: true,
			expectedErr: ErrNoTopicsConfigured,
		},
		{
			name: "config with defaults applied",
			config: &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
				// Concurrency, BatchSize, and ProcessingTimeout not set
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.expectedErr != nil && err != tt.expectedErr {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Check that defaults are applied
				if tt.config.Concurrency <= 0 {
					t.Error("expected Concurrency to be set to default")
				}
				if tt.config.BatchSize <= 0 {
					t.Error("expected BatchSize to be set to default")
				}
				if tt.config.ProcessingTimeout <= 0 {
					t.Error("expected ProcessingTimeout to be set to default")
				}
			}
		})
	}
}

// TestFilterConfigShouldProcess tests the FilterConfig.ShouldProcess method
func TestFilterConfigShouldProcess(t *testing.T) {
	tests := []struct {
		name           string
		filter         *FilterConfig
		event          *saga.SagaEvent
		expectedResult bool
	}{
		{
			name:   "nil filter allows all events",
			filter: nil,
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-1",
			},
			expectedResult: true,
		},
		{
			name:   "empty filter allows all events",
			filter: &FilterConfig{},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-1",
			},
			expectedResult: true,
		},
		{
			name: "include event types - match",
			filter: &FilterConfig{
				IncludeEventTypes: []SagaEventType{
					EventTypeSagaStepCompleted,
					EventTypeSagaStepFailed,
				},
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-1",
			},
			expectedResult: true,
		},
		{
			name: "include event types - no match",
			filter: &FilterConfig{
				IncludeEventTypes: []SagaEventType{
					EventTypeSagaStepCompleted,
					EventTypeSagaStepFailed,
				},
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStarted,
				SagaID: "saga-1",
			},
			expectedResult: false,
		},
		{
			name: "exclude event types - match",
			filter: &FilterConfig{
				ExcludeEventTypes: []SagaEventType{
					EventTypeSagaStepFailed,
				},
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepFailed,
				SagaID: "saga-1",
			},
			expectedResult: false,
		},
		{
			name: "exclude event types - no match",
			filter: &FilterConfig{
				ExcludeEventTypes: []SagaEventType{
					EventTypeSagaStepFailed,
				},
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-1",
			},
			expectedResult: true,
		},
		{
			name: "include saga IDs - match",
			filter: &FilterConfig{
				IncludeSagaIDs: []string{"saga-1", "saga-2"},
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-1",
			},
			expectedResult: true,
		},
		{
			name: "include saga IDs - no match",
			filter: &FilterConfig{
				IncludeSagaIDs: []string{"saga-1", "saga-2"},
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-3",
			},
			expectedResult: false,
		},
		{
			name: "exclude saga IDs - match",
			filter: &FilterConfig{
				ExcludeSagaIDs: []string{"saga-1"},
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-1",
			},
			expectedResult: false,
		},
		{
			name: "exclude saga IDs - no match",
			filter: &FilterConfig{
				ExcludeSagaIDs: []string{"saga-1"},
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-2",
			},
			expectedResult: true,
		},
		{
			name: "custom filter returns true",
			filter: &FilterConfig{
				CustomFilter: func(event *saga.SagaEvent) bool {
					return event.SagaID == "saga-1"
				},
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-1",
			},
			expectedResult: true,
		},
		{
			name: "custom filter returns false",
			filter: &FilterConfig{
				CustomFilter: func(event *saga.SagaEvent) bool {
					return event.SagaID == "saga-1"
				},
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-2",
			},
			expectedResult: false,
		},
		{
			name: "complex filter - all conditions match",
			filter: &FilterConfig{
				IncludeEventTypes: []SagaEventType{EventTypeSagaStepCompleted},
				IncludeSagaIDs:    []string{"saga-1"},
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-1",
			},
			expectedResult: true,
		},
		{
			name: "complex filter - one condition fails",
			filter: &FilterConfig{
				IncludeEventTypes: []SagaEventType{EventTypeSagaStepCompleted},
				IncludeSagaIDs:    []string{"saga-1"},
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-2",
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.ShouldProcess(tt.event)
			if result != tt.expectedResult {
				t.Errorf("ShouldProcess() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

// TestEventHandlerContext tests the EventHandlerContext structure
func TestEventHandlerContext(t *testing.T) {
	ctx := &EventHandlerContext{
		MessageID:           "msg-123",
		Topic:               "saga-events",
		Partition:           0,
		Offset:              100,
		Timestamp:           time.Now(),
		Headers:             map[string]string{"content-type": "application/json"},
		RetryCount:          0,
		OriginalMessageData: []byte(`{"type":"saga.step.completed"}`),
		BrokerName:          "kafka",
		ConsumerGroup:       "saga-handlers",
		Metadata:            map[string]interface{}{"correlation_id": "corr-123"},
	}

	// Verify all fields are properly set
	if ctx.MessageID != "msg-123" {
		t.Errorf("MessageID = %v, want %v", ctx.MessageID, "msg-123")
	}
	if ctx.Topic != "saga-events" {
		t.Errorf("Topic = %v, want %v", ctx.Topic, "saga-events")
	}
	if ctx.Partition != 0 {
		t.Errorf("Partition = %v, want %v", ctx.Partition, 0)
	}
	if ctx.Offset != 100 {
		t.Errorf("Offset = %v, want %v", ctx.Offset, 100)
	}
	if ctx.RetryCount != 0 {
		t.Errorf("RetryCount = %v, want %v", ctx.RetryCount, 0)
	}
	if ctx.BrokerName != "kafka" {
		t.Errorf("BrokerName = %v, want %v", ctx.BrokerName, "kafka")
	}
	if ctx.ConsumerGroup != "saga-handlers" {
		t.Errorf("ConsumerGroup = %v, want %v", ctx.ConsumerGroup, "saga-handlers")
	}
	if len(ctx.Headers) == 0 {
		t.Error("Headers should not be empty")
	}
	if len(ctx.Metadata) == 0 {
		t.Error("Metadata should not be empty")
	}
}

// TestHandlerMetrics tests the HandlerMetrics structure
func TestHandlerMetrics(t *testing.T) {
	now := time.Now()
	metrics := &HandlerMetrics{
		HandlerID:               "test-handler",
		TotalEventsReceived:     100,
		TotalEventsProcessed:    95,
		TotalEventsFailed:       3,
		TotalEventsFiltered:     2,
		TotalEventsRetried:      3,
		TotalEventsDeadLettered: 0,
		AverageProcessingTime:   50 * time.Millisecond,
		MaxProcessingTime:       200 * time.Millisecond,
		MinProcessingTime:       10 * time.Millisecond,
		EventTypeMetrics: map[SagaEventType]*EventTypeMetrics{
			EventTypeSagaStepCompleted: {
				EventType:             EventTypeSagaStepCompleted,
				Count:                 50,
				FailureCount:          1,
				AverageProcessingTime: 45 * time.Millisecond,
			},
		},
		LastProcessedAt: now,
		LastErrorAt:     now.Add(-1 * time.Hour),
		LastError:       "",
		StartedAt:       now.Add(-24 * time.Hour),
		IsHealthy:       true,
	}

	// Verify metrics fields
	if metrics.HandlerID != "test-handler" {
		t.Errorf("HandlerID = %v, want %v", metrics.HandlerID, "test-handler")
	}
	if metrics.TotalEventsReceived != 100 {
		t.Errorf("TotalEventsReceived = %v, want %v", metrics.TotalEventsReceived, 100)
	}
	if metrics.TotalEventsProcessed != 95 {
		t.Errorf("TotalEventsProcessed = %v, want %v", metrics.TotalEventsProcessed, 95)
	}
	if metrics.TotalEventsFailed != 3 {
		t.Errorf("TotalEventsFailed = %v, want %v", metrics.TotalEventsFailed, 3)
	}
	if !metrics.IsHealthy {
		t.Error("IsHealthy should be true")
	}
	if len(metrics.EventTypeMetrics) == 0 {
		t.Error("EventTypeMetrics should not be empty")
	}
}

// TestEventTypeMetrics tests the EventTypeMetrics structure
func TestEventTypeMetrics(t *testing.T) {
	metrics := &EventTypeMetrics{
		EventType:             EventTypeSagaStepCompleted,
		Count:                 100,
		FailureCount:          5,
		AverageProcessingTime: 50 * time.Millisecond,
	}

	// Verify metrics fields
	if metrics.EventType != EventTypeSagaStepCompleted {
		t.Errorf("EventType = %v, want %v", metrics.EventType, EventTypeSagaStepCompleted)
	}
	if metrics.Count != 100 {
		t.Errorf("Count = %v, want %v", metrics.Count, 100)
	}
	if metrics.FailureCount != 5 {
		t.Errorf("FailureCount = %v, want %v", metrics.FailureCount, 5)
	}
	if metrics.AverageProcessingTime != 50*time.Millisecond {
		t.Errorf("AverageProcessingTime = %v, want %v", metrics.AverageProcessingTime, 50*time.Millisecond)
	}
}

// mockCoordinator is a mock implementation of saga.SagaCoordinator for testing.
type mockCoordinator struct {
	healthCheckErr error
}

func (m *mockCoordinator) StartSaga(ctx context.Context, definition saga.SagaDefinition, initialData interface{}) (saga.SagaInstance, error) {
	return nil, nil
}

func (m *mockCoordinator) GetSagaInstance(sagaID string) (saga.SagaInstance, error) {
	return nil, nil
}

func (m *mockCoordinator) CancelSaga(ctx context.Context, sagaID string, reason string) error {
	return nil
}

func (m *mockCoordinator) GetActiveSagas(filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	return nil, nil
}

func (m *mockCoordinator) GetMetrics() *saga.CoordinatorMetrics {
	return nil
}

func (m *mockCoordinator) HealthCheck(ctx context.Context) error {
	return m.healthCheckErr
}

func (m *mockCoordinator) Close() error {
	return nil
}

// mockEventPublisher is a mock implementation of saga.EventPublisher for testing.
type mockEventPublisher struct{}

func (m *mockEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	return nil
}

func (m *mockEventPublisher) Subscribe(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error) {
	return nil, nil
}

func (m *mockEventPublisher) Unsubscribe(subscription saga.EventSubscription) error {
	return nil
}

func (m *mockEventPublisher) Close() error {
	return nil
}

// TestNewSagaEventHandler tests the NewSagaEventHandler constructor.
func TestNewSagaEventHandler(t *testing.T) {
	tests := []struct {
		name        string
		config      *HandlerConfig
		opts        []HandlerOption
		expectError bool
		expectedErr error
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			expectedErr: ErrInvalidContext,
		},
		{
			name: "invalid config - no handler ID",
			config: &HandlerConfig{
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			},
			expectError: true,
			expectedErr: ErrInvalidHandlerID,
		},
		{
			name: "missing coordinator",
			config: &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			},
			expectError: true,
			expectedErr: ErrHandlerNotInitialized,
		},
		{
			name: "valid config with coordinator",
			config: &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			},
			opts: []HandlerOption{
				WithCoordinator(&mockCoordinator{}),
			},
			expectError: false,
		},
		{
			name: "valid config with coordinator and publisher",
			config: &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			},
			opts: []HandlerOption{
				WithCoordinator(&mockCoordinator{}),
				WithEventPublisher(&mockEventPublisher{}),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewSagaEventHandler(tt.config, tt.opts...)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.expectedErr != nil && err != tt.expectedErr {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if handler == nil {
					t.Error("expected handler, got nil")
				}
			}
		})
	}
}

// TestHandlerStartStop tests the Start and Stop lifecycle methods.
func TestHandlerStartStop(t *testing.T) {
	tests := []struct {
		name                 string
		setupHandler         func() SagaEventHandler
		operationSequence    []string
		expectStartError     bool
		expectStopError      bool
		coordinatorUnhealthy bool
	}{
		{
			name: "normal start and stop",
			setupHandler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-handler",
					HandlerName: "Test Handler",
					Topics:      []string{"test-topic"},
				}
				handler, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return handler
			},
			operationSequence: []string{"start", "stop"},
			expectStartError:  false,
			expectStopError:   false,
		},
		{
			name: "start twice",
			setupHandler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-handler",
					HandlerName: "Test Handler",
					Topics:      []string{"test-topic"},
				}
				handler, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return handler
			},
			operationSequence: []string{"start", "start"},
			expectStartError:  true,
		},
		{
			name: "stop without start",
			setupHandler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-handler",
					HandlerName: "Test Handler",
					Topics:      []string{"test-topic"},
				}
				handler, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return handler
			},
			operationSequence: []string{"stop"},
			expectStopError:   true,
		},
		{
			name: "stop twice",
			setupHandler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-handler",
					HandlerName: "Test Handler",
					Topics:      []string{"test-topic"},
				}
				handler, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return handler
			},
			operationSequence: []string{"start", "stop", "stop"},
			expectStopError:   true,
		},
		{
			name: "start with unhealthy coordinator",
			setupHandler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-handler",
					HandlerName: "Test Handler",
					Topics:      []string{"test-topic"},
				}
				handler, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{
					healthCheckErr: errors.New("coordinator unhealthy"),
				}))
				return handler
			},
			operationSequence:    []string{"start"},
			expectStartError:     true,
			coordinatorUnhealthy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.setupHandler()
			ctx := context.Background()

			var lastStartErr, lastStopErr error

			for _, op := range tt.operationSequence {
				switch op {
				case "start":
					// Type assertion to access Start method
					if h, ok := handler.(*defaultSagaEventHandler); ok {
						lastStartErr = h.Start(ctx)
					}
				case "stop":
					// Type assertion to access Stop method
					if h, ok := handler.(*defaultSagaEventHandler); ok {
						lastStopErr = h.Stop(ctx)
					}
				}
			}

			if tt.expectStartError && lastStartErr == nil && tt.operationSequence[len(tt.operationSequence)-1] == "start" {
				t.Error("expected start error, got nil")
			}

			if tt.expectStopError && lastStopErr == nil && tt.operationSequence[len(tt.operationSequence)-1] == "stop" {
				t.Error("expected stop error, got nil")
			}

			if !tt.expectStartError && !tt.expectStopError && !tt.coordinatorUnhealthy {
				if lastStartErr != nil {
					t.Errorf("unexpected start error: %v", lastStartErr)
				}
				if lastStopErr != nil {
					t.Errorf("unexpected stop error: %v", lastStopErr)
				}
			}
		})
	}
}

// TestHandleSagaEvent tests the HandleSagaEvent method.
func TestHandleSagaEvent(t *testing.T) {
	tests := []struct {
		name        string
		handler     func() SagaEventHandler
		event       *saga.SagaEvent
		handlerCtx  *EventHandlerContext
		startFirst  bool
		expectError bool
	}{
		{
			name: "valid event processing",
			handler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-handler",
					HandlerName: "Test Handler",
					Topics:      []string{"test-topic"},
				}
				h, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return h
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-1",
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-1",
				Topic:     "test-topic",
			},
			startFirst:  true,
			expectError: false,
		},
		{
			name: "nil event",
			handler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-handler",
					HandlerName: "Test Handler",
					Topics:      []string{"test-topic"},
				}
				h, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return h
			},
			event: nil,
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-1",
			},
			startFirst:  true,
			expectError: true,
		},
		{
			name: "nil handler context",
			handler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-handler",
					HandlerName: "Test Handler",
					Topics:      []string{"test-topic"},
				}
				h, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return h
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-1",
			},
			handlerCtx:  nil,
			startFirst:  true,
			expectError: true,
		},
		{
			name: "handler not started",
			handler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-handler",
					HandlerName: "Test Handler",
					Topics:      []string{"test-topic"},
				}
				h, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return h
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-1",
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-1",
			},
			startFirst:  false,
			expectError: true,
		},
		{
			name: "filtered event",
			handler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-handler",
					HandlerName: "Test Handler",
					Topics:      []string{"test-topic"},
					FilterConfig: &FilterConfig{
						IncludeEventTypes: []SagaEventType{EventTypeSagaStarted},
					},
				}
				h, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return h
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-1",
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-1",
			},
			startFirst:  true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.handler()
			ctx := context.Background()

			if tt.startFirst {
				if h, ok := handler.(*defaultSagaEventHandler); ok {
					if err := h.Start(ctx); err != nil {
						t.Fatalf("failed to start handler: %v", err)
					}
					defer h.Stop(ctx)
				}
			}

			err := handler.HandleSagaEvent(ctx, tt.event, tt.handlerCtx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestHandlerGetSupportedEventTypes tests the GetSupportedEventTypes method.
func TestHandlerGetSupportedEventTypes(t *testing.T) {
	tests := []struct {
		name          string
		config        *HandlerConfig
		expectedCount int
		expectedTypes []SagaEventType
	}{
		{
			name: "no filter - all types",
			config: &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			},
			expectedCount: 17, // All event types
		},
		{
			name: "include specific types",
			config: &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
				FilterConfig: &FilterConfig{
					IncludeEventTypes: []SagaEventType{
						EventTypeSagaStepCompleted,
						EventTypeSagaStepFailed,
					},
				},
			},
			expectedCount: 2,
			expectedTypes: []SagaEventType{
				EventTypeSagaStepCompleted,
				EventTypeSagaStepFailed,
			},
		},
		{
			name: "exclude specific types",
			config: &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
				FilterConfig: &FilterConfig{
					ExcludeEventTypes: []SagaEventType{
						EventTypeSagaStepFailed,
					},
				},
			},
			expectedCount: 16, // All types except one
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewSagaEventHandler(tt.config, WithCoordinator(&mockCoordinator{}))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			types := handler.GetSupportedEventTypes()

			if len(types) != tt.expectedCount {
				t.Errorf("expected %d types, got %d", tt.expectedCount, len(types))
			}

			if tt.expectedTypes != nil {
				for _, expectedType := range tt.expectedTypes {
					found := false
					for _, actualType := range types {
						if actualType == expectedType {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected type %v not found in supported types", expectedType)
					}
				}
			}
		})
	}
}

// TestHandlerCanHandle tests the CanHandle method.
func TestHandlerCanHandle(t *testing.T) {
	tests := []struct {
		name     string
		config   *HandlerConfig
		event    *saga.SagaEvent
		expected bool
	}{
		{
			name: "can handle supported event",
			config: &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-1",
			},
			expected: true,
		},
		{
			name: "nil event",
			config: &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			},
			event:    nil,
			expected: false,
		},
		{
			name: "filtered out event",
			config: &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
				FilterConfig: &FilterConfig{
					ExcludeEventTypes: []SagaEventType{EventTypeSagaStepCompleted},
				},
			},
			event: &saga.SagaEvent{
				Type:   saga.EventSagaStepCompleted,
				SagaID: "saga-1",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewSagaEventHandler(tt.config, WithCoordinator(&mockCoordinator{}))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			result := handler.CanHandle(tt.event)

			if result != tt.expected {
				t.Errorf("CanHandle() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestHandlerGetConfiguration tests the GetConfiguration method.
func TestHandlerGetConfiguration(t *testing.T) {
	config := &HandlerConfig{
		HandlerID:   "test-handler",
		HandlerName: "Test Handler",
		Topics:      []string{"test-topic"},
	}

	handler, err := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	retrievedConfig := handler.GetConfiguration()

	if retrievedConfig.HandlerID != config.HandlerID {
		t.Errorf("HandlerID = %v, want %v", retrievedConfig.HandlerID, config.HandlerID)
	}
	if retrievedConfig.HandlerName != config.HandlerName {
		t.Errorf("HandlerName = %v, want %v", retrievedConfig.HandlerName, config.HandlerName)
	}
}

// TestHandlerGetMetrics tests the GetMetrics method.
func TestHandlerGetMetrics(t *testing.T) {
	config := &HandlerConfig{
		HandlerID:   "test-handler",
		HandlerName: "Test Handler",
		Topics:      []string{"test-topic"},
	}

	handler, err := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	metrics := handler.GetMetrics()

	if metrics == nil {
		t.Fatal("expected metrics, got nil")
	}

	if metrics.HandlerID != config.HandlerID {
		t.Errorf("HandlerID = %v, want %v", metrics.HandlerID, config.HandlerID)
	}

	if metrics.EventTypeMetrics == nil {
		t.Error("EventTypeMetrics should not be nil")
	}

	if !metrics.IsHealthy {
		t.Error("IsHealthy should be true for new handler")
	}
}

// TestHandlerHealthCheck tests the HealthCheck method.
func TestHandlerHealthCheck(t *testing.T) {
	tests := []struct {
		name         string
		coordinator  *mockCoordinator
		startHandler bool
		expectError  bool
	}{
		{
			name:         "healthy handler",
			coordinator:  &mockCoordinator{},
			startHandler: true,
			expectError:  false,
		},
		{
			name:         "handler not started",
			coordinator:  &mockCoordinator{},
			startHandler: false,
			expectError:  true,
		},
		{
			name: "unhealthy coordinator",
			coordinator: &mockCoordinator{
				healthCheckErr: errors.New("coordinator unhealthy"),
			},
			startHandler: true,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			}

			handler, err := NewSagaEventHandler(config, WithCoordinator(tt.coordinator))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			ctx := context.Background()

			if tt.startHandler {
				if h, ok := handler.(*defaultSagaEventHandler); ok {
					// Ignore start error for unhealthy coordinator test
					h.Start(ctx)
					defer h.Stop(ctx)
				}
			}

			err = handler.HealthCheck(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestHandlerUpdateConfiguration tests the UpdateConfiguration method.
func TestHandlerUpdateConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		newConfig   *HandlerConfig
		startFirst  bool
		expectError bool
	}{
		{
			name: "valid configuration update",
			newConfig: &HandlerConfig{
				HandlerID:         "test-handler",
				HandlerName:       "Test Handler Updated",
				Topics:            []string{"test-topic"},
				ProcessingTimeout: 60 * time.Second,
			},
			startFirst:  true,
			expectError: false,
		},
		{
			name:        "nil configuration",
			newConfig:   nil,
			startFirst:  true,
			expectError: true,
		},
		{
			name: "invalid configuration",
			newConfig: &HandlerConfig{
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			},
			startFirst:  true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			}

			handler, err := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			ctx := context.Background()

			if tt.startFirst {
				if h, ok := handler.(*defaultSagaEventHandler); ok {
					if err := h.Start(ctx); err != nil {
						t.Fatalf("failed to start handler: %v", err)
					}
					defer h.Stop(ctx)
				}
			}

			err = handler.UpdateConfiguration(tt.newConfig)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestHandlerPriority tests the GetPriority method.
func TestHandlerPriority(t *testing.T) {
	config := &HandlerConfig{
		HandlerID:   "test-handler",
		HandlerName: "Test Handler",
		Topics:      []string{"test-topic"},
	}

	handler, err := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	priority := handler.GetPriority()

	// Default priority should be 0
	if priority != 0 {
		t.Errorf("expected priority 0, got %d", priority)
	}
}

// TestRetryPolicy tests the RetryPolicy structure
func TestRetryPolicy(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:        3,
		InitialDelay:      1 * time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableErrors:   []string{"timeout", "connection_error"},
	}

	// Verify policy fields
	if policy.MaxRetries != 3 {
		t.Errorf("MaxRetries = %v, want %v", policy.MaxRetries, 3)
	}
	if policy.InitialDelay != 1*time.Second {
		t.Errorf("InitialDelay = %v, want %v", policy.InitialDelay, 1*time.Second)
	}
	if policy.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want %v", policy.MaxDelay, 30*time.Second)
	}
	if policy.BackoffMultiplier != 2.0 {
		t.Errorf("BackoffMultiplier = %v, want %v", policy.BackoffMultiplier, 2.0)
	}
	if len(policy.RetryableErrors) != 2 {
		t.Errorf("RetryableErrors length = %v, want %v", len(policy.RetryableErrors), 2)
	}
}

// TestDeadLetterConfig tests the DeadLetterConfig structure
func TestDeadLetterConfig(t *testing.T) {
	config := &DeadLetterConfig{
		Enabled:                true,
		Topic:                  "saga-dlq",
		MaxRetentionDays:       7,
		IncludeOriginalMessage: true,
		IncludeErrorDetails:    true,
	}

	// Verify config fields
	if !config.Enabled {
		t.Error("Enabled should be true")
	}
	if config.Topic != "saga-dlq" {
		t.Errorf("Topic = %v, want %v", config.Topic, "saga-dlq")
	}
	if config.MaxRetentionDays != 7 {
		t.Errorf("MaxRetentionDays = %v, want %v", config.MaxRetentionDays, 7)
	}
	if !config.IncludeOriginalMessage {
		t.Error("IncludeOriginalMessage should be true")
	}
	if !config.IncludeErrorDetails {
		t.Error("IncludeErrorDetails should be true")
	}
}

// TestHandleStepCompleted tests the step completed event handling
func TestHandleStepCompleted(t *testing.T) {
	tests := []struct {
		name        string
		event       *saga.SagaEvent
		expectError bool
	}{
		{
			name: "valid step completed event",
			event: &saga.SagaEvent{
				Type:      EventTypeSagaStepCompleted,
				SagaID:    "saga-1",
				StepID:    "step-1",
				Timestamp: time.Now(),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			}

			handler, err := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			ctx := context.Background()
			if h, ok := handler.(*defaultSagaEventHandler); ok {
				if err := h.Start(ctx); err != nil {
					t.Fatalf("failed to start handler: %v", err)
				}
				defer h.Stop(ctx)
			}

			handlerCtx := &EventHandlerContext{
				MessageID: "msg-1",
				Timestamp: time.Now(),
			}

			err = handler.HandleSagaEvent(ctx, tt.event, handlerCtx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestHandleStepFailed tests the step failed event handling
func TestHandleStepFailed(t *testing.T) {
	tests := []struct {
		name        string
		event       *saga.SagaEvent
		expectError bool
	}{
		{
			name: "valid step failed event",
			event: &saga.SagaEvent{
				Type:   EventTypeSagaStepFailed,
				SagaID: "saga-1",
				StepID: "step-1",
				Error: &saga.SagaError{
					Type:    saga.ErrorTypeService,
					Message: "service unavailable",
				},
				Timestamp: time.Now(),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			}

			handler, err := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			ctx := context.Background()
			if h, ok := handler.(*defaultSagaEventHandler); ok {
				if err := h.Start(ctx); err != nil {
					t.Fatalf("failed to start handler: %v", err)
				}
				defer h.Stop(ctx)
			}

			handlerCtx := &EventHandlerContext{
				MessageID: "msg-1",
				Timestamp: time.Now(),
			}

			err = handler.HandleSagaEvent(ctx, tt.event, handlerCtx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestHandleCompensationEvents tests compensation event handling
func TestHandleCompensationEvents(t *testing.T) {
	tests := []struct {
		name        string
		event       *saga.SagaEvent
		expectError bool
	}{
		{
			name: "compensation started",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationStarted,
				SagaID:    "saga-1",
				Timestamp: time.Now(),
			},
			expectError: false,
		},
		{
			name: "compensation step completed",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationStepCompleted,
				SagaID:    "saga-1",
				StepID:    "step-2",
				Timestamp: time.Now(),
			},
			expectError: false,
		},
		{
			name: "compensation step failed",
			event: &saga.SagaEvent{
				Type:   EventTypeCompensationStepFailed,
				SagaID: "saga-1",
				StepID: "step-1",
				Error: &saga.SagaError{
					Type:    saga.ErrorTypeCompensation,
					Message: "compensation failed",
				},
				Timestamp: time.Now(),
			},
			expectError: false,
		},
		{
			name: "compensation completed",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationCompleted,
				SagaID:    "saga-1",
				Timestamp: time.Now(),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			}

			handler, err := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			ctx := context.Background()
			if h, ok := handler.(*defaultSagaEventHandler); ok {
				if err := h.Start(ctx); err != nil {
					t.Fatalf("failed to start handler: %v", err)
				}
				defer h.Stop(ctx)
			}

			handlerCtx := &EventHandlerContext{
				MessageID: "msg-1",
				Timestamp: time.Now(),
			}

			err = handler.HandleSagaEvent(ctx, tt.event, handlerCtx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestHandleSagaLifecycleEvents tests saga lifecycle event handling
func TestHandleSagaLifecycleEvents(t *testing.T) {
	tests := []struct {
		name        string
		event       *saga.SagaEvent
		expectError bool
	}{
		{
			name: "saga started",
			event: &saga.SagaEvent{
				Type:      EventTypeSagaStarted,
				SagaID:    "saga-1",
				Timestamp: time.Now(),
			},
			expectError: false,
		},
		{
			name: "saga completed",
			event: &saga.SagaEvent{
				Type:      EventTypeSagaCompleted,
				SagaID:    "saga-1",
				Timestamp: time.Now(),
			},
			expectError: false,
		},
		{
			name: "saga failed",
			event: &saga.SagaEvent{
				Type:   EventTypeSagaFailed,
				SagaID: "saga-1",
				Error: &saga.SagaError{
					Type:    saga.ErrorTypeService,
					Message: "saga failed",
				},
				Timestamp: time.Now(),
			},
			expectError: false,
		},
		{
			name: "saga cancelled",
			event: &saga.SagaEvent{
				Type:      EventTypeSagaCancelled,
				SagaID:    "saga-1",
				Timestamp: time.Now(),
			},
			expectError: false,
		},
		{
			name: "saga timed out",
			event: &saga.SagaEvent{
				Type:      EventTypeSagaTimedOut,
				SagaID:    "saga-1",
				Timestamp: time.Now(),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			}

			handler, err := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			ctx := context.Background()
			if h, ok := handler.(*defaultSagaEventHandler); ok {
				if err := h.Start(ctx); err != nil {
					t.Fatalf("failed to start handler: %v", err)
				}
				defer h.Stop(ctx)
			}

			handlerCtx := &EventHandlerContext{
				MessageID: "msg-1",
				Timestamp: time.Now(),
			}

			err = handler.HandleSagaEvent(ctx, tt.event, handlerCtx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestHandleRetryEvents tests retry event handling
func TestHandleRetryEvents(t *testing.T) {
	tests := []struct {
		name        string
		event       *saga.SagaEvent
		retryCount  int
		expectError bool
	}{
		{
			name: "retry attempted",
			event: &saga.SagaEvent{
				Type:      EventTypeRetryAttempted,
				SagaID:    "saga-1",
				Timestamp: time.Now(),
			},
			retryCount:  1,
			expectError: false,
		},
		{
			name: "retry exhausted",
			event: &saga.SagaEvent{
				Type:      EventTypeRetryExhausted,
				SagaID:    "saga-1",
				Timestamp: time.Now(),
			},
			retryCount:  3,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HandlerConfig{
				HandlerID:   "test-handler",
				HandlerName: "Test Handler",
				Topics:      []string{"test-topic"},
			}

			handler, err := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			ctx := context.Background()
			if h, ok := handler.(*defaultSagaEventHandler); ok {
				if err := h.Start(ctx); err != nil {
					t.Fatalf("failed to start handler: %v", err)
				}
				defer h.Stop(ctx)
			}

			handlerCtx := &EventHandlerContext{
				MessageID:  "msg-1",
				Timestamp:  time.Now(),
				RetryCount: tt.retryCount,
			}

			err = handler.HandleSagaEvent(ctx, tt.event, handlerCtx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestEventProcessingWithFilters tests event processing with filter configuration
func TestEventProcessingWithFilters(t *testing.T) {
	tests := []struct {
		name           string
		filterConfig   *FilterConfig
		event          *saga.SagaEvent
		expectFiltered bool
	}{
		{
			name: "event passes include filter",
			filterConfig: &FilterConfig{
				IncludeEventTypes: []SagaEventType{EventTypeSagaCompleted},
			},
			event: &saga.SagaEvent{
				Type:      EventTypeSagaCompleted,
				SagaID:    "saga-1",
				Timestamp: time.Now(),
			},
			expectFiltered: false,
		},
		{
			name: "event filtered by include filter",
			filterConfig: &FilterConfig{
				IncludeEventTypes: []SagaEventType{EventTypeSagaCompleted},
			},
			event: &saga.SagaEvent{
				Type:      EventTypeSagaFailed,
				SagaID:    "saga-1",
				Timestamp: time.Now(),
			},
			expectFiltered: true,
		},
		{
			name: "event filtered by exclude filter",
			filterConfig: &FilterConfig{
				ExcludeEventTypes: []SagaEventType{EventTypeSagaFailed},
			},
			event: &saga.SagaEvent{
				Type:      EventTypeSagaFailed,
				SagaID:    "saga-1",
				Timestamp: time.Now(),
			},
			expectFiltered: true,
		},
		{
			name: "event filtered by saga ID",
			filterConfig: &FilterConfig{
				IncludeSagaIDs: []string{"saga-1", "saga-2"},
			},
			event: &saga.SagaEvent{
				Type:      EventTypeSagaCompleted,
				SagaID:    "saga-3",
				Timestamp: time.Now(),
			},
			expectFiltered: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HandlerConfig{
				HandlerID:    "test-handler",
				HandlerName:  "Test Handler",
				Topics:       []string{"test-topic"},
				FilterConfig: tt.filterConfig,
			}

			handler, err := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			ctx := context.Background()
			if h, ok := handler.(*defaultSagaEventHandler); ok {
				if err := h.Start(ctx); err != nil {
					t.Fatalf("failed to start handler: %v", err)
				}
				defer h.Stop(ctx)
			}

			handlerCtx := &EventHandlerContext{
				MessageID: "msg-1",
				Timestamp: time.Now(),
			}

			err = handler.HandleSagaEvent(ctx, tt.event, handlerCtx)

			if tt.expectFiltered {
				if err != ErrEventFilteredOut {
					t.Errorf("expected ErrEventFilteredOut, got %v", err)
				}
			} else {
				if err == ErrEventFilteredOut {
					t.Error("event should not be filtered")
				}
			}
		})
	}
}

// TestHandleStepStarted tests the handleStepStarted function with comprehensive coverage
func TestHandleStepStarted(t *testing.T) {
	tests := []struct {
		name             string
		event            *saga.SagaEvent
		handlerCtx       *EventHandlerContext
		coordinator      *mockCoordinator
		expectError      bool
		expectedError    error
		expectedErrorMsg string
	}{
		{
			name: "valid step started event",
			event: &saga.SagaEvent{
				Type:      EventTypeSagaStepStarted,
				SagaID:    "saga-123",
				StepID:    "step-1",
				Timestamp: time.Now(),
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-456",
				Topic:     "saga-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "step started event with minimal data",
			event: &saga.SagaEvent{
				Type:      saga.EventSagaStepStarted,
				SagaID:    "saga-minimal",
				Timestamp: time.Now(),
				// StepID is optional in current implementation
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-minimal",
				Topic:     "saga-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "step started event with step ID and data",
			event: &saga.SagaEvent{
				Type:      saga.EventSagaStepStarted,
				SagaID:    "saga-with-step",
				StepID:    "step-1",
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"step_input": "test_data"},
			},
			handlerCtx: &EventHandlerContext{
				MessageID:     "msg-with-step",
				Topic:         "saga-events",
				Partition:     1,
				Offset:        100,
				Timestamp:     time.Now(),
				RetryCount:    0,
				BrokerName:    "kafka",
				ConsumerGroup: "saga-handlers",
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "step started event with valid saga but minimal context",
			event: &saga.SagaEvent{
				Type:      EventTypeSagaStepStarted,
				SagaID:    "saga-minimal",
				StepID:    "step-minimal",
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"step_data": "test"},
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-minimal",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "step started event with complex data",
			event: &saga.SagaEvent{
				Type:      EventTypeSagaStepStarted,
				SagaID:    "saga-complex",
				StepID:    "step-complex",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"input":      "complex_input",
					"parameters": map[string]interface{}{"param1": "value1"},
					"metadata":   map[string]interface{}{"source": "test", "version": "1.0"},
				},
				Metadata: map[string]interface{}{
					"retry_count": 0,
					"priority":    "high",
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID:     "msg-complex",
				Topic:         "saga-events",
				Partition:     1,
				Offset:        12345,
				Timestamp:     time.Now(),
				RetryCount:    0,
				BrokerName:    "kafka",
				ConsumerGroup: "saga-handlers",
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HandlerConfig{
				HandlerID:   "test-step-handler",
				HandlerName: "Test Step Handler",
				Topics:      []string{"saga-events"},
			}

			handler, err := NewSagaEventHandler(config, WithCoordinator(tt.coordinator))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			ctx := context.Background()
			if h, ok := handler.(*defaultSagaEventHandler); ok {
				if err := h.Start(ctx); err != nil {
					t.Fatalf("failed to start handler: %v", err)
				}
				defer h.Stop(ctx)
			}

			err = handler.HandleSagaEvent(ctx, tt.event, tt.handlerCtx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				} else {
					if tt.expectedErrorMsg != "" && !contains(err.Error(), tt.expectedErrorMsg) {
						t.Errorf("expected error message to contain '%s', got '%s'", tt.expectedErrorMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestHandleCompensationStepStarted tests the handleCompensationStepStarted function with comprehensive coverage
func TestHandleCompensationStepStarted(t *testing.T) {
	tests := []struct {
		name             string
		event            *saga.SagaEvent
		handlerCtx       *EventHandlerContext
		coordinator      *mockCoordinator
		expectError      bool
		expectedError    error
		expectedErrorMsg string
	}{
		{
			name: "valid compensation step started event",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationStepStarted,
				SagaID:    "saga-comp-123",
				StepID:    "comp-step-1",
				Timestamp: time.Now(),
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-comp-456",
				Topic:     "compensation-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "compensation step started event with minimal data",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationStepStarted,
				SagaID:    "saga-comp-minimal",
				Timestamp: time.Now(),
				// StepID is optional in current implementation
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-comp-minimal",
				Topic:     "compensation-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "compensation step started event with step ID",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationStepStarted,
				SagaID:    "saga-comp-with-step",
				StepID:    "comp-step-1",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"original_step_id":  "step-1",
					"compensation_type": "rollback",
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID:     "msg-comp-with-step",
				Topic:         "compensation-events",
				Partition:     2,
				Offset:        200,
				Timestamp:     time.Now(),
				RetryCount:    1,
				BrokerName:    "kafka",
				ConsumerGroup: "compensation-handlers",
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "compensation step started with zero timestamp",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationStepStarted,
				SagaID:    "saga-comp-zero",
				StepID:    "comp-step-zero",
				Timestamp: time.Time{}, // Zero time - should still work as timestamp not validated in this handler
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-comp-zero",
				Topic:     "compensation-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "compensation step started with compensation data",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationStepStarted,
				SagaID:    "saga-comp-data",
				StepID:    "comp-step-data",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"original_step_id":  "step-1",
					"compensation_type": "rollback",
					"rollback_data":     map[string]interface{}{"previous_state": "state1"},
				},
				Metadata: map[string]interface{}{
					"compensation_reason": "step_failed",
					"original_error":      "service timeout",
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID:     "msg-comp-data",
				Topic:         "compensation-events",
				Partition:     2,
				Offset:        67890,
				Timestamp:     time.Now(),
				RetryCount:    1,
				BrokerName:    "kafka",
				ConsumerGroup: "compensation-handlers",
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "compensation step started for saga with multiple steps",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationStepStarted,
				SagaID:    "saga-multi-step",
				StepID:    "step-3-reverse", // Compensation for step 3
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"total_compensation_steps":  3,
					"current_compensation_step": 2, // Processing step 3 first, then 2, then 1
					"compensation_order":        []string{"step-3-reverse", "step-2-reverse", "step-1-reverse"},
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-multi-step",
				Topic:     "compensation-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HandlerConfig{
				HandlerID:   "test-compensation-handler",
				HandlerName: "Test Compensation Handler",
				Topics:      []string{"compensation-events"},
			}

			handler, err := NewSagaEventHandler(config, WithCoordinator(tt.coordinator))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			ctx := context.Background()
			if h, ok := handler.(*defaultSagaEventHandler); ok {
				if err := h.Start(ctx); err != nil {
					t.Fatalf("failed to start handler: %v", err)
				}
				defer h.Stop(ctx)
			}

			err = handler.HandleSagaEvent(ctx, tt.event, tt.handlerCtx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				} else {
					if tt.expectedErrorMsg != "" && !contains(err.Error(), tt.expectedErrorMsg) {
						t.Errorf("expected error message to contain '%s', got '%s'", tt.expectedErrorMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestHandleCompensationFailed tests the handleCompensationFailed function with comprehensive coverage
func TestHandleCompensationFailed(t *testing.T) {
	tests := []struct {
		name             string
		event            *saga.SagaEvent
		handlerCtx       *EventHandlerContext
		coordinator      *mockCoordinator
		expectError      bool
		expectedError    error
		expectedErrorMsg string
	}{
		{
			name: "valid compensation failed event",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationFailed,
				SagaID:    "saga-comp-failed-123",
				Timestamp: time.Now(),
				Error: &saga.SagaError{
					Type:    saga.ErrorTypeCompensation,
					Message: "compensation service unavailable",
					Details: map[string]interface{}{"retry_count": 3},
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-comp-failed-456",
				Topic:     "compensation-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "compensation failed event with minimal data",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationFailed,
				SagaID:    "saga-comp-minimal-failed",
				Timestamp: time.Now(),
				// Error is optional in current implementation
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-comp-minimal-failed",
				Topic:     "compensation-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "compensation failed event with error details",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationFailed,
				SagaID:    "saga-comp-with-error",
				Timestamp: time.Now(),
				Error: &saga.SagaError{
					Type:    saga.ErrorTypeCompensation,
					Message: "compensation service unavailable",
					Details: map[string]interface{}{
						"retry_count": 3,
						"service":     "payment-service",
					},
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID:     "msg-comp-with-error",
				Topic:         "compensation-events",
				Partition:     3,
				Offset:        300,
				Timestamp:     time.Now(),
				RetryCount:    2,
				BrokerName:    "kafka",
				ConsumerGroup: "compensation-handlers",
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "compensation failed with different error types",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationFailed,
				SagaID:    "saga-comp-error-types",
				Timestamp: time.Now(),
				Error: &saga.SagaError{
					Type:    saga.ErrorTypeService,
					Message: "database connection failed during compensation",
					Details: map[string]interface{}{
						"database": "postgres",
						"query":    "ROLLBACK saga_transaction",
						"timeout":  "30s",
					},
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID:     "msg-comp-error-types",
				Topic:         "compensation-events",
				Partition:     3,
				Offset:        11111,
				Timestamp:     time.Now(),
				RetryCount:    2,
				BrokerName:    "kafka",
				ConsumerGroup: "compensation-handlers",
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "compensation failed with zero timestamp (should still work)",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationFailed,
				SagaID:    "saga-comp-zero-time",
				Timestamp: time.Time{}, // Zero time
				Error: &saga.SagaError{
					Type:    saga.ErrorTypeCompensation,
					Message: "compensation failed with zero timestamp",
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-comp-zero-time",
				Topic:     "compensation-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "compensation failed with complex error details",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationFailed,
				SagaID:    "saga-comp-complex",
				Timestamp: time.Now(),
				Error: &saga.SagaError{
					Type:      saga.ErrorTypeCompensation,
					Message:   "complex compensation failure",
					Code:      "COMPENSATION_COMPLEX_ERROR",
					Retryable: false,
					Details: map[string]interface{}{
						"failed_operations": []string{"delete_record", "send_notification", "update_inventory"},
						"partial_success":   true,
						"completed_steps":   2,
						"total_steps":       3,
						"remaining_data": map[string]interface{}{
							"records_to_cleanup": 5,
							"notifications_sent": 2,
						},
						"recovery_options": []string{"manual_intervention", "retry_with_different_strategy"},
					},
				},
				Data: map[string]interface{}{
					"compensation_failure_summary": "Partial compensation completed",
					"manual_review_required":       true,
				},
				Metadata: map[string]interface{}{
					"critical_failure": true,
					"escalation_level": 2,
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-comp-complex",
				Topic:     "compensation-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HandlerConfig{
				HandlerID:   "test-compensation-failed-handler",
				HandlerName: "Test Compensation Failed Handler",
				Topics:      []string{"compensation-events"},
			}

			handler, err := NewSagaEventHandler(config, WithCoordinator(tt.coordinator))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			ctx := context.Background()
			if h, ok := handler.(*defaultSagaEventHandler); ok {
				if err := h.Start(ctx); err != nil {
					t.Fatalf("failed to start handler: %v", err)
				}
				defer h.Stop(ctx)
			}

			err = handler.HandleSagaEvent(ctx, tt.event, tt.handlerCtx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				} else {
					if tt.expectedErrorMsg != "" && !contains(err.Error(), tt.expectedErrorMsg) {
						t.Errorf("expected error message to contain '%s', got '%s'", tt.expectedErrorMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestHandleStateChanged tests the handleStateChanged function with comprehensive coverage
func TestHandleStateChanged(t *testing.T) {
	tests := []struct {
		name             string
		event            *saga.SagaEvent
		handlerCtx       *EventHandlerContext
		coordinator      *mockCoordinator
		expectError      bool
		expectedError    error
		expectedErrorMsg string
	}{
		{
			name: "valid state changed event",
			event: &saga.SagaEvent{
				Type:      EventTypeStateChanged,
				SagaID:    "saga-state-123",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"previous_state":      "processing",
					"current_state":       "completed",
					"state_change_reason": "all_steps_completed_successfully",
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-state-456",
				Topic:     "state-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "state changed event with zero timestamp",
			event: &saga.SagaEvent{
				Type:      EventTypeStateChanged,
				SagaID:    "saga-state-zero-time",
				Timestamp: time.Time{}, // Zero timestamp - works in current implementation
				Data: map[string]interface{}{
					"previous_state": "running",
					"current_state":  "failed",
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-state-zero-time",
				Topic:     "state-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "state changed event with full transition data",
			event: &saga.SagaEvent{
				Type:      EventTypeStateChanged,
				SagaID:    "saga-state-full",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"previous_state":    "active",
					"current_state":     "suspended",
					"transition_reason": "manual_pause",
					"metadata": map[string]interface{}{
						"initiated_by": "admin",
						"timestamp":    time.Now().Unix(),
					},
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID:     "msg-state-full",
				Topic:         "state-events",
				Partition:     0,
				Offset:        400,
				Timestamp:     time.Now(),
				RetryCount:    0,
				BrokerName:    "kafka",
				ConsumerGroup: "state-handlers",
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "state changed event with minimal data",
			event: &saga.SagaEvent{
				Type:      EventTypeStateChanged,
				SagaID:    "saga-state-minimal",
				Timestamp: time.Now(),
				// Data can be empty - should still work
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-state-minimal",
				Topic:     "state-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "state changed event with complex state transition",
			event: &saga.SagaEvent{
				Type:      EventTypeStateChanged,
				SagaID:    "saga-state-complex",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"previous_state":               "compensating",
					"current_state":                "compensation_failed",
					"state_change_reason":          "compensation_step_3_failed_permanently",
					"state_transition_duration_ms": 15000,
					"state_machine": map[string]interface{}{
						"current_step":      "cleanup_resources",
						"next_states":       []string{"manual_review", "terminated"},
						"recovery_possible": false,
					},
					"affected_entities": []string{
						"order_service",
						"inventory_service",
						"payment_service",
					},
				},
				Metadata: map[string]interface{}{
					"state_change_id": "transition-789",
					"triggered_by":    "compensation_failure",
					"automated":       true,
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID:     "msg-state-complex",
				Topic:         "state-events",
				Partition:     0,
				Offset:        99999,
				Timestamp:     time.Now(),
				RetryCount:    0,
				BrokerName:    "kafka",
				ConsumerGroup: "state-handlers",
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "state changed event for saga lifecycle transitions",
			event: &saga.SagaEvent{
				Type:      EventTypeStateChanged,
				SagaID:    "saga-state-lifecycle",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"previous_state":       "started",
					"current_state":        "executing",
					"state_change_reason":  "step_1_completed_starting_step_2",
					"saga_lifecycle_stage": "execution",
					"current_step_id":      "step-2",
					"total_steps":          5,
					"completed_steps":      1,
					"failed_steps":         0,
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-state-lifecycle",
				Topic:     "state-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
		{
			name: "state changed event with timestamp edge cases",
			event: &saga.SagaEvent{
				Type:      EventTypeStateChanged,
				SagaID:    "saga-state-edge-time",
				Timestamp: time.Unix(0, 1), // Very early timestamp but not zero
				Data: map[string]interface{}{
					"previous_state": "initializing",
					"current_state":  "ready",
				},
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-state-edge-time",
				Topic:     "state-events",
				Timestamp: time.Now(),
			},
			coordinator: &mockCoordinator{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HandlerConfig{
				HandlerID:   "test-state-handler",
				HandlerName: "Test State Handler",
				Topics:      []string{"state-events"},
			}

			handler, err := NewSagaEventHandler(config, WithCoordinator(tt.coordinator))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			ctx := context.Background()
			if h, ok := handler.(*defaultSagaEventHandler); ok {
				if err := h.Start(ctx); err != nil {
					t.Fatalf("failed to start handler: %v", err)
				}
				defer h.Stop(ctx)
			}

			err = handler.HandleSagaEvent(ctx, tt.event, tt.handlerCtx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				} else {
					if tt.expectedErrorMsg != "" && !contains(err.Error(), tt.expectedErrorMsg) {
						t.Errorf("expected error message to contain '%s', got '%s'", tt.expectedErrorMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestOnEventProcessed tests the OnEventProcessed callback function with comprehensive coverage
func TestOnEventProcessed(t *testing.T) {
	tests := []struct {
		name             string
		event            *saga.SagaEvent
		result           interface{}
		setupHandler     func() SagaEventHandler
		expectedBehavior string
	}{
		{
			name: "callback with successful event processing",
			event: &saga.SagaEvent{
				Type:      EventTypeSagaStepCompleted,
				SagaID:    "saga-callback-123",
				StepID:    "step-callback-1",
				Timestamp: time.Now(),
			},
			result: map[string]interface{}{
				"processing_time_ms": 150,
				"result_data":        "step_completed_successfully",
			},
			setupHandler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-callback-handler",
					HandlerName: "Test Callback Handler",
					Topics:      []string{"callback-events"},
				}
				handler, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return handler
			},
			expectedBehavior: "callback_completes_without_error",
		},
		{
			name: "callback with nil result",
			event: &saga.SagaEvent{
				Type:      EventTypeSagaCompleted,
				SagaID:    "saga-callback-nil",
				Timestamp: time.Now(),
			},
			result: nil,
			setupHandler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-callback-nil-handler",
					HandlerName: "Test Callback Nil Handler",
					Topics:      []string{"callback-events"},
				}
				handler, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return handler
			},
			expectedBehavior: "callback_handles_nil_result",
		},
		{
			name: "callback with complex result data",
			event: &saga.SagaEvent{
				Type:      EventTypeCompensationCompleted,
				SagaID:    "saga-callback-complex",
				Timestamp: time.Now(),
			},
			result: map[string]interface{}{
				"compensation_summary": map[string]interface{}{
					"total_steps":              3,
					"successful_steps":         3,
					"failed_steps":             0,
					"compensation_duration_ms": 2500,
				},
				"resource_cleanup": map[string]interface{}{
					"database_transactions_rolled_back": 5,
					"messages_published":                2,
					"external_api_calls_made":           1,
				},
				"metrics": map[string]interface{}{
					"memory_freed_mb":         15.5,
					"temporary_files_deleted": 8,
				},
			},
			setupHandler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-callback-complex-handler",
					HandlerName: "Test Callback Complex Handler",
					Topics:      []string{"callback-events"},
				}
				handler, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return handler
			},
			expectedBehavior: "callback_handles_complex_result",
		},
		{
			name: "callback with different event types",
			event: &saga.SagaEvent{
				Type:      EventTypeStateChanged,
				SagaID:    "saga-callback-different",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"previous_state": "processing",
					"current_state":  "completed",
				},
			},
			result: "state_transition_successful",
			setupHandler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-callback-different-handler",
					HandlerName: "Test Callback Different Handler",
					Topics:      []string{"callback-events"},
				}
				handler, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return handler
			},
			expectedBehavior: "callback_handles_different_event_types",
		},
		{
			name: "callback with string result",
			event: &saga.SagaEvent{
				Type:      EventTypeRetryExhausted,
				SagaID:    "saga-callback-string",
				Timestamp: time.Now(),
			},
			result: "retry_exhausted_compensation_initiated",
			setupHandler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-callback-string-handler",
					HandlerName: "Test Callback String Handler",
					Topics:      []string{"callback-events"},
				}
				handler, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return handler
			},
			expectedBehavior: "callback_handles_string_result",
		},
		{
			name: "callback with numeric result",
			event: &saga.SagaEvent{
				Type:      EventTypeSagaStepFailed,
				SagaID:    "saga-callback-numeric",
				StepID:    "step-numeric",
				Timestamp: time.Now(),
			},
			result: 42, // Numeric result
			setupHandler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-callback-numeric-handler",
					HandlerName: "Test Callback Numeric Handler",
					Topics:      []string{"callback-events"},
				}
				handler, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return handler
			},
			expectedBehavior: "callback_handles_numeric_result",
		},
		{
			name: "callback with error result (should still handle gracefully)",
			event: &saga.SagaEvent{
				Type:      EventTypeSagaFailed,
				SagaID:    "saga-callback-error",
				Timestamp: time.Now(),
				Error: &saga.SagaError{
					Type:    saga.ErrorTypeService,
					Message: "service unavailable",
				},
			},
			result: errors.New("processing failed but handled"),
			setupHandler: func() SagaEventHandler {
				config := &HandlerConfig{
					HandlerID:   "test-callback-error-handler",
					HandlerName: "Test Callback Error Handler",
					Topics:      []string{"callback-events"},
				}
				handler, _ := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
				return handler
			},
			expectedBehavior: "callback_handles_error_result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.setupHandler()
			ctx := context.Background()

			// Start the handler
			if h, ok := handler.(*defaultSagaEventHandler); ok {
				if err := h.Start(ctx); err != nil {
					t.Fatalf("failed to start handler: %v", err)
				}
				defer h.Stop(ctx)
			}

			// Call OnEventProcessed callback - this should not panic or return an error
			// since it's a callback hook that's designed to be a no-op in the base implementation
			handler.OnEventProcessed(ctx, tt.event, tt.result)

			// If we reach here without panicking, the test passes
			// The base implementation is a no-op, so we just verify it doesn't crash
			t.Logf("OnEventProcessed callback completed successfully for test: %s", tt.name)
		})
	}
}

// TestOnEventProcessedIntegration tests OnEventProcessed in integration with HandleSagaEvent
func TestOnEventProcessedIntegration(t *testing.T) {
	tests := []struct {
		name        string
		event       *saga.SagaEvent
		handlerCtx  *EventHandlerContext
		expectError bool
	}{
		{
			name: "successful event processing triggers callback",
			event: &saga.SagaEvent{
				Type:      EventTypeSagaStepCompleted,
				SagaID:    "saga-integration-123",
				StepID:    "step-integration-1",
				Timestamp: time.Now(),
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-integration-456",
				Topic:     "integration-events",
				Timestamp: time.Now(),
			},
			expectError: false,
		},
		{
			name: "valid event processing triggers callback",
			event: &saga.SagaEvent{
				Type:      EventTypeSagaStepCompleted,
				SagaID:    "saga-integration-success",
				StepID:    "step-integration-success",
				Timestamp: time.Now(),
			},
			handlerCtx: &EventHandlerContext{
				MessageID: "msg-integration-success",
				Topic:     "integration-events",
				Timestamp: time.Now(),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HandlerConfig{
				HandlerID:   "test-integration-handler",
				HandlerName: "Test Integration Handler",
				Topics:      []string{"integration-events"},
			}

			handler, err := NewSagaEventHandler(config, WithCoordinator(&mockCoordinator{}))
			if err != nil {
				t.Fatalf("failed to create handler: %v", err)
			}

			ctx := context.Background()
			if h, ok := handler.(*defaultSagaEventHandler); ok {
				if err := h.Start(ctx); err != nil {
					t.Fatalf("failed to start handler: %v", err)
				}
				defer h.Stop(ctx)
			}

			// Process the event - this will internally call OnEventProcessed if successful
			err = handler.HandleSagaEvent(ctx, tt.event, tt.handlerCtx)

			if tt.expectError {
				if err == nil {
					t.Error("expected processing error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected processing error: %v", err)
				}
			}
		})
	}
}
