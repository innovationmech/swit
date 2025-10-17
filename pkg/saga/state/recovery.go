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
	"sync"
	"sync/atomic"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

var (
	// ErrRecoveryManagerClosed is returned when operations are attempted on a closed RecoveryManager.
	ErrRecoveryManagerClosed = errors.New("recovery manager is closed")

	// ErrRecoveryManagerNotStarted is returned when operations require the manager to be started.
	ErrRecoveryManagerNotStarted = errors.New("recovery manager not started")

	// ErrInvalidRecoveryConfig is returned when the recovery config is invalid.
	ErrInvalidRecoveryConfig = errors.New("invalid recovery config")

	// ErrStateStorageRequired is returned when state storage is not provided.
	ErrStateStorageRequired = errors.New("state storage is required")

	// ErrCoordinatorRequired is returned when coordinator is not provided.
	ErrCoordinatorRequired = errors.New("coordinator is required")

	// ErrRecoveryInProgress is returned when recovery is already in progress for a Saga.
	ErrRecoveryInProgress = errors.New("recovery already in progress")

	// ErrMaxRecoveryAttemptsExceeded is returned when max recovery attempts are exceeded.
	ErrMaxRecoveryAttemptsExceeded = errors.New("max recovery attempts exceeded")
)

// RecoveryEventType represents the type of recovery event.
type RecoveryEventType string

const (
	// RecoveryEventStarted is emitted when recovery starts.
	RecoveryEventStarted RecoveryEventType = "recovery.started"

	// RecoveryEventCheckStarted is emitted when a recovery check cycle starts.
	RecoveryEventCheckStarted RecoveryEventType = "recovery.check.started"

	// RecoveryEventCheckCompleted is emitted when a recovery check cycle completes.
	RecoveryEventCheckCompleted RecoveryEventType = "recovery.check.completed"

	// RecoveryEventCheckFailed is emitted when a recovery check cycle fails.
	RecoveryEventCheckFailed RecoveryEventType = "recovery.check.failed"

	// RecoveryEventSagaRecovered is emitted when a Saga is successfully recovered.
	RecoveryEventSagaRecovered RecoveryEventType = "recovery.saga.recovered"

	// RecoveryEventSagaRecoveryFailed is emitted when Saga recovery fails.
	RecoveryEventSagaRecoveryFailed RecoveryEventType = "recovery.saga.failed"

	// RecoveryEventStopped is emitted when recovery is stopped.
	RecoveryEventStopped RecoveryEventType = "recovery.stopped"
)

