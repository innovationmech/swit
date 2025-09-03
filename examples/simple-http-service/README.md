# Simple HTTP Service with Prometheus Metrics

This example demonstrates a basic HTTP service using the base server framework with integrated Prometheus metrics collection.

## Features

- HTTP REST API endpoints
- Prometheus metrics integration
- Custom business metrics
- YAML configuration support
- Environment variable overrides
- Health checks

## Endpoints

- `GET /api/v1/hello?name=<name>` - Returns a greeting
- `GET /api/v1/status` - Service status
- `POST /api/v1/echo` - Echo message
- `GET /metrics` - Prometheus metrics endpoint

## Custom Metrics

This service collects the following custom business metrics:

### Counter Metrics
- `swit_simple_http_service_hello_requests_total` - Total hello requests by name and endpoint
- `swit_simple_http_service_echo_requests_total` - Total echo requests by endpoint
- `swit_simple_http_service_echo_errors_total` - Total echo errors by error type and endpoint

### Histogram Metrics
- `swit_simple_http_service_hello_request_duration_seconds` - Hello request duration
- `swit_simple_http_service_echo_request_duration_seconds` - Echo request duration  
- `swit_simple_http_service_echo_message_length_bytes` - Echo message size distribution

## Configuration

### Using YAML Configuration File

Create a `swit.yaml` file in the same directory as the executable:

```yaml
service_name: "simple-http-service"
shutdown_timeout: "30s"

http:
  enabled: true
  port: "8080"
  enable_ready: true

prometheus:
  enabled: true
  endpoint: "/metrics"
  namespace: "swit"
  subsystem: "simple_http_service"
  labels:
    service: "simple-http-service"
    version: "v1"
```

### Environment Variables

Override configuration with environment variables:

- `HTTP_PORT` - HTTP server port (default: 8080)
- `CONFIG_PATH` - Configuration file path (default: swit.yaml)
- `PROMETHEUS_ENABLED` - Enable/disable Prometheus metrics (default: true)
- `DISCOVERY_ENABLED` - Enable/disable service discovery (default: false)
- `CONSUL_ADDRESS` - Consul server address (default: localhost:8500)

## Running the Service

### Basic Run
```bash
go run main.go
```

### With Custom Configuration
```bash
CONFIG_PATH=./custom-config.yaml go run main.go
```

### With Environment Variables
```bash
HTTP_PORT=9090 PROMETHEUS_ENABLED=true go run main.go
```

## Testing Metrics

### Basic Requests
```bash
# Generate some metrics
curl "http://localhost:8080/api/v1/hello?name=Alice"
curl "http://localhost:8080/api/v1/hello?name=Bob"
curl -X POST -H "Content-Type: application/json" -d '{"message":"Hello World"}' http://localhost:8080/api/v1/echo

# View metrics
curl http://localhost:8080/metrics
```

### Sample Metrics Output
```text
# HELP swit_simple_http_service_hello_requests_total Counter metric hello_requests_total
# TYPE swit_simple_http_service_hello_requests_total counter
swit_simple_http_service_hello_requests_total{endpoint="/api/v1/hello",name="Alice"} 1
swit_simple_http_service_hello_requests_total{endpoint="/api/v1/hello",name="Bob"} 1

# HELP swit_simple_http_service_hello_request_duration_seconds Histogram metric hello_request_duration_seconds
# TYPE swit_simple_http_service_hello_request_duration_seconds histogram
swit_simple_http_service_hello_request_duration_seconds_bucket{endpoint="/api/v1/hello",name="Alice",le="0.001"} 1
swit_simple_http_service_hello_request_duration_seconds_bucket{endpoint="/api/v1/hello",name="Alice",le="0.01"} 1
```

## Integration with Monitoring Stack

This service can be integrated with a complete monitoring stack:

1. **Prometheus** - Metrics collection
2. **Grafana** - Visualization and dashboards
3. **Alertmanager** - Alerting rules

See the `../prometheus-monitoring` example for a complete setup with Docker Compose.

## Health Checks

The service includes health checks that can be used with:

