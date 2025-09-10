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
	"sync"
	"time"
)

// ContextKey is a typed key for context values to prevent collisions.
type ContextKey string

// Common context keys used throughout the messaging system.
const (
	// ContextKeyMessage stores the current message being processed
	ContextKeyMessage ContextKey = "messaging.message"

	// ContextKeyTraceID stores the trace ID for distributed tracing
	ContextKeyTraceID ContextKey = "messaging.trace_id"

	// ContextKeyCorrelationID stores the correlation ID for request correlation
	ContextKeyCorrelationID ContextKey = "messaging.correlation_id"

	// ContextKeySubscriberID stores the subscriber ID
	ContextKeySubscriberID ContextKey = "messaging.subscriber_id"

	// ContextKeyHandlerID stores the handler ID
	ContextKeyHandlerID ContextKey = "messaging.handler_id"

	// ContextKeyProcessingStart stores when message processing started
	ContextKeyProcessingStart ContextKey = "messaging.processing_start"

	// ContextKeyRequestID stores the request ID
	ContextKeyRequestID ContextKey = "messaging.request_id"

	// ContextKeyUserID stores the user ID if available
	ContextKeyUserID ContextKey = "messaging.user_id"

	// ContextKeyTenantID stores the tenant ID for multi-tenant applications
	ContextKeyTenantID ContextKey = "messaging.tenant_id"

	// ContextKeyMiddlewareChain stores the middleware chain information
	ContextKeyMiddlewareChain ContextKey = "messaging.middleware_chain"
)

// MessageContext provides context management and propagation for message processing.
// It extends Go's context.Context with messaging-specific functionality and helpers.
type MessageContext interface {
	context.Context

	// GetMessage returns the current message being processed
	GetMessage() *Message

	// GetTraceID returns the trace ID for distributed tracing
	GetTraceID() string

	// GetCorrelationID returns the correlation ID for request correlation
	GetCorrelationID() string

	// GetRequestID returns the request ID
	GetRequestID() string

	// GetUserID returns the user ID if available
	GetUserID() string

	// GetTenantID returns the tenant ID for multi-tenant applications
	GetTenantID() string

	// GetProcessingStart returns when message processing started
	GetProcessingStart() time.Time

	// WithValue creates a new context with the given key-value pair
	WithValue(key ContextKey, value interface{}) MessageContext

	// Clone creates a new MessageContext with the same values
	Clone() MessageContext
}

// messageContext implements MessageContext interface.
type messageContext struct {
	context.Context
	values map[ContextKey]interface{}
	mutex  sync.RWMutex
}

// NewMessageContext creates a new MessageContext from a base context and message.
func NewMessageContext(ctx context.Context, message *Message) MessageContext {
	if ctx == nil {
		ctx = context.Background()
	}

	msgCtx := &messageContext{
		Context: ctx,
		values:  make(map[ContextKey]interface{}),
	}

	// Set default values from message
	if message != nil {
		msgCtx.values[ContextKeyMessage] = message
		msgCtx.values[ContextKeyProcessingStart] = time.Now()

		if message.CorrelationID != "" {
			msgCtx.values[ContextKeyCorrelationID] = message.CorrelationID
		}

		// Extract additional IDs from message headers if available
		if message.Headers != nil {
			if traceID, ok := message.Headers["trace_id"]; ok {
				msgCtx.values[ContextKeyTraceID] = traceID
			}

			if requestID, ok := message.Headers["request_id"]; ok {
				msgCtx.values[ContextKeyRequestID] = requestID
			}

			if userID, ok := message.Headers["user_id"]; ok {
				msgCtx.values[ContextKeyUserID] = userID
			}

			if tenantID, ok := message.Headers["tenant_id"]; ok {
				msgCtx.values[ContextKeyTenantID] = tenantID
			}
		}
	} else {
		msgCtx.values[ContextKeyProcessingStart] = time.Now()
	}

	return msgCtx
}

// GetMessage returns the current message being processed.
func (mc *messageContext) GetMessage() *Message {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if msg, ok := mc.values[ContextKeyMessage]; ok {
		if message, ok := msg.(*Message); ok {
			return message
		}
	}
	return nil
}

