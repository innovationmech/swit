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
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

// ConsistencyChecker performs consistency checks on Saga instances.
type ConsistencyChecker struct {
	// logger is the structured logger.
	logger *zap.Logger
}

// NewConsistencyChecker creates a new ConsistencyChecker instance.
func NewConsistencyChecker(logger *zap.Logger) *ConsistencyChecker {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &ConsistencyChecker{
		logger: logger.With(zap.String("component", "consistency_checker")),
	}
}

// CheckConsistency performs comprehensive consistency checks on a Saga instance.
//
// Parameters:
//   - sagaInstance: The Saga instance to check.
//   - stepStates: The step states associated with the Saga.
//
// Returns:
//   - []HealthIssue: A list of detected issues.
func (cc *ConsistencyChecker) CheckConsistency(
	sagaInstance saga.SagaInstance,
	stepStates []*saga.StepState,
) []HealthIssue {
	issues := make([]HealthIssue, 0)

	// Check required fields
	issues = append(issues, cc.checkRequiredFields(sagaInstance)...)

	// Check state consistency
	issues = append(issues, cc.checkStateConsistency(sagaInstance, stepStates)...)

	// Check step execution order
	issues = append(issues, cc.checkStepExecutionOrder(stepStates)...)

	// Check timestamp logic
	issues = append(issues, cc.checkTimestampLogic(sagaInstance, stepStates)...)

	// Check terminal state completeness
	if sagaInstance.IsTerminal() {
		issues = append(issues, cc.checkTerminalStateCompleteness(sagaInstance, stepStates)...)
	}

	// Check data integrity
	issues = append(issues, cc.checkDataIntegrity(sagaInstance, stepStates)...)

	// Check for orphaned steps
	issues = append(issues, cc.checkOrphanedSteps(sagaInstance, stepStates)...)

	// Check for missing steps
	issues = append(issues, cc.checkMissingSteps(sagaInstance, stepStates)...)

	return issues
}

// checkRequiredFields verifies that all required fields are present.
func (cc *ConsistencyChecker) checkRequiredFields(sagaInstance saga.SagaInstance) []HealthIssue {
	issues := make([]HealthIssue, 0)

	// Check Saga ID
	if sagaInstance.GetID() == "" {
		issues = append(issues, HealthIssue{
			Category:  CategoryDataIntegrity,
			Severity:  SeverityCritical,
			Code:      "MISSING_SAGA_ID",
			Message:   "Saga ID is missing",
			Timestamp: time.Now(),
			Suggestions: []string{
				"This is a critical data integrity issue",
				"The Saga instance should not exist without an ID",
			},
		})
	}

	// Check Definition ID
	if sagaInstance.GetDefinitionID() == "" {
		issues = append(issues, HealthIssue{
			Category:  CategoryDataIntegrity,
			Severity:  SeverityError,
			Code:      "MISSING_DEFINITION_ID",
			Message:   "Saga definition ID is missing",
			Timestamp: time.Now(),
			Suggestions: []string{
				"Verify that the Saga was properly initialized",
				"Check the Saga creation process",
			},
		})
	}

	return issues
}

