# Template System Guide

The switctl template system provides a comprehensive library of production-ready templates for rapidly scaffolding microservices and components. This guide covers all available templates, customization options, and best practices.

## Overview

The template system offers:

- **🏗️ Service Templates**: Complete microservice scaffolding
- **🔐 Authentication Templates**: Security and authentication patterns  
- **💾 Database Templates**: Database integration patterns
- **🛡️ Middleware Templates**: Common middleware components
- **🔧 Component Templates**: Individual component generation
- **🎨 Custom Templates**: Create and share your own templates

## Template Categories

### Service Templates

Complete microservice templates with full project structure.

#### `basic`
Minimal HTTP service for learning and simple use cases.

**Features:**
- HTTP server with Gin framework
- Basic health check endpoint  
- Graceful shutdown
- Docker configuration
- Simple Makefile

**Generated Structure:**
```text
my-service/
├── cmd/my-service/
│   └── main.go
├── internal/
│   ├── handler/
│   ├── service/
│   └── config/
├── Dockerfile
├── Makefile
├── go.mod
└── README.md
```

**Usage:**
```bash
switctl new service my-service --template=basic
```

#### `http-grpc`
Dual-protocol service supporting both HTTP and gRPC.

**Features:**
- HTTP and gRPC servers
- Protocol Buffer definitions
- grpc-gateway for HTTP/gRPC bridge
- Health checks for both protocols
- Docker multi-stage build
- Comprehensive Makefile

**Generated Structure:**
```text
my-service/
├── api/
│   └── proto/
│       └── service.proto
├── cmd/my-service/
├── internal/
│   ├── handler/
│   │   ├── http/
│   │   └── grpc/
│   ├── service/
│   └── config/
├── pkg/
│   └── api/
├── Dockerfile
├── docker-compose.yml
└── buf.yaml
```

**Usage:**
```bash
switctl new service my-service --template=http-grpc
```

#### `full-featured`
Complete microservice with all framework features enabled.

**Features:**
- HTTP + gRPC dual transport
- Database integration (configurable)
- Authentication and authorization
- Caching layer
- Message queue integration
- Monitoring and observability
- CI/CD pipeline files
- Comprehensive testing setup
- Documentation generation

**Additional Components:**
- Middleware stack (auth, logging, recovery, CORS)
- Health checks and metrics endpoints
- Database migrations
- Configuration management
- Development tools

**Usage:**
```bash
switctl new service my-service \
  --template=full-featured \
  --database=postgresql \
  --auth=jwt \
  --cache=redis \
  --monitoring=sentry
```

#### `grpc-only`
Pure gRPC service for high-performance inter-service communication.

**Features:**
- gRPC server with streaming support
- Protocol Buffer definitions  
- Service reflection
- Health check service
- Performance optimizations
- Docker configuration

**Usage:**
```bash
switctl new service my-service --template=grpc-only
```

#### `minimal`
Absolute minimum service for educational purposes.

**Features:**
- Single main.go file
- Basic HTTP handler
- No external dependencies
- Ideal for learning

**Usage:**
```bash
switctl new service my-service --template=minimal
```

### Authentication Templates

Security and authentication patterns for services.

#### `jwt`
JSON Web Token authentication with role-based access control.

**Features:**
- JWT token generation and validation
- Refresh token support
- Role-based access control (RBAC)
- Middleware for HTTP and gRPC
- Token blacklisting
- Configurable token expiry

**Generated Components:**
```go
// JWT service
type JWTService interface {
    GenerateToken(user *User) (string, error)
    ValidateToken(token string) (*Claims, error)
    RefreshToken(token string) (string, error)
}

// Auth middleware
func JWTAuthMiddleware() gin.HandlerFunc
func GRPCJWTInterceptor() grpc.UnaryServerInterceptor
```

**Usage:**
```bash
switctl new service my-service --auth=jwt
# Or add to existing service
switctl generate auth jwt
```

#### `oauth2`
OAuth2 integration with popular providers.

