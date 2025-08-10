# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Key Development Commands

### Build Commands
- `make build` - Build all services (swit-serve, swit-auth, switctl)
- `make build-dev` - Quick build without quality checks
- `make build-release` - Build release versions for all platforms
- `make all` - Full build pipeline (proto + swagger + tidy + copyright + build)

### Testing Commands
- `make test` - Run all tests with dependencies
- `make test-dev` - Quick test without dependency generation
- `make test-coverage` - Generate test coverage report
- `make test-race` - Run tests with race detection

### Code Quality
- `make tidy` - Run go mod tidy
- `make format` - Format code with gofmt
- `make quality` - Run format and vet checks
- `make lint` - Run golangci-lint (if available)

### API Development
- `make proto` - Generate protobuf code and documentation
- `make proto-generate` - Generate code only
- `make proto-lint` - Check proto files
- `make swagger` - Generate Swagger documentation for all services

### Development Environment
- `make setup-dev` - Setup complete development environment
- `make ci` - Run full CI pipeline locally

## High-Level Architecture

### Base Server Framework
The project uses a unified base server framework located in `pkg/server/` that provides:

- **BaseServer Interface** - Consistent lifecycle management across all services
- **Transport Management** - Unified HTTP and gRPC transport handling via Transport Manager
- **Service Registration** - Service Registrar pattern for pluggable service registration
- **Performance Monitoring** - Built-in performance metrics and monitoring hooks
- **Dependency Injection** - DependencyContainer interface for service dependencies
- **Service Discovery Integration** - Pluggable service discovery with Consul support

Key components:
- `pkg/server/base.go` - Core BaseServerImpl with lifecycle management
- `pkg/server/interfaces.go` - All server interfaces and contracts
- `pkg/server/config.go` - Server configuration management
- `pkg/server/performance.go` - Performance monitoring and metrics

### Service Structure
The project follows a microservice architecture with three main services:

1. **swit-serve** (port 9000) - Main user service
   - User management (CRUD operations)
   - Greeter service (gRPC + HTTP)
   - Notification service 
   - Health checks
   - Stop service for graceful shutdown

2. **swit-auth** (port 9001) - Authentication service
   - JWT-based authentication
   - Token refresh and validation
   - User login/logout
   - Health checks

3. **switctl** - Command-line control tool
   - Health checks
   - Service management
   - Version information

### Transport Layer Architecture
Both services use a unified transport architecture via the base server framework:
- **Transport Manager** - Manages HTTP and gRPC transports
- **Service Registry** - Registers services with both transport types
- **Service Registrar Pattern** - Each service implements ServiceRegistrar interface for transport registration
- **Middleware Management** - Centralized middleware configuration for both HTTP and gRPC

Transport integration:
- Services implement `ServiceRegistrar` interface
- Base server handles transport initialization and lifecycle
- Automatic service registration with both HTTP and gRPC transports
- Health check integration across all transports

### Key Dependencies
- **Gin** - HTTP web framework
- **gRPC** - RPC framework with HTTP gateway support
- **GORM** - ORM for database operations
- **Consul** - Service discovery
- **Zap** - Structured logging
- **Viper** - Configuration management
- **JWT** - Token-based authentication
- **Buf** - Protocol Buffer toolchain

### Database
- **MySQL** - Primary database
- Two separate databases: `user_service_db` (user service) and `auth_service_db` (auth service)
- Database schemas in `scripts/sql/`

### Configuration
- `swit.yaml` - Main service configuration
- `switauth.yaml` - Authentication service configuration
- Uses Viper for configuration management with BaseServer integration

### API Structure
The project uses Buf toolchain for Protocol Buffer management:
- Proto definitions in `api/proto/swit/`
- Generated code in `api/gen/`
- Supports both gRPC and HTTP/REST endpoints via grpc-gateway
- OpenAPI v2 documentation generation

### Service Discovery
- Consul-based service discovery
- Automatic service registration/deregistration via BaseServer
- Health check integration
- ServiceDiscoveryManager abstraction in base server

## Project Structure Overview

```
├── cmd/                    # Application entry points
│   ├── swit-serve/        # Main server executable
│   ├── swit-auth/         # Auth service executable  
│   └── switctl/           # CLI tool executable
├── internal/              # Private application code
│   ├── switserve/        # Main server implementation
│   ├── switauth/         # Auth service implementation
│   └── switctl/          # CLI tool implementation
├── pkg/                   # Shared library code
│   ├── server/           # Base server framework (core architecture)
│   ├── discovery/        # Service discovery
│   ├── middleware/       # HTTP/gRPC middleware
│   ├── transport/        # Transport layer abstractions
│   └── utils/           # Utilities (JWT, hashing)
├── api/                   # API definitions and generated code
│   ├── proto/            # Protocol Buffer definitions
│   ├── gen/              # Generated code (Go, OpenAPI)
│   └── buf.yaml          # Buf toolchain configuration
└── scripts/              # Build scripts and tools
    ├── mk/               # Makefile modules
    ├── tools/            # Development tools
    └── sql/              # Database schemas
```

## Base Server Usage Patterns

### Service Implementation Pattern
1. Implement `ServiceRegistrar` interface in your service
2. Register HTTP handlers via `ServiceRegistry.RegisterHTTPHandler()`
3. Register gRPC services via `ServiceRegistry.RegisterGRPCService()`
4. Use `NewBaseServer()` with your service registrar
5. Call `Start()` and `Stop()` for lifecycle management

### Dependency Injection Pattern
- Implement `DependencyContainer` interface for service dependencies
- Pass to `NewBaseServer()` for automatic lifecycle management
- Dependencies are initialized during server startup
- Automatic cleanup during server shutdown

### Performance Monitoring
- Built-in performance metrics collection
- Configurable performance monitoring hooks
- Memory usage and startup/shutdown time tracking
- Access via `GetPerformanceMetrics()` and `GetPerformanceMonitor()`

## Testing Notes
- Comprehensive unit tests for all services including base server framework
- Test coverage reporting available via `make test-coverage`
- Race detection enabled for concurrent testing via `make test-race`
- Mock-based testing for external dependencies
- Performance regression tests in base server framework
- Integration tests for end-to-end service communication

## Important Development Notes

### Base Server Framework Development
When modifying the base server framework (`pkg/server/`):
- All interfaces are in `interfaces.go` - maintain backward compatibility
- Performance monitoring is built-in - add hooks for new metrics
- Transport management is centralized - avoid direct transport access
- Configuration validation is enforced - implement `Validate()` method
- Dependency lifecycle is managed - implement proper `Close()` methods

### Service Development
When adding new services:
- Extend existing services rather than creating new ones when possible
- Implement `ServiceRegistrar` for base server integration
- Use the transport manager for HTTP/gRPC registration
- Follow the established configuration patterns
- Add appropriate health checks and performance monitoring

### API Development
When modifying APIs:
- Update proto files in `api/proto/swit/`
- Run `make proto` to regenerate code
- Update swagger documentation with `make swagger`
- Test both gRPC and HTTP endpoints
- Maintain API versioning in proto packages