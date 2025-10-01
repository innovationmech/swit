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

package integration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
	"go.uber.org/zap"
)

// LifecycleState represents the current state of a handler in its lifecycle.
type LifecycleState string

const (
	// StateCreated indicates the handler has been created but not initialized
	StateCreated LifecycleState = "created"
	// StateInitializing indicates the handler is in the process of initialization
	StateInitializing LifecycleState = "initializing"
	// StateInitialized indicates the handler has been initialized
	StateInitialized LifecycleState = "initialized"
	// StateStarting indicates the handler is in the process of starting
	StateStarting LifecycleState = "starting"
	// StateStarted indicates the handler has been started and is running
	StateStarted LifecycleState = "started"
	// StateStopping indicates the handler is in the process of stopping
	StateStopping LifecycleState = "stopping"
	// StateStopped indicates the handler has been stopped
	StateStopped LifecycleState = "stopped"
	// StateFailed indicates the handler encountered an error during lifecycle transition
	StateFailed LifecycleState = "failed"
)

// LifecycleHandler extends EventHandler with explicit lifecycle management capabilities.
// This interface is optional; handlers that don't implement it will fall back to
// using Initialize and Shutdown methods.
type LifecycleHandler interface {
	server.EventHandler

	// Start starts the handler and prepares it for message processing.
	// This is called after Initialize and before processing begins.
	Start(ctx context.Context) error

	// Stop stops the handler and releases resources.
	// This is called before Shutdown for graceful termination.
	Stop(ctx context.Context) error

	// GetState returns the current lifecycle state of the handler.
	GetState() LifecycleState
}

// LifecycleHook represents a callback that can be invoked during lifecycle transitions.
type LifecycleHook func(ctx context.Context, handlerID string, state LifecycleState) error

// HealthStatusProvider defines an interface for components that can report health status.
type HealthStatusProvider interface {
	// GetHealthStatus returns the current health status of the handler
	GetHealthStatus(ctx context.Context) (HealthStatus, error)
}

// HealthStatus represents the health state of a handler.
type HealthStatus struct {
	// Healthy indicates whether the handler is healthy
	Healthy bool
	// State is the current lifecycle state
	State LifecycleState
	// Message provides additional health information
	Message string
	// LastCheck is the timestamp of the last health check
	LastCheck time.Time
}

// HandlerLifecycleManager manages the lifecycle of event handlers,
// coordinating initialization, startup, shutdown, and health monitoring.
type HandlerLifecycleManager struct {
	mu       sync.RWMutex
	handlers map[string]*handlerLifecycleState
	hooks    map[LifecycleState][]LifecycleHook
}

// handlerLifecycleState tracks the lifecycle state of a single handler.
type handlerLifecycleState struct {
	handler        server.EventHandler
	state          LifecycleState
	lastTransition time.Time
	error          error
	healthStatus   *HealthStatus
}

// NewHandlerLifecycleManager creates a new handler lifecycle manager.
//
// Returns:
//   - *HandlerLifecycleManager: New lifecycle manager instance
//
// Example usage:
//
//	manager := NewHandlerLifecycleManager()
//	manager.RegisterHook(StateStarted, func(ctx context.Context, handlerID string, state LifecycleState) error {
//	    log.Printf("Handler %s started", handlerID)
//	    return nil
//	})
func NewHandlerLifecycleManager() *HandlerLifecycleManager {
	return &HandlerLifecycleManager{
		handlers: make(map[string]*handlerLifecycleState),
		hooks:    make(map[LifecycleState][]LifecycleHook),
	}
}

