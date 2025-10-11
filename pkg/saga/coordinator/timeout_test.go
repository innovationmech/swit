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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTimeoutDetector tests the creation of a timeout detector.
func TestNewTimeoutDetector(t *testing.T) {
	t.Parallel()

	coordinator := createTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)

	assert.NotNil(t, detector)
	assert.Equal(t, coordinator, detector.coordinator)
	assert.False(t, detector.stopped)
}

// TestStartSagaTimeout_NoTimeout tests starting Saga timeout tracking with no timeout.
func TestStartSagaTimeout_NoTimeout(t *testing.T) {
	t.Parallel()

	coordinator := createTimeoutTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)
	instance := createTimeoutTestInstance()

	ctx := context.Background()
	resultCtx, cancel := detector.StartSagaTimeout(ctx, instance, 0)
	defer cancel()

	assert.NotNil(t, resultCtx)
	assert.Equal(t, 0, detector.GetActiveSagaTimeouts())
}

// TestStartSagaTimeout_WithTimeout tests starting Saga timeout tracking with a timeout.
func TestStartSagaTimeout_WithTimeout(t *testing.T) {
	t.Parallel()

	coordinator := createTimeoutTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)
	instance := createTimeoutTestInstance()

	ctx := context.Background()
	timeout := 100 * time.Millisecond
	resultCtx, cancel := detector.StartSagaTimeout(ctx, instance, timeout)
	defer cancel()

	assert.NotNil(t, resultCtx)
	assert.Equal(t, 1, detector.GetActiveSagaTimeouts())

	// Wait a bit and check elapsed time
	time.Sleep(50 * time.Millisecond)
	elapsed := detector.GetSagaElapsedTime(instance.GetID())
	assert.True(t, elapsed >= 50*time.Millisecond)
	assert.False(t, detector.IsSagaTimedOut(instance.GetID()))
}

// TestStartSagaTimeout_TimeoutOccurs tests that a Saga timeout is detected and handled.
func TestStartSagaTimeout_TimeoutOccurs(t *testing.T) {
	coordinator := createTimeoutTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)
	instance := createTimeoutTestInstance()

	ctx := context.Background()
	timeout := 100 * time.Millisecond
	resultCtx, cancel := detector.StartSagaTimeout(ctx, instance, timeout)
	defer cancel()

	// Wait for timeout to occur
	<-resultCtx.Done()
	assert.ErrorIs(t, resultCtx.Err(), context.DeadlineExceeded)

	// Give the timeout handler a moment to process
	time.Sleep(150 * time.Millisecond)

	// Verify timeout tracker was cleaned up (it gets cleaned up after processing)
	// Note: The timeout flag is set before cleanup, so we check the final instance state
	assert.Equal(t, 0, detector.GetActiveSagaTimeouts())

	// Verify instance state was updated
	assert.Equal(t, saga.StateTimedOut, instance.GetState())
	assert.NotNil(t, instance.GetError())
	assert.Equal(t, saga.ErrorTypeTimeout, instance.GetError().Type)

	// Verify metrics were updated
	metrics := coordinator.GetMetrics()
	assert.Equal(t, int64(1), metrics.TimedOutSagas)
}

// TestStartSagaTimeout_ManualCancel tests manually cancelling a Saga timeout.
func TestStartSagaTimeout_ManualCancel(t *testing.T) {
	t.Parallel()

	coordinator := createTimeoutTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)
	instance := createTimeoutTestInstance()

	ctx := context.Background()
	timeout := 1 * time.Second
	resultCtx, cancel := detector.StartSagaTimeout(ctx, instance, timeout)

	assert.Equal(t, 1, detector.GetActiveSagaTimeouts())

	// Cancel before timeout
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Verify context is cancelled
	assert.Error(t, resultCtx.Err())

	// Verify timeout tracker was cleaned up
	assert.Equal(t, 0, detector.GetActiveSagaTimeouts())
	assert.False(t, detector.IsSagaTimedOut(instance.GetID()))
}

