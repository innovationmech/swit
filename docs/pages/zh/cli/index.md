# switctl - Swit 框架 CLI 工具

`switctl`（Swit Control）命令行工具是 Swit 微服务框架的综合开发工具包。它提供脚手架、代码生成、质量检查和开发工具来加速微服务开发。

## 概述

switctl 提供完整的开发工作流：

- **🚀 服务脚手架**：从模板生成完整的微服务
- **🔧 代码生成**：自动创建 API、中间件和模型
- **🛡️ 质量保证**：内置安全扫描、测试和代码质量检查
- **📦 模板系统**：广泛的生产就绪模板库
- **🔌 插件系统**：支持自定义插件的可扩展架构
- **💻 交互式 UI**：用户友好的终端界面，提供引导式工作流

## 安装

### 从源码构建（推荐）

从 Swit 框架仓库构建：

```bash
# 克隆仓库
git clone https://github.com/innovationmech/swit.git
cd swit

# 构建 switctl
make build

# 二进制文件将在 ./bin/switctl
./bin/switctl --help
```

### 添加到 PATH

为了方便使用，将 switctl 添加到您的 PATH：

```bash
# 复制到 PATH 中的目录
sudo cp ./bin/switctl /usr/local/bin/

# 或创建符号链接
sudo ln -s $(pwd)/bin/switctl /usr/local/bin/switctl

# 验证安装
switctl --help
```

## 快速开始

### 1. 初始化新项目

创建新的微服务项目：

```bash
# 交互式项目初始化
switctl init my-service

# 使用默认值快速开始
switctl init my-service --quick
```

### 2. 生成服务

从模板创建完整的微服务：

```bash
# 交互式服务生成
switctl new service user-service

# 指定具体选项
switctl new service user-service \
  --template=http-grpc \
  --database=postgresql \
  --auth=jwt
```

### 3. 生成组件

为现有服务添加组件：

```bash
# 生成 API 端点
switctl generate api user

# 生成中间件
switctl generate middleware auth

# 生成数据库模型
switctl generate model User
```

### 4. 质量检查

运行全面的质量检查：

```bash
# 运行所有检查
switctl check

# 特定检查
switctl check --security
switctl check --tests --coverage
switctl check --performance
```

## 核心命令

### 项目管理

| 命令 | 描述 | 用法 |
|---------|-------------|--------|
| `init` | 初始化新项目 | `switctl init <name> [options]` |
| `new` | 生成新服务/组件 | `switctl new <type> <name> [options]` |
| `config` | 管理项目配置 | `switctl config [get\|set] [key] [value]` |

### 代码生成

| 命令 | 描述 | 用法 |
|---------|-------------|--------|
| `generate api` | 生成 API 端点 | `switctl generate api <name> [options]` |
| `generate middleware` | 创建中间件组件 | `switctl generate middleware <name>` |
| `generate model` | 生成数据模型 | `switctl generate model <name>` |

### 质量保证

| 命令 | 描述 | 用法 |
|---------|-------------|--------|
| `check` | 运行质量检查 | `switctl check [--type] [options]` |
| `test` | 运行测试并生成覆盖率 | `switctl test [options]` |
| `lint` | 代码风格和质量检查 | `switctl lint [options]` |

### 开发工具

| 命令 | 描述 | 用法 |
|---------|-------------|--------|
| `dev` | 开发实用工具 | `switctl dev <command>` |
| `deps` | 依赖管理 | `switctl deps [update\|check]` |
| `plugin` | 插件管理 | `switctl plugin <command>` |

## 主要功能

### 🏗️ 服务脚手架

生成完整的、生产就绪的微服务：

```bash
switctl new service order-service \
  --template=full-featured \
  --database=postgresql \
  --auth=jwt \
  --monitoring=sentry \
  --cache=redis
```

**生成的结构：**
- 完整的项目结构
- Docker 配置
- 包含所有命令的 Makefile
- CI/CD 流水线文件
- 健康检查和监控
- 安全配置
- 全面的测试

### 🔧 智能代码生成

使用框架最佳实践创建组件：

```bash
# 生成带验证的 RESTful API
switctl generate api product \
  --methods=crud \
  --validation=true \
  --swagger=true

# 生成 gRPC 服务
switctl generate grpc inventory \
  --streaming=true \
  --gateway=true

# 生成具有常见模式的中间件
switctl generate middleware auth \
  --type=jwt \
  --rbac=true
```

### 🛡️ 质量保证套件

全面的质量检查：

```bash
# 安全扫描
switctl check --security
# ✓ 漏洞扫描
# ✓ 依赖分析
# ✓ 代码安全模式
# ✓ 配置验证

# 性能测试
switctl check --performance
# ✓ 内存泄漏检测
# ✓ 基准测试
# ✓ 负载测试设置
# ✓ 性能回归

# 代码质量
switctl check --quality
# ✓ 代码风格和格式化
# ✓ 复杂度分析
# ✓ 测试覆盖率
# ✓ 文档覆盖率
```

