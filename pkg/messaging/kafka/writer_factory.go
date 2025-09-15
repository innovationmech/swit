package kafka

import (
    "context"
    "time"

    "github.com/innovationmech/swit/pkg/messaging"
)

// default implementation delegates to segmentio/kafka-go if available.
// To avoid hard dependency in this step, we provide a thin adapter with a build tag in future.

// wireDefaultWriterFactory sets a no-op placeholder. Real factory will be set in init() when kafka-go is available.
func wireDefaultWriterFactory() {
    if writerFactory == nil {
        writerFactory = func(brokerCfg *messaging.BrokerConfig, topic string, pubCfg *messaging.PublisherConfig) (kafkaWriter, error) {
            // minimal in-memory fake to keep tests compiling until kafka-go integration is added
            return &fakeWriter{}, nil
        }
    }
}

type fakeWriter struct{}

func (f *fakeWriter) WriteMessages(ctx context.Context, msgs ...KafkaMessage) error { return nil }
func (f *fakeWriter) Close() error { return nil }

func init() {
    wireDefaultWriterFactory()
}

// helper to translate framework config batching into writer settings will be added alongside kafka-go integration.
func _unused_silence() time.Duration { return time.Duration(0) }


