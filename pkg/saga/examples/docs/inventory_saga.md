# 库存管理 Saga 示例文档

## 概述

库存管理 Saga（`inventory_saga.go`）展示了如何使用 Swit 框架处理多仓库库存协调的复杂分布式事务。这个示例演示了如何在多个仓库间智能分配库存，处理库存预留、锁定和提货单生成等操作。

## 业务场景

### 适用场景

- 多仓库电商平台的库存管理
- 供应链管理系统的库存调配
- 物流系统的货物分配
- 制造业的原料分配
- 零售连锁的库存协调

### 业务流程

```
库存请求 → 查询仓库 → 检查可用性 → 分配库存 → 锁定库存 → 生成提货单
```

**关键特点**：
- 多仓库协调
- 智能分配策略
- 库存预留机制
- 自动过期管理

## 架构设计

### 服务依赖关系

```
┌─────────────────┐
│  Saga           │
│  Coordinator    │
└────────┬────────┘
         │
    ┌────┴──────────┬──────────────┬──────────────┬──────────────┐
    │               │              │              │              │
┌───▼──────────┐ ┌──▼────────┐ ┌──▼──────────┐ ┌──▼──────────┐ ┌──▼──────────┐
│ 仓库服务     │ │ 库存服务  │ │ 分配服务    │ │ 物流服务    │ │ 通知服务    │
│(Warehouse)   │ │(Inventory)│ │(Allocation) │ │(Logistics)  │ │(Notify)     │
└──────────────┘ └───────────┘ └─────────────┘ └─────────────┘ └─────────────┘
```

### 步骤流程图

```
┌──────────────────────────────────────────────────────────┐
│                  库存管理 Saga                            │
└──────────────────────────────────────────────────────────┘

步骤 1: QueryWarehousesStep (查询可用仓库)
  ├─ Execute: 查询可以提供商品的仓库列表
  └─ Compensate: 无需补偿

步骤 2: CheckAvailabilityStep (检查库存可用性)
  ├─ Execute: 检查各仓库的库存可用性
  └─ Compensate: 无需补偿

步骤 3: AllocateInventoryStep (分配库存)
  ├─ Execute: 根据策略分配库存到具体仓库
  └─ Compensate: 取消分配

步骤 4: LockInventoryStep (锁定库存)
  ├─ Execute: 锁定已分配的库存
  └─ Compensate: 解锁库存

步骤 5: GeneratePicklistStep (生成提货单)
  ├─ Execute: 生成仓库提货单
  └─ Compensate: 取消提货单
```

### 数据流转

```
InventoryRequestData (输入)
    │
    ├─> QueryWarehousesStep
    │     └─> WarehouseListResult
    │           │
    │           ├─> CheckAvailabilityStep
    │           │     └─> AvailabilityCheckResult
    │           │           │
    │           │           ├─> AllocateInventoryStep
    │           │           │     └─> AllocationResult
    │           │           │           │
    │           │           │           ├─> LockInventoryStep
    │           │           │           │     └─> LockResult
    │           │           │           │           │
    │           │           │           │           └─> GeneratePicklistStep
    │           │           │           │                 └─> PicklistResult (输出)
    │           │           │           │
    │           │           │           └─ (失败) ─> 补偿 LockInventoryStep
    │           │           │
    │           │           └─ (失败) ─> 补偿 AllocateInventoryStep
    │           │
    │           └─ (失败) ─> 无需补偿
    │
    └─ (失败) ─> 无需补偿
```

## 数据结构

### 输入数据

#### InventoryRequestData

```go
type InventoryRequestData struct {
    // 请求基本信息
    RequestID   string      // 请求ID
    OrderID     string      // 关联的订单ID
    CustomerID  string      // 客户ID
    RequestType string      // 请求类型（reservation、allocation、transfer）
    
    // 商品信息
    Items []InventoryItem  // 需要处理的商品项
    
    // 仓库偏好设置
    PreferredWarehouses []string // 优先选择的仓库列表
    AllocationStrategy  string   // 分配策略（priority、nearest、load_balance）
    
    // 优先级设置
    Priority  int       // 优先级（1-10，10最高）
    IsUrgent  bool      // 是否紧急
    ExpiredAt time.Time // 过期时间
    
    // 元数据
    Metadata map[string]interface{} // 额外元数据
}
```

**分配策略说明**：
- `priority`: 按仓库优先级分配
- `nearest`: 按距离客户最近分配
- `load_balance`: 按仓库负载均衡分配
- `cost_optimize`: 按成本最优分配

