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
