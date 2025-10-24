# 用户注册 Saga 示例文档

## 概述

用户注册 Saga（`user_registration_saga.go`）展示了如何使用 Swit 框架处理用户注册和初始化的完整流程。这是一个典型的用户生命周期管理场景，涉及账户创建、邮件验证、配置初始化和资源分配等多个步骤。

## 业务场景

### 适用场景

- Web应用的用户注册流程
- SaaS平台的租户创建
- 企业系统的员工入职
- 教育平台的学生注册
- 游戏平台的账号创建

### 业务流程

```
用户注册 → 创建账户 → 发送验证邮件 → 初始化配置 → 分配资源配额 → 发送欢迎邮件
```

**关键特点**：
- 多步骤初始化
- 邮件验证机制
- 资源配额管理
- 可回滚的账户创建

## 架构设计

### 服务依赖关系

```
┌─────────────────┐
│  Saga           │
│  Coordinator    │
└────────┬────────┘
         │
    ┌────┴──────────┬──────────────┬──────────────┬──────────────┐
    │               │              │              │              │
┌───▼──────────┐ ┌──▼────────┐ ┌──▼──────────┐ ┌──▼──────────┐ ┌──▼──────────┐
│ 用户服务     │ │ 邮件服务  │ │ 配置服务    │ │ 配额服务    │ │ 通知服务    │
│(User)        │ │(Email)    │ │(Config)     │ │(Quota)      │ │(Notify)     │
└──────────────┘ └───────────┘ └─────────────┘ └─────────────┘ └─────────────┘
```

### 步骤流程图

```
┌──────────────────────────────────────────────────────────┐
│                  用户注册 Saga                            │
└──────────────────────────────────────────────────────────┘

步骤 1: CreateUserStep (创建用户账户)
  ├─ Execute: 在系统中创建用户记录
  └─ Compensate: 删除用户账户

步骤 2: SendVerificationEmailStep (发送验证邮件)
  ├─ Execute: 发送邮箱验证邮件
  └─ Compensate: 标记验证无效

步骤 3: InitializeConfigStep (初始化用户配置)
  ├─ Execute: 创建用户的默认配置
  └─ Compensate: 删除用户配置

步骤 4: AllocateQuotaStep (分配资源配额)
  ├─ Execute: 为用户分配资源配额
  └─ Compensate: 释放已分配的配额

步骤 5: SendWelcomeEmailStep (发送欢迎邮件)
  ├─ Execute: 发送欢迎邮件
  └─ Compensate: 无需补偿
```

### 数据流转

```
UserRegistrationData (输入)
    │
    ├─> CreateUserStep
    │     └─> UserCreationResult
    │           │
    │           ├─> SendVerificationEmailStep
    │           │     └─> EmailVerificationResult
    │           │           │
    │           │           ├─> InitializeConfigStep
    │           │           │     └─> ConfigInitResult
    │           │           │           │
    │           │           │           ├─> AllocateQuotaStep
    │           │           │           │     └─> QuotaAllocationResult
    │           │           │           │           │
    │           │           │           │           └─> SendWelcomeEmailStep
    │           │           │           │                 └─> WelcomeEmailResult (输出)
    │           │           │           │
    │           │           │           └─ (失败) ─> 补偿...
    │           │           │
    │           │           └─ (失败) ─> 补偿 InitializeConfigStep
    │           │
    │           └─ (失败) ─> 补偿 SendVerificationEmailStep + CreateUserStep
    │
    └─ (失败) ─> 补偿 CreateUserStep
```

## 数据结构

### 输入数据

#### UserRegistrationData

```go
type UserRegistrationData struct {
    // 基本信息
    Email     string      // 邮箱地址
    Username  string      // 用户名
    Password  string      // 密码（已加密）
    FirstName string      // 名字
    LastName  string      // 姓氏
    Phone     string      // 电话号码（可选）
    
    // 注册来源
    Source       string   // 注册来源（web、mobile、api）
    ReferralCode string   // 推荐码（可选）
    UtmSource    string   // UTM 来源（可选）
    
    // 配置选项
    Language     string   // 首选语言
    Timezone     string   // 时区
    Subscription string   // 订阅计划（free、premium、enterprise）
    
    // 同意条款
    AcceptedTOS bool      // 是否接受服务条款
    AcceptedAt  time.Time // 接受时间
    
    // 元数据
    Metadata map[string]interface{} // 额外元数据
}
```

**重要字段**：
- `Email`: 必填，用作登录账号和通知接收地址
- `Username`: 必填，用户的唯一标识
- `Password`: 必填，已经过加密处理
- `AcceptedTOS`: 必填，用户必须同意服务条款
- `Subscription`: 决定用户的资源配额

