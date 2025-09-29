package resilience

import (
	"fmt"
	"math"
	"time"
)

// Config 定义重试与退避的通用配置。
//
// 该配置可被不同子系统（如发现、消息、HTTP 客户端）复用，
// 并通过 `Strategy` 与 `Multiplier`/`JitterPercent` 等参数控制退避曲线。
type Config struct {
	// MaxRetries 最大重试次数（不含首次尝试）。<=0 表示不重试。
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// InitialDelay 首次重试的基础延迟。
	InitialDelay time.Duration `json:"initial_delay" yaml:"initial_delay"`

	// MaxDelay 单次延迟上限（0 表示无限制）。
	MaxDelay time.Duration `json:"max_delay" yaml:"max_delay"`

	// Multiplier 指数/线性退避的倍率系数。
	Multiplier float64 `json:"multiplier" yaml:"multiplier"`

	// Strategy 退避策略。
	Strategy Strategy `json:"strategy" yaml:"strategy"`

	// JitterPercent 抖动百分比区间 [0,100]，按 baseDelay * (±percent/100) 计算。
	JitterPercent float64 `json:"jitter_percent" yaml:"jitter_percent"`
}

// Default 返回一套通用的稳健默认值。
func Default() Config {
	return Config{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      30 * time.Second,
		Multiplier:    2.0,
		Strategy:      StrategyExponential,
		JitterPercent: 10.0,
	}
}

// Validate 校验配置合法性。
func (c *Config) Validate() error {
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}
	if c.InitialDelay < 0 {
		return fmt.Errorf("initial_delay cannot be negative")
	}
	if c.MaxDelay < 0 {
		return fmt.Errorf("max_delay cannot be negative")
	}
	if c.Multiplier <= 0 {
		return fmt.Errorf("multiplier must be positive")
	}
	if math.IsNaN(c.Multiplier) || math.IsInf(c.Multiplier, 0) {
		return fmt.Errorf("multiplier must be finite")
	}
	if c.JitterPercent < 0 || c.JitterPercent > 100 {
		return fmt.Errorf("jitter_percent must be between 0 and 100")
	}
	return nil
}
