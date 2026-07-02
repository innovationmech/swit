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

package outbox_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	_ "github.com/lib/pq" // PostgreSQL driver for the gated E2E test
	"github.com/redis/go-redis/v9"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/messaging/inmemory"
	"github.com/innovationmech/swit/pkg/patterns/outbox"
)

// newBrokerPipeline 启动 in-memory broker 并返回真实的 EventPublisher
// 与一个接收指定 topic 消息的 channel。
func newBrokerPipeline(t *testing.T, topic string) (messaging.EventPublisher, <-chan *messaging.Message) {
	t.Helper()

	broker := inmemory.New(nil)
	if err := broker.Connect(context.Background()); err != nil {
		t.Fatalf("failed to connect broker: %v", err)
	}
	t.Cleanup(func() { _ = broker.Close() })

	publisher, err := broker.CreatePublisher(messaging.PublisherConfig{Topic: topic})
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}

	subscriber, err := broker.CreateSubscriber(messaging.SubscriberConfig{
		Topics:        []string{topic},
		ConsumerGroup: "e2e-consumers",
	})
	if err != nil {
		t.Fatalf("failed to create subscriber: %v", err)
	}

	received := make(chan *messaging.Message, 16)
	handler := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		received <- msg
		return nil
	})
	if err := subscriber.Subscribe(context.Background(), handler); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	t.Cleanup(func() { _ = subscriber.Close() })

	return publisher, received
}

func waitForMessage(t *testing.T, received <-chan *messaging.Message) *messaging.Message {
	t.Helper()
	select {
	case msg := <-received:
		return msg
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for message from broker")
		return nil
	}
}

