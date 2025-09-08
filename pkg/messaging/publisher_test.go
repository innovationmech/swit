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

package messaging

import (
	"context"
	"testing"
	"time"
)

// TestStreamingMetrics tests the StreamingMetrics struct and its fields.
func TestStreamingMetrics(t *testing.T) {
	metrics := &StreamingMetrics{
		ActiveStreams:       5,
		TotalStreamsCreated: 100,
		BufferUtilization:   75.5,
		ThroughputMBps:      10.25,
		BackpressureEvents:  3,
		CompressionRatio:    2.5,
		MemoryUsage:         1024 * 1024 * 50, // 50MB
		FlushOperations:     500,
	}

	if metrics.ActiveStreams != 5 {
		t.Errorf("Expected ActiveStreams to be 5, got %d", metrics.ActiveStreams)
	}

	if metrics.BufferUtilization != 75.5 {
		t.Errorf("Expected BufferUtilization to be 75.5, got %f", metrics.BufferUtilization)
	}

	if metrics.ThroughputMBps != 10.25 {
		t.Errorf("Expected ThroughputMBps to be 10.25, got %f", metrics.ThroughputMBps)
	}

	if metrics.CompressionRatio != 2.5 {
		t.Errorf("Expected CompressionRatio to be 2.5, got %f", metrics.CompressionRatio)
	}
}

// TestBufferStatus tests the BufferStatus struct and its fields.
func TestBufferStatus(t *testing.T) {
	buffer := &BufferStatus{
		Size:               1000,
		Capacity:           5000,
		Utilization:        20.0,
		BytesBuffered:      1024 * 1000, // 1MB
		AverageMessageSize: 1024,        // 1KB
		HighWaterMark:      4500,
	}

	if buffer.Size != 1000 {
		t.Errorf("Expected Size to be 1000, got %d", buffer.Size)
	}

	if buffer.Capacity != 5000 {
		t.Errorf("Expected Capacity to be 5000, got %d", buffer.Capacity)
	}

	if buffer.Utilization != 20.0 {
		t.Errorf("Expected Utilization to be 20.0, got %f", buffer.Utilization)
	}

	if buffer.AverageMessageSize != 1024 {
		t.Errorf("Expected AverageMessageSize to be 1024, got %d", buffer.AverageMessageSize)
	}
}

// TestPartitioningStrategy tests the PartitioningStrategy constants.
func TestPartitioningStrategy(t *testing.T) {
	strategies := []PartitioningStrategy{
		PartitioningHash,
		PartitioningRoundRobin,
		PartitioningRandom,
		PartitioningStickyRandom,
	}

	expectedStrategies := []string{
		"hash",
		"round_robin",
		"random",
		"sticky_random",
	}

	for i, strategy := range strategies {
		if string(strategy) != expectedStrategies[i] {
			t.Errorf("Expected strategy %d to be %s, got %s", i, expectedStrategies[i], string(strategy))
		}
	}
}

// TestBatchMetrics tests the BatchMetrics struct and its fields.
func TestBatchMetrics(t *testing.T) {
	batchMetrics := &BatchMetrics{
		BatchesCreated:             100,
		BatchesPublished:           95,
		BatchesFailed:              5,
		AverageBatchSize:           50.5,
		MaxBatchSize:               100,
		AverageBatchLatency:        15 * time.Millisecond,
		BatchCompressionRatio:      2.0,
		BatchPartitionDistribution: map[string]int64{"partition-0": 500, "partition-1": 400},
	}

	if batchMetrics.BatchesCreated != 100 {
		t.Errorf("Expected BatchesCreated to be 100, got %d", batchMetrics.BatchesCreated)
	}

	if batchMetrics.BatchesPublished != 95 {
		t.Errorf("Expected BatchesPublished to be 95, got %d", batchMetrics.BatchesPublished)
	}

	if batchMetrics.AverageBatchSize != 50.5 {
		t.Errorf("Expected AverageBatchSize to be 50.5, got %f", batchMetrics.AverageBatchSize)
	}

	if batchMetrics.AverageBatchLatency != 15*time.Millisecond {
		t.Errorf("Expected AverageBatchLatency to be 15ms, got %v", batchMetrics.AverageBatchLatency)
	}

	if len(batchMetrics.BatchPartitionDistribution) != 2 {
		t.Errorf("Expected 2 partitions in distribution, got %d", len(batchMetrics.BatchPartitionDistribution))
	}
}

