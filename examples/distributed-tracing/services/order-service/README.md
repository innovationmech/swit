# 订单服务 (Order Service)

订单服务是分布式追踪演示系统的核心服务，负责处理订单创建、状态管理和协调其他服务。

## 服务概述

**服务名称**: order-service  
**端口配置**: 
- HTTP: 8081
- gRPC: 9081

**主要职责**:
- 接收和处理订单创建请求
- 协调库存检查和支付处理
- 管理订单生命周期
- 提供订单查询接口

## API 接口

### HTTP 接口

#### 创建订单
```http
POST /api/v1/orders
Content-Type: application/json

{
  "customer_id": "customer-123",
  "product_id": "product-456",
  "quantity": 2,
  "amount": 99.99
}
```

#### 查询订单
```http
GET /api/v1/orders/{order_id}
```

#### 健康检查
```http
GET /health
```

### gRPC 接口

```protobuf
service OrderService {
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse);
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);
}
```

## 业务流程

### 订单创建流程
1. 接收订单创建请求
2. 调用库存服务检查库存
3. 调用支付服务处理支付
4. 预留库存
5. 创建订单记录
6. 返回订单信息

## 追踪集成

该服务集成了完整的 OpenTelemetry 追踪功能：

### 自动追踪
- HTTP 请求/响应
- gRPC 调用
- 数据库操作

### 手动追踪  
- 业务逻辑步骤
- 关键决策点
- 错误和异常

### 追踪属性
- `order.id` - 订单ID
- `customer.id` - 客户ID
- `product.id` - 商品ID
- `order.amount` - 订单金额
- `order.quantity` - 订单数量

## 目录结构

```text
order-service/
├── README.md              # 服务说明（本文件）
├── main.go               # 服务入口点
├── Dockerfile            # Docker 镜像构建
├── go.mod               # Go 模块依赖
├── config/              # 配置文件
├── internal/            # 内部实现
├── proto/              # Protocol Buffer 定义
└── tests/              # 测试文件
```

## 配置示例

```yaml
server:
  name: "order-service"
  version: "1.0.0"
  http_port: 8081
  grpc_port: 9081

database:
  driver: "sqlite"
  dsn: "./data/orders.db"

tracing:
  enabled: true
  jaeger_endpoint: "http://localhost:14268/api/traces"
  service_name: "order-service"
  sampling_rate: 1.0

external_services:
  payment_service_url: "localhost:9082"
  inventory_service_url: "localhost:9083"
```

## 开发和测试

### 本地运行
```bash
cd order-service
go mod tidy
go run main.go
```

### Docker 运行
```bash
docker build -t order-service .
docker run -p 8081:8081 -p 9081:9081 order-service
```

### 测试
```bash
# 单元测试
go test ./...

# 集成测试
go test ./tests/integration/...
```

*注：具体实现将在后续阶段完成。*
