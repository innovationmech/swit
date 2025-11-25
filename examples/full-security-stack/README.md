# Full Security Stack Example

This example demonstrates a comprehensive security integration using the Swit framework, combining OAuth2 authentication, OPA authorization, mTLS transport encryption, audit logging, and security metrics monitoring.

## Features

- **OAuth2/OIDC Authentication**: Integration with Keycloak for user authentication
- **OPA Authorization**: Policy-based access control using Open Policy Agent
- **mTLS Support**: Mutual TLS for secure transport encryption
- **Audit Logging**: Comprehensive security event logging
- **Security Metrics**: Prometheus metrics for security monitoring
- **Grafana Dashboards**: Pre-configured dashboards for security visualization

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Request                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    mTLS Layer (Optional)                         │
│                  Certificate Verification                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   OAuth2 Authentication                          │
│              JWT Token Validation (Keycloak)                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    OPA Authorization                             │
│              RBAC/ABAC Policy Evaluation                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Business Logic Handler                          │
│                    Document Management                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Audit & Metrics                               │
│          Security Events → Prometheus → Grafana                  │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Docker and Docker Compose
- Go 1.23+ (for local development)
- curl or httpie (for testing)

## Quick Start

### 1. Start All Services

```bash
# Navigate to the example directory
cd examples/full-security-stack

# Start all services with Docker Compose
docker-compose up -d
```

This will start:
- **PostgreSQL** (port 5432) - Database for Keycloak
- **Keycloak** (port 8081) - OAuth2/OIDC Provider
- **OPA** (port 8181) - Open Policy Agent
- **Prometheus** (port 9090) - Metrics Collection
- **Grafana** (port 3000) - Metrics Visualization
- **Alertmanager** (port 9093) - Alert Management
- **Application** (port 8080) - Full Security Stack Example

### 2. Wait for Services to Start

```bash
# Check service health
docker-compose ps

# Wait for Keycloak to be ready (may take 1-2 minutes)
docker-compose logs -f keycloak
```

### 3. Access Services

| Service | URL | Credentials |
|---------|-----|-------------|
| Application | http://localhost:8080 | - |
| Keycloak Admin | http://localhost:8081 | admin / admin |
| Grafana | http://localhost:3000 | admin / admin |
| Prometheus | http://localhost:9090 | - |
| OPA | http://localhost:8181 | - |

## Testing the API

### Public Endpoints

```bash
# Get service info
curl http://localhost:8080/api/v1/public/info

# Get security status
curl http://localhost:8080/api/v1/public/security-status
```

### Authentication Flow

```bash
# 1. Start OAuth2 login flow
curl http://localhost:8080/api/v1/public/login

# 2. Open the authorization_url in your browser
# 3. Login with Keycloak credentials (see test users below)
# 4. Use the returned access_token for protected endpoints
```

### Test Users (Pre-configured in Keycloak)

| Username | Password | Roles |
|----------|----------|-------|
| alice | alice123 | admin |
| bob | bob123 | editor |
| charlie | charlie123 | viewer |
| dave | dave123 | guest |

### Protected Endpoints

```bash
# Set your access token
TOKEN="your_access_token_here"

# Get user profile
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/protected/profile

# List documents
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/protected/documents

# Get specific document
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/protected/documents/doc-1

# Create document (requires editor or admin role)
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"doc-5","title":"New Doc","content":"Content","classification":"internal"}' \
  http://localhost:8080/api/v1/protected/documents
```

### Admin Endpoints

```bash
# Admin dashboard (requires admin role)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/admin/dashboard

# Audit logs
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/admin/audit-logs

# Security metrics
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/admin/security-metrics
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_PORT` | HTTP server port | 8080 |
| `OAUTH2_ENABLED` | Enable OAuth2 authentication | true |
| `OPA_ENABLED` | Enable OPA authorization | true |
| `MTLS_ENABLED` | Enable mTLS | false |
| `AUDIT_ENABLED` | Enable audit logging | true |
| `METRICS_ENABLED` | Enable security metrics | true |
| `POLICY_TYPE` | OPA policy type (rbac/abac) | rbac |
| `OPA_MODE` | OPA mode (embedded/remote) | embedded |
| `OAUTH2_ISSUER_URL` | Keycloak issuer URL | http://localhost:8081/realms/swit |

