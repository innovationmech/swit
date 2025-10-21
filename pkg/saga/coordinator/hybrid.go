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

// Package coordinator provides hybrid Saga coordination combining orchestration and choreography.
package coordinator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

var (
	// ErrHybridCoordinatorClosed indicates the hybrid coordinator has been closed.
	ErrHybridCoordinatorClosed = errors.New("hybrid coordinator is closed")

	// ErrOrchestratorNotConfigured indicates orchestrator coordinator is not configured.
	ErrOrchestratorNotConfigured = errors.New("orchestrator coordinator not configured")

	// ErrChoreographyNotConfigured indicates choreography coordinator is not configured.
	ErrChoreographyNotConfigured = errors.New("choreography coordinator not configured")

	// ErrModeSelectorNotConfigured indicates mode selector is not configured.
	ErrModeSelectorNotConfigured = errors.New("mode selector not configured")

	// ErrUnsupportedMode indicates an unsupported coordination mode was requested.
	ErrUnsupportedMode = errors.New("unsupported coordination mode")
)

// HybridCoordinator implements hybrid Saga coordination that can dynamically
// choose between orchestration and choreography modes based on Saga characteristics.
//
// The coordinator follows these key responsibilities:
//  1. Intelligent mode selection based on Saga definition characteristics
//  2. Delegation to appropriate underlying coordinator (orchestrator or choreography)
//  3. Unified interface for both coordination modes
//  4. Mode tracking and metrics collection
//  5. Graceful shutdown and cleanup
type HybridCoordinator struct {
	// orchestrator is the orchestration-based coordinator.
	orchestrator *OrchestratorCoordinator

	// choreography is the choreography-based coordinator.
	choreography *ChoreographyCoordinator

	// modeSelector selects the coordination mode for each Saga.
	modeSelector ModeSelector

	// config holds the coordinator configuration.
	config *HybridConfig

	// modeTracking tracks which mode was used for each Saga.
	modeTracking map[string]CoordinationMode

	// metrics collects runtime metrics.
	metrics *HybridMetrics

	// closed indicates if the coordinator has been shut down.
	closed bool

	// mu protects concurrent access to coordinator state.
	mu sync.RWMutex
}

// HybridConfig contains configuration options for the hybrid coordinator.
type HybridConfig struct {
	// Orchestrator is the orchestration-based coordinator.
	// Required if ForceMode is not set to ModeChoreography.
	Orchestrator *OrchestratorCoordinator

	// Choreography is the choreography-based coordinator.
	// Required if ForceMode is not set to ModeOrchestration.
	Choreography *ChoreographyCoordinator

	// ModeSelector selects the coordination mode for each Saga.
	// If not provided, a default smart selector is used.
	ModeSelector ModeSelector

	// ForceMode forces a specific coordination mode for all Sagas.
	// If set, mode selection is bypassed.
	ForceMode CoordinationMode

	// EnableMetrics enables metrics collection.
	// Default is true.
	EnableMetrics bool
}

// Validate checks if the configuration is valid.
func (c *HybridConfig) Validate() error {
	// Validate based on forced mode or general requirements
	switch c.ForceMode {
	case ModeOrchestration:
		if c.Orchestrator == nil {
			return ErrOrchestratorNotConfigured
		}
	case ModeChoreography:
		if c.Choreography == nil {
			return ErrChoreographyNotConfigured
		}
	case ModeAuto, "":
		// Auto mode requires both coordinators
		if c.Orchestrator == nil {
			return ErrOrchestratorNotConfigured
		}
		if c.Choreography == nil {
			return ErrChoreographyNotConfigured
		}
		if c.ModeSelector == nil {
			return ErrModeSelectorNotConfigured
		}
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedMode, c.ForceMode)
	}

	return nil
}

