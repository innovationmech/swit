# SwitAuth API 文档

SwitAuth 是 SWIT 项目的认证授权服务，负责用户身份验证、Token管理和权限控制。

## 🚀 快速访问

- **API Base URL**: http://localhost:8090
- **Swagger UI**: http://localhost:8090/swagger/index.html
- **健康检查**: http://localhost:8090/health

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

## 🔧 使用示例

### 用户登录
```bash
curl -X POST http://localhost:8090/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "password": "password123"
  }'
```

**响应示例**:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### Token验证
```bash
curl -X GET http://localhost:8090/auth/validate \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**响应示例**:
```json
{
  "message": "Token is valid",
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 刷新Token
```bash
curl -X POST http://localhost:8090/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

**响应示例**:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### 用户登出
```bash
curl -X POST http://localhost:8090/auth/logout \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**响应示例**:
```json
{
  "message": "Logged out successfully"
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
├── handler/          # API处理器
│   ├── auth.go      # 认证相关API
│   ├── login.go     # 登录处理
│   ├── logout.go    # 登出处理
│   ├── refresh_token.go  # Token刷新
│   ├── validate_token.go # Token验证
│   └── health.go    # 健康检查
├── service/         # 业务逻辑层
├── repository/      # 数据访问层
├── model/          # 数据模型
├── config/         # 配置管理
├── docs/           # Swagger生成的API文档
│   ├── docs.go
│   ├── swagger.json
│   ├── swagger.yaml
│   └── README.md
├── middleware/     # 中间件
└── router/         # 路由注册
```

## 🧪 测试和文档

### 重新生成API文档
```bash
make swagger-switauth
```

### 格式化Swagger注释
```bash
make swagger-fmt-switauth
```

### 运行测试
```bash
make test-switauth
```

## 📖 相关文档

- [项目文档首页](../../README.md)
- [SwitServe API文档](../switserve/README.md)
- [快速开始指南](../../quick-start-example.md)
- [API文档汇总](../../generated/) 