# Saga 敏感数据保护指南

## 概述

敏感数据保护机制提供了全面的数据脱敏、掩码和安全存储功能，用于保护 Saga 事务中的敏感信息，如手机号、身份证号、银行卡号、密码等。

## 核心功能

### 1. 敏感数据类型

系统支持以下敏感数据类型：

- `phone` - 手机号码
- `idcard` - 身份证号
- `bankcard` - 银行卡号
- `email` - 电子邮件
- `name` - 姓名
- `address` - 地址
- `password` - 密码
- `custom` - 自定义类型

### 2. 数据脱敏规则

#### 手机号脱敏
```go
// 输入: 13812345678
// 输出: 138****5678
// 规则: 保留前3位和后4位
```

#### 邮箱脱敏
```go
// 输入: test@example.com
// 输出: te**@e*****e.com
// 规则: 本地部分保留前2位，域名部分保留首尾
```

#### 银行卡脱敏
```go
// 输入: 1234567890123456
// 输出: 1234********3456
// 规则: 保留前4位和后4位
```

#### 身份证脱敏
```go
// 输入: 110101199001011234
// 输出: 1101**********1234
// 规则: 保留前4位和后4位
```

#### 姓名脱敏
```go
// 输入: 张三
// 输出: 张*
// 规则: 保留第一个字符
```

#### 密码脱敏
```go
// 输入: MyPassword123
// 输出: ********
// 规则: 完全掩码，固定长度
```

## 快速开始

### 1. 基本使用

```go
import (
    "github.com/innovationmech/swit/pkg/saga/security"
)

// 创建脱敏器
masker, err := security.NewMasker(nil) // 使用默认配置
if err != nil {
    log.Fatal(err)
}

// 脱敏手机号
masked := masker.MaskPhone("13812345678")
fmt.Println(masked) // 输出: 138****5678

// 脱敏邮箱
masked = masker.MaskEmail("user@example.com")
fmt.Println(masked) // 输出: us**@e*****e.com
```

### 2. 使用敏感数据标记

```go
// 创建敏感数据标记器
marker := security.NewSensitiveMarker()

// 标记敏感字段
marker.MarkField("phone", security.SensitiveTypePhone)
marker.MarkField("email", security.SensitiveTypeEmail)
marker.MarkField("password", security.SensitiveTypePassword)

// 检查字段是否敏感
if marker.IsSensitive("phone") {
    fmt.Println("phone 是敏感字段")
}

// 检查是否需要加密
if marker.ShouldEncrypt("password") {
    fmt.Println("password 需要加密存储")
}

// 检查是否需要掩码
if marker.ShouldMask("phone") {
    fmt.Println("phone 需要在日志中掩码")
}
```

### 3. 使用结构体标签

```go
type UserData struct {
    Name     string `json:"name" sensitive:"name"`
    Phone    string `json:"phone" sensitive:"phone"`
    Email    string `json:"email" sensitive:"email"`
    Password string `json:"password" sensitive:"password"`
    Address  string `json:"address" sensitive:"address"`
}

// 创建敏感数据包装器（自动检测 sensitive 标签）
user := UserData{
    Name:     "张三",
    Phone:    "13812345678",
    Email:    "zhangsan@example.com",
    Password: "secret123",
    Address:  "北京市朝阳区建国路88号",
}

sd := security.NewSensitiveData(user)

// JSON 序列化时自动脱敏
jsonData, _ := json.Marshal(sd)
fmt.Println(string(jsonData))
// 输出中敏感字段已被掩码
```

### 4. 脱敏 Map 数据

```go
data := map[string]interface{}{
    "user_id": "12345",
    "phone":   "13812345678",
    "email":   "test@example.com",
    "amount":  1000.00,
}

// 标记敏感字段
marker := security.NewSensitiveMarker()
marker.MarkField("phone", security.SensitiveTypePhone)
marker.MarkField("email", security.SensitiveTypeEmail)

// 脱敏处理
maskedData := security.MaskMap(data, marker)

fmt.Println(maskedData)
// 输出:
// {
//   "user_id": "12345",
//   "phone": "138****5678",
//   "email": "te**@e*****e.com",
//   "amount": 1000.00
// }
```

## 高级配置

### 自定义脱敏配置

