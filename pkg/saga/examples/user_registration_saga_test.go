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

package examples

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// ==========================
// Mock 服务实现
// ==========================

// mockUserService 模拟用户服务
type mockUserService struct {
	createUserFunc        func(ctx context.Context, data *UserRegistrationData) (*UserCreationResult, error)
	deleteUserFunc        func(ctx context.Context, userID string, reason string) error
	getUserByEmailFunc    func(ctx context.Context, email string) (*UserCreationResult, error)
	getUserByUsernameFunc func(ctx context.Context, username string) (*UserCreationResult, error)
}

func (m *mockUserService) CreateUser(ctx context.Context, data *UserRegistrationData) (*UserCreationResult, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, data)
	}
	return &UserCreationResult{
		UserID:       "user-123",
		Username:     data.Username,
		Email:        data.Email,
		Status:       "pending",
		CreatedAt:    time.Now(),
		ActivationID: "act-123",
		Metadata:     map[string]interface{}{},
	}, nil
}

func (m *mockUserService) DeleteUser(ctx context.Context, userID string, reason string) error {
	if m.deleteUserFunc != nil {
		return m.deleteUserFunc(ctx, userID, reason)
	}
	return nil
}

func (m *mockUserService) GetUserByEmail(ctx context.Context, email string) (*UserCreationResult, error) {
	if m.getUserByEmailFunc != nil {
		return m.getUserByEmailFunc(ctx, email)
	}
	return nil, errors.New("user not found")
}

func (m *mockUserService) GetUserByUsername(ctx context.Context, username string) (*UserCreationResult, error) {
	if m.getUserByUsernameFunc != nil {
		return m.getUserByUsernameFunc(ctx, username)
	}
	return nil, errors.New("user not found")
}

// mockEmailService 模拟邮件服务
type mockEmailService struct {
	sendVerificationEmailFunc   func(ctx context.Context, userID string, email string, username string) (*EmailVerificationResult, error)
	cancelVerificationEmailFunc func(ctx context.Context, verificationID string) error
	resendVerificationEmailFunc func(ctx context.Context, verificationID string) error
}

func (m *mockEmailService) SendVerificationEmail(ctx context.Context, userID string, email string, username string) (*EmailVerificationResult, error) {
	if m.sendVerificationEmailFunc != nil {
		return m.sendVerificationEmailFunc(ctx, userID, email, username)
	}
	return &EmailVerificationResult{
		VerificationID:   "verify-123",
		UserID:           userID,
		Email:            email,
		VerificationCode: "ABCD1234",
		SentAt:           time.Now(),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		Provider:         "sendgrid",
		Metadata:         map[string]interface{}{},
	}, nil
}

func (m *mockEmailService) CancelVerificationEmail(ctx context.Context, verificationID string) error {
	if m.cancelVerificationEmailFunc != nil {
		return m.cancelVerificationEmailFunc(ctx, verificationID)
	}
	return nil
}

func (m *mockEmailService) ResendVerificationEmail(ctx context.Context, verificationID string) error {
	if m.resendVerificationEmailFunc != nil {
		return m.resendVerificationEmailFunc(ctx, verificationID)
	}
	return nil
}

// mockConfigService 模拟配置服务
type mockConfigService struct {
	initializeUserConfigFunc func(ctx context.Context, userID string, language string, timezone string) (*ConfigInitResult, error)
	deleteUserConfigFunc     func(ctx context.Context, userID string) error
	updateUserConfigFunc     func(ctx context.Context, userID string, settings map[string]interface{}) error
}

func (m *mockConfigService) InitializeUserConfig(ctx context.Context, userID string, language string, timezone string) (*ConfigInitResult, error) {
	if m.initializeUserConfigFunc != nil {
		return m.initializeUserConfigFunc(ctx, userID, language, timezone)
	}
	return &ConfigInitResult{
		UserID:        userID,
		ConfigID:      "config-123",
		Settings:      map[string]interface{}{"language": language, "timezone": timezone},
		Preferences:   map[string]interface{}{},
		InitializedAt: time.Now(),
		Metadata:      map[string]interface{}{},
	}, nil
}

func (m *mockConfigService) DeleteUserConfig(ctx context.Context, userID string) error {
	if m.deleteUserConfigFunc != nil {
		return m.deleteUserConfigFunc(ctx, userID)
	}
	return nil
}

