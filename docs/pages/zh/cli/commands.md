# CLI 命令参考

所有 `switctl` 命令、选项和使用模式的完整参考。

## 全局选项

这些选项适用于所有命令：

```bash
switctl [全局选项] <命令> [命令选项] [参数]
```

| 选项 | 描述 | 示例 |
|--------|-------------|---------|
| `--debug` | 启用调试输出 | `switctl --debug new service my-service` |
| `--verbose, -v` | 详细输出 | `switctl -v check --all` |
| `--config, -c` | 指定配置文件 | `switctl -c custom.yaml new service` |
| `--help, -h` | 显示帮助 | `switctl --help` |
| `--version` | 显示版本 | `switctl --version` |

## init - 初始化项目

初始化新的 Swit 框架项目。

### 用法

```bash
switctl init <项目名> [选项]
```

### 选项

| 选项 | 描述 | 默认值 | 示例 |
|--------|-------------|---------|---------|
| `--template, -t` | 项目模板 | `basic` | `--template=full-featured` |
| `--module-path, -m` | Go 模块路径 | 自动生成 | `--module-path=github.com/user/project` |
| `--directory, -d` | 目标目录 | 当前目录 | `--directory=./projects` |
| `--quick, -q` | 跳过交互提示 | false | `--quick` |
| `--force, -f` | 覆盖现有文件 | false | `--force` |
| `--dry-run` | 显示将创建的内容 | false | `--dry-run` |

### 示例

```bash
# 交互式初始化
switctl init my-service

# 使用默认值快速初始化
switctl init my-service --quick

# 全功能项目
switctl init my-service \
  --template=full-featured \
  --module-path=github.com/mycompany/my-service

# 自定义目录
switctl init my-service --directory=./projects --force
```

### 可用模板

- `basic` - 基础 HTTP 服务
- `grpc` - 仅 gRPC 服务  
- `http-grpc` - HTTP + gRPC 服务
- `full-featured` - 所有框架功能
- `minimal` - 学习/示例服务

## new - 生成新组件

生成新的服务、API、中间件和其他组件。

### new service

生成完整的微服务。

```bash
switctl new service <服务名> [选项]
```

#### 选项

| 选项 | 描述 | 可选值 | 默认值 |
|--------|-------------|---------|----------|
| `--template, -t` | 服务模板 | 见模板 | `http-grpc` |
| `--database` | 数据库类型 | `postgresql`, `mysql`, `mongodb`, `sqlite`, `none` | `none` |
| `--auth` | 认证类型 | `jwt`, `oauth2`, `api-key`, `none` | `none` |
| `--cache` | 缓存方案 | `redis`, `memcached`, `none` | `none` |
| `--monitoring` | 监控方案 | `sentry`, `prometheus`, `none` | `none` |
| `--queue` | 消息队列 | `rabbitmq`, `kafka`, `redis`, `none` | `none` |
| `--docker` | 包含 Docker 文件 | true/false | `true` |
| `--makefile` | 包含 Makefile | true/false | `true` |
| `--cicd` | 包含 CI/CD 文件 | `github`, `gitlab`, `none` | `github` |
| `--quick, -q` | 跳过交互提示 | - | - |
| `--force, -f` | 覆盖现有文件 | - | - |

#### 示例

```bash
# 交互式服务生成
switctl new service user-service

# 包含所有功能的完整服务
switctl new service order-service \
  --template=full-featured \
  --database=postgresql \
  --auth=jwt \
  --cache=redis \
  --monitoring=sentry \
  --queue=rabbitmq

# 最小化 gRPC 服务
switctl new service inventory-service \
  --template=grpc \
  --database=mongodb \
  --quick
```

### new api

生成 API 端点。

```bash
switctl new api <api名> [选项]
```

#### 选项

