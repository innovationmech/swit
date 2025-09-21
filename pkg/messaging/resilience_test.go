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

func TestResilienceExecutor_CircuitOpensOnFailuresAndRecovers(t *testing.T) {
	cfg := &ResilienceConfig{
		EnableCircuitBreaker: true,
		EnableRetry:          false,
		CircuitBreakerConfig: &CircuitBreakerConfig{
			Name:             "resilience-test",
			MaxRequests:      1,
			Interval:         0,
			Timeout:          200 * time.Millisecond,
			FailureThreshold: 3,
			SuccessThreshold: 1,
			ErrorClassifier:  nil, // treat any error as failure
		},
	}

	exec, err := NewResilienceExecutor(cfg)
	if err != nil {
		t.Fatalf("NewResilienceExecutor: %v", err)
	}

	var executed int
	failing := func() error {
		executed++
		return errors.New("forced")
	}

	// Trip the circuit
	for i := 0; i < 3; i++ {
		if err := exec.Execute(context.Background(), failing); err == nil {
			t.Fatalf("expected failure on attempt %d", i+1)
		}
	}

	// Subsequent call should be short-circuited quickly (circuit open)
	start := time.Now()
	if err := exec.Execute(context.Background(), failing); err == nil {
		t.Fatalf("expected short-circuited error when circuit is open")
	}
	if time.Since(start) > 50*time.Millisecond {
		t.Fatalf("short-circuited call took too long; circuit may not be open")
	}

	// Wait for half-open window, then succeed
	time.Sleep(220 * time.Millisecond)
	executedBefore := executed
	if err := exec.Execute(context.Background(), func() error { executed++; return nil }); err != nil {
		t.Fatalf("expected success after half-open, got %v", err)
	}
	if executed <= executedBefore {
		t.Fatalf("expected operation to execute in half-open state")
	}
}
