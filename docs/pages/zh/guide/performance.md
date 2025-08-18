# 性能监控

本指南涵盖 Swit 中的内置性能监控系统，包括指标收集、分析、阈值监控以及与外部监控系统的集成。

## 概述

Swit 框架通过 `PerformanceMonitor` 系统包含全面的性能监控，提供指标收集、事件驱动的监控钩子、操作分析和基于阈值的警报。

## 核心组件

### PerformanceMetrics 结构

框架收集全面的性能指标：

```go
type PerformanceMetrics struct {
    StartupTime    time.Duration // 服务器启动时间
    ShutdownTime   time.Duration // 服务器关闭时间
    Uptime         time.Duration // 当前运行时间
    MemoryUsage    uint64        // 当前内存使用（字节）
    GoroutineCount int           // 活跃协程数
    ServiceCount   int           // 注册服务数
    TransportCount int           // 活跃传输数
    StartCount     int64         // 启动操作计数
    RequestCount   int64         // 请求计数器
    ErrorCount     int64         // 错误计数器
}
```

### PerformanceMonitor 接口

事件驱动的性能监控：

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

用于扩展功能的自定义监控钩子：

```go
type PerformanceHook func(event string, metrics *PerformanceMetrics)
```

## 基本性能监控

### 访问性能指标

```go
// 获取服务器实例
srv, err := server.NewBusinessServerCore(config, registrar, deps)
if err != nil {
    return err
}

// 访问性能监控
monitor := srv.(server.BusinessServerWithPerformance).GetPerformanceMonitor()
metrics := srv.(server.BusinessServerWithPerformance).GetPerformanceMetrics()

// 打印当前指标
fmt.Printf("运行时间: %v\n", srv.GetUptime())
fmt.Printf("内存使用: %d 字节\n", metrics.MemoryUsage)
fmt.Printf("协程数: %d\n", metrics.GoroutineCount)
fmt.Printf("服务数: %d\n", metrics.ServiceCount)
```

### 记录自定义事件

```go
// 记录自定义事件
monitor.RecordEvent("user_registration")
monitor.RecordEvent("payment_processed")
monitor.RecordEvent("cache_miss")
monitor.RecordEvent("external_api_call")

// 事件会自动添加时间戳并可触发钩子
```

### 分析操作

```go
// 分析数据库操作
err := monitor.ProfileOperation("user_query", func() error {
    users, err := userRepository.GetAllUsers(ctx)
    return err
})

// 分析外部 API 调用
err = monitor.ProfileOperation("payment_api", func() error {
    result, err := paymentClient.ProcessPayment(ctx, payment)
    return err
})

// 分析复杂业务操作
err = monitor.ProfileOperation("order_processing", func() error {
    return orderService.ProcessOrder(ctx, order)
})
```

## 内置性能钩子

### 日志钩子

记录性能事件和指标：

```go
func PerformanceLoggingHook(event string, metrics *PerformanceMetrics) {
    logger.Info("性能事件",
        zap.String("event", event),
        zap.Duration("uptime", metrics.Uptime),
        zap.Uint64("memory_mb", metrics.MemoryUsage/1024/1024),
        zap.Int("goroutines", metrics.GoroutineCount),
        zap.Int64("requests", metrics.RequestCount),
        zap.Int64("errors", metrics.ErrorCount),
    )
}

// 注册钩子
monitor.AddHook(PerformanceLoggingHook)
```

### 阈值违规钩子

当超出性能阈值时发出警报：

```go
func PerformanceThresholdViolationHook(event string, metrics *PerformanceMetrics) {
    const (
        MaxMemoryMB     = 512
        MaxGoroutines   = 1000
        MaxErrorRate    = 0.05 // 5%
    )
    
    memoryMB := metrics.MemoryUsage / 1024 / 1024
    if memoryMB > MaxMemoryMB {
        alerting.SendAlert(fmt.Sprintf("高内存使用: %d MB", memoryMB))
    }
    
    if metrics.GoroutineCount > MaxGoroutines {
        alerting.SendAlert(fmt.Sprintf("高协程数: %d", metrics.GoroutineCount))
    }
    
    if metrics.RequestCount > 0 {
        errorRate := float64(metrics.ErrorCount) / float64(metrics.RequestCount)
        if errorRate > MaxErrorRate {
            alerting.SendAlert(fmt.Sprintf("高错误率: %.2f%%", errorRate*100))
        }
    }
}

monitor.AddHook(PerformanceThresholdViolationHook)
```

### 指标收集钩子

定期触发指标收集：

```go
func PerformanceMetricsCollectionHook(event string, metrics *PerformanceMetrics) {
    // 发送到 Prometheus
    prometheusMetrics.SetGauge("server_uptime_seconds", metrics.Uptime.Seconds())
    prometheusMetrics.SetGauge("server_memory_bytes", float64(metrics.MemoryUsage))
    prometheusMetrics.SetGauge("server_goroutines", float64(metrics.GoroutineCount))
    prometheusMetrics.IncrementCounter("server_requests_total", float64(metrics.RequestCount))
    prometheusMetrics.IncrementCounter("server_errors_total", float64(metrics.ErrorCount))
    
    // 发送到 InfluxDB
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

## 自定义性能监控

### 自定义指标钩子

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
    
    // 发送自定义指标到监控系统
    monitoring.RecordMetric("db_connections", float64(custom.DatabaseConnections))
    monitoring.RecordMetric("cache_hit_rate", custom.CacheHitRate)
    monitoring.RecordMetric("api_latency_ms", custom.ExternalAPILatency.Milliseconds())
    monitoring.RecordMetric("active_sessions", float64(custom.ActiveSessions))
}

monitor.AddHook(CustomMetricsHook)
```