### 步骤结果

#### UserCreationResult

```go
type UserCreationResult struct {
    UserID       string                 // 用户ID
    Username     string                 // 用户名
    Email        string                 // 邮箱地址
    Status       string                 // 用户状态（pending、active、suspended）
    CreatedAt    time.Time              // 创建时间
    ActivationID string                 // 激活ID（用于邮件验证）
    Metadata     map[string]interface{} // 元数据
}
```

#### EmailVerificationResult

```go
type EmailVerificationResult struct {
    VerificationID   string                 // 验证ID
    UserID           string                 // 用户ID
    Email            string                 // 邮箱地址
    VerificationCode string                 // 验证码
    SentAt           time.Time              // 发送时间
    ExpiresAt        time.Time              // 过期时间
    Provider         string                 // 邮件服务提供商
    Metadata         map[string]interface{} // 元数据
}
```

#### ConfigInitResult

```go
type ConfigInitResult struct {
    UserID        string                 // 用户ID
    ConfigID      string                 // 配置ID
    Settings      map[string]interface{} // 用户设置
    Preferences   map[string]interface{} // 用户偏好
    InitializedAt time.Time              // 初始化时间
    Metadata      map[string]interface{} // 元数据
}
```

#### QuotaAllocationResult

```go
type QuotaAllocationResult struct {
    UserID       string                 // 用户ID
    QuotaID      string                 // 配额ID
    Storage      int64                  // 存储空间（字节）
    Bandwidth    int64                  // 带宽（字节/月）
    APIRequests  int                    // API 请求数/天
    AllocatedAt  time.Time              // 分配时间
    ExpiresAt    time.Time              // 过期时间
    Metadata     map[string]interface{} // 元数据
}
```

## 步骤详解

### 步骤 1: CreateUserStep（创建用户账户）

#### 功能描述

在系统中创建用户记录，初始状态为"待验证"。

#### 验证规则

1. **邮箱验证**：格式正确且未被注册
2. **用户名验证**：长度、字符要求，未被占用
3. **密码强度**：满足安全要求
4. **必填字段**：确保所有必填字段已提供
5. **服务条款**：用户必须同意服务条款

#### 执行逻辑

```go
func (s *CreateUserStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    regData := data.(*UserRegistrationData)
    
    // 验证数据
    if err := s.validateRegistrationData(regData); err != nil {
        return nil, fmt.Errorf("注册数据验证失败: %w", err)
    }
    
    // 检查邮箱是否已注册
    exists, err := s.service.CheckEmailExists(ctx, regData.Email)
    if err != nil {
        return nil, err
    }
    if exists {
        return nil, &EmailExistsError{Email: regData.Email}
    }
    
    // 检查用户名是否已占用
    exists, err = s.service.CheckUsernameExists(ctx, regData.Username)
    if err != nil {
        return nil, err
    }
    if exists {
        return nil, &UsernameExistsError{Username: regData.Username}
    }
    
    // 创建用户
    result, err := s.service.CreateUser(ctx, regData)
    if err != nil {
        return nil, fmt.Errorf("创建用户失败: %w", err)
    }
    
    return result, nil
}
```

#### 补偿逻辑

```go
func (s *CreateUserStep) Compensate(ctx context.Context, data interface{}) error {
    result := data.(*UserCreationResult)
    
    // 删除用户账户
    err := s.service.DeleteUser(ctx, result.UserID)
    if err != nil {
        return fmt.Errorf("删除用户失败: %w", err)
    }
    
    return nil
}
```

#### 配置参数

- **超时时间**: 10秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 网络错误可重试，验证错误不可重试

### 步骤 2: SendVerificationEmailStep（发送验证邮件）

#### 功能描述

向用户邮箱发送验证邮件，包含验证链接或验证码。

#### 邮件内容

- 验证链接（包含验证码）
- 验证码（6位数字）
- 过期时间（通常24小时）
- 帮助信息

#### 执行逻辑

```go
func (s *SendVerificationEmailStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    userResult := data.(*UserCreationResult)
    
    // 生成验证码
    verificationCode := generateVerificationCode()
    
    // 生成验证链接
    verificationLink := fmt.Sprintf("https://app.example.com/verify?code=%s&user=%s",
        verificationCode, userResult.UserID)
    
    // 发送邮件
    err := s.service.SendVerificationEmail(ctx, &VerificationEmailRequest{
        To:               userResult.Email,
        Username:         userResult.Username,
        VerificationCode: verificationCode,
        VerificationLink: verificationLink,
        ExpiresIn:        24 * time.Hour,
    })
    
    if err != nil {
        return nil, fmt.Errorf("发送验证邮件失败: %w", err)
    }
    
    return &EmailVerificationResult{
        VerificationID:   generateVerificationID(),
        UserID:           userResult.UserID,
        Email:            userResult.Email,
        VerificationCode: verificationCode,
        SentAt:           time.Now(),
        ExpiresAt:        time.Now().Add(24 * time.Hour),
        Provider:         "sendgrid", // 或其他邮件服务提供商
    }, nil
}
```

