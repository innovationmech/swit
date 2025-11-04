# PostgreSQL Integration Test Setup

本文档说明如何设置和运行 PostgreSQL Saga 状态存储的集成测试。

## 环境要求

- Go 1.23+
- PostgreSQL 13+ （推荐使用 Docker）
- 操作系统：Linux、macOS 或 Windows

## 快速开始

### 方法 1：使用 Docker（推荐）

1. 启动 PostgreSQL 容器：

```bash
docker run --name swit-test-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=swit_test \
  -p 5432:5432 \
  -d postgres:15-alpine
```

2. 设置测试 DSN 环境变量：

```bash
export POSTGRES_TEST_DSN="host=localhost port=5432 user=postgres password=postgres dbname=swit_test sslmode=disable"
```

3. 运行集成测试：

```bash
# 运行所有 PostgreSQL 集成测试
go test -v ./pkg/saga/state/storage -run TestPostgresIntegration

# 运行特定测试
go test -v ./pkg/saga/state/storage -run TestPostgresIntegration_FullLifecycle

# 运行性能基准测试
go test -v ./pkg/saga/state/storage -bench BenchmarkPostgresIntegration -run=^$
```

### 方法 2：使用 Docker Compose

创建 `docker-compose.test.yml`：

```yaml
version: '3.8'

services:
  postgres-test:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: swit_test
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5
```

运行：

```bash
# 启动数据库
docker-compose -f docker-compose.test.yml up -d

# 等待数据库就绪
sleep 5

# 设置环境变量并运行测试
export POSTGRES_TEST_DSN="host=localhost port=5432 user=postgres password=postgres dbname=swit_test sslmode=disable"
go test -v ./pkg/saga/state/storage -run TestPostgresIntegration

# 停止并清理
docker-compose -f docker-compose.test.yml down -v
```

### 方法 3：使用本地 PostgreSQL

如果您已经安装了本地 PostgreSQL：

1. 创建测试数据库：

```sql
CREATE DATABASE swit_test;
```

2. 设置 DSN 并运行测试：

```bash
export POSTGRES_TEST_DSN="host=localhost port=5432 user=your_user password=your_password dbname=swit_test sslmode=disable"
go test -v ./pkg/saga/state/storage -run TestPostgresIntegration
```

## 测试覆盖范围

集成测试包含以下场景：

### 基础 CRUD 操作
- ✅ 完整生命周期测试（创建、读取、更新、删除）
- ✅ 多个活跃 Saga 管理
- ✅ 步骤状态保存和检索
- ✅ 级联删除验证

### 事务支持
- ✅ 事务提交测试
- ✅ 事务回滚测试
- ✅ 批量保存操作
- ✅ 并发事务处理

### 并发和性能
- ✅ 并发写入测试（20 goroutines × 10 操作）
- ✅ 并发事务测试（10 并发事务）
- ✅ 保存操作基准测试
- ✅ 读取操作基准测试
- ✅ 事务操作基准测试

### 乐观锁
- ✅ 版本控制验证
- ✅ 并发更新冲突检测
- ✅ 乐观锁失败错误处理

### 超时和清理
- ✅ Saga 超时检测
- ✅ 过期 Saga 清理

### 健康检查和监控
- ✅ 数据库健康检查
- ✅ 连接池统计
- ✅ 连接泄漏检测

### 数据一致性
- ✅ 复杂元数据序列化
- ✅ JSONB 字段处理
- ✅ 数据完整性验证

### 数据库迁移
- ✅ Schema 自动创建
- ✅ 表结构验证

### 错误处理
- ✅ 连接失败处理
- ✅ 查询超时处理

## 测试配置

测试使用以下 PostgreSQL 配置：