// TestTransactionMetrics tests the TransactionMetrics struct and its fields.
func TestTransactionMetrics(t *testing.T) {
	txMetrics := &TransactionMetrics{
		ActiveTransactions:         10,
		TransactionsCommitted:      500,
		TransactionsRolledBack:     25,
		AverageTransactionDuration: 100 * time.Millisecond,
		SavepointsCreated:          50,
		SavepointRollbacks:         5,
		NestedTransactions:         15,
		MaxTransactionDepth:        3,
	}

	if txMetrics.ActiveTransactions != 10 {
		t.Errorf("Expected ActiveTransactions to be 10, got %d", txMetrics.ActiveTransactions)
	}

	if txMetrics.TransactionsCommitted != 500 {
		t.Errorf("Expected TransactionsCommitted to be 500, got %d", txMetrics.TransactionsCommitted)
	}

	if txMetrics.TransactionsRolledBack != 25 {
		t.Errorf("Expected TransactionsRolledBack to be 25, got %d", txMetrics.TransactionsRolledBack)
	}

	if txMetrics.AverageTransactionDuration != 100*time.Millisecond {
		t.Errorf("Expected AverageTransactionDuration to be 100ms, got %v", txMetrics.AverageTransactionDuration)
	}

	if txMetrics.MaxTransactionDepth != 3 {
		t.Errorf("Expected MaxTransactionDepth to be 3, got %d", txMetrics.MaxTransactionDepth)
	}
}

// TestStreamConfig tests the StreamConfig struct and its default values.
func TestStreamConfig(t *testing.T) {
	config := &StreamConfig{
		BufferSize:         10000,
		FlowControl:        true,
		MaxBatchSize:       1000,
		FlushInterval:      10 * time.Millisecond,
		CompressionEnabled: true,
		CompressionLevel:   6,
		MaxMemoryUsage:     104857600, // 100MB
	}

	if config.BufferSize != 10000 {
		t.Errorf("Expected BufferSize to be 10000, got %d", config.BufferSize)
	}

	if !config.FlowControl {
		t.Error("Expected FlowControl to be true")
	}

	if config.MaxBatchSize != 1000 {
		t.Errorf("Expected MaxBatchSize to be 1000, got %d", config.MaxBatchSize)
	}

	if config.FlushInterval != 10*time.Millisecond {
		t.Errorf("Expected FlushInterval to be 10ms, got %v", config.FlushInterval)
	}

	if !config.CompressionEnabled {
		t.Error("Expected CompressionEnabled to be true")
	}

	if config.CompressionLevel != 6 {
		t.Errorf("Expected CompressionLevel to be 6, got %d", config.CompressionLevel)
	}

	if config.MaxMemoryUsage != 104857600 {
		t.Errorf("Expected MaxMemoryUsage to be 104857600, got %d", config.MaxMemoryUsage)
	}
}

// TestPublisherBuilderPattern tests the builder pattern functionality.
func TestPublisherBuilderPattern(t *testing.T) {
	// Create mock broker for testing
	mockBroker := &mockMessageBroker{}

	// Test builder creation
	builder := NewPublisherBuilder(mockBroker)
	if builder == nil {
		t.Fatal("Expected non-nil publisher builder")
	}

	// Test method chaining
	builder = builder.
		WithTopic("test-topic").
		WithBatching(500, 50*time.Millisecond).
		WithCompression(CompressionGZIP).
		WithRetries(5, 2*time.Second).
		WithStreaming(true).
		WithTransactions(true).
		WithAsync(true).
		WithTimeout(60*time.Second, 60*time.Second, 30*time.Second)

	if builder == nil {
		t.Error("Expected builder chain to return non-nil builder")
	}

	// Test building without topic (should fail)
	emptyBuilder := NewPublisherBuilder(mockBroker)
	_, err := emptyBuilder.Build(context.Background())
	if err == nil {
		t.Error("Expected error when building publisher without topic")
	}

	// Test building with topic (should succeed)
	_, err = builder.Build(context.Background())
	if err != nil {
		t.Errorf("Expected no error when building valid publisher, got: %v", err)
	}
}

