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
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// PolicyInput 策略输入数据结构
// 用于 OPA 策略评估的标准化输入格式
type PolicyInput struct {
	// Request 请求信息
	Request RequestInfo `json:"request"`

	// User 用户信息
	User UserInfo `json:"user"`

	// Resource 资源信息
	Resource ResourceInfo `json:"resource"`

	// Custom 自定义数据
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// RequestInfo 请求信息
type RequestInfo struct {
	// Method HTTP 方法或 gRPC 方法名
	Method string `json:"method"`

	// Path 请求路径或 gRPC 服务路径
	Path string `json:"path"`

	// Headers 请求头（仅包含白名单的头）
	Headers map[string][]string `json:"headers,omitempty"`

	// Query 查询参数（仅 HTTP）
	Query map[string][]string `json:"query,omitempty"`

	// ClientIP 客户端 IP 地址
	ClientIP string `json:"client_ip,omitempty"`

	// Protocol 协议类型（http, grpc）
	Protocol string `json:"protocol"`

	// Time 请求时间
	Time time.Time `json:"time"`

	// Host 主机名
	Host string `json:"host,omitempty"`

	// UserAgent 用户代理
	UserAgent string `json:"user_agent,omitempty"`
}

// UserInfo 用户信息
type UserInfo struct {
	// ID 用户 ID
	ID string `json:"id,omitempty"`

	// Username 用户名
	Username string `json:"username,omitempty"`

	// Email 邮箱
	Email string `json:"email,omitempty"`

	// Roles 角色列表
	Roles []string `json:"roles,omitempty"`

	// Permissions 权限列表
	Permissions []string `json:"permissions,omitempty"`

	// Groups 用户组列表
	Groups []string `json:"groups,omitempty"`

	// Attributes 其他属性
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// ResourceInfo 资源信息
type ResourceInfo struct {
	// Type 资源类型
	Type string `json:"type,omitempty"`

	// ID 资源 ID
	ID string `json:"id,omitempty"`

	// Name 资源名称
	Name string `json:"name,omitempty"`

	// Owner 资源所有者
	Owner string `json:"owner,omitempty"`

	// Attributes 其他属性
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// PolicyInputBuilder 策略输入构建器
type PolicyInputBuilder struct {
	input *PolicyInput
}

// NewPolicyInputBuilder 创建策略输入构建器
func NewPolicyInputBuilder() *PolicyInputBuilder {
	return &PolicyInputBuilder{
		input: &PolicyInput{
			Request: RequestInfo{
				Time:     time.Now(),
				Protocol: "unknown",
			},
			User: UserInfo{
				Roles:       []string{},
				Permissions: []string{},
				Groups:      []string{},
				Attributes:  make(map[string]interface{}),
			},
			Resource: ResourceInfo{
				Attributes: make(map[string]interface{}),
			},
			Custom: make(map[string]interface{}),
		},
	}
}

// FromHTTPRequest 从 Gin HTTP 请求构建输入
func (b *PolicyInputBuilder) FromHTTPRequest(c *gin.Context) *PolicyInputBuilder {
	b.input.Request.Method = c.Request.Method
	b.input.Request.Path = c.Request.URL.Path
	b.input.Request.ClientIP = c.ClientIP()
	b.input.Request.Protocol = "http"
	b.input.Request.Host = c.Request.Host
	b.input.Request.UserAgent = c.Request.UserAgent()

	// 提取查询参数
	if len(c.Request.URL.Query()) > 0 {
		b.input.Request.Query = make(map[string][]string)
		for k, v := range c.Request.URL.Query() {
			b.input.Request.Query[k] = v
		}
	}

	// 提取请求头（仅白名单的头）
	b.input.Request.Headers = extractSafeHeaders(c.Request.Header)

	return b
}

// FromGRPCContext 从 gRPC 上下文构建输入
func (b *PolicyInputBuilder) FromGRPCContext(ctx context.Context, fullMethod string) *PolicyInputBuilder {
	b.input.Request.Method = fullMethod
	b.input.Request.Path = fullMethod
	b.input.Request.Protocol = "grpc"

	// 提取客户端 IP
	if p, ok := peer.FromContext(ctx); ok {
		b.input.Request.ClientIP = p.Addr.String()
	}

	// 提取 metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		b.input.Request.Headers = make(map[string][]string)
		for k, v := range md {
			// 只包含安全的 metadata
			if isSafeMetadataKey(k) {
				b.input.Request.Headers[k] = v
			}
		}
	}

	return b
}

// WithUser 设置用户信息
func (b *PolicyInputBuilder) WithUser(user UserInfo) *PolicyInputBuilder {
	b.input.User = user
	return b
}

// WithUserID 设置用户 ID
func (b *PolicyInputBuilder) WithUserID(id string) *PolicyInputBuilder {
	b.input.User.ID = id
	return b
}

// WithUsername 设置用户名
func (b *PolicyInputBuilder) WithUsername(username string) *PolicyInputBuilder {
	b.input.User.Username = username
	return b
}

// WithUserEmail 设置用户邮箱
func (b *PolicyInputBuilder) WithUserEmail(email string) *PolicyInputBuilder {
	b.input.User.Email = email
	return b
}

// WithUserRoles 设置用户角色
func (b *PolicyInputBuilder) WithUserRoles(roles []string) *PolicyInputBuilder {
	b.input.User.Roles = roles
	return b
}

// WithUserPermissions 设置用户权限
func (b *PolicyInputBuilder) WithUserPermissions(permissions []string) *PolicyInputBuilder {
	b.input.User.Permissions = permissions
	return b
}

// WithUserGroups 设置用户组
func (b *PolicyInputBuilder) WithUserGroups(groups []string) *PolicyInputBuilder {
	b.input.User.Groups = groups
	return b
}

// WithUserAttribute 设置用户属性
func (b *PolicyInputBuilder) WithUserAttribute(key string, value interface{}) *PolicyInputBuilder {
	if b.input.User.Attributes == nil {
		b.input.User.Attributes = make(map[string]interface{})
	}
	b.input.User.Attributes[key] = value
	return b
}

// WithResource 设置资源信息
func (b *PolicyInputBuilder) WithResource(resource ResourceInfo) *PolicyInputBuilder {
	b.input.Resource = resource
	return b
}

// WithResourceType 设置资源类型
func (b *PolicyInputBuilder) WithResourceType(resourceType string) *PolicyInputBuilder {
	b.input.Resource.Type = resourceType
	return b
}

// WithResourceID 设置资源 ID
func (b *PolicyInputBuilder) WithResourceID(id string) *PolicyInputBuilder {
	b.input.Resource.ID = id
	return b
}

// WithResourceName 设置资源名称
func (b *PolicyInputBuilder) WithResourceName(name string) *PolicyInputBuilder {
	b.input.Resource.Name = name
	return b
}

// WithResourceOwner 设置资源所有者
func (b *PolicyInputBuilder) WithResourceOwner(owner string) *PolicyInputBuilder {
	b.input.Resource.Owner = owner
	return b
}

// WithResourceAttribute 设置资源属性
func (b *PolicyInputBuilder) WithResourceAttribute(key string, value interface{}) *PolicyInputBuilder {
	if b.input.Resource.Attributes == nil {
		b.input.Resource.Attributes = make(map[string]interface{})
	}
	b.input.Resource.Attributes[key] = value
	return b
}

// WithCustomData 设置自定义数据
func (b *PolicyInputBuilder) WithCustomData(key string, value interface{}) *PolicyInputBuilder {
	if b.input.Custom == nil {
		b.input.Custom = make(map[string]interface{})
	}
	b.input.Custom[key] = value
	return b
}

// Build 构建策略输入
func (b *PolicyInputBuilder) Build() *PolicyInput {
	return b.input
}

// ExtractUserFromContext 从 Gin 上下文提取用户信息
// 支持从 JWT claims 或其他认证中间件设置的上下文中提取
func ExtractUserFromContext(c *gin.Context) UserInfo {
	user := UserInfo{
		Roles:       []string{},
		Permissions: []string{},
		Groups:      []string{},
		Attributes:  make(map[string]interface{}),
	}

	// 尝试从上下文中提取用户信息
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			user.ID = id
		}
	}

	if username, exists := c.Get("username"); exists {
		if name, ok := username.(string); ok {
			user.Username = name
		}
	}

	if email, exists := c.Get("email"); exists {
		if e, ok := email.(string); ok {
			user.Email = e
		}
	}

	if roles, exists := c.Get("roles"); exists {
		switch r := roles.(type) {
		case []string:
			user.Roles = r
		case []interface{}:
			for _, role := range r {
				if roleStr, ok := role.(string); ok {
					user.Roles = append(user.Roles, roleStr)
				}
			}
		}
	}

	if permissions, exists := c.Get("permissions"); exists {
		switch p := permissions.(type) {
		case []string:
			user.Permissions = p
		case []interface{}:
			for _, perm := range p {
				if permStr, ok := perm.(string); ok {
					user.Permissions = append(user.Permissions, permStr)
				}
			}
		}
	}

	if groups, exists := c.Get("groups"); exists {
		switch g := groups.(type) {
		case []string:
			user.Groups = g
		case []interface{}:
			for _, group := range g {
				if groupStr, ok := group.(string); ok {
					user.Groups = append(user.Groups, groupStr)
				}
			}
		}
	}

	// 尝试从 claims 提取额外信息
	if claims, exists := c.Get("claims"); exists {
		if claimsMap, ok := claims.(map[string]interface{}); ok {
			// 如果上面没有提取到，从 claims 中提取
			if user.ID == "" {
				if sub, ok := claimsMap["sub"].(string); ok {
					user.ID = sub
				}
			}
			if user.Email == "" {
				if email, ok := claimsMap["email"].(string); ok {
					user.Email = email
				}
			}
			// 其他 claims 存入 attributes
			for k, v := range claimsMap {
				if k != "sub" && k != "email" && k != "roles" && k != "permissions" && k != "groups" {
					user.Attributes[k] = v
				}
			}
		}
	}

	return user
}

