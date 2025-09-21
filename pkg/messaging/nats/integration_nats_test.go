//go:build integration
// +build integration

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

package nats

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	compose "github.com/innovationmech/swit/pkg/messaging/testutil/compose"
	natsgo "github.com/nats-io/nats.go"
)

func TestNATSIntegration_JetStream_PublishSubscribe(t *testing.T) {
	h := compose.NewHarness(
		compose.WithServices("nats"),
		compose.WithProjectName("swit-nats-it-js"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := h.Start(ctx); err != nil {
		t.Fatalf("failed to start compose harness: %v", err)
	}
	defer h.Stop(context.Background())

	endpoints := h.Endpoints()

	// Configure JetStream stream and consumer
	brokerCfg := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeNATS,
		Endpoints: []string{endpoints.NATS},
		Extra: map[string]any{
			"nats": map[string]any{
				"jetstream": map[string]any{
					"enabled": true,
					"streams": []map[string]any{
						{
							"name":      "IT",
							"subjects":  []string{"it.js.>"},
							"retention": "limits",
							"storage":   "file",
							"replicas":  1,
						},
					},
					"consumers": []map[string]any{
						{
							"name":            "cg-js",
							"stream":          "IT",
							"durable":         true,
							"deliver_policy":  "all",
							"ack_policy":      "explicit",
							"filter_subject":  "it.js.orders",
							"max_ack_pending": 128,
							"ack_wait":        "5s",
						},
					},
				},
			},
		},
	}

	broker, err := messaging.NewMessageBroker(brokerCfg)
	if err != nil {
		t.Fatalf("NewMessageBroker: %v", err)
	}
	if err := broker.Connect(ctx); err != nil {
		t.Fatalf("broker.Connect: %v", err)
	}
	defer broker.Disconnect(context.Background())

	subject := "it.js.orders"

	// Publisher (JetStream path is enabled via broker topology)
	publisher, err := broker.CreatePublisher(messaging.PublisherConfig{Topic: subject})
	if err != nil {
		t.Fatalf("CreatePublisher: %v", err)
	}
	defer publisher.Close()

	// Publish a message before subscriber starts to assert persistence
	_ = publisher.Publish(ctx, &messaging.Message{Topic: subject, Payload: []byte("bootstrap")})

	// Subscriber (durable consumer group matches configured JS consumer name)
	subCfg := messaging.SubscriberConfig{Topics: []string{subject}, ConsumerGroup: "cg-js"}
	subscriber, err := broker.CreateSubscriber(subCfg)
	if err != nil {
		t.Fatalf("CreateSubscriber: %v", err)
	}
	defer subscriber.Close()

	received := make(chan *messaging.Message, 1)
	go func() {
		_ = subscriber.Subscribe(ctx, messaging.MessageHandlerFunc(func(ctx context.Context, m *messaging.Message) error {
			if string(m.Payload) == "bootstrap" {
				return nil // skip bootstrap message
			}
			select {
			case received <- m:
			default:
			}
			return nil
		}))
	}()

	// Allow consumer to initialize and JS topology to settle
	time.Sleep(500 * time.Millisecond)

	payload := map[string]any{"order_id": "o-1001"}
	b, _ := json.Marshal(payload)
	if err := publisher.Publish(ctx, &messaging.Message{Topic: subject, Payload: b}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case got := <-received:
		if string(got.Payload) != string(b) {
			t.Fatalf("unexpected payload: %s", string(got.Payload))
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for JetStream delivery")
	}
}

func TestNATSIntegration_QueueGroup_LoadBalancing(t *testing.T) {
	h := compose.NewHarness(
		compose.WithServices("nats"),
		compose.WithProjectName("swit-nats-it-qg"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := h.Start(ctx); err != nil {
		t.Fatalf("failed to start compose harness: %v", err)
	}
	defer h.Stop(context.Background())

	endpoints := h.Endpoints()

	// Core NATS (no JetStream) to exercise queue groups
	brokerCfg := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeNATS,
		Endpoints: []string{endpoints.NATS},
	}

	broker, err := messaging.NewMessageBroker(brokerCfg)
	if err != nil {
		t.Fatalf("NewMessageBroker: %v", err)
	}
	if err := broker.Connect(ctx); err != nil {
		t.Fatalf("broker.Connect: %v", err)
	}
	defer broker.Disconnect(context.Background())

	subject := "it.qg.orders"
	group := "order-workers"

	// Two subscribers in the same queue group
	subCfg := messaging.SubscriberConfig{Topics: []string{subject}, ConsumerGroup: group}
	s1, err := broker.CreateSubscriber(subCfg)
	if err != nil {
		t.Fatalf("CreateSubscriber s1: %v", err)
	}
	defer s1.Close()
	s2, err := broker.CreateSubscriber(subCfg)
	if err != nil {
		t.Fatalf("CreateSubscriber s2: %v", err)
	}
	defer s2.Close()

	var c1, c2 int32
	go func() {
		_ = s1.Subscribe(ctx, messaging.MessageHandlerFunc(func(ctx context.Context, m *messaging.Message) error { atomic.AddInt32(&c1, 1); return nil }))
	}()
	go func() {
		_ = s2.Subscribe(ctx, messaging.MessageHandlerFunc(func(ctx context.Context, m *messaging.Message) error { atomic.AddInt32(&c2, 1); return nil }))
	}()

	// Publisher
	pub, err := broker.CreatePublisher(messaging.PublisherConfig{Topic: subject})
	if err != nil {
		t.Fatalf("CreatePublisher: %v", err)
	}
	defer pub.Close()

	time.Sleep(300 * time.Millisecond)

	// Publish a batch of messages; expect load balanced deliveries
	total := 20
	for i := 0; i < total; i++ {
		if err := pub.Publish(ctx, &messaging.Message{Topic: subject, Payload: []byte("msg")}); err != nil {
			t.Fatalf("Publish: %v", err)
		}
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if int(atomic.LoadInt32(&c1)+atomic.LoadInt32(&c2)) >= total {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	got1 := atomic.LoadInt32(&c1)
	got2 := atomic.LoadInt32(&c2)
	if int(got1+got2) < total {
		t.Fatalf("expected %d deliveries, got %d", total, got1+got2)
	}
	if got1 == 0 || got2 == 0 {
		t.Fatalf("expected load balancing across subscribers, got c1=%d c2=%d", got1, got2)
	}
}

func TestNATSIntegration_RequestReply_SuccessAndTimeout(t *testing.T) {
	h := compose.NewHarness(
		compose.WithServices("nats"),
		compose.WithProjectName("swit-nats-it-rr"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := h.Start(ctx); err != nil {
		t.Fatalf("failed to start compose harness: %v", err)
	}
	defer h.Stop(context.Background())

	endpoints := h.Endpoints()

	// Raw NATS connection for request-reply helpers
	nc, err := natsgo.Connect(endpoints.NATS, natsgo.Timeout(3*time.Second))
	if err != nil {
		t.Fatalf("nats.Connect: %v", err)
	}
	defer nc.Drain()

	// Start a simple echo server
	srv := NewRequestReplyServer(nc)
	sub, err := srv.Handle("it.rr.echo", func(data []byte) ([]byte, error) {
		if string(data) == "error" {
			return nil, errors.New("forced")
		}
		return []byte("pong"), nil
	})
	if err != nil {
		t.Fatalf("RequestReplyServer.Handle: %v", err)
	}
	defer sub.Unsubscribe()

	cfg := DefaultConfig()
	cfg.Timeouts.Request = messaging.Duration(500 * time.Millisecond)
	client := NewRequestReplyClient(nc, cfg)

	// Success path
	data, err := client.Request(context.Background(), "it.rr.echo", []byte("ping"))
	if err != nil {
		t.Fatalf("Request (success): %v", err)
	}
	if string(data) != "pong" {
		t.Fatalf("unexpected reply: %s", string(data))
	}

	// Timeout path: slow server causes client deadline exceed
	slowSub, err := srv.Handle("it.rr.slow", func(data []byte) ([]byte, error) {
		time.Sleep(800 * time.Millisecond)
		return []byte("slow"), nil
	})
	if err != nil {
		t.Fatalf("RequestReplyServer.Handle (slow): %v", err)
	}
	defer slowSub.Unsubscribe()

	tctx, tcancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer tcancel()
	_, err = client.Request(tctx, "it.rr.slow", []byte("x"))
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	var me *messaging.MessagingError
	if !errors.As(err, &me) || me.Code != messaging.ErrProcessingTimeout {
		t.Fatalf("expected processing timeout error code, got: %v", err)
	}
}
