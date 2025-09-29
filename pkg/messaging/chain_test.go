package messaging

import (
	"context"
	"testing"
)

func TestHandlerChain_AddPrependAndLen(t *testing.T) {
	c := NewHandlerChain()
	if c.Len() != 0 {
		t.Fatalf("expected 0")
	}

	h1 := MessageHandlerFunc(func(ctx context.Context, m *Message) error { return nil })
	h2 := MessageHandlerFunc(func(ctx context.Context, m *Message) error { return nil })
	h3 := MessageHandlerFunc(func(ctx context.Context, m *Message) error { return nil })

	c.Add(h1)
	c.Add(h2)
	if c.Len() != 2 {
		t.Fatalf("expected 2, got %d", c.Len())
	}

	c.Prepend(h3)
	steps := c.Steps()
	if len(steps) != 3 {
		t.Fatalf("expected 3, got %d", len(steps))
	}
	if &steps[0] == &steps[1] {
		t.Fatalf("unexpected aliasing")
	}
}
