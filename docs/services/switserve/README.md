# SwitServe API 文档

SwitServe 是 SWIT 项目的用户管理和内容服务，提供用户CRUD操作、用户信息查询等核心功能。

## 🚀 快速访问

- **API Base URL**: http://localhost:9000
- **Swagger UI**: http://localhost:9000/swagger/index.html
- **健康检查**: http://localhost:9000/health

## 📋 API概览

### 公开端点（无需认证）
| 方法 | 端点 | 描述 | API版本 | 认证要求 |
|------|------|------|---------|----------|
| GET | `/health` | 服务健康检查 | v1 | 无 |
| POST | `/users/create` | 创建新用户 | v1 | 无 |
| POST | `/stop` | 优雅停止服务器 | v1 | 无 |

### 需要认证的端点
| 方法 | 端点 | 描述 | API版本 | 认证要求 |
|------|------|------|---------|----------|
| GET | `/users/username/{username}` | 根据用户名获取用户信息 | v1 | JWT Token |
| GET | `/users/email/{email}` | 根据邮箱获取用户信息 | v1 | JWT Token |
| DELETE | `/users/{id}` | 删除指定用户 | v1 | JWT Token |

### 内部API（服务间调用）
| 方法 | 端点 | 描述 | API版本 | 认证要求 |
|------|------|------|---------|----------|
| POST | `/internal/validate-user` | 验证用户凭据（供SwitAuth调用） | v1 | 内部调用 |

## 📊 数据模型

### User 结构体
```go
type User struct {
    ID           uuid.UUID `json:"id"`           // 用户唯一标识
    Username     string    `json:"username"`     // 用户名（唯一）
    Email        string    `json:"email"`        // 邮箱地址（唯一）
    PasswordHash string    `json:"-"`            // 密码哈希（不返回给客户端）
    Role         string    `json:"role"`         // 用户角色（默认：user）
    IsActive     bool      `json:"is_active"`    // 账户激活状态
    CreatedAt    time.Time `json:"created_at"`   // 创建时间
    UpdatedAt    time.Time `json:"updated_at"`   // 更新时间
}
```

### 请求/响应模型

#### CreateUserRequest
```go
type CreateUserRequest struct {
    Username string `json:"username" binding:"required"`        // 用户名
    Email    string `json:"email" binding:"required,email"`    // 邮箱地址
    Password string `json:"password" binding:"required,min=6"` // 密码（最少6位）
}
```

#### 通用响应格式
```go
// 成功响应
type SuccessResponse struct {
    Message string      `json:"message"`  // 成功消息
    Data    interface{} `json:"data"`     // 响应数据
}

// 错误响应
type ErrorResponse struct {
    Error   string `json:"error"`   // 错误代码
    Message string `json:"message"` // 错误描述
    Details string `json:"details"` // 详细信息
}
```

## 🔧 使用示例

### 1. 创建用户

```bash
curl -X POST http://localhost:9000/users/create \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "email": "john@example.com", 
    "password": "secret123"
  }'
```

**响应示例**:
```json
{
  "message": "User created successfully",
  "user_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

### 2. 获取用户信息（需要Token）

```bash
curl -X GET http://localhost:9000/users/username/john_doe \
  -H "Authorization: Bearer your-jwt-token"
```

**响应示例**:
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "username": "john_doe",
  "email": "john@example.com",
  "created_at": "2023-01-15T10:30:00Z",
  "updated_at": "2023-01-15T10:30:00Z"
}
```

### 3. 删除用户（需要Token）

```bash
curl -X DELETE http://localhost:9000/users/123e4567-e89b-12d3-a456-426614174000 \
  -H "Authorization: Bearer your-jwt-token"
```

## 🔐 认证说明

### 获取Token
大部分API需要JWT Token认证。首先需要通过SwitAuth服务获取Token：

