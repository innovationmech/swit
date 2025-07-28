# Simple HTTP Service Example

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