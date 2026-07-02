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
	"github.com/segmentio/kafka-go"
)

// defaultProbeTimeout bounds connectivity probes when neither the caller
// context nor the broker configuration provides a deadline.
const defaultProbeTimeout = 10 * time.Second

// connectivityProber verifies that at least one configured Kafka endpoint
// accepts metadata requests. It is a package-level variable so tests can
// substitute a fake probe, mirroring writerFactory/readerFactory.
var connectivityProber = probeKafkaConnectivity

// probeKafkaConnectivity dials each configured endpoint (honoring TLS/SASL
// settings) and issues a metadata request until one endpoint responds.
func probeKafkaConnectivity(ctx context.Context, cfg *messaging.BrokerConfig) error {
	if cfg == nil || len(cfg.Endpoints) == 0 {
		return messaging.NewConfigError("no kafka endpoints configured", nil)
	}

	tlsConf, err := messaging.BuildTLSConfig(cfg.TLS)
	if err != nil {
		return messaging.NewConfigError("failed to build TLS config for kafka", err)
	}
	mech, err := buildSASLMechanism(cfg.Authentication)
	if err != nil {
		return err
	}

	dialTimeout := cfg.Connection.Timeout
	if dialTimeout <= 0 {
		dialTimeout = defaultProbeTimeout
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, dialTimeout)
		defer cancel()
	}

	dialer := &kafka.Dialer{
		Timeout:       dialTimeout,
		DualStack:     true,
		TLS:           tlsConf,
		SASLMechanism: mech,
	}

	var lastErr error
	for _, endpoint := range cfg.Endpoints {
		conn, dErr := dialer.DialContext(ctx, "tcp", endpoint)
		if dErr != nil {
			lastErr = dErr
			continue
		}
		if deadline, ok := ctx.Deadline(); ok {
			_ = conn.SetDeadline(deadline)
		}
		// A successful metadata request proves the broker speaks the Kafka
		// protocol, even when no partitions exist yet.
		_, rErr := conn.ReadPartitions()
		_ = conn.Close()
		if rErr == nil {
			return nil
		}
		lastErr = rErr
	}
	return lastErr
}

// kafkaBroker implements messaging.MessageBroker backed by segmentio/kafka-go
// writers (via producerPool) and readers (via readerFactory).
type kafkaBroker struct {
	config  *messaging.BrokerConfig
	metrics messaging.BrokerMetrics
	mu      sync.RWMutex
	started bool

	// producerPool provides high-throughput publishing shared across topic publishers
	producerPool *producerPool

	registries *schemaRegistryManager
}

func newKafkaBroker(cfg *messaging.BrokerConfig) *kafkaBroker {
	return &kafkaBroker{config: cfg, registries: newSchemaRegistryManager()}
}

func (b *kafkaBroker) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.started {
		return nil
	}

	b.metrics.ConnectionAttempts++
	if err := connectivityProber(ctx, b.config); err != nil {
		b.metrics.ConnectionFailures++
		b.metrics.LastConnectionError = err.Error()
		b.metrics.ConnectionStatus = "disconnected"
		return messaging.NewConnectionError("failed to connect to kafka", err)
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

	pub, err := newKafkaPublisher(b.producerPool, &config, b.registries)
	if err != nil {
		return nil, err
	}
	return pub, nil
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

	sub, err := newKafkaSubscriber(b.config, &config, b.registries)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

func (b *kafkaBroker) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	status := &messaging.HealthStatus{
		Status:      messaging.HealthStatusHealthy,
		Message:     "kafka broker healthy",
		LastChecked: time.Now(),
		Details: map[string]any{
			"connected": b.IsConnected(),
			"endpoints": append([]string{}, b.config.Endpoints...),
		},
	}

	if !b.IsConnected() {
		status.Status = messaging.HealthStatusDegraded
		status.Message = "kafka broker not connected"
		return status, nil
	}

	start := time.Now()
	if err := connectivityProber(ctx, b.config); err != nil {
		status.Status = messaging.HealthStatusUnhealthy
		status.Message = "kafka broker connectivity probe failed"
		status.ResponseTime = time.Since(start)
		status.Details["probe_error"] = err.Error()
		return status, nil
	}
	status.ResponseTime = time.Since(start)
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
