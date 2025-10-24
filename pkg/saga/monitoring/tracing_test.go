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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/innovationmech/swit/pkg/tracing"
)

func TestNewSagaTracingManager(t *testing.T) {
	tests := []struct {
		name     string
		config   *SagaTracingConfig
		wantNil  bool
		expected bool
	}{
		{
			name:     "nil config creates disabled manager",
			config:   nil,
			wantNil:  false,
			expected: false,
		},
		{
			name: "config with enabled=true and valid manager",
			config: &SagaTracingConfig{
				Enabled:        true,
				TracingManager: tracing.NewTracingManager(),
			},
			wantNil:  false,
			expected: true,
		},
		{
			name: "config with enabled=false",
			config: &SagaTracingConfig{
				Enabled:        false,
				TracingManager: tracing.NewTracingManager(),
			},
			wantNil:  false,
			expected: false,
		},
		{
			name: "config with enabled=true but nil manager",
			config: &SagaTracingConfig{
				Enabled:        true,
				TracingManager: nil,
			},
			wantNil:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewSagaTracingManager(tt.config)
			require.NotNil(t, manager, "manager should never be nil")
			assert.Equal(t, tt.expected, manager.IsEnabled(), "enabled state mismatch")
		})
	}
}

func TestSagaTracingManager_StartSagaSpan(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		sagaID     string
		sagaType   string
		wantNilCtx bool
	}{
		{
			name:       "enabled manager creates span",
			enabled:    true,
			sagaID:     "saga-123",
			sagaType:   "OrderSaga",
			wantNilCtx: false,
		},
		{
			name:       "disabled manager returns no-op span",
			enabled:    false,
			sagaID:     "saga-456",
			sagaType:   "PaymentSaga",
			wantNilCtx: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := createTestManager(t, tt.enabled)
			ctx := context.Background()

			newCtx, span := manager.StartSagaSpan(ctx, tt.sagaID, tt.sagaType)
			require.NotNil(t, newCtx, "context should not be nil")
			require.NotNil(t, span, "span should not be nil")

			span.End()
		})
	}
}

func TestSagaTracingManager_StartStepSpan(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		sagaID   string
		stepName string
		stepType string
	}{
		{
			name:     "enabled manager creates step span",
			enabled:  true,
			sagaID:   "saga-123",
			stepName: "CreateOrder",
			stepType: "forward",
		},
		{
			name:     "disabled manager returns no-op span",
			enabled:  false,
			sagaID:   "saga-456",
			stepName: "CancelOrder",
			stepType: "compensate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := createTestManager(t, tt.enabled)
			ctx := context.Background()

			// Start parent saga span first
			ctx, sagaSpan := manager.StartSagaSpan(ctx, tt.sagaID, "TestSaga")
			defer sagaSpan.End()

			newCtx, span := manager.StartStepSpan(ctx, tt.sagaID, tt.stepName, tt.stepType)
			require.NotNil(t, newCtx, "context should not be nil")
			require.NotNil(t, span, "span should not be nil")

			span.End()
		})
	}
}

func TestSagaTracingManager_RecordSagaEvent(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		eventName  string
		attributes map[string]interface{}
	}{
		{
			name:      "record event with string attributes",
			enabled:   true,
			eventName: "saga.started",
			attributes: map[string]interface{}{
				"saga.initial_state": "pending",
				"saga.priority":      "high",
			},
		},
		{
			name:      "record event with mixed attributes",
			enabled:   true,
			eventName: "saga.step.executed",
			attributes: map[string]interface{}{
				"step.name":     "CreateOrder",
				"step.duration": int64(1500),
				"step.success":  true,
				"step.retries":  3,
			},
		},
		{
			name:       "disabled manager does nothing",
			enabled:    false,
			eventName:  "saga.completed",
			attributes: map[string]interface{}{"result": "success"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := createTestManager(t, tt.enabled)
			ctx := context.Background()

			ctx, span := manager.StartSagaSpan(ctx, "saga-123", "TestSaga")
			defer span.End()

			// Should not panic
			manager.RecordSagaEvent(ctx, tt.eventName, tt.attributes)
		})
	}
}