**Features:**
- OAuth2 client configuration
- Provider integrations (Google, GitHub, etc.)
- Token exchange and validation
- User profile mapping
- Session management

**Supported Providers:**
- Google OAuth2
- GitHub OAuth2
- Facebook OAuth2  
- Custom OAuth2 providers

**Usage:**
```bash
switctl new service my-service --auth=oauth2
```

#### `api-key`
API key-based authentication for service-to-service communication.

**Features:**
- API key generation and management
- Key rotation support
- Rate limiting per key
- Usage analytics
- Admin endpoints for key management

**Usage:**
```bash
switctl new service my-service --auth=api-key
```

#### `rbac`
Comprehensive role-based access control system.

**Features:**
- Role and permission management
- Hierarchical roles
- Resource-based permissions
- Policy evaluation engine
- Admin interface

**Usage:**
```bash
switctl generate auth rbac
```

### Database Templates

Database integration patterns for different database systems.

#### `postgresql`
PostgreSQL integration with GORM.

**Features:**
- GORM v2 configuration
- Connection pooling
- Database migrations
- Health checks
- Transaction management
- Performance monitoring

**Generated Components:**
```go
// Database configuration
type DatabaseConfig struct {
    Host     string
    Port     int  
    User     string
    Password string
    DBName   string
    SSLMode  string
}

// Migration system
type Migration interface {
    Up() error
    Down() error
}
```

**Usage:**
```bash
switctl new service my-service --database=postgresql
```

#### `mongodb`
MongoDB integration with official Go driver.

**Features:**
- MongoDB client configuration
- Connection management
- Index management
- Aggregation pipeline helpers
- Change stream support

**Usage:**
```bash
switctl new service my-service --database=mongodb
```

#### `mysql`
MySQL integration with GORM.

**Features:**
- MySQL-specific GORM configuration
- Connection pooling optimized for MySQL
- Migration support
- Performance monitoring

**Usage:**
```bash
switctl new service my-service --database=mysql
```

#### `sqlite`
SQLite integration for development and testing.

**Features:**
- SQLite GORM configuration
- In-memory database support
- Testing utilities
- Development-friendly setup

**Usage:**
```bash
switctl new service my-service --database=sqlite
```

#### `redis`
Redis caching and session storage.

**Features:**
- Redis client configuration
- Caching abstractions
- Session storage
- Pub/Sub support
- Cluster configuration

**Usage:**
```bash
switctl new service my-service --cache=redis
```

### Middleware Templates

Common middleware components for HTTP and gRPC services.

#### `cors`
Cross-Origin Resource Sharing middleware.

**Features:**
- Configurable CORS policies
- Preflight request handling
- Origin validation
- Credentials support

**Generated Code:**
```go
func CORSMiddleware(config CORSConfig) gin.HandlerFunc {
    return cors.New(cors.Config{
        AllowOrigins:     config.AllowOrigins,
        AllowMethods:     config.AllowMethods,
        AllowHeaders:     config.AllowHeaders,
        AllowCredentials: config.AllowCredentials,
    })
}
```

**Usage:**
```bash
switctl generate middleware cors
```

#### `rate-limit`
Rate limiting middleware for API protection.

**Features:**
- Token bucket algorithm
- IP-based and user-based limiting
- Redis backend support
- Configurable limits and windows
- Custom rate limit headers

**Usage:**
```bash
switctl generate middleware rate-limit
```

#### `logging`
Structured logging middleware.

**Features:**
- Request/response logging
- Configurable log levels
- Performance metrics
- Error tracking integration
- Custom field extraction

**Usage:**
```bash
switctl generate middleware logging
```

#### `recovery`
Panic recovery middleware.

**Features:**
- Graceful panic handling
- Stack trace capture
- Error reporting integration
- Custom recovery handlers

**Usage:**
```bash
switctl generate middleware recovery
```

#### `request-id`
Request ID tracking middleware.

**Features:**
- Unique request ID generation
- Header propagation
- Distributed tracing support
- Custom ID generators

