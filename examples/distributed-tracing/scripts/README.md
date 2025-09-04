# 自动化脚本目录

本目录包含分布式追踪示例项目的自动化脚本，用于环境搭建、测试、演示和维护。

## 脚本列表

### 环境管理脚本

#### `setup.sh` - 环境搭建脚本
**用途**: 一键搭建完整的分布式追踪演示环境  
**功能**:
- 检查系统依赖（Docker, Docker Compose）
- 拉取必要的 Docker 镜像
- 启动 Jaeger 和所有微服务
- 等待服务就绪
- 显示访问地址

**使用方法**:
```bash
./scripts/setup.sh
```

#### `health-check.sh` - 健康检查脚本  
**用途**: 检查所有服务的健康状态  
**功能**:
- 检查 Jaeger UI 可访问性
- 检查各微服务健康端点
- 验证服务间通信
- 显示详细状态报告

**使用方法**:
```bash
./scripts/health-check.sh
```

#### `cleanup.sh` - 环境清理脚本
**用途**: 清理演示环境，释放系统资源  
**功能**:
- 停止所有容器
- 删除容器和网络
- 清理临时数据
- 显示清理结果

**使用方法**:
```bash
./scripts/cleanup.sh
```

### 测试脚本

#### `load-test.sh` - 压力测试脚本
**用途**: 生成负载测试，验证系统性能和追踪功能  
**功能**:
- 并发创建大量订单请求
- 模拟不同的业务场景
- 收集性能指标
- 生成测试报告

**使用方法**:
```bash
# 轻度负载测试
./scripts/load-test.sh light

# 中度负载测试
./scripts/load-test.sh medium

# 重度负载测试  
./scripts/load-test.sh heavy

# 自定义测试
./scripts/load-test.sh custom --requests=1000 --concurrency=50
```

#### `demo-scenarios.sh` - 演示场景脚本
**用途**: 运行预定义的演示场景，展示不同的追踪用例  
**功能**:
- 正常订单流程演示
- 异常处理演示（支付失败、库存不足）
- 性能瓶颈演示
- 复杂调用链演示

**使用方法**:
```bash
# 运行所有演示场景
./scripts/demo-scenarios.sh all

# 运行特定场景
./scripts/demo-scenarios.sh normal-flow
./scripts/demo-scenarios.sh payment-failure
./scripts/demo-scenarios.sh inventory-shortage
./scripts/demo-scenarios.sh performance-bottleneck
```

### 分析和监控脚本

#### `trace-analysis.sh` - 追踪分析脚本
**用途**: 分析 Jaeger 中的追踪数据，生成报告  
**功能**:
- 提取追踪数据
- 分析性能指标
- 识别错误模式
- 生成可视化报告

**使用方法**:
```bash
# 分析最近 1 小时的数据
./scripts/trace-analysis.sh --duration=1h

# 分析特定服务
./scripts/trace-analysis.sh --service=order-service

# 生成完整报告
./scripts/trace-analysis.sh --report=full
```

### 开发辅助脚本

#### `dev-start.sh` - 开发环境启动
**用途**: 启动开发环境，支持代码热重载  
**功能**:
- 启动 Jaeger
- 在开发模式下启动各服务
- 监听代码变化
- 自动重启服务

**使用方法**:
```bash
# 启动所有服务
./scripts/dev-start.sh

# 启动特定服务
./scripts/dev-start.sh order-service
```

#### `logs.sh` - 日志查看脚本
**用途**: 聚合查看所有服务的日志  
**功能**:
- 实时显示所有服务日志
- 按服务过滤日志
- 日志搜索和高亮
- 保存日志到文件

**使用方法**:
```bash
# 查看所有服务日志
./scripts/logs.sh

# 查看特定服务日志
./scripts/logs.sh order-service

# 跟踪日志输出
./scripts/logs.sh --follow

# 保存日志到文件
./scripts/logs.sh --save=logs/trace-demo.log
```

## 脚本依赖

### 系统要求
- **操作系统**: Linux/macOS (Windows 需要 WSL)
- **Docker**: 20.10+
- **Docker Compose**: 1.29+
- **curl**: HTTP 请求工具
- **jq**: JSON 处理工具

