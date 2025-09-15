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

package kafka

import (
	"context"
	"testing"

	"github.com/innovationmech/swit/pkg/messaging"
)

func TestKafkaAdapter_CreateBroker_And_Lifecycle(t *testing.T) {
	a := newAdapter()

	cfg := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeKafka,
		Endpoints: []string{"localhost:9092"},
	}

	// Validate config
	res := a.ValidateConfiguration(cfg)
	if res == nil || !res.Valid {
		t.Fatalf("expected valid config, got %#v", res)
	}

	// Create broker
	broker, err := a.CreateBroker(cfg)
	if err != nil {
		t.Fatalf("CreateBroker failed: %v", err)
	}

	// Connect / Disconnect
	ctx := context.Background()
	if err := broker.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if !broker.IsConnected() {
		t.Fatalf("expected connected state")
	}
	if _, err := broker.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
	if err := broker.Disconnect(ctx); err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}
}
