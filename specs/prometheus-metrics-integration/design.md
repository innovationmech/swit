# Prometheus Metrics Integration - Design Document

## Architecture Overview

The Prometheus metrics integration extends the existing observability infrastructure in the Swit framework to provide comprehensive metrics collection and export capabilities.

### High-Level Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP/gRPC     │    │  Observability  │    │   Prometheus    │
│   Middleware    │───▶│    Manager      │───▶│   Collector     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       ▼                       │
         │              ┌─────────────────┐              │
         │              │ Performance     │              │
         └──────────────│ Monitor         │──────────────┘
                        └─────────────────┘
                                 │
                                 ▼
                        ┌─────────────────┐
                        │   /metrics      │
                        │   Endpoint      │
                        └─────────────────┘
```

## Component Design

### 1. PrometheusMetricsCollector

**Location**: `pkg/server/prometheus_collector.go`

```go
type PrometheusMetricsCollector struct {
    counters   map[string]*prometheus.CounterVec
    gauges     map[string]*prometheus.GaugeVec
    histograms map[string]*prometheus.HistogramVec
    summaries  map[string]*prometheus.SummaryVec
    registry   *prometheus.Registry
    config     *PrometheusConfig
    mu         sync.RWMutex
}
```

**Key Features**:
- Implements existing `MetricsCollector` interface
- Thread-safe metric registration and updates
- Automatic label management
- Custom registry for isolation
- Support for all Prometheus metric types

**Methods**:
- `IncrementCounter(name string, labels map[string]string)`
- `SetGauge(name string, value float64, labels map[string]string)`
- `ObserveHistogram(name string, value float64, labels map[string]string)`
- `GetMetrics() []Metric`
- `GetHandler() http.Handler` - Returns Prometheus metrics handler

### 2. Metrics Registry

**Location**: `pkg/server/metrics_registry.go`

```go
type MetricsRegistry struct {
    predefinedMetrics map[string]MetricDefinition
    customMetrics     map[string]MetricDefinition
    mu                sync.RWMutex
}

type MetricDefinition struct {
    Name        string
    Type        MetricType
    Description string
    Labels      []string
    Buckets     []float64 // For histograms
}
```

**Predefined Metrics**:
- HTTP request metrics
- gRPC request metrics  
- Server lifecycle metrics
- Performance metrics
- Transport metrics

### 3. HTTP Middleware

**Location**: `pkg/middleware/prometheus_http.go`

```go
func PrometheusHTTPMiddleware(collector MetricsCollector) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        // Track active requests
        collector.IncrementGauge("http_active_requests", nil)
        defer collector.DecrementGauge("http_active_requests", nil)
        
        // Process request
        c.Next()
        
        // Collect metrics
        duration := time.Since(start)
        labels := map[string]string{
            "method":   c.Request.Method,
            "endpoint": c.FullPath(),
            "status":   strconv.Itoa(c.Writer.Status()),
        }
        
        collector.IncrementCounter("http_requests_total", labels)
        collector.ObserveHistogram("http_request_duration_seconds", 
            duration.Seconds(), labels)
        collector.ObserveHistogram("http_request_size_bytes", 
            float64(c.Request.ContentLength), labels)
        collector.ObserveHistogram("http_response_size_bytes", 
            float64(c.Writer.Size()), labels)
    }
}
```

### 4. gRPC Interceptors

**Location**: `pkg/middleware/prometheus_grpc.go`

```go
func UnaryServerInterceptor(collector MetricsCollector) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, 
        handler grpc.UnaryHandler) (interface{}, error) {
        
        start := time.Now()
        labels := map[string]string{"method": info.FullMethod}
        
        collector.IncrementCounter("grpc_server_started_total", labels)
        
        resp, err := handler(ctx, req)
        
        duration := time.Since(start)
        status := codes.OK
        if err != nil {
            status = codes.Internal // Simplified for example
        }
        
        labels["code"] = status.String()
        collector.IncrementCounter("grpc_server_handled_total", labels)
        collector.ObserveHistogram("grpc_server_handling_seconds", 
            duration.Seconds(), labels)
            
        return resp, err
    }
}
```

### 5. Configuration Integration

**Location**: `pkg/server/config.go`

```go
type PrometheusConfig struct {
    Enabled   bool              `yaml:"enabled" json:"enabled"`
    Endpoint  string            `yaml:"endpoint" json:"endpoint"`
    Namespace string            `yaml:"namespace" json:"namespace"`
    Subsystem string            `yaml:"subsystem" json:"subsystem"`
    Buckets   PrometheusBuckets `yaml:"buckets" json:"buckets"`
    Labels    map[string]string `yaml:"labels" json:"labels"`
}

