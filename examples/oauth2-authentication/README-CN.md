# OAuth2/OIDC 认证示例

本示例演示如何在 Swit 框架服务中实现 OAuth2/OIDC 认证。展示以下功能:

- **授权码流程** 支持 PKCE (Proof Key for Code Exchange)
- **OIDC 发现** 自动配置端点
- **JWT 令牌验证** 支持本地和远程内省
- **令牌刷新** 流程
- **受保护端点** 使用 OAuth2 中间件
- **基于角色的访问控制** (RBAC)
- **Keycloak 集成** 作为 OIDC 提供商

## 功能特性

### 认证流程

- ✅ **授权码流程 + PKCE** - 最安全的 OAuth2 流程，适用于 Web 应用
- ✅ **OIDC 发现** - 从提供商元数据自动配置
- ✅ **令牌刷新** - 无需重新认证的令牌更新
- ✅ **令牌撤销** - 带令牌清理的正确登出

### 安全特性

- ✅ **JWT 验证** - 使用 JWKS 的本地令牌验证
- ✅ **令牌内省** - 通过提供商的远程令牌验证
- ✅ **状态和随机数验证** - CSRF 和重放攻击防护
- ✅ **PKCE 支持** - 公共客户端的增强安全性
- ✅ **基于角色的访问控制** - 细粒度授权

### 集成特性

- ✅ **Keycloak 集成** - 预配置的 realm 和客户端
- ✅ **中间件支持** - 轻松保护端点
- ✅ **可选认证** - 混合公共/私有端点
- ✅ **Docker Compose 设置** - 一键部署

## 快速开始

### 前置要求

- Docker 和 Docker Compose
- Go 1.23+ (用于本地开发)
- curl 或 Postman (用于测试)

### 1. 启动服务

```bash
cd examples/oauth2-authentication
docker-compose up -d
```

这将启动:
- **Keycloak** 在 http://localhost:8081 (OIDC 提供商)
- **PostgreSQL** 在 localhost:5432 (Keycloak 数据库)
- **OAuth2 示例服务** 在 http://localhost:8080

等待所有服务健康检查通过 (Keycloak 约需 60 秒):

```bash
docker-compose ps
```

### 2. 验证 Keycloak 运行

访问 Keycloak 管理控制台:
- URL: http://localhost:8081/admin
- 用户名: `admin`
- 密码: `admin`

示例预配置了名为 **swit** 的 realm，包含:
- 客户端: `swit-example` (客户端密钥: `swit-example-secret`)
- 两个用户:
  - `testuser` / `password` (角色: user)
  - `admin` / `admin` (角色: user, admin)

### 3. 测试认证流程

#### 步骤 1: 获取服务信息

```bash
curl http://localhost:8080/api/v1/public/info
```

#### 步骤 2: 发起登录

```bash
curl http://localhost:8080/api/v1/public/login
```

这将返回一个授权 URL。复制 `authorization_url` 并在浏览器中打开。

#### 步骤 3: 使用 Keycloak 登录

在浏览器中:
1. 输入凭据: `testuser` / `password`
2. 你将被重定向到回调 URL
3. 复制包含 `access_token` 的 JSON 响应

#### 步骤 4: 访问受保护端点

使用步骤 3 中的访问令牌:

```bash
export ACCESS_TOKEN="your-access-token-here"

curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/api/v1/protected/profile
```

### 4. 测试令牌刷新

```bash
export REFRESH_TOKEN="your-refresh-token-here"

curl -X POST http://localhost:8080/api/v1/public/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}"
```

### 5. 测试基于角色的访问

使用普通用户令牌尝试访问管理员端点:

```bash
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/api/v1/admin/dashboard
```

应该返回 403 Forbidden。现在以管理员身份登录再试。

## API 端点

### 公共端点 (无需认证)

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/public/info` | 服务信息 |
| GET | `/api/v1/public/login` | 发起 OAuth2 登录 |
| GET | `/api/v1/public/callback` | OAuth2 回调处理 |
| POST | `/api/v1/public/refresh` | 刷新访问令牌 |
| POST | `/api/v1/public/logout` | 撤销令牌并登出 |

### 受保护端点 (需要认证)

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/protected/profile` | 用户配置文件 |
| GET | `/api/v1/protected/data` | 受保护数据 |

### 可选认证端点

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/optional/content` | 认证和匿名用户都可访问的内容 |

### 管理员端点 (需要管理员角色)

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/admin/dashboard` | 管理员仪表板 |

## 配置

### 环境变量

使用环境变量覆盖配置:

```bash
# OAuth2 配置
export OAUTH2_PROVIDER="keycloak"
export OAUTH2_CLIENT_ID="swit-example"
export OAUTH2_CLIENT_SECRET="swit-example-secret"
export OAUTH2_ISSUER_URL="http://localhost:8081/realms/swit"
export OAUTH2_REDIRECT_URL="http://localhost:8080/api/v1/public/callback"

# 服务器配置
export HTTP_PORT="8080"
```

