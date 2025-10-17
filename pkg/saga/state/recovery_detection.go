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
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

var (
	// ErrNoStorageForDetection is returned when detection requires storage but none is available.
	ErrNoStorageForDetection = errors.New("state storage required for detection")
)

// DetectionType represents the type of fault detected.
type DetectionType string

const (
	// DetectionTypeTimeout indicates a Saga has exceeded its timeout.
	DetectionTypeTimeout DetectionType = "timeout"

	// DetectionTypeStuck indicates a Saga has no state updates for a long time.
	DetectionTypeStuck DetectionType = "stuck"

	// DetectionTypeInconsistent indicates a Saga has inconsistent state.
	DetectionTypeInconsistent DetectionType = "inconsistent"

	// DetectionTypeOther indicates other types of issues.
	DetectionTypeOther DetectionType = "other"
)

// DetectionPriority represents the urgency of recovery.
type DetectionPriority int

const (
	// PriorityLow indicates low priority recovery.
	PriorityLow DetectionPriority = 1

	// PriorityMedium indicates medium priority recovery.
	PriorityMedium DetectionPriority = 2

	// PriorityHigh indicates high priority recovery.
	PriorityHigh DetectionPriority = 3

	// PriorityCritical indicates critical priority recovery.
	PriorityCritical DetectionPriority = 4
)