func TestSagaTracingManager_RecordSagaSuccess(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		sagaID  string
	}{
		{
			name:    "record success for enabled manager",
			enabled: true,
			sagaID:  "saga-success-123",
		},
		{
			name:    "record success for disabled manager",
			enabled: false,
			sagaID:  "saga-success-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := createTestManager(t, tt.enabled)
			ctx := context.Background()

			ctx, span := manager.StartSagaSpan(ctx, tt.sagaID, "TestSaga")
			defer span.End()

			// Should not panic
			manager.RecordSagaSuccess(ctx, tt.sagaID)
		})
	}
}

func TestSagaTracingManager_RecordSagaFailure(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		sagaID  string
		err     error
		reason  string
	}{
		{
			name:    "record failure with error",
			enabled: true,
			sagaID:  "saga-fail-123",
			err:     errors.New("payment processing failed"),
			reason:  "insufficient funds",
		},
		{
			name:    "record failure without error",
			enabled: true,
			sagaID:  "saga-fail-456",
			err:     nil,
			reason:  "timeout waiting for response",
		},
		{
			name:    "record failure for disabled manager",
			enabled: false,
			sagaID:  "saga-fail-789",
			err:     errors.New("some error"),
			reason:  "some reason",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := createTestManager(t, tt.enabled)
			ctx := context.Background()

			ctx, span := manager.StartSagaSpan(ctx, tt.sagaID, "TestSaga")
			defer span.End()

			// Should not panic
			manager.RecordSagaFailure(ctx, tt.sagaID, tt.err, tt.reason)
		})
	}
}

func TestSagaTracingManager_RecordSagaCompensated(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		sagaID  string
		reason  string
	}{
		{
			name:    "record compensation for enabled manager",
			enabled: true,
			sagaID:  "saga-comp-123",
			reason:  "payment failed, inventory released",
		},
		{
			name:    "record compensation for disabled manager",
			enabled: false,
			sagaID:  "saga-comp-456",
			reason:  "order cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := createTestManager(t, tt.enabled)
			ctx := context.Background()

			ctx, span := manager.StartSagaSpan(ctx, tt.sagaID, "TestSaga")
			defer span.End()

			// Should not panic
			manager.RecordSagaCompensated(ctx, tt.sagaID, tt.reason)
		})
	}
}

func TestSagaTracingManager_RecordStepSuccess(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		stepName string
	}{
		{
			name:     "record step success for enabled manager",
			enabled:  true,
			stepName: "CreateOrder",
		},
		{
			name:     "record step success for disabled manager",
			enabled:  false,
			stepName: "ProcessPayment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := createTestManager(t, tt.enabled)
			ctx := context.Background()

			ctx, sagaSpan := manager.StartSagaSpan(ctx, "saga-123", "TestSaga")
			defer sagaSpan.End()

			ctx, stepSpan := manager.StartStepSpan(ctx, "saga-123", tt.stepName, "forward")
			defer stepSpan.End()

			// Should not panic
			manager.RecordStepSuccess(ctx, tt.stepName)
		})
	}
}

func TestSagaTracingManager_RecordStepFailure(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		stepName string
		err      error
		reason   string
	}{
		{
			name:     "record step failure with error",
			enabled:  true,
			stepName: "ProcessPayment",
			err:      errors.New("payment gateway error"),
			reason:   "gateway timeout",
		},
		{
			name:     "record step failure without error",
			enabled:  true,
			stepName: "ReserveInventory",
			err:      nil,
			reason:   "out of stock",
		},
		{
			name:     "record step failure for disabled manager",
			enabled:  false,
			stepName: "CreateOrder",
			err:      errors.New("some error"),
			reason:   "some reason",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := createTestManager(t, tt.enabled)
			ctx := context.Background()

			ctx, sagaSpan := manager.StartSagaSpan(ctx, "saga-123", "TestSaga")
			defer sagaSpan.End()

			ctx, stepSpan := manager.StartStepSpan(ctx, "saga-123", tt.stepName, "forward")
			defer stepSpan.End()

			// Should not panic
			manager.RecordStepFailure(ctx, tt.stepName, tt.err, tt.reason)
		})
	}
}

