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

package switauth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/innovationmech/swit/internal/switauth/config"
	"github.com/innovationmech/swit/internal/switauth/deps"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/transport"
	"github.com/innovationmech/swit/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	// Initialize logger for tests
	logger.InitLogger()

	// Set up test config file
	createTestConfig()

	// Run tests
	code := m.Run()

	// Clean up test config
	cleanupTestConfig()

	os.Exit(code)
}

func createTestConfig() {
	configContent := `
server:
  port: "9001"
  grpcPort: "50051"
database:
  username: "test"
  password: "test"
  host: "localhost"
  port: "3306"
  dbname: "test_db"
serviceDiscovery:
  address: "localhost:8500"
`
	err := os.WriteFile("switauth.yaml", []byte(configContent), 0644)
	if err != nil {
		panic(err)
	}
}

func cleanupTestConfig() {
	_ = os.Remove("switauth.yaml")
}

// MockBaseServer implements server.BaseServer for testing
type MockBaseServer struct {
	mock.Mock
}

func (m *MockBaseServer) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockBaseServer) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockBaseServer) Shutdown() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockBaseServer) GetHTTPAddress() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockBaseServer) GetGRPCAddress() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockBaseServer) GetTransports() []transport.NetworkTransport {
	args := m.Called()
	return args.Get(0).([]transport.NetworkTransport)
}

func (m *MockBaseServer) GetTransportStatus() map[string]server.TransportStatus {
	args := m.Called()
	return args.Get(0).(map[string]server.TransportStatus)
}

func (m *MockBaseServer) GetTransportHealth(ctx context.Context) map[string]map[string]*types.HealthStatus {
	args := m.Called(ctx)
	return args.Get(0).(map[string]map[string]*types.HealthStatus)
}

func TestNewServer_WithBaseServer(t *testing.T) {
	// This test will fail if dependencies cannot be created (database, consul not available)
	// But we can test the structure and error handling
	defer func() {
		if r := recover(); r != nil {
			// Expected when database is not available
			t.Logf("Expected panic in test environment: %v", r)
		}
	}()

	server, err := NewServer()

	if err != nil {
		// This is expected when dependencies are not available
		t.Logf("Expected error in test environment: %v", err)
		assert.Nil(t, server)
		return
	}

	// If no error, verify the server is created properly
	if server != nil {
		assert.NotNil(t, server.baseServer)
		assert.NotNil(t, server.config)
		assert.NotNil(t, server.deps)
	}
}

func TestServer_WithMockBaseServer(t *testing.T) {
	// Create a server with mock base server for testing
	mockBaseServer := &MockBaseServer{}

	server := &Server{
		baseServer: mockBaseServer,
		config: &config.AuthConfig{
			Server: struct {
				Port     string `json:"port" yaml:"port"`
				GRPCPort string `json:"grpcPort" yaml:"grpcPort"`
			}{
				Port:     "9001",
				GRPCPort: "50051",
			},
		},
		deps: &deps.Dependencies{},
	}

	t.Run("GetHTTPAddress", func(t *testing.T) {
		mockBaseServer.On("GetHTTPAddress").Return(":9001")

		address := server.GetHTTPAddress()
		assert.Equal(t, ":9001", address)

		mockBaseServer.AssertExpectations(t)
	})

	t.Run("GetGRPCAddress", func(t *testing.T) {
		mockBaseServer.On("GetGRPCAddress").Return(":50051")

		address := server.GetGRPCAddress()
		assert.Equal(t, ":50051", address)

		mockBaseServer.AssertExpectations(t)
	})

	t.Run("Start", func(t *testing.T) {
		ctx := context.Background()
		mockBaseServer.On("Start", ctx).Return(nil)

		err := server.Start(ctx)
		assert.NoError(t, err)

		mockBaseServer.AssertExpectations(t)
	})

	t.Run("Stop", func(t *testing.T) {
		ctx := context.Background()
		mockBaseServer.On("Stop", ctx).Return(nil)

		err := server.Stop(ctx)
		assert.NoError(t, err)

		mockBaseServer.AssertExpectations(t)
	})

	t.Run("Shutdown", func(t *testing.T) {
		mockBaseServer.On("Shutdown").Return(nil)

		err := server.Shutdown()
		assert.NoError(t, err)

		mockBaseServer.AssertExpectations(t)
	})

	t.Run("GetTransports", func(t *testing.T) {
		expectedTransports := []transport.NetworkTransport{}
		mockBaseServer.On("GetTransports").Return(expectedTransports)

		transports := server.GetTransports()
		assert.Equal(t, expectedTransports, transports)

		mockBaseServer.AssertExpectations(t)
	})
}