// RecoveryEvent represents a recovery event.
type RecoveryEvent struct {
	Type      RecoveryEventType      `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	SagaID    string                 `json:"saga_id,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Error     error                  `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// RecoveryStats holds statistics about recovery operations.
type RecoveryStats struct {
	// TotalChecks is the total number of recovery check cycles performed.
	TotalChecks int64 `json:"total_checks"`

	// TotalRecoveryAttempts is the total number of recovery attempts.
	TotalRecoveryAttempts int64 `json:"total_recovery_attempts"`

	// SuccessfulRecoveries is the number of successful recoveries.
	SuccessfulRecoveries int64 `json:"successful_recoveries"`

	// FailedRecoveries is the number of failed recoveries.
	FailedRecoveries int64 `json:"failed_recoveries"`

	// CurrentlyRecovering is the number of Sagas currently being recovered.
	CurrentlyRecovering int64 `json:"currently_recovering"`

	// LastCheckTime is the timestamp of the last recovery check.
	LastCheckTime time.Time `json:"last_check_time"`

	// LastRecoveryTime is the timestamp of the last successful recovery.
	LastRecoveryTime time.Time `json:"last_recovery_time"`

	// AverageRecoveryDuration is the average time taken for recovery operations.
	AverageRecoveryDuration time.Duration `json:"average_recovery_duration"`
}

// RecoveryManager manages the automatic recovery of failed Saga instances.
// It periodically checks for Sagas that need recovery and attempts to recover them.
type RecoveryManager struct {
	// config holds the recovery configuration.
	config *RecoveryConfig

	// stateStorage provides access to Saga state persistence.
	stateStorage saga.StateStorage

	// coordinator is the Saga coordinator for recovery operations.
	coordinator saga.SagaCoordinator

	// logger is the structured logger for recovery operations.
	logger *zap.Logger

	// stats holds recovery statistics.
	stats *RecoveryStats

	// statsMu protects access to stats.
	statsMu sync.RWMutex

	// recoveringMu protects the recovering map.
	recoveringMu sync.RWMutex

	// recovering tracks Sagas currently being recovered.
	recovering map[string]*recoveryAttempt

	// semaphore limits concurrent recovery operations.
	semaphore chan struct{}

	// ticker triggers periodic recovery checks.
	ticker *time.Ticker

	// stopCh signals the recovery loop to stop.
	stopCh chan struct{}

	// doneCh signals that the recovery loop has stopped.
	doneCh chan struct{}

	// started indicates whether the manager has been started.
	started atomic.Bool

	// closed indicates whether the manager has been closed.
	closed atomic.Bool

	// eventListeners holds registered event listeners.
	eventListeners []RecoveryEventListener

	// listenersMu protects access to eventListeners.
	listenersMu sync.RWMutex
}

// recoveryAttempt tracks a single recovery attempt.
type recoveryAttempt struct {
	sagaID       string
	startTime    time.Time
	attemptCount int
	lastAttempt  time.Time
}

// RecoveryEventListener is a callback function for recovery events.
type RecoveryEventListener func(event *RecoveryEvent)

// NewRecoveryManager creates a new RecoveryManager instance.
//
// Parameters:
//   - config: Recovery configuration. If nil, default configuration is used.
//   - stateStorage: State storage for accessing Saga instances. Required.
//   - coordinator: Saga coordinator for recovery operations. Required.
//   - logger: Structured logger. If nil, a no-op logger is used.
//
// Returns:
//   - *RecoveryManager: The created recovery manager.
//   - error: An error if validation fails.
func NewRecoveryManager(
	config *RecoveryConfig,
	stateStorage saga.StateStorage,
	coordinator saga.SagaCoordinator,
	logger *zap.Logger,
) (*RecoveryManager, error) {
	// Validate required dependencies
	if stateStorage == nil {
		return nil, ErrStateStorageRequired
	}

	if coordinator == nil {
		return nil, ErrCoordinatorRequired
	}

	// Use default config if not provided
	if config == nil {
		config = DefaultRecoveryConfig()
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRecoveryConfig, err)
	}

	// Use no-op logger if not provided
	if logger == nil {
		logger = zap.NewNop()
	}

	rm := &RecoveryManager{
		config:         config,
		stateStorage:   stateStorage,
		coordinator:    coordinator,
		logger:         logger.With(zap.String("component", "recovery_manager")),
		stats:          &RecoveryStats{},
		recovering:     make(map[string]*recoveryAttempt),
		semaphore:      make(chan struct{}, config.MaxConcurrentRecoveries),
		stopCh:         make(chan struct{}),
		doneCh:         make(chan struct{}),
		eventListeners: make([]RecoveryEventListener, 0),
	}

	rm.logger.Info("recovery manager created",
		zap.Duration("check_interval", config.CheckInterval),
		zap.Duration("recovery_timeout", config.RecoveryTimeout),
		zap.Int("max_concurrent_recoveries", config.MaxConcurrentRecoveries),
		zap.Bool("auto_recovery_enabled", config.EnableAutoRecovery),
	)

	return rm, nil
}

// Start begins the automatic recovery process.
// If auto-recovery is disabled, this method returns immediately.
//
// Returns:
//   - error: An error if the manager is already started or closed.
func (rm *RecoveryManager) Start(ctx context.Context) error {
	if rm.closed.Load() {
		return ErrRecoveryManagerClosed
	}

	if rm.started.Swap(true) {
		return errors.New("recovery manager already started")
	}

	rm.logger.Info("starting recovery manager")
	rm.emitEvent(&RecoveryEvent{
		Type:      RecoveryEventStarted,
		Timestamp: time.Now(),
		Message:   "Recovery manager started",
	})

	// If auto-recovery is disabled, just mark as started
	if !rm.config.EnableAutoRecovery {
		rm.logger.Info("auto-recovery disabled, recovery manager will operate in manual mode only")
		return nil
	}

	// Start the recovery loop in a goroutine
	go rm.recoveryLoop(ctx)

	return nil
}

