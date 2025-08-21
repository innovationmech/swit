# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Key Development Commands

### Framework Development Commands
- `make build` - Build framework components and example services
- `make build-dev` - Quick build without quality checks
- `make build-release` - Build release versions for all platforms
- `make all` - Full build pipeline (proto + swagger + tidy + copyright + build)

### Testing Commands
- `make test` - Run all tests including framework and examples
- `make test-dev` - Quick test without dependency generation
- `make test-coverage` - Generate test coverage report for framework
- `make test-race` - Run tests with race detection
- `make test-advanced TYPE=unit PACKAGE=pkg` - Test specific framework packages

### Code Quality
- `make tidy` - Run go mod tidy
- `make format` - Format code with gofmt
- `make quality` - Run format and vet checks
- `make quality-dev` - Quick quality checks for development

### API Development
- `make proto` - Generate protobuf code and documentation
- `make proto-generate` - Generate code only
- `make proto-lint` - Check proto files
- `make swagger` - Generate Swagger documentation for example services

### Development Environment
- `make setup-dev` - Setup complete framework development environment
- `make setup-quick` - Quick setup for essential components
- `make ci` - Run full CI pipeline locally

### Running Examples
- Run simple HTTP service: `cd examples/simple-http-service && go run main.go`
- Run gRPC service: `cd examples/grpc-service && go run main.go`
- Run reference services: `./bin/swit-serve` or `./bin/swit-auth`

## High-Level Architecture

### Microservice Framework Core
This is a comprehensive Go microservice framework providing production-ready components for building scalable microservices. The framework consists of:

### Core Framework (`pkg/server/`)
- **BusinessServerCore Interface** - Main server interface providing complete lifecycle management
- **BusinessServerImpl** - Complete server implementation with transport coordination
- **BusinessServiceRegistrar** - Interface pattern for services to register with the framework
- **BusinessDependencyContainer** - Dependency injection system with factory patterns
- **Configuration Management** - Comprehensive validation with environment overrides
- **Performance Monitoring** - Built-in metrics collection and monitoring hooks
- **Lifecycle Management** - Phased startup/shutdown with proper resource cleanup

Key components:
- `pkg/server/base.go` - Core server implementation with all framework features
- `pkg/server/interfaces.go` - All framework interfaces and contracts
- `pkg/server/config.go` - Framework configuration structures and validation
- `pkg/server/performance.go` - Performance monitoring and metrics collection
- `pkg/server/dependency.go` - Dependency injection container implementation

### Transport Layer (`pkg/transport/`)
- **TransportCoordinator** - Central coordinator for HTTP and gRPC transports
- **NetworkTransport Interface** - Base interface for transport implementations
- **MultiTransportRegistry** - Service registry handling cross-transport operations
- **Service Registration** - Unified registration patterns for HTTP and gRPC services

Key components:
- `pkg/transport/transport.go` - Main transport coordinator implementation
- `pkg/transport/http.go` - HTTP transport with Gin framework integration
- `pkg/transport/grpc.go` - gRPC transport with advanced configuration
- `pkg/transport/registry_manager.go` - Multi-transport service registry

### Example Services and Reference Implementations

#### Examples (`examples/`)
- **simple-http-service** - Basic HTTP-only service demonstrating framework basics
- **grpc-service** - gRPC service implementation with Protocol Buffers
- **full-featured-service** - Complete framework showcase with all features
- **symmetric_transport** - Transport layer usage patterns

#### Reference Services (`internal/`)
- **switserve** - Complete user management service demonstrating full framework integration
- **switauth** - Authentication service showcasing JWT and security patterns
- **switctl** - CLI tool showing framework integration for command-line applications

### Framework Integration Architecture

#### Service Registration Pattern
All services (examples and reference implementations) follow the same integration pattern:
- Implement `BusinessServiceRegistrar` interface to register with framework
- Framework handles transport initialization and lifecycle automatically
- Automatic service registration with both HTTP and gRPC transports
- Built-in health check integration across all transports

