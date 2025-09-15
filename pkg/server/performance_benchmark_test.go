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