// GetTraceID returns the trace ID for distributed tracing.
func (mc *messageContext) GetTraceID() string {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if traceID, ok := mc.values[ContextKeyTraceID]; ok {
		if traceIDStr, ok := traceID.(string); ok {
			return traceIDStr
		}
	}
	return ""
}

// GetCorrelationID returns the correlation ID for request correlation.
func (mc *messageContext) GetCorrelationID() string {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if correlationID, ok := mc.values[ContextKeyCorrelationID]; ok {
		if correlationIDStr, ok := correlationID.(string); ok {
			return correlationIDStr
		}
	}
	return ""
}

// GetRequestID returns the request ID.
func (mc *messageContext) GetRequestID() string {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if requestID, ok := mc.values[ContextKeyRequestID]; ok {
		if requestIDStr, ok := requestID.(string); ok {
			return requestIDStr
		}
	}
	return ""
}

// GetUserID returns the user ID if available.
func (mc *messageContext) GetUserID() string {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if userID, ok := mc.values[ContextKeyUserID]; ok {
		if userIDStr, ok := userID.(string); ok {
			return userIDStr
		}
	}
	return ""
}

// GetTenantID returns the tenant ID for multi-tenant applications.
func (mc *messageContext) GetTenantID() string {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if tenantID, ok := mc.values[ContextKeyTenantID]; ok {
		if tenantIDStr, ok := tenantID.(string); ok {
			return tenantIDStr
		}
	}
	return ""
}

// GetProcessingStart returns when message processing started.
func (mc *messageContext) GetProcessingStart() time.Time {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if start, ok := mc.values[ContextKeyProcessingStart]; ok {
		if startTime, ok := start.(time.Time); ok {
			return startTime
		}
	}
	return time.Time{}
}

// WithValue creates a new context with the given key-value pair.
func (mc *messageContext) WithValue(key ContextKey, value interface{}) MessageContext {
	newCtx := &messageContext{
		Context: mc.Context,
		values:  make(map[ContextKey]interface{}),
	}

	// Copy existing values
	mc.mutex.RLock()
	for k, v := range mc.values {
		newCtx.values[k] = v
	}
	mc.mutex.RUnlock()

	// Set new value
	newCtx.values[key] = value

	return newCtx
}

// Clone creates a new MessageContext with the same values.
func (mc *messageContext) Clone() MessageContext {
	newCtx := &messageContext{
		Context: mc.Context,
		values:  make(map[ContextKey]interface{}),
	}

	// Copy all values
	mc.mutex.RLock()
	for k, v := range mc.values {
		newCtx.values[k] = v
	}
	mc.mutex.RUnlock()

	return newCtx
}

// Value returns the value associated with the key, checking both message context values
// and the underlying context.Context values.
func (mc *messageContext) Value(key interface{}) interface{} {
	// Check if it's a ContextKey in our values
	if contextKey, ok := key.(ContextKey); ok {
		mc.mutex.RLock()
		defer mc.mutex.RUnlock()

		if value, exists := mc.values[contextKey]; exists {
			return value
		}
	}

	// Fall back to underlying context
	return mc.Context.Value(key)
}

// MiddlewarePipeline represents the execution pipeline for message processing middleware.
type MiddlewarePipeline struct {
	name        string
	chain       *MiddlewareChain
	config      PipelineConfig
	metrics     *PipelineMetrics
	metricsMux  sync.RWMutex
	errorPolicy ErrorPolicy
}

// PipelineConfig defines configuration for middleware pipeline execution.
type PipelineConfig struct {
	// Name of the pipeline for identification
	Name string `json:"name"`

	// MaxConcurrency for pipeline execution (0 = unlimited)
	MaxConcurrency int `json:"max_concurrency" default:"0"`

	// Timeout for entire pipeline execution
	Timeout time.Duration `json:"timeout" default:"30s"`

	// ErrorPolicy determines how pipeline errors are handled
	ErrorPolicy ErrorPolicy `json:"error_policy" default:"stop_on_first_error"`

	// ShortCircuit enables short-circuiting on specific conditions
	ShortCircuit bool `json:"short_circuit" default:"false"`

	// ContextPropagation enables context propagation between middleware
	ContextPropagation bool `json:"context_propagation" default:"true"`
}

