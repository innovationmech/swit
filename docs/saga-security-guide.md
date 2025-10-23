# Saga Security Configuration Guide

本指南提供了 Swit Saga 框架中安全功能的完整配置参考和最佳实践。

## 目录

- [概述](#概述)
- [快速开始](#快速开始)
- [配置结构](#配置结构)
  - [认证配置](#认证配置)
  - [授权配置](#授权配置)
  - [加密配置](#加密配置)
  - [审计配置](#审计配置)
  - [数据保护配置](#数据保护配置)
- [部署场景](#部署场景)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)
- [故障排查](#故障排查)

## 概述

Swit Saga 框架提供了企业级的安全特性，包括：

- **认证 (Authentication)**: 支持 JWT 和 API Key 认证
- **授权 (Authorization)**: 基于角色的访问控制 (RBAC) 和访问控制列表 (ACL)
- **加密 (Encryption)**: AES-256-GCM 数据加密
- **审计 (Audit)**: 完整的操作审计日志
- **数据保护 (Data Protection)**: 敏感数据脱敏和保护

所有安全特性通过统一的配置结构进行管理，支持灵活的组合和定制。

## 快速开始

### 最小配置

适用于开发环境的最小安全配置：

```yaml
authentication:
  enabled: true
  default_provider: "jwt"
  jwt:
    secret: "your-secret-key-at-least-32-characters-long"
    issuer: "swit"
    audience: "saga"
```

### 推荐生产配置

生产环境推荐配置：

```yaml
authentication:
  enabled: true
  default_provider: "jwt"
  jwt:
    secret: "${JWT_SECRET}"  # 从环境变量读取
    issuer: "swit-production"
    audience: "saga-production"
    token_expiry: 1h

authorization:
  enabled: true
  rbac:
    enabled: true
    predefined_roles: true

encryption:
  enabled: true
  algorithm: "aes-gcm"
  key_size: 32

audit:
  enabled: true
  storage:
    type: "file"
    file:
      path: "/var/log/swit/audit.log"
      max_file_size: 104857600  # 100MB
      max_backups: 30

data_protection:
  enabled: true
```

## 配置结构

### 认证配置

认证配置控制用户和服务的身份验证。

#### JWT 认证

JWT (JSON Web Token) 适用于用户认证和服务间通信。

```yaml
authentication:
  enabled: true
  default_provider: "jwt"
  jwt:
    # JWT 签名密钥（必需）
    # 生产环境建议至少 64 个随机字符
    secret: "${JWT_SECRET}"
    
    # JWT 颁发者（可选）
    issuer: "swit-production"
    
    # JWT 受众（可选）
    audience: "saga-production"
    
    # Token 过期时间（可选，默认 1h）
    token_expiry: 1h
  
  # 缓存配置（可选）
  cache:
    enabled: true
    ttl: 5m
    max_size: 10000
```

**配置说明：**

- `secret`: JWT 签名密钥，必须安全存储，建议使用环境变量或密钥管理服务
- `issuer`: JWT 颁发者标识，用于验证 token 来源
- `audience`: JWT 目标受众，用于验证 token 用途
- `token_expiry`: Token 有效期，生产环境建议 15 分钟到 1 小时
- `cache`: 缓存已验证的 token 以提高性能

**最佳实践：**

1. **密钥管理**
   - 使用至少 64 个随机字符的强密钥
   - 通过环境变量或密钥管理系统（如 HashiCorp Vault）注入
   - 定期轮换密钥（建议 30-90 天）
   - 永远不要将密钥提交到版本控制

2. **Token 过期时间**
   - 开发环境: 24 小时（方便调试）
   - 测试环境: 2 小时
   - 生产环境: 15 分钟到 1 小时
   - 高安全环境: 15 分钟

3. **性能优化**
   - 启用缓存以减少验证开销
   - 设置合理的缓存 TTL（推荐 5 分钟）
   - 监控缓存命中率

#### API Key 认证

API Key 适用于服务到服务的通信和自动化工具。

```yaml
authentication:
  enabled: true
  default_provider: "apikey"
  api_key:
    # 方式 1: 直接配置（不推荐用于生产）
    keys:
      "api-key-1": "service-1"
      "api-key-2": "service-2"
    
    # 方式 2: 从文件加载（推荐）
    keys_file: "/etc/swit/api-keys.yaml"
  
  cache:
    enabled: true
    ttl: 10m
    max_size: 1000
```

**API Keys 文件格式 (`api-keys.yaml`)：**

```yaml
keys:
  "sk-1234567890abcdef": "payment-service"
  "sk-fedcba0987654321": "order-service"
  "sk-abcdef1234567890": "inventory-service"
```

**最佳实践：**

1. **密钥格式**
   - 使用长度至少 32 字符的随机字符串
   - 添加前缀标识（如 `sk-` 表示 secret key）
   - 使用密码学安全的随机生成器

2. **密钥存储**
   - 生产环境使用文件存储，限制文件权限（600）
   - 考虑使用外部密钥管理服务
   - 定期审计和轮换 API keys

3. **监控和审计**
   - 记录所有 API key 使用情况
   - 设置异常使用告警
   - 实施速率限制

### 授权配置

授权配置控制用户和服务的访问权限。

#### RBAC (基于角色的访问控制)

RBAC 通过角色和权限管理访问控制。

```yaml
authorization:
  enabled: true
  rbac:
    # 启用 RBAC
    enabled: true
    
    # 使用预定义角色（admin, user, readonly）
    predefined_roles: true
    
    # 自定义角色配置文件（可选）
    roles_file: "/etc/swit/roles.yaml"
  
  cache:
    enabled: true
    ttl: 5m
    max_size: 10000
```

**角色定义文件格式 (`roles.yaml`)：**

```yaml
roles:
  # 管理员角色
  - name: "admin"
    description: "System administrator with full access"
    permissions:
      - "saga:*:*"           # 所有 Saga 操作
      - "config:*:*"         # 所有配置操作
      - "audit:read:*"       # 读取审计日志
      - "user:*:*"           # 所有用户管理操作
  
  # 操作员角色
  - name: "operator"
    description: "Operations team member"
    permissions:
      - "saga:execute:*"     # 执行 Saga
      - "saga:read:*"        # 读取 Saga 状态
      - "saga:cancel:*"      # 取消 Saga
      - "audit:read:*"       # 读取审计日志
  
  # 开发者角色
  - name: "developer"
    description: "Developer with read-only access"
    permissions:
      - "saga:read:*"        # 读取 Saga
      - "config:read:*"      # 读取配置
      - "audit:read:own"     # 读取自己的审计日志
  
  # 只读角色
  - name: "readonly"
    description: "Read-only access"
    permissions:
      - "saga:read:*"
      - "config:read:*"
```

**权限格式：**

权限使用 `resource:action:scope` 格式：

- `resource`: 资源类型（saga, config, audit, user）
- `action`: 操作类型（read, write, execute, cancel, delete, *）
- `scope`: 作用域（*, own, specific-id）

**用户角色分配：**

在代码中分配用户角色：

```go
rbacManager.AssignRoleToUser("user-123", "operator")
rbacManager.AssignRoleToUser("user-456", "developer")
```

**最佳实践：**

1. **角色设计原则**
   - 遵循最小权限原则
   - 角色应基于职能而非个人
   - 避免过度细化导致管理复杂
   - 定期审查和清理不使用的角色

2. **权限粒度**
   - 使用通配符减少配置复杂度
   - 对敏感操作使用精确权限
   - 考虑使用范围限制（own, team, org）

3. **性能优化**
   - 启用权限缓存
   - 预加载常用角色权限
   - 监控权限检查性能

#### ACL (访问控制列表)

ACL 提供更细粒度的资源级访问控制。

```yaml
authorization:
  enabled: true
  acl:
    # 启用 ACL
    enabled: true
    
    # 默认决策（无匹配规则时）
    default_effect: "deny"
    
    # ACL 规则配置文件
    rules_file: "/etc/swit/acl-rules.yaml"
    
    # 启用性能指标
    enable_metrics: true
```

**ACL 规则文件格式 (`acl-rules.yaml`)：**

```yaml
rules:
  # 高优先级规则：允许管理员访问所有资源
  - id: "admin-all-access"
    priority: 1000
    effect: "allow"
    principal: "role:admin"
    resource: "*"
    action: "*"
  
  # 允许用户管理自己的 Saga
  - id: "user-own-saga"
    priority: 500
    effect: "allow"
    principal: "user:*"
    resource: "saga:${user.id}:*"
    action: "*"
  
  # 拒绝删除生产环境的 Saga
  - id: "deny-delete-prod"
    priority: 800
    effect: "deny"
    principal: "*"
    resource: "saga:prod-*"
    action: "delete"
    conditions:
      environment: "production"
  
  # 允许支付服务执行支付 Saga
  - id: "payment-service-saga"
    priority: 600
    effect: "allow"
    principal: "service:payment"
    resource: "saga:payment-*"
    action: "execute"
```

**规则匹配逻辑：**

1. 按优先级从高到低评估规则
2. 第一个匹配的规则决定访问结果
3. 如果没有规则匹配，使用 `default_effect`

**最佳实践：**

1. **规则优先级**
   - 1000-900: 全局拒绝规则
   - 899-700: 管理员和系统级规则
   - 699-500: 服务级规则
   - 499-300: 用户级规则
   - 299-100: 默认规则

2. **规则组织**
   - 将相关规则分组
   - 使用清晰的 ID 和描述
   - 定期审查和清理过期规则

3. **性能考虑**
   - 减少规则数量
   - 将高频规则设置为高优先级
   - 使用通配符减少规则复杂度
   - 启用指标监控规则评估性能

### 加密配置

加密配置控制敏感数据的加密和密钥管理。

```yaml
encryption:
  # 启用加密
  enabled: true
  
  # 加密算法（aes-gcm 或 aes-cbc）
  algorithm: "aes-gcm"
  
  # 密钥大小（16=AES-128, 24=AES-192, 32=AES-256）
  key_size: 32
  
  # 密钥管理配置
  key_manager:
    # 密钥管理器类型（memory, file, external）
    type: "file"
    
    # 密钥文件路径（当 type=file 时）
    key_file: "/etc/swit/secrets/encryption.key"
    
    # 密钥轮换间隔（0=不轮换）
    rotation_interval: 2160h  # 90 天
```

**加密算法选择：**

- **AES-GCM (推荐)**: 认证加密，提供数据完整性和保密性
- **AES-CBC**: 传统分组加密，需要额外的完整性保护

**密钥管理器类型：**

1. **Memory (内存)**
   - 适用于：开发和测试环境
   - 优点：简单，无需外部依赖
   - 缺点：重启后丢失密钥

2. **File (文件)**
   - 适用于：生产环境（小型部署）
   - 优点：持久化，易于管理
   - 缺点：需要安全的文件权限管理

3. **External (外部 KMS)**
   - 适用于：生产环境（大型部署）、高安全环境
   - 优点：集中管理，审计，自动轮换
   - 缺点：需要外部服务（AWS KMS、Azure Key Vault、HashiCorp Vault）

**生成加密密钥：**

```bash
# 生成 256 位（32 字节）随机密钥
openssl rand -hex 32 > encryption.key

# 设置安全的文件权限
chmod 600 encryption.key
chown swit:swit encryption.key
```

**最佳实践：**

1. **密钥管理**
   - 始终使用 AES-256（key_size: 32）
   - 生产环境使用外部 KMS
   - 定期轮换密钥（30-90 天）
   - 保存密钥备份（加密存储）
   - 实施密钥访问审计

2. **加密策略**
   - 仅加密真正敏感的数据
   - 考虑性能影响
   - 使用字段级加密而非整体加密
   - 记录哪些数据被加密

3. **密钥轮换**
   - 制定密钥轮换计划
   - 保留旧密钥以解密历史数据
   - 使用密钥版本管理
   - 测试密钥轮换流程

### 审计配置

审计配置控制操作日志的记录和存储。

```yaml
audit:
  # 启用审计
  enabled: true
  
  # 存储配置
  storage:
    # 存储类型（memory, file, database）
    type: "file"
    
    # 文件存储配置
    file:
      path: "/var/log/swit/audit.log"
      max_file_size: 104857600  # 100MB
      max_backups: 30
      compress: true
    
    # 保留天数（0=永久）
    retention_days: 365
  
  # 审计级别
  levels:
    - info
    - warning
    - error
    - critical
  
  # 审计类别
  categories:
    - saga
    - auth
    - data
    - config
    - security
    - system
  
  # 是否包含敏感数据（谨慎使用）
  include_sensitive: false
```

**存储类型：**

1. **Memory (内存)**
   - 适用于：开发和测试
   - 特点：不持久化，重启丢失

2. **File (文件)**
   - 适用于：中小型部署
   - 特点：本地存储，支持轮换和压缩

3. **Database (数据库)**
   - 适用于：大型部署，需要集中审计
   - 特点：结构化存储，易于查询和分析

**文件存储配置：**

```yaml
storage:
  type: "file"
  file:
    path: "/var/log/swit/audit.log"
    max_file_size: 104857600  # 100MB
    max_backups: 30            # 保留 30 个备份
    compress: true             # 压缩旧文件
  retention_days: 365          # 保留 365 天
```

**数据库存储配置：**

```yaml
storage:
  type: "database"
  database:
    driver: "postgres"
    dsn: "${AUDIT_DATABASE_URL}"
    table_name: "audit_logs"
  retention_days: 365
```

**审计级别：**

- `info`: 正常操作（Saga 执行、步骤完成）
- `warning`: 警告事件（重试、降级）
- `error`: 错误事件（步骤失败、验证失败）
- `critical`: 关键事件（安全违规、系统故障）

**审计类别：**

- `saga`: Saga 生命周期事件
- `auth`: 认证和授权事件
- `data`: 数据访问和修改
- `config`: 配置变更
- `security`: 安全相关事件
- `system`: 系统级事件

**最佳实践：**

1. **存储选择**
   - 小型部署: 文件存储
   - 大型部署: 数据库存储
   - 合规要求: 数据库存储 + 不可变存储

2. **保留策略**
   - 开发环境: 7-30 天
   - 生产环境: 根据合规要求
     - GDPR: 30-90 天（除非法律要求）
     - HIPAA: 6 年
     - SOX: 7 年
     - PCI DSS: 1 年

3. **性能优化**
   - 异步写入审计日志
   - 使用批量写入
   - 定期归档旧日志
   - 监控存储空间

4. **安全考虑**
   - 永远不要记录敏感数据（密码、token）
   - 限制审计日志访问权限
   - 使用只追加存储
   - 实施防篡改措施

### 数据保护配置

数据保护配置控制敏感数据的脱敏和保护。

```yaml
data_protection:
  # 启用数据保护
  enabled: true
  
  # 脱敏规则
  masking_rules:
    # 默认脱敏字符
    default_mask_char: "*"
    
    # 邮箱脱敏策略（full, partial, domain）
    email_strategy: "partial"
    
    # 电话脱敏策略（full, partial, last4）
    phone_strategy: "last4"
    
    # 信用卡脱敏策略（full, last4）
    credit_card_strategy: "last4"
  
  # 敏感字段列表
  sensitive_fields:
    - password
    - secret
    - token
    - api_key
    - credit_card
    - ssn
    - email
    - phone
```

**脱敏策略示例：**

1. **邮箱脱敏**
   - `full`: `***@***.***`
   - `partial`: `u***@example.com`
   - `domain`: `***@example.com`

2. **电话脱敏**
   - `full`: `***-***-****`
   - `partial`: `***-***-1234`
   - `last4`: `+123***7890`

3. **信用卡脱敏**
   - `full`: `**** **** **** ****`
   - `last4`: `**** **** **** 3456`

**在代码中使用数据脱敏：**

```go
import "github.com/innovationmech/swit/pkg/saga/security"

// 创建脱敏器
masker := security.NewDataMasker(&security.MaskConfig{
    DefaultMaskChar: '*',
})

// 脱敏邮箱
masked := masker.MaskEmail("user@example.com", security.MaskPartial)
// 结果: u***@example.com

// 脱敏信用卡
masked = masker.MaskCreditCard("1234567890123456", security.MaskLast4)
// 结果: ************3456

// 脱敏电话
masked = masker.MaskPhone("+12345678901", security.MaskLast4)
// 结果: +123***8901
```

**最佳实践：**

1. **敏感字段识别**
   - 定期审查和更新敏感字段列表
   - 包括所有 PII（个人身份信息）
   - 包括所有 PCI（支付卡信息）
   - 包括所有 PHI（健康信息）

2. **脱敏策略选择**
   - 根据业务需求选择策略
   - 平衡安全性和可用性
   - 考虑合规要求
   - 测试脱敏效果

3. **应用场景**
   - 日志输出
   - API 响应
   - 报告生成
   - 调试信息

## 部署场景

### 开发环境

**目标**: 方便调试，最小安全开销

```yaml
authentication:
  enabled: true
  default_provider: "jwt"
  jwt:
    secret: "dev-secret-key-change-in-production-minimum-32-chars"
    token_expiry: 24h

authorization:
  enabled: false

encryption:
  enabled: false

audit:
  enabled: true
  storage:
    type: "file"
    file:
      path: "./logs/audit-dev.log"

data_protection:
  enabled: false
```

### 测试/预发布环境

**目标**: 接近生产的安全配置，用于集成测试

```yaml
authentication:
  enabled: true
  default_provider: "jwt"
  jwt:
    secret: "test-secret-key-change-in-production-at-least-32-characters"
    token_expiry: 2h

authorization:
  enabled: true
  rbac:
    enabled: true
    predefined_roles: true

encryption:
  enabled: true
  algorithm: "aes-gcm"
  key_size: 32

audit:
  enabled: true
  storage:
    type: "file"
    file:
      path: "/var/log/swit/audit-test.log"
      max_file_size: 52428800  # 50MB
      max_backups: 5
    retention_days: 30

data_protection:
  enabled: true
```

### 生产环境（标准）

**目标**: 平衡安全性和性能

```yaml
authentication:
  enabled: true
  default_provider: "jwt"
  jwt:
    secret: "${JWT_SECRET}"
    issuer: "swit-production"
    audience: "saga-production"
    token_expiry: 1h
  cache:
    enabled: true
    ttl: 5m
    max_size: 10000

authorization:
  enabled: true
  rbac:
    enabled: true
    predefined_roles: true
    roles_file: "/etc/swit/roles.yaml"
  acl:
    enabled: true
    default_effect: "deny"
    rules_file: "/etc/swit/acl-rules.yaml"
    enable_metrics: true

encryption:
  enabled: true
  algorithm: "aes-gcm"
  key_size: 32
  key_manager:
    type: "file"
    key_file: "/etc/swit/secrets/encryption.key"
    rotation_interval: 2160h  # 90 天

audit:
  enabled: true
  storage:
    type: "database"
    database:
      driver: "postgres"
      dsn: "${AUDIT_DATABASE_URL}"
      table_name: "audit_logs"
    retention_days: 365
  levels:
    - info
    - warning
    - error
    - critical
  categories:
    - saga
    - auth
    - data
    - config
    - security
    - system

data_protection:
  enabled: true
  masking_rules:
    email_strategy: "partial"
    phone_strategy: "last4"
    credit_card_strategy: "last4"
```

### 高安全环境

**目标**: 最大安全性，适用于金融、医疗、政府等行业

```yaml
authentication:
  enabled: true
  default_provider: "jwt"
  jwt:
    secret: "${JWT_SECRET}"  # 64+ 字符
    token_expiry: 15m  # 短过期时间
  cache:
    enabled: true
    ttl: 2m  # 短缓存时间

authorization:
  enabled: true
  rbac:
    enabled: true
    predefined_roles: false  # 仅自定义角色
    roles_file: "/etc/swit/security/roles.yaml"
  acl:
    enabled: true
    default_effect: "deny"
    rules_file: "/etc/swit/security/acl-rules.yaml"
    enable_metrics: true

encryption:
  enabled: true
  algorithm: "aes-gcm"
  key_size: 32
  key_manager:
    type: "external"  # 使用外部 KMS
    rotation_interval: 720h  # 30 天

audit:
  enabled: true
  storage:
    type: "database"
    database:
      driver: "postgres"
      dsn: "${AUDIT_DATABASE_URL}"
      table_name: "audit_logs"
    retention_days: 2555  # 7 年
  levels:
    - info
    - warning
    - error
    - critical
  categories:
    - saga
    - auth
    - data
    - config
    - security
    - system

data_protection:
  enabled: true
  masking_rules:
    email_strategy: "full"  # 完全脱敏
    phone_strategy: "full"
    credit_card_strategy: "full"
```

## 最佳实践

### 1. 密钥管理

**生成强密钥：**

```bash
# JWT Secret (64 字符)
openssl rand -base64 48 > jwt-secret.txt

# 加密密钥 (32 字节 = AES-256)
openssl rand -hex 32 > encryption.key

# API Key (32 字符)
openssl rand -base64 32 | tr -d "=+/" | cut -c1-32
```

**存储密钥：**

1. **环境变量** (推荐用于容器部署)
```bash
export JWT_SECRET=$(cat jwt-secret.txt)
export ENCRYPTION_KEY=$(cat encryption.key)
```

2. **密钥文件** (推荐用于传统部署)
```bash
# 创建密钥目录
mkdir -p /etc/swit/secrets
chmod 700 /etc/swit/secrets

# 存储密钥
echo "your-jwt-secret" > /etc/swit/secrets/jwt.key
chmod 600 /etc/swit/secrets/jwt.key
chown swit:swit /etc/swit/secrets/jwt.key
```

3. **外部密钥管理系统** (推荐用于生产)
   - AWS Secrets Manager
   - HashiCorp Vault
   - Azure Key Vault
   - Google Secret Manager

**密钥轮换：**

```bash
# 1. 生成新密钥
openssl rand -hex 32 > encryption-new.key

# 2. 更新配置（支持旧密钥解密）
# 3. 重启服务
systemctl restart swit-serve

# 4. 验证新密钥工作正常
# 5. 重新加密历史数据（可选）
# 6. 删除旧密钥
```

### 2. 配置管理

**配置文件组织：**

```
/etc/swit/
├── swit.yaml              # 主配置
├── security.yaml          # 安全配置
├── roles.yaml             # RBAC 角色
├── acl-rules.yaml         # ACL 规则
├── api-keys.yaml          # API Keys
└── secrets/
    ├── jwt.key            # JWT 密钥
    └── encryption.key     # 加密密钥
```

**配置分层：**

```yaml
# swit.yaml - 引用安全配置
security:
  config_file: "/etc/swit/security.yaml"

# security.yaml - 具体安全配置
authentication:
  enabled: true
  ...
```

**环境特定配置：**

```yaml
# swit-dev.yaml
security:
  <<: *common-security
  authentication:
    jwt:
      token_expiry: 24h  # 覆盖默认值

# swit-prod.yaml
security:
  <<: *common-security
  authentication:
    jwt:
      token_expiry: 1h
```

### 3. 监控和告警

**关键指标：**

1. **认证指标**
   - 认证成功/失败率
   - Token 验证延迟
   - 缓存命中率

2. **授权指标**
   - 权限检查延迟
   - 访问拒绝率
   - ACL 规则评估时间

3. **加密指标**
   - 加密/解密操作数
   - 加密操作延迟
   - 密钥轮换状态

4. **审计指标**
   - 审计日志写入速率
   - 审计存储使用量
   - 关键安全事件数

**告警配置示例（Prometheus）：**

```yaml
groups:
  - name: security_alerts
    rules:
      # 高认证失败率
      - alert: HighAuthFailureRate
        expr: rate(auth_failures_total[5m]) > 0.1
        for: 5m
        annotations:
          summary: "High authentication failure rate"
      
      # 异常访问拒绝
      - alert: UnusualAccessDenials
        expr: rate(acl_denied_total[15m]) > 10
        for: 10m
        annotations:
          summary: "Unusual number of access denials"
      
      # 审计存储空间不足
      - alert: AuditStorageLow
        expr: audit_storage_available_bytes < 1e9
        annotations:
          summary: "Audit storage below 1GB"
```

### 4. 安全审查清单

**部署前检查：**

- [ ] JWT secret 至少 64 字符
- [ ] 加密使用 AES-256
- [ ] 审计日志已启用
- [ ] 敏感数据已配置脱敏
- [ ] 密钥文件权限正确（600）
- [ ] 密钥不在版本控制中
- [ ] 使用环境变量或密钥管理服务
- [ ] Token 过期时间适当（≤1h）
- [ ] 启用 RBAC 或 ACL
- [ ] 配置了监控和告警
- [ ] 测试了密钥轮换流程
- [ ] 审查了审计日志内容
- [ ] 验证了数据脱敏效果

**定期审查：**

- [ ] 审查用户权限（每季度）
- [ ] 轮换 JWT secret（每 30-90 天）
- [ ] 轮换加密密钥（每 30-90 天）
- [ ] 轮换 API keys（每 90 天）
- [ ] 审查 ACL 规则（每月）
- [ ] 检查审计日志异常（每周）
- [ ] 更新敏感字段列表（每季度）
- [ ] 审查安全告警（持续）

## 常见问题

### Q1: JWT Secret 应该多长？

**A**: 
- 最小: 32 字符（256 位）
- 推荐: 64 字符（512 位）
- 高安全: 128 字符（1024 位）

使用密码学安全的随机生成器生成，避免使用词典单词或可预测模式。

### Q2: 应该使用 RBAC 还是 ACL？

**A**:
- **RBAC**: 适用于基于职能的访问控制，易于管理大量用户
- **ACL**: 适用于细粒度的资源级控制，灵活但复杂
- **推荐**: 同时使用，RBAC 处理常规权限，ACL 处理特殊情况

### Q3: 审计日志应该保留多久？

**A**: 取决于合规要求
- 开发/测试: 7-30 天
- 一般生产: 90 天到 1 年
- GDPR: 30-90 天（除非法律要求）
- HIPAA: 6 年
- SOX: 7 年
- PCI DSS: 至少 1 年

### Q4: 如何在不停机的情况下轮换密钥？

**A**:
1. 实施密钥版本管理
2. 同时支持旧密钥和新密钥
3. 逐步迁移到新密钥
4. 监控旧密钥使用情况
5. 确认无使用后删除旧密钥

### Q5: 数据脱敏会影响性能吗？

**A**: 有轻微影响，但通常可以接受
- 在输出前脱敏（日志、API 响应）
- 不要脱敏内部数据处理
- 使用缓存减少重复脱敏
- 监控性能影响

### Q6: 如何处理密钥泄露？

**A**:
1. 立即轮换所有受影响的密钥
2. 审查审计日志查找异常访问
3. 通知受影响的用户
4. 记录事件并进行事后分析
5. 加强密钥管理流程

## 故障排查

### 认证失败

**症状**: JWT 验证失败，401 Unauthorized

**排查步骤**:

1. **检查 JWT Secret 配置**
```bash
# 验证环境变量
echo $JWT_SECRET

# 检查配置文件
cat /etc/swit/security.yaml | grep -A5 jwt
```

2. **验证 Token 格式**
```bash
# 解码 JWT token（不验证签名）
echo "eyJhbGc..." | base64 -d
```

3. **检查 Token 过期时间**
```go
// 在代码中打印 token 信息
claims := jwt.MapClaims{}
token, _ := jwt.ParseWithClaims(tokenString, claims, nil)
fmt.Printf("Expiry: %v\n", claims["exp"])
```

4. **查看审计日志**
```bash
grep "authentication failed" /var/log/swit/audit.log
```

### 授权失败

**症状**: 403 Forbidden，权限不足

**排查步骤**:

1. **检查用户角色**
```go
roles := rbacManager.GetUserRoles("user-123")
fmt.Printf("User roles: %v\n", roles)
```

2. **检查权限配置**
```bash
cat /etc/swit/roles.yaml
```

3. **启用调试日志**
```yaml
# 在配置中启用详细日志
authorization:
  rbac:
    debug: true
```

4. **测试权限检查**
```go
allowed := rbacManager.CheckPermission("user-123", "saga:execute:*")
fmt.Printf("Permission allowed: %v\n", allowed)
```

### 加密错误

**症状**: 加密/解密失败

**排查步骤**:

1. **验证密钥文件**
```bash
# 检查文件存在和权限
ls -l /etc/swit/secrets/encryption.key

# 验证密钥长度
wc -c /etc/swit/secrets/encryption.key
# 应该是 64 字符（32 字节的十六进制）
```

2. **检查密钥格式**
```bash
# 密钥应该是有效的十六进制
cat /etc/swit/secrets/encryption.key | grep -E '^[0-9a-fA-F]+$'
```

3. **测试加密功能**
```go
encryptor, err := security.NewAESGCMEncryptor(config)
if err != nil {
    fmt.Printf("Encryptor init error: %v\n", err)
}

encrypted, err := encryptor.Encrypt([]byte("test"))
if err != nil {
    fmt.Printf("Encryption error: %v\n", err)
}
```

### 审计日志未写入

**症状**: 审计日志文件为空或数据库无记录

**排查步骤**:

1. **检查审计配置**
```yaml
audit:
  enabled: true  # 确保已启用
```

2. **验证文件权限**
```bash
# 检查日志目录权限
ls -ld /var/log/swit/
ls -l /var/log/swit/audit.log

# 确保进程有写权限
sudo -u swit touch /var/log/swit/test
```

3. **检查磁盘空间**
```bash
df -h /var/log/swit/
```

4. **测试数据库连接（如使用数据库存储）**
```bash
psql ${AUDIT_DATABASE_URL} -c "SELECT 1;"
```

5. **查看应用日志**
```bash
journalctl -u swit-serve | grep -i audit
```

### 性能问题

**症状**: 安全检查导致请求延迟增加

**排查步骤**:

1. **检查缓存配置**
```yaml
cache:
  enabled: true  # 确保缓存已启用
  ttl: 5m
  max_size: 10000
```

2. **监控缓存命中率**
```go
// 添加监控指标
cacheHitRate := float64(cacheHits) / float64(cacheRequests)
```

3. **优化 ACL 规则**
```bash
# 减少规则数量
cat /etc/swit/acl-rules.yaml | grep "^  - id:" | wc -l
```

4. **使用性能分析**
```bash
go tool pprof http://localhost:6060/debug/pprof/profile
```

5. **检查审计日志写入性能**
```bash
# 监控审计日志写入延迟
grep "audit.write" /var/log/swit/app.log | tail -100
```

## 参考资源

### 官方文档

- [Saga 架构文档](architecture.md)
- [开发者指南](developer-guide.md)
- [操作指南](operations-guide.md)

### 示例

- [完整配置示例](../examples/saga-security-config.yaml)
- [RBAC 示例](../examples/rbac/)
- [ACL 示例](../examples/acl-example.md)
- [审计示例](../examples/saga-audit-example.go)

### 工具

- [JWT Debugger](https://jwt.io/)
- [OpenSSL](https://www.openssl.org/)
- [HashiCorp Vault](https://www.vaultproject.io/)

### 标准和合规

- [NIST Cryptographic Standards](https://csrc.nist.gov/projects/cryptographic-standards-and-guidelines)
- [OWASP Security Guidelines](https://owasp.org/)
- [GDPR Compliance](https://gdpr.eu/)
- [PCI DSS](https://www.pcisecuritystandards.org/)

## 支持

如有问题或需要帮助，请：

1. 查看本文档的故障排查部分
2. 检查 [GitHub Issues](https://github.com/innovationmech/swit/issues)
3. 参考示例代码
4. 联系支持团队

---

**最后更新**: 2025-01-23  
**版本**: 1.0.0

