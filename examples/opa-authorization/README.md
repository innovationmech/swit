# OPA Authorization Example

English | [中文](README-CN.md)

This example demonstrates how to integrate Open Policy Agent (OPA) for fine-grained authorization in a microservice, supporting both embedded and remote OPA modes with RBAC and ABAC policies.

## Features

- ✅ **Embedded OPA Mode** - OPA engine runs in the same process
- ✅ **Remote OPA Mode** - Connect to external OPA server
- ✅ **RBAC Policies** - Role-based access control
- ✅ **ABAC Policies** - Attribute-based access control
- ✅ **HTTP Middleware** - Gin-based policy enforcement
- ✅ **gRPC Interceptor** - gRPC policy enforcement
- ✅ **Decision Caching** - Performance optimization
- ✅ **Audit Logging** - Policy decision tracking
- ✅ **Docker Support** - Complete containerized setup

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Client Requests                       │
└───────────────┬─────────────────────────────────────────┘
                │
                ├─── HTTP (Port 8080)
                │    │
                │    v
                │    ┌──────────────────────────┐
                │    │  Gin Router              │
                │    │  + OPA Middleware        │
                │    └───────────┬──────────────┘
                │                │
                └─── gRPC (Port 9090)
                     │
                     v
                     ┌──────────────────────────┐
                     │  gRPC Server             │
                     │  + Policy Interceptor    │
                     └───────────┬──────────────┘
                                 │
        ┌────────────────────────┴────────────────────────┐
        │                                                  │
        v                                                  v
┌───────────────────┐                           ┌──────────────────┐
│  Embedded OPA     │                           │  Remote OPA      │
│  (In-Process)     │                           │  (External)      │
│  - RBAC Policy    │                           │  - Port 8181     │
│  - ABAC Policy    │                           │  - Load Balanced │
│  - Local Cache    │                           │  - Health Check  │
└───────────────────┘                           └──────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.23+
- Docker and Docker Compose (optional)
- OPA CLI (optional, for policy testing)

### 1. Run with Embedded Mode

```bash
# Navigate to the example directory
cd examples/opa-authorization

# Run with RBAC policy
go run main.go -opa-mode embedded -policy-type rbac -policy-dir ./policies

# Run with ABAC policy
go run main.go -opa-mode embedded -policy-type abac -policy-dir ./policies
```

### 2. Run with Remote Mode (Docker)

```bash
# Start OPA server and example applications
docker-compose up

# The following services will be available:
# - OPA Server: http://localhost:8181
# - App (Embedded): http://localhost:8080
# - App (Remote): http://localhost:8081
```

### 3. Run with Configuration File

```bash
# Using swit.yaml configuration file
go run main.go -config swit.yaml
```

## API Testing

### Admin User (Full Access)

```bash
# List all documents
curl -H "X-User: alice" -H "X-Roles: admin" \
  http://localhost:8080/api/v1/documents

# Create a document
curl -X POST -H "X-User: alice" -H "X-Roles: admin" \
  -H "Content-Type: application/json" \
  -d '{"id":"doc-4","title":"New Document","content":"Content"}' \
  http://localhost:8080/api/v1/documents

# Delete a document
curl -X DELETE -H "X-User: alice" -H "X-Roles: admin" \
  http://localhost:8080/api/v1/documents/doc-4
```

### Editor User (Read, Create, Update)

```bash
# Read documents
curl -H "X-User: bob" -H "X-Roles: editor" \
  http://localhost:8080/api/v1/documents

# Create a document
curl -X POST -H "X-User: bob" -H "X-Roles: editor" \
  -H "Content-Type: application/json" \
  -d '{"id":"doc-5","title":"Bob Document","content":"Content"}' \
  http://localhost:8080/api/v1/documents

# Update a document
curl -X PUT -H "X-User: bob" -H "X-Roles: editor" \
  -H "Content-Type: application/json" \
  -d '{"title":"Updated Title","content":"Updated Content"}' \
  http://localhost:8080/api/v1/documents/doc-5

# Delete a document (should be denied)
curl -X DELETE -H "X-User: bob" -H "X-Roles: editor" \
  http://localhost:8080/api/v1/documents/doc-5
```

### Viewer User (Read Only)

```bash
# Read documents
curl -H "X-User: charlie" -H "X-Roles: viewer" \
  http://localhost:8080/api/v1/documents

# Create a document (should be denied)
curl -X POST -H "X-User: charlie" -H "X-Roles: viewer" \
  -H "Content-Type: application/json" \
  -d '{"id":"doc-6","title":"Test","content":"Content"}' \
  http://localhost:8080/api/v1/documents
```

### Anonymous User (No Access)

