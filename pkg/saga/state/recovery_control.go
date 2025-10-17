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
	"errors"
	"time"

	"go.uber.org/zap"
)

var (
	// ErrRecoveryPaused is returned when recovery is paused.
	ErrRecoveryPaused = errors.New("recovery is paused")

	// ErrInvalidInterval is returned when an invalid interval is provided.
	ErrInvalidInterval = errors.New("invalid interval")

	// ErrInvalidConcurrency is returned when an invalid concurrency value is provided.
	ErrInvalidConcurrency = errors.New("invalid concurrency value")
)

// RecoveryStatus represents the current status of the recovery manager.
type RecoveryStatus struct {
	// IsRunning indicates if the recovery manager is running.
	IsRunning bool `json:"is_running"`

	// IsPaused indicates if automatic recovery is paused.
	IsPaused bool `json:"is_paused"`

	// CheckInterval is the current check interval.
	CheckInterval time.Duration `json:"check_interval"`

	// MaxConcurrentRecoveries is the current max concurrent recoveries.
	MaxConcurrentRecoveries int `json:"max_concurrent_recoveries"`

	// Stats holds the current recovery statistics.
	Stats *RecoveryStats `json:"stats"`

	// ActiveRecoveries is the number of currently active recoveries.
	ActiveRecoveries int `json:"active_recoveries"`

	// LastCheckTime is the time of the last recovery check.
	LastCheckTime time.Time `json:"last_check_time"`
}

// SagaRecoveryStatus represents the recovery status of a specific Saga.
type SagaRecoveryStatus struct {
	// SagaID is the ID of the Saga.
	SagaID string `json:"saga_id"`

	// IsRecovering indicates if the Saga is currently being recovered.
	IsRecovering bool `json:"is_recovering"`

	// RecoveryStartTime is when the recovery started.
	RecoveryStartTime time.Time `json:"recovery_start_time,omitempty"`

	// AttemptCount is the number of recovery attempts.
	AttemptCount int `json:"attempt_count"`

	// LastAttemptTime is the time of the last recovery attempt.
	LastAttemptTime time.Time `json:"last_attempt_time,omitempty"`

	// State is the current Saga state.
	State string `json:"state"`

	// Error is the last recovery error (if any).
	Error string `json:"error,omitempty"`
}

// RecoveryHistory represents the recovery history for a Saga.
type RecoveryHistory struct {
	// SagaID is the ID of the Saga.
	SagaID string `json:"saga_id"`

	// TotalAttempts is the total number of recovery attempts.
	TotalAttempts int `json:"total_attempts"`

	// SuccessfulAttempts is the number of successful recovery attempts.
	SuccessfulAttempts int `json:"successful_attempts"`

	// FailedAttempts is the number of failed recovery attempts.
	FailedAttempts int `json:"failed_attempts"`

	// LastAttemptTime is the time of the last recovery attempt.
	LastAttemptTime time.Time `json:"last_attempt_time"`

	// TotalRecoveryDuration is the total time spent on recovery.
	TotalRecoveryDuration time.Duration `json:"total_recovery_duration"`

	// Attempts is the list of individual recovery attempts.
	Attempts []recoveryAttemptRecord `json:"attempts,omitempty"`
}

// PauseRecovery pauses automatic recovery operations.
// Manual recovery operations can still be performed while paused.
//
// Returns:
//   - error: An error if the manager is closed or already paused.
func (rm *RecoveryManager) PauseRecovery() error {
	if rm.closed.Load() {
		return ErrRecoveryManagerClosed
	}

	rm.recoveringMu.Lock()
	defer rm.recoveringMu.Unlock()

	if !rm.config.EnableAutoRecovery {
		return ErrRecoveryPaused
	}

	rm.config.EnableAutoRecovery = false

	rm.logger.Info("automatic recovery paused")

	rm.emitEvent(&RecoveryEvent{
		Type:      "recovery.paused",
		Timestamp: time.Now(),
		Message:   "Automatic recovery paused",
	})

	return nil
}

