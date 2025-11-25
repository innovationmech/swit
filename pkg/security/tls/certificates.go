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
//

package tls

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// LoadCertificate loads a certificate and its private key from PEM files.
// Both certFile and keyFile must be in PEM format.
func LoadCertificate(certFile, keyFile string) (tls.Certificate, error) {
	if certFile == "" {
		return tls.Certificate{}, errors.New("certificate file path cannot be empty")
	}
	if keyFile == "" {
		return tls.Certificate{}, errors.New("key file path cannot be empty")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load X.509 key pair from %s and %s: %w", certFile, keyFile, err)
	}

	// Parse the certificate to validate it
	if len(cert.Certificate) > 0 {
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return tls.Certificate{}, fmt.Errorf("failed to parse certificate: %w", err)
		}

		// Store the parsed certificate for later use
		cert.Leaf = x509Cert
	}

	return cert, nil
}

// loadCertificatesFromFile loads X.509 certificates from a PEM file.
// This is an internal helper function used by LoadCAPool.
func loadCertificatesFromFile(caFile string) ([]*x509.Certificate, error) {
	if caFile == "" {
		return nil, errors.New("CA file path cannot be empty")
	}

	// Read the file
	data, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %w", err)
	}

	// Parse all PEM blocks
	var certs []*x509.Certificate
	rest := data
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}

		if block.Type != "CERTIFICATE" {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate in %s: %w", caFile, err)
		}

		certs = append(certs, cert)
	}

	if len(certs) == 0 {
		return nil, fmt.Errorf("no valid certificates found in %s", caFile)
	}

	return certs, nil
}

// LoadCertificateChain loads a certificate chain from a PEM file.
// Returns all certificates found in the file.
func LoadCertificateChain(certFile string) ([]*x509.Certificate, error) {
	return loadCertificatesFromFile(certFile)
}

// ValidateCertificate validates a certificate against a CA pool.
// This is useful for verifying that a certificate can be validated
// by the provided CA certificates.
func ValidateCertificate(cert *x509.Certificate, caPool *x509.CertPool) error {
	if cert == nil {
		return errors.New("certificate cannot be nil")
	}
	if caPool == nil {
		return errors.New("CA pool cannot be nil")
	}

	opts := x509.VerifyOptions{
		Roots: caPool,
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("certificate validation failed: %w", err)
	}

	return nil
}

// GetSystemCAPool returns the system's certificate pool.
// This is useful for creating a CA pool that includes both
// custom CA certificates and the system's trusted CAs.
func GetSystemCAPool() (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to load system CA pool: %w", err)
	}
	return pool, nil
}

// LoadCAPoolWithSystemCerts loads CA certificates from files and
// includes the system's trusted CA certificates.
func LoadCAPoolWithSystemCerts(caFiles ...string) (*x509.CertPool, error) {
	pool, err := GetSystemCAPool()
	if err != nil {
		// If we can't get the system pool, start with an empty one
		pool = x509.NewCertPool()
	}

	for _, caFile := range caFiles {
		certs, err := loadCertificatesFromFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load CA file %s: %w", caFile, err)
		}
		for _, cert := range certs {
			pool.AddCert(cert)
		}
	}

	return pool, nil
}

// VerifyCertificateFile validates that a file contains valid certificate data.
func VerifyCertificateFile(certFile string) error {
	_, err := loadCertificatesFromFile(certFile)
	return err
}

// VerifyKeyPair validates that a certificate and key pair are valid and match.
func VerifyKeyPair(certFile, keyFile string) error {
	_, err := LoadCertificate(certFile, keyFile)
	return err
}

// CertificateInfo holds extracted information from a client certificate.
type CertificateInfo struct {
	Subject            string   // Certificate subject (full DN)
	CommonName         string   // Common Name (CN)
	Organization       []string // Organization (O)
	OrganizationalUnit []string // Organizational Unit (OU)
	Country            []string // Country (C)
	Province           []string // Province/State (ST)
	Locality           []string // Locality (L)
	DNSNames           []string // Subject Alternative Names (DNS)
	EmailAddresses     []string // Subject Alternative Names (Email)
	IPAddresses        []string // Subject Alternative Names (IP)
	URIs               []string // Subject Alternative Names (URI)
	SerialNumber       string   // Certificate serial number
	NotBefore          string   // Certificate validity start time
	NotAfter           string   // Certificate validity end time
	Issuer             string   // Certificate issuer (full DN)
}

// ExtractCertificateInfo extracts information from an X.509 certificate.
// This is useful for authentication and authorization based on client certificates.
func ExtractCertificateInfo(cert *x509.Certificate) *CertificateInfo {
	if cert == nil {
		return nil
	}

	info := &CertificateInfo{
		Subject:            cert.Subject.String(),
		CommonName:         cert.Subject.CommonName,
		Organization:       cert.Subject.Organization,
		OrganizationalUnit: cert.Subject.OrganizationalUnit,
		Country:            cert.Subject.Country,
		Province:           cert.Subject.Province,
		Locality:           cert.Subject.Locality,
		DNSNames:           cert.DNSNames,
		EmailAddresses:     cert.EmailAddresses,
		SerialNumber:       cert.SerialNumber.String(),
		NotBefore:          cert.NotBefore.Format("2006-01-02 15:04:05 MST"),
		NotAfter:           cert.NotAfter.Format("2006-01-02 15:04:05 MST"),
		Issuer:             cert.Issuer.String(),
	}

	// Extract IP addresses
	for _, ip := range cert.IPAddresses {
		info.IPAddresses = append(info.IPAddresses, ip.String())
	}

	// Extract URIs
	for _, uri := range cert.URIs {
		info.URIs = append(info.URIs, uri.String())
	}

	return info
}

// GetCommonName extracts the Common Name (CN) from a certificate.
// This is a convenience function for quick CN extraction.
func GetCommonName(cert *x509.Certificate) string {
	if cert == nil {
		return ""
	}
	return cert.Subject.CommonName
}

// GetSANs extracts all Subject Alternative Names (SANs) from a certificate.
// Returns DNS names, email addresses, IP addresses, and URIs.
func GetSANs(cert *x509.Certificate) (dnsNames, emails, ips, uris []string) {
	if cert == nil {
		return
	}

	dnsNames = cert.DNSNames
	emails = cert.EmailAddresses

	for _, ip := range cert.IPAddresses {
		ips = append(ips, ip.String())
	}

	for _, uri := range cert.URIs {
		uris = append(uris, uri.String())
	}

	return
}