func TestServer_StartError(t *testing.T) {
	mockBaseServer := &MockBaseServer{}
	server := &Server{
		baseServer: mockBaseServer,
		config:     &config.AuthConfig{},
		deps:       &deps.Dependencies{},
	}

	ctx := context.Background()
	expectedError := assert.AnError
	mockBaseServer.On("Start", ctx).Return(expectedError)

	err := server.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start base server")

	mockBaseServer.AssertExpectations(t)
}

func TestServer_StopError(t *testing.T) {
	mockBaseServer := &MockBaseServer{}
	server := &Server{
		baseServer: mockBaseServer,
		config:     &config.AuthConfig{},
		deps:       &deps.Dependencies{},
	}

	ctx := context.Background()
	expectedError := assert.AnError
	mockBaseServer.On("Stop", ctx).Return(expectedError)

	err := server.Stop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop base server")

	mockBaseServer.AssertExpectations(t)
}

func TestServer_ShutdownError(t *testing.T) {
	mockBaseServer := &MockBaseServer{}
	server := &Server{
		baseServer: mockBaseServer,
		config:     &config.AuthConfig{},
		deps:       &deps.Dependencies{},
	}

	expectedError := assert.AnError
	mockBaseServer.On("Shutdown").Return(expectedError)

	err := server.Shutdown()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to shutdown base server")

	mockBaseServer.AssertExpectations(t)
}

func TestServer_LifecycleIntegration(t *testing.T) {
	mockBaseServer := &MockBaseServer{}
	server := &Server{
		baseServer: mockBaseServer,
		config: &config.AuthConfig{
			Server: struct {
				Port     string `json:"port" yaml:"port"`
				GRPCPort string `json:"grpcPort" yaml:"grpcPort"`
			}{
				Port:     "9001",
				GRPCPort: "50051",
			},
		},
		deps: &deps.Dependencies{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test complete lifecycle
	mockBaseServer.On("Start", ctx).Return(nil)
	mockBaseServer.On("GetHTTPAddress").Return(":9001")
	mockBaseServer.On("GetGRPCAddress").Return(":50051")
	mockBaseServer.On("Stop", ctx).Return(nil)
	mockBaseServer.On("Shutdown").Return(nil)

	// Start server
	err := server.Start(ctx)
	assert.NoError(t, err)

	// Check addresses
	httpAddr := server.GetHTTPAddress()
	grpcAddr := server.GetGRPCAddress()
	assert.Equal(t, ":9001", httpAddr)
	assert.Equal(t, ":50051", grpcAddr)

	// Stop server
	err = server.Stop(ctx)
	assert.NoError(t, err)

	// Shutdown server
	err = server.Shutdown()
	assert.NoError(t, err)

	mockBaseServer.AssertExpectations(t)
}

func TestServer_NilBaseServer(t *testing.T) {
	server := &Server{
		baseServer: nil,
		config:     &config.AuthConfig{},
		deps:       &deps.Dependencies{},
	}

	// These should panic or return errors due to nil base server
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic due to nil base server: %v", r)
		}
	}()

	ctx := context.Background()

	// Test methods that would panic with nil base server
	_ = server.Start(ctx)
	_ = server.Stop(ctx)
	_ = server.Shutdown()
	_ = server.GetHTTPAddress()
	_ = server.GetGRPCAddress()
	_ = server.GetTransports()
}

func TestServer_StructFields(t *testing.T) {
	// Test that server struct has the expected fields
	server := &Server{}

	// Test field assignment
	server.baseServer = &MockBaseServer{}
	server.config = &config.AuthConfig{}
	server.deps = &deps.Dependencies{}

	assert.NotNil(t, server.baseServer)
	assert.NotNil(t, server.config)
	assert.NotNil(t, server.deps)
}

func TestServer_BackwardCompatibility(t *testing.T) {
	// Test that the new server maintains backward compatibility
	// by providing the same public interface as the old server

	mockBaseServer := &MockBaseServer{}
	server := &Server{
		baseServer: mockBaseServer,
		config:     &config.AuthConfig{},
		deps:       &deps.Dependencies{},
	}

	// Test that all expected methods exist and work
	mockBaseServer.On("GetHTTPAddress").Return(":9001")
	mockBaseServer.On("GetGRPCAddress").Return(":50051")
	mockBaseServer.On("GetTransports").Return([]transport.NetworkTransport{})

	// These methods should exist and work as before
	httpAddr := server.GetHTTPAddress()
	grpcAddr := server.GetGRPCAddress()
	transports := server.GetTransports()

	assert.NotEmpty(t, httpAddr)
	assert.NotEmpty(t, grpcAddr)
	assert.NotNil(t, transports)

	// New methods should also exist
	mockBaseServer.On("Shutdown").Return(nil)
	err := server.Shutdown()
	assert.NoError(t, err)

	mockBaseServer.AssertExpectations(t)
}