// checkStateConsistency verifies that Saga state is consistent with step states.
func (cc *ConsistencyChecker) checkStateConsistency(
	sagaInstance saga.SagaInstance,
	stepStates []*saga.StepState,
) []HealthIssue {
	issues := make([]HealthIssue, 0)
	sagaState := sagaInstance.GetState()

	// Count completed steps
	completedCount := 0
	failedCount := 0
	runningCount := 0
	compensatingCount := 0

	for _, step := range stepStates {
		switch step.State {
		case saga.StepStateCompleted:
			completedCount++
		case saga.StepStateFailed:
			failedCount++
		case saga.StepStateRunning:
			runningCount++
		case saga.StepStateCompensating:
			compensatingCount++
		}
	}

	// Check Completed state consistency
	if sagaState == saga.StateCompleted {
		expectedCompletedSteps := sagaInstance.GetTotalSteps()
		if completedCount < expectedCompletedSteps {
			issues = append(issues, HealthIssue{
				Category: CategoryStateConsistency,
				Severity: SeverityError,
				Code:     "INCOMPLETE_COMPLETED_SAGA",
				Message:  fmt.Sprintf("Saga is in Completed state but only %d/%d steps are completed", completedCount, expectedCompletedSteps),
				Details: map[string]interface{}{
					"saga_state":           sagaState.String(),
					"completed_steps":      completedCount,
					"expected_steps":       expectedCompletedSteps,
					"saga_completed_steps": sagaInstance.GetCompletedSteps(),
				},
				Timestamp: time.Now(),
				Suggestions: []string{
					"Verify that all steps were executed",
					"Check for missing step state records",
					"Consider recovering the Saga",
				},
			})
		}
	}

	// Check Running state consistency
	if sagaState == saga.StateRunning && runningCount == 0 {
		issues = append(issues, HealthIssue{
			Category: CategoryStateConsistency,
			Severity: SeverityWarning,
			Code:     "RUNNING_SAGA_NO_RUNNING_STEPS",
			Message:  "Saga is in Running state but no steps are currently running",
			Details: map[string]interface{}{
				"saga_state":      sagaState.String(),
				"running_steps":   runningCount,
				"completed_steps": completedCount,
			},
			Timestamp: time.Now(),
			Suggestions: []string{
				"Check if the Saga is stuck",
				"Verify step execution is progressing",
				"Consider manual recovery",
			},
		})
	}

	// Check Compensating state consistency
	if sagaState == saga.StateCompensating && compensatingCount == 0 {
		issues = append(issues, HealthIssue{
			Category: CategoryStateConsistency,
			Severity: SeverityWarning,
			Code:     "COMPENSATING_SAGA_NO_COMPENSATING_STEPS",
			Message:  "Saga is in Compensating state but no steps are currently compensating",
			Details: map[string]interface{}{
				"saga_state":         sagaState.String(),
				"compensating_steps": compensatingCount,
			},
			Timestamp: time.Now(),
			Suggestions: []string{
				"Check if compensation is stuck",
				"Verify compensation logic",
			},
		})
	}

	// Check Failed state consistency
	if sagaState == saga.StateFailed && failedCount == 0 {
		issues = append(issues, HealthIssue{
			Category: CategoryStateConsistency,
			Severity: SeverityWarning,
			Code:     "FAILED_SAGA_NO_FAILED_STEPS",
			Message:  "Saga is in Failed state but no steps are marked as failed",
			Details: map[string]interface{}{
				"saga_state":   sagaState.String(),
				"failed_steps": failedCount,
			},
			Timestamp: time.Now(),
			Suggestions: []string{
				"Verify the failure was properly recorded",
				"Check error handling logic",
			},
		})
	}

	return issues
}

// checkStepExecutionOrder verifies that steps are executed in the correct order.
func (cc *ConsistencyChecker) checkStepExecutionOrder(stepStates []*saga.StepState) []HealthIssue {
	issues := make([]HealthIssue, 0)

	if len(stepStates) == 0 {
		return issues
	}

	// Build a map of step index to step state
	stepMap := make(map[int]*saga.StepState)
	for _, step := range stepStates {
		stepMap[step.StepIndex] = step
	}

	// Check that completed steps form a continuous sequence from index 0
	maxCompletedIndex := -1
	for i := 0; i < len(stepStates); i++ {
		step, exists := stepMap[i]
		if !exists {
			continue
		}

		if step.State == saga.StepStateCompleted {
			if i != maxCompletedIndex+1 && maxCompletedIndex != -1 {
				issues = append(issues, HealthIssue{
					Category: CategoryStateConsistency,
					Severity: SeverityWarning,
					Code:     "NON_SEQUENTIAL_STEP_COMPLETION",
					Message:  fmt.Sprintf("Step %d is completed but earlier steps are not", i),
					Details: map[string]interface{}{
						"step_index":          i,
						"max_completed_index": maxCompletedIndex,
						"step_name":           step.Name,
					},
					Timestamp: time.Now(),
					Suggestions: []string{
						"This might indicate parallel execution or recovery",
						"Verify that the execution order is intentional",
					},
				})
			}
			maxCompletedIndex = i
		}
	}

	return issues
}

