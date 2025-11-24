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
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/pkg/security/audit"
	"github.com/innovationmech/swit/pkg/security/oauth2"
	"github.com/innovationmech/swit/pkg/security/opa"
	"github.com/innovationmech/swit/pkg/security/scanner"
	"github.com/innovationmech/swit/pkg/security/secrets"
)

// SecurityConfig 定义安全管理器的配置
type SecurityConfig struct {
	// Enabled 是否启用安全功能
	Enabled bool `yaml:"enabled" json:"enabled" mapstructure:"enabled"`

	// OAuth2 OAuth2/OIDC 配置
	OAuth2 *oauth2.Config `yaml:"oauth2,omitempty" json:"oauth2,omitempty" mapstructure:"oauth2"`

	// OPA OPA 策略引擎配置
	OPA *opa.Config `yaml:"opa,omitempty" json:"opa,omitempty" mapstructure:"opa"`

	// Scanner 安全扫描配置
	Scanner *scanner.ScannerConfig `yaml:"scanner,omitempty" json:"scanner,omitempty" mapstructure:"scanner"`

	// Secrets 密钥管理配置
	Secrets *secrets.ManagerConfig `yaml:"secrets,omitempty" json:"secrets,omitempty" mapstructure:"secrets"`

	// Audit 审计日志配置
	Audit *audit.AuditLoggerConfig `yaml:"audit,omitempty" json:"audit,omitempty" mapstructure:"audit"`
}

// Validate 验证安全配置
func (c *SecurityConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	// 验证 OAuth2 配置
	if c.OAuth2 != nil && c.OAuth2.Enabled {
		if err := c.OAuth2.Validate(); err != nil {
			return fmt.Errorf("invalid oauth2 config: %w", err)
		}
	}

	// 验证 OPA 配置
	if c.OPA != nil {
		if err := c.OPA.Validate(); err != nil {
			return fmt.Errorf("invalid opa config: %w", err)
		}
	}

	// 验证 Scanner 配置
	if c.Scanner != nil && c.Scanner.Enabled {
		// Scanner config doesn't have a Validate method, check basic fields
		if len(c.Scanner.Tools) == 0 {
			return fmt.Errorf("scanner enabled but no tools configured")
		}
	}

	// 验证 Secrets 配置
	if c.Secrets != nil {
		if err := c.Secrets.Validate(); err != nil {
			return fmt.Errorf("invalid secrets config: %w", err)
		}
	}

	// 验证 Audit 配置
	if c.Audit != nil {
		if err := c.Audit.Validate(); err != nil {
			return fmt.Errorf("invalid audit config: %w", err)
		}
	}

	return nil
}

// SetDefaults 设置默认值
func (c *SecurityConfig) SetDefaults() {
	// 默认禁用安全功能
	if !c.Enabled {
		c.Enabled = false
	}

	if c.Audit == nil {
		// Use stdout by default to avoid permission issues
		c.Audit = &audit.AuditLoggerConfig{
			Enabled:    false, // Default to disabled
			OutputType: audit.OutputTypeStdout,
		}
	}

	if c.Scanner == nil {
		c.Scanner = scanner.DefaultConfig()
		c.Scanner.Enabled = false // 默认禁用扫描
	}
}

// ApplyEnvironmentOverrides 应用环境变量覆盖
// 环境变量遵循模式: SWIT_SECURITY_*
func (c *SecurityConfig) ApplyEnvironmentOverrides() {
	// 启用/禁用安全功能
	if enabled := os.Getenv("SWIT_SECURITY_ENABLED"); enabled != "" {
		if val, err := strconv.ParseBool(enabled); err == nil {
			c.Enabled = val
		}
	}

	// OAuth2 环境变量覆盖
	c.applyOAuth2Overrides()

	// OPA 环境变量覆盖
	c.applyOPAOverrides()

	// 审计日志环境变量覆盖
	c.applyAuditOverrides()

	// Scanner 环境变量覆盖
	c.applyScannerOverrides()

	// Secrets 环境变量覆盖
	c.applySecretsOverrides()
}

