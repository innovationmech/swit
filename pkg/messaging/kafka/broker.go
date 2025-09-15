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
	return &stubPublisher{}, nil
}

func (b *kafkaBroker) CreateSubscriber(config messaging.SubscriberConfig) (messaging.EventSubscriber, error) {
	return &stubSubscriber{}, nil
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
