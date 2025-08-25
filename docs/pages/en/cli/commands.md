# CLI Commands Reference

Complete reference for all `switctl` commands, options, and usage patterns.

## Global Options

These options are available for all commands:

```bash
switctl [global options] <command> [command options] [arguments]
```

| Option | Description | Example |
|--------|-------------|---------|
| `--debug` | Enable debug output | `switctl --debug new service my-service` |
| `--verbose, -v` | Verbose output | `switctl -v check --all` |
| `--config, -c` | Specify config file | `switctl -c custom.yaml new service` |
| `--help, -h` | Show help | `switctl --help` |
| `--version` | Show version | `switctl --version` |

## init - Initialize Project

Initialize a new Swit framework project.

### Usage

```bash
switctl init <project-name> [options]
```

### Options

| Option | Description | Default | Example |
|--------|-------------|---------|---------|
| `--template, -t` | Project template | `basic` | `--template=full-featured` |
| `--module-path, -m` | Go module path | Auto-generated | `--module-path=github.com/user/project` |
| `--directory, -d` | Target directory | Current dir | `--directory=./projects` |
| `--quick, -q` | Skip interactive prompts | false | `--quick` |
| `--force, -f` | Overwrite existing files | false | `--force` |
| `--dry-run` | Show what would be created | false | `--dry-run` |

### Examples

```bash
# Interactive initialization
switctl init my-service

# Quick initialization with defaults
switctl init my-service --quick

# Full-featured project
switctl init my-service \
  --template=full-featured \
  --module-path=github.com/mycompany/my-service

# Custom directory
switctl init my-service --directory=./projects --force
```

### Templates Available

- `basic` - Basic HTTP service
- `grpc` - gRPC-only service  
- `http-grpc` - HTTP + gRPC service
- `full-featured` - All framework features
- `minimal` - Learning/example service

## new - Generate New Components

Generate new services, APIs, middleware, and other components.

### new service

Generate a complete microservice.

```bash
switctl new service <service-name> [options]
```

#### Options

| Option | Description | Values | Default |
|--------|-------------|---------|----------|
| `--template, -t` | Service template | See templates | `http-grpc` |
| `--database` | Database type | `postgresql`, `mysql`, `mongodb`, `sqlite`, `none` | `none` |
| `--auth` | Authentication type | `jwt`, `oauth2`, `api-key`, `none` | `none` |
| `--cache` | Caching solution | `redis`, `memcached`, `none` | `none` |
| `--monitoring` | Monitoring solution | `sentry`, `prometheus`, `none` | `none` |
| `--queue` | Message queue | `rabbitmq`, `kafka`, `redis`, `none` | `none` |
| `--docker` | Include Docker files | true/false | `true` |
| `--makefile` | Include Makefile | true/false | `true` |
| `--cicd` | Include CI/CD files | `github`, `gitlab`, `none` | `github` |
| `--quick, -q` | Skip interactive prompts | - | - |
| `--force, -f` | Overwrite existing files | - | - |

#### Examples

```bash
# Interactive service generation
switctl new service user-service

# Complete service with all features
switctl new service order-service \
  --template=full-featured \
  --database=postgresql \
  --auth=jwt \
  --cache=redis \
  --monitoring=sentry \
  --queue=rabbitmq

# Minimal gRPC service
switctl new service inventory-service \
  --template=grpc \
  --database=mongodb \
  --quick
```

### new api

Generate API endpoints.

```bash
switctl new api <api-name> [options]
```

#### Options

| Option | Description | Values | Default |
|--------|-------------|---------|----------|
| `--methods` | HTTP methods to generate | `get`, `post`, `put`, `delete`, `crud` | `crud` |
| `--validation` | Include validation | true/false | `true` |
| `--swagger` | Generate Swagger docs | true/false | `true` |
| `--middleware` | Apply middleware | Comma-separated list | `auth,logging` |
| `--model` | Associated model name | Model name | Auto-generated |

#### Examples

```bash
# CRUD API with validation
switctl new api user --methods=crud --validation=true

# Custom endpoints
switctl new api product \
  --methods=get,post,delete \
  --middleware=auth,rate-limit \
  --model=Product
```

### new middleware

Generate middleware components.

```bash
switctl new middleware <middleware-name> [options]
```

#### Options

| Option | Description | Values | Default |
|--------|-------------|---------|----------|
| `--type` | Middleware type | `auth`, `cors`, `rate-limit`, `logging`, `custom` | `custom` |
| `--apply-to` | Apply to transports | `http`, `grpc`, `both` | `both` |
| `--config` | Include configuration | true/false | `true` |

#### Examples

```bash
# Authentication middleware
switctl new middleware auth --type=auth --apply-to=both

# Rate limiting middleware
switctl new middleware rate-limit \
  --type=rate-limit \
  --apply-to=http \
  --config=true
```

