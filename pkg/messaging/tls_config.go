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

package messaging

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// BuildTLSConfig constructs a *tls.Config from TLS settings.
// Returns nil when settings are nil or disabled.
func BuildTLSConfig(settings *TLSConfig) (*tls.Config, error) {
	if settings == nil || !settings.Enabled {
		return nil, nil
	}

	cfg := &tls.Config{InsecureSkipVerify: settings.SkipVerify} // nolint:gosec // explicitly controlled via configuration

	if settings.ServerName != "" {
		cfg.ServerName = settings.ServerName
	}

	// Load client certificate for mTLS if provided
	if settings.CertFile != "" && settings.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(settings.CertFile, settings.KeyFile)
		if err != nil {
			return nil, NewConfigError(fmt.Sprintf("failed to load client TLS cert/key: %v", err), err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	// Load custom CA bundle if provided
	if settings.CAFile != "" {
		caData, err := os.ReadFile(settings.CAFile)
		if err != nil {
			return nil, NewConfigError(fmt.Sprintf("failed to read CA file: %v", err), err)
		}
		pool := x509.NewCertPool()
		if ok := pool.AppendCertsFromPEM(caData); !ok {
			return nil, NewConfigError("failed to append CA certificates", nil)
		}
		cfg.RootCAs = pool
	}

	return cfg, nil
}
