# 支付服务 (Payment Service)

支付服务负责处理订单的支付请求，模拟第三方支付接口集成，演示分布式追踪在外部服务调用中的应用。

## 服务概述

**服务名称**: payment-service  
**端口配置**: 
- gRPC: 9082（纯 gRPC 服务）

**主要职责**:
- 处理支付请求验证
- 模拟第三方支付 API 调用
- 支付状态管理
- 支付结果通知

## gRPC 接口

```protobuf
service PaymentService {
  rpc ProcessPayment(ProcessPaymentRequest) returns (ProcessPaymentResponse);
  rpc ValidatePayment(ValidatePaymentRequest) returns (ValidatePaymentResponse);
  rpc GetPaymentStatus(GetPaymentStatusRequest) returns (GetPaymentStatusResponse);
  rpc CancelPayment(CancelPaymentRequest) returns (CancelPaymentResponse);
}

message ProcessPaymentRequest {
  string customer_id = 1;
  string order_id = 2;
  double amount = 3;
  string payment_method = 4;
  PaymentDetails details = 5;
}

message ProcessPaymentResponse {
  string transaction_id = 1;
  PaymentStatus status = 2;
  string message = 3;
  int64 processed_at = 4;
}
```

## 业务流程

### 支付处理流程
1. 接收支付请求
2. 验证支付信息（客户ID、金额等）
3. 调用模拟的第三方支付 API
4. 处理支付结果
5. 记录支付状态
6. 返回支付结果

### 支付状态
- `PENDING` - 支付处理中
- `SUCCESS` - 支付成功
- `FAILED` - 支付失败
- `CANCELLED` - 支付取消

## 追踪集成

### 追踪范围
- gRPC 服务调用
- 第三方 API 调用（模拟）
- 数据库操作
- 业务逻辑处理

### 追踪属性
- `payment.transaction_id` - 交易ID
- `payment.amount` - 支付金额
- `payment.method` - 支付方式
- `payment.status` - 支付状态
- `customer.id` - 客户ID
- `external_api.provider` - 第三方支付提供商
- `external_api.response_time` - 外部API响应时间

### 错误追踪
- 支付验证失败
- 第三方 API 超时
- 网络连接错误
- 业务规则违反

## 目录结构

```text
payment-service/
├── README.md              # 服务说明（本文件）
├── main.go               # 服务入口点
├── Dockerfile            # Docker 镜像构建
├── go.mod               # Go 模块依赖
├── config/              # 配置文件
│   └── config.yaml      # 服务配置
├── internal/            # 内部实现
│   ├── service/         # gRPC 服务实现
│   ├── model/          # 数据模型
│   ├── client/         # 第三方 API 客户端
│   └── config/         # 配置结构
├── proto/              # Protocol Buffer 定义
│   └── payment.proto    # 支付服务接口定义
└── tests/              # 测试文件
    ├── integration/    # 集成测试
    └── unit/          # 单元测试
```

## 配置示例

```yaml
server:
  name: "payment-service"
  version: "1.0.0"
  grpc_port: 9082

tracing:
  enabled: true
  jaeger_endpoint: "http://localhost:14268/api/traces"
  service_name: "payment-service"
  sampling_rate: 1.0

payment:
  timeout: "30s"
  retry_count: 3
  simulation_mode: true  # 使用模拟模式

external_apis:
  mock_payment_provider:
    base_url: "https://mock-payment-api.example.com"
    timeout: "10s"
    api_key: "mock-api-key"

logging:
  level: "info"
  format: "json"
```

## 模拟支付逻辑

### 支付成功场景（80%）
- 金额 < 1000: 立即成功
- 金额 >= 1000: 延迟 2-5 秒后成功

### 支付失败场景（15%）
- 金额 > 10000: 金额过大失败
- 随机模拟网络错误

### 支付超时场景（5%）
- 随机模拟第三方 API 超时

## 性能特征

### 响应时间
- 正常支付: 100-500ms
- 大额支付: 2-5秒
- 超时场景: 30秒

### 吞吐量
- 支持并发处理: 100+ TPS
- 内存占用: < 100MB
- CPU 使用率: < 10%

## 开发和测试

### 本地运行
```bash
cd payment-service
go mod tidy
go run main.go
```

### Docker 运行
```bash
docker build -t payment-service .
docker run -p 9082:9082 payment-service
```

### gRPC 测试
```bash
# 使用 grpcurl 测试
grpcurl -plaintext \
  -d '{"customer_id":"test-customer","order_id":"test-order","amount":99.99,"payment_method":"credit_card"}' \
  localhost:9082 \
  payment.PaymentService/ProcessPayment
```

### 测试用例
```bash
# 单元测试
go test ./...

# 集成测试
go test ./tests/integration/...

# 压力测试
go test -bench=. ./tests/benchmark/...
```

## 监控指标

### 业务指标
- 支付成功率
- 平均支付金额
- 每分钟支付笔数
- 支付方式分布

### 技术指标
- gRPC 请求响应时间
- 错误率
- 第三方 API 调用延迟
- 服务可用性

*注：具体实现将在后续阶段完成。*
