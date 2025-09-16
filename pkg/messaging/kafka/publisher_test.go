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
	"errors"
	"sync"
	"testing"

	"github.com/innovationmech/swit/pkg/messaging"
)

type transactionRecordingWriter struct {
	mu     sync.Mutex
	writes [][]KafkaMessage
}

func (w *transactionRecordingWriter) WriteMessages(ctx context.Context, msgs ...KafkaMessage) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	copied := make([]KafkaMessage, len(msgs))
	copy(copied, msgs)
	w.writes = append(w.writes, copied)
	return nil
}

func (w *transactionRecordingWriter) Close() error { return nil }

func TestKafkaPublisherBeginTransaction_NotSupported(t *testing.T) {
	brokerCfg := &messaging.BrokerConfig{Type: messaging.BrokerTypeKafka, Endpoints: []string{"localhost:9092"}}
	pubCfg := &messaging.PublisherConfig{Topic: "orders", Transactional: false}

	pool, err := newProducerPool(brokerCfg, pubCfg)
	if err != nil {
		t.Fatalf("unexpected error creating producer pool: %v", err)
	}
	publisher, err := newKafkaPublisher(pool, pubCfg, newSchemaRegistryManager())
	if err != nil {
		t.Fatalf("unexpected error creating publisher: %v", err)
	}

	if _, err := publisher.BeginTransaction(context.Background()); !errors.Is(err, messaging.ErrTransactionsNotSupported) {
		t.Fatalf("expected ErrTransactionsNotSupported, got %v", err)
	}
}

func TestKafkaPublisherTransactionCommit(t *testing.T) {
	rw := &transactionRecordingWriter{}
	originalFactory := writerFactory
	writerFactory = func(_ *messaging.BrokerConfig, _ string, _ *messaging.PublisherConfig) (kafkaWriter, error) {
		return rw, nil
	}
	t.Cleanup(func() { writerFactory = originalFactory })

	brokerCfg := &messaging.BrokerConfig{Type: messaging.BrokerTypeKafka, Endpoints: []string{"localhost:9092"}}
	pubCfg := &messaging.PublisherConfig{Topic: "orders", Transactional: true}

	pool, err := newProducerPool(brokerCfg, pubCfg)
	if err != nil {
		t.Fatalf("unexpected error creating producer pool: %v", err)
	}
	publisher, err := newKafkaPublisher(pool, pubCfg, newSchemaRegistryManager())
	if err != nil {
		t.Fatalf("unexpected error creating publisher: %v", err)
	}

	tx, err := publisher.BeginTransaction(context.Background())
	if err != nil {
		t.Fatalf("unexpected error starting transaction: %v", err)
	}

	message := &messaging.Message{ID: "1", Payload: []byte("hello")}
	if err := tx.Publish(context.Background(), message); err != nil {
		t.Fatalf("unexpected error publishing in transaction: %v", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("unexpected error committing transaction: %v", err)
	}

	rw.mu.Lock()
	if len(rw.writes) != 1 {
		t.Fatalf("expected 1 batch write, got %d", len(rw.writes))
	}
	if len(rw.writes[0]) != 1 {
		t.Fatalf("expected 1 message in batch, got %d", len(rw.writes[0]))
	}
	km := rw.writes[0][0]
	rw.mu.Unlock()

	if km.Headers[transactionIDHeader] == "" {
		t.Error("expected transaction id header to be set")
	}
	if km.Headers[transactionSeqHeader] != "1" {
		t.Errorf("expected transaction sequence header '1', got %q", km.Headers[transactionSeqHeader])
	}
	if km.Headers[transactionProducerHdr] == "" {
		t.Error("expected producer header to be set")
	}

	metrics := publisher.GetMetrics()
	if metrics.TransactionsStarted != 1 {
		t.Errorf("expected TransactionsStarted=1, got %d", metrics.TransactionsStarted)
	}
	if metrics.TransactionsCommitted != 1 {
		t.Errorf("expected TransactionsCommitted=1, got %d", metrics.TransactionsCommitted)
	}
	if metrics.TransactionsAborted != 0 {
		t.Errorf("expected TransactionsAborted=0, got %d", metrics.TransactionsAborted)
	}
}

func TestKafkaPublisherTransactionRollback(t *testing.T) {
	rw := &transactionRecordingWriter{}
	originalFactory := writerFactory
	writerFactory = func(_ *messaging.BrokerConfig, _ string, _ *messaging.PublisherConfig) (kafkaWriter, error) {
		return rw, nil
	}
	t.Cleanup(func() { writerFactory = originalFactory })

	brokerCfg := &messaging.BrokerConfig{Type: messaging.BrokerTypeKafka, Endpoints: []string{"localhost:9092"}}
	pubCfg := &messaging.PublisherConfig{Topic: "orders", Transactional: true}

	pool, err := newProducerPool(brokerCfg, pubCfg)
	if err != nil {
		t.Fatalf("unexpected error creating producer pool: %v", err)
	}
	publisher, err := newKafkaPublisher(pool, pubCfg, newSchemaRegistryManager())
	if err != nil {
		t.Fatalf("unexpected error creating publisher: %v", err)
	}

	tx, err := publisher.BeginTransaction(context.Background())
	if err != nil {
		t.Fatalf("unexpected error starting transaction: %v", err)
	}

	if err := tx.Publish(context.Background(), &messaging.Message{ID: "1", Payload: []byte("hello")}); err != nil {
		t.Fatalf("unexpected error publishing in transaction: %v", err)
	}

	if err := tx.Rollback(context.Background()); err != nil {
		t.Fatalf("unexpected error rolling back transaction: %v", err)
	}

	rw.mu.Lock()
	writes := len(rw.writes)
	rw.mu.Unlock()
	if writes != 0 {
		t.Fatalf("expected no writes after rollback, got %d", writes)
	}

	metrics := publisher.GetMetrics()
	if metrics.TransactionsAborted != 1 {
		t.Errorf("expected TransactionsAborted=1, got %d", metrics.TransactionsAborted)
	}

	// Ensure we can start a new transaction after rollback
	if _, err := publisher.BeginTransaction(context.Background()); err != nil {
		t.Fatalf("expected to start new transaction after rollback, got %v", err)
	}
}

func TestKafkaPublisherTransactionConcurrent(t *testing.T) {
	brokerCfg := &messaging.BrokerConfig{Type: messaging.BrokerTypeKafka, Endpoints: []string{"localhost:9092"}}
	pubCfg := &messaging.PublisherConfig{Topic: "orders", Transactional: true}

	pool, err := newProducerPool(brokerCfg, pubCfg)
	if err != nil {
		t.Fatalf("unexpected error creating producer pool: %v", err)
	}
	publisher, err := newKafkaPublisher(pool, pubCfg, newSchemaRegistryManager())
	if err != nil {
		t.Fatalf("unexpected error creating publisher: %v", err)
	}

	tx, err := publisher.BeginTransaction(context.Background())
	if err != nil {
		t.Fatalf("unexpected error starting transaction: %v", err)
	}

	if _, err := publisher.BeginTransaction(context.Background()); err == nil {
		t.Fatal("expected error when starting second transaction while first active")
	}

	if err := tx.Rollback(context.Background()); err != nil {
		t.Fatalf("unexpected error rolling back transaction: %v", err)
	}
}
