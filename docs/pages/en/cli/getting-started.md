# Getting Started with switctl

This tutorial walks you through using switctl to create your first microservice and explores the key features of the CLI tool.

## Prerequisites

- Go 1.23.12 or later
- Git
- Basic understanding of microservices

## Installation

### 1. Build from Source

```bash
# Clone the Swit framework
git clone https://github.com/innovationmech/swit.git
cd swit

# Build switctl CLI
make build

# Verify installation
./bin/switctl --version
```

### 2. Add to PATH

```bash
# Add to your shell profile (~/.bashrc, ~/.zshrc)
export PATH="$PATH:/path/to/swit/bin"

# Or create a symlink
sudo ln -s /path/to/swit/bin/switctl /usr/local/bin/switctl
```

### 3. Verify Installation

```bash
switctl --help
# Should show the CLI help with all available commands
```

## Your First Service with switctl

### Step 1: Create a New Service

```bash
# Create a user management service
switctl new service user-service
```

The CLI will prompt you for options:

```text
✓ Service template: http-grpc
✓ Database type: postgresql
✓ Authentication: jwt
✓ Include Docker files: yes
✓ Include CI/CD files: yes
```

### Step 2: Explore Generated Structure

```bash
cd user-service
tree
```

Generated structure:
```text
user-service/
├── cmd/user-service/
│   └── main.go                 # Service entry point
├── internal/
│   ├── handler/
│   │   ├── http/              # HTTP handlers
│   │   └── grpc/              # gRPC handlers
│   ├── service/               # Business logic
│   └── config/                # Configuration
├── api/
│   └── proto/                 # Protocol buffers
├── configs/
│   ├── development.yaml       # Development config
│   └── production.yaml        # Production config
├── Dockerfile                 # Container configuration
├── Makefile                   # Build automation
├── docker-compose.yml         # Local development
├── .switctl.yaml             # CLI configuration
├── go.mod                     # Go modules
└── README.md                 # Service documentation
```

### Step 3: Build and Run

```bash
# Initialize Go modules
go mod tidy

# Build the service
make build

# Run in development mode
make run

# Or run directly
go run cmd/user-service/main.go
```

Your service is now running on:
- HTTP: http://localhost:8080
- gRPC: localhost:9080

### Step 4: Test the Service

```bash
# Health check
curl http://localhost:8080/health

# API status
curl http://localhost:8080/api/v1/status

# User operations (generated CRUD endpoints)
curl http://localhost:8080/api/v1/users
```

## Adding Components to Your Service

### Generate API Endpoints

```bash
# Generate CRUD API for products
switctl generate api product --methods=crud --validation=true

# Generate custom API
switctl generate api order \
  --methods=get,post,put \
  --middleware=auth,logging
```

### Generate Middleware

```bash
# Generate authentication middleware
switctl generate middleware auth --type=jwt

# Generate rate limiting middleware
switctl generate middleware rate-limit \
  --apply-to=http \
  --config=true
```

### Generate Database Models

```bash
# Generate User model with GORM
switctl generate model User \
  --database=gorm \
  --validation=true \
  --migration=true
```

This generates:
- `internal/model/user.go` - Model definition
- `internal/repository/user_repository.go` - Database operations
- `migrations/001_create_users.sql` - Database migration

## Quality Assurance with switctl

### Run All Checks

```bash
switctl check --all
```

This runs:
- ✅ Code formatting (gofmt)
- ✅ Code quality (golint)
- ✅ Security scanning
- ✅ Dependency vulnerabilities
- ✅ Test coverage
- ✅ Performance benchmarks

### Specific Checks

```bash
# Security only
switctl check --security
# Found 2 issues:
# - Use of weak cryptographic primitive (MD5)
# - Potential SQL injection in query

# Tests with coverage
switctl check --tests --coverage --threshold=80
# Coverage: 85% (above threshold ✓)

# Performance benchmarks
switctl check --performance
# All benchmarks within acceptable limits ✓
```

## Development Workflow

### 1. Watch Mode for Development

```bash
# Auto-rebuild on file changes
switctl dev watch
```

This monitors your code and automatically:
- Rebuilds the service
- Runs tests
- Restarts the development server

### 2. Generate Documentation

```bash
# Generate service documentation
switctl dev docs --format=html --output=./docs
```

### 3. Dependency Management

```bash
# Check for outdated dependencies
switctl deps check

# Update dependencies safely
switctl deps update --security
```

## Working with Templates

### List Available Templates

```bash
switctl template list
```

