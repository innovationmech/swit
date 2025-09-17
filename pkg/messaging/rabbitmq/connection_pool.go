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
	"context"
	"fmt"
	mathrand "math/rand"
	"sync"
	"time"

	"github.com/streadway/amqp"

	"github.com/innovationmech/swit/pkg/messaging"
)

type dialFunc func(endpoint string, cfg amqp.Config) (amqpConnection, error)

type amqpConnection interface {
	Channel() (amqpChannel, error)
	Close() error
	IsClosed() bool
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
	// NotifyBlocked provides connection-level flow control notifications
	// so clients can react to broker back-pressure and update health.
	NotifyBlocked(receiver chan amqp.Blocking) chan amqp.Blocking
}

type amqpChannel interface {
	Close() error
	Qos(prefetchCount, prefetchSize int, global bool) error
	Confirm(noWait bool) error
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	// Topology management
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
	NotifyPublish(receiver chan amqp.Confirmation) chan amqp.Confirmation
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Ack(tag uint64, multiple bool) error
	Nack(tag uint64, multiple, requeue bool) error
}

type connectionPool struct {
	config       *messaging.BrokerConfig
	rabbitConfig *Config
	dial         dialFunc

	mu          sync.Mutex
	initialized bool
	closed      bool

	connections   []*pooledConnection
	available     chan *pooledConnection
	endpointIndex int
}

func newConnectionPool(config *messaging.BrokerConfig, rabbitCfg *Config) *connectionPool {
	poolSize := config.Connection.PoolSize
	if poolSize <= 0 {
		poolSize = 1
	}
	return &connectionPool{
		config:       config,
		rabbitConfig: rabbitCfg,
		dial:         dialAMQP,
		available:    make(chan *pooledConnection, poolSize),
	}
}