// TestPublisherBuilderConcurrency tests concurrent access to the publisher builder.
func TestPublisherBuilderConcurrency(t *testing.T) {
	mockBroker := &mockMessageBroker{}
	builder := NewPublisherBuilder(mockBroker).WithTopic("concurrent-test")

	// Test concurrent configuration updates
	done := make(chan bool)
	errors := make(chan error, 4)

	// Concurrent configuration updates
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 100; i++ {
			builder.WithBatching(i+1, time.Duration(i+1)*time.Millisecond)
		}
	}()

	go func() {
		defer func() { done <- true }()
		for i := 0; i < 100; i++ {
			builder.WithRetries(i+1, time.Duration(i+1)*time.Second)
		}
	}()

	go func() {
		defer func() { done <- true }()
		for i := 0; i < 100; i++ {
			builder.WithStreaming(i%2 == 0)
		}
	}()

	go func() {
		defer func() { done <- true }()
		for i := 0; i < 10; i++ {
			_, err := builder.Build(context.Background())
			if err != nil {
				errors <- err
				return
			}
		}
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Unexpected error during concurrent access: %v", err)
	}
}

// TestPublisherBuilderDefaults tests that the builder sets appropriate default values.
func TestPublisherBuilderDefaults(t *testing.T) {
	mockBroker := &mockMessageBroker{}
	builder := NewPublisherBuilder(mockBroker).WithTopic("defaults-test")

	// Build publisher to check defaults
	_, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("Expected no error building publisher with defaults, got: %v", err)
	}

	// Cast to implementation to access config
	builderImpl, ok := builder.(*publisherBuilderImpl)
	if !ok {
		t.Fatal("Expected publisherBuilderImpl type")
	}

	// Check default values
	if builderImpl.config.Batching.MaxMessages != 100 {
		t.Errorf("Expected default batch size of 100, got %d", builderImpl.config.Batching.MaxMessages)
	}

	if builderImpl.config.Compression != CompressionNone {
		t.Errorf("Expected default compression to be none, got %v", builderImpl.config.Compression)
	}

	if builderImpl.config.Retry.MaxAttempts != 3 {
		t.Errorf("Expected default retry attempts of 3, got %d", builderImpl.config.Retry.MaxAttempts)
	}

	if builderImpl.config.Timeout.Publish != 30*time.Second {
		t.Errorf("Expected default publish timeout of 30s, got %v", builderImpl.config.Timeout.Publish)
	}
}

// mockStreamingPublisher provides a mock implementation for testing streaming functionality.
type mockStreamingPublisher struct {
	*mockEventPublisher
	streams []*mockPublishStream
}

func (m *mockStreamingPublisher) StartStream(ctx context.Context, config StreamConfig) (PublishStream, error) {
	stream := &mockPublishStream{
		config:      config,
		buffer:      make([]*Message, 0, config.BufferSize),
		closed:      false,
		flowControl: config.FlowControl,
	}
	m.streams = append(m.streams, stream)
	return stream, nil
}

func (m *mockStreamingPublisher) GetStreamMetrics() *StreamingMetrics {
	return &StreamingMetrics{
		ActiveStreams:       int64(len(m.streams)),
		TotalStreamsCreated: int64(len(m.streams)),
		BufferUtilization:   50.0,
		ThroughputMBps:      5.0,
		BackpressureEvents:  0,
		CompressionRatio:    1.5,
		MemoryUsage:         1024 * 1024, // 1MB
		FlushOperations:     100,
	}
}

