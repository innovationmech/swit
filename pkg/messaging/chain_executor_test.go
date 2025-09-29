package messaging

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stepHandler struct {
	name string
	cb   func(ctx context.Context, msg *Message) error
}

func (s stepHandler) Handle(ctx context.Context, message *Message) error { return s.cb(ctx, message) }
func (s stepHandler) OnError(ctx context.Context, message *Message, err error) ErrorAction {
	return ErrorActionRetry
}

func TestBaseChainExecutor_OrderAndShortCircuit(t *testing.T) {
	exec := NewBaseChainExecutor()
	chain := NewHandlerChain(
		stepHandler{"s1", func(ctx context.Context, msg *Message) error { return nil }},
		stepHandler{"s2", func(ctx context.Context, msg *Message) error { return errors.New("boom") }},
		stepHandler{"s3", func(ctx context.Context, msg *Message) error { t.Fatalf("should short-circuit"); return nil }},
	)

	err := exec.Execute(context.Background(), chain, &Message{ID: "1"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestBaseChainExecutor_CollectAll(t *testing.T) {
	exec := NewBaseChainExecutor().WithOptions(ChainExecutorOptions{ShortCircuit: false, ErrorPolicy: ErrorPolicyCollectAll})
	chain := NewHandlerChain(
		stepHandler{"s1", func(ctx context.Context, msg *Message) error { return errors.New("e1") }},
		stepHandler{"s2", func(ctx context.Context, msg *Message) error { return errors.New("e2") }},
	)

	err := exec.Execute(context.Background(), chain, &Message{ID: "1"})
	if err == nil {
		t.Fatalf("expected aggregated error, got nil")
	}
}

func TestBaseChainExecutor_ContextCancel(t *testing.T) {
	exec := NewBaseChainExecutor()
	chain := NewHandlerChain(
		stepHandler{"s1", func(ctx context.Context, msg *Message) error {
			<-ctx.Done()
			return ctx.Err()
		}},
		stepHandler{"s2", func(ctx context.Context, msg *Message) error { t.Fatalf("should not run"); return nil }},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := exec.Execute(ctx, chain, &Message{ID: "1"})
	if err == nil {
		t.Fatalf("expected context error, got nil")
	}
}