// HybridMetrics contains metrics about hybrid coordination.
type HybridMetrics struct {
	// TotalSagas is the total number of Sagas started.
	TotalSagas int64 `json:"total_sagas"`

	// OrchestrationSagas is the number of Sagas using orchestration mode.
	OrchestrationSagas int64 `json:"orchestration_sagas"`

	// ChoreographySagas is the number of Sagas using choreography mode.
	ChoreographySagas int64 `json:"choreography_sagas"`

	// ActiveSagas is the current number of active Sagas.
	ActiveSagas int64 `json:"active_sagas"`

	// CompletedSagas is the number of completed Sagas.
	CompletedSagas int64 `json:"completed_sagas"`

	// FailedSagas is the number of failed Sagas.
	FailedSagas int64 `json:"failed_sagas"`

	mu sync.RWMutex
}

// NewHybridCoordinator creates a new hybrid Saga coordinator.
// It initializes both orchestration and choreography coordinators and prepares
// the hybrid coordinator for managing Sagas with dynamic mode selection.
//
// Parameters:
//   - config: Configuration containing required coordinators and optional mode selector.
//
// Returns:
//   - A configured HybridCoordinator instance ready to coordinate Sagas.
//   - An error if the configuration is invalid or initialization fails.
//
// Example:
//
//	// Create orchestrator coordinator
//	orchestrator, err := NewOrchestratorCoordinator(&OrchestratorConfig{...})
//	if err != nil {
//	    return err
//	}
//
//	// Create choreography coordinator
//	choreography, err := NewChoreographyCoordinator(&ChoreographyConfig{...})
//	if err != nil {
//	    return err
//	}
//
//	// Create hybrid coordinator with smart mode selection
//	hybrid, err := NewHybridCoordinator(&HybridConfig{
//	    Orchestrator: orchestrator,
//	    Choreography: choreography,
//	    ModeSelector: NewSmartModeSelector([]ModeSelectionRule{
//	        NewSimpleLinearRule(5),
//	        NewComplexParallelRule(5),
//	        NewCrossDomainRule(),
//	    }, ModeOrchestration),
//	})
//	if err != nil {
//	    return err
//	}
//	defer hybrid.Close()
func NewHybridCoordinator(config *HybridConfig) (*HybridCoordinator, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Use default smart selector if not provided and not in forced mode
	modeSelector := config.ModeSelector
	if modeSelector == nil && config.ForceMode == "" {
		// Create default smart selector with common rules
		modeSelector = NewSmartModeSelector([]ModeSelectionRule{
			NewCentralizedControlRule(),
			NewCrossDomainRule(),
			NewSimpleLinearRule(5),
			NewComplexParallelRule(5),
		}, ModeOrchestration)
	}

	coordinator := &HybridCoordinator{
		orchestrator: config.Orchestrator,
		choreography: config.Choreography,
		modeSelector: modeSelector,
		config:       config,
		modeTracking: make(map[string]CoordinationMode),
		metrics: &HybridMetrics{
			TotalSagas:         0,
			OrchestrationSagas: 0,
			ChoreographySagas:  0,
			ActiveSagas:        0,
			CompletedSagas:     0,
			FailedSagas:        0,
		},
		closed: false,
	}

	return coordinator, nil
}

