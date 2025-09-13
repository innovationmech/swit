// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Licensed under the MIT License.
package server

import (
	"context"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// BenchmarkMessagingLifecycleOverhead 对比启用/禁用 messaging 的启动时间开销
func BenchmarkMessagingLifecycleOverhead(b *testing.B) {
	cfgBase := &ServerConfig{}
	cfgBase.SetDefaults()
	cfgBase.ServiceName = "bench-svc"
	cfgBase.HTTP.Enabled = false
	cfgBase.GRPC.Enabled = false

	// 禁用 messaging
	cfgNoMsg := cfgBase.DeepCopy()
	cfgNoMsg.Messaging.Enabled = false

	// 启用 messaging（使用 InMemory 配置占位）
	cfgWithMsg := cfgBase.DeepCopy()
	cfgWithMsg.Messaging.Enabled = true
	cfgWithMsg.Messaging.DefaultBroker = "local"
	cfgWithMsg.Messaging.Brokers = map[string]BrokerConfig{
		"local": {
			Type:      string(messaging.BrokerTypeInMemory),
			Endpoints: []string{"localhost:9092"},
		},
	}

	registrar := NewNoopServiceRegistrar()
	deps := NewSimpleBusinessDependencyContainer()

	// 构造无 messaging 的服务
	srvNoMsg, err := NewBusinessServerCore(cfgNoMsg, registrar, deps)
	if err != nil {
		b.Fatal(err)
	}

	// 构造启用 messaging 的服务
	srvWithMsg, err := NewBusinessServerCore(cfgWithMsg, registrar, NewSimpleBusinessDependencyContainer())
	if err != nil {
		b.Fatal(err)
	}

	b.Run("startup-no-messaging", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = srvNoMsg.Start(ctx)
			_ = srvNoMsg.Stop(ctx)
			cancel()
		}
	})

	b.Run("startup-with-messaging", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = srvWithMsg.Start(ctx)
			_ = srvWithMsg.Stop(ctx)
			cancel()
		}
	})
}

// NewNoopServiceRegistrar 提供空注册器，避免引入额外依赖
type NoopServiceRegistrar struct{}

func NewNoopServiceRegistrar() *NoopServiceRegistrar { return &NoopServiceRegistrar{} }

func (n *NoopServiceRegistrar) RegisterServices(registry BusinessServiceRegistry) error { return nil }
