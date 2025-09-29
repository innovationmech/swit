# 迁移与升级指南

本指南帮助你在不破坏现有服务稳定性的前提下，完成框架与配置的迁移与升级，并充分利用包括错误监控（Sentry）、CLI 工具（switctl）、消息适配器切换在内的新能力。

## 适用范围

- 配置结构演进与键名变更的升级迁移
- 使用 `switctl` 对配置进行校验、生成与自动迁移
- 在不同消息中间件适配器之间安全切换（如 RabbitMQ ↔ Kafka ↔ NATS）

---

## 配置迁移与升级

### 1. 基本命令

使用 `switctl` 内置迁移与校验能力：

```bash
# 在仓库根目录或你的服务项目目录下

# 严格校验配置（报错即失败）
switctl config validate --strict

# 将配置从 v1 迁移到 v2，并生成备份
switctl config migrate --from v1 --to v2 --backup

# 指定规则文件（可选，JSON/YAML 均可）
switctl config migrate --from v1 --to v2 --rules ./migration-rules.yaml
```

校验通过后，再进行迁移操作可显著降低风险。`migrate` 支持：

- 键名映射与删除（如 `a.b` → `x.y` 或移除废弃键）
- 默认值补齐（`default`）
- 版本标记写入（`version: v2`）

### 2. 环境变量映射与优先级

- 前缀：`SWIT_`
- 映射：配置键 `a.b.c` → 环境变量 `SWIT_A_B_C`
- 优先级（低 → 高）：Defaults、Base 文件、Env 文件、Override 文件、环境变量

参考：`/zh/guide/configuration-reference`（自动生成）

### 3. 变更示例

```yaml
# 变更前
service_name: "my-service"
http:
  port: "8080"

# 变更后（新增 Sentry 片段，仅示例）
service_name: "my-service"
http:
  port: "8080"
sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  sample_rate: 1.0
  traces_sample_rate: 0.1
```

---

## 消息适配器迁移（Kafka / RabbitMQ / NATS）

在切换消息适配器之前，使用代码中的迁移规划工具生成“能力差异”和“迁移清单”。

### 1. 生成迁移计划

```go
// 使用 messaging.PlanBrokerSwitch 生成迁移计划
ctx := context.Background()
plan, err := messaging.PlanBrokerSwitch(ctx, currentConfig, targetConfig)
if err != nil {
    return fmt.Errorf("plan switch: %w", err)
}

fmt.Printf("Switching %s -> %s (score %d)\n", plan.CurrentType, plan.TargetType, plan.CompatibilityScore)
for _, item := range plan.Checklist {
    fmt.Printf("✔ %s\n", item)
}
```

该计划会给出：

- 能力差异（FeatureDeltas）：如事务、顺序性、DLQ、延迟投递、优先级
- 总体兼容度与迁移难度评估
- 迁移步骤、注意事项与风险
- 是否建议“**双写**”与“**影子消费**”

### 2. 配置驱动切换

```yaml
current: rabbitmq
migration_target: kafka
brokers:
  rabbitmq:
    type: rabbitmq
    endpoints: ["amqp://guest:guest@localhost:5672/"]
  kafka:
    type: kafka
    endpoints: ["localhost:9092"]
```

部署时将 `current`（或通过环境变量 `SWIT_TARGET_BROKER`）指向目标适配器。参考示例：`examples/messaging/adapter-switch/`。

### 3. 常见能力差异的替代策略

- `transactions`：采用“事务外盒 + 幂等消费者”模式
- `ordering`：基于 key 分片或接受乱序并实现幂等
- `dead_letter`：为失败消息设置驻车队列并编写 DLQ 处理器
- `delayed_delivery`：缺失原生延迟时通过定时器/重试调度实现
- `priority`：按优先级拆分队列，并独立扩缩容

---

## 分步清单（建议）

### 预检查
- [ ] 备份现有配置文件
- [ ] `switctl config validate --strict` 通过
- [ ] 明确回滚策略（配置回滚或开关禁用）

### 配置迁移
- [ ] 执行 `switctl config migrate --from vX --to vY --backup`
- [ ] 手动审阅迁移结果与备份差异
- [ ] 在 CI 中增加 `switctl config validate` 检查

### 适配器切换
- [ ] 生成迁移计划：`messaging.PlanBrokerSwitch`
- [ ] 若兼容度中/低或存在关键能力缺失：启用双写与影子消费
- [ ] 在预发环境验证：吞吐、延迟、错误率、消息一致性

### 上线与收敛
- [ ] 分批切流，监控关键指标
- [ ] 指标达标后关闭双写，清理影子消费者

---

## 回滚与故障处理

### 快速回滚配置

```yaml
# 关闭新增能力（示例）
sentry:
  enabled: false
```

或使用迁移备份文件还原。

### 常见问题

1. 迁移后服务启动失败：先执行 `switctl config validate --strict` 定位具体键值错误
2. 切换后消息量异常：检查 Plan 中的能力差异，确认是否需要双写与补偿逻辑
3. 观测/报警异常：核对新老配置的采样率、过滤规则与指标对齐

---

## 参考

- `/zh/guide/configuration`、`/zh/guide/configuration-reference`
- `examples/messaging/adapter-switch/`
- `pkg/messaging/` 兼容性与迁移相关实现