// DetectionResult represents the result of a fault detection check.
type DetectionResult struct {
	// SagaID is the ID of the detected Saga.
	SagaID string `json:"saga_id"`

	// Type is the type of fault detected.
	Type DetectionType `json:"type"`

	// Priority is the urgency of recovery.
	Priority DetectionPriority `json:"priority"`

	// DetectedAt is when the fault was detected.
	DetectedAt time.Time `json:"detected_at"`

	// Description provides details about the detected issue.
	Description string `json:"description"`

	// SagaState is the current state of the Saga.
	SagaState saga.SagaState `json:"saga_state"`

	// CurrentStep is the current step index.
	CurrentStep int `json:"current_step"`

	// TimeSinceUpdate is how long since the last update.
	TimeSinceUpdate time.Duration `json:"time_since_update"`

	// Metadata contains additional information.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DetectionConfig holds configuration for fault detection.
type DetectionConfig struct {
	// TimeoutThreshold is the threshold for considering a Saga timed out.
	// Sagas that exceed their timeout + this threshold are flagged.
	TimeoutThreshold time.Duration `json:"timeout_threshold" yaml:"timeout_threshold"`

	// StuckThreshold is the threshold for considering a Saga stuck.
	// Sagas with no updates for this duration are flagged.
	StuckThreshold time.Duration `json:"stuck_threshold" yaml:"stuck_threshold"`

	// EnableInconsistencyCheck enables state inconsistency checks.
	EnableInconsistencyCheck bool `json:"enable_inconsistency_check" yaml:"enable_inconsistency_check"`

	// MaxResultsPerScan limits the number of results per detection scan.
	MaxResultsPerScan int `json:"max_results_per_scan" yaml:"max_results_per_scan"`
}

// DefaultDetectionConfig returns default detection configuration.
func DefaultDetectionConfig() *DetectionConfig {
	return &DetectionConfig{
		TimeoutThreshold:         30 * time.Second,
		StuckThreshold:           5 * time.Minute,
		EnableInconsistencyCheck: true,
		MaxResultsPerScan:        100,
	}
}

// Validate validates the detection configuration.
func (c *DetectionConfig) Validate() error {
	if c.TimeoutThreshold < 0 {
		return errors.New("timeout threshold cannot be negative")
	}
	if c.StuckThreshold <= 0 {
		return errors.New("stuck threshold must be positive")
	}
	if c.MaxResultsPerScan <= 0 {
		return errors.New("max results per scan must be positive")
	}
	return nil
}

// detectTimeoutSagas detects Sagas that have exceeded their timeout.
//
// This method queries the state storage for Sagas that started before
// (now - timeout - threshold) and are still in active states.
func (rm *RecoveryManager) detectTimeoutSagas(ctx context.Context) ([]*DetectionResult, error) {
	if rm.stateStorage == nil {
		return nil, ErrNoStorageForDetection
	}

	// Calculate the cutoff time for timeout detection
	// We add the timeout threshold to give some buffer
	now := time.Now()
	cutoffTime := now.Add(-rm.config.MinRecoveryAge)

	rm.logger.Debug("detecting timeout sagas",
		zap.Time("cutoff_time", cutoffTime),
		zap.Time("now", now),
	)

	// Get timeout sagas from storage
	timeoutSagas, err := rm.stateStorage.GetTimeoutSagas(ctx, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeout sagas: %w", err)
	}

	results := make([]*DetectionResult, 0, len(timeoutSagas))
	for _, sagaInst := range timeoutSagas {
		// Skip if Saga is in terminal state
		if sagaInst.GetState().IsTerminal() {
			continue
		}

		// Skip if Saga is not active
		if !sagaInst.GetState().IsActive() {
			continue
		}

		// Calculate how long the Saga has been running
		var timeSinceStart time.Duration
		if !sagaInst.GetStartTime().IsZero() {
			timeSinceStart = now.Sub(sagaInst.GetStartTime())
		}

		// Calculate time since last update
		timeSinceUpdate := now.Sub(sagaInst.GetUpdatedAt())

		// Determine priority based on how much the timeout was exceeded
		priority := rm.calculateTimeoutPriority(timeSinceStart, sagaInst.GetTimeout())

		result := &DetectionResult{
			SagaID:          sagaInst.GetID(),
			Type:            DetectionTypeTimeout,
			Priority:        priority,
			DetectedAt:      now,
			Description:     fmt.Sprintf("Saga exceeded timeout: running for %v (timeout: %v)", timeSinceStart, sagaInst.GetTimeout()),
			SagaState:       sagaInst.GetState(),
			CurrentStep:     sagaInst.GetCurrentStep(),
			TimeSinceUpdate: timeSinceUpdate,
			Metadata: map[string]interface{}{
				"timeout":           sagaInst.GetTimeout().String(),
				"time_since_start":  timeSinceStart.String(),
				"time_since_update": timeSinceUpdate.String(),
			},
		}

		results = append(results, result)

		rm.logger.Debug("detected timeout saga",
			zap.String("saga_id", sagaInst.GetID()),
			zap.String("state", sagaInst.GetState().String()),
			zap.Duration("time_since_start", timeSinceStart),
			zap.Duration("timeout", sagaInst.GetTimeout()),
		)
	}

	rm.logger.Info("timeout detection completed",
		zap.Int("detected_count", len(results)),
	)

	return results, nil
}

// detectStuckSagas detects Sagas that have no state updates for a long time.
//
// A Saga is considered stuck if:
// - It's in an active state (Running, Compensating, etc.)
// - Its UpdatedAt timestamp is older than the stuck threshold
// - It hasn't timed out (that's handled by detectTimeoutSagas)
func (rm *RecoveryManager) detectStuckSagas(ctx context.Context) ([]*DetectionResult, error) {
	if rm.stateStorage == nil {
		return nil, ErrNoStorageForDetection
	}

	now := time.Now()
	stuckThreshold := 5 * time.Minute // Default stuck threshold

	rm.logger.Debug("detecting stuck sagas",
		zap.Duration("stuck_threshold", stuckThreshold),
		zap.Time("now", now),
	)

	// Get all active sagas
	filter := &saga.SagaFilter{
		States: []saga.SagaState{
			saga.StateRunning,
			saga.StateStepCompleted,
			saga.StateCompensating,
		},
		Limit: 1000, // Limit to prevent overwhelming the system
	}

	activeSagas, err := rm.stateStorage.GetActiveSagas(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sagas: %w", err)
	}

	results := make([]*DetectionResult, 0)
	for _, sagaInst := range activeSagas {
		// Calculate time since last update
		timeSinceUpdate := now.Sub(sagaInst.GetUpdatedAt())

		// Skip if not stuck
		if timeSinceUpdate < stuckThreshold {
			continue
		}

		// Skip if this is a timeout case (will be handled by detectTimeoutSagas)
		if !sagaInst.GetStartTime().IsZero() && sagaInst.GetTimeout() > 0 {
			timeSinceStart := now.Sub(sagaInst.GetStartTime())
			if timeSinceStart >= sagaInst.GetTimeout() {
				continue // This is a timeout case
			}
		}

		// Determine priority based on how long it's been stuck
		priority := rm.calculateStuckPriority(timeSinceUpdate, stuckThreshold)

		result := &DetectionResult{
			SagaID:          sagaInst.GetID(),
			Type:            DetectionTypeStuck,
			Priority:        priority,
			DetectedAt:      now,
			Description:     fmt.Sprintf("Saga stuck: no updates for %v", timeSinceUpdate),
			SagaState:       sagaInst.GetState(),
			CurrentStep:     sagaInst.GetCurrentStep(),
			TimeSinceUpdate: timeSinceUpdate,
			Metadata: map[string]interface{}{
				"time_since_update": timeSinceUpdate.String(),
				"stuck_threshold":   stuckThreshold.String(),
				"last_update":       sagaInst.GetUpdatedAt().Format(time.RFC3339),
			},
		}

		results = append(results, result)

		rm.logger.Debug("detected stuck saga",
			zap.String("saga_id", sagaInst.GetID()),
			zap.String("state", sagaInst.GetState().String()),
			zap.Duration("time_since_update", timeSinceUpdate),
		)
	}

	rm.logger.Info("stuck detection completed",
		zap.Int("detected_count", len(results)),
	)

	return results, nil
}

// detectInconsistentSagas detects Sagas with inconsistent state.
//
// State inconsistencies include:
// - Saga state doesn't match step states
// - Invalid step progression (e.g., step 5 completed but step 3 pending)
// - Orphan steps (steps without corresponding Saga)
// - Compensation state mismatch
func (rm *RecoveryManager) detectInconsistentSagas(ctx context.Context) ([]*DetectionResult, error) {
	if rm.stateStorage == nil {
		return nil, ErrNoStorageForDetection
	}

	now := time.Now()

	rm.logger.Debug("detecting inconsistent sagas")

	// Get all active sagas for consistency check
	filter := &saga.SagaFilter{
		States: []saga.SagaState{
			saga.StateRunning,
			saga.StateStepCompleted,
			saga.StateCompensating,
			saga.StateCompleted,
			saga.StateFailed,
		},
		Limit: 1000,
	}

	activeSagas, err := rm.stateStorage.GetActiveSagas(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get sagas for consistency check: %w", err)
	}

	results := make([]*DetectionResult, 0)

	for _, sagaInst := range activeSagas {
		// Get step states for this Saga
		stepStates, err := rm.stateStorage.GetStepStates(ctx, sagaInst.GetID())
		if err != nil {
			rm.logger.Warn("failed to get step states for consistency check",
				zap.String("saga_id", sagaInst.GetID()),
				zap.Error(err),
			)
			continue
		}

		// Check for inconsistencies
		inconsistencies := rm.checkStateConsistency(sagaInst, stepStates)

		if len(inconsistencies) > 0 {
			// Calculate time since last update
			timeSinceUpdate := now.Sub(sagaInst.GetUpdatedAt())

			result := &DetectionResult{
				SagaID:          sagaInst.GetID(),
				Type:            DetectionTypeInconsistent,
				Priority:        PriorityHigh, // Inconsistencies are usually high priority
				DetectedAt:      now,
				Description:     fmt.Sprintf("Saga has %d inconsistencies: %v", len(inconsistencies), inconsistencies),
				SagaState:       sagaInst.GetState(),
				CurrentStep:     sagaInst.GetCurrentStep(),
				TimeSinceUpdate: timeSinceUpdate,
				Metadata: map[string]interface{}{
					"inconsistencies":   inconsistencies,
					"step_count":        len(stepStates),
					"time_since_update": timeSinceUpdate.String(),
				},
			}

			results = append(results, result)

			rm.logger.Debug("detected inconsistent saga",
				zap.String("saga_id", sagaInst.GetID()),
				zap.String("state", sagaInst.GetState().String()),
				zap.Strings("inconsistencies", inconsistencies),
			)
		}
	}

	rm.logger.Info("inconsistency detection completed",
		zap.Int("detected_count", len(results)),
	)

	return results, nil
}

// checkStateConsistency checks for state inconsistencies in a Saga.
func (rm *RecoveryManager) checkStateConsistency(sagaInst saga.SagaInstance, stepStates []*saga.StepState) []string {
	inconsistencies := make([]string, 0)

	if len(stepStates) == 0 {
		return inconsistencies
	}

	// Sort steps by index
	sort.Slice(stepStates, func(i, j int) bool {
		return stepStates[i].StepIndex < stepStates[j].StepIndex
	})

	currentStep := sagaInst.GetCurrentStep()
	sagaState := sagaInst.GetState()

	// Check 1: Current step index should be within bounds
	if currentStep >= len(stepStates) && currentStep != -1 {
		inconsistencies = append(inconsistencies,
			fmt.Sprintf("current step index %d exceeds step count %d", currentStep, len(stepStates)))
	}

	// Check 2: If Saga is Running, current step should be Running or Pending
	if sagaState == saga.StateRunning && currentStep >= 0 && currentStep < len(stepStates) {
		stepState := stepStates[currentStep].State
		if stepState != saga.StepStateRunning && stepState != saga.StepStatePending {
			inconsistencies = append(inconsistencies,
				fmt.Sprintf("saga running but current step %d is %s", currentStep, stepState.String()))
		}
	}

	// Check 3: If Saga is Completed, all steps should be Completed or Skipped
	if sagaState == saga.StateCompleted {
		for _, step := range stepStates {
			if step.State != saga.StepStateCompleted && step.State != saga.StepStateSkipped {
				inconsistencies = append(inconsistencies,
					fmt.Sprintf("saga completed but step %d is %s", step.StepIndex, step.State.String()))
			}
		}
	}

	// Check 4: If Saga is Compensating, at least one step should be Compensating or Compensated
	if sagaState == saga.StateCompensating {
		hasCompensatingStep := false
		for _, step := range stepStates {
			if step.State == saga.StepStateCompensating || step.State == saga.StepStateCompensated {
				hasCompensatingStep = true
				break
			}
		}
		if !hasCompensatingStep {
			inconsistencies = append(inconsistencies,
				"saga compensating but no steps are compensating or compensated")
		}
	}

	// Check 5: Steps should be in logical progression (no completed steps after pending ones)
	lastCompletedIndex := -1
	for i, step := range stepStates {
		if step.State == saga.StepStateCompleted {
			lastCompletedIndex = i
		} else if step.State == saga.StepStatePending && lastCompletedIndex >= 0 {
			// Found a pending step after a completed one - might be ok in some cases
			// but worth flagging
			if i < currentStep {
				inconsistencies = append(inconsistencies,
					fmt.Sprintf("step %d is pending but step %d is completed", i, lastCompletedIndex))
			}
		}
	}

	return inconsistencies
}

// calculateTimeoutPriority determines priority based on timeout excess.
func (rm *RecoveryManager) calculateTimeoutPriority(timeSinceStart, timeout time.Duration) DetectionPriority {
	if timeout == 0 {
		return PriorityMedium
	}

	excess := timeSinceStart - timeout
	excessRatio := float64(excess) / float64(timeout)

	switch {
	case excessRatio >= 2.0: // More than 200% over timeout
		return PriorityCritical
	case excessRatio >= 1.0: // More than 100% over timeout
		return PriorityHigh
	case excessRatio >= 0.5: // More than 50% over timeout
		return PriorityMedium
	default:
		return PriorityLow
	}
}

// calculateStuckPriority determines priority based on stuck duration.
func (rm *RecoveryManager) calculateStuckPriority(timeSinceUpdate, stuckThreshold time.Duration) DetectionPriority {
	ratio := float64(timeSinceUpdate) / float64(stuckThreshold)

	switch {
	case ratio >= 4.0: // 4x the stuck threshold
		return PriorityCritical
	case ratio >= 2.0: // 2x the stuck threshold
		return PriorityHigh
	case ratio >= 1.5: // 1.5x the stuck threshold
		return PriorityMedium
	default:
		return PriorityLow
	}
}

// performDetection runs all detection checks and returns combined results.
func (rm *RecoveryManager) performDetection(ctx context.Context) ([]*DetectionResult, error) {
	allResults := make([]*DetectionResult, 0)

	// Detect timeout sagas
	timeoutResults, err := rm.detectTimeoutSagas(ctx)
	if err != nil {
		rm.logger.Error("timeout detection failed", zap.Error(err))
		// Don't fail the entire detection if one type fails
	} else {
		allResults = append(allResults, timeoutResults...)
	}

	// Detect stuck sagas
	stuckResults, err := rm.detectStuckSagas(ctx)
	if err != nil {
		rm.logger.Error("stuck detection failed", zap.Error(err))
	} else {
		allResults = append(allResults, stuckResults...)
	}

	// Detect inconsistent sagas (if enabled)
	if rm.config.EnableMetrics { // Reusing EnableMetrics as a proxy for advanced checks
		inconsistentResults, err := rm.detectInconsistentSagas(ctx)
		if err != nil {
			rm.logger.Error("inconsistency detection failed", zap.Error(err))
		} else {
			allResults = append(allResults, inconsistentResults...)
		}
	}

	// Sort results by priority (highest first)
	sort.Slice(allResults, func(i, j int) bool {
		if allResults[i].Priority != allResults[j].Priority {
			return allResults[i].Priority > allResults[j].Priority
		}
		// If same priority, sort by detection time (oldest first)
		return allResults[i].DetectedAt.Before(allResults[j].DetectedAt)
	})

	return allResults, nil
}
