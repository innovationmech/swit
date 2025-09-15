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

package kafka

import (
	"context"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// kafkaBroker is a minimal broker that satisfies messaging.MessageBroker with
// stubbed publisher/subscriber. Connectivity check is a no-op in this scaffold.
type kafkaBroker struct {
	config  *messaging.BrokerConfig
	metrics messaging.BrokerMetrics
	mu      sync.RWMutex
	started bool

	// producerPool provides high-throughput publishing shared across topic publishers
	producerPool *producerPool
}

func newKafkaBroker(cfg *messaging.BrokerConfig) *kafkaBroker {
	return &kafkaBroker{config: cfg}
}

func (b *kafkaBroker) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.started {
		return nil
	}
	b.started = true
	b.metrics.ConnectionStatus = "connected"
	b.metrics.LastConnectionTime = time.Now()
	return nil
}

func (b *kafkaBroker) Disconnect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.started {
		return nil
	}
	b.started = false
	b.metrics.ConnectionStatus = "disconnected"
	if b.producerPool != nil {
		b.producerPool.Close()
		b.producerPool = nil
	}
	return nil
}

func (b *kafkaBroker) Close() error {
	return nil
}

func (b *kafkaBroker) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.started
}

func (b *kafkaBroker) CreatePublisher(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.started {
		return nil, messaging.ErrBrokerNotConnected
	}
	if config.Topic == "" {
		return nil, messaging.NewConfigError("publisher topic is required", nil)
	}

	// Lazy-init producer pool
	if b.producerPool == nil {
		pool, err := newProducerPool(b.config, &config)
		if err != nil {
			return nil, err
		}
		b.producerPool = pool
	}

	return newKafkaPublisher(b.producerPool, &config), nil
}

func (b *kafkaBroker) CreateSubscriber(config messaging.SubscriberConfig) (messaging.EventSubscriber, error) {
	b.mu.RLock()
	started := b.started
	b.mu.RUnlock()
	if !started {
		return nil, messaging.ErrBrokerNotConnected
	}
	if len(config.Topics) == 0 {
		return nil, messaging.NewConfigError("subscriber topics are required", nil)
	}
	if config.ConsumerGroup == "" {
		return nil, messaging.NewConfigError("subscriber consumer_group is required", nil)
	}

	sub, err := newKafkaSubscriber(b.config, &config)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

func (b *kafkaBroker) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	status := &messaging.HealthStatus{
		Status:       messaging.HealthStatusHealthy,
		Message:      "kafka broker scaffold healthy",
		LastChecked:  time.Now(),
		ResponseTime: 0,
		Details: map[string]any{
			"connected": b.IsConnected(),
		},
	}
	return status, nil
}

func (b *kafkaBroker) GetMetrics() *messaging.BrokerMetrics {
	copy := b.metrics.GetSnapshot()
	return copy
}

func (b *kafkaBroker) GetCapabilities() *messaging.BrokerCapabilities {
	caps, _ := messaging.GetCapabilityProfile(messaging.BrokerTypeKafka)
	return caps
}