```go
&PostgresConfig{
    DSN:               "...",  // 从环境变量读取
    MaxOpenConns:      10,     // 最大打开连接数
    MaxIdleConns:      5,      // 最大空闲连接数
    ConnMaxLifetime:   30m,    // 连接最大生命周期
    ConnMaxIdleTime:   10m,    // 连接最大空闲时间
    ConnectionTimeout: 5s,     // 连接超时
    QueryTimeout:      10s,    // 查询超时
    AutoMigrate:       true,   // 自动创建表
    TablePrefix:       "test_",
    MaxRetries:        3,      // 最大重试次数
    RetryBackoff:      100ms,  // 重试退避时间
    MaxRetryBackoff:   5s,     // 最大重试退避时间
}
```

## CI/CD 集成

### GitHub Actions 示例

```yaml
name: PostgreSQL Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: swit_test
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      
      - name: Run PostgreSQL Integration Tests
        env:
          POSTGRES_TEST_DSN: "host=localhost port=5432 user=postgres password=postgres dbname=swit_test sslmode=disable"
        run: |
          go test -v ./pkg/saga/state/storage -run TestPostgresIntegration
          go test -v ./pkg/saga/state/storage -bench BenchmarkPostgresIntegration -run=^$
```

## 清理测试数据

测试会在每次运行后自动清理测试数据。如需手动清理：

```sql
-- 连接到测试数据库
\c swit_test

-- 删除所有测试表数据
DELETE FROM test_saga_events;
DELETE FROM test_saga_steps;
DELETE FROM test_saga_instances;

-- 或完全删除测试数据库
DROP DATABASE swit_test;
CREATE DATABASE swit_test;
```

## 故障排除

### 问题：测试被跳过

```
SKIP: Skipping PostgreSQL integration tests. Set POSTGRES_TEST_DSN to enable.
```

**解决方案**：设置 `POSTGRES_TEST_DSN` 环境变量。

### 问题：连接失败

```
Failed to create storage: failed to ping postgres database
```

**解决方案**：
1. 检查 PostgreSQL 是否正在运行
2. 验证 DSN 连接参数
3. 确保防火墙允许连接
4. 检查数据库是否已创建

### 问题：权限错误

```
permission denied for table saga_instances
```

**解决方案**：确保数据库用户有创建表和写入数据的权限。

### 问题：端口冲突

```
port is already allocated
```

**解决方案**：
1. 停止占用端口的进程：`lsof -i :5432`
2. 或使用不同的端口并更新 DSN

## 性能基准测试

运行性能基准测试并查看结果：

```bash
# 运行基准测试
go test -v ./pkg/saga/state/storage \
  -bench BenchmarkPostgresIntegration \
  -benchmem \
  -run=^$ \
  -benchtime=10s

# 生成 CPU profile
go test -v ./pkg/saga/state/storage \
  -bench BenchmarkPostgresIntegration_SaveSaga \
  -cpuprofile=cpu.prof \
  -run=^$

# 分析 profile
go tool pprof cpu.prof
```

预期性能指标（在现代硬件上）：
- SaveSaga: ~500-2000 ops/sec
- GetSaga: ~1000-5000 ops/sec
- Transaction: ~300-1000 ops/sec

## 最佳实践

1. **隔离测试环境**：始终使用专用的测试数据库
2. **并行测试**：集成测试支持并行运行（使用 `-p` 标志）
3. **清理数据**：确保测试后清理数据，避免影响后续测试
4. **监控资源**：检查连接池统计，避免连接泄漏
5. **版本兼容**：测试支持 PostgreSQL 13+ 所有版本

## 参考资料

- [PostgreSQL 文档](https://www.postgresql.org/docs/)
- [Go database/sql 包](https://pkg.go.dev/database/sql)
- [lib/pq 驱动](https://github.com/lib/pq)
- [Saga 模式文档](../../../docs/saga-user-guide.md)

## 联系支持

如遇到问题，请：
1. 查看上述故障排除指南
2. 检查 GitHub Issues
3. 提交新 Issue 并附上测试日志

