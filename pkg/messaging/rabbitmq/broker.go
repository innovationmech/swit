package rabbitmq

import (
	"context"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

type rabbitBroker struct {
	config       *messaging.BrokerConfig
	rabbitConfig *Config

	mu      sync.RWMutex
	started bool

	pool    *connectionPool
	metrics messaging.BrokerMetrics
}

func newRabbitBroker(config *messaging.BrokerConfig, rabbitCfg *Config) *rabbitBroker {
	clone := cloneBrokerConfig(config)
	pool := newConnectionPool(clone, rabbitCfg)
	return &rabbitBroker{
		config:       clone,
		rabbitConfig: rabbitCfg,
		pool:         pool,
		metrics: messaging.BrokerMetrics{
			Extended: map[string]any{
				"channel_max_per_connection": rabbitCfg.Channels.MaxPerConnection,
				"reconnect_enabled":          rabbitCfg.Reconnect.Enabled,
			},
		},
	}
}

func (b *rabbitBroker) Connect(ctx context.Context) error {
	b.mu.Lock()
	if b.started {
		b.mu.Unlock()
		return nil
	}
	b.metrics.ConnectionAttempts++
	b.mu.Unlock()

	if err := b.pool.Initialize(ctx); err != nil {
		b.mu.Lock()
		b.metrics.ConnectionFailures++
		b.metrics.LastConnectionError = err.Error()
		b.mu.Unlock()
		return err
	}

	b.mu.Lock()
	b.started = true
	b.metrics.ConnectionStatus = "connected"
	b.metrics.LastConnectionTime = time.Now()
	b.metrics.OpenConnections = int64(b.pool.maxConnections())
	b.metrics.LastConnectionError = ""
	b.mu.Unlock()

	return nil
}

func (b *rabbitBroker) Disconnect(ctx context.Context) error {
	_ = ctx
	b.mu.Lock()
	pool := b.pool
	b.pool = newConnectionPool(b.config, b.rabbitConfig)
	b.started = false
	b.metrics.ConnectionStatus = "disconnected"
	b.metrics.OpenConnections = 0
	b.mu.Unlock()

	if pool != nil {
		pool.Close()
	}

	return nil
}

func (b *rabbitBroker) Close() error {
	return b.Disconnect(context.Background())
}

func (b *rabbitBroker) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.started
}

func (b *rabbitBroker) CreatePublisher(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
	_ = config
	return nil, &messaging.MessagingError{
		Code:      messaging.ErrNotImplemented,
		Message:   "rabbitmq publisher not implemented",
		Retryable: false,
		Timestamp: time.Now(),
	}
}

func (b *rabbitBroker) CreateSubscriber(config messaging.SubscriberConfig) (messaging.EventSubscriber, error) {
	_ = config
	return nil, &messaging.MessagingError{
		Code:      messaging.ErrNotImplemented,
		Message:   "rabbitmq subscriber not implemented",
		Retryable: false,
		Timestamp: time.Now(),
	}
}

func (b *rabbitBroker) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	_ = ctx
	b.mu.RLock()
	connected := b.started
	poolSize := b.pool.maxConnections()
	b.mu.RUnlock()

	status := &messaging.HealthStatus{
		Status:       messaging.HealthStatusHealthy,
		Message:      "rabbitmq adapter ready",
		LastChecked:  time.Now(),
		ResponseTime: 0,
		Details: map[string]any{
			"connected":             connected,
			"connection_pool_size":  poolSize,
			"channel_pool_capacity": b.rabbitConfig.Channels.MaxPerConnection,
			"reconnect_enabled":     b.rabbitConfig.Reconnect.Enabled,
		},
	}

	if !connected {
		status.Status = messaging.HealthStatusDegraded
		status.Message = "rabbitmq adapter not connected"
	}

	return status, nil
}

func (b *rabbitBroker) GetMetrics() *messaging.BrokerMetrics {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.started {
		b.metrics.ConnectionUptime = time.Since(b.metrics.LastConnectionTime)
	} else {
		b.metrics.ConnectionUptime = 0
	}

	b.metrics.LastMetricsUpdate = time.Now()
	snapshot := b.metrics.GetSnapshot()
	return snapshot
}

func (b *rabbitBroker) GetCapabilities() *messaging.BrokerCapabilities {
	caps, err := messaging.GetCapabilityProfile(messaging.BrokerTypeRabbitMQ)
	if err != nil {
		return &messaging.BrokerCapabilities{}
	}
	return caps
}

func cloneBrokerConfig(cfg *messaging.BrokerConfig) *messaging.BrokerConfig {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	if cfg.Endpoints != nil {
		clone.Endpoints = append([]string(nil), cfg.Endpoints...)
	}
	if cfg.Extra != nil {
		clone.Extra = make(map[string]any, len(cfg.Extra))
		for k, v := range cfg.Extra {
			clone.Extra[k] = v
		}
	}
	return &clone
}
