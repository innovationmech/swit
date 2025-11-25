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
	"fmt"
	"os"
	"strconv"
	"time"
)

// Mode 定义 OPA 客户端的运行模式
type Mode string

const (
	// ModeEmbedded 嵌入式模式 - OPA 引擎运行在同一进程中
	ModeEmbedded Mode = "embedded"
	// ModeRemote 远程模式 - 连接到外部 OPA 服务器
	// 注意：Sidecar 模式也使用 ModeRemote，通过连接到本地 sidecar 容器（如 localhost:8181）
	ModeRemote Mode = "remote"
	// ModeSidecar Sidecar 模式 - 连接到本地 OPA sidecar 容器（ModeRemote 的别名）
	ModeSidecar Mode = "sidecar"
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
	// URLs OPA 服务器地址列表（支持多服务器负载均衡）
	// 单服务器时只需配置一个 URL
	URLs []string `json:"urls" yaml:"urls" mapstructure:"urls"`

	// URL 单服务器地址（兼容旧配置，与 URLs 二选一）
	URL string `json:"url,omitempty" yaml:"url,omitempty" mapstructure:"url"`

	// Timeout 请求超时时间
	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout"`

	// MaxRetries 最大重试次数
	MaxRetries int `json:"max_retries,omitempty" yaml:"max_retries,omitempty" mapstructure:"max_retries"`

	// RetryBackoffMultiplier 重试退避倍数（默认 2.0）
	RetryBackoffMultiplier float64 `json:"retry_backoff_multiplier,omitempty" yaml:"retry_backoff_multiplier,omitempty" mapstructure:"retry_backoff_multiplier"`

	// RetryMaxWait 最大重试等待时间
	RetryMaxWait time.Duration `json:"retry_max_wait,omitempty" yaml:"retry_max_wait,omitempty" mapstructure:"retry_max_wait"`

	// PoolConfig 连接池配置
	PoolConfig *PoolConfig `json:"pool,omitempty" yaml:"pool,omitempty" mapstructure:"pool"`

	// LoadBalancing 负载均衡策略 (round_robin, random, least_connections)
	LoadBalancing string `json:"load_balancing,omitempty" yaml:"load_balancing,omitempty" mapstructure:"load_balancing"`

	// HealthCheck 健康检查配置
	HealthCheck *HealthCheckConfig `json:"health_check,omitempty" yaml:"health_check,omitempty" mapstructure:"health_check"`

	// TLSConfig TLS 配置
	TLSConfig *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty" mapstructure:"tls"`

	// AuthConfig 认证配置
	AuthConfig *AuthConfig `json:"auth,omitempty" yaml:"auth,omitempty" mapstructure:"auth"`
}