// Stop gracefully stops the automatic recovery process.
// It waits for ongoing recovery operations to complete.
//
// Returns:
//   - error: An error if the manager is not started or already closed.
func (rm *RecoveryManager) Stop() error {
	if rm.closed.Load() {
		return ErrRecoveryManagerClosed
	}

	if !rm.started.Load() {
		return ErrRecoveryManagerNotStarted
	}

	rm.logger.Info("stopping recovery manager")

	// Signal the recovery loop to stop
	close(rm.stopCh)

	// Wait for the recovery loop to finish
	if rm.config.EnableAutoRecovery {
		<-rm.doneCh
	}

	rm.started.Store(false)

	rm.emitEvent(&RecoveryEvent{
		Type:      RecoveryEventStopped,
		Timestamp: time.Now(),
		Message:   "Recovery manager stopped",
	})

	rm.logger.Info("recovery manager stopped")
	return nil
}

// Close closes the RecoveryManager and releases all resources.
// It first stops the recovery process if it's running.
//
// Returns:
//   - error: An error if closing fails.
func (rm *RecoveryManager) Close() error {
	if rm.closed.Swap(true) {
		return nil // Already closed
	}

	rm.logger.Info("closing recovery manager")

	// Stop if still running
	if rm.started.Load() {
		if err := rm.Stop(); err != nil && !errors.Is(err, ErrRecoveryManagerNotStarted) {
			rm.logger.Error("error stopping recovery manager during close", zap.Error(err))
		}
	}

	// Stop ticker if it exists
	if rm.ticker != nil {
		rm.ticker.Stop()
	}

	rm.logger.Info("recovery manager closed")
	return nil
}

// RecoverSaga manually triggers recovery for a specific Saga instance.
// This can be used even when auto-recovery is disabled.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - sagaID: The ID of the Saga to recover.
//
// Returns:
//   - error: An error if recovery fails or if the Saga is already being recovered.
func (rm *RecoveryManager) RecoverSaga(ctx context.Context, sagaID string) error {
	if rm.closed.Load() {
		return ErrRecoveryManagerClosed
	}

	rm.logger.Info("manual recovery triggered", zap.String("saga_id", sagaID))

	// Check if already recovering
	rm.recoveringMu.Lock()
	if _, exists := rm.recovering[sagaID]; exists {
		rm.recoveringMu.Unlock()
		return fmt.Errorf("%w: saga %s", ErrRecoveryInProgress, sagaID)
	}

	// Mark as recovering - use attemptCount 1 for new attempts
	// The actual attempt tracking happens in recordRecoveryAttempt
	rm.recovering[sagaID] = &recoveryAttempt{
		sagaID:       sagaID,
		startTime:    time.Now(),
		attemptCount: 0, // Will be incremented in recordRecoveryAttempt
		lastAttempt:  time.Now(),
	}
	rm.recoveringMu.Unlock()

	// Ensure cleanup
	defer func() {
		rm.recoveringMu.Lock()
		delete(rm.recovering, sagaID)
		rm.recoveringMu.Unlock()
	}()

	// Perform recovery with timeout
	recoverCtx, cancel := context.WithTimeout(ctx, rm.config.RecoveryTimeout)
	defer cancel()

	return rm.recoverSagaInstance(recoverCtx, sagaID)
}

// GetRecoveryStats returns the current recovery statistics.
//
// Returns:
//   - *RecoveryStats: A copy of the current statistics.
func (rm *RecoveryManager) GetRecoveryStats() *RecoveryStats {
	rm.statsMu.RLock()
	defer rm.statsMu.RUnlock()

	// Return a copy to prevent external modification
	statsCopy := *rm.stats
	return &statsCopy
}