// applyOAuth2Overrides 应用 OAuth2 相关环境变量覆盖
func (c *SecurityConfig) applyOAuth2Overrides() {
	// 检查是否有 OAuth2 相关的环境变量
	hasOAuth2Env := os.Getenv("SWIT_SECURITY_OAUTH2_ENABLED") != "" ||
		os.Getenv("SWIT_SECURITY_OAUTH2_PROVIDER") != "" ||
		os.Getenv("SWIT_SECURITY_OAUTH2_ISSUER_URL") != "" ||
		os.Getenv("SWIT_SECURITY_OAUTH2_CLIENT_ID") != "" ||
		os.Getenv("SWIT_SECURITY_OAUTH2_CLIENT_SECRET") != ""

	// 如果配置为 nil 但有环境变量，创建默认配置
	if c.OAuth2 == nil && hasOAuth2Env {
		c.OAuth2 = &oauth2.Config{}
	}

	if c.OAuth2 == nil {
		return
	}

	if enabled := os.Getenv("SWIT_SECURITY_OAUTH2_ENABLED"); enabled != "" {
		if val, err := strconv.ParseBool(enabled); err == nil {
			c.OAuth2.Enabled = val
		}
	}

	if provider := os.Getenv("SWIT_SECURITY_OAUTH2_PROVIDER"); provider != "" {
		c.OAuth2.Provider = provider
	}

	if issuerURL := os.Getenv("SWIT_SECURITY_OAUTH2_ISSUER_URL"); issuerURL != "" {
		c.OAuth2.IssuerURL = issuerURL
	}

	if clientID := os.Getenv("SWIT_SECURITY_OAUTH2_CLIENT_ID"); clientID != "" {
		c.OAuth2.ClientID = clientID
	}

	if clientSecret := os.Getenv("SWIT_SECURITY_OAUTH2_CLIENT_SECRET"); clientSecret != "" {
		c.OAuth2.ClientSecret = clientSecret
	}
}

// applyOPAOverrides 应用 OPA 相关环境变量覆盖
func (c *SecurityConfig) applyOPAOverrides() {
	// 检查是否有 OPA 相关的环境变量
	hasOPAEnv := os.Getenv("SWIT_SECURITY_OPA_MODE") != "" ||
		os.Getenv("SWIT_SECURITY_OPA_POLICY_DIR") != "" ||
		os.Getenv("SWIT_SECURITY_OPA_SERVER_URL") != ""

	// 如果配置为 nil 但有环境变量，创建默认配置
	if c.OPA == nil && hasOPAEnv {
		c.OPA = &opa.Config{}
	}

	if c.OPA == nil {
		return
	}

	if mode := os.Getenv("SWIT_SECURITY_OPA_MODE"); mode != "" {
		c.OPA.Mode = opa.Mode(mode)
	}

	if policyDir := os.Getenv("SWIT_SECURITY_OPA_POLICY_DIR"); policyDir != "" {
		if c.OPA.EmbeddedConfig == nil {
			c.OPA.EmbeddedConfig = &opa.EmbeddedConfig{}
		}
		c.OPA.EmbeddedConfig.PolicyDir = policyDir
	}

	if serverURL := os.Getenv("SWIT_SECURITY_OPA_SERVER_URL"); serverURL != "" {
		if c.OPA.RemoteConfig == nil {
			c.OPA.RemoteConfig = &opa.RemoteConfig{}
		}
		c.OPA.RemoteConfig.URL = serverURL
	}
}

// applyAuditOverrides 应用审计日志相关环境变量覆盖
func (c *SecurityConfig) applyAuditOverrides() {
	if c.Audit == nil {
		return
	}

	if enabled := os.Getenv("SWIT_SECURITY_AUDIT_ENABLED"); enabled != "" {
		if val, err := strconv.ParseBool(enabled); err == nil {
			c.Audit.Enabled = val
		}
	}

	if logPath := os.Getenv("SWIT_SECURITY_AUDIT_LOG_PATH"); logPath != "" {
		c.Audit.OutputType = audit.OutputTypeFile
		c.Audit.FilePath = logPath
	}

	if outputType := os.Getenv("SWIT_SECURITY_AUDIT_OUTPUT_TYPE"); outputType != "" {
		c.Audit.OutputType = audit.OutputType(outputType)
	}
}

