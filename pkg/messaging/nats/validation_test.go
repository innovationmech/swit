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

package nats

import (
	"testing"

	"github.com/innovationmech/swit/pkg/messaging"
)

func TestValidateNATSConfiguration_EndpointsAndTLS(t *testing.T) {
	base := &messaging.BrokerConfig{Type: messaging.BrokerTypeNATS, Endpoints: []string{"nats://localhost:4222"}}
	cfg := DefaultConfig()
	errs, warns, _ := validateNATSConfiguration(base, cfg)
	if len(errs) != 0 {
		t.Fatalf("expected no endpoint errors, got %v", errs)
	}

	base.TLS = &messaging.TLSConfig{Enabled: true, SkipVerify: true}
	errs, warns, _ = validateNATSConfiguration(base, cfg)
	if len(warns) == 0 {
		t.Fatalf("expected TLS SkipVerify warning")
	}
}

func TestValidateNATSConfiguration_OAuth2(t *testing.T) {
	base := &messaging.BrokerConfig{Type: messaging.BrokerTypeNATS, Endpoints: []string{"nats://localhost:4222"}}
	cfg := DefaultConfig()

	// Static token OK
	base.Authentication = &messaging.AuthConfig{Type: messaging.AuthTypeOAuth2, Token: "abc"}
	errs, _, _ := validateNATSConfiguration(base, cfg)
	if len(errs) != 0 {
		t.Fatalf("expected ok for oauth2 static token, got %v", errs)
	}

	// Client credentials missing fields -> error
	base.Authentication = &messaging.AuthConfig{Type: messaging.AuthTypeOAuth2}
	errs, _, _ = validateNATSConfiguration(base, cfg)
	if len(errs) == 0 {
		t.Fatalf("expected error for incomplete oauth2 client credentials config")
	}

	// Client credentials complete -> ok
	base.Authentication = &messaging.AuthConfig{
		Type:         messaging.AuthTypeOAuth2,
		ClientID:     "id",
		ClientSecret: "secret",
		TokenURL:     "https://example.com/token",
	}
	errs, _, _ = validateNATSConfiguration(base, cfg)
	if len(errs) != 0 {
		t.Fatalf("expected ok for oauth2 client credentials, got %v", errs)
	}
}
