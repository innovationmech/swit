# OPA Authorization Example / OPA 授权示例

[English](#english) | [中文](#中文)

---

## English

This example demonstrates how to integrate Open Policy Agent (OPA) for fine-grained authorization in a microservice, supporting both embedded and remote OPA modes with RBAC and ABAC policies.

### Features

- ✅ **Embedded OPA Mode** - OPA engine runs in the same process
- ✅ **Remote OPA Mode** - Connect to external OPA server
- ✅ **RBAC Policies** - Role-based access control
- ✅ **ABAC Policies** - Attribute-based access control
- ✅ **HTTP Middleware** - Gin-based policy enforcement
- ✅ **gRPC Interceptor** - gRPC policy enforcement
- ✅ **Decision Caching** - Performance optimization
- ✅ **Audit Logging** - Policy decision tracking
- ✅ **Docker Support** - Complete containerized setup

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Client Requests                       │
└───────────────┬─────────────────────────────────────────┘
                │
                ├─── HTTP (Port 8080)
                │    │
                │    v
                │    ┌──────────────────────────┐
                │    │  Gin Router              │
                │    │  + OPA Middleware        │
                │    └───────────┬──────────────┘
                │                │
                └─── gRPC (Port 9090)
                     │
                     v
                     ┌──────────────────────────┐
                     │  gRPC Server             │
                     │  + Policy Interceptor    │
                     └───────────┬──────────────┘
                                 │
        ┌────────────────────────┴────────────────────────┐
        │                                                  │
        v                                                  v
┌───────────────────┐                           ┌──────────────────┐
│  Embedded OPA     │                           │  Remote OPA      │
│  (In-Process)     │                           │  (External)      │
│  - RBAC Policy    │                           │  - Port 8181     │
│  - ABAC Policy    │                           │  - Load Balanced │
│  - Local Cache    │                           │  - Health Check  │
└───────────────────┘                           └──────────────────┘
```

### Quick Start

#### Prerequisites

- Go 1.23+
- Docker and Docker Compose (optional)
- OPA CLI (optional, for policy testing)

#### 1. Run with Embedded Mode

```bash
# Navigate to the example directory
cd examples/opa-authorization

# Run with RBAC policy
go run main.go -opa-mode embedded -policy-type rbac -policy-dir ./policies

# Run with ABAC policy
go run main.go -opa-mode embedded -policy-type abac -policy-dir ./policies
```

#### 2. Run with Remote Mode (Docker)

```bash
# Start OPA server and example applications
docker-compose up

# The following services will be available:
# - OPA Server: http://localhost:8181
# - App (Embedded): http://localhost:8080
# - App (Remote): http://localhost:8081
```

#### 3. Test the API

**Admin user (full access):**

```bash
curl -H "X-User: alice" -H "X-Roles: admin" \
  http://localhost:8080/api/v1/documents
```

**Editor user (read, create, update):**

```bash
curl -H "X-User: bob" -H "X-Roles: editor" \
  http://localhost:8080/api/v1/documents
```

**Viewer user (read only):**

```bash
curl -H "X-User: charlie" -H "X-Roles: viewer" \
  http://localhost:8080/api/v1/documents
```

**Try forbidden action (should fail):**

```bash
curl -X DELETE -H "X-User: charlie" -H "X-Roles: viewer" \
  http://localhost:8080/api/v1/documents/doc-1
```

### Configuration Options

#### Command-Line Flags

- `-port` - HTTP server port (default: 8080)
- `-grpc-port` - gRPC server port (default: 9090)
- `-opa-mode` - OPA mode: `embedded` or `remote` (default: embedded)
- `-opa-url` - OPA server URL for remote mode (default: http://localhost:8181)
- `-policy-dir` - Policy directory for embedded mode (default: ./policies)
- `-policy-type` - Policy type: `rbac` or `abac` (default: rbac)

#### Environment Variables

- `OPA_MODE` - OPA mode
- `OPA_URL` - OPA server URL
- `POLICY_DIR` - Policy directory
- `POLICY_TYPE` - Policy type

### Policy Examples

#### RBAC Policy

The RBAC policy defines role-based permissions:

- **admin** - Full access to all resources
- **editor** - Can create, read, and update documents
- **viewer** - Can only read documents
- **Resource owner** - Can perform any action on their own documents

#### ABAC Policy

The ABAC policy adds attribute-based constraints:

- **Time-based** - Access restricted to business hours (9:00-18:00)
- **IP-based** - Access restricted to allowed IP ranges
- **Resource attributes** - Fine-grained control based on resource properties
- **Context-aware** - Decisions based on request context

### Testing

#### Unit Tests

```bash
# Run unit tests
go test ./pkg/security/opa/...
```

#### Integration Tests

```bash
# Run integration tests (requires integration tag)
go test -tags=integration ./pkg/security/opa/...

# With remote OPA (requires OPA server running)
OPA_URL=http://localhost:8181 go test -tags=integration ./pkg/security/opa/...
```

#### Benchmark Tests

```bash
# Run performance benchmarks
go test -bench=. -benchmem ./pkg/security/opa/...
```

### Performance

The example demonstrates OPA's high-performance policy evaluation:

- **P99 latency** - < 5ms for embedded mode
- **Throughput** - 10,000+ decisions/sec (embedded, cached)
- **Cache hit ratio** - > 90% with proper cache configuration

### Directory Structure

```
examples/opa-authorization/
├── main.go                 # Main application
├── README.md              # This file
├── Dockerfile             # Container image definition
├── docker-compose.yml     # Multi-container setup
└── policies/              # OPA policies
    ├── rbac.rego          # RBAC policy
    └── abac.rego          # ABAC policy
```

### Troubleshooting

#### OPA Server Connection Failed

```bash
# Check if OPA server is running
curl http://localhost:8181/health

# Start OPA server manually
docker run -p 8181:8181 openpolicyagent/opa:latest run --server
```

#### Policy Evaluation Failed

```bash
# Test policy with OPA CLI
opa eval -d policies/rbac.rego 'data.rbac.allow' -i input.json

# Check policy syntax
opa check policies/
```

#### Permission Denied Unexpectedly

- Check the request headers (`X-User`, `X-Roles`)
- Review audit logs for decision reasons
- Verify policy rules match your use case

### References

- [OPA Documentation](https://www.openpolicyagent.org/docs/)
- [Rego Language](https://www.openpolicyagent.org/docs/latest/policy-language/)
- [OPA RBAC Guide](../../docs/opa-rbac-guide.md)
- [OPA ABAC Guide](../../docs/opa-abac-guide.md)

---

## 中文

本示例演示如何在微服务中集成 Open Policy Agent (OPA) 实现细粒度授权，支持嵌入式和远程 OPA 模式以及 RBAC 和 ABAC 策略。

### 功能特性

- ✅ **嵌入式 OPA 模式** - OPA 引擎运行在同一进程中
- ✅ **远程 OPA 模式** - 连接到外部 OPA 服务器
- ✅ **RBAC 策略** - 基于角色的访问控制
- ✅ **ABAC 策略** - 基于属性的访问控制
- ✅ **HTTP 中间件** - 基于 Gin 的策略执行
- ✅ **gRPC 拦截器** - gRPC 策略执行
- ✅ **决策缓存** - 性能优化
- ✅ **审计日志** - 策略决策追踪
- ✅ **Docker 支持** - 完整的容器化部署

### 架构

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

### 快速开始

#### 前置要求

- Go 1.23+
- Docker 和 Docker Compose（可选）
- OPA CLI（可选，用于策略测试）

#### 1. 使用嵌入式模式运行

```bash
# 进入示例目录
cd examples/opa-authorization

# 使用 RBAC 策略运行
go run main.go -opa-mode embedded -policy-type rbac -policy-dir ./policies

# 使用 ABAC 策略运行
go run main.go -opa-mode embedded -policy-type abac -policy-dir ./policies
```

#### 2. 使用远程模式运行（Docker）

```bash
# 启动 OPA 服务器和示例应用
docker-compose up

# 以下服务将可用：
# - OPA 服务器: http://localhost:8181
# - 应用（嵌入式）: http://localhost:8080
# - 应用（远程）: http://localhost:8081
```

#### 3. 测试 API

**管理员用户（完全访问权限）：**

```bash
curl -H "X-User: alice" -H "X-Roles: admin" \
  http://localhost:8080/api/v1/documents
```

**编辑者用户（读取、创建、更新）：**

```bash
curl -H "X-User: bob" -H "X-Roles: editor" \
  http://localhost:8080/api/v1/documents
```

**查看者用户（仅读取）：**

```bash
curl -H "X-User: charlie" -H "X-Roles: viewer" \
  http://localhost:8080/api/v1/documents
```

**尝试禁止的操作（应该失败）：**

```bash
curl -X DELETE -H "X-User: charlie" -H "X-Roles: viewer" \
  http://localhost:8080/api/v1/documents/doc-1
```

### 配置选项

#### 命令行标志

- `-port` - HTTP 服务器端口（默认：8080）
- `-grpc-port` - gRPC 服务器端口（默认：9090）
- `-opa-mode` - OPA 模式：`embedded` 或 `remote`（默认：embedded）
- `-opa-url` - 远程模式的 OPA 服务器 URL（默认：http://localhost:8181）
- `-policy-dir` - 嵌入式模式的策略目录（默认：./policies）
- `-policy-type` - 策略类型：`rbac` 或 `abac`（默认：rbac）

#### 环境变量

- `OPA_MODE` - OPA 模式
- `OPA_URL` - OPA 服务器 URL
- `POLICY_DIR` - 策略目录
- `POLICY_TYPE` - 策略类型

### 策略示例

#### RBAC 策略

RBAC 策略定义了基于角色的权限：

- **admin** - 对所有资源的完全访问权限
- **editor** - 可以创建、读取和更新文档
- **viewer** - 只能读取文档
- **资源所有者** - 可以对自己的文档执行任何操作

#### ABAC 策略

ABAC 策略添加了基于属性的约束：

- **基于时间** - 访问限制在工作时间（9:00-18:00）
- **基于 IP** - 访问限制在允许的 IP 范围内
- **资源属性** - 基于资源属性的细粒度控制
- **上下文感知** - 基于请求上下文的决策

### 测试

#### 单元测试

```bash
# 运行单元测试
go test ./pkg/security/opa/...
```

#### 集成测试

```bash
# 运行集成测试（需要 integration 标签）
go test -tags=integration ./pkg/security/opa/...

# 使用远程 OPA（需要 OPA 服务器运行）
OPA_URL=http://localhost:8181 go test -tags=integration ./pkg/security/opa/...
```

#### 基准测试

```bash
# 运行性能基准测试
go test -bench=. -benchmem ./pkg/security/opa/...
```

### 性能

示例展示了 OPA 的高性能策略评估：

- **P99 延迟** - 嵌入式模式 < 5ms
- **吞吐量** - 10,000+ 决策/秒（嵌入式，有缓存）
- **缓存命中率** - 正确配置下 > 90%

### 目录结构

```
examples/opa-authorization/
├── main.go                 # 主应用程序
├── README.md              # 本文件
├── Dockerfile             # 容器镜像定义
├── docker-compose.yml     # 多容器配置
└── policies/              # OPA 策略
    ├── rbac.rego          # RBAC 策略
    └── abac.rego          # ABAC 策略
```

### 故障排查

#### OPA 服务器连接失败

```bash
# 检查 OPA 服务器是否运行
curl http://localhost:8181/health

# 手动启动 OPA 服务器
docker run -p 8181:8181 openpolicyagent/opa:latest run --server
```

#### 策略评估失败

```bash
# 使用 OPA CLI 测试策略
opa eval -d policies/rbac.rego 'data.rbac.allow' -i input.json

# 检查策略语法
opa check policies/
```

#### 意外的权限拒绝

- 检查请求头（`X-User`、`X-Roles`）
- 查看审计日志了解决策原因
- 验证策略规则是否匹配您的用例

### 参考资料

- [OPA 文档](https://www.openpolicyagent.org/docs/)
- [Rego 语言](https://www.openpolicyagent.org/docs/latest/policy-language/)
- [OPA RBAC 指南](../../docs/opa-rbac-guide.md)
- [OPA ABAC 指南](../../docs/opa-abac-guide.md)

