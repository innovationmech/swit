# Saga 示例应用脚本

本目录包含用于管理和运行 Saga 示例应用的便捷脚本。

## 脚本列表

### 1. run.sh - 示例启动脚本

用于运行各种 Saga 示例，支持不同的执行模式和配置选项。

**基本用法：**
```bash
./run.sh <示例名称>
```

**可用示例：**
- `order` - 订单处理 Saga
- `payment` - 支付处理 Saga
- `inventory` - 库存管理 Saga
- `user` - 用户注册 Saga
- `e2e` - 端到端测试
- `all` - 所有示例

**选项：**
- `-h, --help` - 显示帮助信息
- `-v, --verbose` - 详细输出模式
- `-c, --coverage` - 生成覆盖率报告
- `-r, --race` - 启用竞态检测
- `--timeout DURATION` - 设置测试超时时间
- `--specific TEST` - 运行特定的测试函数
- `-n, --dry-run` - 试运行模式

**示例：**
```bash
# 运行订单处理示例
./run.sh order

# 运行所有测试并生成覆盖率
./run.sh --coverage all

# 详细模式运行支付处理测试（含竞态检测）
./run.sh --verbose --race payment

# 运行特定测试函数
./run.sh --specific TestOrderSagaSuccess order
```

### 2. stop.sh - 停止和清理脚本

用于停止运行中的示例进程，清理测试数据和临时文件。

**基本用法：**
```bash
./stop.sh [选项]
```

**选项：**
- `-h, --help` - 显示帮助信息
- `-a, --all` - 清理所有内容（进程、文件、日志）
- `-c, --coverage` - 仅清理覆盖率文件
- `-l, --logs` - 仅清理日志文件
- `-p, --processes` - 仅停止运行中的进程
- `-f, --force` - 强制清理，不提示确认
- `-n, --dry-run` - 试运行模式
- `-v, --verbose` - 详细输出模式

**示例：**
```bash
# 交互式清理（会提示确认）
./stop.sh

# 清理所有内容
./stop.sh --all

# 仅清理覆盖率文件
./stop.sh --coverage

# 强制清理所有内容（不提示）
./stop.sh --all --force

# 查看将清理的内容（不实际执行）
./stop.sh --dry-run --all
```

### 3. logs.sh - 日志查看工具

用于查看和分析示例应用的运行日志、测试输出和调试信息。

**基本用法：**
```bash
./logs.sh [选项] [日志类型]
```

**日志类型：**
- `test` - 测试输出日志
- `coverage` - 覆盖率报告
- `error` - 错误日志
- `debug` - 调试日志
- `all` - 所有日志（默认）

**选项：**
- `-h, --help` - 显示帮助信息
- `-f, --follow` - 持续跟踪日志（类似 tail -f）
- `-n, --lines NUM` - 显示最后 NUM 行（默认: 50）
- `--filter PATTERN` - 过滤包含指定模式的行
- `-e, --errors-only` - 仅显示错误和警告
- `-v, --verbose` - 显示详细信息
- `--no-color` - 禁用颜色输出
- `-t, --timestamp` - 显示时间戳
- `-l, --list` - 列出所有可用的日志文件

**示例：**
```bash
# 查看所有日志（最后50行）
./logs.sh

# 查看测试输出
./logs.sh test

# 查看覆盖率报告
./logs.sh coverage

# 持续跟踪测试日志
./logs.sh --follow test

# 查看最后100行错误日志
./logs.sh --lines 100 error

# 过滤包含 "Saga" 的日志行
./logs.sh --filter "Saga"

# 仅显示错误和警告
./logs.sh --errors-only
```

### 4. status.sh - 状态监控工具

用于查看示例应用的运行状态、资源使用情况和健康检查。

**基本用法：**
```bash
./status.sh [选项]
```

**选项：**
- `-h, --help` - 显示帮助信息
- `-w, --watch` - 监控模式（持续刷新）
- `-i, --interval SECONDS` - 刷新间隔（默认: 2秒）
- `-d, --detailed` - 显示详细信息
- `-j, --json` - 以 JSON 格式输出
- `-q, --quiet` - 安静模式（仅输出状态数据）
- `--health` - 健康检查

**监控内容：**
- 运行中的进程
- 测试执行状态
- 文件系统状态
- 覆盖率统计
- 最近的错误
- 系统资源使用情况

**示例：**
```bash
# 显示当前状态
./status.sh

# 持续监控（2秒刷新）
./status.sh --watch

# 持续监控（5秒刷新）
./status.sh --watch --interval 5

# 显示详细信息
./status.sh --detailed

# JSON 格式输出
./status.sh --json

# 健康检查
./status.sh --health
```

## 使用 Makefile 目标

