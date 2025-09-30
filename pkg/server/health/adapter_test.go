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

package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/types"
)

func init() {
	// Initialize logger for tests
	logger.InitLogger()
}

// mockHealthChecker implements HealthChecker for testing
type mockHealthChecker struct {
	name      string
	checkFunc func(context.Context) error
}

func (m *mockHealthChecker) Check(ctx context.Context) error {
	if m.checkFunc != nil {
		return m.checkFunc(ctx)
	}
	return nil
}

func (m *mockHealthChecker) GetName() string {
	return m.name
}

// mockReadinessChecker implements ReadinessChecker for testing
type mockReadinessChecker struct {
	name      string
	checkFunc func(context.Context) error
}

func (m *mockReadinessChecker) CheckReadiness(ctx context.Context) error {
	if m.checkFunc != nil {
		return m.checkFunc(ctx)
	}
	return nil
}

func (m *mockReadinessChecker) GetName() string {
	return m.name
}

// mockLivenessChecker implements LivenessChecker for testing
type mockLivenessChecker struct {
	name      string
	checkFunc func(context.Context) error
}

func (m *mockLivenessChecker) CheckLiveness(ctx context.Context) error {
	if m.checkFunc != nil {
		return m.checkFunc(ctx)
	}
	return nil
}

func (m *mockLivenessChecker) GetName() string {
	return m.name
}

func TestNewHealthCheckAdapter(t *testing.T) {
	tests := []struct {
		name   string
		config *HealthCheckConfig
	}{
		{
			name:   "with default config",
			config: nil,
		},
		{
			name: "with custom config",
			config: &HealthCheckConfig{
				Timeout:       10 * time.Second,
				Interval:      60 * time.Second,
				MaxConcurrent: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewHealthCheckAdapter(tt.config)
			if adapter == nil {
				t.Fatal("expected non-nil adapter")
			}

			if adapter.config == nil {
				t.Error("expected non-nil config")
			}

			if adapter.healthCheckers == nil {
				t.Error("expected non-nil healthCheckers map")
			}
		})
	}
}

func TestHealthCheckAdapter_RegisterHealthChecker(t *testing.T) {
	adapter := NewHealthCheckAdapter(nil)

	checker := &mockHealthChecker{name: "test-checker"}
	err := adapter.RegisterHealthChecker(checker)
	if err != nil {
		t.Fatalf("failed to register health checker: %v", err)
	}

	// Try to register the same checker again
	err = adapter.RegisterHealthChecker(checker)
	if err == nil {
		t.Error("expected error when registering duplicate checker")
	}
}

func TestHealthCheckAdapter_CheckHealth(t *testing.T) {
	tests := []struct {
		name           string
		checkers       []HealthChecker
		expectedStatus string
	}{
		{
			name: "all checks pass",
			checkers: []HealthChecker{
				&mockHealthChecker{name: "checker1"},
				&mockHealthChecker{name: "checker2"},
			},
			expectedStatus: types.HealthStatusHealthy,
		},
		{
			name: "one check fails",
			checkers: []HealthChecker{
				&mockHealthChecker{name: "checker1"},
				&mockHealthChecker{
					name: "checker2",
					checkFunc: func(ctx context.Context) error {
						return errors.New("check failed")
					},
				},
			},
			expectedStatus: types.HealthStatusUnhealthy,
		},
		{
			name:           "no checkers registered",
			checkers:       []HealthChecker{},
			expectedStatus: types.HealthStatusHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewHealthCheckAdapter(nil)

			for _, checker := range tt.checkers {
				if err := adapter.RegisterHealthChecker(checker); err != nil {
					t.Fatalf("failed to register checker: %v", err)
				}
			}

			ctx := context.Background()
			status, err := adapter.CheckHealth(ctx)
			if err != nil {
				t.Fatalf("CheckHealth returned error: %v", err)
			}

			if status.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, status.Status)
			}
		})
	}
}

