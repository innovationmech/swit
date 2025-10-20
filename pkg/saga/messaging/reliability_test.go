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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockReliabilityPublisher is a mock implementation of messaging.EventPublisher for reliability testing.
type mockReliabilityPublisher struct {
	publishFunc            func(ctx context.Context, message *messaging.Message) error
	publishWithConfirmFunc func(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error)
	publishCallCount       int
	confirmCallCount       int
}

func (m *mockReliabilityPublisher) Publish(ctx context.Context, message *messaging.Message) error {
	m.publishCallCount++
	if m.publishFunc != nil {
		return m.publishFunc(ctx, message)
	}
	return nil
}

func (m *mockReliabilityPublisher) PublishWithConfirm(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error) {
	m.confirmCallCount++
	if m.publishWithConfirmFunc != nil {
		return m.publishWithConfirmFunc(ctx, message)
	}
	return &messaging.PublishConfirmation{
		MessageID: message.ID,
		Partition: 0,
		Offset:    0,
		Timestamp: time.Now(),
	}, nil
}

func (m *mockReliabilityPublisher) PublishBatch(ctx context.Context, messages []*messaging.Message) error {
	return errors.New("not implemented")
}

func (m *mockReliabilityPublisher) PublishAsync(ctx context.Context, message *messaging.Message, callback messaging.PublishCallback) error {
	return errors.New("not implemented")
}

func (m *mockReliabilityPublisher) BeginTransaction(ctx context.Context) (messaging.Transaction, error) {
	return nil, errors.New("not implemented")
}

func (m *mockReliabilityPublisher) Flush(ctx context.Context) error {
	return nil
}

func (m *mockReliabilityPublisher) Close() error {
	return nil
}

func (m *mockReliabilityPublisher) GetMetrics() *messaging.PublisherMetrics {
	return &messaging.PublisherMetrics{}
}

// TestReliabilityConfigSetDefaults tests the SetDefaults method.
func TestReliabilityConfigSetDefaults(t *testing.T) {
	config := &ReliabilityConfig{}
	config.SetDefaults()

	assert.Equal(t, 5*time.Second, config.ConfirmTimeout)
	assert.Equal(t, 3, config.MaxRetryAttempts)
	assert.Equal(t, 100*time.Millisecond, config.InitialInterval)
	assert.Equal(t, 10*time.Second, config.MaxInterval)
	assert.Equal(t, 2.0, config.Multiplier)
	assert.Equal(t, 5, config.MaxDLQAttempts)
	assert.Equal(t, "saga.dlq", config.DLQTopic)
	assert.Equal(t, 30*time.Second, config.PublishTimeout)
}

