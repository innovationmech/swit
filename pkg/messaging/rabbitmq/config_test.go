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
