# SwitAuth API 文档

SwitAuth 是 SWIT 项目的认证授权服务，负责用户身份验证、Token管理和权限控制。

## 🚀 快速访问

- **API Base URL**: http://localhost:9001
- **Swagger UI**: http://localhost:9001/swagger/index.html
- **健康检查**: http://localhost:9001/health

## 📋 API概览

### 认证端点
| 方法 | 端点 | 描述 | 状态 |
|------|------|------|------|
| POST | `/auth/login` | 用户登录 | ✅ 已实现 |
| POST | `/auth/logout` | 用户登出 | ✅ 已实现 |
| POST | `/auth/refresh` | 刷新Token | ✅ 已实现 |
| GET | `/auth/validate` | 验证Token | ✅ 已实现 |

### 系统端点
| 方法 | 端点 | 描述 | 状态 |
|------|------|------|------|
| GET | `/health` | 健康检查 | ✅ 已实现 |

## 📊 数据模型

### 用户模型 (User)
```go
type User struct {
    ID           uuid.UUID `json:"id"`
    Username     string    `json:"username"`
    Email        string    `json:"email"`
    Role         string    `json:"role"`
    IsActive     bool      `json:"is_active"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

### 请求/响应模型

#### 登录请求
```json
{
  "username": "john_doe",
  "password": "password123"
}
```

#### 登录响应
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

#### Token验证响应
```json
{
  "message": "Token is valid",
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

## 🔧 使用示例

### 1. 用户登录
```bash
curl -X POST http://localhost:9001/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123"
  }'
```

**响应示例:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### 2. 验证Token
```bash
curl -X GET http://localhost:9001/auth/validate \
  -H "Authorization: Bearer your_access_token_here"
```

**响应示例:**
```json
{
  "message": "Token is valid",
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 3. 刷新Token
```bash
curl -X POST http://localhost:9001/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "your_refresh_token_here"
  }'
```

### 4. 用户登出
```bash
curl -X POST http://localhost:9001/auth/logout \
  -H "Authorization: Bearer your_access_token_here"
```

**响应示例:**
```json
{
  "message": "Logged out successfully"
}
```

### 5. 健康检查
```bash
curl -X GET http://localhost:9001/health
```

**响应示例:**
```json
{
  "message": "pong"
}
```

## 🛠 开发状态

### 已完成功能
- ✅ 基础认证框架
- ✅ JWT Token生成和验证
- ✅ 用户登录/登出
- ✅ Token刷新机制
- ✅ Swagger文档生成
- ✅ OpenAPI 3.0支持
- ✅ 完整的API文档

### 待扩展功能
- ⏳ 用户注册接口
- ⏳ 权限管理
- ⏳ 密码重置
- ⏳ 多因子认证

## 📁 源码结构

```
internal/switauth/
├── client/          # 客户端代码
│   ├── user_client.go
│   └── user_client_test.go
├── cmd/            # 命令行相关
│   ├── cmd.go
│   ├── cmd_test.go
│   └── start/      # 启动命令
├── config/         # 配置管理
│   ├── config.go
│   └── config_test.go
├── db/             # 数据库连接
│   ├── db.go
│   └── db_test.go
├── handler/        # API处理器
│   ├── grpc/       # gRPC处理器
│   │   └── auth/
│   └── http/       # HTTP处理器
│       ├── auth/   # 认证相关API
│       └── health/ # 健康检查
├── model/          # 数据模型
│   ├── token.go    # Token模型
│   └── user.go     # 用户模型
├── repository/     # 数据访问层
│   ├── token_repository.go
│   └── token_repository_test.go
├── service/        # 业务逻辑层
│   ├── auth/       # 认证服务
│   │   ├── adapter.go
│   │   ├── registrar.go
│   │   └── v1/
│   ├── auth.go
│   ├── auth_test.go
│   └── health/     # 健康检查服务
├── transport/      # 传输层
│   ├── grpc.go     # gRPC传输
│   ├── http.go     # HTTP传输
│   ├── registrar.go
│   └── transport.go
├── server.go       # 服务器主文件
└── server_test.go
```

## 🧪 测试

### 运行测试
```bash
# 运行所有测试
go test ./internal/switauth/... -v

# 运行特定模块测试
go test ./internal/switauth/handler/... -v
go test ./internal/switauth/service/... -v
go test ./internal/switauth/transport/... -v

# 运行测试并查看覆盖率
go test ./internal/switauth/... -cover

# 运行竞态条件检测
go test ./internal/switauth/... -race
```

### 测试覆盖
- ✅ Handler 层单元测试
- ✅ Service 层业务逻辑测试
- ✅ Transport 层集成测试
- ✅ Repository 层数据访问测试

## 📖 文档

### 生成API文档
```bash
# 生成 Swagger 文档
make swagger-switauth

# 查看生成的文档
open docs/generated/switauth/swagger.json

# 启动服务后访问 Swagger UI
open http://localhost:9001/swagger/index.html
```

### 文档位置
- **生成的 API 文档**: `docs/generated/switauth/` (自动生成)
- **Swagger JSON**: `docs/generated/switauth/swagger.json` (自动生成)
- **Swagger YAML**: `docs/generated/switauth/swagger.yaml` (自动生成)
- **Go 文档**: `docs/generated/switauth/docs.go` (自动生成)

## 🚀 快速开始

### 1. 启动服务
```bash
# 从项目根目录启动
go run cmd/switauth/main.go

# 或使用 Make 命令
make run-switauth
```

### 2. 验证服务
```bash
# 检查健康状态
curl http://localhost:9001/health

# 访问 Swagger UI
open http://localhost:9001/swagger/index.html
```

### 3. 测试认证流程
```bash
# 1. 登录获取 Token
TOKEN=$(curl -s -X POST http://localhost:9001/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}' \
  | jq -r '.access_token')

# 2. 使用 Token 验证
curl -X GET http://localhost:9001/auth/validate \
  -H "Authorization: Bearer $TOKEN"

# 3. 登出
curl -X POST http://localhost:9001/auth/logout \
  -H "Authorization: Bearer $TOKEN"
```

## 📖 相关文档

### 项目文档
- [项目文档首页](../../README.md)
- [服务架构分析](../../service-architecture-analysis.md)
- [SwitAuth 重构指南](../../switauth-refactoring-guide.md)

### API 文档
- [SwitServe API文档](../switserve/README.md)
- [API文档汇总](../../generated/)
- [服务文档导航](../README.md)

### 开发指南
- [快速开始指南](../../quick-start-example.md)
- [开发环境配置](../../development-setup.md)