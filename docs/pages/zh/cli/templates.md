# 模板系统指南

switctl 模板系统提供了一个全面的生产就绪模板库，用于快速搭建微服务和组件。本指南涵盖所有可用模板、自定义选项和最佳实践。

## 概述

模板系统提供：

- **🏗️ 服务模板**：完整的微服务脚手架
- **🔐 认证模板**：安全和认证模式  
- **💾 数据库模板**：数据库集成模式
- **🛡️ 中间件模板**：常见中间件组件
- **🔧 组件模板**：单个组件生成
- **🎨 自定义模板**：创建和分享您自己的模板

## 模板类别

### 服务模板

带有完整项目结构的完整微服务模板。

#### `basic`
用于学习和简单用例的最小化 HTTP 服务。

**功能：**
- 基于 Gin 框架的 HTTP 服务器
- 基本健康检查端点  
- 优雅关闭
- Docker 配置
- 简单的 Makefile

**生成的结构：**
```text
my-service/
├── cmd/my-service/
│   └── main.go
├── internal/
│   ├── handler/
│   ├── service/
│   └── config/
├── Dockerfile
├── Makefile
├── go.mod
└── README.md
```

**用法：**
```bash
switctl new service my-service --template=basic
```

#### `http-grpc`
具有 HTTP 和 gRPC 支持的全面服务模板。

**功能：**
- HTTP 服务器（Gin）+ gRPC 服务器
- 自动 HTTP/gRPC 网关集成
- Protocol Buffer 支持
- 中间件支持（认证、日志、恢复）
- 配置管理（Viper）
- 健康检查端点
- 优雅关闭和信号处理
- Docker 和 docker-compose 配置
- CI/CD 管道设置

**生成的结构：**
```text
my-service/
├── cmd/my-service/
│   └── main.go
├── internal/
│   ├── handler/
│   │   ├── http/           # HTTP 处理器
│   │   └── grpc/           # gRPC 处理器
│   ├── service/            # 业务逻辑
│   ├── middleware/         # 中间件
│   └── config/             # 配置管理
├── api/
│   └── proto/              # Protocol Buffers
├── configs/
│   ├── development.yaml
│   └── production.yaml
├── pkg/                    # 共享包
├── scripts/                # 部署和开发脚本
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── .github/workflows/      # CI/CD
└── README.md
```

**用法：**
```bash
switctl new service user-service --template=http-grpc
```

#### `full-featured`
包含所有框架功能的完整模板。

**功能：**
- 所有 http-grpc 功能
- 数据库集成（GORM、迁移）
- 缓存支持（Redis）
- 消息队列（RabbitMQ/Kafka）
- 监控和可观测性（Sentry、Prometheus）
- 认证和授权（JWT、OAuth2）
- API 文档生成
- 测试设置和示例
- 性能基准测试
- 安全扫描配置

**附加组件：**
```text
├── internal/
│   ├── auth/               # 认证和授权
│   ├── cache/              # 缓存实现
│   ├── database/           # 数据库配置
│   ├── monitor/            # 监控和指标
│   └── queue/              # 消息队列
├── migrations/             # 数据库迁移
├── docs/                   # API 文档
├── tests/                  # 测试套件
│   ├── integration/
│   ├── unit/
│   └── benchmarks/
└── security/               # 安全配置
```

**用法：**
```bash
switctl new service payment-service \
  --template=full-featured \
  --database=postgresql \
  --auth=jwt \
  --monitoring=sentry
```

#### `grpc-only`
纯 gRPC 服务模板。

**功能：**
- 仅 gRPC 服务器
- Protocol Buffer 优先设计
- 流处理支持
- gRPC 中间件
- 服务发现集成
- 负载均衡配置

**用法：**
```bash
switctl new service inventory-service --template=grpc-only
```

#### `minimal`
学习和原型设计的最小模板。

**功能：**
- 基本 HTTP 服务器
- 单个文件结构
- 无外部依赖
- 教育导向

**用法：**
```bash
switctl new service learning-service --template=minimal
```

### 认证模板

#### JWT 认证
具有 JSON Web Token 支持的认证系统。

**功能：**
- JWT 令牌生成和验证
- 中间件集成
- 令牌刷新
- 用户管理端点

**用法：**
```bash
switctl generate auth jwt --middleware=true
```

#### OAuth2 集成
OAuth2 提供商集成。

**功能：**
- 多提供商支持（Google、GitHub、Facebook）
- 令牌交换
- 用户配置文件获取
- 会话管理

