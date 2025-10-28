# Saga 分布式事务系统部署指南

本指南提供 Swit Saga 分布式事务系统在生产环境的完整部署方案,包括架构设计、配置管理、监控告警、高可用方案和运维最佳实践。

## 目录

- [部署架构](#部署架构)
- [生产环境配置](#生产环境配置)
- [监控和告警](#监控和告警)
- [高可用和容灾](#高可用和容灾)
- [性能调优](#性能调优)
- [运维操作手册](#运维操作手册)
- [故障排查](#故障排查)
- [安全最佳实践](#安全最佳实践)

---

## 部署架构

### 推荐架构

#### 1. 单机开发环境

适用于开发测试环境:

```text
┌─────────────────────────────────────────────────┐
│              Development Environment             │
│                                                  │
│  ┌─────────────┐                                │
│  │  Application│                                │
│  │   Service   │                                │
│  └──────┬──────┘                                │
│         │                                        │
│  ┌──────▼──────┐    ┌──────────┐               │
│  │    Saga     │    │  Memory  │               │
│  │ Coordinator │───▶│  Storage │               │
│  └─────────────┘    └──────────┘               │
│                                                  │
└─────────────────────────────────────────────────┘
```

**特点**:
- 内存状态存储
- 单实例部署
- 快速启动和测试
- 无外部依赖

#### 2. 小规模生产环境

适用于小型应用或初期生产环境:

```text
┌─────────────────────────────────────────────────┐
│           Small Production Environment           │
│                                                  │
│  ┌─────────────┐    ┌─────────────┐            │
│  │  Application│    │ Application │            │
│  │   Service   │    │   Service   │            │
│  └──────┬──────┘    └──────┬──────┘            │
│         │                  │                     │
│         └────────┬─────────┘                     │
│                  │                               │
│  ┌───────────────▼──────────┐                   │
│  │    Saga Coordinator      │                   │
│  └──────────┬────────────────┘                  │
│             │                                    │
│    ┌────────┴────────┐                          │
│    │                 │                          │
│  ┌─▼────┐     ┌─────▼──────┐                   │
│  │ Redis│     │   NATS     │                   │
│  │      │     │  (Events)  │                   │
│  └──────┘     └────────────┘                   │
│                                                  │
└─────────────────────────────────────────────────┘
```

**特点**:
- Redis 持久化存储
- NATS 事件发布
- 多应用实例
- 基础监控

**硬件要求**:
- CPU: 2-4 核
- 内存: 4-8 GB
- 存储: 50-100 GB SSD
- 网络: 1 Gbps

#### 3. 大规模高可用生产环境

适用于大型应用和关键业务:

```text
┌────────────────────────────────────────────────────────────────┐
│              High-Availability Production                       │
│                                                                 │
│              ┌─────────────────────┐                           │
│              │   Load Balancer     │                           │
│              └──────────┬──────────┘                           │
│                         │                                       │
│         ┌───────────────┼───────────────┐                      │
│         │               │               │                      │
│  ┌──────▼──────┐ ┌─────▼──────┐ ┌─────▼──────┐              │
│  │Application  │ │Application │ │Application │              │
│  │  Service 1  │ │  Service 2 │ │  Service 3 │              │
│  └──────┬──────┘ └─────┬──────┘ └─────┬──────┘              │
│         │               │               │                      │
│         └───────────────┼───────────────┘                      │
│                         │                                       │
│         ┌───────────────▼───────────────┐                      │
│         │  Saga Coordinator Cluster     │                     │
│         │  (Active-Active / Sharding)   │                     │
│         └───────────────┬───────────────┘                      │
│                         │                                       │
│         ┌───────────────┴───────────────┐                      │
│         │                               │                      │
│  ┌──────▼────────┐            ┌────────▼────────┐             │
│  │ Redis Cluster │            │  Kafka Cluster  │             │
│  │   (3 nodes)   │            │    (3 nodes)    │             │
│  │  Master-Slave │            │  Multi-Broker   │             │
│  └───────────────┘            └─────────────────┘             │
│                                                                 │
│  ┌──────────────────────────────────────────────────┐         │
│  │          Monitoring & Observability              │         │
│  │  Prometheus + Grafana + Jaeger + Alertmanager   │         │
│  └──────────────────────────────────────────────────┘         │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

**特点**:
- Redis 集群模式（主从复制 + 哨兵）
- Kafka 集群（多副本）
- 负载均衡
- 完整的监控和追踪
- 自动故障转移

**硬件要求**:

| 组件 | CPU | 内存 | 存储 | 实例数 |
|------|-----|------|------|--------|
| 应用服务 | 4-8 核 | 8-16 GB | 100 GB SSD | 3+ |
| Redis | 2-4 核 | 8-16 GB | 100 GB SSD | 3 (1主2从) |
| Kafka | 4-8 核 | 16-32 GB | 500 GB SSD | 3+ |
| PostgreSQL | 4-8 核 | 16-32 GB | 500 GB SSD | 3 (主从) |
| 监控栈 | 4 核 | 8 GB | 200 GB SSD | 按需 |

### 架构组件说明

#### Saga 协调器 (Saga Coordinator)

**职责**:
- 编排 Saga 工作流执行
- 管理步骤状态转换
- 执行补偿操作
- 处理超时和重试

**部署模式**:
- **单实例**: 开发/测试环境
- **主备模式**: 小规模生产环境
- **分片模式**: 大规模生产环境（按 Saga ID 哈希分片）

#### 状态存储 (State Storage)

支持的存储后端:

| 存储类型 | 适用场景 | 优点 | 缺点 |
|---------|---------|------|------|
| 内存 | 开发测试 | 快速、无依赖 | 不持久化 |
| Redis | 小规模生产 | 高性能、简单 | 内存限制 |
| PostgreSQL | 大规模生产 | 持久化、ACID | 延迟较高 |
| Cassandra | 超大规模 | 水平扩展 | 运维复杂 |

#### 事件发布器 (Event Publisher)

支持的消息中间件:

| 中间件 | 适用场景 | 优点 | 缺点 |
|--------|---------|------|------|
| 内存 | 开发测试 | 无依赖 | 不可靠 |
| NATS | 小规模生产 | 轻量、快速 | 功能有限 |
| RabbitMQ | 中等规模 | 功能丰富 | 性能一般 |
| Kafka | 大规模生产 | 高吞吐、持久化 | 复杂度高 |

---

## 生产环境配置

### 基础配置

#### 1. Saga 协调器配置

```yaml
# saga-config.yaml
saga:
  coordinator:
    # 协调器类型: memory, redis, postgresql
    type: redis
    
    # 工作线程池大小
    worker_pool_size: 100
    
    # 最大并发 Saga 数量
    max_concurrent_sagas: 500
    
    # 获取锁超时时间
    acquire_timeout: 30s
    
    # 优雅关闭超时
    shutdown_timeout: 2m
    
    # 心跳间隔（用于健康检查）
    heartbeat_interval: 10s

  # 状态存储配置
  state_storage:
    type: redis
    redis:
      # Redis 集群配置
      cluster:
        enabled: true
        nodes:
          - redis-1.prod:6379
          - redis-2.prod:6379
          - redis-3.prod:6379
      
      # Redis 单机配置（二选一）
      # standalone:
      #   address: redis.prod:6379
      
      # 认证
      password: ${REDIS_PASSWORD}
      
      # 连接池配置
      pool_size: 200
      min_idle_conns: 10
      max_retries: 5
      dial_timeout: 5s
      read_timeout: 3s
      write_timeout: 3s
      
      # 键前缀
      key_prefix: "saga:"
      
      # TTL 设置
      ttl: 7d  # Saga 状态保留时间

  # 事件发布器配置
  event_publisher:
    type: kafka
    kafka:
      brokers:
        - kafka-1.prod:9092
        - kafka-2.prod:9092
        - kafka-3.prod:9092
      
      # 主题配置
      topic: saga-events-prod
      partition_count: 12
      replication_factor: 3
      
      # 生产者配置
      producer:
        compression: snappy
        batch_size: 1000
        linger_time: 10ms
        max_message_bytes: 1048576  # 1MB
        idempotent: true
        acks: all  # 等待所有副本确认
        
      # 可靠性配置
      retries: 10
      retry_backoff: 100ms

  # 重试策略配置
  retry:
    default_policy:
      type: exponential_backoff
      max_attempts: 5
      initial_delay: 1s
      max_delay: 2m
      multiplier: 2.0
      jitter: true
      
    # 自定义重试策略
    policies:
      critical:
        type: exponential_backoff
        max_attempts: 10
        initial_delay: 500ms
        max_delay: 5m
        multiplier: 2.0
        jitter: true
      
      fast_fail:
        type: fixed
        max_attempts: 2
        delay: 100ms

  # 超时配置
  timeout:
    # 默认 Saga 超时时间
    default_saga_timeout: 5m
    
    # 默认步骤超时时间
    default_step_timeout: 30s
    
    # 默认补偿超时时间
    default_compensation_timeout: 1m

  # 监控配置
  monitoring:
    enabled: true
    
    # Prometheus 指标
    prometheus:
      enabled: true
      port: 9090
      path: /metrics
      
    # 分布式追踪
    tracing:
      enabled: true
      service_name: saga-coordinator
      exporter:
        type: jaeger
        endpoint: http://jaeger-collector:14268/api/traces
      sampling:
        type: parentbased
        rate: 0.1  # 10% 采样率
        
    # 日志配置
    logging:
      level: info
      format: json
      output: stdout
```

#### 2. Redis 集群配置

```yaml
# redis-cluster.conf
# 集群模式
cluster-enabled yes
cluster-config-file nodes.conf
cluster-node-timeout 5000

# 持久化配置
save 900 1
save 300 10
save 60 10000
appendonly yes
appendfsync everysec

# 内存配置
maxmemory 8gb
maxmemory-policy allkeys-lru

# 日志
loglevel notice
logfile /var/log/redis/redis.log

# 安全
requirepass ${REDIS_PASSWORD}
masterauth ${REDIS_PASSWORD}

# 性能优化
tcp-backlog 511
timeout 300
tcp-keepalive 300
```

**Redis 哨兵配置** (用于主从自动故障转移):

```conf
# sentinel.conf
sentinel monitor saga-redis redis-master 6379 2
sentinel auth-pass saga-redis ${REDIS_PASSWORD}
sentinel down-after-milliseconds saga-redis 5000
sentinel parallel-syncs saga-redis 1
sentinel failover-timeout saga-redis 180000
```

#### 3. Kafka 配置

```properties
# server.properties

############################# Server Basics #############################
broker.id=1
listeners=PLAINTEXT://kafka-1:9092
advertised.listeners=PLAINTEXT://kafka-1.prod:9092
num.network.threads=8
num.io.threads=16

############################# Log Basics #############################
log.dirs=/var/lib/kafka/data
num.partitions=12
default.replication.factor=3
min.insync.replicas=2

############################# Log Retention #############################
log.retention.hours=168  # 7 days
log.segment.bytes=1073741824  # 1GB
log.retention.check.interval.ms=300000

############################# Zookeeper #############################
zookeeper.connect=zk-1:2181,zk-2:2181,zk-3:2181
zookeeper.connection.timeout.ms=18000

############################# Performance #############################
socket.send.buffer.bytes=102400
socket.receive.buffer.bytes=102400
socket.request.max.bytes=104857600

# 压缩
compression.type=snappy

# 副本配置
replica.lag.time.max.ms=10000
replica.socket.timeout.ms=30000
replica.socket.receive.buffer.bytes=65536
```

#### 4. PostgreSQL 配置

用于持久化 Saga 状态（可选,适用于需要强一致性的场景）:

```yaml
# saga-postgresql-config.yaml
saga:
  state_storage:
    type: postgresql
    postgresql:
      host: postgres-master.prod
      port: 5432
      database: saga_db
      username: saga_user
      password: ${POSTGRES_PASSWORD}
      
      # 连接池配置
      max_open_conns: 100
      max_idle_conns: 10
      conn_max_lifetime: 1h
      conn_max_idle_time: 10m
      
      # SSL 配置
      sslmode: require
      sslcert: /etc/certs/client-cert.pem
      sslkey: /etc/certs/client-key.pem
      sslrootcert: /etc/certs/ca-cert.pem
      
      # 性能配置
      query_timeout: 30s
      
      # 表配置
      tables:
        saga_instances: saga_instances
        saga_steps: saga_step_states
        saga_events: saga_events
```

**数据库初始化脚本**:

```sql
-- scripts/sql/saga_schema.sql
CREATE TABLE saga_instances (
    saga_id VARCHAR(255) PRIMARY KEY,
    saga_type VARCHAR(255) NOT NULL,
    state VARCHAR(50) NOT NULL,
    started_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    context JSONB,
    metadata JSONB,
    INDEX idx_state (state),
    INDEX idx_started_at (started_at),
    INDEX idx_saga_type (saga_type)
);

CREATE TABLE saga_step_states (
    id SERIAL PRIMARY KEY,
    saga_id VARCHAR(255) NOT NULL,
    step_id VARCHAR(255) NOT NULL,
    step_name VARCHAR(255) NOT NULL,
    state VARCHAR(50) NOT NULL,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    input JSONB,
    output JSONB,
    error TEXT,
    retry_count INT DEFAULT 0,
    UNIQUE(saga_id, step_id),
    FOREIGN KEY (saga_id) REFERENCES saga_instances(saga_id) ON DELETE CASCADE,
    INDEX idx_saga_step (saga_id, step_id),
    INDEX idx_step_state (state)
);

CREATE TABLE saga_events (
    id SERIAL PRIMARY KEY,
    saga_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB,
    timestamp TIMESTAMP NOT NULL,
    FOREIGN KEY (saga_id) REFERENCES saga_instances(saga_id) ON DELETE CASCADE,
    INDEX idx_saga_events (saga_id, timestamp),
    INDEX idx_event_type (event_type)
);

-- 创建索引提升查询性能
CREATE INDEX idx_saga_timeout ON saga_instances(started_at) WHERE state IN ('RUNNING', 'COMPENSATING');
CREATE INDEX idx_active_sagas ON saga_instances(state, started_at) WHERE state = 'RUNNING';
```

### 环境配置最佳实践

#### 1. 配置文件分离

为不同环境维护独立配置:

```bash
configs/
├── development.yaml    # 开发环境
├── staging.yaml        # 预发布环境
├── production.yaml     # 生产环境
└── production-ha.yaml  # 高可用生产环境
```

#### 2. 敏感信息管理

使用环境变量或密钥管理服务:

```yaml
# 使用环境变量
saga:
  state_storage:
    redis:
      password: ${REDIS_PASSWORD}

# 使用密钥管理服务 (示例: HashiCorp Vault)
saga:
  state_storage:
    redis:
      password: vault:secret/data/saga/redis#password
```

#### 3. 配置验证

启动前验证配置:

```bash
# 使用 swit-serve 验证配置
./swit-serve --config=production.yaml --validate-config

# 使用 switctl 验证配置
./switctl config validate --file=production.yaml
```

---

## 监控和告警

### Prometheus 指标

#### 核心指标

```yaml
# Saga 执行指标
saga_started_total                     # Saga 启动总数
saga_completed_total{status}           # Saga 完成总数（按状态分类）
saga_failed_total{reason}              # Saga 失败总数（按原因分类）
saga_duration_seconds{quantile}        # Saga 执行时长分布
saga_active_count                      # 活跃 Saga 数量

# 步骤执行指标
saga_step_started_total                # 步骤启动总数
saga_step_completed_total{status}      # 步骤完成总数
saga_step_duration_seconds{quantile}   # 步骤执行时长
saga_step_retry_total                  # 步骤重试总数

# 补偿操作指标
saga_compensation_started_total        # 补偿操作启动总数
saga_compensation_completed_total      # 补偿操作完成总数
saga_compensation_failed_total         # 补偿操作失败总数

# 状态存储指标
saga_storage_operation_duration_seconds{operation}  # 存储操作延迟
saga_storage_error_total{operation}                 # 存储错误总数

# 事件发布指标
saga_event_published_total{topic}      # 事件发布总数
saga_event_publish_error_total{topic}  # 事件发布失败总数
saga_event_publish_duration_seconds    # 事件发布延迟

# 系统指标
saga_coordinator_goroutines            # Goroutine 数量
saga_coordinator_memory_bytes          # 内存使用
saga_coordinator_gc_duration_seconds   # GC 耗时
```

#### Prometheus 配置

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'saga-production'
    environment: 'prod'

# 告警规则
rule_files:
  - 'saga-alerts.yml'

# 抓取配置
scrape_configs:
  # Saga 协调器
  - job_name: 'saga-coordinator'
    static_configs:
      - targets:
          - 'saga-1:9090'
          - 'saga-2:9090'
          - 'saga-3:9090'
    metrics_path: /metrics
    scrape_interval: 10s

  # Redis 指标
  - job_name: 'redis'
    static_configs:
      - targets:
          - 'redis-exporter-1:9121'
          - 'redis-exporter-2:9121'
          - 'redis-exporter-3:9121'

  # Kafka 指标
  - job_name: 'kafka'
    static_configs:
      - targets:
          - 'kafka-1:9308'
          - 'kafka-2:9308'
          - 'kafka-3:9308'
```

### Grafana 仪表板

#### 预配置仪表板

创建 Grafana 仪表板 JSON 文件:

```json
{
  "dashboard": {
    "title": "Saga 生产监控",
    "tags": ["saga", "distributed-transaction"],
    "timezone": "Asia/Shanghai",
    "panels": [
      {
        "id": 1,
        "title": "Saga 执行速率",
        "type": "graph",
        "gridPos": {"x": 0, "y": 0, "w": 12, "h": 8},
        "targets": [
          {
            "expr": "rate(saga_started_total[5m])",
            "legendFormat": "启动速率"
          },
          {
            "expr": "rate(saga_completed_total{status=\"COMPLETED\"}[5m])",
            "legendFormat": "完成速率"
          },
          {
            "expr": "rate(saga_failed_total[5m])",
            "legendFormat": "失败速率"
          }
        ]
      },
      {
        "id": 2,
        "title": "成功率 (%)",
        "type": "singlestat",
        "gridPos": {"x": 12, "y": 0, "w": 6, "h": 8},
        "targets": [
          {
            "expr": "(rate(saga_completed_total{status=\"COMPLETED\"}[5m]) / rate(saga_started_total[5m])) * 100"
          }
        ],
        "thresholds": "90,95",
        "colors": ["#d44a3a", "#e0b400", "#299c46"]
      },
      {
        "id": 3,
        "title": "活跃 Saga 数量",
        "type": "singlestat",
        "gridPos": {"x": 18, "y": 0, "w": 6, "h": 8},
        "targets": [
          {
            "expr": "saga_active_count"
          }
        ]
      },
      {
        "id": 4,
        "title": "P50/P95/P99 执行时长",
        "type": "graph",
        "gridPos": {"x": 0, "y": 8, "w": 12, "h": 8},
        "targets": [
          {
            "expr": "histogram_quantile(0.50, rate(saga_duration_seconds_bucket[5m]))",
            "legendFormat": "P50"
          },
          {
            "expr": "histogram_quantile(0.95, rate(saga_duration_seconds_bucket[5m]))",
            "legendFormat": "P95"
          },
          {
            "expr": "histogram_quantile(0.99, rate(saga_duration_seconds_bucket[5m]))",
            "legendFormat": "P99"
          }
        ]
      },
      {
        "id": 5,
        "title": "失败原因分布",
        "type": "piechart",
        "gridPos": {"x": 12, "y": 8, "w": 12, "h": 8},
        "targets": [
          {
            "expr": "sum by (reason) (rate(saga_failed_total[5m]))"
          }
        ]
      }
    ]
  }
}
```

### 告警规则

```yaml
# saga-alerts.yml
groups:
  - name: saga.critical
    interval: 30s
    rules:
      # 高失败率告警
      - alert: HighSagaFailureRate
        expr: |
          (
            rate(saga_failed_total[5m]) 
            / 
            rate(saga_started_total[5m])
          ) > 0.1
        for: 5m
        labels:
          severity: critical
          component: saga-coordinator
        annotations:
          summary: "Saga 失败率过高"
          description: "过去 5 分钟 Saga 失败率超过 10%: {{ $value | humanizePercentage }}"
          
      # Saga 执行缓慢
      - alert: SlowSagaExecution
        expr: |
          histogram_quantile(0.95, 
            rate(saga_duration_seconds_bucket[5m])
          ) > 60
        for: 10m
        labels:
          severity: warning
          component: saga-coordinator
        annotations:
          summary: "Saga 执行缓慢"
          description: "P95 执行时长超过 60 秒: {{ $value }}s"
          
      # 活跃 Saga 过多
      - alert: TooManyActiveSagas
        expr: saga_active_count > 1000
        for: 5m
        labels:
          severity: warning
          component: saga-coordinator
        annotations:
          summary: "活跃 Saga 数量过多"
          description: "当前活跃 Saga: {{ $value }}"
          
      # Saga 协调器不可用
      - alert: SagaCoordinatorDown
        expr: up{job="saga-coordinator"} == 0
        for: 1m
        labels:
          severity: critical
          component: saga-coordinator
        annotations:
          summary: "Saga 协调器服务下线"
          description: "实例 {{ $labels.instance }} 已下线"

  - name: saga.compensation
    interval: 30s
    rules:
      # 补偿操作失败
      - alert: CompensationFailures
        expr: rate(saga_compensation_failed_total[5m]) > 0.1
        for: 5m
        labels:
          severity: critical
          component: saga-coordinator
        annotations:
          summary: "补偿操作失败"
          description: "补偿操作失败率: {{ $value | humanizePercentage }}"

  - name: saga.storage
    interval: 30s
    rules:
      # 存储延迟过高
      - alert: HighStorageLatency
        expr: |
          histogram_quantile(0.95,
            rate(saga_storage_operation_duration_seconds_bucket[5m])
          ) > 1
        for: 5m
        labels:
          severity: warning
          component: redis
        annotations:
          summary: "存储操作延迟过高"
          description: "P95 延迟: {{ $value }}s"
          
      # 存储错误
      - alert: StorageErrors
        expr: rate(saga_storage_error_total[5m]) > 0.01
        for: 2m
        labels:
          severity: critical
          component: redis
        annotations:
          summary: "存储操作错误"
          description: "错误率: {{ $value | humanizePercentage }}"

  - name: saga.events
    interval: 30s
    rules:
      # 事件发布失败
      - alert: EventPublishFailures
        expr: rate(saga_event_publish_error_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
          component: kafka
        annotations:
          summary: "事件发布失败"
          description: "失败率: {{ $value | humanizePercentage }}"
```

### Alertmanager 配置

```yaml
# alertmanager.yml
global:
  resolve_timeout: 5m
  smtp_smarthost: 'smtp.company.com:587'
  smtp_from: 'alerts@company.com'
  smtp_auth_username: 'alerts@company.com'
  smtp_auth_password: ${SMTP_PASSWORD}

# 告警路由
route:
  group_by: ['alertname', 'cluster', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  receiver: 'team-platform'
  
  routes:
    # 严重告警立即发送
    - match:
        severity: critical
      receiver: 'team-platform-critical'
      continue: true
      
    # 工作时间外的告警
    - match_re:
        time_of_day: '(0[0-7]|2[0-3])'
      receiver: 'on-call'

# 接收器配置
receivers:
  - name: 'team-platform'
    email_configs:
      - to: 'platform-team@company.com'
        headers:
          Subject: '[Saga] {{ .GroupLabels.alertname }}'
    
  - name: 'team-platform-critical'
    email_configs:
      - to: 'platform-oncall@company.com'
        send_resolved: true
    slack_configs:
      - api_url: ${SLACK_WEBHOOK_URL}
        channel: '#saga-alerts'
        title: '🚨 [CRITICAL] {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
    
  - name: 'on-call'
    pagerduty_configs:
      - service_key: ${PAGERDUTY_SERVICE_KEY}

# 抑制规则
inhibit_rules:
  # 协调器下线时抑制其他告警
  - source_match:
      alertname: 'SagaCoordinatorDown'
    target_match_re:
      alertname: '(HighSagaFailureRate|SlowSagaExecution)'
    equal: ['instance']
```

### 分布式追踪

#### Jaeger 配置

```yaml
# tracing-config.yaml
tracing:
  enabled: true
  service_name: saga-coordinator
  
  # 导出器配置
  exporter:
    type: jaeger
    endpoint: http://jaeger-collector.observability:14268/api/traces
    timeout: 30s
    batch_timeout: 5s
    max_export_batch_size: 512
    max_queue_size: 2048
  
  # 采样策略
  sampling:
    type: parentbased  # 基于父级 span 决策
    rate: 0.1          # 10% 基础采样率
    
    # 自定义采样规则
    rules:
      - service: saga-coordinator
        operation: ExecuteSaga
        rate: 0.5  # 关键操作提高采样率
      
      - service: saga-coordinator
        operation: HealthCheck
        rate: 0.01  # 健康检查降低采样率
  
  # 资源属性
  resource_attributes:
    deployment.environment: production
    service.namespace: distributed-transactions
    team.name: platform
```

---

## 高可用和容灾

### 高可用架构设计

#### 1. 协调器高可用

**主备模式** (适用于小规模):

```text
┌─────────────┐         ┌─────────────┐
│ Coordinator │         │ Coordinator │
│   Master    │◄───────►│   Standby   │
└──────┬──────┘         └──────┬──────┘
       │                       │
       └───────────┬───────────┘
                   │
            ┌──────▼──────┐
            │    Redis    │
            │  (Leader    │
            │  Election)  │
            └─────────────┘
```

实现方式:
- 使用 Redis/etcd 进行 Leader 选举
- Standby 实例监听 Master 心跳
- Master 故障时自动切换

**分片模式** (适用于大规模):

```text
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│Coordinator  │  │Coordinator  │  │Coordinator  │
│   Shard 1   │  │   Shard 2   │  │   Shard 3   │
│  (Hash 0-N) │  │ (Hash N-2N) │  │(Hash 2N-3N) │
└──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │                │                │
       └────────────────┼────────────────┘
                        │
                 ┌──────▼──────┐
                 │    Redis    │
                 │   Cluster   │
                 └─────────────┘
```

实现方式:
- 按 Saga ID 哈希分片
- 每个分片独立处理
- 支持动态扩缩容

#### 2. 存储高可用

**Redis 主从+哨兵模式**:

```bash
# 启动 Redis 主节点
redis-server /etc/redis/redis-master.conf

# 启动 Redis 从节点
redis-server /etc/redis/redis-slave.conf --slaveof redis-master 6379

# 启动哨兵
redis-sentinel /etc/redis/sentinel.conf
```

**Redis 集群模式**:

```bash
# 创建 Redis 集群
redis-cli --cluster create \
  redis-1:6379 \
  redis-2:6379 \
  redis-3:6379 \
  redis-4:6379 \
  redis-5:6379 \
  redis-6:6379 \
  --cluster-replicas 1
```

#### 3. 消息中间件高可用

**Kafka 多副本配置**:

```properties
# 每个分区 3 个副本
default.replication.factor=3

# 至少 2 个副本同步才算成功
min.insync.replicas=2

# 等待所有副本确认
acks=all

# 启用幂等生产
enable.idempotence=true
```

### 容灾方案

#### 1. 数据备份

**Redis 备份脚本**:

```bash
#!/bin/bash
# saga-redis-backup.sh

BACKUP_DIR="/backup/redis"
DATE=$(date +%Y%m%d_%H%M%S)
REDIS_HOST="redis-master"
REDIS_PORT=6379
REDIS_PASSWORD=${REDIS_PASSWORD}

# 创建备份目录
mkdir -p ${BACKUP_DIR}

# 触发 RDB 快照
redis-cli -h ${REDIS_HOST} -p ${REDIS_PORT} -a ${REDIS_PASSWORD} BGSAVE

# 等待快照完成
while [ $(redis-cli -h ${REDIS_HOST} -p ${REDIS_PORT} -a ${REDIS_PASSWORD} INFO persistence | grep rdb_bgsave_in_progress | cut -d: -f2) -eq 1 ]; do
    sleep 1
done

# 复制 RDB 文件
scp ${REDIS_HOST}:/var/lib/redis/dump.rdb ${BACKUP_DIR}/dump_${DATE}.rdb

# 压缩备份
gzip ${BACKUP_DIR}/dump_${DATE}.rdb

# 清理旧备份 (保留 7 天)
find ${BACKUP_DIR} -name "dump_*.rdb.gz" -mtime +7 -delete

echo "Redis 备份完成: ${BACKUP_DIR}/dump_${DATE}.rdb.gz"
```

**Kafka 数据镜像**:

```bash
# 使用 MirrorMaker 2 进行跨集群复制
bin/connect-mirror-maker.sh mm2.properties
```

`mm2.properties`:
```properties
# 源集群
clusters = primary, backup
primary.bootstrap.servers = kafka-1-primary:9092,kafka-2-primary:9092
backup.bootstrap.servers = kafka-1-backup:9092,kafka-2-backup:9092

# 复制配置
primary->backup.enabled = true
primary->backup.topics = saga-events-prod
backup->primary.enabled = true
backup->primary.topics = saga-events-prod

# 复制策略
replication.factor = 3
offset-syncs.topic.replication.factor = 3
```

#### 2. 故障恢复

**自动故障转移脚本**:

```bash
#!/bin/bash
# saga-failover.sh

COMPONENT=$1
ACTION=${2:-auto}

case $COMPONENT in
  coordinator)
    echo "执行协调器故障转移..."
    
    # 检查主节点状态
    if ! curl -f http://coordinator-master:9090/health > /dev/null 2>&1; then
      echo "主节点不可用,切换到备用节点"
      
      # 更新服务发现
      consul kv put saga/coordinator/leader coordinator-standby
      
      # 触发备用节点接管
      curl -X POST http://coordinator-standby:9090/takeover
      
      # 更新负载均衡
      kubectl set selector service/saga-coordinator app=saga,role=standby
    fi
    ;;
    
  redis)
    echo "执行 Redis 故障转移..."
    redis-cli -h redis-sentinel -p 26379 SENTINEL failover saga-redis
    ;;
    
  kafka)
    echo "Kafka 自动故障转移,无需手动干预"
    kafka-topics.sh --bootstrap-server kafka-1:9092 --describe --topic saga-events-prod
    ;;
esac
```

#### 3. 灾难恢复计划

**RTO 和 RPO 目标**:

| 场景 | RTO (恢复时间目标) | RPO (恢复点目标) | 策略 |
|------|-------------------|------------------|------|
| 协调器故障 | < 1 分钟 | 0 | 主备自动切换 |
| Redis 故障 | < 5 分钟 | < 1 分钟 | 哨兵自动切换 |
| Kafka 故障 | < 10 分钟 | 0 | 多副本自动恢复 |
| 数据中心故障 | < 30 分钟 | < 5 分钟 | 跨区域复制 |

**恢复流程**:

1. **检测故障**: 监控系统发现异常
2. **触发告警**: 通知运维团队
3. **评估影响**: 确定故障范围
4. **执行恢复**:
   - 自动: 触发自动故障转移
   - 手动: 运维人员介入
5. **验证恢复**: 检查服务是否正常
6. **事后分析**: 复盘并改进

---

## 性能调优

### 协调器调优

#### 1. 并发配置

```yaml
saga:
  coordinator:
    # 根据 CPU 核数调整
    worker_pool_size: 100  # 推荐: CPU 核数 * 25
    
    # 根据内存大小调整
    max_concurrent_sagas: 500  # 推荐: 可用内存(GB) * 100
```

**计算公式**:
```text
worker_pool_size = CPU 核数 * 25
max_concurrent_sagas = 可用内存(GB) * 100

例如: 4 核 8GB 内存
worker_pool_size = 4 * 25 = 100
max_concurrent_sagas = 8 * 100 = 800
```

#### 2. 超时优化

```yaml
saga:
  timeout:
    # 根据业务特点调整
    default_saga_timeout: 5m      # 长事务可延长至 10m-30m
    default_step_timeout: 30s     # 快速步骤可缩短至 10s
    default_compensation_timeout: 1m
```

#### 3. 批处理优化

```yaml
saga:
  event_publisher:
    kafka:
      producer:
        # 增加批量大小提升吞吐量
        batch_size: 1000          # 默认 1000,可增至 10000
        linger_time: 10ms         # 增加至 100ms 提升批量效率
        compression: snappy       # 使用压缩减少网络传输
```

### Redis 调优

#### 1. 内存优化

```conf
# redis.conf

# 设置合适的最大内存
maxmemory 8gb

# 使用 LRU 淘汰策略
maxmemory-policy allkeys-lru

# 禁用持久化提升性能 (仅缓存场景)
# save ""
# appendonly no

# 启用持久化保证数据安全 (推荐)
save 900 1
save 300 10
save 60 10000
appendonly yes
appendfsync everysec
```

#### 2. 连接池优化

```yaml
saga:
  state_storage:
    redis:
      # 增加连接池大小
      pool_size: 200         # 默认 200,高并发可增至 500
      min_idle_conns: 20     # 保持最小空闲连接
      
      # 优化超时设置
      dial_timeout: 5s
      read_timeout: 3s
      write_timeout: 3s
```

#### 3. Pipeline 批量操作

在代码中使用 Redis Pipeline:

```go
// 批量保存步骤状态
pipe := redisClient.Pipeline()
for _, step := range steps {
    pipe.HSet(ctx, fmt.Sprintf("saga:%s:step:%s", sagaID, step.ID), step)
}
_, err := pipe.Exec(ctx)
```

### Kafka 调优

#### 1. 生产者优化

```properties
# 批量配置
batch.size=16384
linger.ms=10
buffer.memory=33554432

# 压缩
compression.type=snappy

# 性能 vs 可靠性权衡
acks=1              # 仅等待 leader 确认 (高性能)
# acks=all          # 等待所有副本确认 (高可靠)

# 重试配置
retries=10
retry.backoff.ms=100
```

#### 2. 消费者优化

```properties
# 批量拉取
fetch.min.bytes=1
fetch.max.wait.ms=500
max.partition.fetch.bytes=1048576

# 并发消费
num.consumer.threads=8

# 提交策略
enable.auto.commit=false  # 手动提交保证可靠性
```

#### 3. 分区优化

```bash
# 增加分区数提升并行度
kafka-topics.sh --bootstrap-server kafka-1:9092 \
  --alter --topic saga-events-prod \
  --partitions 24
```

**分区数计算**:
```text
分区数 = max(
  目标吞吐量 / 单分区生产者吞吐量,
  目标吞吐量 / 单分区消费者吞吐量
)

例如:
- 目标吞吐量: 10000 msg/s
- 单分区生产者吞吐量: 1000 msg/s
- 单分区消费者吞吐量: 500 msg/s

分区数 = max(10000/1000, 10000/500) = 20
建议设置 20-24 个分区
```

### 数据库调优

#### PostgreSQL 优化

```sql
-- postgresql.conf

-- 内存配置
shared_buffers = 2GB
effective_cache_size = 6GB
work_mem = 64MB
maintenance_work_mem = 512MB

-- 连接配置
max_connections = 200
max_prepared_transactions = 100

-- 检查点配置
checkpoint_completion_target = 0.9
wal_buffers = 16MB
min_wal_size = 1GB
max_wal_size = 4GB

-- 查询优化
random_page_cost = 1.1  # SSD
effective_io_concurrency = 200

-- 日志配置
logging_collector = on
log_min_duration_statement = 1000  # 记录慢查询
```

**索引优化**:

```sql
-- 分析表统计信息
ANALYZE saga_instances;
ANALYZE saga_step_states;

-- 检查索引使用情况
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
WHERE schemaname = 'public';

-- 查找缺失的索引
SELECT * FROM pg_stat_user_tables WHERE seq_scan > idx_scan AND seq_scan > 1000;
```

### Go 应用调优

#### 1. GOMAXPROCS 设置

```go
import "runtime"

func init() {
    // 设置为 CPU 核数
    runtime.GOMAXPROCS(runtime.NumCPU())
}
```

#### 2. 内存限制

```bash
# 使用环境变量设置 GC 目标
export GOGC=100  # 默认 100,降低可减少内存但增加 GC 频率

# 设置内存软限制 (Go 1.19+)
export GOMEMLIMIT=8GiB
```

#### 3. Profiling

```bash
# 启用 pprof
go run -tags=pprof cmd/swit-serve/main.go

# 分析 CPU 使用
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# 分析内存使用
go tool pprof http://localhost:6060/debug/pprof/heap

# 分析 Goroutine
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

---

## 运维操作手册

### 日常运维

#### 1. 健康检查

```bash
#!/bin/bash
# saga-health-check.sh

echo "=== Saga 系统健康检查 ==="

# 检查协调器
echo "\n[1] 协调器健康状态:"
for host in saga-1 saga-2 saga-3; do
    if curl -f -s http://${host}:9090/health > /dev/null; then
        echo "✅ ${host}: 健康"
    else
        echo "❌ ${host}: 异常"
    fi
done

# 检查 Redis
echo "\n[2] Redis 状态:"
redis-cli -h redis-master INFO replication | grep -E '(role|connected_slaves)'

# 检查 Kafka
echo "\n[3] Kafka 状态:"
kafka-broker-api-versions.sh --bootstrap-server kafka-1:9092 | head -1

# 检查活跃 Saga
echo "\n[4] 活跃 Saga:"
curl -s http://saga-1:9090/metrics | grep saga_active_count

# 检查错误率
echo "\n[5] 错误率:"
curl -s http://saga-1:9090/metrics | grep -E 'saga_(failed|completed)_total'
```

#### 2. 数据清理

```bash
#!/bin/bash
# saga-cleanup.sh

# 清理已完成的 Saga (保留 7 天)
RETENTION_DAYS=7
CLEANUP_DATE=$(date -d "${RETENTION_DAYS} days ago" +%Y-%m-%d)

echo "清理 ${CLEANUP_DATE} 之前的 Saga 数据..."

# Redis 清理
redis-cli -h redis-master --scan --pattern "saga:*" | while read key; do
    created_at=$(redis-cli -h redis-master HGET $key created_at)
    if [[ "$created_at" < "$CLEANUP_DATE" ]]; then
        status=$(redis-cli -h redis-master HGET $key status)
        if [[ "$status" == "COMPLETED" || "$status" == "FAILED" ]]; then
            redis-cli -h redis-master DEL $key
            echo "删除: $key"
        fi
    fi
done

# PostgreSQL 清理
psql -h postgres-master -U saga_user -d saga_db <<EOF
DELETE FROM saga_instances 
WHERE completed_at < NOW() - INTERVAL '${RETENTION_DAYS} days'
  AND state IN ('COMPLETED', 'FAILED', 'CANCELLED');

DELETE FROM saga_events
WHERE timestamp < NOW() - INTERVAL '${RETENTION_DAYS} days';

VACUUM ANALYZE saga_instances;
VACUUM ANALYZE saga_step_states;
VACUUM ANALYZE saga_events;
EOF

echo "清理完成"
```

#### 3. 指标收集

```bash
#!/bin/bash
# saga-collect-metrics.sh

OUTPUT_DIR="/var/log/saga/metrics"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p ${OUTPUT_DIR}

# 收集 Prometheus 指标
curl -s http://saga-1:9090/metrics > ${OUTPUT_DIR}/metrics_${DATE}.txt

# 收集系统指标
echo "=== System Metrics ===" > ${OUTPUT_DIR}/system_${DATE}.txt
top -b -n 1 >> ${OUTPUT_DIR}/system_${DATE}.txt
free -h >> ${OUTPUT_DIR}/system_${DATE}.txt
df -h >> ${OUTPUT_DIR}/system_${DATE}.txt

# 收集 Redis 指标
redis-cli -h redis-master INFO > ${OUTPUT_DIR}/redis_${DATE}.txt

# 收集 Kafka 指标
kafka-topics.sh --bootstrap-server kafka-1:9092 --describe --topic saga-events-prod \
  > ${OUTPUT_DIR}/kafka_${DATE}.txt

echo "指标已保存到 ${OUTPUT_DIR}"
```

### 备份和恢复

#### 1. 全量备份

```bash
#!/bin/bash
# saga-full-backup.sh

BACKUP_ROOT="/backup/saga"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="${BACKUP_ROOT}/${DATE}"

mkdir -p ${BACKUP_DIR}

echo "开始全量备份..."

# 1. 备份 Redis
echo "[1/4] 备份 Redis..."
redis-cli -h redis-master BGSAVE
while [ $(redis-cli -h redis-master INFO persistence | grep rdb_bgsave_in_progress | cut -d: -f2) -eq 1 ]; do
    sleep 1
done
scp redis-master:/var/lib/redis/dump.rdb ${BACKUP_DIR}/redis_dump.rdb

# 2. 备份 PostgreSQL
echo "[2/4] 备份 PostgreSQL..."
pg_dump -h postgres-master -U saga_user saga_db | gzip > ${BACKUP_DIR}/postgres_dump.sql.gz

# 3. 备份配置
echo "[3/4] 备份配置文件..."
cp /etc/saga/production.yaml ${BACKUP_DIR}/
cp /etc/prometheus/saga-alerts.yml ${BACKUP_DIR}/

# 4. 创建备份清单
echo "[4/4] 创建备份清单..."
cat > ${BACKUP_DIR}/backup_manifest.txt <<EOF
备份时间: $(date)
备份类型: 全量备份
Redis 备份: redis_dump.rdb
PostgreSQL 备份: postgres_dump.sql.gz
配置文件: production.yaml, saga-alerts.yml
EOF

# 压缩备份
tar -czf ${BACKUP_ROOT}/saga_backup_${DATE}.tar.gz -C ${BACKUP_ROOT} ${DATE}
rm -rf ${BACKUP_DIR}

echo "备份完成: ${BACKUP_ROOT}/saga_backup_${DATE}.tar.gz"

# 上传到对象存储 (可选)
# aws s3 cp ${BACKUP_ROOT}/saga_backup_${DATE}.tar.gz s3://saga-backups/
```

#### 2. 恢复操作

```bash
#!/bin/bash
# saga-restore.sh

BACKUP_FILE=$1

if [ -z "$BACKUP_FILE" ]; then
    echo "用法: $0 <backup_file>"
    exit 1
fi

echo "从备份恢复: ${BACKUP_FILE}"

# 解压备份
TEMP_DIR=$(mktemp -d)
tar -xzf ${BACKUP_FILE} -C ${TEMP_DIR}
BACKUP_DIR=$(ls ${TEMP_DIR})

# 1. 停止服务
echo "[1/5] 停止 Saga 服务..."
kubectl scale deployment saga-coordinator --replicas=0

# 2. 恢复 Redis
echo "[2/5] 恢复 Redis..."
redis-cli -h redis-master FLUSHALL
scp ${TEMP_DIR}/${BACKUP_DIR}/redis_dump.rdb redis-master:/var/lib/redis/dump.rdb
redis-cli -h redis-master SHUTDOWN NOSAVE
sleep 5
# Redis 会自动重启并加载 dump.rdb

# 3. 恢复 PostgreSQL
echo "[3/5] 恢复 PostgreSQL..."
dropdb -h postgres-master -U saga_user saga_db
createdb -h postgres-master -U saga_user saga_db
gunzip < ${TEMP_DIR}/${BACKUP_DIR}/postgres_dump.sql.gz | \
  psql -h postgres-master -U saga_user saga_db

# 4. 恢复配置
echo "[4/5] 恢复配置..."
cp ${TEMP_DIR}/${BACKUP_DIR}/production.yaml /etc/saga/

# 5. 重启服务
echo "[5/5] 重启服务..."
kubectl scale deployment saga-coordinator --replicas=3
kubectl rollout status deployment/saga-coordinator

# 清理
rm -rf ${TEMP_DIR}

echo "恢复完成"
```

### 升级操作

#### 1. 滚动升级

```bash
#!/bin/bash
# saga-rolling-upgrade.sh

NEW_VERSION=$1

if [ -z "$NEW_VERSION" ]; then
    echo "用法: $0 <new_version>"
    exit 1
fi

echo "开始滚动升级到版本 ${NEW_VERSION}..."

# 1. 拉取新镜像
echo "[1/6] 拉取新镜像..."
docker pull innovationmech/swit:${NEW_VERSION}

# 2. 更新 Kubernetes Deployment
echo "[2/6] 更新 Deployment..."
kubectl set image deployment/saga-coordinator \
  saga-coordinator=innovationmech/swit:${NEW_VERSION}

# 3. 监控滚动更新
echo "[3/6] 监控滚动更新..."
kubectl rollout status deployment/saga-coordinator --timeout=10m

# 4. 验证新版本
echo "[4/6] 验证新版本..."
for i in {1..3}; do
    POD=$(kubectl get pods -l app=saga-coordinator -o jsonpath="{.items[$i].metadata.name}")
    VERSION=$(kubectl exec $POD -- /app/swit-serve --version)
    echo "Pod $POD: $VERSION"
done

# 5. 健康检查
echo "[5/6] 健康检查..."
sleep 30
for i in {1..3}; do
    POD=$(kubectl get pods -l app=saga-coordinator -o jsonpath="{.items[$i].metadata.name}")
    HEALTH=$(kubectl exec $POD -- curl -s http://localhost:9090/health | jq -r .status)
    if [ "$HEALTH" != "healthy" ]; then
        echo "❌ Pod $POD 健康检查失败"
        echo "回滚升级..."
        kubectl rollout undo deployment/saga-coordinator
        exit 1
    fi
    echo "✅ Pod $POD: $HEALTH"
done

# 6. 验证指标
echo "[6/6] 验证指标..."
ERROR_RATE=$(curl -s 'http://prometheus:9090/api/v1/query?query=rate(saga_failed_total[5m])/rate(saga_started_total[5m])' | \
  jq -r '.data.result[0].value[1]')

if (( $(echo "$ERROR_RATE > 0.1" | bc -l) )); then
    echo "❌ 错误率过高: $ERROR_RATE"
    echo "回滚升级..."
    kubectl rollout undo deployment/saga-coordinator
    exit 1
fi

echo "✅ 升级成功完成"
```

#### 2. 蓝绿部署

```bash
#!/bin/bash
# saga-blue-green-deployment.sh

NEW_VERSION=$1

if [ -z "$NEW_VERSION" ]; then
    echo "用法: $0 <new_version>"
    exit 1
fi

echo "开始蓝绿部署到版本 ${NEW_VERSION}..."

# 1. 部署绿色环境
echo "[1/5] 部署绿色环境..."
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: saga-coordinator-green
spec:
  replicas: 3
  selector:
    matchLabels:
      app: saga-coordinator
      environment: green
  template:
    metadata:
      labels:
        app: saga-coordinator
        environment: green
    spec:
      containers:
      - name: saga-coordinator
        image: innovationmech/swit:${NEW_VERSION}
        # ... 其他配置 ...
EOF

# 2. 等待绿色环境就绪
echo "[2/5] 等待绿色环境就绪..."
kubectl rollout status deployment/saga-coordinator-green

# 3. 验证绿色环境
echo "[3/5] 验证绿色环境..."
# 将 1% 流量切换到绿色环境
kubectl patch service saga-coordinator --type='json' \
  -p='[{"op": "add", "path": "/spec/selector/environment", "value": "green"}]'

sleep 60

# 检查错误率
GREEN_ERROR_RATE=$(curl -s 'http://prometheus:9090/api/v1/query?query=rate(saga_failed_total{environment="green"}[5m])/rate(saga_started_total{environment="green"}[5m])' | \
  jq -r '.data.result[0].value[1]')

if (( $(echo "$GREEN_ERROR_RATE > 0.1" | bc -l) )); then
    echo "❌ 绿色环境错误率过高: $GREEN_ERROR_RATE"
    echo "回滚到蓝色环境..."
    kubectl patch service saga-coordinator --type='json' \
      -p='[{"op": "remove", "path": "/spec/selector/environment"}]'
    kubectl delete deployment saga-coordinator-green
    exit 1
fi

# 4. 切换全部流量
echo "[4/5] 切换全部流量到绿色环境..."
kubectl patch service saga-coordinator --type='json' \
  -p='[{"op": "replace", "path": "/spec/selector/environment", "value": "green"}]'

# 5. 清理蓝色环境
echo "[5/5] 清理蓝色环境..."
kubectl delete deployment saga-coordinator

# 将绿色环境重命名为蓝色
kubectl patch deployment saga-coordinator-green --type='json' \
  -p='[{"op": "replace", "path": "/metadata/name", "value": "saga-coordinator"}]'

echo "✅ 蓝绿部署成功完成"
```

### 扩缩容操作

#### 1. 手动扩容

```bash
#!/bin/bash
# saga-scale.sh

ACTION=$1
REPLICAS=$2

if [ -z "$ACTION" ] || [ -z "$REPLICAS" ]; then
    echo "用法: $0 <scale-up|scale-down> <replicas>"
    exit 1
fi

echo "${ACTION} Saga 协调器到 ${REPLICAS} 个副本..."

# 更新副本数
kubectl scale deployment saga-coordinator --replicas=${REPLICAS}

# 等待就绪
kubectl rollout status deployment/saga-coordinator

# 验证
READY_REPLICAS=$(kubectl get deployment saga-coordinator -o jsonpath='{.status.readyReplicas}')
echo "就绪副本数: ${READY_REPLICAS}/${REPLICAS}"

if [ "$READY_REPLICAS" -eq "$REPLICAS" ]; then
    echo "✅ 扩缩容成功"
else
    echo "❌ 扩缩容失败,部分副本未就绪"
    exit 1
fi
```

#### 2. 自动扩缩容

```yaml
# saga-hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: saga-coordinator-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: saga-coordinator
  minReplicas: 3
  maxReplicas: 10
  metrics:
    # 基于 CPU 扩缩容
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    
    # 基于内存扩缩容
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
    
    # 基于自定义指标扩缩容
    - type: Pods
      pods:
        metric:
          name: saga_active_count
        target:
          type: AverageValue
          averageValue: "100"
  
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
        - type: Percent
          value: 50
          periodSeconds: 60
        - type: Pods
          value: 2
          periodSeconds: 60
      selectPolicy: Max
    
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 10
          periodSeconds: 60
        - type: Pods
          value: 1
          periodSeconds: 120
      selectPolicy: Min
```

---

## 故障排查

### 常见问题

#### 1. Saga 执行缓慢

**症状**:
- P95 执行时长超过预期
- 活跃 Saga 数量积压

**排查步骤**:

```bash
# 1. 检查协调器资源使用
kubectl top pods -l app=saga-coordinator

# 2. 检查存储延迟
redis-cli -h redis-master --latency

# 3. 检查事件发布延迟
curl -s http://saga-1:9090/metrics | grep saga_event_publish_duration

# 4. 分析慢 Saga
curl -s http://saga-1:9090/debug/slow-sagas | jq .
```

**解决方案**:
1. 增加协调器副本数
2. 优化 Redis 配置或升级硬件
3. 检查网络延迟
4. 优化 Saga 步骤实现

#### 2. Saga 失败率高

**症状**:
- `saga_failed_total` 指标快速增长
- 大量补偿操作执行

**排查步骤**:

```bash
# 1. 查看失败原因分布
curl -s http://saga-1:9090/metrics | grep saga_failed_total

# 2. 查看最近失败的 Saga
redis-cli -h redis-master --scan --pattern "saga:*" | while read key; do
    status=$(redis-cli -h redis-master HGET $key status)
    if [ "$status" == "FAILED" ]; then
        redis-cli -h redis-master HGETALL $key
    fi
done

# 3. 检查依赖服务
kubectl get pods
kubectl logs -l app=payment-service --tail=100

# 4. 查看 Jaeger 追踪
# 访问 Jaeger UI 查看失败的 trace
```

**解决方案**:
1. 修复依赖服务故障
2. 调整重试策略
3. 增加超时时间
4. 修复业务逻辑bug

#### 3. 内存泄漏

**症状**:
- 内存使用持续增长
- OOM Killed 事件

**排查步骤**:

```bash
# 1. 检查内存使用趋势
kubectl top pods -l app=saga-coordinator --sort-by=memory

# 2. 获取 heap profile
curl http://saga-1:9090/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# 3. 检查 Goroutine 泄漏
curl http://saga-1:9090/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof

# 4. 查看 GC 统计
curl -s http://saga-1:9090/metrics | grep go_gc
```

**解决方案**:
1. 修复代码中的 Goroutine 泄漏
2. 增加内存限制
3. 调整 GOGC 参数
4. 清理过期数据

#### 4. Redis 连接耗尽

**症状**:
- 错误日志: "Too many clients"
- `saga_storage_error_total` 增加

**排查步骤**:

```bash
# 1. 检查当前连接数
redis-cli -h redis-master CLIENT LIST | wc -l
redis-cli -h redis-master CONFIG GET maxclients

# 2. 查看慢查询
redis-cli -h redis-master SLOWLOG GET 10

# 3. 检查连接池配置
kubectl describe configmap saga-config | grep pool_size
```

**解决方案**:
1. 增加 Redis maxclients
2. 优化连接池配置
3. 使用连接池复用
4. 修复连接泄漏

#### 5. Kafka 消费延迟

**症状**:
- Consumer lag 持续增长
- 事件处理延迟

**排查步骤**:

```bash
# 1. 检查消费者 lag
kafka-consumer-groups.sh --bootstrap-server kafka-1:9092 \
  --group saga-event-consumers --describe

# 2. 检查分区分配
kafka-consumer-groups.sh --bootstrap-server kafka-1:9092 \
  --group saga-event-consumers --members

# 3. 检查主题配置
kafka-topics.sh --bootstrap-server kafka-1:9092 \
  --describe --topic saga-events-prod
```

**解决方案**:
1. 增加消费者数量
2. 增加分区数
3. 优化消费者处理逻辑
4. 检查网络带宽

### 应急响应

```bash
#!/bin/bash
# saga-emergency.sh

ISSUE=$1

case $ISSUE in
  high_load)
    echo "处理高负载..."
    # 1. 临时扩容
    kubectl scale deployment saga-coordinator --replicas=10
    
    # 2. 降低采样率
    kubectl set env deployment/saga-coordinator TRACING_SAMPLING_RATE=0.01
    
    # 3. 限流
    # 在应用层实现限流逻辑
    ;;
    
  storage_error)
    echo "处理存储错误..."
    # 1. 切换到备用 Redis
    kubectl set env deployment/saga-coordinator REDIS_HOST=redis-standby
    
    # 2. 重启主 Redis
    redis-cli -h redis-master SHUTDOWN NOSAVE
    ;;
    
  memory_leak)
    echo "处理内存泄漏..."
    # 1. 滚动重启
    kubectl rollout restart deployment/saga-coordinator
    
    # 2. 收集 heap dump
    for pod in $(kubectl get pods -l app=saga-coordinator -o name); do
        kubectl exec $pod -- curl http://localhost:9090/debug/pprof/heap > heap_${pod}.prof
    done
    ;;
    
  *)
    echo "未知问题: $ISSUE"
    exit 1
    ;;
esac
```

---

## 安全最佳实践

### 认证和授权

#### 1. JWT 认证配置

```yaml
# saga-security-config.yaml
saga:
  security:
    authentication:
      enabled: true
      type: jwt
      jwt:
        secret: ${JWT_SECRET}  # 从密钥管理服务获取
        issuer: saga-coordinator
        audience: saga-clients
        expiration: 1h
        
    authorization:
      enabled: true
      rbac:
        enabled: true
        roles:
          admin:
            permissions:
              - saga:*
          operator:
            permissions:
              - saga:read
              - saga:execute
          viewer:
            permissions:
              - saga:read
```

#### 2. mTLS 配置

```yaml
# 协调器 TLS 配置
saga:
  server:
    tls:
      enabled: true
      cert_file: /etc/certs/server.crt
      key_file: /etc/certs/server.key
      client_ca_file: /etc/certs/ca.crt
      client_auth_type: RequireAndVerifyClientCert
```

### 数据加密

#### 1. 传输加密

```yaml
# Redis TLS
saga:
  state_storage:
    redis:
      tls:
        enabled: true
        cert_file: /etc/certs/redis-client.crt
        key_file: /etc/certs/redis-client.key
        ca_file: /etc/certs/ca.crt
        insecure_skip_verify: false

# Kafka TLS
saga:
  event_publisher:
    kafka:
      security:
        protocol: SSL
        ssl:
          ca_cert_file: /etc/certs/ca.crt
          client_cert_file: /etc/certs/kafka-client.crt
          client_key_file: /etc/certs/kafka-client.key
```

#### 2. 静态数据加密

```yaml
saga:
  security:
    encryption:
      enabled: true
      algorithm: AES-256-GCM
      key_manager:
        type: vault  # 使用 Vault 管理密钥
        vault:
          address: https://vault.company.com
          token: ${VAULT_TOKEN}
          key_path: secret/data/saga/encryption-key
      
      # 需要加密的字段
      encrypted_fields:
        - password
        - credit_card
        - ssn
        - api_key
```

### 审计日志

```yaml
saga:
  security:
    audit:
      enabled: true
      # 日志输出
      output:
        type: file
        file:
          path: /var/log/saga/audit.log
          max_size: 100MB
          max_backups: 10
          compress: true
      
      # 审计事件
      events:
        - saga.created
        - saga.started
        - saga.completed
        - saga.failed
        - saga.compensated
        - auth.login
        - auth.logout
        - config.changed
      
      # 审计字段
      fields:
        - timestamp
        - user_id
        - client_ip
        - action
        - resource
        - result
        - error
```

### 网络安全

#### 1. 网络策略

```yaml
# saga-network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: saga-coordinator-network-policy
spec:
  podSelector:
    matchLabels:
      app: saga-coordinator
  
  policyTypes:
    - Ingress
    - Egress
  
  ingress:
    # 只允许来自应用服务的流量
    - from:
      - podSelector:
          matchLabels:
            role: application
      ports:
      - protocol: TCP
        port: 9000
    
    # 允许 Prometheus 抓取指标
    - from:
      - podSelector:
          matchLabels:
            app: prometheus
      ports:
      - protocol: TCP
        port: 9090
  
  egress:
    # 允许访问 Redis
    - to:
      - podSelector:
          matchLabels:
            app: redis
      ports:
      - protocol: TCP
        port: 6379
    
    # 允许访问 Kafka
    - to:
      - podSelector:
          matchLabels:
            app: kafka
      ports:
      - protocol: TCP
        port: 9092
    
    # 允许 DNS 查询
    - to:
      - namespaceSelector: {}
      ports:
      - protocol: UDP
        port: 53
```

#### 2. Pod 安全策略

```yaml
# saga-pod-security.yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: saga-coordinator-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  
  # 只读根文件系统
  readOnlyRootFilesystem: true
  
  # 必需的安全上下文
  requiredDropCapabilities:
    - ALL
  
  # 允许的卷类型
  volumes:
    - configMap
    - secret
    - emptyDir
    - persistentVolumeClaim
  
  # 运行用户
  runAsUser:
    rule: MustRunAsNonRoot
  
  seLinux:
    rule: RunAsAny
  
  supplementalGroups:
    rule: RunAsAny
  
  fsGroup:
    rule: RunAsAny
```

---

## 附录

### A. 配置文件模板

完整的生产环境配置模板已创建在:
- `examples/deployment-templates/saga-production.yaml`
- `examples/deployment-templates/saga-redis.conf`
- `examples/deployment-templates/saga-kafka.properties`

### B. 运维脚本

所有运维脚本已提供在:
- `scripts/tools/saga-health-check.sh`
- `scripts/tools/saga-backup.sh`
- `scripts/tools/saga-restore.sh`
- `scripts/tools/saga-upgrade.sh`

### C. 监控面板

Grafana 面板配置:
- `examples/saga-monitoring/grafana-dashboard.json`

Prometheus 告警规则:
- `examples/saga-monitoring/alert-rules.yml`

### D. 部署清单

Kubernetes 部署清单:
- `deployments/k8s/saga-coordinator.yaml`
- `deployments/helm/swit/templates/saga-*.yaml`

### E. 相关文档

- [Saga API 参考](api-reference.md)
- [Saga 用户指南](user-guide.md)
- [Saga 开发者文档](developer-guide.md)
- [Saga 教程和最佳实践](tutorials.md)
- [运维指南](../operations-guide.md)
- [架构文档](../architecture.md)

---

## 获取帮助

如果在部署或运维过程中遇到问题:

1. **查看日志**: 使用 `kubectl logs` 或相关日志工具
2. **检查指标**: 访问 Grafana 仪表板
3. **查看追踪**: 使用 Jaeger UI 分析问题
4. **社区支持**: 提交 GitHub Issue
5. **商业支持**: 联系技术支持团队

---

**版本**: 1.0  
**最后更新**: 2025-01  
**维护者**: Swit Platform Team

