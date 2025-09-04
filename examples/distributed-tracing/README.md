# 分布式追踪示例项目

本项目展示如何在 SWIT 微服务框架中实现和使用 OpenTelemetry 分布式追踪，提供完整的多服务追踪演示、Jaeger 集成配置和自动化部署方案。

## 项目概览

该示例项目包含了一个完整的分布式追踪演示环境，包括：

- **多个互相调用的微服务**：订单服务、支付服务、库存服务
- **Jaeger 追踪后端**：用于收集、存储和可视化追踪数据
- **Docker 环境**：一键启动完整的演示环境
- **自动化脚本**：环境搭建、测试和清理工具
- **详细文档**：使用指南、最佳实践和故障排查

## 快速开始

### 前置条件

- Docker 和 Docker Compose
- Go 1.19 或更高版本
- 确保端口 16686 (Jaeger UI)、8081-8083 (服务端口) 可用

### 启动演示环境

```bash
# 1. 进入项目目录
cd examples/distributed-tracing

# 2. 启动所有服务
./scripts/setup.sh

# 3. 验证服务状态
./scripts/health-check.sh
```

### 访问服务

- **Jaeger UI**: http://localhost:16686
- **订单服务**: http://localhost:8081
- **支付服务**: http://localhost:8082  
- **库存服务**: http://localhost:8083

### 创建示例追踪

```bash
# 创建订单（触发多服务调用链）
curl -X POST http://localhost:8081/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "customer-123",
    "product_id": "product-456", 
    "quantity": 2,
    "amount": 99.99
  }'

# 查看 Jaeger UI 中的追踪数据
open http://localhost:16686
```

## 项目结构

```text
distributed-tracing/
├── README.md                 # 项目概览和快速开始
├── docker-compose.yml        # Docker 环境配置
├── jaeger/                   # Jaeger 配置
│   ├── config/              # Jaeger 配置文件
│   └── README.md            # Jaeger 配置说明
├── services/                 # 示例服务
│   ├── order-service/       # 订单服务
│   ├── payment-service/     # 支付服务
│   ├── inventory-service/   # 库存服务
│   └── README.md            # 服务架构说明
├── scripts/                  # 自动化脚本
│   ├── setup.sh             # 环境搭建
│   ├── health-check.sh      # 健康检查
│   ├── load-test.sh         # 压力测试
│   ├── demo-scenarios.sh    # 演示场景
│   └── cleanup.sh           # 环境清理
└── docs/                     # 详细文档
    ├── user-guide.md        # 用户使用指南
    ├── developer-guide.md   # 开发者指南
    ├── operations-guide.md  # 运维指南
    ├── architecture.md      # 架构设计文档
    └── troubleshooting.md   # 故障排查手册
```

## 演示场景

### 场景 1: 正常订单流程追踪
演示完整的订单创建流程，包括库存检查、支付处理和订单创建的端到端追踪。

### 场景 2: 异常情况处理追踪  
模拟支付失败、库存不足等异常情况，观察错误传播和补偿事务的追踪记录。

### 场景 3: 性能分析和优化
通过压力测试生成大量追踪数据，分析性能瓶颈和优化建议。

## 核心特性

- ✅ **完整的端到端追踪**：从 HTTP 请求到数据库操作的完整追踪链
- ✅ **服务间调用追踪**：gRPC 和 HTTP 调用的自动追踪
- ✅ **错误和异常追踪**：详细的错误信息和堆栈追踪
- ✅ **性能指标收集**：响应时间、吞吐量等关键指标
- ✅ **业务指标记录**：订单金额、商品数量等业务数据
- ✅ **自定义标签和属性**：丰富的上下文信息
- ✅ **采样策略配置**：灵活的数据采集策略

## 相关文档

- [用户使用指南](docs/user-guide.md) - 详细的使用说明和配置选项
- [开发者指南](docs/developer-guide.md) - 如何在服务中集成追踪
- [运维指南](docs/operations-guide.md) - 生产环境部署和监控
- [架构设计](docs/architecture.md) - 追踪系统的架构设计
- [故障排查](docs/troubleshooting.md) - 常见问题和解决方案

## 技术栈

- **追踪框架**: OpenTelemetry Go SDK
- **追踪后端**: Jaeger
- **微服务框架**: SWIT Framework
- **容器化**: Docker & Docker Compose
- **服务通信**: gRPC + HTTP/REST
- **数据库**: SQLite (示例用)

## 贡献指南

请参考项目根目录的 [CONTRIBUTING.md](../../CONTRIBUTING.md) 了解如何贡献代码和报告问题。

## 许可证

本项目采用与 SWIT 框架相同的许可证。