// ErrorPolicy defines how pipeline errors should be handled.
type ErrorPolicy string

const (
	// ErrorPolicyStopOnFirst stops pipeline execution on the first error
	ErrorPolicyStopOnFirst ErrorPolicy = "stop_on_first_error"

	// ErrorPolicyCollectAll continues execution and collects all errors
	ErrorPolicyCollectAll ErrorPolicy = "collect_all_errors"

	// ErrorPolicyIgnoreErrors continues execution ignoring all errors
	ErrorPolicyIgnoreErrors ErrorPolicy = "ignore_errors"
)

// PipelineMetrics tracks metrics for pipeline execution.
type PipelineMetrics struct {
	// ExecutionCount tracks number of pipeline executions
	ExecutionCount int64 `json:"execution_count"`

	// ErrorCount tracks number of pipeline errors
	ErrorCount int64 `json:"error_count"`

	// SuccessCount tracks number of successful executions
	SuccessCount int64 `json:"success_count"`

	// TotalDuration tracks total time spent in pipeline execution
	TotalDuration time.Duration `json:"total_duration"`

	// MinDuration tracks minimum execution time
	MinDuration time.Duration `json:"min_duration"`

	// MaxDuration tracks maximum execution time
	MaxDuration time.Duration `json:"max_duration"`

	// AvgDuration tracks average execution time
	AvgDuration time.Duration `json:"avg_duration"`

	// LastExecution tracks when pipeline was last executed
	LastExecution time.Time `json:"last_execution"`

	// MiddlewareMetrics tracks per-middleware metrics
	MiddlewareMetrics map[string]*MiddlewareExecutionMetrics `json:"middleware_metrics"`
}

// MiddlewareExecutionMetrics tracks execution metrics for individual middleware.
type MiddlewareExecutionMetrics struct {
	// Name of the middleware
	Name string `json:"name"`

	// ExecutionCount for this middleware
	ExecutionCount int64 `json:"execution_count"`

	// ErrorCount for this middleware
	ErrorCount int64 `json:"error_count"`

	// TotalDuration spent in this middleware
	TotalDuration time.Duration `json:"total_duration"`

	// AvgDuration for this middleware
	AvgDuration time.Duration `json:"avg_duration"`
}

// NewMiddlewarePipeline creates a new middleware pipeline with the given configuration.
func NewMiddlewarePipeline(config PipelineConfig) *MiddlewarePipeline {
	if config.Name == "" {
		config.Name = "default-pipeline"
	}

	return &MiddlewarePipeline{
		name:        config.Name,
		chain:       NewMiddlewareChain(),
		config:      config,
		metrics:     &PipelineMetrics{MiddlewareMetrics: make(map[string]*MiddlewareExecutionMetrics)},
		errorPolicy: config.ErrorPolicy,
	}
}

// AddMiddleware adds middleware to the pipeline.
func (mp *MiddlewarePipeline) AddMiddleware(middleware ...Middleware) {
	mp.chain.Add(middleware...)

	// Initialize metrics for new middleware
	mp.metricsMux.Lock()
	for _, mw := range middleware {
		if _, exists := mp.metrics.MiddlewareMetrics[mw.Name()]; !exists {
			mp.metrics.MiddlewareMetrics[mw.Name()] = &MiddlewareExecutionMetrics{
				Name: mw.Name(),
			}
		}
	}
	mp.metricsMux.Unlock()
}

