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

package coordinator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/saga"
)

// TimeoutDetector is responsible for detecting and handling timeouts
// for both individual steps and entire Saga instances.
type TimeoutDetector struct {
	coordinator *OrchestratorCoordinator
	
	// stepTimeouts tracks ongoing step timeouts
	stepTimeouts sync.Map // map[string]*stepTimeoutTracker
	
	// sagaTimeouts tracks ongoing Saga timeouts
	sagaTimeouts sync.Map // map[string]*sagaTimeoutTracker
	
	// stopped indicates if the detector has been stopped
	stopped bool
	stopMu  sync.RWMutex
}

// stepTimeoutTracker tracks timeout information for a single step execution.
type stepTimeoutTracker struct {
	sagaID       string
	stepIndex    int
	stepName     string
	startTime    time.Time
	timeout      time.Duration
	cancel       context.CancelFunc
	timedOut     bool
	timedOutAt   *time.Time
	mu           sync.RWMutex
}

// sagaTimeoutTracker tracks timeout information for an entire Saga instance.
type sagaTimeoutTracker struct {
	sagaID       string
	startTime    time.Time
	timeout      time.Duration
	cancel       context.CancelFunc
	timedOut     bool
	timedOutAt   *time.Time
	mu           sync.RWMutex
}

// newTimeoutDetector creates a new timeout detector for the coordinator.
func newTimeoutDetector(coordinator *OrchestratorCoordinator) *TimeoutDetector {
	return &TimeoutDetector{
		coordinator: coordinator,
		stopped:     false,
	}
}

// StartSagaTimeout starts tracking timeout for a Saga instance.
// It creates a context with timeout and monitors the Saga execution.
//
// Parameters:
//   - ctx: Parent context for the Saga execution
//   - instance: The Saga instance to monitor
//   - timeout: Timeout duration for the entire Saga
//
// Returns:
//   - A derived context that will be cancelled on timeout
//   - A cancel function to manually cancel the timeout
func (td *TimeoutDetector) StartSagaTimeout(
	ctx context.Context,
	instance *OrchestratorSagaInstance,
	timeout time.Duration,
) (context.Context, context.CancelFunc) {
	td.stopMu.RLock()
	if td.stopped {
		td.stopMu.RUnlock()
		// If detector is stopped, just return the original context
		return context.WithCancel(ctx)
	}
	td.stopMu.RUnlock()

	// If no timeout is specified, return context without timeout
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)

	// Create tracker
	tracker := &sagaTimeoutTracker{
		sagaID:    instance.GetID(),
		startTime: time.Now(),
		timeout:   timeout,
		cancel:    cancel,
		timedOut:  false,
	}

	// Store tracker
	td.sagaTimeouts.Store(instance.GetID(), tracker)

	// Start monitoring in background
	go td.monitorSagaTimeout(timeoutCtx, tracker, instance)

	// Return wrapped cancel function that cleans up tracker
	wrappedCancel := func() {
		cancel()
		td.stopSagaTimeout(instance.GetID())
	}

	return timeoutCtx, wrappedCancel
}

// monitorSagaTimeout monitors a Saga instance for timeout.
func (td *TimeoutDetector) monitorSagaTimeout(
	ctx context.Context,
	tracker *sagaTimeoutTracker,
	instance *OrchestratorSagaInstance,
) {
	<-ctx.Done()
	// Check if this is a timeout or regular cancellation
	if ctx.Err() == context.DeadlineExceeded {
		// Timeout occurred
		td.handleSagaTimeout(tracker, instance)
	}
	// Clean up tracker
	td.stopSagaTimeout(tracker.sagaID)
}

