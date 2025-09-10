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

func TestMessageContext_Creation(t *testing.T) {
	t.Run("NewMessageContext with message", func(t *testing.T) {
		message := &Message{
			ID:            "test-message-id",
			Topic:         "test-topic",
			CorrelationID: "correlation-123",
			Headers: map[string]string{
				"trace_id":   "trace-abc-123",
				"request_id": "request-456",
				"user_id":    "user-789",
				"tenant_id":  "tenant-101",
			},
		}

		msgCtx := NewMessageContext(context.Background(), message)

		assert.Equal(t, message, msgCtx.GetMessage())
		assert.Equal(t, "correlation-123", msgCtx.GetCorrelationID())
		assert.Equal(t, "trace-abc-123", msgCtx.GetTraceID())
		assert.Equal(t, "request-456", msgCtx.GetRequestID())
		assert.Equal(t, "user-789", msgCtx.GetUserID())
		assert.Equal(t, "tenant-101", msgCtx.GetTenantID())
		assert.False(t, msgCtx.GetProcessingStart().IsZero())
	})

	t.Run("NewMessageContext with nil message", func(t *testing.T) {
		msgCtx := NewMessageContext(context.Background(), nil)

		assert.Nil(t, msgCtx.GetMessage())
		assert.Empty(t, msgCtx.GetCorrelationID())
		assert.Empty(t, msgCtx.GetTraceID())
		assert.Empty(t, msgCtx.GetRequestID())
		assert.Empty(t, msgCtx.GetUserID())
		assert.Empty(t, msgCtx.GetTenantID())
		assert.False(t, msgCtx.GetProcessingStart().IsZero())
	})

	t.Run("NewMessageContext with nil base context", func(t *testing.T) {
		message := &Message{ID: "test-id"}
		msgCtx := NewMessageContext(context.TODO(), message)

		assert.NotNil(t, msgCtx)
		assert.Equal(t, message, msgCtx.GetMessage())
	})
}

func TestMessageContext_WithValue(t *testing.T) {
	message := &Message{ID: "test-id"}
	msgCtx := NewMessageContext(context.Background(), message)

	newCtx := msgCtx.WithValue(ContextKeySubscriberID, "subscriber-123")

	// Original context should not be modified
	assert.Empty(t, msgCtx.Value(ContextKeySubscriberID))

	// New context should have the value
	assert.Equal(t, "subscriber-123", newCtx.Value(ContextKeySubscriberID))
	assert.Equal(t, message, newCtx.GetMessage())
}

func TestMessageContext_Clone(t *testing.T) {
	message := &Message{
		ID:            "test-id",
		CorrelationID: "corr-123",
	}
	msgCtx := NewMessageContext(context.Background(), message)
	msgCtx = msgCtx.WithValue(ContextKeyHandlerID, "handler-456")

	cloned := msgCtx.Clone()

	// Should have same values
	assert.Equal(t, message, cloned.GetMessage())
	assert.Equal(t, "corr-123", cloned.GetCorrelationID())
	assert.Equal(t, "handler-456", cloned.Value(ContextKeyHandlerID))

	// Should be different instances
	assert.NotSame(t, msgCtx, cloned)

	// Modifying clone should not affect original
	newCloned := cloned.WithValue(ContextKeyUserID, "user-123")
	assert.Empty(t, msgCtx.Value(ContextKeyUserID))
	assert.Equal(t, "user-123", newCloned.Value(ContextKeyUserID))
}

func TestMessageContext_Value(t *testing.T) {
	// Test with base context value
	baseCtx := context.WithValue(context.Background(), "base-key", "base-value")
	message := &Message{ID: "test-id"}
	msgCtx := NewMessageContext(baseCtx, message)

	// Should find base context value
	assert.Equal(t, "base-value", msgCtx.Value("base-key"))

	// Should find message context value
	assert.Equal(t, message, msgCtx.Value(ContextKeyMessage))

	// Should return nil for non-existent key
	assert.Nil(t, msgCtx.Value("non-existent"))
}

func TestMiddlewarePipeline_Creation(t *testing.T) {
	config := PipelineConfig{
		Name:               "test-pipeline",
		Timeout:            5 * time.Second,
		ErrorPolicy:        ErrorPolicyStopOnFirst,
		ContextPropagation: true,
	}

	pipeline := NewMiddlewarePipeline(config)

	assert.Equal(t, "test-pipeline", pipeline.GetName())
	assert.Equal(t, config, pipeline.GetConfig())
	assert.NotNil(t, pipeline.GetMetrics())
}

