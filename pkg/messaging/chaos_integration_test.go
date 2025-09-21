//go:build integration
// +build integration

package messaging

import (
	"context"
	"testing"
	"time"

	compose "github.com/innovationmech/swit/pkg/messaging/testutil/compose"
)

// 目标：验证在 broker 短暂不可用后，适配器能够自动重连或恢复消费。
func TestChaos_RabbitMQ_ReconnectAfterRestart(t *testing.T) {
	h := compose.NewHarness(
		compose.WithServices("rabbitmq"),
		compose.WithProjectName("swit-chaos-rabbit"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := h.Start(ctx); err != nil {
		t.Fatalf("compose start: %v", err)
	}
	defer h.Stop(context.Background())

	endpoints := h.Endpoints()
	queue := "chaos-rabbit-reconnect-q"

	brokerCfg := &BrokerConfig{Type: BrokerTypeRabbitMQ, Endpoints: []string{endpoints.Rabbit}, Extra: map[string]any{
		"rabbitmq": map[string]any{
			"topology":  map[string]any{"queues": map[string]any{queue: map[string]any{"durable": true}}},
			"reconnect": map[string]any{"enabled": true, "initial_delay": "200ms", "max_delay": "5s"},
		},
	}}

	broker, err := NewMessageBroker(brokerCfg)
	if err != nil {
		t.Fatalf("NewMessageBroker: %v", err)
	}
	if err := broker.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer broker.Disconnect(context.Background())

	// 启动订阅者
	sub, err := broker.CreateSubscriber(SubscriberConfig{Topics: []string{queue}, ConsumerGroup: "cg-chaos"})
	if err != nil {
		t.Fatalf("CreateSubscriber: %v", err)
	}
	defer sub.Close()

	received := make(chan *Message, 1)
	go func() {
		_ = sub.Subscribe(ctx, MessageHandlerFunc(func(ctx context.Context, m *Message) error {
			select {
			case received <- m:
			default:
			}
			return nil
		}))
	}()

	pub, err := broker.CreatePublisher(PublisherConfig{Topic: queue})
	if err != nil {
		t.Fatalf("CreatePublisher: %v", err)
	}
	defer pub.Close()

	// 先发一条，确保订阅工作
	if err := pub.Publish(ctx, &Message{Topic: queue, Payload: []byte("before")}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	select {
	case <-received:
	case <-time.After(5 * time.Second):
		t.Fatalf("did not receive initial message")
	}

	// Chaos：停止 RabbitMQ 短暂一段时间
	if err := h.StopService(ctx, "rabbitmq"); err != nil {
		t.Fatalf("stop service: %v", err)
	}
	time.Sleep(2 * time.Second)
	if err := h.StartService(ctx, "rabbitmq"); err != nil {
		t.Fatalf("start service: %v", err)
	}

	// 等待适配器重连，然后再次发布
	time.Sleep(3 * time.Second)
	if err := pub.Publish(ctx, &Message{Topic: queue, Payload: []byte("after")}); err != nil {
		t.Fatalf("Publish(after): %v", err)
	}
	select {
	case <-received:
	case <-time.After(10 * time.Second):
		t.Fatalf("did not receive message after broker restart")
	}
}
