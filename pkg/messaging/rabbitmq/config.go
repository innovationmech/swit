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
	"encoding/json"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// Config represents RabbitMQ-specific adapter configuration derived from BrokerConfig.Extra.
type Config struct {
	Channels  ChannelPoolConfig `json:"channels" yaml:"channels"`
	QoS       QoSConfig         `json:"qos" yaml:"qos"`
	Timeouts  TimeoutConfig     `json:"timeouts" yaml:"timeouts"`
	Topology  TopologyConfig    `json:"topology" yaml:"topology"`
	Reconnect ReconnectConfig   `json:"reconnect" yaml:"reconnect"`
}

// ChannelPoolConfig controls AMQP channel pooling behaviour per connection.
type ChannelPoolConfig struct {
	MaxPerConnection int                `json:"max_per_connection" yaml:"max_per_connection"`
	IdleTTL          messaging.Duration `json:"idle_ttl" yaml:"idle_ttl"`
	AcquireTimeout   messaging.Duration `json:"acquire_timeout" yaml:"acquire_timeout"`
}

// QoSConfig configures basic quality-of-service settings applied to channels.
type QoSConfig struct {
	PrefetchCount int  `json:"prefetch_count" yaml:"prefetch_count"`
	PrefetchSize  int  `json:"prefetch_size" yaml:"prefetch_size"`
	Global        bool `json:"global" yaml:"global"`
}

// TimeoutConfig configures dial and heartbeat timeouts for connections and operations.
type TimeoutConfig struct {
	Dial      messaging.Duration `json:"dial" yaml:"dial"`
	Operation messaging.Duration `json:"operation" yaml:"operation"`
	Heartbeat messaging.Duration `json:"heartbeat" yaml:"heartbeat"`
}

// ReconnectConfig tunes automatic reconnection behaviour.
type ReconnectConfig struct {
	Enabled       bool               `json:"enabled" yaml:"enabled"`
	InitialDelay  messaging.Duration `json:"initial_delay" yaml:"initial_delay"`
	MaxDelay      messaging.Duration `json:"max_delay" yaml:"max_delay"`
	MaxRetries    int                `json:"max_retries" yaml:"max_retries"`
	JitterPercent int                `json:"jitter_percent" yaml:"jitter_percent"`
}

// TopologyConfig contains optional exchange/queue setup instructions.
type TopologyConfig struct {
	Exchanges map[string]ExchangeConfig `json:"exchanges" yaml:"exchanges"`
	Queues    map[string]QueueConfig    `json:"queues" yaml:"queues"`
	Bindings  []BindingConfig           `json:"bindings" yaml:"bindings"`
}

// ExchangeConfig represents an exchange declaration definition.
type ExchangeConfig struct {
	Name       string                 `json:"name" yaml:"name"`
	Type       string                 `json:"type" yaml:"type"`
	Durable    bool                   `json:"durable" yaml:"durable"`
	AutoDelete bool                   `json:"auto_delete" yaml:"auto_delete"`
	Internal   bool                   `json:"internal" yaml:"internal"`
	Arguments  map[string]interface{} `json:"arguments" yaml:"arguments"`
}

// QueueConfig represents a queue declaration definition.
type QueueConfig struct {
	Name       string                 `json:"name" yaml:"name"`
	Durable    bool                   `json:"durable" yaml:"durable"`
	AutoDelete bool                   `json:"auto_delete" yaml:"auto_delete"`
	Exclusive  bool                   `json:"exclusive" yaml:"exclusive"`
	Arguments  map[string]interface{} `json:"arguments" yaml:"arguments"`
}

// BindingConfig represents a queue binding definition.
type BindingConfig struct {
	Queue      string                 `json:"queue" yaml:"queue"`
	Exchange   string                 `json:"exchange" yaml:"exchange"`
	RoutingKey string                 `json:"routing_key" yaml:"routing_key"`
	Arguments  map[string]interface{} `json:"arguments" yaml:"arguments"`
}