// StartSaga starts a new Saga instance with automatic mode selection.
// The coordinator selects the appropriate mode (orchestration or choreography)
// based on the Saga definition characteristics and delegates to the corresponding
// coordinator.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - definition: The Saga definition containing steps and configuration.
//   - initialData: Initial data to pass to the first step.
//
// Returns:
//   - The created Saga instance if successful.
//   - An error if validation fails, mode selection fails, or the coordinator is closed.
func (hc *HybridCoordinator) StartSaga(
	ctx context.Context,
	definition saga.SagaDefinition,
	initialData interface{},
) (saga.SagaInstance, error) {
	hc.mu.RLock()
	if hc.closed {
		hc.mu.RUnlock()
		return nil, ErrHybridCoordinatorClosed
	}
	hc.mu.RUnlock()

	// Validate definition
	if definition == nil {
		return nil, ErrInvalidDefinition
	}

	if err := definition.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Saga definition: %w", err)
	}

	// Select coordination mode
	mode := hc.selectMode(definition)

	// Delegate to appropriate coordinator
	var instance saga.SagaInstance
	var err error

	switch mode {
	case ModeOrchestration:
		if hc.orchestrator == nil {
			return nil, ErrOrchestratorNotConfigured
		}
		instance, err = hc.orchestrator.StartSaga(ctx, definition, initialData)

	case ModeChoreography:
		if hc.choreography == nil {
			return nil, ErrChoreographyNotConfigured
		}
		// Generate a unique saga ID for this choreography instance
		sagaID, err := generateSagaID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate saga ID: %w", err)
		}
		// For choreography mode, we publish an initial event to start the Saga
		// The actual coordination is handled by event handlers
		event := &saga.SagaEvent{
			ID:        generateEventID(),
			SagaID:    sagaID,
			Type:      saga.EventSagaStarted,
			Timestamp: hc.now(),
			Data:      initialData,
			Source:    "HybridCoordinator",
		}
		err = hc.choreography.PublishEvent(ctx, event)
		if err != nil {
			return nil, fmt.Errorf("failed to start choreography Saga: %w", err)
		}
		// Create a simple instance wrapper for choreography mode
		instance = &choreographySagaInstance{
			sagaID: event.SagaID,
			def:    definition,
		}

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedMode, mode)
	}

	if err != nil {
		// Update failure metrics
		hc.metrics.mu.Lock()
		hc.metrics.FailedSagas++
		hc.metrics.mu.Unlock()
		return nil, err
	}

	// Track the mode used for this Saga
	hc.mu.Lock()
	hc.modeTracking[instance.GetID()] = mode
	hc.mu.Unlock()

	// Update metrics
	hc.metrics.mu.Lock()
	hc.metrics.TotalSagas++
	hc.metrics.ActiveSagas++
	if mode == ModeOrchestration {
		hc.metrics.OrchestrationSagas++
	} else {
		hc.metrics.ChoreographySagas++
	}
	hc.metrics.mu.Unlock()

	return instance, nil
}

// selectMode selects the coordination mode for a Saga definition.
func (hc *HybridCoordinator) selectMode(definition saga.SagaDefinition) CoordinationMode {
	// Check for forced mode
	if hc.config.ForceMode != "" && hc.config.ForceMode != ModeAuto {
		return hc.config.ForceMode
	}

	// Check definition metadata for explicit mode
	metadata := definition.GetMetadata()
	if metadata != nil {
		if modeStr, ok := metadata["coordination_mode"].(string); ok {
			mode := CoordinationMode(modeStr)
			if mode == ModeOrchestration || mode == ModeChoreography {
				return mode
			}
		}
	}

	// Use mode selector
	if hc.modeSelector != nil {
		return hc.modeSelector.SelectMode(definition)
	}

	// Default to orchestration
	return ModeOrchestration
}

// GetSagaInstance retrieves the current state of a Saga instance by its ID.
// It queries the appropriate coordinator based on the tracked mode.
//
// Parameters:
//   - sagaID: The unique identifier of the Saga instance.
//
// Returns:
//   - The Saga instance if found.
//   - An error if the instance does not exist or the coordinator is closed.
func (hc *HybridCoordinator) GetSagaInstance(sagaID string) (saga.SagaInstance, error) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if hc.closed {
		return nil, ErrHybridCoordinatorClosed
	}

	// Get the mode used for this Saga
	mode, exists := hc.modeTracking[sagaID]
	if !exists {
		return nil, saga.NewSagaNotFoundError(sagaID)
	}

	// Query appropriate coordinator
	switch mode {
	case ModeOrchestration:
		if hc.orchestrator == nil {
			return nil, ErrOrchestratorNotConfigured
		}
		return hc.orchestrator.GetSagaInstance(sagaID)

	case ModeChoreography:
		// Choreography mode doesn't maintain centralized instance state
		// Return a minimal instance representation
		return nil, errors.New("choreography mode does not support centralized instance retrieval")

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedMode, mode)
	}
}