// Execute executes the middleware pipeline with the given handler and message.
func (mp *MiddlewarePipeline) Execute(ctx context.Context, handler MessageHandler, message *Message) error {
	start := time.Now()

	// Create MessageContext if not already present
	var msgCtx MessageContext
	if mc, ok := ctx.(MessageContext); ok {
		msgCtx = mc
	} else {
		msgCtx = NewMessageContext(ctx, message)
	}

	// Set pipeline information in context
	msgCtx = msgCtx.WithValue(ContextKeyMiddlewareChain, mp.name)

	// Apply timeout if configured
	if mp.config.Timeout > 0 {
		timeoutCtx, cancel := context.WithTimeout(msgCtx, mp.config.Timeout)
		defer cancel()
		// Wrap timeout context back into MessageContext
		if mc, ok := msgCtx.(*messageContext); ok {
			msgCtx = &messageContext{
				Context: timeoutCtx,
				values:  mc.values,
			}
		} else {
			// Create new MessageContext if cast fails
			msgCtx = NewMessageContext(timeoutCtx, message)
			msgCtx = msgCtx.WithValue(ContextKeyMiddlewareChain, mp.name)
		}
	}

	// Build the handler chain with instrumented middleware
	instrumentedHandler := mp.buildInstrumentedChain(handler)

	// Execute the chain
	err := instrumentedHandler.Handle(msgCtx, message)

	// Update pipeline metrics
	duration := time.Since(start)
	mp.updatePipelineMetrics(duration, err)

	return err
}

// buildInstrumentedChain builds a handler chain with metrics instrumentation.
func (mp *MiddlewarePipeline) buildInstrumentedChain(handler MessageHandler) MessageHandler {
	middleware := mp.chain.GetMiddleware()

	// Instrument each middleware with metrics collection
	for i := len(middleware) - 1; i >= 0; i-- {
		mw := middleware[i]
		handler = mp.instrumentMiddleware(mw, handler)
	}

	return handler
}

// instrumentMiddleware wraps middleware with metrics collection.
func (mp *MiddlewarePipeline) instrumentMiddleware(middleware Middleware, next MessageHandler) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		start := time.Now()

		// Execute middleware
		wrappedHandler := middleware.Wrap(next)
		err := wrappedHandler.Handle(ctx, message)

		// Update middleware metrics
		duration := time.Since(start)
		mp.updateMiddlewareMetrics(middleware.Name(), duration, err)

		return err
	})
}

// updatePipelineMetrics updates pipeline-level metrics.
func (mp *MiddlewarePipeline) updatePipelineMetrics(duration time.Duration, err error) {
	mp.metricsMux.Lock()
	defer mp.metricsMux.Unlock()

	mp.metrics.ExecutionCount++
	mp.metrics.TotalDuration += duration
	mp.metrics.LastExecution = time.Now()

	if err != nil {
		mp.metrics.ErrorCount++
	} else {
		mp.metrics.SuccessCount++
	}

	// Update min/max duration
	if mp.metrics.MinDuration == 0 || duration < mp.metrics.MinDuration {
		mp.metrics.MinDuration = duration
	}
	if duration > mp.metrics.MaxDuration {
		mp.metrics.MaxDuration = duration
	}

	// Update average duration
	if mp.metrics.ExecutionCount > 0 {
		mp.metrics.AvgDuration = time.Duration(int64(mp.metrics.TotalDuration) / mp.metrics.ExecutionCount)
	}
}

// updateMiddlewareMetrics updates metrics for specific middleware.
func (mp *MiddlewarePipeline) updateMiddlewareMetrics(name string, duration time.Duration, err error) {
	mp.metricsMux.Lock()
	defer mp.metricsMux.Unlock()

	metrics, exists := mp.metrics.MiddlewareMetrics[name]
	if !exists {
		metrics = &MiddlewareExecutionMetrics{Name: name}
		mp.metrics.MiddlewareMetrics[name] = metrics
	}

	metrics.ExecutionCount++
	metrics.TotalDuration += duration

	if err != nil {
		metrics.ErrorCount++
	}

	// Update average duration
	if metrics.ExecutionCount > 0 {
		metrics.AvgDuration = time.Duration(int64(metrics.TotalDuration) / metrics.ExecutionCount)
	}
}

