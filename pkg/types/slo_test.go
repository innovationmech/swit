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

package types

import (
	"testing"
	"time"
)

func TestDefaultSLOConfig(t *testing.T) {
	config := DefaultSLOConfig()

	if config == nil {
		t.Fatal("DefaultSLOConfig returned nil")
	}

	if config.Enabled {
		t.Error("Expected Enabled to be false by default")
	}

	if len(config.Targets) == 0 {
		t.Error("Expected default targets to be provided")
	}

	if config.UpdateInterval == 0 {
		t.Error("Expected UpdateInterval to be set")
	}

	if config.RetentionPeriod == 0 {
		t.Error("Expected RetentionPeriod to be set")
	}
}

func TestDefaultSLOTargets(t *testing.T) {
	targets := DefaultSLOTargets()

	if len(targets) == 0 {
		t.Fatal("Expected at least one default SLO target")
	}

	// Check availability target
	var foundAvailability, foundLatency, foundErrorRate bool
	for _, target := range targets {
		switch target.Type {
		case SLOTypeAvailability:
			foundAvailability = true
			if target.Target <= 0 || target.Target > 100 {
				t.Errorf("Invalid availability target: %.2f", target.Target)
			}
		case SLOTypeLatency:
			foundLatency = true
			if target.Target <= 0 || target.Target > 100 {
				t.Errorf("Invalid latency target: %.2f", target.Target)
			}
		case SLOTypeErrorRate:
			foundErrorRate = true
			if target.Target <= 0 || target.Target > 100 {
				t.Errorf("Invalid error rate target: %.2f", target.Target)
			}
		}
	}

	if !foundAvailability {
		t.Error("Expected availability SLO in defaults")
	}
	if !foundLatency {
		t.Error("Expected latency SLO in defaults")
	}
	if !foundErrorRate {
		t.Error("Expected error rate SLO in defaults")
	}
}

func TestNewSLOMonitor(t *testing.T) {
	config := DefaultSLOConfig()
	monitor := NewSLOMonitor(config)

	if monitor == nil {
		t.Fatal("NewSLOMonitor returned nil")
	}

	if monitor.config != config {
		t.Error("Expected config to be set")
	}

	if len(monitor.statuses) == 0 {
		t.Error("Expected statuses to be initialized")
	}
}

func TestSLOMonitor_UpdateStatus(t *testing.T) {
	tests := []struct {
		name           string
		targetName     string
		currentValue   float64
		expectedStatus SLOComplianceStatus
		shouldError    bool
	}{
		{
			name:           "healthy status",
			targetName:     "availability_99_9",
			currentValue:   99.95,
			expectedStatus: SLOStatusHealthy,
			shouldError:    false,
		},
		{
			name:           "warning status",
			targetName:     "availability_99_9",
			currentValue:   99.7,
			expectedStatus: SLOStatusWarning,
			shouldError:    false,
		},
		{
			name:           "critical status",
			targetName:     "availability_99_9",
			currentValue:   98.5,
			expectedStatus: SLOStatusCritical,
			shouldError:    false,
		},
		{
			name:           "non-existent target",
			targetName:     "non_existent",
			currentValue:   99.0,
			expectedStatus: SLOStatusUnknown,
			shouldError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultSLOConfig()
			monitor := NewSLOMonitor(config)

			err := monitor.UpdateStatus(tt.targetName, tt.currentValue)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			status, err := monitor.GetStatus(tt.targetName)
			if err != nil {
				t.Errorf("Failed to get status: %v", err)
				return
			}

			if status.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, status.Status)
			}

			if status.CurrentValue != tt.currentValue {
				t.Errorf("Expected current value %.2f, got %.2f", tt.currentValue, status.CurrentValue)
			}
		})
	}
}

func TestSLOMonitor_GetStatus(t *testing.T) {
	config := DefaultSLOConfig()
	monitor := NewSLOMonitor(config)

	// Update a status
	targetName := "availability_99_9"
	err := monitor.UpdateStatus(targetName, 99.95)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Get the status
	status, err := monitor.GetStatus(targetName)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status == nil {
		t.Fatal("Expected non-nil status")
	}

	if status.Target.Name != targetName {
		t.Errorf("Expected target name %s, got %s", targetName, status.Target.Name)
	}

	// Try to get non-existent status
	_, err = monitor.GetStatus("non_existent")
	if err == nil {
		t.Error("Expected error for non-existent target")
	}
}

func TestSLOMonitor_GetAllStatuses(t *testing.T) {
	config := DefaultSLOConfig()
	monitor := NewSLOMonitor(config)

	statuses := monitor.GetAllStatuses()

	if len(statuses) != len(config.Targets) {
		t.Errorf("Expected %d statuses, got %d", len(config.Targets), len(statuses))
	}

	for _, status := range statuses {
		if status == nil {
			t.Error("Expected non-nil status")
		}
		if status.Target == nil {
			t.Error("Expected non-nil target")
		}
	}
}

