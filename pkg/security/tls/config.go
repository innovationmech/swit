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

// Package tls provides TLS/mTLS configuration management for secure communications.
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
)

// TLSConfig holds comprehensive TLS configuration options.
type TLSConfig struct {
	// Enabled indicates whether TLS is enabled.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// CertFile is the path to the certificate file (PEM format).
	CertFile string `json:"cert_file,omitempty" yaml:"cert_file,omitempty" mapstructure:"cert_file"`

	// KeyFile is the path to the private key file (PEM format).
	KeyFile string `json:"key_file,omitempty" yaml:"key_file,omitempty" mapstructure:"key_file"`

	// CAFiles is a list of paths to CA certificate files (PEM format).
	CAFiles []string `json:"ca_files,omitempty" yaml:"ca_files,omitempty" mapstructure:"ca_files"`

	// MinVersion specifies the minimum TLS version (e.g., "TLS1.2", "TLS1.3").
	// Defaults to TLS 1.2 if not specified.
	MinVersion string `json:"min_version,omitempty" yaml:"min_version,omitempty" mapstructure:"min_version"`

	// MaxVersion specifies the maximum TLS version (e.g., "TLS1.2", "TLS1.3").
	// If not specified, uses the highest version supported by the implementation.
	MaxVersion string `json:"max_version,omitempty" yaml:"max_version,omitempty" mapstructure:"max_version"`

	// CipherSuites is a list of enabled cipher suites.
	// If empty, uses a secure default set.
	CipherSuites []string `json:"cipher_suites,omitempty" yaml:"cipher_suites,omitempty" mapstructure:"cipher_suites"`

	// CurvePreferences is a list of preferred elliptic curves.
	// If empty, uses a secure default set.
	CurvePreferences []string `json:"curve_preferences,omitempty" yaml:"curve_preferences,omitempty" mapstructure:"curve_preferences"`

	// ClientAuth specifies the client authentication mode for server-side TLS.
	// Valid values: "none", "request", "require", "verify_if_given", "require_and_verify"
	// Defaults to "none".
	ClientAuth string `json:"client_auth,omitempty" yaml:"client_auth,omitempty" mapstructure:"client_auth"`

	// ServerName is used for SNI (Server Name Indication) in client-side TLS.
	ServerName string `json:"server_name,omitempty" yaml:"server_name,omitempty" mapstructure:"server_name"`

	// InsecureSkipVerify controls whether a client verifies the server's certificate chain.
	// This should only be set to true for testing purposes.
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty" yaml:"insecure_skip_verify,omitempty" mapstructure:"insecure_skip_verify"`

	// PreferServerCipherSuites controls whether the server's cipher suite preferences are used.
	PreferServerCipherSuites bool `json:"prefer_server_cipher_suites,omitempty" yaml:"prefer_server_cipher_suites,omitempty" mapstructure:"prefer_server_cipher_suites"`

	// SessionTicketsDisabled controls whether session tickets (RFC 5077) are disabled.
	SessionTicketsDisabled bool `json:"session_tickets_disabled,omitempty" yaml:"session_tickets_disabled,omitempty" mapstructure:"session_tickets_disabled"`

	// Renegotiation controls the TLS renegotiation support for both client and server.
	// Valid values: "never", "once", "freely"
	// - "never": disables renegotiation (secure default)
	// - "once": allows renegotiation once per connection (works for both client and server)
	// - "freely": allows repeated renegotiation (works for both client and server)
	// Defaults to "never" for security.
	// Note: Despite the Go internal constant names containing "AsClient", these settings
	// apply to both client and server configurations in crypto/tls.
	Renegotiation string `json:"renegotiation,omitempty" yaml:"renegotiation,omitempty" mapstructure:"renegotiation"`

	// NextProtos is a list of supported application-level protocols (ALPN).
	// Common values: "h2" (HTTP/2), "http/1.1"
	NextProtos []string `json:"next_protos,omitempty" yaml:"next_protos,omitempty" mapstructure:"next_protos"`
}

// ValidateTLSConfig validates the TLS configuration for correctness.
func ValidateTLSConfig(config *TLSConfig) error {
	if config == nil {
		return errors.New("tls config cannot be nil")
	}

	if !config.Enabled {
		return nil // No validation needed if TLS is disabled
	}

	// Validate certificate and key files
	if config.CertFile != "" && config.KeyFile == "" {
		return errors.New("key_file must be specified when cert_file is provided")
	}
	if config.KeyFile != "" && config.CertFile == "" {
		return errors.New("cert_file must be specified when key_file is provided")
	}

	// Validate TLS versions
	if config.MinVersion != "" {
		if !isValidTLSVersion(config.MinVersion) {
			return fmt.Errorf("invalid min_version: %s (valid values: TLS1.0, TLS1.1, TLS1.2, TLS1.3)", config.MinVersion)
		}
	}

	if config.MaxVersion != "" {
		if !isValidTLSVersion(config.MaxVersion) {
			return fmt.Errorf("invalid max_version: %s (valid values: TLS1.0, TLS1.1, TLS1.2, TLS1.3)", config.MaxVersion)
		}
	}

	// Validate client authentication mode
	if config.ClientAuth != "" {
		if !isValidClientAuthMode(config.ClientAuth) {
			return fmt.Errorf("invalid client_auth: %s (valid values: none, request, require, verify_if_given, require_and_verify)", config.ClientAuth)
		}
	}

	// Validate cipher suites
	if len(config.CipherSuites) > 0 {
		for _, suite := range config.CipherSuites {
			if !isValidCipherSuite(suite) {
				return fmt.Errorf("invalid cipher suite: %s", suite)
			}
		}
	}

	// Validate curve preferences
	if len(config.CurvePreferences) > 0 {
		for _, curve := range config.CurvePreferences {
			if !isValidCurve(curve) {
				return fmt.Errorf("invalid curve preference: %s", curve)
			}
		}
	}

	// Validate renegotiation
	if config.Renegotiation != "" {
		if !isValidRenegotiation(config.Renegotiation) {
			return fmt.Errorf("invalid renegotiation: %s (valid values: never, once, freely)", config.Renegotiation)
		}
	}

	return nil
}

