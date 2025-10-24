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

// Package examples provides example implementations of Saga patterns for common business scenarios.
// This file demonstrates a user registration Saga with account creation, email verification,
// configuration initialization, and resource allocation steps.
package examples

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// ==========================
// 数据结构定义
// ==========================

// UserRegistrationData 包含用户注册所需的所有输入数据
type UserRegistrationData struct {
	// 基本信息
	Email     string // 邮箱地址
	Username  string // 用户名
	Password  string // 密码（已加密）
	FirstName string // 名字
	LastName  string // 姓氏
	Phone     string // 电话号码（可选）

	// 注册来源
	Source       string // 注册来源（web、mobile、api）
	ReferralCode string // 推荐码（可选）
	UtmSource    string // UTM 来源（可选）

	// 配置选项
	Language     string // 首选语言
	Timezone     string // 时区
	Subscription string // 订阅计划（free、premium、enterprise）

	// 同意条款
	AcceptedTOS bool      // 是否接受服务条款
	AcceptedAt  time.Time // 接受时间

	// 元数据
	Metadata map[string]interface{} // 额外元数据
}

// UserCreationResult 存储用户账户创建步骤的结果
type UserCreationResult struct {
	UserID       string                 // 用户ID
	Username     string                 // 用户名
	Email        string                 // 邮箱地址
	Status       string                 // 用户状态（pending、active、suspended）
	CreatedAt    time.Time              // 创建时间
	ActivationID string                 // 激活ID（用于邮件验证）
	Metadata     map[string]interface{} // 元数据
}

// EmailVerificationResult 存储邮件验证步骤的结果
type EmailVerificationResult struct {
	VerificationID   string                 // 验证ID
	UserID           string                 // 用户ID
	Email            string                 // 邮箱地址
	VerificationCode string                 // 验证码
	SentAt           time.Time              // 发送时间
	ExpiresAt        time.Time              // 过期时间
	Provider         string                 // 邮件服务提供商（如 SendGrid、SES）
	Metadata         map[string]interface{} // 元数据
}

// ConfigInitResult 存储配置初始化步骤的结果
type ConfigInitResult struct {
	UserID        string                 // 用户ID
	ConfigID      string                 // 配置ID
	Settings      map[string]interface{} // 用户设置
	Preferences   map[string]interface{} // 用户偏好
	InitializedAt time.Time              // 初始化时间
	Metadata      map[string]interface{} // 元数据
}

// ResourceAllocationResult 存储资源分配步骤的结果
type ResourceAllocationResult struct {
	UserID         string                 // 用户ID
	WalletID       string                 // 钱包ID
	StorageQuota   int64                  // 存储配额（字节）
	APIQuota       int                    // API 调用配额
	ComputeQuota   int                    // 计算资源配额
	AllocatedAt    time.Time              // 分配时间
	SubscriptionID string                 // 订阅ID
	Metadata       map[string]interface{} // 元数据
}

// AnalyticsTrackingResult 存储分析跟踪步骤的结果
type AnalyticsTrackingResult struct {
	UserID     string                 // 用户ID
	EventID    string                 // 事件ID
	EventType  string                 // 事件类型（user_registered）
	TrackedAt  time.Time              // 跟踪时间
	SessionID  string                 // 会话ID
	DeviceInfo map[string]interface{} // 设备信息
	Metadata   map[string]interface{} // 元数据
}

// ==========================
// 服务接口定义（用于模拟外部服务）
// ==========================

// UserServiceClient 用户服务客户端接口
type UserServiceClient interface {
	// CreateUser 创建用户账户
	CreateUser(ctx context.Context, data *UserRegistrationData) (*UserCreationResult, error)
	// DeleteUser 删除用户账户（用于补偿）
	DeleteUser(ctx context.Context, userID string, reason string) error
	// GetUserByEmail 根据邮箱查询用户
	GetUserByEmail(ctx context.Context, email string) (*UserCreationResult, error)
	// GetUserByUsername 根据用户名查询用户
	GetUserByUsername(ctx context.Context, username string) (*UserCreationResult, error)
}

