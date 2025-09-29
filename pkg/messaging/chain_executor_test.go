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
