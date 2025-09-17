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
