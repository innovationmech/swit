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

package discovery

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetManager(t *testing.T) {
	t.Run("singleton pattern", func(t *testing.T) {
		// 重置单例状态用于测试
		resetManagerSingleton()

		manager1 := GetManager()
		manager2 := GetManager()

		assert.Same(t, manager1, manager2, "GetManager should return the same instance")
		assert.NotNil(t, manager1.instances, "Manager instances map should be initialized")
	})

	t.Run("concurrent access", func(t *testing.T) {
		// 重置单例状态用于测试
		resetManagerSingleton()

		var wg sync.WaitGroup
		managers := make([]*Manager, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				managers[idx] = GetManager()
			}(i)
		}

		wg.Wait()

		// 验证所有的 manager 都是同一个实例
		firstManager := managers[0]
		for i := 1; i < len(managers); i++ {
			assert.Same(t, firstManager, managers[i], "All managers should be the same instance")
		}
	})
}

func TestManager_GetServiceDiscovery(t *testing.T) {
	manager := &Manager{
		instances: make(map[string]*ServiceDiscovery),
	}

	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{
			name:    "valid address",
			address: "127.0.0.1:8500",
			wantErr: false,
		},
		{
			name:    "localhost address",
			address: "localhost:8500",
			wantErr: false,
		},
		{
			name:    "different port",
			address: "127.0.0.1:8501",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd, err := manager.GetServiceDiscovery(tt.address)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, sd)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, sd)

				// 验证实例被缓存
				sd2, err2 := manager.GetServiceDiscovery(tt.address)
				assert.NoError(t, err2)
				assert.Same(t, sd, sd2, "Should return the same cached instance")
			}
		})
	}
}

func TestManager_GetServiceDiscovery_ConcurrentAccess(t *testing.T) {
	manager := &Manager{
		instances: make(map[string]*ServiceDiscovery),
	}

	address := "127.0.0.1:8500"
	var wg sync.WaitGroup
	results := make([]*ServiceDiscovery, 50)

	// 并发访问同一个地址
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sd, err := manager.GetServiceDiscovery(address)
			require.NoError(t, err)
			results[idx] = sd
		}(i)
	}

	wg.Wait()

	// 验证所有结果都是同一个实例
	firstSD := results[0]
	for i := 1; i < len(results); i++ {
		assert.Same(t, firstSD, results[i], "All ServiceDiscovery instances should be the same")
	}
}

func TestManager_GetDefaultServiceDiscovery(t *testing.T) {
	manager := &Manager{
		instances: make(map[string]*ServiceDiscovery),
	}

	sd, err := manager.GetDefaultServiceDiscovery()
	assert.NoError(t, err)
	assert.NotNil(t, sd)

	// 验证默认地址的实例被缓存
	sd2, err2 := manager.GetDefaultServiceDiscovery()
	assert.NoError(t, err2)
	assert.Same(t, sd, sd2, "Should return the same cached default instance")

	// 验证与直接调用 GetServiceDiscovery 返回的实例相同
	sd3, err3 := manager.GetServiceDiscovery("127.0.0.1:8500")
	assert.NoError(t, err3)
	assert.Same(t, sd, sd3, "Default instance should be the same as direct call")
}

func TestManager_Close(t *testing.T) {
	manager := &Manager{
		instances: make(map[string]*ServiceDiscovery),
	}

	// 添加一些实例
	addresses := []string{"127.0.0.1:8500", "127.0.0.1:8501", "localhost:8500"}
	for _, addr := range addresses {
		_, err := manager.GetServiceDiscovery(addr)
		require.NoError(t, err)
	}

	// 验证实例已创建
	assert.Len(t, manager.instances, len(addresses))

	// 关闭管理器
	manager.Close()

	// 验证所有实例都被清理
	assert.Empty(t, manager.instances)
}

func TestManager_Close_ConcurrentAccess(t *testing.T) {
	manager := &Manager{
		instances: make(map[string]*ServiceDiscovery),
	}

	// 添加一些实例
	addresses := []string{"127.0.0.1:8500", "127.0.0.1:8501"}
	for _, addr := range addresses {
		_, err := manager.GetServiceDiscovery(addr)
		require.NoError(t, err)
	}

	var wg sync.WaitGroup

	// 并发访问和关闭
	wg.Add(1)
	go func() {
		defer wg.Done()
		manager.Close()
	}()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = manager.GetServiceDiscovery("127.0.0.1:8502")
		}()
	}

	wg.Wait()

	// 测试不应该 panic
	assert.True(t, true, "Concurrent access should not panic")
}

func TestGetServiceDiscoveryByAddress(t *testing.T) {
	// 重置单例状态用于测试
	resetManagerSingleton()

	address := "127.0.0.1:8500"
	sd, err := GetServiceDiscoveryByAddress(address)

	assert.NoError(t, err)
	assert.NotNil(t, sd)

	// 验证缓存工作正常
	sd2, err2 := GetServiceDiscoveryByAddress(address)
	assert.NoError(t, err2)
	assert.Same(t, sd, sd2, "Should return the same cached instance")
}

func TestGetDefaultServiceDiscovery_GlobalFunction(t *testing.T) {
	// 重置单例状态用于测试
	resetManagerSingleton()

	sd, err := GetDefaultServiceDiscovery()

	assert.NoError(t, err)
	assert.NotNil(t, sd)

	// 验证缓存工作正常
	sd2, err2 := GetDefaultServiceDiscovery()
	assert.NoError(t, err2)
	assert.Same(t, sd, sd2, "Should return the same cached default instance")

	// 验证与通过地址获取的实例相同
	sd3, err3 := GetServiceDiscoveryByAddress("127.0.0.1:8500")
	assert.NoError(t, err3)
	assert.Same(t, sd, sd3, "Default instance should be the same as direct call")
}

func TestGlobalFunctions_Integration(t *testing.T) {
	// 重置单例状态用于测试
	resetManagerSingleton()

	// 测试全局函数集成
	sd1, err1 := GetDefaultServiceDiscovery()
	require.NoError(t, err1)

	sd2, err2 := GetServiceDiscoveryByAddress("127.0.0.1:8500")
	require.NoError(t, err2)

	sd3, err3 := GetServiceDiscoveryByAddress("127.0.0.1:8501")
	require.NoError(t, err3)

	// 验证默认实例和指定相同地址的实例是同一个
	assert.Same(t, sd1, sd2, "Default and specific address should be the same")

	// 验证不同地址的实例是不同的
	assert.NotSame(t, sd1, sd3, "Different addresses should have different instances")

	// 验证管理器中有正确数量的实例
	manager := GetManager()
	assert.Len(t, manager.instances, 2, "Should have 2 different instances")
}

func TestManager_DoubleCheckLocking(t *testing.T) {
	manager := &Manager{
		instances: make(map[string]*ServiceDiscovery),
	}

	address := "127.0.0.1:8500"
	var wg sync.WaitGroup
	results := make([]*ServiceDiscovery, 100)

	// 大量并发访问同一个地址，测试双重检查锁定
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sd, err := manager.GetServiceDiscovery(address)
			require.NoError(t, err)
			results[idx] = sd
		}(i)
	}

	wg.Wait()

	// 验证所有结果都是同一个实例
	firstSD := results[0]
	for i := 1; i < len(results); i++ {
		assert.Same(t, firstSD, results[i], "All ServiceDiscovery instances should be the same")
	}

	// 验证管理器中只有一个实例
	assert.Len(t, manager.instances, 1, "Should have only 1 instance")
}

// 辅助函数：重置单例状态用于测试
func resetManagerSingleton() {
	manager = nil
	managerOnce = sync.Once{}
}
