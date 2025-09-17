package nats

import (
	"context"
	"testing"
)

func TestGetDeliverGroupFor_MatchAndNoMatch(t *testing.T) {
	jsCfg := &JetStreamConfig{
		Enabled: true,
		Consumers: []JSConsumerConfig{
			{Name: "c1", FilterSubject: "orders.created", DeliverGroup: "order-workers"},
			{Name: "c2", FilterSubject: "payments.>", DeliverGroup: "payment-workers"},
		},
	}

	s := &subscriber{jsConfig: jsCfg}

	if got := s.getDeliverGroupFor("orders.created"); got != "order-workers" {
		t.Fatalf("expected deliver group 'order-workers', got %q", got)
	}
	if got := s.getDeliverGroupFor("orders.updated"); got != "" {
		t.Fatalf("expected empty deliver group for unmatched subject, got %q", got)
	}
}

func TestGetDeliverSubjectFor_Match(t *testing.T) {
	jsCfg := &JetStreamConfig{
		Enabled: true,
		Consumers: []JSConsumerConfig{
			{Name: "c1", FilterSubject: "orders.created", DeliverSubject: "_INBOX.orders.created"},
		},
	}
	s := &subscriber{jsConfig: jsCfg}
	if got := s.getDeliverSubjectFor("orders.created"); got != "_INBOX.orders.created" {
		t.Fatalf("expected deliver subject '_INBOX.orders.created', got %q", got)
	}
}

func TestUnsubscribeWithoutSubscriptions_ReturnsImmediately(t *testing.T) {
	s := &subscriber{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Should return nil quickly without panic
	if err := s.Unsubscribe(ctx); err != nil && err != context.Canceled {
		t.Fatalf("unexpected error from Unsubscribe: %v", err)
	}
}
