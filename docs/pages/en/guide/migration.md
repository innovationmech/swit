# Migration and Upgrade Guide

This guide helps you migrate to new features and upgrade existing Swit framework applications to take advantage of the latest enhancements including Sentry monitoring, switctl CLI tools, and security improvements.

## Recent Major Updates

### Version Timeline

- **v0.3.0**: Comprehensive Sentry integration, switctl CLI tool, security enhancements
- **v0.2.0**: Enhanced testing framework, race condition fixes
- **v0.1.0**: Core framework with HTTP/gRPC transports

## Upgrading to Sentry Integration (v0.3.0)

### 1. Update Dependencies

Add Sentry SDK to your project:

```bash
go get github.com/getsentry/sentry-go@v0.27.0
```

Update your `go.mod`:

```go
require (
    github.com/innovationmech/swit v0.3.0
    github.com/getsentry/sentry-go v0.27.0
)
```

### 2. Configuration Migration

#### Before (v0.2.0 and earlier)
```yaml
service_name: "my-service"
http:
  port: "8080"
grpc:
  port: "9080"
```

#### After (v0.3.0+)
```yaml
service_name: "my-service"
http:
  port: "8080"
grpc:
  port: "9080"

# Add Sentry configuration
sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "production"
  sample_rate: 1.0
  traces_sample_rate: 0.1
```

### 3. Environment Variables

Add Sentry environment variables:

```bash
# Required for Sentry
export SENTRY_DSN="https://your-dsn@sentry.io/project-id"

# Optional
export SENTRY_ENVIRONMENT="production"
export SENTRY_RELEASE="v1.0.0"
```

### 4. Code Changes (Optional)

The framework handles Sentry automatically, but you can add custom context:

```go
// Before: Basic error handling
func (h *Handler) ProcessOrder(c *gin.Context) {
    if err := h.service.Process(); err != nil {
        log.Error("Process failed", zap.Error(err))
        c.JSON(500, gin.H{"error": "Internal error"})
        return
    }
}

// After: Enhanced with Sentry context
func (h *Handler) ProcessOrder(c *gin.Context) {
    if err := h.service.Process(); err != nil {
        // Keep existing logging
        log.Error("Process failed", zap.Error(err))
        
        // Add Sentry context (optional)
        sentry.WithScope(func(scope *sentry.Scope) {
            scope.SetTag("operation", "process_order")
            scope.SetContext("request", map[string]interface{}{
                "user_id": c.GetHeader("User-ID"),
                "order_id": c.Param("id"),
            })
            sentry.CaptureException(err)
        })
        
        c.JSON(500, gin.H{"error": "Internal error"})
        return
    }
}
```

### 5. Gradual Rollout Strategy

#### Phase 1: Enable with Low Sampling
```yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  sample_rate: 0.1          # Start with 10% sampling
  traces_sample_rate: 0.01  # 1% performance sampling
```

#### Phase 2: Increase Sampling
```yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  sample_rate: 1.0          # Full error sampling
  traces_sample_rate: 0.1   # 10% performance sampling
```

#### Phase 3: Full Production Configuration
```yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "production"
  sample_rate: 1.0
  traces_sample_rate: 0.1
  enable_profiling: true
  
  # Production filtering
  http_ignore_status_codes: [400, 401, 403, 404]
  http_ignore_paths: ["/health", "/metrics"]
```

## Adopting switctl CLI Tools

### 1. Installation

Build switctl from the framework source:

```bash
# From framework root directory
make build

# Add to PATH
sudo cp ./bin/switctl /usr/local/bin/
```

### 2. Existing Project Integration

Initialize switctl in existing projects:

```bash
# In your project directory
switctl init --existing

# This creates .switctl.yaml with current project configuration
```

### 3. Configuration File Creation

switctl will create `.switctl.yaml`:

```yaml
project:
  name: "my-service"
  type: "microservice"
  
templates:
  default: "http-grpc"
  
database:
  type: "postgresql"  # Based on existing configuration
  
testing:
  coverage_threshold: 80
  race_detection: true
  
security:
  enabled: true
  scan_deps: true
```

### 4. Gradual Adoption Workflow

#### Start with Quality Checks
```bash
# Run checks on existing codebase
switctl check --quality
switctl check --security
switctl check --tests --coverage
```

#### Add Code Generation
```bash
# Generate new components using templates
switctl generate api user --methods=crud
switctl generate middleware auth --type=jwt
```