func (p *connectionPool) Initialize(ctx context.Context) error {
	p.mu.Lock()
	if p.initialized {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()

	for i := 0; i < p.maxConnections(); i++ {
		conn, err := p.createConnectionWithFailover(ctx)
		if err != nil {
			return err
		}
		p.mu.Lock()
		p.connections = append(p.connections, conn)
		p.mu.Unlock()
		p.available <- conn
	}

	p.mu.Lock()
	p.initialized = true
	p.mu.Unlock()
	return nil
}

func (p *connectionPool) maxConnections() int {
	size := p.config.Connection.PoolSize
	if size <= 0 {
		size = 1
	}
	return size
}

func (p *connectionPool) createConnectionWithFailover(ctx context.Context) (*pooledConnection, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Backoff parameters
	backoff := p.rabbitConfig.Reconnect.InitialBackoff()
	if backoff <= 0 {
		backoff = 500 * time.Millisecond
	}
	maxBackoff := p.rabbitConfig.Reconnect.MaxBackoff()
	if maxBackoff <= 0 {
		maxBackoff = 30 * time.Second
	}
	jitterPct := p.rabbitConfig.Reconnect.JitterPercent
	maxRetries := p.rabbitConfig.Reconnect.MaxRetries

	attempts := 0
	for {
		attempts++

		endpoint := p.nextEndpoint()
		cfg := amqp.Config{
			Locale:    "en_US",
			Heartbeat: p.rabbitConfig.Timeouts.HeartbeatInterval(),
			Dial:      amqp.DefaultDial(p.rabbitConfig.Timeouts.DialTimeout()),
		}

		conn, err := p.dial(endpoint, cfg)
		if err == nil {
			pooled := newPooledConnection(p, conn, endpoint)
			pooled.startWatch()
			return pooled, nil
		}

		// Stop when max retries reached (if configured)
		if maxRetries > 0 && attempts >= maxRetries {
			return nil, messaging.NewConnectionError(
				fmt.Sprintf("rabbitmq: failed to connect after %d attempts; last endpoint=%s", attempts, endpoint),
				err,
			)
		}

		// Sleep with backoff and jitter or exit on context cancellation
		delay := addJitter(backoff, jitterPct)
		if delay > maxBackoff {
			delay = maxBackoff
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}

		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (p *connectionPool) Acquire(ctx context.Context) (*pooledConnection, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case conn, ok := <-p.available:
		if !ok || conn == nil {
			return nil, messaging.NewConnectionError("rabbitmq connection pool closed", nil)
		}
		if err := conn.ensureOpen(ctx); err != nil {
			p.Release(conn)
			return nil, err
		}
		conn.markInUse()
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *connectionPool) Release(conn *pooledConnection) {
	if conn == nil {
		return
	}

	conn.markIdle()

	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()

	if closed {
		return
	}

	p.available <- conn
}

func (p *connectionPool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	close(p.available)
	connections := append([]*pooledConnection(nil), p.connections...)
	p.mu.Unlock()

	for _, conn := range connections {
		conn.Close()
	}
}

func (p *connectionPool) nextEndpoint() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.config.Endpoints) == 0 {
		return "amqp://localhost:5672"
	}
	endpoint := p.config.Endpoints[p.endpointIndex%len(p.config.Endpoints)]
	p.endpointIndex++
	return endpoint
}

func (p *connectionPool) redial(endpoint string) (amqpConnection, error) {
	cfg := amqp.Config{
		Locale:    "en_US",
		Heartbeat: p.rabbitConfig.Timeouts.HeartbeatInterval(),
		Dial:      amqp.DefaultDial(p.rabbitConfig.Timeouts.DialTimeout()),
	}
	return p.dial(endpoint, cfg)
}

func dialAMQP(endpoint string, cfg amqp.Config) (amqpConnection, error) {
	conn, err := amqp.DialConfig(endpoint, cfg)
	if err != nil {
		return nil, err
	}
	return &realConnection{conn: conn}, nil
}

type realConnection struct {
	conn *amqp.Connection
}

func (r *realConnection) Channel() (amqpChannel, error) {
	ch, err := r.conn.Channel()
	if err != nil {
		return nil, err
	}
	return &realChannel{ch: ch}, nil
}

func (r *realConnection) Close() error {
	return r.conn.Close()
}

func (r *realConnection) IsClosed() bool {
	return r.conn.IsClosed()
}

func (r *realConnection) NotifyClose(receiver chan *amqp.Error) chan *amqp.Error {
	return r.conn.NotifyClose(receiver)
}

func (r *realConnection) NotifyBlocked(receiver chan amqp.Blocking) chan amqp.Blocking {
	return r.conn.NotifyBlocked(receiver)
}

type realChannel struct {
	ch *amqp.Channel
}

func (r *realChannel) Close() error {
	return r.ch.Close()
}

func (r *realChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	return r.ch.Qos(prefetchCount, prefetchSize, global)
}

func (r *realChannel) Confirm(noWait bool) error {
	return r.ch.Confirm(noWait)
}

func (r *realChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return r.ch.Publish(exchange, key, mandatory, immediate, msg)
}

func (r *realChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	return r.ch.ExchangeDeclare(name, kind, durable, autoDelete, internal, noWait, args)
}

func (r *realChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	return r.ch.QueueDeclare(name, durable, autoDelete, exclusive, noWait, args)
}

func (r *realChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	return r.ch.QueueBind(name, key, exchange, noWait, args)
}

func (r *realChannel) NotifyClose(receiver chan *amqp.Error) chan *amqp.Error {
	return r.ch.NotifyClose(receiver)
}

func (r *realChannel) NotifyPublish(receiver chan amqp.Confirmation) chan amqp.Confirmation {
	return r.ch.NotifyPublish(receiver)
}

func (r *realChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	return r.ch.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
}

func (r *realChannel) Ack(tag uint64, multiple bool) error {
	return r.ch.Ack(tag, multiple)
}

func (r *realChannel) Nack(tag uint64, multiple, requeue bool) error {
	return r.ch.Nack(tag, multiple, requeue)
}

type pooledConnection struct {
	pool           *connectionPool
	endpoint       string
	mu             sync.Mutex
	conn           amqpConnection
	inUse          bool
	closed         bool
	notify         chan *amqp.Error
	blocked        bool
	blockedCh      chan amqp.Blocking
	lastBlockedAt  time.Time
	lastFailoverAt time.Time
	channels       chan *channelWrapper
	max            int
	total          int
}

func newPooledConnection(pool *connectionPool, conn amqpConnection, endpoint string) *pooledConnection {
	max := pool.rabbitConfig.Channels.MaxPerConnection
	if max <= 0 {
		max = 1
	}
	return &pooledConnection{
		pool:     pool,
		endpoint: endpoint,
		conn:     conn,
		channels: make(chan *channelWrapper, max),
		max:      max,
	}
}

func (p *pooledConnection) startWatch() {
	if p.conn == nil {
		return
	}
	notify := p.conn.NotifyClose(make(chan *amqp.Error, 1))
	p.mu.Lock()
	p.notify = notify
	p.mu.Unlock()

	go func() {
		if _, ok := <-notify; ok {
			p.mu.Lock()
			p.closed = true
			p.mu.Unlock()
		}
	}()

	// Watch for connection-level blocked/unblocked events
	blockedCh := p.conn.NotifyBlocked(make(chan amqp.Blocking, 1))
	if blockedCh != nil {
		p.mu.Lock()
		p.blockedCh = blockedCh
		p.mu.Unlock()
		go func() {
			for ev := range blockedCh {
				p.mu.Lock()
				p.blocked = ev.Active
				if ev.Active {
					p.lastBlockedAt = time.Now()
				}
				p.mu.Unlock()
			}
		}()
	}
}

func (p *pooledConnection) markInUse() {
	p.mu.Lock()
	p.inUse = true
	p.mu.Unlock()
}

func (p *pooledConnection) markIdle() {
	p.mu.Lock()
	p.inUse = false
	p.mu.Unlock()
}

func (p *pooledConnection) ensureOpen(ctx context.Context) error {
	p.mu.Lock()
	closed := p.closed || p.conn == nil || p.conn.IsClosed()
	endpoint := p.endpoint
	p.mu.Unlock()

	if !closed {
		return nil
	}

	if !p.pool.rabbitConfig.Reconnect.Enabled {
		return messaging.NewConnectionError("rabbitmq connection closed", nil)
	}

	backoff := p.pool.rabbitConfig.Reconnect.InitialBackoff()
	if backoff <= 0 {
		backoff = 500 * time.Millisecond
	}
	maxBackoff := p.pool.rabbitConfig.Reconnect.MaxBackoff()
	if maxBackoff <= 0 {
		maxBackoff = 30 * time.Second
	}

	attempts := 0
	var old chan *channelWrapper
	for {
		attempts++
		conn, err := p.pool.redial(endpoint)
		if err == nil {
			p.mu.Lock()
			if p.conn != nil {
				_ = p.conn.Close()
			}
			p.conn = conn
			p.closed = false
			p.total = 0
			p.endpoint = endpoint
			old = p.channels
			p.channels = make(chan *channelWrapper, p.max)
			p.lastFailoverAt = time.Now()
			p.mu.Unlock()
			p.startWatch()
		DrainLoop:
			for {
				select {
				case ch := <-old:
					if ch != nil {
						_ = ch.Close()
					}
				default:
					break DrainLoop
				}
			}
			return nil
		}

		if p.pool.rabbitConfig.Reconnect.MaxRetries > 0 && attempts >= p.pool.rabbitConfig.Reconnect.MaxRetries {
			return messaging.NewConnectionError("rabbitmq connection reconnect attempts exhausted", err)
		}

		delay := backoff
		if delay > maxBackoff {
			delay = maxBackoff
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}

		endpoint = p.pool.nextEndpoint()
	}
}

func (p *pooledConnection) AcquireChannel(ctx context.Context) (*channelWrapper, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	timeout := p.pool.rabbitConfig.Channels.AcquireTimeoutDuration()
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	for {
		select {
		case ch := <-p.channels:
			if ch == nil {
				continue
			}
			if ch.isClosed() {
				p.decrementChannels()
				continue
			}
			ch.markInUse()
			return ch, nil
		default:
		}

		p.mu.Lock()
		if p.total < p.max {
			p.total++
			p.mu.Unlock()

			channel, err := p.conn.Channel()
			if err != nil {
				p.decrementChannels()
				return nil, messaging.NewConnectionError("failed to open channel", err)
			}
			wrapper := newChannelWrapper(p, channel)
			if err := wrapper.applyQoS(p.pool.rabbitConfig.QoS); err != nil {
				_ = wrapper.Close()
				p.decrementChannels()
				return nil, messaging.NewConfigError("failed to apply channel QoS", err)
			}
			wrapper.markInUse()
			return wrapper, nil
		}
		p.mu.Unlock()

		select {
		case ch := <-p.channels:
			if ch == nil {
				continue
			}
			if ch.isClosed() {
				p.decrementChannels()
				continue
			}
			ch.markInUse()
			return ch, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(timeout):
			return nil, messaging.NewConnectionError("channel pool exhausted", nil)
		}
	}
}

func (p *pooledConnection) ReleaseChannel(ch *channelWrapper) {
	if ch == nil {
		return
	}
	if ch.isClosed() {
		p.decrementChannels()
		return
	}

	if p.pool.rabbitConfig.Channels.IdleTTLDuration() > 0 && ch.idleFor() > p.pool.rabbitConfig.Channels.IdleTTLDuration() {
		_ = ch.Close()
		p.decrementChannels()
		return
	}

	ch.markIdle()
	select {
	case p.channels <- ch:
	default:
		_ = ch.Close()
		p.decrementChannels()
	}
}

func (p *pooledConnection) decrementChannels() {
	p.mu.Lock()
	if p.total > 0 {
		p.total--
	}
	p.mu.Unlock()
}

func (p *pooledConnection) Close() {
	p.mu.Lock()
	conn := p.conn
	p.conn = nil
	p.closed = true
	channels := p.channels
	p.channels = make(chan *channelWrapper)
	p.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}

	for {
		select {
		case ch := <-channels:
			if ch != nil {
				_ = ch.Close()
			}
		default:
			return
		}
	}
}

