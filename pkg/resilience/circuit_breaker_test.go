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

func TestCircuitBreakerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  CircuitBreakerConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultCircuitBreakerConfig(),
			wantErr: false,
		},
		{
			name: "empty name",
			config: CircuitBreakerConfig{
				Name:                 "",
				MaxRequests:          1,
				Interval:             time.Second,
				Timeout:              time.Second,
				FailureRateThreshold: 0.5,
				MinimumRequests:      1,
			},
			wantErr: true,
		},
		{
			name: "zero max requests",
			config: CircuitBreakerConfig{
				Name:                 "test",
				MaxRequests:          0,
				Interval:             time.Second,
				Timeout:              time.Second,
				FailureRateThreshold: 0.5,
				MinimumRequests:      1,
			},
			wantErr: true,
		},
		{
			name: "negative interval",
			config: CircuitBreakerConfig{
				Name:                 "test",
				MaxRequests:          1,
				Interval:             -time.Second,
				Timeout:              time.Second,
				FailureRateThreshold: 0.5,
				MinimumRequests:      1,
			},
			wantErr: true,
		},
		{
			name: "invalid failure rate",
			config: CircuitBreakerConfig{
				Name:                 "test",
				MaxRequests:          1,
				Interval:             time.Second,
				Timeout:              time.Second,
				FailureRateThreshold: 1.5,
				MinimumRequests:      1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCircuitBreaker_NewCircuitBreaker(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := DefaultCircuitBreakerConfig()
		cb, err := NewCircuitBreaker(config)
		if err != nil {
			t.Fatalf("NewCircuitBreaker() error = %v", err)
		}
		if cb == nil {
			t.Fatal("NewCircuitBreaker() returned nil")
		}
		if cb.GetState() != StateClosed {
			t.Errorf("initial state = %v, want %v", cb.GetState(), StateClosed)
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		config := CircuitBreakerConfig{
			Name: "", // invalid
		}
		_, err := NewCircuitBreaker(config)
		if err == nil {
			t.Error("NewCircuitBreaker() expected error, got nil")
		}
	})
}

func TestCircuitBreaker_Execute_SuccessPath(t *testing.T) {
	config := CircuitBreakerConfig{
		Name:                 "test",
		MaxRequests:          1,
		Interval:             time.Second,
		Timeout:              time.Second,
		FailureThreshold:     3,
		FailureRateThreshold: 0.5,
		MinimumRequests:      1,
	}

	cb, err := NewCircuitBreaker(config)
	if err != nil {
		t.Fatalf("NewCircuitBreaker() error = %v", err)
	}

	ctx := context.Background()
	successOp := func() error {
		return nil
	}

	err = cb.Execute(ctx, successOp)
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("state = %v, want %v", cb.GetState(), StateClosed)
	}

	counts := cb.GetCounts()
	if counts.TotalSuccesses != 1 {
		t.Errorf("TotalSuccesses = %d, want 1", counts.TotalSuccesses)
	}
}

func TestCircuitBreaker_Execute_FailureThreshold(t *testing.T) {
	config := CircuitBreakerConfig{
		Name:                 "test",
		MaxRequests:          1,
		Interval:             time.Minute,
		Timeout:              100 * time.Millisecond,
		FailureThreshold:     3,
		FailureRateThreshold: 0.5,
		MinimumRequests:      3,
	}

	cb, err := NewCircuitBreaker(config)
	if err != nil {
		t.Fatalf("NewCircuitBreaker() error = %v", err)
	}

	ctx := context.Background()
	failOp := func() error {
		return errors.New("operation failed")
	}

	// 触发失败阈值
	for i := 0; i < 3; i++ {
		err := cb.Execute(ctx, failOp)
		if err == nil {
			t.Error("Execute() expected error, got nil")
		}
	}

	// 熔断器应该打开
	if cb.GetState() != StateOpen {
		t.Errorf("state = %v, want %v", cb.GetState(), StateOpen)
	}

	// 下一次请求应该被拒绝
	err = cb.Execute(ctx, failOp)
	if !errors.Is(err, ErrOpenState) {
		t.Errorf("Execute() error = %v, want %v", err, ErrOpenState)
	}

	metrics := cb.GetMetrics()
	if metrics.GetTotalRejections() != 1 {
		t.Errorf("TotalRejections = %d, want 1", metrics.GetTotalRejections())
	}
}

