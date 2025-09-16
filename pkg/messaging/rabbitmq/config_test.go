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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/stretchr/testify/require"
)

func TestParseConfigDefaults(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}

	cfg, err := ParseConfig(base)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, 16, cfg.Channels.MaxPerConnection)
	require.Equal(t, 50, cfg.QoS.PrefetchCount)
	require.Equal(t, time.Duration(10*time.Second), cfg.Timeouts.DialTimeout())
	require.True(t, cfg.Reconnect.Enabled)
	require.NotNil(t, cfg.Topology.Exchanges)
}

func TestParseConfigOverrides(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
		Extra: map[string]any{
			"rabbitmq": map[string]any{
				"channels": map[string]any{
					"max_per_connection": 4,
					"acquire_timeout":    "750ms",
				},
				"qos": map[string]any{
					"prefetch_count": 12,
					"global":         true,
				},
				"timeouts": map[string]any{
					"dial": "3s",
				},
			},
		},
	}

	cfg, err := ParseConfig(base)
	require.NoError(t, err)
	require.Equal(t, 4, cfg.Channels.MaxPerConnection)
	require.Equal(t, 12, cfg.QoS.PrefetchCount)
	require.True(t, cfg.QoS.Global)
	require.Equal(t, 750*time.Millisecond, cfg.Channels.AcquireTimeoutDuration())
	require.Equal(t, 3*time.Second, cfg.Timeouts.DialTimeout())
}

func TestParseConfigInlineExtra(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
		Extra: map[string]any{
			"channels": map[string]any{
				"max_per_connection": 2,
			},
		},
	}

	cfg, err := ParseConfig(base)
	require.NoError(t, err)
	require.Equal(t, 2, cfg.Channels.MaxPerConnection)
}
