package resilience

import (
	"math"
	"math/rand"
	"time"
)

// Calculator 基于 Config 计算每次重试的延迟。
type Calculator struct {
	cfg Config
	rng *rand.Rand
}

// NewCalculator 创建一个延迟计算器。
func NewCalculator(cfg Config) *Calculator {
	return &Calculator{
		cfg: cfg,
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Delay 计算给定尝试次数的延迟（attempt >= 1 表示第 1 次重试）。
func (c *Calculator) Delay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	base := c.cfg.InitialDelay
	switch c.cfg.Strategy {
	case StrategyFixed:
		// 固定延迟
		// base 保持不变
	case StrategyLinear:
		base = time.Duration(float64(base) * float64(attempt))
	case StrategyExponential, StrategyJittered:
		factor := math.Pow(c.cfg.Multiplier, float64(attempt))
		base = time.Duration(float64(base) * factor)
	case StrategyNone:
		return 0
	default:
		// 未知策略，回退为固定
	}

	if c.cfg.MaxDelay > 0 && base > c.cfg.MaxDelay {
		base = c.cfg.MaxDelay
	}

	if c.cfg.Strategy == StrategyJittered || c.cfg.JitterPercent > 0 {
		base = c.addJitter(base)
		if c.cfg.MaxDelay > 0 && base > c.cfg.MaxDelay {
			base = c.cfg.MaxDelay
		}
	}

	if base < 0 {
		base = 0
	}
	return base
}

func (c *Calculator) addJitter(base time.Duration) time.Duration {
	if c.cfg.JitterPercent <= 0 {
		return base
	}

	p := c.cfg.JitterPercent
	if p > 100 {
		p = 100
	}
	// 在 [1-p%, 1+p%] 区间随机缩放
	min := 1.0 - p/100.0
	max := 1.0 + p/100.0
	factor := min + (max-min)*c.rng.Float64()
	return time.Duration(float64(base) * factor)
}
