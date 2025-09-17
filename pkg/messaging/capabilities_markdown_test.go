package messaging

import (
	"strings"
	"testing"
)

func TestGenerateCapabilitiesMarkdown(t *testing.T) {
	md, err := GenerateCapabilitiesMarkdown()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if md == "" {
		t.Fatalf("expected non-empty markdown output")
	}
	// Basic sanity checks
	if !strings.Contains(md, "# Message Broker Capabilities Matrix") {
		t.Errorf("missing title header")
	}
	if !strings.Contains(md, "| Feature | Kafka | RabbitMQ | NATS |") {
		t.Errorf("missing feature table header")
	}
	if !strings.Contains(md, "Transactions") {
		t.Errorf("missing transactions row")
	}
	if !strings.Contains(md, "Graceful Degradation") {
		t.Errorf("missing graceful degradation section")
	}
}
