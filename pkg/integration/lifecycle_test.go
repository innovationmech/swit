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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLifecycleHandler is a mock handler implementing LifecycleHandler interface
type mockLifecycleHandler struct {
	id               string
	topics           []string
	broker           string
	initCalled       bool
	startCalled      bool
	stopCalled       bool
	shutdownCalled   bool
	initError        error
	startError       error
	stopError        error
	shutdownError    error
	currentState     LifecycleState
	healthStatusFunc func(ctx context.Context) (HealthStatus, error)
}

func newMockLifecycleHandler(id string) *mockLifecycleHandler {
	return &mockLifecycleHandler{
		id:           id,
		topics:       []string{"test.topic"},
		broker:       "",
		currentState: StateCreated,
	}
}

func (m *mockLifecycleHandler) GetHandlerID() string                              { return m.id }
func (m *mockLifecycleHandler) GetTopics() []string                               { return m.topics }
func (m *mockLifecycleHandler) GetBrokerRequirement() string                      { return m.broker }
func (m *mockLifecycleHandler) Handle(ctx context.Context, msg interface{}) error { return nil }
func (m *mockLifecycleHandler) OnError(ctx context.Context, msg interface{}, err error) interface{} {
	return nil
}

func (m *mockLifecycleHandler) Initialize(ctx context.Context) error {
	m.initCalled = true
	if m.initError != nil {
		return m.initError
	}
	m.currentState = StateInitialized
	return nil
}

func (m *mockLifecycleHandler) Start(ctx context.Context) error {
	m.startCalled = true
	if m.startError != nil {
		return m.startError
	}
	m.currentState = StateStarted
	return nil
}

func (m *mockLifecycleHandler) Stop(ctx context.Context) error {
	m.stopCalled = true
	if m.stopError != nil {
		return m.stopError
	}
	m.currentState = StateStopped
	return nil
}

func (m *mockLifecycleHandler) Shutdown(ctx context.Context) error {
	m.shutdownCalled = true
	if m.shutdownError != nil {
		return m.shutdownError
	}
	return nil
}

func (m *mockLifecycleHandler) GetState() LifecycleState {
	return m.currentState
}

func (m *mockLifecycleHandler) GetHealthStatus(ctx context.Context) (HealthStatus, error) {
	if m.healthStatusFunc != nil {
		return m.healthStatusFunc(ctx)
	}
	return HealthStatus{
		Healthy:   m.currentState == StateStarted,
		State:     m.currentState,
		Message:   "Mock handler",
		LastCheck: time.Now(),
	}, nil
}

func TestHandlerLifecycleManager_RegisterHandler(t *testing.T) {
	tests := []struct {
		name        string
		handler     *mockLifecycleHandler
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successful registration",
			handler:     newMockLifecycleHandler("test-handler"),
			expectError: false,
		},
		{
			name:        "nil handler",
			handler:     nil,
			expectError: true,
			errorMsg:    "handler cannot be nil",
		},
		{
			name: "empty handler ID",
			handler: &mockLifecycleHandler{
				id: "",
			},
			expectError: true,
			errorMsg:    "handler ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewHandlerLifecycleManager()

			var err error
			if tt.handler != nil {
				err = manager.RegisterHandler(tt.handler)
			} else {
				err = manager.RegisterHandler(nil)
			}

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				state, err := manager.GetHandlerState(tt.handler.GetHandlerID())
				require.NoError(t, err)
				assert.Equal(t, StateCreated, state)
			}
		})
	}
}

func TestHandlerLifecycleManager_DuplicateRegistration(t *testing.T) {
	manager := NewHandlerLifecycleManager()
	handler := newMockLifecycleHandler("duplicate-handler")

	// First registration should succeed
	err := manager.RegisterHandler(handler)
	require.NoError(t, err)

	// Second registration should fail
	err = manager.RegisterHandler(handler)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestHandlerLifecycleManager_InitializeHandler(t *testing.T) {
	tests := []struct {
		name        string
		setupState  LifecycleState
		initError   error
		expectError bool
		expectState LifecycleState
	}{
		{
			name:        "successful initialization",
			setupState:  StateCreated,
			initError:   nil,
			expectError: false,
			expectState: StateInitialized,
		},
		{
			name:        "initialization error",
			setupState:  StateCreated,
			initError:   errors.New("init failed"),
			expectError: true,
			expectState: StateFailed,
		},
		{
			name:        "wrong state for initialization",
			setupState:  StateInitialized,
			initError:   nil,
			expectError: true,
			expectState: StateInitialized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewHandlerLifecycleManager()
			handler := newMockLifecycleHandler("test-handler")
			handler.initError = tt.initError

			err := manager.RegisterHandler(handler)
			require.NoError(t, err)

			// Set initial state if needed
			if tt.setupState != StateCreated {
				manager.mu.Lock()
				manager.handlers[handler.GetHandlerID()].state = tt.setupState
				manager.mu.Unlock()
			}

			ctx := context.Background()
			err = manager.InitializeHandler(ctx, handler.GetHandlerID())

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.True(t, handler.initCalled)
			}

			state, _ := manager.GetHandlerState(handler.GetHandlerID())
			assert.Equal(t, tt.expectState, state)
		})
	}
}