// NewTLSConfig creates a *tls.Config from TLSConfig.
// Returns nil if TLS is disabled or an error if configuration is invalid.
func NewTLSConfig(config *TLSConfig) (*tls.Config, error) {
	if config == nil || !config.Enabled {
		return nil, nil
	}

	// Validate configuration first
	if err := ValidateTLSConfig(config); err != nil {
		return nil, fmt.Errorf("invalid TLS configuration: %w", err)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.InsecureSkipVerify, // nolint:gosec // explicitly controlled via configuration
	}

	// Load certificate and key
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := LoadCertificate(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificates
	if len(config.CAFiles) > 0 {
		caPool, err := LoadCAPool(config.CAFiles...)
		if err != nil {
			return nil, fmt.Errorf("failed to load CA pool: %w", err)
		}
		tlsConfig.RootCAs = caPool
		tlsConfig.ClientCAs = caPool
	}

	// Set TLS version range
	if config.MinVersion != "" {
		tlsConfig.MinVersion = parseTLSVersion(config.MinVersion)
	} else {
		tlsConfig.MinVersion = tls.VersionTLS12 // Secure default
	}

	if config.MaxVersion != "" {
		tlsConfig.MaxVersion = parseTLSVersion(config.MaxVersion)
	}

	// Validate version range
	if tlsConfig.MaxVersion != 0 && tlsConfig.MaxVersion < tlsConfig.MinVersion {
		return nil, errors.New("max_version must be greater than or equal to min_version")
	}

	// Set cipher suites
	if len(config.CipherSuites) > 0 {
		cipherSuites, err := parseCipherSuites(config.CipherSuites)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cipher suites: %w", err)
		}
		tlsConfig.CipherSuites = cipherSuites
	}

	// Set curve preferences
	if len(config.CurvePreferences) > 0 {
		curvePreferences, err := parseCurvePreferences(config.CurvePreferences)
		if err != nil {
			return nil, fmt.Errorf("failed to parse curve preferences: %w", err)
		}
		tlsConfig.CurvePreferences = curvePreferences
	}

	// Set client authentication mode
	if config.ClientAuth != "" {
		tlsConfig.ClientAuth = parseClientAuthMode(config.ClientAuth)
	}

	// Set server name for SNI
	if config.ServerName != "" {
		tlsConfig.ServerName = config.ServerName
	}

	// Set other options
	tlsConfig.PreferServerCipherSuites = config.PreferServerCipherSuites
	tlsConfig.SessionTicketsDisabled = config.SessionTicketsDisabled

	// Set renegotiation
	if config.Renegotiation != "" {
		tlsConfig.Renegotiation = parseRenegotiation(config.Renegotiation)
	} else {
		tlsConfig.Renegotiation = tls.RenegotiateNever // Secure default
	}

	// Set ALPN protocols
	if len(config.NextProtos) > 0 {
		tlsConfig.NextProtos = config.NextProtos
	}

	return tlsConfig, nil
}

// DefaultTLSConfig returns a secure default TLS configuration.
func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		Enabled:                  true,
		MinVersion:               "TLS1.2",
		MaxVersion:               "TLS1.3",
		ClientAuth:               "none",
		PreferServerCipherSuites: true,
		SessionTicketsDisabled:   false,
		Renegotiation:            "never",
		NextProtos:               []string{"h2", "http/1.1"},
		CipherSuites: []string{
			// TLS 1.3 cipher suites (don't need to be explicitly configured, but listed for documentation)
			"TLS_AES_256_GCM_SHA384",
			"TLS_CHACHA20_POLY1305_SHA256",
			"TLS_AES_128_GCM_SHA256",
			// TLS 1.2 cipher suites
			"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305",
			"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		},
		CurvePreferences: []string{
			"X25519",
			"P256",
			"P384",
		},
	}
}

// isValidTLSVersion checks if a TLS version string is valid.
func isValidTLSVersion(version string) bool {
	validVersions := []string{"TLS1.0", "TLS1.1", "TLS1.2", "TLS1.3"}
	for _, v := range validVersions {
		if v == version {
			return true
		}
	}
	return false
}

