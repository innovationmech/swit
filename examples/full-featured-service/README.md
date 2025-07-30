# Full-Featured Service Example

This example demonstrates a comprehensive service that supports both HTTP and gRPC protocols using the base server framework.

## Features

- **Dual Protocol Support**: Both HTTP REST and gRPC endpoints
- **Service Discovery**: Optional Consul integration
- **Health Checks**: Built-in health monitoring
- **Dependency Injection**: Structured dependency management
- **Graceful Shutdown**: Proper resource cleanup
- **Middleware Support**: CORS, logging, and other middleware
- **Configuration Management**: Environment-based configuration

## API Endpoints

### HTTP Endpoints

- `POST /api/v1/greet` - HTTP version of the gRPC SayHello method
- `GET /api/v1/status` - Service status information
- `GET /api/v1/metrics` - Service metrics (mock data)
- `POST /api/v1/echo` - Echo service for testing
- `GET /health` - Health check endpoint (automatically registered)
- `GET /ready` - Readiness check endpoint (automatically registered)

### gRPC Methods

- `SayHello(SayHelloRequest) -> SayHelloResponse` - Greeting service
- Health check service (automatically registered)

## Running the Service

### Basic Usage

```bash
go run main.go
```

The service will start with:
- HTTP server on port 8080
- gRPC server on port 9090

### With Custom Configuration

```bash
HTTP_PORT=8000 GRPC_PORT=9000 DISCOVERY_ENABLED=true go run main.go
```

### Environment Variables

- `HTTP_PORT` - HTTP port to listen on (default: 8080)
- `GRPC_PORT` - gRPC port to listen on (default: 9090)
- `DISCOVERY_ENABLED` - Enable service discovery (default: false)
- `CONSUL_ADDRESS` - Consul address for service discovery (default: localhost:8500)
- `ENVIRONMENT` - Environment name (default: development)

## Testing the Service

### HTTP Endpoints

#### Greet Endpoint

```bash
curl -X POST http://localhost:8080/api/v1/greet \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice"}'
```



#### Status Endpoint

```bash
curl http://localhost:8080/api/v1/status
```

#### Metrics Endpoint

```bash
curl http://localhost:8080/api/v1/metrics
```

#### Echo Endpoint

```bash
curl -X POST http://localhost:8080/api/v1/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, World!"}'
```

#### Health Check

```bash
curl http://localhost:8080/health
```

### gRPC Endpoints

First, install grpcurl:
```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

#### List Services

```bash
grpcurl -plaintext localhost:9090 list
```

#### SayHello Method

```bash
grpcurl -plaintext -d '{"name": "Alice"}' \
  localhost:9090 swit.interaction.v1.GreeterService/SayHello
```



#### Health Check

```bash
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
```

## Code Structure

### Service Components

- **FullFeaturedService**: Main service implementing `ServiceRegistrar`
- **FullFeaturedHTTPHandler**: HTTP handler implementing `HTTPHandler`
- **FullFeaturedGRPCService**: gRPC service implementing `GRPCService`
- **FullFeaturedHealthCheck**: Health check implementing `HealthCheck`
- **FullFeaturedDependencyContainer**: Dependency container implementing `DependencyContainer`

### Architecture

```
┌─────────────────────┐
│  FullFeaturedService │
│  (ServiceRegistrar)  │
└──────────┬──────────┘
           │
    ┌──────┴──────┐
    │             │
