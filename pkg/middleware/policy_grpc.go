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
	"time"

	"github.com/innovationmech/swit/pkg/security/opa"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCPolicyConfig gRPC 策略拦截器配置
type GRPCPolicyConfig struct {
	// Client OPA 客户端
	Client opa.Client

	// DecisionPath 决策路径（例如 "authz/allow"）
	DecisionPath string

	// SkipMethods 跳过策略评估的方法列表（支持通配符，如 "/grpc.health.v1.Health/*"）
	SkipMethods []string

	// InputBuilder 自定义输入构建函数
	InputBuilder func(context.Context, string) (*opa.PolicyInput, error)

	// ErrorHandler 自定义错误处理函数
	ErrorHandler func(context.Context, error) error

	// DeniedHandler 自定义拒绝处理函数
	DeniedHandler func(context.Context, *opa.Result) error

	// Logger 日志记录器
	Logger *zap.Logger

	// EnableAuditLog 启用审计日志
	EnableAuditLog bool

	// AuditLogHandler 自定义审计日志处理函数
	AuditLogHandler func(*GRPCAuditLog)

	// Timeout 评估超时时间
	Timeout time.Duration

	// Optional 可选模式，如果策略评估失败，仍然允许请求继续（仅记录日志）
	Optional bool
}

// GRPCAuditLog gRPC 审计日志
type GRPCAuditLog struct {
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
		Method   string `json:"method"`
		ClientIP string `json:"client_ip"`
	} `json:"request"`
}

// GRPCPolicyOption gRPC 策略拦截器选项函数
type GRPCPolicyOption func(*GRPCPolicyConfig)

// WithGRPCDecisionPath 设置决策路径
func WithGRPCDecisionPath(path string) GRPCPolicyOption {
	return func(c *GRPCPolicyConfig) {
		c.DecisionPath = path
	}
}

// WithGRPCSkipMethods 设置跳过的方法列表
func WithGRPCSkipMethods(methods []string) GRPCPolicyOption {
	return func(c *GRPCPolicyConfig) {
		c.SkipMethods = methods
	}
}

// WithGRPCInputBuilder 设置自定义输入构建函数
func WithGRPCInputBuilder(builder func(context.Context, string) (*opa.PolicyInput, error)) GRPCPolicyOption {
	return func(c *GRPCPolicyConfig) {
		c.InputBuilder = builder
	}
}

// WithGRPCErrorHandler 设置自定义错误处理函数
func WithGRPCErrorHandler(handler func(context.Context, error) error) GRPCPolicyOption {
	return func(c *GRPCPolicyConfig) {
		c.ErrorHandler = handler
	}
}

// WithGRPCDeniedHandler 设置自定义拒绝处理函数
func WithGRPCDeniedHandler(handler func(context.Context, *opa.Result) error) GRPCPolicyOption {
	return func(c *GRPCPolicyConfig) {
		c.DeniedHandler = handler
	}
}

// WithGRPCLogger 设置日志记录器
func WithGRPCLogger(logger *zap.Logger) GRPCPolicyOption {
	return func(c *GRPCPolicyConfig) {
		c.Logger = logger
	}
}

// WithGRPCAuditLog 启用审计日志
func WithGRPCAuditLog(handler func(*GRPCAuditLog)) GRPCPolicyOption {
	return func(c *GRPCPolicyConfig) {
		c.EnableAuditLog = true
		c.AuditLogHandler = handler
	}
}

// WithGRPCTimeout 设置评估超时时间
func WithGRPCTimeout(timeout time.Duration) GRPCPolicyOption {
	return func(c *GRPCPolicyConfig) {
		c.Timeout = timeout
	}
}

// WithGRPCOptional 设置可选模式
func WithGRPCOptional(optional bool) GRPCPolicyOption {
	return func(c *GRPCPolicyConfig) {
		c.Optional = optional
	}
}

