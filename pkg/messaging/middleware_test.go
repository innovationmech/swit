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
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF SUCH KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package messaging

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test logger for middleware testing
type TestLogger struct {
	logs []LogEntry
	mu   sync.Mutex
}

type LogEntry struct {
	Level  string
	Msg    string
	Fields []interface{}
}

func (tl *TestLogger) Debug(msg string, fields ...interface{}) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.logs = append(tl.logs, LogEntry{Level: "DEBUG", Msg: msg, Fields: fields})
}

func (tl *TestLogger) Info(msg string, fields ...interface{}) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.logs = append(tl.logs, LogEntry{Level: "INFO", Msg: msg, Fields: fields})
}

func (tl *TestLogger) Warn(msg string, fields ...interface{}) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.logs = append(tl.logs, LogEntry{Level: "WARN", Msg: msg, Fields: fields})
}

func (tl *TestLogger) Error(msg string, fields ...interface{}) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.logs = append(tl.logs, LogEntry{Level: "ERROR", Msg: msg, Fields: fields})
}

func (tl *TestLogger) GetLogs() []LogEntry {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	result := make([]LogEntry, len(tl.logs))
	copy(result, tl.logs)
	return result
}

func (tl *TestLogger) Clear() {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.logs = tl.logs[:0]
}

// Test metrics collector for middleware testing
type TestMetricsCollector struct {
	counters   map[string]float64
	histograms map[string][]float64
	gauges     map[string]float64
	mu         sync.Mutex
}

func NewTestMetricsCollector() *TestMetricsCollector {
	return &TestMetricsCollector{
		counters:   make(map[string]float64),
		histograms: make(map[string][]float64),
		gauges:     make(map[string]float64),
	}
}

func (tmc *TestMetricsCollector) IncrementCounter(name string, labels map[string]string, value float64) {
	tmc.mu.Lock()
	defer tmc.mu.Unlock()
	key := name + "_" + tmc.labelsToKey(labels)
	tmc.counters[key] += value
}

func (tmc *TestMetricsCollector) ObserveHistogram(name string, labels map[string]string, value float64) {
	tmc.mu.Lock()
	defer tmc.mu.Unlock()
	key := name + "_" + tmc.labelsToKey(labels)
	if tmc.histograms[key] == nil {
		tmc.histograms[key] = make([]float64, 0)
	}
	tmc.histograms[key] = append(tmc.histograms[key], value)
}

func (tmc *TestMetricsCollector) SetGauge(name string, labels map[string]string, value float64) {
	tmc.mu.Lock()
	defer tmc.mu.Unlock()
	key := name + "_" + tmc.labelsToKey(labels)
	tmc.gauges[key] = value
}

func (tmc *TestMetricsCollector) labelsToKey(labels map[string]string) string {
	var keys []string
	for k := range labels {
		keys = append(keys, k)
	}
	// Sort keys for deterministic output
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s:%s", k, labels[k]))
	}
	return strings.Join(parts, ",")
}

func (tmc *TestMetricsCollector) GetCounter(name string, labels map[string]string) float64 {
	tmc.mu.Lock()
	defer tmc.mu.Unlock()
	key := name + "_" + tmc.labelsToKey(labels)
	return tmc.counters[key]
}

func (tmc *TestMetricsCollector) GetHistogramValues(name string, labels map[string]string) []float64 {
	tmc.mu.Lock()
	defer tmc.mu.Unlock()
	key := name + "_" + tmc.labelsToKey(labels)
	values := tmc.histograms[key]
	if values == nil {
		return []float64{}
	}
	result := make([]float64, len(values))
	copy(result, values)
	return result
}