type PrometheusBuckets struct {
    Duration []float64 `yaml:"duration" json:"duration"`
    Size     []float64 `yaml:"size" json:"size"`
}

// Add to ServerConfig
type ServerConfig struct {
    // ... existing fields
    Prometheus PrometheusConfig `yaml:"prometheus" json:"prometheus"`
}
```

**Default Configuration**:
```yaml
prometheus:
  enabled: true
  endpoint: "/metrics"
  namespace: "swit"
  subsystem: "server"
  buckets:
    duration: [0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10]
    size: [100, 1000, 10000, 100000, 1000000]
  labels:
    service: "${SERVICE_NAME}"
    environment: "${ENVIRONMENT}"
```

### 6. Metrics Endpoint

**Location**: `pkg/server/observability.go` (extension)

```go
func (om *ObservabilityManager) RegisterPrometheusEndpoint(
    router *gin.Engine, collector *PrometheusMetricsCollector) {
    
    if !om.config.Prometheus.Enabled {
        return
    }
    
    endpoint := om.config.Prometheus.Endpoint
    if endpoint == "" {
        endpoint = "/metrics"
    }
    
    router.GET(endpoint, gin.WrapH(collector.GetHandler()))
}
```

## Integration Design

### 1. ObservabilityManager Integration

```go
type ObservabilityManager struct {
    serviceName string
    metrics     *ServerMetrics
    prometheus  *PrometheusMetricsCollector // New field
    startTime   time.Time
    version     string
    buildInfo   map[string]string
}

func NewObservabilityManager(serviceName string, config *ServerConfig) *ObservabilityManager {
    var collector MetricsCollector
    
    if config.Prometheus.Enabled {
        collector = NewPrometheusMetricsCollector(&config.Prometheus)
    } else {
        collector = NewSimpleMetricsCollector()
    }
    
    return &ObservabilityManager{
        serviceName: serviceName,
        metrics:     NewServerMetrics(serviceName, collector),
        prometheus:  collector.(*PrometheusMetricsCollector), // Type assertion if Prometheus
        startTime:   time.Now(),
        buildInfo:   make(map[string]string),
    }
}
```

### 2. MiddlewareManager Integration

```go
func (m *MiddlewareManager) ConfigurePrometheusMiddleware(
    router *gin.Engine, collector MetricsCollector) error {
    
    if !m.config.Prometheus.Enabled {
        return nil
    }
    
    // Register HTTP middleware
    router.Use(PrometheusHTTPMiddleware(collector))
    
    return nil
}
```

### 3. TransportCoordinator Integration

```go
func (tc *TransportCoordinator) initializeGRPCTransport() error {
    var opts []grpc.ServerOption
    
    if tc.config.Prometheus.Enabled && tc.metricsCollector != nil {
        opts = append(opts, 
            grpc.UnaryInterceptor(UnaryServerInterceptor(tc.metricsCollector)),
            grpc.StreamInterceptor(StreamServerInterceptor(tc.metricsCollector)))
    }
    
    tc.grpcServer = grpc.NewServer(opts...)
    return nil
}
```

## Metric Definitions

### HTTP Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|---------|-------------|
| `http_requests_total` | Counter | method, endpoint, status | Total HTTP requests |
| `http_request_duration_seconds` | Histogram | method, endpoint | HTTP request duration |
| `http_request_size_bytes` | Histogram | method, endpoint | HTTP request size |
| `http_response_size_bytes` | Histogram | method, endpoint | HTTP response size |
| `http_active_requests` | Gauge | - | Currently active HTTP requests |

### gRPC Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|---------|-------------|
| `grpc_server_started_total` | Counter | method | Started gRPC calls |
| `grpc_server_handled_total` | Counter | method, code | Completed gRPC calls |
| `grpc_server_handling_seconds` | Histogram | method | gRPC call duration |
| `grpc_server_msg_received_total` | Counter | method | Messages received |
| `grpc_server_msg_sent_total` | Counter | method | Messages sent |

### Server Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|---------|-------------|
| `server_uptime_seconds` | Gauge | service | Server uptime |
| `server_startup_duration_seconds` | Histogram | service | Server startup time |
| `server_shutdown_duration_seconds` | Histogram | service | Server shutdown time |
| `server_goroutines` | Gauge | service | Number of goroutines |
| `server_memory_bytes` | Gauge | service, type | Memory usage |
| `server_gc_duration_seconds` | Gauge | service | GC duration |

### Transport Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|---------|-------------|
| `transport_status` | Gauge | transport, status | Transport status |
| `transport_connections_active` | Gauge | transport | Active connections |
| `transport_connections_total` | Counter | transport | Total connections |

## Performance Considerations

### 1. Memory Efficiency
- Use metric pools for frequently updated metrics
- Implement metric cleanup for dynamic labels
- Configure reasonable cardinality limits

### 2. CPU Efficiency
- Minimize allocations in hot paths
- Use atomic operations for counters
- Batch metric updates where possible

### 3. Cardinality Control
```go
type CardinalityLimiter struct {
    maxCardinality int
    currentMetrics map[string]int
    mu             sync.RWMutex
}