## 定期指标收集

### 启动定期收集

```go
// 每 30 秒启动一次定期指标收集
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

monitor.StartPeriodicCollection(ctx, 30*time.Second)

// 定期收集将使用 "metrics_collection" 事件触发钩子
```

### 自定义定期收集器

```go
func startCustomPeriodicCollection(ctx context.Context, monitor *server.PerformanceMonitor) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // 记录自定义事件
            monitor.RecordEvent("periodic_health_check")
            
            // 执行自定义指标收集
            collectCustomMetrics()
            
            // 检查系统健康
            checkSystemHealth()
        }
    }
}

func collectCustomMetrics() {
    // 收集数据库指标
    dbStats := database.Stats()
    prometheusMetrics.SetGauge("db_open_connections", float64(dbStats.OpenConnections))
    prometheusMetrics.SetGauge("db_in_use", float64(dbStats.InUse))
    prometheusMetrics.SetGauge("db_idle", float64(dbStats.Idle))
    
    // 收集缓存指标
    cacheStats := cache.Stats()
    prometheusMetrics.SetGauge("cache_hit_rate", cacheStats.HitRate)
    prometheusMetrics.SetGauge("cache_memory_usage", float64(cacheStats.MemoryUsage))
    
    // 收集业务指标
    prometheusMetrics.SetGauge("active_users", float64(getUserCount()))
    prometheusMetrics.SetGauge("pending_orders", float64(getPendingOrderCount()))
}
```

## 与外部监控集成

### Prometheus 集成

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// 定义 Prometheus 指标
var (
    serverUptime = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "server_uptime_seconds",
        Help: "服务器运行时间（秒）",
    })
    
    memoryUsage = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "server_memory_bytes",
        Help: "服务器内存使用（字节）",
    })
    
    requestCounter = promauto.NewCounter(prometheus.CounterOpts{
        Name: "server_requests_total",
        Help: "处理的请求总数",
    })
)

// Prometheus 钩子
func PrometheusHook(event string, metrics *PerformanceMetrics) {
    serverUptime.Set(metrics.Uptime.Seconds())
    memoryUsage.Set(float64(metrics.MemoryUsage))
    requestCounter.Add(float64(metrics.RequestCount))
}

// 启动 Prometheus 指标端点
go func() {
    http.Handle("/metrics", promhttp.Handler())
    log.Fatal(http.ListenAndServe(":2112", nil))
}()

monitor.AddHook(PrometheusHook)
```

## 性能分析

### CPU 分析

```go
import (
    _ "net/http/pprof"
    "runtime/pprof"
)

// 启用 pprof 端点
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

// 程序化 CPU 分析
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
        
        // 执行 CPU 密集型操作
        performCPUIntensiveTask()
        
        return nil
    })
}
```

### 内存分析

```go
import (
    "runtime"
    "runtime/pprof"
)

func profileMemory(monitor *server.PerformanceMonitor) {
    monitor.ProfileOperation("memory_intensive_task", func() error {
        // 分析前强制垃圾收集
        runtime.GC()
        
        f, err := os.Create("mem.prof")
        if err != nil {
            return err
        }
        defer f.Close()
        
        // 执行内存密集型操作
        performMemoryIntensiveTask()
        
        // 写入堆分析
        if err := pprof.WriteHeapProfile(f); err != nil {
            return err
        }
        
        return nil
    })
}
```

## 测试性能监控

### 单元测试性能钩子

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
    
    // 记录测试事件
    monitor.RecordEvent("test_event_1")
    monitor.RecordEvent("test_event_2")
    
    // 验证钩子被调用
    assert.Equal(t, 2, len(capturedEvents))
    assert.Contains(t, capturedEvents, "test_event_1")
    assert.Contains(t, capturedEvents, "test_event_2")
    
    // 验证指标被捕获
    assert.Equal(t, 2, len(capturedMetrics))
    assert.NotNil(t, capturedMetrics[0])
    assert.NotNil(t, capturedMetrics[1])
}
```

## 最佳实践

### 性能监控

1. **选择性监控** - 监控关键指标而不使系统过载
2. **阈值设置** - 根据系统正常行为设置现实的阈值
3. **事件粒度** - 在太多和太少事件之间取得平衡
4. **资源影响** - 确保监控不会显著影响性能
5. **数据保留** - 为指标配置适当的数据保留策略

### 警报策略

1. **防止警报疲劳** - 避免太多误报
2. **升级级别** - 使用不同的警报级别（信息、警告、严重）
3. **上下文信息** - 在警报中包含相关上下文
4. **可操作的警报** - 确保警报提供可操作的信息
5. **测试警报** - 定期测试警报机制

这个性能监控指南提供了内置性能监控功能、自定义监控模式、与外部系统集成以及生产部署最佳实践的全面覆盖。