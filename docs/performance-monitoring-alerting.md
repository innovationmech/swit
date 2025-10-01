# Performance Monitoring and Alerting

This guide covers the comprehensive performance monitoring and alerting system in Swit, including Service Level Objectives (SLOs), error budget tracking, and automated alerting hooks integrated with Prometheus and Grafana.

## Overview

The Swit framework provides a complete observability solution with:

- **Service Level Objectives (SLOs)**: Define and track service quality targets
- **Error Budget Tracking**: Monitor error budget consumption and burn rates
- **Automated Alerting**: Hook-based alerting system integrated with performance metrics
- **Prometheus Integration**: Real-time metrics collection and querying
- **Grafana Dashboards**: Pre-built dashboards for SLO visualization
- **Multi-Window Burn Rate Alerts**: Google SRE-inspired alert rules

## Architecture

```
┌─────────────────────┐
│  Performance        │
│  Monitor            │──────┐
└─────────────────────┘      │
                             │
┌─────────────────────┐      │      ┌─────────────────────┐
│  SLO Monitor        │      │      │  Alerting           │
│  - Availability     │──────┼──────│  Manager            │
│  - Latency          │      │      │  - Deduplication    │
│  - Error Rate       │      │      │  - Rate Limiting    │
│  - Throughput       │      │      │  - Handlers         │
└─────────────────────┘      │      └─────────────────────┘
                             │              │
                             │              │
                      ┌──────▼──────┐       │
                      │  Performance │       │
                      │  Hooks       │◄──────┘
                      └──────┬──────┘
                             │
                      ┌──────▼──────────┐
                      │  Prometheus     │
                      │  Metrics        │
                      └─────────────────┘
```

## Service Level Objectives (SLOs)

### What are SLOs?

Service Level Objectives define target levels of service quality. Common SLOs include:

- **Availability**: Percentage of successful requests (e.g., 99.9%)
- **Latency**: Percentage of requests served within a threshold (e.g., 95% < 500ms)
- **Error Rate**: Percentage of requests that succeed (e.g., 99%)
- **Throughput**: Minimum requests per second

### Defining SLOs

```go
import "github.com/innovationmech/swit/pkg/types"

// Create SLO targets
targets := []types.SLOTarget{
    {
        Name:              "availability_99_9",
        Description:       "99.9% service availability over 30 days",
        Type:              types.SLOTypeAvailability,
        Target:            99.9,
        Window:            30 * 24 * time.Hour,
        WarningThreshold:  99.5,
        CriticalThreshold: 99.0,
        MetricQuery:       `100 * (1 - (sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))))`,
        Labels: map[string]string{
            "tier":     "critical",
            "category": "availability",
        },
    },
    {
        Name:              "latency_p95_500ms",
        Description:       "95th percentile latency under 500ms",
        Type:              types.SLOTypeLatency,
        Target:            95.0,
        Window:            7 * 24 * time.Hour,
        WarningThreshold:  90.0,
        CriticalThreshold: 85.0,
        MetricQuery:       `100 * (sum(rate(http_request_duration_seconds_bucket{le="0.5"}[5m])) / sum(rate(http_request_duration_seconds_count[5m])))`,
        Labels: map[string]string{
            "tier":     "important",
            "category": "performance",
        },
    },
}

// Create SLO configuration
sloConfig := &types.SLOConfig{
    Enabled:         true,
    Targets:         targets,
    UpdateInterval:  1 * time.Minute,
    RetentionPeriod: 30 * 24 * time.Hour,
}

// Create SLO monitor
sloMonitor := types.NewSLOMonitor(sloConfig)
```

### Updating SLO Status

```go
// Update SLO status with current measurement
err := sloMonitor.UpdateStatus("availability_99_9", 99.95)
if err != nil {
    log.Printf("Failed to update SLO: %v", err)
}

// Get current status
status, err := sloMonitor.GetStatus("availability_99_9")
if err != nil {
    log.Printf("Failed to get SLO status: %v", err)
}

fmt.Printf("SLO Status: %s\n", status.Status)
fmt.Printf("Current Value: %.2f%%\n", status.CurrentValue)
fmt.Printf("Error Budget Remaining: %.2f%%\n", status.ErrorBudgetRemaining)
```

