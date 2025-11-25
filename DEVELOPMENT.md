# Development Guide

This document describes the development workflow and quality standards for the Swit project.

## Quick Start

### Setup Development Environment

```bash
# Install development tools and Git hooks
make setup-dev
```

This will install:
- swag tool for generating Swagger documentation
- Pre-commit hooks for automatic quality checks

### Build and Test

```bash
# Run full quality checks and build
make all

# Or run individual steps
make tidy      # Run go mod tidy
make format    # Format code
make vet       # Run go vet
make quality   # Run all quality checks
make test      # Run tests
make build     # Build binaries
```

## Development Workflow

### 1. Code Quality

All code must pass the following quality checks before being committed:

- **Tidy**: Dependencies must be clean with `go mod tidy`
- **Format**: Code must be formatted with `gofmt`
- **Vet**: Code must pass `go vet` analysis
- **Tests**: All tests must pass

### 2. Pre-commit Hooks

The pre-commit hook automatically runs:
- `go mod tidy`
- Code formatting with `gofmt`
- `go vet` analysis  
- Tests for affected packages

### 3. Continuous Integration

Our CI pipeline runs on every push and pull request:

1. **Tidy Stage**: Dependencies cleanup with `go mod tidy`
2. **Quality Stage**: Format and vet checks
3. **Test Stage**: Unit tests with race detection and coverage
4. **Build Stage**: Build all binaries
5. **Documentation Stage**: Generate Swagger documentation

## Make Targets

| Target | Description |
|--------|-------------|
| `make all` | Run full build pipeline (tidy + copyright + build + swagger) |
| `make tidy` | Run go mod tidy |
| `make format` | Format code with gofmt |
| `make vet` | Run go vet |
| `make quality` | Run all quality checks (format + vet) |
| `make build` | Build all binaries |
| `make clean` | Delete output binaries |
| `make test` | Run unit tests |
| `make test-pkg` | Run tests for pkg packages only |
| `make test-internal` | Run tests for internal packages only |
| `make test-coverage` | Run tests with coverage report |
| `make test-race` | Run tests with race detection |
| `make image-serve` | Build Docker image for swit-serve |
| `make image-auth` | Build Docker image for swit-auth |
| `make image-all` | Build Docker images for all services |
| `make swagger` | Generate/update Swagger documentation for all services |
| `make swagger-switserve` | Generate Swagger documentation for switserve only |
| `make swagger-switauth` | Generate Swagger documentation for switauth only |
| `make ci` | Run full CI pipeline |
| `make setup-dev` | Setup development environment |

## Code Standards

### Go Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Write clear, self-documenting code
- Include unit tests for new functionality
- Use structured logging with zap

### Git Commit Messages

- Use conventional commit format: `type(scope): description`
- Examples:
  - `feat(user): add user authentication`
  - `fix(api): resolve rate limiting issue`
  - `docs(readme): update installation instructions`

### Testing

- Write unit tests for all new functionality
- Aim for good test coverage (check with `make test-coverage`)
- Use table-driven tests where appropriate
- Mock external dependencies

## Docker Images

The project supports building Docker images for both services:

```bash
# Build individual service images
make image-serve    # Build swit-serve image
make image-auth     # Build swit-auth image

# Build all images
make image-all
```

Images are tagged with the current git branch name.

## Swagger Documentation

The project generates Swagger documentation for both services:

```bash
# Generate all documentation
make swagger

# Generate for specific services
make swagger-switserve   # Generate for switserve
make swagger-switauth    # Generate for switauth
```

Documentation is generated in:
- `docs/generated/switserve/` - SwitServe API docs
- `docs/generated/switauth/` - SwitAuth API docs
- `docs/generated/` - Root for generated documentation

## Dependencies

The Swit framework uses the following key dependencies:

### Core Framework
- **Go 1.24+**: Required Go version
- **gin-gonic/gin**: HTTP web framework
- **grpc/grpc-go**: gRPC framework
- **uber-go/zap**: Structured logging
- **spf13/viper**: Configuration management
- **spf13/cobra**: CLI framework (for switctl)