func TestMiddlewareFunc(t *testing.T) {
	callCount := 0

	middlewareFunc := MiddlewareFunc(func(next MessageHandler) MessageHandler {
		return MessageHandlerFunc(func(ctx context.Context, message *Message) error {
			callCount++
			return next.Handle(ctx, message)
		})
	})

	if middlewareFunc.Name() != "middleware-func" {
		t.Errorf("Expected name 'middleware-func', got %s", middlewareFunc.Name())
	}

	baseHandler := NewMockHandler()
	wrappedHandler := middlewareFunc.Wrap(baseHandler)

	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	err := wrappedHandler.Handle(context.Background(), message)
	if err != nil {
		t.Errorf("Wrapped handler failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected middleware to be called once, got %d calls", callCount)
	}

	if baseHandler.GetCallCount() != 1 {
		t.Errorf("Expected base handler to be called once, got %d calls", baseHandler.GetCallCount())
	}
}

func TestMiddlewareChain(t *testing.T) {
	middleware1 := &MockMiddleware{name: "middleware1"}
	middleware2 := &MockMiddleware{name: "middleware2"}
	middleware3 := &MockMiddleware{name: "middleware3"}

	chain := NewMiddlewareChain(middleware1, middleware2)

	if chain.Len() != 2 {
		t.Errorf("Expected chain length 2, got %d", chain.Len())
	}

	// Test Add
	chain.Add(middleware3)
	if chain.Len() != 3 {
		t.Errorf("Expected chain length 3 after Add, got %d", chain.Len())
	}

	// Test Prepend
	middleware0 := &MockMiddleware{name: "middleware0"}
	chain.Prepend(middleware0)
	if chain.Len() != 4 {
		t.Errorf("Expected chain length 4 after Prepend, got %d", chain.Len())
	}

	// Test GetMiddleware
	middlewares := chain.GetMiddleware()
	if len(middlewares) != 4 {
		t.Errorf("Expected 4 middleware from GetMiddleware, got %d", len(middlewares))
	}

	if middlewares[0].Name() != "middleware0" {
		t.Errorf("Expected first middleware to be 'middleware0', got %s", middlewares[0].Name())
	}

	// Test Remove
	removed := chain.Remove("middleware1")
	if !removed {
		t.Error("Expected Remove to return true for existing middleware")
	}

	if chain.Len() != 3 {
		t.Errorf("Expected chain length 3 after Remove, got %d", chain.Len())
	}

	removed = chain.Remove("non-existent")
	if removed {
		t.Error("Expected Remove to return false for non-existent middleware")
	}

	// Test Build
	baseHandler := NewMockHandler()
	builtHandler := chain.Build(baseHandler)

	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	err := builtHandler.Handle(context.Background(), message)
	if err != nil {
		t.Errorf("Built handler failed: %v", err)
	}

	// Verify base handler was called
	if baseHandler.GetCallCount() != 1 {
		t.Errorf("Expected base handler to be called once, got %d calls", baseHandler.GetCallCount())
	}
}

func TestLoggingMiddleware(t *testing.T) {
	logger := &TestLogger{}
	middleware := NewLoggingMiddleware(logger, LogLevelDebug)

	if middleware.Name() != "logging" {
		t.Errorf("Expected name 'logging', got %s", middleware.Name())
	}

	baseHandler := NewMockHandler()
	wrappedHandler := middleware.Wrap(baseHandler)

	message := &Message{
		ID:            "test-id",
		Topic:         "test-topic",
		Payload:       []byte("test payload"),
		CorrelationID: "test-correlation",
	}

	err := wrappedHandler.Handle(context.Background(), message)
	if err != nil {
		t.Errorf("Wrapped handler failed: %v", err)
	}

	// Check logs
	logs := logger.GetLogs()
	if len(logs) < 2 {
		t.Errorf("Expected at least 2 log entries, got %d", len(logs))
	}

	// Check debug log
	if logs[0].Level != "DEBUG" {
		t.Errorf("Expected first log to be DEBUG, got %s", logs[0].Level)
	}

	// Check info log
	found := false
	for _, log := range logs {
		if log.Level == "INFO" && strings.Contains(log.Msg, "processed successfully") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find INFO log with 'processed successfully'")
	}
}