// TestE2E_TransactionalOutbox_InMemory 验证完整链路：
// 业务事务中保存 outbox 条目 -> processor 通过真实的
// messaging.EventPublisher 发布 -> 订阅方从 broker 收到消息。
func TestE2E_TransactionalOutbox_InMemory(t *testing.T) {
	ctx := context.Background()
	const topic = "orders.events"

	storage := outbox.NewInMemoryStorage()
	outboxPublisher := outbox.NewPublisher(storage)
	brokerPublisher, received := newBrokerPipeline(t, topic)

	// 1. 在业务事务中保存 outbox 条目
	tx, err := storage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	entry := &outbox.OutboxEntry{
		ID:          "order-created-1",
		AggregateID: "order-1",
		EventType:   "order.created",
		Topic:       topic,
		Payload:     []byte(`{"order_id":"order-1"}`),
		Headers:     map[string]string{"source": "e2e-test"},
	}
	if err := outboxPublisher.SaveWithTransaction(ctx, tx, entry); err != nil {
		t.Fatalf("SaveWithTransaction() error = %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("tx.Commit() error = %v", err)
	}

	// 2. processor 将 outbox 条目发布到 broker
	processor := outbox.NewProcessor(storage, brokerPublisher, outbox.ProcessorConfig{
		PollInterval: 50 * time.Millisecond,
		BatchSize:    10,
		MaxRetries:   3,
	})
	if err := processor.ProcessOnce(ctx); err != nil {
		t.Fatalf("ProcessOnce() error = %v", err)
	}

	// 3. 订阅方收到消息
	msg := waitForMessage(t, received)
	if msg.ID != entry.ID {
		t.Errorf("expected message ID %q, got %q", entry.ID, msg.ID)
	}
	if string(msg.Payload) != string(entry.Payload) {
		t.Errorf("unexpected payload: %s", msg.Payload)
	}
	if msg.Headers["aggregate_id"] != "order-1" || msg.Headers["event_type"] != "order.created" {
		t.Errorf("expected outbox metadata headers, got %v", msg.Headers)
	}
	if msg.Headers["source"] != "e2e-test" {
		t.Errorf("expected custom header to propagate, got %v", msg.Headers)
	}

	// 4. 条目被标记为已处理，不会重复发布
	unprocessed, err := storage.FetchUnprocessed(ctx, 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed() error = %v", err)
	}
	if len(unprocessed) != 0 {
		t.Errorf("expected no unprocessed entries, got %d", len(unprocessed))
	}
}

// TestE2E_TransactionalOutbox_Redis 验证 Redis 存储的完整链路。
func TestE2E_TransactionalOutbox_Redis(t *testing.T) {
	ctx := context.Background()
	const topic = "payments.events"

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	storage, err := outbox.NewRedisStorage(client)
	if err != nil {
		t.Fatalf("NewRedisStorage() error = %v", err)
	}
	outboxPublisher := outbox.NewPublisher(storage)
	brokerPublisher, received := newBrokerPipeline(t, topic)

	tx, err := storage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	entry := &outbox.OutboxEntry{
		ID:          "payment-1",
		AggregateID: "pay-1",
		EventType:   "payment.completed",
		Topic:       topic,
		Payload:     []byte(`{"amount":100}`),
	}
	if err := outboxPublisher.SaveWithTransaction(ctx, tx, entry); err != nil {
		t.Fatalf("SaveWithTransaction() error = %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("tx.Commit() error = %v", err)
	}

	processor := outbox.NewProcessor(storage, brokerPublisher, outbox.DefaultProcessorConfig())
	if err := processor.ProcessOnce(ctx); err != nil {
		t.Fatalf("ProcessOnce() error = %v", err)
	}

	msg := waitForMessage(t, received)
	if msg.ID != "payment-1" || msg.Headers["event_type"] != "payment.completed" {
		t.Errorf("unexpected message: id=%s headers=%v", msg.ID, msg.Headers)
	}

	unprocessed, err := storage.FetchUnprocessed(ctx, 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed() error = %v", err)
	}
	if len(unprocessed) != 0 {
		t.Errorf("expected no unprocessed entries, got %d", len(unprocessed))
	}
}

// TestE2E_TransactionalOutbox_BackgroundProcessor 验证后台轮询模式：
// Start 之后 processor 自动发布新保存的条目。
func TestE2E_TransactionalOutbox_BackgroundProcessor(t *testing.T) {
	ctx := context.Background()
	const topic = "inventory.events"

	storage := outbox.NewInMemoryStorage()
	brokerPublisher, received := newBrokerPipeline(t, topic)

	processor := outbox.NewProcessor(storage, brokerPublisher, outbox.ProcessorConfig{
		PollInterval: 20 * time.Millisecond,
		BatchSize:    10,
		MaxRetries:   3,
	})
	if err := processor.Start(ctx); err != nil {
		t.Fatalf("processor.Start() error = %v", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = processor.Stop(stopCtx)
	})

	outboxPublisher := outbox.NewPublisher(storage)
	entry := &outbox.OutboxEntry{
		ID:      "stock-1",
		Topic:   topic,
		Payload: []byte(`{"sku":"A"}`),
	}
	if err := outboxPublisher.SaveForPublish(ctx, entry); err != nil {
		t.Fatalf("SaveForPublish() error = %v", err)
	}

	msg := waitForMessage(t, received)
	if msg.ID != "stock-1" {
		t.Errorf("expected message stock-1, got %s", msg.ID)
	}
}

// TestE2E_TransactionalOutbox_Postgres 使用真实 PostgreSQL 验证 DB 落盘 +
// 可靠发布。需要设置 POSTGRES_TEST_DSN 环境变量，否则跳过。
//
// 示例：
//
//	export POSTGRES_TEST_DSN="postgres://postgres:password@localhost:5432/test?sslmode=disable"
func TestE2E_TransactionalOutbox_Postgres(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set, skipping postgres E2E test")
	}

	ctx := context.Background()
	const topic = "shipping.events"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open postgres: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("failed to ping postgres: %v", err)
	}

	tableName := fmt.Sprintf("outbox_e2e_%d", time.Now().UnixNano())
	storage, err := outbox.NewPostgresStorage(db, outbox.WithPostgresTableName(tableName))
	if err != nil {
		t.Fatalf("NewPostgresStorage() error = %v", err)
	}
	if err := storage.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + tableName)
	})

	outboxPublisher := outbox.NewPublisher(storage)
	brokerPublisher, received := newBrokerPipeline(t, topic)

	// 业务数据与 outbox 条目在同一数据库事务中提交
	tx, err := storage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	entry := &outbox.OutboxEntry{
		ID:          "shipment-1",
		AggregateID: "ship-1",
		EventType:   "shipment.created",
		Topic:       topic,
		Payload:     []byte(`{"shipment_id":"ship-1"}`),
	}
	if err := outboxPublisher.SaveWithTransaction(ctx, tx, entry); err != nil {
		t.Fatalf("SaveWithTransaction() error = %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("tx.Commit() error = %v", err)
	}

	// 确认 DB 落盘
	var count int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM "+tableName+" WHERE processed_at IS NULL").Scan(&count); err != nil {
		t.Fatalf("failed to count entries: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 unprocessed entry in DB, got %d", count)
	}

	// processor 发布到 broker
	processor := outbox.NewProcessor(storage, brokerPublisher, outbox.DefaultProcessorConfig())
	if err := processor.ProcessOnce(ctx); err != nil {
		t.Fatalf("ProcessOnce() error = %v", err)
	}

	msg := waitForMessage(t, received)
	if msg.ID != "shipment-1" {
		t.Errorf("expected message shipment-1, got %s", msg.ID)
	}

	// DB 中的条目被标记为已处理
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM "+tableName+" WHERE processed_at IS NOT NULL").Scan(&count); err != nil {
		t.Fatalf("failed to count processed entries: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 processed entry in DB, got %d", count)
	}
}
