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
	"strings"

	"github.com/open-policy-agent/opa/rego"
)

// Client OPA 客户端接口
type Client interface {
	// Evaluate 评估策略决策
	Evaluate(ctx context.Context, path string, input interface{}) (*Result, error)

	// Query 执行 Rego 查询
	Query(ctx context.Context, query string, input interface{}) (rego.ResultSet, error)

	// PartialEvaluate 执行部分评估
	PartialEvaluate(ctx context.Context, query string, input interface{}) (*PartialResult, error)

	// LoadPolicy 加载策略（仅嵌入式模式支持）
	LoadPolicy(ctx context.Context, name string, policy string) error

	// LoadPolicyFromFile 从文件加载策略（仅嵌入式模式支持）
	LoadPolicyFromFile(ctx context.Context, path string) error

	// LoadData 加载数据到 OPA store（仅嵌入式模式支持）
	LoadData(ctx context.Context, path string, data interface{}) error

	// RemovePolicy 移除策略（仅嵌入式模式支持）
	RemovePolicy(ctx context.Context, name string) error

	// Close 关闭客户端
	Close(ctx context.Context) error

	// IsEmbedded 是否为嵌入式模式
	IsEmbedded() bool
}

// Result 策略评估结果
type Result struct {
	// Decision 决策结果（通常是 bool 或 map）
	Decision interface{} `json:"decision"`

	// DecisionID 决策 ID（用于审计和追踪）
	DecisionID string `json:"decision_id,omitempty"`

	// Allowed 便捷的布尔结果（当决策为 bool 时）
	Allowed bool `json:"allowed"`

	// Metrics 评估指标
	Metrics *Metrics `json:"metrics,omitempty"`
}

// Metrics 评估指标
type Metrics struct {
	// TimerEvalNs 评估时间（纳秒）
	TimerEvalNs int64 `json:"timer_eval_ns,omitempty"`

	// TimerRegoQueryEvalNs Rego 查询评估时间（纳秒）
	TimerRegoQueryEvalNs int64 `json:"timer_rego_query_eval_ns,omitempty"`
}

// PartialResult 部分评估结果
type PartialResult struct {
	// Queries 部分评估后的查询集合
	Queries []interface{} `json:"queries"`

	// Support 支持规则集合
	Support []interface{} `json:"support,omitempty"`
}

// NewClient 创建 OPA 客户端
func NewClient(ctx context.Context, config *Config) (Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	config.SetDefaults()

	switch config.Mode {
	case ModeEmbedded:
		return newEmbeddedClient(ctx, config)
	case ModeRemote:
		return newRemoteClient(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported mode: %s", config.Mode)
	}
}

// ClientOption 客户端选项函数
type ClientOption func(*Config)

// WithEmbeddedMode 设置为嵌入式模式
func WithEmbeddedMode(policyDir string) ClientOption {
	return func(c *Config) {
		c.Mode = ModeEmbedded
		if c.EmbeddedConfig == nil {
			c.EmbeddedConfig = &EmbeddedConfig{}
		}
		c.EmbeddedConfig.PolicyDir = policyDir
	}
}

// WithRemoteMode 设置为远程模式
func WithRemoteMode(url string) ClientOption {
	return func(c *Config) {
		c.Mode = ModeRemote
		if c.RemoteConfig == nil {
			c.RemoteConfig = &RemoteConfig{}
		}
		c.RemoteConfig.URL = url
	}
}

// WithCache 启用缓存
func WithCache(maxSize int, ttl int64) ClientOption {
	return func(c *Config) {
		if c.CacheConfig == nil {
			c.CacheConfig = &CacheConfig{}
		}
		c.CacheConfig.Enabled = true
		c.CacheConfig.MaxSize = maxSize
	}
}

// WithDefaultDecisionPath 设置默认决策路径
func WithDefaultDecisionPath(path string) ClientOption {
	return func(c *Config) {
		c.DefaultDecisionPath = path
	}
}

// NewClientWithOptions 使用选项创建客户端
func NewClientWithOptions(ctx context.Context, opts ...ClientOption) (Client, error) {
	config := &Config{}
	for _, opt := range opts {
		opt(config)
	}
	return NewClient(ctx, config)
}

// normalizePath 规范化决策路径格式
// 接受两种格式：
// - 点分隔（用于嵌入式模式）: "rbac.allow"
// - 斜杠分隔（用于 REST API）: "rbac/allow"
// 返回两种格式：dotPath（用于 Rego 查询）和 slashPath（用于 REST URL）
func normalizePath(path string) (dotPath, slashPath string) {
	if path == "" {
		return "", ""
	}

	// 检测是否为斜杠分隔格式
	if strings.Contains(path, "/") {
		// 转换 rbac/allow -> rbac.allow
		slashPath = path
		dotPath = strings.ReplaceAll(path, "/", ".")
	} else {
		// 已经是点分隔格式
		dotPath = path
		slashPath = strings.ReplaceAll(path, ".", "/")
	}

	return dotPath, slashPath
}
