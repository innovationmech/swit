# OPA 授权示例

[English](README.md) | 中文

本示例演示如何在微服务中集成 Open Policy Agent (OPA) 实现细粒度授权，支持嵌入式和远程 OPA 模式以及 RBAC 和 ABAC 策略。

## 功能特性

- ✅ **嵌入式 OPA 模式** - OPA 引擎运行在同一进程中
- ✅ **远程 OPA 模式** - 连接到外部 OPA 服务器
- ✅ **RBAC 策略** - 基于角色的访问控制
- ✅ **ABAC 策略** - 基于属性的访问控制
- ✅ **HTTP 中间件** - 基于 Gin 的策略执行
- ✅ **gRPC 拦截器** - gRPC 策略执行
- ✅ **决策缓存** - 性能优化
- ✅ **审计日志** - 策略决策追踪
- ✅ **Docker 支持** - 完整的容器化部署

## 架构

```
┌─────────────────────────────────────────────────────────┐
│                    客户端请求                             │
└───────────────┬─────────────────────────────────────────┘
                │
                ├─── HTTP (端口 8080)
                │    │
                │    v
                │    ┌──────────────────────────┐
                │    │  Gin 路由器              │
                │    │  + OPA 中间件            │
                │    └───────────┬──────────────┘
                │                │
                └─── gRPC (端口 9090)
                     │
                     v
                     ┌──────────────────────────┐
                     │  gRPC 服务器             │
                     │  + 策略拦截器            │
                     └───────────┬──────────────┘
                                 │
        ┌────────────────────────┴────────────────────────┐
        │                                                  │
        v                                                  v
┌───────────────────┐                           ┌──────────────────┐
│  嵌入式 OPA       │                           │  远程 OPA        │
│  (进程内)         │                           │  (外部服务)      │
│  - RBAC 策略      │                           │  - 端口 8181     │
│  - ABAC 策略      │                           │  - 负载均衡      │
│  - 本地缓存       │                           │  - 健康检查      │
└───────────────────┘                           └──────────────────┘
```

## 快速开始

### 前置要求

- Go 1.23+
- Docker 和 Docker Compose（可选）
- OPA CLI（可选，用于策略测试）

### 1. 使用嵌入式模式运行

```bash
# 进入示例目录
cd examples/opa-authorization

# 使用 RBAC 策略运行
go run main.go -opa-mode embedded -policy-type rbac -policy-dir ./policies

# 使用 ABAC 策略运行
go run main.go -opa-mode embedded -policy-type abac -policy-dir ./policies
```

### 2. 使用远程模式运行（Docker）

```bash
# 启动 OPA 服务器和示例应用
docker-compose up

# 以下服务将可用：
# - OPA 服务器: http://localhost:8181
# - 应用（嵌入式）: http://localhost:8080
# - 应用（远程）: http://localhost:8081
```

### 3. 使用配置文件运行

```bash
# 使用 swit.yaml 配置文件
go run main.go -config swit.yaml
```

## API 测试

### 管理员用户（完全访问权限）

```bash
# 列出所有文档
curl -H "X-User: alice" -H "X-Roles: admin" \
  http://localhost:8080/api/v1/documents

# 创建文档
curl -X POST -H "X-User: alice" -H "X-Roles: admin" \
  -H "Content-Type: application/json" \
  -d '{"id":"doc-4","title":"New Document","content":"Content"}' \
  http://localhost:8080/api/v1/documents

# 删除文档
curl -X DELETE -H "X-User: alice" -H "X-Roles: admin" \
  http://localhost:8080/api/v1/documents/doc-4
```

### 编辑者用户（读取、创建、更新）

```bash
# 读取文档
curl -H "X-User: bob" -H "X-Roles: editor" \
  http://localhost:8080/api/v1/documents

# 创建文档
curl -X POST -H "X-User: bob" -H "X-Roles: editor" \
  -H "Content-Type: application/json" \
  -d '{"id":"doc-5","title":"Bob Document","content":"Content"}' \
  http://localhost:8080/api/v1/documents

# 更新文档
curl -X PUT -H "X-User: bob" -H "X-Roles: editor" \
  -H "Content-Type: application/json" \
  -d '{"title":"Updated Title","content":"Updated Content"}' \
  http://localhost:8080/api/v1/documents/doc-5

# 删除文档（应该被拒绝）
curl -X DELETE -H "X-User: bob" -H "X-Roles: editor" \
  http://localhost:8080/api/v1/documents/doc-5
```

### 查看者用户（仅读取）

```bash
# 读取文档
curl -H "X-User: charlie" -H "X-Roles: viewer" \
  http://localhost:8080/api/v1/documents

# 创建文档（应该被拒绝）
curl -X POST -H "X-User: charlie" -H "X-Roles: viewer" \
  -H "Content-Type: application/json" \
  -d '{"id":"doc-6","title":"Test","content":"Content"}' \
  http://localhost:8080/api/v1/documents
```

### 匿名用户（无访问权限）

