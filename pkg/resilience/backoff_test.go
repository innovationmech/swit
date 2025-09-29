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
	"testing"
	"time"
)

func TestBackoff_Fixed(t *testing.T) {
	cfg := Config{MaxRetries: 3, InitialDelay: 10 * time.Millisecond, Strategy: StrategyFixed}
	c := NewCalculator(cfg)
	if c.Delay(1) != 10*time.Millisecond {
		t.Fatalf("fixed delay mismatch")
	}
	if c.Delay(2) != 10*time.Millisecond {
		t.Fatalf("fixed delay mismatch")
	}
}

func TestBackoff_Linear(t *testing.T) {
	cfg := Config{MaxRetries: 3, InitialDelay: 5 * time.Millisecond, Strategy: StrategyLinear}
	c := NewCalculator(cfg)
	if c.Delay(1) <= 0 || c.Delay(2) <= c.Delay(1) {
		t.Fatalf("linear delay not increasing")
	}
}
