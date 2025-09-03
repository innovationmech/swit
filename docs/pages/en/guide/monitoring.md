# Monitoring and Observability

The Swit framework provides comprehensive monitoring and observability capabilities through two main systems: **Sentry** for error monitoring and **Prometheus** for metrics collection. This guide covers everything from basic setup to advanced configuration and troubleshooting.

## Overview

Swit's monitoring system includes:

### Error Monitoring (Sentry)
- **Automatic Error Capture**: HTTP and gRPC errors are automatically captured with full context
- **Performance Monitoring**: Request tracing, transaction monitoring, and performance metrics
- **Smart Filtering**: Configurable error filtering to reduce noise (ignores 4xx errors by default)
- **Context Enrichment**: Request metadata, user context, and custom tags in error reports
- **Panic Recovery**: Automatic panic capture and recovery with detailed stack traces

### Metrics Collection (Prometheus)
- **Built-in Metrics**: HTTP/gRPC request metrics, server performance, and transport status
- **Custom Metrics**: Counters, gauges, histograms with configurable labels and buckets
- **Business Metrics**: High-level business logic metrics with event hooks
- **Cardinality Control**: Automatic cardinality limiting to prevent metric explosion
- **Performance Optimized**: <5% overhead with >10k RPS capability
- **Zero-Config Defaults**: Works out-of-the-box with sensible production defaults

## Quick Start

This section covers both Sentry (error monitoring) and Prometheus (metrics collection) setup.

### Sentry Setup

#### 1. Basic Configuration

Add Sentry configuration to your service config file:

```yaml
# swit.yaml
service_name: "my-service"

sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"  # Set via environment variable
  environment: "production"
  sample_rate: 1.0
  traces_sample_rate: 0.1
```

#### 2. Environment Setup

Set your Sentry DSN:

```bash
export SENTRY_DSN="https://your-dsn@sentry.io/your-project-id"
```

#### 3. Framework Integration

The framework handles Sentry automatically:

```go
package main

import (
    "context"
    "log"
    "github.com/innovationmech/swit/pkg/server"
)

func main() {
    config := server.NewServerConfig()
    config.ServiceName = "my-service"
    
    // Sentry will be automatically initialized if enabled in config
    baseServer, err := server.NewBusinessServerCore(config, myService, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start server - Sentry monitoring begins automatically
    baseServer.Start(context.Background())
}
```

That's it! Your service now has comprehensive error monitoring and performance tracking.

### Prometheus Setup

#### 1. Basic Configuration

Add Prometheus configuration to your service config file:

```yaml
# swit.yaml
service_name: "my-service"

prometheus:
  enabled: true
  endpoint: "/metrics"
  namespace: "myapp"
  subsystem: "server" 
  buckets:
    duration: [0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10]
    size: [100, 1000, 10000, 100000, 1000000]
  cardinality_limit: 10000
```

#### 2. Framework Integration

The framework automatically sets up Prometheus metrics:

```go
package main

import (
    "context"
    "log"
    "github.com/innovationmech/swit/pkg/server"
)

func main() {
    config := server.NewServerConfig()
    config.ServiceName = "my-service"
    
    // Prometheus metrics will be automatically enabled
    baseServer, err := server.NewBusinessServerCore(config, myService, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start server - metrics collection begins automatically
    // Metrics endpoint available at http://localhost:8080/metrics
    baseServer.Start(context.Background())
}
```

#### 3. Accessing Metrics

Once running, metrics are available at the `/metrics` endpoint:

```bash
curl http://localhost:8080/metrics

# Example output:
# HELP swit_server_http_requests_total Total number of HTTP requests
# TYPE swit_server_http_requests_total counter
# swit_server_http_requests_total{method="GET",endpoint="/api/users",status="200"} 42
# 
# HELP swit_server_http_request_duration_seconds HTTP request duration in seconds  
# TYPE swit_server_http_request_duration_seconds histogram
# swit_server_http_request_duration_seconds_bucket{method="GET",endpoint="/api/users",le="0.1"} 35
# swit_server_http_request_duration_seconds_bucket{method="GET",endpoint="/api/users",le="0.5"} 40
```

## Sentry Advanced Configuration

### Complete Configuration Reference

```yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "production"
  release: "v1.2.3"
  sample_rate: 1.0              # Error sampling rate (0.0-1.0)
  traces_sample_rate: 0.1       # Performance sampling rate
  attach_stacktrace: true
  enable_tracing: true
  debug: false                  # Enable for troubleshooting
  server_name: "my-server-01"
  
  # Custom tags added to all events
  tags:
    service: "user-management"
    version: "1.2.3"
    datacenter: "us-west"
  
  # Framework integration settings
  integrate_http: true          # Enable HTTP middleware
  integrate_grpc: true          # Enable gRPC middleware
  capture_panics: true          # Capture and recover from panics
  max_breadcrumbs: 30          # Maximum breadcrumb trail length
  
  # Error filtering (reduce noise)
  ignore_errors:
    - "connection timeout"
    - "user not found"
  
  # HTTP-specific filtering
  http_ignore_paths:
    - "/health"
    - "/metrics"
    - "/favicon.ico"
  
  # HTTP status code filtering
  http_ignore_status_codes:
    - 404    # Not found errors
    - 400    # Bad request errors
  
  # Performance monitoring
  enable_profiling: true
  profiles_sample_rate: 0.1
  
  # Context and breadcrumbs
  max_request_body_size: 1024   # Max request body to capture (bytes)
  send_default_pii: false       # Don't send personally identifiable info
```

### Environment-Specific Configuration

Different environments typically need different settings:

#### Development
```yaml
sentry:
  enabled: true
  debug: true
  sample_rate: 1.0
  traces_sample_rate: 1.0       # Capture all traces in dev
  environment: "development"
```

#### Staging
```yaml
sentry:
  enabled: true
  sample_rate: 1.0
  traces_sample_rate: 0.5       # 50% performance sampling
  environment: "staging"
```

#### Production
```yaml
sentry:
  enabled: true
  sample_rate: 1.0              # Capture all errors
  traces_sample_rate: 0.1       # 10% performance sampling
  environment: "production"
  enable_profiling: true
```

## Prometheus Advanced Configuration

### Complete Configuration Reference

```yaml
prometheus:
  enabled: true
  endpoint: "/metrics"
  namespace: "myapp"           # Metric name prefix
  subsystem: "server"          # Metric name subsystem
  
  # Histogram bucket configurations
  buckets:
    # For duration metrics (seconds)
    duration: [0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10]
    # For size metrics (bytes)
    size: [100, 1000, 10000, 100000, 1000000]
  
  # Global labels applied to all metrics
  labels:
    service: "user-service"
    version: "v1.2.3"
    datacenter: "us-west"
  
  # Cardinality control
  cardinality_limit: 10000     # Max unique label combinations
```

### Built-in Metrics

The framework automatically collects these metrics:

#### HTTP Metrics
- `http_requests_total` - Total HTTP requests by method, endpoint, status
- `http_request_duration_seconds` - Request duration histogram 
- `http_request_size_bytes` - Request body size histogram
- `http_response_size_bytes` - Response body size histogram
- `http_active_requests` - Current number of active requests

#### gRPC Metrics  
- `grpc_server_started_total` - Total gRPC calls started
- `grpc_server_handled_total` - Total gRPC calls completed
- `grpc_server_handling_seconds` - gRPC call duration histogram
- `grpc_server_msg_received_total` - Total messages received
- `grpc_server_msg_sent_total` - Total messages sent

#### Server Metrics
- `server_uptime_seconds` - Server uptime
- `server_startup_duration_seconds` - Server startup time
- `server_shutdown_duration_seconds` - Server shutdown time  
- `server_goroutines` - Number of goroutines
- `server_memory_bytes` - Memory usage by type
- `server_start_time` - Server start timestamp

#### Transport Metrics
- `transport_status` - Transport health status (1=up, 0=down)
- `transport_connections_active` - Active connections
- `transport_connections_total` - Total connections
- `active_transports` - Number of active transports

### Custom Business Metrics

Add custom metrics for your business logic:

```go
import (
    "github.com/innovationmech/swit/pkg/server"
    "github.com/innovationmech/swit/pkg/types"
)

type OrderService struct {
    metricsManager *server.BusinessMetricsManager
}

func NewOrderService(bmm *server.BusinessMetricsManager) *OrderService {
    // Register custom metric definitions
    bmm.RegisterCustomMetric(server.MetricDefinition{
        Name:        "orders_processed_total",
        Type:        types.CounterType,
        Description: "Total orders processed",
        Labels:      []string{"status", "payment_method", "region"},
    })
    
    bmm.RegisterCustomMetric(server.MetricDefinition{
        Name:        "order_value_dollars",
        Type:        types.HistogramType,
        Description: "Order value distribution in dollars",
        Labels:      []string{"region"},
        Buckets:     []float64{10, 50, 100, 500, 1000, 5000},
    })
    
    return &OrderService{metricsManager: bmm}
}

func (s *OrderService) ProcessOrder(order *Order) error {
    // Record business metrics
    s.metricsManager.RecordCounter("orders_processed_total", 1.0, map[string]string{
        "status":         "success",
        "payment_method": order.PaymentMethod,
        "region":         order.Region,
    })
    
    s.metricsManager.RecordHistogram("order_value_dollars", order.Value, map[string]string{
        "region": order.Region,
    })
    
    // Track processing queue depth
    s.metricsManager.RecordGauge("order_queue_depth", float64(s.getQueueDepth()), nil)
    
    return nil
}
```

### Metrics Hooks and Event Processing

Set up hooks to process metric events:

```go
import "github.com/innovationmech/swit/pkg/server"

func setupMetricsHooks(bmm *server.BusinessMetricsManager) {
    // Add logging hook
    loggingHook := server.NewLoggingBusinessMetricsHook()
    bmm.RegisterHook(loggingHook)
    
    // Add aggregation hook for dashboards
    aggregationHook := server.NewAggregationBusinessMetricsHook()
    bmm.RegisterHook(aggregationHook)
    
    // Custom hook for alerting
    alertingHook := &AlertingMetricsHook{
        alertManager: myAlertManager,
    }
    bmm.RegisterHook(alertingHook)
}

type AlertingMetricsHook struct {
    name         string
    alertManager AlertManager
}

func (a *AlertingMetricsHook) OnMetricRecorded(event server.BusinessMetricEvent) {
    // Send alerts for high error rates
    if event.Name == "http_requests_total" && 
       event.Labels["status"] >= "500" {
        a.alertManager.SendAlert("High error rate detected", event)
    }
    
    // Alert on queue depth
    if event.Name == "order_queue_depth" && event.Value.(float64) > 1000 {
        a.alertManager.SendAlert("Queue depth critical", event)
    }
}

func (a *AlertingMetricsHook) GetHookName() string {
    return "alerting_hook"
}
```

### Environment-Specific Configuration

#### Development
```yaml
prometheus:
  enabled: true
  endpoint: "/metrics"
  namespace: "dev"
  cardinality_limit: 1000      # Lower limit for dev
```

#### Staging  
```yaml
prometheus:
  enabled: true
  endpoint: "/metrics"  
  namespace: "staging"
  cardinality_limit: 5000
  labels:
    environment: "staging"
```

#### Production
```yaml  
prometheus:
  enabled: true
  endpoint: "/metrics"
  namespace: "prod"
  cardinality_limit: 10000
  labels:
    environment: "production"
    service: "user-service"
    version: "${SERVICE_VERSION}"
```

### Integration with Prometheus Server

#### Prometheus Server Configuration

Add your service to `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  
scrape_configs:
  - job_name: 'swit-services'
    static_configs:
      - targets: ['localhost:8080']  # Your service
    scrape_interval: 10s
    metrics_path: /metrics
    
    # Relabel metrics to add instance info
    relabel_configs:
      - source_labels: [__address__]
        target_label: instance
      - source_labels: [__address__]
        regex: '([^:]+):.*'
        target_label: host
        replacement: '${1}'
```

#### Service Discovery with Consul

If using Consul service discovery:

```yaml
scrape_configs:
  - job_name: 'swit-services-consul'
    consul_sd_configs:
      - server: 'localhost:8500'
        services: ['user-service', 'order-service']
    
    relabel_configs:
      - source_labels: [__meta_consul_service]
        target_label: job
      - source_labels: [__meta_consul_node]
        target_label: instance
```

### Performance Optimization

#### Cardinality Management

Prevent metric explosion with cardinality limits:

```go
// Good: Limited, predictable labels
metricsManager.RecordCounter("api_requests_total", 1.0, map[string]string{
    "method":   "GET",           // ~10 values
    "endpoint": "/api/users",    // ~50 values  
    "status":   "200",           // ~20 values
    // Total combinations: 10 * 50 * 20 = 10,000
})

// Bad: Unlimited cardinality
metricsManager.RecordCounter("api_requests_total", 1.0, map[string]string{
    "method":    "GET",
    "user_id":   "123456",       // Millions of unique values!
    "request_id": uuid.New(),    // Always unique!
    // This will quickly hit cardinality limits
})
```

#### Performance Benchmarks

Expected performance characteristics:

- **Throughput**: >10,000 operations per second
- **Latency**: <1ms per metric operation  
- **Memory**: <1KB per unique metric series
- **CPU Overhead**: <5% of total application CPU
- **Endpoint Response**: <100ms for `/metrics`

#### Memory Management

```go
// Reset metrics collector to free memory
collector.Reset()

// Or create new collector for clean slate
newCollector := types.NewPrometheusMetricsCollector(config)
```

## Sentry Performance Monitoring

### Transaction Tracking

The framework automatically creates transactions for:

- **HTTP Requests**: Each HTTP request becomes a transaction
- **gRPC Calls**: Each gRPC method call is tracked
- **Custom Operations**: You can create custom transactions

Example of custom transaction:

```go
import "github.com/getsentry/sentry-go"

func processOrder(orderID string) error {
    // Create custom transaction
    transaction := sentry.StartTransaction(
        context.Background(), 
        "process-order",
    )
    defer transaction.Finish()
    
    // Add custom data
    transaction.SetTag("order_id", orderID)
    transaction.SetData("operation", "order_processing")
    
    // Your business logic here
    if err := validateOrder(orderID); err != nil {
        transaction.SetStatus(sentry.SpanStatusInvalidArgument)
        return err
    }
    
    return nil
}
```

### Custom Performance Spans

Create detailed performance spans:

```go
func (h *OrderHandler) CreateOrder(c *gin.Context) {
    span := sentry.StartSpan(c.Request.Context(), "database.query")
    span.SetTag("table", "orders")
    defer span.Finish()
    
    // Database operation
    order, err := h.db.CreateOrder(order)
    if err != nil {
        span.SetStatus(sentry.SpanStatusInternalError)
        sentry.CaptureException(err)
        c.JSON(500, gin.H{"error": "Failed to create order"})
        return
    }
    
    span.SetData("order_id", order.ID)
    c.JSON(201, order)
}
```

## Error Handling Best Practices

### Custom Error Context

Add rich context to errors:

```go
func (s *UserService) GetUser(id string) (*User, error) {
    sentry.ConfigureScope(func(scope *sentry.Scope) {
        scope.SetTag("service", "user-service")
        scope.SetContext("user_lookup", map[string]interface{}{
            "user_id": id,
            "timestamp": time.Now(),
        })
    })
    
    user, err := s.repository.FindByID(id)
    if err != nil {
        // This error will include the context above
        sentry.CaptureException(err)
        return nil, err
    }
    
    return user, nil
}
```

### Panic Recovery

The framework automatically recovers from panics, but you can also handle them manually:

```go
func riskyOperation() {
    defer func() {
        if err := recover(); err != nil {
            // Add custom context before reporting
            sentry.WithScope(func(scope *sentry.Scope) {
                scope.SetLevel(sentry.LevelFatal)
                scope.SetContext("panic_context", map[string]interface{}{
                    "operation": "risky_operation",
                    "timestamp": time.Now(),
                })
                sentry.CaptureException(fmt.Errorf("panic: %v", err))
            })
            // Re-panic if needed
            panic(err)
        }
    }()
    
    // Risky code here
}
```

## Testing with Sentry

### Mock Configuration for Testing

Disable Sentry in tests:

```go
func TestMyService(t *testing.T) {
    config := &server.ServerConfig{
        ServiceName: "test-service",
        Sentry: server.SentryConfig{
            Enabled: false, // Disable for testing
        },
    }
    
    // Your test code here
}
```

### Integration Testing

Test Sentry integration with mock DSN:

```go
func TestSentryIntegration(t *testing.T) {
    config := &server.ServerConfig{
        ServiceName: "test-service",
        Sentry: server.SentryConfig{
            Enabled: true,
            DSN: "http://public@example.com/1", // Mock DSN
            Debug: true,
        },
    }
    
    server, err := server.NewBusinessServerCore(config, service, nil)
    assert.NoError(t, err)
    
    // Test error capture
    err = errors.New("test error")
    sentry.CaptureException(err)
    
    // Flush events for testing
    sentry.Flush(time.Second * 2)
}
```

## Troubleshooting

### Common Issues

#### 1. Events Not Appearing in Sentry

**Check DSN Configuration:**
```bash
# Verify DSN is set
echo $SENTRY_DSN

# Test DSN format
curl -X POST "${SENTRY_DSN%/*}/api/${SENTRY_DSN##*/}/store/" \
  -H "Content-Type: application/json" \
  -d '{"message":"test"}'
```

**Enable Debug Mode:**
```yaml
sentry:
  debug: true  # Enable debug logging
```

#### 2. Too Many Events

**Adjust Sampling Rates:**
```yaml
sentry:
  sample_rate: 0.1        # Reduce error sampling
  traces_sample_rate: 0.05 # Reduce performance sampling
```

**Add More Filters:**
```yaml
sentry:
  ignore_errors:
    - "timeout"
    - "connection refused"
  http_ignore_status_codes:
    - 404
    - 400
    - 401
```

#### 3. Missing Context

**Verify Middleware Registration:**
```go
// Framework handles this automatically, but verify in logs
log.Info("HTTP Sentry middleware registered")
log.Info("gRPC Sentry middleware registered")
```

### Debug Information

Enable debug mode to see what Sentry is doing:

```yaml
sentry:
  debug: true
```

This will log:
- Event capture attempts
- Sampling decisions  
- Transport issues
- Configuration problems

## Best Practices

### 1. Production Configuration

- Use environment variables for sensitive data
- Set appropriate sampling rates
- Enable error filtering
- Configure meaningful tags and context

### 2. Development Workflow

- Use debug mode during development
- Test with mock DSN first
- Verify error capture in staging
- Monitor performance impact

### 3. Error Management

- Don't capture expected errors (4xx HTTP)
- Add meaningful context to errors
- Use appropriate severity levels
- Implement proper error boundaries

### 4. Performance Optimization

- Use low sampling rates in production
- Filter out health checks and metrics endpoints  
- Monitor Sentry SDK overhead
- Use async transport when possible

## Migration from Custom Error Handling

If you're migrating from custom error handling:

1. **Keep Existing Logs**: Sentry complements, doesn't replace logging
2. **Gradual Rollout**: Start with low sampling rates
3. **Context Migration**: Move custom context to Sentry tags/data
4. **Alert Migration**: Gradually move alerts to Sentry

Example migration:

```go
// Before: Custom error handling
func (h *Handler) ProcessRequest(c *gin.Context) {
    if err := h.service.Process(); err != nil {
        h.logger.Error("Process failed", 
            zap.Error(err),
            zap.String("request_id", c.GetHeader("X-Request-ID")),
        )
        c.JSON(500, gin.H{"error": "Internal error"})
        return
    }
}

// After: With Sentry integration
func (h *Handler) ProcessRequest(c *gin.Context) {
    if err := h.service.Process(); err != nil {
        // Keep existing logging
        h.logger.Error("Process failed", zap.Error(err))
        
        // Add Sentry context
        sentry.WithScope(func(scope *sentry.Scope) {
            scope.SetTag("request_id", c.GetHeader("X-Request-ID"))
            scope.SetContext("request", map[string]interface{}{
                "method": c.Request.Method,
                "path":   c.Request.URL.Path,
            })
            sentry.CaptureException(err)
        })
        
        c.JSON(500, gin.H{"error": "Internal error"})
        return
    }
}
```

## Related Topics

- [Configuration Guide](/en/guide/configuration) - Server and service configuration
- [Testing Guide](/en/guide/testing) - Testing strategies and patterns
- [Performance Guide](/en/guide/performance) - Performance optimization
- [Troubleshooting](/en/guide/troubleshooting) - Common issues and solutions
