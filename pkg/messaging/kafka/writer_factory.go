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
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/segmentio/kafka-go"
)

// kafkaGoWriter wraps github.com/segmentio/kafka-go.Writer to satisfy kafkaWriter.
type kafkaGoWriter struct {
	w *kafka.Writer
}

func (kw *kafkaGoWriter) WriteMessages(ctx context.Context, msgs ...KafkaMessage) error {
	if len(msgs) == 0 {
		return nil
	}
	converted := make([]kafka.Message, 0, len(msgs))
	for _, m := range msgs {
		headers := make([]kafka.Header, 0, len(m.Headers))
		for k, v := range m.Headers {
			headers = append(headers, kafka.Header{Key: k, Value: []byte(v)})
		}
		converted = append(converted, kafka.Message{
			Key:   m.Key,
			Value: m.Value,
			Time:  m.Timestamp,
			// Topic is configured on writer; headers still forwarded
			Headers: headers,
		})
	}
	return kw.w.WriteMessages(ctx, converted...)
}

func (kw *kafkaGoWriter) Close() error { return kw.w.Close() }

// wireDefaultWriterFactory wires writerFactory to use kafka-go.
func wireDefaultWriterFactory() {
	writerFactory = func(brokerCfg *messaging.BrokerConfig, topic string, pubCfg *messaging.PublisherConfig) (kafkaWriter, error) {
		// Map batching configuration to writer settings with sensible defaults
		batchBytes := int64(1_048_576) // 1MB default
		if pubCfg != nil && pubCfg.Batching.MaxBytes > 0 {
			batchBytes = int64(pubCfg.Batching.MaxBytes)
		}
		batchTimeout := 100 * time.Millisecond
		if pubCfg != nil && pubCfg.Batching.FlushInterval > 0 {
			batchTimeout = pubCfg.Batching.FlushInterval
		}

		w := &kafka.Writer{
			Addr:         kafka.TCP(brokerCfg.Endpoints...),
			Topic:        topic,
			RequiredAcks: kafka.RequireAll,
			BatchBytes:   batchBytes,
			BatchTimeout: batchTimeout,
			// Balancer selection will rely on provided message keys by our partition strategy
			// Default balancer (hash) in kafka-go respects keys.
			AllowAutoTopicCreation: true,
		}

		return &kafkaGoWriter{w: w}, nil
	}
}

func init() { wireDefaultWriterFactory() }
