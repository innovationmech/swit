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
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockServiceRegistrar is a mock implementation of ServiceRegistrar interface
type MockServiceRegistrar struct {
	mock.Mock
}

func (m *MockServiceRegistrar) RegisterGRPC(server *grpc.Server) error {
	args := m.Called(server)
	return args.Error(0)
}

func (m *MockServiceRegistrar) RegisterHTTP(router *gin.Engine) error {
	args := m.Called(router)
	return args.Error(0)
}

func (m *MockServiceRegistrar) GetName() string {
	args := m.Called()
	return args.String(0)
}

func TestNewServiceRegistry(t *testing.T) {
	registry := NewServiceRegistry()

	assert.NotNil(t, registry)
	assert.Empty(t, registry.registrars)
}

func TestServiceRegistry_Register(t *testing.T) {
	registry := NewServiceRegistry()
	mockRegistrar := &MockServiceRegistrar{}
	mockRegistrar.On("GetName").Return("test-service")

	// Test registering a service
	registry.Register(mockRegistrar)

	registrars := registry.GetRegistrars()
	assert.Len(t, registrars, 1)
	assert.Equal(t, mockRegistrar, registrars[0])
}

func TestServiceRegistry_RegisterMultiple(t *testing.T) {
	registry := NewServiceRegistry()

	registrar1 := &MockServiceRegistrar{}
	registrar1.On("GetName").Return("service1")

	registrar2 := &MockServiceRegistrar{}
	registrar2.On("GetName").Return("service2")

	registry.Register(registrar1)
	registry.Register(registrar2)

	registrars := registry.GetRegistrars()
	assert.Len(t, registrars, 2)
	assert.Contains(t, registrars, registrar1)
	assert.Contains(t, registrars, registrar2)
}

func TestServiceRegistry_RegisterAllGRPC_Success(t *testing.T) {
	registry := NewServiceRegistry()
	server := grpc.NewServer()

	registrar1 := &MockServiceRegistrar{}
	registrar1.On("RegisterGRPC", server).Return(nil)

	registrar2 := &MockServiceRegistrar{}
	registrar2.On("RegisterGRPC", server).Return(nil)

	registry.Register(registrar1)
	registry.Register(registrar2)

	err := registry.RegisterAllGRPC(server)

	assert.NoError(t, err)
	registrar1.AssertExpectations(t)
	registrar2.AssertExpectations(t)
}

func TestServiceRegistry_RegisterAllGRPC_WithError(t *testing.T) {
	registry := NewServiceRegistry()
	server := grpc.NewServer()
	expectedError := errors.New("grpc registration error")

	registrar1 := &MockServiceRegistrar{}
	registrar1.On("RegisterGRPC", server).Return(expectedError)

	registrar2 := &MockServiceRegistrar{}
	// This shouldn't be called due to early return

	registry.Register(registrar1)
	registry.Register(registrar2)

	err := registry.RegisterAllGRPC(server)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	registrar1.AssertExpectations(t)
	registrar2.AssertExpectations(t)
}

func TestServiceRegistry_RegisterAllGRPC_EmptyRegistry(t *testing.T) {
	registry := NewServiceRegistry()
	server := grpc.NewServer()

	err := registry.RegisterAllGRPC(server)

	assert.NoError(t, err)
}

func TestServiceRegistry_RegisterAllHTTP_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := NewServiceRegistry()
	router := gin.New()

	registrar1 := &MockServiceRegistrar{}
	registrar1.On("RegisterHTTP", router).Return(nil)

	registrar2 := &MockServiceRegistrar{}
	registrar2.On("RegisterHTTP", router).Return(nil)

	registry.Register(registrar1)
	registry.Register(registrar2)

	err := registry.RegisterAllHTTP(router)

	assert.NoError(t, err)
	registrar1.AssertExpectations(t)
	registrar2.AssertExpectations(t)
}

func TestServiceRegistry_RegisterAllHTTP_WithError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := NewServiceRegistry()
	router := gin.New()
	expectedError := errors.New("http registration error")

	registrar1 := &MockServiceRegistrar{}
	registrar1.On("RegisterHTTP", router).Return(expectedError)

	registrar2 := &MockServiceRegistrar{}
	// This shouldn't be called due to early return

	registry.Register(registrar1)
	registry.Register(registrar2)

	err := registry.RegisterAllHTTP(router)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	registrar1.AssertExpectations(t)
	registrar2.AssertExpectations(t)
}

func TestServiceRegistry_RegisterAllHTTP_EmptyRegistry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := NewServiceRegistry()
	router := gin.New()

	err := registry.RegisterAllHTTP(router)

	assert.NoError(t, err)
}