## generate - Code Generation

Generate specific code components.

### generate model

Generate data models with database integration.

```bash
switctl generate model <model-name> [options]
```

#### Options

| Option | Description | Values | Default |
|--------|-------------|---------|----------|
| `--database` | Target database | `gorm`, `mongo`, `custom` | `gorm` |
| `--validation` | Include validation tags | true/false | `true` |
| `--json` | Include JSON tags | true/false | `true` |
| `--migration` | Generate migration | true/false | `true` |
| `--test` | Generate test file | true/false | `true` |

#### Examples

```bash
# User model with GORM
switctl generate model User \
  --database=gorm \
  --validation=true \
  --migration=true

# MongoDB model
switctl generate model Product --database=mongo --test=true
```

### generate proto

Generate Protocol Buffer definitions and code.

```bash
switctl generate proto <service-name> [options]
```

#### Options

| Option | Description | Values | Default |
|--------|-------------|---------|----------|
| `--methods` | Service methods | Comma-separated list | `Create,Get,Update,Delete` |
| `--streaming` | Include streaming methods | true/false | `false` |
| `--gateway` | Generate HTTP gateway | true/false | `true` |
| `--validate` | Include validation | true/false | `true` |

## check - Quality Assurance

Run comprehensive quality checks on your codebase.

### Usage

```bash
switctl check [options]
```

### Options

| Option | Description | Default |
|--------|-------------|---------|
| `--all` | Run all checks | false |
| `--security` | Security scanning | false |
| `--quality` | Code quality checks | false |
| `--performance` | Performance testing | false |
| `--tests` | Run tests | false |
| `--coverage` | Check test coverage | false |
| `--deps` | Dependency analysis | false |
| `--threshold <n>` | Coverage threshold | 80 |
| `--format` | Output format | `text`, `json`, `junit` | `text` |
| `--output, -o` | Output file | Stdout |

### Check Types

#### Security Check

```bash
switctl check --security
```

**What it checks:**
- Known vulnerabilities in dependencies
- Insecure code patterns
- Configuration security issues
- Secret detection
- License compliance

#### Quality Check

```bash
switctl check --quality
```

**What it checks:**
- Code formatting (gofmt)
- Code complexity
- Dead code detection
- Import organization
- Documentation coverage

#### Performance Check

```bash
switctl check --performance
```

**What it checks:**
- Memory leaks
- Goroutine leaks
- Benchmark regression
- Resource usage patterns

#### Test Check

```bash
switctl check --tests --coverage --threshold=90
```

**What it checks:**
- Test execution
- Coverage percentage
- Race conditions
- Test organization

### Examples

```bash
# Run all checks
switctl check --all

# Security and quality only  
switctl check --security --quality

# Coverage with custom threshold
switctl check --tests --coverage --threshold=85

# Output to file
switctl check --all --format=json --output=quality-report.json
```

## config - Configuration Management

Manage project and global configuration.

### Usage

```bash
switctl config <subcommand> [options]
```

### Subcommands

#### get

Get configuration values.

```bash
switctl config get [key]
```

Examples:
```bash
# Get all configuration
switctl config get

# Get specific value
switctl config get template.default
switctl config get database.type
```

#### set

Set configuration values.

```bash
switctl config set <key> <value>
```

Examples:
```bash
# Set default template
switctl config set template.default full-featured

# Set database preference
switctl config set database.type postgresql

# Set coverage threshold
switctl config set testing.coverage_threshold 85
```

#### list

List all configuration keys and values.

```bash
switctl config list
```

#### reset

Reset configuration to defaults.

```bash
switctl config reset [key]
```

Examples:
```bash
# Reset all configuration
switctl config reset

# Reset specific key
switctl config reset template.default
```

## test - Testing

Run tests with advanced options.

### Usage

```bash
switctl test [options]
```

### Options

| Option | Description | Default |
|--------|-------------|---------|
| `--coverage` | Generate coverage report | false |
| `--race` | Enable race detection | false |
| `--bench` | Run benchmarks | false |
| `--short` | Run short tests only | false |
| `--verbose, -v` | Verbose output | false |
| `--package, -p` | Specific package | All packages |
| `--timeout` | Test timeout | 10m |
| `--parallel` | Parallel test count | Auto |

### Examples

```bash
# Basic test run
switctl test

# With coverage and race detection
switctl test --coverage --race

# Specific package with benchmarks
switctl test --package=./internal/service --bench --verbose

# Short tests only
switctl test --short --timeout=5m
```

## deps - Dependency Management

Manage project dependencies.

### Usage

```bash
switctl deps <subcommand> [options]
```

### Subcommands

#### update

Update dependencies to latest versions.

