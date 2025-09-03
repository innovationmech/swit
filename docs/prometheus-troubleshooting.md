# Prometheus Troubleshooting Guide

This guide helps you diagnose and resolve common issues with Prometheus metrics integration in the Swit framework.

## Common Issues

### 1. Metrics Not Being Collected

#### Symptoms
- `/metrics` endpoint returns empty or minimal data
- Expected custom metrics are missing
- Built-in metrics not appearing in Prometheus

#### Diagnosis
```bash
# Check metrics endpoint
curl http://localhost:8080/metrics

# Verify server configuration
grep -r "prometheus" your-service.yaml

# Check logs for initialization errors
tail -f /var/log/your-service.log | grep -i prometheus
```

#### Solutions

**Missing Prometheus Configuration:**
```yaml
# Ensure Prometheus is enabled in config
monitoring:
  prometheus:
    enabled: true
    endpoint: "/metrics"
    cardinality_limit: 10000
```

**Business Metrics Not Registered:**
```go
// Ensure business metrics manager is properly initialized
bmm := server.NewBusinessMetricsManager("your-service", collector, registry)

// Register hooks if needed
loggingHook := server.NewLoggingBusinessMetricsHook()
bmm.RegisterHook(loggingHook)
```

**Custom Metrics Not Defined:**
```go
// Register custom metrics in your service
definition := server.MetricDefinition{
    Name:        "custom_business_operations_total",
    Type:        types.CounterType,
    Description: "Total number of custom business operations",
    Labels:      []string{"operation", "status"},
}
bmm.RegisterCustomMetric(definition)
```

### 2. High Memory Usage from Metrics

#### Symptoms
- Memory usage continuously growing
- Out of memory errors
- Prometheus scrape timeouts

#### Diagnosis
```bash
# Check metric cardinality
curl -s http://localhost:8080/metrics | wc -l

# Monitor memory usage
top -p $(pgrep your-service)

# Check for high cardinality labels
curl -s http://localhost:8080/metrics | grep -E "user_id|request_id|timestamp" | head -10
```

#### Solutions

**Enable Cardinality Limiting:**
```yaml
monitoring:
  prometheus:
    cardinality_limit: 5000  # Adjust based on your needs
    cardinality_check_interval: "1m"
```

**Remove High-Cardinality Labels:**
```go
// BAD: Using unique identifiers as labels
labels := map[string]string{
    "user_id": userID,        // High cardinality
    "request_id": requestID,  // High cardinality
}

// GOOD: Use aggregated labels
labels := map[string]string{
    "user_type": "premium",   // Low cardinality
    "endpoint": "/api/users", // Low cardinality
}
```

**Implement Metric Cleanup:**
```go
// Regularly clean up unused metrics
collector := server.GetPrometheusCollector()
if err := collector.CleanupExpiredMetrics(); err != nil {
    log.Printf("Failed to cleanup metrics: %v", err)
}
```

### 3. Prometheus Scrape Failures

#### Symptoms
- Prometheus shows targets as "DOWN"
- Scrape duration warnings in Prometheus logs
- Missing data points in Grafana

#### Diagnosis
```bash
# Check Prometheus targets
curl http://prometheus-server:9090/api/v1/targets

# Verify network connectivity
curl -I http://your-service:8080/metrics

# Check service discovery
consul catalog services  # If using Consul
```

#### Solutions

**Timeout Configuration:**
```yaml
# Increase timeouts for large metric sets
http:
  read_timeout: "30s"
  write_timeout: "30s"

# Configure Prometheus scrape timeout
scrape_configs:
  - job_name: 'swit-services'
    scrape_timeout: 30s
    scrape_interval: 15s
```

**Network Issues:**
```yaml
# Ensure metrics endpoint is accessible
http:
  address: "0.0.0.0"  # Not 127.0.0.1 for containerized apps
  port: "8080"

# Check firewall rules
iptables -L INPUT | grep 8080
```

### 4. Missing Built-in Metrics

#### Symptoms
- Standard HTTP/gRPC metrics not appearing
- Server uptime metrics missing
- Transport-specific metrics absent

#### Diagnosis
```bash
# Check for standard metric prefixes
curl -s http://localhost:8080/metrics | grep -E "^(http_|grpc_|server_)"

# Verify middleware configuration
grep -A 10 "middleware" your-service.yaml
```

#### Solutions

**Enable HTTP Metrics Middleware:**
```yaml
http:
  middleware:
    enable_metrics: true
    enable_logging: true
```

