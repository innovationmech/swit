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
