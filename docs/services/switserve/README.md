# SwitServe API 文档

SwitServe 是 SWIT 项目的用户管理和内容服务，提供用户CRUD操作、用户信息查询等核心功能。

## 🚀 快速访问

- **API Base URL**: http://localhost:9000
- **Swagger UI**: http://localhost:9000/swagger/index.html
- **健康检查**: http://localhost:9000/health

## 📋 API概览

### 公开端点（无需认证）
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/health` | 服务健康检查 |
| POST | `/users/create` | 创建新用户 |
| POST | `/stop` | 优雅停止服务器 |

### 需要认证的端点
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/users/username/{username}` | 根据用户名获取用户信息 |
| GET | `/users/email/{email}` | 根据邮箱获取用户信息 |
| DELETE | `/users/{id}` | 删除指定用户 |

### 内部API（服务间调用）
| 方法 | 端点 | 描述 |
|------|------|------|
| POST | `/internal/validate-user` | 验证用户凭据（供SwitAuth调用） |

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
curl -X POST http://localhost:8080/auth/login \
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
make test-switserve
```

## 📁 源码结构

```
internal/switserve/
├── handler/          # API处理器
│   ├── v1/user/     # 用户相关API
│   ├── health/      # 健康检查
│   └── stop/        # 停止服务
├── service/         # 业务逻辑层
├── repository/      # 数据访问层
├── model/          # 数据模型
├── middleware/     # 中间件
├── router/         # 路由注册
└── docs/           # Swagger生成的API文档
    ├── docs.go
    ├── swagger.json
    └── swagger.yaml
```

## 📖 相关文档

- [路由注册系统](../../route-registry-guide.md)
- [OpenAPI集成说明](../../openapi-integration.md)
- [项目快速开始](../../quick-start-example.md) 