```bash
# 访问文档（应该被拒绝，除了健康检查）
curl http://localhost:8080/api/v1/documents

# 健康检查（允许匿名访问）
curl http://localhost:8080/api/v1/health
```

## 配置选项

### 命令行标志

- `-port` - HTTP 服务器端口（默认：8080）
- `-grpc-port` - gRPC 服务器端口（默认：9090）
- `-opa-mode` - OPA 模式：`embedded` 或 `remote`（默认：embedded）
- `-opa-url` - 远程模式的 OPA 服务器 URL（默认：http://localhost:8181）
- `-policy-dir` - 嵌入式模式的策略目录（默认：./policies）
- `-policy-type` - 策略类型：`rbac` 或 `abac`（默认：rbac）
- `-config` - 配置文件路径（默认：swit.yaml）

### 环境变量

- `OPA_MODE` - OPA 模式
- `OPA_URL` - OPA 服务器 URL
- `POLICY_DIR` - 策略目录
- `POLICY_TYPE` - 策略类型

### 配置文件示例（swit.yaml）

参见 [swit.yaml](swit.yaml) 文件，包含完整的 OPA 配置示例。

## 策略详解

### RBAC 策略（基于角色的访问控制）

RBAC 策略定义了基于角色的权限：

#### 角色定义

| 角色 | 权限 | 说明 |
|------|------|------|
| **admin** | 完全访问 | 对所有资源的完全访问权限 |
| **editor** | 读取、创建、更新 | 可以创建、读取和更新文档，但不能删除 |
| **viewer** | 仅读取 | 只能读取文档，不能修改 |
| **owner** | 资源所有者 | 可以对自己拥有的文档执行任何操作 |

#### RBAC 规则示例

```rego
# 管理员拥有所有权限
allow if {
    "admin" in input.subject.roles
}

# 编辑者可以创建、读取、更新文档
allow if {
    "editor" in input.subject.roles
    input.action in ["GET", "POST", "PUT"]
    startswith(input.resource.path, "/api/v1/documents")
}

# 查看者只能读取文档
allow if {
    "viewer" in input.subject.roles
    input.action == "GET"
    startswith(input.resource.path, "/api/v1/documents")
}
```

### ABAC 策略（基于属性的访问控制）

ABAC 策略添加了基于属性的约束，提供更细粒度的访问控制：

#### 属性类型

| 属性类型 | 说明 | 示例 |
|----------|------|------|
| **时间属性** | 访问限制在工作时间（9:00-18:00） | 编辑者只能在工作时间修改文档 |
| **IP 属性** | 访问限制在允许的 IP 范围内 | 只允许内网 IP（192.168.x.x, 172.x.x.x） |
| **资源属性** | 基于资源属性的细粒度控制 | 文档所有者、文档类型等 |
| **上下文属性** | 基于请求上下文的决策 | 请求协议、用户代理等 |

#### ABAC 规则示例

```rego
# 编辑者只能在工作时间和允许的 IP 范围内修改文档
allow if {
    "editor" in input.subject.roles
    input.action in ["POST", "PUT"]
    input.resource.type == "document"
    is_business_hours
    is_allowed_ip
}

# 检查是否是工作时间 (9:00 - 18:00)
is_business_hours if {
    hour := time.clock([input.context.time, "UTC"])[0]
    hour >= 9
    hour < 18
}

# 检查 IP 地址是否在允许的范围内
is_allowed_ip if {
    startswith(input.context.client_ip, "192.168.")
}
```

## 测试

### 单元测试

```bash
# 运行单元测试
go test ./pkg/security/opa/...

# 运行特定测试
go test -run TestEmbeddedClient ./pkg/security/opa/...

# 查看测试覆盖率
go test -cover ./pkg/security/opa/...
```

### 集成测试

```bash
# 运行集成测试（需要 integration 标签）
go test -tags=integration ./pkg/security/opa/...

# 使用远程 OPA（需要 OPA 服务器运行）
OPA_URL=http://localhost:8181 go test -tags=integration ./pkg/security/opa/...
```

### 策略测试

```bash
# 使用 OPA CLI 测试策略
cd policies

# 测试 RBAC 策略
opa eval -d rbac.rego 'data.rbac.allow' \
  -i test_input_rbac.json

# 测试 ABAC 策略
opa eval -d abac.rego 'data.abac.allow' \
  -i test_input_abac.json

# 检查策略语法
opa check rbac.rego abac.rego
```

### 基准测试

```bash
# 运行性能基准测试
go test -bench=. -benchmem ./pkg/security/opa/...

# 运行特定基准测试
go test -bench=BenchmarkEmbeddedClientEvaluate ./pkg/security/opa/...
```

## 性能

示例展示了 OPA 的高性能策略评估：

| 指标 | 嵌入式模式 | 远程模式 |
|------|------------|----------|
| **P50 延迟** | < 1ms | < 10ms |
| **P99 延迟** | < 5ms | < 50ms |
| **吞吐量** | 10,000+ 决策/秒 | 5,000+ 决策/秒 |
| **缓存命中率** | > 90% | > 85% |

