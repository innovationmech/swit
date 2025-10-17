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

package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

// HealthStatus represents the health status of a Saga.
type HealthStatus string

const (
	// HealthStatusHealthy indicates the Saga is in a healthy state.
	HealthStatusHealthy HealthStatus = "healthy"

	// HealthStatusWarning indicates the Saga has minor issues but is operational.
	HealthStatusWarning HealthStatus = "warning"

	// HealthStatusUnhealthy indicates the Saga has critical issues.
	HealthStatusUnhealthy HealthStatus = "unhealthy"

	// HealthStatusUnknown indicates the health status cannot be determined.
	HealthStatusUnknown HealthStatus = "unknown"
)

// IssueSeverity represents the severity level of a health issue.
type IssueSeverity string

const (
	// SeverityInfo represents informational issues.
	SeverityInfo IssueSeverity = "info"

	// SeverityWarning represents warning-level issues.
	SeverityWarning IssueSeverity = "warning"

	// SeverityError represents error-level issues.
	SeverityError IssueSeverity = "error"

	// SeverityCritical represents critical issues.
	SeverityCritical IssueSeverity = "critical"
)

// IssueCategory represents the category of a health issue.
type IssueCategory string

const (
	// CategoryStateConsistency represents state consistency issues.
	CategoryStateConsistency IssueCategory = "state_consistency"

	// CategoryDataIntegrity represents data integrity issues.
	CategoryDataIntegrity IssueCategory = "data_integrity"

	// CategoryTimestamp represents timestamp-related issues.
	CategoryTimestamp IssueCategory = "timestamp"

	// CategoryDependency represents dependency-related issues.
	CategoryDependency IssueCategory = "dependency"

	// CategoryCompleteness represents completeness issues.
	CategoryCompleteness IssueCategory = "completeness"

	// CategoryPerformance represents performance-related issues.
	CategoryPerformance IssueCategory = "performance"
)

// HealthIssue represents a single health issue found during checking.
type HealthIssue struct {
	// Category is the category of the issue.
	Category IssueCategory `json:"category"`

	// Severity is the severity level of the issue.
	Severity IssueSeverity `json:"severity"`

	// Code is a unique code identifying the issue type.
	Code string `json:"code"`

	// Message is a human-readable description of the issue.
	Message string `json:"message"`

	// Details provides additional information about the issue.
	Details map[string]interface{} `json:"details,omitempty"`

	// Suggestions provides recommendations for resolving the issue.
	Suggestions []string `json:"suggestions,omitempty"`

	// Timestamp is when the issue was detected.
	Timestamp time.Time `json:"timestamp"`
}

// HealthReport contains the results of a health check.
type HealthReport struct {
	// SagaID is the ID of the checked Saga.
	SagaID string `json:"saga_id"`

	// Status is the overall health status.
	Status HealthStatus `json:"status"`

	// CheckTime is when the health check was performed.
	CheckTime time.Time `json:"check_time"`

	// Duration is how long the health check took.
	Duration time.Duration `json:"duration"`

	// Issues contains all detected issues.
	Issues []HealthIssue `json:"issues"`

	// Summary provides a summary of the health check.
	Summary map[string]interface{} `json:"summary"`

	// Metadata contains additional metadata about the check.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToJSON serializes the health report to JSON.
func (r *HealthReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal health report: %w", err)
	}
	return string(data), nil
}

// HasIssues returns true if the report contains any issues.
func (r *HealthReport) HasIssues() bool {
	return len(r.Issues) > 0
}

// HasCriticalIssues returns true if the report contains critical issues.
func (r *HealthReport) HasCriticalIssues() bool {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityCritical {
			return true
		}
	}
	return false
}

