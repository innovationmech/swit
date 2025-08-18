# Performance Monitoring

This guide covers the built-in performance monitoring system in Swit, including metrics collection, profiling, threshold monitoring, and integration with external monitoring systems.

## Overview

The Swit framework includes comprehensive performance monitoring through the `PerformanceMonitor` system, providing metrics collection, event-driven monitoring hooks, operation profiling, and threshold-based alerting.

## Core Components

### PerformanceMetrics Structure

The framework collects comprehensive performance metrics:

```go
type PerformanceMetrics struct {
    StartupTime    time.Duration // Server startup time
    ShutdownTime   time.Duration // Server shutdown time
    Uptime         time.Duration // Current uptime
    MemoryUsage    uint64        // Current memory usage in bytes
    GoroutineCount int           // Active goroutines
    ServiceCount   int           // Registered services
    TransportCount int           // Active transports
    StartCount     int64         // Start operations count
    RequestCount   int64         // Request counter
    ErrorCount     int64         // Error counter
}
```

### PerformanceMonitor Interface

Event-driven performance monitoring:

```go
type PerformanceMonitor interface {
    AddHook(hook PerformanceHook)
    RecordEvent(event string)
    ProfileOperation(name string, operation func() error) error
    StartPeriodicCollection(ctx context.Context, interval time.Duration)
    GetSnapshot() *PerformanceMetrics
}
```

### PerformanceHook

Custom monitoring hooks for extending functionality:

```go
type PerformanceHook func(event string, metrics *PerformanceMetrics)
```

## Basic Performance Monitoring

### Accessing Performance Metrics

```go
// Get server instance
srv, err := server.NewBusinessServerCore(config, registrar, deps)
if err != nil {
    return err
}

// Access performance monitoring
monitor := srv.(server.BusinessServerWithPerformance).GetPerformanceMonitor()
metrics := srv.(server.BusinessServerWithPerformance).GetPerformanceMetrics()

// Print current metrics
fmt.Printf("Uptime: %v\n", srv.GetUptime())
fmt.Printf("Memory Usage: %d bytes\n", metrics.MemoryUsage)
fmt.Printf("Goroutines: %d\n", metrics.GoroutineCount)
fmt.Printf("Services: %d\n", metrics.ServiceCount)
```

### Recording Custom Events

```go
// Record custom events
monitor.RecordEvent("user_registration")
monitor.RecordEvent("payment_processed")
monitor.RecordEvent("cache_miss")
monitor.RecordEvent("external_api_call")

// Events are automatically timestamped and can trigger hooks
```

### Profiling Operations

```go
// Profile database operations
err := monitor.ProfileOperation("user_query", func() error {
    users, err := userRepository.GetAllUsers(ctx)
    return err
})

// Profile external API calls
err = monitor.ProfileOperation("payment_api", func() error {
    result, err := paymentClient.ProcessPayment(ctx, payment)
    return err
})

// Profile complex business operations
err = monitor.ProfileOperation("order_processing", func() error {
    return orderService.ProcessOrder(ctx, order)
})
```

## Built-in Performance Hooks

### Logging Hook

Logs performance events and metrics:

```go
func PerformanceLoggingHook(event string, metrics *PerformanceMetrics) {
    logger.Info("Performance event",
        zap.String("event", event),
        zap.Duration("uptime", metrics.Uptime),
        zap.Uint64("memory_mb", metrics.MemoryUsage/1024/1024),
        zap.Int("goroutines", metrics.GoroutineCount),
        zap.Int64("requests", metrics.RequestCount),
        zap.Int64("errors", metrics.ErrorCount),
    )
}

// Register the hook
monitor.AddHook(PerformanceLoggingHook)
```

### Threshold Violation Hook

Alerts when performance thresholds are exceeded:

