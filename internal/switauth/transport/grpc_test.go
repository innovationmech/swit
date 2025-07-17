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

func TestNewGRPCTransport(t *testing.T) {
	grpcTransport := transport.NewGRPCTransport()
	assert.NotNil(t, grpcTransport)
	assert.Equal(t, "grpc", grpcTransport.GetName())
}

func TestGRPCTransport_GetName(t *testing.T) {
	grpcTransport := transport.NewGRPCTransport()
	assert.Equal(t, "grpc", grpcTransport.GetName())
}

func TestGRPCTransport_SetAndGetAddress(t *testing.T) {
	grpcTransport := transport.NewGRPCTransport()

	// Test default address
	assert.Empty(t, grpcTransport.GetAddress())

	// Test setting address
	addr := ":9090"
	grpcTransport.SetAddress(addr)
	assert.Equal(t, addr, grpcTransport.GetAddress())
}

func TestGRPCTransport_GetPort(t *testing.T) {
	grpcTransport := transport.NewGRPCTransport()

	// Test empty address
	assert.Equal(t, 0, grpcTransport.GetPort())

	// Test valid address
	grpcTransport.SetAddress(":9090")
	assert.Equal(t, 9090, grpcTransport.GetPort())

	// Test invalid address
	grpcTransport.SetAddress("invalid")
	assert.Equal(t, 0, grpcTransport.GetPort())
}

func TestGRPCTransport_StartAndStop(t *testing.T) {
	grpcTransport := transport.NewGRPCTransport()
	grpcTransport.SetAddress(":0") // Use random port

	ctx := context.Background()

	// Test start
	err := grpcTransport.Start(ctx)
	require.NoError(t, err)

	// Verify server is running
	assert.NotNil(t, grpcTransport.GetServer())
	assert.NotEmpty(t, grpcTransport.GetAddress())
	assert.Greater(t, grpcTransport.GetPort(), 0)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test stop
	err = grpcTransport.Stop(ctx)
	assert.NoError(t, err)
}

func TestGRPCTransport_StartWithDefaultPort(t *testing.T) {
	grpcTransport := transport.NewGRPCTransport()
	// Don't set address, should use default :50051

	ctx := context.Background()

	// Test start with default port
	err := grpcTransport.Start(ctx)
	require.NoError(t, err)

	// Verify default port is used
	assert.Contains(t, grpcTransport.GetAddress(), ":50051")

	// Clean up
	err = grpcTransport.Stop(ctx)
	assert.NoError(t, err)
}

func TestGRPCTransport_StopWithoutStart(t *testing.T) {
	grpcTransport := transport.NewGRPCTransport()
	ctx := context.Background()

	// Test stop without start should not error
	err := grpcTransport.Stop(ctx)
	assert.NoError(t, err)
}

func TestGRPCTransport_GetServerBeforeStart(t *testing.T) {
	grpcTransport := transport.NewGRPCTransport()

	// Server should be nil before start
	assert.Nil(t, grpcTransport.GetServer())
}

func TestGRPCTransport_StartError(t *testing.T) {
	grpcTransport := transport.NewGRPCTransport()
	grpcTransport.SetAddress("invalid-address")

	ctx := context.Background()

	// Test start with invalid address
	err := grpcTransport.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create gRPC listener")
}

func TestGRPCTransport_ConcurrentAccess(t *testing.T) {
	grpcTransport := transport.NewGRPCTransport()

	// Test concurrent access to address
	go func() {
		for i := 0; i < 100; i++ {
			grpcTransport.SetAddress(":9090")
		}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = grpcTransport.GetAddress()
		}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = grpcTransport.GetPort()
		}
	}()

	// Give goroutines time to complete
	time.Sleep(100 * time.Millisecond)

	// Should not panic
	assert.Equal(t, ":9090", grpcTransport.GetAddress())
}
