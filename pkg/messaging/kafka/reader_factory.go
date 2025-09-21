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
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/segmentio/kafka-go"
)

// kafkaGoReader wraps github.com/segmentio/kafka-go.Reader to satisfy kafkaReader.
type kafkaGoReader struct {
	r       *kafka.Reader
	topic   string
	groupID string

	mu   sync.Mutex
	seen map[string]kafka.Message // key: topic:partition:offset → msg
}

func newKafkaGoReader(brokerCfg *messaging.BrokerConfig, topic string, groupID string, subCfg *messaging.SubscriberConfig) (*kafkaGoReader, error) {
	if brokerCfg == nil || len(brokerCfg.Endpoints) == 0 {
		return nil, messaging.NewConfigError("broker endpoints required for kafka reader", nil)
	}

	// Build SASL/TLS dialer if configured
	var dialer *kafka.Dialer
	if brokerCfg != nil {
		tlsConf, _ := buildTLSConfig(brokerCfg.TLS)
		mech, _ := buildSASLMechanism(brokerCfg.Authentication)
		if tlsConf != nil || mech != nil || brokerCfg.Connection.Timeout > 0 {
			d := &kafka.Dialer{
				Timeout:       brokerCfg.Connection.Timeout,
				DualStack:     true,
				TLS:           tlsConf,
				SASLMechanism: mech,
			}
			dialer = d
		}
	}

	minBytes := 1
	maxBytes := 10 * 1024 * 1024
	maxWait := 500 * time.Millisecond
	commitInterval := 0 * time.Second // we'll commit manually when AutoCommit=false
	if subCfg != nil && subCfg.Offset.Interval > 0 {
		commitInterval = subCfg.Offset.Interval
	}

	startOffset := kafka.LastOffset
	if subCfg != nil {
		switch subCfg.Offset.Initial {
		case messaging.OffsetEarliest:
			startOffset = kafka.FirstOffset
		case messaging.OffsetLatest:
			startOffset = kafka.LastOffset
		}
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokerCfg.Endpoints,
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       minBytes,
		MaxBytes:       maxBytes,
		MaxWait:        maxWait,
		CommitInterval: commitInterval,
		StartOffset:    startOffset,
		Dialer:         dialer,
	})

	return &kafkaGoReader{r: r, topic: topic, groupID: groupID, seen: make(map[string]kafka.Message)}, nil
}

func (gr *kafkaGoReader) Fetch(ctx context.Context) (kafkaRecord, error) {
	msg, err := gr.r.FetchMessage(ctx)
	if err != nil {
		return kafkaRecord{}, err
	}
	headers := make(map[string]string, len(msg.Headers))
	for _, h := range msg.Headers {
		headers[h.Key] = string(h.Value)
	}
	rec := kafkaRecord{
		Topic:     gr.topic,
		Partition: int32(msg.Partition),
		Offset:    msg.Offset,
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   headers,
		Timestamp: msg.Time,
	}
	gr.mu.Lock()
	gr.seen[gr.key(rec.Partition, rec.Offset)] = msg
	gr.mu.Unlock()
	return rec, nil
}

func (gr *kafkaGoReader) Commit(ctx context.Context, rec kafkaRecord) error {
	gr.mu.Lock()
	msg, ok := gr.seen[gr.key(rec.Partition, rec.Offset)]
	if ok {
		delete(gr.seen, gr.key(rec.Partition, rec.Offset))
	}
	gr.mu.Unlock()
	if !ok {
		// best-effort: construct minimal message for commit
		msg = kafka.Message{Topic: gr.topic, Partition: int(rec.Partition), Offset: rec.Offset}
	}
	return gr.r.CommitMessages(ctx, msg)
}

func (gr *kafkaGoReader) Lag(ctx context.Context) (int64, error) {
	_ = ctx // kafka-go Reader.Lag does not accept context in this version
	v := gr.r.Lag()
	return v, nil
}

func (gr *kafkaGoReader) Seek(ctx context.Context, position messaging.SeekPosition) error {
	switch position.Type {
	case messaging.SeekTypeBeginning:
		return gr.r.SetOffsetAt(ctx, time.Unix(0, 0))
	case messaging.SeekTypeEnd:
		// No direct end time; use a near-future timestamp to move to end
		return gr.r.SetOffsetAt(ctx, time.Now().Add(1*time.Second))
	case messaging.SeekTypeOffset:
		return gr.r.SetOffset(position.Offset)
	case messaging.SeekTypeTimestamp:
		return gr.r.SetOffsetAt(ctx, position.Timestamp)
	default:
		return nil
	}
}

func (gr *kafkaGoReader) RebalanceEvents() <-chan rebalanceEvent { return nil }
func (gr *kafkaGoReader) Close() error                           { return gr.r.Close() }
func (gr *kafkaGoReader) Topic() string                          { return gr.topic }
func (gr *kafkaGoReader) GroupID() string                        { return gr.groupID }

func (gr *kafkaGoReader) key(partition int32, offset int64) string {
	return fmt.Sprintf("%s:%d:%d", gr.topic, partition, offset)
}

func wireDefaultReaderFactory() {
	readerFactory = func(brokerCfg *messaging.BrokerConfig, topic string, groupID string, subCfg *messaging.SubscriberConfig) (kafkaReader, error) {
		return newKafkaGoReader(brokerCfg, topic, groupID, subCfg)
	}
}

func init() { wireDefaultReaderFactory() }
