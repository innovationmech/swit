# Swit Framework Tracing Configuration Examples

This directory contains example configurations for enabling OpenTelemetry distributed tracing in Swit microservice framework applications.

## Overview

The Swit framework provides deep integration with OpenTelemetry for distributed tracing with zero-code changes required. Simply configure tracing in your YAML configuration file or through environment variables, and the framework automatically:

- Injects tracing middleware into HTTP transport (Gin)
- Injects tracing interceptors into gRPC transport  
- Correlates traces with Prometheus metrics
- Provides tracing health checks
- Manages tracing lifecycle with the server

## Configuration Files

### 1. `basic-tracing.yaml` - Development Setup

- ✅ Console exporter (traces to stdout)
- ✅ 10% sampling rate for development
- ✅ Basic resource attributes
- ✅ Simple configuration for getting started

Use this configuration when:
- Starting development
- Learning about tracing
- Testing trace generation locally

### 2. `jaeger-production.yaml` - Production with Jaeger

- ✅ Jaeger collector endpoint
- ✅ 1% sampling rate for production efficiency  
- ✅ Comprehensive resource attributes
- ✅ Production-grade server settings
- ✅ Security and monitoring integrations

Use this configuration when:
- Deploying to production with Jaeger backend
- Need detailed distributed tracing
- Have Jaeger infrastructure setup

### 3. `otlp-cloud.yaml` - Cloud Provider Integration

- ✅ OTLP exporter for cloud compatibility
- ✅ Cloud provider resource attributes  
- ✅ Gzip compression for network efficiency
- ✅ Authentication headers for cloud services
- ✅ Kubernetes integration

Use this configuration when:
- Using Google Cloud Trace, AWS X-Ray, or Azure Monitor
- Running on Kubernetes
- Need cloud-native observability

### 4. `environment-variables.env` - Environment Overrides

Complete list of environment variables that override YAML configuration:
- All tracing settings can be controlled via environment variables
- Useful for CI/CD pipelines and containerized deployments
- Follows `SWIT_TRACING_*` naming convention

## Quick Start

1. **Copy a configuration template:**
   ```bash
   cp examples/tracing-config/basic-tracing.yaml ./swit.yaml
   ```

2. **Customize for your service:**
   ```yaml
   service_name: "your-service-name"
   tracing:
     enabled: true
     service_name: "your-service-name"
   ```

3. **Start your service:**
   ```bash
   go run main.go
   ```

4. **View traces:**
   - Console exporter: Check stdout for trace output
   - Jaeger: Open Jaeger UI (typically http://localhost:16686)
   - Cloud: Check your cloud provider's tracing service

## Environment Variable Override

You can override any configuration value using environment variables:

```bash
# Enable tracing
export SWIT_TRACING_ENABLED=true

# Set service name
export SWIT_TRACING_SERVICE_NAME=my-service

# Configure Jaeger endpoint
export SWIT_TRACING_EXPORTER_TYPE=jaeger
export SWIT_TRACING_EXPORTER_ENDPOINT=http://localhost:14268/api/traces

# Set sampling rate  
export SWIT_TRACING_SAMPLING_RATE=0.1

# Run your service
go run main.go
```

## Configuration Options

### Sampling Types

| Type | Description | When to Use |
|------|-------------|-------------|
| `always_on` | Capture 100% of traces | Development only |
| `always_off` | Capture 0% of traces | Disable tracing |  
| `traceidratio` | Capture specified percentage | Production (0.01-0.1) |

### Exporter Types

| Type | Description | Best For |
|------|-------------|----------|
| `console` | Output to stdout/stderr | Development, debugging |
| `jaeger` | Send to Jaeger backend | Self-hosted Jaeger |
| `otlp` | OpenTelemetry Protocol | Cloud providers, collectors |

### Resource Attributes

Standard attributes automatically populated:
- `service.name` - Service name  
- `service.version` - Service version
- `deployment.environment` - Environment (dev/staging/prod)
- `service.namespace` - Service namespace
- `service.instance.id` - Unique instance identifier

## Integration with Framework Features

### Automatic Middleware Injection

The framework automatically injects tracing middleware when tracing is enabled:

```go
// HTTP requests automatically traced
router.GET("/api/users", handler.GetUsers)

// gRPC calls automatically traced  
func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    // Your business logic - tracing is automatic
}
```

### Prometheus Metrics Correlation

Tracing automatically correlates with Prometheus metrics:
- `swit_tracing_enabled` - Whether tracing is active
- `swit_tracing_spans_total` - Number of spans created
- `swit_tracing_span_duration_seconds` - Span duration histogram
- `swit_tracing_errors_total` - Tracing errors

### Health Check Integration

Tracing status is included in health metrics:
- Component health status includes tracing health
- Automatic monitoring of tracing manager lifecycle

## Best Practices

### Production Deployments

1. **Use low sampling rates** (0.01-0.05) to reduce overhead
2. **Set appropriate timeouts** for exporter connections
3. **Include comprehensive resource attributes** for filtering
4. **Use environment variables** for deployment-specific values
5. **Monitor tracing metrics** alongside application metrics

### Development

1. **Use console exporter** for immediate feedback
2. **Higher sampling rates** (0.1-1.0) for testing
3. **Enable reflection** for gRPC services
4. **Use structured logging** with trace correlation

### Security

1. **Disable reflection** in production gRPC
2. **Use TLS** for OTLP endpoints  
3. **Secure authentication tokens** in environment variables
4. **Limit cardinality** of resource attributes

## Troubleshooting

### Tracing Not Working

1. Check configuration:
   ```bash
   # Verify tracing is enabled
   grep "tracing:" your-config.yaml
   ```

2. Check logs for tracing initialization:
   ```text
   INFO: Tracing initialized successfully
   ```

3. Verify exporter connectivity:
   ```bash
   # Test Jaeger endpoint
   curl http://jaeger-collector:14268/api/traces
   ```

### Performance Issues

1. **Reduce sampling rate** in production
2. **Increase exporter timeouts** if network is slow
3. **Monitor tracing overhead** with Prometheus metrics
4. **Use gzip compression** for OTLP exports

### Missing Traces

1. **Check sampling configuration** - may be too low
2. **Verify propagators** match your service mesh  
3. **Check exporter endpoint** accessibility
4. **Review resource attributes** for filtering

## Advanced Configuration

### Custom Propagators

```yaml
tracing:
  propagators: ["tracecontext", "baggage", "b3", "jaeger", "xray"]
```

### Multiple Exporters (via OTLP Collector)

Deploy an OTLP collector to fan out traces to multiple backends:
```yaml
tracing:
  exporter:
    type: "otlp"
    endpoint: "http://otel-collector:4318"
```

### Conditional Tracing

Use environment variables for conditional tracing:
```yaml
tracing:
  enabled: false  # Default disabled

# Enable via environment
SWIT_TRACING_ENABLED=true
```

## Examples Integration

All configuration examples are tested with:
- Framework integration tests  
- Example services in `examples/` directory
- CI/CD pipeline validation

For working examples, see:
- `examples/simple-http-service/` - Basic HTTP service with tracing
- `examples/grpc-service/` - gRPC service with tracing  
- `examples/full-featured-service/` - Complete framework showcase

## Support

For issues or questions:
1. Check framework documentation in `pkg/server/CLAUDE.md`
2. Review tracing package documentation in `pkg/tracing/`
3. Open an issue in the project repository
