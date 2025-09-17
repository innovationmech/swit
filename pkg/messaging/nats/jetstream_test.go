package nats

import (
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/nats-io/nats.go"
)

func TestToNATSStreamConfigMapping(t *testing.T) {
	cases := []struct {
		name  string
		in    JSStreamConfig
		check func(*nats.StreamConfig) error
	}{
		{
			name: "defaults and file storage limits retention",
			in: JSStreamConfig{
				Name:       "ORDERS",
				Subjects:   []string{"orders.>"},
				Retention:  "limits",
				Storage:    "file",
				MaxBytes:   10,
				MaxMsgs:    100,
				MaxMsgSize: 1024,
				MaxAge:     messaging.Duration(time.Hour),
				Replicas:   3,
			},
			check: func(sc *nats.StreamConfig) error {
				if sc.Retention != nats.LimitsPolicy {
					t.Fatalf("retention mismatch")
				}
				if sc.Storage != nats.FileStorage {
					t.Fatalf("storage mismatch")
				}
				if sc.Name != "ORDERS" {
					t.Fatalf("name mismatch")
				}
				if sc.Replicas != 3 {
					t.Fatalf("replicas mismatch")
				}
				if sc.MaxBytes != 10 || sc.MaxMsgs != 100 || sc.MaxMsgSize != 1024 {
					t.Fatalf("limits mismatch")
				}
				if sc.MaxAge != time.Hour {
					t.Fatalf("max age mismatch")
				}
				return nil
			},
		},
		{
			name: "workqueue and memory storage",
			in: JSStreamConfig{
				Name:      "JOBS",
				Subjects:  []string{"jobs.>"},
				Retention: "workqueue",
				Storage:   "memory",
			},
			check: func(sc *nats.StreamConfig) error {
				if sc.Retention != nats.WorkQueuePolicy {
					t.Fatalf("retention mismatch")
				}
				if sc.Storage != nats.MemoryStorage {
					t.Fatalf("storage mismatch")
				}
				return nil
			},
		},
		{
			name: "interest policy",
			in: JSStreamConfig{
				Name:      "EVENTS",
				Retention: "interest",
			},
			check: func(sc *nats.StreamConfig) error {
				if sc.Retention != nats.InterestPolicy {
					t.Fatalf("retention mismatch")
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := toNATSStreamConfig(&tc.in)
			_ = tc.check(out)
		})
	}
}

func TestToNATSConsumerConfigMapping(t *testing.T) {
	cases := []struct {
		name  string
		in    JSConsumerConfig
		check func(*nats.ConsumerConfig)
	}{
		{
			name: "explicit ack default and instant replay",
			in: JSConsumerConfig{
				Name:          "order-processor",
				Stream:        "ORDERS",
				FilterSubject: "orders.created",
				AckWait:       messaging.Duration(30 * time.Second),
				MaxAckPending: 100,
			},
			check: func(cc *nats.ConsumerConfig) {
				if cc.AckPolicy != nats.AckExplicitPolicy {
					t.Fatalf("ack policy mismatch")
				}
				if cc.ReplayPolicy != nats.ReplayInstantPolicy {
					t.Fatalf("replay policy mismatch")
				}
				if cc.AckWait != 30*time.Second {
					t.Fatalf("ack wait mismatch")
				}
				if cc.MaxAckPending != 100 {
					t.Fatalf("max ack pending mismatch")
				}
			},
		},
		{
			name: "ack none and last delivery with durable",
			in: JSConsumerConfig{
				Name:          "events",
				Durable:       true,
				DeliverPolicy: "last",
				AckPolicy:     "none",
			},
			check: func(cc *nats.ConsumerConfig) {
				if cc.DeliverPolicy != nats.DeliverLastPolicy {
					t.Fatalf("deliver policy mismatch")
				}
				if cc.AckPolicy != nats.AckNonePolicy {
					t.Fatalf("ack policy mismatch")
				}
				if cc.Durable != "events" {
					t.Fatalf("durable mismatch")
				}
			},
		},
		{
			name: "ack all and by_start_time",
			in: JSConsumerConfig{
				Name:          "time",
				AckPolicy:     "all",
				DeliverPolicy: "by_start_time",
				ReplayPolicy:  "original",
			},
			check: func(cc *nats.ConsumerConfig) {
				if cc.AckPolicy != nats.AckAllPolicy {
					t.Fatalf("ack policy mismatch")
				}
				if cc.DeliverPolicy != nats.DeliverByStartTimePolicy {
					t.Fatalf("deliver policy mismatch")
				}
				if cc.ReplayPolicy != nats.ReplayOriginalPolicy {
					t.Fatalf("replay policy mismatch")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := toNATSConsumerConfig(&tc.in)
			tc.check(out)
		})
	}
}