#### InventoryItem

```go
type InventoryItem struct {
    ProductID   string // 商品ID
    SKU         string // 库存单位
    Quantity    int    // 数量
    Unit        string // 单位（piece、box、kg等）
    Description string // 描述
}
```

### 步骤结果

#### WarehouseInfo

```go
type WarehouseInfo struct {
    WarehouseID   string    // 仓库ID
    WarehouseName string    // 仓库名称
    Location      string    // 地理位置
    Capacity      int       // 总容量
    Available     int       // 可用容量
    Status        string    // 仓库状态（active、maintenance、closed）
    Priority      int       // 优先级
    LastUpdated   time.Time // 最后更新时间
}
```

#### WarehouseListResult

```go
type WarehouseListResult struct {
    RequestID  string          // 请求ID
    Warehouses []WarehouseInfo // 可用仓库列表
    TotalCount int             // 总数
    QueriedAt  time.Time       // 查询时间
    Metadata   map[string]interface{}
}
```

#### AvailabilityCheckResult

```go
type AvailabilityCheckResult struct {
    RequestID        string                   // 请求ID
    AvailableItems   []AvailableItemInfo      // 可用的商品信息
    UnavailableItems []UnavailableItemInfo    // 不可用的商品信息
    TotalAvailable   bool                     // 是否全部可用
    CheckedAt        time.Time                // 检查时间
    Metadata         map[string]interface{}
}
```

#### AllocationResult

```go
type AllocationResult struct {
    AllocationID string                  // 分配ID
    RequestID    string                  // 请求ID
    Allocations  []WarehouseAllocation   // 各仓库分配详情
    Strategy     string                  // 使用的分配策略
    AllocatedAt  time.Time               // 分配时间
    ExpiresAt    time.Time               // 过期时间
    Metadata     map[string]interface{}
}
```

#### LockResult

```go
type LockResult struct {
    LockID      string                  // 锁定ID
    RequestID   string                  // 请求ID
    LockedItems []LockedInventoryItem   // 锁定的库存项
    LockedAt    time.Time               // 锁定时间
    ExpiresAt   time.Time               // 过期时间
    Metadata    map[string]interface{}
}
```

#### PicklistResult

```go
type PicklistResult struct {
    PicklistID   string                  // 提货单ID
    RequestID    string                  // 请求ID
    Warehouse    string                  // 仓库ID
    Items        []PicklistItem          // 提货项
    Status       string                  // 状态（pending、processing、completed）
    GeneratedAt  time.Time               // 生成时间
    Metadata     map[string]interface{}
}
```

## 步骤详解

### 步骤 1: QueryWarehousesStep（查询可用仓库）

#### 功能描述

查询可以提供所需商品的仓库列表，根据仓库状态、容量和地理位置等因素筛选。

#### 查询条件

1. 仓库状态必须是 `active`
2. 仓库有足够的可用容量
3. 仓库支持所需的商品类型
4. 如果有优先仓库列表，优先返回

#### 执行逻辑

```go
func (s *QueryWarehousesStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    requestData := data.(*InventoryRequestData)
    
    // 查询条件
    query := &WarehouseQuery{
        Status:              "active",
        MinAvailableCapacity: calculateRequiredCapacity(requestData.Items),
        ProductIDs:          extractProductIDs(requestData.Items),
    }
    
    // 如果有优先仓库，添加到查询条件
    if len(requestData.PreferredWarehouses) > 0 {
        query.PreferredIDs = requestData.PreferredWarehouses
    }
    
    // 查询仓库
    warehouses, err := s.service.QueryWarehouses(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("查询仓库失败: %w", err)
    }
    
    if len(warehouses) == 0 {
        return nil, errors.New("没有可用的仓库")
    }
    
    // 按优先级排序
    sort.Slice(warehouses, func(i, j int) bool {
        return warehouses[i].Priority > warehouses[j].Priority
    })
    
    return &WarehouseListResult{
        RequestID:  requestData.RequestID,
        Warehouses: warehouses,
        TotalCount: len(warehouses),
        QueriedAt:  time.Now(),
    }, nil
}
```

#### 补偿逻辑

```go
func (s *QueryWarehousesStep) Compensate(ctx context.Context, data interface{}) error {
    // 查询操作无需补偿
    return nil
}
```

#### 配置参数

- **超时时间**: 5秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 网络错误可重试

