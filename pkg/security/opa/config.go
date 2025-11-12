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
	"fmt"
	"time"
)

// Mode 定义 OPA 客户端的运行模式
type Mode string

const (
	// ModeEmbedded 嵌入式模式 - OPA 引擎运行在同一进程中
	ModeEmbedded Mode = "embedded"
	// ModeRemote 远程模式 - 连接到外部 OPA 服务器
	ModeRemote Mode = "remote"
)

// Config 定义 OPA 客户端的配置
type Config struct {
	// Mode 客户端运行模式 (embedded 或 remote)
	Mode Mode `json:"mode" yaml:"mode" mapstructure:"mode"`

	// EmbeddedConfig 嵌入式模式配置
	EmbeddedConfig *EmbeddedConfig `json:"embedded,omitempty" yaml:"embedded,omitempty" mapstructure:"embedded"`

	// RemoteConfig 远程模式配置
	RemoteConfig *RemoteConfig `json:"remote,omitempty" yaml:"remote,omitempty" mapstructure:"remote"`

	// CacheConfig 决策缓存配置
	CacheConfig *CacheConfig `json:"cache,omitempty" yaml:"cache,omitempty" mapstructure:"cache"`

	// DefaultDecisionPath 默认决策路径（例如 "authz/allow"）
	DefaultDecisionPath string `json:"default_decision_path,omitempty" yaml:"default_decision_path,omitempty" mapstructure:"default_decision_path"`
}

// EmbeddedConfig 嵌入式 OPA 配置
type EmbeddedConfig struct {
	// PolicyDir 策略文件目录
	PolicyDir string `json:"policy_dir" yaml:"policy_dir" mapstructure:"policy_dir"`

	// BundleConfig Bundle 加载配置
	BundleConfig *BundleConfig `json:"bundle,omitempty" yaml:"bundle,omitempty" mapstructure:"bundle"`

	// DataDir 数据目录
	DataDir string `json:"data_dir,omitempty" yaml:"data_dir,omitempty" mapstructure:"data_dir"`

	// EnableLogging 启用 OPA 内部日志
	EnableLogging bool `json:"enable_logging,omitempty" yaml:"enable_logging,omitempty" mapstructure:"enable_logging"`

	// EnableDecisionLogs 启用决策日志
	EnableDecisionLogs bool `json:"enable_decision_logs,omitempty" yaml:"enable_decision_logs,omitempty" mapstructure:"enable_decision_logs"`
}

// BundleConfig Bundle 配置
type BundleConfig struct {
	// ServiceURL Bundle 服务地址
	ServiceURL string `json:"service_url" yaml:"service_url" mapstructure:"service_url"`

	// Resource Bundle 资源路径
	Resource string `json:"resource" yaml:"resource" mapstructure:"resource"`

	// PollingMinDelaySeconds 最小轮询间隔（秒）
	PollingMinDelaySeconds int `json:"polling_min_delay_seconds,omitempty" yaml:"polling_min_delay_seconds,omitempty" mapstructure:"polling_min_delay_seconds"`

	// PollingMaxDelaySeconds 最大轮询间隔（秒）
	PollingMaxDelaySeconds int `json:"polling_max_delay_seconds,omitempty" yaml:"polling_max_delay_seconds,omitempty" mapstructure:"polling_max_delay_seconds"`

	// EnableSigning 启用 Bundle 签名验证
	EnableSigning bool `json:"enable_signing,omitempty" yaml:"enable_signing,omitempty" mapstructure:"enable_signing"`

	// SigningKeyID 签名密钥 ID
	SigningKeyID string `json:"signing_key_id,omitempty" yaml:"signing_key_id,omitempty" mapstructure:"signing_key_id"`
}