### 配置文件

编辑 `swit.yaml` 自定义服务:

```yaml
oauth2:
  enabled: true
  provider: "keycloak"
  client_id: "swit-example"
  client_secret: "your-secret"
  issuer_url: "http://localhost:8081/realms/swit"
  use_discovery: true
  scopes:
    - "openid"
    - "profile"
    - "email"
```

## 本地运行 (不使用 Docker)

### 1. 启动 Keycloak

```bash
docker-compose up -d keycloak postgres
```

### 2. 运行示例服务

```bash
cd examples/oauth2-authentication

# 设置环境变量
export OAUTH2_CLIENT_SECRET="swit-example-secret"

# 运行服务
go run main.go
```

## 使用不同提供商测试

### 使用 Auth0

1. 创建 Auth0 账号和应用
2. 更新 `swit.yaml`:

```yaml
oauth2:
  provider: "auth0"
  client_id: "your-auth0-client-id"
  client_secret: "your-auth0-client-secret"
  issuer_url: "https://your-tenant.auth0.com/"
  redirect_url: "http://localhost:8080/api/v1/public/callback"
```

### 使用 Google

1. 创建 Google OAuth2 凭据
2. 更新 `swit.yaml`:

```yaml
oauth2:
  provider: "google"
  client_id: "your-google-client-id.apps.googleusercontent.com"
  client_secret: "your-google-client-secret"
  issuer_url: "https://accounts.google.com"
  redirect_url: "http://localhost:8080/api/v1/public/callback"
```

## 集成测试

运行 OAuth2 集成测试:

```bash
# 运行集成测试
cd ../../
go test ./pkg/security/oauth2 -v -run Integration

# 运行基准测试
go test ./pkg/security/oauth2 -bench=. -benchmem
```

## 安全考虑

### 生产部署

1. **使用 HTTPS**: 生产环境始终使用 TLS
2. **保护密钥**: 将客户端密钥存储在密钥管理系统中
3. **令牌存储**: 使用安全的 HTTP-only cookie 存储令牌
4. **CORS 配置**: 限制允许的来源
5. **速率限制**: 在认证端点上实施速率限制
6. **审计日志**: 记录所有认证事件

### 令牌管理

1. **令牌过期**: 使用短期访问令牌 (5-15 分钟)
2. **刷新令牌轮换**: 每次使用时轮换刷新令牌
3. **令牌撤销**: 实施带令牌撤销的正确登出
4. **令牌缓存**: 缓存已验证的令牌以减少提供商负载

## 故障排除

### Keycloak 无法启动

检查日志:
```bash
docker-compose logs keycloak
```

等待 "Keycloak started" 消息。

### 无效的重定向 URI

确保 Keycloak 中的重定向 URI 完全匹配:
1. 访问 http://localhost:8081/admin
2. 导航到 Clients → swit-example
3. 检查 "Valid Redirect URIs" 包含 `http://localhost:8080/*`

### 令牌验证失败

1. 检查配置和 Keycloak 之间的 issuer URL 是否匹配
2. 验证 JWKS 端点可访问
3. 检查系统时钟是否同步

### 连接 Keycloak 被拒绝

如果在本地运行示例服务 (不在 Docker 中)，使用:
```bash
export OAUTH2_ISSUER_URL="http://localhost:8081/realms/swit"
```

而不是 Docker 内部主机名。

## 架构

```
┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│   浏览器    │◄───────►│  OAuth2 服务 │◄───────►│  Keycloak    │
│             │         │              │         │  (OIDC)      │
└─────────────┘         └──────────────┘         └──────────────┘
                               │
                               ▼
                        ┌──────────────┐
                        │ JWT 验证器   │
                        │  (with JWKS) │
                        └──────────────┘
```

### 流程图

```
1. 用户 → GET /login
2. 服务 → 重定向到 Keycloak
3. 用户 → 在 Keycloak 登录
4. Keycloak → 重定向到 /callback 带 code
5. 服务 → 交换 code 获取令牌
6. 服务 → 返回令牌给用户
7. 用户 → 使用 access_token 访问受保护端点
8. 服务 → 本地或通过内省验证 JWT
9. 服务 → 返回受保护资源
```

## 扩展阅读

- [OAuth 2.0 规范 (RFC 6749)](https://datatracker.ietf.org/doc/html/rfc6749)
- [OIDC 核心规范](https://openid.net/specs/openid-connect-core-1_0.html)
- [PKCE 规范 (RFC 7636)](https://datatracker.ietf.org/doc/html/rfc7636)
- [JWT 规范 (RFC 7519)](https://datatracker.ietf.org/doc/html/rfc7519)
- [Keycloak 文档](https://www.keycloak.org/documentation)

## 许可证

Copyright 2025 Swit. All rights reserved.

