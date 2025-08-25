# switctl - Swit Framework CLI Tool

The `switctl` (Swit Control) command-line tool is a comprehensive development toolkit for the Swit microservice framework. It provides scaffolding, code generation, quality checking, and development utilities to accelerate microservice development.

## Overview

switctl offers a complete development workflow:

- **🚀 Service Scaffolding**: Generate complete microservices from templates
- **🔧 Code Generation**: Create APIs, middleware, and models automatically  
- **🛡️ Quality Assurance**: Built-in security scanning, testing, and code quality checks
- **📦 Template System**: Extensive library of production-ready templates
- **🔌 Plugin System**: Extensible architecture with custom plugin support
- **💻 Interactive UI**: User-friendly terminal interface with guided workflows

## Installation

### From Source (Recommended)

Build from the Swit framework repository:

```bash
# Clone the repository
git clone https://github.com/innovationmech/swit.git
cd swit

# Build switctl
make build

# The binary will be available at ./bin/switctl
./bin/switctl --help
```

### Add to PATH

For convenient usage, add switctl to your PATH:

```bash
# Copy to a directory in your PATH
sudo cp ./bin/switctl /usr/local/bin/

# Or create a symlink
sudo ln -s $(pwd)/bin/switctl /usr/local/bin/switctl

# Verify installation
switctl --help
```

## Quick Start

### 1. Initialize a New Project

Create a new microservice project:

```bash
# Interactive project initialization
switctl init my-service

# Quick start with defaults
switctl init my-service --quick
```

### 2. Generate a Service

Create a complete microservice from templates:

```bash
# Interactive service generation
switctl new service user-service

# With specific options
switctl new service user-service \
  --template=http-grpc \
  --database=postgresql \
  --auth=jwt
```

### 3. Generate Components

Add components to existing services:

```bash
# Generate API endpoints
switctl generate api user

# Generate middleware
switctl generate middleware auth

# Generate database models
switctl generate model User
```

### 4. Quality Checks

Run comprehensive quality checks:

```bash
# Run all checks
switctl check

# Specific checks
switctl check --security
switctl check --tests --coverage
switctl check --performance
```

## Core Commands

### Project Management

| Command | Description | Usage |
|---------|-------------|--------|
| `init` | Initialize new project | `switctl init <name> [options]` |
| `new` | Generate new services/components | `switctl new <type> <name> [options]` |
| `config` | Manage project configuration | `switctl config [get\|set] [key] [value]` |

### Code Generation

| Command | Description | Usage |
|---------|-------------|--------|
| `generate api` | Generate API endpoints | `switctl generate api <name> [options]` |
| `generate middleware` | Create middleware components | `switctl generate middleware <name>` |
| `generate model` | Generate data models | `switctl generate model <name>` |

### Quality Assurance

| Command | Description | Usage |
|---------|-------------|--------|
| `check` | Run quality checks | `switctl check [--type] [options]` |
| `test` | Run tests with coverage | `switctl test [options]` |
| `lint` | Code style and quality | `switctl lint [options]` |

### Development

| Command | Description | Usage |
|---------|-------------|--------|
| `dev` | Development utilities | `switctl dev <command>` |
| `deps` | Dependency management | `switctl deps [update\|check]` |
| `plugin` | Plugin management | `switctl plugin <command>` |

## Key Features

### 🏗️ Service Scaffolding

Generate complete, production-ready microservices:

```bash
switctl new service order-service \
  --template=full-featured \
  --database=postgresql \
  --auth=jwt \
  --monitoring=sentry \
  --cache=redis
```

**Generated Structure:**
- Complete project structure
- Docker configuration
- Makefile with all commands
- CI/CD pipeline files
- Health checks and monitoring
- Security configurations
- Comprehensive tests

### 🔧 Intelligent Code Generation

Create components with framework best practices:

```bash
# Generate RESTful API with validation
switctl generate api product \
  --methods=crud \
  --validation=true \
  --swagger=true

# Generate gRPC service
switctl generate grpc inventory \
  --streaming=true \
  --gateway=true

# Generate middleware with common patterns
switctl generate middleware auth \
  --type=jwt \
  --rbac=true
```

### 🛡️ Quality Assurance Suite

Comprehensive quality checking:

```bash
# Security scanning
switctl check --security
# ✓ Vulnerability scanning
# ✓ Dependency analysis  
# ✓ Code security patterns
# ✓ Configuration validation

# Performance testing
switctl check --performance
# ✓ Memory leak detection
# ✓ Benchmark testing
# ✓ Load testing setup
# ✓ Performance regression

# Code quality
switctl check --quality
# ✓ Code style and formatting
# ✓ Complexity analysis
# ✓ Test coverage
# ✓ Documentation coverage
```

