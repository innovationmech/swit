# mTLS (Mutual TLS) Authentication Example

This example demonstrates how to implement mutual TLS (mTLS) authentication in a Swit framework service. mTLS provides strong authentication by requiring both the server and client to present and verify certificates.

## Features

### Security Features

- ✅ **Mutual TLS Authentication** - Both server and client verify each other's certificates
- ✅ **Certificate-Based Identity** - Extract client identity from certificate attributes
- ✅ **Role-Based Access Control** - Restrict endpoints based on certificate attributes
- ✅ **TLS 1.2/1.3 Support** - Modern TLS protocol versions
- ✅ **Secure Cipher Suites** - Only strong cryptographic algorithms

### Implementation Features

- ✅ **HTTP mTLS Server** - Gin-based HTTPS server with client certificate verification
- ✅ **gRPC mTLS Server** - gRPC server with mutual TLS
- ✅ **Certificate Info Extraction** - Parse CN, OU, SAN, and other certificate fields
- ✅ **Admin Authorization** - Example of certificate-based authorization
- ✅ **Certificate Generation Script** - Easy setup with automated certificate generation

## Quick Start

### Prerequisites

- Go 1.23+
- OpenSSL (for certificate generation)
- curl (for testing)

### 1. Generate Certificates

```bash
cd examples/mtls-authentication

# Generate CA, server, and client certificates
./scripts/generate-certs.sh
```

This creates:
- `certs/ca.crt` / `certs/ca.key` - Certificate Authority
- `certs/server.crt` / `certs/server.key` - Server certificate
- `certs/client.crt` / `certs/client.key` - Regular client certificate
- `certs/admin-client.crt` / `certs/admin-client.key` - Admin client certificate

### 2. Start the Server

```bash
cd examples/mtls-authentication
go run main.go
```

The server will start on:
- **HTTPS**: https://localhost:8443
- **gRPC**: localhost:50443

### 3. Test the Endpoints

#### Test with regular client certificate:

```bash
# Public endpoint
curl --cacert certs/ca.crt \
     --cert certs/client.crt \
     --key certs/client.key \
     https://localhost:8443/api/v1/public/info

# Protected endpoint
curl --cacert certs/ca.crt \
     --cert certs/client.crt \
     --key certs/client.key \
     https://localhost:8443/api/v1/protected/profile

# Admin endpoint (will be denied with regular client cert)
curl --cacert certs/ca.crt \
     --cert certs/client.crt \
     --key certs/client.key \
     https://localhost:8443/api/v1/admin/dashboard
```

#### Test with admin client certificate:

```bash
# Admin endpoint (will succeed with admin cert)
curl --cacert certs/ca.crt \
     --cert certs/admin-client.crt \
     --key certs/admin-client.key \
     https://localhost:8443/api/v1/admin/dashboard
```

#### Test without client certificate (will fail):

```bash
# This will fail because mTLS requires client certificate
curl --cacert certs/ca.crt \
     https://localhost:8443/api/v1/public/info
```

## API Endpoints

### Public Endpoints (mTLS required, no identity check)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/public/info` | Service information and client certificate details |
| GET | `/api/v1/public/health` | Health check endpoint |

### Protected Endpoints (mTLS required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/protected/profile` | Client certificate profile details |
| GET | `/api/v1/protected/data` | Protected data access |

### Admin Endpoints (Admin certificate required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/admin/dashboard` | Admin dashboard |
| GET | `/api/v1/admin/certificates` | List all certificates |

## Configuration

### Configuration File (swit.yaml)

```yaml
http:
  enabled: true
  port: "8443"
  tls:
    enabled: true
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"
    ca_files:
      - "certs/ca.crt"
    client_auth: "require_and_verify"  # Full mTLS
    min_version: "TLS1.2"
    max_version: "TLS1.3"
```

### Client Authentication Modes

| Mode | Description |
|------|-------------|
| `none` | No client certificate required |
| `request` | Request client certificate but don't require it |
| `require` | Require any client certificate |
| `verify_if_given` | Verify client certificate if provided |
| `require_and_verify` | Require and verify client certificate (full mTLS) |

### Environment Variables

```bash
# Server configuration
export HTTP_PORT="8443"
export GRPC_PORT="50443"
export GRPC_ENABLED="true"

# TLS configuration
export TLS_CERT_FILE="certs/server.crt"
export TLS_KEY_FILE="certs/server.key"
export TLS_CA_FILE="certs/ca.crt"
export TLS_CLIENT_AUTH="require_and_verify"
```

