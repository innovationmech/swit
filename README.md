# Swit

Swit is a microservice-based backend system built with Go, featuring modular design for user management, authentication, and service discovery. The project uses the Gin framework for HTTP requests, GORM for data persistence, and supports gRPC protocol for inter-service communication.

## Core Features

- **Microservice Architecture**: Modular design supporting independent deployment and scaling
- **Authentication System**: Complete JWT-based authentication with token refresh support
- **User Management**: Full CRUD operations and permission management for users
- **Service Discovery**: Consul integration for service registration and discovery
- **Database Support**: MySQL for data persistence
- **Protocol Support**: Both HTTP REST API and gRPC protocol support
- **Health Checks**: Built-in health check endpoints for service monitoring
- **Docker Support**: Containerized deployment solutions
- **OpenAPI Documentation**: Integrated Swagger UI with interactive API documentation

## System Architecture

The project consists of the following main components:

1. **swit-serve** - Main user service (port 9000)
2. **swit-auth** - Authentication service (port 9001)
3. **switctl** - Command-line control tool

## Quick Start

### Prerequisites

- Go 1.22 or higher
- MySQL 5.7+ or 8.0+
- Consul (optional, for service discovery)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/innovationmech/swit.git
   cd swit
   ```

2. Initialize databases:
   ```bash
   # Execute database scripts
   mysql -u root -p < scripts/sql/user_service_db.sql
   mysql -u root -p < scripts/sql/auth_service_db.sql
   ```

3. Configure services:
   - Edit `swit.yaml` to configure main service database and port
   - Edit `switauth.yaml` to configure authentication service database and port

### Building

Build all services:
```bash
make build
```

Or build individually:
```bash
make build-serve    # Build main service
make build-auth     # Build auth service
make build-ctl      # Build control tool
```

Binaries will be generated in the `_output/` directory.

### Running Services

1. Start authentication service:
   ```bash
   ./_output/swit-auth/swit-auth start
   ```

2. Start main service:
   ```bash
   ./_output/swit-serve/swit-serve serve
   ```

3. Check service status with control tool:
   ```bash
   ./_output/switctl/switctl health
   ```

## API Endpoints

### OpenAPI/Swagger Documentation

The project integrates Swagger UI for interactive API documentation. After starting the service, you can access it at:

```
http://localhost:9000/swagger/index.html
```

### Authentication Service API (Port 9001)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/login` | User login |
| POST | `/auth/logout` | User logout |
| POST | `/auth/refresh` | Refresh access token |
| GET | `/auth/validate` | Validate token |
| GET | `/health` | Health check |

### User Service API (Port 9000)

| Method | Path | Description | Auth Required |
|--------|------|-------------|---------------|
| POST | `/api/v1/users/create` | Create user | ✓ |
| GET | `/api/v1/users/username/:username` | Get user by username | ✓ |
| GET | `/api/v1/users/email/:email` | Get user by email | ✓ |
| DELETE | `/api/v1/users/:id` | Delete user | ✓ |
| POST | `/internal/validate-user` | Validate user credentials (internal) | ✗ |
| GET | `/health` | Health check | ✗ |
| POST | `/stop` | Stop service | ✗ |

### gRPC Services

The project also provides gRPC interfaces defined in `api/proto/greeter.proto`:
- `SayHello`: Simple greeting service

## Configuration

### Main Service Configuration (swit.yaml)
```yaml
database:
  host: 127.0.0.1
  port: 3306
  username: root
  password: root
  dbname: user_service_db
server:
  port: 9000
serviceDiscovery:
  address: "localhost:8500"
```

### Authentication Service Configuration (switauth.yaml)
```yaml
database:
  host: 127.0.0.1
  port: 3306
  username: root
  password: root
  dbname: auth_service_db
server:
  port: 9001
serviceDiscovery:
  address: "localhost:8500"
```

## Development Guide

### Project Structure

```
swit/
├── cmd/                    # Application entry points
│   ├── swit-serve/        # Main service
│   ├── swit-auth/         # Authentication service
│   └── switctl/           # Control tool
├── internal/              # Internal packages
│   ├── switserve/         # Main service internal logic
│   ├── switauth/          # Auth service internal logic
│   └── switctl/           # Control tool internal logic
├── api/                   # API definitions
│   └── proto/             # gRPC protocol definitions
├── pkg/                   # Common packages
├── scripts/               # Script files
│   └── sql/               # Database scripts
└── docs/                  # Documentation
```

### Available Make Commands

- `make tidy`: Tidy Go module dependencies
- `make build`: Build all binaries
- `make clean`: Clean output directory
- `make test`: Run tests
- `make test-coverage`: Run tests with coverage report
- `make test-race`: Run tests with race detection
- `make ci`: Run full CI pipeline (tidy, copyright, quality, test)
- `make image-serve`: Build Docker image
- `make swagger`: Generate/update OpenAPI documentation
- `make swagger-install`: Install Swagger documentation tools
- `make swagger-fmt`: Format Swagger annotations

### Docker Support

Build Docker image:
```bash
make image-serve
```

Run container:
```bash
docker run -d -p 9000:9000 -v ./swit.yaml:/root/swit.yaml swit-serve:master
```

## Database Schema

### Users Table (user_service_db.users)
- `id`: UUID primary key
- `username`: Username (unique)
- `email`: Email address (unique)
- `password_hash`: Encrypted password
- `role`: User role
- `is_active`: Whether user is active
- `created_at`/`updated_at`: Timestamps

### Tokens Table (auth_service_db.tokens)
- `id`: UUID primary key
- `user_id`: Associated user ID
- `access_token`: Access token
- `refresh_token`: Refresh token
- `access_expires_at`: Access token expiration time
- `refresh_expires_at`: Refresh token expiration time
- `is_valid`: Whether token is valid

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

[中文文档](README-CN.md)