**Enable gRPC Interceptors:**
```yaml
grpc:
  interceptors:
    enable_metrics: true
    enable_logging: true
```

**Initialize Default Registry:**
```go
// Ensure default metrics are registered
registry := server.NewMetricsRegistry()
// Built-in metrics are automatically registered
```

### 5. Label Sanitization Issues

#### Symptoms
- Metrics with invalid label names
- Prometheus parse errors
- Labels with special characters

#### Diagnosis
```bash
# Check for invalid label names
curl -s http://localhost:8080/metrics | grep -E '[^a-zA-Z0-9_:]'
```

#### Solutions

**Automatic Label Sanitization:**
```go
// Framework automatically sanitizes labels
labels := map[string]string{
    "endpoint-name": "/api/users",  // Becomes "endpoint_name"
    "user.type": "premium",         // Becomes "user_type"
}
```

**Manual Label Validation:**
```go
func sanitizeLabel(label string) string {
    // Replace invalid characters with underscores
    re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
    return re.ReplaceAllString(label, "_")
}
```

### 6. Performance Impact

#### Symptoms
- Request latency increased after enabling metrics
- High CPU usage from metrics collection
- Slow application startup

#### Diagnosis
```bash
# Profile metrics collection
go test -bench=BenchmarkPrometheusMetricsCollector -cpuprofile=cpu.prof

# Check goroutine count
curl http://localhost:8080/debug/pprof/goroutine?debug=1
```

#### Solutions

**Optimize Metric Collection:**
```yaml
monitoring:
  prometheus:
    collection_interval: "10s"  # Reduce collection frequency
    disable_expensive_metrics: true
```

**Use Sampling for High-Volume Metrics:**
```go
// Sample high-volume metrics
if rand.Float64() < 0.1 { // 10% sampling
    bmm.RecordHistogram("expensive_operation_duration", duration, labels)
}
```

**Disable Unnecessary Metrics:**
```yaml
monitoring:
  prometheus:
    disabled_metrics:
      - "go_gc_*"
      - "process_*"
```

## Configuration Troubleshooting

### Environment Variables Not Working

```bash
# Check environment variable precedence
export SWIT_PROMETHEUS_ENABLED=true
export SWIT_PROMETHEUS_ENDPOINT="/custom-metrics"

# Verify variable loading
your-service --show-config
```

### YAML Configuration Issues

```yaml
# Use consistent indentation (spaces, not tabs)
monitoring:
  prometheus:  # 2 spaces
    enabled: true  # 4 spaces
    endpoint: "/metrics"  # 4 spaces
```

### Configuration Validation Errors

```go
// Add custom validation
func (c *PrometheusConfig) Validate() error {
    if c.Enabled && c.Endpoint == "" {
        return fmt.Errorf("prometheus endpoint cannot be empty when enabled")
    }
    return nil
}
```

## Integration Troubleshooting

### Consul Service Discovery Issues

```bash
# Check service registration
consul catalog service your-service

# Verify health checks
consul catalog nodes -service your-service
```

### Docker/Kubernetes Networking

```yaml
# Ensure correct service definition
apiVersion: v1
kind: Service
metadata:
  name: your-service-metrics
spec:
  ports:
  - name: metrics
    port: 8080
    targetPort: 8080
  selector:
    app: your-service
```

### Load Balancer Configuration

```nginx
# Nginx configuration for metrics scraping
upstream your-service-metrics {
    server your-service:8080;
}

server {
    location /metrics {
        proxy_pass http://your-service-metrics;
        proxy_connect_timeout 30s;
        proxy_read_timeout 30s;
    }
}
```

## Advanced Debugging

### Enable Debug Logging

```yaml
logging:
  level: debug
  enable_metrics_logging: true

monitoring:
  prometheus:
    debug_mode: true
```

### Custom Health Checks

```go
type PrometheusHealthCheck struct {
    collector *server.PrometheusMetricsCollector
}

func (h *PrometheusHealthCheck) CheckHealth(ctx context.Context) *types.HealthStatus {
    if h.collector == nil {
        return &types.HealthStatus{
            Status:  types.HealthStatusUnhealthy,
            Message: "Prometheus collector not initialized",
        }
    }
    
    metrics := h.collector.GetMetrics()
    if len(metrics) == 0 {
        return &types.HealthStatus{
            Status:  types.HealthStatusDegraded,
            Message: "No metrics collected yet",
        }
    }
    
    return &types.HealthStatus{
        Status:  types.HealthStatusHealthy,
        Message: fmt.Sprintf("Collecting %d metrics", len(metrics)),
    }
}
```

