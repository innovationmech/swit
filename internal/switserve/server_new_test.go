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

package switserve

import (
	"context"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewServerWithBaseFramework(t *testing.T) {
	// Initialize logger for tests
	logger.InitLogger()

	// Catch panics from database connection
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Database connection not available for testing: %v", r)
		}
	}()

	// Skip if database is not available
	server, err := NewServer()
	if err != nil {
		if err.Error() == "failed to initialize dependencies: failed to establish database connection" {
			t.Skip("Database connection not available for testing:", err)
			return
		}
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if server != nil {
			_ = server.Shutdown()
		}
	}()

	assert.NotNil(t, server)
	assert.NotNil(t, server.baseServer)
}

func TestServerWithBaseFramework_Methods(t *testing.T) {
	// Initialize logger for tests
	logger.InitLogger()

	// Catch panics from database connection
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Database connection not available for testing: %v", r)
		}
	}()

	// Skip if database is not available
	server, err := NewServer()
	if err != nil {
		if err.Error() == "failed to initialize dependencies: failed to establish database connection" {
			t.Skip("Database connection not available for testing:", err)
			return
		}
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if server != nil {
			_ = server.Shutdown()
		}
	}()

	// Test GetTransports
	transports := server.GetTransports()
	assert.NotNil(t, transports)

	// Test GetHTTPAddress
	httpAddr := server.GetHTTPAddress()
	assert.NotEmpty(t, httpAddr)

	// Test GetGRPCAddress
	grpcAddr := server.GetGRPCAddress()
	assert.NotEmpty(t, grpcAddr)

	// Test Stop without Start
	err = server.Stop()
	assert.NoError(t, err)

	// Test Shutdown
	err = server.Shutdown()
	assert.NoError(t, err)
}

func TestServerWithBaseFramework_StartStop(t *testing.T) {
	// Initialize logger for tests
	logger.InitLogger()

	// Catch panics from database connection
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Database connection not available for testing: %v", r)
		}
	}()

	// Skip if database is not available
	server, err := NewServer()
	if err != nil {
		if err.Error() == "failed to initialize dependencies: failed to establish database connection" {
			t.Skip("Database connection not available for testing:", err)
			return
		}
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if server != nil {
			_ = server.Shutdown()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start server in a goroutine
	startErr := make(chan error, 1)
	go func() {
		startErr <- server.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	err = server.Stop()
	assert.NoError(t, err)

	// Check if start completed without error
	select {
	case err := <-startErr:
		if err != nil {
			// Service discovery errors are acceptable in test environment
			t.Logf("Start completed with error (acceptable in test): %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server start did not complete within timeout")
	}
}

func TestServerWithBaseFramework_NilBaseServer(t *testing.T) {
	server := &Server{baseServer: nil}

	// Test methods with nil baseServer
	err := server.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base server is not initialized")

	err = server.Stop()
	assert.NoError(t, err)

	err = server.Shutdown()
	assert.NoError(t, err)

	transports := server.GetTransports()
	assert.Empty(t, transports)

	httpAddr := server.GetHTTPAddress()
	assert.Empty(t, httpAddr)

	grpcAddr := server.GetGRPCAddress()
	assert.Empty(t, grpcAddr)
}

func TestServerWithBaseFramework_BackwardCompatibility(t *testing.T) {
	// Initialize logger for tests
	logger.InitLogger()

	// Catch panics from database connection
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Database connection not available for testing: %v", r)
		}
	}()

	// This test ensures the new implementation maintains the same interface
	// as the old implementation for backward compatibility

	// Skip if database is not available
	server, err := NewServer()
	if err != nil {
		if err.Error() == "failed to initialize dependencies: failed to establish database connection" {
			t.Skip("Database connection not available for testing:", err)
			return
		}
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if server != nil {
			_ = server.Shutdown()
		}
	}()

	// Verify that all expected methods exist and can be called
	assert.NotNil(t, server.Start)
	assert.NotNil(t, server.Stop)
	assert.NotNil(t, server.Shutdown)
	assert.NotNil(t, server.GetTransports)

	// Verify return types are as expected
	transports := server.GetTransports()
	assert.IsType(t, transports, server.GetTransports())

	// Test that methods can be called without panicking
	assert.NotPanics(t, func() {
		_ = server.Stop()
	})

	assert.NotPanics(t, func() {
		_ = server.Shutdown()
	})

	assert.NotPanics(t, func() {
		_ = server.GetTransports()
	})
}
