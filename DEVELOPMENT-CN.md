# 开发指南

本文档描述了 Swit 项目的开发工作流程和质量标准。

## 快速开始

### 设置开发环境

```bash
# 安装开发工具和Git钩子
make setup-dev
```

这将安装：
- swag工具用于生成Swagger文档
- 自动质量检查的预提交钩子

### 构建和测试

```bash
# 运行完整的质量检查和构建
make all

# 或运行单个步骤
make tidy      # 运行go mod tidy
make format    # 格式化代码
make vet       # 运行go vet
make quality   # 运行所有质量检查
make test      # 运行测试
make build     # 构建二进制文件
```

## 开发工作流程

### 1. 代码质量

所有代码在提交前必须通过以下质量检查：

- **Tidy**: 依赖项必须使用`go mod tidy`进行清理
- **Format**: 代码必须使用`gofmt`进行格式化
- **Vet**: 代码必须通过`go vet`分析
- **Tests**: 所有测试必须通过

### 2. 预提交钩子

预提交钩子会自动运行：
- `go mod tidy`
- 使用`gofmt`进行代码格式化
- `go vet`分析
- 受影响包的测试

### 3. 持续集成

我们的CI管道在每次推送和拉取请求时运行：

1. **Tidy阶段**: 使用`go mod tidy`清理依赖项
2. **质量阶段**: 格式化和vet检查
3. **测试阶段**: 单元测试与竞态检测和覆盖率
4. **构建阶段**: 构建所有二进制文件
5. **文档阶段**: 生成Swagger文档

## Make 目标

| 目标 | 描述 |
|------|------|
| `make all` | 运行完整的构建流水线（tidy + copyright + build + swagger） |
| `make tidy` | 运行go mod tidy |
| `make format` | 使用gofmt格式化代码 |
| `make vet` | 运行go vet |
| `make quality` | 运行所有质量检查（format + vet） |
| `make build` | 构建所有二进制文件 |
| `make clean` | 删除输出二进制文件 |
| `make test` | 运行单元测试 |
| `make test-pkg` | 仅运行pkg包测试 |
| `make test-internal` | 仅运行internal包测试 |
| `make test-coverage` | 运行测试并生成覆盖率报告 |
| `make test-race` | 运行竞态检测测试 |
| `make image-serve` | 构建swit-serve的Docker镜像 |
| `make image-auth` | 构建swit-auth的Docker镜像 |
| `make image-all` | 构建所有服务的Docker镜像 |
| `make swagger` | 生成/更新所有服务的Swagger文档 |
| `make swagger-switserve` | 仅生成switserve的Swagger文档 |
| `make swagger-switauth` | 仅生成switauth的Swagger文档 |
| `make ci` | 运行完整的CI流水线 |
| `make setup-dev` | 设置开发环境 |

## 代码标准

### Go代码风格

- 遵循标准Go约定
- 使用`gofmt`进行格式化
- 编写清晰、自文档化的代码
- 为新功能包含单元测试
- 使用zap进行结构化日志记录

### Git提交消息

- 使用conventional commit格式：`type(scope): description`
- 示例：
  - `feat(user): add user authentication`
  - `fix(api): resolve rate limiting issue`
  - `docs(readme): update installation instructions`

### 测试

- 为所有新功能编写单元测试
- 力求良好的测试覆盖率（使用`make test-coverage`检查）
- 在适当的地方使用表驱动测试
- 模拟外部依赖

## Docker镜像

项目支持为两个服务构建Docker镜像：

```bash
# 构建单个服务镜像
make image-serve    # 构建swit-serve镜像
make image-auth     # 构建swit-auth镜像

# 构建所有镜像
make image-all
```

镜像使用当前git分支名称作为标签。

## Swagger文档

项目为两个服务生成Swagger文档：

```bash
# 生成所有文档
make swagger

# 为特定服务生成
make swagger-switserve   # 为switserve生成
make swagger-switauth    # 为switauth生成
```

文档生成位置：
- `docs/generated/switserve/` - SwitServe API 文档
- `docs/generated/switauth/` - SwitAuth API 文档
- `docs/generated/` - 统一文档根目录

## 依赖项

Swit 框架使用以下主要依赖项：

### 核心框架
- **Go 1.24+**: 要求的 Go 版本
- **gin-gonic/gin**: HTTP Web 框架
- **grpc/grpc-go**: gRPC 框架
- **uber-go/zap**: 结构化日志
- **spf13/viper**: 配置管理
- **spf13/cobra**: CLI 框架（用于 switctl）