func TestHandlerLifecycleManager_StartHandler(t *testing.T) {
	tests := []struct {
		name        string
		setupState  LifecycleState
		startError  error
		expectError bool
		expectState LifecycleState
	}{
		{
			name:        "successful start",
			setupState:  StateInitialized,
			startError:  nil,
			expectError: false,
			expectState: StateStarted,
		},
		{
			name:        "start error",
			setupState:  StateInitialized,
			startError:  errors.New("start failed"),
			expectError: true,
			expectState: StateFailed,
		},
		{
			name:        "wrong state for start",
			setupState:  StateCreated,
			startError:  nil,
			expectError: true,
			expectState: StateCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewHandlerLifecycleManager()
			handler := newMockLifecycleHandler("test-handler")
			handler.startError = tt.startError

			err := manager.RegisterHandler(handler)
			require.NoError(t, err)

			// Set initial state
			manager.mu.Lock()
			manager.handlers[handler.GetHandlerID()].state = tt.setupState
			manager.mu.Unlock()

			ctx := context.Background()
			err = manager.StartHandler(ctx, handler.GetHandlerID())

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.True(t, handler.startCalled)
			}

			state, _ := manager.GetHandlerState(handler.GetHandlerID())
			assert.Equal(t, tt.expectState, state)
		})
	}
}

func TestHandlerLifecycleManager_StopHandler(t *testing.T) {
	tests := []struct {
		name          string
		setupState    LifecycleState
		stopError     error
		shutdownError error
		expectError   bool
		expectState   LifecycleState
	}{
		{
			name:          "successful stop",
			setupState:    StateStarted,
			stopError:     nil,
			shutdownError: nil,
			expectError:   false,
			expectState:   StateStopped,
		},
		{
			name:          "stop error",
			setupState:    StateStarted,
			stopError:     errors.New("stop failed"),
			shutdownError: nil,
			expectError:   true,
			expectState:   StateFailed,
		},
		{
			name:          "shutdown error",
			setupState:    StateStarted,
			stopError:     nil,
			shutdownError: errors.New("shutdown failed"),
			expectError:   true,
			expectState:   StateFailed,
		},
		{
			name:          "wrong state for stop",
			setupState:    StateCreated,
			stopError:     nil,
			shutdownError: nil,
			expectError:   true,
			expectState:   StateCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewHandlerLifecycleManager()
			handler := newMockLifecycleHandler("test-handler")
			handler.stopError = tt.stopError
			handler.shutdownError = tt.shutdownError

			err := manager.RegisterHandler(handler)
			require.NoError(t, err)

			// Set initial state
			manager.mu.Lock()
			manager.handlers[handler.GetHandlerID()].state = tt.setupState
			manager.mu.Unlock()

			ctx := context.Background()
			err = manager.StopHandler(ctx, handler.GetHandlerID())

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.True(t, handler.stopCalled)
				assert.True(t, handler.shutdownCalled)
			}

			state, _ := manager.GetHandlerState(handler.GetHandlerID())
			assert.Equal(t, tt.expectState, state)
		})
	}
}

