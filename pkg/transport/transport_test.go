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

package transport

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
)

func init() {
	// Initialize logger for tests
	logger.Logger, _ = zap.NewDevelopment()
}

// MockTransport implements Transport interface for testing
type MockTransport struct {
	name           string
	address        string
	started        bool
	stopped        bool
	startErr       error
	stopErr        error
	startCallCount int
	stopCallCount  int
	mu             sync.RWMutex
}

func NewMockTransport(name, address string) *MockTransport {
	return &MockTransport{
		name:    name,
		address: address,
	}
}

func (m *MockTransport) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startCallCount++
	if m.startErr != nil {
		return m.startErr
	}
	m.started = true
	return nil
}

func (m *MockTransport) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopCallCount++
	if m.stopErr != nil {
		return m.stopErr
	}
	m.stopped = true
	return nil
}

func (m *MockTransport) GetName() string {
	return m.name
}

func (m *MockTransport) GetAddress() string {
	return m.address
}

func (m *MockTransport) SetStartError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startErr = err
}

func (m *MockTransport) SetStopError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopErr = err
}

func (m *MockTransport) IsStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started
}

func (m *MockTransport) IsStopped() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stopped
}

func (m *MockTransport) GetStartCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.startCallCount
}

func (m *MockTransport) GetStopCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stopCallCount
}

func TestNewManager(t *testing.T) {
	manager := NewManager()

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.transports)
	assert.Equal(t, 0, len(manager.transports))
}

func TestManager_Register(t *testing.T) {
	manager := NewManager()
	transport1 := NewMockTransport("http", ":8080")
	transport2 := NewMockTransport("grpc", ":9090")

	// Test registering transports
	manager.Register(transport1)
	manager.Register(transport2)

	transports := manager.GetTransports()
	assert.Equal(t, 2, len(transports))
	assert.Equal(t, "http", transports[0].GetName())
	assert.Equal(t, "grpc", transports[1].GetName())
}

func TestManager_Register_Concurrent(t *testing.T) {
	manager := NewManager()
	const numTransports = 100

	var wg sync.WaitGroup
	wg.Add(numTransports)

	// Register transports concurrently
	for i := 0; i < numTransports; i++ {
		go func(i int) {
			defer wg.Done()
			transport := NewMockTransport(fmt.Sprintf("transport-%d", i), fmt.Sprintf(":%d", 8000+i))
			manager.Register(transport)
		}(i)
	}

	wg.Wait()

	transports := manager.GetTransports()
	assert.Equal(t, numTransports, len(transports))
}

func TestManager_Start_Success(t *testing.T) {
	manager := NewManager()
	transport1 := NewMockTransport("http", ":8080")
	transport2 := NewMockTransport("grpc", ":9090")

	manager.Register(transport1)
	manager.Register(transport2)

	ctx := context.Background()
	err := manager.Start(ctx)

	assert.NoError(t, err)
	assert.True(t, transport1.IsStarted())
	assert.True(t, transport2.IsStarted())
	assert.Equal(t, 1, transport1.GetStartCallCount())
	assert.Equal(t, 1, transport2.GetStartCallCount())
}

