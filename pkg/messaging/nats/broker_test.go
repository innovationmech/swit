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

	"github.com/innovationmech/swit/pkg/messaging"
)

func TestNATSBrokerConnectAndFactory(t *testing.T) {
	cfg := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeNATS,
		Endpoints: []string{"nats://127.0.0.1:4222"},
	}

	adapter := newAdapter()
	if adapter.GetAdapterInfo().Name == "" {
		t.Fatal("adapter name should not be empty")
	}

	res := adapter.ValidateConfiguration(cfg)
	if !res.Valid {
		t.Fatalf("expected config valid, got errors: %+v", res.Errors)
	}

	broker, err := adapter.CreateBroker(cfg)
	if err != nil {
		t.Fatalf("CreateBroker failed: %v", err)
	}

	// Connect will fail without a running NATS server; ensure error type is connection error or ok.
	err = broker.Connect(context.Background())
	if err != nil && !messaging.IsConnectionError(err) {
		t.Fatalf("unexpected error type from Connect without server: %v", err)
	}
}