### 📦 模板系统

针对不同用例的广泛模板库：

#### 服务模板
- `http-only` - 仅 HTTP 微服务
- `grpc-only` - 仅 gRPC 服务
- `http-grpc` - 双协议服务
- `full-featured` - 具有所有功能的完整服务
- `minimal` - 用于学习的最小服务

#### 认证模板
- `jwt` - 基于 JWT 的认证
- `oauth2` - OAuth2 集成
- `api-key` - API 密钥认证
- `rbac` - 基于角色的访问控制

#### 数据库模板
- `postgresql` - 使用 GORM 的 PostgreSQL
- `mysql` - MySQL 集成
- `mongodb` - 使用官方驱动的 MongoDB
- `sqlite` - 用于开发的 SQLite
- `redis` - Redis 缓存

#### 中间件模板
- `cors` - 跨域资源共享
- `rate-limit` - 限流
- `logging` - 请求日志
- `recovery` - 恐慌恢复
- `request-id` - 请求 ID 跟踪

### 🔌 插件系统

用于自定义功能的可扩展架构：

```bash
# 列出可用插件
switctl plugin list

# 安装插件
switctl plugin install <plugin-name>

# 创建自定义插件
switctl plugin create my-plugin

# 管理插件
switctl plugin enable <plugin-name>
switctl plugin disable <plugin-name>
```

### 💻 交互式体验

用户友好的终端界面：

- **引导式工作流**：分步项目设置
- **智能默认值**：基于项目上下文的合理默认值
- **验证**：输入的实时验证
- **进度指示器**：操作期间的可视化反馈
- **错误恢复**：有用的错误消息和恢复建议

## 配置

switctl 支持全局和项目特定配置：

### 全局配置

```bash
# 设置全局默认值
switctl config set template.default full-featured
switctl config set database.default postgresql
switctl config set auth.default jwt

# 查看配置
switctl config list
```

### 项目配置

`.switctl.yaml` 中的项目特定设置：

```yaml
# .switctl.yaml
project:
  name: "my-service"
  type: "microservice"
  
templates:
  default: "http-grpc"
  
database:
  type: "postgresql"
  migrations: true
  
security:
  enabled: true
  scan_deps: true
  
testing:
  coverage_threshold: 80
  race_detection: true
```

## 与框架集成

switctl 与 Swit 框架深度集成：

- **框架感知**：生成的代码遵循框架模式
- **最佳实践**：强制执行框架最佳实践
- **配置**：尊重框架配置结构
- **依赖管理**：自动管理框架依赖
- **更新**：保持生成的代码与框架更改同步

## 开发工作流

使用 switctl 的典型开发工作流：

```bash
# 1. 初始化项目
switctl init payment-service --template=http-grpc

# 2. 生成核心组件
cd payment-service
switctl generate api payment --methods=crud
switctl generate middleware auth --type=jwt
switctl generate model Payment

# 3. 添加数据库集成
switctl generate database --type=postgresql --migrations=true

# 4. 质量检查
switctl check --all

# 5. 运行测试
switctl test --coverage

# 6. 开发实用工具
switctl dev watch  # 变更时自动重建
switctl dev docs   # 生成文档
```

## 示例和教程

探索实际示例：

- [入门教程](/zh/cli/getting-started) - 完整演练
- [服务模板指南](/zh/cli/templates) - 模板系统深入解析
- [命令参考](/zh/cli/commands) - 详细命令文档
- [插件开发](/zh/cli/plugins) - 创建自定义插件

## 支持和故障排除

### 常见问题

**找不到命令：**
```bash
# 确保 switctl 在 PATH 中
echo $PATH
which switctl

# 或使用完整路径
./bin/switctl --help
```

**权限被拒绝：**
```bash
# 确保二进制文件可执行
chmod +x ./bin/switctl
```

**找不到模板：**
```bash
# 列出可用模板
switctl new service --list-templates

# 更新模板
switctl template update
```

### 获取帮助

```bash
# 一般帮助
switctl --help

# 特定命令帮助
switctl new --help
switctl generate --help

# 版本信息
switctl version
```

### 调试模式

启用调试输出进行故障排除：

```bash
# 使用调试输出运行
switctl --debug new service my-service

# 详细输出
switctl --verbose check --all
```

## 下一步

- **[入门指南](/zh/cli/getting-started)** - 详细教程
- **[命令参考](/zh/cli/commands)** - 完整命令文档
- **[模板指南](/zh/cli/templates)** - 模板系统概述
- **[插件开发](/zh/cli/plugins)** - 创建自定义插件

准备好提升您的微服务开发了吗？从[入门指南](/zh/cli/getting-started)开始吧！
