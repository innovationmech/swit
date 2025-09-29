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