```bash
switctl deps update [options]
```

Options:
- `--security` - Update only security fixes
- `--minor` - Allow minor version updates
- `--major` - Allow major version updates
- `--dry-run` - Show what would be updated

#### check

Check for outdated or vulnerable dependencies.

```bash
switctl deps check [options]
```

Options:
- `--security` - Check for security vulnerabilities
- `--outdated` - Check for outdated packages
- `--format` - Output format (`text`, `json`)

#### clean

Clean up unused dependencies.

```bash
switctl deps clean
```

### Examples

```bash
# Check all dependencies
switctl deps check --security --outdated

# Update with security fixes only
switctl deps update --security

# Clean unused dependencies
switctl deps clean
```

## dev - Development Utilities

Development tools and utilities.

### Usage

```bash
switctl dev <subcommand> [options]
```

### Subcommands

#### watch

Watch for file changes and auto-rebuild.

```bash
switctl dev watch [options]
```

Options:
- `--ignore` - Ignore patterns
- `--command` - Custom build command
- `--delay` - Rebuild delay

#### docs

Generate documentation.

```bash
switctl dev docs [options]
```

Options:
- `--format` - Output format (`html`, `markdown`)
- `--output` - Output directory

#### serve

Run development server.

```bash
switctl dev serve [options]
```

Options:
- `--port` - Server port
- `--host` - Server host
- `--reload` - Auto-reload on changes

### Examples

```bash
# Watch and auto-rebuild
switctl dev watch --ignore="*.log,tmp/"

# Generate documentation
switctl dev docs --format=html --output=./docs

# Development server with auto-reload
switctl dev serve --port=8080 --reload
```

## plugin - Plugin Management

Manage switctl plugins.

### Usage

```bash
switctl plugin <subcommand> [arguments]
```

### Subcommands

#### list

List available plugins.

```bash
switctl plugin list [--installed] [--available]
```

#### install

Install a plugin.

```bash
switctl plugin install <plugin-name>
```

#### uninstall

Uninstall a plugin.

```bash
switctl plugin uninstall <plugin-name>
```

#### update

Update plugins.

```bash
switctl plugin update [plugin-name]
```

### Examples

```bash
# List all plugins
switctl plugin list

# Install specific plugin
switctl plugin install swagger-gen

# Update all plugins
switctl plugin update
```

## Exit Codes

switctl uses standard exit codes:

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Misuse of command |
| 126 | Command invoked cannot execute |
| 127 | Command not found |
| 128+n | Fatal error signal "n" |

## Environment Variables

switctl respects these environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `SWITCTL_CONFIG` | Configuration file path | `~/.switctl.yaml` |
| `SWITCTL_TEMPLATE_DIR` | Custom template directory | Built-in |
| `SWITCTL_PLUGIN_DIR` | Plugin directory | `~/.switctl/plugins` |
| `SWITCTL_DEBUG` | Enable debug mode | false |
| `NO_COLOR` | Disable colored output | false |

## Configuration Files

### Global Configuration

Location: `~/.switctl.yaml`

```yaml
template:
  default: "full-featured"
  directory: "~/.switctl/templates"

database:
  default: "postgresql"
  migrations: true

testing:
  coverage_threshold: 80
  race_detection: true

security:
  scan_dependencies: true
  check_secrets: true

output:
  format: "text"
  color: true
```

### Project Configuration

Location: `.switctl.yaml` (in project root)

```yaml
project:
  name: "my-service"
  version: "1.0.0"
  type: "microservice"

framework:
  version: "latest"
  features:
    - "http"
    - "grpc"  
    - "database"

database:
  type: "postgresql"
  migrations: true
  seeds: true

middleware:
  - "auth"
  - "logging"
  - "recovery"

testing:
  coverage_threshold: 85
  parallel: true
```

## Tips and Best Practices

### Performance Tips

1. **Use `--quick` flag** for faster generation when you know what you want
2. **Specify exact packages** with `--package` for faster testing
3. **Use `.switctl.yaml`** to avoid repeating common options
4. **Enable parallel testing** with appropriate `--parallel` values

### Workflow Tips

1. **Start with templates** - Use established patterns rather than building from scratch
2. **Run checks early** - Use `switctl check` during development, not just CI
3. **Automate with Makefile** - Add switctl commands to your Makefile
4. **Use configuration files** - Store common settings in `.switctl.yaml`

### Troubleshooting

1. **Use debug mode** - Add `--debug` to see what switctl is doing
2. **Check configuration** - Run `switctl config get` to verify settings
3. **Validate templates** - Use `--dry-run` to preview generation
4. **Clean cache** - Delete `~/.switctl/cache` if having issues

For more help, see the [troubleshooting guide](/en/guide/troubleshooting) or run `switctl --help`.
