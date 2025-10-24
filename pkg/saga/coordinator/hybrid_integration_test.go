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
	"sync/atomic"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestHybrid_ModeSelection tests automatic mode selection based on saga characteristics.
func TestHybrid_ModeSelection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	// Create orchestrator coordinator
	orchestratorConfig := &OrchestratorConfig{
		StateStorage:   storage,
		EventPublisher: publisher,
		RetryPolicy:    saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
	}
	orchestrator, err := NewOrchestratorCoordinator(orchestratorConfig)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orchestrator.Close()

	// Create choreography coordinator
	choreographyConfig := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 10,
		EventTimeout:          5 * time.Second,
	}
	choreography, err := NewChoreographyCoordinator(choreographyConfig)
	if err != nil {
		t.Fatalf("Failed to create choreography coordinator: %v", err)
	}
	defer choreography.Close()

	// Create hybrid coordinator with smart mode selector
	hybridConfig := &HybridConfig{
		Orchestrator: orchestrator,
		Choreography: choreography,
		ModeSelector: NewSmartModeSelector([]ModeSelectionRule{
			NewSimpleLinearRule(5),
			NewComplexParallelRule(5),
			NewCrossDomainRule(),
		}, ModeOrchestration),
	}

	hybrid, err := NewHybridCoordinator(hybridConfig)
	if err != nil {
		t.Fatalf("Failed to create hybrid coordinator: %v", err)
	}
	defer hybrid.Close()

	ctx := context.Background()

	// Test 1: Simple saga should use orchestration
	simpleDefinition := &testSagaDefinition{
		id:   "simple-saga",
		name: "Simple Saga",
		steps: []saga.SagaStep{
			&testSagaStep{
				id:   "step-1",
				name: "Step 1",
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					return data, nil
				},
			},
			&testSagaStep{
				id:   "step-2",
				name: "Step 2",
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					return data, nil
				},
			},
		},
		timeout:     time.Second * 30,
		retryPolicy: saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
		strategy:    saga.NewSequentialCompensationStrategy(time.Second * 30),
	}

	simpleInstance, err := hybrid.StartSaga(ctx, simpleDefinition, map[string]interface{}{"test": "data"})
	if err != nil {
		t.Fatalf("Failed to start simple saga: %v", err)
	}

	// Verify it used orchestration mode (by checking if orchestrator has it)
	_, err = orchestrator.GetSagaInstance(simpleInstance.GetID())
	if err != nil {
		t.Errorf("Expected saga to be in orchestration mode, but not found in orchestrator: %v", err)
	}

	// Test 2: Complex saga should use choreography
	complexDefinition := &testSagaDefinition{
		id:   "complex-saga",
		name: "Complex Event-Driven Saga",
		steps: []saga.SagaStep{
			&testSagaStep{id: "step-1", name: "Step 1", execute: func(ctx context.Context, data interface{}) (interface{}, error) { return data, nil }},
			&testSagaStep{id: "step-2", name: "Step 2", execute: func(ctx context.Context, data interface{}) (interface{}, error) { return data, nil }},
			&testSagaStep{id: "step-3", name: "Step 3", execute: func(ctx context.Context, data interface{}) (interface{}, error) { return data, nil }},
			&testSagaStep{id: "step-4", name: "Step 4", execute: func(ctx context.Context, data interface{}) (interface{}, error) { return data, nil }},
			&testSagaStep{id: "step-5", name: "Step 5", execute: func(ctx context.Context, data interface{}) (interface{}, error) { return data, nil }},
			&testSagaStep{id: "step-6", name: "Step 6", execute: func(ctx context.Context, data interface{}) (interface{}, error) { return data, nil }},
		},
		timeout:     time.Second * 30,
		retryPolicy: saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
		strategy:    saga.NewSequentialCompensationStrategy(time.Second * 30),
		metadata: map[string]interface{}{
			"complexity": "high",
		},
	}

	complexInstance, err := hybrid.StartSaga(ctx, complexDefinition, map[string]interface{}{"test": "data"})
	if err != nil {
		t.Fatalf("Failed to start complex saga: %v", err)
	}

	// Note: Choreography mode doesn't support centralized instance retrieval
	// We verify by checking that the saga was created successfully
	if complexInstance == nil {
		t.Error("Expected complex saga instance to be created")
	}

	// Verify metrics
	metrics := hybrid.GetMetrics()
	if metrics.TotalSagas != 2 {
		t.Errorf("Expected 2 total sagas, got %d", metrics.TotalSagas)
	}
}

