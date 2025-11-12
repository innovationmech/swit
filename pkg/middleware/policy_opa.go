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

package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/security/opa"
	"go.uber.org/zap"
)

// PolicyMiddlewareConfig OPA 策略中间件配置
type PolicyMiddlewareConfig struct {
	// Client OPA 客户端
	Client opa.Client

	// DecisionPath 决策路径（例如 "authz/allow"）
	DecisionPath string

	// WhiteList 白名单路径，这些路径不需要策略评估
	WhiteList []string

	// InputBuilder 自定义输入构建函数
	InputBuilder func(*gin.Context) (*opa.PolicyInput, error)

	// ErrorHandler 自定义错误处理函数
	ErrorHandler func(*gin.Context, error)

	// DeniedHandler 自定义拒绝处理函数
	DeniedHandler func(*gin.Context, *opa.Result)

	// Logger 日志记录器
	Logger *zap.Logger

	// EnableAuditLog 启用审计日志
	EnableAuditLog bool

	// AuditLogHandler 自定义审计日志处理函数
	AuditLogHandler func(*AuditLog)

	// Timeout 评估超时时间
	Timeout time.Duration
}

// AuditLog 审计日志
type AuditLog struct {
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`

	// DecisionID 决策 ID
	DecisionID string `json:"decision_id,omitempty"`

	// Allowed 是否允许
	Allowed bool `json:"allowed"`

	// DecisionPath 决策路径
	DecisionPath string `json:"decision_path"`

	// Input 策略输入
	Input *opa.PolicyInput `json:"input,omitempty"`

	// Result 评估结果
	Result *opa.Result `json:"result,omitempty"`

	// Error 错误信息
	Error string `json:"error,omitempty"`

	// Duration 评估耗时（毫秒）
	Duration int64 `json:"duration_ms"`

	// Request 请求信息
	Request struct {
		Method    string `json:"method"`
		Path      string `json:"path"`
		ClientIP  string `json:"client_ip"`
		UserAgent string `json:"user_agent"`
	} `json:"request"`
}

// Option 中间件选项函数
type Option func(*PolicyMiddlewareConfig)

// WithDecisionPath 设置决策路径
func WithDecisionPath(path string) Option {
	return func(c *PolicyMiddlewareConfig) {
		c.DecisionPath = path
	}
}

// WithWhiteList 设置白名单
func WithWhiteList(whiteList []string) Option {
	return func(c *PolicyMiddlewareConfig) {
		c.WhiteList = whiteList
	}
}

// WithInputBuilder 设置自定义输入构建函数
func WithInputBuilder(builder func(*gin.Context) (*opa.PolicyInput, error)) Option {
	return func(c *PolicyMiddlewareConfig) {
		c.InputBuilder = builder
	}
}

// WithErrorHandler 设置自定义错误处理函数
func WithErrorHandler(handler func(*gin.Context, error)) Option {
	return func(c *PolicyMiddlewareConfig) {
		c.ErrorHandler = handler
	}
}

// WithDeniedHandler 设置自定义拒绝处理函数
func WithDeniedHandler(handler func(*gin.Context, *opa.Result)) Option {
	return func(c *PolicyMiddlewareConfig) {
		c.DeniedHandler = handler
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger *zap.Logger) Option {
	return func(c *PolicyMiddlewareConfig) {
		c.Logger = logger
	}
}

// WithAuditLog 启用审计日志
func WithAuditLog(handler func(*AuditLog)) Option {
	return func(c *PolicyMiddlewareConfig) {
		c.EnableAuditLog = true
		c.AuditLogHandler = handler
	}
}

// WithTimeout 设置评估超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(c *PolicyMiddlewareConfig) {
		c.Timeout = timeout
	}
}

// OPAMiddleware 创建 OPA 策略中间件
// 对每个请求进行策略评估，决定是否允许访问
func OPAMiddleware(client opa.Client, options ...Option) gin.HandlerFunc {
	config := &PolicyMiddlewareConfig{
		Client:       client,
		DecisionPath: "authz/allow",
		WhiteList:    []string{},
		Timeout:      5 * time.Second,
	}

	// 应用选项
	for _, opt := range options {
		opt(config)
	}

	// 设置默认的输入构建函数
	if config.InputBuilder == nil {
		config.InputBuilder = defaultInputBuilder
	}

	// 设置默认的错误处理函数
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultPolicyErrorHandler
	}

	// 设置默认的拒绝处理函数
	if config.DeniedHandler == nil {
		config.DeniedHandler = defaultPolicyDeniedHandler
	}

	// 设置默认的审计日志处理函数
	if config.EnableAuditLog && config.AuditLogHandler == nil {
		config.AuditLogHandler = defaultAuditLogHandler(config.Logger)
	}

	return func(c *gin.Context) {
		startTime := time.Now()

		// 检查白名单
		if isInWhiteList(c.Request.URL.Path, config.WhiteList) {
			c.Next()
			return
		}

		// 构建策略输入
		input, err := config.InputBuilder(c)
		if err != nil {
			if config.Logger != nil {
				config.Logger.Error("Failed to build policy input",
					zap.Error(err),
					zap.String("path", c.Request.URL.Path))
			}
			config.ErrorHandler(c, err)
			return
		}

		// 创建评估上下文
		ctx := c.Request.Context()
		if config.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, config.Timeout)
			defer cancel()
		}

		// 评估策略
		result, err := client.Evaluate(ctx, config.DecisionPath, input)
		duration := time.Since(startTime)

		// 记录审计日志
		if config.EnableAuditLog {
			auditLog := buildAuditLog(c, config.DecisionPath, input, result, err, duration)
			config.AuditLogHandler(auditLog)
		}

		// 处理评估错误
		if err != nil {
			if config.Logger != nil {
				config.Logger.Error("Policy evaluation failed",
					zap.Error(err),
					zap.String("path", c.Request.URL.Path),
					zap.String("decision_path", config.DecisionPath),
					zap.Duration("duration", duration))
			}
			config.ErrorHandler(c, err)
			return
		}

		// 检查决策结果
		if !result.Allowed {
			if config.Logger != nil {
				config.Logger.Warn("Policy evaluation denied",
					zap.String("path", c.Request.URL.Path),
					zap.String("decision_path", config.DecisionPath),
					zap.String("user_id", input.User.ID),
					zap.Duration("duration", duration))
			}
			config.DeniedHandler(c, result)
			return
		}

		// 允许访问，记录日志
		if config.Logger != nil {
			config.Logger.Debug("Policy evaluation allowed",
				zap.String("path", c.Request.URL.Path),
				zap.String("decision_path", config.DecisionPath),
				zap.String("user_id", input.User.ID),
				zap.Duration("duration", duration))
		}

		c.Next()
	}
}

// ResourcePolicy 创建资源级策略中间件
// 对特定资源进行策略评估
func ResourcePolicy(resourceType string, getResourceID func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取资源 ID
		resourceID := getResourceID(c)
		if resourceID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "resource ID is required",
			})
			return
		}

		// 将资源信息存储到上下文中，供 OPAMiddleware 使用
		c.Set("resource_type", resourceType)
		c.Set("resource_id", resourceID)

		c.Next()
	}
}

// DynamicPolicyPath 创建动态策略路径中间件
// 根据请求动态确定策略路径
func DynamicPolicyPath(getPath func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取策略路径
		path := getPath(c)
		if path == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "failed to determine policy path",
			})
			return
		}

		// 将策略路径存储到上下文中
		c.Set("policy_path", path)

		c.Next()
	}
}

// defaultInputBuilder 默认的输入构建函数
func defaultInputBuilder(c *gin.Context) (*opa.PolicyInput, error) {
	// 使用 PolicyInputBuilder 构建输入
	builder := opa.NewPolicyInputBuilder().FromHTTPRequest(c)

	// 提取用户信息
	user := opa.ExtractUserFromContext(c)
	builder.WithUser(user)

	// 提取资源信息（如果有）
	if resourceType, exists := c.Get("resource_type"); exists {
		if rt, ok := resourceType.(string); ok {
			builder.WithResourceType(rt)
		}
	}

	if resourceID, exists := c.Get("resource_id"); exists {
		if rid, ok := resourceID.(string); ok {
			builder.WithResourceID(rid)
		}
	}

	return builder.Build(), nil
}

// defaultPolicyErrorHandler 默认的错误处理函数
func defaultPolicyErrorHandler(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"error": "policy evaluation failed",
	})
}

// defaultPolicyDeniedHandler 默认的拒绝处理函数
func defaultPolicyDeniedHandler(c *gin.Context, result *opa.Result) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error":       "access denied",
		"decision_id": result.DecisionID,
	})
}

// defaultAuditLogHandler 默认的审计日志处理函数
func defaultAuditLogHandler(logger *zap.Logger) func(*AuditLog) {
	return func(log *AuditLog) {
		if logger == nil {
			return
		}

		// 将审计日志序列化为 JSON
		logData, err := json.Marshal(log)
		if err != nil {
			logger.Error("Failed to marshal audit log", zap.Error(err))
			return
		}

		// 记录审计日志
		if log.Allowed {
			logger.Info("Policy audit log",
				zap.String("decision_id", log.DecisionID),
				zap.Bool("allowed", log.Allowed),
				zap.Int64("duration_ms", log.Duration),
				zap.ByteString("audit_data", logData))
		} else {
			logger.Warn("Policy audit log",
				zap.String("decision_id", log.DecisionID),
				zap.Bool("allowed", log.Allowed),
				zap.String("error", log.Error),
				zap.Int64("duration_ms", log.Duration),
				zap.ByteString("audit_data", logData))
		}
	}
}

// buildAuditLog 构建审计日志
func buildAuditLog(c *gin.Context, decisionPath string, input *opa.PolicyInput, result *opa.Result, err error, duration time.Duration) *AuditLog {
	log := &AuditLog{
		Timestamp:    time.Now(),
		DecisionPath: decisionPath,
		Input:        input,
		Result:       result,
		Duration:     duration.Milliseconds(),
	}

	// 设置请求信息
	log.Request.Method = c.Request.Method
	log.Request.Path = c.Request.URL.Path
	log.Request.ClientIP = c.ClientIP()
	log.Request.UserAgent = c.Request.UserAgent()

	// 设置决策结果
	if result != nil {
		log.DecisionID = result.DecisionID
		log.Allowed = result.Allowed
	}

	// 设置错误信息
	if err != nil {
		log.Error = err.Error()
		log.Allowed = false
	}

	return log
}

// isInWhiteList 检查路径是否在白名单中
func isInWhiteList(path string, whiteList []string) bool {
	for _, whitePath := range whiteList {
		if path == whitePath {
			return true
		}
	}
	return false
}

// PolicyInputModifier 策略输入修改器
// 允许在中间件链中修改策略输入
type PolicyInputModifier func(*opa.PolicyInput)

// WithResourceInfo 添加资源信息到策略输入
func WithResourceInfo(resourceType, resourceID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("resource_type", resourceType)
		c.Set("resource_id", resourceID)
		c.Next()
	}
}

// WithCustomAttribute 添加自定义属性到策略输入
func WithCustomAttribute(key string, getValue func(*gin.Context) interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		value := getValue(c)
		c.Set(fmt.Sprintf("custom_%s", key), value)
		c.Next()
	}
}