func TestMiddlewarePipeline_DefaultConfig(t *testing.T) {
	config := PipelineConfig{} // Empty config
	pipeline := NewMiddlewarePipeline(config)

	assert.Equal(t, "default-pipeline", pipeline.GetName())
}

func TestMiddlewarePipeline_AddMiddleware(t *testing.T) {
	pipeline := NewMiddlewarePipeline(PipelineConfig{Name: "test"})

	middleware1 := &MockMiddleware{name: "middleware1"}
	middleware2 := &MockMiddleware{name: "middleware2"}

	pipeline.AddMiddleware(middleware1, middleware2)

	metrics := pipeline.GetMetrics()
	assert.Contains(t, metrics.MiddlewareMetrics, "middleware1")
	assert.Contains(t, metrics.MiddlewareMetrics, "middleware2")
}

func TestMiddlewarePipeline_Execute(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		pipeline := NewMiddlewarePipeline(PipelineConfig{Name: "test"})

		executed := false
		handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
			executed = true
			// Verify context is MessageContext
			if msgCtx, ok := ctx.(MessageContext); ok {
				assert.Equal(t, message, msgCtx.GetMessage())
				assert.Equal(t, "test", msgCtx.Value(ContextKeyMiddlewareChain))
			}
			return nil
		})

		message := &Message{ID: "test-msg"}
		err := pipeline.Execute(context.Background(), handler, message)

		assert.NoError(t, err)
		assert.True(t, executed)

		metrics := pipeline.GetMetrics()
		assert.Equal(t, int64(1), metrics.ExecutionCount)
		assert.Equal(t, int64(1), metrics.SuccessCount)
		assert.Equal(t, int64(0), metrics.ErrorCount)
	})

	t.Run("execution with middleware", func(t *testing.T) {
		pipeline := NewMiddlewarePipeline(PipelineConfig{Name: "test"})

		var executionOrder []string
		middleware1 := MiddlewareFunc(func(next MessageHandler) MessageHandler {
			return MessageHandlerFunc(func(ctx context.Context, message *Message) error {
				executionOrder = append(executionOrder, "middleware1-before")
				err := next.Handle(ctx, message)
				executionOrder = append(executionOrder, "middleware1-after")
				return err
			})
		})

		middleware2 := MiddlewareFunc(func(next MessageHandler) MessageHandler {
			return MessageHandlerFunc(func(ctx context.Context, message *Message) error {
				executionOrder = append(executionOrder, "middleware2-before")
				err := next.Handle(ctx, message)
				executionOrder = append(executionOrder, "middleware2-after")
				return err
			})
		})

		pipeline.AddMiddleware(middleware1, middleware2)

		handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
			executionOrder = append(executionOrder, "handler")
			return nil
		})

		message := &Message{ID: "test-msg"}
		err := pipeline.Execute(context.Background(), handler, message)

		assert.NoError(t, err)
		expected := []string{
			"middleware1-before",
			"middleware2-before",
			"handler",
			"middleware2-after",
			"middleware1-after",
		}
		assert.Equal(t, expected, executionOrder)
	})

	t.Run("execution with timeout", func(t *testing.T) {
		pipeline := NewMiddlewarePipeline(PipelineConfig{
			Name:    "test",
			Timeout: 100 * time.Millisecond,
		})

		handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
			// Check context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return nil
			}
		})

		message := &Message{ID: "test-msg"}
		err := pipeline.Execute(context.Background(), handler, message)

		assert.Error(t, err)
		// Could be context.DeadlineExceeded or timeout error
		assert.True(t, err == context.DeadlineExceeded || err.Error() == "context deadline exceeded")

		metrics := pipeline.GetMetrics()
		assert.Equal(t, int64(1), metrics.ExecutionCount)
		assert.Equal(t, int64(0), metrics.SuccessCount)
		assert.Equal(t, int64(1), metrics.ErrorCount)
	})
}