// TestReliabilityConfigValidate tests the Validate method.
func TestReliabilityConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ReliabilityConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &ReliabilityConfig{
				EnableConfirm:    true,
				ConfirmTimeout:   5 * time.Second,
				EnableRetry:      true,
				MaxRetryAttempts: 3,
				InitialInterval:  100 * time.Millisecond,
				MaxInterval:      10 * time.Second,
				Multiplier:       2.0,
				EnableDLQ:        true,
				DLQTopic:         "saga.dlq",
				MaxDLQAttempts:   5,
				PublishTimeout:   30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid confirm timeout",
			config: &ReliabilityConfig{
				EnableConfirm:  true,
				ConfirmTimeout: 0,
				PublishTimeout: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "confirm_timeout must be positive",
		},
		{
			name: "invalid max retry attempts",
			config: &ReliabilityConfig{
				EnableRetry:      true,
				MaxRetryAttempts: -1,
				InitialInterval:  100 * time.Millisecond,
				MaxInterval:      10 * time.Second,
				Multiplier:       2.0,
				PublishTimeout:   30 * time.Second,
			},
			wantErr: true,
			errMsg:  "max_retry_attempts must be non-negative",
		},
		{
			name: "invalid initial interval",
			config: &ReliabilityConfig{
				EnableRetry:      true,
				MaxRetryAttempts: 3,
				InitialInterval:  0,
				MaxInterval:      10 * time.Second,
				Multiplier:       2.0,
				PublishTimeout:   30 * time.Second,
			},
			wantErr: true,
			errMsg:  "initial_interval must be positive",
		},
		{
			name: "invalid max interval",
			config: &ReliabilityConfig{
				EnableRetry:      true,
				MaxRetryAttempts: 3,
				InitialInterval:  100 * time.Millisecond,
				MaxInterval:      0,
				Multiplier:       2.0,
				PublishTimeout:   30 * time.Second,
			},
			wantErr: true,
			errMsg:  "max_interval must be positive",
		},
		{
			name: "initial interval exceeds max interval",
			config: &ReliabilityConfig{
				EnableRetry:      true,
				MaxRetryAttempts: 3,
				InitialInterval:  20 * time.Second,
				MaxInterval:      10 * time.Second,
				Multiplier:       2.0,
				PublishTimeout:   30 * time.Second,
			},
			wantErr: true,
			errMsg:  "initial_interval must not exceed max_interval",
		},
		{
			name: "invalid multiplier",
			config: &ReliabilityConfig{
				EnableRetry:      true,
				MaxRetryAttempts: 3,
				InitialInterval:  100 * time.Millisecond,
				MaxInterval:      10 * time.Second,
				Multiplier:       1.0,
				PublishTimeout:   30 * time.Second,
			},
			wantErr: true,
			errMsg:  "multiplier must be greater than 1.0",
		},
		{
			name: "DLQ enabled without topic",
			config: &ReliabilityConfig{
				EnableDLQ:       true,
				DLQTopic:        "",
				MaxDLQAttempts:  5,
				PublishTimeout:  30 * time.Second,
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
			},
			wantErr: true,
			errMsg:  "dlq_topic is required",
		},
		{
			name: "invalid max DLQ attempts",
			config: &ReliabilityConfig{
				EnableDLQ:       true,
				DLQTopic:        "saga.dlq",
				MaxDLQAttempts:  0,
				PublishTimeout:  30 * time.Second,
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
			},
			wantErr: true,
			errMsg:  "max_dlq_attempts must be positive",
		},
		{
			name: "invalid publish timeout",
			config: &ReliabilityConfig{
				PublishTimeout:  0,
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
			},
			wantErr: true,
			errMsg:  "publish_timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestNewReliabilityHandler tests creating a new reliability handler.
func TestNewReliabilityHandler(t *testing.T) {
	tests := []struct {
		name    string
		config  *ReliabilityConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ReliabilityConfig{
				EnableConfirm:    true,
				ConfirmTimeout:   5 * time.Second,
				MaxRetryAttempts: 3,
				InitialInterval:  100 * time.Millisecond,
				MaxInterval:      10 * time.Second,
				Multiplier:       2.0,
				PublishTimeout:   30 * time.Second,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "invalid config - initial interval exceeds max",
			config: &ReliabilityConfig{
				EnableRetry:      true,
				InitialInterval:  20 * time.Second,
				MaxInterval:      10 * time.Second, // Less than InitialInterval
				Multiplier:       2.0,
				MaxRetryAttempts: 3,
				PublishTimeout:   30 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewReliabilityHandler(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, handler)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, handler)
				assert.NotNil(t, handler.GetMetrics())
				assert.NotNil(t, handler.GetConfig())
			}
		})
	}
}