// AddEventListener registers a listener for recovery events.
//
// Parameters:
//   - listener: The callback function to invoke for recovery events.
func (rm *RecoveryManager) AddEventListener(listener RecoveryEventListener) {
	if listener == nil {
		return
	}

	rm.listenersMu.Lock()
	defer rm.listenersMu.Unlock()

	rm.eventListeners = append(rm.eventListeners, listener)
}

// recoveryLoop is the main loop that periodically checks for and recovers failed Sagas.
func (rm *RecoveryManager) recoveryLoop(ctx context.Context) {
	defer close(rm.doneCh)

	rm.ticker = time.NewTicker(rm.config.CheckInterval)
	defer rm.ticker.Stop()

	rm.logger.Info("recovery loop started",
		zap.Duration("check_interval", rm.config.CheckInterval),
	)

	for {
		select {
		case <-ctx.Done():
			rm.logger.Info("recovery loop stopped by context cancellation")
			return

		case <-rm.stopCh:
			rm.logger.Info("recovery loop stopped by stop signal")
			return

		case <-rm.ticker.C:
			if err := rm.performRecoveryCheck(ctx); err != nil {
				rm.logger.Error("recovery check failed", zap.Error(err))
			}
		}
	}
}

// performRecoveryCheck performs a single recovery check cycle.
func (rm *RecoveryManager) performRecoveryCheck(ctx context.Context) error {
	checkStart := time.Now()

	rm.logger.Debug("starting recovery check")
	rm.emitEvent(&RecoveryEvent{
		Type:      RecoveryEventCheckStarted,
		Timestamp: checkStart,
		Message:   "Recovery check started",
	})

	// Update stats
	rm.statsMu.Lock()
	rm.stats.TotalChecks++
	rm.stats.LastCheckTime = checkStart
	rm.statsMu.Unlock()

	// Perform fault detection
	detectionResults, err := rm.performDetection(ctx)
	if err != nil {
		rm.logger.Error("detection failed", zap.Error(err))
		rm.emitEvent(&RecoveryEvent{
			Type:      RecoveryEventCheckFailed,
			Timestamp: time.Now(),
			Message:   "Recovery check failed during detection",
			Error:     err,
		})
		return fmt.Errorf("detection failed: %w", err)
	}

	// Log detection results
	detectionStats := make(map[string]int)
	for _, result := range detectionResults {
		detectionStats[string(result.Type)]++
	}

	rm.logger.Info("detection completed",
		zap.Int("total_detected", len(detectionResults)),
		zap.Any("by_type", detectionStats),
	)

	// Trigger recovery for detected Sagas
	if len(detectionResults) > 0 {
		if err := rm.performRecoveryBatch(ctx, detectionResults); err != nil {
			rm.logger.Error("recovery batch failed", zap.Error(err))
			// Don't fail the entire check if batch recovery fails
		}
	}

	checkDuration := time.Since(checkStart)
	rm.logger.Debug("recovery check completed",
		zap.Duration("duration", checkDuration),
		zap.Int("detected_sagas", len(detectionResults)),
	)

	rm.emitEvent(&RecoveryEvent{
		Type:      RecoveryEventCheckCompleted,
		Timestamp: time.Now(),
		Message:   "Recovery check completed",
		Metadata: map[string]interface{}{
			"duration_ms":     checkDuration.Milliseconds(),
			"detected_count":  len(detectionResults),
			"detection_stats": detectionStats,
		},
	})

	return nil
}

// Note: recoverSagaInstance is now implemented in recovery_execution.go

// emitEvent notifies all registered listeners about a recovery event.
func (rm *RecoveryManager) emitEvent(event *RecoveryEvent) {
	rm.listenersMu.RLock()
	listeners := make([]RecoveryEventListener, len(rm.eventListeners))
	copy(listeners, rm.eventListeners)
	rm.listenersMu.RUnlock()

	// Notify listeners asynchronously to prevent blocking
	for _, listener := range listeners {
		go func(l RecoveryEventListener) {
			defer func() {
				if r := recover(); r != nil {
					rm.logger.Error("recovery event listener panic",
						zap.Any("panic", r),
						zap.String("event_type", string(event.Type)),
					)
				}
			}()
			l(event)
		}(listener)
	}
}
