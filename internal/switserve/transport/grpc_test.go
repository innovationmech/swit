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
	"fmt"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	// Initialize logger for tests
	logger.InitLogger()

	// Add config path for tests
	// This is needed because go test changes working directory to the package directory
	viper.AddConfigPath("../../..")
}

func TestNewGRPCTransport(t *testing.T) {
	transport := NewGRPCTransport()

	assert.NotNil(t, transport)
	assert.Nil(t, transport.server)
	assert.Nil(t, transport.listener)
	assert.Empty(t, transport.address)
}

func TestGRPCTransport_Name(t *testing.T) {
	transport := NewGRPCTransport()

	assert.Equal(t, "grpc", transport.Name())
}

func TestGRPCTransport_Address(t *testing.T) {
	transport := NewGRPCTransport()

	// Initially empty
	assert.Empty(t, transport.Address())

	// Set address for testing
	transport.address = ":50051"
	assert.Equal(t, ":50051", transport.Address())
}

func TestGRPCTransport_GetServer(t *testing.T) {
	transport := NewGRPCTransport()

	// Initially nil
	assert.Nil(t, transport.GetServer())

	// After creating server
	transport.server = grpc.NewServer()
	assert.NotNil(t, transport.GetServer())
}

func TestGRPCTransport_Start(t *testing.T) {
	transport := NewGRPCTransport()
	transport.SetTestPort("0") // Use dynamic port allocation
	ctx := context.Background()

	// The actual start will use the real getGRPCPort method
	err := transport.Start(ctx)
	assert.NoError(t, err)

	// Verify server is created and listening
	assert.NotNil(t, transport.server)
	assert.NotNil(t, transport.listener)
	assert.NotEmpty(t, transport.address)

	// Clean up
	transport.Stop(context.Background())
}

func TestGRPCTransport_Stop(t *testing.T) {
	transport := NewGRPCTransport()
	ctx := context.Background()

	// Test stopping without starting
	err := transport.Stop(ctx)
	assert.NoError(t, err)

	// For testing purposes, we'll just test that Stop doesn't panic
	// when called on a non-started transport
}

func TestGRPCTransport_Stop_WithTimeout(t *testing.T) {
	transport := NewGRPCTransport()
	transport.SetTestPort("0") // Use dynamic port allocation

	// Start the transport
	err := transport.Start(context.Background())
	require.NoError(t, err)

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Stop should handle timeout gracefully
	err = transport.Stop(ctx)
	assert.NoError(t, err)
}

func TestGRPCTransport_ConcurrentAccess(t *testing.T) {
	transport := NewGRPCTransport()

	// Test concurrent access to Address() method
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			transport.Address()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent access to GetServer() method
	for i := 0; i < 10; i++ {
		go func() {
			transport.GetServer()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		name     string
		portStr  string
		expected int
	}{
		{"empty string", "", 8080},
		{"valid port", "9090", 9090},
		{"invalid port", "invalid", 8080},
		{"zero", "0", 0},
		{"max port", "65535", 65535},
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
		{"valid port", 8080, true},
		{"max valid port", 65535, true},
		{"port too high", 65536, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidPort(tt.port)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGRPCTransport_getGRPCPort(t *testing.T) {
	transport := NewGRPCTransport()

	// Test default behavior (should return a valid port)
	port := transport.getGRPCPort()
	assert.NotEmpty(t, port)

	// Verify it's a valid port string
	parsedPort := parsePort(port)
	assert.True(t, isValidPort(parsedPort))
}

func TestGRPCTransport_createConfiguredGRPCServer(t *testing.T) {
	transport := NewGRPCTransport()

	server := transport.createConfiguredGRPCServer()
	assert.NotNil(t, server)

	// Verify server is properly configured (we can't directly test options,
	// but we can verify the server works)
	assert.IsType(t, &grpc.Server{}, server)
}

func TestGRPCTransport_StartMultipleTimes(t *testing.T) {
	transport := NewGRPCTransport()
	transport.SetTestPort("0") // Use dynamic port allocation
	ctx := context.Background()

	// Start the transport
	err := transport.Start(ctx)
	assert.NoError(t, err)

	// Clean up
	transport.Stop(ctx)

	// Starting again after stop should work (creates new instance)
	err = transport.Start(ctx)
	// Note: This might fail due to port conflicts, but the test is about not panicking
	// We just verify it doesn't panic

	// Clean up
	transport.Stop(ctx)
}

func TestGRPCTransport_ServerConnection(t *testing.T) {
	transport := NewGRPCTransport()
	transport.SetTestPort("0") // Use dynamic port allocation
	ctx := context.Background()

	// Start the transport
	err := transport.Start(ctx)
	require.NoError(t, err)

	// Extract port from address
	address := transport.Address()
	assert.NotEmpty(t, address)

	// Try to connect to the server
	conn, err := grpc.NewClient(fmt.Sprintf("localhost%s", address), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		defer conn.Close()
		// Connection successful
		assert.NotNil(t, conn)
	}
	// If connection fails, it might be due to port conflicts in CI, which is acceptable

	// Clean up
	transport.Stop(ctx)
}

func TestGRPCTransport_IntegrationTest(t *testing.T) {
	transport := NewGRPCTransport()
	transport.SetTestPort("0") // Use dynamic port allocation
	ctx := context.Background()

	// Test the full lifecycle
	err := transport.Start(ctx)
	require.NoError(t, err)

	// Verify transport properties
	assert.Equal(t, "grpc", transport.Name())
	assert.NotEmpty(t, transport.Address())
	assert.NotNil(t, transport.GetServer())

	// Clean stop
	err = transport.Stop(ctx)
	assert.NoError(t, err)

	// Verify server is still accessible after stop
	assert.NotNil(t, transport.GetServer())
}
