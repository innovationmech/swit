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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

func TestHashPartitionStrategy_UsesPayloadKey(t *testing.T) {
	cfg := &messaging.PublisherConfig{
		Topic: "orders",
		Routing: messaging.RoutingConfig{
			Strategy:     "hash",
			PartitionKey: "user_id",
		},
	}

	strategy, err := strategyFromConfig(cfg)
	if err != nil {
		t.Fatalf("strategyFromConfig returned error: %v", err)
	}

	message := &messaging.Message{
		ID:        "msg-1",
		Payload:   []byte(`{"user_id":"user-42","event":"created"}`),
		Headers:   map[string]string{},
		Topic:     "orders",
		Timestamp: time.Now(),
	}

	kafkaMsg := toKafkaMessage(cfg.Topic, message)
	strategy.Assign(message, &kafkaMsg)

	if string(kafkaMsg.Key) != "user-42" {
		t.Fatalf("expected key to be derived from payload, got %q", kafkaMsg.Key)
	}
	if kafkaMsg.Headers[partitionStrategyHeader] != "hash" {
		t.Fatalf("expected strategy header 'hash', got %q", kafkaMsg.Headers[partitionStrategyHeader])
	}
	if kafkaMsg.Headers[partitionSourceHeader] != "payload" {
		t.Fatalf("expected source 'payload', got %q", kafkaMsg.Headers[partitionSourceHeader])
	}
	if kafkaMsg.Headers[partitionKeyHeader] != "user-42" {
		t.Fatalf("expected partition key header to equal payload value, got %q", kafkaMsg.Headers[partitionKeyHeader])
	}
	if _, ok := kafkaMsg.Headers[partitionFallbackHeader]; ok {
		t.Fatalf("expected no fallback header when key is resolved")
	}
}

func TestHashPartitionStrategy_FallbackToMessageID(t *testing.T) {
	cfg := &messaging.PublisherConfig{
		Topic: "orders",
		Routing: messaging.RoutingConfig{
			Strategy:     "hash",
			PartitionKey: "user_id",
		},
	}

	strategy, err := strategyFromConfig(cfg)
	if err != nil {
		t.Fatalf("strategyFromConfig returned error: %v", err)
	}

	message := &messaging.Message{
		ID:        "msg-2",
		Payload:   []byte(`{"other":"value"}`),
		Headers:   map[string]string{},
		Topic:     "orders",
		Timestamp: time.Now(),
	}

	kafkaMsg := toKafkaMessage(cfg.Topic, message)
	strategy.Assign(message, &kafkaMsg)

	if string(kafkaMsg.Key) != "msg-2" {
		t.Fatalf("expected fallback to message ID, got %q", kafkaMsg.Key)
	}
	if kafkaMsg.Headers[partitionSourceHeader] != "message_id" {
		t.Fatalf("expected source 'message_id', got %q", kafkaMsg.Headers[partitionSourceHeader])
	}
	if kafkaMsg.Headers[partitionKeyHeader] != "msg-2" {
		t.Fatalf("expected partition key header to use message ID, got %q", kafkaMsg.Headers[partitionKeyHeader])
	}
}

func TestHashPartitionStrategy_FallbackToRoundRobinWhenNoKey(t *testing.T) {
	cfg := &messaging.PublisherConfig{
		Topic: "orders",
		Routing: messaging.RoutingConfig{
			Strategy:     "hash",
			PartitionKey: "unknown",
		},
	}

	strategy, err := strategyFromConfig(cfg)
	if err != nil {
		t.Fatalf("strategyFromConfig returned error: %v", err)
	}

	message := &messaging.Message{
		Payload:   []byte(`{"event":"created"}`),
		Headers:   map[string]string{},
		Topic:     "orders",
		Timestamp: time.Now(),
	}

	kafkaMsg := toKafkaMessage(cfg.Topic, message)
	strategy.Assign(message, &kafkaMsg)

	if len(kafkaMsg.Key) == 0 {
		t.Fatalf("expected fallback to produce a key")
	}
	if kafkaMsg.Headers[partitionStrategyHeader] != "hash" {
		t.Fatalf("expected primary strategy to remain 'hash', got %q", kafkaMsg.Headers[partitionStrategyHeader])
	}
	if kafkaMsg.Headers[partitionFallbackHeader] != "round_robin" {
		t.Fatalf("expected fallback header to be 'round_robin', got %q", kafkaMsg.Headers[partitionFallbackHeader])
	}
	if kafkaMsg.Headers[partitionSourceHeader] != "round_robin" {
		t.Fatalf("expected partition source 'round_robin', got %q", kafkaMsg.Headers[partitionSourceHeader])
	}
}

type recordingWriter struct {
	mu   sync.Mutex
	msgs []KafkaMessage
}

func (r *recordingWriter) WriteMessages(_ context.Context, msgs ...KafkaMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = append(r.msgs, msgs...)
	return nil
}

func (r *recordingWriter) Close() error { return nil }

func (r *recordingWriter) snapshot() []KafkaMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]KafkaMessage, len(r.msgs))
	copy(out, r.msgs)
	return out
}

func TestKafkaPublisher_HashPartitioningConsistentKey(t *testing.T) {
	originalFactory := writerFactory
	rw := &recordingWriter{}
	writerFactory = func(_ *messaging.BrokerConfig, _ string, _ *messaging.PublisherConfig) (kafkaWriter, error) {
		return rw, nil
	}
	defer func() { writerFactory = originalFactory }()

	brokerCfg := &messaging.BrokerConfig{Type: messaging.BrokerTypeKafka}
	pubCfg := &messaging.PublisherConfig{
		Topic: "orders",
		Routing: messaging.RoutingConfig{
			Strategy:     "hash",
			PartitionKey: "user_id",
		},
	}

	pool, err := newProducerPool(brokerCfg, pubCfg)
	if err != nil {
		t.Fatalf("newProducerPool error: %v", err)
	}
	defer pool.Close()

	publisher, err := newKafkaPublisher(pool, pubCfg)
	if err != nil {
		t.Fatalf("newKafkaPublisher error: %v", err)
	}

	msg1 := &messaging.Message{
		ID:        "msg-1",
		Payload:   []byte(`{"user_id":"user-9","value":1}`),
		Topic:     "orders",
		Timestamp: time.Now(),
	}

	msg2 := &messaging.Message{
		ID:        "msg-2",
		Payload:   []byte(`{"user_id":"user-9","value":2}`),
		Topic:     "orders",
		Timestamp: time.Now(),
	}

	if err := publisher.Publish(context.Background(), msg1); err != nil {
		t.Fatalf("Publish msg1 failed: %v", err)
	}
	if err := publisher.Publish(context.Background(), msg2); err != nil {
		t.Fatalf("Publish msg2 failed: %v", err)
	}

	wrote := rw.snapshot()
	if len(wrote) != 2 {
		t.Fatalf("expected 2 messages written, got %d", len(wrote))
	}
	if string(wrote[0].Key) != string(wrote[1].Key) {
		t.Fatalf("expected consistent key for identical partition values, got %q and %q", wrote[0].Key, wrote[1].Key)
	}

	meta := buildConfirmationMetadata("orders", &wrote[0])
	if meta["partition_strategy"] != "hash" {
		t.Fatalf("expected confirmation metadata strategy 'hash', got %q", meta["partition_strategy"])
	}
	if meta["partition_key"] != "user-9" {
		t.Fatalf("expected confirmation metadata partition key 'user-9', got %q", meta["partition_key"])
	}
}