func TestManager_Start_WithError(t *testing.T) {
	manager := NewManager()
	transport1 := NewMockTransport("http", ":8080")
	transport2 := NewMockTransport("grpc", ":9090")

	// Set error for second transport
	expectedErr := errors.New("failed to start grpc transport")
	transport2.SetStartError(expectedErr)

	manager.Register(transport1)
	manager.Register(transport2)

	ctx := context.Background()
	err := manager.Start(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start grpc transport")
	assert.True(t, transport1.IsStarted())
	assert.False(t, transport2.IsStarted())
}

func TestManager_Start_EmptyManager(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	err := manager.Start(ctx)
	assert.NoError(t, err)
}

func TestManager_Stop_Success(t *testing.T) {
	manager := NewManager()
	transport1 := NewMockTransport("http", ":8080")
	transport2 := NewMockTransport("grpc", ":9090")

	manager.Register(transport1)
	manager.Register(transport2)

	// Start transports first
	ctx := context.Background()
	err := manager.Start(ctx)
	require.NoError(t, err)

	// Stop transports
	err = manager.Stop(5 * time.Second)
	assert.NoError(t, err)
	assert.True(t, transport1.IsStopped())
	assert.True(t, transport2.IsStopped())
	assert.Equal(t, 1, transport1.GetStopCallCount())
	assert.Equal(t, 1, transport2.GetStopCallCount())
}

func TestManager_Stop_WithError(t *testing.T) {
	manager := NewManager()
	transport1 := NewMockTransport("http", ":8080")
	transport2 := NewMockTransport("grpc", ":9090")

	// Set error for first transport
	expectedErr := errors.New("failed to stop http transport")
	transport1.SetStopError(expectedErr)

	manager.Register(transport1)
	manager.Register(transport2)

	// Start transports first
	ctx := context.Background()
	err := manager.Start(ctx)
	require.NoError(t, err)

	// Stop transports - should continue even with error and return errors
	err = manager.Stop(5 * time.Second)
	assert.Error(t, err) // Stop method now returns errors

	// Check that it's a MultiError with the expected error
	multiErr, ok := err.(*MultiError)
	assert.True(t, ok, "Error should be a MultiError")
	assert.Equal(t, 1, len(multiErr.Errors))
	assert.Equal(t, "http", multiErr.Errors[0].TransportName)
	assert.Contains(t, multiErr.Errors[0].Error(), "failed to stop http transport")

	assert.False(t, transport1.IsStopped()) // Failed to stop
	assert.True(t, transport2.IsStopped())  // Should still stop
}

func TestManager_Stop_WithMultipleErrors(t *testing.T) {
	manager := NewManager()
	transport1 := NewMockTransport("http", ":8080")
	transport2 := NewMockTransport("grpc", ":9090")
	transport3 := NewMockTransport("websocket", ":8081")

	// Set errors for first two transports
	err1 := errors.New("failed to stop http transport")
	err2 := errors.New("failed to stop grpc transport")
	transport1.SetStopError(err1)
	transport2.SetStopError(err2)

	manager.Register(transport1)
	manager.Register(transport2)
	manager.Register(transport3)

	// Start transports first
	ctx := context.Background()
	err := manager.Start(ctx)
	require.NoError(t, err)

	// Stop transports - should collect all errors
	err = manager.Stop(5 * time.Second)
	assert.Error(t, err)

	// Check that it's a MultiError with both expected errors
	multiErr, ok := err.(*MultiError)
	assert.True(t, ok, "Error should be a MultiError")
	assert.Equal(t, 2, len(multiErr.Errors))

	// Check error details
	assert.Equal(t, "http", multiErr.Errors[0].TransportName)
	assert.Contains(t, multiErr.Errors[0].Error(), "failed to stop http transport")
	assert.Equal(t, "grpc", multiErr.Errors[1].TransportName)
	assert.Contains(t, multiErr.Errors[1].Error(), "failed to stop grpc transport")

	// Check transport states
	assert.False(t, transport1.IsStopped()) // Failed to stop
	assert.False(t, transport2.IsStopped()) // Failed to stop
	assert.True(t, transport3.IsStopped())  // Should still stop successfully
}

func TestManager_Stop_WithTimeout(t *testing.T) {
	manager := NewManager()
	transport := NewMockTransport("slow", ":8080")

	// Mock transport that takes time to stop
	transport.SetStopError(nil)
	manager.Register(transport)

	// Start transport first
	ctx := context.Background()
	err := manager.Start(ctx)
	require.NoError(t, err)

	// Test with very short timeout
	start := time.Now()
	err = manager.Stop(1 * time.Millisecond)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.True(t, duration < 100*time.Millisecond) // Should respect timeout
}

func TestManager_Stop_EmptyManager(t *testing.T) {
	manager := NewManager()

	err := manager.Stop(5 * time.Second)
	assert.NoError(t, err)
}

func TestManager_GetTransports(t *testing.T) {
	manager := NewManager()
	transport1 := NewMockTransport("http", ":8080")
	transport2 := NewMockTransport("grpc", ":9090")

	// Test empty manager
	transports := manager.GetTransports()
	assert.Equal(t, 0, len(transports))

	// Add transports
	manager.Register(transport1)
	manager.Register(transport2)

	transports = manager.GetTransports()
	assert.Equal(t, 2, len(transports))

	// Verify returned slice is a copy (modification doesn't affect original)
	transports[0] = NewMockTransport("modified", ":1234")
	originalTransports := manager.GetTransports()
	assert.Equal(t, "http", originalTransports[0].GetName())
}

func TestManager_GetTransports_Concurrent(t *testing.T) {
	manager := NewManager()
	transport := NewMockTransport("test", ":8080")
	manager.Register(transport)

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent access to GetTransports
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			transports := manager.GetTransports()
			assert.Equal(t, 1, len(transports))
			assert.Equal(t, "test", transports[0].GetName())
		}()
	}

	wg.Wait()
}

