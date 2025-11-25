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

package opa

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// Policy 策略定义
type Policy struct {
	// Name 策略名称
	Name string `json:"name"`

	// Content 策略内容（Rego 代码）
	Content string `json:"content"`

	// Version 策略版本
	Version int `json:"version"`

	// Path 策略文件路径（可选）
	Path string `json:"path,omitempty"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`

	// Metadata 策略元数据
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Manager 策略管理器
type Manager struct {
	client   Client
	mu       sync.RWMutex
	policies map[string]*Policy // 策略注册表
	watcher  *Watcher           // 文件监听器
}

// NewManager 创建策略管理器
func NewManager(client Client) *Manager {
	return &Manager{
		client:   client,
		policies: make(map[string]*Policy),
	}
}

// NewManagerWithConfig 使用配置创建管理器
func NewManagerWithConfig(ctx context.Context, config *Config) (*Manager, error) {
	client, err := NewClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Manager{
		client:   client,
		policies: make(map[string]*Policy),
	}, nil
}

// RegisterPolicy 注册策略到管理器
func (m *Manager) RegisterPolicy(ctx context.Context, name string, policy *Policy) error {
	if !m.client.IsEmbedded() {
		return fmt.Errorf("policy management is only supported in embedded mode")
	}

	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	if name == "" {
		return fmt.Errorf("policy name cannot be empty")
	}

	// 验证策略语法
	if err := ValidatePolicy(policy.Content); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 加载策略到 OPA
	if err := m.client.LoadPolicy(ctx, name, policy.Content); err != nil {
		return fmt.Errorf("failed to load policy to OPA: %w", err)
	}

	// 设置策略元数据
	now := time.Now()
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = now
	}
	policy.UpdatedAt = now
	policy.Name = name
	policy.Version = 1

	// 如果策略已存在，增加版本号
	if existing, ok := m.policies[name]; ok {
		policy.Version = existing.Version + 1
	}

	// 保存到注册表
	m.policies[name] = policy

	return nil
}

// LoadPolicy 从字符串加载策略
func (m *Manager) LoadPolicy(ctx context.Context, name string, policy string) error {
	if !m.client.IsEmbedded() {
		return fmt.Errorf("policy management is only supported in embedded mode")
	}

	// 使用 RegisterPolicy 保证一致性
	return m.RegisterPolicy(ctx, name, &Policy{
		Content: policy,
	})
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

		// 使用 LoadPolicy 而不是直接调用 client.LoadPolicy，以确保策略被注册
		if err := m.LoadPolicy(ctx, entry.Name(), string(content)); err != nil {
			return fmt.Errorf("failed to load policy %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// DeletePolicy 删除策略（注销）
func (m *Manager) DeletePolicy(ctx context.Context, name string) error {
	if !m.client.IsEmbedded() {
		return fmt.Errorf("policy management is only supported in embedded mode")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 从 OPA 移除策略
	if err := m.client.RemovePolicy(ctx, name); err != nil {
		return fmt.Errorf("failed to remove policy from OPA: %w", err)
	}

	// 从注册表移除
	delete(m.policies, name)

	return nil
}

// RemovePolicy 移除策略（DeletePolicy 的别名）
func (m *Manager) RemovePolicy(ctx context.Context, name string) error {
	return m.DeletePolicy(ctx, name)
}

// UpdatePolicy 更新策略
func (m *Manager) UpdatePolicy(ctx context.Context, name string, policy *Policy) error {
	if !m.client.IsEmbedded() {
		return fmt.Errorf("policy management is only supported in embedded mode")
	}

	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}

	// 验证策略语法
	if err := ValidatePolicy(policy.Content); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 获取现有策略以保留版本信息
	existing, exists := m.policies[name]
	if !exists {
		return fmt.Errorf("policy %s not found", name)
	}

	// 先移除旧策略，再加载新策略
	if err := m.client.RemovePolicy(ctx, name); err != nil {
		return fmt.Errorf("failed to remove old policy: %w", err)
	}

	if err := m.client.LoadPolicy(ctx, name, policy.Content); err != nil {
		return fmt.Errorf("failed to load new policy: %w", err)
	}

	// 更新策略元数据
	policy.Name = name
	policy.Version = existing.Version + 1
	policy.CreatedAt = existing.CreatedAt
	policy.UpdatedAt = time.Now()

	// 更新注册表
	m.policies[name] = policy

	return nil
}

// UpdatePolicyFromFile 从文件更新策略
func (m *Manager) UpdatePolicyFromFile(ctx context.Context, name string, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	return m.UpdatePolicy(ctx, name, &Policy{
		Content: string(content),
		Path:    filePath,
	})
}

// ReloadPolicy 重新加载策略（先移除再加载）
func (m *Manager) ReloadPolicy(ctx context.Context, name string, policy string) error {
	return m.UpdatePolicy(ctx, name, &Policy{
		Content: policy,
	})
}

// ReloadPolicyFromFile 从文件重新加载策略
func (m *Manager) ReloadPolicyFromFile(ctx context.Context, name string, filePath string) error {
	return m.UpdatePolicyFromFile(ctx, name, filePath)
}

// ListPolicies 列出所有已注册的策略
func (m *Manager) ListPolicies() ([]*Policy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policies := make([]*Policy, 0, len(m.policies))
	for _, policy := range m.policies {
		// 创建副本以避免外部修改
		policyCopy := *policy
		policies = append(policies, &policyCopy)
	}

	return policies, nil
}

// GetPolicy 获取指定策略
func (m *Manager) GetPolicy(name string) (*Policy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policy, ok := m.policies[name]
	if !ok {
		return nil, fmt.Errorf("policy %s not found", name)
	}

	// 返回副本
	policyCopy := *policy
	return &policyCopy, nil
}

// ValidatePolicy 验证策略（在加载前检查语法）
func (m *Manager) ValidatePolicy(policy string) error {
	return ValidatePolicy(policy)
}

// WatchPolicies 监听策略目录的变化
func (m *Manager) WatchPolicies(ctx context.Context) error {
	if !m.client.IsEmbedded() {
		return fmt.Errorf("policy watching is only supported in embedded mode")
	}

	if m.watcher != nil {
		return fmt.Errorf("watcher is already running")
	}

	// 创建文件监听器
	watcher, err := NewWatcher(m, ctx)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	m.mu.Lock()
	m.watcher = watcher
	m.mu.Unlock()

	// 启动监听
	return watcher.Start(ctx)
}

// Close 关闭管理器
func (m *Manager) Close(ctx context.Context) error {
	// 停止文件监听器
	if m.watcher != nil {
		if err := m.watcher.Stop(); err != nil {
			return fmt.Errorf("failed to stop watcher: %w", err)
		}
	}

	return m.client.Close(ctx)
}

// GetClient 获取底层客户端
func (m *Manager) GetClient() Client {
	return m.client
}

// PolicyInfo 策略信息（已废弃，使用 Policy 结构）
// Deprecated: Use Policy instead
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
