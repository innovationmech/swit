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
package kafka

import (
	"testing"

	"github.com/innovationmech/swit/pkg/messaging"
)

func TestValidateKafkaConfiguration_EndpointsFormat(t *testing.T) {
	base := &messaging.BrokerConfig{Type: messaging.BrokerTypeKafka, Endpoints: []string{"kafka://localhost:9092"}}
	cfg := DefaultConfig()
	errs, _, _ := validateKafkaConfiguration(base, cfg)
	if len(errs) == 0 {
		t.Fatalf("expected endpoint format error")
	}
}

func TestValidateKafkaConfiguration_SASLMechanisms(t *testing.T) {
	tests := []struct {
		name   string
		auth   *messaging.AuthConfig
		wantOK bool
	}{
		{name: "none", auth: &messaging.AuthConfig{Type: messaging.AuthTypeNone}, wantOK: true},
		{name: "sasl-plain-ok", auth: &messaging.AuthConfig{Type: messaging.AuthTypeSASL, Mechanism: "PLAIN", Username: "u", Password: "p"}, wantOK: true},
		{name: "sasl-scram256-ok", auth: &messaging.AuthConfig{Type: messaging.AuthTypeSASL, Mechanism: "SCRAM-SHA-256", Username: "u", Password: "p"}, wantOK: true},
		{name: "sasl-scram512-ok", auth: &messaging.AuthConfig{Type: messaging.AuthTypeSASL, Mechanism: "SCRAM-SHA-512", Username: "u", Password: "p"}, wantOK: true},
		{name: "sasl-auto-missing-cred", auth: &messaging.AuthConfig{Type: messaging.AuthTypeSASL, Mechanism: "AUTO"}, wantOK: false},
		{name: "sasl-plain-missing-cred", auth: &messaging.AuthConfig{Type: messaging.AuthTypeSASL, Mechanism: "PLAIN"}, wantOK: false},
		{name: "unsupported-auth", auth: &messaging.AuthConfig{Type: messaging.AuthTypeOAuth2}, wantOK: false},
		{name: "unsupported-mech", auth: &messaging.AuthConfig{Type: messaging.AuthTypeSASL, Mechanism: "NTLM"}, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := &messaging.BrokerConfig{Type: messaging.BrokerTypeKafka, Endpoints: []string{"localhost:9092"}, Authentication: tt.auth}
			cfg := DefaultConfig()
			errs, _, _ := validateKafkaConfiguration(base, cfg)
			if tt.wantOK && len(errs) > 0 {
				t.Fatalf("expected ok, got errs=%v", errs)
			}
			if !tt.wantOK && len(errs) == 0 {
				t.Fatalf("expected error, got ok")
			}
		})
	}
}

func TestValidateKafkaConfiguration_IdempotenceRequiresAllAcks(t *testing.T) {
	base := &messaging.BrokerConfig{Type: messaging.BrokerTypeKafka, Endpoints: []string{"localhost:9092"}}
	cfg := DefaultConfig()
	cfg.Producer.Idempotent = true
	cfg.Producer.Acks = "leader"
	errs, _, _ := validateKafkaConfiguration(base, cfg)
	if len(errs) == 0 {
		t.Fatalf("expected error for idempotent without acks=all")
	}

	cfg.Producer.Acks = "all"
	errs, _, _ = validateKafkaConfiguration(base, cfg)
	if len(errs) != 0 {
		t.Fatalf("expected ok for idempotent with acks=all, got errs=%v", errs)
	}
}
