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
)

// Evaluator 策略评估器
type Evaluator struct {
	client Client
	config *Config
}

// NewEvaluator 创建评估器
func NewEvaluator(client Client) *Evaluator {
	return &Evaluator{
		client: client,
	}
}

// NewEvaluatorWithConfig 使用配置创建评估器
func NewEvaluatorWithConfig(ctx context.Context, config *Config) (*Evaluator, error) {
	client, err := NewClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Evaluator{
		client: client,
		config: config,
	}, nil
}

// Evaluate 评估策略
func (e *Evaluator) Evaluate(ctx context.Context, path string, input interface{}) (*Result, error) {
	return e.client.Evaluate(ctx, path, input)
}

// IsAllowed 评估是否允许（便捷方法）
func (e *Evaluator) IsAllowed(ctx context.Context, path string, input interface{}) (bool, error) {
	result, err := e.client.Evaluate(ctx, path, input)
	if err != nil {
		return false, err
	}

	return result.Allowed, nil
}

// EvaluateWithDefault 评估策略，失败时返回默认值
func (e *Evaluator) EvaluateWithDefault(ctx context.Context, path string, input interface{}, defaultValue bool) bool {
	result, err := e.client.Evaluate(ctx, path, input)
	if err != nil {
		return defaultValue
	}

	return result.Allowed
}

// BatchEvaluate 批量评估策略
func (e *Evaluator) BatchEvaluate(ctx context.Context, path string, inputs []interface{}) ([]*Result, error) {
	results := make([]*Result, len(inputs))

	for i, input := range inputs {
		result, err := e.client.Evaluate(ctx, path, input)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate input %d: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

// EvaluateRBAC 评估 RBAC 策略
func (e *Evaluator) EvaluateRBAC(ctx context.Context, input *RBACInput) (*Result, error) {
	path := "rbac.allow"
	if e.config != nil && e.config.DefaultDecisionPath != "" {
		path = e.config.DefaultDecisionPath
	}

	return e.client.Evaluate(ctx, path, input)
}

// EvaluateABAC 评估 ABAC 策略
func (e *Evaluator) EvaluateABAC(ctx context.Context, input *ABACInput) (*Result, error) {
	path := "abac.allow"
	if e.config != nil && e.config.DefaultDecisionPath != "" {
		path = e.config.DefaultDecisionPath
	}

	return e.client.Evaluate(ctx, path, input)
}

// Close 关闭评估器
func (e *Evaluator) Close(ctx context.Context) error {
	return e.client.Close(ctx)
}

// RBACInput RBAC 输入结构
type RBACInput struct {
	// Subject 主体（用户）
	Subject *Subject `json:"subject"`

	// Action 动作（操作）
	Action string `json:"action"`

	// Resource 资源
	Resource string `json:"resource"`

	// Context 上下文信息
	Context map[string]interface{} `json:"context,omitempty"`
}

// ABACInput ABAC 输入结构
type ABACInput struct {
	// Subject 主体（用户）
	Subject *Subject `json:"subject"`

	// Action 动作（操作）
	Action string `json:"action"`

	// Resource 资源
	Resource *Resource `json:"resource"`

	// Environment 环境信息
	Environment *Environment `json:"environment,omitempty"`

	// Context 上下文信息
	Context map[string]interface{} `json:"context,omitempty"`
}

// Subject 主体信息
type Subject struct {
	// User 用户 ID
	User string `json:"user"`

	// Roles 角色列表
	Roles []string `json:"roles,omitempty"`

	// Groups 用户组列表
	Groups []string `json:"groups,omitempty"`

	// Attributes 用户属性
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// Resource 资源信息
type Resource struct {
	// Type 资源类型
	Type string `json:"type"`

	// ID 资源 ID
	ID string `json:"id"`

	// Owner 资源所有者
	Owner string `json:"owner,omitempty"`

	// Attributes 资源属性
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// Environment 环境信息
type Environment struct {
	// Time 当前时间
	Time string `json:"time,omitempty"`

	// IPAddress IP 地址
	IPAddress string `json:"ip_address,omitempty"`

	// Location 地理位置
	Location string `json:"location,omitempty"`

	// DeviceType 设备类型
	DeviceType string `json:"device_type,omitempty"`

	// Attributes 环境属性
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}
