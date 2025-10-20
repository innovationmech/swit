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

package messaging

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTransaction implements a mock transaction for testing
type mockTransaction struct {
	id            string
	publishFunc   func(ctx context.Context, message *messaging.Message) error
	commitFunc    func(ctx context.Context) error
	rollbackFunc  func(ctx context.Context) error
	publishedMsgs []*messaging.Message
	committed     bool
	rolledBack    bool
}

func newMockTransaction(id string) *mockTransaction {
	return &mockTransaction{
		id:            id,
		publishedMsgs: make([]*messaging.Message, 0),
	}
}

func (m *mockTransaction) Publish(ctx context.Context, message *messaging.Message) error {
	if m.publishFunc != nil {
		err := m.publishFunc(ctx, message)
		if err == nil {
			m.publishedMsgs = append(m.publishedMsgs, message)
		}
		return err
	}
	m.publishedMsgs = append(m.publishedMsgs, message)
	return nil
}

func (m *mockTransaction) Commit(ctx context.Context) error {
	if m.commitFunc != nil {
		err := m.commitFunc(ctx)
		if err == nil {
			m.committed = true
		}
		return err
	}
	m.committed = true
	return nil
}

func (m *mockTransaction) Rollback(ctx context.Context) error {
	if m.rollbackFunc != nil {
		err := m.rollbackFunc(ctx)
		if err == nil {
			m.rolledBack = true
		}
		return err
	}
	m.rolledBack = true
	return nil
}

func (m *mockTransaction) GetID() string {
	return m.id
}