// UnaryPolicyInterceptor 创建一元 RPC 策略拦截器
// 对每个 gRPC 一元请求进行策略评估，决定是否允许访问
func UnaryPolicyInterceptor(client opa.Client, options ...GRPCPolicyOption) grpc.UnaryServerInterceptor {
	config := &GRPCPolicyConfig{
		Client:       client,
		DecisionPath: "authz/allow",
		SkipMethods:  []string{},
		Timeout:      5 * time.Second,
	}

	// 应用选项
	for _, opt := range options {
		opt(config)
	}

	// 设置默认的输入构建函数
	if config.InputBuilder == nil {
		config.InputBuilder = defaultGRPCPolicyInputBuilder
	}

	// 设置默认的错误处理函数
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultGRPCPolicyErrorHandler
	}

	// 设置默认的拒绝处理函数
	if config.DeniedHandler == nil {
		config.DeniedHandler = defaultGRPCPolicyDeniedHandler
	}

	// 设置默认的审计日志处理函数
	if config.EnableAuditLog && config.AuditLogHandler == nil {
		config.AuditLogHandler = defaultGRPCPolicyAuditLogHandler(config.Logger)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		// 检查是否跳过此方法
		if shouldSkipMethod(info.FullMethod, config.SkipMethods) {
			return handler(ctx, req)
		}

		// 构建策略输入
		input, err := config.InputBuilder(ctx, info.FullMethod)
		if err != nil {
			if config.Logger != nil {
				config.Logger.Error("Failed to build policy input",
					zap.Error(err),
					zap.String("method", sanitizeLogValue(info.FullMethod)))
			}
			if config.Optional {
				return handler(ctx, req)
			}
			return nil, config.ErrorHandler(ctx, err)
		}

		// 创建评估上下文
		evalCtx := ctx
		if config.Timeout > 0 {
			var cancel context.CancelFunc
			evalCtx, cancel = context.WithTimeout(ctx, config.Timeout)
			defer cancel()
		}

		// 获取决策路径（支持从上下文动态获取）
		decisionPath := config.DecisionPath
		if dynamicPath := ctx.Value("policy_path"); dynamicPath != nil {
			if pathStr, ok := dynamicPath.(string); ok && pathStr != "" {
				decisionPath = pathStr
			}
		}

		// 评估策略
		result, err := config.Client.Evaluate(evalCtx, decisionPath, input)
		duration := time.Since(startTime)

		// 记录审计日志
		if config.EnableAuditLog {
			auditLog := buildGRPCAuditLog(ctx, info.FullMethod, decisionPath, input, result, err, duration)
			config.AuditLogHandler(auditLog)
		}

		// 处理评估错误
		if err != nil {
			if config.Logger != nil {
				config.Logger.Error("Policy evaluation failed",
					zap.Error(err),
					zap.String("method", sanitizeLogValue(info.FullMethod)),
					zap.String("decision_path", sanitizeLogValue(decisionPath)),
					zap.Duration("duration", duration))
			}
			if config.Optional {
				return handler(ctx, req)
			}
			return nil, config.ErrorHandler(ctx, err)
		}

		// 检查决策结果
		if !result.Allowed {
			if config.Logger != nil {
				config.Logger.Warn("Policy evaluation denied",
					zap.String("method", sanitizeLogValue(info.FullMethod)),
					zap.String("decision_path", sanitizeLogValue(decisionPath)),
					zap.String("user_id", sanitizeLogValue(input.User.ID)),
					zap.Duration("duration", duration))
			}
			if config.Optional {
				return handler(ctx, req)
			}
			return nil, config.DeniedHandler(ctx, result)
		}

		// 允许访问，记录日志
		if config.Logger != nil {
			config.Logger.Debug("Policy evaluation allowed",
				zap.String("method", sanitizeLogValue(info.FullMethod)),
				zap.String("decision_path", sanitizeLogValue(decisionPath)),
				zap.String("user_id", sanitizeLogValue(input.User.ID)),
				zap.Duration("duration", duration))
		}

		return handler(ctx, req)
	}
}