// RemoteConfig 远程 OPA 配置
type RemoteConfig struct {
	// URL OPA 服务器地址
	URL string `json:"url" yaml:"url" mapstructure:"url"`

	// Timeout 请求超时时间
	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout"`

	// MaxRetries 最大重试次数
	MaxRetries int `json:"max_retries,omitempty" yaml:"max_retries,omitempty" mapstructure:"max_retries"`

	// TLSConfig TLS 配置
	TLSConfig *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty" mapstructure:"tls"`

	// AuthConfig 认证配置
	AuthConfig *AuthConfig `json:"auth,omitempty" yaml:"auth,omitempty" mapstructure:"auth"`
}

// TLSConfig TLS 配置
type TLSConfig struct {
	// Enabled 启用 TLS
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// CertFile 证书文件路径
	CertFile string `json:"cert_file,omitempty" yaml:"cert_file,omitempty" mapstructure:"cert_file"`

	// KeyFile 密钥文件路径
	KeyFile string `json:"key_file,omitempty" yaml:"key_file,omitempty" mapstructure:"key_file"`

	// CAFile CA 证书文件路径
	CAFile string `json:"ca_file,omitempty" yaml:"ca_file,omitempty" mapstructure:"ca_file"`

	// InsecureSkipVerify 跳过证书验证（仅用于测试）
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty" yaml:"insecure_skip_verify,omitempty" mapstructure:"insecure_skip_verify"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	// Type 认证类型 (bearer, basic, api_key)
	Type string `json:"type" yaml:"type" mapstructure:"type"`

	// Token Bearer token
	Token string `json:"token,omitempty" yaml:"token,omitempty" mapstructure:"token"`

	// Username Basic auth 用户名
	Username string `json:"username,omitempty" yaml:"username,omitempty" mapstructure:"username"`

	// Password Basic auth 密码
	Password string `json:"password,omitempty" yaml:"password,omitempty" mapstructure:"password"`

	// APIKey API Key
	APIKey string `json:"api_key,omitempty" yaml:"api_key,omitempty" mapstructure:"api_key"`

	// APIKeyHeader API Key 请求头名称
	APIKeyHeader string `json:"api_key_header,omitempty" yaml:"api_key_header,omitempty" mapstructure:"api_key_header"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	// Enabled 启用缓存
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// MaxSize 最大缓存条目数
	MaxSize int `json:"max_size,omitempty" yaml:"max_size,omitempty" mapstructure:"max_size"`

	// TTL 缓存过期时间
	TTL time.Duration `json:"ttl,omitempty" yaml:"ttl,omitempty" mapstructure:"ttl"`

	// EnableMetrics 启用缓存指标
	EnableMetrics bool `json:"enable_metrics,omitempty" yaml:"enable_metrics,omitempty" mapstructure:"enable_metrics"`
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	if c.Mode == "" {
		return fmt.Errorf("mode is required")
	}

	if c.Mode != ModeEmbedded && c.Mode != ModeRemote {
		return fmt.Errorf("invalid mode: %s (must be 'embedded' or 'remote')", c.Mode)
	}

	if c.Mode == ModeEmbedded {
		if c.EmbeddedConfig == nil {
			return fmt.Errorf("embedded_config is required when mode is 'embedded'")
		}
		if err := c.EmbeddedConfig.Validate(); err != nil {
			return fmt.Errorf("invalid embedded_config: %w", err)
		}
	}

	if c.Mode == ModeRemote {
		if c.RemoteConfig == nil {
			return fmt.Errorf("remote_config is required when mode is 'remote'")
		}
		if err := c.RemoteConfig.Validate(); err != nil {
			return fmt.Errorf("invalid remote_config: %w", err)
		}
	}

	if c.CacheConfig != nil {
		if err := c.CacheConfig.Validate(); err != nil {
			return fmt.Errorf("invalid cache_config: %w", err)
		}
	}

	return nil
}

// Validate 验证嵌入式配置
func (c *EmbeddedConfig) Validate() error {
	// PolicyDir 和 BundleConfig 至少需要一个
	if c.PolicyDir == "" && c.BundleConfig == nil {
		return fmt.Errorf("either policy_dir or bundle_config must be specified")
	}

	if c.BundleConfig != nil {
		if err := c.BundleConfig.Validate(); err != nil {
			return fmt.Errorf("invalid bundle_config: %w", err)
		}
	}

	return nil
}

// Validate 验证 Bundle 配置
func (c *BundleConfig) Validate() error {
	if c.ServiceURL == "" {
		return fmt.Errorf("service_url is required")
	}

	if c.Resource == "" {
		return fmt.Errorf("resource is required")
	}

	if c.PollingMinDelaySeconds < 0 {
		return fmt.Errorf("polling_min_delay_seconds must be >= 0")
	}

	if c.PollingMaxDelaySeconds < 0 {
		return fmt.Errorf("polling_max_delay_seconds must be >= 0")
	}

	if c.PollingMaxDelaySeconds > 0 && c.PollingMinDelaySeconds > c.PollingMaxDelaySeconds {
		return fmt.Errorf("polling_min_delay_seconds must be <= polling_max_delay_seconds")
	}

	if c.EnableSigning && c.SigningKeyID == "" {
		return fmt.Errorf("signing_key_id is required when enable_signing is true")
	}

	return nil
}

// Validate 验证远程配置
func (c *RemoteConfig) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url is required")
	}

	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be >= 0")
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be >= 0")
	}

	if c.TLSConfig != nil {
		if err := c.TLSConfig.Validate(); err != nil {
			return fmt.Errorf("invalid tls_config: %w", err)
		}
	}

	if c.AuthConfig != nil {
		if err := c.AuthConfig.Validate(); err != nil {
			return fmt.Errorf("invalid auth_config: %w", err)
		}
	}

	return nil
}

// Validate 验证 TLS 配置
func (c *TLSConfig) Validate() error {
	if c.Enabled {
		// 如果启用了 TLS，至少需要 CA 文件或跳过验证
		if c.CAFile == "" && !c.InsecureSkipVerify {
			return fmt.Errorf("either ca_file or insecure_skip_verify must be set when TLS is enabled")
		}

		// 如果提供了证书，必须同时提供密钥
		if (c.CertFile != "" && c.KeyFile == "") || (c.CertFile == "" && c.KeyFile != "") {
			return fmt.Errorf("both cert_file and key_file must be provided together")
		}
	}

	return nil
}

// Validate 验证认证配置
func (c *AuthConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("type is required")
	}

	switch c.Type {
	case "bearer":
		if c.Token == "" {
			return fmt.Errorf("token is required for bearer auth")
		}
	case "basic":
		if c.Username == "" || c.Password == "" {
			return fmt.Errorf("username and password are required for basic auth")
		}
	case "api_key":
		if c.APIKey == "" {
			return fmt.Errorf("api_key is required for api_key auth")
		}
		if c.APIKeyHeader == "" {
			return fmt.Errorf("api_key_header is required for api_key auth")
		}
	default:
		return fmt.Errorf("invalid auth type: %s (must be 'bearer', 'basic', or 'api_key')", c.Type)
	}

	return nil
}

// Validate 验证缓存配置
func (c *CacheConfig) Validate() error {
	if c.Enabled {
		if c.MaxSize <= 0 {
			return fmt.Errorf("max_size must be > 0 when cache is enabled")
		}

		if c.TTL < 0 {
			return fmt.Errorf("ttl must be >= 0")
		}
	}

	return nil
}

// SetDefaults 设置默认值
func (c *Config) SetDefaults() {
	if c.Mode == "" {
		c.Mode = ModeEmbedded
	}

	if c.Mode == ModeRemote && c.RemoteConfig != nil {
		c.RemoteConfig.SetDefaults()
	}

	if c.CacheConfig != nil {
		c.CacheConfig.SetDefaults()
	}
}

// SetDefaults 设置远程配置默认值
func (c *RemoteConfig) SetDefaults() {
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}

	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
}

// SetDefaults 设置缓存配置默认值
func (c *CacheConfig) SetDefaults() {
	if c.MaxSize == 0 {
		c.MaxSize = 10000
	}

	if c.TTL == 0 {
		c.TTL = 5 * time.Minute
	}
}