func TestCircuitBreaker_HalfOpenState(t *testing.T) {
	config := CircuitBreakerConfig{
		Name:                 "test",
		MaxRequests:          2,
		Interval:             time.Minute,
		Timeout:              50 * time.Millisecond, // 短超时以便快速测试
		FailureThreshold:     2,
		FailureRateThreshold: 0.5,
		MinimumRequests:      2,
	}

	cb, err := NewCircuitBreaker(config)
	if err != nil {
		t.Fatalf("NewCircuitBreaker() error = %v", err)
	}

	ctx := context.Background()
	failOp := func() error {
		return errors.New("operation failed")
	}
	successOp := func() error {
		return nil
	}

	// 触发熔断器打开
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, failOp)
	}

	if cb.GetState() != StateOpen {
		t.Errorf("state = %v, want %v", cb.GetState(), StateOpen)
	}

	// 等待超时进入半开状态
	time.Sleep(100 * time.Millisecond)

	// 半开状态下的第一个请求
	state := cb.GetState()
	if state != StateHalfOpen {
		t.Errorf("state = %v, want %v", state, StateHalfOpen)
	}

	// 半开状态下成功的请求应该关闭熔断器
	for i := 0; i < 2; i++ {
		err := cb.Execute(ctx, successOp)
		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}
	}

	// 熔断器应该关闭
	if cb.GetState() != StateClosed {
		t.Errorf("state = %v, want %v", cb.GetState(), StateClosed)
	}
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	config := CircuitBreakerConfig{
		Name:                 "test",
		MaxRequests:          1,
		Interval:             time.Minute,
		Timeout:              50 * time.Millisecond,
		FailureThreshold:     2,
		FailureRateThreshold: 0.5,
		MinimumRequests:      2,
	}

	cb, err := NewCircuitBreaker(config)
	if err != nil {
		t.Fatalf("NewCircuitBreaker() error = %v", err)
	}

	ctx := context.Background()
	failOp := func() error {
		return errors.New("operation failed")
	}

	// 打开熔断器
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, failOp)
	}

	// 等待进入半开状态
	time.Sleep(100 * time.Millisecond)

	if cb.GetState() != StateHalfOpen {
		t.Errorf("state = %v, want %v", cb.GetState(), StateHalfOpen)
	}

	// 半开状态下失败应该重新打开
	_ = cb.Execute(ctx, failOp)

	if cb.GetState() != StateOpen {
		t.Errorf("state = %v, want %v", cb.GetState(), StateOpen)
	}
}

func TestCircuitBreaker_TooManyRequests(t *testing.T) {
	config := CircuitBreakerConfig{
		Name:                 "test",
		MaxRequests:          1,
		Interval:             time.Minute,
		Timeout:              50 * time.Millisecond,
		FailureThreshold:     2,
		FailureRateThreshold: 0.5,
		MinimumRequests:      2,
	}

	cb, err := NewCircuitBreaker(config)
	if err != nil {
		t.Fatalf("NewCircuitBreaker() error = %v", err)
	}

	ctx := context.Background()
	failOp := func() error {
		return errors.New("operation failed")
	}
	successOp := func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	// 打开熔断器
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, failOp)
	}

	// 等待进入半开状态
	time.Sleep(100 * time.Millisecond)

	// 启动第一个请求但不等待完成
	go func() {
		_ = cb.Execute(ctx, successOp)
	}()

	// 稍微等待确保第一个请求已经开始
	time.Sleep(5 * time.Millisecond)

	// 第二个请求应该被拒绝（超过 MaxRequests）
	err = cb.Execute(ctx, successOp)
	if !errors.Is(err, ErrTooManyRequests) {
		t.Errorf("Execute() error = %v, want %v", err, ErrTooManyRequests)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := CircuitBreakerConfig{
		Name:                 "test",
		MaxRequests:          1,
		Interval:             time.Minute,
		Timeout:              time.Minute,
		FailureThreshold:     2,
		FailureRateThreshold: 0.5,
		MinimumRequests:      2,
	}

	cb, err := NewCircuitBreaker(config)
	if err != nil {
		t.Fatalf("NewCircuitBreaker() error = %v", err)
	}

	ctx := context.Background()
	failOp := func() error {
		return errors.New("operation failed")
	}

	// 打开熔断器
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, failOp)
	}

	if cb.GetState() != StateOpen {
		t.Errorf("state = %v, want %v", cb.GetState(), StateOpen)
	}

	// 重置熔断器
	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("state after reset = %v, want %v", cb.GetState(), StateClosed)
	}

	counts := cb.GetCounts()
	if counts.TotalFailures != 0 {
		t.Errorf("TotalFailures after reset = %d, want 0", counts.TotalFailures)
	}
}