```go
func PerformanceThresholdViolationHook(event string, metrics *PerformanceMetrics) {
    const (
        MaxMemoryMB     = 512
        MaxGoroutines   = 1000
        MaxErrorRate    = 0.05 // 5%
    )
    
    memoryMB := metrics.MemoryUsage / 1024 / 1024
    if memoryMB > MaxMemoryMB {
        alerting.SendAlert(fmt.Sprintf("High memory usage: %d MB", memoryMB))
    }
    
    if metrics.GoroutineCount > MaxGoroutines {
        alerting.SendAlert(fmt.Sprintf("High goroutine count: %d", metrics.GoroutineCount))
    }
    
    if metrics.RequestCount > 0 {
        errorRate := float64(metrics.ErrorCount) / float64(metrics.RequestCount)
        if errorRate > MaxErrorRate {
            alerting.SendAlert(fmt.Sprintf("High error rate: %.2f%%", errorRate*100))
        }
    }
}

monitor.AddHook(PerformanceThresholdViolationHook)
```

### Metrics Collection Hook

Triggers metrics collection at intervals:

```go
func PerformanceMetricsCollectionHook(event string, metrics *PerformanceMetrics) {
    // Send to Prometheus
    prometheusMetrics.SetGauge("server_uptime_seconds", metrics.Uptime.Seconds())
    prometheusMetrics.SetGauge("server_memory_bytes", float64(metrics.MemoryUsage))
    prometheusMetrics.SetGauge("server_goroutines", float64(metrics.GoroutineCount))
    prometheusMetrics.IncrementCounter("server_requests_total", float64(metrics.RequestCount))
    prometheusMetrics.IncrementCounter("server_errors_total", float64(metrics.ErrorCount))
    
    // Send to InfluxDB
    influxPoint := influxdb2.NewPoint("server_metrics",
        map[string]string{"service": "my-service"},
        map[string]interface{}{
            "uptime_seconds":   metrics.Uptime.Seconds(),
            "memory_bytes":     metrics.MemoryUsage,
            "goroutine_count":  metrics.GoroutineCount,
            "service_count":    metrics.ServiceCount,
            "transport_count":  metrics.TransportCount,
            "request_count":    metrics.RequestCount,
            "error_count":      metrics.ErrorCount,
        },
        time.Now(),
    )
    influxWriter.WritePoint(point)
}

monitor.AddHook(PerformanceMetricsCollectionHook)
```

## Custom Performance Monitoring

### Custom Metrics Hook

```go
type CustomMetrics struct {
    DatabaseConnections int
    CacheHitRate       float64
    ExternalAPILatency time.Duration
    ActiveSessions     int
}

func CustomMetricsHook(event string, metrics *PerformanceMetrics) {
    custom := &CustomMetrics{
        DatabaseConnections: getDatabaseConnectionCount(),
        CacheHitRate:       getCacheHitRate(),
        ExternalAPILatency: getAverageAPILatency(),
        ActiveSessions:     getActiveSessionCount(),
    }
    
    // Send custom metrics to monitoring system
    monitoring.RecordMetric("db_connections", float64(custom.DatabaseConnections))
    monitoring.RecordMetric("cache_hit_rate", custom.CacheHitRate)
    monitoring.RecordMetric("api_latency_ms", custom.ExternalAPILatency.Milliseconds())
    monitoring.RecordMetric("active_sessions", float64(custom.ActiveSessions))
}

monitor.AddHook(CustomMetricsHook)
```

### Business Logic Performance Hook

```go
func BusinessMetricsHook(event string, metrics *PerformanceMetrics) {
    switch event {
    case "user_registration":
        userRegistrationCounter.Inc()
        
    case "order_processed":
        orderProcessingCounter.Inc()
        revenueGauge.Add(getLastOrderValue())
        
    case "payment_failed":
        paymentFailureCounter.Inc()
        alerting.SendAlert("Payment processing failure detected")
        
    case "cache_miss":
        cacheMissCounter.Inc()
        
    case "database_slow_query":
        slowQueryCounter.Inc()
        if slowQueryCounter.Get() > 10 {
            alerting.SendAlert("High number of slow database queries")
        }
    }
}

monitor.AddHook(BusinessMetricsHook)
```