| 选项 | 描述 | 可选值 | 默认值 |
|--------|-------------|---------|----------|
| `--methods` | 要生成的 HTTP 方法 | `get`, `post`, `put`, `delete`, `crud` | `crud` |
| `--validation` | 包含验证 | true/false | `true` |
| `--swagger` | 生成 Swagger 文档 | true/false | `true` |
| `--middleware` | 应用中间件 | 逗号分隔列表 | `auth,logging` |
| `--model` | 关联的模型名 | 模型名 | 自动生成 |

#### 示例

```bash
# 带验证的 CRUD API
switctl new api user --methods=crud --validation=true

# 自定义端点
switctl new api product \
  --methods=get,post,delete \
  --middleware=auth,rate-limit \
  --model=Product
```

### new middleware

生成中间件组件。

```bash
switctl new middleware <中间件名> [选项]
```

#### 选项

| 选项 | 描述 | 可选值 | 默认值 |
|--------|-------------|---------|----------|
| `--type` | 中间件类型 | `auth`, `cors`, `rate-limit`, `logging`, `custom` | `custom` |
| `--apply-to` | 应用到传输层 | `http`, `grpc`, `both` | `both` |
| `--config` | 包含配置 | true/false | `true` |

#### 示例

```bash
# 认证中间件
switctl new middleware auth --type=auth --apply-to=both

# 限流中间件
switctl new middleware rate-limit \
  --type=rate-limit \
  --apply-to=http \
  --config=true
```

## generate - 代码生成

生成特定的代码组件。

### generate model

生成带数据库集成的数据模型。

```bash
switctl generate model <模型名> [选项]
```

#### 选项

| 选项 | 描述 | 可选值 | 默认值 |
|--------|-------------|---------|----------|
| `--database` | 目标数据库 | `gorm`, `mongo`, `custom` | `gorm` |
| `--validation` | 包含验证标签 | true/false | `true` |
| `--json` | 包含 JSON 标签 | true/false | `true` |
| `--migration` | 生成迁移文件 | true/false | `true` |
| `--test` | 生成测试文件 | true/false | `true` |

#### 示例

```bash
# 使用 GORM 的 User 模型
switctl generate model User \
  --database=gorm \
  --validation=true \
  --migration=true

# MongoDB 模型
switctl generate model Product --database=mongo --test=true
```

### generate proto

生成 Protocol Buffer 定义和代码。

```bash
switctl generate proto <服务名> [选项]
```

#### 选项

| 选项 | 描述 | 可选值 | 默认值 |
|--------|-------------|---------|----------|
| `--methods` | 服务方法 | 逗号分隔列表 | `Create,Get,Update,Delete` |
| `--streaming` | 包含流方法 | true/false | `false` |
| `--gateway` | 生成 HTTP 网关 | true/false | `true` |
| `--validate` | 包含验证 | true/false | `true` |

## check - 质量保证

对代码库运行全面的质量检查。

### 用法

```bash
switctl check [选项]
```

### 选项

| 选项 | 描述 | 默认值 |
|--------|-------------|---------|
| `--all` | 运行所有检查 | false |
| `--security` | 安全扫描 | false |
| `--quality` | 代码质量检查 | false |
| `--performance` | 性能测试 | false |
| `--tests` | 运行测试 | false |
| `--coverage` | 检查测试覆盖率 | false |
| `--deps` | 依赖分析 | false |
| `--threshold <n>` | 覆盖率阈值 | 80 |
| `--format` | 输出格式 | `text`, `json`, `junit` | `text` |
| `--output, -o` | 输出文件 | 标准输出 |

### 检查类型

#### 安全检查

```bash
switctl check --security
```

**检查内容：**
- 依赖中的已知漏洞
- 不安全的代码模式
- 配置安全问题
- 密钥检测
- 许可证合规性

#### 质量检查

```bash
switctl check --quality
```

**检查内容：**
- 代码格式化 (gofmt)
- 代码复杂度
- 死代码检测
- 导入组织
- 文档覆盖率