### 步骤 2: CheckAvailabilityStep（检查库存可用性）

#### 功能描述

检查各仓库的库存可用性，确认是否有足够的库存满足需求。

#### 检查逻辑

1. 遍历所有仓库
2. 检查每个商品的可用库存
3. 计算总可用数量
4. 标记不可用的商品

#### 执行逻辑

```go
func (s *CheckAvailabilityStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    warehouseResult := data.(*WarehouseListResult)
    
    // 从元数据获取商品列表
    items := warehouseResult.Metadata["items"].([]InventoryItem)
    
    availableItems := []AvailableItemInfo{}
    unavailableItems := []UnavailableItemInfo{}
    
    // 检查每个仓库的每个商品
    for _, warehouse := range warehouseResult.Warehouses {
        for _, item := range items {
            inventory, err := s.service.CheckInventory(ctx, warehouse.WarehouseID, item.SKU)
            if err != nil {
                continue
            }
            
            if inventory.Available >= item.Quantity {
                availableItems = append(availableItems, AvailableItemInfo{
                    WarehouseID: warehouse.WarehouseID,
                    SKU:         item.SKU,
                    Available:   inventory.Available,
                    Reserved:    inventory.Reserved,
                })
            } else {
                unavailableItems = append(unavailableItems, UnavailableItemInfo{
                    WarehouseID: warehouse.WarehouseID,
                    SKU:         item.SKU,
                    Required:    item.Quantity,
                    Available:   inventory.Available,
                    Shortage:    item.Quantity - inventory.Available,
                })
            }
        }
    }
    
    // 判断是否全部可用
    totalAvailable := len(unavailableItems) == 0
    
    if !totalAvailable {
        return nil, &InsufficientInventoryError{
            Items: unavailableItems,
        }
    }
    
    return &AvailabilityCheckResult{
        RequestID:        warehouseResult.RequestID,
        AvailableItems:   availableItems,
        UnavailableItems: unavailableItems,
        TotalAvailable:   totalAvailable,
        CheckedAt:        time.Now(),
    }, nil
}
```

#### 补偿逻辑

```go
func (s *CheckAvailabilityStep) Compensate(ctx context.Context, data interface{}) error {
    // 检查操作无需补偿
    return nil
}
```

#### 配置参数

- **超时时间**: 10秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 网络错误和临时不可用可重试

### 步骤 3: AllocateInventoryStep（分配库存）

#### 功能描述

根据配置的分配策略，将商品分配到具体的仓库。

#### 分配策略

##### 优先级策略（Priority）

```go
func (s *AllocationService) AllocateByPriority(items []InventoryItem, warehouses []WarehouseInfo) []WarehouseAllocation {
    allocations := []WarehouseAllocation{}
    
    // 按优先级排序仓库
    sort.Slice(warehouses, func(i, j int) bool {
        return warehouses[i].Priority > warehouses[j].Priority
    })
    
    for _, item := range items {
        remaining := item.Quantity
        
        for _, warehouse := range warehouses {
            if remaining == 0 {
                break
            }
            
            // 获取该仓库可提供的数量
            available := s.getAvailableQuantity(warehouse.WarehouseID, item.SKU)
            allocated := min(remaining, available)
            
            if allocated > 0 {
                allocations = append(allocations, WarehouseAllocation{
                    WarehouseID: warehouse.WarehouseID,
                    SKU:         item.SKU,
                    Quantity:    allocated,
                })
                remaining -= allocated
            }
        }
        
        if remaining > 0 {
            return nil // 分配失败
        }
    }
    
    return allocations
}
```

##### 最近仓库策略（Nearest）

```go
func (s *AllocationService) AllocateByNearest(items []InventoryItem, warehouses []WarehouseInfo, customerLocation string) []WarehouseAllocation {
    // 按距离排序仓库
    sort.Slice(warehouses, func(i, j int) bool {
        distI := calculateDistance(warehouses[i].Location, customerLocation)
        distJ := calculateDistance(warehouses[j].Location, customerLocation)
        return distI < distJ
    })
    
    // 类似优先级策略的分配逻辑
    // ...
}
```

##### 负载均衡策略（LoadBalance）

