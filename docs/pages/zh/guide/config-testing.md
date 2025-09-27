### 配置测试框架使用指南

本指南介绍如何使用仓库内置的配置测试框架验证分层配置、变量插值、`*_FILE` 注入与环境变量覆盖等行为。

#### 目录结构

- `pkg/config/testutil/env.go`：提供 `EnvSandbox`，用于：
  - 创建临时工作目录并 `chdir`
  - 写入 YAML/文件型夹具
  - 设置/清理环境变量（自动快照与恢复）

#### 基本用法

```go
sandbox := testutil.NewEnvSandbox(t)
defer sandbox.Cleanup()
sandbox.Chdir()

// 写入配置文件
sandbox.WriteYAML("swit.yaml", map[string]any{"server": map[string]any{"port": 8080}})

// 设置环境变量（自动恢复）
sandbox.SetEnv("SWIT_SERVER_PORT", "10080")

opts := config.DefaultOptions()
opts.WorkDir = sandbox.Dir
opts.EnvironmentName = "dev"
m := config.NewManager(opts)
_ = m.Load()
```

#### 覆盖顺序（低 → 高）

1. 代码默认值（`SetDefault`）
2. 基础文件 `swit.yaml`
3. 环境文件 `swit.<env>.yaml`
4. 覆盖文件 `swit.override.yaml`
5. 环境变量（`SWIT_` 前缀，`.` → `_`）

#### 变量插值

- 启用 `Options.EnableInterpolation` 后，支持：
  - `${VAR}`、`${VAR:-default}`、以及 `$VAR`
  - 插值发生在文件合并后且环境变量覆盖前

#### 机密文件注入（*_FILE）

- 支持将 `VAR_FILE` 内容注入 `VAR`（去除末尾换行），两者同时设置时报错
- 后缀可通过 `Options.SecretFileSuffix` 配置（默认 `_FILE`）

#### 断言场景建议

- 分层合并优先级与嵌套键合并（浅覆盖）
- 插值默认值生效与变量未设置行为
- `*_FILE` 注入与冲突错误
- 自定义覆盖文件名与自动扩展名探测
- `extends` 链接顺序与循环检测