// TestStartStepTimeout_NoTimeout tests starting step timeout tracking with no timeout.
func TestStartStepTimeout_NoTimeout(t *testing.T) {
	t.Parallel()

	coordinator := createTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)

	ctx := context.Background()
	sagaID := "test-saga-1"
	stepIndex := 0
	stepName := "TestStep"

	resultCtx, cancel := detector.StartStepTimeout(ctx, sagaID, stepIndex, stepName, 0)
	defer cancel()

	assert.NotNil(t, resultCtx)
	assert.Equal(t, 0, detector.GetActiveStepTimeouts())
}

// TestStartStepTimeout_WithTimeout tests starting step timeout tracking with a timeout.
func TestStartStepTimeout_WithTimeout(t *testing.T) {
	t.Parallel()

	coordinator := createTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)

	ctx := context.Background()
	sagaID := "test-saga-1"
	stepIndex := 0
	stepName := "TestStep"
	timeout := 100 * time.Millisecond

	resultCtx, cancel := detector.StartStepTimeout(ctx, sagaID, stepIndex, stepName, timeout)
	defer cancel()

	assert.NotNil(t, resultCtx)
	assert.Equal(t, 1, detector.GetActiveStepTimeouts())

	// Wait a bit and check elapsed time
	time.Sleep(50 * time.Millisecond)
	elapsed := detector.GetStepElapsedTime(sagaID, stepIndex)
	assert.True(t, elapsed >= 50*time.Millisecond)
	assert.False(t, detector.IsStepTimedOut(sagaID, stepIndex))
}

// TestStartStepTimeout_TimeoutOccurs tests that a step timeout is detected and handled.
func TestStartStepTimeout_TimeoutOccurs(t *testing.T) {
	coordinator := createTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)

	ctx := context.Background()
	sagaID := "test-saga-1"
	stepIndex := 0
	stepName := "TestStep"
	timeout := 100 * time.Millisecond

	resultCtx, cancel := detector.StartStepTimeout(ctx, sagaID, stepIndex, stepName, timeout)
	defer cancel()

	// Wait for timeout to occur
	<-resultCtx.Done()
	assert.ErrorIs(t, resultCtx.Err(), context.DeadlineExceeded)

	// Give the timeout handler a moment to process
	time.Sleep(150 * time.Millisecond)

	// Verify tracker was cleaned up after timeout
	// The timeout handler cleans up the tracker after processing
	assert.Equal(t, 0, detector.GetActiveStepTimeouts())
}

// TestStartStepTimeout_ManualCancel tests manually cancelling a step timeout.
func TestStartStepTimeout_ManualCancel(t *testing.T) {
	t.Parallel()

	coordinator := createTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)

	ctx := context.Background()
	sagaID := "test-saga-1"
	stepIndex := 0
	stepName := "TestStep"
	timeout := 1 * time.Second

	resultCtx, cancel := detector.StartStepTimeout(ctx, sagaID, stepIndex, stepName, timeout)

	assert.Equal(t, 1, detector.GetActiveStepTimeouts())

	// Cancel before timeout
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Verify context is cancelled
	assert.Error(t, resultCtx.Err())

	// Verify timeout tracker was cleaned up
	assert.Equal(t, 0, detector.GetActiveStepTimeouts())
	assert.False(t, detector.IsStepTimedOut(sagaID, stepIndex))
}

// TestMultipleStepTimeouts tests tracking multiple step timeouts simultaneously.
func TestMultipleStepTimeouts(t *testing.T) {
	t.Parallel()

	coordinator := createTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)

	ctx := context.Background()
	sagaID := "test-saga-1"
	timeout := 200 * time.Millisecond

	// Start multiple step timeouts
	ctx1, cancel1 := detector.StartStepTimeout(ctx, sagaID, 0, "Step1", timeout)
	defer cancel1()
	ctx2, cancel2 := detector.StartStepTimeout(ctx, sagaID, 1, "Step2", timeout)
	defer cancel2()
	ctx3, cancel3 := detector.StartStepTimeout(ctx, sagaID, 2, "Step3", timeout)
	defer cancel3()

	assert.Equal(t, 3, detector.GetActiveStepTimeouts())

	// Cancel one step
	cancel1()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, detector.GetActiveStepTimeouts())

	// Verify contexts
	assert.Error(t, ctx1.Err())
	assert.NoError(t, ctx2.Err())
	assert.NoError(t, ctx3.Err())
}