func TestHealthCheckAdapter_CheckReadiness(t *testing.T) {
	tests := []struct {
		name           string
		checkers       []ReadinessChecker
		expectedStatus string
	}{
		{
			name: "all checks pass",
			checkers: []ReadinessChecker{
				&mockReadinessChecker{name: "ready1"},
				&mockReadinessChecker{name: "ready2"},
			},
			expectedStatus: types.HealthStatusHealthy,
		},
		{
			name: "one check fails",
			checkers: []ReadinessChecker{
				&mockReadinessChecker{name: "ready1"},
				&mockReadinessChecker{
					name: "ready2",
					checkFunc: func(ctx context.Context) error {
						return errors.New("not ready")
					},
				},
			},
			expectedStatus: types.HealthStatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewHealthCheckAdapter(nil)

			for _, checker := range tt.checkers {
				if err := adapter.RegisterReadinessChecker(checker); err != nil {
					t.Fatalf("failed to register readiness checker: %v", err)
				}
			}

			ctx := context.Background()
			status, err := adapter.CheckReadiness(ctx)
			if err != nil {
				t.Fatalf("CheckReadiness returned error: %v", err)
			}

			if status.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, status.Status)
			}
		})
	}
}

func TestHealthCheckAdapter_CheckLiveness(t *testing.T) {
	tests := []struct {
		name           string
		checkers       []LivenessChecker
		expectedStatus string
	}{
		{
			name: "all checks pass",
			checkers: []LivenessChecker{
				&mockLivenessChecker{name: "live1"},
				&mockLivenessChecker{name: "live2"},
			},
			expectedStatus: types.HealthStatusHealthy,
		},
		{
			name: "one check fails",
			checkers: []LivenessChecker{
				&mockLivenessChecker{name: "live1"},
				&mockLivenessChecker{
					name: "live2",
					checkFunc: func(ctx context.Context) error {
						return errors.New("not alive")
					},
				},
			},
			expectedStatus: types.HealthStatusUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewHealthCheckAdapter(nil)

			for _, checker := range tt.checkers {
				if err := adapter.RegisterLivenessChecker(checker); err != nil {
					t.Fatalf("failed to register liveness checker: %v", err)
				}
			}

			ctx := context.Background()
			status, err := adapter.CheckLiveness(ctx)
			if err != nil {
				t.Fatalf("CheckLiveness returned error: %v", err)
			}

			if status.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, status.Status)
			}
		})
	}
}

func TestHealthCheckAdapter_CheckHealthWithTimeout(t *testing.T) {
	config := &HealthCheckConfig{
		Timeout:       100 * time.Millisecond,
		Interval:      30 * time.Second,
		MaxConcurrent: 10,
	}
	adapter := NewHealthCheckAdapter(config)

	// Register a checker that takes longer than the timeout
	slowChecker := &mockHealthChecker{
		name: "slow-checker",
		checkFunc: func(ctx context.Context) error {
			select {
			case <-time.After(200 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	if err := adapter.RegisterHealthChecker(slowChecker); err != nil {
		t.Fatalf("failed to register checker: %v", err)
	}

	ctx := context.Background()
	status, err := adapter.CheckHealth(ctx)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}

	// The slow checker should timeout and be marked as unhealthy
	if status.Status != types.HealthStatusUnhealthy {
		t.Errorf("expected status %s, got %s", types.HealthStatusUnhealthy, status.Status)
	}
}

func TestHealthCheckAdapter_GetLastResult(t *testing.T) {
	adapter := NewHealthCheckAdapter(nil)

	checker := &mockHealthChecker{name: "test-checker"}
	if err := adapter.RegisterHealthChecker(checker); err != nil {
		t.Fatalf("failed to register checker: %v", err)
	}

	ctx := context.Background()
	_, err := adapter.CheckHealth(ctx)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}

	// Get the last result
	result, exists := adapter.GetLastResult("test-checker")
	if !exists {
		t.Error("expected cached result to exist")
	}

	if result.Name != "test-checker" {
		t.Errorf("expected result name %s, got %s", "test-checker", result.Name)
	}
}

func TestHealthCheckAdapter_ConcurrentChecks(t *testing.T) {
	adapter := NewHealthCheckAdapter(nil)

	// Register multiple checkers
	for i := 0; i < 10; i++ {
		checker := &mockHealthChecker{
			name: "checker-" + string(rune(i)),
			checkFunc: func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
		}
		if err := adapter.RegisterHealthChecker(checker); err != nil {
			t.Fatalf("failed to register checker: %v", err)
		}
	}

	ctx := context.Background()
	start := time.Now()
	status, err := adapter.CheckHealth(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}

	if status.Status != types.HealthStatusHealthy {
		t.Errorf("expected status %s, got %s", types.HealthStatusHealthy, status.Status)
	}

	// Checks should run concurrently, so total time should be much less than 10 * 10ms
	if duration > 150*time.Millisecond {
		t.Errorf("checks took too long, expected concurrent execution: %v", duration)
	}
}