// EmailServiceClient 邮件服务客户端接口
type EmailServiceClient interface {
	// SendVerificationEmail 发送验证邮件
	SendVerificationEmail(ctx context.Context, userID string, email string, username string) (*EmailVerificationResult, error)
	// CancelVerificationEmail 取消验证邮件（用于补偿）
	CancelVerificationEmail(ctx context.Context, verificationID string) error
	// ResendVerificationEmail 重发验证邮件
	ResendVerificationEmail(ctx context.Context, verificationID string) error
}

// ConfigServiceClient 配置服务客户端接口
type ConfigServiceClient interface {
	// InitializeUserConfig 初始化用户配置
	InitializeUserConfig(ctx context.Context, userID string, language string, timezone string) (*ConfigInitResult, error)
	// DeleteUserConfig 删除用户配置（用于补偿）
	DeleteUserConfig(ctx context.Context, userID string) error
	// UpdateUserConfig 更新用户配置
	UpdateUserConfig(ctx context.Context, userID string, settings map[string]interface{}) error
}

// ResourceServiceClient 资源服务客户端接口
type ResourceServiceClient interface {
	// AllocateResources 分配用户资源
	AllocateResources(ctx context.Context, userID string, subscription string) (*ResourceAllocationResult, error)
	// ReleaseResources 释放用户资源（用于补偿）
	ReleaseResources(ctx context.Context, userID string) error
	// GetResourceQuota 获取资源配额
	GetResourceQuota(ctx context.Context, subscription string) (*ResourceAllocationResult, error)
}

// AnalyticsServiceClient 分析服务客户端接口
type AnalyticsServiceClient interface {
	// TrackRegistrationEvent 跟踪注册事件
	TrackRegistrationEvent(ctx context.Context, userID string, data *UserRegistrationData) (*AnalyticsTrackingResult, error)
	// DeleteTrackingEvent 删除跟踪事件（用于补偿）
	DeleteTrackingEvent(ctx context.Context, eventID string) error
}

// ==========================
// Saga 步骤实现
// ==========================

// CreateUserAccountStep 创建用户账户步骤
type CreateUserAccountStep struct {
	service UserServiceClient
}

// GetID 返回步骤ID
func (s *CreateUserAccountStep) GetID() string {
	return "create-user-account"
}

// GetName 返回步骤名称
func (s *CreateUserAccountStep) GetName() string {
	return "创建用户账户"
}

// GetDescription 返回步骤描述
func (s *CreateUserAccountStep) GetDescription() string {
	return "在用户系统中创建新用户账户，初始状态为待验证"
}

// Execute 执行创建用户账户操作
// 输入：UserRegistrationData
// 输出：UserCreationResult
func (s *CreateUserAccountStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	registrationData, ok := data.(*UserRegistrationData)
	if !ok {
		return nil, errors.New("invalid data type: expected *UserRegistrationData")
	}

	// 验证注册数据
	if err := s.validateRegistrationData(registrationData); err != nil {
		return nil, fmt.Errorf("注册数据验证失败: %w", err)
	}

	// 检查邮箱是否已存在
	// 注意：必须正确处理查询错误，避免在服务不可用时误创建重复账户
	existingUser, err := s.service.GetUserByEmail(ctx, registrationData.Email)
	if err != nil {
		// 只有明确的"用户不存在"错误才继续，其他错误应该传播
		if !isUserNotFoundError(err) {
			return nil, fmt.Errorf("查询邮箱失败，无法验证唯一性: %w", err)
		}
		// err 是"用户不存在"错误，这是预期的，继续执行
	} else if existingUser != nil {
		// 查询成功且用户存在
		return nil, fmt.Errorf("邮箱已被注册: %s", registrationData.Email)
	}

	// 检查用户名是否已存在
	existingUser, err = s.service.GetUserByUsername(ctx, registrationData.Username)
	if err != nil {
		// 只有明确的"用户不存在"错误才继续，其他错误应该传播
		if !isUserNotFoundError(err) {
			return nil, fmt.Errorf("查询用户名失败，无法验证唯一性: %w", err)
		}
		// err 是"用户不存在"错误，这是预期的，继续执行
	} else if existingUser != nil {
		// 查询成功且用户存在
		return nil, fmt.Errorf("用户名已被使用: %s", registrationData.Username)
	}

	// 调用用户服务创建账户
	result, err := s.service.CreateUser(ctx, registrationData)
	if err != nil {
		return nil, fmt.Errorf("创建用户账户失败: %w", err)
	}

	return result, nil
}