// handleSagaTimeout handles a Saga timeout by updating state and triggering compensation.
func (td *TimeoutDetector) handleSagaTimeout(
	tracker *sagaTimeoutTracker,
	instance *OrchestratorSagaInstance,
) {
	tracker.mu.Lock()
	if tracker.timedOut {
		// Already handled
		tracker.mu.Unlock()
		return
	}
	tracker.timedOut = true
	now := time.Now()
	tracker.timedOutAt = &now
	tracker.mu.Unlock()

	logger.GetSugaredLogger().Warnf(
		"Saga %s timed out after %v (timeout: %v)",
		tracker.sagaID,
		time.Since(tracker.startTime),
		tracker.timeout,
	)

	// Update instance state to TimedOut
	instance.mu.Lock()
	instance.state = saga.StateTimedOut
	instance.timedOutAt = &now
	instance.updatedAt = now
	
	// Create timeout error
	instance.sagaError = &saga.SagaError{
		Code:      "SAGA_TIMEOUT",
		Message:   fmt.Sprintf("Saga execution exceeded timeout of %v", tracker.timeout),
		Type:      saga.ErrorTypeTimeout,
		Retryable: false,
		Timestamp: now,
		Details: map[string]interface{}{
			"timeout":     tracker.timeout.String(),
			"elapsed":     time.Since(tracker.startTime).String(),
			"saga_id":     tracker.sagaID,
		},
	}
	instance.mu.Unlock()

	// Persist state change
	ctx := context.Background()
	if err := td.coordinator.stateStorage.UpdateSagaState(ctx, tracker.sagaID, saga.StateTimedOut, nil); err != nil {
		logger.GetSugaredLogger().Errorf("Failed to update Saga state to TimedOut: %v", err)
	}

	// Update coordinator metrics
	td.coordinator.mu.Lock()
	td.coordinator.metrics.TimedOutSagas++
	td.coordinator.metrics.ActiveSagas--
	td.coordinator.metrics.LastUpdateTime = time.Now()
	td.coordinator.mu.Unlock()

	// Record metrics
	duration := time.Since(tracker.startTime)
	td.coordinator.metricsCollector.RecordSagaTimedOut(instance.GetDefinitionID(), duration)

	// Publish timeout event
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    tracker.sagaID,
		Type:      saga.EventSagaTimedOut,
		Version:   "1.0",
		Timestamp: now,
		Error:     instance.sagaError,
		Duration:  duration,
		Data: map[string]interface{}{
			"timeout":         tracker.timeout.String(),
			"elapsed":         duration.String(),
			"current_step":    instance.GetCurrentStep(),
			"completed_steps": instance.GetCompletedSteps(),
			"total_steps":     instance.GetTotalSteps(),
		},
		TraceID:  instance.GetTraceID(),
		SpanID:   instance.spanID,
		Source:   "TimeoutDetector",
		Metadata: instance.GetMetadata(),
	}

	if err := td.coordinator.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish timeout event: %v", err)
	}

	// Trigger compensation if any steps were completed
	if instance.GetCompletedSteps() > 0 {
		logger.GetSugaredLogger().Infof("Triggering compensation for timed out Saga %s", tracker.sagaID)
		// Compensation will be handled by the step executor's error handling logic
		// when it detects the timeout state
	}
}

// stopSagaTimeout stops tracking timeout for a Saga instance.
func (td *TimeoutDetector) stopSagaTimeout(sagaID string) {
	td.sagaTimeouts.Delete(sagaID)
}

// StartStepTimeout starts tracking timeout for a step execution.
// It monitors the step and cancels execution if timeout is exceeded.
//
// Parameters:
//   - ctx: Parent context for the step execution
//   - sagaID: The Saga instance ID
//   - stepIndex: Index of the step being executed
//   - stepName: Name of the step
//   - timeout: Timeout duration for the step
//
// Returns:
//   - A derived context that will be cancelled on timeout
//   - A cancel function to manually cancel the timeout
func (td *TimeoutDetector) StartStepTimeout(
	ctx context.Context,
	sagaID string,
	stepIndex int,
	stepName string,
	timeout time.Duration,
) (context.Context, context.CancelFunc) {
	td.stopMu.RLock()
	if td.stopped {
		td.stopMu.RUnlock()
		// If detector is stopped, just return the original context
		return context.WithCancel(ctx)
	}
	td.stopMu.RUnlock()

	// If no timeout is specified, return context without timeout
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)

	// Create tracker
	trackerKey := fmt.Sprintf("%s-step-%d", sagaID, stepIndex)
	tracker := &stepTimeoutTracker{
		sagaID:    sagaID,
		stepIndex: stepIndex,
		stepName:  stepName,
		startTime: time.Now(),
		timeout:   timeout,
		cancel:    cancel,
		timedOut:  false,
	}

	// Store tracker
	td.stepTimeouts.Store(trackerKey, tracker)

	// Start monitoring in background
	go td.monitorStepTimeout(timeoutCtx, tracker)

	// Return wrapped cancel function that cleans up tracker
	wrappedCancel := func() {
		cancel()
		td.stopStepTimeout(sagaID, stepIndex)
	}

	return timeoutCtx, wrappedCancel
}

// monitorStepTimeout monitors a step execution for timeout.
func (td *TimeoutDetector) monitorStepTimeout(
	ctx context.Context,
	tracker *stepTimeoutTracker,
) {
	<-ctx.Done()
	// Check if this is a timeout or regular cancellation
	if ctx.Err() == context.DeadlineExceeded {
		// Timeout occurred
		td.handleStepTimeout(tracker)
	}
	// Clean up tracker
	td.stopStepTimeout(tracker.sagaID, tracker.stepIndex)
}