### 可选工具
- **hey**: HTTP 负载测试工具（用于 load-test.sh）
- **ab**: Apache 基准测试工具
- **wget**: 文件下载工具

### 安装依赖
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install curl jq

# macOS
brew install jq hey

# 安装 hey（负载测试工具）
go install github.com/rakyll/hey@latest
```

## 配置文件

### `config/demo-config.env`
演示环境的配置参数：
```bash
# 服务端口配置
ORDER_SERVICE_PORT=8081
PAYMENT_SERVICE_PORT=9082
INVENTORY_SERVICE_PORT=8083
JAEGER_UI_PORT=16686

# 测试配置
LOAD_TEST_REQUESTS=100
LOAD_TEST_CONCURRENCY=10
DEMO_DELAY_SECONDS=2

# Jaeger 配置
JAEGER_ENDPOINT=http://localhost:14268/api/traces
JAEGER_SAMPLING_RATE=1.0
```

### `config/test-data.json`
测试数据模板：
```json
{
  "customers": [
    {"id": "customer-001", "name": "张三"},
    {"id": "customer-002", "name": "李四"}
  ],
  "products": [
    {"id": "product-001", "name": "商品A", "price": 99.99},
    {"id": "product-002", "name": "商品B", "price": 149.99}
  ]
}
```

## 使用示例

### 完整演示流程
```bash
# 1. 搭建环境
./scripts/setup.sh

# 2. 验证服务状态
./scripts/health-check.sh

# 3. 运行演示场景
./scripts/demo-scenarios.sh all

# 4. 运行负载测试
./scripts/load-test.sh medium

# 5. 分析追踪数据
./scripts/trace-analysis.sh --duration=30m

# 6. 查看服务日志
./scripts/logs.sh --follow

# 7. 清理环境
./scripts/cleanup.sh
```

### 开发调试流程
```bash
# 1. 启动开发环境
./scripts/dev-start.sh

# 2. 进行代码修改
# ... 编辑代码 ...

# 3. 测试修改效果
curl -X POST http://localhost:8081/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "test", "product_id": "test", "quantity": 1}'

# 4. 查看追踪数据
open http://localhost:16686

# 5. 查看日志
./scripts/logs.sh order-service
```

## 故障排查

### 常见问题

1. **端口占用**
   ```bash
   # 检查端口使用情况
   ./scripts/check-ports.sh
   
   # 停止占用端口的进程
   ./scripts/kill-port.sh 8081
   ```

2. **Docker 权限问题**
   ```bash
   # 添加用户到 docker 组
   sudo usermod -aG docker $USER
   
   # 重新登录或执行
   newgrp docker
   ```

3. **服务启动失败**
   ```bash
   # 查看详细错误信息
   ./scripts/logs.sh --error-only
   
   # 重启特定服务
   ./scripts/restart-service.sh order-service
   ```

### 调试模式

所有脚本支持调试模式：
```bash
# 启用调试输出
DEBUG=1 ./scripts/setup.sh

# 详细日志模式
VERBOSE=1 ./scripts/load-test.sh
```

## 扩展开发

### 添加新脚本
1. 创建脚本文件：`scripts/my-script.sh`
2. 添加执行权限：`chmod +x scripts/my-script.sh`
3. 遵循命名规范和代码风格
4. 更新本 README 文档

### 脚本模板
```bash
#!/bin/bash
set -e  # 错误时退出

# 脚本描述和使用说明
SCRIPT_NAME="$(basename "$0")"
USAGE="Usage: $SCRIPT_NAME [options] [arguments]"

# 默认配置
DEFAULT_CONFIG_FILE="config/demo-config.env"

# 加载配置
if [ -f "$DEFAULT_CONFIG_FILE" ]; then
    source "$DEFAULT_CONFIG_FILE"
fi

# 主函数
main() {
    echo "Starting $SCRIPT_NAME..."
    
    # 脚本逻辑
    
    echo "✅ $SCRIPT_NAME completed successfully"
}

# 执行主函数
main "$@"
```

## 相关文档

- [分布式追踪用户指南](../docs/user-guide.md)
- [开发环境搭建指南](../docs/developer-guide.md)
- [部署和运维指南](../docs/operations-guide.md)
- [故障排查手册](../docs/troubleshooting.md)