// checkTimestampLogic verifies the logical consistency of timestamps.
func (cc *ConsistencyChecker) checkTimestampLogic(
	sagaInstance saga.SagaInstance,
	stepStates []*saga.StepState,
) []HealthIssue {
	issues := make([]HealthIssue, 0)

	createdAt := sagaInstance.GetCreatedAt()
	updatedAt := sagaInstance.GetUpdatedAt()
	startTime := sagaInstance.GetStartTime()
	endTime := sagaInstance.GetEndTime()

	// Check CreatedAt <= UpdatedAt
	if !updatedAt.IsZero() && updatedAt.Before(createdAt) {
		issues = append(issues, HealthIssue{
			Category: CategoryTimestamp,
			Severity: SeverityError,
			Code:     "INVALID_TIMESTAMP_ORDER_UPDATED_BEFORE_CREATED",
			Message:  "UpdatedAt is before CreatedAt",
			Details: map[string]interface{}{
				"created_at": createdAt,
				"updated_at": updatedAt,
			},
			Timestamp: time.Now(),
			Suggestions: []string{
				"This indicates a data corruption issue",
				"Check the timestamp update logic",
			},
		})
	}

	// Check StartTime >= CreatedAt
	if !startTime.IsZero() && startTime.Before(createdAt) {
		issues = append(issues, HealthIssue{
			Category: CategoryTimestamp,
			Severity: SeverityError,
			Code:     "INVALID_TIMESTAMP_ORDER_START_BEFORE_CREATED",
			Message:  "StartTime is before CreatedAt",
			Details: map[string]interface{}{
				"created_at": createdAt,
				"start_time": startTime,
			},
			Timestamp: time.Now(),
			Suggestions: []string{
				"Verify the Saga initialization logic",
				"Check for clock synchronization issues",
			},
		})
	}

	// Check EndTime >= StartTime for terminal states
	if sagaInstance.IsTerminal() && !endTime.IsZero() && !startTime.IsZero() && endTime.Before(startTime) {
		issues = append(issues, HealthIssue{
			Category: CategoryTimestamp,
			Severity: SeverityError,
			Code:     "INVALID_TIMESTAMP_ORDER_END_BEFORE_START",
			Message:  "EndTime is before StartTime",
			Details: map[string]interface{}{
				"start_time": startTime,
				"end_time":   endTime,
			},
			Timestamp: time.Now(),
			Suggestions: []string{
				"This is a critical timestamp logic error",
				"Check the state transition logic",
			},
		})
	}

	// Check step timestamps
	for _, step := range stepStates {
		if step.StartedAt != nil && step.StartedAt.Before(createdAt) {
			issues = append(issues, HealthIssue{
				Category: CategoryTimestamp,
				Severity: SeverityWarning,
				Code:     "STEP_STARTED_BEFORE_SAGA_CREATED",
				Message:  fmt.Sprintf("Step %s started before Saga was created", step.Name),
				Details: map[string]interface{}{
					"step_name":    step.Name,
					"step_index":   step.StepIndex,
					"saga_created": createdAt,
					"step_started": step.StartedAt,
				},
				Timestamp: time.Now(),
				Suggestions: []string{
					"Check for clock synchronization issues",
					"Verify step execution timing",
				},
			})
		}

		if step.CompletedAt != nil && step.StartedAt != nil && step.CompletedAt.Before(*step.StartedAt) {
			issues = append(issues, HealthIssue{
				Category: CategoryTimestamp,
				Severity: SeverityError,
				Code:     "STEP_COMPLETED_BEFORE_STARTED",
				Message:  fmt.Sprintf("Step %s completed before it started", step.Name),
				Details: map[string]interface{}{
					"step_name":      step.Name,
					"step_index":     step.StepIndex,
					"step_started":   step.StartedAt,
					"step_completed": step.CompletedAt,
				},
				Timestamp: time.Now(),
				Suggestions: []string{
					"This indicates a critical data integrity issue",
					"Check the step state update logic",
				},
			})
		}
	}

	return issues
}

// checkTerminalStateCompleteness verifies completeness of terminal state Sagas.
func (cc *ConsistencyChecker) checkTerminalStateCompleteness(
	sagaInstance saga.SagaInstance,
	stepStates []*saga.StepState,
) []HealthIssue {
	issues := make([]HealthIssue, 0)

	// Check that EndTime is set for terminal states
	if sagaInstance.GetEndTime().IsZero() {
		issues = append(issues, HealthIssue{
			Category: CategoryCompleteness,
			Severity: SeverityWarning,
			Code:     "TERMINAL_SAGA_MISSING_END_TIME",
			Message:  "Saga in terminal state is missing EndTime",
			Details: map[string]interface{}{
				"saga_state": sagaInstance.GetState().String(),
			},
			Timestamp: time.Now(),
			Suggestions: []string{
				"Verify that the state transition properly set EndTime",
				"Check the completion logic",
			},
		})
	}

	// For completed Sagas, verify all steps are completed or skipped
	if sagaInstance.GetState() == saga.StateCompleted {
		totalSteps := sagaInstance.GetTotalSteps()
		completedOrSkipped := 0

		for _, step := range stepStates {
			if step.State == saga.StepStateCompleted || step.State == saga.StepStateSkipped {
				completedOrSkipped++
			}
		}

		if completedOrSkipped < totalSteps {
			issues = append(issues, HealthIssue{
				Category: CategoryCompleteness,
				Severity: SeverityError,
				Code:     "COMPLETED_SAGA_INCOMPLETE_STEPS",
				Message:  fmt.Sprintf("Completed Saga has incomplete steps: %d/%d completed or skipped", completedOrSkipped, totalSteps),
				Details: map[string]interface{}{
					"total_steps":          totalSteps,
					"completed_or_skipped": completedOrSkipped,
					"actual_step_count":    len(stepStates),
				},
				Timestamp: time.Now(),
				Suggestions: []string{
					"Verify all steps were executed",
					"Check for missing step records",
					"Consider recovering the Saga",
				},
			})
		}
	}

	return issues
}

