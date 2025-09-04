# 示例服务目录

本目录包含分布式追踪演示的三个微服务实现，展示服务间调用的完整追踪链路。

## 服务架构

```text
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│   Order Service │───▶│  Payment Service │    │  Inventory Service  │
│   (订单服务)     │    │   (支付服务)      │    │    (库存服务)       │
│                 │    │                  │    │                     │
│ Port: 8081      │    │ Port: 8082       │    │ Port: 8083          │
│ HTTP + gRPC     │    │ gRPC only        │    │ HTTP + gRPC         │
└─────────────────┘    └──────────────────┘    └─────────────────────┘
         │                       ▲                        ▲
         │                       │                        │
         └───────────────────────┼────────────────────────┘
                                 │
                        创建订单流程调用链
```

## 服务说明

### 订单服务 (Order Service)
**目录**: `order-service/`  
**端口**: 8081 (HTTP), 9081 (gRPC)  
**职责**: 
- 处理订单创建请求
- 协调支付和库存服务
- 管理订单生命周期
- 提供订单查询接口

**API 端点**:
- `POST /api/v1/orders` - 创建订单
- `GET /api/v1/orders/{id}` - 查询订单
- `GET /health` - 健康检查

### 支付服务 (Payment Service)
**目录**: `payment-service/`  
**端口**: 9082 (gRPC only)  
**职责**:
- 处理支付请求
- 验证支付信息
- 模拟第三方支付接口
- 支付结果通知

**gRPC 服务**:
- `ProcessPayment` - 处理支付
- `ValidatePayment` - 验证支付
- `GetPaymentStatus` - 查询支付状态

### 库存服务 (Inventory Service)  
**目录**: `inventory-service/`  
**端口**: 8083 (HTTP), 9083 (gRPC)  
**职责**:
- 管理商品库存
- 库存检查和预留
- 库存更新操作
- 库存历史记录

**API 端点**:
- `GET /api/v1/inventory/{product_id}` - 查询库存
- `POST /api/v1/inventory/reserve` - 预留库存  
- `POST /api/v1/inventory/release` - 释放库存
- `GET /health` - 健康检查

## 调用链路

### 正常订单流程
1. **客户端** → `POST /api/v1/orders` → **订单服务**
2. **订单服务** → `CheckInventory(gRPC)` → **库存服务**
3. **订单服务** → `ProcessPayment(gRPC)` → **支付服务**
4. **订单服务** → `ReserveInventory(gRPC)` → **库存服务**
5. **订单服务** → 返回订单结果 → **客户端**

### 异常处理流程
- **库存不足**: 直接返回错误，不调用支付服务
- **支付失败**: 记录失败原因，不预留库存
- **库存预留失败**: 触发支付补偿事务

## 追踪集成

每个服务都集成了 OpenTelemetry 追踪：

### 自动追踪
- HTTP 请求/响应
- gRPC 调用
- 数据库操作
- 外部 API 调用

### 手动追踪
- 业务逻辑步骤
- 关键决策点
- 错误和异常
- 业务指标记录

### 追踪属性
- `service.name` - 服务名称
- `service.version` - 服务版本
- `order.id` - 订单ID
- `customer.id` - 客户ID
- `product.id` - 商品ID
- `payment.amount` - 支付金额
- `inventory.quantity` - 库存数量

## 技术栈

### 共同技术栈
- **框架**: SWIT Microservice Framework
- **语言**: Go 1.19+
- **追踪**: OpenTelemetry Go SDK
- **配置**: Viper + YAML
- **日志**: Zap structured logging
- **健康检查**: 内置健康检查端点

### 订单服务
- **HTTP**: Gin web framework  
- **gRPC**: Google gRPC
- **数据库**: SQLite (示例用)
- **ORM**: GORM

### 支付服务
- **通信**: gRPC only
- **模拟**: 第三方支付 API
- **存储**: 内存存储

### 库存服务  
- **HTTP**: Gin web framework
- **gRPC**: Google gRPC
- **数据库**: SQLite (示例用)
- **缓存**: 内存缓存

## 目录结构

每个服务的目录结构：

```text
{service-name}/
├── README.md              # 服务说明文档
├── main.go               # 服务入口点
├── Dockerfile            # Docker 镜像构建
├── go.mod               # Go 模块依赖
├── config/              # 配置文件
│   └── config.yaml      # 服务配置
├── internal/            # 内部实现
│   ├── handler/         # HTTP 处理器
│   ├── service/         # gRPC 服务实现
│   ├── model/          # 数据模型
│   ├── repository/     # 数据访问层
│   └── config/         # 配置结构
├── proto/              # Protocol Buffer 定义
│   └── {service}.proto  # 服务接口定义
└── tests/              # 测试文件
    ├── integration/    # 集成测试
    └── unit/          # 单元测试
```

## 开发指南

### 本地开发
```bash
# 启动单个服务
cd order-service
go run main.go

# 启动所有服务（需要在各自终端）
./scripts/dev-start.sh
```

### Docker 开发
```bash
# 构建服务镜像
docker build -t order-service ./order-service

# 使用 Docker Compose 启动
docker-compose up -d
```

### 测试
```bash
# 运行单元测试
cd order-service
go test ./...

# 运行集成测试
cd order-service
go test ./tests/integration/...
```

## 配置说明

每个服务支持以下配置：

```yaml
server:
  name: "order-service"
  version: "1.0.0"
  http_port: 8081
  grpc_port: 9081

tracing:
  enabled: true
  jaeger_endpoint: "http://localhost:14268/api/traces"
  service_name: "order-service"
  sampling_rate: 1.0

database:
  driver: "sqlite"
  dsn: "./data/orders.db"

logging:
  level: "info"
  format: "json"
```

## 部署注意事项

1. **环境变量**: 支持环境变量覆盖配置
2. **健康检查**: 提供 `/health` 端点
3. **优雅关闭**: 支持 SIGTERM 信号处理
4. **资源限制**: Docker 容器资源限制配置
5. **网络隔离**: Docker 网络配置

## 相关文档

- [分布式追踪用户指南](../docs/user-guide.md)
- [开发者集成指南](../docs/developer-guide.md)
- [部署和运维指南](../docs/operations-guide.md)
- [API 文档](../docs/api-reference.md)
