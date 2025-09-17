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

package nats

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/nats-io/nats.go"
)

// natsBroker is a minimal broker implementing messaging.MessageBroker for NATS.
// It provides Connect/Disconnect/IsConnected and stub publisher/subscriber factories.
type natsBroker struct {
	config  *messaging.BrokerConfig
	cfg     *Config
	metrics messaging.BrokerMetrics
	conn    *nats.Conn
	js      nats.JetStreamContext
	mu      sync.RWMutex
	started bool
}

func newNATSBroker(base *messaging.BrokerConfig, cfg *Config) *natsBroker {
	return &natsBroker{config: base, cfg: cfg}
}

func (b *natsBroker) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.started {
		return nil
	}

	if len(b.config.Endpoints) == 0 {
		return messaging.NewConnectionError("no NATS endpoints configured", nil)
	}

	opts := []nats.Option{
		nats.Timeout(b.cfg.Timeouts.DialTimeout()),
		nats.PingInterval(b.cfg.Timeouts.PingInterval()),
	}

	// Reconnect options
	if b.cfg.Reconnect.Enabled {
		opts = append(opts,
			nats.MaxReconnects(b.cfg.Reconnect.MaxAttempts),
			nats.ReconnectWait(time.Duration(b.cfg.Reconnect.Wait)),
			nats.ReconnectJitter(time.Duration(b.cfg.Reconnect.Jitter), time.Duration(b.cfg.Reconnect.Jitter)),
		)
	}

	// TLS options
	if b.config.TLS != nil && b.config.TLS.Enabled {
		opts = append(opts, nats.Secure(&tls.Config{InsecureSkipVerify: b.cfg.TLSInsecureSkipVerify}))
	}

	// Auth options (basic token / username/password via URL supported by nats.go)
	if b.config.Authentication != nil {
		switch b.config.Authentication.Type {
		case messaging.AuthTypeAPIKey, messaging.AuthTypeJWT:
			if b.config.Authentication.Token != "" {
				opts = append(opts, nats.Token(b.config.Authentication.Token))
			}
		case messaging.AuthTypeSASL: // Map to user/pass
			if b.config.Authentication.Username != "" || b.config.Authentication.Password != "" {
				opts = append(opts, nats.UserInfo(b.config.Authentication.Username, b.config.Authentication.Password))
			}
		}
	}

	// Connect using first endpoint initially; nats.Connect accepts multiple URLs separated by comma
	url := b.config.Endpoints[0]
	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return messaging.NewConnectionError("failed to connect to NATS", err)
	}

	b.conn = conn

	// Initialize JetStream if enabled
	if b.cfg.JetStream != nil && b.cfg.JetStream.Enabled {
		js, err := conn.JetStream(
			nats.PublishAsyncMaxPending(256),
		)
		if err != nil {
			return messaging.NewConnectionError("failed to initialize JetStream", err)
		}
		b.js = js

		// Ensure streams and consumers declared in configuration
		if err := b.ensureJetStreamTopology(ctx); err != nil {
			return err
		}
	}

	b.started = true
	b.metrics.ConnectionStatus = "connected"
	b.metrics.LastConnectionTime = time.Now()

	return nil
}

func (b *natsBroker) Disconnect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.started {
		return nil
	}
	if b.conn != nil {
		b.conn.Drain()
		b.conn.Close()
		b.conn = nil
	}
	b.started = false
	b.metrics.ConnectionStatus = "disconnected"
	return nil
}

func (b *natsBroker) Close() error {
	return b.Disconnect(context.Background())
}

func (b *natsBroker) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.started
}

func (b *natsBroker) CreatePublisher(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
	b.mu.RLock()
	started := b.started
	conn := b.conn
	b.mu.RUnlock()
	if !started || conn == nil {
		return nil, messaging.ErrBrokerNotConnected
	}
	if config.Topic == "" {
		return nil, messaging.NewConfigError("publisher topic is required", nil)
	}
	return newPublisher(conn, b.js, &config), nil
}

func (b *natsBroker) CreateSubscriber(config messaging.SubscriberConfig) (messaging.EventSubscriber, error) {
	b.mu.RLock()
	started := b.started
	conn := b.conn
	b.mu.RUnlock()
	if !started || conn == nil {
		return nil, messaging.ErrBrokerNotConnected
	}
	if len(config.Topics) == 0 {
		return nil, messaging.NewConfigError("subscriber topics are required", nil)
	}
	if config.ConsumerGroup == "" {
		// For core NATS, map ConsumerGroup to queue group (optional). Allow empty for simple subscribers.
		config.ConsumerGroup = ""
	}
	return newSubscriber(conn, b.js, b.cfg.JetStream, &config), nil
}