```go
config := &security.MaskConfig{
    DefaultMaskChar:    '*',
    PhoneKeepPrefix:    3,
    PhoneKeepSuffix:    4,
    EmailKeepPrefix:    2,
    BankCardKeepPrefix: 4,
    BankCardKeepSuffix: 4,
    IDCardKeepPrefix:   4,
    IDCardKeepSuffix:   4,
    NameKeepFirst:      true,
    CustomRules:        make(map[string]*security.MaskRule),
}

masker, err := security.NewMasker(config)
if err != nil {
    log.Fatal(err)
}
```

### 添加自定义脱敏规则

```go
import "regexp"

masker, _ := security.NewMasker(nil)

// 定义自定义规则（如邮政编码）
rule := &security.MaskRule{
    Pattern:    regexp.MustCompile(`^\d{6}$`),
    KeepPrefix: 2,
    KeepSuffix: 2,
    MaskChar:   '#',
}

// 添加规则
err := masker.AddCustomRule("postal_code", rule)
if err != nil {
    log.Fatal(err)
}

// 使用自定义规则
masked := masker.MaskCustom("100000", "postal_code")
fmt.Println(masked) // 输出: 10##00
```

## 在 Saga 中使用

### 1. 保护 Saga 上下文数据

```go
import (
    "github.com/innovationmech/swit/pkg/saga"
    "github.com/innovationmech/swit/pkg/saga/security"
)

// 定义包含敏感数据的 Saga 上下文
type OrderContext struct {
    OrderID      string `json:"order_id"`
    CustomerName string `json:"customer_name" sensitive:"name"`
    Phone        string `json:"phone" sensitive:"phone"`
    Email        string `json:"email" sensitive:"email"`
    CreditCard   string `json:"credit_card" sensitive:"bankcard"`
    Amount       float64 `json:"amount"`
}

// 创建 Saga 实例时包装敏感数据
ctx := &OrderContext{
    OrderID:      "ORD-12345",
    CustomerName: "李明",
    Phone:        "13912345678",
    Email:        "liming@example.com",
    CreditCard:   "6222021234567890",
    Amount:       999.99,
}

// 包装为敏感数据
sensitiveCtx := security.NewSensitiveData(ctx)

// 在 Saga 定义中使用
sagaInstance := coordinator.CreateSaga("order-saga", sensitiveCtx)
```

### 2. 日志中的敏感数据保护

```go
// 在记录日志前脱敏
func logSagaData(data interface{}) {
    marker := security.NewSensitiveMarker()
    marker.MarkField("phone", security.SensitiveTypePhone)
    marker.MarkField("email", security.SensitiveTypeEmail)
    marker.MarkField("password", security.SensitiveTypePassword)
    
    // 如果是 map 类型
    if mapData, ok := data.(map[string]interface{}); ok {
        maskedData := security.MaskMap(mapData, marker)
        log.Info("Saga data", "data", maskedData)
    } else {
        // 对于其他类型，使用 JSON 脱敏
        maskedData := security.MaskJSON(data, marker)
        log.Info("Saga data", "data", maskedData)
    }
}
```

### 3. 存储前加密敏感字段

```go
// 结合加密功能保护存储的敏感数据
type SagaDataProtector struct {
    encryptor security.Encryptor
    masker    security.Masker
    marker    security.SensitiveMarker
}

func (p *SagaDataProtector) ProtectBeforeStore(data interface{}) (interface{}, error) {
    // 1. 提取敏感字段
    sensitiveFields := security.ExtractSensitiveFields(data, p.marker)
    
    // 2. 加密需要加密的字段
    encrypted := make(map[string]string)
    for field, value := range sensitiveFields {
        if p.marker.ShouldEncrypt(field) {
            strValue := fmt.Sprint(value)
            encData, err := p.encryptor.Encrypt([]byte(strValue))
            if err != nil {
                return nil, err
            }
            encrypted[field] = encData.Ciphertext
        }
    }
    
    // 3. 返回处理后的数据
    return encrypted, nil
}
```

## 性能考虑

### 1. 脱敏性能

数据脱敏操作非常快速：

```go
// 性能基准测试结果
BenchmarkMaskPhone-8      1000000    1.2 µs/op
BenchmarkMaskEmail-8       800000    1.5 µs/op
BenchmarkMaskBankCard-8   1000000    1.1 µs/op
```

