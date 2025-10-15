# Saga 数据库迁移指南

## 概述

本文档介绍如何执行 Saga 状态存储的数据库迁移，包括迁移的应用、回滚和状态检查。

## 迁移系统架构

### 组件

1. **迁移脚本** (`scripts/sql/saga_migrations.sql`)
   - 定义所有迁移版本的 SQL 函数
   - 包含迁移管理基础设施
   - 提供版本跟踪和回滚功能

2. **Go 迁移工具** (`pkg/saga/migrations/migrator.go`)
   - 提供编程式迁移接口
   - 自动初始化和应用迁移
   - 集成到 PostgreSQL 存储实现中

3. **迁移表** (`saga_migrations`)
   - 跟踪所有已执行的迁移
   - 记录执行时间和状态
   - 存储回滚信息

### 迁移版本

- **V1**: 初始 Schema
  - 创建 `saga_instances` 表
  - 创建 `saga_steps` 表
  - 创建 `saga_events` 表
  - 创建视图、函数和触发器

- **V2**: 添加乐观锁
  - 在 `saga_instances` 表添加 `version` 列
  - 创建版本自动递增触发器
  - 支持并发控制

## 使用方法

### 1. SQL 直接执行（手动迁移）

#### 初始化迁移系统

```sql
-- 连接到数据库
psql -U postgres -d saga_db

-- 执行迁移脚本
\i scripts/sql/saga_migrations.sql
```

执行后会看到：

```
========================================
Saga Migration System Initialized
========================================
Current Schema Version: 0

Available Migrations:
  V1: Initial Saga Schema
  V2: Add Optimistic Locking

To apply migrations:
  SELECT apply_all_migrations();

To check status:
  SELECT * FROM get_migration_status();

To rollback a migration:
  SELECT rollback_migration_v2();
  SELECT rollback_migration_v1();
========================================
```

#### 应用所有迁移

```sql
-- 应用所有待处理的迁移
SELECT apply_all_migrations();
```

#### 应用特定版本迁移

```sql
-- 应用 V1 迁移
SELECT apply_migration_v1();

-- 应用 V2 迁移
SELECT apply_migration_v2();
```

#### 检查迁移状态

```sql
-- 查看当前 schema 版本
SELECT get_current_schema_version();

-- 查看所有迁移的详细状态
SELECT * FROM get_migration_status();

-- 检查特定迁移是否已应用
SELECT is_migration_applied(1);  -- 检查 V1
SELECT is_migration_applied(2);  -- 检查 V2
```

#### 回滚迁移

```sql
-- 回滚 V2 迁移
SELECT rollback_migration_v2();

-- 回滚 V1 迁移（警告：会删除所有 Saga 数据！）
SELECT rollback_migration_v1();
```

### 2. Go 程序化执行（推荐）

#### 自动迁移

在应用程序启动时自动执行迁移：

```go
import (
    "context"
    "database/sql"
    "github.com/innovationmech/swit/pkg/saga/migrations"
    _ "github.com/lib/pq"
)

func main() {
    // 连接数据库
    db, err := sql.Open("postgres", "postgres://user:pass@localhost/saga_db?sslmode=disable")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    // 创建迁移器
    migrator := migrations.NewMigrator(db)
    ctx := context.Background()
    
    // 执行迁移
    if err := migrator.Migrate(ctx); err != nil {
        panic(fmt.Sprintf("Migration failed: %v", err))
    }
    
    fmt.Println("Database migrations completed successfully")
}
```

#### 检查迁移状态

```go
// 获取当前版本
version, err := migrator.GetCurrentVersion(ctx)
if err != nil {
    log.Fatalf("Failed to get version: %v", err)
}
fmt.Printf("Current schema version: %d\n", version)

// 获取所有迁移状态
statuses, err := migrator.GetMigrationStatus(ctx)
if err != nil {
    log.Fatalf("Failed to get status: %v", err)
}

for _, status := range statuses {
    fmt.Printf("V%d: %s - %s (applied at: %s, time: %dms)\n",
        status.Version,
        status.Name,
        status.Status,
        status.AppliedAt.Format(time.RFC3339),
        status.ExecutionTimeMs,
    )
}
```