// applyScannerOverrides 应用安全扫描器相关环境变量覆盖
func (c *SecurityConfig) applyScannerOverrides() {
	if c.Scanner == nil {
		return
	}

	if enabled := os.Getenv("SWIT_SECURITY_SCANNER_ENABLED"); enabled != "" {
		if val, err := strconv.ParseBool(enabled); err == nil {
			c.Scanner.Enabled = val
		}
	}
}

// applySecretsOverrides 应用密钥管理相关环境变量覆盖
func (c *SecurityConfig) applySecretsOverrides() {
	if c.Secrets == nil {
		return
	}

	// Secrets 管理使用 Providers 数组，环境变量覆盖比较复杂
	// 这里仅作为占位符，实际环境变量覆盖需要根据具体需求实现
	// 例如：SWIT_SECURITY_SECRETS_PROVIDER_TYPE, SWIT_SECURITY_SECRETS_VAULT_ADDRESS 等
}

// SecurityManager 统一的安全管理器
// 集成 OAuth2、OPA、安全扫描、密钥管理和审计日志
type SecurityManager struct {
	config *SecurityConfig

	// OAuth2 客户端
	oauth2Client *oauth2.Client

	// OPA 客户端
	opaClient opa.Client

	// 安全扫描器
	securityScanner *scanner.SecurityScanner

	// 密钥管理器
	secretsManager *secrets.Manager

	// 审计日志记录器
	auditLogger *audit.AuditLogger

	// 生命周期管理
	mu          sync.RWMutex
	initialized bool
	closed      bool
}

// NewSecurityManager 创建新的安全管理器
func NewSecurityManager(config *SecurityConfig) (*SecurityManager, error) {
	if config == nil {
		return nil, fmt.Errorf("security config is required")
	}

	// 设置默认值
	config.SetDefaults()

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid security config: %w", err)
	}

	sm := &SecurityManager{
		config: config,
	}

	return sm, nil
}

// InitializeSecurity 初始化所有安全组件
func (sm *SecurityManager) InitializeSecurity(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.initialized {
		return fmt.Errorf("security manager already initialized")
	}

	if sm.closed {
		return fmt.Errorf("security manager is closed")
	}

	if !sm.config.Enabled {
		sm.initialized = true
		return nil
	}

	var err error

	// 初始化 OAuth2 客户端
	if sm.config.OAuth2 != nil && sm.config.OAuth2.Enabled {
		sm.oauth2Client, err = oauth2.NewClient(ctx, sm.config.OAuth2)
		if err != nil {
			return fmt.Errorf("failed to initialize oauth2 client: %w", err)
		}
	}

	// 初始化 OPA 客户端
	if sm.config.OPA != nil {
		sm.opaClient, err = opa.NewClient(ctx, sm.config.OPA)
		if err != nil {
			return fmt.Errorf("failed to initialize opa client: %w", err)
		}
	}

	// 初始化安全扫描器
	if sm.config.Scanner != nil && sm.config.Scanner.Enabled {
		sm.securityScanner, err = scanner.NewSecurityScanner(sm.config.Scanner)
		if err != nil {
			return fmt.Errorf("failed to initialize security scanner: %w", err)
		}
	}

	// 初始化密钥管理器
	if sm.config.Secrets != nil {
		sm.secretsManager, err = secrets.NewManager(sm.config.Secrets)
		if err != nil {
			return fmt.Errorf("failed to initialize secrets manager: %w", err)
		}
	}

	// 初始化审计日志记录器
	if sm.config.Audit != nil {
		sm.auditLogger, err = audit.NewAuditLogger(sm.config.Audit)
		if err != nil {
			return fmt.Errorf("failed to initialize audit logger: %w", err)
		}
	}

	sm.initialized = true
	return nil
}

