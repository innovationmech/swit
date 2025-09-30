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

package resilience

import (
	"context"
	"testing"
	"time"
)

func TestNewDLQManager(t *testing.T) {
	tests := []struct {
		name    string
		config  DLQManagerConfig
		wantErr bool
	}{
		{
			name:    "valid default config",
			config:  DefaultDLQManagerConfig(),
			wantErr: false,
		},
		{
			name: "invalid max retries",
			config: DLQManagerConfig{
				Enabled:    true,
				MaxRetries: -1,
				TTL:        24 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "invalid ttl",
			config: DLQManagerConfig{
				Enabled: true,
				TTL:     -1,
			},
			wantErr: true,
		},
		{
			name: "invalid max queue size",
			config: DLQManagerConfig{
				Enabled:      true,
				TTL:          24 * time.Hour,
				MaxQueueSize: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := NewDLQManager(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDLQManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if mgr != nil {
				defer mgr.Close()
			}
		})
	}
}

func TestDLQManager_Route(t *testing.T) {
	tests := []struct {
		name    string
		config  DLQManagerConfig
		msg     *DeadLetterMessage
		wantErr bool
	}{
		{
			name: "store policy",
			config: DLQManagerConfig{
				Enabled:      true,
				Policy:       DeadLetterPolicyStore,
				MaxRetries:   3,
				TTL:          24 * time.Hour,
				MaxQueueSize: 100,
			},
			msg: &DeadLetterMessage{
				ID:            "msg-1",
				OriginalTopic: "test-topic",
				Payload:       []byte("test payload"),
				RetryCount:    0,
			},
			wantErr: false,
		},
		{
			name: "discard policy",
			config: DLQManagerConfig{
				Enabled:    true,
				Policy:     DeadLetterPolicyDiscard,
				MaxRetries: 3,
				TTL:        24 * time.Hour,
			},
			msg: &DeadLetterMessage{
				ID:            "msg-2",
				OriginalTopic: "test-topic",
				Payload:       []byte("test payload"),
			},
			wantErr: false,
		},
		{
			name: "retry policy under max retries",
			config: DLQManagerConfig{
				Enabled:    true,
				Policy:     DeadLetterPolicyRetry,
				MaxRetries: 3,
				TTL:        24 * time.Hour,
			},
			msg: &DeadLetterMessage{
				ID:            "msg-3",
				OriginalTopic: "test-topic",
				Payload:       []byte("test payload"),
				RetryCount:    1,
			},
			wantErr: false,
		},
		{
			name: "retry policy exceeds max retries",
			config: DLQManagerConfig{
				Enabled:      true,
				Policy:       DeadLetterPolicyRetry,
				MaxRetries:   3,
				TTL:          24 * time.Hour,
				MaxQueueSize: 100,
			},
			msg: &DeadLetterMessage{
				ID:            "msg-4",
				OriginalTopic: "test-topic",
				Payload:       []byte("test payload"),
				RetryCount:    5,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := NewDLQManager(tt.config)
			if err != nil {
				t.Fatalf("failed to create DLQManager: %v", err)
			}
			defer mgr.Close()

			ctx := context.Background()
			err = mgr.Route(ctx, tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Route() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDLQManager_StoreAndRetrieve(t *testing.T) {
	config := DefaultDLQManagerConfig()
	config.MaxQueueSize = 10

	mgr, err := NewDLQManager(config)
	if err != nil {
		t.Fatalf("failed to create DLQManager: %v", err)
	}
	defer mgr.Close()

	// 存储消息
	msg := &DeadLetterMessage{
		ID:            "test-msg-1",
		OriginalTopic: "test-topic",
		Payload:       []byte("test payload"),
		Headers: map[string]string{
			"content-type": "application/json",
		},
		RetryCount:    2,
		FailureReason: "processing failed",
	}

	ctx := context.Background()
	if err := mgr.Route(ctx, msg); err != nil {
		t.Fatalf("failed to route message: %v", err)
	}

	// 检索消息
	retrieved, err := mgr.GetMessage("test-msg-1")
	if err != nil {
		t.Fatalf("failed to get message: %v", err)
	}

	if retrieved.ID != msg.ID {
		t.Errorf("message ID mismatch: got %s, want %s", retrieved.ID, msg.ID)
	}
	if retrieved.OriginalTopic != msg.OriginalTopic {
		t.Errorf("original topic mismatch: got %s, want %s", retrieved.OriginalTopic, msg.OriginalTopic)
	}
	if retrieved.RetryCount != msg.RetryCount {
		t.Errorf("retry count mismatch: got %d, want %d", retrieved.RetryCount, msg.RetryCount)
	}

	// 检查队列大小
	if size := mgr.GetQueueSize(); size != 1 {
		t.Errorf("queue size mismatch: got %d, want 1", size)
	}
}

func TestDLQManager_ListMessages(t *testing.T) {
	config := DefaultDLQManagerConfig()
	mgr, err := NewDLQManager(config)
	if err != nil {
		t.Fatalf("failed to create DLQManager: %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// 存储多条消息
	for i := 0; i < 5; i++ {
		msg := &DeadLetterMessage{
			ID:            string(rune('a' + i)),
			OriginalTopic: "test-topic",
			Payload:       []byte("test payload"),
		}
		if err := mgr.Route(ctx, msg); err != nil {
			t.Fatalf("failed to route message %d: %v", i, err)
		}
	}

	// 列出所有消息
	messages := mgr.ListMessages()
	if len(messages) != 5 {
		t.Errorf("message count mismatch: got %d, want 5", len(messages))
	}
}

func TestDLQManager_RemoveMessage(t *testing.T) {
	config := DefaultDLQManagerConfig()
	mgr, err := NewDLQManager(config)
	if err != nil {
		t.Fatalf("failed to create DLQManager: %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// 存储消息
	msg := &DeadLetterMessage{
		ID:            "remove-test",
		OriginalTopic: "test-topic",
		Payload:       []byte("test payload"),
	}
	if err := mgr.Route(ctx, msg); err != nil {
		t.Fatalf("failed to route message: %v", err)
	}

	// 移除消息
	if err := mgr.RemoveMessage("remove-test"); err != nil {
		t.Fatalf("failed to remove message: %v", err)
	}

	// 验证消息已被移除
	if _, err := mgr.GetMessage("remove-test"); err == nil {
		t.Error("expected error when getting removed message, got nil")
	}

	if size := mgr.GetQueueSize(); size != 0 {
		t.Errorf("queue size should be 0 after removal, got %d", size)
	}
}

func TestDLQManager_Purge(t *testing.T) {
	config := DefaultDLQManagerConfig()
	mgr, err := NewDLQManager(config)
	if err != nil {
		t.Fatalf("failed to create DLQManager: %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// 存储多条消息
	for i := 0; i < 10; i++ {
		msg := &DeadLetterMessage{
			ID:            string(rune('a' + i)),
			OriginalTopic: "test-topic",
			Payload:       []byte("test payload"),
		}
		if err := mgr.Route(ctx, msg); err != nil {
			t.Fatalf("failed to route message %d: %v", i, err)
		}
	}

	// 清空队列
	if err := mgr.Purge(); err != nil {
		t.Fatalf("failed to purge queue: %v", err)
	}

	// 验证队列已清空
	if size := mgr.GetQueueSize(); size != 0 {
		t.Errorf("queue size should be 0 after purge, got %d", size)
	}
}

func TestDLQManager_MaxQueueSize(t *testing.T) {
	config := DefaultDLQManagerConfig()
	config.MaxQueueSize = 3

	mgr, err := NewDLQManager(config)
	if err != nil {
		t.Fatalf("failed to create DLQManager: %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// 存储超过容量的消息
	for i := 0; i < 5; i++ {
		msg := &DeadLetterMessage{
			ID:            string(rune('a' + i)),
			OriginalTopic: "test-topic",
			Payload:       []byte("test payload"),
		}
		if err := mgr.Route(ctx, msg); err != nil {
			t.Fatalf("failed to route message %d: %v", i, err)
		}
	}

	// 验证队列大小不超过限制
	if size := mgr.GetQueueSize(); size != 3 {
		t.Errorf("queue size should be 3 (max), got %d", size)
	}

	// 验证最新的消息仍在队列中
	if _, err := mgr.GetMessage("d"); err != nil {
		t.Error("expected newest messages to be retained")
	}

	// 验证旧消息已被驱逐
	if _, err := mgr.GetMessage("a"); err == nil {
		t.Error("expected oldest message to be evicted")
	}
}

func TestDLQManager_NotifyCallback(t *testing.T) {
	notified := make(chan *DeadLetterMessage, 1)

	config := DefaultDLQManagerConfig()
	config.Policy = DeadLetterPolicyNotify
	config.NotifyCallback = func(msg *DeadLetterMessage) {
		notified <- msg
	}

	mgr, err := NewDLQManager(config)
	if err != nil {
		t.Fatalf("failed to create DLQManager: %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// 存储消息
	msg := &DeadLetterMessage{
		ID:            "notify-test",
		OriginalTopic: "test-topic",
		Payload:       []byte("test payload"),
	}
	if err := mgr.Route(ctx, msg); err != nil {
		t.Fatalf("failed to route message: %v", err)
	}

	// 验证回调被调用
	select {
	case notifiedMsg := <-notified:
		if notifiedMsg.ID != msg.ID {
			t.Errorf("notified message ID mismatch: got %s, want %s", notifiedMsg.ID, msg.ID)
		}
	case <-time.After(1 * time.Second):
		t.Error("callback was not invoked within timeout")
	}
}

func TestDLQManager_Metrics(t *testing.T) {
	config := DefaultDLQManagerConfig()
	mgr, err := NewDLQManager(config)
	if err != nil {
		t.Fatalf("failed to create DLQManager: %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// 路由消息并验证度量指标
	msg := &DeadLetterMessage{
		ID:            "metrics-test",
		OriginalTopic: "test-topic",
		Payload:       []byte("test payload"),
	}
	if err := mgr.Route(ctx, msg); err != nil {
		t.Fatalf("failed to route message: %v", err)
	}

	metrics := mgr.GetMetrics()
	if total := metrics.GetTotalMessages(); total != 1 {
		t.Errorf("total messages should be 1, got %d", total)
	}
	if stored := metrics.GetMessagesStored(); stored != 1 {
		t.Errorf("stored messages should be 1, got %d", stored)
	}

	// 移除消息并验证度量指标
	if err := mgr.RemoveMessage("metrics-test"); err != nil {
		t.Fatalf("failed to remove message: %v", err)
	}

	if removed := metrics.GetMessagesRemoved(); removed != 1 {
		t.Errorf("removed messages should be 1, got %d", removed)
	}
}

func TestDLQManagerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  DLQManagerConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultDLQManagerConfig(),
			wantErr: false,
		},
		{
			name: "disabled config",
			config: DLQManagerConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "negative max retries",
			config: DLQManagerConfig{
				Enabled:    true,
				MaxRetries: -1,
				TTL:        24 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "zero ttl",
			config: DLQManagerConfig{
				Enabled:    true,
				MaxRetries: 3,
				TTL:        0,
			},
			wantErr: true,
		},
		{
			name: "negative retry interval",
			config: DLQManagerConfig{
				Enabled:       true,
				MaxRetries:    3,
				TTL:           24 * time.Hour,
				RetryInterval: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