func TestLoggingMiddleware_WithError(t *testing.T) {
	logger := &TestLogger{}
	middleware := NewLoggingMiddleware(logger, LogLevelError)

	baseHandler := NewMockHandler()
	testError := errors.New("test error")
	baseHandler.handleFunc = func(ctx context.Context, message *Message) error {
		return testError
	}

	wrappedHandler := middleware.Wrap(baseHandler)

	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	err := wrappedHandler.Handle(context.Background(), message)
	if err != testError {
		t.Errorf("Expected error to be %v, got %v", testError, err)
	}

	// Check error log
	logs := logger.GetLogs()
	found := false
	for _, log := range logs {
		if log.Level == "ERROR" && strings.Contains(log.Msg, "processing failed") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find ERROR log with 'processing failed'")
	}
}

func TestMetricsMiddleware(t *testing.T) {
	collector := NewTestMetricsCollector()
	middleware := NewMetricsMiddleware(collector)

	if middleware.Name() != "metrics" {
		t.Errorf("Expected name 'metrics', got %s", middleware.Name())
	}

	baseHandler := NewMockHandler()
	wrappedHandler := middleware.Wrap(baseHandler)

	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	err := wrappedHandler.Handle(context.Background(), message)
	if err != nil {
		t.Errorf("Wrapped handler failed: %v", err)
	}

	// Check metrics
	labels := map[string]string{"topic": "test-topic"}

	receivedCount := collector.GetCounter("messages_received_total", labels)
	if receivedCount != 1 {
		t.Errorf("Expected messages_received_total to be 1, got %f", receivedCount)
	}

	processedCount := collector.GetCounter("messages_processed_total", labels)
	if processedCount != 1 {
		t.Errorf("Expected messages_processed_total to be 1, got %f", processedCount)
	}

	histogramValues := collector.GetHistogramValues("message_processing_duration_seconds", labels)
	if len(histogramValues) != 1 {
		t.Errorf("Expected 1 histogram observation, got %d", len(histogramValues))
	}
}