┌───▼───┐    ┌────▼────┐
│ HTTP  │    │  gRPC   │
│Handler│    │ Service │
└───────┘    └─────────┘
```

## Key Concepts Demonstrated

1. **Dual Protocol Support**: Same business logic exposed via both HTTP and gRPC
2. **Service Registration**: How to register multiple handlers and services
3. **Protocol Translation**: Converting between HTTP JSON and gRPC protobuf
4. **Configuration Management**: Environment-based configuration with validation
5. **Dependency Injection**: Structured dependency management
6. **Health Monitoring**: Comprehensive health checks for both protocols
7. **Graceful Shutdown**: Proper cleanup of resources

## Protocol Comparison

| Feature | HTTP | gRPC |
|---------|------|------|
| Data Format | JSON | Protocol Buffers |
| Transport | HTTP/1.1 or HTTP/2 | HTTP/2 |
| Schema | OpenAPI/Swagger | Protocol Buffers |
| Streaming | Limited | Full support |
| Browser Support | Native | Requires proxy |
| Performance | Good | Excellent |

## Development Workflow

### Adding New Endpoints

1. **Define Protocol Buffer**: Add methods to `.proto` files
2. **Generate Code**: Run `make proto` to generate Go code
3. **Implement gRPC**: Add method to `FullFeaturedGRPCService`
4. **Implement HTTP**: Add route to `FullFeaturedHTTPHandler`
5. **Test Both**: Verify both protocols work correctly

### Example: Adding a new "GetInfo" endpoint

1. Add to proto file:
```protobuf
rpc GetInfo(GetInfoRequest) returns (GetInfoResponse);
```

2. Implement gRPC method:
```go
func (s *FullFeaturedGRPCService) GetInfo(ctx context.Context, req *interaction.GetInfoRequest) (*interaction.GetInfoResponse, error) {
    // Implementation
}
```

3. Add HTTP route:
```go
api.GET("/info", h.handleGetInfo)
```

## Production Considerations

### Security

- Enable TLS for gRPC in production
- Implement authentication middleware
- Add rate limiting
- Validate all inputs
- Use HTTPS for HTTP endpoints

### Monitoring

- Add structured logging
- Implement metrics collection (Prometheus)
- Set up distributed tracing
- Monitor both HTTP and gRPC endpoints
- Track error rates and latencies

### Deployment

- Use service discovery for load balancing
- Configure health checks for load balancers
- Set appropriate timeouts
- Configure resource limits
- Use circuit breakers for external dependencies

### Configuration

```yaml
# production.yaml
service_name: "full-featured-service"

http:
  port: "8080"
  enabled: true
  enable_ready: true
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s

grpc:
  port: "9090"
  enabled: true
  enable_reflection: false  # Disable in production
  enable_health_service: true
  max_recv_msg_size: 4194304
  max_send_msg_size: 4194304

discovery:
  enabled: true
  address: "consul.example.com:8500"
  service_name: "full-featured-service"
  tags:
    - "production"
    - "v1"

middleware:
  enable_cors: true
  enable_auth: true
  enable_rate_limit: true
  enable_logging: true
```

## Extending the Example

### Add Database Integration

```go
type DatabaseService struct {
    db *sql.DB
}

func (d *FullFeaturedDependencyContainer) initDatabase() error {
    db, err := sql.Open("mysql", d.config.Database.DSN)
    if err != nil {
        return err
    }
    d.AddService("database", &DatabaseService{db: db})
    return nil
}
```

### Add Authentication

```go
func (h *FullFeaturedHTTPHandler) authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if !isValidToken(token) {
            c.JSON(401, gin.H{"error": "unauthorized"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### Add Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    requestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "requests_total",
            Help: "Total number of requests",
        },
        []string{"method", "endpoint", "status"},
    )
)
```

## Troubleshooting

### Common Issues

1. **Port Conflicts**: Ensure ports 8080 and 9090 are available
2. **Protocol Buffer Issues**: Run `make proto` to regenerate code
3. **Import Errors**: Ensure all dependencies are installed
4. **Service Discovery**: Check Consul connectivity if enabled

### Debug Logging

Enable debug logging by setting the log level:
```bash
LOG_LEVEL=debug go run main.go
```

## Related Examples

- [Simple HTTP Service](../simple-http-service/) - HTTP-only service
- [gRPC Service](../grpc-service/) - gRPC-only service
- [Service with Discovery](../service-with-discovery/) - Advanced service discovery patterns