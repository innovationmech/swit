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

### Service Structure
The project follows a microservice architecture with two main services:

1. **swit-serve** (port 9000) - Main user service
   - User management (CRUD operations)
   - Greeter service (gRPC + HTTP)
   - Notification service 
   - Health checks
   - Stop service for graceful shutdown

2. **swit-auth** (port 8090) - Authentication service
   - JWT-based authentication
   - Token refresh and validation
   - User login/logout
   - Health checks

3. **switctl** - Command-line control tool
   - Health checks
   - Service management
   - Version information

### Transport Layer Architecture
Both services use a unified transport architecture:
- **Transport Manager** - Manages HTTP and gRPC transports
- **Service Registry** - Registers services with both transport types
- **Service Registrar Pattern** - Each service implements registrar interface for transport registration

Key files:
- `internal/switserve/transport/registrar.go` - Service registration for switserve
- `internal/switauth/transport/registrar.go` - Service registration for switauth
- `pkg/discovery/manager.go` - Service discovery management

### Key Dependencies
- **Gin** - HTTP web framework
- **gRPC** - RPC framework with HTTP gateway support
- **GORM** - ORM for database operations
- **Consul** - Service discovery
- **Zap** - Structured logging
- **Viper** - Configuration management
- **JWT** - Token-based authentication

### Database
- **MySQL** - Primary database
- Two separate databases: `swit_db` (user service) and `swit_auth_db` (auth service)
- Database schemas in `scripts/sql/`

### Configuration
- `swit.yaml` - Main service configuration
- `switauth.yaml` - Authentication service configuration
- Uses Viper for configuration management

### API Structure
The project uses Buf toolchain for Protocol Buffer management:
- Proto definitions in `api/proto/swit/`
- Generated code in `api/gen/`
- Supports both gRPC and HTTP/REST endpoints via grpc-gateway

### Service Discovery
- Consul-based service discovery
- Automatic service registration/deregistration
- Health check integration
- Manager pattern for service discovery instances

## Project Structure Overview

```
├── cmd/                    # Application entry points
├── internal/              # Private application code
│   ├── switserve/        # Main server (port 9000)
│   ├── switauth/         # Auth service (port 8090)
│   └── switctl/          # CLI tool
├── pkg/                   # Shared library code
│   ├── discovery/        # Service discovery
│   ├── middleware/       # HTTP/gRPC middleware
│   └── utils/           # Utilities (JWT, hashing)
├── api/                   # API definitions and generated code
└── scripts/              # Build scripts and SQL schemas
```

## Testing Notes
- Comprehensive unit tests for all services
- Test coverage reporting available
- Race detection enabled for concurrent testing
- Mock-based testing for external dependencies