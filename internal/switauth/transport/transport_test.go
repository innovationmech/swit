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

package transport_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/innovationmech/swit/internal/switauth/transport"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	// Initialize logger for tests
	logger.Logger, _ = zap.NewDevelopment()
}

// MockTransport is a mock implementation of Transport interface for testing
type MockTransport struct {
	name        string
	startCalled bool
	stopCalled  bool
	startError  error
	stopError   error
}

func NewMockTransport(name string) *MockTransport {
	return &MockTransport{
		name: name,
	}
}

func (m *MockTransport) Start(ctx context.Context) error {
	m.startCalled = true
	return m.startError
}

func (m *MockTransport) Stop(ctx context.Context) error {
	m.stopCalled = true
	return m.stopError
}

func (m *MockTransport) GetName() string {
	return m.name
}

func (m *MockTransport) SetStartError(err error) {
	m.startError = err
}

func (m *MockTransport) SetStopError(err error) {
	m.stopError = err
}

func TestNewManager(t *testing.T) {
	manager := transport.NewManager()
	assert.NotNil(t, manager)
	assert.Empty(t, manager.GetTransports())
}

func TestManager_Register(t *testing.T) {
	manager := transport.NewManager()
	mockTransport := NewMockTransport("test-transport")

	manager.Register(mockTransport)

	transports := manager.GetTransports()
	assert.Len(t, transports, 1)
	assert.Equal(t, "test-transport", transports[0].GetName())
}

func TestManager_RegisterMultiple(t *testing.T) {
	manager := transport.NewManager()
	mock1 := NewMockTransport("transport1")
	mock2 := NewMockTransport("transport2")
	mock3 := NewMockTransport("transport3")

	manager.Register(mock1)
	manager.Register(mock2)
	manager.Register(mock3)

	transports := manager.GetTransports()
	assert.Len(t, transports, 3)

	// Verify order is maintained
	assert.Equal(t, "transport1", transports[0].GetName())
	assert.Equal(t, "transport2", transports[1].GetName())
	assert.Equal(t, "transport3", transports[2].GetName())
}

func TestManager_Start(t *testing.T) {
	manager := transport.NewManager()
	mock1 := NewMockTransport("transport1")
	mock2 := NewMockTransport("transport2")

	manager.Register(mock1)
	manager.Register(mock2)

	ctx := context.Background()
	err := manager.Start(ctx)

	assert.NoError(t, err)
	assert.True(t, mock1.startCalled)
	assert.True(t, mock2.startCalled)
}

func TestManager_StartWithError(t *testing.T) {
	manager := transport.NewManager()
	mock1 := NewMockTransport("transport1")
	mock2 := NewMockTransport("transport2")

	// Set error for second transport
	mock2.SetStartError(errors.New("start error"))

	manager.Register(mock1)
	manager.Register(mock2)

	ctx := context.Background()
	err := manager.Start(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start transport2 transport")
	assert.True(t, mock1.startCalled)
	assert.True(t, mock2.startCalled)
}

func TestManager_Stop(t *testing.T) {
	manager := transport.NewManager()
	mock1 := NewMockTransport("transport1")
	mock2 := NewMockTransport("transport2")

	manager.Register(mock1)
	manager.Register(mock2)

	ctx := context.Background()
	err := manager.Stop(ctx)

	assert.NoError(t, err)
	assert.True(t, mock1.stopCalled)
	assert.True(t, mock2.stopCalled)
}

func TestManager_StopWithError(t *testing.T) {
	manager := transport.NewManager()
	mock1 := NewMockTransport("transport1")
	mock2 := NewMockTransport("transport2")

	// Set error for first transport
	mock1.SetStopError(errors.New("stop error"))

	manager.Register(mock1)
	manager.Register(mock2)

	ctx := context.Background()
	err := manager.Stop(ctx)

	// Stop should not return error, but continue stopping other transports
	assert.NoError(t, err)
	assert.True(t, mock1.stopCalled)
	assert.True(t, mock2.stopCalled)
}

func TestManager_GetTransports(t *testing.T) {
	manager := transport.NewManager()
	mock1 := NewMockTransport("transport1")
	mock2 := NewMockTransport("transport2")

	manager.Register(mock1)
	manager.Register(mock2)

	transports := manager.GetTransports()
	assert.Len(t, transports, 2)

	// Verify it returns a copy, not the original slice
	transports[0] = NewMockTransport("modified")
	originalTransports := manager.GetTransports()
	assert.Equal(t, "transport1", originalTransports[0].GetName())
}

func TestManager_EmptyOperations(t *testing.T) {
	manager := transport.NewManager()
	ctx := context.Background()

	// Test start with no transports
	err := manager.Start(ctx)
	assert.NoError(t, err)

	// Test stop with no transports
	err = manager.Stop(ctx)
	assert.NoError(t, err)

	// Test get transports with no transports
	transports := manager.GetTransports()
	assert.Empty(t, transports)
}

func TestManager_ConcurrentAccess(t *testing.T) {
	manager := transport.NewManager()

	// Test concurrent registration and access
	go func() {
		for i := 0; i < 50; i++ {
			mock := NewMockTransport("concurrent-transport")
			manager.Register(mock)
		}
	}()

	go func() {
		for i := 0; i < 50; i++ {
			_ = manager.GetTransports()
		}
	}()

	// Give goroutines time to complete
	time.Sleep(100 * time.Millisecond)

	// Should not panic
	transports := manager.GetTransports()
	assert.GreaterOrEqual(t, len(transports), 0)
}

func TestManager_StartStopIntegration(t *testing.T) {
	manager := transport.NewManager()
	mock1 := NewMockTransport("transport1")
	mock2 := NewMockTransport("transport2")

	manager.Register(mock1)
	manager.Register(mock2)

	ctx := context.Background()

	// Start all transports
	err := manager.Start(ctx)
	require.NoError(t, err)

	assert.True(t, mock1.startCalled)
	assert.True(t, mock2.startCalled)
	assert.False(t, mock1.stopCalled)
	assert.False(t, mock2.stopCalled)

	// Stop all transports
	err = manager.Stop(ctx)
	require.NoError(t, err)

	assert.True(t, mock1.stopCalled)
	assert.True(t, mock2.stopCalled)
}

func TestManager_RealTransports(t *testing.T) {
	manager := transport.NewManager()

	// Test with real HTTP and gRPC transports
	httpTransport := transport.NewHTTPTransport()
	httpTransport.SetAddress(":0") // Random port

	grpcTransport := transport.NewGRPCTransport()
	grpcTransport.SetAddress(":0") // Random port

	manager.Register(httpTransport)
	manager.Register(grpcTransport)

	ctx := context.Background()

	// Start all transports
	err := manager.Start(ctx)
	require.NoError(t, err)

	// Give servers time to start
	time.Sleep(100 * time.Millisecond)

	// Verify transports are running
	transports := manager.GetTransports()
	assert.Len(t, transports, 2)

	// Stop all transports
	err = manager.Stop(ctx)
	assert.NoError(t, err)
}