#### 验证迁移版本

```go
// 验证数据库是否满足所需的最低版本
requiredVersion := 2
if err := migrator.ValidateMigrations(ctx, requiredVersion); err != nil {
    log.Fatalf("Schema version validation failed: %v", err)
}
```

#### 手动应用特定迁移

```go
// 应用 V1 迁移
result, err := migrator.ApplyMigration(ctx, 1)
if err != nil {
    log.Fatalf("Failed to apply V1: %v", err)
}
fmt.Println(result)

// 应用 V2 迁移
result, err = migrator.ApplyMigration(ctx, 2)
if err != nil {
    log.Fatalf("Failed to apply V2: %v", err)
}
fmt.Println(result)
```

#### 回滚迁移

```go
// 回滚 V2 迁移
result, err := migrator.RollbackMigration(ctx, 2)
if err != nil {
    log.Fatalf("Failed to rollback V2: %v", err)
}
fmt.Println(result)
```

### 3. 集成到 PostgreSQL 存储

PostgreSQL 存储在初始化时可以自动运行迁移：

```go
import "github.com/innovationmech/swit/pkg/saga/state/storage"

// 创建配置并启用自动迁移
config := &storage.PostgresConfig{
    DSN:         "postgres://user:pass@localhost/saga_db?sslmode=disable",
    AutoMigrate: true,  // 启用自动迁移
    // ... 其他配置
}

// 创建存储（会自动执行迁移）
store, err := storage.NewPostgresStateStorage(config)
if err != nil {
    panic(err)
}
defer store.Close()
```

## 迁移详细信息

### V1: 初始 Schema

**创建的对象：**

表：
- `saga_instances` - Saga 实例主表
- `saga_steps` - Saga 步骤状态表
- `saga_events` - Saga 事件日志表

索引：
- 状态索引、时间索引、复合索引
- GIN 索引用于 JSONB 查询
- 部分索引用于优化特定查询

视图：
- `active_sagas` - 活跃的 Saga 实例
- `failed_sagas` - 失败的 Saga 实例
- `completed_sagas` - 完成的 Saga 实例
- `saga_statistics` - Saga 统计信息

函数：
- `cleanup_expired_sagas()` - 清理过期 Saga
- `update_saga_updated_at()` - 自动更新时间戳

触发器：
- `trigger_saga_updated_at` - 更新 `updated_at` 字段

**执行时间：** 通常 < 500ms

**可回滚：** 是（警告：会删除所有数据）

### V2: 添加乐观锁

**修改的对象：**

表修改：
- 在 `saga_instances` 添加 `version` 列
- 添加 `chk_version` 约束

函数：
- `increment_saga_version()` - 自动递增版本号

触发器：
- `trigger_saga_version_increment` - 更新时递增版本

**执行时间：** 通常 < 100ms

**可回滚：** 是

## 最佳实践

### 1. 备份

在执行迁移前，始终备份数据库：

```bash
# 备份整个数据库
pg_dump -U postgres -d saga_db > saga_db_backup_$(date +%Y%m%d_%H%M%S).sql

# 仅备份 Saga 相关表
pg_dump -U postgres -d saga_db -t saga_instances -t saga_steps -t saga_events > saga_backup.sql
```

### 2. 测试环境验证

在生产环境执行前，先在测试环境验证：

```bash
# 在测试数据库执行
psql -U postgres -d saga_test -f scripts/sql/saga_migrations.sql
```

### 3. 监控执行时间

使用 PostgreSQL 的查询日志监控迁移执行：

```sql
-- 启用查询日志
SET log_duration = on;
SET log_statement = 'all';

-- 执行迁移
SELECT apply_all_migrations();
```

### 4. 检查迁移结果

迁移完成后验证：