// checkDataIntegrity verifies data integrity constraints.
func (cc *ConsistencyChecker) checkDataIntegrity(
	sagaInstance saga.SagaInstance,
	stepStates []*saga.StepState,
) []HealthIssue {
	issues := make([]HealthIssue, 0)
	sagaID := sagaInstance.GetID()

	// Check foreign key integrity: all steps should reference the same Saga ID
	for _, step := range stepStates {
		if step.SagaID != sagaID {
			issues = append(issues, HealthIssue{
				Category: CategoryDataIntegrity,
				Severity: SeverityCritical,
				Code:     "STEP_SAGA_ID_MISMATCH",
				Message:  fmt.Sprintf("Step %s references wrong Saga ID", step.Name),
				Details: map[string]interface{}{
					"step_name":        step.Name,
					"step_saga_id":     step.SagaID,
					"expected_saga_id": sagaID,
				},
				Timestamp: time.Now(),
				Suggestions: []string{
					"This is a critical data integrity violation",
					"The step should be reassociated with the correct Saga",
				},
			})
		}

		// Check step ID is not empty
		if step.ID == "" {
			issues = append(issues, HealthIssue{
				Category: CategoryDataIntegrity,
				Severity: SeverityError,
				Code:     "MISSING_STEP_ID",
				Message:  fmt.Sprintf("Step at index %d is missing ID", step.StepIndex),
				Details: map[string]interface{}{
					"step_index": step.StepIndex,
					"step_name":  step.Name,
				},
				Timestamp: time.Now(),
				Suggestions: []string{
					"Verify step initialization logic",
				},
			})
		}
	}

	// Check for duplicate step indices
	stepIndices := make(map[int]bool)
	for _, step := range stepStates {
		if stepIndices[step.StepIndex] {
			issues = append(issues, HealthIssue{
				Category: CategoryDataIntegrity,
				Severity: SeverityError,
				Code:     "DUPLICATE_STEP_INDEX",
				Message:  fmt.Sprintf("Duplicate step index found: %d", step.StepIndex),
				Details: map[string]interface{}{
					"step_index": step.StepIndex,
				},
				Timestamp: time.Now(),
				Suggestions: []string{
					"Check step creation logic",
					"Verify data consistency",
				},
			})
		}
		stepIndices[step.StepIndex] = true
	}

	return issues
}

// checkOrphanedSteps checks for steps that exist but shouldn't.
func (cc *ConsistencyChecker) checkOrphanedSteps(
	sagaInstance saga.SagaInstance,
	stepStates []*saga.StepState,
) []HealthIssue {
	issues := make([]HealthIssue, 0)
	totalSteps := sagaInstance.GetTotalSteps()

	// Check for steps with indices beyond the total step count
	for _, step := range stepStates {
		if step.StepIndex >= totalSteps {
			issues = append(issues, HealthIssue{
				Category: CategoryDataIntegrity,
				Severity: SeverityError,
				Code:     "ORPHANED_STEP",
				Message:  fmt.Sprintf("Step %s has index %d which exceeds total steps %d", step.Name, step.StepIndex, totalSteps),
				Details: map[string]interface{}{
					"step_name":   step.Name,
					"step_index":  step.StepIndex,
					"total_steps": totalSteps,
				},
				Timestamp: time.Now(),
				Suggestions: []string{
					"This step might be orphaned",
					"Verify the Saga definition",
					"Consider cleaning up orphaned steps",
				},
			})
		}
	}

	return issues
}

// checkMissingSteps checks for expected steps that are missing.
func (cc *ConsistencyChecker) checkMissingSteps(
	sagaInstance saga.SagaInstance,
	stepStates []*saga.StepState,
) []HealthIssue {
	issues := make([]HealthIssue, 0)
	totalSteps := sagaInstance.GetTotalSteps()

	// Build a set of existing step indices
	existingIndices := make(map[int]bool)
	for _, step := range stepStates {
		existingIndices[step.StepIndex] = true
	}

	// Check if we're missing any steps for completed or active Sagas
	if sagaInstance.GetState() != saga.StatePending {
		missingIndices := make([]int, 0)
		for i := 0; i < totalSteps; i++ {
			if !existingIndices[i] {
				missingIndices = append(missingIndices, i)
			}
		}

		if len(missingIndices) > 0 {
			issues = append(issues, HealthIssue{
				Category: CategoryCompleteness,
				Severity: SeverityWarning,
				Code:     "MISSING_STEP_STATES",
				Message:  fmt.Sprintf("Missing step state records for %d steps", len(missingIndices)),
				Details: map[string]interface{}{
					"total_steps":     totalSteps,
					"existing_steps":  len(stepStates),
					"missing_indices": missingIndices,
				},
				Timestamp: time.Now(),
				Suggestions: []string{
					"Some steps might not have been initialized",
					"Check if steps are created lazily",
					"Verify the execution flow",
				},
			})
		}
	}

	return issues
}
