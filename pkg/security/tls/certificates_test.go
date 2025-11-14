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
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCertificate(t *testing.T) {
	certs := setupTestCerts(t)

	tests := []struct {
		name     string
		certFile string
		keyFile  string
		wantErr  bool
	}{
		{
			name:     "valid certificate and key",
			certFile: certs.certFile,
			keyFile:  certs.keyFile,
			wantErr:  false,
		},
		{
			name:     "empty cert file",
			certFile: "",
			keyFile:  certs.keyFile,
			wantErr:  true,
		},
		{
			name:     "empty key file",
			certFile: certs.certFile,
			keyFile:  "",
			wantErr:  true,
		},
		{
			name:     "nonexistent cert file",
			certFile: "/nonexistent/cert.pem",
			keyFile:  certs.keyFile,
			wantErr:  true,
		},
		{
			name:     "nonexistent key file",
			certFile: certs.certFile,
			keyFile:  "/nonexistent/key.pem",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert, err := LoadCertificate(tt.certFile, tt.keyFile)
			if tt.wantErr {
				if err == nil {
					t.Error("LoadCertificate() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("LoadCertificate() unexpected error: %v", err)
				}
				if len(cert.Certificate) == 0 {
					t.Error("LoadCertificate() returned empty certificate")
				}
				if cert.Leaf == nil {
					t.Error("LoadCertificate() did not parse certificate leaf")
				}
			}
		})
	}
}

func TestLoadCertificatesFromFile(t *testing.T) {
	certs := setupTestCerts(t)

	tests := []struct {
		name    string
		caFile  string
		wantErr bool
		wantLen int
	}{
		{
			name:    "valid CA file",
			caFile:  certs.caFile,
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "empty path",
			caFile:  "",
			wantErr: true,
		},
		{
			name:    "nonexistent file",
			caFile:  "/nonexistent/ca.pem",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certs, err := loadCertificatesFromFile(tt.caFile)
			if tt.wantErr {
				if err == nil {
					t.Error("loadCertificatesFromFile() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("loadCertificatesFromFile() unexpected error: %v", err)
				}
				if len(certs) != tt.wantLen {
					t.Errorf("loadCertificatesFromFile() returned %d certificates, want %d", len(certs), tt.wantLen)
				}
			}
		})
	}
}

func TestLoadCertificateChain(t *testing.T) {
	certs := setupTestCerts(t)

	tests := []struct {
		name     string
		certFile string
		wantErr  bool
	}{
		{
			name:     "valid certificate file",
			certFile: certs.certFile,
			wantErr:  false,
		},
		{
			name:     "valid CA file",
			certFile: certs.caFile,
			wantErr:  false,
		},
		{
			name:     "nonexistent file",
			certFile: "/nonexistent/cert.pem",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, err := LoadCertificateChain(tt.certFile)
			if tt.wantErr {
				if err == nil {
					t.Error("LoadCertificateChain() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("LoadCertificateChain() unexpected error: %v", err)
				}
				if len(chain) == 0 {
					t.Error("LoadCertificateChain() returned empty chain")
				}
			}
		})
	}
}