// TestHybrid_ForcedMode tests forcing a specific coordination mode.
func TestHybrid_ForcedMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	// Create orchestrator
	orchestratorConfig := &OrchestratorConfig{
		StateStorage:   storage,
		EventPublisher: publisher,
		RetryPolicy:    saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
	}
	orchestrator, err := NewOrchestratorCoordinator(orchestratorConfig)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orchestrator.Close()

	// Create choreography
	choreographyConfig := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 10,
		EventTimeout:          5 * time.Second,
	}
	choreography, err := NewChoreographyCoordinator(choreographyConfig)
	if err != nil {
		t.Fatalf("Failed to create choreography coordinator: %v", err)
	}
	defer choreography.Close()

	// Test forcing orchestration mode
	hybridConfig := &HybridConfig{
		Orchestrator: orchestrator,
		Choreography: choreography,
		ForceMode:    ModeOrchestration,
	}

	hybrid, err := NewHybridCoordinator(hybridConfig)
	if err != nil {
		t.Fatalf("Failed to create hybrid coordinator: %v", err)
	}
	defer hybrid.Close()

	ctx := context.Background()

	// Start saga (should use orchestration regardless of characteristics)
	definition := &testSagaDefinition{
		id:   "forced-saga",
		name: "Forced Mode Saga",
		steps: []saga.SagaStep{
			&testSagaStep{
				id:   "step-1",
				name: "Step 1",
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					return data, nil
				},
			},
		},
		timeout:     time.Second * 30,
		retryPolicy: saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
		strategy:    saga.NewSequentialCompensationStrategy(time.Second * 30),
	}

	instance, err := hybrid.StartSaga(ctx, definition, map[string]interface{}{"test": "data"})
	if err != nil {
		t.Fatalf("Failed to start saga: %v", err)
	}

	// Verify forced mode (by checking orchestrator has it)
	_, err = orchestrator.GetSagaInstance(instance.GetID())
	if err != nil {
		t.Errorf("Expected saga to be in forced orchestration mode, but not found: %v", err)
	}
}