func TestManager_StartStop_Lifecycle(t *testing.T) {
	manager := NewManager()
	transport := NewMockTransport("test", ":8080")
	manager.Register(transport)

	ctx := context.Background()

	// Test multiple start/stop cycles
	for i := 0; i < 3; i++ {
		err := manager.Start(ctx)
		assert.NoError(t, err)

		err = manager.Stop(5 * time.Second)
		assert.NoError(t, err)
	}

	// Verify call counts
	assert.Equal(t, 3, transport.GetStartCallCount())
	assert.Equal(t, 3, transport.GetStopCallCount())
}

func TestManager_ThreadSafety(t *testing.T) {
	manager := NewManager()
	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // Register, Start, Stop operations

	// Concurrent register operations
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			transport := NewMockTransport(fmt.Sprintf("transport-%d", i), fmt.Sprintf(":%d", 8000+i))
			manager.Register(transport)
		}(i)
	}

	// Concurrent start operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			ctx := context.Background()
			_ = manager.Start(ctx) // Ignore errors in concurrent test
		}()
	}

	// Concurrent stop operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = manager.Stop(100 * time.Millisecond) // Ignore errors in concurrent test
		}()
	}

	wg.Wait()

	// Verify no race conditions occurred
	transports := manager.GetTransports()
	assert.Equal(t, numGoroutines, len(transports))
}

func TestTransportError_Error(t *testing.T) {
	err := &TransportError{
		TransportName: "http",
		Err:           errors.New("connection failed"),
	}

	expected := "transport 'http': connection failed"
	assert.Equal(t, expected, err.Error())
}

func TestMultiError_Error(t *testing.T) {
	tests := []struct {
		name     string
		errors   []TransportError
		expected string
	}{
		{
			name:     "no errors",
			errors:   []TransportError{},
			expected: "no errors",
		},
		{
			name: "single error",
			errors: []TransportError{
				{TransportName: "http", Err: errors.New("failed")},
			},
			expected: "transport 'http': failed",
		},
		{
			name: "multiple errors",
			errors: []TransportError{
				{TransportName: "http", Err: errors.New("http failed")},
				{TransportName: "grpc", Err: errors.New("grpc failed")},
			},
			expected: "multiple transport errors: transport 'http': http failed; transport 'grpc': grpc failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiErr := &MultiError{Errors: tt.errors}
			assert.Equal(t, tt.expected, multiErr.Error())
		})
	}
}

func TestMultiError_HasErrors(t *testing.T) {
	// Empty MultiError
	multiErr := &MultiError{Errors: []TransportError{}}
	assert.False(t, multiErr.HasErrors())

	// MultiError with errors
	multiErr = &MultiError{
		Errors: []TransportError{
			{TransportName: "http", Err: errors.New("failed")},
		},
	}
	assert.True(t, multiErr.HasErrors())
}

