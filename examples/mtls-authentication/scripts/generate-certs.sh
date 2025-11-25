#!/bin/bash

# mTLS Certificate Generation Script
# This script generates a complete set of certificates for mTLS demonstration:
# - CA certificate (root certificate authority)
# - Server certificate (signed by CA)
# - Client certificate (signed by CA)

set -e

# Configuration
CERT_DIR="$(dirname "$0")/../certs"
DAYS_VALID=365
KEY_SIZE=4096

# CA Configuration
CA_SUBJECT="/C=CN/ST=Beijing/L=Beijing/O=Swit/OU=Security/CN=Swit-CA"

# Server Configuration
SERVER_SUBJECT="/C=CN/ST=Beijing/L=Beijing/O=Swit/OU=Server/CN=localhost"
SERVER_SAN="DNS:localhost,DNS:*.localhost,IP:127.0.0.1,IP:::1"

# Client Configuration
CLIENT_SUBJECT="/C=CN/ST=Beijing/L=Beijing/O=Swit/OU=Client/CN=swit-client"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print functions
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Create certificate directory
create_cert_dir() {
    info "Creating certificate directory: $CERT_DIR"
    mkdir -p "$CERT_DIR"
}

# Generate CA certificate
generate_ca() {
    info "Generating CA certificate..."
    
    # Generate CA private key
    openssl genrsa -out "$CERT_DIR/ca.key" $KEY_SIZE 2>/dev/null
    
    # Generate CA certificate
    openssl req -new -x509 -days $DAYS_VALID \
        -key "$CERT_DIR/ca.key" \
        -out "$CERT_DIR/ca.crt" \
        -subj "$CA_SUBJECT" \
        -addext "basicConstraints=critical,CA:TRUE" \
        -addext "keyUsage=critical,keyCertSign,cRLSign"
    
    success "CA certificate generated: $CERT_DIR/ca.crt"
}

# Generate server certificate
generate_server() {
    info "Generating server certificate..."
    
    # Generate server private key
    openssl genrsa -out "$CERT_DIR/server.key" $KEY_SIZE 2>/dev/null
    
    # Generate server CSR
    openssl req -new \
        -key "$CERT_DIR/server.key" \
        -out "$CERT_DIR/server.csr" \
        -subj "$SERVER_SUBJECT"
    
    # Create server extensions file
    cat > "$CERT_DIR/server.ext" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage=critical,digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
subjectAltName=$SERVER_SAN
EOF
    
    # Sign server certificate with CA
    openssl x509 -req -days $DAYS_VALID \
        -in "$CERT_DIR/server.csr" \
        -CA "$CERT_DIR/ca.crt" \
        -CAkey "$CERT_DIR/ca.key" \
        -CAcreateserial \
        -out "$CERT_DIR/server.crt" \
        -extfile "$CERT_DIR/server.ext"
    
    # Clean up CSR and ext files
    rm -f "$CERT_DIR/server.csr" "$CERT_DIR/server.ext"
    
    success "Server certificate generated: $CERT_DIR/server.crt"
}

# Generate client certificate
generate_client() {
    info "Generating client certificate..."
    
    # Generate client private key
    openssl genrsa -out "$CERT_DIR/client.key" $KEY_SIZE 2>/dev/null
    
    # Generate client CSR
    openssl req -new \
        -key "$CERT_DIR/client.key" \
        -out "$CERT_DIR/client.csr" \
        -subj "$CLIENT_SUBJECT"
    
    # Create client extensions file
    cat > "$CERT_DIR/client.ext" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage=critical,digitalSignature,keyEncipherment
extendedKeyUsage=clientAuth
EOF
    
    # Sign client certificate with CA
    openssl x509 -req -days $DAYS_VALID \
        -in "$CERT_DIR/client.csr" \
        -CA "$CERT_DIR/ca.crt" \
        -CAkey "$CERT_DIR/ca.key" \
        -CAcreateserial \
        -out "$CERT_DIR/client.crt" \
        -extfile "$CERT_DIR/client.ext"
    
    # Clean up CSR and ext files
    rm -f "$CERT_DIR/client.csr" "$CERT_DIR/client.ext"
    
    success "Client certificate generated: $CERT_DIR/client.crt"
}

# Generate additional client certificate (for testing multiple clients)
generate_additional_client() {
    local client_name=$1
    local client_cn=$2
    
    info "Generating additional client certificate: $client_name..."
    
    # Generate client private key
    openssl genrsa -out "$CERT_DIR/${client_name}.key" $KEY_SIZE 2>/dev/null
    
    # Generate client CSR
    openssl req -new \
        -key "$CERT_DIR/${client_name}.key" \
        -out "$CERT_DIR/${client_name}.csr" \
        -subj "/C=CN/ST=Beijing/L=Beijing/O=Swit/OU=Client/CN=$client_cn"
    
    # Create client extensions file
    cat > "$CERT_DIR/${client_name}.ext" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage=critical,digitalSignature,keyEncipherment
extendedKeyUsage=clientAuth
EOF
    
    # Sign client certificate with CA
    openssl x509 -req -days $DAYS_VALID \
        -in "$CERT_DIR/${client_name}.csr" \
        -CA "$CERT_DIR/ca.crt" \
        -CAkey "$CERT_DIR/ca.key" \
        -CAcreateserial \
        -out "$CERT_DIR/${client_name}.crt" \
        -extfile "$CERT_DIR/${client_name}.ext"
    
    # Clean up CSR and ext files
    rm -f "$CERT_DIR/${client_name}.csr" "$CERT_DIR/${client_name}.ext"
    
    success "Additional client certificate generated: $CERT_DIR/${client_name}.crt"
}

