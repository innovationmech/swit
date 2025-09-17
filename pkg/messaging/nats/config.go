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
	"encoding/json"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// Config represents NATS-specific adapter configuration derived from BrokerConfig.Extra.
// This focuses on core connection/reconnect/TLS toggles for the initial scaffold.
type Config struct {
	// Reconnect controls automatic reconnection behavior
	Reconnect ReconnectConfig `json:"reconnect" yaml:"reconnect"`

	// Timeouts controls dial and operation timeouts
	Timeouts TimeoutConfig `json:"timeouts" yaml:"timeouts"`

	// TLSInsecureSkipVerify mirrors nats.go option for skipping TLS verification
	TLSInsecureSkipVerify bool `json:"tls_insecure_skip_verify" yaml:"tls_insecure_skip_verify"`

	// JetStream controls NATS JetStream related configurations including
	// stream and consumer management. When enabled, the broker will attempt
	// to create or update declared streams/consumers on Connect.
	JetStream *JetStreamConfig `json:"jetstream" yaml:"jetstream"`
}

// ReconnectConfig tunes automatic reconnection behaviour.
type ReconnectConfig struct {
	Enabled     bool               `json:"enabled" yaml:"enabled"`
	MaxAttempts int                `json:"max_attempts" yaml:"max_attempts"`
	Wait        messaging.Duration `json:"wait" yaml:"wait"`
	Jitter      messaging.Duration `json:"jitter" yaml:"jitter"`
}

// TimeoutConfig configures dial and request timeouts.
type TimeoutConfig struct {
	Dial    messaging.Duration `json:"dial" yaml:"dial"`
	Request messaging.Duration `json:"request" yaml:"request"`
	Ping    messaging.Duration `json:"ping" yaml:"ping"`
}

// DefaultConfig returns a configuration with sensible NATS defaults.
func DefaultConfig() *Config {
	return &Config{
		Reconnect: ReconnectConfig{
			Enabled:     true,
			MaxAttempts: 0, // 0 = unlimited in nats.go semantics
			Wait:        messaging.Duration(2 * time.Second),
			Jitter:      messaging.Duration(500 * time.Millisecond),
		},
		Timeouts: TimeoutConfig{
			Dial:    messaging.Duration(5 * time.Second),
			Request: messaging.Duration(5 * time.Second),
			Ping:    messaging.Duration(2 * time.Minute),
		},
		TLSInsecureSkipVerify: false,
		JetStream:             &JetStreamConfig{Enabled: false},
	}
}

// ParseConfig extracts NATS adapter configuration from the generic BrokerConfig.
func ParseConfig(base *messaging.BrokerConfig) (*Config, error) {
	if base == nil {
		return nil, messaging.NewConfigError("broker config cannot be nil", nil)
	}

	cfg := DefaultConfig()
	if base.Extra == nil {
		return cfg, nil
	}

	raw, ok := base.Extra["nats"]
	if !ok {
		// Support inline form
		raw = base.Extra
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil, messaging.NewConfigError("failed to marshal nats extra config", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, messaging.NewConfigError("failed to unmarshal nats config", err)
	}

	cfg.normalize()
	return cfg, nil
}

func (c *Config) normalize() {
	if c.Reconnect.MaxAttempts < 0 {
		c.Reconnect.MaxAttempts = 0
	}
	if c.Reconnect.Wait <= 0 {
		c.Reconnect.Wait = messaging.Duration(2 * time.Second)
	}
	if c.Timeouts.Dial <= 0 {
		c.Timeouts.Dial = messaging.Duration(5 * time.Second)
	}
	if c.Timeouts.Request <= 0 {
		c.Timeouts.Request = messaging.Duration(5 * time.Second)
	}
	if c.Timeouts.Ping <= 0 {
		c.Timeouts.Ping = messaging.Duration(2 * time.Minute)
	}
	if c.JetStream == nil {
		c.JetStream = &JetStreamConfig{Enabled: false}
	}
	c.JetStream.normalize()
}

// Helpers
func (t *TimeoutConfig) DialTimeout() time.Duration    { return time.Duration(t.Dial) }
func (t *TimeoutConfig) RequestTimeout() time.Duration { return time.Duration(t.Request) }
func (t *TimeoutConfig) PingInterval() time.Duration   { return time.Duration(t.Ping) }

// JetStream configuration types and helpers

// JetStreamConfig is the top-level JetStream configuration.
type JetStreamConfig struct {
	Enabled   bool               `json:"enabled" yaml:"enabled"`
	Streams   []JSStreamConfig   `json:"streams" yaml:"streams"`
	Consumers []JSConsumerConfig `json:"consumers" yaml:"consumers"`
}

// JSStreamConfig declares a stream and its subjects.
type JSStreamConfig struct {
	Name       string             `json:"name" yaml:"name"`
	Subjects   []string           `json:"subjects" yaml:"subjects"`
	Retention  string             `json:"retention" yaml:"retention"` // limits|workqueue|interest
	MaxBytes   int64              `json:"max_bytes" yaml:"max_bytes"`
	MaxAge     messaging.Duration `json:"max_age" yaml:"max_age"`
	MaxMsgs    int64              `json:"max_msgs" yaml:"max_msgs"`
	MaxMsgSize int32              `json:"max_msg_size" yaml:"max_msg_size"`
	Storage    string             `json:"storage" yaml:"storage"` // file|memory
	Replicas   int                `json:"replicas" yaml:"replicas"`
}

// JSConsumerConfig declares a consumer to be created/updated for a stream.
type JSConsumerConfig struct {
	Name          string             `json:"name" yaml:"name"`
	Stream        string             `json:"stream" yaml:"stream"`
	Durable       bool               `json:"durable" yaml:"durable"`
	DeliverPolicy string             `json:"deliver_policy" yaml:"deliver_policy"` // all|last|new|by_start_sequence|by_start_time
	AckPolicy     string             `json:"ack_policy" yaml:"ack_policy"`         // none|all|explicit
	AckWait       messaging.Duration `json:"ack_wait" yaml:"ack_wait"`
	MaxDeliver    int                `json:"max_deliver" yaml:"max_deliver"`
	FilterSubject string             `json:"filter_subject" yaml:"filter_subject"`
	ReplayPolicy  string             `json:"replay_policy" yaml:"replay_policy"` // instant|original
	MaxAckPending int                `json:"max_ack_pending" yaml:"max_ack_pending"`
	// Push delivery options
	DeliverSubject string `json:"deliver_subject" yaml:"deliver_subject"`
	DeliverGroup   string `json:"deliver_group" yaml:"deliver_group"`
}

func (j *JetStreamConfig) normalize() {
	if j == nil {
		return
	}
	// No-op defaults for now; keep values as provided.
}
