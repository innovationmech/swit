package messaging_test

import (
	"context"
	"testing"

	. "github.com/innovationmech/swit/pkg/messaging"
)

// This test suite validates cross-broker configuration invariants surfaced by
// adapter-level validators to ensure the matrix in docs remains enforced.
func TestCrossBrokerConfigurationInvariants(t *testing.T) {
	t.Run("Kafka idempotent requires acks=all", func(t *testing.T) {
		cfg := &BrokerConfig{
			Type:      BrokerTypeKafka,
			Endpoints: []string{"localhost:9092"},
			Extra: map[string]any{
				"kafka": map[string]any{
					"producer": map[string]any{
						"idempotent": true,
						"acks":       "leader", // invalid with idempotent=true
					},
				},
			},
		}

		// Use default factory path that invokes adapter validators
		err := ValidateBrokerConfig(cfg)
		if err == nil {
			t.Fatalf("expected validation error for idempotent=true with acks!=all")
		}
		// Sanity: make it valid
		cfg.Extra["kafka"].(map[string]any)["producer"].(map[string]any)["acks"] = "all"
		if err := ValidateBrokerConfig(cfg); err != nil {
			t.Fatalf("expected valid config when acks=all: %v", err)
		}
	})

	t.Run("RabbitMQ endpoint requires amqp scheme", func(t *testing.T) {
		cfg := &BrokerConfig{
			Type:      BrokerTypeRabbitMQ,
			Endpoints: []string{"localhost:5672"}, // missing amqp://
		}
		if err := ValidateBrokerConfig(cfg); err == nil {
			t.Fatalf("expected validation error for rabbit endpoint without amqp scheme")
		}
		cfg.Endpoints = []string{"amqp://guest:guest@localhost:5672/"}
		if err := ValidateBrokerConfig(cfg); err != nil {
			t.Fatalf("expected valid rabbit config with amqp scheme: %v", err)
		}
	})

	t.Run("NATS endpoint requires scheme and TLS SkipVerify warns", func(t *testing.T) {
		cfg := &BrokerConfig{
			Type:      BrokerTypeNATS,
			Endpoints: []string{"localhost:4222"},
		}
		// Expect error due to missing scheme
		if err := ValidateBrokerConfig(cfg); err == nil {
			t.Fatalf("expected validation error for nats endpoint without scheme")
		}
		cfg.Endpoints = []string{"nats://127.0.0.1:4222"}
		cfg.TLS = &TLSConfig{Enabled: true, SkipVerify: true}
		// SkipVerify only generates a warning inside adapter, not an error; validation should pass
		if err := ValidateBrokerConfig(cfg); err != nil {
			t.Fatalf("expected valid nats config (SkipVerify should warn, not fail): %v", err)
		}
	})
}

func TestCompatibilityAPIsRemainUsable(t *testing.T) {
	checker := NewDefaultBrokerCompatibilityChecker()
	if checker == nil {
		t.Fatalf("nil compatibility checker")
	}
	if _, err := checker.CheckCompatibility(context.Background(), BrokerTypeKafka, BrokerTypeRabbitMQ); err != nil {
		t.Fatalf("CheckCompatibility error: %v", err)
	}
}
