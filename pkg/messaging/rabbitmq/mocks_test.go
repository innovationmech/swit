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

package rabbitmq

import (
	"fmt"
	"sync"

	"github.com/streadway/amqp"
)

type mockConnection struct {
	mu            sync.Mutex
	closed        bool
	notify        chan *amqp.Error
	notifyBlocked chan amqp.Blocking
	channels      []amqpChannel
	channelQueue  []amqpChannel
	factory       func() amqpChannel
}

func newMockConnection(factory func() amqpChannel) *mockConnection {
	return &mockConnection{factory: factory}
}

func (m *mockConnection) Channel() (amqpChannel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil, fmt.Errorf("connection closed")
	}

	var channel amqpChannel
	if len(m.channelQueue) > 0 {
		channel = m.channelQueue[0]
		m.channelQueue = m.channelQueue[1:]
	} else if m.factory != nil {
		channel = m.factory()
	} else {
		return nil, fmt.Errorf("no channel available")
	}

	m.channels = append(m.channels, channel)
	return channel, nil
}

func (m *mockConnection) enqueueChannel(channel amqpChannel) {
	m.mu.Lock()
	m.channelQueue = append(m.channelQueue, channel)
	m.mu.Unlock()
}

func (m *mockConnection) getChannels() []amqpChannel {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to avoid race conditions
	result := make([]amqpChannel, len(m.channels))
	copy(result, m.channels)
	return result
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

func (m *mockConnection) NotifyBlocked(receiver chan amqp.Blocking) chan amqp.Blocking {
	m.mu.Lock()
	if m.closed {
		close(receiver)
	}
	m.notifyBlocked = receiver
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

func (m *mockConnection) triggerBlocked(active bool) {
	m.mu.Lock()
	ch := m.notifyBlocked
	m.mu.Unlock()
	if ch != nil {
		ch <- amqp.Blocking{Active: active, Reason: "mock"}
	}
}

type qosCall struct {
	count  int
	size   int
	global bool
}

type publishCall struct {
	exchange   string
	routingKey string
	mandatory  bool
	immediate  bool
	message    amqp.Publishing
}

type mockChannel struct {
	mu               sync.Mutex
	closed           bool
	notify           chan *amqp.Error
	qosCalls         []qosCall
	confirmEnabled   bool
	confirmErr       error
	notifyPublish    chan amqp.Confirmation
	publishCalls     []publishCall
	publishErrors    []error
	confirmResponses []amqp.Confirmation
	autoConfirm      bool

	// topology calls
	exchangesDeclared []struct {
		name       string
		kind       string
		durable    bool
		autoDelete bool
		internal   bool
		noWait     bool
		args       amqp.Table
	}
	queuesDeclared []struct {
		name       string
		durable    bool
		autoDelete bool
		exclusive  bool
		noWait     bool
		args       amqp.Table
	}
	queueBinds []struct {
		name     string
		key      string
		exchange string
		noWait   bool
		args     amqp.Table
	}

	// consumer/ack simulation
	delivered chan amqp.Delivery
	acks      []uint64
	nacks     []struct {
		tag      uint64
		multiple bool
		requeue  bool
	}
}

func newMockChannel() *mockChannel {
	return &mockChannel{autoConfirm: true}
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

func (m *mockChannel) Confirm(noWait bool) error {
	m.mu.Lock()
	m.confirmEnabled = true
	err := m.confirmErr
	m.mu.Unlock()
	return err
}

func (m *mockChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return fmt.Errorf("channel closed")
	}
	call := publishCall{exchange: exchange, routingKey: key, mandatory: mandatory, immediate: immediate, message: msg}
	m.publishCalls = append(m.publishCalls, call)
	callIndex := len(m.publishCalls)
	var err error
	if len(m.publishErrors) > 0 {
		err = m.publishErrors[0]
		m.publishErrors = m.publishErrors[1:]
	}
	notify := m.notifyPublish
	confirmEnabled := m.confirmEnabled
	autoConfirm := m.autoConfirm
	var confirm *amqp.Confirmation
	if len(m.confirmResponses) > 0 {
		c := m.confirmResponses[0]
		m.confirmResponses = m.confirmResponses[1:]
		confirm = &c
	}
	m.mu.Unlock()

	if err != nil {
		return err
	}

	if confirmEnabled && notify != nil {
		if confirm != nil {
			notify <- *confirm
		} else if autoConfirm {
			notify <- amqp.Confirmation{DeliveryTag: uint64(callIndex), Ack: true}
		}
	}

	return nil
}

func (m *mockChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	m.mu.Lock()
	m.exchangesDeclared = append(m.exchangesDeclared, struct {
		name       string
		kind       string
		durable    bool
		autoDelete bool
		internal   bool
		noWait     bool
		args       amqp.Table
	}{name: name, kind: kind, durable: durable, autoDelete: autoDelete, internal: internal, noWait: noWait, args: args})
	m.mu.Unlock()
	return nil
}

func (m *mockChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	m.mu.Lock()
	m.queuesDeclared = append(m.queuesDeclared, struct {
		name       string
		durable    bool
		autoDelete bool
		exclusive  bool
		noWait     bool
		args       amqp.Table
	}{name: name, durable: durable, autoDelete: autoDelete, exclusive: exclusive, noWait: noWait, args: args})
	m.mu.Unlock()
	return amqp.Queue{Name: name}, nil
}

func (m *mockChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	m.mu.Lock()
	m.queueBinds = append(m.queueBinds, struct {
		name     string
		key      string
		exchange string
		noWait   bool
		args     amqp.Table
	}{name: name, key: key, exchange: exchange, noWait: noWait, args: args})
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

func (m *mockChannel) NotifyPublish(receiver chan amqp.Confirmation) chan amqp.Confirmation {
	m.mu.Lock()
	if m.closed {
		close(receiver)
		m.mu.Unlock()
		return receiver
	}
	m.notifyPublish = receiver
	m.mu.Unlock()
	return receiver
}

func (m *mockChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	m.mu.Lock()
	if m.delivered == nil {
		m.delivered = make(chan amqp.Delivery, 100)
	}
	ch := m.delivered
	m.mu.Unlock()
	return ch, nil
}

func (m *mockChannel) Ack(tag uint64, multiple bool) error {
	m.mu.Lock()
	m.acks = append(m.acks, tag)
	m.mu.Unlock()
	return nil
}

func (m *mockChannel) Nack(tag uint64, multiple, requeue bool) error {
	m.mu.Lock()
	m.nacks = append(m.nacks, struct {
		tag      uint64
		multiple bool
		requeue  bool
	}{tag: tag, multiple: multiple, requeue: requeue})
	m.mu.Unlock()
	return nil
}

func (m *mockChannel) enqueueDelivery(d amqp.Delivery) {
	m.mu.Lock()
	if m.delivered == nil {
		m.delivered = make(chan amqp.Delivery, 100)
	}
	ch := m.delivered
	m.mu.Unlock()
	ch <- d
}

func (m *mockChannel) triggerClose() {
	m.mu.Lock()
	if m.closed {
		notify := m.notify
		notifyPublish := m.notifyPublish
		m.mu.Unlock()
		if notify != nil {
			close(notify)
		}
		if notifyPublish != nil {
			close(notifyPublish)
		}
		return
	}
	notify := m.notify
	notifyPublish := m.notifyPublish
	m.closed = true
	m.mu.Unlock()

	if notify != nil {
		notify <- &amqp.Error{Code: 541, Reason: "channel closed"}
		close(notify)
	}
	if notifyPublish != nil {
		close(notifyPublish)
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

func (m *mockChannel) lastPublish() *publishCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.publishCalls) == 0 {
		return nil
	}
	call := m.publishCalls[len(m.publishCalls)-1]
	return &call
}

func (m *mockChannel) queuePublishError(err error) {
	m.mu.Lock()
	m.publishErrors = append(m.publishErrors, err)
	m.mu.Unlock()
}

func (m *mockChannel) queueConfirmation(confirm amqp.Confirmation) {
	m.mu.Lock()
	m.confirmResponses = append(m.confirmResponses, confirm)
	m.mu.Unlock()
}

func (m *mockChannel) ackedTags() []uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]uint64, len(m.acks))
	copy(result, m.acks)
	return result
}

func (m *mockChannel) nackedTags() []uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]uint64, 0, len(m.nacks))
	for _, n := range m.nacks {
		result = append(result, n.tag)
	}
	return result
}
