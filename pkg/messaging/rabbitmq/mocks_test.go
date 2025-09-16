package rabbitmq

import (
	"fmt"
	"sync"

	"github.com/streadway/amqp"
)

type mockConnection struct {
	mu       sync.Mutex
	closed   bool
	notify   chan *amqp.Error
	channels []*mockChannel
	factory  func() *mockChannel
}

func newMockConnection(factory func() *mockChannel) *mockConnection {
	return &mockConnection{factory: factory}
}

func (m *mockConnection) Channel() (amqpChannel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil, fmt.Errorf("connection closed")
	}

	channel := m.factory()
	m.channels = append(m.channels, channel)
	return channel, nil
}

func (m *mockConnection) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	notify := m.notify
	m.mu.Unlock()

	if notify != nil {
		notify <- &amqp.Error{Code: 320, Reason: "closed"}
		close(notify)
	}
	return nil
}

func (m *mockConnection) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func (m *mockConnection) NotifyClose(receiver chan *amqp.Error) chan *amqp.Error {
	m.mu.Lock()
	if m.closed {
		close(receiver)
	}
	m.notify = receiver
	m.mu.Unlock()
	return receiver
}

func (m *mockConnection) triggerClose() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	notify := m.notify
	m.closed = true
	m.mu.Unlock()

	if notify != nil {
		notify <- &amqp.Error{Code: 320, Reason: "forced close"}
		close(notify)
	}
}

type qosCall struct {
	count  int
	size   int
	global bool
}

type mockChannel struct {
	mu       sync.Mutex
	closed   bool
	notify   chan *amqp.Error
	qosCalls []qosCall
}

func newMockChannel() *mockChannel {
	return &mockChannel{}
}

func (m *mockChannel) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	notify := m.notify
	m.mu.Unlock()

	if notify != nil {
		close(notify)
	}
	return nil
}

func (m *mockChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	m.mu.Lock()
	m.qosCalls = append(m.qosCalls, qosCall{count: prefetchCount, size: prefetchSize, global: global})
	m.mu.Unlock()
	return nil
}

func (m *mockChannel) NotifyClose(receiver chan *amqp.Error) chan *amqp.Error {
	m.mu.Lock()
	if m.closed {
		close(receiver)
	}
	m.notify = receiver
	m.mu.Unlock()
	return receiver
}

func (m *mockChannel) triggerClose() {
	m.mu.Lock()
	if m.closed {
		notify := m.notify
		m.mu.Unlock()
		if notify != nil {
			close(notify)
		}
		return
	}
	notify := m.notify
	m.closed = true
	m.mu.Unlock()

	if notify != nil {
		notify <- &amqp.Error{Code: 541, Reason: "channel closed"}
		close(notify)
	}
}

func (m *mockChannel) lastQoS() *qosCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.qosCalls) == 0 {
		return nil
	}
	call := m.qosCalls[len(m.qosCalls)-1]
	return &call
}