## Periodic Metrics Collection

### Starting Periodic Collection

```go
// Start periodic metrics collection every 30 seconds
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

monitor.StartPeriodicCollection(ctx, 30*time.Second)

// The periodic collection will trigger hooks with "metrics_collection" event
```

### Custom Periodic Collector

```go
func startCustomPeriodicCollection(ctx context.Context, monitor *server.PerformanceMonitor) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Record custom events
            monitor.RecordEvent("periodic_health_check")
            
            // Perform custom metrics collection
            collectCustomMetrics()
            
            // Check system health
            checkSystemHealth()
        }
    }
}

func collectCustomMetrics() {
    // Collect database metrics
    dbStats := database.Stats()
    prometheusMetrics.SetGauge("db_open_connections", float64(dbStats.OpenConnections))
    prometheusMetrics.SetGauge("db_in_use", float64(dbStats.InUse))
    prometheusMetrics.SetGauge("db_idle", float64(dbStats.Idle))
    
    // Collect cache metrics
    cacheStats := cache.Stats()
    prometheusMetrics.SetGauge("cache_hit_rate", cacheStats.HitRate)
    prometheusMetrics.SetGauge("cache_memory_usage", float64(cacheStats.MemoryUsage))
    
    // Collect business metrics
    prometheusMetrics.SetGauge("active_users", float64(getUserCount()))
    prometheusMetrics.SetGauge("pending_orders", float64(getPendingOrderCount()))
}
```

## Integration with External Monitoring

### Prometheus Integration

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Define Prometheus metrics
var (
    serverUptime = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "server_uptime_seconds",
        Help: "Server uptime in seconds",
    })
    
    memoryUsage = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "server_memory_bytes",
        Help: "Server memory usage in bytes",
    })
    
    requestCounter = promauto.NewCounter(prometheus.CounterOpts{
        Name: "server_requests_total",
        Help: "Total number of requests processed",
    })
    
    operationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name: "operation_duration_seconds",
        Help: "Duration of operations in seconds",
        Buckets: prometheus.DefBuckets,
    }, []string{"operation"})
)

// Prometheus hook
func PrometheusHook(event string, metrics *PerformanceMetrics) {
    serverUptime.Set(metrics.Uptime.Seconds())
    memoryUsage.Set(float64(metrics.MemoryUsage))
    requestCounter.Add(float64(metrics.RequestCount))
    
    // Record operation durations
    if strings.HasPrefix(event, "operation_") {
        operation := strings.TrimPrefix(event, "operation_")
        // Duration would need to be passed through event context
        operationDuration.WithLabelValues(operation).Observe(/* duration */)
    }
}

// Start Prometheus metrics endpoint
go func() {
    http.Handle("/metrics", promhttp.Handler())
    log.Fatal(http.ListenAndServe(":2112", nil))
}()

monitor.AddHook(PrometheusHook)
```

### InfluxDB Integration

```go
import (
    influxdb2 "github.com/influxdata/influxdb-client-go/v2"
    "github.com/influxdata/influxdb-client-go/v2/api"
)

type InfluxDBMonitor struct {
    client influxdb2.Client
    writer api.WriteAPI
}

func NewInfluxDBMonitor(url, token, org, bucket string) *InfluxDBMonitor {
    client := influxdb2.NewClient(url, token)
    writer := client.WriteAPI(org, bucket)
    
    return &InfluxDBMonitor{
        client: client,
        writer: writer,
    }
}

