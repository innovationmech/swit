---
title: Resilience - Retry & Backoff
description: Reusable retry/backoff strategies with jitter and configurable policies
---

This module provides a reusable retry/backoff capability that can be shared across subsystems (messaging, discovery, HTTP clients).

Key components:

- Strategy: FIXED, LINEAR, EXPONENTIAL, JITTERED, NONE
- Config: max retries, initial delay, max delay, multiplier, jitter percent
- Calculator: delay calculation per attempt
- Executor: execute operations with retry and callbacks

Usage example:

```go
ctx := context.Background()
cfg := resilience.Config{
    MaxRetries:    3,
    InitialDelay:  100 * time.Millisecond,
    MaxDelay:      5 * time.Second,
    Multiplier:    2.0,
    Strategy:      resilience.StrategyJittered,
    JitterPercent: 10,
}

exec, _ := resilience.NewExecutor(cfg,
    resilience.WithOnRetry(func(attempt int, err error, delay time.Duration) {
        // log retry
    }),
)

err := exec.Do(ctx, func() error {
    return doWork()
})
```

See `examples/resilience/retry_example` for a runnable sample.


