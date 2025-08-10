# Service Discovery Failure Handling

This document explains the configurable service discovery failure handling feature introduced to address the critical issue where services could start but become unreachable through service discovery systems.

## Problem Statement

Previously, the base server framework would only log warnings when service discovery registration failed, allowing services to start successfully but making them unreachable through service mesh or load balancers. This could lead to:

- **Silent Production Failures**: Services appear healthy but are unreachable via discovery
- **Service Mesh Isolation**: Services not registered with discovery systems like Consul
- **Load Balancer Misses**: Traffic not routed to unregistered service instances

## Solution: Configurable Failure Modes

The framework now supports three distinct failure handling modes for service discovery registration:

### 1. Graceful Mode (`graceful`)
- **Behavior**: Logs warnings but continues startup even if discovery registration fails
- **Use Case**: Development environments where discovery is optional
- **Default**: This is the default mode for backward compatibility

```yaml
server:
  discovery:
    enabled: true
    failure_mode: graceful              # Default behavior
    health_check_required: false        # Default
    registration_timeout: 30s           # Default timeout
```

### 2. Fail-Fast Mode (`fail_fast`)
- **Behavior**: Stops server startup immediately if discovery registration fails
- **Use Case**: Production environments where discovery is critical
- **Reliability**: Ensures services are discoverable before accepting traffic

```yaml
server:
  discovery:
    enabled: true
    failure_mode: fail_fast             # Fail startup on registration failure
    health_check_required: false        # Optional health check
    registration_timeout: 60s           # Allow more time for production
```

### 3. Strict Mode (`strict`)
- **Behavior**: Requires discovery service to be healthy AND fails fast on registration failures
- **Use Case**: Critical production environments with strict reliability requirements
- **Guarantees**: Highest level of discovery reliability

```yaml
server:
  discovery:
    enabled: true
    failure_mode: strict                # Strictest mode
    health_check_required: true         # Requires discovery health check
    registration_timeout: 60s           # Production timeout
```

## Configuration Options

### Core Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `failure_mode` | string | `graceful` | Controls behavior on discovery failure |
| `health_check_required` | bool | `false` | Whether to check discovery health first |
| `registration_timeout` | duration | `30s` | Timeout for registration operations |

### Failure Mode Values

| Value | Behavior | Production Ready |
|-------|----------|-----------------|
| `graceful` | Continue startup on failure | No - for development |
| `fail_fast` | Stop startup on failure | Yes - recommended |
| `strict` | Health check + fail fast | Yes - critical systems |

## Examples

### Development Environment
```yaml
server:
  service_name: myservice-dev
  discovery:
    enabled: true
    address: "localhost:8500"
    failure_mode: graceful              # Service starts even without Consul
    health_check_required: false
    registration_timeout: 30s
    tags: ["dev", "v1"]
```

### Production Environment
```yaml
server:
  service_name: myservice-prod
  discovery:
    enabled: true
    address: "consul.production:8500"
    failure_mode: fail_fast             # Must register successfully
    health_check_required: false
    registration_timeout: 60s
    tags: ["prod", "v1"]
```

### Critical Production Environment
```yaml
server:
  service_name: myservice-critical
  discovery:
    enabled: true
    address: "consul.production:8500"
    failure_mode: strict                # Health check + fail fast
    health_check_required: true
    registration_timeout: 45s
    tags: ["prod", "critical", "v1"]
```

## Logging and Monitoring

### Graceful Mode Logs
```
WARN: Failed to register with service discovery in graceful mode
      error=<error details>
      failure_mode=graceful
      action=continuing without discovery registration
```

### Fail-Fast Mode Logs
```
ERROR: Service discovery registration failed with fail-fast mode enabled
       error=<error details>
       failure_mode=fail_fast
       action=failing server startup
```

### Strict Mode Logs
```
INFO: Checking discovery service health before registration (strict mode)
INFO: Discovery service health check passed in strict mode

# OR on failure:
ERROR: Service discovery registration failed with fail-fast mode enabled
       error=discovery service is not healthy and strict mode is enabled
       failure_mode=strict
       action=failing server startup
```