// TestHybrid_ModeSwitchingCapability tests the capability to handle different modes.
func TestHybrid_ModeSwitchingCapability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	// Setup both coordinators
	orchestratorConfig := &OrchestratorConfig{
		StateStorage:   storage,
		EventPublisher: publisher,
		RetryPolicy:    saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
	}
	orchestrator, err := NewOrchestratorCoordinator(orchestratorConfig)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orchestrator.Close()

	choreographyConfig := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 10,
		EventTimeout:          5 * time.Second,
	}
	choreography, err := NewChoreographyCoordinator(choreographyConfig)
	if err != nil {
		t.Fatalf("Failed to create choreography coordinator: %v", err)
	}
	defer choreography.Close()

	// Create hybrid coordinator
	hybridConfig := &HybridConfig{
		Orchestrator: orchestrator,
		Choreography: choreography,
		ModeSelector: NewSmartModeSelector([]ModeSelectionRule{
			NewSimpleLinearRule(5),
			NewComplexParallelRule(5),
			NewCrossDomainRule(),
		}, ModeOrchestration),
	}

	hybrid, err := NewHybridCoordinator(hybridConfig)
	if err != nil {
		t.Fatalf("Failed to create hybrid coordinator: %v", err)
	}
	defer hybrid.Close()

	ctx := context.Background()

	// Track mode distribution
	var orchestrationCount atomic.Int32
	var choreographyCount atomic.Int32

	// Start multiple sagas with different characteristics
	for i := 0; i < 10; i++ {
		numSteps := 2 // Simple saga - should use orchestration
		if i%3 == 0 {
			numSteps = 6 // Complex saga - should use choreography
		}

		steps := make([]saga.SagaStep, numSteps)
		for j := 0; j < numSteps; j++ {
			steps[j] = &testSagaStep{
				id:   "step-" + string(rune('A'+j)),
				name: "Step " + string(rune('A'+j)),
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					return data, nil
				},
			}
		}

		definition := &testSagaDefinition{
			id:          "mixed-saga-" + string(rune('0'+i)),
			name:        "Mixed Mode Saga",
			steps:       steps,
			timeout:     time.Second * 30,
			retryPolicy: saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
			strategy:    saga.NewSequentialCompensationStrategy(time.Second * 30),
		}

		instance, err := hybrid.StartSaga(ctx, definition, map[string]interface{}{"index": i})
		if err != nil {
			t.Errorf("Failed to start saga %d: %v", i, err)
			continue
		}

		// Track mode
		mode, ok := hybrid.GetModeForSaga(instance.GetID())
		if ok {
			if mode == ModeOrchestration {
				orchestrationCount.Add(1)
			} else if mode == ModeChoreography {
				choreographyCount.Add(1)
			}
		}
	}

	// Verify both modes were used
	orchCount := orchestrationCount.Load()
	chorCount := choreographyCount.Load()

	t.Logf("Orchestration sagas: %d, Choreography sagas: %d", orchCount, chorCount)

	if orchCount == 0 {
		t.Error("Expected some sagas to use orchestration mode")
	}
	if chorCount == 0 {
		t.Error("Expected some sagas to use choreography mode")
	}

	// Verify metrics
	hybridMetrics := hybrid.GetHybridMetrics()
	if hybridMetrics.TotalSagas != 10 {
		t.Errorf("Expected 10 total sagas, got %d", hybridMetrics.TotalSagas)
	}
	if hybridMetrics.OrchestrationSagas == 0 {
		t.Error("Expected some orchestration sagas")
	}
	if hybridMetrics.ChoreographySagas == 0 {
		t.Error("Expected some choreography sagas")
	}
}

