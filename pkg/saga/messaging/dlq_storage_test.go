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

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryDLQStorage_StoreGetRemove(t *testing.T) {
	storage := NewInMemoryDLQStorage(0)
	ctx := context.Background()

	msg := &DLQMessage{ID: "msg-1", Error: "boom"}
	require.NoError(t, storage.Store(ctx, msg))

	got, err := storage.Get(ctx, "msg-1")
	require.NoError(t, err)
	assert.Equal(t, "boom", got.Error)

	size, err := storage.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, size)

	require.NoError(t, storage.Remove(ctx, "msg-1"))
	_, err = storage.Get(ctx, "msg-1")
	assert.Error(t, err)

	// Removing again should error
	assert.Error(t, storage.Remove(ctx, "msg-1"))
}

func TestInMemoryDLQStorage_Validation(t *testing.T) {
	storage := NewInMemoryDLQStorage(0)
	ctx := context.Background()

	assert.Error(t, storage.Store(ctx, nil))
	assert.Error(t, storage.Store(ctx, &DLQMessage{}))
}

func TestInMemoryDLQStorage_ListFIFOOrder(t *testing.T) {
	storage := NewInMemoryDLQStorage(0)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		require.NoError(t, storage.Store(ctx, &DLQMessage{ID: fmt.Sprintf("msg-%d", i)}))
	}

	messages, err := storage.List(ctx)
	require.NoError(t, err)
	require.Len(t, messages, 3)
	assert.Equal(t, "msg-0", messages[0].ID)
	assert.Equal(t, "msg-2", messages[2].ID)
}

func TestInMemoryDLQStorage_UpdateExisting(t *testing.T) {
	storage := NewInMemoryDLQStorage(0)
	ctx := context.Background()

	require.NoError(t, storage.Store(ctx, &DLQMessage{ID: "msg-1", RecoveryAttempts: 0}))
	require.NoError(t, storage.Store(ctx, &DLQMessage{ID: "msg-1", RecoveryAttempts: 2}))

	size, err := storage.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, size)

	got, err := storage.Get(ctx, "msg-1")
	require.NoError(t, err)
	assert.Equal(t, 2, got.RecoveryAttempts)
}

func TestInMemoryDLQStorage_MaxSizeEviction(t *testing.T) {
	storage := NewInMemoryDLQStorage(2)
	ctx := context.Background()

	require.NoError(t, storage.Store(ctx, &DLQMessage{ID: "msg-1"}))
	require.NoError(t, storage.Store(ctx, &DLQMessage{ID: "msg-2"}))
	require.NoError(t, storage.Store(ctx, &DLQMessage{ID: "msg-3"}))

	size, err := storage.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, size)

	// Oldest message should have been evicted
	_, err = storage.Get(ctx, "msg-1")
	assert.Error(t, err)
	_, err = storage.Get(ctx, "msg-3")
	assert.NoError(t, err)
}

// newStartedDLQHandler creates and starts a DLQ handler backed by the given
// storage for cleanup/recovery tests.
func newStartedDLQHandler(t *testing.T, publisher saga.EventPublisher, opts ...DLQOption) DeadLetterQueueHandler {
	t.Helper()

	config := &DeadLetterConfig{
		Enabled:          true,
		Topic:            "test-dlq",
		MaxRetentionDays: 7,
	}

	handler, err := NewDeadLetterQueueHandler(config, publisher, opts...)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, handler.Start(ctx))
	t.Cleanup(func() { _ = handler.Stop(ctx) })

	return handler
}

func TestDLQCleanupRemovesExpiredMessages(t *testing.T) {
	publisher := NewMockEventPublisher()
	storage := NewInMemoryDLQStorage(0)
	handler := newStartedDLQHandler(t, publisher, WithDLQStorage(storage))

	ctx := context.Background()
	pastTime := time.Now().Add(-time.Hour)
	futureTime := time.Now().Add(time.Hour)

	require.NoError(t, storage.Store(ctx, &DLQMessage{ID: "expired-1", ExpiresAt: &pastTime}))
	require.NoError(t, storage.Store(ctx, &DLQMessage{ID: "expired-2", ExpiresAt: &pastTime}))
	require.NoError(t, storage.Store(ctx, &DLQMessage{ID: "active-1", ExpiresAt: &futureTime}))
	require.NoError(t, storage.Store(ctx, &DLQMessage{ID: "no-expiry"}))

	cleanedUp, err := handler.CleanupExpiredMessages(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, cleanedUp)

	size, err := storage.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, size)

	metrics := handler.GetDLQMetrics()
	assert.Equal(t, int64(2), metrics.TotalMessagesExpired)
	assert.Equal(t, int64(2), metrics.TotalMessagesCleanedUp)
}

func TestDLQCleanupRequiresStartedHandler(t *testing.T) {
	publisher := NewMockEventPublisher()
	config := &DeadLetterConfig{
		Enabled: true,
		Topic:   "test-dlq",
	}

	handler, err := NewDeadLetterQueueHandler(config, publisher)
	require.NoError(t, err)

	_, err = handler.CleanupExpiredMessages(context.Background())
	assert.ErrorIs(t, err, ErrHandlerNotInitialized)
}

