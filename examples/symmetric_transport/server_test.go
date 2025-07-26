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

package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
)

func init() {
	// Initialize logger for tests
	logger.Logger, _ = zap.NewDevelopment()
}

func TestNewSymmetricServer(t *testing.T) {
	server := NewSymmetricServer()

	assert.NotNil(t, server)
	assert.NotNil(t, server.transportManager)
	assert.NotNil(t, server.httpTransport)
	assert.NotNil(t, server.grpcTransport)
}

func TestSymmetricServer_ServiceRegistration(t *testing.T) {
	server := NewSymmetricServer()

	// Create test services
	httpService := NewExampleService("http-test", "1.0.0")
	grpcService := NewExampleService("grpc-test", "1.0.0")
	dualService := NewExampleService("dual-test", "1.0.0")

	// Test registration
	err := server.RegisterHTTPOnlyService(httpService)
	assert.NoError(t, err)

	err = server.RegisterGRPCOnlyService(grpcService)
	assert.NoError(t, err)

	err = server.RegisterDualProtocolService(dualService)
	assert.NoError(t, err)
}

func TestSymmetricServer_ArchitecturalSymmetry(t *testing.T) {
	// This test demonstrates the architectural symmetry achieved
	server := NewSymmetricServer()

	// Both transports can be used independently for service registration
	httpService := NewExampleService("independent-http", "1.0.0")
	grpcService := NewExampleService("independent-grpc", "1.0.0")

	// HTTP transport independence
	err := server.RegisterHTTPOnlyService(httpService)
	assert.NoError(t, err)

	// gRPC transport independence
	err = server.RegisterGRPCOnlyService(grpcService)
	assert.NoError(t, err)

	// Verify both transports work independently
	registryManager := server.transportManager.GetServiceRegistryManager()
	httpRegistry := registryManager.GetRegistry("http")
	grpcRegistry := registryManager.GetRegistry("grpc")

	// Each transport has its own independent registry
	assert.NotNil(t, httpRegistry)
	assert.NotNil(t, grpcRegistry)
	assert.True(t, httpRegistry != grpcRegistry) // Different instances

	// Services are properly isolated in their respective registries
	assert.Equal(t, 1, httpRegistry.Count())
	assert.Equal(t, 1, grpcRegistry.Count())
}

func TestExampleService(t *testing.T) {
	service := NewExampleService("test", "1.0.0")

	assert.NotNil(t, service)
	assert.Equal(t, "test", service.name)
	assert.Equal(t, "1.0.0", service.version)

	// Test metadata
	metadata := service.GetMetadata()
	assert.Equal(t, "test", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Equal(t, "/test/health", metadata.HealthEndpoint)
	assert.Contains(t, metadata.Tags, "example")
	assert.Contains(t, metadata.Tags, "symmetric")

	// Test health check
	ctx := context.Background()
	health, err := service.IsHealthy(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.Equal(t, "1.0.0", health.Version)

	// Test lifecycle
	err = service.Initialize(ctx)
	assert.NoError(t, err)

	err = service.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestSymmetricServer_Lifecycle(t *testing.T) {
	server := NewSymmetricServer()

	// Use dynamic ports to avoid conflicts
	server.httpTransport.SetTestPort("0")
	server.grpcTransport.SetTestPort("0")

	// Register a test service
	service := NewExampleService("test-service", "1.0.0")
	err := server.RegisterDualProtocolService(service)
	require.NoError(t, err)

	ctx := context.Background()

	// Start server
	err = server.Start(ctx)
	require.NoError(t, err)

	// Verify addresses are available
	httpAddr := server.httpTransport.GetAddress()
	grpcAddr := server.grpcTransport.GetAddress()
	assert.NotEmpty(t, httpAddr)
	assert.NotEmpty(t, grpcAddr)
	assert.NotEqual(t, httpAddr, grpcAddr)

	// Stop server
	err = server.Stop(ctx)
	assert.NoError(t, err)
}