### 📦 Template System

Extensive template library for different use cases:

#### Service Templates
- `http-only` - HTTP-only microservice
- `grpc-only` - gRPC-only service  
- `http-grpc` - Dual-protocol service
- `full-featured` - Complete service with all features
- `minimal` - Minimal service for learning

#### Authentication Templates
- `jwt` - JWT-based authentication
- `oauth2` - OAuth2 integration
- `api-key` - API key authentication
- `rbac` - Role-based access control

#### Database Templates
- `postgresql` - PostgreSQL with GORM
- `mysql` - MySQL integration
- `mongodb` - MongoDB with official driver
- `sqlite` - SQLite for development
- `redis` - Redis caching

#### Middleware Templates
- `cors` - Cross-origin resource sharing
- `rate-limit` - Rate limiting
- `logging` - Request logging
- `recovery` - Panic recovery
- `request-id` - Request ID tracking

### 🔌 Plugin System

Extensible architecture for custom functionality:

```bash
# List available plugins
switctl plugin list

# Install plugin
switctl plugin install <plugin-name>

# Create custom plugin
switctl plugin create my-plugin

# Manage plugins
switctl plugin enable <plugin-name>
switctl plugin disable <plugin-name>
```

### 💻 Interactive Experience

User-friendly terminal interface:

- **Guided Workflows**: Step-by-step project setup
- **Smart Defaults**: Sensible defaults based on project context
- **Validation**: Real-time validation of inputs
- **Progress Indicators**: Visual feedback during operations
- **Error Recovery**: Helpful error messages and recovery suggestions

## Configuration

switctl supports both global and project-specific configuration:

### Global Configuration

```bash
# Set global defaults
switctl config set template.default full-featured
switctl config set database.default postgresql
switctl config set auth.default jwt

# View configuration
switctl config list
```

### Project Configuration

Project-specific settings in `.switctl.yaml`:

```yaml
# .switctl.yaml
project:
  name: "my-service"
  type: "microservice"
  
templates:
  default: "http-grpc"
  
database:
  type: "postgresql"
  migrations: true
  
security:
  enabled: true
  scan_deps: true
  
testing:
  coverage_threshold: 80
  race_detection: true
```

## Integration with Framework

switctl is deeply integrated with the Swit framework:

- **Framework Awareness**: Generated code follows framework patterns
- **Best Practices**: Enforces framework best practices
- **Configuration**: Respects framework configuration structure
- **Dependencies**: Manages framework dependencies automatically
- **Updates**: Keeps generated code up-to-date with framework changes

## Development Workflow

Typical development workflow with switctl:

```bash
# 1. Initialize project
switctl init payment-service --template=http-grpc

# 2. Generate core components
cd payment-service
switctl generate api payment --methods=crud
switctl generate middleware auth --type=jwt
switctl generate model Payment

# 3. Add database integration
switctl generate database --type=postgresql --migrations=true

# 4. Quality checks
switctl check --all

# 5. Run tests
switctl test --coverage

# 6. Development utilities
switctl dev watch  # Auto-rebuild on changes
switctl dev docs   # Generate documentation
```

## Examples and Tutorials

Explore practical examples:

- [Getting Started Tutorial](/en/cli/getting-started) - Complete walkthrough
- [Service Templates Guide](/en/cli/templates) - Template system deep-dive  
- [Command Reference](/en/cli/commands) - Detailed command documentation
- [Plugin Development](/en/cli/plugins) - Creating custom plugins

## Support and Troubleshooting

### Common Issues

**Command not found:**
```bash
# Ensure switctl is in PATH
echo $PATH
which switctl

# Or use full path
./bin/switctl --help
```

**Permission denied:**
```bash
# Make sure binary is executable
chmod +x ./bin/switctl
```

**Template not found:**
```bash
# List available templates
switctl new service --list-templates

# Update templates
switctl template update
```

### Getting Help

```bash
# General help
switctl --help

# Command-specific help
switctl new --help
switctl generate --help

# Version information
switctl version
```

### Debug Mode

Enable debug output for troubleshooting:

```bash
# Run with debug output
switctl --debug new service my-service

# Verbose output
switctl --verbose check --all
```

## Next Steps

- **[Getting Started](/en/cli/getting-started)** - Detailed tutorial
- **[Commands Reference](/en/cli/commands)** - Complete command documentation
- **[Templates Guide](/en/cli/templates)** - Template system overview
- **[Plugin Development](/en/cli/plugins)** - Creating custom plugins

Ready to boost your microservice development? Start with the [getting started guide](/en/cli/getting-started)!