func (im *InfluxDBMonitor) Hook(event string, metrics *PerformanceMetrics) {
    point := influxdb2.NewPoint("server_performance",
        map[string]string{
            "service": "my-service",
            "event":   event,
        },
        map[string]interface{}{
            "uptime_seconds":   metrics.Uptime.Seconds(),
            "memory_bytes":     int64(metrics.MemoryUsage),
            "goroutine_count":  metrics.GoroutineCount,
            "service_count":    metrics.ServiceCount,
            "transport_count":  metrics.TransportCount,
            "request_count":    metrics.RequestCount,
            "error_count":      metrics.ErrorCount,
        },
        time.Now(),
    )
    
    im.writer.WritePoint(point)
}

// Setup InfluxDB monitoring
influxMonitor := NewInfluxDBMonitor(
    "http://influxdb:8086",
    "my-token",
    "my-org",
    "performance",
)

monitor.AddHook(influxMonitor.Hook)
```

### Custom Alerting Integration

```go
type AlertingManager struct {
    slackWebhook string
    emailSender  *EmailSender
    thresholds   *PerformanceThresholds
}

type PerformanceThresholds struct {
    MaxMemoryMB     uint64
    MaxGoroutines   int
    MaxErrorRate    float64
    MaxResponseTime time.Duration
}

func (am *AlertingManager) Hook(event string, metrics *PerformanceMetrics) {
    alerts := am.checkThresholds(metrics)
    
    for _, alert := range alerts {
        am.sendAlert(alert)
    }
}

func (am *AlertingManager) checkThresholds(metrics *PerformanceMetrics) []Alert {
    var alerts []Alert
    
    memoryMB := metrics.MemoryUsage / 1024 / 1024
    if memoryMB > am.thresholds.MaxMemoryMB {
        alerts = append(alerts, Alert{
            Level:   "warning",
            Message: fmt.Sprintf("High memory usage: %d MB", memoryMB),
            Metric:  "memory_usage",
            Value:   float64(memoryMB),
        })
    }
    
    if metrics.GoroutineCount > am.thresholds.MaxGoroutines {
        alerts = append(alerts, Alert{
            Level:   "warning",
            Message: fmt.Sprintf("High goroutine count: %d", metrics.GoroutineCount),
            Metric:  "goroutine_count",
            Value:   float64(metrics.GoroutineCount),
        })
    }
    
    return alerts
}

func (am *AlertingManager) sendAlert(alert Alert) {
    // Send to Slack
    am.sendSlackAlert(alert)
    
    // Send email for critical alerts
    if alert.Level == "critical" {
        am.emailSender.SendAlert(alert)
    }
}
```

## Performance Profiling

### CPU Profiling

```go
import (
    _ "net/http/pprof"
    "runtime/pprof"
)

// Enable pprof endpoint
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

// Programmatic CPU profiling
func profileCPU(monitor *server.PerformanceMonitor) {
    monitor.ProfileOperation("cpu_intensive_task", func() error {
        f, err := os.Create("cpu.prof")
        if err != nil {
            return err
        }
        defer f.Close()
        
        if err := pprof.StartCPUProfile(f); err != nil {
            return err
        }
        defer pprof.StopCPUProfile()
        
        // Perform CPU-intensive operation
        performCPUIntensiveTask()
        
        return nil
    })
}
```

### Memory Profiling

```go
import (
    "runtime"
    "runtime/pprof"
)

