# Security Integration Tests

This directory contains the Docker Compose configuration and supporting files for running security integration tests.

## Overview

The security integration test suite validates:
- OAuth2/OIDC complete authentication flows
- OPA policy evaluation
- mTLS handshake and certificate validation
- Middleware integration
- Cross-component security stack

## Prerequisites

- Docker and Docker Compose
- Go 1.23+
- Make (optional, for convenience commands)

## Quick Start

### 1. Start Test Environment

```bash
# Start all services
docker compose -f tests/security/docker-compose.test.yml up -d

# Wait for services to be healthy
docker compose -f tests/security/docker-compose.test.yml ps
```

### 2. Run Integration Tests

```bash
# Run all security integration tests
go test ./pkg/security/... -v -timeout=300s

# Run specific test suites
go test ./pkg/security/oauth2/... -v -run TestIntegration
go test ./pkg/security/opa/... -v -run TestIntegration
go test ./pkg/security/... -v -run TestMTLS
```

### 3. Run Tests with Docker

```bash
# Run tests inside Docker container
docker compose -f tests/security/docker-compose.test.yml --profile test up test-runner
```

### 4. Cleanup

```bash
# Stop and remove all containers
docker compose -f tests/security/docker-compose.test.yml down -v
```

## Services

### OPA (Open Policy Agent)
- **URL**: http://localhost:8181
- **Purpose**: Policy evaluation server for RBAC/ABAC testing
- **Policies**: Located in `./policies/`

### Keycloak
- **URL**: http://localhost:8080
- **Admin Console**: http://localhost:8080/admin
- **Credentials**: admin/admin
- **Test Realm**: swit-test
- **Test Client**: swit-test-client / swit-test-secret

### CFSSL (Certificate Authority)
- **URL**: http://localhost:8888
- **Purpose**: Generate certificates for mTLS testing

## Test Users

| Username | Password | Roles |
|----------|----------|-------|
| admin-user | admin123 | admin, editor, viewer |
| editor-user | editor123 | editor, viewer |
| viewer-user | viewer123 | viewer |

## Environment Variables

When running tests against the Docker environment:

```bash
export OPA_URL=http://localhost:8181
export KEYCLOAK_URL=http://localhost:8080
export KEYCLOAK_REALM=swit-test
export KEYCLOAK_CLIENT_ID=swit-test-client
export KEYCLOAK_CLIENT_SECRET=swit-test-secret
```

## Directory Structure

```
tests/security/
├── docker-compose.test.yml    # Docker Compose configuration
├── README.md                  # This file
├── policies/                  # OPA policies for testing
│   └── authz.rego            # Authorization policy
├── keycloak/                  # Keycloak configuration
│   └── test-realm.json       # Test realm import
├── cfssl-config/             # CFSSL configuration
│   └── config.json           # Certificate profiles
└── certs/                    # Generated certificates (created at runtime)
```

## Troubleshooting

### Services not starting

```bash
# Check service logs
docker compose -f tests/security/docker-compose.test.yml logs opa
docker compose -f tests/security/docker-compose.test.yml logs keycloak
```

### Connection refused errors

Ensure all services are healthy before running tests:

```bash
docker compose -f tests/security/docker-compose.test.yml ps
```

### Keycloak taking too long to start

Keycloak may take 1-2 minutes to fully initialize. The health check is configured with a 60-second start period.

### OPA policy errors

Validate policies before testing:

```bash
docker exec swit-test-opa opa check /policies/
```

## Writing New Tests

When adding new integration tests:

1. Use the `testing.Short()` check to skip in short mode
2. Use environment variables for service URLs
3. Clean up resources after tests
4. Use table-driven tests where applicable

Example:

```go
func TestIntegrationNewFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    opaURL := os.Getenv("OPA_URL")
    if opaURL == "" {
        opaURL = "http://localhost:8181"
    }
    
    // Test implementation...
}
```