// StreamPolicyInterceptor 创建流式 RPC 策略拦截器
// 对每个 gRPC 流式请求进行策略评估，决定是否允许访问
func StreamPolicyInterceptor(client opa.Client, options ...GRPCPolicyOption) grpc.StreamServerInterceptor {
	config := &GRPCPolicyConfig{
		Client:       client,
		DecisionPath: "authz/allow",
		SkipMethods:  []string{},
		Timeout:      5 * time.Second,
	}

	// 应用选项
	for _, opt := range options {
		opt(config)
	}

	// 设置默认的输入构建函数
	if config.InputBuilder == nil {
		config.InputBuilder = defaultGRPCPolicyInputBuilder
	}

	// 设置默认的错误处理函数
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultGRPCPolicyErrorHandler
	}

	// 设置默认的拒绝处理函数
	if config.DeniedHandler == nil {
		config.DeniedHandler = defaultGRPCPolicyDeniedHandler
	}

	// 设置默认的审计日志处理函数
	if config.EnableAuditLog && config.AuditLogHandler == nil {
		config.AuditLogHandler = defaultGRPCPolicyAuditLogHandler(config.Logger)
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()
		ctx := ss.Context()

		// 检查是否跳过此方法
		if shouldSkipMethod(info.FullMethod, config.SkipMethods) {
			return handler(srv, ss)
		}

		// 构建策略输入
		input, err := config.InputBuilder(ctx, info.FullMethod)
		if err != nil {
			if config.Logger != nil {
				config.Logger.Error("Failed to build policy input",
					zap.Error(err),
					zap.String("method", sanitizeLogValue(info.FullMethod)))
			}
			if config.Optional {
				return handler(srv, ss)
			}
			return config.ErrorHandler(ctx, err)
		}

		// 创建评估上下文
		evalCtx := ctx
		if config.Timeout > 0 {
			var cancel context.CancelFunc
			evalCtx, cancel = context.WithTimeout(ctx, config.Timeout)
			defer cancel()
		}

		// 获取决策路径（支持从上下文动态获取）
		decisionPath := config.DecisionPath
		if dynamicPath := ctx.Value("policy_path"); dynamicPath != nil {
			if pathStr, ok := dynamicPath.(string); ok && pathStr != "" {
				decisionPath = pathStr
			}
		}

		// 评估策略
		result, err := config.Client.Evaluate(evalCtx, decisionPath, input)
		duration := time.Since(startTime)

		// 记录审计日志
		if config.EnableAuditLog {
			auditLog := buildGRPCAuditLog(ctx, info.FullMethod, decisionPath, input, result, err, duration)
			config.AuditLogHandler(auditLog)
		}

		// 处理评估错误
		if err != nil {
			if config.Logger != nil {
				config.Logger.Error("Policy evaluation failed",
					zap.Error(err),
					zap.String("method", sanitizeLogValue(info.FullMethod)),
					zap.String("decision_path", sanitizeLogValue(decisionPath)),
					zap.Duration("duration", duration))
			}
			if config.Optional {
				return handler(srv, ss)
			}
			return config.ErrorHandler(ctx, err)
		}

		// 检查决策结果
		if !result.Allowed {
			if config.Logger != nil {
				config.Logger.Warn("Policy evaluation denied",
					zap.String("method", sanitizeLogValue(info.FullMethod)),
					zap.String("decision_path", sanitizeLogValue(decisionPath)),
					zap.String("user_id", sanitizeLogValue(input.User.ID)),
					zap.Duration("duration", duration))
			}
			if config.Optional {
				return handler(srv, ss)
			}
			return config.DeniedHandler(ctx, result)
		}

		// 允许访问，记录日志
		if config.Logger != nil {
			config.Logger.Debug("Policy evaluation allowed",
				zap.String("method", sanitizeLogValue(info.FullMethod)),
				zap.String("decision_path", sanitizeLogValue(decisionPath)),
				zap.String("user_id", sanitizeLogValue(input.User.ID)),
				zap.Duration("duration", duration))
		}

		return handler(srv, ss)
	}
}

// defaultGRPCPolicyInputBuilder 默认的 gRPC 输入构建函数
func defaultGRPCPolicyInputBuilder(ctx context.Context, fullMethod string) (*opa.PolicyInput, error) {
	// 使用 PolicyInputBuilder 构建输入
	builder := opa.NewPolicyInputBuilder().FromGRPCContext(ctx, fullMethod)

	// 提取用户信息
	user := opa.ExtractUserFromGRPCContext(ctx)
	builder.WithUser(user)

	// 提取资源信息（如果有）
	if resourceType := ctx.Value("resource_type"); resourceType != nil {
		if rt, ok := resourceType.(string); ok {
			builder.WithResourceType(rt)
		}
	}

	if resourceID := ctx.Value("resource_id"); resourceID != nil {
		if rid, ok := resourceID.(string); ok {
			builder.WithResourceID(rid)
		}
	}

	return builder.Build(), nil
}