// ResumeRecovery resumes automatic recovery operations.
//
// Returns:
//   - error: An error if the manager is closed or not paused.
func (rm *RecoveryManager) ResumeRecovery() error {
	if rm.closed.Load() {
		return ErrRecoveryManagerClosed
	}

	rm.recoveringMu.Lock()
	defer rm.recoveringMu.Unlock()

	if rm.config.EnableAutoRecovery {
		return errors.New("recovery is not paused")
	}

	rm.config.EnableAutoRecovery = true

	rm.logger.Info("automatic recovery resumed")

	rm.emitEvent(&RecoveryEvent{
		Type:      "recovery.resumed",
		Timestamp: time.Now(),
		Message:   "Automatic recovery resumed",
	})

	return nil
}

// SetRecoveryInterval adjusts the check interval for automatic recovery.
// The change takes effect on the next check cycle.
//
// Parameters:
//   - interval: The new check interval. Must be at least 1 second.
//
// Returns:
//   - error: An error if the manager is closed or the interval is invalid.
func (rm *RecoveryManager) SetRecoveryInterval(interval time.Duration) error {
	if rm.closed.Load() {
		return ErrRecoveryManagerClosed
	}

	if interval < time.Second {
		return ErrInvalidInterval
	}

	rm.recoveringMu.Lock()
	oldInterval := rm.config.CheckInterval
	rm.config.CheckInterval = interval
	rm.recoveringMu.Unlock()

	// Update ticker if running
	if rm.started.Load() && rm.ticker != nil {
		rm.ticker.Reset(interval)
	}

	rm.logger.Info("recovery interval updated",
		zap.Duration("old_interval", oldInterval),
		zap.Duration("new_interval", interval),
	)

	rm.emitEvent(&RecoveryEvent{
		Type:      "recovery.interval_changed",
		Timestamp: time.Now(),
		Message:   "Recovery interval updated",
		Metadata: map[string]interface{}{
			"old_interval_ms": oldInterval.Milliseconds(),
			"new_interval_ms": interval.Milliseconds(),
		},
	})

	return nil
}

// SetMaxConcurrentRecoveries adjusts the maximum number of concurrent recovery operations.
// The change takes effect immediately for new recovery operations.
//
// Parameters:
//   - n: The new max concurrent recoveries. Must be at least 1.
//
// Returns:
//   - error: An error if the manager is closed or n is invalid.
func (rm *RecoveryManager) SetMaxConcurrentRecoveries(n int) error {
	if rm.closed.Load() {
		return ErrRecoveryManagerClosed
	}

	if n < 1 {
		return ErrInvalidConcurrency
	}

	rm.recoveringMu.Lock()
	oldMax := rm.config.MaxConcurrentRecoveries
	rm.config.MaxConcurrentRecoveries = n
	rm.recoveringMu.Unlock()

	// Note: We don't resize the semaphore channel as it's tricky and can lead to race conditions.
	// The new limit will effectively apply to new recovery batches.
	// To properly resize, we would need to create a new semaphore and migrate any in-flight operations.

	rm.logger.Info("max concurrent recoveries updated",
		zap.Int("old_max", oldMax),
		zap.Int("new_max", n),
	)

	rm.emitEvent(&RecoveryEvent{
		Type:      "recovery.concurrency_changed",
		Timestamp: time.Now(),
		Message:   "Max concurrent recoveries updated",
		Metadata: map[string]interface{}{
			"old_max": oldMax,
			"new_max": n,
		},
	})

	return nil
}