#### 性能检查

```bash
switctl check --performance
```

**检查内容：**
- 内存泄漏
- Goroutine 泄漏
- 基准测试回归
- 资源使用模式

#### 测试检查

```bash
switctl check --tests --coverage --threshold=90
```

**检查内容：**
- 测试执行
- 覆盖率百分比
- 竞态条件
- 测试组织

### 示例

```bash
# 运行所有检查
switctl check --all

# 仅安全和质量检查
switctl check --security --quality

# 带自定义阈值的覆盖率检查
switctl check --tests --coverage --threshold=85

# 输出到文件
switctl check --all --format=json --output=quality-report.json
```

## config - 配置管理

管理项目和全局配置。

### 用法

```bash
switctl config <子命令> [选项]
```

### 子命令

#### get

获取配置值。

```bash
switctl config get [键]
```

示例：
```bash
# 获取所有配置
switctl config get

# 获取特定值
switctl config get template.default
switctl config get database.type
```

#### set

设置配置值。

```bash
switctl config set <键> <值>
```

示例：
```bash
# 设置默认模板
switctl config set template.default full-featured

# 设置数据库偏好
switctl config set database.type postgresql

# 设置覆盖率阈值
switctl config set testing.coverage_threshold 85
```

#### list

列出所有配置键和值。

```bash
switctl config list
```

#### reset

将配置重置为默认值。

```bash
switctl config reset [键]
```

示例：
```bash
# 重置所有配置
switctl config reset

# 重置特定键
switctl config reset template.default
```

## test - 测试

使用高级选项运行测试。

### 用法

```bash
switctl test [选项]
```

### 选项

| 选项 | 描述 | 默认值 |
|--------|-------------|---------|
| `--coverage` | 生成覆盖率报告 | false |
| `--race` | 启用竞态检测 | false |
| `--bench` | 运行基准测试 | false |
| `--short` | 仅运行短测试 | false |
| `--verbose, -v` | 详细输出 | false |
| `--package, -p` | 特定包 | 所有包 |
| `--timeout` | 测试超时 | 10m |
| `--parallel` | 并行测试数量 | 自动 |

### 示例

```bash
# 基础测试运行
switctl test

# 带覆盖率和竞态检测
switctl test --coverage --race

# 特定包的基准测试
switctl test --package=./internal/service --bench --verbose

# 仅短测试
switctl test --short --timeout=5m
```

## deps - 依赖管理

管理项目依赖。

### 用法

```bash
switctl deps <子命令> [选项]
```

### 子命令

#### update

更新依赖到最新版本。

```bash
switctl deps update [选项]
```

选项：
- `--security` - 仅更新安全修复
- `--minor` - 允许小版本更新
- `--major` - 允许大版本更新
- `--dry-run` - 显示将要更新的内容

#### check

检查过时或有漏洞的依赖。

```bash
switctl deps check [选项]
```

选项：
- `--security` - 检查安全漏洞
- `--outdated` - 检查过时包
- `--format` - 输出格式 (`text`, `json`)

#### clean

清理未使用的依赖。

```bash
switctl deps clean
```

### 示例

```bash
# 检查所有依赖
switctl deps check --security --outdated

# 仅安全修复更新
switctl deps update --security

# 清理未使用的依赖
switctl deps clean
```

## dev - 开发工具

开发工具和实用程序。

### 用法

```bash
switctl dev <子命令> [选项]
```

### 子命令

#### watch

监视文件变更并自动重建。

```bash
switctl dev watch [选项]
```

选项：
- `--ignore` - 忽略模式
- `--command` - 自定义构建命令
- `--delay` - 重建延迟

#### docs

生成文档。

```bash
switctl dev docs [选项]
```

选项：
- `--format` - 输出格式 (`html`, `markdown`)
- `--output` - 输出目录

#### serve

运行开发服务器。

```bash
switctl dev serve [选项]
```