func TestSLOMonitor_IsHealthy(t *testing.T) {
	tests := []struct {
		name            string
		statusUpdates   map[string]float64
		expectedHealthy bool
	}{
		{
			name: "all healthy",
			statusUpdates: map[string]float64{
				"availability_99_9":          99.95,
				"latency_p95_500ms":          96.0,
				"error_rate_below_1_percent": 99.5,
			},
			expectedHealthy: true,
		},
		{
			name: "one warning",
			statusUpdates: map[string]float64{
				"availability_99_9":          99.95,
				"latency_p95_500ms":          92.0, // Warning
				"error_rate_below_1_percent": 99.5,
			},
			expectedHealthy: false,
		},
		{
			name: "one critical",
			statusUpdates: map[string]float64{
				"availability_99_9":          98.0, // Critical
				"latency_p95_500ms":          96.0,
				"error_rate_below_1_percent": 99.5,
			},
			expectedHealthy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultSLOConfig()
			monitor := NewSLOMonitor(config)

			// Update all statuses
			for name, value := range tt.statusUpdates {
				err := monitor.UpdateStatus(name, value)
				if err != nil {
					t.Logf("Warning: Failed to update status for %s: %v", name, err)
				}
			}

			isHealthy := monitor.IsHealthy()

			if isHealthy != tt.expectedHealthy {
				t.Errorf("Expected IsHealthy to be %v, got %v", tt.expectedHealthy, isHealthy)
			}
		})
	}
}

func TestSLOMonitor_GetErrorBudgetStatus(t *testing.T) {
	config := DefaultSLOConfig()
	monitor := NewSLOMonitor(config)

	// Update statuses
	monitor.UpdateStatus("availability_99_9", 99.95)
	monitor.UpdateStatus("latency_p95_500ms", 96.0)

	budgets := monitor.GetErrorBudgetStatus()

	if len(budgets) == 0 {
		t.Error("Expected error budgets to be returned")
	}

	for name, budget := range budgets {
		if name == "" {
			t.Error("Expected non-empty SLO name")
		}
		t.Logf("SLO %s: Error budget remaining: %.2f%%", name, budget)
	}
}

func TestSLOMonitor_GetViolations(t *testing.T) {
	config := DefaultSLOConfig()
	monitor := NewSLOMonitor(config)

	// Update status to healthy first
	monitor.UpdateStatus("availability_99_9", 99.95)

	// Now update to warning (should create a violation)
	monitor.UpdateStatus("availability_99_9", 99.3)

	// Get violations from the last minute
	since := time.Now().Add(-1 * time.Minute)
	violations := monitor.GetViolations(since)

	if len(violations) == 0 {
		t.Error("Expected at least one violation")
	}

	for _, v := range violations {
		if v.Timestamp.IsZero() {
			t.Error("Expected non-zero violation timestamp")
		}
		if v.Message == "" {
			t.Error("Expected non-empty violation message")
		}
		t.Logf("Violation: %s at %v (value: %.2f)", v.Message, v.Timestamp, v.Value)
	}
}

func TestSLOMonitor_ErrorBudgetCalculation(t *testing.T) {
	config := &SLOConfig{
		Enabled: true,
		Targets: []SLOTarget{
			{
				Name:              "test_slo",
				Type:              SLOTypeAvailability,
				Target:            99.9,
				Window:            30 * 24 * time.Hour,
				WarningThreshold:  99.5,
				CriticalThreshold: 99.0,
			},
		},
	}

	monitor := NewSLOMonitor(config)

	tests := []struct {
		name           string
		currentValue   float64
		expectedBudget float64
	}{
		{
			name:           "perfect performance",
			currentValue:   100.0,
			expectedBudget: 1.0, // (99.9 - 99.0)
		},
		{
			name:           "at target",
			currentValue:   99.9,
			expectedBudget: 0.9, // (99.9 - 99.0)
		},
		{
			name:           "at critical",
			currentValue:   99.0,
			expectedBudget: 0.0, // (99.0 - 99.0)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := monitor.UpdateStatus("test_slo", tt.currentValue)
			if err != nil {
				t.Fatalf("Failed to update status: %v", err)
			}

			status, err := monitor.GetStatus("test_slo")
			if err != nil {
				t.Fatalf("Failed to get status: %v", err)
			}

			if status.ErrorBudget < tt.expectedBudget-0.1 || status.ErrorBudget > tt.expectedBudget+0.1 {
				t.Errorf("Expected error budget around %.2f, got %.2f", tt.expectedBudget, status.ErrorBudget)
			}
		})
	}
}

func TestSLOMonitor_ConcurrentAccess(t *testing.T) {
	config := DefaultSLOConfig()
	monitor := NewSLOMonitor(config)

	// Concurrent updates
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(val float64) {
			monitor.UpdateStatus("availability_99_9", val)
			done <- true
		}(99.0 + float64(i)*0.05)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and should have a valid status
	status, err := monitor.GetStatus("availability_99_9")
	if err != nil {
		t.Errorf("Failed to get status after concurrent updates: %v", err)
	}

	if status == nil {
		t.Error("Expected non-nil status after concurrent updates")
	}
}
