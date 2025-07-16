# SWIT Server - Enterprise Architecture Usage Guide

## Overview

The SWIT server uses an enterprise-grade improved architecture with clean separation of concerns and dual-protocol support.

## Starting the Server

```bash
# Start the SWIT server
./swit-serve serve
```

Features:
- ✅ Unified gRPC and HTTP transport management
- ✅ Clean separation of business logic and protocol handling  
- ✅ Enterprise-grade middleware support
- ✅ Multiple service registration (Greeter, Notification)
- ✅ Better error handling and logging
- ✅ Graceful shutdown handling

## Available Services

### 1. Greeter Service

#### gRPC Endpoints
```bash
# Basic greeting
grpcurl -plaintext -d '{"name":"World"}' \
  localhost:50051 swit.v1.greeter.GreeterService/SayHello

# Multi-language greeting
grpcurl -plaintext -d '{"name":"世界","language":"chinese"}' \
  localhost:50051 swit.v1.greeter.GreeterService/SayHello
```

#### HTTP REST Endpoints
```bash
# Basic greeting
curl -X POST http://localhost:8080/api/v1/greeter/hello \
  -H "Content-Type: application/json" \
  -d '{"name":"World"}'

# Multi-language greeting
curl -X POST http://localhost:8080/api/v1/greeter/hello \
  -H "Content-Type: application/json" \
  -d '{"name":"世界","language":"chinese"}'
```

### 2. Notification Service (New!)

#### gRPC Endpoints
```bash
# Create notification
grpcurl -plaintext -d '{"user_id":"user123","title":"Hello","content":"World"}' \
  localhost:50051 swit.v1.notification.NotificationService/CreateNotification

# Get notifications
grpcurl -plaintext -d '{"user_id":"user123","limit":10}' \
  localhost:50051 swit.v1.notification.NotificationService/GetNotifications

# Mark as read
grpcurl -plaintext -d '{"notification_id":"notif123"}' \
  localhost:50051 swit.v1.notification.NotificationService/MarkAsRead

# Delete notification
grpcurl -plaintext -d '{"notification_id":"notif123"}' \
  localhost:50051 swit.v1.notification.NotificationService/DeleteNotification
```

#### HTTP REST Endpoints
```bash
# Create notification
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user123","title":"Hello","content":"World"}'

# Get notifications
curl -X GET "http://localhost:8080/api/v1/notifications?user_id=user123&limit=10"

# Mark as read
curl -X PATCH http://localhost:8080/api/v1/notifications/notif123/read

# Delete notification
curl -X DELETE http://localhost:8080/api/v1/notifications/notif123
```

## Server Ports

- **HTTP**: `localhost:8080`
- **gRPC**: `localhost:50051` (HTTP port + 1000)

Ports can be configured via the configuration file.

## Configuration

The server uses the same configuration system as before. Create a `swit.yaml` file:

```yaml
server:
  port: "8080"
  grpc_port: "50051"  # Optional: will default to port + 1000

database:
  host: "localhost"
  port: "3306"
  username: "user"
  password: "pass"
  dbname: "swit"

serviceDiscovery:
  address: "localhost:8500"
```

## Development

### Running Tests
```bash
# Run all service tests
go test ./internal/switserve/service -v

# Run command tests  
go test ./internal/switserve/cmd/... -v

# Run greeter tests specifically
go test ./internal/switserve/service -run TestGreeter -v
```

### Building
```bash
# Build the server
go build -o swit-serve cmd/swit-serve/swit-serve.go
```

## Architecture Benefits

The new improved architecture provides:

1. **Separation of Concerns**: Business logic is separate from transport protocols
2. **Code Reuse**: Same business logic serves both gRPC and HTTP
3. **Extensibility**: Easy to add new services using the `ServiceRegistrar` pattern  
4. **Testing**: Business logic can be tested independently of protocols
5. **Maintenance**: Changes to business logic only need to be made in one place

## Migration Guide

The server uses the improved architecture by default. When migrating from older implementations:

1. Update any direct imports of old service implementations
2. Use the new service interfaces for dependency injection
3. Test both gRPC and HTTP endpoints to ensure functionality
4. Leverage the unified business logic layer for consistent behavior across protocols