### Policy Types

#### RBAC (Role-Based Access Control)

```bash
# Use RBAC policy
docker-compose run -e POLICY_TYPE=rbac app
```

Roles:
- **admin**: Full access to all resources
- **editor**: Create, read, update documents
- **viewer**: Read-only access
- **guest**: Limited access

#### ABAC (Attribute-Based Access Control)

```bash
# Use ABAC policy
docker-compose run -e POLICY_TYPE=abac app
```

Attributes considered:
- User roles and department
- Resource classification and owner
- Time of day (business hours)
- Client IP address

## mTLS Configuration

### Generate Certificates

```bash
# Generate CA, server, and client certificates
./scripts/generate-certs.sh
```

### Enable mTLS

```bash
# Start with mTLS enabled
docker-compose -f docker-compose.yml -f docker-compose.mtls.yml up -d

# Or set environment variable
MTLS_ENABLED=true docker-compose up -d
```

### Test mTLS

```bash
# Test with client certificate
curl --cacert certs/ca.crt \
     --cert certs/client.crt \
     --key certs/client.key \
     https://localhost:8443/api/v1/mtls/verify
```

## Monitoring

### Prometheus Metrics

Access Prometheus at http://localhost:9090

Key metrics:
- `security_auth_attempts_total` - Authentication attempts
- `security_policy_evaluations_total` - OPA policy evaluations
- `security_security_events_total` - Security events
- `security_tls_connections_total` - TLS connections

### Grafana Dashboards

Access Grafana at http://localhost:3000 (admin/admin)

Pre-configured dashboards:
- **Full Security Stack Dashboard**: Overview of all security metrics
- Authentication metrics
- Authorization metrics
- Security events

### Alerting

Alertmanager is configured with the following alerts:
- High authentication failure rate
- Potential brute force attack
- High policy denial rate
- Critical security events
- Certificate expiring soon

## Project Structure

```
full-security-stack/
├── main.go                 # Main application
├── swit.yaml               # Configuration file
├── docker-compose.yml      # Docker Compose configuration
├── Dockerfile              # Application Dockerfile
├── policies/               # OPA policies
│   ├── rbac.rego          # RBAC policy
│   ├── abac.rego          # ABAC policy
│   └── security.rego      # Security checks
├── monitoring/             # Monitoring configuration
│   ├── prometheus.yml     # Prometheus config
│   ├── alert-rules.yml    # Alert rules
│   ├── alertmanager.yml   # Alertmanager config
│   └── grafana/           # Grafana dashboards
├── keycloak/              # Keycloak configuration
│   └── realm-export.json  # Realm export
├── scripts/               # Utility scripts
│   └── generate-certs.sh  # Certificate generation
├── certs/                 # TLS certificates
├── README.md              # This file
└── README-CN.md           # Chinese documentation
```

## Troubleshooting

### Keycloak Not Starting

```bash
# Check logs
docker-compose logs keycloak

# Ensure PostgreSQL is healthy
docker-compose logs postgres
```

### OPA Policy Errors

```bash
# Check OPA logs
docker-compose logs opa

# Test policy manually
curl -X POST http://localhost:8181/v1/data/rbac/allow \
  -H "Content-Type: application/json" \
  -d '{"input":{"user":{"roles":["admin"]},"request":{"method":"GET","path":"/api/v1/protected/documents"}}}'
```

### Authentication Issues

```bash
# Verify Keycloak is accessible
curl http://localhost:8081/realms/swit/.well-known/openid-configuration

# Check application logs
docker-compose logs app
```

## Development

### Run Locally

```bash
# Install dependencies
go mod download

# Run with embedded OPA
OAUTH2_ENABLED=false OPA_ENABLED=true go run main.go

# Run with all features (requires Keycloak and OPA)
go run main.go
```

### Run Tests

```bash
# Run unit tests
go test -v ./...

# Run with coverage
go test -cover ./...
```

## Related Documentation

- [OAuth2 Integration Guide](../../docs/oauth2-integration-guide.md)
- [OPA Policy Guide](../../docs/opa-policy-guide.md)
- [Security Best Practices](../../docs/security-best-practices.md)
- [Security Configuration Reference](../../docs/security-configuration-reference.md)

## License

This example is part of the Swit framework and is licensed under the BSD-style license.

