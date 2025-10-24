# Saga 示例应用 Docker 使用指南

本文档介绍如何使用 Docker Compose 运行 Saga 示例应用及其依赖服务。

## 目录

- [概述](#概述)
- [前置条件](#前置条件)
- [快速开始](#快速开始)
- [服务说明](#服务说明)
- [配置说明](#配置说明)
- [常用操作](#常用操作)
- [监控与可视化](#监控与可视化)
- [故障排除](#故障排除)
- [最佳实践](#最佳实践)

## 概述

Docker Compose 配置提供了 Saga 示例应用的完整运行环境，包括：

- **数据存储**: PostgreSQL（状态持久化）、Redis（缓存）
- **消息队列**: RabbitMQ、NATS（可选）
- **分布式追踪**: Jaeger（追踪可视化）
- **监控系统**: Prometheus（指标收集）、Grafana（可视化面板）
- **数据导出器**: PostgreSQL Exporter、Redis Exporter

## 前置条件

### 软件要求

- Docker Engine 20.10+
- Docker Compose 2.0+ (推荐使用 Docker Compose V2)
- 至少 4GB 可用内存
- 至少 10GB 可用磁盘空间

### 验证安装

```bash
# 检查 Docker 版本
docker --version

# 检查 Docker Compose 版本
docker compose version

# 检查 Docker 运行状态
docker ps
```

## 快速开始

### 1. 准备环境

```bash
# 进入示例目录
cd pkg/saga/examples

# 复制环境变量配置文件
cp .env.example .env

# 根据需要编辑 .env 文件
nano .env  # 或使用您喜欢的编辑器
```

### 2. 启动所有服务

```bash
# 启动所有服务（后台运行）
docker compose up -d

# 查看服务状态
docker compose ps

# 查看服务日志
docker compose logs -f
```

### 3. 验证服务

```bash
# 检查所有服务健康状态
docker compose ps

# 访问 Web UI
# - Grafana: http://localhost:3000 (admin/admin)
# - Jaeger: http://localhost:16686
# - Prometheus: http://localhost:9090
# - RabbitMQ Management: http://localhost:15672 (guest/guest)
```

### 4. 运行示例应用

```bash
# 在宿主机运行示例（连接到 Docker 服务）
cd pkg/saga/examples
go test -v -run TestOrderSaga

# 或使用脚本运行
./scripts/run.sh order
```

### 5. 停止服务

```bash
# 停止所有服务
docker compose down

# 停止并删除所有数据卷（清理所有数据）
docker compose down -v
```

## 服务说明

### 核心数据服务

#### PostgreSQL (端口: 5432)
- **用途**: Saga 状态持久化存储
- **镜像**: postgres:15-alpine
- **数据卷**: `saga-examples-postgres-data`
- **健康检查**: 每 10 秒检查一次连接
- **默认凭据**: postgres/password
- **数据库**: saga

**配置优化**:
- 最大连接数: 200
- 共享缓冲区: 256MB
- 有效缓存: 1GB

#### Redis (端口: 6379)
- **用途**: 缓存和临时状态存储
- **镜像**: redis:7-alpine
- **数据卷**: `saga-examples-redis-data`
- **持久化**: AOF（每秒同步）
- **内存限制**: 512MB（可配置）
- **淘汰策略**: allkeys-lru

### 消息队列服务

#### RabbitMQ (端口: 5672, 15672, 15692)
- **用途**: 事件发布和消息传递
- **镜像**: rabbitmq:3.12-management-alpine
- **数据卷**: `saga-examples-rabbitmq-data`
- **管理界面**: http://localhost:15672
- **Prometheus 指标**: http://localhost:15692/metrics
- **默认凭据**: guest/guest

#### NATS (端口: 4222, 8222)
- **用途**: 可选的消息队列（轻量级）
- **镜像**: nats:2.10-alpine
- **数据卷**: `saga-examples-nats-data`
- **JetStream**: 已启用
- **管理端点**: http://localhost:8222

### 监控与追踪

#### Jaeger (端口: 16686, 14268, 14250)
- **用途**: 分布式追踪可视化
- **镜像**: jaegertracing/all-in-one:1.50
- **UI 界面**: http://localhost:16686
- **存储类型**: Badger（持久化）
- **数据卷**: `saga-examples-jaeger-data`
- **支持协议**: OTLP (gRPC/HTTP), Jaeger, Zipkin

**端口说明**:
- 16686: Jaeger UI
- 14268: HTTP collector
- 14250: gRPC collector
- 6831/udp: Jaeger agent (compact thrift)
- 9411: Zipkin compatible endpoint
- 4317: OTLP gRPC
- 4318: OTLP HTTP

#### Prometheus (端口: 9090)
- **用途**: 指标收集和存储
- **镜像**: prom/prometheus:v2.47.0
- **UI 界面**: http://localhost:9090
- **数据卷**: `saga-examples-prometheus-data`
- **数据保留**: 30 天（可配置）
- **存储限制**: 10GB（可配置）

#### Grafana (端口: 3000)
- **用途**: 监控数据可视化
- **镜像**: grafana/grafana:10.1.0
- **UI 界面**: http://localhost:3000
- **数据卷**: `saga-examples-grafana-data`
- **默认凭据**: admin/admin
- **预配置数据源**: Prometheus

### 数据导出器

#### PostgreSQL Exporter (端口: 9187)
- 导出 PostgreSQL 数据库指标到 Prometheus
- 指标端点: http://localhost:9187/metrics

#### Redis Exporter (端口: 9121)
- 导出 Redis 缓存指标到 Prometheus
- 指标端点: http://localhost:9121/metrics

## 配置说明

### 环境变量

所有配置通过 `.env` 文件管理。主要配置项：

#### 数据库配置
```bash
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=password
POSTGRES_DB=saga
```

#### Redis 配置
```bash
REDIS_PORT=6379
REDIS_MAX_MEMORY=512mb
```

#### RabbitMQ 配置
```bash
RABBITMQ_PORT=5672
RABBITMQ_MANAGEMENT_PORT=15672
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
```

#### Jaeger 配置
```bash
JAEGER_UI_PORT=16686
JAEGER_STORAGE_TYPE=badger
JAEGER_LOG_LEVEL=info
```

#### 监控配置
```bash
PROMETHEUS_PORT=9090
PROMETHEUS_RETENTION_TIME=30d
GRAFANA_PORT=3000
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=admin
```

### Prometheus 配置

编辑 `prometheus.yml` 可以自定义监控配置：

```yaml
# 抓取间隔
scrape_interval: 15s

# 添加新的抓取目标
scrape_configs:
  - job_name: 'my-service'
    static_configs:
      - targets: ['my-service:8080']
```

### Grafana 配置

#### 自动配置的数据源
- Prometheus (默认): http://prometheus:9090

#### 添加自定义仪表板
1. 将 JSON 仪表板文件放入 `grafana/dashboards/`
2. 重启 Grafana 服务：`docker compose restart grafana`

## 常用操作

### 启动和停止

```bash
# 启动所有服务
docker compose up -d

# 启动特定服务
docker compose up -d postgres redis

# 停止所有服务
docker compose down

# 停止特定服务
docker compose stop postgres

# 重启服务
docker compose restart postgres
```

### 查看日志

```bash
# 查看所有服务日志
docker compose logs

# 跟踪日志输出
docker compose logs -f

# 查看特定服务日志
docker compose logs postgres
docker compose logs -f jaeger

# 查看最近 100 行日志
docker compose logs --tail=100
```

### 服务管理

```bash
# 查看服务状态
docker compose ps

# 查看服务详细信息
docker compose ps -a

# 查看资源使用情况
docker stats
```

### 数据管理

```bash
# 列出所有数据卷
docker volume ls | grep saga-examples

# 备份 PostgreSQL 数据
docker compose exec postgres pg_dump -U postgres saga > backup.sql

# 恢复 PostgreSQL 数据
docker compose exec -T postgres psql -U postgres saga < backup.sql

# 清理未使用的数据卷
docker volume prune
```

### 网络管理

```bash
# 查看网络信息
docker network inspect saga-examples-network

# 测试服务连通性
docker compose exec postgres ping redis
```

### 扩展和更新

```bash
# 拉取最新镜像
docker compose pull

# 重新构建并启动
docker compose up -d --build

# 查看镜像版本
docker compose images
```

## 监控与可视化

### Jaeger 追踪

1. 访问 Jaeger UI: http://localhost:16686
2. 选择服务: saga-examples
3. 查看追踪链路和性能分析
4. 搜索特定 Trace ID 或 Operation

**常用功能**:
- Service 视图: 查看服务调用关系
- Trace 视图: 查看完整请求链路
- Compare 功能: 对比不同请求的性能

### Prometheus 监控

1. 访问 Prometheus UI: http://localhost:9090
2. 使用 PromQL 查询指标
3. 查看目标健康状态: Status → Targets
4. 查看告警规则: Status → Rules

**常用查询**:
```promql
# Saga 执行成功率
rate(saga_executions_total{status="success"}[5m])

# PostgreSQL 连接数
pg_stat_database_numbackends

# Redis 内存使用率
redis_memory_used_bytes / redis_memory_max_bytes
```

### Grafana 可视化

1. 访问 Grafana: http://localhost:3000
2. 登录: admin/admin（首次登录会提示修改密码）
3. 浏览预配置的仪表板
4. 创建自定义仪表板

**预置面板**:
- Saga 执行统计
- 数据库性能监控
- Redis 缓存分析
- 消息队列监控

### RabbitMQ 管理

1. 访问 RabbitMQ Management UI: http://localhost:15672
2. 登录: guest/guest
3. 查看队列、交换机、连接
4. 监控消息流量和性能

## 故障排除

### 常见问题

#### 1. 服务启动失败

**问题**: 服务无法启动或健康检查失败

```bash
# 查看详细错误信息
docker compose logs <service-name>

# 检查端口占用
lsof -i :<port>
netstat -tuln | grep <port>

# 检查磁盘空间
df -h
```

**解决方案**:
- 确保端口未被占用
- 检查磁盘空间是否充足
- 增加 Docker 内存限制

#### 2. 数据库连接失败

**问题**: 应用无法连接到 PostgreSQL

```bash
# 检查 PostgreSQL 健康状态
docker compose exec postgres pg_isready -U postgres

# 测试数据库连接
docker compose exec postgres psql -U postgres -d saga -c "SELECT 1;"

# 查看数据库日志
docker compose logs postgres
```

**解决方案**:
- 等待数据库完全启动（查看健康检查状态）
- 检查数据库凭据是否正确
- 验证网络连通性

#### 3. 内存不足

**问题**: Redis 或其他服务因内存不足而重启

```bash
# 检查容器资源使用
docker stats

# 调整 Redis 内存限制
# 编辑 .env 文件
REDIS_MAX_MEMORY=1024mb
```

**解决方案**:
- 增加 Docker Desktop 的内存限制
- 调整服务的内存配置
- 减少并发服务数量

#### 4. 数据卷权限问题

**问题**: 无法写入数据卷

```bash
# 检查数据卷权限
docker volume inspect saga-examples-postgres-data

# 修复权限（如需要）
docker compose down
docker volume rm saga-examples-postgres-data
docker compose up -d
```

#### 5. 网络问题

**问题**: 服务之间无法通信

```bash
# 检查网络配置
docker network inspect saga-examples-network

# 测试服务间连通性
docker compose exec postgres ping redis
docker compose exec redis ping postgres

# 重建网络
docker compose down
docker network prune
docker compose up -d
```

### 性能优化

#### PostgreSQL 优化

```bash
# 调整连接池大小（根据负载）
# 编辑 docker-compose.yml
-c max_connections=500

# 增加共享缓冲区
-c shared_buffers=512MB
```

#### Redis 优化

```bash
# 增加内存限制
REDIS_MAX_MEMORY=1024mb

# 调整持久化策略（根据需求）
--appendfsync no  # 更快但可能丢失数据
--appendfsync everysec  # 平衡（默认）
--appendfsync always  # 最安全但较慢
```

### 清理和重置

```bash
# 完全清理（删除所有数据）
docker compose down -v
docker system prune -a --volumes

# 仅清理未使用的资源
docker system prune

# 重新开始
docker compose up -d
```

## 最佳实践

### 开发环境

1. **使用 .env 文件管理配置**
   - 不要提交 `.env` 文件到版本控制
   - 使用 `.env.example` 作为模板

2. **定期备份数据**
   ```bash
   # 创建备份脚本
   ./scripts/backup.sh
   ```

3. **监控资源使用**
   ```bash
   # 定期检查
   docker stats
   docker system df
   ```

4. **保持镜像更新**
   ```bash
   # 定期更新
   docker compose pull
   docker compose up -d
   ```

### 生产环境

1. **修改默认密码**
   - 更改所有服务的默认密码
   - 使用强密码和密钥

2. **配置持久化存储**
   - 使用外部数据卷或挂载点
   - 定期备份重要数据

3. **启用 TLS/SSL**
   - 配置服务间加密通信
   - 使用反向代理（如 Nginx）

4. **限制资源使用**
   ```yaml
   deploy:
     resources:
       limits:
         cpus: '0.5'
         memory: 512M
   ```

5. **配置日志轮转**
   ```yaml
   logging:
     driver: "json-file"
     options:
       max-size: "10m"
       max-file: "3"
   ```

6. **实施监控和告警**
   - 配置 Prometheus 告警规则
   - 集成告警通知（如 Slack、Email）

### 安全建议

1. **网络隔离**
   - 仅暴露必要的端口
   - 使用防火墙规则限制访问

2. **凭据管理**
   - 使用 Docker secrets 管理敏感信息
   - 不要在代码中硬编码密码

3. **镜像安全**
   - 使用官方镜像或可信来源
   - 定期扫描镜像漏洞

4. **访问控制**
   - 启用服务的认证和授权
   - 限制管理界面的访问

## 相关文档

- [Saga 示例主文档](README.md)
- [端到端测试文档](E2E_TESTING.md)
- [脚本使用指南](scripts/README.md)
- [订单处理示例](docs/order_saga.md)
- [支付处理示例](docs/payment_saga.md)
- [库存管理示例](docs/inventory_saga.md)
- [用户注册示例](docs/user_registration_saga.md)

## 参考资源

- [Docker Compose 文档](https://docs.docker.com/compose/)
- [PostgreSQL Docker 镜像](https://hub.docker.com/_/postgres)
- [Redis Docker 镜像](https://hub.docker.com/_/redis)
- [RabbitMQ Docker 镜像](https://hub.docker.com/_/rabbitmq)
- [Jaeger 文档](https://www.jaegertracing.io/docs/)
- [Prometheus 文档](https://prometheus.io/docs/)
- [Grafana 文档](https://grafana.com/docs/)

## 获取帮助

如果遇到问题或有建议，请：

1. 查看本文档的故障排除部分
2. 搜索相关 Issue: https://github.com/innovationmech/swit/issues
3. 创建新的 Issue 并附上详细信息
4. 联系维护者: dreamerlyj@gmail.com

## 许可证

MIT License - 详见 [LICENSE](../../../LICENSE) 文件

