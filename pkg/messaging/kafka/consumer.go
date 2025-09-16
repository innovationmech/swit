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
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// kafkaReader abstracts kafka-go Reader for testability and to avoid direct dependency.
type kafkaReader interface {
	// Fetch blocks until a record is available or context is cancelled.
	Fetch(ctx context.Context) (kafkaRecord, error)
	// Commit commits the provided record's offset.
	Commit(ctx context.Context, rec kafkaRecord) error
	// Lag returns current lag estimation for this reader.
	Lag(ctx context.Context) (int64, error)
	// Seek positions the reader to the desired position.
	Seek(ctx context.Context, position messaging.SeekPosition) error
	// RebalanceEvents emits group rebalance notifications if available.
	RebalanceEvents() <-chan rebalanceEvent
	// Close releases reader resources.
	Close() error
	// Topic returns the topic this reader is consuming.
	Topic() string
	// GroupID returns consumer group id.
	GroupID() string
}

// readerFactory creates a kafkaReader for the given topic/group based on configuration.
var readerFactory func(brokerCfg *messaging.BrokerConfig, topic string, groupID string, subCfg *messaging.SubscriberConfig) (kafkaReader, error)

// kafkaRecord represents a consumed record metadata.
type kafkaRecord struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   map[string]string
	Timestamp time.Time
}

// rebalanceEvent models consumer group rebalance notifications.
type rebalanceEvent struct {
	Type       string // joined, left, rebalanced
	Partitions []int32
}

// kafkaSubscriber implements messaging.EventSubscriber with consumer group management
// and offset control. It runs one reader per topic and a worker pool for processing.
type kafkaSubscriber struct {
	brokerCfg *messaging.BrokerConfig
	config    *messaging.SubscriberConfig

	readers map[string]kafkaReader // topic -> reader
	serde   *schemaSerDe

	handler    messaging.MessageHandler
	middleware []messaging.Middleware

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	paused atomic.Bool

	metricsMu sync.RWMutex
	metrics   messaging.SubscriberMetrics

	lastRebalance atomic.Value // time.Time
}

func newKafkaSubscriber(brokerCfg *messaging.BrokerConfig, cfg *messaging.SubscriberConfig, registries *schemaRegistryManager) (*kafkaSubscriber, error) {
	cpy := *cfg
	var serde *schemaSerDe
	var err error
	if cfg.Serialization != nil && cfg.Serialization.SchemaRegistry != nil {
		serde, err = newSchemaSerDe(cfg.Serialization, registries)
		if err != nil {
			return nil, err
		}
	}
	return &kafkaSubscriber{
		brokerCfg: brokerCfg,
		config:    &cpy,
		readers:   make(map[string]kafkaReader, len(cfg.Topics)),
		serde:     serde,
		metrics:   messaging.SubscriberMetrics{},
	}, nil
}

// Subscribe implements EventSubscriber.
func (s *kafkaSubscriber) Subscribe(ctx context.Context, handler messaging.MessageHandler) error {
	return s.SubscribeWithMiddleware(ctx, handler)
}

// SubscribeWithMiddleware implements EventSubscriber.
func (s *kafkaSubscriber) SubscribeWithMiddleware(ctx context.Context, handler messaging.MessageHandler, middleware ...messaging.Middleware) error {
	if s.ctx != nil {
		return messaging.NewConfigError("subscription already started", nil)
	}

	// Apply middleware chain (outermost is first)
	finalHandler := handler
	for i := len(middleware) - 1; i >= 0; i-- {
		finalHandler = middleware[i].Wrap(finalHandler)
	}
	s.handler = finalHandler
	s.middleware = middleware

	s.ctx, s.cancel = context.WithCancel(ctx)

	// Create readers per topic
	for _, topic := range s.config.Topics {
		r, err := readerFactory(s.brokerCfg, topic, s.config.ConsumerGroup, s.config)
		if err != nil {
			return err
		}
		s.readers[topic] = r

		// Start rebalance event watcher if available
		if evCh := r.RebalanceEvents(); evCh != nil {
			s.wg.Add(1)
			go s.watchRebalances(evCh)
		}

		// Start consuming loop per reader
		s.wg.Add(1)
		go s.consumeLoop(r)
	}

	return nil
}

func (s *kafkaSubscriber) watchRebalances(evCh <-chan rebalanceEvent) {
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			return
		case _, ok := <-evCh:
			if !ok {
				return
			}
			now := time.Now()
			s.lastRebalance.Store(now)
			s.metricsMu.Lock()
			s.metrics.LastActivity = now
			s.metricsMu.Unlock()
		}
	}
}

