# Saga PostgreSQL 数据库表结构设计文档

## 概述

本文档描述了 Saga 状态存储的 PostgreSQL 数据库表结构设计。该设计旨在支持完整的 Saga 协调器功能，包括状态持久化、步骤追踪、事件审计和性能优化。

## 设计原则

1. **完整性**: 支持 `StateStorage` 接口所需的所有字段和操作
2. **可扩展性**: 使用 JSONB 类型存储灵活的元数据和复杂数据结构
3. **性能**: 通过战略性索引优化常见查询模式
4. **审计**: 通过事件日志提供完整的审计追踪
5. **一致性**: 使用外键约束确保引用完整性
6. **维护性**: 提供清理和统计功能以支持长期运营

## 数据库表结构

### 1. saga_instances 表

**用途**: 存储 Saga 实例的元数据和整体状态。

#### 字段说明

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| `id` | VARCHAR(255) | PRIMARY KEY | Saga 实例的唯一标识符 (UUID) |
| `definition_id` | VARCHAR(255) | NOT NULL | Saga 定义的标识符 |
| `name` | VARCHAR(255) | NULL | Saga 实例的名称 |
| `description` | TEXT | NULL | Saga 实例的描述 |
| `state` | INTEGER | NOT NULL, CHECK | 当前状态 (0-8) |
| `current_step` | INTEGER | NOT NULL, CHECK | 当前执行的步骤索引 |
| `total_steps` | INTEGER | NOT NULL, CHECK | 总步骤数 |
| `created_at` | TIMESTAMP WITH TIME ZONE | NOT NULL, DEFAULT NOW() | 创建时间 |
| `updated_at` | TIMESTAMP WITH TIME ZONE | NOT NULL, DEFAULT NOW() | 最后更新时间 |
| `started_at` | TIMESTAMP WITH TIME ZONE | NULL | 开始执行时间 |
| `completed_at` | TIMESTAMP WITH TIME ZONE | NULL | 完成时间 |
| `timed_out_at` | TIMESTAMP WITH TIME ZONE | NULL | 超时时间 |
| `initial_data` | JSONB | NULL | 初始输入数据 |
| `current_data` | JSONB | NULL | 当前执行数据 |
| `result_data` | JSONB | NULL | 最终结果数据 |
| `error` | JSONB | NULL | 错误信息 (包含 code, message, type 等) |
| `timeout_ms` | BIGINT | NULL | 超时时长（毫秒） |
| `retry_policy` | JSONB | NULL | 重试策略配置 |
| `metadata` | JSONB | NULL | 自定义元数据 |
| `trace_id` | VARCHAR(255) | NULL | 分布式追踪 ID |
| `span_id` | VARCHAR(255) | NULL | 分布式追踪 Span ID |

#### 状态枚举值

- `0` - Pending (待执行)
- `1` - Running (运行中)
- `2` - StepCompleted (步骤完成)
- `3` - Completed (成功完成)
- `4` - Compensating (补偿中)
- `5` - Compensated (已补偿)
- `6` - Failed (失败)
- `7` - Cancelled (已取消)
- `8` - TimedOut (超时)

#### 索引策略

1. **单列索引**:
   - `idx_saga_state`: 按状态查询
   - `idx_saga_definition_id`: 按定义 ID 查询
   - `idx_saga_created_at`: 按创建时间查询
   - `idx_saga_updated_at`: 按更新时间查询
   - `idx_saga_trace_id`: 按追踪 ID 查询

2. **复合索引**:
   - `idx_saga_state_created`: 按状态和创建时间查询 (支持活动 Saga 查询)
   - `idx_saga_state_updated`: 按状态和更新时间查询 (支持超时检测)
   - `idx_saga_definition_state`: 按定义和状态查询 (支持统计)

3. **GIN 索引**:
   - `idx_saga_metadata`: 支持元数据 JSONB 查询