#### API 密钥认证
基于 API 密钥的认证。

**功能：**
- API 密钥生成
- 密钥验证中间件
- 速率限制集成
- 密钥管理端点

### 数据库模板

#### PostgreSQL 集成
全功能 PostgreSQL 集成。

**功能：**
- GORM 配置
- 连接池
- 迁移支持
- 健康检查
- 事务管理

**生成的文件：**
```text
internal/
├── database/
│   ├── connection.go
│   ├── migrations.go
│   └── health.go
└── model/
    └── base.go
migrations/
└── 001_initial.sql
```

#### MongoDB 集成
MongoDB 数据库集成。

**功能：**
- Mongo 驱动配置
- 连接管理
- 索引管理
- 聚合支持

#### Redis 缓存
Redis 缓存集成。

**功能：**
- 连接配置
- 缓存模式
- 分布式锁
- 发布/订阅

### 中间件模板

#### 认证中间件
请求认证中间件。

**功能：**
- JWT 验证
- API 密钥检查
- 会话验证
- 用户上下文注入

#### CORS 中间件
跨域资源共享中间件。

**功能：**
- 可配置的 CORS 策略
- 预检请求处理
- 凭证支持

#### 速率限制中间件
API 速率限制中间件。

**功能：**
- 多种限制策略
- 分布式限制（Redis）
- 自定义限制规则
- 监控集成

#### 日志中间件
请求/响应日志中间件。

**功能：**
- 结构化日志
- 请求/响应跟踪
- 性能指标
- 错误记录

### 监控模板

#### Sentry 集成
错误跟踪和性能监控。

**功能：**
- 自动错误捕获
- 性能跟踪
- 用户上下文
- 发布跟踪

#### Prometheus 指标
指标收集和导出。

**功能：**
- HTTP 指标
- 自定义指标
- 健康检查集成
- Grafana 仪表板

## 使用模板

### 列出可用模板

```bash
# 列出所有模板
switctl template list

# 按类别筛选
switctl template list --category=service
switctl template list --category=middleware

# 显示详细信息
switctl template list --detailed
```

### 查看模板详情

```bash
# 显示模板信息
switctl template info http-grpc

# 显示将生成的文件
switctl template show http-grpc --files

# 显示模板变量
switctl template show http-grpc --variables
```

### 使用特定模板

```bash
# 使用模板创建服务
switctl new service my-service --template=http-grpc

# 带选项的模板
switctl new service my-service \
  --template=full-featured \
  --database=postgresql \
  --auth=jwt \
  --cache=redis
```

## 自定义模板

### 创建自定义模板

#### 1. 模板结构

创建模板目录结构：

```text
~/.switctl/templates/my-template/
├── template.yaml           # 模板配置
├── pre-generate.sh         # 生成前脚本（可选）
├── post-generate.sh        # 生成后脚本（可选）
└── files/                  # 模板文件
    ├── main.go.tmpl
    ├── config/
    │   └── config.go.tmpl
    └── README.md.tmpl
```

#### 2. 模板配置

`template.yaml` 示例：

```yaml
name: "my-template"
description: "My custom service template"
version: "1.0.0"
author: "Your Name"

# 模板类型
type: "service"

# 模板变量
variables:
  - name: "ServiceName"
    description: "Service name"
    type: "string"
    required: true
  - name: "Port"
    description: "HTTP port"
    type: "int"
    default: 8080
  - name: "Database"
    description: "Database type"
    type: "enum"
    values: ["postgresql", "mysql", "mongodb"]
    default: "postgresql"

# 功能支持
features:
  - "http"
  - "database"
  - "docker"

# 依赖
dependencies:
  - "github.com/gin-gonic/gin"
  - "gorm.io/gorm"

# 文件权限
permissions:
  "scripts/*.sh": "755"
```

#### 3. 模板文件

使用 Go 模板语法：

**main.go.tmpl：**
```go
package main

import (
    "log"
    "net/http"
    
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.Default()
    
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "service": "{ {.ServiceName}}",
            "status": "healthy"
        })
    })
    
    log.Printf("Starting { {.ServiceName}} on port { {.Port}}")
    r.Run(":{ {.Port}}")
}
```

### 模板变量

#### 常见变量
模板中可用的常见变量：

