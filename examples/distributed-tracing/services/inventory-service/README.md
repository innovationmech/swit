# 库存服务 (Inventory Service)

库存服务负责管理商品库存，提供库存查询、预留、更新等功能，演示数据库操作的分布式追踪。

## 服务概述

**服务名称**: inventory-service  
**端口配置**: 
- HTTP: 8083
- gRPC: 9083

**主要职责**:
- 管理商品库存信息
- 处理库存查询请求
- 执行库存预留和释放
- 记录库存变更历史

## API 接口

### HTTP 接口

#### 查询库存
```http
GET /api/v1/inventory/{product_id}
```

#### 预留库存
```http
POST /api/v1/inventory/reserve
Content-Type: application/json

{
  "product_id": "product-456",
  "quantity": 2,
  "order_id": "order-123",
  "reservation_timeout": 300
}
```

#### 释放库存
```http
POST /api/v1/inventory/release
Content-Type: application/json

{
  "product_id": "product-456", 
  "quantity": 2,
  "order_id": "order-123"
}
```

#### 更新库存
```http
PUT /api/v1/inventory/{product_id}
Content-Type: application/json

{
  "quantity": 100,
  "operation": "set" // set, add, subtract
}
```

#### 健康检查
```http
GET /health
```

### gRPC 接口

```protobuf
service InventoryService {
  rpc CheckInventory(CheckInventoryRequest) returns (CheckInventoryResponse);
  rpc ReserveInventory(ReserveInventoryRequest) returns (ReserveInventoryResponse);
  rpc ReleaseInventory(ReleaseInventoryRequest) returns (ReleaseInventoryResponse);
  rpc UpdateInventory(UpdateInventoryRequest) returns (UpdateInventoryResponse);
  rpc GetInventoryHistory(GetInventoryHistoryRequest) returns (GetInventoryHistoryResponse);
}

message CheckInventoryRequest {
  string product_id = 1;
  int32 required_quantity = 2;
}

message CheckInventoryResponse {
  bool available = 1;
  int32 current_quantity = 2;
  int32 reserved_quantity = 3;
  string message = 4;
}
```

## 业务流程

### 库存检查流程
1. 接收库存检查请求
2. 查询数据库中的库存信息
3. 计算可用库存（总库存 - 已预留）
4. 返回库存状态

### 库存预留流程
1. 验证库存充足性
2. 创建库存预留记录
3. 更新预留库存数量
4. 设置预留超时时间
5. 返回预留结果

### 库存释放流程
1. 查找对应的预留记录
2. 减少预留库存数量
3. 删除或标记预留记录
4. 记录操作历史

## 数据模型

### 库存表 (inventory)
```sql
CREATE TABLE inventory (
    product_id VARCHAR(255) PRIMARY KEY,
    total_quantity INT NOT NULL DEFAULT 0,
    reserved_quantity INT NOT NULL DEFAULT 0,
    available_quantity INT GENERATED ALWAYS AS (total_quantity - reserved_quantity),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### 库存预留表 (inventory_reservations)
```sql
CREATE TABLE inventory_reservations (
    id VARCHAR(255) PRIMARY KEY,
    product_id VARCHAR(255) NOT NULL,
    order_id VARCHAR(255) NOT NULL,
    quantity INT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES inventory(product_id)
);
```

### 库存历史表 (inventory_history)
```sql
CREATE TABLE inventory_history (
    id VARCHAR(255) PRIMARY KEY,
    product_id VARCHAR(255) NOT NULL,
    operation_type ENUM('check', 'reserve', 'release', 'update') NOT NULL,
    quantity_change INT NOT NULL,
    order_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSON
);
```

## 追踪集成

### 追踪范围
- HTTP/gRPC 请求处理
- 数据库查询和更新操作
- 业务逻辑执行
- 缓存操作（如果使用）

### 追踪属性
- `inventory.product_id` - 商品ID
- `inventory.quantity` - 操作数量
- `inventory.operation` - 操作类型
- `inventory.available` - 可用库存
- `inventory.reserved` - 预留库存
- `order.id` - 关联订单ID
- `db.statement` - 数据库查询语句
- `db.rows_affected` - 影响的行数

### 数据库追踪
- SQL 查询语句追踪
- 查询执行时间
- 连接池状态
- 事务操作

## 目录结构

```text
inventory-service/
├── README.md              # 服务说明（本文件）
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
│   └── inventory.proto  # 库存服务接口定义
├── migrations/         # 数据库迁移脚本
│   └── 001_initial.sql  # 初始化脚本
└── tests/              # 测试文件
    ├── integration/    # 集成测试
    └── unit/          # 单元测试