### 性能优化建议

1. **启用决策缓存** - 减少重复策略评估
2. **使用嵌入式模式** - 避免网络开销
3. **优化策略规则** - 减少计算复杂度
4. **批量评估** - 一次评估多个决策

## 目录结构

```
examples/opa-authorization/
├── main.go                 # 主应用程序
├── README.md              # 英文文档
├── README-CN.md           # 中文文档
├── swit.yaml              # 配置文件示例
├── Dockerfile             # 容器镜像定义
├── docker-compose.yml     # 多容器配置
└── policies/              # OPA 策略
    ├── rbac.rego          # RBAC 策略
    └── abac.rego          # ABAC 策略
```

## 故障排查

### OPA 服务器连接失败

```bash
# 检查 OPA 服务器是否运行
curl http://localhost:8181/health

# 查看 OPA 服务器日志
docker logs opa-server

# 手动启动 OPA 服务器
docker run -p 8181:8181 openpolicyagent/opa:latest run --server --log-level=debug
```

### 策略评估失败

```bash
# 使用 OPA CLI 测试策略
cd policies
opa eval -d rbac.rego 'data.rbac.allow' -i <(echo '
{
  "subject": {"user": "alice", "roles": ["admin"]},
  "action": "GET",
  "resource": {"path": "/api/v1/documents", "type": "document"}
}')

# 检查策略语法
opa check policies/

# 查看策略评估详细信息
opa eval -d rbac.rego 'data.rbac' --explain full -i input.json
```

### 意外的权限拒绝

常见原因和解决方法：

1. **请求头缺失或错误**
   - 检查 `X-User` 和 `X-Roles` 请求头
   - 确保角色名称拼写正确

2. **策略路径不匹配**
   - 检查策略决策路径配置
   - 确认使用正确的策略包名（rbac/abac）

3. **输入数据格式错误**
   - 查看审计日志了解决策原因
   - 验证策略输入数据结构

4. **缓存问题**
   - 清除决策缓存并重试
   - 检查缓存 TTL 配置

### 性能问题

```bash
# 查看策略评估性能
go test -bench=BenchmarkEmbeddedClientEvaluate -benchmem ./pkg/security/opa/

# 启用详细日志查看性能瓶颈
go run main.go -opa-mode embedded --log-level=debug

# 监控 OPA 服务器性能（远程模式）
curl http://localhost:8181/metrics
```

## 最佳实践

### 1. 策略设计

- **最小权限原则** - 默认拒绝，显式授权
- **角色分层** - 合理设计角色层次
- **策略模块化** - 将策略拆分为多个可重用的模块
- **审计日志** - 记录所有决策用于审计

### 2. 性能优化

- **启用缓存** - 对于不经常变化的决策
- **使用嵌入式模式** - 当策略不需要热更新时
- **批量评估** - 一次评估多个相关决策
- **优化策略规则** - 避免复杂的循环和递归

### 3. 安全考虑

- **敏感数据保护** - 不要在策略输入中包含敏感信息
- **策略版本控制** - 使用 Git 管理策略版本
- **策略测试** - 编写完整的策略测试用例
- **定期审计** - 定期审查策略决策日志

### 4. 运维建议

- **监控和告警** - 监控策略评估性能和错误率
- **策略热更新** - 使用远程模式支持策略热更新
- **灾难恢复** - 准备策略回滚方案
- **文档维护** - 保持策略文档与代码同步

## 扩展示例

### 多租户场景

参见 `pkg/security/opa/policies/examples/multi_tenant_abac.rego`，演示如何实现多租户隔离。

### 金融系统场景

参见 `pkg/security/opa/policies/examples/financial_abac.rego`，演示如何实现金融级别的访问控制。

### 医疗系统场景

参见 `pkg/security/opa/policies/examples/healthcare_abac.rego`，演示如何实现医疗数据的访问控制。

## 参考资料

### 官方文档

- [OPA 官方文档](https://www.openpolicyagent.org/docs/)
- [Rego 语言参考](https://www.openpolicyagent.org/docs/latest/policy-language/)
- [OPA REST API](https://www.openpolicyagent.org/docs/latest/rest-api/)

### 项目文档

- [OPA RBAC 指南](../../docs/opa-rbac-guide.md)
- [OPA ABAC 指南](../../docs/opa-abac-guide.md)
- [OPA 策略编写指南](../../docs/opa-policy-guide.md)
- [安全配置参考](../../docs/security-configuration-reference.md)

### 相关示例

- [OAuth2 认证示例](../oauth2-authentication/)
- [mTLS 示例](../security/)
- [综合安全示例](../security/)

## 贡献

欢迎贡献更多的策略示例和使用场景！请参考 [贡献指南](../../CONTRIBUTING.md)。

## 许可证

Apache License 2.0 - 详见 [LICENSE](../../LICENSE) 文件。

