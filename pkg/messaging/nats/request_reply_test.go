package nats

import (
	"context"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	natsgo "github.com/nats-io/nats.go"
)

// Note: These are unit-level tests that do not require a running NATS server.
// They validate error mapping for context timeout and basic constructor usage.

func TestRequestReplyClient_Request_ContextTimeout(t *testing.T) {
	// Use a nil connection to avoid external dependency; we expect connection error
	client := NewRequestReplyClient(nil, DefaultConfig())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	_, err := client.Request(ctx, "subject", []byte("ping"))
	if err == nil {
		t.Fatalf("expected error for nil connection")
	}
	if !messaging.IsConnectionError(err) {
		t.Fatalf("expected connection error, got: %v", err)
	}
}

func TestRequestReplyClient_RequestMessage_NilRequest(t *testing.T) {
	client := &RequestReplyClient{conn: nil}
	_, err := client.RequestMessage(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error for nil request message")
	}
}

func TestNewRequestReplyClient_DefaultTimeout(t *testing.T) {
	// Construct with a dummy conn (nil) and check timeout from DefaultConfig
	cfg := DefaultConfig()
	c := NewRequestReplyClient((*natsgo.Conn)(nil), cfg)
	if c.timeout != cfg.Timeouts.RequestTimeout() {
		t.Fatalf("unexpected timeout: %v", c.timeout)
	}
}