#### Adopt Development Workflow
```bash
# Use development utilities
switctl dev watch      # Auto-rebuild on changes
switctl deps check     # Dependency management
```

## Messaging Adapter Switching (preview)

Before switching messaging adapters (e.g., RabbitMQ → Kafka → NATS), generate a migration plan to surface capability deltas and a safe rollout checklist:

```go
ctx := context.Background()
plan, err := messaging.PlanBrokerSwitch(ctx, currentConfig, targetConfig)
if err != nil {
    return fmt.Errorf("plan switch: %w", err)
}
fmt.Printf("Switching %s -> %s (score %d)\n", plan.CurrentType, plan.TargetType, plan.CompatibilityScore)
for _, item := range plan.Checklist {
    fmt.Printf("✔ %s\n", item)
}
```

See detailed guidance in the Messaging Adapter Switching guide.

## Security Enhancements Migration

### 1. GitHub Actions Workflow Updates

#### Before: Minimal permissions
```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
```

#### After: Explicit minimal permissions
```yaml
name: CI
on: [push, pull_request]

# Add explicit permissions
permissions:
  contents: read

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
```

### 2. Update All Workflows

Apply security improvements to all workflow files:

```bash
# Update existing workflows
find .github/workflows -name "*.yml" -o -name "*.yaml" | \
  xargs sed -i '1i permissions:\n  contents: read\n'
```

### 3. Dependency Security Scanning

Use switctl for enhanced security:

```bash
# Scan dependencies for vulnerabilities
switctl check --security --deps

# Check for outdated packages
switctl deps check --security
```

## Testing Framework Enhancements

### 1. Race Condition Fixes

If you experienced race conditions in tests, update test patterns:

#### Before: Potential race conditions
```go
func TestConcurrentAccess(t *testing.T) {
    service := NewService()
    
    // Multiple goroutines without proper synchronization
    for i := 0; i < 10; i++ {
        go func() {
            service.Process()
        }()
    }
}
```

#### After: Proper synchronization
```go
func TestConcurrentAccess(t *testing.T) {
    service := NewService()
    
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            service.Process()
        }()
    }
    wg.Wait()
}
```

### 2. Enhanced Test Commands

Use new test management features:

```bash
# Run tests with race detection
make test-race

# Generate coverage reports
make test-coverage

# Use switctl for advanced testing
switctl test --coverage --race --benchmark
```

## Configuration Schema Updates

### 1. ServerConfig Structure Changes

New configuration fields added:

```go
type ServerConfig struct {
    // Existing fields
    ServiceName     string        `yaml:"service_name"`
    ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
    HTTP            HTTPConfig    `yaml:"http"`
    GRPC            GRPCConfig    `yaml:"grpc"`
    Discovery       DiscoveryConfig `yaml:"discovery"`
    
    // NEW: Sentry configuration
    Sentry          SentryConfig  `yaml:"sentry"`
}
```

### 2. Validation Enhancements

Enhanced configuration validation:

```go
// Configuration is now validated more strictly
config := &ServerConfig{
    Sentry: SentryConfig{
        Enabled:           true,
        DSN:              "invalid-format", // Will cause validation error
        SampleRate:       2.0,             // Will cause validation error (must be 0.0-1.0)
    },
}

if err := config.Validate(); err != nil {
    // More detailed error messages
    log.Fatal("Configuration validation failed:", err)
}
```

## Backward Compatibility

### What's Preserved

- ✅ All existing HTTP/gRPC transport configurations
- ✅ Service discovery settings
- ✅ Middleware configurations
- ✅ Business service interfaces
- ✅ Dependency injection patterns
- ✅ Health check implementations

### What's New (Opt-in)

- 🆕 Sentry configuration (disabled by default)
- 🆕 switctl CLI tools (optional installation)
- 🆕 Enhanced security checks
- 🆕 Improved testing utilities

### Breaking Changes

**None** - This is a backward-compatible update. All existing services will continue to work without changes.

## Step-by-Step Migration Checklist

### Pre-Migration
- [ ] Backup existing configuration files
- [ ] Review current error handling patterns
- [ ] Document current monitoring setup
- [ ] Test current application thoroughly

### Dependencies
- [ ] Update framework dependency to v0.3.0+
- [ ] Add Sentry SDK dependency
- [ ] Run `go mod tidy`
- [ ] Verify all tests pass

