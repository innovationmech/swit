# gRPC Service Example

This example demonstrates how to build a gRPC-only service using the Swit framework with Protocol Buffers. It showcases gRPC service registration, Protocol Buffer message handling, server reflection, and health service integration.

## Overview

The gRPC Service example (`examples/grpc-service/`) provides:
- **gRPC-only service** - Focused on Protocol Buffer-based RPC functionality
- **Protocol Buffer integration** - Using generated Go code from .proto definitions
- **Server reflection** - gRPC reflection for development and debugging
- **Health service** - Standard gRPC health checking protocol
- **Keepalive configuration** - Connection management and optimization

## Key Features

### Service Architecture
- Implements `BusinessServiceRegistrar` for framework integration
- Registers gRPC services with proper server configuration
- Uses Protocol Buffer definitions for type-safe communication
- Demonstrates proper gRPC service lifecycle management

### gRPC Services
- **GreeterService** - Implements `swit.interaction.v1.GreeterService`
- **SayHello** RPC method - Takes name parameter and returns greeting message
- **Health Service** - Standard gRPC health check integration
- **Reflection Service** - For development tools and debugging

### Advanced gRPC Features
- Message size limits (4MB default)
- Keepalive parameters for connection management
- Server-side validation and error handling
- Proper gRPC status codes and error responses

## Code Structure

### Main Service Implementation

```go
// GreeterService implements the ServiceRegistrar interface
type GreeterService struct {
    name string
}

func (s *GreeterService) RegisterServices(registry server.BusinessServiceRegistry) error {
    // Register gRPC service
    grpcService := &GreeterGRPCService{serviceName: s.name}
    if err := registry.RegisterBusinessGRPCService(grpcService); err != nil {
        return fmt.Errorf("failed to register gRPC service: %w", err)
    }

    // Register health check
    healthCheck := &GreeterHealthCheck{serviceName: s.name}
    if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
        return fmt.Errorf("failed to register health check: %w", err)
    }

    return nil
}
```

### gRPC Service Implementation

```go
type GreeterGRPCService struct {
    serviceName string
    interaction.UnimplementedGreeterServiceServer
}

func (s *GreeterGRPCService) RegisterGRPC(server interface{}) error {
    grpcServer := server.(*grpc.Server)
    interaction.RegisterGreeterServiceServer(grpcServer, s)
    return nil
}

func (s *GreeterGRPCService) SayHello(ctx context.Context, req *interaction.SayHelloRequest) (*interaction.SayHelloResponse, error) {
    // Validate request
    if req.GetName() == "" {
        return nil, status.Error(codes.InvalidArgument, "name cannot be empty")
    }

    // Create response
    response := &interaction.SayHelloResponse{
        Message: fmt.Sprintf("Hello, %s!", req.GetName()),
    }

    return response, nil
}
```

### gRPC Configuration

```go
config := &server.ServerConfig{
    ServiceName: "grpc-greeter-service",
    HTTP: server.HTTPConfig{
        Enabled: false, // gRPC-only service
    },
    GRPC: server.GRPCConfig{
        Port:                getEnv("GRPC_PORT", "9090"),
        EnableReflection:    true,
        EnableHealthService: true,
        Enabled:             true,
        MaxRecvMsgSize:      4 * 1024 * 1024, // 4MB
        MaxSendMsgSize:      4 * 1024 * 1024, // 4MB
        KeepaliveParams: server.GRPCKeepaliveParams{
            MaxConnectionIdle:     15 * time.Minute,
            MaxConnectionAge:      30 * time.Minute,
            MaxConnectionAgeGrace: 5 * time.Minute,
            Time:                  5 * time.Minute,
            Timeout:               1 * time.Minute,
        },
        KeepalivePolicy: server.GRPCKeepalivePolicy{
            MinTime:             5 * time.Minute,
            PermitWithoutStream: false,
        },
    },
}
```

## Protocol Buffer Definition

The service uses Protocol Buffer definitions from `api/proto/swit/interaction/v1/`:

```protobuf
syntax = "proto3";

package swit.interaction.v1;

service GreeterService {
  rpc SayHello(SayHelloRequest) returns (SayHelloResponse);
}

message SayHelloRequest {
  string name = 1;
}

message SayHelloResponse {
  string message = 1;
}
```

## Running the Example

### Prerequisites
- Go 1.23.12+ installed
- gRPC tools installed (for testing)
- Framework dependencies available

### Quick Start

1. **Navigate to the example directory:**
   ```bash
   cd examples/grpc-service
   ```

2. **Run the service:**
   ```bash
   go run main.go
   ```

3. **Test with grpcurl:**
   ```bash
   # Install grpcurl if not available
   go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
   
   # Test SayHello method
   grpcurl -plaintext -d '{"name": "Alice"}' \
           localhost:9090 \
           swit.interaction.v1.GreeterService/SayHello
   
   # Check health service
   grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
   
   # List available services (reflection)
   grpcurl -plaintext localhost:9090 list
   ```

### Environment Configuration

```bash
# Set custom gRPC port
export GRPC_PORT=9999

# Enable service discovery
export DISCOVERY_ENABLED=true
export CONSUL_ADDRESS=localhost:8500

# Run with custom configuration
go run main.go
```

### Expected Responses

**SayHello RPC:**
```json
{
  "message": "Hello, Alice!"
}
```

