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

	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
)

// Manager 服务发现管理器
// 提供统一的服务发现实例管理，支持配置化和单例模式
type Manager struct {
	instances map[string]*ServiceDiscovery
	mu        sync.RWMutex
}

var (
	manager     *Manager
	managerOnce sync.Once
)

// GetManager 获取服务发现管理器单例
func GetManager() *Manager {
	managerOnce.Do(func() {
		manager = &Manager{
			instances: make(map[string]*ServiceDiscovery),
		}
		logger.GetLogger().Info("Service discovery manager initialized")
	})
	return manager
}

// GetServiceDiscovery 获取指定地址的服务发现实例
// 如果实例不存在，会创建新的实例
func (m *Manager) GetServiceDiscovery(address string) (*ServiceDiscovery, error) {
	m.mu.RLock()
	if sd, exists := m.instances[address]; exists {
		m.mu.RUnlock()
		logger.GetLogger().Debug("Using existing service discovery instance",
			zap.String("address", address))
		return sd, nil
	}
	m.mu.RUnlock()

	// 需要创建新实例
	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查，避免并发创建
	if sd, exists := m.instances[address]; exists {
		logger.GetLogger().Debug("Service discovery instance created by concurrent request",
			zap.String("address", address))
		return sd, nil
	}

	// 创建新的服务发现实例
	logger.GetLogger().Info("Creating new service discovery instance",
		zap.String("address", address))

	sd, err := NewServiceDiscovery(address)
	if err != nil {
		logger.GetLogger().Error("Failed to create service discovery instance",
			zap.String("address", address),
			zap.Error(err))
		return nil, err
	}

	m.instances[address] = sd
	logger.GetLogger().Info("Service discovery instance created successfully",
		zap.String("address", address))
	return sd, nil
}

// GetDefaultServiceDiscovery 获取默认的服务发现实例
// 使用默认的 Consul 地址 "127.0.0.1:8500"
func (m *Manager) GetDefaultServiceDiscovery() (*ServiceDiscovery, error) {
	return m.GetServiceDiscovery("127.0.0.1:8500")
}

// Close 关闭所有服务发现实例
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	instanceCount := len(m.instances)
	if instanceCount > 0 {
		logger.GetLogger().Info("Closing service discovery instances",
			zap.Int("count", instanceCount))
	}

	// 清理所有实例
	for address := range m.instances {
		logger.GetLogger().Debug("Closing service discovery instance",
			zap.String("address", address))
		delete(m.instances, address)
	}

	if instanceCount > 0 {
		logger.GetLogger().Info("All service discovery instances closed")
	}
}

// 便捷函数，提供向后兼容性

// GetServiceDiscoveryByAddress 根据地址获取服务发现实例
func GetServiceDiscoveryByAddress(address string) (*ServiceDiscovery, error) {
	return GetManager().GetServiceDiscovery(address)
}

// GetDefaultServiceDiscovery 获取默认的服务发现实例
func GetDefaultServiceDiscovery() (*ServiceDiscovery, error) {
	return GetManager().GetDefaultServiceDiscovery()
}