```go
func (s *AllocationService) AllocateByLoadBalance(items []InventoryItem, warehouses []WarehouseInfo) []WarehouseAllocation {
    // 计算每个仓库的负载率
    loads := make(map[string]float64)
    for _, warehouse := range warehouses {
        loads[warehouse.WarehouseID] = float64(warehouse.Available) / float64(warehouse.Capacity)
    }
    
    // 按负载率排序，优先使用负载低的仓库
    sort.Slice(warehouses, func(i, j int) bool {
        return loads[warehouses[i].WarehouseID] > loads[warehouses[j].WarehouseID]
    })
    
    // 执行分配
    // ...
}
```

#### 执行逻辑

```go
func (s *AllocateInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    availabilityResult := data.(*AvailabilityCheckResult)
    
    // 从元数据获取分配策略和商品
    strategy := availabilityResult.Metadata["strategy"].(string)
    items := availabilityResult.Metadata["items"].([]InventoryItem)
    warehouses := availabilityResult.Metadata["warehouses"].([]WarehouseInfo)
    
    // 根据策略执行分配
    var allocations []WarehouseAllocation
    var err error
    
    switch strategy {
    case "priority":
        allocations, err = s.service.AllocateByPriority(ctx, items, warehouses)
    case "nearest":
        customerLocation := availabilityResult.Metadata["customer_location"].(string)
        allocations, err = s.service.AllocateByNearest(ctx, items, warehouses, customerLocation)
    case "load_balance":
        allocations, err = s.service.AllocateByLoadBalance(ctx, items, warehouses)
    default:
        return nil, fmt.Errorf("未知的分配策略: %s", strategy)
    }
    
    if err != nil {
        return nil, fmt.Errorf("库存分配失败: %w", err)
    }
    
    // 生成分配ID
    allocationID := generateAllocationID()
    
    // 设置30分钟过期
    expiresAt := time.Now().Add(30 * time.Minute)
    
    return &AllocationResult{
        AllocationID: allocationID,
        RequestID:    availabilityResult.RequestID,
        Allocations:  allocations,
        Strategy:     strategy,
        AllocatedAt:  time.Now(),
        ExpiresAt:    expiresAt,
    }, nil
}
```

#### 补偿逻辑

```go
func (s *AllocateInventoryStep) Compensate(ctx context.Context, data interface{}) error {
    allocationResult := data.(*AllocationResult)
    
    // 取消分配
    err := s.service.CancelAllocation(ctx, allocationResult.AllocationID)
    if err != nil {
        return fmt.Errorf("取消分配失败: %w", err)
    }
    
    return nil
}
```

#### 配置参数

- **超时时间**: 10秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 网络错误可重试，分配冲突不可重试

### 步骤 4: LockInventoryStep（锁定库存）

#### 功能描述

锁定已分配的库存，防止被其他订单占用。

#### 执行逻辑

```go
func (s *LockInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    allocationResult := data.(*AllocationResult)
    
    lockedItems := []LockedInventoryItem{}
    
    // 锁定每个分配的库存
    for _, allocation := range allocationResult.Allocations {
        lockResult, err := s.service.LockInventory(ctx, &LockRequest{
            WarehouseID: allocation.WarehouseID,
            SKU:         allocation.SKU,
            Quantity:    allocation.Quantity,
            AllocationID: allocationResult.AllocationID,
        })
        
        if err != nil {
            // 锁定失败，需要回滚已锁定的库存
            s.rollbackLocks(ctx, lockedItems)
            return nil, fmt.Errorf("锁定库存失败: %w", err)
        }
        
        lockedItems = append(lockedItems, LockedInventoryItem{
            LockID:      lockResult.LockID,
            WarehouseID: allocation.WarehouseID,
            SKU:         allocation.SKU,
            Quantity:    allocation.Quantity,
        })
    }
    
    // 生成总锁定ID
    lockID := generateLockID()
    
    // 设置30分钟自动过期
    expiresAt := time.Now().Add(30 * time.Minute)
    
    return &LockResult{
        LockID:      lockID,
        RequestID:   allocationResult.RequestID,
        LockedItems: lockedItems,
        LockedAt:    time.Now(),
        ExpiresAt:   expiresAt,
    }, nil
}
```

#### 补偿逻辑

```go
func (s *LockInventoryStep) Compensate(ctx context.Context, data interface{}) error {
    lockResult := data.(*LockResult)
    
    // 解锁所有已锁定的库存
    for _, item := range lockResult.LockedItems {
        err := s.service.UnlockInventory(ctx, item.LockID)
        if err != nil {
            // 记录错误但继续尝试解锁其他项
            log.Printf("解锁库存失败: %v", err)
        }
    }
    
    return nil
}
```

#### 配置参数