// TestPublishWithReliability_Success tests successful publishing.
func TestPublishWithReliability_Success(t *testing.T) {
	config := &ReliabilityConfig{
		EnableRetry:      true,
		MaxRetryAttempts: 3,
		InitialInterval:  10 * time.Millisecond,
		MaxInterval:      100 * time.Millisecond,
		Multiplier:       2.0,
		PublishTimeout:   5 * time.Second,
	}

	handler, err := NewReliabilityHandler(config)
	require.NoError(t, err)

	mockPublisher := &mockReliabilityPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			return nil // Success on first try
		},
	}

	message := &messaging.Message{
		ID:      "test-message-1",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	ctx := context.Background()
	err = handler.PublishWithReliability(ctx, mockPublisher, message)

	assert.NoError(t, err)
	assert.Equal(t, 1, mockPublisher.publishCallCount)
	assert.Equal(t, int64(1), handler.GetMetrics().TotalPublishAttempts.Load())
	assert.Equal(t, int64(1), handler.GetMetrics().TotalPublishSuccesses.Load())
	assert.Equal(t, int64(0), handler.GetMetrics().TotalPublishFailures.Load())
	assert.Equal(t, int64(0), handler.GetMetrics().TotalRetries.Load())
}

// TestPublishWithReliability_RetrySuccess tests successful publishing after retries.
func TestPublishWithReliability_RetrySuccess(t *testing.T) {
	config := &ReliabilityConfig{
		EnableRetry:      true,
		MaxRetryAttempts: 3,
		InitialInterval:  10 * time.Millisecond,
		MaxInterval:      100 * time.Millisecond,
		Multiplier:       2.0,
		PublishTimeout:   5 * time.Second,
	}

	handler, err := NewReliabilityHandler(config)
	require.NoError(t, err)

	attemptCount := 0
	mockPublisher := &mockReliabilityPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			attemptCount++
			if attemptCount < 3 {
				return errors.New("temporary failure")
			}
			return nil // Success on third attempt
		},
	}

	message := &messaging.Message{
		ID:      "test-message-2",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	ctx := context.Background()
	err = handler.PublishWithReliability(ctx, mockPublisher, message)

	assert.NoError(t, err)
	assert.Equal(t, 3, mockPublisher.publishCallCount)
	assert.Equal(t, int64(1), handler.GetMetrics().TotalPublishAttempts.Load())
	assert.Equal(t, int64(1), handler.GetMetrics().TotalPublishSuccesses.Load())
	assert.Equal(t, int64(0), handler.GetMetrics().TotalPublishFailures.Load())
	assert.Equal(t, int64(2), handler.GetMetrics().TotalRetries.Load())
	assert.Equal(t, int64(1), handler.GetMetrics().TotalRetrySuccesses.Load())
}

// TestPublishWithReliability_MaxRetriesExceeded tests failure after max retries.
func TestPublishWithReliability_MaxRetriesExceeded(t *testing.T) {
	config := &ReliabilityConfig{
		EnableRetry:      true,
		MaxRetryAttempts: 2,
		InitialInterval:  10 * time.Millisecond,
		MaxInterval:      100 * time.Millisecond,
		Multiplier:       2.0,
		PublishTimeout:   5 * time.Second,
	}

	handler, err := NewReliabilityHandler(config)
	require.NoError(t, err)

	mockPublisher := &mockReliabilityPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			return errors.New("persistent failure")
		},
	}

	message := &messaging.Message{
		ID:      "test-message-3",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	ctx := context.Background()
	err = handler.PublishWithReliability(ctx, mockPublisher, message)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish failed after 3 attempts")
	assert.Equal(t, 3, mockPublisher.publishCallCount) // Initial + 2 retries
	assert.Equal(t, int64(1), handler.GetMetrics().TotalPublishAttempts.Load())
	assert.Equal(t, int64(0), handler.GetMetrics().TotalPublishSuccesses.Load())
	assert.Equal(t, int64(1), handler.GetMetrics().TotalPublishFailures.Load())
	assert.Equal(t, int64(2), handler.GetMetrics().TotalRetries.Load())
}