// GetIssuesBySeverity returns issues filtered by severity.
func (r *HealthReport) GetIssuesBySeverity(severity IssueSeverity) []HealthIssue {
	var filtered []HealthIssue
	for _, issue := range r.Issues {
		if issue.Severity == severity {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// GetIssuesByCategory returns issues filtered by category.
func (r *HealthReport) GetIssuesByCategory(category IssueCategory) []HealthIssue {
	var filtered []HealthIssue
	for _, issue := range r.Issues {
		if issue.Category == category {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// HealthChecker performs health checks on Saga instances.
type HealthChecker struct {
	// stateStorage provides access to Saga state.
	stateStorage saga.StateStorage

	// consistencyChecker performs consistency checks.
	consistencyChecker *ConsistencyChecker

	// logger is the structured logger.
	logger *zap.Logger
}

// NewHealthChecker creates a new HealthChecker instance.
func NewHealthChecker(
	stateStorage saga.StateStorage,
	logger *zap.Logger,
) *HealthChecker {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &HealthChecker{
		stateStorage:       stateStorage,
		consistencyChecker: NewConsistencyChecker(logger),
		logger:             logger.With(zap.String("component", "health_checker")),
	}
}

// CheckSagaHealth performs a comprehensive health check on a Saga instance.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - sagaID: The ID of the Saga to check.
//
// Returns:
//   - *HealthReport: The health check report.
//   - error: An error if the check fails.
func (hc *HealthChecker) CheckSagaHealth(ctx context.Context, sagaID string) (*HealthReport, error) {
	startTime := time.Now()

	hc.logger.Debug("starting health check", zap.String("saga_id", sagaID))

	// Create report
	report := &HealthReport{
		SagaID:    sagaID,
		CheckTime: startTime,
		Issues:    make([]HealthIssue, 0),
		Summary:   make(map[string]interface{}),
		Metadata:  make(map[string]interface{}),
	}

	// Retrieve Saga instance
	sagaInstance, err := hc.stateStorage.GetSaga(ctx, sagaID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve saga for health check: %w", err)
	}

	// Retrieve step states
	stepStates, err := hc.stateStorage.GetStepStates(ctx, sagaID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve step states for health check: %w", err)
	}

	// Perform consistency checks
	consistencyResults := hc.consistencyChecker.CheckConsistency(sagaInstance, stepStates)
	report.Issues = append(report.Issues, consistencyResults...)

	// Determine overall health status
	report.Status = hc.determineHealthStatus(report.Issues)

	// Calculate duration
	report.Duration = time.Since(startTime)

	// Build summary
	report.Summary = hc.buildSummary(sagaInstance, stepStates, report.Issues)

	hc.logger.Debug("health check completed",
		zap.String("saga_id", sagaID),
		zap.String("status", string(report.Status)),
		zap.Int("issues_count", len(report.Issues)),
		zap.Duration("duration", report.Duration),
	)

	return report, nil
}

// CheckMultipleSagasHealth performs health checks on multiple Saga instances.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - sagaIDs: The IDs of the Sagas to check.
//
// Returns:
//   - map[string]*HealthReport: Health reports keyed by Saga ID.
//   - error: An error if the check fails.
func (hc *HealthChecker) CheckMultipleSagasHealth(ctx context.Context, sagaIDs []string) (map[string]*HealthReport, error) {
	reports := make(map[string]*HealthReport)

	for _, sagaID := range sagaIDs {
		select {
		case <-ctx.Done():
			return reports, ctx.Err()
		default:
			report, err := hc.CheckSagaHealth(ctx, sagaID)
			if err != nil {
				hc.logger.Error("health check failed",
					zap.String("saga_id", sagaID),
					zap.Error(err),
				)
				// Continue with other Sagas
				continue
			}
			reports[sagaID] = report
		}
	}

	return reports, nil
}

// CheckSystemHealth performs a system-wide health check.
//
// Parameters:
//   - ctx: Context for cancellation.
//   - filter: Filter for selecting Sagas to check.
//
// Returns:
//   - *SystemHealthReport: The system-wide health report.
//   - error: An error if the check fails.
func (hc *HealthChecker) CheckSystemHealth(ctx context.Context, filter *saga.SagaFilter) (*SystemHealthReport, error) {
	startTime := time.Now()

	hc.logger.Info("starting system health check")

	// Retrieve active Sagas
	sagas, err := hc.stateStorage.GetActiveSagas(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve active sagas: %w", err)
	}

	// Collect Saga IDs
	sagaIDs := make([]string, len(sagas))
	for i, s := range sagas {
		sagaIDs[i] = s.GetID()
	}

	// Check health of all Sagas
	reports, err := hc.CheckMultipleSagasHealth(ctx, sagaIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to check multiple sagas health: %w", err)
	}

	// Build system report
	systemReport := &SystemHealthReport{
		CheckTime: startTime,
		Duration:  time.Since(startTime),
		Reports:   reports,
		Summary:   hc.buildSystemSummary(reports),
	}

	hc.logger.Info("system health check completed",
		zap.Int("total_sagas", len(sagas)),
		zap.Int("checked_sagas", len(reports)),
		zap.Duration("duration", systemReport.Duration),
	)

	return systemReport, nil
}

// determineHealthStatus determines the overall health status based on issues.
func (hc *HealthChecker) determineHealthStatus(issues []HealthIssue) HealthStatus {
	if len(issues) == 0 {
		return HealthStatusHealthy
	}

	hasCritical := false
	hasError := false
	hasWarning := false

	for _, issue := range issues {
		switch issue.Severity {
		case SeverityCritical:
			hasCritical = true
		case SeverityError:
			hasError = true
		case SeverityWarning:
			hasWarning = true
		}
	}

	if hasCritical || hasError {
		return HealthStatusUnhealthy
	}

	if hasWarning {
		return HealthStatusWarning
	}

	return HealthStatusHealthy
}

// buildSummary builds a summary of the health check results.
func (hc *HealthChecker) buildSummary(
	sagaInstance saga.SagaInstance,
	stepStates []*saga.StepState,
	issues []HealthIssue,
) map[string]interface{} {
	summary := make(map[string]interface{})

	// Basic info
	summary["saga_id"] = sagaInstance.GetID()
	summary["saga_state"] = sagaInstance.GetState().String()
	summary["is_terminal"] = sagaInstance.IsTerminal()
	summary["is_active"] = sagaInstance.IsActive()

	// Step info
	summary["total_steps"] = sagaInstance.GetTotalSteps()
	summary["completed_steps"] = sagaInstance.GetCompletedSteps()
	summary["actual_step_states_count"] = len(stepStates)

	// Issue counts by severity
	severityCounts := make(map[string]int)
	for _, issue := range issues {
		severityCounts[string(issue.Severity)]++
	}
	summary["issue_counts_by_severity"] = severityCounts

	// Issue counts by category
	categoryCounts := make(map[string]int)
	for _, issue := range issues {
		categoryCounts[string(issue.Category)]++
	}
	summary["issue_counts_by_category"] = categoryCounts

	// Timing info
	summary["created_at"] = sagaInstance.GetCreatedAt()
	summary["updated_at"] = sagaInstance.GetUpdatedAt()

	return summary
}

// buildSystemSummary builds a summary for system-wide health check.
func (hc *HealthChecker) buildSystemSummary(reports map[string]*HealthReport) map[string]interface{} {
	summary := make(map[string]interface{})

	totalSagas := len(reports)
	healthyCount := 0
	warningCount := 0
	unhealthyCount := 0
	totalIssues := 0

	for _, report := range reports {
		switch report.Status {
		case HealthStatusHealthy:
			healthyCount++
		case HealthStatusWarning:
			warningCount++
		case HealthStatusUnhealthy:
			unhealthyCount++
		}
		totalIssues += len(report.Issues)
	}

	summary["total_sagas"] = totalSagas
	summary["healthy_count"] = healthyCount
	summary["warning_count"] = warningCount
	summary["unhealthy_count"] = unhealthyCount
	summary["total_issues"] = totalIssues

	if totalSagas > 0 {
		summary["health_percentage"] = float64(healthyCount) / float64(totalSagas) * 100
	}

	return summary
}

// SystemHealthReport contains the results of a system-wide health check.
type SystemHealthReport struct {
	// CheckTime is when the system health check was performed.
	CheckTime time.Time `json:"check_time"`

	// Duration is how long the system health check took.
	Duration time.Duration `json:"duration"`

	// Reports contains individual Saga health reports.
	Reports map[string]*HealthReport `json:"reports"`

	// Summary provides a summary of the system health.
	Summary map[string]interface{} `json:"summary"`
}

// ToJSON serializes the system health report to JSON.
func (r *SystemHealthReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal system health report: %w", err)
	}
	return string(data), nil
}

// GetUnhealthySagas returns reports for unhealthy Sagas.
func (r *SystemHealthReport) GetUnhealthySagas() map[string]*HealthReport {
	unhealthy := make(map[string]*HealthReport)
	for id, report := range r.Reports {
		if report.Status == HealthStatusUnhealthy {
			unhealthy[id] = report
		}
	}
	return unhealthy
}

// GetSagasWithCriticalIssues returns reports for Sagas with critical issues.
func (r *SystemHealthReport) GetSagasWithCriticalIssues() map[string]*HealthReport {
	critical := make(map[string]*HealthReport)
	for id, report := range r.Reports {
		if report.HasCriticalIssues() {
			critical[id] = report
		}
	}
	return critical
}