// CancelSaga cancels a running Saga instance with the specified reason.
// This triggers compensation operations for all completed steps.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - sagaID: The unique identifier of the Saga instance to cancel.
//   - reason: Human-readable reason for cancellation.
//
// Returns:
//   - An error if the Saga is not found, already terminal, or cancellation fails.
func (hc *HybridCoordinator) CancelSaga(ctx context.Context, sagaID string, reason string) error {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if hc.closed {
		return ErrHybridCoordinatorClosed
	}

	// Get the mode used for this Saga
	mode, exists := hc.modeTracking[sagaID]
	if !exists {
		return saga.NewSagaNotFoundError(sagaID)
	}

	// Delegate to appropriate coordinator
	switch mode {
	case ModeOrchestration:
		if hc.orchestrator == nil {
			return ErrOrchestratorNotConfigured
		}
		return hc.orchestrator.CancelSaga(ctx, sagaID, reason)

	case ModeChoreography:
		// For choreography mode, publish a cancellation event
		if hc.choreography == nil {
			return ErrChoreographyNotConfigured
		}
		event := &saga.SagaEvent{
			ID:        generateEventID(),
			SagaID:    sagaID,
			Type:      saga.EventSagaCancelled,
			Timestamp: hc.now(),
			Data:      reason,
			Source:    "HybridCoordinator",
		}
		return hc.choreography.PublishEvent(ctx, event)

	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedMode, mode)
	}
}

// GetActiveSagas retrieves all currently active Saga instances based on the filter.
// Active Sagas are those in Running, StepCompleted, or Compensating states.
//
// Parameters:
//   - filter: Optional filter to narrow results. Pass nil to get all active Sagas.
//
// Returns:
//   - A slice of active Saga instances.
//   - An error if the query fails or the coordinator is closed.
func (hc *HybridCoordinator) GetActiveSagas(filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if hc.closed {
		return nil, ErrHybridCoordinatorClosed
	}

	var allInstances []saga.SagaInstance

	// Get active Sagas from orchestrator
	if hc.orchestrator != nil {
		instances, err := hc.orchestrator.GetActiveSagas(filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get active Sagas from orchestrator: %w", err)
		}
		allInstances = append(allInstances, instances...)
	}

	// Choreography mode doesn't maintain centralized state
	// Active Sagas are tracked through events

	return allInstances, nil
}

// GetMetrics returns runtime metrics about the coordinator's performance.
// The metrics include mode selection statistics and delegation information.
//
// Returns:
//   - A snapshot of current coordinator metrics.
func (hc *HybridCoordinator) GetMetrics() *saga.CoordinatorMetrics {
	hc.metrics.mu.RLock()
	defer hc.metrics.mu.RUnlock()

	// Create a unified metrics view
	metrics := &saga.CoordinatorMetrics{
		TotalSagas:     hc.metrics.TotalSagas,
		ActiveSagas:    hc.metrics.ActiveSagas,
		CompletedSagas: hc.metrics.CompletedSagas,
		FailedSagas:    hc.metrics.FailedSagas,
	}

	return metrics
}

// GetHybridMetrics returns hybrid-specific metrics.
func (hc *HybridCoordinator) GetHybridMetrics() *HybridMetrics {
	hc.metrics.mu.RLock()
	defer hc.metrics.mu.RUnlock()

	// Return a copy to prevent external mutation
	return &HybridMetrics{
		TotalSagas:         hc.metrics.TotalSagas,
		OrchestrationSagas: hc.metrics.OrchestrationSagas,
		ChoreographySagas:  hc.metrics.ChoreographySagas,
		ActiveSagas:        hc.metrics.ActiveSagas,
		CompletedSagas:     hc.metrics.CompletedSagas,
		FailedSagas:        hc.metrics.FailedSagas,
	}
}