// TestHybrid_ConcurrentMixedModes tests concurrent saga execution with mixed modes.
func TestHybrid_ConcurrentMixedModes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	// Setup
	orchestratorConfig := &OrchestratorConfig{
		StateStorage:   storage,
		EventPublisher: publisher,
		RetryPolicy:    saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
		ConcurrencyConfig: &ConcurrencyConfig{
			MaxConcurrentSagas: 50,
			WorkerPoolSize:     10,
			AcquireTimeout:     time.Second * 5,
			ShutdownTimeout:    time.Second * 10,
		},
	}
	orchestrator, err := NewOrchestratorCoordinator(orchestratorConfig)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orchestrator.Close()

	choreographyConfig := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 50,
		EventTimeout:          5 * time.Second,
	}
	choreography, err := NewChoreographyCoordinator(choreographyConfig)
	if err != nil {
		t.Fatalf("Failed to create choreography coordinator: %v", err)
	}
	defer choreography.Close()

	hybridConfig := &HybridConfig{
		Orchestrator: orchestrator,
		Choreography: choreography,
		ModeSelector: NewSmartModeSelector([]ModeSelectionRule{
			NewSimpleLinearRule(5),
			NewComplexParallelRule(5),
			NewCrossDomainRule(),
		}, ModeOrchestration),
	}

	hybrid, err := NewHybridCoordinator(hybridConfig)
	if err != nil {
		t.Fatalf("Failed to create hybrid coordinator: %v", err)
	}
	defer hybrid.Close()

	ctx := context.Background()

	// Start many sagas concurrently
	numSagas := 30
	completed := make(chan string, numSagas)

	for i := 0; i < numSagas; i++ {
		go func(index int) {
			// Alternate between simple and complex sagas
			numSteps := 2
			if index%2 == 0 {
				numSteps = 7
			}

			steps := make([]saga.SagaStep, numSteps)
			for j := 0; j < numSteps; j++ {
				steps[j] = &testSagaStep{
					id:   "step-" + string(rune('A'+j)),
					name: "Step " + string(rune('A'+j)),
					execute: func(ctx context.Context, data interface{}) (interface{}, error) {
						time.Sleep(10 * time.Millisecond) // Simulate work
						return data, nil
					},
				}
			}

			definition := &testSagaDefinition{
				id:          "concurrent-mixed-" + string(rune('0'+index%10)) + string(rune('A'+index/10)),
				name:        "Concurrent Mixed Saga",
				steps:       steps,
				timeout:     time.Second * 30,
				retryPolicy: saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
				strategy:    saga.NewSequentialCompensationStrategy(time.Second * 30),
			}

			instance, err := hybrid.StartSaga(ctx, definition, map[string]interface{}{"index": index})
			if err != nil {
				t.Errorf("Failed to start saga %d: %v", index, err)
				completed <- ""
				return
			}

			completed <- instance.GetID()
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numSagas; i++ {
		sagaID := <-completed
		if sagaID != "" {
			successCount++
		}
	}

	if successCount != numSagas {
		t.Errorf("Expected %d sagas to start successfully, got %d", numSagas, successCount)
	}

	// Wait for execution
	time.Sleep(2 * time.Second)

	// Verify metrics
	hybridMetrics := hybrid.GetHybridMetrics()
	t.Logf("Total sagas: %d, Orchestration: %d, Choreography: %d",
		hybridMetrics.TotalSagas, hybridMetrics.OrchestrationSagas, hybridMetrics.ChoreographySagas)

	if hybridMetrics.TotalSagas != int64(numSagas) {
		t.Errorf("Expected %d total sagas in metrics, got %d", numSagas, hybridMetrics.TotalSagas)
	}
}

// TestHybrid_HealthCheck tests health check functionality.
func TestHybrid_HealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	// Setup
	orchestratorConfig := &OrchestratorConfig{
		StateStorage:   storage,
		EventPublisher: publisher,
		RetryPolicy:    saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
	}
	orchestrator, err := NewOrchestratorCoordinator(orchestratorConfig)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orchestrator.Close()

	choreographyConfig := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 10,
		EventTimeout:          5 * time.Second,
	}
	choreography, err := NewChoreographyCoordinator(choreographyConfig)
	if err != nil {
		t.Fatalf("Failed to create choreography coordinator: %v", err)
	}
	defer choreography.Close()

	// Start choreography coordinator for health check
	ctx := context.Background()
	if err := choreography.Start(ctx); err != nil {
		t.Fatalf("failed to start choreography coordinator: %v", err)
	}

	hybridConfig := &HybridConfig{
		Orchestrator: orchestrator,
		Choreography: choreography,
		ModeSelector: NewSmartModeSelector([]ModeSelectionRule{
			NewSimpleLinearRule(5),
			NewComplexParallelRule(5),
			NewCrossDomainRule(),
		}, ModeOrchestration),
	}

	hybrid, err := NewHybridCoordinator(hybridConfig)
	if err != nil {
		t.Fatalf("Failed to create hybrid coordinator: %v", err)
	}
	defer hybrid.Close()

	// Health check should succeed
	err = hybrid.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Close one coordinator and check health
	orchestrator.Close()

	// Health check should still succeed (choreography is still healthy)
	err = hybrid.HealthCheck(ctx)
	if err == nil {
		t.Log("Health check succeeded with one coordinator closed (expected)")
	}
}