#### 补偿逻辑

```go
func (s *SendVerificationEmailStep) Compensate(ctx context.Context, data interface{}) error {
    result := data.(*EmailVerificationResult)
    
    // 标记验证码为无效
    err := s.service.InvalidateVerification(ctx, result.VerificationID)
    if err != nil {
        // 邮件已发送，无法撤回，但可以标记为无效
        log.Printf("标记验证无效失败: %v", err)
    }
    
    return nil
}
```

#### 配置参数

- **超时时间**: 15秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 所有错误都可重试

### 步骤 3: InitializeConfigStep（初始化用户配置）

#### 功能描述

为用户创建默认配置和偏好设置。

#### 默认配置

```go
func getDefaultConfig(regData *UserRegistrationData) map[string]interface{} {
    return map[string]interface{}{
        "language":     regData.Language,
        "timezone":     regData.Timezone,
        "theme":        "light",
        "notifications": map[string]bool{
            "email":    true,
            "push":     false,
            "sms":      false,
        },
        "privacy": map[string]bool{
            "profile_visible": true,
            "search_visible":  true,
        },
    }
}
```

#### 配置参数

- **超时时间**: 10秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 所有错误都可重试

### 步骤 4: AllocateQuotaStep（分配资源配额）

#### 功能描述

根据订阅计划为用户分配资源配额。

#### 配额策略

```go
func getQuotaBySubscription(subscription string) *QuotaConfig {
    quotas := map[string]*QuotaConfig{
        "free": {
            Storage:     1 * GB,
            Bandwidth:   10 * GB,
            APIRequests: 1000,
        },
        "premium": {
            Storage:     100 * GB,
            Bandwidth:   1 * TB,
            APIRequests: 100000,
        },
        "enterprise": {
            Storage:     1 * TB,
            Bandwidth:   10 * TB,
            APIRequests: 1000000,
        },
    }
    return quotas[subscription]
}
```

#### 补偿逻辑

```go
func (s *AllocateQuotaStep) Compensate(ctx context.Context, data interface{}) error {
    result := data.(*QuotaAllocationResult)
    
    // 释放配额
    err := s.service.ReleaseQuota(ctx, result.QuotaID)
    if err != nil {
        return fmt.Errorf("释放配额失败: %w", err)
    }
    
    return nil
}
```

#### 配置参数

- **超时时间**: 10秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 所有错误都可重试

### 步骤 5: SendWelcomeEmailStep（发送欢迎邮件）

#### 功能描述

向用户发送欢迎邮件，介绍产品功能和快速开始指南。

#### 邮件内容

- 欢迎语
- 产品功能介绍
- 快速开始指南
- 帮助资源链接
- 联系方式

#### 补偿逻辑

```go
func (s *SendWelcomeEmailStep) Compensate(ctx context.Context, data interface{}) error {
    // 欢迎邮件已发送，无法撤回，无需补偿
    return nil
}
```

#### 配置参数

- **超时时间**: 15秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 所有错误都可重试

## 使用示例

### 基本用法

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/innovationmech/swit/pkg/saga"
    "github.com/innovationmech/swit/pkg/saga/coordinator"
    "github.com/innovationmech/swit/pkg/saga/examples"
)

func main() {
    // 创建 Coordinator
    sagaCoordinator := setupCoordinator()
    defer sagaCoordinator.Close()
    
    // 创建服务客户端
    userService := NewUserService()
    emailService := NewEmailService()
    configService := NewConfigService()
    quotaService := NewQuotaService()
    
    // 创建 Saga 定义
    sagaDef := examples.NewUserRegistrationSaga(
        userService,
        emailService,
        configService,
        quotaService,
    )
    
    // 准备注册数据
    registrationData := &examples.UserRegistrationData{
        Email:        "user@example.com",
        Username:     "newuser",
        Password:     hashPassword("SecurePassword123!"),
        FirstName:    "三",
        LastName:     "张",
        Phone:        "13800138000",
        Source:       "web",
        Language:     "zh-CN",
        Timezone:     "Asia/Shanghai",
        Subscription: "free",
        AcceptedTOS:  true,
        AcceptedAt:   time.Now(),
    }
    
    // 启动 Saga
    ctx := context.Background()
    instance, err := sagaCoordinator.StartSaga(ctx, sagaDef, registrationData)
    if err != nil {
        log.Fatalf("启动 Saga 失败: %v", err)
    }
    
    log.Printf("用户注册 Saga 已启动，实例 ID: %s", instance.GetID())
    
    // 监控执行结果
    time.Sleep(10 * time.Second)
    
    finalInstance, _ := sagaCoordinator.GetSagaInstance(instance.GetID())
    log.Printf("用户注册最终状态: %s", finalInstance.GetStatus())
}
```

### 处理邮箱已存在

```go
registrationData := &examples.UserRegistrationData{
    Email:    "existing@example.com", // 已存在的邮箱
    Username: "newuser",
    // ... 其他字段
}