// PoolConfig 连接池配置
type PoolConfig struct {
	// MaxIdleConns 最大空闲连接数
	MaxIdleConns int `json:"max_idle_conns,omitempty" yaml:"max_idle_conns,omitempty" mapstructure:"max_idle_conns"`

	// MaxConnsPerHost 每个主机的最大连接数
	MaxConnsPerHost int `json:"max_conns_per_host,omitempty" yaml:"max_conns_per_host,omitempty" mapstructure:"max_conns_per_host"`

	// IdleConnTimeout 空闲连接超时时间
	IdleConnTimeout time.Duration `json:"idle_conn_timeout,omitempty" yaml:"idle_conn_timeout,omitempty" mapstructure:"idle_conn_timeout"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	// Enabled 启用健康检查
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// Interval 健康检查间隔
	Interval time.Duration `json:"interval,omitempty" yaml:"interval,omitempty" mapstructure:"interval"`

	// Timeout 健康检查超时时间
	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout"`

	// FailureThreshold 故障阈值（连续失败多少次标记为不健康）
	FailureThreshold int `json:"failure_threshold,omitempty" yaml:"failure_threshold,omitempty" mapstructure:"failure_threshold"`

	// SuccessThreshold 成功阈值（连续成功多少次标记为健康）
	SuccessThreshold int `json:"success_threshold,omitempty" yaml:"success_threshold,omitempty" mapstructure:"success_threshold"`
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

	// 规范化 mode：sidecar 视为 remote
	if c.Mode == ModeSidecar {
		c.Mode = ModeRemote
	}

	if c.Mode != ModeEmbedded && c.Mode != ModeRemote {
		return fmt.Errorf("invalid mode: %s (must be 'embedded', 'remote', or 'sidecar')", c.Mode)
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
			return fmt.Errorf("remote_config is required when mode is 'remote' or 'sidecar'")
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
	// 兼容性处理：如果只配置了 URL，转换为 URLs
	if c.URL != "" && len(c.URLs) == 0 {
		c.URLs = []string{c.URL}
	}

	if len(c.URLs) == 0 {
		return fmt.Errorf("at least one url is required (urls or url)")
	}

	// 验证所有 URL 不为空
	for i, url := range c.URLs {
		if url == "" {
			return fmt.Errorf("url at index %d is empty", i)
		}
	}

	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be >= 0")
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be >= 0")
	}

	if c.RetryBackoffMultiplier < 0 {
		return fmt.Errorf("retry_backoff_multiplier must be >= 0")
	}

	if c.RetryMaxWait < 0 {
		return fmt.Errorf("retry_max_wait must be >= 0")
	}

	if c.LoadBalancing != "" {
		validStrategies := map[string]bool{
			"round_robin":       true,
			"random":            true,
			"least_connections": true,
		}
		if !validStrategies[c.LoadBalancing] {
			return fmt.Errorf("invalid load_balancing strategy: %s (must be 'round_robin', 'random', or 'least_connections')", c.LoadBalancing)
		}
	}

	if c.PoolConfig != nil {
		if err := c.PoolConfig.Validate(); err != nil {
			return fmt.Errorf("invalid pool_config: %w", err)
		}
	}

	if c.HealthCheck != nil {
		if err := c.HealthCheck.Validate(); err != nil {
			return fmt.Errorf("invalid health_check: %w", err)
		}
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

// Validate 验证连接池配置
func (c *PoolConfig) Validate() error {
	if c.MaxIdleConns < 0 {
		return fmt.Errorf("max_idle_conns must be >= 0")
	}

	if c.MaxConnsPerHost < 0 {
		return fmt.Errorf("max_conns_per_host must be >= 0")
	}

	if c.IdleConnTimeout < 0 {
		return fmt.Errorf("idle_conn_timeout must be >= 0")
	}

	return nil
}

// Validate 验证健康检查配置
func (c *HealthCheckConfig) Validate() error {
	if c.Enabled {
		if c.Interval <= 0 {
			return fmt.Errorf("interval must be > 0 when health check is enabled")
		}

		if c.Timeout <= 0 {
			return fmt.Errorf("timeout must be > 0 when health check is enabled")
		}

		if c.FailureThreshold <= 0 {
			return fmt.Errorf("failure_threshold must be > 0")
		}

		if c.SuccessThreshold <= 0 {
			return fmt.Errorf("success_threshold must be > 0")
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
	// 兼容性处理：如果只配置了 URL，转换为 URLs
	if c.URL != "" && len(c.URLs) == 0 {
		c.URLs = []string{c.URL}
	}

	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}

	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}

	if c.RetryBackoffMultiplier == 0 {
		c.RetryBackoffMultiplier = 2.0
	}

	if c.RetryMaxWait == 0 {
		c.RetryMaxWait = 30 * time.Second
	}

	if c.LoadBalancing == "" && len(c.URLs) > 1 {
		c.LoadBalancing = "round_robin"
	}

	if c.PoolConfig != nil {
		c.PoolConfig.SetDefaults()
	}

	if c.HealthCheck != nil {
		c.HealthCheck.SetDefaults()
	}
}

// SetDefaults 设置连接池配置默认值
func (c *PoolConfig) SetDefaults() {
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 100
	}

	if c.MaxConnsPerHost == 0 {
		c.MaxConnsPerHost = 100
	}

	if c.IdleConnTimeout == 0 {
		c.IdleConnTimeout = 90 * time.Second
	}
}

// SetDefaults 设置健康检查配置默认值
func (c *HealthCheckConfig) SetDefaults() {
	if c.Interval == 0 {
		c.Interval = 30 * time.Second
	}

	if c.Timeout == 0 {
		c.Timeout = 5 * time.Second
	}

	if c.FailureThreshold == 0 {
		c.FailureThreshold = 3
	}

	if c.SuccessThreshold == 0 {
		c.SuccessThreshold = 1
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

// LoadFromEnv 从环境变量加载配置并覆盖现有值
// 环境变量命名格式：OPA_<字段名> 或 OPA_<嵌套字段路径>
// 例如：
//   - OPA_MODE=remote
//   - OPA_REMOTE_URL=http://opa-server:8181
//   - OPA_CACHE_ENABLED=true
//   - OPA_CACHE_MAX_SIZE=5000
//
// 注意：此函数提供手动环境变量覆盖。如果使用 pkg/config.Manager，
// 环境变量会通过 Viper 和 mapstructure 标签自动处理。
func (c *Config) LoadFromEnv() {
	// Mode
	if mode := os.Getenv("OPA_MODE"); mode != "" {
		c.Mode = Mode(mode)
	}

	// DefaultDecisionPath
	if path := os.Getenv("OPA_DEFAULT_DECISION_PATH"); path != "" {
		c.DefaultDecisionPath = path
	}

	// EmbeddedConfig
	if c.Mode == ModeEmbedded || c.Mode == "" {
		if c.EmbeddedConfig == nil {
			c.EmbeddedConfig = &EmbeddedConfig{}
		}
		if policyDir := os.Getenv("OPA_EMBEDDED_POLICY_DIR"); policyDir != "" {
			c.EmbeddedConfig.PolicyDir = policyDir
		}
		if dataDir := os.Getenv("OPA_EMBEDDED_DATA_DIR"); dataDir != "" {
			c.EmbeddedConfig.DataDir = dataDir
		}
		if enableLogging := os.Getenv("OPA_EMBEDDED_ENABLE_LOGGING"); enableLogging != "" {
			c.EmbeddedConfig.EnableLogging = parseBool(enableLogging)
		}
		if enableDecisionLogs := os.Getenv("OPA_EMBEDDED_ENABLE_DECISION_LOGS"); enableDecisionLogs != "" {
			c.EmbeddedConfig.EnableDecisionLogs = parseBool(enableDecisionLogs)
		}

		// BundleConfig
		if serviceURL := os.Getenv("OPA_BUNDLE_SERVICE_URL"); serviceURL != "" {
			if c.EmbeddedConfig.BundleConfig == nil {
				c.EmbeddedConfig.BundleConfig = &BundleConfig{}
			}
			c.EmbeddedConfig.BundleConfig.ServiceURL = serviceURL
		}
		if resource := os.Getenv("OPA_BUNDLE_RESOURCE"); resource != "" {
			if c.EmbeddedConfig.BundleConfig == nil {
				c.EmbeddedConfig.BundleConfig = &BundleConfig{}
			}
			c.EmbeddedConfig.BundleConfig.Resource = resource
		}
	}

	// RemoteConfig
	if c.Mode == ModeRemote || c.Mode == ModeSidecar {
		if c.RemoteConfig == nil {
			c.RemoteConfig = &RemoteConfig{}
		}
		if url := os.Getenv("OPA_REMOTE_URL"); url != "" {
			c.RemoteConfig.URL = url
		}
		if timeout := os.Getenv("OPA_REMOTE_TIMEOUT"); timeout != "" {
			if d, err := time.ParseDuration(timeout); err == nil {
				c.RemoteConfig.Timeout = d
			}
		}
		if maxRetries := os.Getenv("OPA_REMOTE_MAX_RETRIES"); maxRetries != "" {
			if n, err := strconv.Atoi(maxRetries); err == nil {
				c.RemoteConfig.MaxRetries = n
			}
		}

		// TLS Config
		if tlsEnabled := os.Getenv("OPA_REMOTE_TLS_ENABLED"); tlsEnabled != "" {
			if c.RemoteConfig.TLSConfig == nil {
				c.RemoteConfig.TLSConfig = &TLSConfig{}
			}
			c.RemoteConfig.TLSConfig.Enabled = parseBool(tlsEnabled)
		}
		if certFile := os.Getenv("OPA_REMOTE_TLS_CERT_FILE"); certFile != "" {
			if c.RemoteConfig.TLSConfig == nil {
				c.RemoteConfig.TLSConfig = &TLSConfig{}
			}
			c.RemoteConfig.TLSConfig.CertFile = certFile
		}
		if keyFile := os.Getenv("OPA_REMOTE_TLS_KEY_FILE"); keyFile != "" {
			if c.RemoteConfig.TLSConfig == nil {
				c.RemoteConfig.TLSConfig = &TLSConfig{}
			}
			c.RemoteConfig.TLSConfig.KeyFile = keyFile
		}
		if caFile := os.Getenv("OPA_REMOTE_TLS_CA_FILE"); caFile != "" {
			if c.RemoteConfig.TLSConfig == nil {
				c.RemoteConfig.TLSConfig = &TLSConfig{}
			}
			c.RemoteConfig.TLSConfig.CAFile = caFile
		}

		// Auth Config
		if authType := os.Getenv("OPA_REMOTE_AUTH_TYPE"); authType != "" {
			if c.RemoteConfig.AuthConfig == nil {
				c.RemoteConfig.AuthConfig = &AuthConfig{}
			}
			c.RemoteConfig.AuthConfig.Type = authType
		}
		if token := os.Getenv("OPA_REMOTE_AUTH_TOKEN"); token != "" {
			if c.RemoteConfig.AuthConfig == nil {
				c.RemoteConfig.AuthConfig = &AuthConfig{}
			}
			c.RemoteConfig.AuthConfig.Token = token
		}
		if username := os.Getenv("OPA_REMOTE_AUTH_USERNAME"); username != "" {
			if c.RemoteConfig.AuthConfig == nil {
				c.RemoteConfig.AuthConfig = &AuthConfig{}
			}
			c.RemoteConfig.AuthConfig.Username = username
		}
		if password := os.Getenv("OPA_REMOTE_AUTH_PASSWORD"); password != "" {
			if c.RemoteConfig.AuthConfig == nil {
				c.RemoteConfig.AuthConfig = &AuthConfig{}
			}
			c.RemoteConfig.AuthConfig.Password = password
		}
		if apiKey := os.Getenv("OPA_REMOTE_AUTH_API_KEY"); apiKey != "" {
			if c.RemoteConfig.AuthConfig == nil {
				c.RemoteConfig.AuthConfig = &AuthConfig{}
			}
			c.RemoteConfig.AuthConfig.APIKey = apiKey
		}
		if apiKeyHeader := os.Getenv("OPA_REMOTE_AUTH_API_KEY_HEADER"); apiKeyHeader != "" {
			if c.RemoteConfig.AuthConfig == nil {
				c.RemoteConfig.AuthConfig = &AuthConfig{}
			}
			c.RemoteConfig.AuthConfig.APIKeyHeader = apiKeyHeader
		}
	}

	// CacheConfig
	if cacheEnabled := os.Getenv("OPA_CACHE_ENABLED"); cacheEnabled != "" {
		if c.CacheConfig == nil {
			c.CacheConfig = &CacheConfig{}
		}
		c.CacheConfig.Enabled = parseBool(cacheEnabled)
	}
	if maxSize := os.Getenv("OPA_CACHE_MAX_SIZE"); maxSize != "" {
		if c.CacheConfig == nil {
			c.CacheConfig = &CacheConfig{}
		}
		if n, err := strconv.Atoi(maxSize); err == nil {
			c.CacheConfig.MaxSize = n
		}
	}
	if ttl := os.Getenv("OPA_CACHE_TTL"); ttl != "" {
		if c.CacheConfig == nil {
			c.CacheConfig = &CacheConfig{}
		}
		if d, err := time.ParseDuration(ttl); err == nil {
			c.CacheConfig.TTL = d
		}
	}
	if enableMetrics := os.Getenv("OPA_CACHE_ENABLE_METRICS"); enableMetrics != "" {
		if c.CacheConfig == nil {
			c.CacheConfig = &CacheConfig{}
		}
		c.CacheConfig.EnableMetrics = parseBool(enableMetrics)
	}
}

// parseBool 解析布尔值环境变量
func parseBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}