func TestServiceRegistry_GetRegistrars(t *testing.T) {
	registry := NewServiceRegistry()

	registrar1 := &MockServiceRegistrar{}
	registrar1.On("GetName").Return("service1")

	registrar2 := &MockServiceRegistrar{}
	registrar2.On("GetName").Return("service2")

	registry.Register(registrar1)
	registry.Register(registrar2)

	registrars := registry.GetRegistrars()

	assert.Len(t, registrars, 2)
	assert.Contains(t, registrars, registrar1)
	assert.Contains(t, registrars, registrar2)

	// Verify that the returned slice is a copy (for thread safety)
	// This prevents external modification of internal state
	registrars[0] = nil
	originalRegistrars := registry.GetRegistrars()
	assert.NotNil(t, originalRegistrars[0]) // Should still be the original registrar, not nil
}

func TestServiceRegistry_MixedErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := NewServiceRegistry()
	server := grpc.NewServer()
	router := gin.New()

	grpcError := errors.New("grpc error")
	httpError := errors.New("http error")

	registrar1 := &MockServiceRegistrar{}
	registrar1.On("RegisterGRPC", server).Return(grpcError)
	registrar1.On("RegisterHTTP", router).Return(nil)

	registrar2 := &MockServiceRegistrar{}
	// This registrar won't be called for gRPC due to early return
	registrar2.On("RegisterHTTP", router).Return(httpError)

	registry.Register(registrar1)
	registry.Register(registrar2)

	// Test gRPC registration with error
	err := registry.RegisterAllGRPC(server)
	assert.Error(t, err)
	assert.Equal(t, grpcError, err)

	// Test HTTP registration with error
	err = registry.RegisterAllHTTP(router)
	assert.Error(t, err)
	assert.Equal(t, httpError, err)

	registrar1.AssertExpectations(t)
	registrar2.AssertExpectations(t)
}

func TestServiceRegistry_OrderPreservation(t *testing.T) {
	registry := NewServiceRegistry()

	registrar1 := &MockServiceRegistrar{}
	registrar1.On("GetName").Return("service1")

	registrar2 := &MockServiceRegistrar{}
	registrar2.On("GetName").Return("service2")

	registrar3 := &MockServiceRegistrar{}
	registrar3.On("GetName").Return("service3")

	// Register in specific order
	registry.Register(registrar1)
	registry.Register(registrar2)
	registry.Register(registrar3)

	registrars := registry.GetRegistrars()

	// Verify order is preserved
	assert.Equal(t, registrar1, registrars[0])
	assert.Equal(t, registrar2, registrars[1])
	assert.Equal(t, registrar3, registrars[2])
}

func TestServiceRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewServiceRegistry()

	// Note: ServiceRegistry doesn't have explicit synchronization
	// This test verifies that basic operations don't cause data races
	// In a real implementation, you might want to add proper synchronization

	done := make(chan bool)

	// Concurrent registrations
	for i := 0; i < 10; i++ {
		go func(index int) {
			registrar := &MockServiceRegistrar{}
			registrar.On("GetName").Return("service")
			registry.Register(registrar)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			_ = registry.GetRegistrars()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 15; i++ {
		<-done
	}

	// Verify final state
	registrars := registry.GetRegistrars()
	assert.Len(t, registrars, 10)
}

func TestServiceRegistry_IntegrationTest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := NewServiceRegistry()
	server := grpc.NewServer()
	router := gin.New()

	// Create a service registrar that implements both gRPC and HTTP
	registrar := &MockServiceRegistrar{}
	registrar.On("RegisterGRPC", server).Return(nil)
	registrar.On("RegisterHTTP", router).Return(nil)

	// Register the service
	registry.Register(registrar)

	// Test full registration flow
	err := registry.RegisterAllGRPC(server)
	assert.NoError(t, err)

	err = registry.RegisterAllHTTP(router)
	assert.NoError(t, err)

	// Verify service is registered
	registrars := registry.GetRegistrars()
	assert.Len(t, registrars, 1)
	assert.Equal(t, registrar, registrars[0])

	registrar.AssertExpectations(t)
}

func TestServiceRegistry_NilRegistrar(t *testing.T) {
	registry := NewServiceRegistry()

	// Registering nil should not cause panic
	registry.Register(nil)

	registrars := registry.GetRegistrars()
	assert.Len(t, registrars, 1)
	assert.Nil(t, registrars[0])
}

func TestServiceRegistry_PartialFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := NewServiceRegistry()
	server := grpc.NewServer()

	registrar1 := &MockServiceRegistrar{}
	registrar1.On("RegisterGRPC", server).Return(nil)
	// RegisterHTTP should not be called since we only test gRPC registration

	registrar2 := &MockServiceRegistrar{}
	registrar2.On("RegisterGRPC", server).Return(errors.New("failure"))
	// RegisterHTTP should not be called due to early return

	registrar3 := &MockServiceRegistrar{}
	// Neither method should be called due to early return

	registry.Register(registrar1)
	registry.Register(registrar2)
	registry.Register(registrar3)

	// Test that first success, second fails, third never called
	err := registry.RegisterAllGRPC(server)
	assert.Error(t, err)
	assert.Equal(t, "failure", err.Error())

	registrar1.AssertExpectations(t)
	registrar2.AssertExpectations(t)
	registrar3.AssertExpectations(t)
}
