# Saga Pattern（轻量内存版）

本包提供 Saga 模式的轻量级、可嵌入实现，用于在单个进程内按顺序编排一组本地步骤，并在失败时按相反顺序执行补偿。

## 与 `pkg/saga` 的定位差异

仓库中存在两个 Saga 相关的包，定位完全不同，请按需选择：

| 维度 | `pkg/patterns/saga`（本包） | `pkg/saga` |
|------|------------------------------|------------|
| 定位 | 轻量模式骨架，开箱即用 | 生产级分布式 Saga 框架 |
| 执行模型 | 同步、顺序执行，失败时反向补偿 | Orchestration / Choreography / Hybrid 多协调模式 |
| 状态持久化 | 无（纯内存） | Memory / PostgreSQL / Redis 多后端 |
| 消息集成 | 无 | 与 `pkg/messaging` 深度集成（`pkg/saga/messaging`） |
| 可观测性 | 无 | 内置监控、指标、健康检查（`pkg/saga/monitoring`） |
| 高级能力 | 无 | DSL、重试策略、安全审计、数据库迁移工具 |
| 适用场景 | 单进程内的简单编排、原型验证、测试 | 跨服务的分布式事务、需要持久化与恢复的生产场景 |

选择建议：

- **使用本包**：步骤都在同一进程内、不需要崩溃恢复、希望零依赖快速落地补偿逻辑（例如一次 API 请求内的多步本地操作回滚）。
- **使用 `pkg/saga`**：跨服务协调、需要将 Saga 状态落盘（PostgreSQL/Redis）、需要事件驱动协调或生产级监控与重试。

## 快速开始

```go
import "github.com/innovationmech/swit/pkg/patterns/saga"

coordinator := saga.NewInMemoryCoordinator()

def := &saga.SagaDefinition{
    Name: "create-order",
    Steps: []saga.SagaStep{
        {
            Name:       "reserve-inventory",
            Execute:    reserveInventory,
            Compensate: releaseInventory,
        },
        {
            Name:       "charge-payment",
            Execute:    chargePayment,
            Compensate: refundPayment,
        },
    },
}

execution, err := coordinator.StartSaga(ctx, def, orderData)
```

`StartSaga` 同步按顺序执行所有步骤；任一步骤失败时，已完成的步骤会按相反顺序自动补偿。

## 相关模式

- 事务性发件箱：[`pkg/patterns/outbox`](../outbox/README.md)
- 幂等收件箱：[`pkg/patterns/inbox`](../inbox/README.md)
- 分布式 Saga 框架：[`pkg/saga`](../../saga/)

## 许可证

MIT License - 查看 [LICENSE](../../../LICENSE) 文件了解详情
