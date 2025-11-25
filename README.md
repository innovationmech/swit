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
- 🔄 **Saga Distributed Transactions**: Enterprise-grade distributed transaction management with orchestration and choreography patterns
- 🔐 **Enterprise Security**: OAuth2/OIDC authentication, OPA policy engine (RBAC/ABAC), TLS/mTLS encryption
- 📱 **Example Services**: Complete reference implementations and usage patterns

## Architecture Overview

### Core Components
- **`pkg/server/`** - Base server framework and lifecycle management
- **`pkg/transport/`** - HTTP/gRPC transport coordination layer
- **`pkg/middleware/`** - Configurable middleware stack
- **`pkg/discovery/`** - Service discovery integration
- **`pkg/saga/`** - Distributed transaction orchestration and state management
- **`pkg/security/`** - Enterprise security (OAuth2, OPA, TLS, audit logging)

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

## Saga Distributed Transactions

Swit provides enterprise-grade distributed transaction management using the Saga pattern.

### Quick Start with Saga

```go
package main

import (
    "context"
    "github.com/innovationmech/swit/pkg/saga"
    "github.com/innovationmech/swit/pkg/saga/base"
)

func main() {
    // Create Saga definition
    def := saga.NewSagaDefinition("order-saga", "v1")
    
    // Add steps with compensation
    def.AddStep("reserve-inventory", reserveInventory, compensateInventory)
    def.AddStep("process-payment", processPayment, refundPayment)
    def.AddStep("create-order", createOrder, cancelOrder)
    
    // Create coordinator
    coordinator := saga.NewCoordinator(storage, publisher)
    
    // Execute Saga
    instance, err := coordinator.Execute(context.Background(), def, orderData)
    if err != nil {
        // Saga failed, compensations executed automatically
    }
}
```

### Saga Features

- **Orchestration & Choreography**: Support for centralized and event-driven patterns
- **Reliable State Management**: PostgreSQL, MySQL, SQLite, and in-memory storage
- **Flexible Retry Strategies**: Exponential backoff, fixed delay, linear backoff
- **Compensation Patterns**: Sequential, parallel, and custom compensation
- **DSL Support**: YAML-based workflow definition
- **Dashboard**: Web UI for monitoring and management
- **Security**: Authentication, RBAC, ACL, and data encryption
- **Observability**: Prometheus metrics, OpenTelemetry tracing, health checks

## Security

Swit provides enterprise-grade security features for building secure microservices.

### Quick Start with Security

```go
package main

import (
    "github.com/innovationmech/swit/pkg/security/oauth2"
    "github.com/innovationmech/swit/pkg/security/opa"
    "github.com/innovationmech/swit/pkg/middleware"
)

func main() {
    // OAuth2/OIDC Authentication
    oauth2Client, _ := oauth2.NewClient(&oauth2.Config{
        Provider:     "keycloak",
        ClientID:     "my-service",
        ClientSecret: os.Getenv("OAUTH2_CLIENT_SECRET"),
        IssuerURL:    "https://auth.example.com/realms/production",
        UseDiscovery: true,
    })
    
    // OPA Policy Engine for Authorization
    opaClient, _ := opa.NewClient(&opa.Config{
        Mode:      "embedded",
        PolicyDir: "./policies",
    })
    
    // Apply middleware
    router.Use(middleware.NewOAuth2Middleware(oauth2Client).Authenticate())
    router.Use(middleware.NewOPAMiddleware(opaClient).Authorize())
}
```

### Security Features

- **Authentication**: OAuth2/OIDC (Keycloak, Auth0, Google, Microsoft, Okta), JWT validation, mTLS
- **Authorization**: OPA policy engine with RBAC and ABAC support
- **Transport Security**: TLS 1.2/1.3, mTLS for service-to-service communication
- **Data Protection**: Encryption at rest, audit logging, sensitive data masking
- **Security Scanning**: Integrated gosec, Trivy, and govulncheck

### Security Documentation

- 📖 [Security Best Practices](https://innovationmech.github.io/swit/guide/security-best-practices.html) - Comprehensive security guidelines
- 🔐 [OAuth2 Integration Guide](https://innovationmech.github.io/swit/guide/oauth2-integration.html) - Authentication setup
- 🛡️ [OPA Policy Guide](https://innovationmech.github.io/swit/guide/opa-policy.html) - Authorization policies

### Saga Documentation

- 📖 [User Guide](https://innovationmech.github.io/swit/saga/user-guide.html) - Getting started and core concepts
- 📚 [API Reference](https://innovationmech.github.io/swit/saga/api-reference.html) - Complete API documentation
- 🎓 [Tutorials](https://innovationmech.github.io/swit/saga/tutorials.html) - Step-by-step guides and best practices
- 🚀 [Deployment Guide](https://innovationmech.github.io/swit/saga/deployment-guide.html) - Production deployment
- 🔧 [Developer Guide](https://innovationmech.github.io/swit/saga/developer-guide.html) - Architecture and extension development

## Examples

### Simple Examples
```bash
# HTTP service
cd examples/simple-http-service && go run main.go

# gRPC service  
cd examples/grpc-service && go run main.go

# Full-featured service
cd examples/full-featured-service && go run main.go

# Full security stack (OAuth2 + OPA + TLS)
cd examples/full-security-stack && go run main.go
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
./bin/saga-dsl-validate --help # Saga DSL validation tool
```

### Saga Examples
```bash
# Saga orchestrator example
cd examples/saga-orchestrator && go run main.go

# Saga choreography example  
cd examples/saga-choreography && go run main.go

# Saga with publisher
cd examples/saga-publisher && go run main.go

# Saga retry patterns
cd examples/saga-retry && go run main.go
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