// parseTLSVersion converts a TLS version string to its uint16 constant.
func parseTLSVersion(version string) uint16 {
	switch version {
	case "TLS1.0":
		return tls.VersionTLS10
	case "TLS1.1":
		return tls.VersionTLS11
	case "TLS1.2":
		return tls.VersionTLS12
	case "TLS1.3":
		return tls.VersionTLS13
	default:
		return tls.VersionTLS12 // Default to TLS 1.2
	}
}

// isValidClientAuthMode checks if a client authentication mode is valid.
func isValidClientAuthMode(mode string) bool {
	validModes := []string{"none", "request", "require", "verify_if_given", "require_and_verify"}
	for _, m := range validModes {
		if m == mode {
			return true
		}
	}
	return false
}

// parseClientAuthMode converts a client authentication mode string to its tls.ClientAuthType constant.
func parseClientAuthMode(mode string) tls.ClientAuthType {
	switch mode {
	case "none":
		return tls.NoClientCert
	case "request":
		return tls.RequestClientCert
	case "require":
		return tls.RequireAnyClientCert
	case "verify_if_given":
		return tls.VerifyClientCertIfGiven
	case "require_and_verify":
		return tls.RequireAndVerifyClientCert
	default:
		return tls.NoClientCert
	}
}

// isValidRenegotiation checks if a renegotiation string is valid.
func isValidRenegotiation(renego string) bool {
	validValues := []string{"never", "once", "freely"}
	for _, v := range validValues {
		if v == renego {
			return true
		}
	}
	return false
}

// parseRenegotiation converts a renegotiation string to its tls.RenegotiationSupport constant.
// Note: Despite the Go constant names containing "AsClient", these apply to both client and server.
func parseRenegotiation(renego string) tls.RenegotiationSupport {
	switch renego {
	case "never":
		return tls.RenegotiateNever
	case "once":
		// Allows renegotiation once per connection (works for both client and server)
		return tls.RenegotiateOnceAsClient
	case "freely":
		// Allows repeated renegotiation (works for both client and server)
		return tls.RenegotiateFreelyAsClient
	default:
		return tls.RenegotiateNever
	}
}

// isValidCipherSuite checks if a cipher suite name is valid.
func isValidCipherSuite(suite string) bool {
	_, exists := cipherSuiteMap[suite]
	return exists
}

// parseCipherSuites converts cipher suite names to their uint16 constants.
func parseCipherSuites(suites []string) ([]uint16, error) {
	var result []uint16
	for _, suite := range suites {
		code, exists := cipherSuiteMap[suite]
		if !exists {
			return nil, fmt.Errorf("unknown cipher suite: %s", suite)
		}
		result = append(result, code)
	}
	return result, nil
}

// cipherSuiteMap maps cipher suite names to their constants.
var cipherSuiteMap = map[string]uint16{
	// TLS 1.3 cipher suites
	"TLS_AES_128_GCM_SHA256":       tls.TLS_AES_128_GCM_SHA256,
	"TLS_AES_256_GCM_SHA384":       tls.TLS_AES_256_GCM_SHA384,
	"TLS_CHACHA20_POLY1305_SHA256": tls.TLS_CHACHA20_POLY1305_SHA256,

	// TLS 1.2 cipher suites (secure ones)
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":    tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305":  tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,

	// Additional TLS 1.2 cipher suites (for compatibility, but less preferred)
	"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
	"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,

	// RSA key exchange (not recommended but sometimes needed for compatibility)
	"TLS_RSA_WITH_AES_128_GCM_SHA256": tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_RSA_WITH_AES_256_GCM_SHA384": tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	"TLS_RSA_WITH_AES_128_CBC_SHA256": tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
	"TLS_RSA_WITH_AES_128_CBC_SHA":    tls.TLS_RSA_WITH_AES_128_CBC_SHA,
	"TLS_RSA_WITH_AES_256_CBC_SHA":    tls.TLS_RSA_WITH_AES_256_CBC_SHA,
}

// isValidCurve checks if a curve name is valid.
func isValidCurve(curve string) bool {
	_, exists := curveMap[curve]
	return exists
}

// parseCurvePreferences converts curve names to their tls.CurveID constants.
func parseCurvePreferences(curves []string) ([]tls.CurveID, error) {
	var result []tls.CurveID
	for _, curve := range curves {
		id, exists := curveMap[curve]
		if !exists {
			return nil, fmt.Errorf("unknown curve: %s", curve)
		}
		result = append(result, id)
	}
	return result, nil
}

// curveMap maps curve names to their constants.
var curveMap = map[string]tls.CurveID{
	"P256":   tls.CurveP256,
	"P384":   tls.CurveP384,
	"P521":   tls.CurveP521,
	"X25519": tls.X25519,
}

// LoadCAPool is a convenience function that loads a CA certificate pool from files.
// It's defined here for convenience but implemented in certificates.go.
func LoadCAPool(caFiles ...string) (*x509.CertPool, error) {
	if len(caFiles) == 0 {
		return nil, errors.New("at least one CA file must be provided")
	}

	pool := x509.NewCertPool()
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
