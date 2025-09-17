package nats

import (
	"context"
	"testing"

	"github.com/innovationmech/swit/pkg/messaging"
)

func TestNATSBrokerConnectAndFactory(t *testing.T) {
	cfg := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeNATS,
		Endpoints: []string{"nats://127.0.0.1:4222"},
	}

	adapter := newAdapter()
	if adapter.GetAdapterInfo().Name == "" {
		t.Fatal("adapter name should not be empty")
	}

	res := adapter.ValidateConfiguration(cfg)
	if !res.Valid {
		t.Fatalf("expected config valid, got errors: %+v", res.Errors)
	}

	broker, err := adapter.CreateBroker(cfg)
	if err != nil {
		t.Fatalf("CreateBroker failed: %v", err)
	}

	// Connect will fail without a running NATS server; ensure error type is connection error or ok.
	err = broker.Connect(context.Background())
	if err != nil && !messaging.IsConnectionError(err) {
		t.Fatalf("unexpected error type from Connect without server: %v", err)
	}
}
