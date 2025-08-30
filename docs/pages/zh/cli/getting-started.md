# switctl 入门指南

本教程将指导您使用 switctl 创建第一个微服务，并探索 CLI 工具的主要功能。

## 前置要求

- Go 1.23.12 或更新版本
- Git
- 对微服务的基本了解

## 安装

### 1. 从源码构建

```bash
# 克隆 Swit 框架
git clone https://github.com/innovationmech/swit.git
cd swit

# 构建 switctl CLI
make build

# 验证安装
./bin/switctl --version
```

### 2. 添加到 PATH

```bash
# 添加到您的 shell 配置文件 (~/.bashrc, ~/.zshrc)
export PATH="$PATH:/path/to/swit/bin"

# 或创建符号链接
sudo ln -s /path/to/swit/bin/switctl /usr/local/bin/switctl
```

### 3. 验证安装

```bash
switctl --help
# 应该显示 CLI 帮助和所有可用命令
```

## 使用 switctl 创建第一个服务

### 步骤 1：创建新服务

```bash
# 创建用户管理服务
switctl new service user-service
```

CLI 将提示您选择选项：

```text
✓ 服务模板: http-grpc
✓ 数据库类型: postgresql
✓ 认证方式: jwt
✓ 包含 Docker 文件: yes
✓ 包含 CI/CD 文件: yes
```

### 步骤 2：探索生成的结构

```bash
cd user-service
tree
```

生成的结构：
```text
user-service/
├── cmd/user-service/
│   └── main.go                 # 服务入口点
├── internal/
│   ├── handler/
│   │   ├── http/              # HTTP 处理器
│   │   └── grpc/              # gRPC 处理器
│   ├── service/               # 业务逻辑
│   └── config/                # 配置
├── api/
│   └── proto/                 # Protocol buffers
├── configs/
│   ├── development.yaml       # 开发环境配置
│   └── production.yaml        # 生产环境配置
├── Dockerfile                 # 容器配置
├── Makefile                   # 构建自动化
├── docker-compose.yml         # 本地开发
├── .switctl.yaml             # CLI 配置
├── go.mod                     # Go 模块
└── README.md                 # 服务文档
```

### 步骤 3：构建和运行

```bash
# 初始化 Go 模块
go mod tidy

# 构建服务
make build

# 开发模式运行
make run

# 或直接运行
go run cmd/user-service/main.go
```

您的服务现在运行在：
- HTTP: http://localhost:8080
- gRPC: localhost:9080

### 步骤 4：测试服务

```bash
# 健康检查
curl http://localhost:8080/health

# API 状态
curl http://localhost:8080/api/v1/status

# 用户操作（生成的 CRUD 端点）
curl http://localhost:8080/api/v1/users
```

## 为服务添加组件

### 生成 API 端点

```bash
# 为产品生成 CRUD API
switctl generate api product --methods=crud --validation=true

# 生成自定义 API
switctl generate api order \
  --methods=get,post,put \
  --middleware=auth,logging
```

### 生成中间件

```bash
# 生成认证中间件
switctl generate middleware auth --type=jwt

# 生成限流中间件
switctl generate middleware rate-limit \
  --apply-to=http \
  --config=true
```

### 生成数据库模型

```bash
# 使用 GORM 生成 User 模型
switctl generate model User \
  --database=gorm \
  --validation=true \
  --migration=true
```

这将生成：
- `internal/model/user.go` - 模型定义
- `internal/repository/user_repository.go` - 数据库操作
- `migrations/001_create_users.sql` - 数据库迁移

## 使用 switctl 进行质量保证

### 运行所有检查

```bash
switctl check --all
```

这会运行：
- ✅ 代码格式化 (gofmt)
- ✅ 代码质量 (golint)
- ✅ 安全扫描
- ✅ 依赖漏洞检查
- ✅ 测试覆盖率
- ✅ 性能基准测试

### 特定检查

```bash
# 仅安全检查
switctl check --security
# 发现 2 个问题:
# - 使用了弱加密原语 (MD5)
# - 查询中可能存在 SQL 注入

# 带覆盖率的测试
switctl check --tests --coverage --threshold=80
# 覆盖率: 85% (超过阈值 ✓)

# 性能基准测试
switctl check --performance
# 所有基准测试均在可接受范围内 ✓
```

## 开发工作流

### 1. 开发监视模式

```bash
# 文件变更时自动重新构建
switctl dev watch
```

这会监控您的代码并自动：
- 重新构建服务
- 运行测试
- 重启开发服务器

### 2. 生成文档

```bash
# 生成服务文档
switctl dev docs --format=html --output=./docs
```

### 3. 依赖管理

```bash
# 检查过时的依赖
switctl deps check

# 安全地更新依赖
switctl deps update --security
```

## 使用模板

### 列出可用模板

