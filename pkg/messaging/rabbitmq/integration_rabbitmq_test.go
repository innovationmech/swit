//go:build integration
// +build integration

package rabbitmq

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	compose "github.com/innovationmech/swit/pkg/messaging/testutil/compose"
)

// alwaysRetryHandler implements MessageHandler that always requests retry on error.
type alwaysRetryHandler struct{ counter *int32 }

func (h alwaysRetryHandler) Handle(ctx context.Context, m *messaging.Message) error {
    if h.counter != nil {
        atomic.AddInt32(h.counter, 1)
    }
    return messaging.NewProcessingError("fail", nil)
}

func (alwaysRetryHandler) OnError(ctx context.Context, m *messaging.Message, err error) messaging.ErrorAction {
    return messaging.ErrorActionRetry
}

// newRabbitBrokerConfig constructs a BrokerConfig with topology declaring the base queue.
func newRabbitBrokerConfig(endpoint, queue string) *messaging.BrokerConfig {
	cfg := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{endpoint},
	}
	// Minimal topology to ensure the base queue exists.
	cfg.Extra = map[string]any{
		"rabbitmq": map[string]any{
			"topology": map[string]any{
				"queues": map[string]any{
					queue: map[string]any{
						"durable":     true,
						"auto_delete": false,
						"exclusive":   false,
					},
				},
			},
		},
	}
	return cfg
}

