# Saga 监控指南

本指南详细介绍如何监控和观测 Saga 分布式事务系统，包括 Prometheus 指标收集、Grafana 可视化、Jaeger 分布式追踪以及告警配置。

## 目录

- [概述](#概述)
- [指标收集](#指标收集)
- [Prometheus 集成](#prometheus-集成)
- [Grafana 可视化](#grafana-可视化)
- [分布式追踪](#分布式追踪)
- [告警配置](#告警配置)
- [性能监控](#性能监控)
- [故障排查](#故障排查)
- [最佳实践](#最佳实践)

## 概述

### 为什么需要监控 Saga？

Saga 分布式事务涉及多个服务和步骤，监控对于以下方面至关重要：

- **可靠性** - 检测和诊断故障
- **性能** - 识别瓶颈和优化机会
- **可观测性** - 理解系统行为
- **SLA 保证** - 确保满足服务级别目标
- **容量规划** - 预测资源需求

### 监控架构

```
┌─────────────────┐
│  Saga Service   │
│                 │
│  ┌───────────┐  │
│  │ Metrics   │──┼─────────┐
│  │ Collector │  │         │
│  └───────────┘  │         │
│                 │         ▼
│  ┌───────────┐  │    ┌─────────────┐
│  │  Tracing  │──┼───▶│ Prometheus  │
│  │  Exporter │  │    └─────────────┘
│  └───────────┘  │         │
└─────────────────┘         │
                            ▼
        ┌─────────────┐    ┌─────────────┐
        │   Jaeger    │◀───│  Grafana    │
        └─────────────┘    └─────────────┘
                                 │
                                 ▼
                          ┌─────────────┐
                          │Alertmanager │
                          └─────────────┘
```

## 指标收集

### 核心指标

Saga 监控系统收集以下关键指标：

#### 1. 执行指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `saga_monitoring_saga_started_total` | Counter | Saga 启动总数 |
| `saga_monitoring_saga_completed_total` | Counter | Saga 成功完成总数 |
| `saga_monitoring_saga_failed_total` | Counter | Saga 失败总数（带 reason 标签） |
| `saga_monitoring_saga_duration_seconds` | Histogram | Saga 执行时长分布 |
| `saga_monitoring_active_sagas` | Gauge | 当前活跃的 Saga 数量 |

#### 2. 补偿指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `saga_compensation_executed_total` | Counter | 补偿操作执行总数 |
| `saga_compensation_failed_total` | Counter | 补偿操作失败总数 |
| `saga_compensation_duration_seconds` | Histogram | 补偿操作执行时长 |

#### 3. 消息系统指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `saga_messaging_publish_total` | Counter | 消息发布总数 |
| `saga_messaging_publish_failures_total` | Counter | 消息发布失败总数 |
| `saga_messaging_consume_total` | Counter | 消息消费总数 |
| `saga_messaging_consume_failures_total` | Counter | 消息消费失败总数 |
| `saga_messaging_queue_depth` | Gauge | 消息队列深度 |

#### 4. 存储指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `saga_storage_operations_total` | Counter | 存储操作总数 |
| `saga_storage_operation_duration_seconds` | Histogram | 存储操作延迟 |
| `saga_storage_connection_pool_size` | Gauge | 连接池大小 |
| `saga_storage_connection_pool_in_use` | Gauge | 正在使用的连接数 |

### 使用指标收集器

#### 基本用法

```go
package main

import (
    "github.com/swit/pkg/saga/monitoring"
)

func main() {
    // 创建默认配置
    config := monitoring.DefaultConfig()
    
    // 创建指标收集器
    collector, err := monitoring.NewSagaMetricsCollector(config)
    if err != nil {
        panic(err)
    }
    
    // 记录 Saga 启动
    collector.RecordSagaStarted("saga-123")
    
    // 记录 Saga 完成
    duration := time.Second * 5
    collector.RecordSagaCompleted("saga-123", duration)
    
    // 或记录 Saga 失败
    collector.RecordSagaFailed("saga-124", "timeout")
    
    // 获取指标快照
    metrics := collector.GetMetrics()
    fmt.Printf("Active Sagas: %d\n", metrics.ActiveSagas)
}
```

#### 高级配置

```go
// 自定义配置
config := &monitoring.Config{
    Namespace: "myapp",
    Subsystem: "saga",
    Registry:  prometheus.DefaultRegisterer,
    DurationBuckets: []float64{
        0.001, 0.01, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0, 120.0,
    },
    EnableLabeledMetrics: true,
}

collector, err := monitoring.NewSagaMetricsCollector(config)
```

## Prometheus 集成

### 配置 Prometheus

#### 基本配置 (`prometheus.yml`)

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'saga-service'
    static_configs:
      - targets: ['saga-service:8080']
    metrics_path: /metrics
    scrape_interval: 10s
```

#### 服务发现（Kubernetes）

```yaml
scrape_configs:
  - job_name: 'kubernetes-saga-pods'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        target_label: __address__
        regex: (.+):(.+);(.+)
        replacement: $1:$3
```

### 暴露指标端点

```go
package main

import (
    "net/http"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
    // 注册 Prometheus handler
    http.Handle("/metrics", promhttp.Handler())
    
    // 启动 HTTP 服务器
    http.ListenAndServe(":8080", nil)
}
```

### 常用 PromQL 查询

#### 成功率

```promql
# 5 分钟内的 Saga 成功率
(
  rate(saga_monitoring_saga_completed_total[5m]) 
  / 
  rate(saga_monitoring_saga_started_total[5m])
) * 100
```

#### 吞吐量

```promql
# 每秒完成的 Saga 数量
rate(saga_monitoring_saga_completed_total[1m])
```

#### 延迟百分位数

```promql
# P50 延迟
histogram_quantile(0.50, 
  rate(saga_monitoring_saga_duration_seconds_bucket[5m])
)

# P95 延迟
histogram_quantile(0.95, 
  rate(saga_monitoring_saga_duration_seconds_bucket[5m])
)

# P99 延迟
histogram_quantile(0.99, 
  rate(saga_monitoring_saga_duration_seconds_bucket[5m])
)
```

#### 失败分析

```promql
# 按失败原因分组的失败率
sum by (reason) (
  rate(saga_monitoring_saga_failed_total[5m])
)

# 最常见的失败原因
topk(5, 
  sum by (reason) (
    rate(saga_monitoring_saga_failed_total[1h])
  )
)
```

## Grafana 可视化

### 导入仪表板

1. 登录 Grafana (默认 admin/admin)
2. 导航到 **Dashboards** → **Import**
3. 上传 `examples/saga-monitoring/grafana-dashboard.json`
4. 选择 Prometheus 数据源
5. 点击 **Import**

### 关键面板

#### 1. Saga 执行速率

显示启动、完成和失败的 Saga 速率。

**查询：**
```promql
rate(saga_monitoring_saga_started_total[5m])
rate(saga_monitoring_saga_completed_total[5m])
rate(saga_monitoring_saga_failed_total[5m])
```

#### 2. 成功率仪表盘

显示实时成功率，带颜色编码的阈值。

**查询：**
```promql
(rate(saga_monitoring_saga_completed_total[5m]) / rate(saga_monitoring_saga_started_total[5m])) * 100
```

**阈值：**
- 绿色：≥ 99%
- 黄色：95% - 99%
- 红色：< 95%

#### 3. 执行时长热图

显示 Saga 执行时长的分布。

**查询：**
```promql
sum(rate(saga_monitoring_saga_duration_seconds_bucket[5m])) by (le)
```

#### 4. 失败原因饼图

显示不同失败原因的占比。

**查询：**
```promql
sum by (reason) (rate(saga_monitoring_saga_failed_total[5m]))
```

### 创建自定义面板

```json
{
  "title": "Custom Saga Metrics",
  "targets": [
    {
      "expr": "your_promql_query_here",
      "legendFormat": "{{label}}"
    }
  ],
  "type": "graph"
}
```

## 分布式追踪

### OpenTelemetry 集成

#### 配置追踪

```yaml
# tracing-config.yaml
opentelemetry:
  enabled: true
  service_name: "saga-coordinator"
  service_version: "1.0.0"
  
  sampling:
    strategy: "parent_based"
    ratio: 1.0
    
  batch_processor:
    max_queue_size: 2048
    max_export_batch_size: 512
    schedule_delay: "5s"
```

#### 初始化追踪

```go
package main

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func initTracer() (*trace.TracerProvider, error) {
    // 创建 Jaeger exporter
    exporter, err := jaeger.New(
        jaeger.WithCollectorEndpoint(
            jaeger.WithEndpoint("http://jaeger:14268/api/traces"),
        ),
    )
    if err != nil {
        return nil, err
    }
    
    // 创建 resource
    res, err := resource.New(
        context.Background(),
        resource.WithAttributes(
            semconv.ServiceNameKey.String("saga-coordinator"),
            semconv.ServiceVersionKey.String("1.0.0"),
        ),
    )
    if err != nil {
        return nil, err
    }
    
    // 创建 TracerProvider
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(res),
        trace.WithSampler(trace.AlwaysSample()),
    )
    
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

#### 为 Saga 添加追踪

```go
func executeSaga(ctx context.Context, sagaID string) error {
    tracer := otel.Tracer("saga")
    
    // 创建根 span
    ctx, span := tracer.Start(ctx, "saga.execute")
    defer span.End()
    
    // 添加属性
    span.SetAttributes(
        attribute.String("saga.id", sagaID),
        attribute.String("saga.type", "order-processing"),
    )
    
    // 执行步骤
    if err := executeStep(ctx, "step1"); err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }
    
    span.SetStatus(codes.Ok, "Saga completed successfully")
    return nil
}

func executeStep(ctx context.Context, stepName string) error {
    tracer := otel.Tracer("saga")
    ctx, span := tracer.Start(ctx, "saga.step."+stepName)
    defer span.End()
    
    // 步骤执行逻辑
    // ...
    
    return nil
}
```

### Jaeger UI 使用

#### 查找追踪

1. 访问 Jaeger UI (http://localhost:16686)
2. 选择服务：`saga-coordinator`
3. 选择操作：`saga.execute`
4. 设置时间范围
5. 点击 **Find Traces**

#### 分析追踪

查看追踪时注意：

- **总时长** - Saga 从开始到结束的总时间
- **Span 数量** - 涉及的服务和操作数量
- **错误标记** - 失败的步骤和错误信息
- **关键路径** - 最长的执行路径
- **并行度** - 同时执行的操作

#### 过滤和搜索

```
# 按标签搜索
saga.id="saga-123"

# 按错误状态搜索
error=true

# 按最小时长搜索
minDuration=5s
```

## 告警配置

### 告警规则

#### 高失败率告警

```yaml
- alert: HighSagaFailureRate
  expr: |
    (
      rate(saga_monitoring_saga_failed_total[5m]) 
      / 
      rate(saga_monitoring_saga_started_total[5m])
    ) > 0.10
  for: 5m
  labels:
    severity: critical
    component: saga
  annotations:
    summary: "High Saga failure rate detected"
    description: "Saga failure rate is {{ $value | humanizePercentage }}"
```

#### 慢执行告警

```yaml
- alert: SlowSagaExecution
  expr: |
    histogram_quantile(0.95, 
      rate(saga_monitoring_saga_duration_seconds_bucket[5m])
    ) > 60
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Slow Saga execution detected"
    description: "P95 duration is {{ $value | humanizeDuration }}"
```

#### 服务健康告警

```yaml
- alert: SagaServiceDown
  expr: up{job="saga-service"} == 0
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: "Saga service is down"
    description: "Instance {{ $labels.instance }} is not responding"
```

### Alertmanager 配置

#### 路由配置

```yaml
route:
  group_by: ['alertname', 'severity']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 3h
  receiver: 'default'
  
  routes:
    - match:
        severity: critical
      receiver: 'critical-alerts'
      group_wait: 5s
      repeat_interval: 30m
```

#### 接收器配置

**Slack 通知：**

```yaml
receivers:
  - name: 'critical-alerts'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'
        channel: '#critical-alerts'
        title: '[CRITICAL] {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

**Email 通知：**

```yaml
receivers:
  - name: 'saga-team'
    email_configs:
      - to: 'saga-team@example.com'
        from: 'alertmanager@example.com'
        smarthost: 'smtp.example.com:587'
        auth_username: 'alertmanager@example.com'
        auth_password: 'password'
```

**PagerDuty 集成：**

```yaml
receivers:
  - name: 'oncall'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_SERVICE_KEY'
        description: '{{ .GroupLabels.alertname }}'
```

## 性能监控

### 关键性能指标 (KPI)

#### 1. 可用性指标

- **成功率** - 目标：≥ 99.9%
- **故障时间** - 目标：< 0.1%
- **恢复时间 (MTTR)** - 目标：< 5 分钟

#### 2. 性能指标

- **P50 延迟** - 目标：< 1 秒
- **P95 延迟** - 目标：< 5 秒
- **P99 延迟** - 目标：< 30 秒
- **吞吐量** - 监控趋势

#### 3. 资源指标

- **CPU 使用率** - 警告：> 70%，严重：> 85%
- **内存使用** - 警告：> 2GB，严重：> 4GB
- **Goroutine 数量** - 警告：> 10000
- **连接池使用** - 警告：> 80%

### 服务级别目标 (SLO)

#### 定义 SLO

```yaml
# 99.9% 成功率 SLO
- alert: SagaSuccessRateSLOAtRisk
  expr: |
    (
      sum(rate(saga_monitoring_saga_completed_total[1h]))
      /
      sum(rate(saga_monitoring_saga_started_total[1h]))
    ) < 0.999
  for: 15m
  labels:
    slo: "99.9% success rate"
```

#### 错误预算

计算每月错误预算：

```
可用性目标 = 99.9%
允许的故障率 = 0.1%
每月总请求 = 10,000,000
错误预算 = 10,000 次失败
```

监控错误预算消耗：

```promql
# 当月已消耗的错误预算百分比
(
  sum(saga_monitoring_saga_failed_total{month="current"})
  /
  10000
) * 100
```

### 性能分析工具

#### 1. Prometheus 查询

```promql
# 识别慢服务
topk(5, 
  histogram_quantile(0.95, 
    sum(rate(saga_monitoring_saga_duration_seconds_bucket[1h])) by (service, le)
  )
)
```

#### 2. Grafana 热图

使用热图面板可视化延迟分布，识别异常模式。

#### 3. Jaeger 性能分析

在 Jaeger UI 中：
- 比较不同时间段的追踪
- 识别最慢的 span
- 分析服务依赖关系

## 故障排查

### 常见问题

#### 问题 1：高失败率

**症状：**
- Saga 失败率突然升高
- `HighSagaFailureRate` 告警触发

**排查步骤：**

1. 检查失败原因分布：
   ```promql
   sum by (reason) (rate(saga_monitoring_saga_failed_total[5m]))
   ```

2. 查看相关追踪：
   - 在 Jaeger UI 搜索 `error=true`
   - 分析错误堆栈

3. 检查下游服务：
   ```bash
   curl http://downstream-service/health
   ```

4. 查看日志：
   ```bash
   kubectl logs -f saga-service-pod --tail=100
   ```

#### 问题 2：慢执行

**症状：**
- P95/P99 延迟升高
- `SlowSagaExecution` 告警

**排查步骤：**

1. 识别慢步骤：
   - 在 Jaeger 查看追踪
   - 找到耗时最长的 span

2. 检查数据库性能：
   ```promql
   histogram_quantile(0.95, 
     rate(saga_storage_operation_duration_seconds_bucket[5m])
   )
   ```

3. 检查消息队列：
   ```promql
   saga_messaging_queue_depth
   ```

4. 查看资源使用：
   ```promql
   rate(process_cpu_seconds_total[5m]) * 100
   process_resident_memory_bytes / 1024 / 1024 / 1024
   ```

#### 问题 3：补偿失败

**症状：**
- 补偿操作失败率升高
- 数据不一致风险

**排查步骤：**

1. 查看补偿失败指标：
   ```promql
   rate(saga_compensation_failed_total[5m])
   ```

2. 检查补偿逻辑：
   - 审查补偿操作代码
   - 验证幂等性实现

3. 查看相关追踪和日志

4. 如需手动干预：
   ```bash
   # 使用管理 API 重试补偿
   curl -X POST http://saga-service/admin/retry-compensation \
     -d '{"saga_id": "saga-123"}'
   ```

### 调试技巧

#### 启用详细日志

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```

#### 临时增加采样率

```yaml
opentelemetry:
  sampling:
    strategy: "always_on"  # 采样所有追踪
```

#### 使用 Prometheus 即时查询

```bash
# 查询当前活跃 Saga
curl 'http://localhost:9090/api/v1/query?query=saga_monitoring_active_sagas'
```

## 最佳实践

### 1. 指标采集

- **选择合适的时间窗口** - 通常使用 5m 或 10m
- **避免高基数标签** - 不要使用 user_id、request_id 等作为标签
- **使用直方图测量延迟** - 比平均值更有价值
- **及时清理旧指标** - 配置合理的保留期

### 2. 追踪策略

- **生产环境使用采样** - 减少开销（如 10% 采样率）
- **关键路径始终采样** - 使用 parent_based 策略
- **添加有意义的属性** - 包含业务上下文
- **传播追踪上下文** - 使用 W3C Trace Context 标准

### 3. 告警设计

- **基于 SLO 设置告警** - 关注用户影响
- **避免告警疲劳** - 合理分组和抑制
- **可执行的告警** - 提供 runbook 链接
- **分级告警** - critical/warning/info

### 4. 性能优化

- **批处理指标上报** - 减少网络开销
- **异步导出追踪** - 不阻塞主流程
- **合理设置队列大小** - 平衡内存和丢失风险
- **监控监控系统** - 确保 Prometheus/Jaeger 健康

### 5. 安全考虑

- **不要记录敏感信息** - 在指标和追踪中避免 PII
- **使用 TLS 加密** - 保护指标和追踪数据传输
- **访问控制** - 限制对监控界面的访问
- **数据保留策略** - 合规性考虑

## 参考资源

### 文档

- [Prometheus 文档](https://prometheus.io/docs/)
- [Grafana 文档](https://grafana.com/docs/)
- [Jaeger 文档](https://www.jaegertracing.io/docs/)
- [OpenTelemetry 文档](https://opentelemetry.io/docs/)

### 示例

- [完整监控示例](../examples/saga-monitoring/)
- [Prometheus 集成示例](../examples/prometheus-monitoring/)
- [分布式追踪示例](../examples/distributed-tracing/)

### 相关指南

- [Saga 用户指南](saga-user-guide.md)
- [Saga 恢复监控](saga-recovery-monitoring.md)
- [日志最佳实践](logging-best-practices.md)
- [性能监控和告警](performance-monitoring-alerting.md)

## 总结

有效的监控是确保 Saga 系统可靠运行的关键。通过结合 Prometheus 指标、Grafana 可视化、Jaeger 追踪和 Alertmanager 告警，您可以：

1. 实时了解系统健康状况
2. 快速诊断和解决问题
3. 持续优化性能
4. 满足 SLA 要求
5. 支持容量规划

记住：**监控不是一次性任务，而是持续改进的过程**。随着系统演进，持续调整和优化您的监控策略。