## Certificate Management

### Regenerate Certificates

```bash
# Clean and regenerate all certificates
./scripts/generate-certs.sh clean
./scripts/generate-certs.sh
```

### Verify Certificates

```bash
# Verify all certificates
./scripts/generate-certs.sh verify
```

### Custom Certificate Configuration

Edit the script variables in `scripts/generate-certs.sh`:

```bash
# CA Configuration
CA_SUBJECT="/C=CN/ST=Beijing/L=Beijing/O=YourOrg/OU=Security/CN=YourOrg-CA"

# Server Configuration
SERVER_SUBJECT="/C=CN/ST=Beijing/L=Beijing/O=YourOrg/OU=Server/CN=your-domain.com"
SERVER_SAN="DNS:your-domain.com,DNS:*.your-domain.com,IP:10.0.0.1"

# Client Configuration
CLIENT_SUBJECT="/C=CN/ST=Beijing/L=Beijing/O=YourOrg/OU=Client/CN=client-name"
```

## Architecture

```
┌─────────────┐         ┌──────────────────┐
│   Client    │◄───────►│  mTLS Server     │
│ (with cert) │  mTLS   │  (Swit Framework)│
└─────────────┘         └──────────────────┘
       │                        │
       │                        │
       ▼                        ▼
┌─────────────┐         ┌──────────────────┐
│ client.crt  │         │ server.crt       │
│ client.key  │         │ server.key       │
└─────────────┘         │ ca.crt (for      │
       │                │ client verify)   │
       │                └──────────────────┘
       ▼
┌─────────────┐
│   ca.crt    │
│ (for server │
│   verify)   │
└─────────────┘
```

### mTLS Handshake Flow

```
1. Client → Server: ClientHello
2. Server → Client: ServerHello + Server Certificate
3. Server → Client: Certificate Request
4. Client → Server: Client Certificate
5. Client → Server: Certificate Verify
6. Both: Verify certificates against CA
7. Connection established with mutual authentication
```

## Security Considerations

### Production Deployment

1. **Certificate Storage**: Store private keys securely (e.g., HSM, Vault)
2. **Certificate Rotation**: Implement automatic certificate rotation
3. **CRL/OCSP**: Implement certificate revocation checking
4. **Audit Logging**: Log all certificate-based authentication events
5. **Network Security**: Use network segmentation for mTLS services

### Certificate Best Practices

1. **Key Size**: Use at least 2048-bit RSA or 256-bit ECDSA keys
2. **Validity Period**: Use short-lived certificates (30-90 days)
3. **SAN Usage**: Always include Subject Alternative Names
4. **Key Usage**: Set appropriate key usage extensions
5. **CA Security**: Protect CA private key with hardware security

### Common Issues

#### Connection Refused
- Ensure server is running on the correct port
- Check firewall rules

#### Certificate Verification Failed
- Verify certificate chain is complete
- Check certificate expiration dates
- Ensure CA certificate is correct

#### Handshake Failure
- Check TLS version compatibility
- Verify cipher suite support
- Ensure certificate key matches

## Testing with gRPC

### Using grpcurl

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List services (with mTLS)
grpcurl -cacert certs/ca.crt \
        -cert certs/client.crt \
        -key certs/client.key \
        localhost:50443 list
```

### Using Go Client

```go
import (
    "crypto/tls"
    "crypto/x509"
    "os"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
)

func createMTLSClient() (*grpc.ClientConn, error) {
    // Load client certificate
    cert, err := tls.LoadX509KeyPair("certs/client.crt", "certs/client.key")
    if err != nil {
        return nil, err
    }
    
    // Load CA certificate
    caCert, err := os.ReadFile("certs/ca.crt")
    if err != nil {
        return nil, err
    }
    caPool := x509.NewCertPool()
    caPool.AppendCertsFromPEM(caCert)
    
    // Create TLS config
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caPool,
    }
    
    // Create gRPC connection
    return grpc.Dial("localhost:50443",
        grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
}
```

## Further Reading

- [TLS 1.3 Specification (RFC 8446)](https://datatracker.ietf.org/doc/html/rfc8446)
- [X.509 Certificate Standard](https://datatracker.ietf.org/doc/html/rfc5280)
- [mTLS Best Practices](https://www.cloudflare.com/learning/access-management/what-is-mutual-tls/)
- [Go crypto/tls Package](https://pkg.go.dev/crypto/tls)

## License

Copyright 2025 Swit. All rights reserved.