```

## 配置示例

```yaml
server:
  name: "inventory-service"
  version: "1.0.0"
  http_port: 8083
  grpc_port: 9083

database:
  driver: "sqlite"
  dsn: "./data/inventory.db"
  max_connections: 10
  max_idle_connections: 5
  connection_timeout: "10s"

tracing:
  enabled: true
  jaeger_endpoint: "http://localhost:14268/api/traces"
  service_name: "inventory-service"
  sampling_rate: 1.0

cache:
  enabled: true
  type: "memory"  # memory, redis
  ttl: "5m"
  max_size: 1000

inventory:
  default_reservation_timeout: "5m"
  cleanup_interval: "1m"  # 清理过期预留

logging:
  level: "info"
  format: "json"
```

## 业务逻辑

### 库存可用性检查
```go
func (s *InventoryService) CheckInventory(productID string, requiredQuantity int) (bool, error) {
    // 1. 查询总库存
    inventory, err := s.repo.GetInventory(productID)
    if err != nil {
        return false, err
    }
    
    // 2. 计算可用库存
    available := inventory.TotalQuantity - inventory.ReservedQuantity
    
    // 3. 检查是否充足
    return available >= requiredQuantity, nil
}
```

### 自动清理过期预留
```go
func (s *InventoryService) CleanupExpiredReservations() {
    ticker := time.NewTicker(s.config.CleanupInterval)
    for range ticker.C {
        expiredReservations := s.repo.GetExpiredReservations()
        for _, reservation := range expiredReservations {
            s.ReleaseInventory(reservation.ProductID, reservation.Quantity, reservation.OrderID)
        }
    }
}
```

## 性能优化

### 缓存策略
- 热点商品库存信息缓存
- 缓存 TTL: 5分钟
- 写入时失效策略

### 数据库优化
- 产品ID索引
- 预留记录定期清理
- 连接池配置优化

### 并发控制
- 乐观锁防止超卖
- 事务隔离级别配置
- 库存更新原子性保证

## 开发和测试

### 本地运行
```bash
cd inventory-service
go mod tidy

# 初始化数据库
sqlite3 data/inventory.db < migrations/001_initial.sql

# 启动服务
go run main.go
```

### Docker 运行
```bash
docker build -t inventory-service .
docker run -p 8083:8083 -p 9083:9083 inventory-service
```

### API 测试
```bash
# 查询库存
curl http://localhost:8083/api/v1/inventory/product-123

# 预留库存
curl -X POST http://localhost:8083/api/v1/inventory/reserve \
  -H "Content-Type: application/json" \
  -d '{"product_id":"product-123","quantity":2,"order_id":"order-456"}'
```

### 测试用例
```bash
# 单元测试
go test ./...

# 集成测试（需要数据库）
go test ./tests/integration/...

# 并发测试
go test -race ./...
```

## 监控指标

### 业务指标
- 总库存价值
- 库存周转率
- 预留超时率
- 热门商品排行

### 技术指标
- API 响应时间
- 数据库连接数
- 缓存命中率
- 错误率统计

## 故障场景

### 库存不足
- 返回明确的错误信息
- 记录详细的追踪信息
- 建议相似商品

### 数据库连接失败
- 自动重试机制
- 降级到只读模式
- 告警通知

### 预留超时处理
- 后台定时清理
- 自动释放过期预留
- 记录清理日志

*注：具体实现将在后续阶段完成。*
