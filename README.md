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

Swit is a microservice-based backend system built with Go, featuring modular design for user management, authentication, and service discovery. The project uses the Gin framework for HTTP requests, GORM for data persistence, and supports gRPC protocol for inter-service communication.

## Core Features

- **Microservice Architecture**: Modular design supporting independent deployment and scaling
- **Authentication System**: Complete JWT-based authentication with token refresh support
- **User Management**: Full CRUD operations and permission management for users
- **Service Discovery**: Consul integration for service registration and discovery
- **Database Support**: MySQL for data persistence
- **Dual Protocol Support**: Both HTTP REST API and gRPC protocol support
- **Modern API**: Buf toolchain for managing gRPC APIs with versioning and automatic documentation generation
- **Health Check**: Built-in health check endpoints for service monitoring
- **Docker Support**: Containerized deployment solution
- **OpenAPI Documentation**: Integrated Swagger UI for interactive API documentation

## System Architecture

The project consists of the following main components:

1. **swit-serve** - Main user service (port 9000)
2. **swit-auth** - Authentication service (port 9001)
3. **switctl** - Command-line control tool

## Modern API Architecture

The project has completed API modernization migration, using Buf toolchain to manage gRPC APIs:

```
api/
├── buf.yaml              # Buf main configuration
├── buf.gen.yaml          # Code generation configuration
├── buf.lock              # Dependency lock file
├── proto/                # Protocol Buffer definitions
│   └── swit/
│       └── v1/          # API version 1
│           └── greeter/ # Greeter service
├── gen/                 # Generated code
│   ├── go/             # Go code
│   └── openapiv2/      # OpenAPI documentation
└── docs/               # Documentation
    ├── README.md       # API usage documentation
    └── MIGRATION.md    # Migration guide
```

### API Design Principles

- **Versioning**: All APIs have clear version numbers (v1, v2, ...)
- **Modular**: Organize proto files by service domain
- **Dual Protocol**: Support both gRPC and HTTP/REST
- **Automation**: Use Buf toolchain for automatic code and documentation generation

## Current Implemented Services

### 1. Greeter Service (gRPC)
**HTTP Port**: 9000  
**gRPC Port**: 10000 (HTTP port + 1000)  
**Protocol**: gRPC + HTTP  
**Implemented Methods**:
- `SayHello` - Simple greeting functionality ✅
- `SayHelloStream` - Streaming greeting functionality ✅

**HTTP Endpoints**:
- `POST /v1/greeter/hello` - Send greeting request
- `POST /v1/greeter/hello/stream` - Send streaming greeting request

**gRPC Endpoints**:
- `greeter.v1.GreeterService/SayHello` - gRPC greeting service
- `greeter.v1.GreeterService/SayHelloStream` - gRPC streaming greeting service

### 2. Auth Service (HTTP REST)
**Port**: 9001  
**Protocol**: HTTP REST + gRPC  
**Main Features**:
- User login/logout
- JWT Token management
- Token refresh and validation
- Password reset
- Email verification
- Change password

### 3. User Service (HTTP REST)
**HTTP Port**: 9000  
**gRPC Port**: 10000 (HTTP port + 1000)  
**Protocol**: HTTP REST + gRPC  
**Main Features**:
- User registration
- User information management
- User query and deletion
- User listing with pagination
- User role and permission management

## Requirements

- Go 1.24+
- MySQL 8.0+
- Consul 1.12+
- Buf CLI 1.0+ (for API development)
- Docker 20.10+ (for containerized deployment)

## Quick Start

### 1. Clone Repository
```bash
git clone https://github.com/innovationmech/swit.git
cd swit
```

### 2. Install Dependencies
```bash
go mod download
```

### 3. Environment Configuration
```bash
# Copy configuration files
cp swit.yaml.example swit.yaml
cp switauth.yaml.example switauth.yaml

# Edit configuration files with database information
```

