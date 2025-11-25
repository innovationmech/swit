# 综合安全示例

本示例展示了使用 Swit 框架的综合安全集成，结合了 OAuth2 认证、OPA 授权、mTLS 传输加密、审计日志和安全指标监控。

## 功能特性

- **OAuth2/OIDC 认证**：与 Keycloak 集成实现用户认证
- **OPA 授权**：使用 Open Policy Agent 实现基于策略的访问控制
- **mTLS 支持**：双向 TLS 实现安全传输加密
- **审计日志**：全面的安全事件日志记录
- **安全指标**：Prometheus 指标用于安全监控
- **Grafana 仪表板**：预配置的安全可视化仪表板

## 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        客户端请求                                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    mTLS 层（可选）                               │
│                     证书验证                                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   OAuth2 认证                                    │
│              JWT 令牌验证（Keycloak）                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    OPA 授权                                      │
│              RBAC/ABAC 策略评估                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                  业务逻辑处理器                                   │
│                    文档管理                                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    审计与指标                                    │
│          安全事件 → Prometheus → Grafana                        │
└─────────────────────────────────────────────────────────────────┘
```

## 前置条件

- Docker 和 Docker Compose
- Go 1.23+（用于本地开发）
- curl 或 httpie（用于测试）

## 快速开始

### 1. 启动所有服务

```bash
# 进入示例目录
cd examples/full-security-stack

# 使用 Docker Compose 启动所有服务
docker-compose up -d
```

这将启动：
- **PostgreSQL**（端口 5432）- Keycloak 数据库
- **Keycloak**（端口 8081）- OAuth2/OIDC 提供者
- **OPA**（端口 8181）- Open Policy Agent
- **Prometheus**（端口 9090）- 指标收集
- **Grafana**（端口 3000）- 指标可视化
- **Alertmanager**（端口 9093）- 告警管理
- **应用程序**（端口 8080）- 综合安全示例

### 2. 等待服务启动

```bash
# 检查服务健康状态
docker-compose ps

# 等待 Keycloak 准备就绪（可能需要 1-2 分钟）
docker-compose logs -f keycloak
```

### 3. 访问服务

| 服务 | URL | 凭据 |
|------|-----|------|
| 应用程序 | http://localhost:8080 | - |
| Keycloak 管理控制台 | http://localhost:8081 | admin / admin |
| Grafana | http://localhost:3000 | admin / admin |
| Prometheus | http://localhost:9090 | - |
| OPA | http://localhost:8181 | - |

## 测试 API

### 公开端点

```bash
# 获取服务信息
curl http://localhost:8080/api/v1/public/info

# 获取安全状态
curl http://localhost:8080/api/v1/public/security-status
```

### 认证流程

```bash
# 1. 启动 OAuth2 登录流程
curl http://localhost:8080/api/v1/public/login

# 2. 在浏览器中打开 authorization_url
# 3. 使用 Keycloak 凭据登录（见下方测试用户）
# 4. 使用返回的 access_token 访问受保护的端点
```

### 测试用户（在 Keycloak 中预配置）

| 用户名 | 密码 | 角色 |
|--------|------|------|
| alice | alice123 | admin |
| bob | bob123 | editor |
| charlie | charlie123 | viewer |
| dave | dave123 | guest |

### 受保护的端点

```bash
# 设置访问令牌
TOKEN="your_access_token_here"

# 获取用户资料
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/protected/profile

# 列出文档
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/protected/documents

# 获取特定文档
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/protected/documents/doc-1

# 创建文档（需要 editor 或 admin 角色）
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"doc-5","title":"新文档","content":"内容","classification":"internal"}' \
  http://localhost:8080/api/v1/protected/documents
```

### 管理端点

```bash
# 管理仪表板（需要 admin 角色）
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/admin/dashboard

# 审计日志
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/admin/audit-logs