#### Adapter Pattern Integration
Framework uses adapter patterns to bridge different service interfaces:
- HTTP handlers implement `BusinessHTTPHandler` interface
- gRPC services implement `BusinessGRPCService` interface  
- Framework adapters translate between transport-specific and business interfaces
- Unified service metadata and health checking

### Supporting Framework Components

#### Key Dependencies
- **Gin** - HTTP web framework (used in transport layer)
- **gRPC** - RPC framework with grpc-gateway support
- **Consul** - Service discovery (optional, can be disabled)
- **Zap** - Structured logging throughout framework
- **Viper** - Configuration management with validation
- **GORM** - ORM for database operations (used in examples)
- **JWT** - Token-based authentication (used in auth examples)
- **Buf** - Protocol Buffer toolchain for API development

#### Configuration System
Framework provides comprehensive configuration management:
- `ServerConfig` structure with validation and defaults
- Environment variable override support
- YAML configuration file support (examples use `swit.yaml`, `switauth.yaml`)
- Configuration validation with meaningful error messages

#### API and Protocol Buffer Integration
- Proto definitions in `api/proto/swit/` for cross-service contracts
- Generated code in `api/gen/` for Go and OpenAPI
- Supports both gRPC and HTTP/REST endpoints via grpc-gateway
- Buf toolchain for API development and documentation generation

#### Service Discovery Integration
- Optional Consul-based service discovery via `ServiceDiscoveryManager`
- Automatic service registration/deregistration
- Health check integration with service discovery
- Can be disabled for development or standalone deployment

## Project Structure Overview

```
├── cmd/                    # Application entry points for reference services
│   ├── swit-serve/        # User management service main
│   ├── swit-auth/         # Authentication service main  
│   └── switctl/           # CLI tool main
├── pkg/                   # CORE FRAMEWORK COMPONENTS
│   ├── server/           # ★ Base server framework (main framework core)
│   ├── transport/        # ★ Transport layer (HTTP/gRPC coordination)
│   ├── discovery/        # Service discovery integration
│   ├── middleware/       # HTTP/gRPC middleware components
│   ├── types/           # Common type definitions
│   └── utils/           # Utilities (JWT, crypto, hashing)
├── examples/              # Framework usage examples
│   ├── simple-http-service/   # Basic HTTP service example
│   ├── grpc-service/         # gRPC service example
│   ├── full-featured-service/ # Complete framework showcase
│   └── symmetric_transport/   # Transport patterns
├── internal/              # Reference service implementations
│   ├── switserve/        # User management reference service
│   ├── switauth/         # Authentication reference service
│   └── switctl/          # CLI reference implementation
├── api/                   # Protocol Buffer definitions and generated code
│   ├── proto/            # Protocol Buffer definitions
│   ├── gen/              # Generated code (Go, OpenAPI)
│   └── buf.yaml          # Buf toolchain configuration
├── docs/                  # Framework and service documentation
└── scripts/              # Build scripts and development tools
    ├── mk/               # Makefile modules for build automation
    ├── tools/            # Development and CI tools
    └── sql/              # Database schemas for reference services
```

## Framework Usage Patterns

### Core Service Implementation Pattern
1. **Implement `BusinessServiceRegistrar`** - Main interface for framework integration
   ```go
   func (s *MyService) RegisterServices(registry BusinessServiceRegistry) error {
       // Register HTTP handlers, gRPC services, health checks
   }
   ```

2. **Register Service Components** with the registry:
   - HTTP handlers via `RegisterBusinessHTTPHandler()`
   - gRPC services via `RegisterBusinessGRPCService()`  
   - Health checks via `RegisterBusinessHealthCheck()`

3. **Create and Start Framework Server**:
   ```go
   config := &server.ServerConfig{...}
   baseServer, _ := server.NewBusinessServerCore(config, myService, dependencies)
   baseServer.Start(ctx)
   ```

### Dependency Injection Framework Pattern
- **Implement `BusinessDependencyContainer`** for service dependencies
- **Factory Pattern Support** - Register singleton and transient dependencies
- **Automatic Lifecycle** - Dependencies initialized during server startup
- **Graceful Cleanup** - Automatic resource cleanup during server shutdown
- **Service Lookup** - Access dependencies via `GetService(name string)`