Available templates:
- `basic` - Simple HTTP service
- `http-grpc` - HTTP + gRPC service
- `full-featured` - All features enabled
- `grpc-only` - Pure gRPC service

### Use Specific Template

```bash
switctl new service payment-service \
  --template=full-featured \
  --database=postgresql \
  --auth=jwt \
  --cache=redis \
  --monitoring=sentry \
  --queue=rabbitmq
```

### Custom Template

Create your own template for team standards:

```bash
# Create template structure
mkdir -p ~/.switctl/templates/company-standard

# Generate template
switctl template create company-standard \
  --based-on=http-grpc \
  --add-feature=monitoring \
  --add-feature=tracing
```

## Configuration Management

### Project Configuration

switctl creates `.switctl.yaml` for project-specific settings:

```yaml
project:
  name: "user-service"
  type: "microservice"
  
templates:
  default: "http-grpc"
  
database:
  type: "postgresql"
  migrations: true
  
testing:
  coverage_threshold: 80
  race_detection: true
  
security:
  enabled: true
  scan_deps: true
```

### Global Configuration

Set global defaults:

```bash
# Set preferred template
switctl config set template.default full-featured

# Set database preference
switctl config set database.type postgresql

# Set coverage threshold
switctl config set testing.coverage_threshold 85
```

## Advanced Features

### Plugin System

Extend switctl with custom plugins:

```bash
# List available plugins
switctl plugin list

# Install plugin for OpenAPI generation
switctl plugin install openapi-gen

# Use plugin
switctl openapi generate --input=./api/proto --output=./docs/openapi.yaml
```

### CI/CD Integration

Generated services include CI/CD configuration:

```bash
# GitHub Actions (automatically generated)
.github/workflows/
├── ci.yml              # Build and test
├── security.yml        # Security scans
└── deploy.yml          # Deployment

# GitLab CI (with --cicd=gitlab)
.gitlab-ci.yml
```

### Multi-Service Projects

Manage multiple services in a monorepo:

```bash
# Initialize multi-service project
switctl init company-platform --type=monorepo

# Add services
switctl new service user-service --directory=services/
switctl new service order-service --directory=services/
switctl new service notification-service --directory=services/

# Generate shared components
switctl generate shared-lib common --type=utils
```

## Best Practices

### 1. Start Simple, Scale Up

```bash
# Begin with basic template
switctl new service my-service --template=basic

# Add features as needed
switctl generate api user --methods=crud
switctl generate middleware auth --type=jwt
```

### 2. Use Quality Checks Early

```bash
# Run checks during development
switctl check --quality --security

# Set up pre-commit hooks
echo "switctl check --all" > .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

### 3. Maintain Configuration

```bash
# Keep .switctl.yaml in version control
git add .switctl.yaml

# Document team standards
switctl config set team.standards "company-standard"
```

### 4. Leverage Templates

```bash
# Create team-specific templates
switctl template create team-api \
  --based-on=http-grpc \
  --add-middleware=auth,logging,cors \
  --add-monitoring=sentry

# Share templates
git commit -m "Add team API template"
```

## Troubleshooting

### Common Issues

**Command not found:**
```bash
# Check PATH
echo $PATH | grep switctl

# Use full path temporarily
/path/to/swit/bin/switctl --help
```

**Template errors:**
```bash
# List available templates
switctl template list

# Debug template generation
switctl --debug new service test --template=basic
```

**Generation failures:**
```bash
# Enable verbose output
switctl --verbose generate api user

# Check project configuration
switctl config get
```

### Debug Mode

Use debug mode to troubleshoot issues:

```bash
# Enable debug output
switctl --debug new service debug-test

# This shows:
# - Template resolution
# - File generation steps
# - Configuration validation
# - Error details
```

## Next Steps

Now that you've created your first service with switctl:

1. **[Commands Reference](/en/cli/commands)** - Learn all available commands
2. **[Template System](/en/cli/templates)** - Master the template system
3. **[Framework Guide](/en/guide/getting-started)** - Understand the underlying framework
4. **[Examples](/en/examples/)** - Explore complete examples

## Quick Reference

```bash
# Essential commands
switctl new service <name>              # Create service
switctl generate api <name>             # Generate API
switctl check --all                     # Quality checks
switctl dev watch                       # Development mode

# Configuration
switctl config set <key> <value>        # Set config
switctl config get                      # View config

# Templates
switctl template list                   # List templates
switctl template create <name>          # Create template

# Help
switctl --help                          # General help
switctl <command> --help               # Command help
```

Welcome to efficient microservice development with switctl! 🚀
