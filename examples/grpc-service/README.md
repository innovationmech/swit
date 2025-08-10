# gRPC Service Example

This example demonstrates how to create a gRPC service using the base server framework.

## Features

- gRPC-only service (no HTTP)
- Protocol Buffer definitions
- Health checks
- Dependency injection
- Graceful shutdown
- Optional service discovery
- gRPC reflection enabled

## gRPC Methods

- `SayHello(SayHelloRequest) -> SayHelloResponse` - Returns a greeting message
- Health check service (automatically registered)

## Running the Service

### Basic Usage

```bash
go run main.go
```

The service will start on port 9090 by default.

### With Custom Configuration

```bash
GRPC_PORT=50051 DISCOVERY_ENABLED=true go run main.go
```

### Environment Variables

- `GRPC_PORT` - gRPC port to listen on (default: 9090)
- `DISCOVERY_ENABLED` - Enable service discovery (default: false)
- `CONSUL_ADDRESS` - Consul address for service discovery (default: localhost:8500)

## Testing the Service

### Using grpcurl

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

### Using Go Client

```go
package main

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    
    "github.com/innovationmech/swit/api/gen/go/proto/swit/interaction/v1"
)

func main() {
    // Connect to the server
    conn, err := grpc.NewClient("localhost:9090", 
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    // Create client
    client := interaction.NewGreeterServiceClient(conn)

    // Call SayHello
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    resp, err := client.SayHello(ctx, &interaction.SayHelloRequest{
        Name: "Alice",
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Response: %s", resp.Message)
}
```

## Code Structure

- **GreeterService**: Implements `ServiceRegistrar` interface
- **GreeterGRPCService**: Implements `GRPCService` interface and the actual gRPC methods
- **GreeterHealthCheck**: Implements `HealthCheck` interface for health monitoring
- **GreeterDependencyContainer**: Implements `DependencyContainer` interface for dependency injection

## Protocol Buffer Definitions

The service uses the existing protocol buffer definitions from `api/proto/swit/interaction/v1/greeter.proto`.

## Key Concepts Demonstrated

1. **gRPC Service Registration**: How to register gRPC services with the server
2. **Protocol Buffers**: Using generated Go code from .proto files
3. **Error Handling**: Proper gRPC status codes and error responses
4. **Configuration Management**: Environment-based configuration with defaults
5. **Dependency Injection**: Simple dependency container implementation
6. **Graceful Shutdown**: Proper signal handling and resource cleanup
7. **Health Monitoring**: Built-in gRPC health service

## Extending the Example

To extend this example:

1. **Add Streaming**: Implement server/client streaming methods
2. **Add Authentication**: Use gRPC interceptors for authentication
3. **Add Metrics**: Implement Prometheus metrics collection
4. **Add Logging**: Enhanced structured logging with request tracing
5. **Add Validation**: Request validation using protoc-gen-validate
6. **Add Tests**: Unit and integration tests for the service

## Production Considerations

For production use, consider:

- Proper error handling and logging
- Request validation and sanitization
- Authentication and authorization
- Rate limiting and security interceptors
- Connection pooling and load balancing
- Monitoring and alerting
- TLS/SSL encryption
- Configuration management (config files, secrets)