func (m *mockConfigService) UpdateUserConfig(ctx context.Context, userID string, settings map[string]interface{}) error {
	if m.updateUserConfigFunc != nil {
		return m.updateUserConfigFunc(ctx, userID, settings)
	}
	return nil
}

// mockResourceService 模拟资源服务
type mockResourceService struct {
	allocateResourcesFunc func(ctx context.Context, userID string, subscription string) (*ResourceAllocationResult, error)
	releaseResourcesFunc  func(ctx context.Context, userID string) error
	getResourceQuotaFunc  func(ctx context.Context, subscription string) (*ResourceAllocationResult, error)
}

func (m *mockResourceService) AllocateResources(ctx context.Context, userID string, subscription string) (*ResourceAllocationResult, error) {
	if m.allocateResourcesFunc != nil {
		return m.allocateResourcesFunc(ctx, userID, subscription)
	}
	return &ResourceAllocationResult{
		UserID:         userID,
		WalletID:       "wallet-123",
		StorageQuota:   1024 * 1024 * 1024, // 1GB
		APIQuota:       1000,
		ComputeQuota:   100,
		AllocatedAt:    time.Now(),
		SubscriptionID: "sub-123",
		Metadata:       map[string]interface{}{},
	}, nil
}

func (m *mockResourceService) ReleaseResources(ctx context.Context, userID string) error {
	if m.releaseResourcesFunc != nil {
		return m.releaseResourcesFunc(ctx, userID)
	}
	return nil
}

func (m *mockResourceService) GetResourceQuota(ctx context.Context, subscription string) (*ResourceAllocationResult, error) {
	if m.getResourceQuotaFunc != nil {
		return m.getResourceQuotaFunc(ctx, subscription)
	}
	return &ResourceAllocationResult{
		StorageQuota: 1024 * 1024 * 1024,
		APIQuota:     1000,
		ComputeQuota: 100,
		Metadata:     map[string]interface{}{},
	}, nil
}

// mockAnalyticsService 模拟分析服务
type mockAnalyticsService struct {
	trackRegistrationEventFunc func(ctx context.Context, userID string, data *UserRegistrationData) (*AnalyticsTrackingResult, error)
	deleteTrackingEventFunc    func(ctx context.Context, eventID string) error
}

func (m *mockAnalyticsService) TrackRegistrationEvent(ctx context.Context, userID string, data *UserRegistrationData) (*AnalyticsTrackingResult, error) {
	if m.trackRegistrationEventFunc != nil {
		return m.trackRegistrationEventFunc(ctx, userID, data)
	}
	return &AnalyticsTrackingResult{
		UserID:     userID,
		EventID:    "event-123",
		EventType:  "user_registered",
		TrackedAt:  time.Now(),
		SessionID:  "session-123",
		DeviceInfo: map[string]interface{}{},
		Metadata:   map[string]interface{}{},
	}, nil
}

func (m *mockAnalyticsService) DeleteTrackingEvent(ctx context.Context, eventID string) error {
	if m.deleteTrackingEventFunc != nil {
		return m.deleteTrackingEventFunc(ctx, eventID)
	}
	return nil
}

// ==========================
// 辅助函数
// ==========================

// createTestRegistrationData 创建测试用的注册数据
func createTestRegistrationData() *UserRegistrationData {
	return &UserRegistrationData{
		Email:        "test@example.com",
		Username:     "testuser",
		Password:     "Password123!",
		FirstName:    "Test",
		LastName:     "User",
		Phone:        "+1234567890",
		Source:       "web",
		Language:     "en",
		Timezone:     "UTC",
		Subscription: "free",
		AcceptedTOS:  true,
		AcceptedAt:   time.Now(),
		Metadata:     map[string]interface{}{},
	}
}

// ==========================
// 单元测试
// ==========================

// TestCreateUserAccountStep_Execute_Success 测试创建用户账户成功场景
func TestCreateUserAccountStep_Execute_Success(t *testing.T) {
	// 准备
	mockService := &mockUserService{}
	step := &CreateUserAccountStep{service: mockService}
	data := createTestRegistrationData()
	ctx := context.Background()

	// 执行
	result, err := step.Execute(ctx, data)

	// 验证
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	userResult, ok := result.(*UserCreationResult)
	if !ok {
		t.Fatal("Expected *UserCreationResult")
	}
	if userResult.UserID == "" {
		t.Error("Expected non-empty user ID")
	}
	if userResult.Email != data.Email {
		t.Errorf("Expected email %s, got %s", data.Email, userResult.Email)
	}
}