这些脚本也可以通过 Makefile 目标调用，提供更便捷的使用方式：

### 核心命令

```bash
# 运行指定示例
make saga-examples-run EXAMPLE=order

# 查看示例状态
make saga-examples-status

# 查看示例日志
make saga-examples-logs

# 停止并清理所有内容
make saga-examples-stop
```

### 快捷命令

```bash
# 运行订单处理示例
make saga-examples-order

# 运行支付处理示例
make saga-examples-payment

# 运行库存管理示例
make saga-examples-inventory

# 运行用户注册示例
make saga-examples-user

# 运行端到端测试
make saga-examples-e2e

# 运行所有示例
make saga-examples-all
```

### 高级命令

```bash
# 高级运行模式（带选项）
make saga-examples-run-advanced EXAMPLE=payment COVERAGE=1

# 监控模式（持续刷新状态）
make saga-examples-watch

# 生成覆盖率报告
make saga-examples-coverage EXAMPLE=order

# 查看详细日志
make saga-examples-logs-detailed

# 跟踪日志（持续输出）
make saga-examples-logs-follow

# 清理特定类型
make saga-examples-clean TYPE=coverage
```

### 实用功能

```bash
# 列出所有可用示例
make saga-examples-list

# 健康检查
make saga-examples-health

# 显示文档位置
make saga-examples-docs

# 显示帮助信息
make saga-examples-help
```

## 环境变量

可以通过环境变量配置示例的行为：

```bash
# 日志级别（debug, info, warn, error）
export SAGA_LOG_LEVEL=debug

# 存储类型（memory, redis, postgres）
export SAGA_STORAGE_TYPE=memory

# 启用分布式追踪（true/false）
export SAGA_TRACE_ENABLED=true
```

## 工作流示例

### 日常开发工作流

```bash
# 1. 运行特定示例测试
make saga-examples-order

# 2. 查看状态
make saga-examples-status

# 3. 查看日志
make saga-examples-logs

# 4. 清理
make saga-examples-stop
```

### 覆盖率分析工作流

```bash
# 1. 运行测试并生成覆盖率
./run.sh --coverage all

# 2. 查看覆盖率报告
./logs.sh coverage

# 3. 生成 HTML 报告
cd ../
go tool cover -html=coverage_all.out

# 4. 清理覆盖率文件
./scripts/stop.sh --coverage --force
```

### 持续监控工作流

```bash
# 终端 1: 运行测试
./run.sh --follow order

# 终端 2: 监控状态
./status.sh --watch

# 终端 3: 跟踪日志
./logs.sh --follow test
```

### 调试工作流

```bash
# 1. 设置调试级别日志
export SAGA_LOG_LEVEL=debug

# 2. 运行特定测试
./run.sh --verbose --specific TestOrderSagaCompensation order

# 3. 查看详细日志
./logs.sh --verbose --filter "Compensation" debug

# 4. 查看错误
./logs.sh --errors-only
```

## 故障排除

### 脚本权限问题

如果脚本无法执行，请确保它们有执行权限：

```bash
chmod +x *.sh
```

### Go 环境问题

确保安装了 Go 1.23+ 并且在 PATH 中：

```bash
go version
```

### 清理卡住的进程

如果进程无法正常停止：

```bash
# 查看进程
ps aux | grep "go.*test.*saga"

# 强制终止
./stop.sh --processes --force
```

### 覆盖率文件过大

定期清理旧的覆盖率文件：

```bash
# 查看覆盖率文件大小
./logs.sh --list

# 清理所有覆盖率文件
./stop.sh --coverage --force
```

## 最佳实践

1. **定期清理**：运行测试后及时清理临时文件和日志
2. **使用试运行**：在执行复杂操作前先使用 `--dry-run` 查看将执行的命令
3. **监控资源**：使用 `--watch` 模式监控长时间运行的测试
4. **详细日志**：调试时使用 `--verbose` 和 `SAGA_LOG_LEVEL=debug`
5. **覆盖率分析**：定期运行覆盖率测试以确保代码质量
6. **健康检查**：部署前运行 `--health` 检查环境状态

## 相关文档

- [Saga 示例主文档](../README.md)
- [端到端测试文档](../E2E_TESTING.md)
- [订单处理示例](../docs/order_saga.md)
- [支付处理示例](../docs/payment_saga.md)
- [库存管理示例](../docs/inventory_saga.md)
- [用户注册示例](../docs/user_registration_saga.md)

## 贡献

欢迎改进这些脚本！如果您有建议或发现问题，请：

1. 创建 Issue 描述问题或建议
2. 提交 Pull Request 包含您的改进
3. 确保脚本通过所有测试
4. 更新相关文档

## 许可证

MIT License - 详见 [LICENSE](../../../../LICENSE) 文件