<code v-pre>{{.ServiceName}}</code> - 服务名称
<code v-pre>{{.ModulePath}}</code> - Go 模块路径  
<code v-pre>{{.Author}}</code> - 作者名称
<code v-pre>{{.Year}}</code> - 当前年份
<code v-pre>{{.Date}}</code> - 当前日期
<code v-pre>{{.Database}}</code> - 数据库类型
<code v-pre>{{.Auth}}</code> - 认证类型
<code v-pre>{{.Port}}</code> - HTTP 端口
<code v-pre>{{.GRPCPort}}</code> - gRPC 端口

#### 条件生成

```go
{ {if .Database}}
import "gorm.io/gorm"
{ {end}}

{ {if eq .Database "postgresql"}}
import _ "gorm.io/driver/postgres"
{ {else if eq .Database "mysql"}}
import _ "gorm.io/driver/mysql"
{ {end}}
```

### 脚本钩子

#### pre-generate.sh
在生成文件之前运行：

```bash
#!/bin/bash
echo "准备生成 { {.ServiceName}}"

# 验证先决条件
if ! command -v go &> /dev/null; then
    echo "错误：需要安装 Go"
    exit 1
fi

# 创建目录
mkdir -p internal/handler
mkdir -p configs
```

#### post-generate.sh
在生成文件之后运行：

```bash
#!/bin/bash
echo "完成生成 { {.ServiceName}}"

# 初始化 Go 模块
go mod init { {.ModulePath}}
go mod tidy

# 格式化代码
go fmt ./...

# 运行初始测试
go test ./...

echo "服务 { {.ServiceName}} 已成功创建!"
```

### 分发模板

#### 1. 本地分享

```bash
# 打包模板
switctl template pack my-template

# 分享模板包
switctl template install my-template.tar.gz
```

#### 2. Git 仓库

```bash
# 从 Git 安装模板
switctl template install https://github.com/user/my-template.git

# 安装特定版本
switctl template install https://github.com/user/my-template.git@v1.0.0
```

#### 3. 团队模板仓库

```bash
# 配置团队模板仓库
switctl config set template.repository https://templates.company.com

# 列出团队模板
switctl template list --remote

# 安装团队模板
switctl template install company-standard
```

## 最佳实践

### 模板设计

1. **保持简单** - 从基本功能开始，逐步添加复杂性
2. **使用清晰的变量名** - 使模板变量自解释
3. **提供合理的默认值** - 减少用户输入负担
4. **包含文档** - 在模板中包含 README 和注释
5. **测试模板** - 确保生成的代码能够编译和运行

### 变量命名

```yaml
# 好的变量名
variables:
  - name: "ServiceName"
  - name: "DatabaseType" 
  - name: "HTTPPort"

# 避免的变量名
variables:
  - name: "x"
  - name: "temp"
  - name: "thing"
```

### 条件逻辑

```go
// 使用清晰的条件
{ {if .EnableDatabase}}
// 数据库相关代码
{ {end}}

// 避免复杂嵌套
{ {if .EnableDatabase}}
  { {if eq .DatabaseType "postgresql"}}
    { {if .EnableMigrations}}
    // 避免深层嵌套
    { {end}}
  { {end}}
{ {end}}
```

### 文件组织

```text
templates/my-template/
├── template.yaml
├── README.md                # 模板文档
├── scripts/
│   ├── pre-generate.sh
│   └── post-generate.sh
└── files/
    ├── cmd/
    ├── internal/
    ├── configs/
    └── docs/
```

## 故障排除

### 常见问题

**模板未找到：**
```bash
# 检查模板位置
switctl template list --local

# 验证模板路径
echo $SWITCTL_TEMPLATE_DIR
```

**生成失败：**
```bash
# 使用调试模式
switctl --debug new service test --template=my-template

# 检查模板语法
switctl template validate my-template
```

**变量错误：**
```bash
# 显示模板变量
switctl template show my-template --variables

# 测试变量替换
switctl template test my-template --var ServiceName=test
```

### 调试技巧

1. **使用 `--dry-run`** 预览生成的文件
2. **启用调试输出** 查看模板处理过程
3. **验证模板语法** 在使用前检查模板
4. **测试小的更改** 逐步验证模板修改

## 社区模板

### 官方模板库

访问官方模板库：
```bash
# 列出官方模板
switctl template list --official

# 更新官方模板
switctl template update --official
```

### 社区贡献

```bash
# 提交模板到社区
switctl template submit my-template

# 评分和评论模板
switctl template rate awesome-template --stars=5
```

了解更多关于模板系统的信息，请参阅[开发者指南](/zh/guide/getting-started)或运行 `switctl template --help`。