// ExtractUserFromGRPCContext 从 gRPC 上下文提取用户信息
func ExtractUserFromGRPCContext(ctx context.Context) UserInfo {
	user := UserInfo{
		Roles:       []string{},
		Permissions: []string{},
		Groups:      []string{},
		Attributes:  make(map[string]interface{}),
	}

	// 尝试从 context.Value 中提取用户信息
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			user.ID = id
		}
	}

	if username := ctx.Value("username"); username != nil {
		if name, ok := username.(string); ok {
			user.Username = name
		}
	}

	if email := ctx.Value("email"); email != nil {
		if e, ok := email.(string); ok {
			user.Email = e
		}
	}

	if roles := ctx.Value("roles"); roles != nil {
		switch r := roles.(type) {
		case []string:
			user.Roles = r
		case []interface{}:
			for _, role := range r {
				if roleStr, ok := role.(string); ok {
					user.Roles = append(user.Roles, roleStr)
				}
			}
		}
	}

	if permissions := ctx.Value("permissions"); permissions != nil {
		switch p := permissions.(type) {
		case []string:
			user.Permissions = p
		case []interface{}:
			for _, perm := range p {
				if permStr, ok := perm.(string); ok {
					user.Permissions = append(user.Permissions, permStr)
				}
			}
		}
	}

	if groups := ctx.Value("groups"); groups != nil {
		switch g := groups.(type) {
		case []string:
			user.Groups = g
		case []interface{}:
			for _, group := range g {
				if groupStr, ok := group.(string); ok {
					user.Groups = append(user.Groups, groupStr)
				}
			}
		}
	}

	return user
}