// TestCreateUserAccountStep_Execute_ValidationError 测试数据验证失败场景
func TestCreateUserAccountStep_Execute_ValidationError(t *testing.T) {
	mockService := &mockUserService{}
	step := &CreateUserAccountStep{service: mockService}
	ctx := context.Background()

	tests := []struct {
		name        string
		data        *UserRegistrationData
		expectedErr string
	}{
		{
			name: "Empty email",
			data: &UserRegistrationData{
				Username:    "testuser",
				Password:    "Password123!",
				FirstName:   "Test",
				LastName:    "User",
				AcceptedTOS: true,
			},
			expectedErr: "邮箱地址不能为空",
		},
		{
			name: "Invalid email format",
			data: &UserRegistrationData{
				Email:       "invalid-email",
				Username:    "testuser",
				Password:    "Password123!",
				FirstName:   "Test",
				LastName:    "User",
				AcceptedTOS: true,
			},
			expectedErr: "邮箱地址格式无效",
		},
		{
			name: "Empty username",
			data: &UserRegistrationData{
				Email:       "test@example.com",
				Password:    "Password123!",
				FirstName:   "Test",
				LastName:    "User",
				AcceptedTOS: true,
			},
			expectedErr: "用户名不能为空",
		},
		{
			name: "Short username",
			data: &UserRegistrationData{
				Email:       "test@example.com",
				Username:    "ab",
				Password:    "Password123!",
				FirstName:   "Test",
				LastName:    "User",
				AcceptedTOS: true,
			},
			expectedErr: "用户名长度必须在3-30个字符之间",
		},
		{
			name: "Empty password",
			data: &UserRegistrationData{
				Email:       "test@example.com",
				Username:    "testuser",
				FirstName:   "Test",
				LastName:    "User",
				AcceptedTOS: true,
			},
			expectedErr: "密码不能为空",
		},
		{
			name: "Short password",
			data: &UserRegistrationData{
				Email:       "test@example.com",
				Username:    "testuser",
				Password:    "Pass1!",
				FirstName:   "Test",
				LastName:    "User",
				AcceptedTOS: true,
			},
			expectedErr: "密码长度至少8个字符",
		},
		{
			name: "TOS not accepted",
			data: &UserRegistrationData{
				Email:       "test@example.com",
				Username:    "testuser",
				Password:    "Password123!",
				FirstName:   "Test",
				LastName:    "User",
				AcceptedTOS: false,
			},
			expectedErr: "必须接受服务条款",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := step.Execute(ctx, tt.data)
			if err == nil {
				t.Error("Expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

// TestCreateUserAccountStep_Execute_EmailExists 测试邮箱已存在场景
func TestCreateUserAccountStep_Execute_EmailExists(t *testing.T) {
	mockService := &mockUserService{
		getUserByEmailFunc: func(ctx context.Context, email string) (*UserCreationResult, error) {
			return &UserCreationResult{
				UserID: "existing-user",
				Email:  email,
			}, nil
		},
	}
	step := &CreateUserAccountStep{service: mockService}
	data := createTestRegistrationData()
	ctx := context.Background()

	_, err := step.Execute(ctx, data)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "邮箱已被注册") {
		t.Errorf("Expected '邮箱已被注册' error, got: %v", err)
	}
}

// TestCreateUserAccountStep_Execute_UsernameExists 测试用户名已存在场景
func TestCreateUserAccountStep_Execute_UsernameExists(t *testing.T) {
	mockService := &mockUserService{
		getUserByUsernameFunc: func(ctx context.Context, username string) (*UserCreationResult, error) {
			return &UserCreationResult{
				UserID:   "existing-user",
				Username: username,
			}, nil
		},
	}
	step := &CreateUserAccountStep{service: mockService}
	data := createTestRegistrationData()
	ctx := context.Background()

	_, err := step.Execute(ctx, data)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "用户名已被使用") {
		t.Errorf("Expected '用户名已被使用' error, got: %v", err)
	}
}

// TestCreateUserAccountStep_Execute_EmailLookupError 测试邮箱查询失败场景
func TestCreateUserAccountStep_Execute_EmailLookupError(t *testing.T) {
	mockService := &mockUserService{
		getUserByEmailFunc: func(ctx context.Context, email string) (*UserCreationResult, error) {
			// 模拟数据库连接错误（非"用户不存在"错误）
			return nil, errors.New("database connection failed")
		},
	}
	step := &CreateUserAccountStep{service: mockService}
	data := createTestRegistrationData()
	ctx := context.Background()

	_, err := step.Execute(ctx, data)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "查询邮箱失败") {
		t.Errorf("Expected '查询邮箱失败' error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "database connection failed") {
		t.Errorf("Expected wrapped error message, got: %v", err)
	}
}

// TestCreateUserAccountStep_Execute_UsernameLookupError 测试用户名查询失败场景
func TestCreateUserAccountStep_Execute_UsernameLookupError(t *testing.T) {
	mockService := &mockUserService{
		getUserByEmailFunc: func(ctx context.Context, email string) (*UserCreationResult, error) {
			// 返回"用户不存在"错误，允许继续
			return nil, errors.New("user not found")
		},
		getUserByUsernameFunc: func(ctx context.Context, username string) (*UserCreationResult, error) {
			// 模拟网络超时错误（非"用户不存在"错误）
			return nil, errors.New("network timeout")
		},
	}
	step := &CreateUserAccountStep{service: mockService}
	data := createTestRegistrationData()
	ctx := context.Background()

	_, err := step.Execute(ctx, data)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "查询用户名失败") {
		t.Errorf("Expected '查询用户名失败' error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "network timeout") {
		t.Errorf("Expected wrapped error message, got: %v", err)
	}
}

// TestCreateUserAccountStep_Execute_UserNotFoundError 测试"用户不存在"错误被正确识别
func TestCreateUserAccountStep_Execute_UserNotFoundError(t *testing.T) {
	mockService := &mockUserService{
		getUserByEmailFunc: func(ctx context.Context, email string) (*UserCreationResult, error) {
			// 返回"用户不存在"错误，应该被识别并允许继续
			return nil, errors.New("user not found")
		},
		getUserByUsernameFunc: func(ctx context.Context, username string) (*UserCreationResult, error) {
			// 返回另一种形式的"不存在"错误
			return nil, errors.New("no rows in result set")
		},
		createUserFunc: func(ctx context.Context, data *UserRegistrationData) (*UserCreationResult, error) {
			return &UserCreationResult{
				UserID:   "new-user-123",
				Username: data.Username,
				Email:    data.Email,
				Status:   "pending",
			}, nil
		},
	}
	step := &CreateUserAccountStep{service: mockService}
	data := createTestRegistrationData()
	ctx := context.Background()

	result, err := step.Execute(ctx, data)
	if err != nil {
		t.Errorf("Expected success when user not found, got error: %v", err)
	}
	if result == nil {
		t.Error("Expected result, got nil")
	}
}

// TestCreateUserAccountStep_Compensate_Success 测试补偿操作成功场景
func TestCreateUserAccountStep_Compensate_Success(t *testing.T) {
	deleteCalled := false
	mockService := &mockUserService{
		deleteUserFunc: func(ctx context.Context, userID string, reason string) error {
			deleteCalled = true
			if userID != "user-123" {
				t.Errorf("Expected userID 'user-123', got '%s'", userID)
			}
			if reason != "registration_failed" {
				t.Errorf("Expected reason 'registration_failed', got '%s'", reason)
			}
			return nil
		},
	}
	step := &CreateUserAccountStep{service: mockService}
	result := &UserCreationResult{
		UserID: "user-123",
		Email:  "test@example.com",
	}
	ctx := context.Background()

	err := step.Compensate(ctx, result)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !deleteCalled {
		t.Error("Expected DeleteUser to be called")
	}
}

// TestSendVerificationEmailStep_Execute_Success 测试发送验证邮件成功场景
func TestSendVerificationEmailStep_Execute_Success(t *testing.T) {
	mockService := &mockEmailService{}
	step := &SendVerificationEmailStep{service: mockService}
	userResult := &UserCreationResult{
		UserID:   "user-123",
		Username: "testuser",
		Email:    "test@example.com",
	}
	ctx := context.Background()

	result, err := step.Execute(ctx, userResult)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	emailResult, ok := result.(*EmailVerificationResult)
	if !ok {
		t.Fatal("Expected *EmailVerificationResult")
	}
	if emailResult.UserID != userResult.UserID {
		t.Errorf("Expected userID %s, got %s", userResult.UserID, emailResult.UserID)
	}
	if emailResult.Email != userResult.Email {
		t.Errorf("Expected email %s, got %s", userResult.Email, emailResult.Email)
	}
}

// TestSendVerificationEmailStep_Execute_ServiceError 测试邮件服务错误场景
func TestSendVerificationEmailStep_Execute_ServiceError(t *testing.T) {
	mockService := &mockEmailService{
		sendVerificationEmailFunc: func(ctx context.Context, userID string, email string, username string) (*EmailVerificationResult, error) {
			return nil, errors.New("email service unavailable")
		},
	}
	step := &SendVerificationEmailStep{service: mockService}
	userResult := &UserCreationResult{
		UserID:   "user-123",
		Username: "testuser",
		Email:    "test@example.com",
	}
	ctx := context.Background()

	_, err := step.Execute(ctx, userResult)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "发送验证邮件失败") {
		t.Errorf("Expected '发送验证邮件失败' error, got: %v", err)
	}
}

// TestInitializeUserConfigStep_Execute_Success 测试初始化配置成功场景
func TestInitializeUserConfigStep_Execute_Success(t *testing.T) {
	mockService := &mockConfigService{}
	step := &InitializeUserConfigStep{service: mockService}
	emailResult := &EmailVerificationResult{
		UserID: "user-123",
		Email:  "test@example.com",
		Metadata: map[string]interface{}{
			"language": "en",
			"timezone": "UTC",
		},
	}
	ctx := context.Background()

	result, err := step.Execute(ctx, emailResult)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	configResult, ok := result.(*ConfigInitResult)
	if !ok {
		t.Fatal("Expected *ConfigInitResult")
	}
	if configResult.UserID != emailResult.UserID {
		t.Errorf("Expected userID %s, got %s", emailResult.UserID, configResult.UserID)
	}
}

// TestInitializeUserConfigStep_Execute_DefaultValues 测试默认值设置
func TestInitializeUserConfigStep_Execute_DefaultValues(t *testing.T) {
	var capturedLanguage, capturedTimezone string
	mockService := &mockConfigService{
		initializeUserConfigFunc: func(ctx context.Context, userID string, language string, timezone string) (*ConfigInitResult, error) {
			capturedLanguage = language
			capturedTimezone = timezone
			return &ConfigInitResult{
				UserID:   userID,
				ConfigID: "config-123",
				Settings: map[string]interface{}{
					"language": language,
					"timezone": timezone,
				},
				Metadata: map[string]interface{}{},
			}, nil
		},
	}
	step := &InitializeUserConfigStep{service: mockService}
	emailResult := &EmailVerificationResult{
		UserID:   "user-123",
		Email:    "test@example.com",
		Metadata: map[string]interface{}{}, // 没有设置 language 和 timezone
	}
	ctx := context.Background()

	_, err := step.Execute(ctx, emailResult)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if capturedLanguage != "en" {
		t.Errorf("Expected default language 'en', got '%s'", capturedLanguage)
	}
	if capturedTimezone != "UTC" {
		t.Errorf("Expected default timezone 'UTC', got '%s'", capturedTimezone)
	}
}

// TestAllocateResourcesStep_Execute_Success 测试分配资源成功场景
func TestAllocateResourcesStep_Execute_Success(t *testing.T) {
	mockService := &mockResourceService{}
	step := &AllocateResourcesStep{service: mockService}
	configResult := &ConfigInitResult{
		UserID:   "user-123",
		ConfigID: "config-123",
		Metadata: map[string]interface{}{
			"subscription": "premium",
		},
	}
	ctx := context.Background()

	result, err := step.Execute(ctx, configResult)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	resourceResult, ok := result.(*ResourceAllocationResult)
	if !ok {
		t.Fatal("Expected *ResourceAllocationResult")
	}
	if resourceResult.UserID != configResult.UserID {
		t.Errorf("Expected userID %s, got %s", configResult.UserID, resourceResult.UserID)
	}
}

// TestAllocateResourcesStep_Compensate_Success 测试释放资源成功场景
func TestAllocateResourcesStep_Compensate_Success(t *testing.T) {
	releaseCalled := false
	mockService := &mockResourceService{
		releaseResourcesFunc: func(ctx context.Context, userID string) error {
			releaseCalled = true
			if userID != "user-123" {
				t.Errorf("Expected userID 'user-123', got '%s'", userID)
			}
			return nil
		},
	}
	step := &AllocateResourcesStep{service: mockService}
	result := &ResourceAllocationResult{
		UserID:   "user-123",
		WalletID: "wallet-123",
	}
	ctx := context.Background()

	err := step.Compensate(ctx, result)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !releaseCalled {
		t.Error("Expected ReleaseResources to be called")
	}
}

// TestTrackRegistrationEventStep_Execute_Success 测试跟踪事件成功场景
func TestTrackRegistrationEventStep_Execute_Success(t *testing.T) {
	mockService := &mockAnalyticsService{}
	step := &TrackRegistrationEventStep{service: mockService}
	registrationData := createTestRegistrationData()
	resourceResult := &ResourceAllocationResult{
		UserID:   "user-123",
		WalletID: "wallet-123",
		Metadata: map[string]interface{}{
			"registration_data": registrationData,
		},
	}
	ctx := context.Background()

	result, err := step.Execute(ctx, resourceResult)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	analyticsResult, ok := result.(*AnalyticsTrackingResult)
	if !ok {
		t.Fatal("Expected *AnalyticsTrackingResult")
	}
	if analyticsResult.UserID != resourceResult.UserID {
		t.Errorf("Expected userID %s, got %s", resourceResult.UserID, analyticsResult.UserID)
	}
	if analyticsResult.EventType != "user_registered" {
		t.Errorf("Expected eventType 'user_registered', got '%s'", analyticsResult.EventType)
	}
}

// TestTrackRegistrationEventStep_Execute_NoRegistrationData 测试没有注册数据的场景
func TestTrackRegistrationEventStep_Execute_NoRegistrationData(t *testing.T) {
	mockService := &mockAnalyticsService{}
	step := &TrackRegistrationEventStep{service: mockService}
	resourceResult := &ResourceAllocationResult{
		UserID:   "user-123",
		WalletID: "wallet-123",
		Metadata: map[string]interface{}{
			// 没有 registration_data
		},
	}
	ctx := context.Background()

	result, err := step.Execute(ctx, resourceResult)
	// 即使没有注册数据，也应该成功（使用最小化数据）
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
}

// TestNewUserRegistrationSaga 测试创建 Saga 定义
func TestNewUserRegistrationSaga(t *testing.T) {
	userService := &mockUserService{}
	emailService := &mockEmailService{}
	configService := &mockConfigService{}
	resourceService := &mockResourceService{}
	analyticsService := &mockAnalyticsService{}

	saga := NewUserRegistrationSaga(
		userService,
		emailService,
		configService,
		resourceService,
		analyticsService,
	)

	// 验证基本属性
	if saga.GetID() == "" {
		t.Error("Expected non-empty saga ID")
	}
	if saga.GetName() == "" {
		t.Error("Expected non-empty saga name")
	}
	if saga.GetDescription() == "" {
		t.Error("Expected non-empty saga description")
	}

	// 验证步骤
	steps := saga.GetSteps()
	if len(steps) != 5 {
		t.Errorf("Expected 5 steps, got %d", len(steps))
	}

	// 验证步骤顺序
	expectedStepIDs := []string{
		"create-user-account",
		"send-verification-email",
		"initialize-user-config",
		"allocate-resources",
		"track-registration-event",
	}
	for i, step := range steps {
		if step.GetID() != expectedStepIDs[i] {
			t.Errorf("Expected step %d to be '%s', got '%s'", i, expectedStepIDs[i], step.GetID())
		}
	}

	// 验证超时
	if saga.GetTimeout() <= 0 {
		t.Error("Expected positive timeout")
	}

	// 验证重试策略
	if saga.GetRetryPolicy() == nil {
		t.Error("Expected retry policy to be set")
	}

	// 验证补偿策略
	if saga.GetCompensationStrategy() == nil {
		t.Error("Expected compensation strategy to be set")
	}

	// 验证元数据
	metadata := saga.GetMetadata()
	if metadata["saga_type"] != "user_registration" {
		t.Errorf("Expected saga_type 'user_registration', got '%v'", metadata["saga_type"])
	}
}

// TestUserRegistrationSagaDefinition_Validate 测试 Saga 定义验证
func TestUserRegistrationSagaDefinition_Validate(t *testing.T) {
	tests := []struct {
		name        string
		saga        *UserRegistrationSagaDefinition
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid saga",
			saga: &UserRegistrationSagaDefinition{
				id:      "test-saga",
				name:    "Test Saga",
				steps:   []saga.SagaStep{&CreateUserAccountStep{}},
				timeout: time.Minute,
			},
			expectError: false,
		},
		{
			name: "Empty ID",
			saga: &UserRegistrationSagaDefinition{
				name:    "Test Saga",
				steps:   []saga.SagaStep{&CreateUserAccountStep{}},
				timeout: time.Minute,
			},
			expectError: true,
			errorMsg:    "saga ID 不能为空",
		},
		{
			name: "Empty name",
			saga: &UserRegistrationSagaDefinition{
				id:      "test-saga",
				steps:   []saga.SagaStep{&CreateUserAccountStep{}},
				timeout: time.Minute,
			},
			expectError: true,
			errorMsg:    "saga 名称不能为空",
		},
		{
			name: "No steps",
			saga: &UserRegistrationSagaDefinition{
				id:      "test-saga",
				name:    "Test Saga",
				steps:   []saga.SagaStep{},
				timeout: time.Minute,
			},
			expectError: true,
			errorMsg:    "至少需要一个步骤",
		},
		{
			name: "Zero timeout",
			saga: &UserRegistrationSagaDefinition{
				id:      "test-saga",
				name:    "Test Saga",
				steps:   []saga.SagaStep{&CreateUserAccountStep{}},
				timeout: 0,
			},
			expectError: true,
			errorMsg:    "超时时间必须大于0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.saga.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestStepMetadata 测试各步骤的元数据
func TestStepMetadata(t *testing.T) {
	tests := []struct {
		name string
		step saga.SagaStep
	}{
		{"CreateUserAccountStep", &CreateUserAccountStep{service: &mockUserService{}}},
		{"SendVerificationEmailStep", &SendVerificationEmailStep{service: &mockEmailService{}}},
		{"InitializeUserConfigStep", &InitializeUserConfigStep{service: &mockConfigService{}}},
		{"AllocateResourcesStep", &AllocateResourcesStep{service: &mockResourceService{}}},
		{"TrackRegistrationEventStep", &TrackRegistrationEventStep{service: &mockAnalyticsService{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := tt.step.GetMetadata()
			if metadata == nil {
				t.Error("Expected non-nil metadata")
			}
			if metadata["step_type"] == "" {
				t.Error("Expected non-empty step_type in metadata")
			}
			if metadata["service"] == "" {
				t.Error("Expected non-empty service in metadata")
			}
		})
	}
}

// TestStepTimeouts 测试各步骤的超时配置
func TestStepTimeouts(t *testing.T) {
	tests := []struct {
		name       string
		step       saga.SagaStep
		minTimeout time.Duration
		maxTimeout time.Duration
	}{
		{"CreateUserAccountStep", &CreateUserAccountStep{service: &mockUserService{}}, 5 * time.Second, 30 * time.Second},
		{"SendVerificationEmailStep", &SendVerificationEmailStep{service: &mockEmailService{}}, 10 * time.Second, 30 * time.Second},
		{"InitializeUserConfigStep", &InitializeUserConfigStep{service: &mockConfigService{}}, 5 * time.Second, 30 * time.Second},
		{"AllocateResourcesStep", &AllocateResourcesStep{service: &mockResourceService{}}, 10 * time.Second, 30 * time.Second},
		{"TrackRegistrationEventStep", &TrackRegistrationEventStep{service: &mockAnalyticsService{}}, 5 * time.Second, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout := tt.step.GetTimeout()
			if timeout < tt.minTimeout {
				t.Errorf("Timeout %v is less than minimum %v", timeout, tt.minTimeout)
			}
			if timeout > tt.maxTimeout {
				t.Errorf("Timeout %v is greater than maximum %v", timeout, tt.maxTimeout)
			}
		})
	}
}

// TestStepRetryPolicies 测试各步骤的重试策略
func TestStepRetryPolicies(t *testing.T) {
	tests := []struct {
		name string
		step saga.SagaStep
	}{
		{"CreateUserAccountStep", &CreateUserAccountStep{service: &mockUserService{}}},
		{"SendVerificationEmailStep", &SendVerificationEmailStep{service: &mockEmailService{}}},
		{"InitializeUserConfigStep", &InitializeUserConfigStep{service: &mockConfigService{}}},
		{"AllocateResourcesStep", &AllocateResourcesStep{service: &mockResourceService{}}},
		{"TrackRegistrationEventStep", &TrackRegistrationEventStep{service: &mockAnalyticsService{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryPolicy := tt.step.GetRetryPolicy()
			if retryPolicy == nil {
				t.Error("Expected non-nil retry policy")
			}
		})
	}
}

// TestIsRetryable 测试各步骤的重试判断
func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		step      saga.SagaStep
		err       error
		retryable bool
	}{
		{
			name:      "CreateUserAccountStep - timeout error",
			step:      &CreateUserAccountStep{service: &mockUserService{}},
			err:       context.DeadlineExceeded,
			retryable: false,
		},
		{
			name:      "CreateUserAccountStep - email exists error",
			step:      &CreateUserAccountStep{service: &mockUserService{}},
			err:       fmt.Errorf("注册数据验证失败: 邮箱已被注册: test@example.com"),
			retryable: false,
		},
		{
			name:      "CreateUserAccountStep - network error",
			step:      &CreateUserAccountStep{service: &mockUserService{}},
			err:       fmt.Errorf("network connection failed"),
			retryable: true,
		},
		{
			name:      "SendVerificationEmailStep - timeout error",
			step:      &SendVerificationEmailStep{service: &mockEmailService{}},
			err:       context.DeadlineExceeded,
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryable := tt.step.IsRetryable(tt.err)
			if retryable != tt.retryable {
				t.Errorf("Expected retryable=%v, got %v", tt.retryable, retryable)
			}
		})
	}
}

// TestValidationFunctions 测试辅助验证函数
func TestValidationFunctions(t *testing.T) {
	t.Run("isValidEmail", func(t *testing.T) {
		tests := []struct {
			email string
			valid bool
		}{
			{"test@example.com", true},
			{"user.name+tag@example.co.uk", true},
			{"invalid-email", false},
			{"@example.com", false},
			{"test@", false},
			{"", false},
		}

		for _, tt := range tests {
			result := isValidEmail(tt.email)
			if result != tt.valid {
				t.Errorf("isValidEmail(%s) = %v, expected %v", tt.email, result, tt.valid)
			}
		}
	})

	t.Run("isValidUsername", func(t *testing.T) {
		tests := []struct {
			username string
			valid    bool
		}{
			{"testuser", true},
			{"test_user", true},
			{"test-user", true},
			{"test123", true},
			{"test user", false},
			{"test@user", false},
			{"", false},
		}

		for _, tt := range tests {
			result := isValidUsername(tt.username)
			if result != tt.valid {
				t.Errorf("isValidUsername(%s) = %v, expected %v", tt.username, result, tt.valid)
			}
		}
	})

	t.Run("isUserNotFoundError", func(t *testing.T) {
		tests := []struct {
			name     string
			err      error
			expected bool
		}{
			{"nil error", nil, false},
			{"user not found", errors.New("user not found"), true},
			{"not found", errors.New("record not found"), true},
			{"no rows", errors.New("no rows in result set"), true},
			{"does not exist", errors.New("user does not exist"), true},
			{"database error", errors.New("database connection failed"), false},
			{"network timeout", errors.New("network timeout"), false},
			{"generic error", errors.New("something went wrong"), false},
		}

		for _, tt := range tests {
			result := isUserNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("%s: isUserNotFoundError(%v) = %v, expected %v", 
					tt.name, tt.err, result, tt.expected)
			}
		}
	})
}

// TestInvalidDataTypes 测试无效数据类型
func TestInvalidDataTypes(t *testing.T) {
	ctx := context.Background()

	t.Run("CreateUserAccountStep - invalid input type", func(t *testing.T) {
		step := &CreateUserAccountStep{service: &mockUserService{}}
		_, err := step.Execute(ctx, "invalid data")
		if err == nil {
			t.Error("Expected error for invalid data type")
		}
		if !strings.Contains(err.Error(), "invalid data type") {
			t.Errorf("Expected 'invalid data type' error, got: %v", err)
		}
	})

	t.Run("CreateUserAccountStep - invalid compensation type", func(t *testing.T) {
		step := &CreateUserAccountStep{service: &mockUserService{}}
		err := step.Compensate(ctx, "invalid data")
		if err == nil {
			t.Error("Expected error for invalid compensation data type")
		}
		if !strings.Contains(err.Error(), "invalid compensation data type") {
			t.Errorf("Expected 'invalid compensation data type' error, got: %v", err)
		}
	})

	t.Run("SendVerificationEmailStep - invalid input type", func(t *testing.T) {
		step := &SendVerificationEmailStep{service: &mockEmailService{}}
		_, err := step.Execute(ctx, "invalid data")
		if err == nil {
			t.Error("Expected error for invalid data type")
		}
	})
}
