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

package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecutor_Do_Fixed_NoRetry(t *testing.T) {
	cfg := Config{MaxRetries: 0, InitialDelay: 10 * time.Millisecond, Strategy: StrategyFixed, Multiplier: 1.0}
	ex, err := NewExecutor(cfg)
	if err != nil {
		t.Fatalf("new executor error: %v", err)
	}

	calls := 0
	op := func() error {
		calls++
		return errors.New("fail")
	}

	if err := ex.Do(context.Background(), op); err == nil {
		t.Fatalf("expected error, got nil")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestExecutor_Do_Exponential_WithJitter(t *testing.T) {
	cfg := Config{MaxRetries: 2, InitialDelay: 5 * time.Millisecond, MaxDelay: 50 * time.Millisecond, Multiplier: 2.0, Strategy: StrategyJittered, JitterPercent: 20}
	ex, err := NewExecutor(cfg)
	if err != nil {
		t.Fatalf("new executor error: %v", err)
	}

	calls := 0
	op := func() error {
		calls++
		if calls < 3 {
			return errors.New("temp")
		}
		return nil
	}

	start := time.Now()
	if err := ex.Do(context.Background(), op); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)
	if calls != 3 {
		t.Fatalf("expected 3 calls (2 retries), got %d", calls)
	}
	if elapsed <= 0 {
		t.Fatalf("elapsed should be > 0, got %v", elapsed)
	}
}

func TestCalculator_Delay_Bounds(t *testing.T) {
	cfg := Config{MaxRetries: 5, InitialDelay: 10 * time.Millisecond, MaxDelay: 300 * time.Millisecond, Multiplier: 3.0, Strategy: StrategyExponential}
	c := NewCalculator(cfg)
	d1 := c.Delay(1)
	d2 := c.Delay(2)
	d3 := c.Delay(3)
	if d1 <= 0 || d2 <= d1 || d3 <= d2 {
		t.Fatalf("ascending delays expected: %v, %v, %v", d1, d2, d3)
	}
	if c.Delay(10) > cfg.MaxDelay {
		t.Fatalf("delay should be capped by MaxDelay")
	}
}