### Checking SLO Health

```go
// Check if all SLOs are healthy
if sloMonitor.IsHealthy() {
    fmt.Println("All SLOs are healthy")
} else {
    fmt.Println("Some SLOs are violated")
}

// Get all SLO statuses
statuses := sloMonitor.GetAllStatuses()
for _, status := range statuses {
    fmt.Printf("SLO: %s, Status: %s, Value: %.2f%%\n",
        status.Target.Name,
        status.Status,
        status.CurrentValue)
}

// Get error budget status
budgets := sloMonitor.GetErrorBudgetStatus()
for name, remaining := range budgets {
    fmt.Printf("SLO: %s, Budget Remaining: %.2f%%\n", name, remaining)
}
```

## Alerting System

### Creating an Alerting Manager

```go
import "github.com/innovationmech/swit/pkg/server"

// Create alerting configuration
alertingConfig := &server.AlertingConfig{
    Enabled:                 true,
    DeduplicationWindow:     5 * time.Minute,
    MaxAlertsPerMinute:      10,
    EnableSLOAlerts:         true,
    EnablePerformanceAlerts: true,
}

// Create alerting manager
alertingManager := server.NewAlertingManager(
    alertingConfig,
    sloMonitor,
    metricsCollector,
)

// Start the alerting manager
alertingManager.Start()
defer alertingManager.Stop()
```

### Adding Alert Handlers

```go
// Add logging handler
alertingManager.AddHandler(server.LoggingAlertHandler)

// Add metrics handler
alertingManager.AddHandler(server.MetricsAlertHandler(metricsCollector))

// Add Prometheus handler
alertingManager.AddHandler(server.PrometheusAlertHandler(metricsCollector))

// Add custom handler
alertingManager.AddHandler(func(alert *server.Alert) error {
    // Send to external system (Slack, PagerDuty, etc.)
    fmt.Printf("Alert: %s - %s\n", alert.Name, alert.Description)
    return nil
})
```

### Manual Alert Triggering

```go
// Create and trigger a custom alert
alert := &server.Alert{
    ID:          "custom-alert-001",
    Name:        "Custom Performance Alert",
    Description: "Response time exceeded threshold",
    Severity:    server.AlertSeverityWarning,
    Timestamp:   time.Now(),
    Labels: map[string]string{
        "service":  "my-service",
        "endpoint": "/api/v1/users",
    },
    Annotations: map[string]string{
        "runbook": "https://docs.example.com/runbooks/latency",
    },
    Value:     850.5,  // milliseconds
    Threshold: 500.0,  // milliseconds
    Source:    "custom",
}

err := alertingManager.TriggerAlert(alert)
if err != nil {
    log.Printf("Failed to trigger alert: %v", err)
}
```

## Performance Monitoring Hooks

### Integrating with Performance Monitor

```go
import "github.com/innovationmech/swit/pkg/server"

// Create performance monitor
perfMonitor := server.NewPerformanceMonitor()

// Add performance alerting hook
perfMonitor.AddHook(server.PerformanceAlertingHook(alertingManager))

// Add SLO alerting hook
perfMonitor.AddHook(server.SLOAlertingHook(alertingManager, sloMonitor))

// Add logging hook
perfMonitor.AddHook(server.PerformanceLoggingHook)

// Add threshold violation hook
perfMonitor.AddHook(server.PerformanceThresholdViolationHook)
```

### Using with Business Server