### Metrics Debugging Endpoint

```go
func (h *DebugHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    
    debug := ginRouter.Group("/debug")
    {
        debug.GET("/metrics/raw", h.getRawMetrics)
        debug.GET("/metrics/stats", h.getMetricsStats)
        debug.POST("/metrics/reset", h.resetMetrics)
    }
    
    return nil
}

func (h *DebugHandler) getMetricsStats(c *gin.Context) {
    collector := server.GetPrometheusCollector()
    stats := collector.GetCollectionStats()
    
    c.JSON(200, gin.H{
        "total_metrics": stats.TotalMetrics,
        "cardinality": stats.Cardinality,
        "memory_usage": stats.MemoryUsage,
        "last_collection": stats.LastCollection,
    })
}
```

## Performance Optimization

### Metric Aggregation

```go
// Use aggregation hooks to reduce metric volume
aggregationHook := server.NewAggregationBusinessMetricsHook()
bmm.RegisterHook(aggregationHook)

// Get aggregated values
counters := aggregationHook.GetCounterTotals()
gauges := aggregationHook.GetGaugeValues()
```

### Batch Processing

```go
// Collect metrics in batches
type BatchCollector struct {
    batch []types.Metric
    mu    sync.Mutex
}

func (b *BatchCollector) AddMetric(metric types.Metric) {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    b.batch = append(b.batch, metric)
    
    if len(b.batch) >= 100 {
        b.flushBatch()
    }
}
```

### Memory Management

```go
// Implement metric expiration
type ExpiringCollector struct {
    metrics map[string]*TimestampedMetric
    ttl     time.Duration
}

func (c *ExpiringCollector) cleanup() {
    now := time.Now()
    for key, metric := range c.metrics {
        if now.Sub(metric.Timestamp) > c.ttl {
            delete(c.metrics, key)
        }
    }
}
```

## Monitoring and Alerting

### Key Metrics to Monitor

```yaml
# Prometheus alerting rules
groups:
- name: swit-metrics-health
  rules:
  - alert: HighMetricCardinality
    expr: prometheus_tsdb_symbol_table_size_bytes > 100000000
    for: 5m
    annotations:
      summary: "High metric cardinality detected"
      
  - alert: MetricsScrapeFailing
    expr: up{job="swit-services"} == 0
    for: 1m
    annotations:
      summary: "Metrics scrape failing"
      
  - alert: HighMetricsMemoryUsage
    expr: go_memstats_alloc_bytes > 500000000
    for: 5m
    annotations:
      summary: "High memory usage from metrics"
```

### Dashboard Templates

Create Grafana dashboards to monitor:
- Metric collection performance
- Memory usage trends
- Cardinality growth
- Scrape success rates
- Application-specific metrics

## Getting Help

### Enable Verbose Logging

```go
// Add detailed logging to troubleshoot issues
logger := zap.NewDevelopment()
server.SetLogger(logger)
```

### Collect Debug Information

```bash
#!/bin/bash
# Debug information collection script

echo "=== Service Status ==="
ps aux | grep your-service

echo "=== Configuration ==="
your-service --show-config

echo "=== Metrics Endpoint ==="
curl -s http://localhost:8080/metrics | head -20

echo "=== Memory Usage ==="
top -p $(pgrep your-service) -n 1 -b

echo "=== Network Connections ==="
netstat -tlnp | grep :8080
```

### Contact Support

When reporting issues, include:
1. Service configuration (YAML)
2. Error messages and logs
3. Prometheus configuration
4. Network topology
5. Expected vs. actual behavior
6. Steps to reproduce

For framework bugs or feature requests, create issues at:
- Framework issues: `github.com/innovationmech/swit/issues`
- Documentation improvements: `github.com/innovationmech/swit/wiki`

## Best Practices for Prevention

1. **Start with minimal metrics** and add more as needed
2. **Monitor cardinality** regularly to prevent memory issues
3. **Use appropriate metric types** (counter, gauge, histogram)
4. **Implement proper error handling** for metrics collection
5. **Test metrics configuration** in non-production environments
6. **Document custom metrics** for team members
7. **Set up alerting** for metrics health
8. **Regular cleanup** of unused metrics
9. **Performance testing** with metrics enabled
10. **Version control** metrics configuration