func TestValidateCertificate(t *testing.T) {
	certs := setupTestCerts(t)

	// Load CA pool
	caPool, err := LoadCAPool(certs.caFile)
	if err != nil {
		t.Fatalf("failed to load CA pool: %v", err)
	}

	// Load server certificate
	serverCert, err := LoadCertificate(certs.certFile, certs.keyFile)
	if err != nil {
		t.Fatalf("failed to load server certificate: %v", err)
	}

	tests := []struct {
		name    string
		cert    *x509.Certificate
		caPool  *x509.CertPool
		wantErr bool
	}{
		{
			name:    "valid certificate",
			cert:    serverCert.Leaf,
			caPool:  caPool,
			wantErr: false,
		},
		{
			name:    "nil certificate",
			cert:    nil,
			caPool:  caPool,
			wantErr: true,
		},
		{
			name:    "nil CA pool",
			cert:    serverCert.Leaf,
			caPool:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCertificate(tt.cert, tt.caPool)
			if tt.wantErr {
				if err == nil {
					t.Error("ValidateCertificate() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateCertificate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetSystemCAPool(t *testing.T) {
	pool, err := GetSystemCAPool()
	if err != nil {
		// On some systems, this might fail
		t.Skipf("GetSystemCAPool() failed: %v (this is expected on some systems)", err)
	}

	if pool == nil {
		t.Error("GetSystemCAPool() returned nil pool")
	}
}

func TestLoadCAPoolWithSystemCerts(t *testing.T) {
	certs := setupTestCerts(t)

	tests := []struct {
		name    string
		caFiles []string
		wantErr bool
	}{
		{
			name:    "valid CA file",
			caFiles: []string{certs.caFile},
			wantErr: false,
		},
		{
			name:    "multiple CA files",
			caFiles: []string{certs.caFile, certs.caFile}, // same file twice, but should work
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
			pool, err := LoadCAPoolWithSystemCerts(tt.caFiles...)
			if tt.wantErr {
				if err == nil {
					t.Error("LoadCAPoolWithSystemCerts() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("LoadCAPoolWithSystemCerts() unexpected error: %v", err)
				}
				if pool == nil {
					t.Error("LoadCAPoolWithSystemCerts() returned nil pool")
				}
			}
		})
	}
}

func TestVerifyCertificateFile(t *testing.T) {
	certs := setupTestCerts(t)

	tests := []struct {
		name     string
		certFile string
		wantErr  bool
	}{
		{
			name:     "valid certificate file",
			certFile: certs.certFile,
			wantErr:  false,
		},
		{
			name:     "valid CA file",
			certFile: certs.caFile,
			wantErr:  false,
		},
		{
			name:     "nonexistent file",
			certFile: "/nonexistent/cert.pem",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyCertificateFile(tt.certFile)
			if tt.wantErr {
				if err == nil {
					t.Error("VerifyCertificateFile() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("VerifyCertificateFile() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestVerifyKeyPair(t *testing.T) {
	certs := setupTestCerts(t)

	tests := []struct {
		name     string
		certFile string
		keyFile  string
		wantErr  bool
	}{
		{
			name:     "valid key pair",
			certFile: certs.certFile,
			keyFile:  certs.keyFile,
			wantErr:  false,
		},
		{
			name:     "nonexistent cert file",
			certFile: "/nonexistent/cert.pem",
			keyFile:  certs.keyFile,
			wantErr:  true,
		},
		{
			name:     "nonexistent key file",
			certFile: certs.certFile,
			keyFile:  "/nonexistent/key.pem",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyKeyPair(tt.certFile, tt.keyFile)
			if tt.wantErr {
				if err == nil {
					t.Error("VerifyKeyPair() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("VerifyKeyPair() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestLoadCertificatesFromFile_InvalidPEM(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with invalid PEM data
	invalidPEMFile := filepath.Join(tmpDir, "invalid.pem")
	if err := os.WriteFile(invalidPEMFile, []byte("invalid pem data"), 0600); err != nil {
		t.Fatalf("failed to create invalid PEM file: %v", err)
	}

	_, err := loadCertificatesFromFile(invalidPEMFile)
	if err == nil {
		t.Error("loadCertificatesFromFile() expected error for invalid PEM but got nil")
	}
}

func TestLoadCertificatesFromFile_NoCertificates(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with valid PEM but no certificates
	noCertFile := filepath.Join(tmpDir, "nocert.pem")
	content := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0Z6V...
-----END RSA PRIVATE KEY-----`
	if err := os.WriteFile(noCertFile, []byte(content), 0600); err != nil {
		t.Fatalf("failed to create no-cert PEM file: %v", err)
	}

	_, err := loadCertificatesFromFile(noCertFile)
	if err == nil {
		t.Error("loadCertificatesFromFile() expected error for file with no certificates but got nil")
	}
}