## Error Handling Details

### Timeout Handling
All failure modes respect the `registration_timeout` configuration:

```go
// Context timeout is applied to registration operations
regCtx, cancel := context.WithTimeout(ctx, config.Discovery.RegistrationTimeout)
defer cancel()
```

### Health Check in Strict Mode
When `failure_mode: strict` and `health_check_required: true`:

1. Discovery health check is performed first (10s timeout)
2. If health check fails, registration is aborted
3. If health check passes, normal registration proceeds
4. Registration failure still causes startup failure

### Error Messages
- **Timeout**: "service discovery registration timed out after {timeout}"
- **Health Check**: "discovery service is not healthy and strict mode is enabled"
- **General Failure**: "failed to register service endpoints: {error}"

## Migration Guide

### Existing Configurations
Existing configurations continue to work unchanged:
- `failure_mode` defaults to `graceful` (existing behavior)
- `health_check_required` defaults to `false`
- `registration_timeout` defaults to `30s`

### Recommended Migration Path

1. **Development**: Keep `graceful` mode
2. **Staging**: Switch to `fail_fast` mode for testing
3. **Production**: Use `fail_fast` or `strict` mode

### Validation
The configuration includes validation:
- `failure_mode` must be one of: `graceful`, `fail_fast`, `strict`
- `registration_timeout` must be positive and ≤ 5 minutes

## Testing

### Unit Tests
The implementation includes comprehensive unit tests covering:
- Configuration validation
- All failure modes
- Timeout handling
- Health check behavior
- Backward compatibility

### Integration Tests
Run discovery failure tests:
```bash
go test ./pkg/server -run TestDiscoveryFailureMode -v
```

### Manual Testing

#### Test Graceful Mode (Development)
1. Start service without Consul running
2. Service should start successfully
3. Check logs for warning messages

#### Test Fail-Fast Mode (Production)
1. Start service without Consul running
2. Service should fail to start
3. Check logs for error messages

#### Test Strict Mode (Critical)
1. Start Consul but make it unhealthy
2. Service should fail to start with health check error
3. Fix Consul health and retry - should succeed

## Best Practices

1. **Development**: Use `graceful` mode for local development
2. **CI/CD**: Use `fail_fast` mode in automated testing
3. **Production**: Use `fail_fast` or `strict` mode depending on criticality
4. **Monitoring**: Set up alerts for discovery registration failures
5. **Timeouts**: Use longer timeouts in production (60s+)
6. **Health Checks**: Enable health checks for mission-critical services

## Troubleshooting

### Common Issues

**Service won't start with fail_fast mode**
- Check Consul connectivity: `curl http://consul:8500/v1/status/leader`
- Verify network access and firewall rules
- Check Consul logs for registration errors

**Timeouts in production**
- Increase `registration_timeout` for high-latency networks
- Monitor Consul performance and resource usage
- Consider network optimization

**Health checks failing**
- Verify Consul cluster health
- Check Consul agent configuration
- Review Consul ACL permissions

### Debug Commands

```bash
# Check Consul connectivity
consul members

# Test service registration manually
curl -X PUT http://consul:8500/v1/agent/service/register \
  -d '{"ID":"test","Name":"test","Address":"localhost","Port":8080}'

# Check registered services
curl http://consul:8500/v1/agent/services
```

## Security Considerations

1. **Network Security**: Ensure secure communication with discovery services
2. **Authentication**: Configure Consul ACLs appropriately
3. **Encryption**: Use TLS for Consul communications in production
4. **Monitoring**: Log all discovery operations for audit trails

## Performance Impact

- **Graceful Mode**: Minimal impact, single registration attempt
- **Fail-Fast Mode**: Single registration attempt with timeout
- **Strict Mode**: Additional health check (~10s timeout) + registration

The performance impact is minimal and acceptable for production use.