# 安全指标
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/admin/security-metrics
```

## 配置

### 环境变量

| 变量 | 描述 | 默认值 |
|------|------|--------|
| `HTTP_PORT` | HTTP 服务器端口 | 8080 |
| `OAUTH2_ENABLED` | 启用 OAuth2 认证 | true |
| `OPA_ENABLED` | 启用 OPA 授权 | true |
| `MTLS_ENABLED` | 启用 mTLS | false |
| `AUDIT_ENABLED` | 启用审计日志 | true |
| `METRICS_ENABLED` | 启用安全指标 | true |
| `POLICY_TYPE` | OPA 策略类型（rbac/abac） | rbac |
| `OPA_MODE` | OPA 模式（embedded/remote） | embedded |
| `OAUTH2_ISSUER_URL` | Keycloak 发行者 URL | http://localhost:8081/realms/swit |

### 策略类型

#### RBAC（基于角色的访问控制）

```bash
# 使用 RBAC 策略
docker-compose run -e POLICY_TYPE=rbac app
```

角色：
- **admin**：对所有资源的完全访问权限
- **editor**：创建、读取、更新文档
- **viewer**：只读访问
- **guest**：有限访问

#### ABAC（基于属性的访问控制）

```bash
# 使用 ABAC 策略
docker-compose run -e POLICY_TYPE=abac app
```

考虑的属性：
- 用户角色和部门
- 资源分类和所有者
- 时间（工作时间）
- 客户端 IP 地址

## mTLS 配置

### 生成证书

```bash
# 生成 CA、服务器和客户端证书
./scripts/generate-certs.sh
```

### 启用 mTLS

```bash
# 启用 mTLS 启动
docker-compose -f docker-compose.yml -f docker-compose.mtls.yml up -d

# 或设置环境变量
MTLS_ENABLED=true docker-compose up -d
```

### 测试 mTLS

```bash
# 使用客户端证书测试
curl --cacert certs/ca.crt \
     --cert certs/client.crt \
     --key certs/client.key \
     https://localhost:8443/api/v1/mtls/verify
```

## 监控

### Prometheus 指标

访问 Prometheus：http://localhost:9090

关键指标：
- `security_auth_attempts_total` - 认证尝试
- `security_policy_evaluations_total` - OPA 策略评估
- `security_security_events_total` - 安全事件
- `security_tls_connections_total` - TLS 连接

### Grafana 仪表板

访问 Grafana：http://localhost:3000（admin/admin）

预配置的仪表板：
- **综合安全仪表板**：所有安全指标概览
- 认证指标
- 授权指标
- 安全事件

### 告警

Alertmanager 配置了以下告警：
- 高认证失败率
- 潜在的暴力破解攻击
- 高策略拒绝率
- 严重安全事件
- 证书即将过期

## 项目结构

```
full-security-stack/
├── main.go                 # 主应用程序
├── swit.yaml               # 配置文件
├── docker-compose.yml      # Docker Compose 配置
├── Dockerfile              # 应用程序 Dockerfile
├── policies/               # OPA 策略
│   ├── rbac.rego          # RBAC 策略
│   ├── abac.rego          # ABAC 策略
│   └── security.rego      # 安全检查
├── monitoring/             # 监控配置
│   ├── prometheus.yml     # Prometheus 配置
│   ├── alert-rules.yml    # 告警规则
│   ├── alertmanager.yml   # Alertmanager 配置
│   └── grafana/           # Grafana 仪表板
├── keycloak/              # Keycloak 配置
│   └── realm-export.json  # Realm 导出
├── scripts/               # 工具脚本
│   └── generate-certs.sh  # 证书生成
├── certs/                 # TLS 证书
├── README.md              # 英文文档
└── README-CN.md           # 本文件
```

## 故障排除

### Keycloak 无法启动

```bash
# 检查日志
docker-compose logs keycloak

# 确保 PostgreSQL 健康
docker-compose logs postgres
```

### OPA 策略错误

```bash
# 检查 OPA 日志
docker-compose logs opa

# 手动测试策略
curl -X POST http://localhost:8181/v1/data/rbac/allow \
  -H "Content-Type: application/json" \
  -d '{"input":{"user":{"roles":["admin"]},"request":{"method":"GET","path":"/api/v1/protected/documents"}}}'
```

### 认证问题

```bash
# 验证 Keycloak 可访问
curl http://localhost:8081/realms/swit/.well-known/openid-configuration

# 检查应用程序日志
docker-compose logs app
```

## 开发

### 本地运行

```bash
# 安装依赖
go mod download

# 使用嵌入式 OPA 运行
OAUTH2_ENABLED=false OPA_ENABLED=true go run main.go

# 使用所有功能运行（需要 Keycloak 和 OPA）
go run main.go
```

### 运行测试

```bash
# 运行单元测试
go test -v ./...

# 运行覆盖率测试
go test -cover ./...
```

## 相关文档

- [OAuth2 集成指南](../../docs/oauth2-integration-guide.md)
- [OPA 策略指南](../../docs/opa-policy-guide.md)
- [安全最佳实践](../../docs/security-best-practices.md)
- [安全配置参考](../../docs/security-configuration-reference.md)

## 许可证

本示例是 Swit 框架的一部分，采用 BSD 风格许可证。

