# 分布式追踪架构设计文档

本文档详细描述 SWIT 微服务框架分布式追踪系统的架构设计、组件交互关系、数据流向和扩展性考虑。

## 目录

- [架构概览](#架构概览)
- [核心组件](#核心组件)
- [数据流分析](#数据流分析)
- [集成架构](#集成架构)
- [存储架构](#存储架构)
- [网络架构](#网络架构)
- [安全架构](#安全架构)
- [扩展性设计](#扩展性设计)
- [技术选型](#技术选型)

## 架构概览

### 整体架构图

```text
                    ┌─────────────────────────────────────────────────────────┐
                    │                    用户层                                │
                    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │
                    │  │ 开发者工具   │  │   Jaeger UI │  │  监控仪表板      │   │
                    │  └─────────────┘  └─────────────┘  └─────────────────┘   │
                    └─────────────────────────────────────────────────────────┘
                                                │
                    ┌─────────────────────────────────────────────────────────┐
                    │                   查询层                                │
                    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │
                    │  │ Jaeger      │  │   REST      │  │    gRPC         │   │
                    │  │ Query       │  │   API       │  │   Gateway       │   │
                    │  └─────────────┘  └─────────────┘  └─────────────────┘   │
                    └─────────────────────────────────────────────────────────┘
                                                │
                    ┌─────────────────────────────────────────────────────────┐
                    │                   存储层                                │
                    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │
                    │  │ Elasticsearch│  │  Cassandra  │  │    Memory       │   │
                    │  │   Cluster    │  │   Cluster   │  │   Storage       │   │
                    │  └─────────────┘  └─────────────┘  └─────────────────┘   │
                    └─────────────────────────────────────────────────────────┘
                                                │
                    ┌─────────────────────────────────────────────────────────┐
                    │                  收集层                                 │
                    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │
                    │  │ Jaeger      │  │  Load       │  │   Batch         │   │
                    │  │ Collector   │  │ Balancer    │  │  Processor      │   │
                    │  └─────────────┘  └─────────────┘  └─────────────────┘   │
                    └─────────────────────────────────────────────────────────┘
                                                │
                    ┌─────────────────────────────────────────────────────────┐
                    │                  代理层                                 │
                    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │
                    │  │ Jaeger      │  │   OTEL      │  │    Sidecar      │   │
                    │  │ Agent       │  │ Collector   │  │   Proxy         │   │
                    │  └─────────────┘  └─────────────┘  └─────────────────┘   │
                    └─────────────────────────────────────────────────────────┘
                                                │
                    ┌─────────────────────────────────────────────────────────┐
                    │                 应用层                                  │
                    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │
                    │  │ Order       │  │  Payment    │  │   Inventory     │   │
                    │  │ Service     │  │  Service    │  │   Service       │   │
                    │  └─────────────┘  └─────────────┘  └─────────────────┘   │
                    └─────────────────────────────────────────────────────────┘
                                                │
                    ┌─────────────────────────────────────────────────────────┐
                    │                框架层                                   │
                    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │
                    │  │ SWIT        │  │ OpenTelemetry│  │   Instrumentation│   │
                    │  │ Framework   │  │    SDK      │  │    Libraries     │   │
                    │  └─────────────┘  └─────────────┘  └─────────────────┘   │
                    └─────────────────────────────────────────────────────────┘
```

### 关键架构特性

- **分层设计**: 清晰的分层架构，每层职责明确
- **模块化**: 组件松耦合，支持独立部署和扩展
- **标准化**: 基于 OpenTelemetry 标准，确保互操作性
- **高可用**: 多实例部署，支持故障转移
- **可伸缩**: 支持水平和垂直扩展
- **可观测**: 全链路追踪和监控

## 核心组件

### SWIT 框架追踪模块

#### 组件结构

```text
pkg/server/tracing/
├── config.go           # 追踪配置管理
├── provider.go         # TracerProvider 创建和管理
├── exporter.go         # 导出器抽象和实现
├── processor.go        # Span 处理器
├── sampler.go          # 采样策略
├── instrumentation.go  # 自动仪表化
└── middleware.go       # 中间件集成
```

#### 核心接口定义

```go
// 追踪管理器接口
type TracingManager interface {
    // 初始化追踪系统
    Initialize(ctx context.Context, config *TracingConfig) error
    
    // 获取 Tracer 实例
    GetTracer(name string) trace.Tracer
    
    // 获取性能指标
    GetMetrics() *TracingMetrics
    
    // 关闭追踪系统
    Shutdown(ctx context.Context) error
}

// 导出器工厂接口
type ExporterFactory interface {
    CreateExporter(config ExporterConfig) (trace.SpanExporter, error)
    GetSupportedTypes() []string
}

// 采样策略接口
type SamplingStrategy interface {
    ShouldSample(p trace.SamplingParameters) trace.SamplingResult
    Description() string
    Update(config SamplingConfig) error
}
```

#### 组件交互图

```text
                    ┌─────────────────┐
                    │ BusinessServer  │
                    │     Core        │
                    └─────────────────┘
                             │
                             │ initialize
                             ▼
                    ┌─────────────────┐
                    │ TracingManager  │◄────┐
                    └─────────────────┘     │
                             │              │
                             │ creates      │
                             ▼              │
                    ┌─────────────────┐     │
                    │ TracerProvider  │     │ configure
                    └─────────────────┘     │
                             │              │
                    ┌────────┼──────────┐   │
                    │        │          │   │
                    ▼        ▼          ▼   │
            ┌─────────┐ ┌─────────┐ ┌─────────┐
            │Exporter │ │Processor│ │ Sampler │─┘
            └─────────┘ └─────────┘ └─────────┘
```

### OpenTelemetry 集成

#### SDK 组件

```go
// TracerProvider 配置
type TracerProviderConfig struct {
    // 资源配置
    Resource *resource.Resource
    
    // Span 处理器
    SpanProcessors []trace.SpanProcessor
    
    // 采样器
    Sampler trace.Sampler
    
    // ID 生成器
    IDGenerator trace.IDGenerator
    
    // Span 限制
    SpanLimits trace.SpanLimits
}

// 导出器配置
type ExporterConfig struct {
    Type     string        // jaeger, otlp, zipkin, console
    Endpoint string        // 导出端点
    Timeout  time.Duration // 超时时间
    Headers  map[string]string // HTTP 头
    
    // 批处理配置
    BatchTimeout         time.Duration
    MaxExportBatchSize   int
    MaxQueueSize         int
    ExportTimeout        time.Duration
}
```

#### 仪表化层次

```text
Application Code
       │
       ▼
┌─────────────────┐
│ Manual Spans    │ (用户手动创建)
└─────────────────┘
       │
       ▼
┌─────────────────┐
│ Auto Instrument │ (自动仪表化)
│ - HTTP Client   │
│ - HTTP Server   │
│ - gRPC         │
│ - Database     │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│ OpenTelemetry   │ (SDK 层)
│ SDK             │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│ Exporters       │ (导出层)
└─────────────────┘
```

### Jaeger 组件架构

#### Jaeger 生态系统

```text
┌─────────────────────────────────────────────────────────────────┐
│                        Jaeger 生态系统                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────────────┐    │
│  │   Jaeger    │   │   Jaeger    │   │      Jaeger         │    │
│  │   Agent     │   │ Collector   │   │      Query          │    │
│  │             │   │             │   │                     │    │
│  │ • 接收追踪   │   │ • 验证数据   │   │ • 查询接口          │    │
│  │ • 批处理     │   │ • 转换格式   │   │ • Web UI           │    │
│  │ • 本地缓存   │   │ • 存储写入   │   │ • API 服务         │    │
│  │ • 负载均衡   │   │ • 采样策略   │   │ • 聚合分析          │    │
│  └─────────────┘   └─────────────┘   └─────────────────────┘    │
│         │                  │                     │              │
│         │                  │                     │              │
│  ┌──────▼──────────────────▼─────────────────────▼──────────────┤
│  │                    存储后端                                  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐   │
│  │  │Elasticsearch│  │  Cassandra  │  │      Memory         │   │
│  │  └─────────────┘  └─────────────┘  └─────────────────────┘   │
│  └─────────────────────────────────────────────────────────────┤
└─────────────────────────────────────────────────────────────────┘
```

#### 数据模型

```go
// Jaeger Span 数据模型
type Span struct {
    TraceID       TraceID           `json:"traceID"`
    SpanID        SpanID            `json:"spanID"`
    ParentSpanID  SpanID            `json:"parentSpanID,omitempty"`
    OperationName string            `json:"operationName"`
    StartTime     uint64            `json:"startTime"`
    Duration      uint64            `json:"duration"`
    Tags          []KeyValue        `json:"tags"`
    Logs          []Log             `json:"logs"`
    Process       Process           `json:"process"`
    References    []SpanRef         `json:"references,omitempty"`
}

type Process struct {
    ServiceName string     `json:"serviceName"`
    Tags        []KeyValue `json:"tags"`
}

type Log struct {
    Timestamp uint64     `json:"timestamp"`
    Fields    []KeyValue `json:"fields"`
}
```

## 数据流分析

### 追踪数据生成流程

```text
┌─────────────────┐
│  应用服务        │
│                │
│ 1. 创建 Span    │
│ 2. 设置属性     │
│ 3. 记录事件     │
│ 4. 结束 Span    │
└─────────────────┘
         │
         │ OpenTelemetry SDK
         ▼
┌─────────────────┐
│  Span 处理器     │
│                │
│ 1. 属性验证     │
│ 2. 采样决策     │
│ 3. 数据转换     │
│ 4. 批处理队列   │
└─────────────────┘
         │
         │ gRPC/HTTP
         ▼
┌─────────────────┐
│  Jaeger Agent   │
│                │
│ 1. 接收数据     │
│ 2. 本地缓存     │
│ 3. 批量转发     │
│ 4. 负载均衡     │
└─────────────────┘
         │
         │ Jaeger Binary Protocol
         ▼
┌─────────────────┐
│ Jaeger Collector│
│                │
│ 1. 数据验证     │
│ 2. 格式转换     │
│ 3. 存储写入     │
│ 4. 索引构建     │
└─────────────────┘
         │
         │ Elasticsearch/Cassandra API
         ▼
┌─────────────────┐
│   存储后端      │
│                │
│ 1. 数据持久化   │
│ 2. 索引管理     │
│ 3. 查询优化     │
│ 4. 数据压缩     │
└─────────────────┘
```

### 查询数据流程

```text
┌─────────────────┐
│   用户界面       │
│                │
│ 1. 查询请求     │
│ 2. 参数设置     │
│ 3. 结果展示     │
│ 4. 分析操作     │
└─────────────────┘
         │
         │ HTTP/REST API
         ▼
┌─────────────────┐
│  Jaeger Query   │
│                │
│ 1. 请求解析     │
│ 2. 查询构建     │
│ 3. 结果聚合     │
│ 4. 数据转换     │
└─────────────────┘
         │
         │ 存储查询 API
         ▼
┌─────────────────┐
│   存储后端       │
│                │
│ 1. 索引查找     │
│ 2. 数据检索     │
│ 3. 排序分页     │
│ 4. 结果返回     │
└─────────────────┘
         │
         │ 查询结果
         ▼
┌─────────────────┐
│   查询引擎      │
│                │
│ 1. 数据关联     │
│ 2. 时间序列     │
│ 3. 统计分析     │
│ 4. 可视化处理   │
└─────────────────┘
```

### 上下文传播机制

```text
Service A (HTTP)              Service B (gRPC)
┌─────────────────┐          ┌─────────────────┐
│     Request     │          │     Request     │
│                 │          │                 │
│ TraceID: abc123 │   HTTP   │ TraceID: abc123 │
│ SpanID:  def456 │   ────►  │ SpanID:  ghi789 │
│ Parent:  -      │          │ Parent:  def456 │
└─────────────────┘          └─────────────────┘
         │                            │
         │                            │
         ▼                            ▼
┌─────────────────┐          ┌─────────────────┐
│ OpenTelemetry   │          │ OpenTelemetry   │
│ Context         │          │ Context         │
│                 │          │                 │
│ • Trace Context │          │ • Trace Context │
│ • Baggage       │          │ • Baggage       │
│ • Correlation   │          │ • Correlation   │
└─────────────────┘          └─────────────────┘
```

## 集成架构

### SWIT 框架集成

#### 框架层次集成

```go
// 框架集成点
type BusinessServer struct {
    config          *ServerConfig
    tracingManager  TracingManager
    httpServer      *HTTPServer
    grpcServer      *GRPCServer
    registry        ServiceRegistry
}

func (s *BusinessServer) Start(ctx context.Context) error {
    // 1. 初始化追踪系统
    if err := s.initializeTracing(ctx); err != nil {
        return err
    }
    
    // 2. 设置中间件
    s.setupTracingMiddleware()
    
    // 3. 启动服务器
    return s.startServers(ctx)
}

func (s *BusinessServer) initializeTracing(ctx context.Context) error {
    if !s.config.Tracing.Enabled {
        return nil
    }
    
    // 创建追踪管理器
    tm, err := NewTracingManager(s.config.Tracing)
    if err != nil {
        return err
    }
    
    // 初始化追踪系统
    if err := tm.Initialize(ctx, &s.config.Tracing); err != nil {
        return err
    }
    
    s.tracingManager = tm
    return nil
}
```

#### 中间件集成架构

```text
                    ┌─────────────────┐
                    │   HTTP Request  │
                    └─────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Tracing         │
                    │ Middleware      │
                    └─────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Business        │
                    │ Handler         │
                    └─────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Service Layer   │
                    └─────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Data Layer      │
                    │ (Database)      │
                    └─────────────────┘
```

### 服务发现集成

#### 服务注册与追踪

```go
type TracedServiceRegistry struct {
    registry ServiceRegistry
    tracer   trace.Tracer
}

func (r *TracedServiceRegistry) RegisterService(ctx context.Context, info *ServiceInfo) error {
    ctx, span := r.tracer.Start(ctx, "RegisterService")
    defer span.End()
    
    span.SetAttribute(attribute.String("service.name", info.Name))
    span.SetAttribute(attribute.String("service.version", info.Version))
    
    err := r.registry.RegisterService(ctx, info)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }
    
    return err
}

func (r *TracedServiceRegistry) DiscoverService(ctx context.Context, serviceName string) (*ServiceInfo, error) {
    ctx, span := r.tracer.Start(ctx, "DiscoverService")
    defer span.End()
    
    span.SetAttribute(attribute.String("target.service", serviceName))
    
    info, err := r.registry.DiscoverService(ctx, serviceName)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    
    span.SetAttribute(attribute.String("discovered.endpoint", info.Endpoint))
    return info, nil
}
```

## 存储架构

### Elasticsearch 存储设计

#### 索引策略

```yaml
# 索引模板
{
  "index_patterns": ["jaeger-span-*"],
  "settings": {
    "number_of_shards": 3,
    "number_of_replicas": 1,
    "refresh_interval": "30s",
    
    # 索引生命周期管理
    "lifecycle": {
      "name": "jaeger-policy",
      "rollover_alias": "jaeger-span-write"
    }
  },
  "mappings": {
    "properties": {
      "traceID": {
        "type": "keyword",
        "index": true
      },
      "spanID": {
        "type": "keyword",
        "index": true
      },
      "parentSpanID": {
        "type": "keyword",
        "index": false
      },
      "operationName": {
        "type": "keyword",
        "index": true
      },
      "startTime": {
        "type": "date",
        "format": "epoch_micros"
      },
      "duration": {
        "type": "long"
      },
      "tags": {
        "type": "nested",
        "properties": {
          "key": {"type": "keyword"},
          "value": {"type": "keyword"}
        }
      },
      "process": {
        "properties": {
          "serviceName": {
            "type": "keyword",
            "index": true
          }
        }
      }
    }
  }
}
```

#### 数据分片策略

```text
时间分片策略:
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│ jaeger-span-    │  │ jaeger-span-    │  │ jaeger-span-    │
│ 2024.01.01      │  │ 2024.01.02      │  │ 2024.01.03      │
│                 │  │                 │  │                 │
│ Shard 0: Hot    │  │ Shard 0: Warm   │  │ Shard 0: Cold   │
│ Shard 1: Hot    │  │ Shard 1: Warm   │  │ Shard 1: Cold   │
│ Shard 2: Hot    │  │ Shard 2: Warm   │  │ Shard 2: Cold   │
└─────────────────┘  └─────────────────┘  └─────────────────┘

服务分片策略 (可选):
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│ jaeger-span-    │  │ jaeger-span-    │  │ jaeger-span-    │
│ order-service   │  │ payment-service │  │inventory-service│
│                 │  │                 │  │                 │
│ 高频查询服务     │  │ 中频查询服务    │  │ 低频查询服务    │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

### Cassandra 存储设计

#### 表设计

```cql
-- 主追踪表
CREATE TABLE jaeger_v1_dc1.traces (
    trace_id blob,
    span_id bigint,
    parent_id bigint,
    operation_name text,
    start_time timestamp,
    duration bigint,
    flags int,
    logs frozen<list<frozen<map<text, text>>>>,
    refs frozen<list<frozen<map<text, text>>>>,
    tags frozen<map<text, text>>,
    service_name text,
    PRIMARY KEY (trace_id, span_id)
) WITH CLUSTERING ORDER BY (span_id ASC);

-- 服务索引表
CREATE TABLE jaeger_v1_dc1.service_names (
    service_name text,
    PRIMARY KEY (service_name)
);

-- 操作索引表  
CREATE TABLE jaeger_v1_dc1.operation_names (
    service_name text,
    operation_name text,
    PRIMARY KEY (service_name, operation_name)
);

-- 时间索引表
CREATE TABLE jaeger_v1_dc1.service_operation_index (
    service_name text,
    operation_name text,
    start_time timestamp,
    trace_id blob,
    PRIMARY KEY ((service_name, operation_name), start_time, trace_id)
) WITH CLUSTERING ORDER BY (start_time DESC);
```

#### 查询模式

```cql
-- 按时间范围查询追踪
SELECT trace_id FROM service_operation_index 
WHERE service_name = 'order-service' 
  AND operation_name = 'CreateOrder'
  AND start_time >= '2024-01-01 00:00:00'
  AND start_time <= '2024-01-01 23:59:59'
ORDER BY start_time DESC
LIMIT 100;

-- 获取完整追踪数据
SELECT * FROM traces 
WHERE trace_id = 0x1234567890abcdef;

-- 按标签查询 (需要二级索引)
CREATE INDEX ON traces (tags);
SELECT trace_id FROM traces 
WHERE tags CONTAINS KEY 'user.id'
  AND tags['user.id'] = '12345';
```

## 网络架构

### 通信协议

#### 协议栈

```text
                应用层
    ┌─────────────────────────────┐
    │   HTTP/REST   │   gRPC      │
    └─────────────────────────────┘
                传输层  
    ┌─────────────────────────────┐
    │   Jaeger      │   OTLP      │
    │   Protocol    │  Protocol   │
    └─────────────────────────────┘
                网络层
    ┌─────────────────────────────┐
    │       TCP/UDP               │
    └─────────────────────────────┘
```

#### 端口分配

```yaml
# Jaeger 组件端口
jaeger_ports:
  agent:
    compact: 6831     # Jaeger thrift compact (UDP)
    binary: 6832      # Jaeger thrift binary (UDP)
    config: 5778      # Config server (HTTP)
    
  collector:
    jaeger: 14268     # Jaeger thrift (HTTP)
    grpc: 14250       # Jaeger gRPC
    zipkin: 9411      # Zipkin HTTP
    otlp_grpc: 4317   # OTLP gRPC
    otlp_http: 4318   # OTLP HTTP
    metrics: 14269    # Metrics (HTTP)
    
  query:
    http: 16686       # Query service (HTTP)
    grpc: 16685       # Query service (gRPC)
    metrics: 16687    # Metrics (HTTP)

# 应用服务端口
application_ports:
  order_service:
    http: 8080
    grpc: 9080
    metrics: 8090
    health: 8099
```

### 负载均衡策略

#### Collector 负载均衡

```yaml
# HAProxy 配置
global:
  maxconn 4096
  log stdout local0

defaults:
  mode http
  timeout connect 5s
  timeout client 10s
  timeout server 10s

frontend jaeger_collector_frontend:
  bind *:14268
  default_backend jaeger_collectors

backend jaeger_collectors:
  balance roundrobin
  option httpchk GET /
  server collector1 jaeger-collector-1:14268 check
  server collector2 jaeger-collector-2:14268 check
  server collector3 jaeger-collector-3:14268 check

frontend jaeger_query_frontend:
  bind *:16686
  default_backend jaeger_queries

backend jaeger_queries:
  balance leastconn
  option httpchk GET /
  server query1 jaeger-query-1:16686 check
  server query2 jaeger-query-2:16686 check
```

## 安全架构

SWIT 框架提供多层次的安全架构，涵盖身份认证、访问授权、传输安全和数据保护。

### 安全组件架构

```text
┌─────────────────────────────────────────────────────────────────────┐
│                        SWIT 安全框架                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌───────────────┐   ┌───────────────┐   ┌───────────────────────┐  │
│  │   OAuth2/OIDC │   │   OPA 策略    │   │      TLS/mTLS        │  │
│  │   认证客户端   │   │   引擎        │   │    传输安全           │  │
│  │               │   │               │   │                       │  │
│  │ • Keycloak    │   │ • RBAC 策略   │   │ • 证书管理            │  │
│  │ • Auth0       │   │ • ABAC 策略   │   │ • 双向认证            │  │
│  │ • Google      │   │ • 自定义策略   │   │ • 密码套件配置        │  │
│  │ • Microsoft   │   │ • 策略缓存    │   │ • 版本控制            │  │
│  │ • Okta        │   │ • 热重载      │   │                       │  │
│  └───────────────┘   └───────────────┘   └───────────────────────┘  │
│           │                   │                      │              │
│           ▼                   ▼                      ▼              │
│  ┌───────────────┐   ┌───────────────┐   ┌───────────────────────┐  │
│  │   JWT 验证    │   │   审计日志    │   │     密钥管理          │  │
│  │               │   │               │   │                       │  │
│  │ • JWKS 缓存   │   │ • 事件记录    │   │ • 环境变量            │  │
│  │ • 声明验证    │   │ • 脱敏处理    │   │ • 文件提供者          │  │
│  │ • 黑名单管理  │   │ • 多输出      │   │ • Vault 集成          │  │
│  └───────────────┘   └───────────────┘   └───────────────────────┘  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 认证授权流程

```text
                    ┌─────────────────┐
                    │   HTTP 请求     │
                    └─────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ OAuth2 中间件   │
                    │  令牌提取/验证   │
                    └─────────────────┘
                             │
                    ┌────────┴────────┐
                    │                 │
              验证成功            验证失败
                    │                 │
                    ▼                 ▼
           ┌─────────────────┐  ┌─────────────────┐
           │ OPA 授权中间件  │  │  401 未授权     │
           │  策略评估       │  └─────────────────┘
           └─────────────────┘
                    │
           ┌────────┴────────┐
           │                 │
        授权通过          授权拒绝
           │                 │
           ▼                 ▼
   ┌─────────────────┐  ┌─────────────────┐
   │  业务处理器     │  │  403 禁止访问   │
   └─────────────────┘  └─────────────────┘
```

### 多层安全架构

```text
┌─────────────────────────────────────────────────────────┐
│                     用户层                               │
│ ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │
│ │   OAuth2    │  │    LDAP     │  │      RBAC       │   │
│ │Integration  │  │Integration  │  │  Authorization  │   │
│ └─────────────┘  └─────────────┘  └─────────────────┘   │
└─────────────────────────────────────────────────────────┘
                             │
┌─────────────────────────────────────────────────────────┐
│                   网关层                                │
│ ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │
│ │    API      │  │   mTLS      │  │    Rate         │   │
│ │  Gateway    │  │Certificate  │  │   Limiting      │   │
│ └─────────────┘  └─────────────┘  └─────────────────┘   │
└─────────────────────────────────────────────────────────┘
                             │
┌─────────────────────────────────────────────────────────┐
│                  传输层                                 │
│ ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │
│ │    TLS      │  │  Network    │  │   Firewall      │   │
│ │Encryption   │  │ Policies    │  │    Rules        │   │
│ └─────────────┘  └─────────────┘  └─────────────────┘   │
└─────────────────────────────────────────────────────────┘
                             │
┌─────────────────────────────────────────────────────────┐
│                  存储层                                 │
│ ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │
│ │  Encryption │  │   Access    │  │    Audit        │   │
│ │  at Rest    │  │  Control    │  │   Logging       │   │
│ └─────────────┘  └─────────────┘  └─────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### 安全组件目录结构

```text
pkg/security/
├── oauth2/           # OAuth2/OIDC 认证
│   ├── client.go     # OAuth2 客户端
│   ├── config.go     # 配置管理
│   ├── provider.go   # 提供商抽象
│   ├── flows.go      # 授权流程
│   ├── token.go      # 令牌管理
│   └── cache.go      # 令牌缓存
├── opa/              # OPA 策略引擎
│   ├── client.go     # OPA 客户端
│   ├── config.go     # 配置管理
│   ├── evaluator.go  # 策略评估器
│   ├── manager.go    # 策略管理器
│   └── policies/     # 内置策略
│       ├── rbac.rego # RBAC 策略
│       └── abac.rego # ABAC 策略
├── jwt/              # JWT 处理
│   ├── validator.go  # 令牌验证
│   ├── claims.go     # 声明处理
│   ├── jwks_cache.go # JWKS 缓存
│   └── blacklist.go  # 黑名单管理
├── tls/              # TLS 配置
│   ├── config.go     # TLS 配置
│   └── certificates.go # 证书管理
├── audit/            # 审计日志
│   ├── logger.go     # 审计日志记录器
│   └── events.go     # 事件定义
├── secrets/          # 密钥管理
│   ├── manager.go    # 密钥管理器
│   ├── env_provider.go    # 环境变量提供者
│   ├── file_provider.go   # 文件提供者
│   └── vault_provider.go  # Vault 提供者
└── scanner/          # 安全扫描
    ├── gosec.go      # gosec 集成
    ├── trivy.go      # Trivy 集成
    └── govulncheck.go # govulncheck 集成
```

### 安全相关文档

- [安全最佳实践](security-best-practices.md) - 全面的安全指南
- [OAuth2 集成指南](oauth2-integration-guide.md) - 认证配置
- [OPA RBAC 指南](opa-rbac-guide.md) - 角色访问控制
- [OPA ABAC 指南](opa-abac-guide.md) - 属性访问控制
- [安全检查清单](security-checklist.md) - 部署前检查
- [安全配置参考](security-configuration-reference.md) - 配置选项

#### 权限模型

```go
// 权限配置
type SecurityConfig struct {
    Authentication AuthConfig     `yaml:"authentication"`
    Authorization  AuthzConfig   `yaml:"authorization"`
    Encryption     CryptoConfig  `yaml:"encryption"`
    Audit          AuditConfig   `yaml:"audit"`
}

type AuthConfig struct {
    Type     string            `yaml:"type"`      // oauth2, ldap, jwt
    Endpoint string            `yaml:"endpoint"`
    ClientID string            `yaml:"client_id"`
    Scopes   []string          `yaml:"scopes"`
}

type AuthzConfig struct {
    Type    string              `yaml:"type"`     // rbac, abac
    Roles   map[string][]string `yaml:"roles"`
    Policies []PolicyRule       `yaml:"policies"`
}

type PolicyRule struct {
    Effect    string   `yaml:"effect"`    // allow, deny
    Actions   []string `yaml:"actions"`   // read, write, delete
    Resources []string `yaml:"resources"` // traces, services, operations
    Subjects  []string `yaml:"subjects"`  // users, groups, services
}
```

### 数据保护

#### 敏感数据过滤

```go
// 数据脱敏处理器
type DataSanitizerProcessor struct {
    sensitiveKeys []string
    hashSalt      string
}

func (p *DataSanitizerProcessor) OnStart(parent context.Context, s trace.ReadWriteSpan) {
    // 检查 Span 属性
    for _, attr := range s.Attributes() {
        key := string(attr.Key)
        if p.isSensitive(key) {
            // 脱敏处理
            hashedValue := p.hashValue(attr.Value.AsString())
            s.SetAttribute(attribute.String(key+"_hash", hashedValue))
            
            // 删除原始敏感数据
            s.RemoveAttribute(attr.Key)
        }
    }
}

func (p *DataSanitizerProcessor) isSensitive(key string) bool {
    for _, sensitiveKey := range p.sensitiveKeys {
        if strings.Contains(strings.ToLower(key), sensitiveKey) {
            return true
        }
    }
    return false
}
```

## 扩展性设计

### 水平扩展

#### 组件扩展策略

```yaml
# Kubernetes 扩展配置
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
  maxReplicas: 20
  metrics:
  # CPU 使用率
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  # 内存使用率        
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  # 自定义指标 - 队列长度
  - type: Pods
    pods:
      metric:
        name: jaeger_collector_queue_length
      target:
        type: AverageValue
        averageValue: "1000"
  # 自定义指标 - Spans/sec
  - type: Pods
    pods:
      metric:
        name: jaeger_collector_spans_per_second
      target:
        type: AverageValue
        averageValue: "10000"
```

#### 存储扩展

```yaml
# Elasticsearch 集群扩展
cluster_scaling:
  master_nodes:
    min_replicas: 3
    max_replicas: 5
    scaling_policy: "manual"  # 主节点数量手动控制
    
  data_nodes:
    min_replicas: 3
    max_replicas: 20
    scaling_triggers:
      - metric: "elasticsearch_cluster_health_status"
        threshold: "yellow"
        action: "scale_up"
      - metric: "elasticsearch_jvm_memory_used_percent"
        threshold: 80
        action: "scale_up"
      - metric: "elasticsearch_search_query_time_seconds"
        threshold: 5
        action: "scale_up"
        
  ingest_nodes:
    min_replicas: 2
    max_replicas: 10
    scaling_triggers:
      - metric: "elasticsearch_indices_indexing_index_time_seconds"
        threshold: 3
        action: "scale_up"
```

### 垂直扩展

#### 资源配置优化

```yaml
# 资源配置模板
resource_profiles:
  small:
    jaeger_collector:
      cpu: "500m"
      memory: "1Gi" 
      jvm_heap: "512m"
      
    jaeger_query:
      cpu: "250m"
      memory: "512Mi"
      jvm_heap: "256m"
      
  medium:
    jaeger_collector:
      cpu: "2"
      memory: "4Gi"
      jvm_heap: "2g"
      
    jaeger_query:
      cpu: "1"
      memory: "2Gi" 
      jvm_heap: "1g"
      
  large:
    jaeger_collector:
      cpu: "4"
      memory: "8Gi"
      jvm_heap: "4g"
      
    jaeger_query:
      cpu: "2"
      memory: "4Gi"
      jvm_heap: "2g"
```

### 插件架构

#### 扩展点设计

```go
// 扩展点接口
type TracingExtension interface {
    Name() string
    Initialize(config map[string]interface{}) error
    Shutdown() error
}

// Span 处理扩展
type SpanProcessorExtension interface {
    TracingExtension
    ProcessSpan(span trace.ReadOnlySpan) error
}

// 导出器扩展
type ExporterExtension interface {
    TracingExtension
    CreateExporter(config ExporterConfig) (trace.SpanExporter, error)
}

// 采样器扩展
type SamplerExtension interface {
    TracingExtension
    CreateSampler(config SamplingConfig) (trace.Sampler, error)
}

// 扩展管理器
type ExtensionManager struct {
    extensions map[string]TracingExtension
    mutex      sync.RWMutex
}

func (em *ExtensionManager) RegisterExtension(ext TracingExtension) error {
    em.mutex.Lock()
    defer em.mutex.Unlock()
    
    name := ext.Name()
    if _, exists := em.extensions[name]; exists {
        return fmt.Errorf("extension %s already registered", name)
    }
    
    em.extensions[name] = ext
    return nil
}
```

## 技术选型

### 选型决策

#### OpenTelemetry vs. 其他方案

| 特性 | OpenTelemetry | Jaeger SDK | Zipkin | 自研方案 |
|------|---------------|------------|--------|-----------|
| **标准化** | ✅ CNCF 标准 | ❌ Jaeger 专用 | ❌ Zipkin 专用 | ❌ 非标准 |
| **生态系统** | ✅ 丰富 | ⚠️ 中等 | ⚠️ 中等 | ❌ 有限 |
| **语言支持** | ✅ 多语言 | ⚠️ 部分语言 | ⚠️ 部分语言 | ❌ 单语言 |
| **扩展性** | ✅ 高 | ⚠️ 中等 | ⚠️ 中等 | ✅ 高 |
| **性能** | ✅ 优化 | ✅ 优化 | ⚠️ 中等 | ❓ 未知 |
| **维护成本** | ✅ 低 | ⚠️ 中等 | ⚠️ 中等 | ❌ 高 |

**选择理由**: OpenTelemetry 是 CNCF 毕业项目，标准化程度高，生态系统丰富，长期维护有保障。

#### 存储后端选择

| 特性 | Elasticsearch | Cassandra | 云存储 | Memory |
|------|---------------|-----------|---------|---------|
| **查询灵活性** | ✅ 高 | ⚠️ 中等 | ✅ 高 | ✅ 高 |
| **写入性能** | ⚠️ 中等 | ✅ 高 | ⚠️ 中等 | ✅ 高 |
| **存储成本** | ⚠️ 中等 | ✅ 低 | ❌ 高 | ❌ 不持久 |
| **运维复杂度** | ⚠️ 中等 | ❌ 高 | ✅ 低 | ✅ 低 |
| **可扩展性** | ✅ 好 | ✅ 优秀 | ✅ 好 | ❌ 有限 |
| **数据分析** | ✅ 强大 | ⚠️ 基础 | ✅ 强大 | ⚠️ 基础 |

**选择策略**: 
- 开发/测试: Memory 存储
- 中小规模: Elasticsearch
- 大规模高并发: Cassandra
- 托管服务: 云存储

#### 部署方案对比

| 方案 | 复杂度 | 可扩展性 | 运维成本 | 适用场景 |
|------|---------|----------|----------|----------|
| **All-in-One** | ✅ 低 | ❌ 差 | ✅ 低 | 开发/测试 |
| **生产分布式** | ❌ 高 | ✅ 优秀 | ❌ 高 | 大规模生产 |
| **Kubernetes** | ⚠️ 中等 | ✅ 好 | ⚠️ 中等 | 容器化环境 |
| **云托管服务** | ✅ 低 | ✅ 好 | ⚠️ 中等 | 快速部署 |

### 架构演进路径

#### 阶段性演进

```bash
Phase 1: 基础架构
┌─────────────────┐
│ All-in-One      │
│ Jaeger          │
│ + Memory        │
│ Storage         │
└─────────────────┘

Phase 2: 生产就绪  
┌─────────────────┐
│ Jaeger          │
│ Components      │
│ + Elasticsearch │
│ Cluster         │
└─────────────────┘

Phase 3: 高可用
┌─────────────────┐
│ Load Balanced   │
│ Jaeger          │
│ + HA            │
│ Elasticsearch   │
└─────────────────┘

Phase 4: 云原生
┌─────────────────┐
│ Kubernetes      │
│ Operator        │
│ + Auto Scaling  │
│ + Multi-Cloud   │
└─────────────────┘
```

## 相关资源

- [用户指南](tracing-user-guide.md) - 快速开始和基础使用
- [开发者指南](developer-guide.md) - 深入集成和开发
- [运维指南](operations-guide.md) - 生产环境部署和监控
- [示例项目](../examples/distributed-tracing/) - 完整的实现示例
- [OpenTelemetry 架构](https://opentelemetry.io/docs/concepts/)
- [Jaeger 架构文档](https://www.jaegertracing.io/docs/architecture/)

---

本架构文档将持续更新，反映系统的演进和最佳实践的改进。如有架构相关问题，请联系架构团队。
