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
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/nats-io/nats.go"
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

// fakeJS embeds the JetStreamContext interface to obtain the full method set;
// tests exercising paths that never invoke JS methods can use it directly.
type fakeJS struct {
	nats.JetStreamContext
}

func TestSeek_CoreNATSNotSupported(t *testing.T) {
	s := &subscriber{}
	err := s.Seek(context.Background(), messaging.SeekPosition{Type: messaging.SeekTypeBeginning})
	if err == nil {
		t.Fatal("expected error when seeking without JetStream")
	}
}

func TestSeek_NoActivePullSubscriptions(t *testing.T) {
	s := &subscriber{js: fakeJS{}}
	err := s.Seek(context.Background(), messaging.SeekPosition{Type: messaging.SeekTypeBeginning})
	if err == nil {
		t.Fatal("expected error when no pull subscriptions are active")
	}
}

func TestSeekConsumerConfig_Mapping(t *testing.T) {
	base := nats.ConsumerConfig{
		Durable:       "cg",
		AckPolicy:     nats.AckExplicitPolicy,
		DeliverPolicy: nats.DeliverAllPolicy,
	}
	ts := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		position   messaging.SeekPosition
		wantPolicy nats.DeliverPolicy
		wantSeq    uint64
		wantTime   *time.Time
		wantErr    bool
	}{
		{
			name:       "beginning",
			position:   messaging.SeekPosition{Type: messaging.SeekTypeBeginning},
			wantPolicy: nats.DeliverAllPolicy,
		},
		{
			name:       "end",
			position:   messaging.SeekPosition{Type: messaging.SeekTypeEnd},
			wantPolicy: nats.DeliverNewPolicy,
		},
		{
			name:       "offset maps to start sequence",
			position:   messaging.SeekPosition{Type: messaging.SeekTypeOffset, Offset: 42},
			wantPolicy: nats.DeliverByStartSequencePolicy,
			wantSeq:    42,
		},
		{
			name:     "offset below 1 rejected",
			position: messaging.SeekPosition{Type: messaging.SeekTypeOffset, Offset: 0},
			wantErr:  true,
		},
		{
			name:       "timestamp maps to start time",
			position:   messaging.SeekPosition{Type: messaging.SeekTypeTimestamp, Timestamp: ts},
			wantPolicy: nats.DeliverByStartTimePolicy,
			wantTime:   &ts,
		},
		{
			name:     "zero timestamp rejected",
			position: messaging.SeekPosition{Type: messaging.SeekTypeTimestamp},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := seekConsumerConfig(base, tt.position)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.DeliverPolicy != tt.wantPolicy {
				t.Fatalf("deliver policy: want %v, got %v", tt.wantPolicy, got.DeliverPolicy)
			}
			if got.OptStartSeq != tt.wantSeq {
				t.Fatalf("start seq: want %d, got %d", tt.wantSeq, got.OptStartSeq)
			}
			if (tt.wantTime == nil) != (got.OptStartTime == nil) {
				t.Fatalf("start time presence mismatch: want %v, got %v", tt.wantTime, got.OptStartTime)
			}
			if tt.wantTime != nil && !got.OptStartTime.Equal(*tt.wantTime) {
				t.Fatalf("start time: want %v, got %v", tt.wantTime, got.OptStartTime)
			}
			// Durable identity must be preserved for rebinding.
			if got.Durable != base.Durable {
				t.Fatalf("durable changed: %q", got.Durable)
			}
		})
	}
}