func TestSagaTracingManager_InjectExtractContext(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		sagaID  string
	}{
		{
			name:    "inject and extract with enabled manager",
			enabled: true,
			sagaID:  "saga-inject-123",
		},
		{
			name:    "inject and extract with disabled manager",
			enabled: false,
			sagaID:  "saga-inject-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := createTestManager(t, tt.enabled)
			ctx := context.Background()

			ctx, span := manager.StartSagaSpan(ctx, tt.sagaID, "TestSaga")
			defer span.End()

			// Inject context into carrier
			carrier := make(map[string]string)
			manager.InjectContext(ctx, carrier)

			// For enabled manager, should have traceparent
			if tt.enabled {
				assert.Contains(t, carrier, "traceparent", "should inject traceparent header")
			}

			// Extract context from carrier
			newCtx := manager.ExtractContext(context.Background(), carrier)
			require.NotNil(t, newCtx, "extracted context should not be nil")
		})
	}
}

func TestSagaTracingManager_CompleteWorkflow(t *testing.T) {
	// This test simulates a complete Saga workflow with tracing
	manager := createTestManager(t, true)
	ctx := context.Background()

	sagaID := "saga-workflow-123"
	sagaType := "OrderProcessingSaga"

	// Start Saga
	ctx, sagaSpan := manager.StartSagaSpan(ctx, sagaID, sagaType)
	defer sagaSpan.End()

	manager.RecordSagaEvent(ctx, "saga.started", map[string]interface{}{
		"saga.order_id": "order-123",
		"saga.user_id":  "user-456",
	})

	// Execute steps
	steps := []struct {
		name     string
		stepType string
		success  bool
	}{
		{"ValidateOrder", "forward", true},
		{"ReserveInventory", "forward", true},
		{"ProcessPayment", "forward", true},
		{"CreateShipment", "forward", true},
	}

	for _, step := range steps {
		stepCtx, stepSpan := manager.StartStepSpan(ctx, sagaID, step.name, step.stepType)

		if step.success {
			manager.RecordStepSuccess(stepCtx, step.name)
		} else {
			manager.RecordStepFailure(stepCtx, step.name, errors.New("step failed"), "test failure")
		}

		stepSpan.End()
	}

	// Complete Saga
	manager.RecordSagaSuccess(ctx, sagaID)

	// Test context propagation
	carrier := make(map[string]string)
	manager.InjectContext(ctx, carrier)

	// Should not panic and should complete successfully
	assert.True(t, manager.IsEnabled())
}

func TestSagaTracingManager_CompensationWorkflow(t *testing.T) {
	// This test simulates a Saga compensation workflow
	manager := createTestManager(t, true)
	ctx := context.Background()

	sagaID := "saga-compensation-123"
	sagaType := "OrderProcessingSaga"

	// Start Saga
	ctx, sagaSpan := manager.StartSagaSpan(ctx, sagaID, sagaType)
	defer sagaSpan.End()

	// Execute forward steps
	forwardSteps := []string{"ValidateOrder", "ReserveInventory", "ProcessPayment"}
	for _, stepName := range forwardSteps {
		stepCtx, stepSpan := manager.StartStepSpan(ctx, sagaID, stepName, "forward")
		manager.RecordStepSuccess(stepCtx, stepName)
		stepSpan.End()
	}

	// Simulate failure
	stepCtx, stepSpan := manager.StartStepSpan(ctx, sagaID, "CreateShipment", "forward")
	manager.RecordStepFailure(stepCtx, "CreateShipment", errors.New("shipment service unavailable"), "service timeout")
	stepSpan.End()

	// Execute compensation steps
	compensationSteps := []string{"ReleasePayment", "ReleaseInventory", "CancelOrder"}
	for _, stepName := range compensationSteps {
		compCtx, compSpan := manager.StartStepSpan(ctx, sagaID, stepName, "compensate")
		manager.RecordStepSuccess(compCtx, stepName)
		compSpan.End()
	}

	// Mark Saga as compensated
	manager.RecordSagaCompensated(ctx, sagaID, "shipment service unavailable")

	// Should not panic and should complete successfully
	assert.True(t, manager.IsEnabled())
}

