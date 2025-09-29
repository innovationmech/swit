package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/resilience"
)

func main() {
	cfg := resilience.Config{
		MaxRetries:    2,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      time.Second,
		Multiplier:    2.0,
		Strategy:      resilience.StrategyJittered,
		JitterPercent: 10,
	}

	exec, err := resilience.NewExecutor(cfg,
		resilience.WithOnRetry(func(attempt int, err error, delay time.Duration) {
			fmt.Printf("retry #%d after %v: %v\n", attempt, delay, err)
		}),
	)
	if err != nil {
		panic(err)
	}

	calls := 0
	op := func() error {
		calls++
		if calls < 3 {
			return errors.New("temporary failure")
		}
		fmt.Println("success on attempt", calls)
		return nil
	}

	if err := exec.Do(context.Background(), op); err != nil {
		fmt.Println("operation failed:", err)
	}
}
