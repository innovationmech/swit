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

// Package audit provides comprehensive security audit logging capabilities
// for the Swit microservice framework.
package audit

import "time"

// 审计事件类型常量
const (
	// 认证相关事件
	EventTypeAuthAttempt = "auth.attempt"
	EventTypeAuthSuccess = "auth.success"
	EventTypeAuthFailure = "auth.failure"
	EventTypeAuthLogout  = "auth.logout"
	EventTypeAuthRefresh = "auth.refresh"
	EventTypeAuthRevoke  = "auth.revoke"

	// 授权相关事件
	EventTypePolicyEvaluation = "policy.evaluation"
	EventTypeAccessDenied     = "access.denied"
	EventTypeAccessGranted    = "access.granted"
	EventTypePermissionChange = "permission.change"
	EventTypeRoleChange       = "role.change"

	// 敏感操作相关事件
	EventTypeSensitiveOperation = "sensitive.operation"
	EventTypeDataAccess         = "data.access"
	EventTypeDataModification   = "data.modification"
	EventTypeDataDeletion       = "data.deletion"
	EventTypeDataExport         = "data.export"

	// 配置变更事件
	EventTypeConfigChange   = "config.change"
	EventTypeSecretAccess   = "secret.access"
	EventTypeSecretRotation = "secret.rotation"

	// 安全事件
	EventTypeSecurityViolation  = "security.violation"
	EventTypeSuspiciousActivity = "suspicious.activity"
	EventTypeBruteForceDetected = "brute_force.detected"
	EventTypeRateLimitExceeded  = "rate_limit.exceeded"

	// TLS/连接事件
	EventTypeTLSHandshake       = "tls.handshake"
	EventTypeTLSHandshakeFailed = "tls.handshake.failed"
	EventTypeCertificateExpired = "certificate.expired"

	// 会话事件
	EventTypeSessionCreated = "session.created"
	EventTypeSessionExpired = "session.expired"
	EventTypeSessionRevoked = "session.revoked"
	EventTypeSessionHijack  = "session.hijack"
)