// mockPublishStream provides a mock implementation for testing stream functionality.
type mockPublishStream struct {
	config      StreamConfig
	buffer      []*Message
	closed      bool
	flowControl bool
}

func (m *mockPublishStream) Publish(ctx context.Context, message *Message) error {
	if m.closed {
		return NewPublishError("stream closed", nil)
	}

	if len(m.buffer) >= m.config.BufferSize {
		if !m.flowControl {
			return NewPublishError("buffer full", nil)
		}
		// In real implementation, this would block until space is available
	}

	m.buffer = append(m.buffer, message)
	return nil
}

func (m *mockPublishStream) PublishWithBackpressure(ctx context.Context, message *Message) error {
	// Same as Publish for mock, but would handle backpressure in real implementation
	return m.Publish(ctx, message)
}

func (m *mockPublishStream) Flush(ctx context.Context) error {
	if m.closed {
		return NewPublishError("stream closed", nil)
	}
	// Mock flush - clear buffer
	m.buffer = m.buffer[:0]
	return nil
}

func (m *mockPublishStream) Close() error {
	m.closed = true
	m.buffer = nil
	return nil
}

func (m *mockPublishStream) GetBufferStatus() *BufferStatus {
	return &BufferStatus{
		Size:               int64(len(m.buffer)),
		Capacity:           int64(m.config.BufferSize),
		Utilization:        float64(len(m.buffer)) / float64(m.config.BufferSize) * 100,
		BytesBuffered:      int64(len(m.buffer) * 1024), // Assume 1KB per message
		AverageMessageSize: 1024,
		HighWaterMark:      int64(cap(m.buffer)),
	}
}

func (m *mockPublishStream) SetFlowControl(enabled bool) {
	m.flowControl = enabled
}

// TestStreamingPublisherInterface tests the StreamingPublisher interface.
func TestStreamingPublisherInterface(t *testing.T) {
	publisher := &mockStreamingPublisher{
		mockEventPublisher: &mockEventPublisher{},
		streams:            make([]*mockPublishStream, 0),
	}

	ctx := context.Background()
	config := StreamConfig{
		BufferSize:    1000,
		FlowControl:   true,
		MaxBatchSize:  100,
		FlushInterval: 10 * time.Millisecond,
	}

	// Test StartStream
	stream, err := publisher.StartStream(ctx, config)
	if err != nil {
		t.Errorf("Expected no error starting stream, got: %v", err)
	}
	if stream == nil {
		t.Error("Expected non-nil stream")
	}

	// Test GetStreamMetrics
	metrics := publisher.GetStreamMetrics()
	if metrics == nil {
		t.Error("Expected non-nil streaming metrics")
	}
	if metrics.ActiveStreams != 1 {
		t.Errorf("Expected 1 active stream, got %d", metrics.ActiveStreams)
	}

	// Test stream operations
	msg := &Message{
		ID:      "stream-test",
		Topic:   "test-topic",
		Payload: []byte("stream test"),
	}

	// Test Publish
	err = stream.Publish(ctx, msg)
	if err != nil {
		t.Errorf("Expected no error publishing to stream, got: %v", err)
	}

	// Test GetBufferStatus
	status := stream.GetBufferStatus()
	if status == nil {
		t.Error("Expected non-nil buffer status")
	}
	if status.Size != 1 {
		t.Errorf("Expected buffer size of 1, got %d", status.Size)
	}

	// Test Flush
	err = stream.Flush(ctx)
	if err != nil {
		t.Errorf("Expected no error flushing stream, got: %v", err)
	}

	// Verify buffer is empty after flush
	status = stream.GetBufferStatus()
	if status.Size != 0 {
		t.Errorf("Expected empty buffer after flush, got size %d", status.Size)
	}

	// Test Close
	err = stream.Close()
	if err != nil {
		t.Errorf("Expected no error closing stream, got: %v", err)
	}

	// Test operations after close should fail
	err = stream.Publish(ctx, msg)
	if err == nil {
		t.Error("Expected error when publishing to closed stream")
	}
}