### 2. 缓存优化

对于频繁访问的数据，可以缓存脱敏结果：

```go
type CachedMasker struct {
    masker security.Masker
    cache  map[string]string
    mu     sync.RWMutex
}

func (cm *CachedMasker) Mask(data string, dataType security.SensitiveDataType) string {
    key := fmt.Sprintf("%s:%s", dataType, data)
    
    cm.mu.RLock()
    if masked, ok := cm.cache[key]; ok {
        cm.mu.RUnlock()
        return masked
    }
    cm.mu.RUnlock()
    
    masked := cm.masker.Mask(data, dataType)
    
    cm.mu.Lock()
    cm.cache[key] = masked
    cm.mu.Unlock()
    
    return masked
}
```

## 最佳实践

### 1. 始终在日志中掩码敏感数据

```go
// ❌ 不好的做法
log.Info("User data", "phone", user.Phone)

// ✅ 好的做法
masker, _ := security.NewMasker(nil)
log.Info("User data", "phone", masker.MaskPhone(user.Phone))
```

### 2. 使用结构体标签声明敏感字段

```go
// ✅ 推荐：使用 sensitive 标签
type User struct {
    Name     string `json:"name" sensitive:"name"`
    Phone    string `json:"phone" sensitive:"phone"`
    Email    string `json:"email" sensitive:"email"`
}
```

### 3. 分离敏感数据和非敏感数据

```go
// ✅ 推荐：分离存储
type OrderBasic struct {
    OrderID   string
    Amount    float64
    Status    string
}

type OrderSensitive struct {
    CustomerName string `sensitive:"name"`
    Phone        string `sensitive:"phone"`
    Address      string `sensitive:"address"`
}
```

### 4. 定期审计敏感数据访问

```go
func auditSensitiveAccess(userID, field string, value interface{}) {
    // 记录敏感数据访问
    auditLogger.Log(&security.AuditEntry{
        Level:        security.AuditLevelInfo,
        Action:       "sensitive_data_access",
        ResourceType: "saga_context",
        UserID:       userID,
        Details: map[string]interface{}{
            "field": field,
            "masked_value": security.MaskString(
                fmt.Sprint(value),
                security.SensitiveTypeCustom,
            ),
        },
    })
}
```

## 安全建议

1. **永远不要记录未掩码的敏感数据**
   - 使用脱敏功能处理所有日志输出
   - 配置日志系统自动检测和掩码敏感模式

2. **敏感数据加密存储**
   - 密码、银行卡号等高敏感数据必须加密存储
   - 使用 `security.Encryptor` 进行加密

3. **限制敏感数据访问**
   - 结合 RBAC 控制敏感数据访问权限
   - 审计所有敏感数据访问

4. **传输时使用 TLS**
   - 确保所有敏感数据传输使用 HTTPS/TLS
   - 避免在 URL 中传递敏感数据

5. **定期更新脱敏规则**
   - 根据业务需求调整脱敏规则
   - 添加新的敏感数据类型

## 故障排查

### 脱敏结果不符合预期

```go
// 检查配置
config := security.DefaultMaskConfig()
fmt.Printf("Phone keep prefix: %d\n", config.PhoneKeepPrefix)
fmt.Printf("Phone keep suffix: %d\n", config.PhoneKeepSuffix)

// 调试脱敏过程
masker, _ := security.NewMasker(config)
result := masker.MaskPhone("13812345678")
fmt.Printf("Original: %s, Masked: %s\n", "13812345678", result)
```

### 性能问题

```go
// 使用基准测试识别性能瓶颈
func BenchmarkMasking(b *testing.B) {
    masker, _ := security.NewMasker(nil)
    data := "13812345678"
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        masker.MaskPhone(data)
    }
}
```

## 相关文档

- [Saga 数据加密](./saga-encryption.md)
- [Saga 审计日志](./saga-audit.md)
- [Saga 安全最佳实践](./saga-security-best-practices.md)
- [RBAC 权限控制](./saga-rbac.md)

## 示例代码

完整示例代码请参考：
- `examples/saga/sensitive-data/` - 敏感数据保护示例
- `pkg/saga/security/masking_test.go` - 单元测试示例
- `pkg/saga/security/sensitive_test.go` - 集成测试示例