- **超时时间**: 15秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 网络错误和临时锁冲突可重试

### 步骤 5: GeneratePicklistStep（生成提货单）

#### 功能描述

为每个仓库生成提货单，指导仓库人员拣货。

#### 提货单内容

- 提货单号
- 订单信息
- 商品列表（SKU、数量、货架位置）
- 优先级
- 生成时间

#### 执行逻辑

```go
func (s *GeneratePicklistStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    lockResult := data.(*LockResult)
    
    // 按仓库分组锁定的库存
    byWarehouse := groupByWarehouse(lockResult.LockedItems)
    
    picklists := []PicklistResult{}
    
    // 为每个仓库生成提货单
    for warehouseID, items := range byWarehouse {
        picklist, err := s.service.GeneratePicklist(ctx, &PicklistRequest{
            WarehouseID: warehouseID,
            RequestID:   lockResult.RequestID,
            Items:       items,
            Priority:    getPriorityFromMetadata(lockResult.Metadata),
        })
        
        if err != nil {
            return nil, fmt.Errorf("生成提货单失败: %w", err)
        }
        
        picklists = append(picklists, *picklist)
    }
    
    return &MultiPicklistResult{
        RequestID: lockResult.RequestID,
        Picklists: picklists,
        GeneratedAt: time.Now(),
    }, nil
}
```

#### 补偿逻辑

```go
func (s *GeneratePicklistStep) Compensate(ctx context.Context, data interface{}) error {
    result := data.(*MultiPicklistResult)
    
    // 取消所有提货单
    for _, picklist := range result.Picklists {
        err := s.service.CancelPicklist(ctx, picklist.PicklistID)
        if err != nil {
            log.Printf("取消提货单失败: %v", err)
        }
    }
    
    return nil
}
```

#### 配置参数

- **超时时间**: 10秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 所有错误都可重试

## 使用示例

### 基本用法

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/innovationmech/swit/pkg/saga"
    "github.com/innovationmech/swit/pkg/saga/coordinator"
    "github.com/innovationmech/swit/pkg/saga/examples"
)

func main() {
    // 创建 Coordinator
    sagaCoordinator := setupCoordinator()
    defer sagaCoordinator.Close()
    
    // 创建服务客户端
    warehouseService := NewWarehouseService()
    inventoryService := NewInventoryService()
    allocationService := NewAllocationService()
    
    // 创建 Saga 定义
    sagaDef := examples.NewInventoryManagementSaga(
        warehouseService,
        inventoryService,
        allocationService,
    )
    
    // 准备库存请求数据
    requestData := &examples.InventoryRequestData{
        OrderID:     "ORD-12345",
        CustomerID:  "CUST-12345",
        RequestType: "reservation",
        Items: []examples.InventoryItem{
            {
                ProductID:   "PROD-001",
                SKU:         "SKU-001",
                Quantity:    10,
                Unit:        "piece",
                Description: "商品A",
            },
            {
                ProductID:   "PROD-002",
                SKU:         "SKU-002",
                Quantity:    5,
                Unit:        "piece",
                Description: "商品B",
            },
        },
        PreferredWarehouses: []string{"WH-BJ-01", "WH-SH-01"},
        AllocationStrategy:  "priority",
        Priority:            5,
    }
    
    // 启动 Saga
    ctx := context.Background()
    instance, err := sagaCoordinator.StartSaga(ctx, sagaDef, requestData)
    if err != nil {
        log.Fatalf("启动 Saga 失败: %v", err)
    }
    
    log.Printf("库存管理 Saga 已启动，实例 ID: %s", instance.GetID())
    
    // 监控执行结果
    time.Sleep(10 * time.Second)
    
    finalInstance, _ := sagaCoordinator.GetSagaInstance(instance.GetID())
    log.Printf("库存管理最终状态: %s", finalInstance.GetStatus())
}
```

### 使用不同的分配策略

```go
// 优先级策略
requestData := &examples.InventoryRequestData{
    // ... 其他字段
    AllocationStrategy: "priority",
}

// 最近仓库策略
requestData := &examples.InventoryRequestData{
    // ... 其他字段
    AllocationStrategy: "nearest",
    Metadata: map[string]interface{}{
        "customer_location": "北京市朝阳区",
    },
}