4. **部分索引** (Partial Indexes):
   - `idx_saga_started_at`: 仅索引已开始的 Saga
   - `idx_saga_completed_at`: 仅索引已完成的 Saga
   - `idx_saga_filter_state_timeout`: 优化活动 Saga 过滤
   - `idx_saga_terminal_cleanup`: 优化清理操作

#### JSONB 字段结构

**error 字段结构**:
```json
{
  "code": "ERROR_CODE",
  "message": "错误描述",
  "type": "error_type",
  "retryable": true,
  "timestamp": "2025-01-15T10:30:00Z",
  "stack_trace": "堆栈信息",
  "details": {},
  "cause": null
}
```

**retry_policy 字段结构**:
```json
{
  "max_attempts": 3,
  "initial_interval_ms": 1000,
  "max_interval_ms": 60000,
  "multiplier": 2.0,
  "retry_on": ["network", "timeout"]
}
```

### 2. saga_steps 表

**用途**: 存储 Saga 中各个步骤的状态和执行信息。

#### 字段说明

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| `id` | VARCHAR(255) | PRIMARY KEY | 步骤实例的唯一标识符 |
| `saga_id` | VARCHAR(255) | NOT NULL, FOREIGN KEY | 所属 Saga 实例 ID |
| `step_index` | INTEGER | NOT NULL, CHECK | 步骤在 Saga 中的索引位置 (0-based) |
| `name` | VARCHAR(255) | NOT NULL | 步骤名称 |
| `state` | INTEGER | NOT NULL, CHECK | 步骤状态 (0-6) |
| `attempts` | INTEGER | NOT NULL, CHECK | 已尝试次数 |
| `max_attempts` | INTEGER | NOT NULL, CHECK | 最大尝试次数 |
| `created_at` | TIMESTAMP WITH TIME ZONE | NOT NULL, DEFAULT NOW() | 创建时间 |
| `started_at` | TIMESTAMP WITH TIME ZONE | NULL | 开始执行时间 |
| `completed_at` | TIMESTAMP WITH TIME ZONE | NULL | 完成时间 |
| `last_attempt_at` | TIMESTAMP WITH TIME ZONE | NULL | 最后一次尝试时间 |
| `input_data` | JSONB | NULL | 输入数据 |
| `output_data` | JSONB | NULL | 输出数据 |
| `error` | JSONB | NULL | 错误信息 |
| `compensation_state` | JSONB | NULL | 补偿状态信息 |
| `metadata` | JSONB | NULL | 自定义元数据 |

#### 步骤状态枚举值

- `0` - Pending (待执行)
- `1` - Running (运行中)
- `2` - Completed (成功完成)
- `3` - Failed (失败)
- `4` - Compensating (补偿中)
- `5` - Compensated (已补偿)
- `6` - Skipped (已跳过)

#### 外键约束

- `fk_saga_steps_saga`: 引用 `saga_instances(id)`, 级联删除 (ON DELETE CASCADE)

#### 索引策略

1. **单列索引**:
   - `idx_step_saga_id`: 按 Saga ID 查询所有步骤
   - `idx_step_state`: 按状态查询步骤

2. **复合索引**:
   - `idx_step_saga_index`: 按 Saga ID 和步骤索引查询 (最常用)
   - `idx_step_saga_state`: 按 Saga ID 和状态查询
   - `idx_step_state_attempts`: 支持重试逻辑查询

3. **唯一索引**:
   - `idx_step_unique_saga_index`: 确保每个 Saga 的每个索引位置只有一个步骤

#### JSONB 字段结构

**compensation_state 字段结构**:
```json
{
  "state": 0,
  "attempts": 1,
  "max_attempts": 3,
  "started_at": "2025-01-15T10:30:00Z",
  "completed_at": null,
  "error": null
}
```

### 3. saga_events 表

**用途**: 记录 Saga 生命周期中的所有事件，提供完整的审计日志。