func TestCircuitBreaker_StateCallback(t *testing.T) {
	stateChanges := make([]string, 0)
	config := CircuitBreakerConfig{
		Name:                 "test",
		MaxRequests:          1,
		Interval:             time.Minute,
		Timeout:              50 * time.Millisecond,
		FailureThreshold:     2,
		FailureRateThreshold: 0.5,
		MinimumRequests:      2,
		OnStateChange: func(name string, from State, to State) {
			stateChanges = append(stateChanges, to.String())
		},
	}

	cb, err := NewCircuitBreaker(config)
	if err != nil {
		t.Fatalf("NewCircuitBreaker() error = %v", err)
	}

	ctx := context.Background()
	failOp := func() error {
		return errors.New("operation failed")
	}

	// 打开熔断器
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, failOp)
	}

	// 等待进入半开状态
	time.Sleep(100 * time.Millisecond)
	_ = cb.GetState() // 触发状态检查

	if len(stateChanges) < 2 {
		t.Errorf("state changes = %v, want at least 2", stateChanges)
	}

	if stateChanges[0] != "open" {
		t.Errorf("first state change = %v, want open", stateChanges[0])
	}
}

func TestCircuitBreaker_ExecuteWithResult(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	cb, err := NewCircuitBreaker(config)
	if err != nil {
		t.Fatalf("NewCircuitBreaker() error = %v", err)
	}

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		op := func() (int, error) {
			return 42, nil
		}

		result, err := ExecuteWithResult(ctx, cb, op)
		if err != nil {
			t.Errorf("ExecuteWithResult() error = %v, want nil", err)
		}
		if result != 42 {
			t.Errorf("result = %d, want 42", result)
		}
	})

	t.Run("failure", func(t *testing.T) {
		expectedErr := errors.New("operation failed")
		op := func() (int, error) {
			return 0, expectedErr
		}

		result, err := ExecuteWithResult(ctx, cb, op)
		if !errors.Is(err, expectedErr) {
			t.Errorf("ExecuteWithResult() error = %v, want %v", err, expectedErr)
		}
		if result != 0 {
			t.Errorf("result = %d, want 0", result)
		}
	})
}

func TestCircuitBreaker_ContextCancellation(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	cb, err := NewCircuitBreaker(config)
	if err != nil {
		t.Fatalf("NewCircuitBreaker() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	op := func() error {
		return nil
	}

	err = cb.Execute(ctx, op)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Execute() error = %v, want %v", err, context.Canceled)
	}
}

func TestCounts_FailureRate(t *testing.T) {
	tests := []struct {
		name     string
		counts   Counts
		wantRate float64
	}{
		{
			name:     "no requests",
			counts:   Counts{},
			wantRate: 0,
		},
		{
			name: "50% failure rate",
			counts: Counts{
				Requests:      10,
				TotalFailures: 5,
			},
			wantRate: 0.5,
		},
		{
			name: "100% failure rate",
			counts: Counts{
				Requests:      10,
				TotalFailures: 10,
			},
			wantRate: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate := tt.counts.FailureRate()
			if rate != tt.wantRate {
				t.Errorf("FailureRate() = %v, want %v", rate, tt.wantRate)
			}
		})
	}
}

func TestCircuitBreakerMetrics(t *testing.T) {
	metrics := NewCircuitBreakerMetrics()

	metrics.RecordRequest()
	metrics.RecordSuccess()
	metrics.RecordFailure()
	metrics.RecordRejection()
	metrics.RecordStateChange("open")

	if metrics.GetTotalRequests() != 1 {
		t.Errorf("TotalRequests = %d, want 1", metrics.GetTotalRequests())
	}
	if metrics.GetTotalSuccesses() != 1 {
		t.Errorf("TotalSuccesses = %d, want 1", metrics.GetTotalSuccesses())
	}
	if metrics.GetTotalFailures() != 1 {
		t.Errorf("TotalFailures = %d, want 1", metrics.GetTotalFailures())
	}
	if metrics.GetTotalRejections() != 1 {
		t.Errorf("TotalRejections = %d, want 1", metrics.GetTotalRejections())
	}
	if metrics.GetStateChangesToOpen() != 1 {
		t.Errorf("StateChangesToOpen = %d, want 1", metrics.GetStateChangesToOpen())
	}

	metrics.Reset()
	if metrics.GetTotalRequests() != 0 {
		t.Errorf("TotalRequests after reset = %d, want 0", metrics.GetTotalRequests())
	}
}
