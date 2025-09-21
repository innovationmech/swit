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

package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/innovationmech/swit/pkg/messaging"
	kgo "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// buildTLSConfig constructs a *tls.Config from generic messaging TLS settings.
func buildTLSConfig(tlsCfg *messaging.TLSConfig) (*tls.Config, error) {
	if tlsCfg == nil || !tlsCfg.Enabled {
		return nil, nil
	}

	cfg := &tls.Config{InsecureSkipVerify: tlsCfg.SkipVerify} //nolint:gosec // allow via configuration
	if tlsCfg.ServerName != "" {
		cfg.ServerName = tlsCfg.ServerName
	}

	// Load client cert if provided
	if tlsCfg.CertFile != "" && tlsCfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
		if err != nil {
			return nil, messaging.NewConfigError(fmt.Sprintf("failed to load client TLS cert/key: %v", err), err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	// Load CA if provided
	if tlsCfg.CAFile != "" {
		caData, err := os.ReadFile(tlsCfg.CAFile)
		if err != nil {
			return nil, messaging.NewConfigError(fmt.Sprintf("failed to read CA file: %v", err), err)
		}
		pool := x509.NewCertPool()
		if ok := pool.AppendCertsFromPEM(caData); !ok {
			return nil, messaging.NewConfigError("failed to append CA certificates", nil)
		}
		cfg.RootCAs = pool
	}

	return cfg, nil
}

// buildSASLMechanism constructs a kafka-go SASL mechanism from generic auth config.
func buildSASLMechanism(auth *messaging.AuthConfig) (sasl.Mechanism, error) {
	if auth == nil || auth.Type == messaging.AuthTypeNone {
		return nil, nil
	}
	if auth.Type != messaging.AuthTypeSASL {
		// For now, only SASL is supported for Kafka. Others may be added later.
		return nil, messaging.NewConfigError(fmt.Sprintf("unsupported auth type for Kafka: %s", auth.Type), nil)
	}

	mechanism := auth.Mechanism
	switch mechanism {
	case "PLAIN", "plain", "Plain":
		return plain.Mechanism{Username: auth.Username, Password: auth.Password}, nil
	case "SCRAM-SHA-256", "scram-sha-256":
		m, err := scram.Mechanism(scram.SHA256, auth.Username, auth.Password)
		if err != nil {
			return nil, messaging.NewConfigError("failed to initialize SCRAM-SHA-256 mechanism", err)
		}
		return m, nil
	case "SCRAM-SHA-512", "scram-sha-512":
		m, err := scram.Mechanism(scram.SHA512, auth.Username, auth.Password)
		if err != nil {
			return nil, messaging.NewConfigError("failed to initialize SCRAM-SHA-512 mechanism", err)
		}
		return m, nil
	case "", "AUTO":
		// Default to PLAIN when mechanism unspecified but SASL chosen
		return plain.Mechanism{Username: auth.Username, Password: auth.Password}, nil
	default:
		return nil, messaging.NewConfigError(fmt.Sprintf("unsupported SASL mechanism: %s", mechanism), nil)
	}
}

// mapRequiredAcks converts our Kafka adapter config Acks string into kafka-go RequiredAcks.
func mapRequiredAcks(acks string) kgo.RequiredAcks {
	switch acks {
	case "none":
		return kgo.RequireNone
	case "all":
		return kgo.RequireAll
	case "leader", "":
		return kgo.RequireOne
	default:
		return kgo.RequireOne
	}
}

// mapCompression converts messaging.CompressionType to kafka-go compression codec.
func mapCompression(c messaging.CompressionType) (kgo.Compression, bool) {
	switch c {
	case messaging.CompressionGZIP:
		return kgo.Gzip, true
	case messaging.CompressionSnappy:
		return kgo.Snappy, true
	case messaging.CompressionLZ4:
		return kgo.Lz4, true
	case messaging.CompressionZSTD:
		return kgo.Zstd, true
	case messaging.CompressionNone:
		fallthrough
	default:
		return kgo.Compression(0), false // do not set; leave kafka-go default (no compression)
	}
}