```bash
switctl template list
```

可用模板：
- `basic` - 简单 HTTP 服务
- `http-grpc` - HTTP + gRPC 服务
- `full-featured` - 启用所有功能
- `grpc-only` - 纯 gRPC 服务

### 使用特定模板

```bash
switctl new service payment-service \
  --template=full-featured \
  --database=postgresql \
  --auth=jwt \
  --cache=redis \
  --monitoring=sentry \
  --queue=rabbitmq
```

### 自定义模板

为团队标准创建自定义模板：

```bash
# 创建模板结构
mkdir -p ~/.switctl/templates/company-standard

# 生成模板
switctl template create company-standard \
  --based-on=http-grpc \
  --add-feature=monitoring \
  --add-feature=tracing
```

## 配置管理

### 项目配置

switctl 为项目特定设置创建 `.switctl.yaml`：

```yaml
project:
  name: "user-service"
  type: "microservice"
  
templates:
  default: "http-grpc"
  
database:
  type: "postgresql"
  migrations: true
  
testing:
  coverage_threshold: 80
  race_detection: true
  
security:
  enabled: true
  scan_deps: true
```

### 全局配置

设置全局默认值：

```bash
# 设置首选模板
switctl config set template.default full-featured

# 设置数据库偏好
switctl config set database.type postgresql

# 设置覆盖率阈值
switctl config set testing.coverage_threshold 85
```

## 高级功能

### 插件系统

使用自定义插件扩展 switctl：

```bash
# 列出可用插件
switctl plugin list

# 安装 OpenAPI 生成插件
switctl plugin install openapi-gen

# 使用插件
switctl openapi generate --input=./api/proto --output=./docs/openapi.yaml
```

### CI/CD 集成

生成的服务包含 CI/CD 配置：

```bash
# GitHub Actions（自动生成）
.github/workflows/
├── ci.yml              # 构建和测试
├── security.yml        # 安全扫描
└── deploy.yml          # 部署

# GitLab CI（使用 --cicd=gitlab）
.gitlab-ci.yml
```

### 多服务项目

在单一仓库中管理多个服务：

```bash
# 初始化多服务项目
switctl init company-platform --type=monorepo

# 添加服务
switctl new service user-service --directory=services/
switctl new service order-service --directory=services/
switctl new service notification-service --directory=services/

# 生成共享组件
switctl generate shared-lib common --type=utils
```

## 最佳实践

### 1. 从简单开始，逐步扩展

```bash
# 从基础模板开始
switctl new service my-service --template=basic

# 根据需要添加功能
switctl generate api user --methods=crud
switctl generate middleware auth --type=jwt
```

### 2. 及早使用质量检查

```bash
# 在开发过程中运行检查
switctl check --quality --security

# 设置预提交钩子
echo "switctl check --all" > .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

### 3. 维护配置

```bash
# 将 .switctl.yaml 加入版本控制
git add .switctl.yaml

# 记录团队标准
switctl config set team.standards "company-standard"
```

### 4. 利用模板

```bash
# 创建团队特定模板
switctl template create team-api \
  --based-on=http-grpc \
  --add-middleware=auth,logging,cors \
  --add-monitoring=sentry

# 共享模板
git commit -m "Add team API template"
```

## 故障排除

### 常见问题

**找不到命令：**
```bash
# 检查 PATH
echo $PATH | grep switctl

# 临时使用完整路径
/path/to/swit/bin/switctl --help
```

**模板错误：**
```bash
# 列出可用模板
switctl template list

# 调试模板生成
switctl --debug new service test --template=basic
```

**生成失败：**
```bash
# 启用详细输出
switctl --verbose generate api user

# 检查项目配置
switctl config get
```

### 调试模式

使用调试模式排除问题：

```bash
# 启用调试输出
switctl --debug new service debug-test

# 这会显示：
# - 模板解析
# - 文件生成步骤
# - 配置验证
# - 错误详情
```

## 下一步

现在您已经使用 switctl 创建了第一个服务：

1. **[命令参考](/zh/cli/commands)** - 学习所有可用命令
2. **[模板系统](/zh/cli/templates)** - 掌握模板系统
3. **[框架指南](/zh/guide/getting-started)** - 了解底层框架
4. **[示例](/zh/examples/)** - 探索完整示例

## 快速参考

```bash
# 基本命令
switctl new service <name>              # 创建服务
switctl generate api <name>             # 生成 API
switctl check --all                     # 质量检查
switctl dev watch                       # 开发模式

# 配置
switctl config set <key> <value>        # 设置配置
switctl config get                      # 查看配置

# 模板
switctl template list                   # 列出模板
switctl template create <name>          # 创建模板

# 帮助
switctl --help                          # 通用帮助
switctl <command> --help               # 命令帮助
```

欢迎使用 switctl 进行高效的微服务开发！🚀