# Verify certificates
verify_certificates() {
    info "Verifying certificates..."
    
    # Verify server certificate
    if openssl verify -CAfile "$CERT_DIR/ca.crt" "$CERT_DIR/server.crt" > /dev/null 2>&1; then
        success "Server certificate verification: PASSED"
    else
        error "Server certificate verification: FAILED"
    fi
    
    # Verify client certificate
    if openssl verify -CAfile "$CERT_DIR/ca.crt" "$CERT_DIR/client.crt" > /dev/null 2>&1; then
        success "Client certificate verification: PASSED"
    else
        error "Client certificate verification: FAILED"
    fi
    
    # Verify admin client certificate if exists
    if [ -f "$CERT_DIR/admin-client.crt" ]; then
        if openssl verify -CAfile "$CERT_DIR/ca.crt" "$CERT_DIR/admin-client.crt" > /dev/null 2>&1; then
            success "Admin client certificate verification: PASSED"
        else
            error "Admin client certificate verification: FAILED"
        fi
    fi
}

# Display certificate information
display_cert_info() {
    info "Certificate Information:"
    echo ""
    
    echo "=== CA Certificate ==="
    openssl x509 -in "$CERT_DIR/ca.crt" -noout -subject -issuer -dates
    echo ""
    
    echo "=== Server Certificate ==="
    openssl x509 -in "$CERT_DIR/server.crt" -noout -subject -issuer -dates
    echo "Subject Alternative Names:"
    openssl x509 -in "$CERT_DIR/server.crt" -noout -ext subjectAltName 2>/dev/null || echo "  None"
    echo ""
    
    echo "=== Client Certificate ==="
    openssl x509 -in "$CERT_DIR/client.crt" -noout -subject -issuer -dates
    echo ""
    
    if [ -f "$CERT_DIR/admin-client.crt" ]; then
        echo "=== Admin Client Certificate ==="
        openssl x509 -in "$CERT_DIR/admin-client.crt" -noout -subject -issuer -dates
        echo ""
    fi
}

# Print usage information
print_usage() {
    echo ""
    echo "=== Certificate Files Generated ==="
    echo "CA Certificate:      $CERT_DIR/ca.crt"
    echo "CA Private Key:      $CERT_DIR/ca.key"
    echo "Server Certificate:  $CERT_DIR/server.crt"
    echo "Server Private Key:  $CERT_DIR/server.key"
    echo "Client Certificate:  $CERT_DIR/client.crt"
    echo "Client Private Key:  $CERT_DIR/client.key"
    if [ -f "$CERT_DIR/admin-client.crt" ]; then
        echo "Admin Certificate:   $CERT_DIR/admin-client.crt"
        echo "Admin Private Key:   $CERT_DIR/admin-client.key"
    fi
    echo ""
    echo "=== Usage Examples ==="
    echo ""
    echo "1. Start the mTLS server:"
    echo "   cd examples/mtls-authentication"
    echo "   go run main.go"
    echo ""
    echo "2. Test with curl (mTLS client):"
    echo "   curl --cacert certs/ca.crt \\"
    echo "        --cert certs/client.crt \\"
    echo "        --key certs/client.key \\"
    echo "        https://localhost:8443/api/v1/public/info"
    echo ""
    echo "3. Test without client certificate (should fail with require_and_verify):"
    echo "   curl --cacert certs/ca.crt https://localhost:8443/api/v1/public/info"
    echo ""
}

# Clean up generated certificates
clean() {
    warn "Cleaning up all generated certificates..."
    rm -f "$CERT_DIR"/*.crt "$CERT_DIR"/*.key "$CERT_DIR"/*.srl
    success "Cleanup complete"
}

# Main function
main() {
    case "${1:-}" in
        clean)
            clean
            ;;
        verify)
            verify_certificates
            display_cert_info
            ;;
        *)
            echo "=============================================="
            echo "  mTLS Certificate Generation Script"
            echo "=============================================="
            echo ""
            
            create_cert_dir
            generate_ca
            generate_server
            generate_client
            generate_additional_client "admin-client" "swit-admin"
            verify_certificates
            display_cert_info
            print_usage
            
            success "All certificates generated successfully!"
            ;;
    esac
}

# Run main function
main "$@"

