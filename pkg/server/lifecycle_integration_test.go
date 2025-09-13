package server

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 轻量 InMemory broker，仅用于本文件的生命周期集成测试。
type miniInMemoryBroker struct{ connected int32 }

func (b *miniInMemoryBroker) Connect(ctx context.Context) error {
	atomic.StoreInt32(&b.connected, 1)
	return nil
}
func (b *miniInMemoryBroker) Disconnect(ctx context.Context) error {
	atomic.StoreInt32(&b.connected, 0)
	return nil
}
func (b *miniInMemoryBroker) Close() error      { atomic.StoreInt32(&b.connected, 0); return nil }
func (b *miniInMemoryBroker) IsConnected() bool { return atomic.LoadInt32(&b.connected) == 1 }
func (b *miniInMemoryBroker) CreatePublisher(_ messaging.PublisherConfig) (messaging.EventPublisher, error) {
	return nil, nil
}
func (b *miniInMemoryBroker) CreateSubscriber(_ messaging.SubscriberConfig) (messaging.EventSubscriber, error) {
	return &dummySubscriber{}, nil
}
func (b *miniInMemoryBroker) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	return &messaging.HealthStatus{Status: messaging.HealthStatusHealthy, Message: "ok", LastChecked: time.Now()}, nil
}
func (b *miniInMemoryBroker) GetMetrics() *messaging.BrokerMetrics { return &messaging.BrokerMetrics{} }
func (b *miniInMemoryBroker) GetCapabilities() *messaging.BrokerCapabilities {
	return &messaging.BrokerCapabilities{}
}

type dummySubscriber struct{}

func (*dummySubscriber) Subscribe(ctx context.Context, _ messaging.MessageHandler) error { return nil }
func (*dummySubscriber) SubscribeWithMiddleware(ctx context.Context, _ messaging.MessageHandler, _ ...messaging.Middleware) error {
	return nil
}
func (*dummySubscriber) Unsubscribe(ctx context.Context) error                    { return nil }
func (*dummySubscriber) Pause(ctx context.Context) error                          { return nil }
func (*dummySubscriber) Resume(ctx context.Context) error                         { return nil }
func (*dummySubscriber) Seek(ctx context.Context, _ messaging.SeekPosition) error { return nil }
func (*dummySubscriber) GetLag(ctx context.Context) (int64, error)                { return 0, nil }
func (*dummySubscriber) Close() error                                             { return nil }
func (*dummySubscriber) GetMetrics() *messaging.SubscriberMetrics {
	return &messaging.SubscriberMetrics{}
}

type lifecycleTestHandler struct{ id string }

func (h *lifecycleTestHandler) GetHandlerID() string                                  { return h.id }
func (h *lifecycleTestHandler) GetTopics() []string                                   { return []string{"test-topic"} }
func (h *lifecycleTestHandler) GetBrokerRequirement() string                          { return "" }
func (h *lifecycleTestHandler) Initialize(ctx context.Context) error                  { return nil }
func (h *lifecycleTestHandler) Shutdown(ctx context.Context) error                    { return nil }
func (h *lifecycleTestHandler) Handle(ctx context.Context, message interface{}) error { return nil }
func (h *lifecycleTestHandler) OnError(ctx context.Context, message interface{}, err error) interface{} {
	return "retry"
}

// TestServerMessagingLifecycle 集成：服务启动时完成消息系统启动；停止时完成优雅关停。
func TestServerMessagingLifecycle(t *testing.T) {
	// 注册极简 in-memory broker
	messaging.RegisterBrokerFactory(messaging.BrokerTypeInMemory, func(_ *messaging.BrokerConfig) (messaging.MessageBroker, error) {
		return &miniInMemoryBroker{}, nil
	})

	cfg := &ServerConfig{
		ServiceName: "server-lifecycle-test",
		HTTP: HTTPConfig{
			Port:         "0",
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: GRPCConfig{
			Port:                "0",
			Enabled:             true,
			EnableHealthService: true,
			EnableReflection:    true,
			MaxRecvMsgSize:      4 * 1024 * 1024,
			MaxSendMsgSize:      4 * 1024 * 1024,
			KeepaliveParams: GRPCKeepaliveParams{
				MaxConnectionIdle:     15 * time.Minute,
				MaxConnectionAge:      30 * time.Minute,
				MaxConnectionAgeGrace: 5 * time.Minute,
				Time:                  5 * time.Minute,
				Timeout:               1 * time.Minute,
			},
			KeepalivePolicy: GRPCKeepalivePolicy{MinTime: 5 * time.Minute, PermitWithoutStream: true},
		},
		ShutdownTimeout: 2 * time.Second,
		Discovery:       DiscoveryConfig{Enabled: false},
		Messaging: MessagingConfig{
			Enabled:       true,
			DefaultBroker: "default",
			Brokers: map[string]BrokerConfig{
				"default": {Type: "inmemory", Endpoints: []string{"local"}},
			},
			Connection: MessagingConnectionConfig{
				Timeout:       30 * time.Second,
				KeepAlive:     30 * time.Second,
				MaxAttempts:   3,
				RetryInterval: 1 * time.Second,
				PoolSize:      1,
				IdleTimeout:   1 * time.Second,
			},
			Performance: MessagingPerformanceConfig{
				BatchSize:          1,
				BatchTimeout:       10 * time.Millisecond,
				BufferSize:         1,
				Concurrency:        1,
				PrefetchCount:      1,
				CompressionEnabled: false,
			},
			Monitoring: MessagingMonitoringConfig{
				Enabled:             true,
				MetricsEnabled:      true,
				HealthCheckEnabled:  true,
				HealthCheckInterval: 1 * time.Second,
			},
		},
	}

	registrar := NewMessagingTestServiceRegistrar()
	registrar.AddHTTPHandler(NewTestHTTPHandler("h"))
	registrar.AddHealthCheck(NewTestHealthCheck("h", true))
	registrar.AddEventHandler(&lifecycleTestHandler{id: "h1"})

	deps := NewTestDependencyContainer()

	server, err := NewBusinessServerCore(cfg, registrar, deps)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, server.Start(ctx))

	// 验证已启动
	assert.True(t, server.started)

	// 停止（包含优雅关停路径）
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	require.NoError(t, server.Stop(stopCtx))
}