func TestMultiError_Unwrap(t *testing.T) {
	// Empty MultiError
	multiErr := &MultiError{Errors: []TransportError{}}
	assert.Nil(t, multiErr.Unwrap())

	// Single error
	originalErr := errors.New("original error")
	multiErr = &MultiError{
		Errors: []TransportError{
			{TransportName: "http", Err: originalErr},
		},
	}
	assert.Equal(t, originalErr, multiErr.Unwrap())

	// Multiple errors
	multiErr = &MultiError{
		Errors: []TransportError{
			{TransportName: "http", Err: errors.New("error1")},
			{TransportName: "grpc", Err: errors.New("error2")},
		},
	}
	assert.Nil(t, multiErr.Unwrap())
}

func TestMultiError_GetErrorByTransport(t *testing.T) {
	httpErr := errors.New("http error")
	grpcErr := errors.New("grpc error")

	multiErr := &MultiError{
		Errors: []TransportError{
			{TransportName: "http", Err: httpErr},
			{TransportName: "grpc", Err: grpcErr},
		},
	}

	// Found transport error
	transportErr := multiErr.GetErrorByTransport("http")
	assert.NotNil(t, transportErr)
	assert.Equal(t, "http", transportErr.TransportName)
	assert.Equal(t, httpErr, transportErr.Err)

	// Not found transport error
	transportErr = multiErr.GetErrorByTransport("websocket")
	assert.Nil(t, transportErr)

	// Empty MultiError
	emptyMultiErr := &MultiError{Errors: []TransportError{}}
	transportErr = emptyMultiErr.GetErrorByTransport("http")
	assert.Nil(t, transportErr)
}

func TestIsStopError(t *testing.T) {
	// Non-error case
	assert.False(t, IsStopError(nil))

	// Regular error
	assert.False(t, IsStopError(errors.New("regular error")))

	// TransportError
	transportErr := &TransportError{
		TransportName: "http",
		Err:           errors.New("stop failed"),
	}
	assert.True(t, IsStopError(transportErr))

	// MultiError
	multiErr := &MultiError{
		Errors: []TransportError{*transportErr},
	}
	assert.True(t, IsStopError(multiErr))
}

func TestExtractStopErrors(t *testing.T) {
	// Nil error
	stopErrors := ExtractStopErrors(nil)
	assert.Nil(t, stopErrors)

	// Regular error
	stopErrors = ExtractStopErrors(errors.New("regular error"))
	assert.Nil(t, stopErrors)

	// TransportError
	transportErr := &TransportError{
		TransportName: "http",
		Err:           errors.New("stop failed"),
	}
	stopErrors = ExtractStopErrors(transportErr)
	assert.Equal(t, 1, len(stopErrors))
	assert.Equal(t, "http", stopErrors[0].TransportName)

	// MultiError
	multiErr := &MultiError{
		Errors: []TransportError{
			{TransportName: "http", Err: errors.New("http failed")},
			{TransportName: "grpc", Err: errors.New("grpc failed")},
		},
	}
	stopErrors = ExtractStopErrors(multiErr)
	assert.Equal(t, 2, len(stopErrors))
	assert.Equal(t, "http", stopErrors[0].TransportName)
	assert.Equal(t, "grpc", stopErrors[1].TransportName)
}

// Benchmark tests
func BenchmarkManager_Register(b *testing.B) {
	manager := NewManager()
	transports := make([]*MockTransport, b.N)

	for i := 0; i < b.N; i++ {
		transports[i] = NewMockTransport("transport-"+string(rune(i)), ":"+string(rune(8000+i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Register(transports[i])
	}
}

func BenchmarkManager_GetTransports(b *testing.B) {
	manager := NewManager()

	// Setup: register 100 transports
	for i := 0; i < 100; i++ {
		transport := NewMockTransport("transport-"+string(rune(i)), ":"+string(rune(8000+i)))
		manager.Register(transport)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.GetTransports()
	}
}

func BenchmarkManager_Start(b *testing.B) {
	manager := NewManager()
	transport := NewMockTransport("test", ":8080")
	manager.Register(transport)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.Start(ctx)
	}
}