func TestConvertToAttribute(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"string value", "key1", "value1"},
		{"int value", "key2", 123},
		{"int64 value", "key3", int64(456)},
		{"float64 value", "key4", 3.14},
		{"bool value", "key5", true},
		{"unsupported type", "key6", struct{ Name string }{"test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := convertToAttribute(tt.key, tt.value)
			assert.Equal(t, tt.key, string(attr.Key))
		})
	}
}

// createTestManager creates a test SagaTracingManager for testing.
func createTestManager(t *testing.T, enabled bool) *SagaTracingManager {
	if !enabled {
		return NewSagaTracingManager(nil)
	}

	// Create a tracing manager with console exporter for testing
	tm := tracing.NewTracingManager()
	config := &tracing.TracingConfig{
		Enabled:     true,
		ServiceName: "saga-test-service",
		Sampling: tracing.SamplingConfig{
			Type: "always_on",
			Rate: 1.0,
		},
		Exporter: tracing.ExporterConfig{
			Type: "console",
		},
		Propagators: []string{"tracecontext", "baggage"},
	}

	err := tm.Initialize(context.Background(), config)
	require.NoError(t, err, "failed to initialize tracing manager")

	return NewSagaTracingManager(&SagaTracingConfig{
		Enabled:        true,
		TracingManager: tm,
	})
}

// TestNoOpSpan tests the noOpSpan implementation.
func TestNoOpSpan(t *testing.T) {
	span := &noOpSpan{}

	// Test all methods to ensure they don't panic
	t.Run("SetAttribute", func(t *testing.T) {
		assert.NotPanics(t, func() {
			span.SetAttribute("key", "value")
		})
	})

	t.Run("SetAttributes", func(t *testing.T) {
		assert.NotPanics(t, func() {
			span.SetAttributes()
		})
	})

	t.Run("AddEvent", func(t *testing.T) {
		assert.NotPanics(t, func() {
			span.AddEvent("test-event")
		})
	})

	t.Run("SetStatus", func(t *testing.T) {
		assert.NotPanics(t, func() {
			span.SetStatus(0, "test")
		})
	})

	t.Run("End", func(t *testing.T) {
		assert.NotPanics(t, func() {
			span.End()
		})
	})

	t.Run("RecordError", func(t *testing.T) {
		assert.NotPanics(t, func() {
			span.RecordError(errors.New("test error"))
		})
	})

	t.Run("SpanContext", func(t *testing.T) {
		ctx := span.SpanContext()
		assert.False(t, ctx.IsValid(), "noOpSpan should return invalid span context")
	})
}

// TestNoOpSpan_WithVariousAttributes tests noOpSpan with different attribute types.
func TestNoOpSpan_WithVariousAttributes(t *testing.T) {
	span := &noOpSpan{}

	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"string", "str_key", "string_value"},
		{"int", "int_key", 42},
		{"int64", "int64_key", int64(123456789)},
		{"float64", "float_key", 3.14159},
		{"bool", "bool_key", true},
		{"nil", "nil_key", nil},
		{"complex", "complex_key", map[string]interface{}{"nested": "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				span.SetAttribute(tt.key, tt.value)
			}, "SetAttribute should not panic for %s type", tt.name)
		})
	}
}

// TestNoOpSpan_ConcurrentAccess tests that noOpSpan is safe for concurrent use.
func TestNoOpSpan_ConcurrentAccess(t *testing.T) {
	span := &noOpSpan{}
	done := make(chan bool)

	// Spawn multiple goroutines calling different methods
	for i := 0; i < 10; i++ {
		go func(id int) {
			span.SetAttribute("key", id)
			span.AddEvent("event")
			span.SetStatus(0, "status")
			span.End()
			_ = span.SpanContext()
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
