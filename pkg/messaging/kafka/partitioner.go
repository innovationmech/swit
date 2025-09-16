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
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

const (
	partitionStrategyHeader = "swit.partition.strategy"
	partitionSourceHeader   = "swit.partition.key_source"
	partitionKeyHeader      = "swit.partition.key"
	partitionFallbackHeader = "swit.partition.fallback"
)

// PartitionStrategy determines how Kafka messages should be partitioned.
type PartitionStrategy interface {
	Name() string
	Assign(message *messaging.Message, msg *KafkaMessage)
}

type partitionStrategyFactory func(cfg *messaging.PublisherConfig) (PartitionStrategy, error)

var partitionStrategies = struct {
	sync.RWMutex
	factories map[string]partitionStrategyFactory
}{
	factories: make(map[string]partitionStrategyFactory),
}

// RegisterPartitionStrategy registers a new partition strategy factory.
func RegisterPartitionStrategy(name string, factory func(cfg *messaging.PublisherConfig) (PartitionStrategy, error)) error {
	if name == "" {
		return fmt.Errorf("partition strategy name cannot be empty")
	}
	if factory == nil {
		return fmt.Errorf("partition strategy factory cannot be nil")
	}

	partitionStrategies.Lock()
	defer partitionStrategies.Unlock()
	partitionStrategies.factories[strings.ToLower(name)] = factory
	return nil
}

func strategyFromConfig(cfg *messaging.PublisherConfig) (PartitionStrategy, error) {
	if cfg == nil {
		return newNoopPartitionStrategy(), nil
	}

	strategyName := strings.ToLower(cfg.Routing.Strategy)
	partitionStrategies.RLock()
	factory := partitionStrategies.factories[strategyName]
	partitionStrategies.RUnlock()

	if factory == nil {
		return newNoopPartitionStrategy(), nil
	}

	return factory(cfg)
}

func init() {
	_ = RegisterPartitionStrategy("hash", newHashPartitionStrategy)
	_ = RegisterPartitionStrategy("round_robin", newRoundRobinPartitionStrategy)
	_ = RegisterPartitionStrategy("random", newRandomPartitionStrategy)
}

func newNoopPartitionStrategy() PartitionStrategy {
	return &noopPartitionStrategy{}
}

type noopPartitionStrategy struct{}

func (n *noopPartitionStrategy) Name() string {
	return "noop"
}

func (n *noopPartitionStrategy) Assign(_ *messaging.Message, _ *KafkaMessage) {}

func newHashPartitionStrategy(cfg *messaging.PublisherConfig) (PartitionStrategy, error) {
	fallback, _ := newRoundRobinPartitionStrategy(cfg)
	return &hashPartitionStrategy{
		partitionKey: cfg.Routing.PartitionKey,
		fallback:     fallback,
	}, nil
}

type hashPartitionStrategy struct {
	partitionKey string
	fallback     PartitionStrategy
}

func (h *hashPartitionStrategy) Name() string {
	return "hash"
}

func (h *hashPartitionStrategy) Assign(message *messaging.Message, msg *KafkaMessage) {
	setPartitionStrategy(msg, h.Name())

	if len(msg.Key) > 0 {
		setPartitionHeader(msg, partitionSourceHeader, "message_key")
		return
	}

	if key := h.fromHeaders(message); key != "" {
		h.applyKey(msg, key, "header")
		return
	}

	if key := h.fromPayload(message); key != "" {
		h.applyKey(msg, key, "payload")
		return
	}

	if message.CorrelationID != "" {
		h.applyKey(msg, message.CorrelationID, "correlation_id")
		return
	}

	if message.ID != "" {
		h.applyKey(msg, message.ID, "message_id")
		return
	}

	if h.fallback != nil {
		setPartitionHeader(msg, partitionFallbackHeader, h.fallback.Name())
		h.fallback.Assign(message, msg)
	}
}

func (h *hashPartitionStrategy) applyKey(msg *KafkaMessage, key string, source string) {
	if key == "" {
		return
	}
	msg.Key = []byte(key)
	setPartitionHeader(msg, partitionSourceHeader, source)
	setPartitionHeader(msg, partitionKeyHeader, key)
}