- Kubernetes liveness/readiness probes
- Docker health checks  
- Load balancer health checks
- Service discovery health monitoring

## Development

### Adding Custom Metrics

To add custom business metrics:

1. Get the metrics collector from the dependency container
2. Use appropriate metric types:
   - `IncrementCounter()` for counting events
   - `SetGauge()` for current values  
   - `ObserveHistogram()` for distributions

```go
// Example: Track processing time
start := time.Now()
// ... do processing ...
duration := time.Since(start).Seconds()
metricsCollector.ObserveHistogram("processing_duration_seconds", duration, labels)
```

### Testing

```bash
# Run tests
go test ./...

# Test with coverage
go test -cover ./...
```

## Production Considerations

1. **Cardinality** - Monitor metric cardinality to avoid memory issues
2. **Labels** - Use low-cardinality labels (avoid user IDs, request IDs)
3. **Performance** - Metrics collection has minimal overhead but monitor in high-traffic scenarios
4. **Security** - Restrict access to `/metrics` endpoint in production
5. **Retention** - Configure appropriate retention policies in Prometheus

This example demonstrates how to create a simple HTTP service using the base server framework.

## Features

- HTTP-only service (no gRPC)
- RESTful API endpoints
- Health checks
- Dependency injection
- Graceful shutdown
- Optional service discovery

## API Endpoints

- `GET /api/v1/hello?name=<name>` - Returns a greeting message
- `GET /api/v1/status` - Returns service status
- `POST /api/v1/echo` - Echoes back the request message
- `GET /health` - Health check endpoint (automatically registered)
- `GET /ready` - Readiness check endpoint (automatically registered)

## Running the Service

### Basic Usage

```bash
go run main.go
```

The service will start on port 8080 by default.

### With Custom Configuration

```bash
HTTP_PORT=9000 DISCOVERY_ENABLED=true go run main.go
```

### Environment Variables

- `HTTP_PORT` - HTTP port to listen on (default: 8080)
- `DISCOVERY_ENABLED` - Enable service discovery (default: false)
- `CONSUL_ADDRESS` - Consul address for service discovery (default: localhost:8500)

## Testing the Service

### Hello Endpoint

```bash
curl http://localhost:8080/api/v1/hello
curl http://localhost:8080/api/v1/hello?name=Alice
```

### Status Endpoint

```bash
curl http://localhost:8080/api/v1/status
```

### Echo Endpoint

```bash
curl -X POST http://localhost:8080/api/v1/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, World!"}'
```

### Health Check

```bash
curl http://localhost:8080/health
```

### Readiness Check

```bash
curl http://localhost:8080/ready
```

## Code Structure

- **SimpleHTTPService**: Implements `ServiceRegistrar` interface
- **SimpleHTTPHandler**: Implements `HTTPHandler` interface for HTTP routes
- **SimpleHealthCheck**: Implements `HealthCheck` interface for health monitoring
- **SimpleDependencyContainer**: Implements `DependencyContainer` interface for dependency injection

## Key Concepts Demonstrated

1. **Service Registration**: How to register HTTP handlers and health checks
2. **HTTP Routing**: Using Gin router for RESTful API endpoints
3. **Configuration Management**: Environment-based configuration with defaults
4. **Dependency Injection**: Simple dependency container implementation
5. **Graceful Shutdown**: Proper signal handling and resource cleanup
6. **Health Monitoring**: Built-in health and readiness checks

## Extending the Example

To extend this example:

1. **Add Database**: Implement database connections in the dependency container
2. **Add Authentication**: Use middleware for JWT or other authentication
3. **Add Metrics**: Implement Prometheus metrics collection
4. **Add Logging**: Enhanced structured logging with request tracing
5. **Add Validation**: Request validation and error handling
6. **Add Tests**: Unit and integration tests for the service

## Production Considerations

For production use, consider:

- Proper error handling and logging
- Request validation and sanitization
- Rate limiting and security middleware
- Database connection pooling
- Monitoring and alerting
- Configuration management (config files, secrets)
- Load balancing and service discovery
