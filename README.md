# Swit

[![CI](https://github.com/innovationmech/swit/workflows/CI/badge.svg)](https://github.com/innovationmech/swit/actions/workflows/ci.yml)
[![Test Suite](https://github.com/innovationmech/swit/workflows/Test%20Suite/badge.svg)](https://github.com/innovationmech/swit/actions/workflows/test.yml)
[![Security Checks](https://github.com/innovationmech/swit/workflows/Security%20Checks/badge.svg)](https://github.com/innovationmech/swit/actions/workflows/security-checks.yml)
[![codecov](https://codecov.io/gh/innovationmech/swit/branch/master/graph/badge.svg)](https://codecov.io/gh/innovationmech/swit)
[![Go Report Card](https://goreportcard.com/badge/github.com/innovationmech/swit)](https://goreportcard.com/report/github.com/innovationmech/swit)
[![Go Reference](https://pkg.go.dev/badge/github.com/innovationmech/swit.svg)](https://pkg.go.dev/github.com/innovationmech/swit)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.23.12-blue.svg)
[![GitHub release](https://img.shields.io/github/release/innovationmech/swit.svg)](https://github.com/innovationmech/swit/releases)
[![License](https://img.shields.io/github/license/innovationmech/swit.svg)](LICENSE)

A comprehensive microservice framework for Go that provides a unified, production-ready foundation for building scalable microservices.

📖 **[Complete Documentation](https://innovationmech.github.io/swit/)** | [中文文档](https://innovationmech.github.io/swit/zh/)

## Key Features

- 🚀 **Complete Server Framework**: Unified HTTP and gRPC transport coordination
- 💉 **Dependency Injection**: Factory-based container with automatic lifecycle management
- 📊 **Performance Monitoring**: Built-in metrics collection and monitoring hooks
- 🔍 **Service Discovery**: Consul-based registration with health check integration
- 🛡️ **Middleware Stack**: Configurable CORS, rate limiting, authentication, and timeouts
- ⚡ **Protocol Buffers**: Full Buf toolchain support for API development
- 📱 **Example Services**: Complete reference implementations and usage patterns

## Architecture Overview

### Core Components
- **`pkg/server/`** - Base server framework and lifecycle management
- **`pkg/transport/`** - HTTP/gRPC transport coordination layer
- **`pkg/middleware/`** - Configurable middleware stack
- **`pkg/discovery/`** - Service discovery integration

### Example Services
- **`examples/`** - Simple examples for getting started
- **`internal/switserve/`** - User management reference service
- **`internal/switauth/`** - Authentication service with JWT
- **`internal/switctl/`** - CLI tool implementation

## Requirements

- **Go 1.23.12+** with generics support
- **Optional**: MySQL 8.0+, Redis 6.0+, Consul 1.12+ (for examples)
- **Development**: Buf CLI, Docker, Make

## Quick Start

### 1. Get the Framework
```bash
git clone https://github.com/innovationmech/swit.git
cd swit
go mod download
```

### 2. Create a Simple Service
```go
package main

import (
    "context"
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/server"
)

type MyService struct{}

func (s *MyService) RegisterServices(registry server.BusinessServiceRegistry) error {
    return registry.RegisterBusinessHTTPHandler(&MyHTTPHandler{})
}

type MyHTTPHandler struct{}

func (h *MyHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    ginRouter.GET("/hello", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"message": "Hello from Swit!"})
    })
    return nil
}

func (h *MyHTTPHandler) GetServiceName() string {
    return "my-service"
}

func main() {
    config := &server.ServerConfig{
        ServiceName: "my-service",
        HTTP: server.HTTPConfig{Port: "8080", Enabled: true},
        GRPC: server.GRPCConfig{Enabled: false},
    }
    
    service := &MyService{}
    baseServer, _ := server.NewBusinessServerCore(config, service, nil)
    
    ctx := context.Background()
    baseServer.Start(ctx)
    defer baseServer.Shutdown()
    
    select {} // Keep running
}
```

### 3. Run Your Service
```bash
go run main.go
curl http://localhost:8080/hello
```

## Examples

### Simple Examples
```bash
# HTTP service
cd examples/simple-http-service && go run main.go

# gRPC service  
cd examples/grpc-service && go run main.go

# Full-featured service
cd examples/full-featured-service && go run main.go
```

### Reference Services
```bash
# Build all services
make build

# Run services
./bin/swit-serve    # User management (HTTP: 9000, gRPC: 10000)
./bin/swit-auth     # Authentication (HTTP: 9001, gRPC: 50051)
./bin/switctl --help # CLI tool
./bin/saga-migrate --help # Database migration tool
```

### Database Migrations (Saga Storage)

The `saga-migrate` tool manages Saga database schema migrations:

```bash
# Apply all migrations
saga-migrate -dsn 'postgres://localhost/saga' -action migrate

# Check migration status
saga-migrate -dsn 'postgres://localhost/saga' -action status

# Apply specific version
saga-migrate -dsn 'postgres://localhost/saga' -action apply -version 2

# Rollback migration
saga-migrate -dsn 'postgres://localhost/saga' -action rollback -version 2

# Validate schema version
saga-migrate -dsn 'postgres://localhost/saga' -action validate -version 2

# See full documentation
cat docs/saga-database-migrations.md
```

## Development

### Setup Development Environment
```bash
# Complete setup
make setup-dev

# Quick setup
make setup-quick
```

### Common Commands
```bash
# Build
make build          # Full build
make build-dev      # Quick build

# Test
make test           # All tests
make test-dev       # Quick tests
make test-coverage  # Coverage report

# API Development
make proto          # Generate protobuf code
make swagger        # Generate API docs

# Code Quality
make tidy           # Tidy modules
make format         # Format code
make quality        # Quality checks
```

## Docker Deployment

```bash
# Build images
make docker

# Run with Docker Compose
docker-compose up -d

# Or run individually
docker run -p 9000:9000 -p 10000:10000 swit-serve:latest
docker run -p 9001:9001 swit-auth:latest
```

## Contributing

1. Fork the repository
2. Set up development environment: `make setup-dev`
3. Run tests: `make test`
4. Make changes following existing patterns
5. Add tests for new functionality
6. Submit a pull request

Read our [Code of Conduct](CODE_OF_CONDUCT.md) before contributing.

## License

MIT License - See [LICENSE](LICENSE) file for details

---

**For complete documentation, examples, and advanced usage, visit [innovationmech.github.io/swit](https://innovationmech.github.io/swit/)**