func (b *natsBroker) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	status := &messaging.HealthStatus{
		Status:       messaging.HealthStatusHealthy,
		Message:      "nats broker scaffold healthy",
		LastChecked:  time.Now(),
		ResponseTime: 0,
		Details: map[string]any{
			"connected": b.IsConnected(),
		},
	}
	return status, nil
}

func (b *natsBroker) GetMetrics() *messaging.BrokerMetrics {
	copy := b.metrics.GetSnapshot()
	return copy
}

func (b *natsBroker) GetCapabilities() *messaging.BrokerCapabilities {
	caps, _ := messaging.GetCapabilityProfile(messaging.BrokerTypeNATS)
	// Signal extended capability for NATS KV store (JS required)
	if b.js != nil {
		caps.Extended["nats.kv"] = true
	}
	return caps
}

// GetJetStream implements JetStreamProvider for exposing JS context to utils.
func (b *natsBroker) GetJetStream() nats.JetStreamContext {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.js
}

// ensureJetStreamTopology creates or updates declared streams and consumers.
func (b *natsBroker) ensureJetStreamTopology(ctx context.Context) error {
	if b.js == nil || b.cfg.JetStream == nil || !b.cfg.JetStream.Enabled {
		return nil
	}

	// Streams
	for _, s := range b.cfg.JetStream.Streams {
		sc := toNATSStreamConfig(&s)
		if _, err := b.js.AddStream(sc); err != nil {
			// Try update if exists or partial conflict
			if _, uerr := b.js.UpdateStream(sc); uerr != nil {
				return messaging.NewConnectionError("failed to create/update JetStream stream", err)
			}
		}
	}

	// Consumers
	for _, c := range b.cfg.JetStream.Consumers {
		cc := toNATSConsumerConfig(&c)
		// Add or update consumer on specific stream
		if _, err := b.js.AddConsumer(c.Stream, cc); err != nil {
			if _, uerr := b.js.UpdateConsumer(c.Stream, cc); uerr != nil {
				return messaging.NewConnectionError("failed to create/update JetStream consumer", err)
			}
		}
	}

	return nil
}

// Helpers to translate our config into nats.go configs
func toNATSStreamConfig(in *JSStreamConfig) *nats.StreamConfig {
	var retention nats.RetentionPolicy
	switch in.Retention {
	case "workqueue":
		retention = nats.WorkQueuePolicy
	case "interest":
		retention = nats.InterestPolicy
	default:
		retention = nats.LimitsPolicy
	}

	var storage nats.StorageType
	switch in.Storage {
	case "memory":
		storage = nats.MemoryStorage
	default:
		storage = nats.FileStorage
	}

	return &nats.StreamConfig{
		Name:       in.Name,
		Subjects:   append([]string{}, in.Subjects...),
		Retention:  retention,
		MaxBytes:   in.MaxBytes,
		MaxAge:     time.Duration(in.MaxAge),
		MaxMsgs:    in.MaxMsgs,
		MaxMsgSize: in.MaxMsgSize,
		Storage:    storage,
		Replicas:   in.Replicas,
	}
}

func toNATSConsumerConfig(in *JSConsumerConfig) *nats.ConsumerConfig {
	var dp nats.DeliverPolicy
	switch in.DeliverPolicy {
	case "last":
		dp = nats.DeliverLastPolicy
	case "new":
		dp = nats.DeliverNewPolicy
	case "by_start_sequence":
		dp = nats.DeliverByStartSequencePolicy
	case "by_start_time":
		dp = nats.DeliverByStartTimePolicy
	default:
		dp = nats.DeliverAllPolicy
	}

	var ap nats.AckPolicy
	switch in.AckPolicy {
	case "none":
		ap = nats.AckNonePolicy
	case "all":
		ap = nats.AckAllPolicy
	default:
		ap = nats.AckExplicitPolicy
	}

	var rp nats.ReplayPolicy
	switch in.ReplayPolicy {
	case "original":
		rp = nats.ReplayOriginalPolicy
	default:
		rp = nats.ReplayInstantPolicy
	}

	durable := ""
	if in.Durable {
		durable = in.Name
	}

	return &nats.ConsumerConfig{
		Name:           in.Name,
		Durable:        durable,
		DeliverPolicy:  dp,
		AckPolicy:      ap,
		AckWait:        time.Duration(in.AckWait),
		MaxDeliver:     in.MaxDeliver,
		FilterSubject:  in.FilterSubject,
		ReplayPolicy:   rp,
		MaxAckPending:  in.MaxAckPending,
		DeliverSubject: in.DeliverSubject,
		DeliverGroup:   in.DeliverGroup,
	}
}