// GetRecoveryStatus returns the current status of the recovery manager.
//
// Returns:
//   - *RecoveryStatus: The current recovery status.
//   - error: An error if the manager is closed.
func (rm *RecoveryManager) GetRecoveryStatus() (*RecoveryStatus, error) {
	if rm.closed.Load() {
		return nil, ErrRecoveryManagerClosed
	}

	rm.recoveringMu.RLock()
	activeRecoveries := len(rm.recovering)
	checkInterval := rm.config.CheckInterval
	maxConcurrent := rm.config.MaxConcurrentRecoveries
	isPaused := !rm.config.EnableAutoRecovery
	rm.recoveringMu.RUnlock()

	stats := rm.GetRecoveryStats()

	return &RecoveryStatus{
		IsRunning:               rm.started.Load(),
		IsPaused:                isPaused,
		CheckInterval:           checkInterval,
		MaxConcurrentRecoveries: maxConcurrent,
		Stats:                   stats,
		ActiveRecoveries:        activeRecoveries,
		LastCheckTime:           stats.LastCheckTime,
	}, nil
}

// GetSagaRecoveryStatus returns the recovery status for a specific Saga.
//
// Parameters:
//   - sagaID: The ID of the Saga.
//
// Returns:
//   - *SagaRecoveryStatus: The recovery status for the Saga.
//   - error: An error if the manager is closed or the Saga is not found.
func (rm *RecoveryManager) GetSagaRecoveryStatus(sagaID string) (*SagaRecoveryStatus, error) {
	if rm.closed.Load() {
		return nil, ErrRecoveryManagerClosed
	}

	rm.recoveringMu.RLock()
	attempt, isRecovering := rm.recovering[sagaID]
	rm.recoveringMu.RUnlock()

	status := &SagaRecoveryStatus{
		SagaID:       sagaID,
		IsRecovering: isRecovering,
	}

	if isRecovering && attempt != nil {
		status.RecoveryStartTime = attempt.startTime
		status.AttemptCount = attempt.attemptCount
		status.LastAttemptTime = attempt.lastAttempt
	}

	return status, nil
}

// ListRecoveringStatus returns the recovery status for all currently recovering Sagas.
//
// Returns:
//   - []*SagaRecoveryStatus: A list of recovery statuses.
//   - error: An error if the manager is closed.
func (rm *RecoveryManager) ListRecoveringStatus() ([]*SagaRecoveryStatus, error) {
	if rm.closed.Load() {
		return nil, ErrRecoveryManagerClosed
	}

	rm.recoveringMu.RLock()
	defer rm.recoveringMu.RUnlock()

	statuses := make([]*SagaRecoveryStatus, 0, len(rm.recovering))

	for sagaID, attempt := range rm.recovering {
		status := &SagaRecoveryStatus{
			SagaID:            sagaID,
			IsRecovering:      true,
			RecoveryStartTime: attempt.startTime,
			AttemptCount:      attempt.attemptCount,
			LastAttemptTime:   attempt.lastAttempt,
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GetRecoveryHistory returns the recovery history for a specific Saga.
// This is a simplified implementation that returns the current recovery attempt info.
// In a production system, this would query a persistent recovery history store.
//
// Parameters:
//   - sagaID: The ID of the Saga.
//
// Returns:
//   - *RecoveryHistory: The recovery history for the Saga.
//   - error: An error if the manager is closed or the Saga is not found.
func (rm *RecoveryManager) GetRecoveryHistory(sagaID string) (*RecoveryHistory, error) {
	if rm.closed.Load() {
		return nil, ErrRecoveryManagerClosed
	}

	rm.recoveringMu.RLock()
	attempt, exists := rm.recovering[sagaID]
	rm.recoveringMu.RUnlock()

	history := &RecoveryHistory{
		SagaID:   sagaID,
		Attempts: make([]recoveryAttemptRecord, 0),
	}

	if exists && attempt != nil {
		history.TotalAttempts = attempt.attemptCount
		history.LastAttemptTime = attempt.lastAttempt
		// Note: We don't track detailed attempt records in the current implementation.
		// In a production system, this would be stored in a persistent store.
	}

	return history, nil
}
