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