```go
import (
    "github.com/innovationmech/swit/pkg/server"
    "github.com/innovationmech/swit/pkg/types"
)

// Create server configuration
config := server.NewServerConfig()
config.ServiceName = "my-service"

// Enable Prometheus
config.Prometheus.Enabled = true
config.Prometheus.Endpoint = "/metrics"

// Create server with monitoring
srv, err := server.NewBusinessServerCore(config, registrar, nil)
if err != nil {
    log.Fatalf("Failed to create server: %v", err)
}

// Get performance monitor
perfMonitor := srv.(server.BusinessServerWithPerformance).GetPerformanceMonitor()

// Setup SLO monitoring
sloConfig := types.DefaultSLOConfig()
sloConfig.Enabled = true
sloMonitor := types.NewSLOMonitor(sloConfig)

// Setup alerting
alertingConfig := server.DefaultAlertingConfig()
alertingConfig.Enabled = true
alertingManager := server.NewAlertingManager(
    alertingConfig,
    sloMonitor,
    srv.GetMetricsCollector(),
)

// Wire up hooks
perfMonitor.AddHook(server.PerformanceAlertingHook(alertingManager))
perfMonitor.AddHook(server.SLOAlertingHook(alertingManager, sloMonitor))

// Start monitoring
alertingManager.Start()
defer alertingManager.Stop()

// Start server
if err := srv.Start(); err != nil {
    log.Fatalf("Failed to start server: %v", err)
}
```

## Prometheus Alert Rules

### SLO-Based Alert Rules

The framework provides comprehensive Prometheus alert rules in `examples/prometheus-monitoring/alerting/slo-rules.yml`:

#### Availability Alerts
- **SLOAvailability99_9Violated**: Fires when availability drops below 99.9%
- **SLOAvailabilityErrorBudgetLow**: Warns when 80% of error budget consumed
- **SLOAvailabilityErrorBudgetCritical**: Critical when 95% of error budget consumed

#### Latency Alerts
- **SLOLatencyP95Violated**: Fires when P95 latency SLO violated
- **SLOLatencyP95Warning**: Warns when approaching latency SLO
- **SLOLatencyP99Violated**: Fires when P99 latency exceeds threshold

#### Error Rate Alerts
- **SLOErrorRateViolated**: Fires when error rate SLO violated
- **SLOErrorBudgetBurnRate**: Alerts on rapid error budget consumption

#### Multi-Window Burn Rate Alerts
- **SLOErrorBudgetFastBurn**: Fast burn (2% budget in 1 hour)
- **SLOErrorBudgetMediumBurn**: Medium burn (5% budget in 6 hours)
- **SLOErrorBudgetSlowBurn**: Slow burn tracking (10% budget in 3 days)

### Configuring Prometheus

Add SLO rules to Prometheus configuration:

```yaml
# prometheus.yml
rule_files:
  - /etc/prometheus/rules.yml
  - /etc/prometheus/slo-rules.yml

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']
```

## Grafana Dashboards

### SLO Monitoring Dashboard

The framework provides a pre-built SLO monitoring dashboard at `examples/prometheus-monitoring/grafana/dashboards/slo-monitoring.json`.

**Dashboard Link**: http://localhost:3000/d/swit-slo-monitoring/swit-slo-monitoring

**Dashboard Features**:
- Real-time SLO status gauges for availability, latency, and error rate
- Error budget consumption tracking
- SLO trend visualization over time
- Multi-window error budget burn rate analysis
- Alert status summary

### Accessing Dashboards

1. **Start the monitoring stack**:
   ```bash
   cd examples/prometheus-monitoring
   docker-compose up -d
   ```

2. **Access Grafana**:
   - URL: http://localhost:3000
   - Login: admin/admin
   - Navigate to Dashboards → Swit - SLO Monitoring

3. **View Prometheus**:
   - URL: http://localhost:9090
   - Check targets: http://localhost:9090/targets
   - View alert rules: http://localhost:9090/alerts

4. **View Alertmanager**:
   - URL: http://localhost:9093
   - View active alerts and silences

## Error Budget Management

### Understanding Error Budgets

An error budget is the allowed amount of unreliability. If your SLO is 99.9% availability:
- You have a 0.1% error budget
- Over 30 days: ~43 minutes of downtime allowed
- Consuming the budget faster than expected triggers alerts

### Calculating Error Budget

```go
// Get error budget status
budgets := sloMonitor.GetErrorBudgetStatus()

for name, remaining := range budgets {
    if remaining < 20 {
        fmt.Printf("WARNING: SLO %s has only %.2f%% error budget remaining\n", 
            name, remaining)
    } else if remaining < 10 {
        fmt.Printf("CRITICAL: SLO %s has only %.2f%% error budget remaining\n", 
            name, remaining)
    }
}
```

### Burn Rate Analysis

Burn rate indicates how quickly you're consuming your error budget:

- **1x burn rate**: Consuming budget at expected rate (will last the full SLO window)
- **10x burn rate**: Consuming budget 10x faster (will exhaust in 1/10th of the window)
- **Fast burn (14.4x)**: Will exhaust 30-day budget in ~2 days
- **Medium burn (6x)**: Will exhaust 30-day budget in ~5 days

## Best Practices

### 1. Define Meaningful SLOs

```go
// Good: Based on user experience
types.SLOTarget{
    Name:   "checkout_latency_p95",
    Target: 95.0, // 95% of checkout requests under 1s
    Window: 7 * 24 * time.Hour,
}

// Good: Based on business requirements
types.SLOTarget{
    Name:   "payment_availability",
    Target: 99.95, // Critical payment path
    Window: 30 * 24 * time.Hour,
}
```

### 2. Use Multi-Window Burn Rate Alerts

Avoid alert fatigue by using different windows:

```yaml
# Fast burn: Immediate attention
expr: error_rate[1h] > 14.4x and error_rate[5m] > 14.4x
for: 2m

# Medium burn: Soon needs attention  
expr: error_rate[6h] > 6x and error_rate[30m] > 6x
for: 15m
```

### 3. Implement Alert Handlers Carefully

```go
// Good: Non-blocking alert handler
alertingManager.AddHandler(func(alert *server.Alert) error {
    go func() {
        // Send to external system
        sendToSlack(alert)
    }()
    return nil
})

// Bad: Blocking alert handler
alertingManager.AddHandler(func(alert *server.Alert) error {
    // This blocks alert processing
    return sendToSlackSync(alert)
})
```

### 4. Monitor Alert Volume

```go
// Track alert metrics
metricsCollector.IncrementCounter("alerts_triggered_total", map[string]string{
    "severity": string(alert.Severity),
    "source":   alert.Source,
})

// Use rate limiting
alertingConfig := &server.AlertingConfig{
    MaxAlertsPerMinute: 10,
    DeduplicationWindow: 5 * time.Minute,
}
```

### 5. Review SLOs Regularly

- Review SLO targets quarterly
- Adjust thresholds based on actual performance
- Consider user impact when setting targets
- Balance reliability with development velocity

## Troubleshooting

### Alerts Not Firing

1. **Check alerting manager is enabled**:
   ```go
   alertingConfig.Enabled = true
   ```

2. **Verify handlers are registered**:
   ```go
   alertingManager.AddHandler(server.LoggingAlertHandler)
   ```

3. **Check rate limiting**:
   ```go
   // Increase if needed
   alertingConfig.MaxAlertsPerMinute = 20
   ```

### SLO Status Not Updating

1. **Verify SLO monitor is created**:
   ```go
   sloMonitor := types.NewSLOMonitor(sloConfig)
   ```

2. **Check UpdateStatus is called**:
   ```go
   err := sloMonitor.UpdateStatus(targetName, currentValue)
   ```

3. **Ensure hooks are added**:
   ```go
   perfMonitor.AddHook(server.SLOAlertingHook(alertingManager, sloMonitor))
   ```

### High Alert Volume

1. **Increase deduplication window**:
   ```go
   alertingConfig.DeduplicationWindow = 10 * time.Minute
   ```

2. **Adjust alert thresholds**:
   ```go
   target.WarningThreshold = 99.0  // Lower threshold
   ```

3. **Review alert conditions**:
   ```yaml
   # Add longer 'for' duration
   for: 10m  # Wait longer before firing
   ```

## Examples

See complete working examples in:
- `examples/prometheus-monitoring/`: Full monitoring stack with SLO dashboards
- `examples/prometheus-monitoring/alerting/slo-rules.yml`: SLO alert rules
- `examples/prometheus-monitoring/grafana/dashboards/slo-monitoring.json`: SLO dashboard

## References

- [Google SRE Book - Monitoring](https://sre.google/sre-book/monitoring-distributed-systems/)
- [Google SRE Workbook - Implementing SLOs](https://sre.google/workbook/implementing-slos/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/alerting/)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/best-practices/)
- [Multi-Window, Multi-Burn-Rate Alerts](https://sre.google/workbook/alerting-on-slos/)

