# OAuth2/OIDC Authentication Example

This example demonstrates how to implement OAuth2/OIDC authentication in a Swit framework service. It showcases:

- **Authorization Code Flow** with PKCE (Proof Key for Code Exchange)
- **OIDC Discovery** for automatic endpoint configuration
- **JWT Token Validation** with local and remote introspection
- **Token Refresh** flow
- **Protected Endpoints** with OAuth2 middleware
- **Role-Based Access Control** (RBAC)
- **Keycloak Integration** as OIDC provider

## Features

### Authentication Flows

- ✅ **Authorization Code Flow with PKCE** - Most secure OAuth2 flow for web applications
- ✅ **OIDC Discovery** - Automatic configuration from provider metadata
- ✅ **Token Refresh** - Seamless token renewal without re-authentication
- ✅ **Token Revocation** - Proper logout with token cleanup

### Security Features

- ✅ **JWT Validation** - Local token verification with JWKS
- ✅ **Token Introspection** - Remote token validation with provider
- ✅ **State & Nonce Validation** - CSRF and replay attack protection
- ✅ **PKCE Support** - Enhanced security for public clients
- ✅ **Role-Based Access Control** - Fine-grained authorization

### Integration Features

- ✅ **Keycloak Integration** - Pre-configured realm and client
- ✅ **Middleware Support** - Easy endpoint protection
- ✅ **Optional Authentication** - Mixed public/private endpoints
- ✅ **Docker Compose Setup** - One-command deployment

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.23+ (for local development)
- curl or Postman (for testing)

### 1. Start the Services

```bash
cd examples/oauth2-authentication
docker-compose up -d
```

This will start:
- **Keycloak** on http://localhost:8081 (OIDC Provider)
- **PostgreSQL** on localhost:5432 (Keycloak database)
- **OAuth2 Example Service** on http://localhost:8080

Wait for all services to be healthy (~60 seconds for Keycloak):

```bash
docker-compose ps
```

### 2. Verify Keycloak is Running

Access Keycloak Admin Console:
- URL: http://localhost:8081/admin
- Username: `admin`
- Password: `admin`

The example comes with a pre-configured realm named **swit** that includes:
- Client: `swit-example` (client secret: `swit-example-secret`)
- Two users:
  - `testuser` / `password` (role: user)
  - `admin` / `admin` (roles: user, admin)

### 3. Test the Authentication Flow

#### Step 1: Get Service Information

```bash
curl http://localhost:8080/api/v1/public/info
```

#### Step 2: Initiate Login

```bash
curl http://localhost:8080/api/v1/public/login
```

This returns an authorization URL. Copy the `authorization_url` and open it in your browser.

#### Step 3: Login with Keycloak

In the browser:
1. Enter credentials: `testuser` / `password`
2. You'll be redirected to the callback URL
3. Copy the JSON response containing the `access_token`

#### Step 4: Access Protected Endpoint

Use the access token from step 3:

```bash
export ACCESS_TOKEN="your-access-token-here"

curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/api/v1/protected/profile
```

### 4. Test Token Refresh

```bash
export REFRESH_TOKEN="your-refresh-token-here"

curl -X POST http://localhost:8080/api/v1/public/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}"
```

### 5. Test Role-Based Access

Try accessing the admin endpoint with a regular user token:

```bash
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/api/v1/admin/dashboard
```

This should return 403 Forbidden. Now login as admin and try again.

## API Endpoints

### Public Endpoints (No Authentication Required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/public/info` | Service information |
| GET | `/api/v1/public/login` | Initiate OAuth2 login |
| GET | `/api/v1/public/callback` | OAuth2 callback handler |
| POST | `/api/v1/public/refresh` | Refresh access token |
| POST | `/api/v1/public/logout` | Revoke token and logout |

### Protected Endpoints (Authentication Required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/protected/profile` | User profile |
| GET | `/api/v1/protected/data` | Protected data |

### Optional Auth Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/optional/content` | Content for both authenticated and anonymous users |

### Admin Endpoints (Admin Role Required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/admin/dashboard` | Admin dashboard |

## Configuration

### Environment Variables

Override configuration using environment variables:

```bash
# OAuth2 Configuration
export OAUTH2_PROVIDER="keycloak"
export OAUTH2_CLIENT_ID="swit-example"
export OAUTH2_CLIENT_SECRET="swit-example-secret"
export OAUTH2_ISSUER_URL="http://localhost:8081/realms/swit"
export OAUTH2_REDIRECT_URL="http://localhost:8080/api/v1/public/callback"

# Server Configuration
export HTTP_PORT="8080"
```