func TestRabbitMQIntegration_PublishWithConfirm_Ack(t *testing.T) {
	h := compose.NewHarness(
		compose.WithServices("rabbitmq"),
		compose.WithProjectName("swit-rabbit-it-confirm"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := h.Start(ctx); err != nil {
		t.Fatalf("failed to start compose harness: %v", err)
	}
	defer h.Stop(context.Background())

	endpoints := h.Endpoints()
	queue := "swit-rabbit-it-confirm-q"

	brokerCfg := newRabbitBrokerConfig(endpoints.Rabbit, queue)
	broker, err := messaging.NewMessageBroker(brokerCfg)
	if err != nil {
		t.Fatalf("NewMessageBroker: %v", err)
	}
	if err := broker.Connect(ctx); err != nil {
		t.Fatalf("broker.Connect: %v", err)
	}
	defer broker.Disconnect(context.Background())

	pubCfg := messaging.PublisherConfig{
		Topic: queue,
		Confirmation: messaging.ConfirmationConfig{
			Required: true,
			Timeout:  2 * time.Second,
		},
	}
	publisher, err := broker.CreatePublisher(pubCfg)
	if err != nil {
		t.Fatalf("CreatePublisher: %v", err)
	}
	defer publisher.Close()

	msg := &messaging.Message{ID: "confirm-ack-1", Topic: queue, Payload: []byte("payload")}
	conf, err := publisher.PublishWithConfirm(ctx, msg)
	if err != nil {
		t.Fatalf("PublishWithConfirm: %v", err)
	}
	if conf == nil || conf.MessageID != "confirm-ack-1" {
		t.Fatalf("unexpected confirmation: %#v", conf)
	}
}

func TestRabbitMQIntegration_Consumer_AutoAck_NoRedeliveryOnError(t *testing.T) {
	h := compose.NewHarness(
		compose.WithServices("rabbitmq"),
		compose.WithProjectName("swit-rabbit-it-autoack"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := h.Start(ctx); err != nil {
		t.Fatalf("failed to start compose harness: %v", err)
	}
	defer h.Stop(context.Background())

	endpoints := h.Endpoints()
	queue := "swit-rabbit-it-autoack-q"

	brokerCfg := newRabbitBrokerConfig(endpoints.Rabbit, queue)
	broker, err := messaging.NewMessageBroker(brokerCfg)
	if err != nil {
		t.Fatalf("NewMessageBroker: %v", err)
	}
	if err := broker.Connect(ctx); err != nil {
		t.Fatalf("broker.Connect: %v", err)
	}
	defer broker.Disconnect(context.Background())

	// Start subscriber with auto-ack
	subCfg := messaging.SubscriberConfig{
		Topics:        []string{queue},
		ConsumerGroup: "cg-auto",
		Processing: messaging.ProcessingConfig{
			AckMode: messaging.AckModeAuto,
		},
	}
	subscriber, err := broker.CreateSubscriber(subCfg)
	if err != nil {
		t.Fatalf("CreateSubscriber: %v", err)
	}
	defer subscriber.Close()

	var deliveries int32
	go func() {
		_ = subscriber.Subscribe(ctx, messaging.MessageHandlerFunc(func(ctx context.Context, m *messaging.Message) error {
			atomic.AddInt32(&deliveries, 1)
			// Simulate processing error; with auto-ack it should NOT trigger redelivery
			return messaging.NewProcessingError("fail", nil)
		}))
	}()

	// Publisher
	publisher, err := broker.CreatePublisher(messaging.PublisherConfig{Topic: queue})
	if err != nil {
		t.Fatalf("CreatePublisher: %v", err)
	}
	defer publisher.Close()

	// Give consumer time to start
	time.Sleep(300 * time.Millisecond)

	payload := map[string]any{"case": "auto-ack"}
	b, _ := json.Marshal(payload)
	if err := publisher.Publish(ctx, &messaging.Message{Topic: queue, Payload: b}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Observe for a while to ensure no redelivery happens
	time.Sleep(2 * time.Second)
	if c := atomic.LoadInt32(&deliveries); c != 1 {
		t.Fatalf("expected exactly 1 delivery with auto-ack, got %d", c)
	}
}

func TestRabbitMQIntegration_Consumer_ManualAck_RetryThenDLQ(t *testing.T) {
	h := compose.NewHarness(
		compose.WithServices("rabbitmq"),
		compose.WithProjectName("swit-rabbit-it-retry-dlq"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := h.Start(ctx); err != nil {
		t.Fatalf("failed to start compose harness: %v", err)
	}
	defer h.Stop(context.Background())

	endpoints := h.Endpoints()
	baseQueue := "swit-rabbit-it-retry-q"
	dlq := baseQueue + ".dlq"

	brokerCfg := newRabbitBrokerConfig(endpoints.Rabbit, baseQueue)
	broker, err := messaging.NewMessageBroker(brokerCfg)
	if err != nil {
		t.Fatalf("NewMessageBroker: %v", err)
	}
	if err := broker.Connect(ctx); err != nil {
		t.Fatalf("broker.Connect: %v", err)
	}
	defer broker.Disconnect(context.Background())

	// Manual ACK with retry then DLQ (MaxRetries=2 → one retry, then DLQ)
	subCfg := messaging.SubscriberConfig{
		Topics:        []string{baseQueue},
		ConsumerGroup: "cg-manual",
		Processing: messaging.ProcessingConfig{
			AckMode: messaging.AckModeManual,
		},
	}
	subCfg.DeadLetter.MaxRetries = 2
	subCfg.Retry.InitialDelay = 200 * time.Millisecond

	subscriber, err := broker.CreateSubscriber(subCfg)
	if err != nil {
		t.Fatalf("CreateSubscriber: %v", err)
	}
	defer subscriber.Close()

	// DLQ subscriber to detect final dead-letter
	dlqSubCfg := messaging.SubscriberConfig{Topics: []string{dlq}, ConsumerGroup: "cg-dlq"}
	dlqSubscriber, err := broker.CreateSubscriber(dlqSubCfg)
	if err != nil {
		t.Fatalf("CreateSubscriber (dlq): %v", err)
	}
	defer dlqSubscriber.Close()

    var processed int32
    dlqArrived := make(chan *messaging.Message, 1)

    // Always request retry on error; subscriber will enforce MaxRetries then DLQ
    retryH := alwaysRetryHandler{counter: &processed}
    go func() { _ = subscriber.Subscribe(ctx, retryH) }()

	go func() {
		_ = dlqSubscriber.Subscribe(ctx, messaging.MessageHandlerFunc(func(ctx context.Context, m *messaging.Message) error {
			select {
			case dlqArrived <- m:
			default:
			}
			return nil
		}))
	}()

	publisher, err := broker.CreatePublisher(messaging.PublisherConfig{Topic: baseQueue})
	if err != nil {
		t.Fatalf("CreatePublisher: %v", err)
	}
	defer publisher.Close()

	// Allow consumers to start
	time.Sleep(300 * time.Millisecond)

	msg := &messaging.Message{Topic: baseQueue, Payload: []byte("need-retry")}
	if err := publisher.Publish(ctx, msg); err != nil {
		t.Fatalf("Publish: %v", err)
	}

    select {
    case <-dlqArrived:
        // Wait briefly for the second processing attempt to be counted
        deadline := time.Now().Add(2 * time.Second)
        for {
            if n := atomic.LoadInt32(&processed); n >= 2 {
                break
            }
            if time.Now().After(deadline) {
                t.Fatalf("expected >=2 processing attempts before DLQ, got %d", atomic.LoadInt32(&processed))
            }
            time.Sleep(50 * time.Millisecond)
        }
    case <-time.After(10 * time.Second):
        t.Fatal("timeout waiting for message to arrive on DLQ")
    }
}
