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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"

	"github.com/innovationmech/swit/pkg/logger"
)

func init() {
	// Initialize logger for tests
	logger.Logger, _ = zap.NewDevelopment()
}

func TestNewGRPCTransport(t *testing.T) {
	transport := NewGRPCTransport()

	assert.NotNil(t, transport)
	assert.NotNil(t, transport.config)
	assert.Equal(t, "grpc", transport.GetName())
}

func TestNewGRPCTransportWithConfig(t *testing.T) {
	config := &GRPCTransportConfig{
		Address:             ":9999",
		EnableKeepalive:     true,
		EnableReflection:    true,
		EnableHealthService: true,
		UnaryInterceptors:   []grpc.UnaryServerInterceptor{},
	}

	transport := NewGRPCTransportWithConfig(config)

	assert.NotNil(t, transport)
	assert.Equal(t, config, transport.config)
	assert.Equal(t, "grpc", transport.GetName())
}

func TestNewGRPCTransportWithConfig_NilConfig(t *testing.T) {
	transport := NewGRPCTransportWithConfig(nil)

	assert.NotNil(t, transport)
	assert.NotNil(t, transport.config)
	assert.Equal(t, ":50051", transport.config.Address)
	assert.True(t, transport.config.EnableReflection)
	assert.True(t, transport.config.EnableHealthService)
	assert.True(t, transport.config.EnableKeepalive)
}

func TestGRPCTransport_Start_Success(t *testing.T) {
	config := &GRPCTransportConfig{
		Address:             ":0", // Use dynamic port
		EnableReflection:    true,
		EnableHealthService: true,
	}
	transport := NewGRPCTransportWithConfig(config)

	ctx := context.Background()
	err := transport.Start(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, transport.GetAddress())

	// Clean up
	err = transport.Stop(context.Background())
	assert.NoError(t, err)
}

func TestGRPCTransport_Start_AlreadyStarted(t *testing.T) {
	config := &GRPCTransportConfig{
		Address:             ":0",
		EnableReflection:    true,
		EnableHealthService: true,
	}
	transport := NewGRPCTransportWithConfig(config)

	ctx := context.Background()

	// Start first time
	err := transport.Start(ctx)
	assert.NoError(t, err)

	// Start second time - should not error
	err = transport.Start(ctx)
	assert.NoError(t, err)

	// Clean up
	err = transport.Stop(context.Background())
	assert.NoError(t, err)
}

func TestGRPCTransport_Stop_Success(t *testing.T) {
	config := &GRPCTransportConfig{
		Address:             ":0",
		EnableReflection:    true,
		EnableHealthService: true,
	}
	transport := NewGRPCTransportWithConfig(config)

	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	err = transport.Stop(ctx)
	assert.NoError(t, err)
}

func TestGRPCTransport_Stop_NotStarted(t *testing.T) {
	transport := NewGRPCTransport()

	ctx := context.Background()
	err := transport.Stop(ctx)
	assert.NoError(t, err) // Should not error when stopping non-started transport
}

func TestGRPCTransport_GetPort(t *testing.T) {
	config := &GRPCTransportConfig{
		Address:             ":0",
		EnableReflection:    true,
		EnableHealthService: true,
	}
	transport := NewGRPCTransportWithConfig(config)

	// Before start
	assert.Equal(t, 0, transport.GetPort())

	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	// After start
	port := transport.GetPort()
	assert.True(t, port > 0)

	// Clean up
	transport.Stop(context.Background())
}

func TestGRPCTransport_SetTestPort(t *testing.T) {
	transport := NewGRPCTransport()

	transport.SetTestPort("9999")

	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	assert.Contains(t, transport.GetAddress(), ":9999")

	// Clean up
	transport.Stop(context.Background())
}

func TestGRPCTransport_SetAddress(t *testing.T) {
	transport := NewGRPCTransport()

	transport.SetAddress(":7777")

	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	assert.Contains(t, transport.GetAddress(), ":7777")

	// Clean up
	transport.Stop(context.Background())
}

func TestGRPCTransport_GetServer(t *testing.T) {
	transport := NewGRPCTransport()

	// Server is nil before start
	server1 := transport.GetServer()
	assert.Nil(t, server1)

	// Start the transport to create server
	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	// Now server should exist
	server2 := transport.GetServer()
	assert.NotNil(t, server2)

	// Clean up
	transport.Stop(context.Background())
}

func TestGRPCTransport_FullLifecycle(t *testing.T) {
	config := &GRPCTransportConfig{
		Address:             ":0",
		EnableReflection:    true,
		EnableHealthService: true,
	}
	transport := NewGRPCTransportWithConfig(config)

	// Start transport
	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	// Verify server is running by connecting to it
	addr := transport.GetAddress()
	if strings.HasPrefix(addr, "[::") {
		// IPv6 address, use localhost with port only
		port := strings.Split(addr, "]:")
		if len(port) == 2 {
			addr = "localhost:" + port[1]
		}
	}
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		conn.Close()
	}

	// Stop transport
	err = transport.Stop(ctx)
	assert.NoError(t, err)
}

