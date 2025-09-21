// Copyright © 2025 jackelyj
package kafka

import (
	"testing"

	"github.com/innovationmech/swit/pkg/messaging"
)

func TestBuildSASLMechanism_Plain(t *testing.T) {
	mech, err := buildSASLMechanism(&messaging.AuthConfig{
		Type:      messaging.AuthTypeSASL,
		Mechanism: "PLAIN",
		Username:  "user",
		Password:  "pass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mech == nil {
		t.Fatalf("expected non-nil mechanism")
	}
}

func TestBuildSASLMechanism_Scram256(t *testing.T) {
	mech, err := buildSASLMechanism(&messaging.AuthConfig{
		Type:      messaging.AuthTypeSASL,
		Mechanism: "SCRAM-SHA-256",
		Username:  "user",
		Password:  "pass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mech == nil {
		t.Fatalf("expected non-nil mechanism")
	}
}

func TestBuildSASLMechanism_Scram512(t *testing.T) {
	mech, err := buildSASLMechanism(&messaging.AuthConfig{
		Type:      messaging.AuthTypeSASL,
		Mechanism: "SCRAM-SHA-512",
		Username:  "user",
		Password:  "pass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mech == nil {
		t.Fatalf("expected non-nil mechanism")
	}
}

func TestBuildSASLMechanism_Unsupported(t *testing.T) {
	if _, err := buildSASLMechanism(&messaging.AuthConfig{Type: messaging.AuthTypeOAuth2}); err == nil {
		t.Fatalf("expected error for unsupported auth type")
	}
}

func TestBuildTLSConfig_Basic(t *testing.T) {
	cfg, err := buildTLSConfig(&messaging.TLSConfig{Enabled: true, SkipVerify: true, ServerName: "kafka.local"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected non-nil tls config")
	}
	if !cfg.InsecureSkipVerify {
		t.Fatalf("expected InsecureSkipVerify=true")
	}
	if cfg.ServerName != "kafka.local" {
		t.Fatalf("unexpected server name: %s", cfg.ServerName)
	}
}

func TestMapRequiredAcks(t *testing.T) {
	if v := mapRequiredAcks("none"); v != 0 { // kafka.RequireNone == 0
		t.Fatalf("expected RequireNone (0), got %d", v)
	}
	if v := mapRequiredAcks("leader"); v != 1 { // kafka.RequireOne == 1
		t.Fatalf("expected RequireOne (1), got %d", v)
	}
	if v := mapRequiredAcks("all"); v != -1 { // kafka.RequireAll == -1 in kafka-go
		t.Fatalf("expected RequireAll (-1), got %d", v)
	}
}

func TestMapCompression(t *testing.T) {
	if _, ok := mapCompression(messaging.CompressionNone); ok {
		t.Fatalf("none should not set compression option")
	}
	if c, ok := mapCompression(messaging.CompressionGZIP); !ok || int(c) == 0 {
		t.Fatalf("gzip should set non-zero compression, ok=%v", ok)
	}
}