### Configuration File

Edit `swit.yaml` to customize the service:

```yaml
oauth2:
  enabled: true
  provider: "keycloak"
  client_id: "swit-example"
  client_secret: "your-secret"
  issuer_url: "http://localhost:8081/realms/swit"
  use_discovery: true
  scopes:
    - "openid"
    - "profile"
    - "email"
```

## Running Locally (Without Docker)

### 1. Start Keycloak

```bash
docker-compose up -d keycloak postgres
```

### 2. Run the Example Service

```bash
cd examples/oauth2-authentication

# Set environment variables
export OAUTH2_CLIENT_SECRET="swit-example-secret"

# Run the service
go run main.go
```

## Testing with Different Providers

### Using Auth0

1. Create an Auth0 account and application
2. Update `swit.yaml`:

```yaml
oauth2:
  provider: "auth0"
  client_id: "your-auth0-client-id"
  client_secret: "your-auth0-client-secret"
  issuer_url: "https://your-tenant.auth0.com/"
  redirect_url: "http://localhost:8080/api/v1/public/callback"
```

### Using Google

1. Create Google OAuth2 credentials
2. Update `swit.yaml`:

```yaml
oauth2:
  provider: "google"
  client_id: "your-google-client-id.apps.googleusercontent.com"
  client_secret: "your-google-client-secret"
  issuer_url: "https://accounts.google.com"
  redirect_url: "http://localhost:8080/api/v1/public/callback"
```

## Integration Tests

Run the OAuth2 integration tests:

```bash
# Run integration tests
cd ../../
go test ./pkg/security/oauth2 -v -run Integration

# Run benchmarks
go test ./pkg/security/oauth2 -bench=. -benchmem
```

## Security Considerations

### Production Deployment

1. **Use HTTPS**: Always use TLS in production
2. **Secure Secrets**: Store client secrets in secret management systems
3. **Token Storage**: Use secure, HTTP-only cookies for tokens
4. **CORS Configuration**: Restrict allowed origins
5. **Rate Limiting**: Implement rate limiting on authentication endpoints
6. **Audit Logging**: Log all authentication events

### Token Management

1. **Token Expiration**: Use short-lived access tokens (5-15 minutes)
2. **Refresh Token Rotation**: Rotate refresh tokens on each use
3. **Token Revocation**: Implement proper logout with token revocation
4. **Token Caching**: Cache validated tokens to reduce provider load

## Troubleshooting

### Keycloak Not Starting

Check logs:
```bash
docker-compose logs keycloak
```

Wait for "Keycloak started" message.

### Invalid Redirect URI

Ensure the redirect URI in Keycloak matches exactly:
1. Go to http://localhost:8081/admin
2. Navigate to Clients → swit-example
3. Check "Valid Redirect URIs" includes `http://localhost:8080/*`

### Token Validation Fails

1. Check issuer URL matches between config and Keycloak
2. Verify JWKS endpoint is accessible
3. Check system clock is synchronized

### Connection Refused to Keycloak

If running the example service locally (not in Docker), use:
```bash
export OAUTH2_ISSUER_URL="http://localhost:8081/realms/swit"
```

Instead of the Docker internal hostname.

## Architecture

```
┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│   Browser   │◄───────►│  OAuth2 Svc  │◄───────►│  Keycloak    │
│             │         │              │         │  (OIDC)      │
└─────────────┘         └──────────────┘         └──────────────┘
                               │
                               ▼
                        ┌──────────────┐
                        │ JWT Validator│
                        │  (with JWKS) │
                        └──────────────┘
```

### Flow Diagram

```
1. User → GET /login
2. Service → Redirect to Keycloak
3. User → Login at Keycloak
4. Keycloak → Redirect to /callback with code
5. Service → Exchange code for tokens
6. Service → Return tokens to user
7. User → Access protected endpoint with access_token
8. Service → Validate JWT locally or via introspection
9. Service → Return protected resource
```

## Further Reading

- [OAuth 2.0 Specification (RFC 6749)](https://datatracker.ietf.org/doc/html/rfc6749)
- [OIDC Core Specification](https://openid.net/specs/openid-connect-core-1_0.html)
- [PKCE Specification (RFC 7636)](https://datatracker.ietf.org/doc/html/rfc7636)
- [JWT Specification (RFC 7519)](https://datatracker.ietf.org/doc/html/rfc7519)
- [Keycloak Documentation](https://www.keycloak.org/documentation)

## License

Copyright 2025 Swit. All rights reserved.