func TestHandlerLifecycleManager_FullLifecycle(t *testing.T) {
	manager := NewHandlerLifecycleManager()
	handler := newMockLifecycleHandler("test-handler")

	// Register
	err := manager.RegisterHandler(handler)
	require.NoError(t, err)

	state, err := manager.GetHandlerState(handler.GetHandlerID())
	require.NoError(t, err)
	assert.Equal(t, StateCreated, state)

	ctx := context.Background()

	// Initialize
	err = manager.InitializeHandler(ctx, handler.GetHandlerID())
	require.NoError(t, err)
	assert.True(t, handler.initCalled)

	state, err = manager.GetHandlerState(handler.GetHandlerID())
	require.NoError(t, err)
	assert.Equal(t, StateInitialized, state)

	// Start
	err = manager.StartHandler(ctx, handler.GetHandlerID())
	require.NoError(t, err)
	assert.True(t, handler.startCalled)

	state, err = manager.GetHandlerState(handler.GetHandlerID())
	require.NoError(t, err)
	assert.Equal(t, StateStarted, state)

	// Stop
	err = manager.StopHandler(ctx, handler.GetHandlerID())
	require.NoError(t, err)
	assert.True(t, handler.stopCalled)
	assert.True(t, handler.shutdownCalled)

	state, err = manager.GetHandlerState(handler.GetHandlerID())
	require.NoError(t, err)
	assert.Equal(t, StateStopped, state)

	// Unregister
	err = manager.UnregisterHandler(handler.GetHandlerID())
	require.NoError(t, err)

	// Verify handler is no longer registered
	_, err = manager.GetHandlerState(handler.GetHandlerID())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestHandlerLifecycleManager_InitializeAll(t *testing.T) {
	manager := NewHandlerLifecycleManager()

	handler1 := newMockLifecycleHandler("handler-1")
	handler2 := newMockLifecycleHandler("handler-2")
	handler3 := newMockLifecycleHandler("handler-3")
	handler3.initError = errors.New("init failed")

	err := manager.RegisterHandler(handler1)
	require.NoError(t, err)
	err = manager.RegisterHandler(handler2)
	require.NoError(t, err)
	err = manager.RegisterHandler(handler3)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.InitializeAll(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize")

	// Check that successful handlers were initialized
	state1, _ := manager.GetHandlerState("handler-1")
	assert.Equal(t, StateInitialized, state1)

	state2, _ := manager.GetHandlerState("handler-2")
	assert.Equal(t, StateInitialized, state2)

	// Failed handler should be in failed state
	state3, _ := manager.GetHandlerState("handler-3")
	assert.Equal(t, StateFailed, state3)
}

func TestHandlerLifecycleManager_StartAll(t *testing.T) {
	manager := NewHandlerLifecycleManager()

	handler1 := newMockLifecycleHandler("handler-1")
	handler2 := newMockLifecycleHandler("handler-2")

	err := manager.RegisterHandler(handler1)
	require.NoError(t, err)
	err = manager.RegisterHandler(handler2)
	require.NoError(t, err)

	ctx := context.Background()

	// Initialize all first
	err = manager.InitializeAll(ctx)
	require.NoError(t, err)

	// Start all
	err = manager.StartAll(ctx)
	require.NoError(t, err)

	state1, _ := manager.GetHandlerState("handler-1")
	assert.Equal(t, StateStarted, state1)

	state2, _ := manager.GetHandlerState("handler-2")
	assert.Equal(t, StateStarted, state2)
}

func TestHandlerLifecycleManager_StopAll(t *testing.T) {
	manager := NewHandlerLifecycleManager()

	handler1 := newMockLifecycleHandler("handler-1")
	handler2 := newMockLifecycleHandler("handler-2")

	err := manager.RegisterHandler(handler1)
	require.NoError(t, err)
	err = manager.RegisterHandler(handler2)
	require.NoError(t, err)

	ctx := context.Background()

	// Initialize and start all
	err = manager.InitializeAll(ctx)
	require.NoError(t, err)
	err = manager.StartAll(ctx)
	require.NoError(t, err)

	// Stop all
	err = manager.StopAll(ctx)
	require.NoError(t, err)

	state1, _ := manager.GetHandlerState("handler-1")
	assert.Equal(t, StateStopped, state1)

	state2, _ := manager.GetHandlerState("handler-2")
	assert.Equal(t, StateStopped, state2)
}

func TestHandlerLifecycleManager_Hooks(t *testing.T) {
	manager := NewHandlerLifecycleManager()
	handler := newMockLifecycleHandler("test-handler")

	var hooksCalled []string

	// Register hooks for different states
	manager.RegisterHook(StateInitializing, func(ctx context.Context, handlerID string, state LifecycleState) error {
		hooksCalled = append(hooksCalled, "initializing")
		return nil
	})

	manager.RegisterHook(StateInitialized, func(ctx context.Context, handlerID string, state LifecycleState) error {
		hooksCalled = append(hooksCalled, "initialized")
		return nil
	})

	manager.RegisterHook(StateStarting, func(ctx context.Context, handlerID string, state LifecycleState) error {
		hooksCalled = append(hooksCalled, "starting")
		return nil
	})

	manager.RegisterHook(StateStarted, func(ctx context.Context, handlerID string, state LifecycleState) error {
		hooksCalled = append(hooksCalled, "started")
		return nil
	})

	err := manager.RegisterHandler(handler)
	require.NoError(t, err)

	ctx := context.Background()

	// Initialize
	err = manager.InitializeHandler(ctx, handler.GetHandlerID())
	require.NoError(t, err)

	// Start
	err = manager.StartHandler(ctx, handler.GetHandlerID())
	require.NoError(t, err)

	// Verify hooks were called in order
	assert.Equal(t, []string{"initializing", "initialized", "starting", "started"}, hooksCalled)
}

func TestHandlerLifecycleManager_HookError(t *testing.T) {
	manager := NewHandlerLifecycleManager()
	handler := newMockLifecycleHandler("test-handler")

	// Register a hook that returns an error
	manager.RegisterHook(StateInitializing, func(ctx context.Context, handlerID string, state LifecycleState) error {
		return errors.New("hook failed")
	})

	err := manager.RegisterHandler(handler)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.InitializeHandler(ctx, handler.GetHandlerID())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook failed")

	// Handler should be in failed state
	state, _ := manager.GetHandlerState(handler.GetHandlerID())
	assert.Equal(t, StateFailed, state)
}

func TestHandlerLifecycleManager_GetHealthStatus(t *testing.T) {
	manager := NewHandlerLifecycleManager()
	handler := newMockLifecycleHandler("test-handler")

	// Set custom health status function
	handler.healthStatusFunc = func(ctx context.Context) (HealthStatus, error) {
		return HealthStatus{
			Healthy:   true,
			State:     StateStarted,
			Message:   "Custom health status",
			LastCheck: time.Now(),
		}, nil
	}

	err := manager.RegisterHandler(handler)
	require.NoError(t, err)

	ctx := context.Background()

	// Initialize and start
	err = manager.InitializeHandler(ctx, handler.GetHandlerID())
	require.NoError(t, err)
	err = manager.StartHandler(ctx, handler.GetHandlerID())
	require.NoError(t, err)

	// Get health status
	status, err := manager.GetHandlerHealthStatus(ctx, handler.GetHandlerID())
	require.NoError(t, err)
	assert.True(t, status.Healthy)
	assert.Equal(t, StateStarted, status.State)
	assert.Equal(t, "Custom health status", status.Message)
}

func TestHandlerLifecycleManager_GetAllHandlerStates(t *testing.T) {
	manager := NewHandlerLifecycleManager()

	handler1 := newMockLifecycleHandler("handler-1")
	handler2 := newMockLifecycleHandler("handler-2")
	handler3 := newMockLifecycleHandler("handler-3")

	err := manager.RegisterHandler(handler1)
	require.NoError(t, err)
	err = manager.RegisterHandler(handler2)
	require.NoError(t, err)
	err = manager.RegisterHandler(handler3)
	require.NoError(t, err)

	ctx := context.Background()

	// Initialize handler1 and handler2
	err = manager.InitializeHandler(ctx, "handler-1")
	require.NoError(t, err)
	err = manager.InitializeHandler(ctx, "handler-2")
	require.NoError(t, err)

	// Start handler1
	err = manager.StartHandler(ctx, "handler-1")
	require.NoError(t, err)

	// Get all states
	states := manager.GetAllHandlerStates()
	assert.Equal(t, 3, len(states))
	assert.Equal(t, StateStarted, states["handler-1"])
	assert.Equal(t, StateInitialized, states["handler-2"])
	assert.Equal(t, StateCreated, states["handler-3"])
}

func TestHandlerLifecycleManager_UnregisterHandlerInWrongState(t *testing.T) {
	manager := NewHandlerLifecycleManager()
	handler := newMockLifecycleHandler("test-handler")

	err := manager.RegisterHandler(handler)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.InitializeHandler(ctx, handler.GetHandlerID())
	require.NoError(t, err)

	// Try to unregister without stopping
	err = manager.UnregisterHandler(handler.GetHandlerID())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be stopped")
}

func TestHandlerLifecycleManager_DefaultHealthStatus(t *testing.T) {
	manager := NewHandlerLifecycleManager()

	// Create a simple handler without HealthStatusProvider
	handler := newMockLifecycleHandler("test-handler")
	// Remove the health status function to test default behavior
	handler.healthStatusFunc = nil

	err := manager.RegisterHandler(handler)
	require.NoError(t, err)

	ctx := context.Background()

	// Get health status for created handler
	status, err := manager.GetHandlerHealthStatus(ctx, handler.GetHandlerID())
	require.NoError(t, err)
	assert.False(t, status.Healthy) // Not started yet
	assert.Equal(t, StateCreated, status.State)

	// Initialize and start
	err = manager.InitializeHandler(ctx, handler.GetHandlerID())
	require.NoError(t, err)
	err = manager.StartHandler(ctx, handler.GetHandlerID())
	require.NoError(t, err)

	// Get health status for started handler
	status, err = manager.GetHandlerHealthStatus(ctx, handler.GetHandlerID())
	require.NoError(t, err)
	assert.True(t, status.Healthy) // Started handlers are healthy
	assert.Equal(t, StateStarted, status.State)
}