func (cl *CardinalityLimiter) CheckCardinality(metricName string, 
    labels map[string]string) bool {
    // Implementation to prevent cardinality explosion
}
```

## Security Design

### 1. Metrics Endpoint Security
```go
func (om *ObservabilityManager) RegisterSecureMetricsEndpoint(
    router *gin.Engine, authMiddleware gin.HandlerFunc) {
    
    metricsGroup := router.Group(om.config.Prometheus.Endpoint)
    if authMiddleware != nil {
        metricsGroup.Use(authMiddleware)
    }
    
    metricsGroup.GET("", gin.WrapH(om.prometheus.GetHandler()))
}
```

### 2. Label Sanitization
```go
func sanitizeLabel(value string) string {
    // Remove sensitive information
    // Limit label length
    // Escape special characters
    return sanitized
}
```

## Error Handling

### 1. Graceful Degradation
- Metrics collection failures should not impact service functionality
- Log metrics errors at debug level to avoid noise
- Provide circuit breaker for metrics collection

### 2. Recovery Mechanisms
```go
func (pmc *PrometheusMetricsCollector) safeMetricOperation(
    operation func() error) {
    
    defer func() {
        if r := recover(); r != nil {
            logger.Logger.Debug("Metrics operation panicked", 
                zap.Any("panic", r))
        }
    }()
    
    if err := operation(); err != nil {
        logger.Logger.Debug("Metrics operation failed", 
            zap.Error(err))
    }
}
```

## Testing Strategy

### 1. Unit Tests
- Test metric registration and updates
- Test label handling and sanitization
- Test configuration parsing and validation
- Test error handling and recovery

### 2. Integration Tests
- Test middleware integration with actual HTTP/gRPC requests
- Test metrics endpoint exposure
- Test metric collection across transports

### 3. Performance Tests
- Benchmark metrics collection overhead
- Test under high load conditions
- Memory usage profiling

### 4. End-to-End Tests
- Test complete metrics pipeline
- Test Prometheus scraping
- Test with real monitoring stack

## Migration Strategy

### 1. Backward Compatibility
- Existing `SimpleMetricsCollector` remains default
- Prometheus enabled via configuration
- No breaking changes to existing APIs

### 2. Migration Steps
1. Add Prometheus dependency
2. Implement PrometheusMetricsCollector
3. Update configuration system
4. Add middleware and interceptors
5. Update examples and documentation
6. Gradual rollout to services

### 3. Feature Flags
```go
type FeatureFlags struct {
    EnablePrometheusMetrics    bool
    EnableDetailedHTTPMetrics  bool
    EnableGRPCMessageMetrics   bool
}
```

This design ensures a robust, performant, and secure Prometheus metrics integration that builds upon the existing framework architecture while providing comprehensive observability capabilities.