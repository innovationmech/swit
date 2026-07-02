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
	"errors"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

func TestKafkaBrokerConnect_ProbeFailureReturnsConnectionError(t *testing.T) {
	stubConnectivityProber(t, errors.New("boom"))

	b := newKafkaBroker(&messaging.BrokerConfig{
		Type:      messaging.BrokerTypeKafka,
		Endpoints: []string{"localhost:9092"},
	})

	err := b.Connect(context.Background())
	if err == nil {
		t.Fatal("expected Connect to fail when probe fails")
	}
	if !messaging.IsConnectionError(err) {
		t.Fatalf("expected connection error, got: %v", err)
	}
	if b.IsConnected() {
		t.Fatal("broker must not report connected after failed probe")
	}
	metrics := b.GetMetrics()
	if metrics.ConnectionFailures == 0 {
		t.Fatal("expected connection failure to be recorded in metrics")
	}
}

func TestKafkaBrokerConnect_RealProbeUnreachableEndpoint(t *testing.T) {
	// Use the real prober against a port that is guaranteed to refuse
	// connections; keep the timeout small to keep the test fast.
	b := newKafkaBroker(&messaging.BrokerConfig{
		Type:      messaging.BrokerTypeKafka,
		Endpoints: []string{"127.0.0.1:1"},
		Connection: messaging.ConnectionConfig{
			Timeout: 500 * time.Millisecond,
		},
	})

	err := b.Connect(context.Background())
	if err == nil {
		t.Fatal("expected Connect to fail against unreachable endpoint")
	}
	if !messaging.IsConnectionError(err) {
		t.Fatalf("expected connection error, got: %v", err)
	}
}

func TestKafkaBrokerHealthCheck_NotConnectedIsDegraded(t *testing.T) {
	b := newKafkaBroker(&messaging.BrokerConfig{
		Type:      messaging.BrokerTypeKafka,
		Endpoints: []string{"localhost:9092"},
	})

	status, err := b.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}
	if status.Status != messaging.HealthStatusDegraded {
		t.Fatalf("expected degraded status when not connected, got %s", status.Status)
	}
	if status.Details["connected"] != false {
		t.Fatalf("expected connected=false detail, got %v", status.Details["connected"])
	}
}

func TestKafkaBrokerHealthCheck_ProbeSuccessAndFailure(t *testing.T) {
	stubConnectivityProber(t, nil)

	b := newKafkaBroker(&messaging.BrokerConfig{
		Type:      messaging.BrokerTypeKafka,
		Endpoints: []string{"localhost:9092"},
	})
	if err := b.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	status, err := b.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}
	if status.Status != messaging.HealthStatusHealthy {
		t.Fatalf("expected healthy status, got %s", status.Status)
	}

	// Swap in a failing probe to simulate the broker becoming unreachable.
	stubConnectivityProber(t, errors.New("metadata request failed"))
	status, err = b.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}
	if status.Status != messaging.HealthStatusUnhealthy {
		t.Fatalf("expected unhealthy status after probe failure, got %s", status.Status)
	}
	if _, ok := status.Details["probe_error"]; !ok {
		t.Fatal("expected probe_error detail on failed probe")
	}
}