```sql
-- 1. 检查所有表是否存在
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' 
  AND table_name LIKE 'saga_%';

-- 2. 检查迁移记录
SELECT * FROM saga_migrations ORDER BY version;

-- 3. 验证数据完整性
SELECT COUNT(*) FROM saga_instances;
SELECT COUNT(*) FROM saga_steps;
SELECT COUNT(*) FROM saga_events;

-- 4. 检查索引
SELECT indexname, tablename 
FROM pg_indexes 
WHERE tablename LIKE 'saga_%';
```

### 5. 生产环境部署流程

推荐的生产环境迁移流程：

1. **准备阶段**
   ```bash
   # 1. 备份数据库
   pg_dump -U postgres -d saga_prod > backup.sql
   
   # 2. 在测试环境验证迁移
   psql -U postgres -d saga_test -f scripts/sql/saga_migrations.sql
   psql -U postgres -d saga_test -c "SELECT apply_all_migrations();"
   ```

2. **维护窗口**
   ```sql
   -- 1. 禁止新的 Saga 创建（应用层面）
   
   -- 2. 等待活跃 Saga 完成
   SELECT COUNT(*) FROM saga_instances WHERE state IN (1, 2, 4);
   
   -- 3. 执行迁移
   \i scripts/sql/saga_migrations.sql
   SELECT apply_all_migrations();
   
   -- 4. 验证结果
   SELECT * FROM get_migration_status();
   ```

3. **验证阶段**
   ```sql
   -- 检查所有表和索引
   SELECT * FROM get_migration_status();
   
   -- 运行健康检查
   SELECT get_current_schema_version();
   ```

4. **恢复服务**
   ```bash
   # 启动应用服务
   # 监控日志确认无错误
   ```

## 故障排除

### 迁移失败

如果迁移失败：

```sql
-- 1. 检查错误日志
SELECT * FROM saga_migrations WHERE status = 'failed';

-- 2. 查看 PostgreSQL 日志
-- 查看 /var/log/postgresql/postgresql-*.log

-- 3. 手动修复后重试
-- 根据错误信息修复问题
-- 清除失败的迁移记录
DELETE FROM saga_migrations WHERE version = X AND status = 'failed';

-- 4. 重新执行迁移
SELECT apply_migration_vX();
```

### 回滚失败

如果回滚失败：

```sql
-- 1. 检查依赖关系
-- 确保没有其他对象依赖被回滚的对象

-- 2. 强制清理
DROP TABLE IF EXISTS saga_instances CASCADE;
DROP TABLE IF EXISTS saga_steps CASCADE;
DROP TABLE IF EXISTS saga_events CASCADE;

-- 3. 重新执行迁移
SELECT apply_migration_v1();
```

### 版本不一致

如果迁移版本与代码不匹配：

```go
// 在应用启动时验证
migrator := migrations.NewMigrator(db)
requiredVersion := 2

if err := migrator.ValidateMigrations(ctx, requiredVersion); err != nil {
    log.Fatalf("Schema version mismatch: %v", err)
}
```

## 性能考虑

### 大数据量迁移

对于包含大量数据的表，迁移可能需要较长时间：

```sql
-- 查看表大小
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE tablename LIKE 'saga_%'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- 对于大表，考虑分批处理或在非高峰时段执行
```

### 锁定影响

迁移期间可能会锁定表：

```sql
-- 查看当前锁定
SELECT * FROM pg_locks WHERE relation::regclass::text LIKE 'saga_%';

-- 使用 CONCURRENTLY 创建索引（如果适用）
CREATE INDEX CONCURRENTLY idx_name ON table_name(column_name);
```

## 参考

- [PostgreSQL 迁移文档](https://www.postgresql.org/docs/current/ddl-schemas.html)
- [Saga 架构设计](../../docs/architecture/saga-architecture.md)
- [PostgreSQL 存储实现](../../pkg/saga/state/storage/postgres.go)
- [迁移测试](../../pkg/saga/migrations/migrator_test.go)

## 支持

如有问题或需要帮助，请：

1. 查看 [GitHub Issues](https://github.com/innovationmech/swit/issues)
2. 参考 [开发者指南](../../docs/developer-guide.md)
3. 联系维护团队

