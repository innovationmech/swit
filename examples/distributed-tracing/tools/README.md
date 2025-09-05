# 开发工具目录

本目录包含分布式追踪系统的开发辅助工具，用于简化开发、测试和调试流程。

## 工具列表

### 🎯 `trace-generator` - 追踪数据生成工具
**用途**: 生成各种类型的追踪数据，模拟真实业务场景  
**功能**:
- 模拟订单、支付、库存等业务场景
- 支持并发请求和自定义错误率
- 生成大量测试数据用于性能测试
- 支持随机化参数和真实业务流程模拟

**使用方法**:
```bash
# 生成100条混合场景数据
./tools/trace-generator

# 并发10生成500条订单数据
./tools/trace-generator -s order -n 500 -c 10

# 按RPS生成，持续5分钟，10%错误率
./tools/trace-generator -d 300 -r 50 --error-rate=10

# 预览模式，查看将生成的数据类型
./tools/trace-generator --dry-run -s error
```

**主要参数**:
- `-s, --scenario`: 生成场景 (order, payment, inventory, mixed, error)
- `-n, --count`: 数据条数
- `-c, --concurrency`: 并发度
- `-d, --duration`: 持续时间
- `-r, --rate`: 生成速率 (RPS)
- `--error-rate`: 错误率百分比
- `--dry-run`: 预览模式

### 🔧 `config-validator` - 配置验证工具
**用途**: 验证分布式追踪系统的各种配置文件  
**功能**:
- Docker Compose、环境变量、YAML/JSON 配置验证
- 语法检查、完整性验证、连接性测试
- 安全配置检查和性能配置建议
- 自动修复常见配置问题

**使用方法**:
```bash
# 验证默认配置文件
./tools/config-validator

# 验证指定文件
./tools/config-validator -f docker-compose.yml

# 只验证连接性
./tools/config-validator -c connectivity

# 严格模式，自动修复并输出JSON
./tools/config-validator --strict --fix --output=json
```

**检查类型**:
- `syntax`: 配置文件语法检查
- `completeness`: 必需参数完整性检查  
- `connectivity`: 服务连接性检查
- `security`: 安全配置检查
- `performance`: 性能配置检查
- `all`: 执行所有检查

### 📋 `log-parser` - 日志分析工具
**用途**: 分析分布式追踪系统的日志，提供智能日志解析和分析  
**功能**:
- 多源日志聚合 (Docker、文件、journald)
- 智能错误模式识别和分类
- 性能指标提取和分析
- 追踪ID关联分析
- 实时日志跟踪和高亮显示

**使用方法**:
```bash
# 分析最近1小时的所有日志
./tools/log-parser

# 只看订单服务的错误日志
./tools/log-parser --service=order-service --errors-only

# 分析特定追踪ID的完整链路
./tools/log-parser --trace-id=abc123 --analyze

# 实时跟踪并高亮错误
./tools/log-parser --follow --highlight='ERROR|WARN'

# 分析2小时性能数据，输出JSON
./tools/log-parser -t 2h --performance --output=json
```

**主要功能**:
- 多种日志源支持 (docker, file, journald)
- 灵活的过滤和搜索 (级别、模式、时间范围)
- 深度分析 (错误统计、性能指标、追踪关联)
- 多种输出格式 (text, json, csv, html)

## 工具依赖

### 基础依赖 (必需)
```bash
# macOS
brew install curl jq bc

# Ubuntu/Debian  
sudo apt-get install curl jq bc

# 检查已安装工具
which curl jq bc
```

### 可选依赖 (增强功能)
```bash
# YAML 处理
brew install yq                    # macOS
sudo snap install yq             # Ubuntu

# 负载测试工具 (trace-generator)
go install github.com/rakyll/hey@latest
brew install wrk                  # macOS
sudo apt-get install apache2-utils # Ubuntu (ab工具)

# 配置验证增强 (config-validator)
pip install yamllint
sudo apt-get install shellcheck
```

## 集成使用示例

### 完整的开发测试流程
```bash
# 1. 验证配置
./tools/config-validator --fix

# 2. 启动服务
./scripts/start.sh

# 3. 生成测试数据
./tools/trace-generator -s mixed -n 100 -c 5

# 4. 分析生成的日志
./tools/log-parser --analyze --stats

# 5. 检查错误日志
./tools/log-parser --errors-only --output=html --output-file=error-report.html
```

### 性能测试流程
```bash
# 1. 生成高负载测试数据
./tools/trace-generator -d 300 -r 100 -c 20

# 2. 实时监控日志
./tools/log-parser --follow --performance &

# 3. 分析性能日志
./tools/log-parser -t 5m --performance --output=json > perf-analysis.json
```

### 调试特定问题
```bash
# 1. 找到问题的追踪ID
./tools/log-parser --errors-only | grep -o 'trace[_-]id[=:]?\s*[a-fA-F0-9-]\{8,\}'

# 2. 分析完整追踪链路
./tools/log-parser --trace-id=<TRACE_ID> --analyze --context=5

# 3. 生成详细报告
./tools/log-parser --trace-id=<TRACE_ID> --output=html --output-file=debug-report.html
```

## 工具特性

### 跨平台支持
- ✅ Linux
- ✅ macOS  
- ⚠️ Windows (需要 WSL)

### 智能检测
- 自动发现配置文件和日志源
- 智能参数推荐和默认值
- 错误处理和用户友好的提示

### 高度可配置
- 丰富的命令行选项
- 多种输出格式支持
- 灵活的过滤和搜索条件

### 性能优化
- 并发处理支持
- 大数据量处理优化
- 内存使用控制

## 故障排查

### 常见问题

1. **权限错误**
   ```bash
   chmod +x tools/*
   ```

2. **依赖缺失**
   ```bash
   # 检查依赖
   ./tools/config-validator --help  # 会自动检查依赖
   
   # 安装缺失的工具
   brew install curl jq bc yq  # macOS
   ```

3. **服务连接失败**
   ```bash
   # 确认服务运行状态
   ./scripts/health-check.sh
   
   # 启动服务
   ./scripts/start.sh
   ```

4. **日志文件权限**
   ```bash
   # 检查日志文件权限
   ls -la /var/log/*.log
   
   # 使用 sudo 运行 (如果需要)
   sudo ./tools/log-parser
   ```

### 调试模式
所有工具都支持 `--verbose` 参数输出详细调试信息：

```bash
./tools/trace-generator --verbose --dry-run
./tools/config-validator --verbose
./tools/log-parser --verbose --analyze
```

## 扩展开发

### 添加新的数据生成场景
编辑 `trace-generator` 中的场景生成函数：
```bash
generate_custom_scenario() {
    # 实现自定义场景逻辑
}
```

### 添加新的配置验证规则
扩展 `config-validator` 的检查函数：
```bash
validate_custom_config() {
    # 实现自定义验证逻辑  
}
```

### 添加新的日志解析规则
扩展 `log-parser` 的分析函数：
```bash
analyze_custom_patterns() {
    # 实现自定义日志模式分析
}
```

## 相关文档

- [分布式追踪用户指南](../docs/user-guide.md)
- [开发环境搭建指南](../docs/developer-guide.md) 
- [自动化脚本文档](../scripts/README.md)
- [故障排查手册](../docs/troubleshooting.md)

---

**提示**: 这些工具是为 SWIT 框架分布式追踪系统专门设计的，但其设计模式和实现方法可以适配到其他类似系统中。