func TestDLQSendStoresMessageInStorage(t *testing.T) {
	publisher := NewMockEventPublisher()
	storage := NewInMemoryDLQStorage(0)
	handler := newStartedDLQHandler(t, publisher, WithDLQStorage(storage))

	ctx := context.Background()
	event := &saga.SagaEvent{
		ID:        "event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStepFailed,
		Timestamp: time.Now(),
	}
	handlerCtx := &EventHandlerContext{
		MessageID:  "message-1",
		Topic:      "test-topic",
		RetryCount: 1,
	}

	require.NoError(t, handler.SendToDeadLetterQueue(ctx, event, handlerCtx, errors.New("temporary network error")))

	size, err := storage.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, size)

	messages, err := storage.List(ctx)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, event.ID, messages[0].OriginalEvent.ID)
}

func TestDLQRecoveryRemovesMessageFromStorage(t *testing.T) {
	publisher := NewMockEventPublisher()
	storage := NewInMemoryDLQStorage(0)
	handler := newStartedDLQHandler(t, publisher, WithDLQStorage(storage))

	ctx := context.Background()
	msg := &DLQMessage{
		ID:        "recover-1",
		ErrorType: ErrorTypeRetryable,
		OriginalEvent: &saga.SagaEvent{
			ID:     "event-1",
			SagaID: "saga-1",
			Type:   saga.EventSagaStepFailed,
		},
		RetryCount: 1,
		MaxRetries: 3,
	}
	require.NoError(t, storage.Store(ctx, msg))

	require.NoError(t, handler.RecoverFromDeadLetterQueue(ctx, msg))

	// Original event should have been republished
	published := publisher.GetEvents()
	require.Len(t, published, 1)
	assert.Equal(t, "event-1", published[0].ID)

	// Message should be removed from storage after successful recovery
	size, err := storage.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, size)
}

func TestDLQRecoveryCycleProcessesStoredMessages(t *testing.T) {
	publisher := NewMockEventPublisher()
	storage := NewInMemoryDLQStorage(0)
	handler := newStartedDLQHandler(t, publisher, WithDLQStorage(storage))

	ctx := context.Background()
	pastTime := time.Now().Add(-time.Hour)

	// One recoverable message, one expired message
	require.NoError(t, storage.Store(ctx, &DLQMessage{
		ID:        "recoverable-1",
		ErrorType: ErrorTypeRetryable,
		OriginalEvent: &saga.SagaEvent{
			ID:     "event-recoverable",
			SagaID: "saga-1",
			Type:   saga.EventSagaStepFailed,
		},
		RetryCount: 1,
		MaxRetries: 3,
	}))
	require.NoError(t, storage.Store(ctx, &DLQMessage{
		ID:        "expired-1",
		ErrorType: ErrorTypeRetryable,
		ExpiresAt: &pastTime,
	}))

	handler.(*defaultDeadLetterQueueHandler).performRecoveryCycle(ctx)

	// Expired message cleaned up, recoverable message recovered
	size, err := storage.Size(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, size)

	published := publisher.GetEvents()
	require.Len(t, published, 1)
	assert.Equal(t, "event-recoverable", published[0].ID)

	metrics := handler.GetDLQMetrics()
	assert.Equal(t, int64(1), metrics.TotalMessagesRecovered)
	assert.Equal(t, int64(1), metrics.TotalMessagesCleanedUp)
}

func TestDLQRecoveryCyclePersistsFailedRetryState(t *testing.T) {
	publisher := NewMockEventPublisher()
	publisher.publishFn = func(ctx context.Context, event *saga.SagaEvent) error {
		return errors.New("temporary network error")
	}

	storage := NewInMemoryDLQStorage(0)
	handler := newStartedDLQHandler(t, publisher, WithDLQStorage(storage))

	ctx := context.Background()
	require.NoError(t, storage.Store(ctx, &DLQMessage{
		ID:        "fail-1",
		ErrorType: ErrorTypeRetryable,
		OriginalEvent: &saga.SagaEvent{
			ID:     "event-fail",
			SagaID: "saga-1",
			Type:   saga.EventSagaStepFailed,
		},
		RetryCount: 1,
		MaxRetries: 3,
	}))

	handler.(*defaultDeadLetterQueueHandler).performRecoveryCycle(ctx)

	// Message should remain in storage with updated retry bookkeeping
	got, err := storage.Get(ctx, "fail-1")
	require.NoError(t, err)
	assert.Equal(t, 1, got.RecoveryAttempts)
	assert.NotNil(t, got.NextRetryAt)
}

func TestWithDLQStorageAndReprocessorValidation(t *testing.T) {
	config := &DeadLetterConfig{
		Enabled: true,
		Topic:   "test-dlq",
	}
	publisher := NewMockEventPublisher()

	_, err := NewDeadLetterQueueHandler(config, publisher, WithDLQStorage(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DLQ storage cannot be nil")

	_, err = NewDeadLetterQueueHandler(config, publisher, WithDLQReprocessor(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DLQ reprocessor cannot be nil")
}