#### 字段说明

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| `id` | BIGSERIAL | PRIMARY KEY | 事件序列号 (自增) |
| `saga_id` | VARCHAR(255) | NOT NULL, FOREIGN KEY | 所属 Saga 实例 ID |
| `event_type` | VARCHAR(100) | NOT NULL | 事件类型 |
| `step_id` | VARCHAR(255) | NULL | 相关步骤 ID (如适用) |
| `step_index` | INTEGER | NULL | 相关步骤索引 |
| `timestamp` | TIMESTAMP WITH TIME ZONE | NOT NULL, DEFAULT NOW() | 事件发生时间 |
| `event_data` | JSONB | NULL | 事件详细数据 |
| `old_state` | INTEGER | NULL | 状态转换前的状态 |
| `new_state` | INTEGER | NULL | 状态转换后的状态 |
| `error` | JSONB | NULL | 错误信息 (如适用) |
| `trace_id` | VARCHAR(255) | NULL | 分布式追踪 ID |
| `span_id` | VARCHAR(255) | NULL | 分布式追踪 Span ID |
| `metadata` | JSONB | NULL | 自定义元数据 |

#### 事件类型

**Saga 生命周期事件**:
- `saga.started` - Saga 开始执行
- `saga.step.started` - 步骤开始执行
- `saga.step.completed` - 步骤完成
- `saga.step.failed` - 步骤失败
- `saga.completed` - Saga 成功完成
- `saga.failed` - Saga 失败
- `saga.cancelled` - Saga 取消
- `saga.timed_out` - Saga 超时

**补偿事件**:
- `compensation.started` - 补偿开始
- `compensation.step.started` - 补偿步骤开始
- `compensation.step.completed` - 补偿步骤完成
- `compensation.step.failed` - 补偿步骤失败
- `compensation.completed` - 补偿完成
- `compensation.failed` - 补偿失败

**重试事件**:
- `retry.attempted` - 重试尝试
- `retry.exhausted` - 重试次数耗尽

**状态变更事件**:
- `state.changed` - 状态变更

#### 外键约束

- `fk_saga_events_saga`: 引用 `saga_instances(id)`, 级联删除 (ON DELETE CASCADE)

#### 索引策略

1. **单列索引**:
   - `idx_event_saga_id`: 按 Saga ID 查询所有事件
   - `idx_event_type`: 按事件类型查询
   - `idx_event_timestamp`: 按时间戳降序查询 (最新优先)

2. **复合索引**:
   - `idx_event_saga_timestamp`: 查询特定 Saga 的事件历史
   - `idx_event_saga_type`: 查询特定 Saga 的特定类型事件
   - `idx_event_type_timestamp`: 全局事件类型统计

3. **部分索引**:
   - `idx_event_step_id`: 仅索引与步骤相关的事件
   - `idx_event_trace_id`: 仅索引包含追踪 ID 的事件

## 辅助功能

### 视图 (Views)

#### 1. active_sagas
提供对所有活动 Saga 实例的快速访问。

```sql
SELECT id, definition_id, name, state, current_step, total_steps,
       created_at, updated_at, started_at, trace_id, metadata
FROM saga_instances
WHERE state IN (1, 2, 4)  -- running, step_completed, compensating
ORDER BY created_at DESC;
```

#### 2. failed_sagas
提供对所有失败 Saga 实例的快速访问。

```sql
SELECT id, definition_id, name, state, current_step, total_steps,
       created_at, updated_at, completed_at, error, trace_id
FROM saga_instances
WHERE state IN (6, 7, 8)  -- failed, cancelled, timed_out
ORDER BY updated_at DESC;
```

#### 3. completed_sagas
提供对所有成功完成的 Saga 实例的快速访问。

```sql
SELECT id, definition_id, name, current_step, total_steps,
       created_at, started_at, completed_at, result_data, trace_id
FROM saga_instances
WHERE state = 3  -- completed
ORDER BY completed_at DESC;
```

#### 4. saga_statistics
提供按 Saga 定义分组的聚合统计信息。

