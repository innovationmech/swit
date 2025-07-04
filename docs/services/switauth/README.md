# SwitAuth API 文档

SwitAuth 是 SWIT 项目的认证授权服务，负责用户身份验证、Token管理和权限控制。

## 🚀 快速访问

- **API Base URL**: http://localhost:8080
- **Swagger UI**: http://localhost:8080/swagger/index.html（待完善）
- **健康检查**: http://localhost:8080/health

## 📋 API概览

> **注意**: SwitAuth 服务的详细API文档正在完善中。目前可以参考源码中的handler实现。

### 认证端点
| 方法 | 端点 | 描述 | 状态 |
|------|------|------|------|
| POST | `/auth/login` | 用户登录 | ✅ 已实现 |
| POST | `/auth/logout` | 用户登出 | ✅ 已实现 |
| POST | `/auth/refresh` | 刷新Token | ✅ 已实现 |
| POST | `/auth/validate` | 验证Token | ✅ 已实现 |

### 系统端点
| 方法 | 端点 | 描述 | 状态 |
|------|------|------|------|
| GET | `/health` | 健康检查 | ✅ 已实现 |

## 🔧 使用示例

### 用户登录
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "password": "secret123"
  }'
```

### Token验证
```bash
curl -X POST http://localhost:8080/auth/validate \
  -H "Content-Type: application/json" \
  -d '{
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

## 🛠 开发状态

### 已完成功能
- ✅ 基础认证框架
- ✅ JWT Token生成和验证
- ✅ 用户登录/登出
- ✅ Token刷新机制

### 待完善功能
- ⏳ Swagger文档生成
- ⏳ 用户注册接口
- ⏳ 权限管理
- ⏳ 密码重置

## 📁 源码结构

```
internal/switauth/
├── handler/          # API处理器
│   ├── auth.go      # 认证相关API
│   ├── login.go     # 登录处理
│   ├── logout.go    # 登出处理
│   ├── refresh_token.go  # Token刷新
│   └── validate_token.go # Token验证
├── service/         # 业务逻辑层
├── repository/      # 数据访问层
├── model/          # 数据模型
└── config/         # 配置管理
```

## 🚧 TODO

1. **完善Swagger文档**: 为所有API端点添加Swagger注释
2. **集成测试**: 添加完整的API测试套件
3. **权限系统**: 实现基于角色的权限控制
4. **监控指标**: 添加认证相关的监控指标

## 📖 相关文档

- [项目文档首页](../../README.md)
- [SwitServe API文档](../switserve/README.md)
- [快速开始指南](../../quick-start-example.md) 