instance, err := sagaCoordinator.StartSaga(ctx, sagaDef, registrationData)
if err != nil {
    if emailErr, ok := err.(*EmailExistsError); ok {
        log.Printf("邮箱已存在: %s", emailErr.Email)
    }
}
```

### 不同订阅级别

```go
// 免费用户
freeUser := &examples.UserRegistrationData{
    // ... 基本字段
    Subscription: "free",
}

// 高级用户
premiumUser := &examples.UserRegistrationData{
    // ... 基本字段
    Subscription: "premium",
}

// 企业用户
enterpriseUser := &examples.UserRegistrationData{
    // ... 基本字段
    Subscription: "enterprise",
}
```

## 最佳实践

### 1. 密码安全

```go
import "golang.org/x/crypto/bcrypt"

func hashPassword(password string) string {
    hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(hash)
}

func validatePasswordStrength(password string) error {
    if len(password) < 8 {
        return errors.New("密码长度至少8位")
    }
    // 检查是否包含数字、大小写字母、特殊字符
    // ...
}
```

### 2. 邮箱验证

```go
import "net/mail"

func validateEmail(email string) error {
    _, err := mail.ParseAddress(email)
    return err
}
```

### 3. 用户名规则

```go
import "regexp"

func validateUsername(username string) error {
    // 长度限制
    if len(username) < 3 || len(username) > 20 {
        return errors.New("用户名长度必须在3-20个字符之间")
    }
    
    // 只允许字母、数字、下划线
    matched, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", username)
    if !matched {
        return errors.New("用户名只能包含字母、数字和下划线")
    }
    
    return nil
}
```

### 4. 监控注册转化率

```go
metrics := &RegistrationMetrics{
    RegistrationAttempts: prometheus.NewCounter(...),
    SuccessfulRegistrations: prometheus.NewCounter(...),
    FailedRegistrations: prometheus.NewCounter(...),
    EmailVerificationRate: prometheus.NewGauge(...),
}
```

## 测试

### 运行测试

```bash
# 运行用户注册测试
go test -v -run TestUserRegistrationSaga

# 测试邮箱已存在场景
go test -v -run TestUserRegistrationSaga/EmailExists

# 测试补偿场景
go test -v -run TestUserRegistrationSaga/Compensation
```

### 测试覆盖场景

1. ✅ 正常注册流程
2. ✅ 邮箱已存在
3. ✅ 用户名已占用
4. ✅ 密码强度不足
5. ✅ 邮件发送失败
6. ✅ 配额分配失败
7. ✅ 补偿场景

## 生产环境配置

```yaml
saga:
  user_registration:
    timeout: 2m
    
    retry:
      max_attempts: 3
      initial_interval: 1s
      max_interval: 30s
    
    compensation:
      timeout: 1m
      retry_attempts: 3
    
    steps:
      create_user:
        timeout: 10s
      send_verification_email:
        timeout: 15s
        retry_attempts: 3
      initialize_config:
        timeout: 10s
      allocate_quota:
        timeout: 10s
      send_welcome_email:
        timeout: 15s
        retry_attempts: 3
```

## 常见问题

### Q1: 如何处理社交账号登录（OAuth）？

**A**: 创建一个单独的 `OAuthRegistrationSaga`，跳过密码设置和邮箱验证步骤。

### Q2: 如何实现推荐奖励？

**A**: 在 `AllocateQuotaStep` 中检查推荐码，为推荐人和新用户增加额外配额。

### Q3: 用户长时间不验证邮箱怎么办？

**A**: 后台定期扫描未验证的账户，超过期限（如7天）自动删除。

## 相关资源

- [Saga 用户指南](../../../../docs/saga-user-guide.md)
- [架构设计文档](architecture.md)
- [订单处理 Saga](order_saga.md)