// 负载均衡策略
requestData := &examples.InventoryRequestData{
    // ... 其他字段
    AllocationStrategy: "load_balance",
}
```

### 处理库存不足

```go
requestData := &examples.InventoryRequestData{
    Items: []examples.InventoryItem{
        {
            SKU:      "SKU-OUT-OF-STOCK",
            Quantity: 1000, // 远超可用库存
        },
    },
}

instance, _ := sagaCoordinator.StartSaga(ctx, sagaDef, requestData)

// 监控结果
time.Sleep(5 * time.Second)
finalInstance, _ := sagaCoordinator.GetSagaInstance(instance.GetID())

if finalInstance.GetStatus() == saga.StatusFailed {
    log.Println("库存不足，无法完成分配")
}
```

## 最佳实践

### 1. 设置合理的过期时间

```go
// 分配和锁定都应设置过期时间
allocationExpiry := 30 * time.Minute
lockExpiry := 30 * time.Minute

// 后台任务定期清理过期的分配和锁定
go func() {
    ticker := time.NewTicker(5 * time.Minute)
    for range ticker.C {
        cleanupExpiredAllocations()
        cleanupExpiredLocks()
    }
}()
```

### 2. 实现幂等性

```go
func (s *LockInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    allocationResult := data.(*AllocationResult)
    
    // 检查是否已经锁定
    existing, err := s.service.GetExistingLock(ctx, allocationResult.AllocationID)
    if err == nil && existing != nil {
        return existing, nil // 返回已存在的锁定结果
    }
    
    // 执行新的锁定
    // ...
}
```

### 3. 优化查询性能

```go
// 使用缓存减少数据库查询
func (s *WarehouseService) QueryWarehouses(ctx context.Context, query *WarehouseQuery) ([]WarehouseInfo, error) {
    // 检查缓存
    cacheKey := generateCacheKey(query)
    if cached := s.cache.Get(cacheKey); cached != nil {
        return cached.([]WarehouseInfo), nil
    }
    
    // 查询数据库
    warehouses, err := s.db.Query(ctx, query)
    if err != nil {
        return nil, err
    }
    
    // 缓存结果（5分钟）
    s.cache.Set(cacheKey, warehouses, 5*time.Minute)
    
    return warehouses, nil
}
```

### 4. 监控关键指标

```go
// 监控指标
metrics := &InventoryMetrics{
    AllocationDuration:   prometheus.NewHistogram(...),
    LockFailureRate:      prometheus.NewCounter(...),
    InventoryUtilization: prometheus.NewGauge(...),
    ExpiredAllocations:   prometheus.NewCounter(...),
}
```

## 测试

### 运行测试

```bash
# 运行库存管理测试
go test -v -run TestInventorySaga

# 测试不同的分配策略
go test -v -run TestInventorySaga/PriorityStrategy
go test -v -run TestInventorySaga/NearestStrategy
go test -v -run TestInventorySaga/LoadBalanceStrategy

# 测试库存不足场景
go test -v -run TestInventorySaga/InsufficientInventory
```

### 测试覆盖场景

1. ✅ 单仓库分配
2. ✅ 多仓库分配
3. ✅ 优先级策略
4. ✅ 最近仓库策略
5. ✅ 负载均衡策略
6. ✅ 库存不足
7. ✅ 锁定冲突
8. ✅ 补偿场景

## 生产环境配置

```yaml
saga:
  inventory_management:
    timeout: 3m
    
    retry:
      max_attempts: 3
      initial_interval: 1s
      max_interval: 30s
    
    compensation:
      timeout: 2m
      retry_attempts: 3
    
    steps:
      query_warehouses:
        timeout: 5s
      check_availability:
        timeout: 10s
      allocate_inventory:
        timeout: 10s
      lock_inventory:
        timeout: 15s
      generate_picklist:
        timeout: 10s
    
    # 过期配置
    expiry:
      allocation: 30m
      lock: 30m
      cleanup_interval: 5m
```

## 常见问题

### Q1: 如何处理跨仓库调拨？

**A**: 实现一个单独的 TransferSaga，协调源仓库和目标仓库的库存操作。

### Q2: 多个订单同时请求同一商品怎么办？

**A**: 使用分布式锁和乐观锁机制，确保并发安全。

### Q3: 如何提高分配效率？

**A**: 
1. 使用缓存减少数据库查询
2. 批量处理分配请求
3. 使用异步处理非关键步骤
4. 优化数据库索引

## 相关资源

- [Saga 用户指南](../../../../docs/saga-user-guide.md)
- [架构设计文档](architecture.md)
- [订单处理 Saga](order_saga.md)

