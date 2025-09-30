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
	"sync/atomic"
)

// DLQMetrics 包含死信队列的度量指标。
type DLQMetrics struct {
	// TotalMessages 总共收到的消息数
	TotalMessages atomic.Int64

	// MessagesStored 已存储的消息数
	MessagesStored atomic.Int64

	// MessagesDiscarded 已丢弃的消息数
	MessagesDiscarded atomic.Int64

	// MessagesRetried 重试的消息数
	MessagesRetried atomic.Int64

	// MessagesRemoved 已移除的消息数
	MessagesRemoved atomic.Int64

	// MessagesExpired 已过期的消息数
	MessagesExpired atomic.Int64

	// MessagesEvicted 因容量限制被驱逐的消息数
	MessagesEvicted atomic.Int64

	// NotificationsSent 已发送的通知数
	NotificationsSent atomic.Int64
}

// NewDLQMetrics 创建新的度量指标实例。
func NewDLQMetrics() *DLQMetrics {
	return &DLQMetrics{}
}

// RecordMessage 记录收到一条消息。
func (m *DLQMetrics) RecordMessage() {
	m.TotalMessages.Add(1)
}

// RecordStore 记录存储一条消息。
func (m *DLQMetrics) RecordStore() {
	m.MessagesStored.Add(1)
}

// RecordDiscard 记录丢弃一条消息。
func (m *DLQMetrics) RecordDiscard() {
	m.MessagesDiscarded.Add(1)
}

// RecordRetry 记录重试一条消息。
func (m *DLQMetrics) RecordRetry() {
	m.MessagesRetried.Add(1)
}

// RecordRemoval 记录移除一条消息。
func (m *DLQMetrics) RecordRemoval() {
	m.MessagesRemoved.Add(1)
}

// RecordExpiration 记录过期一条消息。
func (m *DLQMetrics) RecordExpiration() {
	m.MessagesExpired.Add(1)
}

// RecordEviction 记录驱逐一条消息。
func (m *DLQMetrics) RecordEviction() {
	m.MessagesEvicted.Add(1)
}

// RecordNotification 记录发送一条通知。
func (m *DLQMetrics) RecordNotification() {
	m.NotificationsSent.Add(1)
}

// GetTotalMessages 返回总消息数。
func (m *DLQMetrics) GetTotalMessages() int64 {
	return m.TotalMessages.Load()
}

// GetMessagesStored 返回已存储消息数。
func (m *DLQMetrics) GetMessagesStored() int64 {
	return m.MessagesStored.Load()
}

// GetMessagesDiscarded 返回已丢弃消息数。
func (m *DLQMetrics) GetMessagesDiscarded() int64 {
	return m.MessagesDiscarded.Load()
}

// GetMessagesRetried 返回重试消息数。
func (m *DLQMetrics) GetMessagesRetried() int64 {
	return m.MessagesRetried.Load()
}

// GetMessagesRemoved 返回已移除消息数。
func (m *DLQMetrics) GetMessagesRemoved() int64 {
	return m.MessagesRemoved.Load()
}

// GetMessagesExpired 返回已过期消息数。
func (m *DLQMetrics) GetMessagesExpired() int64 {
	return m.MessagesExpired.Load()
}

// GetMessagesEvicted 返回已驱逐消息数。
func (m *DLQMetrics) GetMessagesEvicted() int64 {
	return m.MessagesEvicted.Load()
}

// GetNotificationsSent 返回已发送通知数。
func (m *DLQMetrics) GetNotificationsSent() int64 {
	return m.NotificationsSent.Load()
}

// Reset 重置所有度量指标。
func (m *DLQMetrics) Reset() {
	m.TotalMessages.Store(0)
	m.MessagesStored.Store(0)
	m.MessagesDiscarded.Store(0)
	m.MessagesRetried.Store(0)
	m.MessagesRemoved.Store(0)
	m.MessagesExpired.Store(0)
	m.MessagesEvicted.Store(0)
	m.NotificationsSent.Store(0)
}
