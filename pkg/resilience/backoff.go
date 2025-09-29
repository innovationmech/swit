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
