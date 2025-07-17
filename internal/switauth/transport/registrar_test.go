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
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/internal/switauth/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServiceRegistrar is a mock service registrar for testing
type TestServiceRegistrar struct {
	name        string
	grpcCalled  bool
	httpCalled  bool
	returnError error
}

func (t *TestServiceRegistrar) RegisterGRPC(server *grpc.Server) error {
	t.grpcCalled = true
	return t.returnError
}

func (t *TestServiceRegistrar) RegisterHTTP(router *gin.Engine) error {
	t.httpCalled = true
	return t.returnError
}

func (t *TestServiceRegistrar) GetName() string {
	return t.name
}

func TestNewServiceRegistry(t *testing.T) {
	registry := transport.NewServiceRegistry()
	assert.NotNil(t, registry)
	assert.Empty(t, registry.GetServices())
}

func TestServiceRegistry_Register(t *testing.T) {
	registry := transport.NewServiceRegistry()
	service := &TestServiceRegistrar{name: "test-service"}

	registry.Register(service)

	services := registry.GetServices()
	assert.Len(t, services, 1)
	assert.Equal(t, "test-service", services[0])
}

func TestServiceRegistry_RegisterAllGRPC(t *testing.T) {
	registry := transport.NewServiceRegistry()
	service1 := &TestServiceRegistrar{name: "service1"}
	service2 := &TestServiceRegistrar{name: "service2"}

	registry.Register(service1)
	registry.Register(service2)

	server := grpc.NewServer()
	err := registry.RegisterAllGRPC(server)
	require.NoError(t, err)

	assert.True(t, service1.grpcCalled)
	assert.True(t, service2.grpcCalled)
}

func TestServiceRegistry_RegisterAllHTTP(t *testing.T) {
	registry := transport.NewServiceRegistry()
	service1 := &TestServiceRegistrar{name: "service1"}
	service2 := &TestServiceRegistrar{name: "service2"}

	registry.Register(service1)
	registry.Register(service2)

	router := gin.New()
	err := registry.RegisterAllHTTP(router)
	require.NoError(t, err)

	assert.True(t, service1.httpCalled)
	assert.True(t, service2.httpCalled)
}

func TestServiceRegistry_RegisterAllGRPC_Error(t *testing.T) {
	registry := transport.NewServiceRegistry()
	service := &TestServiceRegistrar{
		name:        "error-service",
		returnError: assert.AnError,
	}

	registry.Register(service)

	server := grpc.NewServer()
	err := registry.RegisterAllGRPC(server)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestServiceRegistry_RegisterAllHTTP_Error(t *testing.T) {
	registry := transport.NewServiceRegistry()
	service := &TestServiceRegistrar{
		name:        "error-service",
		returnError: assert.AnError,
	}

	registry.Register(service)

	router := gin.New()
	err := registry.RegisterAllHTTP(router)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestServiceRegistry_MultipleServices(t *testing.T) {
	registry := transport.NewServiceRegistry()
	services := []*TestServiceRegistrar{
		{name: "service1"},
		{name: "service2"},
		{name: "service3"},
	}

	for _, service := range services {
		registry.Register(service)
	}

	registered := registry.GetServices()
	assert.Len(t, registered, 3)
	assert.Contains(t, registered, "service1")
	assert.Contains(t, registered, "service2")
	assert.Contains(t, registered, "service3")
}

func TestServiceRegistry_Empty(t *testing.T) {
	registry := transport.NewServiceRegistry()

	// Test empty registration
	server := grpc.NewServer()
	err := registry.RegisterAllGRPC(server)
	assert.NoError(t, err)

	router := gin.New()
	err = registry.RegisterAllHTTP(router)
	assert.NoError(t, err)

	services := registry.GetServices()
	assert.Empty(t, services)
}
