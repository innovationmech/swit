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
	"fmt"
	"sync"
	"time"
)

// SLOType represents the type of Service Level Objective
type SLOType string

const (
	// SLOTypeAvailability measures the uptime/availability of the service
	SLOTypeAvailability SLOType = "availability"
	// SLOTypeLatency measures the response time of requests
	SLOTypeLatency SLOType = "latency"
	// SLOTypeErrorRate measures the percentage of failed requests
	SLOTypeErrorRate SLOType = "error_rate"
	// SLOTypeThroughput measures the requests per second
	SLOTypeThroughput SLOType = "throughput"
	// SLOTypeCustom allows custom SLO definitions
	SLOTypeCustom SLOType = "custom"
)

// SLOTarget represents a specific SLO target with threshold and window
type SLOTarget struct {
	// Name is the unique identifier for this SLO target
	Name string `json:"name" yaml:"name"`

	// Description provides context about this SLO
	Description string `json:"description" yaml:"description"`

	// Type is the category of this SLO
	Type SLOType `json:"type" yaml:"type"`

	// Target is the goal percentage (e.g., 99.9 for 99.9% uptime)
	Target float64 `json:"target" yaml:"target"`

	// Window is the time window for measuring this SLO (e.g., 30d, 7d)
	Window time.Duration `json:"window" yaml:"window"`

	// WarningThreshold is the threshold for warning alerts (e.g., 99.5)
	WarningThreshold float64 `json:"warning_threshold" yaml:"warning_threshold"`

	// CriticalThreshold is the threshold for critical alerts (e.g., 99.0)
	CriticalThreshold float64 `json:"critical_threshold" yaml:"critical_threshold"`

	// MetricQuery is the Prometheus query to evaluate this SLO
	MetricQuery string `json:"metric_query" yaml:"metric_query"`

	// Labels are additional labels for categorization
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// SLOStatus represents the current status of an SLO
type SLOStatus struct {
	// Target is the SLO being measured
	Target *SLOTarget `json:"target"`

	// CurrentValue is the current achievement level (e.g., 99.95)
	CurrentValue float64 `json:"current_value"`

	// ErrorBudget is the remaining error budget as percentage
	ErrorBudget float64 `json:"error_budget"`

	// ErrorBudgetRemaining is the absolute remaining error budget
	ErrorBudgetRemaining float64 `json:"error_budget_remaining"`

	// Status indicates if the SLO is being met
	Status SLOComplianceStatus `json:"status"`

	// LastUpdated is when this status was last calculated
	LastUpdated time.Time `json:"last_updated"`

	// Violations tracks recent SLO violations
	Violations []SLOViolation `json:"violations,omitempty"`
}

// SLOComplianceStatus represents whether an SLO is being met
type SLOComplianceStatus string

const (
	// SLOStatusHealthy indicates the SLO is being met
	SLOStatusHealthy SLOComplianceStatus = "healthy"
	// SLOStatusWarning indicates the SLO is approaching violation
	SLOStatusWarning SLOComplianceStatus = "warning"
	// SLOStatusCritical indicates the SLO is violated
	SLOStatusCritical SLOComplianceStatus = "critical"
	// SLOStatusUnknown indicates the SLO status cannot be determined
	SLOStatusUnknown SLOComplianceStatus = "unknown"
)

// SLOViolation represents a recorded SLO violation
type SLOViolation struct {
	// Timestamp when the violation occurred
	Timestamp time.Time `json:"timestamp"`

	// Value at the time of violation
	Value float64 `json:"value"`

	// Severity of the violation
	Severity SLOComplianceStatus `json:"severity"`

	// Duration of the violation
	Duration time.Duration `json:"duration"`

	// Message describing the violation
	Message string `json:"message"`
}

// SLOConfig holds the configuration for SLO monitoring
type SLOConfig struct {
	// Enabled determines if SLO monitoring is active
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Targets defines the SLO targets to monitor
	Targets []SLOTarget `json:"targets" yaml:"targets"`

	// UpdateInterval is how often to recalculate SLO status
	UpdateInterval time.Duration `json:"update_interval" yaml:"update_interval"`

	// RetentionPeriod is how long to keep historical SLO data
	RetentionPeriod time.Duration `json:"retention_period" yaml:"retention_period"`
}

// DefaultSLOConfig returns a default SLO configuration
func DefaultSLOConfig() *SLOConfig {
	return &SLOConfig{
		Enabled:         false,
		Targets:         DefaultSLOTargets(),
		UpdateInterval:  1 * time.Minute,
		RetentionPeriod: 30 * 24 * time.Hour, // 30 days
	}
}

// DefaultSLOTargets returns common SLO targets for a service
func DefaultSLOTargets() []SLOTarget {
	return []SLOTarget{
		{
			Name:              "availability_99_9",
			Description:       "99.9% service availability over 30 days",
			Type:              SLOTypeAvailability,
			Target:            99.9,
			Window:            30 * 24 * time.Hour,
			WarningThreshold:  99.5,
			CriticalThreshold: 99.0,
			MetricQuery:       `100 * (1 - (sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))))`,
			Labels: map[string]string{
				"tier":     "critical",
				"category": "availability",
			},
		},
		{
			Name:              "latency_p95_500ms",
			Description:       "95th percentile latency under 500ms",
			Type:              SLOTypeLatency,
			Target:            95.0,
			Window:            7 * 24 * time.Hour,
			WarningThreshold:  90.0,
			CriticalThreshold: 85.0,
			MetricQuery:       `100 * (sum(rate(http_request_duration_seconds_bucket{le="0.5"}[5m])) / sum(rate(http_request_duration_seconds_count[5m])))`,
			Labels: map[string]string{
				"tier":     "important",
				"category": "performance",
			},
		},
		{
			Name:              "error_rate_below_1_percent",
			Description:       "Error rate below 1% over 7 days",
			Type:              SLOTypeErrorRate,
			Target:            99.0,
			Window:            7 * 24 * time.Hour,
			WarningThreshold:  98.0,
			CriticalThreshold: 97.0,
			MetricQuery:       `100 * (1 - (sum(rate(http_requests_total{status=~"[45].."}[5m])) / sum(rate(http_requests_total[5m]))))`,
			Labels: map[string]string{
				"tier":     "critical",
				"category": "reliability",
			},
		},
	}
}

// SLOMonitor tracks and evaluates SLO compliance
type SLOMonitor struct {
	config   *SLOConfig
	statuses map[string]*SLOStatus
	mu       sync.RWMutex
}

// NewSLOMonitor creates a new SLO monitor
func NewSLOMonitor(config *SLOConfig) *SLOMonitor {
	if config == nil {
		config = DefaultSLOConfig()
	}

	monitor := &SLOMonitor{
		config:   config,
		statuses: make(map[string]*SLOStatus),
	}

	// Initialize statuses for each target
	for i := range config.Targets {
		target := &config.Targets[i]
		monitor.statuses[target.Name] = &SLOStatus{
			Target:      target,
			Status:      SLOStatusUnknown,
			LastUpdated: time.Now(),
			Violations:  make([]SLOViolation, 0),
		}
	}

	return monitor
}

// UpdateStatus updates the status for a specific SLO target
func (m *SLOMonitor) UpdateStatus(targetName string, currentValue float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	status, exists := m.statuses[targetName]
	if !exists {
		return fmt.Errorf("SLO target %s not found", targetName)
	}

	// Update current value
	status.CurrentValue = currentValue
	status.LastUpdated = time.Now()

	// Calculate error budget
	status.ErrorBudget = currentValue - status.Target.CriticalThreshold
	status.ErrorBudgetRemaining = (status.Target.Target - currentValue) / (status.Target.Target - status.Target.CriticalThreshold) * 100

	// Determine compliance status
	previousStatus := status.Status
	if currentValue >= status.Target.Target {
		status.Status = SLOStatusHealthy
	} else if currentValue >= status.Target.WarningThreshold {
		status.Status = SLOStatusWarning
	} else {
		status.Status = SLOStatusCritical
	}

	// Record violation if status changed to warning or critical
	if previousStatus == SLOStatusHealthy && status.Status != SLOStatusHealthy {
		violation := SLOViolation{
			Timestamp: time.Now(),
			Value:     currentValue,
			Severity:  status.Status,
			Message:   fmt.Sprintf("SLO %s dropped to %.2f%% (target: %.2f%%)", targetName, currentValue, status.Target.Target),
		}
		status.Violations = append(status.Violations, violation)
	}

	return nil
}

// GetStatus returns the current status for a specific SLO target
func (m *SLOMonitor) GetStatus(targetName string) (*SLOStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, exists := m.statuses[targetName]
	if !exists {
		return nil, fmt.Errorf("SLO target %s not found", targetName)
	}

	// Return a copy to prevent external modifications
	statusCopy := *status
	return &statusCopy, nil
}

// GetAllStatuses returns all current SLO statuses
func (m *SLOMonitor) GetAllStatuses() []*SLOStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]*SLOStatus, 0, len(m.statuses))
	for _, status := range m.statuses {
		statusCopy := *status
		statuses = append(statuses, &statusCopy)
	}

	return statuses
}

// GetViolations returns all violations within a time window
func (m *SLOMonitor) GetViolations(since time.Time) []SLOViolation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var violations []SLOViolation
	for _, status := range m.statuses {
		for _, v := range status.Violations {
			if v.Timestamp.After(since) {
				violations = append(violations, v)
			}
		}
	}

	return violations
}

// IsHealthy returns true if all SLOs are healthy
func (m *SLOMonitor) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, status := range m.statuses {
		if status.Status != SLOStatusHealthy {
			return false
		}
	}

	return true
}

// GetErrorBudgetStatus returns a summary of error budget consumption
func (m *SLOMonitor) GetErrorBudgetStatus() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	budgets := make(map[string]float64)
	for name, status := range m.statuses {
		budgets[name] = status.ErrorBudgetRemaining
	}

	return budgets
}

