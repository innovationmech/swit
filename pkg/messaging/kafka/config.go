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

package kafka

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// Config represents Kafka-specific adapter configuration derived from BrokerConfig.Extra.
// This keeps only a minimal but useful subset to validate adapter-specific options.
type Config struct {
	// Timeouts groups dial/read/write timeouts used by clients
	Timeouts TimeoutConfig `json:"timeouts" yaml:"timeouts"`

	// Producer controls producer-specific options
	Producer ProducerConfig `json:"producer" yaml:"producer"`

	// Compression selects the compression algorithm
	Compression messaging.CompressionType `json:"compression" yaml:"compression"`
}

// TimeoutConfig configures client timeouts.
type TimeoutConfig struct {
	Dial  messaging.Duration `json:"dial" yaml:"dial"`
	Read  messaging.Duration `json:"read" yaml:"read"`
	Write messaging.Duration `json:"write" yaml:"write"`
}

// ProducerConfig configures producer behaviour.
type ProducerConfig struct {
	// Acks specifies acknowledgement strategy: none | leader | all
	Acks string `json:"acks" yaml:"acks"`

	// Idempotent enables idempotent producer (requires appropriate broker settings)
	Idempotent bool `json:"idempotent" yaml:"idempotent"`

	// Batching controls basic batching strategy
	Batching ProducerBatchingConfig `json:"batching" yaml:"batching"`

	// Compression overrides global compression for producer (optional)
	// Supported values follow messaging.CompressionType
	Compression messaging.CompressionType `json:"compression" yaml:"compression"`
}

// ProducerBatchingConfig controls producer batching.
type ProducerBatchingConfig struct {
	// MaxBytes maximum batch bytes; 0 = library default
	MaxBytes int `json:"max_bytes" yaml:"max_bytes"`

	// Timeout flush timeout between batches
	Timeout messaging.Duration `json:"timeout" yaml:"timeout"`
}

// DefaultConfig returns a configuration pre-populated with sensible Kafka defaults.
func DefaultConfig() *Config {
	return &Config{
		Timeouts: TimeoutConfig{
			Dial:  messaging.Duration(10 * time.Second),
			Read:  messaging.Duration(30 * time.Second),
			Write: messaging.Duration(30 * time.Second),
		},
		Producer: ProducerConfig{
			Acks:       "leader",
			Idempotent: false,
			Batching: ProducerBatchingConfig{
				MaxBytes: 0,
				Timeout:  messaging.Duration(100 * time.Millisecond),
			},
		},
		Compression: messaging.CompressionNone,
	}
}

// ParseConfig extracts Kafka adapter configuration from the generic BrokerConfig.
func ParseConfig(base *messaging.BrokerConfig) (*Config, error) {
	if base == nil {
		return nil, messaging.NewConfigError("broker config cannot be nil", nil)
	}

	cfg := DefaultConfig()
	if base.Extra == nil {
		return cfg, nil
	}

	raw, ok := base.Extra["kafka"]
	if !ok {
		// Support configs that inline Kafka settings at the root of Extra.
		raw = base.Extra
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil, messaging.NewConfigError("failed to marshal kafka extra config", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, messaging.NewConfigError("failed to unmarshal kafka config", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg.normalize()
	return cfg, nil
}

func (c *Config) normalize() {
	// Normalization ensures positive non-zero defaults
	if c.Timeouts.Dial <= 0 {
		c.Timeouts.Dial = messaging.Duration(10 * time.Second)
	}
	if c.Timeouts.Read <= 0 {
		c.Timeouts.Read = messaging.Duration(30 * time.Second)
	}
	if c.Timeouts.Write <= 0 {
		c.Timeouts.Write = messaging.Duration(30 * time.Second)
	}
	if c.Producer.Batching.Timeout <= 0 {
		c.Producer.Batching.Timeout = messaging.Duration(100 * time.Millisecond)
	}
}

func (c *Config) validate() error {
	// Validate acks enumeration
	switch c.Producer.Acks {
	case "", "none", "leader", "all":
		// empty allowed → will be normalized to default later
	default:
		return messaging.NewConfigError(
			fmt.Sprintf("invalid producer.acks: %s (allowed: none, leader, all)", c.Producer.Acks),
			nil,
		)
	}

	// Validate compression type against known set
	switch c.Compression {
	case messaging.CompressionNone, messaging.CompressionGZIP, messaging.CompressionSnappy,
		messaging.CompressionLZ4, messaging.CompressionZSTD:
		// ok
	default:
		return messaging.NewConfigError(
			fmt.Sprintf("invalid compression type: %s", c.Compression),
			nil,
		)
	}

	// Validate numeric ranges
	if c.Producer.Batching.MaxBytes < 0 {
		return messaging.NewConfigError("producer.batching.max_bytes cannot be negative", nil)
	}

	// Validate optional producer compression when provided
	switch c.Producer.Compression {
	case "":
		// not set
	case messaging.CompressionNone, messaging.CompressionGZIP, messaging.CompressionSnappy,
		messaging.CompressionLZ4, messaging.CompressionZSTD:
		// ok
	default:
		return messaging.NewConfigError(
			fmt.Sprintf("invalid producer.compression: %s", c.Producer.Compression),
			nil,
		)
	}

	return nil
}

// Helper duration getters
func (t *TimeoutConfig) DialTimeout() time.Duration  { return time.Duration(t.Dial) }
func (t *TimeoutConfig) ReadTimeout() time.Duration  { return time.Duration(t.Read) }
func (t *TimeoutConfig) WriteTimeout() time.Duration { return time.Duration(t.Write) }