// TestPublishWithReliability_WithConfirmation tests publishing with confirmation.
func TestPublishWithReliability_WithConfirmation(t *testing.T) {
	config := &ReliabilityConfig{
		EnableConfirm:    true,
		ConfirmTimeout:   2 * time.Second,
		EnableRetry:      true,
		MaxRetryAttempts: 3,
		InitialInterval:  10 * time.Millisecond,
		MaxInterval:      100 * time.Millisecond,
		Multiplier:       2.0,
		PublishTimeout:   5 * time.Second,
	}

	handler, err := NewReliabilityHandler(config)
	require.NoError(t, err)

	mockPublisher := &mockReliabilityPublisher{
		publishWithConfirmFunc: func(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error) {
			return &messaging.PublishConfirmation{
				MessageID: message.ID,
				Partition: 0,
				Offset:    0,
				Timestamp: time.Now(),
			}, nil
		},
	}

	message := &messaging.Message{
		ID:      "test-message-4",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	ctx := context.Background()
	err = handler.PublishWithReliability(ctx, mockPublisher, message)

	assert.NoError(t, err)
	assert.Equal(t, 1, mockPublisher.confirmCallCount)
	assert.Equal(t, int64(1), handler.GetMetrics().TotalPublishAttempts.Load())
	assert.Equal(t, int64(1), handler.GetMetrics().TotalPublishSuccesses.Load())
	assert.Equal(t, int64(1), handler.GetMetrics().TotalConfirmations.Load())
}

// TestPublishWithReliability_WithDLQ tests DLQ integration.
func TestPublishWithReliability_WithDLQ(t *testing.T) {
	config := &ReliabilityConfig{
		EnableRetry:      true,
		MaxRetryAttempts: 2,
		InitialInterval:  10 * time.Millisecond,
		MaxInterval:      100 * time.Millisecond,
		Multiplier:       2.0,
		EnableDLQ:        true,
		DLQTopic:         "test.dlq",
		MaxDLQAttempts:   2,
		PublishTimeout:   5 * time.Second,
	}

	handler, err := NewReliabilityHandler(config)
	require.NoError(t, err)

	var dlqMessage *messaging.Message
	mockPublisher := &mockReliabilityPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			// Capture DLQ message
			if message.Topic == "test.dlq" {
				dlqMessage = message
				return nil
			}
			return errors.New("persistent failure")
		},
	}

	message := &messaging.Message{
		ID:      "test-message-5",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	ctx := context.Background()
	err = handler.PublishWithReliability(ctx, mockPublisher, message)

	assert.Error(t, err)
	assert.NotNil(t, dlqMessage)
	assert.Equal(t, "test.dlq", dlqMessage.Topic)
	assert.Contains(t, dlqMessage.Headers["dlq_original_topic"], "test-topic")
	assert.Contains(t, dlqMessage.Headers["dlq_original_id"], "test-message-5")
	assert.Equal(t, int64(1), handler.GetMetrics().TotalDLQSent.Load())
}

// TestPublishWithReliability_ContextCancellation tests context cancellation.
func TestPublishWithReliability_ContextCancellation(t *testing.T) {
	config := &ReliabilityConfig{
		EnableRetry:      true,
		MaxRetryAttempts: 5,
		InitialInterval:  100 * time.Millisecond,
		MaxInterval:      1 * time.Second,
		Multiplier:       2.0,
		PublishTimeout:   10 * time.Second,
	}

	handler, err := NewReliabilityHandler(config)
	require.NoError(t, err)

	mockPublisher := &mockReliabilityPublisher{
		publishFunc: func(ctx context.Context, message *messaging.Message) error {
			return errors.New("failure")
		},
	}

	message := &messaging.Message{
		ID:      "test-message-6",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	// Create a context that will be cancelled after 150ms
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err = handler.PublishWithReliability(ctx, mockPublisher, message)

	assert.Error(t, err)
	// Should have attempted at least once, possibly twice before context cancelled
	assert.LessOrEqual(t, mockPublisher.publishCallCount, 3)
}

// TestCalculateRetryDelay tests the exponential backoff calculation.
func TestCalculateRetryDelay(t *testing.T) {
	tests := []struct {
		name        string
		config      *ReliabilityConfig
		attempt     int
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name: "first retry no jitter",
			config: &ReliabilityConfig{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
				EnableJitter:    false,
			},
			attempt:     0,
			minExpected: 100 * time.Millisecond,
			maxExpected: 100 * time.Millisecond,
		},
		{
			name: "second retry no jitter",
			config: &ReliabilityConfig{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
				EnableJitter:    false,
			},
			attempt:     1,
			minExpected: 200 * time.Millisecond,
			maxExpected: 200 * time.Millisecond,
		},
		{
			name: "third retry no jitter",
			config: &ReliabilityConfig{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
				EnableJitter:    false,
			},
			attempt:     2,
			minExpected: 400 * time.Millisecond,
			maxExpected: 400 * time.Millisecond,
		},
		{
			name: "max interval cap",
			config: &ReliabilityConfig{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     500 * time.Millisecond,
				Multiplier:      2.0,
				EnableJitter:    false,
			},
			attempt:     10,
			minExpected: 500 * time.Millisecond,
			maxExpected: 500 * time.Millisecond,
		},
		{
			name: "with jitter",
			config: &ReliabilityConfig{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
				EnableJitter:    true,
			},
			attempt:     1,
			minExpected: 200 * time.Millisecond, // Jitter is 0 in current implementation
			maxExpected: 200 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewReliabilityHandler(tt.config)
			require.NoError(t, err)

			delay := handler.calculateRetryDelay(tt.attempt)

			// Check if delay is within expected range
			assert.GreaterOrEqual(t, delay, tt.minExpected)
			assert.LessOrEqual(t, delay, tt.maxExpected)
		})
	}
}

