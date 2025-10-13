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

// Package examples provides usage examples for the Saga state management package.
package examples

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/state"
	"github.com/innovationmech/swit/pkg/saga/state/storage"
)

// BasicUsageExample demonstrates basic state storage and manager usage.
func BasicUsageExample() {
	// 1. Create storage backend
	memStorage := storage.NewMemoryStateStorage()
	defer memStorage.Close()

	// 2. Create state manager
	manager := state.NewStateManager(memStorage)
	defer manager.Close()

	ctx := context.Background()

	// 3. Create a mock Saga instance
	sagaInstance := createExampleSagaInstance("order-saga-001")

	// 4. Save the Saga
	fmt.Println("Saving Saga instance...")
	if err := manager.SaveSaga(ctx, sagaInstance); err != nil {
		log.Fatalf("Failed to save Saga: %v", err)
	}
	fmt.Printf("Saga %s saved successfully\n", sagaInstance.GetID())

	// 5. Update Saga state
	fmt.Println("\nUpdating Saga state to Running...")
	metadata := map[string]interface{}{
		"updated_by": "system",
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	if err := manager.UpdateSagaState(ctx, sagaInstance.GetID(), saga.StateRunning, metadata); err != nil {
		log.Fatalf("Failed to update state: %v", err)
	}
	fmt.Println("State updated successfully")

	// 6. Retrieve and verify
	fmt.Println("\nRetrieving Saga...")
	retrieved, err := manager.GetSaga(ctx, sagaInstance.GetID())
	if err != nil {
		log.Fatalf("Failed to retrieve Saga: %v", err)
	}

	fmt.Printf("Saga ID: %s\n", retrieved.GetID())
	fmt.Printf("State: %s\n", retrieved.GetState().String())
	fmt.Printf("Active: %t\n", retrieved.IsActive())

	// 7. Save step states
	fmt.Println("\nSaving step states...")
	for i := 1; i <= 3; i++ {
		step := createExampleStepState(sagaInstance.GetID(), i)
		if err := manager.SaveStepState(ctx, sagaInstance.GetID(), step); err != nil {
			log.Fatalf("Failed to save step state: %v", err)
		}
		fmt.Printf("Step %d saved\n", i)
	}

	// 8. Retrieve step states
	steps, err := manager.GetStepStates(ctx, sagaInstance.GetID())
	if err != nil {
		log.Fatalf("Failed to get step states: %v", err)
	}
	fmt.Printf("\nTotal steps: %d\n", len(steps))

	// 9. Complete the Saga
	fmt.Println("\nCompleting Saga...")
	if err := manager.UpdateSagaState(ctx, sagaInstance.GetID(), saga.StateCompleted, nil); err != nil {
		log.Fatalf("Failed to complete Saga: %v", err)
	}
	fmt.Println("Saga completed successfully")

	// 10. Verify final state
	final, _ := manager.GetSaga(ctx, sagaInstance.GetID())
	fmt.Printf("Final state: %s\n", final.GetState().String())
	fmt.Printf("Is terminal: %t\n", final.IsTerminal())
}

// StateTransitionExample demonstrates state transition validation.
func StateTransitionExample() {
	memStorage := storage.NewMemoryStateStorage()
	defer memStorage.Close()

	manager := state.NewStateManager(memStorage)
	defer manager.Close()

	ctx := context.Background()
	sagaInstance := createExampleSagaInstance("saga-transition-001")

	if err := manager.SaveSaga(ctx, sagaInstance); err != nil {
		log.Fatalf("Failed to save Saga: %v", err)
	}

	fmt.Println("Demonstrating state transitions...")

	// Valid transitions
	validTransitions := []struct {
		from saga.SagaState
		to   saga.SagaState
	}{
		{saga.StatePending, saga.StateRunning},
		{saga.StateRunning, saga.StateStepCompleted},
		{saga.StateStepCompleted, saga.StateCompleted},
	}

	for _, transition := range validTransitions {
		fmt.Printf("\nTransitioning %s -> %s...", transition.from, transition.to)
		err := manager.TransitionState(
			ctx,
			sagaInstance.GetID(),
			transition.from,
			transition.to,
			nil,
		)

		if err != nil {
			fmt.Printf(" FAILED: %v\n", err)
		} else {
			fmt.Printf(" SUCCESS\n")
		}
	}

	// Attempt invalid transition
	fmt.Printf("\nAttempting invalid transition (Completed -> Running)...")
	err := manager.TransitionState(
		ctx,
		sagaInstance.GetID(),
		saga.StateCompleted,
		saga.StateRunning,
		nil,
	)

	if err != nil {
		fmt.Printf(" FAILED (as expected): %v\n", err)
	} else {
		fmt.Printf(" UNEXPECTED SUCCESS\n")
	}
}

// EventListenerExample demonstrates state change event subscription.
func EventListenerExample() {
	memStorage := storage.NewMemoryStateStorage()
	defer memStorage.Close()

	manager := state.NewStateManager(memStorage)
	defer manager.Close()

	ctx := context.Background()

	// Subscribe to state changes
	fmt.Println("Setting up event listener...")
	eventCount := 0
	listener := func(event *state.StateChangeEvent) {
		eventCount++
		fmt.Printf("[Event %d] Saga %s: %s -> %s\n",
			eventCount,
			event.SagaID,
			event.OldState.String(),
			event.NewState.String())
	}

	if err := manager.SubscribeToStateChanges(listener); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// Create and update Saga
	sagaInstance := createExampleSagaInstance("saga-events-001")
	if err := manager.SaveSaga(ctx, sagaInstance); err != nil {
		log.Fatalf("Failed to save Saga: %v", err)
	}

	// Trigger state changes
	fmt.Println("\nTriggering state changes...")
	states := []saga.SagaState{
		saga.StateRunning,
		saga.StateStepCompleted,
		saga.StateCompleted,
	}

	for _, newState := range states {
		if err := manager.UpdateSagaState(ctx, sagaInstance.GetID(), newState, nil); err != nil {
			log.Printf("Failed to update state: %v", err)
		}
		time.Sleep(50 * time.Millisecond) // Wait for async event notification
	}

	fmt.Printf("\nTotal events received: %d\n", eventCount)
}

// BatchUpdateExample demonstrates batch update operations.
func BatchUpdateExample() {
	memStorage := storage.NewMemoryStateStorage()
	defer memStorage.Close()

	manager := state.NewStateManager(memStorage)
	defer manager.Close()

	ctx := context.Background()

	// Create multiple Sagas
	fmt.Println("Creating multiple Sagas...")
	numSagas := 5
	sagaIDs := make([]string, numSagas)

	for i := 0; i < numSagas; i++ {
		sagaID := fmt.Sprintf("saga-batch-%d", i)
		sagaIDs[i] = sagaID

		instance := createExampleSagaInstance(sagaID)
		if err := manager.SaveSaga(ctx, instance); err != nil {
			log.Fatalf("Failed to save Saga %d: %v", i, err)
		}

		if err := manager.UpdateSagaState(ctx, sagaID, saga.StateRunning, nil); err != nil {
			log.Fatalf("Failed to update Saga %d: %v", i, err)
		}
	}
	fmt.Printf("Created %d Sagas\n", numSagas)

	// Prepare batch update
	fmt.Println("\nPreparing batch update...")
	updates := make([]state.SagaUpdate, numSagas)
	for i, sagaID := range sagaIDs {
		updates[i] = state.SagaUpdate{
			SagaID:   sagaID,
			NewState: saga.StateCompleted,
			Metadata: map[string]interface{}{
				"batch_id": "batch-001",
				"index":    i,
			},
		}
	}

	// Execute batch update
	fmt.Println("Executing batch update...")
	if err := manager.BatchUpdateSagas(ctx, updates); err != nil {
		log.Fatalf("Batch update failed: %v", err)
	}
	fmt.Println("Batch update completed successfully")

	// Verify updates
	fmt.Println("\nVerifying updates...")
	for i, sagaID := range sagaIDs {
		instance, err := manager.GetSaga(ctx, sagaID)
		if err != nil {
			log.Printf("Failed to get Saga %d: %v", i, err)
			continue
		}
		fmt.Printf("Saga %s: %s\n", sagaID, instance.GetState().String())
	}
}

// CleanupExample demonstrates Saga cleanup operations.
func CleanupExample() {
	memStorage := storage.NewMemoryStateStorage()
	defer memStorage.Close()

	manager := state.NewStateManager(memStorage)
	defer manager.Close()

	ctx := context.Background()

	fmt.Println("Creating test Sagas...")

	// Create old completed Saga
	oldSaga := createExampleSagaInstance("saga-old")
	if err := memStorage.SaveSaga(ctx, oldSaga); err != nil {
		log.Fatalf("Failed to save old Saga: %v", err)
	}

	// Simulate old Saga by updating state
	if err := memStorage.UpdateSagaState(ctx, oldSaga.GetID(), saga.StateCompleted, nil); err != nil {
		log.Fatalf("Failed to update old Saga: %v", err)
	}

	// Create recent completed Saga
	recentSaga := createExampleSagaInstance("saga-recent")
	if err := manager.SaveSaga(ctx, recentSaga); err != nil {
		log.Fatalf("Failed to save recent Saga: %v", err)
	}

	if err := manager.UpdateSagaState(ctx, recentSaga.GetID(), saga.StateCompleted, nil); err != nil {
		log.Fatalf("Failed to update recent Saga: %v", err)
	}

	fmt.Println("Sagas created")

	// Cleanup old Sagas
	fmt.Println("\nCleaning up Sagas older than 1 hour...")
	cutoff := time.Now().Add(-1 * time.Hour)
	if err := manager.CleanupExpiredSagas(ctx, cutoff); err != nil {
		log.Fatalf("Cleanup failed: %v", err)
	}
	fmt.Println("Cleanup completed")

	// Verify cleanup
	fmt.Println("\nVerifying cleanup...")
	if _, err := manager.GetSaga(ctx, recentSaga.GetID()); err == nil {
		fmt.Println("Recent Saga still exists (correct)")
	} else {
		fmt.Printf("Recent Saga check failed: %v\n", err)
	}
}

// Helper functions

func createExampleSagaInstance(id string) *exampleSagaInstance {
	return &exampleSagaInstance{
		id:           id,
		definitionID: "example-definition",
		state:        saga.StatePending,
		currentStep:  0,
		totalSteps:   3,
		startTime:    time.Now(),
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
		timeout:      30 * time.Minute,
		metadata:     make(map[string]interface{}),
	}
}

func createExampleStepState(sagaID string, stepIndex int) *saga.StepState {
	now := time.Now()
	startTime := now.Add(-1 * time.Minute)

	return &saga.StepState{
		ID:          fmt.Sprintf("step-%d", stepIndex),
		SagaID:      sagaID,
		StepIndex:   stepIndex - 1,
		Name:        fmt.Sprintf("Step %d", stepIndex),
		State:       saga.StepStateCompleted,
		Attempts:    1,
		MaxAttempts: 3,
		CreatedAt:   startTime,
		StartedAt:   &startTime,
		CompletedAt: &now,
		OutputData:  map[string]interface{}{"result": fmt.Sprintf("step-%d-output", stepIndex)},
	}
}

// exampleSagaInstance is a simple implementation of saga.SagaInstance for examples.
type exampleSagaInstance struct {
	id           string
	definitionID string
	state        saga.SagaState
	currentStep  int
	totalSteps   int
	startTime    time.Time
	endTime      time.Time
	createdAt    time.Time
	updatedAt    time.Time
	timeout      time.Duration
	result       interface{}
	err          *saga.SagaError
	metadata     map[string]interface{}
}

func (e *exampleSagaInstance) GetID() string                       { return e.id }
func (e *exampleSagaInstance) GetDefinitionID() string             { return e.definitionID }
func (e *exampleSagaInstance) GetState() saga.SagaState            { return e.state }
func (e *exampleSagaInstance) GetCurrentStep() int                 { return e.currentStep }
func (e *exampleSagaInstance) GetTotalSteps() int                  { return e.totalSteps }
func (e *exampleSagaInstance) GetCompletedSteps() int              { return e.currentStep }
func (e *exampleSagaInstance) GetStartTime() time.Time             { return e.startTime }
func (e *exampleSagaInstance) GetEndTime() time.Time               { return e.endTime }
func (e *exampleSagaInstance) GetCreatedAt() time.Time             { return e.createdAt }
func (e *exampleSagaInstance) GetUpdatedAt() time.Time             { return e.updatedAt }
func (e *exampleSagaInstance) GetTimeout() time.Duration           { return e.timeout }
func (e *exampleSagaInstance) GetResult() interface{}              { return e.result }
func (e *exampleSagaInstance) GetError() *saga.SagaError           { return e.err }
func (e *exampleSagaInstance) GetMetadata() map[string]interface{} { return e.metadata }
func (e *exampleSagaInstance) GetTraceID() string                  { return "" }
func (e *exampleSagaInstance) IsTerminal() bool                    { return e.state.IsTerminal() }
func (e *exampleSagaInstance) IsActive() bool                      { return e.state.IsActive() }