// TestCleanupTimeouts tests cleaning up all timeout trackers.
func TestCleanupTimeouts(t *testing.T) {
	coordinator := createTimeoutTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)
	instance := createTimeoutTestInstance()

	ctx := context.Background()

	// Start multiple timeouts
	sagaCtx, sagaCancel := detector.StartSagaTimeout(ctx, instance, 1*time.Second)
	defer sagaCancel()
	stepCtx1, stepCancel1 := detector.StartStepTimeout(ctx, instance.GetID(), 0, "Step1", 1*time.Second)
	defer stepCancel1()
	stepCtx2, stepCancel2 := detector.StartStepTimeout(ctx, instance.GetID(), 1, "Step2", 1*time.Second)
	defer stepCancel2()

	assert.Equal(t, 1, detector.GetActiveSagaTimeouts())
	assert.Equal(t, 2, detector.GetActiveStepTimeouts())

	// Cleanup all timeouts
	detector.CleanupTimeouts()

	// Verify all trackers were cleaned up
	assert.Equal(t, 0, detector.GetActiveSagaTimeouts())
	assert.Equal(t, 0, detector.GetActiveStepTimeouts())
	assert.True(t, detector.stopped)

	// Verify contexts were cancelled
	assert.Error(t, sagaCtx.Err())
	assert.Error(t, stepCtx1.Err())
	assert.Error(t, stepCtx2.Err())
}

// TestStopSagaTimeout tests stopping a specific Saga timeout.
func TestStopSagaTimeout(t *testing.T) {
	t.Parallel()

	coordinator := createTimeoutTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)
	instance := createTimeoutTestInstance()

	ctx := context.Background()
	_, cancel := detector.StartSagaTimeout(ctx, instance, 1*time.Second)
	defer cancel()

	assert.Equal(t, 1, detector.GetActiveSagaTimeouts())

	// Stop the timeout
	detector.stopSagaTimeout(instance.GetID())

	assert.Equal(t, 0, detector.GetActiveSagaTimeouts())
}

// TestStopStepTimeout tests stopping a specific step timeout.
func TestStopStepTimeout(t *testing.T) {
	t.Parallel()

	coordinator := createTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)

	ctx := context.Background()
	sagaID := "test-saga-1"
	stepIndex := 0
	_, cancel := detector.StartStepTimeout(ctx, sagaID, stepIndex, "TestStep", 1*time.Second)
	defer cancel()

	assert.Equal(t, 1, detector.GetActiveStepTimeouts())

	// Stop the timeout
	detector.stopStepTimeout(sagaID, stepIndex)

	assert.Equal(t, 0, detector.GetActiveStepTimeouts())
}

// TestGetElapsedTime_NonExistent tests getting elapsed time for non-existent trackers.
func TestGetElapsedTime_NonExistent(t *testing.T) {
	t.Parallel()

	coordinator := createTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)

	// Test non-existent Saga
	elapsed := detector.GetSagaElapsedTime("non-existent-saga")
	assert.Equal(t, time.Duration(0), elapsed)

	// Test non-existent step
	elapsed = detector.GetStepElapsedTime("non-existent-saga", 0)
	assert.Equal(t, time.Duration(0), elapsed)
}

// TestIsTimedOut_NonExistent tests checking timeout status for non-existent trackers.
func TestIsTimedOut_NonExistent(t *testing.T) {
	t.Parallel()

	coordinator := createTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)

	// Test non-existent Saga
	timedOut := detector.IsSagaTimedOut("non-existent-saga")
	assert.False(t, timedOut)

	// Test non-existent step
	timedOut = detector.IsStepTimedOut("non-existent-saga", 0)
	assert.False(t, timedOut)
}

// TestTimeoutDetector_AfterStopped tests that the detector doesn't track timeouts after being stopped.
func TestTimeoutDetector_AfterStopped(t *testing.T) {
	t.Parallel()

	coordinator := createTimeoutTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)
	instance := createTimeoutTestInstance()

	// Stop the detector
	detector.CleanupTimeouts()

	ctx := context.Background()

	// Try to start timeouts after stopped
	sagaCtx, sagaCancel := detector.StartSagaTimeout(ctx, instance, 1*time.Second)
	defer sagaCancel()
	stepCtx, stepCancel := detector.StartStepTimeout(ctx, instance.GetID(), 0, "Step1", 1*time.Second)
	defer stepCancel()

	// Verify no trackers were created
	assert.Equal(t, 0, detector.GetActiveSagaTimeouts())
	assert.Equal(t, 0, detector.GetActiveStepTimeouts())

	// Verify contexts are still usable (just without timeout tracking)
	assert.NotNil(t, sagaCtx)
	assert.NotNil(t, stepCtx)
}