// RegisterHandler registers a handler with the lifecycle manager.
//
// Parameters:
//   - handler: The event handler to register
//
// Returns:
//   - error: Registration error if handler is invalid
//
// Example usage:
//
//	handler := &MyEventHandler{...}
//	if err := manager.RegisterHandler(handler); err != nil {
//	    return fmt.Errorf("failed to register handler: %w", err)
//	}
func (m *HandlerLifecycleManager) RegisterHandler(handler server.EventHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	handlerID := handler.GetHandlerID()
	if handlerID == "" {
		return fmt.Errorf("handler ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.handlers[handlerID]; exists {
		return fmt.Errorf("handler with ID '%s' is already registered", handlerID)
	}

	m.handlers[handlerID] = &handlerLifecycleState{
		handler:        handler,
		state:          StateCreated,
		lastTransition: time.Now(),
	}

	logger.Logger.Info("Handler registered with lifecycle manager",
		zap.String("handler_id", handlerID),
		zap.String("state", string(StateCreated)))

	return nil
}

// UnregisterHandler removes a handler from the lifecycle manager.
// The handler must be stopped before unregistration.
//
// Parameters:
//   - handlerID: ID of the handler to unregister
//
// Returns:
//   - error: Unregistration error if handler is not stopped or not found
func (m *HandlerLifecycleManager) UnregisterHandler(handlerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.handlers[handlerID]
	if !exists {
		return fmt.Errorf("handler with ID '%s' is not registered", handlerID)
	}

	if state.state != StateStopped && state.state != StateFailed {
		return fmt.Errorf("handler must be stopped before unregistration, current state: %s", state.state)
	}

	delete(m.handlers, handlerID)

	logger.Logger.Info("Handler unregistered from lifecycle manager",
		zap.String("handler_id", handlerID))

	return nil
}

// InitializeHandler initializes a registered handler.
//
// Parameters:
//   - ctx: Context for initialization
//   - handlerID: ID of the handler to initialize
//
// Returns:
//   - error: Initialization error
func (m *HandlerLifecycleManager) InitializeHandler(ctx context.Context, handlerID string) error {
	m.mu.Lock()
	state, exists := m.handlers[handlerID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("handler with ID '%s' is not registered", handlerID)
	}

	if state.state != StateCreated {
		m.mu.Unlock()
		return fmt.Errorf("handler must be in created state, current state: %s", state.state)
	}

	state.state = StateInitializing
	state.lastTransition = time.Now()
	m.mu.Unlock()

	// Execute initialization hooks
	if err := m.executeHooks(ctx, handlerID, StateInitializing); err != nil {
		m.setFailedState(handlerID, err)
		return fmt.Errorf("initialization hook failed: %w", err)
	}

	// Initialize the handler
	if err := state.handler.Initialize(ctx); err != nil {
		m.setFailedState(handlerID, err)
		return fmt.Errorf("handler initialization failed: %w", err)
	}

	// Update state to initialized
	m.mu.Lock()
	state.state = StateInitialized
	state.lastTransition = time.Now()
	state.error = nil
	m.mu.Unlock()

	logger.Logger.Info("Handler initialized",
		zap.String("handler_id", handlerID),
		zap.String("state", string(StateInitialized)))

	// Execute initialized hooks
	if err := m.executeHooks(ctx, handlerID, StateInitialized); err != nil {
		logger.Logger.Warn("Post-initialization hook failed",
			zap.String("handler_id", handlerID),
			zap.Error(err))
	}

	return nil
}

// StartHandler starts a handler, making it ready for message processing.
//
// Parameters:
//   - ctx: Context for startup
//   - handlerID: ID of the handler to start
//
// Returns:
//   - error: Startup error
func (m *HandlerLifecycleManager) StartHandler(ctx context.Context, handlerID string) error {
	m.mu.Lock()
	state, exists := m.handlers[handlerID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("handler with ID '%s' is not registered", handlerID)
	}

	if state.state != StateInitialized {
		m.mu.Unlock()
		return fmt.Errorf("handler must be initialized before starting, current state: %s", state.state)
	}

	state.state = StateStarting
	state.lastTransition = time.Now()
	m.mu.Unlock()

	// Execute starting hooks
	if err := m.executeHooks(ctx, handlerID, StateStarting); err != nil {
		m.setFailedState(handlerID, err)
		return fmt.Errorf("starting hook failed: %w", err)
	}

	// Check if handler implements LifecycleHandler interface
	if lh, ok := state.handler.(LifecycleHandler); ok {
		if err := lh.Start(ctx); err != nil {
			m.setFailedState(handlerID, err)
			return fmt.Errorf("handler start failed: %w", err)
		}
	}

	// Update state to started
	m.mu.Lock()
	state.state = StateStarted
	state.lastTransition = time.Now()
	state.error = nil
	m.mu.Unlock()

	logger.Logger.Info("Handler started",
		zap.String("handler_id", handlerID),
		zap.String("state", string(StateStarted)))

	// Execute started hooks
	if err := m.executeHooks(ctx, handlerID, StateStarted); err != nil {
		logger.Logger.Warn("Post-start hook failed",
			zap.String("handler_id", handlerID),
			zap.Error(err))
	}

	return nil
}

// StopHandler stops a running handler gracefully.
//
// Parameters:
//   - ctx: Context for shutdown
//   - handlerID: ID of the handler to stop
//
// Returns:
//   - error: Shutdown error
func (m *HandlerLifecycleManager) StopHandler(ctx context.Context, handlerID string) error {
	m.mu.Lock()
	state, exists := m.handlers[handlerID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("handler with ID '%s' is not registered", handlerID)
	}

	if state.state != StateStarted {
		m.mu.Unlock()
		return fmt.Errorf("handler must be started before stopping, current state: %s", state.state)
	}

	state.state = StateStopping
	state.lastTransition = time.Now()
	m.mu.Unlock()

	// Execute stopping hooks
	if err := m.executeHooks(ctx, handlerID, StateStopping); err != nil {
		logger.Logger.Warn("Stopping hook failed",
			zap.String("handler_id", handlerID),
			zap.Error(err))
	}

	// Check if handler implements LifecycleHandler interface
	if lh, ok := state.handler.(LifecycleHandler); ok {
		if err := lh.Stop(ctx); err != nil {
			m.setFailedState(handlerID, err)
			return fmt.Errorf("handler stop failed: %w", err)
		}
	}

	// Shutdown the handler
	if err := state.handler.Shutdown(ctx); err != nil {
		m.setFailedState(handlerID, err)
		return fmt.Errorf("handler shutdown failed: %w", err)
	}

	// Update state to stopped
	m.mu.Lock()
	state.state = StateStopped
	state.lastTransition = time.Now()
	state.error = nil
	m.mu.Unlock()

	logger.Logger.Info("Handler stopped",
		zap.String("handler_id", handlerID),
		zap.String("state", string(StateStopped)))

	// Execute stopped hooks
	if err := m.executeHooks(ctx, handlerID, StateStopped); err != nil {
		logger.Logger.Warn("Post-stop hook failed",
			zap.String("handler_id", handlerID),
			zap.Error(err))
	}

	return nil
}

// InitializeAll initializes all registered handlers.
//
// Parameters:
//   - ctx: Context for initialization
//
// Returns:
//   - error: Error if any handler fails to initialize
func (m *HandlerLifecycleManager) InitializeAll(ctx context.Context) error {
	m.mu.RLock()
	handlerIDs := make([]string, 0, len(m.handlers))
	for id := range m.handlers {
		handlerIDs = append(handlerIDs, id)
	}
	m.mu.RUnlock()

	var initErrors []error
	for _, handlerID := range handlerIDs {
		if err := m.InitializeHandler(ctx, handlerID); err != nil {
			logger.Logger.Error("Failed to initialize handler",
				zap.String("handler_id", handlerID),
				zap.Error(err))
			initErrors = append(initErrors, fmt.Errorf("handler %s: %w", handlerID, err))
		}
	}

	if len(initErrors) > 0 {
		return fmt.Errorf("failed to initialize %d handler(s): %v", len(initErrors), initErrors)
	}

	logger.Logger.Info("All handlers initialized successfully",
		zap.Int("count", len(handlerIDs)))

	return nil
}

// StartAll starts all initialized handlers.
//
// Parameters:
//   - ctx: Context for startup
//
// Returns:
//   - error: Error if any handler fails to start
func (m *HandlerLifecycleManager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	handlerIDs := make([]string, 0, len(m.handlers))
	for id, state := range m.handlers {
		if state.state == StateInitialized {
			handlerIDs = append(handlerIDs, id)
		}
	}
	m.mu.RUnlock()

	var startErrors []error
	for _, handlerID := range handlerIDs {
		if err := m.StartHandler(ctx, handlerID); err != nil {
			logger.Logger.Error("Failed to start handler",
				zap.String("handler_id", handlerID),
				zap.Error(err))
			startErrors = append(startErrors, fmt.Errorf("handler %s: %w", handlerID, err))
		}
	}

	if len(startErrors) > 0 {
		return fmt.Errorf("failed to start %d handler(s): %v", len(startErrors), startErrors)
	}

	logger.Logger.Info("All handlers started successfully",
		zap.Int("count", len(handlerIDs)))

	return nil
}

// StopAll stops all running handlers gracefully.
//
// Parameters:
//   - ctx: Context for shutdown
//
// Returns:
//   - error: Error if any handler fails to stop (non-fatal)
func (m *HandlerLifecycleManager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	handlerIDs := make([]string, 0, len(m.handlers))
	for id, state := range m.handlers {
		if state.state == StateStarted {
			handlerIDs = append(handlerIDs, id)
		}
	}
	m.mu.RUnlock()

	var stopErrors []error
	for _, handlerID := range handlerIDs {
		if err := m.StopHandler(ctx, handlerID); err != nil {
			logger.Logger.Error("Failed to stop handler",
				zap.String("handler_id", handlerID),
				zap.Error(err))
			stopErrors = append(stopErrors, fmt.Errorf("handler %s: %w", handlerID, err))
		}
	}

	if len(stopErrors) > 0 {
		return fmt.Errorf("failed to stop %d handler(s): %v", len(stopErrors), stopErrors)
	}

	logger.Logger.Info("All handlers stopped successfully",
		zap.Int("count", len(handlerIDs)))

	return nil
}

// GetHandlerState returns the current lifecycle state of a handler.
//
// Parameters:
//   - handlerID: ID of the handler
//
// Returns:
//   - LifecycleState: Current state of the handler
//   - error: Error if handler is not found
func (m *HandlerLifecycleManager) GetHandlerState(handlerID string) (LifecycleState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.handlers[handlerID]
	if !exists {
		return "", fmt.Errorf("handler with ID '%s' is not registered", handlerID)
	}

	return state.state, nil
}

// GetAllHandlerStates returns the lifecycle states of all registered handlers.
//
// Returns:
//   - map[string]LifecycleState: Map of handler IDs to their states
func (m *HandlerLifecycleManager) GetAllHandlerStates() map[string]LifecycleState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make(map[string]LifecycleState, len(m.handlers))
	for id, state := range m.handlers {
		states[id] = state.state
	}

	return states
}