func profileMemory(monitor *server.PerformanceMonitor) {
    monitor.ProfileOperation("memory_intensive_task", func() error {
        // Force garbage collection before profiling
        runtime.GC()
        
        f, err := os.Create("mem.prof")
        if err != nil {
            return err
        }
        defer f.Close()
        
        // Perform memory-intensive operation
        performMemoryIntensiveTask()
        
        // Write heap profile
        if err := pprof.WriteHeapProfile(f); err != nil {
            return err
        }
        
        return nil
    })
}
```

### Goroutine Profiling

```go
func profileGoroutines(monitor *server.PerformanceMonitor) {
    monitor.RecordEvent("goroutine_profile_start")
    
    f, err := os.Create("goroutine.prof")
    if err != nil {
        log.Printf("Could not create goroutine profile: %v", err)
        return
    }
    defer f.Close()
    
    if err := pprof.Lookup("goroutine").WriteTo(f, 0); err != nil {
        log.Printf("Could not write goroutine profile: %v", err)
        return
    }
    
    monitor.RecordEvent("goroutine_profile_complete")
}
```

## Testing Performance Monitoring

### Unit Testing Performance Hooks

```go
func TestPerformanceHooks(t *testing.T) {
    var capturedEvents []string
    var capturedMetrics []*server.PerformanceMetrics
    
    testHook := func(event string, metrics *server.PerformanceMetrics) {
        capturedEvents = append(capturedEvents, event)
        capturedMetrics = append(capturedMetrics, metrics)
    }
    
    monitor := server.NewPerformanceMonitor()
    monitor.AddHook(testHook)
    
    // Record test events
    monitor.RecordEvent("test_event_1")
    monitor.RecordEvent("test_event_2")
    
    // Verify hook was called
    assert.Equal(t, 2, len(capturedEvents))
    assert.Contains(t, capturedEvents, "test_event_1")
    assert.Contains(t, capturedEvents, "test_event_2")
    
    // Verify metrics were captured
    assert.Equal(t, 2, len(capturedMetrics))
    assert.NotNil(t, capturedMetrics[0])
    assert.NotNil(t, capturedMetrics[1])
}
```

### Integration Testing with Metrics

```go
func TestServerPerformanceIntegration(t *testing.T) {
    config := server.NewServerConfig()
    config.HTTP.Port = "0" // Dynamic port
    config.Discovery.Enabled = false
    
    srv, err := server.NewBusinessServerCore(config, &TestServiceRegistrar{}, nil)
    require.NoError(t, err)
    
    // Access performance monitor
    monitor := srv.(server.BusinessServerWithPerformance).GetPerformanceMonitor()
    
    // Add test hook
    var events []string
    monitor.AddHook(func(event string, metrics *server.PerformanceMetrics) {
        events = append(events, event)
    })
    
    // Start server
    ctx := context.Background()
    err = srv.Start(ctx)
    require.NoError(t, err)
    defer srv.Shutdown()
    
    // Wait a moment for startup events
    time.Sleep(100 * time.Millisecond)
    
    // Verify startup events were recorded
    assert.Contains(t, events, "server_startup_complete")
    
    // Get performance metrics
    metrics := srv.(server.BusinessServerWithPerformance).GetPerformanceMetrics()
    assert.Greater(t, metrics.StartupTime, time.Duration(0))
    assert.Greater(t, metrics.MemoryUsage, uint64(0))
    assert.Greater(t, metrics.GoroutineCount, 0)
}
```

## Best Practices

### Performance Monitoring

1. **Selective Monitoring** - Monitor key metrics without overwhelming the system
2. **Threshold Setting** - Set realistic thresholds based on your system's normal behavior
3. **Event Granularity** - Balance between too many and too few events
4. **Resource Impact** - Ensure monitoring doesn't significantly impact performance
5. **Data Retention** - Configure appropriate data retention policies for metrics

### Alerting Strategy

1. **Alert Fatigue Prevention** - Avoid too many false positives
2. **Escalation Levels** - Use different alert levels (info, warning, critical)
3. **Context Information** - Include relevant context in alerts
4. **Actionable Alerts** - Ensure alerts provide actionable information
5. **Testing Alerts** - Regularly test alerting mechanisms

### Production Deployment

1. **Gradual Rollout** - Deploy monitoring changes gradually
2. **Baseline Establishment** - Establish performance baselines before making changes
3. **Monitoring the Monitor** - Monitor the monitoring system itself
4. **Documentation** - Document all custom metrics and their meanings
5. **Regular Review** - Regularly review and update monitoring configurations

This performance monitoring guide provides comprehensive coverage of the built-in performance monitoring capabilities, custom monitoring patterns, integration with external systems, and best practices for production deployment.