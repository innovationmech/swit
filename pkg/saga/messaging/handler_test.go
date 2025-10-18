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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestSagaEventType tests the SagaEventType methods
func TestSagaEventType(t *testing.T) {
	tests := []struct {
		name                   string
		eventType              SagaEventType
		expectedIsStepEvent    bool
		expectedIsCompensation bool
		expectedIsLifecycle    bool
		expectedIsRecovery     bool
		expectedStringValue    string
	}{
		{
			name:                   "step started event",
			eventType:              EventTypeStepStarted,
			expectedIsStepEvent:    true,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.step.started",
		},
		{
			name:                   "step completed event",
			eventType:              EventTypeStepCompleted,
			expectedIsStepEvent:    true,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.step.completed",
		},
		{
			name:                   "step failed event",
			eventType:              EventTypeStepFailed,
			expectedIsStepEvent:    true,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.step.failed",
		},
		{
			name:                   "step retrying event",
			eventType:              EventTypeStepRetrying,
			expectedIsStepEvent:    true,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.step.retrying",
		},
		{
			name:                   "step compensating event",
			eventType:              EventTypeStepCompensating,
			expectedIsStepEvent:    false,
			expectedIsCompensation: true,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.step.compensating",
		},
		{
			name:                   "step compensated event",
			eventType:              EventTypeStepCompensated,
			expectedIsStepEvent:    false,
			expectedIsCompensation: true,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.step.compensated",
		},
		{
			name:                   "step compensation failed event",
			eventType:              EventTypeStepCompensationFailed,
			expectedIsStepEvent:    false,
			expectedIsCompensation: true,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.step.compensation_failed",
		},
		{
			name:                   "saga started event",
			eventType:              EventTypeSagaStarted,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    true,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.started",
		},
		{
			name:                   "saga completed event",
			eventType:              EventTypeSagaCompleted,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    true,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.completed",
		},
		{
			name:                   "saga failed event",
			eventType:              EventTypeSagaFailed,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    true,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.failed",
		},
		{
			name:                   "saga cancelled event",
			eventType:              EventTypeSagaCancelled,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    true,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.cancelled",
		},
		{
			name:                   "saga timed out event",
			eventType:              EventTypeSagaTimedOut,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    true,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.timed_out",
		},
		{
			name:                   "compensation started event",
			eventType:              EventTypeCompensationStarted,
			expectedIsStepEvent:    false,
			expectedIsCompensation: true,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.compensation.started",
		},
		{
			name:                   "compensation completed event",
			eventType:              EventTypeCompensationCompleted,
			expectedIsStepEvent:    false,
			expectedIsCompensation: true,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.compensation.completed",
		},
		{
			name:                   "compensation failed event",
			eventType:              EventTypeCompensationFailed,
			expectedIsStepEvent:    false,
			expectedIsCompensation: true,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.compensation.failed",
		},
		{
			name:                   "state changed event",
			eventType:              EventTypeStateChanged,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.state.changed",
		},
		{
			name:                   "state transition failed event",
			eventType:              EventTypeStateTransitionFailed,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     false,
			expectedStringValue:    "saga.state.transition_failed",
		},
		{
			name:                   "recovery initiated event",
			eventType:              EventTypeRecoveryInitiated,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     true,
			expectedStringValue:    "saga.recovery.initiated",
		},
		{
			name:                   "recovery completed event",
			eventType:              EventTypeRecoveryCompleted,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     true,
			expectedStringValue:    "saga.recovery.completed",
		},
		{
			name:                   "recovery failed event",
			eventType:              EventTypeRecoveryFailed,
			expectedIsStepEvent:    false,
			expectedIsCompensation: false,
			expectedIsLifecycle:    false,
			expectedIsRecovery:     true,
			expectedStringValue:    "saga.recovery.failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test String()
			if got := tt.eventType.String(); got != tt.expectedStringValue {
				t.Errorf("String() = %v, want %v", got, tt.expectedStringValue)
			}

			// Test IsStepEvent()
			if got := tt.eventType.IsStepEvent(); got != tt.expectedIsStepEvent {
				t.Errorf("IsStepEvent() = %v, want %v", got, tt.expectedIsStepEvent)
			}

			// Test IsCompensationEvent()
			if got := tt.eventType.IsCompensationEvent(); got != tt.expectedIsCompensation {
				t.Errorf("IsCompensationEvent() = %v, want %v", got, tt.expectedIsCompensation)
			}

			// Test IsSagaLifecycleEvent()
			if got := tt.eventType.IsSagaLifecycleEvent(); got != tt.expectedIsLifecycle {
				t.Errorf("IsSagaLifecycleEvent() = %v, want %v", got, tt.expectedIsLifecycle)
			}

			// Test IsRecoveryEvent()
			if got := tt.eventType.IsRecoveryEvent(); got != tt.expectedIsRecovery {
				t.Errorf("IsRecoveryEvent() = %v, want %v", got, tt.expectedIsRecovery)
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
					EventTypeStepCompleted,
					EventTypeStepFailed,
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
					EventTypeStepCompleted,
					EventTypeStepFailed,
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
					EventTypeStepFailed,
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
					EventTypeStepFailed,
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
				IncludeEventTypes: []SagaEventType{EventTypeStepCompleted},
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
				IncludeEventTypes: []SagaEventType{EventTypeStepCompleted},
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
			EventTypeStepCompleted: {
				EventType:             EventTypeStepCompleted,
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
		EventType:             EventTypeStepCompleted,
		Count:                 100,
		FailureCount:          5,
		AverageProcessingTime: 50 * time.Millisecond,
	}

	// Verify metrics fields
	if metrics.EventType != EventTypeStepCompleted {
		t.Errorf("EventType = %v, want %v", metrics.EventType, EventTypeStepCompleted)
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