// GetHandlerHealthStatus returns the health status of a handler.
//
// Parameters:
//   - ctx: Context for health check
//   - handlerID: ID of the handler
//
// Returns:
//   - HealthStatus: Current health status
//   - error: Error if handler is not found or health check fails
func (m *HandlerLifecycleManager) GetHandlerHealthStatus(ctx context.Context, handlerID string) (HealthStatus, error) {
	m.mu.RLock()
	state, exists := m.handlers[handlerID]
	if !exists {
		m.mu.RUnlock()
		return HealthStatus{}, fmt.Errorf("handler with ID '%s' is not registered", handlerID)
	}
	m.mu.RUnlock()

	// Check if handler implements HealthStatusProvider
	if hsp, ok := state.handler.(HealthStatusProvider); ok {
		return hsp.GetHealthStatus(ctx)
	}

	// Default health status based on lifecycle state
	healthy := state.state == StateStarted
	message := fmt.Sprintf("Handler is in %s state", state.state)

	if state.error != nil {
		message = fmt.Sprintf("%s (error: %s)", message, state.error.Error())
	}

	return HealthStatus{
		Healthy:   healthy,
		State:     state.state,
		Message:   message,
		LastCheck: time.Now(),
	}, nil
}

// RegisterHook registers a lifecycle hook for a specific state transition.
//
// Parameters:
//   - state: The lifecycle state to hook into
//   - hook: The hook function to execute
//
// Example usage:
//
//	manager.RegisterHook(StateStarted, func(ctx context.Context, handlerID string, state LifecycleState) error {
//	    // Perform post-start actions
//	    return nil
//	})
func (m *HandlerLifecycleManager) RegisterHook(state LifecycleState, hook LifecycleHook) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hooks[state] = append(m.hooks[state], hook)

	logger.Logger.Debug("Lifecycle hook registered",
		zap.String("state", string(state)))
}

// executeHooks executes all registered hooks for a specific state.
func (m *HandlerLifecycleManager) executeHooks(ctx context.Context, handlerID string, state LifecycleState) error {
	m.mu.RLock()
	hooks := m.hooks[state]
	m.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook(ctx, handlerID, state); err != nil {
			return err
		}
	}

	return nil
}

// setFailedState sets a handler to the failed state with an error.
func (m *HandlerLifecycleManager) setFailedState(handlerID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state, exists := m.handlers[handlerID]; exists {
		state.state = StateFailed
		state.lastTransition = time.Now()
		state.error = err

		logger.Logger.Error("Handler entered failed state",
			zap.String("handler_id", handlerID),
			zap.Error(err))
	}
}
