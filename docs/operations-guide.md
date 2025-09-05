# 分布式追踪运维指南

本文档为运维人员提供 SWIT 框架分布式追踪系统的生产环境部署、监控、故障排查和维护指南。

## 目录

- [部署最佳实践](#部署最佳实践)
- [环境配置](#环境配置)
- [监控设置](#监控设置)
- [性能调优](#性能调优)
- [故障排查](#故障排查)
- [容量规划](#容量规划)
- [安全配置](#安全配置)
- [维护操作](#维护操作)

## 部署最佳实践

### 生产环境架构

#### 推荐架构

```text
                    ┌─────────────────┐
                    │   Load Balancer │
                    └─────────────────┘
                             │
                ┌────────────┼────────────┐
                │            │            │
         ┌──────▼──────┐ ┌───▼────┐ ┌────▼─────┐
         │ Application │ │   App  │ │   App    │
         │  Services   │ │ Service│ │ Service  │
         └─────────────┘ └────────┘ └──────────┘
                │            │            │
                └────────────┼────────────┘
                             │
                 ┌───────────▼───────────┐
                 │   Jaeger Agents       │
                 │  (on each node)       │
                 └───────────────────────┘
                             │
                ┌────────────▼────────────┐
                │   Jaeger Collectors    │
                │    (Load Balanced)     │
                └────────────────────────┘
                             │
                ┌────────────▼────────────┐
                │     Storage Layer      │
                │  (Elasticsearch/       │
                │   Cassandra)           │
                └────────────────────────┘
                             │
                ┌────────────▼────────────┐
                │    Jaeger Query/UI     │
                │    (Load Balanced)     │
                └────────────────────────┘
```

### Docker Compose 生产配置

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  # Elasticsearch 集群
  elasticsearch-master:
    image: elasticsearch:8.8.0
    environment:
      - cluster.name=jaeger-cluster
      - node.name=master-1
      - discovery.type=zen
      - discovery.zen.ping.unicast.hosts=elasticsearch-master,elasticsearch-data-1,elasticsearch-data-2
      - cluster.initial_master_nodes=master-1
      - "ES_JAVA_OPTS=-Xms2g -Xmx2g"
      - xpack.security.enabled=true
      - ELASTIC_PASSWORD=${ELASTICSEARCH_PASSWORD}
    ulimits:
      memlock:
        soft: -1
        hard: -1
    volumes:
      - elasticsearch_master_data:/usr/share/elasticsearch/data
    networks:
      - jaeger-network
    deploy:
      replicas: 1
      resources:
        limits:
          memory: 4G
        reservations:
          memory: 2G

  elasticsearch-data-1:
    image: elasticsearch:8.8.0
    environment:
      - cluster.name=jaeger-cluster
      - node.name=data-1
      - node.roles=data
      - discovery.zen.ping.unicast.hosts=elasticsearch-master,elasticsearch-data-1,elasticsearch-data-2
      - "ES_JAVA_OPTS=-Xms4g -Xmx4g"
      - xpack.security.enabled=true
      - ELASTIC_PASSWORD=${ELASTICSEARCH_PASSWORD}
    ulimits:
      memlock:
        soft: -1
        hard: -1
    volumes:
      - elasticsearch_data_1:/usr/share/elasticsearch/data
    networks:
      - jaeger-network
    deploy:
      replicas: 1
      resources:
        limits:
          memory: 8G
        reservations:
          memory: 4G

  elasticsearch-data-2:
    image: elasticsearch:8.8.0
    environment:
      - cluster.name=jaeger-cluster
      - node.name=data-2
      - node.roles=data
      - discovery.zen.ping.unicast.hosts=elasticsearch-master,elasticsearch-data-1,elasticsearch-data-2
      - "ES_JAVA_OPTS=-Xms4g -Xmx4g"
      - xpack.security.enabled=true
      - ELASTIC_PASSWORD=${ELASTICSEARCH_PASSWORD}
    ulimits:
      memlock:
        soft: -1
        hard: -1
    volumes:
      - elasticsearch_data_2:/usr/share/elasticsearch/data
    networks:
      - jaeger-network
    deploy:
      replicas: 1
      resources:
        limits:
          memory: 8G
        reservations:
          memory: 4G

  # Jaeger Collectors
  jaeger-collector-1:
    image: jaegertracing/jaeger-collector:1.42
    environment:
      - SPAN_STORAGE_TYPE=elasticsearch
      - ES_SERVER_URLS=http://elasticsearch-master:9200,http://elasticsearch-data-1:9200,http://elasticsearch-data-2:9200
      - ES_USERNAME=elastic
      - ES_PASSWORD=${ELASTICSEARCH_PASSWORD}
      - ES_NUM_SHARDS=3
      - ES_NUM_REPLICAS=1
      - COLLECTOR_ZIPKIN_HOST_PORT=:9411
      - COLLECTOR_OTLP_ENABLED=true
    ports:
      - "14268:14268"  # Jaeger thrift
      - "14250:14250"  # Jaeger gRPC
      - "9411:9411"    # Zipkin
      - "4317:4317"    # OTLP gRPC
      - "4318:4318"    # OTLP HTTP
    depends_on:
      - elasticsearch-master
    networks:
      - jaeger-network
    deploy:
      replicas: 2
      resources:
        limits:
          memory: 2G
          cpus: "1"
        reservations:
          memory: 512M
          cpus: "0.5"

  # Jaeger Query
  jaeger-query-1:
    image: jaegertracing/jaeger-query:1.42
    environment:
      - SPAN_STORAGE_TYPE=elasticsearch
      - ES_SERVER_URLS=http://elasticsearch-master:9200,http://elasticsearch-data-1:9200,http://elasticsearch-data-2:9200
      - ES_USERNAME=elastic
      - ES_PASSWORD=${ELASTICSEARCH_PASSWORD}
      - QUERY_BASE_PATH=/jaeger
    ports:
      - "16686:16686"
    depends_on:
      - elasticsearch-master
    networks:
      - jaeger-network
    deploy:
      replicas: 2
      resources:
        limits:
          memory: 1G
          cpus: "0.5"
        reservations:
          memory: 256M
          cpus: "0.25"

  # Jaeger Agent (部署在每个应用节点)
  jaeger-agent:
    image: jaegertracing/jaeger-agent:1.42
    command:
      - "--collector.host-port=jaeger-collector-1:14267"
      - "--reporter.grpc.host-port=jaeger-collector-1:14250"
    ports:
      - "6831:6831/udp"  # Jaeger thrift compact
      - "6832:6832/udp"  # Jaeger thrift binary
      - "5778:5778"      # Config server
    networks:
      - jaeger-network
    deploy:
      mode: global  # 每个节点部署一个
      resources:
        limits:
          memory: 256M
          cpus: "0.25"

volumes:
  elasticsearch_master_data:
  elasticsearch_data_1:
  elasticsearch_data_2:

networks:
  jaeger-network:
    driver: overlay
```

### Kubernetes 部署

#### Jaeger Operator 部署

```yaml
# jaeger-operator.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: observability
---
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: jaeger-production
  namespace: observability
spec:
  strategy: production
  storage:
    type: elasticsearch
    options:
      es:
        server-urls: "http://elasticsearch.observability.svc.cluster.local:9200"
        username: elastic
        password: ${ELASTICSEARCH_PASSWORD}
        index-prefix: jaeger
        num-shards: 3
        num-replicas: 1
  collector:
    replicas: 3
    resources:
      limits:
        memory: 2Gi
        cpu: 1000m
      requests:
        memory: 512Mi
        cpu: 500m
  query:
    replicas: 2
    resources:
      limits:
        memory: 1Gi
        cpu: 500m
      requests:
        memory: 256Mi
        cpu: 250m
  agent:
    strategy: DaemonSet
    resources:
      limits:
        memory: 256Mi
        cpu: 250m
      requests:
        memory: 128Mi
        cpu: 100m
```

#### 应用部署配置

```yaml
# app-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: order-service
  namespace: production
spec:
  replicas: 3
  selector:
    matchLabels:
      app: order-service
  template:
    metadata:
      labels:
        app: order-service
    spec:
      containers:
      - name: order-service
        image: order-service:v1.0.0
        env:
        - name: SWIT_TRACING_ENABLED
          value: "true"
        - name: SWIT_TRACING_SERVICE_NAME
          value: "order-service"
        - name: SWIT_TRACING_EXPORTER_TYPE
          value: "jaeger"
        - name: SWIT_TRACING_EXPORTER_ENDPOINT
          value: "http://jaeger-production-collector.observability.svc.cluster.local:14268/api/traces"
        - name: SWIT_TRACING_SAMPLING_TYPE
          value: "traceidratio"
        - name: SWIT_TRACING_SAMPLING_RATE
          value: "0.01"  # 1% 采样率
        - name: JAEGER_AGENT_HOST
          valueFrom:
            fieldRef:
              fieldPath: status.hostIP
        resources:
          limits:
            memory: 1Gi
            cpu: 500m
          requests:
            memory: 256Mi
            cpu: 100m
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

## 环境配置

### 生产环境配置

```yaml
# swit-production.yaml
server:
  name: "order-service"
  environment: "production"
  http:
    port: "8080"
    read_timeout: "30s"
    write_timeout: "30s"
    idle_timeout: "60s"

tracing:
  enabled: true
  service_name: "order-service"
  
  exporter:
    type: "jaeger"
    endpoint: "http://jaeger-collector:14268/api/traces"
    timeout: "30s"
    batch_timeout: "5s"
    max_export_batch_size: 512
    max_queue_size: 2048
    
  sampling:
    type: "parentbased"  # 基于父级决策
    rate: 0.01           # 1% 基础采样率
    
  resource_attributes:
    deployment.environment: "production"
    service.version: "${SERVICE_VERSION}"
    service.namespace: "commerce"
    team.name: "platform"
    cost.center: "engineering"

logging:
  level: "info"
  format: "json"
  output: "stdout"
```

### 环境变量配置

```bash
# production.env
# 基础配置
SWIT_SERVER_ENVIRONMENT=production
SWIT_SERVER_NAME=order-service

# 追踪配置
SWIT_TRACING_ENABLED=true
SWIT_TRACING_SERVICE_NAME=order-service

# 导出器配置
SWIT_TRACING_EXPORTER_TYPE=jaeger
SWIT_TRACING_EXPORTER_ENDPOINT=http://jaeger-collector:14268/api/traces
SWIT_TRACING_EXPORTER_TIMEOUT=30s
SWIT_TRACING_EXPORTER_BATCH_TIMEOUT=5s
SWIT_TRACING_EXPORTER_MAX_EXPORT_BATCH_SIZE=512
SWIT_TRACING_EXPORTER_MAX_QUEUE_SIZE=2048

# 采样配置
SWIT_TRACING_SAMPLING_TYPE=parentbased
SWIT_TRACING_SAMPLING_RATE=0.01

# 资源属性
SWIT_TRACING_RESOURCE_ATTRIBUTES_DEPLOYMENT_ENVIRONMENT=production
SWIT_TRACING_RESOURCE_ATTRIBUTES_SERVICE_VERSION=${SERVICE_VERSION}
SWIT_TRACING_RESOURCE_ATTRIBUTES_SERVICE_NAMESPACE=commerce

# 数据库配置
DATABASE_URL=postgres://user:pass@db:5432/orders
REDIS_URL=redis://redis:6379/0

# 外部服务配置  
PAYMENT_SERVICE_URL=http://payment-service:8080
INVENTORY_SERVICE_URL=http://inventory-service:8080
```

## 监控设置

### Jaeger 监控指标

#### Prometheus 配置

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  # Jaeger Collector 指标
  - job_name: 'jaeger-collector'
    static_configs:
      - targets: ['jaeger-collector:14269']
    metrics_path: /metrics
    scrape_interval: 15s

  # Jaeger Query 指标
  - job_name: 'jaeger-query'
    static_configs:
      - targets: ['jaeger-query:16687']
    metrics_path: /metrics
    scrape_interval: 15s

  # Jaeger Agent 指标
  - job_name: 'jaeger-agent'
    static_configs:
      - targets: ['jaeger-agent:14271']
    metrics_path: /metrics
    scrape_interval: 15s
    
  # 应用服务指标
  - job_name: 'order-service'
    static_configs:
      - targets: ['order-service:8080']
    metrics_path: /metrics
    scrape_interval: 15s
```

#### Grafana 仪表板

```json
{
  "dashboard": {
    "title": "Jaeger 追踪监控",
    "panels": [
      {
        "title": "追踪写入率",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(jaeger_collector_spans_received_total[5m])",
            "legendFormat": "接收的 Spans"
          },
          {
            "expr": "rate(jaeger_collector_spans_saved_by_svc_total[5m])",
            "legendFormat": "保存的 Spans"
          }
        ]
      },
      {
        "title": "存储延迟",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, jaeger_collector_save_latency_bucket)",
            "legendFormat": "95th percentile"
          },
          {
            "expr": "histogram_quantile(0.50, jaeger_collector_save_latency_bucket)",
            "legendFormat": "50th percentile"
          }
        ]
      },
      {
        "title": "队列使用率",
        "type": "graph",
        "targets": [
          {
            "expr": "jaeger_collector_queue_length / jaeger_collector_queue_capacity * 100",
            "legendFormat": "队列使用率 %"
          }
        ]
      }
    ]
  }
}
```

### 告警规则

#### Prometheus 告警规则

```yaml
# jaeger-alerts.yml
groups:
  - name: jaeger.rules
    rules:
    # Jaeger Collector 故障
    - alert: JaegerCollectorDown
      expr: up{job="jaeger-collector"} == 0
      for: 1m
      labels:
        severity: critical
      annotations:
        summary: "Jaeger Collector 服务下线"
        description: "Jaeger Collector {{ $labels.instance }} 已经下线超过 1 分钟"
        
    # 高错误率
    - alert: JaegerCollectorHighErrorRate
      expr: rate(jaeger_collector_spans_rejected_total[5m]) / rate(jaeger_collector_spans_received_total[5m]) > 0.1
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Jaeger Collector 高错误率"
        description: "Jaeger Collector 错误率超过 10%，当前值: {{ $value }}"

    # 队列满载
    - alert: JaegerCollectorQueueFull
      expr: jaeger_collector_queue_length / jaeger_collector_queue_capacity > 0.9
      for: 2m
      labels:
        severity: warning
      annotations:
        summary: "Jaeger Collector 队列接近满载"
        description: "队列使用率: {{ $value }}%"

    # 存储延迟过高
    - alert: JaegerCollectorHighLatency
      expr: histogram_quantile(0.95, jaeger_collector_save_latency_bucket) > 1
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "Jaeger 存储延迟过高"
        description: "95th 分位延迟: {{ $value }}s"

    # Elasticsearch 存储空间不足
    - alert: ElasticsearchDiskSpaceWarning
      expr: elasticsearch_filesystem_data_available_bytes / elasticsearch_filesystem_data_size_bytes < 0.2
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Elasticsearch 磁盘空间不足"
        description: "剩余空间: {{ $value }}%"
```

### 日志监控

#### 应用日志配置

```yaml
# 应用日志配置
logging:
  level: "info"
  format: "json"
  
  # 追踪相关日志
  tracing:
    level: "warn"  # 只记录警告和错误
    
  # 日志字段
  fields:
    service: "order-service"
    version: "${SERVICE_VERSION}"
    environment: "production"
```

#### 日志聚合 (ELK Stack)

```yaml
# filebeat.yml
filebeat.inputs:
- type: container
  paths:
    - /var/lib/docker/containers/*/*.log
  processors:
    - add_docker_metadata: ~

processors:
  - decode_json_fields:
      fields: ["message"]
      target: ""
      overwrite_keys: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "application-logs-%{+yyyy.MM.dd}"
  
logging.level: info
logging.to_files: true
```

## 性能调优

### 采样策略优化

#### 动态采样配置

```yaml
# 采样策略配置文件
{
  "service_strategies": [
    {
      "service": "order-service",
      "type": "probabilistic",
      "param": 0.1,
      "operation_strategies": [
        {
          "operation": "CreateOrder",
          "type": "probabilistic", 
          "param": 0.5
        },
        {
          "operation": "GetOrder",
          "type": "probabilistic",
          "param": 0.01
        }
      ]
    },
    {
      "service": "payment-service",
      "type": "probabilistic",
      "param": 0.05
    }
  ],
  "default_strategy": {
    "type": "probabilistic",
    "param": 0.01
  }
}
```

#### 自适应采样

```go
// 自适应采样器
type AdaptiveSampler struct {
    serviceName    string
    targetTPS      float64
    currentRate    float64
    lastAdjustment time.Time
    mutex          sync.RWMutex
}

func NewAdaptiveSampler(serviceName string, targetTPS float64) *AdaptiveSampler {
    return &AdaptiveSampler{
        serviceName:    serviceName,
        targetTPS:      targetTPS,
        currentRate:    0.1, // 初始 10% 采样率
        lastAdjustment: time.Now(),
    }
}

func (s *AdaptiveSampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
    s.mutex.RLock()
    rate := s.currentRate
    s.mutex.RUnlock()
    
    // 基于当前采样率决策
    sampler := trace.TraceIDRatioBased(rate)
    return sampler.ShouldSample(p)
}

// 调整采样率
func (s *AdaptiveSampler) AdjustRate(currentTPS float64) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    if time.Since(s.lastAdjustment) < time.Minute {
        return // 限制调整频率
    }
    
    if currentTPS > s.targetTPS*1.2 {
        // TPS 过高，降低采样率
        s.currentRate = s.currentRate * 0.8
    } else if currentTPS < s.targetTPS*0.8 {
        // TPS 过低，提高采样率
        s.currentRate = s.currentRate * 1.2
    }
    
    // 限制采样率范围
    if s.currentRate < 0.001 {
        s.currentRate = 0.001
    } else if s.currentRate > 1.0 {
        s.currentRate = 1.0
    }
    
    s.lastAdjustment = time.Now()
}
```

### 存储优化

#### Elasticsearch 优化

```yaml
# elasticsearch.yml
cluster.name: jaeger-cluster

# JVM 设置
ES_JAVA_OPTS: "-Xms8g -Xmx8g"

# 索引设置
index.number_of_shards: 3
index.number_of_replicas: 1
index.refresh_interval: "30s"

# 批量设置
bulk.queue_size: 1000
bulk.flush_size: "50mb"
bulk.flush_interval: "5s"

# 内存设置
indices.memory.index_buffer_size: "30%"
indices.fielddata.cache.size: "20%"

# 日志级别
logger.level: WARN

# 自动创建索引模板
template.jaeger-span:
  index_patterns: ["jaeger-span-*"]
  settings:
    number_of_shards: 3
    number_of_replicas: 1
    refresh_interval: "30s"
  mappings:
    properties:
      timestamp:
        type: date
      duration:
        type: long
      tags:
        type: nested
```

#### 索引生命周期管理

```json
{
  "policy": {
    "phases": {
      "hot": {
        "actions": {
          "rollover": {
            "max_size": "10gb",
            "max_age": "1d"
          }
        }
      },
      "warm": {
        "min_age": "7d",
        "actions": {
          "allocate": {
            "number_of_replicas": 0
          }
        }
      },
      "cold": {
        "min_age": "30d",
        "actions": {
          "allocate": {
            "number_of_replicas": 0
          }
        }
      },
      "delete": {
        "min_age": "90d"
      }
    }
  }
}
```

### 网络优化

#### 负载均衡配置

```nginx
# nginx.conf
upstream jaeger_collectors {
    least_conn;
    server jaeger-collector-1:14268 weight=1 max_fails=3 fail_timeout=30s;
    server jaeger-collector-2:14268 weight=1 max_fails=3 fail_timeout=30s;
    server jaeger-collector-3:14268 weight=1 max_fails=3 fail_timeout=30s;
}

upstream jaeger_queries {
    least_conn;
    server jaeger-query-1:16686 weight=1 max_fails=3 fail_timeout=30s;
    server jaeger-query-2:16686 weight=1 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name jaeger.company.com;
    
    # 收集器代理
    location /api/traces {
        proxy_pass http://jaeger_collectors;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
        
        # 限制请求体大小
        client_max_body_size 10m;
    }
    
    # 查询 UI 代理
    location / {
        proxy_pass http://jaeger_queries;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 故障排查

### 常见问题诊断

#### 1. 追踪数据缺失

**诊断步骤：**

```bash
# 1. 检查应用配置
curl -s http://app:8080/debug/config | jq '.tracing'

# 2. 检查 Jaeger Agent 连接
telnet jaeger-agent 6831

# 3. 检查 Collector 状态
curl -s http://jaeger-collector:14269/metrics | grep jaeger_collector_spans

# 4. 检查存储连接
curl -s http://elasticsearch:9200/_cluster/health

# 5. 查看应用日志
docker logs app-service | grep -i tracing
```

**解决方案：**

```yaml
# 应用临时配置 - 启用调试模式
tracing:
  enabled: true
  debug: true
  sampling:
    type: "always_on"  # 临时 100% 采样
  exporter:
    type: "console"    # 临时本地输出
```

#### 2. 高延迟问题

**监控命令：**

```bash
# 查看 Collector 指标
curl -s http://jaeger-collector:14269/metrics | grep -E '(latency|duration)'

# 查看存储性能
curl -s http://elasticsearch:9200/_cat/nodes?v&h=name,heap.percent,ram.percent,cpu,load_1m

# 查看队列状态
curl -s http://jaeger-collector:14269/metrics | grep queue
```

**优化措施：**

```yaml
# Collector 优化配置
collector:
  queue-size: 4096          # 增加队列大小
  num-workers: 100          # 增加工作线程
  batch-size: 1000          # 批量大小
  batch-timeout: "1s"       # 批量超时
  
  # 存储优化
  es-bulk-size: 10000000    # 10MB 批量
  es-bulk-workers: 10       # 10 个写入线程
  es-bulk-flush-interval: "1s"
```

#### 3. 存储空间问题

**监控脚本：**

```bash
#!/bin/bash
# storage-monitor.sh

ES_HOST="elasticsearch:9200"

# 检查索引大小
curl -s "${ES_HOST}/_cat/indices/jaeger-*?v&h=index,store.size&s=store.size:desc"

# 检查集群状态
curl -s "${ES_HOST}/_cluster/health?pretty"

# 检查磁盘使用
curl -s "${ES_HOST}/_cat/allocation?v"

# 删除旧索引
OLD_INDICES=$(curl -s "${ES_HOST}/_cat/indices/jaeger-*" | awk '{print $3}' | grep $(date -d '30 days ago' +%Y-%m-%d) || true)
if [ ! -z "$OLD_INDICES" ]; then
    echo "Deleting old indices: $OLD_INDICES"
    curl -X DELETE "${ES_HOST}/${OLD_INDICES}"
fi
```

#### 4. 内存泄漏

**诊断工具：**

```bash
# 应用内存使用
curl -s http://app:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Jaeger 组件内存
docker stats jaeger-collector jaeger-query jaeger-agent

# 系统内存使用
free -h
cat /proc/meminfo
```

### 应急响应流程

#### 紧急情况处理

```bash
#!/bin/bash
# emergency-response.sh

ISSUE_TYPE=$1

case $ISSUE_TYPE in
  "high_load")
    echo "处理高负载情况..."
    # 降低采样率
    kubectl patch configmap jaeger-config -p '{"data":{"sampling.json":"{\"default_strategy\":{\"type\":\"probabilistic\",\"param\":0.001}}"}}'
    
    # 扩容 Collector
    kubectl scale deployment jaeger-collector --replicas=5
    ;;
    
  "storage_full")
    echo "处理存储空间不足..."
    # 紧急清理旧数据
    curl -X DELETE "elasticsearch:9200/jaeger-span-$(date -d '7 days ago' +%Y-%m-%d)"
    
    # 临时禁用追踪
    kubectl set env deployment/app SWIT_TRACING_ENABLED=false
    ;;
    
  "collector_down")
    echo "处理 Collector 服务故障..."
    # 重启 Collector
    kubectl rollout restart deployment jaeger-collector
    
    # 启用本地缓存
    kubectl set env deployment/app JAEGER_REPORTER_MAX_QUEUE_SIZE=10000
    ;;
esac
```

## 容量规划

### 存储容量计算

#### Span 大小估算

```text
单个 Span 平均大小计算：
- 基础元数据: ~200 bytes
- 服务名称: ~20 bytes  
- 操作名称: ~30 bytes
- 标签 (10个): ~500 bytes
- 日志 (3个): ~300 bytes
- 时间戳等: ~50 bytes

总计: ~1.1 KB/span
```

#### 容量规划表

| 服务数量 | 每秒请求数 | 平均 Spans/请求 | 采样率 | 每日数据量 | 90天存储 |
|----------|------------|-----------------|--------|------------|-----------|
| 10       | 1,000      | 8               | 1%     | 6.6 GB     | 594 GB    |
| 50       | 5,000      | 8               | 1%     | 33 GB      | 2.97 TB   |
| 100      | 10,000     | 8               | 1%     | 66 GB      | 5.94 TB   |
| 200      | 20,000     | 8               | 1%     | 132 GB     | 11.88 TB  |

#### 硬件推荐

```yaml
# 小规模部署 (< 1000 TPS)
small_deployment:
  elasticsearch:
    nodes: 3
    cpu: "2 cores"
    memory: "8 GB"
    storage: "500 GB SSD"
  
  jaeger_collector:
    replicas: 2
    cpu: "1 core"
    memory: "2 GB"

# 中等规模部署 (1000-10000 TPS)  
medium_deployment:
  elasticsearch:
    master_nodes: 3
    data_nodes: 6
    cpu: "4 cores"
    memory: "16 GB"  
    storage: "2 TB SSD"
  
  jaeger_collector:
    replicas: 4
    cpu: "2 cores"
    memory: "4 GB"

# 大规模部署 (> 10000 TPS)
large_deployment:
  elasticsearch:
    master_nodes: 3
    data_nodes: 12
    cpu: "8 cores"
    memory: "32 GB"
    storage: "4 TB NVMe SSD"
  
  jaeger_collector:
    replicas: 8
    cpu: "4 cores" 
    memory: "8 GB"
```

### 扩展策略

#### 自动扩缩容

```yaml
# HorizontalPodAutoscaler
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: jaeger-collector-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: jaeger-collector
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Pods
    pods:
      metric:
        name: jaeger_collector_queue_length
      target:
        type: AverageValue
        averageValue: "1000"
```

## 安全配置

### 认证和授权

#### Jaeger UI 访问控制

```yaml
# oauth2-proxy 配置
apiVersion: apps/v1
kind: Deployment
metadata:
  name: oauth2-proxy
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: oauth2-proxy
        image: quay.io/oauth2-proxy/oauth2-proxy:latest
        args:
          - --provider=oidc
          - --oidc-issuer-url=https://auth.company.com
          - --client-id=${CLIENT_ID}
          - --client-secret=${CLIENT_SECRET}
          - --cookie-secret=${COOKIE_SECRET}
          - --upstream=http://jaeger-query:16686
          - --email-domain=company.com
          - --pass-authorization-header=true
        ports:
        - containerPort: 4180
```

#### mTLS 配置

```yaml
# Jaeger Collector TLS 配置
collector:
  tls:
    cert: /certs/collector.crt
    key: /certs/collector.key
    client_ca: /certs/ca.crt
    
agent:
  tls:
    cert: /certs/agent.crt
    key: /certs/agent.key
    ca: /certs/ca.crt
    server_name: jaeger-collector
```

### 数据加密

#### 传输加密

```yaml
# Elasticsearch TLS 配置
elasticsearch:
  xpack:
    security:
      enabled: true
      transport.ssl.enabled: true
      transport.ssl.verification_mode: certificate
      transport.ssl.key: /certs/elasticsearch.key
      transport.ssl.certificate: /certs/elasticsearch.crt
      transport.ssl.certificate_authorities: [ "/certs/ca.crt" ]
```

#### 敏感数据处理

```go
// 敏感数据过滤器
type SensitiveDataProcessor struct{}

func (p *SensitiveDataProcessor) OnEnd(s trace.ReadOnlySpan) {
    // 检查敏感属性
    for _, attr := range s.Attributes() {
        key := string(attr.Key)
        
        // 过滤敏感信息
        if isSensitive(key) {
            // 记录警告但不存储敏感数据
            log.Warnf("敏感属性被过滤: %s", key)
        }
    }
}

func isSensitive(key string) bool {
    sensitivePatterns := []string{
        "password", "token", "key", "secret",
        "credit_card", "ssn", "email",
    }
    
    lowerKey := strings.ToLower(key)
    for _, pattern := range sensitivePatterns {
        if strings.Contains(lowerKey, pattern) {
            return true
        }
    }
    return false
}
```

## 维护操作

### 日常维护任务

#### 数据清理

```bash
#!/bin/bash
# daily-cleanup.sh

ES_HOST="elasticsearch:9200"
RETENTION_DAYS=90

# 计算清理日期
CLEANUP_DATE=$(date -d "${RETENTION_DAYS} days ago" +%Y.%m.%d)

echo "开始清理 ${CLEANUP_DATE} 之前的数据..."

# 获取要删除的索引
OLD_INDICES=$(curl -s "${ES_HOST}/_cat/indices/jaeger-*-${CLEANUP_DATE}*" | awk '{print $3}')

if [ ! -z "$OLD_INDICES" ]; then
    for index in $OLD_INDICES; do
        echo "删除索引: $index"
        curl -X DELETE "${ES_HOST}/${index}"
    done
else
    echo "没有找到需要清理的索引"
fi

# 优化索引
echo "优化索引..."
curl -X POST "${ES_HOST}/jaeger-*/_forcemerge?max_num_segments=1"

echo "清理完成"
```

#### 健康检查

```bash
#!/bin/bash
# health-check.sh

check_jaeger_health() {
    local component=$1
    local endpoint=$2
    
    echo "检查 ${component}..."
    
    if curl -f -s "${endpoint}" > /dev/null; then
        echo "✅ ${component} 健康"
        return 0
    else
        echo "❌ ${component} 异常"
        return 1
    fi
}

# 检查各组件
check_jaeger_health "Collector" "http://jaeger-collector:14269/"
check_jaeger_health "Query" "http://jaeger-query:16687/"  
check_jaeger_health "Agent" "http://jaeger-agent:14271/"
check_jaeger_health "Elasticsearch" "http://elasticsearch:9200/_cluster/health"

# 检查服务可达性
echo "检查服务可达性..."
services=("order-service:8080" "payment-service:8080" "inventory-service:8080")

for service in "${services[@]}"; do
    if nc -z ${service/:/ }; then
        echo "✅ ${service} 可达"
    else
        echo "❌ ${service} 不可达"
    fi
done
```

### 备份和恢复

#### Elasticsearch 快照备份

```bash
#!/bin/bash
# backup.sh

ES_HOST="elasticsearch:9200"
BACKUP_REPO="jaeger_backup"
SNAPSHOT_NAME="jaeger_$(date +%Y%m%d_%H%M%S)"

# 创建快照
echo "创建快照: ${SNAPSHOT_NAME}"
curl -X PUT "${ES_HOST}/_snapshot/${BACKUP_REPO}/${SNAPSHOT_NAME}" \
  -H "Content-Type: application/json" \
  -d '{
    "indices": "jaeger-*",
    "ignore_unavailable": true,
    "include_global_state": false,
    "metadata": {
      "taken_by": "jaeger-backup-script",
      "taken_because": "scheduled backup"
    }
  }'

# 验证快照
curl -s "${ES_HOST}/_snapshot/${BACKUP_REPO}/${SNAPSHOT_NAME}" | jq '.snapshots[0].state'
```

#### 配置备份

```bash
#!/bin/bash
# config-backup.sh

BACKUP_DIR="/backup/jaeger-config/$(date +%Y%m%d)"
mkdir -p "$BACKUP_DIR"

# 备份 Kubernetes 配置
kubectl get configmap jaeger-config -o yaml > "$BACKUP_DIR/jaeger-config.yaml"
kubectl get secret jaeger-secrets -o yaml > "$BACKUP_DIR/jaeger-secrets.yaml"

# 备份 Docker Compose 配置
cp docker-compose.prod.yml "$BACKUP_DIR/"
cp .env.production "$BACKUP_DIR/"

# 备份监控配置
cp prometheus.yml "$BACKUP_DIR/"
cp jaeger-alerts.yml "$BACKUP_DIR/"

echo "配置备份完成: $BACKUP_DIR"
```

### 升级策略

#### 滚动升级

```bash
#!/bin/bash
# rolling-upgrade.sh

JAEGER_VERSION=$1

if [ -z "$JAEGER_VERSION" ]; then
    echo "用法: $0 <jaeger_version>"
    exit 1
fi

echo "开始升级到 Jaeger ${JAEGER_VERSION}..."

# 1. 升级 Agent (DaemonSet)
echo "升级 Jaeger Agent..."
kubectl set image daemonset/jaeger-agent jaeger-agent=jaegertracing/jaeger-agent:${JAEGER_VERSION}
kubectl rollout status daemonset/jaeger-agent --timeout=300s

# 2. 升级 Collector
echo "升级 Jaeger Collector..."
kubectl set image deployment/jaeger-collector jaeger-collector=jaegertracing/jaeger-collector:${JAEGER_VERSION}
kubectl rollout status deployment/jaeger-collector --timeout=300s

# 3. 升级 Query
echo "升级 Jaeger Query..."
kubectl set image deployment/jaeger-query jaeger-query=jaegertracing/jaeger-query:${JAEGER_VERSION}
kubectl rollout status deployment/jaeger-query --timeout=300s

echo "升级完成"

# 验证升级
kubectl get pods -l app=jaeger
```

## 相关资源

- [用户指南](tracing-user-guide.md) - 快速开始和基础配置
- [开发者指南](developer-guide.md) - 深入集成和开发指南
- [架构文档](architecture.md) - 系统架构和设计原理
- [示例项目](../examples/distributed-tracing/) - 完整的使用示例
- [Jaeger 官方运维文档](https://www.jaegertracing.io/docs/deployment/)
- [Elasticsearch 运维指南](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html)

---

如需运维支持或遇到紧急故障，请联系运维团队或通过监控告警系统获取帮助。
