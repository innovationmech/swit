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

package kafka

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	compose "github.com/innovationmech/swit/pkg/messaging/testutil/compose"
	"github.com/segmentio/kafka-go"
)

func TestKafkaIntegration_PubSub_HappyPath(t *testing.T) {
	h := compose.NewHarness(
		compose.WithServices("kafka"),
		compose.WithProjectName("swit-kafka-it-ps"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := h.Start(ctx); err != nil {
		t.Fatalf("failed to start compose harness: %v", err)
	}
	defer h.Stop(context.Background())

	endpoints := h.Endpoints()

	brokerCfg := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeKafka,
		Endpoints: []string{endpoints.Kafka},
	}

	broker, err := messaging.NewMessageBroker(brokerCfg)
	if err != nil {
		t.Fatalf("NewMessageBroker: %v", err)
	}

	if err := broker.Connect(ctx); err != nil {
		t.Fatalf("broker.Connect: %v", err)
	}
	defer broker.Disconnect(context.Background())

	topic := "swit-kafka-it-orders"

	// Ensure topic exists
	if err := createKafkaTopic(ctx, endpoints.Kafka, topic, 1); err != nil {
		t.Fatalf("create topic: %v", err)
	}

	pubCfg := messaging.PublisherConfig{Topic: topic}
	publisher, err := broker.CreatePublisher(pubCfg)
	if err != nil {
		t.Fatalf("CreatePublisher: %v", err)
	}
	defer publisher.Close()

	// Send a bootstrap message before starting subscriber
	_ = publisher.Publish(ctx, &messaging.Message{Payload: []byte("bootstrap")})

	subCfg := messaging.SubscriberConfig{Topics: []string{topic}, ConsumerGroup: "cg-integration"}
	subscriber, err := broker.CreateSubscriber(subCfg)
	if err != nil {
		t.Fatalf("CreateSubscriber: %v", err)
	}
	defer subscriber.Close()

	received := make(chan *messaging.Message, 1)
	go func() {
		_ = subscriber.Subscribe(ctx, messaging.MessageHandlerFunc(func(ctx context.Context, m *messaging.Message) error {
			if string(m.Payload) == "bootstrap" {
				return nil // skip bootstrap
			}
			received <- m
			return nil
		}))
	}()

	// Give consumer group time to join and assign partitions
	time.Sleep(2 * time.Second)

	payload := map[string]any{"order_id": "o-1", "user": map[string]any{"id": "u-9"}}
	b, _ := json.Marshal(payload)
	msg := &messaging.Message{Payload: b}
	// Try a few times to avoid race with group assignment
	for i := 0; i < 3; i++ {
		if err := publisher.Publish(ctx, msg); err != nil {
			t.Fatalf("Publish: %v", err)
		}
		select {
		case got := <-received:
			if string(got.Payload) != string(b) {
				t.Fatalf("unexpected payload: %s", string(got.Payload))
			}
			return
		case <-time.After(5 * time.Second):
			// retry publish
		}
	}

	t.Fatal("timeout waiting for message")
}

func TestKafkaIntegration_Transactional_PublishCommit(t *testing.T) {
	h := compose.NewHarness(
		compose.WithServices("kafka"),
		compose.WithProjectName("swit-kafka-it-tx"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := h.Start(ctx); err != nil {
		t.Fatalf("failed to start compose harness: %v", err)
	}
	defer h.Stop(context.Background())

	endpoints := h.Endpoints()

	brokerCfg := &messaging.BrokerConfig{Type: messaging.BrokerTypeKafka, Endpoints: []string{endpoints.Kafka}}
	broker, err := messaging.NewMessageBroker(brokerCfg)
	if err != nil {
		t.Fatalf("NewMessageBroker: %v", err)
	}
	if err := broker.Connect(ctx); err != nil {
		t.Fatalf("broker.Connect: %v", err)
	}
	defer broker.Disconnect(context.Background())

	topic := "swit-kafka-it-tx"
	if err := createKafkaTopic(ctx, endpoints.Kafka, topic, 1); err != nil {
		t.Fatalf("create topic: %v", err)
	}
	pubCfg := messaging.PublisherConfig{Topic: topic, Transactional: true}
	publisher, err := broker.CreatePublisher(pubCfg)
	if err != nil {
		t.Fatalf("CreatePublisher: %v", err)
	}
	defer publisher.Close()

	// Pre-create topic with non-transactional publish to avoid readiness races
	_ = publisher.Publish(ctx, &messaging.Message{Payload: []byte("bootstrap")})

	subCfg := messaging.SubscriberConfig{Topics: []string{topic}, ConsumerGroup: "cg-itx"}
	subscriber, err := broker.CreateSubscriber(subCfg)
	if err != nil {
		t.Fatalf("CreateSubscriber: %v", err)
	}
	defer subscriber.Close()

	received := make(chan *messaging.Message, 1)
	go func() {
		_ = subscriber.Subscribe(ctx, messaging.MessageHandlerFunc(func(ctx context.Context, m *messaging.Message) error {
			if string(m.Payload) == "bootstrap" {
				return nil
			}
			received <- m
			return nil
		}))
	}()

	time.Sleep(2 * time.Second)

	tx, err := publisher.BeginTransaction(ctx)
	if err != nil {
		t.Fatalf("BeginTransaction: %v", err)
	}
	p1 := &messaging.Message{Payload: []byte("t1")}
	if err := tx.Publish(ctx, p1); err != nil {
		t.Fatalf("tx.Publish: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("tx.Commit: %v", err)
	}

	for i := 0; i < 3; i++ {
		select {
		case got := <-received:
			if string(got.Payload) != "t1" {
				t.Fatalf("unexpected tx payload: %s", string(got.Payload))
			}
			return
		case <-time.After(5 * time.Second):
			// retry commit path by publishing additional message in new tx
			if tx2, err := publisher.BeginTransaction(ctx); err == nil {
				_ = tx2.Publish(ctx, &messaging.Message{Payload: []byte("t1")})
				_ = tx2.Commit(ctx)
			}
		}
	}
	t.Fatal("timeout waiting for tx message")
}

func createKafkaTopic(ctx context.Context, bootstrap, topic string, partitions int) error {
	// Connect to cluster
	conn, err := kafka.DialContext(ctx, "tcp", bootstrap)
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	controllerAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	cconn, err := kafka.DialContext(ctx, "tcp", controllerAddr)
	if err != nil {
		return err
	}
	defer cconn.Close()

	cfg := kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     partitions,
		ReplicationFactor: 1,
	}
	return cconn.CreateTopics(cfg)
}