// GetModeForSaga returns the coordination mode used for a specific Saga.
func (hc *HybridCoordinator) GetModeForSaga(sagaID string) (CoordinationMode, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	mode, exists := hc.modeTracking[sagaID]
	return mode, exists
}

// HealthCheck performs a health check of the coordinator and its dependencies.
// It verifies that both underlying coordinators are operational.
//
// Parameters:
//   - ctx: Context for timeout control.
//
// Returns:
//   - nil if all components are healthy.
//   - An error describing which component is unhealthy.
func (hc *HybridCoordinator) HealthCheck(ctx context.Context) error {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if hc.closed {
		return ErrHybridCoordinatorClosed
	}

	// Check orchestrator health if configured
	if hc.orchestrator != nil {
		if err := hc.orchestrator.HealthCheck(ctx); err != nil {
			return fmt.Errorf("orchestrator health check failed: %w", err)
		}
	}

	// Choreography coordinator doesn't have a health check method in the current implementation
	// but we can check if it's running
	if hc.choreography != nil && !hc.choreography.IsRunning() {
		return errors.New("choreography coordinator is not running")
	}

	return nil
}

// Close gracefully shuts down the coordinator, releasing all resources.
// It closes both underlying coordinators and cleans up internal state.
//
// Returns:
//   - An error if shutdown encounters issues.
func (hc *HybridCoordinator) Close() error {
	hc.mu.Lock()
	if hc.closed {
		hc.mu.Unlock()
		return ErrHybridCoordinatorClosed
	}
	hc.closed = true
	hc.mu.Unlock()

	var errs []error

	// Close orchestrator
	if hc.orchestrator != nil {
		if err := hc.orchestrator.Close(); err != nil {
			errs = append(errs, fmt.Errorf("orchestrator close error: %w", err))
		}
	}

	// Close choreography coordinator
	if hc.choreography != nil {
		if err := hc.choreography.Close(); err != nil {
			errs = append(errs, fmt.Errorf("choreography close error: %w", err))
		}
	}

	// Clear tracking
	hc.mu.Lock()
	hc.modeTracking = make(map[string]CoordinationMode)
	hc.mu.Unlock()

	// Return combined errors if any
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}

// now returns the current time (mockable for testing).
func (hc *HybridCoordinator) now() time.Time {
	return time.Now()
}

// choreographySagaInstance is a minimal Saga instance wrapper for choreography mode.
type choreographySagaInstance struct {
	sagaID string
	def    saga.SagaDefinition
}

func (c *choreographySagaInstance) GetID() string                       { return c.sagaID }
func (c *choreographySagaInstance) GetDefinitionID() string             { return c.def.GetID() }
func (c *choreographySagaInstance) GetState() saga.SagaState            { return saga.StateRunning }
func (c *choreographySagaInstance) GetCurrentStep() int                 { return 0 }
func (c *choreographySagaInstance) GetStartTime() time.Time             { return time.Time{} }
func (c *choreographySagaInstance) GetEndTime() time.Time               { return time.Time{} }
func (c *choreographySagaInstance) GetResult() interface{}              { return nil }
func (c *choreographySagaInstance) GetError() *saga.SagaError           { return nil }
func (c *choreographySagaInstance) GetTotalSteps() int                  { return len(c.def.GetSteps()) }
func (c *choreographySagaInstance) GetCompletedSteps() int              { return 0 }
func (c *choreographySagaInstance) GetCreatedAt() time.Time             { return time.Time{} }
func (c *choreographySagaInstance) GetUpdatedAt() time.Time             { return time.Time{} }
func (c *choreographySagaInstance) GetTimeout() time.Duration           { return c.def.GetTimeout() }
func (c *choreographySagaInstance) GetMetadata() map[string]interface{} { return c.def.GetMetadata() }
func (c *choreographySagaInstance) GetTraceID() string                  { return "" }
func (c *choreographySagaInstance) IsTerminal() bool                    { return false }
func (c *choreographySagaInstance) IsActive() bool                      { return true }