### Security & Authentication
- **golang.org/x/oauth2**: OAuth2 client library for standard OAuth2 flows
- **coreos/go-oidc/v3**: OpenID Connect (OIDC) support and discovery
- **golang-jwt/jwt/v5**: JWT token validation and management
- **golang.org/x/crypto**: Cryptographic utilities
- **open-policy-agent/opa**: Policy engine for RBAC/ABAC authorization
- **hashicorp/vault/api**: Secret management (optional)

### Messaging & Streaming
- **nats-io/nats.go**: NATS messaging
- **streadway/amqp**: RabbitMQ client
- **segmentio/kafka-go**: Kafka client

### Observability
- **opentelemetry**: Distributed tracing
- **prometheus/client_golang**: Metrics collection
- **getsentry/sentry-go**: Error tracking

### Database & Storage
- **gorm.io/gorm**: ORM framework
- **redis/go-redis**: Redis client
- **lib/pq**: PostgreSQL driver

### Service Discovery
- **hashicorp/consul/api**: Consul integration

For a complete list of dependencies, see `go.mod`.

## Project Structure

```
├── cmd/                    # Application entry points
├── internal/              # Private application code
│   ├── switserve/        # Main server application
│   └── switauth/         # Authentication service
├── pkg/                   # Public library code
│   ├── server/           # Base server framework
│   ├── transport/        # HTTP/gRPC transport layer
│   ├── middleware/       # HTTP/gRPC middleware
│   ├── security/         # Security components
│   │   ├── oauth2/       # OAuth2/OIDC client
│   │   ├── opa/          # OPA policy engine
│   │   ├── jwt/          # JWT validation
│   │   ├── tls/          # TLS configuration
│   │   ├── audit/        # Audit logging
│   │   └── secrets/      # Secret management
│   ├── saga/             # Distributed transactions
│   └── ...
├── api/                   # API definitions (protobuf, OpenAPI)
├── scripts/              # Build and utility scripts
├── build/                # Build configurations (Docker, etc.)
└── _output/              # Build artifacts (generated)
```

## Troubleshooting

### Quality Issues

If you encounter quality issues:

1. Run `make tidy` to clean up dependencies
2. Run `make format` to fix formatting
3. Run `make vet` to check for potential issues
4. Fix issues manually or use IDE suggestions
5. Some issues may require code refactoring

### Pre-commit Hook Issues

If the pre-commit hook is causing problems:

```bash
# Temporarily skip hooks for urgent fixes
git commit --no-verify -m "urgent fix"

# Or remove and reinstall hooks
rm .git/hooks/pre-commit
make install-hooks
```

### CI Pipeline Failures

1. Check the specific stage that failed
2. Run the same commands locally:
   ```bash
   make ci  # Run full CI pipeline locally
   ```
3. Fix issues and push again

## Security Development

### Security Testing

```bash
# Run security scans
make security              # Full security scan (gosec + trivy + govulncheck)

# Individual scans
gosec ./...                # Static code analysis
govulncheck ./...          # Go vulnerability check
trivy fs .                 # File system vulnerability scan
```

### Security Best Practices

When developing security-related features:

1. **Never commit secrets** - Use environment variables or secret management
2. **Validate all inputs** - Use struct tags and validators
3. **Use parameterized queries** - Prevent SQL injection
4. **Enable TLS in production** - Minimum TLS 1.2
5. **Implement proper logging** - Audit security events without sensitive data
6. **Follow OWASP guidelines** - Reference OWASP Top 10

### Security Configuration

Example security configuration in `swit.yaml`:

```yaml
security:
  oauth2:
    enabled: true
    provider: keycloak
    client_id: my-service
    client_secret: ${OAUTH2_CLIENT_SECRET}
    issuer_url: https://auth.example.com/realms/production
    
  opa:
    enabled: true
    mode: embedded
    policy_dir: ./policies
    
  tls:
    enabled: true
    cert_file: /etc/ssl/certs/server.crt
    key_file: /etc/ssl/private/server.key
    min_version: "1.2"
```

### Security Documentation

- [Security Best Practices](docs/security-best-practices.md)
- [OAuth2 Integration Guide](docs/oauth2-integration-guide.md)
- [OPA Policy Guide](docs/opa-policy-guide.md)
- [Security Checklist](docs/security-checklist.md)

## Getting Help

- Run `make help` to see all available targets
- Check existing code for examples and patterns
- Review CI logs for detailed error messages