```bash
# 1. 登录获取Token
curl -X POST http://localhost:9001/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "password": "secret123"
  }'

# 2. 使用返回的Token调用SwitServe API
curl -X GET http://localhost:9000/users/username/john_doe \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Token格式
```
Authorization: Bearer <JWT-TOKEN>
```

## 📊 错误响应

### 通用错误格式
```json
{
  "error": "error_code",
  "message": "Human readable error message",
  "details": "Additional error details"
}
```

### 常见错误码
- `400` - 请求参数错误
- `401` - 未认证或Token无效
- `404` - 用户不存在
- `500` - 服务器内部错误

## 🏗 架构设计

### 服务架构
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   SwitAuth      │    │   SwitServe     │    │   Database      │
│  (认证服务)      │    │  (用户服务)      │    │   (MySQL)       │
│                 │    │                 │    │                 │
│ Port: 9001      │◄──►│ Port: 9000      │◄──►│ Port: 3306      │
│ - JWT认证       │    │ - 用户管理       │    │ - 用户数据       │
│ - Token验证     │    │ - CRUD操作      │    │ - 持久化存储     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 技术栈
- **框架**: Gin (HTTP) + gRPC
- **数据库**: MySQL + GORM
- **认证**: JWT Token (通过SwitAuth)
- **文档**: Swagger/OpenAPI 3.0
- **服务发现**: Consul
- **配置管理**: Viper

### 分层架构
```
┌─────────────────────────────────────────┐
│              Transport Layer            │  ← HTTP/gRPC 传输层
├─────────────────────────────────────────┤
│              Handler Layer              │  ← 请求处理层
├─────────────────────────────────────────┤
│              Service Layer              │  ← 业务逻辑层
├─────────────────────────────────────────┤
│            Repository Layer             │  ← 数据访问层
├─────────────────────────────────────────┤
│              Model Layer                │  ← 数据模型层
└─────────────────────────────────────────┘
```

## 🚧 开发状态

### ✅ 已完成功能
- [x] 用户CRUD操作（创建、查询、删除）
- [x] JWT Token认证集成
- [x] 健康检查端点
- [x] 优雅关机机制
- [x] Swagger API文档
- [x] gRPC服务支持
- [x] 数据库连接和迁移
- [x] 服务发现集成
- [x] 错误处理和日志记录

### 🔄 待扩展功能
- [ ] 用户更新操作（PUT/PATCH）
- [ ] 用户角色权限管理
- [ ] 用户头像上传
- [ ] 用户活动日志
- [ ] 批量用户操作
- [ ] 用户搜索和过滤
- [ ] 缓存机制（Redis）
- [ ] 限流和熔断
- [ ] 监控和指标收集

## 🛠 开发指南

### 本地开发
```bash
# 启动服务
make run-switserve

# 或直接运行
go run cmd/swit-serve/swit-serve.go
```

### 重新生成API文档
```bash
make swagger
```

### 运行测试
```bash
# 运行所有测试
make test-switserve

# 运行特定包的测试
go test ./internal/switserve/...

# 运行测试并生成覆盖率报告
go test -cover ./internal/switserve/...

# 运行集成测试
go test -tags=integration ./internal/switserve/...
```

### 文档生成
```bash
# 生成 Swagger 文档
swag init -g internal/switserve/server.go -o docs/generated/switserve

# 或使用 Makefile
make swagger-switserve
```

## 🚀 快速开始

### 1. 环境准备
```bash
# 克隆项目
git clone <repository-url>
cd swit

# 安装依赖
go mod download

# 启动 MySQL 数据库
docker run -d --name mysql \
  -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_DATABASE=user_service_db \
  -p 3306:3306 mysql:8.0

# 启动 Consul（服务发现）
docker run -d --name consul \
  -p 8500:8500 consul:latest
```

### 2. 配置文件
确保 `swit.yaml` 配置正确：
```yaml
database:
  host: 127.0.0.1
  port: 3306
  username: root
  password: root
  dbname: user_service_db
server:
  port: 9000
serviceDiscovery:
  address: "localhost:8500"
```

### 3. 启动服务
```bash
# 启动 SwitAuth 服务（认证依赖）
make run-switauth

# 启动 SwitServe 服务
make run-switserve
```

### 4. 验证服务
```bash
# 检查健康状态
curl http://localhost:9000/health

# 访问 Swagger UI
open http://localhost:9000/swagger/index.html
```

## 📁 源码结构

```
internal/switserve/
├── cmd/            # 命令行相关
│   ├── cmd.go
│   ├── cmd_test.go
│   ├── serve/      # 服务启动命令
│   └── version/    # 版本命令
├── config/         # 配置管理
│   ├── config.go
│   └── config_test.go
├── db/             # 数据库连接
│   ├── db.go
│   └── db_test.go
├── handler/        # API处理器
│   ├── grpc/       # gRPC处理器
│   │   └── greeter/
│   └── http/       # HTTP处理器
│       ├── health/ # 健康检查
│       ├── stop/   # 停止服务
│       └── user/   # 用户相关API
├── model/          # 数据模型
│   └── user.go     # 用户模型
├── repository/     # 数据访问层
│   ├── user_repository.go
│   └── user_repository_test.go
├── service/        # 业务逻辑层
│   ├── greeter/    # 问候服务
│   ├── health/     # 健康检查服务
│   ├── notification/ # 通知服务
│   ├── stop/       # 停止服务
│   └── user/       # 用户服务
├── transport/      # 传输层
│   ├── grpc.go     # gRPC传输
│   ├── http.go     # HTTP传输
│   ├── registrar.go
│   └── transport.go
├── server.go       # 服务器主文件
└── server_test.go
```

## 📖 文档位置

- **API文档**: `docs/generated/switserve/docs.go` (自动生成)
- **Swagger JSON**: `docs/generated/switserve/swagger.json` (自动生成)
- **Swagger YAML**: `docs/generated/switserve/swagger.yaml` (自动生成)
- **Go文档**: `docs/generated/switserve/` (自动生成)

## 📖 相关文档

- [路由注册系统](../../route-registry-guide.md)
- [OpenAPI集成说明](../../openapi-integration.md)
- [项目快速开始](../../quick-start-example.md)