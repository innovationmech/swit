# Saga Recovery Monitoring Example

本示例展示如何使用 Saga 恢复系统的监控和告警功能。

## 功能特性

- ✅ Prometheus 指标导出
- ✅ 自动故障检测和恢复
- ✅ 告警规则和通知
- ✅ 恢复事件订阅
- ✅ HTTP API 查询统计信息

## 快速开始

### 1. 运行示例

```bash
cd examples/saga-recovery
go run main.go
```

服务将在以下端口启动：
- `9090`: Prometheus 指标和 HTTP API

### 2. 查看指标

访问 Prometheus 指标端点：
```bash
curl http://localhost:9090/metrics
```

查看恢复统计信息：
```bash
curl http://localhost:9090/recovery/stats
```

### 3. 配置 Prometheus

创建 `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'saga-recovery'
    static_configs:
      - targets: ['localhost:9090']
        labels:
          service: 'saga-recovery-example'
```

启动 Prometheus:
```bash
prometheus --config.file=prometheus.yml
```

### 4. 配置告警规则

使用提供的告警规则文件：
```bash
prometheus --config.file=prometheus.yml \
  --web.enable-lifecycle \
  --storage.tsdb.path=./prometheus-data
```

在 `prometheus.yml` 中添加告警规则：
```yaml
rule_files:
  - 'prometheus-alerts.yml'

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['localhost:9093']
```

### 5. 设置 Grafana 仪表盘

导入 Grafana 仪表盘（如果提供）或使用以下查询创建自定义面板：

**恢复成功率**:
```promql
rate(saga_recovery_success_total[5m]) / rate(saga_recovery_attempts_total[5m])
```

**平均恢复时间**:
```promql
rate(saga_recovery_duration_seconds_sum[5m]) / rate(saga_recovery_duration_seconds_count[5m])
```

**P95 恢复时间**:
```promql
histogram_quantile(0.95, rate(saga_recovery_duration_seconds_bucket[5m]))
```

**按类型统计的故障**:
```promql
sum(rate(saga_recovery_detected_failures_total[5m])) by (failure_type)
```

## 可用端点

### Prometheus 指标

```
GET /metrics
```

返回 Prometheus 格式的指标。

### 健康检查

```
GET /health
```

返回服务健康状态。

### 恢复统计

```
GET /recovery/stats
```

返回 JSON 格式的恢复统计信息：
```json
{
  "total_attempts": 100,
  "successful_recoveries": 85,
  "failed_recoveries": 15,
  "currently_recovering": 2,
  "success_rate": 0.85,
  "average_duration": "2.5s"
}
```

## 告警规则

示例包含以下告警规则（参见 `prometheus-alerts.yml`）：

1. **SagaRecoveryHighFailureRate**: 恢复失败率 > 10%
2. **SagaTooManyStuckSagas**: 卡住的 Saga > 10 个
3. **SagaRecoverySlowPerformance**: 平均恢复时间 > 30 秒
4. **SagaTooManyRecovering**: 同时恢复的 Saga > 50 个
5. **SagaHighManualInterventionRate**: 手动干预率 > 20%

## 自定义配置

可以通过修改代码中的配置来自定义恢复行为：

```go
recoveryConfig := state.DefaultRecoveryConfig()
recoveryConfig.EnableAutoRecovery = true
recoveryConfig.CheckInterval = 30 * time.Second
recoveryConfig.RecoveryTimeout = 30 * time.Second
recoveryConfig.MaxConcurrentRecoveries = 10
recoveryConfig.MaxRecoveryAttempts = 3
```

## 自定义告警通知

实现自定义告警通知器：

```go
type CustomNotifier struct {
    // Your notifier fields
}

func (n *CustomNotifier) Notify(ctx context.Context, alert *state.RecoveryAlert) error {
    // Send alert to your notification system
    // (Slack, Email, PagerDuty, etc.)
    return nil
}

// 注册通知器
recoveryManager.AddAlertNotifier(&CustomNotifier{})
```

## 监控最佳实践

1. **设置合理的告警阈值**: 根据业务需求调整告警阈值
2. **监控趋势变化**: 不仅关注当前值，还要关注趋势
3. **关联日志和追踪**: 将指标与日志系统集成
4. **定期审查**: 定期审查告警规则的有效性

## 故障排查

### 高失败率

检查失败类型分布：
```promql
sum(rate(saga_recovery_failure_total[5m])) by (error_type)
```

### 恢复时间过长

按策略分析恢复时间：
```promql
histogram_quantile(0.95, rate(saga_recovery_duration_seconds_bucket[5m])) by (strategy)
```

### 卡住的 Saga

检查卡住的 Saga 数量：
```promql
saga_recovery_detected_failures_total{failure_type="stuck"}
```

## 相关文档

- [Saga 恢复监控指南](../../docs/saga-recovery-monitoring.md)
- [Saga 用户指南](../../docs/saga-user-guide.md)
- [性能监控最佳实践](../../docs/performance-monitoring-alerting.md)