func TestMiddlewarePipeline_Metrics(t *testing.T) {
	pipeline := NewMiddlewarePipeline(PipelineConfig{Name: "test"})

	middleware := &MockMiddleware{name: "test-middleware"}
	pipeline.AddMiddleware(middleware)

	handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	message := &Message{ID: "test-msg"}

	// Execute multiple times
	for i := 0; i < 3; i++ {
		err := pipeline.Execute(context.Background(), handler, message)
		assert.NoError(t, err)
	}

	metrics := pipeline.GetMetrics()
	assert.Equal(t, int64(3), metrics.ExecutionCount)
	assert.Equal(t, int64(3), metrics.SuccessCount)
	assert.Equal(t, int64(0), metrics.ErrorCount)
	assert.True(t, metrics.TotalDuration > 0)
	assert.True(t, metrics.MinDuration > 0)
	assert.True(t, metrics.MaxDuration > 0)
	assert.True(t, metrics.AvgDuration > 0)
	assert.False(t, metrics.LastExecution.IsZero())

	// Check middleware metrics
	mwMetrics, exists := metrics.MiddlewareMetrics["test-middleware"]
	require.True(t, exists)
	assert.Equal(t, int64(3), mwMetrics.ExecutionCount)
	assert.Equal(t, int64(0), mwMetrics.ErrorCount)
}

func TestMiddlewarePipeline_Reset(t *testing.T) {
	pipeline := NewMiddlewarePipeline(PipelineConfig{Name: "test"})

	middleware := &MockMiddleware{name: "test-middleware"}
	pipeline.AddMiddleware(middleware)

	handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		return nil
	})

	// Execute once
	err := pipeline.Execute(context.Background(), handler, &Message{ID: "test"})
	assert.NoError(t, err)

	// Verify metrics
	metrics := pipeline.GetMetrics()
	assert.Equal(t, int64(1), metrics.ExecutionCount)

	// Reset pipeline
	pipeline.Reset()

	// Verify metrics are reset
	metrics = pipeline.GetMetrics()
	assert.Equal(t, int64(0), metrics.ExecutionCount)
	assert.Equal(t, int64(0), metrics.SuccessCount)
	assert.Equal(t, int64(0), metrics.ErrorCount)

	// Middleware metrics should still exist but be reset
	mwMetrics, exists := metrics.MiddlewareMetrics["test-middleware"]
	require.True(t, exists)
	assert.Equal(t, int64(0), mwMetrics.ExecutionCount)
}

func TestPipelineManager(t *testing.T) {
	manager := NewPipelineManager()

	// Test RegisterPipeline
	pipeline1 := NewMiddlewarePipeline(PipelineConfig{Name: "pipeline1"})
	err := manager.RegisterPipeline(pipeline1)
	assert.NoError(t, err)

	// Test duplicate registration
	err = manager.RegisterPipeline(pipeline1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test GetPipeline
	retrieved, exists := manager.GetPipeline("pipeline1")
	assert.True(t, exists)
	assert.Equal(t, pipeline1, retrieved)

	// Test non-existent pipeline
	_, exists = manager.GetPipeline("non-existent")
	assert.False(t, exists)

	// Test GetAllPipelines
	pipeline2 := NewMiddlewarePipeline(PipelineConfig{Name: "pipeline2"})
	err = manager.RegisterPipeline(pipeline2)
	assert.NoError(t, err)

	all := manager.GetAllPipelines()
	assert.Len(t, all, 2)
	assert.Contains(t, all, "pipeline1")
	assert.Contains(t, all, "pipeline2")

	// Test RemovePipeline
	removed := manager.RemovePipeline("pipeline1")
	assert.True(t, removed)

	_, exists = manager.GetPipeline("pipeline1")
	assert.False(t, exists)

	// Test removing non-existent pipeline
	removed = manager.RemovePipeline("non-existent")
	assert.False(t, removed)
}

func TestPipelineManager_ExecutePipeline(t *testing.T) {
	manager := NewPipelineManager()

	pipeline := NewMiddlewarePipeline(PipelineConfig{Name: "test-pipeline"})
	err := manager.RegisterPipeline(pipeline)
	require.NoError(t, err)

	executed := false
	handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		executed = true
		return nil
	})

	message := &Message{ID: "test-msg"}

	// Test successful execution
	err = manager.ExecutePipeline(context.Background(), "test-pipeline", handler, message)
	assert.NoError(t, err)
	assert.True(t, executed)

	// Test non-existent pipeline
	err = manager.ExecutePipeline(context.Background(), "non-existent", handler, message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline not found")
}

// Note: MockMiddleware is defined in middleware_test.go

// Verify MiddlewareFunc implements Middleware interface
func TestMiddlewareFunc_Interface(t *testing.T) {
	var _ Middleware = MiddlewareFunc(nil)

	middlewareFunc := MiddlewareFunc(func(next MessageHandler) MessageHandler {
		return next
	})

	assert.Equal(t, "middleware-func", middlewareFunc.Name())

	handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		return nil
	})

	wrapped := middlewareFunc.Wrap(handler)
	assert.NotNil(t, wrapped)
}