// DefaultConfig returns a configuration pre-populated with sensible RabbitMQ defaults.
func DefaultConfig() *Config {
	return &Config{
		Channels: ChannelPoolConfig{
			MaxPerConnection: 16,
			IdleTTL:          messaging.Duration(1 * time.Minute),
			AcquireTimeout:   messaging.Duration(5 * time.Second),
		},
		QoS: QoSConfig{
			PrefetchCount: 50,
			PrefetchSize:  0,
			Global:        false,
		},
		Timeouts: TimeoutConfig{
			Dial:      messaging.Duration(10 * time.Second),
			Operation: messaging.Duration(5 * time.Second),
			Heartbeat: messaging.Duration(30 * time.Second),
		},
		Topology: TopologyConfig{
			Exchanges: make(map[string]ExchangeConfig),
			Queues:    make(map[string]QueueConfig),
			Bindings:  make([]BindingConfig, 0),
		},
		Reconnect: ReconnectConfig{
			Enabled:       true,
			InitialDelay:  messaging.Duration(500 * time.Millisecond),
			MaxDelay:      messaging.Duration(30 * time.Second),
			MaxRetries:    0,
			JitterPercent: 20,
		},
	}
}

// ParseConfig extracts RabbitMQ adapter configuration from the generic BrokerConfig.
func ParseConfig(base *messaging.BrokerConfig) (*Config, error) {
	if base == nil {
		return nil, messaging.NewConfigError("broker config cannot be nil", nil)
	}

	cfg := DefaultConfig()
	if base.Extra == nil {
		return cfg, nil
	}

	raw, ok := base.Extra["rabbitmq"]
	if !ok {
		// Support configs that inline RabbitMQ settings at the root of Extra.
		raw = base.Extra
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil, messaging.NewConfigError("failed to marshal rabbitmq extra config", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, messaging.NewConfigError("failed to unmarshal rabbitmq config", err)
	}

	cfg.normalize()
	return cfg, nil
}

func (c *Config) normalize() {
	if c.Channels.MaxPerConnection <= 0 {
		c.Channels.MaxPerConnection = 8
	}
	if c.Channels.IdleTTL <= 0 {
		c.Channels.IdleTTL = messaging.Duration(30 * time.Second)
	}
	if c.Channels.AcquireTimeout <= 0 {
		c.Channels.AcquireTimeout = messaging.Duration(3 * time.Second)
	}
	if c.QoS.PrefetchCount < 0 {
		c.QoS.PrefetchCount = 0
	}
	if c.QoS.PrefetchSize < 0 {
		c.QoS.PrefetchSize = 0
	}
	if c.Topology.Exchanges == nil {
		c.Topology.Exchanges = make(map[string]ExchangeConfig)
	}
	if c.Topology.Queues == nil {
		c.Topology.Queues = make(map[string]QueueConfig)
	}
	if c.Topology.Bindings == nil {
		c.Topology.Bindings = make([]BindingConfig, 0)
	}
	if c.Reconnect.InitialDelay <= 0 {
		c.Reconnect.InitialDelay = messaging.Duration(500 * time.Millisecond)
	}
	if c.Reconnect.MaxDelay < c.Reconnect.InitialDelay {
		c.Reconnect.MaxDelay = messaging.Duration(30 * time.Second)
	}
	if c.Reconnect.JitterPercent < 0 {
		c.Reconnect.JitterPercent = 0
	}
}

// IdleTTLDuration returns the idle TTL as time.Duration.
func (c *ChannelPoolConfig) IdleTTLDuration() time.Duration {
	return time.Duration(c.IdleTTL)
}

// AcquireTimeoutDuration returns the acquire timeout as time.Duration.
func (c *ChannelPoolConfig) AcquireTimeoutDuration() time.Duration {
	return time.Duration(c.AcquireTimeout)
}

// DialTimeout returns the dial timeout duration.
func (t *TimeoutConfig) DialTimeout() time.Duration {
	return time.Duration(t.Dial)
}

// OperationTimeout returns the operation timeout duration.
func (t *TimeoutConfig) OperationTimeout() time.Duration {
	return time.Duration(t.Operation)
}

// HeartbeatInterval returns the heartbeat duration.
func (t *TimeoutConfig) HeartbeatInterval() time.Duration {
	return time.Duration(t.Heartbeat)
}

// InitialBackoff returns the reconnect initial delay.
func (r *ReconnectConfig) InitialBackoff() time.Duration {
	return time.Duration(r.InitialDelay)
}

// MaxBackoff returns the reconnect maximum delay.
func (r *ReconnectConfig) MaxBackoff() time.Duration {
	return time.Duration(r.MaxDelay)
}
