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
)

func TestNewBasicHealthChecker(t *testing.T) {
	checker := NewBasicHealthChecker("test", nil)
	if checker == nil {
		t.Fatal("expected non-nil checker")
	}

	if checker.GetName() != "test" {
		t.Errorf("expected name 'test', got %s", checker.GetName())
	}

	// Check with nil function should pass
	ctx := context.Background()
	if err := checker.Check(ctx); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestBasicHealthChecker_CheckWithFunction(t *testing.T) {
	testErr := errors.New("check failed")
	checker := NewBasicHealthChecker("test", func(ctx context.Context) error {
		return testErr
	})

	ctx := context.Background()
	err := checker.Check(ctx)
	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}
}

func TestServiceReadinessProbe(t *testing.T) {
	probe := NewServiceReadinessProbe("test-service", 100*time.Millisecond)

	ctx := context.Background()

	// Should not be ready initially
	err := probe.CheckReadiness(ctx)
	if err == nil {
		t.Error("expected error when not ready")
	}

	// Mark as ready
	probe.SetReady(true)

	// Wait for min start time
	time.Sleep(150 * time.Millisecond)

	// Should be ready now
	err = probe.CheckReadiness(ctx)
	if err != nil {
		t.Errorf("expected nil error when ready, got %v", err)
	}

	// Mark as not ready
	probe.SetReady(false)

	// Should not be ready again
	err = probe.CheckReadiness(ctx)
	if err == nil {
		t.Error("expected error after marking not ready")
	}
}

func TestServiceReadinessProbe_IsReady(t *testing.T) {
	probe := NewServiceReadinessProbe("test-service", 0)

	if probe.IsReady() {
		t.Error("expected IsReady to be false initially")
	}

	probe.SetReady(true)

	if !probe.IsReady() {
		t.Error("expected IsReady to be true after SetReady(true)")
	}
}

func TestServiceLivenessProbe(t *testing.T) {
	probe := NewServiceLivenessProbe("test-service", 100*time.Millisecond)

	ctx := context.Background()

	// Should be alive initially
	err := probe.CheckLiveness(ctx)
	if err != nil {
		t.Errorf("expected nil error when alive, got %v", err)
	}

	// Mark as not running
	probe.SetRunning(false)

	// Should not be alive
	err = probe.CheckLiveness(ctx)
	if err == nil {
		t.Error("expected error when not running")
	}

	// Mark as running again
	probe.SetRunning(true)

	// Should be alive again
	err = probe.CheckLiveness(ctx)
	if err != nil {
		t.Errorf("expected nil error when alive, got %v", err)
	}
}

func TestServiceLivenessProbe_Timeout(t *testing.T) {
	probe := NewServiceLivenessProbe("test-service", 50*time.Millisecond)

	ctx := context.Background()

	// Should be alive initially
	err := probe.CheckLiveness(ctx)
	if err != nil {
		t.Errorf("expected nil error when alive, got %v", err)
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Should timeout
	err = probe.CheckLiveness(ctx)
	if err == nil {
		t.Error("expected error after timeout")
	}

	// Ping to update last ping time
	probe.Ping()

	// Should be alive again
	err = probe.CheckLiveness(ctx)
	if err != nil {
		t.Errorf("expected nil error after ping, got %v", err)
	}
}

func TestServiceLivenessProbe_Heartbeat(t *testing.T) {
	probe := NewServiceLivenessProbe("test-service", 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start heartbeat
	probe.StartHeartbeat(ctx, 20*time.Millisecond)

	// Wait longer than the liveness timeout
	time.Sleep(150 * time.Millisecond)

	// Should still be alive due to heartbeat
	checkCtx := context.Background()
	err := probe.CheckLiveness(checkCtx)
	if err != nil {
		t.Errorf("expected nil error with heartbeat running, got %v", err)
	}

	// Stop heartbeat
	cancel()

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should timeout now
	err = probe.CheckLiveness(checkCtx)
	if err == nil {
		t.Error("expected error after heartbeat stopped and timeout")
	}
}

func TestDependencyHealthChecker(t *testing.T) {
	testErr := errors.New("dependency check failed")
	depChecker := &mockHealthChecker{
		name: "dep",
		checkFunc: func(ctx context.Context) error {
			return testErr
		},
	}

	checker := NewDependencyHealthChecker("my-dependency", depChecker)

	if checker.GetName() != "my-dependency" {
		t.Errorf("expected name 'my-dependency', got %s", checker.GetName())
	}

	ctx := context.Background()
	err := checker.Check(ctx)
	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}
}

func TestCompositeHealthChecker(t *testing.T) {
	tests := []struct {
		name      string
		checkers  []HealthChecker
		expectErr bool
	}{
		{
			name: "all checks pass",
			checkers: []HealthChecker{
				&mockHealthChecker{name: "check1"},
				&mockHealthChecker{name: "check2"},
			},
			expectErr: false,
		},
		{
			name: "one check fails",
			checkers: []HealthChecker{
				&mockHealthChecker{name: "check1"},
				&mockHealthChecker{
					name: "check2",
					checkFunc: func(ctx context.Context) error {
						return errors.New("failed")
					},
				},
			},
			expectErr: true,
		},
		{
			name:      "no checkers",
			checkers:  []HealthChecker{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewCompositeHealthChecker("composite", tt.checkers...)

			ctx := context.Background()
			err := checker.Check(ctx)

			if tt.expectErr && err == nil {
				t.Error("expected error but got nil")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("expected nil error but got %v", err)
			}
		})
	}
}

func TestCompositeHealthChecker_AddChecker(t *testing.T) {
	checker := NewCompositeHealthChecker("composite")

	checker.AddChecker(&mockHealthChecker{name: "check1"})
	checker.AddChecker(&mockHealthChecker{name: "check2"})

	ctx := context.Background()
	err := checker.Check(ctx)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
