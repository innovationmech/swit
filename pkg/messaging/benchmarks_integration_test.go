//go:build integration
// +build integration

package messaging

import (
	"context"
	"testing"
	"time"

	compose "github.com/innovationmech/swit/pkg/messaging/testutil/compose"
)

// 端到端吞吐/延迟基准（RabbitMQ）
func BenchmarkRabbitMQ_EndToEndThroughput(b *testing.B) {
	h := compose.NewHarness(
		compose.WithServices("rabbitmq"),
		compose.WithProjectName("swit-bench-rabbit"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := h.Start(ctx); err != nil {
		b.Fatalf("compose start: %v", err)
	}
	defer h.Stop(context.Background())

	endpoints := h.Endpoints()
	queue := "bench-rabbit-q"
	brokerCfg := &BrokerConfig{Type: BrokerTypeRabbitMQ, Endpoints: []string{endpoints.Rabbit}, Extra: map[string]any{
		"rabbitmq": map[string]any{
			"topology": map[string]any{"queues": map[string]any{queue: map[string]any{"durable": false}}},
		},
	}}
	broker, err := NewMessageBroker(brokerCfg)
	if err != nil {
		b.Fatal(err)
	}
	if err := broker.Connect(ctx); err != nil {
		b.Fatal(err)
	}
	defer broker.Disconnect(context.Background())

	pub, err := broker.CreatePublisher(PublisherConfig{Topic: queue})
	if err != nil {
		b.Fatal(err)
	}
	defer pub.Close()

	sub, err := broker.CreateSubscriber(SubscriberConfig{Topics: []string{queue}, ConsumerGroup: "cg-bench"})
	if err != nil {
		b.Fatal(err)
	}
	defer sub.Close()

	done := make(chan struct{}, 1)
	count := 0
	go func() {
		_ = sub.Subscribe(ctx, MessageHandlerFunc(func(ctx context.Context, m *Message) error {
			count++
			if count >= 1000 {
				select {
				case done <- struct{}{}:
				default:
				}
			}
			return nil
		}))
	}()

	msgs := make([]*Message, 1000)
	payload := make([]byte, 256)
	for i := range msgs {
		msgs[i] = &Message{Topic: queue, Payload: payload}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 发布固定批次并等待消费完成
		for _, m := range msgs {
			_ = pub.Publish(ctx, m)
		}
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			b.Fatalf("timeout waiting for consumption")
		}
	}
}