// 严重性级别常量
const (
	SeverityInfo     = "info"
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// 审计结果常量
const (
	ResultSuccess = "success"
	ResultFailure = "failure"
	ResultDenied  = "denied"
	ResultError   = "error"
)

// AuditEvent 表示一个安全审计事件
type AuditEvent struct {
	// 基础信息
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`

	// 主体信息（谁）
	Actor ActorInfo `json:"actor"`

	// 操作信息（做了什么）
	Action ActionInfo `json:"action"`

	// 目标信息（对什么）
	Target TargetInfo `json:"target"`

	// 结果信息
	Result ResultInfo `json:"result"`

	// 上下文信息
	Context ContextInfo `json:"context"`

	// 附加元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ActorInfo 表示执行操作的主体信息
type ActorInfo struct {
	UserID      string   `json:"user_id,omitempty"`
	Username    string   `json:"username,omitempty"`
	Email       string   `json:"email,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	ServiceID   string   `json:"service_id,omitempty"`
	ServiceName string   `json:"service_name,omitempty"`
	ActorType   string   `json:"actor_type"` // user, service, system, anonymous
	SessionID   string   `json:"session_id,omitempty"`
	IPAddress   string   `json:"ip_address,omitempty"`
	UserAgent   string   `json:"user_agent,omitempty"`
}

// ActionInfo 表示执行的操作信息
type ActionInfo struct {
	Operation   string                 `json:"operation"`
	Method      string                 `json:"method,omitempty"` // HTTP method, gRPC method, etc.
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// TargetInfo 表示操作目标信息
type TargetInfo struct {
	ResourceType string                 `json:"resource_type,omitempty"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	ResourceName string                 `json:"resource_name,omitempty"`
	ResourcePath string                 `json:"resource_path,omitempty"`
	Owner        string                 `json:"owner,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
}

// ResultInfo 表示操作结果信息
type ResultInfo struct {
	Status       string `json:"status"` // success, failure, denied, error
	StatusCode   int    `json:"status_code,omitempty"`
	Message      string `json:"message,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	Duration     int64  `json:"duration_ms,omitempty"` // 操作耗时（毫秒）
}

// ContextInfo 表示操作上下文信息
type ContextInfo struct {
	RequestID      string            `json:"request_id,omitempty"`
	TraceID        string            `json:"trace_id,omitempty"`
	SpanID         string            `json:"span_id,omitempty"`
	CorrelationID  string            `json:"correlation_id,omitempty"`
	Environment    string            `json:"environment,omitempty"`
	ServiceName    string            `json:"service_name,omitempty"`
	ServiceVersion string            `json:"service_version,omitempty"`
	Hostname       string            `json:"hostname,omitempty"`
	Location       *LocationInfo     `json:"location,omitempty"`
	Device         *DeviceInfo       `json:"device,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
}

// LocationInfo 表示地理位置信息
type LocationInfo struct {
	Country   string  `json:"country,omitempty"`
	Region    string  `json:"region,omitempty"`
	City      string  `json:"city,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

// DeviceInfo 表示设备信息
type DeviceInfo struct {
	DeviceID   string `json:"device_id,omitempty"`
	DeviceType string `json:"device_type,omitempty"` // mobile, desktop, tablet, etc.
	OS         string `json:"os,omitempty"`
	Browser    string `json:"browser,omitempty"`
	AppVersion string `json:"app_version,omitempty"`
}

// NewAuditEvent 创建一个新的审计事件
func NewAuditEvent(eventType, severity string) *AuditEvent {
	return &AuditEvent{
		EventID:   generateEventID(),
		EventType: eventType,
		Timestamp: time.Now().UTC(),
		Severity:  severity,
		Metadata:  make(map[string]interface{}),
	}
}

// WithActor 设置审计事件的主体信息
func (e *AuditEvent) WithActor(actor ActorInfo) *AuditEvent {
	e.Actor = actor
	return e
}

// WithAction 设置审计事件的操作信息
func (e *AuditEvent) WithAction(action ActionInfo) *AuditEvent {
	e.Action = action
	return e
}

// WithTarget 设置审计事件的目标信息
func (e *AuditEvent) WithTarget(target TargetInfo) *AuditEvent {
	e.Target = target
	return e
}

// WithResult 设置审计事件的结果信息
func (e *AuditEvent) WithResult(result ResultInfo) *AuditEvent {
	e.Result = result
	return e
}

// WithContext 设置审计事件的上下文信息
func (e *AuditEvent) WithContext(context ContextInfo) *AuditEvent {
	e.Context = context
	return e
}

// AddMetadata 添加元数据
func (e *AuditEvent) AddMetadata(key string, value interface{}) *AuditEvent {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// generateEventID 生成唯一的事件ID
func generateEventID() string {
	// 使用时间戳和随机数生成唯一ID
	// 格式：audit-{timestamp}-{random}
	return time.Now().Format("20060102150405") + "-" + generateRandomString(8)
}

// generateRandomString 生成指定长度的随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%(int64(len(charset)))]
	}
	return string(result)
}

// IsAuthenticationEvent 检查是否为认证事件
func (e *AuditEvent) IsAuthenticationEvent() bool {
	return e.EventType == EventTypeAuthAttempt ||
		e.EventType == EventTypeAuthSuccess ||
		e.EventType == EventTypeAuthFailure ||
		e.EventType == EventTypeAuthLogout ||
		e.EventType == EventTypeAuthRefresh ||
		e.EventType == EventTypeAuthRevoke
}

// IsAuthorizationEvent 检查是否为授权事件
func (e *AuditEvent) IsAuthorizationEvent() bool {
	return e.EventType == EventTypePolicyEvaluation ||
		e.EventType == EventTypeAccessDenied ||
		e.EventType == EventTypeAccessGranted ||
		e.EventType == EventTypePermissionChange ||
		e.EventType == EventTypeRoleChange
}

// IsSensitiveOperation 检查是否为敏感操作
func (e *AuditEvent) IsSensitiveOperation() bool {
	return e.EventType == EventTypeSensitiveOperation ||
		e.EventType == EventTypeDataAccess ||
		e.EventType == EventTypeDataModification ||
		e.EventType == EventTypeDataDeletion ||
		e.EventType == EventTypeDataExport
}

// IsSecurityEvent 检查是否为安全事件
func (e *AuditEvent) IsSecurityEvent() bool {
	return e.EventType == EventTypeSecurityViolation ||
		e.EventType == EventTypeSuspiciousActivity ||
		e.EventType == EventTypeBruteForceDetected ||
		e.EventType == EventTypeRateLimitExceeded
}

// IsCritical 检查是否为严重事件
func (e *AuditEvent) IsCritical() bool {
	return e.Severity == SeverityCritical || e.Severity == SeverityHigh
}