func TestMetricsMiddleware_WithError(t *testing.T) {
	collector := NewTestMetricsCollector()
	middleware := NewMetricsMiddleware(collector)

	baseHandler := NewMockHandler()
	testError := errors.New("test error")
	baseHandler.handleFunc = func(ctx context.Context, message *Message) error {
		return testError
	}

	wrappedHandler := middleware.Wrap(baseHandler)

	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	err := wrappedHandler.Handle(context.Background(), message)
	if err != testError {
		t.Errorf("Expected error to be %v, got %v", testError, err)
	}

	// Check error metrics
	labels := map[string]string{"topic": "test-topic"}

	failedCount := collector.GetCounter("messages_failed_total", labels)
	if failedCount != 1 {
		t.Errorf("Expected messages_failed_total to be 1, got %f", failedCount)
	}

	errorLabels := map[string]string{
		"topic":      "test-topic",
		"error_type": "unknown",
	}
	errorCount := collector.GetCounter("message_errors_by_type_total", errorLabels)
	if errorCount != 1 {
		t.Errorf("Expected message_errors_by_type_total to be 1, got %f", errorCount)
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	middleware := NewTimeoutMiddleware(100 * time.Millisecond)

	if middleware.Name() != "timeout" {
		t.Errorf("Expected name 'timeout', got %s", middleware.Name())
	}

	// Test successful execution within timeout
	baseHandler := NewMockHandler()
	wrappedHandler := middleware.Wrap(baseHandler)

	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	err := wrappedHandler.Handle(context.Background(), message)
	if err != nil {
		t.Errorf("Wrapped handler failed: %v", err)
	}

	// Test timeout scenario
	slowHandler := NewMockHandler()
	slowHandler.handleFunc = func(ctx context.Context, message *Message) error {
		time.Sleep(200 * time.Millisecond) // Longer than timeout
		return nil
	}

	wrappedSlowHandler := middleware.Wrap(slowHandler)

	err = wrappedSlowHandler.Handle(context.Background(), message)
	if err == nil {
		t.Error("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout error message, got: %v", err)
	}
}

func TestRetryMiddleware(t *testing.T) {
	retryConfig := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	middleware := NewRetryMiddleware(retryConfig)

	if middleware.Name() != "retry" {
		t.Errorf("Expected name 'retry', got %s", middleware.Name())
	}

	// Test successful retry
	attemptCount := 0
	retryHandler := NewMockHandler()
	retryHandler.handleFunc = func(ctx context.Context, message *Message) error {
		attemptCount++
		if attemptCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	wrappedHandler := middleware.Wrap(retryHandler)

	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	err := wrappedHandler.Handle(context.Background(), message)
	if err != nil {
		t.Errorf("Expected successful retry, got error: %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}

	// Test max retries exceeded
	attemptCount = 0
	failingHandler := NewMockHandler()
	failingHandler.handleFunc = func(ctx context.Context, message *Message) error {
		attemptCount++
		return errors.New("persistent error")
	}

	wrappedFailingHandler := middleware.Wrap(failingHandler)

	err = wrappedFailingHandler.Handle(context.Background(), message)
	if err == nil {
		t.Error("Expected retry error")
	}

	if !strings.Contains(err.Error(), "max retry attempts") {
		t.Errorf("Expected retry error message, got: %v", err)
	}

	if attemptCount != retryConfig.MaxAttempts+1 { // +1 for initial attempt
		t.Errorf("Expected %d attempts, got %d", retryConfig.MaxAttempts+1, attemptCount)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	logger := &TestLogger{}
	middleware := NewRecoveryMiddleware(logger)

	if middleware.Name() != "recovery" {
		t.Errorf("Expected name 'recovery', got %s", middleware.Name())
	}

	// Test panic recovery
	panicHandler := NewMockHandler()
	panicHandler.handleFunc = func(ctx context.Context, message *Message) error {
		panic("test panic")
	}

	wrappedHandler := middleware.Wrap(panicHandler)

	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	err := wrappedHandler.Handle(context.Background(), message)
	if err == nil {
		t.Error("Expected error from panic recovery")
	}

	if !strings.Contains(err.Error(), "panic recovered") {
		t.Errorf("Expected panic recovery error message, got: %v", err)
	}

	// Check error log
	logs := logger.GetLogs()
	found := false
	for _, log := range logs {
		if log.Level == "ERROR" && strings.Contains(log.Msg, "panic recovered") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find ERROR log with 'panic recovered'")
	}
}

func TestGetErrorType(t *testing.T) {
	// Test with MessagingError
	msgErr := &MessagingError{
		Code:    ErrProcessingFailed,
		Message: "test error",
	}

	errorType := getErrorType(msgErr)
	if errorType != string(ErrProcessingFailed) {
		t.Errorf("Expected error type %s, got %s", ErrProcessingFailed, errorType)
	}

	// Test with regular error
	regularErr := errors.New("regular error")
	errorType = getErrorType(regularErr)
	if errorType != "unknown" {
		t.Errorf("Expected error type 'unknown', got %s", errorType)
	}
}

// Benchmark tests
func BenchmarkLoggingMiddleware(b *testing.B) {
	logger := &TestLogger{}
	middleware := NewLoggingMiddleware(logger, LogLevelInfo)
	baseHandler := NewMockHandler()
	wrappedHandler := middleware.Wrap(baseHandler)

	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wrappedHandler.Handle(ctx, message)
	}
}

func BenchmarkMetricsMiddleware(b *testing.B) {
	collector := NewTestMetricsCollector()
	middleware := NewMetricsMiddleware(collector)
	baseHandler := NewMockHandler()
	wrappedHandler := middleware.Wrap(baseHandler)

	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wrappedHandler.Handle(ctx, message)
	}
}

func BenchmarkMiddlewareChain(b *testing.B) {
	middleware1 := &MockMiddleware{name: "middleware1"}
	middleware2 := &MockMiddleware{name: "middleware2"}
	middleware3 := &MockMiddleware{name: "middleware3"}

	chain := NewMiddlewareChain(middleware1, middleware2, middleware3)
	baseHandler := NewMockHandler()
	wrappedHandler := chain.Build(baseHandler)

	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wrappedHandler.Handle(ctx, message)
	}
}
