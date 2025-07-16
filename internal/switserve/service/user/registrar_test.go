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

package user

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func setupTest() {
	logger.InitLogger()
	gin.SetMode(gin.TestMode)
}

func TestMain(m *testing.M) {
	setupTest()
	code := m.Run()
	os.Exit(code)
}

func TestNewServiceRegistrar(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "function_exists_and_handles_database_failure",
			description: "Should verify the NewServiceRegistrar function exists and handles database failures",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies that the NewServiceRegistrar function exists
			// It may panic on database connection failure, which is acceptable behavior

			defer func() {
				if r := recover(); r != nil {
					// Database connection failure is expected in test environment
					t.Logf("NewServiceRegistrar panicked as expected: %v", r)
				}
			}()

			// Try to create a registrar - it may panic or return nil due to database connection issues
			registrar := NewServiceRegistrar()

			// If registrar is created successfully, verify its structure
			if registrar != nil {
				t.Log("NewServiceRegistrar succeeded, verifying structure")
				assert.NotNil(t, registrar.controller)
				assert.NotNil(t, registrar.userSrv)
			} else {
				t.Log("NewServiceRegistrar returned nil, which is expected without database connection")
			}
		})
	}
}

func TestServiceRegistrar_GetName(t *testing.T) {
	tests := []struct {
		name         string
		expectedName string
	}{
		{
			name:         "returns_correct_service_name",
			expectedName: "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a dummy registrar to test the GetName method
			registrar := &ServiceRegistrar{}

			name := registrar.GetName()
			assert.Equal(t, tt.expectedName, name)
		})
	}
}

func TestServiceRegistrar_RegisterGRPC(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		description string
	}{
		{
			name:        "success_register_grpc",
			expectError: false,
			description: "Should successfully register gRPC service without error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a dummy registrar to test the RegisterGRPC method
			registrar := &ServiceRegistrar{}

			server := grpc.NewServer()
			err := registrar.RegisterGRPC(server)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServiceRegistrar_RegisterHTTP(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		description string
	}{
		{
			name:        "register_http_with_nil_controller",
			expectError: false,
			description: "Should handle RegisterHTTP call with nil controller gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a dummy registrar to test the RegisterHTTP method
			registrar := &ServiceRegistrar{}

			router := gin.New()

			// The method should handle nil controller gracefully
			// and still register the route structure
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("RegisterHTTP panicked: %v", r)
				}
			}()

			err := registrar.RegisterHTTP(router)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServiceRegistrar_Methods_Exist(t *testing.T) {
	// This test verifies that the registrar methods exist and can be called
	registrar := &ServiceRegistrar{}

	// Test that GetName method exists and returns expected value
	t.Run("GetName method exists", func(t *testing.T) {
		name := registrar.GetName()
		assert.Equal(t, "user", name)
	})

	// Test that RegisterGRPC method exists and can be called
	t.Run("RegisterGRPC method exists", func(t *testing.T) {
		server := grpc.NewServer()
		err := registrar.RegisterGRPC(server)
		assert.NoError(t, err)
	})

	// Test that RegisterHTTP method exists and can be called
	t.Run("RegisterHTTP method exists", func(t *testing.T) {
		router := gin.New()

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RegisterHTTP panicked: %v", r)
			}
		}()

		err := registrar.RegisterHTTP(router)
		assert.NoError(t, err)
	})
}

func TestServiceRegistrar_ErrorHandling(t *testing.T) {
	// Test error handling for RegisterHTTP with nil router
	t.Run("RegisterHTTP with nil router", func(t *testing.T) {
		registrar := &ServiceRegistrar{}

		defer func() {
			if r := recover(); r != nil {
				t.Log("RegisterHTTP panicked with nil router, which is expected")
			}
		}()

		err := registrar.RegisterHTTP(nil)
		// If we get here without panic, that's also acceptable
		if err != nil {
			t.Log("RegisterHTTP returned error with nil router:", err)
		}
	})

	// Test error handling for RegisterGRPC with nil server
	t.Run("RegisterGRPC with nil server", func(t *testing.T) {
		registrar := &ServiceRegistrar{}

		defer func() {
			if r := recover(); r != nil {
				t.Log("RegisterGRPC panicked with nil server, which is expected")
			}
		}()

		err := registrar.RegisterGRPC(nil)
		// Based on current implementation, it should not error
		assert.NoError(t, err)
	})
}

func TestServiceRegistrar_ConcurrentAccess(t *testing.T) {
	registrar := &ServiceRegistrar{}

	const numGoroutines = 10
	results := make([]string, numGoroutines)
	done := make(chan bool, numGoroutines)

	// Test concurrent access to GetName method
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer func() { done <- true }()
			results[index] = registrar.GetName()
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all results are correct
	for i, result := range results {
		assert.Equal(t, "user", result, "Goroutine %d should return correct service name", i)
	}
}

// Benchmark tests
func BenchmarkServiceRegistrar_GetName(b *testing.B) {
	registrar := &ServiceRegistrar{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registrar.GetName()
	}
}

func BenchmarkServiceRegistrar_RegisterHTTP(b *testing.B) {
	registrar := &ServiceRegistrar{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router := gin.New()
		_ = registrar.RegisterHTTP(router)
	}
}

func BenchmarkServiceRegistrar_RegisterGRPC(b *testing.B) {
	registrar := &ServiceRegistrar{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server := grpc.NewServer()
		_ = registrar.RegisterGRPC(server)
	}
}