### Configuration
- [ ] Add Sentry section to configuration files
- [ ] Set up environment variables
- [ ] Configure appropriate sampling rates
- [ ] Test configuration validation

### Development Tools
- [ ] Build and install switctl CLI
- [ ] Initialize .switctl.yaml in projects
- [ ] Run quality checks with switctl
- [ ] Update development scripts/Makefile

### Security
- [ ] Update GitHub Actions workflows
- [ ] Add explicit permissions
- [ ] Run security scans
- [ ] Review dependency vulnerabilities

### Testing
- [ ] Run race detection tests
- [ ] Generate coverage reports
- [ ] Test Sentry integration (with mock DSN)
- [ ] Verify monitoring works in staging

### Production Deployment
- [ ] Deploy with Sentry disabled initially
- [ ] Enable Sentry with low sampling
- [ ] Monitor Sentry dashboard
- [ ] Gradually increase sampling rates
- [ ] Configure production filters

## Configuration Migration & Upgrade (switctl)

Use switctl to validate and migrate configuration files across versions. Start with strict validation, then run migration with backups.

```bash
# Strict validation (fail on warnings)
switctl config validate --strict

# Migrate configuration from v1 to v2 with backup
switctl config migrate --from v1 --to v2 --backup

# Optionally provide explicit rules (JSON/YAML)
switctl config migrate --from v1 --to v2 --rules ./migration-rules.yaml
```

Key behaviors:
- Key remapping and removal (e.g., `a.b` → `x.y`, or remove deprecated keys)
- Default value backfilling via `default`
- Version stamping (`version: v2`)

Environment variables follow the generated mapping rules:
- Prefix: `SWIT_`
- Mapping: config key `a.b.c` → env var `SWIT_A_B_C`
- Precedence (low → high): Defaults, Base file, Env file, Override file, Environment variables

See also: `/en/guide/configuration-reference` (generated)

## Rollback Procedures

### If Sentry Causes Issues

1. **Immediate disable**:
```yaml
sentry:
  enabled: false
```

2. **Or remove configuration entirely** - service will work without Sentry

### If switctl Causes Issues

switctl is completely optional:
- Remove .switctl.yaml file
- Continue using existing build processes
- Framework functions normally without CLI tools

### If Security Updates Cause Issues

Revert GitHub Actions workflows:
```bash
git checkout HEAD~1 .github/workflows/
```

## Common Migration Issues

### Issue 1: Sentry Events Not Appearing

**Symptoms**: No events in Sentry dashboard
**Solutions**:
1. Verify DSN format and network connectivity
2. Enable debug mode: `sentry.debug: true`
3. Check sampling rates (must be > 0)
4. Test with curl requests to error endpoints

### Issue 2: Too Many Sentry Events

**Symptoms**: High event volume, billing concerns
**Solutions**:
1. Lower sampling rates
2. Add more filters
3. Filter out health checks and expected errors

### Issue 3: switctl Command Not Found

**Symptoms**: `switctl: command not found`
**Solutions**:
1. Verify build: `make build`
2. Check PATH: `which switctl`
3. Use full path: `./bin/switctl`

### Issue 4: Performance Impact

**Symptoms**: Increased latency or memory usage
**Solutions**:
1. Lower performance sampling rates
2. Disable profiling in production
3. Monitor with smaller batch sizes

## Getting Help

### Documentation

- [Sentry Integration Guide](/en/guide/monitoring)
- [CLI Tools Documentation](/en/cli/)
- [Configuration Reference](/en/guide/configuration)

### Community Support

- [GitHub Issues](https://github.com/innovationmech/swit/issues)
- [Community Discussions](/en/community/)
- [Contributing Guide](/en/community/contributing)

### Professional Support

For enterprise migrations or complex scenarios:
- Create detailed GitHub issue with configuration
- Provide logs and error messages
- Include environment and version information

## Best Practices for Migration

### 1. Incremental Approach
- Migrate one service at a time
- Start with development/staging environments
- Use feature flags for gradual rollout

### 2. Testing Strategy
- Test new features in isolation
- Run comprehensive integration tests
- Monitor performance during migration

### 3. Monitoring During Migration
- Watch error rates during deployment
- Monitor performance metrics
- Have rollback plan ready

### 4. Team Communication
- Inform team about new monitoring capabilities
- Share switctl CLI tool usage
- Document new development workflows

This migration guide ensures smooth transition to the latest Swit framework features while maintaining system reliability and team productivity.