func TestGRPCTransport_ConcurrentAccess(t *testing.T) {
	transport := NewGRPCTransport()
	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Concurrent GetServer operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			server := transport.GetServer()
			_ = server // May be nil before start
		}()
	}

	// Concurrent GetAddress operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			address := transport.GetAddress()
			_ = address // May be empty before start
		}()
	}

	wg.Wait()
}

func TestGRPCTransport_StartStop_Multiple(t *testing.T) {
	config := &GRPCTransportConfig{
		Address:             ":0",
		EnableReflection:    true,
		EnableHealthService: true,
	}
	transport := NewGRPCTransportWithConfig(config)

	ctx := context.Background()

	// Test multiple start/stop cycles
	for i := 0; i < 3; i++ {
		err := transport.Start(ctx)
		require.NoError(t, err)

		err = transport.Stop(ctx)
		require.NoError(t, err)
	}
}

func TestGRPCTransport_WithKeepalive(t *testing.T) {
	config := &GRPCTransportConfig{
		Address:         ":0",
		EnableKeepalive: true,
	}
	transport := NewGRPCTransportWithConfig(config)

	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	// Verify server is running
	assert.NotEmpty(t, transport.GetAddress())

	// Clean up
	err = transport.Stop(ctx)
	assert.NoError(t, err)
}

func TestGRPCTransport_WithInterceptors(t *testing.T) {
	// Create a test interceptor
	testInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}

	config := &GRPCTransportConfig{
		Address:           ":0",
		UnaryInterceptors: []grpc.UnaryServerInterceptor{testInterceptor},
	}
	transport := NewGRPCTransportWithConfig(config)

	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	// Verify the interceptor was configured (we can't easily test if it's called without actual gRPC calls)
	assert.NotNil(t, transport.GetServer())

	// Clean up
	err = transport.Stop(ctx)
	assert.NoError(t, err)
}

func TestGRPCTransport_HealthService(t *testing.T) {
	config := &GRPCTransportConfig{
		Address:             ":0",
		EnableHealthService: true,
	}
	transport := NewGRPCTransportWithConfig(config)

	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	// Connect to the server and test health service
	addr := transport.GetAddress()
	if strings.HasPrefix(addr, "[::") {
		// IPv6 address, use localhost with port only
		port := strings.Split(addr, "]:")
		if len(port) == 2 {
			addr = "localhost:" + port[1]
		}
	}
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)

	// Test health check with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)

	// Clean up
	err = transport.Stop(context.Background())
	assert.NoError(t, err)
}

func TestGRPCTransport_ReflectionService(t *testing.T) {
	config := &GRPCTransportConfig{
		Address:          ":0",
		EnableReflection: true,
	}
	transport := NewGRPCTransportWithConfig(config)

	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	// Connect to the server and test reflection
	addr := transport.GetAddress()
	if strings.HasPrefix(addr, "[::") {
		// IPv6 address, use localhost with port only
		port := strings.Split(addr, "]:")
		if len(port) == 2 {
			addr = "localhost:" + port[1]
		}
	}
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	// Test that reflection is enabled by creating a reflection client
	refClient := grpc_reflection_v1.NewServerReflectionClient(conn)

	// Test reflection with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := refClient.ServerReflectionInfo(ctx)
	require.NoError(t, err)

	// Send a request to list services
	err = stream.Send(&grpc_reflection_v1.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1.ServerReflectionRequest_ListServices{},
	})
	require.NoError(t, err)

	// Receive response
	resp, err := stream.Recv()
	require.NoError(t, err)

	// Check if we got a list services response
	if listResp := resp.GetListServicesResponse(); listResp != nil {
		// Should have at least one service (health or reflection)
		assert.True(t, len(listResp.Service) > 0, "Should have at least one service")
	}

	// Clean up
	stream.CloseSend()
	err = transport.Stop(context.Background())
	assert.NoError(t, err)
}

func TestGRPCTransport_determineAddress(t *testing.T) {
	tests := []struct {
		name         string
		config       *GRPCTransportConfig
		expectedAddr string
	}{
		{
			name: "configured address",
			config: &GRPCTransportConfig{
				Address: ":7777",
			},
			expectedAddr: ":7777",
		},
		{
			name:         "nil config",
			config:       nil,
			expectedAddr: ":50051",
		},
		{
			name: "empty address",
			config: &GRPCTransportConfig{
				Address: "",
			},
			expectedAddr: ":50051",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewGRPCTransportWithConfig(tt.config)
			addr := transport.determineAddress()
			assert.Equal(t, tt.expectedAddr, addr)
		})
	}
}

// Benchmark tests
func BenchmarkGRPCTransport_GetServer(b *testing.B) {
	transport := NewGRPCTransport()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = transport.GetServer()
	}
}

func BenchmarkGRPCTransport_GetAddress(b *testing.B) {
	transport := NewGRPCTransport()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = transport.GetAddress()
	}
}

func BenchmarkGRPCTransport_determineAddress(b *testing.B) {
	transport := NewGRPCTransport()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = transport.determineAddress()
	}
}