**Health Check:**
```json
{
  "status": "SERVING"
}
```

**Service List:**
```
grpc.health.v1.Health
grpc.reflection.v1alpha.ServerReflection
swit.interaction.v1.GreeterService
```

## Development Patterns

### Adding New RPC Methods

1. **Update Protocol Buffer definition:**
   ```protobuf
   service GreeterService {
     rpc SayHello(SayHelloRequest) returns (SayHelloResponse);
     rpc SayGoodbye(SayGoodbyeRequest) returns (SayGoodbyeResponse);
   }
   
   message SayGoodbyeRequest {
     string name = 1;
   }
   
   message SayGoodbyeResponse {
     string message = 1;
   }
   ```

2. **Regenerate Go code:**
   ```bash
   make proto
   ```

3. **Implement method:**
   ```go
   func (s *GreeterGRPCService) SayGoodbye(ctx context.Context, req *interaction.SayGoodbyeRequest) (*interaction.SayGoodbyeResponse, error) {
       if req.GetName() == "" {
           return nil, status.Error(codes.InvalidArgument, "name cannot be empty")
       }
   
       response := &interaction.SayGoodbyeResponse{
           Message: fmt.Sprintf("Goodbye, %s!", req.GetName()),
       }
   
       return response, nil
   }
   ```

### Error Handling Patterns

```go
func (s *GreeterGRPCService) ValidatedMethod(ctx context.Context, req *SomeRequest) (*SomeResponse, error) {
    // Input validation
    if req.GetField() == "" {
        return nil, status.Error(codes.InvalidArgument, "field is required")
    }
    
    // Business logic with error handling
    result, err := s.processRequest(req)
    if err != nil {
        switch {
        case errors.Is(err, ErrNotFound):
            return nil, status.Error(codes.NotFound, "resource not found")
        case errors.Is(err, ErrPermissionDenied):
            return nil, status.Error(codes.PermissionDenied, "access denied")
        default:
            return nil, status.Error(codes.Internal, "internal error")
        }
    }
    
    return result, nil
}
```

### Context and Timeout Handling

```go
func (s *GreeterGRPCService) LongRunningMethod(ctx context.Context, req *SomeRequest) (*SomeResponse, error) {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return nil, status.Error(codes.Canceled, "request canceled")
    default:
    }
    
    // Set timeout for external calls
    timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    result, err := s.externalService.Call(timeoutCtx, req)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            return nil, status.Error(codes.DeadlineExceeded, "operation timed out")
        }
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    return result, nil
}
```

## Testing gRPC Services

### Unit Testing

```go
func TestGreeterService_SayHello(t *testing.T) {
    service := &GreeterGRPCService{serviceName: "test-service"}
    
    tests := []struct {
        name    string
        request *interaction.SayHelloRequest
        want    string
        wantErr codes.Code
    }{
        {
            name:    "valid request",
            request: &interaction.SayHelloRequest{Name: "Alice"},
            want:    "Hello, Alice!",
            wantErr: codes.OK,
        },
        {
            name:    "empty name",
            request: &interaction.SayHelloRequest{Name: ""},
            want:    "",
            wantErr: codes.InvalidArgument,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resp, err := service.SayHello(context.Background(), tt.request)
            
            if tt.wantErr != codes.OK {
                assert.Error(t, err)
                st, ok := status.FromError(err)
                assert.True(t, ok)
                assert.Equal(t, tt.wantErr, st.Code())
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.want, resp.GetMessage())
            }
        })
    }
}
```

### Integration Testing

```go
func TestGRPCServer(t *testing.T) {
    // Start test server
    config := &server.ServerConfig{
        GRPC: server.GRPCConfig{
            Port:    "0", // Dynamic port
            Enabled: true,
            EnableReflection: true,
        },
    }
    
    service := NewGreeterService("test")
    srv, err := server.NewBusinessServerCore(config, service, nil)
    require.NoError(t, err)
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    err = srv.Start(ctx)
    require.NoError(t, err)
    defer srv.Shutdown()
    
    // Create client connection
    conn, err := grpc.DialContext(ctx, srv.GetGRPCAddress(),
        grpc.WithInsecure(),
        grpc.WithBlock(),
    )
    require.NoError(t, err)
    defer conn.Close()
    
    // Test service
    client := interaction.NewGreeterServiceClient(conn)
    resp, err := client.SayHello(ctx, &interaction.SayHelloRequest{
        Name: "TestUser",
    })
    
    assert.NoError(t, err)
    assert.Equal(t, "Hello, TestUser!", resp.GetMessage())
}
```

## Best Practices Demonstrated

1. **Protocol Buffer Design** - Clean message definitions with proper field naming
2. **Error Handling** - Appropriate gRPC status codes for different error conditions
3. **Input Validation** - Server-side validation with meaningful error messages
4. **Context Usage** - Proper context propagation and timeout handling
5. **Service Registration** - Clean integration with framework service registry

## Next Steps

After understanding this gRPC example:

1. **Combine with HTTP** - See `full-featured-service` for dual HTTP/gRPC transport
2. **Advanced Features** - Explore streaming RPCs, interceptors, and metadata
3. **Production Setup** - Add authentication, rate limiting, and monitoring
4. **Client Libraries** - Generate client code for other languages

This example provides a solid foundation for building gRPC-based microservices with the Swit framework using modern Protocol Buffer patterns.