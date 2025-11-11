// Copyright (c) 2024 Six-Thirty Labs, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opa

import (
	"context"
	"fmt"
	"os"
	"sync"
)

// Manager 策略管理器
type Manager struct {
	client Client
	mu     sync.RWMutex
}

// NewManager 创建策略管理器
func NewManager(client Client) *Manager {
	return &Manager{
		client: client,
	}
}

// NewManagerWithConfig 使用配置创建管理器
func NewManagerWithConfig(ctx context.Context, config *Config) (*Manager, error) {
	client, err := NewClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Manager{
		client: client,
	}, nil
}

// LoadPolicy 从字符串加载策略
func (m *Manager) LoadPolicy(ctx context.Context, name string, policy string) error {
	if !m.client.IsEmbedded() {
		return fmt.Errorf("policy management is only supported in embedded mode")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.client.LoadPolicy(ctx, name, policy)
}

// LoadPolicyFromFile 从文件加载策略
func (m *Manager) LoadPolicyFromFile(ctx context.Context, name string, filePath string) error {
	if !m.client.IsEmbedded() {
		return fmt.Errorf("policy management is only supported in embedded mode")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	return m.LoadPolicy(ctx, name, string(content))
}

// LoadPoliciesFromDir 从目录加载所有策略
func (m *Manager) LoadPoliciesFromDir(ctx context.Context, dir string) error {
	if !m.client.IsEmbedded() {
		return fmt.Errorf("policy management is only supported in embedded mode")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 只处理 .rego 文件
		if len(entry.Name()) < 5 || entry.Name()[len(entry.Name())-5:] != ".rego" {
			continue
		}

		filePath := dir + "/" + entry.Name()
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read policy file %s: %w", filePath, err)
		}

		if err := m.client.LoadPolicy(ctx, entry.Name(), string(content)); err != nil {
			return fmt.Errorf("failed to load policy %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// RemovePolicy 移除策略
func (m *Manager) RemovePolicy(ctx context.Context, name string) error {
	if !m.client.IsEmbedded() {
		return fmt.Errorf("policy management is only supported in embedded mode")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.client.RemovePolicy(ctx, name)
}

// UpdatePolicy 更新策略
func (m *Manager) UpdatePolicy(ctx context.Context, name string, policy string) error {
	if !m.client.IsEmbedded() {
		return fmt.Errorf("policy management is only supported in embedded mode")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 先移除旧策略，再加载新策略
	if err := m.client.RemovePolicy(ctx, name); err != nil {
		return fmt.Errorf("failed to remove old policy: %w", err)
	}

	if err := m.client.LoadPolicy(ctx, name, policy); err != nil {
		return fmt.Errorf("failed to load new policy: %w", err)
	}

	return nil
}

// UpdatePolicyFromFile 从文件更新策略
func (m *Manager) UpdatePolicyFromFile(ctx context.Context, name string, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	return m.UpdatePolicy(ctx, name, string(content))
}

// ReloadPolicy 重新加载策略（先移除再加载）
func (m *Manager) ReloadPolicy(ctx context.Context, name string, policy string) error {
	return m.UpdatePolicy(ctx, name, policy)
}

// ReloadPolicyFromFile 从文件重新加载策略
func (m *Manager) ReloadPolicyFromFile(ctx context.Context, name string, filePath string) error {
	return m.UpdatePolicyFromFile(ctx, name, filePath)
}

// Close 关闭管理器
func (m *Manager) Close(ctx context.Context) error {
	return m.client.Close(ctx)
}

// GetClient 获取底层客户端
func (m *Manager) GetClient() Client {
	return m.client
}

// PolicyInfo 策略信息
type PolicyInfo struct {
	// Name 策略名称
	Name string `json:"name"`

	// Path 策略路径
	Path string `json:"path,omitempty"`

	// Size 策略大小（字节）
	Size int64 `json:"size,omitempty"`

	// ModTime 修改时间
	ModTime string `json:"mod_time,omitempty"`
}

