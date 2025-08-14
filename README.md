# Swit

[![CI](https://github.com/innovationmech/swit/workflows/CI/badge.svg)](https://github.com/innovationmech/swit/actions/workflows/ci.yml)
[![Security Checks](https://github.com/innovationmech/swit/workflows/Security%20Checks/badge.svg)](https://github.com/innovationmech/swit/actions/workflows/security-checks.yml)
[![codecov](https://codecov.io/gh/innovationmech/swit/branch/master/graph/badge.svg)](https://codecov.io/gh/innovationmech/swit)
[![Go Report Card](https://goreportcard.com/badge/github.com/innovationmech/swit)](https://goreportcard.com/report/github.com/innovationmech/swit)
[![Go Reference](https://pkg.go.dev/badge/github.com/innovationmech/swit.svg)](https://pkg.go.dev/github.com/innovationmech/swit)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.24-blue.svg)
[![GitHub release](https://img.shields.io/github/release/innovationmech/swit.svg)](https://github.com/innovationmech/swit/releases)
[![License](https://img.shields.io/github/license/innovationmech/swit.svg)](LICENSE)
[![GitHub issues](https://img.shields.io/github/issues/innovationmech/swit.svg)](https://github.com/innovationmech/swit/issues)
[![GitHub stars](https://img.shields.io/github/stars/innovationmech/swit.svg)](https://github.com/innovationmech/swit/stargazers)

Swit is a comprehensive microservice framework for Go that provides a unified, production-ready foundation for building scalable microservices. Built with a focus on developer productivity and architectural consistency, Swit offers a complete base server framework, unified transport layer, dependency injection system, and comprehensive tooling for rapid microservice development.

## Framework Features

- **Base Server Framework**: Complete server lifecycle management with `BusinessServerCore` interface and unified service registration patterns
- **Unified Transport Layer**: Seamless HTTP and gRPC transport coordination through `TransportCoordinator` with pluggable transport architecture
- **Dependency Injection System**: Factory-based dependency container with singleton/transient support and automatic lifecycle management
- **Configuration Management**: Comprehensive configuration validation with environment-based overrides and sensible defaults
- **Performance Monitoring**: Built-in metrics collection, performance profiling, and monitoring hooks with threshold violation detection
- **Service Discovery Integration**: Consul-based service registration with health check integration and automatic deregistration
- **Middleware Framework**: Configurable middleware stack for both HTTP and gRPC transports including CORS, rate limiting, and timeouts
- **Health Check System**: Comprehensive health monitoring with service aggregation and timeout handling
- **Graceful Lifecycle Management**: Phased startup/shutdown with proper resource cleanup and error handling
- **Protocol Buffer Integration**: Buf toolchain support for API versioning and automatic documentation generation
- **Example Services**: Complete reference implementations demonstrating framework usage patterns and best practices

## Framework Architecture

The Swit framework consists of the following core components:

### Core Framework (`pkg/server/`)
- **BusinessServerCore**: Main server interface providing lifecycle management, transport coordination, and service health monitoring
- **BusinessServerImpl**: Complete server implementation with transport management, service discovery, and performance monitoring
- **BusinessServiceRegistrar**: Interface pattern for services to register with the framework's transport layer
- **BusinessDependencyContainer**: Dependency injection system with factory patterns and lifecycle management

### Transport Layer (`pkg/transport/`)
- **TransportCoordinator**: Central coordinator managing multiple transport instances (HTTP/gRPC) with unified service registration
- **NetworkTransport**: Base interface for transport implementations with pluggable architecture
- **MultiTransportRegistry**: Service registry manager handling cross-transport operations and health checking

### Example Services (`internal/`)
- **switserve**: User management service demonstrating complete framework usage with HTTP/gRPC endpoints
- **switauth**: Authentication service showcasing JWT integration and service-to-service communication
- **switctl**: Command-line tool example showing framework integration patterns

### Framework Support (`pkg/`)
- **Discovery**: Consul-based service discovery with automatic registration/deregistration
- **Middleware**: HTTP and gRPC middleware stack with CORS, rate limiting, timeout, and authentication support
- **Types**: Common type definitions and health check abstractions
- **Utils**: Cryptographic utilities, JWT handling, and security components

## Framework API Architecture

The Swit framework provides comprehensive API development support through the Buf toolchain for gRPC APIs:

```
api/
├── buf.yaml              # Buf main configuration
├── buf.gen.yaml          # Code generation configuration
├── buf.lock              # Dependency lock file
└── proto/               # Protocol Buffer definitions
    └── swit/
        ├── common/
        │   └── v1/
        │       ├── common.proto
        │       └── health.proto
        ├── communication/
        │   └── v1/
        │       └── notification.proto
        ├── interaction/
        │   └── v1/
        │       └── greeter.proto
        └── user/
            └── v1/
                ├── auth.proto
                └── user.proto
```

Generated artifacts such as `api/gen/` and Swagger documentation are produced by the build tools (`make proto`, `make swagger`) and are not committed to the repository.

### API Design Principles

- **Versioning**: All APIs have clear version numbers (v1, v2, ...)
- **Modular**: Organize proto files by service domain
- **Dual Protocol**: Support both gRPC and HTTP/REST
- **Automation**: Use Buf toolchain for automatic code and documentation generation

## Framework Examples and Reference Implementations

The Swit framework includes comprehensive examples demonstrating various usage patterns:

### Simple Examples (`examples/`)

#### `examples/simple-http-service/`
- **Purpose**: Basic HTTP-only service demonstration
- **Features**: RESTful API endpoints, health checks, graceful shutdown
- **Best For**: Getting started with the framework, HTTP-only services
- **Key Concepts**: `BusinessServiceRegistrar` implementation, HTTP routing patterns

#### `examples/grpc-service/`
- **Purpose**: gRPC service implementation showcase
- **Features**: Protocol Buffer definitions, gRPC server setup, streaming support
- **Best For**: gRPC-focused microservices, inter-service communication
- **Key Concepts**: `BusinessGRPCService` implementation, Protocol Buffer integration

#### `examples/full-featured-service/`
- **Purpose**: Complete framework feature demonstration
- **Features**: HTTP + gRPC, dependency injection, service discovery, middleware
- **Best For**: Production-ready service patterns, framework evaluation
- **Key Concepts**: Multi-transport services, advanced configuration, monitoring

### Reference Services (`internal/`)

#### `internal/switserve/` - User Management Service
- **Purpose**: Comprehensive user management microservice
- **Architecture**: Full framework integration with database, caching, and external service communication
- **Features**:
  - User CRUD operations (HTTP REST + gRPC)
  - Greeter service with streaming support
  - Notification system integration
  - Health monitoring and graceful shutdown
  - Database integration with GORM
  - Middleware stack demonstration

#### `internal/switauth/` - Authentication Service  
- **Purpose**: JWT-based authentication microservice
- **Architecture**: Secure authentication patterns with token management
- **Features**:
  - User login/logout (HTTP + gRPC)
  - JWT token generation and validation  
  - Token refresh and revocation
  - Password reset workflows
  - Service-to-service authentication
  - Redis integration for session management

#### `internal/switctl/` - CLI Tool
- **Purpose**: Command-line administration tool
- **Architecture**: Framework integration patterns for CLI applications
- **Features**:
  - Health check commands
  - Service management operations
  - Version information and diagnostics

### Usage Patterns Demonstrated

1. **Service Registration**: Multiple implementation patterns for HTTP and gRPC services
2. **Configuration Management**: Environment-based configuration with validation
3. **Dependency Injection**: Database connections, Redis clients, external service clients
4. **Middleware Integration**: Authentication, CORS, rate limiting, logging
5. **Health Monitoring**: Service health checks and readiness probes
6. **Performance Monitoring**: Metrics collection and performance profiling
7. **Service Discovery**: Consul registration and service lookup patterns
8. **Testing Strategies**: Unit tests, integration tests, and performance benchmarks

## Framework Interfaces & Patterns

### Core Server Interfaces

#### `BusinessServerCore`
Main server interface providing complete lifecycle management:
```go
type BusinessServerCore interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Shutdown() error
    GetHTTPAddress() string
    GetGRPCAddress() string
    GetTransports() []transport.NetworkTransport
    GetTransportStatus() map[string]TransportStatus
}
```

#### `BusinessServiceRegistrar`
Interface for services to register with the framework:
```go
type BusinessServiceRegistrar interface {
    RegisterServices(registry BusinessServiceRegistry) error
}
```

#### `BusinessServiceRegistry` 
Registry interface for different service types:
```go
type BusinessServiceRegistry interface {
    RegisterBusinessHTTPHandler(handler BusinessHTTPHandler) error
    RegisterBusinessGRPCService(service BusinessGRPCService) error
    RegisterBusinessHealthCheck(check BusinessHealthCheck) error
}
```

### Transport Layer Interfaces

#### `BusinessHTTPHandler`
Interface for HTTP service implementations:
```go
type BusinessHTTPHandler interface {
    RegisterRoutes(router interface{}) error
    GetServiceName() string
}
```

#### `BusinessGRPCService`
Interface for gRPC service implementations:
```go
type BusinessGRPCService interface {
    RegisterGRPC(server interface{}) error
    GetServiceName() string
}
```

#### `BusinessHealthCheck`
Interface for service health monitoring:
```go
type BusinessHealthCheck interface {
    Check(ctx context.Context) error
    GetServiceName() string
}
```

### Dependency Management Interfaces

#### `BusinessDependencyContainer`
Dependency injection and lifecycle management:
```go
type BusinessDependencyContainer interface {
    Close() error
    GetService(name string) (interface{}, error)
}
```

#### `BusinessDependencyRegistry`
Extended dependency management with factory patterns:
```go
type BusinessDependencyRegistry interface {
    BusinessDependencyContainer
    Initialize(ctx context.Context) error
    RegisterSingleton(name string, factory DependencyFactory) error
    RegisterTransient(name string, factory DependencyFactory) error
    RegisterInstance(name string, instance interface{}) error
}
```

### Configuration Interfaces

#### `ConfigValidator`
Configuration validation and defaults:
```go
type ConfigValidator interface {
    Validate() error
    SetDefaults()
}
```

### Service Implementation Examples

The framework includes working examples of these interfaces:
- **HTTP Services**: RESTful APIs with Gin router integration
- **gRPC Services**: Protocol Buffer service implementations
- **Health Checks**: Database connectivity, external service checks
- **Dependency Injection**: Database connections, Redis clients, external APIs
- **Configuration**: Environment-based config with validation

## Requirements

### Framework Core Requirements
- **Go 1.24+** - Modern Go version with generics support
- **Git** - For framework and example code management

### Optional Dependencies (Service-Specific)
- **MySQL 8.0+** - For database-backed services (demonstrated in examples)
- **Redis 6.0+** - For caching and session management (used in auth examples)
- **Consul 1.12+** - For service discovery (optional, can be disabled)

### Development Tools
- **Buf CLI 1.0+** - For Protocol Buffer API development
- **Docker 20.10+** - For containerized deployment and development
- **Make** - For build automation (standard on most systems)

## Quick Start

### 1. Get the Framework
```bash
git clone https://github.com/innovationmech/swit.git
cd swit
go mod download
```

### 2. Create a Simple Service
```go
// main.go
package main

import (
    "context"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/server"
)

// MyService implements the BusinessServiceRegistrar interface
type MyService struct{}

func (s *MyService) RegisterServices(registry server.BusinessServiceRegistry) error {
    httpHandler := &MyHTTPHandler{}
    return registry.RegisterBusinessHTTPHandler(httpHandler)
}

// MyHTTPHandler implements the BusinessHTTPHandler interface
type MyHTTPHandler struct{}

func (h *MyHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    ginRouter.GET("/hello", h.handleHello)
    return nil
}

func (h *MyHTTPHandler) GetServiceName() string {
    return "my-service"
}

func (h *MyHTTPHandler) handleHello(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "Hello from Swit framework!"})
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
    
    // Server running on :8080
    select {} // Keep running
}
```

### 3. Run Your Service
```bash
go run main.go
curl http://localhost:8080/hello
```

### 4. Explore Examples
```bash
# Simple HTTP service
cd examples/simple-http-service
go run main.go

# gRPC service example
cd examples/grpc-service  
go run main.go

# Full-featured service
cd examples/full-featured-service
go run main.go
```

### 5. Build Framework Components
```bash
# Build all framework components and examples
make build

# Quick development build
make build-dev

# Run example services
./bin/swit-serve    # User management example
./bin/swit-auth     # Authentication example
```

## Framework Development

### Development Environment Setup
```bash
# Setup complete framework development environment
make setup-dev

# Quick setup for essential components only
make setup-quick

# Setup individual development tools
make proto-setup    # Protocol Buffer toolchain
make swagger-setup  # OpenAPI documentation tools
make quality-setup  # Code quality tools
```

### Framework Development Commands

#### Framework API Development
```bash
# Complete API development workflow
make proto          # Generate protobuf code + docs
make swagger        # Generate OpenAPI documentation

# Quick development iterations
make proto-dev      # Skip dependency checks
make swagger-dev    # Skip formatting steps

# Advanced API operations
make proto-advanced OPERATION=format    # Format proto files
make proto-advanced OPERATION=lint      # Lint proto definitions
make proto-advanced OPERATION=breaking  # Check breaking changes
make proto-advanced OPERATION=docs      # Generate documentation only
```

#### Framework Extension Development
```bash
# Build framework components and examples
make build          # Full framework build
make build-dev      # Quick build (skip quality checks)
make build-release  # Multi-platform release build

# Framework testing
make test           # Complete test suite
make test-dev       # Quick tests (skip codegen)
make test-coverage  # Generate coverage reports
make test-race      # Race condition detection
```

### Framework Extension Workflow

1. **Create Your Service**
   ```bash
   # Create service directory
   mkdir my-service
   cd my-service
   
   # Initialize with framework dependency
   go mod init my-service
   go get github.com/innovationmech/swit
   ```

2. **Implement Framework Interfaces**
   ```go
   // Implement BusinessServiceRegistrar
   type MyService struct{}
   
   func (s *MyService) RegisterServices(registry server.BusinessServiceRegistry) error {
       // Register your HTTP/gRPC handlers
       return nil
   }
   ```

3. **Configure and Test**
   ```bash
   # Generate proto code if using gRPC
   make proto-generate
   
   # Build and test your service
   go build .
   go test ./...
   
   # Run with framework
   ./my-service
   ```

4. **Framework Integration Testing**
   ```bash
   # Test with framework examples
   cd examples/simple-http-service
   go run main.go
   
   # Integration testing
   make test-integration
   ```

### Contributing to the Framework

1. **Framework Core Development**
   ```bash
   # Work on pkg/server/ or pkg/transport/
   vim pkg/server/interfaces.go
   
   # Test framework changes
   make test-advanced TYPE=unit PACKAGE=pkg
   ```

2. **Add New Examples**
   ```bash
   # Create new example service
   mkdir examples/my-example
   # Follow example patterns from existing services
   ```

3. **Documentation Updates**
   ```bash
   # Update framework documentation
   vim pkg/server/CLAUDE.md
   vim pkg/transport/CLAUDE.md
   ```

## Framework Configuration

### Core Server Configuration
The framework uses `ServerConfig` structure for comprehensive server setup:

```go
type ServerConfig struct {
    ServiceName     string           // Service identification
    HTTP            HTTPConfig       // HTTP transport configuration  
    GRPC            GRPCConfig       // gRPC transport configuration
    Discovery       DiscoveryConfig  // Service discovery settings
    Middleware      MiddlewareConfig // Middleware configuration
    ShutdownTimeout time.Duration    // Graceful shutdown timeout
}
```

### HTTP Transport Configuration
```go
type HTTPConfig struct {
    Port         string            // Listen port (e.g., "8080")
    Address      string            // Listen address (e.g., ":8080")
    Enabled      bool              // Enable HTTP transport
    EnableReady  bool              // Ready channel for testing
    TestMode     bool              // Test mode settings
    ReadTimeout  time.Duration     // Read timeout
    WriteTimeout time.Duration     // Write timeout
    IdleTimeout  time.Duration     // Idle timeout
    Headers      map[string]string // Default headers
    Middleware   HTTPMiddleware    // Middleware configuration
}
```

### gRPC Transport Configuration
```go
type GRPCConfig struct {
    Port                string              // Listen port (e.g., "9080")
    Address             string              // Listen address
    Enabled             bool                // Enable gRPC transport
    EnableKeepalive     bool                // Enable keepalive
    EnableReflection    bool                // Enable reflection
    EnableHealthService bool                // Enable health service
    MaxRecvMsgSize      int                 // Max receive message size
    MaxSendMsgSize      int                 // Max send message size
    KeepaliveParams     GRPCKeepaliveParams // Keepalive parameters
}
```

### Service Discovery Configuration
```go
type DiscoveryConfig struct {
    Enabled     bool     // Enable service discovery
    Address     string   // Consul address (e.g., "localhost:8500")
    ServiceName string   // Service name for registration
    Tags        []string // Service tags
    CheckPath   string   // Health check path
    CheckInterval string // Health check interval
}
```

### Example Framework Configuration (YAML)
```yaml
service_name: "my-microservice"
shutdown_timeout: "30s"

http:
  enabled: true
  port: "8080" 
  read_timeout: "30s"
  write_timeout: "30s"
  middleware:
    enable_cors: true
    enable_logging: true
    enable_timeout: true

grpc:
  enabled: true
  port: "9080"
  enable_keepalive: true
  enable_reflection: true
  enable_health_service: true

discovery:
  enabled: true
  address: "127.0.0.1:8500"
  service_name: "my-microservice"
  tags: ["v1", "production"]
  check_path: "/health"
  check_interval: "10s"

middleware:
  enable_cors: true
  enable_logging: true
  cors:
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE"]
    allowed_headers: ["*"]
```

### Environment Variable Configuration
```bash
# Service configuration
SERVICE_NAME=my-microservice
SHUTDOWN_TIMEOUT=30s

# HTTP transport
HTTP_ENABLED=true
HTTP_PORT=8080
HTTP_READ_TIMEOUT=30s

# gRPC transport  
GRPC_ENABLED=true
GRPC_PORT=9080
GRPC_ENABLE_REFLECTION=true

# Service discovery
DISCOVERY_ENABLED=true
CONSUL_ADDRESS=localhost:8500
DISCOVERY_SERVICE_NAME=my-microservice
```

## Docker Deployment

### Build Images
```bash
make docker
```

### Run Containers
```bash
# Run user service
docker run -d -p 9000:9000 -p 10000:10000 --name swit-serve swit-serve:latest

# Run authentication service
docker run -d -p 9001:9001 --name swit-auth swit-auth:latest
```

### Using Docker Compose
```bash
docker-compose up -d
```

## Testing

### Run All Tests
```bash
make test
```

### Quick Development Testing
```bash
make test-dev
```

### Test Coverage
```bash
make test-coverage
```

### Advanced Testing
```bash
# Run specific test types
make test-advanced TYPE=unit
make test-advanced TYPE=race
make test-advanced TYPE=bench

# Run tests for specific packages
make test-advanced TYPE=unit PACKAGE=internal
make test-advanced TYPE=unit PACKAGE=pkg
```

## Development Environment

### Setup Development Environment
```bash
# Complete development setup (recommended)
make setup-dev

# Quick setup for minimal requirements
make setup-quick
```

### Available Services and Ports
- **swit-serve**: HTTP: 9000, gRPC: 10000
- **swit-auth**: HTTP: 9001, gRPC: 50051
- **switctl**: CLI tool (no HTTP/gRPC endpoints)

### Development Tools

#### Code Quality
```bash
# Standard quality checks (recommended for CI/CD)
make quality

# Quick quality checks (for development)
make quality-dev

# Setup quality tools
make quality-setup
```

#### Code Formatting and Linting
```bash
# Format code
make format

# Check code
make vet

# Lint code
make lint

# Security scan
make security
```

#### Dependency Management
```bash
# Tidy Go modules
make tidy
```

### Build Commands

#### Standard Build
```bash
# Build all services (development mode)
make build

# Quick build (skip quality checks)
make build-dev

# Release build (all platforms)
make build-release
```

#### Advanced Build
```bash
# Build specific service for specific platform
make build-advanced SERVICE=swit-serve PLATFORM=linux/amd64
make build-advanced SERVICE=swit-auth PLATFORM=darwin/arm64
```

### Cleaning

```bash
# Standard clean (all generated files)
make clean

# Quick clean (build outputs only)
make clean-dev

# Deep clean (reset environment)
make clean-setup

# Advanced clean (specific types)
make clean-advanced TYPE=build
make clean-advanced TYPE=proto
make clean-advanced TYPE=swagger
```

### CI/CD and Copyright Management

#### CI Pipeline
```bash
# Run CI pipeline (automated testing and quality checks)
make ci
```

#### Copyright Management
```bash
# Check and fix copyright headers
make copyright

# Only check copyright headers
make copyright-check

# Setup copyright for new project
make copyright-setup
```

### Docker Development

```bash
# Standard Docker build (production)
make docker

# Quick Docker build (development with cache)
make docker-dev

# Setup Docker development environment
make docker-setup

# Advanced Docker operations
make docker-advanced OPERATION=build COMPONENT=images SERVICE=auth
```

## Makefile Command Reference

The project uses a comprehensive Makefile system with organized commands. Here's a quick reference:

### Core Development Commands
```bash
make all              # Complete build pipeline (proto + swagger + tidy + copyright + build)
make setup-dev        # Setup complete development environment
make setup-quick      # Quick setup with minimal components
make ci               # Run CI pipeline
```

### Build Commands
```bash
make build            # Standard build (development mode)
make build-dev        # Quick build (skip quality checks)
make build-release    # Release build (all platforms)
make build-advanced   # Advanced build with SERVICE and PLATFORM parameters
```

### Test Commands
```bash
make test             # Run all tests (with dependency generation)
make test-dev         # Quick development testing
make test-coverage    # Generate coverage reports
make test-advanced    # Advanced testing with TYPE and PACKAGE parameters
```

### Quality Commands
```bash
make quality          # Standard quality checks (CI/CD)
make quality-dev      # Quick quality checks (development)
make quality-setup    # Setup quality tools
make tidy             # Tidy Go modules
make format           # Format code
make vet              # Code checks
make lint             # Lint code
make security         # Security scan
```

### API Development Commands
```bash
make proto            # Generate protobuf code
make proto-dev        # Quick proto generation
make proto-setup      # Setup protobuf tools
make swagger          # Generate swagger documentation
make swagger-dev      # Quick swagger generation
make swagger-setup    # Setup swagger tools
```

### Clean Commands
```bash
make clean            # Standard clean (all generated files)
make clean-dev        # Quick clean (build outputs only)
make clean-setup      # Deep clean (reset environment)
make clean-advanced   # Advanced clean with TYPE parameter
```

### Docker Commands
```bash
make docker           # Standard Docker build
make docker-dev       # Quick Docker build (with cache)
make docker-setup     # Setup Docker development environment
make docker-advanced  # Advanced Docker operations
```

### Copyright Commands
```bash
make copyright        # Check and fix copyright headers
make copyright-check  # Only check copyright headers
make copyright-setup  # Setup copyright for new project
```

### Help Commands
```bash
make help             # Show all available commands with descriptions
```

For detailed command options and parameters, run `make help` or refer to the specific `.mk` files in `scripts/mk/`.

### Framework Usage Examples

#### Example Services (Reference Implementations)

#### `examples/simple-http-service/`
- **HTTP**: `http://localhost:8080` (configurable)
- **Purpose**: Basic framework demonstration
- **Endpoints**:
  - `GET /api/v1/hello?name=<name>` - Greeting endpoint
  - `GET /api/v1/status` - Service status
  - `POST /api/v1/echo` - Echo request body
  - `GET /health` - Health check (auto-registered)

#### `internal/switserve/` (User Management Reference)
- **HTTP**: `http://localhost:9000`
- **gRPC**: `http://localhost:10000`
- **Purpose**: Complete framework feature demonstration
- **Framework Features Demonstrated**:
  - Multi-transport service registration
  - Database integration patterns
  - Health check implementation
  - Dependency injection usage
  - Middleware configuration

#### `internal/switauth/` (Authentication Reference)
- **HTTP**: `http://localhost:9001`
- **gRPC**: `http://localhost:50051`
- **Purpose**: Authentication service patterns
- **Framework Features Demonstrated**:
  - JWT middleware integration
  - Service-to-service communication
  - Redis dependency injection
  - Secure configuration patterns

#### Framework Patterns Available

1. **HTTP Service Registration**
   ```go
   func (h *MyHandler) RegisterRoutes(router interface{}) error {
       ginRouter := router.(*gin.Engine)
       ginRouter.GET("/api/v1/my-endpoint", h.handleEndpoint)
       return nil
   }
   ```

2. **gRPC Service Registration**
   ```go
   func (s *MyService) RegisterGRPC(server interface{}) error {
       grpcServer := server.(*grpc.Server)
       mypb.RegisterMyServiceServer(grpcServer, s)
       return nil
   }
   ```

3. **Health Check Implementation**
   ```go
   func (h *MyHealthCheck) Check(ctx context.Context) error {
       // Implement your health check logic
       return nil
   }
   ```


## Framework Documentation

### Core Framework Guides
- [Base Server Framework](docs/base-server-framework.md) - Complete framework architecture and usage patterns
- [Configuration Reference](docs/configuration-reference.md) - Comprehensive configuration documentation
- [Service Development Guide](docs/service-development-guide.md) - How to build services with the framework

### Framework Components Documentation
- [Base Server Framework](pkg/server/CLAUDE.md) - Core server interfaces and implementation patterns
- [Transport Layer](pkg/transport/CLAUDE.md) - HTTP and gRPC transport coordination
- [Service Architecture Analysis](docs/service-architecture-analysis.md) - Framework design principles

### Example Service Documentation
- [Example Services Overview](examples/README.md) - Guide to all framework examples
- [Simple HTTP Service](examples/simple-http-service/README.md) - Basic framework usage
- [gRPC Service Example](examples/grpc-service/README.md) - gRPC integration patterns
- [Full-Featured Service](examples/full-featured-service/README.md) - Complete framework showcase

### Reference Service Documentation
- [SwitServe Service](docs/services/switserve/README.md) - User management service implementation
- [SwitAuth Service](docs/services/switauth/README.md) - Authentication service patterns

### API and Protocol Documentation
- [Protocol Buffer Definitions](api/proto/) - Source API specifications
- Generated Swagger reference (`docs/generated/`, via `make swagger`)

### Development and Contribution
- [Development Guide](DEVELOPMENT.md) - Framework development environment setup
- [Code of Conduct](CODE_OF_CONDUCT.md) - Community guidelines
- [Security Policy](SECURITY.md) - Security practices and reporting

## Contributing

We welcome contributions to the Swit microservice framework! Whether you're fixing bugs, improving documentation, adding examples, or enhancing framework features, your contributions are valued.

### Ways to Contribute

1. **Framework Core Development** - Enhance `pkg/server/` and `pkg/transport/` components
2. **Example Services** - Add new examples in `examples/` directory
3. **Documentation** - Improve framework documentation and guides
4. **Testing** - Add tests for framework components and examples
5. **Bug Reports** - Report issues with framework functionality
6. **Feature Requests** - Suggest new framework capabilities

### Getting Started

1. Fork the repository and clone your fork
2. Set up the development environment: `make setup-dev`
3. Run tests to ensure everything works: `make test`
4. Make your changes following the existing patterns
5. Add tests for new functionality
6. Submit a pull request with a clear description

Please read our [Code of Conduct](CODE_OF_CONDUCT.md) before contributing to ensure a positive and inclusive environment for all community members.

## License

MIT License - See [LICENSE](LICENSE) file for details