```bash
# Access documents (should be denied, except health check)
curl http://localhost:8080/api/v1/documents

# Health check (allows anonymous access)
curl http://localhost:8080/api/v1/health
```

## Configuration Options

### Command-Line Flags

- `-port` - HTTP server port (default: 8080)
- `-grpc-port` - gRPC server port (default: 9090)
- `-opa-mode` - OPA mode: `embedded` or `remote` (default: embedded)
- `-opa-url` - OPA server URL for remote mode (default: http://localhost:8181)
- `-policy-dir` - Policy directory for embedded mode (default: ./policies)
- `-policy-type` - Policy type: `rbac` or `abac` (default: rbac)
- `-config` - Configuration file path (default: swit.yaml)

### Environment Variables

- `OPA_MODE` - OPA mode
- `OPA_URL` - OPA server URL
- `POLICY_DIR` - Policy directory
- `POLICY_TYPE` - Policy type

### Configuration File Example (swit.yaml)

See the [swit.yaml](swit.yaml) file for a complete OPA configuration example.

## Policy Explanation

### RBAC Policy (Role-Based Access Control)

The RBAC policy defines role-based permissions:

#### Role Definitions

| Role | Permissions | Description |
|------|-------------|-------------|
| **admin** | Full access | Complete access to all resources |
| **editor** | Read, Create, Update | Can create, read, and update documents (no delete) |
| **viewer** | Read only | Can only read documents, no modifications |
| **owner** | Resource owner | Can perform any action on owned documents |

#### RBAC Rule Examples

```rego
# Admin has all permissions
allow if {
    "admin" in input.subject.roles
}

# Editor can create, read, and update documents
allow if {
    "editor" in input.subject.roles
    input.action in ["GET", "POST", "PUT"]
    startswith(input.resource.path, "/api/v1/documents")
}

# Viewer can only read documents
allow if {
    "viewer" in input.subject.roles
    input.action == "GET"
    startswith(input.resource.path, "/api/v1/documents")
}
```

### ABAC Policy (Attribute-Based Access Control)

The ABAC policy adds attribute-based constraints for more fine-grained access control:

#### Attribute Types

| Attribute Type | Description | Example |
|----------------|-------------|---------|
| **Time-based** | Access restricted to business hours (9:00-18:00) | Editors can only modify documents during work hours |
| **IP-based** | Access restricted to allowed IP ranges | Only internal IPs (192.168.x.x, 172.x.x.x) |
| **Resource attributes** | Fine-grained control based on resource properties | Document owner, document type, etc. |
| **Context attributes** | Decisions based on request context | Request protocol, user agent, etc. |

#### ABAC Rule Examples

```rego
# Editors can only modify documents during business hours and from allowed IPs
allow if {
    "editor" in input.subject.roles
    input.action in ["POST", "PUT"]
    input.resource.type == "document"
    is_business_hours
    is_allowed_ip
}

# Check if it's business hours (9:00 - 18:00)
is_business_hours if {
    hour := time.clock([input.context.time, "UTC"])[0]
    hour >= 9
    hour < 18
}

# Check if IP address is in allowed range
is_allowed_ip if {
    startswith(input.context.client_ip, "192.168.")
}
```

## Testing

### Unit Tests

```bash
# Run unit tests
go test ./pkg/security/opa/...

# Run specific tests
go test -run TestEmbeddedClient ./pkg/security/opa/...

# View test coverage
go test -cover ./pkg/security/opa/...
```

### Integration Tests

```bash
# Run integration tests (requires integration tag)
go test -tags=integration ./pkg/security/opa/...

# With remote OPA (requires OPA server running)
OPA_URL=http://localhost:8181 go test -tags=integration ./pkg/security/opa/...
```

### Policy Tests

```bash
# Test policies using OPA CLI
cd policies

# Test RBAC policy
opa eval -d rbac.rego 'data.rbac.allow' \
  -i test_input_rbac.json

# Test ABAC policy
opa eval -d abac.rego 'data.abac.allow' \
  -i test_input_abac.json

# Check policy syntax
opa check rbac.rego abac.rego
```

### Benchmark Tests

```bash
# Run performance benchmarks
go test -bench=. -benchmem ./pkg/security/opa/...

# Run specific benchmark
go test -bench=BenchmarkEmbeddedClientEvaluate ./pkg/security/opa/...
```

## Performance

The example demonstrates OPA's high-performance policy evaluation:

| Metric | Embedded Mode | Remote Mode |
|--------|---------------|-------------|
| **P50 latency** | < 1ms | < 10ms |
| **P99 latency** | < 5ms | < 50ms |
| **Throughput** | 10,000+ decisions/sec | 5,000+ decisions/sec |
| **Cache hit ratio** | > 90% | > 85% |

### Performance Optimization Tips

1. **Enable decision caching** - Reduce repeated policy evaluations
2. **Use embedded mode** - Avoid network overhead
3. **Optimize policy rules** - Reduce computational complexity
4. **Batch evaluation** - Evaluate multiple decisions at once

## Directory Structure

```
examples/opa-authorization/
├── main.go                 # Main application
├── README.md              # English documentation
├── README-CN.md           # Chinese documentation
├── swit.yaml              # Configuration file example
├── Dockerfile             # Container image definition
├── docker-compose.yml     # Multi-container setup
└── policies/              # OPA policies
    ├── rbac.rego          # RBAC policy
    └── abac.rego          # ABAC policy
```

## Troubleshooting

### OPA Server Connection Failed

```bash
# Check if OPA server is running
curl http://localhost:8181/health

# View OPA server logs
docker logs opa-server

# Start OPA server manually
docker run -p 8181:8181 openpolicyagent/opa:latest run --server --log-level=debug
```

### Policy Evaluation Failed

```bash
# Test policy with OPA CLI
cd policies
opa eval -d rbac.rego 'data.rbac.allow' -i <(echo '
{
  "subject": {"user": "alice", "roles": ["admin"]},
  "action": "GET",
  "resource": {"path": "/api/v1/documents", "type": "document"}
}')

# Check policy syntax
opa check policies/

# View detailed policy evaluation
opa eval -d rbac.rego 'data.rbac' --explain full -i input.json
```

### Unexpected Permission Denied

Common causes and solutions:

1. **Missing or incorrect request headers**
   - Check `X-User` and `X-Roles` headers
   - Ensure role names are spelled correctly

2. **Policy path mismatch**
   - Check policy decision path configuration
   - Confirm correct policy package name (rbac/abac)

3. **Incorrect input data format**
   - Review audit logs for decision reasons
   - Verify policy input data structure

4. **Cache issues**
   - Clear decision cache and retry
   - Check cache TTL configuration

### Performance Issues

```bash
# View policy evaluation performance
go test -bench=BenchmarkEmbeddedClientEvaluate -benchmem ./pkg/security/opa/

# Enable verbose logging to identify bottlenecks
go run main.go -opa-mode embedded --log-level=debug

# Monitor OPA server performance (remote mode)
curl http://localhost:8181/metrics
```

## Best Practices

### 1. Policy Design

- **Principle of least privilege** - Default deny, explicit allow
- **Role hierarchy** - Design reasonable role hierarchies
- **Policy modularization** - Split policies into reusable modules
- **Audit logging** - Record all decisions for auditing

### 2. Performance Optimization

- **Enable caching** - For decisions that don't change frequently
- **Use embedded mode** - When policies don't need hot updates
- **Batch evaluation** - Evaluate multiple related decisions at once
- **Optimize policy rules** - Avoid complex loops and recursion

### 3. Security Considerations

- **Sensitive data protection** - Don't include sensitive info in policy input
- **Policy version control** - Use Git to manage policy versions
- **Policy testing** - Write comprehensive policy test cases
- **Regular audits** - Regularly review policy decision logs

### 4. Operations Recommendations

- **Monitoring and alerting** - Monitor policy evaluation performance and error rates
- **Policy hot updates** - Use remote mode for policy hot updates
- **Disaster recovery** - Prepare policy rollback plans
- **Documentation maintenance** - Keep policy documentation in sync with code

## Extended Examples

### Multi-Tenant Scenario

See `pkg/security/opa/policies/examples/multi_tenant_abac.rego` for multi-tenant isolation example.

### Financial System Scenario

See `pkg/security/opa/policies/examples/financial_abac.rego` for financial-grade access control.

### Healthcare System Scenario

See `pkg/security/opa/policies/examples/healthcare_abac.rego` for healthcare data access control.

## References

### Official Documentation

- [OPA Official Documentation](https://www.openpolicyagent.org/docs/)
- [Rego Language Reference](https://www.openpolicyagent.org/docs/latest/policy-language/)
- [OPA REST API](https://www.openpolicyagent.org/docs/latest/rest-api/)

### Project Documentation

- [OPA RBAC Guide](../../docs/opa-rbac-guide.md)
- [OPA ABAC Guide](../../docs/opa-abac-guide.md)
- [OPA Policy Guide](../../docs/opa-policy-guide.md)
- [Security Configuration Reference](../../docs/security-configuration-reference.md)

### Related Examples

- [OAuth2 Authentication Example](../oauth2-authentication/)
- [mTLS Example](../security/)
- [Comprehensive Security Example](../security/)

## Contributing

Contributions of more policy examples and use cases are welcome! Please refer to the [Contributing Guide](../../CONTRIBUTING.md).

## License

Apache License 2.0 - See [LICENSE](../../LICENSE) file for details.