```sql
SELECT 
    definition_id,
    COUNT(*) as total_instances,
    COUNT(CASE WHEN state = 3 THEN 1 END) as completed_count,
    COUNT(CASE WHEN state IN (6, 7, 8) THEN 1 END) as failed_count,
    COUNT(CASE WHEN state IN (1, 2, 4) THEN 1 END) as active_count,
    AVG(CASE 
        WHEN completed_at IS NOT NULL AND started_at IS NOT NULL 
        THEN EXTRACT(EPOCH FROM (completed_at - started_at))
    END) as avg_duration_seconds,
    MAX(updated_at) as last_execution
FROM saga_instances
GROUP BY definition_id;
```

### 函数 (Functions)

#### 1. cleanup_expired_sagas(older_than)
删除指定时间之前的已终止 Saga 实例。

**参数**:
- `older_than`: TIMESTAMP WITH TIME ZONE - 删除此时间之前更新的 Saga

**返回**: INTEGER - 删除的记录数

**说明**: 仅删除终止状态的 Saga (completed, compensated, failed, cancelled, timed_out)，级联删除相关的步骤和事件。

#### 2. update_saga_updated_at()
触发器函数，自动更新 `saga_instances` 表的 `updated_at` 字段。

### 触发器 (Triggers)

#### trigger_saga_updated_at
在 `saga_instances` 表的每次更新前自动更新 `updated_at` 字段。

## 查询模式优化

### 常见查询及其索引使用

#### 1. 查询活动 Saga
```sql
SELECT * FROM saga_instances 
WHERE state IN (1, 2, 4) 
ORDER BY created_at;
```
**使用索引**: `idx_saga_state_created`

#### 2. 查询超时 Saga
```sql
SELECT * FROM saga_instances 
WHERE state IN (1, 2, 4) 
  AND created_at < NOW() - INTERVAL '1 hour';
```
**使用索引**: `idx_saga_filter_state_timeout`

#### 3. 查询特定 Saga 的所有步骤
```sql
SELECT * FROM saga_steps 
WHERE saga_id = $1 
ORDER BY step_index;
```
**使用索引**: `idx_step_saga_index`

#### 4. 查询特定 Saga 的事件历史
```sql
SELECT * FROM saga_events 
WHERE saga_id = $1 
ORDER BY timestamp DESC;
```
**使用索引**: `idx_event_saga_timestamp`

#### 5. 查询需要重试的步骤
```sql
SELECT * FROM saga_steps 
WHERE state = 3  -- failed
  AND attempts < max_attempts;
```
**使用索引**: `idx_step_state_attempts`

#### 6. 按追踪 ID 查询
```sql
SELECT * FROM saga_instances WHERE trace_id = $1;
SELECT * FROM saga_events WHERE trace_id = $1 ORDER BY timestamp;
```
**使用索引**: `idx_saga_trace_id`, `idx_event_trace_id`

## 性能考虑

### 写入性能

1. **批量插入**: 使用事务批量插入步骤和事件以减少往返次数
2. **JSONB 索引**: GIN 索引会增加写入开销，但显著提升 JSONB 查询性能
3. **部分索引**: 使用 WHERE 子句的部分索引减少索引大小和维护开销

### 查询性能

1. **复合索引**: 覆盖常见的多列查询条件
2. **视图**: 预定义的视图简化常见查询并可能被查询优化器优化
3. **统计信息**: 定期运行 `ANALYZE` 以保持查询计划最优

### 存储优化

1. **JSONB 压缩**: PostgreSQL 自动压缩 JSONB 数据
2. **定期清理**: 使用 `cleanup_expired_sagas()` 函数定期删除旧数据
3. **分区**: 对于高流量环境，考虑按时间分区 `saga_events` 表

## 扩展性建议

### 1. 高并发场景

- 使用连接池 (如 PgBouncer)
- 配置适当的 `max_connections` 和 `shared_buffers`
- 考虑使用只读副本分离读写负载

### 2. 大数据量场景

- 实施 `saga_events` 表分区 (按月或按周)
- 配置自动归档策略
- 使用表分区提升查询和维护性能