### 安全与认证
- **golang.org/x/oauth2**: OAuth2 客户端库，支持标准 OAuth2 流程
- **coreos/go-oidc/v3**: OpenID Connect (OIDC) 支持和发现
- **golang-jwt/jwt/v5**: JWT 令牌验证和管理
- **golang.org/x/crypto**: 加密工具
- **open-policy-agent/opa**: 策略引擎，支持 RBAC/ABAC 授权
- **hashicorp/vault/api**: 密钥管理（可选）

### 消息传递与流式处理
- **nats-io/nats.go**: NATS 消息传递
- **streadway/amqp**: RabbitMQ 客户端
- **segmentio/kafka-go**: Kafka 客户端

### 可观测性
- **opentelemetry**: 分布式追踪
- **prometheus/client_golang**: 指标收集
- **getsentry/sentry-go**: 错误追踪

### 数据库与存储
- **gorm.io/gorm**: ORM 框架
- **redis/go-redis**: Redis 客户端
- **lib/pq**: PostgreSQL 驱动

### 服务发现
- **hashicorp/consul/api**: Consul 集成

完整的依赖项列表请查看 `go.mod`。

## 项目结构

```
├── cmd/                    # 应用程序入口点
├── internal/              # 私有应用程序代码
│   ├── switserve/        # 主服务器应用程序
│   └── switauth/         # 认证服务
├── pkg/                   # 公共库代码
│   ├── server/           # 基础服务器框架
│   ├── transport/        # HTTP/gRPC 传输层
│   ├── middleware/       # HTTP/gRPC 中间件
│   ├── security/         # 安全组件
│   │   ├── oauth2/       # OAuth2/OIDC 客户端
│   │   ├── opa/          # OPA 策略引擎
│   │   ├── jwt/          # JWT 验证
│   │   ├── tls/          # TLS 配置
│   │   ├── audit/        # 审计日志
│   │   └── secrets/      # 密钥管理
│   ├── saga/             # 分布式事务
│   └── ...
├── api/                   # API定义（protobuf、OpenAPI）
├── scripts/              # 构建和实用脚本
├── build/                # 构建配置（Docker等）
└── _output/              # 构建产物（生成的）
```

## 故障排除

### 质量问题

如果遇到质量问题：

1. 运行`make tidy`清理依赖项
2. 运行`make format`修复格式问题
3. 运行`make vet`检查潜在问题
4. 手动修复问题或使用IDE建议
5. 某些问题可能需要代码重构

### 预提交钩子问题

如果预提交钩子导致问题：

```bash
# 临时跳过钩子进行紧急修复
git commit --no-verify -m "urgent fix"

# 或删除并重新安装钩子
rm .git/hooks/pre-commit
make install-hooks
```

### CI流水线失败

1. 检查失败的具体阶段
2. 在本地运行相同的命令：
   ```bash
   make ci  # 在本地运行完整CI流水线
   ```
3. 修复问题并重新推送

## 安全开发

### 安全测试

```bash
# 运行安全扫描
make security              # 完整安全扫描（gosec + trivy + govulncheck）

# 单独扫描
gosec ./...                # 静态代码分析
govulncheck ./...          # Go 漏洞检查
trivy fs .                 # 文件系统漏洞扫描
```

### 安全最佳实践

开发安全相关功能时：

1. **不要提交密钥** - 使用环境变量或密钥管理
2. **验证所有输入** - 使用结构体标签和验证器
3. **使用参数化查询** - 防止 SQL 注入
4. **生产环境启用 TLS** - 最低 TLS 1.2
5. **实现正确的日志记录** - 审计安全事件，不记录敏感数据
6. **遵循 OWASP 指南** - 参考 OWASP Top 10

### 安全配置

`swit.yaml` 中的安全配置示例：

```yaml
security:
  oauth2:
    enabled: true
    provider: keycloak
    client_id: my-service
    client_secret: ${OAUTH2_CLIENT_SECRET}
    issuer_url: https://auth.example.com/realms/production
    
  opa:
    enabled: true
    mode: embedded
    policy_dir: ./policies
    
  tls:
    enabled: true
    cert_file: /etc/ssl/certs/server.crt
    key_file: /etc/ssl/private/server.key
    min_version: "1.2"
```

### 安全文档

- [安全最佳实践](docs/security-best-practices.md)
- [OAuth2 集成指南](docs/oauth2-integration-guide.md)
- [OPA 策略指南](docs/opa-policy-guide.md)
- [安全检查清单](docs/security-checklist.md)

## 获取帮助

- 运行`make help`查看所有可用目标
- 查看现有代码的示例和模式
- 查看CI日志获取详细错误消息