**Usage:**
```bash
switctl generate middleware request-id
```

## Template Customization

### Custom Template Directory

Use your own template directory:

```bash
# Set custom template directory
export SWITCTL_TEMPLATE_DIR=/path/to/custom/templates

# Or use command flag
switctl new service my-service --template-dir=/path/to/templates
```

### Template Variables

Templates support variable substitution:

**Common Variables:**
- <code v-pre>{{.ServiceName}}</code> - Service name
- <code v-pre>{{.ModulePath}}</code> - Go module path
- <code v-pre>{{.PackageName}}</code> - Go package name
- <code v-pre>{{.DatabaseType}}</code> - Database type
- <code v-pre>{{.AuthType}}</code> - Authentication type
- <code v-pre>{{.HasDatabase}}</code> - Boolean for database presence
- <code v-pre>{{.HasAuth}}</code> - Boolean for authentication
- <code v-pre>{{.Year}}</code> - Current year
- <code v-pre>{{.Date}}</code> - Current date

**Example Template:**
```go
// { {.ServiceName}} service implementation
package { {.PackageName}}

import (
    { {if .HasDatabase}}"database/sql"{ {end}}
    { {if .HasAuth}}"github.com/golang-jwt/jwt/v4"{ {end}}
)

type { {.ServiceName}}Service struct {
    { {if .HasDatabase}}db *sql.DB{ {end}}
    { {if .HasAuth}}jwtSecret []byte{ {end}}
}
```

### Creating Custom Templates

#### 1. Template Structure

```text
my-custom-template/
├── template.yaml          # Template metadata
├── files/                 # Template files
│   ├── cmd/
│   │   └── { {.ServiceName}}/
│   │       └── main.go.tmpl
│   ├── internal/
│   │   └── service/
│   │       └── service.go.tmpl
│   └── README.md.tmpl
└── hooks/                 # Optional hooks
    ├── pre-generate.sh
    └── post-generate.sh
```

#### 2. Template Metadata

**template.yaml:**
```yaml
name: "my-custom-template"
description: "Custom service template"
version: "1.0.0"
author: "Your Name"
tags:
  - "http"
  - "database"
  
variables:
  - name: "ServiceName"
    description: "Name of the service"
    type: "string"
    required: true
  - name: "DatabaseType"  
    description: "Database type"
    type: "string"
    default: "postgresql"
    options: ["postgresql", "mysql", "mongodb"]

dependencies:
  - "github.com/gin-gonic/gin"
  - "gorm.io/gorm"

features:
  - "http-server"
  - "database"
  - "health-checks"
```

#### 3. Template Files

Use Go template syntax in `.tmpl` files:

**main.go.tmpl:**
```go
package main

import (
    "context"
    "log"
    
    "{ {.ModulePath}}/internal/config"
    "{ {.ModulePath}}/internal/service"
    { {if eq .DatabaseType "postgresql"}}
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    { {end}}
)

func main() {
    cfg := config.Load()
    
    { {if .HasDatabase}}
    db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    { {end}}
    
    svc := service.New{ {.ServiceName}}Service({ {if .HasDatabase}}db{ {end}})
    
    // Start server
    log.Printf("Starting { {.ServiceName}} service...")
    if err := svc.Start(context.Background()); err != nil {
        log.Fatal("Failed to start service:", err)
    }
}
```

#### 4. Template Hooks

**pre-generate.sh:**
```bash
#!/bin/bash
# Pre-generation hook
echo "Preparing to generate { {.ServiceName}}..."

# Validate requirements
if ! command -v go &> /dev/null; then
    echo "Go is required but not installed"
    exit 1
fi
```

**post-generate.sh:**
```bash
#!/bin/bash
# Post-generation hook
echo "Generated { {.ServiceName}} successfully"

# Initialize go module
cd { {.ServiceName}}
go mod init { {.ModulePath}}
go mod tidy

# Generate protobuf if needed
if [ -f "api/proto/service.proto" ]; then
    buf generate
fi

echo "Service { {.ServiceName}} is ready!"
```