### 3. 高可用场景

- 配置 PostgreSQL 流复制
- 使用自动故障转移工具 (如 Patroni, Stolon)
- 实施定期备份和恢复测试

## 迁移策略

### 初始部署

1. 在目标数据库中执行 `saga_schema.sql`
2. 验证所有表、索引和约束创建成功
3. 插入初始版本记录到 `saga_schema_version`

### 版本升级

1. 创建新的迁移脚本 (如 `saga_schema_v2.sql`)
2. 在迁移脚本中检查当前版本
3. 执行必要的 ALTER TABLE 语句
4. 更新 `saga_schema_version` 表

## 安全性考虑

### 访问控制

1. **应用角色**: 创建专门的数据库角色用于应用访问
   ```sql
   CREATE ROLE saga_app_role;
   GRANT SELECT, INSERT, UPDATE, DELETE ON saga_instances TO saga_app_role;
   GRANT SELECT, INSERT, UPDATE, DELETE ON saga_steps TO saga_app_role;
   GRANT SELECT, INSERT ON saga_events TO saga_app_role;
   ```

2. **只读角色**: 用于监控和报表
   ```sql
   CREATE ROLE saga_readonly_role;
   GRANT SELECT ON ALL TABLES IN SCHEMA public TO saga_readonly_role;
   ```

### 数据加密

1. **传输加密**: 使用 SSL/TLS 连接数据库
2. **静态加密**: 启用 PostgreSQL 透明数据加密 (TDE)
3. **敏感数据**: 考虑在应用层加密敏感的 JSONB 字段

## 监控和维护

### 关键指标

1. **表大小监控**:
   ```sql
   SELECT 
       schemaname,
       tablename,
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
   FROM pg_tables
   WHERE tablename LIKE 'saga_%'
   ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
   ```

2. **索引使用情况**:
   ```sql
   SELECT 
       schemaname,
       tablename,
       indexname,
       idx_scan,
       idx_tup_read,
       idx_tup_fetch
   FROM pg_stat_user_indexes
   WHERE tablename LIKE 'saga_%'
   ORDER BY idx_scan;
   ```

3. **慢查询监控**: 启用 `pg_stat_statements` 扩展

### 定期维护任务

1. **VACUUM**: 定期清理死元组
   ```sql
   VACUUM ANALYZE saga_instances;
   VACUUM ANALYZE saga_steps;
   VACUUM ANALYZE saga_events;
   ```

2. **重建索引**: 对于频繁更新的索引
   ```sql
   REINDEX TABLE CONCURRENTLY saga_instances;
   ```

3. **清理旧数据**: 使用 `cleanup_expired_sagas()` 函数

## 测试建议

### 单元测试

1. 测试所有约束 (CHECK, FOREIGN KEY, UNIQUE)
2. 测试触发器功能
3. 测试清理函数

### 性能测试

1. 使用 `pgbench` 进行基准测试
2. 测试并发插入和查询性能
3. 测试大数据量下的查询性能

### 集成测试

1. 测试完整的 Saga 执行流程
2. 测试补偿流程
3. 测试超时和重试场景

## 总结

本数据库 schema 设计提供了：

- ✅ 完整的 Saga 状态存储支持
- ✅ 全面的审计和追踪能力
- ✅ 优化的查询性能
- ✅ 灵活的扩展性
- ✅ 强大的数据完整性保证
- ✅ 便捷的维护和监控工具

该设计经过精心规划，能够满足从小型应用到大规模分布式系统的各种 Saga 编排需求。

## 参考资料

- [PostgreSQL JSONB 文档](https://www.postgresql.org/docs/current/datatype-json.html)
- [PostgreSQL 索引最佳实践](https://www.postgresql.org/docs/current/indexes.html)
- [Saga Pattern 设计模式](https://microservices.io/patterns/data/saga.html)
- Swit 项目 StateStorage 接口: `pkg/saga/interfaces.go`

