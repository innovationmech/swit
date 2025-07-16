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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTransport is a mock implementation of Transport interface
type MockTransport struct {
	mock.Mock
}

func (m *MockTransport) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTransport) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTransport) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTransport) Address() string {
	args := m.Called()
	return args.String(0)
}

func TestNewManager(t *testing.T) {
	manager := NewManager()

	assert.NotNil(t, manager)
	assert.Empty(t, manager.transports)
	// Verify mutex is properly initialized by checking we can lock/unlock
	manager.mu.Lock()
	defer manager.mu.Unlock()
}

func TestManager_Register(t *testing.T) {
	manager := NewManager()
	mockTransport := &MockTransport{}
	mockTransport.On("Name").Return("test-transport")

	// Test registering a transport
	manager.Register(mockTransport)

	transports := manager.GetTransports()
	assert.Len(t, transports, 1)
	assert.Equal(t, mockTransport, transports[0])
}

func TestManager_RegisterMultiple(t *testing.T) {
	manager := NewManager()

	transport1 := &MockTransport{}
	transport1.On("Name").Return("transport1")

	transport2 := &MockTransport{}
	transport2.On("Name").Return("transport2")

	manager.Register(transport1)
	manager.Register(transport2)

	transports := manager.GetTransports()
	assert.Len(t, transports, 2)
	assert.Contains(t, transports, transport1)
	assert.Contains(t, transports, transport2)
}

func TestManager_Start_Success(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	transport1 := &MockTransport{}
	transport1.On("Start", ctx).Return(nil)

	transport2 := &MockTransport{}
	transport2.On("Start", ctx).Return(nil)

	manager.Register(transport1)
	manager.Register(transport2)

	err := manager.Start(ctx)

	assert.NoError(t, err)
	transport1.AssertExpectations(t)
	transport2.AssertExpectations(t)
}

func TestManager_Start_WithError(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()
	expectedError := errors.New("start error")

	transport1 := &MockTransport{}
	transport1.On("Start", ctx).Return(expectedError)

	transport2 := &MockTransport{}
	transport2.On("Start", ctx).Return(nil)

	manager.Register(transport1)
	manager.Register(transport2)

	err := manager.Start(ctx)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	transport1.AssertExpectations(t)
	transport2.AssertExpectations(t)
}

func TestManager_Start_EmptyTransports(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	err := manager.Start(ctx)

	assert.NoError(t, err)
}

func TestManager_Stop_Success(t *testing.T) {
	manager := NewManager()
	timeout := 5 * time.Second

	transport1 := &MockTransport{}
	transport1.On("Stop", mock.AnythingOfType("*context.timerCtx")).Return(nil)

	transport2 := &MockTransport{}
	transport2.On("Stop", mock.AnythingOfType("*context.timerCtx")).Return(nil)

	manager.Register(transport1)
	manager.Register(transport2)

	err := manager.Stop(timeout)

	assert.NoError(t, err)
	transport1.AssertExpectations(t)
	transport2.AssertExpectations(t)
}

func TestManager_Stop_WithError(t *testing.T) {
	manager := NewManager()
	timeout := 5 * time.Second
	expectedError := errors.New("stop error")

	transport1 := &MockTransport{}
	transport1.On("Stop", mock.AnythingOfType("*context.timerCtx")).Return(expectedError)

	transport2 := &MockTransport{}
	transport2.On("Stop", mock.AnythingOfType("*context.timerCtx")).Return(nil)

	manager.Register(transport1)
	manager.Register(transport2)

	// Stop should not return error even if individual transports fail
	err := manager.Stop(timeout)

	assert.NoError(t, err)
	transport1.AssertExpectations(t)
	transport2.AssertExpectations(t)
}

func TestManager_Stop_EmptyTransports(t *testing.T) {
	manager := NewManager()
	timeout := 5 * time.Second

	err := manager.Stop(timeout)

	assert.NoError(t, err)
}

func TestManager_Stop_WithTimeout(t *testing.T) {
	manager := NewManager()
	timeout := 10 * time.Millisecond // Very short timeout

	transport := &MockTransport{}
	// Mock a slow stop operation
	transport.On("Stop", mock.AnythingOfType("*context.timerCtx")).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		select {
		case <-ctx.Done():
			// Context timeout
		case <-time.After(100 * time.Millisecond):
			// Simulate slow operation
		}
	}).Return(nil)

	manager.Register(transport)

	start := time.Now()
	err := manager.Stop(timeout)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.True(t, elapsed < 50*time.Millisecond, "Stop should respect timeout")
	transport.AssertExpectations(t)
}

func TestManager_GetTransports(t *testing.T) {
	manager := NewManager()

	transport1 := &MockTransport{}
	transport1.On("Name").Return("transport1")

	transport2 := &MockTransport{}
	transport2.On("Name").Return("transport2")

	manager.Register(transport1)
	manager.Register(transport2)

	transports := manager.GetTransports()

	assert.Len(t, transports, 2)
	assert.Contains(t, transports, transport1)
	assert.Contains(t, transports, transport2)

	// Verify that the returned slice is a copy (not the original)
	transports[0] = nil
	originalTransports := manager.GetTransports()
	assert.NotNil(t, originalTransports[0])
}

func TestManager_ConcurrentAccess(t *testing.T) {
	manager := NewManager()

	// Test concurrent registration and access
	var wg sync.WaitGroup
	transportCount := 100

	// Concurrent registrations
	for i := 0; i < transportCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			transport := &MockTransport{}
			transport.On("Name").Return("transport")
			manager.Register(transport)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.GetTransports()
		}()
	}

	wg.Wait()

	transports := manager.GetTransports()
	assert.Len(t, transports, transportCount)
}

func TestManager_StartStop_Concurrent(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	transport := &MockTransport{}
	transport.On("Start", ctx).Return(nil)
	transport.On("Stop", mock.AnythingOfType("*context.timerCtx")).Return(nil)

	manager.Register(transport)

	var wg sync.WaitGroup

	// Start multiple goroutines trying to start and stop
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.Start(ctx)
		}()
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.Stop(time.Second)
		}()
	}

	wg.Wait()

	// Verify that all calls were made (mock will fail if expectations aren't met)
	transport.AssertExpectations(t)
}

func TestManager_StartMultipleErrors(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	error1 := errors.New("error1")
	error2 := errors.New("error2")

	transport1 := &MockTransport{}
	transport1.On("Start", ctx).Return(error1)

	transport2 := &MockTransport{}
	transport2.On("Start", ctx).Return(error2)

	transport3 := &MockTransport{}
	transport3.On("Start", ctx).Return(nil)

	manager.Register(transport1)
	manager.Register(transport2)
	manager.Register(transport3)

	err := manager.Start(ctx)

	// Should return one of the errors (the first one encountered)
	assert.Error(t, err)
	assert.True(t, err == error1 || err == error2)

	transport1.AssertExpectations(t)
	transport2.AssertExpectations(t)
	transport3.AssertExpectations(t)
}