// RegisterSecurityMiddleware 注册安全中间件到 HTTP 和 gRPC 服务器
func (sm *SecurityManager) RegisterSecurityMiddleware(router *gin.Engine, grpcServer *grpc.Server) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.initialized {
		return fmt.Errorf("security manager not initialized")
	}

	if sm.closed {
		return fmt.Errorf("security manager is closed")
	}

	// 注册 HTTP 中间件
	if router != nil {
		// OAuth2 中间件
		if sm.oauth2Client != nil {
			// 这里可以添加 OAuth2 验证中间件
			// 例如：router.Use(oauth2Middleware(sm.oauth2Client))
		}

		// OPA 中间件
		if sm.opaClient != nil {
			// 这里可以添加 OPA 授权中间件
			// 例如：router.Use(opaMiddleware(sm.opaClient))
		}

		// 审计日志中间件
		if sm.auditLogger != nil {
			// 这里可以添加审计日志中间件
			// 例如：router.Use(auditMiddleware(sm.auditLogger))
		}
	}

	// 注册 gRPC 拦截器
	if grpcServer != nil {
		// gRPC 中间件需要在服务器创建时通过 ServerOption 注册
		// 这里只是一个占位符，实际实现需要在服务器配置阶段完成
	}

	return nil
}

// GetOAuth2Client 获取 OAuth2 客户端
func (sm *SecurityManager) GetOAuth2Client() *oauth2.Client {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.oauth2Client
}

// GetOPAClient 获取 OPA 客户端
func (sm *SecurityManager) GetOPAClient() opa.Client {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.opaClient
}

// GetSecurityScanner 获取安全扫描器
func (sm *SecurityManager) GetSecurityScanner() *scanner.SecurityScanner {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.securityScanner
}

// GetSecretsManager 获取密钥管理器
func (sm *SecurityManager) GetSecretsManager() *secrets.Manager {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.secretsManager
}

// GetAuditLogger 获取审计日志记录器
func (sm *SecurityManager) GetAuditLogger() *audit.AuditLogger {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.auditLogger
}

// IsEnabled 检查安全功能是否启用
func (sm *SecurityManager) IsEnabled() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.config.Enabled
}

// IsInitialized 检查是否已初始化
func (sm *SecurityManager) IsInitialized() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.initialized
}

// GetHTTPSecurityMiddleware 返回配置好的 HTTP 安全中间件
func (sm *SecurityManager) GetHTTPSecurityMiddleware() []interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.initialized || sm.closed || !sm.config.Enabled {
		return nil
	}

	var middlewares []interface{}

	// 添加 OAuth2 中间件（如果配置了）
	// Note: 实际项目中，这里应该根据配置创建中间件
	// 现在只是返回空列表，由具体使用方配置

	// 添加 OPA 中间件（如果配置了）
	// Note: 实际项目中，这里应该根据配置创建中间件

	return middlewares
}

// GetGRPCUnaryInterceptors 返回配置好的 gRPC unary 拦截器
func (sm *SecurityManager) GetGRPCUnaryInterceptors() []interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.initialized || sm.closed || !sm.config.Enabled {
		return nil
	}

	var interceptors []interface{}

	// 添加 OAuth2 拦截器（如果配置了）
	// Note: 实际项目中，这里应该根据配置创建拦截器

	// 添加 OPA 拦截器（如果配置了）
	// Note: 实际项目中，这里应该根据配置创建拦截器

	return interceptors
}

// GetGRPCStreamInterceptors 返回配置好的 gRPC stream 拦截器
func (sm *SecurityManager) GetGRPCStreamInterceptors() []interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.initialized || sm.closed || !sm.config.Enabled {
		return nil
	}

	var interceptors []interface{}

	// 添加 OAuth2 拦截器（如果配置了）
	// Note: 实际项目中，这里应该根据配置创建拦截器

	// 添加 OPA 拦截器（如果配置了）
	// Note: 实际项目中，这里应该根据配置创建拦截器

	return interceptors
}

// Shutdown 关闭所有安全组件
func (sm *SecurityManager) Shutdown(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.closed {
		return nil
	}

	var errs []error

	// 关闭 OPA 客户端
	if sm.opaClient != nil {
		if err := sm.opaClient.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to close opa client: %w", err))
		}
	}

	// 关闭密钥管理器
	if sm.secretsManager != nil {
		if err := sm.secretsManager.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close secrets manager: %w", err))
		}
	}

	// 关闭审计日志记录器
	if sm.auditLogger != nil {
		if err := sm.auditLogger.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close audit logger: %w", err))
		}
	}

	sm.closed = true

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}

	return nil
}