// defaultGRPCPolicyErrorHandler 默认的 gRPC 错误处理函数
func defaultGRPCPolicyErrorHandler(ctx context.Context, err error) error {
	return status.Error(codes.Internal, "policy evaluation failed")
}

// defaultGRPCPolicyDeniedHandler 默认的 gRPC 拒绝处理函数
func defaultGRPCPolicyDeniedHandler(ctx context.Context, result *opa.Result) error {
	return status.Error(codes.PermissionDenied, "access denied")
}

// defaultGRPCPolicyAuditLogHandler 默认的 gRPC 审计日志处理函数
func defaultGRPCPolicyAuditLogHandler(logger *zap.Logger) func(*GRPCAuditLog) {
	return func(log *GRPCAuditLog) {
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

// buildGRPCAuditLog 构建 gRPC 审计日志
func buildGRPCAuditLog(ctx context.Context, fullMethod, decisionPath string, input *opa.PolicyInput, result *opa.Result, err error, duration time.Duration) *GRPCAuditLog {
	log := &GRPCAuditLog{
		Timestamp:    time.Now(),
		DecisionPath: decisionPath,
		Input:        input,
		Result:       result,
		Duration:     duration.Milliseconds(),
	}

	// 设置请求信息
	log.Request.Method = fullMethod
	log.Request.ClientIP = input.Request.ClientIP

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

// GRPCResourcePolicy 创建资源级策略拦截器（一元 RPC）
// 对特定资源进行策略评估
func GRPCResourcePolicy(resourceType string, getResourceID func(context.Context) string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 获取资源 ID
		resourceID := getResourceID(ctx)
		if resourceID == "" {
			return nil, status.Error(codes.InvalidArgument, "resource ID is required")
		}

		// 将资源信息存储到上下文中，供策略拦截器使用
		ctx = context.WithValue(ctx, "resource_type", resourceType)
		ctx = context.WithValue(ctx, "resource_id", resourceID)

		return handler(ctx, req)
	}
}

// GRPCDynamicPolicyPath 创建动态策略路径拦截器（一元 RPC）
// 根据请求动态确定策略路径
func GRPCDynamicPolicyPath(getPath func(context.Context) string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 获取策略路径
		path := getPath(ctx)
		if path == "" {
			return nil, status.Error(codes.Internal, "failed to determine policy path")
		}

		// 将策略路径存储到上下文中
		ctx = context.WithValue(ctx, "policy_path", path)

		return handler(ctx, req)
	}
}

// GRPCStreamResourcePolicy 创建资源级策略拦截器（流式 RPC）
// 对特定资源进行策略评估
func GRPCStreamResourcePolicy(resourceType string, getResourceID func(context.Context) string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		// 获取资源 ID
		resourceID := getResourceID(ctx)
		if resourceID == "" {
			return status.Error(codes.InvalidArgument, "resource ID is required")
		}

		// 将资源信息存储到上下文中
		ctx = context.WithValue(ctx, "resource_type", resourceType)
		ctx = context.WithValue(ctx, "resource_id", resourceID)

		// 创建包装的流，使用新的上下文
		wrappedStream := &policyServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

// GRPCStreamDynamicPolicyPath 创建动态策略路径拦截器（流式 RPC）
// 根据请求动态确定策略路径
func GRPCStreamDynamicPolicyPath(getPath func(context.Context) string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		// 获取策略路径
		path := getPath(ctx)
		if path == "" {
			return status.Error(codes.Internal, "failed to determine policy path")
		}

		// 将策略路径存储到上下文中
		ctx = context.WithValue(ctx, "policy_path", path)

		// 创建包装的流，使用新的上下文
		wrappedStream := &policyServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

// policyServerStream 包装 ServerStream 以支持上下文修改
type policyServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context 返回包装的上下文
func (s *policyServerStream) Context() context.Context {
	return s.ctx
}
