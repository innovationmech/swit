package kafka

import (
    "context"
    "errors"
    "testing"
    "time"

    "github.com/innovationmech/swit/pkg/messaging"
)

// fakeReader is a simple in-memory kafkaReader for tests.
type fakeReader struct {
    topic   string
    groupID string
    recCh   chan kafkaRecord
    evCh    chan rebalanceEvent
    closed  bool
}

func (f *fakeReader) Fetch(ctx context.Context) (kafkaRecord, error) {
    select {
    case rec := <-f.recCh:
        return rec, nil
    case <-ctx.Done():
        return kafkaRecord{}, ctx.Err()
    }
}
func (f *fakeReader) Commit(ctx context.Context, rec kafkaRecord) error { return nil }
func (f *fakeReader) Lag(ctx context.Context) (int64, error)            { return 0, nil }
func (f *fakeReader) Seek(ctx context.Context, position messaging.SeekPosition) error {
    return nil
}
func (f *fakeReader) RebalanceEvents() <-chan rebalanceEvent { return f.evCh }
func (f *fakeReader) Close() error {
    if f.closed {
        return errors.New("already closed")
    }
    f.closed = true
    close(f.recCh)
    close(f.evCh)
    return nil
}
func (f *fakeReader) Topic() string   { return f.topic }
func (f *fakeReader) GroupID() string { return f.groupID }

func TestKafkaSubscriber_Subscribe_And_Commit(t *testing.T) {
    // Wire readerFactory to fakeReader
    readerFactory = func(_ *messaging.BrokerConfig, topic string, groupID string, _ *messaging.SubscriberConfig) (kafkaReader, error) {
        return &fakeReader{topic: topic, groupID: groupID, recCh: make(chan kafkaRecord, 1), evCh: make(chan rebalanceEvent, 1)}, nil
    }

    cfg := &messaging.BrokerConfig{Type: messaging.BrokerTypeKafka, Endpoints: []string{"localhost:9092"}}
    subCfg := &messaging.SubscriberConfig{Topics: []string{"orders"}, ConsumerGroup: "cg"}
    sub, err := newKafkaSubscriber(cfg, subCfg)
    if err != nil { t.Fatalf("newKafkaSubscriber: %v", err) }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    received := make(chan *messaging.Message, 1)
    handler := messaging.MessageHandlerFunc(func(ctx context.Context, m *messaging.Message) error {
        received <- m
        return nil
    })

    if err := sub.Subscribe(ctx, handler); err != nil {
        t.Fatalf("Subscribe: %v", err)
    }

    // Send a fake record
    fr := sub.readers["orders"].(*fakeReader)
    fr.recCh <- kafkaRecord{Topic: "orders", Partition: 0, Offset: 1, Key: []byte("k"), Value: []byte("v"), Timestamp: time.Now()}

    select {
    case msg := <-received:
        if msg.Topic != "orders" || string(msg.Payload) != "v" {
            t.Fatalf("unexpected message: %+v", msg)
        }
    case <-time.After(2 * time.Second):
        t.Fatal("timeout waiting for message")
    }

    // Cleanup
    _ = sub.Unsubscribe(context.Background())
}