// extractSafeHeaders 提取安全的请求头
// 排除敏感信息如 Authorization, Cookie 等
func extractSafeHeaders(headers http.Header) map[string][]string {
	safeHeaders := make(map[string][]string)

	// 白名单的请求头
	allowedHeaders := []string{
		"Content-Type",
		"Accept",
		"Accept-Language",
		"Accept-Encoding",
		"User-Agent",
		"Referer",
		"X-Request-Id",
		"X-Forwarded-For",
		"X-Real-Ip",
		"X-Forwarded-Proto",
		"X-Forwarded-Host",
	}

	for _, header := range allowedHeaders {
		if values, ok := headers[header]; ok {
			safeHeaders[header] = values
		}
	}

	return safeHeaders
}

// isSafeMetadataKey 检查 gRPC metadata 键是否安全
// 排除敏感信息如 authorization 等
func isSafeMetadataKey(key string) bool {
	// 不安全的 metadata 键
	unsafeKeys := []string{
		"authorization",
		"cookie",
		"x-api-key",
		"x-auth-token",
	}

	for _, unsafeKey := range unsafeKeys {
		if key == unsafeKey {
			return false
		}
	}

	return true
}

// BuildPolicyInputFromHTTP 从 HTTP 请求构建策略输入（便捷函数）
func BuildPolicyInputFromHTTP(c *gin.Context) *PolicyInput {
	builder := NewPolicyInputBuilder().FromHTTPRequest(c)

	// 自动提取用户信息
	user := ExtractUserFromContext(c)
	builder.WithUser(user)

	return builder.Build()
}

// BuildPolicyInputFromGRPC 从 gRPC 上下文构建策略输入（便捷函数）
func BuildPolicyInputFromGRPC(ctx context.Context, fullMethod string) *PolicyInput {
	builder := NewPolicyInputBuilder().FromGRPCContext(ctx, fullMethod)

	// 自动提取用户信息
	user := ExtractUserFromGRPCContext(ctx)
	builder.WithUser(user)

	return builder.Build()
}