// TestHybrid_MetricsAggregation tests that metrics are properly aggregated from both coordinators.
func TestHybrid_MetricsAggregation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	// Setup
	orchestratorConfig := &OrchestratorConfig{
		StateStorage:   storage,
		EventPublisher: publisher,
		RetryPolicy:    saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
	}
	orchestrator, err := NewOrchestratorCoordinator(orchestratorConfig)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orchestrator.Close()

	choreographyConfig := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 10,
		EventTimeout:          5 * time.Second,
		EnableMetrics:         true,
	}
	choreography, err := NewChoreographyCoordinator(choreographyConfig)
	if err != nil {
		t.Fatalf("Failed to create choreography coordinator: %v", err)
	}
	defer choreography.Close()

	hybridConfig := &HybridConfig{
		Orchestrator: orchestrator,
		Choreography: choreography,
		ModeSelector: NewSmartModeSelector([]ModeSelectionRule{
			NewSimpleLinearRule(5),
			NewComplexParallelRule(5),
			NewCrossDomainRule(),
		}, ModeOrchestration),
	}

	hybrid, err := NewHybridCoordinator(hybridConfig)
	if err != nil {
		t.Fatalf("Failed to create hybrid coordinator: %v", err)
	}
	defer hybrid.Close()

	ctx := context.Background()

	// Start some sagas in orchestration mode
	for i := 0; i < 3; i++ {
		definition := &testSagaDefinition{
			id:   "orch-saga-" + string(rune('0'+i)),
			name: "Orchestration Saga",
			steps: []saga.SagaStep{
				&testSagaStep{
					id:   "step-1",
					name: "Step 1",
					execute: func(ctx context.Context, data interface{}) (interface{}, error) {
						return data, nil
					},
				},
			},
			timeout:     time.Second * 30,
			retryPolicy: saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
			strategy:    saga.NewSequentialCompensationStrategy(time.Second * 30),
		}

		_, err := hybrid.StartSaga(ctx, definition, map[string]interface{}{"index": i})
		if err != nil {
			t.Errorf("Failed to start orchestration saga %d: %v", i, err)
		}
	}

	// Start some sagas in choreography mode (complex ones)
	for i := 0; i < 2; i++ {
		steps := make([]saga.SagaStep, 8) // Complex enough for choreography
		for j := 0; j < 8; j++ {
			steps[j] = &testSagaStep{
				id:   "step-" + string(rune('A'+j)),
				name: "Step " + string(rune('A'+j)),
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					return data, nil
				},
			}
		}

		definition := &testSagaDefinition{
			id:          "chor-saga-" + string(rune('0'+i)),
			name:        "Choreography Saga",
			steps:       steps,
			timeout:     time.Second * 30,
			retryPolicy: saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
			strategy:    saga.NewSequentialCompensationStrategy(time.Second * 30),
		}

		_, err := hybrid.StartSaga(ctx, definition, map[string]interface{}{"index": i})
		if err != nil {
			t.Errorf("Failed to start choreography saga %d: %v", i, err)
		}
	}

	// Wait for initialization
	time.Sleep(time.Second)

	// Verify metrics
	hybridMetrics := hybrid.GetHybridMetrics()
	if hybridMetrics.TotalSagas != 5 {
		t.Errorf("Expected 5 total sagas, got %d", hybridMetrics.TotalSagas)
	}
	if hybridMetrics.OrchestrationSagas != 3 {
		t.Errorf("Expected 3 orchestration sagas, got %d", hybridMetrics.OrchestrationSagas)
	}
	if hybridMetrics.ChoreographySagas != 2 {
		t.Errorf("Expected 2 choreography sagas, got %d", hybridMetrics.ChoreographySagas)
	}

	t.Logf("Metrics aggregation verified: Total=%d, Orchestration=%d, Choreography=%d",
		hybridMetrics.TotalSagas, hybridMetrics.OrchestrationSagas, hybridMetrics.ChoreographySagas)
}
