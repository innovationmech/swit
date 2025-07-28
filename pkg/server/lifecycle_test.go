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

package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/innovationmech/swit/pkg/transport"
)

// Mock transport manager for testing
type mockTransportManager struct {
	mock.Mock
}

func (m *mockTransportManager) GetTransports() []transport.NetworkTransport {
	args := m.Called()
	return args.Get(0).([]transport.NetworkTransport)
}

func (m *mockTransportManager) InitializeAllServices(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockTransportManager) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockTransportManager) Stop(timeout time.Duration) error {
	args := m.Called(timeout)
	return args.Error(0)
}

func TestLifecyclePhase_String(t *testing.T) {
	tests := []struct {
		phase    LifecyclePhase
		expected string
	}{
		{PhaseUninitialized, "uninitialized"},
		{PhaseDependenciesInitialized, "dependencies_initialized"},
		{PhaseTransportsInitialized, "transports_initialized"},
		{PhaseServicesRegistered, "services_registered"},
		{PhaseDiscoveryRegistered, "discovery_registered"},
		{PhaseStarted, "started"},
		{PhaseStopping, "stopping"},
		{PhaseStopped, "stopped"},
		{LifecyclePhase(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.phase.String())
		})
	}
}

func TestNewLifecycleManager(t *testing.T) {
	config := NewServerConfig()
	lm := NewLifecycleManager("test-service", config)

	assert.NotNil(t, lm)
	assert.Equal(t, "test-service", lm.serviceName)
	assert.Equal(t, PhaseUninitialized, lm.currentPhase)
	assert.Equal(t, config, lm.config)
	assert.Empty(t, lm.cleanupFunctions)
}

func TestLifecycleManager_SettersAndGetters(t *testing.T) {
	config := NewServerConfig()
	lm := NewLifecycleManager("test-service", config)

	// Test transport manager
	mockTM := &mockTransportManager{}
	lm.SetTransportManager(mockTM)
	assert.Equal(t, mockTM, lm.transportManager)

	// Test dependencies
	mockDeps := &mockBusinessDependencyContainer{}
	lm.SetDependencies(mockDeps)
	assert.Equal(t, mockDeps, lm.dependencies)

	// Test phase getter
	assert.Equal(t, PhaseUninitialized, lm.GetCurrentPhase())

	// Test state methods
	assert.False(t, lm.IsStarted())
	assert.False(t, lm.IsStopped())
	assert.False(t, lm.IsStopping())
}

func TestLifecycleManager_StartupSequence_Success(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false // Disable discovery for simpler testing

	lm := NewLifecycleManager("test-service", config)

	// Setup mocks
	mockTM := &mockTransportManager{}
	mockDeps := &mockBusinessDependencyContainer{}

	lm.SetTransportManager(mockTM)
	lm.SetDependencies(mockDeps)

	// Setup expectations
	mockTM.On("GetTransports").Return([]transport.NetworkTransport{})
	mockTM.On("InitializeAllServices", mock.Anything).Return(nil)
	mockTM.On("Start", mock.Anything).Return(nil)
	mockDeps.On("Close").Return(nil)

	ctx := context.Background()
	err := lm.StartupSequence(ctx)

	require.NoError(t, err)
	assert.Equal(t, PhaseStarted, lm.GetCurrentPhase())
	assert.True(t, lm.IsStarted())
	assert.False(t, lm.IsStopped())
	assert.False(t, lm.IsStopping())

	mockTM.AssertExpectations(t)
}

func TestLifecycleManager_StartupSequence_TransportInitializationFailure(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false

	lm := NewLifecycleManager("test-service", config)

	// Setup mocks
	mockTM := &mockTransportManager{}
	mockDeps := &mockBusinessDependencyContainer{}

	lm.SetTransportManager(mockTM)
	lm.SetDependencies(mockDeps)

	// Setup expectations - transport initialization will fail
	mockTM.On("GetTransports").Return([]transport.NetworkTransport{})
	mockTM.On("InitializeAllServices", mock.Anything).Return(errors.New("initialization failed"))
	mockDeps.On("Close").Return(nil) // Cleanup should be called

	ctx := context.Background()
	err := lm.StartupSequence(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "services registration")
	assert.Contains(t, err.Error(), "initialization failed")
	assert.NotEqual(t, PhaseStarted, lm.GetCurrentPhase())

	mockTM.AssertExpectations(t)
	mockDeps.AssertExpectations(t)
}

func TestLifecycleManager_StartupSequence_TransportStartFailure(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false

	lm := NewLifecycleManager("test-service", config)

	// Setup mocks
	mockTM := &mockTransportManager{}
	mockDeps := &mockBusinessDependencyContainer{}

	lm.SetTransportManager(mockTM)
	lm.SetDependencies(mockDeps)

	// Setup expectations - transport start will fail
	mockTM.On("GetTransports").Return([]transport.NetworkTransport{})
	mockTM.On("InitializeAllServices", mock.Anything).Return(nil)
	mockTM.On("Start", mock.Anything).Return(errors.New("start failed"))
	mockDeps.On("Close").Return(nil) // Cleanup should be called

	ctx := context.Background()
	err := lm.StartupSequence(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "transports startup")
	assert.Contains(t, err.Error(), "start failed")

	mockTM.AssertExpectations(t)
	mockDeps.AssertExpectations(t)
}