// handleStepTimeout handles a step timeout by recording the timeout and publishing events.
func (td *TimeoutDetector) handleStepTimeout(tracker *stepTimeoutTracker) {
	tracker.mu.Lock()
	if tracker.timedOut {
		// Already handled
		tracker.mu.Unlock()
		return
	}
	tracker.timedOut = true
	now := time.Now()
	tracker.timedOutAt = &now
	tracker.mu.Unlock()

	logger.GetSugaredLogger().Warnf(
		"Step %d (%s) of Saga %s timed out after %v (timeout: %v)",
		tracker.stepIndex,
		tracker.stepName,
		tracker.sagaID,
		time.Since(tracker.startTime),
		tracker.timeout,
	)

	// Publish step timeout event (as a step failed event with timeout error)
	ctx := context.Background()
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    tracker.sagaID,
		StepID:    fmt.Sprintf("%s-step-%d", tracker.sagaID, tracker.stepIndex),
		Type:      saga.EventSagaStepFailed,
		Version:   "1.0",
		Timestamp: now,
		Error: &saga.SagaError{
			Code:      "STEP_TIMEOUT",
			Message:   fmt.Sprintf("Step %s execution exceeded timeout of %v", tracker.stepName, tracker.timeout),
			Type:      saga.ErrorTypeTimeout,
			Retryable: false,
			Timestamp: now,
			Details: map[string]interface{}{
				"step_index": tracker.stepIndex,
				"step_name":  tracker.stepName,
				"timeout":    tracker.timeout.String(),
				"elapsed":    time.Since(tracker.startTime).String(),
			},
		},
		Duration: time.Since(tracker.startTime),
		Data: map[string]interface{}{
			"step_index": tracker.stepIndex,
			"step_name":  tracker.stepName,
			"timeout":    tracker.timeout.String(),
			"elapsed":    time.Since(tracker.startTime).String(),
		},
		Source: "TimeoutDetector",
	}

	if err := td.coordinator.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish step timeout event: %v", err)
	}
}

// stopStepTimeout stops tracking timeout for a step execution.
func (td *TimeoutDetector) stopStepTimeout(sagaID string, stepIndex int) {
	trackerKey := fmt.Sprintf("%s-step-%d", sagaID, stepIndex)
	td.stepTimeouts.Delete(trackerKey)
}

// IsStepTimedOut checks if a specific step has timed out.
func (td *TimeoutDetector) IsStepTimedOut(sagaID string, stepIndex int) bool {
	trackerKey := fmt.Sprintf("%s-step-%d", sagaID, stepIndex)
	if value, ok := td.stepTimeouts.Load(trackerKey); ok {
		tracker := value.(*stepTimeoutTracker)
		tracker.mu.RLock()
		defer tracker.mu.RUnlock()
		return tracker.timedOut
	}
	return false
}

// IsSagaTimedOut checks if a Saga instance has timed out.
func (td *TimeoutDetector) IsSagaTimedOut(sagaID string) bool {
	if value, ok := td.sagaTimeouts.Load(sagaID); ok {
		tracker := value.(*sagaTimeoutTracker)
		tracker.mu.RLock()
		defer tracker.mu.RUnlock()
		return tracker.timedOut
	}
	return false
}

// GetStepElapsedTime returns the elapsed time for a step execution.
func (td *TimeoutDetector) GetStepElapsedTime(sagaID string, stepIndex int) time.Duration {
	trackerKey := fmt.Sprintf("%s-step-%d", sagaID, stepIndex)
	if value, ok := td.stepTimeouts.Load(trackerKey); ok {
		tracker := value.(*stepTimeoutTracker)
		tracker.mu.RLock()
		defer tracker.mu.RUnlock()
		return time.Since(tracker.startTime)
	}
	return 0
}

// GetSagaElapsedTime returns the elapsed time for a Saga instance.
func (td *TimeoutDetector) GetSagaElapsedTime(sagaID string) time.Duration {
	if value, ok := td.sagaTimeouts.Load(sagaID); ok {
		tracker := value.(*sagaTimeoutTracker)
		tracker.mu.RLock()
		defer tracker.mu.RUnlock()
		return time.Since(tracker.startTime)
	}
	return 0
}

// CleanupTimeouts removes all timeout trackers (called when coordinator closes).
func (td *TimeoutDetector) CleanupTimeouts() {
	td.stopMu.Lock()
	td.stopped = true
	td.stopMu.Unlock()

	// Cancel all step timeouts
	td.stepTimeouts.Range(func(key, value interface{}) bool {
		tracker := value.(*stepTimeoutTracker)
		tracker.mu.Lock()
		if tracker.cancel != nil {
			tracker.cancel()
		}
		tracker.mu.Unlock()
		td.stepTimeouts.Delete(key)
		return true
	})

	// Cancel all Saga timeouts
	td.sagaTimeouts.Range(func(key, value interface{}) bool {
		tracker := value.(*sagaTimeoutTracker)
		tracker.mu.Lock()
		if tracker.cancel != nil {
			tracker.cancel()
		}
		tracker.mu.Unlock()
		td.sagaTimeouts.Delete(key)
		return true
	})

	logger.GetSugaredLogger().Info("TimeoutDetector cleanup completed")
}

// GetActiveStepTimeouts returns the count of active step timeout trackers.
func (td *TimeoutDetector) GetActiveStepTimeouts() int {
	count := 0
	td.stepTimeouts.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// GetActiveSagaTimeouts returns the count of active Saga timeout trackers.
func (td *TimeoutDetector) GetActiveSagaTimeouts() int {
	count := 0
	td.sagaTimeouts.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

