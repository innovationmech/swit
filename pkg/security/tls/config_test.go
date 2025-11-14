// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testCertData holds test certificate and key data
type testCertData struct {
	certFile string
	keyFile  string
	caFile   string
}

// setupTestCerts creates temporary test certificates for testing
func setupTestCerts(t *testing.T) *testCertData {
	t.Helper()

	tmpDir := t.TempDir()

	// Generate CA certificate
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
			CommonName:   "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create CA certificate: %v", err)
	}

	// Write CA certificate
	caFile := filepath.Join(tmpDir, "ca.pem")
	caOut, err := os.Create(caFile)
	if err != nil {
		t.Fatalf("failed to create CA file: %v", err)
	}
	if err := pem.Encode(caOut, &pem.Block{Type: "CERTIFICATE", Bytes: caCertBytes}); err != nil {
		t.Fatalf("failed to encode CA certificate: %v", err)
	}
	caOut.Close()

	// Generate server certificate
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate server key: %v", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Server"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	serverCertBytes, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create server certificate: %v", err)
	}

	// Write server certificate
	certFile := filepath.Join(tmpDir, "server.pem")
	certOut, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("failed to create cert file: %v", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: serverCertBytes}); err != nil {
		t.Fatalf("failed to encode server certificate: %v", err)
	}
	certOut.Close()

	// Write server key
	keyFile := filepath.Join(tmpDir, "server-key.pem")
	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("failed to create key file: %v", err)
	}
	keyBytes := x509.MarshalPKCS1PrivateKey(serverKey)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes}); err != nil {
		t.Fatalf("failed to encode server key: %v", err)
	}
	keyOut.Close()

	return &testCertData{
		certFile: certFile,
		keyFile:  keyFile,
		caFile:   caFile,
	}
}

func TestValidateTLSConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *TLSConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "tls config cannot be nil",
		},
		{
			name: "disabled config",
			config: &TLSConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "cert without key",
			config: &TLSConfig{
				Enabled:  true,
				CertFile: "/path/to/cert.pem",
			},
			wantErr: true,
			errMsg:  "key_file must be specified",
		},
		{
			name: "key without cert",
			config: &TLSConfig{
				Enabled: true,
				KeyFile: "/path/to/key.pem",
			},
			wantErr: true,
			errMsg:  "cert_file must be specified",
		},
		{
			name: "invalid min version",
			config: &TLSConfig{
				Enabled:    true,
				MinVersion: "TLS1.4",
			},
			wantErr: true,
			errMsg:  "invalid min_version",
		},
		{
			name: "invalid max version",
			config: &TLSConfig{
				Enabled:    true,
				MaxVersion: "TLS0.9",
			},
			wantErr: true,
			errMsg:  "invalid max_version",
		},
		{
			name: "invalid client auth",
			config: &TLSConfig{
				Enabled:    true,
				ClientAuth: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid client_auth",
		},
		{
			name: "invalid cipher suite",
			config: &TLSConfig{
				Enabled:      true,
				CipherSuites: []string{"INVALID_CIPHER"},
			},
			wantErr: true,
			errMsg:  "invalid cipher suite",
		},
		{
			name: "invalid curve",
			config: &TLSConfig{
				Enabled:          true,
				CurvePreferences: []string{"INVALID_CURVE"},
			},
			wantErr: true,
			errMsg:  "invalid curve preference",
		},
		{
			name: "invalid renegotiation",
			config: &TLSConfig{
				Enabled:       true,
				Renegotiation: "always",
			},
			wantErr: true,
			errMsg:  "invalid renegotiation",
		},
		{
			name: "valid renegotiation once",
			config: &TLSConfig{
				Enabled:       true,
				Renegotiation: "once",
			},
			wantErr: false,
		},
		{
			name: "valid renegotiation freely",
			config: &TLSConfig{
				Enabled:       true,
				Renegotiation: "freely",
			},
			wantErr: false,
		},
		{
			name: "valid config",
			config: &TLSConfig{
				Enabled:    true,
				MinVersion: "TLS1.2",
				MaxVersion: "TLS1.3",
				ClientAuth: "none",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTLSConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateTLSConfig() expected error but got nil")
					return
				}
				if tt.errMsg != "" && err.Error() == "" {
					t.Errorf("ValidateTLSConfig() error message is empty")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTLSConfig() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNewTLSConfig(t *testing.T) {
	certs := setupTestCerts(t)

	tests := []struct {
		name    string
		config  *TLSConfig
		wantNil bool
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantNil: true,
			wantErr: false,
		},
		{
			name: "disabled config",
			config: &TLSConfig{
				Enabled: false,
			},
			wantNil: true,
			wantErr: false,
		},
		{
			name: "basic config",
			config: &TLSConfig{
				Enabled: true,
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "config with certificate",
			config: &TLSConfig{
				Enabled:  true,
				CertFile: certs.certFile,
				KeyFile:  certs.keyFile,
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "config with CA",
			config: &TLSConfig{
				Enabled: true,
				CAFiles: []string{certs.caFile},
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "config with versions",
			config: &TLSConfig{
				Enabled:    true,
				MinVersion: "TLS1.2",
				MaxVersion: "TLS1.3",
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "config with cipher suites",
			config: &TLSConfig{
				Enabled: true,
				CipherSuites: []string{
					"TLS_AES_256_GCM_SHA384",
					"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
				},
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "config with curves",
			config: &TLSConfig{
				Enabled:          true,
				CurvePreferences: []string{"X25519", "P256"},
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "config with client auth",
			config: &TLSConfig{
				Enabled:    true,
				ClientAuth: "require_and_verify",
				CAFiles:    []string{certs.caFile},
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "config with all options",
			config: &TLSConfig{
				Enabled:                  true,
				CertFile:                 certs.certFile,
				KeyFile:                  certs.keyFile,
				CAFiles:                  []string{certs.caFile},
				MinVersion:               "TLS1.2",
				MaxVersion:               "TLS1.3",
				ClientAuth:               "none",
				ServerName:               "localhost",
				PreferServerCipherSuites: true,
				SessionTicketsDisabled:   false,
				Renegotiation:            "never",
				NextProtos:               []string{"h2", "http/1.1"},
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "invalid cert file",
			config: &TLSConfig{
				Enabled:  true,
				CertFile: "/nonexistent/cert.pem",
				KeyFile:  "/nonexistent/key.pem",
			},
			wantNil: true,
			wantErr: true,
		},
		{
			name: "invalid CA file",
			config: &TLSConfig{
				Enabled: true,
				CAFiles: []string{"/nonexistent/ca.pem"},
			},
			wantNil: true,
			wantErr: true,
		},
		{
			name: "max version less than min version",
			config: &TLSConfig{
				Enabled:    true,
				MinVersion: "TLS1.3",
				MaxVersion: "TLS1.2",
			},
			wantNil: true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig, err := NewTLSConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewTLSConfig() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewTLSConfig() unexpected error: %v", err)
				return
			}

			if tt.wantNil {
				if tlsConfig != nil {
					t.Errorf("NewTLSConfig() expected nil but got %v", tlsConfig)
				}
			} else {
				if tlsConfig == nil {
					t.Errorf("NewTLSConfig() expected non-nil but got nil")
				}
			}
		})
	}
}

func TestDefaultTLSConfig(t *testing.T) {
	config := DefaultTLSConfig()

	if config == nil {
		t.Fatal("DefaultTLSConfig() returned nil")
	}

	if !config.Enabled {
		t.Error("DefaultTLSConfig() Enabled should be true")
	}

	if config.MinVersion != "TLS1.2" {
		t.Errorf("DefaultTLSConfig() MinVersion = %s, want TLS1.2", config.MinVersion)
	}

	if config.MaxVersion != "TLS1.3" {
		t.Errorf("DefaultTLSConfig() MaxVersion = %s, want TLS1.3", config.MaxVersion)
	}

	if config.ClientAuth != "none" {
		t.Errorf("DefaultTLSConfig() ClientAuth = %s, want none", config.ClientAuth)
	}

	if !config.PreferServerCipherSuites {
		t.Error("DefaultTLSConfig() PreferServerCipherSuites should be true")
	}

	if config.Renegotiation != "never" {
		t.Errorf("DefaultTLSConfig() Renegotiation = %s, want never", config.Renegotiation)
	}

	if len(config.CipherSuites) == 0 {
		t.Error("DefaultTLSConfig() CipherSuites should not be empty")
	}

	if len(config.CurvePreferences) == 0 {
		t.Error("DefaultTLSConfig() CurvePreferences should not be empty")
	}

	if len(config.NextProtos) == 0 {
		t.Error("DefaultTLSConfig() NextProtos should not be empty")
	}
}

func TestParseTLSVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    uint16
	}{
		{"TLS 1.0", "TLS1.0", tls.VersionTLS10},
		{"TLS 1.1", "TLS1.1", tls.VersionTLS11},
		{"TLS 1.2", "TLS1.2", tls.VersionTLS12},
		{"TLS 1.3", "TLS1.3", tls.VersionTLS13},
		{"Unknown", "unknown", tls.VersionTLS12}, // defaults to TLS 1.2
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTLSVersion(tt.version)
			if got != tt.want {
				t.Errorf("parseTLSVersion(%s) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestParseClientAuthMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want tls.ClientAuthType
	}{
		{"none", "none", tls.NoClientCert},
		{"request", "request", tls.RequestClientCert},
		{"require", "require", tls.RequireAnyClientCert},
		{"verify_if_given", "verify_if_given", tls.VerifyClientCertIfGiven},
		{"require_and_verify", "require_and_verify", tls.RequireAndVerifyClientCert},
		{"unknown", "unknown", tls.NoClientCert}, // defaults to none
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseClientAuthMode(tt.mode)
			if got != tt.want {
				t.Errorf("parseClientAuthMode(%s) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}

func TestParseRenegotiation(t *testing.T) {
	tests := []struct {
		name   string
		renego string
		want   tls.RenegotiationSupport
	}{
		{"never", "never", tls.RenegotiateNever},
		{"once", "once", tls.RenegotiateOnceAsClient},
		{"freely", "freely", tls.RenegotiateFreelyAsClient},
		{"unknown", "unknown", tls.RenegotiateNever}, // defaults to never
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRenegotiation(tt.renego)
			if got != tt.want {
				t.Errorf("parseRenegotiation(%s) = %v, want %v", tt.renego, got, tt.want)
			}
		})
	}
}

func TestParseCipherSuites(t *testing.T) {
	tests := []struct {
		name    string
		suites  []string
		wantErr bool
	}{
		{
			name: "valid TLS 1.3 suites",
			suites: []string{
				"TLS_AES_256_GCM_SHA384",
				"TLS_CHACHA20_POLY1305_SHA256",
			},
			wantErr: false,
		},
		{
			name: "valid TLS 1.2 suites",
			suites: []string{
				"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
				"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
			},
			wantErr: false,
		},
		{
			name:    "invalid suite",
			suites:  []string{"INVALID_CIPHER_SUITE"},
			wantErr: true,
		},
		{
			name: "mixed valid and invalid",
			suites: []string{
				"TLS_AES_256_GCM_SHA384",
				"INVALID_CIPHER",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCipherSuites(tt.suites)
			if tt.wantErr {
				if err == nil {
					t.Error("parseCipherSuites() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("parseCipherSuites() unexpected error: %v", err)
				}
				if len(result) != len(tt.suites) {
					t.Errorf("parseCipherSuites() returned %d suites, want %d", len(result), len(tt.suites))
				}
			}
		})
	}
}

func TestParseCurvePreferences(t *testing.T) {
	tests := []struct {
		name    string
		curves  []string
		wantErr bool
	}{
		{
			name:    "valid curves",
			curves:  []string{"X25519", "P256", "P384"},
			wantErr: false,
		},
		{
			name:    "invalid curve",
			curves:  []string{"INVALID_CURVE"},
			wantErr: true,
		},
		{
			name: "mixed valid and invalid",
			curves: []string{
				"X25519",
				"INVALID_CURVE",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCurvePreferences(tt.curves)
			if tt.wantErr {
				if err == nil {
					t.Error("parseCurvePreferences() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("parseCurvePreferences() unexpected error: %v", err)
				}
				if len(result) != len(tt.curves) {
					t.Errorf("parseCurvePreferences() returned %d curves, want %d", len(result), len(tt.curves))
				}
			}
		})
	}
}

func TestLoadCAPool(t *testing.T) {
	certs := setupTestCerts(t)

	tests := []struct {
		name    string
		caFiles []string
		wantErr bool
	}{
		{
			name:    "no files",
			caFiles: []string{},
			wantErr: true,
		},
		{
			name:    "valid CA file",
			caFiles: []string{certs.caFile},
			wantErr: false,
		},
		{
			name:    "invalid CA file",
			caFiles: []string{"/nonexistent/ca.pem"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := LoadCAPool(tt.caFiles...)
			if tt.wantErr {
				if err == nil {
					t.Error("LoadCAPool() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("LoadCAPool() unexpected error: %v", err)
				}
				if pool == nil {
					t.Error("LoadCAPool() returned nil pool")
				}
			}
		})
	}
}

func TestIsValidTLSVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"TLS 1.0", "TLS1.0", true},
		{"TLS 1.1", "TLS1.1", true},
		{"TLS 1.2", "TLS1.2", true},
		{"TLS 1.3", "TLS1.3", true},
		{"Invalid", "TLS1.4", false},
		{"Invalid", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidTLSVersion(tt.version)
			if got != tt.want {
				t.Errorf("isValidTLSVersion(%s) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestIsValidClientAuthMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want bool
	}{
		{"none", "none", true},
		{"request", "request", true},
		{"require", "require", true},
		{"verify_if_given", "verify_if_given", true},
		{"require_and_verify", "require_and_verify", true},
		{"invalid", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidClientAuthMode(tt.mode)
			if got != tt.want {
				t.Errorf("isValidClientAuthMode(%s) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}

func TestIsValidCipherSuite(t *testing.T) {
	tests := []struct {
		name  string
		suite string
		want  bool
	}{
		{"TLS 1.3 suite", "TLS_AES_256_GCM_SHA384", true},
		{"TLS 1.2 suite", "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", true},
		{"Invalid", "INVALID_CIPHER", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidCipherSuite(tt.suite)
			if got != tt.want {
				t.Errorf("isValidCipherSuite(%s) = %v, want %v", tt.suite, got, tt.want)
			}
		})
	}
}

func TestIsValidCurve(t *testing.T) {
	tests := []struct {
		name  string
		curve string
		want  bool
	}{
		{"X25519", "X25519", true},
		{"P256", "P256", true},
		{"P384", "P384", true},
		{"P521", "P521", true},
		{"Invalid", "INVALID_CURVE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidCurve(tt.curve)
			if got != tt.want {
				t.Errorf("isValidCurve(%s) = %v, want %v", tt.curve, got, tt.want)
			}
		})
	}
}

func TestIsValidRenegotiation(t *testing.T) {
	tests := []struct {
		name   string
		renego string
		want   bool
	}{
		{"never", "never", true},
		{"once", "once", true},
		{"freely", "freely", true},
		{"invalid", "always", false},
		{"invalid once_as_client", "once_as_client", false},
		{"invalid freely_as_client", "freely_as_client", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidRenegotiation(tt.renego)
			if got != tt.want {
				t.Errorf("isValidRenegotiation(%s) = %v, want %v", tt.renego, got, tt.want)
			}
		})
	}
}