func TestLifecycleManager_ShutdownSequence_Success(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false

	lm := NewLifecycleManager("test-service", config)

	// Setup mocks
	mockTM := &mockTransportManager{}
	mockDeps := &mockBusinessDependencyContainer{}

	lm.SetTransportManager(mockTM)
	lm.SetDependencies(mockDeps)

	// First start the server
	mockTM.On("GetTransports").Return([]transport.NetworkTransport{})
	mockTM.On("InitializeAllServices", mock.Anything).Return(nil)
	mockTM.On("Start", mock.Anything).Return(nil)
	mockDeps.On("Close").Return(nil)

	ctx := context.Background()
	err := lm.StartupSequence(ctx)
	require.NoError(t, err)

	// Now test shutdown
	mockTM.On("Stop", config.ShutdownTimeout).Return(nil)

	err = lm.ShutdownSequence(ctx)
	require.NoError(t, err)
	assert.Equal(t, PhaseStopped, lm.GetCurrentPhase())
	assert.False(t, lm.IsStarted())
	assert.True(t, lm.IsStopped())
	assert.False(t, lm.IsStopping())

	mockTM.AssertExpectations(t)
	mockDeps.AssertExpectations(t)
}

func TestLifecycleManager_ShutdownSequence_AlreadyStopped(t *testing.T) {
	config := NewServerConfig()
	lm := NewLifecycleManager("test-service", config)

	// Set phase to stopped
	lm.currentPhase = PhaseStopped

	ctx := context.Background()
	err := lm.ShutdownSequence(ctx)

	require.NoError(t, err)
	assert.Equal(t, PhaseStopped, lm.GetCurrentPhase())
}

func TestLifecycleManager_ShutdownSequence_WithErrors(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false

	lm := NewLifecycleManager("test-service", config)

	// Setup mocks
	mockTM := &mockTransportManager{}
	mockDeps := &mockBusinessDependencyContainer{}

	lm.SetTransportManager(mockTM)
	lm.SetDependencies(mockDeps)

	// Set to started state
	lm.currentPhase = PhaseStarted

	// Setup expectations - both will fail
	mockTM.On("Stop", config.ShutdownTimeout).Return(errors.New("transport stop failed"))
	mockDeps.On("Close").Return(errors.New("dependency close failed"))

	ctx := context.Background()
	err := lm.ShutdownSequence(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "shutdown errors")
	assert.Contains(t, err.Error(), "transport stop failed")
	assert.Contains(t, err.Error(), "dependency close failed")
	assert.Equal(t, PhaseStopped, lm.GetCurrentPhase())

	mockTM.AssertExpectations(t)
	mockDeps.AssertExpectations(t)
}

func TestLifecycleManager_CleanupFunctions(t *testing.T) {
	config := NewServerConfig()
	lm := NewLifecycleManager("test-service", config)

	// Add some cleanup functions
	var cleanupOrder []int
	lm.addCleanupFunction(func() error {
		cleanupOrder = append(cleanupOrder, 1)
		return nil
	})
	lm.addCleanupFunction(func() error {
		cleanupOrder = append(cleanupOrder, 2)
		return nil
	})
	lm.addCleanupFunction(func() error {
		cleanupOrder = append(cleanupOrder, 3)
		return nil
	})

	// Perform cleanup
	err := lm.performCleanup()
	require.NoError(t, err)

	// Verify cleanup functions were called in reverse order (LIFO)
	assert.Equal(t, []int{3, 2, 1}, cleanupOrder)
	assert.Empty(t, lm.cleanupFunctions) // Should be cleared after cleanup
}

func TestLifecycleManager_CleanupFunctions_WithErrors(t *testing.T) {
	config := NewServerConfig()
	lm := NewLifecycleManager("test-service", config)

	// Add cleanup functions, some will fail
	lm.addCleanupFunction(func() error {
		return nil // Success
	})
	lm.addCleanupFunction(func() error {
		return errors.New("cleanup error 1")
	})
	lm.addCleanupFunction(func() error {
		return errors.New("cleanup error 2")
	})

	// Perform cleanup
	err := lm.performCleanup()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cleanup errors")
	assert.Contains(t, err.Error(), "cleanup error 1")
	assert.Contains(t, err.Error(), "cleanup error 2")
	assert.Empty(t, lm.cleanupFunctions) // Should be cleared even with errors
}

func TestLifecycleManager_HandleStartupError(t *testing.T) {
	config := NewServerConfig()
	lm := NewLifecycleManager("test-service", config)

	// Add a cleanup function
	cleanupCalled := false
	lm.addCleanupFunction(func() error {
		cleanupCalled = true
		return nil
	})

	originalError := errors.New("startup failed")
	err := lm.handleStartupError("test phase", originalError)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "startup failed in test phase")
	assert.Contains(t, err.Error(), "startup failed")
	assert.True(t, cleanupCalled)
}

func TestLifecycleManager_MissingTransportManager(t *testing.T) {
	config := NewServerConfig()
	lm := NewLifecycleManager("test-service", config)

	// Don't set transport manager
	ctx := context.Background()
	err := lm.StartupSequence(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "transport manager not set")
}