选项：
- `--port` - 服务器端口
- `--host` - 服务器主机
- `--reload` - 变更时自动重载

### 示例

```bash
# 监视并自动重建
switctl dev watch --ignore="*.log,tmp/"

# 生成文档
switctl dev docs --format=html --output=./docs

# 带自动重载的开发服务器
switctl dev serve --port=8080 --reload
```

## plugin - 插件管理

管理 switctl 插件。

### 用法

```bash
switctl plugin <子命令> [参数]
```

### 子命令

#### list

列出可用插件。

```bash
switctl plugin list [--installed] [--available]
```

#### install

安装插件。

```bash
switctl plugin install <插件名>
```

#### uninstall

卸载插件。

```bash
switctl plugin uninstall <插件名>
```

#### update

更新插件。

```bash
switctl plugin update [插件名]
```

### 示例

```bash
# 列出所有插件
switctl plugin list

# 安装特定插件
switctl plugin install swagger-gen

# 更新所有插件
switctl plugin update
```

## 退出代码

switctl 使用标准退出代码：

| 代码 | 含义 |
|------|---------|
| 0 | 成功 |
| 1 | 一般错误 |
| 2 | 命令误用 |
| 126 | 调用的命令无法执行 |
| 127 | 找不到命令 |
| 128+n | 致命错误信号 "n" |

## 环境变量

switctl 识别这些环境变量：

| 变量 | 描述 | 默认值 |
|----------|-------------|---------|
| `SWITCTL_CONFIG` | 配置文件路径 | `~/.switctl.yaml` |
| `SWITCTL_TEMPLATE_DIR` | 自定义模板目录 | 内置 |
| `SWITCTL_PLUGIN_DIR` | 插件目录 | `~/.switctl/plugins` |
| `SWITCTL_DEBUG` | 启用调试模式 | false |
| `NO_COLOR` | 禁用彩色输出 | false |

## 配置文件

### 全局配置

位置：`~/.switctl.yaml`

```yaml
template:
  default: "full-featured"
  directory: "~/.switctl/templates"

database:
  default: "postgresql"
  migrations: true

testing:
  coverage_threshold: 80
  race_detection: true

security:
  scan_dependencies: true
  check_secrets: true

output:
  format: "text"
  color: true
```

### 项目配置

位置：`.switctl.yaml`（在项目根目录）

```yaml
project:
  name: "my-service"
  version: "1.0.0"
  type: "microservice"

framework:
  version: "latest"
  features:
    - "http"
    - "grpc"  
    - "database"

database:
  type: "postgresql"
  migrations: true
  seeds: true

middleware:
  - "auth"
  - "logging"
  - "recovery"

testing:
  coverage_threshold: 85
  parallel: true
```

## 技巧和最佳实践

### 性能技巧

1. **使用 `--quick` 标志** 当您知道自己想要什么时，可以更快地生成
2. **使用 `--package` 指定确切的包** 以便更快地测试
3. **使用 `.switctl.yaml`** 避免重复常用选项
4. **启用并行测试** 使用适当的 `--parallel` 值

### 工作流技巧

1. **从模板开始** - 使用既定模式而不是从头构建
2. **及早运行检查** - 在开发期间使用 `switctl check`，而不仅仅在 CI 中
3. **使用 Makefile 自动化** - 将 switctl 命令添加到您的 Makefile
4. **使用配置文件** - 将常用设置存储在 `.switctl.yaml` 中

### 故障排除

1. **使用调试模式** - 添加 `--debug` 查看 switctl 正在做什么
2. **检查配置** - 运行 `switctl config get` 验证设置
3. **验证模板** - 使用 `--dry-run` 预览生成
4. **清理缓存** - 如果有问题，删除 `~/.switctl/cache`

如需更多帮助，请参阅[故障排除指南](/zh/guide/troubleshooting)或运行 `switctl --help`。
