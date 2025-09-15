package kafka

import (
	"context"
	"testing"

	"github.com/innovationmech/swit/pkg/messaging"
)

func TestKafkaAdapter_CreateBroker_And_Lifecycle(t *testing.T) {
	a := newAdapter()

	cfg := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeKafka,
		Endpoints: []string{"localhost:9092"},
	}

	// Validate config
	res := a.ValidateConfiguration(cfg)
	if res == nil || !res.Valid {
		t.Fatalf("expected valid config, got %#v", res)
	}

	// Create broker
	broker, err := a.CreateBroker(cfg)
	if err != nil {
		t.Fatalf("CreateBroker failed: %v", err)
	}

	// Connect / Disconnect
	ctx := context.Background()
	if err := broker.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if !broker.IsConnected() {
		t.Fatalf("expected connected state")
	}
	if _, err := broker.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
	if err := broker.Disconnect(ctx); err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}
}