// TestSagaTimeout_EventPublishing tests that timeout events are published correctly.
func TestSagaTimeout_EventPublishing(t *testing.T) {
	coordinator := createTimeoutTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)
	instance := createTimeoutTestInstance()

	// Get the mock event publisher
	mockPublisher := coordinator.eventPublisher.(*mockEventPublisherWithTracking)
	_ = mockPublisher // used for checking events

	ctx := context.Background()
	timeout := 100 * time.Millisecond
	resultCtx, cancel := detector.StartSagaTimeout(ctx, instance, timeout)
	defer cancel()

	// Wait for timeout to occur
	<-resultCtx.Done()
	time.Sleep(50 * time.Millisecond)

	// Verify timeout event was published
	events := mockPublisher.GetEvents()
	found := false
	for _, event := range events {
		if event.Type == saga.EventSagaTimedOut {
			found = true
			assert.Equal(t, instance.GetID(), event.SagaID)
			assert.NotNil(t, event.Error)
			assert.Equal(t, saga.ErrorTypeTimeout, event.Error.Type)
			break
		}
	}
	assert.True(t, found, "Timeout event should be published")
}

// TestStepTimeout_EventPublishing tests that step timeout events are published correctly.
func TestStepTimeout_EventPublishing(t *testing.T) {
	coordinator := createTimeoutTestCoordinator(t)
	detector := newTimeoutDetector(coordinator)

	// Get the mock event publisher
	mockPublisher := coordinator.eventPublisher.(*mockEventPublisherWithTracking)
	_ = mockPublisher // used for checking events

	ctx := context.Background()
	sagaID := "test-saga-1"
	stepIndex := 0
	stepName := "TestStep"
	timeout := 100 * time.Millisecond

	resultCtx, cancel := detector.StartStepTimeout(ctx, sagaID, stepIndex, stepName, timeout)
	defer cancel()

	// Wait for timeout to occur
	<-resultCtx.Done()
	time.Sleep(50 * time.Millisecond)

	// Verify timeout event was published (as a step failed event)
	events := mockPublisher.GetEvents()
	found := false
	for _, event := range events {
		if event.Type == saga.EventSagaStepFailed {
			found = true
			assert.Equal(t, sagaID, event.SagaID)
			assert.NotNil(t, event.Error)
			assert.Equal(t, saga.ErrorTypeTimeout, event.Error.Type)
			break
		}
	}
	assert.True(t, found, "Step timeout event should be published")
}

// createTimeoutTestCoordinator creates a test coordinator for timeout tests.
func createTimeoutTestCoordinator(t *testing.T) *OrchestratorCoordinator {
	config := &OrchestratorConfig{
		StateStorage:   newMockStateStorage(),
		EventPublisher: newMockEventPublisher(),
		RetryPolicy:    saga.NewFixedDelayRetryPolicy(3, 100*time.Millisecond),
	}

	coordinator, err := NewOrchestratorCoordinator(config)
	require.NoError(t, err)
	require.NotNil(t, coordinator)

	return coordinator
}

// createTimeoutTestInstance creates a test Saga instance for timeout tests.
func createTimeoutTestInstance() *OrchestratorSagaInstance {
	now := time.Now()
	return &OrchestratorSagaInstance{
		id:             "test-saga-instance-1",
		definitionID:   "test-definition-1",
		name:           "Test Saga",
		description:    "A test Saga instance",
		state:          saga.StateRunning,
		currentStep:    0,
		completedSteps: 0,
		totalSteps:     3,
		createdAt:      now,
		updatedAt:      now,
		startedAt:      &now,
		timeout:        1 * time.Minute,
		retryPolicy:    saga.NewFixedDelayRetryPolicy(3, 100*time.Millisecond),
		metadata:       make(map[string]interface{}),
	}
}