### 4. Database Initialization
```bash
# Create databases
mysql -u root -p
CREATE DATABASE user_service_db;
CREATE DATABASE auth_service_db;

# Import database schema
mysql -u root -p user_service_db < scripts/sql/user_service_db.sql
mysql -u root -p auth_service_db < scripts/sql/auth_service_db.sql
```

### 5. API Documentation
```bash
# Generate OpenAPI documentation
make swagger

# View documentation at:
# http://localhost:9000/swagger/index.html (for swit-serve)
# http://localhost:9001/swagger/index.html (for swit-auth)
```

### 5. Build and Run
```bash
# Build all services (development mode)
make build

# Or quick build (skip quality checks)
make build-dev

# Run authentication service
./bin/swit-auth

# Run user service
./bin/swit-serve
```

## API Development

### Toolchain Setup
```bash
# Setup protobuf development environment
make proto-setup

# Setup swagger development environment
make swagger-setup
```

### Daily Development Commands

#### Protobuf Development
```bash
# Complete protobuf workflow (recommended)
make proto

# Quick development mode (skip dependencies)
make proto-dev

# Advanced proto operations
make proto-advanced OPERATION=format
make proto-advanced OPERATION=lint
make proto-advanced OPERATION=breaking
make proto-advanced OPERATION=clean
make proto-advanced OPERATION=docs
```

#### Swagger Documentation
```bash
# Generate swagger documentation (recommended)
make swagger

# Quick development mode (skip formatting)
make swagger-dev

# Advanced swagger operations
make swagger-advanced OPERATION=format
make swagger-advanced OPERATION=switserve
make swagger-advanced OPERATION=switauth
make swagger-advanced OPERATION=clean
```

### API Development Workflow

1. **Modify proto files**
   ```bash
   # Edit proto files
   vim api/proto/swit/v1/greeter/greeter.proto
   ```

2. **Generate code**
   ```bash
   make proto-generate
   ```

3. **Implement service**
   ```bash
   # Implement gRPC methods in corresponding service files
   vim internal/switserve/service/greeter.go
   ```

4. **Test and validate**
   ```bash
   make proto-lint
   make build
   make test
   ```

## Configuration

### Main Service Configuration (swit.yaml)
```yaml
server:
  port: 9000

database:
  host: 127.0.0.1
  port: 3306
  username: root
  password: root
  dbname: user_service_db

serviceDiscovery:
  address: "localhost:8500"
```

### Authentication Service Configuration (switauth.yaml)
```yaml
server:
  port: 9001
  grpcPort: 50051

database:
  host: 127.0.0.1
  port: 3306
  username: root
  password: root
  dbname: auth_service_db

serviceDiscovery:
  address: "localhost:8500"
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

### API Endpoints Reference

#### swit-serve (User Service)
- **HTTP**: `http://localhost:9000`
- **gRPC**: `http://localhost:10000`
- `/v1/users` - User operations (POST, GET, PUT, DELETE)
- `/v1/users/username/{username}` - Get user by username
- `/v1/users/email/{email}` - Get user by email
- `/v1/users` - List users with pagination

#### swit-auth (Auth Service)
- **HTTP**: `http://localhost:9001`
- **gRPC**: `http://localhost:50051`
- `/v1/auth/login` - Login
- `/v1/auth/logout` - Logout
- `/v1/auth/refresh` - Refresh token
- `/v1/auth/validate` - Validate token
- `/v1/auth/reset-password` - Reset password
- `/v1/auth/change-password` - Change password
- `/v1/auth/verify-email` - Verify email

#### switctl (CLI Tool)
- Command-line tool for system administration


## Related Documentation

- [Development Guide](DEVELOPMENT.md)
- [API Documentation](api/docs/README.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)

## Contributing

We welcome contributions to the Swit project! Please read our [Code of Conduct](CODE_OF_CONDUCT.md) before contributing to ensure a positive and inclusive environment for all community members.

## License

MIT License - See [LICENSE](LICENSE) file for details