func TestSagaEventPublisher_PublishWithTransaction(t *testing.T) {
	logger.InitLogger()

	tests := []struct {
		name      string
		events    []*saga.SagaEvent
		setupMock func() (*mockMessageBroker, *mockTransaction)
		config    *PublisherConfig
		wantErr   bool
		errMsg    string
		validate  func(t *testing.T, tx *mockTransaction)
	}{
		{
			name: "successful transaction publish",
			events: []*saga.SagaEvent{
				{
					ID:        "event-tx-1",
					SagaID:    "saga-tx",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
				{
					ID:        "event-tx-2",
					SagaID:    "saga-tx",
					Type:      saga.EventSagaStepCompleted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
			},
			setupMock: func() (*mockMessageBroker, *mockTransaction) {
				tx := newMockTransaction("tx-123")
				mockPub := &mockPublisher{
					beginTransactionFunc: func(ctx context.Context) (messaging.Transaction, error) {
						return tx, nil
					},
				}
				broker := &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
				return broker, tx
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				TopicPrefix:     "saga",
				RetryAttempts:   3,
				RetryInterval:   10 * time.Millisecond,
				Timeout:         30 * time.Second,
			},
			wantErr: false,
			validate: func(t *testing.T, tx *mockTransaction) {
				assert.Equal(t, 2, len(tx.publishedMsgs))
				assert.True(t, tx.committed)
				assert.False(t, tx.rolledBack)
				assert.Equal(t, "event-tx-1", tx.publishedMsgs[0].ID)
				assert.Equal(t, "event-tx-2", tx.publishedMsgs[1].ID)
				assert.Equal(t, "tx-123", tx.publishedMsgs[0].Headers["tx_id"])
			},
		},
		{
			name:   "empty events list",
			events: []*saga.SagaEvent{},
			setupMock: func() (*mockMessageBroker, *mockTransaction) {
				return &mockMessageBroker{}, nil
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "transaction not supported",
			events: []*saga.SagaEvent{
				{
					ID:        "event-1",
					SagaID:    "saga-1",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
			},
			setupMock: func() (*mockMessageBroker, *mockTransaction) {
				mockPub := &mockPublisher{
					beginTransactionFunc: func(ctx context.Context) (messaging.Transaction, error) {
						return nil, errors.New("transactions not supported by broker")
					},
				}
				broker := &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
				return broker, nil
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "begin transaction",
		},
		{
			name: "nil event in transaction",
			events: []*saga.SagaEvent{
				{
					ID:        "event-1",
					SagaID:    "saga-1",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
				nil,
			},
			setupMock: func() (*mockMessageBroker, *mockTransaction) {
				return &mockMessageBroker{}, nil
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "event 1 is nil",
		},
		{
			name: "invalid event in transaction",
			events: []*saga.SagaEvent{
				{
					ID:        "event-1",
					SagaID:    "", // Missing saga ID
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
			},
			setupMock: func() (*mockMessageBroker, *mockTransaction) {
				return &mockMessageBroker{}, nil
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "event 0 validation failed",
		},
		{
			name: "publish fails in transaction",
			events: []*saga.SagaEvent{
				{
					ID:        "event-fail",
					SagaID:    "saga-fail",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
			},
			setupMock: func() (*mockMessageBroker, *mockTransaction) {
				tx := newMockTransaction("tx-fail")
				tx.publishFunc = func(ctx context.Context, message *messaging.Message) error {
					return errors.New("publish failed in transaction")
				}
				mockPub := &mockPublisher{
					beginTransactionFunc: func(ctx context.Context) (messaging.Transaction, error) {
						return tx, nil
					},
				}
				broker := &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
				return broker, tx
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "publish event 0 in transaction",
		},
		{
			name: "commit fails",
			events: []*saga.SagaEvent{
				{
					ID:        "event-commit-fail",
					SagaID:    "saga-commit-fail",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				},
			},
			setupMock: func() (*mockMessageBroker, *mockTransaction) {
				tx := newMockTransaction("tx-commit-fail")
				tx.commitFunc = func(ctx context.Context) error {
					return errors.New("commit failed")
				}
				mockPub := &mockPublisher{
					beginTransactionFunc: func(ctx context.Context) (messaging.Transaction, error) {
						return tx, nil
					},
				}
				broker := &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
				return broker, tx
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			wantErr: true,
			errMsg:  "commit transaction",
		},
		{
			name: "large batch in transaction",
			events: func() []*saga.SagaEvent {
				events := make([]*saga.SagaEvent, 50)
				for i := 0; i < 50; i++ {
					events[i] = &saga.SagaEvent{
						ID:        fmt.Sprintf("event-large-%d", i),
						SagaID:    "saga-large-tx",
						Type:      saga.EventSagaStarted,
						Version:   "1.0",
						Timestamp: time.Now(),
					}
				}
				return events
			}(),
			setupMock: func() (*mockMessageBroker, *mockTransaction) {
				tx := newMockTransaction("tx-large")
				mockPub := &mockPublisher{
					beginTransactionFunc: func(ctx context.Context) (messaging.Transaction, error) {
						return tx, nil
					},
				}
				broker := &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
				return broker, tx
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				TopicPrefix:     "saga",
				RetryAttempts:   3,
				RetryInterval:   10 * time.Millisecond,
				Timeout:         30 * time.Second,
			},
			wantErr: false,
			validate: func(t *testing.T, tx *mockTransaction) {
				assert.Equal(t, 50, len(tx.publishedMsgs))
				assert.True(t, tx.committed)
				assert.False(t, tx.rolledBack)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker, tx := tt.setupMock()
			publisher, err := NewSagaEventPublisher(broker, tt.config)
			require.NoError(t, err)
			defer publisher.Close()

			ctx := context.Background()
			err = publisher.PublishWithTransaction(ctx, tt.events)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// Verify metrics for successful publishes
				if len(tt.events) > 0 {
					metrics := publisher.GetMetrics()
					assert.Equal(t, int64(len(tt.events)), metrics.TotalPublished.Load())

					// Run validation if provided
					if tt.validate != nil && tx != nil {
						tt.validate(t, tx)
					}
				}
			}
		})
	}
}

func TestSagaEventPublisher_PublishWithTransactionWhenClosed(t *testing.T) {
	logger.InitLogger()

	broker := &mockMessageBroker{}
	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)

	// Close the publisher
	err = publisher.Close()
	require.NoError(t, err)

	// Try to publish with transaction after closing
	events := []*saga.SagaEvent{
		{
			ID:        "event-1",
			SagaID:    "saga-1",
			Type:      saga.EventSagaStarted,
			Version:   "1.0",
			Timestamp: time.Now(),
		},
	}

	ctx := context.Background()
	err = publisher.PublishWithTransaction(ctx, events)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publisher is closed")
}

func TestSagaEventPublisher_WithTransaction(t *testing.T) {
	logger.InitLogger()

	tests := []struct {
		name      string
		setupMock func() (*mockMessageBroker, *mockTransaction)
		config    *PublisherConfig
		fn        func(ctx context.Context, txPub *TransactionalEventPublisher) error
		wantErr   bool
		errMsg    string
		validate  func(t *testing.T, tx *mockTransaction)
	}{
		{
			name: "successful WithTransaction",
			setupMock: func() (*mockMessageBroker, *mockTransaction) {
				tx := newMockTransaction("tx-with-123")
				mockPub := &mockPublisher{
					beginTransactionFunc: func(ctx context.Context) (messaging.Transaction, error) {
						return tx, nil
					},
				}
				broker := &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
				return broker, tx
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				TopicPrefix:     "saga",
				Timeout:         30 * time.Second,
			},
			fn: func(ctx context.Context, txPub *TransactionalEventPublisher) error {
				event1 := &saga.SagaEvent{
					ID:        "event-with-1",
					SagaID:    "saga-with",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				}
				if err := txPub.PublishEvent(ctx, event1); err != nil {
					return err
				}

				event2 := &saga.SagaEvent{
					ID:        "event-with-2",
					SagaID:    "saga-with",
					Type:      saga.EventSagaStepCompleted,
					Version:   "1.0",
					Timestamp: time.Now(),
				}
				return txPub.PublishEvent(ctx, event2)
			},
			wantErr: false,
			validate: func(t *testing.T, tx *mockTransaction) {
				assert.Equal(t, 2, len(tx.publishedMsgs))
				assert.True(t, tx.committed)
				assert.False(t, tx.rolledBack)
				assert.Equal(t, "event-with-1", tx.publishedMsgs[0].ID)
				assert.Equal(t, "event-with-2", tx.publishedMsgs[1].ID)
			},
		},
		{
			name: "WithTransaction with function error",
			setupMock: func() (*mockMessageBroker, *mockTransaction) {
				tx := newMockTransaction("tx-func-error")
				mockPub := &mockPublisher{
					beginTransactionFunc: func(ctx context.Context) (messaging.Transaction, error) {
						return tx, nil
					},
				}
				broker := &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
				return broker, tx
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			fn: func(ctx context.Context, txPub *TransactionalEventPublisher) error {
				return errors.New("function logic error")
			},
			wantErr: true,
			errMsg:  "transaction function",
		},
		{
			name: "WithTransaction begin fails",
			setupMock: func() (*mockMessageBroker, *mockTransaction) {
				mockPub := &mockPublisher{
					beginTransactionFunc: func(ctx context.Context) (messaging.Transaction, error) {
						return nil, errors.New("begin transaction failed")
					},
				}
				broker := &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
				return broker, nil
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			fn: func(ctx context.Context, txPub *TransactionalEventPublisher) error {
				return nil
			},
			wantErr: true,
			errMsg:  "begin transaction",
		},
		{
			name: "WithTransaction commit fails",
			setupMock: func() (*mockMessageBroker, *mockTransaction) {
				tx := newMockTransaction("tx-commit-fail")
				tx.commitFunc = func(ctx context.Context) error {
					return errors.New("commit failed")
				}
				mockPub := &mockPublisher{
					beginTransactionFunc: func(ctx context.Context) (messaging.Transaction, error) {
						return tx, nil
					},
				}
				broker := &mockMessageBroker{
					publisherFunc: func(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
						return mockPub, nil
					},
				}
				return broker, tx
			},
			config: &PublisherConfig{
				BrokerType:      "nats",
				BrokerEndpoints: []string{"nats://localhost:4222"},
				SerializerType:  "json",
				Timeout:         30 * time.Second,
			},
			fn: func(ctx context.Context, txPub *TransactionalEventPublisher) error {
				event := &saga.SagaEvent{
					ID:        "event-1",
					SagaID:    "saga-1",
					Type:      saga.EventSagaStarted,
					Version:   "1.0",
					Timestamp: time.Now(),
				}
				return txPub.PublishEvent(ctx, event)
			},
			wantErr: true,
			errMsg:  "commit transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker, tx := tt.setupMock()
			publisher, err := NewSagaEventPublisher(broker, tt.config)
			require.NoError(t, err)
			defer publisher.Close()

			ctx := context.Background()
			err = publisher.WithTransaction(ctx, tt.fn)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// Run validation if provided
				if tt.validate != nil && tx != nil {
					tt.validate(t, tx)
				}
			}
		})
	}
}

func TestSagaEventPublisher_WithTransactionWhenClosed(t *testing.T) {
	logger.InitLogger()

	broker := &mockMessageBroker{}
	config := &PublisherConfig{
		BrokerType:      "nats",
		BrokerEndpoints: []string{"nats://localhost:4222"},
		SerializerType:  "json",
		Timeout:         30 * time.Second,
	}

	publisher, err := NewSagaEventPublisher(broker, config)
	require.NoError(t, err)

	// Close the publisher
	err = publisher.Close()
	require.NoError(t, err)

	// Try to use WithTransaction after closing
	ctx := context.Background()
	err = publisher.WithTransaction(ctx, func(ctx context.Context, txPub *TransactionalEventPublisher) error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publisher is closed")
}

func TestTransactionalEventPublisher_PublishEvent(t *testing.T) {
	logger.InitLogger()

	tests := []struct {
		name    string
		event   *saga.SagaEvent
		tx      *mockTransaction
		wantErr bool
		errMsg  string
	}{
		{
			name: "successful publish in transaction",
			event: &saga.SagaEvent{
				ID:        "event-tx-pub-1",
				SagaID:    "saga-tx-pub",
				Type:      saga.EventSagaStarted,
				Version:   "1.0",
				Timestamp: time.Now(),
			},
			tx:      newMockTransaction("tx-pub-1"),
			wantErr: false,
		},
		{
			name:    "nil event",
			event:   nil,
			tx:      newMockTransaction("tx-nil"),
			wantErr: true,
			errMsg:  "event cannot be nil",
		},
		{
			name: "invalid event - missing saga ID",
			event: &saga.SagaEvent{
				ID:        "event-1",
				Type:      saga.EventSagaStarted,
				Version:   "1.0",
				Timestamp: time.Now(),
			},
			tx:      newMockTransaction("tx-invalid"),
			wantErr: true,
			errMsg:  "event validation failed",
		},
		{
			name: "publish fails in transaction",
			event: &saga.SagaEvent{
				ID:        "event-fail",
				SagaID:    "saga-fail",
				Type:      saga.EventSagaStarted,
				Version:   "1.0",
				Timestamp: time.Now(),
			},
			tx: &mockTransaction{
				id: "tx-fail",
				publishFunc: func(ctx context.Context, message *messaging.Message) error {
					return errors.New("publish failed")
				},
			},
			wantErr: true,
			errMsg:  "publish in transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer, err := NewSagaEventSerializer("json", nil)
			require.NoError(t, err)

			txPub := &TransactionalEventPublisher{
				tx:         tt.tx,
				serializer: serializer,
				config: &PublisherConfig{
					TopicPrefix: "saga",
				},
				logger:  logger.GetLogger(),
				metrics: &PublisherMetrics{},
			}

			ctx := context.Background()
			err = txPub.PublishEvent(ctx, tt.event)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 1, len(tt.tx.publishedMsgs))
				assert.Equal(t, tt.event.ID, tt.tx.publishedMsgs[0].ID)
				assert.Equal(t, tt.tx.GetID(), tt.tx.publishedMsgs[0].Headers["tx_id"])
			}
		})
	}
}

func TestTransactionalEventPublisher_GetTransactionID(t *testing.T) {
	tx := newMockTransaction("tx-get-id-123")
	txPub := &TransactionalEventPublisher{
		tx: tx,
	}

	assert.Equal(t, "tx-get-id-123", txPub.GetTransactionID())
}

