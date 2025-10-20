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
	"fmt"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/saga"
)

// BenchmarkPublishSingleVsBatch compares the performance of publishing
// events individually vs batch publishing.
func BenchmarkPublishSingleVsBatch(b *testing.B) {
	logger.InitLogger()

	// Setup mock publisher
	mockPub := &mockPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			// Simulate minimal latency
			time.Sleep(10 * time.Microsecond)
			return nil
		},
	}

	broker := &mockMessageBroker{
		publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
			return mockPub, nil
		},
	}

	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		TopicPrefix:     "saga",
		EnableConfirm:   false,
		RetryAttempts:   0, // No retries for benchmark
		RetryInterval:   0,
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	if err != nil {
		b.Fatalf("Failed to create publisher: %v", err)
	}
	defer publisher.Close()

	ctx := context.Background()

	// Benchmark sizes
	sizes := []int{10, 50, 100, 500, 1000}

	for _, size := range sizes {
		// Create events for benchmarking
		events := make([]*saga.SagaEvent, size)
		for i := 0; i < size; i++ {
			events[i] = &saga.SagaEvent{
				ID:        fmt.Sprintf("event-%d", i),
				SagaID:    "saga-benchmark",
				Type:      saga.EventSagaStarted,
				Version:   "1.0",
				Timestamp: time.Now(),
			}
		}

		// Benchmark individual publishing
		b.Run(fmt.Sprintf("Individual_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, event := range events {
					if err := publisher.PublishSagaEvent(ctx, event); err != nil {
						b.Fatalf("Failed to publish event: %v", err)
					}
				}
			}
		})

		// Benchmark batch publishing
		b.Run(fmt.Sprintf("Batch_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := publisher.PublishBatch(ctx, events); err != nil {
					b.Fatalf("Failed to publish batch: %v", err)
				}
			}
		})
	}
}

// BenchmarkPublishBatch measures the performance of batch publishing
// with different batch sizes.
func BenchmarkPublishBatch(b *testing.B) {
	logger.InitLogger()

	mockPub := &mockPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			return nil
		},
	}

	broker := &mockMessageBroker{
		publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
			return mockPub, nil
		},
	}

	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		TopicPrefix:     "saga",
		EnableConfirm:   false,
		RetryAttempts:   0,
		RetryInterval:   0,
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	if err != nil {
		b.Fatalf("Failed to create publisher: %v", err)
	}
	defer publisher.Close()

	ctx := context.Background()

	sizes := []int{1, 10, 50, 100, 500, 1000}

	for _, size := range sizes {
		events := make([]*saga.SagaEvent, size)
		for i := 0; i < size; i++ {
			events[i] = &saga.SagaEvent{
				ID:        fmt.Sprintf("event-%d", i),
				SagaID:    "saga-bench",
				Type:      saga.EventSagaStarted,
				Version:   "1.0",
				Timestamp: time.Now(),
			}
		}

		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := publisher.PublishBatch(ctx, events); err != nil {
					b.Fatalf("Failed to publish batch: %v", err)
				}
			}
		})
	}
}

// BenchmarkPublishBatchAsync measures the performance of async batch publishing.
func BenchmarkPublishBatchAsync(b *testing.B) {
	logger.InitLogger()

	mockPub := &mockPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			return nil
		},
	}

	broker := &mockMessageBroker{
		publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
			return mockPub, nil
		},
	}

	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		TopicPrefix:     "saga",
		EnableConfirm:   false,
		RetryAttempts:   0,
		RetryInterval:   0,
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	if err != nil {
		b.Fatalf("Failed to create publisher: %v", err)
	}
	defer publisher.Close()

	ctx := context.Background()

	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		events := make([]*saga.SagaEvent, size)
		for i := 0; i < size; i++ {
			events[i] = &saga.SagaEvent{
				ID:        fmt.Sprintf("event-%d", i),
				SagaID:    "saga-async-bench",
				Type:      saga.EventSagaStarted,
				Version:   "1.0",
				Timestamp: time.Now(),
			}
		}

		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			done := make(chan struct{})
			callback := func(err error) {
				if err != nil {
					b.Fatalf("Async publish failed: %v", err)
				}
				done <- struct{}{}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := publisher.PublishBatchAsync(ctx, events, callback); err != nil {
					b.Fatalf("Failed to start async publish: %v", err)
				}
				<-done
			}
		})
	}
}