// TestReliabilityMetrics_GetSuccessRate tests the success rate calculation.
func TestReliabilityMetrics_GetSuccessRate(t *testing.T) {
	metrics := &ReliabilityMetrics{}

	// No attempts
	assert.Equal(t, 0.0, metrics.GetSuccessRate())

	// Some attempts
	metrics.TotalPublishAttempts.Store(10)
	metrics.TotalPublishSuccesses.Store(8)
	assert.Equal(t, 80.0, metrics.GetSuccessRate())

	// All successful
	metrics.TotalPublishAttempts.Store(5)
	metrics.TotalPublishSuccesses.Store(5)
	assert.Equal(t, 100.0, metrics.GetSuccessRate())
}

// TestReliabilityMetrics_GetRetrySuccessRate tests the retry success rate calculation.
func TestReliabilityMetrics_GetRetrySuccessRate(t *testing.T) {
	metrics := &ReliabilityMetrics{}

	// No retries
	assert.Equal(t, 0.0, metrics.GetRetrySuccessRate())

	// Some retries
	metrics.TotalRetries.Store(10)
	metrics.TotalRetrySuccesses.Store(7)
	assert.Equal(t, 70.0, metrics.GetRetrySuccessRate())

	// All retries successful
	metrics.TotalRetries.Store(3)
	metrics.TotalRetrySuccesses.Store(3)
	assert.Equal(t, 100.0, metrics.GetRetrySuccessRate())
}

// TestReliabilityMetrics_Reset tests resetting metrics.
func TestReliabilityMetrics_Reset(t *testing.T) {
	metrics := &ReliabilityMetrics{}

	// Set some values
	metrics.TotalPublishAttempts.Store(10)
	metrics.TotalPublishSuccesses.Store(8)
	metrics.TotalPublishFailures.Store(2)
	metrics.TotalRetries.Store(5)
	metrics.TotalDLQSent.Store(1)

	// Reset
	metrics.Reset()

	// Verify all values are zero
	assert.Equal(t, int64(0), metrics.TotalPublishAttempts.Load())
	assert.Equal(t, int64(0), metrics.TotalPublishSuccesses.Load())
	assert.Equal(t, int64(0), metrics.TotalPublishFailures.Load())
	assert.Equal(t, int64(0), metrics.TotalRetries.Load())
	assert.Equal(t, int64(0), metrics.TotalDLQSent.Load())
	assert.Equal(t, int64(0), metrics.TotalConfirmations.Load())
	assert.Equal(t, int64(0), metrics.TotalConfirmTimeouts.Load())
}
