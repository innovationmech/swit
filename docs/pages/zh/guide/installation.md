# 安装指南

本指南将帮助您安装 Swit Go 微服务框架并设置开发环境。

## 系统要求

在安装 Swit 之前，请确保您满足以下要求：

- **Go 1.24+** - 支持泛型的现代 Go 版本
- **Git** - 用于框架和示例代码管理
- **Make** - 用于构建和开发命令（可选但推荐）

### 验证 Go 安装

检查您的 Go 版本：

```bash
go version
```

您应该看到 Go 1.24 或更高版本。如果您没有安装 Go，请访问 [https://golang.org/doc/install](https://golang.org/doc/install)。

## 安装 Swit 框架

### 方法一：使用 go get（推荐）

直接在项目中安装框架：

```bash
go get github.com/innovationmech/swit
```

### 方法二：从源码克隆构建

适用于开发或定制：

```bash
# 克隆仓库
git clone https://github.com/innovationmech/swit.git
cd swit

# 构建框架
make build

# 运行测试验证安装
make test
```

## 验证安装

创建一个简单的测试来验证安装：

```bash
mkdir test-swit
cd test-swit
go mod init test-swit
```

创建 `main.go` 文件：

```go
package main

import (
    "fmt"
    "github.com/innovationmech/swit/pkg/server"
)

func main() {
    config := &server.ServerConfig{
        Name:    "test-service",
        Version: "1.0.0",
    }
    fmt.Printf("Swit 框架加载成功！\n")
    fmt.Printf("服务: %s v%s\n", config.Name, config.Version)
}
```

运行测试：

```bash
go run main.go
```

您应该看到：
```
Swit 框架加载成功！
服务: test-service v1.0.0
```

## 开发环境设置

对于框架开发，设置完整的开发环境：

```bash
# 克隆仓库
git clone https://github.com/innovationmech/swit.git
cd swit

# 设置开发环境
make setup-dev

# 验证开发环境设置
make ci
```

这将：
- 安装所有开发依赖
- 生成协议缓冲区代码
- 运行所有测试
- 验证代码质量

## IDE 设置

### VS Code

使用 VS Code 获得最佳开发体验：

1. 安装 Go 扩展
2. 安装 Protocol Buffer 扩展（用于 API 开发）
3. 将这些设置添加到工作区：

```json
{
    "go.toolsEnvVars": {
        "GOPROXY": "https://proxy.golang.org,direct"
    },
    "go.useLanguageServer": true,
    "go.lintTool": "golint",
    "go.formatTool": "gofmt"
}
```

### GoLand/IntelliJ

1. 确保安装并启用 Go 插件
2. 将 GOROOT 设置为您的 Go 安装路径
3. 如果在模块模式外使用 Go 模块，请设置 GOPATH

## 故障排除

### 常见问题

**Go 版本过旧：**
```
go get: module github.com/innovationmech/swit requires Go 1.24
```
解决方案：将 Go 升级到 1.24 或更高版本。

**模块未找到：**
```
go: github.com/innovationmech/swit@latest: module github.com/innovationmech/swit: Get "https://proxy.golang.org/github.com/innovationmech/swit/@v/list": dial tcp: lookup proxy.golang.org: no such host
```
解决方案：检查网络连接或配置 GOPROXY。

**构建失败：**
```
make: *** No rule to make target 'build'. Stop.
```
解决方案：确保您在包含 Makefile 的正确目录中。

### 获取帮助

- 查看[故障排除指南](./troubleshooting.md)
- 查看[示例](/zh/examples/)
- 访问我们的[社区论坛](/zh/community/)

## 下一步

现在您已经安装了 Swit：

1. [开始使用](./getting-started.md)您的第一个服务
2. 探索不同用例的[示例](/zh/examples/)
3. 阅读详细参考的 [API 文档](/zh/api/)

## 系统要求

### 最低要求
- **CPU：** 1 核
- **RAM：** 512MB
- **磁盘：** 100MB 可用空间
- **网络：** 互联网连接用于依赖下载

### 推荐要求
- **CPU：** 2+ 核
- **RAM：** 2GB+
- **磁盘：** 1GB+ 可用空间
- **网络：** 高速互联网连接以加快构建速度