// Compensate 补偿操作：删除用户账户
func (s *CreateUserAccountStep) Compensate(ctx context.Context, data interface{}) error {
	result, ok := data.(*UserCreationResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *UserCreationResult")
	}

	// 调用用户服务删除账户
	err := s.service.DeleteUser(ctx, result.UserID, "registration_failed")
	if err != nil {
		// 用户删除失败可能需要人工介入进行数据清理
		return fmt.Errorf("删除用户账户失败，需要手动清理: %w", err)
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *CreateUserAccountStep) GetTimeout() time.Duration {
	return 10 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *CreateUserAccountStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *CreateUserAccountStep) IsRetryable(err error) bool {
	// 网络错误和临时服务不可用错误可重试
	// 验证错误和业务规则错误（如邮箱已存在）不可重试
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// 检查是否是业务规则错误
	errMsg := err.Error()
	if containsSubstring(errMsg, "邮箱已被注册") || containsSubstring(errMsg, "用户名已被使用") {
		return false
	}
	return true
}

// GetMetadata 返回步骤元数据
func (s *CreateUserAccountStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "user_creation",
		"service":   "user-service",
		"critical":  true,
	}
}

// validateRegistrationData 验证注册数据的有效性
func (s *CreateUserAccountStep) validateRegistrationData(data *UserRegistrationData) error {
	// 验证邮箱
	if data.Email == "" {
		return errors.New("邮箱地址不能为空")
	}
	if !isValidEmail(data.Email) {
		return errors.New("邮箱地址格式无效")
	}

	// 验证用户名
	if data.Username == "" {
		return errors.New("用户名不能为空")
	}
	if len(data.Username) < 3 || len(data.Username) > 30 {
		return errors.New("用户名长度必须在3-30个字符之间")
	}
	if !isValidUsername(data.Username) {
		return errors.New("用户名只能包含字母、数字、下划线和连字符")
	}

	// 验证密码
	if data.Password == "" {
		return errors.New("密码不能为空")
	}
	if len(data.Password) < 8 {
		return errors.New("密码长度至少8个字符")
	}

	// 验证姓名
	if data.FirstName == "" {
		return errors.New("名字不能为空")
	}
	if data.LastName == "" {
		return errors.New("姓氏不能为空")
	}

	// 验证服务条款
	if !data.AcceptedTOS {
		return errors.New("必须接受服务条款")
	}

	return nil
}

// SendVerificationEmailStep 发送验证邮件步骤
type SendVerificationEmailStep struct {
	service EmailServiceClient
}

// GetID 返回步骤ID
func (s *SendVerificationEmailStep) GetID() string {
	return "send-verification-email"
}

// GetName 返回步骤名称
func (s *SendVerificationEmailStep) GetName() string {
	return "发送验证邮件"
}

// GetDescription 返回步骤描述
func (s *SendVerificationEmailStep) GetDescription() string {
	return "向用户邮箱发送验证邮件，包含激活链接和验证码"
}

// Execute 执行发送验证邮件操作
// 输入：UserCreationResult（从上一步传递）
// 输出：EmailVerificationResult
func (s *SendVerificationEmailStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	userResult, ok := data.(*UserCreationResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *UserCreationResult")
	}

	// 调用邮件服务发送验证邮件
	result, err := s.service.SendVerificationEmail(ctx, userResult.UserID, userResult.Email, userResult.Username)
	if err != nil {
		return nil, fmt.Errorf("发送验证邮件失败: %w", err)
	}

	return result, nil
}