func (h *hashPartitionStrategy) fromHeaders(message *messaging.Message) string {
	if message == nil {
		return ""
	}
	if len(message.Headers) == 0 || h.partitionKey == "" {
		return ""
	}

	if value, ok := message.Headers[h.partitionKey]; ok && value != "" {
		return value
	}
	if value, ok := message.Headers[strings.ToLower(h.partitionKey)]; ok && value != "" {
		return value
	}
	return ""
}

func (h *hashPartitionStrategy) fromPayload(message *messaging.Message) string {
	if message == nil || len(message.Payload) == 0 || h.partitionKey == "" {
		return ""
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return ""
	}

	if value, ok := payload[h.partitionKey]; ok {
		return fmt.Sprintf("%v", value)
	}

	// Support dotted paths for shallow nesting: e.g., user.id
	if strings.Contains(h.partitionKey, ".") {
		parts := strings.Split(h.partitionKey, ".")
		current := payload
		for idx, part := range parts {
			raw, exists := current[part]
			if !exists {
				return ""
			}
			if idx == len(parts)-1 {
				return fmt.Sprintf("%v", raw)
			}
			next, ok := raw.(map[string]interface{})
			if !ok {
				return ""
			}
			current = next
		}
	}

	return ""
}

func newRoundRobinPartitionStrategy(_ *messaging.PublisherConfig) (PartitionStrategy, error) {
	return &roundRobinPartitionStrategy{}, nil
}

type roundRobinPartitionStrategy struct {
	counter atomic.Uint64
}

func (r *roundRobinPartitionStrategy) Name() string {
	return "round_robin"
}

func (r *roundRobinPartitionStrategy) Assign(_ *messaging.Message, msg *KafkaMessage) {
	if len(msg.Key) > 0 {
		setPartitionStrategy(msg, r.Name())
		if _, ok := msg.Headers[partitionSourceHeader]; !ok {
			setPartitionHeader(msg, partitionSourceHeader, "message_key")
		}
		return
	}

	next := r.counter.Add(1)
	key := fmt.Sprintf("rr-%d", next)
	msg.Key = []byte(key)
	setPartitionStrategy(msg, r.Name())
	setPartitionHeader(msg, partitionSourceHeader, "round_robin")
	setPartitionHeader(msg, partitionKeyHeader, key)
}

func newRandomPartitionStrategy(_ *messaging.PublisherConfig) (PartitionStrategy, error) {
	return &randomPartitionStrategy{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}, nil
}

type randomPartitionStrategy struct {
	rng   *rand.Rand
	rngMu sync.Mutex
}

func (r *randomPartitionStrategy) Name() string {
	return "random"
}

func (r *randomPartitionStrategy) Assign(_ *messaging.Message, msg *KafkaMessage) {
	if len(msg.Key) > 0 {
		setPartitionStrategy(msg, r.Name())
		if _, ok := msg.Headers[partitionSourceHeader]; !ok {
			setPartitionHeader(msg, partitionSourceHeader, "message_key")
		}
		return
	}

	r.rngMu.Lock()
	value := r.rng.Uint64()
	r.rngMu.Unlock()

	key := fmt.Sprintf("rnd-%x", value)
	msg.Key = []byte(key)
	setPartitionStrategy(msg, r.Name())
	setPartitionHeader(msg, partitionSourceHeader, "random")
	setPartitionHeader(msg, partitionKeyHeader, key)
}

func setPartitionStrategy(msg *KafkaMessage, strategy string) {
	if msg == nil || strategy == "" {
		return
	}
	ensureHeaders(msg)
	if current, ok := msg.Headers[partitionStrategyHeader]; ok && current != "" {
		return
	}
	msg.Headers[partitionStrategyHeader] = strategy
}

func setPartitionHeader(msg *KafkaMessage, key, value string) {
	if msg == nil || key == "" || value == "" {
		return
	}
	ensureHeaders(msg)
	msg.Headers[key] = value
}

func ensureHeaders(msg *KafkaMessage) {
	if msg.Headers == nil {
		msg.Headers = make(map[string]string)
	}
}