### Template Development Workflow

#### 1. Create Template

```bash
# Create template directory
mkdir -p ~/.switctl/templates/my-template

# Create template files
switctl template init my-template
```

#### 2. Test Template

```bash
# Test template generation
switctl new service test-service --template=my-template --dry-run

# Generate and test
switctl new service test-service --template=my-template
cd test-service && make test
```

#### 3. Share Template

```bash
# Package template
switctl template package my-template

# Publish to registry (if available)
switctl template publish my-template
```

## Best Practices

### Template Selection

1. **Start Simple**: Begin with `basic` or `http-grpc` templates
2. **Match Requirements**: Choose templates that match your exact needs
3. **Consider Team Standards**: Use templates that align with team conventions
4. **Evaluate Dependencies**: Check generated dependencies match your preferences

### Customization Guidelines

1. **Preserve Structure**: Keep generated project structure intact
2. **Update Documentation**: Modify README and docs for your specific service
3. **Configure Defaults**: Set up appropriate configuration defaults
4. **Add Custom Logic**: Implement your business logic in designated areas

### Template Development

1. **Follow Conventions**: Use established Go and framework conventions
2. **Include Tests**: Generate comprehensive test suites
3. **Document Variables**: Clearly document all template variables
4. **Test Thoroughly**: Test templates with different configurations
5. **Version Templates**: Use semantic versioning for template releases

### Performance Considerations

1. **Minimal Dependencies**: Only include necessary dependencies
2. **Efficient Patterns**: Use performant coding patterns
3. **Resource Management**: Properly manage database connections and other resources
4. **Monitoring Ready**: Include performance monitoring hooks

## Advanced Features

### Conditional Generation

Generate different code based on configuration:

```go
{ {if .HasAuth}}
// Authentication middleware
func AuthMiddleware() gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        // Auth logic here
    })
}
{ {end}}

{ {if eq .DatabaseType "postgresql"}}
import "gorm.io/driver/postgres"
{ {else if eq .DatabaseType "mysql"}}
import "gorm.io/driver/mysql"
{ {end}}
```

### Multi-File Templates

Generate multiple files with dependencies:

```yaml
# template.yaml
files:
  - source: "cmd/main.go.tmpl"
    target: "cmd/{ {.ServiceName}}/main.go"
    
  - source: "internal/service.go.tmpl"  
    target: "internal/service/{ {.ServiceName|lower}}.go"
    conditions:
      - HasService: true
      
  - source: "docker/Dockerfile.tmpl"
    target: "Dockerfile"
    conditions:
      - IncludeDocker: true
```

### Template Inheritance

Extend existing templates:

```yaml
# template.yaml
name: "my-extended-template"
extends: "http-grpc"
description: "Extended HTTP-gRPC template with custom features"

additional_files:
  - "custom/handler.go.tmpl"
  - "custom/middleware.go.tmpl"
  
overrides:
  - "cmd/main.go.tmpl"  # Override parent template file
```

## Troubleshooting Templates

### Common Issues

#### Template Not Found
```bash
# List available templates
switctl template list

# Check template directory
ls ~/.switctl/templates/
```

#### Generation Errors
```bash
# Use dry-run to debug
switctl new service test --template=problematic --dry-run

# Enable debug mode
switctl --debug new service test --template=problematic
```

#### Variable Substitution Issues
```bash
# Verify template variables
switctl template info my-template

# Test with explicit variables
switctl new service test --template=my-template \
  --var="CustomVar=value"
```

### Debug Mode

Enable detailed template processing information:

```bash
export SWITCTL_DEBUG=true
switctl new service my-service --template=debug-template
```

## Related Topics

- [CLI Commands Reference](/en/cli/commands) - Complete command documentation
- [Getting Started Guide](/en/cli/getting-started) - Step-by-step tutorial  
- [Plugin Development](/en/cli/plugins) - Creating custom plugins
- [Configuration Guide](/en/guide/configuration) - Framework configuration