### HTTP Handler Implementation
```go
type MyHTTPHandler struct{}

func (h *MyHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    ginRouter.GET("/api/v1/my-endpoint", h.handleEndpoint)
    return nil
}

func (h *MyHTTPHandler) GetServiceName() string {
    return "my-service"
}
```

### gRPC Service Implementation
```go
type MyGRPCService struct{}

func (s *MyGRPCService) RegisterGRPC(server interface{}) error {
    grpcServer := server.(*grpc.Server)
    mypb.RegisterMyServiceServer(grpcServer, s)
    return nil
}

func (s *MyGRPCService) GetServiceName() string {
    return "my-grpc-service"
}
```

### Performance Monitoring Integration
- **Built-in Metrics Collection** - Memory usage, startup/shutdown timing
- **Performance Hooks** - Configurable monitoring hooks for custom metrics
- **Threshold Monitoring** - Automatic alerts on performance threshold violations
- **Access Methods** - `GetPerformanceMetrics()` and `GetPerformanceMonitor()`

## Framework Testing Strategy

### Framework Core Testing
- **Unit Tests** - Comprehensive coverage for `pkg/server/` and `pkg/transport/`
- **Integration Tests** - End-to-end framework functionality testing
- **Performance Tests** - Performance regression testing for framework components
- **Race Detection** - Concurrent testing with `make test-race`
- **Coverage Reports** - Framework test coverage via `make test-coverage`

### Example Service Testing
- **Reference Implementation Tests** - Tests for `internal/` services as framework usage examples
- **Example Tests** - Tests in `examples/` demonstrating testing patterns
- **Mock-based Testing** - External dependency mocking patterns
- **Transport Testing** - HTTP and gRPC endpoint testing strategies

### Testing Commands for Framework Development
- `make test-advanced TYPE=unit PACKAGE=pkg` - Test framework packages specifically
- `make test-advanced TYPE=integration` - Run integration tests
- `make test-advanced TYPE=performance` - Performance regression tests

## Important Framework Development Guidelines

### Core Framework Development (`pkg/server/`, `pkg/transport/`)
**Interface Stability** - All public interfaces in `interfaces.go` - maintain backward compatibility
**Performance Integration** - Built-in performance monitoring - add hooks for new metrics  
**Transport Abstraction** - Centralized transport management - avoid direct transport access
**Configuration Validation** - Enforced validation - implement `Validate()` method on new config types
**Dependency Lifecycle** - Managed container lifecycle - implement proper `Close()` methods
**Thread Safety** - All framework components must be thread-safe with proper mutex usage

### Example and Reference Service Development
**Framework Integration** - Always implement `BusinessServiceRegistrar` for framework integration
**Pattern Consistency** - Follow established patterns from existing examples
**Transport Registration** - Use framework's transport coordination, not direct registration
**Configuration Patterns** - Follow `ServerConfig` patterns with validation
**Health Checks** - Implement meaningful health checks for service dependencies
**Documentation** - Examples should be well-documented to serve as framework tutorials

### Framework Extension Development
**New Examples** - Add to `examples/` directory following existing patterns
**Reference Services** - Extend `internal/` services to demonstrate new framework features
**Interface Extensions** - Add new interfaces in framework core if extending capabilities
**Backward Compatibility** - Ensure new features don't break existing framework users

### API and Protocol Development
**Protocol Buffer Management** - Update proto files in `api/proto/swit/` for cross-service contracts
**Code Generation** - Run `make proto` to regenerate all API code
**Documentation Generation** - Update OpenAPI docs with `make swagger`
**Transport Testing** - Test both gRPC and HTTP endpoints for dual-protocol support
**API Versioning** - Maintain proper versioning in proto packages for framework APIs


## Workflow
- Must create or update unit test when you’re done making a series of code changes
- Make sure tests your are making or updates all pass
- Prefer running single tests, and not the whole test suite, for performance