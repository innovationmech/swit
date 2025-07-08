# Swit

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
2. **swit-auth** - Authentication service (port 8090)
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
**Port**: 9000  
**Protocol**: gRPC + HTTP  
**Implemented Methods**:
- `SayHello` - Simple greeting functionality ✅

**HTTP Endpoints**:
- `POST /v1/greeter/hello` - Send greeting request

### 2. Auth Service (HTTP REST)
**Port**: 8090  
**Protocol**: HTTP REST  
**Main Features**:
- User login/logout
- JWT Token management
- Token refresh and validation

### 3. User Service (HTTP REST)
**Port**: 9000  
**Protocol**: HTTP REST  
**Main Features**:
- User registration
- User information management
- User query and deletion

## Requirements

- Go 1.24+
- MySQL 8.0+
- Redis 6.0+
- Consul 1.12+
- Buf CLI 1.0+ (for API development)

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
CREATE DATABASE swit_db;
CREATE DATABASE swit_auth_db;

# Import database schema
mysql -u root -p swit_db < scripts/sql/user_service_db.sql
mysql -u root -p swit_auth_db < scripts/sql/auth_service_db.sql
```

### 5. Build and Run
```bash
# Build all services
make build

# Run authentication service
./bin/swit-auth

# Run user service
./bin/swit-serve
```

## API Development

### Toolchain Setup
```bash
# Install Buf CLI
make buf-install

# Setup development environment
make proto-setup
```

### Daily Development Commands
```bash
# Complete protobuf workflow
make proto

# Generate code only
make proto-generate

# Check proto files
make proto-lint

# Format proto files
make proto-format

# Check breaking changes
make proto-breaking

# Clean generated code
make proto-clean

# View OpenAPI documentation
make proto-docs
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
  host: "0.0.0.0"
  port: 9000
  grpc_port: 9001

database:
  host: "localhost"
  port: 3306
  name: "swit_db"
  username: "root"
  password: "password"

consul:
  address: "127.0.0.1:8500"
  datacenter: "dc1"
```

### Authentication Service Configuration (switauth.yaml)
```yaml
server:
  host: "0.0.0.0"
  port: 8090

database:
  host: "localhost"
  port: 3306
  name: "swit_auth_db"
  username: "root"
  password: "password"

jwt:
  secret: "your-secret-key"
  access_token_expiry: "15m"
  refresh_token_expiry: "7d"
```

## Docker Deployment

### Build Images
```bash
make docker-build
```

### Run Containers
```bash
# Run user service
docker run -d -p 9000:9000 -p 9001:9001 --name swit-serve swit-serve:latest

# Run authentication service
docker run -d -p 8090:8090 --name swit-auth swit-auth:latest
```

### Using Docker Compose
```bash
docker-compose up -d
```

## Testing

### Run Unit Tests
```bash
make test
```

### Run Integration Tests
```bash
make test-integration
```

### Test Coverage
```bash
make test-coverage
```

## Development Tools

### Code Formatting
```bash
make fmt
```

### Code Linting
```bash
make lint
```

### Dependency Check
```bash
make deps
```

## Upgrading from Previous Versions

Please refer to the [API Migration Guide](api/MIGRATION.md) for detailed upgrade steps.

Main changes:
- New API directory structure
- Use Buf toolchain to manage gRPC APIs
- Updated package names and import paths
- New HTTP/REST endpoint support

## Related Documentation

- [Development Guide](DEVELOPMENT.md)
- [API Documentation](api/docs/README.md)
- [API Migration Guide](api/MIGRATION.md)
- [Route Registry Guide](docs/route-registry-guide.md)
- [OpenAPI Integration](docs/openapi-integration.md)

## License

MIT License - See [LICENSE](LICENSE) file for details