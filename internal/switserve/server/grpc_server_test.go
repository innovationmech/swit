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
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func init() {
	// Initialize logger for tests
	logger.Logger, _ = zap.NewDevelopment()
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		name     string
		portStr  string
		expected int
	}{
		{"empty string", "", 8080},
		{"valid port", "9000", 9000},
		{"invalid port string", "invalid", 8080},
		{"zero port", "0", 0},
		{"max valid port", "65535", 65535},
		{"over max port", "65536", 65536}, // parsePort doesn't validate, isValidPort does
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePort(tt.portStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidPort(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		expected bool
	}{
		{"negative port", -1, false},
		{"zero port", 0, false},
		{"valid min port", 1, true},
		{"valid standard port", 8080, true},
		{"valid max port", 65535, true},
		{"over max port", 65536, false},
		{"way over max port", 100000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidPort(tt.port)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServer_GRPCServerConcurrency(t *testing.T) {
	server := &Server{}

	// Test concurrent access to gRPC server
	var wg sync.WaitGroup
	numRoutines := 10

	// Create a mock gRPC server
	mockServer := grpc.NewServer()

	// Test concurrent set and get operations
	for i := 0; i < numRoutines; i++ {
		wg.Add(2)

		// Goroutine to set gRPC server
		go func() {
			defer wg.Done()
			server.setGRPCServer(mockServer)
		}()

		// Goroutine to get gRPC server
		go func() {
			defer wg.Done()
			_ = server.getGRPCServer()
		}()
	}

	wg.Wait()

	// Verify that the server is properly set
	retrievedServer := server.getGRPCServer()
	assert.Equal(t, mockServer, retrievedServer)

	// Clean up
	mockServer.Stop()
}

func TestServer_GracefulShutdownGRPC(t *testing.T) {
	server := &Server{}

	// Test graceful shutdown when gRPC server is not initialized
	t.Run("shutdown_uninitialized_server", func(t *testing.T) {
		// Should not panic or block
		server.GracefulShutdownGRPC(1 * time.Second)
		assert.Nil(t, server.getGRPCServer())
	})

	// Test graceful shutdown when gRPC server is initialized
	t.Run("shutdown_initialized_server", func(t *testing.T) {
		mockServer := grpc.NewServer()
		server.setGRPCServer(mockServer)

		// Should shutdown gracefully
		server.GracefulShutdownGRPC(1 * time.Second)

		// Note: After shutdown, the server reference remains but the server is stopped
		assert.NotNil(t, server.getGRPCServer())
	})
}

func TestCreateConfiguredGRPCServer(t *testing.T) {
	server := &Server{}

	grpcServer := server.createConfiguredGRPCServer()
	assert.NotNil(t, grpcServer)

	// Clean up
	grpcServer.Stop()
}

// Test port calculation logic independently
func TestPortCalculationLogic(t *testing.T) {
	tests := []struct {
		name             string
		httpPort         int
		expectedValid    bool
		expectedGRPCPort int
		description      string
	}{
		{
			name:             "normal HTTP port",
			httpPort:         8080,
			expectedValid:    true,
			expectedGRPCPort: 9080,
			description:      "Normal HTTP port should result in valid gRPC port",
		},
		{
			name:             "high HTTP port causing overflow",
			httpPort:         65000,
			expectedValid:    false,
			expectedGRPCPort: 66000,
			description:      "High HTTP port should cause gRPC port to exceed valid range",
		},
		{
			name:             "edge case near limit",
			httpPort:         64535,
			expectedValid:    true,
			expectedGRPCPort: 65535,
			description:      "HTTP port at edge should result in max valid gRPC port",
		},
		{
			name:             "edge case over limit",
			httpPort:         64536,
			expectedValid:    false,
			expectedGRPCPort: 65536,
			description:      "HTTP port just over edge should exceed valid gRPC port range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculatedPort := tt.httpPort + 1000
			isValid := isValidPort(calculatedPort)

			assert.Equal(t, tt.expectedGRPCPort, calculatedPort, "Calculated port should match expected")
			assert.Equal(t, tt.expectedValid, isValid, tt.description)
		})
	}
}