func (s *kafkaSubscriber) consumeLoop(r kafkaReader) {
	defer s.wg.Done()

	// Worker fan-out uses per-message goroutines bounded by config.Concurrency via a semaphore.
	maxWorkers := s.config.Concurrency
	if maxWorkers <= 0 {
		maxWorkers = 1
	}
	sem := make(chan struct{}, maxWorkers)

	for {
		// Respect pause
		if s.paused.Load() {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
				continue
			}
		}

		// Fetch record
		rec, err := r.Fetch(s.ctx)
		if err != nil {
			if s.ctx.Err() != nil {
				return
			}
			// Backoff briefly on fetch errors
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Acquire worker slot
		select {
		case sem <- struct{}{}:
		case <-s.ctx.Done():
			return
		}

		s.wg.Add(1)
		go func(rec kafkaRecord) {
			defer func() {
				<-sem
				s.wg.Done()
			}()

			start := time.Now()
			msg := s.toMessage(rec)
			decoded, decodeErr := s.prepareIncomingMessage(s.ctx, msg)
			if decodeErr != nil {
				s.recordFailure(decodeErr)
				s.handler.OnError(s.ctx, msg, decodeErr)
				return
			}
			if decoded != nil {
				msg = decoded
			}

			// Processing timeout per config
			procTimeout := s.config.Processing.MaxProcessingTime
			pctx := s.ctx
			var cancel context.CancelFunc
			if procTimeout > 0 {
				pctx, cancel = context.WithTimeout(s.ctx, procTimeout)
				defer cancel()
			}

			err := s.handler.Handle(pctx, msg)

			s.metricsMu.Lock()
			s.metrics.MessagesConsumed++
			if err != nil {
				s.metrics.MessagesFailed++
			} else {
				s.metrics.MessagesProcessed++
			}
			d := time.Since(start)
			if s.metrics.MinProcessingTime == 0 || d < s.metrics.MinProcessingTime {
				s.metrics.MinProcessingTime = d
			}
			if d > s.metrics.MaxProcessingTime {
				s.metrics.MaxProcessingTime = d
			}
			s.metrics.LastActivity = time.Now()
			s.metricsMu.Unlock()

			if err != nil {
				action := s.handler.OnError(pctx, msg, err)
				// Basic error handling policy: retry by NOP here (leave offset uncommitted),
				// other actions could be supported later.
				_ = action
				return
			}

			// Manual commit if AutoCommit disabled
			if !s.config.Offset.AutoCommit {
				_ = r.Commit(s.ctx, rec)
			}
		}(rec)
	}
}

func (s *kafkaSubscriber) toMessage(rec kafkaRecord) *messaging.Message {
	headers := make(map[string]string, len(rec.Headers))
	for k, v := range rec.Headers {
		headers[k] = v
	}
	return &messaging.Message{
		ID:        s.makeMessageID(rec),
		Topic:     rec.Topic,
		Key:       rec.Key,
		Payload:   rec.Value,
		Headers:   headers,
		Timestamp: rec.Timestamp,
		BrokerMetadata: map[string]any{
			"partition": rec.Partition,
			"offset":    rec.Offset,
		},
	}
}

func (s *kafkaSubscriber) prepareIncomingMessage(ctx context.Context, message *messaging.Message) (*messaging.Message, error) {
	if s.serde == nil {
		return message, nil
	}
	return s.serde.Decode(ctx, message)
}

func (s *kafkaSubscriber) recordFailure(_ error) {
	s.metricsMu.Lock()
	s.metrics.MessagesConsumed++
	s.metrics.MessagesFailed++
	s.metrics.LastActivity = time.Now()
	s.metricsMu.Unlock()
}

func (s *kafkaSubscriber) makeMessageID(rec kafkaRecord) string {
	// Topic-partition-offset provides a stable id for idempotency/tracing
	return rec.Topic + "-" + itoa32(rec.Partition) + "-" + itoa64(rec.Offset)
}

func itoa32(v int32) string { return fmtInt(int64(v)) }
func itoa64(v int64) string { return fmtInt(v) }
func fmtInt(v int64) string {
	// faster than fmt since we avoid allocations for small numbers; simplicity is fine here
	return strconv.FormatInt(v, 10)
}

// Unsubscribe implements EventSubscriber.
func (s *kafkaSubscriber) Unsubscribe(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}
	for _, r := range s.readers {
		_ = r.Close()
	}
	return nil
}

// Pause implements EventSubscriber.
func (s *kafkaSubscriber) Pause(ctx context.Context) error {
	s.paused.Store(true)
	return nil
}

// Resume implements EventSubscriber.
func (s *kafkaSubscriber) Resume(ctx context.Context) error {
	s.paused.Store(false)
	return nil
}

// Seek implements EventSubscriber.
func (s *kafkaSubscriber) Seek(ctx context.Context, position messaging.SeekPosition) error {
	var firstErr error
	for _, r := range s.readers {
		if err := r.Seek(ctx, position); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// GetLag implements EventSubscriber.
func (s *kafkaSubscriber) GetLag(ctx context.Context) (int64, error) {
	var total int64
	for _, r := range s.readers {
		lag, err := r.Lag(ctx)
		if err != nil {
			return -1, err
		}
		if lag > 0 {
			total += lag
		}
	}
	return total, nil
}

// Close implements EventSubscriber.
func (s *kafkaSubscriber) Close() error { return s.Unsubscribe(context.Background()) }

// GetMetrics implements EventSubscriber.
func (s *kafkaSubscriber) GetMetrics() *messaging.SubscriberMetrics {
	s.metricsMu.RLock()
	defer s.metricsMu.RUnlock()
	copy := s.metrics // shallow copy of value type
	return &copy
}