// Compensate 补偿操作：取消验证邮件
func (s *SendVerificationEmailStep) Compensate(ctx context.Context, data interface{}) error {
	result, ok := data.(*EmailVerificationResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *EmailVerificationResult")
	}

	// 调用邮件服务取消验证
	// 注意：邮件已发送无法撤回，这里主要是标记验证码为无效
	err := s.service.CancelVerificationEmail(ctx, result.VerificationID)
	if err != nil {
		// 取消验证邮件失败，记录日志但不阻止补偿流程
		// 验证码通常有自动过期机制
		return nil
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *SendVerificationEmailStep) GetTimeout() time.Duration {
	return 15 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *SendVerificationEmailStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, 2*time.Second, 15*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *SendVerificationEmailStep) IsRetryable(err error) bool {
	// 邮件服务临时不可用可重试
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

// GetMetadata 返回步骤元数据
func (s *SendVerificationEmailStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "email_verification",
		"service":   "email-service",
	}
}

// InitializeUserConfigStep 初始化用户配置步骤
type InitializeUserConfigStep struct {
	service ConfigServiceClient
}

// GetID 返回步骤ID
func (s *InitializeUserConfigStep) GetID() string {
	return "initialize-user-config"
}

// GetName 返回步骤名称
func (s *InitializeUserConfigStep) GetName() string {
	return "初始化用户配置"
}

// GetDescription 返回步骤描述
func (s *InitializeUserConfigStep) GetDescription() string {
	return "初始化用户的默认配置和偏好设置"
}

// Execute 执行初始化用户配置操作
// 输入：EmailVerificationResult（从上一步传递）
// 输出：ConfigInitResult
func (s *InitializeUserConfigStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	emailResult, ok := data.(*EmailVerificationResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *EmailVerificationResult")
	}

	// 从元数据中获取语言和时区配置
	language, _ := emailResult.Metadata["language"].(string)
	if language == "" {
		language = "en" // 默认英语
	}

	timezone, _ := emailResult.Metadata["timezone"].(string)
	if timezone == "" {
		timezone = "UTC" // 默认UTC
	}

	// 调用配置服务初始化用户配置
	result, err := s.service.InitializeUserConfig(ctx, emailResult.UserID, language, timezone)
	if err != nil {
		return nil, fmt.Errorf("初始化用户配置失败: %w", err)
	}

	return result, nil
}

// Compensate 补偿操作：删除用户配置
func (s *InitializeUserConfigStep) Compensate(ctx context.Context, data interface{}) error {
	result, ok := data.(*ConfigInitResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *ConfigInitResult")
	}

	// 调用配置服务删除用户配置
	err := s.service.DeleteUserConfig(ctx, result.UserID)
	if err != nil {
		// 配置删除失败，记录日志但不阻止补偿流程
		return nil
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *InitializeUserConfigStep) GetTimeout() time.Duration {
	return 10 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *InitializeUserConfigStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *InitializeUserConfigStep) IsRetryable(err error) bool {
	return !errors.Is(err, context.DeadlineExceeded)
}

// GetMetadata 返回步骤元数据
func (s *InitializeUserConfigStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "config_initialization",
		"service":   "config-service",
	}
}

// AllocateResourcesStep 分配资源步骤
type AllocateResourcesStep struct {
	service ResourceServiceClient
}

// GetID 返回步骤ID
func (s *AllocateResourcesStep) GetID() string {
	return "allocate-resources"
}

// GetName 返回步骤名称
func (s *AllocateResourcesStep) GetName() string {
	return "分配用户资源"
}

// GetDescription 返回步骤描述
func (s *AllocateResourcesStep) GetDescription() string {
	return "为用户分配初始资源，包括钱包、存储配额、API配额等"
}

// Execute 执行分配资源操作
// 输入：ConfigInitResult（从上一步传递）
// 输出：ResourceAllocationResult
func (s *AllocateResourcesStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	configResult, ok := data.(*ConfigInitResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *ConfigInitResult")
	}

	// 从元数据中获取订阅计划
	subscription, _ := configResult.Metadata["subscription"].(string)
	if subscription == "" {
		subscription = "free" // 默认免费计划
	}

	// 调用资源服务分配资源
	result, err := s.service.AllocateResources(ctx, configResult.UserID, subscription)
	if err != nil {
		return nil, fmt.Errorf("分配用户资源失败: %w", err)
	}

	return result, nil
}

// Compensate 补偿操作：释放用户资源
func (s *AllocateResourcesStep) Compensate(ctx context.Context, data interface{}) error {
	result, ok := data.(*ResourceAllocationResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *ResourceAllocationResult")
	}

	// 调用资源服务释放资源
	err := s.service.ReleaseResources(ctx, result.UserID)
	if err != nil {
		// 资源释放失败需要记录，可能需要手动清理
		return fmt.Errorf("释放用户资源失败: %w", err)
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *AllocateResourcesStep) GetTimeout() time.Duration {
	return 15 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *AllocateResourcesStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *AllocateResourcesStep) IsRetryable(err error) bool {
	return !errors.Is(err, context.DeadlineExceeded)
}

// GetMetadata 返回步骤元数据
func (s *AllocateResourcesStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "resource_allocation",
		"service":   "resource-service",
	}
}

// TrackRegistrationEventStep 跟踪注册事件步骤
type TrackRegistrationEventStep struct {
	service AnalyticsServiceClient
}

// GetID 返回步骤ID
func (s *TrackRegistrationEventStep) GetID() string {
	return "track-registration-event"
}

// GetName 返回步骤名称
func (s *TrackRegistrationEventStep) GetName() string {
	return "跟踪注册事件"
}

// GetDescription 返回步骤描述
func (s *TrackRegistrationEventStep) GetDescription() string {
	return "向分析系统发送用户注册完成事件，用于统计和分析"
}

// Execute 执行跟踪注册事件操作
// 输入：ResourceAllocationResult（从上一步传递）
// 输出：AnalyticsTrackingResult
func (s *TrackRegistrationEventStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	resourceResult, ok := data.(*ResourceAllocationResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *ResourceAllocationResult")
	}

	// 从元数据中获取原始注册数据
	registrationData, ok := resourceResult.Metadata["registration_data"].(*UserRegistrationData)
	if !ok {
		// 如果元数据中没有注册数据，创建一个最小化的数据对象
		registrationData = &UserRegistrationData{
			Metadata: map[string]interface{}{
				"user_id": resourceResult.UserID,
			},
		}
	}

	// 调用分析服务跟踪事件
	result, err := s.service.TrackRegistrationEvent(ctx, resourceResult.UserID, registrationData)
	if err != nil {
		return nil, fmt.Errorf("跟踪注册事件失败: %w", err)
	}

	return result, nil
}

// Compensate 补偿操作：删除跟踪事件
func (s *TrackRegistrationEventStep) Compensate(ctx context.Context, data interface{}) error {
	result, ok := data.(*AnalyticsTrackingResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *AnalyticsTrackingResult")
	}

	// 调用分析服务删除事件
	// 注意：分析事件通常不需要删除，这里主要是为了演示补偿机制
	err := s.service.DeleteTrackingEvent(ctx, result.EventID)
	if err != nil {
		// 分析事件删除失败不阻止补偿流程
		return nil
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *TrackRegistrationEventStep) GetTimeout() time.Duration {
	return 10 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *TrackRegistrationEventStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(2, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *TrackRegistrationEventStep) IsRetryable(err error) bool {
	// 分析服务失败可重试，但不是关键步骤
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

// GetMetadata 返回步骤元数据
func (s *TrackRegistrationEventStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "analytics_tracking",
		"service":   "analytics-service",
		"critical":  false, // 非关键步骤
	}
}

// ==========================
// Saga 定义
// ==========================

// UserRegistrationSagaDefinition 用户注册 Saga 定义
type UserRegistrationSagaDefinition struct {
	id                   string
	name                 string
	description          string
	steps                []saga.SagaStep
	timeout              time.Duration
	retryPolicy          saga.RetryPolicy
	compensationStrategy saga.CompensationStrategy
	metadata             map[string]interface{}
}

// GetID 返回 Saga 定义ID
func (d *UserRegistrationSagaDefinition) GetID() string {
	return d.id
}

// GetName 返回 Saga 名称
func (d *UserRegistrationSagaDefinition) GetName() string {
	return d.name
}

// GetDescription 返回 Saga 描述
func (d *UserRegistrationSagaDefinition) GetDescription() string {
	return d.description
}

// GetSteps 返回所有步骤
func (d *UserRegistrationSagaDefinition) GetSteps() []saga.SagaStep {
	return d.steps
}

// GetTimeout 返回超时时间
func (d *UserRegistrationSagaDefinition) GetTimeout() time.Duration {
	return d.timeout
}

// GetRetryPolicy 返回重试策略
func (d *UserRegistrationSagaDefinition) GetRetryPolicy() saga.RetryPolicy {
	return d.retryPolicy
}

// GetCompensationStrategy 返回补偿策略
func (d *UserRegistrationSagaDefinition) GetCompensationStrategy() saga.CompensationStrategy {
	return d.compensationStrategy
}

// GetMetadata 返回元数据
func (d *UserRegistrationSagaDefinition) GetMetadata() map[string]interface{} {
	return d.metadata
}

// Validate 验证 Saga 定义的有效性
func (d *UserRegistrationSagaDefinition) Validate() error {
	if d.id == "" {
		return errors.New("saga ID 不能为空")
	}
	if d.name == "" {
		return errors.New("saga 名称不能为空")
	}
	if len(d.steps) == 0 {
		return errors.New("至少需要一个步骤")
	}
	if d.timeout <= 0 {
		return errors.New("超时时间必须大于0")
	}
	return nil
}

// ==========================
// 工厂函数
// ==========================

// NewUserRegistrationSaga 创建用户注册 Saga 定义
func NewUserRegistrationSaga(
	userService UserServiceClient,
	emailService EmailServiceClient,
	configService ConfigServiceClient,
	resourceService ResourceServiceClient,
	analyticsService AnalyticsServiceClient,
) *UserRegistrationSagaDefinition {
	// 创建步骤
	steps := []saga.SagaStep{
		&CreateUserAccountStep{service: userService},
		&SendVerificationEmailStep{service: emailService},
		&InitializeUserConfigStep{service: configService},
		&AllocateResourcesStep{service: resourceService},
		&TrackRegistrationEventStep{service: analyticsService},
	}

	return &UserRegistrationSagaDefinition{
		id:          "user-registration-saga",
		name:        "用户注册Saga",
		description: "处理用户注册的完整流程：创建账户→发送验证邮件→初始化配置→分配资源→跟踪事件",
		steps:       steps,
		timeout:     3 * time.Minute, // 整个流程3分钟超时
		retryPolicy: saga.NewExponentialBackoffRetryPolicy(
			3,              // 最多重试3次
			1*time.Second,  // 初始延迟1秒
			30*time.Second, // 最大延迟30秒
		),
		compensationStrategy: saga.NewSequentialCompensationStrategy(
			2 * time.Minute, // 补偿超时2分钟
		),
		metadata: map[string]interface{}{
			"saga_type":       "user_registration",
			"business_domain": "user-management",
			"version":         "1.0.0",
			"author":          "swit-team",
			"created_at":      time.Now().Format(time.RFC3339),
		},
	}
}

// ==========================
// 辅助函数
// ==========================

// isUserNotFoundError 判断错误是否为"用户不存在"错误
// 这个函数用于区分真正的"用户不存在"和其他类型的错误（如网络错误、数据库错误）
func isUserNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// 检查常见的"用户不存在"错误模式
	errMsg := err.Error()
	return containsSubstring(errMsg, "user not found") ||
		containsSubstring(errMsg, "not found") ||
		containsSubstring(errMsg, "no rows") ||
		containsSubstring(errMsg, "does not exist")
}

// isValidEmail 验证邮箱地址格式
func isValidEmail(email string) bool {
	// 简单的邮箱正则验证
	pattern := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// isValidUsername 验证用户名格式
func isValidUsername(username string) bool {
	// 用户名只能包含字母、数字、下划线和连字符
	pattern := `^[a-zA-Z0-9_\-]+$`
	matched, _ := regexp.MatchString(pattern, username)
	return matched
}

// containsSubstring 检查字符串是否包含子串
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