// GetMetrics returns current pipeline metrics.
func (mp *MiddlewarePipeline) GetMetrics() *PipelineMetrics {
	mp.metricsMux.RLock()
	defer mp.metricsMux.RUnlock()

	// Create a deep copy to prevent external modifications
	metrics := &PipelineMetrics{
		ExecutionCount:    mp.metrics.ExecutionCount,
		ErrorCount:        mp.metrics.ErrorCount,
		SuccessCount:      mp.metrics.SuccessCount,
		TotalDuration:     mp.metrics.TotalDuration,
		MinDuration:       mp.metrics.MinDuration,
		MaxDuration:       mp.metrics.MaxDuration,
		AvgDuration:       mp.metrics.AvgDuration,
		LastExecution:     mp.metrics.LastExecution,
		MiddlewareMetrics: make(map[string]*MiddlewareExecutionMetrics),
	}

	for name, mwMetrics := range mp.metrics.MiddlewareMetrics {
		metrics.MiddlewareMetrics[name] = &MiddlewareExecutionMetrics{
			Name:           mwMetrics.Name,
			ExecutionCount: mwMetrics.ExecutionCount,
			ErrorCount:     mwMetrics.ErrorCount,
			TotalDuration:  mwMetrics.TotalDuration,
			AvgDuration:    mwMetrics.AvgDuration,
		}
	}

	return metrics
}

// GetName returns the pipeline name.
func (mp *MiddlewarePipeline) GetName() string {
	return mp.name
}

// GetConfig returns the pipeline configuration.
func (mp *MiddlewarePipeline) GetConfig() PipelineConfig {
	return mp.config
}

// Reset resets pipeline metrics.
func (mp *MiddlewarePipeline) Reset() {
	mp.metricsMux.Lock()
	defer mp.metricsMux.Unlock()

	mp.metrics = &PipelineMetrics{
		MiddlewareMetrics: make(map[string]*MiddlewareExecutionMetrics),
	}

	// Re-initialize metrics for existing middleware
	middleware := mp.chain.GetMiddleware()
	for _, mw := range middleware {
		mp.metrics.MiddlewareMetrics[mw.Name()] = &MiddlewareExecutionMetrics{
			Name: mw.Name(),
		}
	}
}

// PipelineManager manages multiple middleware pipelines.
type PipelineManager struct {
	pipelines map[string]*MiddlewarePipeline
	mutex     sync.RWMutex
}

// NewPipelineManager creates a new pipeline manager.
func NewPipelineManager() *PipelineManager {
	return &PipelineManager{
		pipelines: make(map[string]*MiddlewarePipeline),
	}
}

// RegisterPipeline registers a pipeline with the manager.
func (pm *PipelineManager) RegisterPipeline(pipeline *MiddlewarePipeline) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	name := pipeline.GetName()
	if _, exists := pm.pipelines[name]; exists {
		return NewConfigError("pipeline already registered: "+name, nil)
	}

	pm.pipelines[name] = pipeline
	return nil
}

// GetPipeline retrieves a pipeline by name.
func (pm *PipelineManager) GetPipeline(name string) (*MiddlewarePipeline, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	pipeline, exists := pm.pipelines[name]
	return pipeline, exists
}

// RemovePipeline removes a pipeline from the manager.
func (pm *PipelineManager) RemovePipeline(name string) bool {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.pipelines[name]; exists {
		delete(pm.pipelines, name)
		return true
	}
	return false
}

// GetAllPipelines returns all registered pipelines.
func (pm *PipelineManager) GetAllPipelines() map[string]*MiddlewarePipeline {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	result := make(map[string]*MiddlewarePipeline)
	for name, pipeline := range pm.pipelines {
		result[name] = pipeline
	}
	return result
}

// ExecutePipeline executes a pipeline by name with the given handler and message.
func (pm *PipelineManager) ExecutePipeline(ctx context.Context, pipelineName string, handler MessageHandler, message *Message) error {
	pipeline, exists := pm.GetPipeline(pipelineName)
	if !exists {
		return NewConfigError("pipeline not found: "+pipelineName, nil)
	}

	return pipeline.Execute(ctx, handler, message)
}