// addJitter adds +/- jitterPercent% randomization to a base duration.
func addJitter(base time.Duration, jitterPercent int) time.Duration {
	if jitterPercent <= 0 {
		return base
	}
	// Bound jitter to [0, 100]
	if jitterPercent > 100 {
		jitterPercent = 100
	}
	// Randomize in range [1 - j, 1 + j]
	jitter := float64(jitterPercent) / 100.0
	min := 1.0 - jitter
	max := 1.0 + jitter
	factor := min + (max-min)*randFloat64()
	return time.Duration(float64(base) * factor)
}

// randFloat64 returns a pseudo-random number in [0.0, 1.0).
// We avoid importing math/rand globally to keep deterministic tests unless needed.
var _randSeedOnce sync.Once
var _randSrc *mathrand.Rand

func randFloat64() float64 {
	_randSeedOnce.Do(func() {
		_randSrc = mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	})
	return _randSrc.Float64()
}

type channelWrapper struct {
	parent  *pooledConnection
	channel amqpChannel

	mu       sync.Mutex
	closed   bool
	lastUsed time.Time
}

func newChannelWrapper(parent *pooledConnection, ch amqpChannel) *channelWrapper {
	wrapper := &channelWrapper{
		parent:   parent,
		channel:  ch,
		lastUsed: time.Now(),
	}
	wrapper.watchClose()
	return wrapper
}

func (c *channelWrapper) applyQoS(cfg QoSConfig) error {
	if cfg.PrefetchCount == 0 && cfg.PrefetchSize == 0 {
		return nil
	}
	return c.channel.Qos(cfg.PrefetchCount, cfg.PrefetchSize, cfg.Global)
}

func (c *channelWrapper) watchClose() {
	notify := c.channel.NotifyClose(make(chan *amqp.Error, 1))
	if notify == nil {
		return
	}

	go func() {
		if _, ok := <-notify; ok {
			c.mu.Lock()
			c.closed = true
			c.mu.Unlock()
		}
	}()
}

func (c *channelWrapper) markInUse() {
	c.mu.Lock()
	c.lastUsed = time.Now()
	c.mu.Unlock()
}

func (c *channelWrapper) markIdle() {
	c.mu.Lock()
	c.lastUsed = time.Now()
	c.mu.Unlock()
}

func (c *channelWrapper) idleFor() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Since(c.lastUsed)
}

func (c *channelWrapper) isClosed() bool {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	return closed
}

func (c *channelWrapper) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()
	return c.channel.Close()
}
